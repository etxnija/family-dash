// Package webcal adapts ICS ("webcal") calendar feeds to the
// dashboard.EventSource port. SportAdmin is the first concrete source.
package webcal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	dashboard "family-dashboard"
)

// Client fetches and parses a SportAdmin ICS calendar feed.
type Client struct {
	url        string
	httpClient *http.Client
}

// New builds a Client for the given ICS feed URL. The caller supplies the
// *http.Client so tests can inject a stub transport instead of hitting the
// network.
func New(url string, client *http.Client) *Client {
	return &Client{url: url, httpClient: client}
}

// Events implements dashboard.EventSource by fetching and parsing the ICS
// feed.
func (c *Client) Events(ctx context.Context) ([]dashboard.Event, error) {
	data, err := c.fetch(ctx)
	if err != nil {
		return nil, err
	}
	return ParseICS(data)
}

// fetch retrieves the raw ICS bytes over HTTP. Error messages never include
// the feed URL, since it embeds a private subscription token.
func (c *Client) fetch(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("sportadmin: build request: %w", scrubURL(err))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sportadmin: fetch calendar feed: %w", scrubURL(err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sportadmin: calendar feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sportadmin: read calendar feed: %w", scrubURL(err))
	}
	return body, nil
}

// scrubURL strips the request URL out of net/http's *url.Error so it never
// leaks into a returned error message.
func scrubURL(err error) error {
	var uerr *url.Error
	if errors.As(err, &uerr) {
		return uerr.Err
	}
	return err
}

// eventBuilder accumulates properties for the VEVENT block currently being
// parsed.
type eventBuilder struct {
	summary, location, description string

	hasDTStart    bool
	dtstartValue  string
	dtstartParams map[string]string
	hasDTEnd      bool
	dtendValue    string
	dtendParams   map[string]string
}

// ParseICS parses raw ICS calendar data into dashboard events. It returns an
// error only for genuinely malformed input (unterminated VEVENT blocks,
// unparsable timestamps, or a VEVENT missing DTSTART) — a feed with zero
// events is not an error.
func ParseICS(data []byte) ([]dashboard.Event, error) {
	lines := unfoldLines(data)

	var events []dashboard.Event
	var cur eventBuilder
	inVevent := false

	for _, line := range lines {
		name, params, value := parseLine(line)

		switch name {
		case "BEGIN":
			if value == "VEVENT" {
				inVevent = true
				cur = eventBuilder{}
			}
		case "END":
			if value != "VEVENT" {
				continue
			}
			if !inVevent {
				return nil, errors.New("sportadmin: END:VEVENT without matching BEGIN:VEVENT")
			}
			event, err := cur.build()
			if err != nil {
				return nil, err
			}
			if event != nil {
				events = append(events, *event)
			}
			inVevent = false
		case "SUMMARY":
			if inVevent {
				cur.summary = unescapeText(value)
			}
		case "LOCATION":
			if inVevent {
				cur.location = unescapeText(value)
			}
		case "DESCRIPTION":
			if inVevent {
				cur.description = unescapeText(value)
			}
		case "DTSTART":
			if inVevent {
				cur.hasDTStart = true
				cur.dtstartValue = value
				cur.dtstartParams = params
			}
		case "DTEND":
			if inVevent {
				cur.hasDTEnd = true
				cur.dtendValue = value
				cur.dtendParams = params
			}
		}
	}

	if inVevent {
		return nil, errors.New("sportadmin: unterminated VEVENT block")
	}
	return events, nil
}

// build converts the accumulated VEVENT properties into a dashboard.Event.
// It returns a nil event (no error) for all-day events, which the dashboard
// doesn't need to schedule to the minute.
func (b eventBuilder) build() (*dashboard.Event, error) {
	if !b.hasDTStart {
		return nil, errors.New("sportadmin: VEVENT missing DTSTART")
	}
	if b.dtstartParams["VALUE"] == "DATE" {
		return nil, nil
	}

	start, err := parseICSTime(b.dtstartValue, b.dtstartParams)
	if err != nil {
		return nil, fmt.Errorf("sportadmin: parse DTSTART: %w", err)
	}

	var end time.Time
	if b.hasDTEnd && b.dtendParams["VALUE"] != "DATE" {
		end, err = parseICSTime(b.dtendValue, b.dtendParams)
		if err != nil {
			return nil, fmt.Errorf("sportadmin: parse DTEND: %w", err)
		}
	}

	return &dashboard.Event{
		Summary:     b.summary,
		Location:    b.location,
		Description: b.description,
		Start:       start,
		End:         end,
	}, nil
}

// parseICSTime parses a DTSTART/DTEND value. SportAdmin's feed uses local
// Stockholm times, with or without an explicit TZID param, so a bare
// timestamp (no trailing Z) is interpreted in that zone unless TZID says
// otherwise.
func parseICSTime(value string, params map[string]string) (time.Time, error) {
	if strings.HasSuffix(value, "Z") {
		t, err := time.Parse("20060102T150405Z", value)
		if err != nil {
			return time.Time{}, err
		}
		return t.UTC(), nil
	}

	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		return time.Time{}, err
	}
	if tzid := params["TZID"]; tzid != "" {
		if l, err := time.LoadLocation(tzid); err == nil {
			loc = l
		}
	}
	t, err := time.ParseInLocation("20060102T150405", value, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// unfoldLines splits raw ICS bytes into logical (unfolded) lines per
// RFC 5545 §3.1: a line starting with a single space or tab is a
// continuation of the previous line, folded there by the feed producer.
// Blank lines are dropped.
func unfoldLines(data []byte) []string {
	raw := strings.ReplaceAll(string(data), "\r\n", "\n")
	rawLines := strings.Split(raw, "\n")

	var lines []string
	for _, line := range rawLines {
		if line == "" {
			continue
		}
		if (line[0] == ' ' || line[0] == '\t') && len(lines) > 0 {
			lines[len(lines)-1] += line[1:]
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

// parseLine splits a logical ICS line into its property name, parameters,
// and value, e.g. "DTSTART;TZID=Europe/Stockholm:20260917T180000" becomes
// ("DTSTART", {"TZID": "Europe/Stockholm"}, "20260917T180000").
func parseLine(line string) (name string, params map[string]string, value string) {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return line, nil, ""
	}

	left := line[:idx]
	value = line[idx+1:]

	parts := strings.Split(left, ";")
	name = parts[0]

	params = make(map[string]string, len(parts)-1)
	for _, p := range parts[1:] {
		if k, v, ok := strings.Cut(p, "="); ok {
			params[k] = v
		}
	}
	return name, params, value
}

// unescapeText reverses RFC 5545 §3.3.11 text escaping used in SUMMARY,
// LOCATION, and DESCRIPTION values.
func unescapeText(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n', 'N':
				b.WriteByte('\n')
				i++
				continue
			case ',', ';', '\\':
				b.WriteByte(s[i+1])
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
