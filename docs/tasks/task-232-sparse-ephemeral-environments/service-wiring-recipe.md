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
grep -rn "tenant.WithContext" --include='*.go' "$S" | grep -v '_test.go'
```

Record all four counts before editing. They are this batch's checklist —
every counted site must be touched (or, for `tenant.WithContext`,
classified — see Step 3b below), and the counts should match the number of
edits made.

**Always filter `tenant.WithContext` with `grep -v '_test.go'`.** The
unfiltered grep overstates the real surface by roughly 40x (most hits are
test fixtures building a tenanted context, not production origination
sites) — running Step 3b unfiltered sends every remaining batch into a
~300-site read for a real surface that is usually single digits. State a
batch's size as **conversion sites (Bootstrap + SetHeaderParsers +
RootUrl) + NON-TEST `tenant.WithContext` audit sites** — the unfiltered
count is not the batch's real scope.

**A clean `requests.RootUrl(` grep does not prove a package is fully
converted.** A package-level shared helper (`getBaseRequest()` /
`getDataBaseRequest()`) called by several `*_requests.go` files in one
package matches the grep only in the file that *defines* the helper — every
other file that *calls* it does not contain the literal string
`requests.RootUrl(` and will silently miss the conversion until `go build`
fails on the helper's changed signature (or, worse, doesn't fail if the
call site never got exercised). Before trusting a clean grep for a package,
check whether any of its `*_requests.go` files share a package-level base
URL helper and convert every caller together.

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

**Fix round 1 (added after the fleet survey):** atlas-monsters' 6
`requests.RootUrl(` sites cover only Patterns A/B/C below. A repo-wide
survey found two more call-site shapes elsewhere in the fleet —
**Pattern D** (19 files, the single most common shape) and **Pattern E**
(1 file, an explicit exception) — added as §4d/§4e. Read them before
starting a batch; do not assume your service only needs A/B/C because
atlas-monsters did.

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

### 4d. Pattern D — package-level `var baseURLProvider` + `SetBaseURLForTest`

This is **the most common shape in the fleet** — atlas-monsters happens not
to use it, which is why it's called out separately here rather than folded
into A/B/C. It appears in **19 files across 13 services** (atlas-channel,
atlas-login, atlas-pets, atlas-consumables, atlas-parties, atlas-maps,
atlas-doors, atlas-character, atlas-query-aggregator, atlas-monster-book,
atlas-mini-games), with `SetBaseURLForTest` called from **38 test sites**.
Verified directly against
`services/atlas-channel/atlas.com/channel/monsterbook/requests.go`,
`services/atlas-pets/atlas.com/pets/location/requests.go`, and
`services/atlas-doors/atlas.com/doors/data/map/requests.go` — all three
share the identical shape below, differing only in the domain string and
whether a `getBaseRequest()` wrapper sits between `baseURLProvider` and the
`fmt.Sprintf` call.

The `var`, not a `func`, is what makes this different from Patterns A/B/C:
"convert the function to take `ctx`" doesn't apply to a package variable
directly — the variable's closure type has to change instead.

**Decision: `baseURLProvider` survives.** It becomes a
`func(ctx context.Context) (string, error)` instead of a `func() string`.
`SetBaseURLForTest`'s exported signature —
`func SetBaseURLForTest(url string) func()` — is **unchanged**; only its
body is updated to match the var's new type. This means **none of the 38
test call sites need to change**. They keep calling
`defer SetBaseURLForTest(srv.URL)()` exactly as today, because the injected
closure ignores whatever environment happens to be on the test's `ctx` and
always resolves to the httptest URL — matching today's unconditional
override.

```go
// before
var baseURLProvider = func() string {
	return requests.RootUrl("MONSTER_BOOK")
}

func getBaseRequest() string {
	return baseURLProvider()
}

func requestByCharacterId(characterId character.Id) requests.Request[CollectionRestModel] {
	return requests.GetRequest[CollectionRestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only
// call from a test; production code uses the env-driven default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
```

```go
// after
var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MONSTER_BOOK")
}

func getBaseRequest(ctx context.Context) (string, error) {
	return baseURLProvider(ctx)
}

func requestByCharacterId(ctx context.Context, characterId character.Id) requests.Request[CollectionRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CollectionRestModel](err)
	}
	return requests.GetRequest[CollectionRestModel](fmt.Sprintf(root+Resource, characterId))
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only
// call from a test; production code uses the env-driven default. The
// injected closure ignores ctx — tests always exercise the fixed httptest
// URL regardless of any environment on the context.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
```

Some of the 19 files (e.g. `atlas-pets/location/requests.go`,
`atlas-channel/effective_stats/requests.go`) call `baseURLProvider()`
directly at the `fmt.Sprintf` site with no `getBaseRequest()` wrapper in
between. Same conversion, one fewer indirection layer: thread `ctx`
straight into `baseURLProvider(ctx)` at that call site and handle the error
the same way Patterns A/B/C do — `requests.ErrorRequest[T]` for a
`Request[T]`-returning function, propagate `(string, error)` for a bare-URL
helper feeding `DrainProvider`.

Every file using this pattern already imports `"context"` for the
`Request[T]` closure's `ctx` param; the import is not new.

### 4e. Pattern E — the exception: raw `http.DefaultClient`, and its paired decorator gap

`services/atlas-character-factory/atlas.com/character-factory/character/name_validity_requests.go`
bypasses `requests.Request[T]` entirely. It builds an `*http.Request` and
calls `http.DefaultClient.Do` directly, because atlas-character's
`GET /characters/name-validity` returns plain JSON, not a JSON:API envelope,
and `requests.GetRequest[T]` unconditionally calls `jsonapi.Unmarshal` on
the body. **Do not force this file into Pattern A/B/C/D's
`Request[T]`/`ErrorRequest[T]` shape** — it would break decoding. Convert
only the base-URL resolution and the header decoration, matching the
`Request[T]` machinery's behavior by hand:

```go
// before
func getCharacterBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func (c *NameValidityClientImpl) Check(ctx context.Context, name string, worldId byte) (NameValidityResult, error) {
	base := getCharacterBaseRequest()
	u := fmt.Sprintf("%s%s?name=%s&worldId=%d", base, nameValidityPath, url.QueryEscape(name), worldId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return NameValidityResult{}, err
	}

	requests.SpanHeaderDecorator(ctx)(req.Header)
	requests.TenantHeaderDecorator(ctx)(req.Header)
	// ... c.l.Debugf, http.DefaultClient.Do, status/decode handling unchanged
```

```go
// after
func getCharacterBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHARACTERS")
}

func (c *NameValidityClientImpl) Check(ctx context.Context, name string, worldId byte) (NameValidityResult, error) {
	base, err := getCharacterBaseRequest(ctx)
	if err != nil {
		return NameValidityResult{}, err
	}
	u := fmt.Sprintf("%s%s?name=%s&worldId=%d", base, nameValidityPath, url.QueryEscape(name), worldId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return NameValidityResult{}, err
	}

	requests.SpanHeaderDecorator(ctx)(req.Header)
	requests.TenantHeaderDecorator(ctx)(req.Header)
	requests.EnvHeaderDecorator(ctx)(req.Header)
	// ... c.l.Debugf, http.DefaultClient.Do, status/decode handling unchanged
```

**This file needs a second, independent edit alongside the base-URL
conversion — and that second edit is not new to this task.**
`.superpowers/sdd/plan/progress.md` (Task 23, "Phase C deferral", also
restated at its Task-25-adjacent summary) already flags that this file and
`services/atlas-login/atlas.com/login/ranking/requests.go` hand-roll their
own header decoration (`SpanHeaderDecorator` + `TenantHeaderDecorator`)
instead of going through `requests.GetRequest[T]`/`PagedGetRequest[T]`,
which apply `EnvHeaderDecorator` automatically (see
`libs/atlas-rest/requests/decorated.go` and `paged.go:65`). Task 23
deliberately did **not** add `EnvHeaderDecorator` to either file — it was
Step 4's scope boundary — and named both as the Phase C wiring batches'
responsibility, i.e. **this recipe's**. Whichever batch touches either file
must add the `EnvHeaderDecorator` call next to the existing manual
Span/Tenant decorator calls, in addition to the `RootUrlFor` conversion.
Do not close out either file's site count on the base-URL fix alone — a
missing `EnvHeaderDecorator` is not a compile error, it's a silent
cross-environment leak (FR-3.1/FR-3.2): the request goes out with no
`ENVIRONMENT` header and lands wherever `BASE_SERVICE_URL` points, same as
before this whole migration.

`ranking/requests.go` is **not** Pattern E — it does return
`requests.Request[T]` (via `requests.MakeGetRequest` plus hand-rolled
`requests.AddHeaderDecorator(requests.SpanHeaderDecorator(ctx))` /
`...TenantHeaderDecorator(ctx))` configurators), so its base-URL half
follows Pattern B like any other `getBaseRequest()` site. It's called out
here only because it shares Pattern E's missing-`EnvHeaderDecorator` defect
and the same `progress.md` tracking entry. A plain
`grep -rn "requests.RootUrl(" $S` will still find it and its base-URL
conversion will look identical to any other Pattern B site — but the
decorator gap will not show up in that grep at all, so the batch that owns
`atlas-login` must add
`requests.AddHeaderDecorator(requests.EnvHeaderDecorator(ctx))` to
`requestByCharacterIds`'s configurator list by hand, from this note, not
from the grep.

### 4f. "No domain package is touched" rule

This recipe's edits are confined to exactly three wiring points: the
`Bootstrap` call in `main.go`, the `SetHeaderParsers` argument list in
`kafka/consumer/**`, and the base-URL resolution in `**/rest.go` /
`**/requests.go` plus their immediate callers' `ctx` threading — whichever
of Patterns A/B/C/D/E that resolution takes, and Pattern E's paired
`EnvHeaderDecorator` addition. If applying the recipe to a service seems to
require touching a handler body, a processor's business logic, a model, or
anything outside those shapes, stop — that service's wiring diverges from
every pattern established here and needs a decision, not a mechanical edit.
Do not improvise a workaround inside this batch.

## Step 5: Confirm every counted site was actually touched

Before running tests, re-run Step 1's greps against the same `$S` and
compare against the counts you recorded:

```sh
grep -rn "service.Bootstrap"        "$S"   # unchanged count — this edit doesn't remove the call, only adds an argument
grep -rn "SetHeaderParsers"         "$S"   # unchanged count — same reason
grep -rn "requests.RootUrl("        "$S"   # MUST be zero
```

`requests.RootUrl(` is the one that matters: `RootUrl` itself is not
deleted (`RootUrlFor` is a sibling function in the same file,
`libs/atlas-rest/requests/url.go`), so a missed call site still compiles,
its test still passes, and `go vet` says nothing — it just keeps routing
that one client's traffic through `BASE_SERVICE_URL` regardless of the
environment on `ctx`. That is exactly the silent-misroute failure mode this
whole phase exists to close, and it produces no error anywhere. A batch is
not done until its `requests.RootUrl(` grep returns nothing in `$S`. If it
doesn't, find the leftover call by pattern letter, not by re-grepping
blind — a Pattern D leftover means an unconverted `baseURLProvider`, a
Pattern E leftover means the two `progress.md`-tracked files (§4e), and any
other unexplained match means a shape this recipe doesn't cover yet and
should be reported back rather than force-fit.

## Step 6: Run the module tests

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

## Step 7: Commit

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
