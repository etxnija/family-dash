package eink

import "image"

// Layout of the portrait 480x800 frame, in pixels. Every number the
// renderer uses to place things lives here so the layout can be tweaked in
// one spot. The renderer never rotates: the panel adapter turns the frame
// to the panel's native landscape orientation (ADR-0004).
// Y values ending in "BaselineY" are text baselines; the rest are edges.
const (
	// ScreenWidth is the frame width.
	ScreenWidth = 480
	// ScreenHeight is the frame height.
	ScreenHeight = 800

	// borderThickness is the black frame drawn along the screen edges.
	borderThickness = 4
	// outerMargin is the white space between the screen edge and any
	// content; the part inside the border stays white.
	outerMargin = 16

	// contentTop is where the header starts.
	contentTop = outerMargin
	// contentBottom is the lowest Y any content may reach.
	contentBottom = ScreenHeight - outerMargin
	// contentX0 and contentX1 bound the single content column.
	contentX0 = outerMargin
	contentX1 = ScreenWidth - outerMargin

	// dateBaselineY is the baseline of the date and the week label.
	dateBaselineY = 44
	// weekLabelGap is the space between the date and the week label.
	weekLabelGap = 12
	// ownersBaselineY is the baseline of the owners, on a second header
	// line below the date.
	ownersBaselineY = 68
	// headerRuleY is the top of the rule under the header.
	headerRuleY = 78
	// headerRuleThickness is the height of the rule under the header.
	headerRuleThickness = 2
	// bodyTop is the first Y below the header rule.
	bodyTop = headerRuleY + headerRuleThickness

	// clockFontSize is the bold clock face size.
	clockFontSize = 72
	// dateFontSize is the bold date face size.
	dateFontSize = 28
	// headingFontSize is the bold section heading face size.
	headingFontSize = 16
	// bodyFontSize is the regular body text face size (minimum 15).
	bodyFontSize = 15
	// bodyDescent is the body face's descent at bodyFontSize, rounded up
	// (checked by TestLayout_BodyDescent).
	bodyDescent = 4
	// temperatureFontSize is the bold temperature face size.
	temperatureFontSize = 32

	// rowHeight is the baseline-to-baseline distance between body rows.
	rowHeight = 20
	// sectionRuleThickness is the thin rule drawn above each heading.
	sectionRuleThickness = 1
	// headingOffset is the distance from a section's rule to its heading
	// baseline; the first row's baseline is one rowHeight below that.
	headingOffset = 20
	// sectionGap is the space after a section's last row slot, before the
	// next section's rule.
	sectionGap = 16

	// lunchLineBudget, todayLineBudget, homeworkLineBudget and
	// upcomingLineBudget limit each section. A budget counts body lines,
	// not entries, and includes the "+N till" line that replaces the
	// entries that don't fit.
	lunchLineBudget    = 4
	todayLineBudget    = 6
	homeworkLineBudget = 5
	upcomingLineBudget = 5
	// maxEntryLines is the most lines one entry wraps to.
	maxEntryLines = 2

	// clockX0, clockX1, clockY0 and clockY1 bound the clock rectangle at
	// the left of the clock/weather band. The Y edges are multiples of 8
	// so the panel can partially refresh it (see ClockRect).
	clockX0 = contentX0
	clockX1 = clockX0 + 232
	clockY0 = 88
	clockY1 = clockY0 + 88

	// weatherRuleX is the left edge of the thin vertical rule between the
	// clock and the weather; it spans the band's Y range.
	weatherRuleX = clockX1 + 8
	// weatherRuleThickness is the width of that vertical rule.
	weatherRuleThickness = 1
	// weatherX0 is the left edge of the weather block, which runs to
	// contentX1.
	weatherX0 = weatherRuleX + weatherRuleThickness + 15

	// temperatureBaselineY is the baseline of the large temperature at the
	// top of the band.
	temperatureBaselineY = clockY0 + 31
	// weatherLinesBaselineY is the baseline of the first weather line.
	weatherLinesBaselineY = temperatureBaselineY + 24
	// maxWeatherLines is the number of weather line slots: the view's four
	// lines plus one for the low/high line, which wraps at " · ".
	maxWeatherLines = 5
	// weatherBottom is the lowest Y of the last weather line slot,
	// including the body face's descent.
	weatherBottom = weatherLinesBaselineY + (maxWeatherLines-1)*rowHeight + bodyDescent

	// bandBottom is the bottom of the clock/weather content: whichever of
	// the clock rect and the last weather slot is lower. The rule above
	// SKOLLUNCH, sectionGap below it, closes the band; the vertical weather
	// rule runs down to meet it at lunchTop.
	bandBottom = max(clockY1, weatherBottom)

	// lunchTop, todayTop, homeworkTop and upcomingTop are the Y of the rule
	// above each section, stacked in that order. Each section has fixed
	// room for its line budget, so positions don't move with the content.
	lunchTop    = bandBottom + sectionGap
	todayTop    = lunchTop + headingOffset + lunchLineBudget*rowHeight + sectionGap
	homeworkTop = todayTop + headingOffset + todayLineBudget*rowHeight + sectionGap
	upcomingTop = homeworkTop + headingOffset + homeworkLineBudget*rowHeight + sectionGap
)

var (
	screenRect  = image.Rect(0, 0, ScreenWidth, ScreenHeight)
	headerRect  = image.Rect(contentX0, contentTop, contentX1, headerRuleY)
	contentRect = image.Rect(contentX0, bodyTop, contentX1, contentBottom)
	weatherRect = image.Rect(weatherX0, clockY0, contentX1, bandBottom)
)

// ClockRect returns the clock region in screen coordinates. The clock is
// drawn only inside it, so it can be partially refreshed on its own.
// Min.Y and Max.Y are multiples of 8 because in portrait the panel's
// native X axis is the portrait Y axis (the panel adapter rotates, per
// ADR-0004), and a partial refresh needs byte-aligned native X edges. The
// left and right edges are free. The weather block lies to its right and
// never inside it.
func ClockRect() image.Rectangle {
	return image.Rect(clockX0, clockY0, clockX1, clockY1)
}

// headingBaselineY returns the heading baseline of the section whose rule
// is at top.
func headingBaselineY(top int) int {
	return top + headingOffset
}

// firstRowBaselineY returns the baseline of the first row in the section
// whose rule is at top.
func firstRowBaselineY(top int) int {
	return headingBaselineY(top) + rowHeight
}
