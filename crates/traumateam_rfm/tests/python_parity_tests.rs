use approx::assert_relative_eq;
use traumateam_rfm::airframes::get_h145_rfm_spec;
use traumateam_rfm::units::{Celsius, Feet, Kilograms};

#[test]
fn test_python_parity_cruise_performance() {
    let h145 = get_h145_rfm_spec();

    // Test at 6000 ft, 18 C, 3100 kg
    let pa = Feet::new(6000.0);
    let oat = Celsius::new(18.0);
    let gw = Kilograms::new(3100.0);

    let tas = h145.get_cruise_tas(pa, oat, gw);
    let ff = h145.get_cruise_fuel_flow(pa, oat, gw);

    // Python model calculation:
    // base_tas = 135.0 + (6000 / 2000) * 1.5 - ((3100 - 2500) / 1200) * 5.0 + (18 - 15) * 0.1
    // = 135.0 + 4.5 - 2.5 + 0.3 = 137.3 kts
    // base_ff = 210.0 + ((3100 - 2500) / 1200) * 45.0 - (6000 / 5000) * 8.0 + (18 / 50) * 6.0
    // = 210.0 + 22.5 - 9.6 + 2.16 = 225.06 kg/hr
    assert_relative_eq!(tas.get(), 137.3, epsilon = 0.5);
    assert_relative_eq!(ff.get(), 225.06, epsilon = 1.0);
}

#[test]
fn test_python_parity_high_da_mountain_rescue() {
    let h145 = get_h145_rfm_spec();

    // Leadville Turquoise Lake condition from Python Benchmark 2:
    // PA ~10000 ft, OAT 28 C
    let pa = Feet::new(10000.0);
    let oat = Celsius::new(28.0);

    let max_oge = h145.get_max_oge_hover_weight(pa, oat);

    // Python bilinear calculation yields 2848 kg
    assert_relative_eq!(max_oge.get(), 2848.0, epsilon = 5.0);
    assert!(
        max_oge.get() < 3000.0,
        "Must be derated due to density altitude"
    );
}

#[test]
fn test_python_parity_bell429_cruise_performance() {
    let b429 = traumateam_rfm::get_bell429_rfm_spec();

    // Test point: PA = 4000 ft, OAT = 10 C, GW = 2700 kg
    let pa = Feet::new(4000.0);
    let oat = Celsius::new(10.0);
    let gw = Kilograms::new(2700.0);

    let tas = b429.get_cruise_tas(pa, oat, gw);
    let ff = b429.get_cruise_fuel_flow(pa, oat, gw);

    // Python model calculation:
    // base_tas = 142.0 + (4000 / 2000)*1.8 - ((2700 - 2200)/975)*6.0 + (10 - 15)*0.1 = 142.02 kts
    // base_ff = 195.0 + ((2700 - 2200)/975)*40.0 - (4000 / 5000)*7.0 + (10 / 50)*5.0 = 210.91 kg/hr
    assert_relative_eq!(tas.get(), 142.02, epsilon = 0.5);
    assert_relative_eq!(ff.get(), 210.91, epsilon = 1.0);
}

#[test]
fn test_python_parity_all_h145_fuel_schedule_points() {
    let h145 = get_h145_rfm_spec();

    // Python schedule: [(0.0, 3900.0), (150.0, 3930.0), (350.0, 3975.0), (550.0, 4005.0), (694.0, 4020.0)]
    let check_points = [
        (0.0, 3900.0),
        (150.0, 3930.0),
        (350.0, 3975.0),
        (550.0, 4005.0),
        (694.0, 4020.0),
    ];

    for (fuel, expected_long) in check_points {
        let (arm_l, arm_w) = h145.fuel_system.fuel_arm(Kilograms::new(fuel));
        assert_relative_eq!(arm_l.get(), expected_long, epsilon = 1e-6);
        assert_relative_eq!(arm_w.get(), 0.0, epsilon = 1e-6);
    }

    // Intermediate points
    // 100 kg: 3900 + (100/150)*30 = 3920.0 mm
    let (arm_l, _) = h145.fuel_system.fuel_arm(Kilograms::new(100.0));
    assert_relative_eq!(arm_l.get(), 3920.0, epsilon = 1e-6);

    // 450 kg: 3975 + (100/200)*30 = 3990.0 mm
    let (arm_l, _) = h145.fuel_system.fuel_arm(Kilograms::new(450.0));
    assert_relative_eq!(arm_l.get(), 3990.0, epsilon = 1e-6);
}

#[test]
fn test_python_parity_all_bell429_fuel_schedule_points() {
    let b429 = traumateam_rfm::get_bell429_rfm_spec();

    // Python schedule: [(0.0, 3320.0), (200.0, 3340.0), (400.0, 3360.0), (590.0, 3380.0)]
    let check_points = [
        (0.0, 3320.0),
        (200.0, 3340.0),
        (400.0, 3360.0),
        (590.0, 3380.0),
    ];

    for (fuel, expected_long) in check_points {
        let (arm_l, arm_w) = b429.fuel_system.fuel_arm(Kilograms::new(fuel));
        assert_relative_eq!(arm_l.get(), expected_long, epsilon = 1e-6);
        assert_relative_eq!(arm_w.get(), 0.0, epsilon = 1e-6);
    }

    // Intermediate midpoint at 300 kg: 3350.0 mm
    let (arm_l, _) = b429.fuel_system.fuel_arm(Kilograms::new(300.0));
    assert_relative_eq!(arm_l.get(), 3350.0, epsilon = 1e-6);
}

#[test]
fn test_python_parity_h145_cg_envelope_margins() {
    let h145 = get_h145_rfm_spec();

    // At gross weight 3020.0 kg (from test_mass_balance_analytical_hand_calculation)
    // Interpolated between 2800 kg (3795, 4190, -80, 80) and 3400 kg (3815, 4170, -78, 78)
    // t = 220 / 600 = 0.36666667
    let limits = h145.get_cg_limits_at_weight(Kilograms::new(3020.0));

    let expected_fwd = 3795.0 + (220.0 / 600.0) * 20.0; // 3802.333 mm
    let expected_aft = 4190.0 - (220.0 / 600.0) * 20.0; // 4182.667 mm
    let expected_left = -80.0 + (220.0 / 600.0) * 2.0; // -79.2667 mm
    let expected_right = 80.0 - (220.0 / 600.0) * 2.0; // 79.2667 mm

    assert_relative_eq!(limits.fwd_limit.get(), expected_fwd, epsilon = 1e-4);
    assert_relative_eq!(limits.aft_limit.get(), expected_aft, epsilon = 1e-4);
    assert_relative_eq!(limits.left_limit.get(), expected_left, epsilon = 1e-4);
    assert_relative_eq!(limits.right_limit.get(), expected_right, epsilon = 1e-4);
}
