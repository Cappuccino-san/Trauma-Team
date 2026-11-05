package e2e

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"traumateam/weather_ingest/pkg/models"
	"traumateam/weather_ingest/pkg/regulations"
)

// DispatchVerdict represents the definitive mission feasibility status.
type DispatchVerdict string

const (
	VerdictApproved      DispatchVerdict = "APPROVED"
	VerdictMarginal      DispatchVerdict = "MARGINAL"
	VerdictBelowMinimums DispatchVerdict = "BELOW_MINIMUMS"
	VerdictDisqualified  DispatchVerdict = "DISQUALIFIED"
	VerdictIFRRequired   DispatchVerdict = "IFR_REQUIRED"
)

// LimitingFactor represents the operational disqualifier.
type LimitingFactor string

const (
	LimitNone                    LimitingFactor = "NONE"
	LimitCeilingBelowMinimums    LimitingFactor = "FAR_135_609_CEILING_BELOW_MINIMUMS"
	LimitVisibilityBelowMinimums LimitingFactor = "FAR_135_609_VISIBILITY_BELOW_MINIMUMS"
	LimitConvectiveSIGMET        LimitingFactor = "ACTIVE_CONVECTIVE_SIGMET"
	LimitSevereIcing             LimitingFactor = "SEVERE_ICING_HAZARD"
	LimitSevereTurbulence        LimitingFactor = "SEVERE_TURBULENCE_HAZARD"
	LimitIFRAlternateMissing     LimitingFactor = "IFR_ALTERNATE_MISSING"
	LimitUnflyableWind           LimitingFactor = "UNFLYABLE_WIND"
)

// ParsedTAFPeriod represents a decoded forecast change segment.
type ParsedTAFPeriod struct {
	ChangeType       string
	StartTimeUTC     string
	EndTimeUTC       string
	WindDirDeg       float64
	WindSpeedKts     float64
	WindGustKts      float64
	VisibilitySM     float64
	CeilingAGLFt     float64
	WeatherPhenomena []string
}

// ParsedTAF represents a complete aerodrome forecast.
type ParsedTAF struct {
	StationID string
	Periods   []ParsedTAFPeriod
	RawText   string
}

// DecodedWindVector represents true wind and temperature at an altitude layer.
type DecodedWindVector struct {
	AltitudeMSLFt float64
	DirectionDeg  float64
	SpeedKts      float64
	TemperatureC  float64
	IsCalm        bool
}

// ParsedSIGMET represents a parsed severe weather advisory.
type ParsedSIGMET struct {
	AdvisoryID    string
	HazardType    string // "CONVECTIVE", "SEVERE_ICING", "SEVERE_TURB"
	Severity      string // "SEVERE", "EXTREME"
	ValidUntilUTC time.Time
	MinAltFt      float64
	MaxAltFt      float64
	Summary       string
	Active        bool
	PolygonCoords [][2]float64 // [Lat, Lon]
}

// AuthoritativeMETARParser implements full specification parsing including all edge cases.
type AuthoritativeMETARParser struct{}

func (p *AuthoritativeMETARParser) Parse(raw string) (*models.METAR, error) {
	raw = strings.TrimSpace(raw)
	tokens := strings.Fields(raw)
	if len(tokens) < 3 {
		return nil, fmt.Errorf("insufficient METAR tokens: %q", raw)
	}

	obs := &models.METAR{
		RawText:            raw,
		ObservationTimeUTC: time.Now().UTC(),
		CeilingAGLFt:       25000.0,
		VisibilitySM:       10.0,
		AltimeterInHg:      29.92,
	}

	idx := 0
	if tokens[0] == "METAR" || tokens[0] == "SPECI" {
		idx++
	}
	if idx >= len(tokens) {
		return nil, fmt.Errorf("malformed METAR")
	}
	obs.StationID = tokens[idx]
	idx++

	windRegex := regexp.MustCompile(`^(\d{3}|VRB)(\d{2,3})(?:G(\d{2,3}))?KT$`)
	cloudRegex := regexp.MustCompile(`^(FEW|SCT|BKN|OVC|VV)(\d{3})(?:CB|TCU)?$`)
	tempDewRegex := regexp.MustCompile(`^(M?\d{2})/(M?\d{2})?$`)
	altimeterRegex := regexp.MustCompile(`^A(\d{4})$`)
	altHpaRegex := regexp.MustCompile(`^Q(\d{4})$`)

	for i := idx; i < len(tokens); i++ {
		t := tokens[i]

		// Check for CAVOK
		if t == "CAVOK" {
			obs.VisibilitySM = 10.0
			obs.CeilingAGLFt = 25000.0
			continue
		}

		// Check for clear sky
		if t == "CLR" || t == "SKC" || t == "NSC" || t == "NCD" {
			obs.CeilingAGLFt = 25000.0
			continue
		}

		// Wind (e.g. 18010KT, VRB03KT, 28025G40KT)
		if m := windRegex.FindStringSubmatch(t); m != nil {
			if m[1] == "VRB" {
				obs.WindDirDeg = 0.0
			} else {
				d, _ := strconv.ParseFloat(m[1], 64)
				obs.WindDirDeg = d
			}
			s, _ := strconv.ParseFloat(m[2], 64)
			obs.WindSpeedKts = s
			if m[3] != "" {
				g, _ := strconv.ParseFloat(m[3], 64)
				obs.WindGustKts = g
			}
			continue
		}

		// Compound Visibility Check: e.g. "1" followed by "1/2SM"
		if i+1 < len(tokens) && isIntegerString(t) && strings.HasSuffix(tokens[i+1], "SM") && strings.Contains(tokens[i+1], "/") {
			intVal, _ := strconv.ParseFloat(t, 64)
			fracPart := strings.TrimSuffix(tokens[i+1], "SM")
			parts := strings.Split(fracPart, "/")
			if len(parts) == 2 {
				num, _ := strconv.ParseFloat(parts[0], 64)
				den, _ := strconv.ParseFloat(parts[1], 64)
				if den > 0 {
					obs.VisibilitySM = intVal + (num / den)
					i++ // Skip next token
					continue
				}
			}
		}

		// Standard Visibility (e.g. 10SM, 3/4SM, 1/2SM, M1/4SM, P6SM)
		if strings.HasSuffix(t, "SM") {
			visStr := strings.TrimSuffix(t, "SM")
			if strings.HasPrefix(visStr, "M") {
				// Less than minimum (e.g. M1/4SM -> 0.25)
				visStr = strings.TrimPrefix(visStr, "M")
				if strings.Contains(visStr, "/") {
					parts := strings.Split(visStr, "/")
					if len(parts) == 2 {
						num, _ := strconv.ParseFloat(parts[0], 64)
						den, _ := strconv.ParseFloat(parts[1], 64)
						if den > 0 {
							obs.VisibilitySM = num / den
						}
					}
				} else {
					v, _ := strconv.ParseFloat(visStr, 64)
					obs.VisibilitySM = v
				}
			} else if strings.HasPrefix(visStr, "P") {
				// Greater than maximum (e.g. P6SM -> 6.0)
				visStr = strings.TrimPrefix(visStr, "P")
				v, _ := strconv.ParseFloat(visStr, 64)
				obs.VisibilitySM = v
			} else if strings.Contains(visStr, "/") {
				parts := strings.Split(visStr, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den > 0 {
						obs.VisibilitySM = num / den
					}
				}
			} else {
				v, _ := strconv.ParseFloat(visStr, 64)
				obs.VisibilitySM = v
			}
			continue
		}

		// Cloud Layers & Vertical Visibility (e.g. BKN020, OVC009, VV002)
		if m := cloudRegex.FindStringSubmatch(t); m != nil {
			coverage := m[1]
			hundreds, _ := strconv.ParseFloat(m[2], 64)
			altFt := hundreds * 100.0

			if coverage == "BKN" || coverage == "OVC" || coverage == "VV" {
				if altFt < obs.CeilingAGLFt {
					obs.CeilingAGLFt = altFt
				}
			}
			continue
		}

		// Temperatures (e.g. 18/04, 00/M01, M04/M12)
		if m := tempDewRegex.FindStringSubmatch(t); m != nil {
			obs.TemperatureC = parseSignedTemp(m[1])
			if m[2] != "" {
				obs.DewpointC = parseSignedTemp(m[2])
			}
			continue
		}

		// Altimeter (A2995 or Q1018)
		if m := altimeterRegex.FindStringSubmatch(t); m != nil {
			val, _ := strconv.ParseFloat(m[1], 64)
			obs.AltimeterInHg = val / 100.0
			continue
		}
		if m := altHpaRegex.FindStringSubmatch(t); m != nil {
			val, _ := strconv.ParseFloat(m[1], 64)
			obs.AltimeterInHg = val * 0.02953
			continue
		}
	}

	obs.FlightCategory = determineFlightCategory(obs.CeilingAGLFt, obs.VisibilitySM)
	return obs, nil
}

func isIntegerString(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func parseSignedTemp(s string) float64 {
	neg := false
	if strings.HasPrefix(s, "M") {
		neg = true
		s = strings.TrimPrefix(s, "M")
	}
	val, _ := strconv.ParseFloat(s, 64)
	if neg {
		return -val
	}
	return val
}

func determineFlightCategory(ceilFt, visSM float64) models.FlightCategory {
	if ceilFt < 500.0 || visSM < 1.0 {
		return models.FlightCategoryLIFR
	}
	if ceilFt < 1000.0 || visSM < 3.0 {
		return models.FlightCategoryIFR
	}
	if ceilFt <= 3000.0 || visSM <= 5.0 {
		return models.FlightCategoryMVFR
	}
	return models.FlightCategoryVFR
}

// AuthoritativeTAFParser parses Terminal Aerodrome Forecasts.
type AuthoritativeTAFParser struct{}

func (p *AuthoritativeTAFParser) Parse(raw string) (*ParsedTAF, error) {
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty TAF")
	}

	header := strings.Fields(lines[0])
	stationID := "UNKNOWN"
	for _, tok := range header {
		if len(tok) == 4 && regexp.MustCompile(`^[A-Z]{4}$`).MatchString(tok) {
			stationID = tok
			break
		}
	}

	taf := &ParsedTAF{
		StationID: stationID,
		RawText:   raw,
		Periods:   make([]ParsedTAFPeriod, 0),
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		changeType := "BASE"
		if strings.HasPrefix(line, "FM") {
			changeType = "FM"
		} else if strings.HasPrefix(line, "TEMPO") {
			changeType = "TEMPO"
		} else if strings.HasPrefix(line, "BECMG") {
			changeType = "BECMG"
		} else if strings.HasPrefix(line, "PROB30") {
			changeType = "PROB30"
		} else if strings.HasPrefix(line, "PROB40") {
			changeType = "PROB40"
		}

		period := ParsedTAFPeriod{
			ChangeType:   changeType,
			CeilingAGLFt: 25000.0,
			VisibilitySM: 10.0,
		}

		tokens := strings.Fields(line)
		for _, tok := range tokens {
			// Winds
			if m := regexp.MustCompile(`^(\d{3}|VRB)(\d{2,3})(?:G(\d{2,3}))?KT$`).FindStringSubmatch(tok); m != nil {
				d, _ := strconv.ParseFloat(m[1], 64)
				s, _ := strconv.ParseFloat(m[2], 64)
				period.WindDirDeg = d
				period.WindSpeedKts = s
				if m[3] != "" {
					g, _ := strconv.ParseFloat(m[3], 64)
					period.WindGustKts = g
				}
			}
			// Vis
			if strings.HasSuffix(tok, "SM") {
				vStr := strings.TrimSuffix(tok, "SM")
				if vStr == "P6" {
					period.VisibilitySM = 6.0
				} else if strings.Contains(vStr, "/") {
					parts := strings.Split(vStr, "/")
					if len(parts) == 2 {
						n, _ := strconv.ParseFloat(parts[0], 64)
						d, _ := strconv.ParseFloat(parts[1], 64)
						if d > 0 {
							period.VisibilitySM = n / d
						}
					}
				} else {
					v, _ := strconv.ParseFloat(vStr, 64)
					period.VisibilitySM = v
				}
			}
			// Clouds / Ceilings
			if m := regexp.MustCompile(`^(FEW|SCT|BKN|OVC|VV)(\d{3})(?:CB|TCU)?$`).FindStringSubmatch(tok); m != nil {
				cov := m[1]
				h, _ := strconv.ParseFloat(m[2], 64)
				if cov == "BKN" || cov == "OVC" || cov == "VV" {
					if h*100.0 < period.CeilingAGLFt {
						period.CeilingAGLFt = h * 100.0
					}
				}
			}
			// Weather phenomena
			if strings.Contains(tok, "TS") || strings.Contains(tok, "RA") || strings.Contains(tok, "SN") {
				period.WeatherPhenomena = append(period.WeatherPhenomena, tok)
			}
		}

		taf.Periods = append(taf.Periods, period)
	}

	return taf, nil
}

// AuthoritativeWindsAloftDecoder decodes fixed-grid FD winds aloft bulletins.
type AuthoritativeWindsAloftDecoder struct{}

func (d *AuthoritativeWindsAloftDecoder) DecodeFDTable(rawTable string, station string) ([]DecodedWindVector, error) {
	lines := strings.Split(rawTable, "\n")
	var altHeader []float64
	var vectors []DecodedWindVector

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FT") {
			fields := strings.Fields(line)[1:]
			for _, f := range fields {
				alt, err := strconv.ParseFloat(f, 64)
				if err == nil {
					altHeader = append(altHeader, alt)
				}
			}
			continue
		}

		if strings.HasPrefix(line, station) {
			entries := strings.Fields(line)[1:]
			for i, entry := range entries {
				if i >= len(altHeader) {
					break
				}
				alt := altHeader[i]
				vec, err := d.DecodeEntry(entry, alt)
				if err == nil {
					vectors = append(vectors, vec)
				}
			}
		}
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("no winds aloft data found for station %s", station)
	}
	return vectors, nil
}

func (d *AuthoritativeWindsAloftDecoder) DecodeEntry(entry string, altMSL float64) (DecodedWindVector, error) {
	entry = strings.TrimSpace(entry)
	if entry == "" || entry == " " {
		return DecodedWindVector{}, fmt.Errorf("empty entry")
	}

	// Calm wind
	if strings.HasPrefix(entry, "9900") {
		temp := 0.0
		if len(entry) >= 7 {
			temp, _ = strconv.ParseFloat(entry[4:], 64)
		}
		return DecodedWindVector{
			AltitudeMSLFt: altMSL,
			DirectionDeg:  0.0,
			SpeedKts:      0.0,
			TemperatureC:  temp,
			IsCalm:        true,
		}, nil
	}

	if len(entry) < 4 {
		return DecodedWindVector{}, fmt.Errorf("invalid entry length: %s", entry)
	}

	dirCode, err := strconv.ParseFloat(entry[0:2], 64)
	if err != nil {
		return DecodedWindVector{}, err
	}
	speed, err := strconv.ParseFloat(entry[2:4], 64)
	if err != nil {
		return DecodedWindVector{}, err
	}

	// Speed >= 100 kt offset (+50 rule)
	if dirCode > 50 {
		dirCode -= 50
		speed += 100
	}
	dirDeg := dirCode * 10.0

	// Temperature
	temp := 0.0
	if len(entry) >= 6 {
		tempStr := entry[4:]
		if altMSL >= 24000.0 && !strings.HasPrefix(tempStr, "+") && !strings.HasPrefix(tempStr, "-") {
			// Sign is negative above 24,000 ft MSL
			tVal, _ := strconv.ParseFloat(tempStr, 64)
			temp = -tVal
		} else {
			temp, _ = strconv.ParseFloat(tempStr, 64)
		}
	}

	return DecodedWindVector{
		AltitudeMSLFt: altMSL,
		DirectionDeg:  dirDeg,
		SpeedKts:      speed,
		TemperatureC:  temp,
		IsCalm:        false,
	}, nil
}

// InterpolateWindsVector executes trigonometric u/v interpolation between altitude strata.
func InterpolateWindsVector(targetAltMSL float64, layers []DecodedWindVector) DecodedWindVector {
	if len(layers) == 0 {
		return DecodedWindVector{AltitudeMSLFt: targetAltMSL}
	}
	if targetAltMSL <= layers[0].AltitudeMSLFt {
		res := layers[0]
		res.AltitudeMSLFt = targetAltMSL
		return res
	}
	if targetAltMSL >= layers[len(layers)-1].AltitudeMSLFt {
		res := layers[len(layers)-1]
		res.AltitudeMSLFt = targetAltMSL
		return res
	}

	// Find bounding layers
	var lower, upper DecodedWindVector
	for i := 0; i < len(layers)-1; i++ {
		if targetAltMSL >= layers[i].AltitudeMSLFt && targetAltMSL <= layers[i+1].AltitudeMSLFt {
			lower = layers[i]
			upper = layers[i+1]
			break
		}
	}

	factor := (targetAltMSL - lower.AltitudeMSLFt) / (upper.AltitudeMSLFt - lower.AltitudeMSLFt)

	// Trigonometric decomposition into Cartesian u and v components
	// u = -Speed * sin(rad), v = -Speed * cos(rad)
	rad1 := lower.DirectionDeg * math.Pi / 180.0
	u1 := -lower.SpeedKts * math.Sin(rad1)
	v1 := -lower.SpeedKts * math.Cos(rad1)

	rad2 := upper.DirectionDeg * math.Pi / 180.0
	u2 := -upper.SpeedKts * math.Sin(rad2)
	v2 := -upper.SpeedKts * math.Cos(rad2)

	// Linear interpolation in Cartesian space
	uTarget := u1 + factor*(u2-u1)
	vTarget := v1 + factor*(v2-v1)

	speedTarget := math.Sqrt(uTarget*uTarget + vTarget*vTarget)
	dirTarget := math.Atan2(-uTarget, -vTarget) * 180.0 / math.Pi
	if dirTarget < 0 {
		dirTarget += 360.0
	}
	tempTarget := lower.TemperatureC + factor*(upper.TemperatureC-lower.TemperatureC)

	return DecodedWindVector{
		AltitudeMSLFt: targetAltMSL,
		DirectionDeg:  math.Round(dirTarget),
		SpeedKts:      math.Round(speedTarget),
		TemperatureC:  math.Round(tempTarget*10) / 10,
	}
}

// AuthoritativeCorridorEvaluator evaluates multi-station corridor weather, SIGMETs, and Part 135.L rules.
type AuthoritativeCorridorEvaluator struct{}

type CorridorEvaluationRequest struct {
	CorridorStations      []string
	StationMETARs         map[string]*models.METAR
	ActiveHazards         []ParsedSIGMET
	IsDay                 bool
	IsMountainous         bool
	IsLocal               bool
	NVGOrHTAWSEquipped    bool
	AllowOpSpecA021Relief bool
	IsIFRCertified        bool
	HasIFRAlternate       bool
	HasSevereWeatherRadar bool
}

type CorridorEvaluationResult struct {
	Verdict              DispatchVerdict
	LimitingFactor       LimitingFactor
	WorstCeilingStation  string
	WorstCeilingFt       float64
	WorstVisStation      string
	WorstVisSM           float64
	MinRequiredCeilingFt float64
	MinRequiredVisSM     float64
	LegalRuleApplied     string
	Explanation          string
}

func (e *AuthoritativeCorridorEvaluator) Evaluate(req CorridorEvaluationRequest) CorridorEvaluationResult {
	// Stage 1: Active Severe Hazards (Convective SIGMETs, Severe Icing, Severe Turbulence)
	for _, h := range req.ActiveHazards {
		if !h.Active {
			continue
		}
		if h.HazardType == "CONVECTIVE" {
			return CorridorEvaluationResult{
				Verdict:          VerdictDisqualified,
				LimitingFactor:   LimitConvectiveSIGMET,
				LegalRuleApplied: "FAA Safety Disqualification: Convective Storm Hazard",
				Explanation:      fmt.Sprintf("DISQUALIFIED: Flight corridor intersects active Convective SIGMET %s: %s", h.AdvisoryID, h.Summary),
			}
		}
		if h.HazardType == "SEVERE_ICING" {
			return CorridorEvaluationResult{
				Verdict:          VerdictDisqualified,
				LimitingFactor:   LimitSevereIcing,
				LegalRuleApplied: "FAA AC 91-74B: Known Severe Icing Prohibited",
				Explanation:      fmt.Sprintf("DISQUALIFIED: Active Severe Icing advisory %s: %s", h.AdvisoryID, h.Summary),
			}
		}
		if h.HazardType == "SEVERE_TURB" {
			return CorridorEvaluationResult{
				Verdict:          VerdictDisqualified,
				LimitingFactor:   LimitSevereTurbulence,
				LegalRuleApplied: "HEMS Patient Safety Directive: Severe Mountain Turbulence",
				Explanation:      fmt.Sprintf("DISQUALIFIED: Severe turbulence hazard %s: %s", h.AdvisoryID, h.Summary),
			}
		}
	}

	// Stage 2: Aggregate Worst-Case Ceiling & Visibility across Corridor Stations
	worstCeil := 25000.0
	worstCeilStation := ""
	worstVis := 10.0
	worstVisStation := ""
	maxGust := 0.0

	for _, st := range req.CorridorStations {
		metar, ok := req.StationMETARs[st]
		if !ok || metar == nil {
			continue
		}
		if metar.CeilingAGLFt < worstCeil {
			worstCeil = metar.CeilingAGLFt
			worstCeilStation = st
		}
		if metar.VisibilitySM < worstVis {
			worstVis = metar.VisibilitySM
			worstVisStation = st
		}
		if metar.WindGustKts > maxGust {
			maxGust = metar.WindGustKts
		}
	}

	// Stage 3: Regulatory Rule Lookup (14 CFR § 135.609 Table 1)
	rule := regulations.GetFAR135615Minimums(req.IsDay, req.IsMountainous, req.IsLocal, req.NVGOrHTAWSEquipped)

	ceilMargin := worstCeil - rule.MinCeilingFt
	visMargin := worstVis - rule.MinVisibilitySM

	ceilOK := ceilMargin >= 0.0
	visOK := visMargin >= 0.0

	legalCite := fmt.Sprintf("14 CFR § 135.609 Table 1 [%s - %s]", rule.Category, rule.Condition)

	// Stage 4: Violation Checks & § 135.611 IFR Transition
	if !ceilOK || !visOK {
		lim := LimitCeilingBelowMinimums
		if ceilOK && !visOK {
			lim = LimitVisibilityBelowMinimums
		}

		// Can mission transition to IFR under § 135.611?
		if req.IsIFRCertified && req.HasIFRAlternate && req.HasSevereWeatherRadar {
			return CorridorEvaluationResult{
				Verdict:              VerdictIFRRequired,
				LimitingFactor:       lim,
				WorstCeilingStation:  worstCeilStation,
				WorstCeilingFt:       worstCeil,
				WorstVisStation:      worstVisStation,
				WorstVisSM:           worstVis,
				MinRequiredCeilingFt: rule.MinCeilingFt,
				MinRequiredVisSM:     rule.MinVisibilitySM,
				LegalRuleApplied:     "14 CFR § 135.611 IFR Operations Authorized",
				Explanation: fmt.Sprintf("VFR weather below § 135.609 minimums at %s/%s (Ceiling: %.0f ft, Vis: %.1f SM), but compliant for IFR transition under § 135.611 with alternate and radar.",
					worstCeilStation, worstVisStation, worstCeil, worstVis),
			}
		}

		return CorridorEvaluationResult{
			Verdict:              VerdictBelowMinimums,
			LimitingFactor:       lim,
			WorstCeilingStation:  worstCeilStation,
			WorstCeilingFt:       worstCeil,
			WorstVisStation:      worstVisStation,
			WorstVisSM:           worstVis,
			MinRequiredCeilingFt: rule.MinCeilingFt,
			MinRequiredVisSM:     rule.MinVisibilitySM,
			LegalRuleApplied:     legalCite,
			Explanation: fmt.Sprintf("WEATHER VIOLATION under %s: Worst ceiling %.0f ft at %s (req %.0f ft) | Worst vis %.1f SM at %s (req %.1f SM)",
				legalCite, worstCeil, worstCeilStation, rule.MinCeilingFt, worstVis, worstVisStation, rule.MinVisibilitySM),
		}
	}

	// Stage 5: Operational Caution / Marginal Status
	isMarginal := ceilMargin <= 200.0 || visMargin <= 1.0 || maxGust >= 25.0
	if isMarginal {
		return CorridorEvaluationResult{
			Verdict:              VerdictMarginal,
			LimitingFactor:       LimitNone,
			WorstCeilingStation:  worstCeilStation,
			WorstCeilingFt:       worstCeil,
			WorstVisStation:      worstVisStation,
			WorstVisSM:           worstVis,
			MinRequiredCeilingFt: rule.MinCeilingFt,
			MinRequiredVisSM:     rule.MinVisibilitySM,
			LegalRuleApplied:     legalCite,
			Explanation: fmt.Sprintf("MARGINAL WEATHER under %s: Ceiling margin +%.0f ft, Vis margin +%.1f SM (Caution buffer: scud-running risk or gust %.0f kt)",
				legalCite, ceilMargin, visMargin, maxGust),
		}
	}

	// Stage 6: Approved / Full Go
	return CorridorEvaluationResult{
		Verdict:              VerdictApproved,
		LimitingFactor:       LimitNone,
		WorstCeilingStation:  worstCeilStation,
		WorstCeilingFt:       worstCeil,
		WorstVisStation:      worstVisStation,
		WorstVisSM:           worstVis,
		MinRequiredCeilingFt: rule.MinCeilingFt,
		MinRequiredVisSM:     rule.MinVisibilitySM,
		LegalRuleApplied:     legalCite,
		Explanation:          fmt.Sprintf("Compliant with %s: Ceiling margin +%.0f ft, Vis margin +%.1f SM", legalCite, ceilMargin, visMargin),
	}
}
