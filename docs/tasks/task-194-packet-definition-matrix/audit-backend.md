# Backend Audit — task-194-packet-definition-matrix (Go changes)

- **Scope:** `services/atlas-configurations/atlas.com/configurations` (new `socket/` package, `templates/socket`+`tenants/socket` REST/adapter changes, processor/resource/validation_error wiring) and `tools/packet-audit` (`seed-fname` subcommand) and `tools/template-duplicate-binding-guard.sh`.
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*, SUB-*, SEC-*, File Responsibilities)
- **Date:** 2026-08-06
- **Merge base:** `31c7a664f975e8fadcd2e0e4e893427bddc340d9`, HEAD `a52adecc6`
- **Build:** PASS (independently re-run)
- **Tests:** PASS (independently re-run on every changed package)
- **Overall:** PASS — zero blocking findings; two pre-existing, already-recorded-as-deferred minors carried forward, not upgraded.

## Independent re-verification (not taken on faith from verification.md)

```
$ cd services/atlas-configurations/atlas.com/configurations && go build ./...     → exit 0
$ cd tools/packet-audit && go build ./...                                          → exit 0
$ cd services/atlas-configurations/atlas.com/configurations && \
    go test ./socket/... ./templates/... ./tenants/... -count=1
    ok  atlas-configurations/socket            0.023s
    ok  atlas-configurations/templates          0.026s
    ok  atlas-configurations/templates/socket   0.004s
    ok  atlas-configurations/tenants            0.028s
    ok  atlas-configurations/tenants/characters  0.005s
    ok  atlas-configurations/tenants/characters/preset 0.006s
    ok  atlas-configurations/tenants/socket     0.003s
$ cd tools/packet-audit && go test ./cmd/... -count=1               → ok (2.1s)
$ ./tools/template-duplicate-binding-guard.sh                       → OK, exit 0
```
Matches the counts in `verification.md` §1/§2/§4. Build/test gate: **PASS**.

## Domain classification (Phase 2)

| Package | Classification | Rationale |
|---|---|---|
| `services/atlas-configurations/atlas.com/configurations/socket` | Support (rules library) | No `model.go`/`resource.go`; pure validation rules shared by both trees (design decision 1) |
| `templates/socket`, `tenants/socket` (+ `handler`, `writer` subpkgs) | Support (REST-only nested DTO trees) | No `model.go`; `RestModel` nests into the parent domain's top-level document, no independent processor/DB |
| `templates` / `tenants` (top-level, diff-touched files only: `processor.go`, `resource.go`, `validation_error.go`) | Pre-existing domain packages, files modified not introduced | Full DOM checklist run against the touched files only |
| `tools/packet-audit/cmd` (`seed_fname.go`) | CLI tool, not a microservice package | Evaluated against the task-specific fidelity-guard requirement, not the REST/DB DOM checklist |

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in `processor.go` | PASS | `templates/processor.go:19-169`, `tenants/processor.go` (same shape) — interface, `NewProcessor`, all `ProcessorImpl` methods co-located. No processor logic leaked into `adapter.go` or `rest.go`. |
| FILE-02 | `RestModel`+Transform/JSON:API in `rest.go` | PASS | `templates/socket/rest.go:8-44`, `tenants/socket/rest.go:8-44` — `RestModel`, `UnsupportedRestModel`, `Normalize` all in `rest.go`. Field additions (`FName`, `Options`) landed in `templates/socket/handler/rest.go:1-20` and `.../writer/rest.go:1-13`, both `rest.go` files. |
| FILE-03 | Cross-service request funcs in `requests.go` | N/A | No new cross-service HTTP client added by this diff (`data.NewProcessor` reused, pre-existing). |
| FILE-04 | Entity+Migration+TableName in `entity.go` | N/A | No entity changes in this diff. |
| FILE-05 | Builder/model/administrator/provider/state placement | N/A | No new persisted domain model added; `socket.Input`/`Binding`/`Issue` are transient validation DTOs, not GORM-backed models — Builder pattern does not apply (see anti-patterns discussion below). |
| FILE-06 | No package-named catch-all file | PASS | `templates/socket/adapter.go` holds exactly one function (`ToValidationInput`) — single-purpose adapter, not a `<pkg>.go` collapse. `socket/validate.go` holds only validation-rule types+funcs, one cohesive concern (rules library), not Processor+RestModel+requests bundled. No file in the diff carries ≥2 of the File-Responsibilities-table roles. |

## Domain Checklist — `templates` / `tenants` processor & resource (touched files only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `templates/processor.go:38` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`; identical in `tenants/processor.go`. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `templates/resource.go:38,73,99,117,136,160`; `tenants/resource.go:44,62,94,117,150` — every `NewProcessor` call passes `d.Logger()`. |
| DOM-09 | Transform errors handled | N/A | No `Transform(` calls added/touched in this diff's resource files. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `grep -rn os.Getenv templates/resource.go tenants/resource.go templates/socket tenants/socket socket/` → zero matches. |
| DOM-13/14/15 | No cross-domain logic, no direct provider/DB calls in handlers | PASS | `templates/resource.go` / `tenants/resource.go` every handler calls `NewProcessor(...).<Method>` only; no `db.Create`/`db.Save`/provider calls in either file. |
| DOM-17 | Domain error → HTTP status mapping | PASS | `templates/resource.go:39-49`, `tenants/resource.go:118-129,97-108` — `errors.As(err, &ve)` → 400 with JSON:API error body; all other errors → `server.WriteErrorResponse` (503/500 per its internal transient classification). |
| DOM-21 | No atlas-constants/atlas-opcodes duplication | PASS | `socket/validate.go:16,28` imports `opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"` and uses `opcodes.ServiceLogin`/`opcodes.ServiceChannel` (confirmed defined at `libs/atlas-opcodes/config.go:6-7`) rather than re-declaring `"login"`/`"channel"` string literals. `isKnownService`/`OpCodePattern`/`ParseOpCode` have no existing equivalent in `libs/atlas-opcodes` (`appliesToService` there is a different, unexported, differently-scoped helper — confirmed no name/behavior collision). |
| DOM-22 | Dockerfile lib-mention coverage for new direct require | PASS (with note) | `templates/processor.go` pulls in a new direct require, `github.com/Chronicle20/atlas/libs/atlas-opcodes` (`go.mod:9`). This repo has moved to **one shared, parameterized root `Dockerfile`** (not per-service) — `atlas-opcodes` appears in the mod-only `COPY` block (`Dockerfile:38`), the source `COPY` block (`Dockerfile:68`), and the synthesized `go.work use()` loop (`Dockerfile:94`); there is no per-lib `go mod edit -replace` block under this architecture (workspace `use()` supersedes it). The objective test — `docker buildx bake atlas-configurations` — passed with an exported image (verification.md §3, independently plausible given the build succeeds against the exact same Dockerfile with the new require present). DOM-22's literal "4 blocks in `services/<svc>/Dockerfile`" template is stale for this repo's current (shared-Dockerfile) architecture; the underlying purpose (missing-COPY breaks docker build) is satisfied. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `grep -rnE '^\s*go (func|[A-Za-z_])'` over every changed package returns zero bare `go` statements. |
| DOM-27 | Transient DB errors → 503 not bare 500 | PASS | `grep -n StatusInternalServerError templates/resource.go tenants/resource.go` → zero matches; every failure path routes through `server.WriteErrorResponse(d.Logger())(w)(err)` (e.g. `templates/resource.go:48,76,102,120,148,163`), and `main.go:47` registers `server.RegisterTransientErrorClassifier` (pre-existing, unmodified by this diff). |

## Sub-Domain / Support Package Checklist — `socket`, `templates/socket`, `tenants/socket`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-04 (adapted) | No manual JSON parsing in resource-adjacent code | N/A | `adapter.go`/`rest.go` are pure struct transforms; no `json.NewDecoder`/manual unmarshal added outside the CLI tool (which is not a REST handler). |
| — | Socket validation unconditional, not gated by `WithValidator` | PASS (spec) | `templates/processor.go:87-90` (`Create`) and `:125-126` (`UpdateById`) call `socketValidate` unconditionally, never behind `p.validator != nil`; matches architectural decision 2. |
| — | Both trees stay parallel (decision 1) | PASS (spec) | `templates/socket/{adapter,rest}.go` and `tenants/socket/{adapter,rest}.go` are byte-for-byte structurally identical except import paths — the documented duplication-by-design, not a collapsed-file defect. |
| — | `Options` uses `omitzero` not `omitempty` (decision 3) | PASS | `templates/socket/handler/rest.go:18` `Options map[string]interface{} \`json:"options,omitzero"\`` (same in `writer/rest.go:10`, both trees). Proven live by `TestRestModel_OptionsOmittedWhenAbsent` and `TestRestModel_EmptyOptionsObjectSurvives` (`templates/socket/rest_test.go:95-126`). |
| — | Socket+preset errors merged into one 400 (decision 4) | PASS | `templates/processor.go:143-145` returns `&validationFailureError{errors: presetErrs, socketIssues: issues}` when either is non-empty, never short-circuiting; proven by `TestUpdateById_MergesSocketAndPresetIssues` (`templates/socket_validation_test.go:164-220`). |
| — | `Issue` has no `Severity` field (decision 5) | PASS | `socket/validate.go:32-35` — `Issue{Path, Message}` only. |

## Multi-Tenancy

| Check | Status | Evidence |
|---|---|---|
| Tenant context synthesis on tenants-tree PATCH | PASS, reviewed for safety | `tenants/resource.go:81-93` builds a `tenant.Model` from URL `tenantId` + body `region/major/minor` via `tenantlib.Create` (confirmed to exist at `libs/atlas-tenant/processor.go:31`) and `tenantlib.WithContext` (`libs/atlas-tenant/processor.go:90`), with a graceful `terr != nil` fallback (Warn log, continues without tenant ctx) rather than a hard failure. Downstream `preset.Validator.validateOne` (pre-existing, unmodified — `templates/characters/preset/validator.go:62-63`) uses `tenant.FromContext(ctx)()` (non-panicking) to gate atlas-data-dependent rules, so a missing/failed tenant synthesis degrades to skipping those rules rather than panicking. No `tenant.MustFromContext` risk introduced. |

## `seed-fname` JSON fidelity guards (tools/packet-audit)

| Check | Status | Evidence |
|---|---|---|
| Unknown top-level/socket/entry key stops the run (Guard 1) | PASS | `tools/packet-audit/cmd/seed_fname.go:198-250` (`loadSeedTemplate`) — three unknown-key checks (`knownTopLevelKeys`, `knownSocketKeys`, `knownEntryKeys`) each return a non-nil error rather than silently dropping the key. Proven live by `TestSeedFName_FailsLoudlyOnUnknownTopLevelKey` / `...UnknownEntryKey` / `...UnknownSocketKey` (`seed_fname_test.go:290,307,331`). |
| Untouched values round-trip verbatim (Guard 2) | PASS | Every non-modelled field is `json.RawMessage` (`seedDoc`/`seedEntry`, `seed_fname.go:70-98`), so re-marshalling cannot alter semantic content; proven by `TestSeedFName_RealTemplatesSemanticFidelity` and `TestSeedFName_RealTemplatesInsertionCoverage` (`seed_fname_test.go:769,902`) against the real committed corpus. |
| Idempotency | PASS | `TestSeedFName_ReRunIsIdempotent` (`seed_fname_test.go:664`) and `TestSeedFName_RealTemplatesIdempotentSecondRun` (`:949`). |
| No shell/subprocess execution of untrusted input | PASS | Pure Go, `flag`/`os`/`encoding/json` only; no `exec.Command`. |

## `tools/template-duplicate-binding-guard.sh` (SEC review of the new shell guard)

| Check | Status | Evidence |
|---|---|---|
| Untrusted-input parsing | PASS | Script only globs and `json.load`s repo-committed files under `services/atlas-configurations/seed-data/templates/` (`template-duplicate-binding-guard.sh:19,27`) — no external/user-supplied input, no shell variable interpolation into a command string, no `eval`. |
| Fails loudly, non-zero exit on defect | PASS | `sys.exit(1)` on any bad opcode or duplicate binding (`:53`); confirmed `OK: 22 template arrays...` / exit 0 on the current (clean) corpus by direct re-run. |

## Test Quality

- Table-driven tests used throughout: `socket/validate_test.go:9-176` (`ParseOpCode`, `Validate` rule table), consistent with DOM-20.
- No `*_testhelpers.go` files added; `testDB`/`validTemplate`/`validTenant` helpers live inline in the relevant `_test.go` files (`templates/socket_validation_test.go:33-66`, `tenants/socket_validation_test.go`), consistent with the project's ban on standalone test-helper files. `RestModel`/`Input`/`Binding` are plain struct literals, not GORM-backed domain `Model`s, so the Builder-pattern requirement for test setup does not apply to them.
- Two **non-blocking, already-recorded** minors carried forward from `.superpowers/sdd/plan/progress.md` (not new findings, not upgraded in severity):
  - `socket/validate.go:108` — duplicate-binding map key uses untrimmed `b.Name`; a name differing only by leading/trailing whitespace would not be flagged as a duplicate. No corpus evidence of this occurring (progress.md Task 3).
  - `tools/packet-audit/cmd/seed_fname_test.go:860-861` — `assertFnamesMatch` discards the two-value type-assertion `ok` (`realSock, _ := real["socket"].(map[string]any)`); if a template's `socket` key were absent entirely, both sides would silently resolve to `nil`/empty and the loop would compare 0 entries, passing vacuously instead of failing loudly. Latent only — the real corpus always has a `socket` section, and `loadSeedTemplate`'s production-path unknown-key guard is unaffected (this is test-only code, not shipped). (progress.md Task 7).

## Summary

### Blocking (must fix)
None found.

### Non-Blocking (should fix)
- `socket/validate.go:108` — trim `b.Name` before using it as the duplicate-detection map key (already tracked as a deferred minor).
- `tools/packet-audit/cmd/seed_fname_test.go:860-861` — capture and assert the `ok` from the two `map[string]any` type assertions in `assertFnamesMatch` so an absent `socket` section fails loudly instead of comparing vacuously (already tracked as a deferred minor).

### Verified but flagged as an architecture-doc staleness, not a code defect
- DOM-22's checklist text assumes a per-service `services/<svc>/Dockerfile` with a `go mod edit -replace` block; this repo uses one shared root `Dockerfile` with a synthesized `go.work use()` block instead. The new `atlas-opcodes` require is correctly represented in all three applicable places, and `docker buildx bake atlas-configurations` passed. Recommend updating the DOM-22 checklist text to describe the shared-Dockerfile pattern, out of scope for this branch.

### Could not independently verify
- The full `docker buildx bake atlas-configurations` run (not re-executed here — expensive, and verification.md's transcript is a verbatim, plausible BuildKit log with exit-0 confirmation; no reason found in code review to doubt it).
