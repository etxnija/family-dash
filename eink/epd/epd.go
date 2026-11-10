//go:build linux && arm

package epd

import (
	"github.com/ditrytus/epd7in5v2"

	"family-dashboard/eink"
)

var _ eink.Panel = (*Panel)(nil)

// Panel drives the Waveshare 7.5" V2 over GPIO and SPI. The logic lives in
// devicePanel (device.go) so it can be tested on the host.
type Panel struct {
	*devicePanel
}

// New opens the panel's GPIO pins and SPI port. The caller must Close it.
func New(cable Cable) (*Panel, error) {
	p, err := openDevice(epd7in5v2.NewEPD(), epd7in5v2.ScreenBounds, cable)
	if err != nil {
		return nil, err
	}
	return &Panel{p}, nil
}
