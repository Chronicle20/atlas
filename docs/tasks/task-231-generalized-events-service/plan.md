# Generalized Events Service (`atlas-events`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce `atlas-events`, a service owning event definitions, durable
scheduled work, occurrences and their history, proven by two dissimilar events
(Crimson Balrog, Anniversary), plus the cross-service primitives they need —
transport voyage identity and generic monster spawn provenance.

**Architecture:** A generic core (`event/definition`, `event/occurrence`,
`event/transition`, `event/scheduling`, `event/registry`) that never names an
event type, plus per-event packages (`events/crimsonbalrog`,
`events/anniversary`) that register a `registry.Handler`. Correctness-critical
state is Postgres rows; the only in-memory component is a stateless poller that
claims work with `SELECT … FOR UPDATE SKIP LOCKED`. Gameplay primitives stay with
their owning services — `atlas-events` issues Kafka commands and never builds a
packet.

**Tech Stack:** Go 1.25.5 (match `services/atlas-party-quests/atlas.com/party-quests/go.mod`), GORM + Postgres,
`segmentio/kafka-go` via `libs/atlas-kafka`, JSON:API via `jtumidanski/api2go`,
`libs/atlas-model` provider/operator composition, `libs/atlas-seeder` for seed
data, React 19 + TanStack Query + shadcn/ui for `atlas-ui`.

**Spec:** [`design.md`](design.md) (PRD: [`prd.md`](prd.md))

## Global Constraints

Every task's requirements implicitly include this section.

- **Worktree.** All work happens in `.worktrees/task-231-generalized-events-service`
  on branch `task-231-generalized-events-service`. Never edit the main repo.
- **Verification scope for implementers.** Module-local `go build ./... && go test ./...`
  only. `tools/verify.sh`, `tools/lint.sh`, `-race` and docker bake belong to the
  `atlas-verifier` agent. (CLAUDE.md, Model & Cost Preferences.)
- **No invention.** Any value not verified from source, WZ data, IDA or live
  output is "unverified" — do not guess. (CLAUDE.md, Grounding.)
- **`libs/atlas-constants` first.** Before defining a domain type, alias or
  numeric constant, check `libs/atlas-constants/` (DOM-21 / FR-N19).
  `world.Id` and `channel.Id` are `byte`; `_map.Id` is `uint32`;
  fields are built with `field.NewBuilder(w, c, m).SetInstance(uuid).Build()`.
- **Builder pattern in tests.** No `*_testhelpers.go` files (FR-N20).
- **Repo-relative paths only** in committed files — never `/home/<name>/…`.
- **Immutable domain models**, builder construction, `provider.go` for reads,
  `administrator.go` for writes, `processor.go` for business logic, REST
  representations separate from domain models (FR-N17). Business logic lives in
  processors, never in REST handlers or Kafka consumers (FR-N18).
- **`COMMAND_TOPIC_MONSTER` fans every message to every handler.** Any new JSON
  field name on that topic must either appear in no sibling body, or appear with
  an identical Go type — otherwise every message logs a spurious unmarshal error.
  Verified absent today: `spawnSourceType`, `spawnSourceId`. (design G4;
  the constraint is documented at `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`
  on `killCommandBody` and `catchCommandBody`.)
- **Buff `duration` is MILLISECONDS** (`ApplyCommandBody.Duration`;
  `tools/buff-duration-guard.sh` enforces it).
- **The generic layer must never switch on event type** (FR-X3). Task 39's AST
  test enforces this; do not add a `switch type { case CRIMSON_BALROG: … }`.
- **`enabled` is never rendered as "occurring"** (FR-UI4).
- **Fixed values, copied verbatim from the design:**
  - Crimson Balrog visual: `CONTI_MOVE(state=10, subState=4)` on show,
    `CONTI_MOVE(state=10, subState=5)` on hide.
  - Anniversary stats: `EXP_BUFF_RATE` → rate type `exp`, `ITEM_UP_BY_ITEM` →
    rate type `item_drop`, both `ConversionDirect` (`amount / 100.0`), so
    `expMultiplier: 2.0` is carried as `amount = 200`. `EVENT_RATE` is rejected
    (JMS movement-affecting set).
  - Spawn source types: `CYCLIC`, `EVENT`, `SCRIPT`, `GM`. Absent/empty
    normalizes to `CYCLIC` at the consumer boundary only.
  - Occurrence states: `ACTIVE`, `COMPLETED`, `CANCELLED`, `FAILED`.
  - Work states: `PENDING`, `PROCESSING`, `COMPLETED`, `CANCELLED`, `FAILED`.
  - Work types: `TRIGGER_EVALUATION`, `OCCURRENCE_TRANSITION`.
  - Completion reasons: `MONSTERS_ELIMINATED`, `VESSEL_ARRIVED`, `SCHEDULED_END`.

## Convention: elided bodies

A test written as `func TestSomething(t *testing.T) { ... }` is a **required**
test, not an optional one. Its body follows the shape of the sibling
immediately above it in the same code block — same harness, same fakes, same
assertion style — varying only in what its name says. The ellipsis exists so
this document stays readable, not because the test is negotiable: a task is not
done while one of its named tests is missing.

The same reading applies to an implementation snippet written
`func f(...) T { ... }`: the signature and the doc comment are fixed by this
plan, the body is yours.

---

## File Structure

New service, `services/atlas-events/atlas.com/events/`:

| Path | Responsibility |
|---|---|
| `main.go` | Bootstrap, migrations, consumer + handler registration, poller goroutine, REST routes, handler registration into `event/registry` |
| `event/registry/handler.go` | The `Handler` interface — the only seam between generic and specific |
| `event/registry/registry.go` | `Register` / `Get` by type string |
| `event/definition/{model,builder,entity,provider,administrator,processor,rest,resource,subdomain}.go` | Definitions (§5) |
| `event/occurrence/{model,builder,entity,provider,administrator,processor,rest,resource,progress,seed}.go` | Occurrences + map/monster child tables (§6) |
| `event/transition/{model,builder,entity,provider,rest}.go` | Transition history (§7) |
| `event/scheduling/{model,builder,entity,provider,administrator,processor,poller}.go` | Durable work + claim-locked poller (§8) |
| `events/crimsonbalrog/{config,handler,consumer_departed,consumer_arrived,consumer_monster}.go` | Event 1 (§10) |
| `events/anniversary/{config,handler,consumer_login}.go` | Event 2 (§11) |
| `kafka/message/{transport,monster,character,buff,event}/kafka.go` | Mirrored consumed contracts + produced commands |
| `kafka/consumer/{transport,monster,character}/consumer.go` | Kafka reactions, owned by the event packages |
| `rest/handler.go` | Shared REST plumbing |

Changed elsewhere: `atlas-monsters` (provenance), `atlas-transports` (voyage
identity + two events), `atlas-buffs` (correlation + cancel-by-correlation),
`atlas-rates` (two mappings), `atlas-channel` (visual consumer + map-entry
block), `atlas-ui` (four pages), plus the registration files in
`docs/adding-a-new-service.md`.

---

# Phase A — `atlas-monsters` spawn provenance (design §8)

Self-contained: no consumer of the new fields exists yet, and every field is
optional on the wire, so this phase lands and verifies on its own.

## Task 1: Provenance on the monster domain model and its Redis form

**Module root for `go build`/`go test`:** `services/atlas-monsters/atlas.com/monsters`

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/model.go` — add two fields to `Model`, two accessors, widen `NewMonster`
- `services/atlas-monsters/atlas.com/monsters/monster/builder.go` — carry the fields through `Clone`/`ModelBuilder`, add setters
- `services/atlas-monsters/atlas.com/monsters/monster/registry.go` — add the fields to `storedMonster` (line ~25-52), `toStored`, `fromStored`, and widen `CreateMonster` (line ~361)
- `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go` — new file if absent; round-trip tests

Patterns to copy: the `Team`/`Stance` fields are threaded through exactly these
five places — follow them literally.

**Interfaces:**
- Produces: `monster.Model.SpawnSourceType() string`, `monster.Model.SpawnSourceId() string`,
  `monster.NewMonster(f field.Model, uniqueId, monsterId uint32, x, y, fh int16, stance byte, team int8, hp, mp uint32, spawnSourceType string, spawnSourceId string) Model`,
  `(*Registry).CreateMonster(ctx context.Context, t tenant.Model, f field.Model, monsterId uint32, x, y, fh int16, stance byte, team int8, hp, mp uint32, spawnSourceType string, spawnSourceId string) Model`,
  `monster.Clone(m).SetSpawnSource(sourceType string, sourceId string) *ModelBuilder`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go`:

```go
package monster

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestStoredMonsterRoundTripsSpawnSource(t *testing.T) {
	tm := testTenant(t)
	f := field.NewBuilder(1, 4, 200090010).Build()
	m := NewMonster(f, 7, 8150000, 10, 20, 30, 5, 0, 100, 0, "EVENT", "occ-1")

	_, back, err := fromStored(toStored(tm, m))
	if err != nil {
		t.Fatalf("fromStored: %v", err)
	}
	if back.SpawnSourceType() != "EVENT" || back.SpawnSourceId() != "occ-1" {
		t.Fatalf("provenance lost: type=%q id=%q", back.SpawnSourceType(), back.SpawnSourceId())
	}
}

// An old Redis payload has neither field. It must unmarshal to empty strings —
// normalization to CYCLIC happens at the consumer boundary (Task 2), not here,
// so the storage layer stays free of the enum.
func TestStoredMonsterTolueratesLegacyPayload(t *testing.T) {
	tm := testTenant(t)
	f := field.NewBuilder(1, 4, 200090010).Build()
	m := NewMonster(f, 7, 8150000, 10, 20, 30, 5, 0, 100, 0, "", "")

	_, back, err := fromStored(toStored(tm, m))
	if err != nil {
		t.Fatalf("fromStored: %v", err)
	}
	if back.SpawnSourceType() != "" || back.SpawnSourceId() != "" {
		t.Fatalf("expected empty provenance, got type=%q id=%q", back.SpawnSourceType(), back.SpawnSourceId())
	}
}

func TestCloneCarriesSpawnSource(t *testing.T) {
	f := field.NewBuilder(1, 4, 200090010).Build()
	m := NewMonster(f, 7, 8150000, 10, 20, 30, 5, 0, 100, 0, "EVENT", "occ-1")

	c := Clone(m).SetX(99).Build()
	if c.SpawnSourceType() != "EVENT" || c.SpawnSourceId() != "occ-1" {
		t.Fatalf("Clone dropped provenance: type=%q id=%q", c.SpawnSourceType(), c.SpawnSourceId())
	}
}
```

If `tenant.Create` or `Clone(...).Build()` has a different signature in this
module, adapt the two helper lines to the local one — read `monster/builder.go`'s
`Build` and one existing test in the package first. Do not change the assertions.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'SpawnSource|LegacyPayload' -v`
Expected: FAIL — compile error, `NewMonster` takes 11 arguments, and
`SpawnSourceType` is undefined.

- [ ] **Step 3: Add the fields to `Model`**

In `monster/model.go`, add to the `Model` struct after `lastDamageTakenMs`:

```go
	// spawnSourceType / spawnSourceId are opaque provenance, set by whatever
	// asked for the spawn (FR-P1). atlas-monsters stores, echoes and compares
	// them for equality; it never interprets spawnSourceId (FR-P6). Empty means
	// the producer omitted them; normalization to CYCLIC happens once, at the
	// consumer boundary.
	spawnSourceType string
	spawnSourceId   string
```

Add the accessors next to `Team()`:

```go
func (m Model) SpawnSourceType() string { return m.spawnSourceType }
func (m Model) SpawnSourceId() string   { return m.spawnSourceId }
```

Widen the constructor:

```go
func NewMonster(f field.Model, uniqueId uint32, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32, spawnSourceType string, spawnSourceId string) Model {
```

and set the two fields in the returned literal.

- [ ] **Step 4: Thread the fields through the builder**

In `monster/builder.go`, add `spawnSourceType string` and `spawnSourceId string`
to `ModelBuilder`, copy them in `Clone`, copy them into the `Model` in `Build`,
and add:

```go
// SetSpawnSource sets the opaque spawn provenance pair.
func (b *ModelBuilder) SetSpawnSource(sourceType string, sourceId string) *ModelBuilder {
	b.spawnSourceType = sourceType
	b.spawnSourceId = sourceId
	return b
}
```

- [ ] **Step 5: Thread the fields through Redis storage**

In `monster/registry.go`, add to `storedMonster` after `LastDamageTakenMs`:

```go
	SpawnSourceType        string           `json:"spawnSourceType,omitempty"`
	SpawnSourceId          string           `json:"spawnSourceId,omitempty"`
```

`omitempty` keeps existing payloads byte-identical. Set them in `toStored` and
read them in `fromStored`. Then widen `CreateMonster`:

```go
func (r *Registry) CreateMonster(ctx context.Context, t tenant.Model, f field.Model, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32, spawnSourceType string, spawnSourceId string) Model {
	uniqueId := GetIdAllocator().Allocate(ctx, t)
	m := NewMonster(f, uniqueId, monsterId, x, y, fh, stance, team, hp, mp, spawnSourceType, spawnSourceId)
	...
}
```

- [ ] **Step 6: Fix the one existing caller**

`monster/processor.go:~224` calls `CreateMonster`. Pass `input.SpawnSourceType,
input.SpawnSourceId` — those RestModel fields are added in Task 2, so for now
pass `"", ""` and leave a one-line comment `// wired to the command body in Task 2`.
Compile the module and fix any other call sites the compiler names.

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/ -v`
Expected: PASS, including every pre-existing test in the package unmodified
(FR-P5 / acceptance 20.7 "existing monster spawn behavior is unchanged").

- [ ] **Step 8: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/
git commit -m "feat(task-231): carry spawn provenance on the monster model and its Redis form"
```

---

## Task 2: `SPAWN_FIELD` provenance fields and boundary normalization

**Module root:** `services/atlas-monsters/atlas.com/monsters`

### Files

- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go` — two fields on `spawnFieldCommandBody`; add `SpawnSourceType*` constants
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go:339` — `handleSpawnFieldCommand`, normalize and pass through
- `services/atlas-monsters/atlas.com/monsters/monster/rest.go` — two fields on `RestModel`
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go:216` — `Create` passes them to `CreateMonster`
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer_test.go` — new file; normalization tests

**Interfaces:**
- Consumes: `monster.Model.SpawnSourceType/SpawnSourceId`, widened `Registry.CreateMonster` (Task 1).
- Produces: `monster.SpawnSourceTypeCyclic/Event/Script/GM` string constants in
  package `monster` (`monster/model.go`), `monster.RestModel.SpawnSourceType`,
  `monster.RestModel.SpawnSourceId`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer_test.go`:

```go
package monster

import (
	"encoding/json"
	"testing"

	"atlas-monsters/monster"
)

// FR-P5: a producer that omits the provenance fields must produce byte-identical
// output to today. This pins the omitempty tags.
func TestSpawnFieldBodyOmitsProvenanceWhenUnset(t *testing.T) {
	b, err := json.Marshal(spawnFieldCommandBody{MonsterId: 8150000, X: 1, Y: 2, Fh: 3, Team: 0})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"monsterId":8150000,"x":1,"y":2,"fh":3,"team":0}`
	if string(b) != want {
		t.Fatalf("wire shape changed:\n got %s\nwant %s", b, want)
	}
}

func TestSpawnFieldBodyCarriesProvenanceWhenSet(t *testing.T) {
	b, err := json.Marshal(spawnFieldCommandBody{
		MonsterId: 8150000, SpawnSourceType: "EVENT", SpawnSourceId: "occ-1",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"monsterId":8150000,"x":0,"y":0,"fh":0,"team":0,"spawnSourceType":"EVENT","spawnSourceId":"occ-1"}`
	if string(b) != want {
		t.Fatalf("wire shape:\n got %s\nwant %s", b, want)
	}
}

// FR-P1: absent or empty normalizes to CYCLIC, once, at the boundary.
func TestNormalizeSpawnSourceType(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", monster.SpawnSourceTypeCyclic},
		{"EVENT", "EVENT"},
		{"GM", "GM"},
	} {
		if got := normalizeSpawnSourceType(tc.in); got != tc.want {
			t.Fatalf("normalizeSpawnSourceType(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./kafka/consumer/monster/ -v`
Expected: FAIL — `SpawnSourceType` is not a field of `spawnFieldCommandBody`,
`normalizeSpawnSourceType` undefined.

- [ ] **Step 3: Add the constants**

In `monster/model.go`, next to the other domain constants:

```go
// Spawn provenance types (FR-P1). The set is open — atlas-monsters never
// interprets these beyond equality, so a new producer may introduce a value
// without a change here. CYCLIC is the normalization target for an absent or
// empty value, applied once at the Kafka consumer boundary.
const (
	SpawnSourceTypeCyclic = "CYCLIC"
	SpawnSourceTypeEvent  = "EVENT"
	SpawnSourceTypeScript = "SCRIPT"
	SpawnSourceTypeGM     = "GM"
)
```

- [ ] **Step 4: Widen the command body**

In `kafka/consumer/monster/kafka.go`:

```go
// spawnFieldCommandBody carries optional spawn provenance (FR-P1). Both field
// names are absent from every sibling body on this shared, fan-to-every-handler
// topic, so adding them cannot produce a spurious unmarshal error the way a
// type-mismatched name would (see killCommandBody's note). omitempty keeps an
// omitting producer's bytes identical to today (FR-P5).
type spawnFieldCommandBody struct {
	MonsterId       uint32 `json:"monsterId"`
	X               int16  `json:"x"`
	Y               int16  `json:"y"`
	Fh              int16  `json:"fh"`
	Team            int8   `json:"team"`
	SpawnSourceType string `json:"spawnSourceType,omitempty"`
	SpawnSourceId   string `json:"spawnSourceId,omitempty"`
}
```

- [ ] **Step 5: Normalize at the boundary and pass through**

In `kafka/consumer/monster/consumer.go`, add above `handleSpawnFieldCommand`:

```go
// normalizeSpawnSourceType is the single enforcement point for FR-P1's
// backward-compatibility rule: an absent or empty spawnSourceType means the
// legacy cyclic spawn path. Doing it here means no read site downstream has to
// handle the empty case.
func normalizeSpawnSourceType(s string) string {
	if s == "" {
		return monster.SpawnSourceTypeCyclic
	}
	return s
}
```

and change the `Create` call:

```go
	_, err := p.Create(f, monster.RestModel{
		MonsterId:       c.Body.MonsterId,
		X:               c.Body.X,
		Y:               c.Body.Y,
		Fh:              c.Body.Fh,
		Team:            c.Body.Team,
		SpawnSourceType: normalizeSpawnSourceType(c.Body.SpawnSourceType),
		SpawnSourceId:   c.Body.SpawnSourceId,
	})
```

- [ ] **Step 6: Carry them through `RestModel` and `Create`**

In `monster/rest.go`, add to `RestModel` after `NextEligibleRepickAtMs`:

```go
	SpawnSourceType        string              `json:"spawnSourceType,omitempty"`
	SpawnSourceId          string              `json:"spawnSourceId,omitempty"`
```

In `monster/processor.go`'s `Create`, replace the `"", ""` placeholder from
Task 1 Step 6 with `input.SpawnSourceType, input.SpawnSourceId`. Also populate
the two fields in whichever `Transform`/`Extract` function maps `Model` ↔
`RestModel` in `monster/rest.go`, so `GET /monsters/{id}` exposes the
provenance (FR-P2's "persists for its lifetime" is observable).

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... `
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/
git commit -m "feat(task-231): accept spawn provenance on SPAWN_FIELD and normalize at the boundary"
```

---

## Task 3: Echo provenance on every monster status event

**Module root:** `services/atlas-monsters/atlas.com/monsters`

The fields go on the **envelope**, not per-body: `statusEventFromField` is the
single constructor for every status event, so one change echoes provenance on
`CREATED`, `KILLED` and `DESTROYED` (FR-P3) and on every other type at no extra
cost, with no risk of one being forgotten.

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/kafka.go` — two fields on `statusEvent[E]`, populated in `statusEventFromField`
- `services/atlas-monsters/atlas.com/monsters/monster/producer.go` — read-only unless a provider bypasses `statusEventFromField`; if one does, route it through
- `services/atlas-monsters/atlas.com/monsters/monster/kafka_test.go` — new file

**Interfaces:**
- Consumes: `monster.Model.SpawnSourceType/SpawnSourceId` (Task 1).
- Produces: JSON fields `spawnSourceType` / `spawnSourceId` on every
  `EVENT_TOPIC_MONSTER_STATUS` message envelope. Consumed by Task 26.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-monsters/atlas.com/monsters/monster/kafka_test.go`:

```go
package monster

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestStatusEventEnvelopeEchoesProvenance(t *testing.T) {
	f := field.NewBuilder(1, 4, 200090010).Build()
	m := NewMonster(f, 7, 8150000, 0, 0, 0, 5, 0, 100, 0, SpawnSourceTypeEvent, "occ-1")

	e := statusEventFromField(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusKilled,
		statusEventKilledBody{}, m.SpawnSourceType(), m.SpawnSourceId())

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"spawnSourceType":"EVENT"`) ||
		!strings.Contains(string(b), `"spawnSourceId":"occ-1"`) {
		t.Fatalf("provenance not echoed: %s", b)
	}
}

func TestStatusEventEnvelopeOmitsEmptyProvenance(t *testing.T) {
	f := field.NewBuilder(1, 4, 200090010).Build()
	e := statusEventFromField(f, 7, 8150000, EventMonsterStatusCreated,
		statusEventCreatedBody{}, "", "")

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "spawnSource") {
		t.Fatalf("expected no provenance keys, got: %s", b)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run StatusEventEnvelope -v`
Expected: FAIL — `statusEventFromField` takes 5 arguments.

- [ ] **Step 3: Widen the envelope**

In `monster/kafka.go`:

```go
type statusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	UniqueId  uint32     `json:"uniqueId"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	// SpawnSourceType / SpawnSourceId echo the monster's provenance on EVERY
	// status type (FR-P3 requires CREATED/KILLED/DESTROYED; the envelope gives
	// the superset for free and makes forgetting one impossible). omitempty so
	// a cyclic monster's events are byte-identical to today.
	SpawnSourceType string `json:"spawnSourceType,omitempty"`
	SpawnSourceId   string `json:"spawnSourceId,omitempty"`
	Body            E      `json:"body"`
}

func statusEventFromField[E any](f field.Model, uniqueId uint32, monsterId uint32, theType string, body E, spawnSourceType string, spawnSourceId string) statusEvent[E] {
	return statusEvent[E]{
		WorldId:         f.WorldId(),
		ChannelId:       f.ChannelId(),
		MapId:           f.MapId(),
		Instance:        f.Instance(),
		UniqueId:        uniqueId,
		MonsterId:       monsterId,
		Type:            theType,
		SpawnSourceType: spawnSourceType,
		SpawnSourceId:   spawnSourceId,
		Body:            body,
	}
}
```

- [ ] **Step 4: Fix every call site**

Compile and let the compiler enumerate them — they are all in
`monster/producer.go`. Each provider already has the `Model` in scope (or the
field + ids); pass `m.SpawnSourceType(), m.SpawnSourceId()`. Where a provider
only has a `field.Model` and ids and no `Model`, look up the monster or thread
the pair in from the caller — do **not** pass `"", ""` to make it compile, as
that silently drops the echo the completion path depends on.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/
git commit -m "feat(task-231): echo spawn provenance on the monster status event envelope"
```

---

## Task 4: `DESTROY_BY_SOURCE` field command

**Module root:** `services/atlas-monsters/atlas.com/monsters`

Field-scoped, per design §15.6: `Registry.GetMonstersInMap` makes a field sweep
one existing call, whereas a global variant would need a new Redis secondary
index maintained on every spawn and death, which nothing in this task needs.

### Files

- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go` — `CommandTypeDestroyBySource` + `destroyBySourceCommandBody`
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go` — `handleDestroyBySourceCommand` + its `rf(...)` registration alongside `handleDestroyFieldCommand` (~line 75)
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — `DestroyBySource`, next to `DestroyInField` (line ~1351)
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` — add cases (create the file if absent)

**Interfaces:**
- Consumes: `monster.Model.SpawnSourceType/SpawnSourceId` (Task 1), `Registry.GetMonstersInMap` (`monster/registry.go:376`), `(*ProcessorImpl).Destroy` (`monster/processor.go:1339`).
- Produces: command type string `"DESTROY_BY_SOURCE"` on `COMMAND_TOPIC_MONSTER`
  with body `{"spawnSourceType":…,"spawnSourceId":…}` inside the standard
  `fieldCommand[E]` envelope. Emitted by Task 27.
  `monster.Processor.DestroyBySource(f field.Model, sourceType string, sourceId string) error`.

- [ ] **Step 1: Write the failing test**

Add to `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`:

```go
// FR-P4 / FR-B20: destroying by a source that matches nothing is success, not
// an error — arrival cleanup runs routinely after every monster was already
// killed.
func TestDestroyBySourceMatchingNothingSucceeds(t *testing.T) {
	p := newTestProcessor(t) // see the package's existing processor test setup
	f := field.NewBuilder(1, 4, 200090010).Build()

	if err := p.DestroyBySource(f, monster.SpawnSourceTypeEvent, "no-such-occurrence"); err != nil {
		t.Fatalf("expected success for zero matches, got %v", err)
	}
}

// Only monsters matching BOTH halves of the pair are destroyed; a cyclic
// monster sharing the map is untouched.
func TestDestroyBySourceMatchesOnBothHalves(t *testing.T) {
	p := newTestProcessor(t)
	f := field.NewBuilder(1, 4, 200090010).Build()

	mine := GetMonsterRegistry().CreateMonster(ctxFor(t), tenantFor(t), f, 8150000, 0, 0, 0, 5, 0, 100, 0, monster.SpawnSourceTypeEvent, "occ-1")
	other := GetMonsterRegistry().CreateMonster(ctxFor(t), tenantFor(t), f, 8150000, 0, 0, 0, 5, 0, 100, 0, monster.SpawnSourceTypeEvent, "occ-2")
	cyclic := GetMonsterRegistry().CreateMonster(ctxFor(t), tenantFor(t), f, 100100, 0, 0, 0, 5, 0, 100, 0, monster.SpawnSourceTypeCyclic, "")

	if err := p.DestroyBySource(f, monster.SpawnSourceTypeEvent, "occ-1"); err != nil {
		t.Fatalf("DestroyBySource: %v", err)
	}

	left := GetMonsterRegistry().GetMonstersInMap(tenantFor(t), f)
	ids := map[uint32]bool{}
	for _, m := range left {
		ids[m.UniqueId()] = true
	}
	if ids[mine.UniqueId()] {
		t.Fatalf("matching monster survived")
	}
	if !ids[other.UniqueId()] || !ids[cyclic.UniqueId()] {
		t.Fatalf("non-matching monsters were destroyed")
	}
}
```

Read the package's existing processor test file first and reuse whatever
`newTestProcessor` / tenant / context helpers it already has (miniredis or a
fake); do not invent a new harness and do not add a `*_testhelpers.go` file.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run DestroyBySource -v`
Expected: FAIL — `DestroyBySource` undefined.

- [ ] **Step 3: Implement `DestroyBySource`**

In `monster/processor.go`, after `DestroyInField`:

```go
// DestroyBySource despawns every live monster in f whose provenance pair equals
// (sourceType, sourceId). Zero matches is success (FR-P4): the caller's
// cleanup is idempotent by construction, and arrival-after-everything-died is
// the ordinary case, not an error path. atlas-monsters never interprets
// sourceId — it compares it for equality and nothing else (FR-P6).
func (p *ProcessorImpl) DestroyBySource(f field.Model, sourceType string, sourceId string) error {
	for _, m := range GetMonsterRegistry().GetMonstersInMap(p.t, f) {
		if m.SpawnSourceType() != sourceType || m.SpawnSourceId() != sourceId {
			continue
		}
		if err := p.Destroy(m.UniqueId()); err != nil {
			p.l.WithError(err).Warnf("Unable to destroy monster [%d] for source [%s/%s].", m.UniqueId(), sourceType, sourceId)
		}
	}
	return nil
}
```

Add `DestroyBySource(f field.Model, sourceType string, sourceId string) error`
to the `Processor` interface in the same file, and regenerate/extend
`monster/mock/processor.go` if the package has a mock for `Processor`.

A per-monster destroy failure logs and continues rather than aborting the sweep:
a single stale Redis key must not leave the rest of an occurrence's monsters
alive.

- [ ] **Step 4: Add the command and handler**

In `kafka/consumer/monster/kafka.go`, add to the const block:

```go
	CommandTypeDestroyBySource   = "DESTROY_BY_SOURCE"
```

and the body:

```go
// destroyBySourceCommandBody despawns every monster in the field matching the
// provenance pair. Field-scoped by design (design §15.6): Registry has a
// map index but no source index, so a global variant would need a new secondary
// index maintained on every spawn and death. Both field names are shared with
// spawnFieldCommandBody at the identical type, which is what keeps this safe on
// the fan-to-every-handler command topic.
type destroyBySourceCommandBody struct {
	SpawnSourceType string `json:"spawnSourceType"`
	SpawnSourceId   string `json:"spawnSourceId"`
}
```

In `kafka/consumer/monster/consumer.go`:

```go
func handleDestroyBySourceCommand(l logrus.FieldLogger, ctx context.Context, c fieldCommand[destroyBySourceCommandBody]) {
	if c.Type != CommandTypeDestroyBySource {
		return
	}

	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	if err := monster.NewProcessor(l, ctx).DestroyBySource(f, normalizeSpawnSourceType(c.Body.SpawnSourceType), c.Body.SpawnSourceId); err != nil {
		l.WithError(err).Errorf("DESTROY_BY_SOURCE failed for source [%s/%s] in field [%s].", c.Body.SpawnSourceType, c.Body.SpawnSourceId, f.Id())
	}
}
```

Register it in `InitHandlers` immediately after the `handleDestroyFieldCommand`
registration.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/
git commit -m "feat(task-231): add field-scoped DESTROY_BY_SOURCE monster command"
```

---

# Phase B — `atlas-transports` voyage identity (design §7)

Self-contained: the two new events have no consumer yet, and per design G2 the
`atlas-channel` route consumer already type-guards, so nothing downstream
changes behavior.

## Task 5: Derive a voyage id; widen `Transition` with the selected trip

**Module root:** `services/atlas-transports/atlas.com/transports`

Per design G3 a trip already has a `tripId`; what it lacks is per-occurrence
identity, because the schedule is time-of-day recurrent and `Evaluate` re-derives
state from the clock each tick. So the voyage id is **derived**, not stored — it
survives restart, a Redis flush and replication, and needs no write path
(design §7.1, §14 A3).

### Files

- `services/atlas-transports/atlas.com/transports/transport/voyage.go` — **new file**; the derivation
- `services/atlas-transports/atlas.com/transports/transport/voyage_test.go` — **new file**
- `services/atlas-transports/atlas.com/transports/transport/model.go` — add `TripId` and `DepartedAt` to `Transition` (line ~115-123), populate them in `Evaluate` (line ~148-227), add `UpdateStateWithTransition`
- `services/atlas-transports/atlas.com/transports/transport/evaluate_test.go` — extend
- `services/atlas-transports/atlas.com/transports/transport/state_test.go` — read-only reference for the existing branch table

**Interfaces:**
- Produces:
  - `transport.VoyageId(t tenant.Model, routeId uuid.UUID, tripId uuid.UUID, departedAt time.Time) uuid.UUID`
  - `transport.Transition{State, NextState, NextAt, TripId uuid.UUID, DepartedAt time.Time}`
  - `(Model).UpdateStateWithTransition(now time.Time) (Model, bool, Transition, error)`
  - `(Model).UpdateState(now) (Model, bool, error)` — unchanged signature, now delegating.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-transports/atlas.com/transports/transport/voyage_test.go`:

```go
package transport

import (
	"testing"
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func voyageTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.MustParse("11111111-1111-1111-1111-111111111111"), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

// FR-V5: the same voyage derives the same id, so a process that restarted
// mid-trip still matches VOYAGE_ARRIVED to VOYAGE_DEPARTED.
func TestVoyageIdIsStableForOneVoyage(t *testing.T) {
	tm := voyageTenant(t)
	routeId, tripId := uuid.New(), uuid.New()
	dep := time.Date(2026, 8, 15, 13, 5, 0, 0, time.UTC)

	a := VoyageId(tm, routeId, tripId, dep)
	b := VoyageId(tm, routeId, tripId, dep.Add(37*time.Minute)) // same calendar day
	if a != b {
		t.Fatalf("id not stable within a day: %s vs %s", a, b)
	}
}

func TestVoyageIdDiffersAcrossConsecutiveDays(t *testing.T) {
	tm := voyageTenant(t)
	routeId, tripId := uuid.New(), uuid.New()
	dep := time.Date(2026, 8, 15, 13, 5, 0, 0, time.UTC)

	if VoyageId(tm, routeId, tripId, dep) == VoyageId(tm, routeId, tripId, dep.AddDate(0, 0, 1)) {
		t.Fatalf("consecutive days collided")
	}
}

func TestVoyageIdDiffersAcrossTenantsRoutesAndTrips(t *testing.T) {
	tmA := voyageTenant(t)
	tmB, err := tenant.Create(uuid.MustParse("22222222-2222-2222-2222-222222222222"), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	routeId, tripId := uuid.New(), uuid.New()
	dep := time.Date(2026, 8, 15, 13, 5, 0, 0, time.UTC)

	base := VoyageId(tmA, routeId, tripId, dep)
	if base == VoyageId(tmB, routeId, tripId, dep) {
		t.Fatalf("tenants collided")
	}
	if base == VoyageId(tmA, uuid.New(), tripId, dep) {
		t.Fatalf("routes collided")
	}
	if base == VoyageId(tmA, routeId, uuid.New(), dep) {
		t.Fatalf("trips collided")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./transport/ -run Voyage -v`
Expected: FAIL — `VoyageId` undefined.

- [ ] **Step 3: Implement the derivation**

Create `services/atlas-transports/atlas.com/transports/transport/voyage.go`:

```go
package transport

import (
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// voyageNamespace scopes voyage-id derivation. Generated once and frozen: it is
// part of the wire contract, because atlas-events matches VOYAGE_ARRIVED to
// VOYAGE_DEPARTED by id equality alone. Changing it orphans every in-flight
// occurrence.
var voyageNamespace = uuid.MustParse("6f3a1b2c-9d4e-4a71-8c55-0b7f2d9e4a10")

// VoyageId derives the durable identity of one trip of one route on one day
// (FR-V1, FR-V5). It is a pure function of facts the service already holds, not
// stored state, which is what makes it survive an atlas-transports restart, a
// Redis flush of the route registry, and two replicas deriving independently
// (design §7.1). departedAt is truncated to the calendar day in its own
// location — ComputeSchedule emits at most one row per tripId per day and
// Evaluate selects at most one in-transit trip, so the day is a sufficient
// discriminator.
func VoyageId(t tenant.Model, routeId uuid.UUID, tripId uuid.UUID, departedAt time.Time) uuid.UUID {
	key := t.Id().String() + "|" + routeId.String() + "|" + tripId.String() + "|" + departedAt.Format("2006-01-02")
	return uuid.NewSHA1(voyageNamespace, []byte(key))
}
```

- [ ] **Step 4: Run the voyage tests**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./transport/ -run Voyage -v`
Expected: PASS.

- [ ] **Step 5: Write the failing test for the widened `Transition`**

Add to `services/atlas-transports/atlas.com/transports/transport/evaluate_test.go`:

```go
// The departure instant a VOYAGE_ARRIVED must report is the departure of the
// trip that is arriving — which, for a midnight-crossing trip observed after
// midnight, is on the PREVIOUS calendar day. Getting this wrong makes the
// arrival derive a different voyage id than the departure did, and a Balrog
// occurrence never completes on arrival (design §18, risk 1).
func TestEvaluateReportsSelectedTripAndDepartureInstant(t *testing.T) {
	routeId := uuid.New()
	tripId := uuid.New()

	// Midnight-crossing trip: departs 23:30, arrives 00:30.
	trip := NewTripScheduleModel(tripId, routeId,
		tod(22, 30), tod(23, 20), tod(23, 30), tod(0, 30))
	m := routeWithSchedule(t, routeId, []TripScheduleModel{trip})

	// Observed at 23:40 on the 15th — in transit, departed today.
	tr := m.Evaluate(time.Date(2026, 8, 15, 23, 40, 0, 0, time.UTC))
	if tr.State != InTransit {
		t.Fatalf("state = %v, want InTransit", tr.State)
	}
	if tr.TripId != tripId {
		t.Fatalf("TripId = %s, want %s", tr.TripId, tripId)
	}
	want := time.Date(2026, 8, 15, 23, 30, 0, 0, time.UTC)
	if !tr.DepartedAt.Equal(want) {
		t.Fatalf("DepartedAt = %s, want %s", tr.DepartedAt, want)
	}

	// Observed at 00:10 on the 16th — still the same voyage, departed YESTERDAY.
	tr = m.Evaluate(time.Date(2026, 8, 16, 0, 10, 0, 0, time.UTC))
	if tr.State != InTransit {
		t.Fatalf("state = %v, want InTransit", tr.State)
	}
	if !tr.DepartedAt.Equal(want) {
		t.Fatalf("post-midnight DepartedAt = %s, want %s (previous day)", tr.DepartedAt, want)
	}
}

// A same-day trip's departure materializes onto today.
func TestEvaluateDepartureInstantForSameDayTrip(t *testing.T) {
	routeId := uuid.New()
	tripId := uuid.New()
	trip := NewTripScheduleModel(tripId, routeId, tod(12, 0), tod(12, 50), tod(13, 0), tod(13, 30))
	m := routeWithSchedule(t, routeId, []TripScheduleModel{trip})

	tr := m.Evaluate(time.Date(2026, 8, 15, 13, 10, 0, 0, time.UTC))
	want := time.Date(2026, 8, 15, 13, 0, 0, 0, time.UTC)
	if tr.State != InTransit || !tr.DepartedAt.Equal(want) {
		t.Fatalf("state=%v DepartedAt=%s, want InTransit at %s", tr.State, tr.DepartedAt, want)
	}
}
```

Add the two helpers to the same file if it does not already have equivalents —
`tod(h, m int) time.Time` returning `time.Date(0, 1, 1, h, m, 0, 0, time.UTC)`
(matching `timeOfDay`'s frame), and `routeWithSchedule(t, routeId, schedule)`
building a `Model` via `NewBuilder(...).SetId(routeId).SetSchedule(schedule).Build()`.
Read the top of `evaluate_test.go` before adding: reuse whatever it has.

- [ ] **Step 6: Run test to verify it fails**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./transport/ -run EvaluateReportsSelected -v`
Expected: FAIL — `Transition` has no field `TripId`.

- [ ] **Step 7: Widen `Transition` and populate it**

In `transport/model.go`, extend the struct and its doc comment:

```go
type Transition struct {
	State     RouteState
	NextState RouteState
	NextAt    time.Time
	// TripId names the trip Evaluate selected, and DepartedAt is that trip's
	// departure time-of-day materialized onto the calendar day it actually
	// departed — the previous day when a midnight-crossing trip is observed
	// after midnight. Together with the route id and the tenant they are the
	// inputs to VoyageId (design §7.1). Both are zero when State is
	// OutOfService, where there is no selected trip.
	TripId     uuid.UUID
	DepartedAt time.Time
}
```

Add, next to `materializeBoundary`:

```go
// materializeDeparture projects a trip's departure time-of-day onto the
// calendar day the trip actually departed, relative to now. For a same-day
// trip that is now's date. For a midnight-crossing trip observed after the
// crossing (now's time-of-day is before the departure time-of-day), the
// departure was yesterday — which is exactly the case that would otherwise
// make VOYAGE_ARRIVED derive a different voyage id than VOYAGE_DEPARTED did.
func materializeDeparture(now time.Time, departure time.Time) time.Time {
	at := time.Date(now.Year(), now.Month(), now.Day(),
		departure.Hour(), departure.Minute(), departure.Second(), departure.Nanosecond(),
		now.Location())
	if at.After(now) {
		at = at.Add(-24 * time.Hour)
	}
	return at
}
```

In `Evaluate`, change the local `to` closure so every branch carries the trip:

```go
	to := func(state RouteState, next RouteState, boundary time.Time) Transition {
		return Transition{
			State:      state,
			NextState:  next,
			NextAt:     materializeBoundary(now, boundary),
			TripId:     nextTrip.TripId(),
			DepartedAt: materializeDeparture(now, timeOfDay(nextTrip.Departure())),
		}
	}
```

Leave both `OutOfService` returns as bare `Transition{State: OutOfService}` —
there is no selected trip in that branch.

- [ ] **Step 8: Expose the transition from the state update**

Still in `transport/model.go`:

```go
// UpdateStateWithTransition is UpdateState plus the Transition the new state
// was derived from. processStateChange already computes and discards everything
// but State; callers that need the trip identity (voyage events) read it here
// rather than calling Evaluate a second time with a slightly different `now`.
func (m Model) UpdateStateWithTransition(now time.Time) (Model, bool, Transition, error) {
	tr := m.Evaluate(now)
	updated, err := m.Builder().SetState(tr.State).Build()
	if err != nil {
		return Model{}, false, Transition{}, err
	}
	return updated, m.State() != tr.State, tr, nil
}

func (m Model) UpdateState(now time.Time) (Model, bool, error) {
	updated, changed, _, err := m.UpdateStateWithTransition(now)
	return updated, changed, err
}
```

- [ ] **Step 9: Run tests to verify they pass**

Run: `cd services/atlas-transports/atlas.com/transports && go build ./... && go test ./...`
Expected: PASS, including the pre-existing `state_test.go` and
`evaluate_test.go` cases unmodified.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/transport/
git commit -m "feat(task-231): derive a durable voyage id and expose the selected trip on Transition"
```

---

## Task 6: Emit `VOYAGE_DEPARTED` / `VOYAGE_ARRIVED` per channel

**Module root:** `services/atlas-transports/atlas.com/transports`

Per design G1 the existing `ARRIVED`/`DEPARTED` emits are once per route
transition and stay exactly where they are; the new voyage events are emitted
**once per channel**, because `atlas-events` has no channel identity of its own.
`InTransit → AwaitingReturn` emits nothing today — that gap is the whole point
of the task (PRD F1).

### Files

- `services/atlas-transports/atlas.com/transports/kafka/message/transport/kafka.go` — two event type constants + `VoyageStatusEventBody`
- `services/atlas-transports/atlas.com/transports/transport/producer.go` — `VoyageDepartedStatusEventProvider`, `VoyageArrivedStatusEventProvider`
- `services/atlas-transports/atlas.com/transports/transport/processor.go:136-191` — per-channel emits in `UpdateRoute`
- `services/atlas-transports/atlas.com/transports/transport/processor_test.go` — extend
- `deploy/k8s/base/env-configmap.yaml` — read-only: `EVENT_TOPIC_TRANSPORT_STATUS` already exists; no new topic var

**Interfaces:**
- Consumes: `VoyageId`, `Transition.TripId`, `Transition.DepartedAt`, `UpdateStateWithTransition` (Task 5); `p.chanP.GetAll()` (`channel.Processor`).
- Produces on `EVENT_TOPIC_TRANSPORT_STATUS`:
  `transport.EventStatusVoyageDeparted = "VOYAGE_DEPARTED"`,
  `transport.EventStatusVoyageArrived = "VOYAGE_ARRIVED"`,
  `transport.VoyageStatusEventBody{VoyageId, WorldId, ChannelId, StagingMapId, EnRouteMapIds, DestinationMapId, ObservationMapId, DepartedAt}`.
  Consumed by Tasks 23 and 27.

- [ ] **Step 1: Write the failing test**

Add to `services/atlas-transports/atlas.com/transports/transport/processor_test.go`:

```go
// FR-V3/FR-V4: InTransit -> AwaitingReturn emits VOYAGE_ARRIVED, once per
// channel, with the full scope. Before this task that transition emitted
// nothing at all (PRD F1).
func TestArrivalEmitsVoyageArrivedPerChannel(t *testing.T) {
	p, mb, chans := newProcessorWithChannels(t, []channelSpec{{world: 1, channel: 1}, {world: 1, channel: 2}})
	_ = chans

	route := inTransitRouteAboutToArrive(t)
	if err := p.UpdateRoute(mb)(route); err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}

	msgs := voyageEvents(t, mb, transport.EventStatusVoyageArrived)
	if len(msgs) != 2 {
		t.Fatalf("expected one VOYAGE_ARRIVED per channel, got %d", len(msgs))
	}
	if msgs[0].Body.VoyageId == uuid.Nil {
		t.Fatalf("voyage id not populated")
	}
	if msgs[0].Body.VoyageId != msgs[1].Body.VoyageId {
		t.Fatalf("per-channel events must share one voyage id")
	}
	if msgs[0].Body.ChannelId == msgs[1].Body.ChannelId {
		t.Fatalf("expected distinct channels, both %d", msgs[0].Body.ChannelId)
	}
	if msgs[0].Body.DestinationMapId != route.DestinationMapId() {
		t.Fatalf("scope not carried")
	}
}

// FR-V6 / acceptance 20.4: the existing ARRIVED and DEPARTED emits are
// unchanged — same count, same single-emit shape, same body.
func TestDepartureStillEmitsOneDepartedAndOneVoyageDepartedPerChannel(t *testing.T) {
	p, mb, _ := newProcessorWithChannels(t, []channelSpec{{world: 1, channel: 1}, {world: 1, channel: 2}})

	route := openEntryRouteAboutToDepart(t)
	if err := p.UpdateRoute(mb)(route); err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}

	if got := countEvents(t, mb, transport.EventStatusDeparted); got != 1 {
		t.Fatalf("DEPARTED emitted %d times, want exactly 1 (unchanged)", got)
	}
	if got := len(voyageEvents(t, mb, transport.EventStatusVoyageDeparted)); got != 2 {
		t.Fatalf("VOYAGE_DEPARTED emitted %d times, want one per channel", got)
	}
}
```

Read `processor_test.go` first: reuse its existing processor construction, its
`message.Buffer` inspection helper and its channel-processor fake. Add only the
small helpers it lacks (`voyageEvents` decoding
`transport.StatusEvent[transport.VoyageStatusEventBody]` out of the buffer's
messages for a topic+type, `countEvents` counting by type).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./transport/ -run Voyage -v`
Expected: FAIL — `EventStatusVoyageArrived` undefined.

- [ ] **Step 3: Add the message contract**

In `kafka/message/transport/kafka.go`:

```go
const (
	EnvEventTopicStatus = "EVENT_TOPIC_TRANSPORT_STATUS"
	EventStatusArrived  = "ARRIVED"
	EventStatusDeparted = "DEPARTED"

	// Voyage lifecycle, distinct from the observation-deck visuals above.
	// ARRIVED/DEPARTED tell a watcher on the pier what the docked ship looks
	// like and carry only a map id; these name a concrete trip in a concrete
	// channel and carry its whole scope (FR-V3, FR-V4).
	EventStatusVoyageDeparted = "VOYAGE_DEPARTED"
	EventStatusVoyageArrived  = "VOYAGE_ARRIVED"
)

// VoyageStatusEventBody serves both voyage types; the envelope's Type
// discriminates. VOYAGE_ARRIVED carries DepartedAt too, so a consumer can
// compute voyage duration without holding the departure event.
type VoyageStatusEventBody struct {
	VoyageId         uuid.UUID  `json:"voyageId"`
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	StagingMapId     _map.Id    `json:"stagingMapId"`
	EnRouteMapIds    []_map.Id  `json:"enRouteMapIds"`
	DestinationMapId _map.Id    `json:"destinationMapId"`
	ObservationMapId _map.Id    `json:"observationMapId"`
	DepartedAt       time.Time  `json:"departedAt"`
}
```

Add the `time`, `channel` and `world` imports.

- [ ] **Step 4: Add the producers**

In `transport/producer.go`:

```go
// voyageStatusEventProvider keys on the voyage id so every event for one voyage
// lands on one partition — VOYAGE_ARRIVED can therefore never overtake the
// VOYAGE_DEPARTED of the same voyage.
func voyageStatusEventProvider(theType string, body transport.VoyageStatusEventBody, routeId uuid.UUID) model.Provider[[]kafka.Message] {
	value := transport.StatusEvent[transport.VoyageStatusEventBody]{
		RouteId: routeId,
		Type:    theType,
		Body:    body,
	}
	return producer.SingleMessageProvider([]byte(body.VoyageId.String()), value)
}

func VoyageDepartedStatusEventProvider(routeId uuid.UUID, body transport.VoyageStatusEventBody) model.Provider[[]kafka.Message] {
	return voyageStatusEventProvider(transport.EventStatusVoyageDeparted, body, routeId)
}

func VoyageArrivedStatusEventProvider(routeId uuid.UUID, body transport.VoyageStatusEventBody) model.Provider[[]kafka.Message] {
	return voyageStatusEventProvider(transport.EventStatusVoyageArrived, body, routeId)
}
```

- [ ] **Step 5: Emit per channel in `UpdateRoute`**

In `transport/processor.go`, switch the state derivation to the widened form and
add the emits. The existing single `ArrivedStatusEventProvider` /
`DepartedStatusEventProvider` calls and the existing warp loops stay untouched.

```go
		now := time.Now()
		r, changed, tr, err := route.UpdateStateWithTransition(now)
```

Add a helper on the processor, below `warpTo`:

```go
// emitVoyageEvent puts one voyage event per channel on mb. atlas-events has no
// channel identity of its own, so unlike the observation-deck ARRIVED/DEPARTED
// pair these must be per (world, channel) — the same fan-out shape the warp
// loops in this function already use (design G1).
func (p *ProcessorImpl) emitVoyageEvent(mb *message.Buffer) func(r Model, tr Transition, provider func(uuid.UUID, transport.VoyageStatusEventBody) model.Provider[[]kafka.Message]) error {
	return func(r Model, tr Transition, provider func(uuid.UUID, transport.VoyageStatusEventBody) model.Provider[[]kafka.Message]) error {
		t := tenant.MustFromContext(p.ctx)
		voyageId := VoyageId(t, r.Id(), tr.TripId, tr.DepartedAt)
		return model.ForEachSlice(model.FixedProvider(p.chanP.GetAll()), func(c channel2.Model) error {
			return mb.Put(transport.EnvEventTopicStatus, provider(r.Id(), transport.VoyageStatusEventBody{
				VoyageId:         voyageId,
				WorldId:          c.WorldId(),
				ChannelId:        c.Id(),
				StagingMapId:     r.StagingMapId(),
				EnRouteMapIds:    r.EnRouteMapIds(),
				DestinationMapId: r.DestinationMapId(),
				ObservationMapId: r.ObservationMapId(),
				DepartedAt:       tr.DepartedAt,
			}))
		}, model.ParallelExecute())
	}
}
```

In the `AwaitingReturn` branch, after the existing en-route warp loop:

```go
			if err = p.emitVoyageEvent(mb)(r, tr, VoyageArrivedStatusEventProvider); err != nil {
				p.l.WithError(err).Errorf("Error sending voyage arrival event for route [%s].", r.Id())
				return err
			}
```

In the `InTransit` branch, after the existing staging→en-route warp loop and
before the existing `DepartedStatusEventProvider` put:

```go
			if err = p.emitVoyageEvent(mb)(r, tr, VoyageDepartedStatusEventProvider); err != nil {
				p.l.WithError(err).Errorf("Error sending voyage departure event for route [%s].", r.Id())
				return err
			}
```

Import `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` and the kafka
`transport` message package alias already used in the file.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-transports/atlas.com/transports && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/
git commit -m "feat(task-231): emit VOYAGE_DEPARTED and VOYAGE_ARRIVED per channel"
```

---

## Task 7: Expose `voyageId` on the route REST resource

**Module root:** `services/atlas-transports/atlas.com/transports`

`atlas-events` uses this for FR-B5 step 1: the voyage is still underway iff the
route is `in_transit` **and** its `voyageId` equals the one on the work row.
Comparing state alone would be wrong — a route that has since departed on the
*next* trip is `in_transit` again but is a different voyage.

### Files

- `services/atlas-transports/atlas.com/transports/transport/rest.go` — nullable `voyageId` on `RestModel`, populated in `Transform`
- `services/atlas-transports/atlas.com/transports/transport/rest_test.go` — extend
- `services/atlas-transports/atlas.com/transports/transport/resource.go:25` — read-only; the route already returns `Transform(m)`

**Interfaces:**
- Consumes: `VoyageId`, `Transition` (Task 5).
- Produces: JSON field `voyageId` (empty string when not in transit) on
  `GET /transports/routes` and `/transports/routes/{routeId}`. Consumed by Task 22.

- [ ] **Step 1: Write the failing test**

Add to `services/atlas-transports/atlas.com/transports/transport/rest_test.go`:

```go
func TestTransformPopulatesVoyageIdOnlyWhenInTransit(t *testing.T) {
	routeId := uuid.New()
	tripId := uuid.New()
	trip := NewTripScheduleModel(tripId, routeId, tod(12, 0), tod(12, 50), tod(13, 0), tod(13, 30))
	m := routeWithSchedule(t, routeId, []TripScheduleModel{trip})

	ctx := tenant.WithContext(context.Background(), voyageTenant(t))
	now := time.Date(2026, 8, 15, 13, 10, 0, 0, time.UTC)

	rm, err := TransformAt(ctx, m, now)
	if err != nil {
		t.Fatalf("TransformAt: %v", err)
	}
	if rm.State != "in_transit" {
		t.Fatalf("state = %q", rm.State)
	}
	want := VoyageId(voyageTenant(t), routeId, tripId, time.Date(2026, 8, 15, 13, 0, 0, 0, time.UTC)).String()
	if rm.VoyageID != want {
		t.Fatalf("voyageId = %q, want %q", rm.VoyageID, want)
	}

	// Boarding window: not in transit, so no voyage.
	rm, err = TransformAt(ctx, m, time.Date(2026, 8, 15, 12, 10, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("TransformAt: %v", err)
	}
	if rm.VoyageID != "" {
		t.Fatalf("voyageId = %q, want empty outside transit", rm.VoyageID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./transport/ -run VoyageIdOnly -v`
Expected: FAIL — `RestModel` has no field `VoyageID`; `TransformAt` undefined.

- [ ] **Step 3: Implement**

Read `transport/rest.go`'s existing `Transform` first — it already derives
`NextTransitionAt` / `NextState`, so it already calls `Evaluate` (or receives a
transition). Add the field:

```go
	// VoyageID identifies the concrete trip currently under way. Empty unless
	// State is in_transit. A consumer holding a voyage id from VOYAGE_DEPARTED
	// tests "is my voyage still under way?" by comparing this for equality —
	// comparing State alone would read the NEXT trip's transit as its own
	// (design §7.4).
	VoyageID string `json:"voyageId,omitempty"`
```

Refactor `Transform` into `TransformAt(ctx context.Context, m Model, now time.Time) (RestModel, error)`
with `Transform(ctx, m)` delegating at `time.Now()`. Inside, after computing the
transition:

```go
	if tr.State == InTransit {
		rm.VoyageID = VoyageId(tenant.MustFromContext(ctx), m.Id(), tr.TripId, tr.DepartedAt).String()
	}
```

If the current `Transform` has no `ctx` parameter, thread one through from
`resource.go` (`d.Context()` on the handler) rather than reaching for a package
global — the tenant must come from the request context (FR-N13).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-transports/atlas.com/transports && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/transport/
git commit -m "feat(task-231): expose voyageId on the transport route resource"
```

---

# Phase C — `atlas-rates` (design §10.3)

## Task 8: Map the Anniversary stats to `exp` and `item_drop`

**Module root:** `services/atlas-rates/atlas.com/rates`

`EXP_BUFF_RATE` and `ITEM_UP_BY_ITEM` are chosen over `EVENT_RATE` because both
are allocated unconditionally on every version including the v83 baseline
(`libs/atlas-packet/model/character_temporary_stat.go:146`, `:168` — before the
gating that begins after `SoulStone` at `:184`), both use
`NoOpForeignValueWriter` so they are mask-only on the remote path, and neither
is in the movement-affecting set — whereas `EVENT_RATE` **is**, on the JMS
branch (`:799`). This is the `item_drop` table's first entry (FR-A10);
`rate.TypeItemDrop` is already consumed by the calculator.

### Files

- `services/atlas-rates/atlas.com/rates/kafka/message/buff/kafka.go:72-76` — two entries in `buffToRateMappings`
- `services/atlas-rates/atlas.com/rates/kafka/message/buff/kafka_test.go` — new file if absent
- `libs/atlas-constants/character/temporary_stat.go` — **read-only**; `TemporaryStatTypeExpBuffRate` (`:80`) and `TemporaryStatTypeItemUpByItem` (`:58`) already exist
- `services/atlas-rates/atlas.com/rates/rate/` — read-only; confirm the exact `item_drop` rate-type string the calculator consumes before writing the literal

**Interfaces:**
- Produces: `buff.GetRateMapping(character.TemporaryStatTypeExpBuffRate)` →
  `{RateType: "exp", Conversion: ConversionDirect}`;
  `buff.GetRateMapping(character.TemporaryStatTypeItemUpByItem)` →
  `{RateType: <the calculator's item-drop type>, Conversion: ConversionDirect}`.

- [ ] **Step 1: Confirm the rate-type string**

Run: `grep -rn "TypeItemDrop\|item_drop" services/atlas-rates/atlas.com/rates/rate/`
Use the literal that grep prints. If `rate.TypeItemDrop` is an exported constant,
reference the constant rather than re-typing its value. **Do not guess the
string** — a wrong value silently composes into a rate type nothing reads.

- [ ] **Step 2: Write the failing test**

Create `services/atlas-rates/atlas.com/rates/kafka/message/buff/kafka_test.go`:

```go
package buff

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// FR-A1/FR-A10: a configured multiplier of 2.0 is carried as amount = 200 and
// must read back as exactly 2.0x. ConversionDirect is amount/100.0, so the
// mapping is exactly invertible, which is what lets the multiplier live in
// configuration and still be displayed in the UI (FR-UI8).
func TestAnniversaryStatsConvertDirectly(t *testing.T) {
	for _, tc := range []struct {
		stat     character.TemporaryStatType
		rateType string
	}{
		{character.TemporaryStatTypeExpBuffRate, "exp"},
		{character.TemporaryStatTypeItemUpByItem, "item_drop"}, // replace with the constant from Step 1
	} {
		m, ok := GetRateMapping(tc.stat)
		if !ok {
			t.Fatalf("%s has no rate mapping", tc.stat)
		}
		if m.RateType != tc.rateType {
			t.Fatalf("%s -> %q, want %q", tc.stat, m.RateType, tc.rateType)
		}
		if m.Conversion != ConversionDirect {
			t.Fatalf("%s conversion = %v, want ConversionDirect", tc.stat, m.Conversion)
		}
		if got := CalculateMultiplier(200, m); got != 2.0 {
			t.Fatalf("%s: amount 200 -> %vx, want 2.0x", tc.stat, got)
		}
	}
}

// EVENT_RATE is deliberately NOT mapped: it is a member of the JMS
// movement-affecting stat set, so buffing it would interact with the client's
// movement filter for no gameplay benefit over EXP_BUFF_RATE (design §10.3).
func TestEventRateIsNotMapped(t *testing.T) {
	if IsRateStatType(character.TemporaryStatTypeEventRate) {
		t.Fatalf("EVENT_RATE must not be mapped — see design §10.3")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd services/atlas-rates/atlas.com/rates && go test ./kafka/message/buff/ -v`
Expected: FAIL — "EXP_BUFF_RATE has no rate mapping".

- [ ] **Step 4: Add the mappings**

In `kafka/message/buff/kafka.go`:

```go
var buffToRateMappings = map[character.TemporaryStatType]RateMapping{
	character.TemporaryStatTypeHolySymbol: {RateType: "exp", Conversion: ConversionAdditive},
	character.TemporaryStatTypeMesoUp:     {RateType: "meso", Conversion: ConversionDirect},
	character.TemporaryStatTypeCurse:      {RateType: "exp", Conversion: ConversionFixed, Multiplier: 0.5},
	// Anniversary (task-231). ConversionDirect matches MESO_UP's established
	// percent-of-base meaning and is exactly invertible, so amount = 200 reads
	// back as the configured 2.0x. EVENT_RATE is deliberately absent — it is in
	// the JMS movement-affecting set (design §10.3).
	character.TemporaryStatTypeExpBuffRate:  {RateType: "exp", Conversion: ConversionDirect},
	character.TemporaryStatTypeItemUpByItem: {RateType: "item_drop", Conversion: ConversionDirect},
}
```

Substitute the exact rate-type constant/string from Step 1.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-rates/atlas.com/rates && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-rates/atlas.com/rates/
git commit -m "feat(task-231): map EXP_BUFF_RATE and ITEM_UP_BY_ITEM into exp and item_drop"
```

---

# Phase D — `atlas-events` generic core (design §3–§6)

Verifiable end to end with **no** event implementation registered: definitions
can be created and listed, work rows can be scheduled and claimed, occurrences
can be written and queried. Tasks 21+ add behavior on top of it.

## Task 9: Bootstrap the module and the build registration

**Module root (all Phase D tasks unless stated):** `services/atlas-events/atlas.com/events`

### Files

- `services/atlas-events/atlas.com/events/go.mod` — new file
- `services/atlas-events/atlas.com/events/main.go` — new file
- `go.work` — add `./services/atlas-events/atlas.com/events` to `use()`
- `.github/config/services.json` — new entry in `services[]`
- `docker-bake.hcl` — add `"atlas-events"` to the hardcoded `go_services` list
- `Dockerfile` — read-only; parameterized by `ARG SERVICE`, no per-service edit needed (no new shared lib in this task)
- `services/atlas-party-quests/atlas.com/party-quests/{go.mod,main.go}` — read-only reference; copy their shape

**Interfaces:**
- Produces: a buildable, bakeable `atlas-events` binary that connects to
  Postgres, serves `/api/` with `/readyz` and `/debug/consumers`, and exits 0
  on `docker buildx bake atlas-events`.

- [ ] **Step 1: Create the module**

`services/atlas-events/atlas.com/events/go.mod`:

```
module atlas-events

go 1.25.5
```

Then add the same `require` + `replace` blocks party-quests has, dropping what
this service does not use. Generate rather than hand-write the versions:

```bash
cd services/atlas-events/atlas.com/events
go mod edit -replace github.com/Chronicle20/atlas/libs/atlas-constants=../../../../libs/atlas-constants
# ... one -replace per lib the service imports, mirroring party-quests' go.mod
```

- [ ] **Step 2: Write `main.go`**

Copy `services/atlas-party-quests/atlas.com/party-quests/main.go` and strip it to
the parts that exist now:

```go
package main

import (
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
	"gorm.io/gorm"
)

const serviceName = "atlas-events"

var consumerGroupId = consumergroup.Resolve("Events Service")

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string { return s.baseUrl }
func (s Server) GetPrefix() string  { return s.prefix }
func GetServer() Server             { return Server{baseUrl: "", prefix: "/api/"} }

func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	db := database.Connect(l, database.SetMigrations(func(db *gorm.DB) error {
		return db.AutoMigrate(&seeder.SeedState{})
	}))
	_ = db

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	_ = consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()
}
```

`consumergroup.Resolve`'s literal ("Events Service") is read by
`deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh` — Task 10 depends
on it being present here.

- [ ] **Step 3: Register in the build**

`go.work`: add `./services/atlas-events/atlas.com/events` to `use()`.

`.github/config/services.json`, in `services[]`:

```json
  {
    "name": "atlas-events",
    "type": "go-service",
    "path": "services/atlas-events",
    "module_path": "services/atlas-events/atlas.com/events",
    "docker_image": "ghcr.io/chronicle20/atlas-events/atlas-events",
    "docker_context": "."
  }
```

`docker-bake.hcl`: add `"atlas-events"` to the `go_services` list.
These two are hand-synced — adding to one does NOT add to the other
(`docs/adding-a-new-service.md` §1.2).

- [ ] **Step 4: Verify it builds and bakes**

Run:
```bash
cd services/atlas-events/atlas.com/events && go mod tidy && go build ./...
cd - && docker buildx bake atlas-events
```
Expected: both exit 0. The bake is not optional here — `go build` against
`go.work` cannot catch a missing `COPY libs/…` in the shared Dockerfile.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-events go.work .github/config/services.json docker-bake.hcl
git commit -m "feat(task-231): scaffold the atlas-events service and register it in the build"
```

---

## Task 10: Register the service in Kubernetes, ingress and the databases

**Module root:** n/a (deployment files only)

Four of the six PR-overlay pieces are **generator-owned** — hand edits are
silently reverted on the next generator run.

### Files

- `deploy/k8s/base/atlas-events.yaml` — new file (Deployment + Service; copy `deploy/k8s/base/atlas-party-quests.yaml`)
- `deploy/k8s/base/kustomization.yaml` — add to `resources:`
- `deploy/k8s/base/env-configmap.yaml` — add `EVENT_TOPIC_EVENT_VISUAL: "EVENT_TOPIC_EVENT_VISUAL"`
- `deploy/k8s/overlays/main/patches/db-name-suffix.yaml` — new document, `DB_NAME: "atlas-events-main"`
- `deploy/k8s/overlays/main/patches/atlas-env-env.yaml` — new document, `ATLAS_ENV: "main"`
- `deploy/k8s/overlays/main/kustomization.yaml` — `images:` entry + the new topic literal in `configMapGenerator`
- `deploy/k8s/overlays/pr/kustomization.yaml` — `ATLAS_DB_NAMES` gains `atlas-events`; `images:` entry
- `deploy/k8s/overlays/pr/patches/{db-name-suffix,consumer-group-env}.yaml` — **generator-owned**, re-run the scripts
- `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` — **generator-owned**, re-run the script
- `deploy/shared/routes.conf` — nginx block for `/api/events`
- `deploy/k8s/base/routes.conf.template.generated` — regenerated, not hand-edited
- `tools/db-bootstrap.sh` — add the unsuffixed `atlas-events` to `DBS`

**Interfaces:**
- Produces: `requests.RootUrl("EVENTS")` resolves through `BASE_SERVICE_URL` →
  nginx → `atlas-events:8080`, which is what Tasks 29 and 35 depend on.

- [ ] **Step 1: Base manifests**

Copy `deploy/k8s/base/atlas-party-quests.yaml` to
`deploy/k8s/base/atlas-events.yaml`. Set the container `name:` to `events`
(overlay patches match on it), `DB_NAME` to the **unsuffixed** `atlas-events`,
and no `namespace:`. Add `atlas-events.yaml` to `resources:` in
`deploy/k8s/base/kustomization.yaml`.

- [ ] **Step 2: New topic env var**

The only new Kafka topic this task introduces is the visual event topic. In
`deploy/k8s/base/env-configmap.yaml`, as an identity value:

```yaml
  EVENT_TOPIC_EVENT_VISUAL: "EVENT_TOPIC_EVENT_VISUAL"
```

Every other topic `atlas-events` touches (`EVENT_TOPIC_TRANSPORT_STATUS`,
`EVENT_TOPIC_MONSTER_STATUS`, `EVENT_TOPIC_CHARACTER_STATUS`,
`COMMAND_TOPIC_MONSTER`, `COMMAND_TOPIC_CHARACTER_BUFF`) already exists —
confirm with `grep -n "EVENT_TOPIC_MONSTER_STATUS" deploy/k8s/base/env-configmap.yaml`
rather than assuming.

- [ ] **Step 3: Main overlay**

Add a patch document to `deploy/k8s/overlays/main/patches/db-name-suffix.yaml`
setting `DB_NAME: "atlas-events-main"` on container `events`, and one to
`atlas-env-env.yaml` setting `ATLAS_ENV: "main"`. In
`deploy/k8s/overlays/main/kustomization.yaml` add:

```yaml
  - name: ghcr.io/chronicle20/atlas-events/atlas-events
    newTag: <current fleet tag, e.g. main-<sha>>
```

Confirm the tag exists first (`docker manifest inspect`). A missing `images:`
entry is a silent failure — the bump workflow's `yq` select writes nothing and
the service runs `:latest` forever. Then add
`EVENT_TOPIC_EVENT_VISUAL=EVENT_TOPIC_EVENT_VISUAL-main` to the
`configMapGenerator` literals: the generator uses `behavior: replace`, so any
base key not re-listed here is **absent** on main. Do **not** add
`KAFKA_CONSUMER_GROUP` to main.

- [ ] **Step 4: PR overlay (run the generators)**

Hand-edit only these two: add `atlas-events` to the `ATLAS_DB_NAMES` literal, and
add the same `images:` entry shape as Step 3. Then run, from the repo root:

```bash
deploy/k8s/overlays/pr/scripts/gen-topic-config.sh
deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh
deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh
```

Paste `gen-topic-config.sh`'s output into the atlas-env generator block as a
whole; do not hand-edit individual literals. `pr-validation.yml` regenerates
`atlas-pr-cleanup-env.example.yaml` and **hard-fails the PR** when the committed
copy is stale.

- [ ] **Step 5: Ingress**

Add to `deploy/shared/routes.conf`, alphabetically:

```nginx
location ~ ^/api/events(/.*)?$ {
  set $u "atlas-events:8080";
  ...
}
```

Copy the body of the adjacent `/api/party-quests` block verbatim. Then run
`tools/gen-routes.sh` and commit both files. Optionally run
`deploy/shared/test/routes_nginxt.sh` — it is operator-run, nothing in CI
invokes it.

- [ ] **Step 6: Databases**

Add the unsuffixed `atlas-events` to the `DBS` list in `tools/db-bootstrap.sh`.
PR envs need nothing beyond Step 4. Note in
`docs/tasks/task-231-generalized-events-service/context.md` that
`atlas-events-main` must be created **manually** on postgres.home (main has no
wave-0 create job) and that the GHCR package is created private on first push
and must be flipped to Public — both are out-of-repo, human steps.

- [ ] **Step 7: Verify**

Run:
```bash
kustomize build deploy/k8s/overlays/main >/dev/null
kustomize build deploy/k8s/overlays/pr >/dev/null
git diff --stat dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml
```
Expected: both builds exit 0; the cleanup-env diff is non-empty (it now names
atlas-events).

- [ ] **Step 8: Commit**

```bash
git add deploy dev tools/db-bootstrap.sh
git commit -m "chore(task-231): register atlas-events in k8s, ingress and the DB bootstrap"
```

---

## Task 11: The event handler registry — the only generic/specific seam

### Files

- `services/atlas-events/atlas.com/events/event/registry/handler.go` — new file
- `services/atlas-events/atlas.com/events/event/registry/registry.go` — new file
- `services/atlas-events/atlas.com/events/event/registry/registry_test.go` — new file

This task defines interfaces only; the concrete `occurrence`/`scheduling`/
`definition` model types it references are defined in Tasks 12–17. To keep the
dependency acyclic, `registry` declares its own minimal input/output structs
rather than importing those packages.

**Interfaces:**
- Produces:

```go
package registry

// Seed is a handler's request to create an occurrence. The generic layer owns
// how it becomes rows.
type Seed struct {
	Stage            string
	Context          json.RawMessage
	WorldId          world.Id
	ChannelId        channel.Id
	VoyageId         uuid.UUID   // uuid.Nil when the event has no voyage scope
	ConcurrencyKey   string
	Maps             []MapScope
	NextTransitionAt *time.Time
}

type MapScope struct {
	MapId  _map.Id
	Visual bool
}

// Progress is what a handler settles an occurrence into after Start/Advance.
// Terminal == true completes the occurrence with CompletionReason.
type Progress struct {
	Stage            string
	NextTransitionAt *time.Time
	Terminal         bool
	CompletionReason string
}

// Definition and Occurrence are the read-only views a handler receives.
type Definition struct {
	Id            uuid.UUID
	Type          string
	Name          string
	Enabled       bool
	Configuration json.RawMessage
}

type Occurrence struct {
	Id           uuid.UUID
	DefinitionId uuid.UUID
	Type         string
	Stage        string
	Context      json.RawMessage
	WorldId      world.Id
	ChannelId    channel.Id
	VoyageId     uuid.UUID
	StartedAt    time.Time
}

// Work is the due scheduled row that drove the call.
type Work struct {
	Id      uuid.UUID
	Type    string
	Context json.RawMessage
}

type Handler interface {
	Type() string
	ValidateConfiguration(raw json.RawMessage) error
	ConcurrencyKey(ctx context.Context, workContext json.RawMessage) (string, error)
	Evaluate(ctx context.Context, d Definition, w Work) (*Seed, error)
	Start(ctx context.Context, o Occurrence) (Progress, error)
	Advance(ctx context.Context, o Occurrence, w Work) (Progress, error)
	Complete(ctx context.Context, o Occurrence, reason string) error
}

func Register(h Handler)
func Get(theType string) (Handler, bool)
func Types() []string
```

- [ ] **Step 1: Write the failing test**

`services/atlas-events/atlas.com/events/event/registry/registry_test.go`:

```go
package registry

import (
	"context"
	"encoding/json"
	"testing"
)

type stubHandler struct{ t string }

func (h stubHandler) Type() string                                    { return h.t }
func (h stubHandler) ValidateConfiguration(json.RawMessage) error     { return nil }
func (h stubHandler) ConcurrencyKey(context.Context, json.RawMessage) (string, error) {
	return "", nil
}
func (h stubHandler) Evaluate(context.Context, Definition, Work) (*Seed, error) { return nil, nil }
func (h stubHandler) Start(context.Context, Occurrence) (Progress, error)       { return Progress{}, nil }
func (h stubHandler) Advance(context.Context, Occurrence, Work) (Progress, error) {
	return Progress{}, nil
}
func (h stubHandler) Complete(context.Context, Occurrence, string) error { return nil }

func TestRegisterAndGet(t *testing.T) {
	reset()
	Register(stubHandler{t: "TEST_EVENT"})

	h, ok := Get("TEST_EVENT")
	if !ok {
		t.Fatalf("registered handler not found")
	}
	if h.Type() != "TEST_EVENT" {
		t.Fatalf("Type = %q", h.Type())
	}
}

// An unregistered type is a FAILURE, not a fallback: a definition row whose
// type has no handler must make its work rows fail loudly rather than silently
// succeed with no occurrence (design §3.2).
func TestGetUnknownTypeReportsMissing(t *testing.T) {
	reset()
	if _, ok := Get("NOPE"); ok {
		t.Fatalf("Get returned ok for an unregistered type")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	reset()
	Register(stubHandler{t: "DUP"})
	defer func() {
		if recover() == nil {
			t.Fatalf("expected a panic on duplicate registration")
		}
	}()
	Register(stubHandler{t: "DUP"})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./event/registry/ -v`
Expected: FAIL — package does not compile.

- [ ] **Step 3: Write `handler.go`**

Exactly the interface block above, with a doc comment on each method restating
what it owns:

```go
// Package registry is the single seam between the generic event infrastructure
// and event-specific behavior (FR-X3). The generic layer reaches event behavior
// ONLY through Handler, resolved by type string. It must never switch on a type
// constant — Task 39's AST test enforces that.
//
// Domain reactions (a monster died, a voyage arrived, a character logged in) do
// NOT go through this interface. They are ordinary Kafka consumers owned by the
// event's own package and registered from that package's InitConsumers. This is
// the difference between "a registry mapping type to a handler" (allowed) and
// "a central dispatch table containing event logic" (forbidden).
package registry
```

Method comments, verbatim from design §3.2:

- `ValidateConfiguration` — rejects a definition whose configuration this handler
  cannot interpret (FR-D6); returns a field-scoped error.
- `ConcurrencyKey` — names the gameplay slot an occurrence would occupy. The
  generic layer enforces at most one non-terminal occurrence per
  `(tenant, definition, key)`. Empty string means unlimited.
- `Evaluate` — decides whether a `TRIGGER_EVALUATION` should produce an
  occurrence. Returning `(nil, nil)` is the ordinary "no occurrence" outcome
  (FR-B7, FR-B8), not an error.
- `Start` — orchestrates the side effects of a newly created occurrence (FR-B11).
- `Advance` — handles a due `OCCURRENCE_TRANSITION` row (FR-A14).
- `Complete` — cleanup for a terminal transition (FR-B18, FR-B19, FR-A15); must
  be idempotent (FR-B20).

- [ ] **Step 4: Write `registry.go`**

```go
package registry

import "sync"

var (
	mu       sync.RWMutex
	handlers = map[string]Handler{}
)

// Register makes h the handler for h.Type(). Called once per event package from
// main.go. Duplicate registration is a programming error, not a runtime
// condition, so it panics at startup rather than silently shadowing.
func Register(h Handler) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := handlers[h.Type()]; exists {
		panic("event registry: duplicate handler for type " + h.Type())
	}
	handlers[h.Type()] = h
}

// Get resolves a handler by definition type. A false second return is a
// FAILURE the caller must surface (the work row fails with
// "no handler for type X"), never a fallback to a default handler.
func Get(theType string) (Handler, bool) {
	mu.RLock()
	defer mu.RUnlock()
	h, ok := handlers[theType]
	return h, ok
}

// Types lists every registered type. Used by the definition REST layer to
// reject an unknown type at write time rather than at trigger time (FR-D6).
func Types() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(handlers))
	for k := range handlers {
		out = append(out, k)
	}
	return out
}

// reset clears the registry. Test-only; the production path registers once at
// startup.
func reset() {
	mu.Lock()
	defer mu.Unlock()
	handlers = map[string]Handler{}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./event/registry/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-events/atlas.com/events/event/registry/
git commit -m "feat(task-231): add the event handler registry, the generic/specific seam"
```

---

## Task 12: `event/definition` persistence

### Files

- `services/atlas-events/atlas.com/events/event/definition/model.go` — new
- `services/atlas-events/atlas.com/events/event/definition/builder.go` — new
- `services/atlas-events/atlas.com/events/event/definition/entity.go` — new
- `services/atlas-events/atlas.com/events/event/definition/provider.go` — new
- `services/atlas-events/atlas.com/events/event/definition/administrator.go` — new
- `services/atlas-events/atlas.com/events/event/definition/entity_test.go` — new
- `services/atlas-party-quests/atlas.com/party-quests/definition/{entity,provider,administrator}.go` — **read-only** reference implementations; copy their shape

**Interfaces:**
- Produces:
  - `definition.Model` with `Id() uuid.UUID`, `Type() string`, `Name() string`, `Enabled() bool`, `Configuration() json.RawMessage`, `CreatedAt()/UpdatedAt() time.Time`
  - `definition.NewBuilder(theType, name string) *Builder` with `SetId`, `SetEnabled`, `SetConfiguration`, `Build() (Model, error)`
  - `definition.Entity{ID, TenantID, Type, Name, Enabled, Configuration, CreatedAt, UpdatedAt}` on table `event_definition`, `definition.MigrateTable(db *gorm.DB) error`
  - `definition.Make(e Entity) (Model, error)`, `definition.ToEntity(m Model, tenantId uuid.UUID) (Entity, error)`
  - providers `getByIdProvider(id)`, `getAllPagedProvider(page)`, `getByTypeProvider(theType)`, `getEnabledByTypeProvider(theType)`
  - administrators `create(m)`, `setEnabled(id uuid.UUID, enabled bool)`

- [ ] **Step 1: Write the failing test**

`services/atlas-events/atlas.com/events/event/definition/entity_test.go`:

```go
package definition

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestModelRoundTripsThroughEntity(t *testing.T) {
	tenantId := uuid.New()
	cfg := json.RawMessage(`{"monsterCount":2}`)

	m, err := NewBuilder("CRIMSON_BALROG", "Crimson Balrog").
		SetId(uuid.New()).
		SetEnabled(false).
		SetConfiguration(cfg).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	e, err := ToEntity(m, tenantId)
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	if e.TenantID != tenantId {
		t.Fatalf("tenant not stamped")
	}

	back, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if back.Id() != m.Id() || back.Type() != "CRIMSON_BALROG" || back.Enabled() {
		t.Fatalf("round trip lost fields: %+v", back)
	}
	if string(back.Configuration()) != string(cfg) {
		t.Fatalf("configuration mangled: %s", back.Configuration())
	}
}

// FR-D2: configuration is opaque to this package. It must survive round trip
// byte-for-byte, including a shape this package has never seen.
func TestConfigurationIsOpaque(t *testing.T) {
	cfg := json.RawMessage(`{"unknownToUs":[1,2,{"deep":true}]}`)
	m, err := NewBuilder("SOMETHING_NEW", "n").SetConfiguration(cfg).Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	e, err := ToEntity(m, uuid.New())
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	back, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if string(back.Configuration()) != string(cfg) {
		t.Fatalf("opaque configuration changed: %s", back.Configuration())
	}
}

func TestBuildRejectsEmptyType(t *testing.T) {
	if _, err := NewBuilder("", "n").Build(); err == nil {
		t.Fatalf("expected an error for an empty type")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./event/definition/ -v`
Expected: FAIL — package does not compile.

- [ ] **Step 3: Write `model.go` and `builder.go`**

Immutable struct with unexported fields and value-receiver accessors; a
`*Builder` with `SetX` returning `*Builder` and `Build() (Model, error)`
validating that `type` and `name` are non-empty and `configuration` is valid
JSON (`json.Valid`). `Model.Builder()` returns a builder seeded from the model,
matching `transport.Model.Builder()`'s shape.

- [ ] **Step 4: Write `entity.go`**

```go
type Entity struct {
	ID            uuid.UUID `gorm:"primaryKey;column:id;type:uuid"`
	TenantID      uuid.UUID `gorm:"column:tenant_id;type:uuid;not null"`
	Type          string    `gorm:"column:type;not null;index:ix_def_tenant_type,priority:2"`
	Name          string    `gorm:"column:name;not null"`
	Enabled       bool      `gorm:"column:enabled;not null;default:false"`
	Configuration string    `gorm:"column:configuration;type:jsonb;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
}

func (Entity) TableName() string { return "event_definition" }

func MigrateTable(db *gorm.DB) error { return db.AutoMigrate(&Entity{}) }
```

Unlike party-quests' `definition.Entity`, the configuration is stored as its own
`jsonb` column rather than the whole RestModel — the generic layer must be able
to hand `Configuration` to a handler without knowing its shape (FR-D2), and the
scalar columns must be indexable (FR-API7).

- [ ] **Step 5: Write `provider.go` and `administrator.go`**

Copy the shape of party-quests'. Providers return
`database.EntityProvider[T]` / `func(db *gorm.DB) model.Provider[Entity]`;
administrators take `db *gorm.DB` and return the written entity. `setEnabled`
must update `updated_at` in the same statement.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./event/definition/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-events/atlas.com/events/event/definition/
git commit -m "feat(task-231): add event definition persistence"
```

---

## Task 13: `event/definition` processor, REST and seeding

### Files

- `services/atlas-events/atlas.com/events/event/definition/processor.go` — new
- `services/atlas-events/atlas.com/events/event/definition/rest.go` — new
- `services/atlas-events/atlas.com/events/event/definition/resource.go` — new
- `services/atlas-events/atlas.com/events/event/definition/subdomain.go` — new
- `services/atlas-events/atlas.com/events/event/definition/processor_test.go` — new
- `services/atlas-events/atlas.com/events/event/definition/resource_test.go` — new
- `services/atlas-configurations/seed-data/` — locate the seed root with `git ls-files | grep -i seed-data | head`; add `events/definitions/event-crimson-balrog.json` and `event-anniversary.json` under the same layout party-quests uses
- `services/atlas-party-quests/atlas.com/party-quests/definition/{processor,rest,resource,subdomain}.go` — **read-only** reference
- `docs/rest-pagination.md` — **read-only**; the pagination contract for FR-API1

**Interfaces:**
- Consumes: `registry.Get`, `registry.Types` (Task 11); definition persistence (Task 12).
- Produces:
  - `definition.Processor` interface with `GetById(id uuid.UUID) (Model, error)`,
    `GetAllPaged(page model.Page) (model.Paged[Model], error)`,
    `GetEnabledByType(theType string) ([]Model, error)`,
    `SetEnabled(id uuid.UUID, enabled bool) (Model, error)`,
    `Create(m Model) (Model, error)`
  - `definition.NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`
  - REST: `GET /events/definitions`, `GET /events/definitions/{definitionId}`,
    `PATCH /events/definitions/{definitionId}`; resource type name `event-definitions`
  - `definition.InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer`
  - `definition.DefinitionSubdomain` implementing `seeder.Subdomain[RestModel, Model]`

- [ ] **Step 1: Write the failing test**

`services/atlas-events/atlas.com/events/event/definition/processor_test.go`:

```go
// FR-D6: an invalid configuration is rejected on WRITE, by the handler that
// owns the type, rather than persisted and failing later at trigger time.
func TestCreateRejectsConfigurationTheHandlerRefuses(t *testing.T) {
	registryReset(t)
	registry.Register(rejectingHandler{t: "PICKY", err: errors.New("monsterCount must be > 0")})

	db := newTestDB(t)
	m, _ := NewBuilder("PICKY", "n").SetConfiguration(json.RawMessage(`{"monsterCount":0}`)).Build()

	if _, err := NewProcessor(testLogger(t), testCtx(t), db).Create(m); err == nil {
		t.Fatalf("expected the handler's validation error to be surfaced")
	}
	var count int64
	db.Model(&Entity{}).Count(&count)
	if count != 0 {
		t.Fatalf("invalid definition was persisted anyway")
	}
}

// A type with no registered handler is rejected at write time, not silently
// stored to fail at trigger time.
func TestCreateRejectsUnknownType(t *testing.T) {
	registryReset(t)
	db := newTestDB(t)
	m, _ := NewBuilder("NO_HANDLER", "n").SetConfiguration(json.RawMessage(`{}`)).Build()

	if _, err := NewProcessor(testLogger(t), testCtx(t), db).Create(m); err == nil {
		t.Fatalf("expected an error for a type with no handler")
	}
}

// FR-D5: disabling a definition must not touch occurrences. This processor has
// no path that writes an occurrence at all — the assertion is that SetEnabled
// returns the updated model and nothing else changes.
func TestSetEnabledTogglesOnly(t *testing.T) {
	registryReset(t)
	registry.Register(acceptingHandler{t: "OK"})
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)

	m, _ := NewBuilder("OK", "n").SetConfiguration(json.RawMessage(`{}`)).Build()
	created, err := p.Create(m)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Enabled() {
		t.Fatalf("definitions must be created disabled unless asked otherwise")
	}

	updated, err := p.SetEnabled(created.Id(), true)
	if err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if !updated.Enabled() || updated.Id() != created.Id() || updated.Type() != created.Type() {
		t.Fatalf("SetEnabled changed more than enabled: %+v", updated)
	}
}
```

Build `newTestDB` on the in-memory/sqlite or dockertest harness the repo already
uses for GORM tests — find it with
`grep -rln "AutoMigrate" --include='*_test.go' services/ | head` and copy that
service's setup. `registryReset` calls the exported test seam you add in Step 3.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./event/definition/ -v`
Expected: FAIL — `NewProcessor` undefined.

- [ ] **Step 3: Expose a registry reset for tests**

`registry.reset` is unexported and in another package. Add to
`event/registry/registry.go`:

```go
// ResetForTest clears the registry. Production registers once at startup; this
// exists so a package that resolves handlers can test both the found and
// not-found branches.
func ResetForTest() { reset() }
```

- [ ] **Step 4: Write `processor.go`**

`Create` resolves `registry.Get(m.Type())`; a missing handler is an error, and
`h.ValidateConfiguration(m.Configuration())` gates the insert. `SetEnabled`
writes through the administrator and returns the re-read model. `GetAllPaged`
delegates to `getAllPagedProvider`. No method switches on a type constant.

- [ ] **Step 5: Write `rest.go` and `resource.go`**

`RestModel` exposes `id`, `type`, `name`, `enabled`, `configuration`,
`createdAt`, `updatedAt` (FR-API3), plus:

```go
	// SingleOccurrence tells the UI how to render this row (FR-UI4) without
	// switching on type: where the handler's concurrency key is a constant, at
	// most one occurrence can exist and the row may show live occurrence state;
	// where it varies, the row must link to the filtered occurrence list
	// instead of implying a single state.
	SingleOccurrence bool `json:"singleOccurrence"`
```

Derive `SingleOccurrence` by calling the handler's `ConcurrencyKey` twice with
two different work contexts and comparing: equal ⇒ constant ⇒ true. Keep that
derivation in the processor, not the REST layer.

Routes, registered exactly like party-quests' `InitResource`:

```
GET   /events/definitions            (paginated per docs/rest-pagination.md; filter[type], filter[enabled])
GET   /events/definitions/{definitionId}
PATCH /events/definitions/{definitionId}   (enabled only; FR-API2)
```

`PATCH` accepts only `enabled`; any other attribute is rejected with a JSON:API
error rather than silently ignored.

- [ ] **Step 6: Write `subdomain.go` and the seed files**

Copy `services/atlas-party-quests/atlas.com/party-quests/definition/subdomain.go`
line for line, changing:

```go
func (DefinitionSubdomain) Name() string { return "definition.event" }
func (DefinitionSubdomain) Path() string { return "events/definitions" }
func (DefinitionSubdomain) Type() string { return "event-definition" }
func (DefinitionSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^event-(.+)\.json$`)
}
```

Both seed files ship `"enabled": false` (FR-D7). Their `configuration` bodies are
written in Tasks 21 (Crimson Balrog) and 33 (Anniversary) — create the two files
here with the envelope and an empty `{}` configuration, and fill them in those
tasks. Register `definition.InitSeedResource` in `main.go` in Task 20.

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./event/definition/ -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-events/atlas.com/events/event/ services/atlas-configurations/seed-data/
git commit -m "feat(task-231): add event definition processor, REST surface and seeding"
```

---

## Task 14: `event/transition` — occurrence history

### Files

- `services/atlas-events/atlas.com/events/event/transition/model.go` — new
- `services/atlas-events/atlas.com/events/event/transition/builder.go` — new
- `services/atlas-events/atlas.com/events/event/transition/entity.go` — new
- `services/atlas-events/atlas.com/events/event/transition/provider.go` — new
- `services/atlas-events/atlas.com/events/event/transition/rest.go` — new
- `services/atlas-events/atlas.com/events/event/transition/entity_test.go` — new

There is **no** `administrator.go` here on purpose: a transition row is only ever
written inside the occurrence administrator's paired transaction (Task 15), so
this package deliberately exposes no standalone write path.

**Interfaces:**
- Produces:
  - `transition.Model` with `Id()`, `OccurrenceId()`, `FromStage() string`, `ToStage() string`, `OccurredAt() time.Time`, `TriggerType() string`, `TriggerReference() string`
  - `transition.Entity` on table `event_occurrence_transition`, `transition.MigrateTable`
  - `transition.Make`, `transition.ToEntity(m Model, tenantId uuid.UUID) (Entity, error)`
  - `transition.ByOccurrenceProvider(occurrenceId uuid.UUID) func(db *gorm.DB) model.Provider[[]Entity]` ordered by `occurred_at ASC`
  - `transition.RestModel` (`GetName() == "event-occurrence-transitions"`)
  - Trigger type constants: `TriggerTypeOccurrenceCreated = "OCCURRENCE_CREATED"`, `TriggerTypeOccurrenceStart = "OCCURRENCE_START"`, `TriggerTypeScheduledWork = "SCHEDULED_WORK"`, `TriggerTypeMonsterKilled = "MONSTER_KILLED"`, `TriggerTypeVoyageArrived = "VOYAGE_ARRIVED"`

- [ ] **Step 1: Write the failing test**

```go
package transition

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransitionRoundTrip(t *testing.T) {
	occId := uuid.New()
	at := time.Date(2026, 8, 15, 12, 5, 3, 0, time.UTC)

	m, err := NewBuilder(occId, "ATTACKING").
		SetFromStage("").
		SetOccurredAt(at).
		SetTrigger(TriggerTypeScheduledWork, "work-1").
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	e, err := ToEntity(m, uuid.New())
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	back, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if back.OccurrenceId() != occId || back.ToStage() != "ATTACKING" ||
		back.TriggerType() != TriggerTypeScheduledWork || back.TriggerReference() != "work-1" {
		t.Fatalf("round trip lost fields: %+v", back)
	}
	if !back.OccurredAt().Equal(at) {
		t.Fatalf("occurredAt = %s, want %s", back.OccurredAt(), at)
	}
}

// FR-T1: fromStage is nullable — the creation row has no prior stage.
func TestCreationTransitionHasNoFromStage(t *testing.T) {
	m, err := NewBuilder(uuid.New(), "ACTIVE").
		SetTrigger(TriggerTypeOccurrenceCreated, "work-1").Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.FromStage() != "" {
		t.Fatalf("FromStage = %q, want empty", m.FromStage())
	}
}

// FR-T3: a transition without a trigger type is not traceable, so it is not
// constructible.
func TestBuildRequiresTriggerType(t *testing.T) {
	if _, err := NewBuilder(uuid.New(), "ACTIVE").Build(); err == nil {
		t.Fatalf("expected an error when no trigger type was set")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./event/transition/ -v`
Expected: FAIL — package does not compile.

- [ ] **Step 3: Implement**

`Entity`:

```go
type Entity struct {
	ID               uuid.UUID `gorm:"primaryKey;column:id;type:uuid"`
	TenantID         uuid.UUID `gorm:"column:tenant_id;type:uuid;not null"`
	OccurrenceID     uuid.UUID `gorm:"column:occurrence_id;type:uuid;not null;index:ix_trans_occurrence"`
	FromStage        string    `gorm:"column:from_stage"`
	ToStage          string    `gorm:"column:to_stage;not null"`
	OccurredAt       time.Time `gorm:"column:occurred_at;not null"`
	TriggerType      string    `gorm:"column:trigger_type;not null"`
	TriggerReference string    `gorm:"column:trigger_reference"`
}

func (Entity) TableName() string { return "event_occurrence_transition" }
```

`NewBuilder(occurrenceId uuid.UUID, toStage string) *Builder` defaults
`OccurredAt` to `time.Now()` and requires `SetTrigger` before `Build`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./event/transition/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-events/atlas.com/events/event/transition/
git commit -m "feat(task-231): add occurrence transition history persistence"
```

---

## Task 15: `event/occurrence` persistence, child tables and the paired write

The two structural rules of this package:

1. **Every state or stage change writes the occurrence row and a transition row
   in one transaction.** The administrator exposes only the paired operation, so
   FR-O6 / FR-T2 hold structurally rather than by convention.
2. **Completion is a guarded UPDATE**, not a lock:
   `… WHERE id = ? AND state = 'ACTIVE'`. `RowsAffected == 0` means someone else
   completed it first — the caller then skips cleanup and returns success. That
   is FR-B20's "racing paths produce exactly one completion" as a database
   predicate.

### Files

- `services/atlas-events/atlas.com/events/event/occurrence/model.go` — new
- `services/atlas-events/atlas.com/events/event/occurrence/builder.go` — new
- `services/atlas-events/atlas.com/events/event/occurrence/entity.go` — new (three entities: occurrence, map scope, monster)
- `services/atlas-events/atlas.com/events/event/occurrence/provider.go` — new
- `services/atlas-events/atlas.com/events/event/occurrence/administrator.go` — new
- `services/atlas-events/atlas.com/events/event/occurrence/processor.go` — new
- `services/atlas-events/atlas.com/events/event/occurrence/administrator_test.go` — new
- `services/atlas-events/atlas.com/events/event/occurrence/processor_test.go` — new

**Interfaces:**
- Consumes: `transition.Entity`, `transition.ToEntity`, trigger constants (Task 14); `registry.Seed`, `registry.MapScope`, `registry.Progress` (Task 11).
- Produces:
  - State constants `StateActive = "ACTIVE"`, `StateCompleted = "COMPLETED"`, `StateCancelled = "CANCELLED"`, `StateFailed = "FAILED"`
  - `occurrence.Model` with `Id()`, `DefinitionId()`, `Type()`, `State()`, `Stage()`, `Context() json.RawMessage`, `WorldId() world.Id`, `ChannelId() channel.Id`, `VoyageId() uuid.UUID`, `ConcurrencyKey() string`, `StartedAt()`, `NextTransitionAt() *time.Time`, `CompletedAt() *time.Time`, `CompletionReason() string`
  - `occurrence.Processor` with:
    - `CreateFromSeed(d definition.Model, s registry.Seed, triggerRef string) (Model, error)` — inserts occurrence + map rows + the `OCCURRENCE_CREATED` transition in one transaction; returns `ErrConcurrencyKeyTaken` on unique violation
    - `ApplyProgress(o Model, p registry.Progress, triggerType, triggerRef string) (Model, error)`
    - `Complete(id uuid.UUID, reason string, triggerType, triggerRef string) (bool, error)` — the guarded update; first return is "we completed it"
    - `GetById(id uuid.UUID) (Model, error)`
    - `GetActiveByType(theType string) ([]Model, error)`
    - `GetActiveByVoyage(voyageId uuid.UUID, worldId world.Id, channelId channel.Id) ([]Model, error)`
    - `VisualsInMap(worldId world.Id, channelId channel.Id, mapId _map.Id) ([]Model, error)`
    - `ObserveMonsterSpawned(occurrenceId uuid.UUID, uniqueId uint32, monsterId uint32) error`
    - `ObserveMonsterGone(occurrenceId uuid.UUID, uniqueId uint32, monsterId uint32) error`
    - `MonsterTally(occurrenceId uuid.UUID) (total int, alive int, err error)`
  - `occurrence.ErrConcurrencyKeyTaken`

- [ ] **Step 1: Write the failing test**

`services/atlas-events/atlas.com/events/event/occurrence/administrator_test.go`:

```go
// FR-O6/FR-T2: there is no path that writes one without the other.
func TestCreateWritesOccurrenceAndTransitionTogether(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)

	o, err := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"), registry.Seed{
		Stage:          "ATTACKING",
		Context:        json.RawMessage(`{"routeId":"r"}`),
		WorldId:        1,
		ChannelId:      4,
		ConcurrencyKey: "v1|1|4",
		Maps: []registry.MapScope{
			{MapId: 200090010, Visual: true},
			{MapId: 200090011, Visual: false},
		},
	}, "work-1")
	if err != nil {
		t.Fatalf("CreateFromSeed: %v", err)
	}
	if o.State() != StateActive || o.Stage() != "ATTACKING" {
		t.Fatalf("occurrence = %s/%s", o.State(), o.Stage())
	}

	var trans int64
	db.Model(&transition.Entity{}).Where("occurrence_id = ?", o.Id()).Count(&trans)
	if trans != 1 {
		t.Fatalf("expected 1 transition row, got %d", trans)
	}
	var maps int64
	db.Model(&MapEntity{}).Where("occurrence_id = ?", o.Id()).Count(&maps)
	if maps != 2 {
		t.Fatalf("expected 2 map rows, got %d", maps)
	}
}

// design §5.3 guard 3: even if dedup and the work-row state machine both fail,
// the concurrency key makes the SECOND occurrence insert fail rather than
// producing two live attacks on one voyage.
func TestConcurrencyKeyRejectsASecondActiveOccurrence(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	d := testDefinition(t, "CRIMSON_BALROG")
	seed := registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "v1|1|4", WorldId: 1, ChannelId: 4}

	if _, err := p.CreateFromSeed(d, seed, "work-1"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := p.CreateFromSeed(d, seed, "work-2")
	if !errors.Is(err, ErrConcurrencyKeyTaken) {
		t.Fatalf("second create err = %v, want ErrConcurrencyKeyTaken", err)
	}
}

// An empty concurrency key opts out of the constraint entirely.
func TestEmptyConcurrencyKeyAllowsMany(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	d := testDefinition(t, "UNBOUNDED")
	seed := registry.Seed{Stage: "", ConcurrencyKey: ""}

	for i := 0; i < 3; i++ {
		if _, err := p.CreateFromSeed(d, seed, "work"); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
}

// FR-B20: two racing completion paths produce exactly one completion and one
// reason. The loser is told it lost, and must NOT run cleanup again.
func TestCompleteIsWonByExactlyOneCaller(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	o, err := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"),
		registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "k"}, "w")
	if err != nil {
		t.Fatalf("CreateFromSeed: %v", err)
	}

	wonA, err := p.Complete(o.Id(), "MONSTERS_ELIMINATED", transition.TriggerTypeMonsterKilled, "u1")
	if err != nil {
		t.Fatalf("Complete A: %v", err)
	}
	wonB, err := p.Complete(o.Id(), "VESSEL_ARRIVED", transition.TriggerTypeVoyageArrived, "v1")
	if err != nil {
		t.Fatalf("Complete B: %v", err)
	}
	if !wonA || wonB {
		t.Fatalf("wonA=%v wonB=%v, want true/false", wonA, wonB)
	}

	final, err := p.GetById(o.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if final.State() != StateCompleted || final.CompletionReason() != "MONSTERS_ELIMINATED" {
		t.Fatalf("final = %s/%s", final.State(), final.CompletionReason())
	}
}

// design §9.5: set semantics, not a counter. A redelivered KILLED must not
// double-decrement, and a KILLED arriving BEFORE its CREATED must not be
// resurrected by the later CREATED.
func TestMonsterTallyIsIdempotentAndOrderIndependent(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	o, _ := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"),
		registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "k"}, "w")

	if err := p.ObserveMonsterSpawned(o.Id(), 1, 8150000); err != nil {
		t.Fatalf("spawned: %v", err)
	}
	// KILLED before CREATED for unique id 2.
	if err := p.ObserveMonsterGone(o.Id(), 2, 8150000); err != nil {
		t.Fatalf("gone: %v", err)
	}
	if err := p.ObserveMonsterSpawned(o.Id(), 2, 8150000); err != nil {
		t.Fatalf("late spawned: %v", err)
	}

	total, alive, err := p.MonsterTally(o.Id())
	if err != nil {
		t.Fatalf("MonsterTally: %v", err)
	}
	if total != 2 || alive != 1 {
		t.Fatalf("total=%d alive=%d, want 2/1 (the late CREATED must not resurrect)", total, alive)
	}

	// Redelivery of both events changes nothing.
	_ = p.ObserveMonsterGone(o.Id(), 2, 8150000)
	_ = p.ObserveMonsterSpawned(o.Id(), 1, 8150000)
	total, alive, _ = p.MonsterTally(o.Id())
	if total != 2 || alive != 1 {
		t.Fatalf("after redelivery total=%d alive=%d, want 2/1", total, alive)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./event/occurrence/ -v`
Expected: FAIL — package does not compile.

- [ ] **Step 3: Write `entity.go`**

```go
type Entity struct {
	ID                uuid.UUID  `gorm:"primaryKey;column:id;type:uuid"`
	TenantID          uuid.UUID  `gorm:"column:tenant_id;type:uuid;not null"`
	EventDefinitionID uuid.UUID  `gorm:"column:event_definition_id;type:uuid;not null"`
	Type              string     `gorm:"column:type;not null"`
	State             string     `gorm:"column:state;not null"`
	Stage             string     `gorm:"column:stage"`
	Context           string     `gorm:"column:context;type:jsonb;not null;default:'{}'"`
	// worldId, channelId and voyageId are PROMOTED out of context to scalar
	// columns so the gameplay queries of FR-API7 are index-served rather than
	// jsonb scans (FR-API8, FR-O7).
	WorldID          *uint8     `gorm:"column:world_id"`
	ChannelID        *uint8     `gorm:"column:channel_id"`
	VoyageID         *uuid.UUID `gorm:"column:voyage_id;type:uuid"`
	ConcurrencyKey   string     `gorm:"column:concurrency_key;not null;default:''"`
	StartedAt        time.Time  `gorm:"column:started_at;not null"`
	NextTransitionAt *time.Time `gorm:"column:next_transition_at"`
	CompletedAt      *time.Time `gorm:"column:completed_at"`
	CompletionReason string     `gorm:"column:completion_reason"`
}

func (Entity) TableName() string { return "event_occurrence" }

// MapEntity is the FR-API8 child table. Visual distinguishes the deck (gets the
// enemy-ship visual) from the cabin (counts toward "aboard", gets nothing) at
// QUERY time, so FR-B13/FR-B14 is a predicate rather than a branch in
// atlas-channel.
type MapEntity struct {
	OccurrenceID uuid.UUID `gorm:"primaryKey;column:occurrence_id;type:uuid"`
	MapID        uint32    `gorm:"primaryKey;column:map_id"`
	Visual       bool      `gorm:"column:visual;not null;default:false"`
}

func (MapEntity) TableName() string { return "event_occurrence_map" }

// MonsterEntity is the durable form of "are any left?" (FR-B18). A SET, not a
// counter: a counter cannot be made idempotent under Kafka redelivery, and
// cannot tolerate KILLED arriving before CREATED (design §9.5, §14 A4).
type MonsterEntity struct {
	OccurrenceID uuid.UUID `gorm:"primaryKey;column:occurrence_id;type:uuid"`
	UniqueID     uint32    `gorm:"primaryKey;column:unique_id"`
	MonsterID    uint32    `gorm:"column:monster_id;not null"`
	Alive        bool      `gorm:"column:alive;not null;default:true"`
	ObservedAt   time.Time `gorm:"column:observed_at;not null"`
}

func (MonsterEntity) TableName() string { return "event_occurrence_monster" }

func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{}, &MapEntity{}, &MonsterEntity{})
}
```

- [ ] **Step 4: Write the administrator's paired transaction**

```go
// createFromSeed inserts the occurrence, its map scope rows and the
// OCCURRENCE_CREATED transition in ONE transaction. The administrator exposes
// no way to write an occurrence without its transition (FR-O6).
func createFromSeed(...) database.EntityProvider[Entity] { ... }
```

Detect the concurrency-key unique violation and return
`ErrConcurrencyKeyTaken` rather than the raw driver error — the caller treats
it as "someone else already created it" and completes the work row
successfully.

```go
// complete is a GUARDED update, not a lock. RowsAffected == 0 means another
// path completed this occurrence first; the caller must then skip its cleanup
// and return success. This is FR-B20 expressed as a database predicate.
func complete(id uuid.UUID, reason string, at time.Time) func(db *gorm.DB) (bool, error) {
	return func(db *gorm.DB) (bool, error) {
		res := db.Model(&Entity{}).
			Where("id = ? AND state = ?", id, StateActive).
			Updates(map[string]any{
				"state":             StateCompleted,
				"completion_reason": reason,
				"completed_at":      at,
			})
		return res.RowsAffected == 1, res.Error
	}
}
```

- [ ] **Step 5: Write the monster-set observers**

```go
// ObserveMonsterSpawned is INSERT-IF-ABSENT, deliberately not an upsert: a
// KILLED that arrived before its CREATED already wrote a dead row, and the late
// CREATED must not resurrect it. The two events share a topic but have no
// ordering guarantee across partitions, so this is a real case (design §9.5).
func (p *ProcessorImpl) ObserveMonsterSpawned(occurrenceId uuid.UUID, uniqueId uint32, monsterId uint32) error {
	return p.db.WithContext(p.ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&MonsterEntity{
			OccurrenceID: occurrenceId, UniqueID: uniqueId, MonsterID: monsterId,
			Alive: true, ObservedAt: time.Now(),
		}).Error
}

// ObserveMonsterGone is an UPSERT to alive=false: idempotent by construction,
// and correct whether or not CREATED was seen first.
func (p *ProcessorImpl) ObserveMonsterGone(occurrenceId uuid.UUID, uniqueId uint32, monsterId uint32) error {
	return p.db.WithContext(p.ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "occurrence_id"}, {Name: "unique_id"}},
		DoUpdates: clause.Assignments(map[string]any{"alive": false, "observed_at": time.Now()}),
	}).Create(&MonsterEntity{
		OccurrenceID: occurrenceId, UniqueID: uniqueId, MonsterID: monsterId,
		Alive: false, ObservedAt: time.Now(),
	}).Error
}
```

- [ ] **Step 6: Write `VisualsInMap`**

```sql
SELECT o.* FROM event_occurrence o
JOIN event_occurrence_map m ON m.occurrence_id = o.id
WHERE m.map_id = ? AND m.visual
  AND o.tenant_id = ? AND o.world_id = ? AND o.channel_id = ?
  AND o.state = 'ACTIVE';
```

Express it with GORM's `Joins`. Do not filter on `type` here — the channel asks
"what visuals are active in this map", which is a generic question; filtering on
`CRIMSON_BALROG` would be the generic layer naming an event type.

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./event/occurrence/ -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-events/atlas.com/events/event/occurrence/
git commit -m "feat(task-231): add occurrence persistence with paired transitions and a guarded completion"
```

---

## Task 16: `event/occurrence` REST surface

### Files

- `services/atlas-events/atlas.com/events/event/occurrence/rest.go` — new
- `services/atlas-events/atlas.com/events/event/occurrence/resource.go` — new
- `services/atlas-events/atlas.com/events/event/occurrence/visual_rest.go` — new (the narrow map-entry projection)
- `services/atlas-events/atlas.com/events/event/occurrence/resource_test.go` — new
- `docs/rest-pagination.md` — **read-only**

**Interfaces:**
- Consumes: `occurrence.Processor` (Task 15), `transition.RestModel` (Task 14).
- Produces:
  - `GET /events/occurrences` — paginated; filters `filter[definitionId]`, `filter[type]`, `filter[state]`, `filter[worldId]`, `filter[channelId]`, `filter[mapId]`, `filter[voyageId]`, `filter[startedAt][from]`, `filter[startedAt][to]` (FR-API6)
  - `GET /events/occurrences/{occurrenceId}` — with transitions as an `included` relationship (FR-API5)
  - `GET /events/worlds/{worldId}/channels/{channelId}/maps/{mapId}/visuals` — the narrow projection, resource name `event-visuals`
  - `occurrence.VisualRestModel{Id, OccurrenceId, Visual, State, SubState, Bgm}`

- [ ] **Step 1: Write the failing test**

```go
// FR-B13/FR-B14: the cabin's map row has visual=false, so the projection
// returns nothing for it. This is what makes the deck/cabin distinction a
// query predicate rather than a branch in atlas-channel.
func TestVisualsProjectionExcludesNonVisualMaps(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	o, _ := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"), registry.Seed{
		Stage: "ATTACKING", WorldId: 1, ChannelId: 4, ConcurrencyKey: "k",
		Context: json.RawMessage(`{"visual":"CONTI_MOVE","state":10,"subState":4,"bgm":"Bgm04/ArabPirate"}`),
		Maps: []registry.MapScope{
			{MapId: 200090010, Visual: true},
			{MapId: 200090011, Visual: false},
		},
	}, "w")

	deck := doGET(t, db, "/events/worlds/1/channels/4/maps/200090010/visuals")
	if len(deck) != 1 || deck[0].OccurrenceId != o.Id().String() {
		t.Fatalf("deck returned %d visuals, want 1", len(deck))
	}
	if deck[0].State != 10 || deck[0].SubState != 4 || deck[0].Bgm != "Bgm04/ArabPirate" {
		t.Fatalf("projection = %+v", deck[0])
	}

	cabin := doGET(t, db, "/events/worlds/1/channels/4/maps/200090011/visuals")
	if len(cabin) != 0 {
		t.Fatalf("cabin returned %d visuals, want 0", len(cabin))
	}
}

// A COMPLETED occurrence's visual is gone even though its map rows remain.
func TestVisualsProjectionExcludesCompletedOccurrences(t *testing.T) { ... }

// FR-API5: transitions come back as an included relationship.
func TestOccurrenceDetailIncludesTransitions(t *testing.T) { ... }

// FR-API9: another tenant's occurrences are never returned.
func TestOccurrenceListIsTenantScoped(t *testing.T) { ... }
```

Fill in the three elided bodies following the first one's shape — read
`services/atlas-transports/atlas.com/transports/transport/resource_paginate_test.go`
and `resource_tenant_test.go` for the repo's REST test harness and copy it.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./event/occurrence/ -run Visuals -v`
Expected: FAIL — no resource registered.

- [ ] **Step 3: Implement**

`RestModel` exposes exactly FR-API4's fields plus a `definition` relationship.
The projection is deliberately **not** the full occurrence resource — the channel
needs four fields and must not be coupled to the occurrence schema (design §9.7):

```go
// VisualRestModel is the map-entry projection. It is intentionally narrow: the
// channel needs "what should I draw", not the occurrence's whole shape. The
// visual/state/subState/bgm values are read out of the occurrence context,
// which the EVENT package populated — the generic layer copies them, it does
// not interpret them.
type VisualRestModel struct {
	Id           string `json:"-"`
	OccurrenceId string `json:"occurrenceId"`
	Visual       string `json:"visual"`
	State        byte   `json:"state"`
	SubState     byte   `json:"subState"`
	Bgm          string `json:"bgm"`
}

func (m VisualRestModel) GetName() string { return "event-visuals" }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./event/occurrence/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-events/atlas.com/events/event/occurrence/
git commit -m "feat(task-231): add the occurrence REST surface and the map-entry visual projection"
```

---

## Task 17: `event/scheduling` persistence and dedup

### Files

- `services/atlas-events/atlas.com/events/event/scheduling/model.go` — new
- `services/atlas-events/atlas.com/events/event/scheduling/builder.go` — new
- `services/atlas-events/atlas.com/events/event/scheduling/entity.go` — new
- `services/atlas-events/atlas.com/events/event/scheduling/provider.go` — new
- `services/atlas-events/atlas.com/events/event/scheduling/administrator.go` — new
- `services/atlas-events/atlas.com/events/event/scheduling/administrator_test.go` — new

**Interfaces:**
- Produces:
  - Work types `WorkTypeTriggerEvaluation = "TRIGGER_EVALUATION"`, `WorkTypeOccurrenceTransition = "OCCURRENCE_TRANSITION"`
  - States `StatePending`, `StateProcessing`, `StateCompleted`, `StateCancelled`, `StateFailed`
  - `scheduling.Model` with `Id()`, `DefinitionId()`, `OccurrenceId() uuid.UUID`, `Type()`, `Context() json.RawMessage`, `ExecuteAt() time.Time`, `State()`, `Attempts() int`, `LastError() string`, `DedupeKey() string`
  - `scheduling.Entity` on table `scheduled_event_work`, `scheduling.MigrateTable`
  - `scheduling.Administrator` with `Schedule(m Model) (Model, bool, error)` — second return is "newly inserted"; a dedupe collision is `(existing, false, nil)`, never an error
  - `scheduling.CancelPendingForDefinition(definitionId uuid.UUID) (int64, error)` (FR-S10)

- [ ] **Step 1: Write the failing test**

```go
// FR-B4/FR-S8 guard 1: a redelivered Kafka message must be a no-op insert, not
// a second work row and not an error.
func TestScheduleDedupesOnKey(t *testing.T) {
	db := newTestDB(t)
	a := NewAdministrator(testLogger(t), testCtx(t), db)

	m, _ := NewBuilder(defId, WorkTypeTriggerEvaluation).
		SetExecuteAt(at).
		SetDedupeKey("balrog:d1:v1:1:4").
		Build()

	first, created, err := a.Schedule(m)
	if err != nil || !created {
		t.Fatalf("first schedule: created=%v err=%v", created, err)
	}
	second, created, err := a.Schedule(m)
	if err != nil {
		t.Fatalf("redelivery must not error: %v", err)
	}
	if created {
		t.Fatalf("redelivery created a second row")
	}
	if second.Id() != first.Id() {
		t.Fatalf("redelivery returned a different row")
	}
}

// The dedupe index is PARTIAL on (PENDING, PROCESSING), so a cancelled or
// failed row does not block a legitimate retry of the same logical work.
func TestDedupeDoesNotBlockAfterCancellation(t *testing.T) {
	db := newTestDB(t)
	a := NewAdministrator(testLogger(t), testCtx(t), db)
	m, _ := NewBuilder(defId, WorkTypeTriggerEvaluation).SetExecuteAt(at).SetDedupeKey("k").Build()

	first, _, _ := a.Schedule(m)
	if _, err := a.SetState(first.Id(), StateCancelled, ""); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	if _, created, err := a.Schedule(m); err != nil || !created {
		t.Fatalf("re-schedule after cancel: created=%v err=%v", created, err)
	}
}

// An empty dedupe key opts out — an OCCURRENCE_TRANSITION scheduled per
// occurrence needs no cross-message dedup.
func TestEmptyDedupeKeyAllowsMany(t *testing.T) { ... }

// FR-S10: a definition's pending work can be cancelled — e.g. an Anniversary
// definition whose start time is edited before it fires. Only PENDING rows are
// affected; a row already PROCESSING belongs to a claimer.
func TestCancelPendingForDefinitionLeavesClaimedWorkAlone(t *testing.T) { ... }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./event/scheduling/ -v`
Expected: FAIL — package does not compile.

- [ ] **Step 3: Implement `entity.go`**

```go
type Entity struct {
	ID                uuid.UUID  `gorm:"primaryKey;column:id;type:uuid"`
	TenantID          uuid.UUID  `gorm:"column:tenant_id;type:uuid;not null"`
	EventDefinitionID uuid.UUID  `gorm:"column:event_definition_id;type:uuid;not null"`
	EventOccurrenceID *uuid.UUID `gorm:"column:event_occurrence_id;type:uuid"`
	Type              string     `gorm:"column:type;not null"`
	Context           string     `gorm:"column:context;type:jsonb;not null;default:'{}'"`
	ExecuteAt         time.Time  `gorm:"column:execute_at;not null"`
	State             string     `gorm:"column:state;not null"`
	ClaimedBy         string     `gorm:"column:claimed_by"`
	ClaimedAt         *time.Time `gorm:"column:claimed_at"`
	Attempts          int        `gorm:"column:attempts;not null;default:0"`
	LastError         string     `gorm:"column:last_error"`
	DedupeKey         string     `gorm:"column:dedupe_key;not null;default:''"`
}

func (Entity) TableName() string { return "scheduled_event_work" }
```

`Schedule` inserts inside a transaction and, on a unique violation of
`ux_sew_dedupe`, re-reads the existing row and returns `(existing, false, nil)`.
The index itself is created in Task 19.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./event/scheduling/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-events/atlas.com/events/event/scheduling/
git commit -m "feat(task-231): add scheduled event work persistence with dedup"
```

---

## Task 18: The claim-locked poller

Not leader-elected. `libs/atlas-lock` would also prevent double execution, but it
makes the scheduler single-replica by construction — a slow or wedged leader
stalls **all** events, which directly contradicts FR-N6. `SKIP LOCKED` keeps
every replica working and isolates a stuck row to itself, and it avoids
`libs/atlas-lock`'s documented `ATLAS_ENV` lease-scoping hazard entirely
(design §5.1, §14 A2).

### Files

- `services/atlas-events/atlas.com/events/event/scheduling/processor.go` — new (`ClaimBatch`, `Reclaim`, `Execute`)
- `services/atlas-events/atlas.com/events/event/scheduling/poller.go` — new (the ticker loop)
- `services/atlas-events/atlas.com/events/event/scheduling/poller_test.go` — new
- `libs/atlas-outbox/drainer.go:223` — **read-only** precedent for the `SKIP LOCKED` claim
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/store.go:242` — **read-only** precedent for `WithoutTenantFilter` + `SKIP LOCKED`
- `libs/atlas-database/tenant_scope.go:22` — **read-only**; `WithoutTenantFilter`

**Interfaces:**
- Consumes: `scheduling.Entity` (Task 17), `registry.Get` (Task 11),
  `definition.Processor` (Task 13), `occurrence.Processor` (Task 15).
- Produces:
  - `scheduling.Processor.ClaimBatch(instanceId string, limit int) ([]Model, error)`
  - `scheduling.Processor.Reclaim(lease time.Duration) (int64, error)`
  - `scheduling.Processor.ExecuteOne(m Model) error`
  - `scheduling.NewPoller(l, ctx, db, cfg Config) *Poller` with `Run()`;
    `Config{Interval, Lease, BatchSize, MaxAttempts, Backoff time.Duration}`
    read from env in `main.go` (FR-N16)

- [ ] **Step 1: Write the failing test**

```go
// PRD §20.3 explicitly demands a CONCURRENT test rather than inspection: two
// instances against one database, N rows, each executed exactly once.
func TestTwoInstancesNeverExecuteTheSameRow(t *testing.T) {
	db := newSharedPostgres(t) // must be real Postgres — SKIP LOCKED is the thing under test
	const rows = 50
	seedPendingWork(t, db, rows)

	var mu sync.Mutex
	executed := map[uuid.UUID]int{}
	record := func(m Model) error {
		mu.Lock()
		defer mu.Unlock()
		executed[m.Id()]++
		return nil
	}

	var wg sync.WaitGroup
	for _, instance := range []string{"a", "b"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			p := NewProcessorWithExecutor(testLogger(t), testCtx(t), db, record)
			for i := 0; i < 20; i++ {
				claimed, err := p.ClaimBatch(id, 10)
				if err != nil {
					t.Errorf("claim: %v", err)
					return
				}
				for _, m := range claimed {
					if err := p.ExecuteOne(m); err != nil {
						t.Errorf("execute: %v", err)
					}
				}
			}
		}(instance)
	}
	wg.Wait()

	if len(executed) != rows {
		t.Fatalf("executed %d distinct rows, want %d", len(executed), rows)
	}
	for id, n := range executed {
		if n != 1 {
			t.Fatalf("row %s executed %d times", id, n)
		}
	}
}

// FR-S5: work whose executeAt passed while the service was down runs on
// recovery — late, not lost.
func TestOverdueWorkIsClaimedOnFirstPoll(t *testing.T) {
	db := newSharedPostgres(t)
	insertWork(t, db, StatePending, time.Now().Add(-2*time.Hour))

	claimed, err := NewProcessor(testLogger(t), testCtx(t), db).ClaimBatch("a", 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("claimed %d overdue rows, want 1", len(claimed))
	}
}

// Work not yet due is not claimed.
func TestFutureWorkIsNotClaimed(t *testing.T) { ... }

// FR-S7: a row left PROCESSING by a dead replica returns to PENDING after the
// lease, with attempts incremented.
func TestLeaseReclaimReturnsOrphanedWork(t *testing.T) {
	db := newSharedPostgres(t)
	id := insertClaimedWork(t, db, "dead-replica", time.Now().Add(-10*time.Minute))

	n, err := NewProcessor(testLogger(t), testCtx(t), db).Reclaim(5 * time.Minute)
	if err != nil {
		t.Fatalf("Reclaim: %v", err)
	}
	if n != 1 {
		t.Fatalf("reclaimed %d, want 1", n)
	}
	got := readWork(t, db, id)
	if got.State != StatePending || got.Attempts != 1 {
		t.Fatalf("state=%s attempts=%d, want PENDING/1", got.State, got.Attempts)
	}
}

// FR-S9: repeated failure lands in FAILED with lastError retained, and a FAILED
// row never blocks a PENDING sibling — the poll index is partial on PENDING, so
// it is not even visible to the poller.
func TestRepeatedFailureLandsInFailedWithoutBlockingSiblings(t *testing.T) {
	db := newSharedPostgres(t)
	failing := insertWork(t, db, StatePending, time.Now().Add(-time.Minute))
	sibling := insertWork(t, db, StatePending, time.Now().Add(-time.Minute))

	p := NewProcessorWithExecutor(testLogger(t), testCtx(t), db, func(Model) error {
		return errors.New("boom")
	})
	p.SetMaxAttempts(2)

	for i := 0; i < 3; i++ {
		claimed, _ := p.ClaimBatch("a", 10)
		for _, m := range claimed {
			_ = p.ExecuteOne(m)
		}
		_, _ = p.Reclaim(0)
	}

	f := readWork(t, db, failing)
	if f.State != StateFailed || f.LastError == "" {
		t.Fatalf("failing row: state=%s lastError=%q", f.State, f.LastError)
	}
	s := readWork(t, db, sibling)
	if s.State == StatePending {
		t.Fatalf("sibling was never processed — a FAILED row blocked the queue")
	}
}

// A definition whose type has no registered handler makes its work FAIL loudly
// with a named reason, rather than silently completing (design §3.2).
func TestWorkForAnUnregisteredTypeFails(t *testing.T) {
	...
	if !strings.Contains(got.LastError, "no handler for type") {
		t.Fatalf("lastError = %q", got.LastError)
	}
}
```

`newSharedPostgres` must be a real Postgres — `SKIP LOCKED` is precisely what is
under test and sqlite does not have it. Use whatever container/dockertest helper
the repo already has; find it with
`grep -rln "dockertest\|testcontainers\|postgres" --include='*_test.go' services/ libs/ | head`.
If none exists, use `libs/atlas-outbox`'s drainer test setup as the model. Gate
the test with `testing.Short()` if the harness needs it, but it MUST run in the
default `go test ./...` — an acceptance criterion depends on it.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./event/scheduling/ -run TwoInstances -v`
Expected: FAIL — `ClaimBatch` undefined.

- [ ] **Step 3: Implement `ClaimBatch`**

```go
// ClaimBatch atomically moves up to `limit` due rows PENDING -> PROCESSING and
// stamps claimedBy/claimedAt (FR-S6). SKIP LOCKED means a second replica takes
// the NEXT rows rather than blocking, so both keep working and a stuck row is
// isolated to its own claimer (FR-N6).
//
// The poller is the ONE deliberate exception to tenant filtering: it must see
// every tenant's work. It re-enters a tenant-scoped context per claimed row
// before invoking any handler, so a handler never runs unfiltered (FR-N13,
// FR-N14).
func (p *ProcessorImpl) ClaimBatch(instanceId string, limit int) ([]Model, error) {
	var claimed []Entity
	err := p.db.Transaction(func(tx *gorm.DB) error {
		var rows []Entity
		if err := tx.WithContext(database.WithoutTenantFilter(p.ctx)).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("state = ? AND execute_at <= ?", StatePending, time.Now()).
			Order("execute_at ASC").Limit(limit).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		now := time.Now()
		if err := tx.WithContext(database.WithoutTenantFilter(p.ctx)).
			Model(&Entity{}).Where("id IN ?", ids).
			Updates(map[string]any{
				"state": StateProcessing, "claimed_by": instanceId, "claimed_at": now,
			}).Error; err != nil {
			return err
		}
		claimed = rows
		return nil
	})
	...
}
```

- [ ] **Step 4: Implement `Reclaim` and the outcome state machine**

`Reclaim(lease)` moves `state = 'PROCESSING' AND claimed_at < now() - lease`
back to `PENDING` and increments `attempts` (FR-S7).

`ExecuteOne` applies the design §5.2 table exactly:

| Outcome | Row transition |
|---|---|
| Handler returned normally | `COMPLETED` |
| Handler errored, `attempts < max` | `PENDING`, `execute_at = now + backoff`, `last_error` set |
| Handler errored, `attempts >= max` | `FAILED`, `last_error` retained (FR-S9) |
| Definition disabled | `COMPLETED`, no occurrence |
| No handler for the type | `FAILED`, `last_error = "no handler for type X"` |

Dispatch: read the definition, `registry.Get(d.Type())`, then
`h.Evaluate` for `TRIGGER_EVALUATION` (a `(nil, nil)` return is the ordinary
"no occurrence" outcome and completes the row) or `h.Advance` for
`OCCURRENCE_TRANSITION`. On a `Seed`, call `occurrence.CreateFromSeed` then
`h.Start`, and apply the returned `Progress`. Treat
`occurrence.ErrConcurrencyKeyTaken` as success — someone else already created it.

- [ ] **Step 5: Implement `poller.go`**

A plain ticker over `ClaimBatch` + `Reclaim`, wrapped in `routine.Go`.
`Run` calls `Reclaim` **once immediately** before the first tick, so work
orphaned by the previous process's death resumes promptly rather than after a
full lease interval (design §6). Recovering active occurrences needs no code:
they are rows, and pending work is picked up by the ordinary poll (FR-N5).

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./event/scheduling/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-events/atlas.com/events/event/scheduling/
git commit -m "feat(task-231): add the claim-locked scheduled work poller"
```

---

## Task 19: Indexes, with `EXPLAIN` evidence

PRD §14 explicitly asks that the composite shape be confirmed against the query
plan rather than assumed. Do that here, and record the output — a plan that
falls back to a sequential scan is a finding, not a formality.

### Files

- `services/atlas-events/atlas.com/events/event/{definition,occurrence,scheduling}/entity.go` — add the partial/unique indexes via raw SQL in each `MigrateTable`
- `docs/tasks/task-231-generalized-events-service/index-plans.md` — **new file**; the captured `EXPLAIN` output

**Interfaces:**
- Consumes: all three entity sets.
- Produces: the seven indexes below, created idempotently
  (`CREATE INDEX IF NOT EXISTS`) after `AutoMigrate` in each `MigrateTable`.

- [ ] **Step 1: Add the indexes**

GORM tags cannot express partial indexes, so each `MigrateTable` runs raw SQL
after `AutoMigrate`:

```sql
-- FR-S4 poller hot path. PARTIAL on state: this is what makes the poll cost
-- independent of how many COMPLETED rows have accumulated (FR-N16), and is why
-- no retention policy is needed in this task (design §15.7).
CREATE INDEX IF NOT EXISTS ix_sew_pending_due ON scheduled_event_work (execute_at)
    WHERE state = 'PENDING';

-- FR-S7 lease reclaim sweep.
CREATE INDEX IF NOT EXISTS ix_sew_processing_claimed ON scheduled_event_work (claimed_at)
    WHERE state = 'PROCESSING';

-- FR-B4 dedup. Partial so a cancelled/failed row does not block a retry.
CREATE UNIQUE INDEX IF NOT EXISTS ux_sew_dedupe ON scheduled_event_work (tenant_id, dedupe_key)
    WHERE state IN ('PENDING','PROCESSING') AND dedupe_key <> '';

-- FR-API7 second query ("is Anniversary happening?").
CREATE INDEX IF NOT EXISTS ix_occ_type_state ON event_occurrence (tenant_id, type, state);

-- design §5.3 concurrency policy.
CREATE UNIQUE INDEX IF NOT EXISTS ux_occ_concurrency ON event_occurrence
    (tenant_id, event_definition_id, concurrency_key)
    WHERE state = 'ACTIVE' AND concurrency_key <> '';

-- FR-B15 map-entry query. Leading map_id because that is the most selective
-- column at the call site; world/channel/state are filtered from the joined
-- occurrence row.
CREATE INDEX IF NOT EXISTS ix_occ_map ON event_occurrence_map (map_id, occurrence_id)
    WHERE visual = true;
CREATE INDEX IF NOT EXISTS ix_occ_active_scope ON event_occurrence (tenant_id, world_id, channel_id, state)
    WHERE state = 'ACTIVE';
```

Note the `AND dedupe_key <> ''` on `ux_sew_dedupe` — without it, every
`OCCURRENCE_TRANSITION` row (which opts out of dedup with an empty key) would
collide with every other one for the same tenant.

- [ ] **Step 2: Capture the plans**

Against a seeded database (at least a few thousand occurrence and work rows so
the planner does not just prefer a seq scan on a toy table), run and capture:

```sql
EXPLAIN ANALYZE
SELECT * FROM scheduled_event_work
WHERE state = 'PENDING' AND execute_at <= now()
ORDER BY execute_at ASC LIMIT 50;

EXPLAIN ANALYZE
SELECT o.* FROM event_occurrence o
JOIN event_occurrence_map m ON m.occurrence_id = o.id
WHERE m.map_id = 200090010 AND m.visual
  AND o.tenant_id = '…' AND o.world_id = 1 AND o.channel_id = 4
  AND o.state = 'ACTIVE';

EXPLAIN ANALYZE
SELECT * FROM event_occurrence
WHERE tenant_id = '…' AND type = 'ANNIVERSARY' AND state = 'ACTIVE';
```

- [ ] **Step 3: Record the evidence**

Write `docs/tasks/task-231-generalized-events-service/index-plans.md` containing
the **verbatim** `EXPLAIN ANALYZE` output for all three, each under a heading
naming the FR it serves. If any plan shows a sequential scan on
`scheduled_event_work`, `event_occurrence` or `event_occurrence_map`, state that
plainly and fix the index before moving on — do not paraphrase the plan or
assert "index-served" without the output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-events/atlas.com/events/event/ docs/tasks/task-231-generalized-events-service/index-plans.md
git commit -m "perf(task-231): add the partial indexes the poller and gameplay queries need"
```

---

## Task 20: Wire the core into `main.go`

### Files

- `services/atlas-events/atlas.com/events/main.go` — migrations, routes, poller, seeding
- `services/atlas-events/atlas.com/events/rest/handler.go` — new; copy `services/atlas-party-quests/atlas.com/party-quests/rest/handler.go`
- `services/atlas-events/atlas.com/events/tenant/{processor,requests,rest}.go` — new; copy `services/atlas-party-quests/atlas.com/party-quests/tenant/`

**Interfaces:**
- Consumes: everything from Tasks 11–19.
- Produces: a running service whose REST surface answers Tasks 13 and 16's
  routes and whose poller executes due work.

- [ ] **Step 1: Register migrations**

```go
	db := database.Connect(l, database.SetMigrations(
		definition.MigrateTable,
		occurrence.MigrateTable,
		transition.MigrateTable,
		scheduling.MigrateTable,
		func(db *gorm.DB) error { return db.AutoMigrate(&seeder.SeedState{}) },
	))
```

- [ ] **Step 2: Start the poller**

```go
	// The poller is the only in-memory component, and it is stateless: every
	// correctness-critical fact is a row (FR-N1). Interval, lease, batch size
	// and max attempts are configuration (FR-N16).
	pollerCfg := scheduling.ConfigFromEnv()
	routine.Go(l, rt.Context(), func(ctx context.Context) {
		scheduling.NewPoller(l, ctx, db, pollerCfg).Run()
	})
```

`ConfigFromEnv` reads `EVENTS_POLL_INTERVAL`, `EVENTS_WORK_LEASE`,
`EVENTS_POLL_BATCH_SIZE`, `EVENTS_WORK_MAX_ATTEMPTS`, `EVENTS_WORK_BACKOFF`,
each with a documented default. Add all five to
`deploy/k8s/base/atlas-events.yaml`'s env block with their defaults.

- [ ] **Step 3: Register routes and seeding**

```go
		AddRouteInitializer(definition.InitResource(GetServer())(db)).
		AddRouteInitializer(definition.InitSeedResource(GetServer())(db)).
		AddRouteInitializer(occurrence.InitResource(GetServer())(db)).
		AddRouteInitializer(occurrence.InitVisualResource(GetServer())(db)).
```

- [ ] **Step 4: Verify**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./...`
Then, from the repo root, `docker buildx bake atlas-events`.
Expected: all exit 0.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-events/atlas.com/events deploy/k8s/base/atlas-events.yaml
git commit -m "feat(task-231): wire migrations, REST routes and the poller into atlas-events"
```

---

# Phase E — Event 1: Crimson Balrog, and `atlas-channel` (design §9, §11)

**Module root for Tasks 21–27:** `services/atlas-events/atlas.com/events`
**Module root for Tasks 28–30:** `services/atlas-channel/atlas.com/channel`

## Task 21: `events/crimsonbalrog` configuration and the handler's static half

Every value is configuration. The package contains no `2`, no `0.42`, no
`8150000` outside the seed JSON (FR-B1). `spawnPosition` is modelled per attack
map rather than as one flat position, because two decks with different geometry
must both be configurable; `relatedMapIds` stays flat because cabins get nothing.

### Files

- `services/atlas-events/atlas.com/events/events/crimsonbalrog/config.go` — new
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/handler.go` — new (`Type`, `ValidateConfiguration`, `ConcurrencyKey`; the other four methods return `errors.New("not implemented")` and are filled in Tasks 24, 25, 27 — **remove every one of those before the phase ends**; no stub may land in the final branch)
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/config_test.go` — new
- `services/atlas-configurations/seed-data/events/definitions/event-crimson-balrog.json` — fill in the configuration created in Task 13

**Interfaces:**
- Consumes: `registry.Handler` (Task 11).
- Produces:
  - `crimsonbalrog.TypeName = "CRIMSON_BALROG"`
  - `crimsonbalrog.Config{ApplicableRouteIds []uuid.UUID, TriggerDelay, TriggerDelayJitter time.Duration, AttackProbability float64, MonsterId uint32, MonsterCount uint32, AttackMaps []AttackMap, RelatedMapIds []_map.Id, BackgroundMusic string, Visual VisualConfig}`
  - `crimsonbalrog.AttackMap{MapId _map.Id, SpawnPositions []Position}`, `Position{X, Y int16}`
  - `crimsonbalrog.VisualConfig{Name string, ShowState, ShowSubState, HideState, HideSubState byte}`
  - `crimsonbalrog.WorkContext{VoyageId uuid.UUID, RouteId uuid.UUID, WorldId world.Id, ChannelId channel.Id, StagingMapId, DestinationMapId, ObservationMapId _map.Id, EnRouteMapIds []_map.Id, DepartedAt time.Time}`
  - `crimsonbalrog.NewHandler() registry.Handler`

- [ ] **Step 1: Write the failing test**

```go
package crimsonbalrog

import (
	"context"
	"encoding/json"
	"testing"
)

const validConfig = `{
  "applicableRouteIds": ["4c9a1e2b-0000-4000-8000-000000000001"],
  "triggerDelay": "3m",
  "triggerDelayJitter": "60s",
  "attackProbability": 0.42,
  "monsterId": 8150000,
  "monsterCount": 2,
  "attackMaps": [{"mapId": 200090010, "spawnPositions": [{"x": 0, "y": 0}, {"x": 100, "y": 0}]}],
  "relatedMapIds": [200090011],
  "backgroundMusic": "Bgm04/ArabPirate",
  "visual": {"name": "CONTI_MOVE", "showState": 10, "showSubState": 4, "hideState": 10, "hideSubState": 5}
}`

func TestValidateAcceptsAWellFormedConfiguration(t *testing.T) {
	if err := NewHandler().ValidateConfiguration(json.RawMessage(validConfig)); err != nil {
		t.Fatalf("ValidateConfiguration: %v", err)
	}
}

// FR-D6: the rejection must name the offending field, so an administrator sees
// what is wrong rather than "invalid configuration".
func TestValidateRejectsFieldByField(t *testing.T) {
	for _, tc := range []struct{ name, mutate, wantField string }{
		{"zero monster count", `"monsterCount": 2`, "monsterCount"},
		{"probability above one", `"attackProbability": 0.42`, "attackProbability"},
		{"no attack maps", `"attackMaps"`, "attackMaps"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := mutateConfig(t, validConfig, tc.name)
			err := NewHandler().ValidateConfiguration(json.RawMessage(bad))
			if err == nil {
				t.Fatalf("expected rejection")
			}
			if !strings.Contains(err.Error(), tc.wantField) {
				t.Fatalf("error %q does not name %q", err, tc.wantField)
			}
		})
	}
}

// FR-B1: there are more spawn positions than monsters, or fewer. Fewer is
// rejected at write time rather than producing an out-of-range panic at spawn.
func TestValidateRejectsFewerSpawnPositionsThanMonsters(t *testing.T) { ... }

// design §15.5: the key makes the occurrence one-per-voyage-per-channel, so two
// simultaneous voyages in different channels are independent (FR-N11) but one
// voyage cannot be attacked twice.
func TestConcurrencyKeyIsPerVoyageAndChannel(t *testing.T) {
	h := NewHandler()
	ctx := context.Background()

	a, err := h.ConcurrencyKey(ctx, workContextJSON(t, "voyage-1", 1, 4))
	if err != nil {
		t.Fatalf("ConcurrencyKey: %v", err)
	}
	same, _ := h.ConcurrencyKey(ctx, workContextJSON(t, "voyage-1", 1, 4))
	otherChannel, _ := h.ConcurrencyKey(ctx, workContextJSON(t, "voyage-1", 1, 5))
	otherVoyage, _ := h.ConcurrencyKey(ctx, workContextJSON(t, "voyage-2", 1, 4))

	if a != same {
		t.Fatalf("key not stable: %q vs %q", a, same)
	}
	if a == otherChannel || a == otherVoyage {
		t.Fatalf("key not discriminating: %q / %q / %q", a, otherChannel, otherVoyage)
	}
}
```

Fill in `mutateConfig` (re-marshal `validConfig` with one field replaced) and
`workContextJSON` (marshal a `WorkContext`) in the same file, and write the
elided `TestValidateRejectsFewerSpawnPositionsThanMonsters` body following the
shape of the first two.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./events/crimsonbalrog/ -v`
Expected: FAIL — package does not compile.

- [ ] **Step 3: Write `config.go`**

Durations decode from Go duration strings via a `json.Unmarshaler` on a small
`Duration` wrapper (or `time.ParseDuration` in a custom `UnmarshalJSON` on
`Config`). `Validate() error` returns an error naming the first offending field:

```go
// Validate rejects a configuration this handler cannot interpret (FR-D6). Each
// error names its field so the JSON:API error an administrator sees is
// actionable. Every bound here exists because violating it would fail LATER, at
// trigger time, where nobody is watching.
func (c Config) Validate() error {
	if len(c.ApplicableRouteIds) == 0 {
		return errors.New("applicableRouteIds: must contain at least one route")
	}
	if c.AttackProbability < 0 || c.AttackProbability > 1 {
		return fmt.Errorf("attackProbability: must be in [0,1], got %v", c.AttackProbability)
	}
	if c.MonsterCount == 0 {
		return errors.New("monsterCount: must be greater than zero")
	}
	if c.TriggerDelay < 0 || c.TriggerDelayJitter < 0 {
		return errors.New("triggerDelay/triggerDelayJitter: must not be negative")
	}
	if len(c.AttackMaps) == 0 {
		return errors.New("attackMaps: must contain at least one map")
	}
	for _, am := range c.AttackMaps {
		if uint32(len(am.SpawnPositions)) < c.MonsterCount {
			return fmt.Errorf("attackMaps[%d].spawnPositions: %d positions for monsterCount %d", am.MapId, len(am.SpawnPositions), c.MonsterCount)
		}
	}
	if c.Visual.Name == "" {
		return errors.New("visual.name: must be set")
	}
	return nil
}
```

- [ ] **Step 4: Write the handler's static half**

```go
const TypeName = "CRIMSON_BALROG"

func (h *Handler) Type() string { return TypeName }

func (h *Handler) ValidateConfiguration(raw json.RawMessage) error {
	c, err := DecodeConfig(raw)
	if err != nil {
		return err
	}
	return c.Validate()
}

// ConcurrencyKey scopes an occurrence to one voyage in one channel of one
// world (FR-V2). The generic layer turns this into a unique constraint among
// ACTIVE occurrences, which is design §5.3's third idempotency guard: even if
// Kafka dedup and the work-row state machine BOTH fail, the second occurrence
// insert fails rather than producing two live Balrog attacks on one voyage.
func (h *Handler) ConcurrencyKey(_ context.Context, workContext json.RawMessage) (string, error) {
	var wc WorkContext
	if err := json.Unmarshal(workContext, &wc); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s|%d|%d", wc.VoyageId, wc.WorldId, wc.ChannelId), nil
}
```

- [ ] **Step 5: Fill in the seed configuration**

`services/atlas-configurations/seed-data/events/definitions/event-crimson-balrog.json`
carries the reference values from PRD F6 (`Cosmic/scripts/event/Boats.js`,
`Cosmic/src/main/java/tools/PacketCreator.java:4639-4651`) and ships **disabled**:
monster `8150000`, count `2`, probability `0.42`, delay `3m` + jitter `60s`,
BGM `Bgm04/ArabPirate`, visual `CONTI_MOVE` show `(10,4)` / hide `(10,5)`,
attack maps `200090010` (To Orbis) and `200090000` (To Ellinia), related maps
`200090011` and `200090001` (the cabins). `applicableRouteIds` must reference
the route ids the transport seed data actually uses — read
`services/atlas-configurations/seed-data/` for the transport route seeds and use
those ids; do not invent uuids.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./events/crimsonbalrog/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-events/atlas.com/events/events/crimsonbalrog/ services/atlas-configurations/seed-data/
git commit -m "feat(task-231): add Crimson Balrog configuration, validation and concurrency policy"
```

---

## Task 22: External REST clients in `atlas-events`

Two network calls, both cheap, both on the evaluation path. Neither may be read
as a negative answer on failure: an unreachable `atlas-maps` must **retry**, not
be silently interpreted as "nobody aboard" (design §9.3).

### Files

- `services/atlas-events/atlas.com/events/external/transports/{processor,requests,rest}.go` — new
- `services/atlas-events/atlas.com/events/external/maps/{processor,requests,rest}.go` — new
- `services/atlas-events/atlas.com/events/external/transports/processor_test.go` — new
- `services/atlas-channel/atlas.com/channel/weather/{processor,requests,rest}.go` — **read-only** reference for the client shape
- `services/atlas-transports/atlas.com/transports/map/requests.go` — **read-only** reference for the paginated `DrainProvider` character list

**Interfaces:**
- Consumes: the `voyageId` field on the route resource (Task 7).
- Produces:
  - `transports.Processor.GetRoute(routeId uuid.UUID) (RestModel, error)` over
    `GET /transports/routes/{routeId}`; `RestModel` mirrors only `state` and `voyageId`
  - `transports.VoyageUnderway(rm RestModel, voyageId uuid.UUID) bool`
  - `maps.Processor.CharacterIdsInMap(f field.Model) ([]uint32, error)` over the
    paginated `worlds/%d/channels/%d/maps/%d/instances/%s/characters/` list,
    drained with `requests.DrainProvider`

- [ ] **Step 1: Write the failing test**

```go
// design §7.4: "still underway" is state == in_transit AND the SAME voyage. A
// route that has since departed on the NEXT trip is in_transit again, and
// comparing state alone would wrongly report our voyage as ongoing.
func TestVoyageUnderwayRequiresBothStateAndIdentity(t *testing.T) {
	mine := uuid.New()
	next := uuid.New()

	for _, tc := range []struct {
		name string
		rm   RestModel
		want bool
	}{
		{"in transit, same voyage", RestModel{State: "in_transit", VoyageID: mine.String()}, true},
		{"in transit, next voyage", RestModel{State: "in_transit", VoyageID: next.String()}, false},
		{"awaiting return", RestModel{State: "awaiting_return", VoyageID: ""}, false},
		{"open entry", RestModel{State: "open_entry", VoyageID: ""}, false},
	} {
		if got := VoyageUnderway(tc.rm, mine); got != tc.want {
			t.Fatalf("%s: VoyageUnderway = %v, want %v", tc.name, got, tc.want)
		}
	}
}
```

Confirm the exact `state` strings the route resource emits by reading
`services/atlas-transports/atlas.com/transports/transport/state.go` — use those
literals, do not guess them.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./external/transports/ -v`
Expected: FAIL — package does not compile.

- [ ] **Step 3: Implement both clients**

Copy `services/atlas-channel/atlas.com/channel/weather/` verbatim as the shape:
a `Processor` interface, a `ProcessorImpl` with `l`/`ctx`, `NewProcessor`, a
`requests.go` with `getBaseRequest() string { return requests.RootUrl("TRANSPORTS") }`
(and `"MAPS"` for the other), and a `rest.go` mirroring **only** the fields this
service reads.

The character list is paginated server-side (task-117), so use
`requests.DrainProvider` against a bare URL exactly as
`services/atlas-transports/atlas.com/transports/map/requests.go` does — a
`requests.GetRequest` would silently return only the first page.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./external/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-events/atlas.com/events/external/
git commit -m "feat(task-231): add atlas-transports and atlas-maps REST clients to atlas-events"
```

---

## Task 23: Consume `VOYAGE_DEPARTED` into delayed trigger work

The jitter is rolled **here**, at scheduling time, and persisted in `execute_at`.
Rolling it at execution time would make the delay non-durable across a restart,
which FR-B5 and the acceptance criterion "the configured delay survives restart"
both forbid.

### Files

- `services/atlas-events/atlas.com/events/kafka/message/transport/kafka.go` — new; mirror only the consumed types
- `services/atlas-events/atlas.com/events/kafka/consumer/transport/consumer.go` — new
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/trigger.go` — new; the scheduling logic (a processor, not consumer code — FR-N18)
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/trigger_test.go` — new
- `services/atlas-transports/atlas.com/transports/kafka/message/transport/kafka.go` — **read-only** source of truth for the mirrored contract

**Interfaces:**
- Consumes: `EVENT_TOPIC_TRANSPORT_STATUS` / `VOYAGE_DEPARTED` (Task 6);
  `definition.Processor.GetEnabledByType` (Task 13);
  `scheduling.Administrator.Schedule` (Task 17).
- Produces:
  `crimsonbalrog.TriggerProcessor.OnVoyageDeparted(e transport.StatusEvent[transport.VoyageStatusEventBody]) error`.

- [ ] **Step 1: Write the failing test**

```go
// FR-B2: one TRIGGER_EVALUATION per surviving definition, with the full voyage
// scope in context. FR-B3: no occurrence yet.
func TestVoyageDepartedSchedulesOneEvaluationPerApplicableDefinition(t *testing.T) {
	db := newTestDB(t)
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(true), routes(routeA))
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(true), routes(routeB)) // different route
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(false), routes(routeA)) // disabled

	if err := NewTriggerProcessor(testLogger(t), testCtx(t), db).OnVoyageDeparted(departedOn(routeA)); err != nil {
		t.Fatalf("OnVoyageDeparted: %v", err)
	}

	work := readAllWork(t, db)
	if len(work) != 1 {
		t.Fatalf("scheduled %d rows, want 1 (route B and the disabled definition must be skipped)", len(work))
	}
	if work[0].Type != scheduling.WorkTypeTriggerEvaluation {
		t.Fatalf("type = %s", work[0].Type)
	}

	var occurrences int64
	db.Model(&occurrence.Entity{}).Count(&occurrences)
	if occurrences != 0 {
		t.Fatalf("departure created %d occurrences, want 0 (FR-B3)", occurrences)
	}
}

// FR-B4: redelivery creates no second row.
func TestVoyageDepartedRedeliveryIsANoOp(t *testing.T) {
	db := newTestDB(t)
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(true), routes(routeA))
	p := NewTriggerProcessor(testLogger(t), testCtx(t), db)

	e := departedOn(routeA)
	if err := p.OnVoyageDeparted(e); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := p.OnVoyageDeparted(e); err != nil {
		t.Fatalf("redelivery must not error: %v", err)
	}
	if got := len(readAllWork(t, db)); got != 1 {
		t.Fatalf("%d work rows after redelivery, want 1", got)
	}
}

// The delay is DURABLE: executeAt is departedAt + delay + a jitter rolled once,
// now, and stored. Restarting the service cannot re-roll it.
func TestExecuteAtIsDepartureP2lusDelayPlusRolledJitter(t *testing.T) {
	db := newTestDB(t)
	departedAt := time.Date(2026, 8, 15, 13, 0, 0, 0, time.UTC)
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(true), routes(routeA),
		delay(3*time.Minute), jitter(60*time.Second))

	if err := NewTriggerProcessor(testLogger(t), testCtx(t), db).
		OnVoyageDeparted(departedAtOn(routeA, departedAt)); err != nil {
		t.Fatalf("OnVoyageDeparted: %v", err)
	}

	at := readAllWork(t, db)[0].ExecuteAt
	lo := departedAt.Add(3 * time.Minute)
	hi := lo.Add(60 * time.Second)
	if at.Before(lo) || at.After(hi) {
		t.Fatalf("executeAt %s outside [%s, %s]", at, lo, hi)
	}
}

// The whole voyage scope rides on the work row, so evaluation needs no
// follow-up query to learn its maps (FR-V4).
func TestWorkContextCarriesTheFullVoyageScope(t *testing.T) { ... }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./events/crimsonbalrog/ -run VoyageDeparted -v`
Expected: FAIL — `NewTriggerProcessor` undefined.

- [ ] **Step 3: Mirror the consumed contract**

`kafka/message/transport/kafka.go` mirrors **only** what is consumed, with a
pointer to the source of truth, exactly as
`services/atlas-buffs/atlas.com/buffs/kafka/message/characterstatus/kafka.go`
does:

```go
// Package transport mirrors the atlas-transports status events this service
// consumes (source of truth:
// services/atlas-transports/atlas.com/transports/kafka/message/transport/kafka.go).
// Only the consumed types/fields are mirrored; unknown event types on the topic
// are ignored by the handlers' type guards.
package transport
```

- [ ] **Step 4: Write the trigger processor**

```go
// OnVoyageDeparted turns a departure into durable delayed work — one row per
// enabled, applicable definition (FR-B2), and NO occurrence (FR-B3): whether an
// attack happens is decided minutes later, when the delay elapses.
//
// The jitter is rolled HERE and persisted in executeAt. Rolling it at execution
// time would make the delay non-durable across a restart.
func (p *TriggerProcessorImpl) OnVoyageDeparted(e transport.StatusEvent[transport.VoyageStatusEventBody]) error {
	ds, err := definition.NewProcessor(p.l, p.ctx, p.db).GetEnabledByType(crimsonbalrog.TypeName)
	if err != nil {
		return err
	}
	for _, d := range ds {
		c, err := DecodeConfig(d.Configuration())
		if err != nil {
			p.l.WithError(err).Warnf("Skipping definition [%s] with undecodable configuration.", d.Id())
			continue
		}
		if !slices.Contains(c.ApplicableRouteIds, e.RouteId) {
			continue
		}
		wc := WorkContext{
			VoyageId: e.Body.VoyageId, RouteId: e.RouteId,
			WorldId: e.Body.WorldId, ChannelId: e.Body.ChannelId,
			StagingMapId: e.Body.StagingMapId, EnRouteMapIds: e.Body.EnRouteMapIds,
			DestinationMapId: e.Body.DestinationMapId, ObservationMapId: e.Body.ObservationMapId,
			DepartedAt: e.Body.DepartedAt,
		}
		raw, err := json.Marshal(wc)
		if err != nil {
			return err
		}
		executeAt := e.Body.DepartedAt.Add(c.TriggerDelay).Add(rollJitter(c.TriggerDelayJitter))
		m, err := scheduling.NewBuilder(d.Id(), scheduling.WorkTypeTriggerEvaluation).
			SetExecuteAt(executeAt).
			SetContext(raw).
			SetDedupeKey(fmt.Sprintf("balrog:%s:%s:%d:%d", d.Id(), wc.VoyageId, wc.WorldId, wc.ChannelId)).
			Build()
		if err != nil {
			return err
		}
		if _, _, err := scheduling.NewAdministrator(p.l, p.ctx, p.db).Schedule(m); err != nil {
			return err
		}
	}
	return nil
}
```

Inject the jitter source so the test can pin it: `rollJitter` is a field on the
processor defaulting to `func(d time.Duration) time.Duration` over
`math/rand`, with `NewTriggerProcessorWithJitter` for tests.

- [ ] **Step 5: Write the consumer**

`kafka/consumer/transport/consumer.go` follows
`services/atlas-buffs/atlas.com/buffs/kafka/consumer/characterstatus/consumer.go`:
`InitConsumers` + `InitHandlers`, one `handleVoyageDeparted` guarded on
`e.Type != transport.EventStatusVoyageDeparted`, delegating straight to the
processor (FR-N18). Register it in `main.go`.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./... `
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-events/atlas.com/events/
git commit -m "feat(task-231): schedule delayed Balrog evaluation on voyage departure"
```

---

## Task 24: `Handler.Evaluate` — the four gates

In FR-B5's order, because each step is cheaper than the next.

### Files

- `services/atlas-events/atlas.com/events/events/crimsonbalrog/evaluate.go` — new
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/evaluate_test.go` — new
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/handler.go` — replace the `Evaluate` stub

**Interfaces:**
- Consumes: `transports.Processor.GetRoute` / `VoyageUnderway`,
  `maps.Processor.CharacterIdsInMap` (Task 22); `registry.Seed` (Task 11).
- Produces: `Handler.Evaluate(ctx, d registry.Definition, w registry.Work) (*registry.Seed, error)`.

- [ ] **Step 1: Write the failing test**

```go
// Each rejection path asserts NO occurrence is seeded — that is what preserves
// the occurrence table's meaning as a history of real events (§4).
func TestEvaluateRejectionPaths(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*fakes)
	}{
		{"voyage already arrived", func(f *fakes) { f.route.State = "awaiting_return" }},
		{"voyage replaced by the next trip", func(f *fakes) { f.route.VoyageID = uuid.New().String() }},
		{"definition disabled since departure", func(f *fakes) { f.definition.Enabled = false }},
		{"probability roll failed", func(f *fakes) { f.roll = func() float64 { return 0.99 } }},
		{"nobody aboard", func(f *fakes) { f.charactersByMap = map[_map.Id][]uint32{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes(t)
			tc.setup(f)
			seed, err := f.handler().Evaluate(f.ctx, f.definition, f.work)
			if err != nil {
				t.Fatalf("rejection must not be an error: %v", err)
			}
			if seed != nil {
				t.Fatalf("expected no occurrence, got %+v", seed)
			}
		})
	}
}

// FR-B6: "aboard" is the UNION of attack maps and related maps — a character in
// the cabin counts.
func TestCharacterInTheCabinCountsAsAboard(t *testing.T) {
	f := newFakes(t)
	f.charactersByMap = map[_map.Id][]uint32{200090011: {42}} // cabin only

	seed, err := f.handler().Evaluate(f.ctx, f.definition, f.work)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if seed == nil {
		t.Fatalf("cabin occupancy must count as aboard (FR-B6)")
	}
}

// FR-B9/FR-B10/FR-API8: attack maps are visual, related maps are not.
func TestSuccessfulEvaluationSeedsTheCorrectScope(t *testing.T) {
	f := newFakes(t)
	f.charactersByMap = map[_map.Id][]uint32{200090010: {42}}

	seed, err := f.handler().Evaluate(f.ctx, f.definition, f.work)
	if err != nil || seed == nil {
		t.Fatalf("Evaluate: seed=%v err=%v", seed, err)
	}
	if seed.Stage != StageAttacking {
		t.Fatalf("stage = %q, want %q", seed.Stage, StageAttacking)
	}
	if seed.WorldId != 1 || seed.ChannelId != 4 || seed.VoyageId != f.voyageId {
		t.Fatalf("scope = %d/%d/%s", seed.WorldId, seed.ChannelId, seed.VoyageId)
	}
	got := map[_map.Id]bool{}
	for _, ms := range seed.Maps {
		got[ms.MapId] = ms.Visual
	}
	if !got[200090010] {
		t.Fatalf("attack map must be visual")
	}
	if got[200090011] {
		t.Fatalf("cabin must NOT be visual (FR-B13)")
	}
}

// An unreachable dependency must RETRY, not be read as a negative answer. This
// is the difference between "nobody was aboard" and "we could not tell".
func TestUnreachableDependenciesReturnErrors(t *testing.T) {
	for _, tc := range []struct{ name string; setup func(*fakes) }{
		{"transports down", func(f *fakes) { f.routeErr = errors.New("connection refused") }},
		{"maps down", func(f *fakes) { f.charactersErr = errors.New("connection refused") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes(t)
			tc.setup(f)
			if _, err := f.handler().Evaluate(f.ctx, f.definition, f.work); err == nil {
				t.Fatalf("expected an error so the work row retries")
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./events/crimsonbalrog/ -run Evaluate -v`
Expected: FAIL — `Evaluate` returns "not implemented".

- [ ] **Step 3: Implement**

```go
const StageAttacking = "ATTACKING"

// Evaluate applies FR-B5's gates in order, cheapest first. Returning
// (nil, nil) is the ORDINARY "no occurrence" outcome (FR-B7, FR-B8), not an
// error: the work row completes and no history is written, which is what keeps
// the occurrence table a record of real events.
//
// Returning an ERROR is reserved for "we could not tell" — an unreachable
// atlas-transports or atlas-maps. The work row then retries. Reading an
// unreachable atlas-maps as "nobody aboard" would silently suppress attacks
// during any maps outage.
func (h *Handler) Evaluate(ctx context.Context, d registry.Definition, w registry.Work) (*registry.Seed, error) {
	c, err := DecodeConfig(d.Configuration)
	if err != nil {
		return nil, err
	}
	var wc WorkContext
	if err := json.Unmarshal(w.Context, &wc); err != nil {
		return nil, err
	}

	// 1. Is the voyage still underway?
	route, err := h.transports(ctx).GetRoute(wc.RouteId)
	if err != nil {
		return nil, err
	}
	if !transports.VoyageUnderway(route, wc.VoyageId) {
		return nil, nil
	}

	// 2. Is the definition still enabled? (It may have been disabled between
	//    departure and now — FR-D4.)
	if !d.Enabled {
		return nil, nil
	}

	// 3. The roll.
	if h.roll() >= c.AttackProbability {
		return nil, nil
	}

	// 4. Is anyone aboard? The UNION of attack and related maps: a character in
	//    the cabin counts (FR-B6).
	aboard, err := h.anyoneAboard(ctx, c, wc)
	if err != nil {
		return nil, err
	}
	if !aboard {
		return nil, nil
	}

	return h.seed(c, wc)
}
```

`seed` builds the `registry.Seed` with `Stage: StageAttacking`, the occurrence
context (route/voyage/world/channel/attack maps/related maps/monsterId/
monsterCount/resolved spawn positions plus the visual and BGM the projection
reads — FR-B10), and the map scope rows: attack maps `Visual: true`, related
maps `Visual: false`. `ConcurrencyKey` is filled by the generic layer from
`h.ConcurrencyKey`.

Inject `roll func() float64` and the two clients on the handler so the fakes in
the test can replace them; `NewHandler()` wires the real ones.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./events/crimsonbalrog/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-events/atlas.com/events/events/crimsonbalrog/
git commit -m "feat(task-231): implement Crimson Balrog trigger evaluation"
```

---

## Task 25: `Handler.Start` — the visual event and the spawn commands

`atlas-events` constructs no packets (FR-B12). The `state`/`subState` bytes ride
on the event because they are gameplay content ("which visual"), not encoding;
`atlas-channel` turns them into `ContiMoveBody(state, subState)`.

### Files

- `services/atlas-events/atlas.com/events/kafka/message/event/kafka.go` — new; the visual event contract
- `services/atlas-events/atlas.com/events/kafka/message/monster/kafka.go` — new; mirror `SPAWN_FIELD` and `DESTROY_BY_SOURCE` as a producer
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/start.go` — new
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/producer.go` — new
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/start_test.go` — new
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go` — **read-only** source of truth for the command shapes

**Interfaces:**
- Consumes: `registry.Occurrence`, `registry.Progress` (Task 11); the provenance
  fields on `SPAWN_FIELD` (Task 2).
- Produces:
  - `event.EnvEventTopicEventVisual = "EVENT_TOPIC_EVENT_VISUAL"`,
    `event.VisualTypeShow = "SHOW"`, `event.VisualTypeHide = "HIDE"`
  - `event.VisualEvent[E]{OccurrenceId, WorldId, ChannelId, MapId, Type, Body}`
  - `event.ShowVisualBody{Visual string, State, SubState byte, Bgm string}`,
    `event.HideVisualBody{Visual string, State, SubState byte}`
  - `Handler.Start(ctx, o registry.Occurrence) (registry.Progress, error)`

- [ ] **Step 1: Write the failing test**

```go
// FR-B11/FR-B22: monsterCount monsters, in the ATTACK map only, each carrying
// the provenance pair that makes the occurrence's cleanup possible.
func TestStartSpawnsExactlyMonsterCountInTheAttackMapWithProvenance(t *testing.T) {
	f := newStartFakes(t)
	o := activeOccurrence(t, withAttackMaps(200090010), withRelatedMaps(200090011), monsterCount(2))

	if _, err := f.handler().Start(f.ctx, o); err != nil {
		t.Fatalf("Start: %v", err)
	}

	spawns := f.emitted(monster.EnvCommandTopic, monster.CommandTypeSpawnField)
	if len(spawns) != 2 {
		t.Fatalf("emitted %d spawns, want 2", len(spawns))
	}
	for _, s := range spawns {
		if s.MapId != 200090010 {
			t.Fatalf("spawn in map %d — the cabin must get no monsters (FR-B13)", s.MapId)
		}
		if s.Body.SpawnSourceType != "EVENT" || s.Body.SpawnSourceId != o.Id.String() {
			t.Fatalf("provenance = %s/%s, want EVENT/%s", s.Body.SpawnSourceType, s.Body.SpawnSourceId, o.Id)
		}
	}
	// The configured positions are used, in order — not a single position reused.
	if spawns[0].Body.X == spawns[1].Body.X && spawns[0].Body.Y == spawns[1].Body.Y {
		t.Fatalf("both monsters spawned at the same configured position")
	}
}

// FR-B11/FR-B12: the visual is an EVENT, and it carries the gameplay content
// (which visual, which state bytes, which music) rather than a packet.
func TestStartEmitsTheVisualForTheAttackMapOnly(t *testing.T) {
	f := newStartFakes(t)
	o := activeOccurrence(t, withAttackMaps(200090010), withRelatedMaps(200090011))

	if _, err := f.handler().Start(f.ctx, o); err != nil {
		t.Fatalf("Start: %v", err)
	}

	vs := f.emittedVisuals(event.VisualTypeShow)
	if len(vs) != 1 {
		t.Fatalf("emitted %d SHOW events, want 1", len(vs))
	}
	if vs[0].MapId != 200090010 {
		t.Fatalf("SHOW sent to map %d, want the attack map", vs[0].MapId)
	}
	if vs[0].Body.Visual != "CONTI_MOVE" || vs[0].Body.State != 10 || vs[0].Body.SubState != 4 {
		t.Fatalf("visual = %+v", vs[0].Body)
	}
	if vs[0].Body.Bgm != "Bgm04/ArabPirate" {
		t.Fatalf("bgm = %q", vs[0].Body.Bgm)
	}
}

// Start settles the occurrence into ATTACKING with no scheduled transition:
// completion is externally driven (monsters die, or the vessel arrives), not
// timed.
func TestStartReturnsAttackingWithNoScheduledTransition(t *testing.T) {
	f := newStartFakes(t)
	p, err := f.handler().Start(f.ctx, activeOccurrence(t))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if p.Stage != StageAttacking || p.Terminal || p.NextTransitionAt != nil {
		t.Fatalf("progress = %+v", p)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./events/crimsonbalrog/ -run Start -v`
Expected: FAIL — `Start` returns "not implemented".

- [ ] **Step 3: Define the visual event contract**

`kafka/message/event/kafka.go`:

```go
// Package event is the events service's OWN outbound contract: what should be
// rendered, not how. atlas-events never builds a packet (FR-B12) — it names the
// visual and the gameplay bytes, and atlas-channel maps them onto a writer it
// already has registered.
package event

const (
	EnvEventTopicEventVisual = "EVENT_TOPIC_EVENT_VISUAL"
	VisualTypeShow           = "SHOW"
	VisualTypeHide           = "HIDE"

	// VisualContiMove is the enemy-ship visual. The name selects the writer on
	// the channel side; the state/subState bytes are gameplay content carried
	// in the body, so a future visual needs no new event type.
	VisualContiMove = "CONTI_MOVE"
)

type VisualEvent[E any] struct {
	OccurrenceId uuid.UUID  `json:"occurrenceId"`
	WorldId      world.Id   `json:"worldId"`
	ChannelId    channel.Id `json:"channelId"`
	MapId        _map.Id    `json:"mapId"`
	Type         string     `json:"type"`
	Body         E          `json:"body"`
}

type ShowVisualBody struct {
	Visual   string `json:"visual"`
	State    byte   `json:"state"`
	SubState byte   `json:"subState"`
	Bgm      string `json:"bgm,omitempty"`
}

type HideVisualBody struct {
	Visual   string `json:"visual"`
	State    byte   `json:"state"`
	SubState byte   `json:"subState"`
}
```

Key the message on the map id so a SHOW and its later HIDE for one map land on
one partition and cannot be reordered.

- [ ] **Step 4: Implement `Start`**

```go
// Start orchestrates the side effects of a newly created occurrence, in the
// attack maps of THIS channel only (FR-B11). Order matters: the visual first,
// so a player sees the enemy ship before its monsters materialize.
func (h *Handler) Start(ctx context.Context, o registry.Occurrence) (registry.Progress, error) {
	oc, err := DecodeOccurrenceContext(o.Context)
	if err != nil {
		return registry.Progress{}, err
	}
	for _, am := range oc.AttackMaps {
		if err := h.emitShow(ctx, o, am.MapId, oc); err != nil {
			return registry.Progress{}, err
		}
		for i := uint32(0); i < oc.MonsterCount; i++ {
			pos := am.SpawnPositions[i]
			if err := h.emitSpawn(ctx, o, am.MapId, oc.MonsterId, pos); err != nil {
				return registry.Progress{}, err
			}
		}
	}
	// No NextTransitionAt: completion is externally driven — every monster dies,
	// or the vessel arrives (FR-B17). Nothing about this occurrence is timed.
	return registry.Progress{Stage: StageAttacking}, nil
}
```

`emitSpawn` builds the monster `fieldCommand[spawnFieldCommandBody]` with
`SpawnSourceType: "EVENT"`, `SpawnSourceId: o.Id.String()` (FR-B22).

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./events/crimsonbalrog/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-events/atlas.com/events/
git commit -m "feat(task-231): start a Balrog occurrence — visual event plus provenance-tagged spawns"
```

---

## Task 26: Track event-owned monsters and complete on elimination

### Files

- `services/atlas-events/atlas.com/events/kafka/message/monsterstatus/kafka.go` — new; mirror the consumed status envelope **including** the two provenance fields
- `services/atlas-events/atlas.com/events/kafka/consumer/monsterstatus/consumer.go` — new
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/monsters.go` — new
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/monsters_test.go` — new
- `services/atlas-monsters/atlas.com/monsters/monster/kafka.go` — **read-only** source of truth (Task 3)

**Interfaces:**
- Consumes: the envelope provenance echo (Task 3);
  `occurrence.Processor.ObserveMonsterSpawned/ObserveMonsterGone/MonsterTally/Complete` (Task 15).
- Produces:
  `crimsonbalrog.MonsterProcessor.OnMonsterStatus(e monsterstatus.StatusEvent[json.RawMessage]) error`.

- [ ] **Step 1: Write the failing test**

```go
// FR-B18: completion fires only when every spawned monster is accounted for AND
// none is alive. The first conjunct is what stops a completion firing in the
// window after the first spawn's CREATED but before the second's.
func TestCompletionWaitsForTheFullSpawnSet(t *testing.T) {
	db := newTestDB(t)
	o := seedActiveOccurrence(t, db, monsterCount(2))
	p := NewMonsterProcessor(testLogger(t), testCtx(t), db)

	must(t, p.OnMonsterStatus(created(o.Id(), 1)))
	must(t, p.OnMonsterStatus(killed(o.Id(), 1)))

	if state(t, db, o.Id()) != occurrence.StateActive {
		t.Fatalf("completed after one of two monsters died")
	}

	must(t, p.OnMonsterStatus(created(o.Id(), 2)))
	must(t, p.OnMonsterStatus(killed(o.Id(), 2)))

	got := readOccurrence(t, db, o.Id())
	if got.State != occurrence.StateCompleted || got.CompletionReason != "MONSTERS_ELIMINATED" {
		t.Fatalf("state=%s reason=%s", got.State, got.CompletionReason)
	}
}

// A monster with someone else's provenance, or none, is ignored entirely.
func TestForeignMonstersAreIgnored(t *testing.T) {
	db := newTestDB(t)
	o := seedActiveOccurrence(t, db, monsterCount(1))
	p := NewMonsterProcessor(testLogger(t), testCtx(t), db)

	must(t, p.OnMonsterStatus(createdWithSource(uuid.New().String(), 9)))
	must(t, p.OnMonsterStatus(cyclicCreated(10)))

	total, alive, err := occurrence.NewProcessor(testLogger(t), testCtx(t), db).MonsterTally(o.Id())
	if err != nil {
		t.Fatalf("MonsterTally: %v", err)
	}
	if total != 0 || alive != 0 {
		t.Fatalf("foreign monsters were tracked: total=%d alive=%d", total, alive)
	}
}

// FR-B18 cleanup: the visual is removed on this path. The BGM is deliberately
// NOT restored — see design §15.4: atlas-data does not expose Map.wz info/bgm,
// so no service can name the map's default, and "restore the default" would
// mean hard-coding a guessed string.
func TestEliminationCleanupHidesTheVisualAndLeavesTheMusic(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedActiveOccurrence(t, db, monsterCount(1), withAttackMaps(200090010))
	p := NewMonsterProcessorWith(testLogger(t), testCtx(t), db, f)

	must(t, p.OnMonsterStatus(created(o.Id(), 1)))
	must(t, p.OnMonsterStatus(killed(o.Id(), 1)))

	hides := f.emittedVisuals(event.VisualTypeHide)
	if len(hides) != 1 || hides[0].MapId != 200090010 {
		t.Fatalf("HIDE = %+v", hides)
	}
	if hides[0].Body.State != 10 || hides[0].Body.SubState != 5 {
		t.Fatalf("hide visual = %+v, want CONTI_MOVE(10,5)", hides[0].Body)
	}
	if len(f.emitted(monster.EnvCommandTopic, monster.CommandTypeDestroyBySource)) != 0 {
		t.Fatalf("elimination must not issue DESTROY_BY_SOURCE — nothing is left")
	}
}

// Redelivery of the final KILLED must not complete twice or re-run cleanup.
func TestRedeliveredFinalKillIsANoOp(t *testing.T) { ... }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./events/crimsonbalrog/ -run Monster -v`
Expected: FAIL — `NewMonsterProcessor` undefined.

- [ ] **Step 3: Implement**

```go
// OnMonsterStatus maintains the occurrence's monster SET from the provenance
// echoed on the status envelope (design §8.2). Only CREATED, KILLED and
// DESTROYED are consumed; every other type on the topic is ignored.
//
// CREATED is insert-if-absent, not upsert: a KILLED that arrived first already
// wrote a dead row, and the late CREATED must not resurrect it. The two events
// share a topic but have no ordering guarantee across partitions (design §9.5).
func (p *MonsterProcessorImpl) OnMonsterStatus(e monsterstatus.StatusEvent[json.RawMessage]) error {
	if e.SpawnSourceType != monsterSourceEvent {
		return nil
	}
	occurrenceId, err := uuid.Parse(e.SpawnSourceId)
	if err != nil {
		return nil // not one of ours
	}
	op := occurrence.NewProcessor(p.l, p.ctx, p.db)
	o, err := op.GetById(occurrenceId)
	if err != nil {
		return nil // completed and swept, or another service's id space
	}
	if o.Type() != TypeName || o.State() != occurrence.StateActive {
		return nil
	}

	switch e.Type {
	case monsterstatus.EventMonsterStatusCreated:
		return op.ObserveMonsterSpawned(occurrenceId, e.UniqueId, e.MonsterId)
	case monsterstatus.EventMonsterStatusKilled, monsterstatus.EventMonsterStatusDestroyed:
		if err := op.ObserveMonsterGone(occurrenceId, e.UniqueId, e.MonsterId); err != nil {
			return err
		}
		return p.completeIfEliminated(o)
	}
	return nil
}

// completeIfEliminated fires only when the FULL spawn set is accounted for and
// none is alive. Without the total check a completion could fire in the window
// after the first spawn's CREATED but before the second's (FR-B18).
func (p *MonsterProcessorImpl) completeIfEliminated(o occurrence.Model) error {
	oc, err := DecodeOccurrenceContext(o.Context())
	if err != nil {
		return err
	}
	op := occurrence.NewProcessor(p.l, p.ctx, p.db)
	total, alive, err := op.MonsterTally(o.Id())
	if err != nil {
		return err
	}
	if uint32(total) < oc.MonsterCount*uint32(len(oc.AttackMaps)) || alive > 0 {
		return nil
	}

	won, err := op.Complete(o.Id(), ReasonMonstersEliminated, transition.TriggerTypeMonsterKilled, "")
	if err != nil {
		return err
	}
	if !won {
		// Another path completed it first. Cleanup already ran; running it
		// again is not merely wasteful, it would emit a second HIDE (FR-B20).
		return nil
	}
	return p.hideVisuals(o, oc)
}
```

`hideVisuals` emits `HIDE` with the configured hide pair for each attack map.
It does **not** emit `DESTROY_BY_SOURCE` — on this path nothing is left — and it
does **not** restore the BGM (design §15.4).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./events/crimsonbalrog/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-events/atlas.com/events/
git commit -m "feat(task-231): track event-owned monsters and complete on elimination"
```

---

## Task 27: Complete on voyage arrival

### Files

- `services/atlas-events/atlas.com/events/kafka/consumer/transport/consumer.go` — add the `VOYAGE_ARRIVED` handler
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/arrival.go` — new
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/handler.go` — replace the `Complete` and `Advance` stubs
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/arrival_test.go` — new

**Interfaces:**
- Consumes: `VOYAGE_ARRIVED` (Task 6); `DESTROY_BY_SOURCE` (Task 4);
  `occurrence.Processor.GetActiveByVoyage`/`Complete` (Task 15).
- Produces:
  `crimsonbalrog.ArrivalProcessor.OnVoyageArrived(e transport.StatusEvent[transport.VoyageStatusEventBody]) error`;
  `Handler.Complete(ctx, o registry.Occurrence, reason string) error`;
  `Handler.Advance` returns a clear error — this event schedules no transitions,
  so a work row of that type is a bug, not a silent no-op.

- [ ] **Step 1: Write the failing test**

```go
// FR-B19: arrival despawns everything remaining, removes the visual and
// completes with VESSEL_ARRIVED.
func TestArrivalCompletesAndCleansUp(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedActiveOccurrence(t, db, withVoyage(voyageId, 1, 4), withAttackMaps(200090010))

	must(t, NewArrivalProcessorWith(testLogger(t), testCtx(t), db, f).OnVoyageArrived(arrived(voyageId, 1, 4)))

	got := readOccurrence(t, db, o.Id())
	if got.State != occurrence.StateCompleted || got.CompletionReason != "VESSEL_ARRIVED" {
		t.Fatalf("state=%s reason=%s", got.State, got.CompletionReason)
	}
	ds := f.emitted(monster.EnvCommandTopic, monster.CommandTypeDestroyBySource)
	if len(ds) != 1 || ds[0].MapId != 200090010 {
		t.Fatalf("DESTROY_BY_SOURCE = %+v", ds)
	}
	if ds[0].Body.SpawnSourceId != o.Id().String() {
		t.Fatalf("destroy targeted %q, want the occurrence id", ds[0].Body.SpawnSourceId)
	}
	if len(f.emittedVisuals(event.VisualTypeHide)) != 1 {
		t.Fatalf("expected one HIDE")
	}
}

// FR-B20: arrival cleanup must succeed when every Balrog was killed a second
// earlier. Zero matches is the ORDINARY case, not an error path.
func TestArrivalAfterEliminationIsANoOp(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedCompletedOccurrence(t, db, "MONSTERS_ELIMINATED", withVoyage(voyageId, 1, 4))

	must(t, NewArrivalProcessorWith(testLogger(t), testCtx(t), db, f).OnVoyageArrived(arrived(voyageId, 1, 4)))

	got := readOccurrence(t, db, o.Id())
	if got.CompletionReason != "MONSTERS_ELIMINATED" {
		t.Fatalf("reason overwritten to %q — the first completion must win", got.CompletionReason)
	}
	if len(f.emittedVisuals(event.VisualTypeHide)) != 0 {
		t.Fatalf("cleanup ran twice")
	}
}

// FR-N11: an arrival in world 1 channel 4 must not touch an occurrence in
// channel 5 of the same voyage.
func TestArrivalIsScopedToItsChannel(t *testing.T) { ... }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./events/crimsonbalrog/ -run Arrival -v`
Expected: FAIL — `NewArrivalProcessor` undefined.

- [ ] **Step 3: Implement**

```go
// OnVoyageArrived completes any ACTIVE occurrence belonging to this
// (voyageId, worldId, channelId). Completion is the GUARDED update, so if the
// elimination path already won, `won` is false and cleanup is skipped — which
// is exactly FR-B20's "the two completion paths racing produce exactly one
// completion".
func (p *ArrivalProcessorImpl) OnVoyageArrived(e transport.StatusEvent[transport.VoyageStatusEventBody]) error {
	op := occurrence.NewProcessor(p.l, p.ctx, p.db)
	os, err := op.GetActiveByVoyage(e.Body.VoyageId, e.Body.WorldId, e.Body.ChannelId)
	if err != nil {
		return err
	}
	for _, o := range os {
		if o.Type() != TypeName {
			continue
		}
		won, err := op.Complete(o.Id(), ReasonVesselArrived, transition.TriggerTypeVoyageArrived, e.Body.VoyageId.String())
		if err != nil {
			return err
		}
		if !won {
			continue
		}
		if err := p.cleanup(o); err != nil {
			return err
		}
	}
	return nil
}

// cleanup despawns everything the occurrence owns, then removes the visual.
// Zero surviving monsters is success (FR-P4) — the destroy command is issued
// unconditionally rather than gated on a tally, because the tally can only be
// stale and issuing it costs one message.
func (p *ArrivalProcessorImpl) cleanup(o occurrence.Model) error { ... }
```

`Handler.Complete` delegates to the same `cleanup`, so a completion driven by
the generic layer behaves identically to one driven by the consumer.
`Handler.Advance` returns
`fmt.Errorf("crimsonbalrog: unexpected OCCURRENCE_TRANSITION work for occurrence %s", o.Id)`
— this event schedules none, so such a row is a bug that must surface as a
`FAILED` work row with a named reason, not be swallowed.

- [ ] **Step 4: Register the consumer and the handler**

Add `handleVoyageArrived` to `kafka/consumer/transport/consumer.go`, guarded on
`transport.EventStatusVoyageArrived`. In `main.go`, add
`registry.Register(crimsonbalrog.NewHandler())` and the monster-status consumer
from Task 26. Confirm no `errors.New("not implemented")` remains anywhere in
`events/crimsonbalrog/`:

```bash
grep -rn "not implemented\|TODO\|FIXME" services/atlas-events/atlas.com/events/
```

Expected: no output.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-events/atlas.com/events/
git commit -m "feat(task-231): complete a Balrog occurrence on voyage arrival"
```

---

## Task 28: `atlas-channel` renders the visual

**Module root:** `services/atlas-channel/atlas.com/channel`

No packet work: `ContiMoveWriter` is registered at
`services/atlas-channel/atlas.com/channel/main.go:778`, `FieldEffectWriter` at
`:803`, and `ContiMoveBody(state, subState)` already exists at
`socket/writer/conti_move.go` (PRD F3).

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/event/kafka.go` — new; mirror the visual contract from Task 25
- `services/atlas-channel/atlas.com/channel/kafka/consumer/event/consumer.go` — new
- `services/atlas-channel/atlas.com/channel/kafka/consumer/event/consumer_test.go` — new
- `services/atlas-channel/atlas.com/channel/main.go` — register the consumer and handlers alongside the other `InitConsumers`/`InitHandlers` pairs
- `services/atlas-channel/atlas.com/channel/kafka/consumer/route/consumer.go` — **read-only** reference for the consumer shape and the `sc.Is(...)` guard
- `services/atlas-channel/atlas.com/channel/socket/writer/conti_move.go` — **read-only**

**Interfaces:**
- Consumes: `EVENT_TOPIC_EVENT_VISUAL` (Task 25).
- Produces: `ContiMoveBody(state, subState)` and
  `fieldpkt.FieldEffectBackgroundMusicBody(bgm)` broadcast to the map via
  `_map.NewProcessor(l, ctx).ForSessionsInMap`.

- [ ] **Step 1: Write the failing test**

```go
// The visual is broadcast to the named map only, and only when the event is for
// THIS channel — the same sc.Is guard every other channel consumer applies.
func TestShowVisualBroadcastsContiMoveToTheMap(t *testing.T) {
	sc := serverFor(t, world(1), channel(4))
	sent := captureAnnounce(t)

	handleVisualShow(sc, sent.Producer())(testLogger(t), testCtx(t), showEvent(world(1), channel(4), 200090010, 10, 4, "Bgm04/ArabPirate"))

	if got := sent.CountFor(200090010, fieldcb.ContiMoveWriter); got != 1 {
		t.Fatalf("ContiMove announced %d times, want 1", got)
	}
	if got := sent.CountFor(200090010, fieldcb.FieldEffectWriter); got != 1 {
		t.Fatalf("FieldEffect (bgm) announced %d times, want 1", got)
	}
}

func TestShowVisualIgnoresOtherChannels(t *testing.T) {
	sc := serverFor(t, world(1), channel(4))
	sent := captureAnnounce(t)

	handleVisualShow(sc, sent.Producer())(testLogger(t), testCtx(t), showEvent(world(1), channel(5), 200090010, 10, 4, ""))

	if sent.Total() != 0 {
		t.Fatalf("announced %d packets for another channel", sent.Total())
	}
}

// A SHOW with no bgm configured sends the visual and nothing else.
func TestShowVisualWithoutBgmSendsOnlyTheVisual(t *testing.T) { ... }

func TestHideVisualBroadcastsContiMove(t *testing.T) { ... }
```

Read the channel's existing consumer tests for the announce-capture harness and
reuse it; if none exists, follow
`services/atlas-buffs/atlas.com/buffs/kafka/consumer/skillstatus/consumer_test.go`'s
shape.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/event/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement the consumer**

```go
// handleVisualShow renders an event's visual for everyone currently in the map.
// atlas-events named the visual and its gameplay bytes; this consumer's whole
// job is to map that onto writers the channel already has registered — it makes
// no decision about whether the visual should be shown.
func handleVisualShow(sc server.Model, wp writer.Producer) message.Handler[event2.VisualEvent[event2.ShowVisualBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e event2.VisualEvent[event2.ShowVisualBody]) {
		if e.Type != event2.VisualTypeShow {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		if e.Body.Visual != event2.VisualContiMove {
			l.Warnf("Unknown visual [%s] for occurrence [%s]; ignoring.", e.Body.Visual, e.OccurrenceId)
			return
		}
		if err := _map.NewProcessor(l, ctx).ForSessionsInMap(e.WorldId, e.ChannelId, e.MapId,
			session.Announce(l)(ctx)(wp)(fieldcb.ContiMoveWriter)(writer.ContiMoveBody(e.Body.State, e.Body.SubState))); err != nil {
			l.WithError(err).Errorf("Unable to broadcast event visual to map [%d].", e.MapId)
		}
		if e.Body.Bgm == "" {
			return
		}
		if err := _map.NewProcessor(l, ctx).ForSessionsInMap(e.WorldId, e.ChannelId, e.MapId,
			session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWriter)(fieldpkt.FieldEffectBackgroundMusicBody(e.Body.Bgm))); err != nil {
			l.WithError(err).Errorf("Unable to broadcast event music to map [%d].", e.MapId)
		}
	}
}
```

Confirm the exact `sc.Is` signature and the `ForSessionsInMap` variant against
`kafka/consumer/route/consumer.go` before writing — that file uses
`ForSessionsInMapAllInstances`; pick whichever matches the map-scoped semantics
the other map-broadcast consumers in this service use.

- [ ] **Step 4: Register in `main.go`**

Add `eventConsumer.InitConsumers(l)(cmf)(consumerGroupId)` and the
`InitHandlers` call next to the route consumer's.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(task-231): render event visuals in atlas-channel"
```

---

## Task 29: Map entry sends the visual to an arriving character

**Module root:** `services/atlas-channel/atlas.com/channel`

Per design G6 this is one more `routine.Go` block in the run at
`kafka/consumer/map/consumer.go:188-350` — the same shape as the existing
`IsBoatInMap` and `weather.GetActive` blocks. Non-blocking and fail-open are
properties of that existing shape (a panic-recovering goroutine whose error path
only logs), not new machinery.

### Files

- `services/atlas-channel/atlas.com/channel/events/{processor,requests,rest}.go` — new REST client
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:~350` — one new `routine.Go` block in `SpawnForSelf`
- `services/atlas-channel/atlas.com/channel/events/processor_test.go` — new
- `services/atlas-channel/atlas.com/channel/weather/` — **read-only** reference for the client shape

**Interfaces:**
- Consumes: `GET /events/worlds/{worldId}/channels/{channelId}/maps/{mapId}/visuals` (Task 16).
- Produces: `events.Processor.ActiveVisualsInMap(f field.Model) ([]RestModel, error)`;
  `events.RestModel{OccurrenceId, Visual, State, SubState, Bgm}`.

- [ ] **Step 1: Write the failing test**

```go
// FR-B16/FR-N15: an unreachable atlas-events costs the visual and nothing else.
// The lookup must never surface as an error that aborts map entry.
func TestActiveVisualsInMapFailsOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("BASE_SERVICE_URL", srv.URL+"/api/")

	_, err := NewProcessor(testLogger(t), testCtx(t)).ActiveVisualsInMap(field.NewBuilder(1, 4, 200090010).Build())
	if err == nil {
		t.Fatalf("expected the transport error to be returned so the CALLER can log and move on")
	}
}

func TestActiveVisualsInMapDecodesTheProjection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"type":"event-visuals","id":"1","attributes":{"occurrenceId":"o1","visual":"CONTI_MOVE","state":10,"subState":4,"bgm":"Bgm04/ArabPirate"}}]}`))
	}))
	defer srv.Close()
	t.Setenv("BASE_SERVICE_URL", srv.URL+"/api/")

	vs, err := NewProcessor(testLogger(t), testCtx(t)).ActiveVisualsInMap(field.NewBuilder(1, 4, 200090010).Build())
	if err != nil {
		t.Fatalf("ActiveVisualsInMap: %v", err)
	}
	if len(vs) != 1 || vs[0].State != 10 || vs[0].SubState != 4 || vs[0].Bgm != "Bgm04/ArabPirate" {
		t.Fatalf("decoded %+v", vs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./events/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the client**

Copy `services/atlas-channel/atlas.com/channel/weather/` and change the resource
path and `RootUrl("EVENTS")`.

- [ ] **Step 4: Add the `SpawnForSelf` block**

At the end of the `routine.Go` run in
`kafka/consumer/map/consumer.go`, after the weather block:

```go
		routine.Go(l, ctx, func(_ context.Context) {
			vs, verr := events.NewProcessor(l, ctx).ActiveVisualsInMap(f)
			if verr != nil {
				// Fail open (FR-B16, FR-N15): the character entered the map;
				// an unreachable atlas-events costs the visual, not the entry.
				l.WithError(verr).Debugf("SpawnForSelf: unable to retrieve active event visuals for map [%d].", f.MapId())
				return
			}
			for _, v := range vs {
				if v.Visual != "CONTI_MOVE" {
					continue
				}
				_ = session.Announce(l)(ctx)(wp)(fieldcb.ContiMoveWriter)(writer.ContiMoveBody(v.State, v.SubState))(s)
				if v.Bgm != "" {
					_ = session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWriter)(fieldpkt.FieldEffectBackgroundMusicBody(v.Bgm))(s)
				}
			}
		})
```

The cabin returns no rows because its child row has `visual = false` (FR-B13) —
there is no branch here for it, which is the point of the projection.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(task-231): send active event visuals to a character entering a map"
```

---

## Task 30: Pin FR-V6, and check the `ContiMove` template routing

**Module root:** `services/atlas-channel/atlas.com/channel`

Per design G2 the route consumer already guards on type
(`kafka/consumer/route/consumer.go:62` and `:82`), so FR-V6 needs **no consumer
edit** — it needs a test that stops a future edit from breaking it. This is
exactly the class of seam CLAUDE.md's "green verify.sh ≠ correct" rule names.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/route/consumer_test.go` — new
- `docs/tasks/task-231-generalized-events-service/template-routing.md` — new; the ContiMove routing evidence
- `services/atlas-configurations/seed-data/templates/*.json` — read; edit only if the check below finds a gap

**Interfaces:**
- Consumes: the two new transport event types (Task 6).
- Produces: a regression test and a recorded routing check.

- [ ] **Step 1: Write the test**

```go
// FR-V6 / acceptance 20.4: the two new transport event types must remain inert
// here. The type guards already do this today — this test is what stops a
// future edit (a new handler, a relaxed guard) from silently turning a voyage
// event into a CONTI_STATE broadcast.
func TestRouteConsumerIgnoresVoyageEventTypes(t *testing.T) {
	sc := serverFor(t, world(1), channel(4))
	sent := captureAnnounce(t)

	for _, theType := range []string{"VOYAGE_DEPARTED", "VOYAGE_ARRIVED"} {
		handleStatusEventArrived(sc, sent.Producer())(testLogger(t), testCtx(t),
			route2.StatusEvent[route2.ArrivedStatusEventBody]{Type: theType, Body: route2.ArrivedStatusEventBody{MapId: 200090010}})
		handleStatusEventDeparted(sc, sent.Producer())(testLogger(t), testCtx(t),
			route2.StatusEvent[route2.DepartedStatusEventBody]{Type: theType, Body: route2.DepartedStatusEventBody{MapId: 200090010}})
	}

	if sent.Total() != 0 {
		t.Fatalf("voyage event types produced %d packets, want 0", sent.Total())
	}
}

// And the existing behavior still fires — the guard must not have been
// tightened into silence.
func TestRouteConsumerStillBroadcastsArrivedAndDeparted(t *testing.T) {
	sc := serverFor(t, world(1), channel(4))
	sent := captureAnnounce(t)

	handleStatusEventArrived(sc, sent.Producer())(testLogger(t), testCtx(t),
		route2.StatusEvent[route2.ArrivedStatusEventBody]{Type: route2.EventStatusArrived, Body: route2.ArrivedStatusEventBody{MapId: 200090010}})

	if sent.CountFor(200090010, fieldcb.FieldTransportStateWriter) != 1 {
		t.Fatalf("ARRIVED no longer broadcasts CONTI_STATE")
	}
}
```

- [ ] **Step 2: Run the test**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/route/ -v`
Expected: PASS immediately — the guards already exist. If it fails, the guard
was removed and that is a real regression to fix here.

- [ ] **Step 3: Check the template routing and record the evidence**

PRD §15 says to confirm `ContiMove` routing during design rather than trusting
F3's "all nine templates". Run:

```bash
ls services/atlas-configurations/seed-data/templates/
grep -c '"writer"' services/atlas-configurations/seed-data/templates/*.json
grep -l '"ContiMove"' services/atlas-configurations/seed-data/templates/*.json
```

The check performed while writing this plan found **eleven** template files, of
which **six** route `ContiMove`: `gms_79`, `gms_83`, `gms_84`, `gms_87`,
`gms_95`, `jms_185`. Those six are exactly the versions
`libs/atlas-packet/field/clientbound/conti_move.go` carries verify markers for.
The five that do not — `gms_12`, `gms_48`, `gms_61`, `gms_72`, `gms_92` — are
partial bring-ups generally (`gms_92` routes 135 writers against `gms_83`'s
220), so the absence is a version-bring-up gap, not a regression this task
introduces.

Write `docs/tasks/task-231-generalized-events-service/template-routing.md` with
the **verbatim** output of the three commands above and that conclusion. Then:

- If any template whose version has a **verified** `ContiMove` codec cell lacks
  the route, add it — that is a silent-drop bug and it is producible here.
- If the gap is only in the five partial templates, add nothing: routing an
  opcode there requires deriving it per version from the IDB, which is
  `/bringup-version` work with its own playbook. Say so in the file.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/route/consumer_test.go \
        docs/tasks/task-231-generalized-events-service/template-routing.md
git commit -m "test(task-231): pin that the route consumer ignores the voyage event types"
```

---

# Phase F — Event 2: Anniversary, and `atlas-buffs` correlation (design §10)

## Task 31: Carry an event correlation on an applied buff

**Module root:** `services/atlas-buffs/atlas.com/buffs`

FR-A12 needs the buff to remember which occurrence granted it, so completion can
cancel exactly those buffs without relying on buff identity alone.

### Files

- `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` — `CorrelationId` on `ApplyCommandBody`
- `services/atlas-buffs/atlas.com/buffs/buff/model.go` — `correlationId` field, accessor, `MarshalJSON`/`UnmarshalJSON`, `NewBuff`/`NewNoExpiryBuff`
- `services/atlas-buffs/atlas.com/buffs/character/registry.go:70` — `Apply` takes and stores it
- `services/atlas-buffs/atlas.com/buffs/character/processor.go:65` — `Apply` threads it
- `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go:53` — `handleApply` passes it
- `services/atlas-buffs/atlas.com/buffs/buff/model_test.go` — new file if absent

**Interfaces:**
- Produces:
  - `character.ApplyCommandBody.CorrelationId string \`json:"correlationId,omitempty"\``
  - `buff.Model.CorrelationId() string`
  - `(*Registry).Apply(ctx, worldId, channelId, characterId, sourceId, level, duration, changes, accumulate, noExpiry bool, correlationId string) ([]buff.Model, error)`
  - `(*ProcessorImpl).Apply(worldId, channelId, characterId, fromId, sourceId, level, duration, changes, accumulate, noExpiry bool, correlationId string) error`

- [ ] **Step 1: Write the failing test**

```go
package buff

import (
	"encoding/json"
	"testing"

	"atlas-buffs/buff/stat"
)

func TestCorrelationSurvivesTheRedisRoundTrip(t *testing.T) {
	b, err := NewBuff(9000, 1, 60000, []stat.Model{stat.NewStat("EXP_BUFF_RATE", 200)}, "occ-1")
	if err != nil {
		t.Fatalf("NewBuff: %v", err)
	}

	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Model
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.CorrelationId() != "occ-1" {
		t.Fatalf("correlation lost: %q", back.CorrelationId())
	}
}

// An existing buff in Redis has no correlationId key. It must unmarshal to
// empty — no migration, no backfill.
func TestLegacyBuffPayloadUnmarshalsWithEmptyCorrelation(t *testing.T) {
	raw := `{"id":"11111111-1111-1111-1111-111111111111","sourceId":9000,"level":1,"duration":60000,"changes":[],"createdAt":"2026-08-15T00:00:00Z","expiresAt":"2026-08-15T00:01:00Z"}`
	var m Model
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.CorrelationId() != "" {
		t.Fatalf("correlation = %q, want empty", m.CorrelationId())
	}
}

// An ordinary buff (no correlation) must marshal byte-identically to today.
func TestUncorrelatedBuffMarshalsWithoutTheKey(t *testing.T) {
	b, _ := NewBuff(9000, 1, 60000, []stat.Model{stat.NewStat("EXP_BUFF_RATE", 200)}, "")
	raw, _ := json.Marshal(b)
	if strings.Contains(string(raw), "correlationId") {
		t.Fatalf("uncorrelated buff carries the key: %s", raw)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./buff/ -v`
Expected: FAIL — `NewBuff` takes 4 arguments.

- [ ] **Step 3: Implement**

Add `correlationId string` to `buff.Model` with:

```go
// CorrelationId names the thing that granted this buff, when something other
// than a skill did — an event occurrence id, today. It is opaque to
// atlas-buffs: the service stores it, echoes it, and matches it for equality so
// the granter can cancel exactly what it granted (FR-A12), without knowing what
// an event is.
func (m Model) CorrelationId() string { return m.correlationId }
```

Add `CorrelationId string \`json:"correlationId,omitempty"\`` to both anonymous
structs in `MarshalJSON`/`UnmarshalJSON` (`buff/model.go:92` and `:114`) —
`omitempty` is what keeps an ordinary buff's Redis payload byte-identical.
Widen `NewBuff` and `NewNoExpiryBuff` with a trailing `correlationId string`,
then follow the compiler through `Registry.Apply`, `ProcessorImpl.Apply` and
`handleApply`. `handleApply` passes `c.Body.CorrelationId`.

Also add the pointer comment on the new command field, matching the convention
the `Duration` field already uses:

```go
	// CorrelationId identifies what granted this buff, for cancel-by-correlation
	// (FR-A12). Opaque to atlas-buffs. Optional — omitting it leaves every
	// existing producer's bytes unchanged.
	CorrelationId string `json:"correlationId,omitempty"`
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-buffs/atlas.com/buffs && go build ./... && go test ./...`
Expected: PASS, with every pre-existing test unmodified.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/
git commit -m "feat(task-231): carry an opaque correlation id on applied buffs"
```

---

## Task 32: `CANCEL_BY_CORRELATION`

**Module root:** `services/atlas-buffs/atlas.com/buffs`

One command, not one per character, so completion cost does not scale with the
online population (FR-A15).

### Files

- `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` — `CommandTypeCancelByCorrelation` + body
- `services/atlas-buffs/atlas.com/buffs/character/registry.go` — `CancelByCorrelation`
- `services/atlas-buffs/atlas.com/buffs/character/processor.go` — `CancelByCorrelation`, modelled on `CancelAll` (line ~125)
- `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go` — `handleCancelByCorrelation` + registration
- `services/atlas-buffs/atlas.com/buffs/character/processor_test.go` — extend

**Interfaces:**
- Consumes: `buff.Model.CorrelationId()` (Task 31);
  `(*Registry).GetCharacters(ctx)` (`character/registry.go:147`).
- Produces: `"CANCEL_BY_CORRELATION"` on `COMMAND_TOPIC_CHARACTER_BUFF` with
  `CancelByCorrelationCommandBody{CorrelationId string}`;
  `(*ProcessorImpl).CancelByCorrelation(correlationId string) error`.

- [ ] **Step 1: Write the failing test**

```go
// FR-A15: one command cancels the occurrence's buff for every affected online
// character, without waiting for logout or expiry.
func TestCancelByCorrelationSweepsEveryCharacter(t *testing.T) {
	p := newTestProcessor(t)
	must(t, p.Apply(1, 4, 100, 0, 9000, 1, 60000, expBuff(), false, false, "occ-1"))
	must(t, p.Apply(1, 5, 200, 0, 9000, 1, 60000, expBuff(), false, false, "occ-1"))
	must(t, p.Apply(1, 4, 300, 0, 9001, 1, 60000, expBuff(), false, false, "occ-2"))
	must(t, p.Apply(1, 4, 400, 0, 1004, 1, 60000, holySymbol(), false, false, ""))

	if err := p.CancelByCorrelation("occ-1"); err != nil {
		t.Fatalf("CancelByCorrelation: %v", err)
	}

	if buffsFor(t, 100) != 0 || buffsFor(t, 200) != 0 {
		t.Fatalf("correlated buffs survived the sweep")
	}
	if buffsFor(t, 300) != 1 || buffsFor(t, 400) != 1 {
		t.Fatalf("uncorrelated or differently-correlated buffs were cancelled")
	}
}

// One EXPIRED per removed buff, carrying that character's OWN world — the
// command envelope's world is not authoritative for a tenant-wide sweep.
func TestCancelByCorrelationEmitsExpiredPerRemovedBuff(t *testing.T) { ... }

// An empty correlation id matches nothing, rather than cancelling every
// uncorrelated buff in the tenant.
func TestCancelByEmptyCorrelationMatchesNothing(t *testing.T) {
	p := newTestProcessor(t)
	must(t, p.Apply(1, 4, 100, 0, 1004, 1, 60000, holySymbol(), false, false, ""))

	if err := p.CancelByCorrelation(""); err != nil {
		t.Fatalf("CancelByCorrelation: %v", err)
	}
	if buffsFor(t, 100) != 1 {
		t.Fatalf("an empty correlation cancelled an uncorrelated buff")
	}
}

// Re-issuing the command after the sweep is a no-op (FR-N4).
func TestCancelByCorrelationIsIdempotent(t *testing.T) { ... }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run Correlation -v`
Expected: FAIL — `CancelByCorrelation` undefined.

- [ ] **Step 3: Implement**

```go
// CancelByCorrelation removes every buff in the tenant carrying correlationId,
// across all worlds and channels, and emits one EXPIRED per removed buff so
// each client clears its icon. It is ONE command rather than one per character
// (FR-A15), so an event's completion cost does not scale with the online
// population.
//
// An empty correlationId matches NOTHING. Guarding here rather than in the
// registry query is deliberate: an accidental empty id would otherwise cancel
// every uncorrelated buff on the server.
func (p *ProcessorImpl) CancelByCorrelation(correlationId string) error {
	if correlationId == "" {
		return nil
	}
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, c := range GetRegistry().GetCharacters(p.ctx) {
			cancelled, err := GetRegistry().CancelByCorrelation(p.ctx, c.Id(), correlationId)
			if err != nil {
				return err
			}
			for _, b := range cancelled {
				// c.WorldId(), not the command envelope's: the sweep is
				// tenant-wide and each character's own world is authoritative.
				if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(
					c.WorldId(), c.Id(), b.SourceId(), b.Level(), b.Duration(),
					b.Changes(), b.CreatedAt(), b.ExpiresAt(), b.NoExpiry())); err != nil {
					return err
				}
			}
			sets := make([][]stat.Model, 0, len(cancelled))
			for _, b := range cancelled {
				sets = append(sets, b.Changes())
			}
			GetRegistry().ClearPeriodicTicksFor(p.ctx, c.Id(), sets...)
			markBerserkDirtyOnMaxHpChange(p.l, p.ctx, c.Id(), sets...)
		}
		return nil
	})
}
```

`Registry.CancelByCorrelation` mirrors `Registry.CancelByStatTypes`
(`character/registry.go:238`) — same read-modify-write shape, matching on
`CorrelationId()` instead of stat type.

The consumer handler ignores the envelope's `CharacterId` entirely and must say
so:

```go
// handleCancelByCorrelation sweeps the whole tenant; the envelope's
// characterId and worldId are not consulted. Emitters send a single command
// with characterId 0.
func handleCancelByCorrelation(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.CancelByCorrelationCommandBody]) {
	if c.Type != character2.CommandTypeCancelByCorrelation {
		return
	}
	if err := character.NewProcessor(l, ctx).CancelByCorrelation(c.Body.CorrelationId); err != nil {
		l.WithError(err).Errorf("Unable to cancel buffs for correlation [%s].", c.Body.CorrelationId)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-buffs/atlas.com/buffs && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/
git commit -m "feat(task-231): add CANCEL_BY_CORRELATION to atlas-buffs"
```

---

## Task 33: `events/anniversary` — configuration, scheduling and completion

**Module root:** `services/atlas-events/atlas.com/events`

### Files

- `services/atlas-events/atlas.com/events/events/anniversary/config.go` — new
- `services/atlas-events/atlas.com/events/events/anniversary/handler.go` — new (all seven `registry.Handler` methods; no stubs)
- `services/atlas-events/atlas.com/events/events/anniversary/schedule.go` — new
- `services/atlas-events/atlas.com/events/events/anniversary/handler_test.go` — new
- `services/atlas-events/atlas.com/events/kafka/message/buff/kafka.go` — new; the `COMMAND_TOPIC_CHARACTER_BUFF` producer contract
- `services/atlas-events/atlas.com/events/event/definition/processor.go` — `SetEnabled` schedules one generic `TRIGGER_EVALUATION` on a false→true transition (see Step 3; no type switch)
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/evaluate.go` + `evaluate_test.go` — the empty-work-context guard that change requires
- `services/atlas-configurations/seed-data/events/definitions/event-anniversary.json` — fill in the configuration

**Interfaces:**
- Consumes: `registry.Handler` (Task 11), `scheduling.Administrator` (Task 17),
  `occurrence.Processor` (Task 15), the buff `CorrelationId` (Task 31) and
  `CANCEL_BY_CORRELATION` (Task 32).
- Produces:
  - `anniversary.TypeName = "ANNIVERSARY"`
  - `anniversary.Config{ScheduledStart, ScheduledEnd time.Time, ExpMultiplier, DropMultiplier float64, BuffSourceId int32}`
  - `anniversary.ReasonScheduledEnd = "SCHEDULED_END"`
  - `anniversary.Scheduler.OnDefinitionEnabled(d definition.Model) error`

- [ ] **Step 1: Write the failing test**

```go
// FR-A2/FR-A6: enabling schedules the start; a start already past with an end
// still future schedules it for NOW — still a durable row, picked up by the
// ordinary poll. There is no special synchronous path.
func TestEnablingSchedulesTheStart(t *testing.T) {
	for _, tc := range []struct {
		name       string
		start, end time.Time
		wantRows   int
		wantAtNow  bool
	}{
		{"future window", now.Add(time.Hour), now.Add(48 * time.Hour), 1, false},
		{"already started", now.Add(-time.Hour), now.Add(48 * time.Hour), 1, true},
		{"window fully elapsed", now.Add(-48 * time.Hour), now.Add(-time.Hour), 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			d := seedDefinition(t, db, TypeName, enabled(true), window(tc.start, tc.end))

			must(t, NewScheduler(testLogger(t), testCtx(t), db).OnDefinitionEnabled(d))

			work := readAllWork(t, db)
			if len(work) != tc.wantRows {
				t.Fatalf("scheduled %d rows, want %d", len(work), tc.wantRows)
			}
			if tc.wantAtNow && work[0].ExecuteAt.After(now.Add(time.Minute)) {
				t.Fatalf("executeAt %s, want ~now", work[0].ExecuteAt)
			}
		})
	}
}

// FR-A3: creating the occurrence schedules its end.
func TestStartSchedulesTheEndTransition(t *testing.T) {
	end := now.Add(48 * time.Hour)
	db := newTestDB(t)
	o := seedActiveOccurrence(t, db, TypeName, window(now, end))

	p, err := NewHandler(db).Start(testCtx(t), registryOccurrence(o))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if p.NextTransitionAt == nil || !p.NextTransitionAt.Equal(end) {
		t.Fatalf("NextTransitionAt = %v, want %s", p.NextTransitionAt, end)
	}
	if p.Terminal {
		t.Fatalf("Start must not be terminal")
	}
}

// FR-A4: an event whose whole window elapsed during an outage completes the
// work with NO occurrence rather than starting a retroactive one.
func TestEvaluateRefusesAFullyElapsedWindow(t *testing.T) {
	seed, err := NewHandler(newTestDB(t)).Evaluate(testCtx(t),
		definitionWith(window(now.Add(-48*time.Hour), now.Add(-time.Hour))), registry.Work{})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if seed != nil {
		t.Fatalf("started a retroactive occurrence: %+v", seed)
	}
}

// FR-A14/FR-A15: the end transition completes with SCHEDULED_END and issues
// exactly ONE cancel command, not one per character.
func TestAdvanceCompletesAndCancelsOnce(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedActiveOccurrence(t, db, TypeName, window(now.Add(-48*time.Hour), now))

	p, err := NewHandlerWith(db, f).Advance(testCtx(t), registryOccurrence(o), registry.Work{Type: scheduling.WorkTypeOccurrenceTransition})
	if err != nil {
		t.Fatalf("Advance: %v", err)
	}
	if !p.Terminal || p.CompletionReason != ReasonScheduledEnd {
		t.Fatalf("progress = %+v", p)
	}
	cancels := f.emitted(buff.EnvCommandTopic, buff.CommandTypeCancelByCorrelation)
	if len(cancels) != 1 {
		t.Fatalf("emitted %d cancels, want exactly 1 (FR-A15)", len(cancels))
	}
	if cancels[0].Body.CorrelationId != o.Id().String() {
		t.Fatalf("cancel correlation = %q", cancels[0].Body.CorrelationId)
	}
}

// design §15.5: Anniversary's concurrency key is a CONSTANT, so at most one
// occurrence can be active — which is also what tells the UI it may render
// live occurrence state on the definition row (FR-UI4).
func TestConcurrencyKeyIsConstant(t *testing.T) {
	h := NewHandler(newTestDB(t))
	a, _ := h.ConcurrencyKey(testCtx(t), json.RawMessage(`{}`))
	b, _ := h.ConcurrencyKey(testCtx(t), json.RawMessage(`{"anything":1}`))
	if a == "" || a != b {
		t.Fatalf("key must be a non-empty constant, got %q / %q", a, b)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./events/anniversary/ -v`
Expected: FAIL — package does not compile.

- [ ] **Step 3: Implement `config.go` and the handler**

`Validate` requires `scheduledEnd` after `scheduledStart` and both multipliers
`> 0`. `Evaluate` re-checks `scheduledEnd.After(time.Now())` (FR-A4) and returns
a `registry.Seed` with no stage, no map scope, a constant `ConcurrencyKey`, and
a context carrying the two multipliers and the buff source id.

`Start` returns `registry.Progress{NextTransitionAt: &scheduledEnd}` — the
generic layer turns that into the `OCCURRENCE_TRANSITION` row (FR-A3).

`Advance` completes with `ReasonScheduledEnd` and emits one
`CANCEL_BY_CORRELATION` for the occurrence id. `Complete` is the same emit, so a
completion driven from either direction behaves identically and is idempotent
(re-issuing the cancel after the sweep is a no-op — Task 32).

**How enabling schedules the start, without a type switch.** FR-A2 needs
enabling an Anniversary definition to schedule work, but the generic layer must
not know that Anniversary is the type that wants it. Resolve it generically:
`definition.Processor.SetEnabled`, on a false→true transition, schedules **one**
`TRIGGER_EVALUATION` at `time.Now()` for whatever definition was enabled, with
an empty work context and a dedupe key of `enable:<definitionId>`. The handler's
`Evaluate` then decides what that means — which is exactly the division FR-X3
asks for.

Two consequences to implement here:

1. Anniversary's `Evaluate` treats an empty work context as "the definition was
   just enabled" and applies the FR-A6 rule (start now if `scheduledStart` has
   passed and `scheduledEnd` has not; otherwise the row it schedules for
   `scheduledStart` does the work).
2. Crimson Balrog's `Evaluate` will now be invoked once with an empty work
   context. Add a guard returning `(nil, nil)` when `w.Context` does not decode
   into a `WorkContext` carrying a non-nil `VoyageId`, and a test for it in this
   task:

```go
// Enabling a definition schedules one generic TRIGGER_EVALUATION with an empty
// work context. For an externally-triggered event that means "nothing to do" —
// not an error, and certainly not an occurrence with a zero voyage id.
func TestCrimsonBalrogEvaluateIgnoresAnEmptyWorkContext(t *testing.T) {
	seed, err := crimsonbalrog.NewHandler().Evaluate(testCtx(t), definitionWith(validConfig), registry.Work{Context: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatalf("an enable-triggered evaluation must not error: %v", err)
	}
	if seed != nil {
		t.Fatalf("created an occurrence with no voyage: %+v", seed)
	}
}
```

Put that test in `events/crimsonbalrog/evaluate_test.go` alongside Task 24's.

- [ ] **Step 4: Fill in the seed configuration**

`event-anniversary.json`, disabled, with `expMultiplier: 2.0`,
`dropMultiplier: 2.0`, and a `scheduledStart`/`scheduledEnd` pair in the past so
a fresh environment does not spontaneously start an event when someone enables
it to look around. Pick the buff `sourceId` from a range no skill uses —
`grep -rn "SourceId" services/atlas-buffs/atlas.com/buffs/` to see what the
existing producers use, and document the choice in the seed file's comment
field if the schema has one, otherwise in `context.md`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-events/atlas.com/events/ services/atlas-configurations/seed-data/
git commit -m "feat(task-231): add the Anniversary event — schedule, window and completion"
```

---

## Task 34: Grant the Anniversary buff at login

**Module root:** `services/atlas-events/atlas.com/events`

Reaction to `EVENT_TOPIC_CHARACTER_STATUS` / `LOGIN`, the same event and shape
`atlas-buffs` already uses for berserk tracking
(`services/atlas-buffs/atlas.com/buffs/kafka/consumer/characterstatus/consumer.go:52`).
Critically this means **login never touches `atlas-events`** — there is no
synchronous query in the login path, so `atlas-events` being down delays the
buff rather than the login (design §15.2, FR-A8). Buff propagation across
channel changes is already `atlas-buffs`' responsibility.

### Files

- `services/atlas-events/atlas.com/events/kafka/message/characterstatus/kafka.go` — new; mirror only `LOGIN`
- `services/atlas-events/atlas.com/events/kafka/consumer/characterstatus/consumer.go` — new
- `services/atlas-events/atlas.com/events/events/anniversary/login.go` — new
- `services/atlas-events/atlas.com/events/events/anniversary/login_test.go` — new
- `services/atlas-events/atlas.com/events/main.go` — register the consumer and `registry.Register(anniversary.NewHandler(db))`
- `services/atlas-buffs/atlas.com/buffs/kafka/message/characterstatus/kafka.go` — **read-only** reference for the mirrored contract

**Interfaces:**
- Consumes: `EVENT_TOPIC_CHARACTER_STATUS` / `LOGIN`;
  `occurrence.Processor.GetActiveByType` (Task 15), index `ix_occ_type_state` (Task 19).
- Produces: `anniversary.LoginProcessor.OnLogin(e characterstatus.StatusEvent[characterstatus.StatusEventLoginBody]) error`.

- [ ] **Step 1: Write the failing test**

```go
// FR-A7: logging in while the occurrence is active grants the configured buff,
// with the occurrence id as its correlation (FR-A12).
func TestLoginDuringAnActiveOccurrenceGrantsTheBuff(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedActiveOccurrence(t, db, TypeName, multipliers(2.0, 2.0))

	must(t, NewLoginProcessorWith(testLogger(t), testCtx(t), db, f).OnLogin(login(world(1), channel(4), character(42))))

	applies := f.emitted(buff.EnvCommandTopic, buff.CommandTypeApply)
	if len(applies) != 1 {
		t.Fatalf("emitted %d applies, want 1", len(applies))
	}
	a := applies[0]
	if a.CharacterId != 42 || a.WorldId != 1 || a.ChannelId != 4 {
		t.Fatalf("targeted %d in %d/%d", a.CharacterId, a.WorldId, a.ChannelId)
	}
	if a.Body.CorrelationId != o.Id().String() {
		t.Fatalf("correlation = %q", a.Body.CorrelationId)
	}
	// 2.0x carried as amount 200, per ConversionDirect (Task 8).
	want := map[string]int32{"EXP_BUFF_RATE": 200, "ITEM_UP_BY_ITEM": 200}
	got := map[string]int32{}
	for _, c := range a.Body.Changes {
		got[c.Type] = c.Amount
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("changes = %v, want %v", got, want)
	}
	// FR-A5: the occurrence is the authoritative fact. The buff must not carry
	// a duration derived from the window — it is cancelled at completion.
	if !a.Body.NoExpiry || a.Body.Duration != 0 {
		t.Fatalf("noExpiry=%v duration=%d, want true/0", a.Body.NoExpiry, a.Body.Duration)
	}
}

// FR-A16: after completion, a newly logging-in character gets nothing.
func TestLoginAfterCompletionGrantsNothing(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	seedCompletedOccurrence(t, db, TypeName, ReasonScheduledEnd)

	must(t, NewLoginProcessorWith(testLogger(t), testCtx(t), db, f).OnLogin(login(world(1), channel(4), character(42))))

	if got := len(f.emitted(buff.EnvCommandTopic, buff.CommandTypeApply)); got != 0 {
		t.Fatalf("emitted %d applies after completion, want 0", got)
	}
}

// FR-A1: the multipliers are configuration. A 1.5x configuration produces 150.
func TestConfiguredMultipliersAreCarriedVerbatim(t *testing.T) { ... }

// A login with no Anniversary occurrence at all is a cheap no-op.
func TestLoginWithNoOccurrenceIsANoOp(t *testing.T) { ... }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-events/atlas.com/events && go test ./events/anniversary/ -run Login -v`
Expected: FAIL — `NewLoginProcessor` undefined.

- [ ] **Step 3: Implement**

```go
// OnLogin grants the active occurrence's buff to a character entering gameplay
// (FR-A7). This is a REACTION, not a query in the login path: atlas-events
// being unavailable delays the buff, it never delays or fails the login
// (FR-A8).
//
// The buff is applied with noExpiry: the occurrence — not a duration — is the
// authoritative fact that Anniversary is happening (FR-A5), and completion
// cancels it explicitly by correlation (FR-A15).
func (p *LoginProcessorImpl) OnLogin(e characterstatus.StatusEvent[characterstatus.StatusEventLoginBody]) error {
	os, err := occurrence.NewProcessor(p.l, p.ctx, p.db).GetActiveByType(TypeName)
	if err != nil {
		return err
	}
	for _, o := range os {
		c, err := DecodeOccurrenceContext(o.Context())
		if err != nil {
			return err
		}
		if err := p.emitApply(e.WorldId, e.Body.ChannelId, e.CharacterId, o.Id(), c); err != nil {
			return err
		}
	}
	return nil
}
```

`emitApply` builds the `APPLY` command with
`Changes: [{EXP_BUFF_RATE, int32(expMultiplier*100)}, {ITEM_UP_BY_ITEM, int32(dropMultiplier*100)}]`,
`NoExpiry: true`, `Duration: 0` (the consumer rejects `noExpiry` with a nonzero
duration — `kafka/consumer/character/consumer.go:58`), and
`CorrelationId: o.Id().String()`.

- [ ] **Step 4: Register the consumer and the handler**

In `main.go`: `characterstatusConsumer.InitConsumers` / `InitHandlers`, and
`registry.Register(anniversary.NewHandler(db))`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-events/atlas.com/events && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-events/atlas.com/events/
git commit -m "feat(task-231): grant the Anniversary buff on character login"
```

---

# Phase G — `atlas-ui` events administration (design §13)

**Module root for Tasks 35–38:** `services/atlas-ui`
`atlas-ui` requires nvm; run `nvm use` before `npm` if the shell has no node.
The build type-checks tests, so a broken test file fails the build.

## Task 35: The events API client

### Files

- `services/atlas-ui/src/services/api/events.service.ts` — new
- `services/atlas-ui/src/services/api/index.ts` — export it
- `services/atlas-ui/src/types/models/events.ts` — new (follow the existing `src/types/models/` layout; confirm with `ls services/atlas-ui/src/types/models/`)
- `services/atlas-ui/src/services/api/reward-pools.service.ts` — **read-only** reference
- `services/atlas-ui/src/services/api/__tests__/` — add a test file mirroring the neighbours

**Interfaces:**
- Consumes: the REST surface of Tasks 13 and 16.
- Produces:
  - `eventsService.getDefinitions(params)`, `getDefinition(id)`, `setDefinitionEnabled(id, enabled)`
  - `eventsService.getOccurrences(filters, page)`, `getOccurrence(id)` (with transitions)
  - types `EventDefinition`, `EventOccurrence`, `EventOccurrenceTransition`,
    `EventOccurrenceFilters`

- [ ] **Step 1: Write the failing test**

```ts
describe("eventsService", () => {
  it("sends JSON:API filter params for the occurrence list", async () => {
    const fetchMock = mockFetch({ data: [] });
    await eventsService.getOccurrences(
      { type: "CRIMSON_BALROG", state: "ACTIVE", worldId: 1, channelId: 4 },
      { number: 1, size: 25 },
    );
    const url = new URL(fetchMock.mock.calls[0][0] as string, "http://x");
    expect(url.searchParams.get("filter[type]")).toBe("CRIMSON_BALROG");
    expect(url.searchParams.get("filter[state]")).toBe("ACTIVE");
    expect(url.searchParams.get("filter[worldId]")).toBe("1");
    expect(url.searchParams.get("page[number]")).toBe("1");
  });

  it("PATCHes only enabled when toggling a definition", async () => {
    const fetchMock = mockFetch({ data: {} });
    await eventsService.setDefinitionEnabled("d1", true);
    const body = JSON.parse(fetchMock.mock.calls[0][1].body as string);
    expect(Object.keys(body.data.attributes)).toEqual(["enabled"]);
    expect(body.data.attributes.enabled).toBe(true);
  });

  it("surfaces the included transitions on the occurrence detail", async () => {
    mockFetch({
      data: { id: "o1", type: "event-occurrences", attributes: { state: "COMPLETED" } },
      included: [{ id: "t1", type: "event-occurrence-transitions", attributes: { toStage: "ATTACKING" } }],
    });
    const o = await eventsService.getOccurrence("o1");
    expect(o.transitions).toHaveLength(1);
    expect(o.transitions[0].toStage).toBe("ATTACKING");
  });
});
```

Read `services/atlas-ui/src/services/api/__tests__/` first and reuse its
`mockFetch`/tenant-header helpers rather than writing new ones.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm test -- events.service`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement**

Copy `reward-pools.service.ts`'s shape: the shared JSON:API client, the tenant
header from context, typed responses, and pagination per `pagination.ts`. Export
from `index.ts`.

- [ ] **Step 4: Run tests and lint**

Run: `cd services/atlas-ui && npm test -- events.service && npm run lint`
Expected: PASS; lint clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/
git commit -m "feat(task-231): add the atlas-ui events API client"
```

---

## Task 36: Definitions page

### Files

- `services/atlas-ui/src/pages/EventDefinitionsPage.tsx` — new
- `services/atlas-ui/src/pages/event-definitions-columns.tsx` — new
- `services/atlas-ui/src/pages/__tests__/EventDefinitionsPage.test.tsx` — new
- `services/atlas-ui/src/App.tsx` — lazy import + `<Route path="/events/definitions" …>`
- `services/atlas-ui/src/components/app-sidebar-items.ts` — nav entry
- `services/atlas-ui/src/lib/breadcrumbs/routes.ts` — breadcrumb label
- `services/atlas-ui/src/pages/RewardPoolsPage.tsx` + `reward-pools-columns.tsx` — **read-only** reference

**Interfaces:**
- Consumes: `eventsService.getDefinitions`, `setDefinitionEnabled` (Task 35).
- Produces: route `/events/definitions`.

- [ ] **Step 1: Write the failing test**

```tsx
describe("EventDefinitionsPage", () => {
  it("shows name, type and enablement", async () => {
    renderWithProviders(<EventDefinitionsPage />, { definitions: [balrogDefinition, anniversaryDefinition] });
    expect(await screen.findByText("Crimson Balrog")).toBeInTheDocument();
    expect(screen.getByText("CRIMSON_BALROG")).toBeInTheDocument();
  });

  it("toggles enablement", async () => {
    const setEnabled = vi.spyOn(eventsService, "setDefinitionEnabled").mockResolvedValue(undefined);
    renderWithProviders(<EventDefinitionsPage />, { definitions: [balrogDefinition] });
    await userEvent.click(await screen.findByRole("switch", { name: /crimson balrog/i }));
    expect(setEnabled).toHaveBeenCalledWith(balrogDefinition.id, true);
  });

  // FR-UI4: "enabled" must never read as "occurring". A definition that can
  // have MANY concurrent occurrences (singleOccurrence: false) shows a count
  // linking to the filtered list, never a single live state.
  it("does not render enabled as occurring for a multi-occurrence type", async () => {
    renderWithProviders(<EventDefinitionsPage />, {
      definitions: [{ ...balrogDefinition, enabled: true, singleOccurrence: false }],
      activeCounts: { [balrogDefinition.id]: 0 },
    });
    expect(await screen.findByText(/0 active/i)).toBeInTheDocument();
    expect(screen.queryByText(/^occurring$/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/in progress/i)).not.toBeInTheDocument();
  });

  // A single-occurrence type MAY show live state, because at most one exists.
  it("shows live occurrence state for a single-occurrence type", async () => {
    renderWithProviders(<EventDefinitionsPage />, {
      definitions: [{ ...anniversaryDefinition, enabled: true, singleOccurrence: true }],
      activeCounts: { [anniversaryDefinition.id]: 1 },
    });
    expect(await screen.findByText(/active now/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm test -- EventDefinitionsPage`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement**

The occurrence column reads `singleOccurrence` off the definition resource — it
must **not** switch on `type`. FR-X3 applies to the frontend too: a third event
must render correctly with no edit here.

Register the route, the sidebar entry ("Events" group → "Definitions") and the
breadcrumb.

- [ ] **Step 4: Run tests, lint and build**

Run: `cd services/atlas-ui && npm test -- EventDefinitionsPage && npm run lint && npm run build`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/
git commit -m "feat(task-231): add the event definitions admin page"
```

---

## Task 37: Occurrences list page

### Files

- `services/atlas-ui/src/pages/EventOccurrencesPage.tsx` — new
- `services/atlas-ui/src/pages/event-occurrences-columns.tsx` — new
- `services/atlas-ui/src/pages/__tests__/EventOccurrencesPage.test.tsx` — new
- `services/atlas-ui/src/App.tsx`, `components/app-sidebar-items.ts`, `lib/breadcrumbs/routes.ts` — register `/events/occurrences`

**Interfaces:**
- Consumes: `eventsService.getOccurrences` (Task 35).

- [ ] **Step 1: Write the failing test**

```tsx
describe("EventOccurrencesPage", () => {
  // FR-UI5: id, name/type, state, stage, scope, start, completion, reason.
  it("renders the occurrence summary columns", async () => {
    renderWithProviders(<EventOccurrencesPage />, { occurrences: [completedBalrogOccurrence] });
    expect(await screen.findByText("VESSEL_ARRIVED")).toBeInTheDocument();
    expect(screen.getByText("COMPLETED")).toBeInTheDocument();
    expect(screen.getByText(/w1 ch4/i)).toBeInTheDocument();
  });

  // FR-UI5: active occurrences must be readily distinguishable from history.
  it("distinguishes active from historical occurrences", async () => {
    renderWithProviders(<EventOccurrencesPage />, {
      occurrences: [activeBalrogOccurrence, completedBalrogOccurrence],
    });
    const active = await screen.findByTestId(`occurrence-${activeBalrogOccurrence.id}`);
    const historical = screen.getByTestId(`occurrence-${completedBalrogOccurrence.id}`);
    expect(within(active).getByText("ACTIVE")).toBeInTheDocument();
    expect(within(historical).getByText("COMPLETED")).toBeInTheDocument();
    expect(active.className).not.toEqual(historical.className);
  });

  // FR-UI6: filterable by type, state, active-vs-historical, date range, world
  // and channel.
  it("passes the selected filters to the service", async () => {
    const get = vi.spyOn(eventsService, "getOccurrences");
    renderWithProviders(<EventOccurrencesPage />, { occurrences: [] });
    await userEvent.selectOptions(await screen.findByLabelText(/state/i), "ACTIVE");
    await waitFor(() => expect(get).toHaveBeenLastCalledWith(
      expect.objectContaining({ state: "ACTIVE" }), expect.anything(),
    ));
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm test -- EventOccurrencesPage`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement**

Follow `RewardPoolsPage.tsx`'s data-table + TanStack Query shape. Filters use
`react-hook-form` + Zod where they are a form; a simple select row needs neither.

- [ ] **Step 4: Run tests, lint and build**

Run: `cd services/atlas-ui && npm test -- EventOccurrencesPage && npm run lint && npm run build`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/
git commit -m "feat(task-231): add the event occurrences list page"
```

---

## Task 38: Occurrence detail page

### Files

- `services/atlas-ui/src/pages/EventOccurrenceDetailPage.tsx` — new
- `services/atlas-ui/src/pages/event-occurrence-panels.tsx` — new; the per-type detail panels
- `services/atlas-ui/src/pages/__tests__/EventOccurrenceDetailPage.test.tsx` — new
- `services/atlas-ui/src/App.tsx`, `lib/breadcrumbs/routes.ts` — register `/events/occurrences/:id`

**Interfaces:**
- Consumes: `eventsService.getOccurrence` with included transitions (Task 35).

- [ ] **Step 1: Write the failing test**

```tsx
describe("EventOccurrenceDetailPage", () => {
  // FR-UI7
  it("shows the transition history", async () => {
    renderWithProviders(<EventOccurrenceDetailPage />, { occurrence: completedBalrogOccurrence });
    expect(await screen.findByText("OCCURRENCE_CREATED")).toBeInTheDocument();
    expect(screen.getByText("ATTACKING")).toBeInTheDocument();
    expect(screen.getByText("VESSEL_ARRIVED")).toBeInTheDocument();
  });

  // FR-UI8
  it("shows Crimson Balrog scope and monster status", async () => {
    renderWithProviders(<EventOccurrenceDetailPage />, { occurrence: completedBalrogOccurrence });
    expect(await screen.findByText(/voyage/i)).toBeInTheDocument();
    expect(screen.getByText("200090010")).toBeInTheDocument();
  });

  it("shows Anniversary schedule and multipliers", async () => {
    renderWithProviders(<EventOccurrenceDetailPage />, { occurrence: anniversaryOccurrence });
    expect(await screen.findByText("2.0x")).toBeInTheDocument();
    expect(screen.getByText(/scheduled end/i)).toBeInTheDocument();
  });

  // An occurrence of a type with no bespoke panel still renders: the generic
  // context view is the fallback, so a third event needs no edit here to be
  // usable.
  it("falls back to the generic context view for an unknown type", async () => {
    renderWithProviders(<EventOccurrenceDetailPage />, {
      occurrence: { ...completedBalrogOccurrence, type: "MYSTERIOUS_MERCHANT" },
    });
    expect(await screen.findByTestId("occurrence-context-json")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm test -- EventOccurrenceDetailPage`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement**

The per-type panel is a **component lookup with a generic fallback**, not a
switch that must be edited for the page to keep working (design §13, §16):

```tsx
const detailPanels: Record<string, React.ComponentType<{ occurrence: EventOccurrence }>> = {
  CRIMSON_BALROG: CrimsonBalrogPanel,
  ANNIVERSARY: AnniversaryPanel,
};

// Falls back to the raw context view. This is why adding a third event needs no
// change to the page structure — only, optionally, a new panel component.
const Panel = detailPanels[occurrence.type] ?? GenericContextPanel;
```

- [ ] **Step 4: Run tests, lint and build**

Run: `cd services/atlas-ui && npm test -- EventOccurrenceDetailPage && npm run lint && npm run build`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/
git commit -m "feat(task-231): add the event occurrence detail page"
```

---

# Phase H — Boundary enforcement and the gate

## Task 39: Pin the generic/specific boundary with an AST test

**Module root:** `services/atlas-events/atlas.com/events`

FR-X3 and acceptance 20.9 are the only requirements in this task that no
ordinary test can hold: they are properties of the code's *shape*, and they
survive contact with the next feature only if something checks them
mechanically.

### Files

- `services/atlas-events/atlas.com/events/event/boundary_test.go` — new
- `docs/tasks/task-231-generalized-events-service/third-event-walkthrough.md` — new; copy design §16 verbatim, as the written artifact acceptance 20.9 asks for

**Interfaces:**
- Consumes: nothing at runtime. Walks the source of `event/…` with `go/parser`.

- [ ] **Step 1: Write the test**

```go
package event

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// knownEventTypes are the type discriminators an event package owns. The
// GENERIC layer must never name one: FR-X3 permits "a registry mapping type to
// a handler" and forbids "a switch containing event logic". A literal here in
// event/definition, event/occurrence, event/transition or event/scheduling is
// that forbidden switch beginning to form.
var knownEventTypes = []string{"CRIMSON_BALROG", "ANNIVERSARY"}

// genericPackages is every directory under event/. Adding a package here is
// intentional: a new generic package must obey the same rule.
func TestGenericLayerNeverNamesAnEventType(t *testing.T) {
	root := "."
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}
		ast.Inspect(f, func(n ast.Node) bool {
			bl, ok := n.(*ast.BasicLit)
			if !ok || bl.Kind != token.STRING {
				return true
			}
			for _, known := range knownEventTypes {
				if strings.Contains(bl.Value, known) {
					t.Errorf("%s: the generic layer names event type %s (FR-X3). "+
						"Reach event behavior through registry.Handler instead.",
						fset.Position(bl.Pos()), known)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
```

Place the file in `event/` and walk `"."` from there, so it covers every
generic package and no event package. If the test lives in a package that
cannot import `event/...`, put it in its own `boundary_test` package.

- [ ] **Step 2: Run the test**

Run: `cd services/atlas-events/atlas.com/events && go test ./event/ -run Boundary -v`
Expected: PASS. If it fails, the named file is the violation — move the
behavior behind `registry.Handler` rather than adding an exception to
`knownEventTypes`.

- [ ] **Step 3: Write the walkthrough artifact**

Copy design §16 (the Mysterious Merchant walkthrough) into
`docs/tasks/task-231-generalized-events-service/third-event-walkthrough.md`,
and check each of its claims against the code as it now stands rather than as
designed. Specifically confirm, file by file, that adding that event would touch
only: a new `events/mysteriousmerchant/` package, one `registry.Register` line in
`main.go`, one seed JSON, and one new Kafka command package — and **not**
`event/definition`, `event/occurrence`, `event/transition`, `event/scheduling`,
the poller, the REST resources, the UI page structure, or the schema. Where the
implementation diverged from the design, say so plainly in the file; that is the
finding, not a formality.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-events/atlas.com/events/event/boundary_test.go \
        docs/tasks/task-231-generalized-events-service/third-event-walkthrough.md
git commit -m "test(task-231): enforce the generic/specific boundary and record the third-event walkthrough"
```

---

## Task 40: The gate

**Module root:** repo root

Acceptance 20.10. Neither half is optional, and CLAUDE.md is explicit that a
green `tools/verify.sh` does **not** mean the branch is correct — every service
can build, vet, test and bake clean while a cross-service seam is broken.

### Files

- No source files. Produces `docs/tasks/task-231-generalized-events-service/audit.md` (written by the reviewer agents).

- [ ] **Step 1: Run the flagless gate**

Run, in the background, from the worktree root:

```bash
tools/verify.sh
```

**Flagless only.** `--quick` and `--no-docker` also exit 0 — they print a caveat
and skip the bake and `-race` — so "verify.sh exited 0" is not a pass unless it
ran with no flags. The bake in particular is what catches a missing
`COPY libs/...` in the shared Dockerfile, which `go build` against `go.work`
cannot see.

This branch touches nine modules, so expect a long run. Launch it once with
`run_in_background: true` and do Step 2 while it runs; do not poll it in a loop.

- [ ] **Step 2: Trace the cross-service seams by hand**

`verify.sh` cannot see these. For each, confirm a test asserts the **new**
contract rather than the old silent drop:

| Seam | The test that must exist |
|---|---|
| `atlas-transports` emits `VOYAGE_*` → `atlas-channel` route consumer ignores them | Task 30 |
| `atlas-transports` emits `VOYAGE_*` → `atlas-events` consumes them | Tasks 23, 27 |
| `atlas-monsters` echoes provenance → `atlas-events` tracks the monster set | Tasks 3, 26 |
| `atlas-events` emits `SPAWN_FIELD` with provenance → `atlas-monsters` stores it | Tasks 2, 25 |
| `atlas-events` emits `DESTROY_BY_SOURCE` → `atlas-monsters` sweeps the field | Tasks 4, 27 |
| `atlas-events` emits the visual → `atlas-channel` renders `CONTI_MOVE` | Tasks 25, 28 |
| `atlas-events` applies a correlated buff → `atlas-buffs` stores the correlation | Tasks 31, 34 |
| `atlas-events` cancels by correlation → `atlas-buffs` sweeps | Tasks 32, 33 |
| `atlas-buffs` emits `APPLIED` → `atlas-rates` composes `exp`/`item_drop` | Task 8 |

Any row without a test is a gap to close before the review, not a note for
later.

- [ ] **Step 3: Run the code review**

Invoke `superpowers:requesting-code-review`, which dispatches the three modular
reviewers in parallel. Go and TypeScript both changed, so all three apply:

- `plan-adherence-reviewer` — every task in this plan actually implemented
- `backend-guidelines-reviewer` — the DOM-* checklist over the Go changes
- `frontend-guidelines-reviewer` — the FE-* checklist over `atlas-ui`

Pin `model: sonnet` on every dispatch (review/verify/audit jobs always run
sonnet). Each writes to
`docs/tasks/task-231-generalized-events-service/audit.md`. Ensure they operate
inside this worktree and that `git status` is clean after they run.

- [ ] **Step 4: Fix what the review found, then re-run**

Address every finding on this branch — do not defer to a follow-up task when the
blocker is something producible here. Re-run `tools/verify.sh` flagless after
the fixes and confirm exit 0.

- [ ] **Step 5: Confirm no stubs landed**

```bash
grep -rn "not implemented\|// TODO\|// FIXME\|StatusNotImplemented" \
  services/atlas-events services/atlas-monsters services/atlas-transports \
  services/atlas-buffs services/atlas-rates services/atlas-channel \
  --include='*.go' | grep -v _test.go
```

Expected: no output. No `// TODO`, stubbed handler or 501 may land.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-231-generalized-events-service/audit.md
git commit -m "chore(task-231): record the code-review audit"
```

---

## Deferred, and why it is safe to defer

Carried from design §19. Each is recorded so a later task knows exactly what it
would be picking up — none is a hidden blocker inside this one.

- **Administrative termination of a live occurrence** (FR-D5). Disabling a
  definition deliberately leaves an `ACTIVE` occurrence running.
- **Scheduled-work retention** (design §15.7). The poller's index is partial on
  `state = 'PENDING'`, so completed and failed rows are not in it and the poll
  cost does not grow with them — which is exactly what FR-N16 asks for. The only
  cost of accumulation is disk.
- **BGM restoration on the monsters-eliminated path** (design §15.4). Blocked on
  `atlas-data` exposing Map.wz `info/bgm`, which it does not — verified:
  `grep -n "bgm\|Bgm" services/atlas-data/atlas.com/data/map/*.go` returns
  nothing outside tests. "Restore the default" would mean hard-coding a guessed
  string. The visual **is** removed on this path, which is the part players see.
- **Global `DESTROY_BY_SOURCE`** (design §15.6). Field-scoped suffices; a global
  variant would need a new Redis secondary index maintained on every spawn and
  death.
- **A third event.** §16 is a walkthrough (Task 39's artifact), not an
  implementation.
- **`ContiMove` routing in the five partial templates** (`gms_12`, `gms_48`,
  `gms_61`, `gms_72`, `gms_92`). Those templates are incomplete bring-ups
  generally, and routing an opcode there requires deriving it per version from
  the IDB — `/bringup-version` work with its own playbook. Recorded with
  evidence in Task 30's `template-routing.md`.
