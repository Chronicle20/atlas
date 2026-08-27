# GMS v95.1 Packet Verification & Template Completeness — Design

Version: v1
Status: Draft
Created: 2026-08-27
Inputs: `docs/tasks/task-146-v95-packet-verification-batch/prd.md` (approved)
---

## 0. What the evidence changed about the PRD

The PRD was written from `status.json` cell *states*. Reading the grader
(`tools/packet-audit/internal/matrix/grade.go`, `build.go`) and the v95 export
changes the shape of W2 substantially, and adds two blockers the PRD did not
have. Everything below is quoted from the tree at branch point.

**Finding 1 — six of the nine W2 cells are already fixtured; the op row cannot
see the fixture.** `build.go:125-133` indexes reports by `baseFName(r.IDAName)`,
and `grade.go:282-292` (`findReport`) looks the op up by its **primary `fname`
only** — `fname_alts` are never consulted. Six of the nine W2 ops carry a
`provenance: csv-import` primary fname that **does not exist in the committed
v95 export** (`docs/packets/ida-exports/gms_v95.json`, 834 functions):

| Op × dir | Registry primary `fname` | In v95 export? | Sender that IS in the export |
|---|---|---|---|
| `ENTER_CASHSHOP` sb | `CWvsContext::SendMigrateToShopRequest` | no | — (no alt) |
| `CHANGE_MAP` sb | `CCashShop::SendTransferFieldPacket` | no | `CField::SendTransferFieldRequest` @0x5345c0 (alt) |
| `NPC_TALK` sb | `CNpc::ShowQuestList` | no | `CUserLocal::TalkToNpc` @0x9321f0 (alt) |
| `CHANGE_MAP_SPECIAL` sb | `CUserLocal::HandleUpKeyDown` | no | `CUserLocal::CheckPortal_Collision` @0x919a10 (alt) |
| `STORAGE` sb | `CTrunkDlg::SetRet` | no | `CTrunkDlg::SendGetItemRequest` @0x769e00 (alt) |
| `NPC_TALK` cb | `CScriptMan::OnPacket` | no | `CScriptMan::OnScriptMessage` @0x6de0f0 (no alt listed) |
| `MULTI_CHAT` sb | `CUIStatusBar::SendGroupMessage` | no | — (no alt) |
| `CHECK_SPW_RESULT` cb | `CLogin::OnCheckSPWResult` | no | — (no alt) |
| `NPC_ACTION` cb | `CNpc::OnMove` | **yes** @0x678060 | itself |

The codecs behind five of those six are **already `verified` on `gms_v95`** at
the packet level:

```
field/serverbound/FieldChange                  verified   marker✓ evidence✓ report✓
npc/serverbound/NpcStartConversation           verified   marker✓ evidence✓ report✓
portal/serverbound/PortalScript                verified   marker✓ evidence✗ report✓
storage/serverbound/StorageOperation           verified   marker✓ evidence✓ report✓
  + StorageOperation{Meso,RetrieveAsset,StoreAsset}      all verified
npc/clientbound/NpcNpcConversation             verified   marker✓ evidence✓ report✓
```

So W2 is not "nine verifications." It is **three real verifications** (rows 1,
2, 8), **one real verification of an existing report** (row 9, `NPC_ACTION` cb —
`tier-1 without fixture; verdict 🔍`), and **five linkage repairs** where the
wire is already proven and the registry key is wrong.

**Finding 2 — `AUTO_DISTRIBUTE_AP` and `DISTRIBUTE_AP` both claim v95 opcode
98.** `docs/packets/registry/gms_v95.yaml` gives both ops `opcode: 98`
(0x62); `gms_v83.yaml` gives them 87 and 88 — distinct. `template_gms_95_1.json`
already routes 0x62 to `CharacterDistributeApHandle`. W1 cannot add
`CharacterAutoDistributeApHandle` at 0x62 without displacing it. One of the two
v95 registry opcodes is a csv-import error. **Blocker B1**, resolved by IDB
derivation in Task W0 below — never by assuming the v83 `+1`.

**Finding 3 — the v95 Pong slot is stubbed.** `template_gms_95_1.json` routes
`0x19` to `NoOpHandler` with `NoOpValidator`, `services: [login, channel]` —
byte-for-byte the shape v83 gives `PongHandle` at `0x18`, and `PONG` sb is 25
(0x19) in the v95 registry. W1's `PongHandle` add is a **replacement** of that
stub, not an insertion.

**Finding 4 — Open Question 3 resolves clean.** All nine W1 handlers and all
three W1 writers have reachable implementations. The one that looks missing is
a naming artifact: `libs/atlas-packet/chat/serverbound/multi.go:15` declares
`const CharacterChatMultiHandle = "CharacterMultiChatHandle"`, registered at
`services/atlas-channel/atlas.com/channel/main.go:973`. `PicResult` is
registered in `services/atlas-login/atlas.com/login/main.go:353`;
`NpcConversationWriter = "NPCConversation"` at
`libs/atlas-packet/npc/clientbound/conversation.go:14`. No inert route, no stub.

**Finding 5 — `families.yaml` `dispatchers:` is empty.** Every mode-prefix
family has graduated (task-183). Nothing in W2 is capped at 🧩, so the family
question in Open Question 1 is not a grading question — it is only an honesty
question about arm coverage.

## 1. Architecture of the change

Five artifact classes, in dependency order:

```
W0  registry corrections  docs/packets/registry/gms_v95.yaml
        │                 (fname promotion ×5, opcode derivation ×2, packet: ×4)
        ├──────────────► W1  template routing   template_gms_95_1.json (+12 entries)
        │                 W3  template options   template_gms_95_1.json (12 blocks)
        │                        └── new handler-keyed docs/packets/dispatchers/*.yaml ×6
        ▼
W2  fixtures + evidence   libs/atlas-packet/**/*_test.go, docs/packets/evidence/gms_v95/
        │
        ▼
    regenerate            docs/packets/audits/STATUS.md + status.json
```

W0 must land before W2's matrix regeneration, because a `fname`/`packet:` change
moves which report (or which fixture) a cell grades against.

## 2. W0 — registry corrections (new workstream)

### 2.1 fname promotion (5 ops)

For each of the five ops whose primary is absent from the v95 export while a
proven alt is present, **swap**: the export-resident, report-backed sender
becomes `fname`; the csv-import name is demoted into `fname_alts` (never
deleted — it is the CSV's name for the op and other versions still key on it).

| Op × dir | New `fname` | Demoted to `fname_alts` |
|---|---|---|
| `CHANGE_MAP` sb | `CField::SendTransferFieldRequest` | `CCashShop::SendTransferFieldPacket`, `CITC::SendTransferFieldPacket` |
| `NPC_TALK` sb | `CUserLocal::TalkToNpc` | `CNpc::ShowQuestList` |
| `CHANGE_MAP_SPECIAL` sb | `CUserLocal::CheckPortal_Collision` | `CUserLocal::HandleUpKeyDown` |
| `NPC_TALK` cb | `CScriptMan::OnScriptMessage` | `CScriptMan::OnPacket` |

Each promotion is admissible **only** after the v95 IDB confirms the promoted
function is a real `COutPacket::COutPacket(&pkt, <op>)` build site for that
opcode (send) or the real dispatch target (receive). `run.go` already documents
the addresses for all four (`run.go:3073`, `:3840`, `:3047-3052`, `:3758`), but
the promotion re-checks them live rather than trusting the comment — the
`packet-audit` comments are derived, not primary.

`STORAGE` sb is deliberately **not** in that table. Its four Atlas codecs are
four distinct senders; promoting any one of them would make the op row grade
that single arm and silently hide the other three (`worstCandidateCell` only
walks writers whose report `IDAName` equals the primary). It takes the `packet:`
route instead (§2.3).

**Why promotion rather than a `candidatesFromFName` case.** Adding a case for
the csv name would need a *report* whose `IDAName` is that csv name, which needs
the function to exist in the export, which it does not — closing that path would
require a live harvest plus a surgical export splice (VERIFYING_A_PACKET §10)
for a function the client does not have under that name. Promotion costs one
YAML edit and makes the registry *more* truthful, not less.

### 2.2 opcode derivation (2 ops) — Blocker B1

`AUTO_DISTRIBUTE_AP` sb and `DISTRIBUTE_AP` sb both read `opcode: 98`. Derive
both from the v95 IDB: `CWvsContext::SendAbilityUpRequest` and its auto-assign
sibling each construct `COutPacket` with an integer literal; those two integers
are the answer. Whichever op's registry opcode disagrees is corrected. If the
IDB shows the two genuinely share one opcode (a sub-op byte demux), then W1
drops `CharacterAutoDistributeApHandle` entirely — one opcode routes one
handler — and that decision is recorded, not worked around.

`PARTY_RESULT` sb 146 (0x92) is the registry-backed candidate opcode for
`PartyInviteRejectHandle` (v83 routes it at `PARTY_OPERATION + 1` = 0x7D; v95
`PARTY_OPERATION` sb is 145/0x91). 0x92 is unoccupied in the v95 template. It is
still IDB-confirmed before it is written.

### 2.3 `packet:` links (4 ops)

`grade.go:127-139` gives a second promotion path: when no report resolves but
the registry entry carries `packet: <pkg>/<dir>/<Struct>`, grading reads that
packet's marker + evidence directly. Twenty-eight v95 entries already use it.
Add it for the four ops whose codec is fixtured but whose fname will not resolve
to a report:

| Op × dir | `packet:` | Today |
|---|---|---|
| `STORAGE` sb | `storage/serverbound/StorageOperation` | marker ✓ evidence ✓ |
| `ENTER_CASHSHOP` sb | `cash/serverbound/CashShopEntry` | W2 supplies both |
| `MULTI_CHAT` sb | `chat/serverbound/ChatMulti` | W2 supplies both |
| `CHECK_SPW_RESULT` cb | `login/clientbound/LoginPicResult` | W2 supplies both |

The exact packet id is `qualifiedWriterName(pkg, name)` = TitleCase(pkg)+name
(VERIFYING_A_PACKET §9); the three above resolve to `CashShopEntry`
(pkg `cash` + struct `ShopEntry`), `ChatMulti`, `LoginPicResult`. Plan must
confirm each against the marker the fixture actually writes rather than
re-deriving the rule by hand.

**Consequence for tiering.** The no-report path requires marker **and fresh
evidence** regardless of tier (`grade.go:200-215`). `ENTER_CASHSHOP`,
`MULTI_CHAT` and `CHECK_SPW_RESULT` are tier-0 rows, so this design **pins
evidence for three tier-0 cells**, against VERIFYING_A_PACKET §7's default. That
is the documented exception in §7 itself ("or a deferral that evidence
justifies"): without a record the cell cannot promote at all, because no report
exists and none can be generated without a live harvest for a function absent
from the export. The freshness liability is accepted and stated in each record's
`category`.

`CHANGE_MAP_SPECIAL` deliberately takes the promotion route (§2.1) instead of
`packet:`, precisely to avoid pinning a fourth tier-0 record: `PortalScript` has
a report, so promotion lets the cell grade off the report + tool verdict with no
evidence at all.

### 2.4 `LOGIN_AUTH` cb is not proven absent

The PRD lists `LoginAuth` under W4 as "no GMS v95 counterpart." The registry
disagrees with that reading: `gms_v95.yaml:4` has `LOGIN_AUTH` clientbound with
`opcode: 0`, `fname: ""` — an **unresolved csv-import seed**, which is
absence-of-evidence, not evidence-of-absence (VERIFYING_A_PACKET, "Is this cell
n-a?", rule 1). `gms_v83/v84/v87` all carry it at opcode 23 as
`CLogin::OnEnableSPWResult`, and v95 has `CHECK_SPW_RESULT` at 27. Resolve it
in the IDB by the invariant anchor (the `OnEnableSPWResult` dispatch arm and its
`StringPool` ids), then either fill in the real opcode + fname, or delete the
entry from `gms_v95.yaml` — deletion is how absence is expressed
(`opregistry.go:144-153`: not present ⇒ `Absent` ⇒ ⬜). `gms_v92.yaml` carries
the identical `opcode: 0` seed; leave it (out of scope, §7 follow-up).

## 3. W1 — template routing

Twelve entries added to `template_gms_95_1.json`, each opcode taken from the
**v95 registry** (post-W0) and never from v83.

| Entry | Kind | Source op × dir | v95 opcode | Note |
|---|---|---|---|---|
| `CashShopEntryHandle` | handler | `ENTER_CASHSHOP` sb 43 | `0x2B` | free |
| `MapChangeHandle` | handler | `CHANGE_MAP` sb 41 | `0x29` | free |
| `NPCStartConversationHandle` | handler | `NPC_TALK` sb 63 | `0x3F` | free |
| `StorageOperationHandle` | handler | `STORAGE` sb 67 | `0x43` | free; carries `operations` (§4.1) |
| `PortalScriptHandle` | handler | `CHANGE_MAP_SPECIAL` sb 112 | `0x70` | free |
| `CharacterMultiChatHandle` | handler | `MULTI_CHAT` sb 140 | `0x8C` | free |
| `PartyInviteRejectHandle` | handler | `PARTY_RESULT` sb 146 | `0x92` | free; IDB-confirm |
| `PongHandle` | handler | `PONG` sb 25 | `0x19` | **replaces** the `NoOpHandler` stub; `NoOpValidator`, `services: [login, channel]` |
| `CharacterAutoDistributeApHandle` | handler | `AUTO_DISTRIBUTE_AP` sb | **B1** | blocked on §2.2 |
| `PicResult` | writer | `CHECK_SPW_RESULT` cb 27 | `0x1B` | `services: [login]` |
| `CashShopOpen` | writer | `SET_CASH_SHOP` cb 143 | `0x8F` | `services: [channel]` |
| `NPCConversation` | writer | `NPC_TALK` cb 363 | `0x16B` | `services: [channel]`; carries `messageType` (§4.2) |

Validators and `services` mirror the v83 entry (all `LoggedInValidator`,
`services: [channel]`, except `PongHandle` and `PicResult` as noted) — those are
Atlas-side policy, not client-derived, so carrying them across is correct where
the opcode is not.

**Ordering.** `template_gms_95_1.json` keeps handler and writer arrays sorted by
`opCode`; new entries are inserted in place, not appended, and the file's
existing line endings are preserved (CLAUDE.md; the NFR in PRD §8).

## 4. W3 — template options

Three distinct table kinds hide under "populate `options`", and only one of them
has a generator and a CI gate. Treating them alike is how the v83→v95 drift this
task exists to catch would be re-introduced.

### 4.1 `operations` (5 handlers + `StorageOperationHandle`) — generated

`tools/packet-audit/cmd/operations.go` generates `options.operations` /
`options.errors` into templates from `docs/packets/dispatchers/*.yaml`, and
`operations --check` is a CI gate (`.github/workflows/packet-matrix.yml:43`).
Today it exits 0 with "0 absent-writer notes" **because no dispatcher doc covers
these six handlers.** The five existing docs — `party.yaml`, `guild.yaml`,
`buddy.yaml`, `messenger_operation.yaml`, `guild_bbs.yaml` — are all
`writer:`-keyed **clientbound** result tables (`CWvsContext::OnPartyResult` etc.).
The W3 gap is the **serverbound handler** table, a different enumeration derived
from the client's *send* sites, which nothing in the repo currently owns.

So: author six new **handler-keyed** docs, on the model of the four that already
exist (`cash_shop_operation_handle.yaml`, `character_interaction_handle.yaml`,
`duey_action.yaml`, `cash_shop_coupon_code.yaml`):

```
docs/packets/dispatchers/party_operation_handle.yaml       PartyOperationHandle
docs/packets/dispatchers/guild_operation_handle.yaml       GuildOperationHandle
docs/packets/dispatchers/buddy_operation_handle.yaml       BuddyOperationHandle
docs/packets/dispatchers/messenger_operation_handle.yaml   MessengerOperationHandle
docs/packets/dispatchers/guild_bbs_handle.yaml             GuildBBSHandle
docs/packets/dispatchers/storage_operation_handle.yaml     StorageOperationHandle
```

Each carries `handler:`, `fname:` (the registry primary send-site), `op:`,
`direction: serverbound`, and an `operations:` list whose modes come from
enumerating that opcode's sub-op byte in the **v95 IDB**. Then
`go run ./tools/packet-audit operations` writes the tables and `--check` gates
them forever after.

`storage_operation_handle.yaml` additionally carries
`opcodes: { gms_v95: "0x43" }`, `validator: LoggedInValidator`,
`services: [channel]`, which makes `operations` **generate the whole W1 entry**
rather than hand-adding it (`operations.go:60-73`). Doing the same for the other
five is unnecessary — they are already routed.

**Scoping decision: the new docs declare a `gms_v95` column only.** A doc that
also declared `gms_v83` would make `operations` rewrite v83's hand-authored
table, violating PRD §8's "no non-v95 template is modified" — and those columns
would be un-derived carry-over, exactly the drift the gate exists to stop.
Consequence: v83/v84/v87/jms handler tables stay hand-maintained and un-gated.
That is a knowing narrowing, filed as the §7 follow-up, not an oversight.

### 4.2 `messageType` (1 handler) — hand-authored, ungated

`NPCContinueConversationHandle.options.messageType` is a flat `string → int`
map (14 keys on v83) with no generator and no schema in `packet-audit`. Derive
each value from the v95 `CScriptMan::OnPacket` / `OnScriptMessage` `msgType`
switch — the same switch `run.go:3758-3830` already enumerates for the 14
`*ConversationDetail` structs, whose per-type bodies are verified on v95. The
detail structs are the cross-check: a `messageType` value that does not match
the case its detail struct is fixtured against is wrong.

### 4.3 `types` (4 movement writers) — hand-authored, ungated, highest risk

`options.types` is **not** a map. It is an **index-addressed array** of
`{Name, Type}` objects (`libs/atlas-packet/model/movement.go:384-405`): the
movement fragment's attribute byte indexes it directly, and `Type` selects the
fragment's decode shape. A missing table means *every* fragment resolves
`NOT_FOUND`/`DEFAULT` — v95 movement decode is currently misparsing, which makes
this the highest-severity item in the task even though it has no matrix cell.

Derive the array from the v95 client's move-path attribute switch (the
`CVecCtrl`/`CMovePath` decode reached from `CUserRemote::OnMove`,
`CMob::OnMove`, `CPet::OnMove`, `CNpc::OnMove` — the same four functions the
four writers key on). Index order **is** the wire contract; a shifted index is a
silent misparse, not an error. v83's 23-entry table is a shape reference for the
`{Name, Type}` vocabulary only; no v83 index is copied.

Protection, in lieu of a CI gate: each of the four movement writers already has
byte fixtures in `libs/atlas-packet/*/clientbound/movement_test.go` that pass a
`types` table in as options. Extend the v95 fixture of each to feed **the exact
table this task writes into the template** (read from a shared test fixture, not
a hand-copy), so a future edit to the template that desynchronises the array
fails `go test`. Per-index IDA citations go in
`docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md`.

### 4.4 `failedReasonCodes` (2 writers) — hand-authored, ungated

`AuthPermanentBan` and `AuthTemporaryBan` share one table (both are arms of
`CLogin::OnCheckPasswordResult`, `LOGIN_STATUS` cb, verified on v95). Derive the
19 codes from that function's v95 reason switch. Both writers get the identical
block, as on v83.

## 5. W2 — cell verification, resolved per row

| # | Op × dir | Route | Work |
|---|---|---|---|
| 1 | `ENTER_CASHSHOP` sb 43 | `packet:` | **full verify**: `cash/serverbound/shop_entry.go` has no v95 marker/evidence/report. `CWvsContext::SendMigrateToShopRequest` is absent from the export ⇒ live IDB decompile required. tier-0 evidence pin (§2.3). |
| 2 | `MULTI_CHAT` sb 140 | `packet:` | **full verify**: `chat/serverbound/multi.go`, no v95 artifacts. `run.go:1775-1783` records a v95-only leading `Encode4(updateTime)` — re-derive live, do not trust the comment. tier-0 evidence pin. |
| 3 | `CHANGE_MAP` sb 41 | fname promotion | **linkage only**. `FieldChange` already marker+evidence+report verified (`change_test.go:158`, ida=0x5345c0). |
| 4 | `NPC_TALK` sb 63 | fname promotion | **linkage only**. `NpcStartConversation` verified (`start_conversation_test.go:12`, ida=0x9321f0). |
| 5 | `CHANGE_MAP_SPECIAL` sb 112 | fname promotion | **linkage only**. `PortalScript` verified via report (`script_test.go:11`, ida=0x919a10); no evidence pin needed (tier-0). |
| 6 | `STORAGE` sb 67 | `packet:` | **linkage only**. `StorageOperation` + 3 arm structs all verified with markers and evidence. |
| 7 | `NPC_TALK` cb 363 | fname promotion | **linkage + 1 arm**. `NpcNpcConversation` verified (`conversation_test.go:64`, ida=0x6de0f0); 13 of 14 detail structs verified; `NpcAskSlideMenuConversationDetail` is `incomplete: no audit report` on v95 and is the only real gap. |
| 8 | `CHECK_SPW_RESULT` cb 27 | `packet:` | **full verify**: `login/clientbound/pic_result.go` is `type PicResult struct{}` — a bodyless packet; the fixture is the opcode alone. `CLogin::OnCheckSPWResult` absent from the export ⇒ live IDB. tier-0 evidence pin. |
| 9 | `NPC_ACTION` cb 314 | already linked | **full verify**: report exists, cell reads `tier-1 without fixture; verdict 🔍`. tier-1 ⇒ byte fixture + evidence pin, straight playbook. |

Three live-IDB decompiles (rows 1, 2, 8), one export-resident decompile (row 9),
one arm (row 7). Everything else is YAML.

### 5.1 Open Question 1 — family scope: arm-complete, no family dispatch

`docs/packets/evidence/families.yaml` `dispatchers:` is **empty** and
`dispatcher-lint` reports `clean`, so neither `STORAGE` sb nor `NPC_TALK` cb is
capped at 🧩 and neither needs a `dispatcher-family-implementer` pass — both are
already discrete-per-mode. Arm coverage is nonetheless completed, because it is
cheap here: `STORAGE` sb's four arms are all verified today, and `NPC_TALK` cb
needs exactly one detail struct (`AskSlideMenu`). Verifying it is one fixture,
so "verify the primary arm and leave the row at the worst sibling" would be the
PRD's forbidden "documented gap when the blocker is producible."

### 5.2 Open Question 4 — `CHANGE_MAP` alt reconciliation

`CField::SendTransferFieldRequest` is in the export at 0x5345c0 with a report and
a passing fixture; `CCashShop::SendTransferFieldPacket` and
`CITC::SendTransferFieldPacket` are in neither the export nor the IDB under those
names. Treat them as CSV aliases of the one field-transfer request shape, keep
them as `fname_alts`, and record the negative search in the task folder. If the
live IDB turns up a genuinely distinct cash-shop or ITC transfer build site with
a different read order, that is a **second** codec and a second cell — surface it,
do not fold it into `FieldChange`.

## 6. W4 — `n-a`, resolved

**Open Question 5: no.** Do not create matrix rows to hold an `n-a` verdict.
The grader derives ⬜ from *absence* in the version registry
(`opregistry.go:144-153`), so a row invented to hold a verdict would be a row
the tool has to be told to ignore. `docs/packets/feature-na-evidence.yaml` is a
narrower instrument than the PRD assumes: `na_consistency.go` consults it only
when a cell is `n-a` **while a same-feature sibling is `verified`**, and
`feature-families.yaml` has five families (teleport_rock, claim,
cash_item_gachapon, dragon, skill_macro) — none of the four W4 symbols is in
any of them. No `feature-na-evidence.yaml` entry is required or appropriate.

Per symbol:

| Symbol | Disposition |
|---|---|
| `CreateSecurityHandle` | jms-only template symbol, no Atlas struct, no matrix row, no v95 registry op. Nothing to record in tooling. Positive-absence proof (invariant anchors, per VERIFYING_A_PACKET) goes in `v95-na-proof.md` in the task folder. |
| `WorldSelectHandle` | Same, `template_gms_12_1` only. There is no `gms_v12` registry file at all. Task-folder proof only. |
| `ServerLoad` (writer) | `libs/atlas-packet/login/clientbound/server_load.go` exists but has no v95 report and therefore no sub-struct row. If the v95 sweep proves the wire is absent, add a `{fname, packet, reason}` entry to `docs/packets/audits/gms_v95/_unimplemented.json` (the one existing entry of that shape is the `cash/serverbound/CashItemUsePetSkill` precedent) so a future report can never grade it ❌. |
| `LoginAuth` (writer) | **Not n-a by default** — see §2.4. Resolve the `opcode: 0 / fname: ""` seed in the IDB first; the outcome decides between "fill in" and "delete the registry entry." |

The bar for all four is the same as a positive verification: anchor on
invariants (the `COutPacket` construction site, the dispatcher jump arm, the
`StringPool` ids), never on a failed name search.

## 7. Scope boundaries and follow-ups

Carried forward from the PRD, plus what this design adds:

- **Open Question 2 — yes, file a follow-up.** `gms_v87` (16/145 handlers with
  options), `gms_v92` (13/89), `jms_v185` have the same `options: null` gap.
  Out of scope here.
- The six new handler-keyed dispatcher docs declare a `gms_v95` column only
  (§4.1). Backfilling IDA-derived v83/v84/v87/jms columns — which would put
  those templates' tables under the CI gate — is the same follow-up.
- `gms_v92.yaml` carries the same unresolved `LOGIN_AUTH` `opcode: 0` seed
  (§2.4). Left alone.
- `messageType`, `types` and `failedReasonCodes` remain outside
  `packet-audit`'s schema. Extending the tool to gate them is a candidate
  follow-up; this task protects `types` with test-fixture coupling (§4.3) and
  leaves the other two on review.

## 8. Risks

| Risk | Mitigation |
|---|---|
| **B1**: the two AP ops share an opcode in the IDB, so only one handler can route. | §2.2 — resolve in IDB before W1; if genuinely shared, drop the second handler and record why. |
| A promoted `fname` (§2.1) turns out to name a *different* wire shape than the demoted CSV name. | Promotion is gated on a live `COutPacket` opcode check, not on the `run.go` comment. A divergence is a second codec + second cell, surfaced not folded. |
| Three tier-0 evidence pins (§2.3) become freshness liabilities on the next export refresh. | Accepted and documented per record. The alternative — a live harvest + surgical export splice for three functions absent from the export — is strictly worse (VERIFYING_A_PACKET §10: the export is not idempotent). |
| A `types` index derived off-by-one silently misparses every v95 movement packet. | §4.3 — the template's array is fed into the four movement byte fixtures, so a desync fails `go test`, and per-index IDA citations are recorded. |
| A W1 opcode collides with an existing v95 route. | Checked at design time: only `0x19` (NoOpHandler stub, intentionally replaced) and `0x62` (B1) collide. Re-checked mechanically before the template edit. |
| `matrix --check` degrades an unrelated cell after the registry edits. | It exits 0 at branch point (verified this session, with two expected `n-a evidence consumed` notes). Re-run after every W0/W2 commit, not once at the end. |

## 9. Gates

Unchanged from PRD §8/§10, with the invocation confirmed this session (all run
flagless from the worktree root):

```
go run ./tools/packet-audit matrix --check        # exits 0 at branch point
go run ./tools/packet-audit operations --check    # exits 0 at branch point
go run ./tools/packet-audit dispatcher-lint       # "dispatcher-lint: clean"
go run ./tools/packet-audit fname-doc --check     # exits 0 at branch point
tools/verify.sh                                   # flagless, before PR
```

`na-consistency` is **not** a standalone subcommand — it runs inside
`matrix --check` (`na_consistency.go:15-24`). The PRD's separate `na-consistency`
acceptance line is satisfied by `matrix --check` exiting 0.

`docs/tasks/task-146-v95-packet-verification-batch/coverage-manifest.yaml` is
written before review, declaring every op × version this branch changes coverage
for, on the model of `docs/tasks/task-139-pet-auto-pot-validation/coverage-manifest.yaml`.
