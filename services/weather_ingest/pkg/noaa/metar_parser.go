package noaa

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"traumateam/weather_ingest/pkg/models"
)

var (
	stationRegex   = regexp.MustCompile(`^[A-Z0-9]{4}$`)
	timeRegex      = regexp.MustCompile(`^(\d{2})(\d{2})(\d{2})Z$`)
	windRegex      = regexp.MustCompile(`^(\d{3}|VRB)(\d{2,3})(?:G(\d{2,3}))?KT$`)
	windVarRegex   = regexp.MustCompile(`^(\d{3})V(\d{3})$`)
	cloudRegex     = regexp.MustCompile(`^(FEW|SCT|BKN|OVC|VV)(\d{3})(CB|TCU)?$`)
	tempDewRegex   = regexp.MustCompile(`^(M?\d{2})/((?:M?\d{2})|//)?$`)
	altimeterInHg  = regexp.MustCompile(`^A(\d{4})$`)
	altimeterHPa   = regexp.MustCompile(`^Q(\d{4})$`)
	visFractionReg = regexp.MustCompile(`^(?:M)?(\d+)/(\d+)SM$`)
	visIntegerReg  = regexp.MustCompile(`^(?:P)?(\d+)SM$`)
)

// ParseMETAR parses a raw NOAA AWC METAR/SPECI string into a structured models.METAR struct.
func ParseMETAR(raw string) (*models.METAR, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty METAR string")
	}

	rawTokens := strings.Fields(raw)
	if len(rawTokens) < 2 {
		return nil, fmt.Errorf("insufficient METAR tokens: %q", raw)
	}

	// Pre-process tokens to merge compound visibility tokens like "1" + "1/2SM" -> "1 1/2SM"
	tokens := make([]string, 0, len(rawTokens))
	for i := 0; i < len(rawTokens); i++ {
		t := rawTokens[i]
		if i+1 < len(rawTokens) && isIntegerString(t) && strings.HasSuffix(rawTokens[i+1], "SM") && strings.Contains(rawTokens[i+1], "/") {
			tokens = append(tokens, t+" "+rawTokens[i+1])
			i++
		} else {
			tokens = append(tokens, t)
		}
	}

	obsTime := time.Now().UTC()
	metar := &models.METAR{
		RawText:            raw,
		ObservationTimeUTC: obsTime,
		CeilingAGLFt:       25000.0, // Default unlimited ceiling
		HasCeiling:         false,
		VisibilitySM:       10.0,
		AltimeterInHg:      29.92,
		CloudLayers:        make([]models.CloudLayer, 0),
		PresentWeather:     make([]string, 0),
	}

	// Step 1: Skip leading "METAR" or "SPECI" type identifier
	startIdx := 0
	if rawTokens[0] == "METAR" || rawTokens[0] == "SPECI" {
		startIdx = 1
	}
	if startIdx >= len(tokens) {
		return nil, fmt.Errorf("missing station identifier in %q", raw)
	}

	// Station ID
	metar.StationID = tokens[startIdx]
	startIdx++

	// Step 2: Iterate over tokens
	for i := startIdx; i < len(tokens); i++ {
		t := tokens[i]

		// Skip Remarks section
		if t == "RMK" {
			break
		}

		// Modifiers
		if t == "AUTO" || t == "COR" || t == "NIL" {
			continue
		}

		// Observation time (e.g. 050550Z)
		if m := timeRegex.FindStringSubmatch(t); m != nil {
			day, _ := strconv.Atoi(m[1])
			hour, _ := strconv.Atoi(m[2])
			min, _ := strconv.Atoi(m[3])

			year, month, _ := obsTime.Date()
			metar.ObservationTimeUTC = time.Date(year, month, day, hour, min, 0, 0, time.UTC)
			continue
		}

		// Wind (e.g. 18010KT, 28022G35KT, VRB03KT, 00000KT)
		if m := windRegex.FindStringSubmatch(t); m != nil {
			if m[1] == "VRB" {
				metar.IsVariableWind = true
				metar.WindDirDeg = 0.0
			} else {
				if d, err := strconv.ParseFloat(m[1], 64); err == nil {
					metar.WindDirDeg = d
				}
			}
			if s, err := strconv.ParseFloat(m[2], 64); err == nil {
				metar.WindSpeedKts = s
			}
			if m[3] != "" {
				if g, err := strconv.ParseFloat(m[3], 64); err == nil {
					metar.WindGustKts = g
				}
			}
			continue
		}

		// Wind variation range (e.g. 140V220)
		if m := windVarRegex.FindStringSubmatch(t); m != nil {
			metar.IsVariableWind = true
			if minD, err := strconv.ParseFloat(m[1], 64); err == nil {
				metar.VarMinDeg = minD
			}
			if maxD, err := strconv.ParseFloat(m[2], 64); err == nil {
				metar.VarMaxDeg = maxD
			}
			continue
		}

		// CAVOK
		if t == "CAVOK" {
			metar.VisibilitySM = 10.0
			metar.HasCeiling = false
			metar.CeilingAGLFt = 25000.0
			continue
		}

		// Clear skies
		if t == "CLR" || t == "SKC" || t == "NSC" || t == "NCD" {
			metar.HasCeiling = false
			metar.CeilingAGLFt = 25000.0
			metar.CloudLayers = append(metar.CloudLayers, models.CloudLayer{
				Coverage:  models.CloudCoverage(t),
				BaseFtAGL: 25000.0,
				IsCeiling: false,
			})
			continue
		}

		// Visibility: handles "10SM", "P6SM", "M1/4SM", "1/2SM", "1 1/2SM"
		if strings.HasSuffix(t, "SM") {
			vis, lessThan, greaterThan, ok := parseVisibility(t)
			if ok {
				metar.VisibilitySM = vis
				metar.VisibilityLessThan = lessThan
				metar.VisibilityGreaterThan = greaterThan
				continue
			}
		}

		// Cloud layers (e.g. FEW010, SCT020, BKN035, OVC050, VV002)
		if m := cloudRegex.FindStringSubmatch(t); m != nil {
			coverage := models.CloudCoverage(m[1])
			hundreds, _ := strconv.ParseFloat(m[2], 64)
			baseFt := hundreds * 100.0
			modifier := m[3]

			isCeiling := (coverage == models.CoverageBKN || coverage == models.CoverageOVC || coverage == models.CoverageVV)

			layer := models.CloudLayer{
				Coverage:  coverage,
				BaseFtAGL: baseFt,
				IsCeiling: isCeiling,
				Modifier:  modifier,
			}
			metar.CloudLayers = append(metar.CloudLayers, layer)

			// Ceiling rule: lowest BKN, OVC, or VV
			if isCeiling {
				if !metar.HasCeiling || baseFt < metar.CeilingAGLFt {
					metar.CeilingAGLFt = baseFt
					metar.HasCeiling = true
				}
			}
			continue
		}

		// Temperature / Dewpoint (e.g. 18/04, 00/M01, M04/M12, 15/M//, 15/)
		if m := tempDewRegex.FindStringSubmatch(t); m != nil {
			metar.TemperatureC = parseSignedTemp(m[1])
			if m[2] != "" && m[2] != "//" {
				metar.DewpointC = parseSignedTemp(m[2])
			}
			continue
		}

		// Altimeter (A2992 -> 29.92 inHg, Q1013 -> 29.91 inHg)
		if m := altimeterInHg.FindStringSubmatch(t); m != nil {
			if val, err := strconv.ParseFloat(m[1], 64); err == nil {
				metar.AltimeterInHg = val / 100.0
			}
			continue
		}
		if m := altimeterHPa.FindStringSubmatch(t); m != nil {
			if hpa, err := strconv.ParseFloat(m[1], 64); err == nil {
				metar.AltimeterInHg = hpa * 0.0295299830714
			}
			continue
		}

		// Weather Phenomena (e.g. -SN, +TSRA, FG, FZFG, BR, HZ)
		if isWeatherPhenomenon(t) {
			metar.PresentWeather = append(metar.PresentWeather, t)
			continue
		}
	}

	// Compute Flight Category (VFR, MVFR, IFR, LIFR)
	metar.FlightCategory = determineFlightCategory(metar.CeilingAGLFt, metar.HasCeiling, metar.VisibilitySM)

	// Pressure altitude & density altitude calculations
	metar.PressureAltitudeFt = PressureAltitudeFt(0, metar.AltimeterInHg)
	metar.DensityAltitudeFt = DensityAltitudeFt(metar.PressureAltitudeFt, metar.TemperatureC)

	return metar, nil
}

func parseVisibility(s string) (vis float64, lessThan bool, greaterThan bool, ok bool) {
	s = strings.TrimSuffix(s, "SM")

	if strings.HasPrefix(s, "M") {
		lessThan = true
		s = strings.TrimPrefix(s, "M")
	} else if strings.HasPrefix(s, "P") {
		greaterThan = true
		s = strings.TrimPrefix(s, "P")
	}

	// Compound fraction: "1 1/2"
	if strings.Contains(s, " ") {
		parts := strings.Split(s, " ")
		if len(parts) == 2 {
			whole, err1 := strconv.ParseFloat(parts[0], 64)
			frac, okFrac := parseFraction(parts[1])
			if err1 == nil && okFrac {
				return whole + frac, lessThan, greaterThan, true
			}
		}
	}

	// Simple fraction: "1/2", "1/4", "3/4"
	if strings.Contains(s, "/") {
		frac, okFrac := parseFraction(s)
		if okFrac {
			return frac, lessThan, greaterThan, true
		}
	}

	// Simple integer: "10", "3", "6"
	val, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return val, lessThan, greaterThan, true
	}

	return 10.0, false, false, false
}

func parseFraction(s string) (float64, bool) {
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		num, err1 := strconv.ParseFloat(parts[0], 64)
		den, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && den > 0 {
			return num / den, true
		}
	}
	return 0, false
}

func isIntegerString(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func isWeatherPhenomenon(token string) bool {
	if strings.HasPrefix(token, "-") || strings.HasPrefix(token, "+") || strings.HasPrefix(token, "VC") {
		return true
	}
	phenomena := []string{"TS", "RA", "SN", "DZ", "SG", "IC", "PL", "GR", "GS", "UP", "FG", "BR", "FZFG", "HZ", "FU", "VA", "DU", "SA", "PY", "SQ", "FC", "SS", "DS"}
	for _, p := range phenomena {
		if strings.Contains(token, p) {
			return true
		}
	}
	return false
}

func determineFlightCategory(ceilFt float64, hasCeiling bool, visSM float64) models.FlightCategory {
	// If no ceiling, ceiling does not restrict flight category
	effectiveCeil := ceilFt
	if !hasCeiling {
		effectiveCeil = 25000.0
	}

	if effectiveCeil < 500.0 || visSM < 1.0 {
		return models.FlightCategoryLIFR
	}
	if effectiveCeil < 1000.0 || visSM < 3.0 {
		return models.FlightCategoryIFR
	}
	if effectiveCeil <= 3000.0 || visSM <= 5.0 {
		return models.FlightCategoryMVFR
	}
	return models.FlightCategoryVFR
}

// DensityAltitudeFt calculates density altitude given pressure altitude and temperature in Celsius.
func DensityAltitudeFt(pressureAltFt, tempC float64) float64 {
	isaC := 15.0 - (0.0019812 * pressureAltFt)
	deltaISA := tempC - isaC
	return pressureAltFt + (118.8 * deltaISA)
}

// PressureAltitudeFt calculates pressure altitude from station elevation and altimeter setting in inHg.
func PressureAltitudeFt(elevFt, altimeterInHg float64) float64 {
	return elevFt + (29.92126-altimeterInHg)*1000.0
}

func parseSignedTemp(s string) float64 {
	isNeg := false
	if strings.HasPrefix(s, "M") {
		isNeg = true
		s = strings.TrimPrefix(s, "M")
	}
	val, _ := strconv.ParseFloat(s, 64)
	if isNeg {
		return -val
	}
	return val
}

