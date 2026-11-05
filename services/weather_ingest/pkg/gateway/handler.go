package gateway

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"traumateam/weather_ingest/pkg/noaa"
	"traumateam/weather_ingest/pkg/regulations"
)

// NewHandler constructs the HTTP ServeMux for the WebSocket gateway and REST endpoints.
func NewHandler(hub *Hub, weatherClient *noaa.WeatherFeedClient) http.Handler {
	mux := http.NewServeMux()

	// 1. WebSocket endpoint for frontend Svelte clients
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[WebSocket Gateway] Upgrade error: %v", err)
			return
		}

		client := NewClient(hub, conn, weatherClient)
		hub.register <- client

		go client.writePump()
		go client.readPump()
	})

	// 2. Health checks (/healthz and /health)
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "UP",
			"service":   "weather_ingest",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/health", healthHandler)

	// 3. REST API: GET /api/v1/conditions?station=KAPA
	mux.HandleFunc("/api/v1/conditions", func(w http.ResponseWriter, r *http.Request) {
		station := r.URL.Query().Get("station")
		if station == "" {
			station = "KAPA"
		}
		if weatherClient == nil {
			http.Error(w, "weather client not initialized", http.StatusInternalServerError)
			return
		}

		metar, err := weatherClient.GetMETAR(r.Context(), station)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		taf, _ := weatherClient.GetTAF(r.Context(), station)
		winds, _ := weatherClient.GetWindsAloft(r.Context(), station)
		hazards, _ := weatherClient.GetActiveHazards(r.Context())

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"station_id":  station,
			"metar":       metar,
			"taf":         taf,
			"winds_aloft": winds,
			"hazards":     hazards,
		})
	})

	// 4. REST API: GET /api/v1/minimums
	mux.HandleFunc("/api/v1/minimums", func(w http.ResponseWriter, r *http.Request) {
		station := r.URL.Query().Get("station")
		if station == "" {
			station = "KLXV"
		}
		if weatherClient == nil {
			http.Error(w, "weather client not initialized", http.StatusInternalServerError)
			return
		}

		metar, err := weatherClient.GetMETAR(r.Context(), station)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		isDay := r.URL.Query().Get("day") != "false"
		isMtn := r.URL.Query().Get("mountainous") == "true"
		isLocal := r.URL.Query().Get("local") == "true"
		nvg := r.URL.Query().Get("nvg") != "false"

		report := regulations.EvaluateMinimums(metar, isDay, isMtn, isLocal, nvg)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	})

	return mux
}
