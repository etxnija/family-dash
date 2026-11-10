package eink

import (
	"image"
	"strings"
	"testing"

	"golang.org/x/image/font"
)

func fullView() View {
	return View{
		DateLabel: "Onsdag 16 september",
		WeekLabel: "v.38",
		Owners:    "Kaj · Lillian",
		Clock:     "07:45",
		Weather: WeatherView{Temperature: "11°", Lines: []string{
			"SMHI", "Lägst idag: -5° · Högst idag: 12°", "Sol", "torrt idag",
		}},
		Lunch:  []string{"Köttbullar med potatismos", "Vegetarisk gröt"},
		Events: []EventRow{{Time: "07:15", Owner: "Kaj", Summary: "Tandläkare", Location: "Folktandvården"}},
		Homework: []HomeworkGroup{{Owner: "Lillian", Initial: "L", Items: []HomeworkRow{
			{Summary: "Läxa engelska", DayLabel: "idag", IsProv: true},
		}}},
		Upcoming: []UpcomingRow{{Weekday: "Tors", Time: "17:00", Owner: "Kaj", Summary: "Simning", Location: "Hässelbybadet"}},
	}
}

func emptyView(t *testing.T) View {
	t.Helper()
	v, err := NewView(nil, nil, nil, nil, nil, nil, testNow)
	if err != nil {
		t.Fatalf("NewView: %v", err)
	}
	return v
}

// stressView has 20 rows in every section and 300-rune Swedish summaries.
func stressView() View {
	long := strings.Repeat("Läxa åäöÅÄÖ ", 25)
	v := View{
		DateLabel: "Onsdag 16 september " + long,
		WeekLabel: "v.38",
		Owners:    "Kaj · Lillian · " + long,
		Clock:     "88:88",
		Weather:   WeatherView{Temperature: "-88°" + long, Lines: []string{long, long + " · " + long, long, long, long}},
	}
	for i := range 20 {
		v.Lunch = append(v.Lunch, long)
		v.Events = append(v.Events, EventRow{Time: "heldag", Owner: "Lillian", Summary: long, Location: long})
		v.Upcoming = append(v.Upcoming, UpcomingRow{Weekday: "Tors", Time: "heldag", Owner: "Lillian", Summary: long, Location: long})
		owner := "Kaj"
		if i%2 == 1 {
			owner = "Lillian"
		}
		v.Homework = append(v.Homework, HomeworkGroup{Owner: owner, Initial: owner[:1], Items: []HomeworkRow{
			{Summary: long, DayLabel: "imorgon", IsProv: i%3 == 0},
		}})
	}
	return v
}

func TestStressView_TextLength(t *testing.T) {
	v := stressView()
	tests := []struct {
		name string
		text string
	}{
		{name: "lunch", text: v.Lunch[0]},
		{name: "event summary", text: v.Events[0].Summary},
		{name: "homework summary", text: v.Homework[0].Items[0].Summary},
		{name: "upcoming summary", text: v.Upcoming[0].Summary},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if n := len([]rune(tt.text)); n != 300 {
				t.Errorf("stress %s has %d runes, want 300", tt.name, n)
			}
		})
	}
}

func TestRender_BoundsAndBinaryPixels(t *testing.T) {
	tests := []struct {
		name string
		view View
	}{
		{name: "fully populated", view: fullView()},
		{name: "empty", view: emptyView(t)},
		{name: "stress", view: stressView()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := Render(tt.view)
			if got, want := img.Bounds(), image.Rect(0, 0, 800, 480); got != want {
				t.Fatalf("Bounds = %v, want %v", got, want)
			}
			for i, p := range img.Pix {
				if p != 0 && p != 255 {
					t.Fatalf("pixel %d = %d, want 0 or 255", i, p)
				}
			}
		})
	}
}

// rowRect returns the area of the body row with baseline y in column.
func rowRect(column image.Rectangle, baseline int) image.Rectangle {
	m := bodyFace.Metrics()
	return image.Rect(column.Min.X, baseline-m.Ascent.Ceil(), column.Max.X, baseline+m.Descent.Ceil())
}

func TestRender_EmptyStatesDrawn(t *testing.T) {
	img := Render(emptyView(t))

	tests := []struct {
		name string
		area image.Rectangle
	}{
		{name: "lunch", area: rowRect(leftColumnRect, firstRowBaselineY(lunchTop))},
		{name: "idag", area: rowRect(rightColumnRect, firstRowBaselineY(todayTop))},
		{name: "läxor", area: rowRect(rightColumnRect, firstRowBaselineY(homeworkTop))},
		{name: "kommande", area: rowRect(rightColumnRect, firstRowBaselineY(upcomingTop))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if countBlack(img, tt.area) == 0 {
				t.Errorf("no black pixels in %s empty-state area %v", tt.name, tt.area)
			}
		})
	}
}

func TestRender_StressStaysContained(t *testing.T) {
	img := Render(stressView())

	inner := image.Rect(borderThickness, borderThickness, ScreenWidth-borderThickness, ScreenHeight-borderThickness)
	tests := []struct {
		name string
		area image.Rectangle
	}{
		{name: "gutter below header rule", area: image.Rect(leftColumnX1, bodyTop, rightColumnX0, inner.Max.Y)},
		{name: "top margin", area: image.Rect(inner.Min.X, inner.Min.Y, inner.Max.X, outerMargin)},
		{name: "bottom margin", area: image.Rect(inner.Min.X, ScreenHeight-outerMargin, inner.Max.X, inner.Max.Y)},
		{name: "left margin", area: image.Rect(inner.Min.X, inner.Min.Y, outerMargin, inner.Max.Y)},
		{name: "right margin", area: image.Rect(ScreenWidth-outerMargin, inner.Min.Y, inner.Max.X, inner.Max.Y)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if n := countBlack(img, tt.area); n != 0 {
				t.Errorf("%d black pixels in %v, want none", n, tt.area)
			}
		})
	}
}

func TestVisibleRows_StressLimits(t *testing.T) {
	v := stressView()
	// Every stress section has 20 rows; a section with room for n shows
	// n-1 of them plus "+N till" for the other 21-n.
	tests := []struct {
		name     string
		rows     []string
		width    int
		limit    int
		wantLast string
	}{
		{name: "lunch", rows: lunchRows(v.Lunch, leftColumnRect.Dx()), width: leftColumnRect.Dx(), limit: maxLunchRows, wantLast: "+17 till"},
		{name: "events", rows: eventRows(v.Events, rightColumnRect.Dx()), width: rightColumnRect.Dx(), limit: maxEventRows, wantLast: "+16 till"},
		{name: "homework, flattened across groups", rows: homeworkRows(v.Homework, rightColumnRect.Dx()), width: rightColumnRect.Dx(), limit: maxHomeworkRows, wantLast: "+17 till"},
		{name: "upcoming", rows: upcomingRows(v.Upcoming, rightColumnRect.Dx()), width: rightColumnRect.Dx(), limit: maxUpcomingRows, wantLast: "+16 till"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.rows) != tt.limit {
				t.Fatalf("got %d rows, want %d", len(tt.rows), tt.limit)
			}
			if last := tt.rows[len(tt.rows)-1]; last != tt.wantLast {
				t.Errorf("last row = %q, want %q", last, tt.wantLast)
			}
			for _, row := range tt.rows {
				if w := font.MeasureString(bodyFace, row).Ceil(); w > tt.width {
					t.Errorf("row %q is %dpx wide, want ≤ %d", row, w, tt.width)
				}
			}
		})
	}
}

func TestVisibleRows_KeepSuffix(t *testing.T) {
	long := strings.Repeat("Läxa ", 100)
	tests := []struct {
		name       string
		row        string
		wantSuffix string
	}{
		{
			name:       "homework keeps prov marker and day label",
			row:        homeworkRows([]HomeworkGroup{{Initial: "K", Items: []HomeworkRow{{Summary: long, DayLabel: "fredag", IsProv: true}}}}, rightColumnRect.Dx())[0],
			wantSuffix: "… (Prov) – fredag",
		},
		{
			name:       "upcoming keeps time",
			row:        upcomingRows([]UpcomingRow{{Weekday: "Mån", Time: "18:30", Owner: "Kaj", Summary: long}}, rightColumnRect.Dx())[0],
			wantSuffix: "…, 18:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.HasSuffix(tt.row, tt.wantSuffix) {
				t.Errorf("row %q does not end in %q", tt.row, tt.wantSuffix)
			}
		})
	}
}

func TestLimitRows(t *testing.T) {
	tests := []struct {
		name string
		rows []string
		max  int
		want []string
	}{
		{name: "fewer than max", rows: []string{"a", "b"}, max: 3, want: []string{"a", "b"}},
		{name: "exactly max", rows: []string{"a", "b", "c"}, max: 3, want: []string{"a", "b", "c"}},
		{name: "one over max", rows: []string{"a", "b", "c", "d"}, max: 3, want: []string{"a", "b", "+2 till"}},
		{name: "none", rows: nil, max: 3, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := limitRows(tt.rows, tt.max)
			if strings.Join(got, "|") != strings.Join(tt.want, "|") || len(got) != len(tt.want) {
				t.Errorf("limitRows = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFitText(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxWidth int
		want     string // "" means: must end in "…" and fit maxWidth
	}{
		{name: "short text unchanged", s: "Simning", maxWidth: 200, want: "Simning"},
		{name: "empty text unchanged", s: "", maxWidth: 200, want: ""},
		{name: "long text cut", s: strings.Repeat("Fotbollsträning på Åvallen ", 10), maxWidth: 200},
		{name: "long Swedish text cut", s: strings.Repeat("åäöÅÄÖ", 50), maxWidth: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fitText(bodyFace, tt.s, tt.maxWidth)
			if tt.want != "" || tt.s == "" {
				if got != tt.want {
					t.Errorf("fitText = %q, want %q", got, tt.want)
				}
				return
			}
			if !strings.HasSuffix(got, ellipsis) {
				t.Errorf("fitText = %q, want it to end in %q", got, ellipsis)
			}
			if w := font.MeasureString(bodyFace, got).Ceil(); w > tt.maxWidth {
				t.Errorf("fitText = %q is %dpx wide, want ≤ %d", got, w, tt.maxWidth)
			}
		})
	}
}

func TestClockRect(t *testing.T) {
	r := ClockRect()
	inner := image.Rect(borderThickness, borderThickness, ScreenWidth-borderThickness, ScreenHeight-borderThickness)
	m := clockFace.Metrics()

	tests := []struct {
		name string
		ok   bool
	}{
		{name: "X edges are multiples of 8", ok: r.Min.X%8 == 0 && r.Max.X%8 == 0},
		{name: "inside the screen", ok: r.In(screenRect)},
		{name: "inside the border", ok: r.In(inner)},
		{name: "inside the left column", ok: r.In(leftColumnRect)},
		{name: "wide enough for 88:88", ok: font.MeasureString(clockFace, "88:88").Ceil() <= r.Dx()},
		{name: "wide enough for 00:00", ok: font.MeasureString(clockFace, "00:00").Ceil() <= r.Dx()},
		{name: "tall enough for ascent plus descent", ok: m.Ascent.Ceil()+m.Descent.Ceil() <= r.Dy()},
		{name: "above the temperature", ok: r.Max.Y <= temperatureBaselineY-temperatureFace.Metrics().Ascent.Ceil()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.ok {
				t.Errorf("ClockRect %v: %s does not hold", r, tt.name)
			}
		})
	}
}

func TestRender_ClockDrawnInsideClockRect(t *testing.T) {
	tests := []struct {
		name  string
		clock string
	}{
		{name: "typical", clock: "12:34"},
		{name: "widest digits", clock: "88:88"},
		{name: "midnight", clock: "00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := emptyView(t)
			v.Clock = tt.clock
			if countBlack(Render(v), ClockRect()) == 0 {
				t.Errorf("no black pixels inside ClockRect %v for %q", ClockRect(), tt.clock)
			}
		})
	}
}

func TestLayout_SectionsFitAboveBottomMargin(t *testing.T) {
	descent := bodyFace.Metrics().Descent.Ceil()
	tests := []struct {
		name         string
		lastBaseline int
	}{
		{name: "lunch", lastBaseline: firstRowBaselineY(lunchTop) + (maxLunchRows-1)*rowHeight},
		{name: "upcoming", lastBaseline: firstRowBaselineY(upcomingTop) + (maxUpcomingRows-1)*rowHeight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if bottom := tt.lastBaseline + descent; bottom > contentBottom {
				t.Errorf("last row reaches y=%d, want ≤ %d", bottom, contentBottom)
			}
		})
	}
}

func TestFaces_HaveGlyphs(t *testing.T) {
	faces := []struct {
		name string
		face font.Face
	}{
		{name: "clock", face: clockFace},
		{name: "date", face: dateFace},
		{name: "heading", face: headingFace},
		{name: "body", face: bodyFace},
		{name: "temperature", face: temperatureFace},
	}

	for _, f := range faces {
		t.Run(f.name, func(t *testing.T) {
			for _, r := range "åäöÅÄÖ…·°–—" {
				if _, ok := f.face.GlyphAdvance(r); !ok {
					t.Errorf("%s face has no glyph for %q", f.name, r)
				}
			}
		})
	}
}

// countBlack returns the number of black pixels of img inside r.
func countBlack(img *image.Gray, r image.Rectangle) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if img.GrayAt(x, y).Y == 0 {
				n++
			}
		}
	}
	return n
}
