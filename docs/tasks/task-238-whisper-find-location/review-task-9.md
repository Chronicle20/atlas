# Review — Task 9: Promote the `gms_v92` clientbound WHISPER matrix cell

Range: `8ca62121b..17873fb17` (single commit `17873fb17`)
Brief: `.superpowers/sdd/plan/task-9-brief.md`
Report: `.superpowers/sdd/plan/task-9-report.md`

## Scope

`git diff --stat 8ca62121b..17873fb17` — 28 files, 505 insertions / 236
deletions, entirely under `docs/packets/**` and
`libs/atlas-packet/field/clientbound/whisper_test.go`. Matches the brief's
declared `### Files` list. No scope mismatch.

## Adjudication of the disclosed deviation (main task)

The implementer disclosed pinning evidence for all 8 WHISPER arms instead of
just `FieldWhisperError` (Step 6 named only the latter). Verified independently:

1. **Worst-of-8-arms grading via shared fname — confirmed.**
   `docs/packets/registry/gms_v92.yaml:790-793` (and `gms_v95.yaml:796-799`)
   declare a single clientbound WHISPER op with `fname: CField::OnWhisper` —
   no per-arm registry entries. `tools/packet-audit/internal/matrix/build.go:364-392`
   (`worstCandidateCell`) iterates every writer whose report shares that base
   FName, grades each independently via `gradeOpCell`, and keeps the worst
   `severity()` result (`build.go:387-388`). Doc comment at `build.go:349-355`
   states this explicitly for "demux families" like `CField::OnWhisper`. This
   is exactly the mechanism the implementer traced — confirmed, not asserted.

2. **`gms_v83`/`gms_v95` precedent — confirmed.** `ls
   docs/packets/evidence/gms_v83/` and `.../gms_v95/` each list all 8
   `field.clientbound.FieldWhisper*.yaml` files (Error, SendResult, Receive,
   FindResultMap, FindResultCashShop, FindResultChannel, FindResultError,
   Weather) plus `chat.serverbound.ChatWhisper.yaml` (a different op, not part
   of this family). `docs/packets/evidence/gms_v92/` now mirrors the 8
   clientbound files exactly (no ChatWhisper — correct, since serverbound
   `ChatWhisper` stays `incomplete` as the brief anticipated).

3. **The 7 added evidence records are genuinely derived, not copies asserting
   unsupported verification.** `tools/packet-audit/cmd/evidence.go:47` and
   `tools/packet-audit/internal/evidence/hash.go:14-38` show `evidence pin`
   computes `decompile_sha256` by canonical-JSON-hashing the specific
   `ida-exports` entry keyed by `--ida <fname>#<arm>`. Each of the 8 v92
   records carries a distinct hash (spot-checked `SendResult`
   `7f8c335b...`, `Error` `5981c418...`, `Receive` `8a2497f9...`,
   `FindResultCashShop` `441e408f...`, `Weather` `aae38875...` — all
   different, all tied to the arm-specific `calls` array the implementer
   hand-authored in `gms_v92.json` per Step 1). This is the same generation
   path the brief's own Step 6 used for `FieldWhisperError`, run 7 more
   times against real, distinct JSON content — not a hand-copied fixture
   claiming a read order that was never checked. `calls` arrays for all 8
   arms verified byte-identical to `gms_v95.json`'s (Python diff, all 8
   `True`), matching the brief's explicit instruction ("Copy the `calls`
   arrays verbatim from `gms_v95.json`").

**Verdict on the deviation: not scope creep, not fabricated evidence.** It is
a correct completion of what "promote the cell" requires given the registry's
actual worst-of-8 grading, verified against the tool's own source and the
sibling versions' file layout. Approving this as within-brief-intent.

## No-wire-change check

- `git diff 8ca62121b..17873fb17 -- libs/atlas-packet/field/clientbound/whisper.go` — 0 lines.
- `git diff --stat 8ca62121b..17873fb17 -- libs/atlas-packet/` — only
  `whisper_test.go` (+97/-0). No `Encode`/`Decode` body touched.
- `services/atlas-cashshop`, `services/atlas-mts`, `services/atlas-character`:
  zero diff each (checked individually — `git diff --stat` empty for all
  three).

## `gms_v92.json` arm records

`python3` census: 9 `OnWhisper` entries (8 arms + base), 0 `unresolved`. All 8
arms carry `address: "0x53e2a0"`, no stray `unresolved`/`notes` keys (only
`address`, `calls`, `direction`, `note` remain), notes reworded per-arm citing
`task-238` and `v83=v92=v95` invariance — matches the brief's Step 1 example
for `FindResultMap` and extends the same pattern correctly to the other 7.

## Byte test

`TestWhisperByteOutputV92` (`libs/atlas-packet/field/clientbound/whisper_test.go`,
appended after `TestWhisperByteOutputV48`) carries all 8
`packet-audit:verify` markers (`version=gms_v92 ida=0x53e2a0`), asserts real
encoded bytes for all 8 sub-modes including the parity-gated x/y case
(`NewWhisperFindResultMapWithXY` vs. the even-mode `0x48` case with no x/y).
Ran it directly:

```
$ cd libs/atlas-packet && go test ./field/clientbound/ -run TestWhisperByteOutputV92 -v
--- PASS: TestWhisperByteOutputV92 (0.00s)
```

This is a genuine byte fixture, not a tautology — it hand-encodes expected
byte sequences (including little-endian int32 x/y) independent of the codec
under test.

## Matrix check

Ran (read-only, non-mutating) `go run ./tools/packet-audit matrix --check`:
exit 0, only the two pre-existing "n-a evidence consumed" notes (unrelated,
CASHSHOP/TELEPORT_ROCK), no conflict/orphan/dangling/stale/drift finding.

Diffed `STATUS.md` and `status.json` for the full commit range: only the
`gms_v92` `exportHashes` entry and the single WHISPER op-row cell (`gms_v92`:
`incomplete`/❌ → `verified`/✅) changed, plus the v92 summary-row count
(51→52 verified, 5.8%→5.9%). No unrelated v92 cell moved. Target cell
confirmed directly from `status.json`:
`{'state': 'verified', 'opcode': 150}` for `field/clientbound/FieldWhisperError` × `gms_v92`.
Serverbound `chat/serverbound/ChatWhisper` × `gms_v92` unchanged at
`incomplete` (`'no audit report'` / `'tier-1 without fixture; verdict 🔍'`) —
expected, per brief.

## Regenerated audit reports

Spot-checked `docs/packets/audits/gms_v92/FieldWhisperFindResultMap.json`
diff: `Address` moved from `""` to `"0x53e2a0"`, all row `Verdict`s moved from
2/4 (extra-field / unresolved) to 0 (match), `Note` fields cleared. Consistent
with a genuine tool-regenerated report, not a hand-edit.

## Not evaluable

- The implementer's report notes `packet-audit -template ... -ida-source ...`
  emitted two pre-existing, unrelated errors (`SummonBagItemUse`,
  `ReturnScrollItemUse` method-not-found) during Step 5's report generation,
  and exited 1 before this reviewer could re-run it (re-running
  `-output` writers would touch files outside the read-only constraint given
  for this review). Taken on the implementer's word that these are
  pre-existing and unrelated to WHISPER; not independently reproduced here
  since doing so risks writing outside the permitted read-only surface. This
  is worth a one-line confirmation from whoever runs the next `verify.sh`
  pass, but does not block this cell's promotion.

## Verdict

APPROVED. The disclosed deviation is correct and independently verified
against the grading tool's source and the sibling-version file layout — it
is not scope creep, and none of the 7 additional evidence records assert an
unsupported verification claim.
