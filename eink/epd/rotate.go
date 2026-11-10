// Package epd is the e-paper panel adapter for the Waveshare 7.5" V2. It
// turns the portrait frames drawn by eink.Render into the panel's native
// landscape orientation (ADR-0004). The hardware driver is only built on
// linux/arm; on other platforms New returns an error.
package epd

import (
	"fmt"
	"image"
	"image/color"

	"family-dashboard/eink"
)

const (
	// nativeWidth and nativeHeight are the panel's landscape size.
	nativeWidth  = 800
	nativeHeight = 480
)

// Cable is the side of the hung portrait frame where the panel's ribbon
// cable exits. It decides which way the frame is rotated.
type Cable int

const (
	// CableLeft means the cable exits on the left of the portrait frame.
	CableLeft Cable = iota
	// CableRight means the cable exits on the right of the portrait frame.
	CableRight
)

// ParseCable parses "left" or "right" into a Cable.
func ParseCable(s string) (Cable, error) {
	switch s {
	case "left":
		return CableLeft, nil
	case "right":
		return CableRight, nil
	}
	return 0, fmt.Errorf("unknown cable side %q, want \"left\" or \"right\"", s)
}

// Rotate turns a portrait eink.ScreenWidth x eink.ScreenHeight frame into
// the panel's native 800x480 orientation for cable. Pixels are
// thresholded at 128 like the renderer, so the result is strictly 1-bit.
func Rotate(frame image.Image, cable Cable) (*image.Gray, error) {
	if cable != CableLeft && cable != CableRight {
		return nil, fmt.Errorf("unknown cable %d", cable)
	}
	b := frame.Bounds()
	if b.Dx() != eink.ScreenWidth || b.Dy() != eink.ScreenHeight {
		return nil, fmt.Errorf("frame is %dx%d, want %dx%d", b.Dx(), b.Dy(), eink.ScreenWidth, eink.ScreenHeight)
	}

	out := image.NewGray(image.Rect(0, 0, nativeWidth, nativeHeight))
	for py := range eink.ScreenHeight {
		for px := range eink.ScreenWidth {
			var x, y int
			if cable == CableLeft {
				x, y = py, nativeHeight-1-px
			} else {
				x, y = nativeWidth-1-py, px
			}
			g := color.GrayModel.Convert(frame.At(b.Min.X+px, b.Min.Y+py)).(color.Gray)
			v := uint8(255)
			if g.Y < 128 {
				v = 0
			}
			out.Pix[out.PixOffset(x, y)] = v
		}
	}
	return out, nil
}
