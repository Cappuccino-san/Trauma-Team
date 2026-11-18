"""Comprehensive Edge-Case & Boundary Tests for TraumaTeam Reference Engine.

Covers:
1. BELL429 full end-to-end mission evaluation via TraumaTeamDispatchEngine (both explicit airframe_id and station remapping).
2. evaluate_terrain_and_obstacles unit tests (asymmetric waypoint terrain vs obstacle height, day vs night, mountainous vs non-mountainous, TERRAIN_OBSTACLE_CLEARANCE_VIOLATION).
3. evaluate_corridor_weather and interpolate_winds_aloft (SEVERE/EXTREME SIGMET hazard NO-GO, intermediate METAR worst-case ceiling/vis, and 350/010 wind direction vector wrap-around).
4. OEI_CAT_A_HELIPAD_DEFICIT at high-altitude hospital helipad.
5. UNFLYABLE_WIND when crosswind exceeds TAS or headwind >= TAS.
6. LONGITUDINAL_CG_FORWARD_EXCEEDED, LONGITUDINAL_CG_AFT_EXCEEDED, LATERAL_CG_LEFT_EXCEEDED, and MIN_WEIGHT_VIOLATION.
7. IFR_RESERVE_VIOLATION (is_vfr=False), IFR CAUTION message formatting, and mid-mission fuel exhaustion.
"""

import pytest
from traumateam.dispatch.decision_engine import TraumaTeamDispatchEngine
from traumateam.core.kinematics import WindVector, solve_wind_triangle
from traumateam.models.mission import Waypoint, PayloadItem, LegSpec, LegType, MissionProfile
from traumateam.models.weather import (
    CorridorWeather,
    METARObservation,
    WindsAloftLayer,
    WeatherHazardAdvisory,
    HazardType,
    HazardSeverity,
    FlightRuleCategory,
)
from traumateam.models.verdict import DispatchVerdict, LimitingFactor
from traumateam.performance.airframes.h145 import get_h145_rfm_spec
from traumateam.performance.airframes.bell429 import get_bell429_rfm_spec
from traumateam.performance.mass_balance import compute_mass_and_balance
from traumateam.regulations.fuel_reserves import evaluate_fuel_reserves
from traumateam.weather_terrain.terrain_clearance import evaluate_terrain_and_obstacles
from traumateam.weather_terrain.weather_eval import evaluate_corridor_weather, interpolate_winds_aloft


@pytest.fixture
def dispatch_engine():
    return TraumaTeamDispatchEngine()


def _nominal_metar(station_id: str = "KAPA", ceil: float = 5000.0, vis: float = 10.0) -> METARObservation:
    return METARObservation(
        station_id=station_id,
        observation_time_utc="1200Z",
        wind_dir_deg=180.0,
        wind_speed_kts=8.0,
        visibility_sm=vis,
        ceiling_agl_ft=ceil,
        temperature_c=15.0,
        dewpoint_c=5.0,
        altimeter_inhg=29.92,
        flight_category=FlightRuleCategory.VFR,
    )


def test_bell429_end_to_end_dispatch_evaluation(dispatch_engine):
    """Verifies Bell 429 GlobalRanger evaluates end-to-end without KeyError both with default create_standard_hems_mission and explicit airframe_id."""
    base = Waypoint(identifier="KAPA", name="Centennial Base", lat=39.5701, lon=-104.8493, elevation_ft=5885.0)
    scene = Waypoint(identifier="SCENE", name="I-25 Scene", lat=39.6500, lon=-104.8800, elevation_ft=5600.0)
    hospital = Waypoint(
        identifier="SWED",
        name="Swedish Helipad",
        lat=39.6536,
        lon=-104.9781,
        elevation_ft=5350.0,
        is_elevated_helipad=True,
    )
    weather = CorridorWeather(departure_metar=_nominal_metar("KAPA"))

    # 1. Mission created with default airframe_id ("H145") and H145 crew station keys, then evaluated on BELL429
    mission_default = dispatch_engine.create_standard_hems_mission(
        mission_id="BELL429_E2E_REMAP",
        tail_number="N429TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hospital,
        initial_fuel_kg=320.0,
        flight_crew_weights={
            "PILOT_RH": 82.0,
            "CREW_ATTENDANT_AFT_LH": 75.0,
        },
        patient_weight_kg=78.0,
        medical_gear_weight_kg=45.0,
        aft_cargo_weight_kg=25.0,
        cruise_alt_msl_ft=6800.0,
        scene_hover_oge_min=3.0,
        is_day=True,
        is_mountainous=False,
        is_local_flying_area=True,
    )
    res_remap = dispatch_engine.evaluate_mission(
        airframe_type="BELL429",
        mission=mission_default,
        weather=weather,
        altimeter_inhg=29.92,
        ambient_temp_c=15.0,
    )
    assert res_remap.verdict == DispatchVerdict.GO
    assert res_remap.limiting_factor == LimitingFactor.NONE
    assert res_remap.airframe_type == "Bell 429 GlobalRanger"
    for leg in res_remap.legs:
        assert 3120.0 <= leg.cg_long_mm <= 3480.0

    # 2. Mission created with explicit airframe_id="BELL429"
    mission_explicit = dispatch_engine.create_standard_hems_mission(
        mission_id="BELL429_E2E_EXPLICIT",
        tail_number="N429TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hospital,
        initial_fuel_kg=320.0,
        flight_crew_weights={
            "PILOT_RH": 82.0,
            "CREW_ATTENDANT_LH": 75.0,
        },
        patient_weight_kg=78.0,
        medical_gear_weight_kg=45.0,
        aft_cargo_weight_kg=25.0,
        cruise_alt_msl_ft=6800.0,
        scene_hover_oge_min=3.0,
        airframe_id="BELL429",
    )
    res_explicit = dispatch_engine.evaluate_mission(
        airframe_type="BELL429",
        mission=mission_explicit,
        weather=weather,
    )
    assert res_explicit.verdict == DispatchVerdict.GO


def test_terrain_and_obstacles_asymmetric_waypoints_and_modes():
    """Verifies evaluate_terrain_and_obstacles computes obstacle MSL per waypoint (no cross-waypoint summation) and enforces Day/Night/Mountainous clearances."""
    high_base_no_tower = Waypoint(
        identifier="KLXV",
        name="Leadville Base",
        lat=39.22,
        lon=-106.31,
        elevation_ft=9900.0,
        obstacle_height_ft=0.0,
    )
    low_hospital_with_crane = Waypoint(
        identifier="KDEN_HOSP",
        name="Lowland Hospital",
        lat=39.75,
        lon=-104.99,
        elevation_ft=5000.0,
        obstacle_height_ft=250.0,
    )

    # Highest true obstacle is max(9900 + 0, 5000 + 250) = 9900 ft MSL (NOT 9900 + 250 = 10150 ft MSL!)
    # Day clearance requires +500 ft -> min safe altitude = 10400 ft MSL.
    rep_day_ok = evaluate_terrain_and_obstacles(
        origin=high_base_no_tower,
        destination=low_hospital_with_crane,
        planned_cruise_alt_msl_ft=10450.0,
        is_day=True,
        is_mountainous=True,
    )
    assert rep_day_ok.is_cleared is True
    assert rep_day_ok.verdict == DispatchVerdict.GO
    assert rep_day_ok.min_safe_altitude_msl_ft == 10400.0
    assert rep_day_ok.margin_ft == 50.0

    # Night mountainous requires +1000 ft -> min safe altitude = 10900 ft MSL.
    # At 10500 ft MSL, should fail with TERRAIN_OBSTACLE_CLEARANCE_VIOLATION.
    rep_night_mtn_fail = evaluate_terrain_and_obstacles(
        origin=high_base_no_tower,
        destination=low_hospital_with_crane,
        planned_cruise_alt_msl_ft=10500.0,
        is_day=False,
        is_mountainous=True,
    )
    assert rep_night_mtn_fail.is_cleared is False
    assert rep_night_mtn_fail.verdict == DispatchVerdict.NO_GO
    assert rep_night_mtn_fail.limiting_factor == LimitingFactor.TERRAIN_OBSTACLE_CLEARANCE_VIOLATION
    assert rep_night_mtn_fail.min_safe_altitude_msl_ft == 10900.0
    assert rep_night_mtn_fail.margin_ft == -400.0

    # Night non-mountainous requires +500 ft; en-route ridge 6000 ft + 300 ft tower = 6300 ft MSL -> 6800 ft min safe
    rep_enroute = evaluate_terrain_and_obstacles(
        origin=low_hospital_with_crane,
        destination=low_hospital_with_crane,
        planned_cruise_alt_msl_ft=6700.0,
        is_day=False,
        is_mountainous=False,
        max_terrain_along_leg_ft=6000.0,
        max_obstacle_along_leg_ft=300.0,
    )
    assert rep_enroute.is_cleared is False
    assert rep_enroute.limiting_factor == LimitingFactor.TERRAIN_OBSTACLE_CLEARANCE_VIOLATION
    assert rep_enroute.min_safe_altitude_msl_ft == 6800.0


def test_corridor_weather_hazards_intermediates_and_winds_wrap_around():
    """Verifies SEVERE/EXTREME SIGMET hazard override, intermediate METAR worst-case selection, and 350/010 wind wrap-around."""
    # 1. Wind vector wrap-around across North (350 deg at 3000 ft, 010 deg at 6000 ft, both 20 kt)
    layers = [
        WindsAloftLayer(altitude_msl_ft=3000.0, direction_deg=350.0, speed_kts=20.0, temperature_c=10.0),
        WindsAloftLayer(altitude_msl_ft=6000.0, direction_deg=10.0, speed_kts=20.0, temperature_c=4.0),
    ]
    mid_wind = interpolate_winds_aloft(4500.0, layers)
    # Midpoint direction must be ~0/360 deg (north), NOT 180 deg (south)!
    assert mid_wind.direction_deg < 2.0 or mid_wind.direction_deg > 358.0
    assert pytest.approx(mid_wind.speed_kts, rel=0.05) == 20.0 * 0.9848077

    # 2. Intermediate station with worst ceiling/visibility triggers NO-GO even when dep & dest are VFR
    corridor_inter = CorridorWeather(
        departure_metar=_nominal_metar("KAPA", ceil=6000.0, vis=10.0),
        destination_metar=_nominal_metar("KLXV", ceil=5000.0, vis=10.0),
        intermediate_metars=[_nominal_metar("KDEN", ceil=450.0, vis=1.5)],
    )
    verdict_inter = evaluate_corridor_weather(
        corridor=corridor_inter,
        is_day=True,
        is_mountainous=True,
        is_local=False,
        nvg_or_htaws_equipped=True,
    )
    assert verdict_inter.is_flyable is False
    assert verdict_inter.verdict == DispatchVerdict.NO_GO
    assert verdict_inter.worst_ceiling_ft == 450.0
    assert verdict_inter.worst_vis_sm == 1.5
    assert verdict_inter.limiting_factor == LimitingFactor.FAR_135_615_CEILING_BELOW_MINIMUMS

    # 3. Active SEVERE SIGMET overrides VFR ceiling/visibility with SEVERE_WEATHER_HAZARD
    corridor_sigmet = CorridorWeather(
        departure_metar=_nominal_metar("KBOS", ceil=8000.0, vis=10.0),
        destination_metar=_nominal_metar("KPWM", ceil=8000.0, vis=10.0),
        active_hazards=[
            WeatherHazardAdvisory(
                advisory_id="SIGE-22E",
                hazard_type=HazardType.SEVERE_ICING,
                severity=HazardSeverity.SEVERE,
                min_altitude_ft=2000.0,
                max_altitude_ft=14000.0,
                summary="Severe Icing in freezing rain",
                active=True,
            )
        ],
    )
    verdict_sigmet = evaluate_corridor_weather(
        corridor=corridor_sigmet,
        is_day=True,
        is_mountainous=False,
        is_local=True,
        nvg_or_htaws_equipped=True,
    )
    assert verdict_sigmet.is_flyable is False
    assert verdict_sigmet.verdict == DispatchVerdict.NO_GO
    assert verdict_sigmet.limiting_factor == LimitingFactor.SEVERE_WEATHER_HAZARD
    assert "SEVERE_ICING: Severe Icing" in (verdict_sigmet.active_hazard_summary or "")


def test_oei_cat_a_helipad_deficit_at_high_altitude_hospital(dispatch_engine):
    """Verifies OEI_CAT_A_HELIPAD_DEFICIT when scene OGE hover passes at lower elevation (5500 ft) but high-altitude elevated hospital helipad (9800 ft) exceeds OEI Cat A limit."""
    base = Waypoint(identifier="KAPA", name="Centennial Base", lat=39.5701, lon=-104.8493, elevation_ft=5500.0)
    scene = Waypoint(identifier="VALLEY_SCENE", name="Valley Scene", lat=39.5800, lon=-105.1000, elevation_ft=5500.0)
    hospital = Waypoint(
        identifier="HIGH_HELIPAD",
        name="High Altitude Rooftop Helipad",
        lat=39.5772,
        lon=-106.0714,
        elevation_ft=9800.0,
        is_elevated_helipad=True,
    )
    weather = CorridorWeather(departure_metar=_nominal_metar("KAPA", ceil=8000.0, vis=10.0))

    mission = dispatch_engine.create_standard_hems_mission(
        mission_id="HEMS_OEI_CAT_A_DEFICIT",
        tail_number="N145TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hospital,
        initial_fuel_kg=500.0,
        flight_crew_weights={
            "PILOT_RH": 90.0,
            "COPILOT_LH": 85.0,
            "CREW_ATTENDANT_AFT_LH": 80.0,
            "CREW_ATTENDANT_AFT_RH": 85.0,
        },
        patient_weight_kg=100.0,
        medical_gear_weight_kg=80.0,
        cruise_alt_msl_ft=11000.0,
        scene_hover_oge_min=3.0,
        is_day=True,
        is_mountainous=True,
        is_local_flying_area=True,
    )

    result = dispatch_engine.evaluate_mission(
        airframe_type="H145",
        mission=mission,
        weather=weather,
        altimeter_inhg=29.80,
        ambient_temp_c=25.0,
    )
    assert result.verdict == DispatchVerdict.NO_GO
    assert result.limiting_factor == LimitingFactor.OEI_CAT_A_HELIPAD_DEFICIT
    assert "OEI Cat A Deficit" in result.limiting_description


def test_unflyable_wind_crosswind_and_headwind(dispatch_engine):
    """Verifies UNFLYABLE_WIND when crosswind > TAS or direct headwind >= TAS."""
    # 1. Direct unit test on solve_wind_triangle: crosswind > TAS
    res_xw = solve_wind_triangle(true_course_deg=360.0, tas_kts=120.0, wind=WindVector(direction_deg=90.0, speed_kts=130.0))
    assert res_xw.is_flyable is False
    assert "Crosswind" in (res_xw.unflyable_reason or "")

    # 2. Direct unit test on solve_wind_triangle: headwind >= TAS
    res_hw = solve_wind_triangle(true_course_deg=270.0, tas_kts=130.0, wind=WindVector(direction_deg=270.0, speed_kts=135.0))
    assert res_hw.is_flyable is False
    assert "headwind" in (res_hw.unflyable_reason or "").lower()

    # 3. Full mission integrator test triggering UNFLYABLE_WIND
    base = Waypoint(identifier="KAPA", name="Base", lat=39.5701, lon=-104.8493, elevation_ft=5885.0)
    dest = Waypoint(identifier="KDEN", name="Dest", lat=39.8561, lon=-104.6737, elevation_ft=5434.0)
    mission = MissionProfile(
        mission_id="UNFLYABLE_WIND_MISSION",
        tail_number="N145TT",
        initial_fuel_kg=400.0,
        payloads=[PayloadItem(item_id="p1", name="Pilot", station_id="PILOT_RH", weight_kg=85.0)],
        legs=[
            LegSpec(
                leg_index=0,
                name="Transit in Hurricane Wind",
                leg_type=LegType.TRANSIT,
                origin=base,
                destination=dest,
                planned_cruise_altitude_msl_ft=7000.0,
            )
        ],
    )
    weather = CorridorWeather(
        departure_metar=_nominal_metar("KAPA"),
        winds_aloft_profile=[
            WindsAloftLayer(altitude_msl_ft=7000.0, direction_deg=25.0, speed_kts=160.0, temperature_c=5.0)
        ],
    )
    eval_res = dispatch_engine.evaluate_mission("H145", mission, weather)
    assert eval_res.verdict == DispatchVerdict.NO_GO
    assert eval_res.limiting_factor == LimitingFactor.UNFLYABLE_WIND


def test_mass_balance_forward_aft_left_cg_and_min_weight():
    """Verifies LONGITUDINAL_CG_FORWARD_EXCEEDED, LONGITUDINAL_CG_AFT_EXCEEDED, LATERAL_CG_LEFT_EXCEEDED, and MIN_WEIGHT_VIOLATION."""
    h145 = get_h145_rfm_spec()

    # 1. Forward longitudinal CG exceedance: heavy forward cockpit crew with minimal fuel
    fwd_payloads = [
        PayloadItem(item_id="p1", name="Heavy Pilot", station_id="PILOT_RH", weight_kg=180.0),
        PayloadItem(item_id="p2", name="Heavy Copilot", station_id="COPILOT_LH", weight_kg=180.0),
    ]
    mb_fwd = compute_mass_and_balance(h145, fuel_kg=20.0, active_payloads=fwd_payloads)
    assert mb_fwd.is_long_cg_compliant is False
    assert mb_fwd.limiting_factor == LimitingFactor.LONGITUDINAL_CG_FORWARD_EXCEEDED
    assert mb_fwd.cg_long_mm < mb_fwd.fwd_limit_mm

    # 2. Aft longitudinal CG exceedance: heavy aft clamshell cargo & oxygen/rack with low fuel
    aft_payloads = [
        PayloadItem(item_id="a1", name="Aft Cargo", station_id="AFT_AVIONICS_CARGO", weight_kg=900.0),
        PayloadItem(item_id="a2", name="Oxygen", station_id="OXYGEN_SYSTEM", weight_kg=200.0),
    ]
    mb_aft = compute_mass_and_balance(h145, fuel_kg=300.0, active_payloads=aft_payloads)
    assert mb_aft.is_long_cg_compliant is False
    assert mb_aft.limiting_factor == LimitingFactor.LONGITUDINAL_CG_AFT_EXCEEDED
    assert mb_aft.cg_long_mm > mb_aft.aft_limit_mm

    # 3. Left lateral CG exceedance: heavy LH stations (Copilot LH -360mm, Attendant LH -420mm, Litter 1 -280mm, Oxygen -350mm)
    left_payloads = [
        PayloadItem(item_id="l1", name="Copilot LH", station_id="COPILOT_LH", weight_kg=160.0),
        PayloadItem(item_id="l2", name="Attendant LH", station_id="CREW_ATTENDANT_AFT_LH", weight_kg=200.0),
        PayloadItem(item_id="l3", name="Primary Litter", station_id="PRIMARY_PATIENT_LITTER", weight_kg=250.0),
        PayloadItem(item_id="l4", name="Oxygen", station_id="OXYGEN_SYSTEM", weight_kg=120.0),
    ]
    mb_left = compute_mass_and_balance(h145, fuel_kg=100.0, active_payloads=left_payloads)
    assert mb_left.is_lat_cg_compliant is False
    assert mb_left.limiting_factor == LimitingFactor.LATERAL_CG_LEFT_EXCEEDED
    assert mb_left.cg_lat_mm < mb_left.left_limit_mm

    # 4. Minimum flight weight violation: airframe modified with high min_flight_weight_kg
    custom_spec = h145.model_copy(update={"min_flight_weight_kg": 2300.0})
    mb_min = compute_mass_and_balance(custom_spec, fuel_kg=10.0, active_payloads=[])
    assert mb_min.is_gw_compliant is False
    assert mb_min.limiting_factor == LimitingFactor.MIN_WEIGHT_VIOLATION
    assert mb_min.gross_weight_kg < 2300.0


def test_ifr_fuel_reserves_and_mid_mission_fuel_exhaustion(dispatch_engine):
    """Verifies IFR_RESERVE_VIOLATION (is_vfr=False), dynamic statutory minute formatting, and mid-mission fuel starvation."""
    # 1. Direct evaluate_fuel_reserves with is_vfr=False (30-min statutory minimum = 120 kg at 240 kg/hr)
    rep_ifr_nogo = evaluate_fuel_reserves(
        remaining_fuel_kg=100.0,  # 25 min < 30 min IFR statutory
        cruise_fuel_flow_kg_hr=240.0,
        is_vfr=False,
        company_policy_min=45.0,
        faa_ifr_min=30.0,
    )
    assert rep_ifr_nogo.is_faa_legal is False
    assert rep_ifr_nogo.verdict == DispatchVerdict.NO_GO
    assert rep_ifr_nogo.limiting_factor == LimitingFactor.IFR_RESERVE_VIOLATION
    assert "30-minute statutory reserve" in rep_ifr_nogo.detail_message

    # 2. IFR CAUTION (between 30-min IFR statutory and 45-min company policy) formats "30-min minimum"
    rep_ifr_caution = evaluate_fuel_reserves(
        remaining_fuel_kg=140.0,  # 35 min
        cruise_fuel_flow_kg_hr=240.0,
        is_vfr=False,
        company_policy_min=45.0,
        faa_ifr_min=30.0,
    )
    assert rep_ifr_caution.is_faa_legal is True
    assert rep_ifr_caution.verdict == DispatchVerdict.CAUTION
    assert "30-min minimum" in rep_ifr_caution.detail_message

    # 3. Mid-mission fuel exhaustion under IFR (is_vfr=False) triggers IFR_RESERVE_VIOLATION and negative landing reserve
    base = Waypoint(identifier="KAPA", name="Base", lat=39.5701, lon=-104.8493, elevation_ft=5885.0)
    scene = Waypoint(identifier="SCENE", name="Scene", lat=39.8561, lon=-103.5000, elevation_ft=4500.0)
    hospital = Waypoint(identifier="HOSP", name="Hospital", lat=39.7420, lon=-104.8390, elevation_ft=5380.0)
    mission = dispatch_engine.create_standard_hems_mission(
        mission_id="IFR_EXHAUSTION",
        tail_number="N145TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hospital,
        initial_fuel_kg=80.0,  # Starves mid-mission
        flight_crew_weights={"PILOT_RH": 85.0, "COPILOT_LH": 80.0},
        cruise_alt_msl_ft=6500.0,
    )
    mission = mission.model_copy(update={"is_vfr": False})
    weather = CorridorWeather(departure_metar=_nominal_metar("KAPA"))
    res = dispatch_engine.evaluate_mission("H145", mission, weather)
    assert res.verdict == DispatchVerdict.NO_GO
    assert res.limiting_factor == LimitingFactor.IFR_RESERVE_VIOLATION
    assert res.landing_reserve_fuel_kg < 0.0


def test_nan_inf_and_boundary_safety_guards(dispatch_engine):
    """Verifies NaN/Inf/non-physical guards across kinematics, atmosphere, fuel reserves, and empty-leg missions."""
    from traumateam.core.atmosphere import pressure_altitude_ft, density_altitude_ft, speed_of_sound_knots

    # 1. Wind triangle rejects NaN, Inf, negative wind speed, and exact crosswind == TAS
    for bad_res in [
        solve_wind_triangle(180.0, float("nan"), WindVector(180.0, 10.0)),
        solve_wind_triangle(float("nan"), 130.0, WindVector(180.0, 10.0)),
        solve_wind_triangle(180.0, 130.0, WindVector(float("inf"), 10.0)),
        solve_wind_triangle(180.0, 130.0, WindVector(180.0, -5.0)),
        solve_wind_triangle(360.0, 120.0, WindVector(90.0, 120.0)),
    ]:
        assert bad_res.is_flyable is False
        assert bad_res.groundspeed_kts == 0.0

    # 2. Atmosphere guards NaN/Inf and non-physical pressures/temperatures
    with pytest.raises(ValueError):
        pressure_altitude_ft(float("nan"), altimeter_inhg=29.92)
    with pytest.raises(ValueError):
        pressure_altitude_ft(5000.0, altimeter_inhg=0.0)
    with pytest.raises(ValueError):
        density_altitude_ft(5000.0, oat_c=float("nan"))
    with pytest.raises(ValueError):
        density_altitude_ft(5000.0, oat_c=-273.15)
    with pytest.raises(ValueError):
        speed_of_sound_knots(-273.15)

    # 3. Fuel reserves fail closed on NaN or non-positive fuel flow
    bad_fuel_rep = evaluate_fuel_reserves(remaining_fuel_kg=150.0, cruise_fuel_flow_kg_hr=0.0)
    assert bad_fuel_rep.is_faa_legal is False
    assert bad_fuel_rep.verdict == DispatchVerdict.NO_GO
    bad_nan_rep = evaluate_fuel_reserves(remaining_fuel_kg=float("nan"), cruise_fuel_flow_kg_hr=220.0)
    assert bad_nan_rep.is_faa_legal is False
    assert bad_nan_rep.verdict == DispatchVerdict.NO_GO

    # 4. Mission integrator rejects NaN initial fuel and handles empty legs without UnboundLocalError
    weather = CorridorWeather(departure_metar=_nominal_metar("KAPA"))
    nan_fuel_mission = MissionProfile(
        mission_id="NAN_FUEL",
        tail_number="N145TT",
        initial_fuel_kg=float("nan"),
        payloads=[],
        legs=[],
    )
    with pytest.raises(ValueError, match="Invalid initial fuel weight"):
        dispatch_engine.evaluate_mission("H145", nan_fuel_mission, weather)

    empty_legs_mission = MissionProfile(
        mission_id="EMPTY_LEGS",
        tail_number="N145TT",
        initial_fuel_kg=300.0,
        payloads=[PayloadItem(item_id="p1", name="Pilot", station_id="PILOT_RH", weight_kg=85.0)],
        legs=[],
    )
    empty_res = dispatch_engine.evaluate_mission("H145", empty_legs_mission, weather)
    assert empty_res.verdict == DispatchVerdict.GO
    assert empty_res.legs == []
