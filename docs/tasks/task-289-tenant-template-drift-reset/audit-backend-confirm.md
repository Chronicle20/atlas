# Backend Audit — atlas-configurations (task-289, confirmation pass)

- **Service Path:** services/atlas-configurations/atlas.com/configurations
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-02
- **Baseline audit:** `docs/tasks/task-289-tenant-template-drift-reset/audit-backend.md` / `.json` (9613e7259..0f1b3c75e)
- **Range re-audited:** 9613e7259 (merge-base) .. 95b4e98f0 (HEAD), i.e. baseline + remediation commits `9950fd065`, `95b4e98f0`
- **Build:** PASS
- **Tests:** all packages `ok`, 0 failed
- **Overall:** NEEDS-WORK (one prior blocking finding remains open; no new blocking finding)

## Build & Test Results

```
$ go build ./...          -> exit 0, no output
$ go test ./... -count=1  -> all listed packages "ok" or "[no test files]"; none failed
$ go vet ./tenants/...    -> exit 0
```

## Scope note — two findings excluded by explicit user ruling

Per the dispatching instructions, **DOM-04/DOM-05 in both `tenants` and
`templates/characters/preset`** are out of scope for this confirmation and are
NOT counted toward the 7 in-scope baseline findings, and are NOT reported as
blocking here. They remain knowingly-accepted: neither package has the
canonical immutable domain `Model` the rule's `Transform(Model) (RestModel,
error)` shape requires (`services/atlas-ban/atlas.com/ban/report/rest.go:41`
is the canonical shape); building that layer is a separate task. Verified
unchanged by the remediation commits: `grep -rn "^func Transform"
tenants/*.go templates/characters/preset/*.go` → zero matches, same as
baseline.

## Disposition of the original 7 in-scope blocking findings

| # | Baseline ID(s) | Baseline evidence | Status now | Evidence now |
|---|---|---|---|---|
| 1 | FILE-01 | `ResetById`/`validateReset` in bare `tenants/reset.go` | **CLOSED** | `tenants/reset.go` renamed to `tenants/processor_reset.go` (commit `9950fd065`); `ResetById` is a `*ProcessorImpl` method there (`tenants/processor_reset.go:36`), matching the `processor_<group>.go` split FILE-01 permits. `ls tenants/*.go` confirms no `reset.go` remains. |
| 2 | DOM-08 / SUB-03 | `POST /{tenantId}/reset` via bare `rest.RegisterHandler` | **CLOSED** | `tenants/resource.go:37`: `r.HandleFunc("/{tenantId}/reset", normalizeResetBody(rest.RegisterInputHandler[ResetRestModel](l)(si)("reset_configuration_tenant", handleResetConfigurationTenant(db)))).Methods(http.MethodPost)`. The route resolves through `rest.RegisterInputHandler[T]`; `normalizeResetBody` is a delegating middleware wrapper around its output (the pattern doc's own precedent for tracing an alias to its definition — file-responsibilities.md:474), not a substitute registration path. |
| 3 | SUB-04 | `json.NewDecoder(r.Body).Decode(&body)` in `parseResetSections`, `resource.go:78` | **STILL OPEN** | `parseResetSections` and its manual decode were removed, but manual body parsing was relocated rather than eliminated: `grep -n "json.NewDecoder\|json.Unmarshal\|io.ReadAll" tenants/resource.go` → `resource.go:76` (`io.ReadAll(r.Body)` in `normalizeResetBody`) and `resource.go:131` (`json.Unmarshal(raw, &doc)` in `hasResetEnvelope`). SUB-04's pass criterion (file-responsibilities.md:228) is literally "grep `resource.go` for `json.NewDecoder`, `json.Unmarshal`, `io.ReadAll` → zero matches." Both symbols are present. The *purpose* changed (this is now envelope-presence normalization ahead of `RegisterInputHandler`'s own decode, not a hand-rolled substitute for it, and it is tested — `resource_reset_test.go:87` `TestResetEndpoint`, comment at `resource_reset_test.go:309`), but the rule's literal grep-based pass bar is not met. See Blocking below. |
| 4 | DOM-19 | `resetRequest` (`Data.Attributes.Sections`) nested envelope, defined outside `rest.go` | **CLOSED** | `ResetRestModel` (`tenants/rest.go:54-57`) is flat — `Id string`, `Sections []string` — no nested `Data`/`Attributes`; implements `GetName()`/`GetID()`/`SetID()` (`rest.go:59-70`); lives in `rest.go`. |
| 5 | DOM-32 | Hand-rolled `writeJSONAPIError` helper, `resource.go:53-63` | **CLOSED** | `grep -n "writeJSONAPIError" tenants/*.go` → zero matches; the function was deleted. Every error branch in `handleResetConfigurationTenant`, `handleUpdateConfigurationTenant`, `handleCreateConfigurationTenant`, and `normalizeResetBody` now writes the status code and body inline at the call site rather than through an extracted helper — the pattern doc's pass criterion explicitly allows this ("write the status code directly", file-responsibilities.md:474); the 500 default arm still uses `server.WriteErrorResponse` (`resource.go:351`). No manual tenant-header parsing (`grep -n "r.Header.Get" tenants/resource.go` → zero matches). |
| 6 | DOM-11 | `byIdEntityProvider`/`byRegionVersionEntityProvider` eager `.First()` wrapped in `model.FixedProvider` | **CLOSED** | `tenants/provider.go:23-30` and `:32-43` now return `database.Query[Entity](scope.Strict(db.WithContext(ctx), caller), map[string]any{...})` directly — no `.First(`, no `FixedProvider` anywhere in the file (`grep -n "FixedProvider\|\.First(" tenants/provider.go` → zero matches, confirmed via diff `9613e7259..HEAD -- tenants/provider.go`). The map-key filter is the mechanical equivalent of the prior chained `.Where(...)` calls; `scope.Strict` environment scoping is preserved verbatim, so the DOM-31 multi-tenancy analysis in the baseline audit is unaffected. |
| 7 | DOM-28 | `baselineFor` degrades a cross-domain fetch failure with only a `Warn` log, no metric | **CLOSED** | `tenants/processor.go:151-161`: the non-`ErrRecordNotFound` branch logs `p.l.WithError(err)....Warn(...)` (line 152-156) and calls `degrade.Observe(p.l, "configurations.tenants.templates", 0, err)` (line 161), matching the pattern doc's `degrade.Observe(l, "<svc>.<domain>.<enrichment>", id, err)` shape (patterns-resilience.md:125). Covered by new tests `TestBaselineFor_DegradesLoudlyOnNonNotFoundError` and `TestBaselineFor_NoBaselineIsNotADegradation` (`tenants/processor_test.go:891,926`). |

**6 of 7 in-scope findings closed. 1 remains open (SUB-04).**

## New findings introduced by the two remediation commits

None found. Specifically checked and clear:

- **DOM-27** (no direct `w.WriteHeader(http.StatusInternalServerError)` in changed handlers): `grep -n "StatusInternalServerError" tenants/resource.go` → zero matches.
- **FILE-06** (no catch-all file): `normalizeResetBody`/`hasResetEnvelope` (new in `resource.go`) are HTTP-layer input normalization, resource.go's own designated responsibility — not a second copy of Processor/RestModel/requests/Entity/Builder logic.
- **DOM-33** (mocks track interface changes): no `Processor` interface method changed in either remediation commit (`ResetById`/`GetById` signatures unchanged); `go build ./...` and `go test ./...` both pass, which would fail on a stale mock against a changed interface.
- **DOM-14** (handlers call processor methods only): the new pre-read at `resource.go:284` (`NewProcessor(...).GetById(tenantId)`) calls a processor method, not a provider directly — same shape as the pre-existing PATCH handler.
- **SUB-02/DOM-15** (no `db.Create/Save/Delete` in resource.go): unaffected by remediation; `grep -n "db\.Create\|db\.Save\|db\.Delete" tenants/resource.go` → zero matches.

## Not evaluable from the diff

none — both remediation commits are fully contained in the previously-audited `tenants` package plus `tenants/processor_test.go`/`tenants/resource_reset_test.go`, all read in full for this pass.

## Summary

### Blocking (must fix)
- SUB-04: `tenants/resource.go:76` (`io.ReadAll(r.Body)`) and `tenants/resource.go:131` (`json.Unmarshal(raw, &doc)`) inside `normalizeResetBody`/`hasResetEnvelope` — manual JSON/body parsing in `resource.go`, the exact three symbols SUB-04's grep-based pass criterion forbids there. The parsing was moved from the deleted `parseResetSections` into a new pre-`RegisterInputHandler` middleware rather than eliminated; the rule's literal bar (zero matches for `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` in `resource.go`) is not met regardless of the now-legitimate purpose (JSON:API-shaped 400s for envelope-less bodies that `jsonapi.Unmarshal` itself can't produce).

### Non-Blocking (should fix)
- Carried forward, unrelitigated (out of scope by explicit ruling): DOM-04/DOM-05 absent in `tenants` and DOM-04 absent in `templates/characters/preset` — no `Transform`/`TransformSlice`, confirmed still absent (`grep -rn "^func Transform" tenants/*.go templates/characters/preset/*.go` → zero matches).
- Carried forward from baseline, unaffected by remediation: `drift/crosstype_test.go` and `templates/characters/preset/rest_test.go` multi-scenario tests without `t.Run`/table shape (DOM-20 WARN); redundant `GetById` pre-read in `handleResetConfigurationTenant` (`resource.go:284`) duplicating `ResetById`'s internal read.
