package skolmaten_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"family-dashboard/skolmaten"
)

// exampleFeed is a realistic skolmaten.se RSS response: one <item> per day,
// with dishes separated by literal "<br/>" text inside the description.
const exampleFeed = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <title>Skolmaten - Nya Elementar</title>
    <link>https://skolmaten.se/nya-elementar</link>
    <description>Lunch menu</description>
    <item>
      <title>Måndag 21 september</title>
      <description><![CDATA[Fiskgratäng med potatismos och grönsaker,<br/>Vegetarisk gryta med bönor,<br/>Pasta med tomatsås,<br/>]]></description>
      <pubDate>Mon, 21 Sep 2026 00:00:00 +0200</pubDate>
    </item>
  </channel>
</rss>
`

const zeroItemFeed = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <title>Skolmaten - Nya Elementar</title>
    <link>https://skolmaten.se/nya-elementar</link>
    <description>Lunch menu</description>
  </channel>
</rss>
`

// stubRoundTripper returns a canned response or error without touching the
// network.
type stubRoundTripper struct {
	resp *http.Response
	err  error
}

func (s stubRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return s.resp, s.err
}

func TestLunch(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    []string
		wantErr bool
	}{
		{
			name: "example feed yields three dishes",
			body: exampleFeed,
			want: []string{
				"Fiskgratäng med potatismos och grönsaker",
				"Vegetarisk gryta med bönor",
				"Pasta med tomatsås",
			},
		},
		{
			name: "zero items yields empty dishes",
			body: zeroItemFeed,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				Transport: stubRoundTripper{
					resp: &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
					},
				},
			}

			c := skolmaten.New("https://skolmaten.se/api/4/rss/day/nya-elementar", client)
			got, err := c.Lunch(context.Background())
			if err != nil {
				t.Fatalf("Lunch() error = %v", err)
			}
			if len(got.Dishes) != len(tt.want) {
				t.Fatalf("Lunch() dishes = %v, want %v", got.Dishes, tt.want)
			}
			for i := range tt.want {
				if got.Dishes[i] != tt.want[i] {
					t.Errorf("dish %d = %q, want %q", i, got.Dishes[i], tt.want[i])
				}
			}
		})
	}
}

func TestLunch_HTTPError(t *testing.T) {
	client := &http.Client{
		Transport: stubRoundTripper{
			resp: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
			},
		},
	}

	c := skolmaten.New("https://skolmaten.se/api/4/rss/day/nya-elementar", client)
	_, err := c.Lunch(context.Background())
	if err == nil {
		t.Fatal("Lunch() error = nil, want error")
	}
}
