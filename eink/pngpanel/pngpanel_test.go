package pngpanel_test

import (
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"family-dashboard/eink"
	"family-dashboard/eink/pngpanel"
)

func TestPanel_Full(t *testing.T) {
	dir := t.TempDir()
	frame := image.NewGray(image.Rect(0, 0, eink.ScreenWidth, eink.ScreenHeight))

	tests := []struct {
		name    string
		path    string
		frame   image.Image
		wantErr bool
	}{
		{
			name:  "writes decodable 480x800 frame",
			path:  filepath.Join(dir, "frame.png"),
			frame: frame,
		},
		{
			name:    "landscape frame is rejected without creating a file",
			path:    filepath.Join(dir, "landscape.png"),
			frame:   image.NewGray(image.Rect(0, 0, eink.ScreenHeight, eink.ScreenWidth)),
			wantErr: true,
		},
		// png.Encode would reject a zero-size image after os.Create had made
		// the file; the size check rejects it first, so nothing is left behind.
		{
			name:    "empty frame is rejected without creating a file",
			path:    filepath.Join(dir, "empty.png"),
			frame:   image.NewGray(image.Rect(0, 0, 0, 0)),
			wantErr: true,
		},
		{
			name:    "missing directory",
			path:    filepath.Join(dir, "missing", "frame.png"),
			frame:   frame,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pngpanel.New(tt.path).Full(tt.frame)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Full() to %q = nil error, want error", tt.path)
				}
				if _, statErr := os.Stat(tt.path); !errors.Is(statErr, os.ErrNotExist) {
					t.Errorf("os.Stat(%q) error = %v, want file not to exist", tt.path, statErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Full() to %q unexpected error: %v", tt.path, err)
			}

			f, err := os.Open(tt.path)
			if err != nil {
				t.Fatalf("opening written PNG: %v", err)
			}
			decoded, err := png.Decode(f)
			if cerr := f.Close(); cerr != nil {
				t.Fatalf("closing written PNG: %v", cerr)
			}
			if err != nil {
				t.Fatalf("png.Decode() unexpected error: %v", err)
			}
			if b := decoded.Bounds(); b.Dx() != eink.ScreenWidth || b.Dy() != eink.ScreenHeight {
				t.Errorf("decoded bounds = %v, want %dx%d", b, eink.ScreenWidth, eink.ScreenHeight)
			}
		})
	}
}

func TestPanel_SleepAndClose(t *testing.T) {
	p := pngpanel.New(filepath.Join(t.TempDir(), "frame.png"))

	tests := []struct {
		name string
		call func() error
	}{
		{name: "Sleep", call: p.Sleep},
		{name: "Close", call: p.Close},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err != nil {
				t.Errorf("%s() = %v, want nil", tt.name, err)
			}
		})
	}
}
