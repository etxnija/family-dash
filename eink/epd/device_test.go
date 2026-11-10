package epd

import (
	"errors"
	"image"
	"reflect"
	"testing"
)

var nativeBounds = image.Rect(0, 0, nativeWidth, nativeHeight)

// fakeDevice records the calls made to it and fails the ones in fail.
type fakeDevice struct {
	calls []string
	fail  map[string]error
	shown image.Image
}

func (d *fakeDevice) call(name string) error {
	d.calls = append(d.calls, name)
	return d.fail[name]
}

func (d *fakeDevice) GPIOInit() error { return d.call("GPIOInit") }
func (d *fakeDevice) Init() error     { return d.call("Init") }
func (d *fakeDevice) Sleep() error    { return d.call("Sleep") }
func (d *fakeDevice) Close() error    { return d.call("Close") }

func (d *fakeDevice) DisplayImage(img image.Image) error {
	d.shown = img
	return d.call("DisplayImage")
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
			if !reflect.DeepEqual(dev.calls, tt.wantCalls) {
				t.Errorf("calls = %v, want %v", dev.calls, tt.wantCalls)
			}
			if tt.wantOK != (err == nil) {
				t.Fatalf("Full = %v, want ok %v", err, tt.wantOK)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want it to wrap %v", err, tt.wantErr)
			}
			if tt.wantOK && dev.shown.Bounds() != nativeBounds {
				t.Errorf("shown bounds = %v, want %v", dev.shown.Bounds(), nativeBounds)
			}
		})
	}
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
