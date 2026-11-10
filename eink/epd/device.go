package epd

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"time"

	"family-dashboard/eink"
)

// primeBeforePartial makes Partial write the shown image of the rect before
// the new one. On the Pi Zero W a partial refresh after about 60 s of deep
// sleep left the changed clock digit grainy and half-written (2 s gaps were
// clean); writing the previously shown image of the rect first, then the new
// image, fixed it. Working theory, not confirmed: the controller loses what
// is on screen during a long deep sleep, so the partial update diffs against
// bad data. Set it to false if a later driver no longer needs it.
const primeBeforePartial = true

// device is the part of the epd7in5v2 driver the panel uses. It is a seam so
// the call order and error wrapping can be tested on the host.
type device interface {
	GPIOInit() error
	Init() error
	DisplayImage(img image.Image) error
	InitPart() error
	DisplayPart(img image.Image, rect image.Rectangle) error
	Sleep() error
	Close() error
}

// devicePanel drives a device. bounds is the native frame size the device
// wants. Full calls Init before every full refresh, which also returns the
// controller from the partial mode Partial puts it in.
//
// shown is a copy of the last portrait frame a Full or Partial showed
// successfully (origin (0,0)), nil until then. When prime is set, Partial
// writes shown's rect before the new frame's (see primeBeforePartial).
type devicePanel struct {
	dev    device
	bounds image.Rectangle
	cable  Cable
	prime  bool
	shown  *image.Gray
}

// openDevice runs GPIOInit once; on failure it closes dev and joins both
// errors.
func openDevice(dev device, bounds image.Rectangle, cable Cable) (*devicePanel, error) {
	if err := dev.GPIOInit(); err != nil {
		return nil, fmt.Errorf("epd: gpio init: %w", errors.Join(err, dev.Close()))
	}
	return &devicePanel{dev: dev, bounds: bounds, cable: cable, prime: primeBeforePartial}, nil
}

// Full rotates frame to the native orientation and shows it with a full
// refresh.
func (p *devicePanel) Full(frame image.Image) error {
	start := time.Now()
	native, err := Rotate(frame, p.cable)
	if err != nil {
		return fmt.Errorf("epd: rotate: %w", err)
	}
	slog.Info("rotated e-paper frame", "rotate_ms", time.Since(start).Milliseconds(), "cable", p.cable)
	if native.Bounds() != p.bounds {
		return fmt.Errorf("epd: rotated frame is %v, driver wants %v", native.Bounds(), p.bounds)
	}

	start = time.Now()
	if err := p.dev.Init(); err != nil {
		return fmt.Errorf("epd: init: %w", err)
	}
	if err := p.dev.DisplayImage(native); err != nil {
		return fmt.Errorf("epd: display: %w", err)
	}
	slog.Info("full refresh", "refresh_ms", time.Since(start).Milliseconds())
	p.shown = copyFrame(frame)
	return nil
}

// Partial rotates rect of frame to the native orientation and shows it
// with a partial refresh. The rest of the panel is left as it is.
//
// When p.prime is set it first writes rect of the last shown frame, then
// rect of frame, after a single InitPart; it fails without touching the
// device if no frame has been shown yet. On success frame becomes the
// shown frame.
func (p *devicePanel) Partial(frame image.Image, rect image.Rectangle) error {
	if p.prime && p.shown == nil {
		return errors.New("epd: partial refresh before any frame was shown")
	}
	start := time.Now()
	small, native, err := RotateRect(frame, rect, p.cable)
	if err != nil {
		return fmt.Errorf("epd: rotate rect: %w", err)
	}
	var previous *image.Gray
	if p.prime {
		// p.shown is a valid portrait frame and rect passed above, so this
		// cannot fail in practice.
		if previous, _, err = RotateRect(p.shown, rect, p.cable); err != nil {
			return fmt.Errorf("epd: rotate rect (previous image): %w", err)
		}
	}
	slog.Info("rotated e-paper rect", "rotate_ms", time.Since(start).Milliseconds(), "cable", p.cable, "native_rect", native.String())
	if !native.In(p.bounds) {
		return fmt.Errorf("epd: native rect %v is outside driver bounds %v", native, p.bounds)
	}

	start = time.Now()
	if err := p.dev.InitPart(); err != nil {
		return fmt.Errorf("epd: init part: %w", err)
	}
	if previous != nil {
		if err := p.dev.DisplayPart(previous, native); err != nil {
			return fmt.Errorf("epd: display part (previous image): %w", err)
		}
	}
	if err := p.dev.DisplayPart(small, native); err != nil {
		return fmt.Errorf("epd: display part: %w", err)
	}
	slog.Info("partial refresh", "partial_ms", time.Since(start).Milliseconds(), "primed", p.prime)
	p.shown = copyFrame(frame)
	return nil
}

// copyFrame copies a portrait frame into a new *image.Gray with its origin
// at (0,0), reading relative to Bounds().Min and thresholding at 128 like
// rotateInto, so the copy is independent of frame and rotates identically.
func copyFrame(frame image.Image) *image.Gray {
	b := frame.Bounds()
	out := image.NewGray(image.Rect(0, 0, eink.ScreenWidth, eink.ScreenHeight))
	for y := 0; y < eink.ScreenHeight; y++ {
		for x := 0; x < eink.ScreenWidth; x++ {
			g := color.GrayModel.Convert(frame.At(b.Min.X+x, b.Min.Y+y)).(color.Gray)
			v := uint8(255)
			if g.Y < 128 {
				v = 0
			}
			out.Pix[out.PixOffset(x, y)] = v
		}
	}
	return out
}

// Sleep puts the panel into deep sleep.
func (p *devicePanel) Sleep() error {
	if err := p.dev.Sleep(); err != nil {
		return fmt.Errorf("epd: sleep: %w", err)
	}
	return nil
}

// Close releases the GPIO pins and the SPI port.
func (p *devicePanel) Close() error {
	if err := p.dev.Close(); err != nil {
		return fmt.Errorf("epd: close: %w", err)
	}
	return nil
}
