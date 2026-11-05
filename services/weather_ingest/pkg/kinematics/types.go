package kinematics

import (
	"fmt"
	"traumateam/weather_ingest/pkg/models"
)

// AirframeID identifies the certified OEM rotorcraft model in the Rust RFM engine.
type AirframeID uint32

const (
	AirframeAirbusH145 AirframeID = 0
	AirframeBell429    AirframeID = 1
)

func (a AirframeID) String() string {
	switch a {
	case AirframeAirbusH145:
		return "Airbus H145 (BK117 D-2)"
	case AirframeBell429:
		return "Bell 429 GlobalRanger"
	default:
		return "Unknown Airframe"
	}
}

// StationID identifies an airframe mass-and-balance payload station in the Rust RFM engine.
type StationID uint32

const (
	StationPilotRH                StationID = 0
	StationCopilotLH              StationID = 1
	StationCrewAttendantAftLH     StationID = 2
	StationCrewAttendantAftRH     StationID = 3
	StationCrewAttendantLH        StationID = 4
	StationCrewAttendantRH        StationID = 5
	StationPrimaryPatientLitter   StationID = 6
	StationSecondaryPatientLitter StationID = 7
	StationMedicalCabinetRack     StationID = 8
	StationMedicalEquipmentRack   StationID = 9
	StationOxygenSystem           StationID = 10
	StationAftAvionicsCargo       StationID = 11
	StationAftBaggage             StationID = 12
	StationRescueHoistRH          StationID = 13
)

// PayloadStationInput represents a discrete payload weight placed at an aircraft station.
type PayloadStationInput struct {
	StationID StationID `json:"station_id"`
	WeightKg  float64   `json:"weight_kg"`
}

// cFfiPayloadItem mirrors Rust `#[repr(C)] pub struct FfiPayloadItem`.
type cFfiPayloadItem struct {
	StationID uint32
	Padding   uint32
	WeightKg  float64
}

// cFfiWeatherMissionInput mirrors Rust `#[repr(C)] pub struct FfiWeatherMissionInput`.
type cFfiWeatherMissionInput struct {
	AirframeID         uint32
	Padding            uint32
	StationElevationFt float64
	AltimeterInHg      float64
	OATCelsius         float64
	WindDirDeg         float64
	WindSpeedKts       float64
	TrueCourseDeg      float64
	LegDistanceNM      float64
	FuelWeightKg       float64
}

// cFfiMassBalanceResult mirrors Rust `#[repr(C)] pub struct FfiMassBalanceResult`.
type cFfiMassBalanceResult struct {
	StatusCode         int32
	LimitingFactorCode uint32
	IsGWCompliant      uint8
	IsLongCGCompliant  uint8
	IsLatCGCompliant   uint8
	IsOverallCompliant uint8
	Padding            uint32
	GrossWeightKg      float64
	CGLongMM           float64
	CGLatMM            float64
	FwdLimitMM         float64
	AftLimitMM         float64
	LeftLimitMM        float64
	RightLimitMM       float64
	FwdMarginMM        float64
	AftMarginMM        float64
	LeftMarginMM       float64
	RightMarginMM      float64
}

// cFfiPerformanceKinematicResult mirrors Rust `#[repr(C)] pub struct FfiPerformanceKinematicResult`.
type cFfiPerformanceKinematicResult struct {
	StatusCode              int32
	VerdictCode             uint32
	LimitingFactorCode      uint32
	IsFeasible              uint8
	IsWindFlyable           uint8
	Padding                 [2]uint8
	PressureAltitudeFt      float64
	DensityAltitudeFt       float64
	MaxIGEHoverWeightKg     float64
	MaxOGEHoverWeightKg     float64
	MaxOEICatAWeightKg      float64
	IGEMarginKg             float64
	OGEMarginKg             float64
	OEICatAMarginKg         float64
	CruiseTASKts            float64
	CruiseFuelFlowKgHr      float64
	HoverOGEFuelFlowKgHr    float64
	LoiterFuelFlowKgHr      float64
	GroundspeedKts          float64
	TrueHeadingDeg          float64
	WindCorrectionAngleDeg  float64
	HeadwindComponentKts    float64
	CrosswindComponentKts   float64
	LegETEMinutes           float64
	LegFuelBurnKg           float64
	RemainingFuelKg         float64
	ReserveFuel20MinKg      float64
	ReserveMarginKg         float64
	MassBalance             cFfiMassBalanceResult
}

// MassBalanceEvaluation is the idiomatic Go struct parsed from `FfiMassBalanceResult`.
type MassBalanceEvaluation struct {
	IsGWCompliant      bool    `json:"is_gw_compliant"`
	IsLongCGCompliant  bool    `json:"is_long_cg_compliant"`
	IsLatCGCompliant   bool    `json:"is_lat_cg_compliant"`
	IsOverallCompliant bool    `json:"is_overall_compliant"`
	LimitingFactor     string  `json:"limiting_factor"`
	GrossWeightKg      float64 `json:"gross_weight_kg"`
	CGLongMM           float64 `json:"cg_long_mm"`
	CGLatMM            float64 `json:"cg_lat_mm"`
	FwdLimitMM         float64 `json:"fwd_limit_mm"`
	AftLimitMM         float64 `json:"aft_limit_mm"`
	LeftLimitMM        float64 `json:"left_limit_mm"`
	RightLimitMM       float64 `json:"right_limit_mm"`
	FwdMarginMM        float64 `json:"fwd_margin_mm"`
	AftMarginMM        float64 `json:"aft_margin_mm"`
	LeftMarginMM       float64 `json:"left_margin_mm"`
	RightMarginMM      float64 `json:"right_margin_mm"`
}

// MissionKinematicsRequest combines meteorological data (from a decoded NOAA METAR)
// with mission route and payload parameters to be evaluated by the Rust physics core.
type MissionKinematicsRequest struct {
	Airframe           AirframeID            `json:"airframe"`
	StationElevationFt float64               `json:"station_elevation_ft"`
	Weather            *models.METAR         `json:"weather"`
	TrueCourseDeg      float64               `json:"true_course_deg"`
	LegDistanceNM      float64               `json:"leg_distance_nm"`
	FuelWeightKg       float64               `json:"fuel_weight_kg"`
	Payloads           []PayloadStationInput `json:"payloads"`
}

// MissionKinematicsEvaluation is the idiomatic Go struct parsed from `FfiPerformanceKinematicResult`.
type MissionKinematicsEvaluation struct {
	Airframe               string                `json:"airframe"`
	Verdict                string                `json:"verdict"`
	LimitingFactor         string                `json:"limiting_factor"`
	IsFeasible             bool                  `json:"is_feasible"`
	IsWindFlyable          bool                  `json:"is_wind_flyable"`
	PressureAltitudeFt     float64               `json:"pressure_altitude_ft"`
	DensityAltitudeFt      float64               `json:"density_altitude_ft"`
	MaxIGEHoverWeightKg    float64               `json:"max_ige_hover_weight_kg"`
	MaxOGEHoverWeightKg    float64               `json:"max_oge_hover_weight_kg"`
	MaxOEICatAWeightKg     float64               `json:"max_oei_cat_a_weight_kg"`
	IGEMarginKg            float64               `json:"ige_margin_kg"`
	OGEMarginKg            float64               `json:"oge_margin_kg"`
	OEICatAMarginKg        float64               `json:"oei_cat_a_margin_kg"`
	CruiseTASKts           float64               `json:"cruise_tas_kts"`
	CruiseFuelFlowKgHr     float64               `json:"cruise_fuel_flow_kg_hr"`
	HoverOGEFuelFlowKgHr   float64               `json:"hover_oge_fuel_flow_kg_hr"`
	LoiterFuelFlowKgHr     float64               `json:"loiter_fuel_flow_kg_hr"`
	GroundspeedKts         float64               `json:"groundspeed_kts"`
	TrueHeadingDeg         float64               `json:"true_heading_deg"`
	WindCorrectionAngleDeg float64               `json:"wind_correction_angle_deg"`
	HeadwindComponentKts   float64               `json:"headwind_component_kts"`
	CrosswindComponentKts  float64               `json:"crosswind_component_kts"`
	LegETEMinutes          float64               `json:"leg_ete_minutes"`
	LegFuelBurnKg          float64               `json:"leg_fuel_burn_kg"`
	RemainingFuelKg        float64               `json:"remaining_fuel_kg"`
	ReserveFuel20MinKg     float64               `json:"reserve_fuel_20min_kg"`
	ReserveMarginKg        float64               `json:"reserve_margin_kg"`
	MassBalance            MassBalanceEvaluation `json:"mass_balance"`
}

func decodeVerdictCode(code uint32) string {
	switch code {
	case 0:
		return "GO"
	case 1:
		return "CAUTION"
	default:
		return "NO-GO"
	}
}

func decodeLimitingFactorCode(code uint32) string {
	switch code {
	case 0:
		return "NONE"
	case 1:
		return "MTOW_EXCEEDED"
	case 2:
		return "MIN_WEIGHT_VIOLATION"
	case 3:
		return "LONGITUDINAL_CG_FORWARD_EXCEEDED"
	case 4:
		return "LONGITUDINAL_CG_AFT_EXCEEDED"
	case 5:
		return "LATERAL_CG_LEFT_EXCEEDED"
	case 6:
		return "LATERAL_CG_RIGHT_EXCEEDED"
	case 7:
		return "OGE_HOVER_SCENE_DEFICIT"
	case 8:
		return "IGE_HOVER_DEFICIT"
	case 9:
		return "OEI_CAT_A_HELIPAD_DEFICIT"
	case 10:
		return "FAR_135_615_CEILING_BELOW_MINIMUMS"
	case 11:
		return "FAR_135_615_VISIBILITY_BELOW_MINIMUMS"
	case 12:
		return "FAR_135_209_VFR_RESERVE_VIOLATION"
	case 13:
		return "IFR_RESERVE_VIOLATION"
	case 14:
		return "SEVERE_WEATHER_HAZARD"
	case 15:
		return "TERRAIN_OBSTACLE_CLEARANCE_VIOLATION"
	case 16:
		return "UNFLYABLE_WIND"
	default:
		return fmt.Sprintf("FFI_ERROR_CODE_%d", code)
	}
}

func marshalCPayloads(payloads []PayloadStationInput) []cFfiPayloadItem {
	if len(payloads) == 0 {
		return nil
	}
	out := make([]cFfiPayloadItem, len(payloads))
	for i, p := range payloads {
		out[i] = cFfiPayloadItem{
			StationID: uint32(p.StationID),
			Padding:   0,
			WeightKg:  p.WeightKg,
		}
	}
	return out
}

func parseMassBalanceResult(raw cFfiMassBalanceResult) MassBalanceEvaluation {
	return MassBalanceEvaluation{
		IsGWCompliant:      raw.IsGWCompliant != 0,
		IsLongCGCompliant:  raw.IsLongCGCompliant != 0,
		IsLatCGCompliant:   raw.IsLatCGCompliant != 0,
		IsOverallCompliant: raw.IsOverallCompliant != 0,
		LimitingFactor:     decodeLimitingFactorCode(raw.LimitingFactorCode),
		GrossWeightKg:      raw.GrossWeightKg,
		CGLongMM:           raw.CGLongMM,
		CGLatMM:            raw.CGLatMM,
		FwdLimitMM:         raw.FwdLimitMM,
		AftLimitMM:         raw.AftLimitMM,
		LeftLimitMM:        raw.LeftLimitMM,
		RightLimitMM       : raw.RightLimitMM,
		FwdMarginMM:        raw.FwdMarginMM,
		AftMarginMM:        raw.AftMarginMM,
		LeftMarginMM:       raw.LeftMarginMM,
		RightMarginMM:      raw.RightMarginMM,
	}
}

func parseKinematicsResult(airframe AirframeID, raw cFfiPerformanceKinematicResult) *MissionKinematicsEvaluation {
	return &MissionKinematicsEvaluation{
		Airframe:               airframe.String(),
		Verdict:                decodeVerdictCode(raw.VerdictCode),
		LimitingFactor:         decodeLimitingFactorCode(raw.LimitingFactorCode),
		IsFeasible:             raw.IsFeasible != 0,
		IsWindFlyable:          raw.IsWindFlyable != 0,
		PressureAltitudeFt:     raw.PressureAltitudeFt,
		DensityAltitudeFt:      raw.DensityAltitudeFt,
		MaxIGEHoverWeightKg:    raw.MaxIGEHoverWeightKg,
		MaxOGEHoverWeightKg:    raw.MaxOGEHoverWeightKg,
		MaxOEICatAWeightKg:     raw.MaxOEICatAWeightKg,
		IGEMarginKg:            raw.IGEMarginKg,
		OGEMarginKg:            raw.OGEMarginKg,
		OEICatAMarginKg:        raw.OEICatAMarginKg,
		CruiseTASKts:           raw.CruiseTASKts,
		CruiseFuelFlowKgHr:     raw.CruiseFuelFlowKgHr,
		HoverOGEFuelFlowKgHr:   raw.HoverOGEFuelFlowKgHr,
		LoiterFuelFlowKgHr:     raw.LoiterFuelFlowKgHr,
		GroundspeedKts:         raw.GroundspeedKts,
		TrueHeadingDeg:         raw.TrueHeadingDeg,
		WindCorrectionAngleDeg: raw.WindCorrectionAngleDeg,
		HeadwindComponentKts:   raw.HeadwindComponentKts,
		CrosswindComponentKts:  raw.CrosswindComponentKts,
		LegETEMinutes:          raw.LegETEMinutes,
		LegFuelBurnKg:          raw.LegFuelBurnKg,
		RemainingFuelKg:        raw.RemainingFuelKg,
		ReserveFuel20MinKg:     raw.ReserveFuel20MinKg,
		ReserveMarginKg:        raw.ReserveMarginKg,
		MassBalance:            parseMassBalanceResult(raw.MassBalance),
	}
}
