package character

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const charactersFixture = `{
  "data": [
    {
      "type": "characters",
      "id": "1",
      "attributes": {
        "accountId": 1000,
        "worldId": 0,
        "name": "Alpha",
        "level": 50,
        "experience": 1234,
        "jobId": 112,
        "gm": 0
      },
      "relationships": {
        "equipment": {"data": []},
        "inventories": {"data": []}
      }
    },
    {
      "type": "characters",
      "id": "2",
      "attributes": {
        "accountId": 1001,
        "worldId": 1,
        "name": "Beta",
        "level": 30,
        "experience": 55,
        "jobId": 0,
        "gm": 1
      },
      "relationships": {
        "equipment": {"data": []},
        "inventories": {"data": []}
      }
    }
  ]
}`

func TestGetAllDecodesTrimmedModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(charactersFixture))
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)

	cs, err := NewProcessor(logrus.New(), ctx).GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(cs) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(cs))
	}
	if cs[0].Id() != 1 || cs[0].Level() != 50 || cs[0].Experience() != 1234 || cs[0].JobId() != 112 || cs[0].Gm() != 0 {
		t.Fatalf("character 1 decoded wrong: %+v", cs[0])
	}
	if cs[1].WorldId() != 1 || cs[1].Gm() != 1 {
		t.Fatalf("character 2 decoded wrong: %+v", cs[1])
	}
}
