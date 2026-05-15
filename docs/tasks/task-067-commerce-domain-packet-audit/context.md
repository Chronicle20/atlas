# Context — Commerce-Domain Packet Audit (task-067)

Companion to `plan.md`. Captures the key files, decisions, and dependencies the implementer needs without re-reading the full PRD/design.

---

## Source artifacts (read these first)

- `docs/tasks/task-067-commerce-domain-packet-audit/prd.md` — full requirements, coverage matrix (78 src files), deferral rules.
- `docs/tasks/task-067-commerce-domain-packet-audit/design.md` — scope/sequencing constraints (§1), multi-shape file inflation (§3.1), hard parts #1–#6 (§4–§9), phasing (§10), templates (§11), risks (§12), out-of-scope (§13), file enumeration (§14), reference points (§15), what plan-task should produce (§16).
- `docs/tasks/task-027-atlas-packet-v95-audit/` — original audit pipeline, `EncodeForeign` registry precedent.
- `docs/tasks/task-028-character-domain-audit/post-phase-b.md` — closing memo template; 0xE7-vs-0xB4 lesson (template opcode case-statement verification).
- `docs/tasks/task-066-social-domain-packet-audit/plan.md` — direct structural precedent for this plan (registry batch → sub-domain audits → cross-version pass → regression → closeout).

---

## Key existing code references

| Concern | File | Notes |
|---|---|---|
| Registry pass-2 method switch | `tools/packet-audit/internal/atlaspacket/registry.go:101-114` | Phase 0 Task 2 extension site. Currently handles `Encode` (closure return via `findReturnClosure`) + `Write` (flat body). Add `EncodeEntry` (closure) + `EncodeBytes` (flat). |
| Registry receiver-type discovery | `tools/packet-audit/internal/atlaspacket/registry.go:81-100` | Pass-1 walks every Go file under `libs/atlas-packet/` and records every struct decl. No change needed; receiver-type discovery is method-name-agnostic. |
| `findReturnClosure` helper | `tools/packet-audit/internal/atlaspacket/registry.go` (search for symbol) | Used by `Encode` case to descend into the closure body of `Encode(l, ctx) func(opts) []byte`. Reuse for `EncodeEntry`. |
| `collectCallsWithCtx` | `tools/packet-audit/internal/atlaspacket/registry.go` | The analyzer's call-collection routine. Method-name-agnostic — pass it the right body. |
| `CashInventoryItem` definition | `libs/atlas-packet/cash/clientbound/shop_inventory.go:14-38` | The sub-struct whose `EncodeBytes` returns flat `[]byte` (49 bytes). Used by `CashShopInventory.Encode`, `CashShopPurchaseSuccess.Encode`, `CashItemMovedToCashInventory.Encode`. |
| `inventory.ChangeEntry` family | `libs/atlas-packet/inventory/change_entry.go:31-210` | Four receiver types (`AddEntry`/`QuantityUpdateEntry`/`MoveEntry`/`RemoveEntry`) each with `EncodeEntry` closure. Used by `inventory/clientbound/change_batch.go`. |
| `inventory/clientbound/change.go` four shapes | `libs/atlas-packet/inventory/clientbound/change.go:18-210` | Silent-loss hot path (design §6). `QuantityUpdate`/`ChangeMove`/`Remove`/`Add` each write own mode byte (1/2/3/0). Equip-slot `addMov` byte at `change.go:92-99` and `change.go:146-148`. |
| `interaction/serverbound/operation.go` dispatcher | `libs/atlas-packet/interaction/serverbound/operation.go:29-41` | Writes only the `mode` byte; sub-op files carry payloads. Design §5 best-case scenario. |
| `cash/clientbound/shop_operation_body.go` | `libs/atlas-packet/cash/clientbound/shop_operation_body.go:12-138` | 78-entry constants block + 8 factory functions delegating to per-shape structs (mapped in Task 3 `phase-0-survey.md`). |
| `storage/clientbound/error.go` three shapes | `libs/atlas-packet/storage/clientbound/error.go` | `ErrorSimple` / `UpdateMeso` / `ErrorMessage` — each writes its own mode byte. Audit independently per design §7. |
| `model.Asset.Encode` | `libs/atlas-packet/model/asset.go:164` | Inventory item slot sub-struct; auto-discovered via `Encode`. Used across change.go, shop_item_moved.go, storage panels. |
| `pt.Variants` test harness | `libs/atlas-packet/test/context.go:18-23` | `GMSv28`/`GMSv83`/`GMSv95`/`JMSv185` tenant variants for 4-variant test sweeps. |
| Round-trip harness | `libs/atlas-packet/test/roundtrip.go:12-24` | `reader.Available() == 0` after decode — the "wire complete" oracle. |
| Existing 4-variant byte-output test | `libs/atlas-packet/login/clientbound/auth_success_test.go:9-37` | Copy this shape for new clientbound encoder tests. |
| `tenant.Model` accessors | `libs/atlas-tenant/tenant.go:10-31` | `Region()` returns `string`, `MajorVersion()` returns `uint16`. **Don't confuse `Region()` with `world.Id`** — bug-pattern from memory. |
| Template entity | `services/atlas-configurations/atlas.com/configurations/templates/entity.go:14-20` | JSON blob in `Data` column — **no DB migration required** when adding fields/opcodes. |
| Template seed files | `services/atlas-configurations/seed-data/templates/template_gms_{12,28,83,87,92,95}_1.json` + `template_jms_185_1.json` | Opcode/enum fixes land here. Each fix lands with a commit message citing the IDA case-statement value. |

---

## Critical decisions locked in design

- **Pipeline is mature; don't redesign it.** Don't touch the analyzer unless a concrete commerce finding forces a fix. Phase 0 registry extension is *method-name recognition only* (additive switch case), not analyzer behavior change.
- **78 src files / ~89 wire shapes.** Five multi-shape files (design §3.1) inflate the audit-row count above the file count: `cash/clientbound/shop_inventory.go` × 4, `cash/clientbound/shop_operation_result.go` × 4, `cash/clientbound/shop_item_moved.go` × 2, `storage/clientbound/error.go` × 3, `inventory/clientbound/change.go` × 4. SUMMARY.md rows use `<file>:<TypeName>` for these.
- **Phase 1 sub-domain ordering:** storage → inventory → interaction → cash. Small/durable first, big/version-sensitive last.
- **`shop_operation_body.go` is a router, not a shape.** One SUMMARY.md row (verdict ⚠️ "router; per-shape rows tracked under target structs"). The 78 named operation/error constants are template-driven; opcode drift is fixed in `template_*.json`, not in atlas-packet.
- **`interaction/serverbound/operation.go` is dispatcher-only** (mode byte only at line 32). 29 named sub-op files are independent wire shapes per design §5. One `_pending.md` row INTERACTION-MODE-MAP for the mode-byte → sub-op-file mapping (routing-layer concern, outside `libs/atlas-packet/`).
- **`storage/serverbound/operation.go` is dispatcher-only** (3 sub-ops). One `_pending.md` row OP-FAMILY-storage.
- **`CashInventoryItem.EncodeBytes` registry extension is *the* enabling Phase 0 work.** Without it the analyzer reports opaque `WriteByteArray` calls and the three cash recursion sites (`CashShopInventory`, `CashShopPurchaseSuccess`, `CashItemMovedToCashInventory`) lose sub-shape verification. Phase 0 Task 2 lands the extension as a separate commit before any cash audit row is written.
- **`inventory.{Add,QuantityUpdate,Move,Remove}Entry.EncodeEntry` is the same gap** for `change_batch.go`. Same Phase 0 Task 2 extension covers it.
- **`model.Asset.Encode`** (inventory item slot) is auto-discovered — no action.
- **Bare-handler exclusion stays.** Atlas-side handlers without a `libs/atlas-packet` decoder go to `_pending.md` — no service-code descent into atlas-cashshop, atlas-inventory, atlas-storage, atlas-channel.
- **Hard cap: 2 nested region/version guards per encoder.** 3+ → STOP, `_pending.md`, sibling task. Awk scan per Phase 2 Step 6.

---

## Decisions deferred to execution time

- **EncodeBytes/EncodeEntry survey result.** Task 1 Step 1 re-runs the survey. If a NEW non-`Encode`/`Write` method name appears beyond the predicted set, the executor decides Option A (extend the switch) vs Option B (per-call ack in `_pending.md`) based on the method's return shape:
  - Flat `[]byte` return → mirror the `EncodeBytes` case.
  - `func(opts) []byte` closure return → mirror the `EncodeEntry` case.
  - Anything else (e.g., requires extra params, returns multiple values) → Option B.
- **`interaction/clientbound/interaction.go` vs `interaction_body.go` shape.** Task 6 Step 4 confirms whether `interaction_body.go` is a router (constructor block parallel to cash) or a standalone shape. Default expectation: two standalone shapes; if it turns out to be a router, the executor mirrors Task 3's pattern and appends a constructor-to-struct map to `phase-0-survey.md`.
- **Cash wire-shape denominator.** ~36 is the design's estimate. Task 3 locks the actual count via constructor-to-struct enumeration. Task 7 re-verifies before running the audit.
- **`shop_operation_body.go` operations/errors enum case-statement values.** Each fix lands with a commit message citing the IDA dispatcher offset. If a single Phase 2 version surfaces >10 stale code values, pause and triage — design §4.3 cap.
- **JMS v185 cash-shop NX system.** PRD §9 q3 flags this as uncharted. Likely outcome per design §9: a 3+ nested gate request triggers `_pending.md` + sibling task. Don't ship a 3-gate encoder under cover.

---

## Workflow notes for the implementer

1. **Verify cwd before every commit** — you're in a worktree at `.worktrees/task-067-commerce-domain-packet-audit/`. The branch is `task-067-commerce-domain-packet-audit`. `git rev-parse --show-toplevel` and `git branch --show-current` must agree.
2. **Run `go test -race ./tools/packet-audit/...` after Phase 0 registry changes.** A registry change that regresses task-027/028/066 fixtures is the worst-case Phase 0 outcome; re-run before continuing.
3. **Run `go test -race ./libs/atlas-packet/...` after every encoder fix.** A v95 fix that regresses v83 round-trip is the second-worst case; the suite is the canary.
4. **Run `go vet ./libs/atlas-packet/...` before each fix-batch bucket commit.**
5. **Service Dockerfile rebuilds — ripple-driven, not pre-emptive.** Constructor-ripple into `atlas-cashshop` / `atlas-inventory` / `atlas-storage` / `atlas-channel` triggers a `docker build` per CLAUDE.md Build & Verification §3. Task 12 Step 3 belt-and-suspenders check covers the case where a `go.mod` or `Dockerfile` changed.
6. **Audit-report ack footers go on AFTER the final audit run.** If you re-run the audit after fixes land, the tool overwrites the report file — re-add the ack footer (or `git checkout HEAD -- <report.md>` to revert if the post-fix verdict no longer needs it).
7. **The audit pipeline produces reports; never auto-mutates `.go` files.** Every encoder fix is a hand edit anchored to a freshly-generated audit report.

---

## External dependencies / open questions

- **IDA-MCP availability:** Phase 1 requires v95 IDA loaded; Phase 2 requires user-driven binary swaps (v83 → v87 → JMS v185). Confirm via `mcp__ida-pro__get_metadata` at the start of each sub-task.
- **Concurrent in-flight branches:** PRD §11 lists `legacy-merchant-audit-remediation` (service architecture, no `libs/atlas-packet/` overlap) and task-065 (combat, no commerce overlap) and task-066 (social, no commerce overlap). Run `git log --since="14 days" -- libs/atlas-packet/{cash,interaction,inventory,storage}/` before starting Phase 1 to catch any new concurrent edits.
- **Cross-version IDA exports:** `gms_v87.json` may not exist when Phase 2 Task 9 begins — task-066 was scheduled to create it. Confirm via `ls docs/packets/ida-exports/` and create from scratch if absent (Task 9 Step 2).
- **gh CLI auth:** required only at Task 12 Step 7 (PR creation). Per project memory, `~/.config/atlas/gh.env` must be sourced (`set -a; . ~/.config/atlas/gh.env; set +a;`) before `gh` since direnv hook does not fire in fresh shells.

---

## Known bug patterns (from prior audits)

- **Dispatcher-layer offset:** CUserPool dispatchers prepend `characterId` before routing — atlas wire includes it at offset 0. Trade and personal-store packets encoding buyer/seller IDs are susceptible (design §10 risks).
- **`EncodeMask` / sub-struct method calls:** appear as one analyzer call but emit multiple bytes — ack as tool-limitation.
- **Loop linearization:** Fixed-count loops are flattened by the analyzer — ack as tool-limitation and cite the IDA loop bound.
- **Dispatcher case-statement validation:** every new template opcode must be confirmed against IDA dispatcher decompile before commit (task-028 0xE7 vs 0xB4 lesson).
- **Cross-version gate boundaries:** Cash-shop wire shapes likely differ between v83 (no NX credit categories) and v95 (expanded payment categories) — don't assume gate boundaries until cross-version IDA confirms.
- **Item slot width changes:** v95 likely widens item slot types (parallel to task-028's `GW_CharacterStat` HP/MP widening). Flag every `int16`/`int32` item-slot field as a cross-version gate candidate.
- **Hidden constructor-signature ripples:** adding fields to encoder structs ripples to `atlas-channel` / `atlas-cashshop` / `atlas-inventory` handler call sites — verify build clean across services.
- **Audit-report absolute paths:** task-027/028/066 all had gitleaks bait from `/home/<user>/source/atlas-ms/atlas/` strings inadvertently captured in audit reports. Task 12 Step 4 scrubs.
