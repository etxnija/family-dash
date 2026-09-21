// Package web is the HTML display adapter: it renders dashboard.Event data
// as a self-refreshing HTML page.
package web

import (
	"embed"
	"html/template"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	dashboard "family-dashboard"
)

//go:embed templates/page.html.tmpl
var templateFS embed.FS

var pageTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"ownerColor":   ownerColor,
	"svWeekday":    svWeekday,
	"svDateLabel":  svDateLabel,
	"svDayLabel":   svDayLabel,
	"svTime":       svTime,
	"ownerInitial": ownerInitial,
}).ParseFS(templateFS, "templates/page.html.tmpl"))

// EventRow is the flattened, display-ready view of a dashboard.Event.
type EventRow struct {
	Time        string
	TimeRange   string
	Summary     string
	Location    string
	Description string
	Owner       string
}

// HomeworkRow is the flattened, display-ready view of a homework
// dashboard.Event.
type HomeworkRow struct {
	Summary  string
	DayLabel string
	IsToday  bool
}

// HomeworkGroup collects the homework items due for a single owner.
type HomeworkGroup struct {
	Owner string
	Items []HomeworkRow
}

// PageData holds everything the page template needs to render.
type PageData struct {
	Weekday     string
	DateLabel   string
	WeekNumber  int
	CurrentTime string
	Events      []EventRow
	Owners      []string
	Homework    []HomeworkGroup
}

// ownerColors maps a known owner name to its CSS custom-property name.
var ownerColors = map[string]string{
	"Kaj":     "--kaj",
	"Lillian": "--lillian",
}

// ownerColor returns the CSS custom-property name for name, or "--family"
// if name has no dedicated color.
func ownerColor(name string) string {
	if color, ok := ownerColors[name]; ok {
		return color
	}
	return "--family"
}

// NewPageData builds a PageData for events, homeworkEvents, owners, and now,
// converting all times to Europe/Stockholm for display.
func NewPageData(events []dashboard.Event, homeworkEvents []dashboard.Event, owners []string, now time.Time) (PageData, error) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		return PageData{}, err
	}

	localNow := now.In(loc)
	_, week := localNow.ISOWeek()

	return PageData{
		Weekday:     localNow.Format("Monday"),
		DateLabel:   localNow.Format("2 January"),
		WeekNumber:  week,
		CurrentTime: localNow.Format("15:04"),
		Events:      buildEventRows(events, loc),
		Owners:      owners,
		Homework:    buildHomeworkGroups(homeworkEvents, owners, localNow, loc),
	}, nil
}

// buildEventRows flattens events into display-ready rows, converting times
// to loc.
func buildEventRows(events []dashboard.Event, loc *time.Location) []EventRow {
	rows := make([]EventRow, 0, len(events))
	for _, e := range events {
		var timeStr, timeRange string
		if e.AllDay {
			timeStr = "All day"
			timeRange = "All day"
		} else {
			start := e.Start.In(loc)
			timeStr = start.Format("15:04")
			timeRange = timeStr
			if !e.End.IsZero() {
				timeRange += "–" + e.End.In(loc).Format("15:04")
			}
		}
		rows = append(rows, EventRow{
			Time:        timeStr,
			TimeRange:   timeRange,
			Summary:     e.Summary,
			Location:    e.Location,
			Description: e.Description,
			Owner:       e.Owner,
		})
	}
	return rows
}

// buildHomeworkGroups groups homeworkEvents by owner, in owners order, with
// each item's day label computed relative to localNow.
func buildHomeworkGroups(homeworkEvents []dashboard.Event, owners []string, localNow time.Time, loc *time.Location) []HomeworkGroup {
	eventsByOwner := make(map[string][]dashboard.Event, len(owners))
	for _, e := range homeworkEvents {
		eventsByOwner[e.Owner] = append(eventsByOwner[e.Owner], e)
	}

	year, month, day := localNow.Date()
	todayNoon := time.Date(year, month, day, 12, 0, 0, 0, loc)

	var homework []HomeworkGroup
	for _, owner := range owners {
		ownerEvents := eventsByOwner[owner]
		if len(ownerEvents) == 0 {
			continue
		}
		sort.Slice(ownerEvents, func(i, j int) bool {
			return ownerEvents[i].Start.Before(ownerEvents[j].Start)
		})

		items := make([]HomeworkRow, 0, len(ownerEvents))
		for _, e := range ownerEvents {
			eventLocal := e.Start.In(loc)
			ey, em, ed := eventLocal.Date()
			eventNoon := time.Date(ey, em, ed, 12, 0, 0, 0, loc)
			offset := int(math.Round(eventNoon.Sub(todayNoon).Hours() / 24))

			items = append(items, HomeworkRow{
				Summary:  e.Summary,
				DayLabel: homeworkDayLabel(offset, eventLocal),
				IsToday:  offset == 0,
			})
		}
		homework = append(homework, HomeworkGroup{Owner: owner, Items: items})
	}
	return homework
}

// homeworkDayLabel returns the display label for a homework item due offset
// days from today: "Due today", "Due tomorrow", or the Stockholm weekday
// name of eventDate for offset 2 through 6.
func homeworkDayLabel(offset int, eventDate time.Time) string {
	switch offset {
	case 0:
		return "Due today"
	case 1:
		return "Due tomorrow"
	default:
		return eventDate.Weekday().String()
	}
}

// svWeekdays maps English weekday names (as produced by time.Format
// "Monday") to their Swedish equivalents.
var svWeekdays = map[string]string{
	"Monday":    "Måndag",
	"Tuesday":   "Tisdag",
	"Wednesday": "Onsdag",
	"Thursday":  "Torsdag",
	"Friday":    "Fredag",
	"Saturday":  "Lördag",
	"Sunday":    "Söndag",
}

// svMonths maps English month names (as produced by time.Format "January")
// to their lowercase Swedish equivalents.
var svMonths = map[string]string{
	"January":   "januari",
	"February":  "februari",
	"March":     "mars",
	"April":     "april",
	"May":       "maj",
	"June":      "juni",
	"July":      "juli",
	"August":    "augusti",
	"September": "september",
	"October":   "oktober",
	"November":  "november",
	"December":  "december",
}

// svWeekday translates an English weekday name to Swedish, passing through
// any string it doesn't recognize.
func svWeekday(s string) string {
	if sv, ok := svWeekdays[s]; ok {
		return sv
	}
	return s
}

// svDateLabel translates the month name within a DateLabel string (e.g.
// "18 September") to lowercase Swedish.
func svDateLabel(s string) string {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return s
	}
	month, ok := svMonths[parts[1]]
	if !ok {
		return s
	}
	return parts[0] + " " + month
}

// svDayLabel translates a homework day label to Swedish: "Due today" and
// "Due tomorrow" to fixed phrases, and any weekday name (the fallback from
// homeworkDayLabel) to its lowercase Swedish equivalent.
func svDayLabel(s string) string {
	switch s {
	case "Due today":
		return "idag"
	case "Due tomorrow":
		return "imorgon"
	}
	if sv, ok := svWeekdays[s]; ok {
		return strings.ToLower(sv)
	}
	return s
}

// svTime translates the "All day" event time label to Swedish, passing
// through any other value (e.g. "15:30") unchanged.
func svTime(s string) string {
	if s == "All day" {
		return "heldag"
	}
	return s
}

// ownerInitial returns the first rune of s, uppercased, as a string. It
// returns "" for an empty input.
func ownerInitial(s string) string {
	for _, r := range s {
		return strings.ToUpper(string(r))
	}
	return ""
}

// Render executes the page template for data into w.
func Render(w io.Writer, data PageData) error {
	return pageTemplate.ExecuteTemplate(w, "page.html.tmpl", data)
}
