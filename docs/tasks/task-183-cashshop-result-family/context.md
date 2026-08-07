# Context — task-183 CashShopOperation Result Family

Companion to `plan.md`. Key files, decisions, and dependencies for the executor.

## What this task is

Model, enumerate, and verify **every** arm of the `CCashShop::OnCashItemResult` mode-prefix
dispatcher (56 arms — 9 already modeled, **47 new**) as discrete clientbound codecs across all 9
client versions, driven by a full IDA RE pass with gms_v95.1 as the reference. It is a **codec +
enumeration + verification** task — **no new cash-shop feature logic** (no gifting/coupon/gachapon/
transfer/maple-point domain flows). Body funcs are the usable API; producers are wired only where a
flow already emits (default: none).

## Reference implementation files (copy these patterns)

| Concern | Reference |
|---|---|
| Discrete struct (mode + reason) | `libs/atlas-packet/cash/clientbound/shop_operation_result.go` → `LoadInventoryFailure` |
| Discrete struct (mode + type + count) | same file → `InventoryCapacitySuccess` |
| Discrete struct (mode + item blob) | `shop_inventory.go` → `CashShopPurchaseSuccess`; `shop_item_moved.go` → `CashItemMovedToInventory` |
| List arm (mode + fixed list) | `shop_operation_result.go` → `WishListLoad` / `WishListUpdate` |
| Body func — success (`WithResolvedCode`) | `shop_operation_body.go:99` `CashShopInventoryCapacityIncreaseSuccessBody` |
| Body func — failure (`ResolveCode` + `errors`) | `shop_operation_body.go:89` `CashShopLoadInventoryFailureBody` |
| `run.go` `#`-entry map | `tools/packet-audit/cmd/run.go` — cash block ~line 2343 (`case "CCashShop::OnCashItemResult#…"`) |
| Byte-fixture test + `packet-audit:verify` markers | `shop_operation_result_test.go` (`TestLoadInventoryFailureByteFixture`, `loadInventoryFailureModes`, `variantKey`, `bytesEqual`) |
| yaml source of truth | `docs/packets/dispatchers/cash_shop_operation.yaml` |
| Template operations map | `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` (`CashShopOperation` writer) |

## The 56-arm set

Grounded from the user-supplied v95 `OnCashItemResult` decompile — reproduced in **design.md
Appendix A** (v95 mode bytes, hex). 9 existing (`LOAD_INVENTORY_SUCCESS/FAILURE`, `LOAD_WISHLIST`,
`UPDATE_WISHLIST`, `PURCHASE_SUCCESS`, `INVENTORY_CAPACITY_INCREASE_SUCCESS/FAILED`,
`CASH_ITEM_MOVED_TO_INVENTORY/CASH_INVENTORY`) + 47 new. `arm-catalog.md` (Wave 0) is the single
source of wire truth for every downstream layer.

## Locked scope decisions (design §1.2)

- **D1 — in-switch arms only.** Exactly the v95 switch cases. Sibling **separate-opcode**
  dispatchers (`OnCashItemGachaponResult`, `OnGiftMateInfoResult`, `OnCheckNameChangePossibleResult`,
  `OnCheckTransferWorldPossibleResult`, `OnChargeParamResult`, `OnPurchaseExpChanged`, `OnOneADay`,
  `OnNoticeFreeCashItem`, `CASHSHOP_REGISTER_NEW_CHARACTER_RESULT`, `CASHSHOP_GACHAPON_STAMP_RESULT`)
  are OUT (own matrix rows). **Naming trap**: in-switch `…ResCashGachaponOpenDone`/`…ResTransferWorldDone`/
  `…NameChangeResBuyDone` are IN; the `OnCheck…`/`OnCashItemGachaponResult` siblings are OUT.
- **D2 — full legacy RE, subset arms + `n-a`.** RE all 4 legacy IDBs; model+verify arms that exist,
  record the rest `n-a` with enumeration evidence. Legacy is structurally divergent (v48/v61 =
  `COutPacket(160)+Enc1(sub-op)` subset; v72 op 291 / v79 op 303).
- **D3 — single aggregate op-row.** One `CASHSHOP_OPERATION` matrix row, worst-of-all-arms
  (FIELD_EFFECT model). Adding 47 unverified arms **regresses** the currently-`✅` v83/v84/v87/v95/jms
  cells until every arm×version is verified-or-`n-a`. Expected + accepted; PR description must state it.

## Key invariants (DISPATCHER_FAMILY.md — CI-enforced by `dispatcher-lint`)

- INV-1 / AP-1: discrete struct per mode, even wire-identical arms; bodyless arms are still
  `struct { mode byte }`.
- INV-2 / AP-2/3 / DOM-25: mode byte config-resolved (`WithResolvedCode("operations", KEY, func(mode byte)…)`);
  no `mode: 0x` literal, no `func(_ byte)`. Reason byte config-resolved from the `errors` table.
- INV-3 / AP-4: no body func takes a caller-picked op/code/mode/key — nor any param flowing into the
  `operations` key. (A `message string` → `errors` key is fine.)
- INV-4/5: no dangling `#`-entry; every struct constructed by a body func.
- Version gating: `MajorAtLeast`/`MajorVersion()`, never raw `> N`; straddling gates need a fixture on
  BOTH adjacent versions (`gates.yaml` + `gate-check`).

## Dependencies & tooling

- **RE**: `ida-pro-mcp`. Confirm the instance matches the version (`select_instance`) before reading.
  Name every touched function; `idb_save`. Unresolved fname → **stop-and-ask** (never invent). See
  memory: IDA-MCP API notes, per-version IDB packet-naming notes.
- **`packet-audit` subcommands** (run `go run ./tools/packet-audit <cmd>` from worktree root):
  `operations` (regen templates from yaml) / `operations --check` (drift gate) · `dispatcher-lint` ·
  `fname-doc --check` · `gate-check --check` · `matrix --check` (hard gate).
- **Verification leaf step**: `packet-verifier` agent / `/verify-packet` per `VERIFYING_A_PACKET.md`.
  Batch per IDB; pin verifier/review subagents to a **cheaper model** (Sonnet/Haiku — cost rule).
- **Guards** (worktree root): `tools/lint.sh --check`, `tools/redis-key-guard.sh`,
  `tools/goroutine-guard.sh`, `tools/template-opcode-order-guard.sh`.

## Gotchas / grounding notes

- **`0x4D` gift TODO** (`shop_operation_body.go:80` `CashShopCashGiftsBody` → `NewCashShopGifts(0x4D)`,
  `shop_inventory.go:172` stub `CashShopGifts` writing `mode + short(0)`). Only caller is a
  **commented-out** line in `services/atlas-channel/.../cash_shop_entry.go:102` → no live consumer.
  Task 1.3 replaces the stub with the RE'd `GiftDone` (0x6B) shape + config-resolved body; delete the
  literal and the TODO.
- **v84 dispatcher is unnamed** (DEVM build): its column is template-derived (uniform +3 vs v83, per
  the yaml header note). Mark the v84 catalog column source `template-derived`; do not fabricate names.
- **Templates in EVERY supporting version** — a key missing from a version template → the server drops
  the packet at emit (`bug_new_opcodes_not_in_live_tenant_config`). `operations` regen + `--check`
  enforces this; templates are **regenerated**, never hand-edited.
- **`n-a` only via full case enumeration** at the switch address (address cited), never assumed
  (`feedback_dispatcher_mode_byte_is_false_pass`, `feedback_verify_packets_not_cross_version_opcodes`).
- **No `go.mod` change expected** → no `docker buildx bake` needed (confirm at close-out). Only
  `libs/atlas-packet` code + `atlas-configurations` seed JSON + `tools/packet-audit/cmd/run.go`.

## Wave sequencing (design §8)

0. RE all 9 IDBs → `arm-catalog.md` + `coverage-manifest.yaml` + extended yaml. **HARD review
   checkpoint before any codec.**
1. Modern-5 codecs (discrete struct + body func + `run.go` entry + round-trip test per arm; delete
   `0x4D`). `dispatcher-lint` + `operations --check` + `go build/vet/test -race` clean.
2. Modern-5 verification → op-row modern columns back to `✅`.
3. Legacy codecs + verification → legacy columns `✅`/`n-a`.
4. Close-out: full verification suite + code review (`plan-adherence` + `backend-guidelines` +
   `packet-completeness-critic`) before PR.
