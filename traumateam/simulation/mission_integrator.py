"""Sequential Mission State-Machine & Kinematic Integrator Engine.

Simulates the complete HEMS mission loop leg-by-leg with dynamic fuel burn integration,
shifted CG envelope compliance, wind-corrected groundspeed, and local density altitude.
"""

from __future__ import annotations
from datetime import datetime, timezone
from typing import List, Dict, Optional, Tuple
import numpy as np

from traumateam.models.rfm import RFMAirframeSpec
from traumateam.models.mission import (
    MissionProfile,
    LegSpec,
    LegType,
    PayloadItem,
    Waypoint,
)
from traumateam.models.weather import CorridorWeather
from traumateam.models.verdict import (
    DispatchVerdict,
    LimitingFactor,
    ComplianceMetric,
    MissionLegReport,
    DispatchEvaluation,
)
from traumateam.core.kinematics import (
    great_circle_distance_nm,
    initial_bearing_deg,
    solve_wind_triangle,
    WindVector,
)
from traumateam.core.atmosphere import (
    pressure_altitude_ft,
    density_altitude_ft,
)
from traumateam.performance.mass_balance import compute_mass_and_balance, MassBalanceSnapshot
from traumateam.performance.rfm_engine import RFMPerformanceEngine
from traumateam.weather_terrain.weather_eval import interpolate_winds_aloft, evaluate_corridor_weather
from traumateam.weather_terrain.terrain_clearance import evaluate_terrain_and_obstacles
from traumateam.regulations.fuel_reserves import evaluate_fuel_reserves

class MissionSimulationIntegrator:
    def __init__(self, airframe: RFMAirframeSpec):
        self.airframe = airframe
        self.rfm_engine = RFMPerformanceEngine(airframe)

    def _get_active_payloads(self, all_payloads: List[PayloadItem], leg_idx: int) -> List[PayloadItem]:
        active = []
        for p in all_payloads:
            if leg_idx >= p.loaded_at_leg:
                if p.unloaded_at_leg is None or leg_idx < p.unloaded_at_leg:
                    active.append(p)
        return active

    def execute_mission(
        self,
        mission: MissionProfile,
        weather: CorridorWeather,
        altimeter_inhg: float = 29.92,
        ambient_temp_c: float = 15.0,
    ) -> DispatchEvaluation:
        """Executes the full deterministic multi-leg mission kinematic simulation."""
        timestamp_utc = datetime.now(timezone.utc).isoformat()
        audit_trace: List[str] = []
        mitigation_actions: List[str] = []
        legs_report: List[MissionLegReport] = []

        audit_trace.append(f"Starting Mission Simulation ID '{mission.mission_id}' with Tail '{mission.tail_number}' ({self.airframe.model_name})")
        audit_trace.append(f"Airframe MTOW: {self.airframe.mtow_kg:.1f} kg | Initial Fuel: {mission.initial_fuel_kg:.1f} kg")

        # 1. Pre-flight Corridor Weather & FAR 135.615 Minimums Evaluation
        weather_verdict = evaluate_corridor_weather(
            corridor=weather,
            is_day=mission.is_day,
            is_mountainous=mission.is_mountainous,
            is_local=mission.is_local_flying_area,
            nvg_or_htaws_equipped=mission.nvg_or_htaws_equipped,
        )
        audit_trace.extend(weather_verdict.audit_notes)

        overall_verdict = weather_verdict.verdict
        primary_limiting_factor = weather_verdict.limiting_factor
        limiting_description = ""
        if not weather_verdict.is_flyable:
            if weather_verdict.active_hazard_summary:
                limiting_description = weather_verdict.active_hazard_summary
            else:
                limiting_description = weather_verdict.compliance_report.explanation

        # Track state across legs
        current_fuel_kg = mission.initial_fuel_kg
        total_distance_nm = 0.0
        total_flight_time_min = 0.0
        total_fuel_burn_kg = 0.0

        # Initial fuel check against fuel capacity
        if current_fuel_kg > self.airframe.fuel_system.usable_fuel_capacity_kg:
            overall_verdict = DispatchVerdict.NO_GO
            primary_limiting_factor = LimitingFactor.MTOW_EXCEEDED
            excess_fuel = current_fuel_kg - self.airframe.fuel_system.usable_fuel_capacity_kg
            limiting_description = f"Fuel planned ({current_fuel_kg:.1f} kg) exceeds usable fuel capacity ({self.airframe.fuel_system.usable_fuel_capacity_kg:.1f} kg) by {excess_fuel:.1f} kg."
            mitigation_actions.append(f"Defuel aircraft by at least {excess_fuel:.1f} kg.")

        # 2. Iterate sequentially through each leg
        for leg in mission.legs:
            leg_idx = leg.leg_index
            audit_trace.append(f"\n--- Simulating Leg {leg_idx}: {leg.name} ({leg.leg_type.value}) ---")

            # Active payload at start of leg
            active_payloads = self._get_active_payloads(mission.payloads, leg_idx)
            
            # Mass & Balance snapshot before leg operations
            mb_start = compute_mass_and_balance(self.airframe, current_fuel_kg, active_payloads)
            
            audit_trace.append(
                f"Start Leg {leg_idx}: GW={mb_start.gross_weight_kg:.1f} kg, Fuel={current_fuel_kg:.1f} kg, "
                f"CG_long={mb_start.cg_long_mm:.1f} mm [{mb_start.fwd_limit_mm:.1f} to {mb_start.aft_limit_mm:.1f}], "
                f"CG_lat={mb_start.cg_lat_mm:.1f} mm [{mb_start.left_limit_mm:.1f} to {mb_start.right_limit_mm:.1f}]"
            )

            if not mb_start.is_overall_compliant:
                audit_trace.append(f"MB VIOLATION at start of Leg {leg_idx}: {mb_start.detail_message}")
                if overall_verdict != DispatchVerdict.NO_GO:
                    overall_verdict = DispatchVerdict.NO_GO
                    primary_limiting_factor = mb_start.limiting_factor
                    limiting_description = mb_start.detail_message
                if mb_start.gross_weight_kg > self.airframe.mtow_kg:
                    mitigation_actions.append(f"Offload {mb_start.gross_weight_kg - self.airframe.mtow_kg:.1f} kg before Leg {leg_idx}.")

            # Atmospheric environment at destination/scene
            dest_pa = pressure_altitude_ft(leg.destination.elevation_ft, altimeter_inhg=altimeter_inhg)
            dest_da = density_altitude_ft(dest_pa, oat_c=ambient_temp_c)
            cruise_pa = pressure_altitude_ft(leg.planned_cruise_altitude_msl_ft, altimeter_inhg=altimeter_inhg)

            # Terrain & Obstacle Clearance Check for transit legs
            if leg.leg_type == LegType.TRANSIT:
                terrain_rep = evaluate_terrain_and_obstacles(
                    origin=leg.origin,
                    destination=leg.destination,
                    planned_cruise_alt_msl_ft=leg.planned_cruise_altitude_msl_ft,
                    is_day=mission.is_day,
                    is_mountainous=mission.is_mountainous,
                )
                audit_trace.append(terrain_rep.detail_message)
                if not terrain_rep.is_cleared:
                    if overall_verdict != DispatchVerdict.NO_GO:
                        overall_verdict = DispatchVerdict.NO_GO
                        primary_limiting_factor = terrain_rep.limiting_factor
                        limiting_description = terrain_rep.detail_message
                    mitigation_actions.append(f"Increase cruise altitude for Leg {leg_idx} to at least {terrain_rep.min_safe_altitude_msl_ft:.0f} ft MSL.")

            # Kinematic calculations: Distance, Course, Wind Triangle
            leg_dist_nm = 0.0
            tas_kts = 0.0
            gs_kts = 0.0
            hw_kts = 0.0
            xw_kts = 0.0
            transit_time_hr = 0.0

            if leg.leg_type == LegType.TRANSIT:
                leg_dist_nm = great_circle_distance_nm(leg.origin.geo_coord, leg.destination.geo_coord)
                true_course = initial_bearing_deg(leg.origin.geo_coord, leg.destination.geo_coord)
                
                # Interpolate winds aloft at cruise altitude
                wind_at_alt = interpolate_winds_aloft(leg.planned_cruise_altitude_msl_ft, weather.winds_aloft_profile)
                
                # True Airspeed from RFM tables
                if leg.target_tas_kts is not None:
                    tas_kts = leg.target_tas_kts
                else:
                    tas_kts = self.rfm_engine.get_cruise_tas_knots(cruise_pa, ambient_temp_c, mb_start.gross_weight_kg)
                
                # Solve wind triangle
                wt = solve_wind_triangle(true_course, tas_kts, wind_at_alt)
                if not wt.is_flyable:
                    overall_verdict = DispatchVerdict.NO_GO
                    primary_limiting_factor = LimitingFactor.UNFLYABLE_WIND
                    limiting_description = wt.unflyable_reason or "Unflyable wind conditions"
                    audit_trace.append(f"WIND TRIANGLE FAILURE: {limiting_description}")
                    gs_kts = 1.0  # Avoid div by zero in reporting
                else:
                    gs_kts = wt.groundspeed_kts
                    hw_kts = wt.headwind_component_kts
                    xw_kts = wt.crosswind_component_kts

                transit_time_hr = leg_dist_nm / max(1.0, gs_kts)
                audit_trace.append(
                    f"Transit Kinematics: Dist={leg_dist_nm:.1f} NM, TC={true_course:.1f} deg, TAS={tas_kts:.1f} kt, "
                    f"Wind={wind_at_alt.direction_deg:03.0f}@{wind_at_alt.speed_kts:.0f}kt, HW={hw_kts:+.1f}kt, "
                    f"XW={xw_kts:+.1f}kt, GS={gs_kts:.1f}kt, ETE={transit_time_hr*60.0:.1f} min"
                )

            # Fuel Burn Integration across all operations in this leg
            cruise_ff = self.rfm_engine.get_cruise_fuel_flow_kg_per_hr(cruise_pa, ambient_temp_c, mb_start.gross_weight_kg)
            oge_ff = self.rfm_engine.get_hover_oge_fuel_flow_kg_per_hr(dest_pa, ambient_temp_c, mb_start.gross_weight_kg)
            loiter_ff = self.rfm_engine.get_loiter_fuel_flow_kg_per_hr(dest_pa, ambient_temp_c, mb_start.gross_weight_kg)

            # Leg phase times
            transit_min = transit_time_hr * 60.0
            hover_oge_min = leg.hover_oge_min
            hover_ige_min = leg.hover_ige_min
            ground_idle_min = leg.ground_idle_min

            # Startup fuel on Leg 0
            startup_fuel = self.airframe.startup_fuel_kg if leg.leg_type == LegType.STARTUP else 0.0

            leg_burn_kg = (
                startup_fuel +
                (transit_min / 60.0) * cruise_ff +
                (hover_oge_min / 60.0) * oge_ff +
                (hover_ige_min / 60.0) * cruise_ff +
                (ground_idle_min / 60.0) * loiter_ff
            )

            total_leg_duration_min = transit_min + hover_oge_min + hover_ige_min + ground_idle_min

            total_distance_nm += leg_dist_nm
            total_flight_time_min += total_leg_duration_min
            total_fuel_burn_kg += leg_burn_kg
            current_fuel_kg = max(0.0, current_fuel_kg - leg_burn_kg)

            # Mass & Balance snapshot at end of leg
            mb_end = compute_mass_and_balance(self.airframe, current_fuel_kg, active_payloads)
            
            if not mb_end.is_overall_compliant:
                audit_trace.append(f"MB VIOLATION at end of Leg {leg_idx}: {mb_end.detail_message}")
                if overall_verdict != DispatchVerdict.NO_GO:
                    overall_verdict = DispatchVerdict.NO_GO
                    primary_limiting_factor = mb_end.limiting_factor
                    limiting_description = mb_end.detail_message

            # Hover & Helipad Performance Verification
            max_allowable_gw = self.airframe.mtow_kg
            hover_margin = 0.0
            metrics: List[ComplianceMetric] = []

            # 1. Scene Operations: OGE / IGE Hover check
            if leg.leg_type == LegType.SCENE_OPERATIONS or hover_oge_min > 0.0:
                max_oge = self.rfm_engine.get_max_oge_hover_weight(dest_pa, ambient_temp_c)
                max_allowable_gw = max_oge
                hover_margin = max_oge - mb_start.gross_weight_kg
                is_oge_ok = (hover_margin >= 0.0)

                metrics.append(ComplianceMetric(
                    name="Scene OGE Hover Weight Margin",
                    actual_value=mb_start.gross_weight_kg,
                    limit_value=max_oge,
                    margin=hover_margin,
                    unit="kg",
                    is_compliant=is_oge_ok,
                    notes=f"Scene PA={dest_pa:.0f} ft, OAT={ambient_temp_c:.1f} C, DA={dest_da:.0f} ft"
                ))

                audit_trace.append(
                    f"Scene Hover Check: Scene PA={dest_pa:.0f} ft, DA={dest_da:.0f} ft -> "
                    f"Max OGE={max_oge:.1f} kg vs Current GW={mb_start.gross_weight_kg:.1f} kg (Margin: {hover_margin:+.1f} kg)"
                )

                if not is_oge_ok:
                    audit_trace.append(f"CRITICAL DEFICIT: Cannot perform OGE hover at scene! Deficit is {abs(hover_margin):.1f} kg.")
                    if overall_verdict != DispatchVerdict.NO_GO:
                        overall_verdict = DispatchVerdict.NO_GO
                        primary_limiting_factor = LimitingFactor.OGE_HOVER_SCENE_DEFICIT
                        limiting_description = (
                            f"Scene OGE Hover Performance Deficit: Aircraft Gross Weight ({mb_start.gross_weight_kg:.1f} kg) "
                            f"exceeds RFM Out-of-Ground-Effect hover ceiling ({max_oge:.1f} kg) by {abs(hover_margin):.1f} kg "
                            f"at Density Altitude {dest_da:.0f} ft MSL (PA: {dest_pa:.0f} ft, OAT: {ambient_temp_c:.1f} C)."
                        )
                    mitigation_actions.append(
                        f"Reduce departure gross weight by at least {abs(hover_margin):.1f} kg (e.g. defuel {abs(hover_margin):.1f} kg Jet-A or offload non-critical gear) to achieve OGE hover at scene."
                    )

            # 2. Hospital Helipad Operations: Category A / OEI Balked Landing check
            if leg.leg_type == LegType.HOSPITAL_OPERATIONS and leg.destination.is_elevated_helipad:
                max_oei = self.rfm_engine.get_max_oei_cat_a_weight(dest_pa, ambient_temp_c)
                max_allowable_gw = min(max_allowable_gw, max_oei)
                oei_margin = max_oei - mb_start.gross_weight_kg
                is_oei_ok = (oei_margin >= 0.0)

                metrics.append(ComplianceMetric(
                    name="Hospital Helipad OEI Category A Margin",
                    actual_value=mb_start.gross_weight_kg,
                    limit_value=max_oei,
                    margin=oei_margin,
                    unit="kg",
                    is_compliant=is_oei_ok,
                    notes=f"Elevated Helipad Cat A: PA={dest_pa:.0f} ft, OAT={ambient_temp_c:.1f} C"
                ))

                audit_trace.append(
                    f"Hospital Helipad Cat A Check: Max OEI={max_oei:.1f} kg vs Current GW={mb_start.gross_weight_kg:.1f} kg (Margin: {oei_margin:+.1f} kg)"
                )

                if not is_oei_ok:
                    audit_trace.append(f"OEI HELIPAD DEFICIT: Cannot meet Cat A balked landing gradient! Deficit is {abs(oei_margin):.1f} kg.")
                    if overall_verdict != DispatchVerdict.NO_GO:
                        overall_verdict = DispatchVerdict.NO_GO
                        primary_limiting_factor = LimitingFactor.OEI_CAT_A_HELIPAD_DEFICIT
                        limiting_description = (
                            f"Elevated Helipad OEI Cat A Deficit: Gross weight ({mb_start.gross_weight_kg:.1f} kg) "
                            f"exceeds Category A single-engine climb capability ({max_oei:.1f} kg) by {abs(oei_margin):.1f} kg."
                        )
                    mitigation_actions.append(
                        f"Divert to ground-level helipad or reduce aircraft weight by {abs(oei_margin):.1f} kg."
                    )

            # Assemble Leg Report
            leg_rep = MissionLegReport(
                leg_index=leg_idx,
                name=leg.name,
                duration_min=total_leg_duration_min,
                distance_nm=leg_dist_nm,
                start_gw_kg=mb_start.gross_weight_kg,
                end_gw_kg=mb_end.gross_weight_kg,
                fuel_burn_kg=leg_burn_kg,
                remaining_fuel_kg=current_fuel_kg,
                cg_long_mm=mb_start.cg_long_mm,
                cg_lat_mm=mb_start.cg_lat_mm,
                tas_kts=tas_kts,
                gs_kts=gs_kts,
                headwind_kts=hw_kts,
                crosswind_kts=xw_kts,
                pa_ft=dest_pa,
                oat_c=ambient_temp_c,
                da_ft=dest_da,
                max_allowable_gw_kg=max_allowable_gw,
                hover_margin_kg=hover_margin,
                metrics=metrics,
            )
            legs_report.append(leg_rep)

        # 3. Post-Mission Fuel Reserve Evaluation (FAR 135.209)
        ref_cruise_pa = pressure_altitude_ft(2000.0, altimeter_inhg=altimeter_inhg)
        ref_cruise_ff = self.rfm_engine.get_cruise_fuel_flow_kg_per_hr(ref_cruise_pa, ambient_temp_c, mb_end.gross_weight_kg)
        
        reserve_eval = evaluate_fuel_reserves(
            remaining_fuel_kg=current_fuel_kg,
            cruise_fuel_flow_kg_hr=ref_cruise_ff,
            is_vfr=mission.is_vfr,
            company_policy_min=30.0,
            faa_vfr_min=20.0,
            faa_ifr_min=30.0,
        )
        audit_trace.append(f"\n--- Final Fuel Reserve Audit ---")
        audit_trace.append(reserve_eval.detail_message)

        if not reserve_eval.is_faa_legal:
            if overall_verdict != DispatchVerdict.NO_GO:
                overall_verdict = DispatchVerdict.NO_GO
                primary_limiting_factor = reserve_eval.limiting_factor
                limiting_description = reserve_eval.detail_message
            mitigation_actions.append(
                f"Add at least {reserve_eval.required_reserve_kg - current_fuel_kg:.1f} kg fuel or plan an en-route fuel stop."
            )
        elif reserve_eval.verdict == DispatchVerdict.CAUTION:
            if overall_verdict == DispatchVerdict.GO:
                overall_verdict = DispatchVerdict.CAUTION
                primary_limiting_factor = LimitingFactor.NONE
                limiting_description = reserve_eval.detail_message
            mitigation_actions.append("Consider precautionary fuel top-off before launch to maintain 30-minute company safety reserve.")

        if overall_verdict == DispatchVerdict.GO:
            limiting_description = "All FAR Part 135 Subpart L regulations, RFM hover ceilings, CG envelopes, and fuel reserves satisfied."

        return DispatchEvaluation(
            verdict=overall_verdict,
            limiting_factor=primary_limiting_factor,
            limiting_description=limiting_description,
            timestamp_utc=timestamp_utc,
            tail_number=mission.tail_number,
            airframe_type=self.airframe.model_name,
            mission_id=mission.mission_id,
            total_distance_nm=total_distance_nm,
            total_flight_time_min=total_flight_time_min,
            total_fuel_burn_kg=total_fuel_burn_kg,
            landing_reserve_fuel_kg=current_fuel_kg,
            required_reserve_fuel_kg=reserve_eval.required_reserve_kg,
            reserve_margin_kg=reserve_eval.reserve_margin_kg,
            reserve_margin_minutes=reserve_eval.reserve_margin_min,
            legs=legs_report,
            mitigation_actions=mitigation_actions,
            audit_trace=audit_trace,
        )
