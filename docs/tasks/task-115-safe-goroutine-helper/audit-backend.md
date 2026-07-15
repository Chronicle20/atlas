# Backend Audit — task-115 (safe goroutine helper)

- **Scope:** cross-cutting infrastructure change — new `libs/atlas-routine`, AST guard `tools/goroutineguard`, DOM-25 rule, and a blanket migration of every bare `go` statement under `services/`+`libs/` onto `routine.Go`.
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-*)
- **Date:** 2026-07-02
- **Build/Tests/Guards:** Controller-confirmed clean (goroutine-guard exit 0, redis-key-guard exit 0, 44 modules build + `-race`, `docker buildx bake all-go-services` exit 0). Not re-run here.
- **Overall:** PASS — zero blocking findings.

## Note on checklist applicability

This branch introduces no domain models, REST resources, processors, providers, administrators, or Kafka topics. It is a mechanical goroutine-wrapping migration plus one new leaf lib. The DOM-01..DOM-20, DOM-22..DOM-24, SUB-*, and EXT-* checks have **no changed surface to fail against** — no `model.go`/`resource.go`/`entity.go`/`rest.go`/`go.mod`-require/topic-config was added or altered in a way those checks target. The checks with live surface are DOM-21 (shared-constants reuse), DOM-25 (the rule this branch originates), and the immutability/processor-pattern guardrails on the touched `processor.go` files. Those are audited below.

## The helper — `libs/atlas-routine/routine.go`

| Check | Status | Evidence |
|---|---|---|
| Signature matches design (`l, ctx, fn(ctx)`) | PASS | routine.go:15 |
| ctx passed through unmodified, never inspected/cancelled | PASS | routine.go:24 (`fn(ctx)`); no ctx read in body |
| Recover installed before fn runs | PASS | routine.go:17-23 defer registered, then `fn(ctx)` at :24 |
| Panic logged Error-level with stable message | PASS | routine.go:21 `Errorf("Recovered panic in background goroutine.")` |
| Structured `panic` + `stack` fields; caller fields preserved | PASS | routine.go:19-20 (`fmt.Sprintf("%v", r)`, `debug.Stack()`); logs on the passed `l` |
| Panic swallowed, process continues; no leak beyond a bare `go` | PASS | recover returns nil-effect; no supervision added; routine_test.go:43-59 proves non-propagation |
| fn's own defers run before the helper recover | PASS | routine_test.go:83-97 pins the ordering the lock-leader migration depends on |
| Only dep is logrus; std-lib fmt/debug | PASS | routine.go:3-9 |

Design verdict: the recover/log/swallow shape is correct and safe. Stack is captured at recover time (standard idiom). No goroutine leak is introduced relative to the bare `go` it replaces.

## Migration correctness (spot-check of the hard cases)

| Site | Status | Evidence |
|---|---|---|
| kafka `manager.go` loop-var hoist + semaphore | PASS | manager.go:524 hoists `p := pm` before closure; `sem <- struct{}{}` acquire stays outside (:523-pre), `defer func(){ <-sem }()` release moved inside |
| kafka `manager.go:557` handler wg | PASS | `handlerWg.Add(1)` outside, `defer handlerWg.Done()` inside; `safeHandle` left untouched (§6.4) |
| lock `leader.go` completed-flag recovery | PASS | leader.go:157-168 — inner `if !completed` defer (reason+cancel) is LIFO-ordered before `close(fnDone)`, then helper recover logs+stack; semantics preserved, stack added |
| atlas-model `ExecuteForEachSlice/Map` wg | PASS | processor.go:157/165, 205/220 — `wg.Add` outside, `defer wg.Done()` inside; closer goroutine wraps `wg.Wait();close` |
| atlas-model `SliceMap` capture + no deadlock | PASS | processor.go:441-443 captures per-iteration `i,m` (Go 1.25 safe); `parallelTransform` defers `wg.Done()` at :474 so a worker panic cannot deadlock `wg.Wait()`; missing `resCh` send drained by `close(resCh)`+range (documented §6.1 silent-nil, accepted) |
| maps `respawn.go` capture | PASS | respawn.go:40-46 — original param-passed `mk` to dodge pre-1.22 capture; new closure captures per-iteration `mk`/`tctx`/`transactionId`, safe on Go 1.25 |
| monsters `processor.go:698` delayed effect | PASS | body byte-identical; `p.l`/`p.ctx` in scope |
| Signature plumbing: buffs `tasks.Register` curry | PASS | task.go:17-26 returns `func(Task)`; callers main.go:68,71 pass `l, tdm.Context()` |
| Signature plumbing: asset-expiration `NewPeriodicTask` ctx | PASS | periodic.go:30 adds `ctx`; caller main.go:47 passes `tdm.Context()` |
| Signature plumbing: atlas-data `l`/`ctx` thread | PASS | data/processor.go:225,234 use in-scope `l`/`ctx` from `RegisterAllData(l)(ctx)` |
| No `context.Background()` introduced inside a service | PASS | Background appears only in shared no-ctx libs (atlas-model SliceMap:441, atlas-redis coalesced.go:80) per design rule 3; service sites all use a real ctx (`tdm.Context()`, `p.ctx`, `tctx`, `sweepCtx`) |
| No leftover bare `go` in non-test code | PASS | sweep hits only routine.go:16 (exempt), testutil/helpers.go:190 (justified marker), and `tools/goroutineguard/testdata` (not swept) |

## DOM-21 — shared-constants reuse

PASS — no new domain type, alias, or numeric constant is introduced anywhere in the diff; `atlas-routine` declares only the `Go` function. Nothing to duplicate against `libs/atlas-constants/`.

## DOM-25 — the new rule (soundness)

PASS. The rule is defined consistently in three places and its mechanical check is authoritative:
- `.claude/skills/backend-dev-guidelines/SKILL.md:39`, `resources/anti-patterns.md:36`, `.claude/agents/backend-guidelines-reviewer.md:102`.
- The pass criterion delegates to `tools/goroutine-guard.sh` exit 0, which builds and runs an AST analyzer (`tools/goroutineguard/analyzer.go`) — comments/strings/`//go:` directives are structurally unmatchable, both spawn forms are the same `*ast.GoStmt`, and an empty-justification marker is itself a diagnostic (analyzer.go:57-63). Skip rules (routine pkg exempt, `_test.go` skipped, justified marker) match the design. The wrapper self-tests the analysistest fixtures before every run (goroutine-guard.sh) so the guard cannot rot into a no-op.
- The agent's grep in the DOM-25 "How to Verify" column (`^\s*go (func|[A-Za-z_])`) is only a triage aid, not the gate; the gate is the AST guard. Sound.

Guard/testdata coverage confirmed: `testdata/src/bad/bad.go` exercises both spawn forms and the empty-marker case; `good/good.go:19-22` exercises above-line and trailing justified markers.

## Locked design tradeoffs (adjudicated plan-mandated, NOT defects)

- atlas-model combinators + the 3 no-public-logger libs (atlas-service, atlas-redis, atlas-outbox) use `logrus.StandardLogger()` — design §6.1/§4.2.3, extended per the branch's stated scope. A recovered worker panic in `ExecuteForEachSlice`/`AwaitSlice` no longer reaches the error channel; the Error log is the detection path, and `defer wg.Done()` prevents any deadlock. Accepted design decision, verified non-deadlocking.

## Security (SEC-*)

Not applicable — no auth/token/redirect logic is touched. SEC-04: no hardcoded secrets introduced. The helper logs panic value + stack; that is inherent panic observability, not a secrets-leak defect.

## Non-blocking observations (informational, no change required)

- **Double-panic in the log call**: if the injected logger's hook itself panics inside the deferred recover, that second panic is unrecovered and would crash the goroutine — a theoretical edge shared by any panic-logging path, and not a regression versus a bare `go` (which had no logging at all). No action needed.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking
- Double-panic-in-logger theoretical edge (informational only).

Helper design and migration are sound: recover/log/swallow is correct, ctx is pure pass-through, stacks are captured, and every protocol-heavy site (kafka semaphore/wg, lock-leader reason/cancel, atlas-model wg + resCh drain) preserves its synchronization with defers correctly ordered before the helper's recover. Loop-variable captures are per-iteration-safe on Go 1.25. Guard and DOM-25 are wired end-to-end (analyzer + shell self-test + CI job + Dockerfile COPY + go.work).
