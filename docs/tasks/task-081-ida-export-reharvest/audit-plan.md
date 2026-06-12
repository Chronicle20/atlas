# Plan Audit — task-081-ida-export-reharvest (EXECUTED plan: V1–V7 + Phases 0/1/1.5)

**Plan Path (executed):** docs/tasks/task-081-ida-export-reharvest/plan-validation.md (+ Phase 0/1 from plan.md, Phase 1.5 from phase-1.5-real-decompile-hardening.md)
**Audit Date:** 2026-06-05
**Branch:** task-081-ida-export-reharvest
**Commit Range:** d0ae6e902..19f7275b5 (49 commits; HEAD = 19f7275b5, == branch tip)
**Module audited:** tools/packet-audit

## Executive Summary

The executed plan was followed faithfully. Phases 0, 1, and 1.5 are fully DONE; the
validation pivot (V1–V5) is fully DONE with non-vacuous TDD coverage; V6 is DONE as a
documented live proof (four versions validated, 330 shapes confirmed) but its *acted-on*
steps (commit dispatch annotations to baselines, commit per-version reports, fix
real-Atlas-bug divergences) were intentionally deferred; V7 is PARTIAL (results docs
written; ledger/guide/`_pending.md` updates not done). All deferrals are documented in
the proof docs. The original plan.md Phases 2–7 ("replace baselines + re-audit") were
abandoned with a committed empirical justification (26 ✅→❌ regression). Build, vet, and
`go test -race` are all clean on `tools/packet-audit`.

## Task Completion

### Phase 0 — rebase + verdict snapshot
| # | Task | Status | Evidence |
|---|------|--------|----------|
| P0 | Snapshot task-080 per-packet verdicts | DONE | commit c0e8e2a0d adds verdict-snapshot-080.md (1094 lines) |

### Phase 1 — build the exporter (11 TDD tasks)
| # | Task | Status | Evidence |
|---|------|--------|----------|
| P1 | Unresolved primitive + export schema marker | DONE | 302db858e; idasrc.go:21 `Unresolved` const, run.go renders 🚫 (6cf99b80e) |
| P1 | ParseDecompile linear/struct-descent/mode-switch/loop-guard/unresolved-fallback | DONE | 69b82c9ae, 93d837a6c, 44e243b63, 4a71c2930, f3195ebc0, 41f3bddd8 |
| P1 | MCPClient + real MCP-HTTP client + GetCallees/StructInfo | DONE | c06d2fdb0, 2a71c9bbe |
| P1 | Harvest descent driver (BFS, cycle/depth guard) + Delegate neutralize | DONE | 8d87df0ef, cb33e3039, b36e542ad, 0fbb69069 |
| P1 | runExport driver (roster, determinism, provenance, unresolved summary) | DONE | 8c8c0a5a1; cmd/export.go exportRun, FIXED provenance (no time.Now) |

### Phase 1.5 — real-decompile hardening
| # | Task | Status | Evidence |
|---|------|--------|----------|
| A | MCP client: session id, JSON address, soft-fail classification, get_callees arg | DONE | 81fbef698; mcphttp.go (Mcp-Session-Id, ErrToolSoftFail) |
| B | Harvest soft-fail → Unresolved entry, no abort | DONE | 8983af52e |
| C/D | Parser: alias SET, sub_XXXX descent, switch-expr, line prefixes, denylist | DONE | 631b37db8, 0d24e497e |
| E | Real BuddyInvite descent-chain regression | DONE | cdbffa5db; harvest_test.go real OnFriendResult→sub_A40028→Unresolved |
| +  | Direction-aware parsing (clientbound CInPacket / serverbound COutPacket) | DONE | 2e6c64adc |
| +  | New-server API re-map (lookup_funcs/decompile/callees, structuredContent, select_instance) | DONE | f1bf410b3; mcphttp.go:318 select_instance, :337 lookup_funcs |
| +  | 202/2xx tolerate on notifications/initialized | DONE | a095169f7; mcphttp.go:195 |
| +  | --ida-port multi-IDB instance selection | DONE | 0ec08cd38; root.go:80, NewMCPHTTPClientWithInstance |
| +  | C integer suffix (u/l) on switch case labels | DONE | a6380f107; parse.go:62 reCase `[uUlL]*` |

### Phase V-A — offline buildable core (V1–V4)
| # | Task | Status | Evidence |
|---|------|--------|----------|
| V1 | ExtractShape + Selector + guardSatisfies | DONE | 35c1a1b89; extract.go:11/20/51; TestExtractShapeOnFriendResultCase9, TestExtractShapeHexAndComposedGuards |
| V2 | `dispatch` schema field + ResolveShape | DONE | 57278bae1; export.go:42 `Dispatch []Selector`, :142 ResolveShape; TestResolveShapeUsesDispatch, TestExportFnParsesDispatch |
| V3 | InferDispatch auto-inference matcher | DONE | 9bd159e13; infer.go:17; +V3+ Unresolved-run scoring (10789ec52), +V3++ joint assignment (3f1a9210f) infer.go:111 InferDispatchJoint, +joint-aware confidence (83b96df3f) |
| V4 | validate command — ValidateShape + ResolveLive + report | DONE | 33f55308f (ValidateShape), c561b3b6b (ResolveLive/Entries), 14d2236a1+b950a799f (validate cmd), 53c87cab0 (un-isolable→unverifiable); shapediff.go, live.go, cmd/validate.go |

### Phase V-B — bootstrap selectors (live)
| # | Task | Status | Evidence |
|---|------|--------|----------|
| V5 | infer command + commit dispatch annotations to baselines | DONE (tool) / DEFERRED (committed annotations) | 8f95e6c14 infer command (cmd/infer.go inferRun); annotations applied to /tmp copies only — NO `"dispatch"` key in any committed docs/packets/ida-exports/*.json (intentional, see Gaps) |

### Phase V-C — validate + triage (live)
| # | Task | Status | Evidence |
|---|------|--------|----------|
| V6 | Run validation per version + triage divergences | DONE-as-proof / DEFERRED (triage + reports) | v83-validation-proof.md (23-entry annotated: 18 verified/3 divergent/2 unverifiable), four-version-validation-results.md (330 shapes confirmed across all 4 versions). No docs/packets/validation/*.md committed; no libs/atlas-packet fix landed (triage deferred) |

### Phase V-D — ledger + docs
| # | Task | Status | Evidence |
|---|------|--------|----------|
| V7 | Update STARTING_A_NEW_VERSION_PASS.md, _pending.md, final verify, code review | PARTIAL | Results docs written (four-version-validation-results.md). STARTING_A_NEW_VERSION_PASS.md NOT touched; _pending.md registries NOT updated to "live-verified"; this audit-plan.md is the plan-adherence review artifact |

**Completion Rate (executed plan):** Phases 0/1/1.5 + V1–V5(tool)+V6(proof) DONE; V5-commit + V6-triage + V7 deferred-documented. ~0 silent gaps.

## Abandonment of original plan.md Phases 2–7 — JUSTIFIED & DOCUMENTED

The original plan.md Phases 2–7 (build exporter → re-export four baselines → re-audit →
fix wire bugs by *replacing* the hand-authored baseline) were abandoned. This is NOT a
silent skip:
- design-validation-pivot.md §1 commits the empirical measurement: overlaying the
  exporter's flattened reads REGRESSES the audit **26 ✅→❌ vs 3 ❌→✅** (169✅→146✅),
  with documented structural root cause (whole-function flatten vs per-`#`-mode entries).
- The pivot to validate-not-replace is committed (design 8d04b324a, plan 0392a3362).
- Conclusion ("a fully-automated exporter cannot replace the hand-traced baseline; it can
  verify it") is consistent with the PRD's "genuinely verified" goal.
This abandonment is correct and well-evidenced.

## Deferred-but-Documented (not silent)

1. **V5 committed annotations** — dispatch selectors were applied only to /tmp baseline
   copies, not written back into docs/packets/ida-exports/*.json. Documented in
   v83-validation-proof.md ("Artifacts … in /tmp, not committed") and
   four-version-validation-results.md ("Per-version artifacts (in /tmp, not committed)").
   Rationale also documented: the confidence formula under-credits joint picks, so the
   maintainer chose not to commit low-confidence annotations.
2. **V6 triage + committed reports** — divergences were surfaced (6 high-confidence
   candidate findings across 4 versions) but not yet triaged into atlas-bug-fix /
   baseline-correction, and per-version reports live in /tmp. Documented in both proof
   docs' "follow-ups" / "next grind" sections.
3. **V7 ledger/guide/_pending** — STARTING_A_NEW_VERSION_PASS.md and the two `_pending.md`
   registries were not updated to mark `#`-entries live-verified.

## Gaps / Observations

- **MINOR (traceability):** The V5/V6/V7 deferrals are documented in the task's own proof
  docs but are NOT registered as a numbered follow-up task or a docs/TODO.md entry. The
  deferral is therefore visible to anyone reading the task folder, but not surfaced in the
  global backlog. Recommend registering a follow-up (verify the next number against
  `git log --all` per the task-numbers gotcha) covering: commit high-confidence dispatch
  annotations, triage the 6 high-confidence divergences, resolve demangled Class::Method
  helper names (recall lever), and the V7 ledger/guide updates.
- **No silent claimed-but-missing tasks found.** Every V-task with a "create file X"
  instruction has the file present with passing tests; every commit referenced in the
  plan's commit-message convention exists in the range.
- The `divergent`-heavy four-version table is honest noise (all joint picks applied,
  including low-confidence), explicitly explained in four-version-validation-results.md;
  the clean high-confidence-only run (precision 75–100%) is the trustworthy result.

## Build & Test Results

| Module | Build | Vet | Tests (`-race`) | Notes |
|--------|-------|-----|-----------------|-------|
| tools/packet-audit | PASS | PASS | PASS | All packages ok; idasrc + cmd carry full V1–V5 TDD coverage |

Test coverage is non-vacuous: ExtractShape (case9 + hex/composed guards), ResolveShape +
dispatch parse, InferDispatch (clear/ambiguous/Unresolved-wildcard/run-absorb/joint/
joint-confidence/short-entry), ValidateShape (verified/representation-equiv/divergent/
length/unverifiable/divergent-before-unresolved), ResolveLive (by-address + soft-fail),
validate cmd (report + undispatchable), infer cmd (proposes).

Redis-key-guard not applicable (no service go.mod or Redis usage touched; only the
standalone tools/packet-audit module). docker buildx bake not applicable (no service
go.mod changed).

## Overall Assessment

- **Plan Adherence (executed plan):** MOSTLY_COMPLETE — Phases 0/1/1.5 + V1–V5(tool) +
  V6(proof) fully delivered and verified; V5-commit/V6-triage/V7-ledger deferred with
  in-folder documentation.
- **Recommendation:** NEEDS_REVIEW — the toolchain and proof are merge-ready and the
  abandonment of Phases 2–7 is justified; the only open item is deciding whether the
  deferred V5-commit/V6-triage/V7 work blocks this PR or ships as a registered follow-up.

## Action Items

1. Register a numbered follow-up task (check `git log --all` for the next free number)
   capturing: commit high-confidence dispatch annotations to baselines, triage the 6
   high-confidence divergences, resolve demangled helper names for higher recall, and the
   V7 ledger (`STARTING_A_NEW_VERSION_PASS.md` + both `_pending.md`) updates.
2. (Optional, V7) Run `superpowers:requesting-code-review` (backend-guidelines-reviewer)
   over the Go changes before PR, per CLAUDE.md "Code Review Before PR".
