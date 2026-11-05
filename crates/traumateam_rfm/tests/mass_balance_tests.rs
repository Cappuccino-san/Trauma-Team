use approx::assert_relative_eq;
use traumateam_rfm::airframes::get_h145_rfm_spec;
use traumateam_rfm::mass_balance::compute_mass_and_balance;
use traumateam_rfm::rfm::{LimitingFactor, PayloadItem, StationId};
use traumateam_rfm::units::Kilograms;

#[test]
fn test_mass_balance_analytical_hand_calculation() {
    let h145 = get_h145_rfm_spec();

    let payloads = vec![
        PayloadItem::new("p1", "Pilot", StationId::PilotRh, Kilograms::new(85.0)),
        PayloadItem::new("p2", "Copilot", StationId::CopilotLh, Kilograms::new(80.0)),
        PayloadItem::new("p3", "Nurse", StationId::CrewAttendantAftLh, Kilograms::new(70.0)),
        PayloadItem::new("p4", "Medic", StationId::CrewAttendantAftRh, Kilograms::new(85.0)),
        PayloadItem::new("p5", "Med Rack", StationId::MedicalCabinetRack, Kilograms::new(60.0)),
    ];
    let fuel = Kilograms::new(500.0);

    let mb = compute_mass_and_balance(&h145, fuel, &payloads).expect("Calculation should succeed");

    // Hand-calculated expected values from Python reference test:
    // Expected GW = 3020.0 kg
    // Expected M_long = 11541500.0 kg*mm -> CG_long = 3821.6887 mm
    // Expected M_lat = 20200.0 kg*mm -> CG_lat = 6.6887 mm
    let expected_gw = 3020.0;
    let expected_cg_long = 11541500.0 / 3020.0;
    let expected_cg_lat = 20200.0 / 3020.0;

    assert_relative_eq!(mb.gross_weight.get(), expected_gw, epsilon = 1e-6);
    assert_relative_eq!(mb.cg_long.get(), expected_cg_long, epsilon = 1e-4);
    assert_relative_eq!(mb.cg_lat.get(), expected_cg_lat, epsilon = 1e-4);
    assert!(mb.is_overall_compliant);
    assert_eq!(mb.limiting_factor, LimitingFactor::None);
}

#[test]
fn test_lateral_cg_exceedance_detection() {
    let h145 = get_h145_rfm_spec();

    let payloads = vec![
        PayloadItem::new("p1", "Heavy Pilot", StationId::PilotRh, Kilograms::new(110.0)),
        PayloadItem::new("p2", "Heavy Attendant", StationId::CrewAttendantAftRh, Kilograms::new(110.0)),
        PayloadItem::new("p3", "Heavy Med Rack", StationId::MedicalCabinetRack, Kilograms::new(100.0)),
        PayloadItem::new("p4", "Rescue Hoist Load", StationId::RescueHoistRh, Kilograms::new(220.0)),
    ];
    let fuel = Kilograms::new(100.0);

    let mb = compute_mass_and_balance(&h145, fuel, &payloads).expect("Calculation should succeed");

    assert!(!mb.is_lat_cg_compliant);
    assert_eq!(mb.limiting_factor, LimitingFactor::LateralCgRightExceeded);
    assert!(mb.cg_lat > mb.right_limit);
}

#[test]
fn test_mtow_exceedance_detection() {
    let h145 = get_h145_rfm_spec();

    let payloads = vec![
        PayloadItem::new("p1", "Pilot", StationId::PilotRh, Kilograms::new(100.0)),
        PayloadItem::new("p2", "Copilot", StationId::CopilotLh, Kilograms::new(100.0)),
        PayloadItem::new("p3", "Crew LH", StationId::CrewAttendantAftLh, Kilograms::new(100.0)),
        PayloadItem::new("p4", "Crew RH", StationId::CrewAttendantAftRh, Kilograms::new(100.0)),
        PayloadItem::new("p5", "Patient", StationId::PrimaryPatientLitter, Kilograms::new(180.0)),
        PayloadItem::new("p6", "Patient 2", StationId::SecondaryPatientLitter, Kilograms::new(150.0)),
        PayloadItem::new("p7", "Cargo", StationId::AftAvionicsCargo, Kilograms::new(150.0)),
        PayloadItem::new("p8", "Rack", StationId::MedicalCabinetRack, Kilograms::new(120.0)),
        PayloadItem::new("p9", "Oxygen", StationId::OxygenSystem, Kilograms::new(50.0)),
    ];
    let fuel = Kilograms::new(694.0);

    let mb = compute_mass_and_balance(&h145, fuel, &payloads).expect("Calculation should succeed");

    assert!(!mb.is_gw_compliant);
    assert_eq!(mb.limiting_factor, LimitingFactor::MtowExceeded);
    assert!(mb.gross_weight > h145.mtow);
}

#[test]
fn test_forward_cg_exceedance_detection() {
    let h145 = get_h145_rfm_spec();

    // Heavy cockpit loading, zero cabin payload, 0 fuel -> CG shifts forward of limit
    let payloads = vec![
        PayloadItem::new("p1", "Heavy Pilot", StationId::PilotRh, Kilograms::new(130.0)),
        PayloadItem::new("p2", "Heavy Copilot", StationId::CopilotLh, Kilograms::new(130.0)),
    ];
    let fuel = Kilograms::new(0.0);

    let mb = compute_mass_and_balance(&h145, fuel, &payloads).expect("Calculation should succeed");

    // GW = 2140 + 260 = 2400 kg
    // M_long = 2140 * 3910 + 260 * 2420 = 8,996,600 kg*mm
    // CG_long = 8,996,600 / 2400 = 3748.58 mm
    // Fwd limit at 2400 kg is 3787.5 mm -> CG is ~39 mm forward of boundary!
    assert!(!mb.is_long_cg_compliant);
    assert_eq!(mb.limiting_factor, LimitingFactor::LongitudinalCgForwardExceeded);
    assert!(mb.cg_long < mb.fwd_limit);
}

#[test]
fn test_aft_cg_exceedance_detection() {
    let mut h145 = get_h145_rfm_spec();

    // Add an extreme aft boom station to test aft envelope excursion
    let aft_station_id = StationId::Custom("EXTENDED_AFT_TAIL_LOAD");
    h145.stations.insert(
        aft_station_id,
        traumateam_rfm::StationArm::new(
            aft_station_id,
            "Extended Aft Tail Loading Platform",
            traumateam_rfm::Millimeters::new(7000.0),
            traumateam_rfm::Millimeters::new(0.0),
            Kilograms::new(500.0),
            false,
        ),
    );

    let payloads = vec![
        PayloadItem::new("p1", "Light Pilot", StationId::PilotRh, Kilograms::new(50.0)),
        PayloadItem::new("p2", "Heavy Aft Load", aft_station_id, Kilograms::new(350.0)),
        PayloadItem::new("p3", "Aft Cargo", StationId::AftAvionicsCargo, Kilograms::new(150.0)),
    ];
    let fuel = Kilograms::new(0.0);

    let mb = compute_mass_and_balance(&h145, fuel, &payloads).expect("Calculation should succeed");

    assert!(!mb.is_long_cg_compliant);
    assert_eq!(mb.limiting_factor, LimitingFactor::LongitudinalCgAftExceeded);
    assert!(mb.cg_long > mb.aft_limit);
}

#[test]
fn test_lateral_cg_left_exceedance_detection() {
    let mut h145 = get_h145_rfm_spec();

    // Add a left external rescue hoist / searchlight arm
    let left_hoist_id = StationId::Custom("RESCUE_HOIST_LH");
    h145.stations.insert(
        left_hoist_id,
        traumateam_rfm::StationArm::new(
            left_hoist_id,
            "LH Side Rescue Winch Boom",
            traumateam_rfm::Millimeters::new(3580.0),
            traumateam_rfm::Millimeters::new(-1100.0),
            Kilograms::new(300.0),
            false,
        ),
    );

    let payloads = vec![
        PayloadItem::new("p1", "LH Pilot", StationId::CopilotLh, Kilograms::new(110.0)),
        PayloadItem::new("p2", "LH Attendant", StationId::CrewAttendantAftLh, Kilograms::new(110.0)),
        PayloadItem::new("p3", "LH Hoist Load", left_hoist_id, Kilograms::new(250.0)),
    ];
    let fuel = Kilograms::new(100.0);

    let mb = compute_mass_and_balance(&h145, fuel, &payloads).expect("Calculation should succeed");

    assert!(!mb.is_lat_cg_compliant);
    assert_eq!(mb.limiting_factor, LimitingFactor::LateralCgLeftExceeded);
    assert!(mb.cg_lat < mb.left_limit);
}

#[test]
fn test_min_weight_violation_detection() {
    let mut airframe = get_h145_rfm_spec();
    // Simulate an airframe whose empty weight is set below certified minimum flight weight
    airframe.empty_weight = Kilograms::new(1800.0);
    airframe.min_flight_weight = Kilograms::new(1900.0);

    let payloads: Vec<PayloadItem> = Vec::new();
    let fuel = Kilograms::new(0.0);

    let mb = compute_mass_and_balance(&airframe, fuel, &payloads).expect("Calculation should succeed");

    assert!(!mb.is_gw_compliant);
    assert_eq!(mb.limiting_factor, LimitingFactor::MinWeightViolation);
    assert!(mb.gross_weight < airframe.min_flight_weight);
}

#[test]
fn test_unknown_station_error() {
    let h145 = get_h145_rfm_spec();

    let payloads = vec![
        PayloadItem::new("bad", "Ghost Gear", StationId::Custom("NON_EXISTENT_STATION"), Kilograms::new(50.0)),
    ];
    let fuel = Kilograms::new(200.0);

    let result = compute_mass_and_balance(&h145, fuel, &payloads);
    assert!(matches!(result, Err(traumateam_rfm::MassBalanceError::StationNotFound(_))));
}

#[test]
fn test_zero_gross_weight_error() {
    let mut airframe = get_h145_rfm_spec();
    airframe.empty_weight = Kilograms::new(0.0);

    let payloads: Vec<PayloadItem> = Vec::new();
    let fuel = Kilograms::new(0.0);

    let result = compute_mass_and_balance(&airframe, fuel, &payloads);
    assert_eq!(result, Err(traumateam_rfm::MassBalanceError::ZeroGrossWeight));
}

#[test]
fn test_bell429_mass_balance_calculation() {
    let b429 = traumateam_rfm::get_bell429_rfm_spec();

    let payloads = vec![
        PayloadItem::new("p1", "Pilot", StationId::PilotRh, Kilograms::new(85.0)),
        PayloadItem::new("p2", "Copilot", StationId::CopilotLh, Kilograms::new(80.0)),
        PayloadItem::new("p3", "Nurse", StationId::CrewAttendantLh, Kilograms::new(75.0)),
        PayloadItem::new("p4", "Medic", StationId::CrewAttendantRh, Kilograms::new(80.0)),
    ];
    let fuel = Kilograms::new(300.0);

    let mb = compute_mass_and_balance(&b429, fuel, &payloads).expect("Calculation should succeed");

    // Hand calculation for Bell 429:
    // Empty: 2025 kg @ 3260 mm (long), -3 mm (lat)
    // Pilot: 85 kg @ 2180 mm, 330 mm
    // Copilot: 80 kg @ 2180 mm, -330 mm
    // Attendant LH: 75 kg @ 3100 mm, -400 mm
    // Attendant RH: 80 kg @ 3100 mm, 400 mm
    // Fuel: 300 kg -> fuel arm interpolated between 200kg (3340mm) and 400kg (3360mm) = 3350 mm, lat = 0.0 mm
    // Total GW = 2025 + 85 + 80 + 75 + 80 + 300 = 2645.0 kg
    // M_long = 2025*3260 + 85*2180 + 80*2180 + 75*3100 + 80*3100 + 300*3350
    //        = 6601500 + 185300 + 174400 + 232500 + 248000 + 1005000 = 8446700 kg*mm
    // Expected CG_long = 8446700 / 2645 = 3193.4594 mm
    // M_lat = 2025*(-3) + 85*330 + 80*(-330) + 75*(-400) + 80*400 + 0 = -2425 kg*mm
    // Expected CG_lat = -2425 / 2645 = -0.9168 mm

    let expected_gw = 2645.0;
    let expected_cg_long = 8446700.0 / 2645.0;
    let expected_cg_lat = -2425.0 / 2645.0;

    assert_relative_eq!(mb.gross_weight.get(), expected_gw, epsilon = 1e-5);
    assert_relative_eq!(mb.cg_long.get(), expected_cg_long, epsilon = 1e-4);
    assert_relative_eq!(mb.cg_lat.get(), expected_cg_lat, epsilon = 1e-4);
    assert!(mb.is_overall_compliant);
    assert_eq!(mb.limiting_factor, LimitingFactor::None);
}

#[test]
fn test_nan_payload_weight_returns_error() {
    let h145 = get_h145_rfm_spec();
    let payloads = vec![
        PayloadItem::new("bad", "Corrupted Sensor", StationId::PilotRh, Kilograms::new(f64::NAN)),
    ];
    let fuel = Kilograms::new(200.0);

    let result = compute_mass_and_balance(&h145, fuel, &payloads);
    assert_eq!(result, Err(traumateam_rfm::MassBalanceError::ZeroGrossWeight));
}

#[test]
fn test_nan_fuel_weight_returns_error() {
    let h145 = get_h145_rfm_spec();
    let result = compute_mass_and_balance(&h145, Kilograms::new(f64::NAN), &[]);
    assert_eq!(result, Err(traumateam_rfm::MassBalanceError::ZeroGrossWeight));
}

#[test]
fn test_nan_empty_weight_returns_error() {
    let mut h145 = get_h145_rfm_spec();
    h145.empty_weight = Kilograms::new(f64::NAN);

    let result = compute_mass_and_balance(&h145, Kilograms::new(200.0), &[]);
    assert_eq!(result, Err(traumateam_rfm::MassBalanceError::ZeroGrossWeight));
}
