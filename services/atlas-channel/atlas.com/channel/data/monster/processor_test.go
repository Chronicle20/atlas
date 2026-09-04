package monster

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// monsterResponse is a real JSON:API monster response, mirroring what
// atlas-data's monster resource marshals (resource type "monsters", matching
// RestModel.GetName()). It is served verbatim so the test exercises the same
// api2go unmarshal path the live client uses (the relationship-stub gotcha
// in libs/atlas-rest).
const monsterResponse = `{
  "data": { "type": "monsters", "id": "8510000", "attributes": { "boss": true, "fixed_damage": 5, "tag_color": 6, "tag_background_color": 1 } }
}`

// TestUpstreamFn_DecodesMonster stands up an httptest server emulating
// atlas-data's monster lookup and drives upstreamFn directly — the real
// requestById -> requests.GetRequest[RestModel] -> Extract path — bypassing
// the process-wide TTL cache (getByIdCached/getMonsterCache) so the test
// cannot pass on a stale cache hit without ever contacting the server.
func TestUpstreamFn_DecodesMonster(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(monsterResponse))
	}))
	defer srv.Close()

	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	m, err := upstreamFn(logrus.New(), context.Background(), 8510000)
	if err != nil {
		t.Fatalf("upstreamFn returned error: %v", err)
	}

	if gotPath != "/data/monsters/8510000" {
		t.Errorf("request path: want /data/monsters/8510000, got %q", gotPath)
	}

	if m.Id() != 8510000 {
		t.Errorf("Id=%d, want 8510000", m.Id())
	}
	if !m.Boss() {
		t.Error("Boss=false, want true")
	}
	if m.FixedDamage() != 5 {
		t.Errorf("FixedDamage=%d, want 5", m.FixedDamage())
	}
	if m.TagColor() != 6 {
		t.Errorf("TagColor=%d, want 6", m.TagColor())
	}
	if m.TagBackgroundColor() != 1 {
		t.Errorf("TagBackgroundColor=%d, want 1", m.TagBackgroundColor())
	}
}

// TestUpstreamFn_TagColorsDecodeFromWireTags pins the wire json tags
// "tag_color" / "tag_background_color" (matching atlas-data's producer
// field-for-field) so a drift between the two services' tags fails loudly
// here instead of silently decoding to 0 and producing a wrong-coloured HP
// gauge at runtime with no failure anywhere.
func TestUpstreamFn_TagColorsDecodeFromWireTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"monsters","id":"8510001","attributes":{"tag_color":9,"tag_background_color":3}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	m, err := upstreamFn(logrus.New(), context.Background(), 8510001)
	if err != nil {
		t.Fatalf("upstreamFn returned error: %v", err)
	}
	if m.TagColor() != 9 {
		t.Errorf("TagColor=%d, want 9", m.TagColor())
	}
	if m.TagBackgroundColor() != 3 {
		t.Errorf("TagBackgroundColor=%d, want 3", m.TagBackgroundColor())
	}
}

// TestUpstreamFn_DecodesWithRelationshipsBlock asserts the api2go unmarshal
// succeeds when the served document includes a relationships block,
// exercising the SetToOneReferenceID / SetToManyReferenceIDs no-ops added to
// RestModel (EXT-01): without them, unmarshal errors the moment the upstream
// response grows a relationships member.
func TestUpstreamFn_DecodesWithRelationshipsBlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
  "data": {
    "type": "monsters",
    "id": "8510002",
    "attributes": { "boss": false, "fixed_damage": 0, "tag_color": 2, "tag_background_color": 4 },
    "relationships": {
      "drops": { "data": [ { "type": "drops", "id": "1" } ] }
    }
  }
}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	m, err := upstreamFn(logrus.New(), context.Background(), 8510002)
	if err != nil {
		t.Fatalf("upstreamFn returned error: %v", err)
	}
	if m.TagColor() != 2 {
		t.Errorf("TagColor=%d, want 2", m.TagColor())
	}
	if m.TagBackgroundColor() != 4 {
		t.Errorf("TagBackgroundColor=%d, want 4", m.TagBackgroundColor())
	}
}
