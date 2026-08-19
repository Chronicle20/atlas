# Review: Task 12 — REBATE_LOCKER_ITEM, `atlas-channel` side

Commit range: `9f044e1f9..a6948029b` (single commit, "feat(channel): implement REBATE_LOCKER_ITEM end to end")
Brief: `.superpowers/sdd/plan/task-12-brief.md` (Controller corrections C1-C5 authoritative)
Implementer report: `.superpowers/sdd/plan/task-12-report.md`

## Scope

`git diff --stat 9f044e1f9..a6948029b`:

```
 .../atlas.com/channel/cashshop/processor.go        | 11 ++++
 .../atlas.com/channel/cashshop/producer.go         | 14 ++++
 .../channel/kafka/consumer/cashshop/consumer.go    | 26 ++++++++
 .../channel/kafka/message/cashshop/kafka.go        | 28 ++++++++
 .../channel/socket/handler/cash_shop_credential.go |  2 --
 .../channel/socket/handler/cash_shop_operation.go  | 16 ++++-
 .../socket/handler/cash_shop_rebate_test.go        | 76 ++++++++++++++++++++++
 7 files changed, 170 insertions(+), 3 deletions(-)
```

Matches the brief's `### Files` list plus one out-of-list edit
(`cash_shop_credential.go`), addressed under Finding 1 below. All files in
scope reviewed; no files outside this diff were needed to evaluate
correctness (Task 4's `verifySecondaryCredential` contract was read to
confirm the diff there, per instructions).

## C1 — four-field `LockerRebatedBody`, byte-identical mirror

`services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go:249-255`:

```go
type LockerRebatedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CashId        int64     `json:"cashId"`
	Amount        int32     `json:"amount"`
	Currency      uint32    `json:"currency"`
}
```

Compared field-by-field, in order, against atlas-cashshop's source
(`services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go:245-250`)
— identical field names, types, order, and json tags. **PASS.**

`RequestLockerRebateCommandBody` (the command going the other direction) was
also compared and is field-identical between the two sides
(`TransactionId uuid.UUID`, `AccountId uint32`, `CashId int64`), as are the
string constants `CommandTypeRequestLockerRebate = "REQUEST_LOCKER_REBATE"`
and `StatusEventTypeLockerRebated = "LOCKER_REBATED"` on both sides. **PASS.**

## C2 — no branch on `Currency`; mirror-and-ignore

`handleStatusEventLockerRebated`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go:320-341`)
reads only `e.Body.CashId` and `e.Body.Amount` when building
`cashcb.CashShopRebateDoneBody(e.Body.CashId, e.Body.Amount)`. `e.Body.Currency`
is never read anywhere in the diff — no switch, no map, no lookup keyed on it.
The doc comment on the handler explicitly states why (`CashShopRebateDoneBody`
takes only sn/amount). Confirmed against the encoder itself
(`libs/atlas-packet/cash/clientbound/shop_operation_body.go:600-604`,
`CashShopRebateDoneBody(sn int64, amount int32)`). **PASS** — mirror-and-ignore
as directed, currency `3` is inert through this consumer exactly as required.

## C3 — real constant identifiers and byte-exact test

Verified against `libs/atlas-packet/cash/clientbound/shop_operation_body.go`:
- `CashShopOperationRebateDone = "REBATE_SUCCESS"` (line 65)
- `CashShopOperationRebateFailed = "REBATE_FAILED"` (line 38)
- `CashShopRebateFailedBody(message string)` (lines 381-388) resolves both
  `operations` (mode) and `errors` (error code) — the test's `options` map
  populates both (`cash_shop_rebate_test.go:56-63`). **PASS.**

Verified the production template mapping matches what the test hardcodes
(`services/atlas-configurations/seed-data/templates/template_gms_95_1.json:4848-4849`):
```
"REBATE_SUCCESS": 150,
"REBATE_FAILED": 151,
```
identical to the mode bytes asserted in the test. **PASS** — the test's
hardcoded option maps are not arbitrary; they match the real wire configuration.

Verified `RebateDone.Encode`
(`libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go:552-559`):
`WriteByte(mode)` + `WriteInt64(sn)` + `WriteInt32(amount)`. Independently
computed the expected bytes for `(sn=900001, amount=1200)`:
```
$ python3 -c "import struct; print(struct.pack('<q', 900001).hex()); print(struct.pack('<i', 1200).hex())"
a1bb0d0000000000
b0040000
```
Matches the test's `want` slice byte-for-byte
(`cash_shop_rebate_test.go:34-38`: `0xa1, 0xbb, 0xd, 0, 0, 0, 0, 0, 0xb0, 0x4, 0, 0`).
**PASS** — the test is not inferred from Go parameter widths, it is
hand-verified against the encoder.

## C4 — no Kafka command on the credential-mismatch path

`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go:172-181`:

```go
if cErr := verifySecondaryCredential(l, ctx)(s, sp.SPW(), sp.Birthday()); cErr != nil {
	if errors.Is(cErr, ErrCredentialMismatch) {
		if aErr := session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopRebateFailedBody("INVALID_BIRTHDAY"))(s); aErr != nil {
			l.WithError(aErr).Errorf(...)
		}
		return
	}
	l.WithError(cErr).Errorf(...)
	return
}
cashId := int64(sp.Unk())
transactionId := uuid.New()
if err = cashshop.NewProcessor(l, ctx).RequestLockerRebate(...); err != nil {
```

Both error branches (`ErrCredentialMismatch` and any other error from
`verifySecondaryCredential`, e.g. account-lookup failure) `return` before
reaching `RequestLockerRebate`. `RequestLockerRebate` is only called after
falling through the entire `if cErr != nil` block. **PASS** — no Kafka
command is emitted on the credential-mismatch path, or on any other
credential-verification error path.

`verifySecondaryCredential` itself is untouched functionally (see Finding 1).

## C5 — 150/151 pinned such that a swap fails the suite

`TestCashShopRebateDoneBodyEncodes` builds its `options` map with only
`cashcb.CashShopOperationRebateDone: float64(150)` and asserts `body[0] == 150`
via both `bytes.Equal` against the full expected slice and an explicit
`body[0] != 150` check. `TestCashShopRebateFailedBodyMode` builds its
`options` map with only `cashcb.CashShopOperationRebateFailed: float64(151)`
and asserts `body[0] == 151` **and** explicitly asserts `body[0] != 150`
(`cash_shop_rebate_test.go:73-75`, with a comment naming the failure mode
this guards against: "failure and success arms must not share a mode byte").

Reasoned through the swap scenario: if a future change accidentally paired
`CashShopOperationRebateDone` and `CashShopOperationRebateFailed`'s resolved
codes (e.g. wired 151 into the success test's map or vice versa), the
`body[0] != 150` / `body[0] == 151` assertions in the failure test and the
`bytes.Equal`/`body[0] != 150` assertions in the success test would each
independently fail — the two tests use disjoint hardcoded option maps, so
there is no shared fixture that could silently absorb a swap. **PASS** — this
is the C5 discrimination the controller asked for, verified by reasoning
through the failure mode rather than assumed from the test's existence.

## Finding 1 — unbriefed edit to `cash_shop_credential.go` (Task 4's file)

The diff removes two lines from
`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_credential.go:35-36`:

```diff
-//
-//nolint:unused // First consumers land in plan Tasks 12/14/17/20 (gift, ring, rebate arms); remove this directive when Task 12 wires the first caller.
 func verifySecondaryCredential(l logrus.FieldLogger, ctx context.Context) func(s session.Model, spw string, birthday uint32) error {
```

This is not in Task 12's `### Files` list, but the removed comment's own text
names Task 12 as the trigger for its removal ("remove this directive when
Task 12 wires the first caller"), and Task 12 does exactly that — this is the
first caller of `verifySecondaryCredential`
(`cash_shop_operation.go:172`). The edit is a documentation-only removal of a
stale `//nolint:unused` directive; the function body of
`verifySecondaryCredential` (lines 38+, the actual credential-check logic
reviewed under Task 4) is **byte-for-byte unchanged** — confirmed via
`git diff` showing only the 2-line comment removal, no other hunk in this
file. **Non-blocking** — legitimate and anticipated, no regression to Task
4's behaviour or tests.

## Build and test verification (run directly, not trusted from report)

```
$ cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
Build: clean, no output.
All packages report `ok`, including `atlas-channel/socket/handler`.

```
$ go test ./socket/handler/... -run TestCashShopRebate -v
=== RUN   TestCashShopRebateDoneBodyEncodes
=== RUN   TestCashShopRebateDoneBodyEncodes/refund
--- PASS: TestCashShopRebateDoneBodyEncodes (0.00s)
    --- PASS: TestCashShopRebateDoneBodyEncodes/refund (0.00s)
=== RUN   TestCashShopRebateFailedBodyMode
--- PASS: TestCashShopRebateFailedBodyMode (0.00s)
PASS
```

## Other checks

- **Consumer registration**: `handleStatusEventLockerRebated` is registered
  in `InitHandlers` on `EnvEventTopicStatus`
  (`kafka/consumer/cashshop/consumer.go:102-106`), consistent with the other
  seven status-event handlers registered on the same topic. Follows the
  `handleStatusEventSurpriseOpened` pattern cited in the brief (type check +
  tenant check + `IfPresentByCharacterId` announce). **PASS.**
- **`RequestLockerRebate` signature**: matches the brief exactly —
  `RequestLockerRebate(accountId uint32, characterId uint32, cashId int64, transactionId uuid.UUID) error`
  (`cashshop/processor.go:29`, `:233`). Producer emits on
  `cashshop.EnvCommandTopic` via `CreateKey(int(characterId))`
  (`cashshop/producer.go:157-170`), consistent with `OpenSurpriseCommandProvider`'s
  shape as directed. **PASS.**
- **`cashId` naming**: `cashId := int64(sp.Unk())` at
  `cash_shop_operation.go:182`, matching the brief's explicit instruction to
  name the local variable `cashId`, not `unk`. **PASS.**
- **Idempotency**: `transactionId := uuid.New()` is minted per-click in the
  handler (not by the processor), matching the brief's Step 4 instruction and
  documented in the processor's doc comment (`processor.go:225-229`).
  **PASS.**

## Verdict rationale

All five controller corrections (C1-C5) are independently verified against
source, not taken on the fix's premise. The one edit outside the brief's file
list is legitimate, self-documenting (the removed comment named this exact
task as its trigger), and functionally inert. Build and tests pass under a
fresh run. No blocking defects found.
