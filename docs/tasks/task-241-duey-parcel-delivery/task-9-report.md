# Task 9 report: PARCEL body-carrying notify arms

## Summary

Added the four body-carrying `PARCEL` clientbound arms (`ParcelRemoved`,
`ParcelArrived`, `AlarmNamed`, `AlarmGeneric`) to
`libs/atlas-packet/parcel/clientbound/parcel.go` /
`parcel_body.go`, four `#`-entry cases appended to
`tools/packet-audit/cmd/run.go`'s `candidatesFromFName`, and a new test
file `parcel_notify_test.go` covering all four plus the Task-8 carried
finding (`OpenQuick.Decode` test coverage).

All four modes and body shapes were re-confirmed directly against
`CParcelDlg::OnPacket` @0x6F56EA (v83, IDA session `41f09cce`) before
writing any code — case 23/24/25/27 decompile output is quoted below.

## Mode resolution — routed versions per key

Every mode is resolved via `WithResolvedCode("operations", KEY, …)` against
`docs/packets/dispatchers/parcel.yaml`; no literal mode byte appears in
`parcel.go`/`parcel_body.go` (self-audit grep below).

| key | struct | gms (v72/79/83/84/87/92/95) | jms_v185 in yaml? |
|---|---|---|---|
| `PARCEL_REMOVED` | `ParcelRemoved` | 0x17 (23) all seven | **yes** — 24 |
| `PARCEL_ARRIVED` | `ParcelArrived` | 0x18 (24) all seven | **yes** — 25 |
| `ALARM_NAMED` | `AlarmNamed` | 0x19 (25) all seven | **yes** — 26 |
| `ALARM_GENERIC` | `AlarmGeneric` | 0x1B (27) all seven | **yes** — 28 |

All four of Task 9's keys are among the 7 `jms_v185`-populated keys in
`parcel.yaml` (Ruling 5), so all four are routed for `jms_v185` too — none
were left GMS-only. This is called out explicitly per the task's
instruction to check each key against the yaml rather than assume.

## `ParcelRemovedKindClaimed` derivation

Decompiled `CParcelDlg::OnPacket` case 23 (mode 0x17, v83 @0x6F56EA) live,
IDA session `41f09cce`:

```
case 23:
  v18 = CInPacket::Decode4(v1);   // parcelId
  v19 = CInPacket::Decode1(v1);   // kind
  ...
  if ( v19 == 3 )                 // @0x6f5a62
    ... SP_3899_SUCCESSFULLY_DELETED ...
  else
    ... SP_3900_SUCCESSFULLY_CLAIMED ...   // @0x6f5a8e
```

The decompile confirms `kind == 3` → "deleted" (discarded) and pins nothing
more specific than "any other value" for "claimed" — there is no second
literal to cite. Per the brief's own framing ("the design establishes only
that the client shows 'claimed' for anything other than 3, not which value
the server sends"), `ParcelRemovedKindClaimed = byte(0)` is documented in
`parcel.go` as the canonical non-3 value chosen by this pass, not a
decompile-cited literal — flagged as such in the doc comment rather than
asserted as verified.

Case 24 (mode 0x18, PARCEL_ARRIVED) confirmed as a single
`PARCEL::Decode` call with no other body fields. Case 25 (mode 0x19,
ALARM_NAMED) confirmed as `CInPacket::DecodeStr` (senderName) +
`CInPacket::Decode1` (hasItem). Case 27 (mode 0x1B, ALARM_GENERIC)
confirmed as `CInPacket::Decode1` (hasItem) only. All match the brief's
arm table exactly.

## Files changed

- `libs/atlas-packet/parcel/clientbound/parcel.go` — appended
  `ParcelRemoved`, `ParcelArrived`, `AlarmNamed`, `AlarmGeneric` structs +
  `ParcelRemovedKindDiscarded`/`ParcelRemovedKindClaimed` constants.
- `libs/atlas-packet/parcel/clientbound/parcel_body.go` — appended the
  four operation-key consts and `ParcelRemovedBody`/`ParcelArrivedBody`/
  `ParcelAlarmNamedBody`/`ParcelAlarmGenericBody`.
- `libs/atlas-packet/parcel/clientbound/parcel_notify_test.go` — new file,
  `TestParcelNotifyArms` (six subtests) + `TestParcelOpenQuickDecode` (the
  carried Task-8 finding: `OpenQuick.Decode` now has coverage, behavior
  unchanged).
- `tools/packet-audit/cmd/run.go` — four appended `#`-entry cases
  (`ParcelRemoved`, `ParcelArrived`, `AlarmNamed`, `AlarmGeneric`); the
  fifteen prior parcel cases were left untouched (pure append, verified by
  diff).

## Verification (run after the last edit above)

```
$ go run ./tools/packet-audit dispatcher-lint
dispatcher-lint: clean
EXIT 0

$ go run ./tools/packet-audit fname-doc --check
fname-doc check OK (289 structs without an audit report carry no fname)
EXIT 0

$ go run ./tools/packet-audit operations --check
operations note (writer absent): gms_v72/79/83/84/87/92/95/jms_v185: writer
"DueyAction" not in template (cannot populate 4 ops; ...)
operations check OK (8 absent-writer note(s))
EXIT 0
```
(The `DueyAction` writer-absent notes are pre-existing, out of scope —
Task 10 wires that writer's opcode entries.)

```
$ go run ./tools/packet-audit matrix --check
note   n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 ...
note   n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 ...
matrix --check: docs/packets/audits/STATUS.md is stale — regenerate and commit
matrix --check: docs/packets/audits/status.json is stale
exit status 1
EXIT 1
```
Exit 1 is **expected per Ruling 6** — the only two failing lines are the
stale `STATUS.md`/`status.json` self-hash lines (Task 8 edited `run.go`;
Task 10 owns the single regeneration pass). No ORPHAN, coverage, or drift
complaint appeared. This task did **not** regenerate the matrix or commit
`STATUS.md`/`status.json`, per instruction.

Build/test/vet (both modules, after the final edit):

```
$ cd libs/atlas-packet && go build ./... && go vet ./... && go test ./...
ok   github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound  0.004s
(all other packages ok, no failures)

$ cd tools/packet-audit && go build ./... && go vet ./... && go test ./...
ok   github.com/Chronicle20/atlas/tools/packet-audit/cmd  2.239s
(all other packages ok)
```

Self-audit greps (both new/modified family files):

```
$ grep -rn 'mode:\s*0x' libs/atlas-packet/parcel/clientbound/parcel.go \
    libs/atlas-packet/parcel/clientbound/parcel_body.go
(no output — 0 matches)

$ grep -rn 'func(_ byte)' libs/atlas-packet/parcel/clientbound/parcel.go \
    libs/atlas-packet/parcel/clientbound/parcel_body.go
(no output — 0 matches)
```

`gofmt -l` on all touched files: no output (clean).

## Out of scope, left untouched

- `packet-audit:verify` markers — not added (assigned Task 28).
- `duey_action.yaml` provenance comment nit — Task 10.
- Task 10's serverbound `DueyAction` codecs.
- Matrix regeneration / `STATUS.md` / `status.json` — Task 10's single
  regeneration pass, per Ruling 6.

## Handoff to Task 10

- Mode-byte source of truth remains `docs/packets/dispatchers/parcel.yaml`
  and, for the serverbound side, `docs/packets/dispatchers/duey_action.yaml`
  (already present, SEND/RECEIVE/DISCARD/CLOSE modes populated for all
  eight versions).
- `parcel.Parcel`'s wire struct is unchanged (still the Task 7 234-byte
  fixed block + hasItem + optional item).
- All 19 clientbound `PARCEL` arms (Task 7's 2 + Task 8's 15 + this task's
  4) are now present in `parcel.go`/`parcel_body.go`/`run.go`.
  `docs/packets/dispatchers/parcel.yaml`'s 21-key operations list is fully
  wired.
- The family is not present in `docs/packets/dispatcher-lint-baseline.yaml`
  (dispatcher-lint was already clean going in and stayed clean).
