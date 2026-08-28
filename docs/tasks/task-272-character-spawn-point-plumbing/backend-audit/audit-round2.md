# Backend Audit — task-272 (round 2: DOM-01 builder-validation sweep)

- **Service Path(s):** services/atlas-cashshop, atlas-channel (untouched), atlas-character,
  atlas-consumables, atlas-dragons, atlas-login, atlas-messages, atlas-npc-shops,
  atlas-pets, atlas-query-aggregator
- **Range audited:** `61e5e4b94..HEAD` (`HEAD = e2b1723b8`), the nine builder-validation commits
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-27
- **Build:** PASS (all 9 modules)
- **Tests:** PASS (all 9 modules, `-count=1`)
- **Overall:** NEEDS-WORK

## Build & Test Results

`go build ./...` and `go test ./... -count=1` run per-module from each service's
`atlas.com/<module>` root.

| Module | Build | Test |
|---|---|---|
| atlas-cashshop | PASS | PASS (all packages `ok`) |
| atlas-consumables | PASS | PASS |
| atlas-dragons | PASS | PASS |
| atlas-login | PASS | PASS |
| atlas-messages | PASS | PASS |
| atlas-npc-shops | PASS | PASS |
| atlas-pets | PASS | PASS |
| atlas-query-aggregator | PASS | PASS |
| atlas-character | PASS | PASS (`atlas-character/pending_change` 206.9s — slow but green) |

No build or test failures in the audited range.

## Applicability

- **DOM structure (DOM-01..05, 11, 16):** Fired — every touched package has `model.go`
  (`character/model.go` in all nine services). Opened `file-responsibilities.md`.
- **FILE placement (FILE-01..06):** Fired — every changed Go package. No new file was
  added by this sweep (only `builder.go`, `model.go`, `processor.go`, `rest.go`, and
  test files were edited in place), so no new collapse risk.
- **SUB sub-domain:** N/A — no changed package has `resource.go` without `model.go`.
- **REST (DOM-06..09, 12..15, 17..19, 32):** Fired for the `rest.go`/`processor.go`
  touches (dragons `rest.go`, login `rest.go`/`processor.go`, cashshop/npc-shops/
  query-aggregator `processor.go`). Opened `patterns-rest-jsonapi.md` for DOM-09.
- **Constants reuse (DOM-21):** N/A — diff declares no new type, const block, or
  numeric-literal classification; it only changes `Build()` signatures and error text.
- **Testing (DOM-10, 20, 24, 33):** Fired — diff touches many `_test.go` files and
  the query-aggregator `mock/processor.go` implements `character.ProcessorImpl`'s
  `GetById`. DOM-24 fired for atlas-cashshop/atlas-dragons/atlas-pets test files that
  reach an emit path, but the diff's edits in those files are confined to
  `Build()` call-site error handling, not producer wiring — see "Not evaluable."
- **Cache (DOM-29):** N/A — no changed package has `cache.go`; no processor gained
  cached state in this diff.
- **Messaging (DOM-30):** N/A — no new `AndEmit`/`producer.ProviderImpl` call site
  was added by this diff (pre-existing emit paths in touched test files are
  untouched by the sweep's edits).
- **Multi-tenancy (DOM-31):** N/A — no `rest.go` gained a new tenant/trace field;
  the sweep changes `Build()` return shape only.
- **Migration hygiene (DOM-34/35):** N/A — no symbol moved between a service and
  `libs/atlas-*`.
- **Deploy & topics (DOM-22/23):** N/A — no `libs/` module or Kafka topic env var added.
- **Runtime safety (DOM-26):** Fired (any non-test Go file changed) — no `go`
  statement appears anywhere in the nine `builder.go` diffs or their call-site
  edits; N/A per rule's own trigger.
- **Channel wire values (DOM-25):** N/A — diff does not touch `services/atlas-channel`
  or `libs/atlas-packet`, and none of these character-model `Build()` changes
  carries a client-interpreted byte.
- **Resilience (DOM-27, 28):** Fired — `model.Decorator[Model]` implementations
  (`InventoryDecorator`, `GuildDecorator`, `SkillModelDecorator`) changed in six of
  the nine services to call the now-fallible `Build()`. Opened `patterns-resilience.md`.
  DOM-27 N/A — no handler in this diff writes `http.StatusInternalServerError`.
- **External clients (EXT-01..04):** N/A — no new `requests.*Request[T]` call site added.
- **Scaffolding (SCAFFOLD-*):** N/A — no new service directory, channel writer/handler,
  or `routes.conf` change.
- **Security (SEC-*):** N/A — none of the nine services in scope handle
  authentication/tokens/redirects in the touched files.
- **Foundational — patterns-provider.md:** N/A — no provider defined or composed
  in this diff (only `Extract`/`Build` calls, which are covered by DOM-01/DOM-11).
- **Foundational — patterns-functional.md:** Fired — `model.Decorator[Model]`
  composition is exactly this document's subject. Opened; its guidance is
  subsumed by the DOM-28 finding below (no separate foundational-only finding).

## Checklist Results

### character (domain package, all nine services)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` has `NewBuilder()`, fluent setters, validating `Build()` | PASS | `services/atlas-cashshop/atlas.com/cashshop/character/builder.go:121-155` (`id == 0` guard); `services/atlas-consumables/atlas.com/consumables/character/builder.go:125`; `services/atlas-dragons/atlas.com/dragons/character/builder.go:26-28`; `services/atlas-login/atlas.com/login/character/builder.go:94`; `services/atlas-messages/atlas.com/messages/character/builder.go:117`; `services/atlas-npc-shops/atlas.com/npc/character/builder.go:124`; `services/atlas-pets/atlas.com/pets/character/builder.go:119`; `services/atlas-query-aggregator/atlas.com/query-aggregator/character/builder.go:134`; `services/atlas-character/atlas.com/character/character/builder.go:63-68` (`accountId == 0` / `name == ""`). The prior round's blocking finding against `atlas-npc-shops` (non-validating `Build() Model`) is cleared: `services/atlas-npc-shops/atlas.com/npc/character/builder.go:124` now returns `(Model, error)` and rejects `id == 0`. |
| DOM-01 (judgment item) | Same, for `modelBuilder`/`NewEmptyBuilder()` | FINDING (non-blocking, deferred) | `services/atlas-character/atlas.com/character/character/builder.go:340-341` — `func (c *modelBuilder) Build() Model` remains non-validating, confirmed unchanged by this diff (`git diff --stat` for `builder.go` touches only lines 2-95). This is a genuine DOM-01 gap on a `builder.go` file with `model.go` in the same package — the rule's trigger (`package has model.go`) does not distinguish between a package's two builder types. The task doc's own rationale (fixed-shape hydration of partial test/read models, ~40 call sites, `character/hp_mp_gain_test.go:55` sets no name) is a real justification for a *different* invariant, not for *no* invariant, and DOM-01 does not carve out an exception for "used by tests." Recommendation: apply the identity invariant only where `modelBuilder` is used to *construct* a persisted-shape model (e.g. reject `accountId == 0` at the `NewBuilder`-equivalent creation path) while leaving `CloneModel`/partial-hydration paths alone — a design decision for a follow-up task, not a mechanical fix on this branch. Recorded as a finding per the audit brief's explicit instruction not to treat the doc as already-settling this. |
| DOM-02/DOM-03 | `Model.ToEntity()` / `Make(Entity)` in `entity.go` | N/A | None of the nine `character` packages has an `entity.go` — grep for `entity.go` in each touched dir returns nothing; these are REST/read-model packages, not GORM-backed. |
| DOM-04/05 | `Transform`/`TransformSlice` in `rest.go` | PASS (unaffected) | `services/atlas-dragons/atlas.com/dragons/character/rest.go:34` `Transform` unchanged by this diff; `services/atlas-login/atlas.com/login/character/rest.go` unchanged in this respect — the diff only touches the `Extract`-side `Build()` call, not `Transform`. |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS (unaffected) | All nine `NewProcessor(l logrus.FieldLogger, ctx context.Context)` signatures unchanged by this diff, e.g. `services/atlas-cashshop/atlas.com/cashshop/character/processor.go:28`. |
| DOM-09 | Every `Transform(` call site checks its error | N/A | No changed `resource.go` in this diff (no `resource.go` file appears in the diff's file list) — trigger did not fire. |
| DOM-11 | Providers evaluate lazily via `database.Query`/`SliceQuery` | N/A | None of the nine `character` packages has a `provider.go`; these are REST-client read models using `requests.Provider`/`requests.SliceProvider`, not DB-backed providers — outside DOM-11's trigger. |
| DOM-28 | Fallible decorator paths degrade loudly (`model.ErrDecorator` + `degrade.Observe`), never bare `if err != nil { return m }` | **FAIL** (atlas-cashshop, atlas-npc-shops, atlas-query-aggregator) | `services/atlas-cashshop/atlas.com/cashshop/character/processor.go:50-56` — `InventoryDecorator` binds `err` from `m.SetInventory(i)` and returns `m` on failure with no log and no metric, though `p.l logrus.FieldLogger` is in scope on the same struct (`processor.go:22`). `services/atlas-npc-shops/atlas.com/npc/character/processor.go:72-79` — identical pattern, `p.l` in scope at `processor.go:26`. `services/atlas-query-aggregator/atlas.com/query-aggregator/character/processor.go:48-56` (`InventoryDecorator`) and `:58-66` (`GuildDecorator`) — same pattern twice, `p.l` in scope at `processor.go:21`. This also violates the task's own controller ruling in `docs/tasks/task-272-character-spawn-point-plumbing/builder-validation.md` ("Never `_`... Log it. If a `logrus.FieldLogger` is in scope, `l.WithError(err).Errorf(...)`..."), which these three services do not satisfy — `err` is bound but neither logged nor commented. |
| DOM-28 | Same | **FAIL** (atlas-messages) | `services/atlas-messages/atlas.com/messages/character/model.go:237-243` — `Model.SetSkills` swallows the `Build()` error with no logger in scope (it's a value-receiver `Model` method with no logger field) and, unlike its sibling implementations, **no explanatory comment**. Compare `services/atlas-consumables/atlas.com/consumables/character/model.go:292-301` and `services/atlas-pets/atlas.com/pets/character/model.go:254-262`, which both carry a comment establishing why the branch is unreachable in practice — the second, documented branch of the controller's own ruling. `atlas-messages/character/model.go:237-243` has neither branch satisfied. |
| DOM-28 | Same | WARN (atlas-consumables, atlas-pets) | `services/atlas-consumables/atlas.com/consumables/character/model.go:275-278` (`SetInventory`) and `:303-310` (`SetPets`), `services/atlas-pets/atlas.com/pets/character/model.go:254-262` (`SetInventory`) — no logger in scope at the `Model` method level, but each carries an explanatory comment ("Unreachable in practice: `Clone(m)` carries forward `m.id`...") satisfying the controller ruling's documented fallback. Rated WARN rather than PASS because DOM-28's own text ("regardless of justification") does not itself carve out a comment-only exception — the comment satisfies the task ruling, not the numbered rule, and the two are in tension; recorded as non-blocking pending a ruling on which document governs. |
| DOM-28 | Same | PASS (atlas-login) | `services/atlas-login/atlas.com/login/character/processor.go:183-195` — `InventoryDecorator()` returns `model.ErrDecorator(func(m Model) (Model, error) {... return m.SetInventory(i) }, func(m Model, err error) { degrade.Observe(p.l, "login.character.inventory", m.Id(), err) })`. Fully DOM-28-compliant, and demonstrates the `model.Decorator[Model]` signature constraint the other five services cite is not actually a hard blocker — `ErrDecorator` already satisfies `model.Decorator[Model]` while degrading loudly. This file was touched only for the `MergeRankings`/`decorateRankings` signature change (`processor.go:95-148`); the `InventoryDecorator` body at line 187 is pre-existing and unmodified by this sweep. |
| Call-site rule (builder-validation.md, this task's own decision) | Every `.Build()` call site propagates or is a documented decorator exception | PASS (all direct propagation sites) | No `m, _ := ...Build()` or `_, _ := ...Build()` found across the diff (`grep -n "Build()" ... | grep -E "_, _|, _ ="` returns empty for every file). All `Extract`/`SetInventory`/`SetGuild`/`MergeRankings` sites that already return `(Model, error)` bind and propagate the `Build()` error, e.g. `services/atlas-dragons/atlas.com/dragons/character/rest.go:31`, `services/atlas-cashshop/atlas.com/cashshop/character/model.go:304`, `services/atlas-npc-shops/atlas.com/npc/character/model.go:251,255`, `services/atlas-query-aggregator/atlas.com/query-aggregator/character/model.go:318,322`. Test call sites use `t.Fatalf`/`assert.NoError`/`require.NoError` immediately after `Build()`, e.g. `services/atlas-login/atlas.com/login/character/builder_test.go:11-13`, `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go:490-491`. |

### atlas-character/character (domain package, deep-dive)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | Validating `Build()` | PASS (for `Builder`) / FINDING (for `modelBuilder`, see above) | `services/atlas-character/atlas.com/character/character/builder.go:63-68`. |
| Scope fence | Sweep touches only `builder.go` in atlas-character | PASS | `git diff --stat 61e5e4b94..HEAD -- services/atlas-character/` returns exactly one file, `character/builder.go`, +10/-2 — no call-site churn, consistent with the doc's claim of zero callers of the validating `Builder`. |

## Not evaluable from the diff

- DOM-24 (producertest stub) for `services/atlas-cashshop/atlas.com/cashshop/coupon/{concurrency_test.go,processor_test.go}`, `services/atlas-dragons/atlas.com/dragons/dragon/processor_test.go`, and `services/atlas-pets/atlas.com/pets/pet/processor_test.go`: these packages reach `AndEmit`/`producer.ProviderImpl` (grep confirms), and this diff edits `Build()` call sites inside them, but the producer-stub wiring itself lives outside the changed hunks (no `producertest` reference in the diff, and `atlas-pets/pet/processor_test.go`'s `TestMain` at line 49 sets up only a miniredis client, not a producer stub — the stub, if any, is injected per-test via `ProcessorImpl` fields not visible in this diff). Confirming DOM-24 compliance would require reading the full `NewProcessor`/mock wiring in each package, which is outside the diff's changed lines. `go test` passing is suggestive but not the file:line citation the evidence bar requires.
- Whether `libs/atlas-model`'s `model.ErrDecorator`/`degrade.Observe` signatures used at `services/atlas-login/atlas.com/login/character/processor.go:183-195` are the correct/current contract was not re-verified against `libs/atlas-model` source in this round (it compiled and the existing test `TestInventoryDecoratorDegradesLoudly` at `services/atlas-login/atlas.com/login/character/processor_test.go:54` passed, which is evidence but the library source itself was not read).

## Summary

### Blocking (must fix)
- DOM-28: `services/atlas-cashshop/atlas.com/cashshop/character/processor.go:50-56` — `InventoryDecorator` swallows the new `Build()` error with `p.l` in scope but never logs it.
- DOM-28: `services/atlas-npc-shops/atlas.com/npc/character/processor.go:72-79` — same pattern.
- DOM-28: `services/atlas-query-aggregator/atlas.com/query-aggregator/character/processor.go:48-66` — same pattern, two call sites (`InventoryDecorator`, `GuildDecorator`).
- DOM-28: `services/atlas-messages/atlas.com/messages/character/model.go:237-243` — `SetSkills` swallows the `Build()` error with neither a logger nor an explanatory comment, unlike its `atlas-consumables`/`atlas-pets` siblings.

### Non-Blocking (should fix)
- DOM-01 (judgment call, not this sweep's scope): `services/atlas-character/atlas.com/character/character/builder.go:340-341` — `modelBuilder.Build()` remains non-validating. Real DOM-01 gap on a `builder.go` in a `model.go` package; deferred as a design question per the task doc, not accepted here as settled.
- DOM-28 (documentation tension): `services/atlas-consumables/atlas.com/consumables/character/model.go:275-310`, `services/atlas-pets/atlas.com/pets/character/model.go:254-262` — comment-only fallback satisfies the task's own ruling but not DOM-28's literal "regardless of justification" text; flag for a controller ruling on which document governs model-level (no-logger) decorator fallbacks going forward.

## Not evaluable from the diff
See section above (DOM-24 producer-stub wiring; `libs/atlas-model` `ErrDecorator` contract).
