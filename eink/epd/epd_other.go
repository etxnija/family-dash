//go:build !(linux && arm)

package epd

import (
	"errors"
	"image"

	"family-dashboard/eink"
)

var errUnsupported = errors.New("e-paper panel is only supported on linux/arm")

var _ eink.Panel = (*Panel)(nil)

// Panel is a stub on platforms without the e-paper hardware.
type Panel struct{}

// New always fails off linux/arm.
func New(cable Cable) (*Panel, error) {
	return nil, errUnsupported
}

// Full always fails off linux/arm.
func (*Panel) Full(frame image.Image) error { return errUnsupported }

// Sleep always fails off linux/arm.
func (*Panel) Sleep() error { return errUnsupported }

// Close always fails off linux/arm.
func (*Panel) Close() error { return errUnsupported }
