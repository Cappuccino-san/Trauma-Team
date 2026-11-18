package regulations

import (
	"fmt"
	"traumateam/weather_ingest/pkg/models"
)

// FlightMinimumsReport is a backward-compatible report structure.
type FlightMinimumsReport struct {
	StationID               string  `json:"station_id"`
	IsCompliant             bool    `json:"is_compliant"`
	Verdict                 string  `json:"verdict"` // "GO" or "NO-GO"
	LimitingFactor          string  `json:"limiting_factor"`
	MinCeilingRequiredFt    float64 `json:"min_ceiling_required_ft"`
	ActualCeilingFt         float64 `json:"actual_ceiling_ft"`
	CeilingMarginFt         float64 `json:"ceiling_margin_ft"`
	MinVisibilityRequiredSM float64 `json:"min_visibility_required_sm"`
	ActualVisibilitySM      float64 `json:"actual_visibility_sm"`
	VisibilityMarginSM      float64 `json:"visibility_margin_sm"`
	LegalRuleApplied        string  `json:"legal_rule_applied"`
	Explanation             string  `json:"explanation"`
}

// GetFAR135615Minimums looks up statutory minimums (historical alias for § 135.609 Table 1).
func GetFAR135615Minimums(isDay, isMountainous, isLocal, nvgOrHtaws bool) WeatherMinimumRule {
	ctx := models.RegulatoryContext{
		IsDay:             isDay,
		IsMountainous:     isMountainous,
		IsLocalFlyingArea: isLocal,
		Equipment: models.EquipmentFlags{
			NVGEquipped:   nvgOrHtaws,
			HTAWSEquipped: nvgOrHtaws,
		},
		AllowOpSpecA021Relief: true,
	}
	return GetFAR135609Minimums(ctx)
}

// EvaluateMinimums evaluates a single observation against Table 1 minimums.
func EvaluateMinimums(obs *models.METAR, isDay, isMountainous, isLocal, nvgOrHtaws bool) FlightMinimumsReport {
	rule := GetFAR135615Minimums(isDay, isMountainous, isLocal, nvgOrHtaws)

	actualCeil := obs.CeilingAGLFt
	if actualCeil == 0 && !obs.HasCeiling {
		actualCeil = 25000.0
	}

	ceilMargin := actualCeil - rule.MinCeilingFt
	visMargin := obs.VisibilitySM - rule.MinVisibilitySM

	ceilOK := ceilMargin >= 0.0
	visOK := visMargin >= 0.0
	isCompliant := ceilOK && visOK

	limitingFactor := "NONE"
	verdict := "GO"
	var explanation string

	legalCite := fmt.Sprintf("14 CFR § 135.609 Table 1 [%s - %s]", rule.Category, rule.Condition)

	if !ceilOK {
		limitingFactor = string(models.LimitFAR135609CeilingBelowMinimums)
		verdict = "NO-GO"
		explanation = fmt.Sprintf("Observed ceiling (%.0f ft AGL) is below required minimum (%.0f ft AGL) by %.0f ft under %s",
			actualCeil, rule.MinCeilingFt, -ceilMargin, legalCite)
	} else if !visOK {
		limitingFactor = string(models.LimitFAR135609VisBelowMinimums)
		verdict = "NO-GO"
		explanation = fmt.Sprintf("Observed visibility (%.1f SM) is below required minimum (%.1f SM) by %.1f SM under %s",
			obs.VisibilitySM, rule.MinVisibilitySM, -visMargin, legalCite)
	} else {
		explanation = fmt.Sprintf("Compliant with %s: Ceiling margin +%.0f ft, Vis margin +%.1f SM",
			legalCite, ceilMargin, visMargin)
	}

	return FlightMinimumsReport{
		StationID:               obs.StationID,
		IsCompliant:             isCompliant,
		Verdict:                 verdict,
		LimitingFactor:          limitingFactor,
		MinCeilingRequiredFt:    rule.MinCeilingFt,
		ActualCeilingFt:         actualCeil,
		CeilingMarginFt:         ceilMargin,
		MinVisibilityRequiredSM: rule.MinVisibilitySM,
		ActualVisibilitySM:      obs.VisibilitySM,
		VisibilityMarginSM:      visMargin,
		LegalRuleApplied:        legalCite,
		Explanation:             explanation,
	}
}
