"""TraumaTeam Regulations Package."""

from traumateam.regulations.far135_615 import (
    get_far_135_615_minimums,
    evaluate_far_135_615_compliance,
    WeatherComplianceReport,
    WeatherMinimumRule,
)
from traumateam.regulations.fuel_reserves import (
    evaluate_fuel_reserves,
    FuelReserveReport,
)

__all__ = [
    "get_far_135_615_minimums",
    "evaluate_far_135_615_compliance",
    "WeatherComplianceReport",
    "WeatherMinimumRule",
    "evaluate_fuel_reserves",
    "FuelReserveReport",
]
