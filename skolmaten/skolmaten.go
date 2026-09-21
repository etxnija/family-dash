// Package skolmaten adapts the skolmaten.se lunch RSS feed to the
// dashboard.LunchSource port.
package skolmaten

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	dashboard "family-dashboard"
)

// rssFeed is the top-level XML structure of the skolmaten.se RSS feed.
type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

// rssItem is a single day's entry in the feed.
type rssItem struct {
	Description string `xml:"description"`
}

// Client fetches and parses the skolmaten.se lunch RSS feed.
type Client struct {
	url        string
	httpClient *http.Client
}

// New builds a Client for the given RSS feed URL. The caller supplies the
// *http.Client so tests can inject a stub transport instead of hitting the
// network.
func New(url string, client *http.Client) *Client {
	return &Client{url: url, httpClient: client}
}

// Lunch implements dashboard.LunchSource by fetching and parsing the RSS
// feed.
func (c *Client) Lunch(ctx context.Context) (dashboard.LunchMenu, error) {
	data, err := c.fetch(ctx)
	if err != nil {
		return dashboard.LunchMenu{}, err
	}
	return parseRSS(data)
}

// fetch retrieves the raw RSS bytes over HTTP.
func (c *Client) fetch(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("skolmaten: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("skolmaten: fetch lunch feed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("skolmaten: lunch feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("skolmaten: read lunch feed: %w", err)
	}
	return body, nil
}

// parseRSS parses raw RSS XML into a dashboard.LunchMenu. A feed with zero
// <item> elements is not an error — it yields an empty dish list. The feed
// is assumed to hold at most one <item>; only the first is used.
func parseRSS(data []byte) (dashboard.LunchMenu, error) {
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return dashboard.LunchMenu{}, fmt.Errorf("skolmaten: parse lunch feed: %w", err)
	}

	if len(feed.Channel.Items) == 0 {
		return dashboard.LunchMenu{}, nil
	}

	return dashboard.LunchMenu{Dishes: parseDishes(feed.Channel.Items[0].Description)}, nil
}

// parseDishes splits a feed item's description into individual dish names.
// Dishes are separated by literal "<br/>" text within the description
// (not parsed as XML), each trimmed of surrounding whitespace and a
// trailing comma.
func parseDishes(description string) []string {
	var dishes []string
	for _, part := range strings.Split(description, "<br/>") {
		dish := strings.TrimSpace(part)
		dish = strings.TrimSuffix(dish, ",")
		dish = strings.TrimSpace(dish)
		if dish == "" {
			continue
		}
		dishes = append(dishes, dish)
	}
	return dishes
}
