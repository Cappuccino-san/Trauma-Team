use approx::assert_relative_eq;
use traumateam_rfm::airframes::{get_bell429_rfm_spec, get_h145_rfm_spec};
use traumateam_rfm::rfm::LimitingFactor;
use traumateam_rfm::units::{Celsius, Feet, Kilograms};

// ============================================================================
// 1. Aerodynamic Monotonicity Stress Tests across Tables
// ============================================================================

#[test]
fn test_aerodynamic_monotonicity_fine_grid_h145() {
    let h145 = get_h145_rfm_spec();

    // Altitude monotonicity test:
    // As altitude increases for ANY fixed temperature, allowable hover weight MUST NOT increase.
    let temps = [-20.0, -15.0, -10.0, -5.0, 0.0, 5.0, 10.0, 15.0, 20.0, 25.0, 30.0, 35.0, 40.0, 45.0, 50.0];
    for &oat_val in &temps {
        let oat = Celsius::new(oat_val);

        let mut prev_ige = h145.get_max_ige_hover_weight(Feet::new(-2000.0), oat).get();
        let mut prev_oge = h145.get_max_oge_hover_weight(Feet::new(-2000.0), oat).get();
        let mut prev_oei = h145.get_max_oei_cat_a_weight(Feet::new(-2000.0), oat).get();

        let mut alt = -1950.0;
        while alt <= 15000.0 {
            let pa = Feet::new(alt);
            let cur_ige = h145.get_max_ige_hover_weight(pa, oat).get();
            let cur_oge = h145.get_max_oge_hover_weight(pa, oat).get();
            let cur_oei = h145.get_max_oei_cat_a_weight(pa, oat).get();

            assert!(
                cur_ige <= prev_ige + 1e-5,
                "H145 IGE monotonicity violation: alt={alt}, oat={oat_val}, {cur_ige} > {prev_ige}"
            );
            assert!(
                cur_oge <= prev_oge + 1e-5,
                "H145 OGE monotonicity violation: alt={alt}, oat={oat_val}, {cur_oge} > {prev_oge}"
            );
            assert!(
                cur_oei <= prev_oei + 1e-5,
                "H145 OEI monotonicity violation: alt={alt}, oat={oat_val}, {cur_oei} > {prev_oei}"
            );

            prev_ige = cur_ige;
            prev_oge = cur_oge;
            prev_oei = cur_oei;
            alt += 50.0;
        }
    }

    // Temperature monotonicity test:
    // As temperature increases for ANY fixed altitude, allowable hover weight MUST NOT increase.
    let altitudes = [-2000.0, -1000.0, 0.0, 2000.0, 4000.0, 6000.0, 8000.0, 10000.0, 12000.0, 14000.0];
    for &alt_val in &altitudes {
        let pa = Feet::new(alt_val);

        let mut prev_ige = h145.get_max_ige_hover_weight(pa, Celsius::new(-20.0)).get();
        let mut prev_oge = h145.get_max_oge_hover_weight(pa, Celsius::new(-20.0)).get();
        let mut prev_oei = h145.get_max_oei_cat_a_weight(pa, Celsius::new(-20.0)).get();

        let mut temp = -19.5;
        while temp <= 55.0 {
            let oat = Celsius::new(temp);
            let cur_ige = h145.get_max_ige_hover_weight(pa, oat).get();
            let cur_oge = h145.get_max_oge_hover_weight(pa, oat).get();
            let cur_oei = h145.get_max_oei_cat_a_weight(pa, oat).get();

            assert!(
                cur_ige <= prev_ige + 1e-5,
                "H145 IGE temp monotonicity violation: alt={alt_val}, oat={temp}, {cur_ige} > {prev_ige}"
            );
            assert!(
                cur_oge <= prev_oge + 1e-5,
                "H145 OGE temp monotonicity violation: alt={alt_val}, oat={temp}, {cur_oge} > {prev_oge}"
            );
            assert!(
                cur_oei <= prev_oei + 1e-5,
                "H145 OEI temp monotonicity violation: alt={alt_val}, oat={temp}, {cur_oei} > {prev_oei}"
            );

            prev_ige = cur_ige;
            prev_oge = cur_oge;
            prev_oei = cur_oei;
            temp += 0.5;
        }
    }
}

#[test]
fn test_aerodynamic_monotonicity_fine_grid_bell429() {
    let b429 = get_bell429_rfm_spec();

    // Altitude monotonicity test for Bell 429
    let temps = [-20.0, -15.0, -10.0, -5.0, 0.0, 5.0, 10.0, 15.0, 20.0, 25.0, 30.0, 35.0, 40.0, 45.0, 50.0];
    for &oat_val in &temps {
        let oat = Celsius::new(oat_val);

        let mut prev_ige = b429.get_max_ige_hover_weight(Feet::new(-2000.0), oat).get();
        let mut prev_oge = b429.get_max_oge_hover_weight(Feet::new(-2000.0), oat).get();
        let mut prev_oei = b429.get_max_oei_cat_a_weight(Feet::new(-2000.0), oat).get();

        let mut alt = -1950.0;
        while alt <= 13000.0 {
            let pa = Feet::new(alt);
            let cur_ige = b429.get_max_ige_hover_weight(pa, oat).get();
            let cur_oge = b429.get_max_oge_hover_weight(pa, oat).get();
            let cur_oei = b429.get_max_oei_cat_a_weight(pa, oat).get();

            assert!(
                cur_ige <= prev_ige + 1e-5,
                "Bell 429 IGE monotonicity violation: alt={alt}, oat={oat_val}, {cur_ige} > {prev_ige}"
            );
            assert!(
                cur_oge <= prev_oge + 1e-5,
                "Bell 429 OGE monotonicity violation: alt={alt}, oat={oat_val}, {cur_oge} > {prev_oge}"
            );
            assert!(
                cur_oei <= prev_oei + 1e-5,
                "Bell 429 OEI monotonicity violation: alt={alt}, oat={oat_val}, {cur_oei} > {prev_oei}"
            );

            prev_ige = cur_ige;
            prev_oge = cur_oge;
            prev_oei = cur_oei;
            alt += 50.0;
        }
    }

    // Temperature monotonicity test for Bell 429
    let altitudes = [-2000.0, -1000.0, 0.0, 2000.0, 4000.0, 6000.0, 8000.0, 10000.0, 12000.0];
    for &alt_val in &altitudes {
        let pa = Feet::new(alt_val);

        let mut prev_ige = b429.get_max_ige_hover_weight(pa, Celsius::new(-20.0)).get();
        let mut prev_oge = b429.get_max_oge_hover_weight(pa, Celsius::new(-20.0)).get();
        let mut prev_oei = b429.get_max_oei_cat_a_weight(pa, Celsius::new(-20.0)).get();

        let mut temp = -19.5;
        while temp <= 55.0 {
            let oat = Celsius::new(temp);
            let cur_ige = b429.get_max_ige_hover_weight(pa, oat).get();
            let cur_oge = b429.get_max_oge_hover_weight(pa, oat).get();
            let cur_oei = b429.get_max_oei_cat_a_weight(pa, oat).get();

            assert!(
                cur_ige <= prev_ige + 1e-5,
                "Bell 429 IGE temp monotonicity violation: alt={alt_val}, oat={temp}, {cur_ige} > {prev_ige}"
            );
            assert!(
                cur_oge <= prev_oge + 1e-5,
                "Bell 429 OGE temp monotonicity violation: alt={alt_val}, oat={temp}, {cur_oge} > {prev_oge}"
            );
            assert!(
                cur_oei <= prev_oei + 1e-5,
                "Bell 429 OEI temp monotonicity violation: alt={alt_val}, oat={temp}, {cur_oei} > {prev_oei}"
            );

            prev_ige = cur_ige;
            prev_oge = cur_oge;
            prev_oei = cur_oei;
            temp += 0.5;
        }
    }
}

// ============================================================================
// 2. Fuel Burn Rate Transitions Stress Tests
// ============================================================================

#[test]
fn test_fuel_burn_rate_transitions_h145() {
    let h145 = get_h145_rfm_spec();

    assert_relative_eq!(h145.hover_fuel_flow_factor, 1.20, epsilon = 1e-9);
    assert_relative_eq!(h145.loiter_fuel_flow_factor, 0.85, epsilon = 1e-9);

    let pas = [-3000.0, -2000.0, 0.0, 3000.0, 7000.0, 10000.0, 14000.0, 20000.0];
    let oats = [-30.0, -15.0, 0.0, 15.0, 30.0, 45.0, 60.0];
    let gws = [1500.0, 1900.0, 2500.0, 3100.0, 3700.0, 4200.0];

    for &pa_val in &pas {
        for &oat_val in &oats {
            for &gw_val in &gws {
                let pa = Feet::new(pa_val);
                let oat = Celsius::new(oat_val);
                let gw = Kilograms::new(gw_val);

                let cruise = h145.get_cruise_fuel_flow(pa, oat, gw);
                let hover = h145.get_hover_oge_fuel_flow(pa, oat, gw);
                let loiter = h145.get_loiter_fuel_flow(pa, oat, gw);

                assert!(cruise.get() > 0.0, "Cruise fuel flow must be positive: {cruise:?}");
                assert!(hover > cruise, "Hover fuel flow must exceed cruise fuel flow: {hover:?} vs {cruise:?}");
                assert!(loiter < cruise, "Loiter fuel flow must be less than cruise fuel flow: {loiter:?} vs {cruise:?}");

                assert_relative_eq!(hover.get(), cruise.get() * 1.20, epsilon = 1e-6);
                assert_relative_eq!(loiter.get(), cruise.get() * 0.85, epsilon = 1e-6);
                assert_relative_eq!(hover.get() / cruise.get(), 1.20, epsilon = 1e-7);
                assert_relative_eq!(loiter.get() / cruise.get(), 0.85, epsilon = 1e-7);
            }
        }
    }
}

#[test]
fn test_fuel_burn_rate_transitions_bell429() {
    let b429 = get_bell429_rfm_spec();

    assert_relative_eq!(b429.hover_fuel_flow_factor, 1.18, epsilon = 1e-9);
    assert_relative_eq!(b429.loiter_fuel_flow_factor, 0.86, epsilon = 1e-9);

    let pas = [-3000.0, -2000.0, 0.0, 3000.0, 6000.0, 10000.0, 12000.0, 18000.0];
    let oats = [-30.0, -15.0, 0.0, 15.0, 30.0, 45.0, 60.0];
    let gws = [1500.0, 1800.0, 2200.0, 2700.0, 3175.0, 3800.0];

    for &pa_val in &pas {
        for &oat_val in &oats {
            for &gw_val in &gws {
                let pa = Feet::new(pa_val);
                let oat = Celsius::new(oat_val);
                let gw = Kilograms::new(gw_val);

                let cruise = b429.get_cruise_fuel_flow(pa, oat, gw);
                let hover = b429.get_hover_oge_fuel_flow(pa, oat, gw);
                let loiter = b429.get_loiter_fuel_flow(pa, oat, gw);

                assert!(cruise.get() > 0.0, "Cruise fuel flow must be positive: {cruise:?}");
                assert!(hover > cruise, "Hover fuel flow must exceed cruise fuel flow: {hover:?} vs {cruise:?}");
                assert!(loiter < cruise, "Loiter fuel flow must be less than cruise fuel flow: {loiter:?} vs {cruise:?}");

                assert_relative_eq!(hover.get(), cruise.get() * 1.18, epsilon = 1e-6);
                assert_relative_eq!(loiter.get(), cruise.get() * 0.86, epsilon = 1e-6);
                assert_relative_eq!(hover.get() / cruise.get(), 1.18, epsilon = 1e-7);
                assert_relative_eq!(loiter.get() / cruise.get(), 0.86, epsilon = 1e-7);
            }
        }
    }
}

// ============================================================================
// 3. LimitingFactor 1:1 Mapping Validation
// ============================================================================

#[test]
fn test_all_17_limiting_factors_map_1_to_1() {
    let factors = [
        (LimitingFactor::None, "NONE"),
        (LimitingFactor::MtowExceeded, "MTOW_EXCEEDED"),
        (LimitingFactor::MinWeightViolation, "MIN_WEIGHT_VIOLATION"),
        (LimitingFactor::LongitudinalCgForwardExceeded, "LONGITUDINAL_CG_FORWARD_EXCEEDED"),
        (LimitingFactor::LongitudinalCgAftExceeded, "LONGITUDINAL_CG_AFT_EXCEEDED"),
        (LimitingFactor::LateralCgLeftExceeded, "LATERAL_CG_LEFT_EXCEEDED"),
        (LimitingFactor::LateralCgRightExceeded, "LATERAL_CG_RIGHT_EXCEEDED"),
        (LimitingFactor::OgeHoverSceneDeficit, "OGE_HOVER_SCENE_DEFICIT"),
        (LimitingFactor::IgeHoverDeficit, "IGE_HOVER_DEFICIT"),
        (LimitingFactor::OeiCatAHelipadDeficit, "OEI_CAT_A_HELIPAD_DEFICIT"),
        (LimitingFactor::Far135_615CeilingBelowMinimums, "FAR_135_615_CEILING_BELOW_MINIMUMS"),
        (LimitingFactor::Far135_615VisibilityBelowMinimums, "FAR_135_615_VISIBILITY_BELOW_MINIMUMS"),
        (LimitingFactor::Far135_209VfrReserveViolation, "FAR_135_209_VFR_RESERVE_VIOLATION"),
        (LimitingFactor::IfrReserveViolation, "IFR_RESERVE_VIOLATION"),
        (LimitingFactor::SevereWeatherHazard, "SEVERE_WEATHER_HAZARD"),
        (LimitingFactor::TerrainObstacleClearanceViolation, "TERRAIN_OBSTACLE_CLEARANCE_VIOLATION"),
        (LimitingFactor::UnflyableWind, "UNFLYABLE_WIND"),
    ];

    assert_eq!(factors.len(), 17, "Exactly 17 LimitingFactors must be defined");

    let mut set = std::collections::HashSet::new();
    for (factor, expected_str) in factors {
        assert_eq!(factor.as_str(), expected_str);
        assert_eq!(format!("{factor}"), expected_str);
        assert!(set.insert(expected_str), "Duplicate enum string: {expected_str}");
    }
    assert_eq!(set.len(), 17);
}

// ============================================================================
// 4. Fuel System & CG Envelope Stress Checks
// ============================================================================

#[test]
fn test_fuel_arm_depletion_schedule_stress_bounds() {
    let h145 = get_h145_rfm_spec();
    let fs = &h145.fuel_system;

    // Below zero clamps to 0 fuel arm (3900.0 mm)
    let (arm_l_neg, arm_w_neg) = fs.fuel_arm(Kilograms::new(-100.0));
    assert_relative_eq!(arm_l_neg.get(), 3900.0, epsilon = 1e-6);
    assert_relative_eq!(arm_w_neg.get(), 0.0, epsilon = 1e-6);

    // Above capacity clamps to 694 fuel arm (4020.0 mm)
    let (arm_l_pos, arm_w_pos) = fs.fuel_arm(Kilograms::new(1000.0));
    assert_relative_eq!(arm_l_pos.get(), 4020.0, epsilon = 1e-6);
    assert_relative_eq!(arm_w_pos.get(), 0.0, epsilon = 1e-6);

    // Monotonicity of fuel arm schedule: arm moves aft as fuel increases
    let mut prev_arm = arm_l_neg.get();
    let mut f = 0.0;
    while f <= 700.0 {
        let (arm, _) = fs.fuel_arm(Kilograms::new(f));
        assert!(arm.get() >= prev_arm - 1e-9, "Fuel arm should move aft with increasing fuel");
        prev_arm = arm.get();
        f += 10.0;
    }
}

#[test]
fn test_cg_envelope_boundary_stress() {
    let h145 = get_h145_rfm_spec();

    // Below min envelope weight clamps to first point
    let lim_low = h145.get_cg_limits_at_weight(Kilograms::new(1000.0));
    let lim_first = h145.get_cg_limits_at_weight(Kilograms::new(2000.0));
    assert_eq!(lim_low, lim_first);

    // Above MTOW clamps to last point
    let lim_high = h145.get_cg_limits_at_weight(Kilograms::new(5000.0));
    let lim_last = h145.get_cg_limits_at_weight(Kilograms::new(3700.0));
    assert_eq!(lim_high, lim_last);

    // Check envelope geometry sanity: fwd < aft, left < right everywhere
    let mut gw = 1500.0;
    while gw <= 4000.0 {
        let lim = h145.get_cg_limits_at_weight(Kilograms::new(gw));
        assert!(lim.fwd_limit < lim.aft_limit, "Fwd limit must be strictly ahead of aft limit");
        assert!(lim.left_limit < lim.right_limit, "Left limit must be strictly less than right limit");
        gw += 50.0;
    }
}
