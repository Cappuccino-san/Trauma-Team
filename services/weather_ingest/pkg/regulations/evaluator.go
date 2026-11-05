package regulations

import (
	"fmt"
	"math"
	"time"
	"traumateam/weather_ingest/pkg/models"
)

var stationCoords = map[string]models.GeoCoordinate{
	"KAPA": {Latitude: 39.57, Longitude: -104.85},
	"KDEN": {Latitude: 39.86, Longitude: -104.67},
	"KLXV": {Latitude: 39.22, Longitude: -106.31},
	"KASE": {Latitude: 39.22, Longitude: -106.87},
	"KBJC": {Latitude: 39.91, Longitude: -105.12},
	"KCOS": {Latitude: 38.81, Longitude: -104.70},
	"KPUB": {Latitude: 38.29, Longitude: -104.50},
	"KCYS": {Latitude: 41.15, Longitude: -104.81},
	"CYS":  {Latitude: 41.15, Longitude: -104.81},
	"AKO":  {Latitude: 40.17, Longitude: -103.22},
	"SNY":  {Latitude: 41.14, Longitude: -102.98},
}

// PointInPolygon tests if a coordinate is inside a polygon using ray casting.
func pointInPolygon(pt models.GeoCoordinate, poly []models.GeoCoordinate) bool {
	if len(poly) < 3 {
		return false
	}
	inside := false
	j := len(poly) - 1
	for i := 0; i < len(poly); i++ {
		xi, yi := poly[i].Longitude, poly[i].Latitude
		xj, yj := poly[j].Longitude, poly[j].Latitude
		intersect := ((yi > pt.Latitude) != (yj > pt.Latitude)) &&
			(pt.Longitude < (xj-xi)*(pt.Latitude-yi)/(yj-yi)+xi)
		if intersect {
			inside = !inside
		}
		j = i
	}
	return inside
}

func pointNearOrInPolygon(pt models.GeoCoordinate, poly []models.GeoCoordinate) bool {
	if pointInPolygon(pt, poly) {
		return true
	}
	// Aviation corridor safety buffer: ~10 NM (0.15 deg) proximity to storm vertices
	for _, v := range poly {
		dLat := pt.Latitude - v.Latitude
		dLon := pt.Longitude - v.Longitude
		if (dLat*dLat + dLon*dLon) < 0.03 {
			return true
		}
	}
	return false
}

func corridorIntersects(waypoints []models.GeoCoordinate, poly []models.GeoCoordinate) bool {
	if len(poly) < 3 {
		return true
	}
	if len(waypoints) == 0 {
		return true
	}
	for _, wp := range waypoints {
		if pointNearOrInPolygon(wp, poly) {
			return true
		}
	}
	for k := 0; k < len(waypoints)-1; k++ {
		w1 := waypoints[k]
		w2 := waypoints[k+1]
		for s := 1; s < 10; s++ {
			frac := float64(s) / 10.0
			sample := models.GeoCoordinate{
				Latitude:  w1.Latitude + frac*(w2.Latitude-w1.Latitude),
				Longitude: w1.Longitude + frac*(w2.Longitude-w1.Longitude),
			}
			if pointNearOrInPolygon(sample, poly) {
				return true
			}
		}
	}
	return false
}

// EvaluateCorridorCompliance executes the multi-stage deterministic compliance evaluation.
func EvaluateCorridorCompliance(
	stations []models.METAR,
	hazards []models.WeatherHazard,
	ctx models.RegulatoryContext,
) models.ComplianceReport {
	audit := make([]string, 0)
	now := time.Now().UTC()

	// Stage 1: Evaluate Station Weather
	rule := GetFAR135609Minimums(ctx)
	legalCite := FormatLegalCitation(rule)

	worstCeil := math.Inf(1)
	worstVis := math.Inf(1)
	maxGust := 0.0
	stationReports := make([]models.StationComplianceReport, 0, len(stations))
	corridorPts := make([]models.GeoCoordinate, 0)

	for _, st := range stations {
		if pt, ok := stationCoords[st.StationID]; ok {
			corridorPts = append(corridorPts, pt)
		}

		stCeil := st.CeilingAGLFt
		if !st.HasCeiling {
			stCeil = 25000.0
		}
		if stCeil < worstCeil {
			worstCeil = stCeil
		}
		if st.VisibilitySM < worstVis {
			worstVis = st.VisibilitySM
		}
		if st.WindGustKts > maxGust {
			maxGust = st.WindGustKts
		}

		ceilMargin := stCeil - rule.MinCeilingFt
		visMargin := st.VisibilitySM - rule.MinVisibilitySM
		isStationOK := ceilMargin >= 0 && visMargin >= 0

		var explanation string
		if !isStationOK {
			if ceilMargin < 0 {
				explanation = fmt.Sprintf("Station %s ceiling %.0f ft below required %.0f ft", st.StationID, stCeil, rule.MinCeilingFt)
			} else {
				explanation = fmt.Sprintf("Station %s visibility %.1f SM below required %.1f SM", st.StationID, st.VisibilitySM, rule.MinVisibilitySM)
			}
		} else {
			explanation = fmt.Sprintf("Station %s compliant (ceil margin +%.0f ft, vis margin +%.1f SM)", st.StationID, ceilMargin, visMargin)
		}

		stationReports = append(stationReports, models.StationComplianceReport{
			StationID:         st.StationID,
			IsCompliant:       isStationOK,
			ActualCeilingFt:   stCeil,
			RequiredCeilingFt: rule.MinCeilingFt,
			CeilingMarginFt:   ceilMargin,
			ActualVisSM:       st.VisibilitySM,
			RequiredVisSM:     rule.MinVisibilitySM,
			VisMarginSM:       visMargin,
			Category:          st.FlightCategory,
			Explanation:       explanation,
		})

		audit = append(audit, fmt.Sprintf("Station %s: Ceil=%.0f ft, Vis=%.1f SM, Wind=%.0f@%.0fG%.0f kt -> Compliant=%v",
			st.StationID, stCeil, st.VisibilitySM, st.WindDirDeg, st.WindSpeedKts, st.WindGustKts, isStationOK))
	}

	if math.IsInf(worstCeil, 1) {
		worstCeil = 25000.0
	}
	if math.IsInf(worstVis, 1) {
		worstVis = 10.0
	}

	corridorCeilMargin := worstCeil - rule.MinCeilingFt
	corridorVisMargin := worstVis - rule.MinVisibilitySM

	// Stage 2: Active In-Flight Severe Weather Hazards (SIGMETs)
	activeHazards := make([]models.WeatherHazard, 0)
	for _, h := range hazards {
		if !h.Active {
			continue
		}
		if !h.ValidUntil.IsZero() && h.ValidUntil.Before(now) {
			continue
		}
		activeHazards = append(activeHazards, h)

		// Altitude band filtering: check if flight profile overlaps the hazard layer
		if h.MinAltFt > 0 && ctx.PlannedCruiseAltMSL > 0 && ctx.PlannedCruiseAltMSL < h.MinAltFt {
			continue // Flight is below hazard layer (e.g. low-level HEMS below FL180 turbulence)
		}
		if h.MaxAltFt > 0 && ctx.PlannedCruiseAltMSL > 0 && ctx.PlannedCruiseAltMSL > h.MaxAltFt {
			continue // Flight is above hazard layer
		}
		if h.MinAltFt >= 18000.0 && ctx.PlannedCruiseAltMSL < 18000.0 {
			continue // High-altitude FL180+ Class A airspace turbulence does not affect low-altitude VFR flights
		}

		intersects := corridorIntersects(corridorPts, h.Polygon)
		if !intersects {
			continue
		}

		if h.HazardType == models.HazardTypeConvectiveSIGMET {
			return models.ComplianceReport{
				IsCompliant:          false,
				Verdict:              models.VerdictDisqualified,
				LimitingFactor:       models.LimitConvectiveSIGMET,
				WorstCeilingFt:       worstCeil,
				WorstVisibilitySM:    worstVis,
				MinRequiredCeilingFt: rule.MinCeilingFt,
				MinRequiredVisSM:     rule.MinVisibilitySM,
				CeilingMarginFt:      corridorCeilMargin,
				VisMarginSM:          corridorVisMargin,
				LegalRuleApplied:     "FAA Safety Directive: Active Convective Storm Hazard",
				Explanation:          fmt.Sprintf("DISQUALIFIED: Flight corridor intersects active Convective SIGMET %s: %s", h.AdvisoryID, h.Summary),
				StationReports:       stationReports,
				ActiveHazards:        activeHazards,
				EvaluationTimeUTC:    now.Format(time.RFC3339),
				AuditTrace:           append(audit, "Active Convective SIGMET intersects corridor -> automatic disqualification"),
			}
		}
		if h.HazardType == models.HazardTypeSevereIcing {
			return models.ComplianceReport{
				IsCompliant:          false,
				Verdict:              models.VerdictDisqualified,
				LimitingFactor:       models.LimitSevereIcing,
				WorstCeilingFt:       worstCeil,
				WorstVisibilitySM:    worstVis,
				MinRequiredCeilingFt: rule.MinCeilingFt,
				MinRequiredVisSM:     rule.MinVisibilitySM,
				CeilingMarginFt:      corridorCeilMargin,
				VisMarginSM:          corridorVisMargin,
				LegalRuleApplied:     "FAA AC 91-74B: Flight Into Known Severe Icing Prohibited",
				Explanation:          fmt.Sprintf("DISQUALIFIED: Active Severe Icing advisory %s: %s", h.AdvisoryID, h.Summary),
				StationReports:       stationReports,
				ActiveHazards:        activeHazards,
				EvaluationTimeUTC:    now.Format(time.RFC3339),
				AuditTrace:           append(audit, "Active Severe Icing hazard intersects corridor -> automatic disqualification"),
			}
		}
		if h.HazardType == models.HazardTypeSevereTurbulence {
			return models.ComplianceReport{
				IsCompliant:          false,
				Verdict:              models.VerdictDisqualified,
				LimitingFactor:       models.LimitSevereTurbulence,
				WorstCeilingFt:       worstCeil,
				WorstVisibilitySM:    worstVis,
				MinRequiredCeilingFt: rule.MinCeilingFt,
				MinRequiredVisSM:     rule.MinVisibilitySM,
				CeilingMarginFt:      corridorCeilMargin,
				VisMarginSM:          corridorVisMargin,
				LegalRuleApplied:     "HEMS Patient Safety Directive: Severe Turbulence Hazard",
				Explanation:          fmt.Sprintf("DISQUALIFIED: Active Severe Turbulence advisory %s: %s", h.AdvisoryID, h.Summary),
				StationReports:       stationReports,
				ActiveHazards:        activeHazards,
				EvaluationTimeUTC:    now.Format(time.RFC3339),
				AuditTrace:           append(audit, "Active Severe Turbulence advisory intersects corridor -> automatic disqualification"),
			}
		}
	}

	// Stage 3: 14 CFR § 135.615 Obstacle Clearance Check
	if ctx.PlannedCruiseAltMSL > 0 && ctx.HighestObstacleMSL > 0 {
		requiredClearance := 300.0 // Day
		if !ctx.IsDay {
			requiredClearance = 500.0 // Night
		}
		actualClearance := ctx.PlannedCruiseAltMSL - ctx.HighestObstacleMSL
		audit = append(audit, fmt.Sprintf("§ 135.615 Obstacle Check: Cruise Alt=%.0f ft MSL, Obstacle=%.0f ft MSL, Actual Clearance=%.0f ft (req %.0f ft)",
			ctx.PlannedCruiseAltMSL, ctx.HighestObstacleMSL, actualClearance, requiredClearance))

		if actualClearance < requiredClearance {
			return models.ComplianceReport{
				IsCompliant:          false,
				Verdict:              models.VerdictBelowMinimums,
				LimitingFactor:       models.LimitFAR135615ObstacleClearance,
				WorstCeilingFt:       worstCeil,
				WorstVisibilitySM:    worstVis,
				MinRequiredCeilingFt: rule.MinCeilingFt,
				MinRequiredVisSM:     rule.MinVisibilitySM,
				CeilingMarginFt:      corridorCeilMargin,
				VisMarginSM:          corridorVisMargin,
				LegalRuleApplied:     "14 CFR § 135.615 VFR Obstacle Clearance Violation",
				Explanation: fmt.Sprintf("VIOLATION: Planned cruise altitude (%.0f ft MSL) provides %.0f ft clearance over highest obstacle (%.0f ft MSL), failing § 135.615 mandatory clearance of %.0f ft",
					ctx.PlannedCruiseAltMSL, actualClearance, ctx.HighestObstacleMSL, requiredClearance),
				StationReports:    stationReports,
				ActiveHazards:     activeHazards,
				EvaluationTimeUTC: now.Format(time.RFC3339),
				AuditTrace:        audit,
			}
		}
	}

	// Stage 4: 14 CFR § 135.609 Ceiling & Visibility Check & § 135.611 IFR Transition
	vfrCeilOK := corridorCeilMargin >= 0.0
	vfrVisOK := corridorVisMargin >= 0.0

	if !vfrCeilOK || !vfrVisOK {
		limiting := models.LimitFAR135609CeilingBelowMinimums
		if vfrCeilOK && !vfrVisOK {
			limiting = models.LimitFAR135609VisBelowMinimums
		}

		// Check if eligible for IFR transition under 14 CFR § 135.611
		if ctx.IsIFRCertified && ctx.HasIFRAlternate && ctx.HasSevereWeatherRadar {
			return models.ComplianceReport{
				IsCompliant:          true, // Compliant for IFR
				Verdict:              models.VerdictIFRRequired,
				LimitingFactor:       limiting,
				WorstCeilingFt:       worstCeil,
				WorstVisibilitySM:    worstVis,
				MinRequiredCeilingFt: rule.MinCeilingFt,
				MinRequiredVisSM:     rule.MinVisibilitySM,
				CeilingMarginFt:      corridorCeilMargin,
				VisMarginSM:          corridorVisMargin,
				LegalRuleApplied:     "14 CFR § 135.611 IFR Operations Authorized",
				Explanation: fmt.Sprintf("VFR weather below § 135.609 minimums (Ceil margin: %.0f ft, Vis margin: %.1f SM), but compliant for IFR dispatch under § 135.611.",
					corridorCeilMargin, corridorVisMargin),
				StationReports:    stationReports,
				ActiveHazards:     activeHazards,
				EvaluationTimeUTC: now.Format(time.RFC3339),
				AuditTrace:        audit,
			}
		}

		// Standard VFR below minimums
		return models.ComplianceReport{
			IsCompliant:          false,
			Verdict:              models.VerdictBelowMinimums,
			LimitingFactor:       limiting,
			WorstCeilingFt:       worstCeil,
			WorstVisibilitySM:    worstVis,
			MinRequiredCeilingFt: rule.MinCeilingFt,
			MinRequiredVisSM:     rule.MinVisibilitySM,
			CeilingMarginFt:      corridorCeilMargin,
			VisMarginSM:          corridorVisMargin,
			LegalRuleApplied:     legalCite,
			Explanation: fmt.Sprintf("WEATHER VIOLATION under %s: Worst ceiling %.0f ft (req %.0f ft, margin %.0f ft) | Worst vis %.1f SM (req %.1f SM, margin %.1f SM)",
				legalCite, worstCeil, rule.MinCeilingFt, corridorCeilMargin, worstVis, rule.MinVisibilitySM, corridorVisMargin),
			StationReports:    stationReports,
			ActiveHazards:     activeHazards,
			EvaluationTimeUTC: now.Format(time.RFC3339),
			AuditTrace:        audit,
		}
	}

	// Stage 5: Operational Caution & Marginal Assessment
	isMarginal := corridorCeilMargin <= 200.0 || corridorVisMargin <= 1.0 || maxGust >= 25.0
	if isMarginal {
		return models.ComplianceReport{
			IsCompliant:          true,
			Verdict:              models.VerdictMarginal,
			LimitingFactor:       models.LimitNone,
			WorstCeilingFt:       worstCeil,
			WorstVisibilitySM:    worstVis,
			MinRequiredCeilingFt: rule.MinCeilingFt,
			MinRequiredVisSM:     rule.MinVisibilitySM,
			CeilingMarginFt:      corridorCeilMargin,
			VisMarginSM:          corridorVisMargin,
			LegalRuleApplied:     legalCite,
			Explanation: fmt.Sprintf("MARGINAL WEATHER under %s: Ceil margin +%.0f ft, Vis margin +%.1f SM (Caution: scud-running risk or gust %.0f kt)",
				legalCite, corridorCeilMargin, corridorVisMargin, maxGust),
			StationReports:    stationReports,
			ActiveHazards:     activeHazards,
			EvaluationTimeUTC: now.Format(time.RFC3339),
			AuditTrace:        audit,
		}
	}

	// Stage 6: Approved (Full GO)
	return models.ComplianceReport{
		IsCompliant:          true,
		Verdict:              models.VerdictApproved,
		LimitingFactor:       models.LimitNone,
		WorstCeilingFt:       worstCeil,
		WorstVisibilitySM:    worstVis,
		MinRequiredCeilingFt: rule.MinCeilingFt,
		MinRequiredVisSM:     rule.MinVisibilitySM,
		CeilingMarginFt:      corridorCeilMargin,
		VisMarginSM:          corridorVisMargin,
		LegalRuleApplied:     legalCite,
		Explanation: fmt.Sprintf("APPROVED under %s: Ceiling margin +%.0f ft, Vis margin +%.1f SM",
			legalCite, corridorCeilMargin, corridorVisMargin),
		StationReports:    stationReports,
		ActiveHazards:     activeHazards,
		EvaluationTimeUTC: now.Format(time.RFC3339),
		AuditTrace:        audit,
	}
}
