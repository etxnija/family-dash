package eink

import (
	"fmt"
	"image"
	"image/draw"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	ellipsis = "…"
	// weatherSeparator is where an over-wide weather line is wrapped.
	weatherSeparator = " · "
)

// Fonts are embedded constants, so they are parsed and their faces built
// once at package init.
var (
	regularFont = mustParseFont("gomono", gomono.TTF)
	boldFont    = mustParseFont("gomonobold", gomonobold.TTF)

	clockFace       = mustNewFace("gomonobold", boldFont, clockFontSize)
	dateFace        = mustNewFace("gomonobold", boldFont, dateFontSize)
	headingFace     = mustNewFace("gomonobold", boldFont, headingFontSize)
	temperatureFace = mustNewFace("gomonobold", boldFont, temperatureFontSize)
	bodyFace        = mustNewFace("gomono", regularFont, bodyFontSize)
)

// renderMu serializes Render: opentype faces cache glyph data and are not
// safe for concurrent use.
var renderMu sync.Mutex

func mustParseFont(name string, src []byte) *opentype.Font {
	f, err := opentype.Parse(src)
	if err != nil {
		panic(fmt.Sprintf("eink: parsing font %s: %v", name, err))
	}
	return f
}

// mustNewFace builds a face of size pixels (DPI 72 makes points pixels).
func mustNewFace(name string, f *opentype.Font, size float64) font.Face {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic(fmt.Sprintf("eink: creating %s face at %vpx: %v", name, size, err))
	}
	return face
}

// Render draws v onto a black/white 800x480 frame. Every pixel of the
// result is either 0 (black) or 255 (white).
func Render(v View) *image.Gray {
	renderMu.Lock()
	defer renderMu.Unlock()

	img := image.NewGray(screenRect)
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	drawFrame(img)
	drawHeader(subImage(img, headerRect), v)

	left := subImage(img, leftColumnRect)
	drawClock(subImage(img, ClockRect()), v.Clock)
	drawWeather(left, v.Weather)
	drawSection(left, lunchTop, headingLunch, lunchRows(v.Lunch, left.Bounds().Dx()), emptyLunch)

	right := subImage(img, rightColumnRect)
	width := right.Bounds().Dx()
	drawSection(right, todayTop, headingToday, eventRows(v.Events, width), emptyToday)
	drawSection(right, homeworkTop, headingHomework, homeworkRows(v.Homework, width), emptyHomework)
	drawSection(right, upcomingTop, headingUpcoming, upcomingRows(v.Upcoming, width), emptyUpcoming)

	threshold(img)
	return img
}

// drawFrame draws the border along the screen edges and the header rule.
func drawFrame(img *image.Gray) {
	b := img.Bounds()
	fillRect(img, image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+borderThickness))
	fillRect(img, image.Rect(b.Min.X, b.Max.Y-borderThickness, b.Max.X, b.Max.Y))
	fillRect(img, image.Rect(b.Min.X, b.Min.Y, b.Min.X+borderThickness, b.Max.Y))
	fillRect(img, image.Rect(b.Max.X-borderThickness, b.Min.Y, b.Max.X, b.Max.Y))

	fillRect(img, image.Rect(headerRect.Min.X, headerRuleY, headerRect.Max.X, headerRuleY+headerRuleThickness))
}

// drawHeader draws the date and week label on the left and the owners
// right-aligned, all on dst (the header area).
func drawHeader(dst *image.Gray, v View) {
	b := dst.Bounds()
	date := fitText(dateFace, v.DateLabel, b.Dx())
	x := b.Min.X + drawText(dst, dateFace, b.Min.X, dateBaselineY, date)

	if v.WeekLabel != "" {
		x += weekLabelGap
		x += drawText(dst, bodyFace, x, dateBaselineY, fitText(bodyFace, v.WeekLabel, b.Max.X-x))
	}

	if v.Owners != "" {
		owners := fitText(bodyFace, v.Owners, b.Max.X-x-weekLabelGap)
		w := font.MeasureString(bodyFace, owners).Ceil()
		drawText(dst, bodyFace, b.Max.X-w, ownersBaselineY, owners)
	}
}

// drawClock draws clock centered in dst (the clock rectangle).
func drawClock(dst *image.Gray, clock string) {
	b := dst.Bounds()
	m := clockFace.Metrics()
	textHeight := m.Ascent.Ceil() + m.Descent.Ceil()
	baseline := b.Min.Y + (b.Dy()-textHeight)/2 + m.Ascent.Ceil()

	clock = fitText(clockFace, clock, b.Dx())
	w := font.MeasureString(clockFace, clock).Ceil()
	drawText(dst, clockFace, b.Min.X+(b.Dx()-w)/2, baseline, clock)
}

// drawWeather draws the large temperature and the lines beneath it.
func drawWeather(dst *image.Gray, w WeatherView) {
	b := dst.Bounds()
	drawText(dst, temperatureFace, b.Min.X, temperatureBaselineY, fitText(temperatureFace, w.Temperature, b.Dx()))

	lines := wrapAt(bodyFace, w.Lines, weatherSeparator, b.Dx())
	for i, line := range lines {
		if i == maxWeatherLines {
			break
		}
		drawText(dst, bodyFace, b.Min.X, weatherLinesBaselineY+i*rowHeight, line)
	}
}

// drawSection draws a thin rule at top, heading below it and rows below
// that, or empty when there are no rows.
func drawSection(dst *image.Gray, top int, heading string, rows []string, empty string) {
	b := dst.Bounds()
	fillRect(dst, image.Rect(b.Min.X, top, b.Max.X, top+sectionRuleThickness))

	drawText(dst, headingFace, b.Min.X, headingBaselineY(top), fitText(headingFace, heading, b.Dx()))

	if len(rows) == 0 {
		rows = []string{fitText(bodyFace, empty, b.Dx())}
	}
	baseline := firstRowBaselineY(top)
	for i, row := range rows {
		drawText(dst, bodyFace, b.Min.X, baseline+i*rowHeight, row)
	}
}

// lunchRows returns the visible lunch rows, each fitted to maxWidth.
func lunchRows(lunch []string, maxWidth int) []string {
	rows := make([]string, 0, len(lunch))
	for _, dish := range lunch {
		rows = append(rows, fitText(bodyFace, dish, maxWidth))
	}
	return limitRows(rows, maxLunchRows)
}

// eventRows returns the visible "<time>  <Owner> – <Summary>[, <Location>]"
// rows, each fitted to maxWidth.
func eventRows(events []EventRow, maxWidth int) []string {
	rows := make([]string, 0, len(events))
	for _, e := range events {
		rows = append(rows, fitParts(bodyFace, e.Time+"  "+ownerPrefix(e.Owner), withLocation(e.Summary, e.Location), "", maxWidth))
	}
	return limitRows(rows, maxEventRows)
}

// homeworkRows returns the visible "<Initial>  <Summary>[ (Prov)] –
// <daylabel>" rows across all groups, each fitted to maxWidth so the day
// label is never cut.
func homeworkRows(groups []HomeworkGroup, maxWidth int) []string {
	var rows []string
	for _, g := range groups {
		for _, item := range g.Items {
			suffix := " – " + item.DayLabel
			if item.IsProv {
				suffix = " (Prov)" + suffix
			}
			rows = append(rows, fitParts(bodyFace, g.Initial+"  ", item.Summary, suffix, maxWidth))
		}
	}
	return limitRows(rows, maxHomeworkRows)
}

// upcomingRows returns the visible "<Abbrev>  <Owner> – <Summary>[,
// <Location>], <time|heldag>" rows, each fitted to maxWidth so the time is
// never cut.
func upcomingRows(upcoming []UpcomingRow, maxWidth int) []string {
	rows := make([]string, 0, len(upcoming))
	for _, u := range upcoming {
		rows = append(rows, fitParts(bodyFace, u.Weekday+"  "+ownerPrefix(u.Owner), withLocation(u.Summary, u.Location), ", "+u.Time, maxWidth))
	}
	return limitRows(rows, maxUpcomingRows)
}

// ownerPrefix returns "<owner> – ", or "" when owner is empty.
func ownerPrefix(owner string) string {
	if owner == "" {
		return ""
	}
	return owner + " – "
}

// withLocation returns "<summary>, <location>", or summary alone when
// location is empty.
func withLocation(summary, location string) string {
	if location == "" {
		return summary
	}
	return summary + ", " + location
}

// limitRows returns rows unchanged when there are at most limit of them.
// Otherwise it keeps the first limit-1 rows and adds "+N till", where N is
// the number of hidden rows.
func limitRows(rows []string, limit int) []string {
	if len(rows) <= limit {
		return rows
	}
	visible := append([]string(nil), rows[:limit-1]...)
	return append(visible, fmt.Sprintf("+%d till", len(rows)-(limit-1)))
}

// fitText returns s unchanged when it fits in maxWidth pixels, otherwise
// the longest prefix of s followed by "…" that fits.
func fitText(face font.Face, s string, maxWidth int) string {
	limit := fixed.I(maxWidth)
	if font.MeasureString(face, s) <= limit {
		return s
	}
	runes := []rune(s)
	fits := func(n int) bool {
		return font.MeasureString(face, strings.TrimRight(string(runes[:n]), " ")+ellipsis) <= limit
	}
	if !fits(0) {
		return ""
	}
	// Binary search for the largest n with fits(n); fits(0) holds and
	// fits(len(runes)) doesn't.
	lo, hi := 0, len(runes)
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if fits(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return strings.TrimRight(string(runes[:lo]), " ") + ellipsis
}

// fitParts joins prefix, middle and suffix, cutting only middle so the
// result fits in maxWidth pixels. If prefix and suffix alone don't fit,
// the whole joined text is cut instead.
func fitParts(face font.Face, prefix, middle, suffix string, maxWidth int) string {
	fixedWidth := font.MeasureString(face, prefix+suffix).Ceil()
	if fixedWidth+font.MeasureString(face, ellipsis).Ceil() > maxWidth {
		return fitText(face, prefix+middle+suffix, maxWidth)
	}
	return prefix + fitText(face, middle, maxWidth-fixedWidth) + suffix
}

// wrapAt returns lines with every line wider than maxWidth split at sep
// into several lines, each fitted to maxWidth.
func wrapAt(face font.Face, lines []string, sep string, maxWidth int) []string {
	var wrapped []string
	for _, line := range lines {
		parts := []string{line}
		if font.MeasureString(face, line).Ceil() > maxWidth {
			parts = strings.Split(line, sep)
		}
		for _, part := range parts {
			wrapped = append(wrapped, fitText(face, part, maxWidth))
		}
	}
	return wrapped
}

// drawText draws s on dst with its baseline origin at (x, y) and returns
// its advance width in pixels. Drawing is clipped to dst's bounds.
func drawText(dst draw.Image, face font.Face, x, y int, s string) int {
	if s == "" {
		return 0
	}
	d := font.Drawer{Dst: dst, Src: image.Black, Face: face, Dot: fixed.P(x, y)}
	d.DrawString(s)
	return (d.Dot.X - fixed.I(x)).Ceil()
}

// fillRect paints r black on dst, clipped to dst's bounds.
func fillRect(dst draw.Image, r image.Rectangle) {
	draw.Draw(dst, r, image.Black, image.Point{}, draw.Src)
}

// subImage returns the part of img inside r, sharing img's pixels and
// coordinate space, so drawing on it can't spill outside r.
func subImage(img *image.Gray, r image.Rectangle) *image.Gray {
	return img.SubImage(r).(*image.Gray)
}

// threshold turns every pixel of img pure black (<128) or pure white.
func threshold(img *image.Gray) {
	for i, p := range img.Pix {
		if p < 128 {
			img.Pix[i] = 0
		} else {
			img.Pix[i] = 255
		}
	}
}
