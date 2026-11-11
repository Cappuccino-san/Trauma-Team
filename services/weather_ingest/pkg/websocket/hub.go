package websocket

import (
    "encoding/json"
    "log"
    "sync"
    "time"
    "traumateam/weather_ingest/pkg/models"
    "traumateam/weather_ingest/pkg/regulations"
)

// WSEvent defines the typed JSON contract streamed to frontend clients (e.g. Svelte).
type WSEvent struct {
    Type      string      `json:"type"`      // "METAR_UPDATE", "MINIMUMS_EVAL", "CORRIDOR_ALERT", "PING"
    Timestamp string      `json:"timestamp"` // ISO 8601 UTC
    Payload   interface{} `json:"payload"`
}

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan []byte, 256),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            activeCount := len(h.clients)
            h.mu.Unlock()
            log.Printf("[WebSocket Hub] Client connected. Total active: %d", activeCount)

        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                client.closeSend()
            }
            activeCount := len(h.clients)
            h.mu.Unlock()
            log.Printf("[WebSocket Hub] Client disconnected. Total active: %d", activeCount)

        case message := <-h.broadcast:
            h.mu.RLock()
            var slowClients []*Client
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    slowClients = append(slowClients, client)
                }
            }
            h.mu.RUnlock()

            if len(slowClients) > 0 {
                h.mu.Lock()
                for _, client := range slowClients {
                    if _, ok := h.clients[client]; ok {
                        delete(h.clients, client)
                        client.closeSend()
                    }
                }
                h.mu.Unlock()
            }
        }
    }
}

// BroadcastMETAR formats and dispatches a METAR update to all connected Svelte clients.
func (h *Hub) BroadcastMETAR(obs *models.METAR) {
    evt := WSEvent{
        Type:      "METAR_UPDATE",
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Payload: map[string]interface{}{
            "station_id":      obs.StationID,
            "flight_category": obs.FlightCategory,
            "wind_dir_deg":    obs.WindDirDeg,
            "wind_speed_kts":  obs.WindSpeedKts,
            "wind_gust_kts":   obs.WindGustKts,
            "visibility_sm":   obs.VisibilitySM,
            "ceiling_agl_ft":  obs.CeilingAGLFt,
            "temperature_c":   obs.TemperatureC,
            "dewpoint_c":      obs.DewpointC,
            "altimeter_inhg":  obs.AltimeterInHg,
            "raw_text":        obs.RawText,
        },
    }
    data, err := json.Marshal(evt)
    if err == nil {
        h.broadcast <- data
    }
}

// BroadcastMinimumsEvaluation formats and dispatches a 14 CFR § 135.615 evaluation to all clients.
func (h *Hub) BroadcastMinimumsEvaluation(rep regulations.FlightMinimumsReport) {
    evt := WSEvent{
        Type:      "MINIMUMS_EVAL",
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Payload:   rep,
    }
    data, err := json.Marshal(evt)
    if err == nil {
        h.broadcast <- data
    }
}
