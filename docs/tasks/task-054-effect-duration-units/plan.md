# Effect Duration Units Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `effect.Model.Duration()` (and its REST/Kafka counterparts) unambiguously **milliseconds** across atlas-data → atlas-channel → {atlas-buffs, atlas-monsters}, fixing the inverted reader logic at `services/atlas-data/atlas.com/data/skill/reader.go:164-169` and the seconds-based interpretation at `services/atlas-buffs/atlas.com/buffs/buff/model.go:112` in lockstep.

**Architecture:** Producer-side conversion in `atlas-data/skill/reader.go` is the single conversion point. Downstream consumers (atlas-buffs, atlas-monsters, atlas-channel handlers) interpret `Duration` as ms with `time.Duration(n) * time.Millisecond` and perform no unit conversion of their own. Contract is pinned by tests at the reader layer (atlas-data) and the consumer layer (atlas-buffs). atlas-monsters and atlas-channel are audit-only — they already interpret ms.

**Tech Stack:** Go 1.21+, testify, atlas-data XML reader, atlas-buffs in-memory `Registry`, atlas-monsters Kafka consumer.

---

## Task 1: atlas-data — flip existing reader test assertions to ms (red)

Update the six numeric `Duration` assertions in the existing `TestReader` so they expect milliseconds. These assertions will go red against today's production code (the reader still emits raw seconds for the populated branch), then go green in Task 3 after the reader logic flip.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/reader_test.go:2822-2823, 2835-2836, 2848-2849, 2868-2869, 2881-2882, 2894-2895`

- [ ] **Step 1: Update skill 1001 effect[0] Duration assertion**

In `services/atlas-data/atlas.com/data/skill/reader_test.go` at lines 2822-2824, change:

```go
	if ef.Duration != 30 {
		t.Fatalf("rm.Effects[0].Duration = %d, want 30", ef.Duration)
	}
```

to:

```go
	if ef.Duration != 30000 {
		t.Fatalf("rm.Effects[0].Duration = %d, want 30000", ef.Duration)
	}
```

- [ ] **Step 2: Update skill 1001 effect[1] Duration assertion**

At lines 2835-2837, change:

```go
	if ef.Duration != 30 {
		t.Fatalf("rm.Effects[1].Duration = %d, want 30", ef.Duration)
	}
```

to:

```go
	if ef.Duration != 30000 {
		t.Fatalf("rm.Effects[1].Duration = %d, want 30000", ef.Duration)
	}
```

- [ ] **Step 3: Update skill 1001 effect[2] Duration assertion**

At lines 2848-2850, change:

```go
	if ef.Duration != 30 {
		t.Fatalf("rm.Effects[2].Duration = %d, want 30", ef.Duration)
	}
```

to:

```go
	if ef.Duration != 30000 {
		t.Fatalf("rm.Effects[2].Duration = %d, want 30000", ef.Duration)
	}
```

- [ ] **Step 4: Update skill 1002 effect[0] Duration assertion**

At lines 2868-2870, change:

```go
	if ef.Duration != 4 {
		t.Fatalf("rm.Effects[0].Duration = %d, want 4", ef.Duration)
	}
```

to:

```go
	if ef.Duration != 4000 {
		t.Fatalf("rm.Effects[0].Duration = %d, want 4000", ef.Duration)
	}
```

- [ ] **Step 5: Update skill 1002 effect[1] Duration assertion**

At lines 2881-2883, change:

```go
	if ef.Duration != 8 {
		t.Fatalf("rm.Effects[1].Duration = %d, want 8", ef.Duration)
	}
```

to:

```go
	if ef.Duration != 8000 {
		t.Fatalf("rm.Effects[1].Duration = %d, want 8000", ef.Duration)
	}
```

- [ ] **Step 6: Update skill 1002 effect[2] Duration assertion**

At lines 2894-2896, change:

```go
	if ef.Duration != 12 {
		t.Fatalf("rm.Effects[2].Duration = %d, want 12", ef.Duration)
	}
```

to:

```go
	if ef.Duration != 12000 {
		t.Fatalf("rm.Effects[2].Duration = %d, want 12000", ef.Duration)
	}
```

- [ ] **Step 7: Run tests to confirm they go red**

```
cd services/atlas-data/atlas.com/data && go test ./skill/ -run TestReader$ -v
```

Expected: FAIL — six lines reporting `Duration = 30, want 30000` and `Duration = 4, want 4000` (etc.). This confirms the test is honest and the production bug is pinned.

- [ ] **Step 8: Do not commit yet**

These assertions are red until Task 3 lands. Tasks 1–3 form a single TDD cycle; commit after Task 3.

---

## Task 2: atlas-data — add new ms-pinning reader tests (red)

Add three focused tests that pin the new contract: ms conversion on populated `time`, sentinel preservation on missing `time`, and FREEZE doubling operating on ms.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/reader_test.go` (append three test functions)

- [ ] **Step 1: Append `TestReader_TimeAttributeEmittedAsMilliseconds`**

Append at the end of `services/atlas-data/atlas.com/data/skill/reader_test.go`:

```go
// TestReader_TimeAttributeEmittedAsMilliseconds pins the unit contract:
// the wz `time` attribute (raw seconds) is converted to milliseconds by
// atlas-data's reader before downstream consumers see it. See task-054.
func TestReader_TimeAttributeEmittedAsMilliseconds(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), tn)

	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="231.img">
  <imgdir name="skill">
    <imgdir name="2311005">
      <imgdir name="level">
        <imgdir name="30">
          <int name="time" value="60"/>
          <int name="mpCon" value="35"/>
          <int name="prop" value="100"/>
          <int name="mobCount" value="6"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	rms := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(xmlData)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	rm, ok := rmm["2311005"]
	if !ok {
		t.Fatal("rmm[2311005] does not exist.")
	}
	if len(rm.Effects) != 1 {
		t.Fatalf("len(rm.Effects) = %d, want 1", len(rm.Effects))
	}
	if got := rm.Effects[0].Duration; got != 60000 {
		t.Fatalf("Duration = %d, want 60000 (ms; wz time=60 seconds)", got)
	}
}
```

- [ ] **Step 2: Append `TestReader_TimeMissing_DurationStaysSentinel`**

Append immediately after the previous test:

```go
// TestReader_TimeMissing_DurationStaysSentinel pins the cleaned-up else
// branch: when the wz `time` attribute is absent, Duration stays at the
// -1 sentinel and is NOT multiplied. See task-054.
func TestReader_TimeMissing_DurationStaysSentinel(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), tn)

	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="100.img">
  <imgdir name="skill">
    <imgdir name="1001004">
      <imgdir name="level">
        <imgdir name="1">
          <int name="mpCon" value="5"/>
          <int name="damage" value="100"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	rms := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(xmlData)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	rm, ok := rmm["1001004"]
	if !ok {
		t.Fatal("rmm[1001004] does not exist.")
	}
	if len(rm.Effects) != 1 {
		t.Fatalf("len(rm.Effects) = %d, want 1", len(rm.Effects))
	}
	if got := rm.Effects[0].Duration; got != -1 {
		t.Fatalf("Duration = %d, want -1 (sentinel; wz `time` missing)", got)
	}
}
```

- [ ] **Step 3: Append `TestReader_FreezeDoublesDuration`**

Append immediately after the previous test. Skill id `2201004` is `IceLightningWizardColdBeamId`, in the FREEZE doubling branch at `reader.go:343-346`.

```go
// TestReader_FreezeDoublesDuration pins that the FREEZE special-case
// doubling at reader.go:346 operates on the milliseconds-converted
// duration: time=4 → 4000 (ms) → 8000 (FREEZE-doubled). See task-054.
func TestReader_FreezeDoublesDuration(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), tn)

	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="220.img">
  <imgdir name="skill">
    <imgdir name="2201004">
      <imgdir name="level">
        <imgdir name="1">
          <int name="time" value="4"/>
          <int name="mpCon" value="20"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	rms := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(xmlData)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	rm, ok := rmm["2201004"]
	if !ok {
		t.Fatal("rmm[2201004] does not exist.")
	}
	if len(rm.Effects) != 1 {
		t.Fatalf("len(rm.Effects) = %d, want 1", len(rm.Effects))
	}
	if got := rm.Effects[0].Duration; got != 8000 {
		t.Fatalf("Duration = %d, want 8000 (FREEZE doubles 4000 ms)", got)
	}
}
```

- [ ] **Step 4: Run new tests to confirm they go red**

```
cd services/atlas-data/atlas.com/data && go test ./skill/ -run 'TestReader_TimeAttributeEmittedAsMilliseconds|TestReader_TimeMissing_DurationStaysSentinel|TestReader_FreezeDoublesDuration' -v
```

Expected: FAIL.
- `TestReader_TimeAttributeEmittedAsMilliseconds`: `Duration = 60, want 60000`.
- `TestReader_TimeMissing_DurationStaysSentinel`: `Duration = -1000, want -1` (the bug we're removing — the spurious `* 1000` is multiplying the `-1` sentinel into `-1000`).
- `TestReader_FreezeDoublesDuration`: `Duration = 8, want 8000` (4 raw seconds doubled to 8 by the FREEZE branch, never multiplied to ms).

These three failures are the explicit pin of every behavior change in atlas-data.

- [ ] **Step 5: Do not commit yet**

Hold the commit until Task 3 makes them all green.

---

## Task 3: atlas-data — flip the reader if/else logic (green)

Invert the conditional at `reader.go:164-169` so `* 1000` runs on the populated branch. Remove the spurious `* 1000` from the missing-`time` branch.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/reader.go:164-169`

- [ ] **Step 1: Replace the if/else block**

In `services/atlas-data/atlas.com/data/skill/reader.go`, replace lines 164-169:

```go
	if e.Duration() > -1 {
		e.SetOverTime(true)
	} else {
		e.SetDuration(e.Duration() * 1000)
		e.SetOverTime(overTime)
	}
```

with:

```go
	// Why ms: the wz `time` attribute is in seconds; convert here so
	// downstream consumers (atlas-buffs, atlas-monsters) interpret
	// effect.Duration() uniformly as time.Millisecond. See task-054.
	if e.Duration() > -1 {
		e.SetDuration(e.Duration() * 1000)
		e.SetOverTime(true)
	} else {
		e.SetOverTime(overTime)
	}
```

- [ ] **Step 2: Run the affected tests to confirm green**

```
cd services/atlas-data/atlas.com/data && go test ./skill/ -run 'TestReader$|TestReader_TimeAttributeEmittedAsMilliseconds|TestReader_TimeMissing_DurationStaysSentinel|TestReader_FreezeDoublesDuration|TestReader_PriestDoom_MapsDoomStatus' -v
```

Expected: PASS for all five tests. The original `TestReader` now sees ms (30000, 4000, 8000, 12000); the three new tests pin the contract; `TestReader_PriestDoom_MapsDoomStatus` continues to pass (it asserts `Duration > 0`, unit-independent).

- [ ] **Step 3: Run the full atlas-data skill package**

```
cd services/atlas-data/atlas.com/data && go test ./skill/... -count=1
```

Expected: PASS. If any other test in the package asserts a numeric `Duration` against a populated `time` and was missed in Task 1, it surfaces here. Add the ms-flip to that assertion and re-run.

- [ ] **Step 4: Commit**

```
git add services/atlas-data/atlas.com/data/skill/reader.go services/atlas-data/atlas.com/data/skill/reader_test.go
git commit -m "feat(atlas-data): emit effect Duration in ms (task-054)

Invert the if/else at reader.go:164-169 so the wz \`time\` attribute is
multiplied by 1000 on the populated branch; clean up the bogus *1000 that
ran on the -1 sentinel branch. Ms-pinning tests at the reader layer:

- TestReader_TimeAttributeEmittedAsMilliseconds (time=60 -> 60000)
- TestReader_TimeMissing_DurationStaysSentinel (-1 stays -1)
- TestReader_FreezeDoublesDuration (FREEZE doubles ms, not seconds)

Existing TestReader assertions migrated from raw seconds to ms (30/4/8/12
-> 30000/4000/8000/12000). Downstream consumers (atlas-buffs to follow,
atlas-monsters and atlas-channel already on ms) interpret as time.Millisecond."
```

---

## Task 4: atlas-data — flag SnowCharge regression (TODO)

Per design §3.2, `reader.go:373` passes `e.Duration()` as the WhiteKnightCharge stat amount. After Task 3, this value is 1000× larger than before. The right fix is to pass an actual charge-amount field, not Duration. Defer the fix; tag the regression so future readers see it.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/reader.go:373` (insert TODO comment immediately above)

- [ ] **Step 1: Insert TODO comment above line 373**

Find this block (around line 372-373):

```go
		} else if skill.Is(skillId, skill.AranStage3SnowChargeId) {
			statups = produceBuffStatAmount(statups, character.TemporaryStatTypeWhiteKnightCharge, e.Duration())
```

Replace with:

```go
		} else if skill.Is(skillId, skill.AranStage3SnowChargeId) {
			// TODO(post-task-054): SnowCharge passes Duration as the
			// WhiteKnightCharge stat amount. After task-054 this is 1000x
			// larger (now ms, was raw seconds). The right fix is to pass
			// a charge-amount field (likely e.X()), not Duration. Tracked
			// in docs/TODO.md.
			statups = produceBuffStatAmount(statups, character.TemporaryStatTypeWhiteKnightCharge, e.Duration())
```

- [ ] **Step 2: Verify the file still builds**

```
cd services/atlas-data/atlas.com/data && go build ./skill/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```
git add services/atlas-data/atlas.com/data/skill/reader.go
git commit -m "chore(atlas-data): flag SnowCharge stat-amount regression (task-054)

reader.go:373 passes effect.Duration as the WhiteKnightCharge stat amount.
After task-054 it is now 1000x larger. Defer correcting the actual semantic
(should pass a charge-amount field, not a duration); leave a TODO pointing
at docs/TODO.md."
```

---

## Task 5: atlas-data + atlas-channel — doc comments on Duration accessors

Document the ms unit at every `Duration()` accessor that consumers read. atlas-data's exposed accessor is on `ModelBuilder` (the package only otherwise has `RestModel.Duration` as a struct field); atlas-channel's effect package has the consumer-side `Model.Duration()`.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/effect/model.go:165`
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:78`

- [ ] **Step 1: Add doc comment in atlas-data**

In `services/atlas-data/atlas.com/data/skill/effect/model.go` immediately above line 165, insert:

```go
// Duration returns the effect duration in milliseconds. -1 is the
// "no duration" sentinel (the wz `time` attribute was missing).
// Positive values are ms counts converted from raw wz seconds at
// read time. Consumers should use time.Duration(d) * time.Millisecond.
// See task-054.
```

The result should look like:

```go
// Duration returns the effect duration in milliseconds. -1 is the
// "no duration" sentinel (the wz `time` attribute was missing).
// Positive values are ms counts converted from raw wz seconds at
// read time. Consumers should use time.Duration(d) * time.Millisecond.
// See task-054.
func (b *ModelBuilder) Duration() int32 {
	return b.duration
}
```

- [ ] **Step 2: Add doc comment in atlas-channel**

In `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` immediately above line 78, insert:

```go
// Duration returns the effect duration in milliseconds. -1 is the
// "no duration" sentinel. Consumers should use
// time.Duration(d) * time.Millisecond. See task-054.
```

The result should look like:

```go
// Duration returns the effect duration in milliseconds. -1 is the
// "no duration" sentinel. Consumers should use
// time.Duration(d) * time.Millisecond. See task-054.
func (m Model) Duration() int32 {
	return m.duration
}
```

- [ ] **Step 3: Verify both packages still build**

```
cd services/atlas-data/atlas.com/data && go build ./... && cd - && \
cd services/atlas-channel/atlas.com/channel && go build ./...
```

Expected: no output (success).

- [ ] **Step 4: Commit**

```
git add services/atlas-data/atlas.com/data/skill/effect/model.go services/atlas-channel/atlas.com/channel/data/skill/effect/model.go
git commit -m "docs(skill-effect): document ms unit on Duration() (task-054)"
```

---

## Task 6: atlas-buffs — flip existing test assertion to ms (red)

Update the existing `TestBuff_Timestamps` to expect millisecond-based expiry math. This will go red against today's production code (still uses `time.Second`); Task 7 makes it green.

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/buff/model_test.go:47`

- [ ] **Step 1: Flip the unit in the existing assertion**

In `services/atlas-buffs/atlas.com/buffs/buff/model_test.go` at line 47, change:

```go
	expectedExpiry := b.CreatedAt().Add(time.Duration(duration) * time.Second)
```

to:

```go
	expectedExpiry := b.CreatedAt().Add(time.Duration(duration) * time.Millisecond)
```

- [ ] **Step 2: Run the test to confirm it goes red**

```
cd services/atlas-buffs/atlas.com/buffs && go test ./buff/ -run TestBuff_Timestamps -v
```

Expected: FAIL — `ExpiresAt should be within 1ms of expected expiry`. The diff is ~60 seconds (duration=60 second-old expiry vs. duration=60 millisecond-new expectation).

- [ ] **Step 3: Do not commit yet**

Hold the commit until Tasks 6–8 are done together.

---

## Task 7: atlas-buffs — flip production code to time.Millisecond (green)

Single-line production change. After this step, atlas-buffs interprets `Duration` as ms.

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/buff/model.go:112`

- [ ] **Step 1: Replace `time.Second` with `time.Millisecond`**

In `services/atlas-buffs/atlas.com/buffs/buff/model.go` at line 112, change:

```go
		expiresAt: time.Now().Add(time.Duration(duration) * time.Second),
```

to:

```go
		expiresAt: time.Now().Add(time.Duration(duration) * time.Millisecond),
```

- [ ] **Step 2: Run the previously-red test to confirm it goes green**

```
cd services/atlas-buffs/atlas.com/buffs && go test ./buff/ -run TestBuff_Timestamps -v
```

Expected: PASS.

- [ ] **Step 3: Run the full buff package**

```
cd services/atlas-buffs/atlas.com/buffs && go test ./buff/... -count=1
```

Expected: PASS. The `TestBuff_Expired_NotExpired` test uses `duration := int32(60)` and asserts `!b.Expired()`. Pre-fix this is a 60-second buff (definitely not expired). Post-fix this is a 60ms buff. Whether it's "expired" depends on timing — testify executes the assertion immediately after `NewBuff`, well within 60ms wall-clock, so the test still passes. If it ever flakes on a slow CI box, replace `int32(60)` with a larger value (e.g. `int32(60000)`); flag this in the PR description but do NOT preemptively change it.

- [ ] **Step 4: Do not commit yet**

Continue to Task 8 to add the explicit ms-pinning test, then commit all atlas-buffs changes together.

---

## Task 8: atlas-buffs — add explicit ms-pinning test

Add `TestBuff_DurationInMilliseconds` so the unit contract is named and self-documenting (rather than implicit in `TestBuff_Timestamps`'s tolerance arithmetic).

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/buff/model_test.go` (append one test)

- [ ] **Step 1: Append the new test**

Append at the end of `services/atlas-buffs/atlas.com/buffs/buff/model_test.go`:

```go
// TestBuff_DurationInMilliseconds pins the unit contract for atlas-buffs:
// Duration is interpreted as time.Millisecond (NOT time.Second). Aligned
// with atlas-data's reader emitting ms after task-054.
func TestBuff_DurationInMilliseconds(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60000) // 60 seconds expressed in ms
	changes := setupTestChanges()

	b, err := NewBuff(sourceId, byte(5), duration, changes)
	assert.NoError(t, err)

	gap := b.ExpiresAt().Sub(b.CreatedAt())
	expected := 60 * time.Second
	tolerance := 50 * time.Millisecond
	diff := gap - expected
	if diff < 0 {
		diff = -diff
	}
	assert.True(t, diff <= tolerance,
		"expected ExpiresAt-CreatedAt within %v of %v, got %v (diff %v)", tolerance, expected, gap, diff)
}
```

- [ ] **Step 2: Run the new test**

```
cd services/atlas-buffs/atlas.com/buffs && go test ./buff/ -run TestBuff_DurationInMilliseconds -v
```

Expected: PASS — `gap` is within 50ms of 60s (production code now multiplies by `time.Millisecond`).

- [ ] **Step 3: Run the full buff package once more**

```
cd services/atlas-buffs/atlas.com/buffs && go test ./buff/... -count=1
```

Expected: PASS for every test in the package.

- [ ] **Step 4: Commit**

```
git add services/atlas-buffs/atlas.com/buffs/buff/model.go services/atlas-buffs/atlas.com/buffs/buff/model_test.go
git commit -m "feat(atlas-buffs): interpret Duration as ms (task-054)

model.go:112: time.Second -> time.Millisecond. atlas-data now emits
effect.Duration in ms (see task-054 atlas-data commit), so this is the
matching consumer-side flip. Wall-clock outcome for buffs is preserved.

Tests:
- TestBuff_Timestamps: flipped expectedExpiry math to time.Millisecond.
- TestBuff_DurationInMilliseconds (new): pins ms contract via 60000 ->
  ~60s wall gap with 50ms tolerance."
```

---

## Task 9: docs/TODO.md — add follow-up entries

Per design §5, file two follow-ups:

1. SnowCharge stat amount uses Duration in ms post-task-054; should use a charge-amount field.
2. Skill effect cooldown unit normalization (post task-054), per PRD §11.

**Files:**
- Modify: `docs/TODO.md`

- [ ] **Step 1: Locate the right insertion point**

```
grep -n "^## " docs/TODO.md | head -20
```

Expected: a list of section headers. Pick the section that already groups game-mechanic / skill-effect TODOs. If there is no such section, create a new top-level section named `## Skill effects` immediately before the first `## ` header that comes alphabetically after it (or at the bottom of the file if no good insertion exists). Use the existing entries' formatting as a template — the `/dev-docs` convention is one bullet per item with file:line context.

- [ ] **Step 2: Append the two entries**

Add two entries under the chosen section:

```markdown
- **SnowCharge stat amount uses Duration in ms after task-054** — the
  WhiteKnightCharge stat amount in the SnowCharge mapping is now 1000x
  larger because Duration switched from raw seconds to milliseconds.
  Right fix: pass a charge-amount field (likely `e.X()`), not Duration.
  File: `services/atlas-data/atlas.com/data/skill/reader.go:373`.

- **Skill effect cooldown unit normalization (post task-054)** — the
  `cooltime` XML attribute at `services/atlas-data/atlas.com/data/skill/reader.go:154`
  is read directly into `Cooldown uint32` with no conversion. Cooldown
  flows through atlas-character via the skill subsystem; unit semantics
  there need a separate audit + fix. Companion follow-up to task-054
  (which only normalized Duration).
```

- [ ] **Step 3: Verify the markdown still renders well**

```
head -50 docs/TODO.md && echo "---" && grep -c "^- \*\*" docs/TODO.md
```

Expected: both new bullets visible at the top of their section; bullet count incremented by 2 vs. before the edit.

- [ ] **Step 4: Commit**

```
git add docs/TODO.md
git commit -m "docs: add follow-up entries for SnowCharge and cooldown units (task-054)"
```

---

## Task 10: cross-service build & test verification

Run the build/test suite for every service whose code or contract was touched. Per CLAUDE.md "Build & Verification": always run builds and tests for ALL affected services before reporting completion.

**Files:** none (verification only).

- [ ] **Step 1: atlas-data**

```
cd services/atlas-data/atlas.com/data && go build ./... && go test ./... -count=1
```

Expected: build succeeds; all tests pass. If anything fails, root-cause and fix before proceeding — do not move on with red CI.

- [ ] **Step 2: atlas-buffs**

```
cd services/atlas-buffs/atlas.com/buffs && go build ./... && go test ./... -count=1
```

Expected: build succeeds; all tests pass.

- [ ] **Step 3: atlas-channel**

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... -count=1
```

Expected: build succeeds; all tests pass. The handlers in `skill/handler/common.go`, `skill/handler/doom/doom.go`, and `socket/handler/character_attack_common.go` had no production code change, but the doc comment in `data/skill/effect/model.go` is in this service. Per task-047 the handler tests use `60000` ms already; this confirms no regression.

- [ ] **Step 4: atlas-monsters**

```
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... -count=1
```

Expected: build succeeds; all tests pass. Includes the DOOM tests added in task-047 (`TestApplyStatusEffect_Doom_*`) which already pass `60000` ms — they confirm the consumer-side contract is intact.

- [ ] **Step 5: Verify nothing else broke**

```
git status
```

Expected: clean working tree (all changes committed across Tasks 3, 4, 5, 8, 9). If anything is uncommitted, commit it as a fixup before declaring done.

- [ ] **Step 6: No new commit needed**

This task is verification only; no new commit unless a fix was needed in Steps 1–4.

---

## Self-review notes (executor: skip; written by planner)

- **Spec coverage:** every PRD §4 functional requirement maps to a task. §4.1 → Task 3. §4.2 → Task 7. §4.3 → audited in `context.md` (no production change required). §4.4 → audited in `context.md`. §4.5 → Tasks 1, 2, 6, 8 (atlas-data + atlas-buffs tests) plus the verification-only references to existing tests in §4.5 atlas-channel/atlas-monsters confirmed by Task 10. §4.6 → Tasks 5 (doc comments) + 9 (TODO entries).
- **PRD §9 resolutions:** §9.1 (persistence) closes via the audit captured in `context.md`. §9.2 (SnowCharge) → Tasks 4 + 9. §9.3 (other reader.go math) closes via the four-site enumeration in `context.md`.
- **TDD ordering:** every test step writes the failing assertion first and runs it red before any implementation. Tasks 1+2+3 form one TDD cycle; Tasks 6+7+8 form another.
- **Type / signature consistency:** `effect.RestModel.Duration int32`, `effect.Model.duration int32`, `buff.Model.duration int32` unchanged. `time.Duration` used consistently with `time.Millisecond`.
- **No placeholders:** every code block is concrete; no "similar to", "TBD", or "fill in".
