# Review: admit PROVISIONING environments through the REST env gate

**Commit:** `eb3818018` (range `04f37fa95..eb3818018`)
**Branch:** `fix-provisioning-env-self-writes`
**Brief:** `docs/tasks/fix-provisioning-env-self-writes/brief.md`
**Worktree reviewed:** `.worktrees/fix-provisioning-env-self-writes`

## Scope

`git diff --stat 04f37fa95..eb3818018`:

```
libs/atlas-env/record.go               | 11 ++++
libs/atlas-env/registry.go             | 22 +++++++-
libs/atlas-env/registry_test.go        | 52 ++++++++++++++++++
libs/atlas-rest/server/handler.go      | 13 +++--
libs/atlas-rest/server/handler_test.go | 96 ++++++++++++++++++++++++++++++++++
```

Matches the brief's required-changes list exactly: `Record.Provisionable()`,
`Registry.IsProvisionable()` (+ `MapRegistry`/`legacyRegistry` impls),
`ParseEnvironment` gate swap + doc comment, tests in both modules. No
unrelated files touched. Scope confirmed.

## 1. Is the security argument actually true?

**Verdict: false as stated; true only for 2 of ~66 consuming services. This
is the headline finding.**

The brief's and the new code comment's safety argument
(`libs/atlas-rest/server/handler.go:41-47`, and the commit message) is:

> Confinement to the caller's own data is enforced downstream by the scope
> layer (scope.Strict on reads, scope.AuthorizeWrite on writes), not by
> this handler.

This is written unconditionally, in a doc comment on a function in
`libs/atlas-rest/server` — a shared library. I checked how widely that
function's blast radius actually is:

- `ParseEnvironment` is invoked by every handler registered through
  `server.RegisterHandler` / `RegisterInputHandler` / `RegisterSimpleHandler`
  / `RegisterSimpleInputHandler` (`libs/atlas-rest/server/register.go:16,31,46,59`).
  `grep -rl "server.Register"` finds 83 call sites across **41 services**
  (atlas-buddies, atlas-buffs, atlas-cashshop, atlas-inventory, atlas-guilds,
  atlas-maps, atlas-parties, atlas-saga-orchestrator, atlas-world, … — the
  full list is in the transcript).
- The `ENVIRONMENT` header that feeds this gate is not something only
  atlas-configurations sends: `EnvHeaderDecorator`
  (`libs/atlas-rest/requests/header.go:46-56`) is wired centrally into the
  generic decorated-request helpers (`libs/atlas-rest/requests/decorated.go:13,22,31,40,49`,
  `libs/atlas-rest/requests/paged.go:66`) and into the Kafka producer/outbox
  paths (`libs/atlas-kafka/producer/provider.go:18`,
  `libs/atlas-outbox/bridge.go:59`) precisely so a PR environment's id
  propagates on every downstream call (FR-3.1/FR-3.2 in
  `docs/tasks/task-232-sparse-ephemeral-environments/prd.md:339-341`).
- The env-status projection that backs `env.CurrentRegistry()` is wired live
  (not the legacy no-op) via `service.WithEnvironmentRegistry` in **66**
  services' `main.go` (`libs/atlas-service/envregistry.go:29-36` +
  grep over `services/*/**/main.go`).
- A `scope` package (`Strict`, `AuthorizeWrite`) exists in exactly **two**
  services: `services/atlas-configurations/.../scope/scope.go` and
  `services/atlas-tenants/.../scope/scope.go`. Every one of the other ~39
  services reachable through the same shared `ParseEnvironment` gate has
  **no** such layer — `grep -rl "func Strict\|AuthorizeWrite"` returns
  nothing outside those two services.

So the comment's claim ("Confinement... is enforced downstream by the scope
layer... not by this handler") is only actually true for the two services
that implement `scope`. For every other service reachable through this gate,
nothing downstream confines a request naming a `PROVISIONING` (or `ACTIVE`,
pre-existing) environment to "its own data" — the comment states a
service-specific fact as if it were a property of the shared library.

**Mitigating context, not verified by this diff:** none of those other ~39
services read `env.FromContext`/`env.Id` for row-level filtering at all
(`grep -rln "env.FromContext\|env.Id\b" services/**` returns only
projection/state files in atlas-login, atlas-channel, atlas-character-factory,
atlas-world — none of which do row-scoping). Isolation for those services, if
it exists, is presumably enforced by deployment topology (separate
namespace/DB per environment), not by an in-process scope check. I did not
verify that topology — it is outside this diff's surface — so I cannot
confirm the practical exposure is zero, only that the *documented*
justification is wrong for those services.

**Also note:** this diff does not introduce a *new class* of exposure for
those 39 services — before this change, `IsActive`-gated `ACTIVE`-phase
requests with an `ENVIRONMENT` header already reached those services with
the same absence of a scope layer. What this diff does is widen the
admission window for all 66 wired services from `{ACTIVE}` to
`{PROVISIONING, ACTIVE}`, on the strength of a justification that is
services-specific but written as if general.

**Recommendation (non-blocking as a code defect, but should be fixed before
merge given "shared library, wide blast radius"):** qualify the doc comment
in `handler.go` and the commit message to say the scope-layer confinement is
a property of *scope-aware services* (name atlas-configurations /
atlas-tenants explicitly), not a guarantee this handler provides generically.
As currently worded, a future engineer reading this shared library's doc
comment could reasonably (and wrongly) conclude every service downstream of
`ParseEnvironment` is safe by default.

## 2. `IsActive` semantics preserved

Confirmed unchanged:

- `MapRegistry.IsActive` (`libs/atlas-env/registry.go:179-186`) — untouched,
  still `ok && rec.Active()`.
- `IsOwner` (`registry.go:205-219`) — still gates on `rec.Active()` at
  line 212 (`if !ok || !rec.Active() { return false }`), comment cites D4.
  Not touched by the diff.
- `EnvironmentsOwnedBy` (`registry.go:225+`) — still `if !rec.Active() { continue }`.
  Not touched by the diff.
- No caller of `IsActive` was redirected to `IsProvisionable`; the only
  caller switched is `ParseEnvironment`.

PASS.

## 3. Interface completeness

`Registry.IsProvisionable(e Id) bool` added at `libs/atlas-env/registry.go:30`.
Grepped every `.go` file in the repo for a method literally named
`IsActive` (the interface's sibling method) to find every type asserting
`Registry`: only `*MapRegistry` (`registry.go:189`) and `legacyRegistry`
(`registry.go:270`) implement it. Both got `IsProvisionable`:

- `MapRegistry.IsProvisionable` — `registry.go:188-197`, mirrors
  `IsActive`'s structure exactly (`e == ""` short-circuit, RLock, `ok &&
  rec.Provisionable()`).
- `legacyRegistry.IsProvisionable` — `registry.go:271`, returns `true`
  unconditionally, consistent with `legacyRegistry.IsActive` (`registry.go:270`,
  also unconditional `true`) — correct for legacy mode: no projection exists,
  so every query answers the FR-1.8 legacy default.

`go build ./...` passes clean in `libs/atlas-env`, `libs/atlas-rest`,
`libs/atlas-kafka` (consumes `env.Registry` as an interface, not an
implementer — no update needed), and `libs/atlas-service` (wires
`env.SetRegistry(env.NewMapRegistry(...))`, unaffected). No test doubles
implementing the full `Registry` interface exist elsewhere in the repo
(`grep -rln "func (.*) IsActive(" --include='*_test.go'` — empty). No compile
break found anywhere in the monorepo. PASS.

## 4. FR-5.2 / D4 / FR-3.6

Read `docs/tasks/task-232-sparse-ephemeral-environments/prd.md`:

- **FR-5.2** (`prd.md:379`): "During `PROVISIONING`, baseline deployments
  continue to own the environment's services; overrides receive no work; the
  ingress does not route." This is a statement about *traffic ownership*
  (`IsOwner`), which this diff does not touch — `IsOwner` still requires
  `rec.Active()`. **FR-5.2 holds**, and the new doc comment's claim that it
  is "still governed by Registry.IsOwner" is correct.
- **D4** (`prd.md:250`): "Fail closed. An operation whose environment cannot
  be resolved is dropped and alerted, never executed by the baseline." This
  is about *unknown* environments, not phase. `IsProvisionable` still
  rejects unknown ids exactly like `IsActive` did. **D4 holds.**
- **FR-3.6** (`prd.md:348`, restated verbatim in the error-semantics table
  at `prd.md:543`: "Unknown / inactive environment on a REST request |
  Reject (FR-3.6)"): "A request naming an unknown or inactive environment is
  rejected." This is a REST-specific requirement, and its literal wording
  says *inactive* — not merely *unknown*. `PROVISIONING` is, by definition,
  not `Active`. **This diff makes the implementation violate FR-3.6 as
  literally written in the PRD.** The old code comment correctly cited
  FR-3.6 for the old behavior; the new code comment silently drops the
  FR-3.6 citation rather than claiming compliance — that is the right call
  by the implementer (no false claim in the code), but it leaves the PRD
  itself stale: `prd.md:348` and `prd.md:543` still assert the old, now
  superseded requirement with no note that it was narrowed to "unknown /
  DEACTIVATING / DELETED" for the self-write case.

  The brief acknowledges this is "a genuine ordering conflict, not a typo"
  and explicitly instructs not to re-litigate the decision, so I am not
  treating the behavior change itself as a defect. But the brief's own
  framing ("FR-3.6's ... guarantee ... is satisfied by IsOwner alone") is
  imprecise: FR-3.6 is not about ownership at all, it is a REST-gate
  requirement, and this commit is exactly what changes it. The requirement
  that's actually satisfied by `IsOwner` alone is FR-5.2, not FR-3.6.

  **Recommendation (non-blocking, doc-only):** update
  `docs/tasks/task-232-sparse-ephemeral-environments/prd.md`'s FR-3.6 text
  and its error-semantics table row to reflect the amended rule ("unknown,
  DEACTIVATING, or DELETED" rather than "unknown or inactive"), so the PRD
  and the code stay in sync for the next reader.

## 5. Test honesty

- `TestParseEnvironmentAdmitsAProvisioningEnvironment`
  (`libs/atlas-rest/server/handler_test.go`, new) — applies a `PROVISIONING`
  record and asserts 200 + handler invoked. Against the pre-change gate
  (`IsActive`, requiring `PhaseActive`), a `PROVISIONING` record fails
  `IsActive`, so this test would get 400 pre-change. **Fails on old code,
  passes on new — honest.**
- `TestParseEnvironmentRejectsADeactivatingEnvironment` (new) — applies a
  `DEACTIVATING` record, asserts 400. Exercises the real
  `ok && !rec.Provisionable()` branch (record present, phase excluded), not
  the "unknown" branch. Genuine new coverage.
- `TestParseEnvironmentRejectsADeletedEnvironment` (new) — per the brief's
  own question 5, I checked whether this tests the DELETED path or the
  unknown path. `MapRegistry.Apply` (`registry.go:94-103`) deletes the
  record when `rec.Phase == PhaseDeleted`, so after `Apply`, the record is
  simply absent from the map — `ok` is `false`, identical to an unrelated
  unknown id. **This test does not exercise a distinct DELETED code path;
  it re-exercises the "unknown id" branch under a DELETED-flavored name.**
  This is honestly disclosed in the test's own comment ("Apply with
  PhaseDeleted removes the record, so a DELETED environment is
  indistinguishable from unknown to the registry — both are rejected.") and
  mirrored in `registry_test.go`'s equivalent case. Not a hidden gap —
  correctly labeled, and there is genuinely no way to construct a
  DELETED-but-present record through the registry's own public API
  (`Apply`/`ApplyTombstone` both remove it), so this is the best coverage
  achievable at this layer. Non-blocking, matches brief's own framing of the
  question.
- No pre-existing test asserted PROVISIONING-rejection through this gate.
  `git show 04f37fa95:libs/atlas-rest/server/handler_test.go`'s
  `TestParseEnvironmentRejectsAnUnknownEnvironment` uses `pr-999`, which is
  never `Apply`'d to the registry at all (i.e., it is testing "unknown", not
  "PROVISIONING"). So there was nothing to update, consistent with the
  brief's instruction to say so explicitly if none needed updating.
- `libs/atlas-env/registry_test.go`'s two new tests
  (`TestRecordProvisionableAcrossPhases`,
  `TestIsProvisionableAcrossPhasesAndUnknownAndLegacy`) cover all four
  phases + unknown + empty-legacy-id, matching the brief's requirement.

`go test ./...` passes in both `libs/atlas-env` and `libs/atlas-rest`.

## 6. Empty-id / legacy short-circuit (FR-1.8)

`MapRegistry.IsProvisionable` (`registry.go:189-191`) and
`legacyRegistry.IsProvisionable` (`registry.go:271`) both return `true` for
`e == ""`/unconditionally, exactly mirroring `IsActive`'s `e == ""` branch.
Covered by `TestIsProvisionableAcrossPhasesAndUnknownAndLegacy`'s
`r.IsProvisionable(Id(""))` assertion. PASS.

## Not evaluable

- Whether the ~39 non-scope-aware services' actual deployment topology
  (separate namespace / separate database per environment) makes the
  documented-but-absent "scope layer" claim practically moot. This requires
  reading each service's deployment manifests and DB wiring, which is
  outside this diff's surface (handler.go / registry.go / record.go + their
  tests). Flagged, not resolved.
- Whether `docs/tasks/task-232-sparse-ephemeral-environments/prd.md` will
  actually get updated — out of this commit's scope, noted as a
  recommendation only.

## Disposition

The mechanical implementation is correct and matches every concrete
"required change" in the brief: `Record.Provisionable()`,
`Registry.IsProvisionable()`, both implementations updated, `ParseEnvironment`
swapped, `IsActive`/`IsOwner`/`EnvironmentsOwnedBy` untouched, tests honest
and complete, builds and tests green in every touched module. The one real
defect is documentation, not logic: the safety argument written into
`handler.go`'s doc comment and the commit message is stated as a property of
the shared gate, but is verifiably true for only 2 of the ~41 services that
gate protects. Given this module's explicitly "wide blast radius" framing in
the review brief, I am treating that as blocking — it should be corrected
(scope the claim to the services that actually implement it, or explicitly
flag the gap for the other services) before this ships, since the comment
will be read by every future engineer who touches any of those other ~39
services.
