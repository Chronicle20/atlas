# Review — Task 1: The deterministic service-id derivation

Range reviewed: `5d1657989..4074a7f7a` (single commit `4074a7f7a`,
`feat(tools): add deterministic service-id derivation script`).

Files touched: `tools/derive-service-id.sh` (new, 42 lines),
`tools/derive-service-id_test.sh` (new, 108 lines). Matches the brief's file
list exactly — no scope drift.

## Requirement-by-requirement

### Namespace constant (D1, "never regenerate")

`tools/derive-service-id.sh:24-30`:

```sh
ATLAS_SERVICE_NS=c8f90111-a0cf-513e-95e6-c54609e5dec0
```

with the exact comment block required by the brief (never-regenerable,
re-derivable via `uuid5(NAMESPACE_DNS, "service-config.atlas.chronicle20")`).

- PASS — verified independently:
  `python3 -c "import uuid; print(uuid.uuid5(uuid.NAMESPACE_DNS,
  'service-config.atlas.chronicle20'))"` → `c8f90111-a0cf-513e-95e6-c54609e5dec0`,
  matching the hard-coded literal.
- PASS — grep across the whole worktree
  (`grep -rn c8f90111-a0cf-513e-95e6-c54609e5dec0 . --include='*.sh' --include='*.go'`)
  finds it in exactly this one file. Task 4's keeper rule (which is meant to
  cross-reference this constant) does not exist yet on this branch, which is
  expected — Task 4 is a separate unit, out of scope here. Nothing in this
  diff falsely claims that cross-reference exists.

### Derivation key is (service type, environment)

`tools/derive-service-id.sh:32-40` passes `"$SERVICE_TYPE/$ENVIRONMENT"` as
the uuid5 name, never a canonical/nil service id. Confirmed against the
§1.2-regression assertion in the test (four nil-UUID services →
four distinct derived ids) — this is the assertion that would fail if the key
were keyed on canonical id instead of type.

### Pinned values (all 10 rows of the brief's table)

Independently re-derived with `python3 -c` outside the test harness for two
spot rows (`login-service/pr-1411` → `6439ca9c-d28d-5db9-821b-8dd93d318a25`,
`channel-service/main` → `dff6f040-d4aa-51fa-914b-ff1dff6f6a76`) — both match
the pinned table and the script's own output. The full 10-row table is
asserted verbatim in `tools/derive-service-id_test.sh:24-33`. Ran the suite
directly:

```
$ sh tools/derive-service-id_test.sh
... (all 10 PASS lines, plus the four structural assertions)
ALL PASS
```

PASS.

### Structural test assertions (brief Step 1)

- §1.2 nil-UUID collision regression — `tools/derive-service-id_test.sh:36-48`,
  named exactly `FAIL: nil-UUID services collided (design §1.2)` as required.
  PASS.
- Determinism — `tools/derive-service-id_test.sh:51-53`. PASS.
- Environment sensitivity (`pr-1411` != `pr-1412`) —
  `tools/derive-service-id_test.sh:56-62`. PASS.
- Argument validation (zero args / one arg / empty-string second arg, each
  non-zero exit + non-empty stderr) — `tools/derive-service-id_test.sh:65-89`.
  Independently re-ran all three cases directly against the script outside
  the test harness; all three exit `2` with the usage line on stderr and
  empty stdout. PASS.

### Implementation correctness (brief Step 2)

- `set -eu` present (`tools/derive-service-id.sh:11`).
- Validation rejects `$# -lt 2` or either arg empty, usage on stderr, `exit 2`
  — matches the brief exactly (`tools/derive-service-id.sh:16-19`).
- `python3` absence guarded: `command -v python3` check with a named
  `FATAL:` message on stderr and `exit 1` before ever reaching the python3
  invocation (`tools/derive-service-id.sh:32-35`) — never falls through to
  empty stdout on that path, satisfying the brief's explicit anti-pattern
  callout (the `uuidgen`-swallowed-error defect analog).
- No trailing newline: `sh tools/derive-service-id.sh login-service pr-1411 |
  wc -c` → `36` (exactly the UUID's character count, no newline byte).
  Verified directly; not itself asserted by the test suite, but the brief's
  Step 1 table does not require a dedicated "no trailing newline" test
  assertion — it only requires this as an implementation property (Step 2),
  which is met. Non-blocking observation, not a finding: since Task 8/12
  consume this stdout, a `wc -c`/no-newline assertion would have been a
  nice-to-have regression guard, but its absence is not a violation of the
  brief as written.
- Emission uses the exact python3 heredoc from the brief, invoked as a
  subprocess with the namespace and `"type/environment"` string as
  positional args (`tools/derive-service-id.sh:37-40`) — matches verbatim.

### Placement / gating (Deviation 1, explicitly not to be flagged)

- Confirmed `tools/verify.sh:488` gates `touched '^tools/.*\.sh$'` with
  `./tools/shell-guard.sh --require-shellcheck`, and `changed_tool_suites()`
  (`tools/verify.sh:159-177`) maps `tools/derive-service-id.sh` →
  `tools/derive-service-id_test.sh` by the `${f%.sh}_test.sh` naming
  convention, so both new files are genuinely selected by the existing gate
  machinery, not merely runnable ad hoc. PASS — placement claim is real, not
  aspirational.
- `shellcheck -S error tools/derive-service-id.sh tools/derive-service-id_test.sh`
  exits 0 independently (re-run, not just trusting the report).

## Cross-service seams

None applicable — this is a leaf tool script with no producer/consumer
relationship established yet in this branch (Tasks 8 and 12, the declared
future callers, are separate units and correctly out of scope for this
review).

## Test honesty

Re-ran the full suite and several individual assertions independently
(pinned values via a fresh `python3 -c` invocation outside the test harness,
argument-validation cases via direct script invocation) rather than trusting
the implementer's report output. All independently reproduced. The pinned
values are not derivable by accident — an implementer using a different
namespace constant, wrong separator (e.g. `type-environment` instead of
`type/environment`), or a different serialization would fail at least one of
the 10 pinned rows; this is a real, load-bearing test, not one that would
pass regardless of implementation.

## Not evaluable

- Task 4's keeper-rule cross-reference to this constant does not exist on
  this branch yet; correctness of *that* cross-reference is out of scope for
  this review and will need to be checked when Task 4 lands.
- Task 8/12's actual consumption of this script's stdout contract (sed
  rendering, CI loop) is not yet implemented; this review can only confirm
  the contract as documented is met by the script, not that future callers
  will consume it correctly.

## Verdict

All brief requirements (D1, FR-1.3, FR-2.2) are met, all 10 pinned values are
independently reproduced, the four-distinct-ids regression is real, argument
validation matches the specified failure mode exactly, and the placement
deviation is confirmed genuinely gated rather than just claimed. No blocking
defects found.
