# Task 25 review — atlas-channel: parcel arrival alarm consumer

Commit reviewed: `b8efda7` "feat(channel): parcel arrival alarm consumer"
Brief: `.superpowers/sdd/plan/task-25-brief.md`
Report: `.superpowers/sdd/plan/task-25-report.md`

## Scope

Reviewed the full diff of `b8efda7` (3 files, 254 insertions / 1 deletion):

- `services/atlas-channel/atlas.com/channel/kafka/message/parcel/kafka.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/status_test.go` (new)

Plus, as the seam contract this task must not break, the producer file
`services/atlas-parcel/atlas.com/parcel/kafka/message/parcel/kafka.go` and the
mirrored pattern at
`services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go:454-482`,
and the config-resolved constructor
`libs/atlas-packet/parcel/clientbound/parcel_body.go:198-203` plus its
verbatim test fixture at
`libs/atlas-packet/parcel/clientbound/parcel_notify_test.go:78-92`.
`main.go` was inspected (not modified in this commit) to confirm the
implementer's no-edit claim.

## 1. Producer/consumer contract equality

Compared byte-for-byte:

Producer (`services/atlas-parcel/atlas.com/parcel/kafka/message/parcel/kafka.go`):
```go
const EnvStatusEventTopic = "EVENT_TOPIC_PARCEL_STATUS"
const StatusEventParcelArrived = "PARCEL_ARRIVED"
type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}
type StatusEventParcelArrivedBody struct {
	SenderName string `json:"senderName"`
	HasItem    bool   `json:"hasItem"`
}
```

Consumer, this commit
(`services/atlas-channel/atlas.com/channel/kafka/message/parcel/kafka.go:31-64`):
identical topic string, `Type` constant, envelope field names/JSON tags, and
body field names/JSON tags. Confirmed with independent `cat` of both files —
the implementer's side-by-side quote in the report is accurate.

**PASS.**

## 2. No literal mode byte in the handler

`handleParcelArrivedEvent`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go:218`)
calls `parcelcb.ParcelAlarmNamedBody(e.Body.SenderName, e.Body.HasItem)`,
which resolves through
`atlas_packet.WithResolvedCode("operations", ParcelOperationAlarmNamed, ...)`
(`libs/atlas-packet/parcel/clientbound/parcel_body.go:198-203`). No literal
0x19/25 anywhere in production code.

The only literal mode byte (`0x19`, in `alarmNamedOperations` in
`status_test.go:18-20`) is inside a test's tenant-operations fixture map,
matching the shape used in
`libs/atlas-packet/parcel/clientbound/parcel_notify_test.go:78-82` — the
legitimate exception per the controller's notes.

**PASS.**

## 3. Tenant guard

`consumer.go:213-216`:
```go
t := tenant.MustFromContext(ctx)
if !t.Is(sc.Tenant()) {
	return
}
```
placed after the `Type` guard and before the announce call — mirrors
`handleFrederickNotificationEvent`
(`kafka/consumer/merchant/consumer.go:454-461`) exactly, and rejects before
any `IfPresentByCharacterId`/announce call. The `wrong tenant` subtest
(`status_test.go:150-169`) exercises this: constructs the event under a
different tenant in context than the session's tenant and asserts zero
captured announces.

**PASS.**

## 4. main.go claim

Confirmed by reading `services/atlas-channel/atlas.com/channel/main.go`:
- line 49: `parcelConsumer "atlas-channel/kafka/consumer/parcel"`
- line 267: `parcelConsumer.InitConsumers(l)(cmf)(consumerGroupId)`
- line 601: `register(parcelConsumer.InitHandlers(fl)(sc)(wp)(rh))`

Both calls invoke the whole `kafka/consumer/parcel` package's
`InitConsumers`/`InitHandlers` functions. This commit's diff
(`consumer.go` hunks) adds the new `EnvStatusEventTopic` registration inside
`InitConsumers` and the new `handleParcelArrivedEvent` registration inside
`InitHandlers`, in the same functions `main.go` already calls. No separate
wiring point was needed; the claim is verified true, not an omission.

**PASS.**

## 5. Brief's five subtests

All five present in `status_test.go` and match the brief's table:

| subtest | location | assertion checked |
|---|---|---|
| `online recipient` | :64-90 | 1 capture, decoded senderName == "Alice", hasItem == true |
| `no item` | :106-127 | 1 capture, decoded hasItem == false |
| `offline recipient` | :129-144 | no session created for characterId 100; event addressed to characterId 999; 0 captures |
| `wrong tenant` | :146-167 | event's `ctx` tenant differs from `sc`'s tenant; 0 captures |
| `wrong event type` | :169-178 (truncated in slice, confirmed present) | different `Type`; 0 captures |

`offline recipient` implicitly covers "no error": `IfPresentByCharacterId`
(`session/processor.go:181-189`) swallows the not-found provider error and
returns `nil` when no session matches, so the handler's own
`l.WithError(err).Errorf(...)` branch (`consumer.go:227-229`) is unreachable
in this path — confirmed by reading `ByCharacterIdModelProvider` /
`IfPresentByCharacterId`. The test doesn't assert on an error return value
directly (the handler has no return value to assert on), but the
unreachable-error-path reasoning holds and the subtest name matches the
brief's `no announce, NO error` intent.

**PASS.**

## 6. Design §7.1 trade-off recorded in doc comment

`consumer.go:200-207` (doc comment on `handleParcelArrivedEvent`):
> "This always resolves to ALARM_NAMED (0x19), never PARCEL_ARRIVED (0x18).
> Design §7.1 makes this an explicit trade: 0x18 would both append the
> dialog row and raise SP_3902, but selecting it requires knowing the
> session has an open parcel dialog, which the channel does not track. A
> player with the dialog open sees a toast instead of a live row —
> accepted, low severity."

Matches the brief's Step 3 requirement verbatim in substance.

**PASS.**

## Verification

Independently re-ran (not just trusted the report):
```
$ go build ./...          # clean
$ go test ./kafka/consumer/parcel/... -run TestParcelArrivedEvent -v
--- PASS: TestParcelArrivedEvent (all 5 subtests PASS)
```

## Non-blocking notes

- None.

## Not evaluable

- None — the full commit and its cross-service seam (topic name, envelope,
  body, tenant guard, mode-byte resolution, main.go wiring) were all
  directly checked against the producer source and the mirrored pattern.

## Verdict

All six review points PASS with direct file:line evidence. No blocking or
non-blocking findings.
