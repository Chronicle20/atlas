# Task 10 report — `DUEY_ACTION` serverbound codecs

Branch `task-241-duey-parcel-delivery`, worktree
`<repo-root>/.worktrees/task-241-duey-parcel-delivery`.

Commits:
- `6bb465f86` — `feat(packet): DUEY_ACTION serverbound codecs`
- `586ee9ebb` — `chore(packets): regenerate matrix status after DUEY_ACTION run.go commit`

HEAD at report time: `586ee9ebb`.

## What was built

`libs/atlas-packet/parcel/serverbound/`:

- `action.go` — `Action` (mode-byte dispatcher; mirrors
  `storage/serverbound.Operation` exactly), `DueyActionHandle` const.
- `action_send.go` — `ActionSend` (SEND arm, mode 2 / jms_v185 mode 3):
  `byte invType, uint16 slot, uint16 quantity, uint32 mesos,
  string recipientName, bool quick`, and **only when `quick`**:
  `string message, uint32 ticketRef`. NPC send (v83
  `CTabSend::SendParcel` @0x6F36A8) stops after the flag; quick send
  (@0x6F1DF5) appends the two trailing fields — `Decode` gates them on
  the flag it just read.
- `action_parcel_id.go` — `ActionReceive` (RECEIVE, mode 4/5,
  `CTabReceive::ReceiveParcel` @0x6F0CA3), `ActionDiscard` (DISCARD,
  mode 5/6, `CTabReceive::DiscardParcel` @0x6F0DC3) — same
  `uint32 parcelId` wire shape as `ActionReceive` but kept as a
  **separate struct** per the brief's constraint 5 — and `ActionClose`
  (CLOSE, mode 7/8, `CParcelDlg::CloseParcelDlg` @0x6F5691, no body).
- `action_test.go` — `TestDueyActionDecode`, table-driven, all 7
  subtests from the brief: `mode send`, `send npc`, `send quick`,
  `send npc trailing garbage` (asserts the reader has 4 unconsumed
  bytes after `Decode`, proving the quick-only fields aren't read on
  the NPC path), `receive`, `discard`, `close`. Built directly against
  hand-crafted wire bytes (not `pt.RoundTrip`) so the malformed-input
  case is expressible; `pt.RoundTrip`'s "0 bytes left" assertion
  wouldn't allow it.

`tools/packet-audit/cmd/run.go` — appended four new cases to
`candidatesFromFName` (zero lines deleted — confirmed via
`git diff --stat` showing `24 ++++...` / `0 --`):
`CTabSend::SendParcel` → `ActionSend`, `CTabReceive::ReceiveParcel` →
`ActionReceive`, `CTabReceive::DiscardParcel` → `ActionDiscard`,
`CParcelDlg::CloseParcelDlg` → `ActionClose`, all `pkg: "parcel"`,
`dir: csvpkg.DirServerbound`.

No `WithResolvedCode`/body-function layer was added. This is correct
for this family: `DUEY_ACTION` is a **serverbound** dispatcher (client
constructs the outgoing bytes; Atlas only decodes what the client
already sent), and `DISPATCHER_FAMILY.md`'s canonical pattern (steps
1–6, `WithResolvedCode`, per-mode body functions, FAM-CAP) is
explicitly scoped to **clientbound** mode-prefix demultiplexers — see
its "Serverbound dispatcher files are out of scope for FAM-CAP"
section. The reference pattern the brief names
(`storage/serverbound/operation.go` + `operation_store_asset.go`) is
itself decode-only with no `WithResolvedCode`/body-function layer, and
that is what `Action`/`ActionSend`/`ActionReceive`/`ActionDiscard`/
`ActionClose` mirror. `docs/packets/dispatchers/duey_action.yaml`
already carries `direction: serverbound`, opting it out of FAM-CAP the
same way. Mode-byte resolution against that YAML (raw byte → semantic
key, e.g. `isStorageOperation`-style) happens in the **atlas-channel**
handler, which is explicitly out of scope (Tasks 17/18).

## Controller-assigned extras

1. **Matrix regeneration.** Landed in a **second commit**
   (`586ee9ebb`), not folded into the `run.go` commit. Reason: `matrix`
   computes the `Tool:`/`toolSha` line from
   `git ls-tree -r HEAD tools/packet-audit` — i.e. the **committed**
   tree, not the working tree. Regenerating in the same commit as the
   `run.go` edit computes the hash against the *pre-commit* HEAD
   (before that edit landed), so the freshly-committed STATUS.md/
   status.json go stale the instant the commit lands — reproduced this
   directly: `matrix --check` was green immediately pre-commit, then
   red immediately post-commit with the identical file, only the git
   HEAD state differing. Fixed by running the regeneration **after**
   `run.go`'s commit was already on HEAD, verifying the resulting diff
   touched only the `Tool:`/`toolSha` line in both files (confirmed
   with `git diff` before staging), and committing that as its own
   change. `matrix --check` is green on the final HEAD, re-verified
   twice.
2. **`duey_action.yaml` provenance comment.** Fixed in
   `6bb465f86`: the top-of-file "SOURCE OF TRUTH" comment (grouped
   under the GMS v83 heading) cited `CTabReceive::ReceiveParcel
   @0x65AF41` — a v72 address — for RECEIVE. Replaced with v83's own
   `ReceiveParcel` address, `@0x6F0CA3` (matches design.md §5.4 and the
   per-version block further down the same file, which already had it
   right). No mode value touched; RECEIVE stays 4 (jms_v185: 5)
   everywhere.

## Verification (all on final HEAD `586ee9ebb`)

```
$ go run ./tools/packet-audit matrix --check
note  n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 (docs/packets/feature-na-evidence.yaml)
note  n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (docs/packets/feature-na-evidence.yaml)
matrix EXIT=0

$ go run ./tools/packet-audit dispatcher-lint
dispatcher-lint: clean
lint EXIT=0

$ go run ./tools/packet-audit fname-doc --check
fname-doc check OK (294 structs without an audit report carry no fname)
fname EXIT=0

$ go run ./tools/packet-audit operations --check
operations note (writer absent): gms_v72: writer "DueyAction" not in template (cannot populate 4 ops; add an opcodes entry to the YAML to wire it)
operations note (writer absent): gms_v79: writer "DueyAction" not in template ...
operations note (writer absent): gms_v83: writer "DueyAction" not in template ...
operations note (writer absent): gms_v84: writer "DueyAction" not in template ...
operations note (writer absent): gms_v87: writer "DueyAction" not in template ...
operations note (writer absent): gms_v92: writer "DueyAction" not in template ...
operations note (writer absent): gms_v95: writer "DueyAction" not in template ...
operations note (writer absent): jms_v185: writer "DueyAction" not in template ...
operations check OK (8 absent-writer note(s))
ops EXIT=0
```

**On the pre-existing `DueyAction` writer-absent notes:** these persist
after this task and are correctly out of scope for it. The note comes
from `packet-audit operations`, which compares
`docs/packets/dispatchers/duey_action.yaml` against per-version
**tenant seed template** files under
`services/atlas-configurations/seed-data/templates` (the
`--templates-dir` default in `tools/packet-audit/cmd/operations.go`).
Task 10's file inventory is `libs/atlas-packet` + `run.go` only — no
seed template is in scope, and the brief explicitly carves out
"atlas-channel handling (Tasks 17/18)". Wiring `DueyAction` into each
version's seed template `opcodes` entry (so `options.operations` can
be populated per tenant) is config/deploy work belonging to whichever
task adds the live `DueyAction` writer wiring — most likely Task 17/18
alongside the atlas-channel handler that will actually consume
`Action.Mode()`. This task adds the codec side only; it does not (and
per its file inventory should not) touch the seed templates. The
check remains a non-blocking **note**, exit 0, both before and after
this task's commits — unchanged by this task, as expected.

```
$ (cd libs/atlas-packet && go build ./... && go vet ./... && go test ./parcel/...)
Go test: 41 passed in 3 packages

$ (cd tools/packet-audit && go build ./... && go vet ./... && go test ./cmd/...)
Go test: 199 passed in 1 packages

$ grep -rn 'mode:\s*0x' libs/atlas-packet/parcel/serverbound/
(no matches)

$ grep -rn 'func(_ byte)' libs/atlas-packet/parcel/serverbound/
(no matches)
```

## Mode → struct table

| Mode (GMS / jms_v185) | Key | Struct | fname |
|---|---|---|---|
| 2 / 3 | SEND | `ActionSend` | `CTabSend::SendParcel` |
| 4 / 5 | RECEIVE | `ActionReceive` | `CTabReceive::ReceiveParcel` |
| 5 / 6 | DISCARD | `ActionDiscard` | `CTabReceive::DiscardParcel` |
| 7 / 8 | CLOSE | `ActionClose` | `CParcelDlg::CloseParcelDlg` |

All four modes route on every version `duey_action.yaml` covers
(gms_v72/v79/v83/v84/v87/v92/v95, jms_v185) — the yaml is fully
populated for jms per the task instructions, and this task's `run.go`
entries are not version-gated (the struct choice is the same across
all versions; only the per-version mode byte differs, which is a
config-table concern, not a struct-selection one).

## Task 10 is the last packet task in this block

This closes out the `DUEY_ACTION` serverbound side and, per the
controller-assigned extras, the branch's deferred matrix-staleness
item. Tasks 17/18 (atlas-channel) will construct `Action`/`ActionSend`/
`ActionReceive`/`ActionDiscard`/`ActionClose`, resolve the mode byte
against a tenant `operations` table the same way
`storage_operation.go`'s `isStorageOperation` does, and add the
`DueyAction` writer entry to each version's seed template (closing the
`operations --check` writer-absent notes as a side effect of that
work, not this one).
