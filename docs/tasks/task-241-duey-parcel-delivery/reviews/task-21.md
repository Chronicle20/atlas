# Review: Task 21 — atlas-channel SHOW_PARCEL consumer

Commit range: `a6acc9f38..3e9d27635` (`976b00bae` substance, `3e9d27635` continuation).

## Scope confirmed

11 files changed, all in the reported scope: atlas-channel's new
`kafka/message/parcel`, `kafka/consumer/parcel` packages, `parcel/model.go`
(new `ToPacket`), `parcel/processor.go` + `parcel/requests.go` additions,
`main.go` wiring, plus an out-of-brief-Files-list-but-flagged addition to
atlas-parcel (`parcel/processor.go`, `parcel/resource.go`,
`parcel/resource_test.go`), and the one-line `env-configmap.yaml` change.
No stray edits. `go build ./...` and the relevant `go test` packages pass
in both modules (verified locally; `tools/verify.sh` was not run, per
instructions).

## Priority 1 — WireId reuse (highest-value check)

**PASS.** `parcel/model.go:26`:

```go
p := packetparcel.NewParcel(WireId(m.Id()), m.SenderName(), m.MesoAmount(), m.CreatedAt(), m.Message())
```

`WireId` is not re-declared; it is the same exported function defined at
`parcel/processor.go:135` (`func WireId(id uuid.UUID) uint32 { return
binary.BigEndian.Uint32(id[:4]) }`), moved there in Task 18. Grep across
`services/atlas-channel/atlas.com/channel` confirms exactly one definition
and three call sites: `parcel/model.go:26` (this task's emit),
`socket/handler/duey_action_receive.go:109` (`WireId(m.Id()) == wireId`,
Task 18's resolve-by-scan), and `duey_action_receive.go:257` (the DISCARD
removal announce). No second projection, no open-coded truncation anywhere
in the diff.

**Gap, reported per the review brief's instruction:** no test in this diff
(or in Task 18's, taken independently) exercises emit and resolve as a
round trip. `consumer_test.go`'s `openCounts` helper reads only the raw
`quickEnabled`/mailbox-count/arrived-count bytes (`consumer_test.go:73-90`)
— it never decodes a `PARCEL` entry's `parcelId` field and never calls
`WireId` at all (confirmed by grep: zero occurrences of `WireId` in the
test file). Because `WireId` is a single shared function, drift between
emit and resolve is structurally impossible today (both literally call the
same code), so this is not a live defect — but it is real absence of
regression protection: a future edit to either call site's assumption
(e.g. someone inlining the truncation at one site "for perf") would not be
caught by either task's suite. Recorded as `not_evaluable`/gap, not
blocking.

## Priority 2 — the `PATCH /parcels/{id}/notify` endpoint

**Judged warranted, not scope creep.**

The existing `PATCH /parcels/{parcelId}` route (Task 18) is not a general
update endpoint — it is discard-specific: registered with
`rest.RegisterInputHandler[DiscardRestModel]` (`resource.go:46,51`), and
`DiscardRestModel` (`rest.go:87-94`) carries a mandatory `recipientId` used
to gate ownership (`handleDiscardParcel`'s doc, `resource.go:195-197`).
mux cannot register a second handler on the same path+method with a
different input type, and reusing the discard route to also mean "stamp
notified" would conflate two different semantics (a status transition
gated by recipient identity vs. inter-service bookkeeping with no caller
identity check) behind one body shape. A second, narrower route is the
correct shape here, not scope creep.

It was also actually required by the brief's own Step 1 test table: the
`open with new arrivals` row explicitly requires "that parcel's
`LastNotified` is stamped afterwards" — and no existing write path reached
`StampNotified` (`administrator.go:64`, previously called from nowhere;
its own doc comment attributed it only to Task 24's future sweep). Without
a write path, that subtest row is unsatisfiable. This matches CLAUDE.md's
"finish producible work" instruction rather than deferring a genuine
blocker.

Checked as a REST surface in its own right:
- **Tenant scoping**: inherited automatically — `registerGet` wraps
  `handleNotifyParcel` through `server.ParseTenant` (`rest.go:74`, same
  path every other handler in this file uses), and `GetById`/`StampNotified`
  run against `p.db.WithContext(p.ctx)` where `ctx` carries the
  tenant-scoped context. Consistent with every other handler in the file.
- **Authorization**: deliberately not recipient-gated (`resource.go:236-238`'s
  doc comment: "not recipient-gated... atlas-channel's own bookkeeping,
  invisible to the player"). This is defensible — the caller is
  atlas-channel (an internal service), not a player-supplied action, the
  operation is non-destructive (idempotent stamp, no status transition,
  no data returned), and the id space is tenant-scoped. No meaningful
  privilege-escalation surface. Compare to Discard, which correctly does
  gate on recipientId because it drives a real state transition
  reachable from a player action.
- **Idempotency**: `StampNotified` is a plain `UPDATE ... SET last_notified =
  ?` (`administrator.go:64-72`) — re-stamping an already-notified row is a
  harmless no-op. Not explicitly tested, but the mechanism makes this
  low-risk; non-blocking.
- **JSON:API shape**: `204 No Content`, no request/response body
  (`resource.go:271`, `notifyRestModel` exists only to satisfy
  `requests.PatchRequest`'s generic constraint on the channel side,
  `requests.go:90-93`). Consistent with the "PATCH with no body" pattern
  already established by nothing else in this file, but is a reasonable,
  narrow choice for a stamp-only mutation.
- **Rejection paths tested**: `resource_test.go`'s new `notify missing`
  subtest asserts `404` for an unknown id (`resource_test.go:333-345`).
  There is no dedicated cross-tenant rejection subtest for `/notify`
  specifically (the file's existing `tenant isolation` subtest only
  exercises the `GET` list route), but the isolation mechanism is shared
  code path (`ParseTenant` + tenant-scoped `db.WithContext`) already proven
  elsewhere in the same file. Non-blocking gap, not a defect.

The report's revert-if-wrong instructions
(`.superpowers/sdd/plan/task-21-report.md:99-108`) are noted but not acted
on — this review's conclusion is that the endpoint should stand.

## Priority 3 — Step-1 test table (six subtests)

All six exist in `consumer_test.go` and assert what the table specifies:

| subtest | location | verified against table |
|---|---|---|
| `open with mailbox` | `consumer_test.go:160-195` | `quickEnabled=false` (uses a v61 GMS tenant, below the 72 floor), mailbox=2, arrived=0, no notify calls — matches |
| `open with new arrivals` | `consumer_test.go:197-227` | mailbox=2, arrived=1, and `notified == [newArrival.Id()]` — the stamp assertion is present, not just the count |
| `open quick` | `consumer_test.go:229-255` | one announce, `mailboxFetched` flag proves `getMailbox` is never called |
| `not yet receivable excluded` | `consumer_test.go:257-285` | `ReceivableAt` in the future, mailbox=0, comment cites FR-12 |
| `wrong tenant` | `consumer_test.go:287-300` | different tenant in context vs. server, `captured` stays empty |
| `recipient offline` | `consumer_test.go:302-314` | no session registered for characterId 999, `captured` stays empty, handler returns without error (no `t.Fatal` on an error path) |

Each subtest's expected values trace to either the brief's table directly
(counts, booleans) or to fixture data the test itself constructs
(`newArrival.Id()` is the fixture's own id, not read back off the
implementation) — not implementation-derived pinning.

## Priority 4 — `quickEnabled` derivation

**PASS.** `consumer.go:162-164`:

```go
func quickDeliveryEnabled(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(72)
}
```

Passed through at `consumer.go:149` (`parcelcb.ParcelOpenBody(quickDeliveryEnabled(t), mailbox, arrived)`),
not hard-coded. The `open with mailbox` subtest specifically uses a
below-floor GMS-61 tenant to prove `false` is reachable
(`newQuickDisabledTenant`, `consumer_test.go:49-56`), distinguishing it
from a stub that always returns one value.

Note (non-blocking): `plan.md:2322-2323` documents the full condition for
the sibling Task 22 gate as `t.IsRegion("GMS") && t.MajorAtLeast(72), plus
JMS 185`. This implementation only covers the GMS clause; the JMS clause is
absent. The consumer.go doc comment (`consumer.go:153-161`) explicitly
flags this as a known partial condition to keep in sync when Task 22 lands
("Task 22 does not exist yet as of this task, so the two can't share code
across packages; this is the condition to keep in sync"), and the brief
only asked to derive from "the same condition Task 22 uses" without
requiring the JMS clause be anticipated in a task that predates Task 22.
Not blocking, but flagged for whoever implements Task 22 to reconcile.

## Priority 5 — configmap line

**PASS.** `deploy/k8s/base/env-configmap.yaml:61`:

```yaml
COMMAND_TOPIC_PARCEL: "COMMAND_TOPIC_PARCEL"
```

Single line, identity-mapped value matching `COMMAND_TOPIC_PARCEL_CUSTODY`'s
shape exactly, alphabetically ordered ahead of its `_CUSTODY` sibling. No
other line in the file touched (`git diff --stat` shows `1 file changed, 1
insertion(+)`).

## Continuation commit (`3e9d27635`) verification

Lint was run and reported clean per the continuation report
(`tools/lint.sh --go services/atlas-channel/atlas.com/channel` → "0
issues."). Not independently re-run here per the "do not run verify.sh /
repo-wide gates" instruction, but this is a scoped, single-module lint
invocation the brief explicitly asked for and the report quotes verbatim
output for — accepted.

## Independent build/test verification (this review)

```
cd services/atlas-channel/atlas.com/channel && go build ./...        # clean
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/parcel/... ./parcel/...   # ok
cd services/atlas-parcel/atlas.com/parcel && go build ./...           # clean
cd services/atlas-parcel/atlas.com/parcel && go test ./parcel/... -run TestParcelResource   # ok
```

## Findings summary

No blocking findings.

Non-blocking notes:
1. No round-trip test asserts emit (`ToPacket`) and resolve (`duey_action_receive.go`'s scan) agree on a concrete uuid → uint32 → uuid path in one test. Currently safe only because both sides call the literal same `WireId` function; a future refactor that duplicates the truncation would go undetected by either task's suite.
2. `quickDeliveryEnabled` implements only the GMS clause of the condition `plan.md` documents for Task 22 (GMS MajorAtLeast(72) OR JMS 185); the JMS clause is explicitly flagged in-code as a known future sync point, not silently dropped.
3. No dedicated cross-tenant rejection subtest for `PATCH /parcels/{id}/notify` specifically (relies on the same tenant-scoping mechanism already tested via the file's other subtests).
4. `MarkNotified`/`notify` idempotency (re-stamping an already-notified parcel) is not explicitly tested; the underlying `UPDATE` is naturally idempotent so this is low risk.

None of the above block approval; they are visibility notes for a future task or a follow-up hardening pass.
