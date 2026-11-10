package pngpanel_test

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"

	"family-dashboard/eink"
	"family-dashboard/eink/pngpanel"
	"family-dashboard/eink/pngpanel/pngpaneltest"
)

func TestPanel_FullAndPartial(t *testing.T) {
	dir := t.TempDir()
	frame := image.NewGray(image.Rect(0, 0, eink.ScreenWidth, eink.ScreenHeight))
	clock := eink.ClockRect()
	full := func(p *pngpanel.Panel, f image.Image) error { return p.Full(f) }
	partial := func(p *pngpanel.Panel, f image.Image) error { return p.Partial(f, clock) }

	tests := []struct {
		name    string
		call    func(*pngpanel.Panel, image.Image) error
		path    string
		frame   image.Image
		wantErr bool
	}{
		{
			name:  "Full writes decodable 480x800 frame",
			call:  full,
			path:  filepath.Join(dir, "frame.png"),
			frame: frame,
		},
		{
			name:    "Full rejects landscape frame without creating a file",
			call:    full,
			path:    filepath.Join(dir, "landscape.png"),
			frame:   image.NewGray(image.Rect(0, 0, eink.ScreenHeight, eink.ScreenWidth)),
			wantErr: true,
		},
		// png.Encode would reject a zero-size image after os.Create had made
		// the file; the size check rejects it first, so nothing is left behind.
		{
			name:    "Full rejects empty frame without creating a file",
			call:    full,
			path:    filepath.Join(dir, "empty.png"),
			frame:   image.NewGray(image.Rect(0, 0, 0, 0)),
			wantErr: true,
		},
		{
			name:    "Full to missing directory",
			call:    full,
			path:    filepath.Join(dir, "missing", "frame.png"),
			frame:   frame,
			wantErr: true,
		},
		{
			name:  "Partial writes the whole decodable 480x800 frame",
			call:  partial,
			path:  filepath.Join(dir, "partial.png"),
			frame: frame,
		},
		{
			name:    "Partial rejects wrong-size frame without creating a file",
			call:    partial,
			path:    filepath.Join(dir, "small.png"),
			frame:   image.NewGray(clock),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(pngpanel.New(tt.path), tt.frame)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("write to %q = nil error, want error", tt.path)
				}
				if _, statErr := os.Stat(tt.path); !errors.Is(statErr, os.ErrNotExist) {
					t.Errorf("os.Stat(%q) error = %v, want file not to exist", tt.path, statErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("write to %q unexpected error: %v", tt.path, err)
			}
			pngpaneltest.AssertFrame(t, tt.path)
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
