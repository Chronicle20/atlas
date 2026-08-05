# Task 145 — Scope Amendment: gms_92 + jms_185 report support

**Decided:** 2026-08-04, by the user, at the start of the execute phase (option C).
**Supersedes:** `plan.md` Global Constraints "Scope exclusions" bullet and Task 25
Step 1's gms-92 / jms deferral entries.

## Why the plan's exclusion rationale was wrong

`plan.md` defers gms_92 and jms as *"blocked on registry files + IDBs (opcodes
unverifiable)"*. Verified against the live IDA-MCP sessions and the repo on
2026-08-04, that rationale does not hold:

- **gms_92** — the IDB (`GMS_v92_1_DEVM.exe.i64`) carries full mangled MSVC
  symbols for all five ops: `CWvsContext::OnClaimResult` `0x9cf310`,
  `OnSetClaimSvrAvailableTime` `0x9c5d30`, `OnClaimSvrStatusChanged` `0x9c5d60`,
  `OnSueCharacterResult` `0x9cf950`, `SendClaimRequest` `0x9d9c30`. Both ops CSVs
  already have a `GMS v92` column. Nothing is unverifiable; what is missing is the
  *column infrastructure* (registry yaml, IDA export, audit dir, `matrix.VersionKeys`
  entry).
- **jms_185** — is already a matrix column with a registry yaml carrying opcodes:
  `CLAIM_RESULT` `0x2A`, `CLAIM_AVAILABLE_TIME` `0x2B`, `CLAIM_STATUS_CHANGED`
  `0x2C` (all three also named in the JMS IDB), and `CLAIM_REQUEST` `0x65`
  (`provenance: csv-import`, **not** named in the IDB). Sue has no jms registry row
  and no named IDB function.

## Amended scope

Tasks 1–17 and 19–23c execute unchanged. The following are added or amended.

### New Task 26 — gms_92 registry

- `packet-audit registry seed` from both ops CSVs → `docs/packets/registry/gms_v92.yaml`.
- `discover-ops` against IDB session for `GMS_v92_1_DEVM.exe.i64`, with the
  dispatcher curation checklist from `STARTING_A_NEW_VERSION_PASS.md` §1.1, then
  `--apply`; worklist committed as `docs/packets/registry/discover_gms_v92.md`.
- **Tooling prerequisite:** `discover-ops` currently exposes only `-ida-port`, and
  port-based instance selection is dead (task-138). Add `-ida-database` to
  `discover_ops.go` (and `verify_serverbound.go`) mirroring `runExport`'s flag at
  `tools/packet-audit/cmd/root.go:133`. Produce this fix; do not work around it.

### New Task 27 — gms_92 IDA export

- Bootstrap the roster from `docs/packets/ida-exports/gms_v95.json` (nearest
  version; the v92 IDB was named from v95), purge cross-IDB coincidentals.
- `packet-audit export --version gms_v92 --ida-database <session> --output
  docs/packets/ida-exports/gms_v92.json`. Smoke-test a small roster first.

### New Task 28 — gms_92 matrix column

- Add `gms_v92` to `matrix.VersionKeys` (positioned between `gms_v87` and
  `gms_v95`) and `shortLabels` in `tools/packet-audit/internal/matrix/render.go:12`.
- Update the `doclint` facts doc (`version_count` 9 → 10, `version_keys`) and
  `docs/packets/PROCESS.md`'s version set.
- Run the static audit pass against `template_gms_92_1.json` → `docs/packets/audits/gms_v92/`.
- Regenerate the matrix; `matrix --check`, `fname-doc --check`, `doclint` clean.

### New Task 29 — jms_185 `CLAIM_REQUEST` send-site

- Name the jms serverbound claim send-site in the JMS IDB and `idb_save`, the same
  procedure as Task 23 does for v83. The registry's `0x65` is `csv-import` and
  unconfirmed — confirm or correct it from the IDB.
- **If no candidate decompiles to the expected shape, STOP AND ASK.** An
  unresolvable fname is never substituted or hashed (CLAUDE.md; `plan.md` Task 23).

### New Task 30 — jms sue absence

- Record sue-on-jms as a verified absence using the Task 23c evidence format, or —
  if a send-site/handler is found — a registry correction. Do not leave it implicit.

### Amended Task 18 — seed templates

Add sue/claim entries to `template_gms_92_1.json` (all five ops) and claim-only
entries to `template_jms_185_1.json` (sue omitted unless Task 30 finds it), at
their sorted `opCode` positions. `gms_12`, `gms_48` still get none; `gms_61` stays
sue-only. Opcodes come from the Task 26/29 registry work, never ported.

### Amended Task 24 — verification campaign

31 cells → **41**: +5 gms_92 (4 clientbound + `CLAIM_REQUEST`), +5 jms_185
(3 clientbound claim + `CLAIM_REQUEST` + whatever Task 30 resolves). jms
`SUE_CHARACTER_RESULT` promotes only if Task 30 finds it; otherwise it stays ⬜
with a recorded absence.

### Amended Task 25 — deferrals

Delete the gms-92 and jms deferral entries (now in scope). `gms_12` remains
deferred, with the **corrected** rationale: no registry yaml and no matrix column —
not "unverifiable". Do not restate the false IDB claim.

## Execution order

Tasks 1–17 → 26 → 27 → 28 → 29 → 30 → 18 → 19 → 20–23c → 24 → 25.

The bring-up lands before Task 18 so templates and the verification campaign see
`gms_v92` as a real column rather than needing a second pass.

## Risk accepted

A version bring-up on a feature branch is broader than this task's original
charter; it touches `packet-audit` internals and STATUS.md wholesale, so the
branch diff will be large and the final review correspondingly broad. This was
raised at decision time and accepted.
