# Commerce-Domain Packet Audit — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-15
---

## 1. Overview

task-027 established the packet-audit pipeline (analyzer, `tools/packet-audit/` CLI, `EncodeForeign` registry, cycle guard, `candidatesFromFName` mappings) and shipped wire-correct login packets. task-028 scaled the same pipeline to the character domain (48 packets), delivering IDA-export JSON, 4-variant `pt.Variants` round-trip tests, template opcode drift fixes, and an analyzer early-return fix. task-065 (combat) and task-066 (social) are the in-flight predecessors covering combat and social envelopes. Both task-027 and task-028 have shipped to main.

The commerce domain — cash, interaction, inventory, storage — is the next audit grouping. Commerce packets are high-stakes: wire bugs in cash-shop buy/gift/equip-slot flows or inventory move operations result in visible item loss or duplication at the client; storage desyncs cause durable confusion across sessions. The `interaction` domain (32 packets) covers trade, hire-merchant, personal-store, and mini-game flows — each sub-interaction is dispatched by a leading-byte discriminator written by the shared `Operation` struct in `libs/atlas-packet/interaction/serverbound/operation.go`, while each sub-file encodes only its payload after the discriminator. This separation means each sub-file is independently analyzer-addressable.

GMS v83, v87, v95, and JMS v185 IDA exports are available for cross-version gate validation.

## 2. Goals

Primary goals:

- Produce per-packet audit reports under `docs/packets/audits/gms_v95/{cash,interaction,inventory,storage}/` for every commerce-domain packet.
- Land real wire-bug fixes with 4-variant `pt.Variants` round-trip tests (GMS v28/v83/v95 + JMS v185).
- Append commerce-domain IDA-export entries to `docs/packets/ida-exports/gms_{v83,v87,v95}.json` and `gms_jms_185.json`.
- Verify template opcodes against IDA dispatcher case-statements for every new template entry touched.
- Defer interaction sub-op enum drift to `_pending.md` per task-028 design §9 where the discriminator is not encoded in the sub-file itself.
- Defer bare handlers (no `libs/atlas-packet` decoder) to `_pending.md` per task-028 design §1.

Non-goals:

- Business logic in `services/atlas-cashshop/`, `services/atlas-inventory/`, `services/atlas-storage/`.
- Re-auditing login (task-027), character (task-028), combat (task-065), or social (task-066) domain packets.
- Sub-op enum expansion for interaction type-discriminators that the individual packet files do not encode themselves — these defer to `_pending.md`.
- Atlas-side bare handlers (no `libs/atlas-packet` decoder) — document in `_pending.md`.
- atlas-cashshop Kafka CREATED/DELETED event handling — that service does not process map/field packets.

## 3. User Stories

- As an atlas-channel maintainer, I want every commerce-domain encoder/decoder verified against IDA v95 so client-visible bugs (item loss, NX duplication, storage desync) are caught before they ship.
- As a server operator, I want commerce wire correctness gated per region/major version so v83 clients are not regressed by v95-targeted fixes.
- As a future audit-pipeline contributor, I want `tools/packet-audit/` to recognize commerce sub-struct types (cash-shop item, inventory change record, storage slot, trade slot) so subsequent re-audits do not require manual TypeRegistry edits.
- As a release engineer, I want template opcode/enum corrections committed alongside the IDA case-statement they were verified against so future opcode drift is auditable.

## 4. Functional Requirements

### 4.1 Coverage matrix

Produce a verdict row in `docs/packets/audits/gms_v95/SUMMARY.md` for each of the 78 source-file packets (counts exclude `*_test.go`):

| Domain | Clientbound | Serverbound | Total |
|---|---|---|---|
| cash | 6 | 24 | 30 |
| interaction | 2 | 30 | 32 |
| inventory | 4 | 5 | 9 |
| storage | 3 | 4 | 7 |
| **Total** | **15** | **63** | **78** |

**cash/clientbound (6):** `query_result.go`, `shop_inventory.go`, `shop_item_moved.go`, `shop_open.go`, `shop_operation_body.go`, `shop_operation_result.go`.

**cash/serverbound (24):** Enumerate from `libs/atlas-packet/cash/serverbound/`. Includes `check_wallet`, `item_use`, `item_use_chalkboard`, `item_use_field_effect`, `item_use_pet_consumable`, `shop_entry`, `shop_operation`, `shop_operation_buy_*`, `shop_operation_gift`, `shop_operation_increase_*`, `shop_operation_move_*`, `shop_operation_rebate_locker_item`, `shop_operation_set_wishlist`. Each operation variant is individually addressable.

**interaction/clientbound (2):** `interaction.go`, `interaction_body.go`.

**interaction/serverbound (30):** `operation.go` (shared router writing the `mode` discriminator) plus 29 named sub-op files covering trade, hire-merchant, personal-store, memory-game, cash-trade, and visit operations. Sub-files encode payload only; the discriminator lives in `operation.go`.

**inventory/clientbound (4):** Enumerate from `libs/atlas-packet/inventory/clientbound/`.

**inventory/serverbound (5):** Enumerate from `libs/atlas-packet/inventory/serverbound/`.

**storage/clientbound (3):** `error.go`, `show.go`, `update_assets.go`.

**storage/serverbound (4):** `operation.go`, `operation_meso.go`, `operation_retrieve_asset.go`, `operation_store_asset.go`.

Each row: FName, IDA function address, verdict (✅ / ⚠️ / ❌), notes with file:line citation or `_pending.md` reference.

### 4.2 IDA exports

Populate commerce-domain FName entries in `docs/packets/ida-exports/gms_v95.json` for all 78 packets. During the cross-version pass, add matching entries to `gms_v83.json`, `gms_v87.json`, and `gms_jms_185.json`. Use the existing entry schema established by task-027/028.

### 4.3 Wire bug fixes

Every ❌ verdict gets a fix in `libs/atlas-packet/<domain>/{clientbound,serverbound}/*.go`, gated on affected versions only. Each fix lands with:

- A 4-variant test sweep using `pt.Variants` (GMS v28/v83/v95 + JMS v185).
- A 1-3 sentence comment citing the IDA function name and finding.
- A row in `post-phase-b.md`'s "Real wire bugs fixed" table.

### 4.4 Template fixes

Template drift gets fixed in `services/atlas-configurations/seed-data/templates/template_gms_*.json` and `template_jms_185_1.json`. Each fix lands with a row in `post-phase-b.md`'s "Template opcode/enum fixes" table and a commit message citing the IDA case-statement value (the task-028 lesson about 0xE7 vs 0xB4 applies — never commit a template opcode that has not been confirmed against an IDA dispatcher decompile).

### 4.5 TypeRegistry extensions

When the analyzer surfaces an unregistered sub-struct type, extend `tools/packet-audit/internal/atlaspacket/registry.go`. Likely new types for this domain:

- Cash-shop item sub-structs (used in `shop_entry`, `shop_inventory`, `shop_operation_buy_*` variants).
- Inventory change record sub-struct (shared across many inventory clientbound packets).
- Storage item slot sub-struct.
- Interaction trade item slot sub-struct.

Each registry addition commits with a tagged test in `tools/packet-audit/internal/atlaspacket/registry_test.go`.

### 4.6 Cross-version re-verification

After the v95 pass, the user loads v83 IDA and verifies each fixed file's `Region/MajorVersion` gate. Repeat for v87, then JMS v185. Record per-version notes in `post-phase-b.md`.

### 4.7 Deferral handling

Two categories defer to `_pending.md` rather than blocking the audit:

- **Sub-op enum drift in interaction:** Per design observation, the `mode` discriminator is written by the shared `Operation` struct in `interaction/serverbound/operation.go`, not by the individual sub-files. Enum drift (mode-byte → handler-file mapping) lives in atlas-channel routing, not in `libs/atlas-packet`. Sub-files are auditable for payload correctness in isolation; mode-byte mapping is recorded in `_pending.md` for downstream routing-layer audit.
- **Bare handlers:** Atlas-side handlers that have no `libs/atlas-packet` decoder counterpart get a `_pending.md` row referencing the handler file:line. Mirror task-028 §1 treatment.

## 5. API Surface

No new HTTP/REST endpoints. The audit produces:

- Updated `tools/packet-audit/internal/atlaspacket/registry.go` TypeRegistry entries.
- New per-packet audit reports under `docs/packets/audits/gms_v95/{cash,interaction,inventory,storage}/`.
- Updated `docs/packets/ida-exports/gms_{v83,v87,v95}.json` and `gms_jms_185.json`.
- Updated `template_*.json` opcode/enum values for any drift found.
- `post-phase-b.md` ledger in the task folder.

## 6. Data Model

No persistent-data changes. `services/atlas-configurations/seed-data/templates/` JSON files receive opcode corrections only.

## 7. Service Impact

| Service / Lib | Impact |
|---|---|
| `libs/atlas-packet/cash/` | Wire-bug fixes in clientbound/serverbound encoders. 4-variant tests per fixed file. |
| `libs/atlas-packet/interaction/` | Wire-bug fixes. 4-variant tests per fixed sub-file. `operation.go` (the shared discriminator writer) audited as a single packet. |
| `libs/atlas-packet/inventory/` | Wire-bug fixes. 4-variant tests per fixed file. |
| `libs/atlas-packet/storage/` | Wire-bug fixes. 4-variant tests per fixed file. |
| `tools/packet-audit/` | TypeRegistry additions for commerce sub-structs. Each addition tested. |
| `services/atlas-configurations/seed-data/templates/` | Opcode/enum corrections gated to the templates whose dispatcher case-statements were inspected. |
| `docs/packets/ida-exports/` | Refreshed export JSON per version (v83, v87, v95, JMS 185). |
| `services/atlas-cashshop/`, `services/atlas-inventory/`, `services/atlas-storage/` | Constructor-signature ripples only if encoder structs gain fields. Build-clean verification required across all three. No business-logic changes. |

## 8. Non-Functional Requirements

- All fixes ship with 4-variant test sweeps using `pt.Variants` (GMS v28/v83/v95 + JMS v185).
- Hard cap: no encoder/decoder grows beyond 2 nested region/version guards (per task-028 design §7). 3+ nested → STOP, log to `_pending.md`.
- gitleaks scrub clean before PR (no `/home/<user>/` absolute paths in audit reports — strip via `sed` before commit).
- `go test -race ./libs/atlas-packet/... ./tools/packet-audit/...` clean.
- `go build ./...` clean across all services (catch constructor-ripple breakage).
- `go vet ./libs/atlas-packet/...` clean.
- Login (task-027), character (task-028), combat (task-065), and social (task-066) verdicts unchanged.
- Per CLAUDE.md "Build & Verification": if any service `go.mod` or `Dockerfile` is touched, run `docker build -f services/<svc>/Dockerfile .` from the worktree root for that service.

### Working assumptions

- IDA v95 is the primary target; v83/v87/JMS v185 IDA available for cross-version pass when user loads them.
- Templates `template_gms_{12,28,83,87,92,95}_1.json` and `template_jms_185_1.json` exist; new opcodes added per audit.
- `interaction` will produce the heaviest `_pending.md` deferral load. Plan for ≥20% deferral rate on interaction sub-op packets given the routing-layer enum mapping lives outside `libs/atlas-packet`.
- `cash/serverbound/` count of 24 is driven by individually named shop-operation variants — each is individually analyzer-addressable (they write different field sequences, not just different discriminator bytes).
- Inventory change encoding follows a version-sensitive pattern (v83 vs v95 item slot widths) — expect gate-boundary findings similar to the `GW_CharacterStat` HP/MP widening found in task-028.
- atlas-cashshop only handles CREATED/DELETED Kafka events; map/field messages are N/A for that service.

### Key risks / patterns to watch for (from task-027/028 lessons)

- **Dispatcher-layer offset:** CUserPool dispatchers prepend `characterId` before routing — atlas wire includes it at offset 0. Trade and personal-store packets encoding buyer/seller IDs are susceptible.
- **`EncodeMask` / sub-struct method calls:** appear as one analyzer call but emit multiple bytes — ack as tool-limitation.
- **Loop linearization:** Cash-shop item lists and inventory batch-change records involve fixed-count loops — the analyzer flattens them; ack as tool-limitation and verify loop bounds against IDA.
- **Dispatcher case-statement validation:** every new template opcode MUST be confirmed against IDA dispatcher decompile before commit (task-028 0xE7 vs 0xB4 lesson).
- **Cross-version gate boundaries:** Cash-shop wire shapes likely differ between v83 (no NX credit categories) and v95 (expanded payment categories). Do not assume gate boundaries until cross-version IDA confirms.
- **Item slot width changes:** v95 likely widens item slot types (similar to character stat HP/MP widening in task-028). Flag every `int16`/`int32` item-slot field as a cross-version gate candidate.
- **Hidden constructor-signature ripples:** adding fields to encoder structs ripples to `atlas-channel`, `atlas-cashshop`, `atlas-inventory` handlers — verify build clean across services.
- **General git discipline:** before modifying any `interaction/` packet, run `git log --since="14 days" -- libs/atlas-packet/interaction/` and check for concurrent in-flight branches. The `legacy-merchant-audit-remediation` branch targets atlas-merchant service architecture (ARCH/STRUCT/KAFKA checks), not `libs/atlas-packet/`, but the same discipline applies to any future commerce-related branch.
- **Audit-report ack footers:** add ack footer AFTER the final audit run; if re-running, `git checkout HEAD -- <report.md>` to revert.

## 9. Open Questions

- **Resolved:** `interaction/serverbound/operation.go` writes the `mode` discriminator byte. Individual sub-files (e.g., `operation_merchant_buy.go`) write payload only. Each sub-file is individually auditable; the mode-byte → handler-file mapping is recorded in `_pending.md` as a routing-layer concern outside `libs/atlas-packet`.
- Are there v95-only cash-shop features (e.g., bonus items, gift certificates, Android cosmetics) that add new fields not present in v83? Cross-version IDA pass will reveal.
- JMS v185 cash-shop: JMS had a significantly different NX point system in v185. Does `atlas-packet/cash/` already have `Region() == "JMS"` gates, or is this entirely uncharted? Design phase to inspect.
- Are there v95 vs v83 differences in inventory item-slot width (parallel to task-028's `GW_CharacterStat` finding)? Cross-version IDA pass will reveal.

## 10. Acceptance Criteria

- [ ] All 78 listed packet source files have audit reports under `docs/packets/audits/gms_v95/{cash,interaction,inventory,storage}/`.
- [ ] Every ❌ has either a fix commit (with 4-variant test) OR a `_pending.md` row.
- [ ] `docs/packets/audits/gms_v95/SUMMARY.md` row for every packet with verdict ✅/⚠️/❌.
- [ ] `docs/packets/ida-exports/gms_{v83,v87,v95}.json` and `gms_jms_185.json` contain commerce-domain FName entries.
- [ ] All verification commands pass cleanly:
  - `go build ./...`
  - `go vet ./libs/atlas-packet/...`
  - `go test -race ./libs/atlas-packet/...`
  - `go test -race ./tools/packet-audit/...`
- [ ] `docker build` clean for any service whose `go.mod` or `Dockerfile` was touched.
- [ ] gitleaks scrub clean (no absolute paths leaking user home).
- [ ] `post-phase-b.md` ledger written, listing wire bugs fixed, template fixes, deferrals, and per-version cross-verification notes.
- [ ] plan-adherence-reviewer + backend-guidelines-reviewer dispatched and findings addressed before PR.

## 11. Dependencies

- task-027 (login audit, analyzer baseline) — merged.
- task-028 (character audit, EncodeForeign registry, cycle guard, ack pattern) — merged.
- task-065 (combat audit) — in flight; no file overlap with commerce domain expected.
- task-066 (social audit) — in flight; no file overlap with commerce domain expected.
- `legacy-merchant-audit-remediation` — service-architecture remediation for atlas-merchant; no `libs/atlas-packet/` file overlap. Listed for git-discipline awareness only.
