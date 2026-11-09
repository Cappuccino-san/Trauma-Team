"""TraumaTeam Demonstration CLI & Multi-Scenario Benchmark Suite.

A safety-critical Helicopter Emergency Medical Services (HEMS) dispatch decision
engine simulation suite under FAA 14 CFR Part 135 Subpart L and OEM Rotorcraft
Flight Manual (RFM) physics.

Covers 5 distinct operational scenarios:
1. Nominal Lowland Dispatch (Airbus H145, Denver Metro, VMC, balanced CG) -> GO
2. High Density Altitude Mountain Rescue (Airbus H145, Leadville 3V2, DA > 12,000 ft, OGE hover deficit) -> NO-GO with defueling mitigation
3. Convective SIGMET & Ceiling Bust (Bell 429, Front Range, ceiling 400 ft below FAR 135.609 minimums) -> NO-GO with regulatory citation
4. Severe Headwind & Fuel Reserve Margin Exhaustion (Airbus H145, 45 kt adverse headwind, landing fuel below FAR 135.209 reserve) -> NO-GO with deficit calculation
5. Lateral & Longitudinal CG Asymmetry (Bell 429, external hoist & boom arm exceeding RFM envelope) -> NO-GO with envelope boundary calculation
"""

from __future__ import annotations

import argparse
import json
import sys
from dataclasses import asdict, dataclass
from typing import Any, Dict, List, Optional

from traumateam.dispatch.decision_engine import TraumaTeamDispatchEngine
from traumateam.models.mission import MissionProfile, PayloadItem, Waypoint
from traumateam.models.rfm import StationArm
from traumateam.models.verdict import DispatchEvaluation, DispatchVerdict, LimitingFactor
from traumateam.models.weather import (
    CorridorWeather,
    FlightRuleCategory,
    HazardSeverity,
    HazardType,
    METARObservation,
    WeatherHazardAdvisory,
    WindsAloftLayer,
)


# ANSI Color Codes for Professional Avionics Terminal Boards
class TerminalColors:
    RESET = "\033[0m"
    BOLD = "\033[1m"
    DIM = "\033[2m"
    UNDERLINE = "\033[4m"
    
    # Foreground colors
    BLACK = "\033[30m"
    RED = "\033[31m"
    GREEN = "\033[32m"
    YELLOW = "\033[33m"
    BLUE = "\033[34m"
    MAGENTA = "\033[35m"
    CYAN = "\033[36m"
    WHITE = "\033[37m"
    
    # Bright foreground
    BRIGHT_GREEN = "\033[92m"
    BRIGHT_YELLOW = "\033[93m"
    BRIGHT_RED = "\033[91m"
    BRIGHT_CYAN = "\033[96m"
    BRIGHT_WHITE = "\033[97m"
    
    # Backgrounds
    BG_GREEN = "\033[42m"
    BG_RED = "\033[41m"
    BG_YELLOW = "\033[43m"
    BG_BLUE = "\033[44m"


@dataclass
class ScenarioDefinition:
    scenario_id: int
    title: str
    description: str
    expected_verdict: DispatchVerdict
    expected_limiting_factor: LimitingFactor
    builder: Any


@dataclass
class ScenarioExecutionResult:
    scenario_id: int
    title: str
    expected_verdict: str
    actual_verdict: str
    expected_limiting_factor: str
    actual_limiting_factor: str
    passed: bool
    evaluation: DispatchEvaluation

    def to_dict(self) -> Dict[str, Any]:
        return {
            "scenario_id": self.scenario_id,
            "title": self.title,
            "expected_verdict": self.expected_verdict,
            "actual_verdict": self.actual_verdict,
            "expected_limiting_factor": self.expected_limiting_factor,
            "actual_limiting_factor": self.actual_limiting_factor,
            "passed": self.passed,
            "airframe_type": self.evaluation.airframe_type,
            "tail_number": self.evaluation.tail_number,
            "mission_id": self.evaluation.mission_id,
            "total_distance_nm": round(self.evaluation.total_distance_nm, 1),
            "total_flight_time_min": round(self.evaluation.total_flight_time_min, 1),
            "total_fuel_burn_kg": round(self.evaluation.total_fuel_burn_kg, 1),
            "landing_reserve_fuel_kg": round(self.evaluation.landing_reserve_fuel_kg, 1),
            "reserve_margin_minutes": round(self.evaluation.reserve_margin_minutes, 1),
            "limiting_description": self.evaluation.limiting_description,
            "mitigation_actions": self.evaluation.mitigation_actions,
        }


# ==============================================================================
# SCENARIO BUILDERS
# ==============================================================================

def build_scenario_1(engine: TraumaTeamDispatchEngine) -> tuple[str, MissionProfile, CorridorWeather, float, float]:
    """Scenario 1: Nominal Lowland Dispatch (Airbus H145, Denver Metro lowland, VMC, balanced CG).
    Outcome: GO
    """
    base = Waypoint(identifier="KAPA", name="Centennial Airport", lat=39.5701, lon=-104.8493, elevation_ft=5885.0)
    scene = Waypoint(identifier="SCENE", name="Highway 85 Trauma Scene", lat=39.7500, lon=-104.9900, elevation_ft=5280.0)
    hosp = Waypoint(identifier="SWEDISH", name="Swedish Medical Center (Elevated Helipad)", lat=39.6536, lon=-104.9781, elevation_ft=5350.0, is_elevated_helipad=True)

    wx = CorridorWeather(
        departure_metar=METARObservation(
            station_id="KAPA",
            observation_time_utc="1400Z",
            wind_dir_deg=180.0,
            wind_speed_kts=10.0,
            visibility_sm=10.0,
            ceiling_agl_ft=5000.0,
            temperature_c=18.0,
            dewpoint_c=4.0,
            altimeter_inhg=29.95,
            flight_category=FlightRuleCategory.VFR,
        ),
        winds_aloft_profile=[
            WindsAloftLayer(altitude_msl_ft=6000.0, direction_deg=180.0, speed_kts=12.0, temperature_c=15.0)
        ],
    )

    mission = engine.create_standard_hems_mission(
        mission_id="HEMS_SCENARIO_01_NOMINAL",
        tail_number="N145TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hosp,
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
        airframe_id="H145",
    )

    return "H145", mission, wx, 29.95, 18.0


def build_scenario_2(engine: TraumaTeamDispatchEngine) -> tuple[str, MissionProfile, CorridorWeather, float, float]:
    """Scenario 2: High Density Altitude Mountain Rescue (Airbus H145, Leadville Lake County 3V2 / Turquoise Lake).
    Elevation 9,934 ft MSL, ambient 28 deg C -> Density Altitude > 12,000 ft.
    Outcome: NO-GO (OGE_HOVER_SCENE_DEFICIT) with actionable defueling mitigation.
    """
    base = Waypoint(identifier="KLXV", name="Leadville Lake County Airport", lat=39.2244, lon=-106.3169, elevation_ft=9934.0)
    scene = Waypoint(identifier="PASS", name="Turquoise Lake Mountain Pass", lat=39.2600, lon=-106.4000, elevation_ft=9800.0)
    hosp = Waypoint(identifier="SUMMIT", name="St. Anthony Summit Helipad", lat=39.5772, lon=-106.0714, elevation_ft=9100.0, is_elevated_helipad=True)

    wx = CorridorWeather(
        departure_metar=METARObservation(
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
            WindsAloftLayer(altitude_msl_ft=12000.0, direction_deg=270.0, speed_kts=20.0, temperature_c=18.0)
        ],
    )

    mission = engine.create_standard_hems_mission(
        mission_id="HEMS_SCENARIO_02_HIGH_DA",
        tail_number="N145TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hosp,
        initial_fuel_kg=550.0,
        flight_crew_weights={
            "PILOT_RH": 90.0,
            "COPILOT_LH": 85.0,
            "CREW_ATTENDANT_AFT_LH": 80.0,
            "CREW_ATTENDANT_AFT_RH": 85.0,
        },
        patient_weight_kg=90.0,
        medical_gear_weight_kg=85.0,
        cruise_alt_msl_ft=11500.0,
        scene_hover_oge_min=6.0,
        is_day=True,
        is_mountainous=True,
        is_local_flying_area=True,
        nvg_or_htaws_equipped=True,
        airframe_id="H145",
    )

    return "H145", mission, wx, 29.70, 28.0


def build_scenario_3(engine: TraumaTeamDispatchEngine) -> tuple[str, MissionProfile, CorridorWeather, float, float]:
    """Scenario 3: Convective SIGMET & Ceiling Bust (Bell 429, Front Range thunderstorm corridor).
    Observed cloud ceiling 600 ft AGL is 400 ft below FAR 135.609 Table 1 mountainous cross-country minimum of 1000 ft.
    Outcome: NO-GO (FAR_135_615_CEILING_BELOW_MINIMUMS).
    """
    base = Waypoint(identifier="KBJC", name="Rocky Mountain Metropolitan Airport", lat=39.9088, lon=-105.1172, elevation_ft=5673.0)
    scene = Waypoint(identifier="NEDERLAND", name="Nederland Peak Trauma Scene", lat=39.9614, lon=-105.5108, elevation_ft=8230.0)
    hosp = Waypoint(identifier="BOULDER", name="Boulder Community Hospital Helipad", lat=40.0150, lon=-105.2705, elevation_ft=5350.0, is_elevated_helipad=False)

    wx = CorridorWeather(
        departure_metar=METARObservation(
            station_id="KBJC",
            observation_time_utc="2000Z",
            wind_dir_deg=310.0,
            wind_speed_kts=24.0,
            visibility_sm=4.0,
            ceiling_agl_ft=600.0,  # 400 ft below 1000 ft req under FAR 135.609 Table 1
            temperature_c=11.0,
            dewpoint_c=9.0,
            altimeter_inhg=29.75,
            flight_category=FlightRuleCategory.IFR,
        ),
        winds_aloft_profile=[
            WindsAloftLayer(altitude_msl_ft=9000.0, direction_deg=310.0, speed_kts=30.0, temperature_c=5.0)
        ],
        active_hazards=[
            WeatherHazardAdvisory(
                advisory_id="SIGMET_42C",
                hazard_type=HazardType.CONVECTIVE_SIGMET,
                severity=HazardSeverity.MODERATE,
                min_altitude_ft=0.0,
                max_altitude_ft=18000.0,
                summary="Convective SIGMET 42C: Active convective cell line along Front Range foothills",
                active=True,
            )
        ],
    )

    mission = engine.create_standard_hems_mission(
        mission_id="HEMS_SCENARIO_03_WX_CEILING",
        tail_number="N429TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hosp,
        initial_fuel_kg=350.0,
        flight_crew_weights={
            "PILOT_RH": 85.0,
            "CREW_ATTENDANT_LH": 75.0,
            "CREW_ATTENDANT_RH": 80.0,
        },
        patient_weight_kg=80.0,
        medical_gear_weight_kg=55.0,
        cruise_alt_msl_ft=9500.0,
        scene_hover_oge_min=4.0,
        is_day=True,
        is_mountainous=True,
        is_local_flying_area=False,  # Cross-country -> 1000 ft ceiling required
        nvg_or_htaws_equipped=True,
        airframe_id="BELL429",
    )

    return "BELL429", mission, wx, 29.75, 11.0


def build_scenario_4(engine: TraumaTeamDispatchEngine) -> tuple[str, MissionProfile, CorridorWeather, float, float]:
    """Scenario 4: Severe Headwind & Fuel Reserve Margin Exhaustion (Airbus H145).
    Eastern plains cross-country mission (Centennial -> Limon -> UCHealth) facing 45 kt adverse headwind.
    Fuel remaining on landing is below the FAR 135.209 mandatory 20-minute VFR day reserve.
    Outcome: NO-GO (FAR_135_209_VFR_RESERVE_VIOLATION).
    """
    base = Waypoint(identifier="KAPA", name="Centennial Airport", lat=39.5701, lon=-104.8493, elevation_ft=5885.0)
    scene = Waypoint(identifier="LIMON", name="I-70 Limon Rural Trauma Scene", lat=39.2639, lon=-103.6922, elevation_ft=5377.0)
    hosp = Waypoint(identifier="UCH", name="UCHealth University Hospital Helipad", lat=39.7420, lon=-104.8390, elevation_ft=5380.0, is_elevated_helipad=False)

    wx = CorridorWeather(
        departure_metar=METARObservation(
            station_id="KAPA",
            observation_time_utc="2100Z",
            wind_dir_deg=280.0,
            wind_speed_kts=25.0,
            visibility_sm=10.0,
            ceiling_agl_ft=5000.0,
            temperature_c=16.0,
            dewpoint_c=3.0,
            altimeter_inhg=29.92,
            flight_category=FlightRuleCategory.VFR,
        ),
        winds_aloft_profile=[
            WindsAloftLayer(altitude_msl_ft=7500.0, direction_deg=280.0, speed_kts=45.0, temperature_c=8.0)
        ],
    )

    mission = engine.create_standard_hems_mission(
        mission_id="HEMS_SCENARIO_04_FUEL_EXHAUSTION",
        tail_number="N145TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hosp,
        initial_fuel_kg=350.0,
        flight_crew_weights={
            "PILOT_RH": 85.0,
            "COPILOT_LH": 80.0,
            "CREW_ATTENDANT_AFT_LH": 75.0,
            "CREW_ATTENDANT_AFT_RH": 80.0,
        },
        patient_weight_kg=85.0,
        medical_gear_weight_kg=60.0,
        cruise_alt_msl_ft=7500.0,
        scene_hover_oge_min=5.0,
        is_day=True,
        is_mountainous=False,
        is_local_flying_area=False,
        nvg_or_htaws_equipped=True,
        airframe_id="H145",
    )

    return "H145", mission, wx, 29.92, 16.0


def build_scenario_5(engine: TraumaTeamDispatchEngine) -> tuple[str, MissionProfile, CorridorWeather, float, float]:
    """Scenario 5: Lateral & Longitudinal Center-of-Gravity Asymmetry (Bell 429).
    Extreme lateral arm displacement with specialized external rescue hoist and rigging package.
    Exceeds certified Bell 429 RFM lateral CG envelope boundary (67.5 mm).
    Outcome: NO-GO (LATERAL_CG_RIGHT_EXCEEDED).
    """
    spec = engine.get_airframe("BELL429")
    if "RESCUE_HOIST_ARM" not in spec.stations:
        spec.stations["RESCUE_HOIST_ARM"] = StationArm(
            station_id="RESCUE_HOIST_ARM",
            name="RH Extended Rescue Hoist & Rigging Kit",
            arm_long_mm=3100.0,
            arm_lat_mm=1150.0,
            max_capacity_kg=300.0,
        )
        engine.register_airframe("BELL429", spec)

    base = Waypoint(identifier="KAPA", name="Centennial Airport", lat=39.5701, lon=-104.8493, elevation_ft=5885.0)
    scene = Waypoint(identifier="CANYON", name="Platte River Canyon Technical Rescue", lat=39.4200, lon=-105.1500, elevation_ft=6200.0)
    hosp = Waypoint(identifier="SWEDISH", name="Swedish Medical Center (Ground Helipad)", lat=39.6536, lon=-104.9781, elevation_ft=5350.0, is_elevated_helipad=False)

    mission = engine.create_standard_hems_mission(
        mission_id="HEMS_SCENARIO_05_CG_ASYMMETRY",
        tail_number="N429TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hosp,
        initial_fuel_kg=300.0,
        flight_crew_weights={
            "PILOT_RH": 90.0,
            "CREW_ATTENDANT_RH": 85.0,
        },
        patient_weight_kg=75.0,
        medical_gear_weight_kg=40.0,
        aft_cargo_weight_kg=20.0,
        cruise_alt_msl_ft=7500.0,
        scene_hover_oge_min=4.0,
        is_day=True,
        is_mountainous=False,
        is_local_flying_area=True,
        nvg_or_htaws_equipped=True,
        airframe_id="BELL429",
    )

    mission.payloads.append(
        PayloadItem(
            item_id="HOIST_RIG",
            name="RH Rescue Hoist & Specialized Technical Rig",
            station_id="RESCUE_HOIST_ARM",
            weight_kg=145.0,
            loaded_at_leg=0,
            unloaded_at_leg=None,
        )
    )

    wx = CorridorWeather(
        departure_metar=METARObservation(
            station_id="KAPA",
            observation_time_utc="1700Z",
            wind_dir_deg=180.0,
            wind_speed_kts=8.0,
            visibility_sm=10.0,
            ceiling_agl_ft=5000.0,
            temperature_c=18.0,
            dewpoint_c=4.0,
            altimeter_inhg=29.95,
            flight_category=FlightRuleCategory.VFR,
        )
    )

    return "BELL429", mission, wx, 29.95, 18.0


# ==============================================================================
# SCENARIO SUITE REGISTRY
# ==============================================================================

def get_scenario_suite() -> Dict[int, ScenarioDefinition]:
    """Returns the certified benchmark suite of 5 operational scenarios."""
    return {
        1: ScenarioDefinition(
            scenario_id=1,
            title="Nominal Lowland Dispatch",
            description="Airbus H145, Denver Metro lowland corridor, VMC, balanced CG.",
            expected_verdict=DispatchVerdict.GO,
            expected_limiting_factor=LimitingFactor.NONE,
            builder=build_scenario_1,
        ),
        2: ScenarioDefinition(
            scenario_id=2,
            title="High Density Altitude Mountain Rescue",
            description="Airbus H145, Leadville Lake County 3V2 (9,934 ft MSL, DA > 12,000 ft), OGE hover ceiling deficit.",
            expected_verdict=DispatchVerdict.NO_GO,
            expected_limiting_factor=LimitingFactor.OGE_HOVER_SCENE_DEFICIT,
            builder=build_scenario_2,
        ),
        3: ScenarioDefinition(
            scenario_id=3,
            title="Convective SIGMET & Ceiling Bust",
            description="Bell 429, Front Range thunderstorm corridor, cloud ceiling 400 ft below FAR 135.609 Table 1 minimums.",
            expected_verdict=DispatchVerdict.NO_GO,
            expected_limiting_factor=LimitingFactor.FAR_135_615_CEILING_BELOW_MINIMUMS,
            builder=build_scenario_3,
        ),
        4: ScenarioDefinition(
            scenario_id=4,
            title="Severe Headwind & Fuel Reserve Margin Exhaustion",
            description="Airbus H145, 45 kt adverse headwind vector, landing fuel below FAR 135.209 20-minute statutory reserve.",
            expected_verdict=DispatchVerdict.NO_GO,
            expected_limiting_factor=LimitingFactor.FAR_135_209_VFR_RESERVE_VIOLATION,
            builder=build_scenario_4,
        ),
        5: ScenarioDefinition(
            scenario_id=5,
            title="Lateral & Longitudinal CG Asymmetry",
            description="Bell 429, extreme external rescue hoist lateral arm displacement exceeding RFM envelope boundary.",
            expected_verdict=DispatchVerdict.NO_GO,
            expected_limiting_factor=LimitingFactor.LATERAL_CG_RIGHT_EXCEEDED,
            builder=build_scenario_5,
        ),
    }


def execute_scenario(scenario_id: int, engine: Optional[TraumaTeamDispatchEngine] = None) -> ScenarioExecutionResult:
    """Executes a single scenario and returns verification result."""
    if engine is None:
        engine = TraumaTeamDispatchEngine()

    suite = get_scenario_suite()
    if scenario_id not in suite:
        raise ValueError(f"Scenario ID {scenario_id} not found. Available: {list(suite.keys())}")

    defn = suite[scenario_id]
    airframe_type, mission, wx, altimeter, temp = defn.builder(engine)
    evaluation = engine.evaluate_mission(airframe_type, mission, wx, altimeter_inhg=altimeter, ambient_temp_c=temp)

    passed = (
        evaluation.verdict == defn.expected_verdict
        and evaluation.limiting_factor == defn.expected_limiting_factor
    )

    return ScenarioExecutionResult(
        scenario_id=scenario_id,
        title=defn.title,
        expected_verdict=defn.expected_verdict.value,
        actual_verdict=evaluation.verdict.value,
        expected_limiting_factor=defn.expected_limiting_factor.value,
        actual_limiting_factor=evaluation.limiting_factor.value,
        passed=passed,
        evaluation=evaluation,
    )


# ==============================================================================
# ANSI TERMINAL DISPATCH BOARDS
# ==============================================================================

def print_header(title: str) -> None:
    c = TerminalColors
    width = 96
    print(f"\n{c.BOLD}{c.BRIGHT_CYAN}{'═' * width}{c.RESET}")
    print(f"{c.BOLD}{c.BRIGHT_WHITE}  TRAUMATEAM // HEMS DISPATCH DECISION SUITE  ::  14 CFR PART 135 SUBPART L{c.RESET}")
    print(f"{c.CYAN}{'─' * width}{c.RESET}")
    print(f"{c.BOLD}{c.WHITE}  SCENARIO :: {title.upper()}{c.RESET}")
    print(f"{c.BOLD}{c.BRIGHT_CYAN}{'═' * width}{c.RESET}")


def render_scenario_board(result: ScenarioExecutionResult) -> None:
    c = TerminalColors
    ev = result.evaluation
    width = 96

    # Determine verdict color
    if ev.verdict == DispatchVerdict.GO:
        verdict_badge = f"{c.BOLD}{c.BG_GREEN}{c.WHITE} [ DISPATCH VERDICT: GO ] {c.RESET}"
    elif ev.verdict == DispatchVerdict.CAUTION:
        verdict_badge = f"{c.BOLD}{c.BG_YELLOW}{c.BLACK} [ DISPATCH VERDICT: CAUTION ] {c.RESET}"
    else:
        verdict_badge = f"{c.BOLD}{c.BG_RED}{c.WHITE} [ DISPATCH VERDICT: NO-GO ] {c.RESET}"

    status_tag = f"{c.GREEN}✓ PASSED TEST HARNESS{c.RESET}" if result.passed else f"{c.RED}✗ FAILED HARNESS{c.RESET}"

    print(f"\n{c.BOLD}[SCENARIO {result.scenario_id}] {result.title}{c.RESET}  ({status_tag})")
    print(f"Airframe: {c.BRIGHT_WHITE}{ev.airframe_type}{c.RESET} | Tail: {ev.tail_number} | Mission ID: {ev.mission_id}")
    print(f"Evaluation Verdict: {verdict_badge}  Limiting Factor: {c.BOLD}{c.YELLOW}{ev.limiting_factor.value}{c.RESET}")
    print(f"Operational Description: {c.DIM}{ev.limiting_description}{c.RESET}")

    # Flight release tabular summary
    print(f"\n{c.DIM}{'─' * width}{c.RESET}")
    print(f"{c.BOLD}{c.WHITE}{'#':<3} {'LEG NAME':<38} {'DIST':>7} {'TIME':>7} {'BURN':>7} {'REM FL':>8} {'GW':>7} {'TAS/GS':>10}{c.RESET}")
    print(f"{c.DIM}{'─' * width}{c.RESET}")

    for leg in ev.legs:
        tas_gs = f"{int(leg.tas_kts)}/{int(leg.gs_kts)}" if leg.tas_kts > 0 else "0/0"
        leg_name_truncated = (leg.name[:35] + "..") if len(leg.name) > 37 else leg.name
        print(
            f"{leg.leg_index:<3} "
            f"{leg_name_truncated:<38} "
            f"{leg.distance_nm:6.1f}M "
            f"{leg.duration_min:6.1f}m "
            f"{leg.fuel_burn_kg:6.1f}k "
            f"{leg.remaining_fuel_kg:7.1f}k "
            f"{leg.end_gw_kg:6.0f}k "
            f"{tas_gs:>10}"
        )

    print(f"{c.DIM}{'─' * width}{c.RESET}")
    print(
        f"{c.BOLD}TOTALS:{c.RESET} Distance: {c.BRIGHT_WHITE}{ev.total_distance_nm:.1f} NM{c.RESET} | "
        f"Flight Time: {c.BRIGHT_WHITE}{ev.total_flight_time_min:.1f} min{c.RESET} | "
        f"Total Fuel Burn: {c.BRIGHT_WHITE}{ev.total_fuel_burn_kg:.1f} kg{c.RESET} | "
        f"Landing Reserve: {c.BRIGHT_WHITE}{ev.landing_reserve_fuel_kg:.1f} kg{c.RESET} "
        f"({ev.reserve_margin_minutes:.1f} min vs 20-min FAR 135.209 req)"
    )

    # Actionable mitigations
    if ev.mitigation_actions:
        print(f"\n{c.BOLD}{c.BRIGHT_YELLOW}ACTIONABLE MITIGATION RECOMMENDATIONS:{c.RESET}")
        for i, act in enumerate(ev.mitigation_actions, 1):
            print(f"  {c.YELLOW}[{i}]{c.RESET} {act}")

    print(f"{c.DIM}{'═' * width}{c.RESET}")


def render_suite_matrix(results: List[ScenarioExecutionResult]) -> None:
    c = TerminalColors
    width = 96
    print(f"\n{c.BOLD}{c.BRIGHT_CYAN}{'═' * width}{c.RESET}")
    print(f"{c.BOLD}{c.BRIGHT_WHITE}  CERTIFIED SCENARIO BENCHMARK VERIFICATION MATRIX (FAA PART 135 & RFM PHYSICS){c.RESET}")
    print(f"{c.CYAN}{'─' * width}{c.RESET}")
    print(f"{c.BOLD}{'ID':<4} {'SCENARIO NAME':<34} {'AIRCRAFT':<10} {'EXP VERDICT':<13} {'ACT VERDICT':<13} {'STATUS':<10}{c.RESET}")
    print(f"{c.DIM}{'─' * width}{c.RESET}")

    all_passed = True
    for r in results:
        passed_text = f"{c.BRIGHT_GREEN}PASS{c.RESET}" if r.passed else f"{c.BRIGHT_RED}FAIL{c.RESET}"
        if not r.passed:
            all_passed = False
        print(
            f"{r.scenario_id:<4} "
            f"{r.title:<34} "
            f"{r.evaluation.airframe_type[:9]:<10} "
            f"{r.expected_verdict:<13} "
            f"{r.actual_verdict:<13} "
            f"{passed_text}"
        )

    print(f"{c.DIM}{'─' * width}{c.RESET}")
    overall_status = f"{c.BOLD}{c.BG_GREEN}{c.WHITE} ALL 5 SCENARIOS VERIFIED DETERMINISTICALLY {c.RESET}" if all_passed else f"{c.BOLD}{c.BG_RED}{c.WHITE} VERIFICATION DEFICIT DETECTED {c.RESET}"
    print(f"OVERALL STATUS: {overall_status}")
    print(f"{c.BOLD}{c.BRIGHT_CYAN}{'═' * width}{c.RESET}\n")


# ==============================================================================
# MAIN CLI ENTRYPOINT
# ==============================================================================

def main(argv: Optional[List[str]] = None) -> int:
    """Main CLI entry point for traumateam-demo."""
    parser = argparse.ArgumentParser(
        prog="traumateam-demo",
        description="TraumaTeam Multi-Scenario HEMS Dispatch Feasibility & RFM Aerodynamic Suite (14 CFR Part 135 Subpart L)",
    )
    group = parser.add_mutually_exclusive_group()
    group.add_argument(
        "--scenario",
        "-s",
        type=int,
        choices=[1, 2, 3, 4, 5],
        help="Run a specific scenario (1: Nominal GO, 2: Mountain High-DA, 3: Wx Ceiling Bust, 4: Headwind Fuel, 5: CG Asymmetry)",
    )
    group.add_argument(
        "--all",
        "-a",
        action="store_true",
        default=True,
        help="Run all 5 certified operational benchmark scenarios (default: True)",
    )
    parser.add_argument(
        "--json",
        "-j",
        action="store_true",
        help="Emit results in structured JSON format for automated integration",
    )

    args = parser.parse_args(argv)

    suite = get_scenario_suite()
    engine = TraumaTeamDispatchEngine()

    scenario_ids = [args.scenario] if args.scenario else sorted(suite.keys())
    results: List[ScenarioExecutionResult] = []

    for sid in scenario_ids:
        res = execute_scenario(sid, engine=engine)
        results.append(res)

    if args.json:
        payload = {
            "title": "TraumaTeam HEMS Dispatch Feasibility Demonstration Suite",
            "regulation": "14 CFR Part 135 Subpart L (§§ 135.601 - 135.621)",
            "all_passed": all(r.passed for r in results),
            "scenarios": [r.to_dict() for r in results],
        }
        print(json.dumps(payload, indent=2))
    else:
        print_header("HEMS Operational Benchmark Suite (14 CFR Part 135 Subpart L)")
        for res in results:
            render_scenario_board(res)
        if len(results) > 1:
            render_suite_matrix(results)

    # Return code semantics: 0 on expected scenario outcome, non-zero on unexpected error
    return 0 if all(r.passed for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
