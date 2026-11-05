package test

import (
	"context"
	"encoding/json"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gorilla "github.com/gorilla/websocket"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"traumateam/weather_ingest/pkg/gateway"
	grpcpkg "traumateam/weather_ingest/pkg/grpc"
	"traumateam/weather_ingest/pkg/models"
	"traumateam/weather_ingest/pkg/noaa"
	"traumateam/weather_ingest/pkg/pb"
	"traumateam/weather_ingest/pkg/regulations"
)

const bufSize = 1024 * 1024

// integrationHarness bundles an in-memory gRPC server and in-memory HTTP/WS gateway.
type integrationHarness struct {
	client    *noaa.WeatherFeedClient
	hub       *gateway.Hub
	grpcConn  *googlegrpc.ClientConn
	grpcCl    pb.WeatherIngestServiceClient
	httpSrv   *httptest.Server
	grpcLis   *bufconn.Listener
	rawServer *googlegrpc.Server
}

func newIntegrationHarness(t *testing.T) *integrationHarness {
	t.Helper()

	feedClient := noaa.NewWeatherFeedClient()
	wsHub := gateway.NewHub()
	go wsHub.Run()

	// In-memory gRPC server
	lis := bufconn.Listen(bufSize)
	grpcSrv := googlegrpc.NewServer()
	weatherSrv := grpcpkg.NewWeatherServer(feedClient, wsHub)
	pb.RegisterWeatherIngestServiceServer(grpcSrv, weatherSrv)

	go func() {
		_ = grpcSrv.Serve(lis)
	}()

	// In-memory gRPC client dialer
	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := googlegrpc.NewClient(
		"passthrough://bufnet",
		googlegrpc.WithContextDialer(dialer),
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial in-memory gRPC: %v", err)
	}

	// In-memory HTTP/WebSocket server
	httpHandler := gateway.NewHandler(wsHub, feedClient)
	httpSrv := httptest.NewServer(httpHandler)

	h := &integrationHarness{
		client:    feedClient,
		hub:       wsHub,
		grpcConn:  conn,
		grpcCl:    pb.NewWeatherIngestServiceClient(conn),
		httpSrv:   httpSrv,
		grpcLis:   lis,
		rawServer: grpcSrv,
	}

	t.Cleanup(func() {
		conn.Close()
		grpcSrv.GracefulStop()
		lis.Close()
		httpSrv.Close()
	})

	return h
}

func (h *integrationHarness) dialWS(t *testing.T) *gorilla.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(h.httpSrv.URL, "http") + "/ws"
	wsConn, resp, err := gorilla.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket handshake failed: %v", err)
	}
	if resp.StatusCode != 101 {
		t.Fatalf("WebSocket unexpected handshake status: %d", resp.StatusCode)
	}
	return wsConn
}

// TestIntegration_DualServing_E2E verifies synchronous gRPC ingestion coupled with
// real-time WebSocket push notifications to subscribed clients.
func TestIntegration_DualServing_E2E(t *testing.T) {
	h := newIntegrationHarness(t)

	// 1. Connect WebSocket client and subscribe to station KAPA and KLXV
	wsConn := h.dialWS(t)
	defer wsConn.Close()

	subMsg := gateway.ClientEnvelope{
		Action: gateway.ActionSubscribe,
		Params: gateway.ClientParams{
			Stations: []string{"KAPA", "KLXV"},
		},
	}
	if err := wsConn.WriteJSON(subMsg); err != nil {
		t.Fatalf("Failed to send WebSocket subscription: %v", err)
	}

	// Read initial connected/ack event
	var ackEvt gateway.ServerEnvelope
	_ = wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := wsConn.ReadJSON(&ackEvt); err != nil {
		t.Fatalf("Failed to read WS connection ack: %v", err)
	}
	if ackEvt.Event != gateway.EventAck {
		t.Errorf("expected ACK event, got %s", ackEvt.Event)
	}

	// 2. Query gRPC GetCurrentConditions for KAPA (Centennial)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	condResp, err := h.grpcCl.GetCurrentConditions(ctx, &pb.StationConditionsRequest{
		StationId: "KAPA",
	})
	if err != nil {
		t.Fatalf("GetCurrentConditions RPC failed: %v", err)
	}

	if condResp.StationId != "KAPA" {
		t.Errorf("expected station KAPA, got %s", condResp.StationId)
	}
	if condResp.Metar == nil {
		t.Fatalf("expected populated METAR response")
	}
	if condResp.Metar.FlightCategory != pb.FlightRuleCategory_CATEGORY_VFR {
		t.Errorf("expected VFR flight category, got %v", condResp.Metar.FlightCategory)
	}
	if condResp.Metar.VisibilitySm != 10.0 {
		t.Errorf("expected 10 SM visibility, got %.1f", condResp.Metar.VisibilitySm)
	}

	// 3. Verify WebSocket received the broadcasted weather update triggered by gRPC request
	_ = wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var wsEvt gateway.ServerEnvelope
	if err := wsConn.ReadJSON(&wsEvt); err != nil {
		t.Fatalf("Failed to receive broadcasted WS event: %v", err)
	}

	if wsEvt.Event != gateway.EventWeatherUpdate {
		t.Errorf("expected event WEATHER_UPDATE, got %s", wsEvt.Event)
	}

	payloadBytes, err := json.Marshal(wsEvt.Payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}
	var metarPayload models.METAR
	if err := json.Unmarshal(payloadBytes, &metarPayload); err != nil {
		t.Fatalf("Failed to unmarshal METAR payload: %v", err)
	}
	if metarPayload.StationID != "KAPA" {
		t.Errorf("expected KAPA in WS payload, got %s", metarPayload.StationID)
	}

	// 4. Evaluate FAA Part 135 Subpart L compliance via gRPC
	evalReq := &pb.FlightMinimumsEvaluationRequest{
		DepartureStation:     "KAPA",
		DestinationStation:   "KLXV",
		IsMountainous:        true,
		IsDay:                true,
		IsLocal:              false,
		Equipment: &pb.EquipmentFlags{
			NvgEquipped:   true,
			HtawsEquipped: true,
		},
		PlannedCruiseAltMsl:          11500.0,
		HighestObstacleAlongRouteMsl: 11000.0,
	}

	evalResp, err := h.grpcCl.EvaluateFlightMinimums(ctx, evalReq)
	if err != nil {
		t.Fatalf("EvaluateFlightMinimums RPC failed: %v", err)
	}

	if evalResp.Verdict != pb.DispatchVerdict_VERDICT_APPROVED && evalResp.Verdict != pb.DispatchVerdict_VERDICT_MARGINAL {
		t.Errorf("expected APPROVED or MARGINAL verdict, got %v (explanation: %s)", evalResp.Verdict, evalResp.Explanation)
	}
	if !evalResp.IsCompliant {
		t.Errorf("expected compliant flight profile")
	}
}

// TestIntegration_SIGMET_HazardCorridorDisqualification verifies that when a corridor
// encounters a severe convective SIGMET, the evaluation deterministically disqualifies the flight.
func TestIntegration_SIGMET_HazardCorridorDisqualification(t *testing.T) {
	h := newIntegrationHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	evalReq := &pb.FlightMinimumsEvaluationRequest{
		DepartureStation:     "KDEN",
		DestinationStation:   "KCYS", // Corridor crossing convective SIGMET
		IsMountainous:        false,
		IsDay:                true,
		IsLocal:              false,
		Equipment: &pb.EquipmentFlags{
			NvgEquipped:   false,
			HtawsEquipped: false,
		},
		PlannedCruiseAltMsl:          7500.0,
		HighestObstacleAlongRouteMsl: 6000.0,
	}

	evalResp, err := h.grpcCl.EvaluateFlightMinimums(ctx, evalReq)
	if err != nil {
		t.Fatalf("EvaluateFlightMinimums RPC failed: %v", err)
	}

	if evalResp.Verdict != pb.DispatchVerdict_VERDICT_DISQUALIFIED {
		t.Errorf("expected DISQUALIFIED verdict due to convective SIGMET, got %v", evalResp.Verdict)
	}
	if evalResp.LimitingFactor != pb.LimitingFactor_ACTIVE_CONVECTIVE_SIGMET {
		t.Errorf("expected ACTIVE_CONVECTIVE_SIGMET, got %v", evalResp.LimitingFactor)
	}
}

// TestIntegration_FAR135609_LegalMatrixBoundaries exercises Table 1 standard minimums.
func TestIntegration_FAR135609_LegalMatrixBoundaries(t *testing.T) {
	testCases := []struct {
		name         string
		isMtn        bool
		isDay        bool
		isLocal      bool
		nvg          bool
		expectedCeil float64
		expectedVis  float64
	}{
		{"NonMtn_Day_Local_Standard", false, true, true, false, 800, 2},
		{"NonMtn_Day_XC_Standard", false, true, false, false, 800, 3},
		{"NonMtn_Night_Local_Standard", false, false, true, false, 1000, 3},
		{"NonMtn_Night_XC_Standard", false, false, false, false, 1000, 3},
		{"Mtn_Day_Local", true, true, true, false, 800, 3},
		{"Mtn_Day_XC", true, true, false, false, 1000, 3},
		{"Mtn_Night_Local_NVG", true, false, true, true, 1000, 3},
		{"Mtn_Night_Local_Unaided", true, false, true, false, 1500, 3},
		{"Mtn_Night_XC_NVG", true, false, false, true, 1000, 3},
		{"Mtn_Night_XC_Unaided", true, false, false, false, 1500, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := regulations.GetFAR135615Minimums(tc.isDay, tc.isMtn, tc.isLocal, tc.nvg)
			if m.MinCeilingFt != tc.expectedCeil {
				t.Errorf("expected ceiling %.0f ft, got %.0f ft", tc.expectedCeil, m.MinCeilingFt)
			}
			if m.MinVisibilitySM != tc.expectedVis {
				t.Errorf("expected visibility %.0f SM, got %.0f SM", tc.expectedVis, m.MinVisibilitySM)
			}
		})
	}
}

// TestIntegration_DensityAltitudeThermodynamicAccuracy verifies high-density altitude computation.
func TestIntegration_DensityAltitudeThermodynamicAccuracy(t *testing.T) {
	// Leadville (elev 9,927 ft, altimeter 29.70 inHg, oat 28C)
	pa := noaa.PressureAltitudeFt(9927.0, 29.70)
	da := noaa.DensityAltitudeFt(pa, 28.0)

	if pa < 10000.0 || pa > 10300.0 {
		t.Errorf("expected pressure altitude ~10150 ft, got %.1f", pa)
	}
	if da < 13500.0 || da > 14500.0 {
		t.Errorf("expected density altitude ~14080 ft, got %.1f", da)
	}
}
