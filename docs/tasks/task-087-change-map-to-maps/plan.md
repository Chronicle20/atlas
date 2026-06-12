# Move Character Change-Map Write to atlas-maps + Retire the Location Shim — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make atlas-maps own the character change-map write (one warp method shared by the Kafka consumer and a new REST endpoint), migrate every consumer off the atlas-character `mapId` echo, and delete the atlas-character location shim.

**Architecture:** A new `character/warp` processor in atlas-maps factors the existing `CHANGE_MAP` consumer body into one `ChangeMap` method. A new `PATCH /characters/{id}/location` handler validates target-map existence and calls that same method. The UI and three "active" Go services switch to reading location from atlas-maps; five "passive" services drop their dead `MapId` mirror field; atlas-character's GET shim is removed **last**.

**Tech Stack:** Go 1.25 microservices (DDD, immutable models + builders, JSON:API via api2go, Kafka via `message.Buffer`/`Emit`, GORM/sqlite tests); atlas-ui (Next.js/React 19, TanStack React Query, vitest).

**Execution constraint:** Tasks are ordered. atlas-maps (Tasks 1–2) first; atlas-character shim removal (Task 11) **last**, after all consumer migrations (Tasks 7–10). Run the full verification gate (Task 12) before declaring done.

**Worktree discipline:** All work happens in `.worktrees/task-087-change-map-to-maps/` on branch `task-087-change-map-to-maps`. Every implementer must `cd` into the worktree first and confirm `git branch --show-current` after each commit.

---

## Task 1: atlas-maps — shared warp processor + consumer rewire

Factor the `CHANGE_MAP` consumer body into a single `warp.Processor.ChangeMap` method, then make the consumer delegate to it. This is the single authoritative warp implementation (FR-1.4, FR-7.2 command side, FR-7.1).

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/character/warp/processor.go`
- Create: `services/atlas-maps/atlas.com/maps/character/warp/processor_test.go`
- Modify: `services/atlas-maps/atlas.com/maps/kafka/consumer/character/change_map.go`

- [x] **Step 1: Write the warp processor (production code first, then its test fails to compile until present).** Create `character/warp/processor.go`:

```go
package warp

import (
	"context"

	"atlas-maps/character/location"
	"atlas-maps/kafka/message"
	characterKafka "atlas-maps/kafka/message/character"
	"atlas-maps/kafka/producer"
	_map "atlas-maps/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// mapTransitioner is the narrow slice of _map.Processor that warp needs. It
// keeps the processor unit-testable without standing up the full map processor
// (which makes external calls). _map.Processor satisfies it.
type mapTransitioner interface {
	TransitionMapAndEmit(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model) error
}

// Processor is the single authoritative character warp implementation. Both the
// CHANGE_MAP Kafka consumer and the PATCH /characters/{id}/location REST handler
// call ChangeMap so the two paths cannot diverge (FR-1.4).
type Processor interface {
	// ChangeMap persists dest as the character's location, emits the canonical
	// MAP_CHANGED status event, and transitions the per-map registries. dest
	// must be a fully-formed field (world, channel, map, instance). The current
	// row is read internally for the MAP_CHANGED "old" side; if absent, oldField
	// defaults to dest (parity with the pre-task-087 consumer). Returns an error
	// only when the durable Set fails; emit/transition failures are logged and
	// the call still succeeds (parity with the consumer).
	ChangeMap(transactionId uuid.UUID, characterId uint32, worldId world.Id, dest field.Model, portalId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	lp  location.Processor
	pp  producer.Provider
	mp  mapTransitioner
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	pp := producer.ProviderImpl(l)(ctx)
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		lp:  location.NewProcessor(l, ctx, db),
		pp:  pp,
		mp:  _map.NewProcessor(l, ctx, pp, db),
	}
}

// newProcessorWithDeps is the unit-test seam (mirrors location's
// newProcessorWithInfo). It is not exported and not a *_testhelpers.go file.
func newProcessorWithDeps(l logrus.FieldLogger, ctx context.Context, lp location.Processor, pp producer.Provider, mp mapTransitioner) *ProcessorImpl {
	return &ProcessorImpl{l: l, ctx: ctx, lp: lp, pp: pp, mp: mp}
}

func (p *ProcessorImpl) ChangeMap(transactionId uuid.UUID, characterId uint32, worldId world.Id, dest field.Model, portalId uint32) error {
	oldField := dest
	if old, err := p.lp.GetById(characterId); err == nil {
		oldField = old.Field()
	}

	if _, err := p.lp.Set(characterId, dest); err != nil {
		p.l.WithError(err).Errorf("ChangeMap: location.Set failed for character [%d].", characterId)
		return err
	}

	if err := message.Emit(p.pp)(func(buf *message.Buffer) error {
		return buf.Put(characterKafka.EnvEventTopicCharacterStatus,
			producer.MapChangedStatusProvider(transactionId, characterId, worldId, oldField, dest, portalId))
	}); err != nil {
		p.l.WithError(err).Errorf("ChangeMap: failed to emit MAP_CHANGED status for character [%d].", characterId)
	}

	if err := p.mp.TransitionMapAndEmit(transactionId, dest, characterId, oldField); err != nil {
		p.l.WithError(err).Warnf("ChangeMap: TransitionMapAndEmit failed for character [%d].", characterId)
	}

	return nil
}
```

- [x] **Step 2: Write the failing test.** Create `character/warp/processor_test.go`:

```go
package warp

import (
	"context"
	"testing"

	"atlas-maps/character/location"
	characterKafka "atlas-maps/kafka/message/character"
	mapsproducer "atlas-maps/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// capturingProducer records every emitted message by topic.
type capturingProducer struct {
	messages map[string][]kafka.Message
}

func newCapturingProducer() *capturingProducer {
	return &capturingProducer{messages: make(map[string][]kafka.Message)}
}

func (c *capturingProducer) Provider() mapsproducer.Provider {
	return func(token string) kafkaproducer.MessageProducer {
		return func(p model.Provider[[]kafka.Message]) error {
			ms, err := p()
			if err != nil {
				return err
			}
			c.messages[token] = append(c.messages[token], ms...)
			return nil
		}
	}
}

// noopTransitioner satisfies mapTransitioner without external calls.
type noopTransitioner struct{ calls int }

func (n *noopTransitioner) TransitionMapAndEmit(_ uuid.UUID, _ field.Model, _ uint32, _ field.Model) error {
	n.calls++
	return nil
}

func newCtxTenant(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

func newLocationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := location.Migration(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestChangeMap_PersistsAndEmitsMapChanged(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newLocationDB(t)
	lp := location.NewProcessor(logrus.New(), ctx, db)

	// Seed an existing location row (the "old" side).
	start := field.NewBuilder(world0(), channel1(), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	if _, err := lp.Set(12345, start); err != nil {
		t.Fatalf("seed Set: %v", err)
	}

	cp := newCapturingProducer()
	mt := &noopTransitioner{}
	p := newProcessorWithDeps(logrus.New(), ctx, lp, cp.Provider(), mt)

	dest := field.NewBuilder(world0(), channel1(), _map.Id(104000000)).SetInstance(uuid.Nil).Build()
	if err := p.ChangeMap(uuid.New(), 12345, world0(), dest, 0); err != nil {
		t.Fatalf("ChangeMap: %v", err)
	}

	// Durable row updated.
	got, err := lp.GetById(12345)
	if err != nil {
		t.Fatalf("GetById after warp: %v", err)
	}
	if got.MapId() != _map.Id(104000000) {
		t.Fatalf("persisted MapId = %d, want 104000000", got.MapId())
	}

	// MAP_CHANGED emitted on the character-status topic.
	msgs := cp.messages[characterKafka.EnvEventTopicCharacterStatus]
	if len(msgs) != 1 {
		t.Fatalf("emitted %d status messages, want 1", len(msgs))
	}
	if mt.calls != 1 {
		t.Fatalf("TransitionMapAndEmit called %d times, want 1", mt.calls)
	}
}
```

> Note: `world0()`/`channel1()` are tiny local helpers — add at the bottom of the
> test file: `func world0() world.Id { return 0 }` and
> `func channel1() channel.Id { return 1 }` with the matching imports
> (`.../atlas-constants/world`, `.../atlas-constants/channel`). Confirm the
> location package exposes a migration helper named `Migration` (check
> `character/location/administrator.go` / `entity.go`); if it is named
> differently (e.g. `Migrate`/`AutoMigrate`), use that exact name. If the
> existing `location` tests already migrate via `db.AutoMigrate(&entity{})` in
> the same package, replicate that call here instead of `location.Migration(db)`.

- [x] **Step 3: Run the test to verify it fails.**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/warp/ -run TestChangeMap_PersistsAndEmitsMapChanged -v`
Expected: compiles and FAILS only if the implementation is wrong; since Step 1 already wrote the implementation, this should PASS. If it does not compile, fix the migration helper name per the Step 2 note. (TDD note: if you prefer strict red-first, comment out the `ChangeMap` body to see RED, then restore.)

- [x] **Step 4: Rewire the consumer to delegate to warp.** Replace the body of `kafka/consumer/character/change_map.go` so the warp is built and called via a small command-side helper (this helper is the FR-7.2 command-side seam):

```go
package character

import (
	"atlas-maps/character/warp"
	characterKafka "atlas-maps/kafka/message/character"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// changeMapFromCommand builds the destination field from a CHANGE_MAP command
// body and funnels it through the single shared warp method.
func changeMapFromCommand(wp warp.Processor) func(c characterKafka.Command[characterKafka.ChangeMapBody]) error {
	return func(c characterKafka.Command[characterKafka.ChangeMapBody]) error {
		dest := field.NewBuilder(c.WorldId, c.Body.ChannelId, c.Body.MapId).SetInstance(c.Body.Instance).Build()
		return wp.ChangeMap(c.TransactionId, c.CharacterId, c.WorldId, dest, c.Body.PortalId)
	}
}

func handleChangeMapFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c characterKafka.Command[characterKafka.ChangeMapBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c characterKafka.Command[characterKafka.ChangeMapBody]) {
		if c.Type != characterKafka.CommandChangeMap {
			return
		}
		wp := warp.NewProcessor(l, ctx, db)
		if err := changeMapFromCommand(wp)(c); err != nil {
			l.WithError(err).Errorf("CHANGE_MAP: warp failed for character [%d].", c.CharacterId)
		}
	}
}
```

- [x] **Step 5: Add the command-side parity test.** Append to `character/warp/processor_test.go` a mock `Processor` capturing the dest, and assert the command helper funnels through `ChangeMap`. Because `changeMapFromCommand` lives in the consumer package, put this test in the consumer package instead — create `kafka/consumer/character/change_map_test.go`:

```go
package character

import (
	"testing"

	"atlas-maps/character/warp"
	characterKafka "atlas-maps/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/google/uuid"
)

type recordingWarp struct {
	gotDest     field.Model
	gotPortalId uint32
	calls       int
}

func (r *recordingWarp) ChangeMap(_ uuid.UUID, _ uint32, _ world0Type, dest field.Model, portalId uint32) error {
	r.calls++
	r.gotDest = dest
	r.gotPortalId = portalId
	return nil
}

func TestChangeMapFromCommand_FunnelsThroughWarp(t *testing.T) {
	rw := &recordingWarp{}
	inst := uuid.New()
	cmd := characterKafka.Command[characterKafka.ChangeMapBody]{
		WorldId:     1,
		CharacterId: 999,
		Type:        characterKafka.CommandChangeMap,
		Body: characterKafka.ChangeMapBody{
			ChannelId: 2,
			MapId:     _map.Id(240000000),
			Instance:  inst,
			PortalId:  3,
		},
	}
	if err := changeMapFromCommand(warp.Processor(rw))(cmd); err != nil {
		t.Fatalf("changeMapFromCommand: %v", err)
	}
	if rw.calls != 1 {
		t.Fatalf("ChangeMap called %d times, want 1", rw.calls)
	}
	if rw.gotDest.MapId() != _map.Id(240000000) || rw.gotDest.ChannelId() != 2 || rw.gotDest.Instance() != inst {
		t.Fatalf("dest mismatch: %+v", rw.gotDest)
	}
	if rw.gotPortalId != 3 {
		t.Fatalf("portalId = %d, want 3", rw.gotPortalId)
	}
}
```

> Note: replace `world0Type` with the real type in `warp.Processor.ChangeMap`'s
> signature — it is `world.Id` (`github.com/Chronicle20/atlas/libs/atlas-constants/world`).
> The placeholder is only here to remind you to import `world` and use
> `world.Id` for the third parameter. Confirm the exact field names on
> `characterKafka.Command` and `ChangeMapBody` against
> `kafka/message/character/` (the consumer already reads `c.WorldId`,
> `c.CharacterId`, `c.TransactionId`, `c.Type`, `c.Body.ChannelId`,
> `c.Body.MapId`, `c.Body.Instance`, `c.Body.PortalId`).

- [x] **Step 6: Run the package tests.**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/warp/ ./kafka/consumer/character/ -v`
Expected: PASS.

- [x] **Step 7: Build & vet the module.**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go vet ./...`
Expected: clean.

- [x] **Step 8: Commit.**

```bash
git add services/atlas-maps/atlas.com/maps/character/warp/ services/atlas-maps/atlas.com/maps/kafka/consumer/character/
git commit -m "feat(atlas-maps): factor shared warp.ChangeMap; consumer delegates (task-087)"
git branch --show-current   # must print task-087-change-map-to-maps
```

---

## Task 2: atlas-maps — PATCH location write endpoint + map validation

Add the REST write that calls the shared warp method, validates target-map existence (400), and 404s when the character has no location row (FR-1.1–1.6, FR-2.1–2.3, FR-7.1 REST, FR-7.2 REST side, FR-7.3).

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/character/location/resource.go`
- Create: `services/atlas-maps/atlas.com/maps/character/location/resource_test.go`

- [x] **Step 1: Add the handler + testable helper to `resource.go`.** Add imports `"atlas-maps/character/warp"`, `"atlas-maps/data/map/info"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/field"`, `"github.com/Chronicle20/atlas/libs/atlas-rest/requests"`, `"github.com/google/uuid"`, and `_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"`. Register the route inside `InitResource` (next to the GET line):

```go
r.HandleFunc("/{characterId}/location",
	rest.RegisterInputHandler[RestModel](l)(si)("change_character_location", handleChangeCharacterLocation(db)),
).Methods(http.MethodPatch)
```

Add the handler and the pure helper:

```go
// warpProcessor is the narrow slice of warp.Processor the helper needs.
type warpProcessor interface {
	ChangeMap(transactionId uuid.UUID, characterId uint32, worldId world.Id, dest field.Model, portalId uint32) error
}

func handleChangeCharacterLocation(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				lp := NewProcessor(d.Logger(), d.Context(), db)
				ip := info.NewProcessor(d.Logger(), d.Context())
				wp := warp.NewProcessor(d.Logger(), d.Context(), db)
				status := changeCharacterLocation(d.Logger(), lp, ip, wp, characterId, input.MapId)
				w.WriteHeader(status)
			}
		})
	}
}

// changeCharacterLocation is the unit-testable core of the write handler. It
// returns the HTTP status to write. channelId/instance from the body are
// ignored — this is a map-only warp (FR-1.2); destination channel is the
// stored channel (OQ-5), instance is uuid.Nil (non-instanced), spawn portal 0.
func changeCharacterLocation(l logrus.FieldLogger, lp Processor, ip info.Processor, wp warpProcessor, characterId uint32, targetMapId _map.Id) int {
	cur, err := lp.GetById(characterId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		l.Warnf("change_character_location: no location row for character [%d]; rejecting 404.", characterId)
		return http.StatusNotFound
	}
	if err != nil {
		l.WithError(err).Errorf("change_character_location: loading location for character [%d].", characterId)
		return http.StatusInternalServerError
	}

	if _, err := ip.GetById(targetMapId); err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			l.WithError(err).Warnf("change_character_location: target map [%d] does not exist; rejecting 400.", targetMapId)
			return http.StatusBadRequest
		}
		l.WithError(err).Errorf("change_character_location: map-existence check failed for [%d] (infrastructure).", targetMapId)
		return http.StatusInternalServerError
	}

	dest := field.NewBuilder(cur.WorldId(), cur.ChannelId(), targetMapId).SetInstance(uuid.Nil).Build()
	if err := wp.ChangeMap(uuid.New(), characterId, cur.WorldId(), dest, 0); err != nil {
		l.WithError(err).Errorf("change_character_location: warp failed for character [%d].", characterId)
		return http.StatusInternalServerError
	}

	l.WithFields(logrus.Fields{"character_id": characterId, "map_id": targetMapId}).
		Infof("change_character_location: warped character [%d] to map [%d].", characterId, targetMapId)
	return http.StatusNoContent
}
```

Add `world "github.com/Chronicle20/atlas/libs/atlas-constants/world"` to the imports (used by the `warpProcessor` interface signature).

- [x] **Step 2: Write the failing test.** Create `character/location/resource_test.go`:

```go
package location

import (
	"net/http"
	"testing"

	"atlas-maps/data/map/info"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// recordingWarp captures ChangeMap calls.
type recordingWarp struct {
	calls   int
	gotDest field.Model
}

func (r *recordingWarp) ChangeMap(_ uuid.UUID, _ uint32, _ world.Id, dest field.Model, _ uint32) error {
	r.calls++
	r.gotDest = dest
	return nil
}

func TestChangeCharacterLocation_HappyPath(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newLocationDB(t) // sqlite + migration, mirror the warp test helper
	lp := NewProcessor(logrus.New(), ctx, db)
	if _, err := lp.Set(7, field.NewBuilder(world.Id(0), 1, _map.Id(100000000)).SetInstance(uuid.Nil).Build()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ip := &stubInfoProcessor{out: info.NewBuilder().SetId(104000000).Build()} // err nil ⇒ map exists
	rw := &recordingWarp{}

	status := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(104000000))
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", status)
	}
	if rw.calls != 1 {
		t.Fatalf("ChangeMap calls = %d, want 1", rw.calls)
	}
	if rw.gotDest.MapId() != _map.Id(104000000) || rw.gotDest.ChannelId() != 1 || rw.gotDest.Instance() != uuid.Nil {
		t.Fatalf("dest mismatch: %+v", rw.gotDest)
	}
}

func TestChangeCharacterLocation_InvalidMap_400_NoWarp(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newLocationDB(t)
	lp := NewProcessor(logrus.New(), ctx, db)
	if _, err := lp.Set(7, field.NewBuilder(world.Id(0), 1, _map.Id(100000000)).SetInstance(uuid.Nil).Build()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ip := &stubInfoProcessor{err: requests.ErrNotFound} // map does not exist
	rw := &recordingWarp{}

	status := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(999999999))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
	if rw.calls != 0 {
		t.Fatalf("ChangeMap must not be called on invalid map; got %d calls", rw.calls)
	}
}

func TestChangeCharacterLocation_NoRow_404(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newLocationDB(t)
	lp := NewProcessor(logrus.New(), ctx, db)
	ip := &stubInfoProcessor{out: info.NewBuilder().SetId(104000000).Build()}
	rw := &recordingWarp{}

	status := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(104000000))
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
	if rw.calls != 0 {
		t.Fatalf("ChangeMap must not be called when no row; got %d calls", rw.calls)
	}
}
```

> Notes: `stubInfoProcessor` and `newCtxTenant` already exist in
> `character/location/processor_test.go` (same package) — reuse them; do **not**
> redeclare. Add a `newLocationDB(t)` helper to the location test package if one
> does not already exist (copy the sqlite-open + migration from the existing
> location tests / the warp test). Confirm `lp.GetById` on a missing row returns
> `gorm.ErrRecordNotFound` (the GET handler at `resource.go:32` already relies on
> this `errors.Is(err, gorm.ErrRecordNotFound)`); if `getByTenantAndCharacterIdProvider`
> wraps it, adjust the helper's not-found check to match the wrapped sentinel.

- [x] **Step 3: Run the test to verify it passes.**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/location/ -run TestChangeCharacterLocation -v`
Expected: PASS (3 sub-tests). If the no-row case does not return `gorm.ErrRecordNotFound`, fix the sentinel in both the helper and the test.

- [x] **Step 4: Build, vet, and race-test the whole module.**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go vet ./... && go test -race ./...`
Expected: clean.

- [x] **Step 5: Commit.**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/
git commit -m "feat(atlas-maps): PATCH /characters/{id}/location write + map validation (task-087)"
git branch --show-current
```

---

## Task 3: atlas-ui — locations service, type, and query hook

Introduce the read+write client for atlas-maps location, used by the dialog (Task 4) and the table column (Task 5). (FR-5.1, FR-5.2, FR-5.5.)

**Files:**
- Create: `services/atlas-ui/src/types/models/location.ts`
- Create: `services/atlas-ui/src/services/api/locations.service.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/useCharacterLocation.ts`
- Create: `services/atlas-ui/src/services/api/__tests__/locations.service.test.ts`

- [x] **Step 1: Add the location type.** Create `types/models/location.ts`:

```typescript
export interface CharacterLocationAttributes {
  worldId: number;
  channelId: number;
  mapId: number;
  instance: string;
}

export interface CharacterLocation {
  id: string;
  type: "character-locations";
  attributes: CharacterLocationAttributes;
}

export interface ChangeMapData {
  mapId: number;
}
```

- [x] **Step 2: Add the service.** Create `services/api/locations.service.ts`, mirroring `maps.service.ts`'s envelope pattern:

```typescript
import { api } from "@/services/api/client"; // confirm the actual client import used by characters.service.ts
import type { ServiceOptions } from "@/services/api/types"; // confirm path used by sibling services
import type { CharacterLocation, ChangeMapData } from "@/types/models/location";

const BASE_PATH = "/api/characters";

export const locationsService = {
  async getByCharacterId(characterId: string, options?: ServiceOptions): Promise<CharacterLocation> {
    return api.getOne<CharacterLocation>(`${BASE_PATH}/${characterId}/location`, options);
  },

  async changeMap(characterId: string, data: ChangeMapData, options?: ServiceOptions): Promise<void> {
    const requestBody = {
      data: {
        type: "character-locations",
        id: characterId,
        attributes: data,
      },
    };
    return api.patch<void>(`${BASE_PATH}/${characterId}/location`, requestBody, options);
  },
};
```

> Note: open `services/api/characters.service.ts` and copy its exact import
> lines for the api client and `ServiceOptions` so this file matches (the
> placeholder import paths above must be replaced with the real ones). `getOne`
> and `patch` are the same methods `characters.service.ts` and `maps.service.ts`
> already use.

- [x] **Step 3: Add the read hook.** Create `lib/hooks/api/useCharacterLocation.ts`:

```typescript
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { locationsService } from "@/services/api/locations.service";
import type { CharacterLocation } from "@/types/models/location";
import type { Tenant } from "@/types/models/tenant"; // confirm tenant type path used by useCharacters.ts

export const characterLocationKeys = {
  all: ["character-location"] as const,
  detail: (tenantId: string | undefined, characterId: string) =>
    ["character-location", tenantId, characterId] as const,
};

export function useCharacterLocation(
  tenant: Tenant | null | undefined,
  characterId: string,
): UseQueryResult<CharacterLocation, Error> {
  return useQuery({
    queryKey: characterLocationKeys.detail(tenant?.id, characterId),
    queryFn: () => locationsService.getByCharacterId(characterId),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
    retry: (failureCount, error) => {
      const msg = error?.message?.toLowerCase() ?? "";
      if (msg.includes("404") || msg.includes("not found")) return false;
      return failureCount < 3;
    },
  });
}
```

> Note: match the `Tenant` import path and the `enabled`/`retry` conventions to
> `lib/hooks/api/useCharacterEffectiveStats.ts` exactly.

- [x] **Step 4: Write the failing service test.** Create `services/api/__tests__/locations.service.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";
import { locationsService } from "@/services/api/locations.service";
import { api } from "@/services/api/client"; // same import the service uses

vi.mock("@/services/api/client", () => ({
  api: { getOne: vi.fn(), patch: vi.fn() },
}));

describe("locationsService", () => {
  beforeEach(() => vi.clearAllMocks());

  it("getByCharacterId hits the atlas-maps location endpoint", async () => {
    (api.getOne as any).mockResolvedValue({ id: "7", type: "character-locations", attributes: { worldId: 0, channelId: 1, mapId: 100000000, instance: "" } });
    const res = await locationsService.getByCharacterId("7");
    expect(api.getOne).toHaveBeenCalledWith("/api/characters/7/location", undefined);
    expect(res.attributes.mapId).toBe(100000000);
  });

  it("changeMap PATCHes a character-locations JSON:API envelope", async () => {
    (api.patch as any).mockResolvedValue(undefined);
    await locationsService.changeMap("7", { mapId: 104000000 });
    expect(api.patch).toHaveBeenCalledWith(
      "/api/characters/7/location",
      { data: { type: "character-locations", id: "7", attributes: { mapId: 104000000 } } },
      undefined,
    );
  });
});
```

> Note: copy the mock style and import paths from an existing test under
> `services/api/__tests__/` (e.g. `characterRender.service.test.ts`) so the
> `vi.mock` target matches the real client module path.

- [x] **Step 5: Run the test.**

Run: `cd services/atlas-ui && npx vitest run src/services/api/__tests__/locations.service.test.ts`
Expected: PASS.

- [x] **Step 6: Type-check.**

Run: `cd services/atlas-ui && npm run build`
Expected: clean (tsc compiles the new files + test).

- [x] **Step 7: Commit.**

```bash
git add services/atlas-ui/src/types/models/location.ts services/atlas-ui/src/services/api/locations.service.ts services/atlas-ui/src/lib/hooks/api/useCharacterLocation.ts services/atlas-ui/src/services/api/__tests__/locations.service.test.ts
git commit -m "feat(atlas-ui): character location service, type, and read hook (task-087)"
git branch --show-current
```

---

## Task 4: atlas-ui — repoint ChangeMapDialog to the location endpoint

Read current map from `GET .../location` and write via `PATCH .../location` (FR-5.1, FR-5.2, FR-7.4).

**Files:**
- Modify: `services/atlas-ui/src/components/features/characters/ChangeMapDialog.tsx`
- Create: `services/atlas-ui/src/components/features/characters/__tests__/ChangeMapDialog.test.tsx`

- [x] **Step 1: Read the current component fully** to capture its prop shape and the four `character.attributes.mapId` sites (initial value ~19, validation ~51, reset ~94, description ~156) and the `charactersService.update` write (~90).

Run: `cat services/atlas-ui/src/components/features/characters/ChangeMapDialog.tsx`

- [x] **Step 2: Replace the current-map source and the write call.** Make these edits:
  1. Import the hook and service:
     ```typescript
     import { useCharacterLocation } from "@/lib/hooks/api/useCharacterLocation";
     import { locationsService } from "@/services/api/locations.service";
     ```
  2. Derive current map from the location query (the dialog already has access to
     the `tenant` and `character` — confirm how `tenant` is obtained in this
     component; if not present, read it from the same context other dialogs use):
     ```typescript
     const { data: location } = useCharacterLocation(tenant, character.id);
     const currentMapId = location?.attributes.mapId;
     ```
  3. Initialize the input from `currentMapId` (fall back to empty string until
     the query resolves), and reset to `currentMapId` on cancel:
     ```typescript
     const [mapId, setMapId] = useState<string>(currentMapId != null ? String(currentMapId) : "");
     ```
     Add an effect so the field syncs once location loads:
     ```typescript
     useEffect(() => {
       if (currentMapId != null) setMapId(String(currentMapId));
     }, [currentMapId]);
     ```
  4. "Differs from current" validation compares against `currentMapId`:
     ```typescript
     if (currentMapId != null && numValue === currentMapId) {
       return "Character is already on this map";
     }
     ```
  5. The description text shows `currentMapId` (render `—` while undefined).
  6. The write switches to the location service:
     ```typescript
     await locationsService.changeMap(character.id, { mapId: mapIdNumber });
     ```
  Remove the now-unused `charactersService` import **only if** nothing else in
  the file uses it.

- [x] **Step 3: Write the failing component test.** Create `__tests__/ChangeMapDialog.test.tsx`. Mock the location hook and the service; assert the dialog shows the location-sourced current map and writes via `locationsService.changeMap`:

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ChangeMapDialog } from "@/components/features/characters/ChangeMapDialog";

vi.mock("@/lib/hooks/api/useCharacterLocation", () => ({
  useCharacterLocation: () => ({
    data: { id: "7", type: "character-locations", attributes: { worldId: 0, channelId: 1, mapId: 100000000, instance: "" } },
  }),
}));
vi.mock("@/services/api/locations.service", () => ({
  locationsService: { changeMap: vi.fn().mockResolvedValue(undefined) },
}));
import { locationsService } from "@/services/api/locations.service";

describe("ChangeMapDialog", () => {
  beforeEach(() => vi.clearAllMocks());

  it("writes the warp via the location endpoint", async () => {
    // Render with the dialog open. Match the real required props of the
    // component (character object, open/onOpenChange, tenant, etc.) — read the
    // component signature in Step 1 and supply minimal valid props/wrappers
    // (QueryClientProvider if the component uses hooks beyond the mocked one).
    render(<ChangeMapDialog /* ...required props with character.id="7" and open */ />);

    const input = screen.getByLabelText(/map/i); // adjust matcher to the real label
    fireEvent.change(input, { target: { value: "104000000" } });
    fireEvent.click(screen.getByRole("button", { name: /change|confirm|save/i }));

    await waitFor(() =>
      expect(locationsService.changeMap).toHaveBeenCalledWith("7", { mapId: 104000000 }),
    );
  });
});
```

> Note: this component test must be made to match the real component (props,
> labels, button text, any required `QueryClientProvider`/tenant-context
> wrapper). Use the closest existing component test in the repo as the wrapper
> template. If no component tests exist and wiring a render harness proves
> heavy, it is acceptable to scope FR-7.4's automated coverage to the service
> test (Task 3) plus a lighter test that the dialog's submit handler calls
> `locationsService.changeMap` — but prefer the full render test.

- [x] **Step 4: Run the test.**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/__tests__/ChangeMapDialog.test.tsx`
Expected: PASS.

- [x] **Step 5: Type-check.**

Run: `cd services/atlas-ui && npm run build`
Expected: clean.

- [x] **Step 6: Commit.**

```bash
git add services/atlas-ui/src/components/features/characters/ChangeMapDialog.tsx services/atlas-ui/src/components/features/characters/__tests__/ChangeMapDialog.test.tsx
git commit -m "feat(atlas-ui): ChangeMapDialog reads/writes location via atlas-maps (task-087)"
git branch --show-current
```

---

## Task 5: atlas-ui — characters table map column from per-row location

Source the "Map" column from a per-row location query, keep the link, degrade gracefully when unknown (FR-5.3, OQ-6).

**Files:**
- Modify: `services/atlas-ui/src/pages/characters-columns.tsx`
- Create (if needed): a small cell component `services/atlas-ui/src/components/features/characters/CharacterMapCell.tsx`

- [x] **Step 1: Read the columns file and confirm how cells access `tenant`** (the `getColumns(tenant, …)` factory at ~line 39 passes `tenant` in). Run: `sed -n '1,200p' services/atlas-ui/src/pages/characters-columns.tsx`

- [x] **Step 2: Create a per-row cell component** that runs the location hook (a column cell may use hooks because it renders as a React component). Create `components/features/characters/CharacterMapCell.tsx`:

```tsx
import { Link } from "react-router"; // match the router import used in characters-columns.tsx
import { useCharacterLocation } from "@/lib/hooks/api/useCharacterLocation";
import { MapCell } from "@/components/map-cell";
import type { Tenant } from "@/types/models/tenant";

export function CharacterMapCell({ characterId, tenant }: { characterId: string; tenant: Tenant }) {
  const { data: location } = useCharacterLocation(tenant, characterId);
  const mapId = location?.attributes.mapId;
  if (mapId == null) {
    return <span className="text-muted-foreground">—</span>;
  }
  return (
    <Link to={"/maps/" + String(mapId)}>
      <MapCell mapId={String(mapId)} tenant={tenant} />
    </Link>
  );
}
```

- [x] **Step 3: Replace the Map column cell** in `characters-columns.tsx` to render the new component (drop the `accessorKey: "attributes.mapId"` value read; keep a stable `id`):

```tsx
{
  id: "map",
  header: "Map",
  cell: ({ row }) => (
    <CharacterMapCell characterId={row.original.id} tenant={tenant} />
  ),
},
```

Add the import `import { CharacterMapCell } from "@/components/features/characters/CharacterMapCell";`. If any sorting/filtering referenced `attributes_mapId`, remove those references (the column is no longer backed by a character field).

- [x] **Step 4: Type-check + run UI tests.**

Run: `cd services/atlas-ui && npm run build && npx vitest run`
Expected: clean / PASS. Fix any test or call site that referenced the old `attributes.mapId` column accessor.

- [x] **Step 5: Commit.**

```bash
git add services/atlas-ui/src/pages/characters-columns.tsx services/atlas-ui/src/components/features/characters/CharacterMapCell.tsx
git commit -m "feat(atlas-ui): characters table map column sourced from location (task-087)"
git branch --show-current
```

---

## Task 6: atlas-ui — remove mapId from the character type

Now that no UI code reads `mapId` off the character resource, remove it from the types (FR-5.4).

**Files:**
- Modify: `services/atlas-ui/src/types/models/character.ts`

- [x] **Step 1: Remove `mapId: number;`** from `CharacterAttributes` (line ~34) and `mapId?: number;` from `UpdateCharacterData` (line ~44).

- [x] **Step 2: Find any remaining readers.**

Run: `cd services/atlas-ui && grep -rn "attributes.mapId\|\.mapId" src --include=*.ts --include=*.tsx | grep -vi location`
Expected: no character-resource readers remain (location-typed `.mapId` reads are fine). Fix any stragglers (e.g. an optimistic-update spread in `useCharacters.ts` that referenced `mapId`).

- [x] **Step 3: Type-check + tests (the build type-checks tests too).**

Run: `cd services/atlas-ui && npm run build && npx vitest run`
Expected: clean / PASS. If `useUpdateCharacter`'s optimistic update or any test still references `mapId`, update it in this commit.

- [x] **Step 4: Commit.**

```bash
git add services/atlas-ui/src/types/models/character.ts services/atlas-ui/src/lib/hooks/api/useCharacters.ts
git commit -m "refactor(atlas-ui): drop mapId from character type (task-087)"
git branch --show-current
```

---

## Task 7: atlas-parties — full-field member construction from location

Build foreign-member fields from atlas-maps location (full world/channel/map/instance) and drop the dead `MapId` mirror field (FR-6.1).

**Files:**
- Create: `services/atlas-parties/atlas.com/parties/location/requests.go`
- Modify: `services/atlas-parties/atlas.com/parties/character/processor.go:260-272`
- Modify: `services/atlas-parties/atlas.com/parties/character/rest.go` (drop `MapId` at lines 38 and 103)

- [x] **Step 1: Add the location client.** Create `location/requests.go` — a verbatim copy of atlas-character's client (`services/atlas-character/atlas.com/character/location/requests.go`), package `location`, module-local imports unchanged (it only depends on shared libs):

```go
package location

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var ErrNotFound = errors.New("location not found")

const Resource = "characters/%d/location"

type RestModel struct {
	Id        uint32     `json:"-"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

func (r RestModel) GetName() string { return "character-locations" }
func (r RestModel) GetID() string   { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *RestModel) SetID(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

var baseURLProvider = func() string { return requests.RootUrl("MAPS") }

func requestByCharacterId(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(baseURLProvider()+Resource, characterId))
}

func GetField(l logrus.FieldLogger, ctx context.Context, characterId uint32) (field.Model, error) {
	rm, err := requestByCharacterId(characterId)(l, ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return field.Model{}, ErrNotFound
		}
		return field.Model{}, err
	}
	return field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(), nil
}
```

- [x] **Step 2: Switch the member field construction to location.** In `character/processor.go`, change the `ByIdProvider` fallback (currently `f := field.NewBuilder(fm.WorldId(), 0, fm.MapId()).Build()` at line 268) to:

```go
fm, ferr := p.GetForeignCharacterInfo(characterId)
if ferr != nil {
	return Model{}, err
}
f := field.NewBuilder(fm.WorldId(), 0, 0).Build()
if lf, lerr := location.GetField(p.l, p.ctx, characterId); lerr == nil {
	f = lf
}
c = GetRegistry().Create(p.ctx, f, characterId, fm.Name(), fm.Level(), fm.JobId(), fm.GM())
```

Add the import `"atlas-parties/location"`. On a location lookup failure (incl. `ErrNotFound`), the field falls back to world-only with channel/map 0 — at least as good as the prior hardcoded-channel-0 behavior, and no longer relies on the soon-deleted echo.

- [x] **Step 3: Drop the `MapId` mirror field.** In `character/rest.go`, remove the `MapId _map.Id \`json:"mapId"\`` field (line 38) and the `mapId: rm.MapId,` line in `ExtractForeign` (line 103). Remove the `_map` import if it becomes unused. Remove the `ForeignModel.mapId` field and its getter if present (`character/model.go`) — grep first; if `ForeignModel.MapId()` has no remaining callers after Step 2, delete it.

Run: `cd services/atlas-parties/atlas.com/parties && grep -rn "MapId\|mapId\|\.mapId" character/ | grep -v _test.go`
Expected: no references to the foreign-character mapId remain.

- [x] **Step 4: Build, vet, race-test.**

Run: `cd services/atlas-parties/atlas.com/parties && go build ./... && go vet ./... && go test -race ./...`
Expected: clean. Fix any test that constructed a `ForeignRestModel`/`ForeignModel` with `MapId`.

- [x] **Step 5: Commit.**

```bash
git add services/atlas-parties/atlas.com/parties/
git commit -m "refactor(atlas-parties): build member field from atlas-maps location; drop mapId mirror (task-087)"
git branch --show-current
```

---

## Task 8: atlas-consumables — source summoning field from location

`ConsumeSummoningSack` reads `c.MapId()` (the atlas-character mirror) at two sites. Replace with the atlas-maps location field; drop the mirror `MapId` (FR-6.2 active). The in-memory `map/character` registry `GetMap` (used elsewhere) is unrelated and untouched.

**Files:**
- Create: `services/atlas-consumables/atlas.com/consumables/location/requests.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (`ConsumeSummoningSack`, ~lines 419-440)
- Modify: `services/atlas-consumables/atlas.com/consumables/character/rest.go:38` and `character/model.go:207`

- [x] **Step 1: Add the location client.** Create `location/requests.go` — the same client as Task 7 Step 1, but `package location` under the consumables module (imports are shared libs only, so the file is byte-identical to Task 7's). Copy it verbatim.

- [x] **Step 2: Repoint the two `c.MapId()` reads.** In `ConsumeSummoningSack`, after `c, ci := fc.Get(), fi.Get()`, fetch the location field once and use it for both the position lookup and the spawn field:

```go
c, ci := fc.Get(), fi.Get()

lf, lerr := location.GetField(l, ctx, characterId)
if lerr != nil {
	return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, lerr)
}

// Dependent read: needs the character's map (from atlas-maps) + temporal X/Y.
pos, err := position.NewProcessor(l, ctx).GetInMap(lf.MapId(), c.X(), c.Y(), c.X(), c.Y())()
if err != nil {
	return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
}

l.Debugf("Character [%d] summoning [%d] monsters at [%d,%d]. They are at [%d,%d].", characterId, len(ci.MonsterSummons()), pos.X(), pos.Y(), c.X(), c.Y())
for _, msm := range ci.MonsterSummons() {
	roll := uint32(rand.Int31n(100))
	if roll < msm.Probability() {
		f := field.NewBuilder(ch.WorldId(), ch.Id(), lf.MapId()).Build()
		err = monster.NewProcessor(l, ctx).CreateMonster(f, msm.TemplateId(), pos.X(), pos.Y(), 0, 0)
		// ... unchanged ...
	}
}
```

Add the import `"atlas-consumables/location"`. (`field`, `ch`, `c.X()`, `c.Y()` are already in scope.)

- [x] **Step 3: Drop the mirror `MapId`.** Remove `MapId _map.Id \`json:"mapId"\`` from `character/rest.go:38`, the `mapId: rm.MapId,` in `Extract`, and the `MapId()` getter at `character/model.go:207` plus the `mapId` struct field. Remove the now-unused `_map` import in those files if applicable.

Run: `cd services/atlas-consumables/atlas.com/consumables && grep -rn "c.MapId()\|m.MapId()\|\.MapId()" consumable/ character/ | grep -v _test.go`
Expected: the only remaining `.MapId()` calls are on `field.Model` values (`m` from `GetMap`, `lf` from location, `toField`), **never** on the `character.Model` (`c`).

- [x] **Step 4: Build, vet, race-test.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./... && go vet ./... && go test -race ./...`
Expected: clean. Fix any test constructing the character `RestModel`/`Model` with `MapId`.

- [x] **Step 5: Commit.**

```bash
git add services/atlas-consumables/atlas.com/consumables/
git commit -m "refactor(atlas-consumables): summoning sack uses atlas-maps location; drop mapId mirror (task-087)"
git branch --show-current
```

---

## Task 9: atlas-query-aggregator — MapCondition from location

`MapCondition` validation reads `character.MapId()`. Repoint it to an atlas-maps location lookup and drop the mirror field (FR-6.2/6.3 — internal use only, no public-contract change).

**Files:**
- Create: `services/atlas-query-aggregator/atlas.com/query-aggregator/location/requests.go`
- Modify: `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go:~397`
- Modify: `services/atlas-query-aggregator/atlas.com/query-aggregator/character/rest.go:39` and `character/model.go:218`

- [x] **Step 1: Add the location client.** Create `location/requests.go` — same client as Task 7 Step 1, `package location` under the query-aggregator module. Copy verbatim.

- [x] **Step 2: Read the `MapCondition` evaluation context.** Run: `sed -n '360,450p' services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` to see how `character` and the logger/context are available where `actualValue = int(character.MapId())` runs.

- [x] **Step 3: Repoint the condition.** Replace `actualValue = int(character.MapId())` with a location lookup. The validation runs with a logger + context in scope (the same ones used for the character fetch); use them:

```go
case MapCondition:
	lf, lerr := location.GetField(l, ctx, character.Id())
	if lerr != nil {
		// No location ⇒ cannot satisfy a map condition; treat as map 0 (fails
		// any positive map check) rather than erroring the whole validation.
		l.WithError(lerr).Warnf("MapCondition: location unavailable for character [%d]; using map 0.", character.Id())
		actualValue = 0
	} else {
		actualValue = int(lf.MapId())
	}
	description = fmt.Sprintf("Map ID %s %d", c.operator, c.value)
```

> Note: confirm the exact identifiers for the logger (`l`/`p.l`) and context
> (`ctx`/`p.ctx`) in scope at this `case` (Step 2 output). Confirm
> `character.Id()` is the getter (the model has `Id()`). Add the `location`
> import. If the evaluation function does not currently receive a logger/context,
> thread them in from the caller (the function that already fetched the character
> via the REST client has them).

- [x] **Step 4: Drop the mirror `MapId`.** Remove `MapId _map.Id \`json:"mapId"\`` from `character/rest.go:39`, the `mapId: m.MapId,` in `Extract` (line ~168), and the `MapId()` getter + `mapId` field at `character/model.go:218-220`. Remove unused `_map` import where applicable.

Run: `cd services/atlas-query-aggregator/atlas.com/query-aggregator && grep -rn "character.MapId()\|\.MapId()\|mapId" character/ validation/ | grep -v _test.go`
Expected: no remaining read of the character mirror's `MapId`; `MapCondition` now reads `lf.MapId()`.

- [x] **Step 5: Build, vet, race-test.**

Run: `cd services/atlas-query-aggregator/atlas.com/query-aggregator && go build ./... && go vet ./... && go test -race ./...`
Expected: clean. Update any `MapCondition` test to stub the location lookup (or assert via the location `baseURLProvider` seam) — confirm how existing validation tests inject the character; mirror that for location.

- [x] **Step 6: Commit.**

```bash
git add services/atlas-query-aggregator/atlas.com/query-aggregator/
git commit -m "refactor(atlas-query-aggregator): MapCondition reads atlas-maps location; drop mapId mirror (task-087)"
git branch --show-current
```

---

## Task 10: Passive services — strip the dead `MapId` mirror field

Five services declare `MapId` on their atlas-character mirror but never read it. Remove the field, its `Extract`/builder assignment, and the getter (FR-6.2/6.4). Decoding tolerates the absent JSON key.

Do each service as its own commit. The edit is mechanical: delete the struct field, the extract assignment, the model getter, and the `mapId` model field; drop a now-unused `_map` import if it falls out.

**Files (per service):**

| Service (module) | rest.go field | extract/builder | model getter + field |
|---|---|---|---|
| atlas-channel (`atlas-channel`) | `character/rest.go:38` | `character/rest.go:116` `mapId: m.MapId,` | `character/model.go:217` |
| atlas-login (`atlas-login`) | `character/rest.go:38` | `character/rest.go:125` `SetMapId(m.MapId).` | `character/model.go:205` |
| atlas-npc-shops (`atlas-npc`) | `character/rest.go:38` | `character/rest.go:125` `mapId: rm.MapId,` | `character/model.go:202` |
| atlas-cashshop (`atlas-cashshop`) | `character/rest.go:38` | `character/rest.go:116` `mapId: m.MapId,` | `character/model.go:205` |
| atlas-messengers (`atlas-messengers`) | `character/rest.go:37` (`ForeignRestModel`) | `character/rest.go:102` `mapId: rm.MapId,` | `character/model.go:128` (`ForeignModel`) |

- [x] **Step 1 (atlas-channel):** Remove the `MapId` field (`character/rest.go:38`), the `mapId: m.MapId,` extract line, and the `MapId()` getter + `mapId` field in `model.go`. Verify nothing reads it: `cd services/atlas-channel/atlas.com/channel && grep -rn "\.MapId()\|mapId" character/ | grep -v _test.go` — remaining `.MapId()` calls must be on live-session `field.Model` (e.g. `portal/processor.go`), not the character mirror. Then `go build ./... && go vet ./... && go test -race ./...`.

```bash
git add services/atlas-channel/atlas.com/channel/character/
git commit -m "refactor(atlas-channel): strip dead mapId character mirror field (task-087)"
git branch --show-current
```

- [x] **Step 2 (atlas-login):** Same removal (note the builder form `SetMapId(m.MapId)` — remove that builder call; if `SetMapId` becomes unused on the builder, remove the builder method too). Verify: `cd services/atlas-login/atlas.com/login && grep -rn "\.MapId()\|MapId\|mapId" character/ | grep -v _test.go` — `character_list.go` map use comes from `location.GetField`, not the mirror. Then build/vet/test.

```bash
git add services/atlas-login/atlas.com/login/character/
git commit -m "refactor(atlas-login): strip dead mapId character mirror field (task-087)"
git branch --show-current
```

- [x] **Step 3 (atlas-npc-shops):** Same removal (module is `atlas-npc`). Verify the only `MapId` references left are Kafka event bodies in `kafka/message/character/`, not the GET mirror. Build/vet/test.

```bash
git add services/atlas-npc-shops/atlas.com/npc/character/
git commit -m "refactor(atlas-npc-shops): strip dead mapId character mirror field (task-087)"
git branch --show-current
```

- [x] **Step 4 (atlas-cashshop):** Same removal. Verify no `.MapId()` reads remain. Build/vet/test.

```bash
git add services/atlas-cashshop/atlas.com/cashshop/character/
git commit -m "refactor(atlas-cashshop): strip dead mapId character mirror field (task-087)"
git branch --show-current
```

- [x] **Step 5 (atlas-messengers):** Remove from `ForeignRestModel` (`rest.go:37`), `ExtractForeign` (`rest.go:102`), and `ForeignModel.MapId()` + `mapId` field (`model.go:128`). Verify only Kafka event bodies reference `MapId`. Build/vet/test.

```bash
git add services/atlas-messengers/atlas.com/messengers/character/
git commit -m "refactor(atlas-messengers): strip dead mapId character mirror field (task-087)"
git branch --show-current
```

---

## Task 11: atlas-character — retire the location shim (LAST)

Only after Tasks 7–10: delete the dead `Update` branch, remove the `Transform` shim and the GET `MapId`/`Instance` projection, and drop the `Instance` struct field. Keep `MapId` as create input (FR-3.1, FR-4.1–4.3, FR-4.5).

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/processor.go` (delete the `if input.MapId != 0 { … Debug … }` block)
- Modify: `services/atlas-character/atlas.com/character/character/rest.go` (Transform shim, projection, `Instance` field)

- [x] **Step 1: Delete the dead write branch.** In `processor.go`, remove the entire block (the comment + the `if input.MapId != 0 { p.l.WithFields(...).Debug(...) }`). Leave the surrounding `changes`/early-return logic intact.

- [x] **Step 2: Remove the Transform shim.** In `rest.go`, change `Transform` so it no longer calls `location.GetField`. The GET no longer emits location:

```go
// Transform produces the JSON:API projection for a character. MapId/Instance
// are NOT part of the GET projection — atlas-maps owns location (task-087).
// MapId remains on RestModel as a create-time input only (see Extract / POST).
func Transform(l logrus.FieldLogger, ctx context.Context) func(m Model) (RestModel, error) {
	t := tenant.MustFromContext(ctx)
	return func(m Model) (RestModel, error) {
		td := GetTemporalRegistry().GetById(ctx, t, m.Id())
		return transformWithTemporal(m, td), nil
	}
}
```

Update `transformWithTemporal` to drop the `f field.Model` parameter and the `MapId`/`Instance` assignments:

```go
func transformWithTemporal(m Model, td temporalData) RestModel {
	rm := RestModel{
		Id:                 m.Id(),
		AccountId:          m.AccountId(),
		WorldId:            m.WorldId(),
		Name:               m.Name(),
		Level:              m.Level(),
		Experience:         m.Experience(),
		GachaponExperience: m.GachaponExperience(),
		Strength:           m.Strength(),
		Dexterity:          m.Dexterity(),
		Intelligence:       m.Intelligence(),
		Luck:               m.Luck(),
		Hp:                 m.Hp(),
		MaxHp:              m.MaxHp(),
		Mp:                 m.Mp(),
		MaxMp:              m.MaxMp(),
		Meso:               m.Meso(),
		HpMpUsed:           m.HpMpUsed(),
		JobId:              m.JobId(),
		SkinColor:          m.SkinColor(),
		Gender:             m.Gender(),
		Fame:               m.Fame(),
		Hair:               m.Hair(),
		Face:               m.Face(),
		Ap:                 m.AP(),
		Sp:                 m.SPString(),
		SpawnPoint:         m.SpawnPoint(),
		Gm:                 m.GM(),
		X:                  td.X(),
		Y:                  td.Y(),
		Stance:             td.Stance(),
	}
	return rm
}
```

- [x] **Step 3: Drop `Instance` from `RestModel`; keep `MapId` (input-only).** In the `RestModel` struct, remove the `Instance uuid.UUID \`json:"instance"\`` field (line 45). Keep `MapId _map.Id \`json:"mapId"\`` (line 44) and add a comment noting it is create-input only and absent from GET responses. Remove the `field`/`location`/`uuid` imports from `rest.go` **only if** they become unused after removing the shim (the file may still use `uuid` elsewhere — grep before deleting). `Extract` keeps populating from `input.MapId` for create — leave that path as-is.

- [x] **Step 4: Verify no other call site broke.** `Transform` and `transformWithTemporal` signatures changed (the `f` param is gone).

Run: `cd services/atlas-character/atlas.com/character && grep -rn "transformWithTemporal\|location.GetField" character/`
Expected: `transformWithTemporal` callers updated to the new 2-arg signature; `location.GetField` still appears at `processor.go:391, 425, 1144, 1194` (those stay) but **no longer** in `rest.go`.

- [x] **Step 5: Build, vet, race-test.**

Run: `cd services/atlas-character/atlas.com/character && go build ./... && go vet ./... && go test -race ./...`
Expected: clean. Update any test asserting `mapId`/`instance` on a GET response (e.g. a rest/transform test) to no longer expect them; update tests calling `transformWithTemporal` with the old 3-arg signature.

- [x] **Step 6: Commit.**

```bash
git add services/atlas-character/atlas.com/character/character/
git commit -m "refactor(atlas-character): remove GET location shim and dead mapId update branch (task-087)"
git branch --show-current
```

---

## Task 12: Full verification gate

Run the complete CLAUDE.md gate across every changed module before declaring done.

- [x] **Step 1: Per-module Go gate.** From the worktree root, for each changed Go module run build + vet + race tests:

```bash
for m in \
  services/atlas-maps/atlas.com/maps \
  services/atlas-parties/atlas.com/parties \
  services/atlas-consumables/atlas.com/consumables \
  services/atlas-query-aggregator/atlas.com/query-aggregator \
  services/atlas-channel/atlas.com/channel \
  services/atlas-login/atlas.com/login \
  services/atlas-npc-shops/atlas.com/npc \
  services/atlas-cashshop/atlas.com/cashshop \
  services/atlas-messengers/atlas.com/messengers \
  services/atlas-character/atlas.com/character ; do
  echo "== $m ==" && (cd "$m" && go build ./... && go vet ./... && go test -race ./...) || break
done
```

Expected: every module clean.

- [x] **Step 2: redis-key-guard.** From the repo/worktree root:

```bash
GOWORK=off tools/redis-key-guard.sh
```

Expected: clean (this task adds no raw go-redis usage).

- [x] **Step 3: docker bake for any service whose `go.mod`/`go.sum` changed.** Check first:

```bash
git diff --name-only main... | grep -E 'services/.*/go\.(mod|sum)$' || echo "no go.mod/go.sum changes"
```

For each service listed (and, conservatively per design §10, for `atlas-maps`), bake from the worktree root:

```bash
docker buildx bake atlas-maps atlas-parties atlas-consumables atlas-query-aggregator atlas-channel atlas-login atlas-npc-shops atlas-cashshop atlas-messengers atlas-character
```

Expected: all targets build. (No new shared lib is added, so no Dockerfile/`go.work` edit is required.)

- [x] **Step 4: atlas-ui build + tests.**

```bash
cd services/atlas-ui && npm run build && npx vitest run
```

Expected: clean / PASS.

- [x] **Step 5: Final commit (only if Step 3 surfaced doc/lockfile changes or any fix was needed).**

```bash
git add -A && git commit -m "chore(task-087): verification gate fixes"
git branch --show-current
```

---

## Self-Review

**Spec coverage (design.md / prd.md):**
- FR-1.1–1.6 (write endpoint, 204) → Task 2. FR-1.4 shared method → Task 1.
- FR-2.1–2.3 (map validation 400, 404 no-row) → Task 2.
- FR-3.1 dead branch / FR-3.2 create-input → Task 11.
- FR-4.1–4.5 (Transform shim, projection, `Instance` removed, `location` client stays) → Task 11.
- FR-5.1–5.5 (UI read/write/table/type/envelope) → Tasks 3–6.
- FR-6.1 parties / FR-6.2 passive+consumables / FR-6.3 query-aggregator / FR-6.4 no readers left → Tasks 7–10.
- FR-7.1 emit+persist → Task 1/2 tests; FR-7.2 single method (command + REST seams) → Task 1 Step 5 + Task 2 test; FR-7.3 invalid 400 / no-row 404 → Task 2 tests; FR-7.4 UI dialog → Tasks 3–4 tests.
- Verification gate (design §10) → Task 12.
- Design §2 decisions (204, 404, info.GetById, stored channel, per-row query) all encoded in Tasks 2 and 5.

**Type consistency:** `warp.Processor.ChangeMap(uuid.UUID, uint32, world.Id, field.Model, uint32) error` is used identically in Task 1 (consumer seam), Task 1 test, and Task 2 (`warpProcessor` interface + handler). `changeCharacterLocation(logrus.FieldLogger, Processor, info.Processor, warpProcessor, uint32, _map.Id) int` matches its tests. The location-client `GetField(logrus.FieldLogger, context.Context, uint32) (field.Model, error)` + `ErrNotFound` is identical across Tasks 7–9. UI `locationsService.changeMap(id, {mapId})` / `getByCharacterId(id)` and `useCharacterLocation(tenant, id)` are consistent across Tasks 3–5.

**Placeholder scan:** Remaining `// Note:` blocks are explicit verify-against-source instructions (migration-helper name, exact import paths, component props), not deferred work — each names the file to check and the concrete fallback. No "TBD"/"implement later"/"add error handling" placeholders.
