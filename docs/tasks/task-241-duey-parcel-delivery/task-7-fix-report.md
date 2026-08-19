# Task 7 fix round — report

Commit: `29ec83a32` (parent `797baebef`), branch `task-241-duey-parcel-delivery`.
`git rev-parse --abbrev-ref HEAD` confirmed `task-241-duey-parcel-delivery`
after commit.

## Item 1 — regenerate stale matrix status (blocking)

Root cause (already isolated by review): commit `797baebef` added two case
arms to `tools/packet-audit/cmd/run.go`, changing the packet-audit tool's
self-hash, without regenerating `docs/packets/audits/STATUS.md` /
`status.json`.

Before fix:
```
$ go run ./tools/packet-audit matrix --check
matrix --check: docs/packets/audits/STATUS.md is stale — regenerate and commit
matrix --check: docs/packets/audits/status.json is stale
exit status 1
```

Fix: `go run ./tools/packet-audit matrix` (no `-check`), then diffed both
files before staging:

```
docs/packets/audits/STATUS.md
  @@ -4,7 +4,7 @@
  -Tool: `319a7ad9b3347b81a4a0eed68718ebad0639d1ae89d0a93857f0c26f6ac9ca21`
  +Tool: `9d9dd920191eaa8785d3551484f998cf54ac3b6afbe02b97a9412a539671a2fb`

docs/packets/audits/status.json
  @@ -1,5 +1,5 @@
  -  "toolSha": "319a7ad9b3347b81a4a0eed68718ebad0639d1ae89d0a93857f0c26f6ac9ca21",
  +  "toolSha": "9d9dd920191eaa8785d3551484f998cf54ac3b6afbe02b97a9412a539671a2fb",
```

Only the `Tool:`/`toolSha` hash line changed in each file — no coverage
cell, orphan entry, or drift entry moved. Committed as-is.

## Item 2 — parcel.go: discard-flow function name

`libs/atlas-packet/parcel/parcel.go`'s comment for the `+0 uint32 parcelId`
field attributed the discard-flow encode at `0x6F0DC3` to "RemoveParcel".
The function at that exact address is unnamed (`sub_6F0DC3`) in the v83 IDB
— corrected the comment to cite `sub_6F0DC3` instead of a name the binary
does not carry. Address and `+0` semantics were already correct and are
unchanged.

## Item 3 — parcel.go: v83 corroboration for +21 sentAt

Added the v83 `CTabReceive::ReceiveParcel @0x6F0D11` `<30`-day-check
corroboration for the `+21 uint64 sentAt` field alongside the pre-existing
v72 `ReceiveParcel @0x65AF41` citation (both cited in the review artifact,
`docs/tasks/task-241-duey-parcel-delivery/reviews/task-7.md`).

## Verification (run in order, after the final edit)

```
$ go run ./tools/packet-audit matrix --check
note	n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 (docs/packets/feature-na-evidence.yaml)
note	n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (docs/packets/feature-na-evidence.yaml)
EXIT:0

$ go run ./tools/packet-audit dispatcher-lint
dispatcher-lint: clean
EXIT:0

$ go run ./tools/packet-audit fname-doc --check
fname-doc check OK (270 structs without an audit report carry no fname)
EXIT:0

$ go run ./tools/packet-audit operations --check
operations note (writer absent): gms_v72/v79/v83/v84/v87/v92/v95/jms_v185:
  writer "DueyAction" not in template (cannot populate 4 ops; add an
  opcodes entry to the YAML to wire it)   [8 pre-existing notes, unrelated
  to this fix round — out of scope]
operations check OK (8 absent-writer note(s))
EXIT:0
```

Module builds/tests (post-edit):

```
$ (cd libs/atlas-packet && go build ./... && go test ./...)
EXIT:0 / all packages ok (including parcel, parcel/clientbound)

$ (cd tools/packet-audit && go build ./... && go test ./...)
EXIT:0 / all packages ok
```

Self-audit greps:
```
$ grep -rn 'mode:\s*0x' libs/atlas-packet/parcel/clientbound/parcel.go
(no matches)

$ grep -rn 'func(_ byte)' libs/atlas-packet/parcel/clientbound/parcel_body.go
(no matches)
```

## Scope discipline

- Did not touch the `+29..233` message-span inference or the `Parcel` struct
  shape — out of scope per brief, independently reviewed and resolved.
- Did not add or restore any `packet-audit:verify` markers — that remains
  Task 28's scope.
- Did not touch any jms_v185 notice-arm values.
- Staged and committed only the three intended files
  (`docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`,
  `libs/atlas-packet/parcel/parcel.go`); did not use `git add -A`/`.`.

## Files changed

- `docs/packets/audits/STATUS.md`
- `docs/packets/audits/status.json`
- `libs/atlas-packet/parcel/parcel.go`
