package eink

import (
	"fmt"
	"image"
	"reflect"
	"slices"
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
	for _, tt := range layoutViews(t) {
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

// sectionNames names the sections in subtests, in section order.
var sectionNames = [sectionCount]string{"lunch", "idag", "läxor", "kommande"}

// layoutFor returns the section layout Render uses for v.
func layoutFor(v View) [sectionCount]sectionLayout {
	return layoutSections(sectionEntries(v, contentRect.Dx()))
}

// layoutViews returns the full, empty and stress views.
func layoutViews(t *testing.T) []struct {
	name string
	view View
} {
	t.Helper()
	return []struct {
		name string
		view View
	}{
		{name: "fully populated", view: fullView()},
		{name: "empty", view: emptyView(t)},
		{name: "stress", view: stressView()},
	}
}

// sectionBottom returns the Y below section i: the next section's rule,
// or contentBottom for the last section.
func sectionBottom(layout [sectionCount]sectionLayout, i int) int {
	if i+1 < sectionCount {
		return layout[i+1].top
	}
	return contentBottom
}

// flatten returns the lines of entries in order.
func flatten(entries [][]string) []string {
	var lines []string
	for _, e := range entries {
		lines = append(lines, e...)
	}
	return lines
}

func TestRender_EmptyStatesDrawn(t *testing.T) {
	v := emptyView(t)
	img := Render(v)
	layout := layoutFor(v)

	for i, name := range sectionNames {
		t.Run(name, func(t *testing.T) {
			area := rowRect(contentRect, firstRowBaselineY(layout[i].top))
			if countBlack(img, area) == 0 {
				t.Errorf("no black pixels in %s empty-state area %v", name, area)
			}
		})
	}
}

func TestRender_StaysContained(t *testing.T) {
	inner := image.Rect(borderThickness, borderThickness, ScreenWidth-borderThickness, ScreenHeight-borderThickness)
	type area struct {
		name string
		area image.Rectangle
	}

	for _, view := range layoutViews(t) {
		t.Run(view.name, func(t *testing.T) {
			img := Render(view.view)
			layout := layoutFor(view.view)
			tests := []area{
				{name: "top margin", area: image.Rect(inner.Min.X, inner.Min.Y, inner.Max.X, outerMargin)},
				{name: "bottom margin", area: image.Rect(inner.Min.X, contentBottom, inner.Max.X, inner.Max.Y)},
				{name: "left margin", area: image.Rect(inner.Min.X, inner.Min.Y, contentX0, inner.Max.Y)},
				{name: "right margin", area: image.Rect(contentX1, inner.Min.Y, inner.Max.X, inner.Max.Y)},
			}
			// Each section's Y range runs from its rule to the next section's
			// rule (or contentBottom); both strips beside the content column
			// stay white.
			for i, name := range sectionNames {
				y0, y1 := layout[i].top, sectionBottom(layout, i)
				tests = append(tests,
					area{name: name + " left of content", area: image.Rect(inner.Min.X, y0, contentX0, y1)},
					area{name: name + " right of content", area: image.Rect(contentX1, y0, inner.Max.X, y1)},
				)
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					if n := countBlack(img, tt.area); n != 0 {
						t.Errorf("%d black pixels in %v, want none", n, tt.area)
					}
				})
			}
		})
	}
}

// lineRoom is the number of body lines the four packed sections hold in
// total above contentBottom.
const lineRoom = (contentBottom - bodyDescent - sectionsTop - (sectionCount-1)*sectionGap - sectionCount*headingOffset) / rowHeight

// trimmedLines returns how many lines budgetLines gives for entries of
// entryLines lines each under budget.
func trimmedLines(entries, entryLines, budget int) int {
	if entries*entryLines <= budget {
		return entries * entryLines
	}
	return (budget-1)/entryLines*entryLines + 1
}

func TestLayoutSections_StressView(t *testing.T) {
	v := stressView()
	width := contentRect.Dx()
	layout := layoutFor(v)

	// Every stress section has 20 entries: one-line lunch dishes and
	// two-line events, homework and upcoming. Far too much fits, so
	// KOMMANDE, LÄXOR and SKOLLUNCH are trimmed to minSectionLines in that
	// order, and IDAG keeps the largest budget that fits the lines left.
	// With the current numbers: "+20 till", "+20 till", one dish +
	// "+19 till", and IDAG budget 16 = 7 entries + "+13 till" (19 lines).
	entryLines := [sectionCount]int{sectionLunch: 1, sectionToday: 2, sectionHomework: 2, sectionUpcoming: 2}
	entries := [sectionCount]int{
		sectionLunch:    len(v.Lunch),
		sectionToday:    len(v.Events),
		sectionHomework: len(v.Homework),
		sectionUpcoming: len(v.Upcoming),
	}
	var budget [sectionCount]int
	others := 0
	for _, s := range []int{sectionLunch, sectionHomework, sectionUpcoming} {
		budget[s] = minSectionLines
		others += trimmedLines(entries[s], entryLines[s], budget[s])
	}
	budget[sectionToday] = entries[sectionToday] * entryLines[sectionToday]
	for trimmedLines(entries[sectionToday], entryLines[sectionToday], budget[sectionToday])+others > lineRoom {
		budget[sectionToday]--
	}
	if budget[sectionToday] < minTodayLines {
		t.Fatalf("IDAG budget %d below minTodayLines %d", budget[sectionToday], minTodayLines)
	}

	for i, name := range sectionNames {
		t.Run(name, func(t *testing.T) {
			rows := layout[i].lines
			// Whole entries that fit in budget-1 lines, then "+N till".
			shown := (budget[i] - 1) / entryLines[i]
			wantLines := shown*entryLines[i] + 1
			wantLast := fmt.Sprintf("+%d till", entries[i]-shown)
			if len(rows) != wantLines {
				t.Fatalf("got %d lines %q, want %d", len(rows), rows, wantLines)
			}
			if last := rows[len(rows)-1]; last != wantLast {
				t.Errorf("last line = %q, want %q", last, wantLast)
			}
			// Line 1 of every entry is flush left, later lines are indented.
			for j, row := range rows[:len(rows)-1] {
				if indented := strings.HasPrefix(row, " "); indented != (j%entryLines[i] != 0) {
					t.Errorf("line %d %q: indented = %v, want entries of %d lines", j, row, indented, entryLines[i])
				}
			}
			for _, row := range rows {
				if w := font.MeasureString(bodyFace, row).Ceil(); w > width {
					t.Errorf("line %q is %dpx wide, want ≤ %d", row, w, width)
				}
			}
		})
	}
}

// synthEntries returns n entries of lines lines each, tagged so every
// line is distinct; later lines are indented like wrapped entries.
func synthEntries(tag string, n, lines int) [][]string {
	entries := make([][]string, n)
	for i := range entries {
		for j := range lines {
			line := fmt.Sprintf("%s%d", tag, i)
			if j > 0 {
				line = fmt.Sprintf("  %s%d.%d", tag, i, j)
			}
			entries[i] = append(entries[i], line)
		}
	}
	return entries
}

// mixedEntries returns n entries tagged like synthEntries that alternate
// between one and two lines, starting with one.
func mixedEntries(tag string, n int) [][]string {
	entries := make([][]string, n)
	for i := range entries {
		entries[i] = synthEntries(fmt.Sprintf("%s%d.", tag, i), 1, 1+i%2)[0]
	}
	return entries
}

func TestLayoutSections(t *testing.T) {
	// lineRoom is 20 with the current constants; the cases below are sized
	// against it. shown is how many whole entries each section shows; when
	// it is below the entry count the section ends in "+N till".
	tests := []struct {
		name    string
		entries [sectionCount][][]string
		shown   [sectionCount]int
		// minTodayVisible, when set, is the fewest lines IDAG must show.
		minTodayVisible int
	}{
		{
			name:    "all fit",
			entries: [sectionCount][][]string{synthEntries("l", 2, 1), synthEntries("t", 3, 2), synthEntries("h", 2, 2), synthEntries("u", 2, 2)},
			shown:   [sectionCount]int{2, 3, 2, 2},
		},
		{
			name: "empty",
		},
		{
			name:    "mixed, only IDAG and KOMMANDE",
			entries: [sectionCount][][]string{sectionToday: synthEntries("t", 5, 2), sectionUpcoming: synthEntries("u", 3, 2)},
			shown:   [sectionCount]int{0, 5, 0, 3},
		},
		{
			// 2+6+4+10 = 22 lines: KOMMANDE budget 8 → 3 entries + "+2 till".
			name:    "slight overflow trims KOMMANDE only",
			entries: [sectionCount][][]string{synthEntries("l", 2, 1), synthEntries("t", 3, 2), synthEntries("h", 2, 2), synthEntries("u", 5, 2)},
			shown:   [sectionCount]int{2, 3, 2, 3},
		},
		{
			// KOMMANDE at its floor (1 line), then LÄXOR budget 12 → 5 entries.
			name:    "larger overflow trims LÄXOR after KOMMANDE",
			entries: [sectionCount][][]string{synthEntries("l", 2, 1), synthEntries("t", 3, 2), synthEntries("h", 7, 2), synthEntries("u", 5, 2)},
			shown:   [sectionCount]int{2, 3, 5, 0},
		},
		{
			// KOMMANDE and LÄXOR at 1 line, SKOLLUNCH budget 4 → 3 dishes.
			name:    "further overflow trims SKOLLUNCH",
			entries: [sectionCount][][]string{synthEntries("l", 6, 1), synthEntries("t", 7, 2), synthEntries("h", 7, 2), synthEntries("u", 5, 2)},
			shown:   [sectionCount]int{3, 7, 0, 0},
		},
		{
			// Others at budget 2 (2+1+1 lines), IDAG budget 16 → 7 entries.
			// IDAG must still show at least minTodayLines lines; this fails
			// if the constants change so that IDAG's budget floor shows fewer.
			name:            "extreme overflow trims IDAG last",
			entries:         [sectionCount][][]string{synthEntries("l", 10, 1), synthEntries("t", 20, 2), synthEntries("h", 10, 2), synthEntries("u", 10, 2)},
			shown:           [sectionCount]int{1, 7, 0, 0},
			minTodayVisible: minTodayLines,
		},
		{
			// IDAG's two lines are below minTodayLines and are kept; SKOLLUNCH
			// takes the rest: budget 16 → 15 dishes.
			name:    "IDAG below its floor keeps its lines",
			entries: [sectionCount][][]string{synthEntries("l", 20, 1), synthEntries("t", 1, 2), synthEntries("h", 20, 2), synthEntries("u", 20, 2)},
			shown:   [sectionCount]int{15, 1, 0, 0},
		},
		{
			// IDAG alternates one- and two-line entries (30 lines). Others
			// take 1 line each; IDAG budget 18 shows 11 entries (16 lines)
			// + "+9 till" = 17 lines, one below its budget, because the
			// 12th entry has two lines and is never cut.
			name:    "mixed-length IDAG shows fewer lines than its budget",
			entries: [sectionCount][][]string{synthEntries("l", 20, 2), mixedEntries("t", 20), synthEntries("h", 20, 2), synthEntries("u", 20, 2)},
			shown:   [sectionCount]int{0, 11, 0, 0},
		},
		{
			// Others 1 line each, IDAG budget 18 → 8 entries + "+12 till".
			name:    "stress, 20 two-line entries everywhere",
			entries: [sectionCount][][]string{synthEntries("l", 20, 2), synthEntries("t", 20, 2), synthEntries("h", 20, 2), synthEntries("u", 20, 2)},
			shown:   [sectionCount]int{0, 8, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := checkLayoutSections(t, tt.entries, tt.shown)
			if got := len(layout[sectionToday].lines); got < tt.minTodayVisible {
				t.Errorf("IDAG shows %d lines, want ≥ %d", got, tt.minTodayVisible)
			}
		})
	}
}

// checkLayoutSections lays out entries and checks the result is deterministic,
// shows the expected lines and packs the sections correctly, and returns it.
func checkLayoutSections(t *testing.T, entries [sectionCount][][]string, shown [sectionCount]int) [sectionCount]sectionLayout {
	t.Helper()
	layout := layoutSections(entries)
	if again := layoutSections(entries); !reflect.DeepEqual(layout, again) {
		t.Fatalf("layoutSections is not deterministic: %v then %v", layout, again)
	}

	checkSectionLines(t, layout, entries, shown)
	checkSectionPositions(t, layout)
	return layout
}

// checkSectionLines checks that each section shows its first shown entries
// whole, followed by "+N till" for the hidden ones.
func checkSectionLines(t *testing.T, layout [sectionCount]sectionLayout, entries [sectionCount][][]string, shown [sectionCount]int) {
	t.Helper()
	for i, name := range sectionNames {
		e := entries[i]
		want := flatten(e[:shown[i]])
		if hidden := len(e) - shown[i]; hidden > 0 {
			want = append(want, fmt.Sprintf("+%d till", hidden))
		}
		if got := layout[i].lines; !slices.Equal(got, want) {
			t.Errorf("%s lines = %q, want %q", name, got, want)
		}
		if len(e) == 0 && layout[i].lines != nil {
			t.Errorf("%s has no entries, lines = %q, want nil", name, layout[i].lines)
		}
	}
	// minTodayLines bounds IDAG's budget, not its visible lines, so this
	// only holds because lineRoom leaves IDAG far more than its floor even
	// with every other section at its own floor.
	if natural := len(flatten(entries[sectionToday])); len(layout[sectionToday].lines) < min(minTodayLines, natural) {
		t.Errorf("IDAG has %d lines, want ≥ %d", len(layout[sectionToday].lines), min(minTodayLines, natural))
	}
}

// checkSectionPositions checks that sections pack from lunchTop down and the
// last one stays above contentBottom.
func checkSectionPositions(t *testing.T, layout [sectionCount]sectionLayout) {
	t.Helper()
	if layout[0].top != lunchTop {
		t.Errorf("first top = %d, want lunchTop %d", layout[0].top, lunchTop)
	}
	for i := 1; i < sectionCount; i++ {
		want := layout[i-1].top + sectionHeight(max(len(layout[i-1].lines), 1)) + sectionGap
		if layout[i].top != want {
			t.Errorf("%s top = %d, want %d (packed below %s)", sectionNames[i], layout[i].top, want, sectionNames[i-1])
		}
	}
	last := layout[sectionCount-1]
	if bottom := last.top + sectionHeight(max(len(last.lines), 1)) + bodyDescent; bottom > contentBottom {
		t.Errorf("last row reaches y=%d, want ≤ %d", bottom, contentBottom)
	}
}

func TestVisibleRows_KeepSuffix(t *testing.T) {
	long := strings.Repeat("Läxa ", 100)
	last := func(rows []string) string { return rows[len(rows)-1] }
	width := contentRect.Dx()
	tests := []struct {
		name       string
		rows       []string
		wantSuffix string
	}{
		{
			name:       "homework keeps prov marker and day label",
			rows:       flatten(homeworkEntries([]HomeworkGroup{{Initial: "K", Items: []HomeworkRow{{Summary: long, DayLabel: "fredag", IsProv: true}}}}, width)),
			wantSuffix: "… (Prov) – fredag",
		},
		{
			name:       "upcoming keeps time",
			rows:       flatten(upcomingEntries([]UpcomingRow{{Weekday: "Mån", Time: "18:30", Owner: "Kaj", Summary: long}}, width)),
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

type wrapEntryCase struct {
	name       string
	prefix     string
	body       string
	suffix     string
	wantLines  int
	wantSuffix string   // the last line must end in this
	cutWhole   bool     // prefix and suffix don't fit, so the whole text is cut
	want       []string // exact lines, when set
}

func TestWrapEntry(t *testing.T) {
	const maxWidth = 300
	long := strings.Repeat("Fotbollsträning på Åvallen ", 20)
	// Go Mono is 9 px per rune, so a line holds 33 runes. The indent is as
	// wide as the prefix, so a full line after the indent has the same room
	// as line 1 after the prefix: 33-13 = 20 runes for "07:15  Kaj – ".
	word20 := strings.Repeat("å", 20)
	indent12 := strings.Repeat(" ", 12)
	tests := []wrapEntryCase{
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
			if len(lines) == 2 {
				checkHangingIndent(t, lines[1], tt.prefix)
			}
		})
	}
}

// checkHangingIndent checks that line 2 starts with one space per prefix
// rune, as wide as the prefix, followed directly by text.
func checkHangingIndent(t *testing.T, line, prefix string) {
	t.Helper()
	second := []rune(line)
	n := len([]rune(prefix))
	indent := string(second[:n])
	if strings.TrimLeft(indent, " ") != "" {
		t.Errorf("line 2 %q does not start with %d spaces", line, n)
	}
	if got, want := measure(bodyFace, indent), measure(bodyFace, prefix); got != want {
		t.Errorf("line 2 indent is %dpx wide, want prefix width %d", got, want)
	}
	if second[n] == ' ' {
		t.Errorf("line 2 %q has a space after the indent", line)
	}
}

func TestEntries_LabelsPaddedToWidest(t *testing.T) {
	width := contentRect.Dx()
	long := "Fotbollsträning på Åvallen med hela laget och alla tränarna efter skolan"
	tests := []struct {
		name      string
		entries   [][]string
		summaries []string // the summary each entry's first line holds
		// wantPrefix is the rune count before every summary: the widest
		// label, two spaces and "Kaj – ".
		wantPrefix int
	}{
		{
			name: "IDAG pads times to heldag",
			entries: eventEntries([]EventRow{
				{Time: "15:30", Owner: "Kaj", Summary: "Tandläkare"},
				{Time: "heldag", Owner: "Kaj", Summary: "Studiedag"},
				{Time: "18:00", Owner: "Kaj", Summary: long},
			}, width),
			summaries:  []string{"Tandläkare", "Studiedag", "Fotbollsträning"},
			wantPrefix: len("heldag") + 2 + len([]rune("Kaj – ")),
		},
		{
			name: "KOMMANDE pads weekdays to Tors",
			entries: upcomingEntries([]UpcomingRow{
				{Weekday: "Ons", Time: "10:00", Owner: "Kaj", Summary: "Simning"},
				{Weekday: "Tors", Time: "heldag", Owner: "Kaj", Summary: "Utflykt"},
				{Weekday: "Fre", Time: "17:00", Owner: "Kaj", Summary: long},
				{Weekday: "Lör", Time: "09:00", Owner: "Kaj", Summary: "Match"},
			}, width),
			summaries:  []string{"Simning", "Utflykt", "Fotbollsträning", "Match"},
			wantPrefix: len("Tors") + 2 + len([]rune("Kaj – ")),
		},
		{
			name: "single label length gets no padding",
			entries: upcomingEntries([]UpcomingRow{
				{Weekday: "Ons", Time: "10:00", Owner: "Kaj", Summary: "Simning"},
				{Weekday: "Fre", Time: "17:00", Owner: "Kaj", Summary: long},
			}, width),
			summaries:  []string{"Simning", "Fotbollsträning"},
			wantPrefix: len("Ons") + 2 + len([]rune("Kaj – ")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := false
			for i, entry := range tt.entries {
				prefix := checkSummaryAt(t, entry[0], tt.summaries[i], tt.wantPrefix)
				if len(entry) == 2 {
					wrapped = true
					checkHangingIndent(t, entry[1], prefix)
				}
			}
			if !wrapped {
				t.Fatal("no entry wrapped, so the hanging indent is untested")
			}
		})
	}
}

// checkSummaryAt checks that summary starts at rune index want in line,
// at the pixel x of want runes, and returns the text before it.
func checkSummaryAt(t *testing.T, line, summary string, want int) string {
	t.Helper()
	idx := strings.Index(line, summary)
	if idx < 0 {
		t.Fatalf("line %q does not hold %q", line, summary)
	}
	prefix := line[:idx]
	if got := len([]rune(prefix)); got != want {
		t.Errorf("line %q: summary at rune %d, want %d", line, got, want)
	}
	if got, wantPx := measure(bodyFace, prefix), measure(bodyFace, strings.Repeat(" ", want)); got != wantPx {
		t.Errorf("line %q: summary at x=%d, want %d", line, got, wantPx)
	}
	return prefix
}

func TestLayoutSections_PaddingFromHiddenEntry(t *testing.T) {
	width := contentRect.Dx()
	long := "Fotbollsträning på Åvallen med hela laget och alla tränarna efter skolan"
	var upcoming []UpcomingRow
	for range 12 {
		upcoming = append(upcoming, UpcomingRow{Weekday: "Ons", Time: "10:00", Owner: "Kaj", Summary: long})
	}
	// Only the last, hidden row has a four-rune weekday.
	upcoming = append(upcoming, UpcomingRow{Weekday: "Tors", Time: "10:00", Owner: "Kaj", Summary: long})

	var entries [sectionCount][][]string
	entries[sectionLunch] = lunchEntries([]string{"Köttbullar", "Gröt"}, width)
	entries[sectionToday] = eventEntries([]EventRow{{Time: "07:15", Owner: "Kaj", Summary: "Tandläkare"}}, width)
	entries[sectionUpcoming] = upcomingEntries(upcoming, width)
	layout := layoutSections(entries)

	lines := layout[sectionUpcoming].lines
	if len(lines) == 0 || !strings.HasPrefix(lines[len(lines)-1], "+") {
		t.Fatalf("KOMMANDE lines = %q, want it trimmed to end in \"+N till\"", lines)
	}
	for _, line := range lines[:len(lines)-1] {
		if strings.Contains(line, "Tors") {
			t.Fatalf("the Tors row is visible in %q, want it hidden", lines)
		}
	}
	checkSummaryAt(t, lines[0], "Fotbollsträning", len("Tors")+2+len([]rune("Kaj – ")))
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
	if bandBottom >= lunchTop {
		t.Errorf("bandBottom y=%d is not above lunchTop y=%d", bandBottom, lunchTop)
	}
	for _, view := range layoutViews(t) {
		t.Run(view.name, func(t *testing.T) {
			layout := layoutFor(view.view)
			if layout[0].top != lunchTop {
				t.Errorf("%s top = %d, want lunchTop %d", sectionNames[0], layout[0].top, lunchTop)
			}
			for i := 1; i < sectionCount; i++ {
				if layout[i-1].top >= layout[i].top {
					t.Errorf("%s y=%d is not above %s y=%d", sectionNames[i-1], layout[i-1].top, sectionNames[i], layout[i].top)
				}
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
	// Each section's last line stays above the next section's rule, and
	// the last section above the bottom margin.
	for _, view := range layoutViews(t) {
		t.Run(view.name, func(t *testing.T) {
			layout := layoutFor(view.view)
			for i, name := range sectionNames {
				lastBaseline := firstRowBaselineY(layout[i].top) + (max(len(layout[i].lines), 1)-1)*rowHeight
				if bottom, limit := lastBaseline+descent, sectionBottom(layout, i); bottom > limit {
					t.Errorf("%s last line reaches y=%d, want ≤ %d", name, bottom, limit)
				}
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
