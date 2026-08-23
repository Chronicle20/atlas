# Review: Task 23 — atlas-parcel expiry and return-to-sender sweep

Commit range: `bc09d238b..b3981f49a` (`b4f2ea0c7`, `d9941beb4`, `b3981f49a`)
Briefs: `.superpowers/sdd/plan/task-23-brief.md` (+ CONTROLLER ADDENDUM /
RULING 12), `.superpowers/sdd/plan/task-23-brief-cont.md`
Implementer report: `.superpowers/sdd/plan/task-23-report.md`

## Scope confirmed

Reviewed the full diff: `libs/atlas-packet/parcel/parcel.go`,
`libs/atlas-saga/payloads.go`, atlas-channel's `parcel/{model,processor}.go`
+ tests, `socket/handler/duey_action_send{.go,_test.go}`, atlas-parcel's
`parcel/{administrator,builder,entity,model,rest,task}.go` + tests,
`kafka/{message,consumer}/custody/*`, `main.go`, atlas-saga-orchestrator's
`saga/{processor,handler}.go`, `parcel/{processor,producer}.go`,
`kafka/message/parcel/custody/kafka.go`, `saga/parcel_expansion_test.go`,
and `deploy/k8s/base/atlas-parcel.yaml`. This matches the report's file list
exactly — no scope drift found beyond what both implementers self-disclosed
(the orchestrator's second wire-contract mirror, explicitly flagged and
traced in the report and independently re-traced below).

Ran module-local `go build ./... && go test ./...` in all four touched
modules (`atlas-parcel`, `atlas-channel`, `atlas-saga-orchestrator`,
`atlas-saga`) as a sanity check, not as the verification gate. All green.

## 1. Ruling 12 wire fix — VERIFIED end to end

- `libs/atlas-packet/parcel/parcel.go`: struct field `sentAt`→`expiresAt`,
  accessor `SentAt()`→`ExpiresAt()`, `NewParcel` param renamed, `Encode`
  writes `p.expiresAt` at +21. Doc comment rewritten with the full
  derivation (v72 `0x65AF41`/v83 `0x6F0D11`, `__aulldiv`, `cmp eax,1Eh; jl`
  polarity), and correctly frames it as "not an eligibility window — a
  countdown to a future deadline."
- `services/atlas-channel/atlas.com/channel/parcel/model.go:26` —
  `ToPacket` now passes `m.ExpiresAt()`, not `m.CreatedAt()`, into
  `NewParcel`.
- `services/atlas-channel/atlas.com/channel/parcel/processor.go` —
  `RestModel.ExpiresAt` (`json:"expiresAt"`) added, `Model.expiresAt` +
  `ExpiresAt()` accessor added, `Extract` maps `rm.ExpiresAt` through. The
  stale `CreatedAt` doc comment claiming it was the wire's +21 field is
  fixed.
- `services/atlas-parcel/atlas.com/parcel/parcel/rest.go:42,83` — REST
  already emitted `expiresAt`; confirmed unchanged (no producer-side change
  needed, as the addendum predicted).
- `grep` across `services/atlas-channel/atlas.com/channel/parcel/*.go`
  confirms no remaining call site routes `CreatedAt` into `NewParcel`.
- Test: `services/atlas-channel/atlas.com/channel/parcel/model_test.go`
  `TestToPacketExpiresAt` builds a `RestModel` with distinct `CreatedAt`
  and `ExpiresAt`, asserts `ToPacket()`'s wire value equals `ExpiresAt` and
  is NOT `CreatedAt`. This is a genuine regression pin — it fails against
  the pre-fix code (`CreatedAt` routed in) and passes against the fix.

No partial fix found. The full chain (atlas-parcel REST → atlas-channel
`RestModel`/`Model` → `ToPacket` → wire struct) carries `ExpiresAt`, not
`CreatedAt`.

## 2. `ExpiryWindow` = 29 days — VERIFIED, test traces to real arithmetic

- `services/atlas-parcel/atlas.com/parcel/parcel/entity.go` —
  `ExpiryWindow = 29 * 24 * time.Hour`, doc comment gives the exact
  return-leg-zero-delay derivation from the addendum.
- `entity_test.go`'s `TestTimers` pin updated to `696*time.Hour` (29 days)
  with an explanatory comment, not left as a bare magic number.
- `TestReturnLegSurvivesClientExpiryGuard` (new) computes
  `quotient := uint64(expiresAt.Sub(now)) / uint64(24*time.Hour)` against
  `createdAt.Add(ExpiryWindow)` at the exact zero-delay instant a return leg
  becomes receivable, and asserts `quotient < 30` — the client's own
  predicate, written against the live `ExpiryWindow` constant. Manually
  confirmed: at 30 days the quotient is exactly 30 and `30 < 30` is false,
  so this test genuinely re-breaks if `ExpiryWindow` is raised back to 30.
  At 29 days the quotient is 29, `29 < 30` holds. Not hardcoded to 29 —
  traces to the constant.

## 3. `RecipientName` thread — traced end to end, VERIFIED complete

Traced the full path by hand, independent of the report's own claim:

`duey_action_send.go:263` (`buildParcelSendSaga`) sets
`TransferToParcelPayload.RecipientName = recipient.Name()` →
`libs/atlas-saga/payloads.go` carries the field on both
`TransferToParcelPayload` and `AcceptToParcelPayload` →
`saga-orchestrator/saga/processor.go`'s `expandTransferToParcel` copies
`payload.RecipientName` into **both** `AcceptToParcelPayload` literals (item
branch and meso-only branch — checked both) → `saga/handler.go`'s
`handleAcceptToParcel` copies `payload.RecipientName` into
`parcel.AcceptToParcelParams.RecipientName` (the orchestrator's OWN mirror
struct, confirmed as a genuinely separate type from
`saga/processor.go`'s `AcceptToParcelPayload`, per the report's claim) →
`saga-orchestrator/parcel/producer.go`'s `AcceptToParcelProvider` copies
`params.RecipientName` into the outbound
`kafka/message/parcel/custody/kafka.go`'s `AcceptToParcelCommandBody`
(`json:"recipientName"`) → atlas-parcel's own
`kafka/message/custody/kafka.go`'s `AcceptToParcelCommandBody` carries the
matching tag → `kafka/consumer/custody/consumer.go`'s
`handleAcceptToParcel` maps `b.RecipientName` into
`parcel.AcceptParams.RecipientName` → `processor_custody.go`'s
`AcceptCustody` calls `.SetRecipientName(params.RecipientName)` on the
builder before `Create`.

This confirms the report's claim that the orchestrator's second mirror
(`kafka/message/parcel/custody/kafka.go`, `parcel/processor.go`,
`parcel/producer.go`, `saga/handler.go`) was a real, necessary hop, not an
overcautious addition — omitting it would have type-checked but silently
dropped the value between `handleAcceptToParcel` and the Kafka message
atlas-parcel actually receives.

Tests at each seam pin the NEW contract, not just compilation:
- `duey_action_send_test.go`: `tp.RecipientName != f.recipientCandidates[0].Name()` —
  compares against a real seeded name ("Bob"), not an empty string; fails
  pre-fix.
- `parcel_expansion_test.go`: seeds `RecipientName: "Bob"` on the input,
  asserts `acc.RecipientName == "Bob"` on the expanded payload.
- `consumer_test.go`'s `TestCustodyCommands` "accept with item": seeds
  `RecipientName: "Bob"` on the custody command, asserts
  `m.RecipientName() == "Bob"` on the row read back after
  `handleAcceptToParcel` — this is the genuine end-to-end pin inside
  atlas-parcel's own module (command → `AcceptParams` → `Model` → DB read).

Not separately tested (and reasonably so, per the report's own note): the
orchestrator's `handler.go`→`producer.go` copy and the meso-only branch of
`expandTransferToParcel` — both are one-line copies of an already-tested
shape, not new plumbing patterns.

**Non-blocking finding**: `services/atlas-parcel/atlas.com/parcel/parcel/model.go`,
`RecipientName()`'s doc comment (added in the first commit, `b4f2ea0c7`)
still reads "NOT yet populated by the currently-landed atlas-channel send
saga; see entity.go's doc comment for the exact follow-up wiring this
needs." That is now stale — Unit B closed exactly that gap in the same
range, and `entity.go`'s and `task.go`'s parallel doc comments were
correctly updated to describe the completed wiring (confirmed by diff), but
`model.go`'s was missed. Misleading to a future reader who trusts the
accessor's own doc rather than the entity's; not a functional defect since
the field is genuinely populated end-to-end.

## 4. Concurrent-claim guarantee — honestly scoped, not oversold

`task_test.go`'s "concurrent claim" subtest carries an explicit comment
stating it is NOT a genuine race — `databasetest.NewInMemoryTenantDB` is a
single sqlite handle and cannot express two replicas' `UPDATE`s
overlapping — and that what it actually verifies is `ClaimExpired`'s
compare-and-swap property sequentially (second claim against an
already-claimed row affects zero rows). This matches exactly what the task
brief in this dispatch asked for: "a test that looks like it proves this
and does not is worse than an honest note saying the harness cannot express
it." The mechanism itself (`ClaimExpired`'s `WHERE status='pending'` in
both the outer UPDATE and the candidate subquery — `administrator.go:77-99`)
is the real correctness argument, and it is sound: a losing UPDATE's WHERE
clause stops matching a row the instant the winning UPDATE's
`status='expired'` write commits, which is exactly the row-level guard
NFR-7 relies on. `deploy/k8s/base/atlas-parcel.yaml` runs `replicas: 2`, so
this guarantee is operationally load-bearing, not theoretical.

## 5. Step 5 deploy wiring — VERIFIED correct

- `deploy/k8s/base/atlas-parcel.yaml` — `PARCEL_EXPIRY_INTERVAL_SECONDS`
  added as a plain per-deployment `env:` entry (`value: "3600"`) inside the
  existing block, after `DB_PASSWORD`. Confirmed by diff:
  `deploy/k8s/base/env-configmap.yaml` and both
  `deploy/k8s/overlays/{main,pr}/kustomization.yaml` are untouched in this
  range (`git diff --stat` shows zero hits) — correctly avoiding Ruling
  10's trap, matching the `EXPIRATION_CHECK_INTERVAL_SECONDS` precedent.
- `main.go` — `getExpirationInterval()` reads the env var, falls back to
  `parcel.DefaultExpiryInterval` on unset/invalid (non-positive parse also
  falls back, checked). `expiryTask.Start()` is called and
  `rt.TeardownFunc(expiryTask.Stop)` is registered, so the sweep stops
  cleanly on shutdown alongside the existing Kafka-producer teardown.

## Test-value provenance check

Walked every new numeric/string literal back to its source:
- `29`/`696h` — derived arithmetically from the client's `< 30` unsigned
  check and the return leg's zero-delay receivability (addendum's own
  derivation, restated correctly in both the entity.go doc comment and
  context.md §11).
- `T = 2026-03-01T00:00:00Z`, batch `2`/`5` in "batch bound",
  `SenderId 100`/`RecipientId 200`, `FeePaid 800`/`0`, `MesoAmount 5000`,
  item `1302000` — all trace directly to the brief's Step 2 table
  (`task-23-brief.md` lines 87-93); the test fixtures reproduce them
  exactly.
- `"Bob"` in the `RecipientName` seam tests — a real seeded fixture name
  (`character.Model` / `Entity`), not a placeholder that happens to satisfy
  an empty-string comparison; each assertion would fail against pre-fix
  code where `RecipientName` is never threaded.

No test value found to be reverse-engineered from the implementation to
make an assertion pass.

## Findings

### Blocking

None.

### Non-blocking

1. `services/atlas-parcel/atlas.com/parcel/parcel/model.go` —
   `RecipientName()`'s doc comment is stale post-Unit-B: it still says the
   field is "NOT yet populated by the currently-landed atlas-channel send
   saga," which was true when written in `b4f2ea0c7` but is no longer true
   after `b3981f49a` closed that gap. `entity.go`'s and `task.go`'s
   parallel comments were correctly updated; this one was missed. Fix is a
   one-comment edit.

### Not evaluable

None — the full range was within the four-module review surface and all
five focus areas were traceable to `file:line` evidence without needing to
survey code outside the diff.

## Verdict

APPROVED_WITH_FINDINGS — the Ruling 12 wire fix, the 29-day window, the
RecipientName cross-service thread, the concurrent-claim honesty, and the
Step 5 deploy wiring all check out against the brief and against manual
end-to-end tracing. The single finding is a stale doc comment with no
functional impact.
