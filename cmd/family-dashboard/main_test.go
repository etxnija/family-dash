package main

import (
	"testing"

	"family-dashboard/config"
)

func TestBuildSource(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Source
		wantErr bool
	}{
		{
			name: "known type sportadmin",
			cfg:  config.Source{Name: "alice", Type: "sportadmin", URL: "https://example.com/feed.ics"},
		},
		{
			name: "known type infomentor",
			cfg:  config.Source{Name: "alice", Type: "infomentor", URL: "https://example.com/feed.ics"},
		},
		{
			name:    "unknown type",
			cfg:     config.Source{Name: "bob", Type: "carrier-pigeon", URL: "https://example.com/feed.ics"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := buildSource(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("buildSource(%+v) = nil error, want error", tt.cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildSource(%+v) unexpected error: %v", tt.cfg, err)
			}
			if source == nil {
				t.Fatalf("buildSource(%+v) = nil source, want non-nil", tt.cfg)
			}
		})
	}
}
