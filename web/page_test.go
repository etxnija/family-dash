package web

import (
	"bytes"
	"strings"
	"testing"
	"time"

	dashboard "family-dashboard"
)

type pageDataTestCase struct {
	name            string
	events          []dashboard.Event
	now             time.Time
	wantTime        string
	wantTimeRange   string
	wantWeekday     string
	wantDateLabel   string
	wantDescription string
}

func newPageDataTestCases() []pageDataTestCase {
	at := func(hour, min int) time.Time { return time.Date(2026, 9, 18, hour, min, 0, 0, time.UTC) }
	return []pageDataTestCase{
		{
			name:          "event with end time",
			events:        []dashboard.Event{{Summary: "Football practice", Start: at(15, 0), End: at(16, 30)}},
			now:           at(12, 0),
			wantTime:      "17:00",
			wantTimeRange: "17:00–18:30",
			wantWeekday:   "Friday",
			wantDateLabel: "18 September",
		},
		{
			name:          "event with zero end time",
			events:        []dashboard.Event{{Summary: "Dentist", Start: at(8, 0)}},
			now:           at(6, 0),
			wantTime:      "10:00",
			wantTimeRange: "10:00",
			wantWeekday:   "Friday",
			wantDateLabel: "18 September",
		},
		{
			name:          "all-day event",
			events:        []dashboard.Event{{Summary: "Helgcup", AllDay: true, Start: at(0, 0)}},
			now:           at(6, 0),
			wantTime:      "All day",
			wantTimeRange: "All day",
			wantWeekday:   "Friday",
			wantDateLabel: "18 September",
		},
		{
			name:            "event with description",
			events:          []dashboard.Event{{Summary: "Dentist", Description: "Bring insurance card", Start: at(8, 0)}},
			now:             at(6, 0),
			wantTime:        "10:00",
			wantTimeRange:   "10:00",
			wantWeekday:     "Friday",
			wantDateLabel:   "18 September",
			wantDescription: "Bring insurance card",
		},
	}
}

func TestNewPageData(t *testing.T) {
	for _, tt := range newPageDataTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			data, err := NewPageData(tt.events, nil, nil, []string{"Kaj", "Lillian"}, nil, nil, tt.now)
			if err != nil {
				t.Fatalf("NewPageData() error = %v", err)
			}
			if len(data.Events) != 1 {
				t.Fatalf("len(data.Events) = %d, want 1", len(data.Events))
			}
			if got := data.Events[0].Time; got != tt.wantTime {
				t.Errorf("Time = %q, want %q", got, tt.wantTime)
			}
			if got := data.Events[0].TimeRange; got != tt.wantTimeRange {
				t.Errorf("TimeRange = %q, want %q", got, tt.wantTimeRange)
			}
			if data.Weekday != tt.wantWeekday {
				t.Errorf("Weekday = %q, want %q", data.Weekday, tt.wantWeekday)
			}
			if data.DateLabel != tt.wantDateLabel {
				t.Errorf("DateLabel = %q, want %q", data.DateLabel, tt.wantDateLabel)
			}
			if got := data.Events[0].Description; got != tt.wantDescription {
				t.Errorf("Description = %q, want %q", got, tt.wantDescription)
			}
		})
	}
}

func TestNewPageData_WeekNumberAndCurrentTime(t *testing.T) {
	now := time.Date(2026, 9, 18, 7, 42, 0, 0, time.UTC)

	data, err := NewPageData(nil, nil, nil, []string{"Kaj", "Lillian"}, nil, nil, now)
	if err != nil {
		t.Fatalf("NewPageData() error = %v", err)
	}
	if data.WeekNumber != 38 {
		t.Errorf("WeekNumber = %d, want 38", data.WeekNumber)
	}
	if data.CurrentTime != "09:42" {
		t.Errorf("CurrentTime = %q, want %q", data.CurrentTime, "09:42")
	}
}

func assertHomeworkGroups(t *testing.T, got, want []HomeworkGroup) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Homework = %+v, want %+v", got, want)
	}
	for i, wantGroup := range want {
		gotGroup := got[i]
		if gotGroup.Owner != wantGroup.Owner {
			t.Errorf("Homework[%d].Owner = %q, want %q", i, gotGroup.Owner, wantGroup.Owner)
		}
		if len(gotGroup.Items) != len(wantGroup.Items) {
			t.Fatalf("Homework[%d].Items = %+v, want %+v", i, gotGroup.Items, wantGroup.Items)
		}
		for j, wantItem := range wantGroup.Items {
			gotItem := gotGroup.Items[j]
			if gotItem != wantItem {
				t.Errorf("Homework[%d].Items[%d] = %+v, want %+v", i, j, gotItem, wantItem)
			}
		}
	}
}

type homeworkTestCase struct {
	name           string
	homeworkEvents []dashboard.Event
	owners         []string
	wantGroups     []HomeworkGroup
}

func newHomeworkTestCases(loc *time.Location) []homeworkTestCase {
	homeworkEvent := func(owner, summary, description string, dayOffset int) dashboard.Event {
		start := time.Date(2026, 9, 17+dayOffset, 9, 0, 0, 0, loc)
		return dashboard.Event{Owner: owner, Summary: summary, Description: description, Start: start}
	}
	singleGroup := func(owner, summary, dayLabel string, isToday, isProv bool) []HomeworkGroup {
		return []HomeworkGroup{{Owner: owner, Items: []HomeworkRow{{Summary: summary, DayLabel: dayLabel, IsToday: isToday, IsProv: isProv}}}}
	}

	return []homeworkTestCase{
		{name: "day label is Due today for an event starting today", homeworkEvents: []dashboard.Event{homeworkEvent("Kaj", "Math worksheet", "", 0)}, owners: []string{"Kaj"}, wantGroups: singleGroup("Kaj", "Math worksheet", "Due today", true, false)},
		{name: "day label is Due tomorrow for an event starting tomorrow", homeworkEvents: []dashboard.Event{homeworkEvent("Kaj", "Read ch. 4", "", 1)}, owners: []string{"Kaj"}, wantGroups: singleGroup("Kaj", "Read ch. 4", "Due tomorrow", false, false)},
		{name: "day label is the weekday name for day+2 through day+6", homeworkEvents: []dashboard.Event{homeworkEvent("Kaj", "Spelling words", "", 6)}, owners: []string{"Kaj"}, wantGroups: singleGroup("Kaj", "Spelling words", "Wednesday", false, false)},
		{name: "prov event has IsProv true", homeworkEvents: []dashboard.Event{homeworkEvent("Kaj", "Matteprov", "Prov i matematik", 0)}, owners: []string{"Kaj"}, wantGroups: singleGroup("Kaj", "Matteprov", "Due today", true, true)},
		{name: "owners with no homework events produce no group entry", homeworkEvents: []dashboard.Event{homeworkEvent("Kaj", "Math worksheet", "", 0)}, owners: []string{"Kaj", "Lillian"}, wantGroups: singleGroup("Kaj", "Math worksheet", "Due today", true, false)},
		{
			name: "grouping respects the owners slice order",
			homeworkEvents: []dashboard.Event{
				homeworkEvent("Lillian", "Spelling words", "", 0),
				homeworkEvent("Kaj", "Math worksheet", "", 0),
			},
			owners: []string{"Lillian", "Kaj"},
			wantGroups: []HomeworkGroup{
				{Owner: "Lillian", Items: []HomeworkRow{{Summary: "Spelling words", DayLabel: "Due today", IsToday: true, IsProv: false}}},
				{Owner: "Kaj", Items: []HomeworkRow{{Summary: "Math worksheet", DayLabel: "Due today", IsToday: true, IsProv: false}}},
			},
		},
	}
}

func TestNewPageData_Homework(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatalf("time.LoadLocation() error = %v", err)
	}
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, loc)

	for _, tt := range newHomeworkTestCases(loc) {
		t.Run(tt.name, func(t *testing.T) {
			data, err := NewPageData(nil, tt.homeworkEvents, nil, tt.owners, nil, nil, now)
			if err != nil {
				t.Fatalf("NewPageData() error = %v", err)
			}
			assertHomeworkGroups(t, data.Homework, tt.wantGroups)
		})
	}
}

func assertUpcomingRows(t *testing.T, got, want []UpcomingRow) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Upcoming = %+v, want %+v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("Upcoming[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestNewPageData_Upcoming(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatalf("time.LoadLocation() error = %v", err)
	}
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, loc)

	tests := []struct {
		name           string
		upcomingEvents []dashboard.Event
		want           []UpcomingRow
	}{
		{
			name: "event with end time produces correct fields",
			upcomingEvents: []dashboard.Event{
				{Owner: "Lillian", Summary: "Bortamatch", Location: "Arena", Start: time.Date(2026, 9, 23, 10, 0, 0, 0, loc), End: time.Date(2026, 9, 23, 11, 0, 0, 0, loc)},
			},
			want: []UpcomingRow{
				{Weekday: "Wednesday", Time: "10:00", Owner: "Lillian", Summary: "Bortamatch", Location: "Arena"},
			},
		},
		{
			name: "all-day event produces Time All day",
			upcomingEvents: []dashboard.Event{
				{Owner: "Kaj", Summary: "Innebandyturnering", AllDay: true, Start: time.Date(2026, 9, 24, 0, 0, 0, 0, loc)},
			},
			want: []UpcomingRow{
				{Weekday: "Thursday", Time: "All day", Owner: "Kaj", Summary: "Innebandyturnering"},
			},
		},
		{
			name: "events are sorted by Start",
			upcomingEvents: []dashboard.Event{
				{Owner: "Kaj", Summary: "Second", Start: time.Date(2026, 9, 24, 9, 0, 0, 0, loc)},
				{Owner: "Lillian", Summary: "First", Start: time.Date(2026, 9, 23, 9, 0, 0, 0, loc)},
			},
			want: []UpcomingRow{
				{Weekday: "Wednesday", Time: "09:00", Owner: "Lillian", Summary: "First"},
				{Weekday: "Thursday", Time: "09:00", Owner: "Kaj", Summary: "Second"},
			},
		},
		{
			name:           "empty input produces empty Upcoming slice",
			upcomingEvents: nil,
			want:           nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := NewPageData(nil, nil, tt.upcomingEvents, []string{"Kaj", "Lillian"}, nil, nil, now)
			if err != nil {
				t.Fatalf("NewPageData() error = %v", err)
			}
			assertUpcomingRows(t, data.Upcoming, tt.want)
		})
	}
}

func TestNewPageData_Lunch(t *testing.T) {
	now := time.Date(2026, 9, 18, 7, 42, 0, 0, time.UTC)
	lunch := []string{"Fiskgratäng med potatis", "Vegetariska tacos"}

	data, err := NewPageData(nil, nil, nil, []string{"Kaj", "Lillian"}, lunch, nil, now)
	if err != nil {
		t.Fatalf("NewPageData() error = %v", err)
	}
	if len(data.Lunch) != len(lunch) {
		t.Fatalf("Lunch = %+v, want %+v", data.Lunch, lunch)
	}
	for i, want := range lunch {
		if data.Lunch[i] != want {
			t.Errorf("Lunch[%d] = %q, want %q", i, data.Lunch[i], want)
		}
	}
}

func TestNewPageData_Weather(t *testing.T) {
	now := time.Date(2026, 9, 18, 7, 42, 0, 0, time.UTC)

	t.Run("non-nil weather", func(t *testing.T) {
		weather := &dashboard.WeatherReading{Temperature: 10.9, LowTemperature: 8.5, HighTemperature: 13.2, Sunny: true, PrecipitationExpected: false}
		data, err := NewPageData(nil, nil, nil, []string{"Kaj", "Lillian"}, nil, weather, now)
		if err != nil {
			t.Fatalf("NewPageData() error = %v", err)
		}
		if data.Weather == nil {
			t.Fatal("Weather = nil, want non-nil")
		}
		if *data.Weather != *weather {
			t.Errorf("*Weather = %+v, want %+v", *data.Weather, *weather)
		}
	})

	t.Run("nil weather", func(t *testing.T) {
		data, err := NewPageData(nil, nil, nil, []string{"Kaj", "Lillian"}, nil, nil, now)
		if err != nil {
			t.Fatalf("NewPageData() error = %v", err)
		}
		if data.Weather != nil {
			t.Errorf("Weather = %v, want nil", data.Weather)
		}
	})
}

func TestSvWeekday(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Monday", "Måndag"},
		{"Tuesday", "Tisdag"},
		{"Wednesday", "Onsdag"},
		{"Thursday", "Torsdag"},
		{"Friday", "Fredag"},
		{"Saturday", "Lördag"},
		{"Sunday", "Söndag"},
		{"Someday", "Someday"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := svWeekday(tt.in); got != tt.want {
				t.Errorf("svWeekday(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSvWeekdayAbbrev(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Monday", "Mån"},
		{"Tuesday", "Tis"},
		{"Wednesday", "Ons"},
		{"Thursday", "Tors"},
		{"Friday", "Fre"},
		{"Saturday", "Lör"},
		{"Sunday", "Sön"},
		{"Someday", "Someday"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := svWeekdayAbbrev(tt.in); got != tt.want {
				t.Errorf("svWeekdayAbbrev(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSvDateLabel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"18 September", "18 september"},
		{"1 January", "1 januari"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := svDateLabel(tt.in); got != tt.want {
				t.Errorf("svDateLabel(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSvDayLabel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Due today", "idag"},
		{"Due tomorrow", "imorgon"},
		{"Wednesday", "onsdag"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := svDayLabel(tt.in); got != tt.want {
				t.Errorf("svDayLabel(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSvTime(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"All day", "heldag"},
		{"15:30", "15:30"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := svTime(tt.in); got != tt.want {
				t.Errorf("svTime(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestOwnerInitial(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Kaj", "K"},
		{"Lillian", "L"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := ownerInitial(tt.in); got != tt.want {
				t.Errorf("ownerInitial(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name string
		data PageData
		want []string
	}{
		{
			name: "non-empty events",
			data: PageData{
				Weekday:   "Friday",
				DateLabel: "18 September",
				Events: []EventRow{
					{Time: "17:00", TimeRange: "17:00–18:30", Summary: "Football practice", Location: "Arena", Owner: "Kaj"},
				},
				Owners: []string{"Kaj", "Lillian"},
			},
			want: []string{"Football practice", "Kaj"},
		},
		{
			name: "empty events",
			data: PageData{
				Weekday:   "Friday",
				DateLabel: "18 September",
				Events:    nil,
				Owners:    []string{"Kaj", "Lillian"},
			},
			want: []string{"Inget planerat idag."},
		},
		{
			name: "weather with temperature",
			data: PageData{
				Weekday:   "Friday",
				DateLabel: "18 September",
				Owners:    []string{"Kaj", "Lillian"},
				Weather:   &dashboard.WeatherReading{Temperature: 10.9, LowTemperature: 8.5, HighTemperature: 13.2, Sunny: true},
			},
			want: []string{"11°", "SMHI", "Sol", "torrt idag", "H&ouml;gst idag: 13°"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := Render(&buf, tt.data); err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			out := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q\ngot: %s", want, out)
				}
			}
		})
	}
}
