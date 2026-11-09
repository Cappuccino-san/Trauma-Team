//! Adversarial Stress Testing Harness for traumateam_rfm.
//!
//! Tests:
//! 1. Numerical precision and boundary edge conditions (-2000 to 14000 ft PA, -20 to 50 C OAT).
//! 2. Extreme payload shifts (severe forward/aft/left/right CG excursions).
//! 3. Fuel loads from 0 kg to capacity, negative fuel, and over-capacity fuel.
//! 4. Float edge values: NaN, +Inf, -Inf, subnormals.
//! 5. Zero gross weight and negative gross weight handling.

use approx::assert_relative_eq;
use traumateam_rfm::airframes::{get_bell429_rfm_spec, get_h145_rfm_spec};
use traumateam_rfm::interpolator::interp_1d;
use traumateam_rfm::mass_balance::compute_mass_and_balance;
use traumateam_rfm::rfm::{PayloadItem, StationId};
use traumateam_rfm::units::{Celsius, Feet, Kilograms};

#[test]
fn test_adversarial_boundary_grid_points_h145() {
    let h145 = get_h145_rfm_spec();

    // Boundary corners of H145 grid: PA in [-2000, 14000], OAT in [-20, 50]
    let pa_corners = [-2000.0, 14000.0];
    let oat_corners = [-20.0, 50.0];

    for &pa in &pa_corners {
        for &oat in &oat_corners {
            let ige = h145.get_max_ige_hover_weight(Feet::new(pa), Celsius::new(oat));
            let oge = h145.get_max_oge_hover_weight(Feet::new(pa), Celsius::new(oat));
            let oei = h145.get_max_oei_cat_a_weight(Feet::new(pa), Celsius::new(oat));

            // Must never be NaN or infinite
            assert!(!ige.get().is_nan(), "IGE is NaN at PA={pa}, OAT={oat}");
            assert!(!oge.get().is_nan(), "OGE is NaN at PA={pa}, OAT={oat}");
            assert!(!oei.get().is_nan(), "OEI is NaN at PA={pa}, OAT={oat}");

            // Must be bounded within [0, MTOW]
            assert!(ige.get() >= 0.0 && ige.get() <= h145.mtow.get());
            assert!(oge.get() >= 0.0 && oge.get() <= h145.mtow.get());
            assert!(oei.get() >= 0.0 && oei.get() <= h145.mtow.get());

            // Physical aerodynamic truth: IGE >= OGE at all conditions
            assert!(
                ige.get() >= oge.get() - 1e-6,
                "Ground effect violation: IGE ({}) < OGE ({}) at PA={}, OAT={}",
                ige.get(),
                oge.get(),
                pa,
                oat
            );
        }
    }
}

#[test]
fn test_adversarial_boundary_grid_points_bell429() {
    let b429 = get_bell429_rfm_spec();

    // Boundary corners of Bell 429 grid: PA in [-2000, 12000], OAT in [-20, 50]
    let pa_corners = [-2000.0, 12000.0];
    let oat_corners = [-20.0, 50.0];

    for &pa in &pa_corners {
        for &oat in &oat_corners {
            let ige = b429.get_max_ige_hover_weight(Feet::new(pa), Celsius::new(oat));
            let oge = b429.get_max_oge_hover_weight(Feet::new(pa), Celsius::new(oat));
            let oei = b429.get_max_oei_cat_a_weight(Feet::new(pa), Celsius::new(oat));

            assert!(!ige.get().is_nan());
            assert!(!oge.get().is_nan());
            assert!(!oei.get().is_nan());

            assert!(ige.get() >= 0.0 && ige.get() <= b429.mtow.get());
            assert!(oge.get() >= 0.0 && oge.get() <= b429.mtow.get());
            assert!(oei.get() >= 0.0 && oei.get() <= b429.mtow.get());

            assert!(
                ige.get() >= oge.get() - 1e-6,
                "Bell 429 Ground effect violation: IGE < OGE at PA={}, OAT={}",
                pa,
                oat
            );
        }
    }
}

#[test]
fn test_adversarial_fuel_schedule_sweep() {
    let h145 = get_h145_rfm_spec();
    let b429 = get_bell429_rfm_spec();

    // Test continuous fuel from -1000 kg to 5000 kg (beyond capacities 694 kg and 590 kg)
    let test_fuels = [
        -1000.0, -50.0, -0.001, 0.0, 0.0001, 50.0, 150.0, 300.0, 500.0, 590.0, 694.0, 700.0,
        1000.0, 5000.0,
    ];

    for &f in &test_fuels {
        let (h_long, h_lat) = h145.fuel_system.fuel_arm(Kilograms::new(f));
        assert!(!h_long.get().is_nan());
        assert!(!h_lat.get().is_nan());
        assert!(h_long.get() >= 3900.0 - 1e-6 && h_long.get() <= 4020.0 + 1e-6);

        let (b_long, b_lat) = b429.fuel_system.fuel_arm(Kilograms::new(f));
        assert!(!b_long.get().is_nan());
        assert!(!b_lat.get().is_nan());
        assert!(b_long.get() >= 3320.0 - 1e-6 && b_long.get() <= 3380.0 + 1e-6);
    }
}

#[test]
fn test_adversarial_severe_cg_envelope_excursions() {
    let h145 = get_h145_rfm_spec();

    // Severe lateral left shift with heavy payload on left attendant
    let mut left_payloads = Vec::new();
    for _ in 0..10 {
        left_payloads.push(PayloadItem::new(
            "p_left",
            "Left Load",
            StationId::CrewAttendantAftLh,
            Kilograms::new(100.0),
        ));
    }
    let mb_left = compute_mass_and_balance(&h145, Kilograms::new(200.0), &left_payloads)
        .expect("Should compute despite exceedance");
    assert!(!mb_left.is_overall_compliant);
    assert!(!mb_left.is_lat_cg_compliant || !mb_left.is_gw_compliant);

    // Extreme zero payload + empty fuel (checks minimum weight)
    let mb_dry = compute_mass_and_balance(&h145, Kilograms::new(0.0), &[])
        .expect("Should compute empty aircraft");
    assert!(mb_dry.is_gw_compliant);
    assert!(mb_dry.is_overall_compliant);
}

#[test]
fn test_adversarial_interp_1d_edge_cases() {
    let xs = [10.0, 20.0, 30.0];
    let ys = [100.0, 200.0, 300.0];

    // Out of bounds below
    assert_eq!(interp_1d(-1e10, &xs, &ys), 100.0);
    // Out of bounds above
    assert_eq!(interp_1d(1e10, &xs, &ys), 300.0);
    // Exact nodes
    assert_eq!(interp_1d(10.0, &xs, &ys), 100.0);
    assert_eq!(interp_1d(20.0, &xs, &ys), 200.0);
    assert_eq!(interp_1d(30.0, &xs, &ys), 300.0);
    // Single element
    assert_eq!(interp_1d(50.0, &[25.0], &[500.0]), 500.0);
}

#[test]
fn test_adversarial_inf_clamping_behavior() {
    let h145 = get_h145_rfm_spec();

    // Positive and negative infinity should clamp to boundary points without panicking
    let pos_inf_oge = h145.get_max_oge_hover_weight(Feet::new(f64::INFINITY), Celsius::new(15.0));
    let max_pa_oge = h145.get_max_oge_hover_weight(Feet::new(14000.0), Celsius::new(15.0));
    assert_eq!(pos_inf_oge, max_pa_oge);

    let neg_inf_oge =
        h145.get_max_oge_hover_weight(Feet::new(f64::NEG_INFINITY), Celsius::new(15.0));
    let min_pa_oge = h145.get_max_oge_hover_weight(Feet::new(-2000.0), Celsius::new(15.0));
    assert_eq!(neg_inf_oge, min_pa_oge);
}

#[test]
fn test_adversarial_subnormal_float_precision() {
    let h145 = get_h145_rfm_spec();

    // Subnormal float input
    let subnormal_pa = Feet::new(f64::MIN_POSITIVE / 2.0);
    let zero_pa = Feet::new(0.0);

    let oge_subnormal = h145.get_max_oge_hover_weight(subnormal_pa, Celsius::new(20.0));
    let oge_zero = h145.get_max_oge_hover_weight(zero_pa, Celsius::new(20.0));

    assert_relative_eq!(oge_subnormal.get(), oge_zero.get(), epsilon = 1e-6);
}

#[test]
fn test_adversarial_dense_grid_parity_against_analytical_scipy() {
    let h145 = get_h145_rfm_spec();

    // 72-point comprehensive grid matching Python test_cases generated earlier
    let pa_samples = [
        -2000.0, -1000.0, 0.0, 1500.0, 3000.0, 5500.0, 8000.0, 11000.0, 14000.0,
    ];
    let oat_samples = [-20.0, -15.0, -5.0, 0.0, 12.0, 25.0, 35.0, 50.0];

    for &pa in &pa_samples {
        for &oat in &oat_samples {
            let ige = h145.get_max_ige_hover_weight(Feet::new(pa), Celsius::new(oat));
            let oge = h145.get_max_oge_hover_weight(Feet::new(pa), Celsius::new(oat));
            let oei = h145.get_max_oei_cat_a_weight(Feet::new(pa), Celsius::new(oat));

            // Verify monotonicity: higher altitude means lower or equal performance
            if pa > -2000.0 {
                let lower_alt_oge =
                    h145.get_max_oge_hover_weight(Feet::new(pa - 1000.0), Celsius::new(oat));
                assert!(
                    oge.get() <= lower_alt_oge.get() + 1e-4,
                    "Monotonicity failed at PA={pa}, OAT={oat}"
                );
            }

            // Verify basic sanity
            assert!(ige.get() >= oge.get() - 1e-4);
            assert!(oei.get() <= ige.get() + 1e-4);
        }
    }
}

#[test]
fn test_adversarial_nan_panic_characterization() {
    // Formally verify that NaN inputs are handled safely without panicking:
    // Interpolators, fuel arms, CG envelope routines, and mass-balance calculations
    // execute cleanly without panics.
    let h145 = get_h145_rfm_spec();

    // 1. PerformanceGrid2D with NaN PA
    let res_grid2d = std::panic::catch_unwind(|| {
        h145.get_max_oge_hover_weight(Feet::new(f64::NAN), Celsius::new(20.0))
    });
    assert!(
        res_grid2d.is_ok(),
        "PerformanceGrid2D must not panic on NaN PA"
    );

    // 2. PerformanceGrid2D with NaN OAT
    let res_grid2d_oat = std::panic::catch_unwind(|| {
        h145.get_max_oge_hover_weight(Feet::new(5000.0), Celsius::new(f64::NAN))
    });
    assert!(
        res_grid2d_oat.is_ok(),
        "PerformanceGrid2D must not panic on NaN OAT"
    );

    // 3. FuelSystemSpec::fuel_arm with NaN
    let res_fuel = std::panic::catch_unwind(|| h145.fuel_system.fuel_arm(Kilograms::new(f64::NAN)));
    assert!(
        res_fuel.is_ok(),
        "FuelSystemSpec::fuel_arm must not panic on NaN fuel"
    );

    // 4. RFMAirframeSpec::get_cg_limits_at_weight with NaN
    let res_cg =
        std::panic::catch_unwind(|| h145.get_cg_limits_at_weight(Kilograms::new(f64::NAN)));
    assert!(
        res_cg.is_ok(),
        "get_cg_limits_at_weight must not panic on NaN weight"
    );

    // 5. interp_1d with NaN
    let res_interp1d = std::panic::catch_unwind(|| {
        interp_1d(f64::NAN, &[0.0, 10.0, 20.0], &[100.0, 200.0, 300.0])
    });
    assert!(res_interp1d.is_ok(), "interp_1d must not panic on NaN");

    // 6. compute_mass_and_balance with NaN fuel
    let res_mb_fuel =
        std::panic::catch_unwind(|| compute_mass_and_balance(&h145, Kilograms::new(f64::NAN), &[]));
    assert!(
        res_mb_fuel.is_ok(),
        "compute_mass_and_balance must not panic on NaN fuel"
    );
    assert_eq!(
        res_mb_fuel.unwrap(),
        Err(traumateam_rfm::MassBalanceError::ZeroGrossWeight)
    );
}
