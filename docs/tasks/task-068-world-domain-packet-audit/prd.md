# World-Domain Packet Audit — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-15
---

## 1. Overview

task-027 established the packet-audit pipeline (analyzer, `tools/packet-audit/` CLI, `EncodeForeign` registry, cycle guard, `candidatesFromFName` mappings) and shipped wire-correct login packets. task-028 applied the same pipeline to the character domain (48 packets), delivering per-packet audit reports under `docs/packets/audits/gms_v95/`, IDA-export JSON for four cross-version targets, and 4-variant `pt.Variants` round-trip tests. Both tasks have shipped to `main` and the pipeline is mature.

The world domain — `field`, `portal`, `npc` — is the structural backbone of every map transition and NPC interaction in the game. Wire bugs here are immediately user-visible: a corrupted `WarpToMap` drops the client to a black screen; a malformed `SetField` sends the character to the wrong map; an NPC conversation packet with a wrong text-type byte renders as garbled dialog. The field domain also carries the highest rate of sub-op type diversity — clock types, weather effect types, kite types, transport route IDs — which mirrors the sub-op risk pattern seen in the character effect family.

GMS v83, v87, v95, and JMS v185 IDA exports are available for cross-version gate validation. No prior tasks have targeted the world domain specifically. The `portal/` domain is tiny (0 clientbound, 2 serverbound — a portal-script trigger and test) and should be a quick pass. The `field/` and `npc/` domains have more depth. `instance-based-transports` has already merged to `main` (multiple merge commits, including `456ec8717` and `05bdfd7b0`), so `field/clientbound/transport.go` is in-place and audited as part of this task with no coordination overhead.

## 2. Goals

Primary goals:

- Produce per-packet audit reports under `docs/packets/audits/gms_v95/` for every packet in the `field`, `portal`, and `npc` domains (57 files total).
- Land real wire-bug fixes with 4-variant `pt.Variants` round-trip tests (GMS v28 / v83 / v95 + JMS v185).
- Append world-domain IDA-export entries to `docs/packets/ida-exports/gms_v95.json` and, during the cross-version pass, the matching `gms_v83.json`, `gms_v87.json`, and `gms_jms_185.json` files.
- Verify template opcodes against IDA dispatcher case-statements for every new template entry touched.
- Extend `tools/packet-audit/internal/atlaspacket/registry.go` with world sub-struct types (`SetField` map-header, `WarpToMap` coordinate block, NPC shop item entry, NPC conversation text-type encoding, clock time-of-day) as the analyzer surfaces them.

Non-goals:

- Business logic in `services/atlas-channel/` beyond the minimum needed to wire a fix through.
- Re-auditing login (task-027) or character (task-028) packets.
- Sub-op enum expansion for field effect type-discriminators where the analyzer cannot statically resolve the discriminator byte — defer to `_pending.md` per task-028 design §9.
- Atlas-side bare handlers that have no `libs/atlas-packet` decoder — document in `_pending.md`, mirror task-028 treatment.
- NPC scripting engine (`atlas-npcs` service) business logic — only atlas-packet wire shapes are in scope.
- Refactoring `npc/clientbound/conversation.go` into one file per dialog type. Per-type variants are audited within the existing monolithic file (see §4.5).

## 3. User Stories

- As a game developer, I want every world-domain packet validated against IDA so that map transitions and NPC interactions render correctly on every supported client version (v28/v83/v95/JMS v185).
- As an Atlas maintainer, I want a verdict matrix for the world domain so that I can see at a glance which packets are wire-correct, which have known drift, and which are deferred with rationale.
- As a future audit-pipeline user, I want world-domain sub-struct types registered in `tools/packet-audit/` so that subsequent audits (mob, item, party, etc.) inherit clean type resolution.
- As a release engineer, I want template opcode drift caught in this audit so that template JSON files match IDA dispatcher case-statements per region/version.
- As an IDA cross-version verifier, I want a single batched verification window per non-v95 IDA load (v83, v87, JMS v185) so that I'm not context-switching IDA databases per packet.

## 4. Functional Requirements

### 4.1 Coverage matrix

Produce a verdict row in `docs/packets/audits/gms_v95/SUMMARY.md` for each of the 57 packets:

- **field/clientbound (21):** `affected_area_created`, `affected_area_removed`, `clock`, `effect`, `effect_weather`, `kite_destroy`, `kite_error`, `kite_spawn`, `set_field`, `transport`, `warp_to_map`, plus the test-companion shapes implicit in this directory. The audit MUST enumerate the directory at audit time and produce one row per non-test `.go` file.
- **field/serverbound (2):** `change` (and its test companion); enumerate at audit time.
- **portal/serverbound (2):** `script` (portal-script trigger) plus its test companion.
- **npc/clientbound (14):** `action`, `conversation`, `guide_talk`, `shop_list`, `shop_operation`, `shop_operation_body`, `spawn`, `spawn_request_controller`, plus test companions.
- **npc/serverbound (18):** `action`, `continue_conversation`, `continue_conversation_selection`, `continue_conversation_text`, `shop`, `shop_buy`, `shop_recharge`, `shop_sell`, `start_conversation`, plus test companions. Enumerate at audit time.

Each row records: FName, IDA function address, verdict (✅ / ⚠️ / ❌), notes with `file:line` citation OR a reference to a `_pending.md` row.

### 4.2 IDA exports

Populate world-domain FName entries in `docs/packets/ida-exports/gms_v95.json` for all 57 packets during the v95 pass. During the §4.7 cross-version pass, add matching entries to `gms_v83.json`, `gms_v87.json`, and `gms_jms_185.json`. Use the existing entry schema (FName, address, decompile excerpt).

### 4.3 Wire bug fixes

Every ❌ verdict gets a fix in `libs/atlas-packet/<domain>/{clientbound,serverbound}/*.go`, gated on affected versions only. Each fix lands with:

- A 4-variant test sweep using `pt.Variants` (GMS v28 / v83 / v95 + JMS v185).
- A 1–3 sentence comment citing the IDA function name and the specific finding.
- A row in `post-phase-b.md`'s "Real wire bugs fixed" table.

### 4.4 Template fixes

Template opcode/enum drift gets fixed in `services/atlas-configurations/seed-data/templates/template_gms_*.json` and `template_jms_185_1.json`. Each fix lands with:

- A row in `post-phase-b.md`'s "Template opcode/enum fixes" table.
- A commit message citing the IDA dispatcher case-statement value.

### 4.5 NPC conversation — audit shape per dialog type

`npc/clientbound/conversation.go` (360 lines) is monolithic: every dialog type (`say` / `askText` / `askYesNo` / `askMenu` / `askNumber` / `askAvatar` / `askPet` / `askBoxText`) is encoded behind a leading text-type byte discriminator. This task does NOT refactor that file. Instead:

- Produce ONE audit report for `conversation.go` that contains per-dialog-type sections, each tracing the corresponding IDA dispatcher branch.
- Each per-type section gets its own verdict (✅ / ⚠️ / ❌) and, when ❌, its own wire-fix block.
- Branches the analyzer cannot statically resolve (e.g., text-type byte assembled from a runtime parameter) defer to `_pending.md` with a one-line rationale; the verdict row in `SUMMARY.md` marks the file ⚠️.
- 4-variant `pt.Variants` tests cover each fixed dialog-type branch independently.

### 4.6 `set_field` nesting policy

`field/clientbound/set_field.go` is the map-initialization envelope and is highly version-sensitive (already 8 region/version references). The task-028 design §7 cap of **2 nested region/version guards per encoder** is RELAXED to **3 deep** for `set_field.go` ONLY. This exception is:

- Called out explicitly in `design.md` §7.
- Documented in the audit report header.
- Not generalized — every other encoder (NPC, portal, the rest of field) remains under the 2-deep cap. Guards beyond 3 in `set_field.go` still defer to `_pending.md`.

### 4.7 TypeRegistry extensions

When the analyzer surfaces an unregistered sub-struct type, extend `tools/packet-audit/internal/atlaspacket/registry.go`. Likely additions for this domain:

- `SetField` map-header sub-struct (map ID, portal ID, spawn point).
- `WarpToMap` destination coordinate block.
- NPC shop item entry sub-struct in `shop_list`.
- NPC conversation text-type sub-encoders (one per dialog type, per §4.5).
- Clock time-of-day sub-struct.

Each registry addition commits with a tagged test in `registry_test.go`.

### 4.8 Cross-version re-verification — after full v95 pass

The cross-version pass runs **after** every v95 audit and fix has landed. The cadence is:

1. v95 pass complete (all 57 reports + all fixes + `post-phase-b.md` initial draft).
2. User loads v83 IDA. For every file fixed in step 1, verify the `Region/MajorVersion` gate matches v83 behavior. Add v83 IDA-export entries.
3. Repeat for v87.
4. Repeat for JMS v185.
5. Record per-version notes in `post-phase-b.md`.

This batches IDA context-switching to three discrete windows rather than per-packet hops.

## 5. API Surface

No new HTTP/REST endpoints. The audit produces:

- Updated `tools/packet-audit/` TypeRegistry entries.
- New per-packet audit reports under `docs/packets/audits/gms_v95/field/`, `portal/`, `npc/`.
- Updated `docs/packets/ida-exports/gms_{v83,v87,v95}.json` and `gms_jms_185.json`.
- Updated `template_*.json` opcode/enum values for any drift found.

## 6. Data Model

No persistent-data changes. `services/atlas-configurations/seed-data/templates/` JSON files receive opcode corrections only.

## 7. Service Impact

| Service / Lib | Impact |
|---|---|
| `libs/atlas-packet/field/` | Wire-bug fixes in clientbound/serverbound encoders. Tests per variant. `set_field` allowed 3-deep nesting; all other encoders stay 2-deep. |
| `libs/atlas-packet/portal/` | Wire-bug fixes (small domain — 2 serverbound packets). Tests per variant. |
| `libs/atlas-packet/npc/` | Wire-bug fixes. Tests per variant. `conversation.go` audited per dialog type within the existing monolithic file. Shop-list bounds verified against IDA. |
| `tools/packet-audit/` | TypeRegistry additions for world sub-structs; new entries in `registry_test.go`. |
| `services/atlas-configurations/seed-data/templates/` | Opcode/enum corrections; commits cite IDA dispatcher case-statement values. |
| `docs/packets/ida-exports/` | New/refreshed export JSON per version (v83/v87/v95/JMS v185). |
| `docs/packets/audits/gms_v95/` | 57 per-packet audit reports + updated `SUMMARY.md`. |

## 8. Non-Functional Requirements

- All fixes ship with 4-variant test sweeps using `pt.Variants` (GMS v28 / v83 / v95 + JMS v185).
- Nesting cap: **2 nested region/version guards per encoder**, with `set_field.go` as the documented sole exception at **3 deep**. 4+ → STOP, defer to `_pending.md`.
- gitleaks scrub clean before PR (no `/home/<user>/` absolute paths or other path leakage in audit reports).
- `go test -race ./libs/atlas-packet/... ./tools/packet-audit/...` clean.
- `go build ./...` clean across every changed module.
- `go vet ./libs/atlas-packet/...` clean.
- Login (task-027) and character (task-028) verdicts unchanged — the SUMMARY rows for those domains MUST be byte-identical before vs. after this task.
- Audit-report ack footers added AFTER the final audit run for each file. Re-running an audit requires `git checkout HEAD -- <report.md>` before re-execution.

## 9. Open Questions

- `field/clientbound/set_field.go` current gate boundaries: the 8 existing `Region`/`MajorVersion` references may already exceed the 3-deep cap once dispatcher decompile is overlaid. If so, the design must flag which sub-sections defer to `_pending.md` before plan-phase. (Defer this discovery to phase 2.)
- JMS v185 field-format divergence: whether the existing `Region() == "JMS"` gating covers v185 spawn-point / field-limit encoding differences, or whether a third gate dimension (region × major-version) is required. (Discovery occurs in §4.8 step 4.)
- Whether `npc/clientbound/conversation.go`'s text-type byte is statically resolvable per branch — if not, the analyzer will surface ⚠️ verdicts for the unresolvable branches; quantification deferred to phase 2.

## 10. Acceptance Criteria

- [ ] All 57 listed packet files have audit reports under `docs/packets/audits/gms_v95/`.
- [ ] `npc/clientbound/conversation.go` audit report contains a per-dialog-type section for every text-type branch (say / askText / askYesNo / askMenu / askNumber / askAvatar / askPet / askBoxText) with its own verdict.
- [ ] Every ❌ has either a fix commit OR a `_pending.md` row.
- [ ] `set_field.go` does not exceed 3 nested region/version guards; every other encoder does not exceed 2.
- [ ] All 4 verification commands pass cleanly:
  - `go build ./...`
  - `go vet ./libs/atlas-packet/...`
  - `go test -race ./libs/atlas-packet/...`
  - `go test -race ./tools/packet-audit/...`
- [ ] gitleaks scrub clean.
- [ ] `docs/packets/ida-exports/gms_v95.json` populated for all 57 packets; v83/v87/JMS v185 exports populated per §4.8 after the v95 pass.
- [ ] `post-phase-b.md` ledger written, with "Real wire bugs fixed", "Template opcode/enum fixes", and per-version cross-verification notes.
- [ ] plan-adherence-reviewer + backend-guidelines-reviewer dispatched and findings addressed before PR.
- [ ] Login (task-027) and character (task-028) SUMMARY rows byte-identical to pre-task state.

## 11. Dependencies

- task-027 (login audit, analyzer baseline) — merged.
- task-028 (character audit, `EncodeForeign` registry, cycle guard, ack pattern) — merged.
- `instance-based-transports` branch (`field/clientbound/transport.go`) — merged to `main` (`456ec8717`, `05bdfd7b0`, plus follow-ups). No coordination required; `transport.go` is audited in-place as part of this task.

## 12. Key Risks / Patterns to Watch For (from task-027/028 lessons)

- **Field effect sub-op enum drift**: `effect.go` and `effect_weather.go` likely use a leading-byte type discriminator. Same pattern as `character/clientbound/effect.go` from task-028 — defer per design §9 if the discriminator is not encoded per-file.
- **`SetField` complexity**: map-init envelope with many version-sensitive fields (channel number format, field limits, mobility data). 3-deep nesting allowed; anything beyond defers to `_pending.md`.
- **Dispatcher-layer offset**: NPC action/conversation packets routed through CUserPool may have `characterId` prepended before the NPC-specific payload — atlas wire includes it at offset 0.
- **`EncodeMask` / sub-struct method calls**: NPC shop items appear as sub-struct calls — ack as tool-limitation.
- **Loop linearization**: NPC shop list item arrays flatten incorrectly — ack and verify bounds against IDA.
- **Dispatcher case-statement validation**: every new template opcode MUST be confirmed against IDA dispatcher decompile before commit.
- **Cross-version gate boundaries**: `SetField` and `WarpToMap` are well-known to differ between v83 and v95 (field-limits encoding, spawn-point format). Don't assume gate boundaries until IDA confirms.
- **Hidden constructor-signature ripples**: field/npc encoder struct changes ripple to `atlas-channel` warp/spawn handlers — verify build clean across services.
- **Audit-report ack footers**: add ack footer AFTER the final audit run; if re-running, `git checkout HEAD -- <report.md>` to revert.
- **Conversation per-type audit**: `conversation.go` is monolithic by design for this task — per-type sections live inside the single audit report. Do NOT refactor the encoder file.
