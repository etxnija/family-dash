package webcal_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	dashboard "family-dashboard"
	"family-dashboard/webcal"
)

func stockholm(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	return loc
}

func TestParseICS_NormalEvents(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.ics")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	loc := stockholm(t)

	got, err := webcal.ParseICS(data)
	if err != nil {
		t.Fatalf("ParseICS() error = %v", err)
	}

	want := []dashboard.Event{
		{
			Summary:     "Träning - F2015 Vit",
			Location:    "Kålltorps IP A-plan",
			Description: "Samling 17:45, träning 18:00-19:30. Ta med vattenflaska.",
			Start:       time.Date(2026, 9, 17, 18, 0, 0, 0, loc),
			End:         time.Date(2026, 9, 17, 19, 30, 0, 0, loc),
		},
		{
			Summary:     "Match - F2015 Vit - IFK Exempel",
			Location:    "Kålltorps IP A-plan",
			Description: "Samling 13:15, matchstart 14:00.",
			Start:       time.Date(2026, 9, 20, 14, 0, 0, 0, loc),
			End:         time.Date(2026, 9, 20, 16, 0, 0, 0, loc),
		},
	}

	if len(got) != len(want) {
		t.Fatalf("ParseICS() returned %d events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if !got[i].Start.Equal(want[i].Start) {
			t.Errorf("event %d Start = %v, want %v", i, got[i].Start, want[i].Start)
		}
		if !got[i].End.Equal(want[i].End) {
			t.Errorf("event %d End = %v, want %v", i, got[i].End, want[i].End)
		}
		if got[i].Summary != want[i].Summary {
			t.Errorf("event %d Summary = %q, want %q", i, got[i].Summary, want[i].Summary)
		}
		if got[i].Location != want[i].Location {
			t.Errorf("event %d Location = %q, want %q", i, got[i].Location, want[i].Location)
		}
		if got[i].Description != want[i].Description {
			t.Errorf("event %d Description = %q, want %q", i, got[i].Description, want[i].Description)
		}
	}
}

func TestParseICS_EmptyFeed(t *testing.T) {
	const feed = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//SportAdmin//Kalender//SV
CALSCALE:GREGORIAN
END:VCALENDAR
`
	got, err := webcal.ParseICS([]byte(feed))
	if err != nil {
		t.Fatalf("ParseICS() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ParseICS() = %+v, want empty slice", got)
	}
}

func TestParseICS_LineFolding(t *testing.T) {
	// The SUMMARY value is folded across a continuation line, which per
	// RFC 5545 §3.1 starts with a single space and must be rejoined.
	feed := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART;TZID=Europe/Stockholm:20260917T180000\r\n" +
		"DTEND;TZID=Europe/Stockholm:20260917T190000\r\n" +
		"SUMMARY:Träning - F2015 Vit (samlingsplats Kålltorps I\r\n" +
		" P A-plan)\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	got, err := webcal.ParseICS([]byte(feed))
	if err != nil {
		t.Fatalf("ParseICS() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ParseICS() returned %d events, want 1", len(got))
	}
	want := "Träning - F2015 Vit (samlingsplats Kålltorps IP A-plan)"
	if got[0].Summary != want {
		t.Errorf("Summary = %q, want %q", got[0].Summary, want)
	}
}

func TestParseICS_MalformedEvent(t *testing.T) {
	tests := []struct {
		name string
		feed string
	}{
		{
			name: "unterminated VEVENT",
			feed: "BEGIN:VCALENDAR\n" +
				"BEGIN:VEVENT\n" +
				"DTSTART;TZID=Europe/Stockholm:20260917T180000\n" +
				"SUMMARY:Träning\n" +
				"END:VCALENDAR\n",
		},
		{
			name: "missing DTSTART",
			feed: "BEGIN:VCALENDAR\n" +
				"BEGIN:VEVENT\n" +
				"SUMMARY:Träning\n" +
				"END:VEVENT\n" +
				"END:VCALENDAR\n",
		},
		{
			name: "unparsable DTSTART",
			feed: "BEGIN:VCALENDAR\n" +
				"BEGIN:VEVENT\n" +
				"DTSTART;TZID=Europe/Stockholm:not-a-timestamp\n" +
				"SUMMARY:Träning\n" +
				"END:VEVENT\n" +
				"END:VCALENDAR\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := webcal.ParseICS([]byte(tt.feed))
			if err == nil {
				t.Fatal("ParseICS() error = nil, want error")
			}
		})
	}
}

// stubRoundTripper returns a canned response or error without touching the
// network.
type stubRoundTripper struct {
	resp *http.Response
	err  error
}

func (s stubRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return s.resp, s.err
}

func TestEvents_HTTPError(t *testing.T) {
	const secretURL = "https://sportadmin.example.com/calendar/very-secret-token.ics"

	client := &http.Client{
		Transport: stubRoundTripper{
			resp: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
			},
		},
	}

	c := webcal.New(secretURL, client)
	_, err := c.Events(context.Background())
	if err == nil {
		t.Fatal("Events() error = nil, want error")
	}
	if strings.Contains(err.Error(), secretURL) {
		t.Errorf("Events() error %q contains the feed URL", err.Error())
	}
	if strings.Contains(err.Error(), "very-secret-token") {
		t.Errorf("Events() error %q contains the feed token", err.Error())
	}
}
