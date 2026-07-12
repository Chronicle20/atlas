package wish_test

import (
	"atlas-mts/test"
	"atlas-mts/wish"
	"bytes"
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

func newWishServer(t *testing.T, db *gorm.DB) *httptest.Server {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	router := mux.NewRouter()
	wish.InitResource(testServerInfo{})(db)(router, l)
	return httptest.NewServer(router)
}

func withTenant(t *testing.T, method, url string, body []byte) *http.Request {
	t.Helper()
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
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

// TestWishCRUD asserts the wish CRUD lifecycle: POST (with JSON:API envelope)
// creates an entry, GET lists it, and DELETE removes it.
func TestWishCRUD(t *testing.T) {
	_, db, cleanup := test.CreateWishProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM wish_entries").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}

	srv := newWishServer(t, db)
	defer srv.Close()
	client := &http.Client{}

	// POST a wish via the JSON:API envelope.
	envelope := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "wish-entries",
			"attributes": map[string]interface{}{
				"itemId": 1302000,
			},
		},
	}
	body, _ := json.Marshal(envelope)
	resp, err := client.Do(withTenant(t, http.MethodPost, fmt.Sprintf("%s/characters/100/mts/wishlist", srv.URL), body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		Data struct {
			Type       string `json:"type"`
			Id         string `json:"id"`
			Attributes struct {
				CharacterId uint32 `json:"characterId"`
				ItemId      uint32 `json:"itemId"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	resp.Body.Close()
	if created.Data.Type != "wish-entries" {
		t.Errorf("type = %q, want wish-entries", created.Data.Type)
	}
	if created.Data.Attributes.CharacterId != 100 {
		t.Errorf("characterId = %d, want 100 (from path)", created.Data.Attributes.CharacterId)
	}
	if created.Data.Attributes.ItemId != 1302000 {
		t.Errorf("itemId = %d, want 1302000", created.Data.Attributes.ItemId)
	}
	if created.Data.Id == "" {
		t.Fatal("create did not assign an id")
	}

	// GET the character's wishlist => 1 entry.
	respGet, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/characters/100/mts/wishlist", srv.URL), nil))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var listEnv struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respGet.Body).Decode(&listEnv); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	respGet.Body.Close()
	if len(listEnv.Data) != 1 {
		t.Fatalf("wishlist returned %d, want 1", len(listEnv.Data))
	}

	// DELETE the wish entry => 204; it then no longer lists.
	respDel, err := client.Do(withTenant(t, http.MethodDelete, fmt.Sprintf("%s/characters/100/mts/wishlist/%s", srv.URL, created.Data.Id), nil))
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if respDel.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", respDel.StatusCode)
	}
	respDel.Body.Close()

	respGet2, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/characters/100/mts/wishlist", srv.URL), nil))
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	var listEnv2 struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(respGet2.Body).Decode(&listEnv2); err != nil {
		t.Fatalf("decode list after delete: %v", err)
	}
	respGet2.Body.Close()
	if len(listEnv2.Data) != 0 {
		t.Errorf("wishlist after delete returned %d, want 0", len(listEnv2.Data))
	}
}

// TestWorldWishlistCrossCharacter asserts GET /worlds/{worldId}/mts/wishlist
// returns every want-ad in the world across all characters (excluding cart
// entries and other worlds' want-ads). Want-ads are seeded through the processor
// because the POST create path always defaults type=cart.
func TestWorldWishlistCrossCharacter(t *testing.T) {
	p, db, cleanup := test.CreateWishProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM wish_entries").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}

	seed := func(worldId byte, characterId uint32, itemId uint32, wishType string) {
		m, err := wish.NewBuilder(test.TestTenantId, characterId, itemId).
			SetWorldId(world.Id(worldId)).
			SetType(wishType).
			Build()
		if err != nil {
			t.Fatalf("build seed: %v", err)
		}
		if _, err := p.Create(m); err != nil {
			t.Fatalf("seed create: %v", err)
		}
	}
	seed(0, 100, 1302000, wish.TypeWanted)
	seed(0, 101, 1302001, wish.TypeWanted)
	seed(0, 102, 1302002, wish.TypeCart)   // cart entry: must not surface
	seed(1, 103, 1302003, wish.TypeWanted) // other world: must not surface

	srv := newWishServer(t, db)
	defer srv.Close()
	client := &http.Client{}

	resp, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/worlds/0/mts/wishlist", srv.URL), nil))
	if err != nil {
		t.Fatalf("get world wishlist: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("world wishlist status = %d, want 200", resp.StatusCode)
	}
	var env struct {
		Data []struct {
			Attributes struct {
				CharacterId uint32 `json:"characterId"`
				ItemId      uint32 `json:"itemId"`
				Type        string `json:"type"`
				WorldId     byte   `json:"worldId"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode world wishlist: %v", err)
	}
	resp.Body.Close()
	if len(env.Data) != 2 {
		t.Fatalf("world wishlist returned %d, want 2 (cross-character want-ads only)", len(env.Data))
	}
	for _, d := range env.Data {
		if d.Attributes.Type != wish.TypeWanted {
			t.Errorf("world wishlist returned a non-wanted entry (type=%s)", d.Attributes.Type)
		}
		if d.Attributes.WorldId != 0 {
			t.Errorf("world wishlist returned a world-%d entry, want 0", d.Attributes.WorldId)
		}
		if d.Attributes.CharacterId != 100 && d.Attributes.CharacterId != 101 {
			t.Errorf("world wishlist returned unexpected character %d", d.Attributes.CharacterId)
		}
	}
}

// TestWishDeleteMalformedId is the resource-level regression guard for the GORM
// zero-id elision bug: a DELETE with a non-UUID wishId must be rejected
// (400/404), and must NOT wipe the character's wishlist.
func TestWishDeleteMalformedId(t *testing.T) {
	_, db, cleanup := test.CreateWishProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM wish_entries").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}

	srv := newWishServer(t, db)
	defer srv.Close()
	client := &http.Client{}

	// Seed 3 wishes via the API.
	for i := 0; i < 3; i++ {
		envelope := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "wish-entries",
				"attributes": map[string]interface{}{
					"itemId": 1302000 + i,
				},
			},
		}
		body, _ := json.Marshal(envelope)
		resp, err := client.Do(withTenant(t, http.MethodPost, fmt.Sprintf("%s/characters/100/mts/wishlist", srv.URL), body))
		if err != nil {
			t.Fatalf("create #%d: %v", i, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create #%d status = %d, want 201", i, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// DELETE with a malformed wishId must be rejected, not a nil-delete.
	respDel, err := client.Do(withTenant(t, http.MethodDelete, fmt.Sprintf("%s/characters/100/mts/wishlist/not-a-uuid", srv.URL), nil))
	if err != nil {
		t.Fatalf("delete malformed: %v", err)
	}
	if respDel.StatusCode != http.StatusBadRequest && respDel.StatusCode != http.StatusNotFound {
		t.Errorf("delete malformed status = %d, want 400 or 404", respDel.StatusCode)
	}
	respDel.Body.Close()

	// The wishlist must be intact.
	respGet, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/characters/100/mts/wishlist", srv.URL), nil))
	if err != nil {
		t.Fatalf("list after malformed delete: %v", err)
	}
	var listEnv struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(respGet.Body).Decode(&listEnv); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	respGet.Body.Close()
	if len(listEnv.Data) != 3 {
		t.Errorf("wishlist after malformed delete returned %d, want 3 (all survive)", len(listEnv.Data))
	}
}
