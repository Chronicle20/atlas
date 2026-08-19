# Whisper `/find` — Accurate Target Location — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `/find` report where a character actually is — their real channel, the cash shop, or "not findable" when offline — instead of guessing channel 1.

**Architecture:** A three-valued `PresenceState` discriminator (`OFFLINE` / `IN_FIELD` / `IN_CASH_SHOP`) is added to `atlas-maps`'s existing `character_locations` row and driven by the LOGIN / LOGOUT / CHANNEL_CHANGED / cash-shop-ENTER / cash-shop-EXIT events `atlas-maps` already consumes. It is projected onto the existing `GET /characters/{id}/location` endpoint. `atlas-channel` rewrites `produceFindResultBody` into a pure, table-driven `findDecision` over three injected lookup seams, so every selection rule is unit-testable.

**Tech Stack:** Go 1.x, GORM (sqlite in tests), Kafka consumers via `libs/atlas-kafka`, JSON:API via api2go, `libs/atlas-packet` codecs, `libs/atlas-constants`.

**Spec:** [design.md](design.md) (PRD at [prd.md](prd.md))

## Global Constraints

- **`atlas-cashshop`, `atlas-mts`, and `atlas-character` are not touched.** The design supersedes PRD §4.2 / §5.1 / §6.1 / §7 — approved by the user during `/plan-task`. Do not create a cash-shop presence store.
- **No wire change to any packet version.** `libs/atlas-packet` changes in this plan are tests and evidence only. Design §2.6's "flagged divergence" on `gms_v92`/`gms_v95` was **withdrawn** — re-derived against the live IDBs, both versions gate x/y on the odd-mode (`0x09`) test exactly as v83/v84/v87 do, and Atlas's current encoding is already correct.
- **`PresenceState` string values are exactly** `"OFFLINE"`, `"IN_FIELD"`, `"IN_CASH_SHOP"`. They cross a REST boundary; do not rename or re-case them.
- **`OFFLINE` is the zero value.** Existing rows, an absent `state` in a REST payload, and any unrecognised value all resolve to `OFFLINE`. Failing toward "not findable" is deliberate (design §1.4).
- **`OFFLINE` is terminal except via LOGIN and CHANNEL_CHANGED.** The cash-shop status topic and the character status topic have no mutual ordering guarantee, so cash-shop transitions apply only when the current state is not `OFFLINE` (design §1.3).
- **`location.ResolveMapId` is not used by the find path** and must not be modified. Its collapse-to-map-0 behaviour is what lets a transport failure render as a real location; it stays for its other callers.
- Never land a placeholder comment, a stubbed handler, or an unimplemented status response. Both existing placeholder comments in `character_chat_whisper.go` are removed by Task 8, not carried forward.
- Per-task verification is module-local only: `go build ./... && go test ./...` from the module root named in each task. Repo-wide `tools/verify.sh` is the controller's gate, not the implementer's.

---

## Task 1: Shared `PresenceState` enum

`libs/atlas-constants/character/` was checked for an existing equivalent — it holds only `constants.go`, `temporary_stat.go` and `energy_charge.go`. There is no presence/liveness type anywhere in `libs/atlas-constants/`, so this is new.

The value lives in `atlas-constants` rather than in either service because it crosses the `atlas-maps` → `atlas-channel` REST boundary as a wire string; duplicating the literals in two services is the drift this library exists to prevent.

### Files

- `libs/atlas-constants/character/presence.go` — **new file**; the enum and its parse helper
- `libs/atlas-constants/character/presence_test.go` — **new file**; tests
- `libs/atlas-constants/character/energy_charge.go` — read-only; the file-shape convention to copy (package clause, const block style)

Module root for `go build`/`go test`: `libs/atlas-constants`

**Interfaces:**
- Consumes: nothing.
- Produces: `character.PresenceState` (a `string` type); constants `PresenceStateOffline`, `PresenceStateInField`, `PresenceStateInCashShop`; `func ParsePresenceState(s string) PresenceState`. Tasks 2, 3, 4, 5, 6 and 8 all depend on these exact names.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-constants/character/presence_test.go`:

```go
package character

import "testing"

func TestPresenceStateValues(t *testing.T) {
	// These literals cross the atlas-maps -> atlas-channel REST boundary.
	// Renaming or re-casing one silently degrades /find to "not findable".
	cases := []struct {
		state PresenceState
		want  string
	}{
		{PresenceStateOffline, "OFFLINE"},
		{PresenceStateInField, "IN_FIELD"},
		{PresenceStateInCashShop, "IN_CASH_SHOP"},
	}
	for _, c := range cases {
		if string(c.state) != c.want {
			t.Errorf("state = %q, want %q", string(c.state), c.want)
		}
	}
}

func TestParsePresenceState(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  PresenceState
	}{
		{"offline", "OFFLINE", PresenceStateOffline},
		{"in field", "IN_FIELD", PresenceStateInField},
		{"in cash shop", "IN_CASH_SHOP", PresenceStateInCashShop},
		// The zero value is OFFLINE by design: a row written before this
		// column existed, or a REST payload from an atlas-maps that has not
		// been redeployed, must fail toward "not findable" rather than
		// asserting liveness.
		{"empty string is offline", "", PresenceStateOffline},
		{"unrecognised is offline", "IN_ORBIT", PresenceStateOffline},
		{"lowercase is not accepted", "in_field", PresenceStateOffline},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ParsePresenceState(c.input); got != c.want {
				t.Errorf("ParsePresenceState(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./character/ -run TestPresenceState -v`
Expected: FAIL — build error, `undefined: PresenceState`.

- [ ] **Step 3: Write minimal implementation**

Create `libs/atlas-constants/character/presence.go`:

```go
package character

// PresenceState discriminates a character's liveness on the durable location
// record held by atlas-maps. It crosses the atlas-maps -> atlas-channel REST
// boundary as a wire string, which is why it lives here rather than in either
// service.
type PresenceState string

const (
	// PresenceStateOffline means the character is not logged in. It is the
	// zero value: an unwritten column, an absent REST attribute, and an
	// unrecognised value all resolve here, so /find fails toward
	// "not findable" rather than asserting a channel it cannot support.
	PresenceStateOffline PresenceState = "OFFLINE"

	// PresenceStateInField means the character is logged in and on a map.
	PresenceStateInField PresenceState = "IN_FIELD"

	// PresenceStateInCashShop means the character is logged in and inside the
	// cash shop. This covers the MTS as well: the ITC renders inside the
	// cash-shop CStage and emits the identical CHARACTER_ENTER event, so
	// atlas-maps cannot distinguish them and does not need to.
	PresenceStateInCashShop PresenceState = "IN_CASH_SHOP"
)

// ParsePresenceState converts a wire value into a PresenceState, resolving
// anything it does not recognise — including the empty string — to
// PresenceStateOffline.
func ParsePresenceState(s string) PresenceState {
	switch PresenceState(s) {
	case PresenceStateInField:
		return PresenceStateInField
	case PresenceStateInCashShop:
		return PresenceStateInCashShop
	default:
		return PresenceStateOffline
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-constants && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/character/presence.go libs/atlas-constants/character/presence_test.go
git commit -m "feat(atlas-constants): add PresenceState for character liveness"
```

---

## Task 2: `atlas-maps` — `state` column, model, and `SetState`

Adds the discriminator to the persistence layer. Additive: one column on `character_locations`, no new table, no backfill, no change to existing columns. `AutoMigrate` adds it; existing rows adopt the Go zero value, which `Make` maps to `OFFLINE`.

### Files

- `services/atlas-maps/atlas.com/maps/character/location/entity.go` — add the `State` column and map it in `Make`
- `services/atlas-maps/atlas.com/maps/character/location/model.go` — add the `state` field, `State()` getter, `SetState` builder method, and carry it in `ToEntity`
- `services/atlas-maps/atlas.com/maps/character/location/administrator.go` — add `setLocationState`; make `upsertLocation` preserve/carry state
- `services/atlas-maps/atlas.com/maps/character/location/processor.go` — add `SetState` to the `Processor` interface and `ProcessorImpl`
- `services/atlas-maps/atlas.com/maps/character/location/processor_test.go` — new test cases
- `services/atlas-maps/atlas.com/maps/character/location/provider.go` — read-only; the curried provider shape to mirror

Module root for `go build`/`go test`: `services/atlas-maps/atlas.com/maps`

Patterns to copy: `services/atlas-maps/atlas.com/maps/character/location/administrator.go:10` (`upsertLocation`'s curried `db -> tenantId -> characterId -> …` shape).

**Interfaces:**
- Consumes: `character.PresenceState`, `character.PresenceStateOffline`, `character.PresenceStateInField`, `character.PresenceStateInCashShop` from Task 1, imported as `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"` (the local package is also named `character`, hence the alias — but note this package is `location`, so a plain import works here; use the alias only where a collision exists).
- Produces:
  - `location.Model.State() characterconst.PresenceState`
  - `(*location.Builder).SetState(v characterconst.PresenceState) *location.Builder`
  - `location.Processor` interface gains `SetState(characterId uint32, state characterconst.PresenceState) error` (unconditional) and `SetStateIfOnline(characterId uint32, state characterconst.PresenceState) error` (applies only when the row is not already `OFFLINE`)
  - Task 3 uses `State()`; Task 4 uses `SetState`; Task 5 uses `SetStateIfOnline`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-maps/atlas.com/maps/character/location/processor_test.go`. These tests are **internal to package `location`**, so processors are constructed as `NewProcessor(...)`, not `location.NewProcessor(...)`. Reuse the file's existing helpers — `newCtxTenant(t)` (line 29) and `newTestDB(t)` (line 100) — and its `logrus.New()` logger convention; do not add new helpers. The only import to add is `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`.

```go
// TestSetState_TransitionsWithoutDisturbingPosition proves the state column is
// an independent discriminator: flipping it must not move the character.
func TestSetState_TransitionsWithoutDisturbingPosition(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)

	const characterId uint32 = 42
	f := field.NewBuilder(world.Id(1), 7, _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	p := NewProcessor(logrus.New(), ctx, db)
	if _, err := p.Set(characterId, f); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// A freshly written row is OFFLINE until something asserts liveness.
	m, err := p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateOffline {
		t.Errorf("initial state = %q, want OFFLINE", m.State())
	}

	if err := p.SetState(characterId, characterconst.PresenceStateInCashShop); err != nil {
		t.Fatalf("SetState: %v", err)
	}

	m, err = p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById after SetState: %v", err)
	}
	if m.State() != characterconst.PresenceStateInCashShop {
		t.Errorf("state = %q, want IN_CASH_SHOP", m.State())
	}
	if m.WorldId() != world.Id(1) || m.ChannelId() != 7 || m.MapId() != _map.Id(100000000) {
		t.Errorf("SetState disturbed position: world=%d channel=%d map=%d", m.WorldId(), m.ChannelId(), m.MapId())
	}
}

// TestSet_PreservesState proves a position write does not reset the
// discriminator. LOGOUT calls Set then SetState; CHANGE_MAP calls Set alone and
// must leave liveness as it found it. upsertLocation used db.Save (a full-row
// overwrite), which would silently zero the state here.
func TestSet_PreservesState(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)

	const characterId uint32 = 43
	f := field.NewBuilder(world.Id(1), 2, _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	p := NewProcessor(logrus.New(), ctx, db)
	if _, err := p.Set(characterId, f); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := p.SetState(characterId, characterconst.PresenceStateInField); err != nil {
		t.Fatalf("SetState: %v", err)
	}

	moved := field.NewBuilder(world.Id(1), 2, _map.Id(104000000)).SetInstance(uuid.Nil).Build()
	if _, err := p.Set(characterId, moved); err != nil {
		t.Fatalf("Set after move: %v", err)
	}

	m, err := p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.MapId() != _map.Id(104000000) {
		t.Errorf("map = %d, want 104000000", m.MapId())
	}
	if m.State() != characterconst.PresenceStateInField {
		t.Errorf("Set reset the state to %q, want IN_FIELD preserved", m.State())
	}
}
```

This file already carries a tenant-isolation test (`processor_test.go:180`), so do not add another.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/location/ -run 'TestSetState|TestSet_PreservesState' -v`
Expected: FAIL — build error, `m.State undefined` and `p.SetState undefined`.

- [ ] **Step 3: Add the column and map it**

In `services/atlas-maps/atlas.com/maps/character/location/entity.go`, add the field to `entity` (after `Instance`, before `UpdatedAt`):

```go
	State       string     `gorm:"not null;default:'OFFLINE'"`
```

and add the import `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`, then map it in `Make`:

```go
// Make rehydrates a persistence entity into its immutable domain Model.
func Make(e entity) (Model, error) {
	return NewBuilder(e.CharacterId).
		SetWorldId(e.WorldId).
		SetChannelId(e.ChannelId).
		SetMapId(e.MapId).
		SetInstance(e.Instance).
		SetState(characterconst.ParsePresenceState(e.State)).
		Build(), nil
}
```

`ParsePresenceState` is what makes rows written before this column existed read as `OFFLINE` rather than as an empty state.

- [ ] **Step 4: Add the model field, getter, builder method, and projection**

In `services/atlas-maps/atlas.com/maps/character/location/model.go`, add the import `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`, add the field to `Model`:

```go
	state       characterconst.PresenceState
```

add the getter next to the others:

```go
func (m Model) State() characterconst.PresenceState { return m.state }
```

carry it in `ToEntity` (add to the returned `entity` literal, after `Instance`):

```go
		State:       string(m.state),
```

and add the builder method next to `SetInstance`:

```go
func (b *Builder) SetState(v characterconst.PresenceState) *Builder { b.m.state = v; return b }
```

Do **not** touch `SetField` — it sets position only, which is what lets `Set` preserve state in Step 5.

- [ ] **Step 5: Add the administrator functions**

In `services/atlas-maps/atlas.com/maps/character/location/administrator.go`, add the imports `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"` and `"gorm.io/gorm/clause"`, then change `upsertLocation` so a position write preserves the existing state, and add `setLocationState`:

```go
// upsertLocation persists the (tenantId, characterId, field) tuple into the
// character_locations table, replacing any existing row for that composite
// primary key. It mirrors the visit/ peer's curried administrator shape:
// db -> tenantId -> characterId -> field -> (entity, error).
//
// The state discriminator is deliberately NOT written here: this is a position
// write, and CHANGE_MAP must leave liveness as it found it. New rows take the
// column default (OFFLINE); existing rows keep whatever SetState last wrote.
func upsertLocation(db *gorm.DB) func(tenantId uuid.UUID) func(characterId uint32) func(f field.Model) (entity, error) {
	return func(tenantId uuid.UUID) func(characterId uint32) func(f field.Model) (entity, error) {
		return func(characterId uint32) func(f field.Model) (entity, error) {
			return func(f field.Model) (entity, error) {
				m := NewBuilder(characterId).SetField(f).Build()
				e := m.ToEntity(tenantId)
				e.State = ""
				if err := db.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"world_id", "channel_id", "map_id", "instance", "updated_at",
					}),
				}).Create(&e).Error; err != nil {
					return entity{}, err
				}
				var out entity
				if err := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).
					First(&out).Error; err != nil {
					return entity{}, err
				}
				return out, nil
			}
		}
	}
}

// setLocationState writes only the state discriminator for
// (tenantId, characterId), leaving world/channel/map/instance untouched.
//
// conditional=true applies the write only when the row is not already OFFLINE.
// The cash-shop status topic and the character status topic are separate Kafka
// topics with no mutual ordering guarantee, so a late-delivered CHARACTER_EXIT
// could otherwise resurrect a logged-off character as IN_FIELD.
func setLocationState(db *gorm.DB) func(tenantId uuid.UUID) func(characterId uint32) func(state characterconst.PresenceState, conditional bool) error {
	return func(tenantId uuid.UUID) func(characterId uint32) func(state characterconst.PresenceState, conditional bool) error {
		return func(characterId uint32) func(state characterconst.PresenceState, conditional bool) error {
			return func(state characterconst.PresenceState, conditional bool) error {
				q := db.Model(&entity{}).Where("tenant_id = ? AND character_id = ?", tenantId, characterId)
				if conditional {
					q = q.Where("state <> ?", string(characterconst.PresenceStateOffline))
				}
				return q.Update("state", string(state)).Error
			}
		}
	}
}
```

The `e.State = ""` plus the explicit `DoUpdates` column list is what keeps `Save`'s full-row overwrite from clobbering the discriminator; the re-read returns the row as it now stands so `Make` sees the preserved state.

- [ ] **Step 6: Add `SetState` to the processor**

In `services/atlas-maps/atlas.com/maps/character/location/processor.go`, add the import `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`, extend the interface:

```go
type Processor interface {
	GetById(characterId uint32) (Model, error)
	Set(characterId uint32, f field.Model) (Model, error)
	SetState(characterId uint32, state characterconst.PresenceState) error
	SetStateIfOnline(characterId uint32, state characterconst.PresenceState) error
	Delete(characterId uint32) error
	Resolve(currentField field.Model) (field.Model, ResolutionReason, error)
}
```

and implement both next to `Set`:

```go
// SetState writes the liveness discriminator unconditionally. Only LOGIN and
// CHANNEL_CHANGED (which genuinely mean "this character is live right now")
// and LOGOUT use this.
func (p *ProcessorImpl) SetState(characterId uint32, state characterconst.PresenceState) error {
	t := tenant.MustFromContext(p.ctx)
	return setLocationState(p.db.WithContext(p.ctx))(t.Id())(characterId)(state, false)
}

// SetStateIfOnline writes the liveness discriminator only when the row is not
// already OFFLINE. Cash-shop transitions use this so a late-delivered
// CHARACTER_EXIT cannot resurrect a logged-off character.
func (p *ProcessorImpl) SetStateIfOnline(characterId uint32, state characterconst.PresenceState) error {
	t := tenant.MustFromContext(p.ctx)
	return setLocationState(p.db.WithContext(p.ctx))(t.Id())(characterId)(state, true)
}
```

- [ ] **Step 7: Add the terminal-OFFLINE test**

Append to `services/atlas-maps/atlas.com/maps/character/location/processor_test.go`:

```go
// TestSetStateIfOnline_DoesNotResurrectOfflineRow is the ordering rule of
// design §1.3: a CHARACTER_EXIT that arrives after LOGOUT must not flip a
// logged-off character back to IN_FIELD.
func TestSetStateIfOnline_DoesNotResurrectOfflineRow(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)

	const characterId uint32 = 45
	f := field.NewBuilder(world.Id(1), 3, _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	p := NewProcessor(logrus.New(), ctx, db)
	if _, err := p.Set(characterId, f); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := p.SetState(characterId, characterconst.PresenceStateOffline); err != nil {
		t.Fatalf("SetState OFFLINE: %v", err)
	}

	// The late CHARACTER_EXIT.
	if err := p.SetStateIfOnline(characterId, characterconst.PresenceStateInField); err != nil {
		t.Fatalf("SetStateIfOnline: %v", err)
	}

	m, err := p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateOffline {
		t.Errorf("late CHARACTER_EXIT resurrected the row to %q, want OFFLINE", m.State())
	}
}

// TestSetStateIfOnline_AppliesWhenOnline is the same call on a live row.
func TestSetStateIfOnline_AppliesWhenOnline(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)

	const characterId uint32 = 46
	f := field.NewBuilder(world.Id(1), 3, _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	p := NewProcessor(logrus.New(), ctx, db)
	if _, err := p.Set(characterId, f); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := p.SetState(characterId, characterconst.PresenceStateInCashShop); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	if err := p.SetStateIfOnline(characterId, characterconst.PresenceStateInField); err != nil {
		t.Fatalf("SetStateIfOnline: %v", err)
	}

	m, err := p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateInField {
		t.Errorf("state = %q, want IN_FIELD", m.State())
	}
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./character/location/ -v`
Expected: PASS, including the pre-existing tests in that package.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/
git commit -m "feat(atlas-maps): add presence state to the character location row"
```

---

## Task 3: `atlas-maps` — project `state` onto the REST surface

`GET /characters/{characterId}/location` gains one attribute. No new endpoint, no new resource type. Adding a JSON key is backward compatible for the endpoint's existing consumers (`atlas-channel`'s session bootstrap, `atlas-character`'s logout path, `atlas-login`'s character-list writer) — they decode into structs that ignore unknown fields.

### Files

- `services/atlas-maps/atlas.com/maps/character/location/rest.go` — add `State` to `RestModel` and to `Transform`
- `services/atlas-maps/atlas.com/maps/character/location/resource_test.go` — assert the attribute round-trips through the handler
- `services/atlas-maps/atlas.com/maps/character/location/resource.go` — read-only; `handleGetCharacterLocation` needs no change (it marshals whatever `Transform` produces)

Module root for `go build`/`go test`: `services/atlas-maps/atlas.com/maps`

**Interfaces:**
- Consumes: `location.Model.State()` from Task 2; `characterconst.PresenceState` from Task 1.
- Produces: JSON attribute `"state"` on resource type `character-locations`. Task 6 (`atlas-channel`'s client) decodes this exact key.

- [ ] **Step 1: Write the failing test**

`resource_test.go` exercises `changeCharacterLocation` directly and has no router/HTTP harness, so do not build one: the projection is what carries the contract, and `handleGetCharacterLocation` marshals whatever `Transform` returns. Test `Transform` and the JSON encoding of `RestModel`.

Append to `services/atlas-maps/atlas.com/maps/character/location/resource_test.go` (internal to package `location`; add imports `encoding/json`, `github.com/Chronicle20/atlas/libs/atlas-constants/channel` and `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`):

```go
// TestTransform_CarriesState proves the discriminator survives the projection
// atlas-channel's /find path reads.
func TestTransform_CarriesState(t *testing.T) {
	m := NewBuilder(1234).
		SetWorldId(world.Id(0)).
		SetChannelId(channel.Id(7)).
		SetMapId(_map.Id(100000000)).
		SetInstance(uuid.Nil).
		SetState(characterconst.PresenceStateInCashShop).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.State != characterconst.PresenceStateInCashShop {
		t.Errorf("State = %q, want IN_CASH_SHOP", rm.State)
	}
	if rm.ChannelId != channel.Id(7) {
		t.Errorf("ChannelId = %d, want 7", rm.ChannelId)
	}
	if rm.GetName() != "character-locations" {
		t.Errorf("GetName = %q, want character-locations", rm.GetName())
	}
}

// TestRestModel_StateJSONKey pins the wire key itself. atlas-channel decodes
// "state"; renaming or re-casing it silently degrades /find to "not findable"
// on every off-channel target, with no error anywhere.
func TestRestModel_StateJSONKey(t *testing.T) {
	rm := RestModel{
		Id:        1234,
		WorldId:   world.Id(0),
		ChannelId: channel.Id(7),
		MapId:     _map.Id(100000000),
		Instance:  uuid.Nil,
		State:     characterconst.PresenceStateInField,
	}

	b, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	got, ok := decoded["state"]
	if !ok {
		t.Fatalf("no state key in %s", string(b))
	}
	if got != "IN_FIELD" {
		t.Errorf("state = %v, want IN_FIELD", got)
	}
	// The pre-existing attributes must still be there — this is an additive
	// change, and atlas-character / atlas-login / the session bootstrap all
	// read them.
	for _, k := range []string{"worldId", "channelId", "mapId", "instance"} {
		if _, ok := decoded[k]; !ok {
			t.Errorf("attribute %q disappeared from %s", k, string(b))
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/location/ -run 'TestTransform_CarriesState|TestRestModel_StateJSONKey' -v`
Expected: FAIL — build error, `rm.State undefined` and `unknown field State in struct literal`.

- [ ] **Step 3: Add the attribute**

In `services/atlas-maps/atlas.com/maps/character/location/rest.go`, add the import `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`, add the field to `RestModel`:

```go
type RestModel struct {
	Id        uint32                       `json:"-"`
	WorldId   world.Id                     `json:"worldId"`
	ChannelId channel.Id                   `json:"channelId"`
	MapId     _map.Id                      `json:"mapId"`
	Instance  uuid.UUID                    `json:"instance"`
	State     characterconst.PresenceState `json:"state"`
}
```

and carry it in `Transform`:

```go
		State:     m.State(),
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./character/location/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/rest.go services/atlas-maps/atlas.com/maps/character/location/resource_test.go
git commit -m "feat(atlas-maps): expose presence state on the location endpoint"
```

---

## Task 4: `atlas-maps` — character-status transitions (LOGIN / LOGOUT / CHANNEL_CHANGED)

Three existing handlers each gain one call. `CHANGE_MAP` and `CREATED` are deliberately left alone: `CHANGE_MAP` is a position write that must not assert liveness, and `CREATED` seeds a row for a character who has never logged in — `OFFLINE` is correct for it.

### Files

- `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go` — add `SetState` to `handleStatusEventLoginFunc`, `handleStatusEventLogoutFunc`, `handleStatusEventChannelChangedFunc`
- `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer_test.go` — new transition tests

Module root for `go build`/`go test`: `services/atlas-maps/atlas.com/maps`

Patterns to copy: `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer_test.go:25` (`newTestDB`) and `:39` (`newTestCtx`) — reuse both, do not redefine them.

**Interfaces:**
- Consumes: `location.Processor.SetState` from Task 2; `characterconst.PresenceState*` from Task 1.
- Produces: nothing new for later tasks.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer_test.go`:

```go
// TestLoginHandler_SetsInField — LOGIN is one of the two events that
// legitimately mean "this character is live right now", so it asserts liveness
// unconditionally.
func TestLoginHandler_SetsInField(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 42
	event := characterKafka.StatusEvent[characterKafka.StatusEventLoginBody]{
		CharacterId: characterId,
		WorldId:     world.Id(1),
		Type:        characterKafka.EventCharacterStatusTypeLogin,
		Body: characterKafka.StatusEventLoginBody{
			ChannelId: channel.Id(7),
			MapId:     _map.Id(100000000),
			Instance:  uuid.Nil,
		},
	}
	handleStatusEventLoginFunc(db)(logger, ctx, event)

	m, err := location.NewProcessor(logger, ctx, db).GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateInField {
		t.Errorf("state = %q, want IN_FIELD", m.State())
	}
	if m.ChannelId() != channel.Id(7) {
		t.Errorf("channel = %d, want 7", m.ChannelId())
	}
}

// TestLogoutHandler_SetsOfflineAndPreservesPosition — LOGOUT persists the
// last-known position (so the next login can restore it) AND marks the
// character offline. Both halves matter: /find must stop reporting a channel,
// but the login path still needs the map.
func TestLogoutHandler_SetsOfflineAndPreservesPosition(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 43
	seed := field.NewBuilder(world.Id(1), channel.Id(4), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	lp := location.NewProcessor(logger, ctx, db)
	if _, err := lp.Set(characterId, seed); err != nil {
		t.Fatalf("seed Set: %v", err)
	}
	if err := lp.SetState(characterId, characterconst.PresenceStateInField); err != nil {
		t.Fatalf("seed SetState: %v", err)
	}

	event := characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]{
		CharacterId: characterId,
		WorldId:     world.Id(1),
		Type:        characterKafka.EventCharacterStatusTypeLogout,
		Body: characterKafka.StatusEventLogoutBody{
			ChannelId: channel.Id(4),
			MapId:     _map.Id(100000000),
			Instance:  uuid.Nil,
		},
	}
	handleStatusEventLogoutFunc(db)(logger, ctx, event)

	m, err := lp.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateOffline {
		t.Errorf("state = %q, want OFFLINE", m.State())
	}
	if m.MapId() == 0 {
		t.Error("LOGOUT discarded the position; the login path needs it")
	}
}

// TestChannelChangedHandler_SetsInFieldOnNewChannel — the other event that
// legitimately asserts liveness. Channel 7 is deliberately neither 0 nor 1, so
// a handler that writes a constant cannot pass.
func TestChannelChangedHandler_SetsInFieldOnNewChannel(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 44
	event := characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]{
		CharacterId: characterId,
		WorldId:     world.Id(1),
		Type:        characterKafka.EventCharacterStatusTypeChannelChanged,
		Body: characterKafka.ChangeChannelEventLoginBody{
			ChannelId:    channel.Id(7),
			OldChannelId: channel.Id(2),
			MapId:        _map.Id(100000000),
			Instance:     uuid.Nil,
		},
	}
	handleStatusEventChannelChangedFunc(db)(logger, ctx, event)

	m, err := location.NewProcessor(logger, ctx, db).GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateInField {
		t.Errorf("state = %q, want IN_FIELD", m.State())
	}
	if m.ChannelId() != channel.Id(7) {
		t.Errorf("channel = %d, want 7", m.ChannelId())
	}
}
```

Add `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"` to the import block. If the `Body` struct field names above do not match `services/atlas-maps/atlas.com/maps/kafka/message/character`, read that package and use its actual field names — do not invent them.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./kafka/consumer/character/ -run 'TestLoginHandler_SetsInField|TestLogoutHandler_SetsOffline|TestChannelChangedHandler_SetsInField' -v`
Expected: FAIL — the state stays `OFFLINE` in the LOGIN and CHANNEL_CHANGED cases.

- [ ] **Step 3: Add the three transitions**

In `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go`, add the import `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`.

In `handleStatusEventLoginFunc`, after the existing `location.NewProcessor(l, ctx, db).Set(...)` block, replace that block with:

```go
			lp := location.NewProcessor(l, ctx, db)
			if _, err := lp.Set(event.CharacterId, f); err != nil {
				l.WithError(err).Warnf("location.Set on LOGIN failed for character [%d].", event.CharacterId)
			}
			if err := lp.SetState(event.CharacterId, characterconst.PresenceStateInField); err != nil {
				l.WithError(err).Warnf("location.SetState on LOGIN failed for character [%d].", event.CharacterId)
			}
```

In `handleStatusEventLogoutFunc`, immediately after the existing `lp.Set(event.CharacterId, resolved)` block and before the `_map.NewProcessor(...)` line:

```go
			if err := lp.SetState(event.CharacterId, characterconst.PresenceStateOffline); err != nil {
				l.WithError(err).Warnf("location.SetState on LOGOUT failed for character [%d].", event.CharacterId)
			}
```

In `handleStatusEventChannelChangedFunc`, replace the existing `location.NewProcessor(l, ctx, db).Set(...)` block with:

```go
			lp := location.NewProcessor(l, ctx, db)
			if _, err := lp.Set(event.CharacterId, newField); err != nil {
				l.WithError(err).Warnf("location.Set on CHANNEL_CHANGED failed for character [%d].", event.CharacterId)
			}
			if err := lp.SetState(event.CharacterId, characterconst.PresenceStateInField); err != nil {
				l.WithError(err).Warnf("location.SetState on CHANNEL_CHANGED failed for character [%d].", event.CharacterId)
			}
```

Leave `handleStatusEventCreatedFunc`, `handleStatusEventDeletedFunc` and `handleChangeMapFunc` unchanged.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./kafka/consumer/character/ -v`
Expected: PASS, including the pre-existing tests in that package.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/character/
git commit -m "feat(atlas-maps): set presence state on login, logout and channel change"
```

---

## Task 5: `atlas-maps` — cash-shop transitions (CHARACTER_ENTER / CHARACTER_EXIT)

The cash-shop consumer currently has no `*gorm.DB` — it passes `nil` to `_map.NewProcessor`. Threading the handle through is most of this task's surface.

Both transitions are **conditional** (`SetStateIfOnline`). The two topics have no mutual ordering guarantee, and disconnecting from inside the cash shop emits no `CHARACTER_EXIT` at all — only a LOGOUT — so an `EXIT` arriving after that LOGOUT must not resurrect the row.

### Files

- `services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/consumer.go` — thread `db`, add both transitions
- `services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/consumer_test.go` — **new file**; transition tests
- `services/atlas-maps/atlas.com/maps/main.go:93` — pass `db` to `cashshop.InitHandlers`

Module root for `go build`/`go test`: `services/atlas-maps/atlas.com/maps`

Patterns to copy: `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go:40` (`InitHandlers(l, db)` — the same signature shape this task gives the cash-shop consumer) and `main.go:90` (its call site).

**Interfaces:**
- Consumes: `location.Processor.SetStateIfOnline` from Task 2.
- Produces: `cashshop.InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error` — a changed signature; `main.go` is updated in the same task.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/consumer_test.go`:

```go
package cashshop

import (
	"atlas-maps/character/location"
	"context"
	"testing"

	cashshopKafka "atlas-maps/kafka/message/cashshop"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := location.Migration(db); err != nil {
		t.Fatalf("location.Migration: %v", err)
	}
	return db
}

func newTestCtx(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

// seedOnline puts a live row on channel 5 in the given state.
func seedOnline(t *testing.T, ctx context.Context, db *gorm.DB, characterId uint32, state characterconst.PresenceState) {
	t.Helper()
	logger, _ := test.NewNullLogger()
	f := field.NewBuilder(world.Id(1), channel.Id(5), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	lp := location.NewProcessor(logger, ctx, db)
	if _, err := lp.Set(characterId, f); err != nil {
		t.Fatalf("seed Set: %v", err)
	}
	if err := lp.SetState(characterId, state); err != nil {
		t.Fatalf("seed SetState: %v", err)
	}
}

func stateOf(t *testing.T, ctx context.Context, db *gorm.DB, characterId uint32) characterconst.PresenceState {
	t.Helper()
	logger, _ := test.NewNullLogger()
	m, err := location.NewProcessor(logger, ctx, db).GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	return m.State()
}

func enterEvent(characterId uint32) cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody] {
	return cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]{
		WorldId: world.Id(1),
		Type:    cashshopKafka.EventCashShopStatusTypeCharacterEnter,
		Body: cashshopKafka.CharacterMovementBody{
			CharacterId: characterId,
			ChannelId:   channel.Id(5),
			MapId:       _map.Id(100000000),
		},
	}
}

func exitEvent(characterId uint32) cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody] {
	e := enterEvent(characterId)
	e.Type = cashshopKafka.EventCashShopStatusTypeCharacterExit
	return e
}

func TestEnterHandler_SetsInCashShop(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 42
	seedOnline(t, ctx, db, characterId, characterconst.PresenceStateInField)

	handleStatusEventEnterFunc(db)(logger, ctx, enterEvent(characterId))

	if got := stateOf(t, ctx, db, characterId); got != characterconst.PresenceStateInCashShop {
		t.Errorf("state = %q, want IN_CASH_SHOP", got)
	}
}

// Kafka delivery is at-least-once; a replayed ENTER must be a no-op.
func TestEnterHandler_IsIdempotent(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 43
	seedOnline(t, ctx, db, characterId, characterconst.PresenceStateInField)

	handleStatusEventEnterFunc(db)(logger, ctx, enterEvent(characterId))
	handleStatusEventEnterFunc(db)(logger, ctx, enterEvent(characterId))

	if got := stateOf(t, ctx, db, characterId); got != characterconst.PresenceStateInCashShop {
		t.Errorf("state after replay = %q, want IN_CASH_SHOP", got)
	}
}

func TestExitHandler_SetsInField(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 44
	seedOnline(t, ctx, db, characterId, characterconst.PresenceStateInCashShop)

	handleStatusEventExitFunc(db)(logger, ctx, exitEvent(characterId))

	if got := stateOf(t, ctx, db, characterId); got != characterconst.PresenceStateInField {
		t.Errorf("state = %q, want IN_FIELD", got)
	}
}

// Design §1.3: OFFLINE is terminal except via LOGIN / CHANNEL_CHANGED.
// Disconnecting from inside the cash shop emits LOGOUT and no CHARACTER_EXIT,
// so an EXIT arriving after a LOGOUT is exactly the late-delivery case.
func TestExitHandler_DoesNotResurrectOfflineCharacter(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 45
	seedOnline(t, ctx, db, characterId, characterconst.PresenceStateOffline)

	handleStatusEventExitFunc(db)(logger, ctx, exitEvent(characterId))

	if got := stateOf(t, ctx, db, characterId); got != characterconst.PresenceStateOffline {
		t.Errorf("late CHARACTER_EXIT resurrected the row to %q, want OFFLINE", got)
	}
}

func TestEnterHandler_DoesNotResurrectOfflineCharacter(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 46
	seedOnline(t, ctx, db, characterId, characterconst.PresenceStateOffline)

	handleStatusEventEnterFunc(db)(logger, ctx, enterEvent(characterId))

	if got := stateOf(t, ctx, db, characterId); got != characterconst.PresenceStateOffline {
		t.Errorf("late CHARACTER_ENTER resurrected the row to %q, want OFFLINE", got)
	}
}
```

If the `cashshopKafka` field names above do not match `services/atlas-maps/atlas.com/maps/kafka/message/cashshop`, read that package and use its actual names.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./kafka/consumer/cashshop/ -v`
Expected: FAIL — build error, `undefined: handleStatusEventEnterFunc`.

- [ ] **Step 3: Thread `db` and add the transitions**

In `services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/consumer.go`, add the imports `"atlas-maps/character/location"`, `"gorm.io/gorm"` and `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`, then change `InitHandlers` and convert both handlers into `db`-closing constructors:

```go
func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(cashshopKafka.EnvEventTopicCashShopStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventEnterFunc(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExitFunc(db)))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventEnterFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]) {
		if event.Type != cashshopKafka.EventCashShopStatusTypeCharacterEnter {
			return
		}
		l.Debugf("Character [%d] has entered cash shop.", event.Body.CharacterId)
		transactionId := uuid.New()
		f := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).Build()
		p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), nil)
		_ = p.ExitAndEmit(transactionId, f, event.Body.CharacterId)

		// Conditional: OFFLINE is terminal except via LOGIN / CHANNEL_CHANGED.
		// The cash-shop and character status topics have no mutual ordering
		// guarantee, so a late ENTER must not resurrect a logged-off row.
		if err := location.NewProcessor(l, ctx, db).SetStateIfOnline(event.Body.CharacterId, characterconst.PresenceStateInCashShop); err != nil {
			l.WithError(err).Warnf("location.SetStateIfOnline on CHARACTER_ENTER failed for character [%d].", event.Body.CharacterId)
		}
	}
}

func handleStatusEventExitFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]) {
		if event.Type != cashshopKafka.EventCashShopStatusTypeCharacterExit {
			return
		}
		l.Debugf("Character [%d] has exited cash shop.", event.Body.CharacterId)
		transactionId := uuid.New()
		f := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).Build()
		p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), nil)
		_ = p.EnterAndEmit(transactionId, f, event.Body.CharacterId)

		// Conditional for the same reason: disconnecting from inside the cash
		// shop emits LOGOUT and no CHARACTER_EXIT, so an EXIT that arrives
		// after that LOGOUT must leave the row OFFLINE.
		if err := location.NewProcessor(l, ctx, db).SetStateIfOnline(event.Body.CharacterId, characterconst.PresenceStateInField); err != nil {
			l.WithError(err).Warnf("location.SetStateIfOnline on CHARACTER_EXIT failed for character [%d].", event.Body.CharacterId)
		}
	}
}
```

The `nil` passed to `_map.NewProcessor` is pre-existing and stays: that processor's in-memory registry work does not use the handle.

- [ ] **Step 4: Update the call site**

In `services/atlas-maps/atlas.com/maps/main.go`, change line 93 from `cashshop.InitHandlers(l)` to:

```go
	if err := cashshop.InitHandlers(l, db)(consumer.GetManager().RegisterHandler); err != nil {
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./...`
Expected: PASS across the whole module.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/ services/atlas-maps/atlas.com/maps/main.go
git commit -m "feat(atlas-maps): set presence state on cash shop entry and exit"
```

---

## Task 6: `atlas-channel` — state-bearing location client

`GetField` returns a `field.Model`, which has nowhere to carry the state. This adds a sibling `Get` returning a richer model. `GetField` is kept exactly as-is for its existing callers (session bootstrap, `ResolveMapId`).

A response with an empty or absent `state` resolves to `OFFLINE`, so an `atlas-maps` that has not yet been redeployed degrades `/find` to "not findable" rather than to a fabricated channel.

### Files

- `services/atlas-channel/atlas.com/channel/maps/location/requests.go` — add `State` to `RestModel`, add the `Model` type and `Get`
- `services/atlas-channel/atlas.com/channel/maps/location/requests_test.go` — new test cases
- `services/atlas-channel/atlas.com/channel/maps/location/resolve.go` — read-only; `ResolveMapId` must NOT be modified

Module root for `go build`/`go test`: `services/atlas-channel/atlas.com/channel`

Patterns to copy: `services/atlas-channel/atlas.com/channel/maps/location/requests.go:75` (`GetField` — the `ErrNotFound` translation `Get` repeats) and `:88` (`SetBaseURLForTest`, which the tests use).

**Interfaces:**
- Consumes: the `"state"` JSON attribute from Task 3; `characterconst.PresenceState` and `ParsePresenceState` from Task 1.
- Produces:
  - `location.Model` with getters `CharacterId() uint32`, `WorldId() world.Id`, `ChannelId() channel.Id`, `MapId() _map.Id`, `Instance() uuid.UUID`, `State() characterconst.PresenceState`
  - `func Get(l logrus.FieldLogger, ctx context.Context, characterId uint32) (Model, error)`, returning `ErrNotFound` on HTTP 404
  - Task 8 depends on both.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/maps/location/requests_test.go`. It already has an `httptest` + `SetBaseURLForTest` harness and imports `context`, `errors`, `net/http`, `net/http/httptest`, `strings`, `testing`, `uuid`, `logrus` and `require`; add only `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`.

```go
// serveLocation stands up an atlas-maps stub returning one character-locations
// document. attrs is spliced in verbatim so a test can omit "state" entirely.
func serveLocation(t *testing.T, attrs string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasSuffix(r.URL.Path, "/characters/1234/location"), "path: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "character-locations",
				"id": "1234",
				"attributes": {` + attrs + `}
			}
		}`))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(SetBaseURLForTest(srv.URL))
}

// TestGet_DecodesState pins the wire contract atlas-maps publishes.
func TestGet_DecodesState(t *testing.T) {
	serveLocation(t, `
		"worldId": 0,
		"channelId": 7,
		"mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"state": "IN_CASH_SHOP"`)

	m, err := Get(logrus.New(), context.Background(), 1234)
	require.NoError(t, err)
	require.Equal(t, characterconst.PresenceStateInCashShop, m.State())
	require.Equal(t, uint8(7), uint8(m.ChannelId()))
	require.Equal(t, uint32(100000000), uint32(m.MapId()))
	require.Equal(t, uuid.Nil, m.Instance())
}

// TestGet_DecodesInField — channel 7 deliberately, not 0 or 1: the bug being
// fixed is a hard-coded 0 on the channel arm, and the client adds one for
// display, so a fixture on 0 or 1 passes against the broken code.
func TestGet_DecodesInField(t *testing.T) {
	serveLocation(t, `
		"worldId": 0,
		"channelId": 7,
		"mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"state": "IN_FIELD"`)

	m, err := Get(logrus.New(), context.Background(), 1234)
	require.NoError(t, err)
	require.Equal(t, characterconst.PresenceStateInField, m.State())
	require.Equal(t, uint8(7), uint8(m.ChannelId()))
}

// TestGet_AbsentStateIsOffline covers an atlas-maps that has not been
// redeployed yet: /find must degrade to "not findable", never to a fabricated
// channel. The channel is still decoded — it is simply not trusted.
func TestGet_AbsentStateIsOffline(t *testing.T) {
	serveLocation(t, `
		"worldId": 0,
		"channelId": 7,
		"mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000"`)

	m, err := Get(logrus.New(), context.Background(), 1234)
	require.NoError(t, err)
	require.Equal(t, characterconst.PresenceStateOffline, m.State())
	require.Equal(t, uint8(7), uint8(m.ChannelId()))
}

// TestGet_UnrecognisedStateIsOffline — same failure direction for a value this
// build does not know.
func TestGet_UnrecognisedStateIsOffline(t *testing.T) {
	serveLocation(t, `
		"worldId": 0,
		"channelId": 7,
		"mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"state": "IN_ORBIT"`)

	m, err := Get(logrus.New(), context.Background(), 1234)
	require.NoError(t, err)
	require.Equal(t, characterconst.PresenceStateOffline, m.State())
}

// TestGet_NotFoundIsErrNotFound — 404 means "no row at all", i.e. a character
// who has never logged in. Task 8 logs it as a distinct branch from OFFLINE.
func TestGet_NotFoundIsErrNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(SetBaseURLForTest(srv.URL))

	_, err := Get(logrus.New(), context.Background(), 1234)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got %v", err)
}

// TestGet_InfrastructureErrorIsNotErrNotFound — a 5xx must stay distinguishable
// from a missing row: /find logs the two at different levels.
func TestGet_InfrastructureErrorIsNotErrNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(SetBaseURLForTest(srv.URL))

	_, err := Get(logrus.New(), context.Background(), 1234)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrNotFound), "expected non-ErrNotFound, got ErrNotFound")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./maps/location/ -run TestGet_ -v`
Expected: FAIL — build error, `undefined: Get`.

- [ ] **Step 3: Add the state attribute, the model, and `Get`**

In `services/atlas-channel/atlas.com/channel/maps/location/requests.go`, add the import `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`, add the field to `RestModel`:

```go
	State     string     `json:"state"`
```

`State` is a plain `string` on the REST model on purpose: an unrecognised or absent value must decode without error and then be narrowed by `ParsePresenceState`.

Add the domain model and the accessor below `GetField`:

```go
// Model is the state-bearing projection of atlas-maps's character location.
// field.Model has nowhere to carry the presence discriminator, so /find reads
// this instead; GetField is unchanged for its existing callers.
type Model struct {
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	mapId       _map.Id
	instance    uuid.UUID
	state       characterconst.PresenceState
}

func (m Model) CharacterId() uint32                  { return m.characterId }
func (m Model) WorldId() world.Id                    { return m.worldId }
func (m Model) ChannelId() channel.Id                { return m.channelId }
func (m Model) MapId() _map.Id                       { return m.mapId }
func (m Model) Instance() uuid.UUID                  { return m.instance }
func (m Model) State() characterconst.PresenceState  { return m.state }

// Get returns the character's stored location including the presence state.
//
// On HTTP 404 (no location row at all — a character who has never logged in)
// it returns ErrNotFound. On any other error (5xx, network, decode) it returns
// the underlying error, so callers can distinguish infrastructure failure from
// missing data; /find logs those two at different levels.
//
// An absent or unrecognised state resolves to OFFLINE, so an atlas-maps that
// has not been redeployed degrades /find to "not findable" rather than to a
// fabricated channel.
func Get(l logrus.FieldLogger, ctx context.Context, characterId uint32) (Model, error) {
	rm, err := requestByCharacterId(ctx, characterId)(l, ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return Model{}, ErrNotFound
		}
		return Model{}, err
	}
	return Model{
		characterId: characterId,
		worldId:     rm.WorldId,
		channelId:   rm.ChannelId,
		mapId:       rm.MapId,
		instance:    rm.Instance,
		state:       characterconst.ParsePresenceState(rm.State),
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./maps/location/ -v`
Expected: PASS, including the pre-existing tests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/maps/location/
git commit -m "feat(atlas-channel): read presence state from the maps location endpoint"
```

---

## Task 7: `atlas-channel` — correct GM level semantics

`character.Model.Gm()` returns `m.gm == 1`, so a GM at level 2 reads as **not** a GM. For `/find`'s concealment gate that leaks exactly the accounts it most needs to hide. `GmLevel` exists elsewhere in the repo as an int (`atlas-query-aggregator`, `libs/atlas-saga/validation.go:20`), confirming levels above 1 are meaningful.

Widening to `m.gm > 0` was approved repo-wide during `/plan-task`. Its three non-test callers — `kafka/consumer/session/consumer.go:212` (session GM flag), `kafka/consumer/message/consumer.go:99` and `:178` (GM chat colouring) — all want "is this player a GM", for which `> 0` is strictly more correct.

`session.Model` carries `gm bool` but exposes only `setGm`; `/find` needs to read the requester's flag, so a getter is added.

### Files

- `services/atlas-channel/atlas.com/channel/character/model.go:64` — widen `Gm()`, add `GmLevel()`
- `services/atlas-channel/atlas.com/channel/character/builder_test.go` — new test cases
- `services/atlas-channel/atlas.com/channel/session/model.go:103` — add a `Gm()` getter next to `setGm`
- `services/atlas-channel/atlas.com/channel/session/model_test.go` — **new file**; getter test
- `services/atlas-channel/atlas.com/channel/character/builder.go:137` — read-only; `SetGm(int)` already exists

Module root for `go build`/`go test`: `services/atlas-channel/atlas.com/channel`

**Interfaces:**
- Consumes: nothing.
- Produces: `character.Model.Gm() bool` (now `gm > 0`), `character.Model.GmLevel() int`, `session.Model.Gm() bool`. Task 8 uses `Gm()` on both types.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/character/builder_test.go` (reuse whatever builder entry point the file already uses to construct a `Model`):

```go
// TestGm_IsTrueForEveryLevelAboveZero — gm == 1 was the old predicate, which
// classified a level-2 GM as an ordinary player. /find's concealment gate
// would then leak exactly the accounts it exists to hide.
func TestGm_IsTrueForEveryLevelAboveZero(t *testing.T) {
	cases := []struct {
		level int
		want  bool
	}{
		{0, false},
		{1, true},
		{2, true},
		{5, true},
	}
	for _, c := range cases {
		m := NewModelBuilder().SetGm(c.level).Build()
		if got := m.Gm(); got != c.want {
			t.Errorf("gm level %d: Gm() = %t, want %t", c.level, got, c.want)
		}
		if got := m.GmLevel(); got != c.level {
			t.Errorf("gm level %d: GmLevel() = %d, want %d", c.level, got, c.level)
		}
	}
}
```

Replace `NewModelBuilder()` with the actual constructor `builder_test.go` already uses.

Create `services/atlas-channel/atlas.com/channel/session/model_test.go`:

```go
package session

import "testing"

// TestGm_Accessor — /find reads the requester's GM flag from the session
// (PRD FR-3). The field existed but had no getter.
func TestGm_Accessor(t *testing.T) {
	var s Model
	if s.Gm() {
		t.Error("zero-value session reports Gm() = true")
	}
	promoted := s.setGm(true)
	if !promoted.Gm() {
		t.Error("after setGm(true), Gm() = false")
	}
	if s.Gm() {
		t.Error("setGm mutated the receiver; Model is meant to be immutable")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/ ./session/ -run 'TestGm_' -v`
Expected: FAIL — `m.GmLevel undefined`, `s.Gm undefined`, and the `{2, true}` case failing against `gm == 1`.

- [ ] **Step 3: Widen `Gm()` and add `GmLevel()`**

In `services/atlas-channel/atlas.com/channel/character/model.go`, replace:

```go
func (m Model) Gm() bool {
	return m.gm == 1
}
```

with:

```go
// Gm reports whether this character has any GM level at all. It is deliberately
// `> 0` rather than `== 1`: GM levels above 1 exist in this repo (see
// libs/atlas-saga/validation.go and atlas-query-aggregator's character model),
// and `== 1` classified those accounts as ordinary players — which, for
// /find's concealment gate, leaked exactly the accounts it exists to hide.
func (m Model) Gm() bool {
	return m.gm > 0
}

// GmLevel returns the raw GM level. Prefer Gm() for visibility decisions:
// GM visibility is a boolean predicate on level > 0, not a tier comparison.
func (m Model) GmLevel() int {
	return m.gm
}
```

- [ ] **Step 4: Add the session getter**

In `services/atlas-channel/atlas.com/channel/session/model.go`, add above `setGm`:

```go
// Gm reports whether this session's character is a GM. The flag is set at
// login bootstrap from character.Model.Gm(), so it inherits that predicate.
func (s *Model) Gm() bool {
	return s.gm
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./character/ ./session/ ./kafka/... -v`
Expected: PASS. The `kafka/...` packages are included because they hold the three `Gm()` callers whose semantics just widened.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/model.go services/atlas-channel/atlas.com/channel/character/builder_test.go services/atlas-channel/atlas.com/channel/session/model.go services/atlas-channel/atlas.com/channel/session/model_test.go
git commit -m "fix(atlas-channel): treat every GM level above zero as a GM"
```

---

## Task 8: `atlas-channel` — the `/find` decision table

The core of the task. `produceFindResultBody` today constructs its collaborators inline, which is why none of FR-1…FR-7 is testable. It splits in two: a pure `findDecision` over three injected lookups, and a thin adapter that projects the outcome onto the packet constructor.

Both existing placeholder comments (the cash-shop query and the remote-channel lookup) are removed outright — not moved, not reworded.

**The decision table** (evaluated in order, first match wins):

| # | Condition | Outcome | Branch name |
|---|---|---|---|
| FR-1 | name does not resolve | error shape, requested name echoed | `unresolved` |
| FR-2 | `target.WorldId() != s.Field().WorldId()` | error shape | `cross-world` |
| FR-3 | `target.Gm() && !s.Gm()` | error shape | `gm-concealed` |
| FR-4a | local session exists **and** `CashScene() != CashSceneNone` | cash-shop shape | `cash-shop-local` |
| FR-5 | local session exists **and** `CashScene() == CashSceneNone` | map shape (`WithXY` on `0x09`) | `map-local` |
| FR-4b | `location.State() == IN_CASH_SHOP` | cash-shop shape | `cash-shop-remote` |
| FR-6 | `location.State() == IN_FIELD` | channel shape, `location.ChannelId()` | `channel-remote` |
| FR-7a | `location.State() == OFFLINE` | error shape | `offline` |
| FR-7b | `ErrNotFound` | error shape | `never-logged-in` |
| FR-7c | any other lookup error | error shape, logged at error level | `lookup-failed` |

The local session is consulted before the location row: a live local session is by construction authoritative about both liveness and cash-scene, and costs nothing. `location.ResolveMapId` is **not** used — its collapse-to-map-0 is what lets a transport failure render as a real location.

Channel id conversion is `uint32(loc.ChannelId())` and nothing else: `WhisperFindResultChannel.Encode` writes `w.WriteInt(m.channelId)` with no adjustment (`libs/atlas-packet/field/clientbound/whisper.go:227`) and the client adds one for display; `channel.Id` is already the 0-based internal value. The existing hard-coded `0` is why every off-channel target reads as "channel 1".

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper.go` — the rewrite
- `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper_test.go` — **new file**; the decision table
- `services/atlas-channel/atlas.com/channel/maps/location/requests.go` — add `NewModelForTest` (the tests need to build a `location.Model` directly; see Step 1's closing note)
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change.go:26` — read-only; the `var …Func` seam pattern to copy
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change_test.go` — read-only; the seam-swap-and-restore pattern to copy (the `orig := …Func` / `t.Cleanup` block around line 108)
- `services/atlas-channel/atlas.com/channel/session/registry_test_helper.go` — read-only; `AddSessionToRegistry` / `ClearRegistryForTenant`

Module root for `go build`/`go test`: `services/atlas-channel/atlas.com/channel`

**Interfaces:**
- Consumes: `location.Get` and `location.Model` from Task 6; `character.Model.Gm()` and `session.Model.Gm()` from Task 7; `characterconst.PresenceState*` from Task 1.
- Produces: nothing later tasks depend on.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper_test.go`, following the env-struct pattern of `cash_shop_check_name_change_test.go`. That file's harness pieces are reused as-is: `mustTenant` (`character_cash_item_use_test.go:23`), `discardConn` (`character_damage_test.go:524`), `session.NewSession`, `session.AddSessionToRegistry` / `session.ClearRegistryForTenant`, and the capturing `writer.Producer` shape.

The session processor setters this needs already exist: `SetAccountId`, `SetCharacterId`, `SetField`, `SetCashScene`, `SetGm` (`session/processor.go:55-59`).

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/maps/location"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	chat "github.com/Chronicle20/atlas/libs/atlas-packet/chat/serverbound"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	findRequesterId   = uint32(7100)
	findRequesterAcct = uint32(5100)
	findTargetId      = uint32(7200)
	findWorldId       = world.Id(0)
	// The requester sits on channel 2 and the remote target on channel 7.
	// 7 is deliberately neither 0 nor 1: the bug being fixed is a hard-coded 0
	// on the channel arm, and the client adds one for display, so a fixture on
	// 0 or 1 passes against the broken code.
	findRequesterChannel = channel.Id(2)
	findRemoteChannel    = channel.Id(7)
)

type findEnv struct {
	t         *testing.T
	ctx       context.Context
	s         session.Model
	l         logrus.FieldLogger
	logs      *bytes.Buffer
	wp        writer.Producer
	announced [][]byte

	// seam returns
	target     character.Model
	targetErr  error
	localSess  session.Model
	localErr   error
	loc        location.Model
	locErr     error
	locCalls   int
	tenantId   uuid.UUID
	sessionId  uuid.UUID
}

func newFindEnv(t *testing.T) *findEnv {
	t.Helper()

	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, discardConn{})
	session.AddSessionToRegistry(ten.Id(), s)
	t.Cleanup(func() { session.ClearRegistryForTenant(ten.Id()) })

	logs := &bytes.Buffer{}
	l := logrus.New()
	l.SetOutput(logs)
	l.SetLevel(logrus.DebugLevel)

	sp := session.NewProcessor(l, ctx)
	sp.SetAccountId(sessionId, findRequesterAcct)
	sp.SetCharacterId(sessionId, findRequesterId)
	f := field.NewBuilder(findWorldId, findRequesterChannel, _map.Id(100000000)).Build()
	updated := session.NewProcessor(l, ctx).SetField(sessionId, f)

	env := &findEnv{t: t, ctx: ctx, s: updated, l: l, logs: logs, tenantId: ten.Id(), sessionId: sessionId}

	env.wp = func(name string) (swriter.BodyFunc, error) {
		return func(bl logrus.FieldLogger, bctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(bl, bctx)(nil)
				env.announced = append(env.announced, b)
				return b
			}
		}, nil
	}

	// Defaults: the target resolves, is in the requester's world, is not a GM,
	// has no local session, and has no location row. Each subtest overrides
	// only what its rule needs.
	env.target = character.NewModelBuilder().
		SetId(findTargetId).
		SetName("Bob").
		SetWorldId(findWorldId).
		SetGm(0).
		SetX(250).
		SetY(-75).
		Build()
	env.localErr = errors.New("no local session")
	env.locErr = location.ErrNotFound

	origName := findCharacterByNameFunc
	findCharacterByNameFunc = func(_ logrus.FieldLogger, _ context.Context, _ string) (character.Model, error) {
		return env.target, env.targetErr
	}
	t.Cleanup(func() { findCharacterByNameFunc = origName })

	origSess := findLocalSessionFunc
	findLocalSessionFunc = func(_ logrus.FieldLogger, _ context.Context, _ channel.Model, _ uint32) (session.Model, error) {
		return env.localSess, env.localErr
	}
	t.Cleanup(func() { findLocalSessionFunc = origSess })

	origLoc := findCharacterLocationFunc
	findCharacterLocationFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (location.Model, error) {
		env.locCalls++
		return env.loc, env.locErr
	}
	t.Cleanup(func() { findCharacterLocationFunc = origLoc })

	return env
}

// dispatch drives the real handler for one arm and returns the announced body.
func (e *findEnv) dispatch(mode chat.WhisperMode, targetName string) []byte {
	e.t.Helper()
	e.announced = nil
	if err := produceFindResultBody(e.l)(e.ctx)(e.wp)(mode, targetName)(e.s); err != nil {
		e.t.Fatalf("produceFindResultBody: %v", err)
	}
	if len(e.announced) != 1 {
		e.t.Fatalf("announced %d packets, want 1", len(e.announced))
	}
	return e.announced[0]
}

// The two find arms. 0x09 is the chat /find; 0x48 is the buddy-window find.
var findArms = []struct {
	name string
	mode chat.WhisperMode
	echo byte
}{
	{"chat arm", chat.WhisperModeFind, 0x09},
	{"buddy window arm", chat.WhisperModeBuddyWindowFind, 0x48},
}

// decodeFindError decodes an error-shape body and returns the echoed name.
func decodeFindError(t *testing.T, ctx context.Context, l logrus.FieldLogger, b []byte) (byte, string) {
	t.Helper()
	var m fieldcb.WhisperFindResultError
	m.Decode(l, ctx)(request.Request(b).Reader(), nil)
	return m.Mode(), m.TargetName()
}

func decodeFindChannel(t *testing.T, ctx context.Context, l logrus.FieldLogger, b []byte) fieldcb.WhisperFindResultChannel {
	t.Helper()
	var m fieldcb.WhisperFindResultChannel
	m.Decode(l, ctx)(request.Request(b).Reader(), nil)
	return m
}

func decodeFindCashShop(t *testing.T, ctx context.Context, l logrus.FieldLogger, b []byte) fieldcb.WhisperFindResultCashShop {
	t.Helper()
	var m fieldcb.WhisperFindResultCashShop
	m.Decode(l, ctx)(request.Request(b).Reader(), nil)
	return m
}

// FR-1: an unresolvable name answers the error shape and echoes the REQUESTED
// name — there is no resolved one.
func TestFind_FR1_UnresolvableName(t *testing.T) {
	for _, arm := range findArms {
		t.Run(arm.name, func(t *testing.T) {
			env := newFindEnv(t)
			env.targetErr = errors.New("no such character")

			mode, name := decodeFindError(t, env.ctx, env.l, env.dispatch(arm.mode, "Ghost"))
			if mode != arm.echo {
				t.Errorf("mode = %#x, want %#x", mode, arm.echo)
			}
			if name != "Ghost" {
				t.Errorf("echoed name = %q, want the requested name %q", name, "Ghost")
			}
			if !bytes.Contains(env.logs.Bytes(), []byte("unresolved")) {
				t.Errorf("no branch=unresolved log line in %s", env.logs.String())
			}
		})
	}
}

// FR-2: a cross-world target is wire-identical to FR-1 by design (PRD §8) but
// must be a DISTINCT branch — otherwise it is FR-1 passing by accident.
func TestFind_FR2_CrossWorld(t *testing.T) {
	for _, arm := range findArms {
		t.Run(arm.name, func(t *testing.T) {
			env := newFindEnv(t)
			env.target = character.NewModelBuilder().
				SetId(findTargetId).SetName("Bob").SetWorldId(world.Id(1)).SetGm(0).Build()

			mode, name := decodeFindError(t, env.ctx, env.l, env.dispatch(arm.mode, "Bob"))
			if mode != arm.echo {
				t.Errorf("mode = %#x, want %#x", mode, arm.echo)
			}
			if name != "Bob" {
				t.Errorf("echoed name = %q, want Bob", name)
			}
			if !bytes.Contains(env.logs.Bytes(), []byte("cross-world")) {
				t.Errorf("no branch=cross-world log line in %s", env.logs.String())
			}
			if env.locCalls != 0 {
				t.Errorf("location looked up %d times for a cross-world target, want 0", env.locCalls)
			}
		})
	}
}

// FR-3: GM visibility is a boolean predicate on level > 0, not a tier
// comparison. The gm=2 case is the whole point — it passes against the old
// `gm == 1` predicate, which classified a level-2 GM as an ordinary player.
func TestFind_FR3_GmConcealment(t *testing.T) {
	cases := []struct {
		name         string
		targetGm     int
		requesterGm  bool
		wantConcealed bool
	}{
		{"gm 1 target, ordinary requester", 1, false, true},
		{"gm 2 target, ordinary requester", 2, false, true},
		{"gm 1 target, gm requester", 1, true, false},
		{"gm 2 target, gm requester", 2, true, false},
		{"ordinary target, ordinary requester", 0, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := newFindEnv(t)
			env.target = character.NewModelBuilder().
				SetId(findTargetId).SetName("Bob").SetWorldId(findWorldId).SetGm(c.targetGm).Build()
			if c.requesterGm {
				env.s = session.NewProcessor(env.l, env.ctx).SetGm(env.sessionId, true)
			}
			env.loc = locationFixture(characterconst.PresenceStateInField, findRemoteChannel)
			env.locErr = nil

			b := env.dispatch(chat.WhisperModeFind, "Bob")
			if c.wantConcealed {
				if _, name := decodeFindError(t, env.ctx, env.l, b); name != "Bob" {
					t.Errorf("echoed name = %q, want Bob", name)
				}
				if !bytes.Contains(env.logs.Bytes(), []byte("gm-concealed")) {
					t.Errorf("no branch=gm-concealed log line in %s", env.logs.String())
				}
				return
			}
			if got := decodeFindChannel(t, env.ctx, env.l, b); got.ChannelId() != uint32(findRemoteChannel) {
				t.Errorf("channel = %d, want %d", got.ChannelId(), findRemoteChannel)
			}
		})
	}
}

// FR-4a: a live local session in a cash scene answers the cash-shop shape
// without consulting the location row at all.
func TestFind_FR4a_LocalCashScene(t *testing.T) {
	scenes := []struct {
		name  string
		scene byte
	}{
		{"cash shop", session.CashSceneCashShop},
		// The ITC renders inside the cash-shop CStage and emits the identical
		// CHARACTER_ENTER, so MTS folds into the same shape (FR-11).
		{"mts", session.CashSceneMts},
	}
	for _, sc := range scenes {
		for _, arm := range findArms {
			t.Run(sc.name+" "+arm.name, func(t *testing.T) {
				env := newFindEnv(t)
				targetSessionId := uuid.New()
				ts := session.NewSession(targetSessionId, mustTenant(t, "GMS", 83, 1), 0, discardConn{})
				session.AddSessionToRegistry(env.tenantId, ts)
				env.localSess = session.NewProcessor(env.l, env.ctx).SetCashScene(targetSessionId, sc.scene)
				env.localErr = nil

				got := decodeFindCashShop(t, env.ctx, env.l, env.dispatch(arm.mode, "Bob"))
				if got.Mode() != arm.echo {
					t.Errorf("mode = %#x, want %#x", got.Mode(), arm.echo)
				}
				if got.TargetName() != "Bob" {
					t.Errorf("name = %q, want Bob", got.TargetName())
				}
				if env.locCalls != 0 {
					t.Errorf("location looked up %d times for a local cash-scene target, want 0", env.locCalls)
				}
			})
		}
	}
}

// FR-5: a live local session NOT in a cash scene answers the map shape. The
// 0x09 arm carries x/y; the 0x48 arm does not.
func TestFind_FR5_LocalOnMap(t *testing.T) {
	for _, arm := range findArms {
		t.Run(arm.name, func(t *testing.T) {
			env := newFindEnv(t)
			targetSessionId := uuid.New()
			ts := session.NewSession(targetSessionId, mustTenant(t, "GMS", 83, 1), 0, discardConn{})
			session.AddSessionToRegistry(env.tenantId, ts)
			env.localSess = session.NewProcessor(env.l, env.ctx).SetCashScene(targetSessionId, session.CashSceneNone)
			env.localErr = nil
			env.loc = locationFixture(characterconst.PresenceStateInField, findRequesterChannel)
			env.locErr = nil

			var m fieldcb.WhisperFindResultMap
			b := env.dispatch(arm.mode, "Bob")
			m.Decode(env.l, env.ctx)(request.Request(b).Reader(), nil)
			if m.Mode() != arm.echo {
				t.Errorf("mode = %#x, want %#x", m.Mode(), arm.echo)
			}
			if m.MapId() != 100000000 {
				t.Errorf("mapId = %d, want 100000000", m.MapId())
			}

			// Assert the x/y presence by wire length rather than by decoding,
			// because WhisperFindResultMap.Decode reads x/y only when the
			// receiving struct already has includeXY set.
			withoutXY := len(b)
			if arm.echo == 0x09 {
				if withoutXY != mapBodyLen("Bob")+8 {
					t.Errorf("0x09 body is %d bytes; want map body + 8 for x/y", withoutXY)
				}
			} else if withoutXY != mapBodyLen("Bob") {
				t.Errorf("0x48 body is %d bytes; want map body with NO x/y", withoutXY)
			}
		})
	}
}

// FR-4b: no local session, but the location row says the target is in the
// cash shop.
func TestFind_FR4b_RemoteCashShop(t *testing.T) {
	for _, arm := range findArms {
		t.Run(arm.name, func(t *testing.T) {
			env := newFindEnv(t)
			env.loc = locationFixture(characterconst.PresenceStateInCashShop, findRemoteChannel)
			env.locErr = nil

			got := decodeFindCashShop(t, env.ctx, env.l, env.dispatch(arm.mode, "Bob"))
			if got.Mode() != arm.echo || got.TargetName() != "Bob" {
				t.Errorf("mode=%#x name=%q, want %#x Bob", got.Mode(), got.TargetName(), arm.echo)
			}
			if !bytes.Contains(env.logs.Bytes(), []byte("cash-shop-remote")) {
				t.Errorf("no branch=cash-shop-remote log line in %s", env.logs.String())
			}
		})
	}
}

// FR-6: the bug this task exists to fix. The channel arm must carry the
// target's REAL channel, not a hard-coded 0.
func TestFind_FR6_RemoteChannel(t *testing.T) {
	for _, arm := range findArms {
		t.Run(arm.name, func(t *testing.T) {
			env := newFindEnv(t)
			env.loc = locationFixture(characterconst.PresenceStateInField, findRemoteChannel)
			env.locErr = nil

			got := decodeFindChannel(t, env.ctx, env.l, env.dispatch(arm.mode, "Bob"))
			if got.Mode() != arm.echo {
				t.Errorf("mode = %#x, want %#x", got.Mode(), arm.echo)
			}
			if got.ChannelId() != uint32(findRemoteChannel) {
				t.Errorf("channelId = %d, want %d (a hard-coded 0 must not pass)", got.ChannelId(), findRemoteChannel)
			}
		})
	}
}

// FR-7: the three not-findable branches. All three answer the same wire shape
// and differ only in the branch name and log level.
func TestFind_FR7_NotFindable(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(*findEnv)
		wantBranch string
	}{
		{
			name: "offline",
			setup: func(e *findEnv) {
				e.loc = locationFixture(characterconst.PresenceStateOffline, findRemoteChannel)
				e.locErr = nil
			},
			wantBranch: "offline",
		},
		{
			name:       "never logged in",
			setup:      func(e *findEnv) { e.locErr = location.ErrNotFound },
			wantBranch: "never-logged-in",
		},
		{
			name:       "lookup failed",
			setup:      func(e *findEnv) { e.locErr = errors.New("500 from atlas-maps") },
			wantBranch: "lookup-failed",
		},
	}
	for _, c := range cases {
		for _, arm := range findArms {
			t.Run(c.name+" "+arm.name, func(t *testing.T) {
				env := newFindEnv(t)
				c.setup(env)

				mode, name := decodeFindError(t, env.ctx, env.l, env.dispatch(arm.mode, "Bob"))
				if mode != arm.echo {
					t.Errorf("mode = %#x, want %#x", mode, arm.echo)
				}
				if name != "Bob" {
					t.Errorf("echoed name = %q, want Bob", name)
				}
				if !bytes.Contains(env.logs.Bytes(), []byte(c.wantBranch)) {
					t.Errorf("no branch=%s log line in %s", c.wantBranch, env.logs.String())
				}
				if c.wantBranch == "lookup-failed" && !bytes.Contains(env.logs.Bytes(), []byte("level=error")) {
					t.Errorf("infrastructure failure was not logged at error level: %s", env.logs.String())
				}
			})
		}
	}
}

// The two arms differ only in the echoed mode byte for every outcome except
// the map shape, where 0x09 additionally carries x/y.
func TestFind_ArmSymmetry(t *testing.T) {
	setups := map[string]func(*findEnv){
		"cash shop": func(e *findEnv) {
			e.loc = locationFixture(characterconst.PresenceStateInCashShop, findRemoteChannel)
			e.locErr = nil
		},
		"channel": func(e *findEnv) {
			e.loc = locationFixture(characterconst.PresenceStateInField, findRemoteChannel)
			e.locErr = nil
		},
		"error": func(e *findEnv) { e.locErr = location.ErrNotFound },
	}
	for name, setup := range setups {
		t.Run(name, func(t *testing.T) {
			envA := newFindEnv(t)
			setup(envA)
			a := envA.dispatch(chat.WhisperModeFind, "Bob")

			envB := newFindEnv(t)
			setup(envB)
			b := envB.dispatch(chat.WhisperModeBuddyWindowFind, "Bob")

			if len(a) != len(b) {
				t.Fatalf("arm bodies differ in length: 0x09 %d, 0x48 %d", len(a), len(b))
			}
			if a[0] != 0x09 || b[0] != 0x48 {
				t.Fatalf("mode bytes = %#x / %#x, want 0x09 / 0x48", a[0], b[0])
			}
			if !bytes.Equal(a[1:], b[1:]) {
				t.Errorf("arm bodies differ past the mode byte:\n 0x09 %v\n 0x48 %v", a[1:], b[1:])
			}
		})
	}
}

// locationFixture builds a location.Model in the given state on the given
// channel, at map 100000000.
func locationFixture(state characterconst.PresenceState, ch channel.Id) location.Model {
	// Build via whatever constructor Task 6 exposed on location.Model.
	// If Task 6 left the struct fields unexported with no builder, add a small
	// test-only constructor in the location package rather than reflecting.
	return location.NewModelForTest(findTargetId, findWorldId, ch, _map.Id(100000000), uuid.Nil, state)
}

// mapBodyLen returns the byte length of a map-shape body without x/y:
// mode(1) + ascii string(2 + len) + findMode(1) + mapId(4).
func mapBodyLen(name string) int {
	return 1 + 2 + len(name) + 1 + 4
}
```

Two notes on the above:

- `locationFixture` calls `location.NewModelForTest`. Task 6 does not create it, so **add it in this task**, in `services/atlas-channel/atlas.com/channel/maps/location/requests.go`, next to `SetBaseURLForTest` which exists for the same reason:

  ```go
  // NewModelForTest constructs a Model directly. Only call from a test;
  // production code builds one through Get.
  func NewModelForTest(characterId uint32, w world.Id, ch channel.Id, m _map.Id, instance uuid.UUID, state characterconst.PresenceState) Model {
  	return Model{characterId: characterId, worldId: w, channelId: ch, mapId: m, instance: instance, state: state}
  }
  ```

- Every builder name used above is real and needs no substitution: `character.NewModelBuilder()` (`character/builder.go:61`), `SetId` (`:109`), `SetWorldId` (`:111`), `SetName` (`:112`), `SetGm(int)` (`:137`), `SetX(int16)` (`:138`), `SetY(int16)` (`:139`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestFind_ -v`
Expected: FAIL — build error: `undefined: findCharacterByNameFunc`, `findLocalSessionFunc`, `findCharacterLocationFunc`, and `location.NewModelForTest`.

- [ ] **Step 3: Add the seams and the outcome type**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper.go`, add above `CharacterChatWhisperHandleFunc`:

```go
// The /find path's three lookups, exposed as package-level seams so the
// decision table below is unit-testable. This mirrors
// checkNameChangeValidityFunc in cash_shop_check_name_change.go; tests swap
// them and restore via t.Cleanup.
var findCharacterByNameFunc = func(l logrus.FieldLogger, ctx context.Context, name string) (character.Model, error) {
	return character.NewProcessor(l, ctx).GetByName(name)
}

var findLocalSessionFunc = func(l logrus.FieldLogger, ctx context.Context, ch channel.Model, characterId uint32) (session.Model, error) {
	return session.NewProcessor(l, ctx).GetByCharacterId(ch)(characterId)
}

// findCharacterLocationFunc returns location.Model rather than field.Model
// because field.Model has nowhere to carry the presence state. Note this is
// location.Get, NOT location.ResolveMapId: ResolveMapId collapses every
// failure to map id 0, which is exactly how a transport failure renders as a
// real location today.
var findCharacterLocationFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (location.Model, error) {
	return location.Get(l, ctx, characterId)
}

// findOutcomeKind discriminates the four wire shapes /find can answer with.
type findOutcomeKind int

const (
	findOutcomeError findOutcomeKind = iota
	findOutcomeCashShop
	findOutcomeMap
	findOutcomeChannel
)

// findOutcome is the result of the decision table. branch names the rule that
// matched, so the four wire-identical error branches stay separable in logs
// (FR-13).
type findOutcome struct {
	kind      findOutcomeKind
	branch    string
	name      string
	mapId     uint32
	x         int32
	y         int32
	channelId uint32
	err       error // set only for branch "lookup-failed"
}
```

Add `"atlas-channel/maps/location"` (already imported), `channel "github.com/Chronicle20/atlas/libs/atlas-constants/channel"` and `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"` to the import block as needed. Use the actual type of `s.Field().Channel()` for the `ch` parameter — read `session/processor.go:234` for `GetByCharacterId`'s real signature and match it rather than assuming.

- [ ] **Step 4: Write `findDecision`**

Add to the same file:

```go
// findDecision is the pure core of /find: PRD §4.1's rule ordering with the
// FR-4/FR-6/FR-7 sources corrected to read atlas-maps's presence state.
// Evaluated in order; first match wins. No writer, no session mutation, no
// packet types — every FR-1..FR-7 test lands here.
func findDecision(l logrus.FieldLogger, ctx context.Context, s session.Model, targetName string) findOutcome {
	tc, err := findCharacterByNameFunc(l, ctx, targetName)
	if err != nil {
		// FR-1. Echo the REQUESTED name: there is no resolved one.
		return findOutcome{kind: findOutcomeError, branch: "unresolved", name: targetName}
	}

	// FR-2. Cross-world targets are not findable. Wire-identical to FR-1 by
	// design (PRD §8) but a distinct branch in the logs.
	if tc.WorldId() != s.Field().WorldId() {
		return findOutcome{kind: findOutcomeError, branch: "cross-world", name: targetName}
	}

	// FR-3. GM visibility is a boolean predicate on level > 0, not a tier
	// comparison: any GM sees any GM, no GM is hidden from another GM.
	if tc.Gm() && !s.Gm() {
		return findOutcome{kind: findOutcomeError, branch: "gm-concealed", name: targetName}
	}

	// The local session is consulted before the location row. A live local
	// session is by construction authoritative about both liveness and
	// cash-scene, and costs nothing.
	if ls, lerr := findLocalSessionFunc(l, ctx, s.Field().Channel(), tc.Id()); lerr == nil {
		if ls.CashScene() != session.CashSceneNone {
			// FR-4a. CashSceneMts folds in here: the ITC renders inside the
			// cash-shop CStage and the client has no separate shape for it.
			return findOutcome{kind: findOutcomeCashShop, branch: "cash-shop-local", name: tc.Name()}
		}
		// FR-5. Same channel, on a map. The map id comes from the location
		// row; if that lookup fails we answer the error shape rather than
		// map 0 — "not findable" for a demonstrably online player is wrong but
		// recoverable, whereas map 0 is confidently wrong.
		loc, lerr2 := findCharacterLocationFunc(l, ctx, tc.Id())
		if lerr2 != nil {
			return locationErrorOutcome(targetName, lerr2)
		}
		return findOutcome{
			kind:   findOutcomeMap,
			branch: "map-local",
			name:   tc.Name(),
			mapId:  uint32(loc.MapId()),
			x:      int32(tc.X()),
			y:      int32(tc.Y()),
		}
	}

	loc, err := findCharacterLocationFunc(l, ctx, tc.Id())
	if err != nil {
		return locationErrorOutcome(targetName, err)
	}

	switch loc.State() {
	case characterconst.PresenceStateInCashShop:
		// FR-4b.
		return findOutcome{kind: findOutcomeCashShop, branch: "cash-shop-remote", name: tc.Name()}
	case characterconst.PresenceStateInField:
		// FR-6. channel.Id is already the 0-based internal value and the codec
		// writes it unadjusted; the client adds one for display.
		return findOutcome{
			kind:      findOutcomeChannel,
			branch:    "channel-remote",
			name:      tc.Name(),
			channelId: uint32(loc.ChannelId()),
		}
	default:
		// FR-7a. OFFLINE, and anything unrecognised, which ParsePresenceState
		// has already narrowed to OFFLINE.
		return findOutcome{kind: findOutcomeError, branch: "offline", name: targetName}
	}
}

// locationErrorOutcome separates "no row at all" (a character who has never
// logged in) from an infrastructure failure. Both answer the same wire shape;
// only the log line and level differ.
func locationErrorOutcome(targetName string, err error) findOutcome {
	if errors.Is(err, location.ErrNotFound) {
		return findOutcome{kind: findOutcomeError, branch: "never-logged-in", name: targetName}
	}
	return findOutcome{kind: findOutcomeError, branch: "lookup-failed", name: targetName, err: err}
}
```

Add `"errors"` to the import block.

- [ ] **Step 5: Rewrite the adapter**

Replace the body of `produceFindResultBody` with a thin adapter — pick the mode byte, call `findDecision`, project, log, announce. Delete both placeholder comments and the `cs := false` block entirely; nothing deferred survives this step:

```go
func produceFindResultBody(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
			return func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
				return func(s session.Model) error {
					var resultMode byte
					if mode == chat.WhisperModeBuddyWindowFind {
						resultMode = 0x48
					} else {
						resultMode = 0x09
					}

					o := findDecision(l, ctx, s, targetName)

					// FR-13. One line per /find, carrying the branch, so the
					// four wire-identical error branches stay separable.
					entry := l.WithFields(logrus.Fields{
						"requester_id": s.CharacterId(),
						"target_name":  targetName,
						"arm":          resultMode,
						"branch":       o.branch,
					})
					if o.err != nil {
						entry.WithError(o.err).Error("/find location lookup failed")
					} else {
						entry.Debug("/find resolved")
					}

					af := session.Announce(l)(ctx)(wp)(fieldcb.WhisperWriter)
					switch o.kind {
					case findOutcomeCashShop:
						return af(fieldcb.NewWhisperFindResultCashShop(resultMode, o.name).Encode)(s)
					case findOutcomeChannel:
						return af(fieldcb.NewWhisperFindResultChannel(resultMode, o.name, o.channelId).Encode)(s)
					case findOutcomeMap:
						// The 0x09 arm carries x/y; 0x48 does not. Confirmed
						// against the client on every version — the read is
						// gated on the mode being odd AND findMode == 1.
						if resultMode == 0x09 {
							return af(fieldcb.NewWhisperFindResultMapWithXY(resultMode, o.name, o.mapId, o.x, o.y).Encode)(s)
						}
						return af(fieldcb.NewWhisperFindResultMap(resultMode, o.name, o.mapId).Encode)(s)
					default:
						return af(fieldcb.NewWhisperFindResultError(resultMode, o.name).Encode)(s)
					}
				}
			}
		}
	}
}
```

Drop now-unused imports (`"atlas-channel/maps/location"` stays — `location.ErrNotFound` and `location.Get` are both used).

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/ -v`
Expected: PASS across the whole handler package.

- [ ] **Step 7: Run the module test suite**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper_test.go \
        services/atlas-channel/atlas.com/channel/maps/location/requests.go
git commit -m "fix(atlas-channel): report the target's real location for /find"
```

---

## Task 9: Promote the `gms_v92` clientbound WHISPER matrix cell

**Read [docs/packets/audits/VERIFYING_A_PACKET.md](../../packets/audits/VERIFYING_A_PACKET.md) §5–§8 and §10 before starting.**

`docs/packets/audits/status.json` has op `field/clientbound/FieldWhisperError` × `gms_v92` at `incomplete` — *"tier-1 without fixture; verdict ❌"*. The ❌ comes from `docs/packets/ida-exports/gms_v92.json`, whose eight `CField::OnWhisper#*` arm records are `"unresolved": true` with the note *"requires a per-arm decompile pass against the v92 IDB"*.

**That decompile pass was run during `/plan-task`.** `CField::OnWhisper` @`0x53e2a0` in `GMS_v92_1_DEVM.exe` has the same shape as v83/v84/v87/v95:

```c
case 9:
case 72:
  DecodeStr(name); v24 = Decode1();  // findMode
  v25 = Decode4();                   // payload
  if ( (v4 & 1) == 0 )               // EVEN mode -> 0x48
  {
    if ( (v4 & 0x40) != 0 ) { switch (v24) { case 2: case 3: case 1: } }
    goto LABEL_121;                  // no further decode -- no x/y
  }
  switch ( v24 )                     // ODD mode -> 0x09
  {
    case 1:
      v36 = Decode4();               // x
      v67 = Decode4();               // y
```

plus `case 10/138` SendResult, `case 18` Receive, `case 34` Error, `case 146` Weather — the same eight arms every other version has.

**No wire change.** This task adds arm records, a fixture, and evidence. It does not touch any codec.

**Do not re-run `packet-audit export`.** It is not idempotent (§10) and would drift ~150 unrelated function keys. Hand-author the eight arm entries, mirroring `gms_v95.json`'s shape exactly.

### Files

- `docs/packets/ida-exports/gms_v92.json` — replace the eight `CField::OnWhisper#*` `unresolved` stubs
- `libs/atlas-packet/field/clientbound/whisper_test.go` — add `TestWhisperByteOutputV92` and its markers
- `docs/packets/evidence/gms_v92/field.clientbound.FieldWhisperError.yaml` — **new file**; written by `packet-audit evidence pin`
- `docs/packets/audits/gms_v92/FieldWhisper*.json` and `.md` — regenerated (9 writers × 2 files)
- `docs/packets/audits/STATUS.md` and `docs/packets/audits/status.json` — regenerated
- `docs/packets/evidence/gms_v95/field.clientbound.FieldWhisperError.yaml` — read-only; the evidence-record shape to expect
- `libs/atlas-packet/field/clientbound/whisper_test.go:228` — read-only; `TestWhisperByteOutputV61` is the byte-test to copy

Module root for `go build`/`go test`: `libs/atlas-packet`. The `packet-audit` commands run from the repo root.

**Interfaces:**
- Consumes: nothing from earlier tasks. This task is independent of Tasks 1–8 and may be done in any order relative to them.
- Produces: nothing later tasks depend on.

- [ ] **Step 1: Replace the eight v92 arm records**

In `docs/packets/ida-exports/gms_v92.json`, each of `CField::OnWhisper#SendResult`, `#Receive`, `#FindResultMap`, `#FindResultCashShop`, `#FindResultChannel`, `#FindResultError`, `#Error`, `#Weather` currently reads:

```json
{"address": "", "direction": "clientbound", "notes": "dispatcher-family arm not harvested for gms_v92 ...", "unresolved": true, "calls": [{"op": "Unresolved", "comment": "..."}]}
```

Replace each with the v95 record for the same arm, with `address` set to `"0x53e2a0"` and the note re-worded for v92. Copy the `calls` arrays verbatim from `gms_v95.json` — the read orders are identical, which is the finding. For example, `#FindResultMap` becomes:

```json
{
  "address": "0x53e2a0",
  "direction": "clientbound",
  "note": "mode=0x09/0x48 sub=1 (FIND on map). x/y are Decode4 (int32) only when the mode is ODD (0x09): CField::OnWhisper @0x53e2a0 gates the even arm on `if ((v4 & 1) == 0)` and returns via LABEL_121 without a further decode; the odd arm's `case 1:` reads v36=Decode4(x), v67=Decode4(y). Wire version-invariant v83=v92=v95. Derived task-238.",
  "calls": [
    {"op": "Decode1", "comment": "mode"},
    {"op": "DecodeStr", "comment": "target"},
    {"op": "Decode1", "comment": "findMode (=1)"},
    {"op": "Decode4", "comment": "mapId"},
    {"op": "Decode4", "comment": "x (mode 0x09 only)", "guard": "mode == 0x09"},
    {"op": "Decode4", "comment": "y (mode 0x09 only)", "guard": "mode == 0x09"}
  ]
}
```

Remove the `unresolved` and `notes` keys from every one of the eight. Do not touch any other entry in the file.

- [ ] **Step 2: Verify the JSON still parses**

Run: `python3 -c "import json; d=json.load(open('docs/packets/ida-exports/gms_v92.json')); print(sum(1 for k in d['functions'] if 'OnWhisper' in k), 'whisper entries'); print(sum(1 for k,v in d['functions'].items() if 'OnWhisper' in k and v.get('unresolved')), 'still unresolved')"`
Expected: `9 whisper entries` (eight arms plus the raw `CField::OnWhisper`) and `0 still unresolved`.

- [ ] **Step 3: Write the failing byte test**

Append to `libs/atlas-packet/field/clientbound/whisper_test.go`, copying the structure of `TestWhisperByteOutputV61` (line 228). `pt.CreateContext("GMS", 92, 1)` is valid — `GMS v92` is `pt.Variants[11]`.

```go
// packet-audit:verify packet=field/clientbound/FieldWhisperSendResult version=gms_v92 ida=0x53e2a0
// packet-audit:verify packet=field/clientbound/FieldWhisperReceive version=gms_v92 ida=0x53e2a0
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultCashShop version=gms_v92 ida=0x53e2a0
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultMap version=gms_v92 ida=0x53e2a0
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultChannel version=gms_v92 ida=0x53e2a0
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultError version=gms_v92 ida=0x53e2a0
// packet-audit:verify packet=field/clientbound/FieldWhisperError version=gms_v92 ida=0x53e2a0
// packet-audit:verify packet=field/clientbound/FieldWhisperWeather version=gms_v92 ida=0x53e2a0
//
// TestWhisperByteOutputV92 pins every gms_v92 WHISPER (op 0x96 = 150)
// clientbound sub-mode. IDA: CField::OnWhisper @0x53e2a0
// (GMS_v92_1_DEVM.exe) switches on Decode1(mode):
//
//	case 10/138 SendResult : DecodeStr(target) + Decode1(success)
//	case 18     Receive    : DecodeStr(from) + Decode1(ch) + Decode1(gm) + DecodeStr(msg)
//	case 34     Error      : DecodeStr(target) + Decode1(whispersEnabled)
//	case 146    Weather    : DecodeStr(from) + Decode1(flag) + DecodeStr(msg)
//	case 9/72   FindResult : DecodeStr(target) + Decode1(findMode) + Decode4(value)
//
// The find arm splits on mode parity, NOT on the mode value: the even arm
// (0x48) is guarded by `if ((v4 & 1) == 0)` and exits via LABEL_121 with no
// further decode, while the odd arm (0x09) reaches `case 1:` and reads
// v36=Decode4(x), v67=Decode4(y). x/y are therefore read only for mode 0x09 at
// findMode 1 — the same rule as v83/v84/v87/v95, and byte-identical to the v83
// golden (version-invariant layout).
func TestWhisperByteOutputV92(t *testing.T) {
	ctx := pt.CreateContext("GMS", 92, 1)

	send := NewWhisperSendResult(0x0A, "TargetPlayer", true)
	if got := pt.Encode(t, ctx, send.Encode, nil); !bytes.Equal(got, append(append([]byte{0x0A}, wstrV79("TargetPlayer")...), 0x01)) {
		t.Errorf("v92 sendResult: got %v", got)
	}

	recv := NewWhisperReceive(0x12, "SenderPlayer", 3, false, "secret whisper")
	var rw []byte
	rw = append(rw, 0x12)
	rw = append(rw, wstrV79("SenderPlayer")...)
	rw = append(rw, 0x03, 0x00)
	rw = append(rw, wstrV79("secret whisper")...)
	if got := pt.Encode(t, ctx, recv.Encode, nil); !bytes.Equal(got, rw) {
		t.Errorf("v92 receive: got %v want %v", got, rw)
	}

	cash := NewWhisperFindResultCashShop(0x09, "ShopPlayer")
	if got := pt.Encode(t, ctx, cash.Encode, nil); !bytes.Equal(got, append(append([]byte{0x09}, wstrV79("ShopPlayer")...), 0x02, 0xFF, 0xFF, 0xFF, 0xFF)) {
		t.Errorf("v92 cashShop: got %v", got)
	}

	mp := NewWhisperFindResultMap(0x09, "MapPlayer", 100000000)
	if got := pt.Encode(t, ctx, mp.Encode, nil); !bytes.Equal(got, append(append([]byte{0x09}, wstrV79("MapPlayer")...), 0x01, 0x00, 0xE1, 0xF5, 0x05)) {
		t.Errorf("v92 map: got %v", got)
	}

	// The odd arm carries x/y; @0x53e2a0 case 1 reads v36=Decode4, v67=Decode4.
	mpxy := NewWhisperFindResultMapWithXY(0x09, "MapPlayer", 100000000, 250, -75)
	var mxy []byte
	mxy = append(mxy, 0x09)
	mxy = append(mxy, wstrV79("MapPlayer")...)
	mxy = append(mxy, 0x01, 0x00, 0xE1, 0xF5, 0x05)
	mxy = append(mxy, 0xFA, 0x00, 0x00, 0x00)
	mxy = append(mxy, 0xB5, 0xFF, 0xFF, 0xFF)
	if got := pt.Encode(t, ctx, mpxy.Encode, nil); !bytes.Equal(got, mxy) {
		t.Errorf("v92 mapWithXY: got %v want %v", got, mxy)
	}

	// The even arm does NOT: `if ((v4 & 1) == 0)` exits via LABEL_121.
	mp48 := NewWhisperFindResultMap(0x48, "MapPlayer", 100000000)
	if got := pt.Encode(t, ctx, mp48.Encode, nil); !bytes.Equal(got, append(append([]byte{0x48}, wstrV79("MapPlayer")...), 0x01, 0x00, 0xE1, 0xF5, 0x05)) {
		t.Errorf("v92 map 0x48: got %v", got)
	}

	ch := NewWhisperFindResultChannel(0x09, "ChannelPlayer", 5)
	if got := pt.Encode(t, ctx, ch.Encode, nil); !bytes.Equal(got, append(append([]byte{0x09}, wstrV79("ChannelPlayer")...), 0x03, 0x05, 0x00, 0x00, 0x00)) {
		t.Errorf("v92 channel: got %v", got)
	}

	fe := NewWhisperFindResultError(0x09, "MissingPlayer")
	if got := pt.Encode(t, ctx, fe.Encode, nil); !bytes.Equal(got, append(append([]byte{0x09}, wstrV79("MissingPlayer")...), 0x00, 0x00, 0x00, 0x00, 0x00)) {
		t.Errorf("v92 findError: got %v", got)
	}

	er := WhisperError{mode: 0x22, targetName: "BlockedPlayer", whispersEnabled: false}
	if got := pt.Encode(t, ctx, er.Encode, nil); !bytes.Equal(got, append(append([]byte{0x22}, wstrV79("BlockedPlayer")...), 0x00)) {
		t.Errorf("v92 error: got %v", got)
	}

	wx := NewWhisperWeather(0x92, "GMPlayer", "Weather alert!")
	var ww []byte
	ww = append(ww, 0x92)
	ww = append(ww, wstrV79("GMPlayer")...)
	ww = append(ww, 0x01)
	ww = append(ww, wstrV79("Weather alert!")...)
	if got := pt.Encode(t, ctx, wx.Encode, nil); !bytes.Equal(got, ww) {
		t.Errorf("v92 weather: got %v want %v", got, ww)
	}
}
```

The x/y bytes above are little-endian int32: `250` = `FA 00 00 00`, `-75` = `B5 FF FF FF`. If the assertion fails, hand-recompute rather than pasting whatever the failure prints.

- [ ] **Step 4: Run the test**

Run: `cd libs/atlas-packet && go test ./field/clientbound/ -run TestWhisperByteOutputV92 -v`
Expected: PASS. The codec is unchanged and v92 shares the v83 layout, so this pins existing behaviour rather than driving a change. If it fails, the byte expectation is wrong — fix the expectation, never the codec.

- [ ] **Step 5: Regenerate the v92 whisper audit reports**

From the repo root, generate to a temp directory and copy only the whisper reports in (§9 — do not copy the whole tree):

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_v92.json \
  -ida-source docs/packets/ida-exports/gms_v92.json \
  -output /tmp/rpt-v92
```

If `template_v92.json` is not the actual filename, list `services/atlas-configurations/seed-data/templates/` and use the real v92 template.

Then copy the nine regenerated whisper reports over the committed ones:

```bash
for f in FieldWhisperError FieldWhisperSendResult FieldWhisperReceive \
         FieldWhisperFindResultMap FieldWhisperFindResultCashShop \
         FieldWhisperFindResultChannel FieldWhisperFindResultError \
         FieldWhisperWeather; do
  cp /tmp/rpt-v92/gms_v92/$f.json /tmp/rpt-v92/gms_v92/$f.md docs/packets/audits/gms_v92/
done
```

Confirm the verdict moved: `python3 -c "import json; d=json.load(open('docs/packets/audits/gms_v92/FieldWhisperError.json')); print('Verdict', d['Verdict'], '| Address', d['Address']); [print(r['Verdict'], r['Note']) for r in d['Rows']]"`
Expected: no row carries *"IDA read-order unresolved"*, and `Address` is now `0x53e2a0`.

- [ ] **Step 6: Pin the evidence record**

```bash
go run ./tools/packet-audit evidence pin \
  --packet field/clientbound/FieldWhisperError \
  --version gms_v92 \
  --ida "CField::OnWhisper#Error" \
  --category TIER1-FIXTURE
```

Then open `docs/packets/evidence/gms_v92/field.clientbound.FieldWhisperError.yaml` and append the `verifies:` field by hand:

```yaml
verifies:
  - libs/atlas-packet/field/clientbound/whisper_test.go#TestWhisperByteOutputV92
```

- [ ] **Step 7: Regenerate the matrix and confirm promotion**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

`matrix --check` must exit 0 — it is a hard, blocking CI gate. Then confirm the target cell:

```bash
python3 -c "
import json
d=json.load(open('docs/packets/audits/status.json'))
for r in d['rows']:
    if r.get('packet')=='field/clientbound/FieldWhisperError' and r.get('kind')=='op':
        print(r['cells']['gms_v92'])
"
```
Expected: `state` is `verified`.

Editing `gms_v92.json` changes its entry in `status.json`'s `exportHashes`, so re-run the two commands above and re-read the output rather than assuming only the whisper row moved. If `matrix --check` reports any 🟥 conflict, orphan, dangling, stale or drift finding — including on an unrelated v92 cell — stop and report it; do not paper over it.

The serverbound `chat/serverbound/ChatWhisper` × `gms_v92` cell (*"no audit report"*) is a different op and stays `incomplete`. That is expected, not a regression.

- [ ] **Step 8: Run the packet module tests**

Run: `cd libs/atlas-packet && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 9: Commit test, evidence, export and matrix together**

```bash
git add docs/packets/ida-exports/gms_v92.json \
        libs/atlas-packet/field/clientbound/whisper_test.go \
        docs/packets/evidence/gms_v92/field.clientbound.FieldWhisperError.yaml \
        docs/packets/audits/gms_v92/ \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "test(atlas-packet): verify clientbound WHISPER on gms_v92"
```

---

## Final gate (controller, not the implementer)

- [ ] Flagless `tools/verify.sh` exits 0. `--quick` / `--no-docker` do not count — they skip the bake and `-race`.
- [ ] Code review runs before the PR is opened.
- [ ] `atlas-cashshop`, `atlas-mts` and `atlas-character` have no diff on this branch.
- [ ] `git diff --stat` shows no change to any `Encode`/`Decode` body in `libs/atlas-packet`.
