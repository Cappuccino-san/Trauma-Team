"""Tests for FAA Regulations: 14 CFR § 135.615 and § 135.209."""

import pytest
from traumateam.regulations.far135_615 import (
    get_far_135_615_minimums,
    evaluate_far_135_615_compliance,
)
from traumateam.regulations.fuel_reserves import evaluate_fuel_reserves
from traumateam.models.verdict import LimitingFactor, DispatchVerdict

def test_far_135_615_complete_matrix():
    """Validates every legal cell in 14 CFR § 135.615 Table 1."""

    # 1. Non-mountainous local
    # Day without NVG/HTAWS: 800-2; with NVG/HTAWS: 500-2
    r = get_far_135_615_minimums(is_day=True, is_mountainous=False, is_local=True, nvg_or_htaws_equipped=False)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (800.0, 2.0)
    r = get_far_135_615_minimums(is_day=True, is_mountainous=False, is_local=True, nvg_or_htaws_equipped=True)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (500.0, 2.0)

    # Night local: 1000-3
    r = get_far_135_615_minimums(is_day=False, is_mountainous=False, is_local=True, nvg_or_htaws_equipped=False)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (1000.0, 3.0)
    r = get_far_135_615_minimums(is_day=False, is_mountainous=False, is_local=True, nvg_or_htaws_equipped=True)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (1000.0, 3.0)

    # 2. Non-mountainous cross-country
    # Day without NVG: 800-3; with NVG: 600-2
    r = get_far_135_615_minimums(is_day=True, is_mountainous=False, is_local=False, nvg_or_htaws_equipped=False)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (800.0, 3.0)
    r = get_far_135_615_minimums(is_day=True, is_mountainous=False, is_local=False, nvg_or_htaws_equipped=True)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (600.0, 2.0)

    # Night without NVG: 1000-3; with NVG: 800-3
    r = get_far_135_615_minimums(is_day=False, is_mountainous=False, is_local=False, nvg_or_htaws_equipped=False)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (1000.0, 3.0)
    r = get_far_135_615_minimums(is_day=False, is_mountainous=False, is_local=False, nvg_or_htaws_equipped=True)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (800.0, 3.0)

    # 3. Mountainous local
    # Day: 800-3
    r = get_far_135_615_minimums(is_day=True, is_mountainous=True, is_local=True, nvg_or_htaws_equipped=False)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (800.0, 3.0)
    # Night: 1500-3 without NVG; 1000-3 with NVG
    r = get_far_135_615_minimums(is_day=False, is_mountainous=True, is_local=True, nvg_or_htaws_equipped=False)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (1500.0, 3.0)
    r = get_far_135_615_minimums(is_day=False, is_mountainous=True, is_local=True, nvg_or_htaws_equipped=True)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (1000.0, 3.0)

    # 4. Mountainous cross-country
    # Day: 1000-3
    r = get_far_135_615_minimums(is_day=True, is_mountainous=True, is_local=False, nvg_or_htaws_equipped=False)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (1000.0, 3.0)
    # Night: 1500-5 without NVG; 1000-3 with NVG
    r = get_far_135_615_minimums(is_day=False, is_mountainous=True, is_local=False, nvg_or_htaws_equipped=False)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (1500.0, 5.0)
    r = get_far_135_615_minimums(is_day=False, is_mountainous=True, is_local=False, nvg_or_htaws_equipped=True)
    assert (r.min_ceiling_ft, r.min_visibility_sm) == (1000.0, 3.0)

def test_far_135_209_fuel_reserve_evaluation():
    # Fuel flow = 240 kg/hr -> 4.0 kg/min
    # Mandatory 20-min reserve = 80.0 kg
    # Company 30-min policy = 120.0 kg

    # Case 1: 150 kg remaining -> GO
    rep_go = evaluate_fuel_reserves(remaining_fuel_kg=150.0, cruise_fuel_flow_kg_hr=240.0)
    assert rep_go.verdict == DispatchVerdict.GO
    assert rep_go.limiting_factor == LimitingFactor.NONE
    assert rep_go.is_faa_legal is True

    # Case 2: 100 kg remaining (25 min) -> Meets FAA legal, but below company 30-min -> CAUTION
    rep_caution = evaluate_fuel_reserves(remaining_fuel_kg=100.0, cruise_fuel_flow_kg_hr=240.0)
    assert rep_caution.verdict == DispatchVerdict.CAUTION
    assert rep_caution.limiting_factor == LimitingFactor.NONE
    assert rep_caution.is_faa_legal is True

    # Case 3: 60 kg remaining (15 min) -> Below mandatory 20 min -> NO-GO
    rep_nogo = evaluate_fuel_reserves(remaining_fuel_kg=60.0, cruise_fuel_flow_kg_hr=240.0)
    assert rep_nogo.verdict == DispatchVerdict.NO_GO
    assert rep_nogo.limiting_factor == LimitingFactor.FAR_135_209_VFR_RESERVE_VIOLATION
    assert rep_nogo.is_faa_legal is False
    assert rep_nogo.reserve_margin_kg < 0.0
