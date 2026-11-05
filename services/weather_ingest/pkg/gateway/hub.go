package gateway

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"traumateam/weather_ingest/pkg/models"
)

type broadcastItem struct {
	topic   string
	message []byte
}

// Hub manages active WebSocket clients, topic subscriptions, and broadcast routing.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcastItem
	mu         sync.RWMutex
	seqCounter uint64
}

// NewHub creates a new broadcast Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan broadcastItem, 512),
	}
}

// NextSeq returns an incrementing sequence number for outgoing events.
func (h *Hub) NextSeq() uint64 {
	return atomic.AddUint64(&h.seqCounter, 1)
}

// Run executes the central hub event loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WebSocket Hub] Client connected. Total active: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[WebSocket Hub] Client disconnected. Total active: %d", len(h.clients))

		case item := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				if client.IsSubscribed(item.topic) {
					select {
					case client.send <- item.message:
					default:
						// Slow consumer protection: non-blocking drop and unregister
						log.Printf("[WebSocket Hub] Slow consumer detected. Dropping client to protect gateway.")
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastWeatherUpdate sends a METAR weather update to all subscribers for stationID or wildcard.
func (h *Hub) BroadcastWeatherUpdate(stationID string, metar *models.METAR) {
	stationID = strings.ToUpper(strings.TrimSpace(stationID))
	evt := ServerEnvelope{
		Event:        EventWeatherUpdate,
		TimestampUTC: time.Now().UTC().Format(time.RFC3339),
		Seq:          h.NextSeq(),
		StationID:    stationID,
		Payload:      metar,
		Metadata: map[string]string{
			"source":    "NOAA_AWC",
			"feed_type": "METAR",
		},
	}
	data, err := json.Marshal(evt)
	if err == nil {
		h.broadcast <- broadcastItem{
			topic:   "station:" + stationID,
			message: data,
		}
	}
}

// BroadcastMinimumsEvaluation sends a flight minimums compliance evaluation event to subscribers.
func (h *Hub) BroadcastMinimumsEvaluation(rep *models.ComplianceReport) {
	evt := ServerEnvelope{
		Event:        EventMinimumsEvaluation,
		TimestampUTC: time.Now().UTC().Format(time.RFC3339),
		Seq:          h.NextSeq(),
		Payload:      rep,
		Metadata: map[string]string{
			"evaluator": "FAA_PART_135_SUBPART_L",
		},
	}
	data, err := json.Marshal(evt)
	if err == nil {
		h.broadcast <- broadcastItem{
			topic:   "minimums",
			message: data,
		}
	}
}

// BroadcastHazardAlert sends an active SIGMET alert to all hazard subscribers.
func (h *Hub) BroadcastHazardAlert(hazard *models.WeatherHazard) {
	evt := ServerEnvelope{
		Event:        EventHazardAlert,
		TimestampUTC: time.Now().UTC().Format(time.RFC3339),
		Seq:          h.NextSeq(),
		Payload:      hazard,
		Metadata: map[string]string{
			"source": "NOAA_AWC_AIRSIGMET",
		},
	}
	data, err := json.Marshal(evt)
	if err == nil {
		h.broadcast <- broadcastItem{
			topic:   "hazards",
			message: data,
		}
	}
}
