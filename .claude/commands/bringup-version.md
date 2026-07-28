---
description: Bring up a new client-version column in the packet coverage matrix — narrate-and-delegate through the STARTING_A_NEW_VERSION_PASS stages
argument-hint: <region> <major> <minor>  (e.g. GMS 92 1)
---

You are the orchestrator for adding a new client-version column to the packet
coverage matrix. This is the entry point task-113 lacked — it was
hand-orchestrated. You **narrate and delegate** (like `/execute-task`); you do
NOT do the whole pass inside one monolithic agent. A human stays in the loop
between stages.

Arguments: $ARGUMENTS → `<region> <major> <minor>` (e.g. `GMS 92 1`). Derive the
version key from region+major: `GMS 92` → `gms_v92`; `JMS 185` → `jms_v185`
(note the export/audit-dir naming quirks below).

**Canonical playbook: `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md`.** Read
it FIRST, in full. It owns every stage's exact command, flags, and the JMS
audit-dir quirk. Do not paraphrase or work from a remembered version. Also read
`docs/packets/PROCESS.md` (current version set, baseline status, CI gates) so you
know the column you're adding and the gates it must pass.

## The stages, in order (STARTING §1 → §3)

Drive these SERIALLY, one at a time, reporting the outcome and pausing for the
human before the next. Maintain a **ledger** (a scratch note in the task folder,
`docs/tasks/<task>/bringup-<version>-ledger.md`) recording each stage's status,
key artifacts, and any blocker — so the pass is RESUMABLE if interrupted.

1. **Registry seed** (STARTING §1.1 Step A) — `registry seed` from the CSVs; for
   a version the CSVs lack a column for, copy the nearest version's YAML and
   annotate the provenance.
2. **discover-ops** (STARTING §1.1 Step B) — run against the IDB, curate the
   dispatcher set per the include/exclude checklist, then re-run `--apply`.
   Resolve every collision (`provenance: manual` + IDA citation) before applying.
3. **Tenant template** (STARTING §1.2) — add
   `template_<region>_<major>_<minor>.json`; its routed opcodes must match the
   registry (disagreement becomes a 🟥 conflict).
4. **IDA export** (STARTING §1.3) — seed the roster from the nearest version's
   export, purge cross-IDB coincidentals, smoke-test a small roster, then run the
   full `export`.  **See export hygiene below — this is where passes go wrong.**
5. **Static audit pass** (STARTING §1.4) — run the static audit; then optionally
   `validate` / `decompose` / `triage` / `resolve-dispatch` against the live IDB,
   allowlisting genuine `missing-mode` cases into `_unimplemented.json`.
6. **Matrix wire-up** (STARTING §2) — `matrix` then `matrix --check`; the new
   column appears pre-filled (⬜ absent / ❌ present-unverified / 🟥 conflict).
   Resolve every conflict the pass introduces (or own it via §5.1) and commit
   registry + template + export + audit output + STATUS.md/status.json together.
7. **packet-verifier fan-out campaign** (STARTING §3) — promote ❌ cells hottest
   tier first. Dispatch the `packet-verifier` agent per cell (one packet ×
   version), coordinating via the discover-ops worklist. For a whole dispatcher
   family, dispatch `dispatcher-family-implementer` (do-mode) or `family-auditor`
   (read-only triage) instead. This stage is where the bulk of the work lives.

## SERIAL constraints — the campaign is NOT freely parallel

- **The IDA-MCP instance is a single global, single-threaded resource.** Stages
  2, 4, 5 and every `packet-verifier` that decompiles all drive the same IDA
  server. NEVER run two IDA-writing agents in parallel — dispatch verifiers one
  at a time (or strictly batched per IDB with `select_instance`), never as a
  concurrent fan-out. Confirm the loaded IDB matches the target version via
  `list_instances` before any decompile; do not hardcode ports (launch-order
  specific).
- **`run.go` `candidatesFromFName` and `evidence/families.yaml` are shared
  single files.** Two agents editing them concurrently corrupt each other. Any
  stage that touches them is serialized.

## Export hygiene — `VERIFYING_A_PACKET.md` §10 (non-negotiable)

**The export is NON-idempotent. NEVER re-run a full `export` over a committed
export to "refresh" it — it will churn hashes and drop hand-spliced entries.**
When a single fname is missing or stale, surgically SPLICE only that entry into
the committed export (absent-only merge), per §10. A full re-harvest is a
last-resort, task-081-playbook operation, not a routine stage step. An
unresolved fname is a STOP-and-ask, not an auto-re-export.

## JMS naming quirks (STARTING notes)

- Version key `jms_v185`, but the export file is `gms_jms_185.json`.
- `decompose` / `triage` default `--audit-dir docs/packets/audits/gms_jms_185`
  (which does not exist) — always pass `--audit-dir docs/packets/audits/jms_v185`
  explicitly for JMS.

## Done

The column is up when every cell in the declared scope is ✅, 🟡-with-evidence,
or ⬜, and `matrix --check` / `operations --check` / `fname-doc --check` /
`dispatcher-lint` all exit 0 on a committed tree (STARTING §4 task-close gate).
Report the per-stage ledger state and the four `--check` exit codes; surface any
conflict cell or unresolved fname as an explicit blocker.
