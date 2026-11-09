//! Bell 429 GlobalRanger RFM Performance Dataset in Rust.

use crate::interpolator::{CruisePerformanceGrid3D, PerformanceGrid2D};
use crate::rfm::{
    CGEnvelopePoint, FuelSystemSpec, RFMAirframeSpec, StationArm, StationId, VariableFuelArmPoint,
};
use crate::units::{Celsius, Feet, KgPerHour, Kilograms, Knots, Millimeters};
use std::collections::HashMap;

/// Builds the Bell 429 GlobalRanger RFM dataset.
pub fn get_bell429_rfm_spec() -> RFMAirframeSpec {
    let mut stations = HashMap::new();

    stations.insert(
        StationId::PilotRh,
        StationArm::new(
            StationId::PilotRh,
            "Flight Crew RH Pilot Seat",
            Millimeters::new(2180.0),
            Millimeters::new(330.0),
            Kilograms::new(130.0),
            false,
        ),
    );
    stations.insert(
        StationId::CopilotLh,
        StationArm::new(
            StationId::CopilotLh,
            "Flight Crew LH Co-Pilot Seat",
            Millimeters::new(2180.0),
            Millimeters::new(-330.0),
            Kilograms::new(130.0),
            false,
        ),
    );
    stations.insert(
        StationId::CrewAttendantLh,
        StationArm::new(
            StationId::CrewAttendantLh,
            "Medical Attendant LH Seat",
            Millimeters::new(3100.0),
            Millimeters::new(-400.0),
            Kilograms::new(130.0),
            false,
        ),
    );
    stations.insert(
        StationId::CrewAttendantRh,
        StationArm::new(
            StationId::CrewAttendantRh,
            "Medical Attendant RH Seat",
            Millimeters::new(3100.0),
            Millimeters::new(400.0),
            Kilograms::new(130.0),
            false,
        ),
    );
    stations.insert(
        StationId::PrimaryPatientLitter,
        StationArm::new(
            StationId::PrimaryPatientLitter,
            "Patient Litter Deck",
            Millimeters::new(3550.0),
            Millimeters::new(-250.0),
            Kilograms::new(200.0),
            false,
        ),
    );
    stations.insert(
        StationId::MedicalEquipmentRack,
        StationArm::new(
            StationId::MedicalEquipmentRack,
            "Intensive Care Unit Equipment Rack",
            Millimeters::new(3900.0),
            Millimeters::new(350.0),
            Kilograms::new(100.0),
            false,
        ),
    );
    stations.insert(
        StationId::AftBaggage,
        StationArm::new(
            StationId::AftBaggage,
            "Aft Baggage Compartment",
            Millimeters::new(4650.0),
            Millimeters::new(0.0),
            Kilograms::new(180.0),
            false,
        ),
    );

    let fuel_arms = vec![
        VariableFuelArmPoint {
            fuel_mass: Kilograms::new(0.0),
            arm_long: Millimeters::new(3320.0),
            arm_lat: Millimeters::new(0.0),
        },
        VariableFuelArmPoint {
            fuel_mass: Kilograms::new(200.0),
            arm_long: Millimeters::new(3340.0),
            arm_lat: Millimeters::new(0.0),
        },
        VariableFuelArmPoint {
            fuel_mass: Kilograms::new(400.0),
            arm_long: Millimeters::new(3360.0),
            arm_lat: Millimeters::new(0.0),
        },
        VariableFuelArmPoint {
            fuel_mass: Kilograms::new(590.0),
            arm_long: Millimeters::new(3380.0),
            arm_lat: Millimeters::new(0.0),
        },
    ];

    let fuel_system = FuelSystemSpec {
        usable_fuel_capacity: Kilograms::new(590.0),
        unusable_fuel: Kilograms::new(6.0),
        tank_arms: fuel_arms,
        default_arm_long: Millimeters::new(3350.0),
        default_arm_lat: Millimeters::new(0.0),
    };

    let cg_envelope = vec![
        CGEnvelopePoint {
            gross_weight: Kilograms::new(1900.0),
            cg_long_fwd: Millimeters::new(3120.0),
            cg_long_aft: Millimeters::new(3480.0),
            cg_lat_left: Millimeters::new(-70.0),
            cg_lat_right: Millimeters::new(70.0),
        },
        CGEnvelopePoint {
            gross_weight: Kilograms::new(2600.0),
            cg_long_fwd: Millimeters::new(3140.0),
            cg_long_aft: Millimeters::new(3470.0),
            cg_lat_left: Millimeters::new(-68.0),
            cg_lat_right: Millimeters::new(68.0),
        },
        CGEnvelopePoint {
            gross_weight: Kilograms::new(3175.0),
            cg_long_fwd: Millimeters::new(3170.0),
            cg_long_aft: Millimeters::new(3450.0),
            cg_lat_left: Millimeters::new(-65.0),
            cg_lat_right: Millimeters::new(65.0),
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
        [
            3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3175.0,
        ],
        [
            3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3120.0,
        ],
        [
            3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3110.0, 3010.0,
        ],
        [
            3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3100.0, 3000.0, 2880.0,
        ],
        [
            3175.0, 3175.0, 3175.0, 3150.0, 3090.0, 2980.0, 2860.0, 2740.0,
        ],
        [
            3175.0, 3175.0, 3140.0, 3060.0, 2950.0, 2830.0, 2700.0, 2580.0,
        ],
        [
            3110.0, 3050.0, 2970.0, 2880.0, 2770.0, 2640.0, 2510.0, 2390.0,
        ],
        [
            2930.0, 2860.0, 2780.0, 2680.0, 2570.0, 2440.0, 2320.0, 2200.0,
        ],
    ];

    let raw_oge = [
        [
            3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3080.0,
        ],
        [
            3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3070.0, 2950.0,
        ],
        [
            3175.0, 3175.0, 3175.0, 3175.0, 3120.0, 3020.0, 2900.0, 2780.0,
        ],
        [
            3175.0, 3175.0, 3175.0, 3080.0, 2970.0, 2850.0, 2730.0, 2610.0,
        ],
        [
            3175.0, 3110.0, 3030.0, 2920.0, 2800.0, 2680.0, 2560.0, 2440.0,
        ],
        [
            3050.0, 2960.0, 2860.0, 2750.0, 2630.0, 2500.0, 2380.0, 2260.0,
        ],
        [
            2880.0, 2790.0, 2680.0, 2570.0, 2450.0, 2320.0, 2200.0, 2080.0,
        ],
        [
            2700.0, 2600.0, 2500.0, 2380.0, 2260.0, 2140.0, 2010.0, 1900.0,
        ],
    ];

    let raw_oei = [
        [
            3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3175.0, 3050.0, 2910.0,
        ],
        [
            3175.0, 3175.0, 3175.0, 3175.0, 3090.0, 2970.0, 2840.0, 2710.0,
        ],
        [
            3175.0, 3175.0, 3080.0, 2960.0, 2830.0, 2710.0, 2580.0, 2460.0,
        ],
        [
            3070.0, 2970.0, 2850.0, 2730.0, 2600.0, 2480.0, 2360.0, 2230.0,
        ],
        [
            2890.0, 2780.0, 2660.0, 2530.0, 2410.0, 2290.0, 2160.0, 2040.0,
        ],
        [
            2690.0, 2580.0, 2460.0, 2330.0, 2210.0, 2080.0, 1960.0, 1840.0,
        ],
        [
            2490.0, 2380.0, 2250.0, 2120.0, 2000.0, 1870.0, 1750.0, 1630.0,
        ],
        [
            2280.0, 2160.0, 2040.0, 1910.0, 1790.0, 1660.0, 1540.0, 1420.0,
        ],
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

    let ige_hover_grid = PerformanceGrid2D::new(
        "Bell 429 IGE Hover",
        pa_grid.clone(),
        oat_grid.clone(),
        ige_weights,
    );
    let oge_hover_grid = PerformanceGrid2D::new(
        "Bell 429 OGE Hover",
        pa_grid.clone(),
        oat_grid.clone(),
        oge_weights,
    );
    let oei_cat_a_grid = PerformanceGrid2D::new(
        "Bell 429 OEI Cat A",
        pa_grid.clone(),
        oat_grid.clone(),
        oei_weights,
    );

    let gw_grid = vec![
        Kilograms::new(2200.0),
        Kilograms::new(2700.0),
        Kilograms::new(3175.0),
    ];

    let mut tas_table = Vec::with_capacity(pa_grid.len());
    let mut ff_table = Vec::with_capacity(pa_grid.len());

    for pa in &pa_grid {
        let mut tas_oat = Vec::with_capacity(oat_grid.len());
        let mut ff_oat = Vec::with_capacity(oat_grid.len());
        for oat in &oat_grid {
            let mut tas_gw = Vec::with_capacity(gw_grid.len());
            let mut ff_gw = Vec::with_capacity(gw_grid.len());
            for gw in &gw_grid {
                let base_tas = 142.0 + (pa.get() / 2000.0) * 1.8
                    - ((gw.get() - 2200.0) / 975.0) * 6.0
                    + (oat.get() - 15.0) * 0.1;
                let base_ff = 195.0 + ((gw.get() - 2200.0) / 975.0) * 40.0
                    - (pa.get() / 5000.0) * 7.0
                    + (oat.get() / 50.0) * 5.0;
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
        model_name: "Bell 429 GlobalRanger",
        oem: "Bell Helicopter",
        certification_basis: "14 CFR Part 29 Category A",
        empty_weight: Kilograms::new(2025.0),
        empty_cg_long: Millimeters::new(3260.0),
        empty_cg_lat: Millimeters::new(-3.0),
        mtow: Kilograms::new(3175.0),
        min_flight_weight: Kilograms::new(1800.0),
        fuel_system,
        stations,
        cg_envelope,
        ige_hover_grid,
        oge_hover_grid,
        oei_cat_a_grid,
        cruise_performance,
        startup_fuel: Kilograms::new(12.0),
        hover_fuel_flow_factor: 1.18,
        loiter_fuel_flow_factor: 0.86,
    }
}
