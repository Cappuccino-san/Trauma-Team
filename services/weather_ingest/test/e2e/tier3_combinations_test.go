package e2e

import (
	"context"
	"testing"
	"time"

	"traumateam/weather_ingest/pkg/models"
	"traumateam/weather_ingest/pkg/pb"
)

// =============================================================================
// Tier 3: Cross-Feature Combinations
// =============================================================================

func TestTier3_CorridorWeather_ConvectiveSIGMET_Override(t *testing.T) {
	evaluator := &AuthoritativeCorridorEvaluator{}

	// Surface METARs are clear VFR
	req := CorridorEvaluationRequest{
		CorridorStations: []string{"KAPA", "KDEN"},
		StationMETARs: map[string]*models.METAR{
			"KAPA": {StationID: "KAPA", CeilingAGLFt: 10000.0, VisibilitySM: 10.0},
			"KDEN": {StationID: "KDEN", CeilingAGLFt: 12000.0, VisibilitySM: 10.0},
		},
		ActiveHazards: []ParsedSIGMET{
			{
				AdvisoryID:    "15E",
				HazardType:    "CONVECTIVE",
				Severity:      "EXTREME",
				ValidUntilUTC: time.Now().Add(1 * time.Hour),
				Summary:       "LINE SEV TS WITH 2 IN HAIL AND 60KT GUSTS",
				Active:        true,
			},
		},
		IsDay:         true,
		IsMountainous: false,
		IsLocal:       true,
	}

	result := evaluator.Evaluate(req)

	// Convective SIGMET must immediately override surface VFR and disqualify
	if result.Verdict != VerdictDisqualified {
		t.Errorf("expected DISQUALIFIED due to Convective SIGMET, got %s", result.Verdict)
	}
	if result.LimitingFactor != LimitConvectiveSIGMET {
		t.Errorf("expected LimitConvectiveSIGMET, got %s", result.LimitingFactor)
	}
}

func TestTier3_CorridorWeather_SevereIcing_Disqualification(t *testing.T) {
	evaluator := &AuthoritativeCorridorEvaluator{}

	req := CorridorEvaluationRequest{
		CorridorStations: []string{"KLXV", "KASE"},
		StationMETARs: map[string]*models.METAR{
			"KLXV": {StationID: "KLXV", CeilingAGLFt: 5000.0, VisibilitySM: 10.0},
			"KASE": {StationID: "KASE", CeilingAGLFt: 4000.0, VisibilitySM: 6.0},
		},
		ActiveHazards: []ParsedSIGMET{
			{
				AdvisoryID:    "CHARLIE 2",
				HazardType:    "SEVERE_ICING",
				Severity:      "SEVERE",
				ValidUntilUTC: time.Now().Add(2 * time.Hour),
				Summary:       "SEV ICE BTN FL100 AND FL180",
				Active:        true,
			},
		},
		IsDay:         true,
		IsMountainous: true,
		IsLocal:       false,
	}

	result := evaluator.Evaluate(req)

	if result.Verdict != VerdictDisqualified {
		t.Errorf("expected DISQUALIFIED due to severe icing, got %s", result.Verdict)
	}
	if result.LimitingFactor != LimitSevereIcing {
		t.Errorf("expected LimitSevereIcing, got %s", result.LimitingFactor)
	}
}

func TestTier3_IFRTransition_AuthorizedUnderFAR135611(t *testing.T) {
	evaluator := &AuthoritativeCorridorEvaluator{}

	// Weather is below VFR minimums (400 ft ceiling / 1 SM vis)
	// But helicopter is IFR certified, has an approved IFR alternate airport, and functioning weather radar
	req := CorridorEvaluationRequest{
		CorridorStations: []string{"KBJC"},
		StationMETARs: map[string]*models.METAR{
			"KBJC": {StationID: "KBJC", CeilingAGLFt: 400.0, VisibilitySM: 1.0},
		},
		IsDay:                 true,
		IsMountainous:         false,
		IsLocal:               true,
		IsIFRCertified:        true,
		HasIFRAlternate:       true,
		HasSevereWeatherRadar: true,
	}

	result := evaluator.Evaluate(req)

	// Under 14 CFR § 135.611, this transitions to IFR instead of outright rejection
	if result.Verdict != VerdictIFRRequired {
		t.Errorf("expected IFR_REQUIRED under § 135.611, got %s", result.Verdict)
	}
	if result.LegalRuleApplied != "14 CFR § 135.611 IFR Operations Authorized" {
		t.Errorf("expected 14 CFR § 135.611 legal citation, got %s", result.LegalRuleApplied)
	}
}

func TestTier3_IFRTransition_DisqualifiedIfMissingAlternate(t *testing.T) {
	evaluator := &AuthoritativeCorridorEvaluator{}

	// Below VFR minimums, IFR certified, but NO alternate airport
	req := CorridorEvaluationRequest{
		CorridorStations: []string{"KBJC"},
		StationMETARs: map[string]*models.METAR{
			"KBJC": {StationID: "KBJC", CeilingAGLFt: 400.0, VisibilitySM: 1.0},
		},
		IsDay:                 true,
		IsMountainous:         false,
		IsLocal:               true,
		IsIFRCertified:        true,
		HasIFRAlternate:       false, // Missing alternate!
		HasSevereWeatherRadar: true,
	}

	result := evaluator.Evaluate(req)

	// Without an alternate, § 135.611 cannot be utilized -> must be BELOW_MINIMUMS
	if result.Verdict != VerdictBelowMinimums {
		t.Errorf("expected BELOW_MINIMUMS when missing alternate, got %s", result.Verdict)
	}
}

func TestTier3_MultiStationCorridor_WorstCaseAggregation(t *testing.T) {
	evaluator := &AuthoritativeCorridorEvaluator{}

	// Corridor across 4 stations:
	// KAPA: 10SM, 8000ft
	// KDEN: 8SM, 5000ft
	// KLXV: 10SM, 4000ft
	// KASE: 2SM, 900ft (Bottleneck!)
	req := CorridorEvaluationRequest{
		CorridorStations: []string{"KAPA", "KDEN", "KLXV", "KASE"},
		StationMETARs: map[string]*models.METAR{
			"KAPA": {StationID: "KAPA", CeilingAGLFt: 8000.0, VisibilitySM: 10.0},
			"KDEN": {StationID: "KDEN", CeilingAGLFt: 5000.0, VisibilitySM: 8.0},
			"KLXV": {StationID: "KLXV", CeilingAGLFt: 4000.0, VisibilitySM: 10.0},
			"KASE": {StationID: "KASE", CeilingAGLFt: 900.0, VisibilitySM: 2.0},
		},
		IsDay:              false,
		IsMountainous:      true,
		IsLocal:            false,
		NVGOrHTAWSEquipped: true, // Req: 1000 ft / 3 SM
	}

	result := evaluator.Evaluate(req)

	if result.WorstCeilingStation != "KASE" || result.WorstCeilingFt != 900.0 {
		t.Errorf("expected worst ceiling KASE 900ft, got %s %.0fft", result.WorstCeilingStation, result.WorstCeilingFt)
	}
	if result.WorstVisStation != "KASE" || result.WorstVisSM != 2.0 {
		t.Errorf("expected worst vis KASE 2SM, got %s %.1fSM", result.WorstVisStation, result.WorstVisSM)
	}
	if result.Verdict != VerdictBelowMinimums {
		t.Errorf("expected corridor verdict BELOW_MINIMUMS due to KASE, got %s", result.Verdict)
	}
}

func TestTier3_GRPC_WebSocket_DualServingSynchronization(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	// 1. Connect WebSocket client
	wsConn, err := h.DialWebSocket()
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer wsConn.Close()

	time.Sleep(50 * time.Millisecond)

	// 2. Trigger gRPC GetCurrentConditions for KAPA
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	grpcResp, err := h.GRPCClient.GetCurrentConditions(ctx, &pb.StationConditionsRequest{
		StationId: "KAPA",
	})
	if err != nil {
		t.Fatalf("gRPC GetCurrentConditions failed: %v", err)
	}
	if grpcResp.StationId != "KAPA" {
		t.Errorf("expected KAPA, got %s", grpcResp.StationId)
	}

	// 3. Verify WebSocket client automatically receives the broadcast event triggered by gRPC!
	evt, err := h.ReadWSEvent(wsConn, 2*time.Second)
	if err != nil {
		t.Fatalf("WebSocket failed to receive event dispatched by gRPC call: %v", err)
	}
	if evt.Type != "METAR_UPDATE" {
		t.Errorf("expected METAR_UPDATE, got %s", evt.Type)
	}
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok || payload["station_id"] != "KAPA" {
		t.Errorf("expected payload station KAPA, got %v", payload["station_id"])
	}
}
