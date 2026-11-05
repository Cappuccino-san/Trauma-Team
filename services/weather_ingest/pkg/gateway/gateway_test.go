package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"traumateam/weather_ingest/pkg/models"
	"traumateam/weather_ingest/pkg/noaa"
)

func setupTestGateway(t *testing.T) (*Hub, *httptest.Server, *noaa.WeatherFeedClient, func()) {
	weatherClient := noaa.NewWeatherFeedClient()
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(NewHandler(hub, weatherClient))
	cleanup := func() {
		server.Close()
	}
	return hub, server, weatherClient, cleanup
}

func dialWS(t *testing.T, server *httptest.Server) *websocket.Conn {
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket %s: %v", wsURL, err)
	}
	return conn
}

func TestGateway_ConnectAndPingPong(t *testing.T) {
	_, server, _, cleanup := setupTestGateway(t)
	defer cleanup()

	conn := dialWS(t, server)
	defer conn.Close()

	// Send JSON PING
	pingReq := ClientEnvelope{
		Action:    ActionPing,
		RequestID: "ping-101",
		Timestamp: 1730784000000,
	}
	if err := conn.WriteJSON(pingReq); err != nil {
		t.Fatalf("WriteJSON PING failed: %v", err)
	}

	// Read JSON PONG
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp ServerEnvelope
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("ReadJSON PONG failed: %v", err)
	}

	if resp.Event != EventPong {
		t.Errorf("expected PONG event, got %s", resp.Event)
	}
	if resp.RequestID != "ping-101" {
		t.Errorf("expected request_id ping-101, got %s", resp.RequestID)
	}
}

func TestGateway_SubscribeAndBroadcast(t *testing.T) {
	hub, server, _, cleanup := setupTestGateway(t)
	defer cleanup()

	conn := dialWS(t, server)
	defer conn.Close()

	// Send SUBSCRIBE for station:KAPA
	subReq := ClientEnvelope{
		Action:    ActionSubscribe,
		RequestID: "sub-1",
		Params: ClientParams{
			Stations: []string{"KAPA"},
		},
	}
	if err := conn.WriteJSON(subReq); err != nil {
		t.Fatalf("WriteJSON SUBSCRIBE failed: %v", err)
	}

	// Read ACK
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var ackResp ServerEnvelope
	if err := conn.ReadJSON(&ackResp); err != nil {
		t.Fatalf("ReadJSON ACK failed: %v", err)
	}
	if ackResp.Event != EventAck {
		t.Errorf("expected ACK event, got %s", ackResp.Event)
	}

	// Broadcast weather update for KAPA
	time.Sleep(50 * time.Millisecond) // Allow subscription to settle
	metar := &models.METAR{
		StationID:          "KAPA",
		ObservationTimeUTC: time.Now().UTC(),
		WindDirDeg:         180,
		WindSpeedKts:       12,
		VisibilitySM:       10,
		CeilingAGLFt:       5000,
		FlightCategory:     models.FlightCategoryVFR,
	}
	hub.BroadcastWeatherUpdate("KAPA", metar)

	// Read WEATHER_UPDATE
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var updateResp ServerEnvelope
	if err := conn.ReadJSON(&updateResp); err != nil {
		t.Fatalf("ReadJSON WEATHER_UPDATE failed: %v", err)
	}

	if updateResp.Event != EventWeatherUpdate {
		t.Errorf("expected WEATHER_UPDATE event, got %s", updateResp.Event)
	}
	if updateResp.StationID != "KAPA" {
		t.Errorf("expected station KAPA, got %s", updateResp.StationID)
	}
}

func TestGateway_TopicFiltering(t *testing.T) {
	hub, server, _, cleanup := setupTestGateway(t)
	defer cleanup()

	conn := dialWS(t, server)
	defer conn.Close()

	// Subscribe only to KAPA
	subReq := ClientEnvelope{
		Action: ActionSubscribe,
		Params: ClientParams{
			Stations: []string{"KAPA"},
		},
	}
	conn.WriteJSON(subReq)

	var ack ServerEnvelope
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	conn.ReadJSON(&ack)

	time.Sleep(50 * time.Millisecond)

	// Broadcast for KLXV (client is NOT subscribed to KLXV)
	metarLXV := &models.METAR{
		StationID:    "KLXV",
		VisibilitySM: 10,
	}
	hub.BroadcastWeatherUpdate("KLXV", metarLXV)

	// Attempt to read, should timeout without receiving KLXV
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	var unexp ServerEnvelope
	err := conn.ReadJSON(&unexp)
	if err == nil {
		t.Errorf("expected timeout, but received message for unrequested station: %v", unexp)
	}
}

func TestGateway_EvaluateMinimumsAction(t *testing.T) {
	_, server, _, cleanup := setupTestGateway(t)
	defer cleanup()

	conn := dialWS(t, server)
	defer conn.Close()

	evalReq := ClientEnvelope{
		Action:    ActionEvaluateMinimums,
		RequestID: "eval-1",
		Params: ClientParams{
			Corridor: &CorridorParams{
				Departure:   "KAPA",
				Destination: "KASE",
			},
			RegulatoryContext: &RegulatoryContextParams{
				IsDay:         false,
				IsMountainous: true,
				IsLocal:       false,
			},
		},
	}

	if err := conn.WriteJSON(evalReq); err != nil {
		t.Fatalf("WriteJSON EVALUATE_MINIMUMS failed: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var evalResp ServerEnvelope
	if err := conn.ReadJSON(&evalResp); err != nil {
		t.Fatalf("ReadJSON EVALUATE_MINIMUMS failed: %v", err)
	}

	if evalResp.Event != EventMinimumsEvaluation {
		t.Errorf("expected MINIMUMS_EVALUATION event, got %s", evalResp.Event)
	}
	if evalResp.Payload == nil {
		t.Errorf("expected populated payload in evaluation response")
	}
}

func TestGateway_SlowConsumerProtection(t *testing.T) {
	hub, server, _, cleanup := setupTestGateway(t)
	defer cleanup()

	// Connect slow client that never reads from websocket
	slowConn := dialWS(t, server)
	defer slowConn.Close()

	// Subscribe to all
	sub := ClientEnvelope{
		Action: ActionSubscribe,
		Params: ClientParams{Topics: []string{"all"}},
	}
	slowConn.WriteJSON(sub)

	// Drain ack
	var ack ServerEnvelope
	slowConn.ReadJSON(&ack)

	// Flood broadcast channel with > 300 messages to trigger slow-consumer drop
	for i := 0; i < 300; i++ {
		hub.BroadcastWeatherUpdate("KAPA", &models.METAR{StationID: "KAPA"})
	}

	// Hub should continue running smoothly without blocking or crashing
	fastConn := dialWS(t, server)
	defer fastConn.Close()

	ping := ClientEnvelope{Action: ActionPing}
	if err := fastConn.WriteJSON(ping); err != nil {
		t.Fatalf("Fast client write failed: %v", err)
	}
	fastConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	var pong ServerEnvelope
	if err := fastConn.ReadJSON(&pong); err != nil {
		t.Fatalf("Fast client read failed: %v", err)
	}
	if pong.Event != EventPong {
		t.Errorf("expected PONG, got %s", pong.Event)
	}
}

func TestGateway_RESTEndpoints(t *testing.T) {
	_, server, _, cleanup := setupTestGateway(t)
	defer cleanup()

	// 1. GET /healthz
	respHealth, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz failed: %v", err)
	}
	if respHealth.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /healthz, got %d", respHealth.StatusCode)
	}

	// 2. GET /api/v1/conditions?station=KAPA
	respCond, err := http.Get(server.URL + "/api/v1/conditions?station=KAPA")
	if err != nil {
		t.Fatalf("GET /api/v1/conditions failed: %v", err)
	}
	if respCond.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/conditions, got %d", respCond.StatusCode)
	}

	var data map[string]interface{}
	json.NewDecoder(respCond.Body).Decode(&data)
	if data["station_id"] != "KAPA" {
		t.Errorf("expected station_id KAPA, got %v", data["station_id"])
	}

	// 3. GET /api/v1/kinematics?station=KAPA (invokes Rust FFI physics engine)
	respKin, err := http.Get(server.URL + "/api/v1/kinematics?station=KAPA")
	if err != nil {
		t.Fatalf("GET /api/v1/kinematics failed: %v", err)
	}
	if respKin.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/kinematics, got %d", respKin.StatusCode)
	}
}

func TestGateway_EvaluateKinematicsWebSocketAction(t *testing.T) {
	_, server, _, cleanup := setupTestGateway(t)
	defer cleanup()

	conn := dialWS(t, server)
	defer conn.Close()

	req := ClientEnvelope{
		Action:    ActionEvaluateKinematics,
		RequestID: "kin-101",
		Params: ClientParams{
			Station: "KAPA",
			Kinematics: &KinematicsParams{
				AirframeID:         0,
				StationElevationFt: 5885.0,
				TrueCourseDeg:      270.0,
				LegDistanceNM:      45.0,
				FuelWeightKg:       450.0,
			},
		},
	}

	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("WriteJSON EVALUATE_KINEMATICS failed: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp ServerEnvelope
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("ReadJSON EVALUATE_KINEMATICS failed: %v", err)
	}

	if resp.Event != EventKinematicsEvaluation {
		t.Errorf("expected KINEMATICS_EVALUATION event, got %s", resp.Event)
	}
	if resp.RequestID != "kin-101" {
		t.Errorf("expected request_id kin-101, got %s", resp.RequestID)
	}
}
