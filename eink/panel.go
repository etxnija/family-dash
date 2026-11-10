package eink

import "image"

// Panel is the port to an e-paper display. Frames are portrait
// ScreenWidth x ScreenHeight, as drawn by Render; the adapter does any
// rotation to the hardware's native orientation. Sleep is called after
// every update.
type Panel interface {
	// Full shows frame with a full refresh.
	Full(frame image.Image) error
	// Partial refreshes only rect of the display. frame is the whole
	// portrait ScreenWidth x ScreenHeight frame, as for Full, and rect is
	// in portrait frame coordinates; the adapter crops and rotates it. The
	// panel keeps showing everything outside rect.
	Partial(frame image.Image, rect image.Rectangle) error
	// Sleep puts the panel into its low-power state.
	Sleep() error
	// Close releases the panel's resources.
	Close() error
}
