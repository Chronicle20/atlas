package seeder

import (
	"regexp"
	"strings"
	"testing"
)

func TestParseEnvelope_Valid(t *testing.T) {
	env, err := ParseEnvelope([]byte(`{"data":{"type":"widget","id":"42","attributes":{"name":"hi"}}}`))
	if err != nil {
		t.Fatalf("ParseEnvelope: %v", err)
	}
	if env.Data.Type != "widget" || env.Data.ID != "42" {
		t.Fatalf("got type=%q id=%q", env.Data.Type, env.Data.ID)
	}
	if string(env.Data.Attributes) == "" {
		t.Fatalf("attributes empty")
	}
}

func TestParseEnvelope_MissingData(t *testing.T) {
	_, err := ParseEnvelope([]byte(`{"type":"widget"}`))
	if err == nil || !strings.Contains(err.Error(), "data") {
		t.Fatalf("expected data-missing error, got: %v", err)
	}
}

func TestParseEnvelope_MalformedJSON(t *testing.T) {
	_, err := ParseEnvelope([]byte(`{"data":`))
	if err == nil {
		t.Fatalf("expected error on malformed JSON")
	}
}

func TestExtractEntityID_Match(t *testing.T) {
	id, err := ExtractEntityID("monster-100100.json", monsterPattern())
	if err != nil {
		t.Fatalf("ExtractEntityID: %v", err)
	}
	if id != "100100" {
		t.Fatalf("id = %q, want 100100", id)
	}
}

func TestExtractEntityID_NoMatch(t *testing.T) {
	_, err := ExtractEntityID("bogus.json", monsterPattern())
	if err == nil {
		t.Fatalf("expected error on no match")
	}
}

func monsterPattern() *regexp.Regexp { return regexp.MustCompile(`^monster-(\d+)\.json$`) }
