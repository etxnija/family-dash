// Package eink is the e-paper display adapter: it shapes dashboard data
// into a display-ready view and renders it as a portrait 480x800 frame for
// the 1-bit e-paper panel. The renderer never rotates; turning the frame
// to the panel's native landscape orientation is the panel adapter's job
// (ADR-0004).
package eink

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	dashboard "family-dashboard"
)

// Section headings, drawn uppercase on the panel.
const (
	headingLunch    = "SKOLLUNCH"
	headingToday    = "IDAG"
	headingHomework = "LÄXOR OCH PROV"
	headingUpcoming = "KOMMANDE DAGAR"
)

// Empty-state texts, drawn when a section has nothing to show.
const (
	emptyLunch    = "Ingen lunchmeny idag."
	emptyToday    = "Inget planerat idag."
	emptyHomework = "Inga läxor den här veckan."
	emptyUpcoming = "Inga aktiviteter de kommande dagarna."
	emptyWeather  = "ingen väderkälla"

	noTemperature = "—°"
	allDay        = "heldag"

	// Feed notice texts: the prefixes of its missing and stale parts, the
	// separator between parts and owners, and the stale age formats.
	feedMissingPrefix   = "Kalender saknas: "
	feedStalePrefix     = "Kalender gammal: "
	feedPartSeparator   = " · "
	feedOwnerSeparator  = ", "
	feedAgeMinuteFormat = "%d min"
	feedAgeHourFormat   = "%d h"
)

// View holds everything the e-paper renderer draws, as ready-to-draw
// Swedish strings; the renderer does no translation or formatting.
type View struct {
	DateLabel string
	WeekLabel string
	Owners    string
	Clock     string
	Weather   WeatherView
	Lunch     []string
	Events    []EventRow
	Homework  []HomeworkGroup
	Upcoming  []UpcomingRow
	// FeedNotice is one ready-to-draw row about failing calendar feeds,
	// drawn at the bottom of the frame; empty when every feed is fine.
	FeedNotice string
}

// WeatherView is the weather panel: a large temperature and the lines
// drawn beneath it.
type WeatherView struct {
	Temperature string
	Lines       []string
}

// EventRow is one of today's events.
type EventRow struct {
	Time     string
	Owner    string
	Summary  string
	Location string
}

// HomeworkRow is a single homework item.
type HomeworkRow struct {
	Summary  string
	DayLabel string
	IsProv   bool
}

// HomeworkGroup collects the homework items due for a single owner.
// Initial is the owner's uppercased first letter, drawn as the row prefix.
type HomeworkGroup struct {
	Owner   string
	Initial string
	Items   []HomeworkRow
}

// UpcomingRow is one event in the coming days.
type UpcomingRow struct {
	Weekday  string
	Time     string
	Owner    string
	Summary  string
	Location string
}

// NewView builds a View for events, homeworkEvents, upcomingEvents, owners,
// lunch, weather, problems, and now, converting all times to
// Europe/Stockholm.
func NewView(events, homeworkEvents, upcomingEvents []dashboard.Event, owners []string, lunch []string, weather *dashboard.WeatherReading, problems []dashboard.FeedProblem, now time.Time) (View, error) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		return View{}, err
	}

	localNow := now.In(loc)
	_, week := localNow.ISOWeek()

	return View{
		DateLabel: dateLabel(localNow),
		WeekLabel: fmt.Sprintf("v.%d", week),
		Owners:    strings.Join(owners, " · "),
		Clock:     clockText(now, loc),
		Weather:   buildWeatherView(weather),
		Lunch:     lunch,
		Events:    buildEventRows(events, loc),
		Homework:  buildHomeworkGroups(homeworkEvents, owners, localNow, loc),
		Upcoming:  buildUpcomingRows(upcomingEvents, loc),

		FeedNotice: buildFeedNotice(problems, now),
	}, nil
}

// clockText formats t as the "15:04" clock in loc.
func clockText(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("15:04")
}

// dateLabel formats t as e.g. "Onsdag 16 september".
func dateLabel(t time.Time) string {
	return fmt.Sprintf("%s %d %s", svWeekdays[t.Weekday()], t.Day(), svMonths[t.Month()])
}

// buildWeatherView returns the weather panel for weather, or a placeholder
// panel when weather is nil.
func buildWeatherView(weather *dashboard.WeatherReading) WeatherView {
	if weather == nil {
		return WeatherView{Temperature: noTemperature, Lines: []string{emptyWeather}}
	}

	sky := "Mulet"
	if weather.Sunny {
		sky = "Sol"
	}
	precipitation := "torrt idag"
	if weather.PrecipitationExpected {
		precipitation = "regn idag"
	}

	return WeatherView{
		Temperature: temperatureDisplay(weather.Temperature),
		Lines: []string{
			"SMHI",
			"Lägst idag: " + temperatureDisplay(weather.LowTemperature) + " · Högst idag: " + temperatureDisplay(weather.HighTemperature),
			sky,
			precipitation,
		},
	}
}

// buildEventRows flattens events into display-ready rows, converting times
// to loc.
func buildEventRows(events []dashboard.Event, loc *time.Location) []EventRow {
	rows := make([]EventRow, 0, len(events))
	for _, e := range events {
		rows = append(rows, EventRow{
			Time:     eventTime(e, loc),
			Owner:    e.Owner,
			Summary:  e.Summary,
			Location: e.Location,
		})
	}
	return rows
}

// buildHomeworkGroups groups homeworkEvents by owner, in owners order, with
// each owner's items sorted by Start and labeled relative to localNow.
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
				DayLabel: dayLabel(offset, eventLocal),
				IsProv:   strings.Contains(strings.ToLower(e.Description), "prov"),
			})
		}
		homework = append(homework, HomeworkGroup{Owner: owner, Initial: initial(owner), Items: items})
	}
	return homework
}

// buildUpcomingRows flattens events into display-ready rows sorted by Start,
// converting times to loc.
func buildUpcomingRows(events []dashboard.Event, loc *time.Location) []UpcomingRow {
	sorted := make([]dashboard.Event, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Start.Before(sorted[j].Start)
	})

	rows := make([]UpcomingRow, 0, len(sorted))
	for _, e := range sorted {
		rows = append(rows, UpcomingRow{
			Weekday:  svWeekdayAbbrevs[e.Start.In(loc).Weekday()],
			Time:     eventTime(e, loc),
			Owner:    e.Owner,
			Summary:  e.Summary,
			Location: e.Location,
		})
	}
	return rows
}

// buildFeedNotice returns the feed notice for problems as of now, e.g.
// "Kalender saknas: K · Kalender gammal: L (3 h)", or "" when there are no
// problems. Each part lists every owner's initial once, in first-seen
// order; a stale owner shows the age of its oldest stale source. now need
// not be in Stockholm time: only durations (now - LastSuccess) are used.
func buildFeedNotice(problems []dashboard.FeedProblem, now time.Time) string {
	var missing, stale []string
	oldest := make(map[string]time.Duration)
	for _, p := range problems {
		if !p.Stale {
			if !slices.Contains(missing, p.Owner) {
				missing = append(missing, p.Owner)
			}
			continue
		}
		age := now.Sub(p.LastSuccess)
		if prev, seen := oldest[p.Owner]; !seen {
			stale = append(stale, p.Owner)
			oldest[p.Owner] = age
		} else {
			oldest[p.Owner] = max(prev, age)
		}
	}

	var parts []string
	if len(missing) > 0 {
		initials := make([]string, 0, len(missing))
		for _, owner := range missing {
			initials = append(initials, initial(owner))
		}
		parts = append(parts, feedMissingPrefix+strings.Join(initials, feedOwnerSeparator))
	}
	if len(stale) > 0 {
		ages := make([]string, 0, len(stale))
		for _, owner := range stale {
			ages = append(ages, initial(owner)+" ("+feedAge(oldest[owner])+")")
		}
		parts = append(parts, feedStalePrefix+strings.Join(ages, feedOwnerSeparator))
	}
	return strings.Join(parts, feedPartSeparator)
}

// feedAge formats age rounded down: "N min" (at least 1) under one hour,
// otherwise "N h".
func feedAge(age time.Duration) string {
	if age < time.Hour {
		return fmt.Sprintf(feedAgeMinuteFormat, max(int(age/time.Minute), 1))
	}
	return fmt.Sprintf(feedAgeHourFormat, int(age/time.Hour))
}

// eventTime returns e's local start time as "15:04", or "heldag" for an
// all-day event.
func eventTime(e dashboard.Event, loc *time.Location) string {
	if e.AllDay {
		return allDay
	}
	return e.Start.In(loc).Format("15:04")
}

// dayLabel returns the Swedish label for a homework item due offset days
// from today: "idag", "imorgon", or the lowercase weekday of eventDate.
func dayLabel(offset int, eventDate time.Time) string {
	switch offset {
	case 0:
		return "idag"
	case 1:
		return "imorgon"
	default:
		return strings.ToLower(svWeekdays[eventDate.Weekday()])
	}
}

// initial returns the first rune of s, uppercased, as a string. It returns
// "" for an empty input.
func initial(s string) string {
	for _, r := range s {
		return strings.ToUpper(string(r))
	}
	return ""
}

// temperatureDisplay formats temperature as a rounded whole-degree string
// (e.g. "11°").
func temperatureDisplay(temperature float64) string {
	return fmt.Sprintf("%d°", int(math.Round(temperature)))
}

// svWeekdays maps weekdays to their Swedish names.
var svWeekdays = map[time.Weekday]string{
	time.Monday:    "Måndag",
	time.Tuesday:   "Tisdag",
	time.Wednesday: "Onsdag",
	time.Thursday:  "Torsdag",
	time.Friday:    "Fredag",
	time.Saturday:  "Lördag",
	time.Sunday:    "Söndag",
}

// svWeekdayAbbrevs maps weekdays to their abbreviated Swedish names.
var svWeekdayAbbrevs = map[time.Weekday]string{
	time.Monday:    "Mån",
	time.Tuesday:   "Tis",
	time.Wednesday: "Ons",
	time.Thursday:  "Tors",
	time.Friday:    "Fre",
	time.Saturday:  "Lör",
	time.Sunday:    "Sön",
}

// svMonths maps months to their lowercase Swedish names.
var svMonths = map[time.Month]string{
	time.January:   "januari",
	time.February:  "februari",
	time.March:     "mars",
	time.April:     "april",
	time.May:       "maj",
	time.June:      "juni",
	time.July:      "juli",
	time.August:    "augusti",
	time.September: "september",
	time.October:   "oktober",
	time.November:  "november",
	time.December:  "december",
}
