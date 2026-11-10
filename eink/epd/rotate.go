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
	if err := checkFrame(frame, cable); err != nil {
		return nil, err
	}
	out := image.NewGray(image.Rect(0, 0, nativeWidth, nativeHeight))
	rotateInto(out, image.Point{}, frame, image.Rect(0, 0, eink.ScreenWidth, eink.ScreenHeight), cable)
	return out, nil
}

// RotateRect rotates only rect of a portrait frame, for a partial refresh.
// rect is relative to the frame's own origin and must lie inside it. The
// native rect it covers is, for CableLeft, X [y0, y1) and Y [480-x1,
// 480-x0); for CableRight, X [800-y1, 800-y0) and Y [x0, x1). Its native X
// edges must be multiples of 8, as the panel's partial refresh requires.
//
// It returns an image of the native rect's size with its origin at (0,0)
// and the native rect itself. Every pixel equals Rotate(frame, cable) at
// the same point offset by the native rect's Min.
func RotateRect(frame image.Image, rect image.Rectangle, cable Cable) (*image.Gray, image.Rectangle, error) {
	if err := checkFrame(frame, cable); err != nil {
		return nil, image.Rectangle{}, err
	}
	if rect.Empty() {
		return nil, image.Rectangle{}, fmt.Errorf("rect %v is empty", rect)
	}
	if !rect.In(image.Rect(0, 0, eink.ScreenWidth, eink.ScreenHeight)) {
		return nil, image.Rectangle{}, fmt.Errorf("rect %v is outside the %dx%d frame", rect, eink.ScreenWidth, eink.ScreenHeight)
	}

	var native image.Rectangle
	if cable == CableLeft {
		native = image.Rect(rect.Min.Y, nativeHeight-rect.Max.X, rect.Max.Y, nativeHeight-rect.Min.X)
	} else {
		native = image.Rect(nativeWidth-rect.Max.Y, rect.Min.X, nativeWidth-rect.Min.Y, rect.Max.X)
	}
	if native.Min.X%8 != 0 || native.Max.X%8 != 0 {
		return nil, image.Rectangle{}, fmt.Errorf("native rect %v (from %v) has X edges that are not multiples of 8", native, rect)
	}

	out := image.NewGray(image.Rect(0, 0, native.Dx(), native.Dy()))
	rotateInto(out, native.Min, frame, rect, cable)
	return out, native, nil
}

// checkFrame validates cable and that frame is portrait
// eink.ScreenWidth x eink.ScreenHeight.
func checkFrame(frame image.Image, cable Cable) error {
	if cable != CableLeft && cable != CableRight {
		return fmt.Errorf("unknown cable %d", cable)
	}
	b := frame.Bounds()
	if b.Dx() != eink.ScreenWidth || b.Dy() != eink.ScreenHeight {
		return fmt.Errorf("frame is %dx%d, want %dx%d", b.Dx(), b.Dy(), eink.ScreenWidth, eink.ScreenHeight)
	}
	return nil
}

// rotateInto writes the portrait pixels in rect (relative to the frame's
// origin) to out at their native position minus origin, thresholded at
// 128. Rotate and RotateRect share it so their mapping cannot drift.
func rotateInto(out *image.Gray, origin image.Point, frame image.Image, rect image.Rectangle, cable Cable) {
	b := frame.Bounds()
	for py := rect.Min.Y; py < rect.Max.Y; py++ {
		for px := rect.Min.X; px < rect.Max.X; px++ {
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
			out.Pix[out.PixOffset(x-origin.X, y-origin.Y)] = v
		}
	}
}
