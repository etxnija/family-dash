package dashboard

import "context"

// LunchMenu is a single day's school lunch menu.
type LunchMenu struct {
	Dishes []string
}

// LunchSource is a port for fetching the school lunch menu; adapters
// implement it.
type LunchSource interface {
	Lunch(ctx context.Context) (LunchMenu, error)
}
