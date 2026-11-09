use approx::assert_relative_eq;
use traumateam_rfm::airframes::{get_bell429_rfm_spec, get_h145_rfm_spec};
use traumateam_rfm::units::{Celsius, Feet};

#[test]
fn test_h145_rfm_exact_grid_points() {
    let h145 = get_h145_rfm_spec();

    // From H145 raw table: PA = 8000 ft, OAT = 20 C
    let pa = Feet::new(8000.0);
    let oat = Celsius::new(20.0);

    let oge = h145.get_max_oge_hover_weight(pa, oat);
    let ige = h145.get_max_ige_hover_weight(pa, oat);
    let oei = h145.get_max_oei_cat_a_weight(pa, oat);

    assert_relative_eq!(oge.get(), 3160.0, epsilon = 1e-4);
    assert_relative_eq!(ige.get(), 3510.0, epsilon = 1e-4);
    assert_relative_eq!(oei.get(), 2710.0, epsilon = 1e-4);
}

#[test]
fn test_h145_rfm_intermediate_bilinear_interpolation() {
    let h145 = get_h145_rfm_spec();

    // Halfway point: PA = 7000 ft, OAT = 25 C
    // Python benchmark calculation yielded exactly 3185.0 kg
    let pa = Feet::new(7000.0);
    let oat = Celsius::new(25.0);

    let computed_oge = h145.get_max_oge_hover_weight(pa, oat);
    assert_relative_eq!(computed_oge.get(), 3185.0, epsilon = 1e-3);
}

#[test]
fn test_bell429_rfm_hover_performance() {
    let b429 = get_bell429_rfm_spec();

    // Sea level 15 C: MTOW capped at 3175.0 kg
    let oge_sl = b429.get_max_oge_hover_weight(Feet::new(0.0), Celsius::new(15.0));
    assert_relative_eq!(oge_sl.get(), 3175.0, epsilon = 1e-4);

    // 10,000 ft, 30 C: 2320.0 kg
    let oge_high = b429.get_max_oge_hover_weight(Feet::new(10000.0), Celsius::new(30.0));
    assert_relative_eq!(oge_high.get(), 2320.0, epsilon = 1e-4);
}

#[test]
fn test_bell429_rfm_exact_grid_points() {
    let b429 = get_bell429_rfm_spec();

    // From Bell 429 raw tables at PA = 6000 ft (index 4), OAT = 20 C (index 4)
    let pa = Feet::new(6000.0);
    let oat = Celsius::new(20.0);

    let ige = b429.get_max_ige_hover_weight(pa, oat);
    let oge = b429.get_max_oge_hover_weight(pa, oat);
    let oei = b429.get_max_oei_cat_a_weight(pa, oat);

    assert_relative_eq!(ige.get(), 3090.0, epsilon = 1e-4);
    assert_relative_eq!(oge.get(), 2800.0, epsilon = 1e-4);
    assert_relative_eq!(oei.get(), 2410.0, epsilon = 1e-4);
}

#[test]
fn test_bell429_rfm_intermediate_bilinear_interpolation() {
    let b429 = get_bell429_rfm_spec();

    // Halfway point: PA = 5000 ft (between 4000 and 6000), OAT = 25 C (between 20 and 30)
    // 4000 ft: 20C=2970, 30C=2850 -> mid at 25C = 2910 kg
    // 6000 ft: 20C=2800, 30C=2680 -> mid at 25C = 2740 kg
    // 5000 ft, 25C: (2910 + 2740) / 2 = 2825.0 kg
    let computed_oge = b429.get_max_oge_hover_weight(Feet::new(5000.0), Celsius::new(25.0));
    assert_relative_eq!(computed_oge.get(), 2825.0, epsilon = 1e-3);
}

#[test]
fn test_phase_specific_fuel_flows() {
    let h145 = get_h145_rfm_spec();
    let pa = Feet::new(5000.0);
    let oat = Celsius::new(15.0);
    let gw = traumateam_rfm::units::Kilograms::new(3200.0);

    let cruise_ff = h145.get_cruise_fuel_flow(pa, oat, gw);
    let hover_ff = h145.get_hover_oge_fuel_flow(pa, oat, gw);
    let loiter_ff = h145.get_loiter_fuel_flow(pa, oat, gw);

    // H145 factors: hover = 1.20, loiter = 0.85
    assert_relative_eq!(hover_ff.get(), cruise_ff.get() * 1.20, epsilon = 1e-6);
    assert_relative_eq!(loiter_ff.get(), cruise_ff.get() * 0.85, epsilon = 1e-6);

    let b429 = get_bell429_rfm_spec();
    let b_gw = traumateam_rfm::units::Kilograms::new(2800.0);
    let b_cruise = b429.get_cruise_fuel_flow(pa, oat, b_gw);
    let b_hover = b429.get_hover_oge_fuel_flow(pa, oat, b_gw);
    let b_loiter = b429.get_loiter_fuel_flow(pa, oat, b_gw);

    // Bell 429 factors: hover = 1.18, loiter = 0.86
    assert_relative_eq!(b_hover.get(), b_cruise.get() * 1.18, epsilon = 1e-6);
    assert_relative_eq!(b_loiter.get(), b_cruise.get() * 0.86, epsilon = 1e-6);
}

#[test]
fn test_rfm_boundary_conditions_and_clamping() {
    let h145 = get_h145_rfm_spec();

    // Out of bounds PA: -5000 ft (below -2000 ft table minimum)
    let below_pa = h145.get_max_oge_hover_weight(Feet::new(-5000.0), Celsius::new(15.0));
    let table_min_pa = h145.get_max_oge_hover_weight(Feet::new(-2000.0), Celsius::new(15.0));
    assert_eq!(below_pa, table_min_pa);

    // Out of bounds PA: 20000 ft (above 14000 ft table maximum)
    let above_pa = h145.get_max_oge_hover_weight(Feet::new(20000.0), Celsius::new(15.0));
    let table_max_pa = h145.get_max_oge_hover_weight(Feet::new(14000.0), Celsius::new(15.0));
    assert_eq!(above_pa, table_max_pa);

    // Hover weights must never exceed MTOW
    assert!(
        h145.get_max_ige_hover_weight(Feet::new(-2000.0), Celsius::new(-20.0))
            .get()
            <= h145.mtow.get()
    );
    assert!(
        h145.get_max_oge_hover_weight(Feet::new(-2000.0), Celsius::new(-20.0))
            .get()
            <= h145.mtow.get()
    );
}

#[test]
fn test_aerodynamic_monotonicity_across_pressure_altitudes() {
    let h145 = get_h145_rfm_spec();

    // Property: Hover ceiling must monotonically decrease or stay constant with increasing altitude
    for oat in [-10.0, 0.0, 15.0, 25.0, 35.0] {
        let temp = Celsius::new(oat);
        let mut prev_weight = h145.get_max_oge_hover_weight(Feet::new(0.0), temp).get();

        let mut alt = 1000.0;
        while alt <= 14000.0 {
            let current_weight = h145.get_max_oge_hover_weight(Feet::new(alt), temp).get();
            assert!(
                current_weight <= prev_weight + 1e-5,
                "Monotonicity violation at PA={alt} ft, OAT={oat} C: {current_weight} > {prev_weight}"
            );
            prev_weight = current_weight;
            alt += 1000.0;
        }
    }
}
