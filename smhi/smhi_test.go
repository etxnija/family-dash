package smhi_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"family-dashboard/smhi"
)

// exampleResponse is a realistic (trimmed) SMHI point forecast response,
// covering two entries "today" (2026-09-22) and one entry "tomorrow"
// (2026-09-23).
const exampleResponse = `{
  "timeSeries": [
    {
      "validTime": "2026-09-22T12:00:00Z",
      "time": "2026-09-22T12:00:00Z",
      "parameters": [],
      "data": {
        "air_temperature": 10.9,
        "symbol_code": 1,
        "precipitation_amount_mean_deterministic": 0.0
      }
    },
    {
      "validTime": "2026-09-22T18:00:00Z",
      "time": "2026-09-22T18:00:00Z",
      "parameters": [],
      "data": {
        "air_temperature": 7.2,
        "symbol_code": 1,
        "precipitation_amount_mean_deterministic": 0.4
      }
    },
    {
      "validTime": "2026-09-23T12:00:00Z",
      "time": "2026-09-23T12:00:00Z",
      "parameters": [],
      "data": {
        "air_temperature": 15.5,
        "symbol_code": 5,
        "precipitation_amount_mean_deterministic": 0.0
      }
    }
  ]
}`

const emptyTimeSeriesResponse = `{"timeSeries": []}`

// stubRoundTripper returns a canned response or error without touching the
// network.
type stubRoundTripper struct {
	resp *http.Response
	err  error
}

func (s stubRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return s.resp, s.err
}

func mustLoadStockholm(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatalf("time.LoadLocation() error = %v", err)
	}
	return loc
}

func TestWeather(t *testing.T) {
	loc := mustLoadStockholm(t)

	tests := []struct {
		name                      string
		statusCode                int
		body                      string
		now                       time.Time
		wantTemp                  float64
		wantLowTemp               float64
		wantHighTemp              float64
		wantSunny                 bool
		wantPrecipitationExpected bool
		wantErr                   bool
	}{
		{
			// exampleResponse's today entries accumulate to 0.4mm, which is
			// >= dailyPrecipitationThresholdMM (0.2).
			name:                      "example response yields first entry's temperature and today's low/high/precipitation",
			statusCode:                http.StatusOK,
			body:                      exampleResponse,
			now:                       time.Date(2026, 9, 22, 8, 0, 0, 0, loc),
			wantTemp:                  10.9,
			wantLowTemp:               7.2,
			wantHighTemp:              10.9,
			wantSunny:                 true,
			wantPrecipitationExpected: true,
		},
		{
			name:         "not sunny when first entry's symbol_code is 3 or higher",
			statusCode:   http.StatusOK,
			body:         `{"timeSeries":[{"time":"2026-09-22T12:00:00Z","data":{"air_temperature":10.9,"symbol_code":5,"precipitation_amount_mean_deterministic":0.0}}]}`,
			now:          time.Date(2026, 9, 22, 8, 0, 0, 0, loc),
			wantTemp:     10.9,
			wantLowTemp:  10.9,
			wantHighTemp: 10.9,
			wantSunny:    false,
		},
		{
			name:                      "no precipitation when amount below threshold",
			statusCode:                http.StatusOK,
			body:                      `{"timeSeries":[{"time":"2026-09-22T12:00:00Z","data":{"air_temperature":10.9,"symbol_code":1,"precipitation_amount_mean_deterministic":0.05}}]}`,
			now:                       time.Date(2026, 9, 22, 8, 0, 0, 0, loc),
			wantTemp:                  10.9,
			wantLowTemp:               10.9,
			wantHighTemp:              10.9,
			wantSunny:                 true,
			wantPrecipitationExpected: false,
		},
		{
			name:                      "precipitation expected at exact threshold boundary",
			statusCode:                http.StatusOK,
			body:                      `{"timeSeries":[{"time":"2026-09-22T12:00:00Z","data":{"air_temperature":10.9,"symbol_code":1,"precipitation_amount_mean_deterministic":0.1}}]}`,
			now:                       time.Date(2026, 9, 22, 8, 0, 0, 0, loc),
			wantTemp:                  10.9,
			wantLowTemp:               10.9,
			wantHighTemp:              10.9,
			wantSunny:                 true,
			wantPrecipitationExpected: true,
		},
		{
			name:         "high temperature is max of today's entries",
			statusCode:   http.StatusOK,
			body:         exampleResponse,
			now:          time.Date(2026, 9, 23, 8, 0, 0, 0, loc),
			wantTemp:     10.9,
			wantLowTemp:  15.5,
			wantHighTemp: 15.5,
			wantSunny:    true,
		},
		{
			name:         "near-midnight now with no matching today entries falls back to first entry's temperature",
			statusCode:   http.StatusOK,
			body:         exampleResponse,
			now:          time.Date(2026, 9, 30, 0, 30, 0, 0, loc),
			wantTemp:     10.9,
			wantLowTemp:  10.9,
			wantHighTemp: 10.9,
			wantSunny:    true,
		},
		{
			name:                      "all hours dry stays torrt",
			statusCode:                http.StatusOK,
			body:                      `{"timeSeries":[{"time":"2026-09-22T06:00:00Z","data":{"air_temperature":10.0,"symbol_code":1,"precipitation_amount_mean_deterministic":0.0}},{"time":"2026-09-22T18:00:00Z","data":{"air_temperature":8.0,"symbol_code":1,"precipitation_amount_mean_deterministic":0.0}}]}`,
			now:                       time.Date(2026, 9, 22, 8, 0, 0, 0, loc),
			wantTemp:                  10.0,
			wantLowTemp:               8.0,
			wantHighTemp:              10.0,
			wantSunny:                 true,
			wantPrecipitationExpected: false,
		},
		{
			// No single entry reaches the old per-entry threshold (0.1), but
			// the accumulated total (0.25mm) crosses dailyPrecipitationThresholdMM.
			name:                      "sum crossing threshold triggers regn",
			statusCode:                http.StatusOK,
			body:                      `{"timeSeries":[{"time":"2026-09-22T00:00:00Z","data":{"air_temperature":5.0,"symbol_code":1,"precipitation_amount_mean_deterministic":0.05}},{"time":"2026-09-22T06:00:00Z","data":{"air_temperature":6.0,"symbol_code":1,"precipitation_amount_mean_deterministic":0.05}},{"time":"2026-09-22T12:00:00Z","data":{"air_temperature":7.0,"symbol_code":1,"precipitation_amount_mean_deterministic":0.05}},{"time":"2026-09-22T18:00:00Z","data":{"air_temperature":8.0,"symbol_code":1,"precipitation_amount_mean_deterministic":0.05}}]}`,
			now:                       time.Date(2026, 9, 22, 8, 0, 0, 0, loc),
			wantTemp:                  5.0,
			wantLowTemp:               5.0,
			wantHighTemp:              8.0,
			wantSunny:                 true,
			wantPrecipitationExpected: true,
		},
		{
			name:                      "sum just below threshold stays torrt",
			statusCode:                http.StatusOK,
			body:                      `{"timeSeries":[{"time":"2026-09-22T12:00:00Z","data":{"air_temperature":10.9,"symbol_code":1,"precipitation_amount_mean_deterministic":0.095}}]}`,
			now:                       time.Date(2026, 9, 22, 8, 0, 0, 0, loc),
			wantTemp:                  10.9,
			wantLowTemp:               10.9,
			wantHighTemp:              10.9,
			wantSunny:                 true,
			wantPrecipitationExpected: false,
		},
		{
			name:       "empty time series yields error",
			statusCode: http.StatusOK,
			body:       emptyTimeSeriesResponse,
			now:        time.Date(2026, 9, 22, 8, 0, 0, 0, loc),
			wantErr:    true,
		},
		{
			name:       "HTTP error yields error",
			statusCode: http.StatusNotFound,
			body:       "",
			now:        time.Date(2026, 9, 22, 8, 0, 0, 0, loc),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				Transport: stubRoundTripper{
					resp: &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
					},
				},
			}

			c := smhi.New(59.33, 18.07, client)
			got, err := c.Weather(context.Background(), tt.now)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Weather() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Weather() error = %v", err)
			}
			if got.Temperature != tt.wantTemp {
				t.Errorf("Weather().Temperature = %v, want %v", got.Temperature, tt.wantTemp)
			}
			if got.LowTemperature != tt.wantLowTemp {
				t.Errorf("Weather().LowTemperature = %v, want %v", got.LowTemperature, tt.wantLowTemp)
			}
			if got.HighTemperature != tt.wantHighTemp {
				t.Errorf("Weather().HighTemperature = %v, want %v", got.HighTemperature, tt.wantHighTemp)
			}
			if got.Sunny != tt.wantSunny {
				t.Errorf("Weather().Sunny = %v, want %v", got.Sunny, tt.wantSunny)
			}
			if got.PrecipitationExpected != tt.wantPrecipitationExpected {
				t.Errorf("Weather().PrecipitationExpected = %v, want %v", got.PrecipitationExpected, tt.wantPrecipitationExpected)
			}
		})
	}
}
