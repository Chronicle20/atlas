# Priest Dispel — Party Debuff Cure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Casting Priest Dispel (2311001) cures the six dispellable debuffs (CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW) on the caster and bitmap-selected in-map party members, honoring the skill's per-recipient prop roll.

**Architecture:** All changes in `atlas-channel`. A new `CANCEL_BY_TYPES` producer on the existing `character/buff` processor emits to `COMMAND_TOPIC_CHARACTER_BUFF` (atlas-buffs already consumes it). A new `skill/handler/dispel` subpackage registers in the per-skill dispatcher (heal/mysticdoor pattern) and wires recipient selection → prop roll → per-recipient emit. Zero changes to `atlas-buffs` and zero changes to the existing mob-side Dispel path in `common.go`.

**Tech Stack:** Go, Kafka (segmentio/kafka-go via atlas-kafka producer), logrus, atlas-constants.

## Global Constraints

- Working directory for all Go commands: `services/atlas-channel/atlas.com/channel` (module `atlas-channel`) inside the task worktree `.worktrees/task-163-priest-dispel-party/`.
- The debuff set is EXACTLY: `CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW` (typed constants from `github.com/Chronicle20/atlas/libs/atlas-constants/character`). ZOMBIFY / SEDUCE / CONFUSE are intentionally excluded.
- `services/atlas-buffs/` must have ZERO diffs at the end (design §3.6, acceptance-gated).
- `skill/handler/common.go` must be untouched — the mob half of Dispel already works there.
- No `*_testhelpers.go` files; use the project's Builder pattern for test setup.
- Seams are package-level func vars overridden in tests with `t.Cleanup` restore (mysticdoor precedent).
- The handler never returns a non-nil error — all failures are logged and the cast continues.
- Commit messages use the `<type>(task-163): <summary>` convention on branch `task-163-priest-dispel-party`.

---

### Task 1: CANCEL_BY_TYPES message types, producer provider, and processor method

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/character/buff/producer_test.go` (new)

**Interfaces:**
- Consumes: existing `buff.Command[E]` envelope, `buff2.EnvCommandTopic` (`COMMAND_TOPIC_CHARACTER_BUFF`), `producer.CreateKey`, `producer.SingleMessageProvider`, `producer.ProviderImpl`.
- Produces (Task 2 relies on these exact signatures):
  - `buff.CancelByTypesCommandProvider(f field.Model, characterId uint32, types []string) model.Provider[[]kafka.Message]` (package `atlas-channel/character/buff`)
  - `Processor` interface method `CancelByTypes(f field.Model, types []charcon.TemporaryStatType) model.Operator[uint32]` where `charcon` = `github.com/Chronicle20/atlas/libs/atlas-constants/character` — field + types bound first, returns the per-character emitter.
  - Message constants `buff.CommandTypeCancelByTypes = "CANCEL_BY_TYPES"` and `buff.CancelByTypesCommandBody{ Types []string }` (package `atlas-channel/kafka/message/buff`).

- [ ] **Step 1: Write the failing producer test**

Create `services/atlas-channel/atlas.com/channel/character/buff/producer_test.go`. The wire shape must match what atlas-buffs consumes (`services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go:56-58` — `CancelByTypesCommandBody{ Types []string \`json:"types"\` }`, type string `CANCEL_BY_TYPES`). Cross-module import of atlas-buffs types is impossible across service modules, so the contract is asserted with literal JSON field names:

```go
package buff

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

// TestCancelByTypesCommandProvider_WireContract pins the JSON wire shape
// consumed by atlas-buffs' CANCEL_BY_TYPES handler (field names asserted
// literally — the consumer lives in a different Go module).
func TestCancelByTypesCommandProvider_WireContract(t *testing.T) {
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	types := []string{"CURSE", "DARKNESS", "POISON", "SEAL", "WEAKEN", "SLOW"}

	msgs, err := CancelByTypesCommandProvider(f, 1001, types)()
	if err != nil {
		t.Fatalf("CancelByTypesCommandProvider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if want := producer.CreateKey(1001); !bytes.Equal(msgs[0].Key, want) {
		t.Fatalf("message key = %v, want CreateKey(characterId) = %v", msgs[0].Key, want)
	}

	var decoded struct {
		WorldId     byte   `json:"worldId"`
		ChannelId   byte   `json:"channelId"`
		MapId       uint32 `json:"mapId"`
		CharacterId uint32 `json:"characterId"`
		Type        string `json:"type"`
		Body        struct {
			Types []string `json:"types"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msgs[0].Value, &decoded); err != nil {
		t.Fatalf("failed to unmarshal message value: %v", err)
	}
	if decoded.Type != "CANCEL_BY_TYPES" {
		t.Fatalf("type = %q, want %q", decoded.Type, "CANCEL_BY_TYPES")
	}
	if decoded.WorldId != 1 || decoded.ChannelId != 2 || decoded.MapId != 100000000 {
		t.Fatalf("envelope = world %d channel %d map %d, want 1/2/100000000",
			decoded.WorldId, decoded.ChannelId, decoded.MapId)
	}
	if decoded.CharacterId != 1001 {
		t.Fatalf("characterId = %d, want 1001", decoded.CharacterId)
	}
	if !reflect.DeepEqual(decoded.Body.Types, types) {
		t.Fatalf("body.types = %v, want %v", decoded.Body.Types, types)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test ./character/buff/ -run TestCancelByTypesCommandProvider -v
```
Expected: FAIL to compile with `undefined: CancelByTypesCommandProvider`.

- [ ] **Step 3: Add the message constant and body type**

In `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go`, extend the existing const block (lines 12-16) and add the body type after `CancelCommandBody` (line 43), mirroring atlas-buffs:

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

- [ ] **Step 4: Add the producer provider**

Append to `services/atlas-channel/atlas.com/channel/character/buff/producer.go`, identical in structure to `CancelCommandProvider`:

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
			Types: types,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Add the processor method**

In `services/atlas-channel/atlas.com/channel/character/buff/processor.go`:

Add to the `Processor` interface (after `Cancel`, line 20):

```go
	CancelByTypes(f field.Model, types []charcon.TemporaryStatType) model.Operator[uint32]
```

Add the import `charcon "github.com/Chronicle20/atlas/libs/atlas-constants/character"` to the import block, then append the implementation. Curried per design §3.3: field + types bound first (typed→wire `[]string` conversion happens once), returning the per-character emitter:

```go
func (p *ProcessorImpl) CancelByTypes(f field.Model, types []charcon.TemporaryStatType) model.Operator[uint32] {
	strTypes := make([]string, 0, len(types))
	for _, st := range types {
		strTypes = append(strTypes, string(st))
	}
	return func(characterId uint32) error {
		p.l.Debugf("Character [%d] cancelling buffs by stat types %v.", characterId, strTypes)
		return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(CancelByTypesCommandProvider(f, characterId, strTypes))
	}
}
```

Note: `character/buff` has no mock (verified — only `model.go, processor.go, producer.go, requests.go, rest.go, stat/`); nothing else to update for the interface change.

- [ ] **Step 6: Run test to verify it passes**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test ./character/buff/... -v && go build ./... && go vet ./character/buff/... ./kafka/message/buff/...
```
Expected: `TestCancelByTypesCommandProvider_WireContract` PASS; build and vet clean.

- [ ] **Step 7: Commit**

```bash
git add kafka/message/buff/kafka.go character/buff/producer.go character/buff/processor.go character/buff/producer_test.go
git commit -m "feat(task-163): CANCEL_BY_TYPES buff command producer and processor method"
```

---

### Task 2: Dispel handler subpackage with tests

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/dispel/dispel.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/dispel/dispel_test.go` (new, internal package `dispel` — it overrides unexported seams)

**Interfaces:**
- Consumes:
  - `channelhandler.Register(id skill2.Id, h channelhandler.Handler)` and `channelhandler.Lookup(id skill2.Id) (Handler, bool)` from `atlas-channel/skill/handler` (`registry.go`).
  - `channelhandler.SelectPartyMembersInMap(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, memberBitmap byte) []channelhandler.PartyRecipient` (`recipients.go:116`) — already filters offline / other-map / no-session / dead members and applies the MSB-first bitmap decode.
  - `channelhandler.PartyRecipient.Id() uint32` and `channelhandler.NewPartyRecipientBuilder()` (tests).
  - `buff.NewProcessor(l, ctx).CancelByTypes(f, types)` from Task 1.
  - `e.Prop() float64` — already normalized 0.0–1.0 (`data/skill/effect/model.go:138`); no /100 scaling.
  - `skill2.PriestDispelId = Id(2311001)` (`libs/atlas-constants/skill/constants.go:3068`).
  - `charcon.TemporaryStatTypeCurse/Darkness/Poison/Seal/Weaken/Slow` (`libs/atlas-constants/character/temporary_stat.go`).
- Produces: `dispel.Apply` matching `channelhandler.Handler`, registered for `skill2.PriestDispelId` in `init()` (Task 3 adds the blank import that triggers it in production).

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/dispel/dispel_test.go`:

```go
package dispel

import (
	"context"
	"errors"
	"testing"

	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"

	charcon "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

const (
	testCasterId = uint32(1001)
	testBitmap   = byte(0x30) // party slots 0 and 1 -> bits 5 and 4 (MSB-first)
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	return l
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
}

func testInfo() packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill2.PriestDispelId)).
		SetSkillLevel(1).
		SetAffectedPartyMemberBitmap(testBitmap).
		Build()
}

// fullPropEffect returns an effect with Prop 1.0 (always passes the roll).
func fullPropEffect(t *testing.T) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{Prop: 1.0})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return e
}

func members(ids ...uint32) []channelhandler.PartyRecipient {
	out := make([]channelhandler.PartyRecipient, 0, len(ids))
	for _, id := range ids {
		out = append(out, channelhandler.NewPartyRecipientBuilder().SetId(id).Build())
	}
	return out
}

// harness installs the three seams with t.Cleanup restore and records calls.
type harness struct {
	gotField    field.Model
	gotCasterId uint32
	gotBitmap   byte
	gotTypes    []charcon.TemporaryStatType
	emittedIds  []uint32
}

func install(
	t *testing.T,
	selected []channelhandler.PartyRecipient,
	propRoll func(float64) bool,
	emitErr func(recipientId uint32) error,
) *harness {
	t.Helper()
	h := &harness{}
	origSelect, origProp, origCancel := selectPartyMembersFunc, propRollFunc, cancelByTypesFunc
	t.Cleanup(func() {
		selectPartyMembersFunc, propRollFunc, cancelByTypesFunc = origSelect, origProp, origCancel
	})

	selectPartyMembersFunc = func(_ logrus.FieldLogger, _ context.Context, f field.Model, casterId uint32, bitmap byte) []channelhandler.PartyRecipient {
		h.gotField = f
		h.gotCasterId = casterId
		h.gotBitmap = bitmap
		return selected
	}
	propRollFunc = propRoll
	cancelByTypesFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, types []charcon.TemporaryStatType) model.Operator[uint32] {
		h.gotTypes = types
		return func(recipientId uint32) error {
			if err := emitErr(recipientId); err != nil {
				return err
			}
			h.emittedIds = append(h.emittedIds, recipientId)
			return nil
		}
	}
	return h
}

func alwaysPass(float64) bool { return true }
func noEmitErr(uint32) error  { return nil }

func runApply(t *testing.T, e effect.Model) error {
	t.Helper()
	return Apply(testLogger())(context.Background())(nil, testField(), testCasterId, testInfo(), e)
}

// TestDispelRegistered: the package init() installs Apply for PriestDispelId.
func TestDispelRegistered(t *testing.T) {
	h, ok := channelhandler.Lookup(skill2.PriestDispelId)
	if !ok {
		t.Fatal("Lookup(PriestDispelId) returned ok=false; init() registration missing")
	}
	if h == nil {
		t.Fatal("Lookup(PriestDispelId) returned nil handler")
	}
}

// TestDispelCuresCasterAndMembersInOrder: all prop-pass -> one emit per
// recipient, caster first then members in selector order, six exact types.
func TestDispelCuresCasterAndMembersInOrder(t *testing.T) {
	h := install(t, members(2002, 3003), alwaysPass, noEmitErr)

	if err := runApply(t, fullPropEffect(t)); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	wantIds := []uint32{testCasterId, 2002, 3003}
	if len(h.emittedIds) != len(wantIds) {
		t.Fatalf("emitted %v, want %v", h.emittedIds, wantIds)
	}
	for i := range wantIds {
		if h.emittedIds[i] != wantIds[i] {
			t.Fatalf("emitted %v, want %v (caster-first order)", h.emittedIds, wantIds)
		}
	}

	wantTypes := []charcon.TemporaryStatType{
		charcon.TemporaryStatTypeCurse,
		charcon.TemporaryStatTypeDarkness,
		charcon.TemporaryStatTypePoison,
		charcon.TemporaryStatTypeSeal,
		charcon.TemporaryStatTypeWeaken,
		charcon.TemporaryStatTypeSlow,
	}
	if len(h.gotTypes) != len(wantTypes) {
		t.Fatalf("types = %v, want %v", h.gotTypes, wantTypes)
	}
	for i := range wantTypes {
		if h.gotTypes[i] != wantTypes[i] {
			t.Fatalf("types = %v, want %v", h.gotTypes, wantTypes)
		}
	}
}

// TestDispelSelectorReceivesCastArgs: the selector sees the exact
// (f, casterId, bitmap) from the cast packet.
func TestDispelSelectorReceivesCastArgs(t *testing.T) {
	h := install(t, nil, alwaysPass, noEmitErr)

	if err := runApply(t, fullPropEffect(t)); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	if h.gotCasterId != testCasterId {
		t.Fatalf("selector casterId = %d, want %d", h.gotCasterId, testCasterId)
	}
	if h.gotBitmap != testBitmap {
		t.Fatalf("selector bitmap = 0x%02x, want 0x%02x", h.gotBitmap, testBitmap)
	}
	if h.gotField.MapId() != testField().MapId() {
		t.Fatalf("selector field mapId = %d, want %d", h.gotField.MapId(), testField().MapId())
	}
}

// TestDispelEmptySelectorCastsCasterOnly: no members selected -> caster-only cure.
func TestDispelEmptySelectorCastsCasterOnly(t *testing.T) {
	h := install(t, nil, alwaysPass, noEmitErr)

	if err := runApply(t, fullPropEffect(t)); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(h.emittedIds) != 1 || h.emittedIds[0] != testCasterId {
		t.Fatalf("emitted %v, want caster-only [%d]", h.emittedIds, testCasterId)
	}
}

// TestDispelPropRollPerRecipient: alternating pass/fail -> only passing
// recipients are emitted; the cast never errors.
func TestDispelPropRollPerRecipient(t *testing.T) {
	rolls := 0
	alternating := func(float64) bool {
		rolls++
		return rolls%2 == 1 // pass, fail, pass
	}
	h := install(t, members(2002, 3003), alternating, noEmitErr)

	if err := runApply(t, fullPropEffect(t)); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	// recipients [caster, 2002, 3003]; rolls pass/fail/pass -> caster and 3003 cured.
	wantIds := []uint32{testCasterId, 3003}
	if len(h.emittedIds) != len(wantIds) || h.emittedIds[0] != wantIds[0] || h.emittedIds[1] != wantIds[1] {
		t.Fatalf("emitted %v, want %v", h.emittedIds, wantIds)
	}
	if rolls != 3 {
		t.Fatalf("prop rolled %d times, want 3 (once per recipient)", rolls)
	}
}

// TestDispelEmitErrorContinues: an emit failure for one recipient does not
// abort the remaining recipients (FR-7).
func TestDispelEmitErrorContinues(t *testing.T) {
	failFor := uint32(2002)
	h := install(t, members(2002, 3003), alwaysPass, func(id uint32) error {
		if id == failFor {
			return errors.New("kafka down")
		}
		return nil
	})

	if err := runApply(t, fullPropEffect(t)); err != nil {
		t.Fatalf("Apply returned error: %v (emit failures must not fail the cast)", err)
	}
	wantIds := []uint32{testCasterId, 3003}
	if len(h.emittedIds) != len(wantIds) || h.emittedIds[0] != wantIds[0] || h.emittedIds[1] != wantIds[1] {
		t.Fatalf("emitted %v, want %v (recipient after the failure still cured)", h.emittedIds, wantIds)
	}
}

// TestDispelZeroPropCuresNobody: the real propRollFunc with a zero-prop
// effect (effect.Model zero value) cures no one.
func TestDispelZeroPropCuresNobody(t *testing.T) {
	h := install(t, members(2002), propRollFunc, noEmitErr) // real roll, prop=0

	if err := runApply(t, effect.Model{}); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(h.emittedIds) != 0 {
		t.Fatalf("emitted %v, want none for prop=0", h.emittedIds)
	}
}

// TestDispelSummaryLogFields: the per-cast summary Debug line carries the
// FR-8 structured fields with correct counts (3 recipients, pass/fail/pass
// rolls -> 2 cures, 1 prop-skipped).
func TestDispelSummaryLogFields(t *testing.T) {
	rolls := 0
	alternating := func(float64) bool {
		rolls++
		return rolls%2 == 1
	}
	install(t, members(2002, 3003), alternating, noEmitErr)

	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	if err := Apply(logger)(context.Background())(nil, testField(), testCasterId, testInfo(), fullPropEffect(t)); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	var summary *logrus.Entry
	for _, entry := range hook.AllEntries() {
		if entry.Message == "dispel_party_cure_summary" {
			summary = entry
			break
		}
	}
	if summary == nil {
		t.Fatal("no dispel_party_cure_summary log entry emitted")
	}
	if got := summary.Data["caster"]; got != testCasterId {
		t.Fatalf("summary caster = %v, want %d", got, testCasterId)
	}
	if got := summary.Data["bitmap"]; got != testBitmap {
		t.Fatalf("summary bitmap = %v, want 0x%02x", got, testBitmap)
	}
	if got := summary.Data["recipients_selected"]; got != 3 {
		t.Fatalf("summary recipients_selected = %v, want 3", got)
	}
	if got := summary.Data["cures_emitted"]; got != 2 {
		t.Fatalf("summary cures_emitted = %v, want 2", got)
	}
	if got := summary.Data["prop_skipped"]; got != 1 {
		t.Fatalf("summary prop_skipped = %v, want 1", got)
	}
}

// TestPropRollBoundaries: the mirrored roll matches common.go semantics —
// prop <= 0 never passes, prop >= 1 always passes (no RNG).
func TestPropRollBoundaries(t *testing.T) {
	if propRollFunc(0) {
		t.Fatal("propRollFunc(0) = true, want false")
	}
	if propRollFunc(-0.5) {
		t.Fatal("propRollFunc(-0.5) = true, want false")
	}
	if !propRollFunc(1.0) {
		t.Fatal("propRollFunc(1.0) = false, want true")
	}
	if !propRollFunc(1.5) {
		t.Fatal("propRollFunc(1.5) = false, want true")
	}
}
```

Note the `install(t, members(2002), propRollFunc, noEmitErr)` call in `TestDispelZeroPropCuresNobody` captures the REAL `propRollFunc` as the roll (evaluated before `install` swaps the var), exercising the production zero-prop guard.

- [ ] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test ./skill/handler/dispel/ -v
```
Expected: FAIL to compile — the package `dispel` does not exist yet (`no Go files` / undefined `Apply`, `selectPartyMembersFunc`, etc.).

- [ ] **Step 3: Implement the handler**

Create `services/atlas-channel/atlas.com/channel/skill/handler/dispel/dispel.go`:

```go
package dispel

import (
	"context"
	"math/rand"

	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/socket/writer"

	charcon "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

func init() {
	channelhandler.Register(skill2.PriestDispelId, Apply)
}

// dispellableStatTypes is the exact Dispel cure set (FR-5, Cosmic
// Character.dispelDebuffs parity). ZOMBIFY / SEDUCE / CONFUSE are
// intentionally excluded — those are purgeDebuffs (cure-all) semantics,
// owned by task-156's SuperGM Heal+Dispel.
var dispellableStatTypes = []charcon.TemporaryStatType{
	charcon.TemporaryStatTypeCurse,
	charcon.TemporaryStatTypeDarkness,
	charcon.TemporaryStatTypePoison,
	charcon.TemporaryStatTypeSeal,
	charcon.TemporaryStatTypeWeaken,
	charcon.TemporaryStatTypeSlow,
}

// selectPartyMembersFunc is the recipient-selection seam tests can replace.
// Production uses the map-wide bitmap selector — the party cure has no
// LT/RB rectangle (the WZ rect governs the mob half, enforced in
// applyToMobs).
var selectPartyMembersFunc = channelhandler.SelectPartyMembersInMap

// propRollFunc gates each recipient's cure by the skill's prop value.
// Mirrors the unexported seam in skill/handler/common.go exactly (the
// parent seam cannot be shared across the package boundary — heal
// precedent). e.Prop() is already normalized to 0.0–1.0.
var propRollFunc = func(prop float64) bool {
	if prop <= 0 {
		return false
	}
	if prop >= 1 {
		return true
	}
	return rand.Float64() <= prop
}

// cancelByTypesFunc is the emit seam tests can replace. Production binds
// field + stat types once per cast and returns the per-recipient emitter.
var cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, types []charcon.TemporaryStatType) model.Operator[uint32] {
	return buff.NewProcessor(l, ctx).CancelByTypes(f, types)
}

// Apply is the Priest Dispel party-cure handler installed in the per-skill
// registry. The mob half (cancel mob buffs, magic-reflect-aware) already
// runs in UseSkill's applyToMobs before this dispatch; the two halves are
// independent.
//
// Lifecycle:
//  1. Resolve recipients: the caster, plus map-wide bitmap-selected party
//     members (same channel + map, live session, alive).
//  2. Bind the CANCEL_BY_TYPES emitter once (field + the six dispellable
//     stat types).
//  3. Per recipient: roll the skill's prop; on pass, emit the cure. A
//     failed roll skips only that recipient; an emit error is logged and
//     the remaining recipients still process.
//  4. Log a per-cast structured summary.
//
// The handler always returns nil — no failure aborts the cast.
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
			bitmap := info.AffectedPartyMemberBitmap()
			members := selectPartyMembersFunc(l, ctx, f, characterId, bitmap)

			recipients := make([]uint32, 0, len(members)+1)
			recipients = append(recipients, characterId)
			for _, m := range members {
				recipients = append(recipients, m.Id())
			}

			op := cancelByTypesFunc(l, ctx, f, dispellableStatTypes)

			curesEmitted, propSkipped := 0, 0
			for _, recipientId := range recipients {
				if !propRollFunc(e.Prop()) {
					propSkipped++
					continue
				}
				if err := op(recipientId); err != nil {
					l.WithError(err).Errorf("Dispel: CANCEL_BY_TYPES emit failed for recipient [%d] from caster [%d].", recipientId, characterId)
					continue
				}
				curesEmitted++
			}

			l.WithFields(buildSummaryFields(characterId, skill2.Id(info.SkillId()), uint32(info.SkillLevel()), bitmap, len(recipients), curesEmitted, propSkipped)).Debug("dispel_party_cure_summary")
			return nil
		}
	}
}

// buildSummaryFields packs the FR-8 per-cast summary fields (the
// common.go buildSummaryFields precedent, local to this package).
func buildSummaryFields(characterId uint32, sid skill2.Id, slvl uint32, bitmap byte, recipientsSelected, curesEmitted, propSkipped int) logrus.Fields {
	return logrus.Fields{
		"caster":              characterId,
		"skill_id":            uint32(sid),
		"skill_level":         slvl,
		"bitmap":              bitmap,
		"recipients_selected": recipientsSelected,
		"cures_emitted":       curesEmitted,
		"prop_skipped":        propSkipped,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test -race ./skill/handler/dispel/ -v && go vet ./skill/handler/dispel/
```
Expected: all 9 tests PASS, vet clean.

- [ ] **Step 5: Verify the mob-side path is untouched**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test -race ./skill/handler/ -v 2>&1 | tail -20
git -C ../../../.. status --porcelain -- services/atlas-channel/atlas.com/channel/skill/handler/common.go services/atlas-buffs/
```
Expected: existing handler tests (including `common_apply_to_mobs_test.go`) PASS; the `git status` output is EMPTY (no diffs in `common.go`, no diffs under `services/atlas-buffs/`).

- [ ] **Step 6: Commit**

```bash
git add skill/handler/dispel/
git commit -m "feat(task-163): Priest Dispel party debuff cure handler"
```

---

### Task 3: Registration blank import and module-wide verification

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`

**Interfaces:**
- Consumes: `atlas-channel/skill/handler/dispel` package `init()` from Task 2.
- Produces: production registration — `main.go` blank-imports `registrations`, which now pulls in dispel's `init()`.

- [ ] **Step 1: Add the blank import**

Edit `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` to:

```go
// Package registrations exists solely to drive init() registration of
// per-skill handler subpackages. main.go blank-imports this package;
// each new handler subpackage is added below as a blank import.
package registrations

import (
	_ "atlas-channel/skill/handler/dispel"     // Priest Dispel party cure — task-163
	_ "atlas-channel/skill/handler/heal"       // Cleric Heal — task 045
	_ "atlas-channel/skill/handler/mysticdoor" // Priest Mystic Door — task-093
)
```

- [ ] **Step 2: Run the full module verification**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go build ./... && go vet ./... && go test -race ./...
```
Expected: all clean / all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add skill/handler/registrations/registrations.go
git commit -m "feat(task-163): register dispel handler for skill dispatch"
```

---

### Task 4: Cross-module verification gates

**Files:** none created or modified — verification only (CLAUDE.md Build & Verification + design §7). A failure here reopens the offending task; do not claim done until every gate passes.

- [ ] **Step 1: Docker bake the changed service**

Run from the worktree root (`.worktrees/task-163-priest-dispel-party/`):
```bash
docker buildx bake atlas-channel
```
Expected: image builds successfully. (`go build` will NOT catch a missing `COPY libs/...` in the shared Dockerfile — only bake will. No new lib was added, so no Dockerfile edits are expected.)

- [ ] **Step 2: Redis key guard**

Run from the worktree root:
```bash
tools/redis-key-guard.sh
```
Expected: clean (exit 0). No new Redis usage was added, so any failure means an environment problem — do not prefix with a global `GOWORK=off`.

- [ ] **Step 3: Confirm zero atlas-buffs diffs and untouched mob path**

Run from the worktree root:
```bash
git diff --stat main...HEAD -- services/atlas-buffs/ services/atlas-channel/atlas.com/channel/skill/handler/common.go
git log --oneline main..HEAD
```
Expected: the diff --stat output is EMPTY. The log shows the three task commits (producer, handler, registration).

- [ ] **Step 4: Re-run the changed-module gates one final time**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test -race ./... && go vet ./... && go build ./...
```
Expected: clean. Only after all four steps pass may the branch proceed to code review (`superpowers:requesting-code-review` — mandatory before PR).

---

## Out of Scope (do not implement)

- SuperGM Heal+Dispel (9101000) — task-156. It will reuse Task 1's `CancelByTypes` with a wider stat set; whichever task lands second rebases onto the shared producer.
- Mob-side Dispel changes (`applyToMobs`, reflect handling) — already implemented.
- Curing ZOMBIFY / SEDUCE / CONFUSE.
- Mob→character disease infliction — no such path exists yet (PRD open question 1); live-play Dispel is a correct no-op until it lands. Acceptance seeds debuffs via a direct buff `APPLY` command.
- Skill-use announce packets for the party cure — PRD scopes none.
- `atlas-buffs` changes of any kind.
