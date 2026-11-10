// Package pngpaneltest holds test helpers for code that writes frames
// through pngpanel.
package pngpaneltest

import (
	"image/png"
	"os"
	"testing"

	"family-dashboard/eink"
)

// AssertFrame fails t unless path holds a decodable PNG of the portrait
// screen size.
func AssertFrame(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
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
}
