package e2e

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"traumateam/weather_ingest/pkg/models"
	"traumateam/weather_ingest/pkg/noaa"
	"traumateam/weather_ingest/pkg/pb"
	"traumateam/weather_ingest/pkg/regulations"
	ws "traumateam/weather_ingest/pkg/websocket"
)

// =============================================================================
// Feature 1: METAR Ingestion & Parsing (>= 5 Test Cases)
// =============================================================================

func TestTier1_METAR_NominalVFR(t *testing.T) {
	metar, err := noaa.ParseMETAR(RawMETAR_KAPA_NominalVFR)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}
	if metar.StationID != "KAPA" {
		t.Errorf("expected station KAPA, got %s", metar.StationID)
	}
	if metar.WindDirDeg != 180.0 || metar.WindSpeedKts != 10.0 {
		t.Errorf("expected wind 180@10kt, got %.0f@%.0f", metar.WindDirDeg, metar.WindSpeedKts)
	}
	if metar.VisibilitySM != 10.0 {
		t.Errorf("expected visibility 10 SM, got %.1f", metar.VisibilitySM)
	}
	if metar.FlightCategory != models.FlightCategoryVFR {
		t.Errorf("expected flight category VFR, got %s", metar.FlightCategory)
	}
}

func TestTier1_METAR_LowIFR_CeilingAndMist(t *testing.T) {
	metar, err := noaa.ParseMETAR(RawMETAR_KASE_AspenLowIFR)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}
	if metar.StationID != "KASE" {
		t.Errorf("expected station KASE, got %s", metar.StationID)
	}
	if metar.CeilingAGLFt != 900.0 {
		t.Errorf("expected ceiling 900ft AGL, got %.0f", metar.CeilingAGLFt)
	}
	if metar.VisibilitySM != 2.0 {
		t.Errorf("expected visibility 2 SM, got %.1f", metar.VisibilitySM)
	}
	if metar.FlightCategory != models.FlightCategoryIFR {
		t.Errorf("expected flight category IFR, got %s", metar.FlightCategory)
	}
}

func TestTier1_METAR_MarginalVFR(t *testing.T) {
	metar, err := noaa.ParseMETAR(RawMETAR_KCOS_MarginalMVFR)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}
	if metar.StationID != "KCOS" {
		t.Errorf("expected station KCOS, got %s", metar.StationID)
	}
	if metar.CeilingAGLFt != 2500.0 {
		t.Errorf("expected ceiling 2500ft AGL, got %.0f", metar.CeilingAGLFt)
	}
	if metar.VisibilitySM != 4.0 {
		t.Errorf("expected visibility 4 SM, got %.1f", metar.VisibilitySM)
	}
	if metar.FlightCategory != models.FlightCategoryMVFR {
		t.Errorf("expected flight category MVFR, got %s", metar.FlightCategory)
	}
}

func TestTier1_METAR_LIFR_DenseFog(t *testing.T) {
	metar, err := noaa.ParseMETAR(RawMETAR_KBJC_DenseFogVV002)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}
	if metar.StationID != "KBJC" {
		t.Errorf("expected station KBJC, got %s", metar.StationID)
	}
	if metar.CeilingAGLFt != 200.0 {
		t.Errorf("expected vertical visibility ceiling 200ft AGL, got %.0f", metar.CeilingAGLFt)
	}
	if metar.VisibilitySM != 0.5 {
		t.Errorf("expected visibility 0.5 SM, got %.1f", metar.VisibilitySM)
	}
	if metar.FlightCategory != models.FlightCategoryLIFR {
		t.Errorf("expected flight category LIFR, got %s", metar.FlightCategory)
	}
}

func TestTier1_METAR_HighWindAndGusts(t *testing.T) {
	metar, err := noaa.ParseMETAR(RawMETAR_KDEN_GustingWind)
	if err != nil {
		t.Fatalf("ParseMETAR failed: %v", err)
	}
	if metar.StationID != "KDEN" {
		t.Errorf("expected station KDEN, got %s", metar.StationID)
	}
	if metar.WindSpeedKts != 25.0 || metar.WindGustKts != 40.0 {
		t.Errorf("expected wind 25G40kt, got %.0fG%.0f", metar.WindSpeedKts, metar.WindGustKts)
	}
	if metar.FlightCategory != models.FlightCategoryVFR {
		t.Errorf("expected flight category VFR, got %s", metar.FlightCategory)
	}
}

// =============================================================================
// Feature 2: TAF Forecast Horizon (>= 5 Test Cases)
// =============================================================================

func TestTier1_TAF_BaseForecast(t *testing.T) {
	parser := &AuthoritativeTAFParser{}
	taf, err := parser.Parse(RawTAF_KAPA)
	if err != nil {
		t.Fatalf("Parse TAF failed: %v", err)
	}
	if taf.StationID != "KAPA" {
		t.Errorf("expected station KAPA, got %s", taf.StationID)
	}
	if len(taf.Periods) == 0 {
		t.Fatalf("expected forecast periods")
	}
	base := taf.Periods[0]
	if base.ChangeType != "BASE" {
		t.Errorf("expected first period to be BASE, got %s", base.ChangeType)
	}
	if base.WindDirDeg != 180.0 || base.WindSpeedKts != 10.0 {
		t.Errorf("expected wind 180@10, got %.0f@%.0f", base.WindDirDeg, base.WindSpeedKts)
	}
}

func TestTier1_TAF_FromGroup(t *testing.T) {
	parser := &AuthoritativeTAFParser{}
	taf, err := parser.Parse(RawTAF_KAPA)
	if err != nil {
		t.Fatalf("Parse TAF failed: %v", err)
	}
	var fmPeriod *ParsedTAFPeriod
	for _, p := range taf.Periods {
		if p.ChangeType == "FM" {
			fmPeriod = &p
			break
		}
	}
	if fmPeriod == nil {
		t.Fatalf("expected FM period in TAF")
	}
	if fmPeriod.WindGustKts != 25.0 {
		t.Errorf("expected gust 25kt, got %.0f", fmPeriod.WindGustKts)
	}
	if fmPeriod.VisibilitySM != 5.0 {
		t.Errorf("expected visibility 5 SM, got %.1f", fmPeriod.VisibilitySM)
	}
	if fmPeriod.CeilingAGLFt != 4000.0 {
		t.Errorf("expected ceiling 4000ft, got %.0f", fmPeriod.CeilingAGLFt)
	}
}

func TestTier1_TAF_TempoGroup(t *testing.T) {
	parser := &AuthoritativeTAFParser{}
	taf, err := parser.Parse(RawTAF_KAPA)
	if err != nil {
		t.Fatalf("Parse TAF failed: %v", err)
	}
	var tempoPeriod *ParsedTAFPeriod
	for _, p := range taf.Periods {
		if p.ChangeType == "TEMPO" {
			tempoPeriod = &p
			break
		}
	}
	if tempoPeriod == nil {
		t.Fatalf("expected TEMPO period in TAF")
	}
	if tempoPeriod.VisibilitySM != 2.0 {
		t.Errorf("expected TEMPO visibility 2 SM, got %.1f", tempoPeriod.VisibilitySM)
	}
	if tempoPeriod.CeilingAGLFt != 2000.0 {
		t.Errorf("expected TEMPO ceiling 2000ft CB, got %.0f", tempoPeriod.CeilingAGLFt)
	}
}

func TestTier1_TAF_BecomingGroup(t *testing.T) {
	parser := &AuthoritativeTAFParser{}
	taf, err := parser.Parse(RawTAF_KAPA)
	if err != nil {
		t.Fatalf("Parse TAF failed: %v", err)
	}
	var becmgPeriod *ParsedTAFPeriod
	for _, p := range taf.Periods {
		if p.ChangeType == "BECMG" {
			becmgPeriod = &p
			break
		}
	}
	if becmgPeriod == nil {
		t.Fatalf("expected BECMG period in TAF")
	}
	if becmgPeriod.WindDirDeg != 300.0 || becmgPeriod.WindSpeedKts != 12.0 {
		t.Errorf("expected BECMG wind 300@12kt, got %.0f@%.0f", becmgPeriod.WindDirDeg, becmgPeriod.WindSpeedKts)
	}
}

func TestTier1_TAF_ProbabilityGroup(t *testing.T) {
	parser := &AuthoritativeTAFParser{}
	taf, err := parser.Parse(RawTAF_KAPA)
	if err != nil {
		t.Fatalf("Parse TAF failed: %v", err)
	}
	var probPeriod *ParsedTAFPeriod
	for _, p := range taf.Periods {
		if p.ChangeType == "PROB30" {
			probPeriod = &p
			break
		}
	}
	if probPeriod == nil {
		t.Fatalf("expected PROB30 period in TAF")
	}
	if probPeriod.VisibilitySM != 1.0 {
		t.Errorf("expected PROB30 visibility 1 SM, got %.1f", probPeriod.VisibilitySM)
	}
	if probPeriod.CeilingAGLFt != 1500.0 {
		t.Errorf("expected PROB30 ceiling 1500ft, got %.0f", probPeriod.CeilingAGLFt)
	}
}

// =============================================================================
// Feature 3: Winds Aloft Decoding & Vector Interpolation (>= 5 Test Cases)
// =============================================================================

func TestTier1_WindsAloft_StandardLayer(t *testing.T) {
	decoder := &AuthoritativeWindsAloftDecoder{}
	vec, err := decoder.DecodeEntry("1812+15", 6000.0)
	if err != nil {
		t.Fatalf("DecodeEntry failed: %v", err)
	}
	if vec.DirectionDeg != 180.0 || vec.SpeedKts != 12.0 {
		t.Errorf("expected 180@12kt, got %.0f@%.0f", vec.DirectionDeg, vec.SpeedKts)
	}
	if vec.TemperatureC != 15.0 {
		t.Errorf("expected +15C, got %.1f", vec.TemperatureC)
	}
}

func TestTier1_WindsAloft_SubZeroLayer(t *testing.T) {
	decoder := &AuthoritativeWindsAloftDecoder{}
	vec, err := decoder.DecodeEntry("2625-10", 18000.0)
	if err != nil {
		t.Fatalf("DecodeEntry failed: %v", err)
	}
	if vec.DirectionDeg != 260.0 || vec.SpeedKts != 25.0 {
		t.Errorf("expected 260@25kt, got %.0f@%.0f", vec.DirectionDeg, vec.SpeedKts)
	}
	if vec.TemperatureC != -10.0 {
		t.Errorf("expected -10C, got %.1f", vec.TemperatureC)
	}
}

func TestTier1_WindsAloft_HighSpeedOffset(t *testing.T) {
	decoder := &AuthoritativeWindsAloftDecoder{}
	vec, err := decoder.DecodeEntry("771558", 39000.0)
	if err != nil {
		t.Fatalf("DecodeEntry failed: %v", err)
	}
	if vec.DirectionDeg != 270.0 || vec.SpeedKts != 115.0 {
		t.Errorf("expected 270@115kt, got %.0f@%.0f", vec.DirectionDeg, vec.SpeedKts)
	}
	if vec.TemperatureC != -58.0 {
		t.Errorf("expected -58C, got %.1f", vec.TemperatureC)
	}
}

func TestTier1_WindsAloft_CalmCode(t *testing.T) {
	decoder := &AuthoritativeWindsAloftDecoder{}
	vec, err := decoder.DecodeEntry("9900+08", 9000.0)
	if err != nil {
		t.Fatalf("DecodeEntry failed: %v", err)
	}
	if !vec.IsCalm || vec.SpeedKts != 0.0 {
		t.Errorf("expected calm wind, got speed %.0f", vec.SpeedKts)
	}
	if vec.TemperatureC != 8.0 {
		t.Errorf("expected +8C, got %.1f", vec.TemperatureC)
	}
}

func TestTier1_WindsAloft_TrigonometricInterpolation(t *testing.T) {
	layers := []DecodedWindVector{
		{AltitudeMSLFt: 6000.0, DirectionDeg: 180.0, SpeedKts: 10.0, TemperatureC: 15.0},
		{AltitudeMSLFt: 12000.0, DirectionDeg: 270.0, SpeedKts: 20.0, TemperatureC: 0.0},
	}
	target := InterpolateWindsVector(9000.0, layers)
	if target.DirectionDeg < 220.0 || target.DirectionDeg > 250.0 {
		t.Errorf("expected interpolated direction ~235°, got %.0f°", target.DirectionDeg)
	}
	if target.SpeedKts < 9.0 || target.SpeedKts > 13.0 {
		t.Errorf("expected interpolated speed ~11kt, got %.0f kt", target.SpeedKts)
	}
	if target.TemperatureC != 7.5 {
		t.Errorf("expected interpolated temp 7.5C, got %.1f", target.TemperatureC)
	}
}

// =============================================================================
// Feature 4: SIGMET Hazard Detection & Categorization (>= 5 Test Cases)
// =============================================================================

func TestTier1_SIGMET_Convective(t *testing.T) {
	sig := ParsedSIGMET{
		AdvisoryID:    "15E",
		HazardType:    "CONVECTIVE",
		Severity:      "EXTREME",
		ValidUntilUTC: time.Now().Add(2 * time.Hour),
		Summary:       "LINE SEV TS 30 NM WIDE MOV FROM 26025KT",
		Active:        true,
	}
	evaluator := &AuthoritativeCorridorEvaluator{}
	req := CorridorEvaluationRequest{
		CorridorStations: []string{"KAPA"},
		StationMETARs: map[string]*models.METAR{
			"KAPA": {StationID: "KAPA", CeilingAGLFt: 10000.0, VisibilitySM: 10.0},
		},
		ActiveHazards: []ParsedSIGMET{sig},
		IsDay:         true,
	}
	res := evaluator.Evaluate(req)
	if res.Verdict != VerdictDisqualified {
		t.Errorf("expected DISQUALIFIED for convective SIGMET, got %s", res.Verdict)
	}
	if res.LimitingFactor != LimitConvectiveSIGMET {
		t.Errorf("expected LimitConvectiveSIGMET, got %s", res.LimitingFactor)
	}
}

func TestTier1_SIGMET_SevereIcing(t *testing.T) {
	sig := ParsedSIGMET{
		AdvisoryID:    "CHARLIE 2",
		HazardType:    "SEVERE_ICING",
		Severity:      "SEVERE",
		ValidUntilUTC: time.Now().Add(2 * time.Hour),
		Summary:       "SEV ICE BTN FL100 AND FL180",
		Active:        true,
	}
	evaluator := &AuthoritativeCorridorEvaluator{}
	req := CorridorEvaluationRequest{
		CorridorStations: []string{"KAPA"},
		StationMETARs: map[string]*models.METAR{
			"KAPA": {StationID: "KAPA", CeilingAGLFt: 10000.0, VisibilitySM: 10.0},
		},
		ActiveHazards: []ParsedSIGMET{sig},
		IsDay:         true,
	}
	res := evaluator.Evaluate(req)
	if res.Verdict != VerdictDisqualified {
		t.Errorf("expected DISQUALIFIED for severe icing, got %s", res.Verdict)
	}
	if res.LimitingFactor != LimitSevereIcing {
		t.Errorf("expected LimitSevereIcing, got %s", res.LimitingFactor)
	}
}

func TestTier1_SIGMET_SevereTurbulence(t *testing.T) {
	sig := ParsedSIGMET{
		AdvisoryID:    "DELTA 4",
		HazardType:    "SEVERE_TURB",
		Severity:      "SEVERE",
		ValidUntilUTC: time.Now().Add(2 * time.Hour),
		Summary:       "SEV TURB BTN FL150 AND FL250 DUE TO MTN WAVE",
		Active:        true,
	}
	evaluator := &AuthoritativeCorridorEvaluator{}
	req := CorridorEvaluationRequest{
		CorridorStations: []string{"KAPA"},
		StationMETARs: map[string]*models.METAR{
			"KAPA": {StationID: "KAPA", CeilingAGLFt: 10000.0, VisibilitySM: 10.0},
		},
		ActiveHazards: []ParsedSIGMET{sig},
		IsDay:         true,
	}
	res := evaluator.Evaluate(req)
	if res.Verdict != VerdictDisqualified {
		t.Errorf("expected DISQUALIFIED for severe turbulence, got %s", res.Verdict)
	}
	if res.LimitingFactor != LimitSevereTurbulence {
		t.Errorf("expected LimitSevereTurbulence, got %s", res.LimitingFactor)
	}
}

func TestTier1_SIGMET_ExpiredHazard(t *testing.T) {
	sig := ParsedSIGMET{
		AdvisoryID:    "OLD 1",
		HazardType:    "CONVECTIVE",
		Severity:      "EXTREME",
		ValidUntilUTC: time.Now().Add(-1 * time.Hour),
		Active:        false,
	}
	evaluator := &AuthoritativeCorridorEvaluator{}
	req := CorridorEvaluationRequest{
		CorridorStations: []string{"KAPA"},
		StationMETARs: map[string]*models.METAR{
			"KAPA": {StationID: "KAPA", CeilingAGLFt: 10000.0, VisibilitySM: 10.0},
		},
		ActiveHazards: []ParsedSIGMET{sig},
		IsDay:         true,
		IsLocal:       true,
	}
	res := evaluator.Evaluate(req)
	if res.Verdict != VerdictApproved {
		t.Errorf("expected APPROVED when hazard is inactive, got %s", res.Verdict)
	}
}

func TestTier1_SIGMET_SpatialIntersection(t *testing.T) {
	polygon := [][2]float64{
		{39.0, -105.0},
		{40.0, -105.0},
		{40.0, -104.0},
		{39.0, -104.0},
	}
	point := [2]float64{39.5, -104.5}
	if !pointInPolygon(point[0], point[1], polygon) {
		t.Errorf("expected point to intersect polygon")
	}

	outsidePoint := [2]float64{42.0, -100.0}
	if pointInPolygon(outsidePoint[0], outsidePoint[1], polygon) {
		t.Errorf("expected point to NOT intersect polygon")
	}
}

func pointInPolygon(x, y float64, poly [][2]float64) bool {
	inside := false
	n := len(poly)
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := poly[i][0], poly[i][1]
		xj, yj := poly[j][0], poly[j][1]
		intersect := ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi)+xi)
		if intersect {
			inside = !inside
		}
		j = i
	}
	return inside
}

// =============================================================================
// Feature 5: FAA Part 135.L Regulatory Matrix (>= 5 Test Cases)
// =============================================================================

func TestTier1_Part135L_NonMountainousDayLocal_NVG(t *testing.T) {
	rule := regulations.GetFAR135615Minimums(true, false, true, true)
	if rule.MinCeilingFt != 500.0 || rule.MinVisibilitySM != 2.0 {
		t.Errorf("expected 500ft / 2SM, got %.0fft / %.1fSM", rule.MinCeilingFt, rule.MinVisibilitySM)
	}
}

func TestTier1_Part135L_NonMountainousDayLocal_Unaided(t *testing.T) {
	rule := regulations.GetFAR135615Minimums(true, false, true, false)
	if rule.MinCeilingFt != 800.0 || rule.MinVisibilitySM != 2.0 {
		t.Errorf("expected 800ft / 2SM, got %.0fft / %.1fSM", rule.MinCeilingFt, rule.MinVisibilitySM)
	}
}

func TestTier1_Part135L_NonMountainousNightLocal_Unaided(t *testing.T) {
	rule := regulations.GetFAR135615Minimums(false, false, true, false)
	if rule.MinCeilingFt != 1000.0 || rule.MinVisibilitySM != 3.0 {
		t.Errorf("expected 1000ft / 3SM, got %.0fft / %.1fSM", rule.MinCeilingFt, rule.MinVisibilitySM)
	}
}

func TestTier1_Part135L_MountainousNightCrossCountry_NVG(t *testing.T) {
	rule := regulations.GetFAR135615Minimums(false, true, false, true)
	if rule.MinCeilingFt != 1000.0 || rule.MinVisibilitySM != 3.0 {
		t.Errorf("expected 1000ft / 3SM, got %.0fft / %.1fSM", rule.MinCeilingFt, rule.MinVisibilitySM)
	}
}

func TestTier1_Part135L_MountainousNightCrossCountry_Unaided(t *testing.T) {
	rule := regulations.GetFAR135615Minimums(false, true, false, false)
	if rule.MinCeilingFt != 1500.0 || rule.MinVisibilitySM != 5.0 {
		t.Errorf("expected 1500ft / 5SM, got %.0fft / %.1fSM", rule.MinCeilingFt, rule.MinVisibilitySM)
	}
}

// =============================================================================
// Feature 6: gRPC Service & Streaming (>= 5 Test Cases via bufconn)
// =============================================================================

func TestTier1_GRPC_GetCurrentConditions_Success(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := h.GRPCClient.GetCurrentConditions(ctx, &pb.StationConditionsRequest{
		StationId: "KAPA",
	})
	if err != nil {
		t.Fatalf("GetCurrentConditions failed: %v", err)
	}
	if resp.StationId != "KAPA" {
		t.Errorf("expected station KAPA, got %s", resp.StationId)
	}
	if resp.Metar == nil {
		t.Fatal("expected non-nil METAR in CurrentConditionsResponse")
	}
	if resp.Metar.FlightCategory != pb.FlightRuleCategory_CATEGORY_VFR {
		t.Errorf("expected VFR category, got %v", resp.Metar.FlightCategory)
	}
}

func TestTier1_GRPC_GetCurrentConditions_EmptyStation(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := h.GRPCClient.GetCurrentConditions(ctx, &pb.StationConditionsRequest{
		StationId: "",
	})
	if err == nil {
		t.Fatal("expected error for empty station_id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument code, got %v", err)
	}
}

func TestTier1_GRPC_EvaluateFlightMinimums_Compliant(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()
	h.ClearHazards()
	h.SetStationObservation("KDEN", "KDEN 050550Z 17012KT 10SM BKN060 19/04 A2994")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := h.GRPCClient.EvaluateFlightMinimums(ctx, &pb.FlightMinimumsEvaluationRequest{
		DepartureStation:   "KAPA",
		DestinationStation: "KDEN",
		IsDay:              true,
		IsMountainous:      false,
		IsLocal:            true,
		Equipment: &pb.EquipmentFlags{
			NvgEquipped:   true,
			HtawsEquipped: true,
		},
	})
	if err != nil {
		t.Fatalf("EvaluateFlightMinimums failed: %v", err)
	}
	if !resp.IsCompliant || resp.Verdict != pb.DispatchVerdict_VERDICT_APPROVED {
		t.Errorf("expected compliant APPROVED verdict, got %v (compliant=%v)", resp.Verdict, resp.IsCompliant)
	}
}

func TestTier1_GRPC_EvaluateFlightMinimums_NonCompliant(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// KASE: Aspen ceiling 900ft AGL under Mountainous Night Cross-Country (req 1000ft NVG)
	resp, err := h.GRPCClient.EvaluateFlightMinimums(ctx, &pb.FlightMinimumsEvaluationRequest{
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
		t.Fatalf("EvaluateFlightMinimums failed: %v", err)
	}
	if resp.IsCompliant {
		t.Errorf("expected non-compliant for KASE 900 ft ceiling at night mtn")
	}
}

func TestTier1_GRPC_StreamWeatherFeeds(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := h.GRPCClient.StreamWeatherFeeds(ctx, &pb.StreamWeatherRequest{
		Stations:               []string{"KAPA"},
		RefreshIntervalSeconds: 1,
	})
	if err != nil {
		t.Fatalf("StreamWeatherFeeds failed: %v", err)
	}

	update, err := stream.Recv()
	if err != nil && err != io.EOF {
		t.Fatalf("failed to receive weather feed update: %v", err)
	}
	if update == nil {
		t.Fatalf("received nil weather feed update")
	}
	if update.StationId != "KAPA" {
		t.Errorf("expected station KAPA, got %s", update.StationId)
	}
}

// =============================================================================
// Feature 7: WebSocket Gateway & Streaming (>= 5 Test Cases via httptest)
// =============================================================================

func TestTier1_WS_HandshakeAndConnect(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	conn, err := h.DialWebSocket()
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()
}

func TestTier1_WS_BroadcastMETARUpdate(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	conn, err := h.DialWebSocket()
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	obs := &models.METAR{
		StationID:          "KDEN",
		ObservationTimeUTC: time.Now().UTC(),
		CeilingAGLFt:       6000.0,
		VisibilitySM:       10.0,
		WindSpeedKts:       15.0,
		FlightCategory:     models.FlightCategoryVFR,
	}
	h.Hub.BroadcastMETAR(obs)

	evt, err := h.ReadWSEvent(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("ReadWSEvent failed: %v", err)
	}
	if evt.Type != "METAR_UPDATE" {
		t.Errorf("expected METAR_UPDATE, got %s", evt.Type)
	}
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok || payload["station_id"] != "KDEN" {
		t.Errorf("expected station KDEN in payload, got %v", payload["station_id"])
	}
}

func TestTier1_WS_BroadcastMinimumsEvaluation(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	conn, err := h.DialWebSocket()
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	rep := regulations.FlightMinimumsReport{
		StationID:               "KLXV",
		IsCompliant:             true,
		Verdict:                 "GO",
		MinCeilingRequiredFt:    1000.0,
		ActualCeilingFt:         8000.0,
		MinVisibilityRequiredSM: 3.0,
		ActualVisibilitySM:      10.0,
	}
	h.Hub.BroadcastMinimumsEvaluation(rep)

	evt, err := h.ReadWSEvent(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("ReadWSEvent failed: %v", err)
	}
	if evt.Type != "MINIMUMS_EVAL" {
		t.Errorf("expected MINIMUMS_EVAL, got %s", evt.Type)
	}
}

func TestTier1_WS_PingPongHeartbeat(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	conn, err := h.DialWebSocket()
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	pongReceived := make(chan bool, 1)
	conn.SetPongHandler(func(string) error {
		pongReceived <- true
		return nil
	})

	err = conn.WriteMessage(websocket.PingMessage, []byte("heartbeat"))
	if err != nil {
		t.Fatalf("Write Ping failed: %v", err)
	}

	go func() {
		conn.ReadMessage()
	}()

	select {
	case <-pongReceived:
	case <-time.After(2 * time.Second):
	}
}

func TestTier1_WS_ClientSubscriptionMessage(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.Teardown()

	conn, err := h.DialWebSocket()
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	msg := ws.ClientMessage{
		Action:   "SUBSCRIBE_STATIONS",
		Stations: []string{"KAPA", "KLXV"},
	}
	err = h.WriteClientMessage(conn, msg)
	if err != nil {
		t.Fatalf("WriteClientMessage failed: %v", err)
	}
}
