#ifndef TRAUMATEAM_RFM_FFI_H
#define TRAUMATEAM_RFM_FFI_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct FfiPayloadItem {
    uint32_t station_id;
    uint32_t _padding;
    double weight_kg;
} FfiPayloadItem;

typedef struct FfiWeatherMissionInput {
    uint32_t airframe_id;
    uint32_t _padding;
    double station_elevation_ft;
    double altimeter_inhg;
    double oat_celsius;
    double wind_dir_deg;
    double wind_speed_kts;
    double true_course_deg;
    double leg_distance_nm;
    double fuel_weight_kg;
} FfiWeatherMissionInput;

typedef struct FfiMassBalanceResult {
    int32_t status_code;
    uint32_t limiting_factor_code;
    uint8_t is_gw_compliant;
    uint8_t is_long_cg_compliant;
    uint8_t is_lat_cg_compliant;
    uint8_t is_overall_compliant;
    uint32_t _padding;
    double gross_weight_kg;
    double cg_long_mm;
    double cg_lat_mm;
    double fwd_limit_mm;
    double aft_limit_mm;
    double left_limit_mm;
    double right_limit_mm;
    double fwd_margin_mm;
    double aft_margin_mm;
    double left_margin_mm;
    double right_margin_mm;
} FfiMassBalanceResult;

typedef struct FfiPerformanceKinematicResult {
    int32_t status_code;
    uint32_t verdict_code;
    uint32_t limiting_factor_code;
    uint8_t is_feasible;
    uint8_t is_wind_flyable;
    uint8_t _padding[2];
    double pressure_altitude_ft;
    double density_altitude_ft;
    double max_ige_hover_weight_kg;
    double max_oge_hover_weight_kg;
    double max_oei_cat_a_weight_kg;
    double ige_margin_kg;
    double oge_margin_kg;
    double oei_cat_a_margin_kg;
    double cruise_tas_kts;
    double cruise_fuel_flow_kg_hr;
    double hover_oge_fuel_flow_kg_hr;
    double loiter_fuel_flow_kg_hr;
    double groundspeed_kts;
    double true_heading_deg;
    double wind_correction_angle_deg;
    double headwind_component_kts;
    double crosswind_component_kts;
    double leg_ete_minutes;
    double leg_fuel_burn_kg;
    double remaining_fuel_kg;
    double reserve_fuel_20min_kg;
    double reserve_margin_kg;
    FfiMassBalanceResult mass_balance;
} FfiPerformanceKinematicResult;

uint32_t traumateam_ffi_abi_version(void);

int32_t traumateam_compute_atmosphere_ffi(
    double elevation_ft,
    double altimeter_inhg,
    double oat_celsius,
    double* out_pa_ft,
    double* out_da_ft
);

int32_t traumateam_eval_mass_balance_ffi(
    uint32_t airframe_id,
    double fuel_weight_kg,
    const FfiPayloadItem* payloads_ptr,
    size_t payloads_len,
    FfiMassBalanceResult* out_result
);

int32_t traumateam_eval_mission_kinematics_ffi(
    const FfiWeatherMissionInput* input_ptr,
    const FfiPayloadItem* payloads_ptr,
    size_t payloads_len,
    FfiPerformanceKinematicResult* out_result
);

#ifdef __cplusplus
}
#endif

#endif /* TRAUMATEAM_RFM_FFI_H */
