package lyrics_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/lyrics-cli/lyrics"
)

// --- test data ---

const fakeLyricsJSON = `{"lyrics":"Is this the real life ?\nIs this just fantasy ?\nCaught in a landslide\nNo escape from reality"}`

const fakeLyricsNotFoundJSON = `{"error":"No lyrics found"}`

const fakeSuggestJSON = `{
  "total": {"value": 62},
  "data": [
    {
      "id": 7234,
      "title": "Here Comes The Sun (Remastered 2009)",
      "duration": 186,
      "rank": 849432,
      "artist": {"id": 1289, "name": "The Beatles"}
    },
    {
      "id": 1234,
      "title": "Let It Be",
      "duration": 243,
      "rank": 900000,
      "artist": {"id": 1289, "name": "The Beatles"}
    }
  ]
}`

const fakeSuggestEmptyJSON = `{"total": {"value": 0}, "data": []}`

// --- helpers ---

func newTestClient(ts *httptest.Server) *lyrics.Client {
	cfg := lyrics.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return lyrics.NewClient(cfg)
}

func serve(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, body)
	}))
}

// --- tests ---

func TestGetLyricsSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeLyricsJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.GetLyrics(context.Background(), "Queen", "Bohemian Rhapsody")
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestGetLyricsParsesResponse(t *testing.T) {
	ts := serve(fakeLyricsJSON)
	defer ts.Close()

	c := newTestClient(ts)
	lyr, err := c.GetLyrics(context.Background(), "Queen", "Bohemian Rhapsody")
	if err != nil {
		t.Fatal(err)
	}
	if lyr.Artist != "Queen" {
		t.Errorf("Artist = %q, want Queen", lyr.Artist)
	}
	if lyr.Song != "Bohemian Rhapsody" {
		t.Errorf("Song = %q, want Bohemian Rhapsody", lyr.Song)
	}
	if lyr.Text == "" {
		t.Error("Text is empty")
	}
	if lyr.Text != "Is this the real life ?\nIs this just fantasy ?\nCaught in a landslide\nNo escape from reality" {
		t.Errorf("Text = %q, unexpected", lyr.Text)
	}
}

func TestGetLyricsNotFound(t *testing.T) {
	hits := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = fmt.Fprint(w, fakeLyricsNotFoundJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.GetLyrics(context.Background(), "Unknown", "NoSong")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should read once and not retry (it's a 200 with error in body, not a 4xx)
	if hits != 1 {
		t.Errorf("server saw %d hits, want 1", hits)
	}
}

func TestGetLyricsRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeLyricsJSON)
	}))
	defer ts.Close()

	cfg := lyrics.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := lyrics.NewClient(cfg)

	_, err := c.GetLyrics(context.Background(), "Queen", "Bohemian Rhapsody")
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestGetLyricsNonRetryable404(t *testing.T) {
	hits := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.GetLyrics(context.Background(), "X", "Y")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if hits != 1 {
		t.Errorf("server saw %d hits, want 1 (no retry on 404)", hits)
	}
}

func TestSuggestParsesResponse(t *testing.T) {
	ts := serve(fakeSuggestJSON)
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Suggest(context.Background(), "beatles")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	s := items[0]
	if s.ID != 7234 {
		t.Errorf("ID = %d, want 7234", s.ID)
	}
	if s.Title != "Here Comes The Sun (Remastered 2009)" {
		t.Errorf("Title = %q, unexpected", s.Title)
	}
	if s.Artist != "The Beatles" {
		t.Errorf("Artist = %q, want The Beatles", s.Artist)
	}
	if s.Duration != 186 {
		t.Errorf("Duration = %d, want 186", s.Duration)
	}
	if s.URL == "" {
		t.Error("URL is empty")
	}
}

func TestSuggestEmptyResult(t *testing.T) {
	ts := serve(fakeSuggestEmptyJSON)
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Suggest(context.Background(), "zzznoresults")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}
