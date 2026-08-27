# GMS v95.1 Packet Verification & Template Completeness — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-27
---

## 1. Overview

The GMS v95 column of the packet coverage matrix (`docs/packets/audits/STATUS.md`,
`docs/packets/audits/status.json`) still carries `incomplete` cells for a set of
core gameplay and login ops, and the tenant template that drives the v95 runtime
(`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`) is
incomplete in two independent ways: a set of handlers/writers is **not routed at
all**, and a second set is **routed but carries `"options": null`**, so its
dispatcher mode table / movement type table / reason-code table cannot be
resolved at runtime.

The request that seeded this task named 28 handler/writer symbols. Resolving each
against the v95 template, the v95 registry (`docs/packets/registry/gms_v95.yaml`)
and `status.json` shows they are not 28 units of the same work — they decompose
into four distinct workstreams (W1–W4 below), and a substantial fraction is
already `verified` on v95 and needs no verification work at all. This PRD records
that decomposition so downstream phases do not re-derive it, and so no already-green
cell is needlessly re-verified.

The outcome is a v95 column where every op named here is either `verified` with a
pinned byte fixture or provably `n-a`, and a `template_gms_95_1.json` that routes
and fully configures every handler/writer the v95 client can actually send or
receive for those ops.

## 2. Goals

Primary goals:

- Promote 9 `gms_v95` coverage-matrix cells from `incomplete` to `verified` via
  the `docs/packets/audits/VERIFYING_A_PACKET.md` playbook (W2).
- Close the v95 template routing gap: add the 9 handlers and 3 writers that v83
  routes and v95 does not (W1).
- Close the v95 template configuration gap: populate `options` for the 6 handlers
  and 6 writers that are routed in v95 with `"options": null` but carry a mode /
  type / reason-code table in v83 (W3).
- Prove and record `n-a` for the 4 symbols that have no GMS v95 counterpart (W4).

Non-goals:

- Re-verifying cells already `verified` on `gms_v95` (see §4.4). Their existing
  evidence is accepted as-is; no re-derivation.
- Fixing the same `options: null` gap on `gms_v87`, `gms_v92`, or `jms_v185`.
  The gap exists there too (§9) but this task is scoped to v95.1.
- Completing whole dispatcher families beyond the arms required by the cells in
  scope. Where a family trap applies it is called out, not resolved wholesale.
- Any change to already-`verified` v95 wire encoding. Per playbook §4, a wire
  divergence found on a verified version is its own commit and its own review.

## 3. User Stories

- As a server operator running a GMS v95.1 tenant, I want the cash-shop entry,
  map change, portal script, NPC conversation, storage and multi-chat opcodes
  routed so that those interactions do not silently drop on the v95 socket.
- As a server operator, I want party/guild/buddy/messenger/BBS handlers to carry
  their `operations` mode table on v95 so that a mode byte resolves to the right
  arm instead of falling through unconfigured.
- As a server operator, I want the movement writers (player, monster, pet, NPC)
  to carry their `types` table on v95 so that movement fragments encode with the
  correct type codes.
- As a packet maintainer, I want each in-scope v95 cell backed by a byte fixture
  whose every field cites a decompile line, so the matrix state is evidence, not
  assertion.
- As a packet maintainer, I want the four JMS/v12-only symbols recorded as `n-a`
  with evidence, so they stop reappearing in v95 gap sweeps.

## 4. Functional Requirements

### 4.1 W1 — v95 template routing gaps

Add the following to `template_gms_95_1.json`. Each is present in
`template_gms_83_1.json` and absent from `template_gms_95_1.json`. The opcode
MUST be derived from the v95 registry / v95 IDB, never carried over from v83.

Handlers (9): `CashShopEntryHandle`, `CharacterAutoDistributeApHandle`,
`CharacterMultiChatHandle`, `MapChangeHandle`, `NPCStartConversationHandle`,
`PartyInviteRejectHandle`, `PongHandle`, `PortalScriptHandle`,
`StorageOperationHandle`.

Writers (3): `CashShopOpen`, `NPCConversation`, `PicResult`.

Each added entry MUST carry its `opCode`, `validator` (handlers), `fname`,
`services`, and — where the v83 entry has one — a v95-correct `options` block.
An entry whose v95 opcode cannot be established from the registry or the IDB is
a blocker, not a guess (CLAUDE.md: never invent an opcode).

### 4.2 W2 — v95 cell verification

Verify each cell below per `VERIFYING_A_PACKET.md` §0–10. All nine ops are
already present in `docs/packets/registry/gms_v95.yaml`, so §1 resolves to
"verify", not "confirm n-a".

| # | Op × direction | v95 opcode | Registry fname | Atlas package (to be pinned in design) |
|---|---|---|---|---|
| 1 | `ENTER_CASHSHOP` / serverbound | 43 | `CWvsContext::SendMigrateToShopRequest` | `cash/serverbound/ShopEntry` |
| 2 | `MULTI_CHAT` / serverbound | 140 | `CUIStatusBar::SendGroupMessage` | `chat/serverbound` (`multi.go`) |
| 3 | `CHANGE_MAP` / serverbound | 41 | `CCashShop::SendTransferFieldPacket` | `field/serverbound/Change` |
| 4 | `NPC_TALK` / serverbound | 63 | `CNpc::ShowQuestList` | `npc/serverbound` |
| 5 | `CHANGE_MAP_SPECIAL` / serverbound | 112 | `CUserLocal::HandleUpKeyDown` | `portal/serverbound` |
| 6 | `STORAGE` / serverbound | 67 | `CTrunkDlg::SetRet` | `storage/serverbound/Operation` (+ arms) |
| 7 | `NPC_TALK` / clientbound | 363 | `CScriptMan::OnPacket` | `npc/clientbound/Conversation` |
| 8 | `CHECK_SPW_RESULT` / clientbound | 27 | `CLogin::OnCheckSPWResult` | `login/clientbound/PicResult` |
| 9 | `NPC_ACTION` / clientbound | 314 | `CNpc::OnMove` | `npc/clientbound/Action` |

Requirements:

- Every fixture byte cites a decompile line from the v95 IDB (playbook §5).
- Every fixture carries the `// packet-audit:verify packet=<id> version=gms_v95 ida=<0xaddr>`
  marker (playbook §6).
- Tier-1 cells pin an evidence record; tier-0 cells MUST NOT (playbook §7).
- `go run ./tools/packet-audit matrix --check` exits 0 before any cell is claimed.
- **Dispatcher-family trap (playbook §2):** rows 6 (`STORAGE` sb —
  `CTrunkDlg::SetRet` plus `SendSortItemRequest` / `SendPutItemRequest` alts,
  backed by `storage/serverbound/{operation,operation_meso,operation_retrieve_asset,operation_store_asset}.go`)
  and 7 (`NPC_TALK` cb — `CScriptMan::OnPacket`) are mode-prefix families. Each
  needs one fixture per mode, and the STATUS.md op row stays at the worst sibling
  until every arm is done. Design must decide whether these two rows are verified
  arm-complete in this task or handed to `dispatcher-family-implementer`.
- Row 3 (`CHANGE_MAP` sb) has two `fname_alts` (`CField::SendTransferFieldRequest`,
  `CITC::SendTransferFieldPacket`); the read order must be reconciled across them.

### 4.3 W3 — v95 template `options` gaps

Populate the `options` block for the following entries in `template_gms_95_1.json`.
Each is currently routed with `"options": null` and has a populated `options` on
`template_gms_83_1.json`. This is the complete set for v95 (verified by a full
sweep of all 150 handler and 239 writer entries, not a spot check).

| Symbol | Kind | v95 opCode | Missing options key |
|---|---|---|---|
| `NPCContinueConversationHandle` | handler | 0x41 | `messageType` |
| `MessengerOperationHandle` | handler | 0x8F | `operations` |
| `PartyOperationHandle` | handler | 0x91 | `operations` |
| `GuildOperationHandle` | handler | 0x95 | `operations` |
| `BuddyOperationHandle` | handler | 0x99 | `operations` |
| `GuildBBSHandle` | handler | 0xB3 | `operations` |
| `AuthPermanentBan` | writer | 0x00 | `failedReasonCodes` |
| `AuthTemporaryBan` | writer | 0x00 | `failedReasonCodes` |
| `PetMovement` | writer | 0xC9 | `types` |
| `CharacterMovement` | writer | 0xD2 | `types` |
| `MoveMonster` | writer | 0x11F | `types` |
| `NPCAction` | writer | 0x13A | `types` |

Requirements:

- Every mode value, message-type value, reason code and movement type code MUST
  be derived from the v95 client (IDB / registry / dispatcher yaml), not copied
  from the v83 table. v83→v95 mode-table drift is the exact failure this task
  exists to catch.
- `go run ./tools/packet-audit operations --check` MUST exit 0, confirming the
  tenant `operations` tables agree with `docs/packets/dispatchers/<family>.yaml`.
- `go run ./tools/packet-audit dispatcher-lint` MUST exit 0.

### 4.4 Already verified on gms_v95 — no work

Recorded here so downstream phases do not reopen them. Each resolves to a cell
whose `gms_v95` state in `status.json` is already `verified`:

| Symbol | Cell(s) | v95 state |
|---|---|---|
| `CharacterAutoDistributeApHandle` | `AUTO_DISTRIBUTE_AP` sb (98), `DISTRIBUTE_AP` sb (98) | verified |
| `PongHandle` | `PONG` sb (25), `PING` cb (17) | verified |
| `PartyInviteRejectHandle` | `PARTY_OPERATION` sb (145), `PARTY_RESULT` sb (146); `DENY_PARTY_REQUEST` is `n-a` | verified / n-a |
| `CashShopOpen` (writer) | `SET_CASH_SHOP` cb (143) | verified |
| `NPCContinueConversationHandle` | `NPC_TALK_MORE` sb (65) | verified |
| `MessengerOperationHandle` | `MESSENGER` sb (143) | verified |
| `PartyOperationHandle` | `PARTY_OPERATION` sb (145) | verified |
| `GuildOperationHandle` | `GUILD_OPERATION` sb (149) | verified |
| `BuddyOperationHandle` | `BUDDYLIST_MODIFY` sb (153) | verified |
| `GuildBBSHandle` | `BBS_OPERATION` sb (179) | verified |
| `AuthPermanentBan`, `AuthTemporaryBan` | `LOGIN_STATUS` cb | verified |
| `PetMovement` | `MOVE_PET` cb (201) | verified |
| `CharacterMovement` | `MOVE_PLAYER` cb (210) | verified |
| `MoveMonster` | `MOVE_MONSTER` cb (287) | verified |

Note that "verified codec" and "correctly routed/configured template" are
independent. Several rows above are verified *and* appear in W1 or W3 — the codec
is proven, the v95 tenant wiring is not.

### 4.5 W4 — prove and record `n-a`

The following have no GMS v95 counterpart. Per playbook §1, the job is to confirm
`n-a`: verify no v95 template routes the opcode, no Atlas struct claims it, and
the v95 registry has no entry — then record the result. Where no matrix row
exists, one must be created so the `n-a` is durable rather than an absence.

| Symbol | Current reach | Action |
|---|---|---|
| `CreateSecurityHandle` | `template_jms_185_1` 0x1A only; no `fname`; no matrix row | prove + record `n-a` on gms_v95 |
| `LoginAuth` (writer) | `template_jms_185_1` 0x18; row `LOGIN_AUTH` cb exists, v95 opcode `null` | prove + record `n-a` on gms_v95 |
| `WorldSelectHandle` | `template_gms_12_1` 0x03 only; no `fname`; no matrix row | prove + record `n-a` on gms_v95 |
| `ServerLoad` (writer) | `template_gms_12_1` 0x02 only; no `fname`; no matrix row | prove + record `n-a` on gms_v95 |

`go run ./tools/packet-audit na-consistency` (or the equivalent `--check`) MUST
exit 0 after the records land.

## 5. API Surface

No HTTP/JSON:API surface changes. The externally visible contract changed by this
task is the tenant socket configuration payload
(`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`),
consumed by `atlas-channel` and `atlas-login` at socket-handler resolution time.
Added handler entries follow the existing shape:

```json
{ "opCode": "0x..", "validator": "<Validator>", "handler": "<Name>Handle",
  "fname": "<CClass::Method>", "options": { ... }, "services": ["channel"] }
```

Added writer entries follow:

```json
{ "opCode": "0x..", "writer": "<Name>", "fname": "<CClass::Method>",
  "options": { ... }, "services": ["channel"] }
```

## 6. Data Model

No database entities, migrations, or `tenant_id` scoping changes. The artifacts
written are:

- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` (W1, W3)
- `libs/atlas-packet/**/*_test.go` byte fixtures with `packet-audit:verify` markers (W2)
- `docs/packets/evidence/gms_v95/*.yaml` evidence records (W2 tier-1, W4)
- `docs/packets/audits/status.json` + `docs/packets/audits/STATUS.md` (regenerated)
- Any wire fix to `libs/atlas-packet/**` required by playbook §4 — separate commit.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-configurations` | v95 seed template gains 9 handlers + 3 writers (W1) and 12 populated `options` blocks (W3). |
| `atlas-channel` | No code change expected; newly routed handlers already exist in `socket/handler/`. A handler missing a Go implementation is a blocker to surface, not to stub. |
| `atlas-login` | Affected only if `CHECK_SPW_RESULT` / `PicResult` verification exposes a wire divergence in `libs/atlas-packet/login/clientbound/pic_result.go`. |
| `libs/atlas-packet` | New byte-fixture tests for 9 cells; wire fixes only where a divergence is proven against the v95 IDB. |
| `tools/packet-audit` | No change; used as the gate (`matrix --check`, `operations --check`, `dispatcher-lint`, `na-consistency`). |

## 8. Non-Functional Requirements

- **Evidence discipline:** no fixture byte, opcode, mode value, or type code may
  originate from remembered MapleStory knowledge or from another version's table.
  The v95 IDB and v95 registry are the sources of truth.
- **No regression on verified versions:** no wire change may alter the encoding
  of an already-verified version. Version-gate divergent fields with the
  `MajorAtLeast` idiom, never a raw `> N` comparison.
- **Gate:** flagless `tools/verify.sh` exits 0 before the branch is called done.
  `packet-audit matrix --check`, `operations --check`, `dispatcher-lint`, and
  `fname-doc --check` all exit 0.
- **Review:** code review is a required gate before PR. `packet-completeness-critic`
  runs against this task's `coverage-manifest.yaml` alongside the guideline reviewers.
- **Line endings preserved** on the template JSON; no CRLF→LF normalization.
- **Multi-tenancy:** template changes are per-template (v95.1) and must not alter
  any other version's template.

## 9. Open Questions

1. **Family scope (W2 rows 6 & 7).** `STORAGE` serverbound and `NPC_TALK`
   clientbound are mode-prefix dispatcher families. Does this task verify every
   arm (arm-complete, likely a `dispatcher-family-implementer` dispatch per
   family), or verify the primary arm and leave the STATUS.md row at the worst
   sibling? Design must decide; it materially changes the size of the task.
2. **v87 / v92 / jms_v185 `options` gap.** The same `options: null` condition
   exists on other columns — `gms_v87` has 16/145 handlers with options,
   `gms_v92` 13/89, versus `gms_v83` 23/152. Out of scope here; should a
   follow-up task be filed?
3. **W1 routing without a Go handler.** If any of the 9 handlers or 3 writers has
   no corresponding implementation reachable from `atlas-channel`'s `main.go`
   registration for v95, routing it in the template is inert. Surface per symbol
   during design.
4. **`CHANGE_MAP` alt reconciliation.** Whether `CCashShop::SendTransferFieldPacket`,
   `CField::SendTransferFieldRequest`, and `CITC::SendTransferFieldPacket` share
   one read order on v95, or need distinct fixtures.
5. **W4 matrix rows.** Creating new matrix rows solely to hold an `n-a` verdict
   for `CreateSecurityHandle`, `WorldSelectHandle`, and `ServerLoad` — is that
   the right mechanism, or should `docs/packets/audits/feature-na-evidence.yaml`
   carry them instead?

## 10. Acceptance Criteria

- [ ] All 9 handlers and 3 writers in §4.1 are present in `template_gms_95_1.json`
      with v95-derived opcodes; no opcode copied from another version without
      IDB/registry confirmation.
- [ ] All 12 entries in §4.3 carry a populated `options` block whose values are
      derived from the v95 client.
- [ ] All 9 cells in §4.2 report `verified` for `gms_v95` in `status.json`, each
      backed by a byte fixture carrying a `packet-audit:verify` marker and
      per-field decompile citations.
- [ ] Tier-1 cells among the 9 have a pinned evidence record under
      `docs/packets/evidence/gms_v95/`; tier-0 cells have none.
- [ ] All 4 symbols in §4.5 are recorded `n-a` for `gms_v95` with evidence.
- [ ] No cell listed in §4.4 changed state.
- [ ] No non-v95 template file is modified, and no already-verified version's
      wire encoding changed (or, if one did, it landed as its own reviewed commit).
- [ ] `go run ./tools/packet-audit matrix --check` exits 0.
- [ ] `go run ./tools/packet-audit operations --check` exits 0.
- [ ] `go run ./tools/packet-audit dispatcher-lint` exits 0.
- [ ] `go run ./tools/packet-audit fname-doc --check` exits 0.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `docs/tasks/task-146-v95-packet-verification-batch/coverage-manifest.yaml`
      exists and `packet-completeness-critic` reports no CHANGED-BUT-UNCLAIMED or
      CLAIMED-BUT-UNVERIFIED findings.
- [ ] Code review completed before the PR is opened.
