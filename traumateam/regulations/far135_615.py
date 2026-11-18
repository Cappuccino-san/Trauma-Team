"""FAA 14 CFR § 135.609 VFR Flight Minimums for Helicopter Air Ambulance (HAA).

Implements the complete deterministic legal matrix from 14 CFR § 135.609 Table 1
(Part 135 Subpart L), reserving § 135.615 for VFR flight planning and obstacle clearance.
"""

from __future__ import annotations
from dataclasses import dataclass
from traumateam.models.verdict import LimitingFactor

@dataclass(frozen=True)
class WeatherMinimumRule:
    category: str
    condition: str
    min_ceiling_ft: float
    min_visibility_sm: float
    equipped_nvg_htaws: bool

@dataclass(frozen=True)
class WeatherComplianceReport:
    is_compliant: bool
    limiting_factor: LimitingFactor
    min_ceiling_required_ft: float
    actual_ceiling_ft: float
    ceiling_margin_ft: float
    min_visibility_required_sm: float
    actual_visibility_sm: float
    visibility_margin_sm: float
    legal_rule_applied: str
    explanation: str

def get_far_135_615_minimums(
    is_day: bool,
    is_mountainous: bool,
    is_local: bool,
    nvg_or_htaws_equipped: bool
) -> WeatherMinimumRule:
    """Evaluates 14 CFR § 135.609 Table 1 to determine required ceiling and visibility."""
    
    if not is_mountainous:
        if is_local:
            if is_day:
                if nvg_or_htaws_equipped:
                    return WeatherMinimumRule("Non-mountainous local", "Day (with NVG/HTAWS)", 500.0, 2.0, True)
                return WeatherMinimumRule("Non-mountainous local", "Day (standard)", 800.0, 2.0, False)
            else: # Night
                if nvg_or_htaws_equipped:
                    return WeatherMinimumRule("Non-mountainous local", "Night (with NVG/HTAWS)", 1000.0, 3.0, True)
                return WeatherMinimumRule("Non-mountainous local", "Night (standard)", 1000.0, 3.0, False)
        else: # Cross-country
            if is_day:
                if nvg_or_htaws_equipped:
                    return WeatherMinimumRule("Non-mountainous cross-country", "Day (with NVG/HTAWS)", 600.0, 2.0, True)
                return WeatherMinimumRule("Non-mountainous cross-country", "Day (standard)", 800.0, 3.0, False)
            else: # Night
                if nvg_or_htaws_equipped:
                    return WeatherMinimumRule("Non-mountainous cross-country", "Night (with NVG/HTAWS)", 800.0, 3.0, True)
                return WeatherMinimumRule("Non-mountainous cross-country", "Night (standard)", 1000.0, 3.0, False)
    else: # Mountainous
        if is_local:
            if is_day:
                return WeatherMinimumRule("Mountainous local", "Day", 800.0, 3.0, nvg_or_htaws_equipped)
            else: # Night
                if nvg_or_htaws_equipped:
                    return WeatherMinimumRule("Mountainous local", "Night (with NVG/HTAWS)", 1000.0, 3.0, True)
                return WeatherMinimumRule("Mountainous local", "Night (standard)", 1500.0, 3.0, False)
        else: # Cross-country
            if is_day:
                return WeatherMinimumRule("Mountainous cross-country", "Day", 1000.0, 3.0, nvg_or_htaws_equipped)
            else: # Night
                if nvg_or_htaws_equipped:
                    return WeatherMinimumRule("Mountainous cross-country", "Night (with NVG/HTAWS)", 1000.0, 3.0, True)
                return WeatherMinimumRule("Mountainous cross-country", "Night (standard)", 1500.0, 5.0, False)

def evaluate_far_135_615_compliance(
    observed_ceiling_ft: float,
    observed_vis_sm: float,
    is_day: bool,
    is_mountainous: bool,
    is_local: bool,
    nvg_or_htaws_equipped: bool
) -> WeatherComplianceReport:
    """Evaluates observed ceiling and visibility against 14 CFR § 135.609 Table 1 minimums."""
    rule = get_far_135_615_minimums(is_day, is_mountainous, is_local, nvg_or_htaws_equipped)
    
    ceil_margin = observed_ceiling_ft - rule.min_ceiling_ft
    vis_margin = observed_vis_sm - rule.min_visibility_sm

    ceil_ok = (ceil_margin >= 0.0)
    vis_ok = (vis_margin >= 0.0)

    is_compliant = (ceil_ok and vis_ok)
    limiting_factor = LimitingFactor.NONE
    explanation_parts = []

    if not ceil_ok:
        limiting_factor = LimitingFactor.FAR_135_615_CEILING_BELOW_MINIMUMS
        explanation_parts.append(
            f"Observed ceiling ({observed_ceiling_ft:.0f} ft AGL) is below required minimum ({rule.min_ceiling_ft:.0f} ft AGL) by {abs(ceil_margin):.0f} ft."
        )
    if not vis_ok:
        if limiting_factor == LimitingFactor.NONE:
            limiting_factor = LimitingFactor.FAR_135_615_VISIBILITY_BELOW_MINIMUMS
        explanation_parts.append(
            f"Observed visibility ({observed_vis_sm:.1f} SM) is below required minimum ({rule.min_visibility_sm:.1f} SM) by {abs(vis_margin):.1f} SM."
        )

    legal_cite = f"14 CFR § 135.609 Table 1 [{rule.category} - {rule.condition}]"
    if not explanation_parts:
        explanation = f"Weather complies with {legal_cite}: Ceiling margin +{ceil_margin:.0f} ft, Vis margin +{vis_margin:.1f} SM."
    else:
        explanation = f"WEATHER VIOLATION under {legal_cite}: " + " | ".join(explanation_parts)

    return WeatherComplianceReport(
        is_compliant=is_compliant,
        limiting_factor=limiting_factor,
        min_ceiling_required_ft=rule.min_ceiling_ft,
        actual_ceiling_ft=observed_ceiling_ft,
        ceiling_margin_ft=ceil_margin,
        min_visibility_required_sm=rule.min_visibility_sm,
        actual_visibility_sm=observed_vis_sm,
        visibility_margin_sm=vis_margin,
        legal_rule_applied=legal_cite,
        explanation=explanation,
    )
