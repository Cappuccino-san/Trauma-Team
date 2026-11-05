package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"traumateam/weather_ingest/pkg/models"
	"traumateam/weather_ingest/pkg/noaa"
	"traumateam/weather_ingest/pkg/pb"
	"traumateam/weather_ingest/pkg/regulations"
)

// Broadcaster is an interface for broadcasting weather events (e.g. to WebSocket clients).
type Broadcaster interface {
	BroadcastWeatherUpdate(stationID string, metar *models.METAR)
	BroadcastMinimumsEvaluation(rep *models.ComplianceReport)
}

// WeatherServer implements pb.WeatherIngestServiceServer.
type WeatherServer struct {
	pb.UnimplementedWeatherIngestServiceServer
	client      *noaa.WeatherFeedClient
	broadcaster Broadcaster
}

// NewWeatherServer constructs a new WeatherServer.
func NewWeatherServer(client *noaa.WeatherFeedClient, broadcaster Broadcaster) *WeatherServer {
	return &WeatherServer{
		client:      client,
		broadcaster: broadcaster,
	}
}

// GetCurrentConditions retrieves METAR, TAF, Winds Aloft, and SIGMET hazards for a station.
func (s *WeatherServer) GetCurrentConditions(ctx context.Context, req *pb.StationConditionsRequest) (*pb.CurrentConditionsResponse, error) {
	if req.StationId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "station_id must not be empty")
	}

	metar, err := s.client.GetMETAR(ctx, req.StationId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get METAR for %s: %v", req.StationId, err)
	}

	taf, _ := s.client.GetTAF(ctx, req.StationId)
	winds, _ := s.client.GetWindsAloft(ctx, req.StationId)
	hazards, _ := s.client.GetActiveHazards(ctx)

	// Broadcast to frontend subscribers if broadcaster is configured
	if s.broadcaster != nil {
		s.broadcaster.BroadcastWeatherUpdate(req.StationId, metar)
	}

	resp := &pb.CurrentConditionsResponse{
		StationId: req.StationId,
		Metar:     ConvertMETARToPB(metar),
	}

	if taf != nil {
		resp.Taf = ConvertTAFToPB(taf)
	}
	if winds != nil {
		resp.WindsAloft = ConvertWindsAloftToPB(winds)
	}
	for _, h := range hazards {
		resp.Hazards = append(resp.Hazards, ConvertHazardToPB(h))
	}

	return resp, nil
}

// StreamWeatherFeeds streams real-time weather feed updates for the specified stations.
func (s *WeatherServer) StreamWeatherFeeds(req *pb.StreamWeatherRequest, stream pb.WeatherIngestService_StreamWeatherFeedsServer) error {
	if len(req.Stations) == 0 {
		return status.Errorf(codes.InvalidArgument, "must specify at least one station to stream")
	}

	interval := time.Duration(req.RefreshIntervalSeconds) * time.Second
	if interval < 1*time.Second {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial immediate send
	for _, st := range req.Stations {
		metar, err := s.client.GetMETAR(stream.Context(), st)
		if err == nil {
			update := &pb.WeatherFeedUpdate{
				UpdateType:   "METAR",
				TimestampUtc: time.Now().UTC().Format(time.RFC3339),
				StationId:    st,
				Metar:        ConvertMETARToPB(metar),
			}
			if err := stream.Send(update); err != nil {
				return err
			}
		}
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			for _, st := range req.Stations {
				metar, err := s.client.GetMETAR(stream.Context(), st)
				if err != nil {
					continue
				}

				update := &pb.WeatherFeedUpdate{
					UpdateType:   "METAR",
					TimestampUtc: time.Now().UTC().Format(time.RFC3339),
					StationId:    st,
					Metar:        ConvertMETARToPB(metar),
				}
				if err := stream.Send(update); err != nil {
					return fmt.Errorf("failed to send weather feed update: %w", err)
				}
			}
		}
	}
}

// EvaluateFlightMinimums evaluates corridor weather and obstacles against 14 CFR Part 135 Subpart L.
func (s *WeatherServer) EvaluateFlightMinimums(ctx context.Context, req *pb.FlightMinimumsEvaluationRequest) (*pb.FlightMinimumsEvaluationResponse, error) {
	if req.DepartureStation == "" || req.DestinationStation == "" {
		return nil, status.Errorf(codes.InvalidArgument, "departure_station and destination_station must not be empty")
	}

	stationList := []string{req.DepartureStation}
	stationList = append(stationList, req.IntermediateStations...)
	stationList = append(stationList, req.DestinationStation)

	metars := make([]models.METAR, 0, len(stationList))
	for _, stID := range stationList {
		m, err := s.client.GetMETAR(ctx, stID)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "failed to get METAR for station %s: %v", stID, err)
		}
		metars = append(metars, *m)
	}

	hazards, _ := s.client.GetActiveHazards(ctx)
	hazardModels := make([]models.WeatherHazard, 0, len(hazards))
	for _, h := range hazards {
		hazardModels = append(hazardModels, *h)
	}

	regCtx := models.RegulatoryContext{
		IsMountainous:         req.IsMountainous,
		IsLocalFlyingArea:     req.IsLocal,
		IsDay:                 req.IsDay,
		AllowOpSpecA021Relief: req.AllowOpspecA021Relief,
		IsIFRCertified:        req.IsIfrCertified,
		HasIFRAlternate:       req.HasIfrAlternate,
		HasSevereWeatherRadar: req.HasSevereWeatherRadar,
		PlannedCruiseAltMSL:   req.PlannedCruiseAltMsl,
		HighestObstacleMSL:    req.HighestObstacleAlongRouteMsl,
	}
	if req.Equipment != nil {
		regCtx.Equipment = models.EquipmentFlags{
			NVGEquipped:            req.Equipment.NvgEquipped,
			HTAWSEquipped:          req.Equipment.HtawsEquipped,
			RadarAltimeterEquipped: req.Equipment.RadarAltimeterEquipped,
			AutopilotEquipped:      req.Equipment.AutopilotEquipped,
		}
	}

	report := regulations.EvaluateCorridorCompliance(metars, hazardModels, regCtx)

	if s.broadcaster != nil {
		s.broadcaster.BroadcastMinimumsEvaluation(&report)
	}

	return ConvertComplianceReportToPB(&report), nil
}

// StreamFlightMinimums streams periodic re-evaluations of flight minimums for a corridor.
func (s *WeatherServer) StreamFlightMinimums(req *pb.StreamFlightMinimumsRequest, stream pb.WeatherIngestService_StreamFlightMinimumsServer) error {
	if req.Parameters == nil {
		return status.Errorf(codes.InvalidArgument, "parameters must not be nil")
	}

	interval := time.Duration(req.RefreshIntervalSeconds) * time.Second
	if interval < 1*time.Second {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial evaluation
	initialResp, err := s.EvaluateFlightMinimums(stream.Context(), req.Parameters)
	if err == nil {
		update := &pb.FlightMinimumsUpdate{
			TimestampUtc: time.Now().UTC().Format(time.RFC3339),
			Evaluation:   initialResp,
		}
		if err := stream.Send(update); err != nil {
			return err
		}
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			resp, err := s.EvaluateFlightMinimums(stream.Context(), req.Parameters)
			if err != nil {
				continue
			}

			update := &pb.FlightMinimumsUpdate{
				TimestampUtc: time.Now().UTC().Format(time.RFC3339),
				Evaluation:   resp,
			}
			if err := stream.Send(update); err != nil {
				return fmt.Errorf("failed to send flight minimums update: %w", err)
			}
		}
	}
}

// -----------------------------------------------------------------------------
// Type Converters: Models -> Protobuf
// -----------------------------------------------------------------------------

func ConvertMETARToPB(m *models.METAR) *pb.METARMessage {
	if m == nil {
		return nil
	}
	pbMsg := &pb.METARMessage{
		StationId:             m.StationID,
		ObservationTimeUtc:    m.ObservationTimeUTC.Format(time.RFC3339),
		WindDirDeg:            m.WindDirDeg,
		WindSpeedKts:          m.WindSpeedKts,
		WindGustKts:           m.WindGustKts,
		IsVariableWind:        m.IsVariableWind,
		VarMinDeg:             m.VarMinDeg,
		VarMaxDeg:             m.VarMaxDeg,
		VisibilitySm:          m.VisibilitySM,
		VisibilityLessThan:    m.VisibilityLessThan,
		VisibilityGreaterThan: m.VisibilityGreaterThan,
		CeilingAglFt:          m.CeilingAGLFt,
		HasCeiling:            m.HasCeiling,
		TemperatureC:          m.TemperatureC,
		DewpointC:             m.DewpointC,
		AltimeterInhg:         m.AltimeterInHg,
		WeatherPhenomena:      m.PresentWeather,
		DensityAltitudeFt:     m.DensityAltitudeFt,
		PressureAltitudeFt:    m.PressureAltitudeFt,
		RawText:               m.RawText,
	}

	switch m.FlightCategory {
	case models.FlightCategoryVFR:
		pbMsg.FlightCategory = pb.FlightRuleCategory_CATEGORY_VFR
	case models.FlightCategoryMVFR:
		pbMsg.FlightCategory = pb.FlightRuleCategory_CATEGORY_MVFR
	case models.FlightCategoryIFR:
		pbMsg.FlightCategory = pb.FlightRuleCategory_CATEGORY_IFR
	case models.FlightCategoryLIFR:
		pbMsg.FlightCategory = pb.FlightRuleCategory_CATEGORY_LIFR
	default:
		pbMsg.FlightCategory = pb.FlightRuleCategory_CATEGORY_UNSPECIFIED
	}

	for _, c := range m.CloudLayers {
		pbMsg.Clouds = append(pbMsg.Clouds, &pb.CloudLayer{
			Coverage:       string(c.Coverage),
			AltitudeAglFt: c.BaseFtAGL,
			IsCeiling:      c.IsCeiling,
			Modifier:       c.Modifier,
		})
	}

	return pbMsg
}

func ConvertTAFToPB(t *models.TAF) *pb.TAFMessage {
	if t == nil {
		return nil
	}
	pbMsg := &pb.TAFMessage{
		StationId:      t.StationID,
		IssueTimeUtc:   t.IssueTimeUTC.Format(time.RFC3339),
		ValidFromUtc:   t.ValidFromUTC.Format(time.RFC3339),
		ValidToUtc:     t.ValidToUTC.Format(time.RFC3339),
		RawText:        t.RawText,
	}

	for _, p := range t.Periods {
		pbPeriod := &pb.TAFFilterPeriod{
			ChangeType:       string(p.ChangeType),
			StartTimeUtc:     p.ValidFromUTC.Format(time.RFC3339),
			EndTimeUtc:       p.ValidToUTC.Format(time.RFC3339),
			WindDirDeg:       p.WindDirDeg,
			WindSpeedKts:     p.WindSpeedKts,
			WindGustKts:      p.WindGustKts,
			VisibilitySm:     p.VisibilitySM,
			CeilingAglFt:     p.CeilingAGLFt,
			WeatherPhenomena: p.PresentWeather,
		}
		for _, c := range p.CloudLayers {
			pbPeriod.Clouds = append(pbPeriod.Clouds, &pb.CloudLayer{
				Coverage:       string(c.Coverage),
				AltitudeAglFt: c.BaseFtAGL,
				IsCeiling:      c.IsCeiling,
				Modifier:       c.Modifier,
			})
		}
		pbMsg.ForecastPeriods = append(pbMsg.ForecastPeriods, pbPeriod)
	}

	return pbMsg
}

func ConvertWindsAloftToPB(w *models.WindsAloftForecast) *pb.WindsAloftMessage {
	if w == nil {
		return nil
	}
	pbMsg := &pb.WindsAloftMessage{
		StationId:          w.StationID,
		ObservationTimeUtc: w.ObservationTimeUTC.Format(time.RFC3339),
		RawText:            w.RawText,
	}
	for _, l := range w.Layers {
		pbMsg.Layers = append(pbMsg.Layers, &pb.WindsAloftLayer{
			AltitudeMslFt: l.AltitudeMSLFt,
			DirectionDeg:  l.DirectionDeg,
			SpeedKts:      l.SpeedKts,
			TemperatureC:  l.TemperatureC,
		})
	}
	return pbMsg
}

func ConvertHazardToPB(h *models.WeatherHazard) *pb.SIGMETMessage {
	if h == nil {
		return nil
	}
	pbMsg := &pb.SIGMETMessage{
		AdvisoryId:   h.AdvisoryID,
		Summary:      h.Summary,
		ValidFromUtc: h.ValidFrom.Format(time.RFC3339),
		ValidToUtc:   h.ValidUntil.Format(time.RFC3339),
		MinAltitudeFt: h.MinAltFt,
		MaxAltitudeFt: h.MaxAltFt,
		Active:       h.Active,
		RawText:      h.RawText,
	}
	if pbMsg.AdvisoryId == "" {
		pbMsg.AdvisoryId = h.HazardID
	}

	switch h.HazardType {
	case models.HazardTypeConvectiveSIGMET:
		pbMsg.HazardType = pb.HazardType_HAZARD_TYPE_CONVECTIVE_SIGMET
	case models.HazardTypeSevereTurbulence:
		pbMsg.HazardType = pb.HazardType_HAZARD_TYPE_SEVERE_TURBULENCE
	case models.HazardTypeSevereIcing:
		pbMsg.HazardType = pb.HazardType_HAZARD_TYPE_SEVERE_ICING
	case models.HazardTypeVolcanicAsh:
		pbMsg.HazardType = pb.HazardType_HAZARD_TYPE_VOLCANIC_ASH
	case models.HazardTypeDustSandstorm:
		pbMsg.HazardType = pb.HazardType_HAZARD_TYPE_DUST_SANDSTORM
	default:
		pbMsg.HazardType = pb.HazardType_HAZARD_TYPE_UNSPECIFIED
	}

	if h.Severity == "SEVERE" {
		pbMsg.Severity = pb.HazardSeverity_SEVERITY_SEVERE
	} else if h.Severity == "EXTREME" {
		pbMsg.Severity = pb.HazardSeverity_SEVERITY_EXTREME
	} else if h.Severity == "MODERATE" {
		pbMsg.Severity = pb.HazardSeverity_SEVERITY_MODERATE
	} else {
		pbMsg.Severity = pb.HazardSeverity_SEVERITY_LIGHT
	}

	for _, pt := range h.Polygon {
		pbMsg.PolygonCoordinates = append(pbMsg.PolygonCoordinates, &pb.GeoCoordinate{
			Latitude:  pt.Latitude,
			Longitude: pt.Longitude,
		})
	}

	return pbMsg
}

func ConvertComplianceReportToPB(r *models.ComplianceReport) *pb.FlightMinimumsEvaluationResponse {
	if r == nil {
		return nil
	}
	pbMsg := &pb.FlightMinimumsEvaluationResponse{
		IsCompliant:             r.IsCompliant,
		WorstCeilingFt:          r.WorstCeilingFt,
		WorstVisibilitySm:       r.WorstVisibilitySM,
		MinRequiredCeilingFt:    r.MinRequiredCeilingFt,
		MinRequiredVisibilitySm: r.MinRequiredVisSM,
		LegalRuleApplied:        r.LegalRuleApplied,
		Explanation:             r.Explanation,
		EvaluationTimeUtc:       r.EvaluationTimeUTC,
		AuditTrace:              r.AuditTrace,
	}

	switch r.Verdict {
	case models.VerdictApproved:
		pbMsg.Verdict = pb.DispatchVerdict_VERDICT_APPROVED
	case models.VerdictMarginal:
		pbMsg.Verdict = pb.DispatchVerdict_VERDICT_MARGINAL
	case models.VerdictBelowMinimums:
		pbMsg.Verdict = pb.DispatchVerdict_VERDICT_BELOW_MINIMUMS
	case models.VerdictDisqualified:
		pbMsg.Verdict = pb.DispatchVerdict_VERDICT_DISQUALIFIED
	case models.VerdictIFRRequired:
		pbMsg.Verdict = pb.DispatchVerdict_VERDICT_IFR_REQUIRED
	default:
		pbMsg.Verdict = pb.DispatchVerdict_VERDICT_UNSPECIFIED
	}

	switch r.LimitingFactor {
	case models.LimitFAR135609CeilingBelowMinimums, models.LimitFAR135615CeilingBelowMinimums:
		pbMsg.LimitingFactor = pb.LimitingFactor_FAR_135_609_CEILING_BELOW_MINIMUMS
	case models.LimitFAR135609VisBelowMinimums, models.LimitFAR135615VisBelowMinimums:
		pbMsg.LimitingFactor = pb.LimitingFactor_FAR_135_609_VISIBILITY_BELOW_MINIMUMS
	case models.LimitFAR135615ObstacleClearance:
		pbMsg.LimitingFactor = pb.LimitingFactor_FAR_135_615_OBSTACLE_CLEARANCE_VIOLATION
	case models.LimitFAR135209FuelReserveViolation:
		pbMsg.LimitingFactor = pb.LimitingFactor_FAR_135_209_VFR_RESERVE_VIOLATION
	case models.LimitConvectiveSIGMET:
		pbMsg.LimitingFactor = pb.LimitingFactor_ACTIVE_CONVECTIVE_SIGMET
	case models.LimitSevereIcing:
		pbMsg.LimitingFactor = pb.LimitingFactor_SEVERE_ICING_HAZARD
	case models.LimitSevereTurbulence:
		pbMsg.LimitingFactor = pb.LimitingFactor_SEVERE_TURBULENCE_HAZARD
	case models.LimitUnflyableWind:
		pbMsg.LimitingFactor = pb.LimitingFactor_UNFLYABLE_WIND
	case models.LimitIFRAlternateMissing:
		pbMsg.LimitingFactor = pb.LimitingFactor_IFR_ALTERNATE_MISSING
	case models.LimitNoWeatherWithin15NM:
		pbMsg.LimitingFactor = pb.LimitingFactor_NO_WEATHER_REPORTING_WITHIN_15NM
	default:
		pbMsg.LimitingFactor = pb.LimitingFactor_LIMITING_FACTOR_NONE
	}

	for _, sr := range r.StationReports {
		cat := pb.FlightRuleCategory_CATEGORY_UNSPECIFIED
		switch sr.Category {
		case models.FlightCategoryVFR:
			cat = pb.FlightRuleCategory_CATEGORY_VFR
		case models.FlightCategoryMVFR:
			cat = pb.FlightRuleCategory_CATEGORY_MVFR
		case models.FlightCategoryIFR:
			cat = pb.FlightRuleCategory_CATEGORY_IFR
		case models.FlightCategoryLIFR:
			cat = pb.FlightRuleCategory_CATEGORY_LIFR
		}

		pbMsg.StationReports = append(pbMsg.StationReports, &pb.StationComplianceReport{
			StationId:            sr.StationID,
			IsCompliant:          sr.IsCompliant,
			ActualCeilingFt:      sr.ActualCeilingFt,
			RequiredCeilingFt:    sr.RequiredCeilingFt,
			CeilingMarginFt:      sr.CeilingMarginFt,
			ActualVisibilitySm:   sr.ActualVisSM,
			RequiredVisibilitySm: sr.RequiredVisSM,
			VisibilityMarginSm:   sr.VisMarginSM,
			Category:             cat,
			Explanation:          sr.Explanation,
		})
	}

	for _, h := range r.ActiveHazards {
		hc := h
		pbMsg.ActiveHazards = append(pbMsg.ActiveHazards, ConvertHazardToPB(&hc))
	}

	return pbMsg
}
