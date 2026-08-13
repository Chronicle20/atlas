package character

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// characterResponse is a real JSON:API "characters" single-resource response
// carrying a relationships block, mirroring what atlas-characters actually
// returns for this resource. It is served verbatim so the test drives the
// real api2go unmarshal path (and the SetToOneReferenceID stub) that the
// live client uses. Without the EXT-01 stubs, GetById fails the entire
// decode with "struct *character.RestModel does not implement
// UnmarshalToOneRelations" — a mock built from a RestModel literal (bypassing
// requests.GetRequest) would never catch that.
const characterResponse = `{
  "data": {
    "type": "characters", "id": "100100", "attributes": { "name": "Bob" },
    "relationships": {
      "account": { "data": { "type": "accounts", "id": "500" } }
    }
  }
}`

const charactersListResponse = `{
  "data": [
    { "type": "characters", "id": "100100", "attributes": { "name": "Bob" },
      "relationships": { "account": { "data": { "type": "accounts", "id": "500" } } } }
  ]
}`

func TestGetById_UnmarshalsRelationships(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(characterResponse))
	}))
	defer srv.Close()

	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), context.Background()).GetById(100100)
	if err != nil {
		t.Fatalf("GetById returned error: %v", err)
	}
	if gotPath != "/characters/100100" {
		t.Errorf("request path: want /characters/100100, got %q", gotPath)
	}
	if m.Id() != 100100 {
		t.Errorf("id = %d, want 100100", m.Id())
	}
	if m.Name() != "Bob" {
		t.Errorf("name = %q, want Bob", m.Name())
	}
}

func TestGetByName_UnmarshalsRelationships(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(charactersListResponse))
	}))
	defer srv.Close()

	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), context.Background()).GetByName("Bob")
	if err != nil {
		t.Fatalf("GetByName returned error: %v", err)
	}
	if m.Id() != 100100 {
		t.Errorf("id = %d, want 100100", m.Id())
	}
	if m.Name() != "Bob" {
		t.Errorf("name = %q, want Bob", m.Name())
	}
}
