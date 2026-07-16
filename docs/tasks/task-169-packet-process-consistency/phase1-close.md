# Task 169 — Phase 1 close (de-drift & single-source docs)

Checklist of every stale fact (RC-B) resolved in Phase 1, as the assertion set a
future doc-freshness lint (FR-2.3 / T4.5) can encode. Each row: the doc, the
stale claim (before), and the corrected claim (after), grounded against the tool
source cited.

## Ground-truth sources
- Version set = **9**: `matrix.VersionKeys` in
  `tools/packet-audit/internal/matrix/model.go:14` =
  `gms_v48,gms_v61,gms_v72,gms_v79,gms_v83,gms_v84,gms_v87,gms_v95,jms_v185`.
- Baselines **empty**: `docs/packets/dispatcher-lint-baseline.yaml` →
  `exempt_families: []`; `docs/packets/evidence/families.yaml` → `dispatchers:`
  list all commented (graduated).
- `matrix --check` **hard gate**: `.github/workflows/packet-matrix.yml` "Matrix
  check gate" step `exit 1` on non-zero; no `continue-on-error`.
- `export` flags: `runExport` in `tools/packet-audit/cmd/root.go:104-168` —
  `--version/--output/--ida-url/--ida-port/--ida-timeout/--descent-depth/`
  `--generated-at/--prior-export/--pending`. No `--ida-source/--csv/--template`.
- State symbols incl `🧩`: `State.Symbol()` `model.go:91-106`; render legend
  `render.go:32`.
- Subcommands = **15** + root pipeline: `cmd.Run` dispatch `root.go:25-70`.

## Facts fixed

| # | File | Before | After |
|---|---|---|---|
| 1 | `docs/packets/PROCESS.md` | (did not exist) | Created: index + `packet-process-facts` block (version_count 9, keys, empty baselines, 5 CI gates, `matrix_check_hard_gate: true`). |
| 2 | `docs/packets/IMPLEMENTING_A_PACKET.md` (Wire step) | "route the per-version opcode in all **five** seed templates" | "all **nine** seed templates" |
| 3 | `docs/packets/IMPLEMENTING_A_PACKET.md` §3 | "route the opcode in all **five** templates" | "all **nine** templates" |
| 4 | `docs/packets/IMPLEMENTING_A_PACKET.md` §3 heading + brace list | "Route in all **five** seed templates" / `template_{gms_83,gms_84,gms_87,gms_95,jms_185}_1.json` | "all **nine**" / `template_{gms_48,gms_61,gms_72,gms_79,gms_83,gms_84,gms_87,gms_95,jms_185}_1.json` |
| 5 | `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` baseline table | 5 rows (v83/v84/v87/v95/jms) | 9 rows (prepend v48/v61/v72/v79) |
| 6 | `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` `-versions` default | `"gms_v83,gms_v84,gms_v87,gms_v95,jms_v185"` | `"gms_v48,gms_v61,gms_v72,gms_v79,gms_v83,gms_v84,gms_v87,gms_v95,jms_v185"` (matches `strings.Join(matrix.VersionKeys,",")` at `cmd/matrix.go:56`) |
| 7 | `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` §2 | "Pre-existing 🟥 conflicts … grandfathered via `continue-on-error` in CI" | "hard, blocking CI gate … no `continue-on-error` … backlog burned to zero (task-085)" |
| 8 | `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` §3 | no cell-state legend; `🧩` absent | added `✅/🧩/🟡/❌/⬜/🟥` legend; `🧩` caps ops in `families.yaml` (currently empty → caps none) |
| 9 | `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` §3 | packet-verifier "follows the `VERIFYING_A_PACKET.md` **steps 1–8**" | "**§0–10**" |
| 10 | `docs/packets/DISPATCHER_FAMILY.md` baseline note | "baseline lists families not yet migrated (currently **party, guild, buddy**)" | "baseline is **empty** (`exempt_families: []`); all graduated (task-103/104/105); `families.yaml` also empty so `🧩` caps no op today" |
| 11 | `docs/packets/audits/VERIFYING_A_PACKET.md` §0 | "The **five** registry files" | "The **nine** registry files" (named) |
| 12 | `docs/packets/audits/VERIFYING_A_PACKET.md` (top) | no cell-state legend | added `✅/🧩/🟡/❌/⬜/🟥` legend |
| 13 | `docs/packets/audits/VERIFYING_A_PACKET.md` §8 | "until the registry-seed conflict backlog is burned down … `matrix --check` exits 1 from pre-existing 🟥 conflicts … Once conflicts reach zero, the bar becomes exit 0" | "hard, blocking CI gate … no grandfathering/`continue-on-error` … bar is a clean **exit 0**" |
| 14 | `tools/packet-audit/README.md` export invocation | `packet-audit export --ida-source mcp --csv-clientbound … --csv-serverbound … --template … --output …` (non-existent flags) | real `runExport` flags: `--version/--output/--ida-url/--ida-port/--ida-timeout/--descent-depth/--generated-at/--prior-export/--pending` |
| 15 | `tools/packet-audit/README.md` | no subcommand list | 15-subcommand table (matches `cmd.Run`) + root pipeline section |
| 16 | `.claude/commands/verify-packet.md` | restated "Non-negotiable rules 1–5" (incl. stale "exit 0 once seed-conflict backlog gone") | thin pointer: "follow `VERIFYING_A_PACKET.md` §0–10 verbatim"; keep args/output |
| 17 | `.claude/agents/packet-verifier.md` | "follow … literally, **steps 1–8**" + restated constraints 1–5 (incl. stale seed-backlog note) | "§0–10 verbatim"; keep role/inputs/outputs; `BLOCKED at §<n>` |
| 18 | `.claude/agents/dispatcher-family-implementer.md` | restated canonical-pattern steps 1–6 + constraints | pointer to `DISPATCHER_FAMILY.md` canonical pattern; keep def, unique stale-run.go-comment warning, DoD gates, report format |

## Deliberate non-changes (flagged)
- `docs/packets/IMPLEMENTING_A_PACKET.md:123` — the `MobCrcKeyChanged` example
  comment "IDA-verified, identical across all **5** versions" cites only 3
  addresses (v83/v87/v95). This is a **per-packet historical verification
  claim**, not a "current version set" statement; changing it to 9 would assert
  identity across v48/v61/v72/v79 that was never audited. Left as-is per the
  "do not change genuinely-historical statements" constraint. A future freshness
  lint should target PROCESS.md's `packet-process-facts` block, not prose
  example comments like this one.

## Gate P1 manual checklist
- [x] Every RC-B stale fact above resolved and grounded against source.
- [x] `docs/packets/PROCESS.md` exists with a parseable `packet-process-facts` block.
- [x] PROCESS.md relative links resolve (all four playbooks present).
- [x] No divergent step-count / rule copies remain in the `.claude` command/agent files.
- [x] Residual `5/five/grandfather/continue-on-error` hits are all intentional
      (new hard-gate language + the one flagged historical example).
