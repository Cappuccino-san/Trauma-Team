package noaa

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"traumateam/weather_ingest/pkg/models"
)

var (
	convectiveHeaderRegx = regexp.MustCompile(`CONVECTIVE SIGMET\s+([0-9]+[E|C|W]?)`)
	standardSigmetRegx   = regexp.MustCompile(`SIGMET\s+([A-Z0-9\s]+?)\s+VALID\s+(?:UNTIL\s+)?(\d{2})(\d{4})Z?`)
	validUntilRegx       = regexp.MustCompile(`VALID UNTIL\s+(\d{2})(\d{2})Z`)
	flRangeRegx          = regexp.MustCompile(`(?:BTWN|BETWEEN)\s+FL(\d{3})\s+AND\s+FL(\d{3})`)
	topsRegx             = regexp.MustCompile(`TOPS\s+(?:ABV|TO)?\s*FL(\d{3})`)
	latLonCoordRegx      = regexp.MustCompile(`([-+]?\d{1,2}\.\d+)\s+([-+]?\d{1,3}\.\d+)`)
)

// Known navigation reference points for VOR / route decoding in Colorado/Wyoming/Nebraska HEMS region.
var knownNavPoints = map[string]models.GeoCoordinate{
	"CYS": {Latitude: 41.15, Longitude: -104.81},
	"SNY": {Latitude: 41.14, Longitude: -102.98},
	"AKO": {Latitude: 40.17, Longitude: -103.22},
	"DEN": {Latitude: 39.86, Longitude: -104.67},
	"PUB": {Latitude: 38.29, Longitude: -104.50},
	"LXV": {Latitude: 39.22, Longitude: -106.31},
	"ASE": {Latitude: 39.22, Longitude: -106.87},
	"APA": {Latitude: 39.57, Longitude: -104.85},
	"BFF": {Latitude: 41.87, Longitude: -103.60},
	"LAR": {Latitude: 41.31, Longitude: -105.67},
	"CHE": {Latitude: 41.08, Longitude: -104.99},
}

// ParseSIGMET decodes a raw NOAA AWC SIGMET bulletin.
func ParseSIGMET(raw string) (*models.WeatherHazard, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty SIGMET bulletin")
	}

	hazard := &models.WeatherHazard{
		RawText:    raw,
		Active:     true,
		Severity:   "SEVERE",
		HazardType: models.HazardTypeConvectiveSIGMET,
		Polygon:    make([]models.GeoCoordinate, 0),
		MinAltFt:   0,
		MaxAltFt:   45000.0,
		ValidFrom:  time.Now().UTC(),
		ValidUntil: time.Now().UTC().Add(2 * time.Hour),
	}

	// 1. Detect Advisory ID & Type
	if m := convectiveHeaderRegx.FindStringSubmatch(raw); m != nil {
		hazard.AdvisoryID = m[1]
		hazard.HazardType = models.HazardTypeConvectiveSIGMET
		hazard.Summary = "Active Convective SIGMET (Severe Thunderstorms / Line)"
	} else if m := standardSigmetRegx.FindStringSubmatch(raw); m != nil {
		hazard.AdvisoryID = m[1]
		hazard.Summary = fmt.Sprintf("Active SIGMET %s", m[1])
	} else {
		hazard.AdvisoryID = "SIGMET-1"
		hazard.Summary = "Active SIGMET Advisory"
	}

	// 2. Classify Hazard Type
	upper := strings.ToUpper(raw)
	if strings.Contains(upper, "CONVECTIVE") || strings.Contains(upper, "TS") || strings.Contains(upper, "THUNDERSTORM") {
		hazard.HazardType = models.HazardTypeConvectiveSIGMET
		hazard.Summary = "Active Convective SIGMET (Severe Thunderstorms / Line)"
	} else if strings.Contains(upper, "SEV ICE") || strings.Contains(upper, "SEVERE ICING") {
		hazard.HazardType = models.HazardTypeSevereIcing
		hazard.Summary = "Severe Icing Advisory"
	} else if strings.Contains(upper, "SEV TURB") || strings.Contains(upper, "SEVERE TURBULENCE") {
		hazard.HazardType = models.HazardTypeSevereTurbulence
		hazard.Summary = "Severe Turbulence Advisory"
	} else if strings.Contains(upper, "VOLCANIC ASH") || strings.Contains(upper, " VA ") {
		hazard.HazardType = models.HazardTypeVolcanicAsh
		hazard.Summary = "Volcanic Ash Hazard"
	} else if strings.Contains(upper, "DUST") || strings.Contains(upper, "SAND") {
		hazard.HazardType = models.HazardTypeDustSandstorm
		hazard.Summary = "Severe Dust/Sandstorm"
	}

	// 3. Parse Validity Time
	if m := validUntilRegx.FindStringSubmatch(raw); m != nil {
		hour, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		now := time.Now().UTC()
		hazard.ValidUntil = time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, time.UTC)
		if hazard.ValidUntil.Before(now) {
			// Crosses midnight
			hazard.ValidUntil = hazard.ValidUntil.Add(24 * time.Hour)
		}
	}

	// 4. Altitude Bounds
	if m := flRangeRegx.FindStringSubmatch(raw); m != nil {
		minFL, _ := strconv.ParseFloat(m[1], 64)
		maxFL, _ := strconv.ParseFloat(m[2], 64)
		hazard.MinAltFt = minFL * 100.0
		hazard.MaxAltFt = maxFL * 100.0
	} else if m := topsRegx.FindStringSubmatch(raw); m != nil {
		topFL, _ := strconv.ParseFloat(m[1], 64)
		hazard.MaxAltFt = topFL * 100.0
	}

	// 5. Polygon Coordinates Extraction
	// Check for explicit lat/lon coordinate pairs: e.g. "40.5 -104.2"
	coordMatches := latLonCoordRegx.FindAllStringSubmatch(raw, -1)
	for _, cm := range coordMatches {
		lat, err1 := strconv.ParseFloat(cm[1], 64)
		lon, err2 := strconv.ParseFloat(cm[2], 64)
		if err1 == nil && err2 == nil && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 {
			// Avoid altimeter settings or other numbers
			if lat > 20 && lat < 70 && lon < -50 && lon > -170 {
				hazard.Polygon = append(hazard.Polygon, models.GeoCoordinate{
					Latitude:  lat,
					Longitude: lon,
				})
			}
		}
	}

	// Check for FROM [VOR-VOR-VOR] lines (e.g. FROM 40ESE CYS-30W SNY-50SE AKO-40ESE CYS or FROM CHE TO LAR TO BFF TO CHE)
	fromIdx := strings.Index(raw, "FROM ")
	if fromIdx != -1 {
		fromSub := raw[fromIdx+5:]
		endOfLine := strings.IndexAny(fromSub, "\r\n")
		if endOfLine != -1 {
			fromSub = fromSub[:endOfLine]
		}
		points := parseFromVORLine(fromSub)
		if len(points) >= 3 {
			hazard.Polygon = points
		}
	}

	return hazard, nil
}

func parseFromVORLine(s string) []models.GeoCoordinate {
	s = strings.ReplaceAll(s, " TO ", "-")
	tokens := strings.Split(s, "-")
	coords := make([]models.GeoCoordinate, 0)

	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		// Extract base 3-letter VOR ID
		fields := strings.Fields(tok)
		for _, f := range fields {
			f = strings.Trim(f, "., ")
			if len(f) >= 3 {
				vorKey := f[len(f)-3:]
				if pt, ok := knownNavPoints[vorKey]; ok {
					coords = append(coords, pt)
					break
				}
			}
		}
	}
	return coords
}

// PointInPolygon tests if a coordinate is inside a polygon using the Ray-Casting algorithm.
func PointInPolygon(pt models.GeoCoordinate, poly []models.GeoCoordinate) bool {
	if len(poly) < 3 {
		return false
	}

	inside := false
	j := len(poly) - 1

	for i := 0; i < len(poly); i++ {
		xi, yi := poly[i].Longitude, poly[i].Latitude
		xj, yj := poly[j].Longitude, poly[j].Latitude

		intersect := ((yi > pt.Latitude) != (yj > pt.Latitude)) &&
			(pt.Longitude < (xj-xi)*(pt.Latitude-yi)/(yj-yi)+xi)

		if intersect {
			inside = !inside
		}
		j = i
	}

	return inside
}

// CorridorIntersectsPolygon checks if any point along a flight corridor intersects a hazard polygon or its buffer.
func CorridorIntersectsPolygon(corridorWaypoints []models.GeoCoordinate, poly []models.GeoCoordinate, bufferDeg float64) bool {
	if len(poly) < 3 {
		return false
	}

	// 1. Direct waypoint check
	for _, wp := range corridorWaypoints {
		if PointInPolygon(wp, poly) {
			return true
		}
		// Also check distance to each polygon edge within bufferDeg
		for i := 0; i < len(poly); i++ {
			p1 := poly[i]
			p2 := poly[(i+1)%len(poly)]
			if distPointToSegment(wp, p1, p2) <= bufferDeg {
				return true
			}
		}
	}

	// 2. Sample flight legs between corridor waypoints
	for k := 0; k < len(corridorWaypoints)-1; k++ {
		w1 := corridorWaypoints[k]
		w2 := corridorWaypoints[k+1]
		samples := 10
		for s := 1; s < samples; s++ {
			frac := float64(s) / float64(samples)
			samplePt := models.GeoCoordinate{
				Latitude:  w1.Latitude + frac*(w2.Latitude-w1.Latitude),
				Longitude: w1.Longitude + frac*(w2.Longitude-w1.Longitude),
			}
			if PointInPolygon(samplePt, poly) {
				return true
			}
		}
	}

	return false
}

func distPointToSegment(p, v, w models.GeoCoordinate) float64 {
	l2 := (w.Longitude-v.Longitude)*(w.Longitude-v.Longitude) + (w.Latitude-v.Latitude)*(w.Latitude-v.Latitude)
	if l2 == 0 {
		return math.Hypot(p.Longitude-v.Longitude, p.Latitude-v.Latitude)
	}
	t := ((p.Longitude-v.Longitude)*(w.Longitude-v.Longitude) + (p.Latitude-v.Latitude)*(w.Latitude-v.Latitude)) / l2
	t = math.Max(0, math.Min(1, t))
	projLon := v.Longitude + t*(w.Longitude-v.Longitude)
	projLat := v.Latitude + t*(w.Latitude-v.Latitude)
	return math.Hypot(p.Longitude-projLon, p.Latitude-projLat)
}
