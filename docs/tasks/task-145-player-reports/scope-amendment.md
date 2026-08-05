# Task 145 — Scope Amendment: gms_92 + jms_185 report support

**Decided:** 2026-08-04, by the user, at the start of the execute phase (option C).
**Supersedes:** `plan.md` Global Constraints "Scope exclusions" bullet and Task 25
Step 1's gms-92 / jms deferral entries.

## Why the plan's exclusion rationale was wrong

`plan.md` defers gms_92 and jms as *"blocked on registry files + IDBs (opcodes
unverifiable)"*. Verified against the live IDA-MCP sessions and the repo on
2026-08-04, that rationale does not hold:

- **gms_92** — the IDB (`GMS_v92_1_DEVM.exe.i64`) carries full mangled MSVC
  symbols for all five ops: `CWvsContext::OnClaimResult` `0x9cf310`,
  `OnSetClaimSvrAvailableTime` `0x9c5d30`, `OnClaimSvrStatusChanged` `0x9c5d60`,
  `OnSueCharacterResult` `0x9cf950`, `SendClaimRequest` `0x9d9c30`. Both ops CSVs
  already have a `GMS v92` column. Nothing is unverifiable; what is missing is the
  *column infrastructure* (registry yaml, IDA export, audit dir, `matrix.VersionKeys`
  entry).
- **jms_185** — is already a matrix column with a registry yaml carrying opcodes:
  `CLAIM_RESULT` `0x2A`, `CLAIM_AVAILABLE_TIME` `0x2B`, `CLAIM_STATUS_CHANGED`
  `0x2C` (all three also named in the JMS IDB), and `CLAIM_REQUEST` `0x65`
  (`provenance: csv-import`, **not** named in the IDB). Sue has no jms registry row
  and no named IDB function.

## Amended scope

Tasks 1–17 and 19–23c execute unchanged. The following are added or amended.

### New Task 26 — gms_92 registry

- `packet-audit registry seed` from both ops CSVs → `docs/packets/registry/gms_v92.yaml`.
- `discover-ops` against IDB session for `GMS_v92_1_DEVM.exe.i64`, with the
  dispatcher curation checklist from `STARTING_A_NEW_VERSION_PASS.md` §1.1, then
  `--apply`; worklist committed as `docs/packets/registry/discover_gms_v92.md`.
- **Tooling prerequisite:** `discover-ops` currently exposes only `-ida-port`, and
  port-based instance selection is dead (task-138). Add `-ida-database` to
  `discover_ops.go` (and `verify_serverbound.go`) mirroring `runExport`'s flag at
  `tools/packet-audit/cmd/root.go:133`. Produce this fix; do not work around it.

### New Task 27 — gms_92 IDA export

- Bootstrap the roster from `docs/packets/ida-exports/gms_v95.json` (nearest
  version; the v92 IDB was named from v95), purge cross-IDB coincidentals.
- `packet-audit export --version gms_v92 --ida-database <session> --output
  docs/packets/ida-exports/gms_v92.json`. Smoke-test a small roster first.

### New Task 28 — gms_92 matrix column

- Add `gms_v92` to `matrix.VersionKeys` (positioned between `gms_v87` and
  `gms_v95`) and `shortLabels` in `tools/packet-audit/internal/matrix/render.go:12`.
- Update the `doclint` facts doc (`version_count` 9 → 10, `version_keys`) and
  `docs/packets/PROCESS.md`'s version set.
- Run the static audit pass against `template_gms_92_1.json` → `docs/packets/audits/gms_v92/`.
- Regenerate the matrix; `matrix --check`, `fname-doc --check`, `doclint` clean.

### New Task 29 — jms_185 `CLAIM_REQUEST` send-site

- Name the jms serverbound claim send-site in the JMS IDB and `idb_save`, the same
  procedure as Task 23 does for v83. The registry's `0x65` is `csv-import` and
  unconfirmed — confirm or correct it from the IDB.
- **If no candidate decompiles to the expected shape, STOP AND ASK.** An
  unresolvable fname is never substituted or hashed (CLAUDE.md; `plan.md` Task 23).

### New Task 30 — jms sue absence

- Record sue-on-jms as a verified absence using the Task 23c evidence format, or —
  if a send-site/handler is found — a registry correction. Do not leave it implicit.

### Amended Task 18 — seed templates

Add sue/claim entries to `template_gms_92_1.json` (all five ops) and claim-only
entries to `template_jms_185_1.json` (sue omitted unless Task 30 finds it), at
their sorted `opCode` positions. `gms_12`, `gms_48` still get none; `gms_61` stays
sue-only. Opcodes come from the Task 26/29 registry work, never ported.

### Amended Task 24 — verification campaign

31 cells → **41**: +5 gms_92 (4 clientbound + `CLAIM_REQUEST`), +5 jms_185
(3 clientbound claim + `CLAIM_REQUEST` + whatever Task 30 resolves). jms
`SUE_CHARACTER_RESULT` promotes only if Task 30 finds it; otherwise it stays ⬜
with a recorded absence.

### Amended Task 25 — deferrals

Delete the gms-92 and jms deferral entries (now in scope). `gms_12` remains
deferred, with the **corrected** rationale: no registry yaml and no matrix column —
not "unverifiable". Do not restate the false IDB claim.

## Amendment 2 — chat-history endpoint IS routed through the ingress

**Decided:** 2026-08-05, by the user, during Task 12 review.
**Supersedes:** `plan.md` / `context.md`'s "the messages chat-history endpoint gets NO
ingress entry (server-to-server only)".

### Why the plan's design did not work

`RootUrl(domain)` (`libs/atlas-rest/requests/url.go:14-19`) returns `<DOMAIN>_SERVICE_URL`
when set and otherwise falls back to `BASE_SERVICE_URL`. There are **zero** `*_SERVICE_URL`
overrides in `deploy/k8s/base` — the entire fleet relies on the fallback, and
`BASE_SERVICE_URL` points at `atlas-ingress`. With no ingress route for `/api/chat/`,
atlas-ban's transcript fetch would 404, and Task 7's best-effort degradation would swallow
it and write a **null transcript on every report** — silently, with no error, no failing
test, and no CI signal.

### Decision

Add `/api/chat/history` to `deploy/shared/routes.conf` (and `deploy/compose/routes.conf`),
proxying to `atlas-messages:8080`, alongside the `/api/reports` route. Task 19 owns both.

### Accepted risk (explicit)

`routes.conf` is a flat nginx proxy map with no auth, allow/deny, or internal-only
markers, and `atlas-ingress` is exposed via traefik at `dev.atlas.home`. Routing this
endpoint therefore makes captured chat — **including whispers** — reachable by anything
that can reach that host. This was raised at decision time and accepted on the basis that
authentication is being added to the API surface generally in the near term; this endpoint
is not a special case and should be covered by that work.

Task 25's documentation must state this exposure plainly rather than describing the
endpoint as server-to-server only.

## Execution order

Tasks 1–17 → 26 → 27 → 28 → 29 → 30 → 18 → 19 → 20–23c → 24 → 25.

The bring-up lands before Task 18 so templates and the verification campaign see
`gms_v92` as a real column rather than needing a second pass.

## Risk accepted

A version bring-up on a feature branch is broader than this task's original
charter; it touches `packet-audit` internals and STATUS.md wholesale, so the
branch diff will be large and the final review correspondingly broad. This was
raised at decision time and accepted.

## Amendment 3 — Task 28 turned out to require wiring `template_gms_92_1.json`, not just registering the column

**Decided:** 2026-08-05, by the coordinator, mid-execution of Task 28.
**Supersedes:** Task 28's brief, which scoped only `matrix.VersionKeys` +
`shortLabels` + the doclint facts + a static-audit-pass regen, and assumed
`matrix --check` would come up clean from that alone.

### Why the brief's assumption was wrong

`template_gms_92_1.json` predates this task's registry/export work (history
back to `b8f97493e`) and was a **stub**: 47 handlers / 65 writers, versus
115–129 / 160–215 for every neighboring version (v61 115/153, v72 124/160,
v87 122/207, v95 129/215). It routed only login/char-select/movement/chat/
cash-shop-entry — nothing else. Flipping on the `gms_v92` matrix column
(Task 28 as briefed) exposed this as **67 hard-gate conflicts** in
`matrix --check` (a CI-blocking gate per `PROCESS.md`'s
`matrix_check_hard_gate: true`), none of them the five task-145 sue/claim
ops. Per `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` §1.2, this is
exactly what a real version bring-up's template must satisfy
("ops the registry marks present should be routed") — the brief just didn't
anticipate the template needed that level of work.

The coordinator directed completing the wiring rather than reverting or
descoping, citing CLAUDE.md's "No Deferring Producible Work" (an unrouted
template is exactly the "produce it yourself" case) and that `matrix --check`
being a CI gate makes this non-optional for the branch to merge at all.

### What got wired (Class A — 59 template-wiring-gap conflicts)

Each: opcode from `docs/packets/registry/gms_v92.yaml` (Task 26, the source
of truth — never copied from a neighbor's own opcode), identifier/validator/
services cross-checked for agreement between `template_gms_87_1.json` and
`template_gms_95_1.json`. Six needed extra care because the two references
disagreed or were incomplete, each resolved with independent evidence, not a
coin-flip:

- `MOVE_SUMMON` (serverbound `SummonMoveHandle`): v87's entry carries an
  `options.types` movement-table, v95's doesn't. Read
  `libs/atlas-packet/summon/serverbound/move.go` — `Decode` treats the move
  blob as an opaque byte-faithful rebroadcast and never touches `options` at
  all. The table is dead configuration; wired without it (matches v95).
- `SUMMON_SKILL` / `DAMAGE_SUMMON` (clientbound `SummonSkill`/`SummonDamage`):
  v87's own template has these two writer names **swapped** relative to
  v87's own registry opcodes (self-inconsistent — a pre-existing v87 bug,
  out of scope here, not touched). v95 is self-consistent, and its fname↔name
  pairing matches v92's own registry fnames (`OnSkill`→SummonSkill,
  `OnHit`→SummonDamage) exactly. Wired per v95.
- `CHATTEXT1` (`ChatGeneralChat`) / `LOCK_UI` (`UiLock`): v87 has no entry at
  all for either op; v95 is the only reference. Wired per v95 (no
  disagreement to adjudicate, just one source).
- `STORAGE` (`StorageOperation`'s `options.operations.ERROR_MESSAGE`): v87/v84/v83=23,
  v95=24 — a genuine, IDA-documented per-version shift (see
  `docs/packets/dispatchers/storage_operation.yaml`'s existing v83/v95
  correction note). Decompiled v92's `CTrunkDlg::OnPacket` directly
  (0x74b620): case 24 is the only arm calling `Decode1`+`DecodeStr` (the
  dynamic-string error read); case 23 is a fixed-string notice. v92 already
  made the v95-side shift: **24**, added to `storage_operation.yaml`'s
  `gms_v92` column with the citation. `MESSENGER`'s table (stable 0–7 across
  every other version) was independently re-confirmed via v92's
  `CUIMessenger::OnPacket` decompile (0x7d5600) before adding its `gms_v92`
  column to `messenger_operation.yaml` — both dispatcher-family YAMLs now
  drive their v92 template entries through `packet-audit operations`'s
  source of truth rather than a hand-copied table.

### Pre-existing bug found and fixed (unrelated to task-145, discovered because it collided with the wiring)

The template's **existing** (pre-Task-28) handler entries at opcodes
`0xC8`/`0xC9`/`0xCA` were named `SummonMoveHandle`/`SummonAttackHandle`/
`SummonDamageHandle` — but `docs/packets/registry/gms_v92.yaml` says those
three opcodes are `PET_AUTO_POT`/`PET_EXCLUDE_ITEMS`/`UNNAMED_R288`, not
summon ops at all (confirmed: v87 and v95 both route `PetItemUseHandle`/
`PetItemExcludeHandle` at their own `PET_AUTO_POT`/`PET_EXCLUDE_ITEMS`
opcodes). The **real** `MOVE_SUMMON`/`SUMMON_ATTACK`/`DAMAGE_SUMMON` opcodes
are `0xCC`/`0xCD`/`0xCE`. Same pattern on the writer side at `0xC2`–`0xC7`
(`SummonSpawn`/`SummonRemove`/`SummonMove`/`SummonAttack`/`SummonDamage`/
`SummonSkill` sitting on what registry says are
`SHOW_RECOVERY_UPGRADE_COUNT_EFFECT`/`SPAWN_PET`/`EVOLVE_PET`/(unconfirmed)/
`MOVE_PET`/`PET_CHAT`). **This means the live v92 tenant, as configured today,
misdecodes a v92 client's pet auto-pot and pet-exclude-items requests as
summon-move/summon-attack commands** (and the writer side would emit
summon-family payloads under opcodes the client reads as pet/recovery
features) — a real, active bug, not a hypothetical. Fixed in the template:
`0xC8`→`PetItemUseHandle`, `0xC9`→`PetItemExcludeHandle`, the unconfirmed
`0xCA` (`UNNAMED_R288`) left unrouted rather than invented; writers `0xC3`→
`PetActivated` (+ its stable operations table), `0xC6`→`PetMovement`, `0xC7`→
`PetChat`, `0xC2`/`0xC4`/`0xC5` removed (no Atlas writer exists for
`SHOW_RECOVERY_UPGRADE_COUNT_EFFECT`/`EVOLVE_PET`, and the registry has no op
at all for `0xC5` — nothing to route there); the real summon opcodes
(`0xCB`–`0xD0`) added correctly.

### Class B — 8 registry-vs-audit-report conflicts, all resolved with direct v92 IDA evidence

Every one of these turned out to be a genuine v92-registry gap (not a false
audit report), confirmed by decompiling the actual v92 IDB
(`GMS_v92_1_DEVM.exe.i64`, session `acdfccff` at investigation time — always
re-resolve via `idb_list`, never assume) rather than trusting either the
registry's silence or the ops CSV, which is itself stale/incomplete for
several of these (verified column-by-column against a known-good row,
`CLAIM_RESULT`, to rule out a transcription error before trusting any CSV
"0x000"):

| op | direction | real v92 opcode | evidence |
|---|---|---|---|
| `FOOTHOLD_INFO` | clientbound | 175 (`0xAF`) | `CField::OnPacket` (0x5406B0) case 175 → `CField::OnFootHoldInfo`; CSV confirms 0x0AF once read correctly. Follows the v87/v95 task-096 "shared op label, different opcode per version" precedent — added as a second (clientbound) entry under the existing `FOOTHOLD_INFO` op name, template writer `FootholdInfo`. |
| (unnamed, `CField::OnStalkResult`) | clientbound | 171 (`0xAB`) | Same dispatcher, case 171. This fname was never given a canonical op name in the CSV (blank Op column in every version) — v79/v83/v87/v95 each carry their own opcode-derived `IDA_0X<hex>` placeholder; added v92's own, `IDA_0X0AB`, following that established convention. Template writer `StalkResult` (matches v87/v95's identifier). |
| `USE_SHOP_SCANNER_ITEM` | serverbound | 90 (`0x5A`) | `CWvsContext::SendShopScannerItemUseRequest` (0x9B6050) constructs `COutPacket(0x5Au)`. Matches gms_v95's own opcode for the same fname exactly. Template handler `ShopScannerItemUseHandle` (from v95). |
| `ITC_QUERY_CASH_REQUEST` | serverbound | 297 (`0x129`) | `CITC::TrySendQueryCashRequest` (0x56C2A0) constructs `COutPacket(0x129u)`; renamed from `sub_56C2A0` in the IDB and saved. Distinct from `CHECK_CASH`/`CCashShop::TrySendQueryCashRequest` (different class, different op). Template handler `ItcQueryCashRequestHandle` (from v95). |
| `WITCH_TOWER_SCORE_UPDATE` | clientbound | 350 (`0x15E`) | `CField_Witchtower::OnPacket` (0x55D7C0, a `CField::OnPacket` virtual override) does `if (a3==350) OnScoreUpdate(...) else CField::OnPacket(...)`. Fits cleanly between v87's IDA-verified 318 and v95's 360. v95's own registry entry for this op is flagged pending-follow-up there (fname ambiguity) — v92's entry is independently verified and does not inherit that. Template writer `AriantScore` (`libs/atlas-packet/field/clientbound/ariant_score.go`). **Correction (post-review):** this codec was NOT previously unrouted — v95 already routes it, at `opCode "0x166"` (358, `ARIANT_SCORE`'s own opcode there), since commit `0564037e4`, long predating this branch; v83/v87 still don't route it. So v95 wires `AriantScore` under `ARIANT_SCORE` (358), and this change wires the SAME codec under v92's `WITCH_TOWER_SCORE_UPDATE` (350) — a second, distinct routing of an already-shared codec, not a first-ever one. Same underlying decision either way (v92's opcode is IDA-verified at 350 regardless of what any other version routes), just described accurately: wiring it for v92 doesn't retroactively cover v83/v87, and doesn't newly cover v95 either — v95 already had it. |

All five added to `docs/packets/registry/gms_v92.yaml` as `provenance: manual`
with an `ida:` address citation, per the un-named-fname / no-invented-value
discipline.

### fname-doc side effect (tool fix, not data)

`packet-audit fname-doc --check` broke after the static audit pass: v92 is
the **only** version to ever produce an audit report for
`CashItemUseVegaScroll` (a `packet-audit:fname
CWvsContext::SendConsumeCashItemUseRequest#VegaScroll` dispatcher arm,
task-130, IDA-verified per-version), and that report's row 0 carries
`diff.VerdictUnresolved` (🚫 — the export's own `unresolved: true,
calls:[{op:Unresolved}]` marker for this fname, expected per the Task 27
brief). `fnamedoc.go`'s `loadReportFNames` had no concept of "this specific
report never actually resolved the IDA side" and would have overwritten the
correct arm-suffixed comment with the weaker, unsuffixed one. Fixed in
`tools/packet-audit/cmd/fnamedoc.go`: `loadReportFNames` now skips any report
carrying a `VerdictUnresolved` row (general fix, not scoped to v92 — any
version's under-resolved report would hit the same problem); `gms_v92` was
also moved to the end of `fnamedocOrder` (lower priority) since its export
is disproportionately unresolved-heavy relative to the other versions.

### A second pre-existing bug, same pattern: `DeleteCharacterHandle` at the wrong opcode (found AND fixed)

Cross-referencing the template against the registry to locate the pet/summon
collision also surfaced `template_gms_92_1.json` handler `0x17`
(`DeleteCharacterHandle`, login service) as suspicious: registry says `0x17`
is `CREATE_CHAR_IN_CS`, and `DELETE_CHAR`'s registered v92 opcode is `0x18`.
Initially recorded as a flagged-but-unverified suspicion; **settled with
direct v92 IDA evidence, the same way as the Class B conflicts**, rather than
left as a follow-up:

- `CLogin::SendDeleteCharPacket` (0x5cb860) — the real client send site —
  constructs `COutPacket(0x18u)`. Unambiguous: **`DELETE_CHAR`'s real v92
  opcode is 24 (`0x18`)**, matching the registry and matching v95's own
  opcode (v92 already made the same +1 shift v95 made relative to
  v83/v84/v87, the same pattern as `SUE_CHARACTER_RESULT` and `STORAGE`'s
  `ERROR_MESSAGE`).
- `CLogin::SendNewCharPacket` (0x5ce1e0) — the `CREATE_CHAR_IN_CS` /
  `CREATE_CHARACTER` send site — branches on job type: the primary path
  constructs `COutPacket(0x17u)` (`CREATE_CHAR_IN_CS`, matching the
  registry's opcode 23 exactly), the alternate path constructs
  `COutPacket(0x16u)` (already correctly routed as `CreateCharacterHandle`).
  No version's template (v83/v84/v87/v95 all checked) has EVER routed
  `CREATE_CHAR_IN_CS` — Atlas has no handler implementation for that op in
  any version, so `0x17` is correctly left unrouted for v92 too (not
  invented).

Fixed: `template_gms_92_1.json` handler `0x17`/`DeleteCharacterHandle`
removed, `DeleteCharacterHandle` added at `0x18` (`LoggedInValidator`,
`services: ["login"]`, matching v95's entry exactly).

**Honesty check on how this was found vs. how `matrix` graded it**: unlike
the 59 Class A conflicts, this one was never a `matrix --check` conflict —
before and after the fix, `gms_v92`'s `DELETE_CHAR` cell both grade `partial`
/ `🟡ᶠ` ("tier-1: needs byte-fixture test to verify"), confirmed by diffing
`status.json` at the pre-fix commit (`de0e4b2f5`) against post-fix. Tier-1
ops with a linked `packet:` path apparently don't route through the same
`!routed && routedElsewhere → Conflict` check the 59 Class A ops did (an
existing tool gap, out of scope to chase down here — not something this fix
needed to touch). So `matrix --check` staying green before AND after is
expected, not evidence the fix was unnecessary: it's the same class of
"present but wrong" bug the pet/summon handlers had, just one the automated
conflict scan doesn't structurally catch for tier-1/linked-packet ops. The
correctness claim rests entirely on the direct v92 decompile evidence above,
not on a tool-reported diff — which is exactly why the coordinator asked for
decompile evidence rather than accepting the matrix's silence as clearance.
`matrix --check` / `operations --check` / `fname-doc --check` /
`doc-freshness --check` / `template-opcode-order-guard.sh` all exit clean
after this change (as they did before it).

**Live-tenant implication**: this is `atlas-login`, a different service
from the pet/summon bug (`atlas-channel`), but same class of impact — the
live v92 tenant (see below) is decoding real character-deletion requests
(opcode `0x18`) as whatever `0x18` meant before this fix (nothing was routed
there — the packet would have been silently dropped/unhandled) while
routing a should-be-unrouted `CREATE_CHAR_IN_CS` opcode (`0x17`) to the
delete-character handler instead. **Character deletion is very likely
non-functional on the live v92 tenant today**, and the live send path for
`CREATE_CHAR_IN_CS` (job-type-dependent) hits a handler that decodes the
wrong wire shape.

### Live tenant

A `GMS v92` tenant **does exist** in the `atlas-main` environment
(`atlas-tenants` id `db1dbfb3-4345-4731-9223-c40b0c7f6457`, confirmed via
`GET /api/tenants` against the live pod) — this is not a hypothetical. Seed
templates apply only at tenant creation, so **none of this task's template
fixes reach that tenant automatically.** A live-config PATCH + channel/login
restart is needed to pick them up. Precisely what it needs to cover, in
priority order:

1. **Pet/summon misdecode (`atlas-channel`, highest urgency)** —
   `0xC8`/`0xC9` (handlers) and `0xC3`/`0xC6`/`0xC7` (writers) corrected from
   `Summon*` to `Pet*`; real summon opcodes moved to `0xCB`–`0xD0`.
   **User-visible effect today, un-patched**: a v92 player using pet
   auto-HP/MP-potion or the pet item-exclude-list feature has their request
   server-side misdecoded as a `SummonMoveHandle`/`SummonAttackHandle`
   packet — wrong struct layout read from the same bytes, so the server
   either processes garbage as a summon-move/attack command (silently wrong
   behavior, possibly moving/attacking with a summon the player doesn't have
   active) or errors out decoding malformed fields. Separately, if the
   server ever emits a genuine summon-family clientbound packet on v92
   today, it goes out on `SummonSpawn`/`SummonMove`/etc.'s pre-fix opcodes
   (`0xC2`/`0xC4`/`0xC5`), which the real v92 client reads as
   `SHOW_RECOVERY_UPGRADE_COUNT_EFFECT`/`EVOLVE_PET`/nothing — the client
   renders the wrong effect or ignores the packet outright, and the player
   never sees their summon appear/move/attack correctly.
2. **Character deletion (`atlas-login`)** — `0x17`→`0x18` fix above.
   **User-visible effect today, un-patched**: a v92 player's delete-character
   request (real client opcode `0x18`) has no handler at all (dropped —
   deletion silently does nothing), while whatever currently exists at the
   live tenant's `0x17` slot (`DeleteCharacterHandle`, pre-fix) fires
   instead if the client ever sends `CREATE_CHAR_IN_CS`'s opcode — a
   character-creation request read as a character-deletion request.
3. **The 59 Class A + 5 Class B newly-wired ops** — mostly previously-silent
   drops (unrouted opcodes are simply ignored, not misdecoded), lower
   urgency than 1–2 but still real functional gaps (chat, skills, reactors,
   monster carnival, storage, messenger, etc. — see the Class A table above
   for the full op list — currently do nothing on v92).

The patch itself was not attempted here — it is a live-system change and
the coordinator is surfacing it to the user directly, per instruction.

### Task 18 overlap

None of the five sue/claim ops were touched by this amendment's wiring —
all five still grade `❌`/`⬜` with **no audit report** in v92 (not a
conflict; see Task 28's brief for why), so Task 18 lands its five entries
into an already-populated template with no risk of a duplicate opcode.
