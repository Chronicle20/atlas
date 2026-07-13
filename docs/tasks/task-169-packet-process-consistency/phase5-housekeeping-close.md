# Phase 5 housekeeping close (T5.1 / FR-5.2)

DOCS-ONLY. Full gate + review + PR handled separately by the controller. Branch
`task-169-packet-process-consistency`; worktree confirmed before every edit.

## T5.1a — `verify-serverbound` decision: document + wire (NOT delete)

Confirmed the subcommand still exists (`tools/packet-audit/cmd/verify_serverbound.go`,
dispatched from `cmd/root.go:77`) and emits
`docs/packets/registry/verify_serverbound_<version>.md` — a send-site worklist
bucketed **Confirmed / Mismatch — REVIEW / Unresolved**. For each serverbound
registry entry it decompiles the client send function (address from the committed
audit reports for that version) and checks the registry opcode against the
`COutPacket::COutPacket` literal. Exit 0 regardless of bucket counts (worklist
generator, not a CI gate). Existing outputs: `verify_serverbound_{gms_v87,gms_v95,jms_v185}.md`.

Wired it in (thin pointers, canonical-doc style):

- **`IMPLEMENTING_A_PACKET.md`** — canonical explanation lives in the Step-1
  "Guards before you write any Go" block (opcode↔fname cross-check); the Step-4
  verify subsection is a thin pointer: a serverbound op must land in **Confirmed**
  before pinning, feeding the §9 three-artifact verification.
- **`STARTING_A_NEW_VERSION_PASS.md`** — canonical explanation in **§1.5** (with
  full flags); §3 "Promote cells" has a thin pointer (serverbound analogue of the
  `discover-ops` clientbound worklist → hand Confirmed rows to `packet-verifier`).

Flags documented (verified against `cmd/root.go` / `verify_serverbound.go`):
`--version` (req), `--registry-dir` (default `docs/packets/registry`),
`--audits-dir` (default `docs/packets/audits`), `--ida-url`, `--ida-port`,
`--out` (default `docs/packets/registry/verify_serverbound_<version>.md`).

## T5.1b — Maintenance re-audit playbook

Created **`docs/packets/RE_AUDITING_A_COLUMN.md`** — how to re-audit an EXISTING
version column after drift, framed around the three triggers (family-audit bug ·
export re-harvest · degraded matrix cell / hash drift). Documents each diagnostic
tool with one-line purpose + verified flags + when-to-reach-for-it:

- **`validate`** — baseline vs live IDB; buckets each entry (`divergent` /
  `missing-mode`). Flags: `--version`(req) `--report`(req) `--baseline`
  `--allowlist` `--descent-depth` `--ida-url` `--ida-port` `--ida-timeout`.
- **`decompose`** — extend baseline with live reads. Adds `--out`(req)
  `--audit-dir`; JMS `--audit-dir` quirk noted.
- **`triage`** — divergence worklist. `--version`(req) `--report`(req)
  `--baseline` `--audit-dir` + IDA flags.
- **`diff-shape`** — read-only shape diagnostic (material vs cosmetic arbiter).
  `--version`(req) `--report`(req) `--baseline` + IDA flags.
- **`infer`** — propose selectors, read-only (called out vs mutating
  `resolve-dispatch`, excluded as a bring-up tool). Adds `--out`(req)
  `--min-confidence` (default 0.6).

Hash-drift branch cross-links `STARTING §5.2` (degraded verified cells) for the
re-pin-vs-reverify decision and `VERIFYING §10` for the `export --splice` surgical
merge (never a full re-export overwrite). All flags verified against
`tools/packet-audit/cmd/root.go`; no invented flags.

Cross-links: added to `PROCESS.md` playbook index (Maintenance table →
RE_AUDITING_A_COLUMN.md) and from the `family-auditor` agent's Recommendations
section (points maintainers at RE_AUDITING trigger 1 for the fix after it reports).

## T5.1c — PROCESS.md consistency

Added the Maintenance table pointing at `RE_AUDITING_A_COLUMN.md`. The facts block
(9 versions / 7 CI gates) was **not** touched. All playbook links resolve.

## Concurrent-session collision (reconciled)

A concurrent session was editing the same files during this pass and converging on
the same design (it added STARTING §1.5, an inline §6 maintenance procedure, an
IMPLEMENTING Step-1 guard, and a competing PROCESS task-type row). My first commit
(`5825e0a2e`) swept its uncommitted work in alongside mine, creating duplication.
Reconciled to a single-source set: one canonical home per procedure + thin
pointers; the inline STARTING §6 procedure collapsed to a pointer at
`RE_AUDITING_A_COLUMN.md`; the competing PROCESS row removed in favor of one
Maintenance row. Final tree is internally consistent and deduplicated.

## Verification

- `doc-freshness --check` → **exit 0** ("PROCESS.md packet-process-facts agree
  with the tool (9 versions, 7 CI gates)") on the committed state.
- Working tree clean; branch `task-169-packet-process-consistency`.

## Commits

- `5825e0a2e` — wire verify-serverbound into IMPLEMENTING + STARTING (incl. swept concurrent edits).
- `dcfdebe58` — add RE_AUDITING_A_COLUMN + PROCESS index + family-auditor cross-link.
- `a3d65a3cb` — dedupe verify-serverbound + maintenance to single-source pointers.
