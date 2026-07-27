# Resurrection Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **`<worktree-root>`** in every command below means the task worktree root: `.worktrees/task-111-resurrection-skill` under the Atlas repo. `cd` there first; all paths are relative to it.

**Goal:** Add an active-skill handler in atlas-channel that revives dead characters in place at full HP for Bishop Resurrection (`2321006`), GM Resurrection (`9001005`), and SuperGM Resurrection (`9101005`).

**Architecture:** A new `skill/handler/resurrection/` package registers a handler for the three skill IDs, dispatched from the generic `UseSkill` path (which has already validated ownership/level, loaded the WZ effect, and consumed MP + applied cooldown). Per dead recipient the handler emits a new absolute `SET_HP` channel command (atlas-character clamps `0xFFFF` to effective MaxHP) then the existing task-093 `portal.WarpToPosition` chase-warp to the recipient's death coordinates — the death-stance client fires its native `OnRevive`, standing the avatar up in place. Recipient selection reuses the shared party selector, generalized with a dead/alive predicate, plus a new party-agnostic map-wide dead-player selector for the GM variants. No new REST endpoints, Kafka topics, tables, or migrations.

**Tech Stack:** Go 1.x, atlas-channel microservice, Kafka command/event messaging, `libs/atlas-constants/skill`, `libs/atlas-packet`, the project's immutable-model + Builder + function-seam test idioms.

---

## File Structure

**`libs/atlas-constants/skill/`**
- `constants.go` (modify) — add `GmResurrectionId = Id(9001005)` const, `GmResurrection` `Skill` var, and the `GmResurrectionId: GmResurrection` entry in the `Skills` registry map.
- `resurrection_test.go` (create) — assert the three resurrection IDs and their registry entries.

**`services/atlas-channel/atlas.com/channel/kafka/message/character/`**
- `kafka.go` (modify) — add `CommandSetHP = "SET_HP"` const and `SetHPCommandBody` struct (JSON-compatible with atlas-character's `SetHPBody`).

**`services/atlas-channel/atlas.com/channel/character/`**
- `producer.go` (modify) — add `SetHPCommandProvider`.
- `processor.go` (modify) — add `SetHP` to the `Processor` interface and `ProcessorImpl`.
- `mock/processor.go` (modify) — add `SetHP` mock.
- `builder.go` (modify) — add `SetX`/`SetY` to `modelBuilder` (model fields already exist; needed for coordinate-based selector tests).

**`services/atlas-channel/atlas.com/channel/skill/handler/`**
- `recipients.go` (modify) — generalize `selectPartyMembers` with a `wantDead` predicate; add `SelectDeadInRangePartyMembers`, `SelectDeadInRangeMapPlayers`, and the `loadMapPlayerFunc` seam.
- `recipients_test.go` (modify) — add dead-selector tests and a living-only regression test.

**`services/atlas-channel/atlas.com/channel/skill/handler/resurrection/`** (new package)
- `recipients.go` (create) — `selectByVariant` dispatch + `selectDeadParty`/`selectDeadMap` seams.
- `recipients_test.go` (create) — variant→selector dispatch tests.
- `resurrection.go` (create) — `init()` registration + `Apply` handler + seams.
- `resurrection_test.go` (create) — registration, ordering, no-op, per-recipient isolation, caster-load-failure tests.

**`services/atlas-channel/atlas.com/channel/skill/handler/registrations/`**
- `registrations.go` (modify) — blank import of the new resurrection package.

---

## Task 1: Add the GM Resurrection constant

**Files:**
- Modify: `libs/atlas-constants/skill/constants.go` (const block ~line 3242, var block ~line 1448, `Skills` map ~line 2691)
- Test: `libs/atlas-constants/skill/resurrection_test.go`

Context: `BishopResurrectionId` (`2321006`) and `SuperGmResurrectionId` (`9101005`) already exist; `GmResurrectionId` (`9001005`) does **not** (PRD FR-1 was wrong — confirmed by grep). The GM IDs run `9001000`–`9001004`; `9001005` is the next slot. `SuperGmResurrection` is a plain `Skill{ id: ... }` (no `buff`), so `GmResurrection` mirrors it.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-constants/skill/resurrection_test.go`:

```go
package skill

import "testing"

func TestResurrectionIds(t *testing.T) {
	if GmResurrectionId != Id(9001005) {
		t.Fatalf("GmResurrectionId = %d, want 9001005", GmResurrectionId)
	}
	if BishopResurrectionId != Id(2321006) {
		t.Fatalf("BishopResurrectionId = %d, want 2321006", BishopResurrectionId)
	}
	if SuperGmResurrectionId != Id(9101005) {
		t.Fatalf("SuperGmResurrectionId = %d, want 9101005", SuperGmResurrectionId)
	}
}

func TestResurrectionRegistryEntries(t *testing.T) {
	for _, id := range []Id{BishopResurrectionId, GmResurrectionId, SuperGmResurrectionId} {
		s, ok := Skills[id]
		if !ok {
			t.Fatalf("Skills[%d] missing", id)
		}
		if s.Id() != id {
			t.Fatalf("Skills[%d].Id() = %d, want %d", id, s.Id(), id)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd <worktree-root>/libs/atlas-constants && go test ./skill/ -run TestResurrection -v`
Expected: FAIL — `undefined: GmResurrectionId`.

- [ ] **Step 3: Add the constant, var, and registry entry**

In `constants.go`, in the `const (...)` id block, immediately after the `GmHideId = Id(9001004)` line (~3242), add:

```go
	GmResurrectionId                            = Id(9001005)
```

In the var block, immediately after the `GmHide` var (~line 1453), add:

```go
var GmResurrection = Skill{
	id: GmResurrectionId,
}
```

In the `Skills = map[Id]Skill{...}` map, immediately after the `GmHideId: GmHide,` entry (~line 2693), add:

```go
	GmResurrectionId:                            GmResurrection,
```

(Alignment whitespace need not be exact — `gofmt` normalizes it in Step 5.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd <worktree-root>/libs/atlas-constants && gofmt -w skill/constants.go && go test ./skill/ -run TestResurrection -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd <worktree-root>
git add libs/atlas-constants/skill/constants.go libs/atlas-constants/skill/resurrection_test.go
git commit -m "feat(skill-constants): add GmResurrectionId (9001005)"
git branch --show-current  # must be task-111-resurrection-skill
```

---

## Task 2: Add the channel-side absolute SET_HP command type

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go:16,55-58`

Context: atlas-channel only has the relative `CHANGE_HP` command. atlas-character already consumes `SET_HP` with `SetHPBody{ ChannelId channel.Id; Amount uint16 }` (`services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:170-173`) and clamps `Amount` to effective MaxHP (`services/atlas-character/atlas.com/character/character/processor.go:1166-1185`). We add the producing-side mirror. The channel `Command[E]` wrapper has no `TransactionId` field; atlas-character's wrapper does and defaults it to the zero UUID on deserialize — identical to how `CHANGE_HP` already works, so no transaction plumbing is needed.

This task is a pure type addition exercised by Task 3's test, so it has no standalone test.

- [ ] **Step 1: Add the command constant**

In `kafka/message/character/kafka.go`, in the `const (...)` block, after `CommandChangeMP = "CHANGE_MP"` (line 17), add:

```go
	CommandSetHP               = "SET_HP"
```

- [ ] **Step 2: Add the command body struct**

After the `ChangeMPCommandBody` struct (line 63), add:

```go
type SetHPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    uint16     `json:"amount"`
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go build ./kafka/...`
Expected: clean build, no output.

- [ ] **Step 4: Commit**

```bash
cd <worktree-root>
git add services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go
git commit -m "feat(channel): add SET_HP command type"
git branch --show-current  # must be task-111-resurrection-skill
```

---

## Task 3: Add the SetHP command producer

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/producer.go:57-69`
- Test: `services/atlas-channel/atlas.com/channel/character/producer_test.go` (create)

Context: Mirror `ChangeHPCommandProvider` exactly, but with `uint16` amount, `CommandSetHP` type, and `SetHPCommandBody`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/character/producer_test.go`:

```go
package character

import (
	"encoding/json"
	"testing"

	messagechar "atlas-channel/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestSetHPCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(3), 100000000).Build()
	msgs, err := SetHPCommandProvider(f, 4242, 0xFFFF)()
	if err != nil {
		t.Fatalf("provider err: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}

	var cmd messagechar.Command[messagechar.SetHPCommandBody]
	if uErr := json.Unmarshal(msgs[0].Value, &cmd); uErr != nil {
		t.Fatalf("unmarshal: %v", uErr)
	}
	if cmd.Type != messagechar.CommandSetHP {
		t.Fatalf("Type = %q, want %q", cmd.Type, messagechar.CommandSetHP)
	}
	if cmd.CharacterId != 4242 {
		t.Fatalf("CharacterId = %d, want 4242", cmd.CharacterId)
	}
	if cmd.Body.ChannelId != channel.Id(3) {
		t.Fatalf("Body.ChannelId = %d, want 3", cmd.Body.ChannelId)
	}
	if cmd.Body.Amount != 0xFFFF {
		t.Fatalf("Body.Amount = %d, want 65535", cmd.Body.Amount)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go test ./character/ -run TestSetHPCommandProvider -v`
Expected: FAIL — `undefined: SetHPCommandProvider`.

- [ ] **Step 3: Add the producer**

In `character/producer.go`, after `ChangeHPCommandProvider` (line 69), add:

```go
func SetHPCommandProvider(f field.Model, characterId uint32, amount uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character.Command[character.SetHPCommandBody]{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		Type:        character.CommandSetHP,
		Body: character.SetHPCommandBody{
			ChannelId: f.ChannelId(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go test ./character/ -run TestSetHPCommandProvider -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd <worktree-root>
git add services/atlas-channel/atlas.com/channel/character/producer.go services/atlas-channel/atlas.com/channel/character/producer_test.go
git commit -m "feat(channel): add SetHPCommandProvider"
git branch --show-current  # must be task-111-resurrection-skill
```

---

## Task 4: Add SetHP to the character Processor + mock

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/processor.go:41` (interface) and `:271-273` (impl, after `ChangeHP`)
- Modify: `services/atlas-channel/atlas.com/channel/character/mock/processor.go:119` (after `ChangeHP` mock)

Context: Mirror `ChangeHP`. The interface and the mock must stay in lockstep or the package won't compile. The mock has no recording need here (handler tests stub the `setHP` seam, not the processor), so a no-op mock suffices and keeps `mock.MockProcessor` satisfying the interface.

- [ ] **Step 1: Add to the interface**

In `character/processor.go`, in the `Processor` interface, immediately after the `ChangeHP(f field.Model, characterId uint32, amount int16) error` line (line 41), add:

```go
	SetHP(f field.Model, characterId uint32, amount uint16) error
```

- [ ] **Step 2: Add the impl method**

After the `ChangeHP` impl method (lines 271-273), add:

```go
func (p *ProcessorImpl) SetHP(f field.Model, characterId uint32, amount uint16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(SetHPCommandProvider(f, characterId, amount))
}
```

- [ ] **Step 3: Add the mock**

In `character/mock/processor.go`, after the `ChangeHP` mock (line 119), add:

```go
func (m *MockProcessor) SetHP(_ field.Model, _ uint32, _ uint16) error {
	return nil
}
```

- [ ] **Step 4: Verify it compiles and existing tests pass**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go build ./character/... && go test ./character/ -run TestSetHP -v`
Expected: clean build; `TestSetHPCommandProvider` (from Task 3) still PASSes. (No new test here — `SetHP` is a one-line passthrough whose payload is already covered by Task 3's provider test and whose interface conformance is enforced by the mock compiling.)

- [ ] **Step 5: Commit**

```bash
cd <worktree-root>
git add services/atlas-channel/atlas.com/channel/character/processor.go services/atlas-channel/atlas.com/channel/character/mock/processor.go
git commit -m "feat(channel): add SetHP to character Processor and mock"
git branch --show-current  # must be task-111-resurrection-skill
```

---

## Task 5a: Add SetX/SetY to the character model builder

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/builder.go` (setter block ~line 109-144)

Context: `Model` already has `x`/`y` fields and `X()`/`Y()` getters, and `Build()` already copies `b.x`/`b.y`, but the builder exposes **no** `SetX`/`SetY`. The dead map-player selector (Task 5b) filters by coordinate, and its tests need characters at specific positions. Adding these setters is the sanctioned Builder-pattern extension (no `*_testhelpers.go`).

- [ ] **Step 1: Add the setters**

In `character/builder.go`, alongside the other one-line setters (e.g. after `SetLevel`, line 117), add:

```go
func (b *modelBuilder) SetX(v int16) *modelBuilder { b.x = v; return b }
func (b *modelBuilder) SetY(v int16) *modelBuilder { b.y = v; return b }
```

- [ ] **Step 2: Verify it compiles**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go build ./character/...`
Expected: clean build.

- [ ] **Step 3: Commit**

```bash
cd <worktree-root>
git add services/atlas-channel/atlas.com/channel/character/builder.go
git commit -m "feat(channel): add SetX/SetY to character model builder"
git branch --show-current  # must be task-111-resurrection-skill
```

---

## Task 5b: Generalize the shared selector and add the two dead-target selectors

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/recipients.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/recipients_test.go`

Context: `selectPartyMembers` hard-codes `if mc.Hp() == 0 { continue }` (living-only, line 174). Generalize it with a `wantDead bool` parameter (existing callers pass `false`, preserving behavior) and add:
- `SelectDeadInRangePartyMembers` — Bishop variant (party + LT/RB + dead-only).
- `SelectDeadInRangeMapPlayers` — GM variant (all in-map players, party-agnostic, LT/RB + dead-only, caster excluded), reusing the existing `inMapCharacterIdsFunc` seam (its own mutex already guards the concurrent `ForSessionsInMap` callback) plus a new `loadMapPlayerFunc` seam.

The map iteration order is nondeterministic; tests sort by id (the existing `recipientIds` helper already sorts).

- [ ] **Step 1: Write the failing tests**

Append to `skill/handler/recipients_test.go`. First, a seam installer for the map selector (place it near `installPartySeams`):

```go
// installMapSeams replaces the in-map id set and the per-player loader.
func installMapSeams(t *testing.T, inMap map[uint32]struct{}, players map[uint32]character.Model) {
	t.Helper()
	prevInMap := inMapCharacterIdsFunc
	prevLoad := loadMapPlayerFunc

	inMapCharacterIdsFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model) map[uint32]struct{} {
		return inMap
	}
	loadMapPlayerFunc = func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
		mc, ok := players[id]
		if !ok {
			return character.Model{}, errors.New("player not found")
		}
		return mc, nil
	}
	t.Cleanup(func() {
		inMapCharacterIdsFunc = prevInMap
		loadMapPlayerFunc = prevLoad
	})
}

// mkPlayerCharAt builds a character at (x,y) with the given hp.
func mkPlayerCharAt(id uint32, hp uint16, x, y int16) character.Model {
	return character.NewModelBuilder().SetId(id).SetHp(hp).SetMaxHp(1000).SetX(x).SetY(y).MustBuild()
}

// rectEffect returns an effect with the v83 Resurrection LT/RB rectangle.
func rectEffect(t *testing.T) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{
		Lt: &effect.PointRestModel{X: -400, Y: -350},
		Rb: &effect.PointRestModel{X: 400, Y: 250},
	})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return e
}

func TestSelectDeadInRangePartyMembers_KeepsOnlyDead(t *testing.T) {
	caster := uint32(1)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	p, _ := party.Extract(party.RestModel{Members: []party.MemberRestModel{
		{Id: 1, ChannelId: 0, MapId: 100000000, Online: true},
		{Id: 2, ChannelId: 0, MapId: 100000000, Online: true},
		{Id: 3, ChannelId: 0, MapId: 100000000, Online: true},
	}})
	inMap := map[uint32]struct{}{1: {}, 2: {}, 3: {}}
	members := map[uint32]character.Model{
		2: mkMemberChar(2, 0),   // dead
		3: mkMemberChar(3, 500), // alive
	}
	installPartySeams(t, p, nil, inMap, members)

	// bitmap with bits for slots 1 and 2 (members index 1,2 -> bits 4,3).
	bitmap := byte(1<<4 | 1<<3)
	got := SelectDeadInRangePartyMembers(testLogger(), context.Background(), f, caster, 0, 0, rectEffect(t), bitmap)
	if !eqIds(recipientIds(got), []uint32{2}) {
		t.Fatalf("got %v, want [2] (dead only)", recipientIds(got))
	}
}

func TestSelectDeadInRangePartyMembers_MissingRectangleReturnsNil(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	got := SelectDeadInRangePartyMembers(testLogger(), context.Background(), f, 1, 0, 0, effect.Model{}, 0x7E)
	if got != nil {
		t.Fatalf("got %v, want nil for missing rectangle", got)
	}
}

func TestSelectDeadInRangeMapPlayers_AllDeadRegardlessOfParty(t *testing.T) {
	caster := uint32(1)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	inMap := map[uint32]struct{}{1: {}, 2: {}, 3: {}, 4: {}}
	players := map[uint32]character.Model{
		1: mkPlayerCharAt(1, 800, 0, 0),    // caster (alive) — excluded
		2: mkPlayerCharAt(2, 0, 100, 50),   // dead, in range
		3: mkPlayerCharAt(3, 600, 0, 0),    // alive — excluded
		4: mkPlayerCharAt(4, 0, 5000, 0),   // dead but out of range — excluded
	}
	installMapSeams(t, inMap, players)

	got := SelectDeadInRangeMapPlayers(testLogger(), context.Background(), f, caster, 0, 0, rectEffect(t))
	if !eqIds(recipientIds(got), []uint32{2}) {
		t.Fatalf("got %v, want [2]", recipientIds(got))
	}
}

func TestSelectDeadInRangeMapPlayers_CapturesDeathCoords(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	inMap := map[uint32]struct{}{2: {}}
	players := map[uint32]character.Model{2: mkPlayerCharAt(2, 0, 123, -45)}
	installMapSeams(t, inMap, players)

	got := SelectDeadInRangeMapPlayers(testLogger(), context.Background(), f, 1, 0, 0, rectEffect(t))
	if len(got) != 1 || got[0].X() != 123 || got[0].Y() != -45 {
		t.Fatalf("got %+v, want one recipient at (123,-45)", got)
	}
}

func TestSelectInRangePartyMembers_StillExcludesDead(t *testing.T) {
	caster := uint32(1)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	p, _ := party.Extract(party.RestModel{Members: []party.MemberRestModel{
		{Id: 1, ChannelId: 0, MapId: 100000000, Online: true},
		{Id: 2, ChannelId: 0, MapId: 100000000, Online: true},
		{Id: 3, ChannelId: 0, MapId: 100000000, Online: true},
	}})
	inMap := map[uint32]struct{}{1: {}, 2: {}, 3: {}}
	members := map[uint32]character.Model{
		2: mkMemberChar(2, 0),   // dead -> excluded by living-only selector
		3: mkMemberChar(3, 500), // alive -> included
	}
	installPartySeams(t, p, nil, inMap, members)

	bitmap := byte(1<<4 | 1<<3)
	got := SelectInRangePartyMembers(testLogger(), context.Background(), f, caster, 0, 0, rectEffect(t), bitmap)
	if !eqIds(recipientIds(got), []uint32{3}) {
		t.Fatalf("got %v, want [3] (alive only)", recipientIds(got))
	}
}
```

> **Implementation note:** verify the exact `effect.RestModel` LT/RB field names and the `party.Extract`/`party.RestModel` shape against the current source before running — if they differ (e.g. `LT`/`RB` casing, or members built via `party.ExtractMember`/`mkPartyMember` as the existing tests do), adapt these test fixtures to the existing helpers in `recipients_test.go` rather than inventing new ones. The existing `mkPartyMember` + `installPartySeams` helpers are the canonical reference for party fixtures.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run 'SelectDead|StillExcludesDead' -v`
Expected: FAIL — `undefined: SelectDeadInRangePartyMembers` / `undefined: loadMapPlayerFunc`.

- [ ] **Step 3: Generalize the selector and add the new selectors**

In `skill/handler/recipients.go`:

(a) Change the `selectPartyMembers` signature to add `wantDead bool`:

```go
func selectPartyMembers(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model, memberBitmap byte, requireRect bool, wantDead bool,
) []PartyRecipient {
```

(b) Replace the hard-coded living-only skip (line 174, `if mc.Hp() == 0 { continue }`) with:

```go
		if wantDead {
			if mc.Hp() != 0 {
				continue
			}
		} else {
			if mc.Hp() == 0 {
				continue
			}
		}
```

(c) Update the two existing callers to pass `wantDead=false`:

```go
	return selectPartyMembers(l, ctx, f, casterId, casterX, casterY, e, memberBitmap, true, false)   // in SelectInRangePartyMembers
```
```go
	return selectPartyMembers(l, ctx, f, casterId, 0, 0, effect.Model{}, memberBitmap, false, false)  // in SelectPartyMembersInMap
```

(d) Add the new party-variant selector (after `SelectInRangePartyMembers`):

```go
// SelectDeadInRangePartyMembers is the dead-only counterpart of
// SelectInRangePartyMembers: same bitmap / same-channel-map / live-session /
// LT-RB-rectangle filters, but keeps only members with Hp()==0. Used by Bishop
// Resurrection. Missing rectangle returns nil (no one to revive in range).
func SelectDeadInRangePartyMembers(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model, memberBitmap byte,
) []PartyRecipient {
	lt, rb := e.LT(), e.RB()
	if lt.X() == 0 && lt.Y() == 0 && rb.X() == 0 && rb.Y() == 0 {
		return nil
	}
	return selectPartyMembers(l, ctx, f, casterId, casterX, casterY, e, memberBitmap, true, true)
}
```

(e) Add the `loadMapPlayerFunc` seam (near `loadPartyMemberFunc`):

```go
// loadMapPlayerFunc is the per-player character-load seam (GM-variant map-wide
// selection) tests can replace.
var loadMapPlayerFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
	return character.NewProcessor(l, ctx).GetById()(characterId)
}
```

(f) Add the GM-variant selector:

```go
// SelectDeadInRangeMapPlayers returns every dead player (Hp()==0) other than the
// caster who has a live session in the caster's field and whose position lies in
// the caster-relative LT/RB rectangle — party-agnostic. Used by GM / SuperGM
// Resurrection. Missing rectangle returns nil. The in-map id set is produced by
// inMapCharacterIdsFunc (which already mutex-guards the concurrent
// ForSessionsInMap callback); this function then loads and filters serially.
func SelectDeadInRangeMapPlayers(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model,
) []PartyRecipient {
	lt, rb := e.LT(), e.RB()
	if lt.X() == 0 && lt.Y() == 0 && rb.X() == 0 && rb.Y() == 0 {
		return nil
	}

	inMap := inMapCharacterIdsFunc(l, ctx, f)
	out := make([]PartyRecipient, 0, len(inMap))
	for id := range inMap {
		if id == casterId {
			continue
		}
		mc, err := loadMapPlayerFunc(l, ctx, id)
		if err != nil {
			l.WithError(err).Debugf("Skipping map player [%d] from resurrection recipients: fetch failed.", id)
			continue
		}
		if mc.Hp() != 0 {
			continue
		}
		dx := mc.X() - casterX
		dy := mc.Y() - casterY
		if dx < int16(lt.X()) || dx > int16(rb.X()) || dy < int16(lt.Y()) || dy > int16(rb.Y()) {
			continue
		}
		out = append(out, NewPartyRecipientBuilder().
			SetId(mc.Id()).
			SetX(mc.X()).
			SetY(mc.Y()).
			SetHp(mc.Hp()).
			SetMaxHp(mc.MaxHp()).
			Build())
	}
	return out
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -v`
Expected: PASS — all new tests **and** the pre-existing selector tests (the `wantDead=false` default preserved their behavior).

- [ ] **Step 5: Commit**

```bash
cd <worktree-root>
git add services/atlas-channel/atlas.com/channel/skill/handler/recipients.go services/atlas-channel/atlas.com/channel/skill/handler/recipients_test.go
git commit -m "feat(channel): add dead-target skill recipient selectors"
git branch --show-current  # must be task-111-resurrection-skill
```

---

## Task 6: Resurrection package — variant→selector dispatch

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/resurrection/recipients.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/resurrection/recipients_test.go`

Context: `selectByVariant` maps each skill ID to a dead-target selector — Bishop → party, GM/SuperGM → map. The two shared selectors are referenced through `selectDeadParty`/`selectDeadMap` package vars so tests can record which one fired.

- [ ] **Step 1: Write the failing test**

Create `skill/handler/resurrection/recipients_test.go`:

```go
package resurrection

import (
	"context"
	"io"
	"testing"

	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
}

func installSelectorSpies(t *testing.T) (partyCalled, mapCalled *bool) {
	t.Helper()
	pc, mc := false, false
	prevParty, prevMap := selectDeadParty, selectDeadMap
	selectDeadParty = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _, _ int16, _ effect.Model, _ byte) []channelhandler.PartyRecipient {
		pc = true
		return nil
	}
	selectDeadMap = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _, _ int16, _ effect.Model) []channelhandler.PartyRecipient {
		mc = true
		return nil
	}
	t.Cleanup(func() {
		selectDeadParty = prevParty
		selectDeadMap = prevMap
	})
	return &pc, &mc
}

func TestSelectByVariant_BishopUsesPartySelector(t *testing.T) {
	pc, mc := installSelectorSpies(t)
	selectByVariant(testLogger(), context.Background(), testField(), 1, 0, 0, effect.Model{}, 0x7E, skill2.BishopResurrectionId)
	if !*pc || *mc {
		t.Fatalf("Bishop: partyCalled=%v mapCalled=%v, want party only", *pc, *mc)
	}
}

func TestSelectByVariant_GmUsesMapSelector(t *testing.T) {
	pc, mc := installSelectorSpies(t)
	selectByVariant(testLogger(), context.Background(), testField(), 1, 0, 0, effect.Model{}, 0x7E, skill2.GmResurrectionId)
	if *pc || !*mc {
		t.Fatalf("GM: partyCalled=%v mapCalled=%v, want map only", *pc, *mc)
	}
}

func TestSelectByVariant_SuperGmUsesMapSelector(t *testing.T) {
	pc, mc := installSelectorSpies(t)
	selectByVariant(testLogger(), context.Background(), testField(), 1, 0, 0, effect.Model{}, 0x7E, skill2.SuperGmResurrectionId)
	if *pc || !*mc {
		t.Fatalf("SuperGM: partyCalled=%v mapCalled=%v, want map only", *pc, *mc)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go test ./skill/handler/resurrection/ -v`
Expected: FAIL — the package doesn't exist / `undefined: selectByVariant`.

- [ ] **Step 3: Create the dispatch file**

Create `skill/handler/resurrection/recipients.go`:

```go
package resurrection

import (
	"context"

	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/sirupsen/logrus"
)

// selectDeadParty / selectDeadMap are seams (aliases to the shared dead-target
// selectors) so the variant dispatch is unit-testable without the live stack.
var selectDeadParty = channelhandler.SelectDeadInRangePartyMembers
var selectDeadMap = channelhandler.SelectDeadInRangeMapPlayers

// selectByVariant routes each Resurrection variant to its recipient selector:
// Bishop -> dead party members in range; GM / SuperGM -> all dead players in
// range (party-agnostic).
func selectByVariant(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model, memberBitmap byte, skillId skill2.Id,
) []channelhandler.PartyRecipient {
	switch skillId {
	case skill2.BishopResurrectionId:
		return selectDeadParty(l, ctx, f, casterId, casterX, casterY, e, memberBitmap)
	default:
		// GmResurrectionId / SuperGmResurrectionId — party-agnostic.
		return selectDeadMap(l, ctx, f, casterId, casterX, casterY, e)
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go test ./skill/handler/resurrection/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd <worktree-root>
git add services/atlas-channel/atlas.com/channel/skill/handler/resurrection/recipients.go services/atlas-channel/atlas.com/channel/skill/handler/resurrection/recipients_test.go
git commit -m "feat(channel): add resurrection variant-to-selector dispatch"
git branch --show-current  # must be task-111-resurrection-skill
```

---

## Task 7: Resurrection package — the Apply handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/resurrection/resurrection.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/resurrection/resurrection_test.go`

Context: `Apply` matches the shared `handler.Handler` signature. Lifecycle: load caster (X/Y/Level) → `selectByVariant` → per recipient `setHP(0xFFFF)` **then** `warpToPosition(deathX, deathY)` (per-recipient failure logged + skipped) → broadcast self/foreign skill-use effect. Five seams (`loadCaster`, `selectDeadParty`/`selectDeadMap` from Task 6, `setHP`, `warpToPosition`, `broadcastEffects`) make every branch testable.

Design deviations grounded in code (state explicitly so the reviewer doesn't flag them):
- **No `warnIfMissingRectangle`** (heal's is unexported and package-private). The dead selectors already return nil on a zero LT/RB rectangle, so a missing rectangle yields a clean zero-recipient no-op without a separate guard. This is the deliberate simplification noted in design §4.2.
- **Caster-load failure returns `nil` without broadcasting** (matches Heal `heal.go:82-85`): the effect packet needs the caster's level, which we don't have on load failure. The "still broadcast on empty" rule in design §8 applies to the **empty recipient set** case (where the caster *did* load), which this handler honors.

- [ ] **Step 1: Write the failing test**

Create `skill/handler/resurrection/resurrection_test.go`:

```go
package resurrection

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

const (
	testCasterId = uint32(1001)
	testLevel    = byte(7)
)

func bishopInfo() packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill2.BishopResurrectionId)).
		SetSkillLevel(1).
		SetAffectedPartyMemberBitmap(0x7E).
		Build()
}

// installHandlerSeams swaps every Apply seam with deterministic stubs and
// returns a pointer to the recorded event log (e.g. "setHP:42:65535",
// "warp:42:100:50") and whether broadcastEffects fired.
func installHandlerSeams(
	t *testing.T,
	recipients []channelhandler.PartyRecipient,
	casterErr error,
	setHPErr map[uint32]error,
) (*[]string, *bool) {
	t.Helper()
	prevCaster, prevParty, prevMap := loadCaster, selectDeadParty, selectDeadMap
	prevSetHP, prevWarp, prevBroadcast := setHP, warpToPosition, broadcastEffects

	events := []string{}
	broadcastCalled := false

	loadCaster = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, int16, byte, error) {
		return 0, 0, testLevel, casterErr
	}
	selectDeadParty = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _, _ int16, _ effect.Model, _ byte) []channelhandler.PartyRecipient {
		return recipients
	}
	selectDeadMap = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _, _ int16, _ effect.Model) []channelhandler.PartyRecipient {
		return recipients
	}
	setHP = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, id uint32, amount uint16) error {
		events = append(events, fmt.Sprintf("setHP:%d:%d", id, amount))
		if setHPErr != nil {
			return setHPErr[id]
		}
		return nil
	}
	warpToPosition = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, id uint32, x, y int16) error {
		events = append(events, fmt.Sprintf("warp:%d:%d:%d", id, x, y))
		return nil
	}
	broadcastEffects = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ field.Model, _ uint32, _ byte, _ uint32, _ byte) {
		broadcastCalled = true
	}

	t.Cleanup(func() {
		loadCaster, selectDeadParty, selectDeadMap = prevCaster, prevParty, prevMap
		setHP, warpToPosition, broadcastEffects = prevSetHP, prevWarp, prevBroadcast
	})
	return &events, &broadcastCalled
}

func mkRecipient(id uint32, x, y int16) channelhandler.PartyRecipient {
	return channelhandler.NewPartyRecipientBuilder().SetId(id).SetX(x).SetY(y).Build()
}

func TestResurrection_RegistersAllThreeIds(t *testing.T) {
	for _, id := range []skill2.Id{skill2.BishopResurrectionId, skill2.GmResurrectionId, skill2.SuperGmResurrectionId} {
		h, ok := channelhandler.Lookup(id)
		if !ok || h == nil {
			t.Fatalf("Lookup(%d) = (%v, %v), want non-nil handler", id, h, ok)
		}
	}
}

func TestResurrection_SetHPBeforeWarpPerRecipient(t *testing.T) {
	events, broadcast := installHandlerSeams(t,
		[]channelhandler.PartyRecipient{mkRecipient(42, 100, 50), mkRecipient(43, -10, 20)},
		nil, nil)

	err := Apply(testLogger())(context.Background())(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply err: %v", err)
	}
	want := []string{"setHP:42:65535", "warp:42:100:50", "setHP:43:65535", "warp:43:-10:20"}
	if fmt.Sprint(*events) != fmt.Sprint(want) {
		t.Fatalf("events = %v, want %v", *events, want)
	}
	if !*broadcast {
		t.Fatal("broadcastEffects not called")
	}
}

func TestResurrection_EmptyRecipientsBroadcastsNoSetHP(t *testing.T) {
	events, broadcast := installHandlerSeams(t, nil, nil, nil)
	err := Apply(testLogger())(context.Background())(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply err: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("events = %v, want none (no recipients)", *events)
	}
	if !*broadcast {
		t.Fatal("broadcastEffects must fire even with no recipients")
	}
}

func TestResurrection_PerRecipientFailureIsolation(t *testing.T) {
	events, _ := installHandlerSeams(t,
		[]channelhandler.PartyRecipient{mkRecipient(42, 0, 0), mkRecipient(43, 0, 0)},
		nil,
		map[uint32]error{42: errors.New("setHP boom")})

	_ = Apply(testLogger())(context.Background())(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
	// 42 fails at setHP (no warp); 43 fully processed.
	want := []string{"setHP:42:65535", "setHP:43:65535", "warp:43:0:0"}
	if fmt.Sprint(*events) != fmt.Sprint(want) {
		t.Fatalf("events = %v, want %v", *events, want)
	}
}

func TestResurrection_CasterLoadErrorNoOp(t *testing.T) {
	events, broadcast := installHandlerSeams(t,
		[]channelhandler.PartyRecipient{mkRecipient(42, 0, 0)},
		errors.New("caster load failed"), nil)

	err := Apply(testLogger())(context.Background())(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply err: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("events = %v, want none on caster load failure", *events)
	}
	if *broadcast {
		t.Fatal("broadcastEffects must not fire on caster load failure")
	}
}
```

> **Implementation note:** `testLogger`/`testField` are defined once in `recipients_test.go` (Task 6) — same package, so do **not** redefine them here. If the compiler flags a redeclaration, delete the duplicate from whichever file.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go test ./skill/handler/resurrection/ -run TestResurrection -v`
Expected: FAIL — `undefined: Apply` / `undefined: loadCaster`.

- [ ] **Step 3: Create the handler**

Create `skill/handler/resurrection/resurrection.go`:

```go
package resurrection

import (
	"context"
	"math"

	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	channelmap "atlas-channel/map"
	"atlas-channel/portal"
	"atlas-channel/session"
	channelhandler "atlas-channel/skill/handler"
	socketHandler "atlas-channel/socket/handler"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

func init() {
	channelhandler.Register(skill2.BishopResurrectionId, Apply)
	channelhandler.Register(skill2.GmResurrectionId, Apply)
	channelhandler.Register(skill2.SuperGmResurrectionId, Apply)
}

// loadCaster returns the caster's position and level. Seam for tests.
var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, byte, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, 0, 0, err
	}
	return c.X(), c.Y(), c.Level(), nil
}

// setHP sends an absolute SET_HP command; atlas-character clamps to effective
// MaxHP, so math.MaxUint16 yields a full-HP restore. Seam for tests.
var setHP = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, amount uint16) error {
	return character.NewProcessor(l, ctx).SetHP(f, characterId, amount)
}

// warpToPosition warps a character to (x,y) on the current map via the task-093
// chase-warp primitive, which fires the client's OnRevive for a dead target.
// Seam for tests.
var warpToPosition = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, x, y int16) error {
	return portal.NewProcessor(l, ctx).WarpToPosition(f, characterId, f.MapId(), x, y)
}

// broadcastEffects fires the holy-light skill-use effect to the caster and the
// foreign skill-use effect to other players in the map. Seam for tests.
var broadcastEffects = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, casterId uint32, casterLevel byte, skillId uint32, skillLevel byte) {
	_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(f.Channel())(
		casterId,
		socketHandler.AnnounceSkillUse(l)(ctx)(wp)(skillId, casterLevel, skillLevel),
	)
	_ = channelmap.NewProcessor(l, ctx).ForOtherSessionsInMap(
		f, casterId,
		socketHandler.AnnounceForeignSkillUse(l)(ctx)(wp)(casterId, skillId, casterLevel, skillLevel),
	)
}

// Apply is the Resurrection handler installed in the per-skill registry for the
// Bishop and GM/SuperGM skill IDs. By the time it runs, UseSkill has already
// validated ownership/level, loaded the WZ effect, consumed MP, and applied the
// cooldown. For each dead recipient it restores full HP then warps the recipient
// to its own death coordinates (the chase-warp fires the client's OnRevive).
//
// Per-recipient failures are logged and skipped; caster-load failure / empty
// recipient set are clean no-ops (the latter still broadcasts the effect).
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer, f field.Model, characterId uint32,
	info packetmodel.SkillUsageInfo, e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer, f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(
			wp writer.Producer, f field.Model, characterId uint32,
			info packetmodel.SkillUsageInfo, e effect.Model,
		) error {
			casterX, casterY, casterLevel, err := loadCaster(l, ctx, characterId)
			if err != nil {
				l.WithError(err).Errorf("Resurrection: failed to load caster [%d].", characterId)
				return nil
			}

			recipients := selectByVariant(l, ctx, f, characterId, casterX, casterY, e, info.AffectedPartyMemberBitmap(), skill2.Id(info.SkillId()))

			for _, r := range recipients {
				if hpErr := setHP(l, ctx, f, r.Id(), math.MaxUint16); hpErr != nil {
					l.WithError(hpErr).Errorf("Resurrection: SetHP failed for recipient [%d]; skipping warp.", r.Id())
					continue
				}
				if wErr := warpToPosition(l, ctx, f, r.Id(), r.X(), r.Y()); wErr != nil {
					l.WithError(wErr).Errorf("Resurrection: WarpToPosition failed for recipient [%d].", r.Id())
					continue
				}
				l.Debugf("Resurrection: revived [%d] at (%d,%d).", r.Id(), r.X(), r.Y())
			}

			broadcastEffects(l, ctx, wp, f, characterId, casterLevel, info.SkillId(), info.SkillLevel())

			l.Debugf("Resurrection: caster=[%d] skill=[%d] level=[%d] recipients=[%d].",
				characterId, info.SkillId(), info.SkillLevel(), len(recipients))
			return nil
		}
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && go test ./skill/handler/resurrection/ -v`
Expected: PASS. (`TestResurrection_RegistersAllThreeIds` passes because `init()` runs on package load.)

- [ ] **Step 5: Commit**

```bash
cd <worktree-root>
git add services/atlas-channel/atlas.com/channel/skill/handler/resurrection/resurrection.go services/atlas-channel/atlas.com/channel/skill/handler/resurrection/resurrection_test.go
git commit -m "feat(channel): add Resurrection skill handler"
git branch --show-current  # must be task-111-resurrection-skill
```

---

## Task 8: Wire the handler into production registrations

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`

Context: The resurrection package's `init()` only runs if some package imports it. `registrations.go` is blank-imported by `main.go` for exactly this purpose (the Task 7 registration test passed only because the test binary imports the package directly — production needs this wiring).

- [ ] **Step 1: Add the blank import**

In `skill/handler/registrations/registrations.go`, add to the import block (keep imports gofmt-sorted):

```go
	_ "atlas-channel/skill/handler/resurrection" // Bishop/GM/SuperGM Resurrection — task-111
```

- [ ] **Step 2: Verify it compiles**

Run: `cd <worktree-root>/services/atlas-channel/atlas.com/channel && gofmt -w skill/handler/registrations/registrations.go && go build ./...`
Expected: clean build.

- [ ] **Step 3: Commit**

```bash
cd <worktree-root>
git add services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go
git commit -m "feat(channel): register resurrection handler in production"
git branch --show-current  # must be task-111-resurrection-skill
```

---

## Task 9: Full module verification (per CLAUDE.md)

**Files:** none (verification only).

Context: Two modules changed — `libs/atlas-constants` and `atlas-channel`. Both have `go.mod`, so both need the full gate, and any service whose `go.mod` was touched needs a docker bake. `libs/atlas-constants` is consumed by many services, so a broad bake is the safe check.

- [ ] **Step 1: Tests with the race detector**

```bash
cd <worktree-root>/libs/atlas-constants && go test -race ./...
cd <worktree-root>/services/atlas-channel/atlas.com/channel && go test -race ./...
```
Expected: all PASS, no race warnings.

- [ ] **Step 2: Vet**

```bash
cd <worktree-root>/libs/atlas-constants && go vet ./...
cd <worktree-root>/services/atlas-channel/atlas.com/channel && go vet ./...
```
Expected: clean, no output.

- [ ] **Step 3: Build**

```bash
cd <worktree-root>/services/atlas-channel/atlas.com/channel && go build ./...
```
Expected: clean.

- [ ] **Step 4: Redis key guard**

```bash
cd <worktree-root> && GOWORK=off tools/redis-key-guard.sh
```
Expected: clean (no new raw keyed go-redis usage — this task adds none).

- [ ] **Step 5: Docker bake**

`atlas-channel`'s code changed and `libs/atlas-constants` is a shared lib consumed widely. Bake atlas-channel:

```bash
cd <worktree-root> && docker buildx bake atlas-channel
```
Expected: build succeeds. (No new `libs/<name>` was added, so no Dockerfile/`go.work` COPY edits are needed — but the bake confirms it.)

- [ ] **Step 6: Final state check (no commit — verification only)**

```bash
cd <worktree-root>
git status   # working tree clean
git log --oneline -9
git rev-parse --show-toplevel   # must end with /.worktrees/task-111-resurrection-skill
git branch --show-current       # must be task-111-resurrection-skill
```

---

## Live verification gates (post-implementation, not blockers to code completion)

These are settled on the running environment (PRD §9 / design §12) and recorded in the eventual PR description, not in code:

- **OQ-1** — On v83, cast Bishop Resurrection on a dead party member in range; confirm the client closes the death prompt and stands the avatar up at the death position.
- **OQ-2** — If OQ-1 is unreliable, build the conditional `revive`-byte fallback (design §7): add a chase+revive variant in `libs/atlas-packet/field/clientbound/warp_to_map.go` writing the `revive` byte as `1`, and plumb a `UseRevive` flag through the portal/maps warp command + `MAP_CHANGED` body + channel `warpCharacter` branch. **Do not build this up front.**
- **OQ-3** — Watch for same-map warp despawn/respawn flicker for observers.
- **OQ-4** — Confirm the tracked recipient X/Y is close enough to the actual death position; if not, source death coords from atlas-maps location state.
- **OQ-5** — Confirm the dead-player chase-warp revive on v87/v95/JMS, and that the GM/SuperGM skill IDs + WZ range exist per version.

---

## Self-Review

**Spec coverage** (design §13 file-level inventory + PRD §4 FRs):

| Item | Task |
|---|---|
| New `resurrection/` package + 3-ID registration (FR-1, FR-2) | 6, 7, 8 |
| Handler matches `handler.Handler` signature, invoked from UseSkill (FR-3) | 7 |
| Dead-only recipient selection (FR-4) | 5b |
| Bishop = dead party in range (FR-5) | 5b, 6 |
| GM/SuperGM = all dead in range, party-agnostic (FR-6) | 5b, 6 |
| Caster never a recipient (FR-7) | 5b (caster excluded in both selectors) |
| Recipient carries death X/Y (FR-8) | 5b |
| No eligible target = clean no-op (FR-9) | 7 |
| Full HP before warp (FR-10) | 7 (setHP before warp), 2-4 (SET_HP) |
| Warp to death position, same map, chase-warp (FR-11) | 7 |
| Holy-light effect to self + foreign (FR-12) | 7 |
| No XP (FR-13) | 7 (no XP path) |
| MP/cooldown by UseSkill, range from WZ (FR-14, FR-15) | inherited (UseSkill `common.go:73-95`), not re-implemented |
| Multi-version (FR-16) | inherited (chase-warp + config), gate OQ-5 |
| Missing `GmResurrectionId` constant (design key finding) | 1 |
| Generalize shared selector (design D2) | 5b |
| New absolute SET_HP producer (design D1) | 2, 3, 4 |
| Verification gate (acceptance criteria) | 9 |

No spec requirement is left without a task. `character_damage.go` is intentionally untouched (invincibility out of scope) — no task modifies it.

**Placeholder scan:** No `TODO`/`TBD`/"add error handling"/"similar to Task N" placeholders; every code step contains the full code.

**Type consistency:** `SetHP`/`SetHPCommandProvider`/`SetHPCommandBody`/`CommandSetHP` are consistent across Tasks 2-4. `selectByVariant`, `selectDeadParty`, `selectDeadMap` signatures match between Task 6's definitions and Task 7's seam stubs. `SelectDeadInRangePartyMembers` (8 params incl. bitmap) vs `SelectDeadInRangeMapPlayers` (7 params, no bitmap) are used consistently in Tasks 5b/6. The `loadCaster` seam returns `(int16, int16, byte, error)` consistently in Task 7's stub and impl. `math.MaxUint16` (= `65535` = `0xFFFF`) is the SET_HP sentinel throughout.
