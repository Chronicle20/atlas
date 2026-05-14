# task-069: Misc-Domain Packet Audit — Product Requirements Document

Version: v1
Status: Stub
Created: 2026-05-14
---

## 1. Background

task-027 established the packet-audit pipeline (analyzer, `tools/packet-audit/` CLI, `EncodeForeign` registry, cycle guard, `candidatesFromFName` mappings) and shipped wire-correct login packets. task-028 applied the same pipeline to the character domain (48 packets), delivering per-packet audit reports, IDA-export JSON for four cross-version targets, 4-variant `pt.Variants` round-trip tests, and template opcode drift fixes. Both tasks have shipped to main. The social (task-066), commerce (task-067), world (task-068), and combat (task-065) tasks cover the remaining large domains.

This task sweeps the small remaining domains that don't fit a single gameplay theme: account, fame, stat, ui, socket, channel, merchant (the merchant employee-shop packets, distinct from the commerce-domain hire-merchant), tool (infrastructure), and quest. Together these total ~50 packets. These are the "leftover" domains — individually small, but wiring bugs here are still operationally relevant: a malformed account TOS-accept packet breaks first-login flows; a wrong socket handshake byte breaks all new connections; a quest action packet mismatch corrupts quest state for every player.

GMS v83, v87, v95, and JMS v185 IDA exports are available for cross-version gate validation. Several of these domains have prior task cross-references: task-014 (conversation reward notices), task-015 (quest start reward notices), and task-023 (quest selected-skill gate) touched quest packet handling. Coordinate to avoid conflicting changes on quest files.

## 2. Scope

### Packet inventory

| Domain | Clientbound | Serverbound | Total |
|---|---|---|---|
| account | 0 | 6 | 6 |
| fame | 2 | 2 | 4 |
| stat | 2 | 0 | 2 |
| ui | 6 | 0 | 6 |
| socket | 4 | 6 | 10 |
| channel | 2 | 2 | 4 |
| merchant | 2 | 1 | 3 |
| tool | 0 | 0 | 0 |
| quest | 2 | 12 | 14 |
| **Total** | **20** | **29** | **49** |

Note: `tool/` has no `.go` files under `clientbound/` or `serverbound/` in the current codebase — it contains only `uint128.go` and `uint128_test.go` (utility types) plus a `serverbound/` subdirectory. Confirm at audit time whether tool serverbound is populated or empty. `account/clientbound/` also has no files — the domain is serverbound-only.

### Out of scope

- Business logic in `services/atlas-account/`, `services/atlas-channel/`, `services/atlas-quest/`, etc.
- Re-auditing login (task-027) or character (task-028) domain packets.
- Sub-struct expansion that requires Phase 3 TypeRegistry work beyond what is surfaced during this audit pass.
- Atlas-side bare handlers that have no `libs/atlas-packet` decoder — document in `_pending.md`, mirror task-028 treatment.
- Full hire-merchant packet audit (those are `interaction/` domain, handled by task-067).

## 3. Goals

- Produce per-packet audit reports under `docs/packets/audits/gms_v95/` for every packet in the account, fame, stat, ui, socket, channel, merchant, tool, and quest domains.
- Land real wire-bug fixes with 4-variant `pt.Variants` round-trip tests (v28/v83/v95 + JMS v185).
- Append misc-domain IDA-export entries to `docs/packets/ida-exports/gms_v95.json` and the matching `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` files.
- Verify template opcodes against IDA dispatcher case-statements for every new template entry touched.
- Defer sub-op enum drift to `_pending.md` per design §9.
- Defer bare handlers to `_pending.md` per task-028 design §1.
- Confirm whether `tool/` constitutes a real packet domain or is a utility-only helper package; document finding in `post-phase-b.md`.

## 4. Functional Requirements

### 4.1 Coverage matrix

Produce a verdict row in `docs/packets/audits/gms_v95/SUMMARY.md` for each of the ~49 packets:

**account/serverbound (6):** `accept_tos`, `register_pin`, `set_gender`, plus any remaining files  
**fame/clientbound (2), fame/serverbound (2):** (enumerate from `libs/atlas-packet/fame/`)  
**stat/clientbound (2):** (enumerate from `libs/atlas-packet/stat/clientbound/`)  
**ui/clientbound (6):** (enumerate from `libs/atlas-packet/ui/clientbound/`)  
**socket/clientbound (4):** `hello`, `ping`; `socket/serverbound (6)`: `channel_connect`, `pong`, `start_error`, plus any remaining  
**channel/clientbound (2), channel/serverbound (2):** (enumerate from `libs/atlas-packet/channel/`)  
**merchant/clientbound (2), merchant/serverbound (1):** (enumerate from `libs/atlas-packet/merchant/`)  
**tool/**: Confirm whether serverbound files exist. If `tool/serverbound/` is empty, document as "no packets; utility-only package" in `_pending.md`.  
**quest/clientbound (2), quest/serverbound (12):** `action`, `action_complete`, `action_restore_lost_item`, `action_script_end`, `action_script_start`, `action_start`, plus any remaining

Each row: FName, IDA function address, verdict (✅ / ⚠️ / ❌), notes with file:line citation or `_pending.md` reference.

### 4.2 IDA exports

Populate misc-domain FName entries in `docs/packets/ida-exports/gms_v95.json` for all ~49 packets. During the cross-version pass, add matching entries to `gms_v83.json`, `gms_v87.json`, and `gms_jms_185.json`. Use the existing entry schema.

### 4.3 Wire bug fixes

Every ❌ verdict gets a fix in `libs/atlas-packet/<domain>/{clientbound,serverbound}/*.go`, gated on affected versions only. Each fix lands with:

- A 4-variant test sweep using `pt.Variants` (GMS v28/v83/v95 + JMS v185).
- A 1-3 sentence comment citing the IDA function name and finding.
- A row in `post-phase-b.md`'s "Real wire bugs fixed" table.

### 4.4 Template fixes

Template drift gets fixed in `services/atlas-configurations/seed-data/templates/template_gms_*.json` and `template_jms_185_1.json`. Each fix lands with a row in `post-phase-b.md`'s "Template opcode/enum fixes" table and a commit message citing the IDA case-statement value.

### 4.5 TypeRegistry extensions

When the analyzer surfaces an unregistered sub-struct type, extend `tools/packet-audit/internal/atlaspacket/registry.go`. Likely new types for this domain:

- Quest reward sub-struct in `quest/clientbound/` (shared with task-014/015 reward notices — coordinate type definition if those tasks added a TypeRegistry entry).
- Socket handshake version-info sub-struct in `socket/clientbound/hello`.
- Channel migrate address block in `channel/clientbound/`.

Each registry addition commits with a tagged test in `registry_test.go`.

### 4.6 Cross-version re-verification

After the v95 pass, user loads v83 IDA and verifies each fixed file's `Region/MajorVersion` gate. Repeat for v87, then JMS v185. Record per-version notes in `post-phase-b.md`.

### 4.7 Task-domain confirmation

At the start of audit, run `find libs/atlas-packet/tool -name '*.go' | grep -v _test | sort` to confirm whether `tool/` has any packet encoder/decoder files. Document the finding in `post-phase-b.md`. If hidden domains are discovered (e.g., a subdirectory that doesn't appear in `ls libs/atlas-packet/`), enumerate them and expand scope accordingly.

## 5. API Surface

No new HTTP/REST endpoints. The audit produces:

- Updated `tools/packet-audit/` TypeRegistry entries.
- New per-packet audit reports under `docs/packets/audits/gms_v95/account/`, `fame/`, `stat/`, `ui/`, `socket/`, `channel/`, `merchant/`, `tool/`, `quest/`.
- Updated `docs/packets/ida-exports/gms_{v83,v87,v95}.json` and `gms_jms_185.json`.
- Updated `template_*.json` opcode/enum values for any drift found.

## 6. Data Model

No persistent-data changes. `services/atlas-configurations/seed-data/templates/` JSON files receive opcode corrections only.

## 7. Service Impact

| Service / Lib | Impact |
|---|---|
| `libs/atlas-packet/account/` | Wire-bug fixes in serverbound encoders. Tests per variant. |
| `libs/atlas-packet/fame/` | Wire-bug fixes. Tests per variant. |
| `libs/atlas-packet/stat/` | Wire-bug fixes. Tests per variant. |
| `libs/atlas-packet/ui/` | Wire-bug fixes. Tests per variant. |
| `libs/atlas-packet/socket/` | Wire-bug fixes in handshake/ping/pong encoders. Particularly sensitive — any socket hello change affects ALL clients. Tests per variant. |
| `libs/atlas-packet/channel/` | Wire-bug fixes. Tests per variant. |
| `libs/atlas-packet/merchant/` | Wire-bug fixes (employee-shop variants, not hire-merchant). Tests per variant. |
| `libs/atlas-packet/quest/` | Wire-bug fixes. Tests per variant. Coordinate with task-014/015/023. |
| `tools/packet-audit/` | TypeRegistry additions for misc sub-structs. |
| `services/atlas-configurations/seed-data/templates/` | Opcode/enum corrections. |
| `docs/packets/ida-exports/` | New/refreshed export JSON per version. |

## 8. Non-Functional / Quality Bar

- All fixes ship with 4-variant test sweeps using `pt.Variants` (GMS v28/v83/v95 + JMS v185).
- Hard cap: no encoder/decoder grows beyond 2 nested region/version guards (per task-028 design §7). 3+ nested → STOP, log to `_pending.md`.
- gitleaks scrub clean before PR (no `/home/<user>/` absolute paths in audit reports).
- `go test -race ./libs/atlas-packet/... ./tools/packet-audit/...` clean.
- Login (task-027) and character (task-028) verdicts unchanged.
- **Extra caution on socket packets**: `socket/clientbound/hello` and `socket/serverbound/pong` are on the critical path for every client connection — any wire change here must be verified against ALL version templates before commit.

## 9. Working Assumptions

- IDA v95 is the primary target; v83/v87/JMS v185 IDA available for cross-version pass when user loads them.
- Templates `template_gms_{12,28,83,87,92,95}_1.json` and `template_jms_185_1.json` exist; new opcodes added per audit.
- `tool/` is a utility-only package (uint128 type), not a packet domain — expect 0 audit rows for tool. Confirm at runtime.
- `socket/` handshake packets (`hello`, `ping`, `pong`) are the most structurally stable in the library (few version branches) but mistakes here break ALL clients. Extra verification step before any socket fix lands.
- Quest serverbound packets (`action`, `action_complete`, `action_script_*`, `action_start`, `action_restore_lost_item`) have version-sensitive reward-encoding fields touched by task-014/015/023. Read those tasks' commit history before modifying quest files.
- `stat/clientbound/` has only 2 files — likely a very quick pass. `ui/clientbound/` has 6 files; these are UI notification packets (stat up/down popups, UI state changes) — expect minimal version branching.

## 10. Key Risks / Patterns to Watch For (from task-027/028 lessons)

- **Socket handshake sensitivity (primary risk)**: Any wire change to `socket/clientbound/hello` or `socket/serverbound/pong` breaks every client. Validate the fix against all 6 version templates before commit. Build clean for atlas-login AND atlas-channel before PR.
- **Quest file conflict with task-014/015/023**: These tasks added quest reward field changes. Check their commit history; if they introduced `Region/MajorVersion` gates, don't widen or narrow those gates without IDA evidence from the same version context used in those tasks.
- **Dispatcher-layer offset**: Account serverbound (pin registration, TOS acceptance) packets may have `characterId` or `accountId` prepended by the dispatcher. Verify offset at offset 0 before assuming payload starts at byte 0.
- **`EncodeMask` / sub-struct method calls**: Quest reward sub-structs appear as one analyzer call — ack as tool-limitation.
- **Dispatcher case-statement validation**: every new template opcode MUST be confirmed against IDA dispatcher decompile before commit.
- **Cross-version gate boundaries**: Socket handshake version fields differ between v83 and v95 (encryption, IV seeds). Don't assume the gate boundary until cross-version IDA confirms.
- **Hidden domains**: Run `find libs/atlas-packet -maxdepth 1 -type d | sort` at audit start and cross-reference against the domain list in this PRD. Any directory not listed here that contains `*.go` files should be investigated and either added to scope (with team agreement) or documented as out-of-scope in `_pending.md`.
- **Hidden constructor-signature ripples**: socket/account/quest encoder struct changes ripple to `atlas-login`, `atlas-channel`, `atlas-account` handlers — verify build clean across services.
- **Audit-report ack footers**: add ack footer AFTER the final audit run; if re-running, `git checkout HEAD -- <report.md>` to revert.

## 11. Acceptance Criteria

- [ ] All ~49 listed packet files have audit reports under `docs/packets/audits/gms_v95/`.
- [ ] `tool/` domain is investigated and confirmed as utility-only or expanded if packet files are found.
- [ ] Every ❌ has either a fix commit OR a `_pending.md` row.
- [ ] All 4 verification commands pass cleanly: `go build ./...`, `go vet ./libs/atlas-packet/...`, `go test -race ./libs/atlas-packet/...`, `go test -race ./tools/packet-audit/...`.
- [ ] gitleaks scrub clean.
- [ ] `post-phase-b.md` ledger written (includes tool-domain confirmation note).
- [ ] plan-adherence-reviewer + backend-guidelines-reviewer dispatched and findings addressed before PR.

## 12. Dependencies

- task-027 (login audit, analyzer baseline) — merged.
- task-028 (character audit, EncodeForeign registry, cycle guard, ack pattern) — merged.
- task-014 (conversation reward notices) — merged; check quest packet changes.
- task-015 (quest start reward notices) — merged; check quest packet changes.
- task-023 (quest selected-skill gate) — merged; check quest packet changes.

## 13. Open Questions

- `tool/`: does `libs/atlas-packet/tool/serverbound/` contain packet files, or is the directory empty? If non-empty, what do those packets encode?
- Are there any hidden domains under `libs/atlas-packet/` not captured in this PRD? Run `find libs/atlas-packet -maxdepth 1 -type d` at execution time to verify the full list.
- After tasks 065-069 all complete, will the `docs/packets/audits/gms_v95/SUMMARY.md` be a complete audit of the entire `libs/atlas-packet/` library? Or are there domains in the library not covered by tasks 027/028/065/066/067/068/069? A post-069 sweep should confirm 100% domain coverage.
- Should task-069 produce a final "complete library audit ledger" that cross-references all 7 audit tasks (027, 028, 065-069) into a single TOTAL.md? Or is the cumulative SUMMARY.md sufficient?
