package web

import (
	"bytes"
	"strings"
	"testing"
	"time"

	dashboard "family-dashboard"
)

type pageDataTestCase struct {
	name          string
	events        []dashboard.Event
	now           time.Time
	wantTime        string
	wantTimeRange   string
	wantWeekday     string
	wantDateLabel   string
	wantDescription string
}

func newPageDataTestCases() []pageDataTestCase {
	return []pageDataTestCase{
		{
			name: "event with end time",
			events: []dashboard.Event{
				{
					Summary: "Football practice",
					Start:   time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC),
					End:     time.Date(2026, 9, 18, 16, 30, 0, 0, time.UTC),
				},
			},
			now:           time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC),
			wantTime:      "17:00",
			wantTimeRange: "17:00–18:30",
			wantWeekday:   "Friday",
			wantDateLabel: "18 September",
		},
		{
			name: "event with zero end time",
			events: []dashboard.Event{
				{
					Summary: "Dentist",
					Start:   time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC),
				},
			},
			now:           time.Date(2026, 9, 18, 6, 0, 0, 0, time.UTC),
			wantTime:      "10:00",
			wantTimeRange: "10:00",
			wantWeekday:   "Friday",
			wantDateLabel: "18 September",
		},
		{
			name: "all-day event",
			events: []dashboard.Event{
				{
					Summary: "Helgcup",
					AllDay:  true,
					Start:   time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
				},
			},
			now:           time.Date(2026, 9, 18, 6, 0, 0, 0, time.UTC),
			wantTime:      "All day",
			wantTimeRange: "All day",
			wantWeekday:   "Friday",
			wantDateLabel: "18 September",
		},
		{
			name: "event with description",
			events: []dashboard.Event{
				{
					Summary:     "Dentist",
					Description: "Bring insurance card",
					Start:       time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC),
				},
			},
			now:             time.Date(2026, 9, 18, 6, 0, 0, 0, time.UTC),
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
			data, err := NewPageData(tt.events, nil, []string{"Kaj", "Lillian"}, tt.now)
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

	data, err := NewPageData(nil, nil, []string{"Kaj", "Lillian"}, now)
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
	homeworkEvent := func(owner string, summary string, dayOffset int) dashboard.Event {
		start := time.Date(2026, 9, 17+dayOffset, 9, 0, 0, 0, loc)
		return dashboard.Event{Owner: owner, Summary: summary, Start: start}
	}

	return []homeworkTestCase{
		{
			name:           "day label is Due today for an event starting today",
			homeworkEvents: []dashboard.Event{homeworkEvent("Kaj", "Math worksheet", 0)},
			owners:         []string{"Kaj"},
			wantGroups: []HomeworkGroup{
				{Owner: "Kaj", Items: []HomeworkRow{{Summary: "Math worksheet", DayLabel: "Due today", IsToday: true}}},
			},
		},
		{
			name:           "day label is Due tomorrow for an event starting tomorrow",
			homeworkEvents: []dashboard.Event{homeworkEvent("Kaj", "Read ch. 4", 1)},
			owners:         []string{"Kaj"},
			wantGroups: []HomeworkGroup{
				{Owner: "Kaj", Items: []HomeworkRow{{Summary: "Read ch. 4", DayLabel: "Due tomorrow", IsToday: false}}},
			},
		},
		{
			name:           "day label is the weekday name for day+2 through day+6",
			homeworkEvents: []dashboard.Event{homeworkEvent("Kaj", "Spelling words", 6)},
			owners:         []string{"Kaj"},
			wantGroups: []HomeworkGroup{
				{Owner: "Kaj", Items: []HomeworkRow{{Summary: "Spelling words", DayLabel: "Wednesday", IsToday: false}}},
			},
		},
		{
			name:           "owners with no homework events produce no group entry",
			homeworkEvents: []dashboard.Event{homeworkEvent("Kaj", "Math worksheet", 0)},
			owners:         []string{"Kaj", "Lillian"},
			wantGroups: []HomeworkGroup{
				{Owner: "Kaj", Items: []HomeworkRow{{Summary: "Math worksheet", DayLabel: "Due today", IsToday: true}}},
			},
		},
		{
			name: "grouping respects the owners slice order",
			homeworkEvents: []dashboard.Event{
				homeworkEvent("Lillian", "Spelling words", 0),
				homeworkEvent("Kaj", "Math worksheet", 0),
			},
			owners: []string{"Lillian", "Kaj"},
			wantGroups: []HomeworkGroup{
				{Owner: "Lillian", Items: []HomeworkRow{{Summary: "Spelling words", DayLabel: "Due today", IsToday: true}}},
				{Owner: "Kaj", Items: []HomeworkRow{{Summary: "Math worksheet", DayLabel: "Due today", IsToday: true}}},
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
			data, err := NewPageData(nil, tt.homeworkEvents, tt.owners, now)
			if err != nil {
				t.Fatalf("NewPageData() error = %v", err)
			}
			assertHomeworkGroups(t, data.Homework, tt.wantGroups)
		})
	}
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
