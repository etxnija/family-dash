package epd

import (
	"errors"
	"fmt"
	"image"
	"log/slog"
	"time"
)

// device is the part of the epd7in5v2 driver the panel uses. It is a seam so
// the call order and error wrapping can be tested on the host.
type device interface {
	GPIOInit() error
	Init() error
	DisplayImage(img image.Image) error
	Sleep() error
	Close() error
}

// devicePanel drives a device. bounds is the native frame size the device
// wants.
type devicePanel struct {
	dev    device
	bounds image.Rectangle
	cable  Cable
}

// openDevice runs GPIOInit once; on failure it closes dev and joins both
// errors.
func openDevice(dev device, bounds image.Rectangle, cable Cable) (*devicePanel, error) {
	if err := dev.GPIOInit(); err != nil {
		return nil, fmt.Errorf("epd: gpio init: %w", errors.Join(err, dev.Close()))
	}
	return &devicePanel{dev: dev, bounds: bounds, cable: cable}, nil
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
	return nil
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
