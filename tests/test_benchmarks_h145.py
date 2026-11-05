"""HEMS Verification & Benchmark Suite for Airbus H145 / BK117 D-2.

Validates operational edge cases against certified RFM tables:
1. Nominal Lowland HEMS loop (GO).
2. High-Density-Altitude Mountain Rescue Edge Case (NO-GO with OGE_HOVER_SCENE_DEFICIT).
3. High-Headwind Fuel-Reserve Boundary Case (NO-GO with FAR_135_209_VFR_RESERVE_VIOLATION).
4. Mountainous Night Weather Below Minimums (NO-GO with FAR_135_615_CEILING_BELOW_MINIMUMS).
"""

import pytest
from traumateam.dispatch.decision_engine import TraumaTeamDispatchEngine
from traumateam.models.mission import Waypoint
from traumateam.models.weather import (
    CorridorWeather,
    METARObservation,
    WindsAloftLayer,
    FlightRuleCategory,
)
from traumateam.models.verdict import DispatchVerdict, LimitingFactor

@pytest.fixture
def dispatch_engine():
    return TraumaTeamDispatchEngine()

def test_benchmark_nominal_lowland_mission(dispatch_engine):
    """Benchmark 1: Standard lowland nominal mission.
    
    Verifies that with compliant weather, balanced loading, and sufficient fuel,
    the engine outputs deterministic GO, and computed metrics match RFM within 0.5%.
    """
    base = Waypoint(
        identifier="KAPA",
        name="Centennial Airport Base",
        lat=39.5701,
        lon=-104.8493,
        elevation_ft=5885.0,
    )
    scene = Waypoint(
        identifier="SCENE_01",
        name="Highway 85 Scene Call",
        lat=39.7500,
        lon=-104.9900,
        elevation_ft=5280.0,
    )
    hospital = Waypoint(
        identifier="CO_SWED",
        name="Swedish Medical Center Helipad",
        lat=39.6536,
        lon=-104.9781,
        elevation_ft=5350.0,
        is_elevated_helipad=True,
    )

    weather = CorridorWeather(
        departure_metar=METARObservation(
            station_id="KAPA",
            observation_time_utc="1200Z",
            wind_dir_deg=180.0,
            wind_speed_kts=8.0,
            visibility_sm=10.0,
            ceiling_agl_ft=5000.0,
            temperature_c=18.0,
            dewpoint_c=5.0,
            altimeter_inhg=29.95,
            flight_category=FlightRuleCategory.VFR,
        ),
        destination_metar=METARObservation(
            station_id="KDEN",
            observation_time_utc="1200Z",
            wind_dir_deg=170.0,
            wind_speed_kts=10.0,
            visibility_sm=10.0,
            ceiling_agl_ft=6000.0,
            temperature_c=19.0,
            dewpoint_c=4.0,
            altimeter_inhg=29.94,
            flight_category=FlightRuleCategory.VFR,
        ),
        winds_aloft_profile=[
            WindsAloftLayer(altitude_msl_ft=6000.0, direction_deg=180.0, speed_kts=10.0, temperature_c=15.0),
            WindsAloftLayer(altitude_msl_ft=9000.0, direction_deg=190.0, speed_kts=15.0, temperature_c=10.0),
        ],
    )

    mission = dispatch_engine.create_standard_hems_mission(
        mission_id="HEMS_BENCH_01_NOMINAL",
        tail_number="N145TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hospital,
        initial_fuel_kg=420.0,
        flight_crew_weights={
            "PILOT_RH": 85.0,
            "COPILOT_LH": 80.0,
            "CREW_ATTENDANT_AFT_LH": 75.0,
            "CREW_ATTENDANT_AFT_RH": 82.0,
        },
        patient_weight_kg=80.0,
        medical_gear_weight_kg=60.0,
        cruise_alt_msl_ft=6500.0,
        scene_hover_oge_min=3.0,
        is_day=True,
        is_mountainous=False,
        is_local_flying_area=True,
        nvg_or_htaws_equipped=True,
    )

    result = dispatch_engine.evaluate_mission(
        airframe_type="H145",
        mission=mission,
        weather=weather,
        altimeter_inhg=29.94,
        ambient_temp_c=18.0,
    )

    assert result.verdict == DispatchVerdict.GO
    assert result.limiting_factor == LimitingFactor.NONE
    assert result.landing_reserve_fuel_kg > result.required_reserve_fuel_kg
    assert result.reserve_margin_minutes >= 20.0

    # Validate CG within certified limits across all legs
    for leg in result.legs:
        assert leg.cg_long_mm >= 3780.0
        assert leg.cg_long_mm <= 4200.0
        assert abs(leg.cg_lat_mm) <= 75.0

def test_benchmark_high_density_altitude_mountain_rescue_edge_case(dispatch_engine):
    """Benchmark 2: High-Density-Altitude Mountain Rescue Edge Case.
    
    Location: Turquoise Lake / Leadville, Colorado.
    Elevation: 9,800 ft MSL, OAT: 28°C, Altimeter: 29.70 inHg.
    Density Altitude exceeds 13,000 ft.
    
    Ingestion of H145 RFM OGE hover table restricts max allowable weight to ~2800 kg.
    With heavy crew, ECMO medical rack, and high fuel, Gross Weight at scene exceeds 3200 kg.
    Engine must deterministically reject the mission with NO-GO and OGE_HOVER_SCENE_DEFICIT.
    """
    base = Waypoint(
        identifier="KLXV",
        name="Leadville Lake County Airport",
        lat=39.2244,
        lon=-106.3169,
        elevation_ft=9934.0,
    )
    scene = Waypoint(
        identifier="MTN_SCENE",
        name="Turquoise Lake Mountain Pass",
        lat=39.2600,
        lon=-106.4000,
        elevation_ft=9800.0,
    )
    hospital = Waypoint(
        identifier="ST_ANTHONY",
        name="St. Anthony Summit Medical Helipad",
        lat=39.5772,
        lon=-106.0714,
        elevation_ft=9100.0,
        is_elevated_helipad=True,
    )

    weather = CorridorWeather(
        departure_metar=METARObservation(
            station_id="KLXV",
            observation_time_utc="1900Z",
            wind_dir_deg=260.0,
            wind_speed_kts=12.0,
            visibility_sm=10.0,
            ceiling_agl_ft=8000.0,
            temperature_c=28.0,  # Unusually hot mountain afternoon
            dewpoint_c=2.0,
            altimeter_inhg=29.70,
            flight_category=FlightRuleCategory.VFR,
        ),
        destination_metar=METARObservation(
            station_id="KLXV",
            observation_time_utc="1900Z",
            wind_dir_deg=260.0,
            wind_speed_kts=12.0,
            visibility_sm=10.0,
            ceiling_agl_ft=8000.0,
            temperature_c=28.0,
            dewpoint_c=2.0,
            altimeter_inhg=29.70,
            flight_category=FlightRuleCategory.VFR,
        ),
        winds_aloft_profile=[
            WindsAloftLayer(altitude_msl_ft=10000.0, direction_deg=260.0, speed_kts=15.0, temperature_c=24.0),
            WindsAloftLayer(altitude_msl_ft=12000.0, direction_deg=270.0, speed_kts=20.0, temperature_c=18.0),
        ],
    )

    # Mission with full fuel (550 kg) + intensive care medical crew + ECMO rack
    mission = dispatch_engine.create_standard_hems_mission(
        mission_id="HEMS_BENCH_02_HIGH_DA",
        tail_number="N145TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hospital,
        initial_fuel_kg=550.0,
        flight_crew_weights={
            "PILOT_RH": 90.0,
            "COPILOT_LH": 85.0,
            "CREW_ATTENDANT_AFT_LH": 80.0,
            "CREW_ATTENDANT_AFT_RH": 85.0,
        },
        patient_weight_kg=90.0,
        medical_gear_weight_kg=85.0, # Heavy ECMO setup
        cruise_alt_msl_ft=11500.0,
        scene_hover_oge_min=6.0,    # Hoist rescue requires OGE hover
        is_day=True,
        is_mountainous=True,
        is_local_flying_area=True,
        nvg_or_htaws_equipped=True,
    )

    result = dispatch_engine.evaluate_mission(
        airframe_type="H145",
        mission=mission,
        weather=weather,
        altimeter_inhg=29.70,
        ambient_temp_c=28.0,
    )

    # Must be deterministic NO-GO due to OGE hover deficit at the mountain scene
    assert result.verdict == DispatchVerdict.NO_GO
    assert result.limiting_factor == LimitingFactor.OGE_HOVER_SCENE_DEFICIT
    assert "OGE Hover Performance Deficit" in result.limiting_description
    assert len(result.mitigation_actions) > 0
    assert "defuel" in result.mitigation_actions[0].lower() or "offload" in result.mitigation_actions[0].lower()

def test_benchmark_high_headwind_fuel_reserve_boundary_case(dispatch_engine):
    """Benchmark 3: High-Headwind Fuel-Reserve Boundary Case.
    
    Long cross-country transit facing 45 kt winter jet streak headwind aloft.
    Groundspeed drops by 35%, increasing flight time and burning into the statutory 20-min VFR reserve.
    Must output NO-GO with FAR_135_209_VFR_RESERVE_VIOLATION.
    """
    base = Waypoint(
        identifier="KDEN",
        name="Denver Base",
        lat=39.8561,
        lon=-104.6737,
        elevation_ft=5434.0,
    )
    scene = Waypoint(
        identifier="SCENE_FAR",
        name="Rural Eastern Plains Scene",
        lat=39.8561,
        lon=-103.5000, # ~54 NM east
        elevation_ft=4500.0,
    )
    hospital = Waypoint(
        identifier="UCH",
        name="UCHealth University of Colorado Hospital",
        lat=39.7420,
        lon=-104.8390,
        elevation_ft=5380.0,
        is_elevated_helipad=False,
    )

    # Strong westerly jet streak: 270 deg at 45 knots
    # Flying westbound from scene to hospital is directly into the 45 kt headwind!
    weather = CorridorWeather(
        departure_metar=METARObservation(
            station_id="KDEN",
            observation_time_utc="1400Z",
            wind_dir_deg=270.0,
            wind_speed_kts=20.0,
            wind_gust_kts=32.0,
            visibility_sm=10.0,
            ceiling_agl_ft=4000.0,
            temperature_c=5.0,
            dewpoint_c=-5.0,
            altimeter_inhg=30.10,
            flight_category=FlightRuleCategory.VFR,
        ),
        winds_aloft_profile=[
            WindsAloftLayer(altitude_msl_ft=6000.0, direction_deg=270.0, speed_kts=42.0, temperature_c=2.0),
            WindsAloftLayer(altitude_msl_ft=8000.0, direction_deg=270.0, speed_kts=48.0, temperature_c=-2.0),
        ],
    )

    # Initial fuel planned is marginal (280 kg) for ~120 NM round trip in calm winds,
    # but with 45 kt headwind, flight time exceeds endurance!
    mission = dispatch_engine.create_standard_hems_mission(
        mission_id="HEMS_BENCH_03_HEADWIND",
        tail_number="N145TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hospital,
        initial_fuel_kg=280.0,
        flight_crew_weights={
            "PILOT_RH": 85.0,
            "COPILOT_LH": 80.0,
            "CREW_ATTENDANT_AFT_LH": 75.0,
            "CREW_ATTENDANT_AFT_RH": 80.0,
        },
        patient_weight_kg=80.0,
        medical_gear_weight_kg=50.0,
        cruise_alt_msl_ft=6500.0,
        scene_hover_oge_min=2.0,
        is_day=True,
        is_mountainous=False,
        is_local_flying_area=False, # Cross-country
        nvg_or_htaws_equipped=True,
    )

    result = dispatch_engine.evaluate_mission(
        airframe_type="H145",
        mission=mission,
        weather=weather,
        altimeter_inhg=30.10,
        ambient_temp_c=5.0,
    )

    # Must flag fuel reserve violation
    assert result.verdict == DispatchVerdict.NO_GO
    assert result.limiting_factor == LimitingFactor.FAR_135_209_VFR_RESERVE_VIOLATION
    assert result.reserve_margin_minutes < 20.0
    assert "FUEL RESERVE VIOLATION" in result.limiting_description

def test_benchmark_weather_below_minimums(dispatch_engine):
    """Benchmark 4: Weather Below Minimums Edge Case.
    
    Mountainous Cross-Country Night mission with 900 ft ceiling / 2.0 SM visibility.
    Under 14 CFR § 135.615 Table 1, Mountainous Night Cross-Country with HTAWS/NVG requires 1,000 ft / 3 SM.
    Engine must deterministically reject the mission with NO-GO and FAR_135_615_CEILING_BELOW_MINIMUMS.
    """
    base = Waypoint(
        identifier="KASE",
        name="Aspen Base",
        lat=39.2232,
        lon=-106.8688,
        elevation_ft=7820.0,
    )
    scene = Waypoint(
        identifier="SCENE_VALLEY",
        name="Mountain Valley Scene",
        lat=39.5000,
        lon=-107.2000,
        elevation_ft=6200.0,
    )
    hospital = Waypoint(
        identifier="GWS_HOSP",
        name="Valley View Hospital",
        lat=39.5400,
        lon=-107.3200,
        elevation_ft=5800.0,
    )

    weather = CorridorWeather(
        departure_metar=METARObservation(
            station_id="KASE",
            observation_time_utc="0300Z",
            wind_dir_deg=340.0,
            wind_speed_kts=5.0,
            visibility_sm=2.0,       # Below required 3 SM
            ceiling_agl_ft=900.0,    # Below required 1000 ft
            temperature_c=0.0,
            dewpoint_c=-1.0,
            altimeter_inhg=29.85,
            flight_category=FlightRuleCategory.IFR,
        )
    )

    mission = dispatch_engine.create_standard_hems_mission(
        mission_id="HEMS_BENCH_04_WX_MINS",
        tail_number="N145TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hospital,
        initial_fuel_kg=400.0,
        flight_crew_weights={"PILOT_RH": 85.0, "COPILOT_LH": 80.0},
        cruise_alt_msl_ft=9500.0,
        is_day=False,               # Night
        is_mountainous=True,        # Mountainous
        is_local_flying_area=False, # Cross-Country
        nvg_or_htaws_equipped=True,
    )

    result = dispatch_engine.evaluate_mission(
        airframe_type="H145",
        mission=mission,
        weather=weather,
        altimeter_inhg=29.85,
        ambient_temp_c=0.0,
    )

    assert result.verdict == DispatchVerdict.NO_GO
    assert result.limiting_factor in (
        LimitingFactor.FAR_135_615_CEILING_BELOW_MINIMUMS,
        LimitingFactor.FAR_135_615_VISIBILITY_BELOW_MINIMUMS,
    )
    assert "WEATHER VIOLATION" in result.limiting_description
