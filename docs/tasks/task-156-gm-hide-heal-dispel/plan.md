# SuperGM Skills: Hide + Heal & Dispel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement two SuperGM active skills end-to-end in `atlas-channel` — Heal + Dispel (`9101000`, restores HP/MP and purges disease debuffs for every player in the caster's map) and Hide (`9101004`, a persistent-buff toggle that makes the caster invisible/untargetable to other players and survives map changes).

**Architecture:** Both skills register per-skill handlers via `channelhandler.Register(skillId, Apply)` (blank-imported from `skill/handler/registrations`) and are dispatched from the generic `UseSkill` orchestrator after it validates the cast and consumes MP/cooldown. Heal + Dispel reuses the existing `ChangeHP`/`ChangeMP` character commands plus a new channel-side `CancelByTypes` buff producer. Hide represents "hidden" as a `DARK_SIGHT` buff sourced from `SuperGmHideId` with a `math.MaxInt32` duration; a suppression gate in the map consumer's single spawn choke point reads that buff and refuses to spawn the caster to other viewers. All new code is in `atlas-channel`; `atlas-data`, `atlas-buffs`, and `libs/atlas-constants` are unchanged.

**Tech Stack:** Go, Kafka (segmentio), JSON:API REST between services, the project's immutable-model + Builder + processor-seam patterns. Tests run offline against function seams (no Kafka/REST/session), using the character `modelBuilder` for fixtures (no `*_testhelpers.go`).

## Global Constraints

- **Only `atlas-channel`'s `go.mod` is touched.** `atlas-data`, `atlas-buffs`, `libs/atlas-constants` get **no** changes (all ids/types already exist). Do not edit them.
- **Reuse shared constants (DOM-21).** Disease stat strings, skill ids, job ids, and temporary-stat types come from `libs/atlas-constants` — never bare literals. Verified symbols: `skill.SuperGmHealDispelId = Id(9101000)`, `skill.SuperGmHideId = Id(9101004)`, `skill.RogueDarkSightId = Id(4001003)`, `job.SuperGmId = Id(910)`, `character.TemporaryStatTypeDarkSight = "DARK_SIGHT"`, and the 11 disease constants (`TemporaryStatTypeStun/Poison/Seal/Darkness/Weaken/Curse/Seduce/Confuse/Undead/Slow/StopPortion`).
- **SuperGM gate uses `job.SuperGmId` only.** `job.IsA(c.JobId(), job.SuperGmId)`. Plain `GmId` (900) must NOT pass — verified: `job.Is(900, 910)` is false. `c.JobId()` already returns `job.Id`; no cast needed.
- **No experience** is ever awarded by Heal + Dispel (never call `AwardExperience`).
- **Per-recipient failure isolation:** a failed HP/MP/dispel for one player is logged and `continue`d; it never aborts the cast for other players.
- **`ChangeHP`/`ChangeMP` take `int16`.** All computed HP/MP deltas must be clamped to `[0, effMax-current]` and to the `int16` ceiling before the command call.
- **Buff duration:** atlas-buffs rejects `duration <= 0`; the "permanent" convention is `int32(math.MaxInt32)` (~24.8 days), exactly as mounts (`skill/handler/mount.go` `MountBuffDuration`).
- **TDD, DRY, YAGNI, frequent commits.** Builder pattern for fixtures; no test-only constructor files.
- **Working directory for all Go commands:** `services/atlas-channel/atlas.com/channel` (module `atlas-channel`). Paths below are relative to that directory unless prefixed `services/` or `docs/`.
- **Design decision resolved during planning (Hide foreign broadcast):** The design's §3.4 step-5 code snippet (`foreign only if !hidden`) contradicts its own parenthetical ("suppressed both when hiding … and when revealing"). Per FR-17's intent ("MUST NOT reveal the caster's position") the Hide handler broadcasts the **self** skill-use animation only and **never** broadcasts the foreign animation in either toggle direction. Heal + Dispel keeps the general FR-17 rule: foreign only when the caster is currently visible.
- **Execute-time verification gates (per CLAUDE.md, do not skip):** (a) confirm the live `9101000` WZ recovery fields (`hp`/`mp`/`hpR`/`mpR`) against live WZ data before claiming the heal magnitude correct — the flat+ratio formula tolerates either shape but the values are unverified in-repo; (b) byte-verify any `CharacterSpawn`/`CharacterDespawn`/buff give/cancel packet exercised on the hide/reveal path, and confirm the self `DARK_SIGHT` give serializes non-zero so the v83 client's `CUser::IsDarkSight` reads it.

---

## File Structure

**New files:**
- `data/skill/effect/model_test.go` — accessor tests (Task 1).
- `character/buff/hidden.go` — `IsGmHidden` predicate (Task 2).
- `character/buff/hidden_test.go` — predicate tests (Task 2).
- `skill/handler/recipients_map_test.go` — `SelectAllCharactersInMap` tests (Task 4). *(Same `handler` package as `recipients_test.go`; a separate file keeps the map-wide selector's tests isolated.)*
- `skill/handler/healdispel/healdispel.go` — Heal + Dispel handler (Task 5).
- `skill/handler/healdispel/healdispel_test.go` — handler tests (Task 5).
- `skill/handler/hide/hide.go` — Hide handler (Task 7).
- `skill/handler/hide/hide_test.go` — handler tests (Task 7).

**Modified files:**
- `data/skill/effect/model.go` — add `MP()`, `HpR()`, `MpR()` accessors (Task 1).
- `character/buff/kafka.go`?  → **`kafka/message/buff/kafka.go`** — add `CommandTypeCancelByTypes` + `CancelByTypesCommandBody` (Task 3).
- `character/buff/producer.go` — add `CancelByTypesCommandProvider` (Task 3).
- `character/buff/processor.go` — add `CancelByTypes` to interface + impl (Task 3).
- `skill/handler/recipients.go` — add `mp`/`maxMp` to `PartyRecipient` + builder + `SelectAllCharactersInMap` (Task 4).
- `kafka/consumer/map/consumer.go` — suppression gate in `spawnCharacterForSession`; exported `SpawnCharacterInMap`/`DespawnCharacterInMap` (Task 6).
- `skill/handler/registrations/registrations.go` — blank-import `healdispel` (Task 5) and `hide` (Task 7).

---

## Task 1: Effect recovery accessors

Surface the MP flat value and HP/MP recovery ratios the Heal component reads. The fields (`mp`, `hpr`, `mpr`) are already parsed by `Extract` (`data/skill/effect/rest.go:92-95`) into private `Model` fields; only accessor methods are missing (`HP()` already exists at `model.go:111`).

**Files:**
- Modify: `data/skill/effect/model.go`
- Test: `data/skill/effect/model_test.go` (create)

**Interfaces:**
- Consumes: `effect.Model` with private fields `mp uint16`, `hpr float64`, `mpr float64` (already populated by `Extract`).
- Produces: `func (m Model) MP() uint16`, `func (m Model) HpR() float64`, `func (m Model) MpR() float64`.

- [ ] **Step 1: Write the failing test**

Create `data/skill/effect/model_test.go`:

```go
package effect

import (
	"testing"
)

func TestRecoveryAccessors(t *testing.T) {
	m, err := Extract(RestModel{Hp: 100, Mp: 50, HPR: 0.5, MPR: 0.25})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if got := m.HP(); got != 100 {
		t.Errorf("HP() = %d, want 100", got)
	}
	if got := m.MP(); got != 50 {
		t.Errorf("MP() = %d, want 50", got)
	}
	if got := m.HpR(); got != 0.5 {
		t.Errorf("HpR() = %v, want 0.5", got)
	}
	if got := m.MpR(); got != 0.25 {
		t.Errorf("MpR() = %v, want 0.25", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./data/skill/effect/ -run TestRecoveryAccessors -v`
Expected: FAIL — `m.MP undefined`, `m.HpR undefined`, `m.MpR undefined`.

- [ ] **Step 3: Add the accessors**

In `data/skill/effect/model.go`, immediately after the existing `HP()` method (around line 113), add:

```go
// MP exposes the skill's `mp` flat recovery attribute (used by the GM
// Heal + Dispel restore formula, alongside HP()).
func (m Model) MP() uint16 {
	return m.mp
}

// HpR exposes the skill's `hpR` recovery ratio (fraction of effective
// MaxHp restored). Zero means no ratio component.
func (m Model) HpR() float64 {
	return m.hpr
}

// MpR exposes the skill's `mpR` recovery ratio (fraction of effective
// MaxMp restored). Zero means no ratio component.
func (m Model) MpR() float64 {
	return m.mpr
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./data/skill/effect/ -run TestRecoveryAccessors -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add data/skill/effect/model.go data/skill/effect/model_test.go
git commit -m "feat(channel): expose MP/HpR/MpR accessors on skill effect model (task-156)"
```

---

## Task 2: `IsGmHidden` buff predicate

A single shared predicate answering "is this character GM-hidden?" — keyed on a buff whose `SourceId == SuperGmHideId` (NOT on the `DARK_SIGHT` stat, because Rogue Dark Sight also produces `DARK_SIGHT` but must stay visible). Three call sites depend on it (heal handler FR-17, map suppression gate, hide handler toggle), so it lives in the shared `character/buff` package to stay DRY.

**Files:**
- Create: `character/buff/hidden.go`
- Test: `character/buff/hidden_test.go` (create)

**Interfaces:**
- Consumes: `[]buff.Model`, each with `SourceId() int32` and `Expired() bool` (verified in `character/buff/model.go`); `buff.NewBuff(sourceId int32, level byte, duration int32, changes []stat.Model, createdAt, expiresAt time.Time) Model`.
- Produces: `func IsGmHidden(bs []Model) bool`.

- [ ] **Step 1: Write the failing test**

Create `character/buff/hidden_test.go`:

```go
package buff

import (
	"math"
	"testing"
	"time"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestIsGmHidden(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	hide := NewBuff(int32(skill2.SuperGmHideId), 1, math.MaxInt32, nil, time.Now(), future)
	if !IsGmHidden([]Model{hide}) {
		t.Errorf("IsGmHidden = false for an active SuperGmHide buff, want true")
	}

	// Rogue Dark Sight is a different source and must NOT read as GM-hidden,
	// even though it also produces a DARK_SIGHT stat.
	darkSight := NewBuff(int32(skill2.RogueDarkSightId), 1, 1000, nil, time.Now(), future)
	if IsGmHidden([]Model{darkSight}) {
		t.Errorf("IsGmHidden = true for a Rogue Dark Sight buff, want false")
	}

	// An expired SuperGmHide buff does not count as hidden.
	expired := NewBuff(int32(skill2.SuperGmHideId), 1, math.MaxInt32, nil, past.Add(-time.Hour), past)
	if IsGmHidden([]Model{expired}) {
		t.Errorf("IsGmHidden = true for an expired SuperGmHide buff, want false")
	}

	if IsGmHidden(nil) {
		t.Errorf("IsGmHidden = true for nil buff slice, want false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./character/buff/ -run TestIsGmHidden -v`
Expected: FAIL — `undefined: IsGmHidden`.

- [ ] **Step 3: Write the predicate**

Create `character/buff/hidden.go`:

```go
package buff

import (
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// IsGmHidden reports whether any buff in bs is an active GM-hide buff — one
// sourced from the SuperGM Hide skill (SuperGmHideId) and not yet expired.
//
// Keying on SourceId, NOT the DARK_SIGHT stat, is essential: Rogue Dark Sight
// (RogueDarkSightId) also produces a DARK_SIGHT stat but must remain visible
// to other players. Only a SuperGmHide-sourced buff means "GM-hidden."
func IsGmHidden(bs []Model) bool {
	for _, b := range bs {
		if b.SourceId() == int32(skill2.SuperGmHideId) && !b.Expired() {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./character/buff/ -run TestIsGmHidden -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add character/buff/hidden.go character/buff/hidden_test.go
git commit -m "feat(channel): add IsGmHidden buff predicate keyed on SuperGmHide source (task-156)"
```

---

## Task 3: `CancelByTypes` buff producer + processor (dispel)

The channel currently has only `APPLY`/`CANCEL` buff commands. Add `CANCEL_BY_TYPES`, mirroring the working `atlas-consumables` template (`services/atlas-consumables/atlas.com/consumables/character/buff/producer.go:57-72` and `kafka/message/character/buff/kafka.go:44-46`). `atlas-buffs` **already** consumes `CANCEL_BY_TYPES` → `CancelByStatTypes` (verified `services/atlas-buffs/.../kafka/consumer/character/consumer.go:81`) and emits `EXPIRED` events, which the existing channel buff-status consumer broadcasts to self + foreign — so no broadcast code and no atlas-buffs change.

**Files:**
- Modify: `kafka/message/buff/kafka.go`
- Modify: `character/buff/producer.go`
- Modify: `character/buff/processor.go`
- Test: `character/buff/producer_test.go` (create)

**Interfaces:**
- Consumes: `buff.Command[E]`, `field.Model`, `producer.CreateKey`, `producer.SingleMessageProvider` (patterns already in `producer.go`); `producer.ProviderImpl(l)(ctx)(topic)(provider)` (pattern already in `processor.go`).
- Produces:
  - `const buff.CommandTypeCancelByTypes = "CANCEL_BY_TYPES"` and `type buff.CancelByTypesCommandBody struct { Types []string }` in `kafka/message/buff`.
  - `func CancelByTypesCommandProvider(f field.Model, characterId uint32, types []string) model.Provider[[]kafka.Message]` in `character/buff`.
  - `CancelByTypes(f field.Model, characterId uint32, types []string) error` on `buff.Processor` interface + `*ProcessorImpl`.

- [ ] **Step 1: Add the message type + body**

In `kafka/message/buff/kafka.go`, extend the command-const block (currently `CommandTypeApply`/`CommandTypeCancel`) and add the body struct after `CancelCommandBody`:

```go
const (
	EnvCommandTopic          = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply         = "APPLY"
	CommandTypeCancel        = "CANCEL"
	CommandTypeCancelByTypes = "CANCEL_BY_TYPES"
)
```

```go
type CancelByTypesCommandBody struct {
	Types []string `json:"types"`
}
```

- [ ] **Step 2: Add the producer provider**

In `character/buff/producer.go`, after `CancelCommandProvider`, add:

```go
func CancelByTypesCommandProvider(f field.Model, characterId uint32, types []string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.CancelByTypesCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeCancelByTypes,
		Body: buff.CancelByTypesCommandBody{
			Types: append([]string(nil), types...),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 3: Write the failing producer test**

Create `character/buff/producer_test.go`:

```go
package buff

import (
	"encoding/json"
	"testing"

	buffmsg "atlas-channel/kafka/message/buff"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestCancelByTypesCommandProvider(t *testing.T) {
	f := field.NewBuilder(0, 0, 100000000).Build()
	types := []string{"STUN", "POISON"}

	msgs, err := CancelByTypesCommandProvider(f, 42, types)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd buffmsg.Command[buffmsg.CancelByTypesCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if cmd.Type != buffmsg.CommandTypeCancelByTypes {
		t.Errorf("Type = %q, want %q", cmd.Type, buffmsg.CommandTypeCancelByTypes)
	}
	if cmd.CharacterId != 42 {
		t.Errorf("CharacterId = %d, want 42", cmd.CharacterId)
	}
	if len(cmd.Body.Types) != 2 || cmd.Body.Types[0] != "STUN" || cmd.Body.Types[1] != "POISON" {
		t.Errorf("Body.Types = %v, want [STUN POISON]", cmd.Body.Types)
	}
}
```

- [ ] **Step 4: Run test to verify it fails, then passes**

Run: `go test ./character/buff/ -run TestCancelByTypesCommandProvider -v`
Expected first: FAIL (compile error `undefined: CancelByTypesCommandProvider` before Step 2 is saved). After Steps 1–2 are saved: PASS.

- [ ] **Step 5: Add the processor method**

In `character/buff/processor.go`, add `CancelByTypes` to the `Processor` interface (after `Cancel`):

```go
	Cancel(f field.Model, characterId uint32, sourceId int32) error
	CancelByTypes(f field.Model, characterId uint32, types []string) error
```

and the impl (after the existing `Cancel` method):

```go
func (p *ProcessorImpl) CancelByTypes(f field.Model, characterId uint32, types []string) error {
	p.l.Debugf("Character [%d] cancelling buffs by types %v.", characterId, types)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(CancelByTypesCommandProvider(f, characterId, types))
}
```

- [ ] **Step 6: Run the package tests + build**

Run: `go test ./character/buff/ -v && go build ./character/buff/...`
Expected: PASS, build clean. (Any in-repo mock implementing `buff.Processor` must also gain `CancelByTypes`; grep first: `grep -rln "buff.Processor" --include=*.go .` and add a stub to any mock that fails to build.)

- [ ] **Step 7: Commit**

```bash
git add kafka/message/buff/kafka.go character/buff/producer.go character/buff/processor.go character/buff/producer_test.go
git commit -m "feat(channel): add CancelByTypes buff producer + processor for dispel (task-156)"
```

---

## Task 4: `SelectAllCharactersInMap` selector (map-wide recipients)

Heal + Dispel targets every player in the map, party or not — the existing selectors are party-bitmap scoped (`recipients.go:132` rejects `memberBitmap == 0`). Add a map-wide selector and extend `PartyRecipient` with MP fields so the handler gets a full HP/MP snapshot without a second character load.

**Files:**
- Modify: `skill/handler/recipients.go`
- Test: `skill/handler/recipients_map_test.go` (create)

**Interfaces:**
- Consumes: seams `inMapCharacterIdsFunc(l, ctx, f) map[uint32]struct{}` and `loadPartyMemberFunc(l, ctx, id) (character.Model, error)` (both already in `recipients.go`); `character.Model` accessors `Id()/X()/Y()/Hp()/MaxHp()/Mp()/MaxMp()`.
- Produces:
  - `PartyRecipient` gains `mp`, `maxMp` fields with getters `Mp() uint16`, `MaxMp() uint16` and builder setters `SetMp`, `SetMaxMp`.
  - `func SelectAllCharactersInMap(l logrus.FieldLogger, ctx context.Context, f field.Model) []PartyRecipient`.

- [ ] **Step 1: Extend PartyRecipient with MP fields**

In `skill/handler/recipients.go`, add `mp`/`maxMp` to the struct, getters, and builder setters:

```go
type PartyRecipient struct {
	id    uint32
	x     int16
	y     int16
	hp    uint16
	maxHp uint16
	mp    uint16
	maxMp uint16
}
```

```go
func (r PartyRecipient) Mp() uint16    { return r.mp }
func (r PartyRecipient) MaxMp() uint16 { return r.maxMp }
```

```go
func (b *PartyRecipientBuilder) SetMp(v uint16) *PartyRecipientBuilder    { b.r.mp = v; return b }
func (b *PartyRecipientBuilder) SetMaxMp(v uint16) *PartyRecipientBuilder { b.r.maxMp = v; return b }
```

(The existing party selectors leave `mp`/`maxMp` at zero — harmless, Cleric Heal ignores MP.)

- [ ] **Step 2: Write the failing test**

Create `skill/handler/recipients_map_test.go`:

```go
package handler

import (
	"context"
	"testing"

	"atlas-channel/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus"
	"io"
)

func mapTestLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func mkFullChar(id uint32, hp, maxHp, mp, maxMp uint16) character.Model {
	return character.NewModelBuilder().
		SetId(id).SetHp(hp).SetMaxHp(maxHp).SetMp(mp).SetMaxMp(maxMp).MustBuild()
}

func TestSelectAllCharactersInMap(t *testing.T) {
	prevInMap := inMapCharacterIdsFunc
	prevMember := loadPartyMemberFunc
	t.Cleanup(func() {
		inMapCharacterIdsFunc = prevInMap
		loadPartyMemberFunc = prevMember
	})

	inMapCharacterIdsFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model) map[uint32]struct{} {
		return map[uint32]struct{}{1: {}, 2: {}, 3: {}}
	}
	members := map[uint32]character.Model{
		1: mkFullChar(1, 100, 500, 20, 200),
		2: mkFullChar(2, 0, 800, 0, 300), // HP 0 is NOT filtered by the map-wide selector
		// id 3 intentionally absent -> load error -> skipped
	}
	loadPartyMemberFunc = func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
		mc, ok := members[id]
		if !ok {
			return character.Model{}, errFakeNotFound
		}
		return mc, nil
	}

	got := SelectAllCharactersInMap(mapTestLogger(), context.Background(), field.NewBuilder(0, 0, 100000000).Build())

	ids := recipientIds(got) // helper already defined in recipients_test.go
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("recipient ids = %v, want [1 2] (id 3 skipped on load error, id 2 kept despite HP 0)", ids)
	}
	// Verify the MP snapshot flows through for a known recipient.
	for _, r := range got {
		if r.Id() == 1 {
			if r.Hp() != 100 || r.MaxHp() != 500 || r.Mp() != 20 || r.MaxMp() != 200 {
				t.Errorf("recipient 1 snapshot = hp %d/%d mp %d/%d, want 100/500 20/200",
					r.Hp(), r.MaxHp(), r.Mp(), r.MaxMp())
			}
		}
	}
}

var errFakeNotFound = &fakeErr{}

type fakeErr struct{}

func (*fakeErr) Error() string { return "not found" }
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./skill/handler/ -run TestSelectAllCharactersInMap -v`
Expected: FAIL — `undefined: SelectAllCharactersInMap` (and `SetMp`/`Mp` undefined if Step 1 not yet saved).

- [ ] **Step 4: Write the selector**

In `skill/handler/recipients.go`, after `SelectPartyMembersInMap`, add:

```go
// SelectAllCharactersInMap returns a recipient for EVERY character with a live
// session in the field f, irrespective of party membership, HP, or position.
// Unlike the party selectors it applies no bitmap, no LT/RB rectangle, and no
// HP>0 filter — it is the map-wide selector for GM Heal + Dispel, which
// benefits every player in the map INCLUDING the caster.
//
// The in-map id set comes from the same live-session source the spawn paths
// use (ForSessionsInMap via inMapCharacterIdsFunc). Each id is loaded for its
// HP/MP snapshot; a member whose load fails is logged and skipped.
func SelectAllCharactersInMap(l logrus.FieldLogger, ctx context.Context, f field.Model) []PartyRecipient {
	inMap := inMapCharacterIdsFunc(l, ctx, f)
	out := make([]PartyRecipient, 0, len(inMap))
	for id := range inMap {
		mc, err := loadPartyMemberFunc(l, ctx, id)
		if err != nil {
			l.WithError(err).Debugf("SelectAllCharactersInMap: skipping character [%d]: fetch failed.", id)
			continue
		}
		out = append(out, NewPartyRecipientBuilder().
			SetId(mc.Id()).
			SetX(mc.X()).
			SetY(mc.Y()).
			SetHp(mc.Hp()).
			SetMaxHp(mc.MaxHp()).
			SetMp(mc.Mp()).
			SetMaxMp(mc.MaxMp()).
			Build())
	}
	return out
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./skill/handler/ -run TestSelectAllCharactersInMap -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add skill/handler/recipients.go skill/handler/recipients_map_test.go
git commit -m "feat(channel): add map-wide recipient selector + MP snapshot for GM heal (task-156)"
```

---

## Task 5: Heal + Dispel handler

Register `SuperGmHealDispelId` and, for every player in the map, restore HP/MP (flat + ratio, clamped) and dispel the 11 disease debuffs. All logic behind a `deps` struct (mirroring `skill/handler/mount.go`'s `mountDeps`) so the core is tested offline. `Apply` is the production wiring; `applyHealDispel` is the tested core.

**Files:**
- Create: `skill/handler/healdispel/healdispel.go`
- Test: `skill/handler/healdispel/healdispel_test.go`
- Modify: `skill/handler/registrations/registrations.go`

**Interfaces:**
- Consumes: `channelhandler.Register`, `channelhandler.SelectAllCharactersInMap`, `channelhandler.PartyRecipient` (Task 4); `effect.Model.HP()/MP()/HpR()/MpR()` (Task 1); `buff.IsGmHidden` (Task 2) + `buff.Processor.CancelByTypes`/`GetByCharacterId` (Task 3); `character.Processor.GetById()/ChangeHP/ChangeMP`; `effective_stats.Processor.GetByCharacterId`; `socketHandler.AnnounceSkillUse`/`AnnounceForeignSkillUse`; `session.Processor.IfPresentByCharacterId`; `channelmap.Processor.ForOtherSessionsInMap`; `job.IsA`, `job.SuperGmId`.
- Produces: `func Apply(...)` matching `channelhandler.Handler`; package-level `diseaseTypes []string`; helpers `ratioAmount`, `clampRestore`, `effectiveMaxOrBase`, and the tested core `applyHealDispel(l, f, characterId, e, d healDispelDeps) error`.

- [ ] **Step 1: Write the failing test**

Create `skill/handler/healdispel/healdispel_test.go`:

```go
package healdispel

import (
	"io"
	"testing"

	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/sirupsen/logrus"
)

func tl() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func superGm(id uint32) character.Model {
	return character.NewModelBuilder().SetId(id).SetLevel(200).SetJobId(job.SuperGmId).MustBuild()
}

func recip(id uint32, hp, maxHp, mp, maxMp uint16) channelhandler.PartyRecipient {
	return channelhandler.NewPartyRecipientBuilder().
		SetId(id).SetHp(hp).SetMaxHp(maxHp).SetMp(mp).SetMaxMp(maxMp).Build()
}

type capture struct {
	hp        map[uint32]int16
	mp        map[uint32]int16
	dispelled map[uint32][]string
	selfCount int
	fgnCount  int
}

func newDeps(caster character.Model, casterErr error, hidden bool, recips []channelhandler.PartyRecipient, c *capture) healDispelDeps {
	c.hp = map[uint32]int16{}
	c.mp = map[uint32]int16{}
	c.dispelled = map[uint32][]string{}
	return healDispelDeps{
		loadCaster:  func(uint32) (character.Model, error) { return caster, casterErr },
		isGmHidden:  func(uint32) (bool, error) { return hidden, nil },
		selectInMap: func(field.Model) []channelhandler.PartyRecipient { return recips },
		// Effective max mirrors base for the test (no gear bonus).
		effectiveMax: func(_ field.Model, id uint32) (uint32, uint32, error) {
			for _, r := range recips {
				if r.Id() == id {
					return uint32(r.MaxHp()), uint32(r.MaxMp()), nil
				}
			}
			return 0, 0, nil
		},
		changeHP: func(_ field.Model, id uint32, amt int16) error { c.hp[id] = amt; return nil },
		changeMP: func(_ field.Model, id uint32, amt int16) error { c.mp[id] = amt; return nil },
		dispel:   func(_ field.Model, id uint32, types []string) error { c.dispelled[id] = types; return nil },
		announceSelf:    func(byte) error { c.selfCount++; return nil },
		announceForeign: func(byte) error { c.fgnCount++; return nil },
	}
}

func TestNonSuperGmRejected(t *testing.T) {
	nonGm := character.NewModelBuilder().SetId(1).SetJobId(job.Id(100)).MustBuild() // Warrior
	var cap capture
	d := newDeps(nonGm, nil, false, []channelhandler.PartyRecipient{recip(1, 1, 100, 1, 100)}, &cap)
	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, effect.RestModelToModel(t), d)
	if len(cap.hp) != 0 || len(cap.mp) != 0 || len(cap.dispelled) != 0 {
		t.Errorf("non-SuperGM caster produced effects: hp=%v mp=%v dispel=%v", cap.hp, cap.mp, cap.dispelled)
	}
}

func TestHealDispelAllRecipients(t *testing.T) {
	// hp=100 flat + 0.5*maxHp(1000)=500 -> 600 restore, clamped to headroom 900 -> 600.
	e := mustEffect(t, effect.RestModel{Hp: 100, Mp: 50, HPR: 0.5, MPR: 0.5})
	recips := []channelhandler.PartyRecipient{
		recip(1, 100, 1000, 100, 1000), // caster
		recip(2, 950, 1000, 990, 1000), // near-full: HP headroom 50, MP headroom 10
	}
	var cap capture
	d := newDeps(superGm(1), nil, false, recips, &cap)
	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, e, d)

	if cap.hp[1] != 600 {
		t.Errorf("recipient 1 HP delta = %d, want 600", cap.hp[1])
	}
	if cap.hp[2] != 50 { // clamped to headroom
		t.Errorf("recipient 2 HP delta = %d, want 50 (clamped)", cap.hp[2])
	}
	if cap.mp[2] != 10 { // 50 flat + 500 ratio -> clamped to headroom 10
		t.Errorf("recipient 2 MP delta = %d, want 10 (clamped)", cap.mp[2])
	}
	if len(cap.dispelled[1]) != 11 || len(cap.dispelled[2]) != 11 {
		t.Errorf("dispel types = %d/%d, want 11 each", len(cap.dispelled[1]), len(cap.dispelled[2]))
	}
	if cap.selfCount != 1 || cap.fgnCount != 1 {
		t.Errorf("announce self=%d foreign=%d, want 1/1 (visible caster)", cap.selfCount, cap.fgnCount)
	}
}

func TestForeignSuppressedWhenHidden(t *testing.T) {
	e := mustEffect(t, effect.RestModel{Hp: 10})
	var cap capture
	d := newDeps(superGm(1), nil, true, []channelhandler.PartyRecipient{recip(1, 1, 100, 1, 100)}, &cap)
	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, e, d)
	if cap.selfCount != 1 || cap.fgnCount != 0 {
		t.Errorf("announce self=%d foreign=%d, want 1/0 (hidden caster)", cap.selfCount, cap.fgnCount)
	}
}

func mustEffect(t *testing.T, rm effect.RestModel) effect.Model {
	t.Helper()
	m, err := effect.Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return m
}
```

*(Delete the stray `effect.RestModelToModel(t)` call — it was a typo; `TestNonSuperGmRejected` should use `mustEffect(t, effect.RestModel{Hp: 10})`. Fix it to:*

```go
	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, mustEffect(t, effect.RestModel{Hp: 10}), d)
```

*before running.)*

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skill/handler/healdispel/ -v`
Expected: FAIL — package/symbols undefined.

- [ ] **Step 3: Write the handler**

Create `skill/handler/healdispel/healdispel.go`:

```go
package healdispel

import (
	"context"
	"math"

	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	channelmap "atlas-channel/map"
	"atlas-channel/session"
	channelhandler "atlas-channel/skill/handler"
	socketHandler "atlas-channel/socket/handler"
	"atlas-channel/socket/writer"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

func init() {
	channelhandler.Register(skill2.SuperGmHealDispelId, Apply)
}

// diseaseTypes is the exact atlas-buffs disease set (buffs/character/immunity.go)
// that GM Heal + Dispel purges. Sourced from libs/atlas-constants (DOM-21).
var diseaseTypes = []string{
	string(charconst.TemporaryStatTypeStun),
	string(charconst.TemporaryStatTypePoison),
	string(charconst.TemporaryStatTypeSeal),
	string(charconst.TemporaryStatTypeDarkness),
	string(charconst.TemporaryStatTypeWeaken),
	string(charconst.TemporaryStatTypeCurse),
	string(charconst.TemporaryStatTypeSeduce),
	string(charconst.TemporaryStatTypeConfuse),
	string(charconst.TemporaryStatTypeUndead),
	string(charconst.TemporaryStatTypeSlow),
	string(charconst.TemporaryStatTypeStopPortion),
}

// healDispelDeps holds HealDispel's collaborators as function seams so the
// core loop is unit-testable offline (no Kafka/REST/session). announceSelf and
// announceForeign take the caster level so the wiring can build the skill-use
// packets without re-loading the caster.
type healDispelDeps struct {
	loadCaster      func(characterId uint32) (character.Model, error)
	isGmHidden      func(characterId uint32) (bool, error)
	selectInMap     func(f field.Model) []channelhandler.PartyRecipient
	effectiveMax    func(f field.Model, characterId uint32) (maxHp uint32, maxMp uint32, err error)
	changeHP        func(f field.Model, characterId uint32, amount int16) error
	changeMP        func(f field.Model, characterId uint32, amount int16) error
	dispel          func(f field.Model, characterId uint32, types []string) error
	announceSelf    func(level byte) error
	announceForeign func(level byte) error
}

// ratioAmount returns floor(max * ratio) as an int, or 0 for a non-positive ratio.
func ratioAmount(max uint16, ratio float64) int {
	if ratio <= 0 {
		return 0
	}
	return int(math.Floor(float64(max) * ratio))
}

// clampRestore turns a computed restore into the int16 delta to apply, clamped
// to [0, max-current] (never past the effective cap) and to the int16 ceiling.
func clampRestore(restore int, current uint16, max uint16) int16 {
	if restore <= 0 {
		return 0
	}
	headroom := int(max) - int(current)
	if headroom <= 0 {
		return 0
	}
	if restore > headroom {
		restore = headroom
	}
	if restore > math.MaxInt16 {
		restore = math.MaxInt16
	}
	return int16(restore)
}

// effectiveMaxOrBase narrows an effective-stats max (uint32) into uint16,
// falling back to the recipient's base max when the upstream returned zero or
// out-of-range. Mirrors the Cleric Heal clamp idiom.
func effectiveMaxOrBase(effective uint32, base uint16) uint16 {
	if effective == 0 {
		return base
	}
	if effective > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(effective)
}

// applyHealDispel is the tested core: gate, select recipients, restore HP/MP,
// dispel diseases, then broadcast. Per-recipient failures are logged and never
// abort the others. No experience is ever awarded (GM utility, not combat heal).
func applyHealDispel(l logrus.FieldLogger, f field.Model, characterId uint32, e effect.Model, d healDispelDeps) error {
	c, err := d.loadCaster(characterId)
	if err != nil {
		l.WithError(err).Errorf("Heal+Dispel: failed to load caster [%d].", characterId)
		return nil
	}
	if !job.IsA(c.JobId(), job.SuperGmId) {
		l.Warnf("Character [%d] cast SuperGM Heal+Dispel without SuperGM job; rejecting.", characterId)
		return nil
	}

	hidden, hErr := d.isGmHidden(characterId)
	if hErr != nil {
		l.WithError(hErr).Debugf("Heal+Dispel: unable to resolve hidden state for caster [%d]; treating as visible.", characterId)
		hidden = false
	}

	recipients := d.selectInMap(f)
	for _, r := range recipients {
		effMaxHpRaw, effMaxMpRaw, sErr := d.effectiveMax(f, r.Id())
		if sErr != nil {
			l.WithError(sErr).Debugf("Heal+Dispel: effective stats fetch failed for recipient [%d]; using base maxes.", r.Id())
		}
		maxHp := effectiveMaxOrBase(effMaxHpRaw, r.MaxHp())
		maxMp := effectiveMaxOrBase(effMaxMpRaw, r.MaxMp())

		if hpDelta := clampRestore(int(e.HP())+ratioAmount(maxHp, e.HpR()), r.Hp(), maxHp); hpDelta > 0 {
			if err := d.changeHP(f, r.Id(), hpDelta); err != nil {
				l.WithError(err).Errorf("Heal+Dispel: ChangeHP failed for recipient [%d].", r.Id())
			}
		}
		if mpDelta := clampRestore(int(e.MP())+ratioAmount(maxMp, e.MpR()), r.Mp(), maxMp); mpDelta > 0 {
			if err := d.changeMP(f, r.Id(), mpDelta); err != nil {
				l.WithError(err).Errorf("Heal+Dispel: ChangeMP failed for recipient [%d].", r.Id())
			}
		}
		if err := d.dispel(f, r.Id(), diseaseTypes); err != nil {
			l.WithError(err).Errorf("Heal+Dispel: dispel failed for recipient [%d].", r.Id())
		}
	}

	if err := d.announceSelf(c.Level()); err != nil {
		l.WithError(err).Debugf("Heal+Dispel: self skill-use announce failed for caster [%d].", characterId)
	}
	if !hidden {
		if err := d.announceForeign(c.Level()); err != nil {
			l.WithError(err).Debugf("Heal+Dispel: foreign skill-use announce failed for caster [%d].", characterId)
		}
	}
	return nil
}

// Apply is the registered Heal + Dispel handler. It builds production deps and
// delegates to applyHealDispel.
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
			cp := character.NewProcessor(l, ctx)
			bp := buff.NewProcessor(l, ctx)
			esp := effective_stats.NewProcessor(l, ctx)
			sp := session.NewProcessor(l, ctx)
			mp := channelmap.NewProcessor(l, ctx)

			d := healDispelDeps{
				loadCaster: func(id uint32) (character.Model, error) { return cp.GetById()(id) },
				isGmHidden: func(id uint32) (bool, error) {
					bs, err := bp.GetByCharacterId(id)
					if err != nil {
						return false, err
					}
					return buff.IsGmHidden(bs), nil
				},
				selectInMap: func(f field.Model) []channelhandler.PartyRecipient {
					return channelhandler.SelectAllCharactersInMap(l, ctx, f)
				},
				effectiveMax: func(f field.Model, id uint32) (uint32, uint32, error) {
					s, err := esp.GetByCharacterId(f.WorldId(), f.ChannelId(), id)
					return s.MaxHp, s.MaxMp, err
				},
				changeHP: cp.ChangeHP,
				changeMP: cp.ChangeMP,
				dispel:   func(f field.Model, id uint32, types []string) error { return bp.CancelByTypes(f, id, types) },
				announceSelf: func(level byte) error {
					return sp.IfPresentByCharacterId(f.Channel())(
						characterId,
						socketHandler.AnnounceSkillUse(l)(ctx)(wp)(info.SkillId(), level, info.SkillLevel()),
					)
				},
				announceForeign: func(level byte) error {
					return mp.ForOtherSessionsInMap(
						f, characterId,
						socketHandler.AnnounceForeignSkillUse(l)(ctx)(wp)(characterId, info.SkillId(), level, info.SkillLevel()),
					)
				},
			}
			return applyHealDispel(l, f, characterId, e, d)
		}
	}
}
```

- [ ] **Step 4: Fix the test typo and run**

Apply the `TestNonSuperGmRejected` fix noted in Step 1, then run: `go test ./skill/handler/healdispel/ -v`
Expected: PASS (all three tests). If `character.NewModelBuilder().MustBuild()` requires fields the fixtures omit, mirror the exact fields `skill/handler/heal`'s tests set.

- [ ] **Step 5: Register the handler**

In `skill/handler/registrations/registrations.go`, add the blank import (keep alphabetical grouping):

```go
import (
	_ "atlas-channel/skill/handler/heal"        // Cleric Heal — task 045
	_ "atlas-channel/skill/handler/healdispel"  // SuperGM Heal + Dispel — task-156
	_ "atlas-channel/skill/handler/mysticdoor"  // Priest Mystic Door — task-093
)
```

- [ ] **Step 6: Build the package + registrations**

Run: `go build ./skill/handler/... && go vet ./skill/handler/...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add skill/handler/healdispel/ skill/handler/registrations/registrations.go
git commit -m "feat(channel): SuperGM Heal + Dispel skill handler (task-156)"
```

---

## Task 6: Spawn suppression gate + exported broadcast helpers

Make the map consumer refuse to spawn a GM-hidden caster to other viewers, and expose two helpers the hide handler calls to synchronously despawn/spawn the caster in its current map. `spawnCharacterForSession` (`kafka/consumer/map/consumer.go:427`) is the single choke point for every character spawn (both `enterMap`→others at `:370` and SpawnForSelf-of-others at `:174`) and already fetches the spawned character's buff list at `:432` — the gate reuses it, so there is no extra cost and the check lives in the exact function that emits the spawn (race-safe per §8).

**Files:**
- Modify: `kafka/consumer/map/consumer.go`

**Interfaces:**
- Consumes: `buff.IsGmHidden` (Task 2); existing package-internal `spawnCharacterForSession`, `despawnForSession`, `_map.NewProcessor(...).ForOtherSessionsInMap`, `character.NewProcessor`, `guild.NewProcessor` (all already imported in this file).
- Produces:
  - `func SpawnCharacterInMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model, characterId uint32) error`
  - `func DespawnCharacterInMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model, characterId uint32) error`

- [ ] **Step 1: Insert the suppression gate**

In `kafka/consumer/map/consumer.go`, modify the inner closure of `spawnCharacterForSession` (currently lines ~431-437). Replace:

```go
				return func(s session.Model) error {
					bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(c.Id())
					if err != nil {
						bs = make([]buff.Model, 0)
					}

					return session.Announce(l)(ctx)(wp)(charpkt.CharacterSpawnWriter)(writer.CharacterSpawnBody(c, bs, g, enteringField))(s)
				}
```

with:

```go
				return func(s session.Model) error {
					bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(c.Id())
					if err != nil {
						bs = make([]buff.Model, 0)
					}

					// GM-hide suppression (task-156). A character hidden via the
					// SuperGM Hide skill must not be spawned to any OTHER viewer.
					// This is the single choke point for every character spawn —
					// enterMap->others and SpawnForSelf-of-others both pass here —
					// so a viewer entering while a GM is hidden never sees the
					// spawn (race-safe: the check is in the same path that emits
					// it). c is never the viewer's own character (both callers
					// skip k == s.CharacterId()), so self-view is never suppressed.
					if buff.IsGmHidden(bs) {
						return nil
					}

					return session.Announce(l)(ctx)(wp)(charpkt.CharacterSpawnWriter)(writer.CharacterSpawnBody(c, bs, g, enteringField))(s)
				}
```

- [ ] **Step 2: Add the exported broadcast helpers**

In the same file (place them next to `despawnForSession`, after `spawnCharacterForSession`), add:

```go
// DespawnCharacterInMap broadcasts a CharacterDespawn for characterId to every
// OTHER session in field f — the "hide on" half of the GM-hide toggle. Reuses
// the existing per-session despawn operator so the packet matches a normal exit.
func DespawnCharacterInMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model, characterId uint32) error {
	return func(f field.Model, characterId uint32) error {
		return _map.NewProcessor(l, ctx).ForOtherSessionsInMap(f, characterId, despawnForSession(l)(ctx)(wp)(characterId))
	}
}

// SpawnCharacterInMap broadcasts a CharacterSpawn for characterId to every OTHER
// session in field f — the "hide off" (reveal) half of the GM-hide toggle. It
// reuses spawnCharacterForSession so the spawn packet is byte-identical to a
// normal map-entry spawn (buffs + guild + enteringField=false, since the caster
// is already standing in the map). The suppression gate does NOT fire here
// because, by the time this runs, the hide buff has been cancelled.
func SpawnCharacterInMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model, characterId uint32) error {
	return func(f field.Model, characterId uint32) error {
		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator)(characterId)
		if err != nil {
			return err
		}
		g, _ := guild.NewProcessor(l, ctx).GetByMemberId(characterId)
		return _map.NewProcessor(l, ctx).ForOtherSessionsInMap(f, characterId, spawnCharacterForSession(l)(ctx)(wp)(c, g, false))
	}
}
```

*(Confirm the exact import alias for the map-constants processor in this file — the internal calls use `_map.NewProcessor`. Match whatever the file already uses; if the character-load decorators differ from `enterMap`'s call at `:352`, copy that call verbatim.)*

- [ ] **Step 3: Build + vet the consumer package**

Run: `go build ./kafka/consumer/map/... && go vet ./kafka/consumer/map/...`
Expected: clean. (No unit test is added for `spawnCharacterForSession`: it is a deeply-curried consumer closure with no existing test harness, and the decision logic is fully covered by Task 2's `IsGmHidden` tests. The gate itself is a one-line call to that tested predicate; its placement and the reveal packet are covered by the execute-time byte/behavior verification gate.)

- [ ] **Step 4: Commit**

```bash
git add kafka/consumer/map/consumer.go
git commit -m "feat(channel): GM-hide spawn suppression gate + reveal/despawn broadcast helpers (task-156)"
```

---

## Task 7: Hide handler

Register `SuperGmHideId` as a toggle: hide ON applies a `DARK_SIGHT` buff sourced from `SuperGmHideId` (`math.MaxInt32` duration) and synchronously despawns the caster from others; hide OFF cancels the buff and spawns the caster back. Self skill-use animation always fires; the foreign animation is never broadcast (see Global Constraints — resolves the design's §3.4 inconsistency). Modeled on `skill/handler/mount.go`; logic behind a `deps` struct for offline tests.

**Files:**
- Create: `skill/handler/hide/hide.go`
- Test: `skill/handler/hide/hide_test.go`
- Modify: `skill/handler/registrations/registrations.go`

**Interfaces:**
- Consumes: `channelhandler.Register`; `buff.Processor.Apply/Cancel/GetByCharacterId` + `buff.IsGmHidden` (Task 2); `statup.NewModel`; `_mapconsumer.SpawnCharacterInMap`/`DespawnCharacterInMap` (Task 6, package `atlas-channel/kafka/consumer/map`); `character.Processor.GetById`; `socketHandler.AnnounceSkillUse`; `session.Processor.IfPresentByCharacterId`; `job.IsA`, `job.SuperGmId`; `charconst.TemporaryStatTypeDarkSight`; `skill2.SuperGmHideId`.
- Produces: `func Apply(...)` matching `channelhandler.Handler`; `const HideBuffDuration = int32(math.MaxInt32)`; tested core `applyHide(l, f, characterId, info, d hideDeps) error`.

*(No import cycle: verified `kafka/consumer/map` does not import `skill/handler`, and nothing imports the new `skill/handler/hide` package except `registrations`.)*

- [ ] **Step 1: Write the failing test**

Create `skill/handler/hide/hide_test.go`:

```go
package hide

import (
	"io"
	"testing"

	"atlas-channel/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

func tl() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func superGm(id uint32) character.Model {
	return character.NewModelBuilder().SetId(id).SetLevel(200).SetJobId(job.SuperGmId).MustBuild()
}

type hideCapture struct {
	applied   int
	cancelled int
	despawned int
	spawned   int
	self      int
}

func deps(caster character.Model, hidden bool, c *hideCapture) hideDeps {
	return hideDeps{
		loadCaster:        func(uint32) (character.Model, error) { return caster, nil },
		isHidden:          func(uint32) (bool, error) { return hidden, nil },
		applyHide:         func(field.Model, uint32, byte) error { c.applied++; return nil },
		cancelHide:        func(field.Model, uint32) error { c.cancelled++; return nil },
		despawnFromOthers: func(field.Model, uint32) error { c.despawned++; return nil },
		spawnToOthers:     func(field.Model, uint32) error { c.spawned++; return nil },
		announceSelf:      func(byte) error { c.self++; return nil },
	}
}

func info() packetmodel.SkillUsageInfo { return packetmodel.SkillUsageInfo{} } // SkillLevel() -> 0 is fine

func TestNonSuperGmRejected(t *testing.T) {
	nonGm := character.NewModelBuilder().SetId(1).SetJobId(job.Id(100)).MustBuild()
	var c hideCapture
	_ = applyHide(tl(), field.NewBuilder(0, 0, 1).Build(), 1, info(), deps(nonGm, false, &c))
	if c.applied+c.cancelled+c.despawned+c.spawned != 0 {
		t.Errorf("non-SuperGM caster produced effects: %+v", c)
	}
}

func TestHideOn(t *testing.T) {
	var c hideCapture
	_ = applyHide(tl(), field.NewBuilder(0, 0, 1).Build(), 1, info(), deps(superGm(1), false, &c))
	if c.applied != 1 || c.despawned != 1 {
		t.Errorf("hide ON: applied=%d despawned=%d, want 1/1", c.applied, c.despawned)
	}
	if c.cancelled != 0 || c.spawned != 0 {
		t.Errorf("hide ON leaked cancel/spawn: %+v", c)
	}
	if c.self != 1 {
		t.Errorf("hide ON self-announce=%d, want 1", c.self)
	}
}

func TestHideOff(t *testing.T) {
	var c hideCapture
	_ = applyHide(tl(), field.NewBuilder(0, 0, 1).Build(), 1, info(), deps(superGm(1), true, &c))
	if c.cancelled != 1 || c.spawned != 1 {
		t.Errorf("hide OFF: cancelled=%d spawned=%d, want 1/1", c.cancelled, c.spawned)
	}
	if c.applied != 0 || c.despawned != 0 {
		t.Errorf("hide OFF leaked apply/despawn: %+v", c)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skill/handler/hide/ -v`
Expected: FAIL — package/symbols undefined.

- [ ] **Step 3: Write the handler**

Create `skill/handler/hide/hide.go`:

```go
package hide

import (
	"context"
	"math"

	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/data/skill/effect/statup"
	_mapconsumer "atlas-channel/kafka/consumer/map"
	"atlas-channel/session"
	channelhandler "atlas-channel/skill/handler"
	socketHandler "atlas-channel/socket/handler"
	"atlas-channel/socket/writer"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

// HideBuffDuration is the effectively-permanent duration for the GM-hide buff.
// atlas-buffs rejects duration <= 0, so the toggle uses the largest int32
// (~24.8 days); the canonical reveal is a re-cast, exactly like mounts.
const HideBuffDuration = int32(math.MaxInt32)

func init() {
	channelhandler.Register(skill2.SuperGmHideId, Apply)
}

// hideDeps holds the Hide toggle's collaborators as function seams so the
// direction logic is unit-testable offline. announceSelf takes the caster level
// so the wiring builds the skill-use packet without re-loading the caster.
// There is deliberately NO foreign-announce seam: the Hide skill never
// broadcasts a foreign skill-use animation (it would leak GM presence in both
// toggle directions — see task-156 plan Global Constraints).
type hideDeps struct {
	loadCaster        func(characterId uint32) (character.Model, error)
	isHidden          func(characterId uint32) (bool, error)
	applyHide         func(f field.Model, characterId uint32, level byte) error
	cancelHide        func(f field.Model, characterId uint32) error
	despawnFromOthers func(f field.Model, characterId uint32) error
	spawnToOthers     func(f field.Model, characterId uint32) error
	announceSelf      func(level byte) error
}

// applyHide is the tested core: gate SuperGM, read current hide state, then
// toggle. Hide ON applies the buff and despawns the caster from others; hide
// OFF cancels the buff and spawns the caster back. Self animation always fires;
// no foreign animation is broadcast.
func applyHide(l logrus.FieldLogger, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, d hideDeps) error {
	c, err := d.loadCaster(characterId)
	if err != nil {
		l.WithError(err).Errorf("Hide: failed to load caster [%d].", characterId)
		return nil
	}
	if !job.IsA(c.JobId(), job.SuperGmId) {
		l.Warnf("Character [%d] cast SuperGM Hide without SuperGM job; rejecting.", characterId)
		return nil
	}

	hidden, hErr := d.isHidden(characterId)
	if hErr != nil {
		l.WithError(hErr).Debugf("Hide: unable to resolve hide state for caster [%d]; treating as visible.", characterId)
		hidden = false
	}

	if !hidden {
		// Hide ON.
		if err := d.applyHide(f, characterId, info.SkillLevel()); err != nil {
			l.WithError(err).Errorf("Hide: failed to apply hide buff for caster [%d].", characterId)
		}
		if err := d.despawnFromOthers(f, characterId); err != nil {
			l.WithError(err).Errorf("Hide: failed to despawn caster [%d] from others.", characterId)
		}
	} else {
		// Hide OFF (reveal).
		if err := d.cancelHide(f, characterId); err != nil {
			l.WithError(err).Errorf("Hide: failed to cancel hide buff for caster [%d].", characterId)
		}
		if err := d.spawnToOthers(f, characterId); err != nil {
			l.WithError(err).Errorf("Hide: failed to spawn caster [%d] to others.", characterId)
		}
	}

	if err := d.announceSelf(c.Level()); err != nil {
		l.WithError(err).Debugf("Hide: self skill-use announce failed for caster [%d].", characterId)
	}
	return nil
}

// Apply is the registered Hide handler. It builds production deps and delegates
// to applyHide.
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
			cp := character.NewProcessor(l, ctx)
			bp := buff.NewProcessor(l, ctx)
			sp := session.NewProcessor(l, ctx)

			d := hideDeps{
				loadCaster: func(id uint32) (character.Model, error) { return cp.GetById()(id) },
				isHidden: func(id uint32) (bool, error) {
					bs, err := bp.GetByCharacterId(id)
					if err != nil {
						return false, err
					}
					return buff.IsGmHidden(bs), nil
				},
				applyHide: func(f field.Model, id uint32, level byte) error {
					// DARK_SIGHT amount must be non-zero: the v83 client's
					// CUser::IsDarkSight tests the stat != 0.
					statups := []statup.Model{statup.NewModel(string(charconst.TemporaryStatTypeDarkSight), 1)}
					return bp.Apply(f, id, int32(skill2.SuperGmHideId), level, HideBuffDuration, statups)(id)
				},
				cancelHide: func(f field.Model, id uint32) error {
					return bp.Cancel(f, id, int32(skill2.SuperGmHideId))
				},
				despawnFromOthers: func(f field.Model, id uint32) error {
					return _mapconsumer.DespawnCharacterInMap(l, ctx, wp)(f, id)
				},
				spawnToOthers: func(f field.Model, id uint32) error {
					return _mapconsumer.SpawnCharacterInMap(l, ctx, wp)(f, id)
				},
				announceSelf: func(level byte) error {
					return sp.IfPresentByCharacterId(f.Channel())(
						characterId,
						socketHandler.AnnounceSkillUse(l)(ctx)(wp)(info.SkillId(), level, info.SkillLevel()),
					)
				},
			}
			return applyHide(l, f, characterId, info, d)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skill/handler/hide/ -v`
Expected: PASS. If `packetmodel.SkillUsageInfo{}` cannot be constructed as a bare literal (unexported fields), replace `info()` in the test with the constructor the codebase uses (grep `packetmodel.SkillUsageInfo` / `NewSkillUsageInfo` in `socket/handler`), or use `channelhandler` fixtures — mirror how `skill/handler/heal`'s tests build it.

- [ ] **Step 5: Register the handler**

In `skill/handler/registrations/registrations.go`, add:

```go
	_ "atlas-channel/skill/handler/hide"        // SuperGM Hide — task-156
```

(keep the import block grouped/sorted; final block imports `heal`, `healdispel`, `hide`, `mysticdoor`).

- [ ] **Step 6: Build the whole service**

Run: `go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add skill/handler/hide/ skill/handler/registrations/registrations.go
git commit -m "feat(channel): SuperGM Hide toggle skill handler (task-156)"
```

---

## Task 8: Full-module verification

Run the mandatory CLAUDE.md gates for the one changed module (`atlas-channel`). Do NOT claim done until every command below is clean.

**Files:** none (verification only).

- [ ] **Step 1: Race tests across the module**

From `services/atlas-channel/atlas.com/channel`:
Run: `go test -race ./...`
Expected: ok / PASS across all packages. Fix any failure before proceeding.

- [ ] **Step 2: Vet**

Run: `go vet ./...`
Expected: no output.

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Docker bake (mandatory — go.mod touched)**

From the **worktree root** (`.worktrees/task-156-gm-hide-heal-dispel`):
Run: `docker buildx bake atlas-channel`
Expected: build succeeds. (Only `atlas-channel`'s `go.mod` region changed; this is the required container check.)

- [ ] **Step 5: Redis key guard**

From the worktree root:
Run: `tools/redis-key-guard.sh`
Expected: clean (no new raw keyed go-redis usage was added).

- [ ] **Step 6: Execute-time correctness gates (per CLAUDE.md — record evidence)**

- [ ] Verify the live `9101000` WZ recovery fields (`hp`/`mp`/`hpR`/`mpR`) against live WZ data; confirm the flat+ratio restore magnitude is correct for the actual values.
- [ ] Byte-verify the `CharacterSpawn`/`CharacterDespawn` packets on the hide/reveal path against source, and confirm the self `DARK_SIGHT` buff-give serializes the stat non-zero (so the v83 client's `CUser::IsDarkSight` reads it).

- [ ] **Step 7: Final commit (if any verification fixups were needed)**

```bash
git add -A
git commit -m "test(channel): verification fixups for SuperGM hide/heal-dispel (task-156)"
```

---

## Self-Review

**Spec coverage (design.md / prd.md → task):**

| Requirement | Task |
|---|---|
| FR-1/FR-2/FR-3 registration + UseSkill dispatch | Tasks 5, 7 (registrations + `channelhandler.Register`) |
| FR-4/FR-4.1 SuperGM gate (910 only) | Tasks 5, 7 (`job.IsA(c.JobId(), job.SuperGmId)`) |
| FR-5 map-wide recipients | Task 4 (`SelectAllCharactersInMap`) |
| FR-6/FR-6.1 HP+MP restore, clamped to effective max | Tasks 1, 5 (`MP/HpR/MpR` accessors + `clampRestore`/`effectiveMaxOrBase`) |
| FR-7/FR-8 dispel 11 diseases via CancelByTypes | Tasks 3, 5 (producer/processor + `diseaseTypes`) |
| FR-9 no experience | Task 5 (never calls `AwardExperience`) |
| FR-10 per-recipient failure isolation | Task 5 (log + continue per recipient) |
| FR-11/FR-12/FR-13 hide toggle + despawn/spawn | Tasks 6, 7 |
| FR-14/§8 persist across maps + race-safe suppression | Task 6 (`math.MaxInt32` buff + gate in `spawnCharacterForSession`) |
| FR-14.1 untargetable = not spawned | Task 6 (suppression gate) |
| FR-15 effect projection accessors | Task 1 |
| FR-16 isCategory1 no action | Design §1.2 — no task needed (fields populated unconditionally); noted, not implemented |
| FR-17 skill-use broadcast (suppress for hidden) | Task 5 (foreign gated on `!hidden`); Task 7 (foreign never broadcast — Global Constraints) |
| OQ-1 DARK_SIGHT / OQ-2 buff storage / OQ-3 / OQ-4 / OQ-5 | Resolved in design §2; encoded in Tasks 2, 6, 7 |

**Placeholder scan:** No `TBD`/`TODO`/"handle edge cases"/"similar to Task N". Two spots intentionally defer to code inspection (the `SkillUsageInfo` fixture constructor in Task 7 Step 4 and the character-model fixture fields in Task 5 Step 4) because those constructors are test-harness details best matched to the sibling `heal` tests at execute time; both give a concrete grep target and fallback, and neither is production code.

**Type consistency:** `PartyRecipient` getters `Mp()`/`MaxMp()` (Task 4) match the `r.Mp()`/`r.MaxMp()` reads in Task 5's fixtures and core. `buff.IsGmHidden([]Model) bool` (Task 2) is called identically in Tasks 5, 6, 7. `CancelByTypes(f, characterId, types)` signature is consistent across producer, processor, and handler. `SpawnCharacterInMap`/`DespawnCharacterInMap(l, ctx, wp)(f, characterId)` curry shape is consistent between Task 6's definitions and Task 7's `_mapconsumer.*` calls. `announceSelf(level byte) error` / `announceForeign(level byte) error` seams match their production closures and test fakes.
