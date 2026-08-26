# Review: bug-cash-shop-live-testing (bd7e2e11b..HEAD)

Commits reviewed: `f64b454a3` (Defect A + B, Go), `bcb403bd7` (Defect C, TS).

## Defect A — table-qualified upsert (purchaserecord/administrator.go)

**PASS.**

- `administrator.go:42` — `"count": gorm.Expr(e.TableName() + ".count + 1")`, i.e. `cash_purchase_records.count + 1`. `entity.TableName()` (`entity.go:29`) is referenced, not hard-coded a second time, per the brief.
- Confirmed the generated SQL is correct for the Postgres construct in question: ran `go test ./purchaserecord/... -run TestRecordConflictUpdateIsTableQualified -v` — PASS, and the dry-run SQL string shows `ON CONFLICT (...) DO UPDATE SET ... = cash_purchase_records.count + 1`. Table-qualifying the right-hand side is exactly what Postgres requires to disambiguate `count` from `excluded.count` inside `ON CONFLICT ... DO UPDATE` (SQLSTATE 42702 in the bug report).
- Confirmed the regression test is honest, not decorative: reverted the fix locally (`gorm.Expr("count + 1")`) and re-ran the same test — it fails with `expected ON CONFLICT DO UPDATE to table-qualify count, got SQL: ... SET \`count\`=count + 1 ...`. Restored the file after (`git checkout -- purchaserecord/administrator.go`); working tree is clean.
- `TestRecordUpsertsAndCounts` (sqlite, functional) still passes unmodified, so the existing behavioral coverage is preserved.

## Defect B — inventory-type math + out-of-range guard (cashshop/processor.go)

**PASS.**

- `processor.go:332` — `inventoryType := inventory.Type((ci.ItemId() - 9110000) / 1000)`, correct grouping; `(9114000-9110000)/1000 = 4`.
- Guard added at `processor.go:353-357`, the *first* statement inside `PurchaseInventoryIncrease`, before `database.ExecuteTransaction` (`processor.go:358`) which performs the wallet debit (`w = w.Purchase(currency, cost)` / `p.walP...Update(...)` inside the tx). Traced by hand: `isValidInventoryType` runs, and only on success does execution reach the transaction that debits the wallet — so the guard genuinely runs before the debit, fixing the "cash taken for a nonexistent compartment" defect.
- Reject-path emission matches the sibling reject path exactly: `processor.go:354` and `processor.go:402` (the existing `txErr` path) both call `producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR", uuid.Nil))` — same topic, same provider, same error code, on the direct producer (not outbox), consistent with the code comment that UNKNOWN_ERROR reflects no committed state change. This is what `atlas-channel` already expects for this class of failure.
- New tests `processor_inventoryincrease_test.go`: `TestPurchaseInventoryIncreaseByItemComputesETCType` (asserts wallet debited by 6800, outbox event `InventoryType == 4`) and `TestPurchaseInventoryIncreaseByItemRejectsOutOfRangeItem` (asserts wallet untouched at 96000, zero outbox entries, one direct `UNKNOWN_ERROR` event). Both reuse existing helpers in `processor_test.go` (`purchaseTestDatabase`, `startPurchaseCharacterServer`, `startPurchaseCommodityServer`, `seedPurchaseWallet`, `purchaseOutboxEntries`, `captureDirectPurchaseEvents`, `purchaseErrorEvents`) rather than duplicating test scaffolding. Ran `go test ./cashshop/... -run TestPurchaseInventoryIncrease -v` — both PASS.

### Non-blocking note (not a defect in the fixed scope)

`ci.ItemId()` is `uint32`; `inventory.Type` is `int8` (`libs/atlas-constants/inventory/constants.go:9`). If a commodity's `ItemId()` were ever below `9110000`, `ci.ItemId() - 9110000` underflows the `uint32` before the `int8` truncation, and the truncated byte could coincidentally land in `[1,5]` and pass `isValidInventoryType`, silently defeating the guard for a non-slot-expansion item. This path is only reachable through the server's own commodity table (not client-supplied), which the report treats as authoritative, so it is not exercised by the reported symptom and is outside the brief's explicit repro. Flagging for awareness only; not blocking.

## Defect C — accounts.service.ts envelope unwrap

**PASS.**

- `accounts.service.ts:129-136`: `api.patch<ApiSingleResponse<Account>>(...)` then `transformAccount(response.data)` — same generic/unwrap idiom as `coupons.service.ts:247`, same `ApiSingleResponse` import source (`@/types/api/responses`).
- Checked the only non-test call site, `useAccounts.ts:270` (`accountsService.updateAccountBirthDate(account, birthDate)` inside a `useMutation`), which expects `Promise<Account>` — unaffected, still receives the unwrapped `Account`.
- New test in `accounts.service.test.ts` mocks `api.patch` to resolve `{ data: {...} }` (the real envelope shape) and asserts `result.attributes.birthDate`/`loggedIn` — this would throw `cannot read properties of undefined` pre-fix, since `transformAccount` immediately reads `data.attributes.loggedIn`. Ran `npx vitest run src/services/api/__tests__/accounts.service.test.ts` — 3/3 PASS. Ran `npx tsc --noEmit -p .` — no errors touching `accounts.service.ts`.

## Forbidden scope check

`git diff --stat bd7e2e11b..HEAD -- services/atlas-configurations/seed-data/templates/` — empty. No `UNKNOWN_ERROR` key or any other change was added under that path. Confirmed via `git log` on the same path range as well — no commits touch it.

## Scope confirmation

Diff matches the three defects described in the brief exactly: `purchaserecord/administrator.go` + its test (Defect A), `cashshop/processor.go` + its new test file (Defect B), `accounts.service.ts` + its test (Defect C). No unrelated files touched. The bug/report markdown files themselves are also part of the diff (documentation, not reviewed as code).

## Verdict rationale

All three defects are fixed correctly, each backed by a test that was verified (by hand, via revert-and-rerun for Defect A, and by tracing the guard's position for Defect B) to actually pin the new behavior rather than pass vacuously. No forbidden seed-data change. One non-blocking theoretical edge case noted for Defect B, outside the reported repro's reach.
