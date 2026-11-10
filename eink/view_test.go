package eink

import (
	"reflect"
	"strings"
	"testing"
	"time"

	dashboard "family-dashboard"
)

// testNow is Stockholm Wednesday 16 September 2026 00:30 (week 38), while
// still Tuesday 15 September in UTC.
var testNow = time.Date(2026, 9, 15, 22, 30, 0, 0, time.UTC)

func utc(month time.Month, day, hour, min int) time.Time {
	return time.Date(2026, month, day, hour, min, 0, 0, time.UTC)
}

func TestNewView_Header(t *testing.T) {
	tests := []struct {
		name          string
		now           time.Time
		owners        []string
		wantDateLabel string
		wantWeekLabel string
		wantClock     string
		wantOwners    string
	}{
		{
			name:          "UTC and local dates differ",
			now:           testNow,
			owners:        []string{"Kaj", "Lillian"},
			wantDateLabel: "Onsdag 16 september",
			wantWeekLabel: "v.38",
			wantClock:     "00:30",
			wantOwners:    "Kaj · Lillian",
		},
		{
			name:          "single owner, winter time",
			now:           utc(time.December, 31, 12, 5),
			owners:        []string{"Kaj"},
			wantDateLabel: "Torsdag 31 december",
			wantWeekLabel: "v.53",
			wantClock:     "13:05",
			wantOwners:    "Kaj",
		},
		{
			name:          "no owners",
			now:           testNow,
			wantDateLabel: "Onsdag 16 september",
			wantWeekLabel: "v.38",
			wantClock:     "00:30",
			wantOwners:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view, err := NewView(nil, nil, nil, tt.owners, nil, nil, nil, tt.now)
			if err != nil {
				t.Fatalf("NewView: %v", err)
			}
			if view.DateLabel != tt.wantDateLabel {
				t.Errorf("DateLabel = %q, want %q", view.DateLabel, tt.wantDateLabel)
			}
			if view.WeekLabel != tt.wantWeekLabel {
				t.Errorf("WeekLabel = %q, want %q", view.WeekLabel, tt.wantWeekLabel)
			}
			if view.Clock != tt.wantClock {
				t.Errorf("Clock = %q, want %q", view.Clock, tt.wantClock)
			}
			if view.Owners != tt.wantOwners {
				t.Errorf("Owners = %q, want %q", view.Owners, tt.wantOwners)
			}
		})
	}
}

func TestNewView_Events(t *testing.T) {
	tests := []struct {
		name   string
		events []dashboard.Event
		want   []EventRow
	}{
		{
			name:   "timed event uses local time",
			events: []dashboard.Event{{Summary: "Tandläkare", Owner: "Kaj", Location: "Folktandvården", Start: utc(time.September, 16, 5, 15)}},
			want:   []EventRow{{Time: "07:15", Owner: "Kaj", Summary: "Tandläkare", Location: "Folktandvården"}},
		},
		{
			name:   "all-day event",
			events: []dashboard.Event{{Summary: "Studiedag", Owner: "Lillian", AllDay: true, Start: utc(time.September, 15, 22, 0)}},
			want:   []EventRow{{Time: "heldag", Owner: "Lillian", Summary: "Studiedag"}},
		},
		{
			name:   "Swedish characters kept",
			events: []dashboard.Event{{Summary: "Fotbollsträning på Åvallen", Location: "Ängby", Start: utc(time.September, 16, 15, 0)}},
			want:   []EventRow{{Time: "17:00", Summary: "Fotbollsträning på Åvallen", Location: "Ängby"}},
		},
		{
			name: "no events",
			want: []EventRow{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view, err := NewView(tt.events, nil, nil, nil, nil, nil, nil, testNow)
			if err != nil {
				t.Fatalf("NewView: %v", err)
			}
			if !reflect.DeepEqual(view.Events, tt.want) {
				t.Errorf("Events = %+v, want %+v", view.Events, tt.want)
			}
		})
	}
}

func TestNewView_Homework(t *testing.T) {
	tests := []struct {
		name   string
		events []dashboard.Event
		owners []string
		want   []HomeworkGroup
	}{
		{
			name: "day labels, prov marker, owner and item order",
			events: []dashboard.Event{
				{Summary: "Läxa matte", Owner: "Kaj", Start: utc(time.September, 18, 6, 0)},
				{Summary: "Läxa engelska", Owner: "Lillian", Description: "Glosprov kapitel 3", Start: utc(time.September, 16, 6, 0)},
				{Summary: "Läxa svenska", Owner: "Kaj", Start: utc(time.September, 17, 6, 0)},
			},
			owners: []string{"Kaj", "Lillian"},
			want: []HomeworkGroup{
				{Owner: "Kaj", Initial: "K", Items: []HomeworkRow{
					{Summary: "Läxa svenska", DayLabel: "imorgon"},
					{Summary: "Läxa matte", DayLabel: "fredag"},
				}},
				{Owner: "Lillian", Initial: "L", Items: []HomeworkRow{
					{Summary: "Läxa engelska", DayLabel: "idag", IsProv: true},
				}},
			},
		},
		{
			name: "owners without homework and unknown owners are skipped",
			events: []dashboard.Event{
				{Summary: "Läxa NO", Owner: "Lillian", Start: utc(time.September, 21, 6, 0)},
				{Summary: "Läxa okänd", Owner: "Okänd", Start: utc(time.September, 16, 6, 0)},
			},
			owners: []string{"Kaj", "Lillian"},
			want: []HomeworkGroup{
				{Owner: "Lillian", Initial: "L", Items: []HomeworkRow{
					{Summary: "Läxa NO", DayLabel: "måndag"},
				}},
			},
		},
		{
			name:   "no homework",
			owners: []string{"Kaj"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view, err := NewView(nil, tt.events, nil, tt.owners, nil, nil, nil, testNow)
			if err != nil {
				t.Fatalf("NewView: %v", err)
			}
			if !reflect.DeepEqual(view.Homework, tt.want) {
				t.Errorf("Homework = %+v, want %+v", view.Homework, tt.want)
			}
		})
	}
}

func TestNewView_Upcoming(t *testing.T) {
	tests := []struct {
		name   string
		events []dashboard.Event
		want   []UpcomingRow
	}{
		{
			name: "sorted by start with abbreviations and heldag",
			events: []dashboard.Event{
				{Summary: "Cup i Örebro", Owner: "Lillian", AllDay: true, Start: utc(time.September, 18, 22, 0)},
				{Summary: "Simning", Owner: "Kaj", Location: "Hässelbybadet", Start: utc(time.September, 17, 15, 0)},
				{Summary: "Föräldramöte", Owner: "Kaj", Start: utc(time.September, 21, 16, 30)},
			},
			want: []UpcomingRow{
				{Weekday: "Tors", Time: "17:00", Owner: "Kaj", Summary: "Simning", Location: "Hässelbybadet"},
				{Weekday: "Lör", Time: "heldag", Owner: "Lillian", Summary: "Cup i Örebro"},
				{Weekday: "Mån", Time: "18:30", Owner: "Kaj", Summary: "Föräldramöte"},
			},
		},
		{
			name: "weekday uses local date across midnight UTC",
			events: []dashboard.Event{
				{Summary: "Tidig buss", Start: utc(time.September, 18, 23, 30)},
			},
			want: []UpcomingRow{{Weekday: "Lör", Time: "01:30", Summary: "Tidig buss"}},
		},
		{
			name: "no upcoming events",
			want: []UpcomingRow{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view, err := NewView(nil, nil, tt.events, nil, nil, nil, nil, testNow)
			if err != nil {
				t.Fatalf("NewView: %v", err)
			}
			if !reflect.DeepEqual(view.Upcoming, tt.want) {
				t.Errorf("Upcoming = %+v, want %+v", view.Upcoming, tt.want)
			}
		})
	}
}

func TestNewView_Weather(t *testing.T) {
	tests := []struct {
		name    string
		weather *dashboard.WeatherReading
		want    WeatherView
	}{
		{
			name:    "no weather source",
			weather: nil,
			want:    WeatherView{Temperature: "—°", Lines: []string{"ingen väderkälla"}},
		},
		{
			name:    "rounded, sunny and dry",
			weather: &dashboard.WeatherReading{Temperature: 10.6, LowTemperature: 4.6, HighTemperature: 12.4, Sunny: true},
			want: WeatherView{Temperature: "11°", Lines: []string{
				"SMHI", "Lägst idag: 5° · Högst idag: 12°", "Sol", "torrt idag",
			}},
		},
		{
			name:    "cloudy with rain and zero degrees",
			weather: &dashboard.WeatherReading{Temperature: 0, LowTemperature: -3.4, HighTemperature: 2.5, PrecipitationExpected: true},
			want: WeatherView{Temperature: "0°", Lines: []string{
				"SMHI", "Lägst idag: -3° · Högst idag: 3°", "Mulet", "regn idag",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view, err := NewView(nil, nil, nil, nil, nil, tt.weather, nil, testNow)
			if err != nil {
				t.Fatalf("NewView: %v", err)
			}
			if !reflect.DeepEqual(view.Weather, tt.want) {
				t.Errorf("Weather = %+v, want %+v", view.Weather, tt.want)
			}
		})
	}
}

func TestNewView_Lunch(t *testing.T) {
	tests := []struct {
		name  string
		lunch []string
	}{
		{name: "dishes passed through", lunch: []string{"Köttbullar med potatismos", "Vegetarisk gröt"}},
		{name: "no menu"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view, err := NewView(nil, nil, nil, nil, tt.lunch, nil, nil, testNow)
			if err != nil {
				t.Fatalf("NewView: %v", err)
			}
			if !reflect.DeepEqual(view.Lunch, tt.lunch) {
				t.Errorf("Lunch = %q, want %q", view.Lunch, tt.lunch)
			}
		})
	}
}

func TestHeadingsAreUppercase(t *testing.T) {
	tests := []struct {
		name    string
		heading string
	}{
		{name: "lunch", heading: headingLunch},
		{name: "today", heading: headingToday},
		{name: "homework", heading: headingHomework},
		{name: "upcoming", heading: headingUpcoming},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.heading == "" || tt.heading != strings.ToUpper(tt.heading) {
				t.Errorf("heading %q is not a non-empty uppercase string", tt.heading)
			}
		})
	}
}

func TestEmptyStatesAreSet(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "lunch", text: emptyLunch},
		{name: "today", text: emptyToday},
		{name: "homework", text: emptyHomework},
		{name: "upcoming", text: emptyUpcoming},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.text == "" {
				t.Errorf("empty-state text for %s is empty", tt.name)
			}
		})
	}
}

func TestNewView_FeedNotice(t *testing.T) {
	missing := func(owner string) dashboard.FeedProblem { return dashboard.FeedProblem{Owner: owner} }
	stale := func(owner string, age time.Duration) dashboard.FeedProblem {
		return dashboard.FeedProblem{Owner: owner, Stale: true, LastSuccess: testNow.Add(-age)}
	}
	tests := []struct {
		name     string
		problems []dashboard.FeedProblem
		want     string
	}{
		{name: "none", want: ""},
		{name: "one missing", problems: []dashboard.FeedProblem{missing("Kaj")}, want: "Kalender saknas: K"},
		{name: "one stale at 3 h", problems: []dashboard.FeedProblem{stale("Kaj", 3*time.Hour+20*time.Minute)}, want: "Kalender gammal: K (3 h)"},
		{name: "stale at 59 min", problems: []dashboard.FeedProblem{stale("Kaj", 59*time.Minute+59*time.Second)}, want: "Kalender gammal: K (59 min)"},
		{name: "stale at 20 s", problems: []dashboard.FeedProblem{stale("Kaj", 20*time.Second)}, want: "Kalender gammal: K (1 min)"},
		{name: "stale at exactly 1 h", problems: []dashboard.FeedProblem{stale("Kaj", time.Hour)}, want: "Kalender gammal: K (1 h)"},
		{
			name:     "both kinds, missing first",
			problems: []dashboard.FeedProblem{stale("Lillian", 40*time.Minute), missing("Kaj")},
			want:     "Kalender saknas: K · Kalender gammal: L (40 min)",
		},
		{
			name: "several owners per kind",
			problems: []dashboard.FeedProblem{
				missing("Kaj"), stale("Kaj", 3*time.Hour), missing("Lillian"), stale("Lillian", 40*time.Minute),
			},
			want: "Kalender saknas: K, L · Kalender gammal: K (3 h), L (40 min)",
		},
		{
			name: "the same owner twice uses the oldest age",
			problems: []dashboard.FeedProblem{
				missing("lillian"), stale("Kaj", 2*time.Hour), missing("lillian"), stale("Kaj", 5*time.Hour), stale("Kaj", time.Hour),
			},
			want: "Kalender saknas: L · Kalender gammal: K (5 h)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view, err := NewView(nil, nil, nil, nil, nil, nil, tt.problems, testNow)
			if err != nil {
				t.Fatalf("NewView: %v", err)
			}
			if view.FeedNotice != tt.want {
				t.Errorf("FeedNotice = %q, want %q", view.FeedNotice, tt.want)
			}
		})
	}
}
