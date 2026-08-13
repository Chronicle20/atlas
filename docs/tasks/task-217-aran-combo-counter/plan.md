# Aran Combo Counter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give an Aran/Legend with Combo Ability and a polearm a server-authoritative combo count that climbs on every damaging melee hit, is echoed back with `SHOW_COMBO`, applies/cancels the Combo Ability buff, and decays on the same idle schedule the client uses.

**Architecture:** The client sends a body-less `ARAN_COMBO_COUNTER` from `CMob::OnHit`; atlas-channel re-derives every gate from authoritative state, advances a **process-local, tenant-keyed `ComboMirror`**, and writes `SHOW_COMBO` back to the acting session. atlas-buffs is only asked to apply the Combo Ability buff once per combo chain and cancel it on decay — the count itself never travels through Kafka. Decay is a 1 Hz tick in atlas-channel that walks the mirror; the client clears its own HUD on the same window (3000 ms, 5000 ms on v95), so decay sends no packet.

**Tech Stack:** Go 1.x modules under `go.work`; `libs/atlas-packet` codecs (`Encode`/`Decode` + `pt.RoundTrip` tests); `libs/atlas-constants` generated identity tables; atlas-channel socket handlers/writers + `tasks.Register`; atlas-buffs Kafka commands on `COMMAND_TOPIC_CHARACTER_BUFF`; atlas-configurations seed templates.

## Global Constraints

Copied verbatim from `design.md` / `prd.md`. Every task's requirements implicitly include this section.

- **Six versions in scope:** gms v83, v84, v87, v92, v95, jms v185. Out of scope: v12, v48, v61, v72, v79.
- **Opcodes are already correct — no registry edits.** serverbound `ARAN_COMBO_COUNTER`: v83 `0x0A3`, v84 `0x0A9`, v87 `0x0AD`, v92 `0x0BA`, v95 `0x0BD`, jms185 `0x09D`. clientbound `SHOW_COMBO`: v83 `0x0E1`, v84 `0x0E6`, v87 `0x0EF`, v92 `0x103`, v95 `0x101`, jms185 `0x0EB`. (design.md §2.1, §2.2)
- **Serverbound body is empty on every version. Clientbound body is exactly one 4-byte little-endian count.** No version divergence in either codec — therefore **no `MajorAtLeast` call site exists in this feature** (design.md §3.6).
- **Idle window is tenant configuration, not a compiled branch:** handler option `idleResetMs` — `5000` in `template_gms_95_1.json`, `3000` in the other five, code default `3000` when absent (design.md §4).
- **Combo cap = 99999** (the client's 5-slot digit array, design.md §2.6).
- **Combo Ability skill id:** `21000000` (`skill.AranStage1ComboAbilityId`), or `20000017` (`skill.LegendComboAbilityId`, added in Task 3) when `c.JobId() == job.LegendId`. Client selector verified: `job != 2000 ? 21000000 : 20000017` (design.md §2.4).
- **Gates are the client's gates and nothing more:** Combo Ability learned at level > 0, and equipped weapon is `item.WeaponTypePolearm`. **No job-range check** — the skill check *is* the job gate (design.md §3.5).
- **Never send `SHOW_COMBO 0`.** `DrawCombo` early-returns on a non-positive count without releasing its layers, so a 0 cannot clear the HUD (design.md §2.5, §5.3).
- **Failure isolation (NFR-4):** every Kafka emit failure in this feature is logged and swallowed; nothing here may fail a player action.
- **Gate rejections are silent (NFR-2):** debug-level log only, no Kafka, no packet.
- **Goroutines only via `routine.Go`** (`tools/goroutine-guard.sh`). No raw `go` statements.
- **No new Redis keys.** The mirror is process-local (design.md §3.2).
- **Template entries must carry a non-empty `validator` (handlers) and an `fname` (writers)** — both are documented silent-drop traps — and must sit at their sorted `opCode` position.
- Tests use the project's Builder pattern for setup; no `*_testhelpers.go` files.
- No `// TODO`, stubs, or 501s in landed commits.

---

## File Structure

| File | Responsibility | New? |
|---|---|---|
| `libs/atlas-packet/character/serverbound/aran_combo_counter.go` | Empty-body serverbound codec + `AranComboCounterHandle` name const | create |
| `libs/atlas-packet/character/serverbound/aran_combo_counter_test.go` | Round-trip + empty-body fixtures | create |
| `libs/atlas-packet/character/clientbound/show_combo.go` | 4-byte count clientbound codec + `ShowComboWriter` name const | create |
| `libs/atlas-packet/character/clientbound/show_combo_test.go` | Round-trip + byte fixtures | create |
| `libs/atlas-constants/gen/identities.yaml` | Source for `LegendComboAbility` identity | modify |
| `libs/atlas-constants/skill/constants.go` | `LegendComboAbilityId` + `LegendComboAbilitySkill` + registry row | modify |
| `libs/atlas-constants/skill/*_gen.go` | Regenerated identity/version tables | regenerate |
| `services/atlas-data/atlas.com/data/skill/reader.go` | `ARAN_COMBO` statup value `100` → `int32(e.X())` | modify |
| `services/atlas-channel/atlas.com/channel/character/combo/mirror.go` | `ComboMirror`: count, lastHit, window, eligibility cache | create |
| `services/atlas-channel/atlas.com/channel/character/combo/mirror_test.go` | Mirror unit tests | create |
| `services/atlas-channel/atlas.com/channel/character/combo/eligibility.go` | Gate evaluation from `character.Model` | create |
| `services/atlas-channel/atlas.com/channel/character/combo/eligibility_test.go` | Gate unit tests | create |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo.go` | `AranComboCounterHandleFunc` — the increment path | create |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo_test.go` | Increment-path unit tests | create |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | Eligibility refresh hook beside `comboOrbTryUpdate` | modify |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_buff_cancel.go` | Clear the mirror on a Combo Ability cancel | modify |
| `services/atlas-channel/atlas.com/channel/session/processor.go` | Clear the mirror on session destroy | modify |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go` | Clear the mirror on map change | modify |
| `services/atlas-channel/atlas.com/channel/character/combo/task.go` | 1 Hz decay tick | create |
| `services/atlas-channel/atlas.com/channel/character/combo/task_test.go` | Decay unit tests | create |
| `services/atlas-channel/atlas.com/channel/main.go` | Handler map row, writer-name row, tick registration | modify |
| `services/atlas-configurations/seed-data/templates/template_{gms_83_1,gms_84_1,gms_87_1,gms_92_1,gms_95_1,jms_185_1}.json` | One handler + one writer entry each | modify |
| `docs/TODO.md` | Record the landed state | modify |

---

### Task 1: Serverbound `ARAN_COMBO_COUNTER` codec

**Files:**
- Create: `libs/atlas-packet/character/serverbound/aran_combo_counter.go`
- Test: `libs/atlas-packet/character/serverbound/aran_combo_counter_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `serverbound.AranComboCounterHandle` (string const, value `"AranComboCounterHandle"`) and `serverbound.AranComboCounterRequest` (zero-field struct with `Operation() string`, `String() string`, `Encode(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`, `Decode(logrus.FieldLogger, context.Context) func(*request.Reader, map[string]interface{})` — `Decode` has a pointer receiver, matching `ChalkboardClose`).

- [x] **Step 1: Write the failing test**

Create `libs/atlas-packet/character/serverbound/aran_combo_counter_test.go`:

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/AranComboCounter version=gms_v83 ida=0x9602f3
// packet-audit:verify packet=character/serverbound/AranComboCounter version=gms_v84 ida=0x99f346
// packet-audit:verify packet=character/serverbound/AranComboCounter version=gms_v87 ida=0x9e37bc
// packet-audit:verify packet=character/serverbound/AranComboCounter version=gms_v92 ida=0x8ef840
// packet-audit:verify packet=character/serverbound/AranComboCounter version=gms_v95 ida=0x909070
// packet-audit:verify packet=character/serverbound/AranComboCounter version=jms_v185 ida=0xa2d435
func TestAranComboCounterRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AranComboCounterRequest{}
			output := AranComboCounterRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestAranComboCounterEmptyBody pins CUserLocal::RequestIncCombo's wire on
// every in-scope version: the function is an m_bHoldCombo guard plus
// COutPacket(op) plus SendPacket, with zero Encode calls (design.md §2.1).
func TestAranComboCounterEmptyBody(t *testing.T) {
	cases := []struct {
		name   string
		region string
		major  uint16
		minor  uint16
	}{
		{"gms_v83", "GMS", 83, 1},
		{"gms_v84", "GMS", 84, 1},
		{"gms_v87", "GMS", 87, 1},
		{"gms_v92", "GMS", 92, 1},
		{"gms_v95", "GMS", 95, 1},
		{"jms_v185", "JMS", 185, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := pt.CreateContext(c.region, c.major, c.minor)
			got := pt.Encode(t, ctx, AranComboCounterRequest{}.Encode, nil)
			if len(got) != 0 {
				t.Errorf("expected empty body, got % x", got)
			}
		})
	}
}
```

Before running, confirm `pt.CreateContext`'s parameter types match the call above by reading an existing sibling test (e.g. `chalkboard_close_test.go`, which calls `pt.CreateContext("GMS", 79, 1)`); if the helper takes different integer widths, adjust the struct field types in the table to match — do not change the helper.

- [x] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-packet && go test ./character/serverbound/ -run AranComboCounter -v
```

Expected: FAIL — `undefined: AranComboCounterRequest`.

- [x] **Step 3: Write minimal implementation**

Create `libs/atlas-packet/character/serverbound/aran_combo_counter.go`:

```go
package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

const AranComboCounterHandle = "AranComboCounterHandle"

// AranComboCounterRequest - CUserLocal::RequestIncCombo. Sent from
// CMob::OnHit when the local Aran lands a damaging hit with a polearm while
// owning Combo Ability. The body is empty on every in-scope version (v83,
// v84, v87, v92, v95, jms185 -- design.md §2.1), so there is nothing here to
// trust: every gate is re-derived server-side.
type AranComboCounterRequest struct{}

func (m AranComboCounterRequest) Operation() string {
	return AranComboCounterHandle
}

func (m AranComboCounterRequest) String() string {
	return ""
}

func (m AranComboCounterRequest) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *AranComboCounterRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
```

- [x] **Step 4: Run tests to verify they pass**

```bash
cd libs/atlas-packet && go test ./character/serverbound/ -run AranComboCounter -v
```

Expected: PASS for both tests, every subtest.

- [x] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/serverbound/aran_combo_counter.go libs/atlas-packet/character/serverbound/aran_combo_counter_test.go
git commit -m "feat(task-217): ARAN_COMBO_COUNTER serverbound codec"
```

---

### Task 2: Clientbound `SHOW_COMBO` codec

**Files:**
- Create: `libs/atlas-packet/character/clientbound/show_combo.go`
- Test: `libs/atlas-packet/character/clientbound/show_combo_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `clientbound.ShowComboWriter` (string const, value `"ShowCombo"` — the name the seed templates bind), `clientbound.NewShowCombo(count uint32) ShowCombo`, accessor `Count() uint32`, plus `Operation()`, `String()`, `Encode`, `Decode` (pointer receiver on `Decode`).

- [x] **Step 1: Write the failing test**

Create `libs/atlas-packet/character/clientbound/show_combo_test.go`:

```go
package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/ShowCombo version=gms_v83 ida=0x95086b
// packet-audit:verify packet=character/clientbound/ShowCombo version=gms_v84 ida=0x988698
// packet-audit:verify packet=character/clientbound/ShowCombo version=gms_v87 ida=0x9cbcd4
// packet-audit:verify packet=character/clientbound/ShowCombo version=gms_v92 ida=0x913a68
// packet-audit:verify packet=character/clientbound/ShowCombo version=gms_v95 ida=0x934238
// packet-audit:verify packet=character/clientbound/ShowCombo version=jms_v185 ida=0xa1208d
func TestShowComboRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewShowCombo(37)
			output := ShowCombo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Count() != 37 {
				t.Errorf("count round-trip mismatch: want 37, got %d", output.Count())
			}
		})
	}
}

// TestShowComboByteFixture pins CUserLocal::OnIncComboResponse's wire: a
// single Decode4 into m_nCombo, then DrawCombo (design.md §2.2). No version
// divergence -- the body is identical on all six in-scope versions.
func TestShowComboByteFixture(t *testing.T) {
	cases := []struct {
		name   string
		region string
		major  uint16
		minor  uint16
	}{
		{"gms_v83", "GMS", 83, 1},
		{"gms_v84", "GMS", 84, 1},
		{"gms_v87", "GMS", 87, 1},
		{"gms_v92", "GMS", 92, 1},
		{"gms_v95", "GMS", 95, 1},
		{"jms_v185", "JMS", 185, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := pt.CreateContext(c.region, c.major, c.minor)
			got := pt.Encode(t, ctx, NewShowCombo(1).Encode, nil)
			want := []byte{0x01, 0x00, 0x00, 0x00}
			if len(got) != len(want) {
				t.Fatalf("length mismatch: want % x, got % x", want, got)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte mismatch: want % x, got % x", want, got)
				}
			}
		})
	}
}
```

Adjust the struct-literal integer widths only if `pt.CreateContext`'s signature differs (see Task 1 Step 1).

- [x] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-packet && go test ./character/clientbound/ -run ShowCombo -v
```

Expected: FAIL — `undefined: NewShowCombo`.

- [x] **Step 3: Write minimal implementation**

Create `libs/atlas-packet/character/clientbound/show_combo.go`. Read a sibling clientbound file (e.g. `hint.go`) first to confirm the writer/reader helper names in this module (`response.NewWriter`, `w.WriteInt32`/`WriteUint32`, and the reader accessor used by other `Decode` implementations) and use whichever the module actually exposes for a 4-byte little-endian value:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ShowComboWriter = "ShowCombo"

// ShowCombo - CUserLocal::OnIncComboResponse. Body is one 4-byte
// little-endian combo count, decoded into m_nCombo and handed to DrawCombo
// (design.md §2.2). Identical on every in-scope version.
//
// Never send a count of 0: DrawCombo early-returns on a non-positive count
// WITHOUT releasing its digit layers, so a 0 leaves stale digits on screen
// rather than clearing them. The client clears its own HUD on its idle timer
// (design.md §2.5, §5.3).
type ShowCombo struct {
	count uint32
}

func NewShowCombo(count uint32) ShowCombo {
	return ShowCombo{count: count}
}

func (m ShowCombo) Count() uint32     { return m.count }
func (m ShowCombo) Operation() string { return ShowComboWriter }
func (m ShowCombo) String() string    { return fmt.Sprintf("count [%d]", m.count) }

func (m ShowCombo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.count)
		return w.Bytes()
	}
}

func (m *ShowCombo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.count = r.ReadUint32()
	}
}
```

If `response.Writer` names its 4-byte method differently (e.g. `WriteInt32(int32)`) or `request.Reader` names its 4-byte read differently (e.g. `ReadInt32()`), use the module's actual names and cast at the boundary — the wire must remain 4 bytes little-endian.

- [x] **Step 4: Run tests to verify they pass**

```bash
cd libs/atlas-packet && go test ./character/clientbound/ -run ShowCombo -v
```

Expected: PASS for both tests, every subtest.

- [x] **Step 5: Full module check and commit**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./...
cd - && git add libs/atlas-packet/character/clientbound/show_combo.go libs/atlas-packet/character/clientbound/show_combo_test.go
git commit -m "feat(task-217): SHOW_COMBO clientbound codec"
```

---

### Task 3: `LegendComboAbilityId` in atlas-constants

The Legend branch is real: `20000017` is present in the WZ snapshots for gms 79/83/84/87/92/95 and jms 185 (`libs/atlas-constants/gen/wzsnapshot/*.json`), contradicting the PRD's FR-5.1 doubt (design.md §2.7). Identity tables are **generated** from `gen/identities.yaml` — never hand-edit `*_gen.go`.

**Files:**
- Modify: `libs/atlas-constants/gen/identities.yaml`
- Modify: `libs/atlas-constants/skill/constants.go`
- Regenerate: `libs/atlas-constants/skill/identities_gen.go`, `libs/atlas-constants/skill/version_*_gen.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `skill.LegendComboAbilityId` (typed `skill.Id`, value `20000017`) and identity `skill.LegendComboAbility`. Later tasks compare `s.Id() == skill.LegendComboAbilityId` directly — permitted, since neither combo id is on `tools/skill-job-id-guard.sh`'s divergent list (design.md §2.7).

- [x] **Step 1: Add the identity to the generator source**

In `libs/atlas-constants/gen/identities.yaml`, add an entry in the `2000xxxx` (Legend) block, keeping the file's existing numeric ordering — insert it at the position where `20000017` sorts among its neighbours:

```yaml
- name: LegendComboAbility
  domain: skill
  canonicalToken: 20000017
  displayName: Legend Combo Ability
```

- [x] **Step 2: Add the wire constant and Skill value**

In `libs/atlas-constants/skill/constants.go`, add the id constant beside the other Legend ids (the `LegendBlessOfNymphId` neighbourhood), the `Skill` value beside `LegendBlessOfNymphSkill` (~line 2038), and the registry row beside the `AranStage1ComboAbilityId:` row (~line 2832). Follow `AranStage1ComboAbilitySkill` exactly — Combo Ability is a buff:

```go
// in the Legend id const block
LegendComboAbilityId = Id(20000017)
```

```go
// beside LegendBlessOfNymphSkill
var LegendComboAbilitySkill = Skill{
	id:   LegendComboAbilityId,
	buff: true,
}
```

```go
// in the id -> Skill registry map, beside AranStage1ComboAbilityId
LegendComboAbilityId: LegendComboAbilitySkill,
```

Match the surrounding alignment/gofmt style; `tools/lint.sh` will rewrite it if not.

- [x] **Step 3: Regenerate the identity tables**

```bash
cd libs/atlas-constants/gen && go run .
```

Expected: `identities_gen.go` and the per-version `version_*_gen.go` files are rewritten. `LegendComboAbility` must appear in the six in-scope version tables (it is in their WZ snapshots).

- [x] **Step 4: Verify generation is complete and non-stale**

```bash
cd libs/atlas-constants/gen && go run . -check
cd .. && go test -race ./... && go vet ./...
grep -n "LegendComboAbility" skill/identities_gen.go skill/version_gms_83_1_gen.go skill/version_gms_95_1_gen.go
```

Expected: `-check` exits 0; tests pass (the drift test in `gen/drift_test.go` and `audit_validate_test.go` must stay green); the grep shows `LegendComboAbility Identity = 20000017` and the per-version mapping rows.

- [x] **Step 5: Commit**

```bash
git add libs/atlas-constants/gen/identities.yaml libs/atlas-constants/skill/
git commit -m "feat(task-217): add Legend Combo Ability (20000017) skill identity"
```

---

### Task 4: `ARAN_COMBO` statup value in atlas-data

`reader.go:470` emits a hardcoded `100` with no provenance in WZ or in the client. The stat is a damage-calculation input decoded as a signed short, not the combo count (design.md §2.3). Combo Ability's WZ node carries `x = <level>` (1–20 for Aran, 10 for the single-level Legend variant), and every sibling Aran statup already passes `e.X()`.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/reader.go:470`
- Test: `services/atlas-data/atlas.com/data/skill/reader_test.go` (add to the existing file; create it only if absent)

**Interfaces:**
- Consumes: `skill.LegendComboAbilityId` from Task 3.
- Produces: nothing consumed by later tasks.

- [x] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/skill/reader_test.go`, following the XML-driven shape of the existing `TestReader_SuperGmHolySymbol_V48Wire_ClassifiesAsHolySymbol` (~line 3264) — same imports, same `Read` → `CollectToMap` → `findStatup` chain. The Combo Ability branch sits in the function-level `else if` chain, not inside the `e.OverTime()` block, so a bare `level/x` node is enough to reach it:

```go
// TestReader_AranComboAbility_StatupUsesEffectX pins the ARAN_COMBO statup
// amount to the skill effect's x. The previous hardcoded 100 had no
// provenance in WZ (Combo Ability's node carries only hs/x/y/z/weapon/
// invisible) or in the client, and the stat is a damage-calculation input
// decoded as a SIGNED SHORT -- not the combo count, which SHOW_COMBO carries
// (task-217 design.md §2.3).
func TestReader_AranComboAbility_StatupUsesEffectX(t *testing.T) {
	tests := []struct {
		name    string
		imgdir  string
		skillId string
		x       int32
	}{
		{"aran combo ability level 1", "2100.img", "21000000", 1},
		{"aran combo ability level 20", "2100.img", "21000000", 20},
		{"legend combo ability", "2000.img", "20000017", 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l, _ := test.NewNullLogger()
			tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
			if err != nil {
				t.Fatal(err)
			}
			ctx := tenant.WithContext(context.Background(), tn)

			xmlData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="%s">
  <imgdir name="skill">
    <imgdir name="%s">
      <imgdir name="level">
        <imgdir name="1">
          <int name="x" value="%d"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`, tc.imgdir, tc.skillId, tc.x)

			d, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(xmlData)))()
			if err != nil {
				t.Fatal(err)
			}
			rms := model.FixedProvider(d.Models)
			rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
			if err != nil {
				t.Fatal(err)
			}
			rm, ok := rmm[tc.skillId]
			if !ok {
				t.Fatalf("rmm[%s] does not exist.", tc.skillId)
			}
			if len(rm.Effects) != 1 {
				t.Fatalf("len(rm.Effects) = %d, want 1", len(rm.Effects))
			}
			su, ok := findStatup(rm.Effects[0].Statups, string(character.TemporaryStatTypeAranCombo))
			if !ok {
				t.Fatalf("expected an ARAN_COMBO statup for skill %s, got none in %+v", tc.skillId, rm.Effects[0].Statups)
			}
			if su.Amount != tc.x {
				t.Fatalf("ARAN_COMBO statup Amount = %d, want %d", su.Amount, tc.x)
			}
		})
	}
}
```

Add `fmt` to the file's imports if it is not already there.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-data/atlas.com/data && go test ./skill/ -run AranCombo -v
```

Expected: FAIL — the Aran level-1 case gets `100`, want `1`; the Legend case finds no `ARAN_COMBO` statup at all (`20000017` is not in the branch yet).

- [x] **Step 3: Change the reader**

In `services/atlas-data/atlas.com/data/skill/reader.go`, replace the Combo Ability branch so it matches its siblings and covers the Legend variant:

```go
	} else if skill.Is(skillId, skill.AranStage1ComboAbilityId, skill.LegendComboAbilityId) {
		// ARAN_COMBO is a damage-calculation input decoded by the client as a
		// signed short (SecondaryStat::DecodeForLocal), NOT the combo count --
		// the count is delivered by SHOW_COMBO and never touches this stat
		// (task-217 design.md §2.3). Combo Ability's WZ node carries no field
		// whose value is 100; x is the level-scaled magnitude, matching every
		// sibling Aran statup below.
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeAranCombo, int32(e.X()))
```

- [x] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./...
```

Expected: PASS, including all three new subtests.

- [x] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/skill/
git commit -m "fix(task-217): ARAN_COMBO statup carries effect x, not hardcoded 100"
```

---

### Task 5: `ComboMirror`

Process-local, tenant-keyed, `sync.RWMutex`-guarded — modelled on `services/atlas-channel/atlas.com/channel/character/buff/beacon.go`. It holds the count, the last-hit time, the resolved idle window, and the cached eligibility so the hot path costs one map read/write.

The mirror stores the `tenant.Model` alongside each tenant's bucket because the decay tick has no request context and must rebuild one per tenant (`tenant.WithContext`), exactly as `ProcessPoisonTicks` does in atlas-buffs.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/combo/mirror.go`
- Test: `services/atlas-channel/atlas.com/channel/character/combo/mirror_test.go`

**Interfaces:**
- Consumes: `skill.Id`, `field.Model`, `tenant.Model`.
- Produces (used by Tasks 6–10):
  - `const ComboCap int32 = 99999`
  - `const DefaultIdleWindow = 3000 * time.Millisecond`
  - `type Eligibility struct{}` with `NewEligibility(comboId skill.Id, comboLevel byte, statAmount int32) Eligibility` and accessors `ComboId() skill.Id`, `ComboLevel() byte`, `StatAmount() int32`
  - `type Entry struct{}` with accessors `Count() int32`, `LastHit() time.Time`, `Window() time.Duration`, `Field() field.Model`, `Eligibility() Eligibility`, `CheckedAt() time.Time`
  - `func GetMirror() *Mirror`
  - `func (m *Mirror) SetEligibility(t tenant.Model, characterId uint32, f field.Model, e Eligibility, now time.Time)`
  - `func (m *Mirror) Eligibility(t tenant.Model, characterId uint32, now time.Time, ttl time.Duration) (Eligibility, bool)` — returns `false` when absent or stale
  - `func (m *Mirror) Increment(t tenant.Model, characterId uint32, f field.Model, window time.Duration, now time.Time) (count int32, seeded bool)` — clamps at `ComboCap`; `seeded` is true only on the 0→1 transition
  - `func (m *Mirror) Clear(t tenant.Model, characterId uint32)`
  - `type Expired struct{}` with accessors `Tenant() tenant.Model`, `CharacterId() uint32`, `Field() field.Model`, `ComboId() skill.Id`
  - `func (m *Mirror) ExpireIdle(now time.Time) []Expired` — zeroes the count of every entry with `Count() > 0 && now.Sub(LastHit()) > Window()` and returns what it zeroed

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/character/combo/mirror_test.go`:

```go
package combo

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return m
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
}

func testEligibility() Eligibility {
	return NewEligibility(skill.AranStage1ComboAbilityId, 20, 20)
}

func TestIncrementSeedsThenAdvances(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)

	m.SetEligibility(tn, 7, testField(), testEligibility(), now)

	count, seeded := m.Increment(tn, 7, testField(), DefaultIdleWindow, now)
	if count != 1 || !seeded {
		t.Fatalf("first increment: want (1,true), got (%d,%v)", count, seeded)
	}
	count, seeded = m.Increment(tn, 7, testField(), DefaultIdleWindow, now.Add(time.Second))
	if count != 2 || seeded {
		t.Fatalf("second increment: want (2,false), got (%d,%v)", count, seeded)
	}
}

func TestIncrementClampsAtCap(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)
	m.SetEligibility(tn, 7, testField(), testEligibility(), now)
	m.setCountForTest(tn, 7, ComboCap)

	count, seeded := m.Increment(tn, 7, testField(), DefaultIdleWindow, now)
	if count != ComboCap || seeded {
		t.Fatalf("at cap: want (%d,false), got (%d,%v)", ComboCap, count, seeded)
	}
}

func TestEligibilityTTL(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)
	m.SetEligibility(tn, 7, testField(), testEligibility(), now)

	if _, ok := m.Eligibility(tn, 7, now.Add(59*time.Second), 60*time.Second); !ok {
		t.Error("within TTL: want fresh, got stale")
	}
	if _, ok := m.Eligibility(tn, 7, now.Add(61*time.Second), 60*time.Second); ok {
		t.Error("past TTL: want stale, got fresh")
	}
	if _, ok := m.Eligibility(tn, 99, now, 60*time.Second); ok {
		t.Error("unknown character: want miss, got hit")
	}
}

func TestClearRemovesEntry(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)
	m.SetEligibility(tn, 7, testField(), testEligibility(), now)
	m.Increment(tn, 7, testField(), DefaultIdleWindow, now)

	m.Clear(tn, 7)

	if _, ok := m.Eligibility(tn, 7, now, 60*time.Second); ok {
		t.Error("after Clear: want miss, got hit")
	}
	count, seeded := m.Increment(tn, 7, testField(), DefaultIdleWindow, now)
	if count != 1 || !seeded {
		t.Fatalf("after Clear the next increment re-seeds: want (1,true), got (%d,%v)", count, seeded)
	}
}

func TestExpireIdleZeroesOnlyStaleNonZeroEntries(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)

	// stale, count > 0 -> expires
	m.SetEligibility(tn, 1, testField(), testEligibility(), now)
	m.Increment(tn, 1, testField(), 3*time.Second, now)
	// fresh, count > 0 -> untouched
	m.SetEligibility(tn, 2, testField(), testEligibility(), now)
	m.Increment(tn, 2, testField(), 3*time.Second, now.Add(3*time.Second))
	// eligibility only, count == 0 -> emits nothing
	m.SetEligibility(tn, 3, testField(), testEligibility(), now)

	got := m.ExpireIdle(now.Add(4 * time.Second))

	if len(got) != 1 {
		t.Fatalf("want exactly 1 expiry, got %d", len(got))
	}
	if got[0].CharacterId() != 1 {
		t.Errorf("want character 1 expired, got %d", got[0].CharacterId())
	}
	if got[0].ComboId() != skill.AranStage1ComboAbilityId {
		t.Errorf("want combo id %d, got %d", skill.AranStage1ComboAbilityId, got[0].ComboId())
	}
	if got[0].Tenant().Id() != tn.Id() {
		t.Error("expired entry carries the wrong tenant")
	}
	// second sweep is empty: the count is already zero
	if again := m.ExpireIdle(now.Add(8 * time.Second)); len(again) != 0 {
		t.Errorf("second sweep: want 0 expiries, got %d", len(again))
	}
}

func TestPerTenantIsolation(t *testing.T) {
	m := &Mirror{}
	a := testTenant(t)
	b := testTenant(t)
	now := time.Unix(1000, 0)

	m.SetEligibility(a, 7, testField(), testEligibility(), now)
	m.SetEligibility(b, 7, testField(), testEligibility(), now)
	m.Increment(a, 7, testField(), DefaultIdleWindow, now)
	m.Increment(a, 7, testField(), DefaultIdleWindow, now)

	count, _ := m.Increment(b, 7, testField(), DefaultIdleWindow, now)
	if count != 1 {
		t.Errorf("tenant b must not see tenant a's count: want 1, got %d", count)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)
	m.SetEligibility(tn, 7, testField(), testEligibility(), now)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Increment(tn, 7, testField(), DefaultIdleWindow, now)
			m.Eligibility(tn, 7, now, 60*time.Second)
			m.ExpireIdle(now)
		}()
	}
	wg.Wait()
}
```

`TestConcurrentAccess` uses bare `go` inside a `_test.go` file; `tools/goroutine-guard.sh` scopes to non-test code — confirm with `tools/goroutine-guard.sh` in Step 4 and, if it flags the test, replace the goroutines with `routine.Go`.

Confirm `tenant.Create`'s signature and `field.NewBuilder`'s argument types against a sibling test (e.g. `character/buff/beacon_test.go`) before running, and adjust the helpers to match.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./character/combo/ -v
```

Expected: FAIL — the package does not exist.

- [x] **Step 3: Write the implementation**

Create `services/atlas-channel/atlas.com/channel/character/combo/mirror.go`:

```go
// Package combo owns the Aran combo counter's channel-local state.
//
// The count is deliberately NOT stored as the ARAN_COMBO buff stat: that stat
// is a damage-calculation input the client decodes as a signed short, and the
// count is delivered to the client by SHOW_COMBO instead (task-217 design.md
// §2.3, §5.1). Keeping the count here also keeps the hit-frequency increment
// path free of both Redis and Kafka.
package combo

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ComboCap bounds the count at five digits. The client's combo renderer
// decomposes m_nCombo into a 5-slot digit-layer array; a six-digit count
// would overrun it (task-217 design.md §2.6). Nothing in WZ governs a cap.
const ComboCap int32 = 99999

// DefaultIdleWindow is the fallback when a tenant's socket config carries no
// idleResetMs handler option. It matches the client's own ClearCombo timer on
// five of the six in-scope versions; v95 uses 5000 ms and configures it
// (task-217 design.md §2.5, §4).
const DefaultIdleWindow = 3000 * time.Millisecond

// Eligibility is the cached result of the server-side gate evaluation: which
// Combo Ability the character owns, at what level, and the ARAN_COMBO statup
// amount to seed the buff with.
type Eligibility struct {
	comboId    skill.Id
	comboLevel byte
	statAmount int32
}

func NewEligibility(comboId skill.Id, comboLevel byte, statAmount int32) Eligibility {
	return Eligibility{comboId: comboId, comboLevel: comboLevel, statAmount: statAmount}
}

func (e Eligibility) ComboId() skill.Id { return e.comboId }
func (e Eligibility) ComboLevel() byte  { return e.comboLevel }
func (e Eligibility) StatAmount() int32 { return e.statAmount }

// Entry is one character's live combo state.
type Entry struct {
	count       int32
	lastHit     time.Time
	window      time.Duration
	f           field.Model
	eligibility Eligibility
	checkedAt   time.Time
}

func (e Entry) Count() int32               { return e.count }
func (e Entry) LastHit() time.Time         { return e.lastHit }
func (e Entry) Window() time.Duration      { return e.window }
func (e Entry) Field() field.Model         { return e.f }
func (e Entry) Eligibility() Eligibility   { return e.eligibility }
func (e Entry) CheckedAt() time.Time       { return e.checkedAt }

type bucket struct {
	t       tenant.Model
	entries map[uint32]Entry
}

// Mirror is the process-wide, tenant-keyed combo state.
//
// Process-local by design: a combo lives 3-5 seconds and dies with the
// session, so losing it to a channel restart is indistinguishable from an
// idle reset, and a session is pinned to one channel process so there is no
// cross-process reader (task-217 design.md §3.2). Same accepted degradation
// as BeaconMirror.
type Mirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]*bucket
}

var (
	mirror     *Mirror
	mirrorOnce sync.Once
)

// GetMirror returns the process-wide singleton, lazily initialising it.
func GetMirror() *Mirror {
	mirrorOnce.Do(func() { mirror = &Mirror{} })
	return mirror
}

// bucketFor returns the tenant's bucket, creating it. Callers hold m.mu.
func (m *Mirror) bucketFor(t tenant.Model) *bucket {
	if m.perTenant == nil {
		m.perTenant = make(map[uuid.UUID]*bucket)
	}
	b, ok := m.perTenant[t.Id()]
	if !ok {
		b = &bucket{t: t, entries: make(map[uint32]Entry)}
		m.perTenant[t.Id()] = b
	}
	// Refresh the stored tenant so the decay sweep always rebuilds a context
	// from a current model.
	b.t = t
	return b
}

// SetEligibility records or refreshes the character's gate result without
// touching the count. Called from the attack pipeline and from the handler's
// lazy cold-start fetch.
func (m *Mirror) SetEligibility(t tenant.Model, characterId uint32, f field.Model, e Eligibility, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.bucketFor(t)
	entry := b.entries[characterId]
	entry.eligibility = e
	entry.checkedAt = now
	entry.f = f
	b.entries[characterId] = entry
}

// Eligibility returns the cached gate result when it is present and no older
// than ttl.
func (m *Mirror) Eligibility(t tenant.Model, characterId uint32, now time.Time, ttl time.Duration) (Eligibility, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.perTenant[t.Id()]
	if !ok {
		return Eligibility{}, false
	}
	e, ok := b.entries[characterId]
	if !ok || e.checkedAt.IsZero() || now.Sub(e.checkedAt) > ttl {
		return Eligibility{}, false
	}
	return e.eligibility, true
}

// Increment advances the count by one, clamped at ComboCap, and refreshes the
// idle timer. seeded reports the 0 -> 1 transition, which is the only moment
// the Combo Ability buff needs applying.
func (m *Mirror) Increment(t tenant.Model, characterId uint32, f field.Model, window time.Duration, now time.Time) (int32, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.bucketFor(t)
	e := b.entries[characterId]
	seeded := e.count == 0
	if e.count < ComboCap {
		e.count++
	} else {
		seeded = false
	}
	e.lastHit = now
	e.window = window
	e.f = f
	b.entries[characterId] = e
	return e.count, seeded
}

// Clear drops the character's entry entirely: count, idle timer, and cached
// eligibility. Session end, map change, and the client's own combo cancel all
// funnel here.
func (m *Mirror) Clear(t tenant.Model, characterId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.perTenant[t.Id()]; ok {
		delete(b.entries, characterId)
	}
}

// Expired describes one combo the idle sweep zeroed.
type Expired struct {
	t           tenant.Model
	characterId uint32
	f           field.Model
	comboId     skill.Id
}

func (e Expired) Tenant() tenant.Model  { return e.t }
func (e Expired) CharacterId() uint32   { return e.characterId }
func (e Expired) Field() field.Model    { return e.f }
func (e Expired) ComboId() skill.Id     { return e.comboId }

// ExpireIdle zeroes every entry whose idle window has elapsed and returns
// what it zeroed so the caller can cancel the buff. The cached eligibility is
// intentionally retained: the character is still an Aran with a polearm, and
// dropping it would force a refetch on their next hit.
//
// No packet is sent for an expiry -- SHOW_COMBO 0 cannot clear the client's
// HUD, and the client clears itself on the same schedule (design.md §5.3).
func (m *Mirror) ExpireIdle(now time.Time) []Expired {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Expired
	for _, b := range m.perTenant {
		for id, e := range b.entries {
			if e.count <= 0 || now.Sub(e.lastHit) <= e.window {
				continue
			}
			e.count = 0
			b.entries[id] = e
			out = append(out, Expired{t: b.t, characterId: id, f: e.f, comboId: e.eligibility.comboId})
		}
	}
	return out
}

// setCountForTest seeds a count directly so cap behaviour is testable without
// 99999 calls to Increment.
func (m *Mirror) setCountForTest(t tenant.Model, characterId uint32, count int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.bucketFor(t)
	e := b.entries[characterId]
	e.count = count
	b.entries[characterId] = e
}
```

- [x] **Step 4: Run tests and guards to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./character/combo/ -v && go vet ./character/combo/
cd ../../../.. && tools/goroutine-guard.sh && tools/redis-key-guard.sh
```

Expected: all subtests PASS under `-race`; both guards exit 0.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/combo/
git commit -m "feat(task-217): channel-local Aran combo mirror"
```

---

### Task 6: Eligibility gates

The gate set is exactly the client's (design.md §2.4, §3.5): Combo Ability owned at level > 0 — `21000000`, or `20000017` when the job is Legend — and an equipped polearm. **No job-range check**: adding one the client does not apply would reject legitimate states.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/combo/eligibility.go`
- Test: `services/atlas-channel/atlas.com/channel/character/combo/eligibility_test.go`

**Interfaces:**
- Consumes: `combo.NewEligibility` (Task 5), `skill.LegendComboAbilityId` (Task 3).
- Produces:
  - `func ComboAbilityId(jobId job.Id) skill.Id`
  - `func Evaluate(c character.Model, getEffect func(skillId uint32, level byte) (effect.Model, error)) (Eligibility, string, bool)` — returns the eligibility, and on failure the name of the failing gate (`"skill"`, `"weapon"`, `"effect"`) for the debug log.

> Import direction note: `combo` imports `atlas-channel/character` and `atlas-channel/data/skill/effect`. `atlas-channel/character` must not import `combo`. Before writing, run `grep -rn "atlas-channel/character/combo" services/atlas-channel/atlas.com/channel/character/*.go` and confirm it is empty; if a cycle appears, move `Evaluate` into the handler package instead and keep only `ComboAbilityId` here.

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/character/combo/eligibility_test.go`. The construction below uses the real project constructors: `character.NewModelBuilder().…MustBuild()`, `skill.Extract(skill.RestModel{...})`, `effect.Extract(effect.RestModel{X: …})`, `equipment.NewModel()` + `Set`, and `asset.NewBuilder(compartmentId, templateId)` — the same helpers `socket/handler/character_attack_combo_test.go:113-143` uses:

```go
package combo

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	chskill "atlas-channel/character/skill"
	"atlas-channel/data/skill/effect"
	"atlas-channel/equipment"
	equipslot "atlas-channel/equipment/slot"
	"errors"
	"testing"

	"github.com/google/uuid"

	slottype "github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func testSkill(t *testing.T, id skill.Id, level byte) chskill.Model {
	t.Helper()
	m, err := chskill.Extract(chskill.RestModel{Id: uint32(id), Level: level})
	if err != nil {
		t.Fatalf("skill.Extract: %v", err)
	}
	return m
}

func buildCharacter(t *testing.T, jobId job.Id, skillId skill.Id, level byte, weaponTemplateId uint32) character.Model {
	t.Helper()
	eq := equipment.NewModel()
	if weaponTemplateId != 0 {
		w := asset.NewBuilder(uuid.New(), weaponTemplateId).Build()
		eq.Set(slottype.Type("weapon"), equipslot.Model{Equipable: &w})
	}
	return character.NewModelBuilder().
		SetId(1).
		SetJobId(jobId).
		SetSkills([]chskill.Model{testSkill(t, skillId, level)}).
		SetEquipment(eq).
		MustBuild()
}

func stubEffect(t *testing.T, x int16) func(uint32, byte) (effect.Model, error) {
	t.Helper()
	return func(uint32, byte) (effect.Model, error) {
		se, err := effect.Extract(effect.RestModel{X: x})
		if err != nil {
			t.Fatalf("effect.Extract: %v", err)
		}
		return se, nil
	}
}

func failingEffect() func(uint32, byte) (effect.Model, error) {
	return func(uint32, byte) (effect.Model, error) {
		return effect.Model{}, errors.New("atlas-data unreachable")
	}
}

func TestComboAbilityIdSelectsLegendVariant(t *testing.T) {
	if got := ComboAbilityId(job.LegendId); got != skill.LegendComboAbilityId {
		t.Errorf("Legend: want %d, got %d", skill.LegendComboAbilityId, got)
	}
	for _, j := range []job.Id{job.AranStage1Id, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id} {
		if got := ComboAbilityId(j); got != skill.AranStage1ComboAbilityId {
			t.Errorf("job %d: want %d, got %d", j, skill.AranStage1ComboAbilityId, got)
		}
	}
}

func TestEvaluateGates(t *testing.T) {
	// polearmItemId: (id/10000)%100 == 44 -> item.WeaponTypePolearm.
	const polearmItemId = uint32(1442000)
	// swordItemId: (id/10000)%100 == 30 -> one-handed sword.
	const swordItemId = uint32(1302000)

	tests := []struct {
		name     string
		jobId    job.Id
		skillId  skill.Id
		level    byte
		weapon   uint32
		wantOk   bool
		wantGate string
	}{
		{"aran with combo ability and polearm", job.AranStage1Id, skill.AranStage1ComboAbilityId, 5, polearmItemId, true, ""},
		{"legend with 20000017 and polearm", job.LegendId, skill.LegendComboAbilityId, 1, polearmItemId, true, ""},
		{"aran without combo ability", job.AranStage1Id, skill.AranStage1DoubleSwingId, 5, polearmItemId, false, "skill"},
		{"aran with combo ability at level 0", job.AranStage1Id, skill.AranStage1ComboAbilityId, 0, polearmItemId, false, "skill"},
		{"aran with a sword", job.AranStage1Id, skill.AranStage1ComboAbilityId, 5, swordItemId, false, "weapon"},
		{"non-aran", job.WarriorId, skill.AranStage1ComboAbilityId, 0, polearmItemId, false, "skill"},
		{"legend holding the aran id", job.LegendId, skill.AranStage1ComboAbilityId, 5, polearmItemId, false, "skill"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := buildCharacter(t, tc.jobId, tc.skillId, tc.level, tc.weapon)
			got, gate, ok := Evaluate(c, stubEffect(t, 7))
			if ok != tc.wantOk {
				t.Fatalf("eligible: want %v, got %v (gate %q)", tc.wantOk, ok, gate)
			}
			if !ok {
				if gate != tc.wantGate {
					t.Errorf("failing gate: want %q, got %q", tc.wantGate, gate)
				}
				return
			}
			if got.ComboId() != ComboAbilityId(tc.jobId) {
				t.Errorf("combo id: want %d, got %d", ComboAbilityId(tc.jobId), got.ComboId())
			}
			if got.ComboLevel() != tc.level {
				t.Errorf("combo level: want %d, got %d", tc.level, got.ComboLevel())
			}
			if got.StatAmount() != 7 {
				t.Errorf("stat amount: want 7 (effect x), got %d", got.StatAmount())
			}
		})
	}
}

func TestEvaluateEffectLookupFailure(t *testing.T) {
	c := buildCharacter(t, job.AranStage1Id, skill.AranStage1ComboAbilityId, 5, uint32(1442000))
	_, gate, ok := Evaluate(c, failingEffect())
	if ok {
		t.Fatal("effect lookup failed: want ineligible")
	}
	if gate != "effect" {
		t.Errorf("failing gate: want \"effect\", got %q", gate)
	}
}
```

Before running, confirm these names against the repo and substitute real neighbours if any differs: `job.AranStage2Id` / `AranStage3Id` / `AranStage4Id` / `WarriorId` and `skill.AranStage1DoubleSwingId` (`grep -n` in `libs/atlas-constants/{job,skill}/constants.go`); the equipment slot key (`equipment.Model.Get` takes an `inventory/slot.Type` — check whether the weapon slot's `Type` constant is exported as e.g. `slottype.TypeWeapon` rather than the literal `"weapon"`, and prefer the constant); and `asset.NewBuilder(...).Build()`'s return (if it returns a pointer, drop the `&`).

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./character/combo/ -run "ComboAbilityId|Evaluate" -v
```

Expected: FAIL — `undefined: ComboAbilityId`, `undefined: Evaluate`.

- [x] **Step 3: Write the implementation**

Create `services/atlas-channel/atlas.com/channel/character/combo/eligibility.go`:

```go
package combo

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// ComboAbilityId picks the Combo Ability id the character's job uses. The
// client's own selector is `job != 2000 ? 21000000 : 20000017`, verified in
// CMob::OnHit and ClearCombo on every in-scope version (task-217 design.md
// §2.4). Neither id is on tools/skill-job-id-guard.sh's version-divergent
// list, so the direct comparison here is permitted.
func ComboAbilityId(jobId job.Id) skill.Id {
	if jobId == job.LegendId {
		return skill.LegendComboAbilityId
	}
	return skill.AranStage1ComboAbilityId
}

// Evaluate re-derives the client's gates from authoritative state. On failure
// it names the gate that rejected ("skill", "weapon", "effect") so the caller
// can debug-log it without a second pass.
//
// There is deliberately NO job-range check: owning Combo Ability at level > 0
// IS the job gate, exactly as the client applies it. A range check would
// reject legitimate states the client accepts (design.md §3.5).
func Evaluate(c character.Model, getEffect func(skillId uint32, level byte) (effect.Model, error)) (Eligibility, string, bool) {
	comboId := ComboAbilityId(c.JobId())

	var level byte
	for _, s := range c.Skills() {
		if s.Id() == comboId {
			level = s.Level()
			break
		}
	}
	if level == 0 {
		return Eligibility{}, "skill", false
	}

	s, ok := c.Equipment().Get("weapon")
	if !ok || s.Equipable == nil {
		return Eligibility{}, "weapon", false
	}
	if item.GetWeaponType(item.Id(s.Equipable.TemplateId())) != item.WeaponTypePolearm {
		return Eligibility{}, "weapon", false
	}

	e, err := getEffect(uint32(comboId), level)
	if err != nil {
		return Eligibility{}, "effect", false
	}

	return NewEligibility(comboId, level, int32(e.X())), "", true
}
```

If `c.Skills()` elements expose the skill id under a different accessor name, or `Equipment().Get` returns a differently-shaped slot, match `equippedWeapon` in `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go:191-197` — it is the canonical reader for the equipped weapon.

- [x] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./character/combo/ -v && go vet ./character/combo/
```

Expected: PASS, every subtest — including all seven gate cases and the effect-failure case.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/combo/
git commit -m "feat(task-217): server-side Aran combo eligibility gates"
```

---

### Task 7: `ARAN_COMBO_COUNTER` handler and wiring

The hot path: one mutex-guarded map read/write plus one socket write in steady state. Kafka is touched once per combo *chain* (the seed), never per hit.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (handler map row ~line 913; writer-name slice ~line 793)

**Interfaces:**
- Consumes: `serverbound.AranComboCounterHandle`, `serverbound.AranComboCounterRequest` (Task 1); `clientbound.ShowComboWriter`, `clientbound.NewShowCombo` (Task 2); `combo.GetMirror`, `combo.Evaluate`, `combo.DefaultIdleWindow`, `combo.NewEligibility` (Tasks 5–6).
- Produces: `handler.AranComboCounterHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{})`, and the unexported seam `aranComboDeps` used by the test.

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo_test.go`. Model the dependency-seam style on `character_attack_combo_test.go` (read it first for the deps/spy conventions used in this package):

```go
package handler

import (
	"testing"
	"time"
)

func TestIdleWindowFromOptions(t *testing.T) {
	tests := []struct {
		name string
		opts map[string]interface{}
		want time.Duration
	}{
		{"v95 5000", map[string]interface{}{"idleResetMs": float64(5000)}, 5000 * time.Millisecond},
		{"v83 3000", map[string]interface{}{"idleResetMs": float64(3000)}, 3000 * time.Millisecond},
		{"int form", map[string]interface{}{"idleResetMs": 3000}, 3000 * time.Millisecond},
		{"absent", map[string]interface{}{}, 3000 * time.Millisecond},
		{"nil options", nil, 3000 * time.Millisecond},
		{"non-numeric", map[string]interface{}{"idleResetMs": "soon"}, 3000 * time.Millisecond},
		{"zero", map[string]interface{}{"idleResetMs": float64(0)}, 3000 * time.Millisecond},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := idleWindowFromOptions(tc.opts); got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}
```

Add a second test to the same file, extending its import block with `context`, `errors`, `atlas-channel/character/combo`, `github.com/sirupsen/logrus/hooks/test`, `skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"`, and `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`. It drives the increment decision directly: the socket closure is split so the decision lives in `aranComboAdvance(l, deps, t, characterId, f, options)`, testable without a live session write. `aranComboDeps` is the seam:

```go
type aranComboDeps struct {
	eligibility func(characterId uint32) (combo.Eligibility, bool)
	seed        func(el combo.Eligibility, characterId uint32) error
	announce    func(count uint32) error
	now         func() time.Time
}
```

```go
func TestAranComboAdvance(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn := comboTestTenant(t)
	f := comboTestField()
	el := combo.NewEligibility(skill3.AranStage1ComboAbilityId, 5, 5)
	now := time.Unix(2000, 0)
	opts := map[string]interface{}{"idleResetMs": float64(3000)}

	newDeps := func(eligible bool, seedErr error, seeds *int, announced *[]uint32) aranComboDeps {
		return aranComboDeps{
			eligibility: func(uint32) (combo.Eligibility, bool) { return el, eligible },
			seed: func(combo.Eligibility, uint32) error {
				*seeds++
				return seedErr
			},
			announce: func(c uint32) error {
				*announced = append(*announced, c)
				return nil
			},
			now: func() time.Time { return now },
		}
	}

	t.Run("ineligible is a silent no-op", func(t *testing.T) {
		seeds, announced := 0, []uint32{}
		aranComboAdvance(l, newDeps(false, nil, &seeds, &announced), tn, 11, f, opts)
		if seeds != 0 || len(announced) != 0 {
			t.Fatalf("want no emissions, got seeds=%d announced=%v", seeds, announced)
		}
	})

	t.Run("first hit seeds and announces 1", func(t *testing.T) {
		seeds, announced := 0, []uint32{}
		d := newDeps(true, nil, &seeds, &announced)
		aranComboAdvance(l, d, tn, 12, f, opts)
		if seeds != 1 {
			t.Errorf("want exactly 1 seed, got %d", seeds)
		}
		if len(announced) != 1 || announced[0] != 1 {
			t.Errorf("want announce [1], got %v", announced)
		}
	})

	t.Run("second hit advances without re-seeding", func(t *testing.T) {
		seeds, announced := 0, []uint32{}
		d := newDeps(true, nil, &seeds, &announced)
		aranComboAdvance(l, d, tn, 13, f, opts)
		aranComboAdvance(l, d, tn, 13, f, opts)
		if seeds != 1 {
			t.Errorf("want exactly 1 seed across two hits, got %d", seeds)
		}
		if len(announced) != 2 || announced[1] != 2 {
			t.Errorf("want announce [1 2], got %v", announced)
		}
	})

	t.Run("seed failure still advances the count", func(t *testing.T) {
		seeds, announced := 0, []uint32{}
		d := newDeps(true, errors.New("broker down"), &seeds, &announced)
		aranComboAdvance(l, d, tn, 14, f, opts)
		if len(announced) != 1 || announced[0] != 1 {
			t.Errorf("combo bookkeeping never fails the action: want announce [1], got %v", announced)
		}
	})

	t.Run("at cap the count holds and no second seed fires", func(t *testing.T) {
		seeds, announced := 0, []uint32{}
		d := newDeps(true, nil, &seeds, &announced)
		aranComboAdvance(l, d, tn, 15, f, opts)
		for i := 0; i < 3; i++ {
			aranComboAdvance(l, d, tn, 15, f, opts)
		}
		if seeds != 1 {
			t.Errorf("want exactly 1 seed, got %d", seeds)
		}
		if len(announced) != 4 || announced[3] != 4 {
			t.Errorf("want announce [1 2 3 4], got %v", announced)
		}
	})
}
```

Write `comboTestTenant` (a fresh `tenant.Create(uuid.New(), "GMS", 83, 1)` per call so the process-wide mirror singleton cannot leak state between subtests — each subtest also uses a distinct character id) and `comboTestField` (`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()`) in this file, or reuse them if the handler package already has equivalents.

The cap case above exercises the clamp indirectly through repeated increments; the exhaustive clamp-at-99999 assertion lives in Task 5's `TestIncrementClampsAtCap`.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run AranCombo -v
```

Expected: FAIL — `undefined: idleWindowFromOptions`, `undefined: aranComboAdvance`.

- [x] **Step 3: Write the implementation**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo.go`:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/combo"
	skill2 "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect/statup"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// eligibilityTTL bounds how long a cached gate result is trusted. The attack
// pipeline refreshes it on every melee hit, so this only covers the cold case
// (a fresh login whose first combo packet beats the attack hook) and a
// modified client sending the op without attacking. Staleness is benign: an
// unequip stops the real client sending at all (task-217 design.md §3.5).
const eligibilityTTL = 60 * time.Second

// idleWindowFromOptions resolves the tenant's configured idle window. The
// client's own ClearCombo timer is 3000 ms on v83/v84/v87/v92/jms185 and
// 5000 ms on v95; the server decay must AGREE with it rather than drive it,
// so the value is tenant configuration instead of a compiled major-version
// branch (task-217 design.md §2.5, §4).
func idleWindowFromOptions(options map[string]interface{}) time.Duration {
	if options == nil {
		return combo.DefaultIdleWindow
	}
	raw, ok := options["idleResetMs"]
	if !ok {
		return combo.DefaultIdleWindow
	}
	var ms float64
	switch v := raw.(type) {
	case float64:
		ms = v
	case int:
		ms = float64(v)
	case int64:
		ms = float64(v)
	default:
		return combo.DefaultIdleWindow
	}
	if ms <= 0 {
		return combo.DefaultIdleWindow
	}
	return time.Duration(ms) * time.Millisecond
}

// aranComboDeps groups the side-effecting lookups the increment path needs so
// the decision is unit-testable without a live session or broker. Mirrors the
// comboOrbDeps / comboOrbProductionDeps split in character_attack_combo.go.
type aranComboDeps struct {
	eligibility func(characterId uint32) (combo.Eligibility, bool)
	seed        func(el combo.Eligibility, characterId uint32) error
	announce    func(count uint32) error
	now         func() time.Time
}

// aranComboProductionDeps wires aranComboDeps to the real cache-then-fetch
// eligibility lookup, the Combo Ability seed emit, and the SHOW_COMBO write.
func aranComboProductionDeps(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) aranComboDeps {
	t := tenant.MustFromContext(ctx)
	return aranComboDeps{
		eligibility: func(characterId uint32) (combo.Eligibility, bool) {
			if el, ok := combo.GetMirror().Eligibility(t, characterId, time.Now(), eligibilityTTL); ok {
				return el, true
			}
			// Cold start: the attack hook has not run for this character yet
			// (fresh login whose first combo packet beats the attack). One
			// triple-decorator fetch, then cached for eligibilityTTL.
			cp := character.NewProcessor(l, ctx)
			c, err := cp.GetById(cp.InventoryDecorator, cp.SkillModelDecorator)(characterId)
			if err != nil {
				l.WithError(err).Debugf("Aran combo: character [%d] fetch failed; ignoring increment.", characterId)
				return combo.Eligibility{}, false
			}
			el, gate, ok := combo.Evaluate(c, skill2.NewProcessor(l, ctx).GetEffect)
			if !ok {
				l.Debugf("Aran combo: character [%d] rejected at gate [%s].", characterId, gate)
				return combo.Eligibility{}, false
			}
			combo.GetMirror().SetEligibility(t, characterId, s.Field(), el, time.Now())
			return el, true
		},
		seed: func(el combo.Eligibility, characterId uint32) error {
			ups := []statup.Model{statup.NewModel(string(constants.TemporaryStatTypeAranCombo), el.StatAmount())}
			return buff.NewProcessor(l, ctx).ApplyNoExpiry(s.Field(), characterId, int32(el.ComboId()), el.ComboLevel(), ups)(characterId)
		},
		announce: func(count uint32) error {
			return session.Announce(l)(ctx)(wp)(charcb.ShowComboWriter)(charcb.NewShowCombo(count).Encode)(s)
		},
		now: time.Now,
	}
}

// aranComboAdvance advances the counter for one accepted increment request.
//
// The Combo Ability buff is seeded exactly once per combo chain -- it carries
// the icon and the ARAN_COMBO damage-calc stat, never the count -- and a seed
// failure is logged and swallowed so the counter still advances (NFR-4).
//
// At the cap, Increment still refreshes lastHit (a player pinned at the cap
// must not decay while they are still hitting) and returns the unchanged
// count, so the client still gets one response per request.
func aranComboAdvance(l logrus.FieldLogger, deps aranComboDeps, t tenant.Model, characterId uint32, f field.Model, options map[string]interface{}) {
	el, ok := deps.eligibility(characterId)
	if !ok {
		return
	}
	now := deps.now()
	count, seeded := combo.GetMirror().Increment(t, characterId, f, idleWindowFromOptions(options), now)
	if seeded {
		if err := deps.seed(el, characterId); err != nil {
			l.WithError(err).Errorf("Aran combo: Combo Ability seed emit failed for character [%d].", characterId)
		}
	}
	if count == 0 {
		return
	}
	l.Debugf("Aran combo: character [%d] count [%d].", characterId, count)
	if err := deps.announce(uint32(count)); err != nil {
		l.WithError(err).Errorf("Aran combo: SHOW_COMBO write failed for character [%d].", characterId)
	}
}

// AranComboCounterHandleFunc advances the Aran combo counter for one client
// increment request. The packet carries NO body (CUserLocal::RequestIncCombo
// is a guard plus COutPacket plus SendPacket), so every gate the client
// applied is re-derived here from authoritative state.
//
// Steady-state cost is one mutex-guarded map read/write plus one socket
// write: the count lives in the channel-local mirror, not on a buff stat, so
// no Kafka round trip stands between the request and its response
// (task-217 design.md §3.3, §5.1).
func AranComboCounterHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.AranComboCounterRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		aranComboAdvance(l, aranComboProductionDeps(l, ctx, wp, s), tenant.MustFromContext(ctx), s.CharacterId(), s.Field(), readerOptions)
	}
}
```

Add the `field` import (`github.com/Chronicle20/atlas/libs/atlas-constants/field`).

- [x] **Step 4: Wire the handler and writer in `main.go`**

In `services/atlas-channel/atlas.com/channel/main.go`:

```go
// beside handlerMap[charsb.CharacterBuffCancelHandle] (~line 913)
handlerMap[charsb.AranComboCounterHandle] = handler.AranComboCounterHandleFunc
```

```go
// in the writer-name slice, beside charcb.CharacterHintWriter (~line 793)
charcb.ShowComboWriter,
```

- [x] **Step 5: Run tests and build to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./socket/handler/ -run AranCombo -v && go vet ./...
```

Expected: build clean; all `AranCombo` subtests PASS.

- [x] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo.go services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo_test.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-217): ARAN_COMBO_COUNTER handler and SHOW_COMBO wiring"
```

---

### Task 8: Attack-pipeline eligibility refresh

The melee attack pipeline already fetches the character with **both** `InventoryDecorator` and `SkillModelDecorator` (`character_attack_common.go:745`) and already calls `comboOrbTryUpdate` at line 981. Hanging the refresh there costs no extra fetch and is inherently fresh — the only legitimate way to gain combo is to land a melee hit (design.md §3.5).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (beside the `comboOrbTryUpdate` call, ~line 981)
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo_refresh_test.go`

**Interfaces:**
- Consumes: `combo.Evaluate`, `combo.GetMirror` (Tasks 5–6).
- Produces: `func aranComboRefreshEligibility(l logrus.FieldLogger, ctx context.Context, f field.Model, c character.Model, getEffect func(skillId uint32, level byte) (effect.Model, error))`.

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo_refresh_test.go`:

This test reuses `buildCharacter` from Task 6 — that helper lives in the `combo` package's tests, so re-declare an equivalent here in the `handler` package (the handler package already has `comboTestCharacter` at `character_attack_combo_test.go:122`; extend the local helper set rather than exporting a test helper across packages).

```go
package handler

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"

	"atlas-channel/character/combo"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// An eligible Aran's gate result lands in the mirror when the attack pipeline
// runs, so the ARAN_COMBO_COUNTER packets that follow cost zero REST calls.
func TestAranComboRefreshCachesEligibility(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn := comboTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	c := aranTestCharacter(t, 21, job.AranStage1Id, skill3.AranStage1ComboAbilityId, 5, aranTestPolearmId)

	aranComboRefreshEligibility(l, ctx, comboTestField(), c, aranTestEffectLookup(t, 5))

	el, ok := combo.GetMirror().Eligibility(tn, 21, time.Now(), time.Minute)
	if !ok {
		t.Fatal("want a cached eligibility after the refresh, got none")
	}
	if el.ComboId() != skill3.AranStage1ComboAbilityId || el.ComboLevel() != 5 || el.StatAmount() != 5 {
		t.Errorf("cached eligibility = (%d,%d,%d), want (%d,5,5)",
			el.ComboId(), el.ComboLevel(), el.StatAmount(), skill3.AranStage1ComboAbilityId)
	}
}

// An ineligible character leaves no entry behind: a stale-eligible cache
// would let a modified client keep incrementing after unequipping.
func TestAranComboRefreshClearsIneligible(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn := comboTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	eligible := aranTestCharacter(t, 22, job.AranStage1Id, skill3.AranStage1ComboAbilityId, 5, aranTestPolearmId)
	aranComboRefreshEligibility(l, ctx, comboTestField(), eligible, aranTestEffectLookup(t, 5))

	swapped := aranTestCharacter(t, 22, job.AranStage1Id, skill3.AranStage1ComboAbilityId, 5, aranTestSwordId)
	aranComboRefreshEligibility(l, ctx, comboTestField(), swapped, aranTestEffectLookup(t, 5))

	if _, ok := combo.GetMirror().Eligibility(tn, 22, time.Now(), time.Minute); ok {
		t.Error("swapping to a non-polearm must clear the cached eligibility")
	}
}

// The refresh must never advance the count -- only ARAN_COMBO_COUNTER does.
func TestAranComboRefreshDoesNotIncrement(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn := comboTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	c := aranTestCharacter(t, 23, job.AranStage1Id, skill3.AranStage1ComboAbilityId, 5, aranTestPolearmId)

	aranComboRefreshEligibility(l, ctx, comboTestField(), c, aranTestEffectLookup(t, 5))
	aranComboRefreshEligibility(l, ctx, comboTestField(), c, aranTestEffectLookup(t, 5))

	count, seeded := combo.GetMirror().Increment(tn, 23, comboTestField(), combo.DefaultIdleWindow, time.Now())
	if count != 1 || !seeded {
		t.Fatalf("refresh must not advance the count: want (1,true) on the first increment, got (%d,%v)", count, seeded)
	}
}
```

Add to this file the local helpers it needs: `aranTestCharacter(t, id, jobId, skillId, level, weaponTemplateId)` (same body as Task 6's `buildCharacter`, plus `SetId(id)`), `aranTestEffectLookup(t, x)` (delegating to the existing `comboTestEffect` at `character_attack_combo_test.go:136`), and the constants `aranTestPolearmId = uint32(1442000)` / `aranTestSwordId = uint32(1302000)`. `comboTestTenant` and `comboTestField` come from Task 7's test file, in the same package. Each test uses a distinct character id so the process-wide mirror singleton cannot leak state between them.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run AranComboRefresh -v
```

Expected: FAIL — `undefined: aranComboRefreshEligibility`.

- [x] **Step 3: Write the implementation**

Add to `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo.go`:

```go
// aranComboRefreshEligibility caches the Aran combo gate result off the back
// of a melee attack the server already paid to fetch (character +
// InventoryDecorator + SkillModelDecorator), so the ARAN_COMBO_COUNTER
// packets that follow -- one per damaging hit -- cost zero REST calls
// (task-217 NFR-1, design.md §3.5).
//
// An ineligible character is cleared rather than cached, so unequipping a
// polearm cannot leave a stale-eligible entry behind for a modified client.
func aranComboRefreshEligibility(l logrus.FieldLogger, ctx context.Context, f field.Model, c character.Model, getEffect func(skillId uint32, level byte) (effect.Model, error)) {
	t := tenant.MustFromContext(ctx)
	el, gate, ok := combo.Evaluate(c, getEffect)
	if !ok {
		l.Debugf("Aran combo: character [%d] not combo-eligible at gate [%s]; clearing.", c.Id(), gate)
		combo.GetMirror().Clear(t, c.Id())
		return
	}
	combo.GetMirror().SetEligibility(t, c.Id(), f, el, time.Now())
}
```

Add the `field` and `effect` imports.

- [x] **Step 4: Hook it into the attack pipeline**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`, immediately after the existing `comboOrbTryUpdate` call inside the `ai.AttackType() == packetmodel.AttackTypeMelee` block (~line 981):

```go
					if ai.AttackType() == packetmodel.AttackTypeMelee {
						comboOrbTryUpdate(l, c, ai, comboOrbProductionDeps(l, ctx, s.Field(), s.CharacterId()))
						// Aran combo eligibility rides the same fetch: the
						// client sends ARAN_COMBO_COUNTER from CMob::OnHit at
						// melee-hit frequency, and this keeps that path free
						// of REST (task-217 design.md §3.5).
						aranComboRefreshEligibility(l, ctx, s.Field(), c, skill2.NewProcessor(l, ctx).GetEffect)
					}
```

Confirm the `skill2` import alias in this file matches `character_attack_combo.go`'s (`skill2 "atlas-channel/data/skill"`); if `character_attack_common.go` uses a different alias for the same package, use its alias.

- [x] **Step 5: Run tests and build to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./socket/handler/ -v 2>&1 | tail -20 && go vet ./...
```

Expected: build clean; the whole handler package's tests PASS (the pre-existing attack tests must stay green).

- [x] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/
git commit -m "feat(task-217): refresh Aran combo eligibility from the attack pipeline"
```

---

### Task 9: Reset paths

Three inputs converge on "drop the mirror entry" (design.md §3.4). The client's own `ClearCombo` sends an ordinary `CANCEL_BUFF` for the Combo Ability id — which also self-heals the out-of-scope combo-consuming skills, since `DoActiveSkill` calls `ClearCombo` when Combo Smash / Fenrir / Tempest fire.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_buff_cancel.go`
- Modify: `services/atlas-channel/atlas.com/channel/session/processor.go` (`Destroy`, ~line 406)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go` (`handleStatusEventMapChanged`, ~line 226)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo_reset_test.go`

**Interfaces:**
- Consumes: `combo.GetMirror`, `combo.ComboAbilityId`.
- Produces: `func aranComboClearOnCancel(ctx context.Context, characterId uint32, cancelledSkillId skill.Id) bool` (reports whether it cleared, for the test), and `func clearAranComboOnDestroy(ctx context.Context, characterId uint32)` in the `session` package.

**Design decision, fixed here:** the predicate takes **no job id**. `character_buff_cancel.go` runs for *every* buff cancel and does not fetch the character, so requiring the job would add a REST call to a hot, unrelated path. Instead the predicate matches either Combo Ability id. This is safe: a character can only hold one of them (they belong to disjoint job branches), so the id itself identifies the branch. There is exactly one predicate — do not add a second, job-aware variant.

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo_reset_test.go`:

```go
package handler

import (
	"context"
	"testing"
	"time"

	"atlas-channel/character/combo"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestAranComboClearOnCancel(t *testing.T) {
	tests := []struct {
		name      string
		cancelled skill3.Id
		want      bool
	}{
		{"combo ability cancel clears", skill3.AranStage1ComboAbilityId, true},
		{"legend variant cancel clears", skill3.LegendComboAbilityId, true},
		{"unrelated buff cancel does not clear", skill3.AranStage4ComboBarrierId, false},
	}
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tn := comboTestTenant(t)
			ctx := tenant.WithContext(context.Background(), tn)
			characterId := uint32(30 + i)

			combo.GetMirror().SetEligibility(tn, characterId, comboTestField(),
				combo.NewEligibility(skill3.AranStage1ComboAbilityId, 5, 5), time.Now())
			combo.GetMirror().Increment(tn, characterId, comboTestField(), combo.DefaultIdleWindow, time.Now())

			if got := aranComboClearOnCancel(ctx, characterId, tc.cancelled); got != tc.want {
				t.Fatalf("cleared: want %v, got %v", tc.want, got)
			}

			_, present := combo.GetMirror().Eligibility(tn, characterId, time.Now(), time.Minute)
			if tc.want && present {
				t.Error("entry survived a Combo Ability cancel")
			}
			if !tc.want && !present {
				t.Error("entry was dropped by an unrelated cancel")
			}
		})
	}
}
```

Verify `skill3.AranStage4ComboBarrierId` exists (`grep -n AranStage4ComboBarrierId libs/atlas-constants/skill/constants.go`) and substitute another Aran buff id if not.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run AranComboClearOnCancel -v
```

Expected: FAIL — `undefined: aranComboClearOnCancel`.

- [x] **Step 3: Implement the cancel branch**

Add to `services/atlas-channel/atlas.com/channel/socket/handler/character_aran_combo.go`:

```go
// aranComboClearOnCancel resets the server's combo when the client cancels
// the Combo Ability buff. The client's ClearCombo -- fired by its own idle
// timer AND by DoActiveSkill when a combo-consuming skill spends the count --
// calls SendSkillCancelRequest for the Combo Ability id, so this single
// branch keeps the two counters from ever drifting (task-217 design.md §3.4).
// It matches either Combo Ability id rather than selecting by job: this path
// runs for EVERY buff cancel and does not fetch the character, and a
// character can only ever hold one of the two ids (disjoint job branches), so
// the id alone identifies the branch.
func aranComboClearOnCancel(ctx context.Context, characterId uint32, cancelledSkillId skill3.Id) bool {
	if cancelledSkillId != skill3.AranStage1ComboAbilityId && cancelledSkillId != skill3.LegendComboAbilityId {
		return false
	}
	combo.GetMirror().Clear(tenant.MustFromContext(ctx), characterId)
	return true
}
```

Wire it in `character_buff_cancel.go` immediately after the existing `_ = buff.NewProcessor(l, ctx).Cancel(...)` call — before the `set.Skill.Resolve` block, so it runs regardless of whether the id resolves to a known Identity:

```go
		// The Aran combo counter resets whenever the client cancels Combo
		// Ability: the client's own 3s/5s idle timer AND every
		// combo-consuming skill route through ClearCombo ->
		// SendSkillCancelRequest (task-217 design.md §3.4).
		aranComboClearOnCancel(ctx, s.CharacterId(), skill3.Id(p.SkillId()))
```

Use the file's existing alias for `libs/atlas-constants/skill` (it currently imports it as `skill`); if so, write `skill.Id(p.SkillId())` and name the constants `skill.AranStage1ComboAbilityId` / `skill.LegendComboAbilityId` in both the predicate and the call, keeping one alias throughout. Adjust the cast to whatever `BuffCancelRequest.SkillId()` returns.

- [x] **Step 4: Implement the session-destroy and map-change clears**

In `services/atlas-channel/atlas.com/channel/session/processor.go`, beside the existing `clearBattleshipOnDestroy` call in `Destroy`:

```go
	// The Aran combo counter cannot outlive the session: logout, disconnect,
	// timeout, and channel change all funnel here (task-217 design.md §3.4).
	clearAranComboOnDestroy(p.ctx, s.CharacterId())
```

and beside `clearBattleshipOnDestroy`'s definition:

```go
// clearAranComboOnDestroy drops any live Aran combo state for a destroyed
// session's character. Extracted from Destroy so the invariant is unit
// testable without exercising Destroy's Kafka emit path, mirroring
// clearBattleshipOnDestroy.
func clearAranComboOnDestroy(ctx context.Context, characterId uint32) {
	if characterId != 0 {
		combo.GetMirror().Clear(tenant.MustFromContext(ctx), characterId)
	}
}
```

In `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go`, inside `handleStatusEventMapChanged` after its existing type guard:

```go
		// A map change ends the combat that built the combo; leaving the
		// count alive would resurrect a counter in the new map (task-217
		// design.md §3.4).
		combo.GetMirror().Clear(tenant.MustFromContext(ctx), event.CharacterId)
```

Use the event's actual character-id accessor (read the surrounding lines — it may be `event.CharacterId` or a body field).

Before writing either, check for an import cycle: `grep -rn "atlas-channel/session" services/atlas-channel/atlas.com/channel/character/combo/`. It must be empty — `combo` must not import `session`.

- [x] **Step 5: Run tests and build to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... 2>&1 | tail -30 && go vet ./...
```

Expected: build clean; the whole module's tests PASS, including the pre-existing `session` battleship-hook tests.

- [x] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(task-217): reset Aran combo on cancel, session destroy, and map change"
```

---

### Task 10: Decay tick

A 1 Hz task that walks the mirror — never a session or tenant scan, so an empty mirror is an empty walk (FR-4.5). It cancels the Combo Ability buff per expired entry and **sends no packet**: `SHOW_COMBO 0` cannot clear the client's HUD, and the client clears itself on the same window.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/combo/task.go`
- Test: `services/atlas-channel/atlas.com/channel/character/combo/task_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (~line 326, beside the heartbeat registration)

**Interfaces:**
- Consumes: `Mirror.ExpireIdle` (Task 5).
- Produces: `func NewDecayTick(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *DecayTick` with `Run()` and `SleepTime() time.Duration`, and the testable seam `func processExpiries(l logrus.FieldLogger, ctx context.Context, expired []Expired, cancel func(l logrus.FieldLogger, ctx context.Context, e Expired) error) int`.

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/character/combo/task_test.go`:

```go
package combo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestProcessExpiriesCancelsOncePerEntry(t *testing.T) {
	l, _ := test.NewNullLogger() // logrus/hooks/test
	tn := testTenant(t)
	expired := []Expired{
		{t: tn, characterId: 1, f: testField(), comboId: skill.AranStage1ComboAbilityId},
		{t: tn, characterId: 2, f: testField(), comboId: skill.LegendComboAbilityId},
	}
	var seen []uint32
	n := processExpiries(l, context.Background(), expired, func(_ logrus.FieldLogger, _ context.Context, e Expired) error {
		seen = append(seen, e.CharacterId())
		return nil
	})
	if n != 2 || len(seen) != 2 {
		t.Fatalf("want 2 cancels, got n=%d seen=%v", n, seen)
	}
}

func TestProcessExpiriesSwallowsCancelFailure(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn := testTenant(t)
	expired := []Expired{{t: tn, characterId: 1, f: testField(), comboId: skill.AranStage1ComboAbilityId}}
	n := processExpiries(l, context.Background(), expired, func(_ logrus.FieldLogger, _ context.Context, _ Expired) error {
		return errors.New("broker down")
	})
	if n != 0 {
		t.Errorf("failed cancels are not counted as processed: want 0, got %d", n)
	}
}

func TestProcessExpiriesEmptySweepDoesNothing(t *testing.T) {
	l, _ := test.NewNullLogger()
	called := false
	n := processExpiries(l, context.Background(), nil, func(_ logrus.FieldLogger, _ context.Context, _ Expired) error {
		called = true
		return nil
	})
	if n != 0 || called {
		t.Errorf("empty sweep must emit nothing: n=%d called=%v", n, called)
	}
}

func TestDecayTickSleepTime(t *testing.T) {
	tick := NewDecayTick(logrus.New(), context.Background(), time.Second)
	if got := tick.SleepTime(); got != time.Second {
		t.Errorf("want 1s, got %v", got)
	}
}
```

Add the missing imports (`github.com/sirupsen/logrus/hooks/test`, the skill constants package) and reuse `testTenant`/`testField` from `mirror_test.go`. Confirm the null-logger helper this repo uses by grepping an existing test that needs a logger (`grep -rn "NewNullLogger" services/atlas-channel/atlas.com/channel | head -3`) and match it.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./character/combo/ -run "Expiries|DecayTick" -v
```

Expected: FAIL — `undefined: processExpiries`, `undefined: NewDecayTick`.

- [x] **Step 3: Write the implementation**

Create `services/atlas-channel/atlas.com/channel/character/combo/task.go`:

```go
package combo

import (
	"atlas-channel/character/buff"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// DecayTick expires idle Aran combos.
//
// It walks the combo mirror only -- never sessions, never a tenant list -- so
// a channel with no Aran in combat does no work per tick (task-217 FR-4.5).
// It deliberately sends NO packet for an expiry: DrawCombo early-returns on a
// non-positive count without releasing its digit layers, so SHOW_COMBO 0
// cannot clear the HUD. The client runs the same idle timer and clears itself
// (design.md §2.5, §5.3); the server's job is to agree with it.
type DecayTick struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
}

func NewDecayTick(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *DecayTick {
	return &DecayTick{l: l, ctx: ctx, interval: interval}
}

func (r *DecayTick) SleepTime() time.Duration {
	return r.interval
}

func (r *DecayTick) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(r.ctx, "aran_combo_decay_tick")
	defer span.End()

	expired := GetMirror().ExpireIdle(time.Now())
	if len(expired) == 0 {
		return
	}
	processExpiries(r.l, ctx, expired, cancelComboBuff)
}

// cancelComboBuff drops the Combo Ability buff for one expired combo. The
// tenant-scoped context is built by processExpiries before this is called --
// the tick has no request context of its own, the same shape
// ProcessPoisonTicks uses in atlas-buffs.
func cancelComboBuff(l logrus.FieldLogger, ctx context.Context, e Expired) error {
	return buff.NewProcessor(l, ctx).Cancel(e.Field(), e.CharacterId(), int32(e.ComboId()))
}

// processExpiries cancels the Combo Ability buff for each expired combo and
// returns how many cancels succeeded. Failures are logged and swallowed:
// combo bookkeeping never fails a player action (task-217 NFR-4), and the
// next sweep will not retry because the count is already zero -- an orphaned
// buff icon is strictly better than a stalled tick.
func processExpiries(l logrus.FieldLogger, ctx context.Context, expired []Expired, cancel func(l logrus.FieldLogger, ctx context.Context, e Expired) error) int {
	n := 0
	for _, e := range expired {
		tctx := tenant.WithContext(ctx, e.Tenant())
		if err := cancel(l, tctx, e); err != nil {
			l.WithError(err).Errorf("Aran combo: decay cancel emit failed for character [%d].", e.CharacterId())
			continue
		}
		l.Debugf("Aran combo: character [%d] decayed to zero; Combo Ability cancelled.", e.CharacterId())
		n++
	}
	return n
}
```

- [x] **Step 4: Register the tick in `main.go`**

Beside the existing heartbeat registration at `services/atlas-channel/atlas.com/channel/main.go:326`, following that line's exact `tasks.Register` shape:

```go
		tasks.Register(l, rt.Context())(combo.NewDecayTick(l, rt.Context(), time.Second))
```

Add the `atlas-channel/character/combo` import. `tasks.Register` already spawns through `routine.Go`, so no bare `go` statement is introduced.

- [x] **Step 5: Run tests, build, and guards to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./character/combo/ -v && go vet ./...
cd ../../../.. && tools/goroutine-guard.sh && tools/redis-key-guard.sh && tools/buff-duration-guard.sh
```

Expected: build clean; all combo tests PASS; all three guards exit 0.

- [x] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/combo/ services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-217): Aran combo idle decay tick"
```

---

### Task 11: Seed templates (six versions)

One handler entry and one writer entry per template, each at its **sorted `opCode` position** — never appended beside a semantically-related entry. The handler carries `idleResetMs`: `5000` for v95, `3000` for the other five.

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

**Interfaces:**
- Consumes: the handler name `"AranComboCounterHandle"` (Task 1) and the writer name `"ShowCombo"` (Task 2). **These strings must match the Go constants exactly** — a mismatch is a silent drop, not a build error.
- Produces: routed opcodes for the six in-scope versions.

- [x] **Step 1: Add the entries, per file, at their sorted positions**

Edit each file individually (per the project's shell conventions — no batch patch loop). Values per version:

| File | handler `opCode` | `idleResetMs` | writer `opCode` |
|---|---|---|---|
| `template_gms_83_1.json` | `0x0A3` | `3000` | `0x0E1` |
| `template_gms_84_1.json` | `0x0A9` | `3000` | `0x0E6` |
| `template_gms_87_1.json` | `0x0AD` | `3000` | `0x0EF` |
| `template_gms_92_1.json` | `0x0BA` | `3000` | `0x103` |
| `template_gms_95_1.json` | `0x0BD` | **`5000`** | `0x101` |
| `template_jms_185_1.json` | `0x09D` | `3000` | `0x0EB` |

Handler entry (v83 shown; substitute the row's values), inserted into `socket.handlers` between the entries whose opcodes bracket it numerically:

```json
    {
      "opCode": "0x0A3",
      "validator": "LoggedInValidator",
      "handler": "AranComboCounterHandle",
      "fname": "CUserLocal::RequestIncCombo",
      "options": {
        "idleResetMs": 3000
      },
      "services": [
        "channel"
      ]
    },
```

Writer entry (v83 shown), inserted into `socket.writers` at its sorted position:

```json
    {
      "opCode": "0x0E1",
      "writer": "ShowCombo",
      "fname": "CUserLocal::OnIncComboResponse",
      "services": [
        "channel"
      ]
    },
```

Match each file's existing indentation and key ordering exactly. Confirm the `validator` name `LoggedInValidator` exists in that template (`grep -c LoggedInValidator <file>`) — an unknown or empty validator silently drops the handler.

- [x] **Step 2: Verify JSON validity and sorted insertion**

```bash
cd services/atlas-configurations/seed-data/templates
for f in template_gms_83_1.json template_gms_84_1.json template_gms_87_1.json template_gms_92_1.json template_gms_95_1.json template_jms_185_1.json; do
  python3 -c "import json,sys; json.load(open('$f')); print('$f ok')"
done
grep -c "AranComboCounterHandle" template_gms_83_1.json template_gms_84_1.json template_gms_87_1.json template_gms_92_1.json template_gms_95_1.json template_jms_185_1.json
grep -c "\"ShowCombo\"" template_gms_83_1.json template_gms_84_1.json template_gms_87_1.json template_gms_92_1.json template_gms_95_1.json template_jms_185_1.json
grep -n "idleResetMs" template_gms_95_1.json
```

Expected: six `ok` lines; every count is exactly `1`; the v95 grep shows `5000`.

- [x] **Step 3: Run the template guards**

```bash
cd ../../../.. && tools/template-opcode-order-guard.sh && tools/template-duplicate-binding-guard.sh && tools/template-movement-types-guard.sh
```

Expected: all three exit 0. A non-zero opcode-order guard means an entry landed out of sorted position — move it, do not adjust the guard.

- [x] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(task-217): route ARAN_COMBO_COUNTER and SHOW_COMBO in six templates"
```

---

### Task 12: Name the v92 sender in its IDB

`CUserLocal::RequestIncCombo` is unnamed in the v92 IDB (`sub_8EF840` @ `0x8ef840`, design.md §2.1). The project's rule is to name symbols while reversing so the next reader does not re-derive them.

**Files:** none in the repo — this mutates the IDA database only.

**Interfaces:** none.

- [x] **Step 1: Resolve the v92 session by binary name**

Use `mcp__ida-pro__idb_list` and select the row whose binary name is `GMS_v92_1_DEVM`. Pass that value as the `database` parameter to every subsequent call — port-based `select_instance` is dead.

- [x] **Step 2: Confirm the function before renaming**

Use `mcp__ida-pro__decompile` on `0x8ef840` with that `database`. Confirm it is the guard + `COutPacket(186)` + `SendPacket` shape described in design.md §2.1. If the address does not decompile to that shape, **stop and report** — do not rename a function you have not confirmed.

- [x] **Step 3: Rename and save**

Use `mcp__ida-pro__rename` to set `0x8ef840` to `CUserLocal__RequestIncCombo_send_0xBA`, then `mcp__ida-pro__idb_save`.

- [x] **Step 4: Verify the rename stuck**

Use `mcp__ida-pro__func_query` with `name_regex` `RequestIncCombo` against the v92 database. Expected: the renamed function is returned at `0x8ef840`.

No commit — nothing in the repo changed. Record the outcome in the task summary.

---

### Task 13: Promote the twelve matrix cells

Two ops × six versions. Each cell goes through the single-cell verify procedure — `/verify-packet` (or the `packet-verifier` agent) driving `docs/packets/audits/VERIFYING_A_PACKET.md`: derive the client read order, write/confirm the byte fixture with its `packet-audit:verify` marker, pin the evidence record, regenerate the matrix, commit the artifacts together. **A cell that does not promote is a failure report, not a prose claim.**

The design already did the derivation work; the IDA addresses in the fixture markers written in Tasks 1–2 are the evidence anchors:

| Cell | serverbound ida | clientbound ida |
|---|---|---|
| gms_v83 | `0x9602f3` | `0x95086b` (case 225) |
| gms_v84 | `0x99f346` | `0x988698` (case 230) |
| gms_v87 | `0x9e37bc` | `0x9cbcd4` (case 239) |
| gms_v92 | `0x8ef840` | `0x913a68` (case 259) |
| gms_v95 | `0x909070` | `0x934238` (case 257) |
| jms_v185 | `0xa2d435` | `0xa1208d` (case 235) |

**Files:** `libs/atlas-packet/character/{serverbound,clientbound}/*_test.go`, `docs/packets/audits/status.json`, `docs/packets/audits/STATUS.md`, and the evidence records the procedure writes.

- [x] **Step 1: Read the procedure**

```bash
cat docs/packets/audits/VERIFYING_A_PACKET.md
```

Follow it exactly. Do not restate or shortcut it.

- [x] **Step 2: Verify the six serverbound cells**

Run `/verify-packet` once per cell for `character/serverbound/AranComboCounter` × {gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185}, or dispatch `packet-verifier` per cell. Each must promote `ARAN_COMBO_COUNTER` from `❌` to `✅` in that version's column.

- [x] **Step 3: Verify the six clientbound cells**

Same, for `character/clientbound/ShowCombo` × the same six versions.

- [x] **Step 4: Confirm every cell promoted**

```bash
grep -n "ARAN_COMBO_COUNTER\|SHOW_COMBO" docs/packets/audits/STATUS.md
```

Expected: the `ARAN_COMBO_COUNTER` row shows `✅` under v83/v84/v87/v92/v95/jms185 (v12/v48/v61/v72/v79 stay `⬜`/`n-a`); the `SHOW_COMBO` row shows `✅` under the same six and keeps its `❌` at v79 (the op exists there but `ARAN_COMBO_COUNTER` is `n-a`, so nothing can drive it — design.md §2.2). If any cell did not promote, report it as a failure with the verifier's output; do not hand-edit `STATUS.md`.

- [x] **Step 5: Commit whatever the procedure did not already commit**

```bash
git status --short
git add -A docs/packets libs/atlas-packet
git commit -m "verify(task-217): promote ARAN_COMBO_COUNTER and SHOW_COMBO across six versions"
```

---

### Task 14: Documentation and full verification sweep

**Files:**
- Modify: `docs/TODO.md`
- Modify (only if it exists in this worktree): `docs/research/missing-features/skills-and-buffs.md` §7 and `docs/research/missing-features/new-jobs-and-version-delta.md` §5

**Interfaces:** none.

- [x] **Step 1: Update the tracking docs**

```bash
ls docs/TODO.md && grep -n -i "aran\|combo" docs/TODO.md | head -20
ls docs/research/missing-features/ 2>/dev/null
```

Update the Aran combo entry in `docs/TODO.md` to the landed state, following the /dev-docs format used by its neighbours. If `docs/research/missing-features/` exists in this worktree, update both entries and correct the version anchor — the corpus says "Aran combo counter (v84+)", but `ARAN_COMBO_COUNTER` exists from **v83** (prd.md §1.2). If the corpus is still untracked here, note that in the commit message rather than creating the files.

Also record, in the `docs/TODO.md` entry, that combo *consumption* (Combo Smash `21100004`, Combo Fenrir `21110004`, Combo Tempest `21120006`) remains out of scope — it overlaps the attack-pipeline surface and the client's own `ClearCombo` already keeps the two counts from drifting (design.md §5.5).

- [x] **Step 2: Run the full module verification**

```bash
for m in libs/atlas-packet libs/atlas-constants services/atlas-channel/atlas.com/channel services/atlas-data/atlas.com/data; do
  echo "=== $m ==="
  (cd "$m" && go build ./... && go vet ./... && go test -race ./...) || echo "FAILED: $m"
done
```

Expected: every module builds, vets, and tests clean. Quote the actual output — a failure here is a failure, not a rounding error.

- [x] **Step 3: Run every guard**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/buff-duration-guard.sh
tools/skill-job-id-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/lint.sh --check
```

Expected: all exit 0. `tools/lint.sh --check` needs nvm on PATH — if it false-fails on the atlas-ui leg, source nvm and re-run before concluding. Run `tools/lint.sh` (no flags) first if formatting drifted.

`tools/service-registration-guard.sh` is **not** required: no service was added and none of `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work`, or `tools/db-bootstrap.sh` changed. Confirm with `git diff --name-only main... | grep -E "services.json|deploy/k8s|docker-bake.hcl|go.work|db-bootstrap"` returning nothing; run the guard if it returns anything.

- [x] **Step 4: Bake every service whose `go.mod` moved**

```bash
git diff --name-only main... | grep "go.mod\|go.sum"
```

If any service's `go.mod`/`go.sum` changed (adding `libs/atlas-constants` or `libs/atlas-packet` requirements can move them), bake that service from the worktree root — this is mandatory, and `go build` against `go.work` will not catch a missing `COPY libs/...`:

```bash
docker buildx bake atlas-channel
docker buildx bake atlas-data
docker buildx bake atlas-configurations
```

Bake only the services whose `go.mod` actually moved. If none did, state that plainly and skip.

- [x] **Step 5: Commit**

```bash
git add docs/
git commit -m "docs(task-217): record landed Aran combo counter state"
```

- [x] **Step 6: Code review before the PR**

Run `superpowers:requesting-code-review` — it dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (Go changed; no atlas-ui changes, so no frontend reviewer). Findings go to `docs/tasks/task-217-aran-combo-counter/audit.md`. Pin the review subagents to Sonnet/Haiku per the project's model-cost preference. Address the findings before opening the PR — this step is not optional.

---

## Acceptance Criteria Traceability

| prd.md acceptance criterion | Task |
|---|---|
| Both opcodes IDA-verified for all six versions; registry corrections | Discharged in design.md §2.1/§2.2 (zero corrections); evidence pinned in Task 13 |
| Serverbound model with `Encode`/`Decode` | Task 1 |
| `SHOW_COMBO` model with `Encode`/`Decode` | Task 2 |
| Handler + writer in six templates, sorted, validator + fname | Task 11 |
| Twelve matrix cells `✅` with fixtures and evidence | Tasks 1, 2, 13 |
| Counter increments on screen for an eligible Aran | Tasks 5, 7 |
| First increment seeds the buff; later ones advance, clamped at cap | Tasks 5, 7 |
| Cap not exceeded however fast the client sends | Task 5 (`Increment` clamp) |
| Idle decay; buff cancelled at zero | Tasks 5, 10 |
| A qualifying hit refreshes the idle timer | Task 5 (`Increment` sets `lastHit`), Task 8 |
| Silent no-op for non-Aran / no Combo Ability / non-polearm | Tasks 6, 7 |
| `reader.go:470` statup corrected or justified | Task 4 |
| `20000017` added to atlas-constants | Task 3 |
| `go test -race` / `go vet` / `go build` clean | Task 14 |
| `docker buildx bake` for touched-`go.mod` services | Task 14 |
| All guards clean | Tasks 5, 10, 11, 14 |
| `docs/TODO.md` updated | Task 14 |
| Code review before PR | Task 14 |

**Deviations from the PRD, carried from design.md and to be reviewed as intentional:**

- **FR-3 (count on the `ARAN_COMBO` buff stat)** — rejected. The count is delivered by `SHOW_COMBO`; the stat is a signed-short damage-calc input. Storing the count there would conflate two values, truncate above 32767, and force a Kafka round trip per melee hit (design.md §5.1). The count lives in `ComboMirror` instead.
- **FR-4.1 (decay ticker in atlas-buffs)** — moved to atlas-channel, following FR-3. **atlas-buffs needs no change at all** (design.md §5.2).
- **FR-4.4 ("tell the client" on decay-to-zero)** — impossible and deliberately omitted. `DrawCombo` early-returns on a non-positive count without releasing its layers, so no clientbound packet can clear the HUD; the client clears itself on the same window (design.md §5.3).
- **FR-2.1 (explicit Aran/Legend job-range gate)** — folded into the skill gate to match the client exactly (design.md §3.5).
