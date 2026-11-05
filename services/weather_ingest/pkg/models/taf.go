package models

import "time"

// TAFChangeType represents the forecast group type.
type TAFChangeType string

const (
	TAFChangeBase   TAFChangeType = "BASE"
	TAFChangeFM     TAFChangeType = "FM"
	TAFChangeTEMPO  TAFChangeType = "TEMPO"
	TAFChangePROB30 TAFChangeType = "PROB30"
	TAFChangePROB40 TAFChangeType = "PROB40"
	TAFChangeBECMG  TAFChangeType = "BECMG"
)

// TAFPeriod represents a single forecast period within a TAF bulletin.
type TAFPeriod struct {
	ChangeType     TAFChangeType  `json:"change_type"`
	ValidFromUTC   time.Time      `json:"valid_from_utc"`
	ValidToUTC     time.Time      `json:"valid_to_utc"`
	WindDirDeg     float64        `json:"wind_dir_deg"`
	WindSpeedKts   float64        `json:"wind_speed_kts"`
	WindGustKts    float64        `json:"wind_gust_kts"`
	VisibilitySM   float64        `json:"visibility_sm"`
	CeilingAGLFt   float64        `json:"ceiling_agl_ft"`
	HasCeiling     bool           `json:"has_ceiling"`
	CloudLayers    []CloudLayer   `json:"cloud_layers"`
	PresentWeather []string       `json:"present_weather"`
	FlightCategory FlightCategory `json:"flight_category"`
}

// TAF represents a decoded Terminal Aerodrome Forecast bulletin.
type TAF struct {
	StationID    string      `json:"station_id"`
	IssueTimeUTC time.Time   `json:"issue_time_utc"`
	ValidFromUTC time.Time   `json:"valid_from_utc"`
	ValidToUTC   time.Time   `json:"valid_to_utc"`
	Periods      []TAFPeriod `json:"periods"`
	RawText      string      `json:"raw_text"`
}

// GetConditionsAtETA extracts the prevailing forecast period at the mission ETA, plus any active TEMPO/PROB periods.
func (t *TAF) GetConditionsAtETA(eta time.Time) (prevailing TAFPeriod, tempo []TAFPeriod) {
	if len(t.Periods) == 0 {
		return TAFPeriod{VisibilitySM: 10, CeilingAGLFt: 25000, FlightCategory: FlightCategoryVFR}, nil
	}

	// Base is initial prevailing
	prevailing = t.Periods[0]

	for _, p := range t.Periods {
		if p.ChangeType == TAFChangeFM {
			if !eta.Before(p.ValidFromUTC) {
				prevailing = p
			}
		} else if p.ChangeType == TAFChangeTEMPO || p.ChangeType == TAFChangePROB30 || p.ChangeType == TAFChangePROB40 {
			if (!eta.Before(p.ValidFromUTC)) && eta.Before(p.ValidToUTC) {
				tempo = append(tempo, p)
			}
		}
	}

	return prevailing, tempo
}

