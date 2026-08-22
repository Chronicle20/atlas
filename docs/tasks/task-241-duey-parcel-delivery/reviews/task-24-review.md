# Task 24 review — atlas-parcel arrival notification sweep

Commit range: `d3e9bf3f2..a2093bc27` ("feat(parcel): arrival notification sweep")
Brief: `.superpowers/sdd/plan/task-24-brief.md`
Report: `.superpowers/sdd/plan/task-24-report.md`

## Scope

`git diff --stat d3e9bf3f2..a2093bc27` shows exactly the files the brief and
report claim: `kafka/message/parcel/kafka.go` (new), `kafka/producer/parcel/producer.go`
(new), `parcel/administrator.go` (+`ClaimNotifiable`), `parcel/notification_task.go`
(new), `parcel/notification_task_test.go` (new), `main.go` (wiring),
`go.mod` (`go mod tidy` promotions), `tools/scopeguard/callsite-allowlist.txt`
(+1 line), plus two unrelated pre-existing review/ledger docs from the prior
commit's own artifacts (`agent-ledger.tsv`, `reviews/task-22.md`,
`reviews/task-23.md` — these are Task 22/23's own review artifacts landing in
this diff range only because the range starts at `d3e9bf3f2`, not because
Task 24 touched them; confirmed via `git show --stat` they were part of the
prior commit, not this one — no scope mismatch). No file outside the brief's
list or its stated deviation was touched.

## 1. Tenant safety of the cross-tenant sweep — PASS

`notification_task.go:145-182` (`Run()`):

- `noTenantCtx := database.WithoutTenantFilter(t.ctx)` (line 151) is used only
  to build `tdb` for the `ClaimNotifiable` claim query (line 153-155). No
  Kafka emit or write is issued against `noTenantCtx` or `tdb`.
- For each claimed row (line 165), the tenant is reconstructed from that row's
  own `TenantId()` via `tenant.Create` (line 166) — not from `t.ctx` or any
  shared value.
- The reconstructed tenant is re-entered via `tenant.WithContext(t.ctx, tm)`
  — built from the task's own `t.ctx`, NOT `noTenantCtx` (line 171), matching
  the load-bearing distinction `task.go:169-174` documents (bypassing the
  tenant filter also disables the tenant-injection callback, so a write context
  built from `noTenantCtx` would carry a zero tenant id). `envContext` is
  layered on top of the re-entered tenant context on the same line, matching
  the brief's stated ordering.
- The Kafka emit (`producer.ProviderImpl(t.l)(tenantCtx)` then `kp(...)`,
  lines 173-174) uses only `tenantCtx`, built from the reconstructed
  per-row tenant. No path in `Run()` emits under `t.ctx`, `noTenantCtx`, or
  any other tenant's context.
- A `tenant.Create` failure for one row (`terr != nil`, line 167) is
  `continue`d — that row is skipped rather than falling through to a stale or
  default tenant context.

This matches `task.go`'s already-gated shape claim-for-claim. No wrong-tenant
or bare-task-context emit path found.

## 2. Scope-guard allowlist entry — CHANGES REQUIRED (see blocking finding)

`tools/scopeguard/callsite-allowlist.txt:37`:

```
atlas-parcel/parcel/notification_task.go:151 # NotificationTask.Run — ... Each
claimed row's tenant is reconstructed from its own TenantId()
(notification_task.go:161) and re-entered via tenant.WithContext(t.ctx, tm),
with this pod's envContext layered on top (notification_task.go:171), before
the PARCEL_ARRIVED Kafka emit is issued through that tenant context
(notification_task.go:173), ...
```

Checked each cited line against the committed file:

| citation | claimed content | actual line 161 / 171 / 173 content |
|---|---|---|
| key `:151` | `WithoutTenantFilter` call | `noTenantCtx := database.WithoutTenantFilter(t.ctx)` — **correct** |
| `:161` | tenant reconstruction (`tenant.Create`) | `return` (inside `if len(claimed) == 0 { return }`, line 160-161) — **wrong**. The actual `tenant.Create` call is at **line 166**, five lines later. |
| `:171` | re-entry via `tenant.WithContext` | `tenantCtx := t.envContext(tenant.WithContext(t.ctx, tm))` — **correct** |
| `:173` | the Kafka emit | `kp := producer.ProviderImpl(t.l)(tenantCtx)` — the producer is constructed here; the actual send (`kp(parcelmsg.EnvStatusEventTopic)(...)`) is on the next line, **174**. Close enough to be locatable by a human auditor reading the surrounding lines; not a hard miss like `:161`. |

The allowlist entry's `:151` key (what the scope-guard tool itself parses and
matches against, per `tools/scopeguard/allowlist.go:59` keyed on `file:line`
of the call site) is correct, so the gate itself passes. But the brief was
explicit that the justification's *prose* citations must name "the concrete
`file:line` at which each claimed row's tenant is reconstructed and re-entered"
— and the `:161` citation points at an unrelated early-return statement, not
the reconstruction. A future auditor following that citation lands on the
wrong line. This is the exact kind of "plausible-sounding but wrong" citation
the controller asked to check for.

## 3. `os.LookupEnv` vs `topic.EnvProvider` deviation — VERIFIED, ACCEPTED

`libs/atlas-kafka/topic/provider.go:14-25`:

```go
func EnvProvider(l logrus.FieldLogger) func(token string) Provider {
	return func(token string) Provider {
		return func() (string, error) {
			t, ok := os.LookupEnv(token)
			if !ok {
				l.Warnf("[%s] environment variable not set. Defaulting to provided token.", token)
				return t, nil  // returns token, nil — never an error
			}
			return t, nil
		}
	}
}
```

Confirmed: `EnvProvider`'s returned `Provider` never returns a non-nil error
on a missing env var; it falls back to the literal token string and returns
`nil` error. The implementer's claim is accurate, and using it as an
"is-configured" gate would make `notification_task.go`'s `topic not
configured` subtest impossible to satisfy (it would never see an error to
key off of).

Cross-checked against frederick's own use of this exact pattern
(`services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go:53-56`):
`_, err = topic.EnvProvider(t.l)(merchant.EnvStatusEventTopic)(); if err != nil { ... skip }`.
Given the confirmed provider behavior, this branch in frederick is dead code —
it can never trigger, which means frederick's own "topic not configured" skip
does not actually work today. That is a latent defect in frederick, out of
this unit's scope to fix, but it means the implementer's chosen `os.LookupEnv`
deviation is the *only* one of the two patterns that actually works, and using
it here is the right call rather than a lesser copy of the "true" original.

Searched the repo for a third, sanctioned "topic optional" idiom
(`grep -rn "not configured"` across all services' non-test Go files): no other
service expresses this via anything other than `topic.EnvProvider`'s (broken)
error path or a bespoke `os.LookupEnv`/`os.Getenv` check. No pre-existing
sanctioned pattern was overlooked.

## 4. Claim-by-update correctness — PASS

`administrator.go:125-144` (`ClaimNotifiable`): a single UPDATE statement
(`db.Clauses(clause.Returning{}).Model(&entities).Where(...).Where("id IN (?)", candidates).Update("last_notified", now)`)
both selects and stamps `last_notified` in one statement, guarded by
`last_notified IS NULL AND status = 'pending' AND receivable_at <= ?` in both
the outer WHERE and the candidate subquery — the identical compare-and-swap
shape as `ClaimExpired` (`administrator.go:77-99`), which the brief and Task
23's already-gated review established as safe under `replicas: 2` with no
leader election (`deploy/k8s/base/atlas-parcel.yaml:9`). Whichever replica's
UPDATE commits first is the only one whose WHERE clause still matches the row
by the time it runs; the other affects zero rows. No SELECT-then-UPDATE
race window exists.

`concurrent claim` subtest (`notification_task_test.go:220-247`): the
subtest's own comment (lines 221-232) explicitly and correctly discloses that
this is a SEQUENTIAL verification against `ClaimNotifiable` directly — two
calls against the same in-memory sqlite handle — not a genuine two-replica
race, and states why (the harness serializes all access through one `*gorm.DB`
handle). This matches the controller's carried-forward instruction from Task
23 and is honestly labeled; the test name itself (`concurrent claim`) is
slightly optimistic on its own but the body's comment immediately corrects
the impression, same as Task 23's precedent.

## 5. Seven brief-specified subtests — PASS

`go test ./parcel/... -run TestNotificationSweep -v` re-run locally:

```
--- PASS: TestNotificationSweep (0.00s)
    --- PASS: TestNotificationSweep/notifies_a_newly_receivable_parcel
    --- PASS: TestNotificationSweep/does_not_renotify
    --- PASS: TestNotificationSweep/not_yet_receivable
    --- PASS: TestNotificationSweep/resolved_parcel
    --- PASS: TestNotificationSweep/offline_recipient_still_stamps
    --- PASS: TestNotificationSweep/topic_not_configured
    --- PASS: TestNotificationSweep/concurrent_claim
```

Checked each against the brief's table (`task-24-brief.md:55-63`):

- `notifies a newly receivable parcel` (`notification_task_test.go:116-136`):
  asserts one `PARCEL_ARRIVED` event, `SenderName` "Alice", `HasItem` true,
  and `LastNotified` stamped to `notifyClock` — matches.
- `does not renotify` (`:138-151`): pre-stamped `LastNotified`, asserts no
  event — matches.
- `not yet receivable` (`:153-168`): `ReceivableAt` in the future, asserts no
  event AND `LastNotified` still nil — matches (brief requires both).
- `resolved parcel` (`:170-183`): `StatusReceived`, asserts no event —
  matches.
- `offline recipient still stamps` (`:185-203`): no session concept exists in
  this service at all (correctly noted inline), asserts the event IS emitted
  and `LastNotified` IS stamped — matches the brief's FR-24 deferral to the
  channel-side OPEN packet.
- `topic not configured` (`:205-218`): uses `unsetEnv` helper (`:255-264`,
  correctly implemented — LookupEnv/Unsetenv/Cleanup-restore, not `t.Setenv`
  which cannot express "absent") to unset `EVENT_TOPIC_PARCEL_STATUS`, asserts
  **both** no event emitted AND `LastNotified` still nil — matches the brief's
  "stamp NOTHING" requirement exactly (the code achieves this by checking the
  topic before the claim, `notification_task.go:146-149`, so an unconfigured
  topic never reaches `ClaimNotifiable` at all).
- `concurrent claim` (`:220-247`): see §4 above — present, and honestly
  labeled.

All seven subtests exist, assert what the brief specifies, and pass.

## Build/test verification

```
$ cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...
ok  	atlas-parcel/kafka/consumer/custody	(cached)
ok  	atlas-parcel/parcel	0.087s
(all other packages: [no test files], clean build)
```

## Not evaluable

- None. The full review surface (the diff plus `task.go`/`task_test.go` as
  the cited reference shape, `libs/atlas-kafka/topic/provider.go` as the
  cited deviation justification, `deploy/k8s/base/atlas-parcel.yaml` for the
  replica count) was reachable and read.

## Findings summary

- **Blocking (1):** `tools/scopeguard/callsite-allowlist.txt:37` — the
  `notification_task.go:161` citation inside the justification prose points
  at an unrelated `return` statement, not the `tenant.Create` reconstruction
  it claims to cite (the real call is at `notification_task.go:166`). The
  scope-guard tool's own key match (`:151`) is unaffected and the gate still
  passes, but the brief explicitly required this citation to be accurate for
  a human auditor, and it is not.
- **Non-blocking (1):** same allowlist line's `:173` citation for "where the
  Kafka emit is issued" points at the producer-construction statement
  (`kp := producer.ProviderImpl(t.l)(tenantCtx)`); the actual send is the next
  line, `:174`. Adjacent and easily located by a reader, unlike the `:161`
  miss — worth a follow-up correction but not blocking.
