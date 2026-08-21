package playernpc

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/inventory"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	mapdata "atlas-player-npcs/data/map"
	npcdata "atlas-player-npcs/data/npc"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// -- HTTP-handler test setup, copied from
// services/atlas-notes/atlas.com/notes/note/resource_test.go's neighbours --

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupPlayerNpcRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitializeRoutes(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

// seedNpc inserts a Player NPC row directly (bypassing Deploy) for tests
// that only exercise GET/DELETE -- those paths never call the external
// read clients, so there is nothing to mock.
func seedNpc(t *testing.T, db *gorm.DB, tenantId uuid.UUID, worldId byte, mapId uint32, characterId uint32, name string, scriptId uint32, objectId uint32) Model {
	t.Helper()
	m, err := NewBuilder().
		SetCharacterId(characterId).
		SetName(name).
		SetWorldId(worldId).
		SetMapId(mapId).
		SetScriptId(scriptId).
		SetObjectId(objectId).
		SetGender(0).
		SetSkin(1).
		SetFace(20000).
		SetHair(30000).
		SetX(100).
		SetCy(200).
		SetFh(17).
		Build()
	if err != nil {
		t.Fatalf("building seed model: %v", err)
	}
	created, err := createPlayerNpc(db, tenantId, m)
	if err != nil {
		t.Fatalf("seeding player npc: %v", err)
	}
	return created
}

// -- external-service mocks -------------------------------------------------
//
// Deploy/Redeploy fan out to atlas-character, atlas-inventory,
// atlas-rankings and atlas-data (npcs + maps) through the same
// requests.RootUrlFor(<DOMAIN>_SERVICE_URL) lookup services/atlas-cashshop's
// coupon/resource_test.go uses (t.Setenv + an httptest.Server). Rankings
// gracefully 404s to the zero-value Model (ranking.Processor's own doc
// comment), so it is never mocked; atlas-tenants' player-npcs configuration
// falls back to configuration.DefaultModel() on any read error, so with no
// TENANTS_SERVICE_URL set at all it is never mocked either.
type externalMocks struct {
	mu         sync.Mutex
	characters map[uuid.UUID]map[uint32]character.RestModel
	// npcUsable[tenantId] == nil means every candidate id in the pool is
	// usable (the convenient default); a non-nil map restricts usability
	// to exactly the ids it lists true, for the pool-exhaustion test.
	npcUsable map[uuid.UUID]map[uint32]bool
	maps      map[uint32]mapdata.RestModel
}

func newExternalMocks(t *testing.T) *externalMocks {
	t.Helper()
	m := &externalMocks{
		characters: make(map[uuid.UUID]map[uint32]character.RestModel),
		npcUsable:  make(map[uuid.UUID]map[uint32]bool),
		maps:       make(map[uint32]mapdata.RestModel),
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/characters/{id}", m.handleCharacter).Methods(http.MethodGet)
	r.HandleFunc("/api/characters/{id}/inventory", m.handleInventory).Methods(http.MethodGet)
	r.HandleFunc("/api/rankings/characters/{id}", m.handleRanking).Methods(http.MethodGet)
	r.HandleFunc("/api/data/npcs/{id}", m.handleNpc).Methods(http.MethodGet)
	r.HandleFunc("/api/data/maps/{id}/ground", m.handleGround).Methods(http.MethodPost)
	r.HandleFunc("/api/data/maps/{id}", m.handleMap).Methods(http.MethodGet)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/api/")
	t.Setenv("RANKINGS_SERVICE_URL", srv.URL+"/api/")
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")

	return m
}

func (m *externalMocks) setCharacter(tenantId uuid.UUID, rm character.RestModel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.characters[tenantId] == nil {
		m.characters[tenantId] = make(map[uint32]character.RestModel)
	}
	m.characters[tenantId][rm.Id] = rm
}

func (m *externalMocks) restrictNpcPool(tenantId uuid.UUID, usable map[uint32]bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.npcUsable[tenantId] = usable
}

func (m *externalMocks) setMap(rm mapdata.RestModel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maps[rm.Id] = rm
}

func (m *externalMocks) handleCharacter(w http.ResponseWriter, r *http.Request) {
	tenantId, err := uuid.Parse(r.Header.Get(tenant.ID))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	rm, ok := m.characters[tenantId][uint32(id)]
	m.mu.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJsonApi(w, rm)
}

// handleInventory always answers with an empty compartment set -- deploy
// tests don't assert equipment, so there is nothing to seed.
func (m *externalMocks) handleInventory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	writeJsonApi(w, inventory.RestModel{Id: uuid.New(), CharacterId: uint32(id), Compartments: nil})
}

func (m *externalMocks) handleRanking(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func (m *externalMocks) handleNpc(w http.ResponseWriter, r *http.Request) {
	tenantId, err := uuid.Parse(r.Header.Get(tenant.ID))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	usable, restricted := m.npcUsable[tenantId]
	m.mu.Unlock()

	imitate := true
	if restricted {
		imitate = usable[uint32(id)]
	}
	writeJsonApi(w, npcdata.RestModel{Id: uint32(id), Name: "Statue", Imitate: imitate})
}

func (m *externalMocks) handleMap(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	rm, ok := m.maps[uint32(id)]
	m.mu.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJsonApi(w, rm)
}

func (m *externalMocks) handleGround(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req mapdata.GroundRequestRestModel
	if err := jsonapi.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	results := make([]mapdata.GroundResultRestModel, 0, len(req.Points))
	for i, pt := range req.Points {
		results = append(results, mapdata.GroundResultRestModel{Id: uint32(i), X: pt.X, Y: pt.Y, Fh: 1, Found: true})
	}
	writeJsonApi(w, results)
}

func writeJsonApi(w http.ResponseWriter, data interface{}) {
	body, err := jsonapi.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// -- shared errors-array decode ----------------------------------------------

type errorsDocument struct {
	Errors []struct {
		Status string `json:"status"`
		Code   string `json:"code"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

func decodeErrors(t *testing.T, body io.Reader) errorsDocument {
	t.Helper()
	var doc errorsDocument
	if err := json.NewDecoder(body).Decode(&doc); err != nil {
		t.Fatalf("decoding errors document: %v", err)
	}
	return doc
}

func mapWithId(id uint32, width, height int16) mapdata.RestModel {
	return mapdata.RestModel{Id: id, MapArea: &mapdata.RectangleRestModel{X: 0, Y: 0, Width: width, Height: height}}
}

func warriorCharacter(id uint32, name string, level byte) character.RestModel {
	return character.RestModel{Id: id, Name: name, Gender: 0, SkinColor: 1, Face: 20000, Hair: 30000, JobId: job.WarriorId, Level: level, Gm: 0}
}

// -- TestPlayerNpcResource ----------------------------------------------------

func TestPlayerNpcResource(t *testing.T) {
	mocks := newExternalMocks(t)
	db := testDatabase(t)
	srv := httptest.NewServer(setupPlayerNpcRouter(db))
	t.Cleanup(srv.Close)

	primaryTenantModel := testTenant(t)
	primaryTenant := primaryTenantModel.Id()
	secondaryTenant := uuid.New()
	poolTenant := uuid.New()

	t.Run("list by map", func(t *testing.T) {
		seedNpc(t, db, primaryTenant, 0, 102000004, 101, "ListHero1", 8800001, 1)
		seedNpc(t, db, primaryTenant, 0, 102000004, 102, "ListHero2", 8800002, 2)
		seedNpc(t, db, primaryTenant, 0, 999000001, 103, "OtherMapHero", 8800003, 3)

		url := fmt.Sprintf("%s/player-npcs?filter[mapId]=102000004&filter[worldId]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, primaryTenant)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var doc jsonapi.Document
		if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(doc.Data.DataArray) != 2 {
			t.Fatalf("len(data) = %d, want 2", len(doc.Data.DataArray))
		}
	})

	t.Run("list is tenant-scoped", func(t *testing.T) {
		seedNpc(t, db, primaryTenant, 0, 102000005, 111, "TenantAHero", 8800010, 10)
		seedNpc(t, db, secondaryTenant, 0, 102000005, 111, "TenantBHero1", 8800011, 11)
		seedNpc(t, db, secondaryTenant, 0, 102000005, 112, "TenantBHero2", 8800012, 12)

		url := fmt.Sprintf("%s/player-npcs?filter[mapId]=102000005&filter[worldId]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, secondaryTenant)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var doc jsonapi.Document
		if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(doc.Data.DataArray) != 2 {
			t.Fatalf("len(data) = %d, want 2 (disjoint from tenant A's row)", len(doc.Data.DataArray))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		seedNpc(t, db, primaryTenant, 0, 0, 121, "PageHero1", 8800020, 20)
		seedNpc(t, db, primaryTenant, 0, 0, 122, "PageHero2", 8800021, 21)

		url := fmt.Sprintf("%s/player-npcs?page[size]=1", srv.URL)
		req := requestWithTenant(http.MethodGet, url, primaryTenant)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var doc jsonapi.Document
		if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(doc.Data.DataArray) != 1 {
			t.Fatalf("len(data) = %d, want 1", len(doc.Data.DataArray))
		}
		if doc.Meta == nil {
			t.Fatalf("meta = nil, want the standard pagination envelope")
		}
	})

	t.Run("get one", func(t *testing.T) {
		created := seedNpc(t, db, primaryTenant, 0, 555000200, 131, "GetOneHero", 8800030, 30)

		url := fmt.Sprintf("%s/player-npcs/%s", srv.URL, created.Id().String())
		req := requestWithTenant(http.MethodGet, url, primaryTenant)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("get missing", func(t *testing.T) {
		url := fmt.Sprintf("%s/player-npcs/%s", srv.URL, uuid.New().String())
		req := requestWithTenant(http.MethodGet, url, primaryTenant)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}
	})

	deployBody := func(characterId uint32, worldId byte, mapId uint32, position *PositionRestModel) []byte {
		rm := DeployRestModel{CharacterId: characterId, WorldId: worldId, MapId: mapId, Position: position}
		body, err := jsonapi.Marshal(rm)
		if err != nil {
			t.Fatalf("marshal deploy body: %v", err)
		}
		return body
	}

	postDeploy := func(t *testing.T, tenantId uuid.UUID, body []byte) *http.Response {
		t.Helper()
		req := requestWithTenant(http.MethodPost, srv.URL+"/player-npcs", tenantId)
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		return resp
	}

	t.Run("deploy", func(t *testing.T) {
		mocks.setCharacter(primaryTenant, warriorCharacter(210, "DeployHero", 200))
		mocks.setMap(mapWithId(555000210, 1000, 1000))

		resp := postDeploy(t, primaryTenant, deployBody(210, 0, 555000210, nil))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want 201: %s", resp.StatusCode, string(b))
		}
		var single jsonapi.Document
		b, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(b, &single); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if single.Data.DataObject == nil {
			t.Fatalf("data = nil, want the created resource")
		}
	})

	t.Run("deploy with explicit position", func(t *testing.T) {
		mocks.setCharacter(primaryTenant, warriorCharacter(211, "PositionHero", 200))
		mocks.setMap(mapWithId(555000211, 100, 100))

		resp := postDeploy(t, primaryTenant, deployBody(211, 0, 555000211, &PositionRestModel{X: 5, Y: 7}))
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", resp.StatusCode, string(b))
		}
		var rm RestModel
		if err := jsonapi.Unmarshal(b, &rm); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if rm.X != 5 || rm.Cy != 7 {
			t.Fatalf("X()/Cy() = %d/%d, want 5/7 (explicit position used verbatim)", rm.X, rm.Cy)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		mocks.setCharacter(primaryTenant, warriorCharacter(212, "DupHero", 200))
		mocks.setMap(mapWithId(555000212, 1000, 1000))

		first := postDeploy(t, primaryTenant, deployBody(212, 0, 555000212, nil))
		first.Body.Close()
		if first.StatusCode != http.StatusCreated {
			t.Fatalf("first deploy status = %d, want 201", first.StatusCode)
		}

		second := postDeploy(t, primaryTenant, deployBody(212, 0, 555000212, nil))
		defer second.Body.Close()
		if second.StatusCode != http.StatusConflict {
			t.Fatalf("second deploy status = %d, want 409", second.StatusCode)
		}
		doc := decodeErrors(t, second.Body)
		if len(doc.Errors) != 1 || doc.Errors[0].Code != CodeDuplicate {
			t.Fatalf("errors = %+v, want code %q", doc.Errors, CodeDuplicate)
		}
	})

	t.Run("pool exhausted", func(t *testing.T) {
		mocks.setCharacter(poolTenant, warriorCharacter(213, "PoolHero", 200))
		mocks.setMap(mapWithId(555000213, 100, 100))
		mocks.restrictNpcPool(poolTenant, map[uint32]bool{9905000: true})

		occupant := buildDeployedNpc(t, 0, 555000213, "Occupant", 9905000, 1)
		if _, err := createPlayerNpc(db, poolTenant, occupant); err != nil {
			t.Fatalf("seeding occupant: %v", err)
		}

		resp := postDeploy(t, poolTenant, deployBody(213, 0, 555000213, nil))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusConflict {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want 409: %s", resp.StatusCode, string(b))
		}
		doc := decodeErrors(t, resp.Body)
		if len(doc.Errors) != 1 || doc.Errors[0].Code != CodePoolExhausted {
			t.Fatalf("errors = %+v, want code %q", doc.Errors, CodePoolExhausted)
		}
	})

	t.Run("map full", func(t *testing.T) {
		mocks.setCharacter(primaryTenant, warriorCharacter(214, "MapFullHero", 200))
		mocks.setMap(mapWithId(555000214, 100, 0)) // zero-height bounds -> no candidate slot at any step

		resp := postDeploy(t, primaryTenant, deployBody(214, 0, 555000214, nil))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusConflict {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want 409: %s", resp.StatusCode, string(b))
		}
		doc := decodeErrors(t, resp.Body)
		if len(doc.Errors) != 1 || doc.Errors[0].Code != CodeMapFull {
			t.Fatalf("errors = %+v, want code %q", doc.Errors, CodeMapFull)
		}
	})

	t.Run("ineligible", func(t *testing.T) {
		mocks.setCharacter(primaryTenant, warriorCharacter(215, "WeakHero", 10)) // below max level 200
		mocks.setMap(mapWithId(555000215, 100, 100))

		resp := postDeploy(t, primaryTenant, deployBody(215, 0, 555000215, nil))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusConflict {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want 409: %s", resp.StatusCode, string(b))
		}
		doc := decodeErrors(t, resp.Body)
		if len(doc.Errors) != 1 || doc.Errors[0].Code != CodeIneligible {
			t.Fatalf("errors = %+v, want code %q", doc.Errors, CodeIneligible)
		}
	})

	t.Run("unresolvable character", func(t *testing.T) {
		resp := postDeploy(t, primaryTenant, deployBody(999999, 0, 555000216, nil))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnprocessableEntity {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want 422: %s", resp.StatusCode, string(b))
		}
	})

	t.Run("re-deploy", func(t *testing.T) {
		mocks.setCharacter(primaryTenant, warriorCharacter(216, "RedeployHero", 200))
		mocks.setMap(mapWithId(555000217, 1000, 1000))

		created := postDeploy(t, primaryTenant, deployBody(216, 0, 555000217, nil))
		b, _ := io.ReadAll(created.Body)
		created.Body.Close()
		if created.StatusCode != http.StatusCreated {
			t.Fatalf("deploy status = %d, want 201: %s", created.StatusCode, string(b))
		}
		var before RestModel
		if err := jsonapi.Unmarshal(b, &before); err != nil {
			t.Fatalf("decode: %v", err)
		}

		req := requestWithTenant(http.MethodPatch, fmt.Sprintf("%s/player-npcs/%s", srv.URL, before.GetID()), primaryTenant)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		rb, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", resp.StatusCode, string(rb))
		}
		var after RestModel
		if err := jsonapi.Unmarshal(rb, &after); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if after.ScriptId != before.ScriptId || after.ObjectId != before.ObjectId {
			t.Fatalf("script/object id changed on redeploy: %+v -> %+v", before, after)
		}
		if after.X != before.X || after.Cy != before.Cy {
			t.Fatalf("position changed on redeploy: %+v -> %+v", before, after)
		}
	})

	t.Run("delete one", func(t *testing.T) {
		created := seedNpc(t, db, primaryTenant, 0, 555000220, 141, "DeleteOneHero", 8800040, 40)

		req := requestWithTenant(http.MethodDelete, fmt.Sprintf("%s/player-npcs/%s", srv.URL, created.Id().String()), primaryTenant)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
	})

	t.Run("delete by character", func(t *testing.T) {
		seedNpc(t, db, primaryTenant, 0, 555000221, 151, "ByCharHero", 8800041, 41)
		seedNpc(t, db, primaryTenant, 0, 555000222, 151, "ByCharHero", 8800042, 42)

		url := fmt.Sprintf("%s/player-npcs?filter[characterId]=151", srv.URL)
		req := requestWithTenant(http.MethodDelete, url, primaryTenant)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}

		remaining, err := entitiesByCharacter(db, 151, nil)
		if err != nil {
			t.Fatalf("checking remaining rows: %v", err)
		}
		if len(remaining) != 0 {
			t.Fatalf("remaining = %d, want 0 (every map)", len(remaining))
		}
	})

	t.Run("delete by character and map", func(t *testing.T) {
		seedNpc(t, db, primaryTenant, 0, 555000223, 161, "ByCharMapHero", 8800043, 43)
		seedNpc(t, db, primaryTenant, 0, 555000224, 161, "ByCharMapHero", 8800044, 44)

		url := fmt.Sprintf("%s/player-npcs?filter[characterId]=161&filter[mapId]=555000223", srv.URL)
		req := requestWithTenant(http.MethodDelete, url, primaryTenant)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}

		remaining, err := entitiesByCharacter(db, 161, nil)
		if err != nil {
			t.Fatalf("checking remaining rows: %v", err)
		}
		if len(remaining) != 1 || remaining[0].MapId != 555000224 {
			t.Fatalf("remaining = %+v, want exactly the other map's row", remaining)
		}
	})

	t.Run("eligibility endpoint", func(t *testing.T) {
		mocks.setCharacter(primaryTenant, warriorCharacter(171, "EligibleHero", 200))

		url := fmt.Sprintf("%s/player-npcs/eligibility?characterId=171&mapId=555000230", srv.URL)
		req := requestWithTenant(http.MethodGet, url, primaryTenant)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", resp.StatusCode, string(b))
		}
		var er eligibilityResponse
		if err := json.Unmarshal(b, &er); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !er.Eligible {
			t.Fatalf("eligibilityResponse = %+v, want eligible", er)
		}
	})
}
