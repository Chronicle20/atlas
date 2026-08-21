# Maple Life — In-Game Character Creation (`Cash/0543`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A player who uses a `Cash/0543` item in the game world completes `CUICharacterSaleDlg` end to end — name probe answered, character created through `atlas-character-factory`, result rendered, item consumed only on success.

**Architecture:** Classification-first dispatch arm in `atlas-channel`'s cash-item handler routes 543 to a new `maplelife` flow. An account-keyed in-memory pending store correlates the three dialog packets with each other and with the asynchronous `POST characters/seed` result, which arrives on `EVENT_TOPIC_SEED_STATUS` carrying a newly-added `transactionId`. Item consumption is a one-step channel saga fired only on `CREATED`.

**Tech Stack:** Go 1.x, `libs/atlas-packet` codecs, `libs/atlas-socket`, `libs/atlas-saga`, Kafka (`segmentio/kafka-go`), JSON:API REST via `libs/atlas-rest`, `packet-audit` coverage tooling, ida-pro-mcp for wire derivation.

**Spec:** `docs/tasks/task-246-maple-life-character-creation/design.md` (PRD at `prd.md`)

## Global Constraints

- **Never invent a wire value.** Every opcode, field order, width and result/error code shipped by this branch must trace to a `derivation.md` section with a per-version IDA address. Tasks 1–2 produce `derivation.md`; **no task from Task 3 onward may begin until Tasks 1 and 2 are complete and committed.**
- **In-scope versions: gms_v83, gms_v84, gms_v87, gms_v92, gms_v95 only.** `git diff` must show **no** change to `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_72_1.json`, `template_gms_79_1.json`, or `template_jms_185_1.json`.
- **Version divergence uses `MajorAtLeast`.** `t.IsRegion("GMS") && t.MajorAtLeast(N)` — never a raw `t.MajorVersion() >= N` and never a bare numeric literal in new code. (`libs/atlas-tenant/tenant.go:88,93`)
- **No bare cash-slot-type comparison for 543.** `it == CashSlotItemType(57)`, `(58)`, `(65)`, `(66)` are forbidden for this family; the arm branches on `item.GetClassification(itemId) == item.ClassificationCharacterCreation`.
- **`GetCashSlotItemType` is not modified.** Its ~40 sibling `MajorVersion() >= 95` branches are out of blast radius (PRD non-goal, task-227's boundary).
- **Result/error codes live in the tenant templates**, not in Go. Go declares named keys; `template_gms_{83,84,87,92,95}_1.json` carry the numbers, following `CashShopCheckNameChange`'s `options.operations` precedent (`template_gms_83_1.json:5072-5078`).
- **Account id and world id come from the session** (`s.AccountId()`, `s.WorldId()`), never from a client packet.
- **The item is consumed only on a confirmed `CREATED`.** Every other terminal outcome leaves it in inventory and writes a client-rendered error or logs an explained rejection. No silent fallthrough on an in-scope version.
- **`atlas-login` is not modified.** Not one file under `services/atlas-login/`.
- **The factory is the sole look validator** (design §5.3, confirmed at plan time). `atlas-channel` does not re-implement the eleven creation-template rules; it maps the factory's `400` onto a client-rendered error.
- **No stubs.** No `// TODO`, no placeholder comment, no unimplemented-status response lands on any in-scope version.
- Module roots: `libs/atlas-packet`, `libs/atlas-saga`, `libs/atlas-constants`, `services/atlas-channel/atlas.com/channel`, `services/atlas-character-factory/atlas.com/character-factory`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`. Implementers run `go build ./... && go test ./...` from the module root their task touches, nothing wider.

---

## Task 1: Derive the serverbound wire — cash-slot split, `SendCreateNewCharacter`, `USE_MAPLELIFE`

Answers PRD FR-1.1, FR-1.2, FR-1.3 and design Open Question 1 / Open Question 2. Read-only against the IDBs; the only repo write is `derivation.md`.

### Files

- `docs/tasks/task-246-maple-life-character-creation/derivation.md` — **new file**; create it with the §1–§3 sections this task fills (Task 2 appends §4–§6)
- `docs/packets/IMPLEMENTING_A_PACKET.md` — read-only; the derivation playbook
- `docs/reverse-engineering.md` — read-only; `func_query` / `idb_open` usage
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — (1457-1469) read-only; Atlas's current 543 branch, the thing being checked against the client
- `libs/atlas-packet/cash/serverbound/item_use.go` — read-only; `UpdateTimeFirst` and the common `ItemUse` prefix every sub-body sits behind

IDA sessions (`mcp__ida-pro__idb_list`): `gms_v84` = `46c2a2eb`, `gms_v87` = `c0829805`, `gms_v92` = `019cd393`, `gms_v95` = `ecc757f4`. **`gms_v83` is discovered but not adopted** (`MapleStory_dump.exe.i64`, pid 10044, port 13339) — adopt it with `mcp__ida-pro__idb_open` before use.

- [ ] **Step 1: Adopt the v83 IDB and confirm all five sessions**

Run `mcp__ida-pro__idb_list`. Adopt `E:\Programs\Nexon\IDBs_v9\GMS\v83_Me\MapleStory_dump.exe.i64` via `idb_open` if its `session_id` is empty. Do not proceed until all five in-scope sessions report `is_analyzing: false`.

- [ ] **Step 2: Derive `get_cashslot_item_type`'s 543 branch on v83 and v95**

Locate the function (`mcp__ida-pro__lookup_funcs` / `find_regex` for `get_cashslot_item_type`) and decompile its classification-543 arm on **both** `gms_v83` and `gms_v95`.

Record in `derivation.md` §1, one row per version:

| version | function address | comparison as compiled | signedness | 543 → type values |
|---|---|---|---|---|
| gms_v83 | *(fill)* | *(fill, e.g. `if (itemId/1000 - 5431 > 1)`)* | signed / unsigned | *(fill)* |
| gms_v95 | *(fill)* | *(fill)* | signed / unsigned | *(fill)* |

Then state explicitly, in prose, the answer to **OQ-2**: given `Cash/0543` ships only 5430xxx / 5431xxx / 5432xxx, is the 57/58 branch reachable? If the compiled comparison is **unsigned**, 5430xxx reaches it and a second sub-body shape may exist; if **signed**, it does not and §1 must say "57/58 unreachable with shipped data — no arm written."

- [ ] **Step 3: Derive `CUICharacterSaleDlg::SendCreateNewCharacter` on all five versions**

Decompile the sender on gms_v83/84/87/92/95. For each, record in `derivation.md` §2:

- the function address
- the `decompile_sha256` (needed verbatim by the evidence records in Tasks 4–6)
- the **complete** encode order as emitted, one line per field: `Encode4 nFace`, `EncodeStr sName`, etc., including the leading cash-slot-type byte and the `nPOS`/`nItemID` the common `ItemUse` prefix already consumes
- whether `update_time` leads or trails, cross-checked against `UpdateTimeFirst` at lines 22-24 of `libs/atlas-packet/cash/serverbound/item_use.go` — GMS ≥ v87 and JMS lead; v83/v84 trail
- any field whose width or presence differs from another version, with the boundary named as a `MajorAtLeast(N)` predicate

Explicitly state whether the layout matches `charsb.CreateCharacter` (the login-socket creation packet). FR-1.1 forbids assuming it does; the answer must be derived either way.

- [ ] **Step 4: Derive `USE_MAPLELIFE` (gms_v95, opcode 303) and settle OQ-1**

`docs/packets/registry/gms_v95.yaml:4038-4042` has `USE_MAPLELIFE` serverbound at opcode 303 with an empty `fname`. Find its sender on `gms_v95` (search the dispatch and the `CUICharacterSaleDlg` methods for a send at that opcode). Record in `derivation.md` §3:

- the sender's address and full encode order, or a positive finding that **no** v95 code path sends opcode 303
- the answer to **OQ-1** stated as exactly one of: *v95 sends only 303*, *v95 sends only the `USE_CASH_ITEM` 543 sub-body*, *v95 sends both in sequence*, with the address evidence for the claim

- [ ] **Step 5: Record the three 543 item ids' `Item.wz` spec differences (OQ-3)**

Read the `Cash/0543` entries in the repo's WZ data for 5430xxx / 5431xxx / 5432xxx and record what distinguishes them in `derivation.md` §1.4. This gates nothing (design §10) but must be on the record.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-246-maple-life-character-creation/derivation.md
git commit -m "docs(task-246): derive serverbound Maple Life wire (cash-slot split, SendCreateNewCharacter, USE_MAPLELIFE)"
```

---

## Task 2: Derive the clientbound wire and the duplicate-probe opcode

Answers PRD FR-1.4, FR-3.1 and design C1 / §7. Read-only against the IDBs; appends to `derivation.md`.

### Files

- `docs/tasks/task-246-maple-life-character-creation/derivation.md` — append §4–§6 (created by Task 1)
- `docs/packets/registry/gms_v83.yaml` — (1805-1814) read-only; `MAPLELIFE_RESULT` 349 / `MAPLELIFE_ERROR` 350
- `docs/packets/registry/gms_v84.yaml` — (2381-2392) read-only; 349 / 350
- `docs/packets/registry/gms_v87.yaml` — (1908-1917,3651-3655) read-only; 370 / 371, and `JMS_SLASH_COMMAND` 270
- `docs/packets/registry/gms_v92.yaml` — (2102-2113) read-only; 404 / 405
- `docs/packets/registry/gms_v95.yaml` — (2118-2127,4100-4104) read-only; 413 / 414, and `JMS_SLASH_COMMAND` 311
- `docs/packets/registry/jms_v185.yaml` — (3640-3644) read-only; `JMS_SLASH_COMMAND` 271

Same five IDA sessions as Task 1, plus `jms_v185` = `a977912e`.

- [ ] **Step 1: Derive `CUICharacterSaleDlg::OnCheckDuplicatedIDResult` on all five versions**

Record in `derivation.md` §4, one block per version (gms_v83/84/87/92/95):

- receiver address and `decompile_sha256`
- the full decode order, one line per field with its width and signedness — e.g. `DecodeStr sName`, `Decode1 nResult (SIGNED)`
- the **complete** branch enumeration the dialog renders, one row per arm: the comparison as compiled, the semantic meaning, and the string or UI action it drives

Follow the precedent set for the sibling op in `libs/atlas-packet/cash/clientbound/check_name_change.go:22-53`, where the three-way signed branch on `nResult` is written out arm by arm with a per-version address.

Name any version whose layout or arm set differs from v83, and state the boundary as a `MajorAtLeast(N)` predicate.

- [ ] **Step 2: Derive `CUICharacterSaleDlg::OnCreateNewCharacterResult` on all five versions**

Same output shape in `derivation.md` §5. The **full error-code enumeration** is the deliverable — every arm the client can render, including the success arm (design §5.4's `MAPLELIFE_ERROR{success code}`) and whatever arms correspond to duplicate-name-at-submit, invalid look, and generic failure. Tasks 5 and 7 consume this list verbatim as the writer's `options` keys.

- [ ] **Step 3: Derive the duplicate-probe sender per version, and settle C1**

`docs/packets/audits/status.json` carries `CUICharacterSaleDlg::SendCheckDuplicateIDPacket` on the row named `JMS_SLASH_COMMAND`: gms_v87 = 270, gms_v95 = 311, jms_v185 = 271, and `-1` on gms_v83/84/92. Decompile the sender on each of the five in-scope versions and record in `derivation.md` §6:

| version | sender address | opcode emitted | body encode order | collides with `CHECK_CHAR_NAME` (21)? |
|---|---|---|---|---|
| gms_v83 | *(fill)* | *(fill)* | *(fill)* | yes / no |
| gms_v84 | *(fill)* | *(fill)* | *(fill)* | yes / no |
| gms_v87 | *(fill — expect 270)* | *(fill)* | *(fill)* | *(fill)* |
| gms_v92 | *(fill)* | *(fill)* | *(fill)* | yes / no |
| gms_v95 | *(fill — expect 311)* | *(fill)* | *(fill)* | *(fill)* |

For v83/84/92 the registry has no row at all. Either find the opcode, or record a positive finding — with the address of the dialog code that shows it — that the dialog uses no separate probe on that version.

State the **routing consequence** explicitly, as exactly one of:

- *(A)* every in-scope version has a Maple-Life-specific probe opcode → Task 12 writes a standalone handler; the pending store is not needed for disambiguation
- *(B)* one or more versions reuse `CHECK_CHAR_NAME` (21), which `template_gms_83_1.json:161-169` already binds to `CashShopCheckNameChangeHandle` for the `channel` service → Task 12 branches **inside** `CashShopCheckNameChangeHandleFunc` on the live pending record

- [ ] **Step 4: Identify jms_v185 opcode 271 and decide split vs rename**

Decompile jms_v185's opcode-271 sender. Record in `derivation.md` §6.3 which it is, and the resulting decision:

- if it genuinely is a JMS slash command → **split** the registry row in Task 3: `JMS_SLASH_COMMAND` keeps jms_v185 271 and is `n-a` on GMS; a new `MAPLELIFE_CHECK_NAME` serverbound row takes the GMS columns
- if it is jms's own sale-dialog probe → the row is misnamed and Task 3 **renames** it. PRD scope says jms stays untouched; if this finding contradicts that, **stop and escalate to the user** rather than widening scope silently

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-246-maple-life-character-creation/derivation.md
git commit -m "docs(task-246): derive Maple Life clientbound bodies, code enumerations, and duplicate-probe opcodes"
```

---

## Task 3: Registry hygiene — resolve the `JMS_SLASH_COMMAND` row

Design §7. Must land before any template routing (Task 7), because the coverage matrix and operation registry are what `packet-audit` checks the templates against.

### Files

- `docs/packets/registry/gms_v83.yaml` — add the `MAPLELIFE_CHECK_NAME` serverbound row if §6 found one
- `docs/packets/registry/gms_v84.yaml` — same
- `docs/packets/registry/gms_v87.yaml` — (3651-3655) split or rename per §6.3
- `docs/packets/registry/gms_v92.yaml` — add the row if §6 found one
- `docs/packets/registry/gms_v95.yaml` — (4100-4104) split or rename per §6.3
- `docs/packets/registry/jms_v185.yaml` — (3640-3644) keep or rename per §6.3
- `docs/tasks/task-246-maple-life-character-creation/derivation.md` — **new file** at plan time, produced by Task 1; read-only here. §6 is the source for every value written.

Module root for the tooling: `tools/packet-audit`.

- [ ] **Step 1: Apply the §6.3 decision to the six registry files**

Follow `derivation.md` §6 exactly. Every `opcode:` written here comes from a §6 table row with an address; a version §6 recorded as having no probe gets **no row**, not a guessed one. Set `provenance:` to reflect that these are IDA-derived, not `csv-import`, and add a `note:` citing `docs/tasks/task-246-maple-life-character-creation/derivation.md §6`.

- [ ] **Step 2: Regenerate the matrix and run the registry checks**

```bash
cd tools/packet-audit && go run . matrix
cd tools/packet-audit && go run . operations --check
cd tools/packet-audit && go run . fname-doc --check
```

Expected: exit 0 on all three. The renamed/split row must appear in `docs/packets/audits/STATUS.md` with the fname `CUICharacterSaleDlg::SendCheckDuplicateIDPacket` attached to the GMS columns.

- [ ] **Step 3: Verify no other op's cell changed state**

```bash
git diff --stat docs/packets/audits/
git diff docs/packets/audits/status.json | grep -E '^[-+].*"state"' | head -50
```

Expected: only the rows this task touched appear. Any other op flipping state is a defect — stop and diagnose.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/registry docs/packets/audits
git commit -m "fix(packets): resolve the JMS_SLASH_COMMAND row for CUICharacterSaleDlg::SendCheckDuplicateIDPacket"
```

---

## Task 4: `MAPLELIFE_RESULT` clientbound codec

### Files

- `libs/atlas-packet/maplelife/clientbound/result.go` — **new file**; the codec, its writer-name constant, its `options` key constants, and its body builders
- `libs/atlas-packet/maplelife/clientbound/result_test.go` — **new file**; byte fixtures with `packet-audit:verify` markers, round-trip, code-resolution and reason-mapping tests
- `docs/packets/evidence/gms_v83/maplelife.clientbound.MapleLifeResult.yaml` — **new file**
- `docs/packets/evidence/gms_v84/maplelife.clientbound.MapleLifeResult.yaml` — **new file**
- `docs/packets/evidence/gms_v87/maplelife.clientbound.MapleLifeResult.yaml` — **new file**
- `docs/packets/evidence/gms_v92/maplelife.clientbound.MapleLifeResult.yaml` — **new file**
- `docs/packets/evidence/gms_v95/maplelife.clientbound.MapleLifeResult.yaml` — **new file**
- `docs/tasks/task-246-maple-life-character-creation/derivation.md` — **new file** at plan time, produced by Tasks 1-2; read-only here. §4 is the sole source of field order, widths, arms and addresses.

Patterns to copy: `libs/atlas-packet/cash/clientbound/check_name_change.go` (immutable struct, `WithResolvedCode` body builders, reason→arm table as data, per-version address comment block); `libs/atlas-packet/cash/clientbound/check_name_change_test.go:1-56` (marker block + `pt.Variants` round-trip); `docs/packets/evidence/gms_v83/cash.clientbound.CashCheckNameChange.yaml` (evidence shape).

Module root: `libs/atlas-packet`.

**Interfaces:**
- Produces: `clientbound.MapleLifeResultWriter` (string constant `"MapleLifeResult"`); one exported `string` constant per arm derivation.md §4 enumerated, to be used as `options.operations` keys; `clientbound.MapleLifeResultBody(name string, key string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`; `clientbound.MapleLifeResultRejectedBody(name string, reason string) ...` mapping `character.NameReason*` values onto arm keys; `mapleLifeResultReasonArms map[string]string` as an unexported package var so a test can assert coverage.
- Consumed by: Task 7 (template `options`), Task 12 (the check-name handler).

- [ ] **Step 1: Write the failing tests**

`result_test.go` — table-driven over `pt.Variants`, with the marker block immediately above the round-trip test. Imports, `t.Run` wrappers and null-logger setup copied from `libs/atlas-packet/cash/clientbound/check_name_change_test.go:1-12,40-56`.

Marker block — one line per in-scope version, `ida=` taken from the §4 receiver address for that version:

```go
// packet-audit:verify packet=maplelife/clientbound/MapleLifeResult version=gms_v83 ida=<derivation.md §4 gms_v83 address>
// packet-audit:verify packet=maplelife/clientbound/MapleLifeResult version=gms_v84 ida=<derivation.md §4 gms_v84 address>
// packet-audit:verify packet=maplelife/clientbound/MapleLifeResult version=gms_v87 ida=<derivation.md §4 gms_v87 address>
// packet-audit:verify packet=maplelife/clientbound/MapleLifeResult version=gms_v92 ida=<derivation.md §4 gms_v92 address>
// packet-audit:verify packet=maplelife/clientbound/MapleLifeResult version=gms_v95 ida=<derivation.md §4 gms_v95 address>
```

Required test functions and what each asserts:

| test | asserts |
|---|---|
| `TestMapleLifeResultByteFixture` | per version, `pt.Encode` of a fixed input produces the exact byte slice §4's decode order requires — the expected `[]byte` is written out literally, derived field by field from §4, never computed by re-running the encoder |
| `TestMapleLifeResultRoundTrip` | over `pt.Variants`, `pt.RoundTrip` leaves zero unconsumed bytes and every accessor round-trips |
| `TestMapleLifeResultVersionDivergence` | for each boundary §4 names, the two straddling variants encode differently; if §4 names none, the test instead asserts all five in-scope variants encode **byte-identically** and is named `TestMapleLifeResultNoVersionDivergence` |
| `TestMapleLifeResultCodesAreConfigResolved` | every arm constant resolves through `options["operations"]`; encoding with an empty options map does **not** silently produce a valid-looking arm |
| `TestMapleLifeResultReasonMapping` | `mapleLifeResultReasonArms` has an entry for each of `character.NameReasonLength`, `NameReasonRegex`, `NameReasonDuplicate`, `NameReasonReserved` (asserted as the four literal strings `"length"`, `"regex"`, `"duplicate"`, `"reserved"` — `libs/atlas-packet` must not import an `atlas-channel` package), and that an unrecognised reason maps to the generic-failure arm |
| `TestMapleLifeResultOperation` | `Operation()` returns `MapleLifeResultWriter`, and that constant equals `"MapleLifeResult"` |

Byte-fixture inputs: name `"Chronicle"`, and one case per arm §4 enumerated.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-packet && go test ./maplelife/... -run TestMapleLifeResult -v
```

Expected: build failure — package `maplelife/clientbound` does not exist.

- [ ] **Step 3: Write the codec**

`result.go` — immutable struct with unexported fields, value-receiver accessors, `Operation()`, `String()`, `Encode` and `Decode`, matching the shape of `check_name_change.go`'s `CheckNameChange`. Fields, widths and order come from `derivation.md` §4. Version gates, if §4 names any, use `t := tenant.MustFromContext(ctx)` plus `t.IsRegion("GMS") && t.MajorAtLeast(N)`.

The doc comment carries, as `check_name_change.go:57-135` does: the fname, the per-version receiver addresses, the full arm enumeration with the comparison as compiled, a `packet-audit:fname CUICharacterSaleDlg::OnCheckDuplicatedIDResult` line, and a `Derivation:` line pointing at `docs/tasks/task-246-maple-life-character-creation/derivation.md §4`.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd libs/atlas-packet && go build ./... && go test ./maplelife/... -v
```

Expected: PASS.

- [ ] **Step 5: Write the five evidence records**

One per version, shape copied from `docs/packets/evidence/gms_v83/cash.clientbound.CashCheckNameChange.yaml`:

```yaml
packet: maplelife/clientbound/MapleLifeResult
direction: clientbound
version: gms_v83
category: TIER1-FIXTURE
ida:
    function: CUICharacterSaleDlg::OnCheckDuplicatedIDResult
    address: "<derivation.md §4 address>"
    decompile_sha256: <derivation.md §4 sha256>
verifies:
    - libs/atlas-packet/maplelife/clientbound/result_test.go#TestMapleLifeResultByteFixture
    - libs/atlas-packet/maplelife/clientbound/result_test.go#TestMapleLifeResultRoundTrip
    - libs/atlas-packet/maplelife/clientbound/result_test.go#TestMapleLifeResultCodesAreConfigResolved
    - libs/atlas-packet/maplelife/clientbound/result_test.go#TestMapleLifeResultReasonMapping
```

- [ ] **Step 6: Regenerate the matrix and confirm the five cells promote**

```bash
cd tools/packet-audit && go run . matrix
```

Then open `docs/packets/audits/STATUS.md` and locate the `MapleLifeResult` row (it does not exist until this task lands, which is why this is a read of the regenerated file rather than a pre-runnable grep).

Expected: the row shows ✅ for gms_v83, v84, v87, v92, v95. A cell that does not promote is a failure — do not claim otherwise.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/maplelife docs/packets/evidence docs/packets/audits
git commit -m "feat(packets): add MAPLELIFE_RESULT clientbound codec, verified gms_v83-v95"
```

---

## Task 5: `MAPLELIFE_ERROR` clientbound codec

Structurally identical to Task 4 for the second clientbound op. Written out in full rather than by reference — the implementer may be reading tasks out of order.

### Files

- `libs/atlas-packet/maplelife/clientbound/error.go` — **new file**
- `libs/atlas-packet/maplelife/clientbound/error_test.go` — **new file**
- `docs/packets/evidence/gms_v83/maplelife.clientbound.MapleLifeError.yaml` — **new file**
- `docs/packets/evidence/gms_v84/maplelife.clientbound.MapleLifeError.yaml` — **new file**
- `docs/packets/evidence/gms_v87/maplelife.clientbound.MapleLifeError.yaml` — **new file**
- `docs/packets/evidence/gms_v92/maplelife.clientbound.MapleLifeError.yaml` — **new file**
- `docs/packets/evidence/gms_v95/maplelife.clientbound.MapleLifeError.yaml` — **new file**
- `docs/tasks/task-246-maple-life-character-creation/derivation.md` — **new file** at plan time, produced by Tasks 1-2; read-only here. §5 is the sole source.

Patterns to copy: `libs/atlas-packet/cash/clientbound/check_name_change.go`; `libs/atlas-packet/cash/clientbound/check_name_change_test.go:1-56`.

Module root: `libs/atlas-packet`.

**Interfaces:**
- Produces: `clientbound.MapleLifeErrorWriter` (string constant `"MapleLifeError"`); one exported `string` constant per arm §5 enumerated — **including the success arm**, since design §5.4 renders success through this op; `clientbound.MapleLifeErrorBody(key string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`.
- Consumed by: Task 7 (template `options`), Tasks 13 and 14 (submit handler and seed consumer).

- [ ] **Step 1: Write the failing tests**

`error_test.go`, table-driven over `pt.Variants`, setup copied from `libs/atlas-packet/cash/clientbound/check_name_change_test.go:1-12,40-56`. Marker block above the round-trip test, one line per in-scope version with `ida=` from §5:

```go
// packet-audit:verify packet=maplelife/clientbound/MapleLifeError version=gms_v83 ida=<derivation.md §5 gms_v83 address>
// packet-audit:verify packet=maplelife/clientbound/MapleLifeError version=gms_v84 ida=<derivation.md §5 gms_v84 address>
// packet-audit:verify packet=maplelife/clientbound/MapleLifeError version=gms_v87 ida=<derivation.md §5 gms_v87 address>
// packet-audit:verify packet=maplelife/clientbound/MapleLifeError version=gms_v92 ida=<derivation.md §5 gms_v92 address>
// packet-audit:verify packet=maplelife/clientbound/MapleLifeError version=gms_v95 ida=<derivation.md §5 gms_v95 address>
```

| test | asserts |
|---|---|
| `TestMapleLifeErrorByteFixture` | one case per arm §5 enumerated; the expected `[]byte` written out literally, derived field by field from §5 |
| `TestMapleLifeErrorRoundTrip` | over `pt.Variants`, `pt.RoundTrip` leaves zero unconsumed bytes; every accessor round-trips |
| `TestMapleLifeErrorVersionDivergence` | for each boundary §5 names, the two straddling variants encode differently; if §5 names none, assert all five encode byte-identically and name it `TestMapleLifeErrorNoVersionDivergence` |
| `TestMapleLifeErrorCodesAreConfigResolved` | every arm constant resolves through `options["operations"]` |
| `TestMapleLifeErrorArmsAreExhaustive` | the exported arm constants are exactly the set §5 enumerated — the test lists them literally so adding a code without deriving it fails here |
| `TestMapleLifeErrorOperation` | `Operation()` returns `MapleLifeErrorWriter`, and that constant equals `"MapleLifeError"` |

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-packet && go test ./maplelife/... -run TestMapleLifeError -v
```

Expected: FAIL — `MapleLifeError` undefined.

- [ ] **Step 3: Write the codec**

`error.go` — immutable struct, value-receiver accessors, `Operation()`, `String()`, `Encode`, `Decode`. Fields, widths and order from `derivation.md` §5. Version gates via `t.IsRegion("GMS") && t.MajorAtLeast(N)` only. Doc comment carries the fname, the five receiver addresses, the full arm enumeration, `packet-audit:fname CUICharacterSaleDlg::OnCreateNewCharacterResult`, and a `Derivation:` pointer to §5.

Note in the doc comment that the **success** arm ships through this op, not through `MAPLELIFE_RESULT` — design §5.4, so a reader does not mistake the writer name for a failure-only channel.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd libs/atlas-packet && go build ./... && go test ./maplelife/... -v
```

Expected: PASS.

- [ ] **Step 5: Write the five evidence records**

One per version, shape copied from `docs/packets/evidence/gms_v83/cash.clientbound.CashCheckNameChange.yaml`, with `packet: maplelife/clientbound/MapleLifeError`, `function: CUICharacterSaleDlg::OnCreateNewCharacterResult`, and `address`/`decompile_sha256` from §5.

- [ ] **Step 6: Regenerate the matrix and confirm the five cells promote**

```bash
cd tools/packet-audit && go run . matrix
```

Then open `docs/packets/audits/STATUS.md` and locate the `MapleLifeError` row.

Expected: ✅ for gms_v83, v84, v87, v92, v95.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/maplelife docs/packets/evidence docs/packets/audits
git commit -m "feat(packets): add MAPLELIFE_ERROR clientbound codec, verified gms_v83-v95"
```

---

## Task 6: Serverbound codecs — 543 sub-body, `USE_MAPLELIFE`, duplicate probe

### Files

- `libs/atlas-packet/cash/serverbound/item_use_maple_life.go` — **new file**; the `USE_CASH_ITEM` 543 sub-body, sitting with its `item_use_*.go` siblings
- `libs/atlas-packet/cash/serverbound/item_use_maple_life_test.go` — **new file**
- `libs/atlas-packet/maplelife/serverbound/use.go` — **new file**; `USE_MAPLELIFE` (gms_v95, opcode 303) — **write only if `derivation.md` §3 found a sender**
- `libs/atlas-packet/maplelife/serverbound/check_name.go` — **new file**; the duplicate probe — **write only if §6 found a Maple-Life-specific opcode on at least one in-scope version**
- `libs/atlas-packet/maplelife/serverbound/serverbound_test.go` — **new file**; covers whichever of the two above were written
- `docs/tasks/task-246-maple-life-character-creation/derivation.md` — **new file** at plan time, produced by Tasks 1-2; read-only here. §2, §3 and §6 are the sources.

Patterns to copy: `libs/atlas-packet/cash/serverbound/item_use_incubator.go` (sub-body with the `updateTimeFirst` trailing-`update_time` convention); `libs/atlas-packet/cash/serverbound/item_use.go:14-24` (`UpdateTimeFirst`); `libs/atlas-packet/cash/serverbound/check_name_change.go` (standalone serverbound op shape).

Evidence records for whichever ops this task lands go in `docs/packets/evidence/gms_v83/`, `gms_v84/`, `gms_v87/`, `gms_v92/`, `gms_v95/` — one per op × applicable version, same shape as `docs/packets/evidence/gms_v83/cash.clientbound.CashCheckNameChange.yaml`.

Module root: `libs/atlas-packet`.

**Interfaces:**
- Produces: `serverbound.NewItemUseMapleLife(updateTimeFirst bool) *ItemUseMapleLife` with a value-receiver accessor per field §2 enumerated plus `UpdateTime() uint32`; if §3 found a v95 sender, `maplelife/serverbound.Use` with `MapleLifeUseHandle` (string constant `"MapleLifeUseHandle"`); if §6 found a probe opcode, `maplelife/serverbound.CheckName` with `MapleLifeCheckNameHandle` (string constant `"MapleLifeCheckNameHandle"`) and a `Name() string` accessor.
- Consumed by: Task 7 (template routing), Tasks 11–13 (handlers).

- [ ] **Step 1: Write the failing tests**

`item_use_maple_life_test.go` — over `pt.Variants` restricted to the five in-scope GMS variants, plus the marker block, `ida=` from §2:

```go
// packet-audit:verify packet=cash/serverbound/ItemUseMapleLife version=gms_v83 ida=<derivation.md §2 gms_v83 address>
// packet-audit:verify packet=cash/serverbound/ItemUseMapleLife version=gms_v84 ida=<derivation.md §2 gms_v84 address>
// packet-audit:verify packet=cash/serverbound/ItemUseMapleLife version=gms_v87 ida=<derivation.md §2 gms_v87 address>
// packet-audit:verify packet=cash/serverbound/ItemUseMapleLife version=gms_v92 ida=<derivation.md §2 gms_v92 address>
// packet-audit:verify packet=cash/serverbound/ItemUseMapleLife version=gms_v95 ida=<derivation.md §2 gms_v95 address>
```

| test | asserts |
|---|---|
| `TestItemUseMapleLifeByteFixture` | per version, decoding the literal byte slice §2's order requires yields the expected field values, each written out literally |
| `TestItemUseMapleLifeRoundTrip` | `pt.RoundTrip` leaves zero unconsumed bytes on every in-scope variant |
| `TestItemUseMapleLifeUpdateTimeTrailing` | with `updateTimeFirst=false` (gms_v83/v84 per `libs/atlas-packet/cash/serverbound/item_use.go:22-24`) the sub-body consumes the trailing `update_time` and `UpdateTime()` returns it; with `updateTimeFirst=true` (gms_v87/92/95) it does not, and `UpdateTime()` returns 0 — mirrors `libs/atlas-packet/cash/serverbound/item_use_incubator.go:42-52` |
| `TestItemUseMapleLifeFieldOrder` | field-by-field, the decoded values match a fixture whose bytes are laid out in §2's exact order, so a transposition fails |

If §3 found a v95 `USE_MAPLELIFE` sender, add `TestMapleLifeUseRoundTrip` and `TestMapleLifeUseByteFixture` in `serverbound_test.go`, gms_v95 only, with a single `packet-audit:verify packet=maplelife/serverbound/MapleLifeUse version=gms_v95 ida=<§3 address>` marker.

If §6 found a probe opcode, add `TestMapleLifeCheckNameRoundTrip` and `TestMapleLifeCheckNameByteFixture` with one marker per version §6 gave an address for. Fixture name: `"Chronicle"`.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-packet && go test ./cash/serverbound/... -run TestItemUseMapleLife -v
```

Expected: FAIL — `ItemUseMapleLife` undefined.

- [ ] **Step 3: Write the codecs**

`item_use_maple_life.go`: immutable struct with the `updateTimeFirst bool` construction parameter, `Encode`, `Decode`, and the `if !m.updateTimeFirst { … updateTime }` tail exactly as `item_use_incubator.go:42-52` does it. Field set and order from `derivation.md` §2.

If §1's OQ-2 answer says the 57/58 branch is reachable and carries a **different** body shape, write the second shape as a distinct struct in the same file with its own tests and markers, and say in the doc comment which item-id range reaches it. If §1 says 57/58 is unreachable with shipped data, write **one** shape and record that finding in the doc comment — do not write an arm on a guess.

`use.go` / `check_name.go` only if §3 / §6 respectively found a sender. Each carries a `packet-audit:fname` line and a `Derivation:` pointer.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd libs/atlas-packet && go build ./... && go test ./cash/... ./maplelife/... -v
```

Expected: PASS.

- [ ] **Step 5: Write the evidence records and regenerate the matrix**

One record per op × version this task landed, shape per `docs/packets/evidence/gms_v83/cash.clientbound.CashCheckNameChange.yaml`. Then:

```bash
cd tools/packet-audit && go run . matrix
cd tools/packet-audit && go run . gate-check
```

Expected: exit 0; the serverbound rows promote for every version they have a marker on. If a new `MajorAtLeast` boundary was introduced in `item_use_maple_life.go`, add its row to `docs/packets/gates.yaml` following the schema documented at `docs/packets/gates.yaml:26-34` before re-running `gate-check`.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet docs/packets
git commit -m "feat(packets): add Maple Life serverbound codecs (543 sub-body, USE_MAPLELIFE, duplicate probe)"
```

---

## Task 7: Route the handlers and writers in the five in-scope templates

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` — add writer + handler entries
- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json` — same
- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` — same
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — same
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — same
- `docs/tasks/task-246-maple-life-character-creation/derivation.md` — **new file** at plan time, produced by Tasks 1-2; read-only here. §4/§5 supply the `options` code values, §6 the serverbound opcodes.

Patterns to copy: `template_gms_83_1.json:5068-5082` (writer entry with `options.operations` and `"services": ["channel"]`); `template_gms_83_1.json:161-169` (handler entry with `"validator": "LoggedInValidator"`).

**Do not open** `template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_72_1.json`, `template_gms_79_1.json`, or `template_jms_185_1.json`.

**Interfaces:**
- Consumes: `MapleLifeResultWriter` / `MapleLifeErrorWriter` (Tasks 4–5); `MapleLifeUseHandle` / `MapleLifeCheckNameHandle` (Task 6).
- Produces: the runtime `options["operations"]` tables the two writers resolve their codes through.

- [ ] **Step 1: Add the two writer entries per template**

Per version, using the opcode from that version's registry file and the code values from `derivation.md` §4/§5:

```json
{
  "opCode": "<MAPLELIFE_RESULT opcode for this version, hex>",
  "writer": "MapleLifeResult",
  "fname": "CUICharacterSaleDlg::OnCheckDuplicatedIDResult",
  "options": { "operations": { "<arm key>": <value>, ... } },
  "services": ["channel"]
},
{
  "opCode": "<MAPLELIFE_ERROR opcode for this version, hex>",
  "writer": "MapleLifeError",
  "fname": "CUICharacterSaleDlg::OnCreateNewCharacterResult",
  "options": { "operations": { "<arm key>": <value>, ... } },
  "services": ["channel"]
}
```

Registry opcodes to convert to hex — re-read them from the registry files rather than transcribing from here: `MAPLELIFE_RESULT` v83 349, v84 349, v87 370, v92 404, v95 413; `MAPLELIFE_ERROR` v83 350, v84 350, v87 371, v92 405, v95 414.

**Every** arm key Task 4 and Task 5 exported must appear in **every** in-scope template. A missing key resolves to `ResolveCode`'s loud 99 sentinel, which is not a safe default — `check_name_change.go:17-24` documents exactly that trap.

- [ ] **Step 2: Add the handler entries per template**

If `derivation.md` §6 found a Maple-Life-specific probe opcode on a version, add for that version:

```json
{
  "opCode": "<probe opcode, hex>",
  "validator": "LoggedInValidator",
  "handler": "MapleLifeCheckNameHandle",
  "fname": "CUICharacterSaleDlg::SendCheckDuplicateIDPacket",
  "services": ["channel"]
}
```

On a version §6 recorded as reusing `CHECK_CHAR_NAME` (21), add **nothing** — the existing `CashShopCheckNameChangeHandle` entry stays, and Task 12 branches inside that handler.

If §3 found a v95 `USE_MAPLELIFE` sender, add to `template_gms_95_1.json` only:

```json
{
  "opCode": "0x12F",
  "validator": "LoggedInValidator",
  "handler": "MapleLifeUseHandle",
  "fname": "<the sender fname derivation.md §3 recorded>",
  "services": ["channel"]
}
```

(303 decimal = `0x12F`; confirm against `docs/packets/registry/gms_v95.yaml:4038-4042` before writing.)

- [ ] **Step 3: Validate the templates**

```bash
cd tools/packet-audit && go run . template --check
cd tools/packet-audit && go run . operations --check
```

Expected: exit 0 on both.

- [ ] **Step 4: Confirm the out-of-scope templates are untouched**

```bash
git status --porcelain services/atlas-configurations/seed-data/templates/
```

Expected: exactly the five in-scope `template_gms_{83,84,87,92,95}_1.json` files, and nothing else.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates
git commit -m "feat(configurations): route Maple Life handlers and writers on gms_v83-v95"
```

---

## Task 8: The pending-dialog store

Design §4.2. Pure in-memory state, no I/O, independently testable.

### Files

- `services/atlas-channel/atlas.com/channel/maplelife/registry.go` — **new file**; the account-keyed store
- `services/atlas-channel/atlas.com/channel/maplelife/registry_test.go` — **new file**

Patterns to copy: `services/atlas-channel/atlas.com/channel/remotemerchant/registry.go` — the whole file. Same `sync.RWMutex` + singleton `once`, same tenant-scoped `Sweep` (its doc comment at `:100-114` explains why an unscoped sweep is destructive on a multi-tenant pod), same `Put` / `Take` / `ClearCharacter` shape.

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces:**
- Produces:
  ```go
  type Phase string
  const (PhaseOpen Phase = "OPEN"; PhaseSubmitted Phase = "SUBMITTED")
  type Key struct { Tenant tenant.Model; AccountId uint32 }
  type Entry struct {
      CharacterId   uint32
      WorldId       world.Id
      ItemId        item.Id
      Slot          slot.Position
      UpdateTime    uint32
      Phase         Phase
      TransactionId string
      CandidateName string
      At            time.Time
  }
  type Expired struct { Tenant tenant.Model; AccountId uint32; Entry Entry }
  func GetRegistry() *Registry
  func (r *Registry) Put(t tenant.Model, accountId uint32, e Entry)
  func (r *Registry) Get(t tenant.Model, accountId uint32) (Entry, bool)
  func (r *Registry) Take(t tenant.Model, accountId uint32) (Entry, bool)
  func (r *Registry) TakeByTransactionId(t tenant.Model, transactionId string) (uint32, Entry, bool)
  func (r *Registry) Submit(t tenant.Model, accountId uint32, transactionId string, name string) (Entry, bool)
  func (r *Registry) ClearAccount(t tenant.Model, accountId uint32)
  func (r *Registry) Sweep(t tenant.Model, now time.Time) []Expired
  const OpenTTL  = 5 * time.Minute
  const SubmittedTTL = 30 * time.Second
  ```
- Consumed by: Tasks 11, 12, 13, 14.

- [ ] **Step 1: Write the failing tests**

`registry_test.go` — table-driven where the cases share a shape, plain functions otherwise. Tenant fixtures built with `tenant.Create`/the repo's tenant builder as `remotemerchant/registry_test.go` does; no `*_testhelpers.go`.

| test | case | asserts |
|---|---|---|
| `TestPutThenGet` | put entry for account 7, tenant A | `Get` returns it; `Get` for account 8 returns `false`; `Get` for tenant B returns `false` |
| `TestPutIsIdempotentPerAccount` | two `Put`s for account 7 with different `At` | one entry survives, the second `Put`'s values win (design §3: a second `Open` refreshes rather than duplicating) |
| `TestTakeRemoves` | put, then `Take` twice | first returns `(entry, true)`, second `(zero, false)` — so a `CREATED` and a `FAILED` racing consume exactly once |
| `TestSubmitTransitionsPhase` | put `PhaseOpen`, `Submit(t, 7, "tx-1", "Chronicle")` | returns `(entry, true)`; the stored entry now has `Phase == PhaseSubmitted`, `TransactionId == "tx-1"`, `CandidateName == "Chronicle"`, and `At` refreshed |
| `TestSubmitWithoutOpenFails` | `Submit` for an account with no entry | returns `(zero, false)`; nothing is stored |
| `TestTakeByTransactionId` | put + `Submit` with `"tx-1"` | `TakeByTransactionId(t, "tx-1")` returns `(7, entry, true)` and removes it; a second call returns `false`; `TakeByTransactionId(t, "")` returns `false` without matching an entry whose `TransactionId` is also empty |
| `TestTakeByTransactionIdIsTenantScoped` | same `"tx-1"` under tenant A and tenant B | taking under A leaves B's entry intact |
| `TestClearAccount` | put, `ClearAccount` | `Get` returns `false` |
| `TestSweepUsesPhaseSpecificTTL` | entry X `PhaseOpen` aged `OpenTTL + 1s`; entry Y `PhaseOpen` aged 1 min; entry Z `PhaseSubmitted` aged `SubmittedTTL + 1s`; entry W `PhaseSubmitted` aged 5s | `Sweep` returns exactly X and Z and removes exactly those two |
| `TestSweepIsTenantScoped` | expired entries under tenant A and tenant B | `Sweep(A, now)` returns and removes only A's |

`SubmittedTTL` is 30s deliberately: it must outlive the orchestrator's 10s character-creation backstop (design §4.2) so a timed-out saga's `FAILED` still finds its record. Assert that relationship directly:

| test | asserts |
|---|---|
| `TestSubmittedTTLOutlivesSagaTimeout` | `SubmittedTTL > 10*time.Second` |

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./maplelife/... -v
```

Expected: build failure — package `maplelife` does not exist.

- [ ] **Step 3: Write the registry**

Copy the structure of `remotemerchant/registry.go` wholesale, changing the key from `CharacterId` to `AccountId` and adding the `Phase` / `TransactionId` fields and the `Submit` / `TakeByTransactionId` / phase-specific `Sweep` behaviour. The package doc comment states, as `remotemerchant/registry.go:1-19` does, why the state is in-process (the account's socket session lives on exactly one pod) and what losing it costs (one dialog, never an item — the item's fate belongs to the saga and to the pre-check ordering).

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./maplelife/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/maplelife
git commit -m "feat(atlas-channel): add the Maple Life pending-dialog registry"
```

---

## Task 9: The `atlas-character-factory` REST client in atlas-channel

### Files

- `services/atlas-channel/atlas.com/channel/character/factory/rest.go` — **new file**; `RestModel` + `CreateCharacterResponse`
- `services/atlas-channel/atlas.com/channel/character/factory/requests.go` — **new file**; `getBaseRequest` + `requestCreate`
- `services/atlas-channel/atlas.com/channel/character/factory/processor.go` — **new file**; `Processor` interface + `ProcessorImpl`
- `services/atlas-channel/atlas.com/channel/character/factory/processor_test.go` — **new file**
- `services/atlas-login/atlas.com/login/character/factory/` — **read-only reference**; the three files there are the pattern and must not be edited

Patterns to copy: `services/atlas-login/atlas.com/login/character/factory/requests.go` (the `requests.RootUrlFor(ctx, "CHARACTER_FACTORY")` + `requests.PostRequest` shape), `rest.go` (field set and JSON tags), `processor.go` (interface + impl shape). `services/atlas-channel/atlas.com/channel/character/name_validity_requests.go:60-70` shows the channel's own decorator conventions.

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces:**
- Produces:
  ```go
  type Processor interface {
      SeedCharacter(accountId uint32, worldId world.Id, name string, jobIndex uint32, subJobIndex uint16,
          face uint32, hair uint32, color uint32, skinColor uint32, gender byte,
          top uint32, bottom uint32, shoes uint32, weapon uint32,
          strength byte, dexterity byte, intelligence byte, luck byte) (string, error)
  }
  func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
  ```
  Returns the `transactionId` — this is the one deliberate divergence from atlas-login's `SeedCharacter`, which discards it (`services/atlas-login/atlas.com/login/character/factory/processor.go:30-42`). Task 14 correlates on it.
- Consumed by: Task 13.

- [ ] **Step 1: Write the failing test**

`processor_test.go` — table-driven over an `httptest.Server` standing in for the factory, with `CHARACTER_FACTORY_SERVICE_URL` pointed at it and a tenant context built the way the package's sibling tests do.

| case | factory responds | expects |
|---|---|---|
| accepted | `202` + `{"data":{"type":"characters","id":"tx-42","attributes":{"transactionId":"tx-42"}}}` | `SeedCharacter` returns `("tx-42", nil)` |
| rejected | `400` + an error body | returns `("", err)`; `err` is non-nil and its text contains `400` so the caller can classify it |
| unreachable | server closed before the call | returns `("", err)` |

| test | asserts |
|---|---|
| `TestSeedCharacterSendsSessionSuppliedIds` | the request body's `accountId` and `worldId` are exactly the arguments passed, and `level`/`hp`/`mp`/`mapId` carry the same defaults atlas-login sends (`1`, `50`, `5`, `0` — `services/atlas-login/atlas.com/login/character/factory/requests.go:34-45`) |
| `TestSeedCharacterCarriesTenantHeader` | the captured request has the tenant header set — the outbound call must never be tenant-blind |
| `TestSeedCharacterPostsToCharactersSeed` | the request path ends `characters/seed` |
| `TestSeedCharacterReturnsTransactionId` | the three cases above |

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./character/factory/... -v
```

Expected: build failure — package does not exist.

- [ ] **Step 3: Write the client**

`rest.go`: copy the `RestModel` field set and JSON tags verbatim from `services/atlas-login/atlas.com/login/character/factory/rest.go` — **read that file, do not copy the field list out of the PRD**, per PRD §5's own instruction. Same for `CreateCharacterResponse` and its `GetName`/`GetID`/`SetID`.

`requests.go`: `const Resource = "characters/seed"`; `getBaseRequest` returns `requests.RootUrlFor(ctx, "CHARACTER_FACTORY")`; `requestCreate` builds the `RestModel` and returns `requests.PostRequest[CreateCharacterResponse](root+Resource, i)`.

`processor.go`: `ProcessorImpl` with `l`/`ctx`, `var _ Processor = (*ProcessorImpl)(nil)`, and `SeedCharacter` returning `resp.TransactionId` alongside the error.

No configuration or deploy change is needed: `requests.RootUrlFor` resolves from `CHARACTER_FACTORY_SERVICE_URL` or `BASE_SERVICE_URL` (`libs/atlas-rest/requests/url.go:34-64`), `BASE_SERVICE_URL` is set for every service from `deploy/k8s/base/env-configmap.yaml:19`, and the ingress already routes `^/api/characters/seed(/.*)?$` (`deploy/k8s/base/routes.conf.template.generated:386-389`). Do not add a service-config seed row or an ingress entry.

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./character/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/factory
git commit -m "feat(atlas-channel): add the character-factory seed client"
```

---

## Task 10: Carry `TransactionId` on the seed status envelope

Design §4.3, confirmed at plan time. Additive to `atlas-character-factory`; `atlas-login` deserialises into its own copy of the struct and ignores unknown fields, so it stays byte-for-byte unmodified.

### Files

- `services/atlas-character-factory/atlas.com/character-factory/kafka/message/seed/kafka.go` — (8-12) add `TransactionId` to `StatusEvent`
- `services/atlas-character-factory/atlas.com/character-factory/kafka/producer/seed/producer.go` — thread `transactionId` through both providers
- `services/atlas-character-factory/atlas.com/character-factory/kafka/consumer/saga/consumer.go` — (73,131) pass `e.TransactionId.String()` into the two providers
- `services/atlas-character-factory/atlas.com/character-factory/kafka/consumer/saga/consumer_test.go` — **new file**; the bridge has no test file today
- `services/atlas-login/atlas.com/login/kafka/message/seed/kafka.go` — **read-only**; confirm the field's absence there is harmless, do not edit

Module root: `services/atlas-character-factory/atlas.com/character-factory`.

**Interfaces:**
- Produces: `seed.StatusEvent[E].TransactionId string` with tag `json:"transactionId,omitempty"`; `seed.CreatedEventStatusProvider(accountId uint32, characterId uint32, transactionId string)`; `seed.FailedEventStatusProvider(accountId uint32, reason string, transactionId string)`.
- Consumed by: Task 14's channel-side mirror of the same struct.

- [ ] **Step 1: Write the failing tests**

In `consumer_test.go`, drive `handleSagaCompletedEvent` and `handleSagaFailedEvent` with a captured producer.

| test | input | asserts |
|---|---|---|
| `TestCompletedBridgeCarriesTransactionId` | `saga.StatusEvent` with `TransactionId` = a fixed UUID, `Type` = completed, `Body.SagaType` = `character_creation`, `Results` carrying `accountId` 7 / `characterId` 99 | the emitted seed event's `TransactionId` equals that UUID's `String()`, `AccountId` is 7, `Body.CharacterId` is 99 |
| `TestFailedBridgeCarriesTransactionId` | same UUID, `Type` = failed, `SagaType` = `character_creation`, `Body.AccountId` 7, `Body.Reason` `"name_taken"` | emitted `TransactionId` equals that UUID's `String()`, `AccountId` 7, `Body.Reason` `"name_taken"` |
| `TestNonCharacterCreationSagaStillDropped` | `SagaType` = `inventory_transaction` | nothing is emitted — the existing filter at `consumer.go:53-56,96-104` is unchanged |
| `TestStatusEventMarshalsTransactionId` | `StatusEvent{AccountId: 7, TransactionId: "tx-1", Type: "CREATED"}` | `json.Marshal` output contains `"transactionId":"tx-1"`; marshalling with an empty `TransactionId` omits the key entirely |

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-character-factory/atlas.com/character-factory && go test ./kafka/... -v
```

Expected: FAIL — `TransactionId` is not a field of `seed.StatusEvent`.

- [ ] **Step 3: Make the additive change**

Add the field between `AccountId` and `Type`:

```go
type StatusEvent[E any] struct {
	AccountId uint32 `json:"accountId"`
	// TransactionId correlates this event with the POST characters/seed
	// response that started it. Optional: an older producer omits it and
	// consumers fall back to AccountId (task-246 design §4.3).
	TransactionId string `json:"transactionId,omitempty"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}
```

Add the `transactionId string` parameter to both providers and set it on the value. In `consumer.go`, pass `e.TransactionId.String()` at both call sites.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd services/atlas-character-factory/atlas.com/character-factory && go build ./... && go test ./... 
```

Expected: PASS.

- [ ] **Step 5: Confirm atlas-login is untouched and still compiles**

```bash
git status --porcelain services/atlas-login/
cd services/atlas-login/atlas.com/login && go build ./... && go test ./kafka/consumer/seed/... -v
```

Expected: `git status` prints nothing for atlas-login; build and tests pass. The unknown `transactionId` key is silently ignored by its own `StatusEvent` copy.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character-factory
git commit -m "feat(atlas-character-factory): carry transactionId on seed status events"
```

---

## Task 11: The 543 dispatch arm and the dialog-open handler

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — insert the classification-first 543 arm and the `mapleLifeSupported` helper
- `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_open.go` — **new file**; `beginMapleLife` and, if §3 found a v95 sender, `MapleLifeUseHandleFunc`
- `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_open_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_maple_life_test.go` — **new file**; the neighbour-collision regression suite
- `services/atlas-channel/atlas.com/channel/maplelife/registry.go` — **new file** at plan time, produced by Task 8; read-only here.
- `libs/atlas-constants/item/constants.go` — (116) read-only; `ClassificationCharacterCreation = Classification(543)`

Patterns to copy: `character_cash_item_use.go:779-793` (the classification-first `RemoteMerchant`/`Megaphones` arms and the comment explaining why classification precedes any `it ==` comparison); `character_cash_item_use.go:1103-1120` (`nameChangeCashSlotItemType`'s `t.IsRegion("GMS") && t.MajorAtLeast(95)` helper shape); `character_cash_item_use.go:1012-1040` (the package-var test-seam convention).

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces:**
- Consumes: `maplelife.GetRegistry()`, `maplelife.Entry`, `maplelife.PhaseOpen` (Task 8); `cashsb.NewItemUseMapleLife` (Task 6).
- Produces: `beginMapleLife(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, source slot.Position, updateTime uint32)`; `mapleLifeSupported(t tenant.Model) bool`; `MapleLifeUseHandleFunc(l, ctx, wp) func(s session.Model, r *request.Reader, readerOptions map[string]interface{})` if §3 found a v95 sender.

- [ ] **Step 1: Write the failing regression test — the neighbours must not move**

`character_cash_item_use_maple_life_test.go`. This is a PRD acceptance criterion and is non-negotiable: because the 543 arm routes on classification, the test asserts the collision *cannot* occur, not merely that it happens not to.

Setup — tenant contexts, session builder, and the `cashItemInSlotFunc` seam swap — copied from the existing handler tests in this package (`services/atlas-channel/atlas.com/channel/socket/handler/`; grep for `cashItemInSlotFunc = ` to find the established swap-and-restore idiom).

`TestCashItemNeighbourArmsUnaffectedByMapleLife` — table-driven, each case run against **both** a GMS v83 tenant and a GMS v95 tenant:

| case | item classification | expected `GetCashSlotItemType` (v83 / v95) | expected arm reached |
|---|---|---|---|
| pet multi-consumable | `ClassificationPetMultiConsumable` | 57 / 58 | the pet-multi-consumable arm, **not** Maple Life |
| sealing lock (timed) | the classification `GetCashSlotItemType` maps to `CashSlotItemTypeSealTimed`(64) / `SealTimedV95`(65) | 64 / 65 | the sealing-lock arm |
| vicious hammer | the classification mapping to `CashSlotItemTypeViciousHammer`(66) / `ViciousHammerV95`(67) | 66 / 67 | the vicious-hammer arm |
| maple life | `ClassificationCharacterCreation` (5431000) | 65 / 66 | the Maple Life arm |

The Maple Life row is what makes the other three meaningful: on v83 Maple Life shares 65 with `SealTimedV95` and on v95 it shares 66 with `ViciousHammer`, so a type-based arm would necessarily break one of them.

`TestMapleLifeArmUsesNoBareCashSlotTypeComparison` — reads `character_cash_item_use.go` and asserts the new arm's source contains none of `CashSlotItemType(57)`, `CashSlotItemType(58)`, `CashSlotItemType(65)`, `CashSlotItemType(66)` in its body. PRD FR-2.2 forbids these; the guard makes the ban executable rather than a comment.

`TestMapleLifeSupported` — table:

| tenant | expected |
|---|---|
| GMS 83 | `true` |
| GMS 84 | `true` |
| GMS 87 | `true` |
| GMS 92 | `true` |
| GMS 95 | `true` |
| GMS 79 | `false` |
| GMS 72 | `false` |
| GMS 61 | `false` |
| GMS 48 | `false` |
| JMS 185 | `false` |

`TestMapleLifeUnsupportedVersionWritesNothing` — on a GMS v79 tenant, using a 5431000 item produces **no** writer call and **no** registry entry, and the reader is left unconsumed past the common prefix (FR-2.4: no wire change on an out-of-scope version).

- [ ] **Step 2: Write the failing open-handler test**

`maple_life_open_test.go`:

| test | scenario | asserts |
|---|---|---|
| `TestBeginMapleLifeRecordsPending` | GMS v83 session, account 7, character 42, world 1, item 5431000 in slot -3, updateTime 1234 | `maplelife.GetRegistry().Get(t, 7)` returns an entry with exactly those values and `Phase == maplelife.PhaseOpen` |
| `TestBeginMapleLifeIsIdempotent` | call twice for the same account | one entry; the second call's `At` wins (design §3: v95 sending both entry points must not create two records) |
| `TestBeginMapleLifeUsesSessionAccountAndWorld` | session account 7 / world 1 | the recorded entry's account and world are the session's, not any packet-supplied value (FR-4.2) |

If `derivation.md` §3 found a v95 `USE_MAPLELIFE` sender, add `TestMapleLifeUseHandleRecordsPending` driving `MapleLifeUseHandleFunc` with a `maplelifesb.Use` fixture and asserting the same registry outcome — the two entry points must converge on one record.

- [ ] **Step 3: Run the tests to verify they fail**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run 'MapleLife' -v
```

Expected: FAIL — `beginMapleLife` / `mapleLifeSupported` undefined.

- [ ] **Step 4: Write the helper, the arm, and the open handler**

`mapleLifeSupported` goes beside `nameChangeCashSlotItemType` in `character_cash_item_use.go` (around `:1103`):

```go
// mapleLifeSupported reports whether the tenant's client has the Maple Life
// dialog at all. GMS v83+ only: USE_MAPLELIFE, MAPLELIFE_RESULT and
// MAPLELIFE_ERROR are all n-a on gms_v48/61/72/79 and on jms_v185
// (docs/packets/audits/status.json). Read-only guard — a tenant this returns
// false for gets no sub-body decode and no wire change (PRD FR-2.4).
func mapleLifeSupported(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(83)
}
```

The arm goes in `CharacterCashItemUseHandleFunc` immediately alongside the `ClassificationRemoteMerchant` / `ClassificationMegaphones` branches at `character_cash_item_use.go:786-793` — after `category := item.GetClassification(itemId)` is computed and **before** any `it ==` comparison:

```go
if category == item.ClassificationCharacterCreation {
    if !mapleLifeSupported(t) {
        l.Warnf("Character [%d] used Maple Life item [%d] on an unsupported client; ignoring.", s.CharacterId(), itemId)
        return
    }
    sp := cashsb.NewItemUseMapleLife(updateTimeFirst)
    sp.Decode(l, ctx)(r, readerOptions)
    if !updateTimeFirst {
        updateTime = sp.UpdateTime()
    }
    beginMapleLife(l, ctx, wp)(s, itemId, source, updateTime)
    return
}
```

The comment above it states, as the sibling arms' comments do, *why* it is classification-first: `GetCashSlotItemType`'s 543 branch (`:1457-1469`) returns 57/58/65/66 and every one of those is claimed elsewhere — 57/58 by `ClassificationPetMultiConsumable` (`:1485-1490`), 65 by `CashSlotItemTypeSealTimedV95` (`:970`), 66 by `CashSlotItemTypeViciousHammer` (`:991`).

`maple_life_open.go` holds `beginMapleLife`, which writes the `maplelife` registry entry with `Phase: PhaseOpen`, account and world from the session, and nothing else. It writes no packet — the client opens its own dialog. If §3 found a v95 sender, `MapleLifeUseHandleFunc` decodes `maplelifesb.Use` and calls the same `beginMapleLife`.

Ownership for this entry point is already enforced upstream: `character_cash_item_use.go:61-66` runs `cashItemInSlotFunc` against the common prefix's `source` slot and rejects a mismatch before any arm is reached (FR-5.3 for the open path). Note that in `beginMapleLife`'s doc comment so a reader does not add a redundant second check.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/... ./maplelife/... -v
```

Expected: PASS, including every pre-existing test in `socket/handler`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler services/atlas-channel/atlas.com/channel/maplelife
git commit -m "feat(atlas-channel): route Cash/0543 to the Maple Life dialog by classification"
```

---

## Task 12: The duplicate-name probe handler

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_check_name.go` — **new file**; the reason→arm table, the `checkNameValidity` call, the `MAPLELIFE_RESULT` write
- `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_check_name_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change.go` — **modify only under `derivation.md` §6 outcome (B)**; add the pending-record branch at the top of `CashShopCheckNameChangeHandleFunc`
- `services/atlas-channel/atlas.com/channel/character/name_validity_requests.go` — read-only; `NameScopeWorld`, the four `NameReason*` constants, `NameValidityResult`
- `docs/tasks/task-246-maple-life-character-creation/derivation.md` — **new file** at plan time, produced by Task 2; read-only here. §6 decides which of (A)/(B) applies.

Patterns to copy: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change.go` — the whole file: the `checkNameChangeValidityFunc` seam (`:26-28`) with its doc comment on why `scope` is a parameter rather than a baked-in constant, the map-not-switch reason table (`:82-104`), and the `announceCheckNameChange` helper.

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces:**
- Consumes: `maplelife.GetRegistry().Get` (Task 8); `mlcb.MapleLifeResultBody` / `MapleLifeResultRejectedBody` / `MapleLifeResultWriter` (Task 4); `msb.CheckName` (Task 6, outcome (A) only).
- Produces: `mapleLifeNameValidityFunc` (package-var seam, same signature as `checkNameChangeValidityFunc`); `MapleLifeCheckNameHandleFunc(l, ctx, wp) func(s session.Model, r *request.Reader, readerOptions map[string]interface{})` under outcome (A); `handleMapleLifeCheckName(l, ctx, wp)(s session.Model, name string)` — the shared body both outcomes call.

- [ ] **Step 1: Write the failing tests**

`maple_life_check_name_test.go`, driving `handleMapleLifeCheckName` with `mapleLifeNameValidityFunc` swapped for a recorder.

`TestMapleLifeCheckNameAsksForWorldScope` — asserts the seam was called with `character.NameScopeWorld`, and with the session's `WorldId()`. Asserted **through** the seam, per `checkNameChangeValidityFunc`'s own doc comment: re-stating the constant on the far side of the swap proves nothing. `WORLD`, not `TENANT` — this is a creation, and `TENANT` is the deliberately stricter rename-only scope (`name_validity_requests.go:20-23`).

`TestMapleLifeCheckNameMapsReasons` — table-driven, one case per reason, each asserting the exact `options` key the writer was handed:

| case | seam returns | expected `MAPLELIFE_RESULT` arm key |
|---|---|---|
| available | `{Valid: true}` | the available arm derivation.md §4 named |
| duplicate | `{Valid: false, Reason: "duplicate"}` | the taken/duplicate arm §4 named |
| reserved | `{Valid: false, Reason: "reserved"}` | the arm §4's mapping assigns — generic-failure if §4 shows no reserved-specific arm |
| length | `{Valid: false, Reason: "length"}` | as §4 assigns |
| regex | `{Valid: false, Reason: "regex"}` | as §4 assigns |
| unknown reason | `{Valid: false, Reason: "banana"}` | the generic-failure arm, **and** a log record at `logrus.ErrorLevel` — FR-3.3 requires the unmapped case be loud, never dropped |
| seam error | `(zero, errors.New("boom"))` | the generic-failure arm, and an error-level log |

Assert the log level with `testlog.NewNullLogger()`'s hook, as the package's other handler tests do.

`TestMapleLifeCheckNameReasonTableIsExhaustive` — the reason→arm map has an entry for each of `character.NameReasonLength`, `NameReasonRegex`, `NameReasonDuplicate`, `NameReasonReserved`. A map rather than a switch precisely so this test can be written (`cash_shop_check_name_change.go:82-88`).

`TestMapleLifeCheckNameWithoutPendingRecordIsRejected` — no live registry entry for the account: the generic-failure arm is written and a warning is logged; nothing is silently dropped (FR-5.4).

Under outcome (B) only, add to `maple_life_check_name_test.go`:

`TestCashShopCheckNameChangeDisambiguatesByPendingRecord` — table:

| case | pending Maple Life record? | expected writer |
|---|---|---|
| cash-shop rename in flight | no | `CashShopCheckNameChangeWriter`, scope `TENANT` |
| Maple Life dialog open | yes, `PhaseOpen` | `MapleLifeResultWriter`, scope `WORLD` |

This is the whole reason the pending store exists on those versions (design §4.1): the body is `EncodeStr(name)` in both cases and carries nothing to distinguish them.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run 'MapleLifeCheckName|CashShopCheckNameChangeDisambiguates' -v
```

Expected: FAIL — `handleMapleLifeCheckName` undefined.

- [ ] **Step 3: Write the handler**

`maple_life_check_name.go`:

- `mapleLifeNameValidityFunc` — a package var with the same signature as `checkNameChangeValidityFunc` (`cash_shop_check_name_change.go:26-28`), defaulting to `character.NewProcessor(l, ctx).CheckNameValidity`. A separate var from the cash-shop one so a test can move one without the other.
- `mapleLifeResultReasons` — a `map[string]string` from `character.NameReason*` to Task 4's arm-key constants, mirroring `nameChangeRejectionReasons` (`:82-88`).
- `handleMapleLifeCheckName` — looks up the pending record, calls the seam with `character.NameScopeWorld` and `s.WorldId()`, maps the outcome, and announces via `session.Announce(l)(ctx)(wp)(mlcb.MapleLifeResultWriter)(body)(s)`.
- Under outcome (A): `MapleLifeCheckNameHandleFunc` decodes `msb.CheckName` and calls it.
- Under outcome (B): the branch at the top of `CashShopCheckNameChangeHandleFunc` in `cash_shop_check_name_change.go` — if a live `PhaseOpen` Maple Life record exists for the session's account, delegate to `handleMapleLifeCheckName` and return; otherwise fall through unchanged. Extend that function's doc comment to record the second sender the opcode now carries.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/... -v
```

Expected: PASS, including the pre-existing `cash_shop_check_name_change_test.go` cases — the cash-shop rename's `TENANT` scope must be unchanged.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler
git commit -m "feat(atlas-channel): answer the Maple Life duplicate-name probe"
```

---

## Task 13: The submit handler — pre-checks and `POST characters/seed`

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create.go` — **new file**; the five ordered pre-checks and the factory call
- `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/maplelife/registry.go` — **new file** at plan time, produced by Task 8; read-only here. `Submit` is the entry point used.
- `services/atlas-channel/atlas.com/channel/character/factory/processor.go` — **new file** at plan time, produced by Task 9; read-only here. `SeedCharacter` is the entry point used.
- `services/atlas-channel/atlas.com/channel/account/model.go` — (40) read-only; `CharacterSlots() int16`
- `services/atlas-channel/atlas.com/channel/character/processor.go` — (248-258) read-only; `ForAccountInWorldProvider` / `GetForAccountInWorld`, already paged and drained

Patterns to copy: `character_cash_item_use.go:1012-1040` (package-var seams); `cash_shop_check_name_change.go:26-28` (seam with a parameter kept explicit so a test can assert the handler's choice).

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces:**
- Consumes: `maplelife.GetRegistry().Get`/`Submit`; `factory.NewProcessor(l, ctx).SeedCharacter`; `mlcb.MapleLifeErrorBody` and Task 5's arm keys; `mapleLifeNameValidityFunc` (Task 12).
- Produces: `handleMapleLifeCreate(l, ctx, wp) func(s session.Model, sub <the submit sub-body type from Task 6>)`; the seams `seedCharacterFunc`, `accountSlotsFunc`, `charactersInWorldFunc`.

- [ ] **Step 1: Write the failing tests**

`maple_life_create_test.go`, all seams swapped — no live REST, no live Kafka.

`TestMapleLifeCreatePreCheckOrder` — table-driven; each row makes exactly one gate fail and asserts the resulting `MAPLELIFE_ERROR` arm, that `seedCharacterFunc` was **not** called, and that no destroy saga was created:

| case | setup | expected arm |
|---|---|---|
| no pending record | registry empty for account 7 | generic-failure arm |
| record in wrong phase | entry present with `Phase == PhaseSubmitted` | generic-failure arm |
| ownership lost | `cashItemInSlotFunc` returns a different template id than `pending.ItemId` | generic-failure arm |
| ownership wrong classification | `cashItemInSlotFunc` returns an item whose `item.GetClassification` is not 543 | generic-failure arm |
| slot limit reached | `accountSlotsFunc` → 3, `charactersInWorldFunc` → 3 characters | the slot-limit arm derivation.md §5 named |
| name taken at submit | `mapleLifeNameValidityFunc` → `{Valid:false, Reason:"duplicate"}` | the duplicate-name arm §5 named |

`TestMapleLifeCreateSlotLimitBoundary` — `accountSlotsFunc` → 3:

| characters in world | expected |
|---|---|
| 2 | proceeds to `seedCharacterFunc` |
| 3 | slot-limit arm, no seed call |
| 4 | slot-limit arm, no seed call |

`TestMapleLifeCreateReChecksName` — asserts `mapleLifeNameValidityFunc` is called at submit time with `character.NameScopeWorld` even when the earlier probe passed. FR-4.5, and design C2's finding that the factory's seed path performs no duplicate check of its own makes this the only duplicate gate.

`TestMapleLifeCreateUsesSessionAccountAndWorld` — the submit packet carries an account id of 999 and a world id of 9; the session's are 7 and 1. Asserts `seedCharacterFunc` received 7 and 1, and that the mismatch was logged. FR-4.2.

`TestMapleLifeCreateMapsFactoryOutcomes` — table:

| `seedCharacterFunc` returns | expected client write | registry after | item |
|---|---|---|---|
| `("tx-1", nil)` | **nothing written yet** — the result awaits the seed event | entry present, `Phase == PhaseSubmitted`, `TransactionId == "tx-1"` | untouched |
| `("", HTTP 400 error)` | the invalid-look/invalid-name arm §5 named | entry removed | untouched |
| `("", HTTP 500 error)` | the generic-failure arm | entry removed | untouched |
| `("", transport error)` | the generic-failure arm | entry removed | untouched |

`TestMapleLifeCreateNeverConsumesTheItem` — across **every** row of the three tables above, assert zero destroy sagas were created. FR-5.1: consumption belongs to Task 14's `CREATED` path alone.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run TestMapleLifeCreate -v
```

Expected: FAIL — `handleMapleLifeCreate` undefined.

- [ ] **Step 3: Write the handler**

`maple_life_create.go` runs design §5.2's five gates in order, each with its own arm and each leaving the item alone:

1. live `PhaseOpen` record for `s.AccountId()`
2. ownership — `cashItemInSlotFunc(l, ctx, s.CharacterId(), int16(pending.Slot))` still returns `pending.ItemId`, and `item.GetClassification` of it is `ClassificationCharacterCreation`
3. slot limit — `accountSlotsFunc` vs `len(charactersInWorldFunc(...))`
4. name re-check — `mapleLifeNameValidityFunc(..., character.NameScopeWorld)`
5. account and world from the session; any packet-carried values are decoded, logged on mismatch, and discarded

Then `seedCharacterFunc`. On `nil` error, `registry.Submit(t, accountId, transactionId, name)` and write nothing — the outcome arrives on the seed topic. On error, classify by HTTP status and write the mapped arm, then `registry.Take` to drop the record.

The look fields go to the factory unvalidated by the channel, by design: `atlas-character-factory`'s `Create` validates face, hair, hairColor, skinColor, top, bottom, shoes, weapon, gender, jobIndex and subJobIndex against the tenant's own creation template and returns `400` for each (`services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:100-155`, `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:73-101`). Record that in the doc comment with the file:line, so a later reader sees a decision rather than an omission. TOCTOU on gates 3 and 4 is accepted and stated in the same comment: an account has one channel session, and the residual race surfaces as a saga `FAILED` → mapped arm → item retained.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/... ./maplelife/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler
git commit -m "feat(atlas-channel): submit Maple Life character creation to the factory"
```

---

## Task 14: The seed-status consumer, the consume saga, and `main.go` wiring

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/seed/kafka.go` — **new file**; the channel's copy of `StatusEvent` including `TransactionId`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/seed/consumer.go` — **new file**; `InitConsumers` / `InitHandlers` and the `CREATED` / `FAILED` handlers
- `services/atlas-channel/atlas.com/channel/kafka/consumer/seed/consumer_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/main.go` — (257,576) register the consumer and its handlers beside `cashshop`
- `libs/atlas-saga/model.go` — (42) add `MapleLifeUse Type = "maple_life_use"`
- `services/atlas-channel/atlas.com/channel/saga/model.go` — (77) add the `MapleLifeUse = sharedsaga.MapleLifeUse` alias
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` — (45) add the alias
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go` — (251,260) add `MapleLifeUse` to `noReverseWalkSagaTypes` **and** to `allSagaTypes`

Patterns to copy: `services/atlas-login/atlas.com/login/kafka/consumer/seed/consumer.go` — the whole file: the `EnvEventTopicStatus` consumer config, the two `message.AdaptHandler(message.PersistentConfig(...))` registrations, the `t.Is(tenant.MustFromContext(ctx))` guard, and the disconnected-session logging at `:82-88`. `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:620-640` for the `DestroyAssetFromSlot` step construction. `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go:53-60` for the channel's own `InitConsumers` shape.

Module roots: `services/atlas-channel/atlas.com/channel`, `libs/atlas-saga`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

**Interfaces:**
- Consumes: `maplelife.GetRegistry().TakeByTransactionId` / `Take` (Task 8); `mlcb.MapleLifeErrorBody` with Task 5's success and generic-failure arm keys; `saga.NewProcessor(l, ctx).Create` with `saga.MapleLifeUse` and `saga.DestroyAssetFromSlot`.
- Produces: `seed.StatusEvent[E]` with `AccountId`, `TransactionId`, `Type`, `Body`; `seed.InitConsumers(l)(rf)(consumerGroupId)`; `seed.InitHandlers(l)(ten)(wp)(rf)`; the `destroyCashItemFunc` seam.

`MapleLifeUse` belongs in `noReverseWalkSagaTypes`, not `reverseWalkSagaTypes`: the saga has exactly one step, so a failure has nothing prior to undo. Both lists must be updated or `TestEverySagaTypeIsClassified` (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer_test.go:180-203`) fails — which is the point of that test.

- [ ] **Step 1: Write the failing consumer tests**

`consumer_test.go`, driving the two handlers directly with a seeded registry and a `destroyCashItemFunc` recorder.

`TestSeedCreatedConsumesItemAndAnnounces` — registry seeded with a `PhaseSubmitted` entry for account 7, `TransactionId "tx-1"`, item 5431000 in slot -3:

| case | event | expected |
|---|---|---|
| matched by transaction id | `{AccountId: 7, TransactionId: "tx-1", Type: "CREATED", Body:{CharacterId: 99}}` | one destroy saga with `SagaType == saga.MapleLifeUse`, one step `DestroyAssetFromSlot` carrying `CharacterId` 42, `InventoryType byte(inventory.TypeValueCash)`, `Slot -3`, `TemplateId 5431000`, `Quantity 1`; `MAPLELIFE_ERROR` written with the success arm key; registry entry gone |
| fallback by account id | same event with `TransactionId: ""` | identical outcome — an older factory build mid-rollout still resolves (design §4.3) |
| wrong transaction id | `TransactionId: "tx-other"`, account 7 | **no** destroy saga, **no** client write, a warning logged, and the pending entry left intact |
| wrong tenant | event under tenant B, entry under tenant A | nothing happens; A's entry intact |
| duplicate delivery | the matched event delivered twice | exactly one destroy saga — `Take` removes the record under the lock |

`TestSeedFailedLeavesItemAndAnnounces`:

| case | event | expected |
|---|---|---|
| matched | `{AccountId: 7, TransactionId: "tx-1", Type: "FAILED", Body:{Reason: "name_taken"}}` | **no** destroy saga; `MAPLELIFE_ERROR` written with the generic-failure arm; the reason logged; registry entry gone |
| fallback by account id | same with `TransactionId: ""` | identical |
| unrelated account | `AccountId: 8` | nothing happens; account 7's entry intact |

`TestSeedCreatedWithDisconnectedSessionStillConsumes` — no session for account 7. Asserts the destroy saga **is** created, no client write is attempted, an info-level log is emitted, and nothing panics. Design §5.4: the entitlement was spent and the character exists; leaving the item would let one item produce two characters.

`TestSeedCreatedDestroyFailureIsLoggedNotRolledBack` — `destroyCashItemFunc` returns an error. Asserts an **error**-level log carrying the account, character, item and transaction ids, and that no compensating character deletion is attempted. Design §5.4: rolling a created character back to reclaim a cash item is destructive and disproportionate.

`TestMapleLifeUseIsClassified` in the orchestrator module — the existing `TestEverySagaTypeIsClassified` covers this once both lists are updated; run it rather than duplicating it.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/seed/... -v
```

Expected: build failure — package does not exist.

- [ ] **Step 3: Add the saga type**

`libs/atlas-saga/model.go` — beside `IncubatorUse` at `:42`:

```go
// MapleLifeUse is the classification-543 flow: create the character through
// atlas-character-factory FIRST, then destroy the cash item once the seed
// saga reports CREATED. One step, so there is nothing to reverse-walk — the
// item survives every failure by construction (task-246 design §5.4).
MapleLifeUse Type = "maple_life_use"
```

Add the alias in both `services/atlas-channel/atlas.com/channel/saga/model.go` and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go`, and add `MapleLifeUse` to **both** `noReverseWalkSagaTypes` and `allSagaTypes` in `timer.go`.

- [ ] **Step 4: Write the consumer**

`kafka/message/seed/kafka.go` mirrors `services/atlas-login/atlas.com/login/kafka/message/seed/kafka.go` plus the `TransactionId string \`json:"transactionId,omitempty"\`` field Task 10 added on the producing side.

`kafka/consumer/seed/consumer.go` follows `services/atlas-login/atlas.com/login/kafka/consumer/seed/consumer.go` structurally. Resolution order in both handlers: `TakeByTransactionId` when the event carries one, else `Take` by account id. A `TransactionId` present on the event but matching nothing is **not** a fallback to account id — it is a mismatch, logged and dropped, or the correlation guarantee buys nothing.

On `CREATED`: build the one-step saga with `saga.MapleLifeUse` and `saga.DestroyAssetFromSlot`, exactly as `character_cash_item_use.go:626-637` builds its `consume_sacrifice` step. Not `RequestItemConsume` — that routes through atlas-consumables' *use* semantics (effects, cooldowns), which is not what happens here; the item is destroyed because a purchase was fulfilled. Then announce `MAPLELIFE_ERROR` with the success arm if a session is present.

On `FAILED`: announce the generic-failure arm if a session is present; touch no item.

- [ ] **Step 5: Wire it into `main.go`**

Add `seedConsumer.InitConsumers(l)(cmf)(consumerGroupId)` beside `cashshop.InitConsumers` at `main.go:257`, and the matching `register(seedConsumer.InitHandlers(fl)(sc)(wp)(rh))` beside `cashshop.InitHandlers` at `main.go:576`. Register the two writers beside `cashcb.CashShopCheckNameChangeWriter` at `main.go:685`:

```go
mlcb.MapleLifeResultWriter,
mlcb.MapleLifeErrorWriter,
```

and the handlers beside `cashsb.CashShopCheckNameChangeHandle` at `main.go:1005`:

```go
handlerMap[msb.MapleLifeCheckNameHandle] = handler.MapleLifeCheckNameHandleFunc   // outcome (A) only
handlerMap[msb.MapleLifeUseHandle] = handler.MapleLifeUseHandleFunc               // only if §3 found a v95 sender
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd libs/atlas-saga && go build ./... && go test ./...
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./saga/... -run TestEverySagaType -v
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Expected: PASS on all three.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-saga services/atlas-saga-orchestrator services/atlas-channel
git commit -m "feat(atlas-channel): consume seed status and destroy the Maple Life item on CREATED"
```

---

## Task 15: Full verification and coverage promotion

### Files

- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated
- `docs/tasks/task-246-maple-life-character-creation/derivation.md` — **new file** at plan time, produced by Tasks 1-2; read-only here.

- [ ] **Step 1: Regenerate the matrix and run every packet-audit check**

```bash
cd tools/packet-audit && go run . matrix
cd tools/packet-audit && go run . operations --check
cd tools/packet-audit && go run . fname-doc --check
cd tools/packet-audit && go run . gate-check
cd tools/packet-audit && go run . template --check
```

Expected: exit 0 on all five.

- [ ] **Step 2: Confirm every in-scope cell promoted**

Open `docs/packets/audits/STATUS.md` and locate the five Maple Life rows — `MapleLifeResult`, `MapleLifeError`, `ItemUseMapleLife`, and (conditionally) `MapleLifeUse` and `MapleLifeCheckName`.

Expected: `MapleLifeResult` and `MapleLifeError` show ✅ on gms_v83, v84, v87, v92, v95. `ItemUseMapleLife` shows ✅ on every version `derivation.md` §2 gave an address for. `MapleLifeUse` shows ✅ on gms_v95 if §3 found a sender. A cell that has not promoted is a failure to report, not a claim to soften.

- [ ] **Step 3: Confirm no previously-verified cell of any other op changed**

```bash
git diff origin/main -- docs/packets/audits/status.json | grep -E '^[-+].*"state"'
```

Expected: only additions for the Maple Life ops and the Task 3 registry row. Any other op's state flipping is a defect — stop and diagnose before proceeding.

- [ ] **Step 4: Confirm the out-of-scope templates and atlas-login are untouched**

```bash
git diff --stat origin/main -- services/atlas-configurations/seed-data/templates/
git diff --stat origin/main -- services/atlas-login/
```

Expected: exactly `template_gms_{83,84,87,92,95}_1.json` in the first; **empty** output from the second.

- [ ] **Step 5: Run the full repo verification gate**

```bash
tools/verify.sh
```

Flagless — `--quick` and `--no-docker` also exit 0 but skip the bake and `-race`, so they do not count. Expected: exit 0. If a guard fails, `docs/verification.md` documents each guard's invariant and escape hatches.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/audits
git commit -m "chore(packets): promote the Maple Life coverage cells"
```

---

## Self-Review

**Spec coverage.**

| Requirement | Task |
|---|---|
| FR-1.1 `SendCreateNewCharacter` derivation | 1 |
| FR-1.2 cash-slot split re-derivation, signedness | 1 |
| FR-1.3 `USE_MAPLELIFE` v95, OQ-1 | 1 |
| FR-1.4 clientbound bodies + code enumerations | 2 |
| FR-2.1 dispatch arm | 11 |
| FR-2.2 classification-first, no bare type comparison | 11 (asserted by `TestMapleLifeArmUsesNoBareCashSlotTypeComparison`) |
| FR-2.3 `MajorAtLeast` helper | 11 (`mapleLifeSupported`) |
| FR-2.4 no wire change on `n-a` versions | 11 (`TestMapleLifeUnsupportedVersionWritesNothing`), 7 (out-of-scope templates unopened) |
| FR-3.1 answer the probe with `MAPLELIFE_RESULT` | 12 |
| FR-3.2 `WORLD` scope | 12 (`TestMapleLifeCheckNameAsksForWorldScope`) |
| FR-3.3 four-reason table, unmapped → generic + error log | 12 |
| FR-3.4 route the probe on every in-scope template | 7 |
| FR-4.1 `POST characters/seed`, no reimplementation | 9, 13 |
| FR-4.2 account/world from session | 11, 13 |
| FR-4.3 look validation (factory-owned, per plan-time decision) | 13 |
| FR-4.4 slot limit | 13 (`TestMapleLifeCreateSlotLimitBoundary`) |
| FR-4.5 name re-check at submit | 13 (`TestMapleLifeCreateReChecksName`) |
| FR-5.1 consume only on success | 13, 14 |
| FR-5.2 restore path | 14 — vacuous under create-then-consume; recorded in `context.md` |
| FR-5.3 ownership verified server-side | 11 (open, via the existing prefix check), 13 (submit) |
| FR-5.4 no silent terminal state | 11, 12, 13, 14 |
| FR-6.1 clientbound codecs, `Encode`+`Decode` | 4, 5 |
| FR-6.2 `USE_MAPLELIFE` serverbound, v95 | 6 |
| FR-6.3 fixtures, markers, evidence, matrix | 4, 5, 6, 15 |
| FR-6.4 route in-scope templates only | 7 |
| FR-6.5 no wire change to a verified cell | 15 (Step 3) |
| Design C1 / §7 registry hygiene | 2, 3 |
| Design C3 / §4.3 transaction-id correlation | 10, 14 |
| Design C4 — no config/deploy work | 9 (Step 3, stated as a non-action) |
| PRD §8 no-regression on 57/58/64/65/66/67 neighbours | 11 (`TestCashItemNeighbourArmsUnaffectedByMapleLife`) |

**Placeholder scan.** The `<derivation.md §N …>` markers in Tasks 3–7 are deliberate and are the only unfilled values in the plan: they are wire facts that do not exist until Tasks 1–2 run, and CLAUDE.md forbids inventing them. Each carries the exact section and table row that supplies it. Every non-wire value — test names, case tables, seam names, arm-key semantics, file paths, saga step fields, TTL relationships — is written out.

**Type consistency.** `maplelife.Entry` / `Phase` / `PhaseOpen` / `PhaseSubmitted` / `TransactionId` are used identically in Tasks 8, 11, 12, 13, 14. `SeedCharacter` returns `(string, error)` in Task 9 and is consumed as such in Task 13. `MapleLifeResultWriter` / `MapleLifeErrorWriter` are the same constants in Tasks 4, 5, 7, 12, 13, 14. `StatusEvent.TransactionId` is `string` on both the producing (Task 10) and consuming (Task 14) side.
