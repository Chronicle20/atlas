# Completeness Critic — task-183-cashshop-result-family

Branch: `task-183-cashshop-result-family`
Base: `cdfb71aa317cbec966709a85f5811cb886a18bb5` (verified ancestor of HEAD)

**Verdict: 1 finding — CLAIMED-BUT-UNVERIFIED (pre-existing over-claim, surfaced by this task's own RE work). No CHANGED-BUT-UNCLAIMED findings.**

## Manifest resolution

`docs/tasks/task-183-cashshop-result-family/coverage-manifest.yaml`:
- `ops`: `CASHSHOP_OPERATION` (D3 aggregate op-row)
- `versions`: all 9 (`gms_v48/61/72/79/83/84/87/95`, `jms_v185`)
- `out_of_scope`: `CASHSHOP_CASH_ITEM_GACHAPON_RESULT`, `CASHSHOP_GIFT_INFO_RESULT`, `CASHSHOP_CHECK_NAME_CHANGE`, `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT`, `CHARGE_PARAM_RESULT`, `CASHSHOP_PURCHASE_EXP_CHANGED`, `ONE_A_DAY`, `CASHSHOP_NOTICE_FREE_CASH_ITEM`, `CASHSHOP_REGISTER_NEW_CHARACTER_RESULT`, `CASHSHOP_GACHAPON_STAMP_RESULT`, `model/asset`

Resolving `CASHSHOP_OPERATION` against `status.json` rows finds two clientbound op-rows sharing the same representative packet `cash/clientbound/CashBuyFailed` and `fnames: [CCashShop::OnCashItemResult]`:
- `CASHSHOP_OPERATION` (clientbound) — the D3 aggregate row this task claims.
- `CASHSHOP_CASH_ITEM_RESULT` (clientbound) — the v48-only alias row (not separately listed in `ops`, but same packet path / same dir `cash/clientbound`, so it resolves into `claimedPackets` via the dir match; not an unclaimed codec).

`claimedPackets` = `{cash/clientbound, cash/serverbound}` (dir-level, since `CASHSHOP_OPERATION` also has a serverbound row at `cash/serverbound/CashShopOperationGetPurchaseRecord`, untouched by this branch).

## Step 2 — CHANGED-BUT-UNCLAIMED

**Codecs.** All non-test `.go` changes under `libs/atlas-packet` are confined to `cash/clientbound/`:
```
libs/atlas-packet/cash/clientbound/shop_inventory.go
libs/atlas-packet/cash/clientbound/shop_operation_body.go
libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go
libs/atlas-packet/cash/clientbound/shop_operation_result_gachapon.go
libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go
libs/atlas-packet/cash/clientbound/shop_operation_result_misc.go
libs/atlas-packet/cash/clientbound/shop_operation_result_slots.go
libs/atlas-packet/cash/clientbound/shop_operation_result_transfer.go
```
All match `claimedPackets` via the `cash/clientbound` dir. **No unclaimed codec.**

**Gates.** Two gate additions found via `git diff ... | grep -E '(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'`:
- `shop_operation_result_gift.go:99`: `+\treturn t.Region() == "GMS"` (nxCashSpent region gate, jms divergence per commit `1d2734ff4`).
- `shop_operation_result_slots.go:161`: `+\treturn t.Region() == "GMS" && t.MajorVersion() == 72` (v72 one-version IncBuyCharacterCountSuccess shape gate, commit `ea62f1c3a`), with an explanatory comment at line 159: `// not MajorAtLeast — this is a one-version divergence, not a floor.`

Both files are in `cash/clientbound/` → in `claimedPackets`. **No unclaimed gate.** No other opcode/gate-adjacent files (`libs/atlas-packet` elsewhere) were touched.

**Matrix delta.** `git diff status.json` shows only `toolSha`/`exportHashes` regeneration and one representative-packet-pointer edit on the two aggregate rows (`cash/clientbound/CashCashItemMovedToCashInventory` → `cash/clientbound/CashBuyFailed` for both `CASHSHOP_OPERATION` and `CASHSHOP_CASH_ITEM_RESULT`). **No `cells[...].state` transitions** base→HEAD — the branch's mid-flight un-graduate (`a21a0834b`) / re-graduate (`698e4fca3`) commits cancel out net-net. No unclaimed matrix delta.

**Out-of-scope siblings.** Confirmed untouched: no diff hits for `CashCashItemGachaponResult`, `CashGiftInfo`, `CheckNameChange`, `CheckTransferWorldPossible`, `ChargeParam`, `PurchaseExpChanged`, `OneADay`, `NoticeFreeCashItem`, `RegisterNewCharacterResult`, `GachaponStampResult` in the branch's `libs/atlas-packet` or `status.json` deltas. Sibling separate-opcode dispatchers correctly left alone.

## Step 3 — CLAIMED-BUT-UNVERIFIED

Final `status.json` cells for `CASHSHOP_OPERATION` (clientbound) — all 9 versions:

| version | state | note |
|---|---|---|
| gms_v48 | n-a (opcode -1) | correctly n-a: v48 tracks this packet under the sibling `CASHSHOP_CASH_ITEM_RESULT` op-row instead (opcode 256), which is `verified`. Net coverage across the alias pair is complete for v48, but the manifest's `ops` list names only `CASHSHOP_OPERATION` — it does not document the `CASHSHOP_CASH_ITEM_RESULT` alias or the v48 n-a/split explicitly in `fields`. Not a functional gap; flagged as a minor manifest-documentation gap only. |
| gms_v61 | verified (255) | OK |
| gms_v72 | verified (291) | OK, but see stray-marker finding below |
| gms_v79 | verified (303) | OK |
| gms_v83 | verified (325) | OK |
| gms_v84 | verified (332) | OK |
| gms_v87 | verified (342) | OK |
| gms_v95 | verified (384) | OK |
| jms_v185 | verified (356) | OK |

Op-row is `✅` (verified or legitimately n-a-with-alias-coverage) across all 9 declared versions. **No hard CLAIMED-BUT-UNVERIFIED at the op-row level.**

### Stray v72 arm-verify marker — over-claim (pre-existing, surfaced by this task)

`libs/atlas-packet/cash/clientbound/v72_test.go:36`:
```go
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v72 ida=0x470e13
```
This marker is **unchanged by task-183** (`git diff $BASE...HEAD -- libs/atlas-packet/cash/clientbound/v72_test.go` is empty — file untouched on this branch; last touched by PR #971, task-096 lineage). It claims client-validated coverage for the `CashItemMovedToInventory` arm at `gms_v72`, backed by a `TestCashShopOperationArmsV72` byte-equality-vs-v83 assertion and an evidence record (`docs/packets/evidence/gms_v72/cash.clientbound.CashCashItemMovedToInventory.yaml`, `verifies: v72_test.go#TestCashShopOperationArmsV72`).

**This task's own RE work contradicts the marker.** `docs/tasks/task-183-cashshop-result-family/arm-catalog.md:179-184`:
> "v72 does not have this arm at all — `CASH_ITEM_MOVED_TO_INVENTORY` and `MOVE_L_TO_S_FAILED` are both `n-a` in v72 (proven by full switch enumeration + `func_query name_regex="MoveLtoS"` returning zero hits anywhere in the binary — only the opposite `MoveStoL` direction exists in this build)."

The pre-existing per-packet audit report agrees and predates task-183: `docs/packets/audits/gms_v72/CashCashItemMovedToInventory.md` — **Verdict ❌**, row 0: `IDAOp: "unresolved"`, `Note: "IDA read-order unresolved: function not found in IDB"`. The JSON twin (`docs/packets/audits/gms_v72/CashCashItemMovedToInventory.json`) records the same `"Verdict": 2` (fail) with `"IDAComment": "function not found in IDB"`.

So the branch inherits a codec-level test that asserts wire-format equality for a mode the client never dispatches to on v72 — a byte-equality tautology (v72 body == v83 body, both gated only on `MajorVersion()>12`) standing in for genuine client-validated proof, while the actual per-packet audit already says the function doesn't exist in this IDB. This is exactly the `bug_matrix_roundtrip_fixture_false_verify` pattern: round-trip/cross-version equality is not client validation.

| op | version | actual state | recommendation |
|---|---|---|---|
| `CashCashItemMovedToInventory` (arm of `CASHSHOP_OPERATION`) | `gms_v72` | over-claimed verified (marker + evidence record) vs. proven `n-a` per this task's own catalog and the pre-existing ❌ audit report | Remove the stray `packet-audit:verify ... packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v72` line at `v72_test.go:36` (and the `MovedToInventory` arm case at line 58 from the byte-equality table, since the arm doesn't exist on this client at all — it isn't a "same body, different dispatch" case, it's absent). Delete or mark n-a the corresponding evidence record `docs/packets/evidence/gms_v72/cash.clientbound.CashCashItemMovedToInventory.yaml`. This is pre-existing (task-096/PR#971) and not introduced by task-183, but task-183's RE explicitly disproves it and should not ship a PR that leaves a now-provably-false verify marker sitting untouched in a file this task otherwise heavily edited (`v72_test.go` itself wasn't touched, but sibling `shop_operation_result_*` files in the same package were) — recommend fixing in this task rather than deferring, per the no-deferring-producible-work rule. |

No other arm-level over/under-claims were found in the sampled per-arm audit reports checked.

## Summary

- CHANGED-BUT-UNCLAIMED: **0**
- CLAIMED-BUT-UNVERIFIED: **1** (the pre-existing `CashCashItemMovedToInventory`/`gms_v72` stray marker, contradicted by this task's own arm-catalog RE and a pre-existing ❌ audit report)
- Manifest documentation gap (informational, not a hard finding): the `gms_v48` n-a/alias split via `CASHSHOP_CASH_ITEM_RESULT` is not explicitly noted in the manifest's `fields`.
