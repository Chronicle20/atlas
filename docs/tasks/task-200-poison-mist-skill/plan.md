# Poison Mist (2111003) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Casting Poison Mist (`2111003`) spawns a server-side mist visible to every session in the field that periodically poisons monsters standing inside it, on every provisioned client version.

**Architecture:** `atlas-channel` gains a per-skill handler that emits a `CREATE` command on `COMMAND_TOPIC_MIST`. The mist command/event contract is generalized with a `targetKind` (`CHARACTER`/`MONSTER`) and `effectKind` (`DISEASE`/`DAMAGE_OVER_TIME`) descriptor pair, defaulting to the pre-existing character-disease behavior. `atlas-maps`' existing mist tick gains a `MONSTER` branch that resolves monsters via the atlas-monsters `in-rect` endpoint and emits `APPLY_STATUS` per monster. No new service, topic, or REST endpoint.

**Tech Stack:** Go 1.x, Kafka (segmentio/kafka-go via `libs/atlas-kafka`), JSON:API REST via `libs/atlas-rest`, `libs/atlas-constants`, testify + `logrus/hooks/test`.

## Global Constraints

- **Design authority.** [`design.md`](design.md) is the spec. Where it deviates from [`prd.md`](prd.md), the design wins (design §8 tabulates the deviations).
- **Units.** `dotInterval` and `dotTime` are WZ **seconds**, converted to **milliseconds at the atlas-data reader** — the one conversion point. `dot` is forwarded raw. Every downstream ms value stays ms; no service re-scales.
- **No server-side lifetime clamp on player-cast mists.** `atlas-monsters`' `MistDurationCapMs` (60 s) must NOT be applied — the client derives its own `tEnd` from its own WZ (design §3.2). `MaxPlayerMistDurationMs = 300_000` **rejects**, it does not truncate.
- **Wire values are fixed and justified by IDB reading (design §3):** `nType = 0`, `dwOwnerId` = casting character id, `skillDelay = 0`, `nElemAttr = 0`, `nPhase = 0`. **No mist's wire bytes change in this task.** Existing `SPAWN_MIST` / `REMOVE_MIST` fixtures must pass unmodified.
- **Poison magnitude is `0` and that is correct.** `atlas-monsters` computes poison damage as `maxHP / (70 - sourceSkillLevel)` (`monster/status_task.go:106-113`); it never reads the `POISON` magnitude. Do not invent a magnitude.
- **Identity dispatch only.** Register against `skill2.FirePoisonMagicianPoisonMist`. Never compare a raw `2111003`. `tools/skill-job-id-guard.sh` enforces this.
- **`COMMAND_TOPIC_MONSTER` is a shared topic.** Every registered handler unmarshals every message. The new `APPLY_STATUS` body must use the **exact** existing key set of `ApplyStatusCommandBody` — no added, renamed, or retyped keys ([[bug_monster_command_topic_shared_handler_unmarshal_collision]]).
- **Buff duration guard.** `tools/buff-duration-guard.sh` fails CI on a seconds-valued `duration` in a `COMMAND_TOPIC_CHARACTER_BUFF` body. The character branch's `duration` stays milliseconds and is untouched.
- **No `// TODO`, stubs, or 501s in landed commits** (CLAUDE.md).
- **Committed docs use repo-relative paths only** — never a literal `/home/<user>/...`.

## Note on one design detail

design §4.1 lists `NType` as a new `CreatedBody` field. `CreatedBody` **already** carries the nType value as `Type int32 \`json:"type"\`` (`atlas-maps/kafka/message/mist/kafka.go:78`, consumed at `atlas-channel/kafka/consumer/mist/consumer.go:120`). Reuse that field; do **not** add a second `nType` key. Only `ElemAttr` and `SkillDelay` are genuinely new. `nPhase` stays a channel-side constant (design §3.4 fixes it at `0` for every version and it is not modelled by atlas-maps).

## File Structure

**atlas-data** (`services/atlas-data/atlas.com/data/`)
- Modify `skill/effect/model.go` — `ModelBuilder` gains `dot`/`dotInterval`/`dotTime` + setters/getters; `Build()` populates the RestModel.
- Modify `skill/effect/rest.go` — `RestModel` gains the three JSON fields.
- Modify `skill/reader.go` — parse the three WZ nodes; seconds→ms for interval and time.
- Modify `skill/reader_test.go` — present / absent / zero coverage.

**atlas-channel** (`services/atlas-channel/atlas.com/channel/`)
- Modify `data/skill/effect/model.go`, `data/skill/effect/rest.go` — mirror the three fields + getters.
- Modify `kafka/message/mist/kafka.go` — add the command side (topic, `CREATE`, `Command`, `CreateCommandBody`, kind constants) and the two new `CreatedBody` fields.
- Create `mist/processor.go`, `mist/producer.go` — the `COMMAND_TOPIC_MIST` producer seam.
- Modify `kafka/consumer/mist/consumer.go` — read `ElemAttr`/`SkillDelay` off the event instead of the local constants.
- Create `skill/handler/poisonmist/poisonmist.go` + `poisonmist_test.go` — the handler.
- Modify `skill/handler/registrations/registrations.go` — blank import.

**atlas-maps** (`services/atlas-maps/atlas.com/maps/`)
- Modify `kafka/message/mist/kafka.go` — kind constants, `CreateCommandBody` fields, `CreatedBody` fields.
- Modify `mist/model.go` — `targetKind`/`effectKind`/`elemAttr`/`skillDelay` + `SetKinds`/`SetRender` + `Rect()`.
- Modify `mist/processor.go` — normalize empty kinds in `Create`.
- Modify `mist/producer.go` — emit the two new `CreatedBody` fields.
- Modify `monster/processor.go`, `monster/requests.go` — `GetInMapRect`.
- Modify `tasks/mist_tick.go` — extract `tickCharacters`, add `tickMonsters` + the `monstersInRect` seam.
- Modify/extend `mist/model_test.go`, `mist/processor_test.go`, `tasks/mist_tick_test.go`.

**atlas-monsters** (`services/atlas-monsters/atlas.com/monsters/`)
- Modify `monster/processor.go` — `buildMistCreateBody` sets both kinds explicitly.
- Modify `monster/processor_test.go` — assert the two fields.

**libs/atlas-packet + docs/packets** — v92 `SPAWN_MIST` / `REMOVE_MIST` fixtures, evidence records, regenerated matrix.

---

### Task 1: atlas-data — parse `dot` / `dotInterval` / `dotTime`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/effect/model.go`
- Modify: `services/atlas-data/atlas.com/data/skill/effect/rest.go`
- Modify: `services/atlas-data/atlas.com/data/skill/reader.go`
- Test: `services/atlas-data/atlas.com/data/skill/reader_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `effect.RestModel` JSON keys `dot`, `dotInterval`, `dotTime` (all `int32`); `effect.ModelBuilder` methods `SetDot(int32) *ModelBuilder`, `SetDotInterval(int32) *ModelBuilder`, `SetDotTime(int32) *ModelBuilder`, `Dot() int32`, `DotInterval() int32`, `DotTime() int32`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-data/atlas.com/data/skill/reader_test.go` (mirrors the existing `TestReader_LT_RB_Present` shape at line 2909):

```go
// TestReader_Dot_Present pins the DoT field reads and their unit contract:
// `dot` is a raw damage-per-tick integer; `dotInterval` and `dotTime` are WZ
// SECONDS converted to MILLISECONDS at the reader, matching the `time`
// treatment (task-054). These nodes do not exist in any provisioned WZ corpus
// (design §2.1) -- the parse is additive and forward-compatible.
func TestReader_Dot_Present(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), tn)

	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="200.img">
  <imgdir name="skill">
    <imgdir name="2111003">
      <imgdir name="level">
        <imgdir name="1">
          <int name="mpCon" value="21"/>
          <int name="dot" value="105"/>
          <int name="dotInterval" value="1"/>
          <int name="dotTime" value="4"/>
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
	rm, ok := rmm["2111003"]
	if !ok {
		t.Fatal("rmm[2111003] does not exist.")
	}
	ef := rm.Effects[0]
	if ef.Dot != 105 {
		t.Fatalf("ef.Dot = %d, want 105 (raw, unscaled)", ef.Dot)
	}
	if ef.DotInterval != 1000 {
		t.Fatalf("ef.DotInterval = %d, want 1000 (1s -> ms)", ef.DotInterval)
	}
	if ef.DotTime != 4000 {
		t.Fatalf("ef.DotTime = %d, want 4000 (4s -> ms)", ef.DotTime)
	}
}

// TestReader_Dot_Absent asserts absent nodes default to 0 and do not change
// the serialized shape for skills that carry no DoT (FR-1.3).
func TestReader_Dot_Absent(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), tn)

	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="200.img">
  <imgdir name="skill">
    <imgdir name="2111003">
      <imgdir name="level">
        <imgdir name="1">
          <int name="mpCon" value="21"/>
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
	ef := rmm["2111003"].Effects[0]
	if ef.Dot != 0 || ef.DotInterval != 0 || ef.DotTime != 0 {
		t.Fatalf("ef dot fields = (%d,%d,%d), want (0,0,0)", ef.Dot, ef.DotInterval, ef.DotTime)
	}
}

// TestReader_Dot_ExplicitZero asserts an explicit zero-valued node stays zero
// rather than being scaled into a non-zero ms value.
func TestReader_Dot_ExplicitZero(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), tn)

	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="200.img">
  <imgdir name="skill">
    <imgdir name="2111003">
      <imgdir name="level">
        <imgdir name="1">
          <int name="dot" value="0"/>
          <int name="dotInterval" value="0"/>
          <int name="dotTime" value="0"/>
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
	ef := rmm["2111003"].Effects[0]
	if ef.Dot != 0 || ef.DotInterval != 0 || ef.DotTime != 0 {
		t.Fatalf("ef dot fields = (%d,%d,%d), want (0,0,0)", ef.Dot, ef.DotInterval, ef.DotTime)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-data/atlas.com/data && go test ./skill/ -run 'TestReader_Dot' -v`
Expected: FAIL — compile error, `ef.Dot undefined (type effect.RestModel has no field or method Dot)`.

- [ ] **Step 3: Add the three fields to the RestModel**

In `services/atlas-data/atlas.com/data/skill/effect/rest.go`, add to `RestModel` immediately after the `FixDamage int32 \`json:"fixDamage"\`` line:

```go
	// Dot is the raw per-tick damage-over-time magnitude (WZ `dot`).
	// Forwarded unscaled.
	Dot int32 `json:"dot"`
	// DotInterval is the DoT tick interval in MILLISECONDS. WZ stores
	// seconds; the reader converts (task-054 unit contract).
	DotInterval int32 `json:"dotInterval"`
	// DotTime is the DoT lifetime in MILLISECONDS. WZ stores seconds; the
	// reader converts.
	DotTime int32 `json:"dotTime"`
```

- [ ] **Step 4: Add the builder fields, setters, getters, and Build wiring**

In `services/atlas-data/atlas.com/data/skill/effect/model.go`, add to the `ModelBuilder` struct next to `fixDamage`:

```go
	dot                  int32
	dotInterval          int32
	dotTime              int32
```

Add the setters and getters next to `SetLT`/`SetRB` (around line 283):

```go
// SetDot sets the raw per-tick DoT magnitude (WZ `dot`, unscaled).
func (b *ModelBuilder) SetDot(v int32) *ModelBuilder {
	b.dot = v
	return b
}

// Dot returns the raw per-tick DoT magnitude.
func (b *ModelBuilder) Dot() int32 {
	return b.dot
}

// SetDotInterval sets the DoT tick interval in MILLISECONDS.
func (b *ModelBuilder) SetDotInterval(v int32) *ModelBuilder {
	b.dotInterval = v
	return b
}

// DotInterval returns the DoT tick interval in milliseconds.
func (b *ModelBuilder) DotInterval() int32 {
	return b.dotInterval
}

// SetDotTime sets the DoT lifetime in MILLISECONDS.
func (b *ModelBuilder) SetDotTime(v int32) *ModelBuilder {
	b.dotTime = v
	return b
}

// DotTime returns the DoT lifetime in milliseconds.
func (b *ModelBuilder) DotTime() int32 {
	return b.dotTime
}
```

In `func (b *ModelBuilder) Build() RestModel` (around line 376), add the three assignments to the returned struct literal, next to `FixDamage: b.fixDamage,`:

```go
		Dot:         b.dot,
		DotInterval: b.dotInterval,
		DotTime:     b.dotTime,
```

- [ ] **Step 5: Parse the WZ nodes in the reader**

In `services/atlas-data/atlas.com/data/skill/reader.go`, in `getEffect`, immediately after the `e.SetLT(...).SetRB(...)` block (currently around line 234), insert:

```go
	// DoT fields. `dot` is a raw damage-per-tick integer, forwarded unscaled.
	// `dotInterval` and `dotTime` are WZ SECONDS -- converted to milliseconds
	// HERE, the single conversion point, matching the `time` treatment above
	// (task-054). No downstream service may re-scale.
	//
	// These nodes are absent from every provisioned WZ corpus (they first
	// appear in v1.17-era Skill.wz, and there only as `common`-block formula
	// strings that this reader does not walk). The parse is additive and
	// forward-compatible: a later re-ingest that carries them needs no
	// plumbing change. See task-200 design §2.1.
	e.SetDot(node.GetIntegerWithDefault("dot", 0)).
		SetDotInterval(node.GetIntegerWithDefault("dotInterval", 0) * 1000).
		SetDotTime(node.GetIntegerWithDefault("dotTime", 0) * 1000)
```

If `GetIntegerWithDefault` does not return `int32`, wrap each call in `int32(...)` to match the setter signatures — check the existing call sites in the same function for the returned type and follow them.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go test ./skill/... -run 'TestReader_Dot' -v`
Expected: PASS (3 tests).

- [ ] **Step 7: Run the full atlas-data suite**

Run: `cd services/atlas-data/atlas.com/data && go build ./... && go vet ./... && go test -race ./...`
Expected: all PASS. No existing test may change shape (FR-1.3).

- [ ] **Step 8: Commit**

```bash
git add services/atlas-data/atlas.com/data/skill/
git commit -m "feat(atlas-data): parse skill dot/dotInterval/dotTime with seconds->ms at the reader"
```

---

### Task 2: atlas-channel — hydrate the DoT fields into the effect model

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`
- Test: `services/atlas-channel/atlas.com/channel/data/skill/effect/rest_dot_test.go` (create)

**Interfaces:**
- Consumes: Task 1's `dot` / `dotInterval` / `dotTime` JSON keys.
- Produces: `effect.Model` getters `Dot() int32`, `DotInterval() int32`, `DotTime() int32`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/data/skill/effect/rest_dot_test.go`:

```go
package effect

import (
	"encoding/json"
	"testing"
)

// TestExtract_DotFields_RoundTrip asserts the three DoT fields survive the
// atlas-data REST payload -> atlas-channel effect.Model hydration with their
// millisecond values intact. atlas-data converts dotInterval/dotTime from WZ
// seconds to ms at its reader (task-200 FR-1.2); the channel must NOT
// re-scale.
func TestExtract_DotFields_RoundTrip(t *testing.T) {
	const payload = `{"dot":105,"dotInterval":1000,"dotTime":4000}`

	var rm RestModel
	if err := json.Unmarshal([]byte(payload), &rm); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Dot() != 105 {
		t.Fatalf("m.Dot() = %d, want 105", m.Dot())
	}
	if m.DotInterval() != 1000 {
		t.Fatalf("m.DotInterval() = %d, want 1000 (ms)", m.DotInterval())
	}
	if m.DotTime() != 4000 {
		t.Fatalf("m.DotTime() = %d, want 4000 (ms)", m.DotTime())
	}
}

// TestExtract_DotFields_AbsentDefaultToZero asserts a payload without the DoT
// keys hydrates to zeros rather than failing -- which is the state of every
// provisioned tenant today (task-200 design §2.1).
func TestExtract_DotFields_AbsentDefaultToZero(t *testing.T) {
	const payload = `{"duration":4000}`

	var rm RestModel
	if err := json.Unmarshal([]byte(payload), &rm); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Dot() != 0 || m.DotInterval() != 0 || m.DotTime() != 0 {
		t.Fatalf("dot fields = (%d,%d,%d), want (0,0,0)", m.Dot(), m.DotInterval(), m.DotTime())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/skill/effect/ -run TestExtract_DotFields -v`
Expected: FAIL — compile error, `m.Dot undefined`.

- [ ] **Step 3: Add the RestModel fields**

In `services/atlas-channel/atlas.com/channel/data/skill/effect/rest.go`, add to `RestModel` after `FixDamage`:

```go
	// Dot is the raw per-tick DoT magnitude. DotInterval and DotTime are
	// MILLISECONDS -- atlas-data converts from WZ seconds at its reader
	// (task-200 FR-1.2). Do not re-scale.
	Dot         int32 `json:"dot"`
	DotInterval int32 `json:"dotInterval"`
	DotTime     int32 `json:"dotTime"`
```

In the `Extract` function's returned `Model{...}` literal, add next to `fixDamage: rm.FixDamage,`:

```go
		dot:         rm.Dot,
		dotInterval: rm.DotInterval,
		dotTime:     rm.DotTime,
```

- [ ] **Step 4: Add the Model fields and getters**

In `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`, add to the `Model` struct next to `fixDamage`:

```go
	dot                  int32
	dotInterval          int32
	dotTime              int32
```

Add the getters next to `LT()`/`RB()` (around line 129):

```go
// Dot returns the raw per-tick damage-over-time magnitude from WZ `dot`.
// It is zero on every provisioned version -- the node does not exist in any
// pre-v1.17 Skill.wz (task-200 design §2.1). Callers that need a DoT
// magnitude must not assume this is populated.
func (m Model) Dot() int32 {
	return m.dot
}

// DotInterval returns the DoT tick interval in MILLISECONDS (atlas-data
// converts from WZ seconds). Zero on every provisioned version.
func (m Model) DotInterval() int32 {
	return m.dotInterval
}

// DotTime returns the DoT lifetime in MILLISECONDS (atlas-data converts from
// WZ seconds). Zero on every provisioned version.
func (m Model) DotTime() int32 {
	return m.dotTime
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/skill/effect/... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/skill/effect/
git commit -m "feat(atlas-channel): hydrate skill dot/dotInterval/dotTime into the effect model"
```

---

### Task 3: atlas-maps — generalize the mist contract with target/effect kinds

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go`
- Modify: `services/atlas-maps/atlas.com/maps/mist/model.go`
- Modify: `services/atlas-maps/atlas.com/maps/mist/processor.go`
- Test: `services/atlas-maps/atlas.com/maps/mist/model_test.go`, `services/atlas-maps/atlas.com/maps/mist/processor_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - Constants `mistKafka.TargetKindCharacter = "CHARACTER"`, `TargetKindMonster = "MONSTER"`, `EffectKindDisease = "DISEASE"`, `EffectKindDamageOverTime = "DAMAGE_OVER_TIME"`.
  - `CreateCommandBody` fields `TargetKind string \`json:"targetKind"\``, `EffectKind string \`json:"effectKind"\``.
  - `mist.Mist` getters `TargetKind() string`, `EffectKind() string`, `Rect() (x1, y1, x2, y2 int16)`.
  - `mist.Builder` method `SetKinds(targetKind, effectKind string) *Builder`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-maps/atlas.com/maps/mist/model_test.go`:

```go
// TestMist_Kinds_RoundTrip asserts the target/effect descriptors survive the
// builder (task-200 FR-2.5). mkField is the file's existing helper (line 13).
func TestMist_Kinds_RoundTrip(t *testing.T) {
	f := mkField(t)
	m := NewBuilder(uuid.New(), f).
		SetKinds(mistKafka.TargetKindMonster, mistKafka.EffectKindDamageOverTime).
		Build()
	if m.TargetKind() != "MONSTER" {
		t.Fatalf("m.TargetKind() = %q, want MONSTER", m.TargetKind())
	}
	if m.EffectKind() != "DAMAGE_OVER_TIME" {
		t.Fatalf("m.EffectKind() = %q, want DAMAGE_OVER_TIME", m.EffectKind())
	}
}

// TestMist_Rect_AgreesWithContains pins Rect() against Contains() on the
// boundary coordinates, so the two rectangle derivations cannot drift.
func TestMist_Rect_AgreesWithContains(t *testing.T) {
	f := mkField(t)
	m := NewBuilder(uuid.New(), f).
		SetOrigin(500, 300).
		SetBounds(-110, -82, 110, 83).
		Build()

	x1, y1, x2, y2 := m.Rect()
	if x1 != 390 || y1 != 218 || x2 != 610 || y2 != 383 {
		t.Fatalf("m.Rect() = (%d,%d,%d,%d), want (390,218,610,383)", x1, y1, x2, y2)
	}
	// Every rect corner is inside (Contains is inclusive of edges).
	for _, p := range [][2]int16{{x1, y1}, {x2, y2}, {x1, y2}, {x2, y1}} {
		if !m.Contains(p[0], p[1]) {
			t.Fatalf("m.Contains(%d,%d) = false, want true", p[0], p[1])
		}
	}
	// One unit outside each edge is outside.
	if m.Contains(x1-1, y1) || m.Contains(x2+1, y2) || m.Contains(x1, y1-1) || m.Contains(x2, y2+1) {
		t.Fatal("Contains returned true outside the Rect bounds")
	}
}
```

Add the `mistKafka "atlas-maps/kafka/message/mist"` import to `model_test.go`.

Append to `services/atlas-maps/atlas.com/maps/mist/processor_test.go`. It already defines `recordingProducer` (line 26), `newRecordingProducer` (line 31), and `newTestMistProcessor(t, tt, rec) (*ProcessorImpl, context.Context)` (line 56) — reuse them; do not add duplicates. Follow `TestProcessor_Create_AddsToRegistryAndEmitsCreated` (line 69) for how the tenant is built.

```go
// TestProcessor_Create_EmptyKinds_NormalizeToCharacterDisease pins FR-2.3: the
// existing atlas-monsters AREA_POISON producer omits the descriptors, and its
// behavior must be unchanged.
func TestProcessor_Create_EmptyKinds_NormalizeToCharacterDisease(t *testing.T) {
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := newTestMistProcessor(t, tt, newRecordingProducer())

	m, err := p.Create(mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: "MONSTER", OwnerId: 7,
		Duration: 5000, TickIntervalMs: 1000,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if m.TargetKind() != mistKafka.TargetKindCharacter {
		t.Fatalf("TargetKind = %q, want CHARACTER", m.TargetKind())
	}
	if m.EffectKind() != mistKafka.EffectKindDisease {
		t.Fatalf("EffectKind = %q, want DISEASE", m.EffectKind())
	}
}

// TestProcessor_Create_ExplicitKinds_RoundTrip pins the player-cast path.
func TestProcessor_Create_ExplicitKinds_RoundTrip(t *testing.T) {
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := newTestMistProcessor(t, tt, newRecordingProducer())

	m, err := p.Create(mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: "CHARACTER", OwnerId: 1001,
		TargetKind: mistKafka.TargetKindMonster,
		EffectKind: mistKafka.EffectKindDamageOverTime,
		Duration:   4000, TickIntervalMs: 1000,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if m.TargetKind() != mistKafka.TargetKindMonster || m.EffectKind() != mistKafka.EffectKindDamageOverTime {
		t.Fatalf("kinds = (%q,%q), want (MONSTER,DAMAGE_OVER_TIME)", m.TargetKind(), m.EffectKind())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./mist/ -v`
Expected: FAIL — compile error, `SetKinds undefined`, `m.Rect undefined`, `TargetKind undefined`.

- [ ] **Step 3: Add the contract constants and command fields**

In `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go`, add to the existing `const` block:

```go
	// TargetKind selects who a mist's per-tick effect is applied to. An empty
	// value means CHARACTER, so producers written before task-200 (the
	// atlas-monsters AREA_POISON path) keep working unchanged.
	TargetKindCharacter = "CHARACTER"
	TargetKindMonster   = "MONSTER"

	// EffectKind selects what the mist's per-tick effect does. An empty value
	// means DISEASE. DISEASE applies a named character status via
	// COMMAND_TOPIC_CHARACTER_BUFF; DAMAGE_OVER_TIME applies a damage-bearing
	// monster status via COMMAND_TOPIC_MONSTER APPLY_STATUS.
	EffectKindDisease        = "DISEASE"
	EffectKindDamageOverTime = "DAMAGE_OVER_TIME"
```

Add to `CreateCommandBody`, after `SourceSkillLevel`:

```go
	// TargetKind is "CHARACTER" or "MONSTER"; empty means CHARACTER.
	TargetKind string `json:"targetKind"`
	// EffectKind is "DISEASE" or "DAMAGE_OVER_TIME"; empty means DISEASE.
	EffectKind string `json:"effectKind"`
```

Do **not** rename `Disease` / `DiseaseValue` / `DiseaseDuration` / `TickIntervalMs` — they are the generic status name / magnitude / per-target duration / tick interval quadruple and their JSON keys are a live contract (design §4.1 D4).

- [ ] **Step 4: Add the model fields, setter, getters, and `Rect()`**

In `services/atlas-maps/atlas.com/maps/mist/model.go`:

Update the type doc comment on `Mist` to say "status" rather than "disease", and note the descriptors:

```go
// Mist represents an area-of-effect mist field placed on a map. It carries a
// status effect that is applied on each tick to whatever its targetKind names
// -- characters (the monster AREA_POISON path) or monsters (player-cast
// mists). The disease* fields are the generic status name / magnitude /
// per-target duration triple; the names are historical.
```

Add to both the `Mist` struct and the `Builder` struct, after `tickInterval`:

```go
	targetKind       string
	effectKind       string
```

Add the getters next to `Disease()`:

```go
// TargetKind reports who this mist's per-tick effect applies to: CHARACTER or
// MONSTER. Never empty on a mist built through Processor.Create, which
// normalizes an absent value to CHARACTER.
func (m Mist) TargetKind() string {
	return m.targetKind
}

// EffectKind reports what this mist's per-tick effect does: DISEASE or
// DAMAGE_OVER_TIME. Never empty on a mist built through Processor.Create.
func (m Mist) EffectKind() string {
	return m.effectKind
}
```

Add `Rect()` immediately above `Contains`, and rewrite `Contains` to use it so the two cannot drift:

```go
// Rect returns the mist's absolute axis-aligned bounding box in world
// coordinates: (x1, y1) top-left, (x2, y2) bottom-right. Bounds are inclusive,
// matching Contains and the atlas-monsters in-rect endpoint.
func (m Mist) Rect() (int16, int16, int16, int16) {
	return m.originX + m.ltX, m.originY + m.ltY, m.originX + m.rbX, m.originY + m.rbY
}

// Contains reports whether the given world coordinates fall within the mist's
// axis-aligned bounding box (inclusive of edges).
func (m Mist) Contains(x, y int16) bool {
	minX, minY, maxX, maxY := m.Rect()
	return x >= minX && x <= maxX && y >= minY && y <= maxY
}
```

Add the grouped setter next to `SetDisease`:

```go
// SetKinds sets the target and effect descriptors. Grouped rather than split
// into two single-field setters because the pair is meaningless apart.
func (b *Builder) SetKinds(targetKind, effectKind string) *Builder {
	b.targetKind = targetKind
	b.effectKind = effectKind
	return b
}
```

Add both fields to the `Build()` struct literal:

```go
		targetKind:       b.targetKind,
		effectKind:       b.effectKind,
```

- [ ] **Step 5: Normalize the kinds in `Processor.Create`**

In `services/atlas-maps/atlas.com/maps/mist/processor.go`, at the top of `Create`, before the builder chain:

```go
	// Normalize the descriptors exactly once, here, so every Mist in the
	// registry has non-empty kinds and the tick task can switch on them
	// without an empty-string case. This is what gives the pre-task-200
	// atlas-monsters producer byte-for-byte unchanged behavior (FR-2.3).
	targetKind := body.TargetKind
	if targetKind == "" {
		targetKind = mistKafka.TargetKindCharacter
	}
	effectKind := body.EffectKind
	if effectKind == "" {
		effectKind = mistKafka.EffectKindDisease
	}
```

and add to the builder chain, after `SetSource(...)`:

```go
		SetKinds(targetKind, effectKind).
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./mist/... -v`
Expected: PASS, including every pre-existing test in the package unchanged.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/message/mist/ services/atlas-maps/atlas.com/maps/mist/
git commit -m "feat(atlas-maps): add mist targetKind/effectKind descriptors and Mist.Rect()"
```

---

### Task 4: Carry the render values on `MIST_CREATED` instead of channel constants

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go`
- Modify: `services/atlas-maps/atlas.com/maps/mist/model.go`
- Modify: `services/atlas-maps/atlas.com/maps/mist/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go`
- Test: `services/atlas-maps/atlas.com/maps/mist/producer_test.go`, `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer_test.go`

**Interfaces:**
- Consumes: Task 3's `mist.Mist`.
- Produces: `CreatedBody` fields `ElemAttr int32 \`json:"elemAttr"\``, `SkillDelay int16 \`json:"skillDelay"\`` on both the atlas-maps and atlas-channel sides; `mist.Mist` getters `ElemAttr() int32`, `SkillDelay() int16`; `mist.Builder` method `SetRender(elemAttr int32, skillDelay int16) *Builder`.

**The wire bytes must not change.** Both values are `0` for both mist kinds (design §3.4). This task moves *where* the zero comes from, not what it is.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-maps/atlas.com/maps/mist/producer_test.go`:

```go
// TestCreatedEventProvider_CarriesRenderValues asserts MIST_CREATED carries
// the render values from the model rather than leaving the channel to
// hard-code them (task-200 FR-2.4). Both are 0 for every mist Atlas creates
// today; the plumbing exists so a future mist kind can differ without a
// contract change.
func TestCreatedEventProvider_CarriesRenderValues(t *testing.T) {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()

	// Non-zero values prove the event carries the MODEL's values. A test that
	// only ever asserted 0 would pass against the unchanged code.
	m := NewBuilder(uuid.New(), f).SetRender(7, 3).Build()

	msgs, err := createdEventProvider(tn, m)()
	if err != nil {
		t.Fatalf("createdEventProvider: %v", err)
	}
	var ev mistKafka.Event[mistKafka.CreatedBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if ev.Body.ElemAttr != 7 {
		t.Fatalf("ev.Body.ElemAttr = %d, want 7", ev.Body.ElemAttr)
	}
	if ev.Body.SkillDelay != 3 {
		t.Fatalf("ev.Body.SkillDelay = %d, want 3", ev.Body.SkillDelay)
	}

	// And the default is 0 for every mist Atlas actually creates.
	zero := NewBuilder(uuid.New(), f).Build()
	if zero.ElemAttr() != 0 || zero.SkillDelay() != 0 {
		t.Fatalf("unset render values = (%d,%d), want (0,0)", zero.ElemAttr(), zero.SkillDelay())
	}
}
```

Add `encoding/json`, `field`, `tenant`, `uuid`, and `mistKafka` imports if the file lacks them.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./mist/ -run TestCreatedEventProvider_CarriesRenderValues -v`
Expected: FAIL — compile error, `SetRender undefined`.

- [ ] **Step 3: Add the model fields and setter**

In `services/atlas-maps/atlas.com/maps/mist/model.go`, add to both `Mist` and `Builder` after `effectKind`:

```go
	elemAttr         int32
	skillDelay       int16
```

Add getters next to `Type()`:

```go
// ElemAttr returns the AffectedAreaCreated `nElemAttr` wire value. The client
// stores it raw at AFFECTEDAREA+0x30 (v83 @0x431b3b, v95 @0x437fd9) and never
// reads it on any rendering path -- it takes the skill's element from its own
// Skill.wz. Atlas models no mist element, so this is 0 for every mist.
func (m Mist) ElemAttr() int32 {
	return m.elemAttr
}

// SkillDelay returns the AffectedAreaCreated `skillDelay` wire value: a
// DRAW DELAY in units of 100 ms, not a lifetime. The client computes
// tStart = get_update_time() + 100*skillDelay (v83 @0x431b50, v95 @0x437fa3)
// and gates the mist's first draw on it, so any non-zero value hides the mist
// for that long. Atlas has no per-mist cast delay to express: 0 = draw now.
func (m Mist) SkillDelay() int16 {
	return m.skillDelay
}
```

Add the setter next to `SetKinds`:

```go
// SetRender sets the client-render wire values carried on MIST_CREATED. Both
// are 0 for every mist Atlas creates; see the getters for why.
func (b *Builder) SetRender(elemAttr int32, skillDelay int16) *Builder {
	b.elemAttr = elemAttr
	b.skillDelay = skillDelay
	return b
}
```

Add both to the `Build()` literal.

- [ ] **Step 4: Add the event fields on both sides and emit them**

In `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go`, add to `CreatedBody` after `Duration`:

```go
	// ElemAttr is the client's `nElemAttr`; SkillDelay is its `skillDelay`
	// draw delay (units of 100 ms). The existing `Type` field IS the client's
	// `nType` -- do not add a second key for it.
	ElemAttr   int32 `json:"elemAttr"`
	SkillDelay int16 `json:"skillDelay"`
```

Make the identical addition to `CreatedBody` in `services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go`.

In `services/atlas-maps/atlas.com/maps/mist/producer.go`, add to the `CreatedBody{...}` literal in `createdEventProvider`:

```go
			ElemAttr:   m.ElemAttr(),
			SkillDelay: m.SkillDelay(),
```

- [ ] **Step 5: Consume them in the channel**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go`, in `handleMistCreated`, replace the `mistSkillDelay` and `mistElemAttr` arguments in the `fieldpkt.NewAffectedAreaCreated(...)` call with `e.Body.SkillDelay` and `e.Body.ElemAttr`. Leave `mistPhase` as-is — `nPhase` is not carried on the event.

Then replace the `mistSkillDelay` and `mistElemAttr` const declarations with a single doc block explaining that atlas-maps now owns the values, keeping the IDB citations (they are the justification for the zeros and must not be lost):

```go
// The `skillDelay` and `nElemAttr` wire values now travel on MIST_CREATED
// (atlas-maps mist.Mist.SkillDelay() / .ElemAttr(), where the full IDB
// rationale lives). Both are 0 for every mist Atlas creates today.
//
// mistPhase is the GMS v92+ `nPhase` wire value (AFFECTEDAREA+0x48, v95
// @0x437fde). It is compared for equality only inside IsSmokeAreaByPoint /
// GetAffectAreaByPoint, neither of which any Atlas mist can reach (task-200
// design §3.3-3.4). Atlas does not model it; 0 matches the legacy versions,
// which omit the field entirely.
const mistPhase = int32(0)
```

- [ ] **Step 6: Add the channel-side consumer test**

Append to `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer_test.go`. It already has `withRecordingBroadcasters`, `newTestTenant`, and `newTestServer` — reuse them.

```go
// TestMistCreated_UsesEventRenderValues asserts the broadcast packet takes
// skillDelay and nElemAttr from the event rather than a channel-local constant
// (task-200 FR-2.4) -- and that the resulting values are still 0, so the wire
// bytes of every already-verified SPAWN_MIST fixture are unchanged (FR-5.5).
func TestMistCreated_UsesEventRenderValues(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, createdCalls, lastCreated, _, _ := withRecordingBroadcasters(t)
	defer restore()

	handleMistCreated(sc, nil)(logrus.New(), ctx, mist2.Event[mist2.CreatedBody]{
		Tenant: tm.Id(), WorldId: 0, ChannelId: 1, MapId: 100000000,
		Instance: uuid.Nil, MistId: uuid.New(), Type: mist2.EventTypeCreated,
		Body: mist2.CreatedBody{
			OwnerType: "CHARACTER", OwnerId: 1001,
			SourceSkillId: 2111003, SourceSkillLevel: 1,
			Type:    0,
			OriginX: 500, OriginY: 300,
			LtX: -110, LtY: -82, RbX: 110, RbY: 83,
			Duration:   4000,
			ElemAttr:   0,
			SkillDelay: 0,
		},
	})

	if *createdCalls != 1 {
		t.Fatalf("createdCalls = %d, want 1", *createdCalls)
	}
	if lastCreated.SkillDelay() != 0 {
		t.Fatalf("SkillDelay() = %d, want 0 (non-zero hides the mist)", lastCreated.SkillDelay())
	}
	if lastCreated.ElemAttr() != 0 {
		t.Fatalf("ElemAttr() = %d, want 0", lastCreated.ElemAttr())
	}
	if lastCreated.NType() != 0 {
		t.Fatalf("NType() = %d, want 0 (3 is the area-buff-ITEM arm)", lastCreated.NType())
	}
	if lastCreated.Phase() != 0 {
		t.Fatalf("Phase() = %d, want 0", lastCreated.Phase())
	}
	if lastCreated.OwnerId() != 1001 {
		t.Fatalf("OwnerId() = %d, want the casting character id 1001", lastCreated.OwnerId())
	}
}

// TestMistCreated_NonZeroRenderValuesPropagate proves the values genuinely
// come from the event and are not still hard-coded to 0 -- a test that only
// ever asserts 0 would pass against the unchanged code.
func TestMistCreated_NonZeroRenderValuesPropagate(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, _, lastCreated, _, _ := withRecordingBroadcasters(t)
	defer restore()

	handleMistCreated(sc, nil)(logrus.New(), ctx, mist2.Event[mist2.CreatedBody]{
		Tenant: tm.Id(), WorldId: 0, ChannelId: 1, MapId: 100000000,
		Instance: uuid.Nil, MistId: uuid.New(), Type: mist2.EventTypeCreated,
		Body: mist2.CreatedBody{ElemAttr: 7, SkillDelay: 3},
	})

	if lastCreated.ElemAttr() != 7 {
		t.Fatalf("ElemAttr() = %d, want 7 (value must come from the event)", lastCreated.ElemAttr())
	}
	if lastCreated.SkillDelay() != 3 {
		t.Fatalf("SkillDelay() = %d, want 3 (value must come from the event)", lastCreated.SkillDelay())
	}
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run:
```
cd services/atlas-maps/atlas.com/maps && go test ./... 2>&1 | tail -20
cd ../../../atlas-channel/atlas.com/channel && go test ./kafka/... 2>&1 | tail -20
cd ../../../../libs/atlas-packet && go test ./field/... 2>&1 | tail -20
```
Expected: all PASS. The `libs/atlas-packet` field fixtures must pass **unmodified** — if any fails, the wire bytes changed and the change is wrong.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/ services/atlas-channel/atlas.com/channel/kafka/
git commit -m "feat(mist): carry elemAttr/skillDelay on MIST_CREATED instead of channel constants"
```

---

### Task 5: atlas-monsters — set the kinds explicitly on the AREA_POISON producer

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (`buildMistCreateBody`, ~line 1067)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` (existing `buildMistCreateBody` tests at :1185, :1245, :1272)

**Interfaces:**
- Consumes: Task 3's `mistKafka.TargetKindCharacter` / `EffectKindDisease` constants (atlas-monsters imports the atlas-maps mist message package as `mistKafka` — confirm the existing import alias before adding).
- Produces: nothing new.

**Behavior must not change** (NFR-5). This makes the implicit default explicit.

- [ ] **Step 1: Add the failing assertions to the existing test**

In the existing `buildMistCreateBody` test around `processor_test.go:1185`, add to the assertion block:

```go
	if body.TargetKind != mistKafka.TargetKindCharacter {
		t.Fatalf("body.TargetKind = %q, want CHARACTER", body.TargetKind)
	}
	if body.EffectKind != mistKafka.EffectKindDisease {
		t.Fatalf("body.EffectKind = %q, want DISEASE", body.EffectKind)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run Mist -v`
Expected: FAIL — `body.TargetKind = "", want CHARACTER`.

- [ ] **Step 3: Set the fields**

In `buildMistCreateBody`, add to the returned `mistKafka.CreateCommandBody{...}` literal, after `SourceSkillLevel`:

```go
		// Explicit rather than relying on atlas-maps' empty-value default.
		// A monster AREA_POISON mist poisons CHARACTERS with a named status;
		// the player-cast mists added in task-200 target MONSTERS with a DoT.
		TargetKind: mistKafka.TargetKindCharacter,
		EffectKind: mistKafka.EffectKindDisease,
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go vet ./... && go test -race ./...`
Expected: all PASS, including every pre-existing mist test.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/
git commit -m "refactor(atlas-monsters): set mist targetKind/effectKind explicitly (no behavior change)"
```

---

### Task 6: atlas-maps — `GetInMapRect` on the monster client

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/monster/requests.go`
- Modify: `services/atlas-maps/atlas.com/maps/monster/processor.go`
- Test: `services/atlas-maps/atlas.com/maps/monster/processor_rect_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `monster.Processor` method `GetInMapRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]RestModel, error)`. Returns `RestModel` (not a domain model) — the package has no domain model and `RestModel` already carries `Id` (the monster **unique** id, as a string) plus `X`/`Y`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-maps/atlas.com/maps/monster/processor_rect_test.go`. It lives in the external test package `monster_test`, alongside `processor_drain_test.go`, and reuses that file's `MONSTERS_SERVICE_URL` override pattern.

```go
package monster_test

import (
	"atlas-maps/monster"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// rectDoc renders a JSON:API "monsters" document carrying position
// attributes, so the rect result's X/Y round-trip can be asserted.
func rectDoc(from, to int, total, number, size, last int) string {
	body := ""
	for id := from; id <= to; id++ {
		if body != "" {
			body += ","
		}
		body += fmt.Sprintf(`{"id":"%d","type":"monsters","attributes":{"x":%d,"y":300}}`, id, 400+id)
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		body, total, number, size, last,
	)
}

// TestGetInMapRect_DrainsAllPagesAndCarriesBounds asserts the rect query is
// drained across pages (a truncated page-1 result would silently under-apply a
// mist) and that the request carries the inclusive bounds and the limit.
func TestGetInMapRect_DrainsAllPagesAndCarriesBounds(t *testing.T) {
	var firstQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if firstQuery == "" {
			firstQuery = r.URL.RawQuery
		}
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(rectDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(rectDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("MONSTERS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()

	got, err := monster.NewProcessor(l, ctx).GetInMapRect(f, 390, 218, 610, 383, 0)
	if err != nil {
		t.Fatalf("GetInMapRect: %v", err)
	}
	if len(got) != 300 {
		t.Fatalf("len(got) = %d, want 300 (drain must not stop at page 1)", len(got))
	}
	for _, want := range []string{"x1=390", "y1=218", "x2=610", "y2=383", "limit=0"} {
		if !strings.Contains(firstQuery, want) {
			t.Fatalf("query %q missing %q", firstQuery, want)
		}
	}
	if got[0].Id != "1" {
		t.Fatalf("got[0].Id = %q, want \"1\" (the monster unique id)", got[0].Id)
	}
	if got[0].X != 401 || got[0].Y != 300 {
		t.Fatalf("got[0] position = (%d,%d), want (401,300)", got[0].X, got[0].Y)
	}
}
```

Add the `strings` import. If `processor_drain_test.go`'s document renderer already suits the assertions, reuse it instead of adding `rectDoc`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./monster/ -run TestGetInMapRect -v`
Expected: FAIL — compile error, `GetInMapRect undefined`.

- [ ] **Step 3: Add the URL builder**

In `services/atlas-maps/atlas.com/maps/monster/requests.go`, add to the `const` block:

```go
	mapMonstersRectResource = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters/in-rect?x1=%d&y1=%d&x2=%d&y2=%d&limit=%d"
```

and the builder, mirroring `atlas-channel/monster/requests.go:33`:

```go
// inMapRectUrl returns the list URL for the atlas-monsters rectangle query.
// Bounds are inclusive; limit == 0 means "no cap". Bare URL (not a
// requests.Request) because the list is paginated server-side and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size] params.
func inMapRectUrl(f field.Model, x1, y1, x2, y2 int16, limit uint32) string {
	return fmt.Sprintf(getBaseRequest()+mapMonstersRectResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String(), x1, y1, x2, y2, limit)
}
```

- [ ] **Step 4: Add the processor method**

In `services/atlas-maps/atlas.com/maps/monster/processor.go`, add to the `Processor` interface:

```go
	GetInMapRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]RestModel, error)
```

and the implementation:

```go
// GetInMapRect returns every monster whose position falls inside the inclusive
// world-coordinate rectangle. The atlas-monsters endpoint is authoritative for
// the containment test -- callers must NOT re-filter the result, because a
// second filter with a different edge convention would silently disagree with
// the server and mask any endpoint bug. One authority per question.
//
// limit == 0 means "no cap". The result is drained across all pages.
func (p *ProcessorImpl) GetInMapRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]RestModel, error) {
	return requests.DrainProvider[RestModel, RestModel](p.l, p.ctx)(inMapRectUrl(f, x1, y1, x2, y2, limit), 250, Extract, model.Filters[RestModel]())()
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./monster/... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/monster/
git commit -m "feat(atlas-maps): add monster GetInMapRect client for mist targeting"
```

---

### Task 7: atlas-maps — extract `tickCharacters` (pure refactor)

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go`
- Test: `services/atlas-maps/atlas.com/maps/tasks/mist_tick_test.go` (must pass **unmodified**)

**Interfaces:**
- Consumes: Task 3's `mist.Mist.TargetKind()`.
- Produces: `func (r *MistTick) tickCharacters(ctx context.Context, prov producer.Provider, t tenant.Model, m mist.Mist)`.

This task changes no behavior. Its whole purpose is that the existing test file passes untouched, proving the character path is bit-identical (NFR-5).

- [ ] **Step 1: Confirm the baseline is green**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./tasks/ -v`
Expected: PASS. Record the test count — it must be identical after the refactor.

- [ ] **Step 2: Extract the character body verbatim**

In `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go`, move the block in `processTenant` from `members := r.charsInField(...)` through the `emitErr` handling into a new method, unchanged:

```go
// tickCharacters applies the mist's status to every character in the field
// whose position falls inside the mist's bounding box. This is the original
// (pre-task-200) mist tick body, extracted verbatim -- the monster AREA_POISON
// path must behave identically before and after (NFR-5).
func (r *MistTick) tickCharacters(ctx context.Context, prov producer.Provider, t tenant.Model, m mist.Mist) {
	members := r.charsInField(t, m.Field())
	if len(members) == 0 {
		return
	}
	emitErr := message.Emit(prov)(func(buf *message.Buffer) error {
		for _, cid := range members {
			x, y, err := r.posLookup(ctx, cid)
			if err != nil {
				r.l.WithError(err).Debugf("MistTick: position fetch failed for character [%d].", cid)
				continue
			}
			if !m.Contains(x, y) {
				continue
			}
			if err := buf.Put(EnvCommandTopicCharacterBuff, applyDiseaseCommandProvider(m, cid)); err != nil {
				return err
			}
		}
		return nil
	})
	if emitErr != nil {
		r.l.WithError(emitErr).Errorf("MistTick: failed to emit apply-disease for mist [%s].", m.Id())
	}
}
```

Note the early `return` replaces the original's `UpdateLastTick + continue`; the caller must now always call `UpdateLastTick`. Replace the extracted block in `processTenant` with:

```go
		r.tickCharacters(tctx, prov, t, m)
		r.registry.UpdateLastTick(t, m.Id(), time.Now())
```

- [ ] **Step 3: Run tests to verify they still pass, unmodified**

Run: `cd services/atlas-maps/atlas.com/maps && go test -race ./tasks/ -v`
Expected: PASS, same test count as Step 1, with **zero edits** to `mist_tick_test.go`. If a test needed editing, the refactor was not behavior-preserving — revert and redo.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/tasks/mist_tick.go
git commit -m "refactor(atlas-maps): extract tickCharacters from the mist tick body"
```

---

### Task 8: atlas-maps — the `MONSTER` tick branch

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go`
- Test: `services/atlas-maps/atlas.com/maps/tasks/mist_tick_monster_test.go` (create)

**Interfaces:**
- Consumes: Task 3's `Mist.TargetKind()` / `Mist.Rect()`; Task 6's `monster.Processor.GetInMapRect`.
- Produces:
  - `MistTick` field `monstersInRect func(ctx context.Context, m mist.Mist) ([]monster.RestModel, error)` — the injectable seam.
  - `func (r *MistTick) tickMonsters(ctx context.Context, prov producer.Provider, t tenant.Model, m mist.Mist)`.
  - `func applyStatusCommandProvider(m mist.Mist, monsterUniqueId uint32) model.Provider[[]kafka.Message]`.
  - Const `EnvCommandTopicMonster = "COMMAND_TOPIC_MONSTER"`.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-maps/atlas.com/maps/tasks/mist_tick_monster_test.go`. It shares the `tasks` package with `mist_tick_test.go`, so reuse `recordingProducer`, `mkTickTenant`, and `newTestMistTick` from there rather than redefining them.

```go
package tasks

import (
	"atlas-maps/mist"
	"atlas-maps/monster"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	mistKafka "atlas-maps/kafka/message/mist"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// mkMonsterMist builds a player-cast mist anchored at (500,300) with the
// level-1 Poison Mist rectangle from WZ (lt -110,-82 / rb 110,83), i.e.
// absolute bounds (390,218)-(610,383). The caller registers it.
func mkMonsterMist(t *testing.T) mist.Mist {
	t.Helper()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	return mist.NewBuilder(uuid.New(), f).
		SetOwner("CHARACTER", 1001).
		SetKinds(mistKafka.TargetKindMonster, mistKafka.EffectKindDamageOverTime).
		SetSource(2111003, 1).
		SetOrigin(500, 300).
		SetBounds(-110, -82, 110, 83).
		SetDisease("POISON", 0, 4000*time.Millisecond).
		SetDuration(4000 * time.Millisecond).
		SetTickInterval(1000 * time.Millisecond).
		Build()
}

func mkMonsterRest(uniqueId uint32, x, y int16) monster.RestModel {
	return monster.RestModel{Id: strconv.Itoa(int(uniqueId)), X: x, Y: y}
}

// TestMistTick_MonsterTarget_EmitsApplyStatusPerMonster asserts one
// APPLY_STATUS per monster returned by the rect endpoint, with the exact body
// atlas-monsters' consumer expects (task-200 FR-4.2).
func TestMistTick_MonsterTarget_EmitsApplyStatusPerMonster(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	mt := newTestMistTick(t, reg, rec, func(context.Context, uint32) (int16, int16, error) {
		t.Fatal("character position lookup must not run for a MONSTER mist")
		return 0, 0, nil
	})

	m := mkMonsterMist(t)
	require.NoError(t, reg.Add(tt, m))

	var gotRect [4]int16
	mt.monstersInRect = func(_ context.Context, mm mist.Mist) ([]monster.RestModel, error) {
		x1, y1, x2, y2 := mm.Rect()
		gotRect = [4]int16{x1, y1, x2, y2}
		return []monster.RestModel{mkMonsterRest(9001, 500, 300), mkMonsterRest(9002, 610, 383)}, nil
	}

	mt.runOnce(context.Background())

	require.Equal(t, [4]int16{390, 218, 610, 383}, gotRect)

	msgs := rec.Messages(EnvCommandTopicMonster)
	require.Len(t, msgs, 2)
	require.Empty(t, rec.Messages(EnvCommandTopicCharacterBuff))

	var cmd struct {
		MonsterId uint32 `json:"monsterId"`
		Type      string `json:"type"`
		Body      struct {
			SourceType        string           `json:"sourceType"`
			SourceCharacterId uint32           `json:"sourceCharacterId"`
			SourceSkillId     uint32           `json:"sourceSkillId"`
			SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
			Statuses          map[string]int32 `json:"statuses"`
			Duration          uint32           `json:"duration"`
			TickInterval      uint32           `json:"tickInterval"`
		} `json:"body"`
	}
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))
	require.Equal(t, uint32(9001), cmd.MonsterId)
	require.Equal(t, "APPLY_STATUS", cmd.Type)
	require.Equal(t, "PLAYER_SKILL", cmd.Body.SourceType)
	require.Equal(t, uint32(1001), cmd.Body.SourceCharacterId)
	require.Equal(t, uint32(2111003), cmd.Body.SourceSkillId)
	require.Equal(t, uint32(1), cmd.Body.SourceSkillLevel)
	require.Equal(t, map[string]int32{"POISON": 0}, cmd.Body.Statuses)
	require.Equal(t, uint32(4000), cmd.Body.Duration)
	require.Equal(t, uint32(1000), cmd.Body.TickInterval)
}

// TestMistTick_MonsterTarget_BodyKeySetMatchesChannel pins the JSON key set
// against atlas-channel's ApplyStatusCommandBody. COMMAND_TOPIC_MONSTER is a
// shared topic: every registered handler unmarshals every message, so an added
// or renamed key causes decode errors on unrelated handlers.
func TestMistTick_MonsterTarget_BodyKeySetMatchesChannel(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	mt := newTestMistTick(t, reg, rec, nil)

	m := mkMonsterMist(t)
	require.NoError(t, reg.Add(tt, m))
	mt.monstersInRect = func(context.Context, mist.Mist) ([]monster.RestModel, error) {
		return []monster.RestModel{mkMonsterRest(9001, 500, 300)}, nil
	}
	mt.runOnce(context.Background())

	var envelope map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Messages(EnvCommandTopicMonster)[0].Value, &envelope))
	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(envelope["body"], &body))

	keys := make([]string, 0, len(body))
	for k := range body {
		keys = append(keys, k)
	}
	require.ElementsMatch(t, []string{
		"sourceType", "sourceCharacterId", "sourceSkillId",
		"sourceSkillLevel", "statuses", "duration", "tickInterval",
	}, keys)
}

// TestMistTick_MonsterTarget_LookupFailureIsolatedPerMist asserts a failing
// rect query on one mist does not prevent another mist in the same tenant from
// ticking (FR-4.6, NFR-2).
func TestMistTick_MonsterTarget_LookupFailureIsolatedPerMist(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	mt := newTestMistTick(t, reg, rec, nil)

	bad := mkMonsterMist(t)
	good := mkMonsterMist(t)
	require.NoError(t, reg.Add(tt, bad))
	require.NoError(t, reg.Add(tt, good))

	mt.monstersInRect = func(_ context.Context, mm mist.Mist) ([]monster.RestModel, error) {
		if mm.Id() == bad.Id() {
			return nil, errors.New("atlas-monsters unavailable")
		}
		return []monster.RestModel{mkMonsterRest(9001, 500, 300)}, nil
	}

	mt.runOnce(context.Background())
	require.Len(t, rec.Messages(EnvCommandTopicMonster), 1)
}

// TestMistTick_MonsterTarget_NoMonsters_EmitsNothing asserts an empty rect
// result produces no commands and still advances lastTick (so a persistently
// empty map does not spin).
func TestMistTick_MonsterTarget_NoMonsters_EmitsNothing(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	mt := newTestMistTick(t, reg, rec, nil)

	m := mkMonsterMist(t)
	require.NoError(t, reg.Add(tt, m))
	mt.monstersInRect = func(context.Context, mist.Mist) ([]monster.RestModel, error) {
		return nil, nil
	}

	mt.runOnce(context.Background())
	require.Empty(t, rec.Messages(EnvCommandTopicMonster))

	after := reg.AllByTenant(tt)
	require.Len(t, after, 1)
	require.False(t, after[0].ShouldTick())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./tasks/ -run MonsterTarget -v`
Expected: FAIL — compile error, `mt.monstersInRect undefined`, `EnvCommandTopicMonster undefined`.

- [ ] **Step 3: Add the topic const and the command body**

In `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go`, next to `EnvCommandTopicCharacterBuff`:

```go
// EnvCommandTopicMonster is the Kafka topic where APPLY_STATUS commands are
// published. Mirrors atlas-channel's value (services communicate via
// topic-name only -- no shared library import).
const EnvCommandTopicMonster = "COMMAND_TOPIC_MONSTER"
```

Add the mirrored envelope and body next to `buffCommand` / `applyDiseaseBody`:

```go
// monsterCommand is the COMMAND_TOPIC_MONSTER envelope, mirrored from
// atlas-channel's kafka/message/monster/kafka.go Command[E].
type monsterCommand[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// applyStatusBody is a byte-compatible mirror of atlas-channel's
// monster.ApplyStatusCommandBody.
//
// COMMAND_TOPIC_MONSTER is a SHARED topic: every registered handler in
// atlas-monsters unmarshals every message on it, so a key that is
// same-named-but-narrower in a sibling command body produces decode-error
// spam on unrelated handlers (see the KILL/useSkill skillId byte-vs-uint32
// collision). Reuse the exact existing key set -- add nothing, rename
// nothing, retype nothing.
//
// Duration and TickInterval are MILLISECONDS.
type applyStatusBody struct {
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	Duration          uint32           `json:"duration"`
	TickInterval      uint32           `json:"tickInterval"`
}

// applyStatusCommandProvider builds one APPLY_STATUS command for a monster
// standing inside the mist. Keyed on the monster unique id so it lands on the
// same partition as every other command for that monster.
//
// The POISON magnitude is intentionally the mist's DiseaseValue, which is 0
// for a player-cast mist: atlas-monsters computes poison damage as
// maxHP/(70 - sourceSkillLevel) (monster/status_task.go calculatePoisonDamage)
// and never reads the magnitude for POISON. VENOM is the status that does.
func applyStatusCommandProvider(m mist.Mist, monsterUniqueId uint32) model.Provider[[]kafka.Message] {
	key := kafkaProducer.CreateKey(int(monsterUniqueId))
	value := &monsterCommand[applyStatusBody]{
		WorldId:   m.Field().WorldId(),
		ChannelId: m.Field().ChannelId(),
		MapId:     m.Field().MapId(),
		Instance:  m.Field().Instance(),
		MonsterId: monsterUniqueId,
		Type:      "APPLY_STATUS",
		Body: applyStatusBody{
			SourceType:        "PLAYER_SKILL",
			SourceCharacterId: m.OwnerId(),
			SourceSkillId:     m.SourceSkillId(),
			SourceSkillLevel:  m.SourceSkillLevel(),
			Statuses:          map[string]int32{m.Disease(): m.DiseaseValue()},
			Duration:          uint32(m.DiseaseDuration().Milliseconds()),
			TickInterval:      uint32(m.TickInterval().Milliseconds()),
		},
	}
	return kafkaProducer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 4: Add the seam and wire it in `NewMistTick`**

Add to the `MistTick` struct:

```go
	monstersInRect   func(ctx context.Context, m mist.Mist) ([]monster.RestModel, error)
```

Add to the `NewMistTick` struct literal:

```go
		monstersInRect: func(ctx context.Context, m mist.Mist) ([]monster.RestModel, error) {
			x1, y1, x2, y2 := m.Rect()
			// limit 0 == no cap. ctx is already tenant-decorated by
			// processTenant, so the REST call is tenant-scoped (NFR-3).
			return monster.NewProcessor(l, ctx).GetInMapRect(m.Field(), x1, y1, x2, y2, 0)
		},
```

Add the `"atlas-maps/monster"` import.

- [ ] **Step 5: Add `tickMonsters` and the branch**

```go
// tickMonsters applies the mist's damage-over-time status to every monster the
// atlas-monsters rect endpoint reports inside the mist's bounding box.
//
// The endpoint is authoritative for containment -- this does NOT re-filter
// with Mist.Contains. Double-filtering would mask an endpoint bug and would
// diverge if the two rect conventions (inclusive vs exclusive edges) ever
// differed. One authority per question.
func (r *MistTick) tickMonsters(ctx context.Context, prov producer.Provider, t tenant.Model, m mist.Mist) {
	monsters, err := r.monstersInRect(ctx, m)
	if err != nil {
		r.l.WithError(err).Errorf("MistTick: monster rect lookup failed for mist [%s]; skipping this mist's tick.", m.Id())
		return
	}
	if len(monsters) == 0 {
		r.l.Debugf("MistTick: mist [%s] found 0 monsters in rect.", m.Id())
		return
	}
	emitErr := message.Emit(prov)(func(buf *message.Buffer) error {
		applied := 0
		for _, rm := range monsters {
			uniqueId, cErr := strconv.Atoi(rm.Id)
			if cErr != nil {
				r.l.WithError(cErr).Warnf("MistTick: unparseable monster id [%s] for mist [%s].", rm.Id, m.Id())
				continue
			}
			if pErr := buf.Put(EnvCommandTopicMonster, applyStatusCommandProvider(m, uint32(uniqueId))); pErr != nil {
				return pErr
			}
			applied++
		}
		r.l.Debugf("MistTick: mist [%s] applied [%s] to %d of %d monsters in rect.", m.Id(), m.Disease(), applied, len(monsters))
		return nil
	})
	if emitErr != nil {
		r.l.WithError(emitErr).Errorf("MistTick: failed to emit apply-status for mist [%s].", m.Id())
	}
}
```

Add the `"strconv"` import. In `processTenant`, replace the `r.tickCharacters(...)` call added in Task 7 with:

```go
		switch m.TargetKind() {
		case mistKafka.TargetKindMonster:
			r.tickMonsters(tctx, prov, t, m)
		default:
			// Empty target kind normalizes to CHARACTER in mist.Create; the
			// default arm also covers any mist built directly by a test.
			r.tickCharacters(tctx, prov, t, m)
		}
		r.registry.UpdateLastTick(t, m.Id(), time.Now())
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go test -race ./tasks/ -v`
Expected: PASS — the four new tests **and** every pre-existing `mist_tick_test.go` test, still unmodified.

- [ ] **Step 7: Run the whole atlas-maps module**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go vet ./... && go test -race ./...`
Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/tasks/
git commit -m "feat(atlas-maps): apply monster-targeting mists via APPLY_STATUS per tick"
```

---

### Task 9: atlas-channel — `COMMAND_TOPIC_MIST` producer

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go`
- Create: `services/atlas-channel/atlas.com/channel/mist/producer.go`
- Create: `services/atlas-channel/atlas.com/channel/mist/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/mist/producer_test.go`

**Interfaces:**
- Consumes: Task 3's contract shape (the channel keeps its own mirrored copy — services communicate via topic name, never a cross-service import).
- Produces:
  - In `kafka/message/mist`: `EnvCommandTopic = "COMMAND_TOPIC_MIST"`, `CommandTypeCreate = "CREATE"`, `TargetKindCharacter`/`TargetKindMonster`/`EffectKindDisease`/`EffectKindDamageOverTime`, `Command[E any]`, `CreateCommandBody` (identical JSON keys to atlas-maps').
  - `mist.CreateCommandProvider(body mistmsg.CreateCommandBody) model.Provider[[]kafka.Message]`.
  - `mist.Processor` interface with `Create(body mistmsg.CreateCommandBody) error`, and `mist.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/mist/producer_test.go`:

```go
package mist

import (
	"encoding/json"
	"testing"

	mistmsg "atlas-channel/kafka/message/mist"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestCreateCommandProvider_KeySetMatchesAtlasMaps pins the CREATE command's
// JSON shape. atlas-maps owns this contract; the channel mirrors it. A key
// that disagrees produces a mist with silently-zero bounds or lifetime.
func TestCreateCommandProvider_KeySetMatchesAtlasMaps(t *testing.T) {
	body := mistmsg.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: "CHARACTER", OwnerId: 1001,
		TargetKind: mistmsg.TargetKindMonster,
		EffectKind: mistmsg.EffectKindDamageOverTime,
		OriginX:    500, OriginY: 300,
		LtX: -110, LtY: -82, RbX: 110, RbY: 83,
		Disease: "POISON", DiseaseValue: 0, DiseaseDuration: 4000,
		Duration: 4000, TickIntervalMs: 1000,
		SourceSkillId: 2111003, SourceSkillLevel: 1,
	}

	msgs, err := CreateCommandProvider(body)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var envelope struct {
		Type string          `json:"type"`
		Body json.RawMessage `json:"body"`
	}
	require.NoError(t, json.Unmarshal(msgs[0].Value, &envelope))
	require.Equal(t, "CREATE", envelope.Type)

	var got map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(envelope.Body, &got))
	require.ElementsMatch(t, []string{
		"worldId", "channelId", "mapId", "instance",
		"ownerType", "ownerId", "originX", "originY",
		"ltX", "ltY", "rbX", "rbY",
		"disease", "diseaseValue", "diseaseDuration",
		"duration", "tickIntervalMs",
		"sourceSkillId", "sourceSkillLevel",
		"targetKind", "effectKind",
	}, keysOf(got))
}

func keysOf(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./mist/ -v`
Expected: FAIL — package `atlas-channel/mist` does not exist.

- [ ] **Step 3: Add the command side of the channel mist contract**

In `services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go`, add to the `const` block and below:

```go
	EnvCommandTopic = "COMMAND_TOPIC_MIST"

	CommandTypeCreate = "CREATE"

	// TargetKind / EffectKind mirror atlas-maps' descriptors. Empty means
	// CHARACTER / DISEASE there; the channel always sets both explicitly.
	TargetKindCharacter = "CHARACTER"
	TargetKindMonster   = "MONSTER"

	EffectKindDisease        = "DISEASE"
	EffectKindDamageOverTime = "DAMAGE_OVER_TIME"
```

```go
// Command is the envelope for mist commands published to EnvCommandTopic.
// Mirrors atlas-maps' kafka/message/mist Command.
type Command[E any] struct {
	Tenant uuid.UUID `json:"tenant"`
	Type   string    `json:"type"`
	Body   E         `json:"body"`
}

// CreateCommandBody requests creation of a new mist on the named field.
// Mirrors atlas-maps' CreateCommandBody exactly -- atlas-maps owns this
// contract. The Disease* fields are the generic status name / magnitude /
// per-target duration triple; the names are historical.
//
// DiseaseDuration, Duration, and TickIntervalMs are all MILLISECONDS.
type CreateCommandBody struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	OwnerType        string     `json:"ownerType"`
	OwnerId          uint32     `json:"ownerId"`
	OriginX          int16      `json:"originX"`
	OriginY          int16      `json:"originY"`
	LtX              int16      `json:"ltX"`
	LtY              int16      `json:"ltY"`
	RbX              int16      `json:"rbX"`
	RbY              int16      `json:"rbY"`
	Disease          string     `json:"disease"`
	DiseaseValue     int32      `json:"diseaseValue"`
	DiseaseDuration  int64      `json:"diseaseDuration"`
	Duration         int64      `json:"duration"`
	TickIntervalMs   int64      `json:"tickIntervalMs"`
	SourceSkillId    uint32     `json:"sourceSkillId"`
	SourceSkillLevel uint32     `json:"sourceSkillLevel"`
	TargetKind       string     `json:"targetKind"`
	EffectKind       string     `json:"effectKind"`
}
```

Note: atlas-maps' consumer sets `Tenant` from the message headers, not the body — but the field exists on its `Command` envelope, so keep it for shape parity. Check `atlas-maps`' mist command consumer before deciding whether to populate it; mirror what the existing `atlas-monsters` producer (`mistCreateCommandProvider`) does.

- [ ] **Step 4: Create the producer**

Create `services/atlas-channel/atlas.com/channel/mist/producer.go`, modelled on `atlas-channel/monster/producer.go`:

```go
package mist

import (
	mistmsg "atlas-channel/kafka/message/mist"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// CreateCommandProvider builds the CREATE command that asks atlas-maps to
// spawn a mist. Keyed on the map id so every mist command for one map lands on
// the same partition and is processed in cast order.
func CreateCommandProvider(body mistmsg.CreateCommandBody) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(body.MapId))
	value := &mistmsg.Command[mistmsg.CreateCommandBody]{
		Type: mistmsg.CommandTypeCreate,
		Body: body,
	}
	return producer.SingleMessageProvider(key, value)
}
```

Match `mistCreateCommandProvider` in `atlas-monsters/monster/producer.go` (or wherever it lives) for the `Tenant` field and key convention — read it first and follow it rather than inventing a second convention.

- [ ] **Step 5: Create the processor**

Create `services/atlas-channel/atlas.com/channel/mist/processor.go`:

```go
package mist

import (
	mistmsg "atlas-channel/kafka/message/mist"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

// Processor emits mist lifecycle commands to atlas-maps.
type Processor interface {
	Create(body mistmsg.CreateCommandBody) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// Create asks atlas-maps to spawn a mist. atlas-maps is authoritative for the
// mist's identity and lifecycle; this is fire-and-forget.
func (p *ProcessorImpl) Create(body mistmsg.CreateCommandBody) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mistmsg.EnvCommandTopic)(CreateCommandProvider(body))
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./mist/... -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/mist/ services/atlas-channel/atlas.com/channel/kafka/message/mist/
git commit -m "feat(atlas-channel): add COMMAND_TOPIC_MIST create producer"
```

---

### Task 10: atlas-channel — the Poison Mist skill handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`

**Interfaces:**
- Consumes: Task 2's `effect.Model` getters; Task 9's `mist.NewProcessor(...).Create(body)` and `mistmsg.CreateCommandBody`; `channelhandler.Register` (`skill/handler/registry.go:39`) and the `channelhandler.Handler` signature.
- Produces: `poisonmist.Apply` (a `channelhandler.Handler`), consts `PlayerMistTickIntervalMs int64 = 1000`, `MaxPlayerMistDurationMs int32 = 300_000`, and the package-level seams `loadCaster` and `emitCreate`.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist_test.go`:

```go
package poisonmist

import (
	"atlas-channel/data/skill/effect"
	"context"
	"errors"
	"testing"

	mistmsg "atlas-channel/kafka/message/mist"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const (
	testCharId  = uint32(1001)
	testSkillId = uint32(2111003)
	testLevel   = byte(1)
	testX       = int16(500)
	testY       = int16(300)
)

// stubEffect builds an effect.Model carrying the level-1 Poison Mist values
// read from the provisioned WZ corpus (task-200 design §2.1): time 4s (4000ms
// after the reader's conversion), lt (-110,-82), rb (110,83).
//
// effect.Model has unexported fields and no exported constructor; hydrating
// through effect.Extract on a RestModel literal is the supported construction
// path.
func stubEffect(durationMs int32, ltX, ltY, rbX, rbY int16) effect.Model {
	m, err := effect.Extract(effect.RestModel{
		Duration: durationMs,
		LT:       &effect.PointRestModel{X: ltX, Y: ltY},
		RB:       &effect.PointRestModel{X: rbX, Y: rbY},
	})
	if err != nil {
		panic(err)
	}
	return m
}

// harness swaps both package seams and restores them on cleanup, returning a
// pointer to the slice of emitted bodies.
func harness(t *testing.T, casterErr error) *[]mistmsg.CreateCommandBody {
	t.Helper()
	origLoad, origEmit := loadCaster, emitCreate
	t.Cleanup(func() { loadCaster, emitCreate = origLoad, origEmit })

	emitted := make([]mistmsg.CreateCommandBody, 0)
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
		if casterErr != nil {
			return 0, 0, casterErr
		}
		return testX, testY, nil
	}
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}
	return &emitted
}

// testInfo builds the cast packet's SkillUsageInfo. The wire skill id is
// 2111003 on all eleven provisioned versions; the handler forwards it verbatim
// because the client compares it against its own WZ.
func testInfo() packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(testSkillId).
		SetSkillLevel(testLevel).
		Build()
}

func testField() field.Model {
	return field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
}

func run(t *testing.T, e effect.Model) (*[]mistmsg.CreateCommandBody, *test.Hook) {
	t.Helper()
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	emitted := harness(t, nil)
	err := Apply(l)(context.Background())(nil, testField(), testCharId, testInfo(), e)
	require.NoError(t, err)
	return emitted, hook
}

// TestApply_HappyPath_EmitsExactlyOneCreate pins every field of the emitted
// CREATE body against the task-200 design §4.2 table.
func TestApply_HappyPath_EmitsExactlyOneCreate(t *testing.T) {
	emitted, _ := run(t, stubEffect(4000, -110, -82, 110, 83))

	require.Len(t, *emitted, 1)
	b := (*emitted)[0]
	require.Equal(t, "CHARACTER", b.OwnerType)
	require.Equal(t, testCharId, b.OwnerId)
	require.Equal(t, mistmsg.TargetKindMonster, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindDamageOverTime, b.EffectKind)
	require.Equal(t, testX, b.OriginX)
	require.Equal(t, testY, b.OriginY)
	require.Equal(t, int16(-110), b.LtX)
	require.Equal(t, int16(-82), b.LtY)
	require.Equal(t, int16(110), b.RbX)
	require.Equal(t, int16(83), b.RbY)
	require.Equal(t, "POISON", b.Disease)
	require.Equal(t, int32(0), b.DiseaseValue)      // design D1c -- magnitude unread for POISON
	require.Equal(t, int64(4000), b.DiseaseDuration) // design D1a -- per-target = mist lifetime
	require.Equal(t, int64(4000), b.Duration)
	require.Equal(t, PlayerMistTickIntervalMs, b.TickIntervalMs)
	require.Equal(t, testSkillId, b.SourceSkillId) // WIRE id -- the client compares it
	require.Equal(t, uint32(testLevel), b.SourceSkillLevel)
}

// TestApply_ZeroLifetime_Rejected covers FR-6.1.
func TestApply_ZeroLifetime_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(0, -110, -82, 110, 83))
	require.Empty(t, *emitted)
	requireLogged(t, hook, "no lifetime")
}

// TestApply_LifetimeShorterThanOneTick_Rejected covers FR-6.2: a mist that
// expires before its first tick is an invisible no-op.
func TestApply_LifetimeShorterThanOneTick_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(500, -110, -82, 110, 83))
	require.Empty(t, *emitted)
	requireLogged(t, hook, "lifetime shorter than one tick")
}

// TestApply_DegenerateRectangle_Rejected covers FR-6.3.
func TestApply_DegenerateRectangle_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(4000, 110, -82, -110, 83))
	require.Empty(t, *emitted)
	requireLogged(t, hook, "degenerate rectangle")
}

// TestApply_ImplausibleLifetime_Rejected covers FR-6.4. The largest legitimate
// `time` for 2111003 is 40s at level 30, so 5 minutes can only be corrupt data.
func TestApply_ImplausibleLifetime_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(MaxPlayerMistDurationMs+1, -110, -82, 110, 83))
	require.Empty(t, *emitted)
	requireLogged(t, hook, "implausible lifetime")
}

// TestApply_CasterLoadFailure_EmitsNothingAndReturnsNil covers FR-3.3: no
// mist, and no error surfaced to the client.
func TestApply_CasterLoadFailure_EmitsNothingAndReturnsNil(t *testing.T) {
	l, _ := test.NewNullLogger()
	emitted := harness(t, errors.New("character service down"))
	err := Apply(l)(context.Background())(nil, testField(), testCharId, testInfo(), stubEffect(4000, -110, -82, 110, 83))
	require.NoError(t, err)
	require.Empty(t, *emitted)
}

// requireLogged asserts one log entry contains the given rejection reason, so
// each FR-6 gate is distinguishable in production (NFR-4).
func requireLogged(t *testing.T, hook *test.Hook, want string) {
	t.Helper()
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, want) {
			return
		}
	}
	t.Fatalf("no log entry containing %q; got %v", want, hook.AllEntries())
}
```

Add the `strings` and `github.com/google/uuid` imports. `packetmodel.NewSkillUsageInfoBuilder` is at `libs/atlas-packet/model/skill_usage_info.go:85`.

Also add a registration test:

```go
// TestRegistration asserts the handler is reachable through the identity
// registry once `registrations` is imported (task-187 dispatch contract).
func TestRegistration(t *testing.T) {
	_, ok := channelhandler.Lookup(skill2.FirePoisonMagicianPoisonMist)
	require.True(t, ok, "poisonmist handler not registered")
}
```

with imports `channelhandler "atlas-channel/skill/handler"`, `skill2 ".../atlas-constants/skill"`, and a blank import of `atlas-channel/skill/handler/registrations` — the package's own `init()` also registers it, so keep the blank import to prove the registrations file is wired.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/poisonmist/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the handler**

Create `services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist.go`:

```go
// Package poisonmist implements the Fire/Poison Mage Poison Mist (2111003)
// cast: it places a server-side mist at the caster's feet that poisons every
// monster inside its rectangle until it expires.
package poisonmist

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/mist"
	"atlas-channel/socket/writer"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func init() {
	channelhandler.Register(skill2.FirePoisonMagicianPoisonMist, Apply)
}

// PlayerMistTickIntervalMs is the per-tick cadence of a player-cast mist.
//
// It is a constant, not a WZ value: the `dotInterval` node does not exist in
// any provisioned Skill.wz (task-200 design §2.1). 1 Hz is already the
// de-facto DoT cadence on both ends of this contract -- the monster
// AREA_POISON producer hard-codes TickIntervalMs: 1000, and atlas-monsters'
// APPLY_STATUS consumer independently defaults a POISON/VENOM tick to 1000ms
// when the command omits one. This makes it explicit rather than relying on
// the consumer's fallback.
//
// Known tuning point: atlas-monsters replaces a same-type status on re-apply,
// minting a fresh lastTick, so a 1000ms re-apply against a 1s DoT cadence can
// under-count ticks. If observed damage is starved, raise this above the DoT
// cadence -- a one-constant change (design §4.4).
const PlayerMistTickIntervalMs int64 = 1000

// MaxPlayerMistDurationMs rejects (never truncates) an implausible mist
// lifetime. The largest legitimate `time` for 2111003 across the provisioned
// corpus is 40s at level 30, so this 5-minute ceiling is 7.5x the largest real
// value and can only fire on corrupt or mis-scaled data.
//
// This is deliberately NOT atlas-monsters' 60s MistDurationCapMs. A clamp
// would desynchronise the client, which computes its own
// tEnd = tStart + 1000*SKILLLEVELDATA::tTime from its own WZ (v83 @0x43200f,
// v95 @0x437c95) and would keep rendering a mist the server stopped ticking.
const MaxPlayerMistDurationMs int32 = 300_000

// loadCaster returns the caster's (X, Y) position from the character service.
// Package-level var so tests can stub it.
var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, 0, err
	}
	return c.X(), c.Y(), nil
}

// emitCreate publishes the CREATE command to atlas-maps. Package-level var so
// tests can record instead of producing.
var emitCreate = func(l logrus.FieldLogger, ctx context.Context, body mistmsg.CreateCommandBody) error {
	return mist.NewProcessor(l, ctx).Create(body)
}

// Apply is the Poison Mist handler installed in the per-skill registry.
//
// By the time it runs, UseSkill has already charged MP and applied the
// cooldown. Every rejection below returns nil and emits nothing: there is no
// MP or cooldown rollback path, by design (FR-3.2 / FR-6.5).
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	info packetmodel.SkillUsageInfo,
	e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model,
		characterId uint32,
		info packetmodel.SkillUsageInfo,
		e effect.Model,
	) error {
		return func(
			wp writer.Producer,
			f field.Model,
			characterId uint32,
			info packetmodel.SkillUsageInfo,
			e effect.Model,
		) error {
			duration := e.Duration()
			lt, rb := e.LT(), e.RB()

			if duration <= 0 {
				l.Warnf("Poison Mist: rejected cast by [%d] — no lifetime (effect duration %d ms).", characterId, duration)
				return nil
			}
			if int64(duration) < PlayerMistTickIntervalMs {
				l.Warnf("Poison Mist: rejected cast by [%d] — lifetime shorter than one tick (%d ms < %d ms).", characterId, duration, PlayerMistTickIntervalMs)
				return nil
			}
			if rb.X() <= lt.X() || rb.Y() <= lt.Y() {
				l.Warnf("Poison Mist: rejected cast by [%d] — degenerate rectangle lt(%d,%d) rb(%d,%d).", characterId, lt.X(), lt.Y(), rb.X(), rb.Y())
				return nil
			}
			if duration > MaxPlayerMistDurationMs {
				l.Warnf("Poison Mist: rejected cast by [%d] — implausible lifetime (%d ms > %d ms ceiling).", characterId, duration, MaxPlayerMistDurationMs)
				return nil
			}

			x, y, err := loadCaster(l, ctx, characterId)
			if err != nil {
				l.WithError(err).Errorf("Poison Mist: failed to load caster [%d]; no mist created.", characterId)
				return nil
			}

			body := mistmsg.CreateCommandBody{
				WorldId:   f.WorldId(),
				ChannelId: f.ChannelId(),
				MapId:     f.MapId(),
				Instance:  f.Instance(),
				OwnerType: "CHARACTER",
				OwnerId:   characterId,
				// A player-cast mist targets MONSTERS with a damage-bearing
				// status, unlike the monster AREA_POISON mist which diseases
				// CHARACTERS.
				TargetKind: mistmsg.TargetKindMonster,
				EffectKind: mistmsg.EffectKindDamageOverTime,
				OriginX:    x,
				OriginY:    y,
				LtX:        lt.X(),
				LtY:        lt.Y(),
				RbX:        rb.X(),
				RbY:        rb.Y(),
				Disease:    "POISON",
				// Magnitude 0 is correct, not a shortcut: atlas-monsters
				// computes poison damage as maxHP/(70 - sourceSkillLevel) and
				// never reads the POISON magnitude (VENOM is the status that
				// does). A non-zero value here would be dead payload.
				DiseaseValue: 0,
				// Per-target duration = the mist's lifetime. With no WZ
				// `dotTime`, this is the value that matches the skill's
				// observable behavior, and atlas-monsters REPLACES a same-type
				// status on re-apply, so a monster inside the cloud simply has
				// its expiry pushed forward each tick (design D1a / §4.4).
				DiseaseDuration: int64(duration),
				Duration:        int64(duration),
				TickIntervalMs:  PlayerMistTickIntervalMs,
				// The WIRE skill id, deliberately -- not the resolved
				// Identity. The client compares this against its own WZ to
				// pick the rendering arm (v83 @0x431d50, v95 @0x437515), so it
				// must be the id that version binds. This is the one place a
				// raw wire id is the correct value.
				SourceSkillId:    uint32(info.SkillId()),
				SourceSkillLevel: uint32(info.SkillLevel()),
			}

			if err := emitCreate(l, ctx, body); err != nil {
				l.WithError(err).Errorf("Poison Mist: failed to emit CREATE for character [%d].", characterId)
				return nil
			}

			l.Infof("Poison Mist: character [%d] cast level [%d] at (%d,%d), rect lt(%d,%d) rb(%d,%d), lifetime %d ms.",
				characterId, info.SkillLevel(), x, y, lt.X(), lt.Y(), rb.X(), rb.Y(), duration)
			return nil
		}
	}
}
```

- [ ] **Step 4: Register the subpackage**

In `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`, add the blank import in alphabetical position (after `mysticdoor`):

```go
	_ "atlas-channel/skill/handler/poisonmist"   // Fire/Poison Mage Poison Mist — task-200
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/... -v 2>&1 | tail -40`
Expected: PASS, including `TestRegistration` and all six `TestApply_*` tests.

- [ ] **Step 6: Run the skill-job-id guard and the full channel module**

Run:
```
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./...
cd ../../../.. && tools/skill-job-id-guard.sh
```
Expected: all clean. The guard must not flag the `uint32(info.SkillId())` use — it is not a comparison. If it does, read the guard's rule before adjusting anything.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/
git commit -m "feat(atlas-channel): add Poison Mist (2111003) skill handler"
```

---

### Task 11: Promote `SPAWN_MIST` × v92 and `REMOVE_MIST` × v92 to ✅

**Files:**
- Create/Modify: byte fixtures under `libs/atlas-packet/field/clientbound/` (`affected_area_created_test.go`, `affected_area_removed_test.go`) with `packet-audit:verify` markers.
- Create: evidence records under `docs/packets/audits/gms_v92/`.
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` (regenerated, never hand-edited).

**Interfaces:**
- Consumes: nothing from prior tasks — this is independent and may run in parallel with Tasks 1–10.
- Produces: two ✅ matrix cells.

**Do not restate the procedure here.** Follow [`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md) — the canonical single-cell playbook — via the `/verify-packet` command or the `packet-verifier` agent, once per cell.

Current state (`docs/packets/audits/STATUS.md:337,340`):
- `SPAWN_MIST × gms_v92` = ❌, opcode `0x140`
- `REMOVE_MIST × gms_v92` = 🟡ᶠ, opcode `0x141`

Expectation per design §6: a **verification pass, not a codec change** — the encoder already models v92 explicitly (`affected_area_created.go:141`, `hasPhase = IsRegion("GMS") && MajorAtLeast(92)`), and v92's `OnAffectedAreaCreated` `@0x4392a0` is documented as identical to v95's. If the v92 read order does diverge, the codec change **is in scope for this task** — do not defer it.

- [ ] **Step 1: Verify `SPAWN_MIST` × gms_v92**

Run `/verify-packet field/clientbound/FieldAffectedAreaCreated gms_v92` (or dispatch `packet-verifier` for that cell). Decompile the v92 read order, write the byte fixture with a `packet-audit:verify packet=... version=gms_v92 ida=0x...` marker, pin the evidence record, regenerate the matrix.

Expected: `STATUS.md:337`'s v92 column shows ✅.

- [ ] **Step 2: Verify `REMOVE_MIST` × gms_v92**

Same procedure for `field/clientbound/FieldAffectedAreaRemoved` × `gms_v92`.

Expected: `STATUS.md:340`'s v92 column shows ✅.

- [ ] **Step 3: Confirm no previously-✅ cell degraded**

Run: `git diff docs/packets/audits/STATUS.md`
Expected: the only status changes are the two v92 cells, ❌→✅ and 🟡ᶠ→✅. Any other cell changing is a regression — stop and investigate.

- [ ] **Step 4: Run the whole packet suite**

Run: `cd libs/atlas-packet && go test ./... 2>&1 | tail -20`
Expected: PASS — including every pre-existing mist fixture, unmodified (FR-5.5).

- [ ] **Step 5: Commit**

The playbook requires the fixture, evidence record, and regenerated matrix to be committed **together**:

```bash
git add libs/atlas-packet/field/clientbound/ docs/packets/audits/
git commit -m "verify(packets): promote SPAWN_MIST and REMOVE_MIST on gms_v92 to verified"
```

---

### Task 12: Full-branch verification gates

**Files:** none created; this task only runs checks and fixes what they surface.

**Interfaces:**
- Consumes: everything from Tasks 1–11.
- Produces: a branch that satisfies CLAUDE.md §Build & Verification.

- [ ] **Step 1: Merge main and re-verify**

The packet matrix's `toolSha` reads git HEAD, so the matrix must be regenerated **after** the merge, not before.

```bash
git fetch origin main
git merge origin/main
```

If the merge touches any Go file, run `tools/lint.sh` (fix mode) before continuing — a pre-task-171 merge base commonly leaves formatting drift.

- [ ] **Step 2: Per-module build, vet, and race tests**

```bash
for m in atlas-data/atlas.com/data atlas-channel/atlas.com/channel atlas-maps/atlas.com/maps atlas-monsters/atlas.com/monsters; do
  echo "=== $m ==="
  ( cd "services/$m" && go build ./... && go vet ./... && go test -race ./... ) || echo "FAILED: $m"
done
( cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./... ) || echo "FAILED: atlas-packet"
```

Expected: no `FAILED:` line, no test failures.

- [ ] **Step 3: Repo-root guards**

```bash
tools/lint.sh --check
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/buff-duration-guard.sh
tools/skill-job-id-guard.sh
```

Expected: each exits 0. `buff-duration-guard.sh` matters here specifically: the character mist branch's `duration` field must still be milliseconds, and the new `applyStatusBody` is on a different topic and must not trip it. If it does trip, do not add an escape hatch — read the guard's rule and fix the value.

- [ ] **Step 4: Packet-audit gates**

```bash
packet-audit matrix --check
packet-audit fname-doc --check
packet-audit operations --check
```

Expected: all three exit 0. Regenerate the matrix **after** the Step 1 merge if it is stale.

- [ ] **Step 5: Confirm no `go.mod` changed**

```bash
git diff --name-only origin/main...HEAD -- '*/go.mod' 'go.work'
```

Expected: empty. If any `go.mod` or `go.work` changed, `docker buildx bake atlas-<svc>` becomes **mandatory** for every affected service (CLAUDE.md §Build & Verification item 4) — run it before proceeding.

- [ ] **Step 6: Confirm the template claim rather than re-adding**

`AffectedAreaCreated` / `AffectedAreaRemoved` are already registered in all eleven seed templates as of `ae3341511` (#1226, task-165). Verify, do not re-add:

```bash
grep -c 'AffectedAreaCreated' services/atlas-configurations/seed-data/templates/*.json
grep -c 'AffectedAreaRemoved' services/atlas-configurations/seed-data/templates/*.json
```

Expected: every template reports at least 1 for each. If any reports 0, that is a real gap and wiring it is in scope — add the writer at its sorted `opCode` position and re-run `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh`.

- [ ] **Step 7: Commit any fixes**

```bash
git add -A
git commit -m "chore(task-200): verification gate fixes"
```

(Skip if nothing changed.)

- [ ] **Step 8: Code review before PR**

Run `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (Go changed). Because this branch changes packet-adjacent artifacts, also dispatch `packet-completeness-critic`. Pin review subagents to a cheaper model per [[feedback_review_workflows_use_cheaper_model]]. Do not open the PR before this step.

---

## Manual end-to-end acceptance (post-merge, on a live tenant)

Not automatable in this plan; record the outcome in the PR description.

- [ ] An FP Mage casting Poison Mist sees the cloud appear at their feet.
- [ ] A second character in the same map sees the same cloud.
- [ ] The cloud disappears at the WZ-specified time (4 s at level 1, up to 40 s at level 30).
- [ ] Monsters standing inside the cloud lose HP periodically for the mist's duration.
- [ ] A monster AREA_POISON mist still spawns, still poisons characters, and still renders on the same tenant (NFR-5).

## Self-review notes

- **`prop` (41–70% apply chance) is deliberately not implemented** — design §2.4 records this as a decision, not an oversight. No task covers it, by design.
- **FR-1.5's non-zero-data acceptance criterion is not met and cannot be** — the WZ nodes do not exist on any provisioned version. design §2.2 is the data-defect report FR-1.5 asks for. Task 1 implements the parse anyway (additive, forward-compatible).
- **OQ-4 is resolved, not deferred** — poison damage is `maxHP/(70-level)`, computed in atlas-monsters. No task adds a magnitude.
