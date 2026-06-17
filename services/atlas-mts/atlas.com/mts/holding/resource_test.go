package holding_test

import (
	"atlas-mts/holding"
	"atlas-mts/test"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type testServerInfo struct{}

func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t testServerInfo) GetPrefix() string  { return "/api" }

func newHoldingServer(t *testing.T, db *gorm.DB) *httptest.Server {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	router := mux.NewRouter()
	holding.InitResource(testServerInfo{})(db)(router, l)
	return httptest.NewServer(router)
}

func withTenant(t *testing.T, method, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("TENANT_ID", test.TestTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func seedHolding(t *testing.T, p holding.Processor, worldId world.Id, ownerId uint32) holding.Model {
	t.Helper()
	m, err := holding.NewBuilder(test.TestTenantId, worldId, ownerId).
		SetOrigin(holding.OriginPurchased).
		SetTemplateId(1302000).
		SetQuantity(1).
		Build()
	if err != nil {
		t.Fatalf("build holding: %v", err)
	}
	created, err := p.Create(m)
	if err != nil {
		t.Fatalf("create holding: %v", err)
	}
	return created
}

// TestGetCharacterHoldings asserts the holding read endpoint returns a JSON:API
// envelope of "holdings" scoped to the character, across worlds, and that the
// optional worldId query param narrows by world.
func TestGetCharacterHoldings(t *testing.T) {
	p, db, cleanup := test.CreateHoldingProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM holdings").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}

	seedHolding(t, p, 0, 100)
	seedHolding(t, p, 1, 100)
	seedHolding(t, p, 0, 200)

	srv := newHoldingServer(t, db)
	defer srv.Close()
	client := &http.Client{}

	// All holdings for character 100 across worlds => 2.
	resp, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/characters/100/mts/holding", srv.URL)))
	if err != nil {
		t.Fatalf("get holdings: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var env struct {
		Data []struct {
			Type       string `json:"type"`
			Attributes struct {
				OwnerId uint32 `json:"ownerId"`
				WorldId byte   `json:"worldId"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	resp.Body.Close()
	if len(env.Data) != 2 {
		t.Fatalf("holdings(c100) returned %d, want 2", len(env.Data))
	}
	for _, d := range env.Data {
		if d.Type != "holdings" {
			t.Errorf("type = %q, want holdings", d.Type)
		}
		if d.Attributes.OwnerId != 100 {
			t.Errorf("ownerId = %d, want 100", d.Attributes.OwnerId)
		}
	}

	// Narrow by worldId=0 => 1.
	resp2, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/characters/100/mts/holding?worldId=0", srv.URL)))
	if err != nil {
		t.Fatalf("get holdings w0: %v", err)
	}
	var env2 struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&env2); err != nil {
		t.Fatalf("decode w0: %v", err)
	}
	resp2.Body.Close()
	if len(env2.Data) != 1 {
		t.Fatalf("holdings(c100, w0) returned %d, want 1", len(env2.Data))
	}
}
