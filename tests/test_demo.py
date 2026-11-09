"""Automated test suite verifying the multi-scenario HEMS demo dispatch runner.

Verifies:
1. All 5 certified operational scenarios evaluate deterministically to their
   expected dispatch verdicts and limiting factors (physics & FAR Part 135).
2. Individual scenario execution via `execute_scenario(1..5)`.
3. Command-line flags `--all`, `--scenario 1..5`, `--json`, and `--help`.
4. Return code semantics (0 on expected outcome).
"""

import json
import pytest
from demo_dispatch import execute_scenario, get_scenario_suite, main
from traumateam.models.verdict import DispatchVerdict, LimitingFactor


def test_scenario_suite_registry():
    """Verify all 5 scenarios are registered with valid metadata."""
    suite = get_scenario_suite()
    assert len(suite) == 5
    for sid in range(1, 6):
        assert sid in suite
        defn = suite[sid]
        assert defn.scenario_id == sid
        assert len(defn.title) > 0
        assert callable(defn.builder)


def test_scenario_1_nominal_lowland_go():
    """Scenario 1: Airbus H145 nominal lowland flight produces GO."""
    res = execute_scenario(1)
    assert res.passed is True
    assert res.actual_verdict == DispatchVerdict.GO.value
    assert res.actual_limiting_factor == LimitingFactor.NONE.value
    assert res.evaluation.airframe_type == "Airbus H145 (BK117 D-2)"
    assert res.evaluation.landing_reserve_fuel_kg > 200.0
    assert res.evaluation.reserve_margin_minutes > 20.0


def test_scenario_2_high_density_altitude_oge_deficit():
    """Scenario 2: Airbus H145 Leadville Lake County high-DA rescue produces OGE deficit."""
    res = execute_scenario(2)
    assert res.passed is True
    assert res.actual_verdict == DispatchVerdict.NO_GO.value
    assert res.actual_limiting_factor == LimitingFactor.OGE_HOVER_SCENE_DEFICIT.value
    assert "OGE Hover Performance Deficit" in res.evaluation.limiting_description
    assert len(res.evaluation.mitigation_actions) >= 1
    assert "Reduce departure gross weight" in res.evaluation.mitigation_actions[0]


def test_scenario_3_convective_sigmet_ceiling_bust():
    """Scenario 3: Bell 429 Front Range corridor with 600 ft ceiling busts FAR 135.609 minimums."""
    res = execute_scenario(3)
    assert res.passed is True
    assert res.actual_verdict == DispatchVerdict.NO_GO.value
    assert res.actual_limiting_factor == LimitingFactor.FAR_135_615_CEILING_BELOW_MINIMUMS.value
    assert "14 CFR § 135.615" in res.evaluation.limiting_description
    assert "400 ft" in res.evaluation.limiting_description


def test_scenario_4_severe_headwind_fuel_exhaustion():
    """Scenario 4: Airbus H145 against 45 kt adverse headwind exhausts VFR fuel reserve."""
    res = execute_scenario(4)
    assert res.passed is True
    assert res.actual_verdict == DispatchVerdict.NO_GO.value
    assert res.actual_limiting_factor == LimitingFactor.FAR_135_209_VFR_RESERVE_VIOLATION.value
    assert "14 CFR § 135.209" in res.evaluation.limiting_description
    assert res.evaluation.reserve_margin_minutes < 0.0
    assert any("fuel" in m.lower() for m in res.evaluation.mitigation_actions)


def test_scenario_5_lateral_cg_asymmetry():
    """Scenario 5: Bell 429 external rescue hoist arm displaces lateral CG beyond boundary."""
    res = execute_scenario(5)
    assert res.passed is True
    assert res.actual_verdict == DispatchVerdict.NO_GO.value
    assert res.actual_limiting_factor == LimitingFactor.LATERAL_CG_RIGHT_EXCEEDED.value
    assert "Lateral CG" in res.evaluation.limiting_description
    assert "right of boundary" in res.evaluation.limiting_description


def test_demo_cli_help(capsys):
    """Verify traumateam-demo --help exits cleanly with code 0."""
    with pytest.raises(SystemExit) as excinfo:
        main(["--help"])
    assert excinfo.value.code == 0
    captured = capsys.readouterr()
    assert "TraumaTeam Multi-Scenario" in captured.out
    assert "--scenario" in captured.out
    assert "--json" in captured.out


def test_demo_cli_all_text_mode(capsys):
    """Verify running all scenarios in text mode produces formatted ANSI boards."""
    exit_code = main(["--all"])
    assert exit_code == 0
    captured = capsys.readouterr()
    assert "TRAUMATEAM // HEMS DISPATCH DECISION SUITE" in captured.out
    assert "[SCENARIO 1] Nominal Lowland Dispatch" in captured.out
    assert "[SCENARIO 2] High Density Altitude Mountain Rescue" in captured.out
    assert "[SCENARIO 3] Convective SIGMET & Ceiling Bust" in captured.out
    assert "[SCENARIO 4] Severe Headwind & Fuel Reserve Margin Exhaustion" in captured.out
    assert "[SCENARIO 5] Lateral & Longitudinal CG Asymmetry" in captured.out
    assert "ALL 5 SCENARIOS VERIFIED DETERMINISTICALLY" in captured.out


def test_demo_cli_single_scenario(capsys):
    """Verify running a single scenario via --scenario flag."""
    exit_code = main(["--scenario", "1"])
    assert exit_code == 0
    captured = capsys.readouterr()
    assert "[SCENARIO 1] Nominal Lowland Dispatch" in captured.out
    assert "[SCENARIO 2]" not in captured.out


def test_demo_cli_json_mode(capsys):
    """Verify running with --json emits valid JSON containing all 5 scenario results."""
    exit_code = main(["--all", "--json"])
    assert exit_code == 0
    captured = capsys.readouterr()
    payload = json.loads(captured.out)
    assert payload["all_passed"] is True
    assert len(payload["scenarios"]) == 5
    assert payload["scenarios"][0]["actual_verdict"] == "GO"
    assert payload["scenarios"][1]["actual_verdict"] == "NO-GO"
    assert payload["scenarios"][1]["actual_limiting_factor"] == "OGE_HOVER_SCENE_DEFICIT"
