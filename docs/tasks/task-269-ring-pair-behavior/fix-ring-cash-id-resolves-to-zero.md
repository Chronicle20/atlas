# fix-ring-cash-id-resolves-to-zero

Task: task-269-ring-pair-behavior · Branch: `task-269-ring-pair-behavior`

## Summary

Fixed `GET /api/rings` returning `cashId`/`partnerCashId` as `0` for a ring
that has been taken out of the cash locker and equipped. `cash_rings` used to
carry only the cash-locker `AssetId`, which stops resolving to a live asset
row once the ring is equipped (compartment.Release/equip removes the locker
row). `ring.ProcessorImpl.enrich` resolved `CashId` at read time via
`astP.GetById(half.AssetId())` only, so the lookup 404'd and the fail-soft
path left both ids at `0`. Because `atlas-channel`'s `selectPair` compares the
equipped asset's real `CashId` against these persisted values, every
comparison failed and no couple/friendship pair was ever selected — no
proximity effect, and `GetRingRecords` reported `OwnSN`/`PairSN` of `0`.

The fix persists each half's own asset `CashId` on `cash_rings` at purchase
time (the identifier survives compartment release/equip, unlike the locker
`AssetId`) and prefers that persisted value at read time, falling back to the
`AssetId` lookup only for rows written before this column existed.

## What changed

- `services/atlas-cashshop/atlas.com/cashshop/ring/entity.go`
  - Added `CashId int64` to `Entity` (`cash_rings.cash_id`, `not null default
    0`; AutoMigrate covers new deployments, no backfill — a PR-environment
    re-purchase produces a correct row). `AssetId` is unchanged: still the
    locker provenance.
  - `Make` now carries `e.CashId` into `Model.cashId`; `ToEntity` writes
    `m.cashId` back to `Entity.CashId`.
- `services/atlas-cashshop/atlas.com/cashshop/ring/model.go`
  - Updated `CashId()`'s doc comment: it is now persisted on `Entity` at
    purchase time, not computed at read time; `enrich` falls back to the
    `AssetId` lookup only for legacy rows where the persisted value is `0`.
- `services/atlas-cashshop/atlas.com/cashshop/ring/administrator.go`
  - Added `CashId int64` to `Half` (the caller reads it off the `asset.Model`
    returned by `astP.Create`/`CreateGift` at purchase time).
  - `CreatePair` now calls `SetCashId(a.CashId)` / `SetCashId(b.CashId)` on
    both builders so the persisted value lands on both `Entity` rows in the
    same insert.
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_ring.go`
  - `PurchaseRingAndEmit` step 7 now populates `ring.Half.CashId` from
    `buyerAsset.CashId()` / `partnerAsset.CashId()`, inside the existing
    purchase transaction (no new transaction, no new step).
- `services/atlas-cashshop/atlas.com/cashshop/ring/processor.go` (`enrich`)
  - `CashId`: only falls back to `astP.GetById(half.AssetId())` when
    `half.CashId() == 0` (the persisted value from `Make`); otherwise the
    persisted value — already the builder's default via `half.Builder()` —
    is kept as-is.
  - `PartnerCashId`: prefers the sibling row's own persisted `CashId()`
    (resolved by `Make` off its `Entity.CashId`, no extra query); falls back
    to `astP.GetById(sibling.AssetId())` only when the sibling's persisted
    value is `0`.
- Tests
  - `ring/processor_test.go` — new subtest
    `"locker asset gone, persisted CashId resolves"` under
    `TestGetByCharacterIdEnrichesCashIdAndPartnerName`: creates a pair with
    non-zero `Half.CashId` on both halves and deliberately seeds NO asset rows
    (reproducing the equipped-and-gone-from-the-locker case from the bug
    report), then asserts `enrich` still returns the persisted
    `CashId`/`PartnerCashId` rather than falling back to `0`.
  - `cashshop/processor_ring_test.go` — extended the existing
    `"creates two assets and one pair"` subtest of `TestPurchaseRing` with
    assertions that both halves persist a non-zero `CashId` at purchase, and
    that the persisted value equals the corresponding compartment asset's own
    `CashId()`.

## Not changed (ruling honored)

- `atlas-channel`'s `ring/processor.go` `selectPair` — untouched, per the
  brief's ruling. Its `h.CashId() != equippedCashId` comparison is correct;
  it was being fed zeros from the cashshop side.
- No backfill migration for existing `cash_rings` rows: AutoMigrate adds the
  column with default `0`, and rows written before this change fall back to
  the `AssetId` lookup exactly as before (which still works while the ring
  is in the locker, and still fails soft to `0` once equipped, matching
  today's behavior for pre-existing rows).

## Not yet answered (unchanged from the bug report)

- Whether `atlas-channel`'s equipped-side `CashId()` is numerically equal to
  the cashshop-side persisted `CashId` for the same asset was **not**
  verified end to end here — this fix is scoped to atlas-cashshop only, per
  the ruling. Both sides derive the id from
  `cashshop/inventory/asset/administrator.go`'s `generateUniqueCashId`
  applied to the same asset row, so they should be equal, but that has not
  been observed live post-fix. If they differ, that is a separate bug per
  the bug report's own note.

## Testing

Module-local only (`services/atlas-cashshop/atlas.com/cashshop`), per the
brief.

```
$ go build ./...
(no output — success)

$ go test ./ring/... ./cashshop/...
ok  	atlas-cashshop/ring	1.174s
ok  	atlas-cashshop/cashshop	7.531s
ok  	atlas-cashshop/cashshop/commodity	0.006s
ok  	atlas-cashshop/cashshop/inventory	0.016s
ok  	atlas-cashshop/cashshop/inventory/asset	0.009s
ok  	atlas-cashshop/cashshop/inventory/asset/reservation	(cached)
ok  	atlas-cashshop/cashshop/inventory/compartment	0.034s

$ go test ./...
ok  	atlas-cashshop	(cached)
... (all packages ok, no failures)
```

`gofmt -l .` reported nothing after formatting the two touched files
(`ring/entity.go`, `ring/administrator.go`) — no other file needed
reformatting.

## Self-review

- `ring.Half.CashId` and `Entity.CashId` both default to the Go zero value
  (`0`), which is indistinguishable from "not yet purchased under this fix" —
  intentional, per the brief: it is exactly the fallback trigger for legacy
  rows.
- `enrich`'s sibling branch now reads `sibling.CashId()` directly off the
  `Model` `GetByPairId` already fetched, instead of adding a second query —
  no new N+1.
- No `*_testhelpers.go` files added; tests use the existing `CreatePair` /
  `seedAsset` / builder-based construction already present in both test
  files.
- Builder pattern preserved: `Half` remains a plain struct (matches the
  pre-existing pattern — `Half` was never built via `ring.Builder`, only
  `Model`/`Entity` are), and the new field slots into the existing struct and
  call sites without touching the `Builder`/`validate` contract beyond adding
  `SetCashId` to the two existing `CreatePair` builder chains (which already
  had a `SetCashId` method on `Builder` from the enrichment work — reused,
  not duplicated).

## Files changed

- `services/atlas-cashshop/atlas.com/cashshop/ring/entity.go`
- `services/atlas-cashshop/atlas.com/cashshop/ring/model.go`
- `services/atlas-cashshop/atlas.com/cashshop/ring/administrator.go`
- `services/atlas-cashshop/atlas.com/cashshop/ring/processor.go`
- `services/atlas-cashshop/atlas.com/cashshop/ring/processor_test.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_ring.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_ring_test.go`

## Commit

`0f1f8872b` — fix(cash-shop): persist ring half CashId so pairs resolve after equip
