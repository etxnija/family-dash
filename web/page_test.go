package web

import (
	"bytes"
	"strings"
	"testing"
	"time"

	dashboard "family-dashboard"
)

func TestNewPageData(t *testing.T) {
	tests := []struct {
		name          string
		events        []dashboard.Event
		now           time.Time
		wantTimeRange string
		wantWeekday   string
		wantDateLabel string
	}{
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
			wantTimeRange: "10:00",
			wantWeekday:   "Friday",
			wantDateLabel: "18 September",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := NewPageData(tt.events, []string{"Kaj", "Lillian"}, tt.now)
			if err != nil {
				t.Fatalf("NewPageData() error = %v", err)
			}
			if len(data.Events) != 1 {
				t.Fatalf("len(data.Events) = %d, want 1", len(data.Events))
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
			want: []string{"Nothing scheduled today."},
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
