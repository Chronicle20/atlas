# Brief: admit PROVISIONING environments through the REST environment gate

**Branch:** `fix-provisioning-env-self-writes`
**Origin:** third defect surfaced by the first sparse ephemeral environment
(PR #1411). See also #1416, #1418, and issue #1420.

## The failure

`atlas-pr-bootstrap`'s sparse `create_service_config` POSTs to
`/api/configurations/services` with `-H "ENVIRONMENT: pr-1411"` (added in
#1418, because the row's `environment` column is server-owned and a
header-less caller is stamped as the legacy `""` environment, which leaks
rows the teardown reclaim can never match).

Every such POST now returns **400**. From atlas-configurations:

```json
{"environment":"pr-1411","log.level":"error","originator":"create_service_configuration",
 "message":"Request names an unknown or inactive environment."}
```

The environment exists but is in `PROVISIONING`:

```json
{"name":"pr-1411","baseline":"main","namespace":"atlas-pr-1411",
 "overrides":{"atlas-channel":"atlas-pr-1411","atlas-login":"atlas-pr-1411"},
 "phase":"PROVISIONING"}
```

## Root cause

`libs/atlas-rest/server/handler.go:41-54`, `ParseEnvironment`:

```go
id := env.Id(r.Header.Get(env.Key))
if id != "" && !env.CurrentRegistry().IsActive(id) {
    l.WithField(env.Key, string(id)).Error("Request names an unknown or inactive environment.")
    w.WriteHeader(http.StatusBadRequest)
    return
}
```

`IsActive` is `ok && rec.Active()`, and `Record.Active()` is
`r.Phase == PhaseActive` (`libs/atlas-env/record.go:28`). So a known
environment still in `PROVISIONING` is rejected exactly like an unknown one.

This is a genuine ordering conflict, not a typo.
`deploy/k8s/overlays/pr-sparse/environment-record.yaml:1-6` states the
intended lifecycle: the record is created in `PROVISIONING`, then "Task 45's
offset seeding and the override rollout happen next; the phase is flipped to
ACTIVE **last**." But bootstrap must write its service-config rows *during*
provisioning — that write **is** part of provisioning. Under the current gate
those two requirements cannot both hold.

(Note: nothing in the repo actually performs the flip to `ACTIVE` either —
see issue #1420. That is filed separately and is NOT in scope here. Do not
implement activation on this branch.)

## The decision (already made — implement this, do not re-litigate)

Relax the gate so an environment may perform **self-writes while
PROVISIONING**. Concretely: `ParseEnvironment` admits a request whose
environment is *known* and in `PROVISIONING` or `ACTIVE`, and continues to
reject unknown environments, `DEACTIVATING`, and `DELETED`.

### Why this is safe

Admitting the request through `ParseEnvironment` does not grant broad access
— it only puts the id on the context. Two existing layers still confine it to
its own data:

- `scope.Strict` (`services/atlas-configurations/.../scope/scope.go`) filters
  every read to the caller's environment.
- `scope.AuthorizeWrite` rejects any write whose target row belongs to a
  different environment.

So a `PROVISIONING` environment can only read and write its **own** rows.

Traffic ownership is unaffected: that is governed by `Registry.IsOwner`, which
keeps using `rec.Active()` and must **not** change. FR-5.2's guarantee — that
during `PROVISIONING` baseline deployments still own the environment's
services, overrides receive no work, and the ingress does not route — is
enforced by `IsOwner`, not by this REST gate.

## Required changes

1. **`libs/atlas-env`** — add a way to ask "is this environment known and in a
   phase that may serve its own requests". Suggested shape, but use your
   judgement and match the file's conventions:
   - `Record.Provisionable() bool` (or similarly named) — true for
     `PhaseProvisioning` and `PhaseActive`.
   - `Registry.IsProvisionable(e Id) bool` mirroring `IsActive`'s structure,
     including the `e == ""` legacy short-circuit returning `true` (FR-1.8).
   - Adding a method to the `Registry` interface (`registry.go:16`) is a
     breaking change — find and update EVERY implementation, including the
     legacy/fallback registry referenced in `BaselineOf`'s doc comment and any
     test doubles across the repo. Do not leave a compile break in another
     module; this is a shared lib in a `go.work` monorepo.
2. **`libs/atlas-rest/server/handler.go`** — `ParseEnvironment` uses the new
   predicate instead of `IsActive`. Update its doc comment, which currently
   documents the old FR-3.6/D4 behaviour, to state precisely what is now
   admitted and why the scope layer is what confines it.
3. **`IsActive` must keep its exact current meaning** — `IsOwner`,
   `EnvironmentsOwnedBy` and every other caller depend on it. Do not redefine
   it.

## Tests (required)

- `libs/atlas-rest/server/handler_test.go` already covers this gate; see the
  FR-3.6 test near line 240. Add cases: a `PROVISIONING` environment is
  admitted (200/next-handler-invoked); `DEACTIVATING`, `DELETED`, and an
  unknown id are still rejected with 400; an absent header still passes
  through as legacy.
- `libs/atlas-env` — unit tests for the new record predicate and registry
  method across all four phases plus unknown and the empty legacy id.
- Confirm no existing test asserted that `PROVISIONING` is rejected *through
  this gate*; if one does, it is encoding the old decision — update it and say
  so explicitly in your report rather than deleting it silently.

## Constraints

- Do NOT implement the `ACTIVE` transition (issue #1420).
- Do NOT change `atlas-pr-bootstrap`; #1418's header is correct as written.
- Do NOT weaken `scope.Strict` or `scope.AuthorizeWrite`.
- Verification scope is module-local only: `go build ./...` and `go test ./...`
  in each module you touch (`libs/atlas-env`, `libs/atlas-rest`, and any
  module whose build you break by extending the interface). Repo-wide
  verification is handled separately — do not run `tools/verify.sh`.
- Preserve existing line endings; match surrounding comment density and style.
