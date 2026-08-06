# Disease Duration Units & CANCEL_DEBUFF Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make mob-skill disease durations correct (milliseconds end-to-end), give the client a working `CANCEL_DEBUFF` recovery path so a desynced temporary stat self-heals, and add a CI analyzer so the seconds↔milliseconds contract cannot silently flip a fourth time.

**Architecture:** Exactly one seconds→milliseconds conversion exists in the system, in `atlas-data mobskill/reader.go`; every downstream site forwards verbatim. The `CANCEL_DEBUFF` recovery is an empty-body serverbound nudge → an in-process per-character throttle in atlas-channel → a new `EXPIRE` command on the existing `COMMAND_TOPIC_CHARACTER_BUFF` → a new per-character expiry sweep in atlas-buffs that reuses the existing `Registry.GetExpired` prune, whose `EXPIRED` events already flow through the existing channel consumer to the existing `CharacterBuffCancel` writer.

**Tech Stack:** Go 1.25.5, Kafka (segmentio/kafka-go via `libs/atlas-kafka`), `golang.org/x/tools/go/analysis` (the CI guard), Next.js/TypeScript (atlas-ui), JSON seed templates (atlas-configurations).

## Global Constraints

- Work happens **only** in the worktree `.worktrees/task-190-disease-duration-cancel-debuff` on branch `task-190-disease-duration-cancel-debuff`. Never edit the main checkout.
- **The `COMMAND_TOPIC_CHARACTER_BUFF` `duration` field is MILLISECONDS.** Authority: `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`, on `ApplyCommandBody.Duration`. Every other copy carries a pointer comment, never a restatement.
- **`services/atlas-monsters/atlas.com/monsters/monster/processor.go` `executeDebuff` (≈`:1242`) must NOT be edited.** It forwards `sd.Duration()` verbatim and becomes correct once the reader lands. Adding a second conversion there is a defect.
- **No wire values in Go.** Opcodes come from tenant socket configuration; the handler is bound by *name*. A literal `0x63` would mis-route on v61 (DOM-25).
- **The `CANCEL_DEBUFF` codec body is empty on all ten clients.** Do not invent a stat mask, skill id, or tick field.
- **Do not create `docs/packets/registry/gms_v92.yaml`.** v92 is not a matrix column; its opcode is recorded in the seed template, `investigation.md` §8.2, and `backfill.md`.
- **Do not route `CANCEL_DEBUFF` in `template_gms_12_1.json`.** That template routes 24 handlers, none of them buff/skill/attack related — the packet is outside its feature surface.
- Tests that pin the old seconds contract are **updated, not deleted**, with corrected comments.
- No `// TODO`, stubbed handlers, or 501s in any commit.
- Never write literal home/absolute paths (`/home/<user>/…`) into committed files.
- Preserve line endings; do not normalize CRLF→LF as a side effect.

**Version set and opcodes (IDB-derived, `investigation.md` §8.2 — do not re-derive):**

| Version | Opcode | Template file | Registry file |
|---|---|---|---|
| gms_v48 | `0x4E` (78) | `template_gms_48_1.json` | `gms_v48.yaml` (ADD) |
| gms_v61 | `0x5B` (91) | `template_gms_61_1.json` | `gms_v61.yaml` (ADD) |
| gms_v72 | `0x62` (98) | `template_gms_72_1.json` | `gms_v72.yaml` (ADD) |
| gms_v79 | `0x61` (97) | `template_gms_79_1.json` | `gms_v79.yaml` (ADD) |
| gms_v83 | `0x63` (99) | `template_gms_83_1.json` | already present |
| gms_v84 | `0x63` (99) | `template_gms_84_1.json` | already present |
| gms_v87 | `0x66` (102) | `template_gms_87_1.json` | already present |
| gms_v92 | `0x6E` (110) | `template_gms_92_1.json` | none (deliberate) |
| gms_v95 | `0x6F` (111) | `template_gms_95_1.json` | already present |
| jms_v185 | `0x5E` (94) | `template_jms_185_1.json` | already present |

---

## File Structure

**Modified — FR-1 (duration units)**
- `services/atlas-data/atlas.com/data/mobskill/reader.go` — the single conversion point.
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — two double-conversions removed.
- `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go` — stale comment replaced, value corrected.
- `services/atlas-ui/src/components/features/monsters/MonsterSkillChip.tsx` — render ms as seconds.

**Created — FR-3 (contract guard)**
- `tools/buffdurationguard/{go.mod,go.sum,analyzer.go,analyzer_test.go,cmd/buffdurationguard/main.go,testdata/src/{bad,good}/*.go}`
- `tools/buff-duration-guard.sh`

**Created — FR-2 (CANCEL_DEBUFF)**
- `libs/atlas-packet/character/serverbound/cancel_debuff.go` (+ `_test.go`) — opcode-only codec.
- `services/atlas-channel/atlas.com/channel/character/statreset/registry.go` (+ `_test.go`) — the throttle.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cancel_debuff.go` (+ `_test.go`).

**Modified — FR-2**
- `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go` — `CommandTypeExpire`, `ExpireCommandBody`.
- `services/atlas-channel/atlas.com/channel/character/buff/{producer.go,processor.go}` — `ExpireCommandProvider`, `Processor.Expire`.
- `services/atlas-channel/atlas.com/channel/socket/init.go` — registry eviction on session destroy.
- `services/atlas-channel/atlas.com/channel/main.go` — handler map registration.
- `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` — command type, body, contract authority comment.
- `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go` — `handleExpire`.
- `services/atlas-buffs/atlas.com/buffs/character/processor.go` — `ExpireForCharacter` + shared `expireInto` helper.
- `services/atlas-configurations/seed-data/templates/template_gms_{48,61,72,79,83,84,87,92,95}_1.json`, `template_jms_185_1.json`.
- `docs/packets/registry/gms_v{48,61,72,79}.yaml`, `docs/packets/audits/status.json`, `docs/packets/audits/STATUS.md`.
- `.github/workflows/pr-validation.yml`, `CLAUDE.md`.

**Created — docs**
- `docs/tasks/task-190-disease-duration-cancel-debuff/producer-audit.md`
- `docs/tasks/task-190-disease-duration-cancel-debuff/backfill.md`

---

### Task 1: atlas-data — the single seconds→milliseconds conversion

**Files:**
- Modify: `services/atlas-data/atlas.com/data/mobskill/reader.go:66`
- Test: `services/atlas-data/atlas.com/data/mobskill/reader_test.go` (create if absent)

**Interfaces:**
- Consumes: nothing.
- Produces: `mobskill.RestModel.Duration` (`uint32`) and `mobskill.Model.Duration()` (`uint32`) are **milliseconds**. Tasks 2 and 3 depend on this.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-data/atlas.com/data/mobskill/reader_test.go`. Match the existing package name in `reader.go` (`package mobskill`).

```go
package mobskill

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-xml/xml"
)

// TestReadLevel_DurationIsMilliseconds pins the ONE seconds→milliseconds
// conversion in the system. WZ MobSkill.img authors `time` in seconds; every
// consumer downstream of this reader (atlas-monsters executeDebuff,
// buildMistCreateBody, executeStatBuff; atlas-maps mist tick) forwards the
// value verbatim as milliseconds. Mirrors skill/reader.go's convention
// (task-054). Do not add a second conversion anywhere downstream.
func TestReadLevel_DurationIsMilliseconds(t *testing.T) {
	node := xml.NewNode("2", map[string]string{
		"time": "15",
	}, nil)

	m := readLevel(logrus.New(), 126, 2, "126", node)

	if m.Duration != 15000 {
		t.Errorf("Duration: got %d, want 15000 (15s authored in WZ, emitted as ms)", m.Duration)
	}
}
```

If the `xml.Node` constructor signature above does not compile, discover the real one first:

```bash
grep -rn "func NewNode\|type Node struct\|type Node interface" libs/atlas-xml/xml/
grep -rn "xml.Node" services/atlas-data/atlas.com/data/ --include=*_test.go | head
```

and build the fixture node with whatever constructor/helper the repo already uses in an existing atlas-data reader test. The assertion (`Duration == 15000` for `time=15`) is the fixed part; the node construction is whatever compiles.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-data/atlas.com/data && go test ./mobskill/ -run TestReadLevel_DurationIsMilliseconds -v
```

Expected: FAIL — `Duration: got 15, want 15000`.

- [ ] **Step 3: Apply the conversion**

In `services/atlas-data/atlas.com/data/mobskill/reader.go`, replace:

```go
	m.Duration = uint32(node.GetIntegerWithDefault("time", 0))
```

with:

```go
	// Why ms: the wz `time` attribute is in seconds; convert here — this is the
	// ONLY seconds→ms conversion for mob-skill data. Downstream consumers
	// (atlas-monsters executeDebuff/buildMistCreateBody/executeStatBuff,
	// atlas-maps mist tick) forward Duration verbatim as milliseconds. Mirrors
	// skill/reader.go's effect-duration convention (task-054). See
	// docs/tasks/task-190-disease-duration-cancel-debuff/. (task-190)
	m.Duration = uint32(node.GetIntegerWithDefault("time", 0)) * 1000
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd services/atlas-data/atlas.com/data && go test ./mobskill/ -run TestReadLevel_DurationIsMilliseconds -v
```

Expected: PASS.

- [ ] **Step 5: Run the whole module**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go vet ./... && go test -race ./...
```

Expected: all clean. If another atlas-data test pins seconds for mob skills, update it (do not delete it) with a comment naming task-190.

- [ ] **Step 6: Record the overflow bound**

`RestModel.Duration` is `uint32` but `executeDebuff` narrows to `int32`, so int32 is the binding constraint: overflow needs an authored `time > 2_147_483` seconds ≈ 24.8 days. Confirm no authored value approaches this by querying the ingested rows in the live baseline (per CLAUDE.md, verify against real data, do not assert from memory):

```bash
kubectl get pods -A | grep atlas-data   # find a running pod/namespace
# then, from a busybox/exec shell with tenant headers, page /api/data/mob-skills
# and take the max `duration`. Record the observed maximum verbatim.
```

If the live baseline is unreachable, read the maximum straight out of the WZ source instead:

```bash
grep -rhoE 'name="time" value="[0-9]+"' <path-to-Skill.wz-extract>/MobSkill.img.xml | \
  grep -oE '[0-9]+$' | sort -n | tail -1
```

Write the observed number (not an assertion of safety) into `docs/tasks/task-190-disease-duration-cancel-debuff/producer-audit.md` in Task 5 under a `## Range check` heading. If neither source is reachable, say so explicitly in that doc — do not claim a number you did not read.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-data/atlas.com/data/mobskill/
git commit -m "fix(atlas-data): mob-skill duration is milliseconds (task-190 FR-1.1)"
```

---

### Task 2: atlas-monsters — remove the two double-conversions

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — `buildMistCreateBody` (≈`:1068`), `executeStatBuff` (≈`:1105`)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` — the tests at ≈`:894`, ≈`:1179`, ≈`:1236`, ≈`:1263` (locate by content, not line number)
- Do **not** modify: `executeDebuff` (≈`:1242`)

**Interfaces:**
- Consumes: `mobskill.Model.Duration()` is milliseconds (Task 1).
- Produces: `mistKafka.CreateCommandBody.DiseaseDuration` and `.Duration` are milliseconds (Task 3 consumes them via the mist registry); `applyDiseaseBody.Duration` is milliseconds.

- [ ] **Step 1: Update the tests that pin the seconds contract**

In `processor_test.go`, `TestBuildMistCreateBody` currently builds `SetDuration(10) // seconds` and asserts `body.Duration != 10_000`. Change the input to milliseconds and keep the expectation:

```go
		SetDuration(10_000). // milliseconds — mobskill.Duration() is ms since task-190
```

and update the doc comment above the test from `(modulo seconds→ms)` to:

```go
// TestBuildMistCreateBody verifies the pure mapping from a casting monster +
// AREA_POISON skill data to the wire MIST_CREATE body. Field identity, owner
// identity, origin coordinates, bounding box, disease/duration, and skill
// references must all flow through unchanged. mobskill.Duration() is
// MILLISECONDS (task-190 FR-1.1) and this body forwards it verbatim — there is
// no scaling here.
```

`TestBuildMistCreateBody_DurationCap` currently uses `SetDuration(1800). // 30 minutes — must clamp`. Change to:

```go
		SetDuration(1_800_000). // 30 minutes in ms — must clamp to MistDurationCapMs
```

and update its doc comment to say the cap now bites on real milliseconds:

```go
// TestBuildMistCreateBody_DurationCap verifies that absurdly long durations
// (e.g. atlas-data reporting 30 minutes) are clamped to MistDurationCapMs so
// per-mist tick load stays bounded. Before task-190 this clamp fired on a
// 1000×-inflated value and silently pinned EVERY mob mist to exactly 60s; it
// now applies to real milliseconds, so only mists authored longer than 60s
// clamp.
```

Add a companion test immediately after it proving the clamp no longer fires on ordinary mists:

```go
// TestBuildMistCreateBody_UnderCapIsNotClamped is the regression guard for the
// defect the cap used to mask: before task-190 a 30s authored mist arrived here
// as 30_000_000 and was pinned to exactly 60_000. It must now pass through.
func TestBuildMistCreateBody_UnderCapIsNotClamped(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100020000)).SetInstance(uuid.New()).Build()
	m := r.CreateMonster(ctx, ten, f, uint32(8800002), 300, 400, 0, 5, 0, 1000, 200)

	sd := mobskill.NewModelBuilder().
		SetSkillId(uint16(monster2.SkillTypeAreaPoison)).
		SetLevel(5).
		SetX(80).
		SetDuration(30_000). // 30s authored in WZ, delivered as ms
		SetBoundingBox(-50, -30, 50, 30).
		Build()

	body := buildMistCreateBody(m, sd, byte(monster2.SkillTypeAreaPoison), 5)

	if body.Duration != 30_000 || body.DiseaseDuration != 30_000 {
		t.Errorf("duration: got %d/%d want 30000/30000 (must not clamp below the 60s cap)",
			body.Duration, body.DiseaseDuration)
	}
	if body.Duration == MistDurationCapMs {
		t.Errorf("duration was pinned to the cap (%d) — the pre-task-190 defect has returned", MistDurationCapMs)
	}
}
```

The reflect-status test at ≈`:894` uses `SetDuration(60)` and never asserts on duration, but it must not silently mean "60 ms" to a future reader. Change it to `SetDuration(60_000). // 60s in ms` and do the same for the other `SetDuration(60)` sites in that file (≈`:1014`, ≈`:1070`, ≈`:1116`) and `SetDuration(10)` at ≈`:1263` → `SetDuration(10_000)`. The `:1263` test asserts `body.Duration != 10_000`, which still holds after the input change.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestBuildMistCreateBody' -v
```

Expected: FAIL — `TestBuildMistCreateBody` reports `duration: got 10000000/10000000 want 10000/10000` (the `*1000` is still in place).

- [ ] **Step 3: Drop the `*1000` in `buildMistCreateBody`**

Replace:

```go
	durMs := int64(sd.Duration()) * int64(time.Second/time.Millisecond)
```

with:

```go
	// mobskill.Duration() is MILLISECONDS (atlas-data mobskill/reader.go —
	// the single conversion point, task-190 FR-1.1). Forward it; do not scale.
	durMs := int64(sd.Duration())
```

- [ ] **Step 4: Flip `executeStatBuff` to milliseconds**

In `executeStatBuff`, replace:

```go
	duration := time.Duration(sd.Duration()) * time.Second
```

with:

```go
	// mobskill.Duration() is MILLISECONDS (task-190 FR-1.1). Before that change
	// this multiplied ms-scale seconds and made a 20s immunity last ~5.5h.
	duration := time.Duration(sd.Duration()) * time.Millisecond
```

- [ ] **Step 5: Confirm `executeDebuff` is untouched**

```bash
git diff services/atlas-monsters/atlas.com/monsters/monster/processor.go | grep -n "sd.Duration()"
```

Expected: exactly two changed `sd.Duration()` lines (the two above). The `executeDebuff` line `duration := int32(sd.Duration())` must NOT appear in the diff. If it does, revert that hunk — FR-1.3 forbids it.

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestBuildMistCreateBody|TestExecuteStatBuff' -v
```

Expected: PASS, including `TestBuildMistCreateBody_UnderCapIsNotClamped`.

- [ ] **Step 7: Run the whole module**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go vet ./... && go test -race ./...
```

Expected: clean. Fix any other test that fails because it assumed seconds — update it with a corrected comment naming task-190, never delete it.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/
git commit -m "fix(atlas-monsters): mob-skill duration is already ms; drop double conversion (task-190 FR-1.2)"
```

---

### Task 3: atlas-maps — mist tick emits milliseconds, stale comment replaced

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go:81-86`
- Modify: `services/atlas-maps/atlas.com/maps/tasks/mist_tick_test.go` (≈`:163`, locate by content)

**Interfaces:**
- Consumes: `mist.Mist.DiseaseDuration()` returns `time.Duration` (built in `mist/processor.go:69` as `time.Duration(body.DiseaseDuration) * time.Millisecond`; Task 2 makes `body.DiseaseDuration` a real ms value).
- Produces: `applyDiseaseBody.Duration` (`int32`) on `COMMAND_TOPIC_CHARACTER_BUFF` is milliseconds.

- [ ] **Step 1: Update the test that pins the seconds contract**

In `mist_tick_test.go`, replace the comment block and assertion:

```go
	// Duration is in SECONDS (atlas-buffs' buff.NewBuff multiplies by
	// time.Second). 30s mist disease -> 30, not 30000ms. The previous
	// 30000 expectation pinned a bug where AREA_POISON DoTs persisted
	// for hours instead of the configured mist disease duration.
	require.Equal(t, int32(30), cmd.Body.Duration)
```

with:

```go
	// Duration is MILLISECONDS. atlas-buffs' buff.NewBuff has computed
	// expiresAt = now + duration*time.Millisecond since task-054 (197324e40);
	// the contract owner is
	// services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go on
	// ApplyCommandBody.Duration. A 30s mist disease is 30000, not 30. This
	// expectation reverses the one commit 11e07dfa7 introduced (task-190).
	require.Equal(t, int32(30_000), cmd.Body.Duration)
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./tasks/ -run TestMistTick -v
```

Expected: FAIL — got `30`, want `30000`.

- [ ] **Step 3: Replace the stale comment and the conversion**

In `mist_tick.go`, replace:

```go
			// atlas-buffs treats Duration as SECONDS (buff.NewBuff multiplies
			// by time.Second). Sending milliseconds here turned a 15s mist
			// poison into a 15000-second buff (~4h10m), so the DoT never
			// stopped ticking. Match atlas-monsters' convention (executeDebuff
			// passes int32(sd.Duration()) — i.e. raw seconds from mob skill
			// data).
			Duration: int32(m.DiseaseDuration() / time.Second),
```

with:

```go
			// MILLISECONDS. Contract owner:
			// services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go
			// (ApplyCommandBody.Duration). atlas-buffs has computed
			// expiresAt = now + duration*time.Millisecond since task-054
			// (197324e40, 2026-05-03).
			//
			// This REVERSES commit 11e07dfa7 ("mist tick publishes disease
			// duration in seconds"), which was correct against the pre-task-054
			// contract and was silently inverted by task-054 one day later. Do
			// not flip it back: tools/buff-duration-guard.sh fails CI on a
			// seconds-valued emitter. (task-190 FR-1.2 / FR-1.4)
			Duration: int32(m.DiseaseDuration().Milliseconds()),
```

- [ ] **Step 4: Check whether the `time` import is still needed**

```bash
grep -n "time\." services/atlas-maps/atlas.com/maps/tasks/mist_tick.go
```

Expected: `time.Now()` (×2), `time.Duration`, `time.Millisecond` in `SleepTime()` remain — keep the import. If the grep shows no remaining uses, remove the import.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./tasks/ -run TestMistTick -v
```

Expected: PASS.

- [ ] **Step 6: Run the whole module**

```bash
cd services/atlas-maps/atlas.com/maps && go build ./... && go vet ./... && go test -race ./...
```

Expected: clean. `mist/model_test.go:39` (`require.Equal(t, 30*time.Second, m.DiseaseDuration())`) is **unaffected** — it asserts the model's `time.Duration` value built from a ms input, and that unit never changed.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/tasks/
git commit -m "fix(atlas-maps): mist disease duration is milliseconds; reverses 11e07dfa7 (task-190 FR-1.2/1.4)"
```

---

### Task 4: atlas-ui — render the now-milliseconds duration as seconds

**Files:**
- Modify: `services/atlas-ui/src/components/features/monsters/MonsterSkillChip.tsx:114`

**Interfaces:**
- Consumes: `MobSkillDetailAttributes.duration` (`number`) is now milliseconds (Task 1).
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Fix the display**

Replace:

```tsx
  if (a.duration > 0) rows.push({ label: "Duration", value: `${a.duration}s` });
```

with:

```tsx
  // `duration` is MILLISECONDS from atlas-data (task-190 FR-1.1 — mobskill
  // reader.go is the single seconds→ms conversion point). Render as seconds.
  if (a.duration > 0)
    rows.push({
      label: "Duration",
      value: `${(a.duration / 1000).toLocaleString()}s`,
    });
```

Leave the adjacent `a.interval` row alone — `interval` is a separate WZ field that task-190 does not touch.

- [ ] **Step 2: Verify no other in-repo consumer scales this field**

```bash
grep -rn "\.duration" services/atlas-ui/src --include=*.ts --include=*.tsx | grep -i "mobskill\|mob_skill\|MobSkill"
grep -rn "duration" services/atlas-ui/src/services/api/mob-skills.service.ts services/atlas-ui/src/lib/hooks/useMobSkillData.ts 2>/dev/null
```

Expected: `mob-skills.service.ts` only types the field; `useMobSkillData.ts` only reads names. If a third consumer scales it, fix that too and note it in the commit body.

- [ ] **Step 3: Build and test the UI**

```bash
cd services/atlas-ui && source ~/.nvm/nvm.sh && nvm use 22 && npm run build
```

Expected: build succeeds (`npm run build` type-checks tests too — vitest alone is not sufficient verification here).

```bash
cd services/atlas-ui && npx vitest run
```

Expected: the existing suite passes at its current baseline.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-ui/src/components/features/monsters/MonsterSkillChip.tsx
git commit -m "fix(atlas-ui): render mob-skill duration ms as seconds (task-190)"
```

---

### Task 5: Contract authority comment + the seven-producer audit

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` — the authoritative unit statement
- Modify (pointer comment only): the six producer-side copies listed below
- Create: `docs/tasks/task-190-disease-duration-cancel-debuff/producer-audit.md`

**Interfaces:**
- Consumes: nothing.
- Produces: the exact pointer-comment string reused by Task 9.

- [ ] **Step 1: Write the authoritative statement**

In `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`, replace the `ApplyCommandBody` declaration's `Duration` line with a documented one:

```go
type ApplyCommandBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Level    byte         `json:"level"`
	// Duration is MILLISECONDS. This is the single authoritative statement of
	// the COMMAND_TOPIC_CHARACTER_BUFF duration unit: atlas-buffs is the
	// consumer that defines it (buff.NewBuff computes
	// expiresAt = now + duration*time.Millisecond), so the unit is its property
	// to declare. Every producer's local copy of this struct carries a one-line
	// pointer back here rather than restating the rule — three separate
	// commits (11e07dfa7, 197324e40, 88d270bf1) flipped it in prose alone.
	// tools/buff-duration-guard.sh fails CI on a seconds-valued emitter.
	// (task-190 FR-3.1)
	Duration int32        `json:"duration"`
	Changes  []StatChange `json:"changes"`
	// Accumulate, when true, stores each change as its own independently-timed
	// buff under the same sourceId (per-stat keying) instead of replacing the
	// whole sourceId buff. Used by the Beholder Hex sweep so its buffs accumulate
	// one-at-a-time (original-GMS behavior). Default false preserves the standard
	// replace-by-sourceId semantics for every other producer.
	Accumulate bool `json:"accumulate,omitempty"`
}
```

- [ ] **Step 2: Add the pointer comment to every producer copy**

Locate every producer-side copy of the apply body:

```bash
grep -rln "COMMAND_TOPIC_CHARACTER_BUFF" services/ --include=*.go | sort -u
grep -rn "Duration int32 *\`json:\"duration\"\`\|Duration: " services/ --include=*.go | grep -v atlas-buffs | grep -v _test
```

For each producer struct field named `Duration` carrying the json tag `duration` on a character-buff APPLY body, insert exactly this line immediately above the field:

```go
	// milliseconds — contract owner: atlas-buffs kafka/message/character/kafka.go (task-190)
```

Known sites (confirm each with the greps above; add any the greps surface):
- `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go` — `ApplyCommandBody.Duration`
- `services/atlas-monsters/atlas.com/monsters/monster/disease.go` — `applyDiseaseBody.Duration`
- `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go` — `applyDiseaseBody.Duration` (declared in that file or a sibling; find it with `grep -rn "type applyDiseaseBody" services/atlas-maps`)
- `services/atlas-consumables/atlas.com/consumables/...`
- `services/atlas-summons/atlas.com/summons/...`
- `services/atlas-messages/atlas.com/messages/...`

Do **not** add the comment to `atlas-saga-orchestrator` — it produces only `CANCEL_ALL` and has no duration field.

- [ ] **Step 3: Write the audit document**

Create `docs/tasks/task-190-disease-duration-cancel-debuff/producer-audit.md`. **Re-confirm every row with a real file:line read — the table below is the design's lead list, not the record.** Replace any lead that a read contradicts.

```markdown
# FR-1.5 — COMMAND_TOPIC_CHARACTER_BUFF producer audit

Every service producing onto `COMMAND_TOPIC_CHARACTER_BUFF`, audited for the
seconds-into-a-milliseconds-field defect. Contract owner:
`services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`
(`ApplyCommandBody.Duration` — milliseconds).

| Service | Verdict | Evidence (file:line) | Notes |
|---|---|---|---|
| atlas-monsters | **fixed** | `monster/processor.go:<line>` (`buildMistCreateBody`), `:<line>` (`executeStatBuff`), `:<line>` (`executeDebuff`) | Two double-conversions removed; `executeDebuff` forwards verbatim and needed no edit (FR-1.3). |
| atlas-maps | **fixed** | `tasks/mist_tick.go:<line>` | Divided ms back to seconds; reverses `11e07dfa7`. |
| atlas-channel | correct | `skill/handler/common.go:<line>`, `skill/handler/mysticdoor.go:<line>` | Every `buff.Apply` duration is `effect.Model.Duration()` (ms since task-054). `HideBuffDuration` / `MountBuffDuration` are `math.MaxInt32` sentinels — unit-agnostic. |
| atlas-consumables | correct | `consumable/processor.go:<line>`, `:<line>` | Already documents and uses ms (task-140, `88d270bf1`). |
| atlas-summons | correct | `data/skill/effect/model.go:<line>` | "duration in milliseconds"; forwarded unchanged to `buff/producer.go:<line>`. |
| atlas-messages | correct | `buff/processor.go:<line>` | Uses `effect.Duration()` (ms). |
| atlas-saga-orchestrator | n-a | `buff/processor.go:<line>` | Produces only `CANCEL_ALL`; no `APPLY`, no duration field. |

## Range check

Maximum authored WZ mob-skill `time` observed: **<value>** seconds, read from
<source: live atlas-data query | WZ extract path>.

The binding constraint is `int32` (`executeDebuff` narrows `uint32` →
`int32`), so overflow after the ×1000 needs `time > 2_147_483` seconds ≈
24.8 days. The observed maximum is <N>× below that bound.
```

Fill every `<…>` placeholder with a real value before committing. If a value could not be read, write `NOT VERIFIED — <what blocked it>` rather than a guess.

- [ ] **Step 4: Build and vet every touched module**

```bash
for m in atlas-buffs/atlas.com/buffs atlas-channel/atlas.com/channel atlas-monsters/atlas.com/monsters atlas-maps/atlas.com/maps atlas-consumables/atlas.com/consumables atlas-summons/atlas.com/summons atlas-messages/atlas.com/messages; do
  echo "== $m"; (cd "services/$m" && go build ./... && go vet ./...) || echo "FAIL $m";
done
```

Expected: no FAIL lines. Comment-only edits cannot break a build; a failure means a stray edit.

- [ ] **Step 5: Commit**

```bash
git add services/ docs/tasks/task-190-disease-duration-cancel-debuff/producer-audit.md
git commit -m "docs(task-190): state the buff duration contract once; audit all seven producers (FR-1.5/FR-3.1)"
```

---

### Task 6: `buffdurationguard` — the CI analyzer that fails a seconds emitter

**Files:**
- Create: `tools/buffdurationguard/go.mod`, `go.sum`
- Create: `tools/buffdurationguard/analyzer.go`
- Create: `tools/buffdurationguard/analyzer_test.go`
- Create: `tools/buffdurationguard/cmd/buffdurationguard/main.go`
- Create: `tools/buffdurationguard/testdata/src/bad/bad.go`
- Create: `tools/buffdurationguard/testdata/src/good/good.go`
- Create: `tools/buff-duration-guard.sh`
- Modify: `.github/workflows/pr-validation.yml`
- Modify: `CLAUDE.md`

**Interfaces:**
- Consumes: nothing.
- Produces: `tools/buff-duration-guard.sh` (exit 0 = clean), and the escape hatch comment `//buffdurationguard:allow <justification>`, honoured on the offending line or the line immediately above it.

**How it works.** The apply body struct is duplicated under seven local names, so the analyzer fingerprints **json tag sets**, not type names:

- **BD-1 (buff APPLY body)** — a composite literal whose struct type has fields tagged `sourceId`, `duration` and `changes`. The expression assigned to the `duration`-tagged field is checked.
- **BD-2 (mist create body)** — a composite literal whose struct type has fields tagged `diseaseDuration` and `tickIntervalMs`. The expressions assigned to the `diseaseDuration`- and `duration`-tagged fields are checked.

A checked expression is a violation if it contains `time.Second`, `time.Minute`, `time.Hour`, or the integer literal `1000`. Because the historical defect lived in a local (`durMs := int64(sd.Duration()) * int64(time.Second/time.Millisecond)`) rather than inline, the checker follows an identifier one level into the local assignments that produced it — which is exactly why it reports at the `durMs :=` line.

- [ ] **Step 1: Create the module**

```bash
mkdir -p tools/buffdurationguard/cmd/buffdurationguard tools/buffdurationguard/testdata/src/bad tools/buffdurationguard/testdata/src/good
```

`tools/buffdurationguard/go.mod`:

```
module github.com/Chronicle20/atlas/tools/buffdurationguard

go 1.25.5

require golang.org/x/tools v0.48.0

require (
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
)
```

Do **not** add this module to `go.work` — `tools/goroutineguard` and `tools/rediskeyguard` are also absent from it, and the guard script runs with `GOWORK=off`.

- [ ] **Step 2: Write the bad fixtures (the historical defects)**

`tools/buffdurationguard/testdata/src/bad/bad.go`:

```go
package bad

import "time"

type statChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

// BD-1 fingerprint: sourceId + duration + changes.
type applyDiseaseBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Level    byte         `json:"level"`
	Duration int32        `json:"duration"`
	Changes  []statChange `json:"changes"`
}

// BD-2 fingerprint: diseaseDuration + tickIntervalMs.
type createCommandBody struct {
	DiseaseDuration int64 `json:"diseaseDuration"`
	Duration        int64 `json:"duration"`
	TickIntervalMs  int64 `json:"tickIntervalMs"`
}

// historicalMistTick is atlas-maps tasks/mist_tick.go:86 as it stood before
// task-190: it divided milliseconds back down to seconds.
func historicalMistTick(d time.Duration) applyDiseaseBody {
	return applyDiseaseBody{
		SourceId: 1,
		Duration: int32(d / time.Second), // want "duration fields .* MILLISECONDS"
		Changes:  []statChange{{Type: "POISON", Amount: 80}},
	}
}

// historicalMist is atlas-monsters monster/processor.go:1068 as it stood before
// task-190: it multiplied an already-ms value by 1000. The scaling lives in a
// local, one level away from the composite literal.
func historicalMist(dur int64) createCommandBody {
	durMs := dur * int64(time.Second/time.Millisecond) // want "duration fields .* MILLISECONDS"
	return createCommandBody{
		DiseaseDuration: durMs,
		Duration:        durMs,
		TickIntervalMs:  1000,
	}
}

// inlineThousand is the other shape of the same defect.
func inlineThousand(sec int32) applyDiseaseBody {
	return applyDiseaseBody{
		SourceId: 3,
		Duration: sec * 1000, // want "duration fields .* MILLISECONDS"
		Changes:  nil,
	}
}
```

- [ ] **Step 3: Write the good fixtures (post-fix forms + a justified allow site)**

`tools/buffdurationguard/testdata/src/good/good.go`:

```go
package good

import "time"

type statChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type applyDiseaseBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Level    byte         `json:"level"`
	Duration int32        `json:"duration"`
	Changes  []statChange `json:"changes"`
}

type createCommandBody struct {
	DiseaseDuration int64 `json:"diseaseDuration"`
	Duration        int64 `json:"duration"`
	TickIntervalMs  int64 `json:"tickIntervalMs"`
}

const mistDurationCapMs int64 = 60_000

// fixedMistTick is atlas-maps after task-190.
func fixedMistTick(d time.Duration) applyDiseaseBody {
	return applyDiseaseBody{
		SourceId: 1,
		Duration: int32(d.Milliseconds()),
		Changes:  []statChange{{Type: "POISON", Amount: 80}},
	}
}

// fixedMist is atlas-monsters after task-190. The 60_000 cap is a named const,
// not a scaling factor, and TickIntervalMs is not a guarded field.
func fixedMist(dur int64) createCommandBody {
	durMs := dur
	if durMs > mistDurationCapMs {
		durMs = mistDurationCapMs
	}
	return createCommandBody{
		DiseaseDuration: durMs,
		Duration:        durMs,
		TickIntervalMs:  1000,
	}
}

// justified shows the escape hatch: an annotated site stays legal and visible.
func justified(sec int32) applyDiseaseBody {
	return applyDiseaseBody{
		SourceId: 2,
		//buffdurationguard:allow upstream field is authored in seconds by design
		Duration: sec * 1000,
		Changes:  nil,
	}
}
```

- [ ] **Step 4: Write the analyzer test**

`tools/buffdurationguard/analyzer_test.go`:

```go
package buffdurationguard_test

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/buffdurationguard"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, buffdurationguard.Analyzer, "bad", "good")
}
```

- [ ] **Step 5: Run the test to verify it fails**

```bash
cd tools/buffdurationguard && GOWORK=off go mod tidy && GOWORK=off go test ./...
```

Expected: FAIL — `undefined: buffdurationguard.Analyzer`.

- [ ] **Step 6: Write the analyzer**

`tools/buffdurationguard/analyzer.go`:

```go
// Package buffdurationguard bans seconds→milliseconds scaling in the duration
// fields of the COMMAND_TOPIC_CHARACTER_BUFF command bodies.
//
// The unit contract is owned by
// services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go
// (ApplyCommandBody.Duration — milliseconds). It has been flipped three times
// in prose alone (11e07dfa7, 197324e40, 88d270bf1); this analyzer is the
// mechanical half of task-190 FR-3.2.
//
// The body struct is duplicated under seven different local names, so the
// analyzer fingerprints json tag sets rather than type names.
package buffdurationguard

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const markerPrefix = "//buffdurationguard:allow"

const diagnostic = "buffdurationguard: duration fields on the character-buff command are MILLISECONDS; " +
	"drop the seconds-to-ms scaling. Contract owner: atlas-buffs kafka/message/character/kafka.go " +
	"(or annotate with //buffdurationguard:allow <justification>)"

// fingerprint identifies one command body by the json tags its struct carries,
// and names the tags whose assigned expressions must not scale.
type fingerprint struct {
	requires []string
	guards   []string
}

var fingerprints = []fingerprint{
	// BD-1 — the buff APPLY body.
	{requires: []string{"sourceId", "duration", "changes"}, guards: []string{"duration"}},
	// BD-2 — the mist create body.
	{requires: []string{"diseaseDuration", "tickIntervalMs"}, guards: []string{"diseaseDuration", "duration"}},
}

// bannedSelectors are the time constants that can only appear in a
// seconds(or coarser)→ms conversion.
var bannedSelectors = map[string]bool{"Second": true, "Minute": true, "Hour": true}

var Analyzer = &analysis.Analyzer{
	Name:     "buffdurationguard",
	Doc:      "bans seconds-to-milliseconds scaling in COMMAND_TOPIC_CHARACTER_BUFF duration fields",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

type lineKey struct {
	file string
	line int
}

func run(pass *analysis.Pass) (interface{}, error) {
	markers := collectMarkers(pass)
	assigns := collectAssignments(pass)
	reported := map[lineKey]bool{}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CompositeLit)(nil)}, func(n ast.Node) {
		cl := n.(*ast.CompositeLit)
		if strings.HasSuffix(pass.Fset.Position(cl.Pos()).Filename, "_test.go") {
			return
		}
		st, ok := structOf(pass, cl)
		if !ok {
			return
		}
		tags := tagSet(st)
		for _, fp := range fingerprints {
			if !hasAll(tags, fp.requires) {
				continue
			}
			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				tag, ok := jsonTagOfField(st, key.Name)
				if !ok || !contains(fp.guards, tag) {
					continue
				}
				if pos, bad := scalingPos(pass, assigns, kv.Value, 0); bad {
					report(pass, markers, reported, pos)
				}
			}
		}
	})
	return nil, nil
}

func collectMarkers(pass *analysis.Pass) map[lineKey]bool {
	markers := map[lineKey]bool{}
	for _, f := range pass.Files {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if !strings.HasPrefix(c.Text, markerPrefix) {
					continue
				}
				p := pass.Fset.Position(c.Pos())
				justification := strings.TrimSpace(strings.TrimPrefix(c.Text, markerPrefix))
				markers[lineKey{p.Filename, p.Line}] = justification != ""
			}
		}
	}
	return markers
}

// collectAssignments maps each local variable to every expression assigned to
// it. Object identity is unique per variable, so no per-function scoping is
// needed. This is what lets the checker follow `durMs` back to its RHS.
func collectAssignments(pass *analysis.Pass) map[types.Object][]ast.Expr {
	out := map[types.Object][]ast.Expr{}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok || len(as.Lhs) != len(as.Rhs) {
				return true
			}
			for i, lhs := range as.Lhs {
				id, ok := lhs.(*ast.Ident)
				if !ok {
					continue
				}
				obj := pass.TypesInfo.ObjectOf(id)
				if _, isVar := obj.(*types.Var); !isVar {
					continue
				}
				out[obj] = append(out[obj], as.Rhs[i])
			}
			return true
		})
	}
	return out
}

func structOf(pass *analysis.Pass, cl *ast.CompositeLit) (*types.Struct, bool) {
	t := pass.TypesInfo.TypeOf(cl)
	if t == nil {
		return nil, false
	}
	st, ok := t.Underlying().(*types.Struct)
	return st, ok
}

func tagSet(st *types.Struct) map[string]bool {
	out := map[string]bool{}
	for i := 0; i < st.NumFields(); i++ {
		if name := jsonName(st.Tag(i)); name != "" {
			out[name] = true
		}
	}
	return out
}

func jsonTagOfField(st *types.Struct, fieldName string) (string, bool) {
	for i := 0; i < st.NumFields(); i++ {
		if st.Field(i).Name() == fieldName {
			name := jsonName(st.Tag(i))
			return name, name != ""
		}
	}
	return "", false
}

func jsonName(tag string) string {
	v := reflect.StructTag(tag).Get("json")
	if v == "" || v == "-" {
		return ""
	}
	return strings.Split(v, ",")[0]
}

func hasAll(set map[string]bool, want []string) bool {
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

// scalingPos reports the position of a seconds→ms scaling factor inside expr,
// following identifiers one level into the assignments that produced them.
func scalingPos(pass *analysis.Pass, assigns map[types.Object][]ast.Expr, expr ast.Expr, depth int) (token.Pos, bool) {
	var found token.Pos
	var ok bool
	ast.Inspect(expr, func(n ast.Node) bool {
		if ok {
			return false
		}
		switch e := n.(type) {
		case *ast.SelectorExpr:
			if pkg, isIdent := e.X.(*ast.Ident); isIdent && pkg.Name == "time" && bannedSelectors[e.Sel.Name] {
				found, ok = e.Pos(), true
				return false
			}
		case *ast.BasicLit:
			if e.Kind == token.INT && strings.ReplaceAll(e.Value, "_", "") == "1000" {
				found, ok = e.Pos(), true
				return false
			}
		case *ast.Ident:
			if depth >= 1 {
				return true
			}
			obj := pass.TypesInfo.ObjectOf(e)
			if obj == nil {
				return true
			}
			for _, rhs := range assigns[obj] {
				if p, bad := scalingPos(pass, assigns, rhs, depth+1); bad {
					found, ok = p, true
					return false
				}
			}
		}
		return true
	})
	return found, ok
}

func report(pass *analysis.Pass, markers map[lineKey]bool, reported map[lineKey]bool, pos token.Pos) {
	p := pass.Fset.Position(pos)
	k := lineKey{p.Filename, p.Line}
	if reported[k] {
		return
	}
	reported[k] = true

	if justified, found := markerFor(markers, p); found {
		if !justified {
			pass.Reportf(pos, "buffdurationguard: allow marker requires a justification")
		}
		return
	}
	pass.Reportf(pos, "%s", diagnostic)
}

// markerFor accepts a marker trailing on the offending line or on the line
// immediately above it.
func markerFor(markers map[lineKey]bool, pos token.Position) (justified bool, found bool) {
	if justified, found = markers[lineKey{pos.Filename, pos.Line}]; found {
		return justified, found
	}
	justified, found = markers[lineKey{pos.Filename, pos.Line - 1}]
	return justified, found
}
```

`tools/buffdurationguard/cmd/buffdurationguard/main.go`:

```go
package main

import (
	"github.com/Chronicle20/atlas/tools/buffdurationguard"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(buffdurationguard.Analyzer)
}
```

- [ ] **Step 7: Run the analyzer test to verify it passes**

```bash
cd tools/buffdurationguard && GOWORK=off go mod tidy && GOWORK=off go test ./... -v
```

Expected: PASS — all three `bad` diagnostics reported at the annotated lines, no diagnostic in `good`. If `analysistest` reports an unexpected diagnostic in `good`, the reported *position* is the thing to inspect first: `historicalMist`-style indirection reports at the assignment, not the literal.

- [ ] **Step 8: Write the runner script**

`tools/buff-duration-guard.sh`:

```bash
#!/usr/bin/env bash
# Self-test the buffdurationguard analyzer fixtures, build it once, then run it
# over every Go module under services/ and libs/. Non-empty diagnostics →
# non-zero exit. Run from the repo root. tools/ is deliberately not swept —
# the analyzer's own testdata must be allowed to contain the defective forms.
#
# Guards the COMMAND_TOPIC_CHARACTER_BUFF duration unit (milliseconds; contract
# owner atlas-buffs kafka/message/character/kafka.go). task-190 FR-3.2/FR-3.3.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUARD_SRC="$ROOT/tools/buffdurationguard"
BIN="$(mktemp -d)/buffdurationguard"

echo "self-testing buffdurationguard..."
( cd "$GUARD_SRC" && GOWORK=off go test ./... )

echo "building buffdurationguard..."
( cd "$GUARD_SRC" && GOWORK=off go build -o "$BIN" ./cmd/buffdurationguard )

rc=0
while IFS= read -r modfile; do
    moddir="$(dirname "$modfile")"
    echo "buffdurationguard: $moddir"
    if ! ( cd "$moddir" && "$BIN" ./... ); then
        rc=1
    fi
done < <(find "$ROOT/services" "$ROOT/libs" -name go.mod -not -path '*/node_modules/*')

if [ "$rc" -ne 0 ]; then
    echo "buffdurationguard: FAIL — seconds-valued buff duration emitter found"
    echo "  The COMMAND_TOPIC_CHARACTER_BUFF duration field is MILLISECONDS."
    echo "  Contract owner: services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go"
fi
exit $rc
```

```bash
chmod +x tools/buff-duration-guard.sh
```

- [ ] **Step 9: Run the guard over the real tree and triage every diagnostic**

```bash
./tools/buff-duration-guard.sh
```

Expected: exit 0, because Tasks 2 and 3 already removed both historical defects.

If it reports a site in a service this plan has not touched, that is a **real find** — it means FR-1.5's audit missed one. Do not silence it with a marker reflexively. For each diagnostic: read the site, decide whether the value is genuinely seconds (fix it, add a test, and update `producer-audit.md`) or genuinely milliseconds with an incidental `1000` (add `//buffdurationguard:allow <why>` with a real justification). Record every decision in `producer-audit.md`.

- [ ] **Step 10: Demonstrate the guard catches the historical defect**

The acceptance criterion is "a deliberately-reintroduced seconds emitter fails CI — **demonstrated**, not asserted." Do it against the real tree and capture the output:

```bash
# Reintroduce the exact pre-task-190 form in atlas-maps.
git stash list >/dev/null   # (do not use bare git stash; see Global Constraints)
sed -i 's|Duration: int32(m.DiseaseDuration().Milliseconds()),|Duration: int32(m.DiseaseDuration() / time.Second),|' \
  services/atlas-maps/atlas.com/maps/tasks/mist_tick.go
./tools/buff-duration-guard.sh; echo "exit=$?"
```

Expected: a diagnostic naming `tasks/mist_tick.go` and `exit=1`.

```bash
git checkout -- services/atlas-maps/atlas.com/maps/tasks/mist_tick.go
./tools/buff-duration-guard.sh; echo "exit=$?"
```

Expected: `exit=0`.

Paste both transcripts verbatim into `docs/tasks/task-190-disease-duration-cancel-debuff/producer-audit.md` under a `## FR-3.2 guard demonstration` heading.

- [ ] **Step 11: Wire the guard into CI**

In `.github/workflows/pr-validation.yml`, add a job immediately after the `goroutine-guard` job (copy its shape exactly):

```yaml
  # ============================================
  # Buff Duration Guard
  # Self-tests + builds the buffdurationguard
  # analyzer and runs it over every Go module
  # under services/ and libs/. Fails on any
  # seconds→ms scaling in a
  # COMMAND_TOPIC_CHARACTER_BUFF duration field
  # (task-190 FR-3.2).
  # ============================================
  buff-duration-guard:
    name: Buff Duration Guard
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: buff duration guard
        run: ./tools/buff-duration-guard.sh
```

Then add `buff-duration-guard` to the aggregate job's `needs:` list (the long list at ≈`:713` that already contains `goroutine-guard`). Inspect the aggregate job body around ≈`:729` — if it explicitly reads each guard's `.result` into a shell variable, add the matching line:

```yaml
          BUFF_DURATION_GUARD_RESULT="${{ needs.buff-duration-guard.result }}"
```

and extend whatever pass/fail expression consumes those variables so the new one is actually checked. A job in `needs:` whose result is never inspected is a silent no-op.

- [ ] **Step 12: Document the guard in CLAUDE.md**

In the repo-root `CLAUDE.md`, under **Build & Verification**, append after item 10:

```markdown
11. **`tools/buff-duration-guard.sh` clean from the repo root.** Bans
    seconds→milliseconds scaling in the `duration` fields of
    `COMMAND_TOPIC_CHARACTER_BUFF` command bodies (task-190 FR-3.2). The unit
    contract is owned by
    `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`
    (`ApplyCommandBody.Duration` — milliseconds); it has been flipped three
    times in prose alone. Fingerprints json tag sets, not type names, because
    the body struct is duplicated under seven local names. Escape hatch:
    `//buffdurationguard:allow <justification>`. Runs alongside `go vet ./...`.
```

- [ ] **Step 13: Commit**

```bash
git add tools/buffdurationguard tools/buff-duration-guard.sh .github/workflows/pr-validation.yml CLAUDE.md docs/tasks/task-190-disease-duration-cancel-debuff/producer-audit.md
git commit -m "feat(tools): buffdurationguard analyzer fails CI on seconds-valued buff durations (task-190 FR-3.2/3.3)"
```

---

### Task 7: `CANCEL_DEBUFF` codec — opcode-only, version-invariant

**Files:**
- Create: `libs/atlas-packet/character/serverbound/cancel_debuff.go`
- Create: `libs/atlas-packet/character/serverbound/cancel_debuff_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `serverbound.CancelDebuffHandle` (`const string = "CancelDebuffHandle"`) — the handler-map key used in Task 10 and the `"handler"` value in every template in Task 12. `serverbound.CancelDebuff` — a zero-field struct with `Operation() string`, `String() string`, `Encode(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`, and `Decode(logrus.FieldLogger, context.Context) func(*request.Reader, map[string]interface{})`.

- [ ] **Step 1: Write the failing test**

`libs/atlas-packet/character/serverbound/cancel_debuff_test.go`:

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CANCEL_DEBUFF is CWvsContext::CheckTemporaryStatDuration: the client finds a
// locally-expired temporary stat via SecondaryStat::CheckByTime, computes the
// expired mask — and then does NOT transmit it. Every client examined
// constructs COutPacket(opcode) and calls SendPacket with no intervening
// encode calls, so the body is empty on all ten versions and there is nothing
// to version-gate. Evidence: docs/tasks/task-190-disease-duration-cancel-debuff/
// investigation.md §8.1/§8.2.
//
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v48 ida=0x71b126
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v61 ida=0x84374e
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v72 ida=0x91914f
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v79 ida=0x96ad48
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v83 ida=0xa20935
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v84 ida=0xa6bd3a
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v87 ida=0xab7fd7
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v95 ida=0x9f2d30
// packet-audit:verify packet=character/serverbound/CancelDebuff version=jms_v185 ida=0xb0783e
func TestCancelDebuffRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CancelDebuff{}
			output := CancelDebuff{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestCancelDebuffEmptyBodyAllVersions pins the empty body per version. The
// v48 client builds the packet with the three-argument
// COutPacket::COutPacket(v5, 78, 0) constructor; that third argument is a
// client-side construction detail with no wire consequence, so v48 encodes the
// same zero bytes as every other version.
func TestCancelDebuffEmptyBodyAllVersions(t *testing.T) {
	versions := []struct {
		name   string
		region string
		major  uint16
		minor  uint16
	}{
		{"gms_v48", "GMS", 48, 1},
		{"gms_v61", "GMS", 61, 1},
		{"gms_v72", "GMS", 72, 1},
		{"gms_v79", "GMS", 79, 1},
		{"gms_v83", "GMS", 83, 1},
		{"gms_v84", "GMS", 84, 1},
		{"gms_v87", "GMS", 87, 1},
		{"gms_v92", "GMS", 92, 1},
		{"gms_v95", "GMS", 95, 1},
		{"jms_v185", "JMS", 185, 1},
	}
	for _, v := range versions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			got := pt.Encode(t, ctx, CancelDebuff{}.Encode, nil)
			if len(got) != 0 {
				t.Errorf("expected empty body, got % x", got)
			}
		})
	}
}

// TestCancelDebuffOperation pins the handler-map key. atlas-channel binds the
// handler by NAME through tenant socket config, never by opcode — 0x63 means
// CANCEL_DEBUFF at v83/v84 but is the calc-damage-stat request at v61, so a
// hard-coded opcode would mis-route (FR-2.3.2, DOM-25).
func TestCancelDebuffOperation(t *testing.T) {
	if got := (CancelDebuff{}).Operation(); got != CancelDebuffHandle {
		t.Errorf("operation: got %q, want %q", got, CancelDebuffHandle)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd libs/atlas-packet && go test ./character/serverbound/ -run TestCancelDebuff -v
```

Expected: FAIL — `undefined: CancelDebuff`.

- [ ] **Step 3: Write the codec**

`libs/atlas-packet/character/serverbound/cancel_debuff.go`:

```go
package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

const CancelDebuffHandle = "CancelDebuffHandle"

// CancelDebuff - CWvsContext::CheckTemporaryStatDuration
//
// A bare "re-evaluate my temporary stats" nudge with ZERO payload. The client
// computes its locally-expired stat mask via SecondaryStat::CheckByTime and
// then discards it rather than transmitting it, on every one of the ten client
// versions examined (investigation.md §8.1). Do not grow a stat mask, a skill
// id, or a tick field onto this struct — there is nothing on the wire to read.
//
// Because the body is empty everywhere, there is nothing to version-gate. The
// 8-vs-16-byte mask split between v48 and v61+ belongs to the CLIENTBOUND
// reply and is already handled by legacyGmsMask in
// libs/atlas-packet/model/character_temporary_stat.go. (task-190 FR-2.1/2.2)
type CancelDebuff struct{}

func (m CancelDebuff) Operation() string {
	return CancelDebuffHandle
}

func (m CancelDebuff) String() string {
	return ""
}

func (m CancelDebuff) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *CancelDebuff) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd libs/atlas-packet && go test ./character/serverbound/ -run TestCancelDebuff -v
```

Expected: PASS.

- [ ] **Step 5: Run the whole library**

```bash
cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./...
```

Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/serverbound/cancel_debuff.go libs/atlas-packet/character/serverbound/cancel_debuff_test.go
git commit -m "feat(atlas-packet): CANCEL_DEBUFF serverbound codec, opcode-only body (task-190 FR-2.1/2.2)"
```

---

### Task 8: atlas-channel — the per-character throttle registry

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/statreset/registry.go`
- Create: `services/atlas-channel/atlas.com/channel/character/statreset/registry_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `statreset.Window` (`time.Duration = 1000 * time.Millisecond`)
  - `statreset.GetRegistry() *statreset.Registry`
  - `func (*Registry) Allow(t tenant.Model, characterId uint32, now time.Time) bool`
  - `func (*Registry) ClearCharacter(t tenant.Model, characterId uint32)`

Tasks 10 uses both methods.

- [ ] **Step 1: Write the failing test**

`services/atlas-channel/atlas.com/channel/character/statreset/registry_test.go`:

```go
package statreset

import (
	"testing"
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mkTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

// TestAllow_FirstNudgePasses — recovery latency must be one packet, not one
// window. A character with no entry (fresh pod, reconnect, or a quiet period)
// is always honoured immediately.
func TestAllow_FirstNudgePasses(t *testing.T) {
	r := GetRegistry()
	tm := mkTenant(t)
	now := time.Unix(1_700_000_000, 0)

	if !r.Allow(tm, 1001, now) {
		t.Error("first nudge must pass")
	}
}

// TestAllow_ThrottlesInsideWindow reproduces the live v83 send rate: the
// client never advances its own m_tLastStatResetRequest anchor on v72–v87, so
// it emits once per frame (~30/s) indefinitely. The server bound must be
// independent of the client's advisory 200ms floor.
func TestAllow_ThrottlesInsideWindow(t *testing.T) {
	r := GetRegistry()
	tm := mkTenant(t)
	now := time.Unix(1_700_000_100, 0)

	if !r.Allow(tm, 1002, now) {
		t.Fatal("first nudge must pass")
	}
	honoured := 1
	// 30 nudges/second for one second, the measured unthrottled rate.
	for i := 1; i <= 30; i++ {
		if r.Allow(tm, 1002, now.Add(time.Duration(i)*33*time.Millisecond)) {
			honoured++
		}
	}
	if honoured != 1 {
		t.Errorf("honoured %d nudges inside a %v window, want 1", honoured, Window)
	}
}

// TestAllow_PassesAfterWindow — one honoured nudge per window, so a genuinely
// stuck client still recovers 10x faster than the 10s fleet sweep.
func TestAllow_PassesAfterWindow(t *testing.T) {
	r := GetRegistry()
	tm := mkTenant(t)
	now := time.Unix(1_700_000_200, 0)

	if !r.Allow(tm, 1003, now) {
		t.Fatal("first nudge must pass")
	}
	if r.Allow(tm, 1003, now.Add(Window-time.Millisecond)) {
		t.Error("nudge just inside the window must be dropped")
	}
	if !r.Allow(tm, 1003, now.Add(Window)) {
		t.Error("nudge at the window boundary must pass")
	}
}

// TestAllow_IsolatesCharactersAndTenants — the key is (tenant, character);
// one wedged client must not throttle anyone else.
func TestAllow_IsolatesCharactersAndTenants(t *testing.T) {
	r := GetRegistry()
	tmA := mkTenant(t)
	tmB := mkTenant(t)
	now := time.Unix(1_700_000_300, 0)

	if !r.Allow(tmA, 1004, now) {
		t.Fatal("tenant A character 1004 first nudge must pass")
	}
	if !r.Allow(tmA, 1005, now) {
		t.Error("a different character must not share throttle state")
	}
	if !r.Allow(tmB, 1004, now) {
		t.Error("the same character id under a different tenant must not share throttle state")
	}
}

// TestClearCharacter_ResetsState — called from the socket destroyer; without
// it the map leaks one entry per character ever seen by the pod.
func TestClearCharacter_ResetsState(t *testing.T) {
	r := GetRegistry()
	tm := mkTenant(t)
	now := time.Unix(1_700_000_400, 0)

	if !r.Allow(tm, 1006, now) {
		t.Fatal("first nudge must pass")
	}
	if r.Allow(tm, 1006, now.Add(10*time.Millisecond)) {
		t.Fatal("second nudge inside the window must be dropped")
	}

	r.ClearCharacter(tm, 1006)

	if !r.Allow(tm, 1006, now.Add(20*time.Millisecond)) {
		t.Error("after ClearCharacter the next nudge must pass")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./character/statreset/ -v
```

Expected: FAIL — no non-test Go files / `undefined: GetRegistry`.

- [ ] **Step 3: Write the registry**

`services/atlas-channel/atlas.com/channel/character/statreset/registry.go`:

```go
// Package statreset rate-limits the serverbound CANCEL_DEBUFF nudge
// (CWvsContext::CheckTemporaryStatDuration) per (tenant, character).
//
// Why the server must bound this independently: every client gates on
// `tick - m_tLastStatResetRequest > 200`, but v72, v79, v83, v84 and v87 never
// assign that anchor anywhere in the function. On those five the guard latches
// open 200ms after the last temporary-stat change and the client then sends
// once per frame, indefinitely — the ~30ms spacing and ~1,500 packets measured
// live on GMS 83.1. The client's 200ms floor is advisory only. (task-190 NFR-2)
//
// Why in-process state is correct rather than partial: a character's socket
// session lives on exactly one atlas-channel pod, so a per-pod map is the whole
// view, not a shard of it. On reconnect to a different pod the entry is absent
// and the first nudge passes — which is the desired behaviour anyway.
package statreset

import (
	"sync"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Window is the minimum spacing between honoured nudges from one character.
//
// 1s caps a wedged or hostile client at one Kafka command per second — a ~30x
// reduction against the measured live rate — while still recovering 10x faster
// than the 10s fleet expiry sweep (atlas-buffs tasks/expiration.go). It is a
// const, not env-configurable: a tunable would add deployment surface for a
// value with no known reason to vary per tenant. Deliberate non-goal; revisit
// only if a real workload disagrees.
const Window = 1000 * time.Millisecond

type Key struct {
	Tenant      tenant.Model
	CharacterId uint32
}

type Registry struct {
	mutex sync.RWMutex
	last  map[Key]time.Time
}

var (
	registry *Registry
	once     sync.Once
)

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.last = make(map[Key]time.Time)
	})
	return registry
}

// Allow reports whether this nudge should be honoured, recording the timestamp
// when it is. The first nudge after a quiet period always passes, so recovery
// latency is one packet rather than one window.
func (r *Registry) Allow(t tenant.Model, characterId uint32, now time.Time) bool {
	k := Key{Tenant: t, CharacterId: characterId}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if prev, ok := r.last[k]; ok && now.Sub(prev) < Window {
		return false
	}
	r.last[k] = now
	return true
}

// ClearCharacter drops the throttle entry for a character (session destroy).
func (r *Registry) ClearCharacter(t tenant.Model, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.last, Key{Tenant: t, CharacterId: characterId})
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./character/statreset/ -v
```

Expected: PASS, all five tests. `-race` matters — the registry is a shared singleton hit from every socket goroutine.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/statreset/
git commit -m "feat(atlas-channel): per-character CANCEL_DEBUFF throttle registry (task-190 FR-2.10/NFR-2)"
```

---

### Task 9: atlas-channel — the `EXPIRE` command

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/processor.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `buff.CommandTypeExpire` (`const string = "EXPIRE"`) and `buff.ExpireCommandBody struct{}` in `kafka/message/buff` — Task 11 mirrors both in atlas-buffs with identical values.
  - `ExpireCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message]`
  - `Processor.Expire(f field.Model, characterId uint32) error` — Task 10 calls this.

- [ ] **Step 1: Add the command type and body**

In `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go`, extend the const block:

```go
const (
	EnvCommandTopic            = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply           = "APPLY"
	CommandTypeCancel          = "CANCEL"
	CommandTypeCancelByTypes   = "CANCEL_BY_TYPES"
	CommandTypeUpdateStatValue = "UPDATE_STAT_VALUE"
	// CommandTypeExpire asks atlas-buffs to re-evaluate ONE character's buffs
	// and announce whatever has genuinely lapsed. Emitted by the CANCEL_DEBUFF
	// handler. Named EXPIRE rather than RECONCILE because the server does not
	// diff against anything the client claims — the packet carries no payload;
	// it prunes against server-side expiresAt and announces the result.
	// (task-190 FR-2.6.1)
	CommandTypeExpire = "EXPIRE"
	...
)
```

and add the body next to `CancelByTypesCommandBody`:

```go
// ExpireCommandBody is deliberately empty: CANCEL_DEBUFF carries no payload,
// so there is nothing for the client to name. The worst a client can assert is
// "please re-check me", and atlas-buffs answers only with buffs that have
// genuinely lapsed. (task-190 FR-2.2 / NFR-2)
type ExpireCommandBody struct{}
```

Also add the FR-3.1 pointer comment on this file's `ApplyCommandBody.Duration` if Task 5 did not already place it:

```go
	// milliseconds — contract owner: atlas-buffs kafka/message/character/kafka.go (task-190)
	Duration int32        `json:"duration"`
```

- [ ] **Step 2: Add the producer**

In `services/atlas-channel/atlas.com/channel/character/buff/producer.go`, after `CancelByTypesCommandProvider`:

```go
// ExpireCommandProvider asks atlas-buffs to sweep ONE character's buffs. The
// world rides in the envelope: the channel knows the live session's world, and
// that is authoritative for an in-session character (the fleet sweep instead
// reads world from the registry model). (task-190 FR-2.6)
func ExpireCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.ExpireCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeExpire,
		Body:        buff.ExpireCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 3: Add the processor method**

In `services/atlas-channel/atlas.com/channel/character/buff/processor.go`, add to the `Processor` interface, immediately after `CancelByTypes`:

```go
	Expire(f field.Model, characterId uint32) error
```

and the implementation after `CancelByTypes`'s:

```go
// Expire asks atlas-buffs to re-evaluate this character's buffs and announce
// whatever has lapsed. Triggered by the client's CANCEL_DEBUFF nudge, which
// carries no payload — so this cannot and must not cancel by name. A sweep
// that finds nothing lapsed emits nothing (FR-2.9 / NFR-2.1). (task-190)
func (p *ProcessorImpl) Expire(f field.Model, characterId uint32) error {
	p.l.Debugf("Character [%d] requesting buff expiry sweep.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(ExpireCommandProvider(f, characterId))
}
```

- [ ] **Step 4: Build and vet**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...
```

Expected: clean. `var _ Processor = (*ProcessorImpl)(nil)` in that file will fail the build if the method signature drifts from the interface — that assertion is the check.

- [ ] **Step 5: Run the module tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./character/... ./kafka/...
```

Expected: clean. If a test elsewhere implements `buff.Processor` as a fake, it now needs an `Expire` method — add it there rather than loosening the interface.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go services/atlas-channel/atlas.com/channel/character/buff/
git commit -m "feat(atlas-channel): EXPIRE command for per-character buff reconcile (task-190 FR-2.6)"
```

---

### Task 10: atlas-channel — the `CANCEL_DEBUFF` handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cancel_debuff.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cancel_debuff_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/init.go` (the destroyer, ≈`:49-55`)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (≈`:880`)

**Interfaces:**
- Consumes: `serverbound.CancelDebuff` / `serverbound.CancelDebuffHandle` (Task 7); `statreset.GetRegistry().Allow` / `.ClearCharacter` (Task 8); `buff.Processor.Expire` (Task 9).
- Produces: `handler.CancelDebuffHandleFunc` with the standard handler signature `func(logrus.FieldLogger, context.Context, writer.Producer) func(session.Model, *request.Reader, map[string]interface{})`.

- [ ] **Step 1: Write the failing test**

`services/atlas-channel/atlas.com/channel/socket/handler/character_cancel_debuff_test.go`, following the `mount_food_test.go` precedent:

```go
package handler

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestCancelDebuffDecode pins the wire format CancelDebuffHandleFunc consumes:
// nothing. CWvsContext::CheckTemporaryStatDuration constructs COutPacket(opcode)
// and sends it with no encode calls on all ten client versions
// (investigation.md §8.1), so the decode must consume zero bytes and must not
// fail on an empty reader.
func TestCancelDebuffDecode(t *testing.T) {
	req := request.Request([]byte{})
	reader := request.NewRequestReader(&req, 0)

	p := charsb.CancelDebuff{}
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.Operation() != charsb.CancelDebuffHandle {
		t.Errorf("operation: got %q, want %q", p.Operation(), charsb.CancelDebuffHandle)
	}
	if p.String() != "" {
		t.Errorf("String(): got %q, want %q", p.String(), "")
	}
}

// TestCancelDebuffHandleFuncSymbol verifies the handler constructor returns a
// non-nil closure with the standard handler signature.
func TestCancelDebuffHandleFuncSymbol(t *testing.T) {
	got := CancelDebuffHandleFunc(logrus.New(), context.Background(), nil)
	if got == nil {
		t.Fatal("CancelDebuffHandleFunc returned nil closure")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestCancelDebuff -v
```

Expected: FAIL — `undefined: CancelDebuffHandleFunc`.

- [ ] **Step 3: Write the handler**

`services/atlas-channel/atlas.com/channel/socket/handler/character_cancel_debuff.go`:

```go
package handler

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/statreset"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CancelDebuffHandleFunc handles the client's CANCEL_DEBUFF nudge
// (CWvsContext::CheckTemporaryStatDuration): "one of my temporary stats looks
// expired — please re-evaluate me."
//
// The packet carries no payload, so the server cannot and must not cancel by
// name. It emits a per-character EXPIRE command; atlas-buffs owns the decision
// about what has genuinely lapsed and answers with the existing EXPIRED events,
// which already flow back through the buff consumer to the existing
// CharacterBuffCancel writer. A sweep that finds nothing lapsed emits nothing
// (FR-2.9 / NFR-2.1).
//
// Throttle FIRST, then emit: the amplification NFR-2 bounds is the Kafka
// message, so the guard must sit before the produce. Dropped nudges log at
// Debug — the unhandled-op line this replaces produced ~1,500 Info lines in
// under four minutes on one wedged client (NFR-4). (task-190 FR-2.5/2.10)
func CancelDebuffHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.CancelDebuff{}
		p.Decode(l, ctx)(r, readerOptions)

		t := tenant.MustFromContext(ctx)
		if !statreset.GetRegistry().Allow(t, s.CharacterId(), time.Now()) {
			l.Debugf("Throttled CANCEL_DEBUFF for character [%d].", s.CharacterId())
			return
		}

		if err := buff.NewProcessor(l, ctx).Expire(s.Field(), s.CharacterId()); err != nil {
			l.WithError(err).Errorf("Unable to request buff expiry sweep for character [%d].", s.CharacterId())
		}
	}
}
```

- [ ] **Step 4: Register the handler**

In `services/atlas-channel/atlas.com/channel/main.go`, add immediately after the `CharacterBuffCancelHandle` line (≈`:880`):

```go
	handlerMap[charsb.CancelDebuffHandle] = handler.CancelDebuffHandleFunc
```

- [ ] **Step 5: Evict the throttle entry on session destroy**

In `services/atlas-channel/atlas.com/channel/socket/init.go`, extend the destroyer closure:

```go
					socket.SetDestroyer(func(sessionId uuid.UUID) {
						sp.IfPresentById(sessionId, func(s session.Model) error {
							shopscanner.GetRegistry().ClearCharacter(t, s.CharacterId())
							// Without this the throttle map leaks one entry per
							// character ever seen by this pod (task-190).
							statreset.GetRegistry().ClearCharacter(t, s.CharacterId())
							return nil
						})
						sp.DestroyByIdWithSpan(sessionId)
					}),
```

and add `"atlas-channel/character/statreset"` to that file's import block.

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestCancelDebuff -v
```

Expected: PASS.

- [ ] **Step 7: Verify no opcode literal leaked in**

```bash
grep -rn "0x63\|0x4E\|0x5B\|0x62\|0x61\|0x66\|0x6E\|0x6F\|0x5E" \
  services/atlas-channel/atlas.com/channel/socket/handler/character_cancel_debuff.go \
  services/atlas-channel/atlas.com/channel/character/statreset/ \
  libs/atlas-packet/character/serverbound/cancel_debuff.go
```

Expected: no output. Handlers bind by name through tenant socket config (DOM-25); `0x63` is CANCEL_DEBUFF at v83/v84 but the calc-damage-stat request at v61.

- [ ] **Step 8: Build and test the whole service**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./...
```

Expected: clean.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/ services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(atlas-channel): CANCEL_DEBUFF handler with per-character throttle (task-190 FR-2.5/2.10)"
```

---

### Task 11: atlas-buffs — the per-character expiry sweep

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/character/processor.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/character/processor_test.go`

**Interfaces:**
- Consumes: `CommandTypeExpire = "EXPIRE"` and an empty `ExpireCommandBody` — must match Task 9's values byte-for-byte on the wire.
- Produces: `Processor.ExpireForCharacter(worldId world.Id, characterId uint32) error`. The fleet-wide `ExpireBuffs()` keeps its existing signature and behaviour.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-buffs/atlas.com/buffs/character/processor_test.go` (it already provides `setupProcessorTest` and `setupProcessorTestChanges`; add `"time"` to the imports):

```go
// TestProcessor_ExpireForCharacter_PrunesLapsedBuff — the CANCEL_DEBUFF
// reconcile path. Duration is MILLISECONDS, so a 1ms buff has lapsed by the
// time the sweep runs and must be pruned. (task-190 FR-2.6.1)
func TestProcessor_ExpireForCharacter_PrunesLapsedBuff(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	const characterId = uint32(5001)
	assert.NoError(t, processor.Apply(world.Id(0), channel.Id(0), characterId, 0, 1002, 1, 1, setupProcessorTestChanges(), false))
	time.Sleep(5 * time.Millisecond)

	assert.NoError(t, processor.ExpireForCharacter(world.Id(0), characterId))

	// The sweep pruned it: a second GetExpired finds nothing left to expire.
	assert.Empty(t, GetRegistry().GetExpired(ctx, characterId))
}

// TestProcessor_ExpireForCharacter_NothingExpired — FR-2.9 / NFR-2.1: a nudge
// for a character with nothing lapsed is a no-op. The live buff must survive.
func TestProcessor_ExpireForCharacter_NothingExpired(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	const characterId = uint32(5000)
	const sourceId = int32(1001)
	assert.NoError(t, processor.Apply(world.Id(0), channel.Id(0), characterId, 0, sourceId, 1, 60_000, setupProcessorTestChanges(), false))

	assert.NoError(t, processor.ExpireForCharacter(world.Id(0), characterId))

	// Still resident: cancelling it now succeeds and returns the buff.
	cancelled, err := GetRegistry().Cancel(ctx, characterId, sourceId)
	assert.NoError(t, err)
	assert.NotEmpty(t, cancelled)
}

// TestProcessor_ExpireBuffs_StillSweepsFleetWide — the shared helper must not
// change the fleet sweep's behaviour.
func TestProcessor_ExpireBuffs_StillSweepsFleetWide(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	const charA = uint32(5002)
	const charB = uint32(5003)
	assert.NoError(t, processor.Apply(world.Id(0), channel.Id(0), charA, 0, 1003, 1, 1, setupProcessorTestChanges(), false))
	assert.NoError(t, processor.Apply(world.Id(0), channel.Id(0), charB, 0, 1004, 1, 1, setupProcessorTestChanges(), false))
	time.Sleep(5 * time.Millisecond)

	assert.NoError(t, processor.ExpireBuffs())

	assert.Empty(t, GetRegistry().GetExpired(ctx, charA))
	assert.Empty(t, GetRegistry().GetExpired(ctx, charB))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run TestProcessor_Expire -v
```

Expected: FAIL — `processor.ExpireForCharacter undefined`.

- [ ] **Step 3: Refactor the sweep and add the per-character variant**

In `services/atlas-buffs/atlas.com/buffs/character/processor.go`, add to the `Processor` interface immediately after `ExpireBuffs()`:

```go
	ExpireForCharacter(worldId world.Id, characterId uint32) error
```

Replace the existing `ExpireBuffs` method with the three below (the loop body moves into `expireInto` unchanged):

```go
func (p *ProcessorImpl) ExpireBuffs() error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, c := range GetRegistry().GetCharacters(p.ctx) {
			if err := p.expireInto(buf, c.WorldId(), c.Id()); err != nil {
				return err
			}
		}
		return nil
	})
}

// ExpireForCharacter sweeps ONE character, so a single client's CANCEL_DEBUFF
// nudge does not force a fleet-wide pass. WorldId comes from the command
// envelope — the channel knows the live session's world, which is authoritative
// for an in-session character. Semantics are identical to the fleet sweep by
// construction: both call expireInto. (task-190 FR-2.6.1)
func (p *ProcessorImpl) ExpireForCharacter(worldId world.Id, characterId uint32) error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		return p.expireInto(buf, worldId, characterId)
	})
}

// expireInto prunes one character's lapsed buffs and puts one EXPIRED event per
// lapsed buff on buf. Registry.GetExpired already does prune-and-return, so no
// new expiry semantics are invented here. When nothing has lapsed it puts
// nothing, and message.Emit then emits nothing — FR-2.9 / NFR-2.1 hold
// structurally, not by an explicit guard.
func (p *ProcessorImpl) expireInto(buf *message.Buffer, worldId world.Id, characterId uint32) error {
	ebs := GetRegistry().GetExpired(p.ctx, characterId)
	for _, eb := range ebs {
		p.l.Debugf("Expired buff for character [%d] from [%d].", characterId, eb.SourceId())
		if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, eb.SourceId(), eb.Level(), eb.Duration(), eb.Changes(), eb.CreatedAt(), eb.ExpiresAt())); err != nil {
			return err
		}
	}
	if len(ebs) > 0 {
		sets := make([][]stat.Model, 0, len(ebs))
		for _, eb := range ebs {
			sets = append(sets, eb.Changes())
		}
		markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	}
	return nil
}
```

Leave the package-level `func ExpireBuffs(l, ctx)` (the per-tenant fan-out driven by `tasks/expiration.go`) untouched.

- [ ] **Step 4: Add the command type and body**

In `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`, extend the const block:

```go
	CommandTypeUpdateStatValue = "UPDATE_STAT_VALUE"
	// CommandTypeExpire asks for ONE character's buffs to be re-evaluated and
	// whatever has genuinely lapsed announced. Emitted by atlas-channel's
	// CANCEL_DEBUFF handler (task-190 FR-2.6.1). Named EXPIRE rather than
	// RECONCILE because there is no two-way diff — the client's packet carries
	// no payload; this prunes against server-side expiresAt.
	CommandTypeExpire = "EXPIRE"
```

and after `CancelByTypesCommandBody`:

```go
// ExpireCommandBody is deliberately empty: CANCEL_DEBUFF carries no payload, so
// a client cannot name anything. Honoring it unconditionally is provably safe —
// the worst assertion is "please re-check me", and only genuinely lapsed buffs
// are announced. Amplification is bounded upstream by atlas-channel's
// per-character throttle. (task-190 FR-2.2 / NFR-2)
type ExpireCommandBody struct{}
```

- [ ] **Step 5: Add the consumer arm**

In `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go`, register the handler inside `InitHandlers` after the `handleUpdateStatValue` registration:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleExpire))); err != nil {
			return err
		}
```

and add the handler function at the end of the file:

```go
// handleExpire answers a character's CANCEL_DEBUFF nudge with a per-character
// expiry sweep. Nothing lapsed ⇒ nothing emitted (task-190 FR-2.9).
func handleExpire(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ExpireCommandBody]) {
	if c.Type != character2.CommandTypeExpire {
		return
	}

	if err := character.NewProcessor(l, ctx).ExpireForCharacter(c.WorldId, c.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to expire buffs for character [%d].", c.CharacterId)
	}
}
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd services/atlas-buffs/atlas.com/buffs && go test -race ./character/ -run TestProcessor_Expire -v
```

Expected: PASS, all three.

- [ ] **Step 7: Verify the command type strings match across services**

```bash
grep -rn "CommandTypeExpire\s*=" services/atlas-channel services/atlas-buffs --include=*.go
```

Expected: exactly two lines, both `= "EXPIRE"`. A mismatch means the channel produces a message no consumer arm claims — the handler would silently do nothing.

- [ ] **Step 8: Build and test the whole service**

```bash
cd services/atlas-buffs/atlas.com/buffs && go build ./... && go vet ./... && go test -race ./...
```

Expected: clean, including the pre-existing fleet-sweep tests.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/
git commit -m "feat(atlas-buffs): per-character EXPIRE sweep for CANCEL_DEBUFF reconcile (task-190 FR-2.6.1)"
```

---

### Task 12: Route `CancelDebuffHandle` in all ten seed templates

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`, `_61_`, `_72_`, `_79_`, `_83_`, `_84_`, `_87_`, `_92_`, `_95_`, and `template_jms_185_1.json`

**Interfaces:**
- Consumes: `serverbound.CancelDebuffHandle` — the literal string `"CancelDebuffHandle"` (Task 7).
- Produces: nothing consumed by later Go tasks; Task 14's backfill doc reuses the same entries.

- [ ] **Step 1: Confirm each target opcode slot is still free**

```bash
for f in 48:0x4E 61:0x5B 72:0x62 79:0x61 83:0x63 84:0x63 87:0x66 92:0x6E 95:0x6F; do
  v="${f%%:*}"; op="${f##*:}"
  echo "== gms_$v $op"
  grep -n "\"opCode\": \"$op\"" "services/atlas-configurations/seed-data/templates/template_gms_${v}_1.json"
done
echo "== jms_185 0x5E"
grep -n "\"opCode\": \"0x5E\"" services/atlas-configurations/seed-data/templates/template_jms_185_1.json
```

Expected: every match, if any, is inside the `writers` array — never `handlers`. Confirm by opening each hit in context. If a `handlers` entry already occupies a target opcode, **stop and report**: the opcode table disagrees with `investigation.md` §8.2 and that is a finding, not something to work around.

- [ ] **Step 2: Insert the handler entry at its sorted position in each template**

For each template, insert into the `handlers` array at the position that keeps `opCode` **strictly ascending** — never appended next to a semantically-related entry (`docs/packets/TEMPLATE_CONVENTIONS.md`). Use the exact entry shape already in these files:

```json
      {
        "opCode": "0x63",
        "validator": "LoggedInValidator",
        "handler": "CancelDebuffHandle",
        "services": ["channel"]
      },
```

substituting the per-version opcode from the Global Constraints table. The design verified the neighbours; insert between them:

| Template | opCode | insert between |
|---|---|---|
| `template_gms_48_1.json` | `0x4E` | `0x4C` and `0x51` |
| `template_gms_61_1.json` | `0x5B` | `0x59` and `0x5C` |
| `template_gms_72_1.json` | `0x62` | `0x61` and `0x63` |
| `template_gms_79_1.json` | `0x61` | `0x60` and `0x62` |
| `template_gms_83_1.json` | `0x63` | `0x62` and `0x64` |
| `template_gms_84_1.json` | `0x63` | `0x62` and `0x64` |
| `template_gms_87_1.json` | `0x66` | `0x65` and `0x67` |
| `template_gms_92_1.json` | `0x6E` | `0x6C` and `0x6F` |
| `template_gms_95_1.json` | `0x6F` | `0x6E` and `0x72` |
| `template_jms_185_1.json` | `0x5E` | `0x5D` and `0x5F` |

Edit each file individually (per-file Edit, not a shell patch loop). **Every entry must carry `"validator": "LoggedInValidator"`** — a handler entry without a validator is silently dropped at load time, which looks exactly like the packet never arriving.

Do **not** touch `template_gms_12_1.json`: it routes 24 handlers (login, character select, map/channel change, move, chat, inventory move, info request, monster movement, NPC action, summon) and no buff, skill, or attack handler at all. `CANCEL_DEBUFF` is outside that template's entire feature surface.

- [ ] **Step 3: Verify all ten landed with a validator**

```bash
grep -c "CancelDebuffHandle" services/atlas-configurations/seed-data/templates/*.json
grep -B2 "CancelDebuffHandle" services/atlas-configurations/seed-data/templates/*.json | grep -c "LoggedInValidator"
```

Expected: exactly ten files reporting `1` (and `template_gms_12_1.json` reporting `0`), and ten `LoggedInValidator` lines.

- [ ] **Step 4: Verify the JSON is still well-formed**

```bash
for f in services/atlas-configurations/seed-data/templates/template_*.json; do
  python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$f" || echo "INVALID: $f"
done
```

Expected: no `INVALID` lines.

- [ ] **Step 5: Run the opcode-order guard**

```bash
./tools/template-opcode-order-guard.sh
```

Expected: exit 0. A failure means an entry landed out of ascending order — move it, do not sort the whole array (that would produce a huge spurious diff).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(atlas-configurations): route CancelDebuffHandle in all ten templates (task-190 FR-2.7)"
```

---

### Task 13: Packet registry entries and the coverage matrix

**Files:**
- Modify: `docs/packets/registry/gms_v48.yaml`, `gms_v61.yaml`, `gms_v72.yaml`, `gms_v79.yaml`
- Modify: `docs/packets/audits/status.json`, `docs/packets/audits/STATUS.md` (regenerated, not hand-edited)

**Interfaces:**
- Consumes: the `packet-audit:verify` markers written in Task 7's `cancel_debuff_test.go`.
- Produces: nine matrix cells at `verified`.

**Why this is a correction, not bookkeeping.** `status.json` currently records `CANCEL_DEBUFF` as `"state": "n-a", "opcode": -1` for gms_v48/v61/v72/v79 — the matrix *asserts the packet does not exist* on those clients. The IDB pass proves it does on all four. The n-a consistency gate will only let the state change once the registry says the opcode exists.

- [ ] **Step 1: Add the four registry entries**

Confirm the decimal addresses first:

```bash
for h in 0x71b126 0x84374e 0x91914f 0x96ad48; do printf "%s = %d\n" "$h" "$h"; done
```

Expected: `7450918`, `8664910`, `9539919`, `9874760`.

Insert into each registry file at the position that keeps the file's existing ordering convention (check the neighbours of the target opcode in the `serverbound` entries of that file before inserting):

`docs/packets/registry/gms_v48.yaml`:

```yaml
- op: CANCEL_DEBUFF
  direction: serverbound
  opcode: 78
  fname: CWvsContext::CheckTemporaryStatDuration
  provenance: ida-discovered
  ida:
    address: 7450918
  note: 'Empty body: COutPacket(78) then SendPacket with no intervening encode calls. The v48 client uses the three-argument COutPacket::COutPacket(v5, 78, 0) constructor — a client-side construction detail with no wire consequence. Corrects a prior n-a assertion (task-190).'
```

`docs/packets/registry/gms_v61.yaml`:

```yaml
- op: CANCEL_DEBUFF
  direction: serverbound
  opcode: 91
  fname: CWvsContext::CheckTemporaryStatDuration
  provenance: ida-discovered
  ida:
    address: 8664910
  note: 'Empty body. Note 0x63 is NOT this packet at v61 — there it is the calc-damage-stat request emitted by OnTemporaryStatReset, so opcodes must come from tenant config per version. Corrects a prior n-a assertion (task-190).'
```

`docs/packets/registry/gms_v72.yaml`:

```yaml
- op: CANCEL_DEBUFF
  direction: serverbound
  opcode: 98
  fname: CWvsContext::CheckTemporaryStatDuration
  provenance: ida-discovered
  ida:
    address: 9539919
  note: 'Empty body. v72 IDB left the function unnamed (sub_91914F); renamed during the task-190 pass. Client never assigns m_tLastStatResetRequest here, so its 200ms self-throttle latches open and it sends once per frame. Corrects a prior n-a assertion (task-190).'
```

`docs/packets/registry/gms_v79.yaml`:

```yaml
- op: CANCEL_DEBUFF
  direction: serverbound
  opcode: 97
  fname: CWvsContext::CheckTemporaryStatDuration
  provenance: ida-discovered
  ida:
    address: 9874760
  note: 'Empty body. v79 IDB left the function unnamed (sub_96AD48); renamed during the task-190 pass. Client never assigns m_tLastStatResetRequest here. Corrects a prior n-a assertion (task-190).'
```

Do **not** create `docs/packets/registry/gms_v92.yaml`. Registry files are matrix-column artifacts (a registry *and* an IDA export *and* an audit dir); v92 has a seed template but is not a column, and a lone half-populated file would read as one to both a human and `matrix --check`. Its `0x6E` is recorded in `template_gms_92_1.json`, `investigation.md` §8.2, and `backfill.md`.

- [ ] **Step 2: Verify the fname-doc gate**

```bash
cd tools/packet-audit && go run . fname-doc --check
```

(If the invocation differs, discover it with `go run . --help` from `tools/packet-audit`.) Expected: exit 0. `CWvsContext::CheckTemporaryStatDuration` already appears in the v83/v84/v87/v95/jms registries, so the fname is known.

- [ ] **Step 3: Promote the nine matrix cells**

Each cell goes through the standard single-cell procedure — `/verify-packet` driving `docs/packets/audits/VERIFYING_A_PACKET.md` and the `packet-verifier` agent, one run per packet × version. Pin the subagents to Sonnet (per project convention for review/verify workflows). The nine cells:

`gms_v48`, `gms_v61`, `gms_v72`, `gms_v79` (n-a → verified — these are the corrections), and `gms_v83`, `gms_v84`, `gms_v87`, `gms_v95`, `jms_v185` (incomplete → verified).

`gms_v92` gets the round-trip test from Task 7 but **no matrix marker** — it is not a column.

Each is the cheapest possible cell: the evidence is "COutPacket(opcode) then SendPacket, zero encode calls between them," and the fixture asserts a zero-length body. The `packet-audit:verify` markers are already in `cancel_debuff_test.go` from Task 7.

- [ ] **Step 4: Regenerate the matrix**

```bash
cd tools/packet-audit && go run . matrix
cd tools/packet-audit && go run . matrix --check
```

Expected: `--check` exits 0. Note the matrix `toolSha` reads git HEAD — regenerate **after** any rebase onto main, not before, or the recorded sha is stale.

- [ ] **Step 5: Confirm the four n-a corrections actually landed**

```bash
grep -A25 '"op": "CANCEL_DEBUFF"' docs/packets/audits/status.json | head -40
```

Expected: `gms_v48`/`v61`/`v72`/`v79` now read `"state": "verified"` with their real opcodes (78/91/98/97), not `"n-a"` with `-1`.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/registry/ docs/packets/audits/ libs/atlas-packet/character/serverbound/
git commit -m "docs(packets): CANCEL_DEBUFF registry entries + nine verified cells; corrects four false n-a (task-190 FR-2.3)"
```

---

### Task 14: Live-tenant backfill, rollout, and the `0x6C` follow-up

**Files:**
- Create: `docs/tasks/task-190-disease-duration-cancel-debuff/backfill.md`

**Interfaces:**
- Consumes: the ten template entries from Task 12.
- Produces: an operator-runnable backfill procedure; nothing consumed by later tasks.

- [ ] **Step 1: Read the precedent**

```bash
cat docs/tasks/task-153-corsair-battleship/backfill.md
```

Model the new doc on its structure. Seed templates apply at **tenant creation only** — an already-provisioned tenant never sees the new handler otherwise, and atlas-channel does not hot-reload socket configuration.

- [ ] **Step 2: Write the backfill document**

Create `docs/tasks/task-190-disease-duration-cancel-debuff/backfill.md`:

```markdown
# task-190 — live-tenant backfill

Routing `CancelDebuffHandle` in the seed templates does not reach any tenant
that already exists. Seed templates apply at tenant creation only, and
atlas-channel does not hot-reload socket configuration — the pods must be
restarted after the PATCH.

Until a tenant is backfilled it degrades to today's behaviour: the opcode has
no handler, `libs/atlas-socket/server.go` logs the unhandled op at Info, and
nothing errors (NFR-5).

## Per-tenant procedure

For each live tenant:

1. `GET /tenants/{tenantId}/configurations/socket` — capture the current
   document.
2. Insert this entry into the `handlers` array at its **sorted `opCode`
   position** (strictly ascending — see `docs/packets/TEMPLATE_CONVENTIONS.md`),
   using the opcode for that tenant's version:

   | Tenant version | opCode |
   |---|---|
   | GMS 48.1 | `0x4E` |
   | GMS 61.1 | `0x5B` |
   | GMS 72.1 | `0x62` |
   | GMS 79.1 | `0x61` |
   | GMS 83.1 | `0x63` |
   | GMS 84.1 | `0x63` |
   | GMS 87.1 | `0x66` |
   | GMS 92.1 | `0x6E` |
   | GMS 95.1 | `0x6F` |
   | JMS 185.1 | `0x5E` |

   ```json
   {
     "opCode": "<from the table>",
     "validator": "LoggedInValidator",
     "handler": "CancelDebuffHandle",
     "services": ["channel"]
   }
   ```

   The `validator` is not optional — a handler entry without one is silently
   dropped, which is indistinguishable from the packet never arriving.

3. `PATCH` the document back.
4. `GET` again and confirm the entry is present at the expected position —
   atlas-configurations does not echo the document on PATCH.
5. Restart the tenant's atlas-channel pods.

## Verification

With a client of that tenant connected, take a mob debuff and watch the channel
logs:

- Before: `Read a unhandled message with op 0x63` (or the tenant's opcode),
  repeating at frame rate.
- After: at most one Debug `Throttled CANCEL_DEBUFF` line per second per
  character, and zero unhandled-op lines for that opcode.

## Also required after this deploy: re-ingest Skill.wz

The duration fix in `atlas-data mobskill/reader.go` does NOT retroactively
correct already-ingested rows — WZ data is ingested, not parsed per request.
See the Rollout section of `design.md` §10; the ordering is:

1. Deploy the code (atlas-data, atlas-monsters, atlas-maps, atlas-channel,
   atlas-buffs, atlas-configurations, atlas-ui).
2. Re-ingest `Skill.wz` per tenant. `document.Storage.Add` writes through with
   an `OnConflict` clause, so re-ingest is an upsert, not a duplicate error.
3. **Roll atlas-data.** `Storage.ByIdProvider` serves from the per-pod in-memory
   `document.Registry` first, so replicas that did not perform the ingest keep
   stale mob-skill rows until restarted. Skipping this is the most likely way
   for the fix to appear not to work.
4. Verify the DATA, not the deploy: `GET /data/mob-skills/126` must show
   `duration: 15000` for level 2. atlas-monsters holds no cache
   (`monster/mobskill/processor.go` fetches per use), so it picks up the new
   value immediately.
5. Backfill the socket configs per the procedure above.
```

- [ ] **Step 3: File the `0x6C` follow-up task**

`USER_CALC_DAMAGE_STAT_SET_REQUEST` is the tail of the handshake this task implements — `CWvsContext::OnTemporaryStatReset` ends with `if (IsCalcDamageStat(mask)) { COutPacket(0x6C); SendPacket(...); }`. Implementing FR-2 makes it fire **more** often. It stays out of scope on evidence (one-shot per reset, not a loop, so it cannot wedge a client; the cost is a possibly-stale damage-range display), but it must not be silently dropped.

```bash
tools/task-numbers.sh next
```

Record the reserved number and file the follow-up. Include in its description: opcode known for only three versions so far (v48 `0x56`/86, v61 `0x63`/99, v83 `0x6C`/108); the remaining seven need the same per-version IDB pass; and note the v61 `0x63` collision with CANCEL_DEBUFF's meaning at v83/v84. Add a line to `docs/TODO.md` if that is the repo's tracking convention — find it first with Glob rather than assuming the path.

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-190-disease-duration-cancel-debuff/backfill.md docs/TODO.md
git commit -m "docs(task-190): live-tenant socket-config backfill + rollout ordering (FR-2.8)"
```

---

### Task 15: Full verification sweep and code review

**Files:** none created; this task is the gate before a PR.

**Interfaces:**
- Consumes: everything.
- Produces: `docs/tasks/task-190-disease-duration-cancel-debuff/audit.md` (written by the reviewer agents).

- [ ] **Step 1: Test, vet, and build every changed Go module**

```bash
for m in atlas-data/atlas.com/data atlas-monsters/atlas.com/monsters atlas-maps/atlas.com/maps atlas-channel/atlas.com/channel atlas-buffs/atlas.com/buffs; do
  echo "===== $m"
  ( cd "services/$m" && go build ./... && go vet ./... && go test -race ./... ) || echo "FAIL $m"
done
( cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./... ) || echo "FAIL libs/atlas-packet"
( cd tools/buffdurationguard && GOWORK=off go vet ./... && GOWORK=off go test ./... ) || echo "FAIL buffdurationguard"
```

Expected: no `FAIL` lines. Quote the actual output when reporting — do not paraphrase.

- [ ] **Step 2: Check whether any `go.mod` changed**

```bash
git diff --name-only main...HEAD | grep 'go\.\(mod\|sum\)$'
```

If any **service** `go.mod` appears, `docker buildx bake` is mandatory for that service (`go build` against `go.work` will not catch a missing `COPY libs/...` line in the shared Dockerfile):

```bash
docker buildx bake atlas-<svc>
```

`tools/buffdurationguard/go.mod` is a new module but is **not** a service and is absent from `go.work` and `docker-bake.hcl` — it needs no bake and no `COPY` line. If only that one appears, state that explicitly rather than silently skipping the step.

- [ ] **Step 3: Run every guard**

```bash
./tools/redis-key-guard.sh
./tools/goroutine-guard.sh
./tools/buff-duration-guard.sh
./tools/template-opcode-order-guard.sh
./tools/skill-job-id-guard.sh
./tools/lint.sh --check
```

Expected: every one exits 0. `tools/lint.sh` with no flags rewrites files in place — run the fix mode first if `--check` fails, then re-commit.

`tools/service-registration-guard.sh` is not required: no service was added and `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work` and `tools/db-bootstrap.sh` are all untouched. Confirm that with `git diff --name-only main...HEAD` before skipping it.

- [ ] **Step 4: Run the packet gates**

```bash
cd tools/packet-audit && go run . matrix --check && go run . fname-doc --check
```

Expected: both exit 0.

- [ ] **Step 5: Build the UI**

```bash
cd services/atlas-ui && source ~/.nvm/nvm.sh && nvm use 22 && npm run build && npx vitest run
```

Expected: build succeeds; vitest at its existing baseline.

- [ ] **Step 6: Re-run the FR-3.2 demonstration end to end**

Repeat Task 6 Step 10 on the final tree and confirm the transcripts in `producer-audit.md` still match what the guard prints. A demonstration recorded against an intermediate tree is not a demonstration of the shipped one.

- [ ] **Step 7: Confirm the untouchable sites are still untouched**

```bash
git diff main...HEAD -- services/atlas-monsters/atlas.com/monsters/monster/processor.go | grep -n "executeDebuff" 
git diff --name-only main...HEAD | grep template_gms_12_1
ls docs/packets/registry/gms_v92.yaml 2>&1
```

Expected: no `executeDebuff` hunk, no `template_gms_12_1` in the diff, and `No such file or directory` for the v92 registry.

- [ ] **Step 8: Run code review**

Invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer`, `backend-guidelines-reviewer` (Go changed), and `frontend-guidelines-reviewer` (atlas-ui TS changed) in parallel, writing to `docs/tasks/task-190-disease-duration-cancel-debuff/audit.md`.

Every reviewer subagent prompt must `cd` into this worktree and verify the branch before doing anything, and reviewers should be pinned to Sonnet. After the run, confirm nothing was written into the main checkout:

```bash
git -C ../.. status --short | head
git status --short
```

Expected: the main checkout is unchanged by the review; this worktree's only new file is `audit.md`.

- [ ] **Step 9: Address the findings, then commit**

Fix what the audit surfaces on this branch — do not defer it to a follow-up task and do not fork a new worktree (produce the clean PR branch by rebase at PR time if needed).

```bash
git add docs/tasks/task-190-disease-duration-cancel-debuff/audit.md
git commit -m "docs(task-190): code review audit"
```

- [ ] **Step 10: End-to-end acceptance on a live tenant (GMS 83.1)**

These are the criteria the whole task exists for. Run them after deploying per `backfill.md`:

- [ ] A mob Slow lasts its authored WZ duration on the client and ends on its own.
- [ ] Zero `Read a unhandled message with op 0x63` lines during a sustained debuff fight.
- [ ] A client that locally expires a stat the server still holds recovers without a relog.
- [ ] Skills remain usable and the mount buff can be cancelled throughout a debuff fight (the originally-reported wedge does not reproduce).
- [ ] A mob-cast mist created after the change lasts its authored duration and is **not** clamped to exactly 60s. Pick a mist skill whose authored `time ≤ 60` and name it in the result — the 60 000 ms cap is real and a longer-authored mist will still legitimately clamp.
- [ ] A monster self-buff / immunity lasts its authored duration, not ~1000× longer.

Record each result with the observed evidence (log lines, timings). An unrun check is reported as unrun, not as passing.

---

## Self-Review

**Spec coverage**

| Requirement | Task |
|---|---|
| FR-1.1 reader ×1000 | 1 |
| FR-1.2 three double-conversion sites | 2 (two), 3 (one) |
| FR-1.3 `executeDebuff` untouched | 2 Step 5, 15 Step 7 |
| FR-1.4 stale comment replaced, names `11e07dfa7` | 3 Step 3 |
| FR-1.5 seven-producer audit | 5 |
| FR-1.6 seconds-pinning tests updated not deleted | 2 Step 1, 3 Step 1 |
| FR-2.1/2.2 opcode-only codec | 7 |
| FR-2.3 ten opcodes; FR-2.3.2 no hard-coded opcode | 12, 13; 10 Step 7 |
| FR-2.3.1 unthrottled send rate assumed | 8 (`TestAllow_ThrottlesInsideWindow`) |
| FR-2.4.1 clientbound mask unchanged | 7 (documented; no clientbound edit anywhere) |
| FR-2.5 handler | 10 |
| FR-2.6 reconcile via Kafka, not cancel-by-name | 9, 11 |
| FR-2.6.1 per-character sweep | 11 |
| FR-2.7 all ten templates with a validator | 12 |
| FR-2.8 backfill documented | 14 |
| FR-2.9 / NFR-2.1 nothing expired ⇒ no packet | 11 (`_NothingExpired`) |
| FR-2.10 / NFR-2 rate limit | 8, 10 |
| FR-3.1 contract stated once | 5 |
| FR-3.2 mechanical guard, demonstrated | 6 (Steps 9, 10) |
| FR-3.3 guard covers mist and direct-disease | 6 (BD-1 + BD-2 fixtures) |
| NFR-1 no sync REST on the hot path | 9 (Kafka emit only) |
| NFR-3 tenant from context, no wire literals | 10 (`tenant.MustFromContext`), 10 Step 7 |
| NFR-4 dropped nudge logs at Debug | 10 |
| NFR-5 un-backfilled tenant degrades, does not error | 14 (documented; no code path added) |
| §6 range check | 1 Step 6 → 5 Step 3 |
| §9.3 `0x6C` follow-up filed | 14 Step 3 |
| design §8.1 atlas-ui consumer | 4 |
| design §8.2 mist cap changed meaning | 2 (`_UnderCapIsNotClamped`), 15 Step 10 |
| design D7 no v92 registry | 13 Step 1, 15 Step 7 |
| design D8 no gms_12 routing | 12 Step 2, 15 Step 7 |
| design D9 four false n-a corrected | 13 Step 5 |
| Build & verification checklist | 15 |

No gaps.

**Placeholder scan:** the only intentional `<…>` placeholders are in the `producer-audit.md` template (Task 5 Step 3), where the step explicitly requires filling each with a read value or the literal string `NOT VERIFIED — <what blocked it>`. Every code step carries real code.

**Type consistency:** `CommandTypeExpire = "EXPIRE"` and `ExpireCommandBody struct{}` are defined identically in Task 9 (atlas-channel) and Task 11 (atlas-buffs), and Task 11 Step 7 greps to prove they match. `Processor.Expire(f field.Model, characterId uint32) error` (Task 9) is the exact call made in Task 10. `ExpireForCharacter(worldId world.Id, characterId uint32) error` (Task 11) matches `handleExpire`'s call. `statreset.GetRegistry().Allow(t, characterId, now)` / `.ClearCharacter(t, characterId)` (Task 8) match both call sites in Task 10. `CancelDebuffHandle` is the same string in Task 7's codec, Task 10's handler-map key, and every template entry in Task 12.
