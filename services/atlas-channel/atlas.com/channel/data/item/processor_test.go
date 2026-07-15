package item

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/sirupsen/logrus"
)

// itemStringsResponse is a real JSON:API item-strings list response, mirroring
// what atlas-data's handleGetItemStringsRequest marshals (resource type
// "item-strings", id = template id, attribute name). It is served verbatim so the
// test exercises the same api2go unmarshal path the live client uses (the
// relationship-stub gotcha in libs/atlas-rest).
const itemStringsResponse = `{
  "data": [
    { "type": "item-strings", "id": "1302000", "attributes": { "name": "Sword" } },
    { "type": "item-strings", "id": "1302001", "attributes": { "name": "Hand Axe" } }
  ]
}`

// TestGetIdsByName_ResolvesTemplateIds stands up an httptest server emulating
// atlas-data's item-string search and asserts the processor returns the matching
// template ids. DATA_SERVICE_URL points the RootUrl("DATA") client at the server.
func TestGetIdsByName_ResolvesTemplateIds(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("search")
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(itemStringsResponse))
	}))
	defer srv.Close()

	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ids, err := NewProcessor(logrus.New(), context.Background()).GetIdsByName("Sword")
	if err != nil {
		t.Fatalf("GetIdsByName returned error: %v", err)
	}

	if gotPath != "/data/item-strings" {
		t.Errorf("request path: want /data/item-strings, got %q", gotPath)
	}
	if gotQuery != "Sword" {
		t.Errorf("search query: want Sword, got %q", gotQuery)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	want := []uint32{1302000, 1302001}
	if len(ids) != len(want) {
		t.Fatalf("ids: want %v, got %v", want, ids)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("ids[%d]: want %d, got %d", i, want[i], ids[i])
		}
	}
}

// TestGetIdsByName_EmptyResult asserts a zero-hit response yields an empty (non-
// error) id slice — the search arm treats that as "matched nothing".
func TestGetIdsByName_EmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer srv.Close()

	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ids, err := NewProcessor(logrus.New(), context.Background()).GetIdsByName("Nonexistent")
	if err != nil {
		t.Fatalf("GetIdsByName returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("ids: want empty, got %v", ids)
	}
}
