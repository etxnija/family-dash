// Package smhi adapts the SMHI open-data point forecast API to the
// dashboard.WeatherSource port.
package smhi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	dashboard "family-dashboard"
)

// forecast is the top-level JSON structure of the SMHI point forecast
// response.
type forecast struct {
	TimeSeries []timeSeriesEntry `json:"timeSeries"`
}

// timeSeriesEntry is a single forecast point in time.
type timeSeriesEntry struct {
	Time string `json:"time"`
	Data struct {
		AirTemperature      float64 `json:"air_temperature"`
		SymbolCode          int     `json:"symbol_code"`
		PrecipitationAmount float64 `json:"precipitation_amount_mean_deterministic"`
	} `json:"data"`
}

// dailyPrecipitationThresholdMM is the minimum accumulated precipitation
// (mm) across today's entries for the day to count as "precipitation
// expected".
const dailyPrecipitationThresholdMM = 0.2

// Client fetches the SMHI point forecast for a fixed location.
type Client struct {
	lat        float64
	lon        float64
	httpClient *http.Client
}

// New builds a Client for the given coordinates. The caller supplies the
// *http.Client so tests can inject a stub transport instead of hitting the
// network.
func New(lat, lon float64, client *http.Client) *Client {
	return &Client{lat: lat, lon: lon, httpClient: client}
}

// Weather implements dashboard.WeatherSource by fetching the SMHI point
// forecast and returning the first (soonest) time series entry's
// temperature, plus today's low/high temperature, sky condition, and
// precipitation outlook.
func (c *Client) Weather(ctx context.Context, now time.Time) (dashboard.WeatherReading, error) {
	data, err := c.fetch(ctx)
	if err != nil {
		return dashboard.WeatherReading{}, err
	}
	return parseForecast(data, now)
}

// fetch retrieves the raw forecast JSON over HTTP.
func (c *Client) fetch(ctx context.Context) ([]byte, error) {
	url := fmt.Sprintf("https://opendata-download-metfcst.smhi.se/api/category/snow1g/version/1/geotype/point/lon/%f/lat/%f/data.json", c.lon, c.lat)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("smhi: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("smhi: fetch forecast: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("smhi: forecast returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("smhi: read forecast: %w", err)
	}
	return body, nil
}

// parseForecast parses raw forecast JSON into a dashboard.WeatherReading,
// using the first time series entry for Temperature and Sunny, and the
// entries whose local date matches now's local date (in Europe/Stockholm,
// falling back to UTC if the zone fails to load) for LowTemperature,
// HighTemperature, and PrecipitationExpected. An empty timeSeries is an
// error, since there is no sensible zero-value reading to fall back to.
func parseForecast(data []byte, now time.Time) (dashboard.WeatherReading, error) {
	var f forecast
	if err := json.Unmarshal(data, &f); err != nil {
		return dashboard.WeatherReading{}, fmt.Errorf("smhi: parse forecast: %w", err)
	}

	if len(f.TimeSeries) == 0 {
		return dashboard.WeatherReading{}, fmt.Errorf("smhi: forecast has no time series entries")
	}

	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		loc = time.UTC
	}
	nowDate := now.In(loc).Format("2006-01-02")

	first := f.TimeSeries[0]
	reading := dashboard.WeatherReading{
		Temperature: first.Data.AirTemperature,
		Sunny:       first.Data.SymbolCode == 1 || first.Data.SymbolCode == 2,
	}

	low := first.Data.AirTemperature
	high := first.Data.AirTemperature
	precipTotal := first.Data.PrecipitationAmount
	haveToday := false
	for _, entry := range f.TimeSeries {
		entryTime, err := time.Parse(time.RFC3339, entry.Time)
		if err != nil {
			continue
		}
		if entryTime.In(loc).Format("2006-01-02") != nowDate {
			continue
		}
		if !haveToday || entry.Data.AirTemperature < low {
			low = entry.Data.AirTemperature
		}
		if !haveToday || entry.Data.AirTemperature > high {
			high = entry.Data.AirTemperature
		}
		precipTotal += entry.Data.PrecipitationAmount
		haveToday = true
	}
	reading.LowTemperature = low
	reading.HighTemperature = high
	reading.PrecipitationExpected = precipTotal >= dailyPrecipitationThresholdMM

	return reading, nil
}
