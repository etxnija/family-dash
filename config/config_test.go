package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"family-dashboard/config"
)

func TestLoadSources(t *testing.T) {
	t.Run("valid JSON file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "sources.json")
		const body = `[{"name":"alice","type":"sportadmin","url":"https://example.com/a.ics"},{"name":"bob","type":"sportadmin","url":"https://example.com/b.ics"}]`
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		sources, err := config.Load(path)
		if err != nil {
			t.Fatalf("Load(%q) unexpected error: %v", path, err)
		}
		want := []config.Source{
			{Name: "alice", Type: "sportadmin", URL: "https://example.com/a.ics"},
			{Name: "bob", Type: "sportadmin", URL: "https://example.com/b.ics"},
		}
		if len(sources) != len(want) {
			t.Fatalf("Load(%q) = %+v, want %+v", path, sources, want)
		}
		for i := range want {
			if sources[i] != want[i] {
				t.Errorf("sources[%d] = %+v, want %+v", i, sources[i], want[i])
			}
		}
	})

	t.Run("missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "does-not-exist.json")
		if _, err := config.Load(path); err == nil {
			t.Fatalf("Load(%q) = nil error, want error", path)
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

		if _, err := config.Load(f.Name()); err == nil {
			t.Fatalf("Load(%q) = nil error, want error", f.Name())
		}
	})
}
