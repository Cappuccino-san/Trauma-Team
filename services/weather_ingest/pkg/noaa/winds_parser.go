package noaa

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"traumateam/weather_ingest/pkg/models"
)

// ParseWindsAloft decodes a raw NOAA FD winds aloft text report into station forecasts.
func ParseWindsAloft(raw string) ([]models.WindsAloftForecast, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty winds aloft report")
	}

	lines := strings.Split(raw, "\n")
	var headerLine string
	var headerIdx int = -1

	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "FT") {
			headerLine = l
			headerIdx = i
			break
		}
	}

	if headerIdx == -1 {
		return nil, fmt.Errorf("missing FT altitude header line in winds aloft")
	}

	// Parse altitudes from header line (e.g. "FT  3000 6000 9000 12000 18000 24000 30000 34000 39000")
	// Track column start indices for reliable fixed-column extraction
	headerTokens := strings.Fields(headerLine)
	if len(headerTokens) < 2 {
		return nil, fmt.Errorf("invalid FT header format: %q", headerLine)
	}

	type altCol struct {
		altFt   float64
		colIdx  int // column index
		charPos int
	}
	altCols := make([]altCol, 0)
	for i := 1; i < len(headerTokens); i++ {
		if alt, err := strconv.ParseFloat(headerTokens[i], 64); err == nil {
			pos := strings.Index(headerLine, headerTokens[i])
			altCols = append(altCols, altCol{
				altFt:   alt,
				colIdx:  i - 1,
				charPos: pos,
			})
		}
	}

	forecasts := make([]models.WindsAloftForecast, 0)

	// Process station data lines after header
	for i := headerIdx + 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r\n")
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		stationID := fields[0]
		// Skip non-station lines
		if len(stationID) < 3 || len(stationID) > 4 {
			continue
		}

		layers := make([]models.WindsAloftLayer, 0)

		dataFields := fields[1:]
		startColIdx := 0
		// If first data field is not a 4-digit 3,000 ft report, it starts at 6,000 ft
		if len(dataFields) > 0 && len(dataFields[0]) > 4 {
			startColIdx = 1
		} else if len(dataFields) == len(altCols)-1 {
			startColIdx = 1
		}

		for idx, f := range dataFields {
			colIdx := startColIdx + idx
			if colIdx < len(altCols) {
				layer, ok := decodeWindsAloftEntry(f, altCols[colIdx].altFt)
				if ok {
					layers = append(layers, layer)
				}
			}
		}

		if len(layers) > 0 {
			forecasts = append(forecasts, models.WindsAloftForecast{
				StationID:          stationID,
				ObservationTimeUTC: time.Now().UTC(),
				Layers:             layers,
				RawText:            line,
			})
		}
	}

	return forecasts, nil
}

func extractColumnToken(line string, charPos, maxLen int) string {
	if charPos >= len(line) {
		return ""
	}
	start := charPos - 2
	if start < 0 {
		start = 0
	}
	end := charPos + maxLen
	if end > len(line) {
		end = len(line)
	}
	snippet := line[start:end]
	tokens := strings.Fields(snippet)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return ""
}

// decodeWindsAloftEntry decodes a single winds aloft token (e.g. "1812+15", "751548", "9900", "2820").
func decodeWindsAloftEntry(token string, altFt float64) (models.WindsAloftLayer, bool) {
	token = strings.TrimSpace(token)
	if token == "" || strings.HasPrefix(token, "//") || len(token) < 4 {
		return models.WindsAloftLayer{}, false
	}

	layer := models.WindsAloftLayer{
		AltitudeMSLFt: altFt,
	}

	// Calm wind: "9900" or "9900+00"
	if strings.HasPrefix(token, "9900") {
		layer.DirectionDeg = 0
		layer.SpeedKts = 0
		if len(token) > 4 {
			layer.TemperatureC = parseFDTemp(token[4:], altFt)
		}
		return layer, true
	}

	dirCode, err1 := strconv.ParseFloat(token[0:2], 64)
	speedVal, err2 := strconv.ParseFloat(token[2:4], 64)
	if err1 != nil || err2 != nil {
		return models.WindsAloftLayer{}, false
	}

	// Speed >= 100 kts: direction code > 50
	if dirCode > 50 {
		layer.DirectionDeg = (dirCode - 50) * 10
		layer.SpeedKts = speedVal + 100
	} else {
		layer.DirectionDeg = dirCode * 10
		layer.SpeedKts = speedVal
	}

	// Parse Temperature if present (at >= 6000 ft)
	if len(token) > 4 {
		layer.TemperatureC = parseFDTemp(token[4:], altFt)
	}

	return layer, true
}

func parseFDTemp(tempStr string, altFt float64) float64 {
	tempStr = strings.TrimSpace(tempStr)
	if tempStr == "" {
		return 0.0
	}

	// Signed temp: e.g. "+15", "-08"
	if strings.HasPrefix(tempStr, "+") || strings.HasPrefix(tempStr, "-") {
		val, _ := strconv.ParseFloat(tempStr, 64)
		return val
	}

	// Above 24,000 ft, sign is omitted and always negative: e.g. "38" -> -38.0
	val, err := strconv.ParseFloat(tempStr, 64)
	if err == nil {
		if altFt >= 24000.0 {
			return -math.Abs(val)
		}
		return val
	}
	return 0.0
}

// InterpolateWindsAloft calculates wind vector and temperature at targetAltFt using trigonometric u/v decomposition.
func InterpolateWindsAloft(layers []models.WindsAloftLayer, targetAltFt float64) models.WindsAloftLayer {
	if len(layers) == 0 {
		return models.WindsAloftLayer{AltitudeMSLFt: targetAltFt}
	}

	// Sort layers by altitude ascending
	sorted := make([]models.WindsAloftLayer, len(layers))
	copy(sorted, layers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].AltitudeMSLFt < sorted[j].AltitudeMSLFt
	})

	// Bounds check
	if targetAltFt <= sorted[0].AltitudeMSLFt {
		res := sorted[0]
		res.AltitudeMSLFt = targetAltFt
		return res
	}
	if targetAltFt >= sorted[len(sorted)-1].AltitudeMSLFt {
		res := sorted[len(sorted)-1]
		res.AltitudeMSLFt = targetAltFt
		return res
	}

	// Find surrounding layers L1 and L2
	var l1, l2 models.WindsAloftLayer
	for i := 0; i < len(sorted)-1; i++ {
		if targetAltFt >= sorted[i].AltitudeMSLFt && targetAltFt <= sorted[i+1].AltitudeMSLFt {
			l1 = sorted[i]
			l2 = sorted[i+1]
			break
		}
	}

	altSpan := l2.AltitudeMSLFt - l1.AltitudeMSLFt
	if altSpan <= 0 {
		res := l1
		res.AltitudeMSLFt = targetAltFt
		return res
	}

	fraction := (targetAltFt - l1.AltitudeMSLFt) / altSpan

	// Convert winds to trigonometric u/v components (meteorological convention: direction is where wind blows FROM)
	// u = -V * sin(theta), v = -V * cos(theta)
	rad1 := l1.DirectionDeg * math.Pi / 180.0
	u1 := -l1.SpeedKts * math.Sin(rad1)
	v1 := -l1.SpeedKts * math.Cos(rad1)

	rad2 := l2.DirectionDeg * math.Pi / 180.0
	u2 := -l2.SpeedKts * math.Sin(rad2)
	v2 := -l2.SpeedKts * math.Cos(rad2)

	// Linearly interpolate u, v, and temp
	uInterp := u1 + fraction*(u2-u1)
	vInterp := v1 + fraction*(v2-v1)
	tempInterp := l1.TemperatureC + fraction*(l2.TemperatureC-l1.TemperatureC)

	// Reconstruct speed and direction
	speedInterp := math.Sqrt(uInterp*uInterp + vInterp*vInterp)

	var dirInterp float64
	if speedInterp > 0.1 {
		// atan2(-u, -v) returns the meteorological wind direction in radians
		radInterp := math.Atan2(-uInterp, -vInterp)
		degInterp := radInterp * 180.0 / math.Pi
		if degInterp < 0 {
			degInterp += 360.0
		}
		dirInterp = degInterp
	} else {
		dirInterp = 0.0
		speedInterp = 0.0
	}

	return models.WindsAloftLayer{
		AltitudeMSLFt: targetAltFt,
		DirectionDeg:  math.Round(dirInterp*10.0) / 10.0,
		SpeedKts:      math.Round(speedInterp*10.0) / 10.0,
		TemperatureC:  math.Round(tempInterp*10.0) / 10.0,
	}
}
