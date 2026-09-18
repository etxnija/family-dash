package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	dashboard "family-dashboard"
	"family-dashboard/webcal"
)

// sourceConfig describes one calendar source entry in the sources config file.
type sourceConfig struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// loadSources reads and parses the sources config file at path. The error
// never includes the file's contents (source URLs are secrets), only the
// path and the underlying cause.
func loadSources(path string) ([]sourceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading %q: %w", path, err)
	}

	var sources []sourceConfig
	if err := json.Unmarshal(data, &sources); err != nil {
		return nil, fmt.Errorf("config: parsing %q: %w", path, err)
	}
	return sources, nil
}

// buildSource constructs the dashboard.EventSource adapter for cfg.
func buildSource(cfg sourceConfig) (dashboard.EventSource, error) {
	switch cfg.Type {
	case "sportadmin":
		return webcal.New(cfg.URL, http.DefaultClient), nil
	default:
		return nil, fmt.Errorf("config: unknown source type %q for %q", cfg.Type, cfg.Name)
	}
}
