package e2e

import (
	"context"
	"testing"
	"time"

	"traumateam/weather_ingest/pkg/models"
	"traumateam/weather_ingest/pkg/noaa"
	"traumateam/weather_ingest/pkg/pb"
	"traumateam/weather_ingest/pkg/regulations"
)

// =============================================================================
// Tier 4: Real-World Application Scenarios
// =============================================================================

// TestTier4_Scenario1_DenverMetroNominalHEMS simulates a standard daytime patient transfer
// from Centennial Airport (KAPA) to Denver Health Medical Center (KDEN).
func TestTier4_Scenario1_DenverMetroNominalHEMS(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()
	h.ClearHazards()

	// 1. Setup mock observations
	h.SetStationObservation("KAPA", RawMETAR_KAPA_NominalVFR)
	h.SetStationObservation("KDEN", "KDEN 050550Z 17012KT 10SM BKN060 19/04 A2994")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 2. Query conditions via gRPC
	condResp, err := h.GRPCClient.GetCurrentConditions(ctx, &pb.StationConditionsRequest{
		StationId: "KAPA",
	})
	if err != nil {
		t.Fatalf("Failed to get KAPA weather: %v", err)
	}
	if condResp.Metar == nil || condResp.Metar.FlightCategory != pb.FlightRuleCategory_CATEGORY_VFR {
		t.Errorf("expected VFR at KAPA, got %v", condResp.Metar)
	}

	// 3. Evaluate FAA Part 135.609 Table 1 Minimums for Day Local Non-Mountainous
	evalResp, err := h.GRPCClient.EvaluateFlightMinimums(ctx, &pb.FlightMinimumsEvaluationRequest{
		DepartureStation:   "KAPA",
		DestinationStation: "KDEN",
		IsDay:              true,
		IsMountainous:      false,
		IsLocal:            true,
		Equipment: &pb.EquipmentFlags{
			NvgEquipped: false, // Standard unaided
		},
	})
	if err != nil {
		t.Fatalf("Failed to evaluate KAPA minimums: %v", err)
	}

	// Rule: Min Ceiling 800 ft, Min Vis 2 SM
	if !evalResp.IsCompliant {
		t.Errorf("expected compliant flight release, got non-compliant")
	}
	if evalResp.Verdict != pb.DispatchVerdict_VERDICT_APPROVED {
		t.Errorf("expected APPROVED verdict, got %v", evalResp.Verdict)
	}
}

// TestTier4_Scenario2_LeadvilleHighDAMountainRescue simulates high-altitude alpine dispatch
// from Centennial (KAPA) to Leadville Lake County Airport (KLXV) at 9,927 ft MSL.
func TestTier4_Scenario2_LeadvilleHighDAMountainRescue(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()
	h.ClearHazards()

	// KLXV: Elevation 9,927 ft MSL, ambient temperature 28°C, altimeter 29.70 inHg
	h.SetStationObservation("KLXV", RawMETAR_KLXV_LeadvilleHighDA)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 1. Evaluate Regulatory Flight Minimums: Mountainous Cross-Country Day
	evalResp, err := h.GRPCClient.EvaluateFlightMinimums(ctx, &pb.FlightMinimumsEvaluationRequest{
		DepartureStation:   "KAPA",
		DestinationStation: "KLXV",
		IsDay:              true,
		IsMountainous:      true,
		IsLocal:            false,
		Equipment: &pb.EquipmentFlags{
			NvgEquipped:   true,
			HtawsEquipped: true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to evaluate KLXV minimums: %v", err)
	}

	// Rule: Min Ceiling 1000 ft, Min Vis 3 SM
	if !evalResp.IsCompliant || evalResp.Verdict != pb.DispatchVerdict_VERDICT_APPROVED {
		t.Errorf("expected weather APPROVED for Leadville, got verdict %v", evalResp.Verdict)
	}

	// 2. Calculate Physical Density Altitude (Aeromedical Performance Margin)
	pressAlt := noaa.PressureAltitudeFt(9927.0, 29.70)
	densAlt := noaa.DensityAltitudeFt(pressAlt, 28.0)

	// In hot summer conditions, Leadville DA exceeds 13,500 ft MSL
	if densAlt < 13400.0 {
		t.Errorf("expected extreme density altitude > 13400 ft MSL, got %.1f ft", densAlt)
	}

	if densAlt > 12000.0 {
		t.Logf("[Scenario 2] Extreme Mountain Density Altitude identified: %.1f ft MSL", densAlt)
	}
}

// TestTier4_Scenario3_AspenLowCeilingSnowDiversion simulates severe mountain blizzard at
// Aspen Pitkin County Airport (KASE), forcing a diversion or IFR alternate dispatch.
func TestTier4_Scenario3_AspenLowCeilingSnowDiversion(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	// KASE: Aspen ceiling 900 ft AGL, visibility 2 SM in snow
	h.SetStationObservation("KASE", RawMETAR_KASE_AspenLowIFR)
	// KGJT: Grand Junction alternate airport (10SM CLR)
	h.SetStationObservation("KGJT", RawMETAR_KGJT_AlternateVFR)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 1. Primary Destination Check: Mountainous Night Cross-Country with NVG
	evalResp, err := h.GRPCClient.EvaluateFlightMinimums(ctx, &pb.FlightMinimumsEvaluationRequest{
		DepartureStation:   "KAPA",
		DestinationStation: "KASE",
		IsDay:              false,
		IsMountainous:      true,
		IsLocal:            false,
		Equipment: &pb.EquipmentFlags{
			NvgEquipped:   true,
			HtawsEquipped: true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to evaluate KASE minimums: %v", err)
	}

	// Required: 1,000 ft ceiling / 3 SM visibility
	// Observed: 900 ft ceiling / 2.0 SM visibility
	// Must result in non-compliant
	if evalResp.IsCompliant {
		t.Errorf("expected non-compliant for Aspen below-minimums snow, got verdict %v", evalResp.Verdict)
	}

	// 2. Diversion Evaluation: Grand Junction (KGJT) Alternate
	kgjtMetar, err := noaa.ParseMETAR(RawMETAR_KGJT_AlternateVFR)
	if err != nil {
		t.Fatalf("Parse KGJT failed: %v", err)
	}
	kgjtReport := regulations.EvaluateMinimums(kgjtMetar, false, false, false, true)

	if !kgjtReport.IsCompliant || kgjtReport.Verdict != "GO" {
		t.Errorf("expected alternate KGJT to be compliant GO, got %s", kgjtReport.Verdict)
	}

	// 3. Full Corridor Evaluator check incorporating § 135.611 IFR Transition
	evaluator := &AuthoritativeCorridorEvaluator{}
	corridorReq := CorridorEvaluationRequest{
		CorridorStations: []string{"KASE", "KGJT"},
		StationMETARs: map[string]*models.METAR{
			"KASE": {StationID: "KASE", CeilingAGLFt: 900.0, VisibilitySM: 2.0},
			"KGJT": kgjtMetar,
		},
		IsDay:                 false,
		IsMountainous:         true,
		IsLocal:               false,
		NVGOrHTAWSEquipped:    true,
		IsIFRCertified:        true,
		HasIFRAlternate:       true, // Alternate KGJT available!
		HasSevereWeatherRadar: true,
	}

	corridorRes := evaluator.Evaluate(corridorReq)
	// Because an alternate (KGJT) and radar are present, mission transitions to IFR under § 135.611
	if corridorRes.Verdict != VerdictIFRRequired {
		t.Errorf("expected IFR_REQUIRED with KGJT alternate, got %s", corridorRes.Verdict)
	}
}
