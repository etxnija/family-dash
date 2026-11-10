// Package pngpanel is an eink.Panel that writes each frame to a PNG file,
// so layouts can be built and checked without the hardware.
package pngpanel

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"

	"family-dashboard/eink"
)

// Panel writes every refresh, full or partial, to a PNG file at path.
type Panel struct {
	path string
}

// New returns a Panel that writes frames to path.
func New(path string) *Panel {
	return &Panel{path: path}
}

// Full encodes frame as a PNG file at the panel's path. frame must be
// eink.ScreenWidth x eink.ScreenHeight. If encoding fails, the partially
// written file is removed so a reader never sees a truncated PNG.
func (p *Panel) Full(frame image.Image) error {
	return p.write(frame)
}

// Partial writes the whole frame exactly like Full and ignores rect, so the
// file always shows the latest frame.
func (p *Panel) Partial(frame image.Image, rect image.Rectangle) error {
	return p.write(frame)
}

// write checks frame's size and encodes it to the panel's path.
func (p *Panel) write(frame image.Image) error {
	if b := frame.Bounds(); b.Dx() != eink.ScreenWidth || b.Dy() != eink.ScreenHeight {
		return fmt.Errorf("pngpanel: frame bounds %v, want %dx%d", b, eink.ScreenWidth, eink.ScreenHeight)
	}
	f, err := os.Create(p.path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, frame); err != nil {
		return errors.Join(err, f.Close(), os.Remove(p.path))
	}
	return f.Close()
}

// Sleep does nothing; a file has no low-power state.
func (p *Panel) Sleep() error { return nil }

// Close does nothing; every refresh closes the file it writes.
func (p *Panel) Close() error { return nil }
