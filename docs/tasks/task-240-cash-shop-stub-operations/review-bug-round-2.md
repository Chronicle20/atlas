# Review — bug-cash-shop-live-testing-round-2, Defects D and E

Range reviewed: `fc9ad271c..59916a651` (commits `648310b18`, `59916a651`
only). `ff8660466` (docs-only, on top) is out of scope and was not
evaluated. Defects F and G and the "Unresolved symptom" section are
deliberately unimplemented per the brief; their absence is not a finding.

## Scope confirmed

`git diff --stat fc9ad271c..59916a651` touches exactly five files, split
cleanly across the two commits:

- `648310b18` (Defect D): `services/atlas-cashshop/.../cashshop/processor.go`,
  `services/atlas-cashshop/.../cashshop/processor_inventoryincrease_test.go`,
  `services/atlas-cashshop/docs/domain.md`.
- `59916a651` (Defect E): `services/atlas-channel/.../kafka/consumer/cashshop/consumer.go`,
  `services/atlas-channel/.../kafka/consumer/cashshop/consumer_test.go`.

This matches the `## Fix` entries for D and E exactly. No code outside
these two defects' prescribed fix locations was touched.

## Defect D — by-type inventory-slot purchase grants 4, not 8

**PASS.**

- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:357` —
  the hard-coded amount argument to `PurchaseInventoryIncrease` changed from
  `8` to `4`; the `4000` cost literal is untouched (confirmed by diff and by
  `grep -n "4000" processor.go` showing only the one, unchanged-value line).
  This now matches `PurchaseInventoryIncreaseByItemAndEmit`'s `4`.
- Regression test `TestPurchaseInventoryIncreaseByTypeGrantsFourSlots` added
  to `processor_inventoryincrease_test.go` asserts wallet debit of 4000,
  `InventoryCapacityIncreasedBody.Amount == 4`, and `Capacity == 4`. Ran it
  directly: `go test ./cashshop/... -run TestPurchaseInventoryIncreaseByType -v`
  → `PASS`.
- `services/atlas-cashshop/docs/domain.md` — both by-type grant-size mentions
  (the `Purchase*` bullet list and the `Processor` method list) updated from
  "8 slots" to "4 slots".
- Swept for stragglers: `grep -rn "8 slots\|grants 8\|amount 8\|, 8)"
  services/atlas-cashshop/ --include="*.go" --include="*.md"` (excluding
  `_test.go`) returns nothing. The report additionally checked
  `services/atlas-cashshop/docs/kafka.md`'s generic `"amount": 8` JSON
  examples and correctly judged them out of the brief's literal "8 slots"
  grep target (they are illustrative wire examples for the command/status
  body shapes, not a stated by-type grant size) — a defensible, non-blocking
  call, not an omission.
- `go build ./...` and the targeted test both succeed in the
  `atlas-cashshop` module.

## Defect E — buy-for-self vs gift discriminator on PACKAGE_PURCHASED

**PASS**, verified by hand across the full producer → topic → consumer seam.

### Producer never emits a zero RecipientCharacterId on the status event

`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_package.go:139-150`:

```go
effectiveRecipientCharacterId := characterId
if recipientCharacterId != 0 {
    recipient, rErr := p.chaP.GetById()(recipientCharacterId)
    ...
    effectiveRecipientCharacterId = recipientCharacterId
}
```

`effectiveRecipientCharacterId` is seeded from `characterId` (the buyer, never
zero — it is the character issuing the command) and only ever overwritten
with a *resolved, non-zero* `recipientCharacterId` on the gift path (any
zero recipient never reaches the resolution branch). Line 226 passes
`effectiveRecipientCharacterId` straight into
`PackagePurchasedStatusEventProvider` as the emitted
`RecipientCharacterId`. There is no other write path to that field before
emission. Confirmed: the field is genuinely never zero on this event, so
`RecipientCharacterId != e.CharacterId` (equal on buy-for-self, since both
resolve to the buyer's own id) is the correct discriminator — not merely
differently-wrong from the old `!= 0` check.

### No other consumer/handler still applies the command's zero-means-self convention to a status body

Swept (not spot-checked):

- `grep -rn "RecipientCharacterId" services/atlas-channel/... --include="*.go"`
  (excluding tests) returns 13 hits. Classified all of them:
  - `cashshop/producer.go:180,197` — building the *command* body from a
    resolved id; not a discriminator.
  - `socket/handler/cash_shop_package.go:61,94` and
    `cashshop/processor.go:257` — doc comments describing the *command's*
    own `== 0` convention (`RequestPackagePurchaseCommandBody`), which is
    legitimate: that's the command the channel itself constructs, and the
    zero convention is correct there.
  - `kafka/message/cashshop/kafka.go:131,138,149,342,353,362-363,372` —
    struct fields and doc comments on the command/status body types
    themselves, not executable discriminators.
  - `kafka/consumer/cashshop/consumer.go:389-394,424` — the fixed
    `handleStatusEventPackagePurchased`, the only executable status-body
    discriminator in the file.
- Enumerated every status body with a recipient-shaped field:
  `PackagePurchasedBody` (fixed) and `GiftPurchasedBody`
  (`kafka/message/cashshop/kafka.go:347-354`). Read
  `handleStatusEventGiftPurchased`
  (`kafka/consumer/cashshop/consumer.go:371-384`): it does not branch on
  `RecipientCharacterId` at all — that event only ever represents a gift, so
  there is no self/gift discriminator to get wrong there. `RingPurchasedBody`
  has no recipient-identity field (it reports only the buyer's own half).
- Checked atlas-cashshop's own consumer
  (`services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go:218-234`)
  for completeness: its `RecipientCharacterId` reads are on the *command*
  body (`c.Body.RecipientCharacterId`), consumed by
  `GiftAndEmit`/`PurchasePackageAndEmit`; unrelated to the status-body seam
  and correctly using the command's own zero convention.

No other handler anywhere in the repo copies the mistaken convention onto a
status body.

### Consumer fix and doc comment

`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go:424`
changed `e.Body.RecipientCharacterId != 0` to
`e.Body.RecipientCharacterId != e.CharacterId`. The doc comment above the
function (lines 385-406) was rewritten to state the status body's actual
convention and cite `kafka/message/cashshop/kafka.go:372-375`, matching the
`## Fix` entry's instruction precisely.

### Test asserts the new contract

`TestPackagePurchasedBuyForSelfProjectsAssetsIntoBuyPackageDone` now sets
`RecipientCharacterId: testCharacterId` (equal to `e.CharacterId`, matching
what the producer actually emits per the trace above) and asserts the
announced writer decodes as `BuyPackageDone` with items projected from
`AssetIds`. `TestPackagePurchasedGiftAnnouncesGiftPackageDone` keeps
`RecipientCharacterId: 99` (≠ `testCharacterId`), asserting the gift arm
(`GiftPackageDone`, no asset lookup). Ran both directly:
`go test ./kafka/consumer/cashshop/... -run TestPackagePurchased -v` →
both `PASS`. (Per the task instruction, I did not re-derive that these fail
against the old `!= 0` code — already confirmed by the requester.) The
fixture choice (`RecipientCharacterId == CharacterId` for self,
`RecipientCharacterId` a distinct id for gift) is exactly the producer's
real wire shape, not an artificial one — this is a genuine seam-honest test,
not a test that merely encodes the same bug the code has.

## Not evaluable

None. Both defects' fix surfaces were fully within reach of the diff plus
the producer/consumer files the diff's correctness depends on.

## Build/test evidence

- `cd services/atlas-cashshop/atlas.com/cashshop && go build ./...` — clean.
- `go test ./cashshop/... -run TestPurchaseInventoryIncreaseByType -v` — PASS.
- `cd services/atlas-channel/atlas.com/channel && go build ./...` — clean.
- `go test ./kafka/consumer/cashshop/... -v -run TestPackagePurchased` —
  both new/updated tests PASS.
- `git status --short` — clean worktree, no stray changes.

## Verdict

Both defects' fixes match their `## Fix` prescriptions, are correct at the
seam (producer trace confirms the new discriminator is correct rather than
differently-wrong), and are backed by tests that assert the new contract
using fixtures that mirror the producer's real wire shape.
