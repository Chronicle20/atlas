# Review — Task 17: DUEY_ACTION handler and the ParcelSend saga

Commit range reviewed: `281ce4242..680a44ff0` (single commit `680a44ff0`,
"feat(channel): DUEY_ACTION handler and the parcel_send saga", 8 files, 944
insertions). `281ce4242` (Task 15) is excluded from scope per instructions.

Brief: `.superpowers/sdd/plan/task-17-brief.md`
Continuation brief: `.superpowers/sdd/plan/task-17-brief-cont.md`
Implementer report: `.superpowers/sdd/plan/task-17-report.md`

## Scope

Reviewed the full diff: `duey_action.go`, `duey_action_send.go`,
`duey_action_send_test.go`, `parcel/requests.go`, `parcel/processor.go`,
`saga/model.go` re-exports, `main.go` registration,
`libs/atlas-constants/item/duey.go`. Also read, as contract dependencies:
atlas-parcel's `parcel/resource.go` (the `GET /parcels` handler the new REST
client calls), atlas-saga's `TransferToParcelPayload`/`AcceptToParcelPayload`
structs, and atlas-saga-orchestrator's `expandTransferToParcel` (the Task 12
consumer of this task's saga payload). Confirmed build and the module's full
test suite are green (re-ran `go build ./...` and
`go test ./socket/handler/... -run TestDueyActionSend -v`; all 13 subtests
pass). Task 16's `Fee`/`TotalCost`/`ValidateSend` arithmetic was NOT
re-audited, per instructions — treated as settled.

`scope_confirmed`: the diff matches the brief's stated file list exactly (no
scope drift); the continuation brief's promised work (full suite + lint,
report) is also present and its claims verified independently rather than
taken on faith.

## Findings, in the reviewer brief's priority order

### 1. `CountPending` signature deviation — RULED: justified, brief was wrong

Verified directly against `services/atlas-parcel/atlas.com/parcel/parcel/resource.go:81-95`:
`filter[worldId]` is parsed with `q.Get("filter[worldId]")`, and a missing
value hits `server.WriteBadRequest(d.Logger(), w, "filter[worldId] is
required")` (`resource.go:83-84`) — a hard 400, not a default. The
implementer's `CountPending(recipientId uint32, worldId world.Id) (int,
error)` (`parcel/processor.go:158-172`) is correctly justified: the brief's
`CountPending(recipientId uint32)` signature could only be honored by
defaulting `worldId` internally, which is exactly the world-0-sentinel class
this branch has flagged as blocking elsewhere. The deviation is documented
inline in `processor.go` and in both the report and the continuation brief.
**No finding against the code; this is a finding against the brief itself,
already correctly self-identified.**

### 2. Config-resolved mode byte — PASS

`duey_action.go:24-34` mirrors `storage_operation.go:31-65` exactly: decode
`parcelsb.Action`, read `p.Mode()`, and dispatch through
`isDueyAction(l)(readerOptions, mode, DueyActionModeSend)`
(`duey_action.go:47-68`), which resolves the code from
`readerOptions["operations"][string(key)]` — never a literal. An unmatched
mode logs at `Warnf` and returns without action
(`duey_action.go:33`). `docs/packets/dispatchers/duey_action.yaml` is the
config source and predates this task (already committed). No hard-coded mode
byte anywhere in this diff.

### 3. The reject path — PASS

`sendParcel` (`duey_action_send.go:92-198`) covers every `ValidateSend`
reason (`RejectIncorrectRequest`, `RejectMesoLimit`, `RejectNotEnoughMesos`
— `duey_action_send.go:113-123`, matching `parcel/validation.go:10-13`
exactly) plus every remote-check reject (`NameDoesNotExist` for both unknown
name and wrong-world match, `SameAccount`, `ReceiverStorageFull`, the
quick-ticket check, and item-lookup failure) — each one calls `reject(...)`
via `session.Announce` inline (`duey_action_send.go:97-100`) and `return`s
immediately, never reaching `deps.createSaga`. `session.Announce`
(`session/processor.go:247-`) only writes to the socket; it never closes the
connection, matching `note_send.go`'s posture. No reject path silently drops
the packet (each maps to a specific `Parcel*Body` clientbound function, all
of which exist in `libs/atlas-packet/parcel/clientbound/parcel_body.go`).

### 4. World-0, tag-for-tag — PASS

- `parcel.RestModel.WorldId byte \`json:"worldId"\`` (`parcel/processor.go:24`)
  matches atlas-parcel's `parcel/rest.go:17` (`WorldId byte
  \`json:"worldId"\``) exactly — both sides use the same field name and tag.
- The REST client's query string
  (`parcel/requests.go:22,34`, `filter[worldId]=%d` fed `byte(worldId)`)
  matches the handler's expectation of a byte-parseable value
  (`resource.go:89-93`, `world.Id(byte(parsed))`).
- `saga.TransferToParcelPayload.WorldId world.Id \`json:"worldId"\`` (defined
  in `libs/atlas-saga/payloads.go:987-1005`, re-exported as a type alias in
  `saga/model.go:64`) is populated from `s.WorldId()`
  (`duey_action_send.go:242`) — the session's own world, not a hardcoded or
  defaulted value — and consumed unchanged by
  `expandTransferToParcel`/`AcceptToParcelPayload.WorldId`
  (`processor.go:2176,2246`).
- Recipient world-filtering is explicit and local
  (`duey_action_send.go:139-149`, `if c.WorldId() == s.WorldId()`), not
  delegated to a REST filter that could silently default.

No world-0 defaulting found anywhere in this diff.

### 5. Saga payload vs. orchestrator's `expandTransferToParcel` — PASS

Read `libs/atlas-saga/payloads.go:987-1005`
(`TransferToParcelPayload`) against
`atlas-saga-orchestrator/.../saga/processor.go:2162-2260`
(`expandTransferToParcel`). Every field the channel populates in
`buildParcelSendSaga` (`duey_action_send.go:240-260`) is read and forwarded
by the expansion: `TransactionId`, `ParcelId`, `CharacterId`, `WorldId`,
`SourceInventoryType`+`AssetId` (compartment lookup key, guarded behind
`AssetId != 0` — matches the channel leaving both zero-valued on a
meso-only send), `Quantity`, `SenderAccountId`, `SenderName`,
`RecipientId`, `RecipientAccountId`, `MesoAmount`, `FeePaid`, `Quick`,
`Message`, `ReceivableAt`, `ExpiresAt`. No field mismatch, no silent no-op
risk found on this seam.

### 6. Out-of-brief additions — PASS

- `saga/model.go:64,90,134` — `TransferToParcelPayload`, `ParcelSend`,
  `TransferToParcel` are all `= sharedsaga.X` type/const aliases, not
  redefinitions (matches the Task 14 pattern the brief calls out).
- `libs/atlas-constants/item/duey.go:17` — `QuickDeliveryTicketId =
  uint32(5330000)`. Grepped the whole `libs/atlas-constants` tree for
  `5330000`/`QuickDeliveryTicketId`: no prior definition exists, so this is
  not a duplicate. The brief explicitly authorized defining it here if Task
  22 had not landed, and the file's own comment records the intended
  hand-off (Task 22 removes the duplicate when it lands).

### 7. Test quality — MIXED: reject-outcome assertions pass; saga step-content assertions are MISSING (BLOCKING)

**What's covered well:** all 13 subtests from the brief's table are present
(`duey_action_send_test.go:214-303`) and assert the specific client-visible
reject reason via a byte-code round-trip (`dueySendTestWP`,
`duey_action_send_test.go:70-98`) rather than merely "some announce
happened." The close-counter (NFR-5) is asserted unconditionally for every
subtest (`duey_action_send_test.go:349-352`, `if *closes != 0`), not merely
present-but-unchecked. `quick without a ticket` also asserts the warn-level
log text (`duey_action_send_test.go:355-357`).

**What's missing, and it's the brief's own requirement:** the brief's test
table (`task-17-brief.md:82-96`) specifies, per accept-path subtest, the
exact saga step composition — e.g. "step 0 `award_mesos` with `Amount`
`-(1000+5000)`, step 1 `transfer_to_parcel` with `Quick` false; **no**
`destroy_asset` step" for the NPC-send case, and "steps: `award_mesos`
`-(1000+0)`, `destroy_asset` of 5330000 qty 1, `transfer_to_parcel` with
`Quick` true and `Message` `"hi"`" for the quick-send case. The pattern this
task was explicitly told to copy, `note_send_test.go:28-44`, does exactly
this: it asserts `Steps[0].Action`, unpacks `Steps[0].Payload` to its
concrete payload type, and asserts fields on it.

`duey_action_send_test.go`'s accept-path assertion block
(`duey_action_send_test.go:317-323`) only checks `len(f.sagas) ==
tc.want.sagaLen` and `sg.SagaType == saga.ParcelSend`. It never inspects
`sg.Steps` — not the count, not the order, not the action per step, not any
payload field (`Amount`, `AssetId`, `Quantity`, `Quick`, `Message`,
`SourceInventoryType`, the conditional presence/absence of
`consume_quick_delivery_ticket`).

**This was verified empirically, not just by inspection.** I temporarily
hardcoded `Quick: false` into `buildParcelSendSaga`'s
`transfer_to_parcel` step (overriding the actual request's quick flag) and
reran `go test ./socket/handler/... -run TestDueyActionSend`: the full
suite, including the `quick send` subtest, still passed. The change was
reverted immediately (`git status` confirms the tree is clean). This proves
the "quick send" subtest — the one place the brief explicitly asks for
`Quick` true and `Message` "hi" to be asserted — does not actually pin the
saga's Quick-arm behavior. The same blind spot applies to the
`destroy_asset` step's presence/absence, the `award_mesos.Amount`
arithmetic, and `AssetId`/`Quantity` on `transfer_to_parcel`. A regression
in any of `buildParcelSendSaga`'s field wiring — the part of this task that
actually joins the send flow to the saga orchestrator — would ship silently
with this test suite green.

This is the task's own core deliverable (`buildParcelSendSaga`) shipping
with no test that would fail if it were wired wrong. Blocking.

## Not evaluable

None. Every check in the reviewer brief's priority list was answerable
within the diff plus its direct contract dependencies (atlas-parcel's
resource handler, atlas-saga's payload structs, the orchestrator's
expansion function).

## Summary

Six of seven priority checks pass cleanly, including the two checks most
likely to hide a real defect on this branch (the world-0 tag audit and the
saga-payload/orchestrator-expansion field match) and the brief-vs-code
signature question (ruled in the implementer's favor with cited evidence).
The one blocking finding is a test-quality gap: `buildParcelSendSaga`'s
step composition and field wiring — this task's actual joining logic — has
no assertion in the diff's own test suite, and that gap was confirmed live
by injecting a wiring bug and watching the suite stay green.

## Verdict

CHANGES_REQUIRED — the fix is scoped and small: extend
`duey_action_send_test.go`'s accept-path assertion block to unpack
`f.sagas[0].Steps` and assert step count/order/action, and the
`AwardMesosPayload.Amount`, `DestroyAssetPayload` presence-when-quick, and
`TransferToParcelPayload.{AssetId,Quantity,Quick,Message,
SourceInventoryType}` fields per the brief's table, following
`note_send_test.go:28-44`'s pattern. No production code change is required
by this review — the six PASS findings above stand.
