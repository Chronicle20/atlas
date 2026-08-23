# Review: task-28 batch gms_v79 (commit `cdaae842e`, range `238faead6..cdaae842e`)

Reviewer: atlas-reviewer (Sonnet 5)
Brief: `.superpowers/sdd/plan/task-28-batch-gms_v79-brief.md`
Implementer report: `.superpowers/sdd/plan/task-28-batch-gms_v79-report.md`

## Scope

Reviewed the full diff of `238faead6..cdaae842e` (59 files, 1725
insertions / 18 deletions): the `gms_v79.json` export splice, the 25
generated audit reports (`docs/packets/audits/gms_v79/Parcel*.{json,md}`),
the 4 DUEY_ACTION evidence YAMLs, the two new fixture files
(`libs/atlas-packet/parcel/{clientbound,serverbound}/v79_test.go`, read at
`cdaae842e` via `git show`, per instructions — a separate agent's
concurrent working-tree edits to the clientbound file were not consulted),
and `STATUS.md`/`status.json`. Cross-referenced against the two dispatcher
YAMLs (`parcel.yaml`, `duey_action.yaml`), the v83 anchor fixtures
(`libs/atlas-packet/parcel/clientbound/v83_test.go`), and the un-gated
`libs/atlas-packet/parcel/` codec. Scope matches the brief; no drift.

## 1. Marker addresses — PASS

All 25 `packet-audit:verify ida=0x...` markers (21 clientbound + 4
serverbound) were diffed against the spliced export entries in
`docs/packets/ida-exports/gms_v79.json` (`git diff 238faead6..cdaae842e --
docs/packets/ida-exports/gms_v79.json`). Every marker address matches its
corresponding export entry's `"address"` field exactly, one-to-one, no
extras and no gaps:

- 21 `CParcelDlg::OnPacket#<Arm>` entries (`0x683b4e` Open ... `0x68382b`
  AlarmGeneric) match the 21 `packet=parcel/clientbound/*` markers in
  `libs/atlas-packet/parcel/clientbound/v79_test.go`.
- `CTabSend::SendParcel` (`0x68170f`), `CTabReceive::ReceiveParcel`
  (`0x67ed0c`), `CTabReceive::DiscardParcel` (`0x67ee2c`),
  `CParcelDlg::CloseParcelDlg` (`0x6836b0`) match the 4
  `packet=parcel/serverbound/*` markers in
  `libs/atlas-packet/parcel/serverbound/v79_test.go`.

The 4 DUEY_ACTION evidence YAMLs' `ida.address` fields also match these
same 4 addresses exactly (verified per-file, e.g.
`docs/packets/evidence/gms_v79/parcel.serverbound.ParcelActionSend.yaml`
→ `0x68170f`).

The generated audit reports independently corroborate: e.g.
`docs/packets/audits/gms_v79/ParcelOpen.json` → `"Address": "0x683b4e"`,
`ParcelActionSend.json` → `"Address": "0x68170f"`, both with
`Verdict: 0` on every row (VerdictMatch).

## 2. Arm completeness — PASS

`yq -r '.operations[].key' docs/packets/dispatchers/parcel.yaml` → 21 keys
(OPEN, SEND_ENABLE_ACTIONS, NOT_ENOUGH_MESOS, INCORRECT_REQUEST,
NAME_DOES_NOT_EXIST, SAME_ACCOUNT, RECEIVER_STORAGE_FULL,
RECEIVER_UNABLE_TO_RECEIVE, SENDER_UNIQUE_CONFLICT, MESO_LIMIT,
SUCCESSFULLY_SENT, UNKNOWN_ERROR, RECV_ENABLE_ACTIONS, RECV_NO_FREE_SLOTS,
RECV_UNIQUE_CONFLICT, PARCEL_REMOVED, PARCEL_ARRIVED, ALARM_NAMED,
OPEN_QUICK, ALARM_GENERIC, UNKNOWN_ERROR_2) — all 21 have a corresponding
marker in `v79_test.go` (clientbound).

`yq -r '.operations[].key' docs/packets/dispatchers/duey_action.yaml` → 4
keys (CLOSE, DISCARD, RECEIVE, SEND) — all 4 have a corresponding marker in
`v79_test.go` (serverbound).

25 markers total, 25 generated report files, 4 evidence YAMLs — counts all
reconcile.

## 3. Duplicate-key hazard — PASS

`git show cdaae842e:docs/packets/ida-exports/gms_v79.json | grep -n
'CPet::OnNameChanged'` → still two occurrences (lines 380 and 8045), plus
one `"ref"` at line 11017.

- Line 380 (first occurrence): `"direction": "clientbound"`, populated
  comments (`"name"`, `"name-tag decoration layer selector
  (CLife::MakeNameTag)"`) — the annotated entry, unchanged.
- Line 8045 (second occurrence): `"direction": ""`, empty comments
  (`""`, `""`) — the bare stub, unchanged.

The annotated entry was NOT replaced by the stub (both entries' content
matches the pre-splice state; the splice touched only the `+346` new
block after `"functions": {` and did not modify existing lines,
confirmed by `--numstat` showing 0 deletions). The distinction the brief
called out is preserved exactly.

## 4. Byte-equality scope — PASS

`grep -rn "MajorAtLeast" libs/atlas-packet/parcel/` → no matches, confirming
the family carries zero version gates for v79 as it did for v83/v72 in
prior batches.

Diffed `v79_test.go`'s literal `want` byte sequences against
`v83_test.go`'s (`TestParcelResultArmsV83`, `TestParcelOpenQuickV83`,
`TestParcelRemovedV83`, `TestParcelArrivedV83`, `TestParcelAlarmNamedV83`,
`TestParcelAlarmGenericV83`, `TestParcelOpenV83`): structurally identical
test bodies, differing only in the `ida=` marker address and
`pt.CreateContext("GMS", 79, 1)` version argument — each arm's export-entry
notes independently cite this IDB's own decompile address and read order
(e.g. `sub_683BFE @0x683bfe` two-nested-switch shape, `sub_4D85F0
@0x4d85f0` PARCEL::Decode-equivalent), satisfying Ruling D's "must still
decompile this IDB" requirement rather than blindly copying v83's
literals. No PARCEL/DUEY_ACTION arm was equality-asserted where a
version-gate should have forced independent derivation — the codec has
none, so equality is correctly scoped.

**`CTabQuickSend::SendQuickDelivery` handling confirmed correct.** Per the
export entry for `CTabSend::SendParcel` (`git diff` lines around 286-326),
the quick-send site `@0x67fe5d` (found via `func_query`, outside the
brief's roster, same as v72) is folded into the `CTabSend::SendParcel`
export entry's `calls` list as two extra guarded rows
(`message`/`ticketRef`, `guard: "quick != 0"`) rather than given its own
export key or marker — this is the same collapse-into-one-entry pattern
the batch-2 (v72) reviewer confirmed against the v83 anchor. No duplicate
or orphaned marker for the quick-send site.

## 5. Evidence — PASS

`git diff --stat 238faead6..cdaae842e -- docs/packets/evidence/` shows
exactly 4 new files, all under `docs/packets/evidence/gms_v79/`, all
`parcel.serverbound.ParcelAction{Send,Receive,Discard,Close}.yaml`. Each
has a `verifies:` field pointing at its corresponding `v79_test.go` test
function (`TestActionSendV79` etc.) and `category: TIER1-FIXTURE`. No
PARCEL (clientbound) evidence file was added — correct per Ruling A
(`tier1: false` clientbound promotes on tool-check + marker alone).

## Independent re-verification (not just re-reading the report)

Ran all five gate commands fresh in this worktree, all exit 0:

```
go run ./tools/packet-audit matrix --check   → exit 0 (only pre-existing n-a-evidence notes)
go run ./tools/packet-audit dispatcher-lint  → "dispatcher-lint: clean"
go run ./tools/packet-audit fname-doc --check → "fname-doc check OK (269 structs without an audit report carry no fname)"
go run ./tools/packet-audit operations --check → "operations check OK (0 absent-writer note(s))"
go test -count=1 ./libs/atlas-packet/parcel/... → ok (clientbound), ok (serverbound)
```

`grep -c PARCEL docs/packets/dispatcher-lint-baseline.yaml` → 0, and
`git diff 238faead6..cdaae842e -- docs/packets/dispatcher-lint-baseline.yaml`
is empty — the baseline was not touched, confirming PARCEL was not added
to dodge a violation.

Checked for residue from the reported `-ida-database` tool-bug abort:
`git show cdaae842e --stat` and the file list contain no temp/scratch
artifacts; the only touched paths are the fixtures, reports, evidence,
export splice, and STATUS.md/status.json. Consistent with a clean revert
before the raw-text splice, as claimed.

## Not evaluable

None. All five priority items and both flagged implementer claims were
independently re-derived from the committed diff, the dispatcher YAMLs,
and live gate-command runs — nothing in this batch's surface required
IDA-session access beyond what the committed export/report artifacts
already record.

## Verdict

APPROVED. No blocking or non-blocking findings.
