package eink

import "image"

// Panel is the port to an e-paper display. Frames are portrait
// ScreenWidth x ScreenHeight, as drawn by Render; the adapter does any
// rotation to the hardware's native orientation. Sleep is called after
// every update. Partial refresh comes with the refresh loop.
type Panel interface {
	// Full shows frame with a full refresh.
	Full(frame image.Image) error
	// Sleep puts the panel into its low-power state.
	Sleep() error
	// Close releases the panel's resources.
	Close() error
}
