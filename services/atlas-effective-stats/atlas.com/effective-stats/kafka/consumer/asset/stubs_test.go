package asset

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	character2 "atlas-effective-stats/character"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// stubCharacter mirrors the subset of atlas-character REST fields the
// initializer reads.
type stubCharacter struct {
	level               byte
	jobId               uint16
	str, dex, intl, luk uint16
	maxHp, maxMp        uint16
}

// stubEquipped is the per-asset shape the inventory stub renders into a
// JSON:API compartment+included document.
type stubEquipped struct {
	assetId             uint32
	templateId          uint32
	slot                int16
	str, dex, intl, luk uint16
	hp, mp              uint16
	wAtk, mAtk          uint16
}

// equipmentReqs holds the requirement gating fields atlas-data exposes for
// each equipment template.
type equipmentReqs struct {
	reqLevel                       byte
	reqJob                         uint16
	reqStr, reqDex, reqInt, reqLuk uint16
}

// stubConfig is the integration-test payload describing what each stub server
// should answer with for a single InitializeCharacter call.
type stubConfig struct {
	character     stubCharacter
	equipped      []stubEquipped
	equipmentReqs map[uint32]equipmentReqs
}

// stubServers holds the five httptest servers backing the initializer's
// upstream calls. Close() shuts them all down; PointEnv() configures the
// *_SERVICE_URL env vars consulted by libs/atlas-rest/requests.RootUrl.
type stubServers struct {
	character *httptest.Server
	inventory *httptest.Server
	data      *httptest.Server
	buffs     *httptest.Server
	skills    *httptest.Server
}

func (s *stubServers) Close() {
	s.character.Close()
	s.inventory.Close()
	s.data.Close()
	s.buffs.Close()
	s.skills.Close()
}

// PointEnv sets the *_SERVICE_URL env vars consulted by requests.RootUrl
// (libs/atlas-rest/requests/url.go) so the initializer talks to our stubs
// instead of real services.
//
// Trailing-slash handling matches the model package's stubs harness — see
// character/stubs_test.go for the per-client breakdown.
func (s *stubServers) PointEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CHARACTERS_SERVICE_URL", s.character.URL+"/")
	t.Setenv("INVENTORY_SERVICE_URL", s.inventory.URL+"/")
	t.Setenv("DATA_SERVICE_URL", s.data.URL)
	t.Setenv("BUFFS_SERVICE_URL", s.buffs.URL+"/")
	t.Setenv("SKILLS_SERVICE_URL", s.skills.URL+"/")
}

// startInitializerStubs spins up five httptest servers — one per upstream the
// initializer touches — and returns a stubServers handle ready for PointEnv()
// + Close(). Handlers are deliberately path-agnostic: each pulls the trailing
// numeric segment so URL-template drift between services doesn't silently
// 404 the test.
func startInitializerStubs(t *testing.T, cfg stubConfig) *stubServers {
	t.Helper()

	character := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := lastSegment(r.URL.Path)
		id, _ := strconv.Atoi(idStr)
		writeJSONAPI(w, map[string]interface{}{
			"data": map[string]interface{}{
				"type": "characters",
				"id":   strconv.Itoa(id),
				"attributes": map[string]interface{}{
					"level":        cfg.character.level,
					"jobId":        cfg.character.jobId,
					"strength":     cfg.character.str,
					"dexterity":    cfg.character.dex,
					"intelligence": cfg.character.intl,
					"luck":         cfg.character.luk,
					"maxHp":        cfg.character.maxHp,
					"maxMp":        cfg.character.maxMp,
					"hp":           cfg.character.maxHp,
					"mp":           cfg.character.maxMp,
				},
			},
		})
	}))

	inventory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		assets := make([]map[string]interface{}, 0, len(cfg.equipped))
		included := make([]map[string]interface{}, 0, len(cfg.equipped))
		for _, e := range cfg.equipped {
			idStr := strconv.FormatUint(uint64(e.assetId), 10)
			assets = append(assets, map[string]interface{}{
				"type": "assets",
				"id":   idStr,
			})
			included = append(included, map[string]interface{}{
				"type": "assets",
				"id":   idStr,
				"attributes": map[string]interface{}{
					"slot":          e.slot,
					"templateId":    e.templateId,
					"strength":      e.str,
					"dexterity":     e.dex,
					"intelligence":  e.intl,
					"luck":          e.luk,
					"hp":            e.hp,
					"mp":            e.mp,
					"weaponAttack":  e.wAtk,
					"magicAttack":   e.mAtk,
					"weaponDefense": 0,
					"magicDefense":  0,
					"accuracy":      0,
					"avoidability":  0,
					"hands":         0,
					"speed":         0,
					"jump":          0,
				},
			})
		}
		writeJSONAPI(w, map[string]interface{}{
			"data": map[string]interface{}{
				"type": "compartments",
				"id":   "1",
				"attributes": map[string]interface{}{
					"type":     1,
					"capacity": 24,
				},
				"relationships": map[string]interface{}{
					"assets": map[string]interface{}{"data": assets},
				},
			},
			"included": included,
		})
	}))

	data := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := lastSegment(r.URL.Path)
		id, _ := strconv.ParseUint(idStr, 10, 32)
		reqs, ok := cfg.equipmentReqs[uint32(id)]
		if !ok {
			http.NotFound(w, r)
			return
		}
		// Mirror the real atlas-data wire format: type "statistics" plus a
		// "slots" toMany relationship. The slot data itself is irrelevant to
		// requirement gating (the qualification engine never reads slots), but
		// the relationship MUST be present for api2go.Unmarshal to succeed
		// against the production RestModel's UnmarshalToManyRelations
		// implementation. See external/data/equipment/rest.go for the
		// full rationale.
		writeJSONAPI(w, map[string]interface{}{
			"data": map[string]interface{}{
				"type": "statistics",
				"id":   idStr,
				"attributes": map[string]interface{}{
					"reqLevel": reqs.reqLevel,
					"reqJob":   reqs.reqJob,
					"reqStr":   reqs.reqStr,
					"reqDex":   reqs.reqDex,
					"reqInt":   reqs.reqInt,
					"reqLuk":   reqs.reqLuk,
				},
				"relationships": map[string]interface{}{
					"slots": map[string]interface{}{
						"data": []interface{}{},
					},
				},
			},
			"included": []interface{}{},
		})
	}))

	buffs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSONAPI(w, map[string]interface{}{"data": []interface{}{}})
	}))

	skills := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSONAPI(w, map[string]interface{}{"data": []interface{}{}})
	}))

	return &stubServers{
		character: character,
		inventory: inventory,
		data:      data,
		buffs:     buffs,
		skills:    skills,
	}
}

// lastSegment returns the substring after the final '/' in path. Used to pull
// {id} out of patterns like `/characters/12345` or `/data/equipment/1052095`
// without baking exact route templates into the stub handlers.
func lastSegment(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}

// writeJSONAPI marshals body as application/vnd.api+json. Any encoding error
// is intentionally swallowed — httptest connections close on test exit and
// no production code path depends on this.
func writeJSONAPI(w http.ResponseWriter, body map[string]interface{}) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	_ = json.NewEncoder(w).Encode(body)
}

// setupAssetTest wires a fresh miniredis-backed character registry for the
// duration of the test. Mirrors setupCharacterTest in the consumer/character
// package so asset-consumer tests can drive the same Get/Put paths the
// initializer exercises in production.
func setupAssetTest(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	character2.InitRegistry(client)
}

// createTestContext returns a no-op logger plus a tenant-scoped context
// matching the consumer/character helper. Asset-consumer tests need both
// because handleAssetMoved threads ctx through the registry and the processor.
func createTestContext() (logrus.FieldLogger, context.Context, tenant.Model) {
	l, _ := test.NewNullLogger()
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), t)
	return l, ctx, t
}

// createTestContextWithHook is like createTestContext but also returns the
// logrus test hook so tests can assert on logged entries (e.g. WARN messages
// emitted by handleItemEquipped's diagnostic paths).
func createTestContextWithHook() (logrus.FieldLogger, context.Context, tenant.Model, *test.Hook) {
	l, hook := test.NewNullLogger()
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), t)
	return l, ctx, t, hook
}
