# Review — Task 8: APPLY_WISHLIST (mode 33/35)

Commit under review: `4d121b8` ("feat(channel): implement APPLY_WISHLIST").

Note on scope: the range given, `4871c8581..4d121b8`, actually contains two
commits (`39de4adc1 fix(cashshop): bound backfill's boot-time query...` and
`4d121b888 feat(channel): implement APPLY_WISHLIST`) — `39de4adc1` is the
concurrent Task 6 fix mentioned in the dispatch, not part of Task 8. This
review covers only `4d121b8` itself (`git show 4d121b8 --stat`), which
touches exactly the two files the brief named:
`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`
and the new `cash_shop_apply_wishlist_test.go`.

## Ruling 1 — D2a = empty body, no `libs/atlas-packet` change

PASS. `git diff 4871c8581..4d121b8 -- libs/atlas-packet` is empty — no
`shop_operation_apply_wishlist.go`/`_test.go` were created, matching the
controller's ruling. `derivation.md:104-108` confirms D2a is RESOLVED: empty
(`CCashShop::ApplyWishListEvent` at `0x482ea0` only emits `Encode1(0x23)`, no
further bytes).

## Ruling 2 — D2b = UPDATE_WISHLIST (mode 98)

PASS. `derivation.md:139-145` states D2b RESOLVED as `UPDATE_WISHLIST` (mode
98). The handler announces `cashcb.CashShopWishListUpdateBody(sns)` at
`cash_shop_operation.go:198`. The test's options map uses
`cashcb.CashShopOperationUpdateWishlist: float64(98)`
(`cash_shop_apply_wishlist_test.go:22-24`) and asserts `body[0] == 98`
(`cash_shop_apply_wishlist_test.go:47`) — consistent, no LOAD_WISHLIST (92)
anywhere in the diff.

## Ruling 3 — the `inferred` hedge

PASS, and correctly worded. `cash_shop_operation.go:179-184`:

```go
// APPLY_WISHLIST (mode 33/35) carries no bytes after the mode byte
// (derivation.md D2a: RESOLVED, empty body). The reply arm is
// inferred (derivation.md D2b: RESOLVED but flagged INFERENTIAL,
// reached via request-in-flight-latch analysis rather than a
// client correlation table) to be UPDATE_WISHLIST (mode 98), the
// same arm SET_WISHLIST already answers with.
```

This states the arm choice is inferred, names the derivation section, and
gives the actual method (latch analysis, not a correlation table) — it does
not upgrade D2b into a confident/established-fact claim. The test file's
doc comment (`cash_shop_apply_wishlist_test.go:13-16`) repeats the same
"RESOLVED but flagged INFERENTIAL" wording. Matches
`derivation.md:169-173`'s own "Residual caveat (honest)" language.

## No invented values

PASS. Verified against `libs/atlas-packet/cash/clientbound/shop_operation_body.go`:
- `CashShopWishListUpdateBody` — declared at `shop_operation_body.go:189-194`, real.
- `CashShopOperationUpdateWishlist` = `"UPDATE_WISHLIST"` — `shop_operation_body.go:19`, real.
- `CashShopLoadWishFailedBody` — declared at `shop_operation_body.go:236-243`, real
  (resolves `CashShopOperationLoadWishFailed` = `"LOAD_WISH_FAILED"`,
  `shop_operation_body.go:26`).

No non-existent identifier was referenced (contrast Task 7's brief, which
named one that didn't exist).

## Blocking finding — mismatched success/failure response pair breaks the latch derivation itself identified

`cash_shop_operation.go:188` answers a wishlist-read failure with
`cashcb.CashShopLoadWishFailedBody("unknown_error")` — the `LOAD_WISH_FAILED`
arm (mode 93). This is what the brief's Step 5 literally instructed
("A read error announces `cashcb.CashShopWishListLoadFailedBody`/the
`LOAD_WISH_FAILED` arm"), and the implementer followed it, resolving the real
constructor name correctly. But it directly contradicts D2b's own evidence in
the same derivation document:

`derivation.md:147-167` establishes that `ApplyWishListEvent` sets the
client's `m_bCashShopRequestSent` latch on send, and that **only two** arms
ever clear it: `OnCashItemResSetWishDone` (mode 98) and
`OnCashItemResSetWishFailed` (mode 99) — the "SetWish" pair.
`OnCashItemResLoadWishDone` (92) and `OnCashItemResLoadWishFailed` (93) —
the "LoadWish" pair — clear the latch in **neither** address range swept
(`derivation.md:155-167`). `derivation.md:169-173`'s own honesty caveat spells
out the failure mode: "If a live capture ever shows the server answering mode
35 with 92, the client would hang on the next cash-shop action rather than
mis-render."

The shipped code answers success with the SetWish-pair member (98,
`UPDATE_WISHLIST`, correctly clears the latch per the evidence) but answers
failure with the LoadWish-pair member (93, `LOAD_WISH_FAILED`, does **not**
clear the latch per the same evidence). A character whose wishlist read fails
(any REST hiccup) receives a reply that — by the derivation's own analysis —
leaves `m_bCashShopRequestSent` set, wedging every subsequent cash-shop
request for that client session. This is the exact symptom
`derivation.md:169-173` names as the diagnostic to watch for, reached via the
mismatched pairing this diff introduces.

The consistent choice, given D2b's reasoning, is `cashcb.CashShopSetWishFailedBody`
(`SET_WISH_FAILED`, mode 99, declared at `shop_operation_body.go:246-251`) —
the failure member of the same SetWish pair the success path already uses.

This finding traces back to a brief instruction that itself conflicts with
the derivation it cites, but the defect ships in `4d121b8` regardless of
origin, and no test in this commit exercises the failure-path arm choice (the
handler test only pins the success-path encoder output;
`cash_shop_apply_wishlist_test.go` never triggers `GetByCharacterId`
returning an error), so nothing here would have caught it.

## Other checks

- **No stub left behind.** PASS. Both branches (success at
  `cash_shop_operation.go:198`, error at `:188`) always announce a body;
  no silent `return`.
- **Empty-wishlist case.** PASS. `cash_shop_apply_wishlist_test.go`'s
  `empty wishlist` subtest asserts a full 41-byte body (mode + 10
  zero-padded uint32 slots), not zero bytes — matches FR-WISH-3's intent.
  The handler itself builds `sns := make([]uint32, len(wl))` unconditionally,
  so `wl == nil`/empty still produces a valid `[]uint32{}` argument to
  `CashShopWishListUpdateBody`, which the test confirms pads correctly.
- **Version gating.** N/A — no new codec, no `MajorAtLeast`/`MajorVersion`
  call added by this diff. Nothing to flag.
- **No Kafka.** PASS. No command emission, no `ErrorEventBody` arm, no new
  consumer registration in the diff — pure REST read via
  `wishlist.NewProcessor(l, ctx).GetByCharacterId`, matching Skeleton B.
- **Tenant scoping.** Not evaluable as a new-code concern — `GetByCharacterId`
  (`cashshop/wishlist/processor.go:47`) is unmodified by this diff (brief
  correctly lists it read-only) and delegates to `byCharacterIdUrl(p.ctx, ...)`
  / `requests.DrainProvider`, the same tenant-context path the pre-existing
  `SET_WISHLIST` arm already relies on at `cash_shop_operation.go:79-82`. No
  new scoping surface was introduced.
- **Builder pattern / no `*_testhelpers.go`.** Not applicable — the test uses
  a plain table-driven loop over an inline options map; no test-only
  constructor file was added. Per plan-mandated shapes, not a finding either
  way.

## Build/test verification (run directly, not trusted from the report)

```
cd services/atlas-channel/atlas.com/channel && go build ./...   # exit 0, no output
cd services/atlas-channel/atlas.com/channel && go test ./...    # all packages ok, including socket/handler
```

Confirmed clean.

## Not evaluable

- None beyond the tenant-scoping note above, which is adequately covered by
  the read-only status of `processor.go` in this diff.

## Verdict rationale

The three controller rulings (D2a empty body / no libs change, D2b mode 98,
the inferred hedge) are all honoured precisely and honestly. The blocking
finding is a genuine correctness defect in the error path that the
derivation's own evidence contradicts — not a hedge-wording problem, a
functional one, reachable by an ordinary REST failure and untested by this
commit.
