"""TraumaTeam Command-Line Interface (CLI).

Provides deterministic HEMS mission feasibility assessment, airframe RFM specification
inspection, and 14 CFR Part 135 Subpart L regulatory criteria evaluation.
"""

from __future__ import annotations

import argparse
import sys
from typing import List, Optional

from traumateam.dispatch.decision_engine import TraumaTeamDispatchEngine
from traumateam.models.mission import Waypoint
from traumateam.models.weather import (
    CorridorWeather,
    FlightRuleCategory,
    METARObservation,
    WindsAloftLayer,
)
from traumateam.regulations.far135_615 import get_far_135_615_minimums


def create_parser() -> argparse.ArgumentParser:
    """Build the argument parser for the traumateam CLI."""
    parser = argparse.ArgumentParser(
        prog="traumateam",
        description="TraumaTeam: Deterministic HEMS Mission Feasibility & Dispatch Engine (14 CFR Part 135 Subpart L)",
    )
    subparsers = parser.add_subparsers(dest="command", help="Available subcommands")

    # Subcommand: dispatch
    p_dispatch = subparsers.add_parser("dispatch", help="Evaluate a HEMS dispatch mission")
    p_dispatch.add_argument(
        "--airframe",
        choices=["H145", "BELL429"],
        default="H145",
        help="Helicopter airframe type (default: H145)",
    )
    p_dispatch.add_argument(
        "--tail",
        default="N145TT",
        help="Helicopter registration tail number (default: N145TT)",
    )
    p_dispatch.add_argument(
        "--fuel",
        type=float,
        default=450.0,
        help="Initial fuel load in kg (default: 450.0)",
    )
    p_dispatch.add_argument(
        "--patient-weight",
        type=float,
        default=80.0,
        help="Patient weight in kg (default: 80.0)",
    )
    p_dispatch.add_argument(
        "--gear-weight",
        type=float,
        default=60.0,
        help="Medical equipment / gear weight in kg (default: 60.0)",
    )
    p_dispatch.add_argument(
        "--base-elev",
        type=float,
        default=5885.0,
        help="Base airport elevation in ft MSL (default: 5885.0 for KAPA)",
    )
    p_dispatch.add_argument(
        "--scene-elev",
        type=float,
        default=5280.0,
        help="Scene elevation in ft MSL (default: 5280.0)",
    )
    p_dispatch.add_argument(
        "--hospital-elev",
        type=float,
        default=5350.0,
        help="Hospital helipad elevation in ft MSL (default: 5350.0)",
    )
    p_dispatch.add_argument(
        "--altimeter",
        type=float,
        default=29.92,
        help="Altimeter setting in inHg (default: 29.92)",
    )
    p_dispatch.add_argument(
        "--temp",
        type=float,
        default=20.0,
        help="Ambient temperature in deg C (default: 20.0)",
    )
    p_dispatch.add_argument(
        "--visibility",
        type=float,
        default=10.0,
        help="Corridor visibility in statute miles (default: 10.0)",
    )
    p_dispatch.add_argument(
        "--ceiling",
        type=float,
        default=5000.0,
        help="Corridor cloud ceiling in ft AGL (default: 5000.0)",
    )
    p_dispatch.add_argument(
        "--wind-speed",
        type=float,
        default=10.0,
        help="Wind speed in knots (default: 10.0)",
    )
    p_dispatch.add_argument(
        "--wind-dir",
        type=float,
        default=180.0,
        help="Wind direction in degrees true (default: 180.0)",
    )
    p_dispatch.add_argument(
        "--night",
        action="store_true",
        help="Flight conducted at night (default: False/Day)",
    )
    p_dispatch.add_argument(
        "--mountainous",
        action="store_true",
        help="Route traverses designated mountainous terrain (FAR 95 Subpart B)",
    )
    p_dispatch.add_argument(
        "--cross-country",
        action="store_true",
        help="Mission is cross-country (> 50 NM from operating base)",
    )
    p_dispatch.add_argument(
        "--no-nvg",
        action="store_true",
        help="Aircraft is NOT equipped with NVG or HTAWS (default: equipped)",
    )

    # Subcommand: airframe
    p_airframe = subparsers.add_parser("airframe", help="Display helicopter RFM specifications")
    p_airframe.add_argument(
        "--type",
        choices=["H145", "BELL429", "ALL"],
        default="ALL",
        help="Airframe model (default: ALL)",
    )

    # Subcommand: regulations
    p_reg = subparsers.add_parser(
        "regulations",
        help="Display FAA 14 CFR Part 135 Subpart L VFR weather minimums matrix",
    )
    p_reg.add_argument(
        "--night",
        action="store_true",
        help="Night condition",
    )
    p_reg.add_argument(
        "--mountainous",
        action="store_true",
        help="Mountainous terrain",
    )
    p_reg.add_argument(
        "--cross-country",
        action="store_true",
        help="Cross-country flight",
    )
    p_reg.add_argument(
        "--no-nvg",
        action="store_true",
        help="Unaided flight without NVG or HTAWS",
    )

    return parser


def run_dispatch(args: argparse.Namespace) -> int:
    """Execute a mission feasibility dispatch evaluation."""
    engine = TraumaTeamDispatchEngine()
    airframe_key = args.airframe.upper()

    base = Waypoint(
        identifier="BASE",
        name="Operating Base",
        lat=39.5701,
        lon=-104.8493,
        elevation_ft=args.base_elev,
    )
    scene = Waypoint(
        identifier="SCENE",
        name="Emergency Trauma Scene",
        lat=39.7500,
        lon=-104.9900,
        elevation_ft=args.scene_elev,
    )
    hosp = Waypoint(
        identifier="HOSP",
        name="Receiving Trauma Center Helipad",
        lat=39.6536,
        lon=-104.9781,
        elevation_ft=args.hospital_elev,
        is_elevated_helipad=True,
    )

    flight_cat = FlightRuleCategory.VFR
    if args.ceiling < 1000.0 or args.visibility < 3.0:
        flight_cat = FlightRuleCategory.IFR
    elif args.ceiling <= 3000.0 or args.visibility <= 5.0:
        flight_cat = FlightRuleCategory.MVFR

    wx = CorridorWeather(
        departure_metar=METARObservation(
            station_id="BASE",
            observation_time_utc="1200Z",
            wind_dir_deg=args.wind_dir,
            wind_speed_kts=args.wind_speed,
            visibility_sm=args.visibility,
            ceiling_agl_ft=args.ceiling,
            temperature_c=args.temp,
            dewpoint_c=max(0.0, args.temp - 10.0),
            altimeter_inhg=args.altimeter,
            flight_category=flight_cat,
        ),
        winds_aloft_profile=[
            WindsAloftLayer(
                altitude_msl_ft=max(args.base_elev + 1000.0, 6000.0),
                direction_deg=args.wind_dir,
                speed_kts=args.wind_speed,
                temperature_c=args.temp - 2.0,
            )
        ],
    )

    # Crew layout by airframe
    if airframe_key == "BELL429":
        crew_weights = {
            "PILOT_RH": 85.0,
            "COPILOT_LH": 80.0,
            "CREW_ATTENDANT_LH": 75.0,
            "CREW_ATTENDANT_RH": 80.0,
        }
    else:
        crew_weights = {
            "PILOT_RH": 85.0,
            "COPILOT_LH": 80.0,
            "CREW_ATTENDANT_AFT_LH": 75.0,
            "CREW_ATTENDANT_AFT_RH": 80.0,
        }

    is_day = not args.night
    is_local = not args.cross_country
    nvg = not args.no_nvg

    mission = engine.create_standard_hems_mission(
        mission_id="CLI_DISPATCH_MISSION",
        tail_number=args.tail,
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hosp,
        initial_fuel_kg=args.fuel,
        flight_crew_weights=crew_weights,
        patient_weight_kg=args.patient_weight,
        medical_gear_weight_kg=args.gear_weight,
        cruise_alt_msl_ft=max(args.base_elev, args.scene_elev, args.hospital_elev) + 1200.0,
        scene_hover_oge_min=4.0,
        is_day=is_day,
        is_mountainous=args.mountainous,
        is_local_flying_area=is_local,
        nvg_or_htaws_equipped=nvg,
        airframe_id=airframe_key,
    )

    eval_result = engine.evaluate_mission(
        airframe_type=airframe_key,
        mission=mission,
        weather=wx,
        altimeter_inhg=args.altimeter,
        ambient_temp_c=args.temp,
    )

    print("\n" + "=" * 78)
    print(" TRAUMATEAM MISSION DISPATCH EVALUATION REPORT")
    print("=" * 78)
    print(f"Airframe:       {airframe_key} (Tail: {args.tail})")
    print(f"Environment:    {'Night' if args.night else 'Day'} | "
          f"{'Mountainous' if args.mountainous else 'Non-mountainous'} | "
          f"{'Cross-Country' if args.cross_country else 'Local Flying Area'} | "
          f"{'Unaided (No NVG)' if args.no_nvg else 'NVG/HTAWS Equipped'}")
    print(f"Atmosphere:     OAT {args.temp:.1f}°C | Altimeter {args.altimeter:.2f} inHg | Wind {args.wind_dir:.0f}°@{args.wind_speed:.0f}kt")
    print(f"Cloud/Vis:      Ceiling {args.ceiling:.0f} ft AGL | Visibility {args.visibility:.1f} SM")
    print("-" * 78)
    print(f"DISPATCH VERDICT:      {eval_result.verdict.value}")
    print(f"LIMITING FACTOR:       {eval_result.limiting_factor.value}")
    print(f"STATUS EXPLANATION:    {eval_result.limiting_description}")
    print("-" * 78)
    print(f"Total Distance:        {eval_result.total_distance_nm:.1f} NM")
    print(f"Total Flight Time:     {eval_result.total_flight_time_min:.1f} min")
    print(f"Fuel Consumption:      {eval_result.total_fuel_burn_kg:.1f} kg")
    print(f"Landing Fuel Reserve:  {eval_result.landing_reserve_fuel_kg:.1f} kg ({eval_result.reserve_margin_minutes:.1f} min)")

    if eval_result.mitigation_actions:
        print("\nACTIONABLE MITIGATIONS:")
        for action in eval_result.mitigation_actions:
            print(f" - {action}")
    print("=" * 78 + "\n")

    return 0 if eval_result.verdict.value == "GO" else 1


def run_airframe_specs(args: argparse.Namespace) -> int:
    """Print helicopter engineering specifications and limits."""
    engine = TraumaTeamDispatchEngine()
    models = ["H145", "BELL429"] if args.type == "ALL" else [args.type]

    print("\n" + "=" * 78)
    print(" TRAUMATEAM CERTIFIED ROTORCRAFT FLIGHT MANUAL (RFM) SPECIFICATIONS")
    print("=" * 78)

    for m in models:
        spec = engine.get_airframe(m)
        print(f"\nAIRFRAME MODEL: {spec.model_name} (OEM: {spec.oem})")
        print(f"Certification:        {spec.certification_basis}")
        print(f"Empty Weight:          {spec.empty_weight_kg:.1f} kg (Arm: {spec.empty_cg_long_mm:.1f} mm long, {spec.empty_cg_lat_mm:.1f} mm lat)")
        print(f"Max Gross Weight MTOW: {spec.mtow_kg:.1f} kg")
        print(f"Min Flight Weight:     {spec.min_flight_weight_kg:.1f} kg")
        print(f"Usable Fuel Capacity:  {spec.fuel_system.usable_fuel_capacity_kg:.1f} kg")
        print(f"Unusable Fuel:         {spec.fuel_system.unusable_fuel_kg:.1f} kg")
        print("Standard Loading Stations:")
        for st_id, st in spec.stations.items():
            print(f"  - {st_id:<24}: arm={st.arm_long_mm:6.1f} mm, lat={st.arm_lat_mm:6.1f} mm, max={st.max_capacity_kg:5.1f} kg")
        print("Certified CG Envelope Boundary:")
        for pt in spec.cg_envelope[:4]:
            print(f"  - GW: {pt.gross_weight_kg:5.1f} kg -> Long: [{pt.cg_long_fwd_mm:.1f} .. {pt.cg_long_aft_mm:.1f}] mm, Lat: [{pt.cg_lat_left_mm:.1f} .. {pt.cg_lat_right_mm:.1f}] mm")
        if len(spec.cg_envelope) > 4:
            last = spec.cg_envelope[-1]
            print(f"  - ... (MTOW {last.gross_weight_kg:.1f} kg -> Long: [{last.cg_long_fwd_mm:.1f} .. {last.cg_long_aft_mm:.1f}] mm)")

    print("\n" + "=" * 78 + "\n")
    return 0


def run_regulations(args: argparse.Namespace) -> int:
    """Print FAR 135.609 Table 1 flight minimums."""
    is_day = not args.night
    is_mountainous = args.mountainous
    is_local = not args.cross_country
    nvg = not args.no_nvg

    rule = get_far_135_615_minimums(
        is_day=is_day,
        is_mountainous=is_mountainous,
        is_local=is_local,
        nvg_or_htaws_equipped=nvg,
    )

    print("\n" + "=" * 78)
    print(" FAA 14 CFR § 135.609 TABLE 1 — VFR WEATHER MINIMUMS FOR HAA")
    print("=" * 78)
    print(f"Operation Category:     {rule.category}")
    print(f"Flight Condition:       {rule.condition}")
    print(f"Required Cloud Ceiling: {rule.min_ceiling_ft:.0f} ft AGL")
    print(f"Required Visibility:    {rule.min_visibility_sm:.1f} SM")
    print(f"Equipment Credit:       {'NVG / HTAWS Enabled' if rule.equipped_nvg_htaws else 'Unaided / Standard'}")
    print("=" * 78 + "\n")
    return 0


def main(argv: Optional[List[str]] = None) -> int:
    """Entry point for the CLI."""
    parser = create_parser()
    if argv is None:
        argv = sys.argv[1:]

    if not argv:
        parser.print_help()
        return 0

    args = parser.parse_args(argv)

    if args.command == "dispatch":
        return run_dispatch(args)
    elif args.command == "airframe":
        return run_airframe_specs(args)
    elif args.command == "regulations":
        return run_regulations(args)
    else:
        parser.print_help()
        return 0


if __name__ == "__main__":
    sys.exit(main())
