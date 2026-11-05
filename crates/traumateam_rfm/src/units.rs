//! Zero-Cost Dimensional Units for Rotorcraft Flight Mechanics.
//!
//! Provides `#[repr(transparent)]` newtypes wrapping IEEE-754 `f64` values.
//! Calculations enforce strict dimensional correctness at compile time with zero runtime overhead.

use std::fmt;
use std::ops::{Add, AddAssign, Div, DivAssign, Mul, MulAssign, Neg, Sub, SubAssign};

macro_rules! define_unit {
    ($name:ident, $doc:expr) => {
        #[doc = $doc]
        #[repr(transparent)]
        #[derive(Copy, Clone, Default, PartialEq, PartialOrd)]
        #[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
        #[cfg_attr(feature = "serde", serde(transparent))]
        pub struct $name(pub f64);

        impl $name {
            #[inline(always)]
            pub const fn new(val: f64) -> Self {
                Self(val)
            }

            #[inline(always)]
            pub const fn get(self) -> f64 {
                self.0
            }
        }

        impl fmt::Debug for $name {
            fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                write!(f, "{}({:.3})", stringify!($name), self.0)
            }
        }

        impl fmt::Display for $name {
            fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                write!(f, "{:.2}", self.0)
            }
        }

        impl Add for $name {
            type Output = Self;
            #[inline(always)]
            fn add(self, rhs: Self) -> Self {
                Self(self.0 + rhs.0)
            }
        }

        impl AddAssign for $name {
            #[inline(always)]
            fn add_assign(&mut self, rhs: Self) {
                self.0 += rhs.0;
            }
        }

        impl Sub for $name {
            type Output = Self;
            #[inline(always)]
            fn sub(self, rhs: Self) -> Self {
                Self(self.0 - rhs.0)
            }
        }

        impl SubAssign for $name {
            #[inline(always)]
            fn sub_assign(&mut self, rhs: Self) {
                self.0 -= rhs.0;
            }
        }

        impl Neg for $name {
            type Output = Self;
            #[inline(always)]
            fn neg(self) -> Self {
                Self(-self.0)
            }
        }

        impl Mul<f64> for $name {
            type Output = Self;
            #[inline(always)]
            fn mul(self, rhs: f64) -> Self {
                Self(self.0 * rhs)
            }
        }

        impl Mul<$name> for f64 {
            type Output = $name;
            #[inline(always)]
            fn mul(self, rhs: $name) -> $name {
                $name(self * rhs.0)
            }
        }

        impl MulAssign<f64> for $name {
            #[inline(always)]
            fn mul_assign(&mut self, rhs: f64) {
                self.0 *= rhs;
            }
        }

        impl Div<f64> for $name {
            type Output = Self;
            #[inline(always)]
            fn div(self, rhs: f64) -> Self {
                Self(self.0 / rhs)
            }
        }

        impl DivAssign<f64> for $name {
            #[inline(always)]
            fn div_assign(&mut self, rhs: f64) {
                self.0 /= rhs;
            }
        }

        impl Div<$name> for $name {
            type Output = f64;
            #[inline(always)]
            fn div(self, rhs: Self) -> f64 {
                self.0 / rhs.0
            }
        }
    };
}

define_unit!(Kilograms, "Mass in kilograms (kg).");
define_unit!(Millimeters, "Station coordinate arm in millimeters (mm).");
define_unit!(KilogramMillimeters, "Static moment in kilogram-millimeters (kg·mm).");
define_unit!(Feet, "Altitude or elevation in feet (ft).");
define_unit!(Celsius, "Temperature in degrees Celsius (°C).");
define_unit!(Knots, "Airspeed or groundspeed in nautical miles per hour (kt).");
define_unit!(KgPerHour, "Fuel mass consumption rate in kg/hr.");

// Cross-dimensional physical operator semantics
impl Mul<Millimeters> for Kilograms {
    type Output = KilogramMillimeters;
    #[inline(always)]
    fn mul(self, rhs: Millimeters) -> KilogramMillimeters {
        KilogramMillimeters(self.0 * rhs.0)
    }
}

impl Mul<Kilograms> for Millimeters {
    type Output = KilogramMillimeters;
    #[inline(always)]
    fn mul(self, rhs: Kilograms) -> KilogramMillimeters {
        KilogramMillimeters(self.0 * rhs.0)
    }
}

impl Div<Kilograms> for KilogramMillimeters {
    type Output = Millimeters;
    #[inline(always)]
    fn div(self, rhs: Kilograms) -> Millimeters {
        Millimeters(self.0 / rhs.0)
    }
}

impl Div<Millimeters> for KilogramMillimeters {
    type Output = Kilograms;
    #[inline(always)]
    fn div(self, rhs: Millimeters) -> Kilograms {
        Kilograms(self.0 / rhs.0)
    }
}

// Unit conversion constants
pub const LBS_TO_KG: f64 = 0.45359237;
pub const KG_TO_LBS: f64 = 1.0 / LBS_TO_KG;
pub const INCHES_TO_MM: f64 = 25.4;
pub const MM_TO_INCHES: f64 = 1.0 / INCHES_TO_MM;
pub const FEET_TO_METERS: f64 = 0.3048;
pub const METERS_TO_FEET: f64 = 3.280839895013123;
pub const NM_TO_METERS: f64 = 1852.0;
pub const METERS_TO_NM: f64 = 1.0 / NM_TO_METERS;
pub const NM_TO_FEET: f64 = NM_TO_METERS * METERS_TO_FEET;
pub const SM_TO_FEET: f64 = 5280.0;
pub const SM_TO_NM: f64 = SM_TO_FEET / NM_TO_FEET;
pub const STD_SEA_LEVEL_PRESSURE_HPA: f64 = 1013.25;
pub const STD_SEA_LEVEL_PRESSURE_INHG: f64 = 29.92126;
pub const INHG_TO_HPA: f64 = 33.863886666667;
pub const HPA_TO_INHG: f64 = 1.0 / INHG_TO_HPA;

// Jet-A1 standard fuel density at 15 deg C (0.804 kg/L = 6.709 lb/US gal)
pub const JET_A_STD_DENSITY_KG_PER_L: f64 = 0.804;
pub const JET_A_THERMAL_EXPANSION_COEFF: f64 = 0.00095;
pub const US_GAL_TO_LITERS: f64 = 3.785411784;
pub const LITERS_TO_US_GAL: f64 = 1.0 / US_GAL_TO_LITERS;

impl Kilograms {
    #[inline(always)]
    pub fn from_lbs(lbs: f64) -> Self {
        Self(lbs * LBS_TO_KG)
    }

    #[inline(always)]
    pub fn to_lbs(self) -> f64 {
        self.0 * KG_TO_LBS
    }
}

impl Millimeters {
    #[inline(always)]
    pub fn from_inches(inches: f64) -> Self {
        Self(inches * INCHES_TO_MM)
    }

    #[inline(always)]
    pub fn to_inches(self) -> f64 {
        self.0 * MM_TO_INCHES
    }
}

impl KilogramMillimeters {
    #[inline(always)]
    pub fn from_in_lb(in_lb: f64) -> Self {
        Self(in_lb * LBS_TO_KG * INCHES_TO_MM)
    }

    #[inline(always)]
    pub fn to_in_lb(self) -> f64 {
        self.0 * KG_TO_LBS * MM_TO_INCHES
    }
}

impl Feet {
    #[inline(always)]
    pub fn from_meters(meters: f64) -> Self {
        Self(meters * METERS_TO_FEET)
    }

    #[inline(always)]
    pub fn to_meters(self) -> f64 {
        self.0 * FEET_TO_METERS
    }
}

impl Celsius {
    #[inline(always)]
    pub fn from_fahrenheit(f: f64) -> Self {
        Self((f - 32.0) * 5.0 / 9.0)
    }

    #[inline(always)]
    pub fn to_fahrenheit(self) -> f64 {
        self.0 * 9.0 / 5.0 + 32.0
    }

    #[inline(always)]
    pub fn to_kelvin(self) -> f64 {
        self.0 + 273.15
    }

    #[inline(always)]
    pub fn from_kelvin(k: f64) -> Self {
        Self(k - 273.15)
    }
}

/// Temperature-corrected Jet-A1 fuel density in kg/L.
/// rho(T) = rho_15 * [1 - beta * (T - 15)]
#[inline]
pub fn jet_a_density_kg_per_l(temp: Celsius) -> f64 {
    JET_A_STD_DENSITY_KG_PER_L * (1.0 - JET_A_THERMAL_EXPANSION_COEFF * (temp.get() - 15.0))
}

/// Converts Jet-A fuel volume in liters to mass in kilograms.
#[inline]
pub fn liters_to_jet_a_kg(liters: f64, temp: Celsius) -> Kilograms {
    Kilograms::new(liters * jet_a_density_kg_per_l(temp))
}

/// Converts Jet-A fuel volume in US gallons to mass in kilograms.
#[inline]
pub fn gallons_to_jet_a_kg(gallons: f64, temp: Celsius) -> Kilograms {
    liters_to_jet_a_kg(gallons * US_GAL_TO_LITERS, temp)
}

/// Converts Jet-A fuel mass in kilograms to volume in US gallons.
#[inline]
pub fn jet_a_kg_to_gallons(mass: Kilograms, temp: Celsius) -> f64 {
    let liters = mass.get() / jet_a_density_kg_per_l(temp);
    liters * LITERS_TO_US_GAL
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_unit_operator_overloads() {
        let mass = Kilograms::new(100.0);
        let arm = Millimeters::new(2500.0);
        let moment: KilogramMillimeters = mass * arm;
        assert_eq!(moment.get(), 250000.0);

        let recovered_arm = moment / mass;
        assert_eq!(recovered_arm.get(), 2500.0);

        let recovered_mass = moment / arm;
        assert_eq!(recovered_mass.get(), 100.0);
    }

    #[test]
    fn test_unit_conversions() {
        let lbs = 220.462262;
        let kg = Kilograms::from_lbs(lbs);
        assert!((kg.get() - 100.0).abs() < 1e-4);

        let inches = 100.0;
        let mm = Millimeters::from_inches(inches);
        assert_eq!(mm.get(), 2540.0);

        let meters = 1000.0;
        let ft = Feet::from_meters(meters);
        assert!((ft.to_meters() - meters).abs() < 1e-6);

        let c = Celsius::new(20.0);
        let f = c.to_fahrenheit();
        assert_eq!(f, 68.0);
        assert_eq!(Celsius::from_fahrenheit(f).get(), 20.0);
        assert_eq!(c.to_kelvin(), 293.15);
        assert_eq!(Celsius::from_kelvin(293.15).get(), 20.0);
    }

    #[test]
    fn test_jet_a_fuel_density_and_volume_conversions() {
        let temp_std = Celsius::new(15.0);
        let density_std = jet_a_density_kg_per_l(temp_std);
        assert!((density_std - 0.804).abs() < 1e-6);

        let liters = 100.0;
        let mass = liters_to_jet_a_kg(liters, temp_std);
        assert!((mass.get() - 80.4).abs() < 1e-4);

        let gal = jet_a_kg_to_gallons(mass, temp_std);
        assert!((gal - 100.0 * LITERS_TO_US_GAL).abs() < 1e-4);

        let mass_from_gal = gallons_to_jet_a_kg(gal, temp_std);
        assert!((mass_from_gal.get() - mass.get()).abs() < 1e-4);

        // Warm fuel is less dense
        let temp_hot = Celsius::new(35.0);
        let density_hot = jet_a_density_kg_per_l(temp_hot);
        assert!(density_hot < density_std);
    }
}
