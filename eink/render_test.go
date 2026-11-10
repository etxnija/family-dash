package eink

import (
	"fmt"
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

// stressView has 20 rows in every section, one-line lunch dishes and
// 300-rune Swedish summaries.
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
		v.Lunch = append(v.Lunch, "Köttbullar med potatismos och lingon")
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
			if got, want := img.Bounds(), image.Rect(0, 0, 480, 800); got != want {
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
		{name: "lunch", area: rowRect(contentRect, firstRowBaselineY(lunchTop))},
		{name: "idag", area: rowRect(contentRect, firstRowBaselineY(todayTop))},
		{name: "läxor", area: rowRect(contentRect, firstRowBaselineY(homeworkTop))},
		{name: "kommande", area: rowRect(contentRect, firstRowBaselineY(upcomingTop))},
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
	type area struct {
		name string
		area image.Rectangle
	}
	tests := []area{
		{name: "top margin", area: image.Rect(inner.Min.X, inner.Min.Y, inner.Max.X, outerMargin)},
		{name: "bottom margin", area: image.Rect(inner.Min.X, contentBottom, inner.Max.X, inner.Max.Y)},
		{name: "left margin", area: image.Rect(inner.Min.X, inner.Min.Y, contentX0, inner.Max.Y)},
		{name: "right margin", area: image.Rect(contentX1, inner.Min.Y, inner.Max.X, inner.Max.Y)},
	}
	// Each section's Y range runs from its rule to the next section's rule
	// (or contentBottom); both strips beside the content column stay white.
	sections := []struct {
		name   string
		y0, y1 int
	}{
		{name: "lunch", y0: lunchTop, y1: todayTop},
		{name: "idag", y0: todayTop, y1: homeworkTop},
		{name: "läxor", y0: homeworkTop, y1: upcomingTop},
		{name: "kommande", y0: upcomingTop, y1: contentBottom},
	}
	for _, s := range sections {
		tests = append(tests,
			area{name: s.name + " left of content", area: image.Rect(inner.Min.X, s.y0, contentX0, s.y1)},
			area{name: s.name + " right of content", area: image.Rect(contentX1, s.y0, inner.Max.X, s.y1)},
		)
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
	width := contentRect.Dx()
	// Every stress section has 20 entries: one-line lunch dishes and
	// two-line events, homework and upcoming. A section shows whole entries
	// in at most budget-1 lines plus "+N till". One-line lunch fills its
	// budget; two-line entries leave a line empty when budget-1 is odd
	// (events, budget 6: two entries + "+18 till" = 5 lines).
	tests := []struct {
		name       string
		rows       []string
		entries    int
		entryLines int
		budget     int
	}{
		{name: "lunch, one-line entries fill budget 4", rows: lunchRows(v.Lunch, width), entries: len(v.Lunch), entryLines: 1, budget: lunchLineBudget},
		{name: "events, two-line entries leave budget 6 one short", rows: eventRows(v.Events, width), entries: len(v.Events), entryLines: 2, budget: todayLineBudget},
		{name: "homework flattened across groups, two-line entries fill budget 5", rows: homeworkRows(v.Homework, width), entries: len(v.Homework), entryLines: 2, budget: homeworkLineBudget},
		{name: "upcoming, two-line entries fill budget 5", rows: upcomingRows(v.Upcoming, width), entries: len(v.Upcoming), entryLines: 2, budget: upcomingLineBudget},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Whole entries that fit in budget-1 lines, then "+N till".
			shown := (tt.budget - 1) / tt.entryLines
			wantLines := shown*tt.entryLines + 1
			wantLast := fmt.Sprintf("+%d till", tt.entries-shown)
			if len(tt.rows) != wantLines {
				t.Fatalf("got %d lines %q, want %d", len(tt.rows), tt.rows, wantLines)
			}
			if last := tt.rows[len(tt.rows)-1]; last != wantLast {
				t.Errorf("last line = %q, want %q", last, wantLast)
			}
			// Line 1 of every entry is flush left, later lines are indented.
			for i, row := range tt.rows[:len(tt.rows)-1] {
				if indented := strings.HasPrefix(row, " "); indented != (i%tt.entryLines != 0) {
					t.Errorf("line %d %q: indented = %v, want entries of %d lines", i, row, indented, tt.entryLines)
				}
			}
			for _, row := range tt.rows {
				if w := font.MeasureString(bodyFace, row).Ceil(); w > width {
					t.Errorf("line %q is %dpx wide, want ≤ %d", row, w, width)
				}
			}
		})
	}
}

func TestVisibleRows_KeepSuffix(t *testing.T) {
	long := strings.Repeat("Läxa ", 100)
	last := func(rows []string) string { return rows[len(rows)-1] }
	tests := []struct {
		name       string
		rows       []string
		wantSuffix string
	}{
		{
			name:       "homework keeps prov marker and day label",
			rows:       homeworkRows([]HomeworkGroup{{Initial: "K", Items: []HomeworkRow{{Summary: long, DayLabel: "fredag", IsProv: true}}}}, contentRect.Dx()),
			wantSuffix: "… (Prov) – fredag",
		},
		{
			name:       "upcoming keeps time",
			rows:       upcomingRows([]UpcomingRow{{Weekday: "Mån", Time: "18:30", Owner: "Kaj", Summary: long}}, contentRect.Dx()),
			wantSuffix: "…, 18:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.rows) != maxEntryLines {
				t.Fatalf("got %d lines, want %d", len(tt.rows), maxEntryLines)
			}
			if !strings.HasSuffix(last(tt.rows), tt.wantSuffix) {
				t.Errorf("last line %q does not end in %q", last(tt.rows), tt.wantSuffix)
			}
		})
	}
}

func TestBudgetLines(t *testing.T) {
	tests := []struct {
		name    string
		entries [][]string
		budget  int
		want    []string
	}{
		{name: "all fit", entries: [][]string{{"a"}, {"b1", "b2"}}, budget: 4, want: []string{"a", "b1", "b2"}},
		{name: "exactly the budget", entries: [][]string{{"a1", "a2"}, {"b"}, {"c"}}, budget: 4, want: []string{"a1", "a2", "b", "c"}},
		{name: "one-line entries overflow", entries: [][]string{{"a"}, {"b"}, {"c"}, {"d"}, {"e"}}, budget: 3, want: []string{"a", "b", "+3 till"}},
		{
			name:    "two-line entry past budget-1 hidden whole",
			entries: [][]string{{"a"}, {"b"}, {"c1", "c2"}, {"d"}},
			budget:  4,
			want:    []string{"a", "b", "+2 till"},
		},
		{name: "no entries", entries: nil, budget: 4, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := budgetLines(tt.entries, tt.budget)
			if strings.Join(got, "|") != strings.Join(tt.want, "|") || len(got) != len(tt.want) {
				t.Errorf("budgetLines = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWrapEntry(t *testing.T) {
	const maxWidth = 300
	long := strings.Repeat("Fotbollsträning på Åvallen ", 20)
	// Go Mono is 9 px per rune, so a line holds 33 runes. The indent is as
	// wide as the prefix, so a full line after the indent has the same room
	// as line 1 after the prefix: 33-13 = 20 runes for "07:15  Kaj – ".
	word20 := strings.Repeat("å", 20)
	indent12 := strings.Repeat(" ", 12)
	tests := []struct {
		name       string
		prefix     string
		body       string
		suffix     string
		wantLines  int
		wantSuffix string   // the last line must end in this
		cutWhole   bool     // prefix and suffix don't fit, so the whole text is cut
		want       []string // exact lines, when set
	}{
		{name: "short entry stays one line", prefix: "07:15  Kaj – ", body: "Tandläkare", wantLines: 1, wantSuffix: "Tandläkare"},
		{name: "long entry wraps at a space", prefix: "07:15  Kaj – ", body: "Fotbollsträning på Åvallen med laget", wantLines: 2, wantSuffix: "laget"},
		{
			name: "heldag suffix ends last line whole", prefix: "Tors  Kaj – ", body: "Simning på Hässelbybadet med skolan", suffix: ", heldag", wantLines: 2, wantSuffix: ", heldag",
			want: []string{"Tors  Kaj – Simning på", indent12 + "Hässelbybade…, heldag"},
		},
		{
			name: "time suffix ends last line whole", prefix: "Tors  Kaj – ", body: "Simning på Hässelbybadet med skolan", suffix: ", 10:00", wantLines: 2, wantSuffix: ", 10:00",
			want: []string{"Tors  Kaj – Simning på", indent12 + "Hässelbybadet…, 10:00"},
		},
		{
			name: "body fits line 1, suffix carries last word to line 2", prefix: "Tors  Kaj – ", body: "Simning i Hässelby", suffix: ", heldag", wantLines: 2, wantSuffix: ", heldag",
			want: []string{"Tors  Kaj – Simning i", indent12 + "Hässelby, heldag"},
		},
		{
			name: "single word cut on line 2", prefix: "Tors  Kaj – ", body: "Simning " + strings.Repeat("å", 30), suffix: ", 10:00", wantLines: 2, wantSuffix: "…, 10:00",
			want: []string{"Tors  Kaj – Simning", indent12 + strings.Repeat("å", 13) + "…, 10:00"},
		},
		{
			name: "word filling a full line stays whole", prefix: "07:15  Kaj – ", body: word20 + " Åvallen", wantLines: 2, wantSuffix: "Åvallen",
			want: []string{"07:15  Kaj – " + word20, "             Åvallen"},
		},
		{
			name: "word one rune wider than a full line cut", prefix: "07:15  Kaj – ", body: word20 + "å Åvallen", wantLines: 2, wantSuffix: "Åvallen",
			want: []string{"07:15  Kaj – " + strings.Repeat("å", 19) + "…", "             Åvallen"},
		},
		{name: "prov suffix ends last line", prefix: "L  ", body: "Läxa engelska kapitel tre och fyra, glosor", suffix: " (Prov) – idag", wantLines: 2, wantSuffix: " (Prov) – idag"},
		{name: "too long for two lines", prefix: "Tors  Kaj – ", body: long, suffix: ", 10:00", wantLines: 2, wantSuffix: "…, 10:00"},
		{name: "single over-wide word cut", prefix: "L  ", body: strings.Repeat("å", 100), wantLines: 1, wantSuffix: "…"},
		{name: "over-wide word on its own line", prefix: "L  ", body: strings.Repeat("å", 100) + " glosor", suffix: " – idag", wantLines: 2, wantSuffix: "glosor – idag"},
		{name: "prefix and suffix too wide", prefix: strings.Repeat("Kaj ", 20), body: "Simning", suffix: ", 10:00", wantLines: 1, wantSuffix: "…", cutWhole: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := wrapEntry(bodyFace, tt.prefix, tt.body, tt.suffix, maxWidth)
			if len(lines) != tt.wantLines {
				t.Fatalf("wrapEntry = %q, want %d lines", lines, tt.wantLines)
			}
			if tt.want != nil && strings.Join(lines, "|") != strings.Join(tt.want, "|") {
				t.Errorf("wrapEntry = %q, want %q", lines, tt.want)
			}
			if !strings.HasPrefix(lines[0], tt.prefix) && !tt.cutWhole {
				t.Errorf("line 1 %q does not start with prefix %q", lines[0], tt.prefix)
			}
			if last := lines[len(lines)-1]; !strings.HasSuffix(last, tt.wantSuffix) {
				t.Errorf("last line %q does not end in %q", last, tt.wantSuffix)
			}
			for _, line := range lines {
				if w := measure(bodyFace, line); w > maxWidth {
					t.Errorf("line %q is %dpx wide, want ≤ %d", line, w, maxWidth)
				}
			}
			if len(lines) < 2 {
				return
			}
			second := []rune(lines[1])
			n := len([]rune(tt.prefix))
			indent := string(second[:n])
			if strings.TrimLeft(indent, " ") != "" {
				t.Errorf("line 2 %q does not start with %d spaces", lines[1], n)
			}
			if got, want := measure(bodyFace, indent), measure(bodyFace, tt.prefix); got != want {
				t.Errorf("line 2 indent is %dpx wide, want prefix width %d", got, want)
			}
			if second[n] == ' ' {
				t.Errorf("line 2 %q has a space after the indent", lines[1])
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
		{name: "Min.Y is a multiple of 8", ok: r.Min.Y%8 == 0},
		{name: "Max.Y is a multiple of 8", ok: r.Max.Y%8 == 0},
		{name: "inside the screen", ok: r.In(screenRect)},
		{name: "inside the border", ok: r.In(inner)},
		{name: "inside contentRect", ok: r.In(contentRect)},
		{name: "left of weatherRuleX", ok: r.Max.X <= weatherRuleX},
		{name: "does not overlap weatherRect", ok: !r.Overlaps(weatherRect)},
		{name: "wide enough for 88:88", ok: font.MeasureString(clockFace, "88:88").Ceil() <= r.Dx()},
		{name: "wide enough for 00:00", ok: font.MeasureString(clockFace, "00:00").Ceil() <= r.Dx()},
		{name: "tall enough for ascent plus descent", ok: m.Ascent.Ceil()+m.Descent.Ceil() <= r.Dy()},
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

func TestRender_WeatherOutsideClockRect(t *testing.T) {
	v := stressView()
	v.Clock = ""
	if n := countBlack(Render(v), ClockRect()); n != 0 {
		t.Errorf("%d black pixels inside ClockRect %v with no clock, want none", n, ClockRect())
	}
}

func TestLayout_SectionOrder(t *testing.T) {
	tests := []struct {
		name         string
		above, below int
	}{
		{name: "band above lunch", above: bandBottom, below: lunchTop},
		{name: "lunch above today", above: lunchTop, below: todayTop},
		{name: "today above homework", above: todayTop, below: homeworkTop},
		{name: "homework above upcoming", above: homeworkTop, below: upcomingTop},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.above >= tt.below {
				t.Errorf("y=%d is not above y=%d", tt.above, tt.below)
			}
		})
	}
}

func TestLayout_WeatherFitsBand(t *testing.T) {
	m := temperatureFace.Metrics()
	tests := []struct {
		name string
		ok   bool
	}{
		{name: "temperature below band top", ok: temperatureBaselineY-m.Ascent.Ceil() >= weatherRect.Min.Y},
		{name: "first line below temperature", ok: weatherLinesBaselineY-bodyFace.Metrics().Ascent.Ceil() >= temperatureBaselineY+m.Descent.Ceil()},
		{name: "last line inside band", ok: weatherBottom <= bandBottom},
		{name: "weather right of the rule", ok: weatherRect.Min.X > weatherRuleX+weatherRuleThickness},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.ok {
				t.Errorf("weatherRect %v: %s does not hold", weatherRect, tt.name)
			}
		})
	}
}

func TestLayout_BodyDescent(t *testing.T) {
	if got := bodyFace.Metrics().Descent.Ceil(); got > bodyDescent {
		t.Errorf("body face descent = %d, want ≤ bodyDescent (%d)", got, bodyDescent)
	}
}

func TestLayout_SectionsFitAboveBottomMargin(t *testing.T) {
	descent := bodyFace.Metrics().Descent.Ceil()
	// Each section's last budgeted line stays above the next section's
	// rule, and the last section above the bottom margin.
	tests := []struct {
		name         string
		lastBaseline int
		limit        int
	}{
		{name: "lunch", lastBaseline: firstRowBaselineY(lunchTop) + (lunchLineBudget-1)*rowHeight, limit: todayTop},
		{name: "today", lastBaseline: firstRowBaselineY(todayTop) + (todayLineBudget-1)*rowHeight, limit: homeworkTop},
		{name: "homework", lastBaseline: firstRowBaselineY(homeworkTop) + (homeworkLineBudget-1)*rowHeight, limit: upcomingTop},
		{name: "upcoming", lastBaseline: firstRowBaselineY(upcomingTop) + (upcomingLineBudget-1)*rowHeight, limit: contentBottom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if bottom := tt.lastBaseline + descent; bottom > tt.limit {
				t.Errorf("last line reaches y=%d, want ≤ %d", bottom, tt.limit)
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
