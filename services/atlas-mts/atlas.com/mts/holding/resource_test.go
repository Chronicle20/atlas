package holding_test

import (
	"atlas-mts/holding"
	"atlas-mts/test"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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

func withTenantBody(t *testing.T, method, url string, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", test.TestTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

// takeHomeBody builds a JSON:API envelope for the take-home POST.
func takeHomeBody(t *testing.T, inventoryType byte, slot int16) []byte {
	t.Helper()
	env := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "holdings",
			"attributes": map[string]interface{}{
				"inventoryType": inventoryType,
				"slot":          slot,
			},
		},
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal take-home body: %v", err)
	}
	return b
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

// TestTakeHomeRoute_NonOwnerForbidden asserts a character who is not the
// holding's owner cannot take it home: 403 Forbidden, and the saga is never
// reached (the owner check short-circuits before emission).
func TestTakeHomeRoute_NonOwnerForbidden(t *testing.T) {
	p, db, cleanup := test.CreateHoldingProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM holdings").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}

	const ownerId = uint32(6660001)
	const intruderId = uint32(6669999)
	created := seedHolding(t, p, 0, ownerId)

	srv := newHoldingServer(t, db)
	defer srv.Close()
	client := &http.Client{}

	url := fmt.Sprintf("%s/characters/%d/mts/holding/%s/take-home", srv.URL, intruderId, created.Id().String())
	resp, err := client.Do(withTenantBody(t, http.MethodPost, url, takeHomeBody(t, 1, 0)))
	if err != nil {
		t.Fatalf("take-home: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-owner take-home status = %d, want 403", resp.StatusCode)
	}

	// The holding row must remain present — a forbidden take-home releases nothing.
	rows, err := p.GetByOwner(0, ownerId)
	if err != nil {
		t.Fatalf("GetByOwner: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected the holding to remain after a forbidden take-home, got %d rows", len(rows))
	}
}

// TestTakeHomeRoute_NotFound asserts a take-home of a non-existent holding is a
// 404, not a panic or a 500.
func TestTakeHomeRoute_NotFound(t *testing.T) {
	_, db, cleanup := test.CreateHoldingProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM holdings").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}

	srv := newHoldingServer(t, db)
	defer srv.Close()
	client := &http.Client{}

	missing := "11111111-1111-1111-1111-111111111111"
	url := fmt.Sprintf("%s/characters/100/mts/holding/%s/take-home", srv.URL, missing)
	resp, err := client.Do(withTenantBody(t, http.MethodPost, url, takeHomeBody(t, 1, 0)))
	if err != nil {
		t.Fatalf("take-home: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing holding take-home status = %d, want 404", resp.StatusCode)
	}
}
