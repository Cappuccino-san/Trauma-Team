package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"traumateam/weather_ingest/pkg/models"
)

func TestWebSocket_ConnectionAndBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	}))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Broadcast sample observation
	obs := &models.METAR{
		StationID:          "KDEN",
		ObservationTimeUTC: time.Now().UTC(),
		WindDirDeg:         170.0,
		WindSpeedKts:       12.0,
		VisibilitySM:       10.0,
		CeilingAGLFt:       6000.0,
		TemperatureC:       19.0,
		DewpointC:          4.0,
		AltimeterInHg:      29.94,
		FlightCategory:     models.FlightCategoryVFR,
		RawText:            "KDEN 050550Z 17012KT 10SM BKN060 19/04 A2994",
	}

	// Give connection a moment to register in hub
	time.Sleep(50 * time.Millisecond)
	hub.BroadcastMETAR(obs)

	// Read message from WebSocket
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message from WebSocket: %v", err)
	}

	var evt WSEvent
	if err := json.Unmarshal(message, &evt); err != nil {
		t.Fatalf("Failed to unmarshal WSEvent: %v", err)
	}

	if evt.Type != "METAR_UPDATE" {
		t.Errorf("expected type METAR_UPDATE, got %s", evt.Type)
	}

	payloadMap, ok := evt.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map payload, got %T", evt.Payload)
	}

	if payloadMap["station_id"] != "KDEN" {
		t.Errorf("expected station KDEN, got %v", payloadMap["station_id"])
	}
}
