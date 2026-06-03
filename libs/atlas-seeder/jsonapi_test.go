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

func TestDecodeAttributes_HappyPath(t *testing.T) {
	type widget struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	var w widget
	err := DecodeAttributes(
		[]byte(`{"data":{"type":"widget","id":"1","attributes":{"name":"hello","count":7}}}`),
		&w,
	)
	if err != nil {
		t.Fatalf("DecodeAttributes: %v", err)
	}
	if w.Name != "hello" || w.Count != 7 {
		t.Fatalf("decoded = %+v, want {Name:hello Count:7}", w)
	}
}

func TestDecodeAttributes_MalformedEnvelope(t *testing.T) {
	var w struct{}
	if err := DecodeAttributes([]byte(`{"data":`), &w); err == nil {
		t.Fatalf("expected error on malformed JSON")
	}
}

// Regression for reactor-drop catalog shape: data has only relationships,
// the actual per-entity attributes live in included[]. DecodeAttributes
// must surface a clear "no attributes" error rather than silently
// returning an empty target — the latter masks the bug that left
// reactor-drop seeded with count=0 on the first real PR-env deploy.
func TestDecodeAttributes_NoAttributesIsError(t *testing.T) {
	var w struct {
		Drops []int `json:"drops"`
	}
	payload := []byte(`{
		"data": {"type":"reactor-drop","id":"1002008","relationships":{"drops":{"data":[]}}},
		"included": [{"type":"drops","id":"1002008:1:2","attributes":{"itemId":1}}]
	}`)
	err := DecodeAttributes(payload, &w)
	if err == nil {
		t.Fatalf("expected error when data.attributes is absent; got nil")
	}
	if !strings.Contains(err.Error(), "attributes") {
		t.Fatalf("expected attributes-related error, got: %v", err)
	}
}
