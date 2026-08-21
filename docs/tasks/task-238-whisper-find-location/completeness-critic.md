# Completeness critic — task-238-whisper-find-location

**Verdict: CLEAN.** No `coverage-manifest.yaml` exists in the task folder; audited
against the plan's Task 9 declaration (`docs/tasks/task-238-whisper-find-location/plan.md:2249-2331`)
instead. 0 CHANGED-BUT-UNCLAIMED findings, 0 CLAIMED-BUT-UNVERIFIED findings.

## Manifest status

`docs/tasks/task-238-whisper-find-location/coverage-manifest.yaml` does not exist
(`cat: ... No such file or directory`). Per the operating brief, this alone would
normally be the top finding requiring a stop — but this task's brief explicitly
redirects to auditing against the plan's Task 9 declaration when no manifest is
present, so the audit proceeded on that basis. Recommend the author add a
`coverage-manifest.yaml` for this task before merge, per the schema in
`docs/packets/PROCESS.md`, so future runs of this critic don't need the fallback.

Declared scope (plan.md Task 9, lines 2249-2331): promote
`field/clientbound/FieldWhisperError` × `gms_v92` from `incomplete` to `verified`,
with **no wire change** — `whisper.go` untouched, only `whisper_test.go` gains a
new test, plus the `gms_v92.json` ida-export, 8 evidence records, and regenerated
audit reports/status.json.

## CHANGED-BUT-UNCLAIMED

None found.

| kind | file-or-packet | evidence | recommendation |
|---|---|---|---|
| (none) | — | — | — |

Evidence checked:
- **Codec files**: `git diff --name-only $BASE...HEAD -- 'libs/atlas-packet'` returns only
  `libs/atlas-packet/field/clientbound/whisper_test.go`. Filtering to non-test `.go`
  files (`grep '\.go$' | grep -v '_test\.go$'`) returns nothing — zero codec source
  files changed in `libs/atlas-packet`. `git log --oneline $BASE..HEAD -- libs/atlas-packet/field/clientbound/whisper.go`
  is empty — `whisper.go` has zero commits touching it in range, confirming "no wire
  change" as declared.
- **Version gates**: `git diff $BASE...HEAD -- 'libs/atlas-packet' | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'`
  returns nothing — no gate lines added or removed anywhere under `libs/atlas-packet`.
- **Matrix delta**: `git diff $BASE...HEAD -- docs/packets/audits/status.json` shows
  exactly one state transition (ignoring the `toolSha` hash-only line):
  `"state": "incomplete"` / `"note": "tier-1 without fixture; verdict ❌"` →
  `"state": "verified"` at line 12523, which sits inside the
  `"packet": "field/clientbound/FieldWhisperError"` row (confirmed at line 12490)
  under the `gms_v92` cell (opcode 150). This is exactly the declared cell — no
  other row in `status.json` changed state. (`chat/serverbound/ChatWhisper` rows
  exist but were not touched.)

Two other diff-touching areas outside `libs/atlas-packet` and `docs/packets/`
(services/atlas-channel, services/atlas-maps, libs/atlas-constants) belong to
earlier plan tasks (1-8) on this branch, not Task 9's packet scope, and this
critic's brief scopes it to the packet-touching Task 9 work only, per the
dispatching instructions.

## CLAIMED-BUT-UNVERIFIED

None found.

| op | version | actual state | recommendation |
|---|---|---|---|
| (none) | — | — | — |

Evidence checked: HEAD `docs/packets/audits/status.json` line 12523-12524 shows
`field/clientbound/FieldWhisperError` × `gms_v92` at `"state": "verified"`. This
is the only claimed op×version pair in Task 9's declared scope, and status.json
carries only one row for the whole `CField::OnWhisper` shared-fname family
(`grep -n 'Whisper' docs/packets/audits/status.json | grep '"packet"'` returns
only `field/clientbound/FieldWhisperError` at line 12490 for the `field/`
namespace — the other 7 arms have no independent status.json rows, matching the
already-ruled note that the op grades worst-of-8-arms under one shared `fname`).

## Independently confirmed (already-ruled items)

- **8 distinct decompile_sha256 values**: `grep -H decompile_sha256 docs/packets/evidence/gms_v92/field.clientbound.FieldWhisper*.yaml`
  shows 8 files, 8 distinct sha256 values (verified via `sort -u | wc -l` = 8).
  Confirms the prior reviewer's finding.
- **whisper_test.go markers**: the 8 `packet-audit:verify` marker lines added
  (`git diff ... -- whisper_test.go | grep 'packet-audit:verify'`) exactly match
  the 8 arms enumerated in plan.md Step 3 (SendResult, Receive,
  FindResultCashShop, FindResultMap, FindResultChannel, FindResultError, Error,
  Weather), all pinned to `version=gms_v92 ida=0x53e2a0`.
- **gms_v92.json parse check** (plan Step 2): `python3 -c "..."` reproduces the
  plan's expected output exactly — `9 whisper entries`, `0 still unresolved`.
