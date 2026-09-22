package dashboard

import "context"

// WeatherReading is a single point-in-time weather observation.
type WeatherReading struct {
	Temperature float64
}

// WeatherSource is a port for fetching the current weather; adapters
// implement it.
type WeatherSource interface {
	Weather(ctx context.Context) (WeatherReading, error)
}
