# Death Items — Wheel of Destiny revive & tomb effect — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the `USE_DEATHITEM` / `SHOW_UPGRADE_TOMB_EFFECT` packet pair end to end, make the in-map wheel revive honour the client's `premium` choice with real charge accounting, and emit the existing protect-on-die effect.

**Architecture:** Three independent units, per [`design.md`](design.md) §2. **Unit A** adds two codecs to `libs/atlas-packet` (both are three/four little-endian 4-byte fields, identical on every version — no version gate). **Unit B** is a stateless relay handler in atlas-channel: validate the sender is dead and owns a wheel, then broadcast the tomb effect to the *other* sessions in the map. It consumes nothing. **Unit C** reworks `respawn/processor.go`: honour `Change.Premium()` (today a Cancel still destroys the wheel), gate on and report the asset's remaining quantity, and emit `EffectProtectOnDie`/`...Foreign` when a protective item is spent. Unit C's decision logic is extracted into a pure function so it can be tested without HTTP mocks.

**Tech Stack:** Go 1.24 workspace (`go.work`), `libs/atlas-packet` codecs (`response.Writer`/`request.Reader`), atlas-channel socket handlers + `session.Announce`, saga commands over Kafka, JSON seed templates in atlas-configurations, `tools/packet-audit` for registry/matrix.

## Global Constraints

- **No invented values.** Every opcode, address, and field order in this plan is cited from [`design.md`](design.md) §0–§1 or verified against the repo during planning. If an execution step finds a discrepancy, stop and report — do not substitute a plausible value.
- **No version gate on the two new codecs.** Every version's `RequestUpgradeTombEffect` / `OnShowUpgradeTombEffect` is byte-identical (design §1.1, §1.2). Do not add a `MajorAtLeast` call to satisfy FR-1.3; it is satisfied vacuously.
- **DOM-25 — no hard-coded wire values.** Opcodes live in the tenant templates only. The `PROTECT_ON_DIE_ITEM_USE` mode byte is resolved through `atlas_packet.WithResolvedCode("operations", …)`, never a literal.
- **DOM-21 — reuse `libs/atlas-constants`.** Use `item.WheelOfFortuneId`, `item.IsWheelOfFortune`, `item.IsSafetyCharm`, `item.IsDeathProtectionItem` from `libs/atlas-constants/item/death_protection.go`. Do not redeclare ids.
- **Immutable models.** Private fields + value-receiver accessors + a `New…` constructor. No setters.
- **Templates:** entries go at their sorted `opCode` position; handlers carry a `validator`; writers carry an `fname`. `services: ["channel"]` on both.
- **No `// TODO`, no stubs, no deferred work.** Landed commits are complete.
- **Line endings:** preserve each file's existing endings; do not normalise.
- **Commit after every task.** Conventional-commit subjects, scoped `feat(task-210)` / `fix(task-210)` / `chore(task-210)`.

**Version set for both new ops (8 versions):** `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185`. `gms_v48` and `gms_v61` are `n-a` for both (design §0 evidence index: the item id `5510000` appears nowhere in either image).

**Opcode table (design §5.2, all sixteen slots confirmed free during planning):**

| Template | `USE_DEATHITEM` handler | `SHOW_UPGRADE_TOMB_EFFECT` writer |
|---|---|---|
| `template_gms_72_1.json` | `0x34` | `0xB1` |
| `template_gms_79_1.json` | `0x33` | `0xB5` |
| `template_gms_83_1.json` | `0x35` | `0xC3` |
| `template_gms_84_1.json` | `0x35` | `0xC7` |
| `template_gms_87_1.json` | `0x38` | `0xD0` |
| `template_gms_92_1.json` | `0x3B` | `0xDF` |
| `template_gms_95_1.json` | `0x3A` | `0xDD` |
| `template_jms_185_1.json` | `0x2D` | `0xC9` |

---

## File Structure

**Created**

| File | Responsibility |
|---|---|
| `libs/atlas-packet/character/serverbound/use_death_item.go` | `UseDeathItem` codec — `itemId`, `x`, `y`. |
| `libs/atlas-packet/character/serverbound/use_death_item_test.go` | Round-trip + eight per-version byte fixtures with `packet-audit:verify` markers. |
| `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect.go` | `ShowUpgradeTombEffect` codec — `characterId`, `itemId`, `x`, `y`. |
| `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect_test.go` | Round-trip + eight per-version byte fixtures. |
| `services/atlas-channel/atlas.com/channel/socket/handler/use_death_item.go` | Unit B — validate + relay. No state change. |
| `services/atlas-channel/atlas.com/channel/socket/handler/use_death_item_test.go` | Handler validation table. |
| `services/atlas-channel/atlas.com/channel/socket/writer/show_upgrade_tomb_effect.go` | Writer body helper (only if the codebase's writer-body convention requires one — see Task 5 Step 3). |
| `services/atlas-channel/atlas.com/channel/respawn/plan.go` | Pure `planRespawn` decision function + `respawnPlan` type. |
| `services/atlas-channel/atlas.com/channel/respawn/plan_test.go` | Unit C decision tests (premium gate, charges, protection). |

**Modified**

| File | Change |
|---|---|
| `services/atlas-channel/atlas.com/channel/respawn/processor.go` | `Respawn` signature; delegate decisions to `planRespawn`; saga built from assets; protect-on-die emission. |
| `services/atlas-channel/atlas.com/channel/socket/handler/map_change.go:54` | Pass `p.Premium() != 0` and `wp` into `respawn`. |
| `services/atlas-channel/atlas.com/channel/main.go` | Register the new handler and writer. |
| `docs/packets/registry/gms_v72.yaml`, `gms_v79.yaml` | New `USE_DEATHITEM` rows. |
| `docs/packets/registry/gms_{72,79,83,84,87,92,95}.yaml`, `jms_v185.yaml` | `packet:` field on both ops' rows. |
| `docs/packets/feature-na-evidence.yaml` | v48/v61 × both ops. |
| `services/atlas-configurations/seed-data/templates/template_gms_{72,79,83,84,87,92,95}_1.json`, `template_jms_185_1.json` | 8 handler + 8 writer entries; v92 `CharacterEffect`/`CharacterEffectForeign` entries; v95/jms185 foreign-writer rename. |
| `docs/packets/audits/STATUS.md`, `status.json` | Regenerated (never hand-edited). |

---

## Task 1: `UseDeathItem` serverbound codec

**Files:**
- Create: `libs/atlas-packet/character/serverbound/use_death_item.go`
- Create: `libs/atlas-packet/character/serverbound/use_death_item_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `const CharacterUseDeathItemHandle = "CharacterUseDeathItemHandle"`; `type UseDeathItem struct` with `NewUseDeathItem(itemId uint32, x int32, y int32) UseDeathItem`, accessors `ItemId() uint32`, `X() int32`, `Y() int32`, `Operation() string`, `String() string`, `Encode(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`, `Decode(logrus.FieldLogger, context.Context) func(*request.Reader, map[string]interface{})` (pointer receiver).

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/character/serverbound/use_death_item_test.go`:

```go
package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// UseDeathItem — CUserLocal::RequestUpgradeTombEffect. Every version encodes the
// same three little-endian 4-byte fields and differs only in the opcode:
//
//	Encode4  itemId   — hard-coded 5510000 (0x541370) by the client
//	Encode4  x        — m_ptRevive.x
//	Encode4  y        — m_ptRevive.y
//
// IDA gms_v95 CUserLocal::RequestUpgradeTombEffect@0x908320 (op 58 = 0x03A).
//
// packet-audit:verify packet=character/serverbound/UseDeathItem version=gms_v72 ida=0x867654
// packet-audit:verify packet=character/serverbound/UseDeathItem version=gms_v79 ida=0x8b2ff0
// packet-audit:verify packet=character/serverbound/UseDeathItem version=gms_v83 ida=0x95af8e
// packet-audit:verify packet=character/serverbound/UseDeathItem version=gms_v84 ida=0x999277
// packet-audit:verify packet=character/serverbound/UseDeathItem version=gms_v87 ida=0x9dd673
// packet-audit:verify packet=character/serverbound/UseDeathItem version=gms_v92 ida=0x8ee9f0
// packet-audit:verify packet=character/serverbound/UseDeathItem version=gms_v95 ida=0x908320
// packet-audit:verify packet=character/serverbound/UseDeathItem version=jms_v185 ida=0xa25fc9
func TestUseDeathItemByteOutput(t *testing.T) {
	variants := []struct {
		name   string
		region string
		major  uint16
	}{
		{"gms_v72", "GMS", 72},
		{"gms_v79", "GMS", 79},
		{"gms_v83", "GMS", 83},
		{"gms_v84", "GMS", 84},
		{"gms_v87", "GMS", 87},
		{"gms_v92", "GMS", 92},
		{"gms_v95", "GMS", 95},
		{"jms_v185", "JMS", 185},
	}
	// itemId 5510000 = 0x00541370; x = 100 (0x64); y = -200 (0xFFFFFF38).
	want := []byte{
		0x70, 0x13, 0x54, 0x00,
		0x64, 0x00, 0x00, 0x00,
		0x38, 0xFF, 0xFF, 0xFF,
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			got := NewUseDeathItem(5510000, 100, -200).Encode(nil, ctx)(nil)
			if !bytes.Equal(got, want) {
				t.Errorf("%s UseDeathItem wire: got %x want %x", v.name, got, want)
			}
		})
	}
}

func TestUseDeathItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewUseDeathItem(5510000, 1234, -5678)
			output := UseDeathItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./character/serverbound/ -run UseDeathItem -v`
Expected: FAIL — `undefined: NewUseDeathItem`, `undefined: UseDeathItem`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/character/serverbound/use_death_item.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterUseDeathItemHandle = "CharacterUseDeathItemHandle"

// UseDeathItem - CUserLocal::RequestUpgradeTombEffect
// packet-audit:fname CUserLocal::RequestUpgradeTombEffect
//
// Sent by CUIRevive::OnCreate — at death-dialog construction time, before the
// player has chosen anything — when the field allows the wheel and the player
// owns at least one. It is a request to play the tomb effect for bystanders,
// NOT a request to revive or to consume the item: the revive itself is
// CUIRevive::Revive -> CField::SendTransferFieldRequest (MAP_CHANGE). The
// client plays its own copy of the effect locally via
// CUser::ShowUpgradeTombEffect immediately after sending, so the server must
// not echo the effect back to the sender.
//
// Wire layout — identical on every version that carries the op (gms v72, v79,
// v83, v84, v87, v92, v95, jms v185); only the opcode differs, and that is
// tenant configuration. There is deliberately no version gate.
//
//	Encode4  itemId   — hard-coded 5510000 (0x541370) by the client
//	Encode4  x        — m_ptRevive.x
//	Encode4  y        — m_ptRevive.y
//
// IDA gms_v95 @0x908320, gms_v92 @0x8ee9f0, gms_v87 @0x9dd673, gms_v84
// @0x999277, gms_v83 @0x95af8e, gms_v79 @0x8b2ff0, gms_v72 @0x867654,
// jms_v185 @0xa25fc9.
type UseDeathItem struct {
	itemId uint32
	x      int32
	y      int32
}

func NewUseDeathItem(itemId uint32, x int32, y int32) UseDeathItem {
	return UseDeathItem{itemId: itemId, x: x, y: y}
}

func (m UseDeathItem) ItemId() uint32    { return m.itemId }
func (m UseDeathItem) X() int32          { return m.x }
func (m UseDeathItem) Y() int32          { return m.y }
func (m UseDeathItem) Operation() string { return CharacterUseDeathItemHandle }
func (m UseDeathItem) String() string {
	return fmt.Sprintf("itemId [%d], x [%d], y [%d]", m.itemId, m.x, m.y)
}

func (m UseDeathItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.itemId)
		w.WriteInt32(m.x)
		w.WriteInt32(m.y)
		return w.Bytes()
	}
}

func (m *UseDeathItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.itemId = r.ReadUint32()
		m.x = r.ReadInt32()
		m.y = r.ReadInt32()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./character/serverbound/ -run UseDeathItem -v`
Expected: PASS — both tests, all sub-tests.

- [ ] **Step 5: Format and vet**

Run: `cd libs/atlas-packet && go vet ./... && gofmt -l character/serverbound/`
Expected: no vet output; `gofmt -l` prints nothing.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/serverbound/use_death_item.go libs/atlas-packet/character/serverbound/use_death_item_test.go
git commit -m "feat(task-210): add USE_DEATHITEM serverbound codec"
```

---

## Task 2: `ShowUpgradeTombEffect` clientbound codec

**Files:**
- Create: `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect.go`
- Create: `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `const CharacterShowUpgradeTombEffectWriter = "CharacterShowUpgradeTombEffect"`; `type ShowUpgradeTombEffect struct` with `NewShowUpgradeTombEffect(characterId uint32, itemId uint32, x int32, y int32) ShowUpgradeTombEffect`, accessors `CharacterId() uint32`, `ItemId() uint32`, `X() int32`, `Y() int32`, plus `Operation`/`String`/`Encode`/`Decode`. The writer-name string `"CharacterShowUpgradeTombEffect"` is what the seed templates bind in Task 4.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ShowUpgradeTombEffect — CUserRemote::OnShowUpgradeTombEffect. The handler
// itself reads three Decode4s (itemId, nPosX, nPosY); the leading Decode4
// characterId is consumed by CUserPool::OnUserRemotePacket before the opcode
// switch, exactly as for every other CUserRemote::On* op.
//
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v72 ida=0x88d0e4
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v79 ida=0x8d9fe6
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v83 ida=0x983e40
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v84 ida=0x9c4206
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v87 ida=0xa098f2
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v92 ida=0x9307e0
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v95 ida=0x954090
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=jms_v185 ida=0xa57a4e
func TestShowUpgradeTombEffectByteOutput(t *testing.T) {
	variants := []struct {
		name   string
		region string
		major  uint16
	}{
		{"gms_v72", "GMS", 72},
		{"gms_v79", "GMS", 79},
		{"gms_v83", "GMS", 83},
		{"gms_v84", "GMS", 84},
		{"gms_v87", "GMS", 87},
		{"gms_v92", "GMS", 92},
		{"gms_v95", "GMS", 95},
		{"jms_v185", "JMS", 185},
	}
	// characterId 4096 (0x1000); itemId 5510000 (0x541370); x 100; y -200.
	want := []byte{
		0x00, 0x10, 0x00, 0x00,
		0x70, 0x13, 0x54, 0x00,
		0x64, 0x00, 0x00, 0x00,
		0x38, 0xFF, 0xFF, 0xFF,
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			got := NewShowUpgradeTombEffect(4096, 5510000, 100, -200).Encode(nil, ctx)(nil)
			if !bytes.Equal(got, want) {
				t.Errorf("%s ShowUpgradeTombEffect wire: got %x want %x", v.name, got, want)
			}
		})
	}
}

func TestShowUpgradeTombEffectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewShowUpgradeTombEffect(77, 5510000, 1234, -5678)
			output := ShowUpgradeTombEffect{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run ShowUpgradeTombEffect -v`
Expected: FAIL — `undefined: NewShowUpgradeTombEffect`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterShowUpgradeTombEffectWriter = "CharacterShowUpgradeTombEffect"

// ShowUpgradeTombEffect - CUserRemote::OnShowUpgradeTombEffect
// packet-audit:fname CUserRemote::OnShowUpgradeTombEffect
//
// Broadcast to the OTHER sessions in the map when a dead player's client asks
// for the Wheel of Destiny tomb effect (USE_DEATHITEM). The requesting client
// already plays its own copy locally, so the owner is excluded.
//
// Wire layout — identical on every version that carries the op:
//
//	Encode4  characterId  — consumed by CUserPool::OnUserRemotePacket before the switch
//	Encode4  itemId
//	Encode4  x
//	Encode4  y
//
// IDA gms_v95 @0x954090 (dispatcher @0x94b390 case 221 = 0x0DD), gms_v92
// @0x9307e0, gms_v87 @0xa098f2, gms_v84 @0x9c4206, gms_v83 @0x983e40, gms_v79
// @0x8d9fe6 (dispatcher @0x8c8d4a case 181), gms_v72 @0x88d0e4 (dispatcher
// @0x87c046 case 177), jms_v185 @0xa57a4e. No version divergence.
type ShowUpgradeTombEffect struct {
	characterId uint32
	itemId      uint32
	x           int32
	y           int32
}

func NewShowUpgradeTombEffect(characterId uint32, itemId uint32, x int32, y int32) ShowUpgradeTombEffect {
	return ShowUpgradeTombEffect{characterId: characterId, itemId: itemId, x: x, y: y}
}

func (m ShowUpgradeTombEffect) CharacterId() uint32 { return m.characterId }
func (m ShowUpgradeTombEffect) ItemId() uint32      { return m.itemId }
func (m ShowUpgradeTombEffect) X() int32            { return m.x }
func (m ShowUpgradeTombEffect) Y() int32            { return m.y }
func (m ShowUpgradeTombEffect) Operation() string   { return CharacterShowUpgradeTombEffectWriter }
func (m ShowUpgradeTombEffect) String() string {
	return fmt.Sprintf("characterId [%d], itemId [%d], x [%d], y [%d]", m.characterId, m.itemId, m.x, m.y)
}

func (m ShowUpgradeTombEffect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.itemId)
		w.WriteInt32(m.x)
		w.WriteInt32(m.y)
		return w.Bytes()
	}
}

func (m *ShowUpgradeTombEffect) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.itemId = r.ReadUint32()
		m.x = r.ReadInt32()
		m.y = r.ReadInt32()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run ShowUpgradeTombEffect -v`
Expected: PASS.

- [ ] **Step 5: Full library test + vet**

Run: `cd libs/atlas-packet && go test -race ./... && go vet ./...`
Expected: all PASS, no vet output.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect.go libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect_test.go
git commit -m "feat(task-210): add SHOW_UPGRADE_TOMB_EFFECT clientbound codec"
```

---

## Task 3: Packet registry rows and n-a evidence

**Files:**
- Modify: `docs/packets/registry/gms_v72.yaml` (new `USE_DEATHITEM` row + `packet:` on `SHOW_UPGRADE_TOMB_EFFECT`)
- Modify: `docs/packets/registry/gms_v79.yaml` (same)
- Modify: `docs/packets/registry/gms_v83.yaml`, `gms_v84.yaml`, `gms_v87.yaml`, `gms_v92.yaml`, `gms_v95.yaml`, `jms_v185.yaml` (`packet:` on both existing rows)
- Modify: `docs/packets/feature-na-evidence.yaml`

**Interfaces:**
- Consumes: the codec paths produced by Tasks 1 and 2 — `character/serverbound/UseDeathItem` and `character/clientbound/ShowUpgradeTombEffect`. These strings must match exactly; the matrix links rows to codecs by them.
- Produces: registry rows the matrix generator reads in Task 11.

- [ ] **Step 1: Confirm the current state of every row**

Run:
```bash
for v in gms_v72 gms_v79 gms_v83 gms_v84 gms_v87 gms_v92 gms_v95 jms_v185; do
  echo "== $v"; grep -n -A6 "^- op: USE_DEATHITEM$" docs/packets/registry/$v.yaml
  grep -n -A6 "^- op: SHOW_UPGRADE_TOMB_EFFECT$" docs/packets/registry/$v.yaml
done
```
Expected: `SHOW_UPGRADE_TOMB_EFFECT` present in all eight; `USE_DEATHITEM` present in six (absent in `gms_v72`, `gms_v79`). If that does not hold, stop and report — the design's §5.1 claim is the basis for this task.

- [ ] **Step 2: Add the two missing `USE_DEATHITEM` rows**

In `docs/packets/registry/gms_v72.yaml`, insert at the serverbound opcode-52 position (after `FACE_EXPRESSION` opcode 50):

```yaml
- op: USE_DEATHITEM
  direction: serverbound
  opcode: 52
  fname: CUserLocal::RequestUpgradeTombEffect
  packet: character/serverbound/UseDeathItem
  provenance: ida-discovered
  ida:
    address: 8812116
  note: 'v72 CUserLocal::RequestUpgradeTombEffect @0x867654 builds COutPacket(52) and encodes itemId/x/y; renamed from sub_867654 during task-210. (task-210)'
```

In `docs/packets/registry/gms_v79.yaml`, insert at the serverbound opcode-51 position (after `FACE_EXPRESSION` opcode 49):

```yaml
- op: USE_DEATHITEM
  direction: serverbound
  opcode: 51
  fname: CUserLocal::RequestUpgradeTombEffect
  packet: character/serverbound/UseDeathItem
  provenance: ida-discovered
  ida:
    address: 9121776
  note: 'v79 CUserLocal::RequestUpgradeTombEffect @0x8b2ff0 builds COutPacket(51) and encodes itemId/x/y; renamed from sub_8b2ff0 during task-210. (task-210)'
```

Match the surrounding indentation and key order of neighbouring entries in each file; if a file orders keys differently (e.g. `packet:` before `provenance:`), follow the file.

- [ ] **Step 3: Add the `packet:` link to every existing row**

For each of the eight registry files, add `packet: character/serverbound/UseDeathItem` to the `USE_DEATHITEM` row and `packet: character/clientbound/ShowUpgradeTombEffect` to the `SHOW_UPGRADE_TOMB_EFFECT` row, in the same key position other rows in that file use. Do not change `opcode`, `fname`, or `provenance` on rows that already have them.

- [ ] **Step 4: Add the n-a evidence entries**

Append to the `entries:` list in `docs/packets/feature-na-evidence.yaml`:

```yaml
  - op: USE_DEATHITEM
    version: gms_v48
    evidence: >
      The Wheel of Destiny item id 5510000 appears nowhere in
      GMS_v48_1_DEVM.exe: an exhaustive immediate-operand scan over all
      1,198,699 instructions in the image returned zero hits, so no
      CUIRevive/CUserLocal path can construct the request (the client
      hard-codes the id at the send site on every version that has the op).
      No CUserLocal::RequestUpgradeTombEffect and no CUIRevive wheel branch
      exist. (task-210)
  - op: USE_DEATHITEM
    version: gms_v61
    evidence: >
      Same positive-absence proof as gms_v48, over GMS_v61.1_U_DEVM.exe:
      exhaustive immediate-operand scan of all 1,615,793 instructions found
      zero references to 5510000, so the hard-coded send site cannot exist.
      (task-210)
  - op: SHOW_UPGRADE_TOMB_EFFECT
    version: gms_v48
    evidence: >
      Receive side absent for the same reason as the send side: 5510000 does
      not occur anywhere in GMS_v48_1_DEVM.exe (1,198,699 instructions
      scanned, zero hits) and CUserRemote has no OnShowUpgradeTombEffect arm
      in the CUserPool remote-packet dispatcher. (task-210)
  - op: SHOW_UPGRADE_TOMB_EFFECT
    version: gms_v61
    evidence: >
      Same positive-absence proof over GMS_v61.1_U_DEVM.exe: 1,615,793
      instructions scanned, zero references to 5510000, and no
      CUserRemote::OnShowUpgradeTombEffect arm in the remote-packet
      dispatcher. (task-210)
```

- [ ] **Step 5: Validate the registry files parse**

Run:
```bash
python3 -c "import yaml,glob,sys; [yaml.safe_load(open(f)) for f in glob.glob('docs/packets/registry/*.yaml')]; yaml.safe_load(open('docs/packets/feature-na-evidence.yaml')); print('ok')"
```
Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/registry docs/packets/feature-na-evidence.yaml
git commit -m "chore(task-210): register USE_DEATHITEM/SHOW_UPGRADE_TOMB_EFFECT rows and v48/v61 n-a evidence"
```

---

## Task 4: Seed-template bindings for both ops

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_72_1.json`, `template_gms_79_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`

**Interfaces:**
- Consumes: `CharacterUseDeathItemHandle` (Task 1) and `CharacterShowUpgradeTombEffect` (Task 2) — the handler/writer name strings the templates bind.
- Produces: tenant socket configuration that routes the opcode to the handler registered in Task 5.

- [ ] **Step 1: Add the handler entry to each of the eight templates**

Per-file edit (never a shell patch loop — one `Edit` per file). Insert into the `handlers` array at the **sorted `opCode` position**, using the opcode from the Global Constraints table:

```json
      {
        "opCode": "0x35",
        "validator": "LoggedInValidator",
        "handler": "CharacterUseDeathItemHandle",
        "fname": "CUserLocal::RequestUpgradeTombEffect",
        "services": [
          "channel"
        ]
      },
```

(`0x35` shown for v83; substitute `0x34` v72, `0x33` v79, `0x35` v84, `0x38` v87, `0x3B` v92, `0x3A` v95, `0x2D` jms185.) A handler with no `validator` is silently dropped at load — `LoggedInValidator` is required.

- [ ] **Step 2: Add the writer entry to each of the eight templates**

Insert into the `writers` array at the sorted `opCode` position:

```json
      {
        "opCode": "0xC3",
        "writer": "CharacterShowUpgradeTombEffect",
        "fname": "CUserRemote::OnShowUpgradeTombEffect",
        "services": [
          "channel"
        ]
      },
```

(`0xC3` shown for v83; substitute `0xB1` v72, `0xB5` v79, `0xC7` v84, `0xD0` v87, `0xDF` v92, `0xDD` v95, `0xC9` jms185.) Match each file's existing `services` formatting — some templates omit the key on writers; if the neighbouring writer entries in that file omit it, omit it too.

- [ ] **Step 3: Verify placement and JSON validity**

Run:
```bash
python3 - <<'EOF'
import json
t={'gms_72':('0x34','0xB1'),'gms_79':('0x33','0xB5'),'gms_83':('0x35','0xC3'),
   'gms_84':('0x35','0xC7'),'gms_87':('0x38','0xD0'),'gms_92':('0x3B','0xDF'),
   'gms_95':('0x3A','0xDD'),'jms_185':('0x2D','0xC9')}
for k,(h,w) in t.items():
    d=json.load(open(f"services/atlas-configurations/seed-data/templates/template_{k}_1.json"))
    hs=[];ws=[]
    def walk(o):
        if isinstance(o,dict):
            hs.extend(o.get('handlers',[])); ws.extend(o.get('writers',[]))
            for v in o.values(): walk(v)
        elif isinstance(o,list):
            for i in o: walk(i)
    walk(d)
    fh=[x for x in hs if x.get('handler')=='CharacterUseDeathItemHandle']
    fw=[x for x in ws if x.get('writer')=='CharacterShowUpgradeTombEffect']
    assert len(fh)==1 and int(fh[0]['opCode'],16)==int(h,16), (k,'handler',fh)
    assert fh[0].get('validator'), (k,'missing validator')
    assert len(fw)==1 and int(fw[0]['opCode'],16)==int(w,16), (k,'writer',fw)
    assert fw[0].get('fname'), (k,'missing fname')
print("all eight templates bound correctly")
EOF
```
Expected: `all eight templates bound correctly`.

- [ ] **Step 4: Run the template guards**

Run:
```bash
tools/template-opcode-order-guard.sh && tools/template-duplicate-binding-guard.sh && tools/template-movement-types-guard.sh
```
Expected: all three exit 0. If the opcode-order guard fails, the entry was inserted at the wrong index — move it, do not re-sort the whole array.

- [ ] **Step 5: Check the seed corpus count expectation**

Run: `grep -rn "corpus\|expected.*count" services/atlas-configurations/seed-data/ --include=*.go --include=*.md | head`
If a checked-in count of template entries exists (PRD FR-3.3), update it to match. If no such count exists, record that in the commit body and move on.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-configurations/seed-data/templates
git commit -m "feat(task-210): bind USE_DEATHITEM handler and SHOW_UPGRADE_TOMB_EFFECT writer in all eight templates"
```

---

## Task 5: Unit B — the `USE_DEATHITEM` relay handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/use_death_item.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/use_death_item_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (handler map + writer list)

**Interfaces:**
- Consumes: `charsb.UseDeathItem` / `charsb.CharacterUseDeathItemHandle` (Task 1); `charcb.NewShowUpgradeTombEffect` / `charcb.CharacterShowUpgradeTombEffectWriter` (Task 2).
- Produces: `handler.CharacterUseDeathItemHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{})`; and a package-level pure predicate `canShowTombEffect(hp uint16, itemId uint32, wheelQuantity uint32) bool` that the test drives directly.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/use_death_item_test.go`:

```go
package handler

import "testing"

// canShowTombEffect is the whole authorisation decision for the USE_DEATHITEM
// relay: the packet is unauthenticated player input carrying free-form
// coordinates that get broadcast to every other client in the map, so a living
// player must not be able to spam tombstones.
func TestCanShowTombEffect(t *testing.T) {
	const wheel = 5510000
	tests := []struct {
		name          string
		hp            uint16
		itemId        uint32
		wheelQuantity uint32
		want          bool
	}{
		{"dead with a wheel", 0, wheel, 1, true},
		{"dead with several charges", 0, wheel, 3, true},
		{"alive with a wheel", 50, wheel, 1, false},
		{"dead without a wheel", 0, wheel, 0, false},
		{"dead but claims another item", 0, 5130000, 1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := canShowTombEffect(tc.hp, tc.itemId, tc.wheelQuantity); got != tc.want {
				t.Errorf("canShowTombEffect(%d, %d, %d) = %v, want %v", tc.hp, tc.itemId, tc.wheelQuantity, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestCanShowTombEffect -v`
Expected: FAIL — `undefined: canShowTombEffect`.

- [ ] **Step 3: Write the handler**

Create `services/atlas-channel/atlas.com/channel/socket/handler/use_death_item.go`:

```go
package handler

import (
	"atlas-channel/character"
	channelInventory "atlas-channel/inventory"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// canShowTombEffect authorises the relay. The client hard-codes the Wheel of
// Destiny id at the send site and only builds the revive dialog while dead, so
// anything else is a forged or stale request and is dropped silently on the
// wire (logged server-side).
func canShowTombEffect(hp uint16, itemId uint32, wheelQuantity uint32) bool {
	if hp != 0 {
		return false
	}
	if !item.IsWheelOfFortune(item.Id(itemId)) {
		return false
	}
	return wheelQuantity >= 1
}

// CharacterUseDeathItemHandleFunc relays CUIRevive's tomb-effect request to the
// other players in the map. It deliberately consumes nothing and changes no
// state: the wheel is spent by the MAP_CHANGE revive path (respawn.Respawn),
// which the client sends separately when the player actually presses OK.
func CharacterUseDeathItemHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.UseDeathItem{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d] for death item tomb effect.", s.CharacterId())
			return
		}

		var quantity uint32
		inv, err := channelInventory.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get inventory for character [%d] for death item tomb effect.", s.CharacterId())
			return
		}
		if a, found := inv.Cash().FindFirstByItemId(uint32(item.WheelOfFortuneId)); found && a != nil {
			quantity = a.Quantity()
		}

		if !canShowTombEffect(c.Hp(), p.ItemId(), quantity) {
			l.Warnf("Character [%d] requested a death item tomb effect for item [%d] with hp [%d] and [%d] charges. Ignoring.", s.CharacterId(), p.ItemId(), c.Hp(), quantity)
			return
		}

		// Owner excluded: CUserLocal::RequestUpgradeTombEffect already ran
		// CUser::ShowUpgradeTombEffect locally before sending, so echoing to
		// the sender would play the effect twice on their screen.
		err = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(),
			session.Announce(l)(ctx)(wp)(charcb.CharacterShowUpgradeTombEffectWriter)(
				charcb.NewShowUpgradeTombEffect(s.CharacterId(), p.ItemId(), p.X(), p.Y()).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast death item tomb effect for character [%d].", s.CharacterId())
		}
	}
}
```

Design §6's "exactly one foreign broadcast, and **no** announce to the owner's own session" is structural here, not a separate test: the handler's only send is a single `ForOtherSessionsInMap` call, which by construction excludes `s.CharacterId()` and visits each other session once. There is no owner-directed `Announce` anywhere in the file. If a future edit adds one, that is the regression to catch in review.

Before writing, confirm the import alias for the channel map package by reading the top of `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go` — it uses `_map "atlas-channel/map"` alongside `session.Announce`. Match whatever the handler package already uses for the same import; several handler files import it under a different alias.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestCanShowTombEffect -v`
Expected: PASS, all five sub-tests.

- [ ] **Step 5: Register the handler and writer in `main.go`**

In the writer-name slice (near `charcb.CharacterExpressionWriter`, around `main.go:673`), add:

```go
		charcb.CharacterShowUpgradeTombEffectWriter,
```

In the handler map (near `handlerMap[charsb.CharacterExpressionHandle]`, around `main.go:877`), add:

```go
	handlerMap[charsb.CharacterUseDeathItemHandle] = handler.CharacterUseDeathItemHandleFunc
```

- [ ] **Step 6: Build and vet the service**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/use_death_item.go services/atlas-channel/atlas.com/channel/socket/handler/use_death_item_test.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-210): relay USE_DEATHITEM as a tomb effect to other sessions in the map"
```

---

## Task 6: Unit C1 — extract the respawn decision and honour `premium`

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/respawn/plan.go`
- Create: `services/atlas-channel/atlas.com/channel/respawn/plan_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/respawn/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/map_change.go`

**Interfaces:**
- Consumes: `character.Model`, `channelInventory.Model`, `map_.Model`, `asset.Model` from atlas-channel.
- Produces:
  - `type mapFacts struct { ReturnMapId _map.Id; Town bool; NoExpLossOnDeath bool }` and `func mapFactsOf(m map_.Model) mapFacts`. **Why:** `map_.Model` (`data/map/model.go:8`) has all-private fields, no builder, and no exported constructor, so a test in package `respawn` cannot construct one. Taking the three facts the decision actually uses keeps `planRespawn` constructible in a test without adding a test-only constructor to the map package.
  - `type respawnPlan struct { TargetMapId _map.Id; Wheel *asset.Model; Protective *asset.Model; ExpLoss uint32 }`
  - `func planRespawn(c character.Model, inv channelInventory.Model, mf mapFacts, currentMapId _map.Id, useDeathItem bool) respawnPlan`
  - `func findWheelOfFortune(inv channelInventory.Model) *asset.Model`
  - `respawn.Processor.Respawn(f field.Model, characterId uint32, useDeathItem bool) error` — replaces `Respawn(ch channel.Model, characterId uint32, currentMapId _map.Id) error`
  - `respawn.NewProcessor(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) Processor` — the `wp` parameter is unused until Task 8; add it now so the signature does not churn twice. Follows the `movement.NewProcessor(l, ctx, wp)` precedent.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/respawn/plan_test.go`. Every fixture is built with the production Builders (`asset.NewModelBuilder(id, compartmentId, templateId)`, `compartment.NewModelBuilder(id, characterId, type, capacity)`, `inventory.NewModelBuilder(characterId)`, `character.NewModelBuilder()`) — no `*_testhelpers.go` file, no test-only constructors.

```go
package respawn

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"atlas-channel/asset"
	"atlas-channel/compartment"
	channelInventory "atlas-channel/inventory"
	"atlas-channel/character"

	inventoryConst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

const (
	characterId = uint32(4096)
	currentMap  = _map.Id(104040000)
	returnMap   = _map.Id(104000000)
)

// ordinaryField is a non-town map with no field limits and a distinct return
// map — the plain case every test varies from.
var ordinaryField = mapFacts{ReturnMapId: returnMap, Town: false, NoExpLossOnDeath: false}

func buildCharacter(experience uint32) character.Model {
	return character.NewModelBuilder().
		SetId(characterId).
		SetName("Tumi").
		SetLevel(30).
		SetJobId(job.WarriorId).
		SetLuck(60).
		SetExperience(experience).
		SetHp(0).
		SetMaxHp(1000).
		MustBuild()
}

// buildAsset creates one stack of templateId with the given quantity and
// expiration (pass the zero time for "no expiration").
func buildAsset(compartmentId uuid.UUID, templateId uint32, quantity uint32, expiration time.Time) asset.Model {
	return asset.NewModelBuilder(1, compartmentId, templateId).
		SetSlot(1).
		SetQuantity(quantity).
		SetExpiration(expiration).
		MustBuild()
}

// buildInventory places each (templateId, quantity) pair into the compartment
// its inventory type dictates: cash items in Cash, the ETC protection items in
// ETC. A quantity of 0 means "the asset is absent entirely".
func buildInventory(items map[uint32]uint32) channelInventory.Model {
	cashId, etcId := uuid.New(), uuid.New()
	cash := compartment.NewModelBuilder(cashId, characterId, inventoryConst.TypeValueCash, 100)
	etc := compartment.NewModelBuilder(etcId, characterId, inventoryConst.TypeValueETC, 100)
	for templateId, quantity := range items {
		if quantity == 0 {
			continue
		}
		if item.Id(templateId) == item.EasterBasketId || item.Id(templateId) == item.ProtectOnDeathId {
			etc = etc.AddAsset(buildAsset(etcId, templateId, quantity, time.Time{}))
			continue
		}
		cash = cash.AddAsset(buildAsset(cashId, templateId, quantity, time.Time{}))
	}
	return channelInventory.NewModelBuilder(characterId).
		SetCash(cash.MustBuild()).
		SetEtc(etc.MustBuild()).
		MustBuild()
}

func TestPlanRespawn_PremiumZeroKeepsTheWheel(t *testing.T) {
	inv := buildInventory(map[uint32]uint32{uint32(item.WheelOfFortuneId): 1})

	got := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, false)

	if got.TargetMapId != returnMap {
		t.Errorf("target map: got %d, want %d (return map — the player pressed Cancel)", got.TargetMapId, returnMap)
	}
	if got.Wheel != nil {
		t.Error("wheel must not be consumed when the client sent premium = 0")
	}
}

func TestPlanRespawn_PremiumOneWithWheelStaysInMap(t *testing.T) {
	inv := buildInventory(map[uint32]uint32{uint32(item.WheelOfFortuneId): 1})

	got := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)

	if got.TargetMapId != currentMap {
		t.Errorf("target map: got %d, want %d (in-map revive)", got.TargetMapId, currentMap)
	}
	if got.Wheel == nil {
		t.Fatal("wheel should be consumed on an in-map revive")
	}
	if got.Wheel.Quantity() != 1 {
		t.Errorf("wheel quantity: got %d, want 1", got.Wheel.Quantity())
	}
}

func TestPlanRespawn_PremiumOneWithoutWheelUsesReturnMap(t *testing.T) {
	inv := buildInventory(map[uint32]uint32{})

	got := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)

	if got.TargetMapId != returnMap {
		t.Errorf("target map: got %d, want %d", got.TargetMapId, returnMap)
	}
	if got.Wheel != nil {
		t.Error("no wheel owned, nothing to consume")
	}
}
```

If `character.NewModelBuilder().MustBuild()` rejects any of these fixtures, read `character/builder.go`'s `Build()` validation and add the missing required setters — do not weaken the validation. Likewise confirm `job.WarriorId` is the constant's real name in `libs/atlas-constants/job/` (any non-beginner job works; `calculateExpLoss` only calls `job.IsBeginner`).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./respawn/ -v`
Expected: FAIL — `undefined: planRespawn`.

- [ ] **Step 3: Write `plan.go`**

Create `services/atlas-channel/atlas.com/channel/respawn/plan.go`:

```go
package respawn

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	map_ "atlas-channel/data/map"
	channelInventory "atlas-channel/inventory"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// mapFacts is the subset of the map's data the respawn decision needs.
// data/map.Model has no exported constructor, so depending on the three facts
// instead of the model keeps planRespawn testable without adding a
// test-only constructor to the map package.
type mapFacts struct {
	ReturnMapId      _map.Id
	Town             bool
	NoExpLossOnDeath bool
}

func mapFactsOf(m map_.Model) mapFacts {
	return mapFacts{
		ReturnMapId:      m.ReturnMapId(),
		Town:             m.Town(),
		NoExpLossOnDeath: m.NoExpLossOnDeath(),
	}
}

// respawnPlan is the pure outcome of a death: where the character comes back
// and which assets a charge is spent from. Nil asset pointers mean "nothing
// consumed".
type respawnPlan struct {
	TargetMapId _map.Id
	Wheel       *asset.Model
	Protective  *asset.Model
	ExpLoss     uint32
}

// findWheelOfFortune returns the Cash-inventory Wheel of Destiny with at least
// one charge left, or nil. Charges live in the asset's quantity: the client
// gates its own revive dialog on CWvsContext::GetItemCount(5510000) > 0, and
// the client's WZ model for death items (CItemInfo::PROTECTONDIEITEM — nItemID,
// nRecoveryRate) carries no use-count field.
func findWheelOfFortune(inv channelInventory.Model) *asset.Model {
	a, found := inv.Cash().FindFirstByItemId(uint32(item.WheelOfFortuneId))
	if !found || a == nil || a.Quantity() < 1 {
		return nil
	}
	return a
}

// planRespawn decides the respawn outcome. useDeathItem is the client's
// Change.Premium() byte: CUIRevive::OnButtonClicked calls Revive(1) for OK and
// Revive(0) for Cancel, so a zero here means the player declined the wheel and
// their charge must survive.
func planRespawn(c character.Model, inv channelInventory.Model, mf mapFacts, currentMapId _map.Id, useDeathItem bool) respawnPlan {
	p := respawnPlan{TargetMapId: mf.ReturnMapId}

	if useDeathItem {
		if w := findWheelOfFortune(inv); w != nil {
			p.Wheel = w
			p.TargetMapId = currentMapId
		}
	}

	p.Protective = findProtectiveItem(inv)
	p.ExpLoss = calculateExpLoss(c, mf, p.Protective != nil)
	return p
}
```

Then change `findProtectiveItem` and `calculateExpLoss` in `processor.go` from methods on `*ProcessorImpl` to package-level functions with the same bodies, with `findProtectiveItem` returning `*asset.Model` instead of `(*uint32, inventoryConst.Type)`:

```go
// findProtectiveItem returns the death-protection asset that suppresses the
// experience loss, or nil. Cash Safety Charm first, then the two ETC items.
func findProtectiveItem(inv channelInventory.Model) *asset.Model {
	if a, found := inv.Cash().FindFirstByItemId(uint32(item.SafetyCharmId)); found && a != nil && a.Quantity() >= 1 {
		return a
	}
	if a, found := inv.ETC().FindFirstByItemId(uint32(item.EasterBasketId)); found && a != nil && a.Quantity() >= 1 {
		return a
	}
	if a, found := inv.ETC().FindFirstByItemId(uint32(item.ProtectOnDeathId)); found && a != nil && a.Quantity() >= 1 {
		return a
	}
	return nil
}
```

`calculateExpLoss` keeps its logic verbatim (beginner / `NoExpLossOnDeath` / protection / 1%-5%-10% tiers) but loses the `p.l.Debugf` calls, which move to `Respawn`'s log line in Step 4. Its new signature is `func calculateExpLoss(c character.Model, mf mapFacts, hasProtection bool) uint32`, reading `mf.NoExpLossOnDeath` and `mf.Town` where it previously read `mapData.NoExpLossOnDeath()` and `mapData.Town()`.

- [ ] **Step 4: Rewire `Respawn` and the map-change handler**

In `processor.go`:

```go
type Processor interface {
	// Respawn handles character death and respawn logic. useDeathItem is the
	// client's Change.Premium() byte — 1 from the revive dialog's OK button,
	// 0 from Cancel. A Cancel must not spend the player's wheel.
	Respawn(f field.Model, characterId uint32, useDeathItem bool) error
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		wp:  wp,
		cp:  character.NewProcessor(l, ctx),
		ip:  channelInventory.NewProcessor(l, ctx),
		mp:  map_.NewProcessor(l, ctx),
		sp:  saga.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) Respawn(f field.Model, characterId uint32, useDeathItem bool) error {
	currentMapId := f.MapId()
	p.l.Debugf("Processing respawn for character [%d] on map [%d]. useDeathItem [%t].", characterId, currentMapId, useDeathItem)

	c, err := p.cp.GetById()(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get character [%d] for respawn.", characterId)
		return err
	}
	inv, err := p.ip.GetByCharacterId(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get inventory for character [%d].", characterId)
		return err
	}
	mapData, err := p.mp.GetById(currentMapId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get map [%d] data for respawn.", currentMapId)
		return err
	}

	rp := planRespawn(c, inv, mapFactsOf(mapData), currentMapId, useDeathItem)
	return p.createRespawnSaga(f, characterId, rp)
}
```

Add `wp writer.Producer` to `ProcessorImpl` and import `"atlas-channel/socket/writer"` and `"github.com/Chronicle20/atlas/libs/atlas-constants/field"`. Drop the now-unused `channel` and `inventoryConst` imports if nothing else uses them.

Split the saga construction in two so the step list can be asserted without Kafka. Move the whole step-building body into a pure function in `plan.go`:

```go
// respawnSagaSteps builds the ordered step list for a death. Order is
// load-bearing: both consume steps precede warp_to_spawn, so a failed
// decrement aborts the saga before the character is moved and cannot grant a
// free in-map respawn.
func respawnSagaSteps(f field.Model, characterId uint32, rp respawnPlan, now time.Time) []saga.Step {
	steps := make([]saga.Step, 0)
	// … the bodies below, then set_hp, deduct_experience (when rp.ExpLoss > 0),
	// cancel_all_buffs, warp_to_spawn — in that order.
	return steps
}
```

and leave `func (p *ProcessorImpl) createRespawnSaga(f field.Model, characterId uint32, rp respawnPlan) error` as the thin wrapper that calls it and submits `saga.Saga{TransactionId: uuid.New(), SagaType: saga.CharacterRespawn, InitiatedBy: "RESPAWN", Steps: respawnSagaSteps(f, characterId, rp, time.Now())}` through `p.sp.Create`. Source `WorldId: f.WorldId()` and `ChannelId: f.ChannelId()` in the payloads that previously used `ch.WorldId()` / `ch.Id()`, and build the two `DestroyAsset` steps from `rp.Wheel` / `rp.Protective`:

```go
	if rp.Wheel != nil {
		steps = append(steps, saga.Step{
			StepId: "consume_wheel_of_fortune",
			Status: saga.Pending,
			Action: saga.DestroyAsset,
			Payload: saga.DestroyAssetPayload{
				CharacterId: characterId,
				TemplateId:  rp.Wheel.TemplateId(),
				Quantity:    1,
				RemoveAll:   false,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if rp.Protective != nil {
		steps = append(steps, saga.Step{
			StepId: "consume_protective_item",
			Status: saga.Pending,
			Action: saga.DestroyAsset,
			Payload: saga.DestroyAssetPayload{
				CharacterId: characterId,
				TemplateId:  rp.Protective.TemplateId(),
				Quantity:    1,
				RemoveAll:   false,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
```

The rest of the saga (`set_hp`, `deduct_experience` gated on `rp.ExpLoss > 0`, `cancel_all_buffs`, `warp_to_spawn` with `MapId: rp.TargetMapId`) is unchanged apart from the field-sourced world/channel ids.

In `socket/handler/map_change.go`, change the unused writer parameter to `wp writer.Producer` and the call site:

```go
			err = respawn.NewProcessor(l, ctx, wp).Respawn(s.Field(), s.CharacterId(), p.Premium() != 0)
```

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./respawn/ -v && go build ./...`
Expected: all three tests PASS; build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/respawn services/atlas-channel/atlas.com/channel/socket/handler/map_change.go
git commit -m "fix(task-210): only spend the Wheel of Destiny when the client sends premium=1"
```

---

## Task 7: Unit C2 — charge accounting

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/respawn/plan.go`
- Modify: `services/atlas-channel/atlas.com/channel/respawn/plan_test.go`

**Interfaces:**
- Consumes: `respawnPlan`, `findWheelOfFortune`, `findProtectiveItem` (Task 6).
- Produces: `func usesRemaining(a *asset.Model) byte` — the post-decrement charge count clamped into a byte, `0` for a nil asset. Task 8 consumes it for `EffectProtectOnDie.usesRemaining`.

- [ ] **Step 1: Write the failing tests**

Append to `plan_test.go`:

```go
func TestPlanRespawn_ZeroQuantityWheelIsNotUsable(t *testing.T) {
	// A wheel present at quantity 0 must read exactly like an absent one.
	cashId := uuid.New()
	cash := compartment.NewModelBuilder(cashId, characterId, inventoryConst.TypeValueCash, 100).
		AddAsset(buildAsset(cashId, uint32(item.WheelOfFortuneId), 0, time.Time{})).
		MustBuild()
	inv := channelInventory.NewModelBuilder(characterId).SetCash(cash).MustBuild()

	got := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)
	if got.Wheel != nil || got.TargetMapId != returnMap {
		t.Errorf("a wheel with no charges must not redirect the respawn: %+v", got)
	}
}

func TestUsesRemaining(t *testing.T) {
	tests := []struct {
		name     string
		quantity uint32
		nilAsset bool
		want     byte
	}{
		{name: "nil asset", nilAsset: true, want: 0},
		{name: "last charge", quantity: 1, want: 0},
		{name: "three charges", quantity: 3, want: 2},
		{name: "clamped at 255", quantity: 1000, want: 255},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var a *asset.Model
			if !tc.nilAsset {
				built := buildAsset(uuid.New(), uint32(item.WheelOfFortuneId), tc.quantity, time.Time{})
				a = &built
			}
			if got := usesRemaining(a); got != tc.want {
				t.Errorf("usesRemaining(quantity %d) = %d, want %d", tc.quantity, got, tc.want)
			}
		})
	}
}

func TestPlanRespawn_MultipleChargesDecrementRatherThanDestroy(t *testing.T) {
	inv := buildInventory(map[uint32]uint32{uint32(item.WheelOfFortuneId): 3})

	got := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)
	if got.Wheel == nil {
		t.Fatal("expected the wheel to be selected")
	}
	if usesRemaining(got.Wheel) != 2 {
		t.Errorf("post-decrement charges: got %d, want 2", usesRemaining(got.Wheel))
	}
	// The saga step removes exactly one unit and never the whole stack, so a
	// quantity-3 wheel survives and a quantity-1 wheel is destroyed by the
	// same step. Task 6's createRespawnSaga always passes
	// Quantity: 1, RemoveAll: false — pinned by TestRespawnSagaStepOrdering.
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./respawn/ -run "UsesRemaining|MultipleCharges|ZeroQuantity" -v`
Expected: FAIL — `undefined: usesRemaining`.

- [ ] **Step 3: Add `usesRemaining` to `plan.go`**

```go
// usesRemaining is the charge count the client is told about after this death
// spends one — the asset's quantity minus the single unit the DestroyAsset
// step removes. The wire field is one byte, so a stack larger than 256 is
// clamped rather than wrapped.
func usesRemaining(a *asset.Model) byte {
	if a == nil {
		return 0
	}
	q := a.Quantity()
	if q == 0 {
		return 0
	}
	if q-1 > 255 {
		return 255
	}
	return byte(q - 1)
}
```

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./respawn/ -v`
Expected: all PASS.

- [ ] **Step 5: Pin the saga step contract**

These two tests cover the remaining design §6 Unit C cases — the single-consume acceptance criterion (PRD FR-2.3) and the failure semantics (design C4). Append to `plan_test.go`:

```go
func stepIds(steps []saga.Step) []string {
	ids := make([]string, 0, len(steps))
	for _, s := range steps {
		ids = append(ids, s.StepId)
	}
	return ids
}

// Both consume steps must precede warp_to_spawn: the saga stops at the first
// failing step, so a failed decrement can never leave the player revived for
// free in the map they died in.
func TestRespawnSagaStepOrdering(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), currentMap).Build()
	inv := buildInventory(map[uint32]uint32{uint32(item.WheelOfFortuneId): 3})
	rp := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)

	steps := respawnSagaSteps(f, characterId, rp, time.Now())

	ids := stepIds(steps)
	consume, warp := -1, -1
	for i, id := range ids {
		switch id {
		case "consume_wheel_of_fortune":
			consume = i
		case "warp_to_spawn":
			warp = i
		}
	}
	if consume == -1 || warp == -1 {
		t.Fatalf("expected both a consume and a warp step, got %v", ids)
	}
	if consume > warp {
		t.Errorf("consume must precede warp, got %v", ids)
	}
	payload, ok := steps[consume].Payload.(saga.DestroyAssetPayload)
	if !ok {
		t.Fatalf("consume payload is %T, want saga.DestroyAssetPayload", steps[consume].Payload)
	}
	if payload.Quantity != 1 || payload.RemoveAll {
		t.Errorf("consume payload: got quantity %d removeAll %v, want 1/false", payload.Quantity, payload.RemoveAll)
	}
	if payload.TemplateId != uint32(item.WheelOfFortuneId) {
		t.Errorf("consume template: got %d, want %d", payload.TemplateId, item.WheelOfFortuneId)
	}
}

// FR-2.3: one death consumes exactly one charge even when the client sends
// both USE_DEATHITEM and MAP_CHANGE. USE_DEATHITEM is handled by
// CharacterUseDeathItemHandleFunc, which builds no saga and consumes nothing,
// so the only DestroyAsset for the wheel is the one MAP_CHANGE produces here.
func TestOneDeathConsumesOneCharge(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), currentMap).Build()
	inv := buildInventory(map[uint32]uint32{uint32(item.WheelOfFortuneId): 3})
	rp := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)

	steps := respawnSagaSteps(f, characterId, rp, time.Now())

	wheelConsumes := 0
	for _, s := range steps {
		if p, ok := s.Payload.(saga.DestroyAssetPayload); ok && p.TemplateId == uint32(item.WheelOfFortuneId) {
			wheelConsumes++
		}
	}
	if wheelConsumes != 1 {
		t.Errorf("wheel DestroyAsset steps: got %d, want exactly 1", wheelConsumes)
	}
}
```

Add the imports these need: `"atlas-channel/saga"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/field"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/world"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/channel"`. Confirm the `field.NewBuilder(worldId, channelId, mapId).Build()` call shape against `libs/atlas-constants/field/` before writing — the project memory records it as `field.NewBuilder(w, c, m)` with an optional `.SetInstance(uuid)`.

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./respawn/ -v`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/respawn
git commit -m "feat(task-210): gate death items on remaining charges and report the post-decrement count"
```

---

## Task 8: Unit C3 — emit the protect-on-die effect

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/respawn/plan.go` (add `expirationDays`)
- Modify: `services/atlas-channel/atlas.com/channel/respawn/processor.go` (emission)
- Modify: `services/atlas-channel/atlas.com/channel/respawn/plan_test.go`

**Interfaces:**
- Consumes: `usesRemaining` (Task 7); `charpkt.CharacterProtectOnDieItemUseEffectBody(safetyCharm bool, usesRemaining byte, days byte, itemId uint32)` and `charpkt.CharacterProtectOnDieItemUseEffectForeignBody(characterId uint32, safetyCharm bool, usesRemaining byte, days byte, itemId uint32)` from `libs/atlas-packet/character/effect_body.go:135,141`; `charcb.CharacterEffectWriter`, `charcb.CharacterEffectForeignWriter`.
- Produces: `func expirationDays(expiration time.Time, now time.Time) byte`; `func (p *ProcessorImpl) announceProtectOnDie(f field.Model, characterId uint32, a *asset.Model) `.

- [ ] **Step 1: Write the failing test**

Append to `plan_test.go`:

```go
func TestExpirationDays(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		expiration time.Time
		want       byte
	}{
		{"no expiration set", time.Time{}, 0},
		{"already expired", now.Add(-48 * time.Hour), 0},
		{"expires in twelve hours rounds down", now.Add(12 * time.Hour), 0},
		{"expires in three and a half days", now.Add(84 * time.Hour), 3},
		{"expires in a year clamps to 255", now.AddDate(1, 0, 0), 255},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := expirationDays(tc.expiration, now); got != tc.want {
				t.Errorf("expirationDays(%v) = %d, want %d", tc.expiration, got, tc.want)
			}
		})
	}
}

func TestPlanRespawn_SafetyCharmSuppressesExpLossAndIsConsumed(t *testing.T) {
	c := buildCharacter(100000)                                                  // non-beginner, non-zero exp
	inv := buildInventory(map[uint32]uint32{uint32(item.SafetyCharmId): 2})      // two charges
	                                                                             // ordinaryField: non-town, no field limit
	got := planRespawn(c, inv, ordinaryField, currentMap, false)

	if got.ExpLoss != 0 {
		t.Errorf("exp loss: got %d, want 0 (protective item held)", got.ExpLoss)
	}
	if got.Protective == nil {
		t.Fatal("expected the safety charm to be selected for consumption")
	}
	if usesRemaining(got.Protective) != 1 {
		t.Errorf("post-decrement charges: got %d, want 1", usesRemaining(got.Protective))
	}
	if !item.IsSafetyCharm(item.Id(got.Protective.TemplateId())) {
		t.Errorf("template id %d is not the safety charm", got.Protective.TemplateId())
	}
}
```

Both helpers (`buildCharacter`, `buildInventory`) and the `time` / `item` imports already exist from Task 6 — no new fixtures are needed.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./respawn/ -run "ExpirationDays|SafetyCharm" -v`
Expected: FAIL — `undefined: expirationDays`.

- [ ] **Step 3: Add `expirationDays` to `plan.go`**

```go
// expirationDays is the whole days left before the asset expires, 0 when no
// expiration is set or it has already passed. This feeds the second byte of
// EffectProtectOnDie. The v83 client (CUser::OnEffect mode-6 arm @0x937e81)
// reads safetyCharm then two bytes and formats StringPool string 0x0B96 from
// both; which of the two the message calls "days" lives in String.wz, not the
// binary, so this sourcing is the defensible reading of the field name and not
// a verified one. See design.md OQ-3 — if the live message renders the two
// values transposed, the fix is to swap them here and update that note.
func expirationDays(expiration time.Time, now time.Time) byte {
	if expiration.IsZero() {
		return 0
	}
	d := expiration.Sub(now)
	if d <= 0 {
		return 0
	}
	days := int64(d.Hours() / 24)
	if days > 255 {
		return 255
	}
	return byte(days)
}
```

- [ ] **Step 4: Emit the effect from `Respawn`**

In `processor.go`, add after the saga is created successfully:

```go
	rp := planRespawn(c, inv, mapFactsOf(mapData), currentMapId, useDeathItem)
	if err = p.createRespawnSaga(f, characterId, rp); err != nil {
		return err
	}
	// A failed broadcast is logged, never fatal — the revive has already been
	// committed to the saga at this point.
	p.announceProtectOnDie(f, characterId, rp.Protective)
	return nil
```

and the emitter:

```go
// announceProtectOnDie tells the dying player (and the map) that a death
// protection item absorbed the experience loss, and how many uses are left.
// The mode byte is resolved from the tenant's CharacterEffect writer options
// (PROTECT_ON_DIE_ITEM_USE) rather than hard-coded.
func (p *ProcessorImpl) announceProtectOnDie(f field.Model, characterId uint32, a *asset.Model) {
	if a == nil {
		return
	}
	templateId := a.TemplateId()
	safetyCharm := item.IsSafetyCharm(item.Id(templateId))
	remaining := usesRemaining(a)
	days := expirationDays(a.Expiration(), time.Now())

	p.l.Debugf("Character [%d] consumed death protection item [%d]: [%d] uses remaining, [%d] days, safetyCharm [%t].", characterId, templateId, remaining, days, safetyCharm)

	err := session.NewProcessor(p.l, p.ctx).IfPresentByCharacterId(f.Channel())(characterId,
		session.Announce(p.l)(p.ctx)(p.wp)(charcb.CharacterEffectWriter)(
			charpkt.CharacterProtectOnDieItemUseEffectBody(safetyCharm, remaining, days, templateId)))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to announce protect-on-die effect to character [%d].", characterId)
	}

	err = channelMap.NewProcessor(p.l, p.ctx).ForOtherSessionsInMap(f, characterId,
		session.Announce(p.l)(p.ctx)(p.wp)(charcb.CharacterEffectForeignWriter)(
			charpkt.CharacterProtectOnDieItemUseEffectForeignBody(characterId, safetyCharm, remaining, days, templateId)))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to broadcast protect-on-die effect for character [%d].", characterId)
	}
}
```

Imports to add: `"atlas-channel/asset"`, `channelMap "atlas-channel/map"`, `"atlas-channel/session"`, `charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"`, `charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/item"`. Confirm the `charpkt` package path by reading the import block of `services/atlas-channel/atlas.com/channel/socket/handler/mystic_door_enter.go`, which already announces a `CharacterEffect` body, and copy its aliases.

The *values* carried by the two effect bodies (`safetyCharm`, `usesRemaining`, `days`, `itemId`) are pinned by the pure tests in Steps 1–3 and by `TestPlanRespawn_SafetyCharmSuppressesExpLossAndIsConsumed`; the announce plumbing itself is two `session.Announce` calls with no branching, following `mystic_door_enter.go` exactly. Do not stand up fake HTTP/Kafka services to test the plumbing — that trades real coverage for fixture maintenance.

Watch for an import cycle: `respawn` now imports `session` and `map`. If `atlas-channel/map` imports `respawn` (it does not today — verify with `go build`), stop and report rather than papering over it.

- [ ] **Step 5: Run the tests and build**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./respawn/ ./socket/handler/ -v && go build ./... && go vet ./...`
Expected: all PASS, build and vet clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/respawn
git commit -m "feat(task-210): emit EffectProtectOnDie when a death protection item is spent"
```

---

## Task 9: Fix the v95 / jms185 foreign-effect writer binding

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: a resolvable `CharacterEffectForeign` writer on v95 and jms185, which Task 8's `announceProtectOnDie` broadcast needs.

**Why this is in scope:** verified during planning — both templates bind the name `CharacterEffect` **twice**, at two distinct opcodes (v95 `0xE0` and `0xE9`; jms185 `0xCC` and `0xD5`), and `RegisterTenantWriterOptions` (`socket/writer/options_registry.go:24`) keys its table by writer *name*, so one of the two silently wins. The registry names the lower opcode on both: `docs/packets/audits/STATUS.md:265` lists `SHOW_FOREIGN_EFFECT` (`CUser::OnEffect`) at v95 `0x0E0` and jms185 `0x0CC`, and line 279 lists the local `SHOW_ITEM_GAIN_INCHAT` at `0x0E9` / `0x0D5`. Every other v61+ template names the foreign slot `CharacterEffectForeign`. Without this fix, Task 8's foreign broadcast cannot resolve a writer on two of the eight versions. The duplicate-binding guard does not catch it — it only bans the same name at the *same* opcode.

- [ ] **Step 1: Confirm the current state**

Run:
```bash
python3 - <<'EOF'
import json
for f,lo in (("gms_95",0xE0),("jms_185",0xCC)):
    d=json.load(open(f"services/atlas-configurations/seed-data/templates/template_{f}_1.json"))
    ws=[]
    def walk(o):
        if isinstance(o,dict):
            ws.extend(o.get('writers',[]))
            for v in o.values(): walk(v)
        elif isinstance(o,list):
            for i in o: walk(i)
    walk(d)
    print(f, [(w['opCode'],w['writer']) for w in ws if 'CharacterEffect' in str(w.get('writer'))])
EOF
```
Expected: each prints two entries both named `CharacterEffect`.

- [ ] **Step 2: Rename the lower-opcode entry**

In `template_gms_95_1.json`, change the `"writer"` value of the `"opCode": "0xE0"` entry from `"CharacterEffect"` to `"CharacterEffectForeign"`. In `template_jms_185_1.json`, do the same for `"opCode": "0xCC"`. Change nothing else — the `fname`, the `operations` table, and the opcode all stay.

- [ ] **Step 3: Verify and run the guards**

Run the Step 1 script again — expected: one `CharacterEffectForeign` and one `CharacterEffect` per file. Then:
```bash
tools/template-opcode-order-guard.sh && tools/template-duplicate-binding-guard.sh && tools/template-movement-types-guard.sh
```
Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json services/atlas-configurations/seed-data/templates/template_jms_185_1.json
git commit -m "fix(task-210): name the v95/jms185 foreign CUser::OnEffect writer CharacterEffectForeign"
```

---

## Task 10: Add the v92 `CharacterEffect` / `CharacterEffectForeign` writers

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: a v92 `CharacterEffect` writer whose `options.operations` table resolves `PROTECT_ON_DIE_ITEM_USE`, without which Task 8's emission is impossible on v92 (design §5.4).

- [ ] **Step 1: Read the opcodes out of the registry**

Run:
```bash
grep -n -B2 -A6 "^- op: SHOW_FOREIGN_EFFECT$" docs/packets/registry/gms_v92.yaml
grep -n -B2 -A6 "^- op: SHOW_ITEM_GAIN_INCHAT$" docs/packets/registry/gms_v92.yaml
```
Expected (per `docs/packets/audits/STATUS.md:265,279`): foreign `0x0E2`, local `0x0EB`. Use whatever the registry actually says; if it disagrees with STATUS.md, stop and report.

- [ ] **Step 2: Derive the v92 `operations` mode table from the client**

The mode byte for each arm is the jumptable index in the v92 `CUser::OnEffect` handler. Do **not** copy v87's or v95's table — the two disagree (v87 `PROTECT_ON_DIE_ITEM_USE` = 6, v95 = 8), so v92 must be read from its own binary.

Resolve the v92 session from `idb_list` by binary name (`GMS_v92_1_DEVM.exe.i64`) and pass it as the `database` parameter; do not use `select_instance`. Decompile the `CUser::OnEffect` receive handler reached from the local-effect opcode found in Step 1 and enumerate its switch arms in index order. Name each arm with the same key spelling the other templates use — read the full 26-entry v87 table out of `template_gms_87_1.json` for the vocabulary — and record any arm v92 lacks by simply omitting the key.

If an arm's identity cannot be established from the v92 binary, stop and report which index; do not guess a mapping.

- [ ] **Step 3: Insert both writer entries**

At their sorted `opCode` positions in the v92 `writers` array:

```json
      {
        "opCode": "0xE2",
        "writer": "CharacterEffectForeign",
        "fname": "CUser::OnEffect",
        "options": {
          "operations": { "…": 0 }
        },
        "services": [
          "channel"
        ]
      },
```

and the same shape at `0xEB` with `"writer": "CharacterEffect"`. Both carry the identical `operations` table derived in Step 2 — v87, v95, and jms185 all pair identical tables across the two entries.

- [ ] **Step 4: Verify placement, guards, and mode resolution**

Run:
```bash
python3 - <<'EOF'
import json
d=json.load(open("services/atlas-configurations/seed-data/templates/template_gms_92_1.json"))
ws=[]
def walk(o):
    if isinstance(o,dict):
        ws.extend(o.get('writers',[]))
        for v in o.values(): walk(v)
    elif isinstance(o,list):
        for i in o: walk(i)
walk(d)
e=[w for w in ws if w['writer'] in ('CharacterEffect','CharacterEffectForeign')]
assert len(e)==2, e
for w in e:
    ops=w['options']['operations']
    assert 'PROTECT_ON_DIE_ITEM_USE' in ops, w['writer']
    print(w['writer'], w['opCode'], 'PROTECT_ON_DIE_ITEM_USE =', ops['PROTECT_ON_DIE_ITEM_USE'], len(ops),'arms')
EOF
tools/template-opcode-order-guard.sh && tools/template-duplicate-binding-guard.sh && tools/template-movement-types-guard.sh
```
Expected: both writers present with a `PROTECT_ON_DIE_ITEM_USE` arm; guards exit 0.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_92_1.json
git commit -m "feat(task-210): add the missing v92 CUser::OnEffect writers with their derived operations table"
```

---

## Task 11: Promote the sixteen coverage-matrix cells

**Files:**
- Modify: evidence records under `docs/packets/audits/<version>/`, `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` (all written by the tooling, never by hand)

**Interfaces:**
- Consumes: the codecs and their `packet-audit:verify` markers (Tasks 1–2), the registry rows (Task 3).
- Produces: sixteen `✅` cells.

**Cells (16):**

| op | versions |
|---|---|
| `character/serverbound/UseDeathItem` | gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185 |
| `character/clientbound/ShowUpgradeTombEffect` | gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185 |

- [ ] **Step 1: Read the playbook**

Read [`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md) in full before dispatching anything. Do not restate or improvise its procedure — it is the single source of truth for what a cell promotion requires.

- [ ] **Step 2: Verify the cells**

Dispatch the `packet-verifier` agent (or `/verify-packet`) once per cell, batched by IDB so each batch reuses one IDA session, and pinned to Sonnet per the project's model-cost rule. Each agent independently re-derives the client read order from its own version's binary rather than trusting this plan's layout — a cell that reproduces the layout is a pass; a cell that contradicts it is a finding to report, not a fixture to bend.

Batch order (one batch per IDB): `GMS_v72.1_U_DEVM.exe.i64`, `GMS_v79_1_DEVM.exe.i64`, `MapleStory_dump.exe.i64` (v83), `GMS_v84.1_U_DEVM.i64`, `GMSv87_4GB.exe.i64`, `GMS_v92_1_DEVM.exe.i64`, `GMS_v95.0_U_DEVM.exe.i64`, `MapleStory_dump_SCY.exe.i64` (jms185) — two cells each.

- [ ] **Step 3: Regenerate the matrix and check the gates**

Run the matrix regeneration and consistency checks documented in the playbook (`tools/packet-audit`), then:
```bash
grep -n "USE_DEATHITEM\|SHOW_UPGRADE_TOMB_EFFECT" docs/packets/audits/STATUS.md
```
Expected: both rows show `✅` in all eight version columns, `n-a` for gms_v48/gms_v61, and the `USE_DEATHITEM` row now carries `0x034` for v72 and `0x033` for v79 where it previously showed a blank `⬜`.

- [ ] **Step 4: Confirm no other cell regressed**

Run: `git diff --stat docs/packets/audits/STATUS.md docs/packets/audits/status.json` and inspect the diff.
Expected: only the two rows and their `status.json` counterparts changed. Any other row moving is a regression — investigate before committing.

- [ ] **Step 5: Commit**

```bash
git add docs/packets/audits
git commit -m "chore(task-210): verify 16 USE_DEATHITEM / SHOW_UPGRADE_TOMB_EFFECT matrix cells"
```

---

## Task 12: Full verification sweep and code review

**Files:** none created; this task gates the PR.

- [ ] **Step 1: Module-level Go verification**

Run, from the worktree root:
```bash
(cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-configurations/atlas.com/configurations && go test -race ./... && go vet ./... && go build ./...)
```
Expected: all clean. `atlas-configurations` is included because its seed data changed even though no Go file did — a JSON parse failure surfaces in its tests.

- [ ] **Step 2: Repo-root guards**

Run:
```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/lint.sh --check
```
Expected: every one exits 0. `tools/lint.sh --check` needs nvm on PATH — if it false-fails with a node error, source nvm (`. "$NVM_DIR/nvm.sh" && nvm use 22`) and re-run rather than declaring it clean.

- [ ] **Step 3: Container builds**

No `go.mod` was touched by this plan, so the bake step is a confirmation rather than a fix cycle. Run:
```bash
git diff --name-only origin/main...HEAD | grep -c 'go\.mod$' || true
docker buildx bake atlas-channel
docker buildx bake atlas-configurations
```
Expected: the grep prints `0`; both bakes succeed. If any `go.mod` did change, bake every service whose module was touched.

- [ ] **Step 4: Code review**

Invoke `superpowers:requesting-code-review`. Only Go and JSON/YAML changed, so it dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no frontend). Pin the reviewers to Sonnet. Ensure every agent runs with cwd inside this worktree, and confirm `git status` is clean afterwards. Findings land in `docs/tasks/task-210-death-item-revive/audit.md`.

- [ ] **Step 5: Address findings**

Fix everything the review raises on this branch — do not open a follow-up task. Re-run Steps 1–2 after any code change.

- [ ] **Step 6: Update the design's open questions**

Append a short "Post-implementation" note to `design.md` §4 recording: OQ-3's `days` sourcing remains unverified against the live String.wz message (Task 8 Step 3 comment says how to correct it if it renders wrong); OQ-5 remains a deliberate, documented gap. Note also the two template defects this task fixed that were not in the PRD: the v95/jms185 duplicate `CharacterEffect` binding (Task 9) and the missing v92 writers (Task 10).

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-210-death-item-revive
git commit -m "docs(task-210): record audit results and post-implementation findings"
```

---

## Acceptance Criteria Mapping

| PRD / design criterion | Task |
|---|---|
| `USE_DEATHITEM` codec with Encode+Decode, per-version fixtures | 1 |
| `SHOW_UPGRADE_TOMB_EFFECT` codec with Encode+Decode, per-version fixtures | 2 |
| Handler registered in the templates with a validator (8, not 6 — design §5.2) | 4 |
| Writer registered in the templates with an `fname` (8) | 4 |
| Template guards clean | 4, 9, 10, 12 |
| In-map revive on the wheel path; exactly one charge per death | 6 (only `MAP_CHANGE` consumes; the relay handler in 5 consumes nothing) |
| Dying without a usable wheel still uses `ReturnMapId()` | 6 |
| Multi-charge item decrements; last charge destroys | 7 (`Quantity: 1, RemoveAll: false`) |
| `EffectProtectOnDie` + `…Foreign` with post-decrement `usesRemaining` | 8, plus 9 and 10 so it can resolve on v95/jms185/v92 |
| Other players see the tomb effect, including v72/v79 | 5 (all eight versions send `USE_DEATHITEM`; design §1.3) |
| 16 matrix cells ✅, n-a cells unchanged | 3, 11 |
| No previously-✅ cell regressed | 11 Step 4 |
| Go build/test/vet, bake, lint and guards clean | 12 |
| OQ-1…OQ-5 answered or explicitly still-open | design §4 (already written) + 12 Step 6 |
