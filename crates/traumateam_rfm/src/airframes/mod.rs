//! Certified OEM Rotorcraft Flight Manual (RFM) Airframe Datasets.

pub mod bell429;
pub mod h145;

pub use bell429::get_bell429_rfm_spec;
pub use h145::get_h145_rfm_spec;
