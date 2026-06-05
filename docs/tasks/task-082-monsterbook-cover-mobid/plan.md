# Monster-Book Cover — Encode Mob ID in Character-Info (Crash Fix) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **Path convention:** `<repo-root>` below means the task worktree root (the `.worktrees/task-082-monsterbook-cover-mobid/` checkout). All `cd <repo-root>` / `cd <repo-root>/services/...` commands run inside that worktree. Substitute your absolute worktree path when running.

**Goal:** Stop the v83 client from crashing when Character Info is opened on a character with a monster-book cover set, by sending the cover card's **mob id** (not the card item id) in the Character-Info packet.

**Architecture:** Set-time resolution in atlas-monster-book. When a cover is set, atlas-monster-book resolves the cover **card item id → mob id** via a new outbound atlas-data consumable client and persists `cover_mob_id` alongside the existing `cover_card_id`. The resolved mob id is exposed on the existing `GET /characters/{id}/monster-book` REST model; atlas-channel — which already fetches that model per Character-Info view — writes the mob id into the Character-Info packet's cover field. Every other cover surface (`0x39` set request, `0x54` set response, monster-book window, card list, login-draw `CharacterData`) stays in card-id space, unchanged.

**Tech Stack:** Go 1.25, GORM (sqlite in tests / postgres in prod), `api2go/jsonapi` JSON:API transport, `libs/atlas-rest/requests` outbound HTTP, `libs/atlas-kafka` messaging, `libs/atlas-packet` wire encoding, multi-tenancy via `libs/atlas-tenant` context.

---

## Context & Ground Rules

- **Worktree:** All work happens in the `task-082-monsterbook-cover-mobid` worktree on branch `task-082-monsterbook-cover-mobid`. Every subagent prompt must `cd` into this worktree first and verify the branch after each commit.
- **Module roots (each has its own `go.mod`):**
  - `services/atlas-monster-book/atlas.com/monster-book/` (module `atlas-monster-book`)
  - `services/atlas-channel/atlas.com/channel/` (module `atlas-channel`)
  - `libs/atlas-packet/` (module `github.com/Chronicle20/atlas/libs/atlas-packet`)
- **Test helpers:** Use the project Builder pattern. Do NOT create `*_testhelpers.go` files. In-package tests may construct struct literals directly and define local fakes.
- **JSON:API stubs are mandatory** for every new external client RestModel (`SetToOneReferenceID` / `SetToManyReferenceIDs` no-ops) — see `libs/atlas-rest/CLAUDE.md`. Every new external client also needs an httptest-backed integration test serving a fixture that includes a `relationships` block (FakeClient mocks bypass the unmarshal path and won't catch a missing stub).
- **No new shared lib** is introduced. `atlas-rest` is already a `replace` directive in atlas-monster-book's `go.mod` and is already `COPY`'d in the repo-root `Dockerfile` (lines 42, 71). Adding the consumable client pulls `atlas-rest` into the **require** block, so `atlas-monster-book/go.mod` changes → `docker buildx bake atlas-monster-book` is mandatory (Task 11). `atlas-channel` and `libs/atlas-packet` get source-only changes (no `go.mod` change).
- **Reference files (read before editing the analogous file):**
  - Canonical consumable client: `services/atlas-npc-shops/atlas.com/npc/data/consumable/{requests,processor,rest,model}.go`
  - atlas-data consumable response shape (source of `monsterBook` / `monsterId`): `services/atlas-data/atlas.com/data/consumable/rest.go:44-106`
  - httptest pattern + `SetBaseURLForTest` helper: `services/atlas-channel/atlas.com/channel/monsterbook/{requests.go,rest_test.go}`

---

## File Structure

**atlas-monster-book** (`services/atlas-monster-book/atlas.com/monster-book/`)
- Create `data/consumable/model.go` — minimal immutable `Model` (`MonsterBook() bool`, `MonsterId() uint32`).
- Create `data/consumable/rest.go` — `RestModel` (partial), JSON:API plumbing + stubs, `Extract`.
- Create `data/consumable/requests.go` — `Resource`/`ById`, swappable `baseURLProvider`, `requestById`, `SetBaseURLForTest`.
- Create `data/consumable/processor.go` — `Processor` interface + `ProcessorImpl{l, ctx}`, `NewProcessor`, `GetById`.
- Create `data/consumable/rest_test.go` — unmarshal-with-relationships test + httptest round-trip + 404 test.
- Modify `collection/entity.go` — add `CoverMobId` column.
- Modify `collection/model.go` — add `coverMobId` field, getter, `ToEntity` mapping.
- Modify `collection/builder.go` — add field, `SetCoverMobId`, thread through `CloneModelBuilder`/`Build`/`Make`.
- Modify `collection/administrator.go` — `setCover` gains `coverMobId` param, persists `cover_mob_id`.
- Modify `collection/processor.go` — add `dp consumable.Processor`, `resolveCoverMobId`, thread into `SetCoverAndEmit`.
- Modify `collection/rest.go` — `RestModel` gains `CoverMonsterId`, `Transform` sets it.
- Tests: `collection/builder_test.go`, `collection/administrator_test.go`, `collection/processor_test.go`, `collection/rest_test.go` (new file).

**atlas-channel** (`services/atlas-channel/atlas.com/channel/`)
- Modify `monsterbook/processor.go` — `Collection` gains `coverMonsterId` field + `CoverMonsterId()` getter.
- Modify `monsterbook/rest.go` — `CollectionRestModel` gains `CoverMonsterId`, `Extract` maps it.
- Modify `monsterbook/model.go` — `Model` gains `CoverMonsterId()` delegating to `Collection`.
- Modify `socket/writer/character_info.go` — `MonsterBookInfo.Cover = mb.CoverMonsterId()`.
- Tests: `monsterbook/rest_test.go`, `socket/writer/character_info_test.go` (new file).

**libs/atlas-packet** (`libs/atlas-packet/`)
- `character/clientbound/info.go` — **no change** (semantic value only). Add a contract-guard test to `character/clientbound/info_test.go`.
- `character/data.go` — **no change** (FR-10: login-draw stays card id). `character/data_test.go` regression guard already exists.

**deploy** (`deploy/k8s/base/atlas-monster-book.yaml`)
- Add `DATA_SERVICE_URL` env var (documents the new dependency; functionally optional via `BASE_SERVICE_URL` fallback).

---

## Task 1: atlas-monster-book — new `data/consumable` outbound client

Mirrors the canonical `services/atlas-npc-shops/atlas.com/npc/data/consumable/` client but with a **partial** `RestModel` (only the two fields the resolver reads) and a swappable `baseURLProvider` so the httptest test can redirect it.

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/data/consumable/model.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/data/consumable/requests.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/data/consumable/processor.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest_test.go`

- [ ] **Step 1: Write the model**

`data/consumable/model.go`:

```go
package consumable

// Model is the minimal immutable view of an atlas-data consumable that the
// monster-book cover resolver needs: whether the item is a monster-book card
// and, if so, the mob id the card represents.
type Model struct {
	monsterBook bool
	monsterId   uint32
}

func (m Model) MonsterBook() bool { return m.monsterBook }
func (m Model) MonsterId() uint32 { return m.monsterId }
```

- [ ] **Step 2: Write the rest model + JSON:API plumbing + Extract**

`data/consumable/rest.go` (partial RestModel — the JSON:API unmarshaller ignores attributes the struct does not declare; the reference stubs are mandatory per `libs/atlas-rest/CLAUDE.md`):

```go
package consumable

import (
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel is a partial view of atlas-data's "consumables" resource. Only the
// fields the cover resolver reads are declared; any other attributes in the
// response are ignored by the JSON:API unmarshaller.
type RestModel struct {
	Id          uint32 `json:"-"`
	MonsterBook bool   `json:"monsterBook"`
	MonsterId   uint32 `json:"monsterId"`
}

func (r RestModel) GetName() string { return "consumables" }

func (r RestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *RestModel) SetID(id string) error {
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

// JSON:API reference interface methods. Required even though this resource has
// no relationships we consume: api2go.Unmarshal errors out walking any
// `relationships` block unless these exist (libs/atlas-rest/CLAUDE.md).
func (r RestModel) GetReferences() []jsonapi.Reference                { return []jsonapi.Reference{} }
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID           { return []jsonapi.ReferenceID{} }
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error     { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// Extract converts the wire model into the immutable domain Model.
func Extract(rm RestModel) (Model, error) {
	return Model{monsterBook: rm.MonsterBook, monsterId: rm.MonsterId}, nil
}
```

- [ ] **Step 3: Write the requests file with a swappable base URL**

`data/consumable/requests.go`:

```go
package consumable

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// Resource is the atlas-data consumable collection path.
	Resource = "data/consumables"
	// ById is the single-consumable path template.
	ById = Resource + "/%d"
)

var baseURLProvider = func() string {
	return requests.RootUrl("DATA")
}

func getBaseRequest() string {
	return baseURLProvider()
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrl("DATA") default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
```

- [ ] **Step 4: Write the processor**

`data/consumable/processor.go`:

```go
package consumable

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(itemId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

// GetById fetches a consumable by item id. Tenant/region/version headers are
// propagated automatically by requests.GetRequest (TenantHeaderDecorator), so
// the lookup is correctly tenant- and version-scoped.
func (p *ProcessorImpl) GetById(itemId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(itemId), Extract)()
}
```

- [ ] **Step 5: Write the failing tests**

`data/consumable/rest_test.go`. The fixture intentionally includes a `relationships` block AND extra attributes to prove (a) the stubs work and (b) unknown attributes are ignored:

```go
package consumable

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// Proves the JSON:API stubs let Unmarshal succeed even with a relationships
// block present, and that unknown attributes are ignored.
func TestRestModel_UnmarshalWithRelationships(t *testing.T) {
	body := []byte(`{
		"data": {
			"type": "consumables",
			"id": "2380000",
			"attributes": {
				"monsterBook": true,
				"monsterId": 100100,
				"slotMax": 1,
				"price": 0
			},
			"relationships": {
				"rewards": { "data": [] }
			}
		}
	}`)
	var rm RestModel
	if err := jsonapi.Unmarshal(body, &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}
	if rm.Id != 2380000 {
		t.Errorf("Id = %d, want 2380000", rm.Id)
	}
	if !rm.MonsterBook {
		t.Errorf("MonsterBook = false, want true")
	}
	if rm.MonsterId != 100100 {
		t.Errorf("MonsterId = %d, want 100100", rm.MonsterId)
	}
}

func TestGetById_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/data/consumables/2380000") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "consumables",
				"id": "2380000",
				"attributes": { "monsterBook": true, "monsterId": 100100 },
				"relationships": { "rewards": { "data": [] } }
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	m, err := NewProcessor(logrus.New(), ctx).GetById(2380000)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if !m.MonsterBook() || m.MonsterId() != 100100 {
		t.Fatalf("model = %+v, want monsterBook=true monsterId=100100", m)
	}
}

func TestGetById_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	_, err := NewProcessor(logrus.New(), ctx).GetById(2380000)
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("expected requests.ErrNotFound, got %T: %v", err, err)
	}
}
```

- [ ] **Step 6: Run go mod tidy, then the tests**

Run:
```bash
cd <repo-root>/services/atlas-monster-book/atlas.com/monster-book && go mod tidy && go test ./data/consumable/ -v
```
Expected: `atlas-rest` is added to the `require` block of `go.mod`; all three tests PASS.

- [ ] **Step 7: Commit**

```bash
cd <repo-root>
git add services/atlas-monster-book/atlas.com/monster-book/data/consumable/ services/atlas-monster-book/atlas.com/monster-book/go.mod services/atlas-monster-book/atlas.com/monster-book/go.sum
git commit -m "feat(monster-book): add atlas-data consumable client for cover mob-id resolution"
git branch --show-current   # must print task-082-monsterbook-cover-mobid
```

---

## Task 2: collection — add `CoverMobId` to entity, model, builder

**Files:**
- Modify: `services/atlas-monster-book/atlas.com/monster-book/collection/entity.go`
- Modify: `services/atlas-monster-book/atlas.com/monster-book/collection/model.go`
- Modify: `services/atlas-monster-book/atlas.com/monster-book/collection/builder.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/collection/builder_test.go`

- [ ] **Step 1: Write the failing test**

Add to `collection/builder_test.go` (read the file first; append this new function):

```go
func TestBuilderCoverMobIdRoundTrip(t *testing.T) {
	m := NewModelBuilder().
		SetCharacterId(1).
		SetCoverCardId(2380000).
		SetCoverMobId(100100).
		MustBuild()

	if m.CoverMobId() != 100100 {
		t.Fatalf("Model.CoverMobId() = %d, want 100100", m.CoverMobId())
	}

	e := m.ToEntity()
	if e.CoverMobId != 100100 {
		t.Fatalf("entity.CoverMobId = %d, want 100100", e.CoverMobId)
	}

	back, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if back.CoverMobId() != 100100 || back.CoverCardId() != 2380000 {
		t.Fatalf("Make round-trip: mobId=%d cardId=%d", back.CoverMobId(), back.CoverCardId())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd <repo-root>/services/atlas-monster-book/atlas.com/monster-book && go test ./collection/ -run TestBuilderCoverMobIdRoundTrip`
Expected: compile error — `SetCoverMobId`, `CoverMobId`, and `entity.CoverMobId` are undefined.

- [ ] **Step 3: Add the entity column**

In `collection/entity.go`, add the field after `CoverCardId`:

```go
	CoverCardId      uint32     `gorm:"not null;default:0"`
	CoverMobId       uint32     `gorm:"not null;default:0"`
```

(`Migration` already runs `AutoMigrate(&entity{})`, which adds the column with default `0` — no manual migration.)

- [ ] **Step 4: Add the model field, getter, and ToEntity mapping**

In `collection/model.go`, add to the `Model` struct (after `coverCardId`):

```go
	coverCardId      item.Id
	coverMobId       uint32
```

Add the getter (next to `CoverCardId()`):

```go
func (m Model) CoverMobId() uint32 { return m.coverMobId }
```

Add to `ToEntity()` (after `CoverCardId`):

```go
		CoverCardId:      uint32(m.coverCardId),
		CoverMobId:       m.coverMobId,
```

- [ ] **Step 5: Add the builder field, setter, and thread it through**

In `collection/builder.go`:

Add to the `ModelBuilder` struct (after `coverCardId`):

```go
	coverCardId      item.Id
	coverMobId       uint32
```

Add to `CloneModelBuilder` (after `coverCardId`):

```go
		coverCardId:      m.coverCardId,
		coverMobId:       m.coverMobId,
```

Add the setter (next to `SetCoverCardId`):

```go
func (b *ModelBuilder) SetCoverMobId(v uint32) *ModelBuilder         { b.coverMobId = v; return b }
```

Add to `Build()`'s returned `Model{}` (after `coverCardId`):

```go
		coverCardId:      b.coverCardId,
		coverMobId:       b.coverMobId,
```

Add to `Make()` (after `SetCoverCardId(...)`):

```go
		SetCoverCardId(item.Id(e.CoverCardId)).
		SetCoverMobId(e.CoverMobId).
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `cd <repo-root>/services/atlas-monster-book/atlas.com/monster-book && go test ./collection/ -run TestBuilderCoverMobIdRoundTrip`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
cd <repo-root>
git add services/atlas-monster-book/atlas.com/monster-book/collection/entity.go services/atlas-monster-book/atlas.com/monster-book/collection/model.go services/atlas-monster-book/atlas.com/monster-book/collection/builder.go services/atlas-monster-book/atlas.com/monster-book/collection/builder_test.go
git commit -m "feat(monster-book): add cover_mob_id to collection entity/model/builder"
git branch --show-current
```

---

## Task 3: collection — `setCover` persists the resolved mob id

**Files:**
- Modify: `services/atlas-monster-book/atlas.com/monster-book/collection/administrator.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/collection/administrator_test.go`

- [ ] **Step 1: Write the failing test**

Add to `collection/administrator_test.go` (read the file first; it already uses `newDB(t)`). Append:

```go
func TestSetCoverPersistsMobId(t *testing.T) {
	db := newDB(t)
	if err := Migration(db); err != nil {
		t.Fatal(err)
	}
	tid := uuid.New()
	cid := character.Id(7)

	// setCover updates an existing row; seed one first.
	if _, err := upsertStats(db, tid, cid, statsUpdate{}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	ev := uuid.New()
	changed, err := setCover(db, tid, cid, item.Id(2380000), 100100, ev)
	if err != nil || !changed {
		t.Fatalf("setCover: changed=%v err=%v", changed, err)
	}

	e, err := getByCharacter(db, tid, cid)
	if err != nil {
		t.Fatalf("getByCharacter: %v", err)
	}
	if e.CoverCardId != 2380000 || e.CoverMobId != 100100 {
		t.Fatalf("persisted cardId=%d mobId=%d, want 2380000/100100", e.CoverCardId, e.CoverMobId)
	}

	// Duplicate eventId must no-op and must NOT overwrite the stored mob id.
	changed2, err := setCover(db, tid, cid, item.Id(0), 0, ev)
	if err != nil {
		t.Fatalf("setCover dup: %v", err)
	}
	if changed2 {
		t.Fatal("duplicate eventId should report changed=false")
	}
	e2, _ := getByCharacter(db, tid, cid)
	if e2.CoverMobId != 100100 || e2.CoverCardId != 2380000 {
		t.Fatalf("duplicate eventId overwrote cover: cardId=%d mobId=%d", e2.CoverCardId, e2.CoverMobId)
	}
}
```

Ensure the test file imports `"github.com/Chronicle20/atlas/libs/atlas-constants/character"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/item"`, and `"github.com/google/uuid"` (add any that are missing).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd <repo-root>/services/atlas-monster-book/atlas.com/monster-book && go test ./collection/ -run TestSetCoverPersistsMobId`
Expected: compile error — `setCover` takes 5 args, test passes 6.

- [ ] **Step 3: Add the parameter and persist the column**

In `collection/administrator.go`, change `setCover`'s signature and `Updates` map:

```go
// setCover updates the cover card + resolved cover mob id, guarded by
// lastCoverEventId. Returns true if the row was modified, false on duplicate
// eventId or missing row (see existence check below).
func setCover(db *gorm.DB, tenantId uuid.UUID, characterId character.Id, coverCardId item.Id, coverMobId uint32, eventId uuid.UUID) (bool, error) {
	res := db.Model(&entity{}).
		Where("tenant_id = ? AND character_id = ?", tenantId, uint32(characterId)).
		Where("last_cover_event_id IS NULL OR last_cover_event_id <> ?", eventId).
		Updates(map[string]interface{}{
			"cover_card_id":       uint32(coverCardId),
			"cover_mob_id":        coverMobId,
			"last_cover_event_id": eventId,
		})
```

Leave the rest of the function (error handling, existence check, return values) unchanged.

- [ ] **Step 4: Update the single caller's compile signature (temporary, finalized in Task 4)**

In `collection/processor.go`, `SetCoverAndEmit` currently calls `setCover(..., cardId, eventId)`. To keep the build green after this task, pass a literal `0` for the new mob id for now:

```go
		changed, err := setCover(p.db.WithContext(p.ctx), p.t.Id(), characterId, cardId, 0, eventId)
```

(Task 4 replaces `0` with the resolved mob id.)

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd <repo-root>/services/atlas-monster-book/atlas.com/monster-book && go test ./collection/ -run TestSetCoverPersistsMobId`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd <repo-root>
git add services/atlas-monster-book/atlas.com/monster-book/collection/administrator.go services/atlas-monster-book/atlas.com/monster-book/collection/administrator_test.go services/atlas-monster-book/atlas.com/monster-book/collection/processor.go
git commit -m "feat(monster-book): persist cover_mob_id in setCover"
git branch --show-current
```

---

## Task 4: collection — resolve card→mob and thread into `SetCoverAndEmit`

**Files:**
- Modify: `services/atlas-monster-book/atlas.com/monster-book/collection/processor.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/collection/processor_test.go`

- [ ] **Step 1: Write the failing test**

Add to `collection/processor_test.go` (it already defines `tenantCtx(t, id)`). Append a local fake and a table-driven test of `resolveCoverMobId` (this isolates FR-2..FR-5 from the Kafka emit path, which needs a live broker):

```go
type fakeConsumable struct {
	model consumable.Model
	err   error
	calls int
}

func (f *fakeConsumable) GetById(uint32) (consumable.Model, error) {
	f.calls++
	return f.model, f.err
}

func mustConsumable(t *testing.T, mb bool, id uint32) consumable.Model {
	t.Helper()
	m, err := consumable.Extract(consumable.RestModel{MonsterBook: mb, MonsterId: id})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return m
}

func TestResolveCoverMobId(t *testing.T) {
	ctx := tenantCtx(t, uuid.New())
	tn := tenant.MustFromContext(ctx)

	cases := []struct {
		name     string
		cardId   item.Id
		model    consumable.Model
		err      error
		want     uint32
		wantCall bool
	}{
		{name: "clear cover skips lookup", cardId: 0, want: 0, wantCall: false},
		{name: "resolves to mob id", cardId: 2380000, model: mustConsumable(t, true, 100100), want: 100100, wantCall: true},
		{name: "atlas-data error fails safe", cardId: 2380000, err: errors.New("boom"), want: 0, wantCall: true},
		{name: "not a monster-book item", cardId: 2380000, model: mustConsumable(t, false, 100100), want: 0, wantCall: true},
		{name: "zero mob id fails safe", cardId: 2380000, model: mustConsumable(t, true, 0), want: 0, wantCall: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeConsumable{model: tc.model, err: tc.err}
			p := &ProcessorImpl{l: logrus.New(), ctx: ctx, t: tn, dp: f}
			got := p.resolveCoverMobId(1, tc.cardId)
			if got != tc.want {
				t.Errorf("resolveCoverMobId = %d, want %d", got, tc.want)
			}
			if (f.calls > 0) != tc.wantCall {
				t.Errorf("lookup calls = %d, wantCall = %v", f.calls, tc.wantCall)
			}
		})
	}
}
```

Add the imports this test needs to `processor_test.go`: `"atlas-monster-book/data/consumable"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/item"` (the package already imports `errors`, `tenant`, `uuid`, `logrus`, `character` — add only what is missing).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd <repo-root>/services/atlas-monster-book/atlas.com/monster-book && go test ./collection/ -run TestResolveCoverMobId`
Expected: compile error — `ProcessorImpl` has no `dp` field and no `resolveCoverMobId` method.

- [ ] **Step 3: Add the consumable dependency to the processor**

In `collection/processor.go`:

Add the import:

```go
	"atlas-monster-book/data/consumable"
```

Add the field to `ProcessorImpl` (after `cp`):

```go
	cp  card.Processor
	dp  consumable.Processor
```

Build it in `NewProcessor` (after `cp:`):

```go
		cp:  card.NewProcessor(l, ctx, db),
		dp:  consumable.NewProcessor(l, ctx),
```

Carry it through `WithTransaction` (the consumable client is DB-agnostic, so copy the same instance):

```go
		cp:  p.cp.WithTransaction(tx),
		dp:  p.dp,
```

- [ ] **Step 4: Add `resolveCoverMobId`**

Add this method to `collection/processor.go` (e.g. just above `SetCoverAndEmit`):

```go
// resolveCoverMobId resolves a cover card item id to its mob id via atlas-data.
// cardId == 0 returns 0 with no lookup. Any failure (atlas-data error, card not
// found, monsterBook == false, or monsterId == 0) returns 0 and logs a warning;
// it never returns an error, so a resolution failure can neither reject the set
// nor produce a client-crashing value (FR-4, FR-5, NFR fail-safe).
func (p *ProcessorImpl) resolveCoverMobId(characterId character.Id, cardId item.Id) uint32 {
	if cardId == 0 {
		return 0
	}
	m, err := p.dp.GetById(uint32(cardId))
	if err != nil || !m.MonsterBook() || m.MonsterId() == 0 {
		p.l.WithError(err).Warnf("Unable to resolve monster-book cover card [%d] to a mob id for character [%d]; storing cover mob id 0.", cardId, characterId)
		return 0
	}
	return m.MonsterId()
}
```

- [ ] **Step 5: Thread the resolved mob id into `SetCoverAndEmit`**

In `SetCoverAndEmit`, resolve the mob id after validation and pass it to `setCover` (replacing the temporary `0` from Task 3):

```go
	coverMobId := p.resolveCoverMobId(characterId, cardId)

	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		changed, err := setCover(p.db.WithContext(p.ctx), p.t.Id(), characterId, cardId, coverMobId, eventId)
```

Leave the `COVER_CHANGED` event body unchanged — it still carries only `CoverCardId` (OQ-3 / FR-7).

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd <repo-root>/services/atlas-monster-book/atlas.com/monster-book && go test ./collection/ -v`
Expected: `TestResolveCoverMobId` (all sub-cases) PASS; existing tests (including `TestSetCoverRejectsUnownedCardBeforeProducerCall`) still PASS.

- [ ] **Step 7: Commit**

```bash
cd <repo-root>
git add services/atlas-monster-book/atlas.com/monster-book/collection/processor.go services/atlas-monster-book/atlas.com/monster-book/collection/processor_test.go
git commit -m "feat(monster-book): resolve cover card to mob id at set time (fail-safe)"
git branch --show-current
```

---

## Task 5: collection REST — expose `coverMonsterId`

**Files:**
- Modify: `services/atlas-monster-book/atlas.com/monster-book/collection/rest.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/collection/rest_test.go` (new file)

- [ ] **Step 1: Write the failing test**

Create `collection/rest_test.go`:

```go
package collection

import "testing"

func TestTransformIncludesCoverMonsterId(t *testing.T) {
	m := NewModelBuilder().
		SetCharacterId(1).
		SetCoverCardId(2380000).
		SetCoverMobId(100100).
		MustBuild()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.CoverMonsterId != 100100 {
		t.Errorf("CoverMonsterId = %d, want 100100", rm.CoverMonsterId)
	}
	if uint32(rm.CoverCardId) != 2380000 {
		t.Errorf("CoverCardId = %d, want 2380000 (must remain card id)", rm.CoverCardId)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd <repo-root>/services/atlas-monster-book/atlas.com/monster-book && go test ./collection/ -run TestTransformIncludesCoverMonsterId`
Expected: compile error — `RestModel` has no `CoverMonsterId` field.

- [ ] **Step 3: Add the field and map it**

In `collection/rest.go`, add to `RestModel` (after `CoverCardId`):

```go
	CoverCardId      item.Id      `json:"coverCardId"`
	CoverMonsterId   uint32       `json:"coverMonsterId"`
```

Add to `Transform`'s returned `RestModel{}` (after `CoverCardId`):

```go
		CoverCardId:      m.CoverCardId(),
		CoverMonsterId:   m.CoverMobId(),
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd <repo-root>/services/atlas-monster-book/atlas.com/monster-book && go test ./collection/ -run TestTransformIncludesCoverMonsterId`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd <repo-root>
git add services/atlas-monster-book/atlas.com/monster-book/collection/rest.go services/atlas-monster-book/atlas.com/monster-book/collection/rest_test.go
git commit -m "feat(monster-book): expose coverMonsterId on the collection REST model"
git branch --show-current
```

---

## Task 6: atlas-channel monsterbook — carry `coverMonsterId` through to the domain model

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/monsterbook/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/monsterbook/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/monsterbook/model.go`
- Test: `services/atlas-channel/atlas.com/channel/monsterbook/rest_test.go`

- [ ] **Step 1: Write the failing test**

Add to `monsterbook/rest_test.go` a test that asserts `coverMonsterId` unmarshals and maps through `Extract`:

```go
func TestExtractIncludesCoverMonsterId(t *testing.T) {
	body := []byte(`{
		"data": {
			"type": "monster-book",
			"id": "42",
			"attributes": {
				"bookLevel": 3,
				"normalCount": 5,
				"specialCount": 2,
				"totalUniqueCards": 7,
				"coverCardId": 2380000,
				"coverMonsterId": 100100,
				"expBonusPercent": 3
			}
		}
	}`)
	var rm CollectionRestModel
	if err := jsonapi.Unmarshal(body, &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}
	if rm.CoverMonsterId != 100100 {
		t.Fatalf("CoverMonsterId = %d, want 100100", rm.CoverMonsterId)
	}
	c, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if c.CoverMonsterId() != 100100 {
		t.Fatalf("Collection.CoverMonsterId() = %d, want 100100", c.CoverMonsterId())
	}
	if c.CoverCardId() != item.Id(2380000) {
		t.Fatalf("Collection.CoverCardId() = %d, want 2380000 (must remain card id)", c.CoverCardId())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd <repo-root>/services/atlas-channel/atlas.com/channel && go test ./monsterbook/ -run TestExtractIncludesCoverMonsterId`
Expected: compile error — `CollectionRestModel` has no `CoverMonsterId`; `Collection` has no `CoverMonsterId()`.

- [ ] **Step 3: Add the field to `Collection` and a getter**

In `monsterbook/processor.go`, add to the `Collection` struct (after `coverCardId`):

```go
	coverCardId      item.Id
	coverMonsterId   uint32
```

Add the getter (after `CoverCardId()`):

```go
func (c Collection) CoverMonsterId() uint32   { return c.coverMonsterId }
```

- [ ] **Step 4: Add the field to the wire model and map it in `Extract`**

In `monsterbook/rest.go`, add to `CollectionRestModel` (after `CoverCardId`):

```go
	CoverCardId      item.Id `json:"coverCardId"`
	CoverMonsterId   uint32  `json:"coverMonsterId"`
```

Add to `Extract`'s returned `Collection{}` (after `coverCardId`):

```go
		coverCardId:      rm.CoverCardId,
		coverMonsterId:   rm.CoverMonsterId,
```

- [ ] **Step 5: Expose it on the monster-book `Model`**

In `monsterbook/model.go`, add (after `CoverCardId()`):

```go
func (m Model) CoverMonsterId() uint32   { return m.collection.CoverMonsterId() }
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `cd <repo-root>/services/atlas-channel/atlas.com/channel && go test ./monsterbook/ -v`
Expected: `TestExtractIncludesCoverMonsterId` PASS; existing tests still PASS.

- [ ] **Step 7: Commit**

```bash
cd <repo-root>
git add services/atlas-channel/atlas.com/channel/monsterbook/processor.go services/atlas-channel/atlas.com/channel/monsterbook/rest.go services/atlas-channel/atlas.com/channel/monsterbook/model.go services/atlas-channel/atlas.com/channel/monsterbook/rest_test.go
git commit -m "feat(channel): carry coverMonsterId through the monster-book model"
git branch --show-current
```

---

## Task 7: atlas-channel — write the mob id into the Character-Info packet

This is the crash fix: the Character-Info cover field must carry the **mob id**, not the card item id.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/character_info.go:60`
- Test: `services/atlas-channel/atlas.com/channel/socket/writer/character_info_test.go` (new file)

- [ ] **Step 1: Write the failing test**

Create `socket/writer/character_info_test.go`. It builds a character with a cover whose card id (`2380000`) differs from its mob id (`100100`), encodes the Character-Info body, decodes it, and asserts the cover field is the **mob id** — proving the card id never reaches the wire:

```go
package writer

import (
	"testing"

	"atlas-channel/character"
	"atlas-channel/guild"
	"atlas-channel/monsterbook"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestCharacterInfoBody_CoverIsMobId(t *testing.T) {
	col, err := monsterbook.Extract(monsterbook.CollectionRestModel{
		CoverCardId:    item.Id(2380000),
		CoverMonsterId: 100100,
	})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	c := character.NewModelBuilder().
		SetId(1).
		SetSp("0").
		SetMonsterBook(monsterbook.NewModel(col, nil)).
		MustBuild()

	enc := CharacterInfoBody(c, guild.Model{}, nil)
	out := charcb.CharacterInfo{}
	ctx := pt.CreateContext("GMS", 83, 1)
	pt.RoundTrip(t, ctx, enc, out.Decode, nil)

	if out.MonsterBookCover() != 100100 {
		t.Errorf("Character-Info cover = %d, want 100100 (mob id, NOT card id 2380000)", out.MonsterBookCover())
	}
}
```

> Note for the implementer: mirror `character_data_test.go`'s builder usage (`SetId`, `SetSp("0")`, `SetMonsterBook`). If `MustBuild` or `CharacterInfoBody` requires additional non-zero model fields (e.g. equipment access for the medal lookup panics on a zero model), set the minimal fields needed — do not weaken the assertion.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd <repo-root>/services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run TestCharacterInfoBody_CoverIsMobId`
Expected: FAIL — cover is `2380000` (the card id) because the writer still sends `mb.CoverCardId()`.

- [ ] **Step 3: Change the writer to send the mob id**

In `socket/writer/character_info.go`, change the `MonsterBookInfo` literal (only the `Cover` line changes):

```go
				charpkt.MonsterBookInfo{
						Level:        uint32(mb.Level()),
						NormalCards:  uint32(mb.NormalCount()),
						SpecialCards: uint32(mb.SpecialCount()),
						TotalCards:   uint32(mb.TotalUniqueCards()),
						Cover:        mb.CoverMonsterId(),
					},
```

(`uint32(mb.CoverCardId())` → `mb.CoverMonsterId()`. When no cover is set, `CoverMonsterId()` is `0` — the client's guarded no-op.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd <repo-root>/services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run TestCharacterInfoBody_CoverIsMobId`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd <repo-root>
git add services/atlas-channel/atlas.com/channel/socket/writer/character_info.go services/atlas-channel/atlas.com/channel/socket/writer/character_info_test.go
git commit -m "fix(channel): send cover mob id (not card id) in Character-Info packet"
git branch --show-current
```

---

## Task 8: libs/atlas-packet — contract guard for the Character-Info cover field

`info.go` does not change (FR-11: the field stays `uint32`/`WriteInt`; only the semantic value supplied by the writer changes). Add a regression test that locks the contract the writer now depends on: `MonsterBookInfo.Cover` round-trips an arbitrary mob-id-shaped value.

**Files:**
- Test: `libs/atlas-packet/character/clientbound/info_test.go`

- [ ] **Step 1: Write the test**

Append to `info_test.go`:

```go
// TestCharacterInfo_CoverCarriesArbitraryValue locks the contract the channel
// writer depends on (task-082): the cover field carries whatever uint32 the
// writer supplies — now a mob id, e.g. 100100 — not a card-id-specific value.
func TestCharacterInfo_CoverCarriesArbitraryValue(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	want := MonsterBookInfo{Level: 1, NormalCards: 0, SpecialCards: 0, TotalCards: 0, Cover: 100100}
	in := NewCharacterInfo(1, 10, 100, 0, "", nil, nil, 0, want)
	out := CharacterInfo{}
	pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	if out.MonsterBookCover() != 100100 {
		t.Errorf("cover = %d, want 100100", out.MonsterBookCover())
	}
}
```

- [ ] **Step 2: Run the test to verify it passes**

Run: `cd <repo-root>/libs/atlas-packet && go test ./character/clientbound/ -run TestCharacterInfo_CoverCarriesArbitraryValue`
Expected: PASS (no production change needed — this guards existing behavior).

- [ ] **Step 3: Commit**

```bash
cd <repo-root>
git add libs/atlas-packet/character/clientbound/info_test.go
git commit -m "test(atlas-packet): guard Character-Info cover field carries arbitrary mob id"
git branch --show-current
```

---

## Task 9: FR-10 — login-draw decision record (no code change, default position)

The design's default and evidence-backed position (OQ-1/FR-10) is that the login-draw `CharacterData` cover stays the **card id** — `data.go` and `character_data.go` are unchanged. The existing regression guard `TestBuildCharacterData_MonsterBook` (`services/atlas-channel/.../socket/writer/character_data_test.go:13`) already asserts the login cover is the card id (`2380001`). This task records the verification and confirms the guard, changing code **only** if IDA proves the login decoder consumes a mob id.

**Files:**
- Create: `docs/tasks/task-082-monsterbook-cover-mobid/fr10-login-draw-finding.md`

- [ ] **Step 1: Confirm the regression guard still passes**

Run: `cd <repo-root>/services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run TestBuildCharacterData_MonsterBook -v`
Expected: PASS — login-draw cover is the card id (`2380001`), unchanged.

- [ ] **Step 2 (best-effort): Verify against the v83 client via IDA**

If an IDA session for `MapleStory_dump.exe` (v83) is available (MCP `mcp__ida-pro__*`), decompile the login `CharacterData` monster-book decoder and confirm it does **not** call `CMobTemplate::GetMobTemplate` on the cover field (contrast with `CWvsContext::OnCharacterInfo` → `sub_684798`, which does). Behavioral evidence already supports this: the live crash occurred only on Character Info, never at login/map-entry, despite the cover being set.

- [ ] **Step 3: Record the finding**

Write `docs/tasks/task-082-monsterbook-cover-mobid/fr10-login-draw-finding.md` capturing: the decision (login-draw stays card id → `data.go` unchanged), the behavioral evidence (crash only on Character Info), and the IDA result (confirmed / unavailable). If IDA proves the login decoder DOES call `GetMobTemplate(cover)`, STOP and escalate: the fix must extend to `character_data.go` (set `MonsterBook.CoverCardId` from `CoverMonsterId()`) and `data.go`'s `encodeMonsterBook`, mirroring Task 7 — re-plan that change before proceeding.

- [ ] **Step 4: Commit**

```bash
cd <repo-root>
git add docs/tasks/task-082-monsterbook-cover-mobid/fr10-login-draw-finding.md
git commit -m "docs(task-082): record FR-10 login-draw decision (cover stays card id)"
git branch --show-current
```

---

## Task 10: Deployment — declare `DATA_SERVICE_URL` for atlas-monster-book

Documents the new outbound dependency. Functionally optional (`requests.RootUrl("DATA")` falls back to `BASE_SERVICE_URL`), but declaring it makes the dependency explicit and allows direct routing.

**Files:**
- Modify: `deploy/k8s/base/atlas-monster-book.yaml`

- [ ] **Step 1: Inspect how a peer service declares a `*_SERVICE_URL`**

Run:
```bash
grep -rn "SERVICE_URL" deploy/k8s/base/atlas-channel.yaml deploy/k8s/base/atlas-npc-shops.yaml
```
Expected: a `- name: <X>_SERVICE_URL` / `value: "..."` env entry (or sourcing from the `atlas-env` configmap). Mirror whichever convention the peer manifests use; if peers rely solely on `BASE_SERVICE_URL` from the `atlas-env` configmap, prefer a literal `DATA_SERVICE_URL` env entry pointing at the atlas-data service for explicitness.

- [ ] **Step 2: Add the env var**

In `deploy/k8s/base/atlas-monster-book.yaml`, under the `monster-book` container's `env:` list (after the existing `LOG_LEVEL`/`DB_*` entries), add:

```yaml
        - name: DATA_SERVICE_URL
          value: "http://atlas-data/"
```

(Match the scheme/host/trailing-slash convention used by the peer `*_SERVICE_URL` entries found in Step 1. The atlas-rest client appends the resource path `data/consumables/{id}` to this base.)

- [ ] **Step 3: Validate the manifest parses**

Run: `cd <repo-root> && python3 -c "import yaml; list(yaml.safe_load_all(open('deploy/k8s/base/atlas-monster-book.yaml')))" && echo OK`
Expected: `OK` (no YAML parse error).

- [ ] **Step 4: Commit**

```bash
cd <repo-root>
git add deploy/k8s/base/atlas-monster-book.yaml
git commit -m "chore(deploy): declare DATA_SERVICE_URL for atlas-monster-book"
git branch --show-current
```

---

## Task 11: Full verification (per CLAUDE.md)

No new code — run the mandatory gates across every changed module and the Docker build for the service whose `go.mod` changed.

**Files:** none (verification only).

- [ ] **Step 1: Tidy + vet + test + build each changed Go module**

Run (atlas-monster-book — `go.mod` changed in Task 1):
```bash
cd <repo-root>/services/atlas-monster-book/atlas.com/monster-book
go mod tidy && go vet ./... && go test -race ./... && go build ./...
```
Expected: all clean.

Run (atlas-channel — source only):
```bash
cd <repo-root>/services/atlas-channel/atlas.com/channel
go vet ./... && go test -race ./... && go build ./...
```
Expected: all clean. Confirm `go.mod`/`go.sum` are unchanged (`git status` shows no diff for them); if `go.mod` changed unexpectedly, atlas-channel must also be baked in Step 3.

Run (libs/atlas-packet — source only):
```bash
cd <repo-root>/libs/atlas-packet
go vet ./... && go test -race ./... && go build ./...
```
Expected: all clean.

- [ ] **Step 2: Redis key guard**

Run:
```bash
cd <repo-root>
tools/redis-key-guard.sh
```
Expected: clean (no new redis usage introduced).

- [ ] **Step 3: Docker bake the service whose `go.mod` changed**

Run from the worktree root:
```bash
cd <repo-root>
docker buildx bake atlas-monster-book
```
Expected: build succeeds. (`atlas-rest` is already `COPY`'d in the repo-root `Dockerfile`, so no Dockerfile edit is needed. If the bake fails on a missing `COPY libs/...`, add the two `COPY` lines for the missing lib and re-bake — but none is expected.)

- [ ] **Step 4: Confirm the working tree is clean and on the task branch**

Run:
```bash
cd <repo-root>
git status --short
git branch --show-current      # must print task-082-monsterbook-cover-mobid
git rev-parse --show-toplevel  # must end with /.worktrees/task-082-monsterbook-cover-mobid
```
Expected: no uncommitted changes (beyond intended), correct branch and worktree.

- [ ] **Step 5: Final commit (only if Step 1 changed go.sum/go.mod via tidy)**

```bash
cd <repo-root>
git add -A
git commit -m "chore(task-082): go mod tidy + verification" || echo "nothing to commit"
git branch --show-current
```

---

## Acceptance Criteria Trace (from PRD §10)

| Acceptance criterion | Covered by |
|---|---|
| Character Info no longer crashes with a cover set; correct monster renders | Task 7 (writer sends mob id) + Task 4 (resolution) |
| No cover (`coverCardId == 0`) behaves as before | Task 4 (`resolveCoverMobId` returns 0, no lookup) + Task 7 (Cover=0 no-op) |
| Character-Info cover field carries the cover card's mob id (or 0), verified by encoder test | Task 7 + Task 8 |
| Setting a cover resolves and persists `cover_mob_id`; `coverCardId` unchanged | Task 2, Task 3, Task 4 |
| Unresolvable cover stores mob id 0 + warning; set still succeeds | Task 4 (`resolveCoverMobId` fail-safe) |
| `0x54`, window, card list unchanged (card-id space) | No change to those paths (OQ-3); guarded by unchanged tests |
| Login-draw decision (FR-10) documented & implemented | Task 9 |
| Schema migration adds `cover_mob_id`; REST exposes it; backfill handled (lazy) | Task 2 (AutoMigrate), Task 5 (REST), OQ-2 lazy (no backfill task) |
| `go test -race`, `go vet`, `go build` clean; `docker buildx bake`; redis-key-guard clean | Task 11 |

---

## Self-Review Notes

- **Spec coverage:** Every FR (FR-1..FR-13) and PRD acceptance criterion maps to a task (see trace above). FR-7/OQ-3 (no Kafka change) and OQ-2 (lazy backfill) are deliberate no-ops, documented here.
- **Type consistency:** `coverMobId`/`CoverMobId()` (monster-book domain), `cover_mob_id` (db column), `coverMonsterId`/`CoverMonsterId` (JSON + channel domain + REST), `MonsterId()`/`MonsterBook()` (consumable model) used consistently across tasks. `setCover` 6-arg signature `(db, tenantId, characterId, coverCardId, coverMobId, eventId)` defined in Task 3, called with the resolved value in Task 4. `consumable.Processor.GetById(uint32)(Model,error)` defined in Task 1, faked in Task 4.
- **No placeholders:** Every code step contains the exact code; every run step states the expected result.
