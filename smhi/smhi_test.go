package smhi_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"family-dashboard/smhi"
)

// exampleResponse is a realistic (trimmed) SMHI point forecast response.
const exampleResponse = `{
  "timeSeries": [
    {
      "validTime": "2026-09-22T12:00:00Z",
      "time": "2026-09-22T12:00:00Z",
      "parameters": [],
      "data": {
        "air_temperature": 10.9
      }
    },
    {
      "validTime": "2026-09-22T13:00:00Z",
      "time": "2026-09-22T13:00:00Z",
      "parameters": [],
      "data": {
        "air_temperature": 11.4
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

func TestWeather(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantTemp   float64
		wantErr    bool
	}{
		{
			name:       "example response yields first entry's temperature",
			statusCode: http.StatusOK,
			body:       exampleResponse,
			wantTemp:   10.9,
		},
		{
			name:       "empty time series yields error",
			statusCode: http.StatusOK,
			body:       emptyTimeSeriesResponse,
			wantErr:    true,
		},
		{
			name:       "HTTP error yields error",
			statusCode: http.StatusNotFound,
			body:       "",
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
			got, err := c.Weather(context.Background())
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
		})
	}
}
