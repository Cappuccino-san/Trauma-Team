package gateway

// Client actions supported by the WebSocket protocol.
const (
	ActionSubscribe       = "SUBSCRIBE"
	ActionUnsubscribe     = "UNSUBSCRIBE"
	ActionGetConditions   = "GET_CONDITIONS"
	ActionEvaluateMinimums = "EVALUATE_MINIMUMS"
	ActionPing            = "PING"
)

// Server events pushed over the WebSocket connection.
const (
	EventWeatherSnapshot   = "WEATHER_SNAPSHOT"
	EventWeatherUpdate     = "WEATHER_UPDATE"
	EventMinimumsEvaluation = "MINIMUMS_EVALUATION"
	EventHazardAlert       = "HAZARD_ALERT"
	EventPong              = "PONG"
	EventAck               = "ACK"
	EventError             = "ERROR"
)

// CorridorParams contains route station parameters.
type CorridorParams struct {
	Departure     string   `json:"departure"`
	Destination   string   `json:"destination"`
	Intermediates []string `json:"intermediates"`
}

// RegulatoryContextParams contains mission regulatory flags.
type RegulatoryContextParams struct {
	IsDay                 bool     `json:"is_day"`
	IsMountainous         bool     `json:"is_mountainous"`
	IsLocal               bool     `json:"is_local"`
	AllowOpSpecA021Relief bool     `json:"allow_opspec_a021_relief"`
	IsIFRCertified        bool     `json:"is_ifr_certified"`
	HasIFRAlternate       bool     `json:"has_ifr_alternate"`
	HasSevereWeatherRadar bool     `json:"has_severe_weather_radar"`
	PlannedCruiseAltMSL   float64  `json:"planned_cruise_alt_msl"`
	HighestObstacleMSL    float64  `json:"highest_obstacle_msl"`
	Equipment             struct {
		NVGEquipped            bool `json:"nvg_equipped"`
		HTAWSEquipped          bool `json:"htaws_equipped"`
		RadarAltimeterEquipped bool `json:"radar_altimeter_equipped"`
		AutopilotEquipped      bool `json:"autopilot_equipped"`
	} `json:"equipment"`
}

// ClientParams represents the polymorphic payload in client requests.
type ClientParams struct {
	Station           string                  `json:"station,omitempty"`
	Stations          []string                `json:"stations,omitempty"`
	Topics            []string                `json:"topics,omitempty"`
	Corridor          *CorridorParams         `json:"corridor,omitempty"`
	RegulatoryContext *RegulatoryContextParams `json:"regulatory_context,omitempty"`
}

// ClientEnvelope defines the inbound JSON contract from frontend clients (e.g. Svelte).
type ClientEnvelope struct {
	Action    string       `json:"action"`
	RequestID string       `json:"request_id,omitempty"`
	Timestamp int64        `json:"timestamp,omitempty"` // epoch ms for ping
	Params    ClientParams `json:"params,omitempty"`
}

// ServerEnvelope defines the outbound JSON contract streamed to frontend clients.
type ServerEnvelope struct {
	Event        string            `json:"event"`
	RequestID    string            `json:"request_id,omitempty"`
	TimestampUTC string            `json:"timestamp_utc"`
	Seq          uint64            `json:"seq"`
	StationID    string            `json:"station_id,omitempty"`
	Payload      interface{}       `json:"payload,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Error        string            `json:"error,omitempty"`
}
