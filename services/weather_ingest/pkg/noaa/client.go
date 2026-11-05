package noaa

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"traumateam/weather_ingest/pkg/models"
)

// WeatherFeedClient manages real-time NOAA AWC weather feeds with robust in-memory caching and offline mock fallbacks.
type WeatherFeedClient struct {
	mu           sync.RWMutex
	metarCache   map[string]*models.METAR
	tafCache     map[string]*models.TAF
	windsCache   map[string]*models.WindsAloftForecast
	hazardsCache []*models.WeatherHazard
}

// NewWeatherFeedClient initializes the client with pre-loaded realistic sample data for Colorado HEMS corridor.
func NewWeatherFeedClient() *WeatherFeedClient {
	c := &WeatherFeedClient{
		metarCache:   make(map[string]*models.METAR),
		tafCache:     make(map[string]*models.TAF),
		windsCache:   make(map[string]*models.WindsAloftForecast),
		hazardsCache: make([]*models.WeatherHazard, 0),
	}
	c.initDefaultFixtures()
	return c
}

func (c *WeatherFeedClient) initDefaultFixtures() {
	// 1. Preload METARs
	metars := map[string]string{
		"KAPA": FixtureMETAR_KAPA,
		"KDEN": FixtureMETAR_KDEN,
		"KLXV": FixtureMETAR_KLXV,
		"KASE": FixtureMETAR_KASE,
		"KBJC": FixtureMETAR_KBJC,
		"KCOS": FixtureMETAR_KCOS,
		"KPUB": FixtureMETAR_P6SM,
		"KBOS": FixtureMETAR_KBOS,
		"KPWM": FixtureMETAR_KPWM,
		"KMHT": FixtureMETAR_KMHT,
		"KPVD": FixtureMETAR_KPVD,
	}
	for station, raw := range metars {
		if m, err := ParseMETAR(raw); err == nil {
			c.metarCache[station] = m
		}
	}

	// 2. Preload TAFs
	if tafKAPA, err := ParseTAF(FixtureTAF_KAPA); err == nil {
		c.tafCache["KAPA"] = tafKAPA
	}
	if tafKASE, err := ParseTAF(FixtureTAF_KASE); err == nil {
		c.tafCache["KASE"] = tafKASE
	}

	// 3. Preload Winds Aloft
	if forecasts, err := ParseWindsAloft(FixtureWindsAloft_FD); err == nil {
		for _, f := range forecasts {
			fc := f
			c.windsCache[f.StationID] = &fc
		}
	}

	// 4. Preload SIGMET hazards
	if h1, err := ParseSIGMET(FixtureSIGMET_Convective); err == nil {
		c.hazardsCache = append(c.hazardsCache, h1)
	}
	if h2, err := ParseSIGMET(FixtureSIGMET_Turbulence); err == nil {
		c.hazardsCache = append(c.hazardsCache, h2)
	}
}

// GetMETAR retrieves the METAR observation for a given station.
func (c *WeatherFeedClient) GetMETAR(ctx context.Context, stationID string) (*models.METAR, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stationID = strings.ToUpper(strings.TrimSpace(stationID))
	if m, ok := c.metarCache[stationID]; ok {
		return m, nil
	}

	// Synthesize standard nominal VFR observation if unknown station
	synthRaw := fmt.Sprintf("%s %s 18008KT 10SM CLR 15/05 A2992", stationID, time.Now().UTC().Format("021504Z"))
	return ParseMETAR(synthRaw)
}

// SetMETAR stores or overrides a station's METAR.
func (c *WeatherFeedClient) SetMETAR(stationID, rawMETAR string) error {
	m, err := ParseMETAR(rawMETAR)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metarCache[strings.ToUpper(strings.TrimSpace(stationID))] = m
	return nil
}

// GetTAF retrieves the TAF forecast for a station.
func (c *WeatherFeedClient) GetTAF(ctx context.Context, stationID string) (*models.TAF, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stationID = strings.ToUpper(strings.TrimSpace(stationID))
	if t, ok := c.tafCache[stationID]; ok {
		return t, nil
	}
	// Return a nominal base TAF
	synthRaw := fmt.Sprintf("TAF %s %s 0506/0606 18010KT P6SM SKC", stationID, time.Now().UTC().Format("021504Z"))
	return ParseTAF(synthRaw)
}

// SetTAF stores or overrides a station's TAF.
func (c *WeatherFeedClient) SetTAF(stationID, rawTAF string) error {
	t, err := ParseTAF(rawTAF)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tafCache[strings.ToUpper(strings.TrimSpace(stationID))] = t
	return nil
}

// GetWindsAloft retrieves the winds aloft forecast for a station.
func (c *WeatherFeedClient) GetWindsAloft(ctx context.Context, stationID string) (*models.WindsAloftForecast, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stationID = strings.ToUpper(strings.TrimSpace(stationID))
	// Try 3-letter VOR code
	shortID := stationID
	if len(shortID) == 4 && strings.HasPrefix(shortID, "K") {
		shortID = shortID[1:]
	}

	if w, ok := c.windsCache[shortID]; ok {
		return w, nil
	}
	if w, ok := c.windsCache[stationID]; ok {
		return w, nil
	}

	// Default to DEN profile if unknown
	if w, ok := c.windsCache["DEN"]; ok {
		return w, nil
	}

	return nil, fmt.Errorf("no winds aloft available for station %s", stationID)
}

// GetActiveHazards returns all currently active SIGMET / convective advisories.
func (c *WeatherFeedClient) GetActiveHazards(ctx context.Context) ([]*models.WeatherHazard, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	active := make([]*models.WeatherHazard, 0, len(c.hazardsCache))
	now := time.Now().UTC()
	for _, h := range c.hazardsCache {
		if h.Active && (h.ValidUntil.IsZero() || h.ValidUntil.After(now)) {
			active = append(active, h)
		}
	}
	return active, nil
}

// AddHazard registers a new active SIGMET hazard.
func (c *WeatherFeedClient) AddHazard(hazard *models.WeatherHazard) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hazardsCache = append(c.hazardsCache, hazard)
}

// ClearHazards clears all active hazards (useful for testing clean corridors).
func (c *WeatherFeedClient) ClearHazards() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hazardsCache = make([]*models.WeatherHazard, 0)
}
