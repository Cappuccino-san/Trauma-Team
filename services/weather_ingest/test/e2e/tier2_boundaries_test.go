package e2e

import (
	"testing"

	"traumateam/weather_ingest/pkg/noaa"
)

// =============================================================================
// Tier 2: Boundary & Corner Cases
// =============================================================================

func TestTier2_CompoundVisibility_OneAndHalfSM(t *testing.T) {
	parser := &AuthoritativeMETARParser{}
	metar, err := parser.Parse(RawMETAR_KORD_CompoundVis)
	if err != nil {
		t.Fatalf("Parse compound vis failed: %v", err)
	}
	if metar.VisibilitySM != 1.5 {
		t.Errorf("expected compound visibility 1.5 SM, got %.2f", metar.VisibilitySM)
	}
}

func TestTier2_CompoundVisibility_TwoAndHalfSM(t *testing.T) {
	parser := &AuthoritativeMETARParser{}
	raw := "METAR KORD 051256Z 18015KT 2 1/2SM -SN BKN022 M02/M05 A2995"
	metar, err := parser.Parse(raw)
	if err != nil {
		t.Fatalf("Parse compound vis failed: %v", err)
	}
	if metar.VisibilitySM != 2.5 {
		t.Errorf("expected compound visibility 2.5 SM, got %.2f", metar.VisibilitySM)
	}
}

func TestTier2_MinimumVisibilityPrefix_MQuarterSM(t *testing.T) {
	parser := &AuthoritativeMETARParser{}
	metar, err := parser.Parse(RawMETAR_KSEA_MinVisLess)
	if err != nil {
		t.Fatalf("Parse min vis prefix failed: %v", err)
	}
	if metar.VisibilitySM != 0.25 {
		t.Errorf("expected visibility 0.25 SM, got %.2f", metar.VisibilitySM)
	}
}

func TestTier2_MaximumVisibilityPrefix_PSixSM(t *testing.T) {
	parser := &AuthoritativeMETARParser{}
	metar, err := parser.Parse(RawMETAR_KPHX_MaxVisP6SM)
	if err != nil {
		t.Fatalf("Parse max vis prefix failed: %v", err)
	}
	if metar.VisibilitySM != 6.0 {
		t.Errorf("expected visibility 6.0 SM, got %.1f", metar.VisibilitySM)
	}
}

func TestTier2_ClearSkyVariants_CLR_SKC_CAVOK(t *testing.T) {
	parser := &AuthoritativeMETARParser{}

	// Test CLR
	m1, err := parser.Parse("KAPA 050550Z 18010KT 10SM CLR 18/04 A2995")
	if err != nil || m1.CeilingAGLFt < 12000.0 {
		t.Errorf("expected unlimited ceiling for CLR, got %.0f", m1.CeilingAGLFt)
	}

	// Test SKC
	m2, err := parser.Parse("KAPA 050550Z 18010KT 10SM SKC 18/04 A2995")
	if err != nil || m2.CeilingAGLFt < 12000.0 {
		t.Errorf("expected unlimited ceiling for SKC, got %.0f", m2.CeilingAGLFt)
	}

	// Test CAVOK
	m3, err := parser.Parse(RawMETAR_CAVOK)
	if err != nil || m3.CeilingAGLFt < 12000.0 || m3.VisibilitySM < 6.0 {
		t.Errorf("expected unlimited ceiling and >6SM vis for CAVOK, got ceil=%.0f vis=%.1f", m3.CeilingAGLFt, m3.VisibilitySM)
	}
}

func TestTier2_VerticalObscurationCeiling_VV002(t *testing.T) {
	// Surface fog causing vertical visibility restriction
	metar, err := noaa.ParseMETAR(RawMETAR_KBJC_DenseFogVV002)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}
	// Under FAR 1.1, VV constitutes an indefinite ceiling at 200 ft AGL
	if metar.CeilingAGLFt != 200.0 {
		t.Errorf("expected vertical obscuration ceiling 200ft, got %.0f", metar.CeilingAGLFt)
	}
}

func TestTier2_VariableWind_LightVRB(t *testing.T) {
	metar, err := noaa.ParseMETAR(RawMETAR_VariableWind)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}
	if metar.WindSpeedKts != 3.0 {
		t.Errorf("expected wind speed 3kt, got %.0f", metar.WindSpeedKts)
	}
	if metar.WindDirDeg != 0.0 {
		t.Errorf("expected wind direction 0 for VRB, got %.0f", metar.WindDirDeg)
	}
}

func TestTier2_VariableWind_DirectionalRange(t *testing.T) {
	parser := &AuthoritativeMETARParser{}
	metar, err := parser.Parse(RawMETAR_WindShift)
	if err != nil {
		t.Fatalf("Parse wind shift failed: %v", err)
	}
	if metar.WindSpeedKts != 15.0 || metar.WindDirDeg != 180.0 {
		t.Errorf("expected base wind 180@15kt, got %.0f@%.0f", metar.WindDirDeg, metar.WindSpeedKts)
	}
}

func TestTier2_WindsAloft_SpeedOffsetRule(t *testing.T) {
	decoder := &AuthoritativeWindsAloftDecoder{}
	// Entry 751548 at 30,000 ft: direction code 75 > 50 -> speed >= 100kt
	vec, err := decoder.DecodeEntry("751548", 30000.0)
	if err != nil {
		t.Fatalf("DecodeEntry failed: %v", err)
	}
	// Direction: (75-50)*10 = 250 deg
	if vec.DirectionDeg != 250.0 {
		t.Errorf("expected direction 250 deg, got %.0f", vec.DirectionDeg)
	}
	// Speed: 15 + 100 = 115 kt
	if vec.SpeedKts != 115.0 {
		t.Errorf("expected speed 115 kt, got %.0f", vec.SpeedKts)
	}
	// Temp: -48 C (negative implicit above 24000ft)
	if vec.TemperatureC != -48.0 {
		t.Errorf("expected temp -48C, got %.1f", vec.TemperatureC)
	}
}

func TestTier2_WindsAloft_Calm9900(t *testing.T) {
	decoder := &AuthoritativeWindsAloftDecoder{}
	vec, err := decoder.DecodeEntry("9900", 9000.0)
	if err != nil {
		t.Fatalf("DecodeEntry failed: %v", err)
	}
	if !vec.IsCalm || vec.SpeedKts != 0.0 {
		t.Errorf("expected calm wind speed 0, got %.0f", vec.SpeedKts)
	}
}

func TestTier2_NegativeTemperatureAndDewpoint(t *testing.T) {
	metar, err := noaa.ParseMETAR("KCOS 050550Z 19008KT 10SM CLR M04/M12 A2992")
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}
	if metar.TemperatureC != -4.0 {
		t.Errorf("expected temp -4.0C, got %.1f", metar.TemperatureC)
	}
	if metar.DewpointC != -12.0 {
		t.Errorf("expected dewpoint -12.0C, got %.1f", metar.DewpointC)
	}
}

func TestTier2_MountainHighDensityAltitude(t *testing.T) {
	// Leadville KLXV: elevation 9,927 ft MSL, altimeter 29.70 inHg, temp 28 C (hot summer day)
	elevFt := 9927.0
	altimeterInHg := 29.70
	tempC := 28.0

	pressAlt := noaa.PressureAltitudeFt(elevFt, altimeterInHg)
	densAlt := noaa.DensityAltitudeFt(pressAlt, tempC)

	// Pressure altitude should be ~10,148 ft
	if pressAlt < 10000.0 || pressAlt > 10300.0 {
		t.Errorf("expected pressure altitude ~10150 ft, got %.1f", pressAlt)
	}
	// Density altitude should exceed 13,500 ft (extremely thin air, calculated: ~14,081 ft)
	if densAlt < 13500.0 || densAlt > 14500.0 {
		t.Errorf("expected density altitude ~14080 ft, got %.1f", densAlt)
	}
}

func TestTier2_CorruptedSensorSlashes(t *testing.T) {
	// Sensor slashes should be tolerated without crash
	parser := &AuthoritativeMETARParser{}
	metar, err := parser.Parse(RawMETAR_CorruptedSensor)
	if err != nil {
		t.Fatalf("Corrupted sensor METAR caused unhandled error: %v", err)
	}
	if metar.StationID != "KTPA" {
		t.Errorf("expected KTPA, got %s", metar.StationID)
	}
	if metar.VisibilitySM != 10.0 {
		t.Errorf("expected visibility 10 SM, got %.1f", metar.VisibilitySM)
	}
}
