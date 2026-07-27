# Task 169 — Packet Process Consistency & Coverage Visibility

## Problem

Atlas reverse-engineers client↔server packet interaction (via IDA + a single-threaded MCP
server) and encodes it in `libs/atlas-packet`, supporting multiple game regions/versions.
Three task types trigger this process:

1. **Feature packet work** — a feature leveraging existing/undefined client-server
   interaction: identify packets, define codecs in the lib, gate fields per region/version.
2. **New region/version bring-up** — a full walkthrough of every supported version plus an
   IDB naming/audit for the new one.
3. **Packet-family audit** — usually bug-triggered; confirm a family's codec definitions are
   consistent with the IDB across every supported version.

The invariant end-state for every task: interesting IDB functions named correctly (GMS
v95.1 = source of truth); discrete atlas-packet codecs that both **encode and decode** and
work for **every** supported version (version-gated where the wire diverges); mode-prefix
("dispatcher") packets fully enumerated per arm; byte constants (modes, message types)
defined and **config-resolved** per tenant, never Go literals; tenant template files wiring
handlers + writers with the string↔byte constant mappings.

**The process falls down repeatedly, producing unnecessary code churn.** A review of
`.claude/`, `docs/packets/`, `tools/packet-audit/`, and the executed-task record
(task-027/080/085/096/100/113 + the task-085 retrospective) identified the root causes below.

## Root causes (evidence-backed)

**RC-A — Orphaned playbooks (structural).** Of five packet playbooks, only two have an
executable command/agent entry point: `VERIFYING_A_PACKET.md` (`/verify-packet` +
`packet-verifier`) and `DISPATCHER_FAMILY.md` (`dispatcher-family-implementer`).
`IMPLEMENTING_A_PACKET.md` (task type 1) and `STARTING_A_NEW_VERSION_PASS.md` (task type 2)
are procedurally orphaned — `/execute-task` runs generic subagent-driven-development with
zero packet awareness. Following the right recipe depends on a human remembering to open the
right doc. There is no read-only family-audit driver for task type 3 (the implementer only
runs in do-the-work mode).

**RC-B — Doc/tool drift (docs mislead agents).** Confirmed stale facts that actively
misdirect a run: docs say **5 versions, the tool has 9** (`matrix.VersionKeys`);
`dispatcher-lint-baseline.yaml` and `evidence/families.yaml` are **empty** (campaigns
graduated) but docs say they list party/guild/buddy; `matrix --check` is a **hard CI gate**
but docs describe "grandfathered via continue-on-error"; `tools/packet-audit/README.md`
`export` flags don't exist; the `🧩` family state is in no legend; `packet-verifier` cites
"steps 1–8" against a doc that runs 0–10. Procedures are also copy-pasted across 3–8 files
and have begun to diverge.

**RC-C — Unguarded churn classes.** CI hard-gates 3 of 8 recurring failure classes
(stale-opcode reshift, dispatcher mode-byte false-pass, dangling artifact). The three with
**no automated guard** are the recurring pain:
- **Off-by-one version gates** (`MajorVersion() > N` vs `>= N`; the v84/v87 boundary class).
- **Export non-idempotence** — re-running `export` drifts ~150 Hex-Rays function keys and
  degrades unrelated cells; today only a doc rule ("never overwrite; splice").
- **Semantic scope holes** — a bug ships in an area reported "audited" because nothing
  tracked what the audit did *not* cover.

**RC-D — Coverage visibility gaps.** The matrix (`STATUS.md`/`status.json`) is the de-facto
support ledger but is not surfaced from the top-level docs and is expressively lossy:
- **Sub-struct `n-a` is unrenderable** — `gradeSubStructCell` hardcodes `Present`, so a
  genuinely-inapplicable shared type is indistinguishable from "not yet audited" (`⬜`).
- **`🟡 partial` collapses three distinct meanings** into one symbol in STATUS.md.
- **No per-arm rollup** — a dispatcher is one row; per-arm coverage lives only in
  `dispatchers/*.yaml`, unrendered.
- **Inert safety nets** — `families.yaml` empty ⇒ the `🧩` cap governs nothing; a newly
  added dispatcher family would not be capped.

**RC-E — Tooling surface unmanaged.** 10 of 16 subcommands are doc-only; the diagnostic set
(`validate`/`decompose`/`triage`/`diff-shape`/`infer`) is documented only as new-version
steps (no "re-audit an existing column after export drift" playbook); `verify-serverbound`
is referenced nowhere; there is no one-shot "what is supported in version X" query.

## Goals

G1. Every one of the three task types has a documented, **executable** entry point that
    routes to the correct playbook and enforces the invariant end-state.
G2. Playbooks are single-sourced and true to the tooling; drift cannot silently reopen
    (a CI freshness lint fails when a doc's version count / baseline membership diverges from
    the tool's constants).
G3. The three unguarded churn classes (off-by-one gates, export clobber, scope holes) each
    gain an automated guard.
G4. Per-region/version support is legible: the matrix expresses sub-struct n-a, disambiguates
    partial, and there is a one-page per-version support summary + a `status <version>` query.
G5. The dispatcher/family safety nets are either re-armed or replaced by an equivalent check
    so a new family can't silently lose its cap.

## Non-goals

- No change to any *existing verified packet's wire output* — this task is process, tooling,
  docs, and matrix-rendering only. Any codec touched (e.g. the sub-struct n-a mechanism) must
  leave all 9 versions' verified counts unchanged except where a cell legitimately moves
  ⬜→n-a or a lossy 🟡 is disambiguated.
- No new region/version bring-up (that's a *use* of this process, not this task).
- No rewrite of the packet-audit export engine; export-clobber protection is a guard layer.

## Functional requirements

### FR-1 Executable entry points (RC-A, G1)
- **FR-1.1** A `packet-implementer` agent + `/implement-packet` command wrapping
  `IMPLEMENTING_A_PACKET.md`, owning the Step-0 "already implemented / shared-codec wrapper?"
  decision, ending at a `matrix --check`-clean verified state (hands to VERIFYING per cell).
- **FR-1.2** A **version-bring-up orchestrator** command driving the `STARTING` pipeline
  stage-by-stage (registry seed → discover-ops → template → export → static audit → matrix
  wire-up → packet-verifier fan-out), reusable and resumable. It must encode the serial
  constraints (shared run.go/families.yaml/global IDA) and the export-hygiene rules.
- **FR-1.3** A read-only `family-auditor` agent (task type 3): enumerates a family's arms
  from `dispatchers/*.yaml` + registry + matrix and reports verified/orphaned/unverified per
  version **without migrating**. Produces a findings doc; does not mutate codecs.
- **FR-1.4** `CLAUDE.md` and `docs/superpowers-integration.md` gain a "Packet work" section
  routing each task type to its entry point and naming the coverage matrix.

### FR-2 Single-source & de-drift docs (RC-B, G2)
- **FR-2.1** Command/agent files reference the canonical playbook rather than restating rules;
  remove the divergent copies (e.g. packet-verifier "steps 1–8").
- **FR-2.2** Fix every confirmed stale fact: 5→9 versions across IMPLEMENTING/STARTING; empty
  baselines in DISPATCHER_FAMILY; hard-gate semantics in VERIFYING §8 / STARTING §2; the
  `🧩` legend in VERIFYING/STARTING; README `export` flags. Name the v48/v61/v72/v79 columns.
- **FR-2.3** A CI doc-freshness lint (`packet-matrix.yml` or a sibling): fail when a
  playbook's asserted version count or baseline-family list diverges from the tool constants
  (`matrix.VersionKeys`, `dispatcher-lint-baseline.yaml`, `evidence/families.yaml`).

### FR-3 Guard the unguarded churn classes (RC-C, G3)
- **FR-3.1 Off-by-one:** a check that every version-gated wire field has a **boundary
  fixture pair** (highest-legacy + lowest-modern straddling the gate), driven from
  `code-gate-audit.md` or an equivalent gate registry; plus a lint nudging raw
  `MajorVersion() > N` → `MajorAtLeast(N)`.
- **FR-3.2 Export clobber:** make `export` non-destructive by default — write-temp + diff,
  refuse a full overwrite without `--force`, prefer splice-mode; encodes VERIFYING §10 as a
  tool guarantee.
- **FR-3.3 Scope holes:** a per-task **coverage manifest** (ops/versions/fields claimed) and
  a completeness-critic that diffs the manifest against touched code + matrix delta, flagging
  "changed but unclaimed" and "claimed but unverified."

### FR-4 Coverage visibility (RC-D, G4)
- **FR-4.1** Sub-struct cells can grade to `n-a` (real applicability signal), so a
  deliberately-inapplicable shared type is distinguishable from unaudited.
- **FR-4.2** STATUS.md disambiguates the three `🟡 partial` meanings (distinct symbol or
  annotation) without losing the status.json contract.
- **FR-4.3** A per-version support summary generated from `status.json`: supported ops,
  verified%, and gaps split into *n-a* vs *genuinely unverified* — one page per version.
- **FR-4.4** A `packet-audit status <version>` convenience printing a version's coverage +
  open gaps + stale evidence in one shot.
- **FR-4.5 (optional / stretch)** A per-arm coverage rollup for dispatcher rows.

### FR-5 Re-arm safety nets (RC-D/RC-E, G5)
- **FR-5.1** A check that every `dispatchers/*.yaml` family is represented in the capping
  mechanism (or an equivalent guard) so a new family cannot silently lose its cap.
- **FR-5.2** `verify-serverbound` is either wired into the serverbound flow + documented, or
  removed. A "re-audit an existing column after export drift" playbook covers the diagnostic
  toolkit for maintenance (not just new-version) use.

## Acceptance criteria

- AC-1 Each task type (1/2/3) is executable from a documented command/agent; a fresh
  contributor reaching CLAUDE.md can find the right entry point for each.
- AC-2 `go run ./tools/packet-audit matrix --check`, `operations --check`, `fname-doc --check`,
  `dispatcher-lint` all remain exit-0; the new doc-freshness + boundary-fixture + family-cap
  checks are green on the current tree and fail on a seeded violation (prove both directions).
- AC-3 No existing verified-cell count drops for any of the 9 versions except a documented,
  intended ⬜→n-a reclassification or a 🟡 disambiguation; `go test -race ./tools/packet-audit/...`
  and `./libs/atlas-packet/...` green.
- AC-4 The per-version support summary + `status <version>` query exist and match `status.json`.
- AC-5 `export` refuses a destructive full overwrite without `--force` (regression-tested).
- AC-6 Every doc/tool contradiction listed in RC-B is resolved and covered by the freshness
  lint where mechanically checkable.
- AC-7 Build/bake gate per CLAUDE.md for any touched Go module; redis-key-guard clean.

## Constraints / grounding

- Verification-Over-Memory and No-Deferring-Producible-Work apply. Any IDB claim (e.g. for a
  boundary fixture) is IDA-verified, never invented.
- Tool-behavior changes are TDD with a regression test proving the guard fires on a violation
  and passes on a clean tree (the "prove both directions" discipline the churn review found
  missing).
- Docs-as-code: prefer a mechanical freshness check over a hand-maintained assertion.
- Work entirely on the `task-169-packet-process-consistency` branch/worktree; PR at the end.

## Phasing (high level — detailed in plan.md)

- **P1 De-drift + single-source docs (FR-2)** — lowest-risk, stops immediate bleeding; also
  the reference baseline the later phases lint against.
- **P2 Matrix expressiveness + visibility (FR-4, FR-5.1)** — sub-struct n-a, partial
  disambiguation, support summary, `status` query, family-cap check.
- **P3 Executable entry points (FR-1)** — the three agents/commands + CLAUDE.md routing.
- **P4 Churn guards (FR-3)** — boundary-fixture pair, export non-destructive default,
  coverage manifest + completeness critic; doc-freshness lint (FR-2.3) lands here or P1.
- **P5 Tooling housekeeping (FR-5.2)** + full gate + review + PR.
