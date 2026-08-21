# Review: Duey dialog and send-confirmation (5b99d25a4..d9adf914d)

Reviewed against `docs/tasks/task-241-duey-parcel-delivery/bug-duey-dialog-and-send-confirmation.md`
and `design.md` §5.2/§5.3/§4.3, on branch `task-241-duey-parcel-delivery`, worktree
`.worktrees/task-241-duey-parcel-delivery`.

Range: `5b99d25a4..d9adf914d` — three substantive commits plus the diagnosis doc commit
(`d9adf914d`, docs-only, not separately reviewed beyond confirming it matches the code).

## Commit 1 — 5b99d25a4 (Duey dialog leading bool)

**Claim**: `PARCEL[OPEN]`'s leading bool is `CParcelDlg::m_nMode`, not a quick-delivery
flag; renamed `quickEnabled`→`receiveOnly`, channel now sends `false` unconditionally,
per-tenant gate deleted.

- `libs/atlas-packet/parcel/clientbound/parcel.go:15-64` — field/accessor/constructor
  renamed `quickEnabled`→`receiveOnly` end to end; `Encode` still calls
  `w.WriteBool(m.receiveOnly)` at the same wire position — confirmed no byte-layout
  change (PASS).
- `libs/atlas-packet/parcel/clientbound/parcel_body.go:39-44` —
  `ParcelOpenBody` parameter renamed to match; call sites pass a literal bool, so the
  rename is purely cosmetic there (PASS).
- Verified via `grep -rn "QuickEnabled\|quickDeliveryEnabled\|quickEnabled" --include="*.go"`
  that no remaining Go code references the deleted `quickDeliveryEnabled(t)` helper or
  the old `QuickEnabled()` accessor — the helper and its call site were fully removed
  together (PASS). One stray comment remains in `tools/packet-audit/cmd/run.go:1345`
  (`bool quickEnabled`) — unrelated file, not touched by this diff, cosmetic only; see
  Non-blocking below.
- `services/atlas-channel/.../kafka/consumer/parcel/consumer.go:158` —
  `ParcelOpenBody(false, mailbox, arrived)`, unconditional, matches the brief's fix
  exactly. `showParcel`'s signature dropped the now-unused `t tenant.Model` parameter;
  `tenant` package is still imported and used elsewhere in the file
  (`t := tenant.MustFromContext(ctx)` at the command-guard site), so no dead import
  (PASS).
- Every fixture call site (`libs/atlas-packet/parcel/clientbound/v72_test.go` through
  `v185_test.go`, `parcel_test.go`) passes a bool literal to `NewParcelOpen`, unaffected
  by the parameter rename — the per-version wire fixtures needed no changes, consistent
  with "no wire byte changed" (PASS, swept all 20 call sites via grep).
- `consumer_test.go` — both OPEN subtests (`newOldGmsTenant`, `newJmsTenant` — the JMS
  v185 column that used to take the "quick enabled" branch of the deleted gate) now
  assert `receiveOnly == false`, which is the load-bearing assertion: it would have
  failed under the pre-fix code for the JMS v185 tenant (PASS — this is a real
  regression test, not one that passes either way).
- `design.md` §5.2/§5.3 updated in the same commit with the IDA evidence (mode table,
  addresses) backing the rename — documentation and code move together (PASS).

No defects found in commit 1.

## Commit 2 — 94cb58a6f (PARCEL_SENT status event, cross-service seam)

**Claim**: `accept_to_parcel` is `parcel_send`'s last step; `handleAcceptToParcel` now
emits `PARCEL_SENT` addressed to the sender; atlas-channel's new
`handleParcelSentEvent` announces `PARCEL[SUCCESSFULLY_SENT]`.

Saga step order (design.md §4.3, quoted): `award_mesos` → optional `destroy_asset`
(ticket) → `transfer_to_parcel` (composite → `release_from_character` +
`accept_to_parcel`). `accept_to_parcel` is confirmed the last leaf action in the last
top-level step — the notice cannot fire before the saga's other steps run (PASS).

Producer → consumer trace:

- `services/atlas-parcel/.../kafka/consumer/custody/consumer.go:129-146` —
  after the existing `custody.EnvStatusTopic` ack, emits
  `parcelproducer.ParcelSentStatusEventProvider(b.CharacterId)` on
  `parcelmsg.EnvStatusEventTopic`, inside the same `buffer.Emit` transaction as the
  `AcceptCustody` DB write (PASS — consistent with the existing `PARCEL_ARRIVED`
  producer's transactional posture).
- Verified `b.CharacterId` (== `AcceptToParcelCommandBody.CharacterId`) is the
  **sender**, not the recipient: traced `TransferToParcelPayload.CharacterId` through
  `saga/processor.go:2175-2264` into `AcceptToParcelPayload.CharacterId`, and
  `services/atlas-parcel/.../parcel/processor_custody.go:88`
  (`SetSenderId(params.CharacterId)`) — `AcceptCustody` writes `CharacterId` into
  `SenderId`, confirming the field really is the sender's id, not the recipient's
  (`RecipientId` is a separate field throughout) (PASS).
- Envelope agreement, field-for-field, across the two Go modules:
  `services/atlas-parcel/.../kafka/message/parcel/kafka.go:28-32` and
  `services/atlas-channel/.../kafka/message/parcel/kafka.go` both define
  `StatusEvent[E]{CharacterId uint32 \`json:"characterId"\`; Type string
  \`json:"type"\`; Body E \`json:"body"\`}` and both `StatusEventParcelSentBody`
  are empty structs — identical JSON shape on both sides (PASS).
- Handler registration: `services/atlas-channel/.../kafka/consumer/parcel/consumer.go:59-64`
  registers `handleParcelSentEvent` via `rf(t, ...)` reusing the same `t` resolved for
  `parcelmsg.EnvStatusEventTopic` (the status topic, not the command topic) — same
  topic as the existing `handleParcelArrivedEvent` registration, following the
  established multi-handler-per-topic pattern (PASS).
- `handleParcelSentEvent` (consumer.go:201-231) guards on event type, tenant, then
  `IfPresentByCharacterId` — same posture as `handleParcelArrivedEvent` (PASS).
- Test honesty: `services/atlas-channel/.../kafka/consumer/parcel/sent_test.go` —
  `TestParcelSentEvent/online sender` asserts the encoded body is exactly `0x12`
  (`SUCCESSFULLY_SENT`) from a real `handleParcelSentEvent` call; this genuinely fails
  without the new handler (there was no caller of `ParcelOpenSuccessfullySentBody`
  before this commit, per the brief). `offline sender`/`wrong tenant`/`wrong event
  type` all assert zero announces — a real negative-path pin (PASS).
- `services/atlas-parcel/.../kafka/consumer/custody/consumer_test.go:155-163` — asserts
  exactly one `PARCEL_SENT` event with `characterId == 200` (the sender in the test
  fixture), explicitly distinguished from `RecipientId` (100) in the accompanying
  comment — this is the assertion that would catch a sender/recipient mix-up (PASS).
- Accepted trade-off (replayed `accept_to_parcel` re-emits the notice): the row create
  in `AcceptCustody` (processor_custody.go:80-83) short-circuits to `return nil` on an
  existing row *before* re-running the builder, but `handleAcceptToParcel` still falls
  through to both `mb.Put` calls unconditionally whenever `aerr == nil` — so a replayed
  delivery really does re-emit both the custody ack and `PARCEL_SENT`, exactly as the
  commit message documents. This is a narrow, stated, and low-consequence trade-off (a
  spurious client notice, not data corruption) — acceptable as documented, not a defect
  in itself.

No defects found in commit 2. This is a well-traced cross-service seam with tests on
both sides of the boundary that pin the new contract.

## Commit 3 — 1610d359c (sparse overlay regeneration)

**Claim**: `overlays/pr-sparse`'s four atlas-parcel topic vars and `DB_NAME` were
regenerated from `gen-topic-config.sh` / `gen-db-name-suffix.sh`, not hand-edited.

- Re-ran `gen-topic-config.sh PLACEHOLDER_BASELINE_ENVIRONMENT`'s logic and diffed all
  174 `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` lines against the committed
  `deploy/k8s/overlays/pr-sparse/kustomization.yaml` — **byte-for-byte identical**
  (`diff` exit 0, 174/174 lines) (PASS).
- Reconstructed `gen-db-name-suffix.sh`'s generation loop (inline, to avoid
  overwriting the live file in place) and diffed the full output against the committed
  `deploy/k8s/overlays/pr-sparse/patches/db-name-suffix.yaml` — **byte-for-byte
  identical** (`diff` exit 0) (PASS). This confirms the new `atlas-parcel` Deployment
  patch block (name/container `parcel`/`DB_NAME=atlas-parcel-PLACEHOLDER_BASELINE_ENVIRONMENT`)
  is exactly what the generator emits — not hand-typed.
- `overlays/pr` (the isolated, non-sparse overlay) already carried atlas-parcel's
  topic vars and `DB_NAME=atlas-parcel-PLACEHOLDER_ATLAS_ENV` entries before this
  branch (`deploy/k8s/overlays/pr/kustomization.yaml:353`,
  `deploy/k8s/overlays/pr/patches/db-name-suffix.yaml:320-328`) — this commit does not
  touch `overlays/pr` and does not need to; the pre-existing entries there are
  consistent with the newly-added `overlays/pr-sparse` entries (same var names, same
  generator, different suffix token) (PASS — no inconsistency introduced).

No defects found in commit 3.

## Not evaluable

- The claim "guard now reports 174/174 topics and 37/37 DB_NAMEs" (from the brief) was
  not re-run as a live guard invocation — `tools/verify.sh` already passed on
  `d9adf914d` per the task instructions, and re-running it was explicitly out of scope
  for this review. The byte-for-byte regeneration diff above is strong independent
  evidence for the topic/DB_NAME counts, but the guard's own pass/fail logic was not
  exercised here.
- Live-client behavior (does the real GMS/JMS client actually render three tabs and
  actually re-enable the send button on `0x12`) is IDA-evidence-only, as it is for the
  rest of this task; no live client capture was available in this review's scope.

## Summary

All three commits do what the brief and design.md describe. Commit 1's rename is
complete and wire-neutral, with regression tests that would have failed under the old
per-tenant gate. Commit 2's cross-service seam was traced by hand from the saga step
order through the payload chain to the DB write, and both sides of the Kafka contract
carry a test that pins the new behavior (not one that passes either way). Commit 3's
overlay regeneration is verified byte-for-byte against the generator scripts, and the
non-sparse overlay was confirmed not left inconsistent.

No blocking findings.

One non-blocking cosmetic note:

- `tools/packet-audit/cmd/run.go:1345` — the comment `bool quickEnabled + mailbox
  list…` still uses the old field name after commit 1's rename. Harmless (it's a
  comment in an unrelated audit tool, not touched by this diff), but worth a follow-up
  touch-up for consistency with `parcel.go`'s renamed doc comment.
