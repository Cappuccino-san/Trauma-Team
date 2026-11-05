//! Airbus Helicopters H145 / BK117 D-2 RFM Performance Dataset in Rust.

use crate::interpolator::{CruisePerformanceGrid3D, PerformanceGrid2D};
use crate::rfm::{
    CGEnvelopePoint, FuelSystemSpec, RFMAirframeSpec, StationArm, StationId, VariableFuelArmPoint,
};
use crate::units::{Celsius, Feet, KgPerHour, Kilograms, Knots, Millimeters};
use std::collections::HashMap;

/// Builds the Airbus H145 / BK117 D-2 RFM dataset.
pub fn get_h145_rfm_spec() -> RFMAirframeSpec {
    let mut stations = HashMap::new();

    stations.insert(
        StationId::PilotRh,
        StationArm::new(
            StationId::PilotRh,
            "Flight Crew RH Pilot Seat",
            Millimeters::new(2420.0),
            Millimeters::new(360.0),
            Kilograms::new(130.0),
            false,
        ),
    );
    stations.insert(
        StationId::CopilotLh,
        StationArm::new(
            StationId::CopilotLh,
            "Flight Crew LH Co-Pilot Seat",
            Millimeters::new(2420.0),
            Millimeters::new(-360.0),
            Kilograms::new(130.0),
            false,
        ),
    );
    stations.insert(
        StationId::CrewAttendantAftLh,
        StationArm::new(
            StationId::CrewAttendantAftLh,
            "Medical Attendant Aft Facing LH Seat",
            Millimeters::new(3350.0),
            Millimeters::new(-420.0),
            Kilograms::new(130.0),
            false,
        ),
    );
    stations.insert(
        StationId::CrewAttendantAftRh,
        StationArm::new(
            StationId::CrewAttendantAftRh,
            "Medical Attendant Forward Facing RH Seat",
            Millimeters::new(3350.0),
            Millimeters::new(420.0),
            Kilograms::new(130.0),
            false,
        ),
    );
    stations.insert(
        StationId::PrimaryPatientLitter,
        StationArm::new(
            StationId::PrimaryPatientLitter,
            "Primary Patient Litter (Stretcher Mount)",
            Millimeters::new(3820.0),
            Millimeters::new(-280.0),
            Kilograms::new(220.0),
            false,
        ),
    );
    stations.insert(
        StationId::SecondaryPatientLitter,
        StationArm::new(
            StationId::SecondaryPatientLitter,
            "Secondary Patient Litter",
            Millimeters::new(3820.0),
            Millimeters::new(280.0),
            Kilograms::new(150.0),
            false,
        ),
    );
    stations.insert(
        StationId::MedicalCabinetRack,
        StationArm::new(
            StationId::MedicalCabinetRack,
            "Aft Right Intensive Care Rack (ECMO/IABP Mount)",
            Millimeters::new(4280.0),
            Millimeters::new(380.0),
            Kilograms::new(120.0),
            false,
        ),
    );
    stations.insert(
        StationId::OxygenSystem,
        StationArm::new(
            StationId::OxygenSystem,
            "Fixed Medical Oxygen Bottles & Regulator",
            Millimeters::new(4600.0),
            Millimeters::new(-350.0),
            Kilograms::new(50.0),
            false,
        ),
    );
    stations.insert(
        StationId::AftAvionicsCargo,
        StationArm::new(
            StationId::AftAvionicsCargo,
            "Aft Clamshell Cargo Bay (Trauma/Rescue Bags)",
            Millimeters::new(4950.0),
            Millimeters::new(0.0),
            Kilograms::new(150.0),
            false,
        ),
    );
    stations.insert(
        StationId::RescueHoistRh,
        StationArm::new(
            StationId::RescueHoistRh,
            "External Goodrich Rescue Hoist Boom (Right Side)",
            Millimeters::new(3580.0),
            Millimeters::new(1050.0),
            Kilograms::new(272.0),
            false,
        ),
    );

    let fuel_arms = vec![
        VariableFuelArmPoint {
            fuel_mass: Kilograms::new(0.0),
            arm_long: Millimeters::new(3900.0),
            arm_lat: Millimeters::new(0.0),
        },
        VariableFuelArmPoint {
            fuel_mass: Kilograms::new(150.0),
            arm_long: Millimeters::new(3930.0),
            arm_lat: Millimeters::new(0.0),
        },
        VariableFuelArmPoint {
            fuel_mass: Kilograms::new(350.0),
            arm_long: Millimeters::new(3975.0),
            arm_lat: Millimeters::new(0.0),
        },
        VariableFuelArmPoint {
            fuel_mass: Kilograms::new(550.0),
            arm_long: Millimeters::new(4005.0),
            arm_lat: Millimeters::new(0.0),
        },
        VariableFuelArmPoint {
            fuel_mass: Kilograms::new(694.0),
            arm_long: Millimeters::new(4020.0),
            arm_lat: Millimeters::new(0.0),
        },
    ];

    let fuel_system = FuelSystemSpec {
        usable_fuel_capacity: Kilograms::new(694.0),
        unusable_fuel: Kilograms::new(9.0),
        tank_arms: fuel_arms,
        default_arm_long: Millimeters::new(4000.0),
        default_arm_lat: Millimeters::new(0.0),
    };

    let cg_envelope = vec![
        CGEnvelopePoint {
            gross_weight: Kilograms::new(2000.0),
            cg_long_fwd: Millimeters::new(3780.0),
            cg_long_aft: Millimeters::new(4200.0),
            cg_lat_left: Millimeters::new(-85.0),
            cg_lat_right: Millimeters::new(85.0),
        },
        CGEnvelopePoint {
            gross_weight: Kilograms::new(2800.0),
            cg_long_fwd: Millimeters::new(3795.0),
            cg_long_aft: Millimeters::new(4190.0),
            cg_lat_left: Millimeters::new(-80.0),
            cg_lat_right: Millimeters::new(80.0),
        },
        CGEnvelopePoint {
            gross_weight: Kilograms::new(3400.0),
            cg_long_fwd: Millimeters::new(3815.0),
            cg_long_aft: Millimeters::new(4170.0),
            cg_lat_left: Millimeters::new(-78.0),
            cg_lat_right: Millimeters::new(78.0),
        },
        CGEnvelopePoint {
            gross_weight: Kilograms::new(3700.0),
            cg_long_fwd: Millimeters::new(3830.0),
            cg_long_aft: Millimeters::new(4150.0),
            cg_lat_left: Millimeters::new(-75.0),
            cg_lat_right: Millimeters::new(75.0),
        },
    ];

    let pa_grid = vec![
        Feet::new(-2000.0),
        Feet::new(0.0),
        Feet::new(2000.0),
        Feet::new(4000.0),
        Feet::new(6000.0),
        Feet::new(8000.0),
        Feet::new(10000.0),
        Feet::new(12000.0),
        Feet::new(14000.0),
    ];

    let oat_grid = vec![
        Celsius::new(-20.0),
        Celsius::new(-10.0),
        Celsius::new(0.0),
        Celsius::new(10.0),
        Celsius::new(20.0),
        Celsius::new(30.0),
        Celsius::new(40.0),
        Celsius::new(50.0),
    ];

    let raw_ige = [
        [3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3700.0],
        [3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3650.0],
        [3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3650.0, 3550.0],
        [3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3660.0, 3550.0, 3420.0],
        [3700.0, 3700.0, 3700.0, 3700.0, 3680.0, 3540.0, 3400.0, 3260.0],
        [3700.0, 3700.0, 3700.0, 3640.0, 3510.0, 3370.0, 3230.0, 3090.0],
        [3680.0, 3620.0, 3540.0, 3450.0, 3320.0, 3180.0, 3040.0, 2900.0],
        [3500.0, 3420.0, 3330.0, 3220.0, 3100.0, 2960.0, 2820.0, 2690.0],
        [3300.0, 3210.0, 3100.0, 2980.0, 2860.0, 2730.0, 2590.0, 2460.0],
    ];

    let raw_oge = [
        [3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3600.0],
        [3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3620.0, 3480.0],
        [3700.0, 3700.0, 3700.0, 3700.0, 3690.0, 3580.0, 3450.0, 3310.0],
        [3700.0, 3700.0, 3700.0, 3660.0, 3530.0, 3400.0, 3260.0, 3130.0],
        [3700.0, 3680.0, 3600.0, 3480.0, 3350.0, 3210.0, 3070.0, 2940.0],
        [3620.0, 3520.0, 3410.0, 3290.0, 3160.0, 3020.0, 2880.0, 2750.0],
        [3440.0, 3330.0, 3210.0, 3090.0, 2960.0, 2820.0, 2680.0, 2550.0],
        [3240.0, 3120.0, 3000.0, 2870.0, 2740.0, 2610.0, 2470.0, 2340.0],
        [3020.0, 2900.0, 2770.0, 2640.0, 2510.0, 2380.0, 2250.0, 2120.0],
    ];

    let raw_oei = [
        [3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3700.0, 3600.0, 3450.0],
        [3700.0, 3700.0, 3700.0, 3700.0, 3650.0, 3520.0, 3380.0, 3230.0],
        [3700.0, 3700.0, 3660.0, 3530.0, 3390.0, 3250.0, 3110.0, 2970.0],
        [3650.0, 3540.0, 3410.0, 3280.0, 3140.0, 3000.0, 2860.0, 2720.0],
        [3460.0, 3340.0, 3210.0, 3070.0, 2930.0, 2790.0, 2650.0, 2510.0],
        [3250.0, 3130.0, 2990.0, 2850.0, 2710.0, 2570.0, 2430.0, 2290.0],
        [3030.0, 2900.0, 2760.0, 2620.0, 2480.0, 2340.0, 2200.0, 2060.0],
        [2800.0, 2670.0, 2530.0, 2390.0, 2250.0, 2110.0, 1970.0, 1830.0],
        [2560.0, 2420.0, 2280.0, 2140.0, 2000.0, 1860.0, 1720.0, 1580.0],
    ];

    let ige_weights: Vec<Vec<Kilograms>> = raw_ige
        .iter()
        .map(|row| row.iter().map(|&w| Kilograms::new(w)).collect())
        .collect();

    let oge_weights: Vec<Vec<Kilograms>> = raw_oge
        .iter()
        .map(|row| row.iter().map(|&w| Kilograms::new(w)).collect())
        .collect();

    let oei_weights: Vec<Vec<Kilograms>> = raw_oei
        .iter()
        .map(|row| row.iter().map(|&w| Kilograms::new(w)).collect())
        .collect();

    let ige_hover_grid = PerformanceGrid2D::new("H145 IGE Hover", pa_grid.clone(), oat_grid.clone(), ige_weights);
    let oge_hover_grid = PerformanceGrid2D::new("H145 OGE Hover", pa_grid.clone(), oat_grid.clone(), oge_weights);
    let oei_cat_a_grid = PerformanceGrid2D::new("H145 OEI Cat A", pa_grid.clone(), oat_grid.clone(), oei_weights);

    let gw_grid = vec![Kilograms::new(2500.0), Kilograms::new(3100.0), Kilograms::new(3700.0)];

    let mut tas_table = Vec::with_capacity(pa_grid.len());
    let mut ff_table = Vec::with_capacity(pa_grid.len());

    for pa in &pa_grid {
        let mut tas_oat = Vec::with_capacity(oat_grid.len());
        let mut ff_oat = Vec::with_capacity(oat_grid.len());
        for oat in &oat_grid {
            let mut tas_gw = Vec::with_capacity(gw_grid.len());
            let mut ff_gw = Vec::with_capacity(gw_grid.len());
            for gw in &gw_grid {
                let base_tas = 135.0 + (pa.get() / 2000.0) * 1.5 - ((gw.get() - 2500.0) / 1200.0) * 5.0 + (oat.get() - 15.0) * 0.1;
                let base_ff = 210.0 + ((gw.get() - 2500.0) / 1200.0) * 45.0 - (pa.get() / 5000.0) * 8.0 + (oat.get() / 50.0) * 6.0;
                tas_gw.push(Knots::new(base_tas));
                ff_gw.push(KgPerHour::new(base_ff));
            }
            tas_oat.push(tas_gw);
            ff_oat.push(ff_gw);
        }
        tas_table.push(tas_oat);
        ff_table.push(ff_oat);
    }

    let cruise_performance = CruisePerformanceGrid3D {
        pa_grid,
        oat_grid,
        gw_grid,
        tas_table,
        fuel_flow_table: ff_table,
    };

    RFMAirframeSpec {
        model_name: "Airbus H145 (BK117 D-2)",
        oem: "Airbus Helicopters",
        certification_basis: "14 CFR Part 29 Category A",
        empty_weight: Kilograms::new(2140.0),
        empty_cg_long: Millimeters::new(3910.0),
        empty_cg_lat: Millimeters::new(-5.0),
        mtow: Kilograms::new(3700.0),
        min_flight_weight: Kilograms::new(1900.0),
        fuel_system,
        stations,
        cg_envelope,
        ige_hover_grid,
        oge_hover_grid,
        oei_cat_a_grid,
        cruise_performance,
        startup_fuel: Kilograms::new(15.0),
        hover_fuel_flow_factor: 1.20,
        loiter_fuel_flow_factor: 0.85,
    }
}
