package gateway

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"traumateam/weather_ingest/pkg/kinematics"
	"traumateam/weather_ingest/pkg/models"
	"traumateam/weather_ingest/pkg/noaa"
	"traumateam/weather_ingest/pkg/regulations"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8192
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin: func(r *http.Request) bool {
		// Allow CORS for local Svelte development consoles
		return true
	},
}

// Client represents an active WebSocket connection to a frontend client.
type Client struct {
	hub           *Hub
	conn          *websocket.Conn
	send          chan []byte
	weatherClient *noaa.WeatherFeedClient
	subscriptions map[string]bool
	subMu         sync.RWMutex
}

// NewClient constructs a new WebSocket client.
func NewClient(hub *Hub, conn *websocket.Conn, weatherClient *noaa.WeatherFeedClient) *Client {
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 256),
		weatherClient: weatherClient,
		subscriptions: make(map[string]bool),
	}
}

// IsSubscribed checks if the client is subscribed to a topic.
func (c *Client) IsSubscribed(topic string) bool {
	topic = strings.ToLower(strings.TrimSpace(topic))
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	// Default: if no subscriptions configured, client receives all events
	if len(c.subscriptions) == 0 {
		return true
	}
	if c.subscriptions["all"] {
		return true
	}
	if c.subscriptions[topic] {
		return true
	}
	if strings.HasPrefix(topic, "station:") && c.subscriptions["station:*"] {
		return true
	}
	return false
}

// Subscribe adds topics to the client's subscription set.
func (c *Client) Subscribe(topics []string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	for _, t := range topics {
		c.subscriptions[strings.ToLower(strings.TrimSpace(t))] = true
	}
}

// Unsubscribe removes topics from the client's subscription set.
func (c *Client) Unsubscribe(topics []string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	for _, t := range topics {
		delete(c.subscriptions, strings.ToLower(strings.TrimSpace(t)))
	}
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket Client] read error: %v", err)
			}
			break
		}

		var req ClientEnvelope
		if err := json.Unmarshal(message, &req); err != nil {
			c.sendError("", "Invalid JSON request format")
			continue
		}

		c.handleClientAction(&req)
	}
}

func (c *Client) handleClientAction(req *ClientEnvelope) {
	switch req.Action {
	case ActionPing:
		resp := ServerEnvelope{
			Event:        EventPong,
			RequestID:    req.RequestID,
			TimestampUTC: time.Now().UTC().Format(time.RFC3339),
			Seq:          c.hub.NextSeq(),
			Payload: map[string]interface{}{
				"client_timestamp": req.Timestamp,
				"server_epoch_ms":  time.Now().UnixMilli(),
			},
		}
		c.sendJSON(resp)

	case ActionSubscribe:
		topics := make([]string, 0)
		for _, st := range req.Params.Stations {
			topics = append(topics, "station:"+strings.ToUpper(st))
		}
		topics = append(topics, req.Params.Topics...)
		c.Subscribe(topics)

		resp := ServerEnvelope{
			Event:        EventAck,
			RequestID:    req.RequestID,
			TimestampUTC: time.Now().UTC().Format(time.RFC3339),
			Seq:          c.hub.NextSeq(),
			Payload: map[string]interface{}{
				"status":        "SUBSCRIBED",
				"active_topics": topics,
			},
		}
		c.sendJSON(resp)

	case ActionUnsubscribe:
		topics := make([]string, 0)
		for _, st := range req.Params.Stations {
			topics = append(topics, "station:"+strings.ToUpper(st))
		}
		topics = append(topics, req.Params.Topics...)
		c.Unsubscribe(topics)

		resp := ServerEnvelope{
			Event:        EventAck,
			RequestID:    req.RequestID,
			TimestampUTC: time.Now().UTC().Format(time.RFC3339),
			Seq:          c.hub.NextSeq(),
			Payload: map[string]interface{}{
				"status":              "UNSUBSCRIBED",
				"unsubscribed_topics": topics,
			},
		}
		c.sendJSON(resp)

	case ActionGetConditions:
		stationID := req.Params.Station
		if stationID == "" && len(req.Params.Stations) > 0 {
			stationID = req.Params.Stations[0]
		}
		if stationID == "" {
			c.sendError(req.RequestID, "station must be specified for GET_CONDITIONS")
			return
		}

		if c.weatherClient == nil {
			c.sendError(req.RequestID, "weather client not configured")
			return
		}

		metar, err := c.weatherClient.GetMETAR(nil, stationID)
		if err != nil {
			c.sendError(req.RequestID, err.Error())
			return
		}

		resp := ServerEnvelope{
			Event:        EventWeatherSnapshot,
			RequestID:    req.RequestID,
			TimestampUTC: time.Now().UTC().Format(time.RFC3339),
			Seq:          c.hub.NextSeq(),
			StationID:    stationID,
			Payload:      metar,
		}
		c.sendJSON(resp)

	case ActionEvaluateMinimums:
		if c.weatherClient == nil {
			c.sendError(req.RequestID, "weather client not configured")
			return
		}
		c.handleEvaluateMinimums(req)

	case ActionEvaluateKinematics:
		if c.weatherClient == nil {
			c.sendError(req.RequestID, "weather client not configured")
			return
		}
		c.handleEvaluateKinematics(req)

	default:
		c.sendError(req.RequestID, "Unsupported action: "+req.Action)
	}
}

func (c *Client) handleEvaluateKinematics(req *ClientEnvelope) {
	stationID := req.Params.Station
	if stationID == "" {
		stationID = "KAPA"
	}
	metar, err := c.weatherClient.GetMETAR(nil, stationID)
	if err != nil {
		c.sendError(req.RequestID, err.Error())
		return
	}

	kReq := kinematics.MissionKinematicsRequest{
		Airframe:           kinematics.AirframeAirbusH145,
		StationElevationFt: 5885.0,
		Weather:            metar,
		TrueCourseDeg:      270.0,
		LegDistanceNM:      45.0,
		FuelWeightKg:       450.0,
		Payloads: []kinematics.PayloadStationInput{
			{StationID: kinematics.StationPilotRH, WeightKg: 88.0},
			{StationID: kinematics.StationCrewAttendantAftLH, WeightKg: 80.0},
			{StationID: kinematics.StationPrimaryPatientLitter, WeightKg: 92.0},
		},
	}

	if kp := req.Params.Kinematics; kp != nil {
		kReq.Airframe = kinematics.AirframeID(kp.AirframeID)
		if kp.StationElevationFt > 0 {
			kReq.StationElevationFt = kp.StationElevationFt
		}
		kReq.TrueCourseDeg = kp.TrueCourseDeg
		if kp.LegDistanceNM > 0 {
			kReq.LegDistanceNM = kp.LegDistanceNM
		}
		if kp.FuelWeightKg > 0 {
			kReq.FuelWeightKg = kp.FuelWeightKg
		}
		if len(kp.Payloads) > 0 {
			kReq.Payloads = make([]kinematics.PayloadStationInput, len(kp.Payloads))
			for i, p := range kp.Payloads {
				kReq.Payloads[i] = kinematics.PayloadStationInput{
					StationID: kinematics.StationID(p.StationID),
					WeightKg:  p.WeightKg,
				}
			}
		}
	}

	eval, err := kinematics.EvaluateMissionKinematics(kReq)
	if err != nil {
		c.sendError(req.RequestID, err.Error())
		return
	}

	resp := ServerEnvelope{
		Event:        EventKinematicsEvaluation,
		RequestID:    req.RequestID,
		TimestampUTC: time.Now().UTC().Format(time.RFC3339),
		Seq:          c.hub.NextSeq(),
		StationID:    stationID,
		Payload:      eval,
		Metadata: map[string]string{
			"engine": "RUST_RFM_FFI",
		},
	}
	c.sendJSON(resp)
}

func (c *Client) handleEvaluateMinimums(req *ClientEnvelope) {
	stations := make([]models.METAR, 0)

	if req.Params.Corridor != nil {
		stIDs := []string{req.Params.Corridor.Departure}
		stIDs = append(stIDs, req.Params.Corridor.Intermediates...)
		stIDs = append(stIDs, req.Params.Corridor.Destination)

		for _, st := range stIDs {
			if st != "" {
				m, err := c.weatherClient.GetMETAR(nil, st)
				if err == nil {
					stations = append(stations, *m)
				}
			}
		}
	} else if len(req.Params.Stations) > 0 {
		for _, st := range req.Params.Stations {
			m, err := c.weatherClient.GetMETAR(nil, st)
			if err == nil {
				stations = append(stations, *m)
			}
		}
	}

	if len(stations) == 0 {
		c.sendError(req.RequestID, "no valid corridor stations specified for evaluation")
		return
	}

	hazards, _ := c.weatherClient.GetActiveHazards(nil)
	hazardModels := make([]models.WeatherHazard, 0, len(hazards))
	for _, h := range hazards {
		hazardModels = append(hazardModels, *h)
	}

	var ctx models.RegulatoryContext
	if req.Params.RegulatoryContext != nil {
		rc := req.Params.RegulatoryContext
		ctx = models.RegulatoryContext{
			IsDay:                 rc.IsDay,
			IsMountainous:         rc.IsMountainous,
			IsLocalFlyingArea:     rc.IsLocal,
			AllowOpSpecA021Relief: rc.AllowOpSpecA021Relief,
			IsIFRCertified:        rc.IsIFRCertified,
			HasIFRAlternate:       rc.HasIFRAlternate,
			HasSevereWeatherRadar: rc.HasSevereWeatherRadar,
			PlannedCruiseAltMSL:   rc.PlannedCruiseAltMSL,
			HighestObstacleMSL:    rc.HighestObstacleMSL,
			Equipment: models.EquipmentFlags{
				NVGEquipped:            rc.Equipment.NVGEquipped,
				HTAWSEquipped:          rc.Equipment.HTAWSEquipped,
				RadarAltimeterEquipped: rc.Equipment.RadarAltimeterEquipped,
				AutopilotEquipped:      rc.Equipment.AutopilotEquipped,
			},
		}
	} else {
		// Default conservative day mountainous cross-country
		ctx = models.RegulatoryContext{
			IsDay:             true,
			IsMountainous:     true,
			IsLocalFlyingArea: false,
		}
	}

	rep := regulations.EvaluateCorridorCompliance(stations, hazardModels, ctx)

	resp := ServerEnvelope{
		Event:        EventMinimumsEvaluation,
		RequestID:    req.RequestID,
		TimestampUTC: time.Now().UTC().Format(time.RFC3339),
		Seq:          c.hub.NextSeq(),
		Payload:      rep,
	}
	c.sendJSON(resp)
}

func (c *Client) sendJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err == nil {
		select {
		case c.send <- data:
		default:
			// Buffer full, drop message
		}
	}
}

func (c *Client) sendError(requestID, errMsg string) {
	resp := ServerEnvelope{
		Event:        EventError,
		RequestID:    requestID,
		TimestampUTC: time.Now().UTC().Format(time.RFC3339),
		Seq:          c.hub.NextSeq(),
		Error:        errMsg,
	}
	c.sendJSON(resp)
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

			// Drain queued messages as discrete WebSocket frames
			n := len(c.send)
			for i := 0; i < n; i++ {
				if err := c.conn.WriteMessage(websocket.TextMessage, <-c.send); err != nil {
					return
				}
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
