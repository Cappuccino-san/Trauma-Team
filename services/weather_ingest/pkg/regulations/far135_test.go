package regulations

import (
	"testing"
	"traumateam/weather_ingest/pkg/models"
)

func TestFAR135609_CompleteMatrix(t *testing.T) {
	// Validates every legal cell in 14 CFR § 135.609 Table 1 (matching Python test_regulations.py)

	// 1. Non-mountainous local
	// Day without NVG/HTAWS: 800-2; with NVG/HTAWS: 500-2
	ctx1 := models.RegulatoryContext{IsDay: true, IsMountainous: false, IsLocalFlyingArea: true}
	r1 := GetFAR135609Minimums(ctx1)
	if r1.MinCeilingFt != 800.0 || r1.MinVisibilitySM != 2.0 {
		t.Errorf("expected 800-2, got %.0f-%.1f", r1.MinCeilingFt, r1.MinVisibilitySM)
	}

	ctx2 := models.RegulatoryContext{IsDay: true, IsMountainous: false, IsLocalFlyingArea: true, Equipment: models.EquipmentFlags{NVGEquipped: true}}
	r2 := GetFAR135609Minimums(ctx2)
	if r2.MinCeilingFt != 500.0 || r2.MinVisibilitySM != 2.0 {
		t.Errorf("expected 500-2, got %.0f-%.1f", r2.MinCeilingFt, r2.MinVisibilitySM)
	}

	// Night local: 1000-3 both standard and NVG
	ctx3 := models.RegulatoryContext{IsDay: false, IsMountainous: false, IsLocalFlyingArea: true}
	r3 := GetFAR135609Minimums(ctx3)
	if r3.MinCeilingFt != 1000.0 || r3.MinVisibilitySM != 3.0 {
		t.Errorf("expected 1000-3, got %.0f-%.1f", r3.MinCeilingFt, r3.MinVisibilitySM)
	}
	ctx4 := models.RegulatoryContext{IsDay: false, IsMountainous: false, IsLocalFlyingArea: true, Equipment: models.EquipmentFlags{NVGEquipped: true}}
	r4 := GetFAR135609Minimums(ctx4)
	if r4.MinCeilingFt != 1000.0 || r4.MinVisibilitySM != 3.0 {
		t.Errorf("expected 1000-3, got %.0f-%.1f", r4.MinCeilingFt, r4.MinVisibilitySM)
	}

	// 2. Non-mountainous cross-country
	// Day without NVG: 800-3; with NVG: 600-2
	ctx5 := models.RegulatoryContext{IsDay: true, IsMountainous: false, IsLocalFlyingArea: false}
	r5 := GetFAR135609Minimums(ctx5)
	if r5.MinCeilingFt != 800.0 || r5.MinVisibilitySM != 3.0 {
		t.Errorf("expected 800-3, got %.0f-%.1f", r5.MinCeilingFt, r5.MinVisibilitySM)
	}
	ctx6 := models.RegulatoryContext{IsDay: true, IsMountainous: false, IsLocalFlyingArea: false, Equipment: models.EquipmentFlags{NVGEquipped: true}}
	r6 := GetFAR135609Minimums(ctx6)
	if r6.MinCeilingFt != 600.0 || r6.MinVisibilitySM != 2.0 {
		t.Errorf("expected 600-2, got %.0f-%.1f", r6.MinCeilingFt, r6.MinVisibilitySM)
	}

	// Night without NVG: 1000-3; with NVG: 800-3
	ctx7 := models.RegulatoryContext{IsDay: false, IsMountainous: false, IsLocalFlyingArea: false}
	r7 := GetFAR135609Minimums(ctx7)
	if r7.MinCeilingFt != 1000.0 || r7.MinVisibilitySM != 3.0 {
		t.Errorf("expected 1000-3, got %.0f-%.1f", r7.MinCeilingFt, r7.MinVisibilitySM)
	}
	ctx8 := models.RegulatoryContext{IsDay: false, IsMountainous: false, IsLocalFlyingArea: false, Equipment: models.EquipmentFlags{NVGEquipped: true}}
	r8 := GetFAR135609Minimums(ctx8)
	if r8.MinCeilingFt != 800.0 || r8.MinVisibilitySM != 3.0 {
		t.Errorf("expected 800-3, got %.0f-%.1f", r8.MinCeilingFt, r8.MinVisibilitySM)
	}

	// 3. Mountainous local
	// Day: 800-3
	ctx9 := models.RegulatoryContext{IsDay: true, IsMountainous: true, IsLocalFlyingArea: true}
	r9 := GetFAR135609Minimums(ctx9)
	if r9.MinCeilingFt != 800.0 || r9.MinVisibilitySM != 3.0 {
		t.Errorf("expected 800-3, got %.0f-%.1f", r9.MinCeilingFt, r9.MinVisibilitySM)
	}
	// Night local: 1500-3 without NVG; 1000-3 with NVG
	ctx10 := models.RegulatoryContext{IsDay: false, IsMountainous: true, IsLocalFlyingArea: true}
	r10 := GetFAR135609Minimums(ctx10)
	if r10.MinCeilingFt != 1500.0 || r10.MinVisibilitySM != 3.0 {
		t.Errorf("expected 1500-3, got %.0f-%.1f", r10.MinCeilingFt, r10.MinVisibilitySM)
	}
	ctx11 := models.RegulatoryContext{IsDay: false, IsMountainous: true, IsLocalFlyingArea: true, Equipment: models.EquipmentFlags{NVGEquipped: true}}
	r11 := GetFAR135609Minimums(ctx11)
	if r11.MinCeilingFt != 1000.0 || r11.MinVisibilitySM != 3.0 {
		t.Errorf("expected 1000-3, got %.0f-%.1f", r11.MinCeilingFt, r11.MinVisibilitySM)
	}

	// 4. Mountainous cross-country
	// Day: 1000-3
	ctx12 := models.RegulatoryContext{IsDay: true, IsMountainous: true, IsLocalFlyingArea: false}
	r12 := GetFAR135609Minimums(ctx12)
	if r12.MinCeilingFt != 1000.0 || r12.MinVisibilitySM != 3.0 {
		t.Errorf("expected 1000-3, got %.0f-%.1f", r12.MinCeilingFt, r12.MinVisibilitySM)
	}
	// Night XC: 1500-5 without NVG; 1000-3 with NVG
	ctx13 := models.RegulatoryContext{IsDay: false, IsMountainous: true, IsLocalFlyingArea: false}
	r13 := GetFAR135609Minimums(ctx13)
	if r13.MinCeilingFt != 1500.0 || r13.MinVisibilitySM != 5.0 {
		t.Errorf("expected 1500-5, got %.0f-%.1f", r13.MinCeilingFt, r13.MinVisibilitySM)
	}
	ctx14 := models.RegulatoryContext{IsDay: false, IsMountainous: true, IsLocalFlyingArea: false, Equipment: models.EquipmentFlags{NVGEquipped: true}}
	r14 := GetFAR135609Minimums(ctx14)
	if r14.MinCeilingFt != 1000.0 || r14.MinVisibilitySM != 3.0 {
		t.Errorf("expected 1000-3, got %.0f-%.1f", r14.MinCeilingFt, r14.MinVisibilitySM)
	}

	// 5. Verify AutopilotEquipped qualifies for Night Table 1 Column 2 credit (14 CFR § 135.609)
	ctxAutopilotNightXC := models.RegulatoryContext{IsDay: false, IsMountainous: true, IsLocalFlyingArea: false, Equipment: models.EquipmentFlags{AutopilotEquipped: true}}
	r15 := GetFAR135609Minimums(ctxAutopilotNightXC)
	if r15.MinCeilingFt != 1000.0 || r15.MinVisibilitySM != 3.0 || !r15.EquippedNVGHTAWS {
		t.Errorf("expected 1000-3 with AutopilotEquipped at Night Mountainous XC, got %.0f-%.1f (equipped=%v)", r15.MinCeilingFt, r15.MinVisibilitySM, r15.EquippedNVGHTAWS)
	}
}

func TestCorridorCompliance_ApprovedAndMarginal(t *testing.T) {
	st1 := models.METAR{
		StationID:    "KAPA",
		CeilingAGLFt: 3000.0,
		HasCeiling:   true,
		VisibilitySM: 10.0,
		WindGustKts:  10.0,
	}
	st2 := models.METAR{
		StationID:    "KLXV",
		CeilingAGLFt: 4000.0,
		HasCeiling:   true,
		VisibilitySM: 10.0,
		WindGustKts:  12.0,
	}

	ctx := models.RegulatoryContext{
		IsDay:             true,
		IsMountainous:     true,
		IsLocalFlyingArea: false, // Req: 1000 ft, 3 SM
	}

	rep := EvaluateCorridorCompliance([]models.METAR{st1, st2}, nil, ctx)
	if !rep.IsCompliant || rep.Verdict != models.VerdictApproved {
		t.Errorf("expected APPROVED, got %s (compliant=%v)", rep.Verdict, rep.IsCompliant)
	}

	// Test Marginal condition: ceiling within 200 ft margin (e.g. 1100 ft vs 1000 ft req)
	stMarginal := models.METAR{
		StationID:    "KASE",
		CeilingAGLFt: 1100.0,
		HasCeiling:   true,
		VisibilitySM: 5.0,
	}
	repMarginal := EvaluateCorridorCompliance([]models.METAR{st1, stMarginal}, nil, ctx)
	if !repMarginal.IsCompliant || repMarginal.Verdict != models.VerdictMarginal {
		t.Errorf("expected MARGINAL for 100 ft ceiling margin, got %s", repMarginal.Verdict)
	}
}

func TestCorridorCompliance_ObstacleClearance(t *testing.T) {
	st := models.METAR{
		StationID:    "KAPA",
		CeilingAGLFt: 5000.0,
		HasCeiling:   true,
		VisibilitySM: 10.0,
	}

	// Day: Obstacle is 10,000 ft MSL, planned cruise is 10,200 ft MSL (clearance = 200 ft < 300 ft req)
	ctxDay := models.RegulatoryContext{
		IsDay:               true,
		IsMountainous:       true,
		PlannedCruiseAltMSL: 10200.0,
		HighestObstacleMSL:  10000.0,
	}
	repDay := EvaluateCorridorCompliance([]models.METAR{st}, nil, ctxDay)
	if repDay.Verdict != models.VerdictBelowMinimums {
		t.Errorf("expected BELOW_MINIMUMS for 200 ft obstacle clearance in day, got %s", repDay.Verdict)
	}
	if repDay.LimitingFactor != models.LimitFAR135615ObstacleClearance {
		t.Errorf("expected obstacle limiting factor, got %s", repDay.LimitingFactor)
	}

	// Night: Obstacle is 10,000 ft MSL, planned cruise is 10,400 ft MSL (clearance = 400 ft < 500 ft req)
	ctxNight := models.RegulatoryContext{
		IsDay:               false,
		IsMountainous:       true,
		PlannedCruiseAltMSL: 10400.0,
		HighestObstacleMSL:  10000.0,
	}
	repNight := EvaluateCorridorCompliance([]models.METAR{st}, nil, ctxNight)
	if repNight.Verdict != models.VerdictBelowMinimums {
		t.Errorf("expected BELOW_MINIMUMS for 400 ft obstacle clearance at night, got %s", repNight.Verdict)
	}
}

func TestCorridorCompliance_IFRTransition(t *testing.T) {
	// Weather is below VFR minimums: Aspen KASE ceiling 600 ft (req 1000 ft)
	st := models.METAR{
		StationID:    "KASE",
		CeilingAGLFt: 600.0,
		HasCeiling:   true,
		VisibilitySM: 2.0,
	}

	// Case 1: Not IFR certified -> BELOW_MINIMUMS
	ctxVFR := models.RegulatoryContext{
		IsDay:          true,
		IsMountainous:  true,
		IsIFRCertified: false,
	}
	rep1 := EvaluateCorridorCompliance([]models.METAR{st}, nil, ctxVFR)
	if rep1.Verdict != models.VerdictBelowMinimums {
		t.Errorf("expected BELOW_MINIMUMS, got %s", rep1.Verdict)
	}

	// Case 2: IFR certified, has alternate, has weather radar -> IFR_REQUIRED
	ctxIFR := models.RegulatoryContext{
		IsDay:                 true,
		IsMountainous:         true,
		IsIFRCertified:        true,
		HasIFRAlternate:       true,
		HasSevereWeatherRadar: true,
	}
	rep2 := EvaluateCorridorCompliance([]models.METAR{st}, nil, ctxIFR)
	if rep2.Verdict != models.VerdictIFRRequired {
		t.Errorf("expected IFR_REQUIRED under § 135.611, got %s", rep2.Verdict)
	}
	if !rep2.IsCompliant {
		t.Errorf("expected compliant for IFR transition")
	}
}

func TestCorridorCompliance_SIGMETDisqualification(t *testing.T) {
	st := models.METAR{
		StationID:    "KAPA",
		CeilingAGLFt: 10000.0,
		HasCeiling:   true,
		VisibilitySM: 10.0,
	}

	// Active Convective SIGMET overrides perfect surface weather
	hazard := models.WeatherHazard{
		HazardID:   "15E",
		HazardType: models.HazardTypeConvectiveSIGMET,
		Severity:   "SEVERE",
		Active:     true,
		Summary:    "Severe Thunderstorm Line",
	}

	ctx := models.RegulatoryContext{IsDay: true}
	rep := EvaluateCorridorCompliance([]models.METAR{st}, []models.WeatherHazard{hazard}, ctx)
	if rep.Verdict != models.VerdictDisqualified {
		t.Errorf("expected DISQUALIFIED for active convective SIGMET, got %s", rep.Verdict)
	}
	if rep.LimitingFactor != models.LimitConvectiveSIGMET {
		t.Errorf("expected ACTIVE_CONVECTIVE_SIGMET, got %s", rep.LimitingFactor)
	}
}

func TestFAR135615_LegacyHelperCompatibility(t *testing.T) {
	obs := &models.METAR{
		StationID:    "KAPA",
		CeilingAGLFt: 3000.0,
		VisibilitySM: 10.0,
	}
	rep := EvaluateMinimums(obs, true, false, true, true)
	if !rep.IsCompliant || rep.Verdict != "GO" {
		t.Errorf("expected GO, got %s", rep.Verdict)
	}

	obsLow := &models.METAR{
		StationID:    "KASE",
		CeilingAGLFt: 900.0,
		VisibilitySM: 4.0,
	}
	repLow := EvaluateMinimums(obsLow, false, true, false, true)
	if repLow.IsCompliant || repLow.Verdict != "NO-GO" {
		t.Errorf("expected NO-GO, got %s", repLow.Verdict)
	}
}
