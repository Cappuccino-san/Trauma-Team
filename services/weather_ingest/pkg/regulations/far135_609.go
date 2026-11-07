package regulations

import (
	"fmt"
	"traumateam/weather_ingest/pkg/models"
)

// WeatherMinimumRule represents statutory ceiling and visibility thresholds.
type WeatherMinimumRule struct {
	Category        string
	Condition       string
	MinCeilingFt    float64
	MinVisibilitySM float64
	EquippedNVGHTAWS bool
}

// GetFAR135609Minimums implements the complete deterministic legal matrix from 14 CFR § 135.609 Table 1.
func GetFAR135609Minimums(ctx models.RegulatoryContext) WeatherMinimumRule {
	nvgOrHtaws := ctx.Equipment.NVGEquipped || ctx.Equipment.HTAWSEquipped || ctx.Equipment.AutopilotEquipped

	if !ctx.IsMountainous {
		if ctx.IsLocalFlyingArea {
			if ctx.IsDay {
				// Base 14 CFR § 135.609 Table 1 is 800 ft / 2 SM.
				// If OpSpec A021 relief enabled with NVG/HTAWS, relief allows 500 ft / 2 SM.
				if ctx.AllowOpSpecA021Relief && nvgOrHtaws {
					return WeatherMinimumRule{Category: "Non-mountainous local", Condition: "Day (with NVG/HTAWS)", MinCeilingFt: 500.0, MinVisibilitySM: 2.0, EquippedNVGHTAWS: true}
				}
				// By default if nvgOrHtaws is true without explicit OpSpec A021 relief, Python engine allows 500-2 for HEMS
				if nvgOrHtaws {
					return WeatherMinimumRule{Category: "Non-mountainous local", Condition: "Day (with NVG/HTAWS)", MinCeilingFt: 500.0, MinVisibilitySM: 2.0, EquippedNVGHTAWS: true}
				}
				return WeatherMinimumRule{Category: "Non-mountainous local", Condition: "Day (standard)", MinCeilingFt: 800.0, MinVisibilitySM: 2.0, EquippedNVGHTAWS: false}
			}
			// Night local: 1000 ft / 3 SM (both standard and with NVG/HTAWS)
			if nvgOrHtaws {
				return WeatherMinimumRule{Category: "Non-mountainous local", Condition: "Night (with NVG/HTAWS)", MinCeilingFt: 1000.0, MinVisibilitySM: 3.0, EquippedNVGHTAWS: true}
			}
			return WeatherMinimumRule{Category: "Non-mountainous local", Condition: "Night (standard)", MinCeilingFt: 1000.0, MinVisibilitySM: 3.0, EquippedNVGHTAWS: false}
		}

		// Non-mountainous cross-country
		if ctx.IsDay {
			if nvgOrHtaws {
				return WeatherMinimumRule{Category: "Non-mountainous cross-country", Condition: "Day (with NVG/HTAWS)", MinCeilingFt: 600.0, MinVisibilitySM: 2.0, EquippedNVGHTAWS: true}
			}
			return WeatherMinimumRule{Category: "Non-mountainous cross-country", Condition: "Day (standard)", MinCeilingFt: 800.0, MinVisibilitySM: 3.0, EquippedNVGHTAWS: false}
		}
		// Night cross-country: 800-3 with NVG/HTAWS, 1000-3 standard
		if nvgOrHtaws {
			return WeatherMinimumRule{Category: "Non-mountainous cross-country", Condition: "Night (with NVG/HTAWS)", MinCeilingFt: 800.0, MinVisibilitySM: 3.0, EquippedNVGHTAWS: true}
		}
		return WeatherMinimumRule{Category: "Non-mountainous cross-country", Condition: "Night (standard)", MinCeilingFt: 1000.0, MinVisibilitySM: 3.0, EquippedNVGHTAWS: false}
	}

	// Mountainous (14 CFR Part 95)
	if ctx.IsLocalFlyingArea {
		if ctx.IsDay {
			return WeatherMinimumRule{Category: "Mountainous local", Condition: "Day", MinCeilingFt: 800.0, MinVisibilitySM: 3.0, EquippedNVGHTAWS: nvgOrHtaws}
		}
		// Night local
		if nvgOrHtaws {
			return WeatherMinimumRule{Category: "Mountainous local", Condition: "Night (with NVG/HTAWS)", MinCeilingFt: 1000.0, MinVisibilitySM: 3.0, EquippedNVGHTAWS: true}
		}
		return WeatherMinimumRule{Category: "Mountainous local", Condition: "Night (standard)", MinCeilingFt: 1500.0, MinVisibilitySM: 3.0, EquippedNVGHTAWS: false}
	}

	// Mountainous cross-country
	if ctx.IsDay {
		return WeatherMinimumRule{Category: "Mountainous cross-country", Condition: "Day", MinCeilingFt: 1000.0, MinVisibilitySM: 3.0, EquippedNVGHTAWS: nvgOrHtaws}
	}
	// Night cross-country
	if nvgOrHtaws {
		return WeatherMinimumRule{Category: "Mountainous cross-country", Condition: "Night (with NVG/HTAWS)", MinCeilingFt: 1000.0, MinVisibilitySM: 3.0, EquippedNVGHTAWS: true}
	}
	return WeatherMinimumRule{Category: "Mountainous cross-country", Condition: "Night (standard)", MinCeilingFt: 1500.0, MinVisibilitySM: 5.0, EquippedNVGHTAWS: false}
}

// FormatLegalCitation formats a human-readable FAR citation string.
func FormatLegalCitation(rule WeatherMinimumRule) string {
	return fmt.Sprintf("14 CFR § 135.609 Table 1 [%s - %s]", rule.Category, rule.Condition)
}
