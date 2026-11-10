package epd_test

import (
	"image"
	"image/color"
	"testing"
	"time"

	"family-dashboard/eink"
	"family-dashboard/eink/epd"
)

// nativePoint is the spec's cable mapping, written out independently of
// Rotate so the tests check it rather than restate it.
func nativePoint(p image.Point, cable epd.Cable) image.Point {
	if cable == epd.CableLeft {
		return image.Pt(p.Y, 479-p.X)
	}
	return image.Pt(799-p.Y, p.X)
}

// nativeRect maps a portrait rectangle to the native one it covers.
func nativeRect(r image.Rectangle, cable epd.Cable) image.Rectangle {
	a := nativePoint(r.Min, cable)
	b := nativePoint(r.Max.Sub(image.Pt(1, 1)), cable)
	n := image.Rectangle{Min: a, Max: b}.Canon()
	n.Max = n.Max.Add(image.Pt(1, 1))
	return n
}

func whiteFrame(r image.Rectangle) *image.Gray {
	img := image.NewGray(r)
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	return img
}

func countBlack(img *image.Gray) int {
	n := 0
	for _, p := range img.Pix {
		if p == 0 {
			n++
		}
	}
	return n
}

func checkBinary(t *testing.T, img *image.Gray) {
	t.Helper()
	if got, want := img.Bounds(), image.Rect(0, 0, 800, 480); got != want {
		t.Fatalf("bounds = %v, want %v", got, want)
	}
	for i, p := range img.Pix {
		if p != 0 && p != 255 {
			t.Fatalf("pixel %d = %d, want 0 or 255", i, p)
		}
	}
}

func TestRotate_Mapping(t *testing.T) {
	type move struct{ portrait, native image.Point }
	tests := []struct {
		name  string
		cable epd.Cable
		moves []move
	}{
		{
			name:  "cable left",
			cable: epd.CableLeft,
			moves: []move{
				{image.Pt(0, 0), image.Pt(0, 479)},
				{image.Pt(479, 0), image.Pt(0, 0)},
				{image.Pt(0, 799), image.Pt(799, 479)},
				{image.Pt(479, 799), image.Pt(799, 0)},
				{image.Pt(10, 3), image.Pt(3, 469)},
				{image.Pt(479, 400), image.Pt(400, 0)},
				{image.Pt(200, 799), image.Pt(799, 279)},
			},
		},
		{
			name:  "cable right",
			cable: epd.CableRight,
			moves: []move{
				{image.Pt(0, 0), image.Pt(799, 0)},
				{image.Pt(479, 0), image.Pt(799, 479)},
				{image.Pt(0, 799), image.Pt(0, 0)},
				{image.Pt(479, 799), image.Pt(0, 479)},
				{image.Pt(10, 3), image.Pt(796, 10)},
				{image.Pt(479, 400), image.Pt(399, 479)},
				{image.Pt(200, 799), image.Pt(0, 200)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := whiteFrame(image.Rect(0, 0, eink.ScreenWidth, eink.ScreenHeight))
			for _, m := range tt.moves {
				frame.SetGray(m.portrait.X, m.portrait.Y, color.Gray{Y: 0})
			}

			out, err := epd.Rotate(frame, tt.cable)
			if err != nil {
				t.Fatalf("Rotate: %v", err)
			}
			checkBinary(t, out)
			for _, m := range tt.moves {
				if got := out.GrayAt(m.native.X, m.native.Y).Y; got != 0 {
					t.Errorf("native %v (from portrait %v) = %d, want 0", m.native, m.portrait, got)
				}
				if got := nativePoint(m.portrait, tt.cable); got != m.native {
					t.Errorf("nativePoint(%v) = %v, want %v", m.portrait, got, m.native)
				}
			}
			if got, want := countBlack(out), len(tt.moves); got != want {
				t.Errorf("black pixels = %d, want %d", got, want)
			}
		})
	}
}

func TestRotate_InputVariants(t *testing.T) {
	// A non-zero origin and a non-Gray type must both read the right
	// pixels and still come out strictly 1-bit.
	offset := image.Rect(5, 7, 5+eink.ScreenWidth, 7+eink.ScreenHeight)
	shifted := whiteFrame(offset)
	shifted.SetGray(offset.Min.X+10, offset.Min.Y+3, color.Gray{Y: 0})

	rgba := image.NewRGBA(image.Rect(0, 0, eink.ScreenWidth, eink.ScreenHeight))
	for y := range eink.ScreenHeight {
		for x := range eink.ScreenWidth {
			rgba.Set(x, y, color.Gray{Y: 200})
		}
	}
	rgba.Set(10, 3, color.Gray{Y: 100})

	tests := []struct {
		name  string
		frame image.Image
	}{
		{name: "non-zero origin", frame: shifted},
		{name: "grey RGBA", frame: rgba},
	}
	for _, tt := range tests {
		for _, cable := range []epd.Cable{epd.CableLeft, epd.CableRight} {
			out, err := epd.Rotate(tt.frame, cable)
			if err != nil {
				t.Fatalf("%s, cable %d: Rotate: %v", tt.name, cable, err)
			}
			checkBinary(t, out)
			p := nativePoint(image.Pt(10, 3), cable)
			if got := out.GrayAt(p.X, p.Y).Y; got != 0 {
				t.Errorf("%s, cable %d: native %v = %d, want 0", tt.name, cable, p, got)
			}
			if got := countBlack(out); got != 1 {
				t.Errorf("%s, cable %d: black pixels = %d, want 1", tt.name, cable, got)
			}
		}
	}
}

func TestRotate_Errors(t *testing.T) {
	good := image.NewGray(image.Rect(0, 0, eink.ScreenWidth, eink.ScreenHeight))
	tests := []struct {
		name  string
		frame image.Image
		cable epd.Cable
	}{
		{name: "landscape frame", frame: image.NewGray(image.Rect(0, 0, 800, 480)), cable: epd.CableLeft},
		{name: "one row short", frame: image.NewGray(image.Rect(0, 0, 480, 799)), cable: epd.CableLeft},
		{name: "empty frame", frame: image.NewGray(image.Rect(0, 0, 0, 0)), cable: epd.CableRight},
		{name: "unknown cable", frame: good, cable: epd.Cable(7)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := epd.Rotate(tt.frame, tt.cable)
			if err == nil {
				t.Fatalf("Rotate(%v, %d) = nil error, want error", tt.frame.Bounds(), tt.cable)
			}
			if out != nil {
				t.Errorf("Rotate(%v, %d) image = non-nil, want nil", tt.frame.Bounds(), tt.cable)
			}
		})
	}
}

func TestParseCable(t *testing.T) {
	tests := []struct {
		in      string
		want    epd.Cable
		wantErr bool
	}{
		{in: "left", want: epd.CableLeft},
		{in: "right", want: epd.CableRight},
		{in: "", wantErr: true},
		{in: "Left", wantErr: true},
		{in: "up", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := epd.ParseCable(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseCable(%q) = %d, nil error, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCable(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseCable(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestRotate_ClockRect(t *testing.T) {
	now := time.Date(2026, 9, 16, 5, 45, 0, 0, time.UTC)
	view, err := eink.NewView(nil, nil, nil, nil, nil, nil, now)
	if err != nil {
		t.Fatalf("NewView: %v", err)
	}
	frame := eink.Render(view)
	clock := eink.ClockRect()

	tests := []struct {
		name           string
		cable          epd.Cable
		wantX0, wantX1 int
	}{
		{name: "cable left", cable: epd.CableLeft, wantX0: 88, wantX1: 176},
		{name: "cable right", cable: epd.CableRight, wantX0: 800 - 176, wantX1: 800 - 88},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := epd.Rotate(frame, tt.cable)
			if err != nil {
				t.Fatalf("Rotate: %v", err)
			}
			native := nativeRect(clock, tt.cable)
			if native.Min.X != tt.wantX0 || native.Max.X != tt.wantX1 {
				t.Fatalf("native clock X = [%d, %d), want [%d, %d)", native.Min.X, native.Max.X, tt.wantX0, tt.wantX1)
			}
			if native.Min.X%8 != 0 || native.Max.X%8 != 0 {
				t.Errorf("native clock X edges [%d, %d) are not multiples of 8", native.Min.X, native.Max.X)
			}

			black := 0
			for y := clock.Min.Y; y < clock.Max.Y; y++ {
				for x := clock.Min.X; x < clock.Max.X; x++ {
					p := nativePoint(image.Pt(x, y), tt.cable)
					if !p.In(native) {
						t.Fatalf("portrait %v maps to %v outside native clock %v", image.Pt(x, y), p, native)
					}
					want := frame.GrayAt(x, y).Y
					if got := out.GrayAt(p.X, p.Y).Y; got != want {
						t.Fatalf("native %v = %d, want portrait %v = %d", p, got, image.Pt(x, y), want)
					}
					if want == 0 {
						black++
					}
				}
			}
			if black == 0 {
				t.Error("native clock rect has no black pixels, want clock glyphs")
			}
		})
	}
}
