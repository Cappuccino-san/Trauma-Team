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
	tafValidRegex = regexp.MustCompile(`^(\d{2})(\d{2})/(\d{2})(\d{2})$`)
	tafFMRegex    = regexp.MustCompile(`^FM(\d{2})(\d{2})(\d{2})$`)
	tafTempoRegex = regexp.MustCompile(`^(TEMPO|PROB30|PROB40|BECMG)$`)
)

// ParseTAF decodes a raw NOAA AWC Terminal Aerodrome Forecast bulletin into a structured models.TAF.
func ParseTAF(raw string) (*models.TAF, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty TAF string")
	}

	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("no lines in TAF string")
	}

	headerTokens := strings.Fields(lines[0])
	if len(headerTokens) < 3 {
		return nil, fmt.Errorf("insufficient TAF header tokens: %q", lines[0])
	}

	now := time.Now().UTC()
	taf := &models.TAF{
		RawText:      raw,
		IssueTimeUTC: now,
		Periods:      make([]models.TAFPeriod, 0),
	}

	idx := 0
	if headerTokens[idx] == "TAF" {
		idx++
	}
	if idx < len(headerTokens) && (headerTokens[idx] == "AMD" || headerTokens[idx] == "COR") {
		idx++
	}
	if idx >= len(headerTokens) {
		return nil, fmt.Errorf("missing station in TAF header")
	}

	taf.StationID = headerTokens[idx]
	idx++

	// Parse Issue Time (e.g. 051120Z)
	if idx < len(headerTokens) && timeRegex.MatchString(headerTokens[idx]) {
		m := timeRegex.FindStringSubmatch(headerTokens[idx])
		day, _ := strconv.Atoi(m[1])
		hour, _ := strconv.Atoi(m[2])
		min, _ := strconv.Atoi(m[3])
		taf.IssueTimeUTC = time.Date(now.Year(), now.Month(), day, hour, min, 0, 0, time.UTC)
		idx++
	}

	// Parse Valid Range (e.g. 0512/0618)
	if idx < len(headerTokens) && tafValidRegex.MatchString(headerTokens[idx]) {
		m := tafValidRegex.FindStringSubmatch(headerTokens[idx])
		startDay, _ := strconv.Atoi(m[1])
		startHour, _ := strconv.Atoi(m[2])
		endDay, _ := strconv.Atoi(m[3])
		endHour, _ := strconv.Atoi(m[4])

		taf.ValidFromUTC = time.Date(now.Year(), now.Month(), startDay, startHour, 0, 0, 0, time.UTC)
		taf.ValidToUTC = time.Date(now.Year(), now.Month(), endDay, endHour, 0, 0, 0, time.UTC)
		idx++
	}

	// Remainder of line 0 is part of BASE forecast
	baseTokens := make([]string, 0)
	for i := idx; i < len(headerTokens); i++ {
		baseTokens = append(baseTokens, headerTokens[i])
	}

	// Parse Base Period
	basePeriod := parseForecastTokens(baseTokens, models.TAFChangeBase, taf.ValidFromUTC, taf.ValidToUTC)
	taf.Periods = append(taf.Periods, basePeriod)

	// Process subsequent lines
	currentPeriodStart := taf.ValidFromUTC
	for lIdx := 1; lIdx < len(lines); lIdx++ {
		line := strings.TrimSpace(lines[lIdx])
		if line == "" {
			continue
		}

		lineTokens := strings.Fields(line)
		if len(lineTokens) == 0 {
			continue
		}

		firstTok := lineTokens[0]
		if m := tafFMRegex.FindStringSubmatch(firstTok); m != nil {
			// FM group: FM051600
			day, _ := strconv.Atoi(m[1])
			hour, _ := strconv.Atoi(m[2])
			min, _ := strconv.Atoi(m[3])
			fmStart := time.Date(now.Year(), now.Month(), day, hour, min, 0, 0, time.UTC)
			currentPeriodStart = fmStart

			p := parseForecastTokens(lineTokens[1:], models.TAFChangeFM, fmStart, taf.ValidToUTC)
			taf.Periods = append(taf.Periods, p)
		} else if tafTempoRegex.MatchString(firstTok) {
			// TEMPO, PROB30, PROB40, BECMG
			changeType := models.TAFChangeType(firstTok)
			pStart := currentPeriodStart
			pEnd := taf.ValidToUTC
			bodyIdx := 1

			if len(lineTokens) > 1 && tafValidRegex.MatchString(lineTokens[1]) {
				m := tafValidRegex.FindStringSubmatch(lineTokens[1])
				sDay, _ := strconv.Atoi(m[1])
				sHour, _ := strconv.Atoi(m[2])
				eDay, _ := strconv.Atoi(m[3])
				eHour, _ := strconv.Atoi(m[4])
				pStart = time.Date(now.Year(), now.Month(), sDay, sHour, 0, 0, 0, time.UTC)
				pEnd = time.Date(now.Year(), now.Month(), eDay, eHour, 0, 0, 0, time.UTC)
				bodyIdx = 2
			}

			p := parseForecastTokens(lineTokens[bodyIdx:], changeType, pStart, pEnd)
			taf.Periods = append(taf.Periods, p)
		}
	}

	return taf, nil
}

func parseForecastTokens(rawTokens []string, changeType models.TAFChangeType, start, end time.Time) models.TAFPeriod {
	p := models.TAFPeriod{
		ChangeType:     changeType,
		ValidFromUTC:   start,
		ValidToUTC:     end,
		WindDirDeg:     0,
		WindSpeedKts:   0,
		VisibilitySM:   10.0,
		CeilingAGLFt:   25000.0,
		HasCeiling:     false,
		CloudLayers:    make([]models.CloudLayer, 0),
		PresentWeather: make([]string, 0),
	}

	// Merge compound fraction visibilities
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

	for _, t := range tokens {
		// Wind
		if m := windRegex.FindStringSubmatch(t); m != nil {
			if m[1] != "VRB" {
				if d, err := strconv.ParseFloat(m[1], 64); err == nil {
					p.WindDirDeg = d
				}
			}
			if s, err := strconv.ParseFloat(m[2], 64); err == nil {
				p.WindSpeedKts = s
			}
			if m[3] != "" {
				if g, err := strconv.ParseFloat(m[3], 64); err == nil {
					p.WindGustKts = g
				}
			}
			continue
		}

		// Visibility
		if strings.HasSuffix(t, "SM") {
			vis, _, _, ok := parseVisibility(t)
			if ok {
				p.VisibilitySM = vis
				continue
			}
		}

		// Clear skies
		if t == "CLR" || t == "SKC" || t == "NSC" {
			p.HasCeiling = false
			p.CeilingAGLFt = 25000.0
			continue
		}

		// Cloud layers
		if m := cloudRegex.FindStringSubmatch(t); m != nil {
			coverage := models.CloudCoverage(m[1])
			hundreds, _ := strconv.ParseFloat(m[2], 64)
			baseFt := hundreds * 100.0
			modifier := m[3]

			isCeiling := (coverage == models.CoverageBKN || coverage == models.CoverageOVC || coverage == models.CoverageVV)
			p.CloudLayers = append(p.CloudLayers, models.CloudLayer{
				Coverage:  coverage,
				BaseFtAGL: baseFt,
				IsCeiling: isCeiling,
				Modifier:  modifier,
			})
			if isCeiling && (!p.HasCeiling || baseFt < p.CeilingAGLFt) {
				p.CeilingAGLFt = baseFt
				p.HasCeiling = true
			}
			continue
		}

		// Weather Phenomena
		if isWeatherPhenomenon(t) {
			p.PresentWeather = append(p.PresentWeather, t)
			continue
		}
	}

	p.FlightCategory = determineFlightCategory(p.CeilingAGLFt, p.HasCeiling, p.VisibilitySM)
	return p
}

