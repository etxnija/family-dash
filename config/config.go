// Package config parses the sources configuration file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Source describes one calendar source entry in the sources config file.
type Source struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// School describes the school lunch feed configured for the dashboard.
type School struct {
	Name   string `json:"name"`
	RSSURL string `json:"rssUrl"`
}

// Config is the top-level shape of the sources config file.
type Config struct {
	Sources []Source `json:"sources"`
	School  School   `json:"school"`
}

// Load reads and parses the sources config file at path. The error
// never includes the file's contents (source URLs are secrets), only the
// path and the underlying cause.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("config: reading %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: parsing %q: %w", path, err)
	}
	return cfg, nil
}
