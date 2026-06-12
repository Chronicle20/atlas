# task-068: World-Domain Packet Audit — Product Requirements Document

Version: v1
Status: Stub
Created: 2026-05-14
---

## 1. Background

task-027 established the packet-audit pipeline (analyzer, `tools/packet-audit/` CLI, `EncodeForeign` registry, cycle guard, `candidatesFromFName` mappings) and shipped wire-correct login packets. task-028 applied the same pipeline to the character domain (48 packets), delivering per-packet audit reports under `docs/packets/audits/gms_v95/`, IDA-export JSON for four cross-version targets, and 4-variant `pt.Variants` round-trip tests. Both tasks have shipped to main and the pipeline is mature.

The world domain — field, portal, npc — is the structural backbone of every map transition and NPC interaction in the game. Wire bugs here are immediately user-visible: a corrupted `WarpToMap` packet drops the client to a black screen; a malformed `SetField` sends the character to the wrong map; NPC conversation packets with wrong text-type bytes render as garbled dialog. The field domain also carries the highest rate of sub-op type diversity — clock types, weather effect types, kite types, transport route IDs — which mirrors the sub-op risk pattern seen in the character effect family.

GMS v83, v87, v95, and JMS v185 IDA exports are available for cross-version gate validation. No prior tasks have targeted the world domain specifically. The `portal/` domain is tiny (0 clientbound, 2 serverbound — a portal-script trigger and test) and should be a quick pass. The `field/` and `npc/` domains have more depth.

## 2. Scope

### Packet inventory

| Domain | Clientbound | Serverbound | Total |
|---|---|---|---|
| field | 21 | 2 | 23 |
| portal | 0 | 2 | 2 |
| npc | 14 | 18 | 32 |
| **Total** | **35** | **22** | **57** |

Note: `portal/clientbound/` does not exist — the portal domain only has serverbound (client→server) packets. `field/clientbound/` has 21 files covering warp, clock, effects, weather, kites, and the transport/set-field pair.

### Out of scope

- Business logic in `services/atlas-channel/` beyond the minimum needed to wire a wire-fix through.
- Re-auditing login (task-027) or character (task-028) domain packets.
- Sub-op enum expansion for field effect type-discriminators where the analyzer cannot statically resolve the discriminator byte — defer to `_pending.md` per design §9.
- Atlas-side bare handlers that have no `libs/atlas-packet` decoder — document in `_pending.md`, mirror task-028 treatment.
- NPC scripting engine (atlas-npcs service) business logic — only atlas-packet wire shapes are in scope.

## 3. Goals

- Produce per-packet audit reports under `docs/packets/audits/gms_v95/` for every packet in the field, portal, and npc domains.
- Land real wire-bug fixes with 4-variant `pt.Variants` round-trip tests (v28/v83/v95 + JMS v185).
- Append world-domain IDA-export entries to `docs/packets/ida-exports/gms_v95.json` and the matching `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` files.
- Verify template opcodes against IDA dispatcher case-statements for every new template entry touched.
- Defer field effect sub-op enum drift to `_pending.md` per design §9 (same pattern as character `effect_*` family from task-028).
- Defer bare handlers to `_pending.md` per task-028 design §1.

## 4. Functional Requirements

### 4.1 Coverage matrix

Produce a verdict row in `docs/packets/audits/gms_v95/SUMMARY.md` for each of the 57 packets:

**field/clientbound (21):** `affected_area_created`, `affected_area_removed`, `clock`, `effect`, `effect_weather`, `kite_destroy`, `kite_error`, `kite_spawn`, `set_field`, `transport`, `warp_to_map`  
**field/serverbound (2):** (enumerate from `libs/atlas-packet/field/serverbound/`)  
**portal/serverbound (2):** `script` (portal trigger), plus test companion  
**npc/clientbound (14):** `action`, `conversation`, `guide_talk`, `shop_list`, `shop_operation`, `shop_operation_body`, `spawn`, `spawn_request_controller`  
**npc/serverbound (18):** (enumerate from `libs/atlas-packet/npc/serverbound/`)

Each row: FName, IDA function address, verdict (✅ / ⚠️ / ❌), notes with file:line citation or `_pending.md` reference.

### 4.2 IDA exports

Populate world-domain FName entries in `docs/packets/ida-exports/gms_v95.json` for all 57 packets. During the cross-version pass, add matching entries to `gms_v83.json`, `gms_v87.json`, and `gms_jms_185.json`. Use the existing entry schema.

### 4.3 Wire bug fixes

Every ❌ verdict gets a fix in `libs/atlas-packet/<domain>/{clientbound,serverbound}/*.go`, gated on affected versions only. Each fix lands with:

- A 4-variant test sweep using `pt.Variants` (GMS v28/v83/v95 + JMS v185).
- A 1-3 sentence comment citing the IDA function name and finding.
- A row in `post-phase-b.md`'s "Real wire bugs fixed" table.

### 4.4 Template fixes

Template drift gets fixed in `services/atlas-configurations/seed-data/templates/template_gms_*.json` and `template_jms_185_1.json`. Each fix lands with a row in `post-phase-b.md`'s "Template opcode/enum fixes" table and a commit message citing the IDA case-statement value.

### 4.5 TypeRegistry extensions

When the analyzer surfaces an unregistered sub-struct type, extend `tools/packet-audit/internal/atlaspacket/registry.go`. Likely new types for this domain:

- `SetField` map-header sub-struct (contains map ID, portal ID, spawn point).
- `WarpToMap` destination coordinate block.
- NPC shop item entry sub-struct in `shop_list`.
- NPC conversation text-type encoding in `conversation`.
- Clock time-of-day sub-struct.

Each registry addition commits with a tagged test in `registry_test.go`.

### 4.6 Cross-version re-verification

After the v95 pass, user loads v83 IDA and verifies each fixed file's `Region/MajorVersion` gate. Repeat for v87, then JMS v185. Record per-version notes in `post-phase-b.md`.

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
| `libs/atlas-packet/field/` | Wire-bug fixes in clientbound/serverbound encoders. Tests per variant. |
| `libs/atlas-packet/portal/` | Wire-bug fixes (small domain — 2 serverbound packets). Tests per variant. |
| `libs/atlas-packet/npc/` | Wire-bug fixes. Tests per variant. Particularly shop-list and conversation variants. |
| `tools/packet-audit/` | TypeRegistry additions for world sub-structs. |
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
- `field/clientbound/effect.go` and `field/clientbound/effect_weather.go` follow the sub-op enum drift pattern seen in `character/clientbound/effect*.go` from task-028 — expect deferral to `_pending.md` for the leading-byte effect-type discriminator.
- `field/clientbound/set_field.go` is the most structurally complex packet in this domain — it encodes the full map initialization header and is highly version-sensitive. Plan for extra audit depth here.
- `npc/clientbound/conversation.go` carries a text-type field that routes to different dialog box types (say/askText/askYesNo/etc.) — this is a candidate for sub-op deferral or per-type individual audit, depending on whether atlas-packet models each type as a separate file.
- `portal/serverbound/` only has one real packet (portal-script trigger) plus a test file — portal is a quick pass.
- Instance-based transports (the `instance-based-transports` branch) added `libs/atlas-packet/field/clientbound/transport.go`. Confirm that branch has merged to main before auditing `transport.go`, or coordinate.

## 10. Key Risks / Patterns to Watch For (from task-027/028 lessons)

- **Field effect sub-op enum drift**: `effect.go` and `effect_weather.go` likely use a leading-byte type discriminator. Same pattern as `character/clientbound/effect.go` from task-028 — defer per design §9 if the discriminator is not encoded per-file.
- **`SetField` complexity**: This packet is a map-initialization header with many version-sensitive fields (channel number format, field limits, mobility data). High risk of multiple nested version guards — stop at 2 deep, remainder to `_pending.md`.
- **Dispatcher-layer offset**: NPC action/conversation packets routed through CUserPool may have `characterId` prepended before the NPC-specific payload — atlas wire includes it at offset 0.
- **`EncodeMask` / sub-struct method calls**: NPC shop items appear as sub-struct calls — ack as tool-limitation.
- **Loop linearization**: NPC shop list item arrays flatten incorrectly — ack and verify bounds against IDA.
- **Dispatcher case-statement validation**: every new template opcode MUST be confirmed against IDA dispatcher decompile before commit.
- **Cross-version gate boundaries**: `SetField` and `WarpToMap` are well-known to differ between v83 and v95 (field-limits encoding, spawn-point format). Don't assume gate boundaries until IDA confirms.
- **Instance-transport coordination**: `field/clientbound/transport.go` may be in-flight on the `instance-based-transports` branch. Confirm merge status before touching this file.
- **Hidden constructor-signature ripples**: field/npc encoder struct changes ripple to `atlas-channel` warp/spawn handlers — verify build clean.
- **Audit-report ack footers**: add ack footer AFTER the final audit run; if re-running, `git checkout HEAD -- <report.md>` to revert.

## 11. Acceptance Criteria

- [ ] All 57 listed packet files have audit reports under `docs/packets/audits/gms_v95/`.
- [ ] Every ❌ has either a fix commit OR a `_pending.md` row.
- [ ] All 4 verification commands pass cleanly: `go build ./...`, `go vet ./libs/atlas-packet/...`, `go test -race ./libs/atlas-packet/...`, `go test -race ./tools/packet-audit/...`.
- [ ] gitleaks scrub clean.
- [ ] `post-phase-b.md` ledger written.
- [ ] plan-adherence-reviewer + backend-guidelines-reviewer dispatched and findings addressed before PR.

## 12. Dependencies

- task-027 (login audit, analyzer baseline) — merged.
- task-028 (character audit, EncodeForeign registry, cycle guard, ack pattern) — merged.
- `instance-based-transports` branch (may affect `field/clientbound/transport.go`) — confirm merge status before pickup.

## 13. Open Questions

- Has `instance-based-transports` merged to main? If not, should task-068 branch off it or wait?
- `npc/clientbound/conversation.go`: does atlas-packet already split dialog types into separate files, or is there one monolithic conversation encoder with a leading type-discriminator? If the latter, expect deferral of the type variants to `_pending.md`.
- `field/clientbound/set_field.go`: what is the current state of version guards here? It may already have `> 83` branches from prior work — audit will reveal if those gates are correctly bounded.
- Are there JMS v185 field formats that diverge structurally (different spawn-point encoding, different field-limit block) from GMS, or does the existing `Region() == "JMS"` gating cover the differences adequately?
