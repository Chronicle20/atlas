# IDA Export Re-Harvest — Trustworthy Four-Version Packet Read-Orders — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-04
---

## 1. Overview

The packet-audit baseline (task-080) compares Atlas's Go packet encoders/decoders against IDA-exported client read-orders for the four-version baseline: GMS v83, GMS v87, GMS v95, and JMS v185. The exports live as checked-in JSON at `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` (~354 functions each), with each function's field read-order captured as a `calls` array.

**Those exports are not trustworthy.** task-080 proved it concretely: while fixing the `BuddyInvite` packet, all four client `CWvsContext::OnFriendResult` case-9 read-orders were decompiled by hand and compared to the exports — **three of the four exports were wrong**. The v83/v87 exports mis-traced a 39-byte `GW_Friend` struct read (via `CFriend::Insert`) as a variable-length `count + buddy[i]` loop, and the JMS export truncated after the `level` field, dropping the entire friend buffer + trailing `inShop` byte. The corrected, hand-decompiled read-orders bore little resemblance to what the exporter produced.

This matters because the export is the audit's *input*. When the export is wrong, every downstream verdict for that packet is meaningless. task-080's regenerated SUMMARYs carry ~325 residual `❌`/`🔍` rows; the single largest bucket (~174) is "export read-order truncation," and a further bucket is "mistrace." Those rows are **not** "Atlas verified correct" and **not** "Atlas verified wrong" — they are *"the audit could not check this because its input is wrong."* That bucket almost certainly hides additional real wire bugs the analyzer cannot surface, because it is comparing Atlas against a corrupted reference.

This task makes the four-version exports faithful to real client behavior — by fixing the **exporter** so future exports descend into struct-reading helpers and never truncate — then re-exports all functions across the four IDBs, re-runs the audit on that trustworthy input, and **fixes the genuine wire divergences that surface**. It additionally closes the two adjacent gaps that block a genuinely-complete four-version socket baseline: the analyzer's **opaque register-boundary types** (which the audit skips entirely) and the **partial per-version templates** (where packet code exists but isn't routed). The end state moves the baseline from "trusted but input-limited" toward genuine, verified four-version socket read/write correctness.

**Dependency / sequencing:** this task builds directly on task-080 (the A1–A5 analyzer enhancements, the `packet-audit` audit/export tooling, the curated `_pending.md` registry, the `STARTING_A_NEW_VERSION_PASS.md` guide, and the corrected baseline). task-080 (PR #678) must merge to `main` before this task's execution phase begins, OR this branch rebases onto task-080. The PRD is written against the post-task-080 codebase.

## 2. Goals

Primary goals:
- **Fix the exporter** (`packet-audit export` / the IDA-MCP harvest path) so a generated export faithfully captures a client function's full wire read-order: (a) it **descends into struct-reading helper functions** (e.g. `CFriend::Insert`→`GW_Friend`, `GW_ItemSlotBase`, `GW_CharacterStat::Decode`) and inlines their reads instead of mislabeling them; (b) it **captures trailing fields with no truncation**; (c) it correctly resolves loops vs. fixed structs (the v83 BuddyInvite mistrace is the canonical anti-case it must now get right).
- **Full re-export** of all functions across all four IDBs (GMS v83/v87/v95 + JMS v185) using the fixed exporter, replacing the checked-in JSONs.
- **Re-run the audit** on the corrected exports; the truncation/mistrace residue must collapse to either `✅` or a genuine, real finding.
- **Fix in-task** every genuine wire divergence the corrected audit surfaces, with byte-level tests + version/region gates (task-080 B1.x discipline: verify against IDA, ship a byte test as the oracle).
- **Decompose the opaque register-boundary types** so the analyzer verifies them instead of skipping them (or, where genuinely undecomposable, confirm them with a byte-test-backed exception that cites IDA evidence — not an unexamined skip).
- **Complete the partial per-version templates** so packet families that have code but no routing (notably JMS NPC-shop and player-interaction) are wired and audited per version, OR explicitly verdicted as client-absent.
- **End state:** a re-run four-version audit where every residual `❌`/`🔍` is a *genuine, justified* exclusion (real client absence, or a verified-by-byte-test representation equivalence) — never "the export was wrong" and never an unexamined opaque skip.

Non-goals:
- Re-litigating packets task-080 already fixed and byte-tested (AffectedAreaCreated, chat Multi, the JMS cash bodies, login Request, BuddyInvite, etc.) unless the corrected export reveals a *new* divergence in them.
- Adding support for packets/features that do not exist in any of the four baseline clients.
- Expanding the baseline beyond the four versions (GMS v83/v87/v95 + JMS v185).
- A from-scratch rewrite of the `packet-audit` analyzer (A1–A5 stand); only the *exporter/harvest* path and any analyzer support needed for opaque decomposition are in scope.

## 3. User Stories

- As a **server developer**, I want the packet-audit's reference read-orders to match the real client, so that a `✅` verdict actually means "this packet is wire-correct for that version" and a `❌` means a real bug — not an export artifact.
- As a **server developer**, I want every opaque/structured packet (item slots, character/monster stat blobs, friend entries) verified against the client, so I am not relying on stale byte-tests for the most complex packets.
- As a **JMS-server operator**, I want the JMS template to route every packet family the client actually sends, so JMS players can use NPC shops, player stores, and merchant interactions that currently have code but no routing.
- As a **future maintainer adding a new client version**, I want the exporter to produce a complete, faithful read-order in one run, so I don't have to hand-decompile every structured packet to trust the audit (the task-080 BuddyInvite experience).
- As a **reviewer**, I want every residual `❌`/`🔍` in the final SUMMARYs to carry a real, IDA-cited justification, so the baseline is a trusted artifact rather than a list of unknowns.

## 4. Functional Requirements

Organized by capability area.

### 4.1 Exporter fidelity (the core fix)
- **FR-1.1 Struct-helper descent.** When the exporter traces a client function and encounters a call to a helper that reads from the packet (e.g. `CFriend::Insert(pkt)`, a `GW_*::Decode`/`Insert`, `DecodeBuffer` into a known struct), it MUST recurse into that helper and inline its `Decode*` sequence into the parent's read-order, rather than emitting a single opaque/loop entry. The v83 `OnFriendResult#Invite` → `sub_A40028` → `sub_4E4427` (GW_Friend 39 bytes) + `Decode1` (inShop) chain is the canonical case that MUST resolve to `…name, GW_Friend(39), inShop`, not `count + [characterId, channelId, mapId] loop`.
- **FR-1.2 No truncation.** The exporter MUST capture the *complete* read-order through the end of the function's packet-consuming path. The JMS BuddyInvite case (truncated after `level`, dropping `GW_Friend`+`inShop`) MUST NOT recur. Where a function has multiple sub-cases (a mode `switch`), each case's full read-order is captured under its discriminator.
- **FR-1.3 Loop vs. fixed-struct disambiguation.** The exporter MUST distinguish a genuine count-prefixed loop (`Decode1 count` + N×entry) from a fixed-size struct read whose internal fields merely *look* like a loop. Heuristics: a fixed `N*Index` stride in the surrounding code, a struct typedef, or an `Insert`/`Decode` helper indicates a fixed struct, not a loop.
- **FR-1.4 Field semantics + width.** Each captured field retains its op (`Decode1/2/4/8/Str/Buffer`), its byte width, a best-effort semantic label, and any guard/branch condition (mode/version) — preserving the existing JSON schema so the audit consumes it unchanged.
- **FR-1.5 Determinism + provenance.** A re-export of the same IDB MUST be deterministic (stable ordering) and record the function address + IDB identity, so diffs are reviewable.
- **FR-1.6 Bounded recursion.** Helper descent MUST be cycle-guarded and bounded (a visited-set), and MUST NOT chase non-packet-reading calls (UI/dialog/alloc helpers like `CUIFadeYesNo::*`, `StringPool::*`).

### 4.2 Full four-version re-export
- **FR-2.1** Using the fixed exporter, regenerate the complete `calls` set for every function in each of the four IDBs (GMS v83/v87/v95 + JMS v185), one IDB loaded at a time (user-cycled).
- **FR-2.2** Replace the four checked-in `docs/packets/ida-exports/*.json` with the corrected exports. The diff is expected to be large; it MUST be reviewable (per-function, stable ordering) and committed with a summary of what changed structurally (struct descents resolved, truncations recovered, loops corrected).
- **FR-2.3** Where the fixed exporter still cannot resolve a function (genuinely indirect dispatch, data-driven reads), it MUST emit an explicit "unresolved" marker rather than a wrong/partial trace, so the audit treats it as a known gap, not a false verdict.

### 4.3 Re-audit + fix surfaced bugs
- **FR-3.1** Re-run the four-version audit (`packet-audit`, the task-080-corrected invocation: `-template <file>` required, `-output docs/packets/audits`) over the corrected exports, regenerating all four SUMMARYs + per-packet reports.
- **FR-3.2** Reconcile the new residual `❌`/`🔍` set against task-080's baseline **via the verdict-delta triage protocol (§4.7)**. Every row must resolve to one of: (a) now `✅`; (b) a genuine real wire bug (→ fix, FR-3.3); (c) a verified representation-equivalence or client-absence exclusion with IDA evidence; (d) an explicit unresolved-export marker (FR-2.3).
- **FR-3.3 Fix surfaced wire bugs in-task.** Each genuine divergence is fixed in Atlas (`libs/atlas-packet` + downstream handlers/producers as needed), following task-080 discipline: confirm the read-order in IDA, ship a per-version byte-level test as the oracle, apply version/region gates symmetrically in Encode/Decode, use the region-dispatch idiom for >2-version divergences (≤2 nested guards). No wire change ships on analyzer verdict alone.
- **FR-3.4** If a surfaced bug is genuinely too large to fix in-task (a multi-service protocol change), it is fixed if feasible, else registered as its own dedicated follow-up task (never parked back into `_pending.md` as accepted).

### 4.4 Opaque register-boundary decomposition
- **FR-4.1** For each type in task-080's opaque set (e.g. `model.Asset`/`GW_ItemSlotBase`, `GW_CharacterStat`, monster stat blobs, `BuddyEntry`, pet bodies, the ~31 A3-flagged types), determine via IDA whether the client read is decomposable into known primitives.
- **FR-4.2** Where decomposable, extend the analyzer/registry (or the corrected export) so the type's fields are verified inline rather than skipped, and the audit produces a real per-field verdict.
- **FR-4.3** Where genuinely undecomposable (mask/mode-driven variable layout the analyzer cannot statically model), confirm Atlas's encoder against the client with a dedicated byte-level test and record a *verified* exception (IDA evidence + the test as oracle) — replacing the current "analyzer skipped it" status with "verified correct, analyzer can't model it."

### 4.5 Per-version template completeness
- **FR-5.1** Enumerate the packet families that have Atlas packet/handler code but are unrouted in a given version's template — notably JMS NPC-shop (`NPCShopHandle`/`NPCShopOperation`) and the player-interaction (mini-room) family beyond the two ops task-080 wired.
- **FR-5.2** For each, either wire the per-version op-byte map (confirmed against IDA, like task-080 B5.1f) so the family routes and audits per version, OR record a verified verdict that the family is client-absent in that version.
- **FR-5.3** Validate every edited template parses (`python3 -m json.tool`) and that the audit reflects the newly-routed families.

### 4.6 Baseline + ledger update
- **FR-6.1** Re-curate `docs/packets/ida-exports/_pending.md` and `docs/packets/audits/gms_v95/_pending.md` so the accepted-exclusions registry contains *only* verified exclusions — the "export read-order truncation/mistrace" category is eliminated (those rows are now either fixed or genuinely-unresolved markers).
- **FR-6.2** Update `docs/packets/audits/gms_v95/TOTAL.md` (verdict roll-up + completeness statement) and `STARTING_A_NEW_VERSION_PASS.md` (document the fixed exporter's descent behavior and the new export workflow).

### 4.7 Verdict-delta triage (the ✅→❌ flip gate)
The corrected exports are *better* input but MUST NOT be assumed to be ground truth — only the IDA decompile is (BuddyInvite proved even a plausible trace can be wrong). Because a corrected exporter can flip a packet's verdict in either direction — including turning a coincidentally-`✅` packet (old wrong export matching a wrong Atlas encoder) into a real `❌` — the re-audit reconciliation (FR-3.2) MUST proceed by verdict-delta triage rather than by trusting the new numbers:
- **FR-7.1 Snapshot.** Before re-export, snapshot task-080's per-packet verdict set (per version) as the comparison baseline.
- **FR-7.2 Per-packet delta.** After re-export + re-audit, compute the per-packet verdict delta against the snapshot — the exact set of packets whose verdict changed — NOT aggregate counts.
- **FR-7.3 Flip classification + handling.** Every delta entry is classified and handled:
  - **`❌`→`✅`** (expected truncation/mistrace wins): confirm a representative sample flipped *because* of the corrected read-order (not coincidence); the remainder are accepted.
  - **`✅`→`❌`** (the dangerous class): MUST be investigated individually by hand-decompiling the function in IDA and comparing to Atlas. The outcome is exactly one of: **(a)** a real Atlas wire bug the old export was hiding → fix Atlas with a byte test (FR-3.3); **(b)** the new export is itself wrong/over-corrected → fix the **exporter** (FR-1), not Atlas; **(c)** a verified representation equivalence → recorded exception with IDA evidence. A `✅`→`❌` flip MUST NOT be acted on by trusting the new export alone.
  - **new `❌`/`🔍` on a previously-unaudited or unresolved packet:** investigated the same way as `✅`→`❌`.
- **FR-7.4 IDA is ground truth; the byte test is the oracle.** For every flip requiring a code change (Atlas *or* exporter), the decision is made from the actual IDA decompile — not from either export — and a per-version byte-level test written *from the decompile* is the final oracle.
- **FR-7.5 Zero unexplained flips.** The task is not complete until every `✅`→`❌` (and every new non-`✅`) flip carries a written disposition: fixed-Atlas-bug, fixed-exporter-gap, or verified-equivalence. No flip is rubber-stamped in either direction.

## 5. API Surface

This is a tooling + audit task; there are no new runtime service endpoints. The relevant "interfaces":

- **Exporter CLI** (`tools/packet-audit` export path): the invocation and flags for producing an export from a loaded IDB via IDA-MCP. Any new/changed flags (e.g. a descent-depth cap, an unresolved-marker mode) are documented in the new-version-pass guide.
- **Export JSON schema** (`docs/packets/ida-exports/*.json`): the `functions[FName] = {address, direction, calls: [{op, comment, guard?}]}` shape is preserved (so the audit consumes it unchanged). Additions: an explicit `unresolved: true` marker (FR-2.3) and inlined struct-descent entries.
- **Audit output** (`docs/packets/audits/<region>_v<major>/{SUMMARY.md, <Packet>.{md,json}}`): unchanged schema; regenerated content.

## 6. Data Model

No database entities. The "data model" is the export/audit file artifacts:

- `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` — corrected, full re-export (replaced).
- `docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/SUMMARY.md` + per-packet `.md`/`.json` — regenerated.
- `docs/packets/ida-exports/_pending.md` + `docs/packets/audits/gms_v95/_pending.md` — re-curated registries (truncation/mistrace category removed).
- `docs/packets/audits/gms_v95/TOTAL.md`, `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` — updated.
- Any Atlas wire fixes: `libs/atlas-packet/**` packet structs + tests; downstream `services/**` handlers/producers as surfaced.

All `docs/packets/**` artifacts are tenant-agnostic build/audit data (no `tenant_id`).

## 7. Service Impact

- **`tools/packet-audit`** — the exporter/harvest path is the primary change (struct descent, no-truncation, loop disambiguation, unresolved markers). Possible analyzer/registry support for opaque decomposition (FR-4.2).
- **`docs/packets/`** — exports, audits, registries regenerated/re-curated.
- **`libs/atlas-packet`** — wire fixes for any genuine divergences the corrected audit surfaces (packets + byte tests).
- **`services/atlas-channel`, `services/atlas-maps`, `services/atlas-cashshop`, etc.** — only if a surfaced wire fix requires a handler/producer/event change (task-080 B1.1-style multi-service change).
- **`services/atlas-configurations`** — per-version template op-byte maps for newly-routed families (FR-5.2).

## 8. Non-Functional Requirements

- **Correctness over coverage:** no wire change ships on analyzer verdict alone — a per-version byte-level test is the oracle for every fix (task-080's hard-won discipline; it caught false plan premises in B1.2/B1.3 and the three mistraced BuddyInvite exports).
- **Reviewable diffs:** the full re-export will touch all four large JSONs. Ordering must be stable and the change summarized (struct descents / truncations / loop fixes) so a reviewer can audit the *exporter's* correctness, not just trust it.
- **CLAUDE.md verify gates:** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` for every service whose `go.mod` is touched; `tools/redis-key-guard.sh` clean; nesting-cap clean.
- **Multi-tenancy / version gating:** version-divergent fixes use `t.Region() == "GMS" && t.MajorVersion() >= N` symmetrically; >2-version divergences use the region-dispatch idiom (now analyzer-visible via task-080 A5).
- **IDA logistics:** the re-export requires the user to cycle the four IDBs (one loaded at a time; no MCP tool switches them). Harvest is batched per-IDB; subagents can reach IDA-MCP (see memory reference_ida_harvest_subagents).
- **Observability:** the exporter logs unresolved functions explicitly so gaps are visible, not silent.

## 9. Open Questions

- **Exporter implementation surface:** is the export path a Go routine in `tools/packet-audit` that calls IDA-MCP, or a separate harvest script? The descent fix lands wherever the trace is produced — to be confirmed by reading `tools/packet-audit/cmd/export.go` + the harvest path at execution time.
- **Decompiler-label reliability for field semantics:** the IDA decompiler's variable names (e.g. id-vs-password order in login, jobId/level labels) are sometimes unreliable. Where a *semantic* label is ambiguous but the *wire width/order* is clear, the audit only needs width/order — confirm the exporter doesn't gate correctness on uncertain labels.
- **Opaque decomposition depth:** some opaque types (mask-driven `GW_CharacterStat`, `model.Asset`) may be only partially statically decomposable. The boundary between "extend the analyzer to model it" vs. "verify with a byte test + exception" is decided per-type during execution.
- **Template completeness scope creep:** wiring every unrouted JMS family could be large. If a family's full routing is a big task on its own, it may be split into a follow-up — confirm the threshold during design.
- **Full re-export blast radius:** the corrected exports may flip currently-`✅` packets to `❌` where the *old* export coincidentally matched a *wrong* Atlas encoder (two wrongs cancelling). This is no longer an open question — it is governed by the verdict-delta triage gate (§4.7): every such flip is individually IDA-confirmed and dispositioned. The open part is only *how many* there are, which is unknown until the re-audit runs.

## 10. Acceptance Criteria

- [ ] The exporter descends into struct-reading helpers and inlines their reads (FR-1.1); the v83/v87/JMS `OnFriendResult#Invite` re-export now matches the hand-decompiled truth (`…name, [jobId, level if ≥87/JMS], GW_Friend(39), inShop`), not a count-loop or a truncated trace.
- [ ] The exporter captures full read-orders with no truncation (FR-1.2) and disambiguates loops from fixed structs (FR-1.3); unresolved functions emit an explicit marker (FR-2.3) rather than a wrong trace.
- [ ] All four `docs/packets/ida-exports/*.json` are fully re-exported with the fixed exporter, committed with a reviewable structural-change summary (FR-2.1, FR-2.2).
- [ ] The audit is re-run over the corrected exports; the "export read-order truncation/mistrace" residual category is **eliminated** — every former such row is now `✅`, a fixed bug, a verified exclusion, or an explicit unresolved marker (FR-3.1, FR-3.2).
- [ ] Verdict-delta triage (§4.7) was applied: task-080's per-packet verdicts were snapshotted, the post-re-export delta computed, and **every `✅`→`❌` (and new non-`✅`) flip carries a written disposition** — fixed-Atlas-bug, fixed-exporter-gap, or verified-equivalence — each backed by an IDA decompile and (for code changes) a byte-level test. Zero flips rubber-stamped.
- [ ] Every genuine wire divergence surfaced is fixed in-task with per-version byte tests + correct gates (FR-3.3), or registered as a dedicated follow-up if genuinely too large (FR-3.4).
- [ ] The opaque register-boundary types are decomposed-and-verified or carry a byte-test-backed verified exception (FR-4); no type remains in an unexamined "analyzer skipped it" state.
- [ ] The partial per-version templates are completed (newly-routed families audit per version) or carry a verified client-absent verdict (FR-5); all edited templates parse.
- [ ] `_pending.md` (both copies) contain only *verified* exclusions — zero "export was wrong" entries and zero unexamined opaque skips; `TOTAL.md` + `STARTING_A_NEW_VERSION_PASS.md` reflect the corrected baseline and the fixed exporter (FR-6).
- [ ] All CLAUDE.md verify gates pass (test/vet/build, docker bake per touched go.mod, redis-key-guard, nesting cap); closed task-080 items remain green.
- [ ] Code review run (plan-adherence + backend-guidelines) before PR.
- [ ] Net result: a four-version audit baseline where a `✅` genuinely means "wire-correct for that version" and every non-`✅` row has a real, IDA-cited justification — the concrete step from "trusted but input-limited" to genuinely-verified four-version socket read/write support.
