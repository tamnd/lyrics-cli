// Package lyrics is the library behind the lyrics command line:
// the HTTP client, request shaping, and typed data models for the
// Lyrics.ovh API (https://api.lyrics.ovh).
//
// The API provides two endpoints: fetch lyrics for a known artist+song,
// and suggest artist+song combinations by keyword. No API key required.
// The Client paces requests and retries transient failures.
package lyrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"sync"
	"time"
)

// Host is the API hostname.
const Host = "api.lyrics.ovh"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns sensible defaults for the Lyrics.ovh API.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api.lyrics.ovh",
		UserAgent: "lyrics-cli/0.1.0 (github.com/tamnd/lyrics-cli)",
		Rate:      200 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Lyrics holds the result of a lyrics lookup.
type Lyrics struct {
	Artist string `json:"artist"`
	Song   string `json:"song"`
	Text   string `json:"text"`
}

// Suggestion is one result from a suggest query.
type Suggestion struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Duration int    `json:"duration"` // seconds
	Rank     int    `json:"rank"`
	Artist   string `json:"artist"`
	URL      string `json:"url"` // constructed API URL for lyrics lookup
}

// Client talks to the Lyrics.ovh API.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// GetLyrics fetches lyrics for the given artist and song title.
func (c *Client) GetLyrics(ctx context.Context, artist, song string) (*Lyrics, error) {
	u := fmt.Sprintf("%s/v1/%s/%s",
		c.cfg.BaseURL,
		neturl.PathEscape(artist),
		neturl.PathEscape(song),
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var wire struct {
		Lyrics string `json:"lyrics"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode lyrics: %w", err)
	}
	if wire.Error != "" {
		return nil, fmt.Errorf("lyrics not found: %s", wire.Error)
	}
	return &Lyrics{
		Artist: artist,
		Song:   song,
		Text:   wire.Lyrics,
	}, nil
}

// Suggest searches for artist+song combinations matching query.
func (c *Client) Suggest(ctx context.Context, query string) ([]Suggestion, error) {
	u := fmt.Sprintf("%s/suggest/%s", c.cfg.BaseURL, neturl.PathEscape(query))
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}

	var wire struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Data []wireSuggestion `json:"data"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode suggest: %w", err)
	}

	out := make([]Suggestion, 0, len(wire.Data))
	for _, d := range wire.Data {
		artistName := d.Artist.Name
		lyricsURL := fmt.Sprintf("%s/v1/%s/%s",
			c.cfg.BaseURL,
			neturl.PathEscape(artistName),
			neturl.PathEscape(d.Title),
		)
		out = append(out, Suggestion{
			ID:       d.ID,
			Title:    d.Title,
			Duration: d.Duration,
			Rank:     d.Rank,
			Artist:   artistName,
			URL:      lyricsURL,
		})
	}
	return out, nil
}

// --- wire types ---

type wireSuggestion struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
	Rank     int    `json:"rank"`
	Artist   struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"artist"`
}

// --- HTTP internals ---

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}
