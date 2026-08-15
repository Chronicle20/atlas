# Pet Name Tag (5170000) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Pet Name Tag (item `5170000`) rename a player's lead pet, persist the rename in atlas-pets, broadcast it to the map via a new `PET_NAMECHANGE` clientbound packet, and consume the tag only after the rename succeeds.

**Architecture:** A cash-slot dispatch fix in atlas-channel routes the type-17 sub-body (a single length-prefixed string) into a new handler, which resolves the character's lead pet (slot 0 — the same pet the client picks), validates the name, and starts a two-step `pet_name_tag_use` saga: `rename_pet` then `consume_pet_name_tag`. atlas-pets owns the write and emits `NAME_CHANGED`; atlas-channel consumes that event and broadcasts the new `PetNameChanged` packet to everyone in the map. Failures reject before the saga starts (nothing consumed) or roll the name back via the compensator.

**Tech Stack:** Go 1.x multi-module workspace (`go.work`), GORM + Postgres, Kafka (`libs/atlas-kafka` message buffers + outbox), JSON:API via api2go, `libs/atlas-packet` codecs with tenant-resolved opcodes, `libs/atlas-saga` shared saga contract.

**Spec:** [`design.md`](design.md) (PRD: [`prd.md`](prd.md))

## Global Constraints

Copied verbatim from the design/PRD and `CLAUDE.md`. Every task's requirements implicitly include this section.

- **Name bounds are 4–12 characters**, not 1–13. Source: the v83 client input dialog `sub_9AC7CB(dlg, NULL, 4, 12, 0, 1)` @`0xa0bb2f`. The `pets.name` column is `size:13` — one wider, deliberately, and NOT the binding constraint. (design C-2)
- **The clientbound `flag` byte is GMS-only.** JMS v185 `CPet::OnNameChanged` @`0x76a5de` performs exactly one `DecodeStr` and no `Decode1`. Gate with `t.IsRegion("GMS")`. (design C-1)
- **The serverbound type-17 sub-body is exactly one `EncodeStr`** plus the standard `updateTimeFirst` trailing gate. No pet identifier rides the wire. (design §3.2, PRD FR-2.2)
- **The server resolves the target pet as the lead pet (`Slot() == 0`).** This is not a policy invention — the client's own arm calls `sub_46D2D5(this, 0)`, resolving pet-locker index 0. (design OQ-4)
- **Saga step order is rename-first, consume-second.** A failed rename must never cost the player the item. (PRD FR-7.2)
- **No raw `> N` version comparisons.** Use `t.IsRegion(...)` / `t.MajorAtLeast(...)`. (`bug_majorversion_gt83_is_off_by_one_v87`)
- **Every seed-template writer entry MUST carry an `fname`** (`bug_seed_template_writers_require_fname`) and MUST be inserted at its sorted `opCode` position (`tools/template-opcode-order-guard.sh`).
- **No `// TODO`, stubbed handlers, or 501s** may land. (CLAUDE.md)
- **Multi-tenancy:** every processor/consumer/producer resolves tenant via `tenant.MustFromContext(ctx)`.
- **`ForEachInMap`/`ForSessionsInMap` on the channel side is parallel** — the broadcast callback must close over immutable values only (`bug_channel_foreachinmap_parallel_shared_state`).
- **Do not restate playbook procedure.** Packet-cell verification follows [`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md) unchanged.

---

## File Structure

**New files**

| Path | Responsibility |
|---|---|
| `libs/atlas-constants/pet/name.go` | The 4/12 bounds, `NormalizeName`, `ValidateName`. Single source shared by atlas-channel and atlas-pets. |
| `libs/atlas-constants/pet/name_test.go` | Bounds/normalize/validate tests. |
| `libs/atlas-packet/cash/serverbound/item_use_pet_name_tag.go` | Type-17 sub-body codec. |
| `libs/atlas-packet/cash/serverbound/item_use_pet_name_tag_test.go` | Round-trip + per-version byte fixtures. |
| `libs/atlas-packet/pet/clientbound/name_changed.go` | `PET_NAMECHANGE` codec + the shared `NameTagLayer` constant. |
| `libs/atlas-packet/pet/clientbound/name_changed_test.go` | Per-version byte fixtures with `packet-audit:verify` markers. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_pet_name_tag.go` | The type-17 handler: pet resolution, validation, saga build. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_pet_name_tag_test.go` | Handler arm tests via package-var seams. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/pet/mock/processor.go` | Pet processor mock — the orchestrator's `pet` package has none today, and the compensation test needs one. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/pet_name_tag_compensation_test.go` | Reverse-walk compensation tests. |
| `services/atlas-channel/atlas.com/channel/kafka/message/pet/contract_mirror_test.go` | Cross-module contract drift guard (the pet mirror has no guard script). |
| `docs/tasks/task-224-pet-name-tag/rollout.md` | Live-tenant socket-config reconciliation step. |

**Modified files**

| Path | Change |
|---|---|
| `libs/atlas-packet/pet/clientbound/activated.go` | `NewPetSpawnActivated` sets `nameTag: NameTagLayer` so spawn and rename agree. |
| `libs/atlas-saga/model.go` | `PetNameTagUse` type, `RenamePet` action. |
| `libs/atlas-saga/payloads.go` | `RenamePetPayload`. |
| `libs/atlas-saga/unmarshal.go` | `case RenamePet` arm. |
| `services/atlas-pets/.../kafka/message/pet/kafka.go` | `RENAME` command + `NAME_CHANGED` event + bodies. |
| `services/atlas-pets/.../pet/builder.go` | `SetName`. |
| `services/atlas-pets/.../pet/administrator.go` | `updateName`. |
| `services/atlas-pets/.../pet/processor.go` | `Rename` / `RenameAndEmit` (+ interface + mock). |
| `services/atlas-pets/.../pet/producer.go` | `nameChangedEventProvider`. |
| `services/atlas-pets/.../kafka/consumer/pet/consumer.go` | `handleRenameCommand`. |
| `services/atlas-pets/.../pet/resource.go` | `PATCH /pets/{petId}`. |
| `services/atlas-saga-orchestrator/.../kafka/message/pet/kafka.go` | Contract mirror. |
| `services/atlas-saga-orchestrator/.../pet/{processor,producer}.go` | `Rename` / `RenameAndEmit` / `RenameProvider`. |
| `services/atlas-saga-orchestrator/.../saga/model.go` | Type/action re-exports + payload unmarshal arm. |
| `services/atlas-saga-orchestrator/.../saga/event_acceptance.go` | `EventKindPetNameChanged` + mappings. |
| `services/atlas-saga-orchestrator/.../saga/handler.go` | `handleRenamePet`. |
| `services/atlas-saga-orchestrator/.../kafka/consumer/pet/consumer.go` | `handleNameChangedEvent`. |
| `services/atlas-saga-orchestrator/.../saga/compensator.go` | `DispatchPetNameTagRollbacks` + `compensatePetNameTagUse`. |
| `services/atlas-saga-orchestrator/.../saga/timer.go` | Classification lists + timeout dispatch arm. |
| `services/atlas-channel/.../kafka/message/pet/kafka.go` | Contract mirror. |
| `services/atlas-channel/.../kafka/message/saga/kafka.go` | `SagaTypePetNameTagUse`. |
| `services/atlas-channel/.../saga/model.go` | Type/action/payload re-exports. |
| `services/atlas-channel/.../socket/handler/character_cash_item_use.go` | Const, 517 predicate fix, type-17 arm. |
| `services/atlas-channel/.../kafka/consumer/pet/consumer.go` | `handleNameChanged` broadcast. |
| `services/atlas-channel/.../kafka/consumer/saga/consumer.go` | Pet-name-tag failure pink text. |
| `services/atlas-channel/.../main.go` | `PetNameChangedWriter` registration. |
| `services/atlas-configurations/seed-data/templates/template_*.json` (9 files) | `PetNameChanged` writer entry. |
| `docs/packets/audits/STATUS.md` (+ evidence records) | Matrix promotion (regenerated, not hand-edited). |
| `docs/research/missing-features/items-and-consumables.md`, `packet-gap-inference.md` | Mark implemented. |

---

## Task 1: Shared pet-name bounds in `libs/atlas-constants`

**Files:**
- Create: `libs/atlas-constants/pet/name.go`
- Test: `libs/atlas-constants/pet/name_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: package `pet` at import path `github.com/Chronicle20/atlas/libs/atlas-constants/pet` (alias it `petconst` in services, which already have local `pet` packages) exporting:
  - `const MinNameLength = 4`
  - `const MaxNameLength = 12`
  - `var ErrNameTooShort error`
  - `var ErrNameTooLong error`
  - `func NormalizeName(s string) string`
  - `func ValidateName(s string) error`

Note: `libs/atlas-constants/pet/` currently contains only the `skill/` subpackage — this task creates the first file in the parent package.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-constants/pet/name_test.go`:

```go
package pet_test

import (
	"errors"
	"strings"
	"testing"

	petconst "github.com/Chronicle20/atlas/libs/atlas-constants/pet"
)

func TestNormalizeNameTrimsSurroundingWhitespace(t *testing.T) {
	if got := petconst.NormalizeName("  Fluffy \t"); got != "Fluffy" {
		t.Fatalf("NormalizeName = %q, want %q", got, "Fluffy")
	}
}

func TestValidateNameAcceptsBounds(t *testing.T) {
	for _, name := range []string{"Abcd", "Abcdefghijkl"} {
		if err := petconst.ValidateName(name); err != nil {
			t.Fatalf("ValidateName(%q) = %v, want nil", name, err)
		}
	}
}

func TestValidateNameRejectsTooShort(t *testing.T) {
	for _, name := range []string{"", "Abc"} {
		if err := petconst.ValidateName(name); !errors.Is(err, petconst.ErrNameTooShort) {
			t.Fatalf("ValidateName(%q) = %v, want ErrNameTooShort", name, err)
		}
	}
}

func TestValidateNameRejectsTooLong(t *testing.T) {
	name := strings.Repeat("A", petconst.MaxNameLength+1)
	if err := petconst.ValidateName(name); !errors.Is(err, petconst.ErrNameTooLong) {
		t.Fatalf("ValidateName(%q) = %v, want ErrNameTooLong", name, err)
	}
}

// A whitespace-only name is rejected only because the caller normalized first;
// ValidateName itself never trims (PRD FR-4.2/FR-4.3).
func TestNormalizeThenValidateRejectsWhitespaceOnly(t *testing.T) {
	if err := petconst.ValidateName(petconst.NormalizeName("     ")); !errors.Is(err, petconst.ErrNameTooShort) {
		t.Fatalf("whitespace-only name accepted")
	}
}

func TestBoundsMatchClientDialog(t *testing.T) {
	if petconst.MinNameLength != 4 || petconst.MaxNameLength != 12 {
		t.Fatalf("bounds = (%d,%d), want (4,12) per sub_9AC7CB @0xa0bb2f", petconst.MinNameLength, petconst.MaxNameLength)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./pet/ -run TestValidateName -v`
Expected: FAIL — build error, `undefined: petconst.ValidateName` (no non-test Go file exists in that package yet).

- [ ] **Step 3: Write minimal implementation**

Create `libs/atlas-constants/pet/name.go`:

```go
// Package pet holds Atlas-canonical pet constants shared across services.
package pet

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// MinNameLength / MaxNameLength are the bounds the GMS v83 client's own pet-name
// input dialog enforces: sub_9AC7CB(dlg, NULL, 4, 12, 0, 1) @0xa0bb2f, where a3
// is the minimum and a4 the maximum. The (min,max) reading is fixed by three
// sibling call sites in the same binary — CTabParty::OnInvite @0x90e1da passes
// (4,12) for a character name, ask_SPW @0x9ad030 passes (8,8) for the 8-digit
// second password, ask_guildname @0x9ad131 passes (4,12) for a guild name.
//
// The pets.name column is size:13 (services/atlas-pets/.../pet/entity.go) — one
// wider than anything the client can send, deliberately. The column is NOT the
// binding constraint; these are.
const (
	MinNameLength = 4
	MaxNameLength = 12
)

var (
	ErrNameTooShort = errors.New("pet name is too short")
	ErrNameTooLong  = errors.New("pet name is too long")
)

// NormalizeName trims the surrounding whitespace the client's own
// TrimLeft/TrimRight removes before it encodes, so both sides validate the same
// string. Callers MUST normalize before calling ValidateName.
func NormalizeName(s string) string {
	return strings.TrimSpace(s)
}

// ValidateName reports whether an ALREADY-NORMALIZED name is acceptable. Length
// is counted in runes, not bytes: the bound the client enforces is a character
// count on its edit control, and a byte count would reject legal multi-byte
// names one character short.
func ValidateName(s string) error {
	n := utf8.RuneCountInString(s)
	if n < MinNameLength {
		return ErrNameTooShort
	}
	if n > MaxNameLength {
		return ErrNameTooLong
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-constants && go test ./pet/... -v`
Expected: PASS (all six tests, plus the existing `pet/skill` tests).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/pet/name.go libs/atlas-constants/pet/name_test.go
git commit -m "feat(task-224): shared pet-name bounds in atlas-constants"
```

---

## Task 2: Serverbound type-17 sub-body codec

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_pet_name_tag.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_pet_name_tag_test.go`
- Reference (do not modify): `libs/atlas-packet/cash/serverbound/item_use_kite.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: package `serverbound` exporting
  - `type ItemUsePetNameTag struct` (unexported fields `name`, `updateTime`, `updateTimeFirst`)
  - `func NewItemUsePetNameTag(updateTimeFirst bool) *ItemUsePetNameTag`
  - `func (m ItemUsePetNameTag) Name() string`
  - `func (m ItemUsePetNameTag) UpdateTime() uint32`
  - `func (m ItemUsePetNameTag) Operation() string` → `"ItemUsePetNameTag"`
  - `func (m ItemUsePetNameTag) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte`
  - `func (m *ItemUsePetNameTag) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{})`

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/cash/serverbound/item_use_pet_name_tag_test.go`:

```go
package serverbound

import (
	"bytes"
	"testing"
)

// The GMS v83 case-17 arm of CWvsContext::SendConsumeCashItemUseRequest
// (arm entry @0xa0ba15) performs exactly ONE encode — COutPacket::EncodeStr
// @0xa0bcb5 — and then falls through to the shared tail at loc_A0E9EC, which
// appends update_time on the builds that trail it.
// packet-audit:verify packet=cash/serverbound/ItemUsePetNameTag version=gms_v83 ida=0xa0bcb5
func TestItemUsePetNameTagBytesTrailingUpdateTime(t *testing.T) {
	m := ItemUsePetNameTag{name: "Fluffy", updateTime: 0x01020304, updateTimeFirst: false}
	got := m.Encode(nil, nil)(nil)
	want := []byte{
		0x06, 0x00, // name length
		'F', 'l', 'u', 'f', 'f', 'y',
		0x04, 0x03, 0x02, 0x01, // trailing update_time
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

// GMS v87+ and JMS v185 lead update_time in the common ItemUse header, so the
// sub-body is the string alone (cash/serverbound/item_use.go UpdateTimeFirst).
func TestItemUsePetNameTagBytesLeadingUpdateTime(t *testing.T) {
	m := ItemUsePetNameTag{name: "Fluffy", updateTime: 0x01020304, updateTimeFirst: true}
	got := m.Encode(nil, nil)(nil)
	want := []byte{
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

func TestItemUsePetNameTagDecodeRoundTrip(t *testing.T) {
	for _, first := range []bool{false, true} {
		src := ItemUsePetNameTag{name: "Rex", updateTime: 0x0A0B0C0D, updateTimeFirst: first}
		b := src.Encode(nil, nil)(nil)

		dst := NewItemUsePetNameTag(first)
		dst.Decode(nil, nil)(newReaderFor(b), nil)

		if dst.Name() != "Rex" {
			t.Fatalf("first=%t name = %q, want %q", first, dst.Name(), "Rex")
		}
		if !first && dst.UpdateTime() != 0x0A0B0C0D {
			t.Fatalf("updateTime = %X, want 0A0B0C0D", dst.UpdateTime())
		}
	}
}
```

Note on `newReaderFor`: reuse whatever the sibling tests in this package already
use to construct a `*request.Reader` from a byte slice — open
`libs/atlas-packet/cash/serverbound/item_use_kite_test.go` and copy that
construction verbatim rather than inventing a helper. If those tests only assert
`Encode` bytes, drop `TestItemUsePetNameTagDecodeRoundTrip` from this file and
instead assert `Decode` through the same mechanism the kite test uses.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run PetNameTag -v`
Expected: FAIL — `undefined: ItemUsePetNameTag`.

- [ ] **Step 3: Write minimal implementation**

Create `libs/atlas-packet/cash/serverbound/item_use_pet_name_tag.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUsePetNameTag is the CashSlotItemType(17) sub-body of USE_CASH_ITEM —
// the new name a player types for their pet with a classification-517 Pet Name
// Tag (5170000).
//
// Derived from the case-17 arm of CWvsContext::SendConsumeCashItemUseRequest
// (gms_v83 @0xa0a63f; arm entry @0xa0ba15, labelled by IDA as
// "jumptable 00A0A6E6 case 17"). The arm resolves the pet at locker index 0
// (sub_46D2D5(this, 0) @0xa0ba47), prompts twice (CUtilDlg::YesNo @0xa0baa2 and
// @0xa0bc88), reads the name from a 4..12-bounded input dialog
// (sub_9AC7CB(dlg, NULL, 4, 12, 0, 1) @0xa0bb2f, GetInputStr_Result @0xa0bb68),
// screens it through CCurseProcess::ProcessString @0xa0bb9a, and then performs
// its ONLY encode: COutPacket::EncodeStr @0xa0bcb5.
//
// So the sub-body is exactly one length-prefixed string. NO pet identifier is on
// the wire — not an index, not a locker SN, not a slot. SetUtilDlgEx_Pet
// (@0x9acb27, the pet-picker) is never called from this send path; its only
// callers are CDraggableItem::OnDoubleClicked, CScriptMan::OnAskPet, and
// CScriptMan::OnAskPetAll. The server therefore resolves the target pet itself
// (the lead pet, slot 0 — matching the client's own sub_46D2D5(…, 0) choice).
//
// updateTimeFirst mirrors ItemUse.UpdateTimeFirst: GMS <= v84 trails
// update_time after the sub-body (this arm falls through to the shared tail at
// loc_A0E9EC), GMS v87+ and JMS lead it in the header.
type ItemUsePetNameTag struct {
	name            string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUsePetNameTag(updateTimeFirst bool) *ItemUsePetNameTag {
	return &ItemUsePetNameTag{updateTimeFirst: updateTimeFirst}
}

func (m ItemUsePetNameTag) Name() string       { return m.name }
func (m ItemUsePetNameTag) UpdateTime() uint32 { return m.updateTime }

func (m ItemUsePetNameTag) Operation() string { return "ItemUsePetNameTag" }

func (m ItemUsePetNameTag) String() string {
	return fmt.Sprintf("name [%s] updateTime [%d]", m.name, m.updateTime)
}

func (m ItemUsePetNameTag) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUsePetNameTag) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/item_use_pet_name_tag.go libs/atlas-packet/cash/serverbound/item_use_pet_name_tag_test.go
git commit -m "feat(task-224): serverbound ItemUsePetNameTag sub-body codec"
```

---

## Task 3: Clientbound `PET_NAMECHANGE` codec

**Files:**
- Create: `libs/atlas-packet/pet/clientbound/name_changed.go`
- Create: `libs/atlas-packet/pet/clientbound/name_changed_test.go`
- Modify: `libs/atlas-packet/pet/clientbound/activated.go` (`NewPetSpawnActivated` only)
- Reference: `libs/atlas-packet/pet/clientbound/chat.go`, `libs/atlas-packet/pet/clientbound/v61_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: package `clientbound` exporting
  - `const PetNameChangedWriter = "PetNameChanged"`
  - `const NameTagLayer = byte(0)`
  - `type NameChanged struct` (fields `ownerId uint32`, `slot int8`, `name string`, `nameTag byte`)
  - `func NewPetNameChanged(ownerId uint32, slot int8, name string) NameChanged`
  - `Operation()`, `String()`, `Encode`, `Decode` with the same signatures as `Chat`.

**Wire layout.** `ownerId uint32` + `slot int8` are the upstream prefix every pet leaf body carries — they are read by `CUser::OnPetPacket` before the per-op dispatch, are byte-verified for the whole family in `v61_test.go:11-14`, and every existing pet codec encodes them. Then `DecodeStr(name)`; then, **on GMS only**, `Decode1(flag)`.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/pet/clientbound/name_changed_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestPetNameChangedBytesV83: CPet::OnNameChanged @0x704801 does
// DecodeStr(name) -> this+38, then `if (CInPacket::Decode1(a2))` selects the
// name-tag decoration layer passed to CLife::MakeNameTag.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v83 ida=0x704801
func TestPetNameChangedBytesV83(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId (upstream)
		0x05,       // slot (upstream)
		0x06, 0x00, // name length
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00, // nameTag layer selector (GMS only)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 = % X, want % X", got, want)
	}
}

// TestPetNameChangedBytesJMS185 is the regression test for design C-1: JMS
// v185's CPet::OnNameChanged @0x76a5de performs exactly one DecodeStr and NO
// Decode1 — it branches on sub_768D82(this), a client-side state query, not a
// wire byte. The JMS body is therefore one byte SHORTER than the GMS body.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=jms_v185 ida=0x76a5de
func TestPetNameChangedBytesJMS185(t *testing.T) {
	ctx := test.CreateContext("JMS", 185, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		// no trailing flag byte
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("jms_v185 = % X, want % X", got, want)
	}
	if len(got) != 13 {
		t.Fatalf("jms body length = %d, want 13 (one shorter than GMS)", len(got))
	}
}

func TestPetNameChangedDecodeRoundTripGMS(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	b := NewPetNameChanged(0x01020304, 0x05, "Rex").Encode(nil, ctx)(nil)

	var dst NameChanged
	dst.Decode(nil, ctx)(newReaderFor(b), nil)

	if dst.name != "Rex" || dst.ownerId != 0x01020304 || dst.slot != 0x05 {
		t.Fatalf("decoded = %+v", dst)
	}
}

// The rename packet and the spawn body must write the SAME decoration selector,
// or a renamed pet's name tag appears on rename and vanishes on the next
// respawn (design §1, "what the clientbound flag byte selects").
func TestNameTagLayerAgreesWithActivated(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	spawn := NewPetSpawnActivated(1, 0, 5000000, "Rex", 7, 0, 0, 0, 0).Encode(nil, ctx)(nil)
	// nameTag is the second-to-last byte of the active spawn body, immediately
	// before chatBalloon (activated.go Encode).
	if spawn[len(spawn)-2] != NameTagLayer {
		t.Fatalf("Activated nameTag = %d, want NameTagLayer (%d)", spawn[len(spawn)-2], NameTagLayer)
	}
}
```

Note on `newReaderFor`: reuse the exact `*request.Reader` construction the
existing tests in this package use (`chat_test.go` / `exclude_test.go`). If they
do not decode, drop `TestPetNameChangedDecodeRoundTripGMS` and keep the byte
fixtures — those are the matrix evidence.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./pet/clientbound/ -run PetNameChanged -v`
Expected: FAIL — `undefined: NewPetNameChanged`.

- [ ] **Step 3: Write minimal implementation**

Create `libs/atlas-packet/pet/clientbound/name_changed.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const PetNameChangedWriter = "PetNameChanged"

// NameTagLayer is the CLife::MakeNameTag decoration selector. GMS v95's
// PDB-backed decompile names it outright (@0x6a125a-0x6a1271):
//
//	if (CInPacket::Decode1(p)) nNameTag = this->m_pTemplate->nNameTag; else nNameTag = 0;
//
// so it selects whether the pet TEMPLATE's decorative name-tag layer is drawn —
// it is NOT a boolean "has a name". v83 (@0x704840), v61 (@0x613615) and v92 do
// the same through unnamed offsets.
//
// The same value already rides the spawn body as Activated.nameTag. The two MUST
// agree: a rename that writes 1 while the spawn writes 0 makes the decoration
// appear on rename and disappear on the next respawn. Atlas has no per-pet
// name-tag inventory, so both write 0. This is a render selector, not a
// per-version wire code, which is why it is a named constant rather than a
// tenant-config lookup (design §5 A5; DOM-25 requires provenance, not a config
// key nobody will tune).
const NameTagLayer = byte(0)

// packet-audit:fname CPet::OnNameChanged
//
// ownerId + slot are the upstream prefix CUser::OnPetPacket reads before
// dispatching to the per-op leaf; every pet codec in this package carries them
// and v61_test.go byte-verifies the framing for the whole family.
type NameChanged struct {
	ownerId uint32
	slot    int8
	name    string
	nameTag byte
}

func NewPetNameChanged(ownerId uint32, slot int8, name string) NameChanged {
	return NameChanged{ownerId: ownerId, slot: slot, name: name, nameTag: NameTagLayer}
}

func (m NameChanged) Operation() string { return PetNameChangedWriter }

func (m NameChanged) String() string {
	return fmt.Sprintf("ownerId [%d], slot [%d], name [%s]", m.ownerId, m.slot, m.name)
}

func (m NameChanged) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt8(m.slot)
		w.WriteAsciiString(m.name)
		// GMS reads a trailing flag byte; JMS v185 does not. jms
		// CPet::OnNameChanged @0x76a5de performs exactly one DecodeStr and
		// branches on sub_768D82(this) — client state, not the wire.
		if t.IsRegion("GMS") {
			w.WriteByte(m.nameTag)
		}
		return w.Bytes()
	}
}

func (m *NameChanged) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.slot = r.ReadInt8()
		m.name = r.ReadAsciiString()
		if t.IsRegion("GMS") {
			m.nameTag = r.ReadByte()
		}
	}
}
```

- [ ] **Step 4: Point `Activated` at the shared constant**

Modify `libs/atlas-packet/pet/clientbound/activated.go` — in `NewPetSpawnActivated`, add the field so one symbol feeds both codecs:

```go
func NewPetSpawnActivated(ownerId uint32, slot int8, templateId uint32, name string, petId uint64, x int16, y int16, stance byte, foothold uint16) Activated {
	return Activated{
		ownerId: ownerId, slot: slot, active: true,
		templateId: templateId, name: name, petId: petId,
		x: x, y: y, stance: stance, foothold: foothold,
		// Must equal what NameChanged writes, or a renamed pet's decoration
		// appears on rename and vanishes on the next respawn.
		nameTag: NameTagLayer,
	}
}
```

This is a no-op on the wire today (`NameTagLayer` is 0 and the field defaulted to 0), so the existing `activated_test.go` fixtures stay green — that is the point of doing it now.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./pet/... -v`
Expected: PASS, including the pre-existing `activated_test.go` and `v61_test.go`.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/pet/clientbound/name_changed.go libs/atlas-packet/pet/clientbound/name_changed_test.go libs/atlas-packet/pet/clientbound/activated.go
git commit -m "feat(task-224): clientbound PetNameChanged codec with GMS-only flag byte"
```

---

## Task 4: Per-version derivation and matrix promotion for `PET_NAMECHANGE`

**Files:**
- Modify: `libs/atlas-packet/pet/clientbound/name_changed_test.go` (add the remaining version fixtures)
- Modify: `docs/packets/audits/STATUS.md` (regenerated — never hand-edited) and the per-cell evidence records the playbook pins

**Interfaces:**
- Consumes: `NewPetNameChanged` from Task 3.
- Produces: no Go symbols; produces the promoted matrix row.

**This task follows [`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md) unchanged — do not restate or improvise its procedure.** The single-cell entry point is the `/verify-packet` command / `packet-verifier` agent, one dispatch per cell.

Derivation status entering this task (read during design, addresses recorded in `design.md` §3.3): **v61** @`0x6135d6` ✓, **v83** @`0x704801` ✓, **v92** @`0x6967c0` ✓, **v95** @`0x6a11f0` ✓, **jms_v185** @`0x76a5de` ✓ (the divergence). **v72, v79, v84, v87 have NOT been read** — each must be decompiled before its fixture is written. The four GMS reads all agree on `str + byte`, but an expectation is not evidence (`bug_matrix_roundtrip_fixture_false_verify`).

Note from the design: **the entire v92 column is `❌`** — every pet op sits unverified there, so v92 is a fresh derivation, not a copy of a verified neighbour. Size it as real work.

- [ ] **Step 1: Confirm the registry opcodes are still what the matrix says**

Run:
```bash
for f in docs/packets/registry/*.yaml; do printf '%s ' "$(basename "$f")"; grep -A3 'PET_NAMECHANGE' "$f" | grep -m1 'opcode:' || echo '(absent)'; done
```
Expected (confirmed at plan time — re-confirm, since a registry edit stales the matrix, `bug_registry_fname_change_stales_packet_matrix`):

| v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms185 |
|---|---|---|---|---|---|---|---|---|---|
| absent | 131 `0x083` | 157 `0x09D` | 161 `0x0A1` | 172 `0x0AC` | 176 `0x0B0` | 185 `0x0B9` | 200 `0x0C8` | 203 `0x0CB` | 178 `0x0B2` |

- [ ] **Step 2: Derive the remaining four GMS bodies**

For each of **v72, v79, v84, v87**: resolve the IDB session from `idb_list` **by binary name** (never by port — `select_instance` is dead) and decompile `CPet::OnNameChanged`. Record the address and the exact read order.

- [ ] **Step 3: Add a byte fixture per version**

Append one fixture per version to `name_changed_test.go`, each in the shape of `TestPetNameChangedBytesV83` from Task 3, each carrying its own `packet-audit:verify` marker line with that version's real IDA address. Example for v92 (substitute the real per-version address you read in Step 2 for the versions you derived there):

```go
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v92 ida=0x6967c0
func TestPetNameChangedBytesV92(t *testing.T) {
	ctx := test.CreateContext("GMS", 92, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v92 = % X, want % X", got, want)
	}
}
```

If a version's decompile shows a body that is NOT `str + byte`, widen the gate in `name_changed.go` using `t.IsRegion(...)`/`t.MajorAtLeast(...)` — never a raw `> N` — and say so in the codec comment.

- [ ] **Step 4: Run the fixtures**

Run: `cd libs/atlas-packet && go test ./pet/clientbound/ -run PetNameChanged -v`
Expected: PASS for all ten fixtures (nine implemented versions + the JMS shortness assertion).

- [ ] **Step 5: Pin evidence and regenerate the matrix**

Per the playbook: pin each cell's evidence record and regenerate `STATUS.md` with the packet-audit tool. Record `gms_v48` as `n-a` (the op is absent from its registry) and confirm it passes the n-a consistency gate. Do NOT hand-edit `STATUS.md`.

- [ ] **Step 6: Verify the row promoted**

Run: `grep -n 'PET_NAMECHANGE' docs/packets/audits/STATUS.md`
Expected: nine `✅` cells and `n-a` for v48 — zero `❌` remaining on that row. A cell that did not promote is a failure to fix, not a note to write.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/pet/clientbound/name_changed_test.go docs/packets/audits/
git commit -m "test(task-224): per-version PET_NAMECHANGE fixtures; promote matrix row"
```

---

## Task 5: atlas-pets Kafka contract — `RENAME` command and `NAME_CHANGED` event

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/kafka/message/pet/kafka.go`

**Interfaces:**
- Consumes: nothing.
- Produces (this is the OWNER copy; Tasks 10 and 13 mirror it byte-for-byte):
  - `const CommandPetRename = "RENAME"`
  - `const StatusEventTypeNameChanged = "NAME_CHANGED"`
  - `type RenameCommandBody struct { Name string \`json:"name"\` }`
  - `type NameChangedStatusEventBody struct { Slot int8 \`json:"slot"\`; Name string \`json:"name"\`; PreviousName string \`json:"previousName"\`; TransactionId uuid.UUID \`json:"transactionId"\` }`

`petId` and `actorId`/`ownerId` are already carried by the generic `Command[E]` / `StatusEvent[E]` envelopes — do not duplicate them in the bodies.

- [ ] **Step 1: Add the command constant and body**

In the first `const (...)` block of `kafka.go`, after `CommandSetSkill`:

```go
	CommandPetRename         = "RENAME"
```

After `type EvolveCommandBody struct{}`:

```go
// RenameCommandBody carries the new pet name. It is ALREADY normalized by the
// caller, but atlas-pets re-validates it regardless (PRD FR-5.6) — the channel
// is not trusted to have validated, and a crafted producer could publish
// straight to this topic.
type RenameCommandBody struct {
	Name string `json:"name"`
}
```

- [ ] **Step 2: Add the status-event constant and body**

In the status-event `const (...)` block, after `StatusEventTypeFlagChanged`:

```go
	StatusEventTypeNameChanged      = "NAME_CHANGED"
```

At the end of the file, beside `EvolvedStatusEventBody`:

```go
// NameChangedStatusEventBody drives two consumers. atlas-channel needs Slot to
// address the clientbound PET_NAMECHANGE packet (the packet carries no pet id —
// it is routed by ownerId+slot). The orchestrator needs TransactionId to
// complete the rename_pet step. PreviousName is what the compensator re-applies
// if the consume step later fails.
type NameChangedStatusEventBody struct {
	Slot          int8      `json:"slot"`
	Name          string    `json:"name"`
	PreviousName  string    `json:"previousName"`
	TransactionId uuid.UUID `json:"transactionId"`
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd services/atlas-pets/atlas.com/pets && go build ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/kafka/message/pet/kafka.go
git commit -m "feat(task-224): pet RENAME command and NAME_CHANGED event contract"
```

---

## Task 6: atlas-pets model write path — `SetName` builder and `updateName` administrator

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/pet/builder.go`
- Modify: `services/atlas-pets/atlas.com/pets/pet/administrator.go`
- Test: `services/atlas-pets/atlas.com/pets/pet/builder_test.go` (append)

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `func (b *ModelBuilder) SetName(name string) *ModelBuilder`
  - `func updateName(db *gorm.DB) func(petId uint32, name string) error` (package-private)

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-pets/atlas.com/pets/pet/builder_test.go`:

```go
func TestModelBuilderSetName(t *testing.T) {
	m, err := NewModelBuilder(1, 0, 5000000, "Original", 42).SetName("Renamed").Build()
	if err != nil {
		t.Fatalf("Build() = %v", err)
	}
	if m.Name() != "Renamed" {
		t.Fatalf("Name() = %q, want %q", m.Name(), "Renamed")
	}
}

// Build's name-required invariant must still hold through the new setter.
func TestModelBuilderSetNameEmptyRejected(t *testing.T) {
	if _, err := NewModelBuilder(1, 0, 5000000, "Original", 42).SetName("").Build(); err == nil {
		t.Fatal("Build() with empty name = nil error, want error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./pet/ -run SetName -v`
Expected: FAIL — `m.SetName undefined`.

- [ ] **Step 3: Add the builder setter**

In `builder.go`, beside `SetTemplateId`:

```go
func (b *ModelBuilder) SetName(name string) *ModelBuilder {
	b.name = name
	return b
}
```

- [ ] **Step 4: Add the administrator function**

In `administrator.go`, after `updateLevel`:

```go
func updateName(db *gorm.DB) func(petId uint32, name string) error {
	return func(petId uint32, name string) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Update("name", name)

		if result.Error != nil {
			return result.Error
		}

		// Deliberately NOT treating RowsAffected == 0 as an error, unlike the
		// sibling update functions above. Kafka is at-least-once: a redelivered
		// RENAME whose value is already applied updates zero rows, and erroring
		// there would fail the orchestrator's rename_pet step on a duplicate
		// that changed nothing (PRD FR-5.5). Existence is proven by the caller's
		// pre-read inside the same transaction.
		return nil
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./pet/ -run SetName -v && go build ./...`
Expected: PASS. (`updateName` is unused until Task 7 — Go permits unused package-level funcs.)

- [ ] **Step 6: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/pet/builder.go services/atlas-pets/atlas.com/pets/pet/administrator.go services/atlas-pets/atlas.com/pets/pet/builder_test.go
git commit -m "feat(task-224): pet SetName builder and updateName administrator"
```

---

## Task 7: atlas-pets rename processor, producer, and command consumer

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/pet/processor.go`
- Modify: `services/atlas-pets/atlas.com/pets/pet/producer.go`
- Modify: `services/atlas-pets/atlas.com/pets/pet/mock/processor.go`
- Modify: `services/atlas-pets/atlas.com/pets/kafka/consumer/pet/consumer.go`
- Test: `services/atlas-pets/atlas.com/pets/pet/processor_test.go` (append)

**Interfaces:**
- Consumes: `petconst.NormalizeName` / `petconst.ValidateName` (Task 1); `updateName` (Task 6); `pet.CommandPetRename`, `pet.RenameCommandBody`, `pet.StatusEventTypeNameChanged`, `pet.NameChangedStatusEventBody` (Task 5).
- Produces:
  - `Processor` interface gains `RenameAndEmit(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error` and `Rename(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error`
  - `func nameChangedEventProvider(m Model, previousName string, transactionId uuid.UUID) model.Provider[[]kafka.Message]`
  - `func handleRenameCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.RenameCommandBody]]`

Import the constants package as `petconst "github.com/Chronicle20/atlas/libs/atlas-constants/pet"` — the local package is also named `pet`. The dependency is already in `services/atlas-pets/atlas.com/pets/go.mod:6`, so no `go.mod` change and no docker-bake round is triggered by this task.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-pets/atlas.com/pets/pet/processor_test.go`, matching the existing test file's setup helpers (it already builds a real sqlite/postgres-backed `*gorm.DB` and a tenant context — reuse those helpers verbatim rather than writing new ones):

```go
func TestRenameAppliesAndEmits(t *testing.T) {
	// Arrange: use the same DB + context setup the sibling tests in this file use.
	l, ctx, db := testSetup(t)
	p := NewProcessor(l, ctx, db)
	created := mustCreatePet(t, p, "Original")

	// Act
	err := p.RenameAndEmit(uuid.New(), created.Id(), created.OwnerId(), "Renamed")

	// Assert
	if err != nil {
		t.Fatalf("RenameAndEmit = %v", err)
	}
	got, err := p.GetById(created.Id())
	if err != nil {
		t.Fatalf("GetById = %v", err)
	}
	if got.Name() != "Renamed" {
		t.Fatalf("Name() = %q, want %q", got.Name(), "Renamed")
	}
}

// FR-5.5: Kafka is at-least-once. A redelivered RENAME whose value is already
// applied must complete, not error — the orchestrator's rename_pet step
// completes on the re-emitted event.
func TestRenameIsIdempotent(t *testing.T) {
	l, ctx, db := testSetup(t)
	p := NewProcessor(l, ctx, db)
	created := mustCreatePet(t, p, "Original")

	if err := p.RenameAndEmit(uuid.New(), created.Id(), created.OwnerId(), "Renamed"); err != nil {
		t.Fatalf("first rename = %v", err)
	}
	if err := p.RenameAndEmit(uuid.New(), created.Id(), created.OwnerId(), "Renamed"); err != nil {
		t.Fatalf("second (redelivered) rename = %v, want nil", err)
	}
}

// FR-5.6: atlas-pets does not trust atlas-channel to have validated.
func TestRenameRejectsInvalidName(t *testing.T) {
	l, ctx, db := testSetup(t)
	p := NewProcessor(l, ctx, db)
	created := mustCreatePet(t, p, "Original")

	for _, bad := range []string{"", "abc", "abcdefghijklm"} {
		if err := p.RenameAndEmit(uuid.New(), created.Id(), created.OwnerId(), bad); err == nil {
			t.Fatalf("RenameAndEmit(%q) = nil, want validation error", bad)
		}
	}
	got, _ := p.GetById(created.Id())
	if got.Name() != "Original" {
		t.Fatalf("name mutated to %q on a rejected rename", got.Name())
	}
}

// FR-3.3 defence in depth: a rename requested by a non-owner is rejected.
func TestRenameRejectsNonOwner(t *testing.T) {
	l, ctx, db := testSetup(t)
	p := NewProcessor(l, ctx, db)
	created := mustCreatePet(t, p, "Original")

	if err := p.RenameAndEmit(uuid.New(), created.Id(), created.OwnerId()+1, "Renamed"); err == nil {
		t.Fatal("RenameAndEmit by non-owner = nil, want error")
	}
}
```

`testSetup` / `mustCreatePet` are placeholders for the file's existing helpers — open `processor_test.go` and substitute the real names before running. Do not add new test-only constructors (CLAUDE.md: use the Builder pattern, no `*_testhelpers.go`).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./pet/ -run TestRename -v`
Expected: FAIL — `p.RenameAndEmit undefined`.

- [ ] **Step 3: Add the processor methods**

In `processor.go`, add to the `Processor` interface beside `EvolveAndEmit`/`Evolve`:

```go
	RenameAndEmit(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error
	Rename(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error
```

And the implementations, beside `EvolveAndEmit`/`Evolve`:

```go
func (p *ProcessorImpl) RenameAndEmit(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) error {
			return p.With(WithTransaction(tx)).Rename(mb)(transactionId, petId, actorId, name)
		})
	})
}

// Rename applies a new pet name and emits NAME_CHANGED.
//
// Idempotent by construction (PRD FR-5.5): the pre-read proves the row exists,
// so updateName's zero-rows-affected case is a no-op rather than an error, and
// the status event is emitted on EVERY delivery — that re-emission is required,
// not incidental, because it is how a redelivered command still completes the
// orchestrator's rename_pet step.
func (p *ProcessorImpl) Rename(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error {
	return func(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error {
		p.l.Debugf("Renaming pet [%d] to [%s].", petId, name)

		// Re-validate here rather than trusting the caller: atlas-channel
		// validates too, but anything can publish to this topic (PRD FR-5.6).
		normalized := petconst.NormalizeName(name)
		if err := petconst.ValidateName(normalized); err != nil {
			p.l.WithError(err).Warnf("Rejecting rename of pet [%d] to [%s]: invalid name.", petId, name)
			return err
		}

		var previousName string
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			pe, err := p.With(WithTransaction(tx)).GetById(petId)
			if err != nil {
				return err
			}
			if pe.OwnerId() != actorId {
				return fmt.Errorf("pet [%d] is not owned by character [%d]", petId, actorId)
			}
			previousName = pe.Name()

			updated, err := Clone(pe).SetName(normalized).Build()
			if err != nil {
				return err
			}
			if err = updateName(tx)(petId, normalized); err != nil {
				return err
			}
			return mb.Put(pet.EnvStatusEventTopic, nameChangedEventProvider(updated, previousName, transactionId))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to rename pet [%d].", petId)
			return txErr
		}

		p.l.Infof("Renamed pet [%d]: [%s] -> [%s].", petId, previousName, normalized)
		return nil
	}
}
```

Add the import `petconst "github.com/Chronicle20/atlas/libs/atlas-constants/pet"` (and `fmt` if not already imported).

- [ ] **Step 4: Add the producer**

In `producer.go`, beside `evolvedEventProvider`:

```go
func nameChangedEventProvider(m Model, previousName string, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.NameChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeNameChanged,
		Body: pet.NameChangedStatusEventBody{
			Slot:          m.Slot(),
			Name:          m.Name(),
			PreviousName:  previousName,
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Extend the processor mock**

In `pet/mock/processor.go`, add the two fields and two methods mirroring the existing `EvolveAndEmitFunc` / `EvolveFunc` pair exactly:

```go
	RenameAndEmitFunc                        func(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error
	RenameFunc                               func(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error
```

```go
func (m *ProcessorMock) RenameAndEmit(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error {
	if m.RenameAndEmitFunc != nil {
		return m.RenameAndEmitFunc(transactionId, petId, actorId, name)
	}
	return nil
}

func (m *ProcessorMock) Rename(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error {
	if m.RenameFunc != nil {
		return m.RenameFunc(mb)
	}
	return func(transactionId uuid.UUID, petId uint32, actorId uint32, name string) error { return nil }
}
```

Match the surrounding mock's default-return convention if it differs from the above.

- [ ] **Step 6: Register the command consumer**

In `kafka/consumer/pet/consumer.go`, add a handler registration in `InitHandlers` after the `handleEvolveCommand` block:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRenameCommand(db)))); err != nil {
				return err
			}
```

and the handler beside `handleEvolveCommand`:

```go
func handleRenameCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.RenameCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.RenameCommandBody]) {
		if c.Type != pet2.CommandPetRename {
			return
		}
		err := pet.NewProcessor(l, ctx, db).RenameAndEmit(c.TransactionId, c.PetId, c.ActorId, c.Body.Name)
		if err != nil {
			l.WithError(err).Errorf("Unable to rename pet [%d] for character [%d].", c.PetId, c.ActorId)
		}
	}
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-pets/atlas.com/pets && go test -race ./... && go vet ./...`
Expected: PASS, clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/pet/ services/atlas-pets/atlas.com/pets/kafka/consumer/pet/consumer.go
git commit -m "feat(task-224): atlas-pets rename processor, producer, and RENAME consumer"
```

---

## Task 8: atlas-pets `PATCH /pets/{petId}` operator surface

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/pet/resource.go`
- Test: `services/atlas-pets/atlas.com/pets/pet/resource_test.go` (append)

**Interfaces:**
- Consumes: `RenameAndEmit` (Task 7), `petconst.NormalizeName`/`ValidateName` (Task 1).
- Produces: `func handleUpdate(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc` and the route registration.

This is deliberately an operator/admin surface. **The gameplay path is Kafka, not REST** — atlas-channel never calls it (PRD §5.1).

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-pets/atlas.com/pets/pet/resource_test.go`, following the request/response construction the existing tests in that file use:

```go
func TestPatchPetRejectsInvalidName(t *testing.T) {
	// Reuse this file's existing router/DB/tenant-context setup.
	router, _ := testRouter(t)

	body := jsonAPIBody(t, "pets", "1", map[string]any{"name": "ab"})
	rr := doRequest(t, router, http.MethodPatch, "/pets/1", body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
```

`testRouter` / `jsonAPIBody` / `doRequest` are placeholders for the file's existing helpers — substitute the real names before running. Note the JSON:API envelope requirement (`bug_ui_jsonapi_envelope_required_for_input_handlers`): the body must be a full `{"data":{"type":"pets","id":"1","attributes":{...}}}` document, not a bare object.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./pet/ -run TestPatchPet -v`
Expected: FAIL — 405 Method Not Allowed (no PATCH route registered).

- [ ] **Step 3: Register the route**

In `InitResource`, after the existing `/{petId}` GET:

```go
			r.HandleFunc("/{petId}", rest.RegisterInputHandler[RestModel](l)(db)(si)("update_pet", handleUpdate)).Methods(http.MethodPatch)
```

- [ ] **Step 4: Implement the handler**

Add to `resource.go`:

```go
// handleUpdate is the operator surface for correcting a pet name without a
// direct DB write. The gameplay rename path is the RENAME Kafka command driven
// by the pet_name_tag_use saga — atlas-channel never calls this endpoint
// (PRD §5.1). `name` is the only writable attribute; every other field on the
// inbound RestModel is ignored.
func handleUpdate(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return rest.ParsePetId(d.Logger(), func(petId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name := petconst.NormalizeName(i.Name)
			if err := petconst.ValidateName(name); err != nil {
				d.Logger().WithError(err).Warnf("Rejecting PATCH of pet [%d]: invalid name [%s].", petId, i.Name)
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			existing, err := p.GetById(petId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate pet [%d].", petId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// The owner is taken from the stored row, never from the request:
			// the processor's ownership check would otherwise be trivially
			// satisfiable by a caller supplying whatever ownerId it liked.
			if err = p.RenameAndEmit(uuid.New(), petId, existing.OwnerId(), name); err != nil {
				d.Logger().WithError(err).Errorf("Unable to rename pet [%d].", petId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			updated, err := p.GetById(petId)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			res, err := model.Map(Transform(d.Context()))(model.FixedProvider(updated))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
```

Add imports: `petconst "github.com/Chronicle20/atlas/libs/atlas-constants/pet"` and `"github.com/google/uuid"`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-pets/atlas.com/pets && go test -race ./... && go vet ./...`
Expected: PASS, clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/pet/resource.go services/atlas-pets/atlas.com/pets/pet/resource_test.go
git commit -m "feat(task-224): PATCH /pets/{petId} operator rename surface"
```

---

## Task 9: Shared saga contract — `PetNameTagUse` type and `RenamePet` action

**Files:**
- Modify: `libs/atlas-saga/model.go`
- Modify: `libs/atlas-saga/payloads.go`
- Modify: `libs/atlas-saga/unmarshal.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `const PetNameTagUse Type = "pet_name_tag_use"`
  - `const RenamePet Action = "rename_pet"`
  - `type RenamePetPayload struct { CharacterId uint32 \`json:"characterId"\`; PetId uint32 \`json:"petId"\`; Name string \`json:"name"\`; PreviousName string \`json:"previousName"\` }`
  - a `case RenamePet` arm in the shared unmarshal switch

- [ ] **Step 1: Add the saga type**

In `libs/atlas-saga/model.go`, in the `Type` const block after `MesoSackUse`:

```go
	PetNameTagUse       Type = "pet_name_tag_use"
```

- [ ] **Step 2: Add the action**

In the `Action` const block, beside `EvolvePet`:

```go
	RenamePet              Action = "rename_pet"
```

- [ ] **Step 3: Add the payload**

In `libs/atlas-saga/payloads.go`, after `EvolvePetPayload`:

```go
// RenamePetPayload drives a pet rename. PreviousName is captured by the
// initiating service BEFORE the rename (atlas-channel already reads the pet to
// resolve the target, so this costs no extra round trip) and exists solely so
// the compensator can revert the name if a later step fails.
type RenamePetPayload struct {
	CharacterId  uint32 `json:"characterId"`
	PetId        uint32 `json:"petId"`
	Name         string `json:"name"`
	PreviousName string `json:"previousName"`
}
```

- [ ] **Step 4: Add the unmarshal arm**

In `libs/atlas-saga/unmarshal.go`, beside the `case EvolvePet:` arm, add the structurally identical arm for `RenamePet` using `RenamePetPayload` — copy the neighbouring arm's exact shape (error handling, assignment) rather than paraphrasing it.

- [ ] **Step 5: Verify**

Run: `cd libs/atlas-saga && go test ./... && go vet ./...`
Expected: PASS, clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-saga/
git commit -m "feat(task-224): PetNameTagUse saga type and RenamePet action"
```

---

## Task 10: Orchestrator wiring — mirror, processor, handler, event acceptance, consumer

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/pet/kafka.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/pet/processor.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/pet/producer.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/pet/consumer.go`

**Interfaces:**
- Consumes: `sharedsaga.RenamePet`, `sharedsaga.RenamePetPayload`, `sharedsaga.PetNameTagUse` (Task 9); the Task 5 contract shape.
- Produces:
  - orchestrator `pet.Processor` gains `RenameAndEmit(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error` and `Rename(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error`
  - `func RenameProvider(transactionId uuid.UUID, petId uint32, characterId uint32, name string) model.Provider[[]kafka.Message]`
  - `const EventKindPetNameChanged EventKind = "pet.name_changed"`
  - `func (h *HandlerImpl) handleRenamePet(s Saga, st Step[any]) error`
  - `func handleNameChangedEvent(l logrus.FieldLogger, ctx context.Context, e pet2.StatusEvent[pet2.NameChangedStatusEventBody])`
  - `saga/model.go` re-exports: `PetNameTagUse`, `RenamePet`, `RenamePetPayload`

- [ ] **Step 1: Mirror the Kafka contract**

Copy the Task 5 additions verbatim into `saga-orchestrator/kafka/message/pet/kafka.go` — same constant names, same struct names, **same json tags**. This mirror has no guard (only trade has one, `tools/trade-contract-mirror-guard.sh`), so a typo here fails no build and silently decodes to a zero-valued body. Task 12 adds the round-trip test that turns that into a red test.

Note: this file spells the command-type constants differently in places (e.g. `CommandTypeAwardCloseness` for what atlas-pets calls `CommandAwardCloseness`). Match **this file's** local naming convention for the constant identifier; the **string value** (`"RENAME"`, `"NAME_CHANGED"`) and every **json tag** must be byte-identical to atlas-pets.

- [ ] **Step 2: Add the producer and processor methods**

In `saga-orchestrator/pet/producer.go`, beside `EvolveProvider`:

```go
func RenameProvider(transactionId uuid.UUID, petId uint32, characterId uint32, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.RenameCommandBody]{
		TransactionId: transactionId,
		ActorId:       characterId,
		PetId:         petId,
		Type:          pet2.CommandPetRename,
		Body: pet2.RenameCommandBody{
			Name: name,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

In `saga-orchestrator/pet/processor.go`, add to the `Processor` interface and implement beside `Evolve`:

```go
	RenameAndEmit(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error
	Rename(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error
```

```go
func (p *ProcessorImpl) RenameAndEmit(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Rename(mb)(transactionId, petId, characterId, name)
	})
}

func (p *ProcessorImpl) Rename(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error {
	return func(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error {
		return mb.Put(pet2.EnvCommandTopic, RenameProvider(transactionId, petId, characterId, name))
	}
}
```

- [ ] **Step 3: Re-export the type/action/payload**

In `saga-orchestrator/saga/model.go`: add `PetNameTagUse = sharedsaga.PetNameTagUse` to the type block, `RenamePet = sharedsaga.RenamePet` to the action block, `RenamePetPayload = sharedsaga.RenamePetPayload` to the payload alias block, and a `case RenamePet:` arm to the payload-unmarshal switch mirroring the `case EvolvePet:` arm exactly. `unmarshal_completeness_test.go` will fail if the arm is missing — that is the intended gate.

- [ ] **Step 4: Wire event acceptance**

In `saga-orchestrator/saga/event_acceptance.go`:

```go
	EventKindPetNameChanged      EventKind = "pet.name_changed"
```

```go
	sharedsaga.RenamePet:              {EventKindPetNameChanged},
```

```go
	EventKindPetNameChanged:      OutcomeSuccess,
```

(one line into each of the three existing blocks, beside their `EventKindPetEvolved` neighbours).

- [ ] **Step 5: Add the step handler**

In `saga-orchestrator/saga/handler.go`, add `handleRenamePet(s Saga, st Step[any]) error` to the `Handler` interface beside `handleEvolvePet`, a `case RenamePet: return h.handleRenamePet, true` arm to the action dispatch switch, and:

```go
func (h *HandlerImpl) handleRenamePet(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(RenamePetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.petP.RenameAndEmit(s.TransactionId(), payload.PetId, payload.CharacterId, payload.Name)
	if err != nil {
		h.logActionError(s, st, err, "Unable to rename pet.")
		return err
	}

	return nil
}
```

- [ ] **Step 6: Consume `NAME_CHANGED`**

In `saga-orchestrator/kafka/consumer/pet/consumer.go`, register `handleNameChangedEvent` alongside `handleEvolvedEvent` and add it in the same shape as its neighbour:

```go
func handleNameChangedEvent(l logrus.FieldLogger, ctx context.Context, e pet2.StatusEvent[pet2.NameChangedStatusEventBody]) {
	if e.Type != pet2.StatusEventTypeNameChanged {
		return
	}
	// Mirror handleEvolvedEvent's body exactly from here: resolve the saga
	// processor, then AcceptEvent(e.Body.TransactionId, saga.EventKindPetNameChanged).
}
```

Copy `handleEvolvedEvent`'s body verbatim, substituting the event kind and body type — do not paraphrase its tenant/processor setup.

- [ ] **Step 7: Write the acceptance test**

Add to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/accept_event_test.go` (or a new `pet_name_tag_accept_event_test.go` in the same package), following the arrangement `processor_test.go:610-660` uses — build the saga, `GetCache().Put(ctx, s)`, then `AcceptEvent`:

```go
// EventKindPetNameChanged must complete a pending rename_pet step. Without the
// event_acceptance.go wiring, AcceptEvent returns ok=false and the saga stalls
// until its timer fires — the rename would apply while the tag was never
// consumed, and the timeout backstop would then revert the name.
func TestRenamePetStepCompletesOnMatchingTransaction(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := testTenantContext()
	tx := uuid.New()

	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(PetNameTagUse).
		SetInitiatedBy("pet-name-tag-accept-test").
		AddStep("rename_pet", Pending, RenamePet, RenamePetPayload{
			CharacterId: 77002, PetId: 4242, Name: "Renamed", PreviousName: "Original",
		}).
		AddStep("consume_pet_name_tag", Pending, DestroyAsset, DestroyAssetPayload{
			CharacterId: 77002, TemplateId: 5170000, Quantity: 1, RemoveAll: false,
		}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	p := NewProcessor(logger, ctx)

	_, ok := p.AcceptEvent(tx, EventKindPetNameChanged)
	require.True(t, ok, "EventKindPetNameChanged must be accepted for a pending rename_pet step")
}

// An event carrying a transaction id this orchestrator knows nothing about must
// complete nothing.
func TestRenamePetStepDoesNotCompleteOnMismatchedTransaction(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := testTenantContext()

	p := NewProcessor(logger, ctx)

	_, ok := p.AcceptEvent(uuid.New(), EventKindPetNameChanged)
	require.False(t, ok, "an unknown transaction id must not be accepted")
}
```

Copy the `testTenantContext` / `NewBuilder` / `AddStep` spellings from `processor_test.go` — that file is the ground truth for both APIs. Add a `//go:build test` tag line if the file you extend carries one.

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./...`
Expected: PASS, clean. `unmarshal_completeness_test.go` must be green.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-saga-orchestrator/
git commit -m "feat(task-224): orchestrator rename_pet action, event acceptance, and consumer"
```

---

## Task 11: Orchestrator compensation and timeout classification

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/pet/mock/processor.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/pet_name_tag_compensation_test.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go`

**Interfaces:**
- Consumes: `PetNameTagUse`, `RenamePet`, `RenamePetPayload` (Tasks 9–10); `petP.RenameAndEmit` (Task 10).
- Produces:
  - `package mock` at `atlas-saga-orchestrator/pet/mock` with `type ProcessorMock struct` implementing `pet.Processor`
  - `Compensator` interface gains `DispatchPetNameTagRollbacks(s Saga)` and `WithPetProcessor(petP pet.Processor) Compensator`
  - `func (c *CompensatorImpl) compensatePetNameTagUse(s Saga, failedStep Step[any]) error`
  - `func petNameTagCharacterId(s Saga) uint32`

Note: the orchestrator's `pet` package currently has **no mock** (unlike `compartment`, `character`, etc.), and `CompensatorImpl` has no `petP` field or `WithPetProcessor` injector. Both are prerequisites for the test below, so they are part of this task — not a deferral.

- [ ] **Step 1: Create the orchestrator pet processor mock**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/pet/mock/processor.go`, following the field-plus-nil-check convention every other orchestrator mock uses (e.g. `compartment/mock/processor.go`):

```go
package mock

import (
	"atlas-saga-orchestrator/kafka/message"

	"github.com/google/uuid"
)

type ProcessorMock struct {
	GainClosenessAndEmitFunc func(transactionId uuid.UUID, petId uint32, amount uint16) error
	GainClosenessFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error
	EvolveAndEmitFunc        func(transactionId uuid.UUID, petId uint32) error
	EvolveFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error
	RenameAndEmitFunc        func(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error
	RenameFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error
}

func (m *ProcessorMock) GainClosenessAndEmit(transactionId uuid.UUID, petId uint32, amount uint16) error {
	if m.GainClosenessAndEmitFunc != nil {
		return m.GainClosenessAndEmitFunc(transactionId, petId, amount)
	}
	return nil
}

func (m *ProcessorMock) GainCloseness(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error {
	if m.GainClosenessFunc != nil {
		return m.GainClosenessFunc(mb)
	}
	return func(uuid.UUID, uint32, uint16) error { return nil }
}

func (m *ProcessorMock) EvolveAndEmit(transactionId uuid.UUID, petId uint32) error {
	if m.EvolveAndEmitFunc != nil {
		return m.EvolveAndEmitFunc(transactionId, petId)
	}
	return nil
}

func (m *ProcessorMock) Evolve(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error {
	if m.EvolveFunc != nil {
		return m.EvolveFunc(mb)
	}
	return func(uuid.UUID, uint32) error { return nil }
}

func (m *ProcessorMock) RenameAndEmit(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error {
	if m.RenameAndEmitFunc != nil {
		return m.RenameAndEmitFunc(transactionId, petId, characterId, name)
	}
	return nil
}

func (m *ProcessorMock) Rename(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error {
	if m.RenameFunc != nil {
		return m.RenameFunc(mb)
	}
	return func(uuid.UUID, uint32, uint32, string) error { return nil }
}
```

Add the compile-time assertion the sibling mocks carry, if they carry one (`var _ pet.Processor = (*ProcessorMock)(nil)`).

- [ ] **Step 2: Write the failing test**

Create `saga/pet_name_tag_compensation_test.go`, directly modelled on `meso_sack_compensation_test.go` (note the `//go:build test` tag — the file will not compile into the default build without it):

```go
//go:build test

package saga

import (
	petmock "atlas-saga-orchestrator/pet/mock"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	petNameTagCharId   = uint32(77002)
	petNameTagItemId   = uint32(5170000)
	petNameTagPetId    = uint32(4242)
	petNameTagNewName  = "Renamed"
	petNameTagOldName  = "Original"
)

func newPetNameTagSaga(t *testing.T, tx uuid.UUID, renameStatus Status) Saga {
	t.Helper()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(PetNameTagUse).
		SetInitiatedBy("pet-name-tag-compensation-test").
		AddStep("rename_pet", renameStatus, RenamePet, RenamePetPayload{
			CharacterId:  petNameTagCharId,
			PetId:        petNameTagPetId,
			Name:         petNameTagNewName,
			PreviousName: petNameTagOldName,
		}).
		AddStep("consume_pet_name_tag", Failed, DestroyAsset, DestroyAssetPayload{
			CharacterId: petNameTagCharId,
			TemplateId:  petNameTagItemId,
			Quantity:    1,
			RemoveAll:   false,
		}).
		Build()
	require.NoError(t, err)
	return s
}

type renameCall struct {
	PetId       uint32
	CharacterId uint32
	Name        string
}

func recordingPetMock(calls *[]renameCall) *petmock.ProcessorMock {
	return &petmock.ProcessorMock{
		RenameAndEmitFunc: func(_ uuid.UUID, petId uint32, characterId uint32, name string) error {
			*calls = append(*calls, renameCall{petId, characterId, name})
			return nil
		},
	}
}

// A failed consume_pet_name_tag must revert the pet's name to the PreviousName
// captured at saga-build time — exactly once (PRD FR-7.4). Without this the
// player is told the rename failed while the pet keeps the new name.
func TestPetNameTagCompensationRevertsName(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	var calls []renameCall
	s := newPetNameTagSaga(t, uuid.New(), Completed)
	NewCompensator(logger, testTenantContext()).
		WithPetProcessor(recordingPetMock(&calls)).
		DispatchPetNameTagRollbacks(s)

	require.Len(t, calls, 1, "completed rename must be reverted exactly once")
	assert.Equal(t, petNameTagPetId, calls[0].PetId)
	assert.Equal(t, petNameTagCharId, calls[0].CharacterId)
	assert.Equal(t, petNameTagOldName, calls[0].Name, "revert must carry PreviousName")
}

// A rename step that never completed applied nothing and has no inverse. Issuing
// a revert here would overwrite a name the saga never set.
func TestPetNameTagCompensationSkipsUncompletedRename(t *testing.T) {
	logger, _ := test.NewNullLogger()

	var calls []renameCall
	s := newPetNameTagSaga(t, uuid.New(), Failed)
	NewCompensator(logger, testTenantContext()).
		WithPetProcessor(recordingPetMock(&calls)).
		DispatchPetNameTagRollbacks(s)

	assert.Empty(t, calls, "an uncompleted rename must not be reverted")
}
```

If `NewBuilder()`/`AddStep` in this package have a different arity than shown, copy the exact spelling from `newMesoSackSaga` in `meso_sack_compensation_test.go` — that helper is the ground truth for the builder API.

- [ ] **Step 3: Run test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags test ./saga/ -run PetNameTag -v`
Expected: FAIL — `undefined: DispatchPetNameTagRollbacks` / `WithPetProcessor`.

(The `-tags test` flag matches how `meso_sack_compensation_test.go` is built; confirm the module's actual test invocation — if `go test ./...` already passes that tag via a Makefile or a `go.work` setting, use the project's normal command instead.)

- [ ] **Step 4: Add the rollback dispatcher and the injector**

First add the field and injector so the test can inject the mock. In `compensator.go`, add `petP pet.Processor` to `CompensatorImpl`, `petP: pet.NewProcessor(l, ctx)` to `NewCompensator`, `WithPetProcessor(petP pet.Processor) Compensator` to the `Compensator` interface, and the setter beside `WithCompartmentProcessor` (`:194`), copying that method's exact shape.

Then the dispatcher:

In `compensator.go`, add to the `Compensator` interface beside `DispatchMesoSackRollbacks`:

```go
	// DispatchPetNameTagRollbacks reverse-walks the completed steps of a
	// pet_name_tag_use saga and reverts the pet's name by re-issuing RENAME
	// with the PreviousName captured at build time. Pure dispatch half — no
	// lifecycle transitions, no Failed emission, no cache eviction.
	//
	// Known, accepted limitation: if some other actor renames the pet between
	// step 1 and this revert, the revert restores a stale name. A rename is
	// player-initiated, serialized by the client's exclusive-request gate, and
	// the window is one Kafka round trip — a compare-and-swap revert keyed on
	// the applied name buys almost nothing for real complexity (design §3.7).
	DispatchPetNameTagRollbacks(s Saga)
```

Implement it beside `DispatchMesoSackRollbacks`: iterate `s.Steps()` in reverse, and for a `Completed` step whose `Action() == RenamePet` with a `RenamePetPayload`, call `c.petP.RenameAndEmit(s.TransactionId(), payload.PetId, payload.CharacterId, payload.PreviousName)`. Follow the existing dispatcher's error-logging and fire-and-forget conventions exactly.

If `CompensatorImpl` has no `petP` field yet, add `petP pet.Processor` to the struct and `petP: pet.NewProcessor(l, ctx)` to `NewCompensator`, matching the existing field wiring.

- [ ] **Step 5: Add the whole-saga compensator**

Beside `compensateMesoSackUse`, add `compensatePetNameTagUse` with the identical sequence — `DispatchPetNameTagRollbacks`, `TryTransition(SagaLifecycleCompensating → SagaLifecycleFailed)` guard, `SagaTimers().Cancel`, `GetCache().Remove`, `EmitSagaFailed(..., sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId())` — and route to it from `CompensateFailedStep`:

```go
	if s.SagaType() == PetNameTagUse {
		return c.compensatePetNameTagUse(s, failedStep)
	}
```

Add `petNameTagCharacterId(s Saga) uint32` in the shape of `mesoSackCharacterId`, preferring the `RenamePet` step's `CharacterId` and falling back to `ExtractCharacterId` over all steps.

- [ ] **Step 6: Classify the saga type for timeouts**

In `timer.go`:
- add `PetNameTagUse,` to `reverseWalkSagaTypes`
- add `PetNameTagUse,` to `allSagaTypes`
- add the dispatch arm to `dispatchTimeoutRollbacks`:

```go
	case PetNameTagUse:
		// Without this a timed-out rename leaves the new name applied while the
		// tag was never consumed — or, on the other ordering, the player's pet
		// keeps a name they were told failed.
		c.DispatchPetNameTagRollbacks(s)
```

`TestEverySagaTypeIsClassified` fails if the type is added to `allSagaTypes` without a classification — that is the intended gate.

- [ ] **Step 7: Handle the character-id resolution in the failed-event producer**

`saga/producer.go:188-193` special-cases saga types that carry no CharacterCreation ids. A `pet_name_tag_use` saga is in the same position as `meso_sack_use`. Extend that condition:

```go
	if s.SagaType() == MesoSackUse || s.SagaType() == PetNameTagUse {
```

Read the surrounding block before editing and follow whatever resolution it performs for `MesoSackUse` — if it calls `mesoSackCharacterId`, branch to `petNameTagCharacterId` for the new type.

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./...`
Expected: PASS, clean.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-saga-orchestrator/
git commit -m "feat(task-224): pet_name_tag_use compensation and timeout classification"
```

---

## Task 12: atlas-channel Kafka mirrors and the cross-service round-trip guard

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/pet/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/saga/model.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/pet/contract_mirror_test.go`

**Interfaces:**
- Consumes: the Task 5 contract as the authoritative shape.
- Produces: channel-side `CommandPetRename`, `StatusEventTypeNameChanged`, `RenameCommandBody`, `NameChangedStatusEventBody`; `SagaTypePetNameTagUse = "pet_name_tag_use"`; channel saga re-exports `PetNameTagUse`, `RenamePet`, `RenamePetPayload`.

The pet contract is mirrored across five modules and has **no mirror guard**. A json-tag typo fails no build and decodes to a zero-valued body at runtime. This task's test is the substitute (design §3.6). `atlas-consumables` and `atlas-messages` carry subset mirrors that need no change.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/kafka/message/pet/contract_mirror_test.go`:

```go
package pet

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// The pet Kafka contract is duplicated across atlas-pets (owner), atlas-channel,
// and atlas-saga-orchestrator, in three separate Go modules with no mirror guard
// (only trade has one). A field name or json tag changed in one and not the
// others fails NO build — it decodes into a zero-valued body at runtime,
// silently. These fixtures are byte-for-byte what atlas-pets' structs marshal;
// if this side drifts, the assertions below go red.
func TestNameChangedStatusEventBodyDecodesOwnerWire(t *testing.T) {
	txn := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	wire := []byte(`{"petId":7,"ownerId":42,"type":"NAME_CHANGED","body":{"slot":0,"name":"Renamed","previousName":"Original","transactionId":"11111111-2222-3333-4444-555555555555"}}`)

	var e StatusEvent[NameChangedStatusEventBody]
	if err := json.Unmarshal(wire, &e); err != nil {
		t.Fatalf("Unmarshal = %v", err)
	}

	if e.PetId != 7 || e.OwnerId != 42 || e.Type != StatusEventTypeNameChanged {
		t.Fatalf("envelope drifted: %+v", e)
	}
	if e.Body.Slot != 0 || e.Body.Name != "Renamed" || e.Body.PreviousName != "Original" || e.Body.TransactionId != txn {
		t.Fatalf("body drifted: %+v", e.Body)
	}
}

func TestRenameCommandBodyDecodesOwnerWire(t *testing.T) {
	wire := []byte(`{"transactionId":"11111111-2222-3333-4444-555555555555","actorId":42,"petId":7,"type":"RENAME","body":{"name":"Renamed"}}`)

	var c Command[RenameCommandBody]
	if err := json.Unmarshal(wire, &c); err != nil {
		t.Fatalf("Unmarshal = %v", err)
	}
	if c.ActorId != 42 || c.PetId != 7 || c.Type != CommandPetRename || c.Body.Name != "Renamed" {
		t.Fatalf("drifted: %+v", c)
	}
}
```

Add the same two tests to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/pet/` (adjusting the local constant identifiers if that package spells them differently — the wire fixture must stay byte-identical).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/pet/ -v`
Expected: FAIL — `undefined: NameChangedStatusEventBody`.

- [ ] **Step 3: Mirror the pet contract**

Copy the Task 5 constants and structs into the channel's `kafka/message/pet/kafka.go`, keeping the string values and json tags byte-identical.

- [ ] **Step 4: Add the saga type constant and re-exports**

In `channel/kafka/message/saga/kafka.go`, beside `SagaTypeMesoSackUse`:

```go
	SagaTypePetNameTagUse    = "pet_name_tag_use"
```

In `channel/saga/model.go`, add `PetNameTagUse = sharedsaga.PetNameTagUse` to the type block, `RenamePet = sharedsaga.RenamePet` to the action block, and `RenamePetPayload = sharedsaga.RenamePetPayload` to the payload alias block.

- [ ] **Step 5: Run tests to verify they pass**

Run:
```bash
cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/... -v && go build ./...
cd ../../../atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./kafka/message/... -v
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/ services/atlas-channel/atlas.com/channel/saga/model.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/pet/
git commit -m "feat(task-224): mirror pet rename contract into channel and orchestrator with drift tests"
```

---

## Task 13: atlas-channel dispatch — 517 predicate fix, type-17 constant and arm, handler

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_pet_name_tag.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_pet_name_tag_test.go`

**Interfaces:**
- Consumes: `cashsb.NewItemUsePetNameTag` (Task 2), `petconst.NormalizeName`/`ValidateName` (Task 1), `saga.PetNameTagUse`/`RenamePet`/`RenamePetPayload` (Task 12).
- Produces:
  - `const CashSlotItemTypePetNameTag = CashSlotItemType(17)`
  - `var petsForOwnerFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]pet.Model, error)` (test seam)
  - `func buildPetNameTagUseSaga(transactionId uuid.UUID, now time.Time, characterId uint32, itemId item.Id, petId uint32, name string, previousName string) saga.Saga`
  - `func handlePetNameTagUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, name string)`

- [ ] **Step 1: Write the failing predicate test**

Add to the existing cash-item-use handler test file (or create `character_cash_item_use_pet_name_tag_test.go` if that is where the arm's tests belong):

```go
// FR-1.2/FR-1.3. The pre-fix predicate was `10000*itemId/10000 != itemId`.
// item.Id is uint32, so 10000 * 5170000 = 51,700,000,000 wraps to 160,392,448;
// /10000 = 16039 != 5170000, and the branch returned 0 — the Pet Name Tag never
// reached a handler at all. The client's actual rule is `itemId % 10000 != 0`
// (get_cashslot_item_type @0x48645b, case 517). THIS TEST FAILS AGAINST THE
// PRE-FIX PREDICATE — that is the point of it.
func TestGetCashSlotItemTypePetNameTag(t *testing.T) {
	ctx := testTenantContext(t) // reuse this package's existing tenant-context helper
	tm := tenant.MustFromContext(ctx)

	if got := GetCashSlotItemType(tm)(5170000); got != CashSlotItemTypePetNameTag {
		t.Fatalf("GetCashSlotItemType(5170000) = %d, want 17", got)
	}
	if got := GetCashSlotItemType(tm)(5170001); got != CashSlotItemType(0) {
		t.Fatalf("GetCashSlotItemType(5170001) = %d, want 0", got)
	}
}
```

Substitute this package's real tenant-context helper and the real `GetCashSlotItemType` argument shape (read the function signature in `character_cash_item_use.go` before writing the call).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run PetNameTag -v`
Expected: FAIL — `GetCashSlotItemType(5170000) = 0, want 17`.

- [ ] **Step 3: Fix the predicate and name the constant**

In `character_cash_item_use.go`, replace the `ClassificationPetImprints` branch:

```go
		if category == item.ClassificationPetImprints {
			// get_cashslot_item_type @0x48645b, case 517:
			//   return a1 % 10000 != 0 ? 0 : 17;
			// The previous spelling of this was `10000*itemId/10000 != itemId`,
			// which OVERFLOWS: item.Id is uint32, so 10000 * 5170000 wraps to
			// 160,392,448 and the branch returned 0 for the one item id it was
			// supposed to admit. The Pet Name Tag never reached a handler.
			if itemId%10000 != 0 {
				return CashSlotItemType(0)
			}
			return CashSlotItemTypePetNameTag
		}
```

Add to the `CashSlotItemType` const block:

```go
	// CashSlotItemTypePetNameTag is classification 517 (Pet Name Tag, 5170000).
	// No other classification maps to 17 in GetCashSlotItemType — meso sacks
	// return 19 on every version by deliberate Atlas policy (see
	// CashSlotItemTypeCurrencySack above) even though the v48 client's own
	// table says 17 — so gating the arm on `it` alone is unambiguous.
	CashSlotItemTypePetNameTag = CashSlotItemType(17)
```

- [ ] **Step 4: Run the predicate test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestGetCashSlotItemTypePetNameTag -v`
Expected: PASS.

- [ ] **Step 5: Write the failing handler test**

Append to `character_cash_item_use_pet_name_tag_test.go`, driving the arm through the package-var seams (`cashItemInSlotFunc`, and the new `petsForOwnerFunc`) exactly as the meso-sack handler tests drive theirs:

```go
// The happy path builds exactly one saga, rename-first, with PreviousName
// captured (PRD FR-7.2/FR-7.4).
func TestHandlePetNameTagUseBuildsRenameFirstSaga(t *testing.T) {
	s := buildPetNameTagUseSaga(uuid.New(), time.Now(), 42, 5170000, 7, "Renamed", "Original")

	if s.SagaType != saga.PetNameTagUse {
		t.Fatalf("SagaType = %s", s.SagaType)
	}
	if len(s.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(s.Steps))
	}
	if s.Steps[0].StepId != "rename_pet" || s.Steps[0].Action != saga.RenamePet {
		t.Fatalf("step 1 = %s/%s, want rename_pet/rename_pet", s.Steps[0].StepId, s.Steps[0].Action)
	}
	if s.Steps[1].StepId != "consume_pet_name_tag" || s.Steps[1].Action != saga.DestroyAsset {
		t.Fatalf("step 2 = %s/%s, want consume_pet_name_tag/destroy_asset", s.Steps[1].StepId, s.Steps[1].Action)
	}
	p, ok := s.Steps[0].Payload.(saga.RenamePetPayload)
	if !ok {
		t.Fatalf("step 1 payload type = %T", s.Steps[0].Payload)
	}
	if p.PetId != 7 || p.CharacterId != 42 || p.Name != "Renamed" || p.PreviousName != "Original" {
		t.Fatalf("payload = %+v", p)
	}
}

// Every rejection consumes nothing and starts no saga, and unlocks the client
// (PRD FR-7.3). Two announces per rejection: the pink text, then enable-actions.
func TestHandlePetNameTagUseRejectsAndUnlocks(t *testing.T) {
	const characterId = uint32(4242)

	cases := []struct {
		name    string
		pets    []pet.Model
		err     error
		newName string
	}{
		{"pet lookup failure", nil, errors.New("503"), "Renamed"},
		{"no active pet", nil, nil, "Renamed"},
		{"no lead pet", petsAtSlots(characterId, 1, 2), nil, "Renamed"},
		{"name too short", leadPetNamed(characterId, "Original"), nil, "abc"},
		{"name too long", leadPetNamed(characterId, "Original"), nil, "abcdefghijklm"},
		{"whitespace only", leadPetNamed(characterId, "Original"), nil, "     "},
		{"unchanged name", leadPetNamed(characterId, "Original"), nil, "Original"},
		{"unchanged after trim", leadPetNamed(characterId, "Original"), nil, "  Original  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restorePets := installPetsForOwnerSeam(t, tc.pets, tc.err)
			defer restorePets()
			captured, restoreProducer := installCapturingProducer()
			defer restoreProducer()

			s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
			defer cleanup()

			rec := &gaugeProducerRecorder{}
			handlePetNameTagUse(logrus.New(), ctx, rec.producer())(s, item.Id(5170000), tc.newName)

			if got := len((*captured)[sagaMsg.EnvCommandTopic]); got != 0 {
				t.Fatalf("emitted %d saga commands, want 0 (nothing may be consumed)", got)
			}
			if rec.calls != 2 {
				t.Fatalf("announced %d packets, want 2 (pink text + enable-actions unlock)", rec.calls)
			}
		})
	}
}

// The happy path emits exactly one saga command and announces nothing: the
// success unlock rides on the consume step's INVENTORY_OPERATION. An extra
// empty StatChanged here would race it.
func TestHandlePetNameTagUseCreatesSagaAndAnnouncesNothing(t *testing.T) {
	const characterId = uint32(4242)

	restorePets := installPetsForOwnerSeam(t, leadPetNamed(characterId, "Original"), nil)
	defer restorePets()
	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	handlePetNameTagUse(logrus.New(), ctx, rec.producer())(s, item.Id(5170000), "Renamed")

	if got := len((*captured)[sagaMsg.EnvCommandTopic]); got != 1 {
		t.Fatalf("emitted %d saga commands, want exactly 1", got)
	}
	if rec.calls != 0 {
		t.Fatalf("announced %d packets on the success path, want 0", rec.calls)
	}
}

// installPetsForOwnerSeam swaps the package-var pet lookup, mirroring
// installCashItemDataSeam in character_cash_item_use_meso_sack_test.go.
func installPetsForOwnerSeam(t *testing.T, ps []pet.Model, err error) func() {
	t.Helper()
	original := petsForOwnerFunc
	petsForOwnerFunc = func(logrus.FieldLogger, context.Context, uint32) ([]pet.Model, error) {
		return ps, err
	}
	return func() { petsForOwnerFunc = original }
}

func leadPetNamed(ownerId uint32, name string) []pet.Model {
	return []pet.Model{
		pet.NewModelBuilder(7, 0, 5000000, name).SetOwnerID(ownerId).SetSlot(0).MustBuild(),
	}
}

func petsAtSlots(ownerId uint32, slots ...int8) []pet.Model {
	ps := make([]pet.Model, 0, len(slots))
	for i, sl := range slots {
		ps = append(ps, pet.NewModelBuilder(uint32(100+i), 0, 5000000, "Other").SetOwnerID(ownerId).SetSlot(sl).MustBuild())
	}
	return ps
}
```

`installCapturingProducer`, `newCashItemUseTestSession`, and `gaugeProducerRecorder` already exist in `character_cash_item_use_meso_sack_test.go` — reuse them, do not redefine. `pet.NewModelBuilder(...).SetOwnerID(...).SetSlot(...).MustBuild()` is the channel-side builder used by `kafka/consumer/pet/consumer.go:224-230`; confirm its exact arity before writing (CLAUDE.md: use the Builder pattern, never a `*_testhelpers.go` file).

- [ ] **Step 6: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run PetNameTagUse -v`
Expected: FAIL — `undefined: buildPetNameTagUseSaga`.

- [ ] **Step 7: Write the handler**

Create `character_cash_item_use_pet_name_tag.go`:

```go
package handler

import (
	"atlas-channel/pet"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	petconst "github.com/Chronicle20/atlas/libs/atlas-constants/pet"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
)

// petsForOwnerFunc is a test seam for the pet lookup (package-var injection
// precedent: cashItemInSlotFunc and cashItemDataFunc in this package).
var petsForOwnerFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]pet.Model, error) {
	return pet.NewProcessor(l, ctx).GetByOwner(characterId)
}

// buildPetNameTagUseSaga assembles the pet_name_tag_use saga: RENAME FIRST,
// consume second. This is the deliberate inverse of meso_sack_use's
// consume-then-award ordering (PRD FR-7.2) — a rename that fails must not cost
// the player a cash item. PreviousName rides the payload so the compensator can
// revert the name if the consume step later fails (FR-7.4); atlas-channel
// already read the pet to resolve the target, so capturing it is free.
//
// The tag is consumed by TEMPLATE, not by slot: the pre-branch guard in
// CharacterCashItemUseHandleFunc already proved the named CASH slot holds this
// template, and the orchestrator's inverse for DestroyAsset is a plain
// RequestCreateItem into the first free CASH slot.
func buildPetNameTagUseSaga(transactionId uuid.UUID, now time.Time, characterId uint32, itemId item.Id, petId uint32, name string, previousName string) saga.Saga {
	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.PetNameTagUse,
		InitiatedBy:   "CASH_ITEM_USE",
		Steps: []saga.Step{
			{
				StepId: "rename_pet",
				Status: saga.Pending,
				Action: saga.RenamePet,
				Payload: saga.RenamePetPayload{
					CharacterId:  characterId,
					PetId:        petId,
					Name:         name,
					PreviousName: previousName,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				StepId: "consume_pet_name_tag",
				Status: saga.Pending,
				Action: saga.DestroyAsset,
				Payload: saga.DestroyAssetPayload{
					CharacterId: characterId,
					TemplateId:  uint32(itemId),
					Quantity:    1,
					RemoveAll:   false,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

// handlePetNameTagUse implements the CashSlotItemType 17 arm: classification 517
// Pet Name Tags (5170000).
//
// WHICH PET. The request carries no pet identifier — the case-17 arm of
// CWvsContext::SendConsumeCashItemUseRequest performs exactly one Encode
// (EncodeStr @0xa0bcb5) and never calls the pet-picker SetUtilDlgEx_Pet
// (@0x9acb27). The server must resolve the target itself, and the rule is the
// character's LEAD pet (Slot() == 0). That is not an Atlas policy invention: the
// client's own arm calls sub_46D2D5(this, 0) @0xa0ba47, which resolves the CASH
// item backing pet-locker index 0 — Atlas's slot 0 (design OQ-4, PRD FR-3.1).
// The two therefore agree by derivation, not by coincidence.
//
// When locker index 0 is empty the unmodified client abandons the arm before any
// dialog and sends nothing (cmp esi,ebx / jz def_A0A6E6 @0xa0ba4e), so the
// no-lead-pet rejection below is a crafted-packet path — rare, but it must still
// fail closed.
//
// Nothing here warps, so a plain enable-actions StatChanged is the correct
// unlock on every rejection path (reference_exclrequest_unlock_contract).
func handlePetNameTagUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, name string) {
	return func(s session.Model, itemId item.Id, name string) {
		enableActions := func() {
			_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		}
		reject := func(msg string) {
			_ = session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePinkTextBody("", "", msg))(s)
			enableActions()
		}

		ps, err := petsForOwnerFunc(l, ctx, s.CharacterId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] used pet name tag [%d] but their pets could not be resolved. Rejecting; nothing consumed.", s.CharacterId(), itemId)
			reject("You are unable to use this item right now.")
			return
		}

		var target pet.Model
		var found bool
		for _, p := range ps {
			if p.Slot() == 0 {
				target = p
				found = true
				break
			}
		}
		if !found {
			l.Warnf("Character [%d] used pet name tag [%d] with no lead pet. Rejecting; nothing consumed.", s.CharacterId(), itemId)
			reject("You must have a pet out to use this.")
			return
		}
		// Belt and braces over a processor that already filters by owner.
		if target.OwnerId() != s.CharacterId() {
			l.Warnf("Character [%d] used pet name tag [%d] but resolved pet [%d] is owned by [%d]. Rejecting; nothing consumed.", s.CharacterId(), itemId, target.Id(), target.OwnerId())
			reject("You are unable to use this item right now.")
			return
		}

		normalized := petconst.NormalizeName(name)
		if verr := petconst.ValidateName(normalized); verr != nil {
			l.WithError(verr).Warnf("Character [%d] used pet name tag [%d] on pet [%d] with an invalid name [%s]. Rejecting; nothing consumed.", s.CharacterId(), itemId, target.Id(), name)
			reject("That name cannot be used.")
			return
		}
		if normalized == target.Name() {
			l.Warnf("Character [%d] used pet name tag [%d] on pet [%d] without changing the name. Rejecting; nothing consumed.", s.CharacterId(), itemId, target.Id())
			reject("That is already your pet's name.")
			return
		}

		l.Debugf("Character [%d] renaming pet [%d] from [%s] to [%s] with tag [%d].", s.CharacterId(), target.Id(), target.Name(), normalized, itemId)
		_ = saga.NewProcessor(l, ctx).Create(buildPetNameTagUseSaga(uuid.New(), time.Now(), s.CharacterId(), itemId, target.Id(), normalized, target.Name()))
	}
}
```

- [ ] **Step 8: Add the dispatch arm**

In `character_cash_item_use.go`, beside the `CashSlotItemTypeCurrencySack` arm:

```go
		if it == CashSlotItemTypePetNameTag {
			sp := cashsb.NewItemUsePetNameTag(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			handlePetNameTagUse(l, ctx, wp)(s, itemId, sp.Name())
			return
		}
```

Placement matters only for readability — the `if it == ...` chain is exclusive by construction and type 17 is unambiguous.

- [ ] **Step 9: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -v`
Expected: PASS, including the pre-existing arms' tests.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/
git commit -m "fix(task-224): correct 517 cash-slot predicate overflow; add pet name tag arm"
```

---

## Task 14: atlas-channel broadcast, writer registration, and failure rendering

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go`

**Interfaces:**
- Consumes: `petpkt.PetNameChangedWriter` / `petpkt.NewPetNameChanged` (Task 3); `pet2.StatusEventTypeNameChanged` / `NameChangedStatusEventBody` (Task 12); `saga.SagaTypePetNameTagUse` (Task 12).
- Produces:
  - `func handleNameChanged(sc server.Model, wp writer.Producer) message.Handler[pet2.StatusEvent[pet2.NameChangedStatusEventBody]]`
  - `func petNameTagFailureMessage(errorCode string) string`

- [ ] **Step 1: Add the broadcast consumer**

In `kafka/consumer/pet/consumer.go`, register `handleNameChanged` in `InitHandlers` beside the other status-event handlers, and add:

```go
func handleNameChanged(sc server.Model, wp writer.Producer) message.Handler[pet2.StatusEvent[pet2.NameChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e pet2.StatusEvent[pet2.NameChangedStatusEventBody]) {
		if e.Type != pet2.StatusEventTypeNameChanged {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.OwnerId)
		if err != nil {
			return
		}

		// ForSessionsInMap includes the owner, so one call reaches both the
		// renaming player and every observer. The callback closes over
		// immutable values only — the iteration is PARALLEL
		// (bug_channel_foreachinmap_parallel_shared_state), so nothing shared
		// may be mutated inside it.
		//
		// Observers who enter the map LATER get the new name from the
		// PetActivated spawn body instead; both codecs write the same
		// NameTagLayer, so the decoration does not flicker between the two.
		_ = _map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(),
			session.Announce(l)(ctx)(wp)(petpkt.PetNameChangedWriter)(
				petpkt.NewPetNameChanged(e.OwnerId, e.Body.Slot, e.Body.Name).Encode))
	}
}
```

Match the surrounding handlers' exact registration idiom and import aliases (`petpkt`, `_map`, `session`, `tenant`) — read `handleCommandResponse` before writing.

Note: the success path is unlocked by the consume step's `INVENTORY_OPERATION`, not here — do not add an enable-actions announce to this handler.

- [ ] **Step 2: Register the writer**

In `main.go`, add to the writer list beside the other pet writers (`main.go:724-732`):

```go
		petcb.PetNameChangedWriter,
```

- [ ] **Step 3: Render the failure**

In `kafka/consumer/saga/consumer.go`, after the meso-sack arm:

```go
		// Pet-name-tag failures: the orchestrator's compensator has already
		// reverted the pet's name (and refunded nothing — the tag is consumed
		// only after the rename succeeds). Tell the player why, then release the
		// client's exclusive-request gate. Nothing in this flow warps, so a
		// plain enable-actions is the correct unlock.
		if e.Body.SagaType == saga.SagaTypePetNameTagUse {
			msg := petNameTagFailureMessage(e.Body.ErrorCode)
			err = session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePinkTextBody("", "", msg))(s)
			if err != nil {
				l.WithError(err).WithField("character_id", e.Body.CharacterId).Error("Failed to send pet-name-tag pink text.")
			}
			err = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
			if err != nil {
				l.WithError(err).WithField("character_id", e.Body.CharacterId).Error("Failed to send enable-actions after pet-name-tag failure.")
			}
			return
		}
```

and, beside `mesoSackFailureMessage`:

```go
// petNameTagFailureMessage maps a pet_name_tag_use saga's errorCode to the pink
// text the player sees. The saga can fail on the rename step, on the consume
// step, or by timeout, and no atlas-pets error code names a player-actionable
// cause today — so the generic message is the honest one. Add specific arms here
// when (and only when) a service starts supplying a machine-readable code.
func petNameTagFailureMessage(errorCode string) string {
	return "You are unable to rename your pet right now."
}
```

Match the parameter usage convention the linter expects — if an unused parameter trips `tools/lint.sh`, name it `_` and note why in the comment.

- [ ] **Step 4: Verify the build and tests**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... && go vet ./...`
Expected: clean, PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/ services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-224): broadcast PET_NAMECHANGE and render rename failures"
```

---

## Task 15: Tenant socket templates — register the `PetNameChanged` writer

**Files:**
- Modify: 9 files under `services/atlas-configurations/seed-data/templates/` — `template_gms_61_1.json`, `template_gms_72_1.json`, `template_gms_79_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`
- NOT modified: `template_gms_48_1.json` (op absent from the v48 registry — recorded `n-a`), `template_gms_12_1.json`

**Interfaces:**
- Consumes: `PetNameChangedWriter` (Task 3), the Task 4 opcode confirmation.
- Produces: no Go symbols.

- [ ] **Step 1: Insert the writer entry in each template**

Add exactly one entry per template to the `writers` array, **at its sorted `opCode` position** (`tools/template-opcode-order-guard.sh` enforces strict ascending order; appending it next to the other pet writers is what the guard exists to reject):

```json
      {
        "opCode": "<per-version>",
        "writer": "PetNameChanged",
        "fname": "CPet::OnNameChanged",
        "services": [
          "channel"
        ]
      },
```

Per-version `opCode` (confirmed against `docs/packets/registry/*.yaml` at plan time; Task 4 Step 1 re-confirms):

| Template | opCode |
|---|---|
| `template_gms_61_1.json` | `"0x83"` |
| `template_gms_72_1.json` | `"0x9D"` |
| `template_gms_79_1.json` | `"0xA1"` |
| `template_gms_83_1.json` | `"0xAC"` |
| `template_gms_84_1.json` | `"0xB0"` |
| `template_gms_87_1.json` | `"0xB9"` |
| `template_gms_92_1.json` | `"0xC8"` |
| `template_gms_95_1.json` | `"0xCB"` |
| `template_jms_185_1.json` | `"0xB2"` |

Use each template's **own** hex spelling convention — v83 writes `"0xAB"` (two digits, no leading zero) for `PetChat`, so v83's new entry is `"0xAC"`, not `"0x0AC"`. Mismatched zero-padding is what created the duplicate-binding class the second guard bans (`tools/template-duplicate-binding-guard.sh`).

In `template_gms_83_1.json` the slot is already free: `PetChat` sits at `0xAB` and `PetExcludeResponse` at `0xAD`, so the new entry goes between them.

Do **not** touch the serverbound `handlers` array — the `CharacterCashItemUseHandle` binding already exists in every template and a second binding for the same `(implementation, opCode)` pair is banned.

- [ ] **Step 2: Verify each file is still valid JSON**

Run:
```bash
for f in services/atlas-configurations/seed-data/templates/template_*.json; do python3 -m json.tool "$f" > /dev/null || echo "INVALID: $f"; done
```
Expected: no output.

- [ ] **Step 3: Verify one entry per template, each with an `fname`**

Run:
```bash
grep -c '"PetNameChanged"' services/atlas-configurations/seed-data/templates/template_*.json
grep -A2 '"PetNameChanged"' services/atlas-configurations/seed-data/templates/template_*.json | grep -c 'CPet::OnNameChanged'
```
Expected: exactly `1` for each of the nine modified templates, `0` for `template_gms_12_1.json` and `template_gms_48_1.json`; and `9` from the second command. A writer without an `fname` is silently dropped at seed time (`bug_seed_template_writers_require_fname`).

- [ ] **Step 4: Run the template guards**

Run from the worktree root:
```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```
Expected: all three exit 0.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(task-224): register PetNameChanged writer in nine tenant templates"
```

---

## Task 16: Documentation, rollout note, and full verification gate

**Files:**
- Create: `docs/tasks/task-224-pet-name-tag/rollout.md`
- Modify: `docs/research/missing-features/items-and-consumables.md`
- Modify: `docs/research/missing-features/packet-gap-inference.md`

**Interfaces:**
- Consumes: everything above.
- Produces: no code.

- [ ] **Step 1: Write the rollout note**

Create `docs/tasks/task-224-pet-name-tag/rollout.md`:

```markdown
# task-224 Rollout — Pet Name Tag

## Prerequisite: none

Unlike task-220 (Meso Sack), this feature reads **no WZ value**. `Item.wz/Cash/0517.img.xml`
carries only `z`, `slotMax`, `cash` and icon canvases — no payout node, nothing to ingest.
The rename payload is player-supplied and the consumption is template-keyed, so there is no
re-ingest step and no version-parity survey to run.

## Required: reconcile live tenant socket configs

The nine seed templates gained a `PetNameChanged` writer at each version's
`PET_NAMECHANGE` opcode. **A live tenant whose socket config predates this change
will silently drop the new writer** — the packet is simply never emitted, with no
error anywhere (`bug_new_opcodes_not_in_live_tenant_config`).

For every live tenant, reconcile its socket configuration to the updated template
for its version before announcing the feature. Verify per tenant that the
`writers` array contains a `PetNameChanged` entry, at the correct opcode, with a
non-empty `fname`.

## Post-deploy verification

1. Use a Pet Name Tag with a pet summoned — the new name renders immediately for
   the player and for another character standing in the map.
2. Have a third character enter the map afterwards — the spawn body carries the
   new name.
3. Relog and change channel — the name persists.
4. Despawn and respawn the pet — the name tag decoration does not flicker
   (`NameTagLayer` is shared by both codecs).
5. Attempt a 3-character and a 13-character name — both are rejected with pink
   text, the client unlocks, and the tag remains in the cash slot.
```

- [ ] **Step 2: Update the missing-features docs**

In `docs/research/missing-features/items-and-consumables.md`, update the `5170000` / classification-517 entry (around line 40) to record it as implemented by task-224. In `docs/research/missing-features/packet-gap-inference.md`, update the `PET_NAMECHANGE` entry (around line 469) the same way. Read the surrounding entries first and match how other implemented items are marked in each file.

- [ ] **Step 3: Run every build and guard gate**

From the worktree root:

```bash
for m in libs/atlas-constants libs/atlas-packet libs/atlas-saga \
         services/atlas-pets/atlas.com/pets \
         services/atlas-channel/atlas.com/channel \
         services/atlas-saga-orchestrator/atlas.com/saga-orchestrator; do
  echo "== $m"; (cd "$m" && go build ./... && go vet ./... && go test -race ./...) || echo "FAILED: $m"
done

tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/lint.sh --check
```

Expected: every module clean, every guard exit 0. Run `tools/lint.sh` (no flags) first if `--check` reports formatting drift — it rewrites in place.

Note on `tools/lint.sh --check`: it false-fails without nvm on PATH (`bug_lint_check_false_fails_without_nvm`), and contends on a golangci-lint lock across worktrees. If it fails, confirm the environment before believing the failure.

- [ ] **Step 4: Confirm no `go.mod` moved**

Run: `git diff --name-only main... | grep 'go\.mod\|go\.sum' || echo "no module files changed"`

`libs/atlas-constants` is already a dependency of atlas-pets, atlas-channel, atlas-saga-orchestrator, and libs/atlas-packet (verified at plan time), so **no `go.mod` should move and no `docker buildx bake` round is required**. If this command lists any `go.mod`, that assumption broke — run `docker buildx bake atlas-<svc>` from the worktree root for each affected service before proceeding. That step is mandatory, not optional (CLAUDE.md §Build & Verification item 4).

- [ ] **Step 5: Commit**

```bash
git add docs/
git commit -m "docs(task-224): rollout note and missing-features updates"
```

- [ ] **Step 6: Code review before PR**

Run `superpowers:requesting-code-review` — it dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (Go files changed; no atlas-ui changes, so no frontend reviewer). Findings land in `docs/tasks/task-224-pet-name-tag/audit.md`. Do not open the PR before this step (CLAUDE.md §Code Review Before PR).

---

## Requirement Traceability

| Requirement | Task |
|---|---|
| FR-1.1 type-17 constant | 13 |
| FR-1.2 517 predicate overflow fix | 13 |
| FR-1.3 predicate unit test that fails pre-fix | 13 |
| FR-1.4 dispatch arm + sibling handler file | 13 |
| FR-1.5 existing cash-slot ownership guard retained | 13 (untouched pre-branch guard) |
| FR-2.1–2.3 serverbound sub-body codec | 2 |
| FR-2.4 per-version re-derivation | 4 |
| FR-3.1–3.4 lead-pet resolution, ownership, doc comment | 13 |
| FR-4.1–4.4 name bounds (4–12 per design C-2), trim, server-side enforcement | 1, 13 |
| FR-4.5 no profanity filter (explicit scope boundary) | — (design OQ-3; unchanged) |
| FR-5.1 RENAME command | 5 |
| FR-5.2 NAME_CHANGED event + producer | 5, 7 |
| FR-5.3 updateName administrator | 6 |
| FR-5.4 SetName builder | 6 |
| FR-5.5 idempotent rename | 6, 7 |
| FR-5.6 atlas-pets re-validates | 7 |
| FR-6.1–6.3 clientbound codec, body, framing | 3 |
| FR-6.4 writer registration in main.go | 14 |
| FR-6.5 event-driven map broadcast | 14 |
| FR-6.6 flag provenance (design A5: named constant, not config key) | 3 |
| FR-7.1 PetNameTagUse saga type | 9, 10, 12 |
| FR-7.2 rename-first ordering | 13 |
| FR-7.3 fail-closed pre-flight + enableActions | 13 |
| FR-7.4 revert on consume failure | 11 |
| FR-7.5 compensator/producer/model/timer registrations | 10, 11 |
| FR-7.6 no warp → plain enableActions unlock | 13, 14 |
| FR-8.1–8.4 nine templates, fname, sorted position, no duplicate handler | 15 |
| FR-8.5 live-tenant reconciliation documented | 16 |
| FR-9.1–9.4 matrix promotion, v48 n-a, serverbound verified, real derivation | 4 |
| Design C-1 JMS region gate | 3 (fixture is the regression test) |
| Design C-2 4–12 bounds | 1 |
| Design C-3 transactionId / slot / previousName on the bodies | 5 |
| Design §3.6 five-way mirror drift guard | 12 |
| Design §5 A5 nameTag shared with Activated | 3 |
| PRD §10 build & guard gates | 16 |
