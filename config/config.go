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

// Load reads and parses the sources config file at path. The error
// never includes the file's contents (source URLs are secrets), only the
// path and the underlying cause.
func Load(path string) ([]Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading %q: %w", path, err)
	}

	var sources []Source
	if err := json.Unmarshal(data, &sources); err != nil {
		return nil, fmt.Errorf("config: parsing %q: %w", path, err)
	}
	return sources, nil
}
