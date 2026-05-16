# Context — Character-Domain Packet Audit (task-028)

Companion to `plan.md`. Captures the key files, decisions, and dependencies the implementer needs without re-reading the full PRD/design.

---

## Source artifacts (read these first)

- `docs/tasks/task-028-character-domain-audit/prd.md` — full requirements, §4.1 coverage matrix, §4.6 wire-bug fix rules, §10 acceptance criteria.
- `docs/tasks/task-028-character-domain-audit/design.md` — analyzer early-return fix (§3), TypeRegistry discipline (§4), phasing (§5), JMS divergence policy (§7), hot-path testing discipline (§8), sub-op limitation (§9), out-of-scope (§11), reference points (§12).
- `docs/tasks/task-027-atlas-packet-v95-audit/post-phase-b.md` — template for `post-phase-b.md` in Phase 4.
- `docs/tasks/task-027-atlas-packet-v95-audit/plan.md` — style anchor for plan structure.
- `docs/packets/audits/gms_v95/SUMMARY.md` — current login-audit verdict matrix. `CharacterList ❌` is the row Phase 0 must flip.
- `docs/packets/audits/gms_v95/CharacterList.md` — the existing false-positive report driving the analyzer fix.
- `docs/packets/MapleStory Ops - ClientBound.csv`, `... - ServerBound.csv` — FName ↔ opcode mappings; consumed by the audit CLI.

---

## Key existing code references

| Concern | File | Notes |
|---|---|---|
| Analyzer walker — `if` handling | `tools/packet-audit/internal/atlaspacket/analyzer.go:191-243` | The `*ast.IfStmt` arm where suffix-taint goes. Existing code pushes/pops a guard stack but does not model `return` as a control-flow exit. |
| Analyzer walker — `*ast.CallExpr` arm | `tools/packet-audit/internal/atlaspacket/analyzer.go:243-329` | Where `EncodeForeign` recurse-marker support lands. Existing `Encode || Decode` branch is at line 269. |
| Analyzer — `collectSub` (loop bodies) | `tools/packet-audit/internal/atlaspacket/analyzer.go:394-413` | Constructs a fresh `callCtx` per loop body. No changes required for the early-return fix (§3.3 — loop-internal early-return is out of scope). |
| Analyzer — guard helpers | `tools/packet-audit/internal/atlaspacket/analyzer.go:519-576` | `guardFromIf`, `conjoin`, `negate`. The `GuardExpr.text` field is unexported; the plan adds an exported `Text()` accessor in `guard.go` (Task 1). |
| Analyzer — `GuardExpr` type | `tools/packet-audit/internal/atlaspacket/guard.go` | Holds `eval` + `text`. Plan adds public `Text()` accessor. |
| TypeRegistry — pass 2 (method discovery) | `tools/packet-audit/internal/atlaspacket/registry.go:81-117` | Currently a switch on `Encode` and `Write`. Task 4 adds an `EncodeForeign` case that registers under the alternate key `<Type>::EncodeForeign`. |
| TypeRegistry — existing fixtures | `tools/packet-audit/internal/atlaspacket/registry_test.go` | Two fixtures (`TestRegistryFindsCharacterListEntry`, `TestRegistryFieldTypeStrips`). New fixtures in Tasks 4–6 mirror this shape. |
| Existing analyzer fixtures | `tools/packet-audit/internal/atlaspacket/testdata/simple_encode.go.txt` | Anchor format. Phase 0 fixtures (`early_return_then.go.txt` etc.) use the same shape. |
| Hot-path encoder — multi-version branching | `libs/atlas-packet/character/clientbound/spawn.go:60-148` | Canonical sample of the version-branching pattern Phase 2 audits. |
| Hot-path encoder — `AttackInfo` consumer | `libs/atlas-packet/character/clientbound/attack.go` | First Phase 2 packet to exercise the auto-discovered `AttackInfo` registration. |
| Hot-path encoder — `EncodeForeign` consumer | `libs/atlas-packet/character/clientbound/buff_give.go:16-96` | Both `BuffGive` and `BuffGiveForeign` embed `model.CharacterTemporaryStat`; spawn.go also uses `cts.EncodeForeign`. |
| Sub-struct sources | `libs/atlas-packet/model/{attack_info,pet,damage_taken_info,movement,character_temporary_stat}.go` | Registered (auto or via Task 4) — see plan §Phase 1 table. |
| Round-trip harness | `libs/atlas-packet/test/roundtrip.go:12-24` | Asserts `reader.Available() == 0` after decode. Every serverbound fix test uses this. |
| Tenant variants for tests | `libs/atlas-packet/test/context.go` | Source of `pt.GMSv83()` / `pt.GMSv87()` / `pt.GMSv95()` / `pt.JMSv185()` used by the 4-variant sweep. |
| Existing roundtrip test pattern | `libs/atlas-packet/login/clientbound/auth_success_test.go:9-37` | `for _, v := range pt.Variants { t.Run(v.Name, ...) }` — copy for new tests. |
| Templates ship | `services/atlas-configurations/seed-data/templates/template_gms_{12,83,87,92,95}_1.json`, `template_jms_185_1.json` | Six templates total. Character-domain template fixes touch the relevant subset. |
| IDA exports | `docs/packets/ida-exports/gms_v95.json` (login-only today), `gms_v83.json` (login-only today) | Phase 2 appends character entries to v95; Phases 3/15–17 create `gms_v87.json` and `gms_jms_185.json`. |
| `_pending.md` ledger | `docs/packets/ida-exports/_pending.md` | Existing login section + new "## Still pending — character domain" section appended in Phase 2 on first deferral. |

---

## Critical decisions locked in design

- **Early-return walker fix scope.** Function-scope `if … return; <suffix>` and `if … else … return; <suffix>` are in scope. Loop-internal early-return (`for … { if … return }`) is OUT (design §3.3). When detected, emit `🔍 unreachable code` and skip — do not extend the walker.
- **TypeRegistry registers only what character packets reference.** Predicted batch is 5 types (Phase 1 table). Mid-audit additions are cheap (one registry test). Do NOT pre-register the rest of `libs/atlas-packet/model/`.
- **`inventory/` package is out of scope** for sub-struct registry work. Spawn delivers its inventory snapshot via `cts.EncodeForeign` — covered by Task 4's registry change. PRD open question 4 resolved by inspection (design §4.2).
- **Bare handlers** (e.g. `create`, `check_name`) stay deferred to `_pending.md` per design §1 working assumption; no descent into atlas-channel or atlas-character service code.
- **v28 binary** is NOT pursued in this task (design §6). If one surfaces mid-task, defer to a sibling task.
- **JMS hard cap** — no atlas-packet encoder may grow beyond 2 nested region/version guards from this task's fixes. 3+ → log to `_pending.md` and stop (design §7).
- **Sub-op enum drift** is a known pipeline limitation; documented in `_pending.md`, NOT fixed in this task (design §9). Effect-family packets and `status_message.go` will likely trigger this; the deferral pattern is in Task 8 and Task 11.
- **No `reflect`, no new `interface{}` params, no benchmarks** (design §8). Encoder fixes change which `WriteX` runs under which guard; encoder structure stays as-is.
- **`clientVariant` flag stays where task-027 left it** — no schema changes in this task.
- **Tracking sub-tasks ≠ single PRs.** Each Phase 2/3 sub-task contains independent fix commits (one fix = one commit); the sub-task itself is the tracking unit, not a single commit/PR.

---

## Decisions deferred to execution time

- **Per-version audit-output directory layout.** v83/v87/JMS-185 reports go under `docs/packets/audits/gms_v83/`, `gms_v87/`, `jms_v185/` respectively — that's the assumption baked into Tasks 15–17 paths. The tool will create the directory; if reviewers prefer a flat layout, restructure on the first cross-version run, not later (consistency matters more than aesthetic).
- **Bucket commit cadence** — Tasks 7–14 may produce 1–6 fix commits before the bucket commit. Maintain the ordering: fixes first, audit-report bucket commit last, so the report shows the post-fix state.
- **Whether to widen or narrow a multi-version gate** — case-by-case, driven by IDA evidence. Default is "widen only when v83/v87/JMS-185 IDA confirms shared behaviour"; default is "narrow only when one of those versions diverges and atlas's current code is the wrong shape for it".
- **`_pending.md` section headings** — the file currently has a "## Still pending — login domain" heading. The first character-domain deferral creates "## Still pending — character domain" sibling heading. Sub-op enum drift gets its own "## Sub-op enum drift — character domain" heading on first hit (design §9).
- **Login regressions surfaced by analyzer fix** — Task 3 step 3 caps at 2 new ❌s. 3+ means STOP and split a sibling task; do not absorb into this plan.

---

## Workflow notes for the implementer

1. **Verify cwd before every commit.** Worktree is `.worktrees/task-028-character-domain-audit/`; branch is `task-028-character-domain-audit`. `git rev-parse --show-toplevel` must end with `/.worktrees/task-028-character-domain-audit`, `git branch --show-current` must be `task-028-character-domain-audit`. The auto-memory pin "Never commit directly to main" is load-bearing here — Phase 2 produces a lot of small commits and the discipline matters.
2. **Run `go test -race ./tools/packet-audit/...` before every analyzer/registry commit.** This is the canary for early-return regressions.
3. **Run `go test -race ./libs/atlas-packet/...` before every encoder commit.** A v95 fix that regresses v83 is the worst case; the 4-variant sweep catches it.
4. **Run `go vet ./libs/atlas-packet/...` and `go vet ./tools/packet-audit/...` before each commit.**
5. **Do NOT run `docker build` unless a service Dockerfile or `go.mod` changed.** Template seed-data JSON edits are visible to atlas-configurations at runtime and require no rebuild.
6. **The audit CLI never auto-mutates `.go` files.** Every encoder fix is a hand edit anchored to a freshly-generated audit report. Do not pipeline-rewrite encoders.
7. **IDA MCP needs the right binary loaded.** Phase 0 / Phase 1 / Phase 2 assume v95 IDA is open. Phase 3 requires the user to swap binaries explicitly (the plan calls this out at the top of each Phase 3 task). If `mcp__ida-pro__get_metadata` returns the wrong binary, ask the user before continuing.
8. **gitleaks risk.** task-027 had a follow-up to scrub `/home/<user>/` paths out of audit reports. Task 18 step 4 is the mandatory pre-PR scrub. Audit reports are generated by the tool; the tool sometimes embeds absolute paths from `--atlas-packet` — keep the flag relative (`libs/atlas-packet`) when invoking from worktree root and the issue stays absent.

---

## External dependencies / open questions resolved

- **IDA-MCP availability** — user-driven. Each Phase 3 task starts with a `get_metadata` check.
- **CSV path** — `docs/packets/MapleStory Ops - ClientBound.csv` and `... - ServerBound.csv` already exist on the branch (shipped with task-027). No new CSVs.
- **`inventory/` ownership** — resolved by inspection (design §4.2); not registered here.
- **v28 binary** — design §6 says no; defer to a sibling task if one surfaces.
- **JMS opcode-space divergence** — design §7.1 spells in-scope vs out-of-scope.
- **Sub-op enum drift in pipeline** — design §9 acknowledges the limitation; documented, not fixed.
- **Bare handler descent** — design §1 + PRD §3 non-goal; defer to `_pending.md`.

---

## Plan-execution checklist (one-glance summary)

- 18 tasks. Phase 0 = Tasks 1–3 (analyzer fix + login re-run). Phase 1 = Tasks 4–6 (registry extensions + fixtures). Phase 2 = Tasks 7–14 (character v95 audit, 8 buckets of ~6 packets). Phase 3 = Tasks 15–17 (cross-version v83/v87/JMS-185). Phase 4 = Task 18 (closeout).
- Hot-path bucket (Task 7) runs first by deliberate design — exercises the early-return fix and `EncodeForeign` registration on the riskiest packets before the cooler ones multiply mistakes.
- Every Phase 2/3 sub-task is a tracking unit. Plan checkboxes mark sub-tasks complete only when every bucket packet has a verdict row + every ❌ has a fix commit or `_pending.md` deferral.
- Final exit gate: Task 18 step 2 + step 4. `go test -race ./...` + `go vet ./...` clean; gitleaks scrub empty.
