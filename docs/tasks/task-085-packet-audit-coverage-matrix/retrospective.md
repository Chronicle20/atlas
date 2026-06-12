# Packet-Audit Workstream Retrospective (tasks 027–081)

Date: 2026-06-12
Scope: tasks 027, 028, 044, 065, 066, 067, 068, 069, 080, 081 — the packet-audit
workstream verifying opcode/mode/encoder-decoder integrity across GMS v83/v87/v95
and JMS v185.

## Headline

The tool never became a trustworthy oracle — it became an increasingly elaborate
triage organizer. Every wire fix that shipped (~30 across all domains) was
hand-confirmed against IDA before anyone believed it. The tool's genuine value was
structure: per-packet reports, cross-version tracking, regression detection. Its
verdicts were never load-bearing.

## State at closeout (post-task-081 four-version baseline)

| Version  | Rows | ✅  | ❌ | 🔍  | ⚠️ |
|----------|------|-----|----|-----|----|
| gms_v83  | 253  | 203 | 9  | 34  | 7  |
| gms_v87  | 248  | 215 | 3  | 24  | 6  |
| gms_v95  | 348  | 312 | 3  | 26  | 7  |
| jms_v185 | 217  | 176 | 6  | 32  | 3  |
| **Total**| 1066 | 906 (84.9%) | 21 | 116 (10.9%) | 23 |

Supporting machinery accreted to reach this: 205 persisted dispatch selectors,
258 missing-mode allowlist entries, 54 off-by-one baseline corrections, an
8-category acceptance taxonomy in `_pending.md`, and an 8-family opaque-type
ledger (`OPAQUE_LEDGER.md`).

## Root-cause findings

### 1. The flat positional diff cannot model the packets that matter most

`tools/packet-audit`'s diff engine compares two linear arrays of field
reads/writes. It has no model for client-side conditionals, Atlas-side
data-dependent branches, loop-body expansion, or mask-driven layouts. Every
workaround (wire-mutex collapse, trailing-buffer absorption, FlatInvalid
reclassification, dispatch selectors) patched a specific case without fixing the
class. Consequence: hot-path packets — character stat, spawn, inventory,
movement, all mask/mode-driven by nature — systematically fell out of automation
into 🔍, and their reports degraded into noise (e.g. `FieldSetField.md`: 85 of 99
rows flagged "atlas: extra" past the first sub-struct boundary).

### 2. IDA export quality was the real bottleneck

~174 of ~325 residual ❌/🔍 rows at task-080 closeout were attributed to export
read-order truncation/mistrace — bad hand-authored reference data, not Atlas
bugs. For BuddyInvite, three of the four version exports were wrong (mistraced
loop in v83/v87, truncation in JMS). Task-081 automated the exporter
(address-based helper descent, no-truncation, explicit unresolved markers), but
it was built against synthetic fixtures and required hardening when it met real
Hex-Rays output.

### 3. Closure rewarded triage, not verification

The `_pending.md` deferral taxonomy started as honest bookkeeping and became the
closure mechanism: a ❌ row gets a category label and prose justification, and
the task closes. "Zero actionable deferrals" measured triage completeness, not
audit completeness. Prose-only acceptance is unverifiable and silently rots.

### 4. Tool evolution never triggered re-audit

The tool improved every task (qualified registry keys in 065, deterministic
candidate selection in 068, exporter rewrite in 081), but earlier domains'
verdicts were never systematically re-run under the improved tool. The final
baseline mixes verdicts produced by four different tool generations with no
record of which.

### 5. Scope holes escaped to production

Three bugs landed after their areas were "audited":

| Bug | Commit | Hole |
|-----|--------|------|
| v87 stat-registry split (24 phantom stats) | 40ade184f | Cross-domain shared structure no single domain task owned |
| NPC continue-conversation discriminator hardcoding | 84bcafe07 | Handler logic outside frame-shape scope |
| Monster-book cover mob-id crash | d365ee580 | Semantics — right shape, wrong value |

The frame-level scope was a legitimate choice, but nothing tracked what the
audit *didn't* cover, so out-of-scope looked the same as verified.

### 6. Coverage state is not legible

No single artifact answers "for opcode X, direction Y, version Z — is this
verified, and on what evidence?" The state is smeared across four `SUMMARY.md`
files, `OPAQUE_LEDGER.md`, `_pending.md`, four `_unimplemented.json` allowlists,
and byte-tests scattered through `libs/atlas-packet`.

## What worked (keep)

- Per-packet markdown + JSON reports as the unit of record.
- The byte-level test discipline (4–5 variant byte-sweep tests per wire fix) —
  these are the only artifacts from the workstream that are still trustworthy
  without re-verification.
- Deterministic re-runs (after task-068) making report diffs reviewable.
- task-081's address-based helper descent and explicit `Unresolved` markers.
- The CSV opcode tables as the cross-version applicability source of truth.

## Decision

Task-085 (this task) implements the remediation: an evidence-graded coverage
matrix as the primary artifact, with a mandatory byte-fixture oracle for the
tier the flat diff cannot verify. See `design.md`.
