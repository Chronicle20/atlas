# task-065: Combat-Domain Packet Audit — Product Requirements Document

Version: v1
Status: Stub
Created: 2026-05-14
---

## 1. Background

task-027 built the packet-audit pipeline from scratch — analyzer, `tools/packet-audit/` CLI, `EncodeForeign` registry, cycle guard, `candidatesFromFName` mappings — and shipped wire-correct login packets. task-028 applied the same pipeline to the character domain (48 packets), fixing real wire bugs in spawn, damage, buff, skill, and movement packets, cross-verified against GMS v83/v87/v95 and JMS v185 IDA exports. The pipeline is mature: per-packet audit reports land under `docs/packets/audits/gms_v95/`, template opcode drift is caught by comparing `template_gms_*.json` dispatcher case-statement values against IDA, and every fix ships with 4-variant `pt.Variants` round-trip tests covering GMS v28/v83/v95 and JMS v185.

The combat domain — monster, drop, reactor, pet — is the next natural grouping. These packets are exercised in every field map and failure here corrupts moment-to-moment gameplay: monsters spawn through floors, drops disappear silently, reactors fire at wrong positions, pet commands are ignored. task-028 lessons apply directly (dispatcher-layer `characterId` offsets, sub-op enum drift in status/damage packets, loop linearization in fixed-count sequences), and IDA exports for GMS v83, GMS v87, GMS v95, and JMS v185 are available for cross-version gate validation.

Existing tasks touch adjacent monster-behavior logic: task-033/034/035/036 (monster AI and control flows), task-057 (monster movement), task-060/061 (monster data TTL cache and cache invalidation). This audit is wire-shape only — no business logic changes — but coordinate with those tasks to avoid file-level conflicts on monster serverbound handlers.

## 2. Scope

### Packet inventory

| Domain | Clientbound | Serverbound | Total |
|---|---|---|---|
| monster | 9 | 1 | 10 |
| drop | 2 | 1 | 3 |
| reactor | 3 | 1 | 4 |
| pet | 6 | 8 | 14 |
| **Total** | **20** | **11** | **31** |

Counts reflect the actual `libs/atlas-packet/{monster,drop,reactor,pet}/{clientbound,serverbound}/*.go` enumeration. The monster clientbound row includes `MonsterStatSet` and `MonsterStatReset` packed in `stat.go`; the pet clientbound row treats `activated_body.go` as a wrapper of `activated.go` per plan.md §5 Step 1, not a separate packet.

Note: `monster/clientbound/movement.go` and `pet/clientbound/movement.go` have no `_test.go` siblings — they appear to be stub-only files. Flag in `_pending.md` if the audit finds no encoder body to analyze.

### Out of scope

- Business logic in `services/atlas-monsters/`, `services/atlas-drops/`, `services/atlas-pets/`.
- Sub-struct expansion that requires Phase 3 TypeRegistry work beyond what's surfaced during this audit pass.
- Re-auditing login (task-027) or character (task-028) domain packets.
- Atlas-side bare handlers that have no `libs/atlas-packet` decoder — document in `_pending.md`, mirror task-028 treatment.

## 3. Goals

- Produce per-packet audit reports under `docs/packets/audits/gms_v95/` for every packet in the monster, drop, reactor, and pet domains.
- Land real wire-bug fixes with 4-variant `pt.Variants` round-trip tests (v28/v83/v95 + JMS v185).
- Append character-domain IDA-export entries to `docs/packets/ida-exports/gms_v95.json` and the matching `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` files.
- Verify template opcodes against IDA dispatcher case-statements for every new template entry touched.
- Defer sub-op enum drift (monster status sub-types, pet command varieties) to `_pending.md` per design §9 effect-family pattern.
- Defer bare handlers (no atlas-packet decoder) to `_pending.md` per task-028 design §1.

## 4. Functional Requirements

### 4.1 Coverage matrix

Produce a verdict row in `docs/packets/audits/gms_v95/SUMMARY.md` for each of the 31 packets:

**monster/clientbound (9):** `control`, `damage`, `destroy`, `health`, `movement_ack`, `movement`, `spawn`, `stat` (× StatSet + StatReset)  
**monster/serverbound (1):** `movement`  
**drop/clientbound (2):** `destroy`, `spawn`  
**drop/serverbound (1):** `pick_up`  
**reactor/clientbound (3):** `destroy`, `hit`, `spawn`  
**reactor/serverbound (1):** `hit`  
**pet/clientbound (6):** `activated`, `cash_food_result`, `chat`, `command`, `exclude`, `movement` (`activated_body.go` is a wrapper of `activated.go`)  
**pet/serverbound (8):** `chat`, `command`, `drop_pick_up`, `exclude_item`, `food`, `item_use`, `movement`, `spawn`

Each row: FName, IDA function address, verdict (✅ / ⚠️ / ❌), notes with file:line citation or `_pending.md` reference.

### 4.2 IDA exports

Populate combat-domain FName entries in `docs/packets/ida-exports/gms_v95.json` for all 31 packets. During the cross-version pass, add matching entries to `gms_v83.json`, `gms_v87.json`, and `gms_jms_185.json`. Use the existing entry schema (op list with `Decode1/2/4/Str/Buffer/Loop` operations + guard expressions).

### 4.3 Wire bug fixes

Every ❌ verdict gets a fix in `libs/atlas-packet/<domain>/{clientbound,serverbound}/*.go`, gated on the affected versions only. Each fix lands with:

- A 4-variant test sweep using `pt.Variants` (covers GMS v28/v83/v95 + JMS v185).
- A 1-3 sentence comment citing the IDA function name and specific finding.
- A row in `post-phase-b.md`'s "Real wire bugs fixed" table.

### 4.4 Template fixes

Template drift gets fixed in `services/atlas-configurations/seed-data/templates/template_gms_*.json` and `template_jms_185_1.json`. Each fix lands with a row in `post-phase-b.md`'s "Template opcode/enum fixes" table and a commit message citing the IDA case-statement value.

### 4.5 TypeRegistry extensions

When the analyzer surfaces an unregistered sub-struct type, extend `tools/packet-audit/internal/atlaspacket/registry.go`. Likely new types for this domain:

- Monster spawn payload sub-structs (position, movement, controller).
- Pet `activated_body.go` item-reference sub-struct.
- Drop spawn/destroy coordinate blocks.

Each registry addition commits with a tagged test in `registry_test.go`.

### 4.6 Cross-version re-verification

After the v95 pass, user loads v83 IDA and verifies each fixed file's `Region/MajorVersion` gate. Repeat for v87, then JMS v185. Widen or narrow gates as IDA evidence demands. Record per-version notes in `post-phase-b.md`.

## 5. API Surface

No new HTTP/REST endpoints. The audit produces:

- Updated `tools/packet-audit/` TypeRegistry entries.
- New per-packet audit reports under `docs/packets/audits/gms_v95/monster/`, `drop/`, `reactor/`, `pet/`.
- Updated `docs/packets/ida-exports/gms_{v83,v87,v95}.json` and `gms_jms_185.json`.
- Updated `template_*.json` opcode/enum values for any drift found.

## 6. Data Model

No persistent-data changes. `services/atlas-configurations/seed-data/templates/` JSON files receive opcode corrections only. No migration needed.

## 7. Service Impact

| Service / Lib | Impact |
|---|---|
| `libs/atlas-packet/monster/` | Wire-bug fixes in clientbound/serverbound encoders. Tests per variant. |
| `libs/atlas-packet/drop/` | Wire-bug fixes. Tests per variant. |
| `libs/atlas-packet/reactor/` | Wire-bug fixes. Tests per variant. |
| `libs/atlas-packet/pet/` | Wire-bug fixes, including self vs foreign movement variants (mirrors character pattern). Tests per variant. |
| `tools/packet-audit/` | TypeRegistry additions for combat sub-structs. |
| `services/atlas-configurations/seed-data/templates/` | Opcode/enum corrections. |
| `docs/packets/ida-exports/` | New/refreshed export JSON per version. |

## 8. Non-Functional / Quality Bar

- All fixes ship with 4-variant test sweeps using `pt.Variants` (GMS v28/v83/v95 + JMS v185).
- Hard cap: no encoder/decoder grows beyond 2 nested region/version guards (per task-028 design §7). 3+ nested → STOP, log to `_pending.md`.
- gitleaks scrub clean before PR (no `/home/<user>/` absolute paths in audit reports).
- `go test -race ./libs/atlas-packet/... ./tools/packet-audit/...` clean.
- Login (task-027) and character (task-028) verdicts unchanged.

## 9. Working Assumptions

- IDA v95 is the primary target; v83/v87/JMS v185 IDA available for cross-version pass when user loads them.
- Templates `template_gms_{12,28,83,87,92,95}_1.json` and `template_jms_185_1.json` exist; new opcodes added per audit.
- `monster/clientbound/movement.go` and `pet/clientbound/movement.go` lack test files — they may be stubs with no encoder body; audit surface and defer if body is absent.
- Pet has self vs foreign variants like `character/clientbound/spawn.go` — expect `pet/clientbound/activated.go` to follow the same dispatcher-offset pattern (client prepends `ownerCharacterId`).
- Monster spawn/despawn opcodes are version-sensitive; v95 likely diverges from v83 on the `movement` control channel structure.

## 10. Key Risks / Patterns to Watch For (from task-027/028 lessons)

- **Dispatcher-layer offset**: CUserPool/CUserRemote dispatchers prepend `characterId` before routing — atlas wire includes it at offset 0. Monster and pet packets that encode "owner" IDs are susceptible.
- **Sub-op enum drift**: Monster status messages and pet command packets use a leading-byte sub-op dispatch (effect-family pattern). These are not modeled by the analyzer; defer to `_pending.md`.
- **`EncodeMask` / sub-struct method calls**: appear as one analyzer call but emit multiple bytes — ack as tool-limitation.
- **Loop linearization**: Monster spawn may include a fixed-count loop over movement data — flattens incorrectly in the analyzer; ack as tool-limitation.
- **Dispatcher case-statement validation**: every new template opcode MUST be confirmed against IDA dispatcher decompile before commit. Do not guess.
- **Cross-version gate boundaries**: don't assume the gate boundary until cross-version IDA confirms. v83 may lack fields that v87 also lacks.
- **Hidden constructor-signature ripples**: adding fields to encoder structs ripples to `atlas-channel`, `atlas-monsters`, `atlas-pets` handlers — verify build clean across services.
- **Audit-report ack footers**: add ack footer AFTER the final audit run; if re-running, `git checkout HEAD -- <report.md>` to revert.
- **Coordination with task-033/034/035/036/057/060/061**: these tasks touch monster handler code. Wire-only fixes here should not conflict, but coordinate on branch ordering if parallel.

## 11. Acceptance Criteria

- [ ] All 31 listed packets have audit reports under `docs/packets/audits/gms_v95/`.
- [ ] Every ❌ has either a fix commit OR a `_pending.md` row.
- [ ] All 4 verification commands pass cleanly: `go build ./...`, `go vet ./libs/atlas-packet/...`, `go test -race ./libs/atlas-packet/...`, `go test -race ./tools/packet-audit/...`.
- [ ] gitleaks scrub clean.
- [ ] `post-phase-b.md` ledger written (packets audited, wire bugs fixed, template fixes, cross-version notes, deferred items).
- [ ] plan-adherence-reviewer + backend-guidelines-reviewer dispatched and findings addressed before PR.

## 12. Dependencies

- task-027 (login audit, analyzer baseline) — merged.
- task-028 (character audit, EncodeForeign registry, cycle guard, ack pattern) — merged.

## 13. Open Questions

- Are there v95-only handlers in the monster or pet domain that have no v83 equivalent? Cross-version pass will reveal.
- `pet/serverbound/` has 16 files — is this driven by sub-op dispatch needing individual files per command type? If so, the analyzer may surface all as one logical packet with a leading discriminator byte. Audit will clarify scope of `_pending.md` deferrals.
- Coordinate with task-033/034/035/036 on file-level ordering: which branches land first?
