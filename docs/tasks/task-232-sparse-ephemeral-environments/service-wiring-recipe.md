# Service-wiring recipe

Established on `atlas-monsters` (Task 30). Tasks 31–40 apply this recipe
**verbatim** to the rest of the fleet — one service (or small batch) per
task. It is a mechanical recipe: no domain package is touched, no business
logic changes, only the three wiring points below.

**Prerequisite:** `SERVICE_NAME` must already be provisioned on the target
service's Deployment (Task 29A landed this fleet-wide). Registering
`EnvHeaderParser` on a domain consumer before `SERVICE_NAME` exists opens a
silent-misroute window, because `WithEnvironmentRegistry`'s consumer group id
is derived from it.

## Step 1: Find every site in the service

```sh
S=services/atlas-<svc>/atlas.com/<name>
grep -rn "service.Bootstrap"        "$S"
grep -rn "SetHeaderParsers"         "$S"
grep -rn "requests.RootUrl("        "$S"
```

Record the three counts before editing. They are this batch's checklist —
every counted site must be touched, and the counts should match the number
of edits made.

**atlas-monsters counts** (recorded for calibration):

| Grep | Sites found |
|---|---|
| `service.Bootstrap` | 1 |
| `SetHeaderParsers` | 5 (across 4 files: monster×2, buff×1, map×1, data×1) |
| `requests.RootUrl(` | 6 (across 6 files: monster/information, monster/drop, map, monster/mobskill, monster/consumable, character/buff) |

Do not assume every service's counts look like this — `SetHeaderParsers`
scales with the number of consumer topics, `requests.RootUrl(` with the
number of outbound REST clients. Some services will have zero of one kind
(e.g. a service with no outbound REST calls has zero `RootUrl(` sites).

## Step 2: Write the failing wiring test

Copy this file verbatim into the target service's module root as
`wiring_test.go`, in `package main`:

```go
// services/atlas-<svc>/atlas.com/<name>/wiring_test.go
package main

import (
	"os"
	"strings"
	"testing"
)

// TestMainWiresTheEnvironmentRegistry pins the one line every service must
// carry. It is a source assertion rather than a behavioural one because the
// wiring's effect is inert until an Environment record exists (FR-1.8), so
// there is nothing observable to assert at this point in the migration.
func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
```

This duplicates what the `env-bootstrap-guard` repo guard checks fleet-wide,
deliberately: the guard tells the whole fleet's story at CI time; this test
fails the batch's own `go test ./...` immediately, before the guard ever
runs.

## Step 3: Run it, confirm it fails

```sh
cd services/<svc>/atlas.com/<name> && go test . -run TestMainWires -v
```

Expected: `FAIL` — `main.go does not pass service.WithEnvironmentRegistry to
Bootstrap`.

## Step 4: Apply the three edits

### 4a. `main.go` — `service.Bootstrap`

```go
// before
rt := service.Bootstrap(serviceName)
// after
rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
```

`serviceName` is the existing `const serviceName = "atlas-<svc>"` (or
equivalent) already declared in `main.go` — do not introduce a new constant.

### 4b. Every `kafka/consumer/**/*.go` that calls `SetHeaderParsers`

```go
// before
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser)
// after — EnvHeaderParser must come AFTER TenantHeaderParser so the tenant
// is on the context when it reconciles (FR-7.7).
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser)
```

**Ordering rule — do not reorder this.** `EnvHeaderParser`
(`libs/atlas-kafka/consumer/header.go`) reads the `ENVIRONMENT` header and
calls `env.Reconcile(env.CurrentRegistry(), id, tenantId)`, where `tenantId`
comes off the context via `tenant.FromContext(ctx)`. `TenantHeaderParser`
is what puts the tenant on the context in the first place. Header parsers run
in the order passed to `SetHeaderParsers`, each threading `ctx` into the
next — so `EnvHeaderParser` before `TenantHeaderParser` would reconcile
against an empty tenant every time, silently disabling FR-7.7 tenant-derived
environment resolution for that consumer while looking fully wired.

A message with no `ENVIRONMENT` header still decodes correctly — `id` stays
the zero value and `env.Reconcile` resolves it against the registry the same
way it always has. This is what makes it safe to roll out per-service ahead
of the environment topic actually publishing anywhere.

Touch **only** the `consumer.SetHeaderParsers(...)` argument list. Do not
touch the handler functions, the topic constants, or `InitHandlers` in the
same file.

### 4c. Every `**/rest.go` (or `requests.go`) that calls `requests.RootUrl(`

The exact shape of the call site varies (inline in `fmt.Sprintf`, or via a
`getBaseRequest()` helper — atlas-monsters had both). Adapt mechanically to
whichever shape the target file uses; the invariant is: **the base-URL
resolution must take `ctx` and propagate a resolution error to the caller
instead of ever falling back to the baseline URL.**

**Pattern A — inline call, no helper** (the shape in the brief):

```go
// before
func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, requests.RootUrl("MONSTERS"), id))
}
// after — the environment on ctx decides which ingress this call targets.
func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := requests.RootUrlFor(ctx, "MONSTERS")
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, root, id))
}
```

**Pattern B — `getBaseRequest()` helper** (the shape atlas-monsters actually
uses in all six of its call sites):

```go
// before
func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(monsterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monsterResource, monsterId))
}
// after
func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestById(ctx context.Context, monsterId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+monsterResource, monsterId))
}
```

**Pattern C — a bare-URL helper feeding `requests.DrainProvider`/pagination**
(atlas-monsters has three of these — the upstream list is paginated and
`DrainProvider` takes a raw `string`, not a `Request[T]`, so there is no
`Request` to wrap in `ErrorRequest`):

```go
// before
func monsterDropsUrl(monsterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+monsterDropsResource, monsterId)
}
// ...at the call site (processor.go):
return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(monsterDropsUrl(monsterId), 250, Extract, model.Filters[Model]())()

// after
func monsterDropsUrl(ctx context.Context, monsterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+monsterDropsResource, monsterId), nil
}
// ...at the call site:
url, err := monsterDropsUrl(p.ctx, monsterId)
if err != nil {
	return nil, err // or model.ErrorProvider[T](err) if the caller returns a model.Provider
}
return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())()
```

Every call site of a changed `getBaseRequest`/`requestXxx`/`xxxUrl` function
needs its `ctx` argument threaded through — grep for the function name
after editing its definition to find every caller. In every processor seen
so far, `p.ctx` (the `context.Context` field already carried by the
`ProcessorImpl`) was already available at the call site; no new parameter
threading through unrelated call chains was needed. If a target service's
caller genuinely has no `ctx` in scope, stop and flag it — do not invent one
(e.g. `context.Background()`), since that silently defeats environment
routing for that one call.

**`requests.ErrorRequest[T]` may not exist yet.** It was added to
`libs/atlas-rest/requests/get.go` as part of Task 30:

```go
// ErrorRequest returns a Request that always fails with err, without issuing
// any call. Used when a request cannot even be constructed — e.g.
// RootUrlFor could not resolve the caller's environment to an ingress
// (FR-3.5, G4) — so there is no URL to build a real Request from.
//
//goland:noinspection GoUnusedExportedFunction
func ErrorRequest[A any](err error) Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		var zero A
		return zero, err
	}
}
```

If it is already present when a later batch runs (it lands with Task 30),
do not redefine it — just use it.

### 4d. "No domain package is touched" rule

This recipe's edits are confined to exactly three shapes: the `Bootstrap`
call in `main.go`, the `SetHeaderParsers` argument list in
`kafka/consumer/**`, and the base-URL resolution in `**/rest.go` /
`**/requests.go` plus their immediate callers' `ctx` threading. If applying
the recipe to a service seems to require touching a handler body, a
processor's business logic, a model, or anything outside those three shapes,
stop — that service's wiring diverges from the pattern established here and
needs a decision, not a mechanical edit. Do not improvise a workaround inside
this batch.

## Step 5: Run the module tests

```sh
cd services/<svc>/atlas.com/<name> && go build ./... && go test ./...
```

Expected: `PASS`, including the new `TestMainWiresTheEnvironmentRegistry`.

Also run lint before committing — a `go build`/`go test` pass does not catch
an unused-after-refactor helper (atlas-monsters had one: a
`requests.Request[T]`-returning wrapper that turned out to have no caller
anywhere in its package once its signature changed to take `ctx`; `staticcheck`'s
`unused` check flagged it where it had not before the edit, because the edit
made the function's shape distinct enough to analyze on its own merits — the
fix was to delete the dead wrapper, not silence the lint):

```sh
tools/lint.sh services/<svc>/atlas.com/<name>
```

## Step 6: Commit

```bash
git add services/<svc> [libs/atlas-rest if ErrorRequest was newly added] docs/tasks/task-232-sparse-ephemeral-environments/<batch-report>.md
git commit -m "feat(<svc>): wire the environment registry (task-232 service-wiring recipe)"
```

Do not `git add -A` — a target worktree may carry unrelated in-flight
changes from another task's review round. Add only the paths this batch
touched.

## Verification scope for each batch

Module-local only, per service:

```sh
cd services/<svc>/atlas.com/<name> && go build ./... && go test ./...
```

Repo-wide `tools/verify.sh` is the controller/verifier's job, not the
implementer's, per the standard budget split.
