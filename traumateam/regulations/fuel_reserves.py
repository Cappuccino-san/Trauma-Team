"""FAA 14 CFR § 135.209 Fuel Reserve Regulations & Operator Contingency Engine."""

from __future__ import annotations
from dataclasses import dataclass
from traumateam.models.verdict import LimitingFactor, DispatchVerdict

@dataclass(frozen=True)
class FuelReserveReport:
    required_reserve_min: float
    required_reserve_kg: float
    remaining_fuel_kg: float
    reserve_margin_kg: float
    reserve_margin_min: float
    is_faa_legal: bool
    verdict: DispatchVerdict
    limiting_factor: LimitingFactor
    detail_message: str

def evaluate_fuel_reserves(
    remaining_fuel_kg: float,
    cruise_fuel_flow_kg_hr: float,
    is_vfr: bool = True,
    company_policy_min: float = 30.0,
    faa_vfr_min: float = 20.0,
    faa_ifr_min: float = 30.0
) -> FuelReserveReport:
    """Evaluates remaining fuel at the final landing point against FAR 135.209 and company safety policy.
    
    FAR 135.209 specifies minimum 20 minutes VFR reserve for rotorcraft.
    If remaining fuel is below the mandatory FAA threshold (20 min VFR / 30 min IFR) -> NO-GO.
    If between FAA statutory minimum and higher company safety policy (e.g., 30 min) -> CAUTION.
    """
    statutory_min = faa_vfr_min if is_vfr else faa_ifr_min
    
    # Fuel flow per minute
    fuel_per_min = cruise_fuel_flow_kg_hr / 60.0
    faa_required_kg = statutory_min * fuel_per_min
    policy_required_kg = company_policy_min * fuel_per_min

    reserve_margin_kg = remaining_fuel_kg - faa_required_kg
    actual_reserve_min = (remaining_fuel_kg / fuel_per_min) if fuel_per_min > 0 else 999.0
    reserve_margin_min = actual_reserve_min - statutory_min

    is_faa_legal = remaining_fuel_kg >= faa_required_kg

    if not is_faa_legal:
        verdict = DispatchVerdict.NO_GO
        limiting_factor = LimitingFactor.FAR_135_209_VFR_RESERVE_VIOLATION if is_vfr else LimitingFactor.IFR_RESERVE_VIOLATION
        deficit_kg = faa_required_kg - remaining_fuel_kg
        deficit_min = statutory_min - actual_reserve_min
        msg = (
            f"FUEL RESERVE VIOLATION (14 CFR § 135.209): Planned fuel on landing is {remaining_fuel_kg:.1f} kg "
            f"({actual_reserve_min:.1f} min), which is {deficit_kg:.1f} kg ({deficit_min:.1f} min) BELOW the "
            f"mandatory {statutory_min:.0f}-minute statutory reserve of {faa_required_kg:.1f} kg."
        )
    elif remaining_fuel_kg < policy_required_kg:
        verdict = DispatchVerdict.CAUTION
        limiting_factor = LimitingFactor.NONE
        msg = (
            f"CAUTION: Fuel reserves meet FAA statutory {statutory_min:.0f}-min minimum ({actual_reserve_min:.1f} min available, "
            f"{remaining_fuel_kg:.1f} kg), but fall below company conservative safety buffer of {company_policy_min:.0f} min "
            f"({policy_required_kg:.1f} kg). Refuel prior to departure or plan en-route fuel stop."
        )
    else:
        verdict = DispatchVerdict.GO
        limiting_factor = LimitingFactor.NONE
        msg = (
            f"Fuel reserves compliant: {remaining_fuel_kg:.1f} kg remaining ({actual_reserve_min:.1f} min) "
            f"vs FAA mandatory {faa_required_kg:.1f} kg ({statutory_min:.0f} min)."
        )

    return FuelReserveReport(
        required_reserve_min=statutory_min,
        required_reserve_kg=faa_required_kg,
        remaining_fuel_kg=remaining_fuel_kg,
        reserve_margin_kg=reserve_margin_kg,
        reserve_margin_min=reserve_margin_min,
        is_faa_legal=is_faa_legal,
        verdict=verdict,
        limiting_factor=limiting_factor,
        detail_message=msg,
    )
