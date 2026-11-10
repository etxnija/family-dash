package eink

import "image"

// Layout of the 800x480 frame, in pixels. Every number the renderer uses
// to place things lives here so the layout can be tweaked in one spot.
// Y values ending in "BaselineY" are text baselines; the rest are edges.
const (
	// ScreenWidth is the panel width.
	ScreenWidth = 800
	// ScreenHeight is the panel height.
	ScreenHeight = 480

	// borderThickness is the black frame drawn along the screen edges.
	borderThickness = 4
	// outerMargin is the white space between the screen edge and any
	// content; the part inside the border stays white.
	outerMargin = 16

	// contentTop is where the header starts.
	contentTop = outerMargin
	// contentBottom is the lowest Y any content may reach.
	contentBottom = ScreenHeight - outerMargin

	// dateBaselineY is the baseline of the date and the week label.
	dateBaselineY = 44
	// weekLabelGap is the space between the date and the week label.
	weekLabelGap = 12
	// ownersBaselineY is the baseline of the owners, right-aligned.
	ownersBaselineY = 44
	// headerRuleY is the top of the rule under the header.
	headerRuleY = 56
	// headerRuleThickness is the height of the rule under the header.
	headerRuleThickness = 2
	// bodyTop is the first Y below the header rule.
	bodyTop = headerRuleY + headerRuleThickness

	// leftColumnX0 and leftColumnX1 bound the left column (width 272).
	leftColumnX0 = outerMargin
	leftColumnX1 = 288
	// rightColumnX0 and rightColumnX1 bound the right column. The gutter
	// between the columns (288 to 312) is kept white, with no rule.
	rightColumnX0 = 312
	rightColumnX1 = ScreenWidth - outerMargin

	// clockFontSize is the bold clock face size.
	clockFontSize = 72
	// dateFontSize is the bold date face size.
	dateFontSize = 28
	// headingFontSize is the bold section heading face size.
	headingFontSize = 16
	// bodyFontSize is the regular body text face size (minimum 14).
	bodyFontSize = 15
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
	sectionGap = 12

	// maxEventRows, maxHomeworkRows, maxUpcomingRows and maxLunchRows
	// limit each section; extra rows collapse into a "+N till" row.
	maxEventRows    = 5
	maxHomeworkRows = 4
	maxUpcomingRows = 5
	maxLunchRows    = 4

	// clockX0, clockX1, clockY0 and clockY1 bound the clock rectangle.
	// The X edges are multiples of 8 so the panel can partially refresh it.
	clockX0 = 16
	clockX1 = 272
	clockY0 = bodyTop + 6
	clockY1 = clockY0 + 88

	// temperatureBaselineY is the baseline of the large temperature.
	temperatureBaselineY = clockY1 + 34
	// weatherLinesBaselineY is the baseline of the first weather line.
	weatherLinesBaselineY = temperatureBaselineY + 24
	// maxWeatherLines is the number of weather line slots: the view's four
	// lines plus one for the low/high line, which wraps at " · ".
	maxWeatherLines = 5

	// lunchTop is the Y of the rule above SKOLLUNCH.
	lunchTop = weatherLinesBaselineY + (maxWeatherLines-1)*rowHeight + sectionGap

	// todayTop, homeworkTop and upcomingTop are the Y of the rule above
	// each right-column section. Each section has fixed room for its
	// maximum row count, so positions don't move with the content.
	todayTop    = bodyTop + 6
	homeworkTop = todayTop + headingOffset + maxEventRows*rowHeight + sectionGap
	upcomingTop = homeworkTop + headingOffset + maxHomeworkRows*rowHeight + sectionGap
)

var (
	screenRect      = image.Rect(0, 0, ScreenWidth, ScreenHeight)
	headerRect      = image.Rect(outerMargin, contentTop, ScreenWidth-outerMargin, headerRuleY)
	leftColumnRect  = image.Rect(leftColumnX0, bodyTop, leftColumnX1, contentBottom)
	rightColumnRect = image.Rect(rightColumnX0, bodyTop, rightColumnX1, contentBottom)
)

// ClockRect returns the clock region in screen coordinates. The clock is
// drawn only inside it, so it can be partially refreshed on its own.
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
