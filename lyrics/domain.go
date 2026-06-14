package lyrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes lyrics as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/lyrics-cli/lyrics"
//
// The same Domain also builds the standalone lyrics binary (see cli.NewApp).
func init() { kit.Register(Domain{}) }

// Domain is the lyrics driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "lyrics",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "lyrics",
			Short:  "Fetch song lyrics and artist/song suggestions from Lyrics.ovh",
			Long: `lyrics fetches song lyrics and artist/song suggestions from Lyrics.ovh.

Get the full text of any song, or search for artist+song combinations
by keyword. No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/lyrics-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// get: fetch lyrics for an artist + song
	kit.Handle(app, kit.OpMeta{
		Name:    "get",
		Group:   "read",
		Single:  true,
		Summary: "Get lyrics for a song",
		Args: []kit.Arg{
			{Name: "artist", Help: "artist name"},
			{Name: "song", Help: "song title"},
		},
	}, getLyricsOp)

	// suggest: search for artist+song suggestions
	kit.Handle(app, kit.OpMeta{
		Name:    "suggest",
		Group:   "read",
		List:    true,
		Summary: "Search for artist+song suggestions",
		Args:    []kit.Arg{{Name: "query", Help: "search keyword"}},
	}, suggestOp)
}

// newClient builds the client from host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type getLyricsInput struct {
	Artist string  `kit:"arg" help:"artist name"`
	Song   string  `kit:"arg" help:"song title"`
	Client *Client `kit:"inject"`
}

type suggestInput struct {
	Query  string  `kit:"arg" help:"search keyword"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func getLyricsOp(ctx context.Context, in getLyricsInput, emit func(*Lyrics) error) error {
	lyr, err := in.Client.GetLyrics(ctx, in.Artist, in.Song)
	if err != nil {
		return err
	}
	return emit(lyr)
}

func suggestOp(ctx context.Context, in suggestInput, emit func(*Suggestion) error) error {
	items, err := in.Client.Suggest(ctx, in.Query)
	if err != nil {
		return err
	}
	for i := range items {
		if in.Limit > 0 && i >= in.Limit {
			break
		}
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if strings.TrimSpace(input) == "" {
		return "", "", errs.Usage("empty lyrics reference")
	}
	return "lyrics", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "lyrics":
		return fmt.Sprintf("https://%s/v1/%s", Host, id), nil
	case "suggest":
		return fmt.Sprintf("https://%s/suggest/%s", Host, id), nil
	default:
		return "", errs.Usage("lyrics has no resource type %q", uriType)
	}
}
