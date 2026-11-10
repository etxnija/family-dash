package epd

import (
	"errors"
	"image"
	"reflect"
	"strings"
	"testing"

	"family-dashboard/eink"
)

var nativeBounds = image.Rect(0, 0, nativeWidth, nativeHeight)

// fakeDevice records the calls made to it and fails the ones in fail. If
// failPart is set, only the DisplayPart call with index failPartAt (counted
// from 0) fails with it.
type fakeDevice struct {
	calls      []string
	fail       map[string]error
	failPart   error
	failPartAt int
	shown      image.Image
	parts      []partCall
}

// partCall is one DisplayPart call.
type partCall struct {
	img  image.Image
	rect image.Rectangle
}

func (d *fakeDevice) call(name string) error {
	d.calls = append(d.calls, name)
	return d.fail[name]
}

func (d *fakeDevice) GPIOInit() error { return d.call("GPIOInit") }
func (d *fakeDevice) Init() error     { return d.call("Init") }
func (d *fakeDevice) InitPart() error { return d.call("InitPart") }
func (d *fakeDevice) Sleep() error    { return d.call("Sleep") }
func (d *fakeDevice) Close() error    { return d.call("Close") }

func (d *fakeDevice) DisplayImage(img image.Image) error {
	d.shown = img
	return d.call("DisplayImage")
}

func (d *fakeDevice) DisplayPart(img image.Image, rect image.Rectangle) error {
	i := len(d.parts)
	d.parts = append(d.parts, partCall{img, rect})
	err := d.call("DisplayPart")
	if d.failPart != nil && i == d.failPartAt {
		return d.failPart
	}
	return err
}

func TestOpenDevice(t *testing.T) {
	errGPIO := errors.New("gpio")
	errClose := errors.New("close")
	tests := []struct {
		name      string
		fail      map[string]error
		wantCalls []string
		wantErrs  []error
	}{
		{"ok", nil, []string{"GPIOInit"}, nil},
		{"gpio fails", map[string]error{"GPIOInit": errGPIO}, []string{"GPIOInit", "Close"}, []error{errGPIO}},
		{"gpio and close fail", map[string]error{"GPIOInit": errGPIO, "Close": errClose}, []string{"GPIOInit", "Close"}, []error{errGPIO, errClose}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := &fakeDevice{fail: tt.fail}
			p, err := openDevice(dev, nativeBounds, CableLeft)
			if !reflect.DeepEqual(dev.calls, tt.wantCalls) {
				t.Errorf("calls = %v, want %v", dev.calls, tt.wantCalls)
			}
			if tt.wantErrs == nil {
				if err != nil || p == nil {
					t.Fatalf("openDevice = %v, %v, want panel, nil", p, err)
				}
				if !p.prime {
					t.Errorf("prime = false, want true (primeBeforePartial)")
				}
				return
			}
			if p != nil {
				t.Errorf("panel = %v, want nil", p)
			}
			for _, want := range tt.wantErrs {
				if !errors.Is(err, want) {
					t.Errorf("err = %v, want it to wrap %v", err, want)
				}
			}
		})
	}
}

// checkUpdate checks the device calls and the error of a Full or Partial
// call, stopping the test when the error does not match wantOK.
func checkUpdate(t *testing.T, method string, dev *fakeDevice, err error, wantCalls []string, wantOK bool, wantErr error) {
	t.Helper()
	if !reflect.DeepEqual(dev.calls, wantCalls) {
		t.Errorf("calls = %v, want %v", dev.calls, wantCalls)
	}
	if wantOK != (err == nil) {
		t.Fatalf("%s = %v, want ok %v", method, err, wantOK)
	}
	if wantErr != nil && !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want it to wrap %v", err, wantErr)
	}
}

func TestDevicePanel_Full(t *testing.T) {
	errInit := errors.New("init")
	errDisplay := errors.New("display")
	portrait := image.NewGray(image.Rect(0, 0, nativeHeight, nativeWidth))
	tests := []struct {
		name      string
		frame     image.Image
		bounds    image.Rectangle
		fail      map[string]error
		wantCalls []string
		wantOK    bool
		wantErr   error
	}{
		{"ok", portrait, nativeBounds, nil, []string{"Init", "DisplayImage"}, true, nil},
		{"rotate fails", image.NewGray(image.Rect(0, 0, 10, 10)), nativeBounds, nil, nil, false, nil},
		{"bounds mismatch", portrait, image.Rect(0, 0, 640, 384), nil, nil, false, nil},
		{"init fails", portrait, nativeBounds, map[string]error{"Init": errInit}, []string{"Init"}, false, errInit},
		{"display fails", portrait, nativeBounds, map[string]error{"DisplayImage": errDisplay}, []string{"Init", "DisplayImage"}, false, errDisplay},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := &fakeDevice{fail: tt.fail}
			p := &devicePanel{dev: dev, bounds: tt.bounds, cable: CableLeft}
			err := p.Full(tt.frame)
			checkUpdate(t, "Full", dev, err, tt.wantCalls, tt.wantOK, tt.wantErr)
			if tt.wantOK && dev.shown.Bounds() != nativeBounds {
				t.Errorf("shown bounds = %v, want %v", dev.shown.Bounds(), nativeBounds)
			}
			if (p.shown != nil) != tt.wantOK {
				t.Errorf("remembered frame set = %v, want %v", p.shown != nil, tt.wantOK)
			}
		})
	}
}

// testFrame returns a white portrait frame with a black block of width w
// at the left of eink.ClockRect(), so frames with different w differ
// inside the clock rect only.
func testFrame(w int) *image.Gray {
	f := image.NewGray(image.Rect(0, 0, eink.ScreenWidth, eink.ScreenHeight))
	for i := range f.Pix {
		f.Pix[i] = 255
	}
	clock := eink.ClockRect()
	for y := clock.Min.Y; y < clock.Max.Y; y++ {
		for x := clock.Min.X; x < clock.Min.X+w; x++ {
			f.Pix[f.PixOffset(x, y)] = 0
		}
	}
	return f
}

// clockRegion is the native image of frame's clock rect, as DisplayPart
// should receive it.
func clockRegion(t *testing.T, frame image.Image, cable Cable) *image.Gray {
	t.Helper()
	small, _, err := RotateRect(frame, eink.ClockRect(), cable)
	if err != nil {
		t.Fatalf("RotateRect: %v", err)
	}
	return small
}

// samePixels reports whether a and b have the same bounds and pixels.
func samePixels(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}
	return true
}

// checkParts checks the DisplayPart calls: one per entry of want, each with
// want's pixels and rect.
func checkParts(t *testing.T, dev *fakeDevice, rect image.Rectangle, want ...*image.Gray) {
	t.Helper()
	if len(dev.parts) != len(want) {
		t.Fatalf("DisplayPart calls = %d, want %d", len(dev.parts), len(want))
	}
	for i, w := range want {
		if dev.parts[i].rect != rect {
			t.Errorf("DisplayPart %d rect = %v, want %v", i, dev.parts[i].rect, rect)
		}
		if !samePixels(dev.parts[i].img, w) {
			t.Errorf("DisplayPart %d image does not match the expected frame region", i)
		}
	}
}

// newShownPanel returns a priming panel on a fresh fakeDevice that has shown
// frame with a successful Full; the device's records are then cleared.
func newShownPanel(t *testing.T, cable Cable, frame image.Image) (*devicePanel, *fakeDevice) {
	t.Helper()
	dev := &fakeDevice{}
	p := &devicePanel{dev: dev, bounds: nativeBounds, cable: cable, prime: true}
	if err := p.Full(frame); err != nil {
		t.Fatalf("Full: %v", err)
	}
	dev.calls, dev.parts = nil, nil
	return p, dev
}

func TestDevicePanel_FullThenPartial(t *testing.T) {
	// ClockRect is X 16-248, Y 88-176 in portrait.
	tests := []struct {
		cable    Cable
		wantRect image.Rectangle
	}{
		{CableLeft, image.Rect(88, 232, 176, 464)},
		{CableRight, image.Rect(624, 16, 712, 248)},
	}
	for _, tt := range tests {
		t.Run(map[Cable]string{CableLeft: "left", CableRight: "right"}[tt.cable], func(t *testing.T) {
			first, second := testFrame(40), testFrame(120)
			dev := &fakeDevice{}
			p := &devicePanel{dev: dev, bounds: nativeBounds, cable: tt.cable, prime: true}
			if err := p.Full(first); err != nil {
				t.Fatalf("Full: %v", err)
			}
			if err := p.Partial(second, eink.ClockRect()); err != nil {
				t.Fatalf("Partial: %v", err)
			}
			wantCalls := []string{"Init", "DisplayImage", "InitPart", "DisplayPart", "DisplayPart"}
			if !reflect.DeepEqual(dev.calls, wantCalls) {
				t.Errorf("calls = %v, want %v", dev.calls, wantCalls)
			}
			checkParts(t, dev, tt.wantRect, clockRegion(t, first, tt.cable), clockRegion(t, second, tt.cable))
		})
	}
}

func TestDevicePanel_PartialPrimesWithLastPartial(t *testing.T) {
	a, b, c := testFrame(40), testFrame(120), testFrame(200)
	p, dev := newShownPanel(t, CableLeft, a)
	if err := p.Partial(b, eink.ClockRect()); err != nil {
		t.Fatalf("Partial b: %v", err)
	}
	dev.calls, dev.parts = nil, nil
	if err := p.Partial(c, eink.ClockRect()); err != nil {
		t.Fatalf("Partial c: %v", err)
	}
	checkParts(t, dev, image.Rect(88, 232, 176, 464), clockRegion(t, b, CableLeft), clockRegion(t, c, CableLeft))
}

func TestDevicePanel_PartialBeforeFull(t *testing.T) {
	tests := []struct {
		name      string
		prime     bool
		wantCalls []string
		wantOK    bool
	}{
		{"priming", true, nil, false},
		{"not priming", false, []string{"InitPart", "DisplayPart"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := &fakeDevice{}
			p := &devicePanel{dev: dev, bounds: nativeBounds, cable: CableLeft, prime: tt.prime}
			err := p.Partial(testFrame(40), eink.ClockRect())
			checkUpdate(t, "Partial", dev, err, tt.wantCalls, tt.wantOK, nil)
			if !tt.wantOK && !strings.Contains(err.Error(), "before any frame was shown") {
				t.Errorf("err = %v, want it to say no frame was shown", err)
			}
		})
	}
}

func TestDevicePanel_PartialNoDeviceCalls(t *testing.T) {
	tests := []struct {
		name   string
		rect   image.Rectangle
		bounds image.Rectangle
	}{
		{"rotate rect fails", image.Rect(0, 90, 10, 170), nativeBounds},
		{"outside driver bounds", eink.ClockRect(), image.Rect(0, 0, 80, 80)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := &fakeDevice{}
			p := &devicePanel{dev: dev, bounds: tt.bounds, cable: CableLeft, prime: true, shown: copyFrame(testFrame(40))}
			err := p.Partial(testFrame(120), tt.rect)
			checkUpdate(t, "Partial", dev, err, nil, false, nil)
		})
	}
}

func TestDevicePanel_FailedUpdateKeepsShown(t *testing.T) {
	errDev := errors.New("device")
	tests := []struct {
		name         string
		partial      bool
		fail         map[string]error
		failPartAt   int // index of the failing DisplayPart, or -1
		wantCalls    []string
		wantPrevious bool // the error names the previous image
	}{
		{"full init", false, map[string]error{"Init": errDev}, -1, []string{"Init"}, false},
		{"full display", false, map[string]error{"DisplayImage": errDev}, -1, []string{"Init", "DisplayImage"}, false},
		{"init part", true, map[string]error{"InitPart": errDev}, -1, []string{"InitPart"}, false},
		{"first display part", true, nil, 0, []string{"InitPart", "DisplayPart"}, true},
		{"second display part", true, nil, 1, []string{"InitPart", "DisplayPart", "DisplayPart"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			older, failed, next := testFrame(40), testFrame(120), testFrame(200)
			p, dev := newShownPanel(t, CableLeft, older)
			dev.fail = tt.fail
			if tt.failPartAt >= 0 {
				dev.failPart, dev.failPartAt = errDev, tt.failPartAt
			}
			var err error
			if tt.partial {
				err = p.Partial(failed, eink.ClockRect())
			} else {
				err = p.Full(failed)
			}
			checkUpdate(t, "update", dev, err, tt.wantCalls, false, errDev)
			if got := strings.Contains(err.Error(), "previous image"); got != tt.wantPrevious {
				t.Errorf("err = %q, mentions previous image %v, want %v", err, got, tt.wantPrevious)
			}

			*dev = fakeDevice{}
			if err := p.Partial(next, eink.ClockRect()); err != nil {
				t.Fatalf("Partial after failure: %v", err)
			}
			checkParts(t, dev, image.Rect(88, 232, 176, 464), clockRegion(t, older, CableLeft), clockRegion(t, next, CableLeft))
		})
	}
}

func TestDevicePanel_ShownIsACopy(t *testing.T) {
	frame := testFrame(40)
	want := clockRegion(t, frame, CableLeft)
	p, dev := newShownPanel(t, CableLeft, frame)
	for i := range frame.Pix {
		frame.Pix[i] = 0
	}
	next := testFrame(120)
	if err := p.Partial(next, eink.ClockRect()); err != nil {
		t.Fatalf("Partial: %v", err)
	}
	checkParts(t, dev, image.Rect(88, 232, 176, 464), want, clockRegion(t, next, CableLeft))
}

func TestDevicePanel_PartialWithoutPriming(t *testing.T) {
	a, b, c := testFrame(40), testFrame(120), testFrame(200)
	p, dev := newShownPanel(t, CableLeft, a)
	p.prime = false
	if err := p.Partial(b, eink.ClockRect()); err != nil {
		t.Fatalf("Partial: %v", err)
	}
	if want := []string{"InitPart", "DisplayPart"}; !reflect.DeepEqual(dev.calls, want) {
		t.Errorf("calls = %v, want %v", dev.calls, want)
	}
	rect := image.Rect(88, 232, 176, 464)
	checkParts(t, dev, rect, clockRegion(t, b, CableLeft))

	// The shown frame was still updated, so priming again starts from b.
	p.prime = true
	dev.calls, dev.parts = nil, nil
	if err := p.Partial(c, eink.ClockRect()); err != nil {
		t.Fatalf("Partial: %v", err)
	}
	checkParts(t, dev, rect, clockRegion(t, b, CableLeft), clockRegion(t, c, CableLeft))
}

func TestDevicePanel_SleepClose(t *testing.T) {
	errDev := errors.New("device")
	tests := []struct {
		name string
		call func(*devicePanel) error
	}{
		{"Sleep", (*devicePanel).Sleep},
		{"Close", (*devicePanel).Close},
	}
	for _, tt := range tests {
		for _, fail := range []bool{false, true} {
			dev := &fakeDevice{}
			if fail {
				dev.fail = map[string]error{tt.name: errDev}
			}
			err := tt.call(&devicePanel{dev: dev, bounds: nativeBounds})
			if !reflect.DeepEqual(dev.calls, []string{tt.name}) {
				t.Errorf("%s: calls = %v, want [%s]", tt.name, dev.calls, tt.name)
			}
			if fail != errors.Is(err, errDev) {
				t.Errorf("%s (fail %v): err = %v", tt.name, fail, err)
			}
		}
	}
}
