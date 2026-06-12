# Mount / Monster-Rider System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Read `context.md` in this folder first — it carries the verified signatures, IDA wire facts, and the game-data values pinned by Task 1.

**Goal:** Deliver tamed-monster and skill-only mounts end-to-end — cast → ride (rendered for self + observers), persistent level/exp/tiredness, 60s tiredness tick, revitalizer feeding, dismount on toggle/job-change, plus the Riding Mimiana skill-acquisition questline.

**Architecture:** Mounting rides the existing `MONSTER_RIDING` character-temporary-stat buff (atlas-buffs + atlas-channel render). A new `atlas-mounts` service (cloned from `atlas-pets`) owns persistence, the active-mount registry, the tiredness ticker, and feed math. A `libs/atlas-packet` fix makes the buff carry the real vehicle id + skill id; a new `SET_TAMING_MOB_INFO` writer broadcasts mount info. Feeding arrives on dedicated opcode 0x4D → atlas-consumables → atlas-mounts.

**Tech Stack:** Go microservices, GORM + Postgres, Kafka (atlas-kafka), Redis via `libs/atlas-redis`, JSON:API REST, `libs/atlas-constants`, `libs/atlas-packet`. Verification via `go test -race`, `go vet`, `go build`, `docker buildx bake`, `tools/redis-key-guard.sh`.

**Worktree:** the task worktree (`<worktree-root>` = the `task-086-mount-system` worktree root) on branch `task-086-mount-system`. Every subagent must `cd` into this worktree first and verify branch after each commit. All relative paths below are from `<worktree-root>`.

---

## Phase ordering & parallelism

Tasks are grouped into phases. **Task 1 (Pin game data) must complete first** — Tasks 7, 12, 17, 25 consume its outputs. After Task 1, these phases are largely independent and may be parallelized:

- **Phase A** (Tasks 2–6): `libs/atlas-packet` — packet fix + new writer. *Acceptance-critical, no deps.*
- **Phase B** (Tasks 7–8): `atlas-data` reader extensions.
- **Phase C** (Tasks 9–24): `atlas-mounts` new service (depends on nothing external; Kafka shapes only).
- **Phase D** (Tasks 25–31): `atlas-channel` — mount toggle, food handler, writer + mount-status consumer (depends on A for the writer const + B for skill-only vehicle ids).
- **Phase E** (Tasks 32–35): `atlas-consumables` — food command + TamingMobFed event.
- **Phase F** (Tasks 36–38): Riding Mimiana questline (data-only; independent).
- **Phase G** (Tasks 39–42): build/deploy wiring + live-config + full verification.

Per the project convention each Go change is TDD: write failing test → run (fail) → implement → run (pass) → commit.

---

## Task 1: Pin game-data values (verify-over-memory gate)

**Files:**
- Modify: `docs/tasks/task-086-mount-system/context.md` (§8)

This task produces no code — it resolves the values downstream tasks consume. **Do not guess from memory.** Use these sources and record what you read.

- [x] **Step 1: Pin mount skill-id / vehicle-id set (OQ 9.6).**

Read `libs/atlas-constants/skill/constants.go` and confirm the band-offset pattern
(beginner `100x`, Noblesse `1000100x`, Legend `2000100x`, Evan `2001100x`). For each
skill-only mount — SpaceShip 1013, Yeti1 1017, Yeti2 1018, Broomstick 1019, Balrog 1031 —
derive the Noblesse/Legend ids by the same offset and confirm against the reference-server
skill table (or live atlas-data skill effects). Record the full id set + vehicle-id mapping
(Yeti1→1932003, Yeti2→1932004, Broomstick→1932005, Balrog→1932010, SpaceShip→`1932000+lvl`)
in context.md §8.

- [x] **Step 2: Pin exp-to-level table + cap (OQ 9.4).**

Source `getMountExpNeededForLevel` from the reference server and confirm the level cap
(believed 31). Record the exact table or formula in context.md §8. If only a formula is
available, write it as a Go-ready expression.

- [x] **Step 3: Pin revitalizer tiredness-heal (OQ 9.4).**

Query the revitalizer consumable's WZ spec via live atlas-data
(`GET /api/data/consumables/{id}` with TENANT_ID/REGION/MAJOR_VERSION/MINOR_VERSION headers —
see the `reference_atlas_data_wz_inspection` memory) or MinIO. If the WZ spec lacks a heal
value, use reference parity (30) and document the fallback. Record the value + whether it is
per-item (data-driven) or a constant.

- [x] **Step 4: Pin questline data ids (OQ for FR-9).**

From script comments / local quest data, record the Riding Mimiana quest id(s), NPC id(s),
the Monster Rider skill id granted, and the starter saddle (class 191) + taming-mob (class 190)
item ids in context.md §8.

- [x] **Step 5: Commit.**

```bash
cd .worktrees/task-086-mount-system   # or your worktree root
git add docs/tasks/task-086-mount-system/context.md
git commit -m "task-086: pin mount game-data values (skill/vehicle ids, exp table, heal, questline)"
git branch --show-current   # must print: task-086-mount-system
```

---

# Phase A — libs/atlas-packet (packet fix + SET_TAMING_MOB_INFO writer)

## Task 2: Add MONSTER_RIDING value to the base-stat encoder

**Files:**
- Modify: `libs/atlas-packet/model/character_temporary_stat.go`
- Test: `libs/atlas-packet/model/character_temporary_stat_test.go`

Goal: replace the zeroed Monster Riding base stat (line ~720, the `// TODO look up actual buff
values if riding mount.` placeholder) so it encodes `nOption = stored stat amount`,
`rOption = stored stat sourceId`.

- [x] **Step 1: Write the failing byte-level test (self path).**

Append to `character_temporary_stat_test.go` (follow the existing `TestCTSEncode…` style — `pt.CreateContext("GMS",83,1)`, `tenant.Create`, `AddStat`, then `bytes.Equal`/`bytes.Contains` on the wire segment):

```go
func TestCTSMonsterRidingBaseStatEncodesVehicleAndSkill(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	input := NewCharacterTemporaryStat()
	// amount = vehicle/taming-mob item id, sourceId = skill id
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeMonsterRiding), 1004, 1902000, 1, time.Now().Add(time.Hour))

	got := input.Encode(nil, ctx)(nil)

	// The Monster Riding base-stat block must contain nOption=1902000 then rOption=1004
	// as consecutive little-endian int32s.
	want := []byte{0x40, 0x07, 0x1d, 0x00, /* 1902000 */ 0xec, 0x03, 0x00, 0x00 /* 1004 */}
	if !bytes.Contains(got, want) {
		t.Fatalf("Monster Riding base stat missing nOption=1902000,rOption=1004; got % x", got)
	}
}
```

- [x] **Step 2: Run the test to confirm it fails.**

Run: `cd libs/atlas-packet && go test ./model/ -run TestCTSMonsterRidingBaseStatEncodes -v`
Expected: FAIL (block is currently zeroed).

- [x] **Step 3: Implement — thread the stored stat into the base stat.**

In `character_temporary_stat.go`, change `getBaseTemporaryStats()` so the Monster Riding entry
looks up the active stat and, when present, builds a populated base stat. Add a constructor that
accepts the two options (next to `NewCharacterTemporaryStatBase`):

```go
func NewCharacterTemporaryStatBaseWithOptions(bDynamicTermSet bool, nOption int32, rOption int32) CharacterTemporaryStatBase {
	return CharacterTemporaryStatBase{
		tLastUpdated:    time.Now().Unix(),
		bDynamicTermSet: bDynamicTermSet,
		nOption:         nOption,
		rOption:         rOption,
	}
}
```

Then replace the placeholder line:

```go
	// Monster Riding 13: nOption = vehicle/taming-mob item id, rOption = source skill id (IDA-confirmed v83).
	if s, ok := m.stats[character.TemporaryStatTypeMonsterRiding]; ok {
		list = append(list, NewCharacterTemporaryStatBaseWithOptions(false, s.Value(), s.SourceId()))
	} else {
		list = append(list, NewCharacterTemporaryStatBase(false))
	}
```

(Use the actual field name for the stats map — confirm it is `m.stats` keyed by
`character.TemporaryStatType` per context.md §5.)

- [x] **Step 4: Run the test to confirm it passes.**

Run: `cd libs/atlas-packet && go test ./model/ -run TestCTSMonsterRidingBaseStatEncodes -v`
Expected: PASS.

- [x] **Step 5: Commit.**

```bash
git add libs/atlas-packet/model/character_temporary_stat.go libs/atlas-packet/model/character_temporary_stat_test.go
git commit -m "task-086: encode MONSTER_RIDING base stat with vehicle id + skill id"
```

## Task 3: Cover the observer (EncodeForeign) path

**Files:**
- Test: `libs/atlas-packet/model/character_temporary_stat_test.go`

`getBaseTemporaryStats` is shared, but the foreign path is acceptance-critical — prove it.

- [x] **Step 1: Write the asserting test (foreign path).**

```go
func TestCTSMonsterRidingForeignEncodesVehicleAndSkill(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeMonsterRiding), 1004, 1902000, 1, time.Now().Add(time.Hour))

	got := input.EncodeForeign(nil, ctx)(nil)

	want := []byte{0x40, 0x07, 0x1d, 0x00, 0xec, 0x03, 0x00, 0x00}
	if !bytes.Contains(got, want) {
		t.Fatalf("foreign Monster Riding base stat missing nOption=1902000,rOption=1004; got % x", got)
	}
}
```

- [x] **Step 2: Run it.**

Run: `cd libs/atlas-packet && go test ./model/ -run TestCTSMonsterRidingForeign -v`
Expected: PASS (Task 2's fix is shared). If FAIL, `EncodeForeign` does not append the base block at the same index — inspect and fix `getBaseTemporaryStats`/`EncodeForeign`.

- [x] **Step 3: Commit.**

```bash
git add libs/atlas-packet/model/character_temporary_stat_test.go
git commit -m "task-086: assert MONSTER_RIDING foreign-buff encoding carries vehicle id + skill id"
```

## Task 4: Full atlas-packet module gate (acceptance-critical)

**Files:** none (verification).

- [x] **Step 1: Run the module test/vet/build.**

```bash
cd libs/atlas-packet
go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean; the two new MONSTER_RIDING tests pass.

- [x] **Step 2: Commit (if anything changed) — otherwise skip.**

## Task 5: SET_TAMING_MOB_INFO writer — packet model

**Files:**
- Create: `libs/atlas-packet/character/clientbound/set_taming_mob_info.go`
- Test: `libs/atlas-packet/character/clientbound/set_taming_mob_info_test.go`

Field order (IDA-confirmed): `characterId(4), level(4), exp(4), tiredness(4), levelUp(1 byte)`.
Model the file on an existing clientbound writer in the same package (open one, e.g. a
character buff writer, to copy the `Encode`/writer-const idiom and the package's response.Writer usage).

- [x] **Step 1: Write the failing encode test.**

```go
func TestSetTamingMobInfoFieldOrder(t *testing.T) {
	m := NewSetTamingMobInfo(100200, 5, 1234, 42, true)
	got := m.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x68, 0x87, 0x01, 0x00, // characterId 100200
		0x05, 0x00, 0x00, 0x00, // level 5
		0xd2, 0x04, 0x00, 0x00, // exp 1234
		0x2a, 0x00, 0x00, 0x00, // tiredness 42
		0x01,                   // levelUp true
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("SET_TAMING_MOB_INFO layout mismatch\n got % x\nwant % x", got, want)
	}
}
```

- [x] **Step 2: Run it (fails — type undefined).**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run TestSetTamingMobInfo -v`
Expected: FAIL (compile error / undefined `NewSetTamingMobInfo`).

- [x] **Step 3: Implement the writer.**

```go
package clientbound

import (
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SetTamingMobInfoWriter = "SetTamingMobInfo"

type SetTamingMobInfo struct {
	characterId uint32
	level       uint32
	exp         uint32
	tiredness   uint32
	levelUp     bool
}

func NewSetTamingMobInfo(characterId, level, exp, tiredness uint32, levelUp bool) SetTamingMobInfo {
	return SetTamingMobInfo{characterId: characterId, level: level, exp: exp, tiredness: tiredness, levelUp: levelUp}
}

func (m SetTamingMobInfo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.level)
		w.WriteInt(m.exp)
		w.WriteInt(m.tiredness)
		w.WriteBool(m.levelUp)
		return w.Bytes()
	}
}
```

(Confirm `response.NewWriter` + `WriteInt`/`WriteBool` are the idioms used by neighboring
writers in this package; match them exactly — names may differ, e.g. `WriteUint32`.)

- [x] **Step 4: Run it.**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run TestSetTamingMobInfo -v`
Expected: PASS.

- [x] **Step 5: Commit.**

```bash
git add libs/atlas-packet/character/clientbound/set_taming_mob_info.go libs/atlas-packet/character/clientbound/set_taming_mob_info_test.go
git commit -m "task-086: add SET_TAMING_MOB_INFO clientbound writer"
```

## Task 6: atlas-packet module gate after the writer

**Files:** none.

- [x] **Step 1: Run.**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...
```
Expected: clean.

---

# Phase B — atlas-data (skill reader + consumable spec)

## Task 7: Skill reader emits vehicle ids for skill-only mounts

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/reader.go` (~line 226-228)
- Test: `services/atlas-data/atlas.com/data/skill/reader_test.go` (create if absent)

Consumes Task 1 §8 (skill-id/vehicle-id set). Today the mount branch emits MONSTER_RIDING with
`amount = skillId`. Skill-only mounts must emit the **vehicle id**; SpaceShip is per-level.

- [x] **Step 1: Write the failing test.**

Test that the reader, for Broomstick (skill 1019), emits a MONSTER_RIDING statup with
`Amount == 1932005`, and for SpaceShip (1013) level N emits `Amount == 1932000 + N`. Use a
minimal XML fixture mirroring an existing reader test; if none exists, drive the smallest
exported function that returns the statups for a level. Assert via the
`statup.RestModel{Type, Amount}` shape.

```go
func TestSkillReaderBroomstickVehicleId(t *testing.T) {
	statups := mountStatupsForSkill(skill.BroomstickId, 1) // helper introduced in Step 3
	got, ok := findStatup(statups, string(character.TemporaryStatTypeMonsterRiding))
	if !ok || got.Amount != 1932005 {
		t.Fatalf("Broomstick expected vehicle 1932005, got %+v ok=%v", got, ok)
	}
}

func TestSkillReaderSpaceShipPerLevelVehicleId(t *testing.T) {
	statups := mountStatupsForSkill(skill.SpaceShipId, 3)
	got, _ := findStatup(statups, string(character.TemporaryStatTypeMonsterRiding))
	if got.Amount != 1932000+3 {
		t.Fatalf("SpaceShip L3 expected base 1932000+3, got %d", got.Amount)
	}
}
```

- [x] **Step 2: Run it (fails — undefined helper / wrong amount).**

Run: `cd services/atlas-data && go test ./atlas.com/data/skill/ -run TestSkillReader -v`
Expected: FAIL.

- [x] **Step 3: Implement the vehicle-id mapping.**

If Phase B runs before Task 17 lands the atlas-constants ids, add a local map keyed on the
literal ids and replace with the constants once Task 17 lands (note this in the commit). Extend
the mount branch in `reader.go`:

```go
} else if skill.Is(skillId, skill.BeginnerMonsterRidingId, skill.NoblesseMonsterRidingId, skill.LegendMonsterRidingId, skill.EvanMonsterRidingId, skill.CorsairBattleshipId) {
	// Tamed mounts: amount is a placeholder; the channel overrides it with the equipped taming-mob id.
	statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMonsterRiding, int32(skillId))
} else if vid, ok := skillOnlyMountVehicleId(skillId, level); ok {
	statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMonsterRiding, vid)
}
```

Add `skillOnlyMountVehicleId(skillId Id, level int) (int32, bool)` mapping the pinned
beginner+Noblesse+Legend skill ids to their vehicle ids (SpaceShip → `1932000 + int32(level)`).
Factor `mountStatupsForSkill`/`findStatup` as production helpers (no `*_testhelpers.go`).

- [x] **Step 4: Run it.**

Run: `cd services/atlas-data && go test ./atlas.com/data/skill/ -run TestSkillReader -v`
Expected: PASS.

- [x] **Step 5: Commit.**

```bash
git add services/atlas-data/atlas.com/data/skill/
git commit -m "task-086: atlas-data skill reader emits vehicle ids for skill-only mounts"
```

## Task 8: Consumable reader exposes tiredness-heal (only if WZ carries it)

> **SKIPPED (condition not met).** Task 1 §8.4 verified via live atlas-data that the revitalizer
> (class 226, item 2260000) carries `incFatigue:0` / `spec.inc:0` — no WZ-driven heal field. The
> heal is a server-side constant (30), so per this task's own conditional it is skipped; the
> constant 30 is passed from atlas-consumables (Task 33). Boxes left unchecked intentionally.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/consumable/rest.go` (+ the reader that fills `Spec`)
- Test: `services/atlas-data/atlas.com/data/consumable/rest_test.go`

**Conditional:** only do this if Task 1 §8 found a WZ spec field for mount-food heal. If the
value is a reference-parity constant, **skip this task** and pass the constant from
atlas-consumables (Task 33) — record the skip in the task checkbox.

- [ ] **Step 1: Add the spec type + reader line + test.**

Add `SpecTypeTirednessHeal = SpecType("<wz-field-name-from-task-1>")` to the enum, read it in
the spec loop (`m.Spec[SpecTypeTirednessHeal] = s.GetIntegerWithDefault(string(SpecTypeTirednessHeal), 0)`),
and a test asserting a revitalizer fixture yields the expected heal.

- [ ] **Step 2: Run / Commit.**

```bash
cd services/atlas-data && go test ./atlas.com/data/consumable/ -v
git add services/atlas-data/atlas.com/data/consumable/
git commit -m "task-086: expose revitalizer tiredness-heal spec in consumable reader"
```

## Task 8b: atlas-data module gate

- [x] **Step 1:** `cd services/atlas-data && go test -race ./... && go vet ./... && go build ./...` → clean.

---

# Phase C — atlas-mounts (new service)

> Clone `services/atlas-pets/atlas.com/pets/` as the template. For each file below, copy the
> pets equivalent and apply the named edits. Module name is the **short** `atlas-mounts`.

## Task 9: Scaffold the module

**Files:**
- Create: `services/atlas-mounts/atlas.com/mounts/go.mod`, `logger/`, helper files (copy pets)
- Modify: `go.work`

- [x] **Step 1: Create the directory + go.mod.**

```bash
mkdir -p services/atlas-mounts/atlas.com/mounts
cp services/atlas-pets/atlas.com/pets/go.mod services/atlas-mounts/atlas.com/mounts/go.mod
# Edit the module line to: module atlas-mounts
```

Copy `logger/` and any Makefile/helper files verbatim from pets, renaming the service string.

- [x] **Step 2: Add to go.work so the workspace resolves it.**

Add this line to repo-root `go.work` (alphabetical, near the atlas-pets line):
```
	./services/atlas-mounts/atlas.com/mounts
```

- [x] **Step 3: Build the empty module.**

Run: `cd services/atlas-mounts/atlas.com/mounts && go build ./... 2>&1 | head` (expect "no Go files" or unresolved imports until main.go lands — fine).

- [x] **Step 4: Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/go.mod go.work services/atlas-mounts/atlas.com/mounts/logger
git commit -m "task-086: scaffold atlas-mounts module"
```

## Task 10: character_mounts entity + migration

**Files:**
- Create: `services/atlas-mounts/atlas.com/mounts/mount/entity.go`
- Test: `services/atlas-mounts/atlas.com/mounts/mount/entity_test.go`

Schema (design §4.1): `tenant_id uuid`, `character_id uint32`, `id uuid pk`, `level int default 1`,
`exp int default 0`, `tiredness int default 0`, `last_tiredness_tick_at *time.Time`; uniqueIndex on
`(tenant_id, character_id)`.

- [x] **Step 1: Write the entity + `Make`/`Migration` (copy pets/entity.go shape).**

```go
type Entity struct {
	TenantId            uuid.UUID  `gorm:"not null;uniqueIndex:idx_character_mount_lookup,priority:1"`
	CharacterId         uint32     `gorm:"not null;uniqueIndex:idx_character_mount_lookup,priority:2"`
	Id                  uuid.UUID  `gorm:"primary_key"`
	Level               int        `gorm:"not null;default:1"`
	Exp                 int        `gorm:"not null;default:0"`
	Tiredness           int        `gorm:"not null;default:0"`
	LastTirednessTickAt *time.Time
}

func (e Entity) TableName() string { return "character_mounts" }
func Migration(db *gorm.DB) error { return db.AutoMigrate(&Entity{}) }
func Make(e Entity) (Model, error) { /* build via NewModelBuilder */ }
```

- [x] **Step 2: Test `Make` round-trips an Entity to a Model.** Run → fail → implement `Make` → pass → commit.

```bash
git add services/atlas-mounts/atlas.com/mounts/mount/entity.go services/atlas-mounts/atlas.com/mounts/mount/entity_test.go
git commit -m "task-086: character_mounts entity + migration"
```

## Task 11: Immutable Model + Builder

**Files:**
- Create: `mount/model.go`, `mount/builder.go`
- Test: `mount/builder_test.go`

- [x] **Step 1: Write Model (private fields + getters) and Builder (copy pets/model.go + builder.go).**

Fields: `id uuid.UUID, tenantId uuid.UUID, characterId uint32, level int, exp int, tiredness int,
lastTirednessTickAt *time.Time`. Builder defaults: `level 1, exp 0, tiredness 0`. Provide
`Clone(m)`, `SetLevel/SetExp/SetTiredness/SetLastTick`, `Build()`.

- [x] **Step 2: Test the builder defaults (new mount → level 1/exp 0/tiredness 0).** Run → fail → implement → pass → commit.

```bash
git add services/atlas-mounts/atlas.com/mounts/mount/model.go services/atlas-mounts/atlas.com/mounts/mount/builder.go services/atlas-mounts/atlas.com/mounts/mount/builder_test.go
git commit -m "task-086: mount Model + Builder"
```

## Task 12: Feed math — pure functions (TDD core)

**Files:**
- Create: `mount/feed.go`
- Test: `mount/feed_test.go`

Consumes Task 1 §8 (exp table + cap). Highest-logic unit — test it hard, no I/O.

- [x] **Step 1: Write failing tests for the feed math (FR-8.1/8.2/8.3).**

```go
func TestExpNeededForLevelTableMatchesPinnedValues(t *testing.T) {
	cases := map[int]int{ /* level: need — from context.md §8 (Task 1) */ }
	for lvl, want := range cases {
		if got := ExpNeededForLevel(lvl); got != want {
			t.Fatalf("ExpNeededForLevel(%d)=%d want %d", lvl, got, want)
		}
	}
}

func TestApplyFeedHealsAndGainsExp(t *testing.T) {
	// healMax 30, current tiredness 20, level 1 → heal=20, gain=ceil((20/30)*(2*1+6))=6
	res := ApplyFeed(FeedInput{Level: 1, Exp: 0, Tiredness: 20, HealMax: 30})
	if res.Tiredness != 0 || res.Exp != 6 || res.LevelUp {
		t.Fatalf("got %+v", res)
	}
}

func TestApplyFeedAtCapDoesNotLevel(t *testing.T) {
	res := ApplyFeed(FeedInput{Level: CAP, Exp: 0, Tiredness: 99, HealMax: 30})
	if res.Level != CAP || res.LevelUp {
		t.Fatalf("cap exceeded: %+v", res)
	}
}
```

- [x] **Step 2: Run (fail). Step 3: Implement.**

```go
const CAP = /* pinned in context.md §8 */

func ExpNeededForLevel(level int) int { /* pinned table/formula */ }

type FeedInput struct{ Level, Exp, Tiredness, HealMax int }
type FeedResult struct {
	Level, Exp, Tiredness int
	LevelUp               bool
}

func ApplyFeed(in FeedInput) FeedResult {
	heal := in.Tiredness
	if in.HealMax < heal {
		heal = in.HealMax
	}
	tiredness := in.Tiredness - heal
	gained := int(math.Ceil((float64(heal) / float64(in.HealMax)) * float64(2*in.Level+6)))
	level, exp, levelUp := in.Level, in.Exp+gained, false
	for level < CAP && exp >= ExpNeededForLevel(level) {
		exp -= ExpNeededForLevel(level)
		level++
		levelUp = true
	}
	return FeedResult{Level: level, Exp: exp, Tiredness: tiredness, LevelUp: levelUp}
}
```

- [x] **Step 4: Run (pass). Step 5: Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/mount/feed.go services/atlas-mounts/atlas.com/mounts/mount/feed_test.go
git commit -m "task-086: mount feed heal->exp->level math (FR-8)"
```

## Task 13: Tiredness clamp — pure function

**Files:**
- Modify: `mount/feed.go` (or new `mount/tiredness.go`)
- Test: `mount/tiredness_test.go`

- [x] **Step 1: Failing test (FR-6.1/6.3): increment clamps at 99 and flags TooTired at the clamp.**

```go
func TestTickTirednessClampsAt99(t *testing.T) {
	n, tooTired := TickTiredness(98)
	if n != 99 || tooTired {
		t.Fatalf("98→%d tooTired=%v", n, tooTired)
	}
	n, tooTired = TickTiredness(99)
	if n != 99 || !tooTired {
		t.Fatalf("99→%d tooTired=%v", n, tooTired)
	}
}
```

- [x] **Step 2-4: implement `TickTiredness(t int) (int, bool)` → `min(99,t+1)`, tooTired when result==99. Run → pass. Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/mount/
git commit -m "task-086: tiredness tick clamp at 99 with TooTired flag (FR-6)"
```

## Task 14: Administrator (upsert) + Processor

**Files:**
- Create: `mount/administrator.go`, `mount/processor.go`
- Test: `mount/processor_test.go` (DB-backed via the project's test DB helper used by atlas-pets)

- [x] **Step 1: Copy pets/administrator.go + processor.go; adapt to character_mounts.**

`NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *ProcessorImpl` with
`t := tenant.MustFromContext(ctx)`, a `kp` producer Provider, `With(WithTransaction(tx))`.

Methods (interface + impl):
- `GetByCharacterId(characterId uint32) (Model, error)` — **default-on-first-read**: no row →
  create `level 1/exp 0/tiredness 0` for `(tenant, characterId)` and return it (FR-5.4).
- `upsert(tx)(t, characterId, m)` keyed on `(tenant_id, character_id)`.
- `ApplyTick(mb)(characterId)` — load, `TickTiredness`, persist, emit `MountStatus(TICK[, TooTired])`.
- `ApplyFeedAndEmit(mb)(characterId, healMax)` — load, `ApplyFeed`, persist, emit `MountStatus(FEED, levelUp)`.
- `EmitSet(mb)(characterId)` — load/create, emit `MountStatus(SET)` (on mount activation).

All persistence wrapped in `database.ExecuteTransaction`; persist + `mb.Put(EnvStatusEventTopic,…)`
share one buffer so a crash neither double-applies exp nor loses tiredness (NFR resilience).

- [x] **Step 2: Test default-on-first-read + upsert scoping by tenant+character (Builder fixtures only).** Run → fail → implement → pass.

- [x] **Step 3: Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/mount/administrator.go services/atlas-mounts/atlas.com/mounts/mount/processor.go services/atlas-mounts/atlas.com/mounts/mount/processor_test.go
git commit -m "task-086: mount processor (default-on-read, upsert, tick/feed/set emit)"
```

## Task 15: Mount-status Kafka message + producer

**Files:**
- Create: `kafka/message/mount/kafka.go`, `kafka/producer/producer.go`
- Test: `kafka/message/mount/kafka_test.go`

- [x] **Step 1: Define topic + event (design §4.4 / §12). Copy pets message + producer Provider.**

```go
const (
	EnvStatusEventTopic = "EVENT_TOPIC_MOUNT_STATUS"
	StatusEventTypeSet  = "SET"
	StatusEventTypeTick = "TICK"
	StatusEventTypeFeed = "FEED"
)

type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type StatusEventBody struct {
	Level     int  `json:"level"`
	Exp       int  `json:"exp"`
	Tiredness int  `json:"tiredness"`
	LevelUp   bool `json:"levelUp"`
	TooTired  bool `json:"tooTired"`
}
```

Producer `ProviderImpl` copied verbatim from pets. Add `setEventProvider`, `tickEventProvider`,
`feedEventProvider` returning `model.Provider[[]kafka.Message]` keyed by characterId.

- [x] **Step 2: Test a provider marshals the expected JSON shape. Run → fail → implement → pass. Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/kafka/message/mount services/atlas-mounts/atlas.com/mounts/kafka/producer
git commit -m "task-086: mount status Kafka event + producer"
```

## Task 16: Redis active-mount registry

**Files:**
- Create: `mount/registry.go`
- Test: `mount/registry_test.go`

Design §4.2: `TenantRegistry[uint32, MountRideContext]` keyed by character id storing
`worldId + skillId + vehicleId`. **Route through `libs/atlas-redis` only** (repo invariant).

- [x] **Step 1: Copy pets character/registry.go pattern.**

```go
type MountRideContext struct {
	WorldId   world.Id
	SkillId   int32
	VehicleId int32
}

func InitRegistry(client *goredis.Client) { /* atlas.NewTenantRegistry[uint32, MountRideContext](client, "mount-active", keyFn) + atlas.NewSet */ }
// Add(ctx, characterId, MountRideContext); Remove(ctx, characterId); GetActive(ctx) (map[uint32]Entry, error) // {Tenant, Ctx}
```

- [x] **Step 2: redis-key-guard check (registry must use lib types).**

```bash
GOWORK=off tools/redis-key-guard.sh   # from worktree root
```
Expected: clean (no raw keyed go-redis outside libs/atlas-redis).

- [x] **Step 3: Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/mount/registry.go services/atlas-mounts/atlas.com/mounts/mount/registry_test.go
git commit -m "task-086: redis active-mount registry (atlas-redis lib types)"
```

## Task 17: atlas-constants — mount skill ids + classification 226 + helpers

**Files:**
- Modify: `libs/atlas-constants/skill/constants.go`, `libs/atlas-constants/item/constants.go`
- Create: `libs/atlas-constants/skill/mount.go` (helper) + `mount_test.go`

> Land before atlas-data Task 7 and atlas-channel Task 25 switch from literal ids to constants.

- [x] **Step 1: Add skill id constants (values pinned in Task 1 §8).**

```go
// Skill-only mount skills (beginner band) + Noblesse/Legend equivalents.
SpaceShipId   = Id(1013)
YetiMount1Id  = Id(1017)
YetiMount2Id  = Id(1018)
BroomstickId  = Id(1019)
BalrogMountId = Id(1031)
// … Noblesse (1000_xxxx) / Legend (2000_xxxx) variants per pinned set …
```

- [x] **Step 2: Add classification 226.**

```go
ClassificationRevitalizer = Classification(226) // mount food / taming-mob food
```

- [x] **Step 3: Add helpers + failing test.**

```go
// skill/mount.go
func IsTamedMountSkill(id Id) bool { return uint32(id)%10000000 == 1004 }
func SkillOnlyMountVehicleId(id Id, level int) (int32, bool) { /* map pinned ids → vehicle ids */ }
```

Test `IsTamedMountSkill(1004) == true`, `IsTamedMountSkill(20001004) == true`,
`IsTamedMountSkill(1019) == false`; `SkillOnlyMountVehicleId(1019,1) == (1932005,true)`;
`SkillOnlyMountVehicleId(1013,3) == (1932003,true)`.

- [x] **Step 4: Run → implement → pass. Step 5: Commit.**

```bash
cd libs/atlas-constants && go test ./... && go vet ./...
git add libs/atlas-constants/skill libs/atlas-constants/item
git commit -m "task-086: mount skill ids, classification 226, mount-skill helpers"
```

- [x] **Step 6: Back-fill Task 7 to use the new constants** (replace any literal-id map). Re-run atlas-data skill tests. Commit if changed.

## Task 18: Buff-status consumer → registry population

**Files:**
- Create: `kafka/message/buff/kafka.go` (mirror atlas-buffs event shape), `kafka/consumer/buff/consumer.go`
- Test: `kafka/consumer/buff/consumer_test.go`

Consume `EVENT_TOPIC_CHARACTER_BUFF_STATUS`. On APPLIED with a MONSTER_RIDING change whose
sourceId is a **tamed** mount skill (`IsTamedMountSkill`) → add to registry + `EmitSet`. On
APPLIED skill-only → `EmitSet` only (no registry add → no ticker; FR-2.2). On EXPIRED → remove
from registry (state already persisted; FR-4.4).

- [ ] **Step 1: Copy pets character-consumer wiring (InitConsumers/InitHandlers curried form).** Define the consumed event struct matching context.md §5 atlas-buffs shapes.

- [ ] **Step 2: Test — APPLIED tamed adds to registry + emits SET; APPLIED skill-only emits SET, no add; EXPIRED removes.** Use a fake registry seam. Run → fail → implement → pass.

- [ ] **Step 3: Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/kafka/message/buff services/atlas-mounts/atlas.com/mounts/kafka/consumer/buff
git commit -m "task-086: atlas-mounts consumes buff APPLIED/EXPIRED to drive registry + SET"
```

## Task 19: Login/logout consumer → online gating

**Files:**
- Create: `kafka/message/character/kafka.go`, `kafka/consumer/character/consumer.go`
- Test: `kafka/consumer/character/consumer_test.go`

- [ ] **Step 1: Copy pets character status consumer.** LOGIN registers the character as online (so
the ticker only touches logged-in chars); LOGOUT deregisters + removes any active-mount registry
entry (FR-4.4). Reuse `EVENT_TOPIC_CHARACTER_STATUS` + login/logout body shapes.

- [ ] **Step 2: Test login adds / logout removes the online entry + active-mount entry. Run → fail → implement → pass. Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/kafka/message/character services/atlas-mounts/atlas.com/mounts/kafka/consumer/character
git commit -m "task-086: atlas-mounts online gating from login/logout events"
```

## Task 20: TamingMobFed consumer → feed application

**Files:**
- Create: `kafka/message/food/kafka.go`, `kafka/consumer/food/consumer.go`
- Test: `kafka/consumer/food/consumer_test.go`

- [ ] **Step 1: Define consumed event + handler.**

```go
const EnvEventTopic = "EVENT_TOPIC_TAMING_MOB_FOOD"

type Event struct {
	CharacterId   uint32 `json:"characterId"`
	ItemId        uint32 `json:"itemId"`
	TirednessHeal int32  `json:"tirednessHeal"`
}
```

Handler calls `mount.NewProcessor(l,ctx,db).ApplyFeedAndEmit(...)(characterId, int(e.TirednessHeal))`.

- [ ] **Step 2: Test the handler routes a fed event into ApplyFeedAndEmit (seam). Run → fail → implement → pass. Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/kafka/message/food services/atlas-mounts/atlas.com/mounts/kafka/consumer/food
git commit -m "task-086: atlas-mounts applies TamingMobFed -> feed math + FEED event"
```

## Task 21: TirednessTask (60s ticker)

**Files:**
- Create: `mount/task.go`, `tasks/task.go` (copy pets/tasks)
- Test: `mount/task_test.go`

- [ ] **Step 1: Copy pets HungerTask → TirednessTask; cadence `time.Minute` (60s, FR-6.1).**

`Run()` iterates online characters with an **active tamed mount** in the registry and calls
`ApplyTick`. Skill-only mounts are absent from the registry → never ticked (FR-2.2). One task
iterating the registry; no per-character goroutines/timers (NFR performance).

- [ ] **Step 2: Test Run() ticks each active mount once (seam over registry + processor). Run → fail → implement → pass. Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/mount/task.go services/atlas-mounts/atlas.com/mounts/tasks
git commit -m "task-086: 60s tiredness ticker over active-mount registry"
```

## Task 22: REST resource (parity/debug)

**Files:**
- Create: `mount/rest.go`, `mount/resource.go`
- Test: `mount/rest_test.go`

- [ ] **Step 1: Copy pets rest.go + resource.go.** `GetName() == "mounts"`; route
`GET /characters/{characterId}/mount` (Transform from Model). Minimal `Transform`/`Extract`.

- [ ] **Step 2: Test Transform maps Model→RestModel. Run → fail → implement → pass. Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/mount/rest.go services/atlas-mounts/atlas.com/mounts/mount/resource.go services/atlas-mounts/atlas.com/mounts/mount/rest_test.go
git commit -m "task-086: atlas-mounts REST resource (GET character mount)"
```

## Task 23: main.go wiring

**Files:**
- Create: `services/atlas-mounts/atlas.com/mounts/main.go`

- [ ] **Step 1: Copy pets/main.go; swap serviceName="atlas-mounts", consumer group "Mounts Service".**

Wire: `atlas.Connect` + `mount.InitRegistry(rc)`; `database.Connect(..., SetMigrations(mount.Migration))`;
consumers `buff`, `character`, `food` (InitConsumers + InitHandlers); REST route initializer
`mount.InitResource`; task `tasks.Register(l, ctx)(mount.NewTirednessTask(l, db, time.Minute))`;
producer teardown.

- [ ] **Step 2: Build the service.**

```bash
cd services/atlas-mounts/atlas.com/mounts && go build ./...
```
Expected: clean.

- [ ] **Step 3: Commit.**

```bash
git add services/atlas-mounts/atlas.com/mounts/main.go
git commit -m "task-086: atlas-mounts main wiring (registry, db, consumers, task, rest)"
```

## Task 24: atlas-mounts full module gate

- [ ] **Step 1:** from worktree root run `go work sync` if needed, then:
```bash
cd services/atlas-mounts/atlas.com/mounts
go test -race ./... && go vet ./... && go build ./...
```
Expected: clean. Commit any fixups.

---

# Phase D — atlas-channel (toggle, food handler, writer + mount-status consumer)

## Task 25: Mount toggle branch in the skill-use path

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go` (in `UseSkill`, around line 97)
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/mount.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/mount_test.go`

Behavior (design §5.1; IDA toggle + both-slots facts). Insert a mount branch **before** the
generic `e.Duration() > 0 && len(e.StatUps()) > 0` apply. Use the existing loadCaster/seam style
in common.go so it is unit-testable offline.

- [ ] **Step 1: Write failing tests (table-driven, seams injected).**

Cases:
1. Already mounted (session buff state has MONSTER_RIDING for this skill) → emits `Cancel(sourceId=skillId)`, no Apply.
2. Tamed, both slots -18 & -19 present, not mounted → `Apply(MONSTER_RIDING, amount=item@-18, sourceId=skillId, duration=MaxInt32)`.
3. Tamed, slot -18 empty → no Apply, no Cancel (silent no-op) + enableActions.
4. Tamed, slot -19 empty → no-op.
5. Skill-only (e.g. 1019), not mounted → `Apply(amount = e.StatUps() MONSTER_RIDING amount, sourceId=skillId, duration=MaxInt32)`, no slot check.

```go
func TestMountToggleCancelsWhenAlreadyMounted(t *testing.T) { /* seam: isMounted=true → expect cancelCalled, no apply */ }
func TestMountTamedRequiresBothSlots(t *testing.T)          { /* -18 set, -19 empty → no apply */ }
func TestMountTamedAppliesVehicleFromSlot18(t *testing.T)   { /* expect apply amount == item@-18 */ }
func TestMountSkillOnlyNoSlotCheck(t *testing.T)            { /* skill 1019 → apply amount from effect */ }
```

- [ ] **Step 2: Run (fail). Step 3: Implement `HandleMount` + branch.**

```go
// mount.go
const MountBuffDuration = int32(math.MaxInt32) // mounts persist until toggle/job-change/logout (atlas-buffs rejects <=0)

func HandleMount(l logrus.FieldLogger, ctx context.Context, wp writer.Producer,
	f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model,
	isMounted func() bool, loadEquip func() (equipSlots, error)) error {

	skillId := int32(info.SkillId())
	bp := buff.NewProcessor(l, ctx)
	if isMounted() {
		_ = bp.Cancel(f, characterId, skillId) // server-driven dismount toggle
		return enableActions(l)(ctx)(wp)( /* session */ )
	}
	if skill2.IsTamedMountSkill(skill2.Id(skillId)) {
		eq, err := loadEquip()
		if err != nil || eq.TamingMob == 0 || eq.Saddle == 0 {
			return enableActions(l)(ctx)(wp)( /* session */ ) // silent no-op, both slots required
		}
		_ = bp.Apply(f, characterId, skillId, info.SkillLevel(), MountBuffDuration, vehicleStatups(eq.TamingMob))(characterId)
		return nil
	}
	// skill-only: vehicle id already produced by atlas-data in e.StatUps()
	_ = bp.Apply(f, characterId, skillId, info.SkillLevel(), MountBuffDuration, e.StatUps())(characterId)
	return nil
}
```

In `common.go UseSkill`, branch before the generic apply:

```go
	if skill2.IsTamedMountSkill(skill2.Id(info.SkillId())) || skillOnlyMount(skill2.Id(info.SkillId())) {
		return HandleMount(l, ctx, wp, f, characterId, info, e, mountedProbe(...), equipLoader(...))
	}
```

Resolve `isMounted` from the session buff state the buff consumer tracks (context.md §5);
`loadEquip` via `cp.GetById(cp.InventoryDecorator)` reading slots -18/-19. `vehicleStatups`
builds a `[]statup.Model` carrying MONSTER_RIDING with `amount = tamingMobItemId`.

- [ ] **Step 4: Run (pass). Step 5: Commit.**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/mount.go services/atlas-channel/atlas.com/channel/skill/handler/mount_test.go services/atlas-channel/atlas.com/channel/skill/handler/common.go
git commit -m "task-086: channel mount toggle (apply/cancel, both-slots, skill-only)"
```

## Task 26: SET_TAMING_MOB_INFO writer registration

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/writer/set_taming_mob_info.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (`produceWriters()`)

- [ ] **Step 1: Add the writer body wrapper + register the const.**

```go
// writer/set_taming_mob_info.go
func SetTamingMobInfoBody(characterId, level, exp, tiredness uint32, levelUp bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return clientbound.NewSetTamingMobInfo(characterId, level, exp, tiredness, levelUp).Encode(l, ctx)
	}
}
```

Add `clientbound.SetTamingMobInfoWriter` to the `produceWriters()` string list in main.go.

- [ ] **Step 2: Build.** `cd services/atlas-channel/atlas.com/channel && go build ./...` → clean.

- [ ] **Step 3: Commit.**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/set_taming_mob_info.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "task-086: register SET_TAMING_MOB_INFO writer in channel"
```

## Task 27: Mount-status consumer → broadcast SET_TAMING_MOB_INFO

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/mount/kafka.go`, `kafka/consumer/mount/consumer.go`
- Modify: `main.go` (register consumer)
- Test: `kafka/consumer/mount/consumer_test.go`

- [ ] **Step 1: Define the consumed `EVENT_TOPIC_MOUNT_STATUS` event (mirror atlas-mounts §15) + handler.**

Handler: resolve session via `session.NewProcessor(l,ctx).IfPresentByCharacterId(channel)(characterId, …)`;
broadcast `SetTamingMobInfoBody(...)` to the map via `_map.NewProcessor(l,ctx).ForSessionsInMap(s.Field(), op)`.
On `TooTired`, also send the FR-6.3 notice to the rider only ("Your mount grew tired! Treat it
some revitalizer before riding it again!") via the existing notice writer.

- [ ] **Step 2: Test the handler emits a broadcast for SET/TICK/FEED and the notice on TooTired (seams). Run → fail → implement → pass.**

- [ ] **Step 3: Register the consumer in main.go (InitConsumers + InitHandlers).**

- [ ] **Step 4: Commit.**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/mount services/atlas-channel/atlas.com/channel/kafka/consumer/mount services/atlas-channel/atlas.com/channel/main.go
git commit -m "task-086: channel broadcasts SET_TAMING_MOB_INFO from mount status events"
```

## Task 28: Food opcode 0x4D inbound handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/mount_food.go`
- Create (packet): `libs/atlas-packet/<mount>/serverbound/food.go` (mirror pet food serverbound)
- Modify: `main.go` (`produceHandlers()`), validator map if needed
- Test: `socket/handler/mount_food_test.go`

Body: `ts(4), slot(2), itemId(4)` (IDA). Model on `socket/handler/pet_food.go` + the pet
serverbound packet.

- [ ] **Step 1: Add the serverbound decode struct (copy pet food serverbound) + a decode test.**

- [ ] **Step 2: Add the handler.**

```go
func MountFoodHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) {
	return func(s session.Model, r *request.Reader, ro map[string]interface{}) {
		p := mount2.Food{}
		p.Decode(l, ctx)(r, ro)
		// emit COMMAND_TOPIC_TAMING_MOB_FOOD {characterId, slot, itemId} to atlas-consumables
		_ = food.NewProcessor(l, ctx).RequestFeed(s.Field(), character.Id(s.CharacterId()), slot.Position(p.Source()), item.Id(p.ItemId()))
	}
}
```

(Introduce a small channel-side `food` producer emitting the new command topic — copy the buff
producer pattern. The handler performs no item mutation; consumables decrements.)

- [ ] **Step 3: Register `MountFoodHandle` in `produceHandlers()`** keyed to the per-tenant opcode
(0x4D supplied by tenant config — do NOT hardcode the byte; it is wired through `Socket.Handlers`).

- [ ] **Step 4: Run handler/decoder tests → pass. Step 5: Commit.**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/mount_food.go libs/atlas-packet/*/serverbound services/atlas-channel/atlas.com/channel/main.go
git commit -m "task-086: channel mount-food inbound handler (opcode 0x4D) -> consumables command"
```

## Task 29: Channel food command message + producer

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/food/kafka.go`, a channel-side food producer
- Test: producer provider shape test

- [ ] **Step 1: Define `COMMAND_TOPIC_TAMING_MOB_FOOD` + `Command{CharacterId, Slot, ItemId, …}` and the producer (copy buff producer).** Provide `RequestFeed(...)` used by Task 28.

- [ ] **Step 2: Test provider marshals expected shape. Run → pass. Commit.**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/food
git commit -m "task-086: channel taming-mob-food command + producer"
```

## Task 30: Job-change dismount wiring (verify/confirm)

**Files:**
- Inspect: how atlas-channel/atlas-character emits buff cancel on job change; confirm MONSTER_RIDING is included.

- [ ] **Step 1: Confirm job-change cancels MONSTER_RIDING (FR-4.2).** atlas-buffs `CancelByStatTypes`
already exists (context.md §5). Verify the job-change flow cancels by stat types and that
MONSTER_RIDING is in the cancelled set (or add it). If the path already does
`CancelAll`/`CancelByStatTypes`, document that no change is needed; otherwise add MONSTER_RIDING
with a test.

- [ ] **Step 2: Commit any change (or record "no change needed" in the task note).**

## Task 31: atlas-channel module gate

- [ ] **Step 1:** `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...` → clean. Fix + commit.

---

# Phase E — atlas-consumables (food command + TamingMobFed event)

## Task 32: TamingMobFood command consumer

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/...` (new `food` message pkg)
- Create: `kafka/consumer/food/consumer.go`
- Modify: `main.go` (register consumer)
- Test: `kafka/consumer/food/consumer_test.go`

- [ ] **Step 1: Define consumed `COMMAND_TOPIC_TAMING_MOB_FOOD` command + handler.**

Handler validates `item.GetClassification(itemId) == item.ClassificationRevitalizer` (226),
decrements one via the existing `ConsumeItem`/`RequestItemConsume` path, then emits `TamingMobFed`.
The heal→exp→level math stays in atlas-mounts (Task 12/20).

- [ ] **Step 2: Test — class-226 item routes to consume + emits fed event; non-226 is rejected. Run → fail → implement → pass.**

- [ ] **Step 3: Register the consumer in main.go. Commit.**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka services/atlas-consumables/atlas.com/consumables/main.go
git commit -m "task-086: consumables handles taming-mob-food command (class 226) + consumes item"
```

## Task 33: TamingMobFed event producer

**Files:**
- Modify: the new `food` event pkg + producer
- Test: provider shape test

- [ ] **Step 1: Define `EVENT_TOPIC_TAMING_MOB_FOOD` + `{characterId, itemId, tirednessHeal}` + provider.**

`tirednessHeal` comes from the consumable spec (Task 8) if data-driven, else the pinned constant
(Task 1 §8). Emit after a successful consume.

- [ ] **Step 2: Test provider shape. Run → pass. Commit.**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka
git commit -m "task-086: consumables emits TamingMobFed event with tiredness heal"
```

## Task 34: atlas-consumables module gate

- [ ] **Step 1:** `cd services/atlas-consumables && go test -race ./... && go vet ./... && go build ./...` → clean. Commit fixups.

## Task 35: Cross-service Kafka contract check

**Files:** none (review).

- [ ] **Step 1: Confirm topic env-var names + JSON field tags match exactly across producer/consumer pairs:**
  - channel→consumables `COMMAND_TOPIC_TAMING_MOB_FOOD` (Task 29 ↔ Task 32)
  - consumables→mounts `EVENT_TOPIC_TAMING_MOB_FOOD` (Task 33 ↔ Task 20)
  - mounts→channel `EVENT_TOPIC_MOUNT_STATUS` (Task 15 ↔ Task 27)
  Grep each topic constant in both modules and diff the body struct field tags. Fix mismatches.

- [ ] **Step 2: Commit any contract fix.**

---

# Phase F — Riding Mimiana questline (FR-9)

## Task 36: Author the quest definition(s)

**Files:**
- Create: quest definition JSON under the project's quest data location (use Task 1 §8 ids)
- Use: `/convert-quest` conventions

- [ ] **Step 1: Convert/author the Riding Mimiana quest so completion grants the Monster Rider
skill (1004 band) + starter saddle (class 191) + taming-mob (class 190)** via the existing
atlas-quest → atlas-skills skill-grant + inventory item-grant reward paths. Use `convert-quest`
tooling; ids from context.md §8. Skip Player-NPC spawning (project convention).

- [ ] **Step 2: Validate via the convert-quest tooling output. Commit.**

```bash
git add <quest-data-paths>
git commit -m "task-086: Riding Mimiana quest awards Monster Rider skill + starter mount items"
```

## Task 37: Author the NPC conversation

**Files:**
- Create: NPC conversation JSON (Task 1 §8 NPC id)
- Use: `/convert-npc` conventions

- [ ] **Step 1: Author the questgiver NPC conversation state machine** following the project's JSON
conventions; wire start/complete to the quest from Task 36.

- [ ] **Step 2: Validate via convert-npc tooling. Commit.**

```bash
git add <npc-data-paths>
git commit -m "task-086: Riding Mimiana NPC conversation"
```

## Task 38: Questline validation gate

- [ ] **Step 1:** Run whatever validation the convert-quest/convert-npc tooling provides over the
new data; confirm no schema errors. Record result in the task note.

---

# Phase G — build/deploy wiring, live-config, full verification

## Task 39: Register atlas-mounts in the build system

**Files:**
- Modify: `.github/config/services.json`, `docker-bake.hcl` (`go.work` already done in Task 9)

- [ ] **Step 1: Add the services.json entry (alphabetical).**

```json
{
  "name": "atlas-mounts",
  "type": "go-service",
  "path": "services/atlas-mounts",
  "module_path": "services/atlas-mounts/atlas.com/mounts",
  "docker_image": "ghcr.io/chronicle20/atlas-mounts/atlas-mounts",
  "docker_context": "."
}
```

- [ ] **Step 2: Add `"atlas-mounts",` to `go_services` in `docker-bake.hcl` (alphabetical).**

- [ ] **Step 3: Commit.**

```bash
git add .github/config/services.json docker-bake.hcl
git commit -m "task-086: register atlas-mounts in services.json + docker-bake"
```

## Task 40: K8s manifest

**Files:**
- Create: `deploy/k8s/base/atlas-mounts.yaml` (copy `atlas-pets.yaml`)

- [ ] **Step 1: Copy atlas-pets.yaml → atlas-mounts.yaml; replace all `atlas-pets`→`atlas-mounts`,
set `DB_NAME: "atlas-mounts"`. No LB socket ports (REST+Kafka only).** Add to the base
kustomization if the repo lists manifests there (check `deploy/k8s/base/kustomization.yaml`).

- [ ] **Step 2: Commit.**

```bash
git add deploy/k8s/base/atlas-mounts.yaml deploy/k8s/base/kustomization.yaml
git commit -m "task-086: atlas-mounts k8s manifest"
```

## Task 41: docker buildx bake — every touched service

**Files:** none (verification — mandatory per CLAUDE.md).

- [ ] **Step 1: From the worktree root, bake the new service + every service whose go.mod changed.**

```bash
docker buildx bake atlas-mounts
docker buildx bake atlas-channel
docker buildx bake atlas-data
docker buildx bake atlas-consumables
docker buildx bake atlas-buffs        # only if its go.mod changed
```
Expected: all succeed. A missing `COPY libs/...` only surfaces here — fix the repo-root
`Dockerfile` if any bake fails (no new shared lib was added, so this should pass clean).

- [ ] **Step 2: Commit any Dockerfile fix.**

## Task 42: Live-config deployment note + final full gate (OQ 9.7)

**Files:**
- Create: `docs/tasks/task-086-mount-system/deploy-notes.md`

- [ ] **Step 1: Document the live-config patch (known pitfall — seed templates apply only at
tenant creation).** Record that existing tenants need, per channel:
  - inbound handler opcode **0x4D** → `MountFoodHandle` added to live `Socket.Handlers`
  - outbound writer opcode for `SetTamingMobInfo` added to live `Socket.Writers`
  - **then restart the channel** (projection does not hot-reload handlers/writers).
Reference the `bug_new_opcodes_not_in_live_tenant_config` memory.

- [ ] **Step 2: Run the full repo verification gate from the worktree root.**

```bash
for m in libs/atlas-packet libs/atlas-constants services/atlas-data services/atlas-mounts/atlas.com/mounts services/atlas-channel/atlas.com/channel services/atlas-consumables services/atlas-buffs; do
  echo "== $m ==" && (cd "$m" && go test -race ./... && go vet ./... && go build ./...) || break
done
GOWORK=off tools/redis-key-guard.sh
```
Expected: every module clean; redis-key-guard clean.

- [ ] **Step 3: Commit deploy notes.**

```bash
git add docs/tasks/task-086-mount-system/deploy-notes.md
git commit -m "task-086: live-config deploy notes + final verification"
```

---

## Self-review traceability (spec → task)

| Requirement | Task(s) |
|---|---|
| FR-1.1..1.4 prereq validation (both slots, silent no-op) | 25 |
| FR-2.1..2.3 skill-only mounts (no equip, no tiredness, reader vehicle id) | 7, 17, 25 |
| FR-3.1..3.5 buff application (vehicle id + skill id, no stacking via toggle) | 2, 3, 25 |
| FR-4.1..4.4 dismount (toggle, job-change, no damage-dismount, logout deregister) | 25, 30, 18, 19 |
| FR-5.1..5.5 persistent state (scoped, survives logout/channel, default-on-read) | 10, 11, 14, 19 |
| FR-6.1..6.4 tiredness (60s, broadcast, clamp 99, notice, per-mount stop) | 13, 21, 27 |
| FR-7.1..7.3 SET_TAMING_MOB_INFO (fields, send triggers, opcode per tenant) | 5, 26, 27, 42 |
| FR-8.1..8.6 feeding (heal, exp formula, level cap, consume, broadcast, exp table) | 1, 8, 12, 20, 32, 33 |
| FR-9.1..9.2 questline | 1, 36, 37, 38 |
| NFR multi-tenancy / redis discipline / resilience / wire correctness | 14, 16, 2/3, 24 |
| Build/deploy (new service, bake, live config) | 9, 39, 40, 41, 42 |
| Battleship NOT touched | `skill.Is(... CorsairBattleshipId)` stays in the tamed branch (Task 7); no new Battleship code |

> **Acceptance-critical:** Tasks 2–4 (MONSTER_RIDING byte-level encoding, self + foreign) gate the
> entire feature's rendering. Run them first after Task 1.
