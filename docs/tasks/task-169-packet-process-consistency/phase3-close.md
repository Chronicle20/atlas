# Phase 3 close — Executable entry points (FR-1)

Phase 3 created the three missing executable entry points for the three packet
task types and wired top-level routing. Every entry point is a **thin pointer**
to its canonical playbook (post-P1 convention): it encodes the invariants and
the definition-of-done, and cites the playbook for the step detail rather than
paraphrasing it.

## Entry points created

### T3.1 `.claude/agents/packet-implementer.md` + T3.2 `.claude/commands/implement-packet.md`
**Wraps:** `docs/packets/IMPLEMENTING_A_PACKET.md` §0–4 (new-codec recipe).
**Enforces the invariant end-state:**
- **Step-0 owner** — decides "already implemented? / shared-codec wrapper?" before any
  Go (a serverbound ❌ is usually an unverified shared decoder like `model.AttackInfo`,
  not a missing codec; the agent links a thin per-op wrapper instead of duplicating).
- **Derive-from-IDB** — GMS v95.1 IDB is source of truth; distrust symbols; every field
  cites a decompile line; unresolved fname is a STOP-and-escalate (no auto-re-export/fake hash).
- **Encode AND Decode** — immutable struct (private fields, getters, no setters); `Decode`
  the exact field-for-field mirror of `Encode` (round-trip test proves symmetry,
  golden-byte test proves byte-exactness).
- **Version gates** — divergent fields branch inside `Encode`/`Decode` on the `MajorAtLeast`
  idiom, never raw `> N` (the documented v84 off-by-one); reuse `model/` sub-structs.
- **All 9 templates** — per-version opcode read per registry file; every `socket.handlers`
  entry carries a `validator` (silent-drop trap); version-absent → no route, no marker.
- **Config-resolved bytes (DOM-25)** — mode/message/sub-code bytes resolved from the tenant
  `operations`/`messageType` table at emit time, never a Go literal.
- **No existing-version wire change** — a genuine existing-version wire bug is its own prior
  commit, never smuggled in.
- **Hands each cell to `packet-verifier`.** DoD = `matrix --check` + `operations --check` +
  `fname-doc --check` + `dispatcher-lint` all exit 0.
The command mirrors `verify-packet.md` structure (frontmatter + arg parse + dispatch).

### T3.3 `.claude/commands/bringup-version.md`
**Wraps:** `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` (version-column pipeline).
A **narrate-and-delegate** orchestrator (like `/execute-task`, NOT a monolith agent) —
a human stays in the loop between stages. Args = `<region> <major> <minor>`. It drives the
stages in order: registry seed → discover-ops → template → export → static audit → matrix
wire-up → `packet-verifier` fan-out campaign. It encodes:
- **Serial constraints** — the IDA-MCP instance is a single global single-threaded resource;
  `run.go candidatesFromFName` and `evidence/families.yaml` are shared single files. Never
  run parallel IDA-writing agents; verifiers dispatch one-at-a-time / batched per IDB.
- **Export hygiene (`VERIFYING_A_PACKET.md` §10)** — the export is NON-idempotent; never
  re-run a full `export` to refresh; splice a single missing fname (absent-only merge).
- **Resumability** — a per-stage ledger (`docs/tasks/<task>/bringup-<version>-ledger.md`).
- **JMS quirks** — `jms_v185` key vs `gms_jms_185.json` export; explicit `--audit-dir`.
This is the entry point task-113 lacked (it was hand-orchestrated).

### T3.4 `.claude/agents/family-auditor.md`
**Wraps (audits against):** `docs/packets/DISPATCHER_FAMILY.md`. READ-ONLY coverage audit —
the bug-triage driver `dispatcher-family-implementer` lacks (that agent only runs in do-mode).
Given a family name it enumerates arms from `docs/packets/dispatchers/<family>.yaml`,
cross-references the registry + `status.json` + the tenant `operations` mode tables, and
reports **per-arm × per-version** verified / unverified / orphaned / n-a, plus every
operations-table mismatch. Writes `docs/tasks/<task>/family-audit-<name>.md` and recommends
follow-up. **MUST NOT mutate** codecs/registries/templates/evidence/run.go/STATUS — its only
write is the findings doc; it runs no `--apply`/`pin`/`export`/regeneration.

## T3.5 Routing (FR-1.4)
- **CLAUDE.md** — new "## Packet work" section: 3-row task-type → entry-point →
  canonical-playbook table + the leaf verify row, pointing at `docs/packets/PROCESS.md`.
- **docs/superpowers-integration.md** — same "Packet Work" section (relative links).
- **docs/packets/PROCESS.md** — flipped the three entry-point statuses from
  "planned (task-169 P3)" to "exists" (verify-packet already existed). The two remaining
  "planned" notes refer to the FR-2.3 freshness lint (a P4 item) and are intentionally left.

## Family-auditor dry-run sketch (validates T3.4 is not hollow)

Family `note_operation` (small, IDA-verified version-stable). Real inputs the agent works from:

- **`docs/packets/dispatchers/note_operation.yaml`** — writer `NoteOperation`, fname
  `CWvsContext::OnMemoResult`, op `NOTE`, clientbound. **4 arms** with mode bytes defined for
  5 versions only: `SHOW`=3, `SEND_SUCCESS`=4, `SEND_ERROR`=5, `REFRESH`=7 across
  gms_v83/v84/v87/v95/jms_v185 (header: switch cases {3,4,5,7} byte-identical v83 0xa2508b
  ↔ v95 0x9f9da0). Per-version writer opcodes: v83/v84/v87 `0x29`, v95 `0x28`, jms `0x26`.
- **`status.json`** — op `MEMO_RESULT` / `note/clientbound/NoteDisplay`, fname
  `CWvsContext::OnMemoResult`: v48 verified, v61 verified, v72 verified, v79 verified,
  v83 verified, **v84 incomplete**, v87 verified, v95 verified, **jms_v185 incomplete**.
- **Tenant templates** — `template_gms_83_1.json` `NoteOperation.options.operations` (the
  yaml header asserts the 3/4/5/7 modes match the seeded gms_83 table — the agent verifies it).

The generated `family-audit-note_operation.md` would contain:
1. **Header** — note_operation, `CWvsContext::OnMemoResult`, NOTE/MEMO_RESULT, clientbound, 4 arms.
2. **Per-arm × per-version table (4×9)** — v83/v87/v95 arms trace to the verified op-row;
   **v84 + jms_v185 flagged unverified** (op-row `incomplete`); v48/v61/v72/v79 have **no mode
   entry in the yaml** despite the matrix marking MEMO_RESULT verified there — flagged as a
   coverage divergence to investigate (is the pre-v83 NoteDisplay a non-dispatcher codec, or
   an unmodeled arm gap?).
3. **Operations-table cross-check** — yaml modes {3,4,5,7} vs each template's
   `NoteOperation.options.operations` per version; report clean/mismatch.
4. **Recommendations (no action)** — top cells: verify `note/clientbound/NoteDisplay ×
   gms_v84` and `× jms_v185`; reconcile the yaml's 5-version mode coverage against the
   matrix's 9-version MEMO_RESULT applicability.

This proves the agent has **real, machine-readable inputs** and produces an actionable,
non-hollow findings doc. (No findings doc committed — this is a sketch only.)

## Verification
- All four agent/command markdown files have well-formed `---` frontmatter and match the
  house style of `packet-verifier.md` / `dispatcher-family-implementer.md`.
- No restated procedure that duplicates a playbook — each entry point cites its canonical doc.
- No Go/tool/wire/matrix change this phase. `go run ./tools/packet-audit matrix --check`
  from the worktree root exits **0** (unchanged).
- No literal home/absolute paths in any committed file.

## Commits
- `87f4aa66b` T3.1+T3.2 packet-implementer agent + /implement-packet command
- `af6773927` T3.3 /bringup-version orchestrator
- `1605ea282` T3.4 family-auditor agent
- `558973166` T3.5 routing (CLAUDE.md + superpowers-integration + PROCESS.md status flip)
