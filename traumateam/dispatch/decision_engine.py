"""TraumaTeam High-Level Dispatch Feasibility Engine & Decision Interface."""

from __future__ import annotations
from typing import Dict, List, Optional
from traumateam.models.rfm import RFMAirframeSpec
from traumateam.models.mission import MissionProfile, LegSpec, LegType, PayloadItem, Waypoint
from traumateam.models.weather import CorridorWeather, METARObservation, WindsAloftLayer
from traumateam.models.verdict import DispatchEvaluation, DispatchVerdict
from traumateam.performance.airframes.h145 import get_h145_rfm_spec
from traumateam.performance.airframes.bell429 import get_bell429_rfm_spec
from traumateam.simulation.mission_integrator import MissionSimulationIntegrator

class TraumaTeamDispatchEngine:
    """The core deterministic dispatch feasibility engine for Helicopter Emergency Medical Services (HEMS)."""

    def __init__(self):
        self._fleet: Dict[str, RFMAirframeSpec] = {
            "H145": get_h145_rfm_spec(),
            "BK117-D2": get_h145_rfm_spec(),
            "BELL429": get_bell429_rfm_spec(),
        }

    def register_airframe(self, key: str, spec: RFMAirframeSpec) -> None:
        self._fleet[key.upper()] = spec

    def get_airframe(self, key: str) -> RFMAirframeSpec:
        airframe = self._fleet.get(key.upper())
        if not airframe:
            raise KeyError(f"Airframe '{key}' not registered in fleet catalog. Available: {list(self._fleet.keys())}")
        return airframe

    def create_standard_hems_mission(
        self,
        mission_id: str,
        tail_number: str,
        base_waypoint: Waypoint,
        scene_waypoint: Waypoint,
        hospital_waypoint: Waypoint,
        initial_fuel_kg: float,
        flight_crew_weights: Dict[str, float], # e.g. {"PILOT_RH": 85, "COPILOT_LH": 85, "CREW_ATTENDANT_AFT_LH": 80, "CREW_ATTENDANT_AFT_RH": 85}
        patient_weight_kg: float = 80.0,
        medical_gear_weight_kg: float = 60.0,
        aft_cargo_weight_kg: float = 35.0,
        cruise_alt_msl_ft: float = 3000.0,
        scene_hover_oge_min: float = 5.0,
        is_day: bool = True,
        is_mountainous: bool = False,
        is_local_flying_area: bool = False,
        nvg_or_htaws_equipped: bool = True,
        airframe_id: str = "H145",
    ) -> MissionProfile:
        """Helper to construct an authentic standard 7-leg HEMS mission loop:
        Leg 0: Base Startup
        Leg 1: Base to Scene Transit
        Leg 2: Scene Operations (OGE Hover & Patient Load)
        Leg 3: Scene to Hospital Transit
        Leg 4: Hospital Operations (Helipad Landing & Patient Unload)
        Leg 5: Hospital to Base Return Transit
        Leg 6: Base Shutdown
        """
        target_spec = self._fleet.get(airframe_id.upper(), self._fleet["H145"])
        payloads: List[PayloadItem] = []

        station_aliases = {
            "CREW_ATTENDANT_AFT_LH": "CREW_ATTENDANT_LH",
            "CREW_ATTENDANT_AFT_RH": "CREW_ATTENDANT_RH",
            "CREW_ATTENDANT_LH": "CREW_ATTENDANT_AFT_LH",
            "CREW_ATTENDANT_RH": "CREW_ATTENDANT_AFT_RH",
            "MEDICAL_CABINET_RACK": "MEDICAL_EQUIPMENT_RACK",
            "MEDICAL_EQUIPMENT_RACK": "MEDICAL_CABINET_RACK",
            "AFT_AVIONICS_CARGO": "AFT_BAGGAGE",
            "AFT_BAGGAGE": "AFT_AVIONICS_CARGO",
        }

        # Crew payloads (onboard for entire mission)
        for st_id, w in flight_crew_weights.items():
            resolved_st = st_id if st_id in target_spec.stations else station_aliases.get(st_id, st_id)
            payloads.append(PayloadItem(
                item_id=f"CREW_{st_id}",
                name=f"Crew member at {resolved_st}",
                station_id=resolved_st,
                weight_kg=w,
                loaded_at_leg=0,
                unloaded_at_leg=None,
            ))

        # Medical equipment rack (ECMO / monitor / bags)
        payloads.append(PayloadItem(
            item_id="MED_GEAR_PRIMARY",
            name="Modular Medical Equipment Rack",
            station_id="MEDICAL_CABINET_RACK" if "MEDICAL_CABINET_RACK" in target_spec.stations else "MEDICAL_EQUIPMENT_RACK",
            weight_kg=medical_gear_weight_kg,
            loaded_at_leg=0,
            unloaded_at_leg=None,
        ))

        # Aft cargo (standard HEMS survival kit / trauma bags / oxygen bottles in clamshell)
        if aft_cargo_weight_kg > 0:
            payloads.append(PayloadItem(
                item_id="AFT_CARGO_BAGS",
                name="Aft Clamshell Trauma Bags & Survival Kit",
                station_id="AFT_AVIONICS_CARGO" if "AFT_AVIONICS_CARGO" in target_spec.stations else "AFT_BAGGAGE",
                weight_kg=aft_cargo_weight_kg,
                loaded_at_leg=0,
                unloaded_at_leg=None,
            ))

        # Patient loaded at Scene (Leg 2) and unloaded at Hospital (Leg 4)
        payloads.append(PayloadItem(
            item_id="PATIENT_CRITICAL",
            name="Trauma Patient on Primary Litter",
            station_id="PRIMARY_PATIENT_LITTER",
            weight_kg=patient_weight_kg,
            loaded_at_leg=2,
            unloaded_at_leg=4,
        ))

        # Define the 7 mission legs
        legs = [
            LegSpec(
                leg_index=0,
                name="Base Startup & Pre-Flight Checks",
                leg_type=LegType.STARTUP,
                origin=base_waypoint,
                destination=base_waypoint,
                planned_cruise_altitude_msl_ft=base_waypoint.elevation_ft,
                ground_idle_min=5.0,
            ),
            LegSpec(
                leg_index=1,
                name=f"Outbound Transit: {base_waypoint.name} -> Scene {scene_waypoint.name}",
                leg_type=LegType.TRANSIT,
                origin=base_waypoint,
                destination=scene_waypoint,
                planned_cruise_altitude_msl_ft=cruise_alt_msl_ft,
            ),
            LegSpec(
                leg_index=2,
                name=f"Scene Operations & Patient Loading ({scene_waypoint.name})",
                leg_type=LegType.SCENE_OPERATIONS,
                origin=scene_waypoint,
                destination=scene_waypoint,
                planned_cruise_altitude_msl_ft=scene_waypoint.elevation_ft,
                hover_oge_min=scene_hover_oge_min,
                ground_idle_min=10.0,
            ),
            LegSpec(
                leg_index=3,
                name=f"Inbound Patient Transit: Scene {scene_waypoint.name} -> {hospital_waypoint.name}",
                leg_type=LegType.TRANSIT,
                origin=scene_waypoint,
                destination=hospital_waypoint,
                planned_cruise_altitude_msl_ft=cruise_alt_msl_ft,
            ),
            LegSpec(
                leg_index=4,
                name=f"Hospital Helipad Landing & Patient Transfer ({hospital_waypoint.name})",
                leg_type=LegType.HOSPITAL_OPERATIONS,
                origin=hospital_waypoint,
                destination=hospital_waypoint,
                planned_cruise_altitude_msl_ft=hospital_waypoint.elevation_ft,
                hover_ige_min=2.0,
                ground_idle_min=8.0,
            ),
            LegSpec(
                leg_index=5,
                name=f"Return Transit: {hospital_waypoint.name} -> {base_waypoint.name}",
                leg_type=LegType.TRANSIT,
                origin=hospital_waypoint,
                destination=base_waypoint,
                planned_cruise_altitude_msl_ft=cruise_alt_msl_ft,
            ),
            LegSpec(
                leg_index=6,
                name="Base Shutdown & Debrief",
                leg_type=LegType.BASE_SHUTDOWN,
                origin=base_waypoint,
                destination=base_waypoint,
                planned_cruise_altitude_msl_ft=base_waypoint.elevation_ft,
                ground_idle_min=3.0,
            ),
        ]

        return MissionProfile(
            mission_id=mission_id,
            tail_number=tail_number,
            initial_fuel_kg=initial_fuel_kg,
            payloads=payloads,
            legs=legs,
            is_vfr=True,
            is_day=is_day,
            is_mountainous=is_mountainous,
            is_local_flying_area=is_local_flying_area,
            nvg_or_htaws_equipped=nvg_or_htaws_equipped,
        )

    def evaluate_mission(
        self,
        airframe_type: str,
        mission: MissionProfile,
        weather: CorridorWeather,
        altimeter_inhg: float = 29.92,
        ambient_temp_c: float = 15.0,
    ) -> DispatchEvaluation:
        """Runs the deterministic mission feasibility engine and generates the complete flight operations verdict."""
        spec = self.get_airframe(airframe_type)
        station_aliases = {
            "CREW_ATTENDANT_AFT_LH": "CREW_ATTENDANT_LH",
            "CREW_ATTENDANT_AFT_RH": "CREW_ATTENDANT_RH",
            "CREW_ATTENDANT_LH": "CREW_ATTENDANT_AFT_LH",
            "CREW_ATTENDANT_RH": "CREW_ATTENDANT_AFT_RH",
            "MEDICAL_CABINET_RACK": "MEDICAL_EQUIPMENT_RACK",
            "MEDICAL_EQUIPMENT_RACK": "MEDICAL_CABINET_RACK",
            "AFT_AVIONICS_CARGO": "AFT_BAGGAGE",
            "AFT_BAGGAGE": "AFT_AVIONICS_CARGO",
        }
        needs_remap = any(
            p.station_id not in spec.stations and station_aliases.get(p.station_id) in spec.stations
            for p in mission.payloads
        )
        if needs_remap:
            remapped_payloads = [
                p.model_copy(update={"station_id": station_aliases[p.station_id]})
                if p.station_id not in spec.stations and station_aliases.get(p.station_id) in spec.stations
                else p
                for p in mission.payloads
            ]
            mission = mission.model_copy(update={"payloads": remapped_payloads})

        integrator = MissionSimulationIntegrator(spec)
        return integrator.execute_mission(
            mission=mission,
            weather=weather,
            altimeter_inhg=altimeter_inhg,
            ambient_temp_c=ambient_temp_c,
        )
