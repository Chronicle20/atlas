# Pet Name Tag (5170000) — Design

Task: task-224-pet-name-tag
PRD: [`prd.md`](prd.md) (approved)
Date: 2026-08-13
Status: Draft for review

---

## 0. What this document adds to the PRD

The PRD scoped the feature and named five open questions. This design closes all five
against the clients themselves, and in doing so **corrects three PRD requirements** that
would have shipped wrong. Those corrections are stated up front because they change the
implementation, not just the prose.

| # | PRD said | Evidence says | Requirement affected |
|---|---|---|---|
| C-1 | The `PET_NAMECHANGE` body is `str name` + `byte flag` on every version. | True for GMS (v61/v83/v92/v95 read). **False for JMS v185** — `CPet::OnNameChanged` @`0x76a5de` performs exactly one `DecodeStr` and *no* `Decode1`; it branches on `sub_768D82(this)`, a client-side state query, not a wire byte. | FR-6.2 → needs a **region gate**, not a uniform body. |
| C-2 | Accepted name length is 1–13 (DB ceiling). | The client's own input dialog is bounded `min 4, max 12` — `sub_9AC7CB(dlg, NULL, 4, 12, 0, 1)` @`0xa0bb2f`, the identical bounds the character-name field uses (`CTabParty::OnInvite` @`0x90e1da`). | FR-4.1 → bounds become **4–12**. |
| C-3 | The `RENAME` command / `NAME_CHANGED` event bodies are `{petId, characterId, name}`. | The saga cannot complete a `rename_pet` step without a correlating `transactionId` (the `EvolvePet` precedent carries one), and atlas-channel cannot address the clientbound packet without the pet's **slot**. | FR-5.1 / FR-5.2 / §5.2 → bodies gain `transactionId`, `slot`, `previousName`. |

Everything else in the PRD stands as written.

---

## 1. Evidence — the five open questions, closed

All addresses below were read this session from the checked-in IDBs via ida-pro-mcp.
Nothing here is inferred from general MapleStory knowledge.

### OQ-1 — Per-version WZ parity: **closed, no atlas-data change on any version**

`Item.wz/Cash/0517.img.xml` in the local GMS 83.1 tree contains exactly:

```xml
<imgdir name="05170000"><imgdir name="info">
  <int name="z" value="0"/><int name="slotMax" value="1"/><int name="cash" value="1"/>
</imgdir></imgdir>
```

plus icon canvases. No value node.

Only the 83.1 tree is extracted locally, so per-version parity cannot be *observed* for the
other nine. It does not need to be: **this feature reads no WZ value at all.** The rename
payload is player-supplied and the consumption is template-keyed. Unlike task-220's meso
sack — where the payout *was* a WZ field and a stale ingest silently paid zero — there is no
node whose presence or absence changes behaviour here. The "no atlas-data change, no
re-ingest prerequisite" conclusion therefore holds by construction rather than by survey.

### OQ-2 — Server-side rejection UX: **closed, option (b) — pink text**

There is a concrete, already-load-bearing message op, and it is present in **all ten**
socket templates (`grep -c '"WorldMessage"'` returns 1 for every
`services/atlas-configurations/seed-data/templates/template_*.json`):

- writer `chatpkt.WorldMessageWriter`
- body `writer.WorldMessagePinkTextBody("", "", msg)`

Two existing failure paths already use exactly this pair and then release the
exclusive-request gate — `point_reset`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer.go:347-360`) and
`meso_sack_use` (`:365-381`). This design reuses that shape verbatim. The user's stated
preference for a real error message is satisfiable; the fallback to a silent
`enableActions()` is **not** needed.

### OQ-3 — Profanity filtering: **out of scope, unchanged**

FR-4.5 stands. The repo has no profanity service to reuse; the client still runs
`CCurseProcess::ProcessString` @`0xa0bb9a`, so the gap is only reachable by a modified
client, and the worst outcome is a rude pet name — not state corruption.

### OQ-4 — Multi-pet disambiguation: **closed, the client picks pet-locker index 0**

At the case-17 arm entry:

```
0xa0ba44  push ebx                  ; a2  (ebx == 0 throughout this arm — it is the
0xa0ba45  mov  ecx, esi             ;      null/zero register: `cmp esi, ebx; jz default`
0xa0ba47  call sub_46D2D5           ;      at 0xa0ba4e, `cmp eax, ebx` at 0xa0bac9)
0xa0ba4e  cmp  esi, ebx
0xa0ba50  jz   def_A0A6E6           ; no pet at that index → the arm is abandoned, nothing sent
```

and `sub_46D2D5` @`0x46d2d5` is:

```c
CashItemSlotPosition = CharacterData::FindCashItemSlotPosition(this, 5, *(this + 8 * a2 + 27));
```

— it resolves the CASH-inventory item backing pet-locker entry `a2`. With `a2 = 0` the
client renames **the pet in locker index 0**, which is Atlas's `slot == 0`, i.e.
`pet.Model.Lead()`. FR-3.1's "lead active pet" rule is therefore *identical* to the client's
choice, not merely compatible with it. No realignment needed.

Note the second half of that jump: when index 0 is empty the client **abandons the arm
before any dialog and sends nothing**. So the no-active-pet rejection (FR-3.2) is a
crafted-packet path, not something an unmodified client can produce — it must still
fail closed, but it will be rare.

### OQ-5 — Client input-field cap: **closed, min 4 / max 12**

```
0xa0bb1e  push edi        ; a6 = 1
0xa0bb1f  push ebx        ; a5 = 0
0xa0bb20  push 0Ch        ; a4 = 12
0xa0bb22  push 4          ; a3 = 4
0xa0bb24  push ebx        ; a2 = NULL (no initial text)
0xa0bb2f  call sub_9AC7CB
```

`sub_9AC7CB` stores `this[55] = a3` and `this[51] = a4`. The (min, max) reading is
established by three sibling call sites in the same client:

| Caller | Args (a3, a4) | Known real bound |
|---|---|---|
| `CTabParty::OnInvite` @`0x90e1da` | 4, 12 | character name — exactly 4–12 |
| `ask_SPW` @`0x9ad030` | 8, 8 | second password — exactly 8 |
| `ask_guildname` @`0x9ad131` | 4, 12 | guild name — max 12 |
| adjacent cash arm @`0xa0aa57` | 0, 60 | free-text message, no minimum |
| adjacent cash arm @`0xa0cf1b` | 0, 40 | free-text message, no minimum |

So a3 is the minimum and a4 the maximum. **A pet name the v83 UI can produce is 4–12
characters.** The DB column is 13 (`services/atlas-pets/atlas.com/pets/pet/entity.go:23`),
which is one wider than anything the client will send — comfortable, not binding.

### Additional finding — the FR-1.2 overflow is real

`item.Id` is `uint32` (`libs/atlas-constants/item/constants.go:5`). At
`character_cash_item_use.go:937`, `10000*itemId/10000` with `itemId = 5170000` computes
`51,700,000,000 mod 2^32 = 160,392,448`, `/10000 = 16039 ≠ 5170000`, so the branch returns
`CashSlotItemType(0)` and the item never reaches a handler. Confirmed as the PRD described.
No other classification in `GetCashSlotItemType` returns 17 (meso sacks return 19 on every
version by deliberate Atlas policy — see the comment at `:702-708`), so type 17 is an
unambiguous key and the arm may gate on `it` rather than on classification.

### Additional finding — the clientbound framing (FR-6.3) is already settled

`ownerId uint32` + `slot int8` precede every pet leaf body; they are read upstream by
`CUser::OnPetPacket` before the per-op dispatch. This is documented and byte-verified for
the whole pet family in `libs/atlas-packet/pet/clientbound/v61_test.go:11-14`, and every
existing pet codec (`chat.go:37-38`, `command.go:51-52`, `movement.go`, `exclude.go`)
encodes exactly that prefix. `PET_NAMECHANGE` inherits it unchanged. Nothing to guess.

### Additional finding — the trailing `update_time` applies here too

The case-17 success path falls through to the shared tail at `loc_A0E9EC` after its
`EncodeStr` @`0xa0bcb5` — the same tail every other sub-body uses. So the serverbound
sub-body carries the standard `updateTimeFirst` gate, exactly like
`libs/atlas-packet/cash/serverbound/item_use_kite.go`, which is the direct model for the new
type.

### Additional finding — what the clientbound `flag` byte selects

GMS v95's PDB-backed decompile names it: `if (Decode1()) nNameTag = this->m_pTemplate->nNameTag; else nNameTag = 0;`
(`0x6a125a`–`0x6a1271`), passed to `CLife::MakeNameTag`. It is a *decoration layer selector*
sourced from the pet template — not a boolean "has a name". v83 (`0x704840`), v61
(`0x613615`) and v92 (`0x69`-arm) do the same thing through unnamed offsets.

The same value already exists on the wire: `Activated.nameTag`
(`libs/atlas-packet/pet/clientbound/activated.go:26,62`), which the spawn body writes and
which Atlas currently always leaves `0`. **The two must agree**, or a renamed pet's tag
decoration would appear on rename and vanish on the next respawn.

---

## 2. Architecture

### 2.1 Flow

```
client                 atlas-channel                 orchestrator            atlas-pets
  │  USE_CASH_ITEM (type 17, str name)
  ├──────────────────────►│
  │                        GetCashSlotItemType → 17          (FR-1.2 fix)
  │                        cashItemInSlotFunc  → owns 5170000
  │                        pet.GetByOwner → lead (slot==0)
  │                        normalize + validate name (4–12)
  │                        ── any failure ──► pink text + enableActions, saga never starts
  │                        saga.Create(PetNameTagUse)
  │                        ├────────────────────►│
  │                        │        step 1 rename_pet ──── RENAME{txn,petId,name} ──►│
  │                        │                                                          updateName
  │                        │◄──────── NAME_CHANGED{txn,petId,slot,name,previousName} ─┤
  │                        │        AcceptEvent → step 1 Completed
  │                        │        step 2 consume_pet_name_tag (DestroyAsset)
  │◄─ PET_NAMECHANGE (map broadcast, driven by NAME_CHANGED) ─┤
  │◄─ INVENTORY_OPERATION (owner-only cash-asset re-announce)  │
  │◄─ INVENTORY_OPERATION (consume; clears the excl-request gate)
```

The name lives in two client-side records and `PET_NAMECHANGE` only updates one.
It repaints the name tag over the *spawned* pet; the *inventory slot* renders
from `GW_ItemSlotPet.sPetName`, which no pet packet writes. The `NAME_CHANGED`
consumer therefore also re-announces the owner's cash pet asset — the same
`petAssetRefresher` seam every other pet status handler
(closeness/fullness/level/flag) uses. Without it the renamed item keeps the old
name until an unrelated full inventory re-send (cash shop entry/exit) corrects
it.

On a `consume_pet_name_tag` failure the compensator reverse-walks the completed
`rename_pet` step by issuing a second `RENAME` carrying `previousName`, then emits
`StatusEventTypeFailed`, which the channel renders as pink text.

### 2.2 Ownership boundaries

| Concern | Owner | Why here |
|---|---|---|
| Wire decode of the type-17 sub-body | `libs/atlas-packet/cash/serverbound` | Where every other sub-body lives. |
| Wire encode of `PET_NAMECHANGE` | `libs/atlas-packet/pet/clientbound` | Where the rest of the pet family lives; shares the `ownerId`+`slot` prefix. |
| Which pet, and is the name legal | atlas-channel handler | It is the only layer holding the session, the cash-slot ownership proof, and the pet list. |
| Name bounds as a *value* | `libs/atlas-constants/pet` | Two modules must agree on 4/12; a duplicated numeric bound is precisely the drift class the buff-duration guard exists to police. |
| Persisting the name, emitting the event | atlas-pets | Sole owner of the `pets` table. |
| Ordering rename-before-consume, and undoing | atlas-saga-orchestrator | Sole owner of cross-service ordering. |
| Rendering the rejection | atlas-channel saga-failed consumer | Where `point_reset` and `meso_sack_use` already render theirs. |

---

## 3. Component design

### 3.1 `libs/atlas-constants/pet` — new `name.go`

```go
const (
    MinNameLength = 4   // client input dialog min, GMS v83 sub_9AC7CB(…, 4, 12, …) @0xa0bb2f
    MaxNameLength = 12  // …and its max. The pets.name column is size:13 — one wider, deliberately.
)

// NormalizeName trims the surrounding whitespace the client's own TrimLeft/TrimRight
// removes before it encodes, so both sides validate the same string.
func NormalizeName(s string) string

// ValidateName reports whether a normalized name is acceptable. Callers MUST normalize first.
func ValidateName(s string) error
```

Both atlas-channel and atlas-pets import it. This is the DOM-21 answer to "the bound is
written down twice"; `libs/atlas-constants/pet/` already exists (it carries `pet/skill`), and
`services/atlas-pets` already depends on the module
(`services/atlas-pets/atlas.com/pets/go.mod:6`).

**Rejected alternative:** duplicating the literals in each service with a cross-referencing
comment. That is the `ApplyCommandBody.Duration` failure mode — a contract that lives only
in prose gets flipped.

### 3.2 `libs/atlas-packet/cash/serverbound/item_use_pet_name_tag.go`

A near-clone of `item_use_kite.go`: one `WriteAsciiString`/`ReadAsciiString` plus the
`updateTimeFirst` trailing gate, with the IDA-derivation doc comment (case-17 arm entry
`0xa0ba15`, sole `EncodeStr` `0xa0bcb5`, shared tail `loc_A0E9EC`).

```go
type ItemUsePetNameTag struct {
    name            string
    updateTime      uint32
    updateTimeFirst bool
}
func NewItemUsePetNameTag(updateTimeFirst bool) *ItemUsePetNameTag
```

Per-version re-derivation (FR-2.4) is a per-cell step, not an assumption: the arm's *case
index* differs between clients (v48's table maps 520→17, v61's maps it to 18 — see the
`CashSlotItemTypeCurrencySack` comment at `character_cash_item_use.go:702-708`), but the
index never rides the wire. Atlas dispatches on its own server-resolved classification, so
only the *body shape* must be confirmed per version. Where a version's client has no arm for
classification 517 at all, that version's serverbound cell is recorded `n-a` and must pass
the n-a consistency gate.

### 3.3 `libs/atlas-packet/pet/clientbound/name_changed.go`

```go
const PetNameChangedWriter = "PetNameChanged"

// nameTagLayer is the CLife::MakeNameTag decoration selector. 0 = no template layer.
// It MUST equal Activated.nameTag (activated.go), which the spawn body writes for the
// same pet: a rename that sets 1 while the spawn writes 0 makes the decoration appear
// on rename and disappear on the next respawn.
const nameTagLayer = byte(0)

// packet-audit:fname CPet::OnNameChanged
type NameChanged struct {
    ownerId uint32
    slot    int8
    name    string
    nameTag byte
}
```

Encode:

```go
w.WriteInt(m.ownerId)      // upstream, CUser::OnPetPacket
w.WriteInt8(m.slot)        // upstream
w.WriteAsciiString(m.name) // DecodeStr — every version
if t.IsRegion("GMS") {     // Decode1 — GMS only; JMS v185 @0x76a5de has no Decode1
    w.WriteByte(m.nameTag)
}
```

`Decode` mirrors it. The region gate is the `reactor/serverbound/hit.go:45` idiom
(`t.IsRegion(…) && t.MajorAtLeast(…)` / `t.Region() == "JMS"`); here no major boundary is
involved, so it degenerates to a region test. **No raw `> N` comparison anywhere.**

To eliminate the drift risk in the constant, `activated.go` is changed to write
`nameTagLayer` instead of its zero-valued struct field, so one symbol feeds both codecs.
This is a targeted improvement inside the file the change already touches, not unrelated
refactoring.

Per-version derivation status entering implementation: **v61 ✓ read** (`0x6135d6`),
**v83 ✓ read** (`0x704801`), **v92 ✓ read** (`0x6967c0`), **v95 ✓ read** (`0x6a11f0`),
**jms_v185 ✓ read** (`0x76a5de`, the divergence). v72 / v79 / v84 / v87 remain to be read
per cell; the expectation from the four GMS reads is `str + byte`, but the byte-fixture
step is what proves it.

### 3.4 atlas-channel — dispatch and handler

`character_cash_item_use.go`:

- add `CashSlotItemTypePetNameTag = CashSlotItemType(17)` to the const block;
- replace `if 10000*itemId/10000 != itemId` with `if itemId%10000 != 0` in the
  `ClassificationPetImprints` branch, with a comment naming
  `get_cashslot_item_type` @`0x48645b` `case 517` and the `uint32` overflow it fixes;
- add the type-17 arm delegating to `handlePetNameTagUse` in a new sibling file
  `character_cash_item_use_pet_name_tag.go`.

`handlePetNameTagUse` (mirrors `handleMesoSackUse`'s guard shape):

1. `enableActions := …StatChanged(empty, true)` closure — the sole unlock on every
   rejection path. Nothing here warps, so this is the correct unlock
   (`reference_exclrequest_unlock_contract`).
2. Resolve pets via a package-var seam over `pet.NewProcessor(l, ctx).GetByOwner(characterId)`
   (test-seam precedent: `cashItemDataFunc`, `cashItemInSlotFunc`). Select the entry with
   `Slot() == 0`. Reject if absent (FR-3.2). Re-check `OwnerId() == s.CharacterId()`
   (FR-3.3) — belt and braces over a processor that already filters by owner.
3. `name := petconst.NormalizeName(sp.Name())`; `petconst.ValidateName(name)`; reject on
   error (FR-4.2/4.3), and reject when `name == pm.Name()` — a no-op rename must not cost
   a tag.
4. Every rejection: `WorldMessagePinkTextBody` + `enableActions()`, logging the resolved
   pet id, character id, and reason (NFR observability). No silent `Warnf` fallthrough.
5. Success: `saga.NewProcessor(l, ctx).Create(buildPetNameTagUseSaga(uuid.New(), time.Now(),
   characterId, itemId, petId, name, pm.Name()))`.

The pet-resolution rule gets a doc comment citing this design §1/OQ-4 and the
`sub_46D2D5(…, 0)` evidence, per FR-3.4.

New consumer arm in `kafka/consumer/pet/consumer.go`, `handleNameChanged`, following
`handleCommandResponse`'s shape:

```go
_ = _map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(),
        session.Announce(l)(ctx)(wp)(petpkt.PetNameChangedWriter)(
            petpkt.NewPetNameChanged(e.OwnerId, e.Body.Slot, e.Body.Name).Encode))
```

`ForSessionsInMap` includes the owner, so one call covers both the player and observers.
The callback closes over immutable values only — no shared state is mutated inside the
parallel iteration (`bug_channel_foreachinmap_parallel_shared_state`).

New arm in the saga-failed consumer for `saga.SagaTypePetNameTagUse`: pink text from a
`petNameTagFailureMessage(errorCode)` mapper (generic string on `ErrorCodeUnknown`, the
`mesoSackFailureMessage` shape) followed by `enableActions`.

Writer registration in `main.go` alongside the other pet writers.

### 3.5 atlas-pets

- `pet/builder.go`: `SetName(string) *ModelBuilder`; `Clone` already threads `name` through
  the constructor, so `Build`'s `name == ""` guard keeps protecting the invariant.
- `pet/administrator.go`: `updateName(db)(petId uint32, name string) error`, shaped like
  `updateLevel`.
- `pet/processor.go`: `RenameAndEmit(transactionId, petId, actorId, name)` /
  `Rename(mb)(…)` — the `AttemptCommandAndEmit`/`EvolveAndEmit` pair shape.
  Order inside `Rename`: read the pet (yields `previousName`, `slot`, and the ownership
  check), re-validate with `petconst.ValidateName` (FR-5.6 — atlas-pets does not trust the
  channel), write, then buffer the status event.
- `pet/producer.go`: `NameChangedStatusEventProvider`.
- `kafka/consumer/pet/consumer.go`: `handleRenameCommand`, registered next to
  `handleEvolveCommand`.
- `pet/resource.go`: `PATCH /pets/{petId}` via `rest.RegisterInputHandler[RestModel]`,
  writable attribute `name` only, mapping validation→400, missing→404, wrong owner→403.
  Explicitly an operator surface; the gameplay path never calls it.

**Idempotency (FR-5.5).** `RowsAffected == 0` is treated as "not found" only *after* the
pre-read has already proven the row exists — so a redelivered `RENAME` whose value is
already applied completes rather than erroring, and still emits `NAME_CHANGED`. That
re-emission is required, not incidental: it is how a redelivered command still completes the
orchestrator's `rename_pet` step.

### 3.6 Kafka contract — five files, no guard

The pet contract is mirrored in five modules:

```
services/atlas-pets/…/kafka/message/pet/kafka.go              (owner)
services/atlas-channel/…/kafka/message/pet/kafka.go
services/atlas-saga-orchestrator/…/kafka/message/pet/kafka.go
services/atlas-consumables/…/kafka/message/pet/kafka.go       (subset — no change needed)
services/atlas-messages/…/kafka/message/pet/kafka.go          (subset — no change needed)
```

The first three each gain:

```go
CommandPetRename            = "RENAME"
StatusEventTypeNameChanged  = "NAME_CHANGED"

type RenameCommandBody struct {
    Name string `json:"name"`
}                                     // petId + actorId are already on Command[E]

type NameChangedStatusEventBody struct {
    Slot          int8      `json:"slot"`
    Name          string    `json:"name"`
    PreviousName  string    `json:"previousName"`
    TransactionId uuid.UUID `json:"transactionId"`
}                                     // petId + ownerId are already on StatusEvent[E]
```

There is no mirror guard for the pet contract (only trade has one —
`tools/trade-contract-mirror-guard.sh`), so a json-tag typo here fails no build and decodes
to a zero-valued body at runtime. **Mitigation: a byte-level round-trip test in atlas-channel
and atlas-saga-orchestrator that unmarshals a fixture emitted from atlas-pets' struct
definition**, which turns the silent seam into a red test. Writing a sixth mirror guard is
deliberately *not* proposed — the mirror set is wider than trade's two, and the test is
cheaper and catches the same class.

### 3.7 atlas-saga-orchestrator

| Addition | File |
|---|---|
| `PetNameTagUse Type = "pet_name_tag_use"` | `libs/atlas-saga/model.go` (re-exported in the orchestrator's and channel's `saga/model.go`) |
| `RenamePet Action = "rename_pet"` + `RenamePetPayload{PetId, CharacterId, Name, PreviousName}` | `libs/atlas-saga/model.go` |
| payload unmarshal arm | `saga/model.go` (`case RenamePet`) — `unmarshal_completeness_test.go` enforces it |
| `EventKindPetNameChanged`, `RenamePet: {EventKindPetNameChanged}`, outcome `OutcomeSuccess` | `saga/event_acceptance.go` |
| `handleRenamePet` → `petP.RenameAndEmit(txn, payload…)` | `saga/handler.go` |
| `Rename`/`RenameAndEmit` + `RenameProvider` | `saga-orchestrator/pet/{processor,producer}.go` |
| `handleNameChangedEvent` → `AcceptEvent(txn, EventKindPetNameChanged)` | `kafka/consumer/pet/consumer.go` |
| `compensatePetNameTagUse` + `DispatchPetNameTagRollbacks` | `saga/compensator.go` |
| step timer registration | `saga/timer.go` |

Step order (FR-7.2), the inverse of `meso_sack_use`:

```
1. rename_pet             RenamePet    {petId, characterId, name, previousName}
2. consume_pet_name_tag   DestroyAsset {characterId, templateId: 5170000, quantity: 1}
```

**Compensation (FR-7.4).** `DispatchPetNameTagRollbacks` reverse-walks completed steps; for a
completed `RenamePet` it re-issues `RenameAndEmit` with `PreviousName`. `PreviousName` is
captured by atlas-channel at build time — it already read the pet to resolve the target, so
no new state and no extra round trip. `compensatePetNameTagUse` then does the
`TryTransition(Compensating → Failed)` / cancel-timer / `EmitSagaFailed` sequence copied from
`compensateMesoSackUse`.

Known, accepted limitation: if some *other* actor renames the pet between step 1 and the
compensation, the revert restores a stale name. A rename is player-initiated, serialized by
the client's exclusive-request gate, and the window is one Kafka round trip — the
alternative (a compare-and-swap revert keyed on the applied name) buys almost nothing for
real complexity. Documented in the compensator comment.

### 3.8 Tenant templates

`PetNameChanged` writer added to nine templates at the registry opcode, each with
`"fname": "CPet::OnNameChanged"` and `"services": ["channel"]`, inserted at the sorted
`opCode` position. Opcodes cross-checked against `docs/packets/audits/STATUS.md:233` this
session:

| v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms185 |
|---|---|---|---|---|---|---|---|---|---|
| n-a | 0x083 | 0x09D | 0x0A1 | 0x0AC | 0x0B0 | 0x0B9 | 0x0C8 | 0x0CB | 0x0B2 |

Re-confirm against the registry at implementation time regardless — a registry edit stales
the matrix (`bug_registry_fname_change_stales_packet_matrix`). The serverbound
`CharacterCashItemUseHandle` binding already exists everywhere and must not be duplicated.
Live-tenant socket-config reconciliation goes in `rollout.md`.

---

## 4. Version-matrix strategy

Applicable clientbound columns: **v61, v72, v79, v83, v84, v87, v92, v95, jms_v185** (nine).
`v48` is already blank/⬜ and is recorded `n-a`.

One thing the PRD did not flag: **the entire v92 column is `❌`** — every pet op (`MOVE_PET`,
`PET_CHAT`, `PET_EXCEPTION_LIST`, `PET_COMMAND`) and even `LOGIN_STATUS` sit at `❌` there,
so v92 has had no verification pass. Promoting `PET_NAMECHANGE` to `✅` on v92 is still
in scope and still achievable — the IDB is present (`GMS_v92_1_DEVM.exe.i64`) and its
`CPet::OnNameChanged` @`0x6967c0` was read this session — but it is a fresh derivation, not
a copy of a neighbouring verified cell. Plan for it as real work rather than a rounding
error.

Each cell follows `docs/packets/audits/VERIFYING_A_PACKET.md` unchanged: derive the read
order from that version's client, write the byte fixture with a `packet-audit:verify`
marker, pin the evidence record, regenerate the matrix. A round-trip fixture alone is not
evidence (`bug_matrix_roundtrip_fixture_false_verify`); the fixture must be justified by an
IDA address in its comment, the way `v61_test.go` does.

---

## 5. Alternatives considered

**A1 — Consume first, refund on rename failure (task-220's shape).** Rejected: FR-7.2 is
explicit, and the failure mode it prevents is real — a rejected name would cost a cash item
and generate exactly the support ticket §3 of the PRD wants to avoid. Rename-first costs one
extra compensator branch and nothing else.

**A2 — Skip the saga; emit `RENAME` and let atlas-consumables consume the item.** Simpler and
matches the kite/pet-consumable arms. Rejected: those arms consume unconditionally because
their effect cannot fail. This one can fail (invalid name reaching atlas-pets, pet deleted
mid-flight), and without the orchestrator there is no ordering guarantee and no rollback.

**A3 — Put the pet id on the wire.** Not available: the case-17 arm performs exactly one
`Encode*`, and `SetUtilDlgEx_Pet` (@`0x9acb27`) is not called from
`SendConsumeCashItemUseRequest` at all. Server-side resolution is forced, and OQ-4 shows it
is unambiguous.

**A4 — Silent `enableActions()` on rejection (OQ-2 option (a)).** Rejected now that a
concrete all-version message op is confirmed. A silent unlock reads to the player as "the
item did nothing", which is the very complaint that opened this task.

**A5 — Config-resolve the `nameTag` flag (a literal reading of FR-6.6).** Rejected. The value
is not a per-version wire *code* — those are what `atlas_packet.WithResolvedCode("operations", …)`
exists for. It is a render selector that must match a sibling codec's field. A shared named
constant with the `CLife::MakeNameTag` provenance in its comment satisfies DOM-25's actual
requirement (no bare unexplained literal) without inventing a tenant config key nobody will
ever tune.

---

## 6. Testing

**Unit — atlas-channel**
- `GetCashSlotItemType(t)(5170000) == 17`, and `5170001 → 0`. Must fail against the pre-fix
  predicate (FR-1.3) — assert that explicitly in the test name/comment.
- Handler arm, using the `cashItemInSlotFunc` / pet-lookup seams: no active pet → pink text +
  enableActions + no saga; name too short / too long / whitespace-only → same; name equal to
  current → same; happy path → exactly one saga with steps in the order
  `rename_pet, consume_pet_name_tag` and `PreviousName` populated.
- Names are asserted **at the handler layer**, not through a client (FR acceptance:
  "validation holds when the request bypasses the client filter").

**Unit — libs/atlas-packet**
- Byte fixtures per version per direction, with an IDA address in each comment. The JMS
  clientbound fixture must be one byte shorter than the GMS one — that asymmetry is the
  regression test for C-1.

**Unit — atlas-pets**
- Builder `SetName`; `updateName`; rename of a nonexistent pet; **re-applying the same name
  emits `NAME_CHANGED` again and does not error** (FR-5.5); atlas-pets rejects an invalid
  name even when atlas-channel would have passed it (FR-5.6).

**Unit — orchestrator**
- Payload unmarshal completeness (existing test picks it up automatically).
- `rename_pet` completes on `EventKindPetNameChanged` with a matching transaction id, and
  does *not* complete on a mismatched one.
- Forced `consume_pet_name_tag` failure → a second `RENAME` carrying `previousName`, exactly
  one `StatusEventTypeFailed`, timer cancelled, cache evicted — the
  `meso_sack_compensation_test.go` shape.

**Cross-service seam** — a green test on each side of a Kafka topic is not coverage
(`feedback_green_tests_miss_cross_service_seams`). The §3.6 round-trip fixture test is the
guard for the five-way mirror; the orchestrator's `AcceptEvent` test is the guard for the
event-kind wiring.

**Manual** — rename with an observer in the map; observer entering afterwards sees the new
name in the spawn body; relog; channel change; despawn/respawn (which also confirms the
`nameTag` value agrees between the two codecs).

---

## 7. Risks

| Risk | Mitigation |
|---|---|
| A version among v72/v79/v84/v87 diverges like JMS did. | Every cell is derived, never copied. The gate is already region-shaped, so adding a major boundary is a one-line change, not a redesign. |
| Five-way Kafka mirror drifts silently. | §3.6 round-trip fixture test. |
| Live tenants drop the new writer. | `rollout.md` reconciliation step (`bug_new_opcodes_not_in_live_tenant_config`). |
| v92 column needs a from-scratch derivation. | Called out in §4 and sized as real work. |
| `nameTag` value drifts between spawn and rename. | One shared constant feeds both codecs (§3.3). |
| Compensation reverts to a stale name under a concurrent rename. | Accepted; window is one round trip, documented in the compensator. |

---

## 8. Build & verification gates

Per `CLAUDE.md` §Build & Verification, all of: `go test -race`, `go vet`, `go build`,
`docker buildx bake` for every service whose `go.mod` moved (adding a `libs/atlas-constants`
symbol does not move one, but adding the constants package to a service that lacks it
would), `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check`,
`tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh`, and — because templates change
— `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh`,
`tools/template-movement-types-guard.sh`. Code review before PR.

Documentation updates: `docs/research/missing-features/items-and-consumables.md:40`,
`docs/research/missing-features/packet-gap-inference.md:469`, and a new `rollout.md`.

---

## 9. Remaining unknowns

Everything the PRD listed is closed. Two items are deliberately deferred to per-cell
implementation, where the playbook owns them, rather than guessed here:

1. The `CPet::OnNameChanged` body on **v72, v79, v84, v87** — four GMS reads agree on
   `str + byte`, but each cell must prove it.
2. Whether every version's client has a classification-517 arm in
   `SendConsumeCashItemUseRequest` at all. Where it does not, that version's *serverbound*
   cell is `n-a` and must pass the n-a consistency gate; the *clientbound* writer is still
   registered, since a GM or REST-driven rename must still render.
