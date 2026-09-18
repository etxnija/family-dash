package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSource(t *testing.T) {
	tests := []struct {
		name    string
		cfg     sourceConfig
		wantErr bool
	}{
		{
			name: "known type sportadmin",
			cfg:  sourceConfig{Name: "alice", Type: "sportadmin", URL: "https://example.com/feed.ics"},
		},
		{
			name:    "unknown type",
			cfg:     sourceConfig{Name: "bob", Type: "carrier-pigeon", URL: "https://example.com/feed.ics"},
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

func TestLoadSources(t *testing.T) {
	t.Run("valid JSON file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "sources.json")
		const body = `[{"name":"alice","type":"sportadmin","url":"https://example.com/a.ics"},{"name":"bob","type":"sportadmin","url":"https://example.com/b.ics"}]`
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		sources, err := loadSources(path)
		if err != nil {
			t.Fatalf("loadSources(%q) unexpected error: %v", path, err)
		}
		want := []sourceConfig{
			{Name: "alice", Type: "sportadmin", URL: "https://example.com/a.ics"},
			{Name: "bob", Type: "sportadmin", URL: "https://example.com/b.ics"},
		}
		if len(sources) != len(want) {
			t.Fatalf("loadSources(%q) = %+v, want %+v", path, sources, want)
		}
		for i := range want {
			if sources[i] != want[i] {
				t.Errorf("sources[%d] = %+v, want %+v", i, sources[i], want[i])
			}
		}
	})

	t.Run("missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "does-not-exist.json")
		if _, err := loadSources(path); err == nil {
			t.Fatalf("loadSources(%q) = nil error, want error", path)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "sources-*.json")
		if err != nil {
			t.Fatalf("CreateTemp: %v", err)
		}
		if _, err := f.WriteString(`{not valid json`); err != nil {
			t.Fatalf("WriteString: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}

		if _, err := loadSources(f.Name()); err == nil {
			t.Fatalf("loadSources(%q) = nil error, want error", f.Name())
		}
	})
}
