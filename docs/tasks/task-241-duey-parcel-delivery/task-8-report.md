# Task 8: PARCEL bodyless result arms — report

Commit: `f1e7d0229` on branch `task-241-duey-parcel-delivery` (verified via
`git rev-parse --abbrev-ref HEAD` after commit).

## What was implemented

Fifteen discrete, bodyless `struct { mode byte }` arms appended to
`libs/atlas-packet/parcel/clientbound/parcel.go`, each with its own
`Mode()`/`Operation()`/`String()`/`Encode`/`Decode` and a
`packet-audit:fname CParcelDlg::OnPacket#<StructName>` doc comment. Fifteen
matching per-mode body functions (`Parcel<StructName>Body()`, no arguments)
appended to `libs/atlas-packet/parcel/clientbound/parcel_body.go`, each fixing
its operation-key const and resolving the mode via
`atlas_packet.WithResolvedCode("operations", <fixed key>, func(mode byte) …)`
— never a hard-coded byte. Fifteen matching `case "CParcelDlg::OnPacket#<Struct>":`
entries appended to `tools/packet-audit/cmd/run.go`'s `candidatesFromFName`
(existing switch untouched, only appended to). New test file
`libs/atlas-packet/parcel/clientbound/parcel_result_test.go` with
`TestParcelResultArms`, a table over all fifteen arms plus the
`unresolved key falls back` negative case (asserts the documented 99-sentinel
fallback, no panic).

No `packet-audit:verify` markers were added, per the explicit task-8 dispatch
instruction overriding the brief's step 1 (Task 7 already discovered these
get rejected as ORPHANS without a real per-version IDA-export + audit-report
pair — that's Task 28's job).

## Arm → key → mode(s) table

All fifteen keys resolve identically across all seven GMS versions
(`gms_v72/79/83/84/87/92/95`), per `docs/packets/dispatchers/parcel.yaml`
(Task 6, independently re-derived per version). None of these fifteen keys
carry a `jms_v185` entry in that table except `SUCCESSFULLY_SENT`.

| key const | struct | GMS mode (all 7 versions) | jms_v185 |
|---|---|---|---|
| `SEND_ENABLE_ACTIONS` | `SendEnableActions` | 0x09 | unset |
| `NOT_ENOUGH_MESOS` | `NotEnoughMesos` | 0x0A | unset |
| `INCORRECT_REQUEST` | `IncorrectRequest` | 0x0B | unset |
| `NAME_DOES_NOT_EXIST` | `NameDoesNotExist` | 0x0C | unset |
| `SAME_ACCOUNT` | `SameAccount` | 0x0D | unset |
| `RECEIVER_STORAGE_FULL` | `ReceiverStorageFull` | 0x0E | unset |
| `RECEIVER_UNABLE_TO_RECEIVE` | `ReceiverUnableToReceive` | 0x0F | unset |
| `SENDER_UNIQUE_CONFLICT` | `SenderUniqueConflict` | 0x10 | unset |
| `MESO_LIMIT` | `MesoLimit` | 0x11 | unset |
| `SUCCESSFULLY_SENT` | `SuccessfullySent` | 0x12 | **0x13** (already present in parcel.yaml before this task — confirmed via the `a1==19` `CloseParcelDlg` side-effect check, task-6 evidence; not something this task added) |
| `UNKNOWN_ERROR` | `UnknownError` | 0x13 | unset |
| `RECV_ENABLE_ACTIONS` | `RecvEnableActions` | 0x14 | unset |
| `RECV_NO_FREE_SLOTS` | `RecvNoFreeSlots` | 0x15 | unset |
| `RECV_UNIQUE_CONFLICT` | `RecvUniqueConflict` | 0x16 | unset |
| `UNKNOWN_ERROR_2` | `UnknownError2` | 0x1C | unset |

**Confirmation: no jms_v185 value was written for any of the fourteen unset
keys.** `docs/packets/dispatchers/parcel.yaml` was read, not edited, by this
task — its `jms_v185` column is exactly as Task 6 left it (Ruling 5). The one
`jms_v185` value among these fifteen keys (`SUCCESSFULLY_SENT: 19`) was
already present in the file before this task started; nothing here added,
changed, or guessed it.

These arms are GMS-routed by construction: `WithResolvedCode` reads whatever
the tenant's `operations` table (built from this YAML at seed-template time)
contains for the resolved tenant/version — for a GMS tenant every key above
resolves; for the JMS tenant only `SUCCESSFULLY_SENT` (and `OPEN`, from
Task 7) resolve, and the other 13 of these 15 keys simply are not present in
the JMS operations table, so `ResolveCode` on a JMS tenant would hit the
loud 99-sentinel/error-log path rather than emit a guessed byte — exactly
the documented miss behaviour, verified by this task's
`unresolved key falls back` test case.

## Verification (all run after the final `run.go` edit)

```
$ go run ./tools/packet-audit dispatcher-lint
dispatcher-lint: clean
(exit 0)

$ go run ./tools/packet-audit matrix
wrote docs/packets/audits/STATUS.md and docs/packets/audits/status.json
(no diff produced — these bodyless arms have no export/audit-report pairs
yet, so no matrix cell changed; STATUS.md/status.json already reflected the
correct state and needed no commit)

$ go run ./tools/packet-audit matrix --check
note  n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 (pre-existing, unrelated)
note  n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (pre-existing, unrelated)
(exit 0)

$ go run ./tools/packet-audit fname-doc --check
fname-doc check OK (285 structs without an audit report carry no fname)
(exit 0)

$ go run ./tools/packet-audit operations --check
operations note (writer absent): gms_v72/79/83/84/87/92/95/jms_v185: writer "DueyAction"
not in template (cannot populate 4 ops; add an opcodes entry to the YAML to wire it)
operations check OK (8 absent-writer note(s))
(exit 0 — pre-existing DueyAction writer-absent notes, out of scope per dispatch instructions)
```

Self-audit greps:

```
$ grep -rn 'mode:\s*0x' libs/atlas-packet/parcel/clientbound/parcel.go
(no matches)

$ grep -rn 'func(_ byte)' libs/atlas-packet/parcel/clientbound/parcel_body.go
(no matches)
```

Module-local build/vet/test:

```
$ cd libs/atlas-packet && go build ./... && go vet ./... && go test ./...
ok for every package with tests, including parcel/clientbound

$ cd tools/packet-audit && go build ./... && go vet ./... && go test ./...
(no test files; build/vet clean)
```

## "Family complete" checklist (task-8 scope only — Tasks 9/10 still pending)

- [x] One discrete struct per supported mode (these fifteen), in the one
      consolidated `parcel.go` file.
- [x] Each struct's `Encode` writes its full arm body — for these arms that
      body is exactly the mode byte (bodyless is the ground truth per
      `CParcelDlg::NoticeResult @0x6F5BE2`, cited in the file-header comment
      added by this task).
- [x] Every constructor takes `mode byte`; every body func resolves it via
      `WithResolvedCode("operations", FIXED_KEY, func(mode byte)…)` — zero
      `mode: 0x` literals, zero `func(_ byte)` (confirmed by grep above).
- [x] No body func takes a caller-supplied op/code/mode/key selector (all
      fifteen `Parcel*Body()` funcs take zero arguments).
- [x] No struct serves >1 mode; no dangling `#`-entry (dispatcher-lint clean,
      INV-1/INV-4).
- [ ] Per-mode export entry + audit report + byte-fixture(marker) + evidence
      — explicitly deferred to Task 28 per Ruling 5/dispatch instructions;
      NOT claimed done here.
- [x] `dispatcher-lint`, `matrix --check`, `fname-doc --check`,
      `operations --check` all exit 0; `go build/vet/test` clean (no `-race`
      run, per instructions — that's out of this task's scope).
- [x] Family not newly added to `dispatcher-lint-baseline.yaml` (parcel was
      never baselined; still clean).

## Remaining work (not this task)

- Task 9: the body-carrying PARCEL result arms (e.g. `PARCEL_REMOVED`,
  `PARCEL_ARRIVED`, `ALARM_NAMED`, `ALARM_GENERIC`).
- Task 10: PARCEL serverbound codecs.
- Task 28 (separate, future): per-version IDA-export + evidence/audit-report
  pass to move these arms' matrix cells past `🧩`/unverified, and — if new
  binary evidence is ever found — a dedicated verification pass to resolve
  the 14 currently-unset `jms_v185` keys. No such evidence was found or used
  in this task; Ruling 5 stands unmodified.
