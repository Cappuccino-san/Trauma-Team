package kinematics

import (
	"math"
	"testing"

	"traumateam/weather_ingest/pkg/noaa"
)

func TestFFI_ABIVersion(t *testing.T) {
	ver, err := ABIVersion()
	if err != nil {
		t.Fatalf("ABIVersion failed: %v", err)
	}
	if ver != 100 {
		t.Errorf("expected ABI version 100, got %d", ver)
	}
}

func TestFFI_ComputeAtmosphere_Leadville(t *testing.T) {
	pa, da, err := ComputeAtmosphere(9800.0, 29.70, 28.0)
	if err != nil {
		t.Fatalf("ComputeAtmosphere failed: %v", err)
	}
	if pa < 9950.0 || pa > 10100.0 {
		t.Errorf("unexpected pressure altitude for Leadville: %.2f ft", pa)
	}
	if da < 13400.0 || da > 13800.0 {
		t.Errorf("unexpected density altitude for Leadville: %.2f ft", da)
	}
}

func TestFFI_EvaluateMassBalance_AnalyticalParity(t *testing.T) {
	// Exact analytical hand-calculation test matching Python & Rust test_mass_balance.rs:
	// H145 Empty = 2140 kg, Pilot(85), Copilot(80), Nurse(70), Medic(85), MedRack(60), Fuel(500)
	// Expected GW = 3020.0 kg
	// Expected CG_long = 11541500.0 / 3020.0 = 3821.6887417 mm
	// Expected CG_lat = 20200.0 / 3020.0 = 6.6887417 mm
	payloads := []PayloadStationInput{
		{StationID: StationPilotRH, WeightKg: 85.0},
		{StationID: StationCopilotLH, WeightKg: 80.0},
		{StationID: StationCrewAttendantAftLH, WeightKg: 70.0},
		{StationID: StationCrewAttendantAftRH, WeightKg: 85.0},
		{StationID: StationMedicalCabinetRack, WeightKg: 60.0},
	}

	mb, err := EvaluateMassBalance(AirframeAirbusH145, 500.0, payloads)
	if err != nil {
		t.Fatalf("EvaluateMassBalance failed: %v", err)
	}

	if !mb.IsOverallCompliant {
		t.Errorf("expected overall compliant, got false (limiting factor: %s)", mb.LimitingFactor)
	}
	if mb.LimitingFactor != "NONE" {
		t.Errorf("expected NONE limiting factor, got %s", mb.LimitingFactor)
	}

	expectedGW := 3020.0
	expectedLong := 11541500.0 / 3020.0
	expectedLat := 20200.0 / 3020.0

	if math.Abs(mb.GrossWeightKg-expectedGW) > 1e-5 {
		t.Errorf("expected GW %.4f, got %.4f", expectedGW, mb.GrossWeightKg)
	}
	if math.Abs(mb.CGLongMM-expectedLong) > 1e-3 {
		t.Errorf("expected CG_long %.4f, got %.4f", expectedLong, mb.CGLongMM)
	}
	if math.Abs(mb.CGLatMM-expectedLat) > 1e-3 {
		t.Errorf("expected CG_lat %.4f, got %.4f", expectedLat, mb.CGLatMM)
	}
}

func TestFFI_EvaluateMassBalance_LateralCGExceedance(t *testing.T) {
	payloads := []PayloadStationInput{
		{StationID: StationPilotRH, WeightKg: 110.0},
		{StationID: StationCrewAttendantAftRH, WeightKg: 110.0},
		{StationID: StationMedicalCabinetRack, WeightKg: 100.0},
		{StationID: StationRescueHoistRH, WeightKg: 220.0},
	}

	mb, err := EvaluateMassBalance(AirframeAirbusH145, 100.0, payloads)
	if err != nil {
		t.Fatalf("EvaluateMassBalance failed: %v", err)
	}

	if mb.IsOverallCompliant || mb.IsLatCGCompliant {
		t.Errorf("expected lateral CG violation, but got compliant")
	}
	if mb.LimitingFactor != "LATERAL_CG_RIGHT_EXCEEDED" {
		t.Errorf("expected LATERAL_CG_RIGHT_EXCEEDED, got %s", mb.LimitingFactor)
	}
}

func TestFFI_EvaluateMissionKinematics_WithNOAAMETAR(t *testing.T) {
	// 1. Parse raw NOAA KAPA METAR and evaluate nominal mission
	kapaMetar, err := noaa.ParseMETAR(noaa.FixtureMETAR_KAPA)
	if err != nil {
		t.Fatalf("ParseMETAR KAPA failed: %v", err)
	}

	stdCrew := []PayloadStationInput{
		{StationID: StationPilotRH, WeightKg: 88.0},
		{StationID: StationCrewAttendantAftLH, WeightKg: 80.0},
		{StationID: StationPrimaryPatientLitter, WeightKg: 92.0},
	}

	kapaEval, err := EvaluateMissionKinematics(MissionKinematicsRequest{
		Airframe:           AirframeAirbusH145,
		StationElevationFt: 5885.0,
		Weather:            kapaMetar,
		TrueCourseDeg:      270.0,
		LegDistanceNM:      45.0,
		FuelWeightKg:       450.0,
		Payloads:           stdCrew,
	})
	if err != nil {
		t.Fatalf("EvaluateMissionKinematics KAPA failed: %v", err)
	}

	if !kapaEval.IsFeasible || kapaEval.Verdict != "GO" {
		t.Errorf("expected KAPA nominal mission GO, got %s (%s)", kapaEval.Verdict, kapaEval.LimitingFactor)
	}
	if kapaEval.CruiseTASKts < 120.0 || kapaEval.GroundspeedKts < 110.0 {
		t.Errorf("unexpected TAS/GS: TAS=%.1f, GS=%.1f", kapaEval.CruiseTASKts, kapaEval.GroundspeedKts)
	}

	// 2. Parse raw NOAA KLXV (Leadville 9,927 ft, 28C OAT) METAR -> High DA OGE Hover Deficit
	klxvMetar, err := noaa.ParseMETAR(noaa.FixtureMETAR_KLXV)
	if err != nil {
		t.Fatalf("ParseMETAR KLXV failed: %v", err)
	}

	klxvEval, err := EvaluateMissionKinematics(MissionKinematicsRequest{
		Airframe:           AirframeAirbusH145,
		StationElevationFt: 9800.0,
		Weather:            klxvMetar,
		TrueCourseDeg:      260.0,
		LegDistanceNM:      55.0,
		FuelWeightKg:       600.0,
		Payloads:           stdCrew,
	})
	if err != nil {
		t.Fatalf("EvaluateMissionKinematics KLXV failed: %v", err)
	}

	if klxvEval.IsFeasible || klxvEval.Verdict != "NO-GO" {
		t.Errorf("expected KLXV high-DA mission NO-GO, got %s", klxvEval.Verdict)
	}
	if klxvEval.LimitingFactor != "OGE_HOVER_SCENE_DEFICIT" {
		t.Errorf("expected OGE_HOVER_SCENE_DEFICIT, got %s", klxvEval.LimitingFactor)
	}
	if klxvEval.OGEMarginKg >= 0 {
		t.Errorf("expected negative OGE hover margin, got %.1f kg", klxvEval.OGEMarginKg)
	}

	// 3. Evaluate Bell 429 GlobalRanger on KDEN METAR
	kdenMetar, err := noaa.ParseMETAR(noaa.FixtureMETAR_KDEN)
	if err != nil {
		t.Fatalf("ParseMETAR KDEN failed: %v", err)
	}
	b429Crew := []PayloadStationInput{
		{StationID: StationPilotRH, WeightKg: 88.0},
		{StationID: StationCrewAttendantLH, WeightKg: 80.0},
		{StationID: StationPrimaryPatientLitter, WeightKg: 92.0},
	}
	b429Eval, err := EvaluateMissionKinematics(MissionKinematicsRequest{
		Airframe:           AirframeBell429,
		StationElevationFt: 5434.0,
		Weather:            kdenMetar,
		TrueCourseDeg:      180.0,
		LegDistanceNM:      40.0,
		FuelWeightKg:       250.0,
		Payloads:           b429Crew,
	})
	if err != nil {
		t.Fatalf("EvaluateMissionKinematics Bell 429 failed: %v", err)
	}
	if !b429Eval.IsFeasible || b429Eval.Verdict != "GO" {
		t.Errorf("expected Bell 429 KDEN mission GO, got %s (%s)", b429Eval.Verdict, b429Eval.LimitingFactor)
	}
}
