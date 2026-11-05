package noaa

import (
	"context"
	"math"
	"testing"
	"time"
	"traumateam/weather_ingest/pkg/models"
)

func TestParseMETAR_StandardVFR(t *testing.T) {
	raw := "KAPA 050550Z 18010KT 10SM SCT050 18/04 A2995"
	m, err := ParseMETAR(raw)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}

	if m.StationID != "KAPA" {
		t.Errorf("expected station KAPA, got %s", m.StationID)
	}
	if m.WindDirDeg != 180.0 || m.WindSpeedKts != 10.0 {
		t.Errorf("wind parsed incorrectly: %.0f@%.0f", m.WindDirDeg, m.WindSpeedKts)
	}
	if m.VisibilitySM != 10.0 {
		t.Errorf("expected vis 10, got %.1f", m.VisibilitySM)
	}
	if m.TemperatureC != 18.0 || m.DewpointC != 4.0 {
		t.Errorf("temp/dewpoint incorrect: %.1f/%.1f", m.TemperatureC, m.DewpointC)
	}
	if math.Abs(m.AltimeterInHg-29.95) > 1e-4 {
		t.Errorf("expected altimeter 29.95, got %.2f", m.AltimeterInHg)
	}
	if m.FlightCategory != models.FlightCategoryVFR {
		t.Errorf("expected VFR, got %s", m.FlightCategory)
	}
	if m.HasCeiling {
		t.Errorf("SCT050 should not be a ceiling, but HasCeiling is true")
	}
}

func TestParseMETAR_CompoundFractionsAndSubzero(t *testing.T) {
	// KCOS with 1 1/2SM, snow, BKN015, subzero temps
	raw := "KCOS 050550Z 19008KT 1 1/2SM -SN BKN015 OVC025 M03/M05 A2992"
	m, err := ParseMETAR(raw)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}

	if m.StationID != "KCOS" {
		t.Errorf("expected KCOS, got %s", m.StationID)
	}
	if m.VisibilitySM != 1.5 {
		t.Errorf("expected visibility 1.5 SM, got %.2f", m.VisibilitySM)
	}
	if !m.HasCeiling || m.CeilingAGLFt != 1500.0 {
		t.Errorf("expected ceiling 1500 ft AGL, got %.0f", m.CeilingAGLFt)
	}
	if m.TemperatureC != -3.0 || m.DewpointC != -5.0 {
		t.Errorf("expected -3/-5 C, got %.1f/%.1f", m.TemperatureC, m.DewpointC)
	}
	if m.FlightCategory != models.FlightCategoryIFR {
		t.Errorf("expected IFR (vis 1.5 SM), got %s", m.FlightCategory)
	}
}

func TestParseMETAR_LessThanAndGreaterThanVisibility(t *testing.T) {
	// M1/4SM
	m1, err := ParseMETAR("KDEN 050600Z 00000KT M1/4SM FZFG VV001 M02/M02 A2990")
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}
	if m1.VisibilitySM != 0.25 || !m1.VisibilityLessThan {
		t.Errorf("expected 0.25 SM with less-than flag, got %.2f (less-than: %v)", m1.VisibilitySM, m1.VisibilityLessThan)
	}
	if !m1.HasCeiling || m1.CeilingAGLFt != 100.0 {
		t.Errorf("expected VV001 ceiling 100 ft, got %.0f", m1.CeilingAGLFt)
	}
	if m1.FlightCategory != models.FlightCategoryLIFR {
		t.Errorf("expected LIFR, got %s", m1.FlightCategory)
	}

	// P6SM and CLR
	m2, err := ParseMETAR("KPUB 050600Z 14006KT P6SM CLR 22/08 A3000")
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}
	if m2.VisibilitySM != 6.0 || !m2.VisibilityGreaterThan {
		t.Errorf("expected 6.0 SM with greater-than flag, got %.2f (greater-than: %v)", m2.VisibilitySM, m2.VisibilityGreaterThan)
	}
	if m2.HasCeiling {
		t.Errorf("CLR should have no ceiling, got HasCeiling=true")
	}
}

func TestParseMETAR_VerticalVisibilityCeiling(t *testing.T) {
	// Vertical visibility VV002 constitutes a 200 ft ceiling
	raw := "KBJC 050550Z VRB03KT 1/2SM FG VV002 01/01 A3005"
	m, err := ParseMETAR(raw)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}

	if !m.HasCeiling || m.CeilingAGLFt != 200.0 {
		t.Errorf("expected VV002 to be ceiling at 200 ft, got %.0f (hasCeiling=%v)", m.CeilingAGLFt, m.HasCeiling)
	}
	if !m.IsVariableWind || m.WindSpeedKts != 3.0 {
		t.Errorf("expected VRB03KT, got dir=%.0f spd=%.0f var=%v", m.WindDirDeg, m.WindSpeedKts, m.IsVariableWind)
	}
	if m.FlightCategory != models.FlightCategoryLIFR {
		t.Errorf("expected LIFR for 200 ft ceiling / 0.5 SM vis, got %s", m.FlightCategory)
	}
}

func TestParseMETAR_CloudLayerCeilingDistinction(t *testing.T) {
	// FEW and SCT are not ceilings; BKN is lowest ceiling
	raw := "KORD 051256Z 18015KT 10SM FEW008 SCT015 BKN035 OVC050 14/08 A2995"
	m, err := ParseMETAR(raw)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}

	if !m.HasCeiling || m.CeilingAGLFt != 3500.0 {
		t.Errorf("expected ceiling at BKN035 (3500 ft), got %.0f", m.CeilingAGLFt)
	}
	if len(m.CloudLayers) != 4 {
		t.Fatalf("expected 4 cloud layers, got %d", len(m.CloudLayers))
	}
	if m.CloudLayers[0].IsCeiling || m.CloudLayers[1].IsCeiling {
		t.Errorf("FEW and SCT should not be marked as ceilings")
	}
	if !m.CloudLayers[2].IsCeiling || !m.CloudLayers[3].IsCeiling {
		t.Errorf("BKN and OVC must be marked as ceilings")
	}
}

func TestParseMETAR_VariableWindRange(t *testing.T) {
	raw := "KDEN 050600Z 18015KT 140V220 10SM FEW040 20/05 A2998"
	m, err := ParseMETAR(raw)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}

	if !m.IsVariableWind || m.VarMinDeg != 140.0 || m.VarMaxDeg != 220.0 {
		t.Errorf("expected 140V220 variable wind, got var=%v min=%.0f max=%.0f",
			m.IsVariableWind, m.VarMinDeg, m.VarMaxDeg)
	}
}

func TestDensityAltitudeCalculation(t *testing.T) {
	// Leadville 9500 ft PA, 25 C -> DA ~ 12,900 ft
	da := DensityAltitudeFt(9500.0, 25.0)
	if da < 12500.0 || da > 13200.0 {
		t.Errorf("expected DA around 12900 ft, got %.0f", da)
	}
}

func TestParseTAF_CompleteLifecycle(t *testing.T) {
	taf, err := ParseTAF(FixtureTAF_KAPA)
	if err != nil {
		t.Fatalf("ParseTAF failed: %v", err)
	}

	if taf.StationID != "KAPA" {
		t.Errorf("expected KAPA, got %s", taf.StationID)
	}
	if len(taf.Periods) < 3 {
		t.Fatalf("expected at least 3 periods in TAF, got %d", len(taf.Periods))
	}

	// Base period: 18010KT P6SM SKC
	base := taf.Periods[0]
	if base.ChangeType != models.TAFChangeBase {
		t.Errorf("expected BASE change type, got %s", base.ChangeType)
	}
	if base.VisibilitySM != 6.0 || base.WindSpeedKts != 10.0 {
		t.Errorf("base forecast parsed incorrectly: vis=%.1f spd=%.0f", base.VisibilitySM, base.WindSpeedKts)
	}

	// Check conditions at ETA
	now := time.Now().UTC()
	// Test ETA in afternoon (e.g. 19:00 UTC) during TEMPO TSRA
	eta := time.Date(now.Year(), now.Month(), 5, 19, 0, 0, 0, time.UTC)
	prev, tempo := taf.GetConditionsAtETA(eta)

	if prev.ChangeType != models.TAFChangeFM {
		t.Errorf("expected prevailing FM group at 19:00 UTC, got %s", prev.ChangeType)
	}
	if len(tempo) == 0 {
		t.Errorf("expected active TEMPO period overlapping 19:00 UTC")
	} else {
		if tempo[0].VisibilitySM != 2.0 {
			t.Errorf("expected TEMPO vis 2.0 SM (TSRA), got %.1f", tempo[0].VisibilitySM)
		}
	}
}

func TestParseWindsAloft_DecodingAndInterpolation(t *testing.T) {
	forecasts, err := ParseWindsAloft(FixtureWindsAloft_FD)
	if err != nil {
		t.Fatalf("ParseWindsAloft failed: %v", err)
	}

	if len(forecasts) < 2 {
		t.Fatalf("expected at least 2 station forecasts, got %d", len(forecasts))
	}

	// Find CYS forecast which has high speed entry 731042 at 30,000 ft
	var cys *models.WindsAloftForecast
	for _, f := range forecasts {
		if f.StationID == "CYS" {
			fc := f
			cys = &fc
			break
		}
	}
	if cys == nil {
		t.Fatalf("CYS winds aloft not found")
	}

	// Layer at 30,000 ft: 731042 -> dir = (73-50)*10 = 230°, spd = 10+100 = 110 kts, temp = -42°C
	var fl300 *models.WindsAloftLayer
	for _, l := range cys.Layers {
		if l.AltitudeMSLFt == 30000.0 {
			fl300 = &l
			break
		}
	}
	if fl300 == nil {
		t.Fatalf("FL300 layer not found in CYS forecast")
	}
	if fl300.DirectionDeg != 230.0 || fl300.SpeedKts != 110.0 || fl300.TemperatureC != -42.0 {
		t.Errorf("expected 230@110kt -42C, got %.0f@%.0fkt %.0fC",
			fl300.DirectionDeg, fl300.SpeedKts, fl300.TemperatureC)
	}

	// Test trigonometric vector interpolation at 10,500 ft (between 9000 ft and 12000 ft)
	interp := InterpolateWindsAloft(cys.Layers, 10500.0)
	if interp.AltitudeMSLFt != 10500.0 {
		t.Errorf("expected alt 10500, got %.0f", interp.AltitudeMSLFt)
	}
	// At 9000: 2925-02 (290@25kt -2C). At 12000: 2930-08 (290@30kt -8C).
	// Midway at 10500: ~290° @ 27.5 kt, -5°C
	if math.Abs(interp.DirectionDeg-290.0) > 2.0 {
		t.Errorf("expected dir ~290°, got %.1f°", interp.DirectionDeg)
	}
	if math.Abs(interp.SpeedKts-27.5) > 1.5 {
		t.Errorf("expected speed ~27.5 kts, got %.1f kts", interp.SpeedKts)
	}
	if math.Abs(interp.TemperatureC-(-5.0)) > 1.0 {
		t.Errorf("expected temp ~-5°C, got %.1f°C", interp.TemperatureC)
	}
}

func TestParseSIGMET_ConvectiveAndIntersection(t *testing.T) {
	sigmet, err := ParseSIGMET(FixtureSIGMET_Convective)
	if err != nil {
		t.Fatalf("ParseSIGMET failed: %v", err)
	}

	if sigmet.HazardType != models.HazardTypeConvectiveSIGMET {
		t.Errorf("expected CONVECTIVE_SIGMET, got %s", sigmet.HazardType)
	}
	if sigmet.Severity != "SEVERE" {
		t.Errorf("expected SEVERE, got %s", sigmet.Severity)
	}
	if len(sigmet.Polygon) < 3 {
		t.Fatalf("expected polygon with at least 3 vertices, got %d", len(sigmet.Polygon))
	}

	// Point inside polygon (Eastern Colorado / Western Nebraska):
	// CYS (41.15, -104.81), SNY (41.14, -102.98), AKO (40.17, -103.22)
	// Centroid is approximately (40.8, -103.6)
	centerPt := models.GeoCoordinate{Latitude: 40.8, Longitude: -103.6}
	if !PointInPolygon(centerPt, sigmet.Polygon) {
		t.Errorf("expected (40.8, -103.6) to be inside convective polygon")
	}

	// Point clearly outside polygon (Leadville in the mountains: 39.22, -106.31)
	outsidePt := models.GeoCoordinate{Latitude: 39.22, Longitude: -106.31}
	if PointInPolygon(outsidePt, sigmet.Polygon) {
		t.Errorf("Leadville should not be inside eastern plains convective SIGMET")
	}

	// Corridor intersection: flight from Cheyenne to Akron intersects polygon
	corridor := []models.GeoCoordinate{
		{Latitude: 41.15, Longitude: -104.81}, // CYS
		{Latitude: 40.17, Longitude: -103.22}, // AKO
	}
	if !CorridorIntersectsPolygon(corridor, sigmet.Polygon, 0.1) {
		t.Errorf("expected corridor CYS-AKO to intersect convective polygon")
	}
}

func TestWeatherFeedClient_PreloadedFixtures(t *testing.T) {
	client := NewWeatherFeedClient()
	ctx := context.Background()

	metar, err := client.GetMETAR(ctx, "KLXV")
	if err != nil {
		t.Fatalf("GetMETAR KLXV failed: %v", err)
	}
	if metar.StationID != "KLXV" || metar.TemperatureC != 28.0 {
		t.Errorf("unexpected KLXV METAR: %v", metar)
	}

	taf, err := client.GetTAF(ctx, "KAPA")
	if err != nil {
		t.Fatalf("GetTAF KAPA failed: %v", err)
	}
	if taf.StationID != "KAPA" {
		t.Errorf("unexpected KAPA TAF: %v", taf)
	}

	winds, err := client.GetWindsAloft(ctx, "KDEN")
	if err != nil {
		t.Fatalf("GetWindsAloft KDEN failed: %v", err)
	}
	if len(winds.Layers) == 0 {
		t.Errorf("expected winds aloft layers for KDEN")
	}

	hazards, err := client.GetActiveHazards(ctx)
	if err != nil {
		t.Fatalf("GetActiveHazards failed: %v", err)
	}
	if len(hazards) < 2 {
		t.Errorf("expected at least 2 active hazards preloaded, got %d", len(hazards))
	}
}
