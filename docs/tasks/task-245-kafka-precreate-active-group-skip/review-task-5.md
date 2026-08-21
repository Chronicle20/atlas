# Review: Task 5 — Documentation (Job header comment and operator runbook)

Commit range: `d941053aa..1abf2e390` (single commit `1abf2e390`,
`docs(kafka-precreate): document re-sync skip semantics and the new Job log lines`)

## Scope

Two files changed:
- `deploy/k8s/base/atlas-kafka-precreate.yaml` (+15/-7, header comment only)
- `docs/runbooks/sparse-environments.md` (+50/-0, new subsection)

`deploy/k8s/overlays/pr-sparse/kustomization.yaml` — confirmed untouched
(empty diff).

This matches the brief (`task-5-brief.md`) exactly: documentation-only, no
shell logic and no Job spec change.

## Findings

### 1. Job header comment matches the brief's replacement text verbatim

`git diff` of `deploy/k8s/base/atlas-kafka-precreate.yaml` shows the comment
block at (now) lines 15-31 replaced with text that is a verbatim match to
the brief's Step 1 block (mechanism, FIRST-sync vs RE-SYNC framing,
`task-245` reference, ConfigMap-mount rationale). PASS.

### 2. No spec change — PRD NG2 (byte-identical fields)

Confirmed via targeted `grep` on both `HEAD` and pre-commit
(`d941053aa:deploy/k8s/base/atlas-kafka-precreate.yaml`):

- `backoffLimit: 3` — present, unchanged (yaml:45, pre-commit yaml:37)
- `ttlSecondsAfterFinished: 600` — present, unchanged (yaml:46, pre-commit yaml:38)
- `argocd.argoproj.io/sync-options: Force=true,Replace=true` — present, unchanged
- `argocd.argoproj.io/sync-wave: "0"` — present, unchanged

The `git diff --stat` shows a single hunk touching only the comment block
(15 insertions / 7 deletions, all in the header). No other lines in the file
changed — line numbers shift only because the comment grew. PASS.

### 3. `pr-sparse/kustomization.yaml` untouched — PRD NG4

`git diff d941053aa..1abf2e390 -- deploy/k8s/overlays/pr-sparse/kustomization.yaml`
returns empty. PASS.

### 4. Runbook subsection — log lines verified verbatim against `kafka-precreate.sh` at HEAD

Cross-checked every quoted log line in the new
`### Re-syncs: an active group is skipped, not re-seeded (task-245)` table
against `deploy/k8s/base/kafka-precreate.sh`:

| Runbook line | Script line | Match |
|---|---|---|
| `skipping group '<g>': already active (Stable) — offsets already initialized` | `kafka-precreate.sh:322` (`$group_current_state`) | The state token is a variable, not literally always `Stable`; but `atlas-kafka-precreate_test.sh:217` asserts exactly the string `"already active (Stable)"` as the tested/common case, so the runbook's literal example is grounded in the test, not invented. Not a defect. |
| `skipping group '<g>': reset refused, group became active during seeding — offsets already initialized` | `kafka-precreate.sh:314` | exact match |
| `skipped group '<g>': committed offsets present on all <N> topics` | `kafka-precreate.sh:411` | exact match |
| `WARN: skipped group '<g>' has no committed offset on <K> of <N> topics: <names> (+<M> more)` | `kafka-precreate.sh:413` (the `>10` variant) | exact match; the `<=10` variant (line 415, no `(+M more)` suffix) is not separately shown in the table, a minor incompleteness, not an inaccuracy — non-blocking |
| `override consumer group offsets seeded (<S> seeded, <K> skipped)` | `kafka-precreate.sh:329` | exact match |
| `all <K> override consumer groups were already active — nothing seeded this run (re-sync no-op)` | `kafka-precreate.sh:327` | exact match |

Also checked the fresh-environment paragraph's quoted `FAIL:` line against
`kafka-precreate.sh:393` (`FAIL: group '$group' has no committed offset on
topic '$topic'`) — exact match, and the claim that it's `exit 1` is correct
(`kafka-precreate.sh:394`).

`state_is_seedable`'s allowlist (`Empty|Dead|""` seedable, everything else
active — `kafka-precreate.sh:164-169`) matches the runbook's "Its groups are
`Empty`" framing for a fresh environment. PASS.

### 5. Test-coverage claim verified

The runbook's closing sentence — "the `state_is_seedable` truth table runs
without a broker, and the active-group skip and two-pass idempotence
assertions run when `BOOTSTRAP_SERVERS` is set" — is accurate:
- `atlas-kafka-precreate_test.sh:43-62` runs the `state_is_seedable`
  allowlist/denylist table with no `BOOTSTRAP_SERVERS` guard.
- `atlas-kafka-precreate_test.sh:64` gates the rest of the file behind
  `BOOTSTRAP_SERVERS`.
- `atlas-kafka-precreate_test.sh:211-232` is the active-group skip assertion
  (logs `already active (Stable)`, the re-sync no-op line, the WARN line).
- `atlas-kafka-precreate_test.sh:234-246` is the two-pass idempotence
  assertion (FR-5.1/FR-5.2: a second full pass exits 0 and moves no
  committed offset).
PASS.

### 6. Subsection placement

`grep -n "^#\{2,3\} " docs/runbooks/sparse-environments.md` confirms
`### Re-syncs: an active group is skipped, not re-seeded (task-245)` (line
160) sits directly between `## Verifying a consumer group is seeded (FR-4.9,
FR-5.3)` (line 125) and `### \`KAFKA_CONSUMER_GROUP\` must be resolved,
post-substitution group names` (line 221) — matching the brief's Step 4
expectation exactly. PASS.

### 7. No literal home/absolute paths; line endings preserved

`grep -n "/home/\|/Users/"` across both changed files returns nothing.
`grep -c $'\r'` on both files returns `0` (no CRLF introduced). PASS.

### 8. Commit message

`1abf2e390` — `docs(kafka-precreate): document re-sync skip semantics and
the new Job log lines`, matches the brief's Step 5 exactly. PASS.

## Not evaluable

None. This is a documentation-only unit and every claim in the new prose was
checkable directly against `kafka-precreate.sh` and
`atlas-kafka-precreate_test.sh` at HEAD within scope.

## Non-blocking notes

- The WARN-line table row only shows the `missing_total > 10` variant
  (`kafka-precreate.sh:413`, with the `(+<M> more)` suffix). The `<= 10`
  variant (`kafka-precreate.sh:415`, no suffix) is not shown as a separate
  row. This is a simplification, not an inaccuracy — the shown line is a
  real producible log line — but a future edit could add the second variant
  for completeness.

## Verdict

APPROVED. All documentation claims (Job header comment, runbook table log
lines, test-coverage sentence, placement) are grounded in the actual script
and test file at HEAD. The Job spec fields required to stay byte-identical
(PRD NG2) do. `pr-sparse/kustomization.yaml` (PRD NG4) is untouched. No
absolute/home paths, no line-ending drift.
