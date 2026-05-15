# Misc-Domain Packet Audit — Design

Version: v1
Status: Proposed
Created: 2026-05-15
PRD: `prd.md`
Prior art:
- `../task-027-atlas-packet-v95-audit/{design,plan,post-phase-b}.md` (login domain — pipeline shipped)
- `../task-028-character-domain-audit/{design,plan,post-phase-b}.md` (character domain — pipeline scaled, `EncodeForeign` registry, cycle guard, suffix-taint walker, ack pattern)
- `../../../../task-065-combat-domain-audit/`, `../../../../task-066-social-domain-packet-audit/`, `../../../../task-067-commerce-domain-packet-audit/`, `../../../../task-068-world-domain-packet-audit/` (sibling audits in flight) — design.md files used as the structural template for this doc

---

## 1. Design Goals

Sixth (and final scoped) application of the audit pipeline. By the time task-069 lands the analyzer, the `EncodeForeign` registry, the cycle guard, the suffix-taint walker, the 4-variant `pt.Variants` test pattern, the audit-report writer, the per-version IDA-export format, and the ack-footer convention are all shipped. Nothing in this task should re-derive any of those.

What this task uniquely owns:

- **Sweep the long tail.** Account, fame, stat, ui, socket, channel, merchant, tool, quest. Nine top-level directories with very few packets each — the sum of misc-domain packet files actually present in the tree is ~21, not the PRD's speculative ~49 (see §3 for the corrected inventory). Each one is small enough that the per-file effort is the floor (open the file, run the analyzer, write the report, gate the fix, write the test); the dominant cost is *file count* and *cross-version repetition*, not per-file complexity.
- **Confirm 100% coverage of `libs/atlas-packet/`.** Tasks 027 + 028 + 065 + 066 + 067 + 068 + 069 should account for every top-level directory under `libs/atlas-packet/`. The PRD §4.10 requires a post-audit sweep against `find libs/atlas-packet -maxdepth 1 -type d`. If any directory is unclaimed, document it in TOTAL.md's gaps section with a defer/new-task/out-of-scope recommendation.
- **Produce TOTAL.md.** A single canonical ledger that says: for every domain directory, which task owns it, what its v95 verdict roll-up is, and what's deferred. This is the only design choice in the task with no prior precedent — the other six tasks each shipped a per-domain `SUMMARY.md` slice; nobody has yet stitched the slices into one document. §10 covers the TOTAL.md design choices.
- **Watch socket and quest specifically.** Socket handshake is the most operationally sensitive surface in the entire library (a one-byte error breaks every client connection). Quest has cross-task overlap with task-014, task-015, task-023 — gate widening/narrowing there is a real footgun.

Constraints inherited from the prior tasks (do not re-relitigate):

- **Don't extend the analyzer.** If a verdict's only escape route is an analyzer change, mark `❌ tool-limitation` and footer it. Same rule as task-028 §3, task-068 §3.
- **2-deep nesting cap.** No file in the misc domain gets a 3-deep exception. The PRD's only 3-deep carve-out was `set_field.go` (world domain, task-068). All misc-domain encoders cap at 2-deep nested guards.
- **Bare-handler exclusion stays.** No descent into `services/atlas-channel/`, `services/atlas-account/`, `services/atlas-quest/` decoder code. Bare handlers with no `libs/atlas-packet` decoder defer to `_pending.md` with an explicit row each.
- **Prior verdict regression baseline.** Phase 0 re-runs every prior task's SUMMARY rows. Any drift on login (28 rows), character (52 rows), or any of the 065-068 rows triggers a STOP-and-investigate.

---

## 2. Architecture Overview

No new architecture. Data flow is identical to task-027 §2 / task-028 §2 / task-068 §2:

```
CSV ─→ template ─→ IDA source ─→ atlas-packet analyzer ─→ diff engine ─→ report writer
                                                ↑
                                                │
                                          TypeRegistry
```

What changes for this task is **what the pieces ingest**:

| Piece                | task-068 input                                                    | task-069 input                                                              |
|----------------------|-------------------------------------------------------------------|-----------------------------------------------------------------------------|
| Atlas source         | `libs/atlas-packet/{field,portal,npc}/`                           | `libs/atlas-packet/{account,fame,stat,ui,socket,channel,merchant,quest}/`   |
| IDA exports          | `gms_v95.json` (world, appended)                                  | `gms_v95.json` (misc, appended)                                             |
| IDA exports (cross)  | `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` (world)        | Same three files, append misc FNames after v95 pass                         |
| Templates            | World-domain opcodes/sub-ops                                      | Misc-domain opcodes/sub-ops (NoticeWriter, ChangeChannelWriter, etc.)       |
| TypeRegistry         | `SetField` map-header, NPC conversation cluster, etc.             | Likely additions per §5 (quest reward sub-struct, socket version-info block, channel migrate address, merchant operation modes) |
| Analyzer             | Untouched                                                         | Untouched                                                                   |
| Audit-report folder  | flat `docs/packets/audits/gms_v95/`                               | flat `docs/packets/audits/gms_v95/` (per task-068 §10 decision)             |
| Cross-task ledger    | N/A                                                               | **NEW**: `docs/packets/audits/gms_v95/TOTAL.md`                             |

The audit pipeline is **read-only** against `libs/atlas-packet/`; the only writes are to `tools/packet-audit/internal/atlaspacket/registry.go` (+test), `libs/atlas-packet/{account,fame,stat,ui,socket,channel,merchant,quest}/` (wire-bug fixes), `services/atlas-configurations/seed-data/templates/template_*.json` (opcode/enum fixes), and `docs/packets/` (audit reports + IDA exports + TOTAL.md).

---

## 3. The hard part #1: inventory correction and scope confirmation

The PRD §4.1/§4.2 was written from MapleStory-knowledge memory, not from a tree walk. A fresh enumeration (`find libs/atlas-packet/<domain> -name '*.go' ! -name '*_test.go'`) gives the real count:

| Domain     | PRD count | Actual files (non-test)                                                                                                                                                                                                                                  | Real count |
|------------|-----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------|
| account    | 6         | `account/serverbound/{accept_tos,register_pin,set_gender}.go`                                                                                                                                                                                            | **3**      |
| fame       | 4         | `fame/clientbound/response.go`, `fame/response_body.go`, `fame/serverbound/change.go`                                                                                                                                                                    | **2 packets + 1 body** |
| stat       | 2         | `stat/clientbound/changed.go`                                                                                                                                                                                                                            | **1**      |
| ui         | 6         | `ui/clientbound/{disable,lock,open}.go`, `ui/ui_open_body.go`                                                                                                                                                                                            | **3 packets + 1 body** |
| socket     | 10        | `socket/clientbound/{hello,ping}.go`, `socket/serverbound/{channel_connect,pong,start_error}.go`                                                                                                                                                         | **5**      |
| channel    | 4         | `channel/clientbound/change.go`, `channel/serverbound/channel_change.go`                                                                                                                                                                                 | **2**      |
| merchant   | 3         | `merchant/clientbound/operation.go`, `merchant/operation_body.go`, `merchant/serverbound/operation.go`                                                                                                                                                   | **2 packets + 1 body** |
| tool       | 0         | `tool/uint128.go` only                                                                                                                                                                                                                                   | **0 packets** (utility) |
| quest      | 14        | `quest/clientbound/script_progress.go`, `quest/serverbound/{action,action_complete,action_restore_lost_item,action_script_end,action_script_start,action_start}.go`                                                                                     | **7**      |
| **Total**  | **49**    | —                                                                                                                                                                                                                                                        | **~21 packets + 3 bodies** |

This is half the PRD's headline number. Two implications:

1. **The audit-report file count is ~21 SUMMARY rows, not ~49.** TOTAL.md's misc-domain section will sum to ~21. This must be reflected in plan.md's per-file scoping so the plan doesn't pad with non-existent files.
2. **The `*_body.go` files (`fame/response_body.go`, `ui/ui_open_body.go`, `merchant/operation_body.go`) are sub-struct definitions, not standalone packets.** They get *registry entries* (per §5), not SUMMARY rows. The audit row for the parent packet (e.g. `Fame`, `UiOpen`, `HiredMerchantOperation`) cites the body's registry hash.

**PRD `Open Question` §12 (tool/) is resolved here**: `tool/` contains only `uint128.go` (a 128-bit unsigned integer utility type, not a packet). Confirmed via tree-walk. TOTAL.md and `_pending.md` document tool as "no packets; utility-only support package — `uint128` is consumed by socket/channel handshake encoders for hash fields but is not itself wire-bound." No audit rows for tool.

The Phase 2 sub-phase ordering (§11) is sized to actual file counts. Each domain except quest is a single-shot pass (1-5 files); quest is the largest at 7 files and gets its own sub-phase.

---

## 4. The hard part #2: socket sensitivity

Socket has the highest blast radius of any domain in the library:

- `socket/clientbound/hello.go` is sent on every TCP connection establish. The packet format encodes `majorVersion (u16)`, `minorVersion (u16)`, `sendIv ([]byte)`, `recvIv ([]byte)`, `locale (byte)`. A one-byte error here breaks every client login.
- `socket/clientbound/ping.go` is the heartbeat keep-alive. Wrong width → connection drops via client timeout.
- `socket/serverbound/pong.go` is the matching heartbeat ack. Same blast radius.
- `socket/serverbound/channel_connect.go` is the cross-server migration packet — wrong format breaks channel changes for every player.
- `socket/serverbound/start_error.go` reports client-side handshake failures. Less critical but still on the connection-establish path.

**Operational policy for socket fixes**:

- Every fix lands with 4-variant `pt.Variants` tests (GMS v28/v83/v95 + JMS v185). No exceptions.
- Before commit, manually re-verify the fix against **all 6 version templates** (`template_gms_{12,28,83,87,92,95}_1.json` + `template_jms_185_1.json`). The `gms_v12` and `gms_v92` templates exist; even though task-028's audit baseline is v95, socket-fix verification widens the template sweep because socket-handshake formats are version-history-sensitive.
- `atlas-login` + `atlas-channel` build clean before PR. Socket encoders are constructed by both services; signature changes ripple through both.
- The `hello` packet specifically: `sendIv` and `recvIv` are 4-byte AES-OFB session seeds. Their byte order is critical and not obvious from the Go encoder. Verify against IDA `CClientSocket::OnPacket` (or whichever symbol holds the inbound-hello decoder) for IV-byte ordering before any change.

**Predicted socket findings**:

- The `locale` byte at end of `hello` may have version drift — early GMS encoded it as a single byte at offset N, later GMS shifted it for an extra padding byte. Verify before assuming.
- `pong` likely has a 4-byte tick-count payload OR is payload-empty depending on version. Both shapes exist in the wild. IDA evidence required.
- `start_error` likely uses a 1-byte error-code enum; verify case values.

If any socket fix would require a 3-deep guard, **defer to `_pending.md`** rather than nest. Socket is too operationally sensitive to risk a misread structural rewrite.

---

## 5. The hard part #3: misc-domain registry extensions

The PRD §4.6 names three likely additions. The tree walk surfaces three more candidates. Triage now, refine in execution:

| Sub-struct                                | Source type / method                                        | Consumed by                                                        | Confidence |
|-------------------------------------------|-------------------------------------------------------------|--------------------------------------------------------------------|------------|
| Fame response body                        | `libs/atlas-packet/fame/response_body.go`                   | `fame/clientbound/response.go`                                     | High — already a sibling-package body file; needs a `TypeRegistry` entry so the analyzer descends through `ResponseBody.Encode()` without an unresolved-type marker. |
| UI open body                              | `libs/atlas-packet/ui/ui_open_body.go`                      | `ui/clientbound/open.go`                                           | High — same rationale as fame body. |
| Merchant operation body                   | `libs/atlas-packet/merchant/operation_body.go`              | `merchant/clientbound/operation.go`, `merchant/serverbound/operation.go` | High — same rationale, but the body is read by both client- and server-bound encoders, so the entry needs to be re-usable across directions. |
| Quest reward sub-struct                   | TBD — likely inline in `action_complete.go` or `action_start.go` | `quest/serverbound/action_complete.go`, `action_start.go`         | Medium — task-014/015 may have already registered an equivalent type; check before duplicating. |
| Socket version-info block                 | TBD — likely inline in `hello.go` (`sendIv` + `recvIv` + version pair) | `socket/clientbound/hello.go`                                      | Low — inline shape may be sufficient; register only if analyzer surfaces an unresolved type during execution. |
| Channel migrate address block             | TBD — likely inline in `channel/clientbound/change.go` (host:port pair) | `channel/clientbound/change.go`                                    | Low — same rationale as socket; inline is probably fine. |

**Registration discipline (inherited from task-028 §4.1, task-068 §4)**:

- Register only types the misc domain *actually calls*. Don't pre-emptively register a "common body" registry across all `*_body.go` files.
- One registry entry + one `registry_test.go` fixture per addition.
- Land the registry entry in the same commit as the first audited packet that references it.
- For body files (fame, ui, merchant), the test fixture asserts the registered type analyzes to a primitive-field list. The audit report for the parent packet's SUMMARY row treats the body verdict as a transitively-included contribution.

**Cross-domain ripple guardrail**: every registry addition triggers a regression-run of login, character, AND world SUMMARY rows. By the time task-069 reaches Phase 1, the regression surface is large (~138+ rows). If a row flips on a prior-task packet, STOP and investigate. Task-028 `b1af67f6d` and `32b585e8f` showed that registry adds can cascade through `FlattenWithRegistry` to verdict-affecting depth.

---

## 6. The hard part #4: quest cross-task overlap with task-014, task-015, task-023

The quest serverbound packets (`action`, `action_complete`, `action_restore_lost_item`, `action_script_end`, `action_script_start`, `action_start`) plus the clientbound `script_progress` were touched by three prior in-flight tasks:

- **task-014** — conversation reward notices. Likely added reward-encoding changes to `action_complete` and/or `action_start`.
- **task-015** — quest start reward notices. Likely added reward-encoding changes to `action_start`.
- **task-023** — quest selected-skill gate. Likely added a `Region/MajorVersion` gate to `action_complete` or `action_start` for skill-reward encoding.

**Policy**:

1. **Read those tasks' commits BEFORE touching any quest file.** `git log --oneline -- libs/atlas-packet/quest/` from main, scoped to the windows where 014/015/023 landed. Treat the existing gates as load-bearing.
2. **Don't widen or narrow a 014/015/023 gate without IDA evidence from the same version context they used.** If the audit reveals a gate is "wrong" against v95 IDA but matches v83 IDA, that's not a bug — that's a region-version intersection the prior task verified. Verify both versions before changing.
3. **If a real wire bug overlaps a 014/015/023-touched line range, document it in the audit report's header as "fix overlaps task-NNN".** The fix still ships, but the closing memo references the prior task so the cross-task lineage is auditable.

The quest reward sub-struct registry entry (§5) is the highest-risk source of regression. If task-014 or task-015 added the same sub-struct under a different name, we get duplicate registry entries with subtly different shapes — the diff engine will flip verdicts. Phase 1 must check existing registry entries for any `Reward`, `QuestReward`, or `RewardInfo` symbol before adding a new one.

---

## 7. The hard part #5: account / channel dispatcher-offset boundaries

Same shape as task-068 §7 (NPC dispatcher offset). Two cases to verify:

### 7.1 Account serverbound dispatcher offset

`accept_tos`, `register_pin`, `set_gender` are received during login-server dispatch via `CLogin::OnPacket` (or equivalent). The dispatcher MAY prepend an `accountId` or `sessionId` before the per-handler payload. Atlas-side decoders must consistently either include the field at offset 0 OR treat it as already-consumed by the dispatcher layer.

**Verification step**: open `services/atlas-login/` handler registration code. If the handler reads `accountId` off the wire before calling the per-packet decoder, atlas-packet decoders should NOT include it. If the handler delegates wire-reading to the decoder, atlas-packet decoders SHOULD include it. Match the documented contract.

### 7.2 Channel migration dispatcher offset

`channel/clientbound/change.go` is a host:port migration packet. The client treats this as a "disconnect and reconnect to this address" trigger. The wire payload typically encodes:
- Host IP (4 bytes, little-endian or big-endian depending on version)
- Port (2 bytes)
- A 1-byte success/failure flag (in some versions)

The endianness of the IP field is a common bug surface. Verify against IDA `CClientSocket::OnSocketDisconnect` (or equivalent dispatcher) for both v95 and v83.

Predicted finding: at most one of the misc-domain serverbound packets has an inconsistent dispatcher-offset assumption. Fix lands in `libs/atlas-packet/<domain>/serverbound/<file>.go` with a one-line ack of the dispatcher contract.

---

## 8. Merchant scope clarification (employee-shop vs hire-merchant)

PRD §3 non-goal: "Full hire-merchant packet audit (those are `interaction/` domain, handled by task-067)."

What's in scope for task-069's merchant directory:

- `merchant/clientbound/operation.go` — employee shop UI operations (OpenShop, etc.)
- `merchant/serverbound/operation.go` — employee shop user actions
- `merchant/operation_body.go` — sub-struct for operation payloads

What's out of scope:

- `interaction/clientbound/` hire-merchant operation messages — task-067 owns these.
- Mini-game packets — task-067 owns these.

The `merchant/` directory has historically been the source of confusion because GMS clients use a single dispatcher opcode (`HiredMerchantOperation`) for BOTH the employee-shop (NPC-driven) and hire-merchant (player-driven) flows, dispatching internally by a mode byte. Task-067 audits the hire-merchant mode bytes; task-069 audits the employee-shop mode bytes. **The audit report header for `merchant/` packets must explicitly state which mode bytes are in-scope and reference task-067 for the others.**

If during audit it becomes apparent that the same Go encoder file handles both flows (one struct, mode byte dispatched at runtime), then the audit produces one report covering the employee-shop sub-cases, with the hire-merchant sub-cases marked "see task-067" — same per-section convention as task-068's `conversation.go` per-dialog-type breakdown.

---

## 9. Cross-version pass — same cadence, smaller surface

PRD §4.7 cadence (v95 complete → v83 batch → v87 batch → JMS v185 batch) is the inherited task-028/task-068 pattern. Misc-domain-specific adjustments:

- **Socket fixes propagate fastest across versions.** The handshake format changed materially between GMS pre-v83 and GMS v83+. The v83 cross-version pass for `socket/` will likely produce the most opcode-shift fixes per file. Budget time for it.
- **Quest serverbound packets are mostly version-stable.** `action`, `action_complete`, `action_start` had stable wire shapes from GMS v28 through v95. Expect light cross-version churn except for fields touched by task-014/015/023.
- **Channel migration `host:port` width may shift in JMS v185.** JMS clients sometimes encoded port as u32 instead of u16. Verify during the JMS pass.
- **Fame, stat, ui are version-trivial.** These domains likely produce 0-1 cross-version fixes total.

Each version's pass ships as its own commit batch: `audit(misc): GMS v83 cross-version pass`, etc.

---

## 10. TOTAL.md design

The TOTAL.md ledger is the unique deliverable of task-069. PRD §4.9 specifies its contents; this section commits to layout decisions.

### 10.1 Location

`docs/packets/audits/gms_v95/TOTAL.md` — co-located with the existing flat per-packet reports. The "TOTAL" name distinguishes it from `SUMMARY.md` (which is the row-per-packet flat index) — TOTAL.md is the row-per-domain meta-ledger.

### 10.2 Sections

1. **Header**: title, last-updated timestamp, total packet count across all contributing tasks (~138+ expected), latest commit refs for each contributing task's `post-phase-b.md`.
2. **Coverage matrix**: one row per top-level directory in `libs/atlas-packet/`. Columns:
   - Directory (e.g., `login`, `character`, `field`, etc.)
   - Owning task (027, 028, 065, 066, 067, 068, 069, or "—" for out-of-scope)
   - Packet file count (from `find ... -name '*.go' ! -name '*_test.go'`)
   - Verdict roll-up: ✅ count / ⚠️ count / ❌ count
   - Notes (e.g., "utility-only; not wire-bound" for tool/)
3. **Gaps section**: any `libs/atlas-packet/<dir>/` not claimed by tasks 027-069. Each row: directory, reason (defer / new task / out-of-scope-permanently), and a one-line rationale. The expectation after the §11 coverage sweep is that this section is **empty** — every directory is either claimed or documented as out-of-scope.
4. **Task references**: links to each contributing task's `prd.md`, `design.md`, `plan.md`, `post-phase-b.md` for traceability.
5. **Coverage-completeness statement**: a single sentence at the end stating "Coverage of `libs/atlas-packet/` is complete as of <commit ref>." This is the closing line — the TOTAL.md's existence purpose is to be the place a reviewer can answer "is the library wire-correct?" in one document.

### 10.3 Maintenance policy

TOTAL.md is **append-on-task-close** — a future audit task (say, task-080-foo-domain) appends its row to the coverage matrix at the same time it ships its closing memo. The first author (task-069) sets the template; subsequent tasks fill rows.

The maintenance policy is documented in TOTAL.md's header so future authors don't have to reverse-engineer it: "To add a domain: append a row to §2 with task-id, file count, and verdict roll-up. Update last-updated timestamp. Recompute coverage-completeness statement if the gap section was non-empty before."

### 10.4 Sibling-task in-flight handling

PRD §12.3 left open: if 065-068 hasn't finalized SUMMARY.md by the time task-069 reaches Phase 4, do we draft with placeholders or wait?

**Resolution: draft with placeholders, revise before PR.** Concretely:

- Phase 4 writes TOTAL.md with verdict counts pulled from each sibling task's latest `SUMMARY.md` state at that moment.
- For any sibling task still in flight (e.g., 067 hasn't shipped `post-phase-b.md` yet), the TOTAL.md row gets a `(draft)` annotation and the commit ref is the sibling's HEAD-of-branch SHA.
- A pre-PR sweep updates the `(draft)` rows to final state. If a sibling task ships AFTER task-069's PR opens, that sibling task's PR amends TOTAL.md as part of its own closing-memo commit.

The maintenance policy (10.3) covers the post-PR amend pattern.

### 10.5 Roll-up arithmetic

Verdict counts come from the SUMMARY.md row count per task (grep for the FName prefix or domain marker). The arithmetic is mechanical:

- ✅ count = grep '`| ✅ |`' for that domain's rows in SUMMARY.md.
- ⚠️ count = grep '`| ⚠️ |`' for same.
- ❌ count = grep '`| ❌ |`' for same.

If a sibling task uses different verdict markers, normalize to ✅/⚠️/❌ in TOTAL.md. Document the normalization rule in the TOTAL.md header if it kicks in.

---

## 11. Phasing — concrete artifacts

Mirror of task-028 §5 / task-068 §11 phasing, misc-domain inputs.

### Phase 0 — Regression baseline + analyzer sanity (gate)

Re-run the existing v95 audit unchanged against the current pipeline. Verify:

- Login SUMMARY rows byte-identical to pre-task state (28 rows from task-027).
- Character SUMMARY rows byte-identical (52 rows from task-028).
- Combat / social / commerce / world SUMMARY rows byte-identical to their respective task-shipped states (counts TBD by task-069 start time; pull from each sibling's `post-phase-b.md` Final state section).
- No analyzer panics, no new cycle-guard fires.

Artifacts:
- Pipeline re-run output (no commit needed if no diffs).
- One commit: `audit(misc): phase-0 regression baseline confirms prior-domain verdicts unchanged`.

Exit: prior SUMMARY rows confirmed unchanged.

### Phase 1 — TypeRegistry extension batch (predicted)

Register the High-confidence types up front per §5 triage:

- Fame response body (`fame/response_body.go`).
- UI open body (`ui/ui_open_body.go`).
- Merchant operation body (`merchant/operation_body.go`).

Quest reward sub-struct: defer to Phase 2g — first check for prior task-014/015 registry entries before adding.

Each registration lands with one `registry_test.go` fixture asserting the registered type analyzes to the expected primitive-field list. Predicted, not exhaustive — Phase 2 surfacings get added then.

Exit: registry test suite green; SUMMARY rows for all prior domains byte-identical.

### Phase 2 — v95 misc audit

Run the audit, triage findings, ship fixes. Sub-phases ordered easy-wins-first to build context for the higher-risk passes (mirror of task-068 §11 ordering rationale):

- **2a — tool/ confirmation (0 packets)**: enumerate, confirm utility-only, write a `_pending.md` row + a TOTAL.md note. No SUMMARY rows. Closes PRD §4.8.
- **2b — stat/ (1 file)**: smallest packet domain. `stat/clientbound/changed.go` audit-and-go.
- **2c — channel/ (2 files)**: clientbound + serverbound. Dispatcher-offset verification per §7.2.
- **2d — ui/ (3 files + 1 body)**: `disable`, `lock`, `open`. The body got registered in Phase 1.
- **2e — fame/ (2 files + 1 body)**: `clientbound/response`, `serverbound/change`. Body registered in Phase 1.
- **2f — merchant/ (2 files + 1 body)**: employee-shop scope per §8. Coordinate with task-067 if hire-merchant mode bytes surface.
- **2g — quest/ (7 files)**: largest misc-domain sub-phase. Read task-014/015/023 commit history before touching files. Quest reward sub-struct registry decision happens here. Dispatcher-offset verification per §7.1.
- **2h — account/ (3 files)**: TOS / pin / gender flows. Dispatcher-offset verification per §7.1.
- **2i — socket/ (5 files)**: highest-blast-radius domain — left for last so accumulated audit context is maximal. Extra-cautious build-clean step per §4 before commit.
- **2j — `_pending.md` updates**: bare-handler exclusions, unresolvable sub-op branches, deep sub-struct deferrals. Each gets an explicit row.

Each sub-phase ends with a verdict-triaged commit set: fix commits individually, audit-report commits batched per sub-phase. Mirror of task-068 §11 Phase 2.

The audit is "done" when SUMMARY.md shows a verdict for every one of the ~21 misc-domain packets OR an explicit `_pending.md` entry for any deferred. No silent skips.

### Phase 3 — Cross-version pass (v83 → v87 → JMS v185)

Per PRD §4.7 cadence. One IDA database at a time, user-driven swap. For each version:

1. User loads the IDA database.
2. Walk the misc-domain FName list (established by Phase 2's v95 export).
3. Populate matching `gms_v{83,87}.json` / `gms_jms_185.json` for misc FNames.
4. Re-run the audit with that version's IDA source and template.
5. For each divergence vs v95 atlas-packet behaviour:
   - Existing `Region/MajorVersion` gate already correct → no atlas change; export row captures evidence.
   - Atlas gate wrong → fix the gate, sweep tests across 4 variants, document.
   - Template opcode drift → fix the template, cite case-statement value.

Each version's pass ships as its own commit batch: `audit(misc): GMS v83 cross-version pass`, etc.

Expected churn (per §9):
- v83: highest for socket; light elsewhere.
- v87: light overall (intermediate version).
- JMS v185: moderate for channel (port-width), light elsewhere.

### Phase 4 — TOTAL.md + post-phase-b.md + finishing

Mirror of task-027/task-028/task-068 closing pattern, plus the unique TOTAL.md deliverable.

1. **Coverage-completeness sweep** per PRD §4.10: run `find libs/atlas-packet -maxdepth 1 -type d | sort` and cross-reference against the 7-task coverage matrix. For each unclaimed directory:
   - If it contains `.go` files that look like packets, add to TOTAL.md gaps with "defer to new task" recommendation.
   - If it's a utility / model / test directory, add to gaps with "out-of-scope-permanently" + rationale.
   - If it's empty or already documented, no entry needed.
2. **Write `docs/packets/audits/gms_v95/TOTAL.md`** per §10 design.
3. **Write `docs/tasks/task-069-misc-domain-packet-audit/post-phase-b.md`** with sections:
   - Final state (packets audited, verdict counts, IDA-export coverage).
   - Real wire bugs fixed (table: packet, file, IDA citation, fix one-liner, affected versions).
   - Template opcode/enum fixes (table: template file, old → new, IDA case-statement, reason).
   - Tooling improvements (registry additions; analyzer should be untouched — if it isn't, explain why).
   - Remaining work (deferred sub-op modeling, loop-count modeling, deep sub-struct descent — likely identical entries to task-028/068's remaining-work tables; plus any task-069-specific deferrals).
   - **Tool-domain confirmation note** per PRD §4.8.
   - **Coverage statement**: explicit reference to TOTAL.md and the sweep result.
4. **Verification**:
   - `go build ./...` clean.
   - `go vet ./libs/atlas-packet/... ./tools/packet-audit/...` clean.
   - `go test -race ./libs/atlas-packet/... ./tools/packet-audit/...` clean.
   - gitleaks scrub: `grep -r '/home/' docs/packets/audits/gms_v95/` empty.
   - `docker build -f services/atlas-configurations/Dockerfile .` if templates changed in structure-affecting ways.
   - `atlas-login`, `atlas-channel`, `atlas-account` build clean (the three services that consume socket / account encoders).
5. **Code review** via `superpowers:requesting-code-review` (plan-adherence + backend-guidelines) before PR.

---

## 12. PRD Open Questions — Resolutions

PRD §12 left three questions.

### 12.1 tool/ as a packet domain

**Resolved (§3 inventory walk)**: `tool/` is utility-only (`uint128.go`). Zero packet files. Documented in TOTAL.md, `_pending.md`, and `post-phase-b.md`.

### 12.2 Hidden domains under `libs/atlas-packet/`

**Resolved (§3 enumeration + §11 Phase 4 sweep)**: tree walk lists 27 top-level directories. The 7-task coverage matrix accounts for: `login` (027), `character` (028), `inventory`/`pet`/`storage` (commerce-domain candidates — verify task-067 scope), `cash` (066 or 067 — verify), `buddy`/`messenger`/`note`/`chat` (066 social), `monster`/`drop`/`reactor`/`interaction` (065 combat + 067), `field`/`portal`/`npc` (068), `account`/`fame`/`stat`/`ui`/`socket`/`channel`/`merchant`/`tool`/`quest` (069), `model`/`test` (non-packet — model holds shared types, test holds harness code). `packet.go`, `resolve.go`, `resolve_test.go` are top-level library files, not domains.

If Phase 4 sweep finds any directory NOT in the above list with `.go` files containing `Encode` or `Decode` methods, treat it as a hidden domain and triage per §11 Phase 4 step 1.

### 12.3 In-flight sibling tasks and TOTAL.md draft

**Resolved (§10.4)**: draft with `(draft)` annotations; revise before PR. Subsequent sibling-task PRs amend TOTAL.md as part of their closing-memo commit.

---

## 13. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Registry addition cascades through `FlattenWithRegistry` and flips a prior-domain verdict | Medium | High | Phase 0 + Phase 1 each re-run prior SUMMARY rows. Any drift triggers STOP-and-investigate. Cross-task surface is large by this task (~138+ rows) — extra-diligent regression check. |
| Socket fix introduces a `hello` / `pong` regression breaking every client | Low-medium | Critical | §4 explicit policy: 4-variant tests, 6-version template re-verification, atlas-login + atlas-channel build clean before commit. Treat socket fixes as the highest-attention category in the task. |
| Quest fix overlaps a task-014/015/023 gate and silently widens/narrows it | Medium | Medium | §6 explicit policy: read task-014/015/023 commits before any quest change; verify both v95 and prior-version IDA evidence before changing any pre-existing gate. |
| Quest reward sub-struct duplicate registry entry with task-014/015 | Medium | Medium | Phase 1 / Phase 2g explicit check for existing `Reward*` registry entries before adding new ones. |
| Account dispatcher-offset inconsistency hides a real bug | Low | Medium | §7.1 verification step against atlas-login handler code before assuming offset 0 layout. |
| Channel migration IP endianness wrong on cross-version pass | Medium | Medium | §9 explicit verification against IDA `CClientSocket::OnSocketDisconnect`; 4-variant test enforces wire correctness. |
| Merchant scope confusion with task-067 hire-merchant | Medium | Low | §8 audit-report header explicitly states which mode bytes are in-scope and references task-067 for others. |
| TOTAL.md draft state at PR time because sibling task hasn't shipped | High | Low | §10.4 explicit policy: `(draft)` annotations, post-merge amends from sibling tasks. Not a blocker for task-069's PR. |
| Coverage-completeness sweep finds a hidden packet domain | Low | Medium | §11 Phase 4 step 1: triage per discovery; defer to a new task rather than expanding scope mid-PR. |
| Sub-op dispatched files (`merchant/operation` if it dispatches by mode) eat manual-annotation budget | Medium | Medium | Manually annotate per task-028 `StatusMessage*` / task-068 `effect_weather` pattern. Cap effort at 2 hours per file; if over, defer to `_pending.md`. |
| `*_body.go` body files contain version branches that get missed | Medium | Medium | Phase 1 registry tests assert primitive-field list; if a body has version branching, register a per-version pair (mirror of task-068's clock per-mode pair). |
| Cross-version template opcode drift across 4 versions explodes the cross-version pass | Medium | Medium | One commit per opcode fix per version, citing IDA case-statement. Same cadence as task-028/068. |
| Audit-report ack footer accidentally added before final run | Medium | Low | Convention: ack footer is the LAST line written. If a re-run is needed, `git checkout HEAD -- <report.md>` first, per task-028/068 closing memo. |
| gitleaks catches absolute paths in audit reports | High | Low | Phase 4 pre-PR check: `grep -r '/home/' docs/packets/audits/gms_v95/` empty. Same as prior tasks. |
| Mid-task scope creep into atlas-account / atlas-quest service code | Medium | Medium | PRD §3 non-goal is explicit. Refuse any "while we're here, fix the account service for X" excursion. Log to `_pending.md` instead. |
| TOTAL.md verdict roll-up arithmetic disagrees with sibling task's own roll-up | Low | Low | §10.5 mechanical grep-based count from SUMMARY.md. If a sibling reports a different count in its own `post-phase-b.md`, TOTAL.md cites the grep-derived count and footnotes the discrepancy. |

---

## 14. Out of scope (explicit)

- Bare-handler descent into atlas-account, atlas-channel, atlas-quest, atlas-login service code (PRD §3).
- Hire-merchant packet audit — task-067 owns those (PRD §3, design §8).
- Sub-op enum modeling in the audit pipeline (limitation acknowledged, not fixed — task-028/068 deferral).
- Loop-count modeling in the audit pipeline (same).
- Deep sub-struct descent in the audit pipeline (same).
- Login (027), character (028), combat (065), social (066), commerce (067), world (068) verdict re-runs beyond regression-only.
- v28 binary integration (inherited deferral from task-028 §6).
- Service-layer changes beyond the minimum needed to wire a fix through (PRD §3).
- Migration of audit reports to per-domain subfolders (task-068 §10 decision, inherited).
- Generic packet-DSL or schema-first encoder rewrite (carried forward from task-027 §12).
- Re-design of `_pending.md` structure across prior tasks (append-only; per-task heading).
- Refactor of `*_body.go` files into their parent encoder files (out of scope; audit-as-is).
- New cross-version variant beyond GMS v28/v83/v95 + JMS v185 (`pt.Variants` set is fixed per task-028 §6).
- Performance work on hot socket / quest packets (no perf regression assumed; not a goal).

---

## 15. Reference points in the existing tree

- `libs/atlas-packet/account/serverbound/{accept_tos,register_pin,set_gender}.go` — login-server dispatcher-offset boundary per §7.1.
- `libs/atlas-packet/fame/{clientbound/response.go,response_body.go,serverbound/change.go}` — fame domain (2 packets + 1 body).
- `libs/atlas-packet/stat/clientbound/changed.go` — single-file stat-change notification.
- `libs/atlas-packet/ui/{clientbound/{disable,lock,open}.go,ui_open_body.go}` — UI state-change packets + body.
- `libs/atlas-packet/socket/{clientbound/{hello,ping}.go,serverbound/{channel_connect,pong,start_error}.go}` — connection-establish + heartbeat surface. §4 critical-path policy applies.
- `libs/atlas-packet/channel/{clientbound/change.go,serverbound/channel_change.go}` — channel migration. §7.2 dispatcher-offset boundary.
- `libs/atlas-packet/merchant/{clientbound/operation.go,operation_body.go,serverbound/operation.go}` — employee-shop scope only (§8).
- `libs/atlas-packet/tool/uint128.go` — utility type; not a packet. §3 / §11 Phase 2a.
- `libs/atlas-packet/quest/{clientbound/script_progress.go,serverbound/{action,action_complete,action_restore_lost_item,action_script_end,action_script_start,action_start}.go}` — quest dispatch surface. §6 cross-task coordination policy applies.
- `tools/packet-audit/internal/atlaspacket/registry.go` — additions per §5.
- `tools/packet-audit/internal/atlaspacket/registry_test.go` — fixture format to mirror.
- `tools/packet-audit/internal/atlaspacket/analyzer.go` — DO NOT TOUCH unless a panic or new cycle surfaces.
- `services/atlas-configurations/seed-data/templates/template_gms_{83,87,95}_1.json` — opcode/enum sites for misc-domain writers (`HelloWriter`, `PingWriter`, `ChangeChannelWriter`, `HiredMerchantOperationWriter`, `QuestActionResultWriter`, etc.).
- `services/atlas-configurations/seed-data/templates/template_gms_{12,28,92}_1.json` — additional templates for socket-fix re-verification per §4.
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — JMS opcode/enum site.
- `docs/packets/audits/gms_v95/SUMMARY.md` — append misc-domain rows to existing flat index (~21 new rows).
- `docs/packets/audits/gms_v95/TOTAL.md` — **NEW** cross-task ledger per §10.
- `docs/packets/audits/gms_v95/` — flat per-packet reports.
- `docs/packets/ida-exports/gms_v95.json` — append misc FNames during Phase 2.
- `docs/packets/ida-exports/{gms_v83,gms_v87,gms_jms_185}.json` — append during Phase 3 per version.
- `docs/packets/ida-exports/_pending.md` — append misc bare-handler exclusions and unresolvable branches; tool-domain confirmation note.
- `docs/tasks/task-027-atlas-packet-v95-audit/post-phase-b.md` — closing-memo template (oldest).
- `docs/tasks/task-028-character-domain-audit/post-phase-b.md` — closing-memo template (preferred — most-recent shipped pattern).
- `docs/tasks/task-068-world-domain-packet-audit/` — design.md and plan.md structural template for this task.

---

## 16. What plan-task should do next

The plan should split this design into **explicit, sequenced, small** tasks. Suggested structure:

- **2 tasks for Phase 0 + Phase 1** (regression baseline + Phase 1 registry batch with 3 predicted body entries).
- **One task per Phase 2 sub-phase (a–j)**: 10 sub-tasks. Each ends with verdict-triaged commit set. Note sub-phase 2a (tool/) is documentation-only.
- **One task per cross-version pass (Phase 3 — v83, v87, JMS v185)**: 3 sub-tasks.
- **One task for Phase 4** (TOTAL.md + post-phase-b.md + coverage sweep + verification + code review).

Total target: ~15-17 plan tasks.

Specifically the plan should answer:

- The order of Phase 2 sub-phases. Recommended (per §11): 2a (tool/) → 2b (stat/) → 2c (channel/) → 2d (ui/) → 2e (fame/) → 2f (merchant/) → 2g (quest/) → 2h (account/) → 2i (socket/) → 2j (_pending.md sweep). Front-loads documentation-only and trivial-domain wins; leaves quest (cross-task coordination), account (dispatcher offset), socket (critical path) for the end when context is maximal.
- The exact list of high-confidence registry additions for Phase 1 (only the 3 body entries from §5; medium/low-confidence wait for Phase 2 evidence).
- The fixture format for new registry tests (mirror existing `CharacterStat::Encode` / `AttackInfo` patterns from task-028).
- The commit-naming convention. Recommended: `audit(misc): <sub-phase> <file>` for audit-report commits; `fix(packet/<domain>): <packet> — <one-line>` for wire-bug fix commits; `audit(misc): GMS v<N> cross-version pass` for Phase 3 batches.
- The pre-PR rebase strategy. Recommended: rebase-clean (squash repetitive `audit(misc):` commits per sub-phase) before opening PR. Fix commits stay individual.
- Whether to bundle the v83/v87/JMS-185 passes into one PR or split per version (suggestion: one PR for the whole task per task-028/068 precedent).
- The TOTAL.md authorship order — write the matrix first with prior-task placeholders, then fill misc-domain rows after Phase 2 verdicts settle.
- The grep-based verdict roll-up procedure for TOTAL.md (§10.5) — confirm the regex patterns work against current SUMMARY.md formatting before relying on them.

---

## 17. What is NOT being decided here (deferred to plan / execution)

- The exact symbol names for the misc-domain sub-struct types (quest reward sub-struct, socket version-info block, channel migrate address). Inspect during execution.
- Whether `socket/clientbound/hello.go` needs a registry entry for its IV-block sub-struct (likely inline is sufficient; verify in Phase 2i).
- The IDA dispatcher case-statement values for `HiredMerchantOperationWriter`, `QuestActionResultWriter`, `HelloWriter`, etc. — these come from Phase 2 IDA work, not pre-execution speculation.
- The final count of `_pending.md` rows. Likely 5-15 per task-028/068 precedent and given the smaller misc-domain surface.
- The exact verdict roll-up arithmetic for sibling tasks if they ship with non-standard SUMMARY.md formatting. §10.5 sets the default; if a sibling deviates, footnote the discrepancy in TOTAL.md.
- Whether the Phase 4 coverage-completeness sweep produces any hidden-domain finding. Expected: no, but the sweep is mandatory per PRD §4.10.
- The post-PR amend protocol for sibling tasks updating TOTAL.md after task-069 ships. §10.3 documents the policy; sibling-task plan-tasks would need to incorporate it.
