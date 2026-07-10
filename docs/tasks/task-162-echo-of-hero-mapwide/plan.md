# Echo of Hero — Map-Wide Buff Application — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Casting any of the four X005 Echo of Hero skills (1005 / 10001005 / 20001005 / 20011005) buffs every eligible live-session character in the caster's field — caster included exactly once, dead characters and hidden GMs excluded — instead of only the caster.

**Architecture:** Only `services/atlas-channel` changes, two production files. The generic buff step in `UseSkill` (`skill/handler/common.go`) gains a routing branch: X005 casts fan out through a new map-wide recipient selector (`skill/handler/recipients.go`); all other skills keep the existing self + party-bitmap path byte-for-byte. Each recipient is buffed via the existing `buff.Processor.Apply` operator — no new packets, topics, or REST surface.

**Tech Stack:** Go, seam-variable test injection (package-level `var ...Func`), Builder-pattern test fixtures, logrus structured logging.

## Global Constraints

- `libs/atlas-packet` MUST NOT change — X005 stays out of `isPartyBuff` / `isMobAffectingBuff` / `isAntiRepeatBuffSkill` (design §1 closed FR-4/OQ-1 via IDA: an X005 cast carries no bitmap and no mob section).
- `services/atlas-data`, `services/atlas-buffs`, and all other services MUST NOT change (PRD §7).
- No new skill-id constants — all five ids already exist in `libs/atlas-constants/skill/constants.go`: `BeginnerEchoOfHeroId = Id(1005)` (line 2908), `SuperGmHideId = Id(9101004)` (line 3247), `NoblesseEchoOfHeroId = Id(10001005)` (line 3262), `LegendEchoOfHeroId = Id(20001005)` (line 3378), `EvanEchoOfHeroId = Id(20011005)` (line 3420) (DOM-21).
- No `*_testhelpers.go` files — tests use the existing seam-variable + Builder patterns in the `handler` package.
- Caster buffed exactly once (FR-1); per-recipient fetch/apply failure skips that recipient and never aborts the cast (FR-2.4); party membership, client bitmap, and LT/RB rectangles ignored for X005 recipient selection (FR-2.3).
- Verification gate before "done": `go test -race ./...`, `go vet ./...`, `go build ./...` clean in the atlas-channel module; `tools/redis-key-guard.sh` clean from the worktree root; `docker buildx bake atlas-channel` only if `go.mod` changed (not expected — no new module deps).
- Commit messages end with `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

## File Structure

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/recipients.go` (modify) | Add `loadBuffsFunc` seam, `MapWideSelectionStats`, `SelectMapWideRecipients` — sits beside the two existing party selectors, reuses `PartyRecipient` + Builder and the existing `inMapCharacterIdsFunc` / `loadPartyMemberFunc` seams. |
| `services/atlas-channel/atlas.com/channel/skill/handler/map_wide_recipients_test.go` (create) | Selector unit tests: caster included, dead / hidden / fetch-failure exclusions, deterministic order, stats counts. |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` (modify) | `isEchoOfHero` predicate, `applyToMap` fan-out + summary log, `applyBuffToRecipients` routing helper; `UseSkill`'s buff step (currently lines 107–111) delegates to it. |
| `services/atlas-channel/atlas.com/channel/skill/handler/common_echo_of_hero_test.go` (create) | Routing unit tests: each X005 id → map-wide (caster exactly once), non-X005 → legacy path (map-wide selector untouched), operator-error continuation. |

**Design-fidelity notes (intentional, small deviations from design.md §3 sketches):**
- The routing branch is extracted into `applyBuffToRecipients(l, ctx, f, characterId, info, applyBuffFunc)` rather than written inline in `UseSkill`, and `applyToMap` takes plain arguments (like `applyToMobs`) rather than the curried shape sketched in design §3.1. Behavior is identical; the named helper is what makes the routing testable with a recorded operator (design §6 "recorded apply operator") without emitting Kafka from tests.
- `MapWideSelectionStats` carries `inMap` (sweep size) instead of the design's `applied` — only the apply loop in `applyToMap` can count successful operator calls, and the summary log (design §3.4) needs both `in_map` and `applied`.

---

### Task 1: Map-wide recipient selector

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/recipients.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/map_wide_recipients_test.go` (create)

**Interfaces:**
- Consumes (all existing):
  - `inMapCharacterIdsFunc(l logrus.FieldLogger, ctx context.Context, f field.Model) map[uint32]struct{}` — seam at `recipients.go:57`
  - `loadPartyMemberFunc(l logrus.FieldLogger, ctx context.Context, memberId uint32) (character.Model, error)` — seam at `recipients.go:73`
  - `buff.NewProcessor(l, ctx).GetByCharacterId(characterId uint32) ([]buff.Model, error)` — `character/buff/processor.go:18`
  - `buff.Model.SourceId() int32` — `character/buff/model.go:31`; test constructor `buff.NewBuff(sourceId int32, level byte, duration int32, changes []stat.Model, createdAt, expiresAt time.Time) Model`
  - `NewPartyRecipientBuilder()` / `PartyRecipient` — `recipients.go:20-46`
  - `skill2.SuperGmHideId` (`Id(9101004)`), `skill2.Id` is `uint32`
- Produces (Task 2 relies on these exact signatures):
  - `var loadBuffsFunc func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]buff.Model, error)`
  - `type MapWideSelectionStats struct { inMap, skippedDead, skippedHidden, fetchFailures int }`
  - `func SelectMapWideRecipients(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32) ([]PartyRecipient, MapWideSelectionStats)`

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/map_wide_recipients_test.go`. Helpers `testLogger()`, `mkField()`, `mkMemberChar()`, `recipientIds()`, `eqIds()`, and the `testCasterId`/`testMemberA`/`testMemberB` constants already exist in this package (`recipients_test.go`, `common_apply_to_mobs_test.go`) — reuse them, do not redefine.

```go
package handler

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"atlas-channel/character"
	"atlas-channel/character/buff"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/sirupsen/logrus"
)

// installMapWideSeams replaces the three external-lookup seams used by the
// map-wide selector with deterministic in-memory implementations.
func installMapWideSeams(t *testing.T, inMap map[uint32]struct{}, chars map[uint32]character.Model, buffs map[uint32][]buff.Model, buffErrs map[uint32]error) {
	t.Helper()
	prevInMap := inMapCharacterIdsFunc
	prevMember := loadPartyMemberFunc
	prevBuffs := loadBuffsFunc

	inMapCharacterIdsFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model) map[uint32]struct{} {
		return inMap
	}
	loadPartyMemberFunc = func(_ logrus.FieldLogger, _ context.Context, memberId uint32) (character.Model, error) {
		mc, ok := chars[memberId]
		if !ok {
			return character.Model{}, errors.New("character not found")
		}
		return mc, nil
	}
	loadBuffsFunc = func(_ logrus.FieldLogger, _ context.Context, characterId uint32) ([]buff.Model, error) {
		if err, ok := buffErrs[characterId]; ok {
			return nil, err
		}
		return buffs[characterId], nil
	}

	t.Cleanup(func() {
		inMapCharacterIdsFunc = prevInMap
		loadPartyMemberFunc = prevMember
		loadBuffsFunc = prevBuffs
	})
}

// mkHideBuff models the task-156 hide state: a buff sourced from SuperGmHide.
func mkHideBuff() buff.Model {
	return buff.NewBuff(int32(skill2.SuperGmHideId), 1, math.MaxInt32, nil, time.Now(), time.Now().Add(time.Hour))
}

// mkOtherBuff is any non-hide buff — its presence must not exclude a recipient.
func mkOtherBuff() buff.Model {
	return buff.NewBuff(int32(skill2.BeginnerEchoOfHeroId), 1, 60000, nil, time.Now(), time.Now().Add(time.Hour))
}

func TestSelectMapWideRecipients_IncludesCasterAndAllLiveCharacters(t *testing.T) {
	installMapWideSeams(t,
		map[uint32]struct{}{testCasterId: {}, testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testCasterId: mkMemberChar(testCasterId, 500),
			testMemberA:  mkMemberChar(testMemberA, 500),
			testMemberB:  mkMemberChar(testMemberB, 500),
		},
		nil, nil,
	)

	got, stats := SelectMapWideRecipients(testLogger(), context.Background(), mkField(), testCasterId)
	if want := []uint32{testCasterId, testMemberA, testMemberB}; !eqIds(recipientIds(got), want) {
		t.Fatalf("got %v, want %v", recipientIds(got), want)
	}
	if stats.inMap != 3 || stats.skippedDead != 0 || stats.skippedHidden != 0 || stats.fetchFailures != 0 {
		t.Fatalf("stats = %+v, want inMap=3 and zero skips", stats)
	}
}

func TestSelectMapWideRecipients_ReturnsIdsInAscendingOrder(t *testing.T) {
	installMapWideSeams(t,
		map[uint32]struct{}{30: {}, 10: {}, 20: {}},
		map[uint32]character.Model{
			10: mkMemberChar(10, 500),
			20: mkMemberChar(20, 500),
			30: mkMemberChar(30, 500),
		},
		nil, nil,
	)

	got, _ := SelectMapWideRecipients(testLogger(), context.Background(), mkField(), 10)
	for i, want := range []uint32{10, 20, 30} {
		if got[i].Id() != want {
			t.Fatalf("recipient[%d] = %d, want %d (ascending order)", i, got[i].Id(), want)
		}
	}
}

func TestSelectMapWideRecipients_ExcludesDead(t *testing.T) {
	installMapWideSeams(t,
		map[uint32]struct{}{testCasterId: {}, testMemberA: {}},
		map[uint32]character.Model{
			testCasterId: mkMemberChar(testCasterId, 500),
			testMemberA:  mkMemberChar(testMemberA, 0), // dead
		},
		nil, nil,
	)

	got, stats := SelectMapWideRecipients(testLogger(), context.Background(), mkField(), testCasterId)
	if want := []uint32{testCasterId}; !eqIds(recipientIds(got), want) {
		t.Fatalf("got %v, want %v", recipientIds(got), want)
	}
	if stats.skippedDead != 1 {
		t.Fatalf("skippedDead = %d, want 1", stats.skippedDead)
	}
}

func TestSelectMapWideRecipients_ExcludesHiddenGm(t *testing.T) {
	installMapWideSeams(t,
		map[uint32]struct{}{testCasterId: {}, testMemberA: {}},
		map[uint32]character.Model{
			testCasterId: mkMemberChar(testCasterId, 500),
			testMemberA:  mkMemberChar(testMemberA, 500),
		},
		map[uint32][]buff.Model{
			testMemberA: {mkHideBuff()},
		},
		nil,
	)

	got, stats := SelectMapWideRecipients(testLogger(), context.Background(), mkField(), testCasterId)
	if want := []uint32{testCasterId}; !eqIds(recipientIds(got), want) {
		t.Fatalf("got %v, want %v", recipientIds(got), want)
	}
	if stats.skippedHidden != 1 {
		t.Fatalf("skippedHidden = %d, want 1", stats.skippedHidden)
	}
}

func TestSelectMapWideRecipients_NonHideBuffDoesNotExclude(t *testing.T) {
	installMapWideSeams(t,
		map[uint32]struct{}{testCasterId: {}, testMemberA: {}},
		map[uint32]character.Model{
			testCasterId: mkMemberChar(testCasterId, 500),
			testMemberA:  mkMemberChar(testMemberA, 500),
		},
		map[uint32][]buff.Model{
			testMemberA: {mkOtherBuff()},
		},
		nil,
	)

	got, _ := SelectMapWideRecipients(testLogger(), context.Background(), mkField(), testCasterId)
	if want := []uint32{testCasterId, testMemberA}; !eqIds(recipientIds(got), want) {
		t.Fatalf("got %v, want %v", recipientIds(got), want)
	}
}

func TestSelectMapWideRecipients_CharacterFetchFailureSkipsOnlyThatRecipient(t *testing.T) {
	installMapWideSeams(t,
		map[uint32]struct{}{testCasterId: {}, testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			// testMemberA intentionally absent -> fetch error
			testCasterId: mkMemberChar(testCasterId, 500),
			testMemberB:  mkMemberChar(testMemberB, 500),
		},
		nil, nil,
	)

	got, stats := SelectMapWideRecipients(testLogger(), context.Background(), mkField(), testCasterId)
	if want := []uint32{testCasterId, testMemberB}; !eqIds(recipientIds(got), want) {
		t.Fatalf("got %v, want %v", recipientIds(got), want)
	}
	if stats.fetchFailures != 1 {
		t.Fatalf("fetchFailures = %d, want 1", stats.fetchFailures)
	}
}

func TestSelectMapWideRecipients_BuffFetchFailureSkipsOnlyThatRecipient(t *testing.T) {
	installMapWideSeams(t,
		map[uint32]struct{}{testCasterId: {}, testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testCasterId: mkMemberChar(testCasterId, 500),
			testMemberA:  mkMemberChar(testMemberA, 500),
			testMemberB:  mkMemberChar(testMemberB, 500),
		},
		nil,
		map[uint32]error{testMemberA: errors.New("buff service down")},
	)

	got, stats := SelectMapWideRecipients(testLogger(), context.Background(), mkField(), testCasterId)
	if want := []uint32{testCasterId, testMemberB}; !eqIds(recipientIds(got), want) {
		t.Fatalf("got %v, want %v", recipientIds(got), want)
	}
	if stats.fetchFailures != 1 {
		t.Fatalf("fetchFailures = %d, want 1", stats.fetchFailures)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run TestSelectMapWideRecipients -v`
Expected: FAIL to build with `undefined: loadBuffsFunc` and `undefined: SelectMapWideRecipients`.

- [ ] **Step 3: Implement the selector**

In `services/atlas-channel/atlas.com/channel/skill/handler/recipients.go`:

Add to the import block: `"sort"`, `"atlas-channel/character/buff"`, and `skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"` (the `skill2` alias matches `common.go`'s convention). The block becomes:

```go
import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	_map "atlas-channel/map"
	"atlas-channel/party"
	"atlas-channel/session"
	"context"
	"sort"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/sirupsen/logrus"
)
```

Append at the end of the file:

```go
// loadBuffsFunc is the recipient-buff-list seam tests can replace. The
// map-wide selector uses it to detect hidden GMs — task-156 models hide as
// a buff with SourceId == SuperGmHideId (9101004). Until task-156 lands no
// such buff exists, so the exclusion is vacuously true (FR-2.2).
var loadBuffsFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]buff.Model, error) {
	return buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
}

// MapWideSelectionStats summarizes one map-wide recipient selection for the
// echo_of_hero_apply_summary log line.
type MapWideSelectionStats struct {
	inMap         int
	skippedDead   int
	skippedHidden int
	fetchFailures int
}

// SelectMapWideRecipients returns every character with a live session in the
// caster's field — INCLUDING the caster — excluding dead characters (Hp 0)
// and hidden GMs (any active buff sourced from SuperGmHide). Party
// membership, the client bitmap, and LT/RB rectangles are all ignored
// (FR-2.3); the field-scoped session sweep already bounds recipients to the
// caster's world/channel/map/instance and therefore tenant. A per-recipient
// fetch failure skips that recipient and continues (FR-2.4). Ids are
// processed in ascending order for deterministic logs and tests.
func SelectMapWideRecipients(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32) ([]PartyRecipient, MapWideSelectionStats) {
	inMap := inMapCharacterIdsFunc(l, ctx, f)
	ids := make([]uint32, 0, len(inMap))
	for id := range inMap {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	stats := MapWideSelectionStats{inMap: len(ids)}
	out := make([]PartyRecipient, 0, len(ids))
	for _, id := range ids {
		c, cErr := loadPartyMemberFunc(l, ctx, id)
		if cErr != nil {
			stats.fetchFailures++
			l.WithError(cErr).Debugf("Skipping map-wide recipient [%d] for caster [%d]: character fetch failed.", id, casterId)
			continue
		}
		if c.Hp() == 0 {
			stats.skippedDead++
			continue
		}
		buffs, bErr := loadBuffsFunc(l, ctx, id)
		if bErr != nil {
			stats.fetchFailures++
			l.WithError(bErr).Debugf("Skipping map-wide recipient [%d] for caster [%d]: buff fetch failed.", id, casterId)
			continue
		}
		hidden := false
		for _, b := range buffs {
			if b.SourceId() == int32(skill2.SuperGmHideId) {
				hidden = true
				break
			}
		}
		if hidden {
			stats.skippedHidden++
			continue
		}
		out = append(out, NewPartyRecipientBuilder().
			SetId(c.Id()).
			SetX(c.X()).
			SetY(c.Y()).
			SetHp(c.Hp()).
			SetMaxHp(c.MaxHp()).
			Build())
	}
	return out, stats
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run TestSelectMapWideRecipients -v`
Expected: PASS (7 tests).

- [ ] **Step 5: Run the whole handler package to catch regressions**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/handler/`
Expected: ok (existing party selector tests unchanged and passing).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/recipients.go services/atlas-channel/atlas.com/channel/skill/handler/map_wide_recipients_test.go
git commit -m "feat(channel): map-wide recipient selector for Echo of Hero (task-162)

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: Echo of Hero routing branch in UseSkill

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go` (buff step currently at lines 107–111; new helpers appended after `applyToParty`)
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/common_echo_of_hero_test.go` (create)

**Interfaces:**
- Consumes:
  - `SelectMapWideRecipients(l, ctx, f, casterId) ([]PartyRecipient, MapWideSelectionStats)` and `loadBuffsFunc` — from Task 1
  - `applyToParty(l)(ctx)(f, casterId, memberBitmap)(idOperator)` — existing, `common.go:342`
  - `model2.Operator[uint32]` = `func(uint32) error` — `libs/atlas-model/model/processor.go:44`
  - `packetmodel.SkillUsageInfo` accessors `SkillId() uint32`, `SkillLevel() byte`, `AffectedPartyMemberBitmap() byte`; test builder `packetmodel.NewSkillUsageInfoBuilder()`
  - `skill2.Is(skillId, refs...)` — `libs/atlas-constants/skill/model.go:76`
  - Existing test fixtures: `installPartySeams`, `threePersonParty`, `mkPartyMember`, `mkMemberChar`, `mkField`, `testLogger`, `testCasterId`/`testMemberA`/`testMemberB`
  - `installMapWideSeams(t *testing.T, inMap map[uint32]struct{}, chars map[uint32]character.Model, buffs map[uint32][]buff.Model, buffErrs map[uint32]error)` — test helper from Task 1's `map_wide_recipients_test.go` (same package)
- Produces:
  - `func isEchoOfHero(skillId skill2.Id) bool`
  - `func applyBuffToRecipients(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, applyBuffFunc model2.Operator[uint32])`
  - `func applyToMap(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, info packetmodel.SkillUsageInfo, idOperator model2.Operator[uint32])`

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/common_echo_of_hero_test.go`:

```go
package handler

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"atlas-channel/character"
	"atlas-channel/character/buff"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

// FR-1: every X005 variant routes map-wide; the caster is enumerated by the
// map sweep and buffed exactly once (no separate self-apply).
func TestApplyBuffToRecipients_EchoOfHeroRoutesMapWide(t *testing.T) {
	for _, sid := range []skill2.Id{
		skill2.BeginnerEchoOfHeroId,
		skill2.NoblesseEchoOfHeroId,
		skill2.LegendEchoOfHeroId,
		skill2.EvanEchoOfHeroId,
	} {
		t.Run(fmt.Sprintf("skill_%d", uint32(sid)), func(t *testing.T) {
			installMapWideSeams(t,
				map[uint32]struct{}{testCasterId: {}, testMemberA: {}},
				map[uint32]character.Model{
					testCasterId: mkMemberChar(testCasterId, 500),
					testMemberA:  mkMemberChar(testMemberA, 500),
				},
				nil, nil,
			)

			calls := map[uint32]int{}
			op := func(id uint32) error { calls[id]++; return nil }

			info := packetmodel.NewSkillUsageInfoBuilder().SetSkillId(uint32(sid)).SetSkillLevel(1).Build()
			applyBuffToRecipients(testLogger(), context.Background(), mkField(), testCasterId, info, op)

			if calls[testCasterId] != 1 {
				t.Fatalf("caster applied %d times, want exactly 1 (FR-1)", calls[testCasterId])
			}
			if calls[testMemberA] != 1 {
				t.Fatalf("member applied %d times, want exactly 1", calls[testMemberA])
			}
			if len(calls) != 2 {
				t.Fatalf("got %d distinct recipients, want 2", len(calls))
			}
		})
	}
}

// FR-1.1: non-X005 skills keep the legacy self + party-bitmap path and never
// touch the map-wide selector.
func TestApplyBuffToRecipients_NonEchoUsesLegacyPartyPath(t *testing.T) {
	a := mkPartyMember(testMemberA, true, channel.Id(0), _map.Id(40000))
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(40000))
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testMemberA: mkMemberChar(testMemberA, 500),
			testMemberB: mkMemberChar(testMemberB, 500),
		},
	)

	// Guard: the map-wide selector must not run for non-echo skills. The
	// selector is the only caller of loadBuffsFunc, so tripping it here
	// means the routing took the wrong branch.
	prevBuffs := loadBuffsFunc
	buffsCalled := false
	loadBuffsFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) ([]buff.Model, error) {
		buffsCalled = true
		return nil, nil
	}
	t.Cleanup(func() { loadBuffsFunc = prevBuffs })

	calls := map[uint32]int{}
	op := func(id uint32) error { calls[id]++; return nil }

	// 1301007 (Spearman Hyper Body) — any non-X005 id exercises the legacy
	// path; bitmap selects members A (bit 4) and B (bit 3).
	info := packetmodel.NewSkillUsageInfoBuilder().SetSkillId(1301007).SetSkillLevel(30).SetAffectedPartyMemberBitmap(0b11000).Build()
	applyBuffToRecipients(testLogger(), context.Background(), mkField(), testCasterId, info, op)

	if buffsCalled {
		t.Fatal("map-wide selector ran for a non-echo skill (FR-1.1)")
	}
	for _, id := range []uint32{testCasterId, testMemberA, testMemberB} {
		if calls[id] != 1 {
			t.Fatalf("recipient %d applied %d times, want 1", id, calls[id])
		}
	}
	if len(calls) != 3 {
		t.Fatalf("got %d distinct recipients, want 3", len(calls))
	}
}

// FR-2.4: a per-recipient apply failure skips that recipient and continues.
func TestApplyToMap_OperatorErrorContinues(t *testing.T) {
	installMapWideSeams(t,
		map[uint32]struct{}{testCasterId: {}, testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testCasterId: mkMemberChar(testCasterId, 500),
			testMemberA:  mkMemberChar(testMemberA, 500),
			testMemberB:  mkMemberChar(testMemberB, 500),
		},
		nil, nil,
	)

	calls := map[uint32]int{}
	op := func(id uint32) error {
		calls[id]++
		if id == testMemberA {
			return errors.New("buff service rejected")
		}
		return nil
	}

	info := packetmodel.NewSkillUsageInfoBuilder().SetSkillId(uint32(skill2.BeginnerEchoOfHeroId)).SetSkillLevel(1).Build()
	applyToMap(testLogger(), context.Background(), mkField(), testCasterId, info, op)

	for _, id := range []uint32{testCasterId, testMemberA, testMemberB} {
		if calls[id] != 1 {
			t.Fatalf("recipient %d attempted %d times, want 1", id, calls[id])
		}
	}
}

func TestIsEchoOfHero(t *testing.T) {
	for _, sid := range []skill2.Id{
		skill2.BeginnerEchoOfHeroId,
		skill2.NoblesseEchoOfHeroId,
		skill2.LegendEchoOfHeroId,
		skill2.EvanEchoOfHeroId,
	} {
		if !isEchoOfHero(sid) {
			t.Fatalf("isEchoOfHero(%d) = false, want true", uint32(sid))
		}
	}
	for _, sid := range []skill2.Id{skill2.SuperGmHideId, skill2.Id(1301007), skill2.Id(0)} {
		if isEchoOfHero(sid) {
			t.Fatalf("isEchoOfHero(%d) = true, want false", uint32(sid))
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run 'TestApplyBuffToRecipients|TestApplyToMap|TestIsEchoOfHero' -v`
Expected: FAIL to build with `undefined: applyBuffToRecipients`, `undefined: applyToMap`, `undefined: isEchoOfHero`.

- [ ] **Step 3: Implement the routing branch**

In `services/atlas-channel/atlas.com/channel/skill/handler/common.go`:

3a. Replace the buff step inside `UseSkill` (currently lines 107–111):

```go
			if e.Duration() > 0 && len(e.StatUps()) > 0 {
				applyBuffFunc := buff.NewProcessor(l, ctx).Apply(f, characterId, int32(info.SkillId()), info.SkillLevel(), e.Duration(), e.StatUps())
				_ = applyBuffFunc(characterId)
				_ = applyToParty(l)(ctx)(f, characterId, info.AffectedPartyMemberBitmap())(applyBuffFunc)
			}
```

with:

```go
			if e.Duration() > 0 && len(e.StatUps()) > 0 {
				applyBuffFunc := buff.NewProcessor(l, ctx).Apply(f, characterId, int32(info.SkillId()), info.SkillLevel(), e.Duration(), e.StatUps())
				applyBuffToRecipients(l, ctx, f, characterId, info, applyBuffFunc)
			}
```

Everything else in `UseSkill` (HP/MP/item consume, cooldown, mount short-circuit, `applyToMobs`, registry dispatch) is untouched (FR-1.2; X005 has no `AffectedMobIds` and no registered handler, so both no-op for it).

3b. Append after `applyToParty` at the end of the file:

```go
// isEchoOfHero reports whether skillId is one of the four X005 Echo of Hero
// variants (Beginner/Noblesse/Legend/Evan), which buff every eligible
// character in the caster's field rather than self + party (FR-1).
func isEchoOfHero(skillId skill2.Id) bool {
	return skill2.Is(skillId,
		skill2.BeginnerEchoOfHeroId,
		skill2.NoblesseEchoOfHeroId,
		skill2.LegendEchoOfHeroId,
		skill2.EvanEchoOfHeroId,
	)
}

// applyBuffToRecipients routes the generic buff step to its recipient set:
// Echo of Hero fans out map-wide via applyToMap — the caster is enumerated
// by the map sweep like everyone else, so there is deliberately NO separate
// self-apply on that branch (FR-1: caster buffed exactly once). Every other
// skill keeps the legacy self + party-bitmap path (FR-1.1).
func applyBuffToRecipients(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, applyBuffFunc model2.Operator[uint32]) {
	if isEchoOfHero(skill2.Id(info.SkillId())) {
		applyToMap(l, ctx, f, characterId, info, applyBuffFunc)
		return
	}
	_ = applyBuffFunc(characterId)
	_ = applyToParty(l)(ctx)(f, characterId, info.AffectedPartyMemberBitmap())(applyBuffFunc)
}

// applyToMap applies idOperator to every map-wide recipient selected by
// SelectMapWideRecipients (FR-2). A per-recipient apply failure skips that
// recipient and continues (FR-2.4). One summary line is logged per cast.
func applyToMap(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, info packetmodel.SkillUsageInfo, idOperator model2.Operator[uint32]) {
	recipients, stats := SelectMapWideRecipients(l, ctx, f, casterId)
	applied := 0
	for _, r := range recipients {
		if err := idOperator(r.Id()); err != nil {
			l.WithError(err).Debugf("Echo of Hero apply failed for recipient [%d]; continuing.", r.Id())
			continue
		}
		applied++
	}
	l.WithFields(logrus.Fields{
		"caster":         casterId,
		"skill_id":       info.SkillId(),
		"skill_level":    info.SkillLevel(),
		"in_map":         stats.inMap,
		"applied":        applied,
		"skipped_dead":   stats.skippedDead,
		"skipped_hidden": stats.skippedHidden,
		"fetch_failures": stats.fetchFailures,
	}).Debug("echo_of_hero_apply_summary")
}
```

No import changes are needed in `common.go` — `skill2`, `model2`, `packetmodel`, `field`, `logrus`, `buff`, and `context` are already imported.

- [ ] **Step 4: Run the new tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run 'TestApplyBuffToRecipients|TestApplyToMap|TestIsEchoOfHero' -v`
Expected: PASS (4 top-level tests, 4 subtests under the echo routing test).

- [ ] **Step 5: Run the whole handler package with race detector**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/handler/`
Expected: ok — including all pre-existing party selector tests (`recipients_test.go`), which guard FR-1.1 / AC-6.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/common.go services/atlas-channel/atlas.com/channel/skill/handler/common_echo_of_hero_test.go
git commit -m "feat(channel): route X005 Echo of Hero casts map-wide (task-162)

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: Full verification gate

**Files:** none created or modified — verification only. All commands run from the worktree root (`.worktrees/task-162-echo-of-hero-mapwide`) except where a `cd` is shown.

**Interfaces:**
- Consumes: the committed output of Tasks 1–2.
- Produces: a verified branch ready for code review (`superpowers:requesting-code-review` is the next phase step per CLAUDE.md — run it before any PR).

- [ ] **Step 1: Full atlas-channel test suite with race detector**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./...`
Expected: all packages `ok` (or cached), zero failures.

- [ ] **Step 2: Vet**

Run: `cd services/atlas-channel/atlas.com/channel && go vet ./...`
Expected: no output, exit 0.

- [ ] **Step 3: Build**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: no output, exit 0.

- [ ] **Step 4: Redis key guard**

Run from the worktree root (no `GOWORK=off` prefix — it causes false FAILs): `tools/redis-key-guard.sh`
Expected: clean pass, exit 0.

- [ ] **Step 5: Confirm the diff is scoped to atlas-channel (AC-7)**

Run: `git diff --name-only main...HEAD -- libs/ services/atlas-data services/atlas-buffs`
Expected: empty output — `libs/atlas-packet` (FR-4), `atlas-data`, and `atlas-buffs` untouched.

Run: `git diff --name-only main...HEAD`
Expected: exactly the two production files, the two test files, and `docs/tasks/task-162-echo-of-hero-mapwide/*`.

- [ ] **Step 6: Docker bake — only if go.mod changed**

Run: `git diff --name-only main...HEAD -- services/atlas-channel/atlas.com/channel/go.mod`
Expected: empty (no new module deps). If empty, `docker buildx bake atlas-channel` is NOT required per CLAUDE.md (bake is mandatory only for services whose `go.mod` was touched). If non-empty, run `docker buildx bake atlas-channel` from the worktree root and require success.

- [ ] **Step 7: Acceptance-criteria sweep**

Confirm each PRD §10 criterion maps to evidence:
- Map-wide application incl. caster once → `TestApplyBuffToRecipients_EchoOfHeroRoutesMapWide` (all four ids)
- Dead excluded → `TestSelectMapWideRecipients_ExcludesDead`
- Hidden GM (SourceId 9101004) excluded → `TestSelectMapWideRecipients_ExcludesHiddenGm`
- Other-map exclusion → by construction (field-scoped `inMapCharacterIdsFunc`); covered by the stubbed in-map set omitting outsiders
- Non-X005 unchanged → `TestApplyBuffToRecipients_NonEchoUsesLegacyPartyPath` + pre-existing `recipients_test.go` suite passing
- Fetch-failure skip → `TestSelectMapWideRecipients_CharacterFetchFailureSkipsOnlyThatRecipient`, `..._BuffFetchFailureSkipsOnlyThatRecipient`, `TestApplyToMap_OperatorErrorContinues`
- `libs/atlas-packet` diff empty → Step 5
- Gates clean → Steps 1–4

No commit in this task (nothing changed). Report the branch as ready for the code-review phase.
