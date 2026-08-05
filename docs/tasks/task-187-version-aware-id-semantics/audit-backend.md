# Backend Guidelines Audit — task-187 (version-aware-id-semantics)

- **Scope:** changed Go packages only (`libs/atlas-constants`, `services/atlas-channel`, `services/atlas-character`, `services/atlas-data`, `services/atlas-monsters`)
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-07-30
- **Build:** PASS (all 5 modules: `go build ./...` clean)
- **Tests:** PASS (all 5 modules: `go test ./... -count=1` clean, 0 failures)
- **go vet:** clean in all 5 modules
- **`tools/goroutine-guard.sh`:** exit 0 (no new bare `go` statements)
- **`tools/skill-job-id-guard.sh`:** clean, "14 divergent const(s) checked", exit 0 — this is the task-187-specific CI gate (CLAUDE.md item 10) banning raw wire-id comparisons against the divergent GM/SuperGM/Pirate/Brawler const set; it passes.
- **`gen -check` drift gate:** `cd libs/atlas-constants/gen && go run . -check` → "OK" — all generated files (`identities_gen.go`, 22 `version_*_gen.go`, `registry_gen.go`, `baseline_gen.go`) are in sync with their sources of truth.
- **Overall:** NEEDS-WORK (2 Important findings, both in the same two new atlas-data packages; 0 Critical)

## Build & Test Results

```
libs/atlas-constants:                     go build/vet clean; go test: 11 packages ok
services/atlas-channel/atlas.com/channel: go build/vet clean; go test: ~80 packages ok
services/atlas-character/atlas.com/character: go build/vet clean; go test: ok (character pkg 5.4s)
services/atlas-data/atlas.com/data:       go build/vet clean; go test: ok (skillavailability: "no test files")
services/atlas-monsters/atlas.com/monsters: go build/vet clean; go test: ok
```

## Findings

### IMPORTANT-1 — GET /data/job-availability and GET /data/skill-availability do not paginate

- `services/atlas-data/atlas.com/data/jobavailability/resource.go:26-33`
- `services/atlas-data/atlas.com/data/skillavailability/resource.go:26-33`

Both handlers call `server.MarshalResponse[[]RestModel](...)` directly on the full unpaged result of `Processor.GetAvailable()`, e.g.:

```go
func handleGetJobAvailability(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ms := NewProcessor(d.Logger(), d.Context()).GetAvailable()
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(ms)
	}
}
```

`patterns-rest-jsonapi.md` line 15 is explicit: "Collection endpoints (`GET` returning a list) MUST paginate — use `paginate.ParseParams` + a paged processor method + `server.MarshalPaginatedResponse`, never `server.MarshalResponse[[]RestModel]` on a bare or filtered list." `docs/rest-pagination.md` (adopted repo-wide, task-117) confirms there is no "naturally small/bounded list" exemption — its §3 "Standard collections" class explicitly includes "atlas-data content dumps, LOW/naturally-bounded lists" at default 50 / max 250, i.e. still paginated, just with a small class.

This is not a case of "the codebase does it this way elsewhere" — the opposite is true: every *other* list endpoint already registered in `services/atlas-data/atlas.com/data/main.go` (skill, commodity, cash, job, cosmetic/hair, pet, etc, cf. `resource.go` in each of those packages) uses `paginate.ParseParams`/`MarshalPaginatedResponse`. These two new packages are the only atlas-data list endpoints still on the pre-task-117 pattern, introduced fresh by this branch.

Severity: Important (structural / File-Responsibilities-adjacent REST convention, explicitly a repo-wide "MUST").

### IMPORTANT-2 — `skillavailability` package has zero test files

- `services/atlas-data/atlas.com/data/skillavailability/` contains `processor.go`, `resource.go`, `rest.go` — no `*_test.go` at all.
- Contrast: the sibling `jobavailability` package (same shape, same task) has `resource_test.go` (94 lines, table-driven).

`GetAvailable()` (`skillavailability/processor.go:30-48`) contains non-trivial logic — it iterates `AvailableIdentities()`, resolves each back to a wire id, and has a defensive skip-on-miss branch (`processor.go:38-44`) that is completely unexercised by any test. DOM-20 ("table-driven tests") and the AI-guidance testing workflow ("Always run tests... No partial implementations") are violated: this is an asymmetric gap against the otherwise-careful test discipline shown everywhere else in this branch (e.g. `resurrection_test.go` +69 lines, `hp_mp_gain_test.go` +68 lines, `character_attack_common_mastery_test.go` +121 lines, `buff/consumer_test.go` +63 lines).

Severity: Important (missing test coverage for a package with real branching logic; asymmetric with its direct sibling).

## Verified as SAFE (not findings) — reviewed under specific instruction to check

### `resurrection.go:104` — `skillIdentity, _ := set.Skill.Resolve(...)`

`services/atlas-channel/atlas.com/channel/skill/handler/resurrection/resurrection.go:94-104` discards the resolve `ok`. Verified safe: `selectByVariant` (`recipients.go:22-40`) `switch`es on `skillId`, with `case skill2.BishopResurrection:` as the only special case and everything else (including the zero-value Identity from a failed resolve) falling to the `default:` map-scoped, party-agnostic selector — which is documented as the correct behavior for GmResurrection/SuperGmResurrection too. The comment at `resurrection.go:94-101` correctly states the miss is unreachable in practice (`Apply` only runs after the dispatcher already resolved this wire id to one of the three registered identities) and, even if it weren't, the fallback is behavior-preserving. PASS.

### `job.Beginner == Identity(0)` zero-value collision

`libs/atlas-constants/job/identities_gen.go:6` defines `Beginner Identity = 0`, so an unresolved-resolve zero-value `Identity` fallback is indistinguishable from a genuine `job.Beginner` match under `job.IsAIdentity`/`isIdentity` (`job/identity.go:115-119`, first-clause `characterBranch == referenceBranch && id >= referenceId` is satisfied at `id=referenceId=0`). Audited every call site in the diff that discards the resolve `ok` (`character/character/processor.go:1548`, `:1832`, `:2193`) and every guarded-jok site that could reach `job.IsBeginnerIdentity`/`job.IsAIdentity(..., job.Beginner, ...)` (`:1658`, `:1819`). Result: the three `ok`-discarding sites feed `getSkillBook`/`pointResetPolicyFor`/pool-min tables, none of whose row sets reference `job.Beginner`, so a coincidental `0` match is behaviorally identical to a genuine Beginner (both fall to the same default row) — confirmed against `point_reset.go:46,109-127` "default: Beginner/Noblesse" comments. The two sites that actually test `job.IsBeginnerIdentity` (`:1658`, `:1819`) are both `jok &&`-guarded, so the zero-fallback never reaches them unguarded. PASS — no bug, but worth flagging that this is a coincidental safety, not a structural guarantee; future callers adding an `IsAIdentity(unresolvedJid, job.Beginner, ...)` check without a `jok` guard would silently misclassify.

### DOM-21 (no atlas-constants type reinvention) across all 5 modules

No new domain type, alias, or numeric constant duplicating `libs/atlas-constants` was found in the diff. Every version-divergent wire-id comparison found in the diff (`hide.go`, `common.go`, `common_consume` paths, `character_buff_cancel.go`, `character_skill_prepare.go`, `character_skill_use.go`, `character_attack_common.go`/`character_attack_melee.go`/writer `character_attack_common.go`, `processor.go` in atlas-character, `buff/model.go` in atlas-monsters, `buff/consumer.go` in both atlas-channel and atlas-monsters, `skill/reader.go` in atlas-data) correctly routes through `constants.For(region, major, minor).Skill.Resolve` / `.Job.Resolve` rather than reinventing per-version tables. PASS.

### Immutability / Set API design (`libs/atlas-constants/skill/identity.go`, `job/identity.go`)

`Set` (both packages) has unexported map fields, is only constructible via the generated `NewSet<Version>()`/`Baseline()` functions, and exposes only read methods (`Resolve`, `Wire`, `Available`, `Name`, `AvailableIdentities`) with value receivers — no exported mutator. `constants.For` (`libs/atlas-constants/constants/for.go`) is a clean, race-safe (`sync.Map` for the dedup-log path) tenant-keyed selector with no swallowed errors. PASS.

### Processor Interface+Impl pattern (`jobavailability`, `skillavailability`)

Both packages: `Processor` interface + `ProcessorImpl` + `NewProcessor(l, ctx) Processor` + `var _ Processor = (*ProcessorImpl)(nil)` compile-time assertion — matches the documented pattern exactly. `resource.go` calls the processor only (no direct provider/DB access — there is no DB in these packages at all). File Responsibilities Checklist: `Processor` in `processor.go` ✓, `RestModel`+JSON:API methods in `rest.go` ✓ (no `Transform` function, but there is no domain `Model` to transform *from* — `GetAvailable()` builds `RestModel` directly off `constants.For(...)`, which is a deliberate, documented design, not a misplaced symbol), no package-named catch-all file. PASS.

## Summary

### Blocking (must fix)
- IMPORTANT-1: `jobavailability/resource.go` and `skillavailability/resource.go` must convert to `paginate.ParseParams` + `server.MarshalPaginatedResponse`, matching every other atlas-data list endpoint (task-117 convention).
- IMPORTANT-2: `skillavailability` needs a test file (mirror `jobavailability/resource_test.go`) covering `GetAvailable()`, the defensive-skip branch, and the JSON:API `RestModel` methods.

### Non-Blocking (should fix)
- None identified beyond the two Important items above. The `job.Beginner==0` coincidental-safety note above is documentation-worthy but not a live bug in this diff — flagged as a future-maintenance trap, not a required fix.
