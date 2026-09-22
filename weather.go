package dashboard

import (
	"context"
	"time"
)

// WeatherReading is a single point-in-time weather observation, plus
// today's forecast low, high, sky condition, and precipitation outlook.
type WeatherReading struct {
	Temperature           float64
	LowTemperature        float64
	HighTemperature       float64
	Sunny                 bool
	PrecipitationExpected bool
}

// WeatherSource is a port for fetching the current weather; adapters
// implement it. now is the reference time used to determine "today" for
// the forecast fields.
type WeatherSource interface {
	Weather(ctx context.Context, now time.Time) (WeatherReading, error)
}
