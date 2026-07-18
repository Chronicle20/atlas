# Backend Audit — task-171 (lint/format-enforcement infra)

- **Scope:** AUTHORED Go/tooling surface only — `tools/lint.sh`, `tools/lint.versions`, `.golangci.yml`; the 23 atlas-tenant import-alias one-liners (commits `9fd3e037c` + `1d9bd4389`); the 12 lint-residue fixes (commits `a0c8ae04b..a64fff1b2`). Commit `cde242a84` (4235-file machine reformat) explicitly EXCLUDED per audit instructions.
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-17
- **Build:** PASS (all 5 residue-touched modules + representative alias-touched modules build clean)
- **Tests:** PASS (all touched packages' test suites green, no regressions)
- **Overall:** PASS

## Applicability of DOM-* Checklist

This branch has no domain-package surface (no `model.go`/`entity.go`/`builder.go`/`processor.go` structural changes, no new REST/Kafka/tenancy code paths). Confirmed DOM-01 through DOM-20 have **no applicable surface** — the diff touches only: an import alias (23 files), five isolated one-line-or-so lint fixes across five services, and shell/YAML tooling config. Listing the ones with any conceivable applicable surface:

| ID | Applicable? | Finding |
|----|-------------|---------|
| DOM-06 (FieldLogger) | N/A | No processor constructor signatures changed. |
| DOM-09 (Transform errors handled) | N/A | No `Transform` call sites touched. |
| DOM-12 (no os.Getenv in handlers) | Touched adjacent | `seeder.go` (not a `resource.go` handler) reads `os.Getenv("SEED_ENABLED")` — pre-existing pattern, only the boolean logic was collapsed (QF1007), not the read location. No handler-layer os.Getenv introduced. PASS by inapplicability. |
| DOM-21 (no atlas-constants duplication) | N/A | No new domain type/enum/constant declared anywhere in this diff. |
| DOM-24 (Kafka producer stubbed in tests) | N/A | No test in the diff calls `AndEmit`/`producer.Produce`/consumer entry points; `seeder_test.go` and `resource_test.go` touched are pure-config / DB-error-path tests, no emit surface. |
| All other DOM items | N/A | No REST/JSON:API, entity, builder, provider, Kafka, or multitenancy surface in this diff. |

File Responsibilities checklist (FILE-01..06) and SUB-*/EXT-* checklists: **N/A** — no package structure was created, split, or reorganized; every touched file already existed and keeps its existing responsibility (a var rename inside an existing `processor.go`, a boolean-expression collapse inside an existing `seeder.go`, an import-alias inside existing files, etc.).

## Findings — Import-Alias One-Liners (9fd3e037c + 1d9bd4389)

23 `.go` files changed `"github.com/Chronicle20/atlas/libs/atlas-tenant"` → `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`. Verified:

- **Correctness**: `libs/atlas-tenant/tenant.go:1`, `registry.go:1`, `processor.go:1` all declare `package tenant` — the alias `tenant` is literally the package's own name, so this is semantically a no-op rename, not a real alias. PASS.
- **No shadowing**: grepped every touched file for a colliding identifier named `tenant`. Only hit: `services/atlas-skills/atlas.com/skills/skill/cooldown_registry.go:129` — `tenant: t,` inside a `CooldownHolder{...}` composite literal. This is a **struct field name**, a distinct namespace from the package identifier in Go; it does not shadow or collide with the `tenant` import. PASS.
- **Consistency**: all 23 sites use the identical form `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`. PASS.
- **Completeness**: `1d9bd4389` itself documents a file missed by the initial 22-file sweep (`atlas-rates/character/initializer.go`) and fixes it — confirmed no unaliased `"github.com/Chronicle20/atlas/libs/atlas-tenant"` import remains anywhere in the tree post-fix (grep of current HEAD returns zero unaliased hits; the four remaining non-`tenant`-literal aliases — `tenantModel`, `tenantlib` ×2, `tenant2` — are pre-existing code outside this diff's scope, already aliased under different names, unaffected by the goimports-duplicate defect this fix targets). PASS.
- **Build verification**: representatively built `atlas-skills` (contains the one struct-field collision candidate) plus a `tools/lint.sh --check --fmt --go services/atlas-skills` run — clean, `lint.sh: OK`, confirming the alias resolves the goimports duplicate-import defect end-to-end. PASS.

**Minor (non-blocking) observation**: `.golangci.yml:24-28`'s NOTE instructs future authors to "always write `tenant \"…/atlas-tenant\"`" as if that literal identifier is technically required. It is not — any alias avoids the goimports duplicate-import bug (three pre-existing, out-of-scope files alias the same import as `tenantModel`, `tenantlib`, `tenant2` and are unaffected). The guidance is a style convention for consistency, not a build-correctness requirement; the comment's wording slightly overstates the constraint. This does not affect correctness of the 23 fixes themselves — recorded as a documentation-precision nit only, not a code defect.

## Findings — Lint-Residue Fixes (a0c8ae04b..a64fff1b2)

| Commit | Change | Verification | Status |
|---|---|---|---|
| `a0c8ae04b` | `atlas-character/character/processor.go:39-40` rename `blockedNameErr`→`errBlockedName`, `invalidLevelErr`→`errInvalidLevel` (ST1012); `resource.go:164` updated to match | Grepped for stray old names post-commit — none. Both declaration and both use sites renamed together. Package builds and `character` tests pass. | PASS — pure rename, behavior-preserving |
| `e2b712d99` | `seeder.go:28-31` collapses `enabled := true; if Getenv==\"false\" { enabled = false }` → `enabled := os.Getenv("SEED_ENABLED") != "false"` (QF1007) | Truth-table check: env unset → `Getenv` returns `""`, `"" != "false"` → `true` (matches original default-true). env=`"false"` → `false` (matches). env=any other string (e.g. `"true"`, `"1"`) → `true` (matches). All three cases equivalent to the original if/else. `seeder_test.go` `_ =`-wrapped `os.Setenv`/`os.Unsetenv` calls in test setup/teardown — return values genuinely irrelevant in a controlled test process (valid, hardcoded env var names, no untrusted input). Tests pass. | PASS — logically equivalent incl. unset-env edge, errcheck ignores are safe |
| `da0aa94cf` | `atlas-inventory/inventory/resource_test.go:43` `failConn{err: f.err}` → `failConn(f)` (S1016) | Confirmed `failConn` (line 27) and `failConnector` (line 39) are both declared as `struct{ err error }` — identical single-field layout, so the type conversion is bit-for-bit equivalent to the struct literal. Test builds and passes. | PASS — conversion is exact, not just superficially similar |
| `a119fe73b` | `atlas-keys/key/processor.go:18-21` removes unused `entityModelMapper = model.Map(Make)`, keeps `entitySliceMapper` | Grepped the entire `atlas-keys` service for `entityModelMapper` post-removal — zero hits anywhere (not just the file). Genuinely dead. Matches the documented anti-pattern "Leaving dead code after refactoring" (`anti-patterns.md:35`). Package builds and tests pass. | PASS — confirmed genuinely unused, not a hidden call site |
| `a64fff1b2` | `atlas-messages/command/character/commands.go` + `commands_test.go`: `golang.org/x/net/context` → stdlib `context` (SA1019) | Both files use only `context.Context` and (transitively via callers) the standard context API — `golang.org/x/net/context` has re-exported the stdlib's `Context`/`Background`/`TODO`/etc. as type aliases since Go 1.9, so the swap is a no-op at the type level. No `x/net/context`-specific API (e.g. the pre-1.7 non-aliased shim) is used. Package builds and tests pass. | PASS — API-identical swap |

## `tools/lint.sh` Review

- Shell correctness: `set -euo pipefail` (line 16); `resolve_base` degrades to un-gated whole-module linting with an explicit warning (lines 100-108) rather than silently gating on nothing or crashing — safe failure mode. PASS.
- The `--go`/`--ui` both-zero-out-both footgun (each flag zeroes the *other* ecosystem's flag rather than restricting to itself, so `--go --ui` together leaves both at their last-assigned value non-intuitively) is present at lines 55-56 (`--go) DO_UI=0 ;; --ui) DO_GO=0 ;;`) — already known/tracked per audit instructions, not re-flagged as new.
- `discover_modules` path handling (lines 91-104) correctly resolves both absolute and relative caller-supplied paths against `$ROOT`. No injection risk observed (paths are used in `find`, not `eval`'d).
- No behavior gap found beyond the already-tracked footgun.

## `.golangci.yml` Review

- `version: "2"`, `linters.default: standard` (errcheck, govet, ineffassign, staticcheck, unused) — matches the residue-fix categories actually seen (ST1012, QF1007, S1016, SA1019, unused/dead-code, errcheck). Consistent. PASS.
- `formatters.enable: [gofumpt, goimports]` with `local-prefixes: github.com/Chronicle20/atlas` (line 21) — correctly groups intra-repo imports per FR-2.2 cited in the comment. PASS.
- The atlas-tenant NOTE (lines 24-28) is accurate as to the underlying defect (goimports mis-resolving package name from import path when the last path segment doesn't match `package tenant`) — verified by reproducing a clean `lint.sh --check --fmt` run against an aliased file. See the Minor note above re: the comment slightly overstating that the identifier must literally be `tenant`.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- `.golangci.yml:24-28` — NOTE wording implies the alias identifier must literally be `tenant`; only *aliasing* (any name) is technically required to avoid the goimports duplicate-import defect. Documentation-precision nit, not a code defect — no action required unless the comment is edited for future accuracy.
