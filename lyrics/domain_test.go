package lyrics

import (
	"testing"
)

// These tests are offline: they exercise the URI driver's pure string functions.
// HTTP behaviour is covered in lyrics_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "lyrics" {
		t.Errorf("Scheme = %q, want lyrics", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "lyrics" {
		t.Errorf("Identity.Binary = %q, want lyrics", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	typ, id, err := Domain{}.Classify("Queen/Bohemian Rhapsody")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if typ != "lyrics" {
		t.Errorf("type = %q, want lyrics", typ)
	}
	if id == "" {
		t.Error("id is empty")
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestLocateLyrics(t *testing.T) {
	got, err := Domain{}.Locate("lyrics", "Queen/Bohemian+Rhapsody")
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if got == "" {
		t.Error("URL is empty")
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "x")
	if err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}
