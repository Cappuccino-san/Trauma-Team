"""Automated test suite verifying the TraumaTeam CLI interface and entry points."""

import pytest
from traumateam.cli import main


def test_cli_help(capsys):
    """Verify that invoking CLI with --help outputs usage cleanly."""
    with pytest.raises(SystemExit) as excinfo:
        main(["--help"])
    assert excinfo.value.code == 0
    captured = capsys.readouterr()
    assert "TraumaTeam" in captured.out
    assert "dispatch" in captured.out
    assert "airframe" in captured.out
    assert "regulations" in captured.out


def test_cli_empty_argv(capsys):
    """Verify invoking with empty args prints help and returns 0."""
    exit_code = main([])
    assert exit_code == 0
    captured = capsys.readouterr()
    assert "TraumaTeam" in captured.out


def test_cli_airframe_specs_all(capsys):
    """Verify airframe inspection outputs both H145 and Bell 429 specifications."""
    exit_code = main(["airframe", "--type", "ALL"])
    assert exit_code == 0
    captured = capsys.readouterr()
    assert "Airbus H145" in captured.out
    assert "Bell 429" in captured.out
    assert "Max Gross Weight MTOW: 3700.0 kg" in captured.out
    assert "Max Gross Weight MTOW: 3175.0 kg" in captured.out


def test_cli_airframe_specs_single(capsys):
    """Verify airframe inspection for a single airframe."""
    exit_code = main(["airframe", "--type", "H145"])
    assert exit_code == 0
    captured = capsys.readouterr()
    assert "Airbus H145" in captured.out
    assert "Bell 429" not in captured.out


def test_cli_regulations(capsys):
    """Verify FAR 135.609 minimums calculation."""
    exit_code = main(["regulations", "--night", "--mountainous"])
    assert exit_code == 0
    captured = capsys.readouterr()
    assert "FAA 14 CFR § 135.609 TABLE 1" in captured.out
    assert "Required Cloud Ceiling: 1000 ft AGL" in captured.out
    assert "Required Visibility:    3.0 SM" in captured.out


def test_cli_dispatch_nominal_go(capsys):
    """Verify nominal dispatch scenario produces GO verdict (exit code 0)."""
    exit_code = main(["dispatch", "--airframe", "H145"])
    assert exit_code == 0
    captured = capsys.readouterr()
    assert "DISPATCH VERDICT:      GO" in captured.out
    assert "LIMITING FACTOR:       NONE" in captured.out


def test_cli_dispatch_low_ceiling_nogo(capsys):
    """Verify low ceiling produces NO-GO verdict (exit code 1)."""
    exit_code = main(["dispatch", "--airframe", "BELL429", "--ceiling", "400"])
    assert exit_code == 1
    captured = capsys.readouterr()
    assert "DISPATCH VERDICT:      NO-GO" in captured.out
    assert "FAR_135_615_CEILING_BELOW_MINIMUMS" in captured.out
