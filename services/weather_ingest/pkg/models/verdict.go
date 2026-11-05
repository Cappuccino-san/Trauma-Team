package models

// DispatchVerdict represents the definitive mission feasibility status.
type DispatchVerdict string

const (
	VerdictUnspecified   DispatchVerdict = "UNSPECIFIED"
	VerdictApproved      DispatchVerdict = "APPROVED"       // Fully compliant with positive safety margins
	VerdictMarginal      DispatchVerdict = "MARGINAL"       // Legally compliant, but within safety caution buffer
	VerdictBelowMinimums DispatchVerdict = "BELOW_MINIMUMS" // Ceiling or visibility below 14 CFR § 135.609
	VerdictDisqualified  DispatchVerdict = "DISQUALIFIED"   // Severe hazard (SIGMET, severe icing/turbulence)
	VerdictIFRRequired   DispatchVerdict = "IFR_REQUIRED"   // VFR below minimums, but compliant with § 135.611 IFR

	// Backward-compatible aliases
	VerdictGo     DispatchVerdict = "GO"
	VerdictCaution DispatchVerdict = "CAUTION"
	VerdictNoGo   DispatchVerdict = "NO-GO"
)

// LimitingFactor details the precise regulatory or environmental disqualifier.
type LimitingFactor string

const (
	LimitNone                          LimitingFactor = "LIMITING_FACTOR_NONE"
	LimitFAR135609CeilingBelowMinimums LimitingFactor = "FAR_135_609_CEILING_BELOW_MINIMUMS"
	LimitFAR135609VisBelowMinimums     LimitingFactor = "FAR_135_609_VISIBILITY_BELOW_MINIMUMS"
	LimitFAR135615ObstacleClearance    LimitingFactor = "FAR_135_615_OBSTACLE_CLEARANCE_VIOLATION"
	LimitFAR135209FuelReserveViolation LimitingFactor = "FAR_135_209_VFR_RESERVE_VIOLATION"
	LimitConvectiveSIGMET              LimitingFactor = "ACTIVE_CONVECTIVE_SIGMET"
	LimitSevereIcing                   LimitingFactor = "SEVERE_ICING_HAZARD"
	LimitSevereTurbulence              LimitingFactor = "SEVERE_TURBULENCE_HAZARD"
	LimitUnflyableWind                 LimitingFactor = "UNFLYABLE_WIND"
	LimitIFRAlternateMissing           LimitingFactor = "IFR_ALTERNATE_MISSING"
	LimitNoWeatherWithin15NM           LimitingFactor = "NO_WEATHER_REPORTING_WITHIN_15NM"

	// Backward-compatible aliases
	LimitFAR135615CeilingBelowMinimums LimitingFactor = "FAR_135_615_CEILING_BELOW_MINIMUMS"
	LimitFAR135615VisBelowMinimums     LimitingFactor = "FAR_135_615_VISIBILITY_BELOW_MINIMUMS"
)

// EquipmentFlags describes aircraft avionics and night-vision capabilities.
type EquipmentFlags struct {
	NVGEquipped            bool `json:"nvg_equipped"`
	HTAWSEquipped          bool `json:"htaws_equipped"`
	RadarAltimeterEquipped bool `json:"radar_altimeter_equipped"`
	AutopilotEquipped      bool `json:"autopilot_equipped"`
}

// RegulatoryContext contains operational mission parameters for compliance evaluation.
type RegulatoryContext struct {
	IsMountainous         bool           `json:"is_mountainous"`
	IsLocalFlyingArea     bool           `json:"is_local_flying_area"`
	IsDay                 bool           `json:"is_day"`
	Equipment             EquipmentFlags `json:"equipment"`
	AllowOpSpecA021Relief bool           `json:"allow_opspec_a021_relief"`
	IsIFRCertified        bool           `json:"is_ifr_certified"`
	HasIFRAlternate       bool           `json:"has_ifr_alternate"`
	HasSevereWeatherRadar bool           `json:"has_severe_weather_radar"`
	MaxCrosswindLimitKts  float64        `json:"max_crosswind_limit_kts"`
	PlannedCruiseAltMSL   float64        `json:"planned_cruise_alt_msl"`
	HighestObstacleMSL    float64        `json:"highest_obstacle_msl"`
}

// StationComplianceReport contains evaluation results for a single observation station.
type StationComplianceReport struct {
	StationID          string         `json:"station_id"`
	IsCompliant        bool           `json:"is_compliant"`
	ActualCeilingFt    float64        `json:"actual_ceiling_ft"`
	RequiredCeilingFt  float64        `json:"required_ceiling_ft"`
	CeilingMarginFt    float64        `json:"ceiling_margin_ft"`
	ActualVisSM        float64        `json:"actual_vis_sm"`
	RequiredVisSM      float64        `json:"required_vis_sm"`
	VisMarginSM        float64        `json:"vis_margin_sm"`
	Category           FlightCategory `json:"category"`
	Explanation        string         `json:"explanation"`
}

// ComplianceReport is the output of the deterministic FAA compliance evaluator.
type ComplianceReport struct {
	IsCompliant          bool                      `json:"is_compliant"`
	Verdict              DispatchVerdict           `json:"verdict"`
	LimitingFactor       LimitingFactor            `json:"limiting_factor"`
	WorstCeilingFt       float64                   `json:"worst_ceiling_ft"`
	WorstVisibilitySM    float64                   `json:"worst_visibility_sm"`
	MinRequiredCeilingFt float64                   `json:"min_required_ceiling_ft"`
	MinRequiredVisSM     float64                   `json:"min_required_vis_sm"`
	CeilingMarginFt      float64                   `json:"ceiling_margin_ft"`
	VisMarginSM          float64                   `json:"vis_margin_sm"`
	LegalRuleApplied     string                    `json:"legal_rule_applied"`
	Explanation          string                    `json:"explanation"`
	StationReports       []StationComplianceReport `json:"station_reports"`
	ActiveHazards        []WeatherHazard           `json:"active_hazards"`
	EvaluationTimeUTC    string                    `json:"evaluation_time_utc"`
	AuditTrace           []string                  `json:"audit_trace"`
}
