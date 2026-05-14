# task-067: Commerce-Domain Packet Audit — Product Requirements Document

Version: v1
Status: Stub
Created: 2026-05-14
---

## 1. Background

task-027 established the packet-audit pipeline (analyzer, `tools/packet-audit/` CLI, `EncodeForeign` registry, cycle guard, `candidatesFromFName` mappings) and shipped wire-correct login packets. task-028 scaled the same pipeline to the character domain (48 packets), delivering IDA-export JSON, 4-variant `pt.Variants` round-trip tests, template opcode drift fixes, and an analyzer early-return fix. Both tasks have shipped to main.

The commerce domain — cash, interaction, inventory, storage — is the largest single audit grouping in the remaining `libs/atlas-packet/` domains, at ~153 packets. Commerce packets are high-stakes: wire bugs in cash-shop buy/gift/equip-slot flows or inventory move operations result in visible item loss or duplication at the client; storage desyncs cause durable confusion across sessions. The `interaction` domain alone is 63 packets (3 clientbound + 60 serverbound), representing the trade, hire-merchant, personal-store, and mini-game interaction flows — each sub-interaction type routed by a leading-byte discriminator. This domain is expected to produce the heaviest `_pending.md` deferral load of any audit task.

GMS v83, v87, v95, and JMS v185 IDA exports are available for cross-version gate validation. Existing legacy-merchant-audit-remediation work has touched hire-merchant flows; cross-reference that branch before modifying any `interaction/` handler to avoid conflicts.

## 2. Scope

### Packet inventory

| Domain | Clientbound | Serverbound | Total |
|---|---|---|---|
| cash | 11 | 48 | 59 |
| interaction | 3 | 60 | 63 |
| inventory | 8 | 10 | 18 |
| storage | 5 | 8 | 13 |
| **Total** | **27** | **126** | **153** |

Note: `cash/serverbound/` and `interaction/serverbound/` are the two largest serverbound envelopes in the library. Many interaction files are individually named sub-op variants (e.g., `operation_trade_put_item.go`, `operation_merchant_buy.go`) — these are individually addressable by the analyzer even when they share a common leading discriminator.

### Out of scope

- Business logic in `services/atlas-cashshop/`, `services/atlas-inventory/`, `services/atlas-storage/`.
- Re-auditing login (task-027) or character (task-028) domain packets.
- Sub-op enum expansion for interaction type-discriminators that the individual packet files do not already make explicit — defer to `_pending.md` per design §9.
- Atlas-side bare handlers that have no `libs/atlas-packet` decoder — document in `_pending.md`, mirror task-028 treatment.
- atlas-cashshop CREATED/DELETED event handling — that service only processes Kafka events, not map/field packets; N/A.

## 3. Goals

- Produce per-packet audit reports under `docs/packets/audits/gms_v95/` for every packet in the cash, interaction, inventory, and storage domains.
- Land real wire-bug fixes with 4-variant `pt.Variants` round-trip tests (v28/v83/v95 + JMS v185).
- Append commerce-domain IDA-export entries to `docs/packets/ida-exports/gms_v95.json` and the matching `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` files.
- Verify template opcodes against IDA dispatcher case-statements for every new template entry touched.
- Defer interaction sub-op enum drift to `_pending.md` per design §9 where the discriminator is not already encoded per-file.
- Defer bare handlers to `_pending.md` per task-028 design §1.

## 4. Functional Requirements

### 4.1 Coverage matrix

Produce a verdict row in `docs/packets/audits/gms_v95/SUMMARY.md` for each of the 153 packets:

**cash/clientbound (11):** (enumerate from `libs/atlas-packet/cash/clientbound/`)  
**cash/serverbound (48):** Includes `check_wallet`, `item_use`, `item_use_chalkboard`, `item_use_field_effect`, `item_use_pet_consumable`, `shop_entry`, `shop_operation`, `shop_operation_buy_*`, `shop_operation_gift`, `shop_operation_increase_*`, `shop_operation_move_*`, `shop_operation_rebate_locker_item`, `shop_operation_set_wishlist`. Each operation variant is individually addressable.  
**interaction/clientbound (3):** (enumerate from `libs/atlas-packet/interaction/clientbound/`)  
**interaction/serverbound (60):** Includes trade, hire-merchant, personal-store, memory-game, cash-trade, and visit operations. All individually named.  
**inventory/clientbound (8):** (enumerate from `libs/atlas-packet/inventory/clientbound/`)  
**inventory/serverbound (10):** (enumerate from `libs/atlas-packet/inventory/serverbound/`)  
**storage/clientbound (5):** (enumerate from `libs/atlas-packet/storage/clientbound/`)  
**storage/serverbound (8):** (enumerate from `libs/atlas-packet/storage/serverbound/`)

Each row: FName, IDA function address, verdict (✅ / ⚠️ / ❌), notes with file:line citation or `_pending.md` reference.

### 4.2 IDA exports

Populate commerce-domain FName entries in `docs/packets/ida-exports/gms_v95.json` for all 153 packets. During the cross-version pass, add matching entries to `gms_v83.json`, `gms_v87.json`, and `gms_jms_185.json`. Use the existing entry schema.

### 4.3 Wire bug fixes

Every ❌ verdict gets a fix in `libs/atlas-packet/<domain>/{clientbound,serverbound}/*.go`, gated on affected versions only. Each fix lands with:

- A 4-variant test sweep using `pt.Variants` (GMS v28/v83/v95 + JMS v185).
- A 1-3 sentence comment citing the IDA function name and finding.
- A row in `post-phase-b.md`'s "Real wire bugs fixed" table.

### 4.4 Template fixes

Template drift gets fixed in `services/atlas-configurations/seed-data/templates/template_gms_*.json` and `template_jms_185_1.json`. Each fix lands with a row in `post-phase-b.md`'s "Template opcode/enum fixes" table and a commit message citing the IDA case-statement value.

### 4.5 TypeRegistry extensions

When the analyzer surfaces an unregistered sub-struct type, extend `tools/packet-audit/internal/atlaspacket/registry.go`. Likely new types for this domain:

- Cash-shop item sub-structs in `shop_entry` and `shop_operation_buy_*` variants.
- Inventory change record sub-struct (shared across many inventory clientbound packets).
- Storage item slot sub-struct.
- Interaction trade item slot sub-struct.

Each registry addition commits with a tagged test in `registry_test.go`.

### 4.6 Cross-version re-verification

After the v95 pass, user loads v83 IDA and verifies each fixed file's `Region/MajorVersion` gate. Repeat for v87, then JMS v185. Record per-version notes in `post-phase-b.md`.

### 4.7 Legacy-merchant coordination

Before modifying any `interaction/` packet, check the legacy-merchant-audit-remediation branch for conflicting changes. If that branch is in flight, either rebase on top of it or coordinate merging order with the team.

## 5. API Surface

No new HTTP/REST endpoints. The audit produces:

- Updated `tools/packet-audit/` TypeRegistry entries.
- New per-packet audit reports under `docs/packets/audits/gms_v95/cash/`, `interaction/`, `inventory/`, `storage/`.
- Updated `docs/packets/ida-exports/gms_{v83,v87,v95}.json` and `gms_jms_185.json`.
- Updated `template_*.json` opcode/enum values for any drift found.

## 6. Data Model

No persistent-data changes. `services/atlas-configurations/seed-data/templates/` JSON files receive opcode corrections only.

## 7. Service Impact

| Service / Lib | Impact |
|---|---|
| `libs/atlas-packet/cash/` | Wire-bug fixes in clientbound/serverbound encoders. Tests per variant. |
| `libs/atlas-packet/interaction/` | Wire-bug fixes. Tests per variant. Coordinate with legacy-merchant-audit-remediation. |
| `libs/atlas-packet/inventory/` | Wire-bug fixes. Tests per variant. |
| `libs/atlas-packet/storage/` | Wire-bug fixes. Tests per variant. |
| `tools/packet-audit/` | TypeRegistry additions for commerce sub-structs. |
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
- `interaction` is the largest single domain (63 packets) and will produce the heaviest `_pending.md` deferral load. Plan for 20-30% deferral rate on interaction sub-op packets.
- `cash/serverbound/` 48-file count is driven by individual named shop-operation variants — each is individually analyzer-addressable (they write different field sequences, not just different discriminator bytes).
- Inventory change encoding follows a version-sensitive pattern (v83 vs v95 item slot widths) — expect gate-boundary findings similar to `GW_CharacterStat` HP/MP widening found in task-028.
- atlas-cashshop only handles CREATED/DELETED Kafka events; map/field messages are N/A for that service. Do not route changes through atlas-cashshop for this audit.

## 10. Key Risks / Patterns to Watch For (from task-027/028 lessons)

- **`interaction` sub-op enum drift (primary risk)**: The interaction domain's leading-byte sub-op dispatch routes to different handler files, but `operation.go` is the shared entry. Confirm whether the discriminator byte appears literally in each sub-file or only in the shared router — if the latter, many sub-files will be `_pending.md` deferrals.
- **Dispatcher-layer offset**: CUserPool dispatchers prepend `characterId` before routing — atlas wire includes it at offset 0. Trade and personal-store packets encoding buyer/seller IDs are susceptible.
- **`EncodeMask` / sub-struct method calls**: appear as one analyzer call but emit multiple bytes — ack as tool-limitation.
- **Loop linearization**: Cash-shop item lists and inventory batch-change records involve fixed-count loops — the analyzer flattens them; ack as tool-limitation and verify loop bounds against IDA.
- **Dispatcher case-statement validation**: every new template opcode MUST be confirmed against IDA dispatcher decompile before commit. The task-028 lesson about wrong opcodes (0xE7 vs 0xB4) applies here.
- **Cross-version gate boundaries**: Cash-shop wire shapes differ between v83 (no NX credit categories) and v95 (expanded payment categories). Don't assume gate boundaries until cross-version IDA confirms.
- **Item slot width changes**: v95 likely widens item slot types — similar to character stat HP/MP widening found in task-028. Flag every `int16`/`int32` item-slot field as a cross-version gate candidate.
- **Hidden constructor-signature ripples**: adding fields to encoder structs ripples to `atlas-channel`, `atlas-cashshop`, `atlas-inventory` handlers — verify build clean across services.
- **Legacy-merchant-audit-remediation conflict**: check that branch status before touching `interaction/` files.
- **Audit-report ack footers**: add ack footer AFTER the final audit run; if re-running, `git checkout HEAD -- <report.md>` to revert.

## 11. Acceptance Criteria

- [ ] All 153 listed packet files have audit reports under `docs/packets/audits/gms_v95/`.
- [ ] Every ❌ has either a fix commit OR a `_pending.md` row.
- [ ] All 4 verification commands pass cleanly: `go build ./...`, `go vet ./libs/atlas-packet/...`, `go test -race ./libs/atlas-packet/...`, `go test -race ./tools/packet-audit/...`.
- [ ] gitleaks scrub clean.
- [ ] `post-phase-b.md` ledger written.
- [ ] plan-adherence-reviewer + backend-guidelines-reviewer dispatched and findings addressed before PR.

## 12. Dependencies

- task-027 (login audit, analyzer baseline) — merged.
- task-028 (character audit, EncodeForeign registry, cycle guard, ack pattern) — merged.
- legacy-merchant-audit-remediation (in-flight) — coordinate before touching `interaction/`.

## 13. Open Questions

- What is the status of legacy-merchant-audit-remediation? If it's still in flight, should this task rebase on it, or defer interaction/ changes until it merges?
- `interaction/serverbound/operation.go` (the shared router): does it contain the discriminator byte write, or do individual sub-files write their own discriminator? This determines how many sub-files are `_pending.md` deferrals vs. individually auditable.
- Are there v95-only cash-shop features (e.g., bonus items, gift certificates, Android cosmetics) that add new fields not present in v83? Cross-version pass will reveal.
- JMS v185 cash-shop: JMS had a significantly different NX point system in v185. Does atlas-packet already have `Region() == "JMS"` gates in cash/, or is this entirely uncharted?
