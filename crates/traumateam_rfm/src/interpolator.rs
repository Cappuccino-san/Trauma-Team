//! Zero-Allocation Multi-Dimensional Linear Interpolator for RFM Charts.

use crate::units::{Celsius, Feet, KgPerHour, Kilograms, Knots};

/// 1D piecewise linear interpolation along monotonic sorted slice.
#[inline]
pub fn interp_1d(x: f64, xs: &[f64], ys: &[f64]) -> f64 {
    debug_assert!(xs.len() == ys.len() && !xs.is_empty());
    if xs.len() == 1 {
        return ys[0];
    }
    if x.is_nan() || x <= xs[0] {
        return ys[0];
    }
    let last = xs.len() - 1;
    if x >= xs[last] {
        return ys[last];
    }

    // Binary search for interval
    let idx = xs.partition_point(|&val| val <= x);
    let i0 = idx.saturating_sub(1);
    let i1 = idx.min(last);

    let x0 = xs[i0];
    let x1 = xs[i1];
    let y0 = ys[i0];
    let y1 = ys[i1];

    let dx = x1 - x0;
    let t = if dx == 0.0 { 0.0 } else { (x - x0) / dx };
    y0 + t * (y1 - y0)
}

/// 2D Bilinear Grid Interpolator (e.g. PA x OAT -> Max Hover Gross Weight).
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct PerformanceGrid2D {
    pub name: &'static str,
    pub pa_grid: Vec<Feet>,
    pub oat_grid: Vec<Celsius>,
    pub weight_table: Vec<Vec<Kilograms>>, // [pa_idx][oat_idx]
}

impl PerformanceGrid2D {
    pub fn new(
        name: &'static str,
        pa_grid: Vec<Feet>,
        oat_grid: Vec<Celsius>,
        weight_table: Vec<Vec<Kilograms>>,
    ) -> Self {
        assert_eq!(weight_table.len(), pa_grid.len());
        for row in &weight_table {
            assert_eq!(row.len(), oat_grid.len());
        }
        Self {
            name,
            pa_grid,
            oat_grid,
            weight_table,
        }
    }

    /// Evaluates the 2D grid using bilinear interpolation with zero heap allocations.
    pub fn interpolate(&self, pa: Feet, oat: Celsius) -> Kilograms {
        let pa_val = pa.get();
        let oat_val = oat.get();

        let (p0_idx, p1_idx, pt) = find_interval_f64(pa_val, &self.pa_grid);
        let (o0_idx, o1_idx, ot) = find_interval_f64(oat_val, &self.oat_grid);

        let q11 = self.weight_table[p0_idx][o0_idx].get();
        let q12 = self.weight_table[p0_idx][o1_idx].get();
        let q21 = self.weight_table[p1_idx][o0_idx].get();
        let q22 = self.weight_table[p1_idx][o1_idx].get();

        // Bilinear interpolation
        let r1 = q11 + ot * (q12 - q11);
        let r2 = q21 + ot * (q22 - q21);
        let result = r1 + pt * (r2 - r1);

        Kilograms::new(result)
    }
}

/// 3D Trilinear Cruise Grid Interpolator (PA x OAT x GW -> TAS and Fuel Flow).
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct CruisePerformanceGrid3D {
    pub pa_grid: Vec<Feet>,
    pub oat_grid: Vec<Celsius>,
    pub gw_grid: Vec<Kilograms>,
    pub tas_table: Vec<Vec<Vec<Knots>>>,
    pub fuel_flow_table: Vec<Vec<Vec<KgPerHour>>>,
}

impl CruisePerformanceGrid3D {
    pub fn new(
        pa_grid: Vec<Feet>,
        oat_grid: Vec<Celsius>,
        gw_grid: Vec<Kilograms>,
        tas_table: Vec<Vec<Vec<Knots>>>,
        fuel_flow_table: Vec<Vec<Vec<KgPerHour>>>,
    ) -> Self {
        assert_eq!(tas_table.len(), pa_grid.len());
        assert_eq!(fuel_flow_table.len(), pa_grid.len());
        for p in 0..pa_grid.len() {
            assert_eq!(tas_table[p].len(), oat_grid.len());
            assert_eq!(fuel_flow_table[p].len(), oat_grid.len());
            for o in 0..oat_grid.len() {
                assert_eq!(tas_table[p][o].len(), gw_grid.len());
                assert_eq!(fuel_flow_table[p][o].len(), gw_grid.len());
            }
        }
        Self {
            pa_grid,
            oat_grid,
            gw_grid,
            tas_table,
            fuel_flow_table,
        }
    }

    pub fn interpolate_tas(&self, pa: Feet, oat: Celsius, gw: Kilograms) -> Knots {
        let val =
            self.trilinear_interp(pa.get(), oat.get(), gw.get(), &self.tas_table, |k| k.get());
        Knots::new(val)
    }

    pub fn interpolate_fuel_flow(&self, pa: Feet, oat: Celsius, gw: Kilograms) -> KgPerHour {
        let val =
            self.trilinear_interp(pa.get(), oat.get(), gw.get(), &self.fuel_flow_table, |ff| {
                ff.get()
            });
        KgPerHour::new(val)
    }

    fn trilinear_interp<T: Copy>(
        &self,
        pa_val: f64,
        oat_val: f64,
        gw_val: f64,
        table: &[Vec<Vec<T>>],
        extract: impl Fn(T) -> f64,
    ) -> f64 {
        let (p0, p1, xd) = find_interval_f64(pa_val, &self.pa_grid);
        let (o0, o1, yd) = find_interval_f64(oat_val, &self.oat_grid);
        let (g0, g1, zd) = find_interval_f64(gw_val, &self.gw_grid);

        let c000 = extract(table[p0][o0][g0]);
        let c001 = extract(table[p0][o0][g1]);
        let c010 = extract(table[p0][o1][g0]);
        let c011 = extract(table[p0][o1][g1]);
        let c100 = extract(table[p1][o0][g0]);
        let c101 = extract(table[p1][o0][g1]);
        let c110 = extract(table[p1][o1][g0]);
        let c111 = extract(table[p1][o1][g1]);

        let c00 = c000 * (1.0 - xd) + c100 * xd;
        let c01 = c001 * (1.0 - xd) + c101 * xd;
        let c10 = c010 * (1.0 - xd) + c110 * xd;
        let c11 = c011 * (1.0 - xd) + c111 * xd;

        let c0 = c00 * (1.0 - yd) + c10 * yd;
        let c1 = c01 * (1.0 - yd) + c11 * yd;

        c0 * (1.0 - zd) + c1 * zd
    }
}

/// Helper to find indices and fraction t along a slice of transparent newtypes.
#[inline]
fn find_interval_f64<U>(val: f64, grid: &[U]) -> (usize, usize, f64)
where
    U: Copy + Into<f64>,
{
    let n = grid.len();
    debug_assert!(n >= 2);
    let first: f64 = grid[0].into();
    let last: f64 = grid[n - 1].into();

    // Early guard: catch NaN and lower-bound out-of-range in O(1)
    if val.is_nan() || val <= first {
        return (0, 0, 0.0);
    }
    // Upper-bound out-of-range in O(1)
    if val >= last {
        return (n - 1, n - 1, 0.0);
    }

    let idx = grid.partition_point(|&item| item.into() <= val);
    // Defense-in-depth: saturating_sub prevents underflow under all circumstances
    let i0 = idx.saturating_sub(1);
    let i1 = idx.min(n - 1);
    let v0: f64 = grid[i0].into();
    let v1: f64 = grid[i1].into();

    let dv = v1 - v0;
    let t = if dv == 0.0 { 0.0 } else { (val - v0) / dv };
    (i0, i1, t)
}

impl From<Feet> for f64 {
    #[inline(always)]
    fn from(u: Feet) -> f64 {
        u.get()
    }
}

impl From<Celsius> for f64 {
    #[inline(always)]
    fn from(u: Celsius) -> f64 {
        u.get()
    }
}

impl From<Kilograms> for f64 {
    #[inline(always)]
    fn from(u: Kilograms) -> f64 {
        u.get()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_interp_1d_behavior() {
        let xs = [0.0, 10.0, 20.0, 30.0];
        let ys = [100.0, 200.0, 300.0, 400.0];

        // Exact points
        assert_eq!(interp_1d(0.0, &xs, &ys), 100.0);
        assert_eq!(interp_1d(10.0, &xs, &ys), 200.0);
        assert_eq!(interp_1d(30.0, &xs, &ys), 400.0);

        // Interpolated intermediate points
        assert_eq!(interp_1d(5.0, &xs, &ys), 150.0);
        assert_eq!(interp_1d(25.0, &xs, &ys), 350.0);

        // Boundary clamping
        assert_eq!(interp_1d(-10.0, &xs, &ys), 100.0);
        assert_eq!(interp_1d(50.0, &xs, &ys), 400.0);

        // Single point slice
        assert_eq!(interp_1d(5.0, &[10.0], &[50.0]), 50.0);
    }

    #[test]
    fn test_performance_grid_2d() {
        let pa_grid = vec![Feet::new(0.0), Feet::new(5000.0)];
        let oat_grid = vec![Celsius::new(0.0), Celsius::new(30.0)];
        let weights = vec![
            vec![Kilograms::new(3700.0), Kilograms::new(3500.0)],
            vec![Kilograms::new(3400.0), Kilograms::new(3200.0)],
        ];

        let grid = PerformanceGrid2D::new("Test Grid", pa_grid, oat_grid, weights);

        // Exact corners
        assert_eq!(
            grid.interpolate(Feet::new(0.0), Celsius::new(0.0)).get(),
            3700.0
        );
        assert_eq!(
            grid.interpolate(Feet::new(0.0), Celsius::new(30.0)).get(),
            3500.0
        );
        assert_eq!(
            grid.interpolate(Feet::new(5000.0), Celsius::new(0.0)).get(),
            3400.0
        );
        assert_eq!(
            grid.interpolate(Feet::new(5000.0), Celsius::new(30.0))
                .get(),
            3200.0
        );

        // Midpoint: (3700+3500)/2 = 3600 at 0 ft, (3400+3200)/2 = 3300 at 5000 ft
        // Midpoint at 2500 ft, 15 C: (3600+3300)/2 = 3450 kg
        let mid = grid.interpolate(Feet::new(2500.0), Celsius::new(15.0));
        assert!((mid.get() - 3450.0).abs() < 1e-6);

        // Clamping out-of-bounds
        assert_eq!(
            grid.interpolate(Feet::new(-1000.0), Celsius::new(-10.0))
                .get(),
            3700.0
        );
        assert_eq!(
            grid.interpolate(Feet::new(10000.0), Celsius::new(50.0))
                .get(),
            3200.0
        );
    }

    #[test]
    fn test_cruise_grid_3d() {
        let pa_grid = vec![Feet::new(0.0), Feet::new(10000.0)];
        let oat_grid = vec![Celsius::new(0.0), Celsius::new(20.0)];
        let gw_grid = vec![Kilograms::new(2000.0), Kilograms::new(3000.0)];

        let tas_table = vec![
            vec![
                vec![Knots::new(130.0), Knots::new(120.0)],
                vec![Knots::new(132.0), Knots::new(122.0)],
            ],
            vec![
                vec![Knots::new(140.0), Knots::new(130.0)],
                vec![Knots::new(142.0), Knots::new(132.0)],
            ],
        ];

        let ff_table = vec![
            vec![
                vec![KgPerHour::new(180.0), KgPerHour::new(220.0)],
                vec![KgPerHour::new(182.0), KgPerHour::new(222.0)],
            ],
            vec![
                vec![KgPerHour::new(160.0), KgPerHour::new(200.0)],
                vec![KgPerHour::new(162.0), KgPerHour::new(202.0)],
            ],
        ];

        let grid = CruisePerformanceGrid3D::new(pa_grid, oat_grid, gw_grid, tas_table, ff_table);

        let tas = grid.interpolate_tas(
            Feet::new(5000.0),
            Celsius::new(10.0),
            Kilograms::new(2500.0),
        );
        let ff = grid.interpolate_fuel_flow(
            Feet::new(5000.0),
            Celsius::new(10.0),
            Kilograms::new(2500.0),
        );

        // Center of the cube:
        // Average of all 8 tas corners = (130+120+132+122+140+130+142+132)/8 = 1048/8 = 131.0
        assert!((tas.get() - 131.0).abs() < 1e-6);
        // Average of all 8 ff corners = (180+220+182+222+160+200+162+202)/8 = 1528/8 = 191.0
        assert!((ff.get() - 191.0).abs() < 1e-6);
    }

    #[test]
    fn test_nan_safety() {
        let xs = [0.0, 10.0, 20.0];
        let ys = [100.0, 200.0, 300.0];
        assert_eq!(interp_1d(f64::NAN, &xs, &ys), 100.0);

        let pa_grid = vec![Feet::new(0.0), Feet::new(5000.0)];
        let oat_grid = vec![Celsius::new(0.0), Celsius::new(30.0)];
        let weights = vec![
            vec![Kilograms::new(3700.0), Kilograms::new(3500.0)],
            vec![Kilograms::new(3400.0), Kilograms::new(3200.0)],
        ];
        let grid = PerformanceGrid2D::new("Test Grid", pa_grid, oat_grid, weights);
        assert_eq!(
            grid.interpolate(Feet::new(f64::NAN), Celsius::new(0.0))
                .get(),
            3700.0
        );
        assert_eq!(
            grid.interpolate(Feet::new(0.0), Celsius::new(f64::NAN))
                .get(),
            3700.0
        );
    }
}
