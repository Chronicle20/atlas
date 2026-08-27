# Review — task-240 plan task 24a (arm sweep, seam trace, deferred fixes)

Commit reviewed: `160191c25` (range `cda6eb58c..160191c25`). Prior commit
`cda6eb58c` was reviewed separately and is out of scope here.

Brief: `.superpowers/sdd/plan/task-24a-brief.md`
Report: `.superpowers/sdd/plan/task-24a-report.md`

## Scope confirmed

`git diff --stat cda6eb58c..160191c25` matches the report's "Files changed"
list exactly: seven files, all test files plus `ring/resource.go`. No
production code outside `ring/resource.go` was touched. Matches brief scope
(Steps 1–2, deferred items 1–3 as code, items 4–5 as write-up/confirmation
only).

## Step 1 — arm sweep

PASS. `grep -n 'l.Infof' services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`
returns zero matches (verified directly). `CashShopOperationBuyOtherPackage`
appears at its declaration
(`cash_shop_operation.go:42`) and its dispatch site (`cash_shop_operation.go:206`)
— verified directly, matches the report.

## Step 2 — seam trace

Spot-checked all six rows against source, not just the report's table.

- Producer (`services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go`)
  and consumer (`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`)
  are both **pre-existing** (from the already-approved `cda6eb58c`) — this
  commit changes neither. The field names in each provider function (e.g.
  `GiftPurchasedBody.RecipientName/TemplateId/Quantity/Price`,
  `RingPurchasedBody.PartnerName/TemplateId/Quantity/RingType/AssetId`,
  `PackagePurchasedBody.RecipientCharacterId/AssetIds/PackageTemplateId`,
  `LockerRebatedBody.CashId/Amount`) do genuinely match the corresponding
  consumer reads (`consumer.go:371-551`) — verified by reading both files in
  full, not just the report's line citations.
- The six new tests in `consumer_test.go` (lines added at the end of the
  file, ~1352 onward) each: (a) call the real `handleStatusEvent*` function
  with a populated event body using distinct, checkable values; (b) decode
  the **real wire packet** the handler wrote via `request.NewRequestReader`
  and the actual `cashpkt.*Done` decoder; (c) assert specific field values
  traced back to the event's input (e.g.
  `TestGiftPurchasedAnnouncesGiftDoneWithRecipientNameAndItem` asserts
  `body.RecipientName() == "Recipient"`, `body.ItemId() == 5010000`,
  `body.Quantity() == 1`, all distinct from any zero-value default). These
  are genuine pinning tests — a field-name typo or a mis-wired handler
  argument would fail them, not merely a build. Confirmed by running them
  fresh (`go test -count=1`, all pass, see Verification below).
- No producer/consumer disagreement found in this pass either — consistent
  with the report's conclusion.

## Step 3 — deferred items

### Item 1 — rollback regression tests (gift.go, ring.go)

PASS, and genuinely forces the rollback path. Both new subtests use
`databasetest.FailWritesOn(t, db, "cash_purchase_records", databasetest.WriteCreate)`
(`libs/atlas-database/databasetest/failwrites.go:25-46`), which registers a
real GORM `Before("gorm:create")` callback that injects an error on any
`cash_purchase_records` write — not a hand-rolled fake. Confirmed
`purchaserecord.Record` is genuinely the LAST write in both flows, after the
cross-account writes:
- `gift.go:118-149` — sender wallet debit (118), recipient asset create
  (140), `purchaserecord.Record` (149, the failure point).
- Ring flow (report cites step 8, consistent with the same
  `database.ExecuteTransaction` pattern; not independently re-verified
  line-by-line beyond confirming the same `FailWritesOn` call and post-assert
  shape).

Both subtests assert wallet balance unchanged AND the *other account's*
asset was rolled back, which `ring/administrator_test.go`'s
`TestCreatePairIsAtomic` (inner-batch-only) cannot reach — matches the
report's stated gap. Ran fresh (`go test -count=1 -run 'TestGift|TestPurchaseRing' -v`):
both subtests pass, `TestGift` (1.67s), `TestPurchaseRing` (2.04s).

### Item 2 — HTTP-level cross-tenant isolation test + the 404 fix

PASS on substance. `ring/resource_test.go`'s `TestAnotherTenantCannotReadTheseRings`
drives a real `mux.Router` via `httptest.Server` with tenant identified only
by request headers, mirroring `coupon/resource_test.go`'s pattern. Ran fresh
(`go test -count=1 -run TestAnotherTenantCannotReadTheseRings -v ./ring/...`):
passes, including the by-id 404 assertion.

The `handleGetRing` 404 fix (`ring/resource.go:107-110`) is an unbriefed
behavioral change (brief did not ask for it) but is judged on its merits per
the review brief:
- Matches the cited precedent (`coupon/resource.go:130-135`,
  `handleGetCoupon`) exactly: same `errors.Is(err, gorm.ErrRecordNotFound)`
  check, same `rest.WriteError(..., http.StatusNotFound, "no such <x>")`
  call shape.
- `GetById` scopes by `tenant_id` (confirmed the query in the test's
  captured SQL log: `WHERE (tenant_id = ... AND id = ...) AND
  cash_rings.tenant_id = ...`), so a cross-tenant lookup and a genuinely
  unknown id are indistinguishable — 404 is the correct answer for both, not
  an information-disclosure risk.
- Response-contract impact: only the status code for the not-found path
  changes (500→404); no other route (`GET /rings` list) is touched — the
  report and code both confirm `byCharacterIdPagedProvider` was already
  correctly tenant-scoped and untouched.
- This is a real, narrowly-scoped, well-tested bug fix, not scope creep in
  the harmful sense — it directly serves the item-2 finding it was found
  investigating.

### Item 3 — raw-error wrapping

PASS. Both `cash_shop_gift.go:117` and `cash_shop_ring.go:117` now force
`atlasmodel.ErrEmptySlice` into `giftRejectionReason`/mapper instead of the
raw error, mirroring the cross-world branch immediately below each
(verified both call sites read `giftRejectionReason(atlasmodel.ErrEmptySlice)`
already on the adjacent branch). Raw error is still logged via
`l.WithError(err)` before the change — diagnostics preserved. No handler
integration test exists for these functions (confirmed:
`cash_shop_gift_test.go`/`cash_shop_ring_test.go` contain only pure-function
and body-encoding tests, no `handleGift`/`handleRingPurchase` driver), so
the report's claim that the existing `TestGiftRejectionReason` mapper test
is the only available pin is accurate — the change itself is a one-line,
self-evidently-correct substitution.

### Items 4 and 5 — scope drift check

PASS — no scope drift. `git diff --stat` confirms `cashshop/equipslot.go`,
`cashshop/ring.go` (production), and
`socket/writer/character_data.go` are **absent** from this commit's diff.
Item 4's write-up is present verbatim in the report and correctly
distinguishes equipslot.go's real double-write risk from ring.go's
read-only (no double-write) shape. Item 5 states "no code change," confirmed
by the diff stat.

## A confirmed blocking defect: the format gate fails on 6 of 7 touched files

The brief states, in bold-equivalent emphasis: *"`gofumpt` formatting is a
hard gate on this branch — an earlier gate run failed on exactly one
unformatted test file. Format every file you touch."* The report claims:
*"`go run mvdan.cc/gofumpt@latest -l -w` run on every file touched ... All
touched files report clean after formatting."*

That claim is false. Running the repo's own authoritative formatting gate
(`tools/lint.sh --check --fmt`, which wraps the pinned `golangci-lint v2.12.2`
`fmt` command — the actual gate `tools/verify.sh` runs, not bare `gofumpt`)
against the touched modules fails with import-grouping violations in **6 of
the 7 touched files**:

- `services/atlas-cashshop/atlas.com/cashshop/cashshop/gift_test.go:8` — the
  five `atlas-cashshop/...` imports were moved into their own group ahead of
  `encoding/json`/`fmt`/etc instead of staying merged into the same
  (stdlib + intra-module) group, which is how they were formatted before
  this commit and how `coupon/resource.go` still is.
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/ring_test.go:11` —
  same defect, six imports.
- `services/atlas-cashshop/atlas.com/cashshop/ring/resource_test.go:11` — the
  new file's own `"atlas-cashshop/ring"` import is split into its own group
  after `bytes`/`encoding/json`/etc instead of merged with them.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go:3`
  — same defect, four imports.
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_gift.go:3`
  — `"context"`/`"errors"` were moved OUT of the merged group and given their
  own group ahead of the `atlas-channel/...` imports, inverting the
  pre-existing (and still-correct, per `coupon/resource.go`) ordering.
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_ring.go:3`
  — same defect; also demoted `messagecashshop "atlas-channel/kafka/message/cashshop"`
  into its own group.

Only `ring/resource.go` — the one production file changed — is actually
gofumpt/goimports-clean.

Reproduced directly with the pinned tool:
```
$ .cache/tools/bin/golangci-lint-v2.12.2 fmt --diff -c .golangci.yml ./socket/handler/...
(from services/atlas-channel/atlas.com/channel)
diff socket/handler/cash_shop_gift.go.orig socket/handler/cash_shop_gift.go
diff socket/handler/cash_shop_ring.go.orig socket/handler/cash_shop_ring.go
```
and
```
$ tools/lint.sh --check --fmt --go services/atlas-cashshop/atlas.com/cashshop
lint.sh: FMT FAIL — services/atlas-cashshop/atlas.com/cashshop
lint.sh: FAIL — 1 failing target(s): fmt:services/atlas-cashshop/atlas.com/cashshop
```
(and the equivalent for `services/atlas-channel/atlas.com/channel`).

This is not cosmetic: `tools/verify.sh`'s lint & format guard (which the
brief's own facts block lists as an applicable guard for this branch,
`applicable_guards=... lint & format guard`) will fail on this commit as-is,
so the branch is not currently in the "done means verified" state CLAUDE.md
requires, and the report's self-review section asserts a verification that
did not happen ("go run mvdan.cc/gofumpt@latest ... All touched files report
clean after formatting" — contradicted by the tree-wide gate). Per
CLAUDE.md's evidence & grounding section, an unverified claim of "verified"
is itself a defect, independent of the one-line fix required to clear it.

The fix is mechanical (`tools/lint.sh` fix mode, or `golangci-lint fmt`, on
the six files) and does not require touching any logic — but it was not
done, and the report claims it was.

## Correctness spot-checks (secondary)

- `go build ./...` clean on both `services/atlas-cashshop/atlas.com/cashshop`
  and `services/atlas-channel/atlas.com/channel` (module-local, foreground,
  as the brief requires).
- `go test ./...` (module-local) green on both modules; re-ran the specific
  new/changed test functions fresh with `-count=1` rather than trusting
  cache, all pass.
- No placeholder comment, stubbed handler, or unimplemented status response
  found in the diff.

## Not evaluable

- The rest of `PurchaseRingAndEmit`'s write ordering (item 1's precise
  step-8 line numbers) was taken from the report rather than independently
  re-derived line-by-line; the `FailWritesOn`/assertion shape and the
  fresh-test pass are independently confirmed, which is the part that
  matters for "does the test force the rollback path."
- `EQUIP_SLOT_INCREASED`'s and the extended `ERROR` row's pinning tests are
  pre-existing (not added this commit); not re-verified beyond confirming
  they still pass, since neither producer nor consumer code for those two
  events changed in this commit.

## Verdict rationale

Steps 1–2 and deferred items 1, 3, 4, 5 are done correctly and are backed by
real, non-trivial tests that would fail without the change. Item 2's fix is
correct, precedent-matching, and appropriately scoped despite being
unbriefed. The one blocking issue is procedural but real and
gate-breaking: the branch fails its own hard-gated formatting check on 6 of
7 touched files, directly contradicting the report's explicit claim that the
formatting pass was run and clean. This must be fixed (and re-verified with
the actual gate, not bare `gofumpt`) before the branch can be called done.
