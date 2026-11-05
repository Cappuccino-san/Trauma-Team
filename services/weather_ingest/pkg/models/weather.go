package models

import (
	"math"
	"time"
)

// FlightCategory represents standard FAA flight visibility/ceiling categories.
type FlightCategory string

const (
	FlightCategoryVFR  FlightCategory = "VFR"
	FlightCategoryMVFR FlightCategory = "MVFR"
	FlightCategoryIFR  FlightCategory = "IFR"
	FlightCategoryLIFR FlightCategory = "LIFR"
)

// CloudCoverage represents cloud amount octas.
type CloudCoverage string

const (
	CoverageCLR CloudCoverage = "CLR"
	CoverageSKC CloudCoverage = "SKC"
	CoverageFEW CloudCoverage = "FEW"
	CoverageSCT CloudCoverage = "SCT"
	CoverageBKN CloudCoverage = "BKN" // Constitutes ceiling
	CoverageOVC CloudCoverage = "OVC" // Constitutes ceiling
	CoverageVV  CloudCoverage = "VV"  // Vertical obscuration, constitutes ceiling
)

// CloudLayer represents a single observed cloud deck.
type CloudLayer struct {
	Coverage  CloudCoverage `json:"coverage"`
	BaseFtAGL float64       `json:"base_ft_agl"`
	IsCeiling bool          `json:"is_ceiling"`
	Modifier  string        `json:"modifier,omitempty"` // "CB" or "TCU"
}

// METAR represents a decoded METAR or SPECI observation.
type METAR struct {
	StationID             string         `json:"station_id"`
	ObservationTimeUTC    time.Time      `json:"observation_time_utc"`
	WindDirDeg            float64        `json:"wind_dir_deg"`
	WindSpeedKts          float64        `json:"wind_speed_kts"`
	WindGustKts           float64        `json:"wind_gust_kts"`
	IsVariableWind        bool           `json:"is_variable_wind"`
	VarMinDeg             float64        `json:"var_min_deg,omitempty"`
	VarMaxDeg             float64        `json:"var_max_deg,omitempty"`
	VisibilitySM          float64        `json:"visibility_sm"`
	VisibilityLessThan    bool           `json:"visibility_less_than"`
	VisibilityGreaterThan bool           `json:"visibility_greater_than"`
	CeilingAGLFt          float64        `json:"ceiling_agl_ft"`
	HasCeiling            bool           `json:"has_ceiling"`
	CloudLayers           []CloudLayer   `json:"cloud_layers"`
	TemperatureC          float64        `json:"temperature_c"`
	DewpointC             float64        `json:"dewpoint_c"`
	AltimeterInHg         float64        `json:"altimeter_inhg"`
	PresentWeather        []string       `json:"present_weather"`
	FlightCategory        FlightCategory `json:"flight_category"`
	DensityAltitudeFt     float64        `json:"density_altitude_ft"`
	PressureAltitudeFt    float64        `json:"pressure_altitude_ft"`
	RawText               string         `json:"raw_text"`
}

// WindsAloftLayer represents wind direction, speed, and temperature at an altitude layer.
type WindsAloftLayer struct {
	AltitudeMSLFt float64 `json:"altitude_msl_ft"`
	DirectionDeg  float64 `json:"direction_deg"`
	SpeedKts      float64 `json:"speed_kts"`
	TemperatureC  float64 `json:"temperature_c"`
}

// WindsAloftForecast represents a complete station forecast across altitude layers.
type WindsAloftForecast struct {
	StationID          string            `json:"station_id"`
	ObservationTimeUTC time.Time         `json:"observation_time_utc"`
	Layers             []WindsAloftLayer `json:"layers"`
	RawText            string            `json:"raw_text"`
}

// GeoCoordinate represents a latitude/longitude pair.
type GeoCoordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// HazardType represents meteorological warning categories.
type HazardType string

const (
	HazardTypeConvectiveSIGMET HazardType = "CONVECTIVE_SIGMET"
	HazardTypeSevereTurbulence HazardType = "SEVERE_TURBULENCE"
	HazardTypeSevereIcing      HazardType = "SEVERE_ICING"
	HazardTypeIFRCloudCover    HazardType = "IFR_CLOUD_COVER"
	HazardTypeMountainObsc     HazardType = "MOUNTAIN_OBSCURATION"
	HazardTypeVolcanicAsh      HazardType = "VOLCANIC_ASH"
	HazardTypeDustSandstorm    HazardType = "DUST_SANDSTORM"
)

// WeatherHazard represents an in-flight advisory or SIGMET.
type WeatherHazard struct {
	HazardID    string          `json:"hazard_id"`
	AdvisoryID  string          `json:"advisory_id"`
	HazardType  HazardType      `json:"hazard_type"`
	Severity    string          `json:"severity"` // "LIGHT", "MODERATE", "SEVERE", "EXTREME"
	ValidFrom   time.Time       `json:"valid_from"`
	ValidUntil  time.Time       `json:"valid_until"`
	Summary     string          `json:"summary"`
	MinAltFt    float64         `json:"min_alt_ft"`
	MaxAltFt    float64         `json:"max_alt_ft"`
	Polygon     []GeoCoordinate `json:"polygon"`
	Active      bool            `json:"active"`
	RawText     string          `json:"raw_text"`
}

// SIGMET is an alias for WeatherHazard for clarity.
type SIGMET = WeatherHazard

// Ensure math package is referenced
var _ = math.Abs
