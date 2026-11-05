"""Tests for RFM Performance Tables Interpolation Accuracy."""

import pytest
from traumateam.performance.airframes.h145 import get_h145_rfm_spec
from traumateam.performance.airframes.bell429 import get_bell429_rfm_spec
from traumateam.performance.rfm_engine import RFMPerformanceEngine

def test_h145_rfm_exact_grid_points():
    """Validates that interpolation at exact RFM grid points reproduces manual chart values with 0% error."""
    h145 = get_h145_rfm_spec()
    engine = RFMPerformanceEngine(h145)

    # From H145 raw table: PA=8000 ft, OAT=20 C:
    # 8000 ft is index 5, 20 C is index 4.
    # OGE table [5][4] = 3160 kg
    # IGE table [5][4] = 3510 kg
    # OEI table [5][4] = 2710 kg
    val_oge = engine.get_max_oge_hover_weight(pa_ft=8000.0, oat_c=20.0)
    assert pytest.approx(val_oge, rel=1e-4) == 3160.0

    val_ige = engine.get_max_ige_hover_weight(pa_ft=8000.0, oat_c=20.0)
    assert pytest.approx(val_ige, rel=1e-4) == 3510.0

    val_oei = engine.get_max_oei_cat_a_weight(pa_ft=8000.0, oat_c=20.0)
    assert pytest.approx(val_oei, rel=1e-4) == 2710.0

def test_h145_rfm_intermediate_interpolation_accuracy():
    """Validates intermediate bilinear interpolation within 0.5% tolerance of manual linear calculation."""
    h145 = get_h145_rfm_spec()
    engine = RFMPerformanceEngine(h145)

    # Test point: PA = 7000 ft (halfway between 6000 and 8000), OAT = 25 C (halfway between 20 and 30 C)
    # At 6000 ft: 20C=3350, 30C=3210 -> mid at 25C is (3350+3210)/2 = 3280 kg
    # At 8000 ft: 20C=3160, 30C=3020 -> mid at 25C is (3160+3020)/2 = 3090 kg
    # At 7000 ft, 25C: mid between 3280 and 3090 is (3280+3090)/2 = 3185 kg
    expected_oge = 3185.0
    computed_oge = engine.get_max_oge_hover_weight(pa_ft=7000.0, oat_c=25.0)

    # Must be within 0.5% of manual calculation
    assert abs(computed_oge - expected_oge) / expected_oge < 0.005
    assert pytest.approx(computed_oge, rel=1e-4) == 3185.0

def test_bell429_rfm_interpolation():
    """Validates Bell 429 RFM performance interpolation."""
    b429 = get_bell429_rfm_spec()
    engine = RFMPerformanceEngine(b429)

    # Check sea level 15 C hover capability: MTOW is 3175 kg
    oge_sl = engine.get_max_oge_hover_weight(pa_ft=0.0, oat_c=15.0)
    assert oge_sl == 3175.0

    # At 10000 ft, 30 C: OGE table index 6 (10000 ft), index 5 (30 C) = 2320 kg
    oge_high = engine.get_max_oge_hover_weight(pa_ft=10000.0, oat_c=30.0)
    assert pytest.approx(oge_high, rel=1e-4) == 2320.0
