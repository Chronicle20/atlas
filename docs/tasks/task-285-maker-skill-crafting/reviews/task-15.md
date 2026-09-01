# Task 15 Review — `atlas-maker` service skeleton

Commit range reviewed: `a1d635c..b2bac01d7` (code commit `b2bac01d7`; docs-only ledger commit
`437bc309d` on top ignored per instructions).

## Scope confirmed

The diff touches exactly the files the brief's `### Files` section names, plus `.gitignore`
(not listed, but see finding below): `go.mod`, `go.sum`, `main.go`, `wiring_test.go`,
`README.md` under `services/atlas-maker/atlas.com/maker/`, plus `go.work` and `.gitignore`.
No deploy-surface files (`docker-bake.hcl`, `deploy/k8s/**`, `deploy/shared/routes.conf`,
`.github/config/services.json`) were touched — correctly deferred to Task 16 per the
controller's ruling. No Task 16-24 domain code (reagent/crystal-band tables, recipe cache,
eligibility, reward draw, REST surface) appears anywhere in the diff. Scope matches the brief.

## Findings

### PASS — module builds and tests pass

```
cd services/atlas-maker/atlas.com/maker && go build ./... && go vet ./... && go test ./... -count=1 -v
```
Output: `TestMainWiresTheEnvironmentRegistry` and `TestServiceBootstraps` both PASS, `ok atlas-maker 0.015s`. `go vet` produced no output. Verified directly, not from the report.

### PASS (with a caveat) — `wiring_test.go` is a real test, but `TestServiceBootstraps` is weaker than the brief's literal ask

`wiring_test.go:13-21` (`TestMainWiresTheEnvironmentRegistry`) is a byte-for-byte copy of
reward-pools' only test — a source-string assertion, matching the pattern used identically
across all 60+ other services' `wiring_test.go` files I sampled (`atlas-ban`, `atlas-portals`,
`atlas-character-factory`, `atlas-storage`, `atlas-character`, etc. — all 21-27 lines, none of
which actually construct `server.New(l)...` in a test). This part is a faithful, non-tautological
copy of established repo convention.

`wiring_test.go:26-34` (`TestServiceBootstraps`) is new — no other service in the repo has a
test by this name (`grep -rl TestServiceBootstraps services/` returns only this file). The brief
(`task-15-brief.md:74`) asked for a test that "construct[s] the router the way main.go does and
assert[s] it builds without panicking and that the service's base route prefix resolves." What
was written instead calls `GetServer()` — a pure struct-literal constructor — and asserts its two
fields equal the literals hardcoded in the same file (`main.go:29-34`). It never calls
`server.New(l)`, never calls `.AddRouteInitializer(...)`, never exercises anything that could
panic during real route construction. It is a same-file regression pin (it would catch someone
accidentally changing `GetServer()`'s literals), not a router-construction smoke test.

This is a minor gap, not a blocking one: no other service in the fleet actually builds the
router in a test either, so "construct the router the way main.go does" was arguably not
achievable without inventing new test infrastructure the brief didn't ask for, and the
implementer's choice at least adds *something* over reward-pools' bare minimum. Noting as
non-blocking.

### PASS — `main.go` bootstrap matches reward-pools' shape and is append-ready

Diffed directly against `services/atlas-reward-pools/atlas.com/reward-pools/main.go:41-74`.
Structurally identical: `service.Bootstrap` → `database.Connect(l, database.SetMigrations(...))`
→ `server.RegisterTransientErrorClassifier` → `server.New(l).WithContext(...).WithWaitGroup(...).
SetBasePath(...).SetPort(...).AddRouteInitializer(...).Run()` → `rt.Wait()`. The migration list
(`main.go:40-42`) contains only the `seeder.SeedState` AutoMigrate, matching the brief's "empty
for now; Tasks 17-18 append." The `AddRouteInitializer` chain (`main.go:61`) contains only
`server.MountReadiness("/readyz", rt.Ready)`, matching "empty for now; Tasks 17, 18 and 24
append." Both lists are structured as a comma-separated arg list / method-chain, so later tasks
can append entries without restructuring — verified this is exactly reward-pools' own shape
(`reward-pools/main.go:45-50` and `:65-70`), which four other domains already append to.

### PASS — `go.work` addition is correct and non-disruptive

`git diff a1d635c..b2bac01d7 -- go.work` shows a single insertion:
`./services/atlas-maker/atlas.com/maker` at line 57, alphabetically between
`atlas-login` (line 56) and `atlas-map-actions` (line 58). No other line in `go.work` was
touched.

### PASS — `.gitignore` addition is warranted and consistent with the existing pattern

Not in the brief's file list, but `git diff -- .gitignore` shows one line added:
`/services/atlas-maker/atlas.com/maker/atlas-maker` (line 57), directly following the existing
`atlas-reward-pools`/`atlas-events` binary-ignore lines (`.gitignore:41`, `:56`). This is the
established per-service pattern for ignoring the `go build ./...` main-package binary artifact;
without it a 33 MB binary would be a candidate for accidental `git add`. Warranted, not
scope creep.

### PASS — `go.mod`/`go.sum` are faithful copies

`diff services/atlas-reward-pools/.../go.mod services/atlas-maker/.../go.mod` shows exactly one
line changed: `module atlas-reward-pools` → `module atlas-maker`. `go.sum` is byte-identical
between the two modules (`diff` produces no output) — consistent with the report's claim that
`go mod tidy` couldn't run in the sandbox but the copy is correct because the require blocks are
unchanged. Go version (`go1.27.0`) and `replace` directives preserved.

### PASS — no scope creep into Tasks 16-24

`README.md` documents an empty REST endpoints section ("_None yet — Task 24 adds..._") and empty
Kafka section, with no domain code anywhere. `main.go` has no reagent/crystal-band migrations, no
recipe cache, no eligibility or reward-draw logic, no REST handlers. Confirmed by direct read of
every file in `services/atlas-maker/`.

### Settled per controller ruling — not re-raised

- Deployment registration absence (Task 16 owns it).
- Omitted `consumerGroupId` (matches corrected reward-pools precedent, `eef9b2c7e`).
- README path inside `atlas.com/maker/` rather than `services/atlas-maker/README.md` (Task 16
  fixes it per controller note).

## Not evaluable

- Whether `go mod tidy` would in fact produce a byte-identical `go.sum` outside this sandbox
  (network-restricted here) — the module's `go.mod` require block is unchanged from
  reward-pools' so the copy is very likely correct, but I could not independently regenerate it
  to confirm bit-for-bit against a live `GOPROXY`.
- Whether a later plan task explicitly picks up `docs/adding-a-new-service.md` §1-7
  (CI/build registration, k8s base, overlays, ingress, database, GHCR visibility) — out of this
  unit's diff surface; the brief and controller's ruling both place it at Task 16, but I did not
  read Task 16's brief to confirm its file list actually covers all seven sections.

## Verdict

APPROVED_WITH_FINDINGS — one non-blocking note (`TestServiceBootstraps` is a same-file
regression pin rather than a literal router-construction smoke test). No blocking defects found;
build, vet, and tests all pass; `main.go`/`go.work`/`.gitignore`/`go.mod`/`go.sum` all match
reward-pools' precedent exactly where the brief calls for a match, and no Task 16-24 domain work
leaked into this commit.
