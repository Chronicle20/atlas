# review: fix-ring-cash-id-resolves-to-zero (commit 0f1f8872b, range d8b0ec8d1..HEAD)

Reviewed against `docs/tasks/task-269-ring-pair-behavior/bug-ring-cash-id-resolves-to-zero.md`.

## Scope

`git diff --stat d8b0ec8d1..HEAD`:

```
docs/tasks/task-269-ring-pair-behavior/bug-ring-cash-id-resolves-to-zero.md | 102 +++
docs/tasks/task-269-ring-pair-behavior/fix-ring-cash-id-resolves-to-zero.md | 150 +++
services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_ring.go      |   4 +-
services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_ring_test.go |   8 +
services/atlas-cashshop/atlas.com/cashshop/ring/administrator.go           |  12 +-
services/atlas-cashshop/atlas.com/cashshop/ring/entity.go                  |  20 +-
services/atlas-cashshop/atlas.com/cashshop/ring/model.go                   |  12 +-
services/atlas-cashshop/atlas.com/cashshop/ring/processor.go               |  27 +-
services/atlas-cashshop/atlas.com/cashshop/ring/processor_test.go          |  38 ++
```

Confirmed `git diff d8b0ec8d1..HEAD -- services/atlas-channel` is empty — no
atlas-channel files touched. Scope matches the bug/fix documents; no
unrelated files touched.

## Checklist

### 1. Purchase path persists a non-zero CashId for BOTH halves, in the existing transaction, with a test

PASS.

- `cashshop/processor_ring.go:191-193` (post-fix):
  ```go
  ring.Half{CharacterId: characterId, AssetId: buyerAsset.Id(), CashId: buyerAsset.CashId(), ItemTemplateId: ci.ItemId()},
  ring.Half{CharacterId: partnerCharacterId, AssetId: partnerAsset.Id(), CashId: partnerAsset.CashId(), ItemTemplateId: ci.ItemId()},
  ```
  Both `buyerAsset`/`partnerAsset` come from `astP := asset.NewProcessor(p.l, p.ctx, tx)` (`processor_ring.go:174`), and `ring.NewProcessor(p.l, p.ctx, tx, p.chaP).CreatePair(tx, ...)` (`processor_ring.go:191`) uses the same `tx` — this is step 6/7 of the existing purchase transaction, no new transaction opened.
- `ring/administrator.go` `CreatePair` (`SetCashId(a.CashId)` / `SetCashId(b.CashId)`, lines added around 40/52) writes both halves in a single `db.Create` batch, consistent with the pre-existing "both rows or neither" comment at `processor_ring.go:186-189`.
- Test: `cashshop/processor_ring_test.go` extends `TestPurchaseRing/"creates two assets and one pair"` with `require.NotZero(t, buyerRings[0].CashId(), ...)`, `require.NotZero(t, partnerRings[0].CashId(), ...)`, and equality checks against `buyerCcm.Assets()[0].CashId()` / `partnerCcm.Assets()[0].CashId()`. Ran locally: `go test -run TestPurchaseRing ./cashshop/... -v` — PASS, including this subtest. Before the fix, `ring.Half.CashId` did not exist and the persisted value would have been the Go zero value, so this test is a real regression guard, not a tautology.

### 2. enrich prefers the persisted CashId, falls back to locker-asset lookup only when stored value is 0, with a test covering the failing case

PASS.

- `ring/processor.go` `enrich` (post-fix):
  ```go
  if half.CashId() == 0 {
      if a, err := astP.GetById(half.AssetId()); err == nil {
          b.SetCashId(a.CashId())
      }
  }
  ```
  `b := half.Builder()` (`ring/processor.go:125`) pre-populates the builder with `half.cashId` (confirmed in `ring/builder.go:113` — `Builder()` copies all of `Model`'s current fields), so when `half.CashId() != 0` the persisted value is kept as-is with no query; the `AssetId` lookup only fires when the stored value is exactly 0.
- Test: `ring/processor_test.go`, new subtest `"locker asset gone, persisted CashId resolves"` under `TestGetByCharacterIdEnrichesCashIdAndPartnerName` — creates a pair via `CreatePair` with `Half{..., CashId: 9001/9002, ...}` and **seeds no asset rows at all** (the exact "locker asset deleted / GetById returns an error" case named in the review brief), then asserts `buyerRows[0].CashId() == 9001`. Ran locally: `go test -run TestGetByCharacterIdEnrichesCashIdAndPartnerName ./ring/... -v` — PASS. A fallback-only (pre-fix) implementation would return 0 here since no asset exists to resolve, so the test genuinely pins the new behavior.

### 3. partnerCashId resolves from the SIBLING row's persisted value, not the sibling's locker asset alone

PASS.

- `ring/processor.go` sibling branch (post-fix):
  ```go
  if sibling.CashId() != 0 {
      b.SetPartnerCashId(sibling.CashId())
  } else if sa, err := astP.GetById(sibling.AssetId()); err == nil {
      b.SetPartnerCashId(sa.CashId())
  }
  ```
  `sibling` here is a `Model` returned by `GetByPairId`, populated via `Make` from `Entity.CashId` — the sibling's own persisted value, no extra query (self-review in the fix doc correctly notes this avoids an N+1).
- Same test as #2 asserts `buyerRows[0].PartnerCashId() == 9002` with no asset rows seeded for either half, proving the partner id also comes from the persisted sibling row, not an asset lookup.

### 4. The REST model carries the persisted value out to the channel

PASS — pre-existing, unaffected by this diff, and still correct after it.

`ring/rest.go` (`git diff` shows this file untouched by the current range) already has `CashId int64 \`json:"cashId"\`` / `PartnerCashId int64 \`json:"partnerCashId"\`` in `RestModel`, and `Transform` maps `m.CashId()` / `m.PartnerCashId()` straight through. Since `enrich` (checklist #2/#3) is what feeds those `Model` accessors and now returns the persisted, equip-durable value, the REST payload the channel joins on carries the fix through without any transformer change being needed.

### 5. No change leaked into atlas-channel

PASS. `git diff d8b0ec8d1..HEAD -- services/atlas-channel` returns nothing. `fix-ring-cash-id-resolves-to-zero.md` explicitly documents this as "not changed (ruling honored)."

## Verification run

```
$ go build ./...          # atlas-cashshop module — success, no output
$ go test ./ring/... ./cashshop/...
ok atlas-cashshop/ring       0.358s
ok atlas-cashshop/cashshop   1.670s
(+ sibling cashshop subpackages, no relevant tests, ok)
```

## Non-blocking findings

- `fix-ring-cash-id-resolves-to-zero.md:150` records commit hash `2fd87f4a4`,
  but the actual HEAD of this range is `0f1f8872b`. Stale — the branch was
  evidently rebased/re-committed after the fix doc was written and the doc
  was not updated. Cosmetic only; does not affect the code under review.

## Not evaluable

- Whether `atlas-channel`'s equipped-asset `CashId()` is numerically equal to
  the cashshop-side persisted `CashId` for the same underlying asset was not
  verified end-to-end by this unit — and the bug report and fix doc both
  explicitly scope that verification out ("this fix is scoped to
  atlas-cashshop only... has not been observed live post-fix"). Both docs
  agree this is deliberately deferred, not silently dropped, so it is
  recorded here as not evaluable rather than a defect. If it turns out to
  differ, the bug report already stipulates that is a separate bug.

## Verdict rationale

All five seam-crossing requirements in the review brief are satisfied with
`file:line` evidence and each has a test that would fail without the fix
(confirmed by inspection: `ring.Half.CashId` and `Entity.CashId` did not
exist before this change, so the new assertions have no pre-fix zero value
to coincidentally pass against). `selectPair` is untouched. The only findings
are a stale commit hash in the fix writeup (non-blocking) and one explicitly
scoped-out live-verification item (not evaluable, already flagged by the
authors themselves).
