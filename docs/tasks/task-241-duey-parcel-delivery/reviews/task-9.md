# Task 9 review — PARCEL body-carrying arms (removed/arrived/alarms)

Scope: commit `707e97efc` (range `f1e7d0229..707e97efc`), read-only review
against `.superpowers/sdd/plan/task-9-brief.md` and
`docs/tasks/task-241-duey-parcel-delivery/task-9-report.md`. No edits made to
any tracked file; artifact only.

## Priority 1 — `ParcelRemovedKindClaimed = 0`

Independently decompiled `CParcelDlg::OnPacket` @0x6F56EA in IDA session
`41f09cce` (`MapleStory_dump.exe.i64`, matches the report's cited v83 IDB).
Case 23 body:

```
v18 = CInPacket::Decode4(v1);   /* parcelId, 0x6f59fd */
v19 = CInPacket::Decode1(v1);   /* kind, 0x6f5a06 */
...
if ( v19 == 3 )                 /* 0x6f5a62 */
  ... SP_3899_SUCCESSFULLY_DELETED ...
else
  ... SP_3900_SUCCESSFULLY_CLAIMED ...   /* 0x6f5a8e */
```

- **Confirmed**: the decompile genuinely has only one literal comparison
  (`== 3`). The "claimed" branch is a plain `else` with no second
  comparison, switch, or table — there is no missed pinned value. The
  report's transcript matches the live decompile verbatim.
- **Confirmed**: `parcel.go` documents `ParcelRemovedKindClaimed` as a
  *chosen* canonical non-3 value, not a decompile-cited literal — the doc
  comment (parcel.go:571–576) states plainly "there is no single
  decompile-cited literal for it — 0 is used as the canonical non-3 value."
  This is not silently presented as derived.
- **Plan check**: the brief itself only established the `!= 3` constraint
  and explicitly delegated the choice to the implementer ("not which value
  the server sends. Derive it..."); no value was pre-named in the plan, so
  there is no discrepancy to flag.
- **Family sanity check**: `0` is not used elsewhere in this dispatcher as
  an "unset"/sentinel sentinel that would make it a confusing choice here —
  the mode byte space uses distinct small integers (8–28) for arm
  selection, and `kind` is a wire byte scoped only to this one arm's body;
  the real client treats it purely as `== 3` vs `!= 3`, so any non-3 value
  is wire-equivalent. `0` is a reasonable, clearly-labelled choice.

No blocking finding here.

## Priority 2 — mode resolution, Ruling 5, append discipline

- `docs/packets/dispatchers/parcel.yaml` verified to carry, for all four
  keys, a `jms_v185` value that matches the report exactly:
  `PARCEL_REMOVED` → 24, `PARCEL_ARRIVED` → 25, `ALARM_NAMED` → 26,
  `ALARM_GENERIC` → 28 (parcel.yaml:82–86). GMS columns are 23/24/25/27 for
  all seven versions, also matching.
- `git show 707e97efc -- docs/packets/dispatchers/parcel.yaml` produces
  **no diff output** — confirmed empty, no yaml edits in this commit.
- Re-ran the greps myself: `grep -rn 'mode:\s*0x' …` and
  `grep -rn 'func(_ byte)' …` against both `parcel.go` and `parcel_body.go`
  — zero matches in both. No hard-coded mode byte anywhere in the new code.
- Spot-checked two arms end-to-end:
  - `PARCEL_REMOVED`: key const → `ParcelRemovedBody` resolves
    `ParcelOperationParcelRemoved` = `"PARCEL_REMOVED"` via
    `WithResolvedCode("operations", …)` → yaml gms_v83 = 23 (0x17) → test
    asserts `0x17 07 00 00 00 <kind>`, matches.
  - `ALARM_GENERIC`: key const → `ParcelAlarmGenericBody` resolves
    `"ALARM_GENERIC"` → yaml gms_v83 = 27 (0x1B) → test asserts
    `0x1B 0x01`, matches.
- `run.go` diff for the four new `#`-entries is a pure append (`git diff
  f1e7d0229..707e97efc -- tools/packet-audit/cmd/run.go` shows only `+21`
  lines, no `-` lines, inserted before the trailing `CSV: NOTE_ACTION`
  block). Confirmed by diff shape, not by re-reading final state.

No blocking finding here.

## Priority 3 — bodies, tests, no collateral damage

- `ParcelRemoved.Encode`: mode, `WriteInt(parcelId)` (little-endian, per
  `response.Writer.WriteInt`), `WriteByte(kind)` — order matches the
  decompile's Decode4-then-Decode1 read order. `Decode` round-trips the
  same three fields in the same order.
- `ParcelArrived.Encode`: mode + `parcel.Encode(...)` — uses the shared
  `parcel.Parcel` struct (Task 7), not a re-implemented layout, matching
  the brief. **Note**: `ParcelArrived` has no `Decode` method — this is not
  a Task 9 gap: `parcel.Parcel` itself (`libs/atlas-packet/parcel/parcel.go`)
  has no `Decode` method either (Encode-only, established in Task 7), and
  `Open` (also Task 7, also embeds `[]parcel.Parcel`) has the identical
  omission. Consistent with the existing family pattern, not a regression.
- `AlarmNamed.Encode`: mode, `WriteAsciiString(senderName)`,
  `WriteBool(hasItem)` — matches `DecodeStr`+`Decode1` read order from the
  decompile. Verified `response.Writer.WriteAsciiString` writes a 2-byte
  little-endian length prefix then the raw bytes, matching the test's
  literal `05 00` + `"Alice"` expectation exactly (read the writer source,
  not assumed).
- `AlarmGeneric.Encode`: mode + `WriteBool(hasItem)`, matches `Decode1`
  only.
- `parcel_notify_test.go`/`TestParcelNotifyArms`: all six subtests assert
  exact encoded byte slices (`bytes.Equal(got, want)`), not just that a
  constructor didn't panic — genuine wire-format tests.
- `TestParcelOpenQuickDecode`: constructs `OpenQuick` via `Encode`,
  round-trips through `Decode`, asserts `Mode() == 0x1A` and
  `reader.Available() == 0` — genuine new coverage for the Task-8 carried
  finding.
- Task 7/8 arms unmodified: `git diff f1e7d0229..707e97efc --
  libs/atlas-packet/parcel/clientbound/parcel.go` and `parcel_body.go` show
  **zero** `-` lines beyond the diff file headers — pure append confirmed
  by diff shape for both files, not just final-state inspection.
- Ran `go run ./tools/packet-audit dispatcher-lint` myself: output
  `dispatcher-lint: clean`, matches the report.
- Also independently ran `fname-doc --check` (`289 structs … carry no
  fname`, OK) and `operations --check` (8 pre-existing DueyAction
  writer-absent notes, all Task-10 scope) — both match the report's quoted
  output verbatim.
- `go build ./... && go test ./parcel/...` in `libs/atlas-packet`: passes,
  `clientbound` package `ok`.
- `matrix --check` re-run myself: exit 1, exactly the two expected stale
  lines (`STATUS.md is stale`, `status.json is stale`) plus two unrelated
  `n-a evidence consumed` notes for other packets — no ORPHAN, coverage, or
  drift complaint. Per Ruling 6 this is expected and not reported as a
  finding.

No blocking finding here.

## Not checked / out of scope for this review

- Did not verify versions other than gms_v83/jms_v185 by direct decompile
  (v72/79/84/87/92/95 mode-table entries were taken from the already-closed
  `parcel.yaml` header provenance comments, not re-decompiled in this
  session).
- Did not review `packet-audit:verify` marker absence — correctly assigned
  to Task 28 per the task instructions.
- Did not re-review the shared `parcel.Parcel` wire layout — closed per
  task instructions.
- Did not run the full repo-wide `tools/verify.sh` gate (out of scope for
  a per-task review; a separate gate is running concurrently against this
  worktree).

## Verdict

APPROVED — no blocking or non-blocking findings.
