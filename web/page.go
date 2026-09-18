// Package web is the HTML display adapter: it renders dashboard.Event data
// as a self-refreshing HTML page.
package web

import (
	"embed"
	"html/template"
	"io"
	"time"

	dashboard "family-dashboard"
)

//go:embed templates/page.html.tmpl
var templateFS embed.FS

var pageTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"ownerColor": ownerColor,
}).ParseFS(templateFS, "templates/page.html.tmpl"))

// EventRow is the flattened, display-ready view of a dashboard.Event.
type EventRow struct {
	Time      string
	TimeRange string
	Summary   string
	Location  string
	Owner     string
}

// PageData holds everything the page template needs to render.
type PageData struct {
	Weekday   string
	DateLabel string
	Events    []EventRow
	Owners    []string
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

// NewPageData builds a PageData for events, owners, and now, converting all
// times to Europe/Stockholm for display.
func NewPageData(events []dashboard.Event, owners []string, now time.Time) (PageData, error) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		return PageData{}, err
	}

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
			Time:      timeStr,
			TimeRange: timeRange,
			Summary:   e.Summary,
			Location:  e.Location,
			Owner:     e.Owner,
		})
	}

	localNow := now.In(loc)
	return PageData{
		Weekday:   localNow.Format("Monday"),
		DateLabel: localNow.Format("2 January"),
		Events:    rows,
		Owners:    owners,
	}, nil
}

// Render executes the page template for data into w.
func Render(w io.Writer, data PageData) error {
	return pageTemplate.ExecuteTemplate(w, "page.html.tmpl", data)
}
