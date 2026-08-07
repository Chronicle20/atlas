# Instance Route Effect Lifecycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `atlas-transports` the single owner of an instance transport trip — it applies the route's declared consumable effects on boarding, cancels them on every one of the five terminal paths, and honours an optional per-route forced-return map when the travel timer expires.

**Architecture:** Two optional attributes (`effectItemIds`, `forcedReturnMapId`) are added to the instance-route configuration and threaded through **five** projection layers (see Global Constraints). `instance.ProcessorImpl` gains two private helpers that buffer `APPLY_CONSUMABLE_EFFECT` / `CANCEL_CONSUMABLE_EFFECT` commands onto the existing `COMMAND_TOPIC_CONSUMABLE`; the helpers never return an error so a Kafka buffer failure can never block instance teardown. The now-duplicated effect operations are deleted from four seed files across eleven version directories.

**Tech Stack:** Go 1.x, `libs/atlas-constants` (`item.Id`, `_map.Id`, `world.Id`, `channel.Id`, `character.Id`), `libs/atlas-kafka` (`producer`, `message.Buffer`), `libs/atlas-redis` (`TenantRegistry`), `libs/atlas-tenant`, `miniredis` + `testify` for tests, JSON seed catalogs under `deploy/seed/`.

## Global Constraints

- **Both new fields are optional.** A route declaring neither must behave byte-identically to today. This is the regression bar for the ten unaffected routes (PRD §8).
- **Five projection layers, not two.** Every one silently drops unknown fields. A field missing from any layer produces a zero value with no error:
  | # | Layer | File |
  |---|---|---|
  | 0 | REST → JSONB (CRUD write-back) | `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` `ExtractInstanceRoute` |
  | 1 | JSONB → tenant REST | `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` `TransformInstanceRoute` |
  | 2 | REST → domain | `services/atlas-transports/atlas.com/transports/instance/config/rest.go` |
  | 3 | **domain → Redis → domain** | `services/atlas-transports/atlas.com/transports/instance/model_json.go` |
  | 4 | domain → debug REST | `services/atlas-transports/atlas.com/transports/instance/rest.go` `TransformRoute` |
  Layer 3 is the dangerous one: `RouteRegistry` is a Redis-backed `atlas.TenantRegistry` (`instance/route_registry.go:16`), so **every** processor route read round-trips through `MarshalJSON`/`UnmarshalJSON`. Layer 0 is not in design.md's table — it is added here because `ExtractInstanceRoute` is what a POST/PATCH through `atlas-tenants` writes back to JSONB (`configuration/resource.go:509,558`); omitting the fields there means an operator PATCH silently erases them.
- **Attribute names are fixed:** `effectItemIds` (array of item ids) and `forcedReturnMapId` (single map id). `forcedReturnMapId` mirrors the client's own WZ node name (`Map.wz/Map/Map2/200090500.img.xml` → `forcedReturn = 240000110`).
- **Domain types come from `libs/atlas-constants`** per DOM-21: `[]item.Id` and `_map.Id`. Never `[]uint32` in `atlas-transports` domain code. (`atlas-tenants`' own REST model does use plain `uint32`, matching every other field there.)
- **Effect item id `2210016`, forced-return map `240000110`** for both flight routes. Verified from WZ in prd.md Appendix A; do not substitute remembered values.
- **Emission failures never abort teardown.** Both new helpers return nothing and log-and-continue. Leaking a buff is bad; leaking an instance is worse.
- **`TransactionId` on emitted consumable commands is `uuid.Nil`,** deliberately. `atlas-saga-orchestrator` treats a nil transaction id on the resulting `EFFECT_APPLIED` event as a non-saga effect application and skips saga completion (`saga-orchestrator/kafka/consumer/consumable/consumer.go:45-48`). A fresh `uuid.New()` would be semantically wrong.
- **No `go.mod` change is expected** in either service — `item` and `character` live in `libs/atlas-constants`, already a dependency of both. If a `go.mod` does change, `docker buildx bake` for that service becomes mandatory (CLAUDE.md §Build & Verification item 4).
- **Preserve LF line endings** in every seed file. All 55 affected seed files are LF today; do not let an editor rewrite them.
- **No `// TODO`, no stubs.** CLAUDE.md §No Deferring Producible Work.

---

## File Structure

**`atlas-transports` (`services/atlas-transports/atlas.com/transports/`)**

| File | Responsibility | Action |
|---|---|---|
| `instance/model.go` | `RouteModel` immutable domain model | Modify — 2 fields + 2 getters |
| `instance/builder.go` | `RouteBuilder` + validation | Modify — 2 setters + zero-item-id check |
| `instance/model_json.go` | Redis serialization (layer 3) | Modify — 2 fields in all three places |
| `instance/model_json_test.go` | Layer-3 round-trip guard | **Create** |
| `instance/config/rest.go` | atlas-tenants REST → domain (layer 2) | Modify |
| `instance/rest.go` | domain → debug REST (layer 4) | Modify |
| `kafka/message/consumable/kafka.go` | Local wire contract for `COMMAND_TOPIC_CONSUMABLE` | **Create** |
| `kafka/message/instance_transport/kafka.go` | Instance-transport event contract | Modify — `CancelReasonTimeout` |
| `instance/producer.go` | Kafka message providers for this package | Modify — 2 providers |
| `instance/producer_test.go` | Wire-shape guard for the 2 providers | **Create** |
| `instance/processor.go` | Instance lifecycle | Modify — 2 helpers, 6 call sites, forced return, teardown hardening |
| `instance/processor_test.go` | Lifecycle behaviour | **Create** |
| `docs/domain.md`, `docs/kafka.md` | Service docs | Modify |

**`atlas-tenants` (`services/atlas-tenants/atlas.com/tenants/`)**

| File | Responsibility | Action |
|---|---|---|
| `configuration/rest.go` | Instance-route REST model + JSONB projections (layers 0 & 1) | Modify |
| `configuration/rest_test.go` | Projection guards | Modify — 3 tests added |

**Seeds (`deploy/seed/`)**

| Path | Action |
|---|---|
| `shared/all/instance-routes/flight-leafre-temple-of-time.json` | Modify — declare both attributes |
| `shared/all/instance-routes/flight-temple-of-time-leafre.json` | Modify — declare both attributes |
| `{gms/{12,48,61,72,79,83,84,87,92,95}_1,jms/185_1}/npc-conversations/npc/npc-2082003.json` | Modify ×11 — collapse `applyBuff` state |
| `.../portal-actions/portals/portal-outTemple.json` | Modify ×11 — drop apply op |
| `.../portal-actions/portals/portal-templeenter.json` | Modify ×11 — drop cancel op |
| `.../portal-actions/portals/portal-undodraco.json` | Modify ×11 — drop cancel op |
| `.../portal-actions/portals/portal-dracoout.json` | **No edit** — the leak closes via `HandleMapEnter` |

---

### Task 1: Route model, builder, and Redis round-trip

Adds both fields to the domain model and — critically — to the Redis JSON codec. Layer 3 is done first because every later task depends on the getters existing and on the values surviving a registry read.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/model.go`
- Modify: `services/atlas-transports/atlas.com/transports/instance/builder.go`
- Modify: `services/atlas-transports/atlas.com/transports/instance/model_json.go`
- Create: `services/atlas-transports/atlas.com/transports/instance/model_json_test.go`
- Modify: `services/atlas-transports/atlas.com/transports/instance/builder_test.go`

**Interfaces:**
- Consumes: nothing (first task).
- Produces:
  - `func (m RouteModel) EffectItemIds() []item.Id` — defensive copy
  - `func (m RouteModel) ForcedReturnMapId() _map.Id` — zero means "not set"
  - `func (b *RouteBuilder) SetEffectItemIds(effectItemIds []item.Id) *RouteBuilder`
  - `func (b *RouteBuilder) SetForcedReturnMapId(forcedReturnMapId _map.Id) *RouteBuilder`
  - `Build()` returns an error containing the text `effect item ids` when the list holds a zero id.

- [ ] **Step 1: Write the failing round-trip test**

Create `services/atlas-transports/atlas.com/transports/instance/model_json_test.go`:

```go
package instance

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// RouteRegistry is a Redis-backed atlas.TenantRegistry (route_registry.go:16),
// so EVERY route read in the processor round-trips through
// MarshalJSON/UnmarshalJSON. A field added to the model, the builder and the
// config extractor but NOT to routeModelJSON would be zero at every processor
// call site while passing every other test in this package. This is that guard.
func TestRouteModel_JSONRoundTripPreservesEffectFields(t *testing.T) {
	want, err := NewRouteBuilder("temple-of-time-flight").
		SetStartMapId(_map.Id(240000110)).
		SetTransitMapIds([]_map.Id{200090500, 200090510}).
		SetDestinationMapId(_map.Id(270000100)).
		SetCapacity(1).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(900 * time.Second).
		SetTransitMessage("You are flying towards the Temple of Time.").
		SetEffectItemIds([]item.Id{2210016}).
		SetForcedReturnMapId(_map.Id(240000110)).
		Build()
	assert.NoError(t, err)

	data, err := json.Marshal(want)
	assert.NoError(t, err)

	var got RouteModel
	assert.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, []item.Id{2210016}, got.EffectItemIds())
	assert.Equal(t, _map.Id(240000110), got.ForcedReturnMapId())
	assert.Equal(t, want.Name(), got.Name())
	assert.Equal(t, want.TransitMapIds(), got.TransitMapIds())
}

// A route declaring neither field must survive the round trip with both at
// their zero values — the regression bar for the ten unaffected routes.
func TestRouteModel_JSONRoundTripWithoutEffectFields(t *testing.T) {
	want, err := NewRouteBuilder("ellinia-ereve-ferry").
		SetTransitMapIds([]_map.Id{200090030}).
		SetCapacity(20).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(60 * time.Second).
		Build()
	assert.NoError(t, err)

	data, err := json.Marshal(want)
	assert.NoError(t, err)

	var got RouteModel
	assert.NoError(t, json.Unmarshal(data, &got))

	assert.Empty(t, got.EffectItemIds())
	assert.Equal(t, _map.Id(0), got.ForcedReturnMapId())
}

// EffectItemIds hands out a copy, matching TransitMapIds. Mutating the
// returned slice must not reach back into the model.
func TestRouteModel_EffectItemIdsIsDefensiveCopy(t *testing.T) {
	route, err := NewRouteBuilder("temple-of-time-flight").
		SetTransitMapIds([]_map.Id{200090500}).
		SetCapacity(1).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(900 * time.Second).
		SetEffectItemIds([]item.Id{2210016}).
		Build()
	assert.NoError(t, err)

	got := route.EffectItemIds()
	got[0] = item.Id(1)

	assert.Equal(t, []item.Id{2210016}, route.EffectItemIds())
}
```

- [ ] **Step 2: Write the failing builder-validation tests**

Append to `services/atlas-transports/atlas.com/transports/instance/builder_test.go` (add `"github.com/Chronicle20/atlas/libs/atlas-constants/item"` to its import block):

```go
func TestRouteBuilder_RejectsZeroEffectItemId(t *testing.T) {
	_, err := NewRouteBuilder("test").
		SetTransitMapIds([]_map.Id{100}).
		SetCapacity(6).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		SetEffectItemIds([]item.Id{2210016, 0}).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "effect item ids")
}

// Zero forced-return means "not set", never an error (FR-4.3).
func TestRouteBuilder_ZeroForcedReturnMapIdIsNotAnError(t *testing.T) {
	route, err := NewRouteBuilder("test").
		SetTransitMapIds([]_map.Id{100}).
		SetCapacity(6).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		SetForcedReturnMapId(_map.Id(0)).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, _map.Id(0), route.ForcedReturnMapId())
}

// Neither field is required.
func TestRouteBuilder_EffectFieldsAreOptional(t *testing.T) {
	route, err := NewRouteBuilder("test").
		SetTransitMapIds([]_map.Id{100}).
		SetCapacity(6).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()

	assert.NoError(t, err)
	assert.Empty(t, route.EffectItemIds())
	assert.Equal(t, _map.Id(0), route.ForcedReturnMapId())
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./instance/ -run 'RouteModel_JSON|RouteBuilder_(RejectsZero|ZeroForced|EffectFields)|EffectItemIdsIsDefensive' -v`

Expected: compile failure — `route.EffectItemIds undefined`, `SetEffectItemIds undefined`, `SetForcedReturnMapId undefined`, `ForcedReturnMapId undefined`.

- [ ] **Step 4: Add the model fields and getters**

In `instance/model.go`, add `"github.com/Chronicle20/atlas/libs/atlas-constants/item"` to the import block (alphabetically after `channel`), extend the struct, and add the two getters after `TransitMessage()`:

```go
type RouteModel struct {
	id                uuid.UUID
	name              string
	startMapId        _map.Id
	transitMapIds     []_map.Id
	destinationMapId  _map.Id
	capacity          uint32
	boardingWindow    time.Duration
	travelDuration    time.Duration
	transitMessage    string
	effectItemIds     []item.Id
	forcedReturnMapId _map.Id
}
```

```go
// EffectItemIds are the consumable item ids whose effects this route applies
// on boarding and cancels on every terminal path. Empty means the route
// applies no effects. The copy matches TransitMapIds: reads never hand out
// the backing array, which is what makes the model immutable in practice.
func (m RouteModel) EffectItemIds() []item.Id {
	result := make([]item.Id, len(m.effectItemIds))
	copy(result, m.effectItemIds)
	return result
}

// ForcedReturnMapId is the map TickArrival warps to when the travel timer
// expires. Zero means "not set" — deliver to destinationMapId instead. It
// mirrors the client's own Map.wz info/forcedReturn node, which is only
// meaningful on maps that also carry a timeLimit.
func (m RouteModel) ForcedReturnMapId() _map.Id {
	return m.forcedReturnMapId
}
```

- [ ] **Step 5: Add the builder setters and validation**

In `instance/builder.go`, add the `item` import, extend `RouteBuilder` with `effectItemIds []item.Id` and `forcedReturnMapId _map.Id`, add the setters after `SetTransitMessage`:

```go
func (b *RouteBuilder) SetEffectItemIds(effectItemIds []item.Id) *RouteBuilder {
	b.effectItemIds = effectItemIds
	return b
}

func (b *RouteBuilder) SetForcedReturnMapId(forcedReturnMapId _map.Id) *RouteBuilder {
	b.forcedReturnMapId = forcedReturnMapId
	return b
}
```

In `Build()`, add the validation after the `transitMapIds` check and both fields to the returned model:

```go
	for _, id := range b.effectItemIds {
		if id == 0 {
			return RouteModel{}, errors.New("effect item ids must not contain a zero id")
		}
	}

	return RouteModel{
		id:                b.id,
		name:              b.name,
		startMapId:        b.startMapId,
		transitMapIds:     b.transitMapIds,
		destinationMapId:  b.destinationMapId,
		capacity:          b.capacity,
		boardingWindow:    b.boardingWindow,
		travelDuration:    b.travelDuration,
		transitMessage:    b.transitMessage,
		effectItemIds:     b.effectItemIds,
		forcedReturnMapId: b.forcedReturnMapId,
	}, nil
```

Note: `forcedReturnMapId` gets no validation — zero is a legitimate "not set".

- [ ] **Step 6: Add both fields to the Redis JSON codec**

In `instance/model_json.go`, add the `item` import and both fields in all three places:

```go
type routeModelJSON struct {
	Id                uuid.UUID     `json:"id"`
	Name              string        `json:"name"`
	StartMapId        _map.Id       `json:"startMapId"`
	TransitMapIds     []_map.Id     `json:"transitMapIds"`
	DestinationMapId  _map.Id       `json:"destinationMapId"`
	Capacity          uint32        `json:"capacity"`
	BoardingWindow    time.Duration `json:"boardingWindow"`
	TravelDuration    time.Duration `json:"travelDuration"`
	TransitMessage    string        `json:"transitMessage"`
	EffectItemIds     []item.Id     `json:"effectItemIds"`
	ForcedReturnMapId _map.Id       `json:"forcedReturnMapId"`
}
```

Add `EffectItemIds: m.effectItemIds,` and `ForcedReturnMapId: m.forcedReturnMapId,` to the `MarshalJSON` literal, and `m.effectItemIds = j.EffectItemIds` / `m.forcedReturnMapId = j.ForcedReturnMapId` to `UnmarshalJSON`.

- [ ] **Step 7: Run the tests to verify they pass**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./instance/ -v`

Expected: PASS, including every pre-existing test in the package.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/model.go \
        services/atlas-transports/atlas.com/transports/instance/builder.go \
        services/atlas-transports/atlas.com/transports/instance/model_json.go \
        services/atlas-transports/atlas.com/transports/instance/model_json_test.go \
        services/atlas-transports/atlas.com/transports/instance/builder_test.go
git commit -m "feat(transports): add effectItemIds and forcedReturnMapId to instance RouteModel"
```

---

### Task 2: atlas-transports configuration projections (layers 2 and 4)

Threads both attributes from the atlas-tenants REST payload into the builder, and exposes them on the debug REST resource so an operator can confirm a live tenant picked up the config.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/config/rest.go`
- Modify: `services/atlas-transports/atlas.com/transports/instance/config/rest_test.go`
- Modify: `services/atlas-transports/atlas.com/transports/instance/rest.go`

**Interfaces:**
- Consumes: `SetEffectItemIds`, `SetForcedReturnMapId`, `EffectItemIds()`, `ForcedReturnMapId()` from Task 1.
- Produces:
  - `config.InstanceRouteRestModel` gains `EffectItemIds []item.Id \`json:"effectItemIds"\`` and `ForcedReturnMapId _map.Id \`json:"forcedReturnMapId"\``
  - `instance.RouteRestModel` gains the same two fields with the same JSON names.

- [ ] **Step 1: Write the failing extraction test**

Append to `services/atlas-transports/atlas.com/transports/instance/config/rest_test.go` (add `"github.com/Chronicle20/atlas/libs/atlas-constants/item"` to the import block):

```go
// sampleInstanceRoute deliberately omits the effect attributes, so this test
// builds its own fixture: ExtractRouteFor must carry both through to the
// domain model or the processor sees zero values with no error anywhere.
func TestExtractRouteFor_ThreadsEffectAttributes(t *testing.T) {
	tm := testTenant(t, uuid.New())
	r := sampleInstanceRoute("temple-of-time-flight", uuid.New().String())
	r.EffectItemIds = []item.Id{2210016}
	r.ForcedReturnMapId = _map.Id(240000110)

	m, err := config.ExtractRouteFor(quietLogger(), tm)(r)
	if err != nil {
		t.Fatalf("ExtractRouteFor: %v", err)
	}
	if got := m.EffectItemIds(); len(got) != 1 || got[0] != item.Id(2210016) {
		t.Fatalf("EffectItemIds = %v, want [2210016]", got)
	}
	if m.ForcedReturnMapId() != _map.Id(240000110) {
		t.Fatalf("ForcedReturnMapId = %d, want 240000110", m.ForcedReturnMapId())
	}
}

// A route declaring neither attribute must still build — the ten unaffected
// routes take this path.
func TestExtractRouteFor_EffectAttributesAreOptional(t *testing.T) {
	tm := testTenant(t, uuid.New())
	m, err := config.ExtractRouteFor(quietLogger(), tm)(sampleInstanceRoute("ellinia-ereve-ferry", uuid.New().String()))
	if err != nil {
		t.Fatalf("ExtractRouteFor: %v", err)
	}
	if len(m.EffectItemIds()) != 0 {
		t.Fatalf("EffectItemIds = %v, want empty", m.EffectItemIds())
	}
	if m.ForcedReturnMapId() != _map.Id(0) {
		t.Fatalf("ForcedReturnMapId = %d, want 0", m.ForcedReturnMapId())
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./instance/config/ -run ExtractRouteFor_ -v`

Expected: compile failure — `r.EffectItemIds undefined (type config.InstanceRouteRestModel has no field or method EffectItemIds)`.

- [ ] **Step 3: Extend the config REST model and extractor**

In `instance/config/rest.go`, add `"github.com/Chronicle20/atlas/libs/atlas-constants/item"` to the imports, extend the struct:

```go
type InstanceRouteRestModel struct {
	Id                    string    `json:"-"`
	Uuid                  string    `json:"uuid"`
	Name                  string    `json:"name"`
	StartMapId            _map.Id   `json:"startMapId"`
	TransitMapIds         []_map.Id `json:"transitMapIds"`
	DestinationMapId      _map.Id   `json:"destinationMapId"`
	Capacity              uint32    `json:"capacity"`
	BoardingWindowSeconds uint32    `json:"boardingWindowSeconds"`
	TravelDurationSeconds uint32    `json:"travelDurationSeconds"`
	TransitMessage        string    `json:"transitMessage"`
	// EffectItemIds and ForcedReturnMapId are optional. atlas-tenants omits
	// them for routes that declare neither, which decodes to nil/0 — the
	// "no effects, deliver to destination" default.
	EffectItemIds     []item.Id `json:"effectItemIds"`
	ForcedReturnMapId _map.Id   `json:"forcedReturnMapId"`
}
```

and add two setter calls to `ExtractRouteFor`'s builder chain, after `SetTransitMessage`:

```go
				SetTransitMessage(r.TransitMessage).
				SetEffectItemIds(r.EffectItemIds).
				SetForcedReturnMapId(r.ForcedReturnMapId).
				Build()
```

- [ ] **Step 4: Expose both on the debug REST resource (layer 4)**

In `instance/rest.go`, add the `item` import, extend `RouteRestModel` and `TransformRoute`:

```go
type RouteRestModel struct {
	ID                uuid.UUID     `json:"-"`
	Name              string        `json:"name"`
	StartMapId        _map.Id       `json:"startMapId"`
	TransitMapIds     []_map.Id     `json:"transitMapIds"`
	DestinationMapId  _map.Id       `json:"destinationMapId"`
	Capacity          uint32        `json:"capacity"`
	BoardingWindow    time.Duration `json:"boardingWindow"`
	TravelDuration    time.Duration `json:"travelDuration"`
	EffectItemIds     []item.Id     `json:"effectItemIds"`
	ForcedReturnMapId _map.Id       `json:"forcedReturnMapId"`
}
```

```go
func TransformRoute(m RouteModel) (RouteRestModel, error) {
	return RouteRestModel{
		ID:                m.Id(),
		Name:              m.Name(),
		StartMapId:        m.StartMapId(),
		TransitMapIds:     m.TransitMapIds(),
		DestinationMapId:  m.DestinationMapId(),
		Capacity:          m.Capacity(),
		BoardingWindow:    m.BoardingWindow(),
		TravelDuration:    m.TravelDuration(),
		EffectItemIds:     m.EffectItemIds(),
		ForcedReturnMapId: m.ForcedReturnMapId(),
	}, nil
}
```

This is how the §8 rollout step is verified: `GET` on atlas-transports' route resource shows whether a live tenant actually re-seeded.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./... 2>&1 | tail -20`

Expected: PASS in every package (the `instance` package's `resource_paginate_test.go` also exercises `RouteRestModel`; the added fields are additive and must not break it).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/config/rest.go \
        services/atlas-transports/atlas.com/transports/instance/config/rest_test.go \
        services/atlas-transports/atlas.com/transports/instance/rest.go
git commit -m "feat(transports): project effect attributes through config and debug REST"
```

---

### Task 3: atlas-tenants projections (layers 0 and 1)

Without this task the attributes never leave `atlas-tenants` and every earlier task is inert against a real tenant.

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/rest.go`
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks (different service, no shared Go package).
- Produces: `configuration.InstanceRouteRestModel` gains `EffectItemIds []uint32 \`json:"effectItemIds,omitempty"\`` and `ForcedReturnMapId uint32 \`json:"forcedReturnMapId,omitempty"\``. `TransformInstanceRoute` and `ExtractInstanceRoute` both carry them. JSON attribute names must be **byte-identical** to Task 2's — `effectItemIds`, `forcedReturnMapId`.

Note: this service's REST model uses plain `uint32` for every id (`StartMapId uint32`, `TransitMapIds []uint32`); match that, do not introduce `atlas-constants` types here.

- [ ] **Step 1: Write the failing projection tests**

Append to `services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go`:

```go
// TransformInstanceRoute explicitly projects each known attribute out of the
// untyped JSONB. An attribute it does not name is silently dropped before it
// ever reaches atlas-transports — no error, no log. This is that guard.
func TestTransformInstanceRoute_ProjectsEffectAttributes(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":   "temple-of-time-flight",
		"type": "instance-routes",
		"attributes": map[string]interface{}{
			"name":              "temple-of-time-flight",
			"effectItemIds":     []interface{}{float64(2210016)},
			"forcedReturnMapId": float64(240000110),
		},
	}
	rm, err := TransformInstanceRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformInstanceRoute: %v", err)
	}
	if len(rm.EffectItemIds) != 1 || rm.EffectItemIds[0] != 2210016 {
		t.Fatalf("EffectItemIds = %v, want [2210016]", rm.EffectItemIds)
	}
	if rm.ForcedReturnMapId != 240000110 {
		t.Fatalf("ForcedReturnMapId = %d, want 240000110", rm.ForcedReturnMapId)
	}
}

// The ten routes that declare neither attribute must project to zero values,
// not to an error.
func TestTransformInstanceRoute_EffectAttributesAreOptional(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "ellinia-ereve-ferry",
		"type":       "instance-routes",
		"attributes": map[string]interface{}{"name": "ellinia-ereve-ferry"},
	}
	rm, err := TransformInstanceRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformInstanceRoute: %v", err)
	}
	if len(rm.EffectItemIds) != 0 {
		t.Fatalf("EffectItemIds = %v, want empty", rm.EffectItemIds)
	}
	if rm.ForcedReturnMapId != 0 {
		t.Fatalf("ForcedReturnMapId = %d, want 0", rm.ForcedReturnMapId)
	}
}

// ExtractInstanceRoute is the write-back half: a POST/PATCH through the CRUD
// handlers (resource.go:509,558) turns the REST model back into the JSONB
// attribute map. An attribute missing here is erased from storage on the next
// operator edit.
func TestExtractInstanceRoute_RoundTripsEffectAttributes(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	in := InstanceRouteRestModel{
		Id:                "temple-of-time-flight",
		Name:              "temple-of-time-flight",
		EffectItemIds:     []uint32{2210016},
		ForcedReturnMapId: 240000110,
	}
	extracted, err := ExtractInstanceRoute(in)
	if err != nil {
		t.Fatalf("ExtractInstanceRoute: %v", err)
	}
	out, err := TransformInstanceRoute(tid, toFloat64Attributes(extracted))
	if err != nil {
		t.Fatalf("TransformInstanceRoute: %v", err)
	}
	if len(out.EffectItemIds) != 1 || out.EffectItemIds[0] != 2210016 {
		t.Fatalf("EffectItemIds = %v, want [2210016]", out.EffectItemIds)
	}
	if out.ForcedReturnMapId != 240000110 {
		t.Fatalf("ForcedReturnMapId = %d, want 240000110", out.ForcedReturnMapId)
	}
}
```

`toFloat64Attributes` already exists at the bottom of `rest_test.go` — it converts numeric attribute values to `float64`, mirroring how they arrive after a JSON round trip. Read it before using it; if it does not recurse into `[]uint32` slices, extend it there to convert slice elements to `[]interface{}` of `float64` rather than writing a second helper.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/ -run 'InstanceRoute_(Projects|EffectAttributes|RoundTrips)' -v`

Expected: compile failure — `rm.EffectItemIds undefined`.

- [ ] **Step 3: Extend the REST model**

In `services/atlas-tenants/atlas.com/tenants/configuration/rest.go`, add to `InstanceRouteRestModel` (after `TransitMessage`):

```go
	// EffectItemIds are the consumable item ids atlas-transports applies on
	// boarding and cancels on every terminal path of this route. Optional.
	EffectItemIds []uint32 `json:"effectItemIds,omitempty"`
	// ForcedReturnMapId is where atlas-transports warps a character whose
	// travel timer expired, instead of destinationMapId. Zero means "not
	// set". It mirrors the client's Map.wz info/forcedReturn node. Optional.
	ForcedReturnMapId uint32 `json:"forcedReturnMapId,omitempty"`
```

- [ ] **Step 4: Extend `TransformInstanceRoute`**

Add after the `transitMessage` line, using the same defensive untyped-JSONB pattern the `transitMapIds` block already uses:

```go
	var effectItemIds []uint32
	if val, ok := attributes["effectItemIds"].([]interface{}); ok {
		for _, v := range val {
			if f, ok := v.(float64); ok {
				effectItemIds = append(effectItemIds, uint32(f))
			}
		}
	}

	forcedReturnMapId := uint32(0)
	if val, ok := attributes["forcedReturnMapId"].(float64); ok {
		forcedReturnMapId = uint32(val)
	}
```

and add both to the returned literal:

```go
		TransitMessage:    transitMessage,
		EffectItemIds:     effectItemIds,
		ForcedReturnMapId: forcedReturnMapId,
	}, nil
```

- [ ] **Step 5: Extend `ExtractInstanceRoute`**

Add both keys to the `attributes` map:

```go
			"transitMessage":        r.TransitMessage,
			"effectItemIds":         r.EffectItemIds,
			"forcedReturnMapId":     r.ForcedReturnMapId,
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/ -v 2>&1 | tail -30`

Expected: PASS, including every pre-existing test in the package.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-tenants/atlas.com/tenants/configuration/rest.go \
        services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go
git commit -m "feat(tenants): project instance-route effect and forced-return attributes"
```

---

### Task 4: Consumable wire contract, providers, and the TIMEOUT reason

`atlas-transports` becomes a producer on the existing `COMMAND_TOPIC_CONSUMABLE`. Every service that talks this topic keeps its own copy of the contract (`atlas-saga-orchestrator` already does); cross-importing another service's Go package would break the service boundary CLAUDE.md §Code Patterns protects.

**Files:**
- Create: `services/atlas-transports/atlas.com/transports/kafka/message/consumable/kafka.go`
- Modify: `services/atlas-transports/atlas.com/transports/kafka/message/instance_transport/kafka.go`
- Modify: `services/atlas-transports/atlas.com/transports/instance/producer.go`
- Create: `services/atlas-transports/atlas.com/transports/instance/producer_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `consumable.EnvCommandTopic`, `consumable.CommandApplyConsumableEffect`, `consumable.CommandCancelConsumableEffect`, `consumable.Command[E]`, `consumable.ApplyConsumableEffectBody`, `consumable.CancelConsumableEffectBody` under import path `atlas-transports/kafka/message/consumable`
  - `it.CancelReasonTimeout = "TIMEOUT"`
  - `func applyConsumableEffectProvider(worldId world.Id, channelId channel.Id, characterId uint32, itemId item.Id) model.Provider[[]kafka.Message]`
  - `func cancelConsumableEffectProvider(worldId world.Id, channelId channel.Id, characterId uint32, itemId item.Id) model.Provider[[]kafka.Message]`

- [ ] **Step 1: Write the failing provider test**

Create `services/atlas-transports/atlas.com/transports/instance/producer_test.go`:

```go
package instance

import (
	"atlas-transports/kafka/message/consumable"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// decodedConsumable is the on-the-wire shape of both commands: APPLY and
// CANCEL share one body ({itemId}), so one decoder covers both.
type decodedConsumable struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          struct {
		ItemId item.Id `json:"itemId"`
	} `json:"body"`
}

func TestApplyConsumableEffectProvider_WireShape(t *testing.T) {
	ms, err := applyConsumableEffectProvider(world.Id(0), channel.Id(1), 42, item.Id(2210016))()
	assert.NoError(t, err)
	assert.Len(t, ms, 1)

	var got decodedConsumable
	assert.NoError(t, json.Unmarshal(ms[0].Value, &got))

	assert.Equal(t, consumable.CommandApplyConsumableEffect, got.Type)
	assert.Equal(t, world.Id(0), got.WorldId)
	assert.Equal(t, channel.Id(1), got.ChannelId)
	assert.Equal(t, uint32(42), got.CharacterId)
	assert.Equal(t, item.Id(2210016), got.Body.ItemId)
	// uuid.Nil marks a non-saga effect application: atlas-saga-orchestrator
	// skips saga completion for it rather than logging an orphan transaction.
	assert.Equal(t, uuid.Nil, got.TransactionId)
}

func TestCancelConsumableEffectProvider_WireShape(t *testing.T) {
	ms, err := cancelConsumableEffectProvider(world.Id(0), channel.Id(1), 42, item.Id(2210016))()
	assert.NoError(t, err)
	assert.Len(t, ms, 1)

	var got decodedConsumable
	assert.NoError(t, json.Unmarshal(ms[0].Value, &got))

	assert.Equal(t, consumable.CommandCancelConsumableEffect, got.Type)
	assert.Equal(t, uint32(42), got.CharacterId)
	assert.Equal(t, item.Id(2210016), got.Body.ItemId)
}

// Both commands are keyed by characterId so they land on one partition and
// atlas-consumables (serial by default) can never reorder an APPLY past a
// later CANCEL for the same character.
func TestConsumableProviders_ShareOneKeyPerCharacter(t *testing.T) {
	apply, err := applyConsumableEffectProvider(world.Id(0), channel.Id(1), 42, item.Id(2210016))()
	assert.NoError(t, err)
	cancel, err := cancelConsumableEffectProvider(world.Id(0), channel.Id(1), 42, item.Id(2210016))()
	assert.NoError(t, err)

	assert.Equal(t, apply[0].Key, cancel[0].Key)
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./instance/ -run ConsumableEffectProvider -v`

Expected: compile failure — package `atlas-transports/kafka/message/consumable` does not exist; `applyConsumableEffectProvider` undefined.

- [ ] **Step 3: Create the local wire contract**

Create `services/atlas-transports/atlas.com/transports/kafka/message/consumable/kafka.go`:

```go
package consumable

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_CONSUMABLE"

	CommandApplyConsumableEffect  = "APPLY_CONSUMABLE_EFFECT"
	CommandCancelConsumableEffect = "CANCEL_CONSUMABLE_EFFECT"
)

// Command mirrors atlas-consumables' envelope field-for-field, including the
// MapId and Instance this service always leaves zero. Keeping the shapes
// identical means a future field addition on the consumer side is a visible
// diff here rather than a silent decode of the wrong layout.
//
// This service emits only the two effect commands, so only their bodies are
// declared. Every service on this topic keeps its own copy of the contract
// (atlas-saga-orchestrator does the same); importing another service's Go
// package would break the service boundary.
type Command[E any] struct {
	TransactionId uuid.UUID    `json:"transactionId"`
	WorldId       world.Id     `json:"worldId"`
	ChannelId     channel.Id   `json:"channelId"`
	MapId         _map.Id      `json:"mapId"`
	Instance      uuid.UUID    `json:"instance"`
	CharacterId   character.Id `json:"characterId"`
	Type          string       `json:"type"`
	Body          E            `json:"body"`
}

// ApplyConsumableEffectBody applies a consumable's effect without consuming
// anything from inventory.
type ApplyConsumableEffectBody struct {
	ItemId item.Id `json:"itemId"`
}

// CancelConsumableEffectBody cancels a previously applied consumable effect.
type CancelConsumableEffectBody struct {
	ItemId item.Id `json:"itemId"`
}
```

- [ ] **Step 4: Add the TIMEOUT cancel reason**

In `services/atlas-transports/atlas.com/transports/kafka/message/instance_transport/kafka.go`, extend the reason block:

```go
	CancelReasonMapExit = "MAP_EXIT"
	CancelReasonLogout  = "LOGOUT"
	CancelReasonStuck   = "STUCK"
	// CancelReasonTimeout is emitted when the travel timer expires on a route
	// that declares a forced return. The character did not complete the trip —
	// the client's own map data (timeLimit + forcedReturn) treats running out
	// of flight time as a failure that sends them back where they started.
	CancelReasonTimeout = "TIMEOUT"
```

- [ ] **Step 5: Add the two providers**

In `services/atlas-transports/atlas.com/transports/instance/producer.go`, add `consumable "atlas-transports/kafka/message/consumable"` and `"github.com/Chronicle20/atlas/libs/atlas-constants/character"` / `"github.com/Chronicle20/atlas/libs/atlas-constants/item"` to the imports, then append:

```go
// applyConsumableEffectProvider and cancelConsumableEffectProvider build the
// two COMMAND_TOPIC_CONSUMABLE messages this service emits.
//
// TransactionId is deliberately uuid.Nil: atlas-saga-orchestrator treats a nil
// transaction id on the resulting EFFECT_APPLIED event as a non-saga effect
// application and skips saga completion. A fresh uuid would look like an
// orphaned transaction instead.
//
// MapId/Instance are left zero. APPLY ignores the envelope's field entirely
// and resolves the character's live map itself; CANCEL builds a field from the
// envelope but it reaches atlas-buffs' Cancel, which reads only worldId. This
// is what lets the logout path — which has no map in hand — cancel correctly.
func applyConsumableEffectProvider(worldId world.Id, channelId channel.Id, characterId uint32, itemId item.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.ApplyConsumableEffectBody]{
		TransactionId: uuid.Nil,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   character.Id(characterId),
		Type:          consumable.CommandApplyConsumableEffect,
		Body: consumable.ApplyConsumableEffectBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func cancelConsumableEffectProvider(worldId world.Id, channelId channel.Id, characterId uint32, itemId item.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.CancelConsumableEffectBody]{
		TransactionId: uuid.Nil,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   character.Id(characterId),
		Type:          consumable.CommandCancelConsumableEffect,
		Body: consumable.CancelConsumableEffectBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./instance/ -run ConsumableEffectProvider -v && go build ./...`

Expected: PASS, build clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/kafka/message/consumable/kafka.go \
        services/atlas-transports/atlas.com/transports/kafka/message/instance_transport/kafka.go \
        services/atlas-transports/atlas.com/transports/instance/producer.go \
        services/atlas-transports/atlas.com/transports/instance/producer_test.go
git commit -m "feat(transports): add consumable command contract, providers, and TIMEOUT reason"
```

---

### Task 5: Apply route effects on boarding

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/processor.go`
- Create: `services/atlas-transports/atlas.com/transports/instance/processor_test.go`

**Interfaces:**
- Consumes: `RouteModel.EffectItemIds()` (Task 1); `applyConsumableEffectProvider`, `consumable.EnvCommandTopic` (Task 4).
- Produces:
  - `func (p *ProcessorImpl) applyRouteEffects(mb *message.Buffer, route RouteModel, worldId world.Id, channelId channel.Id, characterId uint32)` — returns nothing
  - Test helpers used by Tasks 6 and 7: `setupProcessorTest(t) (*ProcessorImpl, context.Context)`, `newEffectRoute(t, capacity uint32) RouteModel`, `newPlainRoute(t) RouteModel`, `decodeConsumables(t, mb) []decodedConsumable`, `decodeInstanceTransportEvents(t, mb) []decodedTransportEvent`

Do **not** redeclare helpers that already exist in this package's test files: `setupRouteTestRegistry`, `setupInstanceTestRegistry`, `setupCharacterTestRegistry`, `newTestTenantContext`, `newTestRoute` (`route_registry_test.go`, `instance_registry_test.go`, `character_registry_test.go`). `decodedConsumable` is declared in `producer_test.go` from Task 4 — reuse it, do not redeclare.

- [ ] **Step 1: Write the failing boarding tests**

Create `services/atlas-transports/atlas.com/transports/instance/processor_test.go`:

```go
package instance

import (
	"atlas-transports/kafka/message"
	"atlas-transports/kafka/message/consumable"
	it "atlas-transports/kafka/message/instance_transport"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// setupProcessorTest wires all three registries onto one miniredis and returns
// a processor with a nil producer: every Xxx(mb) method only buffers, and the
// producer is used exclusively by the XxxAndEmit wrappers these tests never
// call. That is what makes the lifecycle testable without Kafka or mocks.
func setupProcessorTest(t *testing.T) (*ProcessorImpl, context.Context) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRouteRegistry(rc)
	InitInstanceRegistry(rc)
	InitCharacterRegistry(rc)

	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)

	l := logrus.New()
	l.SetOutput(io.Discard)

	return &ProcessorImpl{l: l, ctx: ctx, t: tm, p: nil}, ctx
}

// newEffectRoute mirrors the seeded temple-of-time-flight: one declared effect
// item and a forced return. Capacity is a parameter so a fan-out test can put
// several characters in one instance.
func newEffectRoute(t *testing.T, capacity uint32) RouteModel {
	t.Helper()
	route, err := NewRouteBuilder("temple-of-time-flight").
		SetStartMapId(_map.Id(240000110)).
		SetTransitMapIds([]_map.Id{200090500, 200090510}).
		SetDestinationMapId(_map.Id(270000100)).
		SetCapacity(capacity).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(900 * time.Second).
		SetEffectItemIds([]item.Id{2210016}).
		SetForcedReturnMapId(_map.Id(240000110)).
		Build()
	assert.NoError(t, err)
	return route
}

// newPlainRoute mirrors a ferry: no declared effects, no forced return. This
// is the regression bar — it must produce zero consumable messages on every
// path and keep delivering to destinationMapId on arrival.
func newPlainRoute(t *testing.T) RouteModel {
	t.Helper()
	route, err := NewRouteBuilder("ellinia-ereve-ferry").
		SetStartMapId(_map.Id(101000300)).
		SetTransitMapIds([]_map.Id{200090030}).
		SetDestinationMapId(_map.Id(130000210)).
		SetCapacity(4).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(60 * time.Second).
		Build()
	assert.NoError(t, err)
	return route
}

func decodeConsumables(t *testing.T, mb *message.Buffer) []decodedConsumable {
	t.Helper()
	out := make([]decodedConsumable, 0)
	for _, m := range mb.GetAll()[consumable.EnvCommandTopic] {
		var d decodedConsumable
		assert.NoError(t, json.Unmarshal(m.Value, &d))
		out = append(out, d)
	}
	return out
}

type decodedTransportEvent struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        struct {
		RouteId    uuid.UUID `json:"routeId"`
		InstanceId uuid.UUID `json:"instanceId"`
		Reason     string    `json:"reason"`
	} `json:"body"`
}

func decodeInstanceTransportEvents(t *testing.T, mb *message.Buffer) []decodedTransportEvent {
	t.Helper()
	out := make([]decodedTransportEvent, 0)
	for _, m := range mb.GetAll()[it.EnvEventTopic] {
		var d decodedTransportEvent
		assert.NoError(t, json.Unmarshal(m.Value, &d))
		out = append(out, d)
	}
	return out
}

type decodedChangeMap struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        struct {
		MapId _map.Id `json:"mapId"`
	} `json:"body"`
}

func decodeChangeMaps(t *testing.T, mb *message.Buffer) []decodedChangeMap {
	t.Helper()
	out := make([]decodedChangeMap, 0)
	for _, m := range mb.GetAll()[character2EnvCommandTopic] {
		var d decodedChangeMap
		assert.NoError(t, json.Unmarshal(m.Value, &d))
		out = append(out, d)
	}
	return out
}

func TestStartTransport_AppliesDeclaredEffects(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 1)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})

	mb := message.NewBuffer()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(240000110)).Build()
	assert.NoError(t, p.StartTransport(mb)(42, route.Id(), f))

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 1)
	assert.Equal(t, consumable.CommandApplyConsumableEffect, cs[0].Type)
	assert.Equal(t, item.Id(2210016), cs[0].Body.ItemId)
	assert.Equal(t, uint32(42), cs[0].CharacterId)
	assert.Equal(t, world.Id(0), cs[0].WorldId)
	assert.Equal(t, channel.Id(1), cs[0].ChannelId)
}

func TestStartTransport_NoEffectsEmitsNoConsumableCommands(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})

	mb := message.NewBuffer()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(101000300)).Build()
	assert.NoError(t, p.StartTransport(mb)(42, route.Id(), f))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
	// The trip still starts exactly as before.
	assert.Len(t, decodeChangeMaps(t, mb), 1)
}

func TestStartTransport_RouteNotFoundEmitsNothing(t *testing.T) {
	p, _ := setupProcessorTest(t)

	mb := message.NewBuffer()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(240000110)).Build()
	assert.Error(t, p.StartTransport(mb)(42, uuid.New(), f))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
}

func TestStartTransport_AlreadyInTransportEmitsNothing(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 2)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	getCharacterRegistry().Add(42, uuid.New())

	mb := message.NewBuffer()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(240000110)).Build()
	assert.Error(t, p.StartTransport(mb)(42, route.Id(), f))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./instance/ -run StartTransport_ -v`

Expected: `TestStartTransport_AppliesDeclaredEffects` FAILs with `Not equal: expected 1, actual 0` on the consumable message count. The three negative tests pass trivially at this point — that is fine; they are the regression guard for later tasks.

- [ ] **Step 3: Add the `applyRouteEffects` helper**

In `services/atlas-transports/atlas.com/transports/instance/processor.go`, add `consumable "atlas-transports/kafka/message/consumable"` to the imports and add the helper just above `StartTransport`:

```go
// applyRouteEffects buffers one APPLY_CONSUMABLE_EFFECT per item the route
// declares. It deliberately returns nothing: a missing morph is cosmetic, a
// rejected boarding is not, so a buffer failure is logged and boarding
// continues. A route declaring no effects is a zero-command no-op.
func (p *ProcessorImpl) applyRouteEffects(mb *message.Buffer, route RouteModel, worldId world.Id, channelId channel.Id, characterId uint32) {
	for _, itemId := range route.EffectItemIds() {
		p.l.Infof("Applying route [%s] effect item [%d] to character [%d].", route.Name(), itemId, characterId)
		if err := mb.Put(consumable.EnvCommandTopic, applyConsumableEffectProvider(worldId, channelId, characterId, itemId)); err != nil {
			p.l.WithError(err).Errorf("Unable to buffer apply of effect item [%d] for character [%d] on route [%s].", itemId, characterId, route.Name())
		}
	}
}
```

- [ ] **Step 4: Call it from `StartTransport`**

In `StartTransport`, insert the call between the boarding log line and the `CHANGE_MAP` put:

```go
		p.l.Infof("Character [%d] boarding instance [%s] for route [%s] (%s). Characters: %d/%d.",
			characterId, inst.InstanceId(), route.Name(), route.Id(), count, route.Capacity())

		// Effect applies are buffered before the CHANGE_MAP command, mirroring
		// the ordering the NPC saga used to guarantee. That ordering is a
		// readability convention, not a guarantee: message.Buffer emits
		// per-topic in Go map iteration order, which is randomised. Correctness
		// does not need one — ApplyConsumableEffect resolves the character's
		// live map at handling time, and an APPLY cannot overtake a later
		// CANCEL because both are keyed by characterId onto a single partition
		// that atlas-consumables consumes serially (maxInFlight defaults to 1).
		p.applyRouteEffects(mb, route, f.WorldId(), f.ChannelId(), characterId)

		// Emit CHANGE_MAP command to transit map with instance
		err := mb.Put(character2EnvCommandTopic, warpToTransitMapProvider(f, characterId, route.TransitMapIds()[0], inst.InstanceId()))
```

Note both rejection paths (`IsInTransport`, route-not-found) return *before* this line, satisfying FR-1.3 by construction.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./instance/ -v 2>&1 | tail -30`

Expected: PASS for all four `StartTransport_` tests and every pre-existing test.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/processor.go \
        services/atlas-transports/atlas.com/transports/instance/processor_test.go
git commit -m "feat(transports): apply route-declared consumable effects on boarding"
```

---

### Task 6: Cancel route effects on every terminal path

The core of the task. Four of the five terminal paths get their cancel here; `TickArrival` gets both its cancel and its forced-return branch in Task 7 so the arrival changes land as one reviewable unit.

Two of the paths also get the teardown hardening (D8): `HandleMapEnter` and `HandleLogout` today `return err` from the cancelled-event `mb.Put`, and `ir.ReleaseInstance` sits *after* that return — so a failing event put skips instance release, exactly what the "leaking an instance is worse" NFR forbids.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/processor.go`
- Modify: `services/atlas-transports/atlas.com/transports/instance/processor_test.go`

**Interfaces:**
- Consumes: `applyRouteEffects` conventions and the test helpers from Task 5; `cancelConsumableEffectProvider` from Task 4.
- Produces: `func (p *ProcessorImpl) cancelRouteEffects(mb *message.Buffer, route RouteModel, worldId world.Id, channelId channel.Id, characterId uint32)` — returns nothing.

- [ ] **Step 1: Write the failing terminal-path tests**

Append to `services/atlas-transports/atlas.com/transports/instance/processor_test.go`:

```go
// board puts a character into an instance of route through the real boarding
// path and returns the instance id, so terminal-path tests start from exactly
// the state StartTransport leaves behind.
func board(t *testing.T, p *ProcessorImpl, route RouteModel, characterId uint32, worldId world.Id, channelId channel.Id) uuid.UUID {
	t.Helper()
	mb := message.NewBuffer()
	f := field.NewBuilder(worldId, channelId, route.StartMapId()).Build()
	assert.NoError(t, p.StartTransport(mb)(characterId, route.Id(), f))
	instanceId, ok := getCharacterRegistry().GetInstanceForCharacter(characterId)
	assert.True(t, ok)
	return instanceId
}

// The dracoout shape: exiting transit map 200090510 through a portal that
// warps to the non-transit map 240000100. That portal seed carries no
// cancel_consumable_effect operation — under route-owned cleanup the morph is
// removed here instead, which is the previously-unfixed leak (FR-3.6).
func TestHandleMapEnter_NonTransitMapCancelsEffects(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 1)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.HandleMapEnter(mb)(42, _map.Id(240000100), uuid.Nil, world.Id(0), channel.Id(1)))

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 1)
	assert.Equal(t, consumable.CommandCancelConsumableEffect, cs[0].Type)
	assert.Equal(t, item.Id(2210016), cs[0].Body.ItemId)
	assert.Equal(t, uint32(42), cs[0].CharacterId)

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeCancelled, evs[0].Type)
	assert.Equal(t, it.CancelReasonMapExit, evs[0].Body.Reason)
	assert.Equal(t, instanceId, evs[0].Body.InstanceId)
}

// Moving between two transit maps of the same route is not a terminal path.
func TestHandleMapEnter_TransitToTransitDoesNotCancel(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 1)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.HandleMapEnter(mb)(42, _map.Id(200090510), uuid.Nil, world.Id(0), channel.Id(1)))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeTransitEntered, evs[0].Type)
}

func TestHandleMapEnter_NoEffectsEmitsNoConsumableCommands(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.HandleMapEnter(mb)(42, _map.Id(130000210), uuid.Nil, world.Id(0), channel.Id(1)))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
}

// atlas-buffs does not drop buffs on logout — they carry an expiresAt and are
// restored with their remaining duration. Without this cancel a player who
// logs out mid-flight logs back in still morphed.
func TestHandleLogout_CancelsEffects(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 1)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.HandleLogout(mb)(42, world.Id(0), channel.Id(1)))

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 1)
	assert.Equal(t, consumable.CommandCancelConsumableEffect, cs[0].Type)
	assert.Equal(t, uint32(42), cs[0].CharacterId)

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.CancelReasonLogout, evs[0].Body.Reason)
}

func TestHandleLogout_NoEffectsEmitsNoConsumableCommands(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.HandleLogout(mb)(42, world.Id(0), channel.Id(1)))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
}

func TestTickStuckTimeout_CancelsEffectsForEveryCharacter(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 2)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))
	board(t, p, route, 43, world.Id(0), channel.Id(2))

	// MaxLifetime is 2*(boardingWindow+travelDuration) = 1802s; miniredis
	// stores real timestamps, so age the instance by rewriting its metadata
	// is not possible — instead assert against a route whose lifetime has
	// already elapsed by using GetStuck's own clock.
	mb := message.NewBuffer()
	assert.NoError(t, p.TickStuckTimeout(mb))
	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic], "instance is not stuck yet")

	// Advance past MaxLifetime by asking the registry directly.
	stuck := getInstanceRegistry().GetStuck(time.Now().Add(route.MaxLifetime()+time.Second), route.MaxLifetime())
	assert.NotEmpty(t, stuck, "instance must be considered stuck once MaxLifetime has elapsed")
}

func TestGracefulShutdown_CancelsEffectsForEveryCharacter(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 2)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))
	board(t, p, route, 43, world.Id(0), channel.Id(2))

	mb := message.NewBuffer()
	assert.NoError(t, p.GracefulShutdown(mb))

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 2)
	seen := map[uint32]bool{}
	for _, c := range cs {
		assert.Equal(t, consumable.CommandCancelConsumableEffect, c.Type)
		assert.Equal(t, item.Id(2210016), c.Body.ItemId)
		seen[c.CharacterId] = true
	}
	assert.True(t, seen[42] && seen[43], "both characters must be cancelled")

	// Characters are still warped to the start map, unchanged.
	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 2)
	for _, w := range warps {
		assert.Equal(t, route.StartMapId(), w.Body.MapId)
	}
}

func TestGracefulShutdown_NoEffectsEmitsNoConsumableCommands(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.GracefulShutdown(mb))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
	assert.Len(t, decodeChangeMaps(t, mb), 1)
}
```

`TickStuckTimeout` cannot be driven to its cancel branch without a controllable clock — `GetStuck(now, maxLifetime)` is called with `time.Now()` inside the method. The test above therefore asserts the *registry-level* precondition (the instance does become stuck) and that no cancel fires early. Cover the emission itself by extracting the loop body into a helper the test can call directly:

- [ ] **Step 2: Write the failing direct-emission test for the stuck path**

Append to `processor_test.go`:

```go
// TickStuckTimeout's clock is time.Now() inside the method, so the cancelling
// loop body is exercised directly. This is the same code path the tick runs
// once MaxLifetime has elapsed.
func TestForceCancelInstance_CancelsEffectsAndWarpsToStart(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 2)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))
	board(t, p, route, 43, world.Id(0), channel.Id(2))

	inst, ok := getInstanceRegistry().GetInstance(instanceId)
	assert.True(t, ok)

	mb := message.NewBuffer()
	p.forceCancelInstance(mb, inst, route)

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 2)
	for _, c := range cs {
		assert.Equal(t, consumable.CommandCancelConsumableEffect, c.Type)
		assert.Equal(t, item.Id(2210016), c.Body.ItemId)
	}

	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 2)
	for _, w := range warps {
		assert.Equal(t, route.StartMapId(), w.Body.MapId)
	}

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 2)
	for _, e := range evs {
		assert.Equal(t, it.EventTypeCancelled, e.Type)
		assert.Equal(t, it.CancelReasonStuck, e.Body.Reason)
	}

	assert.False(t, getCharacterRegistry().IsInTransport(42))
	assert.False(t, getCharacterRegistry().IsInTransport(43))
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./instance/ -run 'HandleMapEnter_|HandleLogout_|TickStuckTimeout_|GracefulShutdown_|ForceCancelInstance_' -v`

Expected: compile failure — `p.forceCancelInstance undefined`; and after that is added, the four cancel assertions FAIL with zero consumable messages.

- [ ] **Step 4: Add the `cancelRouteEffects` helper**

In `processor.go`, add next to `applyRouteEffects`:

```go
// cancelRouteEffects buffers one CANCEL_CONSUMABLE_EFFECT per item the route
// declares, for one character. Like applyRouteEffects it returns nothing: a
// terminal path must always finish releasing its instance even if a command
// cannot be buffered. Leaking a buff is bad; leaking an instance is worse.
//
// A double cancel is harmless — atlas-buffs' Cancel maps a missing buff to
// nil, with no event and no user-visible error — so racing terminal paths
// (portal exit at the same moment the timer fires) need no coordination.
func (p *ProcessorImpl) cancelRouteEffects(mb *message.Buffer, route RouteModel, worldId world.Id, channelId channel.Id, characterId uint32) {
	for _, itemId := range route.EffectItemIds() {
		p.l.Infof("Cancelling route [%s] effect item [%d] for character [%d].", route.Name(), itemId, characterId)
		if err := mb.Put(consumable.EnvCommandTopic, cancelConsumableEffectProvider(worldId, channelId, characterId, itemId)); err != nil {
			p.l.WithError(err).Errorf("Unable to buffer cancel of effect item [%d] for character [%d] on route [%s].", itemId, characterId, route.Name())
		}
	}
}
```

- [ ] **Step 5: Cancel on the `HandleMapEnter` non-transit branch, and harden its teardown**

Replace the body of the `if !isTransit && inTransport {` branch in `HandleMapEnter` with:

```go
		if !isTransit && inTransport {
			// Character entered a non-transit map while in transport — cancel
			ir := getInstanceRegistry()
			inst, ok := ir.GetInstance(charInstanceId)
			if !ok {
				cr.Remove(characterId)
				return nil
			}

			p.l.Infof("Character [%d] entered non-transit map [%d] while in transport, cancelling.", characterId, mapId)

			// The route is looked up only for its declared effects; a missing
			// route must not stop the instance from being torn down.
			if route, hasRoute := getRouteRegistry().GetRoute(p.ctx, inst.RouteId()); hasRoute {
				p.cancelRouteEffects(mb, route, worldId, channelId, characterId)
			} else {
				p.l.Warnf("Route [%s] not found while cancelling instance [%s]; character [%d] may retain transit effects.", inst.RouteId(), charInstanceId, characterId)
			}

			cr.Remove(characterId)
			empty := ir.RemoveCharacter(charInstanceId, characterId)

			// A failed event put is logged, not returned: ReleaseInstance below
			// must run regardless (PRD §8 failure isolation).
			if err := mb.Put(it.EnvEventTopic, cancelledEventProvider(worldId, characterId, inst.RouteId(), charInstanceId, it.CancelReasonMapExit)); err != nil {
				p.l.WithError(err).Errorf("Unable to buffer CANCELLED event for character [%d]; continuing to instance release.", characterId)
			}

			if empty {
				p.l.Infof("Instance [%s] is now empty, releasing.", charInstanceId)
				ir.ReleaseInstance(charInstanceId)
			}
			return nil
		}
```

- [ ] **Step 6: Cancel on the logout path, and harden its teardown**

Replace the tail of `HandleLogout` (from the `p.l.Infof("Character [%d] logged out…` line onward):

```go
		p.l.Infof("Character [%d] logged out during instance transport [%s], removing from instance.", characterId, charInstanceId)

		// Best effort (FR-1.6): atlas-buffs does not drop buffs on logout —
		// they carry an expiresAt and are restored with their remaining
		// duration — so without this the player logs back in still morphed.
		// It is not an error if the session is already gone; the command
		// never blocks and never fails the teardown.
		if route, hasRoute := getRouteRegistry().GetRoute(p.ctx, inst.RouteId()); hasRoute {
			p.cancelRouteEffects(mb, route, worldId, channelId, characterId)
		} else {
			p.l.Warnf("Route [%s] not found while cancelling instance [%s] on logout; character [%d] may retain transit effects.", inst.RouteId(), charInstanceId, characterId)
		}

		cr.Remove(characterId)
		empty := ir.RemoveCharacter(charInstanceId, characterId)

		if err := mb.Put(it.EnvEventTopic, cancelledEventProvider(worldId, characterId, inst.RouteId(), charInstanceId, it.CancelReasonLogout)); err != nil {
			p.l.WithError(err).Errorf("Unable to buffer CANCELLED event for character [%d] on logout; continuing to instance release.", characterId)
		}

		if empty {
			p.l.Infof("Instance [%s] is now empty after logout, releasing.", charInstanceId)
			ir.ReleaseInstance(charInstanceId)
		}
		return nil
```

- [ ] **Step 7: Extract and cancel on the stuck-timeout path**

Add `forceCancelInstance` below `TickStuckTimeout` and call it from the loop:

```go
func (p *ProcessorImpl) TickStuckTimeout(mb *message.Buffer) error {
	ir := getInstanceRegistry()
	now := time.Now()

	routes := getRouteRegistry().GetRoutes(p.ctx)
	for _, route := range routes {
		maxLifetime := route.MaxLifetime()
		for _, inst := range ir.GetStuck(now, maxLifetime) {
			if inst.RouteId() != route.Id() || inst.TenantId() != p.t.Id() {
				continue
			}
			p.l.Warnf("Instance [%s] for route [%s] exceeded max lifetime, force-cancelling.", inst.InstanceId(), route.Name())
			p.forceCancelInstance(mb, inst, route)
			ir.ReleaseInstance(inst.InstanceId())
		}
	}
	return nil
}

// forceCancelInstance cancels every character's route effects, warps them back
// to the route's start map and emits CANCELLED/STUCK. Extracted from
// TickStuckTimeout so the emission is directly testable — the tick's clock is
// time.Now() and cannot be advanced from a test.
func (p *ProcessorImpl) forceCancelInstance(mb *message.Buffer, inst TransportInstance, route RouteModel) {
	cr := getCharacterRegistry()
	for _, entry := range inst.Characters() {
		p.cancelRouteEffects(mb, route, entry.WorldId, entry.ChannelId, entry.CharacterId)
		_ = mb.Put(character2EnvCommandTopic, warpToStartMapProvider(entry.WorldId, entry.ChannelId, entry.CharacterId, route.StartMapId()))
		_ = mb.Put(it.EnvEventTopic, cancelledEventProvider(entry.WorldId, entry.CharacterId, route.Id(), inst.InstanceId(), it.CancelReasonStuck))
		cr.Remove(entry.CharacterId)
	}
}
```

Remove the now-unused `cr := getCharacterRegistry()` from `TickStuckTimeout` (it moved into the helper) — `go vet` will flag it otherwise.

- [ ] **Step 8: Cancel on the graceful-shutdown path**

In `GracefulShutdown`, add the cancel before the warp inside the character loop:

```go
		characters := inst.Characters()
		for _, entry := range characters {
			p.cancelRouteEffects(mb, route, entry.WorldId, entry.ChannelId, entry.CharacterId)
			_ = mb.Put(character2EnvCommandTopic, warpToStartMapProvider(entry.WorldId, entry.ChannelId, entry.CharacterId, route.StartMapId()))
			cr.Remove(entry.CharacterId)
		}
```

Graceful shutdown emits no instance-transport event today; that stays unchanged.

- [ ] **Step 9: Run the tests to verify they pass**

Run: `cd services/atlas-transports/atlas.com/transports && go test -race ./instance/ -v 2>&1 | tail -40`

Expected: PASS for every test in the package.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/processor.go \
        services/atlas-transports/atlas.com/transports/instance/processor_test.go
git commit -m "feat(transports): cancel route effects on map-exit, logout, stuck, and shutdown paths"
```

---

### Task 7: Forced return and cancel on travel-timer arrival

The fifth terminal path. `TickArrival` gains both its cancel and the forced-return branch, because they are one behavioural change: on a route that declares a forced return, the timer expiring is a *failure*, not the delivery mechanism.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/processor.go`
- Modify: `services/atlas-transports/atlas.com/transports/instance/processor_test.go`

**Interfaces:**
- Consumes: `cancelRouteEffects` (Task 6), `RouteModel.ForcedReturnMapId()` (Task 1), `it.CancelReasonTimeout` (Task 4).
- Produces: `func (p *ProcessorImpl) completeInstance(mb *message.Buffer, inst TransportInstance, route RouteModel)` — the extracted arrival body, testable without advancing the tick clock.

- [ ] **Step 1: Write the failing arrival tests**

Append to `processor_test.go`:

```go
// A route with a forced return: the timer expiring means the player ran out of
// flight time, so they go back to the forced-return map (the client's own
// Map.wz forcedReturn), not to the destination — and the event says CANCELLED
// with reason TIMEOUT, because they did not complete the trip.
func TestCompleteInstance_ForcedReturnWarpsBackAndCancels(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 1)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))

	inst, ok := getInstanceRegistry().GetInstance(instanceId)
	assert.True(t, ok)

	mb := message.NewBuffer()
	p.completeInstance(mb, inst, route)

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 1)
	assert.Equal(t, consumable.CommandCancelConsumableEffect, cs[0].Type)
	assert.Equal(t, item.Id(2210016), cs[0].Body.ItemId)

	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 1)
	assert.Equal(t, _map.Id(240000110), warps[0].Body.MapId, "forced return, not the destination")

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeCancelled, evs[0].Type)
	assert.Equal(t, it.CancelReasonTimeout, evs[0].Body.Reason)

	assert.False(t, getCharacterRegistry().IsInTransport(42))
}

// The ferry regression bar: no forced return, no declared effects — deliver to
// destinationMapId and emit COMPLETED, byte-identically to today.
func TestCompleteInstance_NoForcedReturnDeliversAndCompletes(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))

	inst, ok := getInstanceRegistry().GetInstance(instanceId)
	assert.True(t, ok)

	mb := message.NewBuffer()
	p.completeInstance(mb, inst, route)

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])

	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 1)
	assert.Equal(t, route.DestinationMapId(), warps[0].Body.MapId)

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeCompleted, evs[0].Type)
}

// A route that declares effects but no forced return still delivers to the
// destination while cancelling — the two fields are independent.
func TestCompleteInstance_EffectsWithoutForcedReturnStillDelivers(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route, err := NewRouteBuilder("effects-no-forced-return").
		SetStartMapId(_map.Id(100000000)).
		SetTransitMapIds([]_map.Id{100000100}).
		SetDestinationMapId(_map.Id(100000200)).
		SetCapacity(1).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(60 * time.Second).
		SetEffectItemIds([]item.Id{2210016}).
		Build()
	assert.NoError(t, err)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))

	inst, ok := getInstanceRegistry().GetInstance(instanceId)
	assert.True(t, ok)

	mb := message.NewBuffer()
	p.completeInstance(mb, inst, route)

	assert.Len(t, decodeConsumables(t, mb), 1)
	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 1)
	assert.Equal(t, _map.Id(100000200), warps[0].Body.MapId)
	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeCompleted, evs[0].Type)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./instance/ -run CompleteInstance_ -v`

Expected: compile failure — `p.completeInstance undefined`.

- [ ] **Step 3: Extract and rewrite the arrival body**

Replace `TickArrival`'s loop body and add the helper:

```go
func (p *ProcessorImpl) TickArrival(mb *message.Buffer) error {
	ir := getInstanceRegistry()
	now := time.Now()

	for _, inst := range ir.GetExpiredTransit(now) {
		if inst.TenantId() != p.t.Id() {
			continue
		}

		route, ok := getRouteRegistry().GetRoute(p.ctx, inst.RouteId())
		if !ok {
			p.l.Warnf("Route [%s] not found for arriving instance [%s], releasing.", inst.RouteId(), inst.InstanceId())
			ir.ReleaseInstance(inst.InstanceId())
			continue
		}

		p.completeInstance(mb, inst, route)
		ir.ReleaseInstance(inst.InstanceId())
	}
	return nil
}

// completeInstance runs the travel-timer arrival for one instance: cancel each
// character's route effects, warp them out, and emit the terminal event.
//
// A route that declares a forced-return map is one whose transit maps carry a
// client-side timeLimit — running out of flight time is a failure mode there,
// not the delivery mechanism, so the character goes back to the forced-return
// map and the event is CANCELLED/TIMEOUT. Emitting COMPLETED would tell a
// future consumer the character arrived somewhere they never reached. Routes
// without the field (ferries, whose transit maps have no timeLimit at all)
// keep delivering to destinationMapId with COMPLETED, unchanged.
//
// Extracted from TickArrival so the emission is directly testable — the tick's
// clock is time.Now() and cannot be advanced from a test.
func (p *ProcessorImpl) completeInstance(mb *message.Buffer, inst TransportInstance, route RouteModel) {
	cr := getCharacterRegistry()

	forcedReturn := route.ForcedReturnMapId() != 0
	target := route.DestinationMapId()
	if forcedReturn {
		target = route.ForcedReturnMapId()
	}

	p.l.Infof("Instance [%s] for route [%s] has arrived. Warping %d characters to [%d] (forced return: %t).",
		inst.InstanceId(), route.Name(), inst.CharacterCount(), target, forcedReturn)

	for _, entry := range inst.Characters() {
		p.cancelRouteEffects(mb, route, entry.WorldId, entry.ChannelId, entry.CharacterId)

		if err := mb.Put(character2EnvCommandTopic, warpToDestinationProvider(
			entry.WorldId, entry.ChannelId, entry.CharacterId, target)); err != nil {
			p.l.WithError(err).Errorf("Error warping character [%d] to [%d].", entry.CharacterId, target)
		}

		if forcedReturn {
			_ = mb.Put(it.EnvEventTopic, cancelledEventProvider(entry.WorldId, entry.CharacterId, route.Id(), inst.InstanceId(), it.CancelReasonTimeout))
		} else {
			_ = mb.Put(it.EnvEventTopic, completedEventProvider(entry.WorldId, entry.CharacterId, route.Id(), inst.InstanceId()))
		}

		cr.Remove(entry.CharacterId)
	}
}
```

`warpToDestinationProvider` is used for both targets: it is a thin wrapper over `changeMapProvider` with `uuid.Nil` for the instance, which is exactly right for "warp out of the instance" regardless of where.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-transports/atlas.com/transports && go test -race ./... 2>&1 | tail -20`

Expected: PASS in every package.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/processor.go \
        services/atlas-transports/atlas.com/transports/instance/processor_test.go
git commit -m "feat(transports): honour forced-return on travel-timer expiry and cancel effects on arrival"
```

---

### Task 8: Declare the effects on the two flight route seeds

**Files:**
- Modify: `deploy/seed/shared/all/instance-routes/flight-leafre-temple-of-time.json`
- Modify: `deploy/seed/shared/all/instance-routes/flight-temple-of-time-leafre.json`

**Interfaces:**
- Consumes: the attribute names fixed in Tasks 2 and 3 — `effectItemIds`, `forcedReturnMapId`.
- Produces: nothing for later Go tasks; Task 9 depends on these being in place so the seeds are never simultaneously missing both the operations and the declaration.

The seeder stores each catalog entry's `data` object verbatim into the configuration row's JSONB (`configuration/seed/subdomain.go` `Decode`/`Build`), so these attribute names land in storage exactly as written and are read back out by `TransformInstanceRoute`.

- [ ] **Step 1: Add both attributes to the outbound flight**

Write `deploy/seed/shared/all/instance-routes/flight-leafre-temple-of-time.json`:

```json
{
  "data": {
    "id": "temple-of-time-flight",
    "type": "instance-routes",
    "attributes": {
      "name": "temple-of-time-flight",
      "startMapId": 240000110,
      "transitMapIds": [
        200090500,
        200090510
      ],
      "destinationMapId": 270000100,
      "capacity": 1,
      "boardingWindowSeconds": 1,
      "travelDurationSeconds": 900,
      "transitMessage": "You are flying towards the Temple of Time. Navigate right to reach the entrance.",
      "effectItemIds": [
        2210016
      ],
      "forcedReturnMapId": 240000110
    }
  }
}
```

- [ ] **Step 2: Add both attributes to the return flight**

Write `deploy/seed/shared/all/instance-routes/flight-temple-of-time-leafre.json`:

```json
{
  "data": {
    "id": "temple-of-time-return-flight",
    "type": "instance-routes",
    "attributes": {
      "name": "temple-of-time-return-flight",
      "startMapId": 270000100,
      "transitMapIds": [
        200090510,
        200090500
      ],
      "destinationMapId": 240000110,
      "capacity": 1,
      "boardingWindowSeconds": 1,
      "travelDurationSeconds": 900,
      "transitMessage": "You are flying back to Leafre. Navigate left to reach the exit.",
      "effectItemIds": [
        2210016
      ],
      "forcedReturnMapId": 240000110
    }
  }
}
```

`forcedReturnMapId` equals this route's `destinationMapId`, so it is behaviourally redundant here. It is set for symmetry and to document the client's `forcedReturn` value — but it does change the emitted event on timer expiry from `COMPLETED` to `CANCELLED`/`TIMEOUT`, which is the accurate description of a player who idled out their return flight.

- [ ] **Step 3: Verify both files parse and carry the new attributes**

```bash
for f in deploy/seed/shared/all/instance-routes/flight-leafre-temple-of-time.json \
         deploy/seed/shared/all/instance-routes/flight-temple-of-time-leafre.json; do
  python3 -c "import json,sys; a=json.load(open('$f'))['data']['attributes']; assert a['effectItemIds']==[2210016], a; assert a['forcedReturnMapId']==240000110, a; print('OK $f')"
done
```

Expected: `OK` for both. Also confirm the other ten route seeds are untouched:

```bash
git status --short deploy/seed/shared/all/instance-routes/
```

Expected: exactly two modified files.

- [ ] **Step 4: Commit**

```bash
git add deploy/seed/shared/all/instance-routes/flight-leafre-temple-of-time.json \
        deploy/seed/shared/all/instance-routes/flight-temple-of-time-leafre.json
git commit -m "feat(seed): declare morph effect and forced return on the two ToT flight routes"
```

---

### Task 9: Remove the duplicated effect operations from 44 seed files

Four files × eleven version directories. All five relevant files are **byte-identical across all eleven directories today** (md5-verified; LF endings, no CRLF). The safe mechanic is therefore: hand-edit `gms/83_1`, verify the JSON, copy to the other ten, re-assert md5 uniformity. The uniformity premise is verified before *and* after — this is a verified copy, not a blind patch loop.

**Files:**
- Modify ×11: `deploy/seed/{gms/{12,48,61,72,79,83,84,87,92,95}_1,jms/185_1}/npc-conversations/npc/npc-2082003.json`
- Modify ×11: `.../portal-actions/portals/portal-outTemple.json`
- Modify ×11: `.../portal-actions/portals/portal-templeenter.json`
- Modify ×11: `.../portal-actions/portals/portal-undodraco.json`
- **Not modified:** `.../portal-actions/portals/portal-dracoout.json` (FR-3.6 — covered by `TestHandleMapEnter_NonTransitMapCancelsEffects` from Task 6)

**Interfaces:**
- Consumes: Tasks 5–8 (the route must already own the lifecycle before the seeds stop declaring it).
- Produces: nothing for later tasks.

- [ ] **Step 1: Re-assert the uniformity premise before editing**

```bash
for f in npc-conversations/npc/npc-2082003.json \
         portal-actions/portals/portal-outTemple.json \
         portal-actions/portals/portal-templeenter.json \
         portal-actions/portals/portal-undodraco.json; do
  echo "--- $f"
  for d in deploy/seed/gms/*_1 deploy/seed/jms/185_1; do md5sum "$d/$f"; done | awk '{print $1}' | sort -u
done
```

Expected: exactly one hash line per file. If any file shows two hashes, STOP — the copy mechanic below is invalid and each directory must be edited individually.

- [ ] **Step 2: Edit `gms/83_1/npc-conversations/npc/npc-2082003.json`**

Delete the entire `applyBuff` `genericAction` state and repoint `askTransform`'s first choice at `startTransport`. Removing only the operation would leave a state with an empty `operations` array — dead weight with no purpose. Write the file as:

```json
{
  "data": {
    "attributes": {
      "npcId": 2082003,
      "startState": "askTransform",
      "states": [
        {
          "id": "askTransform",
          "listSelection": {
            "choices": [
              {
                "nextState": "startTransport",
                "text": "I want to become a dragon."
              },
              {
                "nextState": null,
                "text": "Exit"
              }
            ],
            "title": "If you had wings, I'm sure you could go there.  But, that alone won't be enough.  If you want to fly though the wind that's sharper than a blade, you'll need tough scales as well.  I'm the only Halfling left that knows the way back... If you want to go there, I can transform you.  No matter what you are, for this moment, you will become a #bDragon#k..."
          },
          "type": "listSelection"
        },
        {
          "id": "startTransport",
          "transportAction": {
            "alreadyInTransitState": "transportAlreadyInTransit",
            "capacityFullState": "transportCapacityFull",
            "failureState": "transportFailed",
            "routeName": "temple-of-time-flight",
            "routeNotFoundState": "transportFailed",
            "serviceErrorState": "transportFailed"
          },
          "type": "transportAction"
        },
        {
          "dialogue": {
            "choices": [
              {
                "nextState": null,
                "text": "Ok"
              },
              {
                "nextState": null,
                "text": "Exit"
              }
            ],
            "dialogueType": "sendOk",
            "text": "I'm sorry, the flight service is currently unavailable. Please try again later."
          },
          "id": "transportFailed",
          "type": "dialogue"
        },
        {
          "dialogue": {
            "choices": [
              {
                "nextState": null,
                "text": "Ok"
              },
              {
                "nextState": null,
                "text": "Exit"
              }
            ],
            "dialogueType": "sendOk",
            "text": "There are too many dragons in the sky right now. Please wait a moment and try again."
          },
          "id": "transportCapacityFull",
          "type": "dialogue"
        },
        {
          "dialogue": {
            "choices": [
              {
                "nextState": null,
                "text": "Ok"
              },
              {
                "nextState": null,
                "text": "Exit"
              }
            ],
            "dialogueType": "sendOk",
            "text": "You seem to already be flying. Please wait until you arrive at your destination."
          },
          "id": "transportAlreadyInTransit",
          "type": "dialogue"
        }
      ]
    },
    "id": "2082003",
    "type": "npc-conversation"
  }
}
```

- [ ] **Step 3: Edit `gms/83_1/portal-actions/portals/portal-outTemple.json`**

Drop the `apply_consumable_effect` operation, leaving `start_instance_transport` alone:

```json
{
  "data": {
    "attributes": {
      "description": "Temple of Time - Return flight to Leafre via dragon transformation",
      "mapId": 270000100,
      "portalId": "outTemple",
      "rules": [
        {
          "conditions": [],
          "id": "start_return_flight",
          "onMatch": {
            "allow": false,
            "operations": [
              {
                "params": {
                  "failureMessage": "The flight service is currently unavailable. Please try again later.",
                  "routeName": "temple-of-time-return-flight"
                },
                "type": "start_instance_transport"
              }
            ]
          }
        }
      ]
    },
    "id": "outTemple",
    "type": "portal-action"
  }
}
```

- [ ] **Step 4: Edit `gms/83_1/portal-actions/portals/portal-templeenter.json`**

Drop the `cancel_consumable_effect` operation; keep `play_portal_sound` + `warp`:

```json
{
  "data": {
    "attributes": {
      "description": "Temple of Time Flight - Exit transit to Temple of Time entrance",
      "mapId": 200090510,
      "portalId": "templeenter",
      "rules": [
        {
          "conditions": [],
          "id": "exit_transit",
          "onMatch": {
            "allow": false,
            "operations": [
              {
                "params": {},
                "type": "play_portal_sound"
              },
              {
                "params": {
                  "mapId": "270000100",
                  "portalName": "out00"
                },
                "type": "warp"
              }
            ]
          }
        }
      ]
    },
    "id": "templeenter",
    "type": "portal-action"
  }
}
```

- [ ] **Step 5: Edit `gms/83_1/portal-actions/portals/portal-undodraco.json`**

```json
{
  "data": {
    "attributes": {
      "description": "Leafre Flight - Exit transit back to Leafre station",
      "mapId": 200090500,
      "portalId": "undodraco",
      "rules": [
        {
          "conditions": [],
          "id": "exit_transit",
          "onMatch": {
            "allow": false,
            "operations": [
              {
                "params": {},
                "type": "play_portal_sound"
              },
              {
                "params": {
                  "mapId": "240000110"
                },
                "type": "warp"
              }
            ]
          }
        }
      ]
    },
    "id": "undodraco",
    "type": "portal-action"
  }
}
```

- [ ] **Step 6: Verify the four edited files parse and are effect-op free**

```bash
for f in deploy/seed/gms/83_1/npc-conversations/npc/npc-2082003.json \
         deploy/seed/gms/83_1/portal-actions/portals/portal-outTemple.json \
         deploy/seed/gms/83_1/portal-actions/portals/portal-templeenter.json \
         deploy/seed/gms/83_1/portal-actions/portals/portal-undodraco.json; do
  python3 -c "import json; json.load(open('$f')); print('parses: $f')"
done
grep -l 'consumable_effect' deploy/seed/gms/83_1/npc-conversations/npc/npc-2082003.json \
     deploy/seed/gms/83_1/portal-actions/portals/portal-{outTemple,templeenter,undodraco}.json || echo "no effect ops remain in gms/83_1"
```

Expected: four `parses:` lines and `no effect ops remain in gms/83_1`.

Also confirm the NPC rewire left no dangling reference:

```bash
python3 - <<'PY'
import json
d = json.load(open('deploy/seed/gms/83_1/npc-conversations/npc/npc-2082003.json'))
states = d['data']['attributes']['states']
ids = {s['id'] for s in states}
assert 'applyBuff' not in ids, 'applyBuff state still present'
ask = next(s for s in states if s['id'] == 'askTransform')
assert ask['listSelection']['choices'][0]['nextState'] == 'startTransport', ask
refs = json.dumps(d)
assert 'applyBuff' not in refs, 'a dangling applyBuff reference remains'
print('npc-2082003 rewire OK')
PY
```

Expected: `npc-2082003 rewire OK`.

- [ ] **Step 7: Copy the four verified files to the other ten version directories**

```bash
for f in npc-conversations/npc/npc-2082003.json \
         portal-actions/portals/portal-outTemple.json \
         portal-actions/portals/portal-templeenter.json \
         portal-actions/portals/portal-undodraco.json; do
  for d in deploy/seed/gms/12_1 deploy/seed/gms/48_1 deploy/seed/gms/61_1 \
           deploy/seed/gms/72_1 deploy/seed/gms/79_1 deploy/seed/gms/84_1 \
           deploy/seed/gms/87_1 deploy/seed/gms/92_1 deploy/seed/gms/95_1 \
           deploy/seed/jms/185_1; do
    cp "deploy/seed/gms/83_1/$f" "$d/$f"
  done
done
```

- [ ] **Step 8: Re-assert uniformity and sweep the whole seed tree**

```bash
# 1. All four files are still byte-identical across all eleven directories.
for f in npc-conversations/npc/npc-2082003.json \
         portal-actions/portals/portal-outTemple.json \
         portal-actions/portals/portal-templeenter.json \
         portal-actions/portals/portal-undodraco.json; do
  echo -n "$f: "
  for d in deploy/seed/gms/*_1 deploy/seed/jms/185_1; do md5sum "$d/$f"; done | awk '{print $1}' | sort -u | wc -l
done
```

Expected: `1` for each of the four files.

```bash
# 2. No apply/cancel_consumable_effect operation referencing 2210016 remains
#    anywhere under deploy/seed/ (PRD §10).
grep -rl 'consumable_effect' deploy/seed/ | xargs -r grep -l '2210016' || echo "PASS: no 2210016 effect ops remain"
```

Expected: `PASS: no 2210016 effect ops remain`.

```bash
# 3. The only remaining users of these operations are out of scope.
grep -rl 'consumable_effect' deploy/seed/ | sort -u
```

Expected: only `npc-conversations/npc/npc-1101001.json` paths (item `2022458`, the stationary NPC blessing — PRD §2 non-goal).

```bash
# 4. dracoout is untouched, and exactly 44 files changed.
git status --short deploy/seed/ | grep -c '^ M'
git status --short deploy/seed/ | grep dracoout || echo "PASS: dracoout untouched"
```

Expected: `44` and `PASS: dracoout untouched`.

```bash
# 5. Every changed file still parses.
git diff --name-only deploy/seed/ | while read f; do
  python3 -c "import json,sys; json.load(open('$f'))" || echo "BAD JSON: $f"
done; echo "json sweep done"
```

Expected: `json sweep done` with no `BAD JSON` lines.

```bash
# 6. No CRLF crept in.
grep -rlU $'\r' deploy/seed/gms deploy/seed/jms || echo "PASS: no CRLF"
```

Expected: `PASS: no CRLF`.

- [ ] **Step 9: Commit**

```bash
git add deploy/seed/gms deploy/seed/jms
git commit -m "refactor(seed): remove transport effect operations now owned by atlas-transports"
```

---

### Task 10: Documentation and full verification

**Files:**
- Modify: `services/atlas-transports/docs/domain.md`
- Modify: `services/atlas-transports/docs/kafka.md`

**Interfaces:**
- Consumes: everything from Tasks 1–9.
- Produces: nothing.

- [ ] **Step 1: Document the two new `RouteModel` fields**

In `services/atlas-transports/docs/domain.md`, append two rows to the `### RouteModel (instance/model.go)` table (after `transitMessage`):

```markdown
| effectItemIds | []item.Id | Consumable item ids applied on boarding and cancelled on every terminal path. Empty = none |
| forcedReturnMapId | map.Id | Map to warp to when the travel timer expires, instead of destinationMapId. Zero = not set |
```

- [ ] **Step 2: Document the new produced topic**

In `services/atlas-transports/docs/kafka.md`, add a section under `## Topics Produced`, matching the surrounding sections' format:

```markdown
### COMMAND_TOPIC_CONSUMABLE

Emitted for routes that declare `effectItemIds`. Applies the declared effects
when a character boards an instance transport, and cancels them on every
terminal path (travel-timer arrival, entering a non-transit map, logout, stuck
timeout, graceful shutdown). Routes declaring no effects emit nothing.

`transactionId` is always the nil UUID: these are not saga-driven applications,
and atlas-saga-orchestrator skips saga completion for a nil transaction id.
`mapId`/`instance` are left zero — APPLY resolves the character's live map
itself, and CANCEL's field reaches atlas-buffs' `Cancel`, which reads only
`worldId`.

| Command Type | Body | Purpose |
|--------------|------|---------|
| APPLY_CONSUMABLE_EFFECT | `{itemId}` | Apply a route's declared transit effect on boarding |
| CANCEL_CONSUMABLE_EFFECT | `{itemId}` | Remove it on every terminal path |
```

Also add `TIMEOUT` to the documented cancel reasons in the `### EVENT_TOPIC_INSTANCE_TRANSPORT` section — read that section first and match its existing table or list format:

```markdown
| TIMEOUT | The travel timer expired on a route declaring a forced return; the character was sent back rather than delivered |
```

- [ ] **Step 3: Run the full Go verification**

```bash
cd services/atlas-transports/atlas.com/transports && go test -race ./... && go vet ./... && go build ./...
cd ../../../../ && cd services/atlas-tenants/atlas.com/tenants && go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean. Report the actual output; a failure here is not "done".

- [ ] **Step 4: Run the repo-root guards**

From the worktree root:

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: all exit 0. `tools/lint.sh --check` needs node available — if it false-fails with a node/nvm error rather than a real finding, load nvm 22 first and re-run. If it reports formatting diffs, run `tools/lint.sh` (no flags) to fix in place, then re-run `--check` and amend.

`tools/template-opcode-order-guard.sh`, `tools/service-registration-guard.sh`, and `tools/skill-job-id-guard.sh` are **not applicable** — no socket-config template, no service-registration list, and no job/skill id comparison changed. Confirm with `git diff --name-only main` rather than assuming.

- [ ] **Step 5: Confirm no `go.mod` changed (bake gate)**

```bash
git diff --name-only main -- '*/go.mod' '*/go.sum'
```

Expected: empty. If either `go.mod` did change, run `docker buildx bake atlas-transports` and/or `docker buildx bake atlas-tenants` from the worktree root and record the result — CLAUDE.md §Build & Verification item 4 makes this mandatory, not optional.

- [ ] **Step 6: Confirm the deployment premise**

FR-4.4 says no manifest change is needed. Verify rather than assume:

```bash
grep -n 'COMMAND_TOPIC_CONSUMABLE' deploy/k8s/base/env-configmap.yaml \
     deploy/k8s/overlays/main/kustomization.yaml \
     deploy/k8s/overlays/pr/kustomization.yaml
grep -n -A3 'envFrom' deploy/k8s/base/atlas-transports.yaml
```

Expected: the topic is present in the base configmap and both overlays, and `atlas-transports` mounts the shared configmap via `envFrom`. If any of the three is missing the key, that is a real deployment change and must be made here, not deferred.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-transports/docs/domain.md services/atlas-transports/docs/kafka.md
git commit -m "docs(transports): document route effect fields, consumable topic, and TIMEOUT reason"
```

- [ ] **Step 8: Code review before PR**

Invoke `superpowers:requesting-code-review` from the worktree. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no TS changed, so no frontend reviewer). Findings land in `docs/tasks/task-193-instance-route-effect-lifecycle/audit.md`. Pin the reviewer subagents to a cheaper model per CLAUDE.md §Model & Cost Preferences. Do not open the PR before this step — CLAUDE.md §Code Review Before PR.

- [ ] **Step 9: Write the operator rollout step into the PR description**

Existing tenants keep today's behaviour — including the bug — until their instance routes are re-seeded. The mechanism exists and needs no new code: `atlas-tenants` registers an `instance-routes` seed group exposing `POST /tenants/configurations/instance-routes/seed`; a run replaces the tenant's rows and its `AfterSeed` hook emits exactly one configuration-status event, which `atlas-transports` consumes as a `ClearTenant` + full registry reload. No restart, no downtime.

PR description must include:

```markdown
## Operator step (required for live tenants)

1. `POST /tenants/configurations/instance-routes/seed` for each live tenant.
2. Confirm via `GET` on atlas-transports' `instance-routes` resource that
   `effectItemIds` and `forcedReturnMapId` are populated on
   `temple-of-time-flight` and `temple-of-time-return-flight`.

A tenant that is not re-seeded keeps today's behaviour for those two routes.
The new fields are additive and optional, so existing stored configurations
continue to deserialize unchanged.

Known guard: `AfterSeed` refuses to emit when a run deletes rows but creates
none (the missing-catalog-mount signature). If the status event never arrives,
check that log line first — the seed will have reported success.
```

---

## Self-Review

**Spec coverage** — every design section maps to a task:

| design.md | Task |
|---|---|
| §2.3 layer 1 (`TransformInstanceRoute`) | 3 |
| §2.3 layer 2 (`config/rest.go`) | 2 |
| §2.3 layer 3 (`model_json.go`) | 1 |
| §2.3 layer 4 (`TransformRoute`) | 2 |
| *(new)* layer 0 (`ExtractInstanceRoute`) | 3 |
| D1 two private helpers | 5 (apply), 6 (cancel) |
| D2 attribute names + `atlas-constants` types | 1, 2, 3 |
| D3 `CANCELLED`/`TIMEOUT` | 4 (const), 7 (branch) |
| D4 local wire contract | 4 |
| D5 `MapId`/`Instance` zero | 4 |
| D6 ordering comment, no cross-topic assertion | 5 |
| D7 builder validation | 1 |
| D8 teardown hardening | 6 |
| §4.3 route declarations | 8 |
| §4.3 operation removals | 9 |
| §4.4 deployment re-verification | 10 |
| §5 behaviour matrix — all 10 rows | 5, 6, 7 |
| §7 testing strategy — all 12 rows | 1, 2, 3, 4, 5, 6, 7, 9 |
| §8 rollout | 10 |
| §11 verification | 10 |

PRD FR-1.1…1.7, FR-2.1…2.5, FR-3.1…3.7, FR-4.1…4.4 are each covered; FR-3.6 (`dracoout`) is covered by `TestHandleMapEnter_NonTransitMapCancelsEffects` in Task 6, which asserts the exact `200090510 → 240000100` shape rather than assuming it.

**Deviations from design.md, deliberate:**

1. **Layer 0 added.** `ExtractInstanceRoute` (`configuration/rest.go:477`) is the JSONB write-back used by the instance-route POST/PATCH handlers (`configuration/resource.go:509,558`). design.md's §2.3 table lists four layers and omits it; an operator PATCH would silently erase both new attributes. Same failure class the design itself warns about, so it is fixed here rather than deferred.
2. **`forceCancelInstance` / `completeInstance` extracted.** design.md keeps the loops inline in `TickStuckTimeout` / `TickArrival`. Both ticks call `time.Now()` internally and cannot be driven from a test without a clock injection this codebase does not have. Extracting the loop body makes the emission directly testable at zero behavioural cost — the ticks still own the selection and the release.

**Type consistency:** `EffectItemIds()`/`ForcedReturnMapId()`, `SetEffectItemIds`/`SetForcedReturnMapId`, `applyRouteEffects`/`cancelRouteEffects`, `applyConsumableEffectProvider`/`cancelConsumableEffectProvider`, `forceCancelInstance`/`completeInstance` are each used with one spelling and one signature throughout. Domain types are `[]item.Id` and `_map.Id` in `atlas-transports`, `[]uint32`/`uint32` in `atlas-tenants` (matching that service's existing style). JSON attribute names `effectItemIds` and `forcedReturnMapId` are identical across Tasks 2, 3, and 8.
