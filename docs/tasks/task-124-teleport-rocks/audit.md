# Task 124 — Teleport Rocks — Code Review Summary

Pre-PR code review (project Code Review Pattern): `plan-adherence-reviewer` +
`backend-guidelines-reviewer` dispatched in parallel. No TypeScript changed, so
no frontend review. Detailed findings in the companion files:

- [`audit-plan-adherence.md`](audit-plan-adherence.md)
- [`audit-backend.md`](audit-backend.md)

## Plan adherence — PASS

23/23 tasks faithfully executed, each with file:line evidence. All build/test
gates green (`go build`/`go vet`/`go test -race` in every changed module; docker
bake ×4; `matrix`/`operations`/`gate-check`/`doc-freshness`/`dispatcher-lint
--check` exit 0; redis/goroutine/service-registration guards clean; kustomize
overlays build). Task 22 exceeded the plan's conservative bar — all five IDB
versions (v83/v84/v87/v95/jms) were verified live rather than stopping at
v83/v95; gms_92 remains the sanctioned unverified exception (no v92 IDB, and it
is not a coverage-matrix column).

Beyond-plan work (correct, not a defect): commit `8c91089ba` fixed a real
cross-version bug discovered during Task 22 live-IDA verification — the cash
teleport-rock sub-body reserved a phantom trailing `updateTime` on v87+/v95 that
would have mis-decoded by-map payloads. Fix gates the trailing byte on
`updateTimeFirst` (`MajorVersion()>=87`), threaded consistently: `use.go` passes
`hasTrailingUpdateTime=true` (USE op always has it); the cash body passes
`!updateTimeFirst`.

## Backend guidelines — 2 fixed, 3 adjudicated

**Fixed** (commit `221c0e4ef`, both match existing sibling packages):

- **DOM-27** — `teleport_rock/resource.go` now uses
  `server.WriteErrorResponse(d.Logger())(w)(err)` (transient-error → 503) instead
  of a bare `WriteHeader(500)`, matching `saved_location/resource.go`.
- **EXT-01** — channel `character/teleportrock/rest.go` `RestModel` now defines
  the required `SetToOneReferenceID` / `SetToManyReferenceIDs` no-op stubs,
  matching `character/rest.go`.

**Adjudicated (not fixed, with reasoning):**

- **DOM-28 (degrade.Observe) — rejected.** `libs/atlas-rest/degrade` is used
  nowhere in atlas-channel, and the existing enrichment fail-open sites use the
  same `WithError(...).Warnf(...)` pattern the teleport-rock code follows.
  Adopting `degrade` only here would make it a lone inconsistent outlier; the
  code correctly follows the service's actual convention.
- **EXT-02 (httptest REST test) — downgraded to Minor.** Sibling channel
  read-client packages have no httptest REST tests either (they use
  drain-provider tests, which don't apply to a single-resource GET);
  `model_test.go` is comparable coverage to siblings. Nice-to-have, not blocking.
- **Multitenancy explicit-tenantId (Important → Note).** Plan-mandated by the
  Task 8 brief, tenant-isolation-tested (`TestTenantIsolation`), and the sibling
  `saved_location` administrator's *writes* pass explicit `tenantId` the same
  way. Only the read-side `WHERE tenant_id = ?` is redundant with the
  `WithContext` tenant callback — it can only narrow, never leak. Left as-is:
  changing it ripples through the processor + `DeleteForCharacter`'s signature +
  the Task 12 caller for a purely stylistic gain over working, tested code.

**Minor (left, per reviewer's own justification):** builder `Build()` returns no
error (matches all sibling builders); no `Make/ToEntity` on the aggregate entity
(justified by the multi-row-to-one-Model shape); no character-delete cascade
test (the direct `DeleteForCharacter` unit test covers the behavior).

## Verdict

Branch is green and PR-ready. Post-merge rollout (live-tenant config PATCH +
channel restart) is documented in `plan.md` §Deploy/Rollout.
