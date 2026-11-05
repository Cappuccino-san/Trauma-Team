"""Multi-Dimensional RFM Performance Calculation Engine.

Interpolates raw Rotorcraft Flight Manual performance grids using SciPy RegularGridInterpolator.
Calculates IGE/OGE hover ceilings, OEI Category A helipad capabilities, and phase-specific fuel flow.
"""

from __future__ import annotations
import numpy as np
from scipy.interpolate import RegularGridInterpolator
from traumateam.models.rfm import (
    RFMAirframeSpec,
    RFMPerformanceGrid,
    RFMCruiseGrid,
)

class RFMPerformanceEngine:
    def __init__(self, airframe: RFMAirframeSpec):
        self.airframe = airframe
        self._init_interpolators()

    def _init_interpolators(self) -> None:
        # 1. IGE Hover Max Weight (PA, OAT)
        ige_grid = self.airframe.ige_hover_grid
        ige_pa = np.array(ige_grid.pa_grid_ft, dtype=float)
        ige_oat = np.array(ige_grid.oat_grid_c, dtype=float)
        ige_weights = np.array(ige_grid.weight_table_kg, dtype=float)
        self._interp_ige = RegularGridInterpolator(
            (ige_pa, ige_oat),
            ige_weights,
            bounds_error=False,
            fill_value=None  # Extrapolates cleanly at boundary edge
        )

        # 2. OGE Hover Max Weight (PA, OAT)
        oge_grid = self.airframe.oge_hover_grid
        oge_pa = np.array(oge_grid.pa_grid_ft, dtype=float)
        oge_oat = np.array(oge_grid.oat_grid_c, dtype=float)
        oge_weights = np.array(oge_grid.weight_table_kg, dtype=float)
        self._interp_oge = RegularGridInterpolator(
            (oge_pa, oge_oat),
            oge_weights,
            bounds_error=False,
            fill_value=None
        )

        # 3. OEI Cat A Helipad Max Weight (PA, OAT)
        oei_grid = self.airframe.oei_cat_a_grid
        oei_pa = np.array(oei_grid.pa_grid_ft, dtype=float)
        oei_oat = np.array(oei_grid.oat_grid_c, dtype=float)
        oei_weights = np.array(oei_grid.weight_table_kg, dtype=float)
        self._interp_oei = RegularGridInterpolator(
            (oei_pa, oei_oat),
            oei_weights,
            bounds_error=False,
            fill_value=None
        )

        # 4. Cruise Performance 3D Grids (PA, OAT, GW)
        cruise = self.airframe.cruise_performance
        c_pa = np.array(cruise.pa_grid_ft, dtype=float)
        c_oat = np.array(cruise.oat_grid_c, dtype=float)
        c_gw = np.array(cruise.gw_grid_kg, dtype=float)

        tas_data = np.array(cruise.tas_table_kts, dtype=float)
        ff_data = np.array(cruise.fuel_flow_kg_hr, dtype=float)

        self._interp_tas = RegularGridInterpolator(
            (c_pa, c_oat, c_gw),
            tas_data,
            bounds_error=False,
            fill_value=None
        )
        self._interp_fuel_flow = RegularGridInterpolator(
            (c_pa, c_oat, c_gw),
            ff_data,
            bounds_error=False,
            fill_value=None
        )

    def get_max_ige_hover_weight(self, pa_ft: float, oat_c: float) -> float:
        """Computes In-Ground-Effect (IGE) hover maximum allowable gross weight in kg."""
        val = float(self._interp_ige([[pa_ft, oat_c]])[0])
        return min(self.airframe.mtow_kg, max(0.0, val))

    def get_max_oge_hover_weight(self, pa_ft: float, oat_c: float) -> float:
        """Computes Out-of-Ground-Effect (OGE) hover maximum allowable gross weight in kg."""
        val = float(self._interp_oge([[pa_ft, oat_c]])[0])
        return min(self.airframe.mtow_kg, max(0.0, val))

    def get_max_oei_cat_a_weight(self, pa_ft: float, oat_c: float) -> float:
        """Computes One-Engine-Inoperative (OEI) Category A helipad takeoff/landing max weight in kg."""
        val = float(self._interp_oei([[pa_ft, oat_c]])[0])
        return min(self.airframe.mtow_kg, max(0.0, val))

    def get_cruise_tas_knots(self, pa_ft: float, oat_c: float, gw_kg: float) -> float:
        """Computes True Airspeed (TAS) in knots at specified cruise condition."""
        clamped_gw = max(self.airframe.min_flight_weight_kg, min(gw_kg, self.airframe.mtow_kg))
        return float(self._interp_tas([[pa_ft, oat_c, clamped_gw]])[0])

    def get_cruise_fuel_flow_kg_per_hr(self, pa_ft: float, oat_c: float, gw_kg: float) -> float:
        """Computes cruise fuel flow rate in kg/hr."""
        clamped_gw = max(self.airframe.min_flight_weight_kg, min(gw_kg, self.airframe.mtow_kg))
        return float(self._interp_fuel_flow([[pa_ft, oat_c, clamped_gw]])[0])

    def get_hover_oge_fuel_flow_kg_per_hr(self, pa_ft: float, oat_c: float, gw_kg: float) -> float:
        """Computes OGE hover fuel flow rate in kg/hr (typically higher than cruise due to induced power)."""
        cruise_ff = self.get_cruise_fuel_flow_kg_per_hr(pa_ft, oat_c, gw_kg)
        return cruise_ff * self.airframe.hover_fuel_flow_factor

    def get_loiter_fuel_flow_kg_per_hr(self, pa_ft: float, oat_c: float, gw_kg: float) -> float:
        """Computes holding/loiter fuel flow rate in kg/hr (minimum power required speed Vy)."""
        cruise_ff = self.get_cruise_fuel_flow_kg_per_hr(pa_ft, oat_c, gw_kg)
        return cruise_ff * self.airframe.loiter_fuel_flow_factor
