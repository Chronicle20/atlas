# Skill Books and Mastery Books — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A player consuming a Skill Book (228) or Mastery Book (229) gets a server-validated success roll, atomic book-destruction + master-level update via a new `skill_book_use` saga, and a map-broadcast `SKILL_LEARN_ITEM_RESULT` packet.

**Architecture:** atlas-channel handler → `REQUEST_SKILL_BOOK_USE` on `COMMAND_TOPIC_CONSUMABLE` → atlas-consumables validates + rolls → two-step saga (`destroy_asset_from_slot` → conditional `create_skill`/`update_skill`) → consumables gates the result on a one-time `EVENT_TOPIC_SAGA_STATUS` handler → `SKILL_BOOK_RESULT` on `EVENT_TOPIC_CONSUMABLE_STATUS` → atlas-channel writer broadcasts. Compensation: a new saga-type-routed reverse walk in the orchestrator re-awards the book (via a new `TemplateId` field on `DestroyAssetFromSlotPayload`) when the skill step fails.

**Tech Stack:** Go workspace (`go.work`), Kafka via `libs/atlas-kafka`, JSON:API REST via `libs/atlas-rest`, packet codecs in `libs/atlas-packet`, sagas via `libs/atlas-saga` + atlas-saga-orchestrator.

**Spec:** `docs/tasks/task-125-skill-mastery-books/design.md` (approved). PRD: `prd.md`.

## Global Constraints

- No hard-coded opcodes anywhere; opcodes resolve from tenant config (handler/writer names only in code).
- Every seed-template handler entry MUST carry a `validator` (`LoggedInValidator` here) — a validator-less entry is silently dropped (`libs/atlas-opcodes/producer.go:47-51`).
- Success roll uses strictly-less-than: `roll < int32(successRate)` — `success=0` never passes, `success=100` always passes. Do NOT copy the scroll path's `<=` (deliberate; design §2 D-3).
- **Clientbound codec version gate (IDB-verified): prepend the `bOnExclRequest` byte when `MajorVersion() >= 84`.** v48/61/72/79/83 = 15-byte body (no leading byte); v84/87/92/95/jms = 16-byte body (leading `bOnExclRequest`, value 1). This is a real `v84 ≠ v83` exception — do NOT gate at `>= 87`, and do NOT treat v84 as v83 for this packet (v84 serverbound IS identical to v83, but its clientbound diverges). Serverbound codec has NO version gate (uniform body all versions).
- Immutable models: private fields + getters. No `*_testhelpers.go` files; table-driven tests.
- Multi-tenancy: producers/consumers use span+tenant header decorators/parsers (existing `producer.ProviderImpl` / `consumer.SetHeaderParsers` patterns).
- Verification gates before "done": `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed module; `docker buildx bake` for every touched service (libs changes ⇒ `all-go-services`); `tools/redis-key-guard.sh` clean.
- Never invent opcodes/addresses. An fname that fails to resolve in an IDB is a stop-and-ask, never a substitution.
- Commit after every task. All work happens in this worktree (`.worktrees/task-125-skill-mastery-books`, branch `task-125-skill-mastery-books`).

## Pre-verified facts (plan-time, do not re-derive)

- **WZ `success` sweep (design §2 D-3 verification item): CLEAN.** All 26 items in v83 `Item.wz/Consume/0228.img.xml` and all 139 in `0229.img.xml` carry explicit `success` (none 0), `masterLevel`, and non-empty `skill[]` nodes. Verified against the local v83 XML dump (`tmp/<tenant-uuid>/GMS/83.1/`). No unusable-at-0% books ship.
- **Sample fixture values (v83 WZ):** 2280000 (skill book): success 100, masterLevel 10, no reqSkillLevel node (→0), skills `[2121003]`. 2290000 (mastery book): success 70, masterLevel 20, reqSkillLevel 5, skills `[1121001, 1221001, 1321001]`. 2280002: multi-job skills `[1121011, 1221012, …, 21121008]`.
- **Opcodes (IDB-verified 2026-07-25, all 10 versions — hex).** Derived directly from each IDB by decompiling the send (`COutPacket(op)` ctor arg) and the receive (`CWvsContext::OnPacket` switch case; dispatch is base-0 in every version, so switch-case == raw wire opcode). **The audit CSV / `docs/packets/audits/support/*.md` are WRONG for several cells** (legacy serverbound marked `n-a`; `gms_48` clientbound `n-a`/delta "47/48") — trust the decompile.

  | version | `USE_SKILL_BOOK` (serverbound) | `SKILL_LEARN_ITEM_RESULT` (clientbound) | clientbound body |
  |---|---|---|---|
  | gms_48  | 0x40 | 0x2B | 15-byte |
  | gms_61  | 0x4B | 0x30 | 15-byte |
  | gms_72  | 0x51 | 0x30 | 15-byte |
  | gms_79  | 0x50 | 0x30 | 15-byte |
  | gms_83  | 0x52 | 0x33 | 15-byte |
  | gms_84  | 0x52 | 0x33 | 16-byte (bOnExclRequest) |
  | gms_87  | 0x55 | 0x33 | 16-byte |
  | gms_92  | 0x59 | 0x34 | 16-byte |
  | gms_95  | 0x58 | 0x32 | 16-byte |
  | jms_185 | 0x4A | 0x30 | 16-byte |

- **Template collision check: all 20 slots FREE** across the ten templates (re-check case-insensitively in Task 11).
- **`gms_12` is the ONLY excluded template** — the feature is genuinely absent (no `CWvsContext::SendSkillLearnItemUseRequest`, no result opcode in that client). Evidence-backed, not a deferral. The v1 plan's "gms_92 EXCLUDED" decision is **REVERSED**: gms_92 has IDB-verified opcodes (0x59 / 0x34) and a loaded IDB; its minimal template (only `CharacterCashItemUseHandle`) is irrelevant because the skill-book handler is a dedicated standalone opcode.
- **Every version uses the dedicated `CWvsContext::Send…`/`On…Result` pair** — there is NO generic item-use routing to build (legacy included). The mid-design "legacy = generic" hypothesis was disproved by decompiling the send in every IDB (v48 `sub_70E3E7`→0x40, v61 0x4B, v72 `sub_904B55`→0x51, v79 `0x955EBD`→0x50).
- **No deployment/env changes needed:** `COMMAND_TOPIC_SAGA` (line 66) and `EVENT_TOPIC_SAGA_STATUS` (line 138) are already in the shared `deploy/k8s/base/env-configmap.yaml` consumed via `envFrom: atlas-env`; `requests.RootUrl("SKILLS")` falls back to `BASE_SERVICE_URL` (`libs/atlas-rest/requests/url.go:14-19`) — do NOT add a `SKILLS_SERVICE_URL` override (see bug_service_url_hardcoded_base_namespace).
- **atlas-skills needs no code change:** `RequestUpdateBody` already carries `SkillId/Level/MasterLevel/Expiration`; clobber semantics handled caller-side by passing current Level/Expiration.
- **Skill saga payload precedent:** all existing producers (atlas-messages, npc-conversations, portal-actions) emit the shared `sharedsaga.UpdateSkillPayload`/`CreateSkillPayload` (no WorldId field; the orchestrator's local mirror unmarshals worldId=0). Follow that precedent — do not add WorldId.
- **No new saga Action** is introduced, so the orchestrator's `Step[any].UnmarshalJSON` switch and `event_acceptance.go` need no changes (`DestroyAssetFromSlot → {AssetDeleted, AssetQuantityChanged}`, `CreateSkill → {SkillCreated}`, `UpdateSkill → {SkillUpdated}` acceptances already exist).
- **IDB availability (2026-07-25): ALL loaded** — v48, v61, v72, v79, v83 (dump), v84, v87, v92, v95, jms. Every send/receive function is already located and named in its IDB (the derivation pass renamed the real `Send…`/`On…Result` and neutralized the v72 `0x90A591` / v79 `0x95B951` mis-ports). The v1 "IDB-blocked" constraint is VOID — Task 13 can promote every cell. Always `idb_list` and match binary NAME first (sessions/ports rotate).
- **Pre-existing issue observed, NOT in scope:** `template_gms_95_1.json` has 35 validator-less handler entries (silently dropped at runtime). Surface to the owner; do not fix here.

---

### Task 1: libs/atlas-packet — serverbound UseSkillBook codec

**Files:**
- Create: `libs/atlas-packet/character/serverbound/use_skill_book.go`
- Test: `libs/atlas-packet/character/serverbound/use_skill_book_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `const CharacterSkillBookUseHandle = "CharacterSkillBookUseHandle"`; type `UseSkillBook` with `UpdateTime() uint32`, `Slot() int16`, `ItemId() uint32`, `Operation() string`, `String() string`, `Encode(l, ctx) func(options) []byte`, `(*UseSkillBook).Decode(l, ctx) func(r, options)`. Tasks 10 and 11 use the handle const; Task 10 uses the struct.

- [ ] **Step 1: Write the failing test**

`libs/atlas-packet/character/serverbound/use_skill_book_test.go`:

```go
package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestUseSkillBookRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}
			output := UseSkillBook{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

// Golden bytes, v83: updateTime(4 LE) + slot(2 LE) + itemId(4 LE).
// 12345 = 0x3039; 2 = 0x0002; 2290000 = 0x22F150.
func TestUseSkillBookGoldenBytesV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()
	got := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}.Encode(l, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00, 0x02, 0x00, 0x50, 0xF1, 0x22, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}
```

(Logger idiom `l, _ := testlog.NewNullLogger()` verified against `libs/atlas-packet/storage/clientbound/show_test.go:35`.)

- [ ] **Step 2: Run the test to verify it fails**

Run (from `libs/atlas-packet`): `go test ./character/serverbound/ -run TestUseSkillBook -v`
Expected: FAIL — `undefined: UseSkillBook`.

- [ ] **Step 3: Write the codec**

`libs/atlas-packet/character/serverbound/use_skill_book.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterSkillBookUseHandle = "CharacterSkillBookUseHandle"

// UseSkillBook - CWvsContext::SendSkillLearnItemUseRequest
// packet-audit:fname CWvsContext::SendSkillLearnItemUseRequest
//
// Wire layout — IDB-verified byte-identical in all 10 versions
// (v48 0x70E3E7 … v95 0x9d65e0 … jms 0xAEEE61):
//
//	Encode4 updateTime
//	Encode2 slot
//	Encode4 itemId
//
// No version gate — only the per-tenant opcode differs (task-125 design §5.1).
type UseSkillBook struct {
	updateTime uint32
	slot       int16
	itemId     uint32
}

func (m UseSkillBook) UpdateTime() uint32 { return m.updateTime }
func (m UseSkillBook) Slot() int16        { return m.slot }
func (m UseSkillBook) ItemId() uint32     { return m.itemId }

func (m UseSkillBook) Operation() string {
	return CharacterSkillBookUseHandle
}

func (m UseSkillBook) String() string {
	return fmt.Sprintf("updateTime [%d], slot [%d], itemId [%d]", m.updateTime, m.slot, m.itemId)
}

func (m UseSkillBook) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.slot)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *UseSkillBook) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.slot = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
```

(Model exactly mirrors `inventory/serverbound/item_use.go` — same field widths, distinct fname/handle.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./character/serverbound/ -run TestUseSkillBook -v` (from `libs/atlas-packet`)
Expected: PASS for all variants + golden bytes.

- [ ] **Step 5: Run the whole package**

Run: `go test -race ./... && go vet ./...` (from `libs/atlas-packet`)
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/serverbound/use_skill_book.go libs/atlas-packet/character/serverbound/use_skill_book_test.go
git commit -m "feat(atlas-packet): USE_SKILL_BOOK serverbound codec (task-125)"
```

---

### Task 2: libs/atlas-packet — clientbound SkillLearnItemResult codec

**Files:**
- Create: `libs/atlas-packet/character/clientbound/skill_learn_item_result.go`
- Test: `libs/atlas-packet/character/clientbound/skill_learn_item_result_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `const CharacterSkillLearnItemResultWriter = "CharacterSkillLearnItemResult"`; type `SkillLearnItemResult` with constructor `NewSkillLearnItemResult(characterId uint32, isMasteryBook bool, skillId uint32, masterLevel uint32, canUse bool, success bool) SkillLearnItemResult`, getters, `Encode`/`Decode`. Task 9 announces via this writer const + constructor; Task 11 registers the writer name `"CharacterSkillLearnItemResult"` in templates.

- [ ] **Step 1: Write the failing test**

`libs/atlas-packet/character/clientbound/skill_learn_item_result_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestSkillLearnItemResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewSkillLearnItemResult(12345, true, 1121001, 20, true, false)
			output := SkillLearnItemResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.IsMasteryBook() != input.IsMasteryBook() {
				t.Errorf("isMasteryBook: got %v, want %v", output.IsMasteryBook(), input.IsMasteryBook())
			}
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
			if output.MasterLevel() != input.MasterLevel() {
				t.Errorf("masterLevel: got %v, want %v", output.MasterLevel(), input.MasterLevel())
			}
			if output.CanUse() != input.CanUse() {
				t.Errorf("canUse: got %v, want %v", output.CanUse(), input.CanUse())
			}
			if output.Success() != input.Success() {
				t.Errorf("success: got %v, want %v", output.Success(), input.Success())
			}
		})
	}
}

// Golden bytes, v83 — 15-byte body (NO leading bOnExclRequest byte):
// characterId(4 LE) + isMasteryBook(1) + skillId(4 LE) + masterLevel(4 LE) + canUse(1) + success(1).
// Trivially-readable values: characterId=1, mastery, skillId=2, masterLevel=3, canUse=1, success=0.
func TestSkillLearnItemResultGoldenBytesV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()
	got := NewSkillLearnItemResult(1, true, 2, 3, true, false).Encode(l, ctx)(nil)
	want := []byte{
		0x01, 0x00, 0x00, 0x00, // characterId
		0x01,                   // isMasteryBook
		0x02, 0x00, 0x00, 0x00, // skillId
		0x03, 0x00, 0x00, 0x00, // masterLevel
		0x01, // canUse
		0x00, // success
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}

// Golden bytes, v84 — 16-byte body (LEADING bOnExclRequest byte = 0x01).
// Proves the MajorVersion() >= 84 gate. Same field values as the v83 golden,
// so the only difference is the extra leading 0x01. (v84 clientbound diverges
// from v83 despite identical serverbound — the v84≠v83 exception.)
func TestSkillLearnItemResultGoldenBytesV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	l, _ := testlog.NewNullLogger()
	got := NewSkillLearnItemResult(1, true, 2, 3, true, false).Encode(l, ctx)(nil)
	want := []byte{
		0x01,                   // bOnExclRequest (v84+ leading byte)
		0x01, 0x00, 0x00, 0x00, // characterId
		0x01,                   // isMasteryBook
		0x02, 0x00, 0x00, 0x00, // skillId
		0x03, 0x00, 0x00, 0x00, // masterLevel
		0x01, // canUse
		0x00, // success
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}
```

The round-trip test above (`pt.Variants`) exercises Encode→Decode with a consistent ctx per variant, so both the 15- and 16-byte paths round-trip as long as `pt.Variants` spans versions on both sides of the v84 boundary; if it does not, add explicit v83 and v84 sub-cases so both codec paths are covered.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./character/clientbound/ -run TestSkillLearnItemResult -v` (from `libs/atlas-packet`)
Expected: FAIL — `undefined: NewSkillLearnItemResult`.

- [ ] **Step 3: Write the codec**

`libs/atlas-packet/character/clientbound/skill_learn_item_result.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterSkillLearnItemResultWriter = "CharacterSkillLearnItemResult"

// SkillLearnItemResult - CWvsContext::OnSkillLearnItemResult
// packet-audit:fname CWvsContext::OnSkillLearnItemResult
//
// Wire layout (IDB-verified all 10 versions):
//
//	Decode1 bOnExclRequest — v84+ ONLY (leading byte); server sends 1 so the
//	                         requesting/local client clears its exclusive-request
//	                         lock. Observers ignore it.
//	Decode4 characterId    — resolved via CUserPool::GetUser; glow renders on that avatar for any observer
//	Decode1 isMasteryBook
//	Decode4 skillId        — decoded-then-discarded by the client (every version)
//	Decode4 masterLevel    — decoded-then-discarded by the client (every version)
//	Decode1 canUse         — gates the on-avatar glow; effect renders only when 1
//	Decode1 success        — success vs failure sound/message (local user only)
//
// Atlas sends the real skillId/masterLevel even though the client discards them
// (task-125 design §D-4 — do not copy Cosmic's hardcoded zeros).
//
// VERSION GATE (IDB-verified, NOT speculative): the leading bOnExclRequest byte
// is present iff MajorVersion() >= 84. 15-byte body: gms_48/61/72/79/83.
// 16-byte body: gms_84/87/92/95/jms_185. This is a real v84≠v83 exception — v84
// serverbound is byte-identical to v83 but its clientbound diverges. Do NOT gate
// at >=87. bOnExclRequest is NOT domain data (server always sends 1), so it is
// not a struct field: Encode writes it, Decode consumes-and-discards it.
type SkillLearnItemResult struct {
	characterId   uint32
	isMasteryBook bool
	skillId       uint32
	masterLevel   uint32
	canUse        bool
	success       bool
}

// skillLearnResultHasExclByte reports whether this tenant's client reads the
// leading bOnExclRequest byte (GMS v84+; JMS v185 satisfies >=84 naturally).
func skillLearnResultHasExclByte(t tenant.Model) bool {
	return t.MajorVersion() >= 84
}

func NewSkillLearnItemResult(characterId uint32, isMasteryBook bool, skillId uint32, masterLevel uint32, canUse bool, success bool) SkillLearnItemResult {
	return SkillLearnItemResult{
		characterId:   characterId,
		isMasteryBook: isMasteryBook,
		skillId:       skillId,
		masterLevel:   masterLevel,
		canUse:        canUse,
		success:       success,
	}
}

func (m SkillLearnItemResult) CharacterId() uint32 { return m.characterId }
func (m SkillLearnItemResult) IsMasteryBook() bool { return m.isMasteryBook }
func (m SkillLearnItemResult) SkillId() uint32     { return m.skillId }
func (m SkillLearnItemResult) MasterLevel() uint32 { return m.masterLevel }
func (m SkillLearnItemResult) CanUse() bool        { return m.canUse }
func (m SkillLearnItemResult) Success() bool       { return m.success }
func (m SkillLearnItemResult) Operation() string   { return CharacterSkillLearnItemResultWriter }

func (m SkillLearnItemResult) String() string {
	return fmt.Sprintf("skill learn item result characterId [%d] mastery [%t] skillId [%d] masterLevel [%d] canUse [%t] success [%t]",
		m.characterId, m.isMasteryBook, m.skillId, m.masterLevel, m.canUse, m.success)
}

func (m SkillLearnItemResult) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if skillLearnResultHasExclByte(t) {
			w.WriteBool(true) // bOnExclRequest — clears the requester's exclusive-request lock (v84+)
		}
		w.WriteInt(m.characterId)
		w.WriteBool(m.isMasteryBook)
		w.WriteInt(m.skillId)
		w.WriteInt(m.masterLevel)
		w.WriteBool(m.canUse)
		w.WriteBool(m.success)
		return w.Bytes()
	}
}

func (m *SkillLearnItemResult) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if skillLearnResultHasExclByte(t) {
			_ = r.ReadBool() // bOnExclRequest — consumed then discarded (not domain data)
		}
		m.characterId = r.ReadUint32()
		m.isMasteryBook = r.ReadBool()
		m.skillId = r.ReadUint32()
		m.masterLevel = r.ReadUint32()
		m.canUse = r.ReadBool()
		m.success = r.ReadBool()
	}
}
```

Note: if `request.Reader` has no `ReadBool`, read a byte and compare `!= 0` — check a neighboring clientbound codec's Decode (e.g. `item_upgrade.go` uses `r.ReadBool()`, so it exists).

- [ ] **Step 4: Run tests**

Run: `go test -race ./... && go vet ./...` (from `libs/atlas-packet`)
Expected: PASS/clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/clientbound/skill_learn_item_result.go libs/atlas-packet/character/clientbound/skill_learn_item_result_test.go
git commit -m "feat(atlas-packet): SKILL_LEARN_ITEM_RESULT clientbound codec (task-125)"
```

---

### Task 3: libs/atlas-saga — `TemplateId` payload field + `skill_book_use` saga type

**Files:**
- Modify: `libs/atlas-saga/model.go` (type constants block, lines 13-27)
- Modify: `libs/atlas-saga/payloads.go` (`DestroyAssetFromSlotPayload`, lines 101-108)
- Test: `libs/atlas-saga/payloads_test.go` (create)

**Interfaces:**
- Produces: `saga.SkillBookUse Type = "skill_book_use"`; `DestroyAssetFromSlotPayload.TemplateId uint32` (json `templateId`). Tasks 4, 7 consume both.
- Backward compatibility: additive field — existing producers omit it → 0; the orchestrator's destroy handler passes through unchanged.

- [ ] **Step 1: Write the failing test**

`libs/atlas-saga/payloads_test.go`:

```go
package saga

import (
	"encoding/json"
	"testing"
)

// TemplateId is additive (task-125): absent in old payloads → 0; round-trips when set.
func TestDestroyAssetFromSlotPayloadTemplateIdRoundTrip(t *testing.T) {
	in := DestroyAssetFromSlotPayload{
		CharacterId:   1,
		InventoryType: 2,
		Slot:          3,
		Quantity:      1,
		TemplateId:    2290000,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out DestroyAssetFromSlotPayload
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.TemplateId != in.TemplateId {
		t.Errorf("templateId: got %d, want %d", out.TemplateId, in.TemplateId)
	}

	var legacy DestroyAssetFromSlotPayload
	if err := json.Unmarshal([]byte(`{"characterId":1,"inventoryType":2,"slot":3,"quantity":1,"showEffect":false}`), &legacy); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if legacy.TemplateId != 0 {
		t.Errorf("legacy templateId: got %d, want 0", legacy.TemplateId)
	}
}

func TestSkillBookUseSagaType(t *testing.T) {
	if string(SkillBookUse) != "skill_book_use" {
		t.Errorf("SkillBookUse: got %q", string(SkillBookUse))
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./... -run 'TestDestroyAssetFromSlotPayload|TestSkillBookUseSagaType' -v` (from `libs/atlas-saga`)
Expected: FAIL — unknown field `TemplateId`, undefined `SkillBookUse`.

- [ ] **Step 3: Implement**

In `libs/atlas-saga/model.go`, extend the type constants block:

```go
	PetEvolution         Type = "pet_evolution"
	SkillBookUse         Type = "skill_book_use"
```

In `libs/atlas-saga/payloads.go`, replace `DestroyAssetFromSlotPayload` with:

```go
// DestroyAssetFromSlotPayload represents the payload required to destroy an asset from a specific inventory slot.
type DestroyAssetFromSlotPayload struct {
	CharacterId   uint32 `json:"characterId"`          // CharacterId associated with the action
	InventoryType byte   `json:"inventoryType"`        // Type of inventory (1=equip, 2=use, 3=setup, 4=etc, 5=cash)
	Slot          int16  `json:"slot"`                 // Slot to destroy from (negative for equipped slots, positive for inventory slots)
	Quantity      uint32 `json:"quantity"`             // Quantity to destroy (0 or 1 for equipment)
	ShowEffect    bool   `json:"showEffect"`           // Render the item-loss chat line on the client when true
	TemplateId    uint32 `json:"templateId,omitempty"` // TemplateId of the destroyed item; 0 when unknown. Lets a reverse-walk compensator re-award the item (task-125)
}
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./... && go vet ./...` (from `libs/atlas-saga`)
Expected: PASS/clean.

- [ ] **Step 5: Workspace ripple check**

Run from the worktree root: `go build ./...` inside `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-npc-conversations/atlas.com/npc`, `services/atlas-messages/atlas.com/messages` (heaviest atlas-saga importers).
Expected: clean (additive field).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-saga/model.go libs/atlas-saga/payloads.go libs/atlas-saga/payloads_test.go
git commit -m "feat(atlas-saga): skill_book_use saga type + DestroyAssetFromSlotPayload.TemplateId (task-125)"
```

---

### Task 4: atlas-saga-orchestrator — skill_book_use compensation

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` (saga-type re-export block, lines 31-40)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` (interface ~lines 40-60, routing ~line 195, new funcs after `DispatchPetEvolutionRollbacks`)
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go` (append)

**Interfaces:**
- Consumes: `sharedsaga.SkillBookUse`, `DestroyAssetFromSlotPayload.TemplateId` (Task 3); existing `c.compP.RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error`.
- Produces: `CompensateFailedStep` routes `SkillBookUse` sagas to `compensateSkillBookUse`; `DispatchSkillBookUseRollbacks(s Saga)` exported on the interface for tests (mirrors `DispatchPetEvolutionRollbacks`).

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go` (mirrors `TestPetEvolutionCompensationRefundsResources` at the top of the file — same mock/spy idiom, exercising the Dispatch half directly to avoid the Kafka emit path):

```go
// TestSkillBookUseCompensationRefundsBook verifies that when a skill_book_use
// saga fails on the skill step AFTER the book was destroyed, the reverse walk
// re-awards the book using the destroy step's TemplateId (task-125).
func TestSkillBookUseCompensationRefundsBook(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	const (
		testCharId = uint32(88001)
		bookId     = uint32(2290000)
	)

	type createCall struct {
		CharacterId uint32
		TemplateId  uint32
		Quantity    uint32
	}
	var createItemCalls []createCall
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, _ time.Time) error {
			createItemCalls = append(createItemCalls, createCall{characterId, templateId, quantity})
			return nil
		},
	}

	transactionId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(SkillBookUse).
		SetInitiatedBy("skill-book-compensation-test").
		AddStep("destroy_asset_from_slot", Completed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId:   testCharId,
			InventoryType: 2,
			Slot:          5,
			Quantity:      1,
			TemplateId:    bookId,
		}).
		AddStep("update_skill", Failed, UpdateSkill, UpdateSkillPayload{
			CharacterId: testCharId,
			SkillId:     1121001,
			Level:       9,
			MasterLevel: 20,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, tctx).WithCompartmentProcessor(compP)
	compensator.DispatchSkillBookUseRollbacks(s)

	assert.Equal(t, 1, len(createItemCalls), "book should be re-awarded exactly once")
	if len(createItemCalls) == 1 {
		assert.Equal(t, testCharId, createItemCalls[0].CharacterId)
		assert.Equal(t, bookId, createItemCalls[0].TemplateId)
		assert.Equal(t, uint32(1), createItemCalls[0].Quantity)
	}
}

// TestSkillBookUseCompensationDestroyFailedNoRefund verifies that when the
// destroy step itself fails (nothing completed), the reverse walk re-awards
// nothing — the book never left the slot.
func TestSkillBookUseCompensationDestroyFailedNoRefund(t *testing.T) {
	logger, _ := test.NewNullLogger()

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	var createItemCalls int
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			createItemCalls++
			return nil
		},
	}

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(SkillBookUse).
		SetInitiatedBy("skill-book-compensation-test").
		AddStep("destroy_asset_from_slot", Failed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId:   88002,
			InventoryType: 2,
			Slot:          5,
			Quantity:      1,
			TemplateId:    2290000,
		}).
		AddStep("update_skill", Pending, UpdateSkill, UpdateSkillPayload{
			CharacterId: 88002,
			SkillId:     1121001,
		}).
		Build()
	assert.NoError(t, err)

	compensator := NewCompensator(logger, tctx).WithCompartmentProcessor(compP)
	compensator.DispatchSkillBookUseRollbacks(s)

	assert.Equal(t, 0, createItemCalls, "no refund when the destroy step never completed")
}

// TestSkillBookUseCompensationMissingTemplateIdSkips verifies a completed
// destroy step with TemplateId 0 (legacy producer) is skipped rather than
// re-awarding item 0.
func TestSkillBookUseCompensationMissingTemplateIdSkips(t *testing.T) {
	logger, _ := test.NewNullLogger()

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	var createItemCalls int
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			createItemCalls++
			return nil
		},
	}

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(SkillBookUse).
		SetInitiatedBy("skill-book-compensation-test").
		AddStep("destroy_asset_from_slot", Completed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId:   88003,
			InventoryType: 2,
			Slot:          5,
			Quantity:      1,
		}).
		AddStep("update_skill", Failed, UpdateSkill, UpdateSkillPayload{
			CharacterId: 88003,
			SkillId:     1121001,
		}).
		Build()
	assert.NoError(t, err)

	compensator := NewCompensator(logger, tctx).WithCompartmentProcessor(compP)
	compensator.DispatchSkillBookUseRollbacks(s)

	assert.Equal(t, 0, createItemCalls, "TemplateId 0 must not re-award item 0")
}
```

(All imports already present at the top of `compensator_test.go`. Verify `NewCompensator(logger, tctx).WithCompartmentProcessor(compP)` compiles standalone — the pet test chains `WithCharacterProcessor` first; `WithCompartmentProcessor` alone is fine since each `With*` copies the struct.)

- [ ] **Step 2: Run to verify failure**

Run (from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`): `go test ./saga/ -run TestSkillBookUse -v`
Expected: FAIL — `undefined: SkillBookUse`, `DispatchSkillBookUseRollbacks`.

- [ ] **Step 3: Implement**

1. `saga/model.go` — extend the saga-type re-export block (after `PetEvolution = sharedsaga.PetEvolution`):

```go
	SkillBookUse         = sharedsaga.SkillBookUse
```

2. `saga/compensator.go` — add to the `Compensator` interface, next to the pet-evolution pair (`compensatePetEvolution` at line 42, `DispatchPetEvolutionRollbacks` at line 56):

```go
	compensateSkillBookUse(s Saga, failedStep Step[any]) error

	// DispatchSkillBookUseRollbacks reverse-walks the completed steps of a
	// skill_book_use saga and re-awards the destroyed book (task-125). Pure
	// dispatch half — no lifecycle transitions, no event emission.
	DispatchSkillBookUseRollbacks(s Saga)
```

3. `saga/compensator.go` — in `CompensateFailedStep`, after the `PetEvolution` routing block (line ~195-197):

```go
	// Skill-book reverse-walk (task-125). A failed create_skill/update_skill
	// must re-award the already-destroyed book rather than only compensating
	// the failed step; a failed destroy step has nothing to reverse.
	if s.SagaType() == SkillBookUse {
		return c.compensateSkillBookUse(s, failedStep)
	}
```

4. `saga/compensator.go` — append after `DispatchPetEvolutionRollbacks` (line ~1140), mirroring `compensatePetEvolution` exactly:

```go
// compensateSkillBookUse is the skill_book_use reverse-walk compensator
// (task-125). On a failed create_skill/update_skill it re-awards the book
// destroyed by the completed destroy_asset_from_slot step (using the
// payload's TemplateId), emits exactly one StatusEventTypeFailed, cancels
// the Phase-4 timer, and evicts the saga. A failed destroy step (first
// step) has no completed steps to reverse — the walk is a no-op and the
// saga just terminates with the failed event.
//
// Double-emission is prevented by TryTransition(Compensating → Failed): if
// the timer already emitted Failed, the transition is refused and this
// function returns without re-emitting. Mirrors compensatePetEvolution.
func (c *CompensatorImpl) compensateSkillBookUse(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("SkillBookUse saga failing — dispatching reverse-walk compensation.")

	c.DispatchSkillBookUseRollbacks(s)

	if !GetCache().TryTransition(c.ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed) {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Info("saga already in terminal Failed state; reverse-walk emission skipped.")
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(c.ctx, s.TransactionId())
		return nil
	}

	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	reason := fmt.Sprintf("Skill book use failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after skill-book compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Skill-book reverse-walk compensation complete; saga terminated.")
	return nil
}

// DispatchSkillBookUseRollbacks reverse-walks the saga's completed steps and
// re-awards each completed destroy_asset_from_slot via CreateItem using the
// payload's TemplateId. Pure dispatch half — callers own lifecycle/emission.
// Slot position is not preserved (the freed slot guarantees space). A destroy
// payload without TemplateId (legacy producer) cannot be re-awarded and is
// skipped with an error log. An error re-awarding one step does not abort the
// chain.
func (c *CompensatorImpl) DispatchSkillBookUseRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		if step.Action() != DestroyAssetFromSlot {
			continue
		}
		payload, ok := step.Payload().(DestroyAssetFromSlotPayload)
		if !ok {
			continue
		}
		if payload.TemplateId == 0 {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"step_id":        step.StepId(),
				"tenant_id":      c.t.Id().String(),
			}).Error("Reverse-walk: destroy step carries no TemplateId; cannot re-award the book.")
			continue
		}
		qty := payload.Quantity
		if qty == 0 {
			qty = 1
		}
		if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
			c.l.WithError(err).WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"step_id":        step.StepId(),
				"template_id":    payload.TemplateId,
			}).Error("Reverse-walk: DestroyAssetFromSlot → CreateItem dispatch failed; continuing chain.")
		}
	}
}
```

- [ ] **Step 4: Run tests**

Run (from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`): `go test -race ./... && go vet ./...`
Expected: PASS/clean (including the existing `unmarshal_completeness_test.go` — no new Action was added).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go
git commit -m "feat(atlas-saga-orchestrator): skill_book_use compensation reverse-walk (task-125)"
```

---

### Task 5: atlas-consumables — consumable-data accessors + skills REST client

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go` (append getters)
- Create: `services/atlas-consumables/atlas.com/consumables/skill/requests.go`
- Create: `services/atlas-consumables/atlas.com/consumables/skill/rest.go`
- Create: `services/atlas-consumables/atlas.com/consumables/skill/model.go`
- Create: `services/atlas-consumables/atlas.com/consumables/skill/processor.go`

**Interfaces:**
- Produces: `data/consumable.Model.MasterLevel() uint32`, `.ReqSkillLevel() uint32`, `.Skills() []uint32`; package `skill` with `Processor` interface (`GetByCharacterId(characterId uint32) ([]Model, error)`) and `Model` with `Id() uint32`, `Level() byte`, `MasterLevel() byte`, `Expiration() time.Time`. Task 8 consumes both.
- The skills client is a copy of the established `services/atlas-messages/atlas.com/messages/skill/` shape (resource `characters/%d/skills` under `requests.RootUrl("SKILLS")`, which falls back to `BASE_SERVICE_URL` — already wired in the shared configmap).

- [ ] **Step 1: Add the data accessors**

Append to `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go` (fields exist at lines 60-64, 105; only `SuccessRate()` is exposed today):

```go
func (m Model) MasterLevel() uint32 {
	return m.masterLevel
}

func (m Model) ReqSkillLevel() uint32 {
	return m.reqSkillLevel
}

func (m Model) Skills() []uint32 {
	return m.skills
}
```

- [ ] **Step 2: Create the skills client**

`skill/requests.go`:

```go
package skill

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/skills"
)

func getBaseRequest() string {
	return requests.RootUrl("SKILLS")
}

func requestByCharacterId(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
```

`skill/rest.go`:

```go
package skill

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id                uint32    `json:"-"`
	Level             byte      `json:"level"`
	MasterLevel       byte      `json:"masterLevel"`
	Expiration        time.Time `json:"expiration"`
	CooldownExpiresAt time.Time `json:"cooldownExpiresAt"`
}

func (r RestModel) GetName() string {
	return "skills"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		level:       rm.Level,
		masterLevel: rm.MasterLevel,
		expiration:  rm.Expiration,
	}, nil
}
```

`skill/model.go`:

```go
package skill

import "time"

type Model struct {
	id          uint32
	level       byte
	masterLevel byte
	expiration  time.Time
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) MasterLevel() byte {
	return m.masterLevel
}

func (m Model) Expiration() time.Time {
	return m.expiration
}
```

`skill/processor.go`:

```go
package skill

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}
```

- [ ] **Step 3: Build**

Run (from `services/atlas-consumables/atlas.com/consumables`): `go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/data/consumable/model.go services/atlas-consumables/atlas.com/consumables/skill/
git commit -m "feat(atlas-consumables): consumable book accessors + skills REST client (task-125)"
```

---

### Task 6: atlas-consumables — Kafka plumbing (command, event, saga client, saga-status topic, once validator)

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go`
- Create: `services/atlas-consumables/atlas.com/consumables/kafka/message/saga/kafka.go`
- Create: `services/atlas-consumables/atlas.com/consumables/saga/producer.go`
- Create: `services/atlas-consumables/atlas.com/consumables/saga/processor.go`
- Create: `services/atlas-consumables/atlas.com/consumables/kafka/once/saga/once.go`
- Create: `services/atlas-consumables/atlas.com/consumables/kafka/consumer/saga/consumer.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/main.go` (register the saga-status consumer topic)
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/producer.go` (result-event provider)

**Interfaces:**
- Produces (consumed by Tasks 7-9):
  - `consumable.CommandRequestSkillBookUse = "REQUEST_SKILL_BOOK_USE"`, `consumable.RequestSkillBookUseBody{Slot slot.Position, ItemId item.Id}`
  - `consumable.EventTypeSkillBookResult = "SKILL_BOOK_RESULT"`, `consumable.SkillBookResultBody{IsMasteryBook bool, SkillId uint32, MasterLevel uint32, CanUse bool, Success bool}`
  - package `saga` (service-local): `NewProcessor(l, ctx) Processor` with `Create(s sharedsaga.Saga) error`
  - `kafka/message/saga`: `EnvCommandTopic`, `EnvStatusEventTopic`, `StatusEventTypeCompleted/Failed`, `StatusEvent[T]`, `StatusEventFailedBody`
  - `kafka/once/saga.TransactionValidator(transactionId uuid.UUID) message.Validator[saga.StatusEvent[json.RawMessage]]`
  - `consumable.SkillBookResultEventProvider(characterId character.Id) func(isMasteryBook bool, skillId uint32, masterLevel uint32, canUse bool, success bool) model.Provider[[]kafka.Message]`

- [ ] **Step 1: Extend the consumable message contract**

In `kafka/message/consumable/kafka.go`, extend the command constants block:

```go
	CommandRequestItemConsume     = "REQUEST_ITEM_CONSUME"
	CommandRequestScroll          = "REQUEST_SCROLL"
	CommandRequestSkillBookUse    = "REQUEST_SKILL_BOOK_USE"
	CommandApplyConsumableEffect  = "APPLY_CONSUMABLE_EFFECT"
	CommandCancelConsumableEffect = "CANCEL_CONSUMABLE_EFFECT"
```

Add after `RequestScrollBody`:

```go
// RequestSkillBookUseBody is the body for a Skill Book (228) / Mastery Book
// (229) consume request (task-125). Slot is the USE-compartment slot holding
// the book.
type RequestSkillBookUseBody struct {
	Slot   slot.Position `json:"slot"`
	ItemId item.Id       `json:"itemId"`
}
```

Extend the event constants block:

```go
	EnvEventTopic            = "EVENT_TOPIC_CONSUMABLE_STATUS"
	EventTypeError           = "ERROR"
	EventTypeScroll          = "SCROLL"
	EventTypeEffectApplied   = "EFFECT_APPLIED"
	EventTypeSkillBookResult = "SKILL_BOOK_RESULT"
```

Add after `ScrollBody`:

```go
// SkillBookResultBody carries the outcome of a skill-book use — the writer's
// inputs for SKILL_LEARN_ITEM_RESULT. CanUse=false is a validation rejection
// or saga failure (requester-only); CanUse=true broadcasts to the map with
// Success carrying the roll outcome.
type SkillBookResultBody struct {
	IsMasteryBook bool   `json:"isMasteryBook"`
	SkillId       uint32 `json:"skillId"`
	MasterLevel   uint32 `json:"masterLevel"`
	CanUse        bool   `json:"canUse"`
	Success       bool   `json:"success"`
}
```

- [ ] **Step 2: Saga message mirror**

`kafka/message/saga/kafka.go` (copy of the map-actions mirror):

```go
package saga

import "github.com/google/uuid"

const (
	EnvCommandTopic     = "COMMAND_TOPIC_SAGA"
	EnvStatusEventTopic = "EVENT_TOPIC_SAGA_STATUS"
)

const (
	StatusEventTypeCompleted = "COMPLETED"
	StatusEventTypeFailed    = "FAILED"
)

type StatusEvent[T any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          T         `json:"body"`
}

type StatusEventCompletedBody struct{}

type StatusEventFailedBody struct {
	ErrorCode  string `json:"errorCode"`
	Reason     string `json:"reason"`
	FailedStep string `json:"failedStep"`
}
```

- [ ] **Step 3: Saga client**

`saga/producer.go` (per `services/atlas-map-actions/atlas.com/map-actions/saga/producer.go`):

```go
package saga

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(s sharedsaga.Saga) model.Provider[[]kafka.Message] {
	key := []byte(s.TransactionId.String())
	return producer.SingleMessageProvider(key, &s)
}
```

`saga/processor.go`:

```go
package saga

import (
	"atlas-consumables/kafka/message/saga"
	"atlas-consumables/kafka/producer"
	"context"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Create(s sharedsaga.Saga) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) Create(s sharedsaga.Saga) error {
	return producer.ProviderImpl(p.l)(p.ctx)(saga.EnvCommandTopic)(CreateCommandProvider(s))
}
```

- [ ] **Step 4: One-time validator**

`kafka/once/saga/once.go` (mirrors `kafka/once/compartment/once.go`). One registration must catch BOTH terminal outcomes — registering two typed handlers would leak the one that never fires. `json.RawMessage` as the body type captures either body; the handler demuxes on `Type`:

```go
package saga

import (
	"atlas-consumables/kafka/message/saga"
	"context"
	"encoding/json"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// TransactionValidator passes for the terminal (COMPLETED or FAILED) status
// event of one saga transaction. Used with message.OneTimeConfig so a single
// registration observes whichever outcome arrives (a saga emits exactly one).
func TransactionValidator(transactionId uuid.UUID) message.Validator[saga.StatusEvent[json.RawMessage]] {
	return func(l logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[json.RawMessage]) bool {
		return e.TransactionId == transactionId &&
			(e.Type == saga.StatusEventTypeCompleted || e.Type == saga.StatusEventTypeFailed)
	}
}
```

- [ ] **Step 5: Saga-status topic consumer registration**

`kafka/consumer/saga/consumer.go` — topic registration only (no persistent handlers; one-time handlers attach dynamically per request):

```go
package saga

import (
	consumer2 "atlas-consumables/kafka/consumer"
	"atlas-consumables/kafka/message/saga"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// InitConsumers registers the saga status-event topic so one-time handlers
// (skill-book result gating, task-125) can attach at request time. No
// persistent handlers — events for unknown transactions are no-ops.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("saga_status_event")(saga.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}
```

In `main.go`, add the import `sagaconsumer "atlas-consumables/kafka/consumer/saga"` and register after the pickup consumer (line ~45):

```go
	pickupconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	sagaconsumer.InitConsumers(l)(cmf)(consumerGroupId)
```

- [ ] **Step 6: Result-event provider**

Append to `consumable/producer.go`:

```go
// SkillBookResultEventProvider builds the SKILL_BOOK_RESULT event (task-125),
// keyed by characterId. Emitted exactly once per skill-book request: on
// validation rejection (canUse=false, immediately) or on the saga's terminal
// status (canUse=completed, success=roll outcome).
func SkillBookResultEventProvider(characterId character.Id) func(isMasteryBook bool, skillId uint32, masterLevel uint32, canUse bool, success bool) model.Provider[[]kafka.Message] {
	return func(isMasteryBook bool, skillId uint32, masterLevel uint32, canUse bool, success bool) model.Provider[[]kafka.Message] {
		key := producer.CreateKey(int(characterId))
		value := &consumable.Event[consumable.SkillBookResultBody]{
			CharacterId: characterId,
			Type:        consumable.EventTypeSkillBookResult,
			Body: consumable.SkillBookResultBody{
				IsMasteryBook: isMasteryBook,
				SkillId:       skillId,
				MasterLevel:   masterLevel,
				CanUse:        canUse,
				Success:       success,
			},
		}
		return producer.SingleMessageProvider(key, value)
	}
}
```

- [ ] **Step 7: Build and commit**

Run (from `services/atlas-consumables/atlas.com/consumables`): `go build ./... && go vet ./...`
Expected: clean.

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka/ services/atlas-consumables/atlas.com/consumables/saga/ services/atlas-consumables/atlas.com/consumables/consumable/producer.go services/atlas-consumables/atlas.com/consumables/main.go
git commit -m "feat(atlas-consumables): skill-book command/event contract + saga client + saga-status one-time plumbing (task-125)"
```

---

### Task 7: atlas-consumables — pure skill-book helpers (TDD)

**Files:**
- Create: `services/atlas-consumables/atlas.com/consumables/consumable/skill_book.go`
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/skill_book_test.go`

**Interfaces:**
- Consumes: `sharedsaga` (Task 3 additions), `job.IdFromSkillId` (`libs/atlas-constants/job/model.go:53` — the DOM-21 job-prefix helper; do NOT re-implement `/10000`).
- Produces (consumed by Task 8):
  - `SelectSkillBookTargetSkill(skills []uint32, jobId job.Id) (uint32, bool)`
  - `SkillBookRollPasses(roll int32, successRate uint32) bool`
  - `ValidateSkillBookSkillState(isMasteryBook bool, hasRecord bool, currentLevel byte, currentMasterLevel byte, reqSkillLevel uint32, bookMasterLevel uint32) error`
  - sentinel errors `ErrSkillBookSkillNotLearned`, `ErrSkillBookReqSkillLevel`, `ErrSkillBookMasterLevelCeiling`
  - `BuildSkillBookSaga(transactionId uuid.UUID, characterId uint32, slot int16, itemId item.Id, rollPassed bool, hasRecord bool, currentLevel byte, currentExpiration time.Time, targetSkillId uint32, bookMasterLevel byte) sharedsaga.Saga`

- [ ] **Step 1: Write the failing tests**

`consumable/skill_book_test.go`:

```go
package consumable

import (
	"testing"
	"time"

	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/google/uuid"
)

// Job-prefix rule (Cosmic ItemInformationProvider.java:1481-1483):
// first skills[] entry with skillId/10000 == jobId; no match ⇒ unusable.
func TestSelectSkillBookTargetSkill(t *testing.T) {
	// v83 WZ 2290000 (Monster Magnet books): [1121001 Hero, 1221001 Paladin, 1321001 DK]
	skills := []uint32{1121001, 1221001, 1321001}
	tests := []struct {
		name   string
		jobId  job.Id
		wantId uint32
		wantOk bool
	}{
		{"hero matches first entry", job.Id(112), 1121001, true},
		{"paladin matches second entry", job.Id(122), 1221001, true},
		{"dark knight matches third entry", job.Id(132), 1321001, true},
		{"bishop matches nothing", job.Id(232), 0, false},
		{"beginner matches nothing", job.Id(0), 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := SelectSkillBookTargetSkill(skills, tc.jobId)
			if ok != tc.wantOk || got != tc.wantId {
				t.Errorf("got (%d,%t), want (%d,%t)", got, ok, tc.wantId, tc.wantOk)
			}
		})
	}
}

// First-match wins when multiple entries share the job prefix.
func TestSelectSkillBookTargetSkillFirstMatch(t *testing.T) {
	got, ok := SelectSkillBookTargetSkill([]uint32{1121001, 1121002}, job.Id(112))
	if !ok || got != 1121001 {
		t.Errorf("got (%d,%t), want (1121001,true)", got, ok)
	}
}

// Strictly-less-than semantics (design §2 D-3): success=0 NEVER passes,
// success=100 ALWAYS passes. This is deliberately NOT the scroll path's <=.
func TestSkillBookRollPassesBoundaries(t *testing.T) {
	tests := []struct {
		roll int32
		rate uint32
		want bool
	}{
		{0, 0, false},   // success=0 never passes even on the best roll
		{99, 0, false},  //
		{0, 100, true},  // success=100 always passes
		{99, 100, true}, //
		{69, 70, true},  // roll < rate passes
		{70, 70, false}, // roll == rate fails (the <= off-by-one guard)
		{71, 70, false},
	}
	for _, tc := range tests {
		if got := SkillBookRollPasses(tc.roll, tc.rate); got != tc.want {
			t.Errorf("SkillBookRollPasses(%d, %d) = %t, want %t", tc.roll, tc.rate, got, tc.want)
		}
	}
}

// FR-2.5–2.8 skill-state gates. 228 (skill book) may target an unlearned
// skill only when reqSkillLevel==0 (D-1); 229 (mastery book) requires a
// learned record (Level >= 1).
func TestValidateSkillBookSkillState(t *testing.T) {
	tests := []struct {
		name               string
		isMasteryBook      bool
		hasRecord          bool
		currentLevel       byte
		currentMasterLevel byte
		reqSkillLevel      uint32
		bookMasterLevel    uint32
		wantErr            error
	}{
		{"mastery: happy path", true, true, 5, 10, 5, 20, nil},
		{"mastery: no record", true, false, 0, 0, 5, 20, ErrSkillBookSkillNotLearned},
		{"mastery: record at level 0", true, true, 0, 10, 0, 20, ErrSkillBookSkillNotLearned},
		{"mastery: below reqSkillLevel", true, true, 4, 10, 5, 20, ErrSkillBookReqSkillLevel},
		{"mastery: at reqSkillLevel passes", true, true, 5, 10, 5, 20, nil},
		{"mastery: master level at ceiling", true, true, 5, 20, 5, 20, ErrSkillBookMasterLevelCeiling},
		{"mastery: master level above ceiling", true, true, 5, 30, 5, 20, ErrSkillBookMasterLevelCeiling},
		{"skill book: no record, req 0 (teach)", false, false, 0, 0, 0, 10, nil},
		{"skill book: no record, req > 0", false, false, 0, 0, 1, 10, ErrSkillBookReqSkillLevel},
		{"skill book: existing record ok", false, true, 3, 5, 0, 10, nil},
		{"skill book: existing at ceiling", false, true, 3, 10, 0, 10, ErrSkillBookMasterLevelCeiling},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateSkillBookSkillState(tc.isMasteryBook, tc.hasRecord, tc.currentLevel, tc.currentMasterLevel, tc.reqSkillLevel, tc.bookMasterLevel)
			if got != tc.wantErr {
				t.Errorf("got %v, want %v", got, tc.wantErr)
			}
		})
	}
}

// Saga shapes (design §5.3): destroy step always (consume-on-fail, with
// TemplateId for compensation); skill step only on a passing roll —
// update_skill carries CURRENT level/expiration (atlas-skills Update clobbers
// unconditionally), create_skill teaches at level 0 with zero expiration.
func TestBuildSkillBookSaga(t *testing.T) {
	txId := uuid.New()
	exp := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("failed roll: destroy only", func(t *testing.T) {
		s := BuildSkillBookSaga(txId, 100, 5, item2.Id(2290000), false, true, 9, exp, 1121001, 20)
		if s.TransactionId != txId {
			t.Errorf("transactionId: got %v", s.TransactionId)
		}
		if s.SagaType != sharedsaga.SkillBookUse {
			t.Errorf("sagaType: got %v", s.SagaType)
		}
		if len(s.Steps) != 1 {
			t.Fatalf("steps: got %d, want 1", len(s.Steps))
		}
		if s.Steps[0].Action != sharedsaga.DestroyAssetFromSlot {
			t.Errorf("step 0 action: got %v", s.Steps[0].Action)
		}
		p, ok := s.Steps[0].Payload.(sharedsaga.DestroyAssetFromSlotPayload)
		if !ok {
			t.Fatalf("step 0 payload type: %T", s.Steps[0].Payload)
		}
		if p.CharacterId != 100 || p.InventoryType != 2 || p.Slot != 5 || p.Quantity != 1 || p.TemplateId != 2290000 {
			t.Errorf("destroy payload: %+v", p)
		}
	})

	t.Run("passed roll with record: destroy + update_skill carrying current level/expiration", func(t *testing.T) {
		s := BuildSkillBookSaga(txId, 100, 5, item2.Id(2290000), true, true, 9, exp, 1121001, 20)
		if len(s.Steps) != 2 {
			t.Fatalf("steps: got %d, want 2", len(s.Steps))
		}
		if s.Steps[1].Action != sharedsaga.UpdateSkill {
			t.Errorf("step 1 action: got %v", s.Steps[1].Action)
		}
		p, ok := s.Steps[1].Payload.(sharedsaga.UpdateSkillPayload)
		if !ok {
			t.Fatalf("step 1 payload type: %T", s.Steps[1].Payload)
		}
		if p.CharacterId != 100 || p.SkillId != 1121001 || p.Level != 9 || p.MasterLevel != 20 || !p.Expiration.Equal(exp) {
			t.Errorf("update payload: %+v", p)
		}
	})

	t.Run("passed roll without record: destroy + create_skill at level 0, permanent", func(t *testing.T) {
		s := BuildSkillBookSaga(txId, 100, 5, item2.Id(2280000), true, false, 0, time.Time{}, 2121003, 10)
		if len(s.Steps) != 2 {
			t.Fatalf("steps: got %d, want 2", len(s.Steps))
		}
		if s.Steps[1].Action != sharedsaga.CreateSkill {
			t.Errorf("step 1 action: got %v", s.Steps[1].Action)
		}
		p, ok := s.Steps[1].Payload.(sharedsaga.CreateSkillPayload)
		if !ok {
			t.Fatalf("step 1 payload type: %T", s.Steps[1].Payload)
		}
		if p.CharacterId != 100 || p.SkillId != 2121003 || p.Level != 0 || p.MasterLevel != 10 || !p.Expiration.IsZero() {
			t.Errorf("create payload: %+v", p)
		}
	})
}
```

- [ ] **Step 2: Run to verify failure**

Run (from `services/atlas-consumables/atlas.com/consumables`): `go test ./consumable/ -run 'SkillBook' -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Implement**

`consumable/skill_book.go`:

```go
package consumable

import (
	"errors"
	"time"

	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/google/uuid"
)

// Skill-book eligibility sentinels (FR-2.5–2.8). Distinct values so tests and
// warn-logs can name the gate that rejected the request.
var (
	ErrSkillBookSkillNotLearned    = errors.New("mastery book requires the target skill to be learned")
	ErrSkillBookReqSkillLevel      = errors.New("current skill level below the book's required level")
	ErrSkillBookMasterLevelCeiling = errors.New("master level already at or above the book's grant")
)

// SelectSkillBookTargetSkill returns the first entry in the book's skills[]
// whose job prefix equals the character's job id (Cosmic
// ItemInformationProvider rule: skillId/10000 == jobId); no match means the
// book is unusable for this character (FR-2.4).
func SelectSkillBookTargetSkill(skills []uint32, jobId job.Id) (uint32, bool) {
	for _, s := range skills {
		if job.IdFromSkillId(skill.Id(s)) == jobId {
			return s, true
		}
	}
	return 0, false
}

// SkillBookRollPasses reports whether a roll in [0,100) passes for a percent
// success rate. Strictly-less-than: success=0 never passes, success=100
// always passes. Deliberately NOT the scroll path's <= (which gives
// success=0 a 1% pass rate) — design §2 D-3.
func SkillBookRollPasses(roll int32, successRate uint32) bool {
	return roll < int32(successRate)
}

// ValidateSkillBookSkillState enforces the skill-state gates (FR-2.5–2.8):
// a mastery book (229) requires a learned record (level >= 1); a skill book
// (228) may target an unlearned skill only when the book's reqSkillLevel is
// 0 (an absent record counts as level 0); the current level must meet
// reqSkillLevel; and the current master level must be below the book's grant.
func ValidateSkillBookSkillState(isMasteryBook bool, hasRecord bool, currentLevel byte, currentMasterLevel byte, reqSkillLevel uint32, bookMasterLevel uint32) error {
	if isMasteryBook && (!hasRecord || currentLevel < 1) {
		return ErrSkillBookSkillNotLearned
	}
	level := uint32(0)
	if hasRecord {
		level = uint32(currentLevel)
	}
	if level < reqSkillLevel {
		return ErrSkillBookReqSkillLevel
	}
	if uint32(currentMasterLevel) >= bookMasterLevel {
		return ErrSkillBookMasterLevelCeiling
	}
	return nil
}

// BuildSkillBookSaga constructs the skill_book_use saga (design §5.3).
// Step 1 destroys exactly one book from the slot on BOTH outcomes
// (consume-on-fail, Cosmic parity); TemplateId rides along so the
// orchestrator's reverse walk can re-award the book if the skill step fails.
// Step 2 exists only on a passing roll: update_skill carries the CURRENT
// level/expiration (atlas-skills Update clobbers all columns), create_skill
// teaches an unlearned skill at level 0 with zero (permanent) expiration.
func BuildSkillBookSaga(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id, rollPassed bool, hasRecord bool, currentLevel byte, currentExpiration time.Time, targetSkillId uint32, bookMasterLevel byte) sharedsaga.Saga {
	b := sharedsaga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(sharedsaga.SkillBookUse).
		SetInitiatedBy("SKILL_BOOK").
		AddStep("destroy_asset_from_slot", sharedsaga.Pending, sharedsaga.DestroyAssetFromSlot, sharedsaga.DestroyAssetFromSlotPayload{
			CharacterId:   characterId,
			InventoryType: byte(inventory2.TypeValueUse),
			Slot:          slot,
			Quantity:      1,
			ShowEffect:    false,
			TemplateId:    uint32(itemId),
		})
	if rollPassed {
		if hasRecord {
			b.AddStep("update_skill", sharedsaga.Pending, sharedsaga.UpdateSkill, sharedsaga.UpdateSkillPayload{
				CharacterId: characterId,
				SkillId:     targetSkillId,
				Level:       currentLevel,
				MasterLevel: bookMasterLevel,
				Expiration:  currentExpiration,
			})
		} else {
			b.AddStep("create_skill", sharedsaga.Pending, sharedsaga.CreateSkill, sharedsaga.CreateSkillPayload{
				CharacterId: characterId,
				SkillId:     targetSkillId,
				Level:       0,
				MasterLevel: bookMasterLevel,
				Expiration:  time.Time{},
			})
		}
	}
	return b.Build()
}
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./consumable/ -run 'SkillBook' -v` then `go test -race ./... && go vet ./...` (from `services/atlas-consumables/atlas.com/consumables`)
Expected: PASS/clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/skill_book.go services/atlas-consumables/atlas.com/consumables/consumable/skill_book_test.go
git commit -m "feat(atlas-consumables): skill-book target selection, roll, state gates, saga shape (task-125)"
```

---

### Task 8: atlas-consumables — RequestSkillBookUse processor + command consumer

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (append two methods)
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go` (new handler + registration)

**Interfaces:**
- Consumes: everything from Tasks 5-7. Existing: `character.NewProcessor(l, ctx)` with `GetById(cp.InventoryDecorator)(characterId)` (`Hp() uint16`, `JobId() job.Id`, `Inventory().Consumable().FindBySlot(slot) (*asset.Model, bool)`); `p.cdp.GetById(itemId uint32) (data/consumable.Model, error)`; `consumer.GetManager().RegisterHandler`; `message.OneTimeConfig`.
- Produces: `(*Processor).RequestSkillBookUse(f field.Model, characterId uint32, slot int16, itemId item2.Id) error` — Task 10's command lands here via the consumer.

- [ ] **Step 1: Implement the processor methods**

Append to `consumable/processor.go`. New imports needed: `"encoding/json"`, `sagamsg "atlas-consumables/kafka/message/saga"`, `sagaonce "atlas-consumables/kafka/once/saga"`, `saga2 "atlas-consumables/saga"`, `skill2 "atlas-consumables/skill"` (the file already imports `math/rand`, `uuid`, `topic`, `consumer`, `message`, `producer`, `ts` for constants character, `item2`, `field` etc. — verify and add only what's missing):

```go
// rejectSkillBookUse warn-logs an FR-2 validation rejection and emits the
// canUse=false result event so the client's exclusive-request lock clears
// (the client only unlocks on receipt of SKILL_LEARN_ITEM_RESULT).
func (p *Processor) rejectSkillBookUse(characterId uint32, itemId item2.Id, isMasteryBook bool, skillId uint32, reason string) error {
	p.l.Warnf("Character [%d] skill book [%d] use rejected: %s.", characterId, itemId, reason)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(SkillBookResultEventProvider(ts.Id(characterId))(isMasteryBook, skillId, 0, false, false))
}

// RequestSkillBookUse handles a Skill Book (228) / Mastery Book (229) consume
// request (task-125). All eligibility gates run server-side (FR-2); every
// request produces exactly one SKILL_BOOK_RESULT event. Book destruction and
// the master-level grant ride one skill_book_use saga (destroy-first, so a
// duplicate request's destroy fails on the emptied slot rather than
// double-granting); the book is consumed on failed rolls too.
func (p *Processor) RequestSkillBookUse(f field.Model, characterId uint32, slot int16, itemId item2.Id) error {
	classification := item2.GetClassification(itemId)
	isMasteryBook := classification == item2.ClassificationConsumableMasteryBook
	if classification != item2.ClassificationConsumableSkillBook && !isMasteryBook {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, 0, "item is not a skill or mastery book")
	}

	cp := character.NewProcessor(p.l, p.ctx)
	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, 0, "unable to fetch character")
	}
	if c.Hp() == 0 {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, 0, "character is dead")
	}

	ci, err := p.cdp.GetById(uint32(itemId))
	if err != nil {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, 0, "unable to fetch consumable data")
	}
	if len(ci.Skills()) == 0 {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, 0, "book declares no skills")
	}

	a, ok := c.Inventory().Consumable().FindBySlot(slot)
	if !ok || a.TemplateId() != uint32(itemId) || a.Quantity() < 1 {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, 0, "slot does not hold the requested book")
	}

	targetSkillId, ok := SelectSkillBookTargetSkill(ci.Skills(), c.JobId())
	if !ok {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, 0, "no skill in book matches job")
	}

	skills, err := skill2.NewProcessor(p.l, p.ctx).GetByCharacterId(characterId)
	if err != nil {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, targetSkillId, "unable to fetch skills")
	}
	var hasRecord bool
	var currentLevel, currentMasterLevel byte
	var currentExpiration time.Time
	for _, sm := range skills {
		if sm.Id() == targetSkillId {
			hasRecord = true
			currentLevel = sm.Level()
			currentMasterLevel = sm.MasterLevel()
			currentExpiration = sm.Expiration()
			break
		}
	}
	if err := ValidateSkillBookSkillState(isMasteryBook, hasRecord, currentLevel, currentMasterLevel, ci.ReqSkillLevel(), ci.MasterLevel()); err != nil {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, targetSkillId, err.Error())
	}

	roll := rand.Int31n(100)
	rollPassed := SkillBookRollPasses(roll, ci.SuccessRate())
	p.l.Infof("Character [%d] skill book [%d] targeting skill [%d]: rolled [%d] against [%d], passed [%t].", characterId, itemId, targetSkillId, roll, ci.SuccessRate(), rollPassed)

	transactionId := uuid.New()
	s := BuildSkillBookSaga(transactionId, characterId, slot, itemId, rollPassed, hasRecord, currentLevel, currentExpiration, targetSkillId, byte(ci.MasterLevel()))

	bookMasterLevel := ci.MasterLevel()
	p.l.Debugf("Creating OneTime saga-status consumer to await skill book transaction [%s].", transactionId.String())
	t, _ := topic.EnvProvider(p.l)(sagamsg.EnvStatusEventTopic)()
	validator := sagaonce.TransactionValidator(transactionId)
	handler := func(l logrus.FieldLogger, hctx context.Context, e sagamsg.StatusEvent[json.RawMessage]) {
		completed := e.Type == sagamsg.StatusEventTypeCompleted
		if completed {
			l.Infof("Character [%d] skill book [%d] saga [%s] completed. Roll success [%t].", characterId, itemId, e.TransactionId, rollPassed)
		} else {
			var body sagamsg.StatusEventFailedBody
			_ = json.Unmarshal(e.Body, &body)
			l.Warnf("Character [%d] skill book [%d] saga [%s] failed at step [%s]: %s.", characterId, itemId, e.TransactionId, body.FailedStep, body.Reason)
		}
		if err := producer.ProviderImpl(l)(hctx)(consumable.EnvEventTopic)(SkillBookResultEventProvider(ts.Id(characterId))(isMasteryBook, targetSkillId, bookMasterLevel, completed, completed && rollPassed)); err != nil {
			l.WithError(err).Errorf("Character [%d] skill book result emission failed; client stays locked until relog.", characterId)
		}
	}
	if _, err := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler))); err != nil {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, targetSkillId, "unable to register saga result handler")
	}

	if err := saga2.NewProcessor(p.l, p.ctx).Create(s); err != nil {
		return p.rejectSkillBookUse(characterId, itemId, isMasteryBook, targetSkillId, "unable to submit saga")
	}
	return nil
}
```

Import-collision note: `consumable/processor.go` already imports the constants character package as `ts` and the service character package as `character`; check the actual aliases at the top of the file before writing (`ts.Id` is the `character.Id` constants type used by the event provider). `time` may already be imported — add if missing. The unused `f field.Model` parameter is intentional (parity with the command envelope and future use); name it `_` if `go vet` complains about nothing — it won't; keep `f` and reference it in the initial debug log if preferred.

- [ ] **Step 2: Wire the command consumer**

In `kafka/consumer/consumable/consumer.go`, add to `InitHandlers` after the `handleRequestScroll` registration:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestSkillBookUse))); err != nil {
			return err
		}
```

And append the handler:

```go
func handleRequestSkillBookUse(l logrus.FieldLogger, ctx context.Context, c consumable2.Command[consumable2.RequestSkillBookUseBody]) {
	if c.Type != consumable2.CommandRequestSkillBookUse {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	err := consumable.NewProcessor(l, ctx).RequestSkillBookUse(f, uint32(c.CharacterId), int16(c.Body.Slot), c.Body.ItemId)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to use skill book in slot [%d] as expected.", c.CharacterId, c.Body.Slot)
	}
}
```

(`field` is already imported in this file for `handleCancelConsumableEffect`.)

- [ ] **Step 3: Build and test**

Run (from `services/atlas-consumables/atlas.com/consumables`): `go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go
git commit -m "feat(atlas-consumables): RequestSkillBookUse validation, roll, saga submission, result gating (task-125)"
```

---

### Task 9: atlas-channel — result event consumer (canUse routing split)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go` (event mirror)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go` (new handler + registration)

**Interfaces:**
- Consumes: `charpkt.CharacterSkillLearnItemResultWriter` + `charpkt.NewSkillLearnItemResult(...)` (Task 2; the file already imports `charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"`).
- Produces: channel consumes `SKILL_BOOK_RESULT`: `canUse=true` → `ForSessionsInMap` (map broadcast; the client demuxes glow-for-everyone / sound+message-local); `canUse=false` → requester session only (design §2 D-2).

- [ ] **Step 1: Mirror the event**

In `kafka/message/consumable/kafka.go` (channel side), extend the event constants:

```go
	EnvEventTopic            = "EVENT_TOPIC_CONSUMABLE_STATUS"
	EventTypeError           = "ERROR"
	EventTypeScroll          = "SCROLL"
	EventTypeSkillBookResult = "SKILL_BOOK_RESULT"
```

Add after `ScrollBody`:

```go
type SkillBookResultBody struct {
	IsMasteryBook bool   `json:"isMasteryBook"`
	SkillId       uint32 `json:"skillId"`
	MasterLevel   uint32 `json:"masterLevel"`
	CanUse        bool   `json:"canUse"`
	Success       bool   `json:"success"`
}
```

- [ ] **Step 2: Add the consumer handler**

In `kafka/consumer/consumable/consumer.go`, register in `InitHandlers` after the scroll handler registration (same `handles = append(...)` pattern):

```go
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSkillBookResultEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
```

Append the handler (mirrors `handleScrollConsumableEvent`, splitting on `CanUse`):

```go
// handleSkillBookResultEvent announces SKILL_LEARN_ITEM_RESULT for a
// skill-book outcome (task-125). canUse results broadcast to the whole map
// (the client renders the glow on the user's avatar for every observer and
// plays sound/message only locally); validation rejections and saga failures
// (canUse=false) go to the requester only — just enough to clear the client's
// exclusive-request lock.
func handleSkillBookResultEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.SkillBookResultBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.SkillBookResultBody]) {
		if e.Type != consumable2.EventTypeSkillBookResult {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		announce := session.Announce(l)(ctx)(wp)(charpkt.CharacterSkillLearnItemResultWriter)(charpkt.NewSkillLearnItemResult(uint32(e.CharacterId), e.Body.IsMasteryBook, e.Body.SkillId, e.Body.MasterLevel, e.Body.CanUse, e.Body.Success).Encode)
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
			if e.Body.CanUse {
				return _map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(), announce)
			}
			return announce(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to process skill book result event for character [%d].", e.CharacterId)
		}
	}
}
```

(All imports — `session`, `_map`, `tenant`, `charpkt` — are already present in this file.)

- [ ] **Step 3: Build**

Run (from `services/atlas-channel/atlas.com/channel`): `go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go
git commit -m "feat(atlas-channel): SKILL_BOOK_RESULT consumer with map-broadcast/requester-only split (task-125)"
```

---

### Task 10: atlas-channel — USE_SKILL_BOOK handler + command producer + registrations

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_book_use.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go` (command mirror)
- Modify: `services/atlas-channel/atlas.com/channel/consumable/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/consumable/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (handlerMap entry ~line 891 area; produceWriters entry ~line 703 area)

**Interfaces:**
- Consumes: `charsb.CharacterSkillBookUseHandle` / `charsb.UseSkillBook` (Task 1); `charcb.CharacterSkillLearnItemResultWriter` (Task 2); `consumable.CommandRequestSkillBookUse` mirror (Step 1 below).
- Produces: `CharacterSkillBookUseHandleFunc` registered under the handle name; the writer name registered in `produceWriters()` so tenant `WriterConfig` can bind the opcode.

- [ ] **Step 1: Command mirror**

In `kafka/message/consumable/kafka.go` (channel side), extend the command constants:

```go
	CommandRequestItemConsume  = "REQUEST_ITEM_CONSUME"
	CommandRequestScroll       = "REQUEST_SCROLL"
	CommandRequestSkillBookUse = "REQUEST_SKILL_BOOK_USE"
```

Add after `RequestScrollBody`:

```go
type RequestSkillBookUseBody struct {
	Slot   slot.Position `json:"slot"`
	ItemId item.Id       `json:"itemId"`
}
```

- [ ] **Step 2: Producer + processor**

Append to `consumable/producer.go`:

```go
func RequestSkillBookUseCommandProvider(f field.Model, characterId character.Id, slot slot.Position, itemId item.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestSkillBookUseBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        consumable.CommandRequestSkillBookUse,
		Body: consumable.RequestSkillBookUseBody{
			Slot:   slot,
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

Append to `consumable/processor.go`:

```go
func (p *Processor) RequestSkillBookUse(f field.Model, characterId character.Id, slot slot.Position, itemId item.Id, updateTime uint32) error {
	p.l.Debugf("Character [%d] using skill book [%d] from slot [%d]. updateTime [%d]", characterId, itemId, slot, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestSkillBookUseCommandProvider(f, characterId, slot, itemId))
}
```

(`updateTime` is logged and dropped — no existing consumable command forwards it; same treatment as `RequestItemConsume`.)

- [ ] **Step 3: Socket handler**

`socket/handler/character_skill_book_use.go`:

```go
package handler

import (
	"atlas-channel/consumable"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterSkillBookUseHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.UseSkillBook{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = consumable.NewProcessor(l, ctx).RequestSkillBookUse(s.Field(), character.Id(s.CharacterId()), slot.Position(p.Slot()), item.Id(p.ItemId()), p.UpdateTime())
	}
}
```

- [ ] **Step 4: main.go registrations**

In `produceHandlers()` (handlerMap block, near `handlerMap[invsb.CharacterItemUseHandle]` at line ~879):

```go
	handlerMap[charsb.CharacterSkillBookUseHandle] = handler.CharacterSkillBookUseHandleFunc
```

In `produceWriters()` (near `charcb.CharacterItemUpgradeWriter` at line ~703):

```go
		charcb.CharacterSkillLearnItemResultWriter,
```

(`charsb` and `charcb` aliases already exist in main.go's imports.)

- [ ] **Step 5: Build and test**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_skill_book_use.go services/atlas-channel/atlas.com/channel/consumable/ services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(atlas-channel): USE_SKILL_BOOK handler and SKILL_LEARN_ITEM_RESULT writer wiring (task-125)"
```

---

### Task 11: atlas-configurations — seed templates (10 versions)

**Files (modify — 10 templates; `template_gms_12_1.json` is intentionally NOT touched):**
- `template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_72_1.json`, `template_gms_79_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`

**Interfaces:**
- Consumes: handler name `CharacterSkillBookUseHandle` (Task 1), writer name `CharacterSkillLearnItemResult` (Task 2), validator `LoggedInValidator`.
- **Opcodes (IDB-verified; collision-checked FREE at plan time — re-check in Step 3):**

| Template | handler opCode | writer opCode |
|---|---|---|
| gms_48  | 0x40 | 0x2B |
| gms_61  | 0x4B | 0x30 |
| gms_72  | 0x51 | 0x30 |
| gms_79  | 0x50 | 0x30 |
| gms_83  | 0x52 | 0x33 |
| gms_84  | 0x52 | 0x33 |
| gms_87  | 0x55 | 0x33 |
| gms_92  | 0x59 | 0x34 |
| gms_95  | 0x58 | 0x32 |
| jms_185 | 0x4A | 0x30 |

- **`gms_12` is EXCLUDED** — the feature is genuinely absent in that client (no send/result opcode). This is the only exclusion; it is evidence-backed, not a deferral. The PR description should note it in one line.

- [ ] **Step 1: Insert handler entries (one per template, 10 total)**

JSON entry order is irrelevant (`BuildHandlerMap` builds a map), so the exact anchor does not matter — insert the new handler object adjacent to an existing item-use handler in that template's `socket.handlers` array. Use `CharacterItemUseScrollHandle` as the anchor where present; where it is absent use `CharacterItemUseHandle`; for **gms_92** (minimal template — only `CharacterCashItemUseHandle`) anchor on `CharacterCashItemUseHandle`. The inserted object is always:

```json
      {
        "opCode": "<handler opCode from table>",
        "validator": "LoggedInValidator",
        "handler": "CharacterSkillBookUseHandle"
      }
```

Every entry MUST carry `"validator": "LoggedInValidator"` even when the anchor block lacks a validator line (e.g. gms_95's `CharacterItemUseScrollHandle` has none) — a validator-less handler is silently dropped at runtime. Do per-file Edits (no shell patch loop); preserve each file's existing line endings.

- [ ] **Step 2: Insert writer entries (one per template, 10 total)**

Insert the new writer object adjacent to any existing writer entry in that template's `socket.writers` array (e.g. `CharacterItemUpgrade` where present; otherwise any writer object):

```json
      {
        "opCode": "<writer opCode from table>",
        "writer": "CharacterSkillLearnItemResult"
      }
```

- [ ] **Step 3: Validate JSON + collision check (all 10 templates)**

```bash
python3 - <<'EOF'
import json
base="services/atlas-configurations/seed-data/templates"
specs=[("template_gms_48_1.json","0x40","0x2B"),("template_gms_61_1.json","0x4B","0x30"),
       ("template_gms_72_1.json","0x51","0x30"),("template_gms_79_1.json","0x50","0x30"),
       ("template_gms_83_1.json","0x52","0x33"),("template_gms_84_1.json","0x52","0x33"),
       ("template_gms_87_1.json","0x55","0x33"),("template_gms_92_1.json","0x59","0x34"),
       ("template_gms_95_1.json","0x58","0x32"),("template_jms_185_1.json","0x4A","0x30")]
for f,h,w in specs:
    d=json.load(open(f"{base}/{f}"))
    hs=[e for e in d["socket"]["handlers"] if int(e["opCode"],16)==int(h,16)]
    ws=[e for e in d["socket"]["writers"] if int(e["opCode"],16)==int(w,16)]
    assert len(hs)==1 and hs[0]["handler"]=="CharacterSkillBookUseHandle" and hs[0].get("validator")=="LoggedInValidator", (f,"handler",hs)
    assert len(ws)==1 and ws[0]["writer"]=="CharacterSkillLearnItemResult", (f,"writer",ws)
    print(f, "OK")
# gms_12 must NOT have been touched
d12=json.load(open(f"{base}/template_gms_12_1.json"))
assert not any(e.get("handler")=="CharacterSkillBookUseHandle" for e in d12["socket"]["handlers"]), "gms_12 must stay excluded"
print("template_gms_12_1.json untouched OK")
EOF
```

Expected: ten `OK` lines + the gms_12 assertion (also proves each file still parses).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(atlas-configurations): seed USE_SKILL_BOOK handler + SKILL_LEARN_ITEM_RESULT writer for 10 versions (task-125)"
```

---

### Task 12: Workspace verification (CLAUDE.md gates)

**Files:** none new — fix whatever the gates surface.

- [ ] **Step 1: Per-module tests/vet/build**

From each changed module root (`libs/atlas-packet`, `libs/atlas-saga`, `services/atlas-consumables/atlas.com/consumables`, `services/atlas-channel/atlas.com/channel`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`):

```bash
go test -race ./... && go vet ./... && go build ./...
```

Expected: clean everywhere.

- [ ] **Step 2: Redis key guard**

From the worktree root: `tools/redis-key-guard.sh`
Expected: clean (no new raw Redis usage was introduced).

- [ ] **Step 3: Docker bakes**

`libs/atlas-saga` and `libs/atlas-packet` are imported by many services — bake everything rather than guessing the blast radius:

```bash
docker buildx bake all-go-services
```

Expected: all targets build. If any service fails on a missing `COPY libs/...`, that's a Dockerfile gap — no new lib was added in this task, so a failure means something else; investigate, don't shortcut.

- [ ] **Step 4: Commit any fixes**

```bash
git add -A && git commit -m "fix(task-125): verification-gate fixes" # only if fixes were needed
```

---

### Task 13: Fixture campaign — promote all 20 cells (10 versions × 2 packets)

**Files:**
- Modify: `libs/atlas-packet/character/serverbound/use_skill_book_test.go` (per-version markers)
- Modify: `libs/atlas-packet/character/clientbound/skill_learn_item_result_test.go` (per-version markers)
- Create: per-fname audit reports under `docs/packets/audits/gms_v48/`, `gms_v61/`, `gms_v72/`, `gms_v79/`, `gms_v83/`, `gms_v84/`, `gms_v87/`, `gms_v92/`, `gms_v95/`, `jms_v185/` (surgical splice per cell — never a full re-export)
- Modify: evidence records + regenerated `docs/packets/audits/STATUS.md` (rows 72, 570)
- **Correct** the stale audit artifacts (see below).

**All IDBs are loaded and every function is located + named** (the derivation pass in this task already did this). **No cell is IDB-blocked.** Dispatch the `packet-verifier` agent per cell, serialized (shared run.go / matrix / global IDA instance). Follow `docs/packets/audits/VERIFYING_A_PACKET.md` exactly; this plan does not restate the playbook.

**IDB-verified addresses + opcodes (do NOT re-derive from the CSV/support-docs — they are wrong for several cells):**

| version | serverbound `Send…` @addr → op | clientbound `On…Result` @addr → op | body |
|---|---|---|---|
| gms_48  | 0x70E3E7 → 0x40 | 0x71A135 → 0x2B | 15-byte |
| gms_61  | 0x8325D2 → 0x4B | 0x841E5F → 0x30 | 15-byte |
| gms_72  | 0x904B55 → 0x51 | 0x9175E6 → 0x30 | 15-byte |
| gms_79  | 0x955EBD → 0x50 | 0x969022 → 0x30 | 15-byte |
| gms_83  | 0xA0A1B2 → 0x52 | 0xA1E5AF → 0x33 | 15-byte |
| gms_84  | 0xA5459C → 0x52 | 0xA6984E → 0x33 | 16-byte |
| gms_87  | 0xA9FA66 → 0x55 | 0xAB58E8 → 0x33 | 16-byte |
| gms_92  | 0x9AB570 → 0x59 | 0x9CC640 → 0x34 | 16-byte |
| gms_95  | 0x9D65E0 → 0x58 | 0x9F7AF0 → 0x32 | 16-byte |
| jms_185 | 0xAEEE61 → 0x4A | 0xB05116 → 0x30 | 16-byte |

- [ ] **Step 1: Serverbound cells (10).** For each version, splice the per-fname `CWvsContext::SendSkillLearnItemUseRequest` report from the address above into `docs/packets/audits/<ver>/` (surgical), add `// packet-audit:verify packet=USE_SKILL_BOOK version=<ver> ida=0x...` to `use_skill_book_test.go`, pin evidence, regenerate the matrix. Body is uniform (updateTime/slot/itemId) — the golden already asserts it.
- [ ] **Step 2: Clientbound cells (10).** Same for `CWvsContext::OnSkillLearnItemResult`. **Split by body shape:** v48/61/72/79/83 fixtures assert the 15-byte body; v84/87/92/95/jms fixtures assert the 16-byte body (leading `bOnExclRequest`). This confirms the codec's `MajorVersion() >= 84` gate against real client reads.
- [ ] **Step 3: Correct the stale audit artifacts.** As part of the campaign, fix the wrong cells in `docs/packets/audits/support/gms_v48.md` (serverbound + clientbound were `n-a`), `gms_v61/72/79.md` (serverbound `n-a`), and the `MapleStory Ops - *.csv` where they disagree with the verified opcodes. Note the corrections in the commit message so future version passes are not re-poisoned.
- [ ] **Step 4: If any fname fails to resolve — STOP and escalate** (not expected; all are named). Never substitute or fake.
- [ ] **Step 5: Commit per cell** (test marker + evidence + matrix together), e.g.:

```bash
git add libs/atlas-packet/character/serverbound/use_skill_book_test.go docs/packets/audits/
git commit -m "verify(packets): USE_SKILL_BOOK gms_v48 byte fixture + evidence (task-125)"
```

Exit state: all 20 cells (2 rows × 10 versions) promoted; `gms_12` has no column for either row (feature absent; nothing to record).

---

### Task 14: Code review

- [ ] **Step 1:** Invoke `superpowers:requesting-code-review` — it dispatches `plan-adherence-reviewer` (against this plan) and `backend-guidelines-reviewer` (Go changes) in parallel; findings land in `docs/tasks/task-125-skill-mastery-books/audit.md`.
- [ ] **Step 2:** Address findings per `superpowers:receiving-code-review` (verify before implementing; push back with evidence where a finding is wrong).
- [ ] **Step 3:** Commit fixes; re-run Task 12's gates if code changed.

---

### Task 15: Post-merge live rollout + acceptance (runs AFTER the PR merges and images deploy)

Seed templates only apply at tenant creation — existing tenants need a live PATCH + channel restart (new-opcodes gotcha: handlers/writers do not hot-reload).

- [ ] **Step 1:** For each live tenant on **any of the 10 lineages (gms_48/61/72/79/83/84/87/92/95/jms)**: PATCH the tenant's socket configuration (via atlas-tenants configurations REST, same procedure as prior opcode rollouts) adding the handler entry (with `LoggedInValidator`) and writer entry using the Task 11 opcode table. Full sweep — every tenant, not a spot-check. (gms_12 tenants: skip — feature absent.)
- [ ] **Step 2:** Restart atlas-channel pods for the affected tenants.
- [ ] **Step 3:** Live acceptance on a v83 tenant (PRD §10):
  - Mastery book (e.g. 2290000, success 70): use with an eligible character → on success master level rises (verify via skills REST/UI), book consumed, map-broadcast glow + local sound/message; on failure book still consumed, failure effect shown.
  - Ineligibility (wrong job / unlearned skill / master level at cap): "cannot use" path — no consumption, client not wedged (excl lock cleared by the canUse=0 packet).
  - Skill book (e.g. 2280000, success 100, reqSkillLevel 0): teaches the unlearned skill at level 0 with master level 10.
  - Compensation: hardest to force live — verified by the Task 4 unit tests; live check optional (e.g. transient skills-service outage) — do not block acceptance on it.
- [ ] **Step 4:** Check atlas-channel logs for `unhandled message op` on 0x52 (would indicate a missed tenant PATCH) and atlas-consumables logs for rejection reasons.

---

## Self-review checklist (ran at plan time)

- **Spec coverage:** FR-1 → Tasks 1, 10, 11; FR-2 → Tasks 7, 8; FR-3 → Tasks 3, 4, 7, 8; FR-4 → Tasks 2, 9, 11; FR-5 → Tasks 11, 15 (10 versions wired; only gms_12 excluded — evidence-backed feature-absence, exceeds the original FR-5.1 six-template list); FR-6 → Task 13 (all 20 cells promotable now — every IDB loaded; gms_12 has no column). Design §5.1-§5.6, §6 error matrix, §7 testing, §8 fixtures, §9 risks all mapped.
- **v2 scope change (post-`main` merge, 2026-07-25):** version set 5 → 10 (all but gms_12), all opcodes IDB-verified, clientbound codec gains a `bOnExclRequest` byte gated at `MajorVersion() >= 84` (a v84≠v83 exception). The "gms_92 excluded / IDB-blocked" v1 assumptions are void. See design §0.
- **Design deviations (intentional):** (1) no `socket/writer/` file in atlas-channel — current codebase pattern announces lib codecs directly by writer name (see `CharacterItemUpgradeWriter`); a thin writer file would be dead code. (2) Skill saga payloads follow the shared-library precedent (no WorldId) used by all three existing producers.
- **Type consistency:** `CharacterSkillBookUseHandle` / `CharacterSkillLearnItemResultWriter` / `NewSkillLearnItemResult(uint32, bool, uint32, uint32, bool, bool)` / `RequestSkillBookUse(field.Model, character.Id, slot.Position, item.Id, uint32)` (channel) vs `RequestSkillBookUse(field.Model, uint32, int16, item2.Id)` (consumables) — names verified consistent across Tasks 1-11.
- **Placeholders:** none — every code step carries the actual code.
