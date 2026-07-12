# Safe Goroutine Helper (RR-6) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce `libs/atlas-routine` with a single recover-wrapping spawn helper `routine.Go(l, ctx, fn)`, migrate every bare `go` statement in non-test code under `services/` and `libs/` onto it, and enforce the ban with an AST-analyzer CI guard plus a DOM-25 guidelines item.

**Architecture:** New tiny Go module `libs/atlas-routine` (spawn + recover + structured Error log, nothing else). A `golang.org/x/tools/go/analysis` analyzer at `tools/goroutineguard` (twin of `tools/rediskeyguard`) flags every `*ast.GoStmt` outside the helper lib, `_test.go` files, and sites carrying a justified `//goroutine-guard:allow` marker; `tools/goroutine-guard.sh` self-tests the analyzer, then sweeps every module under `services/` **and** `libs/`. Migration is mechanical (body byte-identical inside the wrapped closure) except two protocol-heavy sites resolved explicitly in Tasks 4–5.

**Tech Stack:** Go 1.25.5 modules, `github.com/sirupsen/logrus v1.9.4`, `golang.org/x/tools v0.47.0` (analyzer + `analysistest`), GitHub Actions (`.github/workflows/pr-validation.yml`).

## Global Constraints

- Design doc (normative for the helper implementation, guard semantics, and per-site resolutions): `docs/tasks/task-115-safe-goroutine-helper/design.md`. PRD: `prd.md` in the same folder.
- **Zero behavior change** apart from panic containment: migrated bodies stay byte-identical inside the closure; `defer wg.Done()`, semaphore releases, and `defer close(ch)` move inside the closure unchanged (design §4.1).
- Panic log line is fixed and greppable: message `Recovered panic in background goroutine.`, Error level, fields `panic` (via `fmt.Sprintf("%v", r)`) and `stack` (`string(debug.Stack())`).
- `ctx` is pass-through only — the helper never inspects or cancels it (design §2.2).
- Only permitted bare `go` statements after migration: inside `libs/atlas-routine` and the **single** allowlisted site `libs/atlas-model/testutil/helpers.go` (`ConcurrentRunner.Go`).
- `safeHandle` in `libs/atlas-kafka/consumer/manager.go` is **untouched** (inline recovery around a synchronous call, not a spawn).
- `_test.go` files are never migrated and never flagged.
- New lib wiring = one `go.work` line + **three** root-`Dockerfile` edits (two `COPY` lines **and** the synthesized-go.work `for L in ...` loop at `Dockerfile:91` — the Dockerfile's own comment at line 15 mandates the loop edit; design §2.4 lists only the COPY lines, the Dockerfile is authoritative).
- `tools/goroutineguard` is a standalone `GOWORK=off` module, **not** added to `go.work` (matches `tools/rediskeyguard`, which is also absent from `go.work`).
- Every migrated module's `go.mod` gains `require github.com/Chronicle20/atlas/libs/atlas-routine v0.0.0-00010101000000-000000000000` + a relative `replace` (`../atlas-routine` from libs, `../../../../libs/atlas-routine` from services).
- Import the helper as `routine "github.com/Chronicle20/atlas/libs/atlas-routine"` (package name ≠ path tail, so always use the named import).
- Commit after every task. All commands below run from the worktree root `.worktrees/task-115-safe-goroutine-helper/` unless a `cd` is shown.
- Branch gate before "done": `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed module; `docker buildx bake all-go-services`; `tools/redis-key-guard.sh` AND `tools/goroutine-guard.sh` clean (CLAUDE.md Build & Verification).

## Mechanical Transform Rules (referenced by Tasks 4–10)

| Original form | Migrated form |
|---|---|
| `go func() { BODY }()` | `routine.Go(l, ctx, func(_ context.Context) { BODY })` — body byte-identical. Bind the parameter as `_` (bodies keep referencing their originally-captured ctx variable; never rewrite body references). |
| `go func(a T) { BODY }(x)` | Hoist the binding: `a := x` on the line before, then `routine.Go(l, ctx, func(_ context.Context) { BODY })` with BODY unchanged (it references `a`). |
| `go pkg.Fn(args...)` / `go expr(args...)` | `routine.Go(l, ctx, func(_ context.Context) { pkg.Fn(args...) })` |

Logger/ctx sourcing priority (design §4.2, record per-site in the audit table):
1. Both in scope → use them (handlers, `main.go` tickers, socket/rest/kafka libs).
2. In a service but not in scope → plumb from the nearest constructor/caller (every service has `l` and a root ctx in `main.go`; processors have `p.l`/`p.ctx`).
3. Shared lib with no logger in its public API (**only `libs/atlas-model`**) → `logrus.StandardLogger()` + the site's own ctx; `context.Background()` only where no ctx exists at all (only `SliceMap`'s parallel branch).

---

### Task 1: `libs/atlas-routine` module + repo wiring

**Files:**
- Create: `libs/atlas-routine/go.mod`
- Create: `libs/atlas-routine/routine.go`
- Create: `libs/atlas-routine/routine_test.go`
- Modify: `go.work` (add `./libs/atlas-routine` in the `use` block, alphabetically after `./libs/atlas-rest` / `./libs/atlas-retry`)
- Modify: `Dockerfile` (3 edits: mod-only COPY block ~line 43, source COPY block ~line 72, `for L in` loop line 91)

**Interfaces:**
- Produces: `routine.Go(l logrus.FieldLogger, ctx context.Context, fn func(context.Context))` at module path `github.com/Chronicle20/atlas/libs/atlas-routine`, package `routine`. Every later task imports exactly this.

- [ ] **Step 1: Create the module**

```bash
mkdir -p libs/atlas-routine
```

Write `libs/atlas-routine/go.mod`:

```
module github.com/Chronicle20/atlas/libs/atlas-routine

go 1.25.5

require github.com/sirupsen/logrus v1.9.4
```

- [ ] **Step 2: Write the failing tests**

Write `libs/atlas-routine/routine_test.go`:

```go
package routine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

type ctxKey string

// waitFor polls cond until it returns true or the deadline passes.
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(msg)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestGoRunsFnWithGivenContext(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := context.WithValue(context.Background(), ctxKey("k"), "v")
	got := make(chan context.Context, 1)
	routine.Go(l, ctx, func(c context.Context) { got <- c })
	select {
	case c := <-got:
		if c.Value(ctxKey("k")) != "v" {
			t.Fatalf("ctx not passed through unmodified: got %v", c.Value(ctxKey("k")))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fn never ran")
	}
}

func TestGoPanicDoesNotPropagate(t *testing.T) {
	l, _ := test.NewNullLogger()
	panicked := make(chan struct{})
	routine.Go(l, context.Background(), func(context.Context) {
		defer close(panicked)
		panic("boom")
	})
	<-panicked
	// Sibling work keeps running after a contained panic.
	ok := make(chan struct{})
	routine.Go(l, context.Background(), func(context.Context) { close(ok) })
	select {
	case <-ok:
	case <-time.After(2 * time.Second):
		t.Fatal("sibling goroutine did not run after a panic")
	}
}

func TestGoPanicIsLogged(t *testing.T) {
	l, hook := test.NewNullLogger()
	routine.Go(l, context.Background(), func(context.Context) {
		panic("kaboom-sentinel")
	})
	waitFor(t, func() bool { return hook.LastEntry() != nil }, "panic was not logged")
	e := hook.LastEntry()
	if e.Level != logrus.ErrorLevel {
		t.Fatalf("expected Error level, got %v", e.Level)
	}
	if e.Message != "Recovered panic in background goroutine." {
		t.Fatalf("unexpected message: %q", e.Message)
	}
	if p, _ := e.Data["panic"].(string); p != "kaboom-sentinel" {
		t.Fatalf("panic field = %q, want %q", p, "kaboom-sentinel")
	}
	stack, _ := e.Data["stack"].(string)
	if !strings.Contains(stack, "TestGoPanicIsLogged") {
		t.Fatalf("stack field missing this test's frame: %q", stack)
	}
}

func TestGoFnDefersRunBeforeRecoverLog(t *testing.T) {
	l, hook := test.NewNullLogger()
	deferRan := make(chan struct{})
	routine.Go(l, context.Background(), func(context.Context) {
		defer close(deferRan)
		panic("boom")
	})
	waitFor(t, func() bool { return hook.LastEntry() != nil }, "panic was not logged")
	// The log entry exists, so recover already ran; fn's defer must have run first.
	select {
	case <-deferRan:
	default:
		t.Fatal("fn's defer did not run before the helper's recover")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
cd libs/atlas-routine && go mod tidy && go test ./...
```

Expected: FAIL — `undefined: routine.Go` (compile error; `routine.go` doesn't exist yet).

- [ ] **Step 4: Write the implementation**

Write `libs/atlas-routine/routine.go` (normative shape from design §2.2 — copy exactly):

```go
package routine

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// Go runs fn in a new goroutine, recovering any panic. A recovered panic is
// logged at Error level with the panic value and full stack trace, then
// swallowed — the goroutine ends and the process continues. ctx is passed
// through to fn unmodified; Go itself never inspects or cancels it.
func Go(l logrus.FieldLogger, ctx context.Context, fn func(context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.WithField("panic", fmt.Sprintf("%v", r)).
					WithField("stack", string(debug.Stack())).
					Errorf("Recovered panic in background goroutine.")
			}
		}()
		fn(ctx)
	}()
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd libs/atlas-routine && go test -race ./... && go vet ./...
```

Expected: `ok  github.com/Chronicle20/atlas/libs/atlas-routine` with `-race`, vet clean.

- [ ] **Step 6: Wire go.work**

In `go.work`, add `./libs/atlas-routine` to the `use` block between `./libs/atlas-retry` and `./libs/atlas-saga`.

- [ ] **Step 7: Wire the root Dockerfile (3 edits)**

Edit 1 — mod-only block (after line 43 `COPY libs/atlas-retry/go.mod ...`, keeping alphabetical order):

```dockerfile
COPY libs/atlas-routine/go.mod     libs/atlas-routine/go.sum     libs/atlas-routine/
```

Edit 2 — source block (after `COPY libs/atlas-retry        libs/atlas-retry`):

```dockerfile
COPY libs/atlas-routine     libs/atlas-routine
```

Edit 3 — the synthesized go.work loop at line 91: insert `atlas-routine` between `atlas-retry` and `atlas-saga`:

```dockerfile
         for L in atlas-constants atlas-database atlas-kafka atlas-lock atlas-model \
                  atlas-object-id atlas-opcodes atlas-outbox atlas-packet atlas-redis \
                  atlas-rest atlas-retry atlas-routine atlas-saga atlas-script-core \
                  atlas-seeder atlas-service atlas-socket atlas-tenant atlas-tracing atlas-wz; do \
```

Also bump the stale lib counts in the Dockerfile comments ("all 20 atlas libs" → 21 at ~line 30; "the 18 libs" → 21 at ~line 82) so they don't drift further.

- [ ] **Step 8: Verify go.sum exists (Dockerfile COPY depends on it)**

```bash
ls libs/atlas-routine/go.sum
```

Expected: file exists (logrus has transitive deps, so `go mod tidy` in Step 3 created it). If it somehow doesn't, the mod-only COPY line must drop the go.sum operand (pattern: `libs/atlas-retry` line 43).

- [ ] **Step 9: Commit**

```bash
git add libs/atlas-routine go.work Dockerfile
git commit -m "feat(task-115): add libs/atlas-routine safe-spawn helper"
```

---

### Task 2: `tools/goroutineguard` analyzer

**Files:**
- Create: `tools/goroutineguard/go.mod`
- Create: `tools/goroutineguard/analyzer.go`
- Create: `tools/goroutineguard/analyzer_test.go`
- Create: `tools/goroutineguard/testdata/src/bad/bad.go`
- Create: `tools/goroutineguard/testdata/src/good/good.go`
- Create: `tools/goroutineguard/testdata/src/good/good_test.go`
- Create: `tools/goroutineguard/testdata/src/github.com/Chronicle20/atlas/libs/atlas-routine/routine.go`
- Create: `tools/goroutineguard/cmd/goroutineguard/main.go`

**Interfaces:**
- Consumes: nothing from other tasks (standalone `GOWORK=off` module).
- Produces: `goroutineguard.Analyzer` (`*analysis.Analyzer`, name `goroutineguard`) and the `cmd/goroutineguard` singlechecker binary that Task 3's shell wrapper builds and runs. Diagnostic strings (Task 3 greps them): `goroutineguard: bare go statement; use routine.Go from libs/atlas-routine (or add //goroutine-guard:allow <justification>)` and `goroutineguard: allow marker requires a justification`.

- [ ] **Step 1: Create the module**

Write `tools/goroutineguard/go.mod`:

```
module github.com/Chronicle20/atlas/tools/goroutineguard

go 1.25.5

require golang.org/x/tools v0.47.0
```

- [ ] **Step 2: Write the failing analysistest + fixtures**

Write `tools/goroutineguard/analyzer_test.go` (mirror of `tools/rediskeyguard/analyzer_test.go`):

```go
package goroutineguard_test

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/goroutineguard"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, goroutineguard.Analyzer,
		"bad", "good", "github.com/Chronicle20/atlas/libs/atlas-routine")
}
```

Write `tools/goroutineguard/testdata/src/bad/bad.go`:

```go
package bad

func named() {}

func spawn() {
	go func() {}() // want `goroutineguard: bare go statement`
	go named()     // want `goroutineguard: bare go statement`

	//goroutine-guard:allow
	go named() // want `goroutineguard: allow marker requires a justification`
}
```

Write `tools/goroutineguard/testdata/src/good/good.go`:

```go
package good

import "context"

func named() {}

// spawnLike mimics routine.Go's shape; ordinary calls are not go statements.
func spawnLike(fn func(context.Context)) { fn(context.Background()) }

//go:generate echo "go func() in a directive must not match"

const doc = "go func() inside a string literal must not match"

// A comment mentioning go func() and go named() must not match.

func fine() {
	spawnLike(func(context.Context) {})

	//goroutine-guard:allow fixture: marker on the line above with justification
	go named()

	go named() //goroutine-guard:allow fixture: trailing marker with justification
}
```

Write `tools/goroutineguard/testdata/src/good/good_test.go`:

```go
package good

func helperUsedOnlyInTests() {
	go named() // _test.go files are exempt; no diagnostic expected
}
```

Write `tools/goroutineguard/testdata/src/github.com/Chronicle20/atlas/libs/atlas-routine/routine.go`:

```go
package routine

// The helper lib itself is the only package allowed bare go statements.
func spawnInternal() {
	go func() {}()
}
```

- [ ] **Step 3: Run the test to verify it fails**

```bash
cd tools/goroutineguard && GOWORK=off go mod tidy && GOWORK=off go test ./...
```

Expected: FAIL — `undefined: goroutineguard.Analyzer` (analyzer.go doesn't exist yet).

- [ ] **Step 4: Write the analyzer**

Write `tools/goroutineguard/analyzer.go`:

```go
package goroutineguard

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	// The helper lib is the only package allowed to contain bare go statements.
	routinePkgPath = "github.com/Chronicle20/atlas/libs/atlas-routine"
	markerPrefix   = "//goroutine-guard:allow"
)

var Analyzer = &analysis.Analyzer{
	Name:     "goroutineguard",
	Doc:      "bans bare go statements outside libs/atlas-routine; spawn via routine.Go instead",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

type lineKey struct {
	file string
	line int
}

func run(pass *analysis.Pass) (interface{}, error) {
	if strings.HasPrefix(pass.Pkg.Path(), routinePkgPath) {
		return nil, nil
	}

	// Collect allow markers: file:line → justification present?
	markers := map[lineKey]bool{}
	for _, f := range pass.Files {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if !strings.HasPrefix(c.Text, markerPrefix) {
					continue
				}
				pos := pass.Fset.Position(c.Pos())
				justification := strings.TrimSpace(strings.TrimPrefix(c.Text, markerPrefix))
				markers[lineKey{pos.Filename, pos.Line}] = justification != ""
			}
		}
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.GoStmt)(nil)}, func(n ast.Node) {
		pos := pass.Fset.Position(n.Pos())
		if strings.HasSuffix(pos.Filename, "_test.go") {
			return
		}
		if justified, found := markerFor(markers, pos); found {
			if !justified {
				pass.Reportf(n.Pos(), "goroutineguard: allow marker requires a justification")
			}
			return
		}
		pass.Reportf(n.Pos(), "goroutineguard: bare go statement; use routine.Go from libs/atlas-routine (or add //goroutine-guard:allow <justification>)")
	})
	return nil, nil
}

// markerFor accepts a marker trailing on the statement's own line or on the
// line immediately above it.
func markerFor(markers map[lineKey]bool, pos token.Position) (justified bool, found bool) {
	if justified, found = markers[lineKey{pos.Filename, pos.Line}]; found {
		return justified, found
	}
	justified, found = markers[lineKey{pos.Filename, pos.Line - 1}]
	return justified, found
}
```

Write `tools/goroutineguard/cmd/goroutineguard/main.go`:

```go
package main

import (
	"github.com/Chronicle20/atlas/tools/goroutineguard"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(goroutineguard.Analyzer)
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd tools/goroutineguard && GOWORK=off go test -race ./... && GOWORK=off go vet ./... && GOWORK=off go build ./cmd/goroutineguard
```

Expected: PASS (all fixtures), vet clean, binary builds.

- [ ] **Step 6: Commit**

```bash
git add tools/goroutineguard
git commit -m "feat(task-115): goroutineguard AST analyzer banning bare go statements"
```

---

### Task 3: `tools/goroutine-guard.sh` wrapper, pre-migration baseline, audit-table skeleton, CI wiring

**Files:**
- Create: `tools/goroutine-guard.sh` (mode 755)
- Create: `docs/tasks/task-115-safe-goroutine-helper/migration-audit.md`
- Modify: `.github/workflows/pr-validation.yml` (new job after the `redis-key-guard` job ending ~line 98; `needs:` list ~line 480; results block ~lines 495–518)

**Interfaces:**
- Consumes: `tools/goroutineguard` (Task 2).
- Produces: `./tools/goroutine-guard.sh` — exit 0 = clean tree; used as the completion oracle by Tasks 4–11 and the branch gate. `migration-audit.md` skeleton whose rows Tasks 4–10 fill.

- [ ] **Step 1: Write the wrapper**

Write `tools/goroutine-guard.sh` (structure copied from `tools/redis-key-guard.sh`, two deltas per design §3.2 — self-test first, sweep includes `libs/`):

```bash
#!/usr/bin/env bash
# Self-test the goroutineguard analyzer fixtures, build it once, then run it
# over every Go module under services/ and libs/. Non-empty diagnostics →
# non-zero exit. Run from the repo root. tools/ is deliberately not swept —
# the analyzer's own testdata must be allowed to contain bare go statements.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUARD_SRC="$ROOT/tools/goroutineguard"
BIN="$(mktemp -d)/goroutineguard"

echo "self-testing goroutineguard..."
( cd "$GUARD_SRC" && GOWORK=off go test ./... )

echo "building goroutineguard..."
( cd "$GUARD_SRC" && GOWORK=off go build -o "$BIN" ./cmd/goroutineguard )

rc=0
# Every Go module with a go.mod under services/ or libs/ is a guard target.
while IFS= read -r modfile; do
    moddir="$(dirname "$modfile")"
    echo "goroutineguard: $moddir"
    if ! ( cd "$moddir" && "$BIN" ./... ); then
        rc=1
    fi
done < <(find "$ROOT/services" "$ROOT/libs" -name go.mod -not -path '*/node_modules/*')

if [ "$rc" -ne 0 ]; then
    echo "goroutineguard: FAIL — bare go statements found (use routine.Go from libs/atlas-routine)"
fi
exit $rc
```

```bash
chmod +x tools/goroutine-guard.sh
```

- [ ] **Step 2: Run against the pre-migration tree and record the baseline**

```bash
./tools/goroutine-guard.sh 2>&1 | tee /tmp/task-115-guard-baseline.txt || true
grep -c 'goroutineguard: bare go statement' /tmp/task-115-guard-baseline.txt
```

Expected: non-zero exit; finding count **≈165** (a repo-root grep for statement-form `go` lines found 164 — the AST count is authoritative; anywhere in 160–170 is consistent, outside that range STOP and reconcile before proceeding). This validates FR-3.2/PRD acceptance "produced ~165 findings against the pre-migration tree". Record the exact number — it is the required row count for the audit table.

- [ ] **Step 3: Generate the audit-table skeleton**

Create `docs/tasks/task-115-safe-goroutine-helper/migration-audit.md` with a header and one row per finding (design §4.3):

```markdown
# task-115 Migration Audit

Pre-migration `tools/goroutine-guard.sh` findings: <N> (recorded <date>).
Every row must carry a disposition before the branch is done. Row count must equal <N>.

| # | file:line | form | classification | logger source | ctx source | disposition |
|---|---|---|---|---|---|---|
```

Populate the `file:line` and `form` columns mechanically from the baseline output (diagnostic positions are `path/file.go:LINE:COL`; form = `anon` for `go func`, `named-call` otherwise — check each flagged source line). A helper one-liner (adjust the sed prefix to the worktree root):

```bash
grep 'goroutineguard: bare go statement' /tmp/task-115-guard-baseline.txt \
  | sed "s|^$(pwd)/||" | awk -F: '{n++; printf "| %d | %s:%s | | | | | |\n", n, $1, $2}'
```

Fill `form` per row now (a `sed -n '<line>p' <file>` loop over the rows); leave classification/logger/ctx/disposition blank — Tasks 4–10 fill them as they migrate. Use repo-relative paths in the committed file (never absolute home paths).

- [ ] **Step 4: Wire CI**

In `.github/workflows/pr-validation.yml`:

(a) Insert a new job immediately after the `redis-key-guard` job (which ends at the `run: ./tools/redis-key-guard.sh` step ~line 98):

```yaml
  # ============================================
  # Goroutine Guard
  # Self-tests + builds the goroutineguard
  # analyzer and runs it over every Go module
  # under services/ and libs/. Fails on any bare
  # go statement outside libs/atlas-routine or a
  # justified //goroutine-guard:allow marker
  # (RR-6, task-115).
  # ============================================
  goroutine-guard:
    name: Goroutine Guard
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: goroutine guard
        run: ./tools/goroutine-guard.sh
```

(b) Add `goroutine-guard` to the `pr-validation-complete` job's `needs:` list (~line 480):

```yaml
    needs: [detect-changes, test-go-libraries, test-go-services, test-ui, build-docker, update-pr-overlay, redis-key-guard, goroutine-guard, gen-lb-ports]
```

(c) In the `Check results` step: add the result var next to `GUARD_RESULT` (~line 496):

```bash
          GOROUTINE_GUARD_RESULT="${{ needs.goroutine-guard.result }}"
```

add a summary-table row after the Redis Key Guard row:

```bash
          echo "| Goroutine Guard | $GOROUTINE_GUARD_RESULT |" >> $GITHUB_STEP_SUMMARY
```

and extend the failure `if` with `|| [ "$GOROUTINE_GUARD_RESULT" == "failure" ]`.

- [ ] **Step 5: Sanity-check the workflow edit and commit**

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/pr-validation.yml'))" && echo YAML-OK
# If PyYAML is unavailable in the environment, instead diff the new job block
# field-by-field against the adjacent redis-key-guard job (same structure).
git add tools/goroutine-guard.sh docs/tasks/task-115-safe-goroutine-helper/migration-audit.md .github/workflows/pr-validation.yml
git commit -m "feat(task-115): goroutine-guard.sh wrapper, CI job, migration audit baseline"
```

Expected: `YAML-OK`. (The guard job will fail on this branch's CI until migration completes — expected; the PR lands with the full migration.)

---

### Task 4: Migrate `libs/atlas-lock` (hand-rolled recover → completed-flag pattern)

**Files:**
- Modify: `libs/atlas-lock/leader.go:155-190` (both goroutines)
- Modify: `libs/atlas-lock/go.mod`
- Test: existing `libs/atlas-lock/leader_test.go` (notably the `panic-test` case at :288-320 asserting `lostTotal{name="panic-test",reason="panic"}` and lease release — must stay green unchanged)

**Interfaces:**
- Consumes: `routine.Go` (Task 1).
- Produces: `leader.go` with zero bare `go` statements and zero hand-rolled goroutine `recover()`; `lostReason`/`cancelLeader` panic semantics preserved (the existing test is the proof).

- [ ] **Step 1: Add the dependency**

```bash
cd libs/atlas-lock
go mod edit -require=github.com/Chronicle20/atlas/libs/atlas-routine@v0.0.0-00010101000000-000000000000 \
  -replace=github.com/Chronicle20/atlas/libs/atlas-routine=../atlas-routine
go mod tidy
```

- [ ] **Step 2: Replace the fn goroutine (leader.go:155-165) with the completed-flag pattern**

The current recover does three things: log (no stack), `setReason("panic")`, `cancelLeader()`. `routine.Go` takes over the logging (and adds the stack); the completed-flag keeps reason+cancel without a second recover (design §6.3 — copy exactly, variable names match the file):

```go
		routine.Go(le.cfg.log, leaderCtx, func(c context.Context) {
			defer close(fnDone)
			completed := false
			defer func() {
				if !completed {
					setReason("panic")
					cancelLeader()
				}
			}()
			fn(c)
			completed = true
		})
```

Add the import: `routine "github.com/Chronicle20/atlas/libs/atlas-routine"`.

Unwind order on panic (why the test stays green): fn's defers → completed-check defer (sets reason, cancels) → `close(fnDone)` → helper's recover logs with stack. `lostTotal` keeps reporting reason `panic`.

- [ ] **Step 3: Migrate the renewer goroutine (leader.go:167-190) mechanically**

```go
		routine.Go(le.cfg.log, leaderCtx, func(_ context.Context) {
			defer close(renewerDone)
			t := time.NewTicker(le.cfg.refreshInterval)
			defer t.Stop()
			for {
				select {
				case <-leaderCtx.Done():
					return
				case <-t.C:
					rerr := rl.Refresh(ctx, le.cfg.ttl, nil)
					if rerr == nil {
						continue
					}
					if errors.Is(rerr, redislock.ErrNotObtained) {
						setReason("renew_failed")
						le.cfg.log.WithError(rerr).Warnf("Lease lost during refresh for [%s].", le.name)
						cancelLeader()
						return
					}
					renewFailedTotal.WithLabelValues(le.name).Inc()
					le.cfg.log.WithError(rerr).Warnf("Renewal attempt failed for [%s] (transient).", le.name)
				}
			}
		})
```

Body byte-identical (still references `leaderCtx` and the outer `ctx` for `Refresh` — bind `_`, don't rewrite).

- [ ] **Step 4: Verify**

```bash
cd libs/atlas-lock && go test -race ./... && go vet ./... && go build ./...
grep -rn 'recover()' . --include='*.go' | grep -v _test.go
```

Expected: tests PASS (including the `panic-test` leader test), vet/build clean, and the grep returns **nothing** (no hand-rolled recover left in this lib).

- [ ] **Step 5: Update audit rows + commit**

Fill this lib's two rows in `migration-audit.md` (classification `lifecycle`; logger `le.cfg.log`; ctx `leaderCtx`; disposition `migrated`).

```bash
git add libs/atlas-lock docs/tasks/task-115-safe-goroutine-helper/migration-audit.md
git commit -m "refactor(task-115): atlas-lock leader onto routine.Go, completed-flag panic reason"
```

---

### Task 5: Migrate `libs/atlas-kafka` (consumer manager, 3 sites; `safeHandle` untouched)

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager.go` (sites at :145, :523, :558; **do not touch `safeHandle` at :577**)
- Modify: `libs/atlas-kafka/go.mod`

**Interfaces:**
- Consumes: `routine.Go` (Task 1).
- Produces: manager with zero bare `go` statements; per-message commit protocol (`sem`, `advanceCommit`) and handler fan-out (`handlerWg`) semantics unchanged.

- [ ] **Step 1: Add the dependency**

```bash
cd libs/atlas-kafka
go mod edit -require=github.com/Chronicle20/atlas/libs/atlas-routine@v0.0.0-00010101000000-000000000000 \
  -replace=github.com/Chronicle20/atlas/libs/atlas-routine=../atlas-routine
go mod tidy
```

- [ ] **Step 2: Migrate the three sites** (add import `routine "github.com/Chronicle20/atlas/libs/atlas-routine"`)

Site :145 (named call; `l`, `ctx` in scope):

```go
		routine.Go(l, ctx, func(_ context.Context) { con.start(l, ctx, wg) })
```

Site :523 (parameter-passing form — hoist the binding, body unchanged; semaphore release stays inside the closure):

```go
		sem <- struct{}{}
		p := pm
		routine.Go(l, ctx, func(_ context.Context) {
			defer func() { <-sem }()
			ok := c.processMessage(l, ctx, p.msg)
			p.ok.Store(ok)
			p.done.Store(true)
			advanceCommit()
		})
```

Site :558 (handler fan-out; use `handlerLogger`/`wctx` — both in scope; `handlerWg.Done` stays inside):

```go
	for id, h := range handlersCopy {
		var handle = h
		var handleId = id
		handlerWg.Add(1)
		routine.Go(handlerLogger, wctx, func(_ context.Context) {
			defer handlerWg.Done()
			cont, handlerErr := c.safeHandle(handle, handlerLogger, wctx, msg)
			if !cont {
				c.mu.Lock()
				delete(c.handlers, handleId)
				c.mu.Unlock()
			}
			if handlerErr != nil {
				hadError.Store(true)
				handlerLogger.WithError(handlerErr).Errorf("Handler [%s] failed.", handleId)
			}
		})
	}
```

`safeHandle` (:577-585) stays exactly as-is: it is inline recovery around a synchronous call with continue-on-panic handler policy, not a spawn (design §6.4).

- [ ] **Step 3: Verify**

```bash
cd libs/atlas-kafka && go test -race ./... && go vet ./... && go build ./...
```

Expected: PASS/clean. `handlerWg.Wait()` and the commit-ordering protocol are exercised by the existing manager tests.

- [ ] **Step 4: Update audit rows + commit**

Fill the 3 rows (classification `lib-internal`; logger `l`/`handlerLogger`; ctx `ctx`/`wctx`; disposition `migrated`).

```bash
git add libs/atlas-kafka docs/tasks/task-115-safe-goroutine-helper/migration-audit.md
git commit -m "refactor(task-115): atlas-kafka consumer manager onto routine.Go"
```

---

### Task 6: Migrate `libs/atlas-model` (6 sites, std-logger rule) + testutil allowlist marker

**Files:**
- Modify: `libs/atlas-model/model/processor.go` (sites :155, :167, :208, :220, :441)
- Modify: `libs/atlas-model/async/processor.go` (site :72)
- Modify: `libs/atlas-model/testutil/helpers.go` (site :189 — allow marker, NOT migrated)
- Modify: `libs/atlas-model/go.mod` (gains `atlas-routine` require+replace AND a direct `github.com/sirupsen/logrus v1.9.4` require — this module currently has no logrus dependency)

**Interfaces:**
- Consumes: `routine.Go` (Task 1); marker syntax from Task 2.
- Produces: combinators (`ExecuteForEachSlice/Map`, `SliceMap`, `async.AwaitSlice`) spawning via the helper with `logrus.StandardLogger()`; the **only** repo allowlist entry at `testutil/helpers.go`.

- [ ] **Step 1: Add dependencies**

```bash
cd libs/atlas-model
go mod edit -require=github.com/Chronicle20/atlas/libs/atlas-routine@v0.0.0-00010101000000-000000000000 \
  -replace=github.com/Chronicle20/atlas/libs/atlas-routine=../atlas-routine \
  -require=github.com/sirupsen/logrus@v1.9.4
go mod tidy
```

- [ ] **Step 2: Migrate `model/processor.go`** (imports: add `routine "github.com/Chronicle20/atlas/libs/atlas-routine"` and `"github.com/sirupsen/logrus"`)

Worker sites :155 (`ExecuteForEachSlice`) and :208 (`ExecuteForEachMap`) — per-item worker, `ctx` from the function's own `context.WithCancel`; shown for :155, :208 is identical with `f(key)(value)`:

```go
				routine.Go(logrus.StandardLogger(), ctx, func(_ context.Context) {
					defer wg.Done()
					select {
					case <-ctx.Done():
						return
					default:
						err := f(model)
						errChannels <- err
					}
				})
```

Closer sites :167 and :220 (identical in both functions):

```go
			routine.Go(logrus.StandardLogger(), ctx, func(_ context.Context) {
				wg.Wait()
				close(errChannels)
			})
```

Site :441 (`SliceMap` parallel branch — **no ctx exists in this function**; the one `context.Background()` case, design §4.2 rule 3):

```go
						routine.Go(logrus.StandardLogger(), context.Background(), func(_ context.Context) {
							parallelTransform(&wg, transformer, i, m, resCh)
						})
```

(`context` may need adding to this file's imports if not present at that point — it is already imported for `WithCancel`.)

- [ ] **Step 3: Migrate `async/processor.go:72`**

```go
		for _, provider := range providers {
			p := provider
			routine.Go(logrus.StandardLogger(), ctx, func(_ context.Context) {
				p(ctx, resultChannels, errChannels)
			})
		}
```

(The existing `p := provider` hoist already exists — keep it; body unchanged.)

- [ ] **Step 4: Allowlist `testutil/helpers.go:189`** (design §3.3 — the ONLY allow entry; do not migrate)

```go
// Go runs a function in a goroutine and tracks any errors
func (cr *ConcurrentRunner) Go(fn func() error) {
	//goroutine-guard:allow test-support: a swallowed panic here would convert a failing test into a silent pass; panic propagation is the desired behavior in test scaffolding
	go func() {
		defer cr.wg.Done()
		if err := fn(); err != nil {
			cr.mu.Lock()
			cr.errors = append(cr.errors, err)
			cr.mu.Unlock()
		}
	}()
}
```

- [ ] **Step 5: Verify**

```bash
cd libs/atlas-model && go test -race ./... && go vet ./... && go build ./...
```

Expected: PASS/clean. Documented behavioral consequence (design §6.1, accepted — record in the audit table notes, do NOT "fix"): a recovered worker panic means its error never reaches the combinator's error channel — `defer wg.Done()` still fires so nothing deadlocks; `ExecuteForEach*` returns nil for that item, `SliceMap` leaves a zero-value element, `async.AwaitSlice` times out with `ErrAwaitTimeout`. The Error log line is the detection path.

- [ ] **Step 6: Update audit rows + commit**

Fill 7 rows: 6 × (classification `lib-internal`; logger `logrus.StandardLogger()`; ctx as used; disposition `migrated`) + 1 × (`testutil/helpers.go:189`, classification `test-support`, disposition `allowlisted — panic propagation is the point of a test harness`).

```bash
git add libs/atlas-model docs/tasks/task-115-safe-goroutine-helper/migration-audit.md
git commit -m "refactor(task-115): atlas-model combinators onto routine.Go; allowlist testutil runner"
```

---

### Task 7: Migrate `libs/atlas-socket`, `libs/atlas-rest`, `libs/atlas-seeder` (7 sites)

**Files:**
- Modify: `libs/atlas-socket/server.go` (:125, :152, :173, :226) + `libs/atlas-socket/go.mod`
- Modify: `libs/atlas-rest/server/server.go` (:171, :186) + `libs/atlas-rest/go.mod`
- Modify: `libs/atlas-seeder/handlers.go` (:49) + `libs/atlas-seeder/go.mod`

**Interfaces:**
- Consumes: `routine.Go` (Task 1).
- Produces: three libs with zero bare `go` statements; no public API changes (all sites have `l`/`ctx` in scope — design §6.5).

- [ ] **Step 1: Add the dependency to each of the three modules**

```bash
for m in libs/atlas-socket libs/atlas-rest libs/atlas-seeder; do
  ( cd "$m" && go mod edit \
      -require=github.com/Chronicle20/atlas/libs/atlas-routine@v0.0.0-00010101000000-000000000000 \
      -replace=github.com/Chronicle20/atlas/libs/atlas-routine=../atlas-routine \
    && go mod tidy )
done
```

- [ ] **Step 2: Migrate `libs/atlas-socket/server.go`** (add the named `routine` import)

:125 (listener closer; `l`, `ctx` in scope):

```go
	routine.Go(l, ctx, func(_ context.Context) {
		<-ctx.Done()
		l.Infof("Closing listener.")
		err := lis.Close()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			l.WithError(err).Errorf("Error closing listener.")
		}
	})
```

:152 (named-call form; note `run` does `wg.Add(1)` inside itself — body unchanged, that pre-existing pattern is not this task's to fix):

```go
		routine.Go(l, ctx, func(_ context.Context) { run(l, ctx, wg)(c, conn, uuid.New(), 4) })
```

:173 (connection closer, inside `run`'s returned func; `l`, `ctx` captured):

```go
		routine.Go(l, ctx, func(_ context.Context) {
			<-ctx.Done()
			l.Infof("Closing connection from [%s].", conn.RemoteAddr())
			conn.Close()
		})
```

:226 (packet dispatch; `fl` and `ctx` in scope):

```go
				routine.Go(fl, ctx, func(_ context.Context) { handle(fl)(config, sessionId, result) })
```

- [ ] **Step 3: Migrate `libs/atlas-rest/server/server.go`**

:171 (`Builder.Run`'s outer goroutine — `sb.l`, `sb.ctx`): wrap the entire existing body:

```go
func (sb *Builder) Run() {
	routine.Go(sb.l, sb.ctx, func(_ context.Context) {
		hs := http.Server{
			// ... body byte-identical through the shutdown logging ...
		}
		// ...
	})
}
```

:186 (inner ListenAndServe goroutine — uses the `ctx, cancel := context.WithCancel(sb.ctx)` created just above it):

```go
		routine.Go(sb.l, ctx, func(_ context.Context) {
			sb.wg.Add(1)
			defer sb.wg.Done()
			err := hs.ListenAndServe()
			if !errors.Is(err, http.ErrServerClosed) {
				sb.l.WithError(err).Errorf("Error while serving.")
				return
			}
		})
```

- [ ] **Step 4: Migrate `libs/atlas-seeder/handlers.go:49`** (`postSeed`; `l`, `ctx` are the function's params; `backgroundSeeds.Add(1)` stays outside, `Done` stays inside; the body's own `bgCtx := tenant.WithContext(context.Background(), t)` is unchanged):

```go
		backgroundSeeds.Add(1)
		routine.Go(l, ctx, func(_ context.Context) {
			defer backgroundSeeds.Done()
			bgCtx := tenant.WithContext(context.Background(), t)
			res, err := Seed(bgCtx, db, src, g)
			// ... rest byte-identical ...
		})
```

- [ ] **Step 5: Verify all three modules**

```bash
for m in libs/atlas-socket libs/atlas-rest libs/atlas-seeder; do
  ( cd "$m" && go test -race ./... && go vet ./... && go build ./... ) || exit 1
done
```

Expected: PASS/clean ×3.

- [ ] **Step 6: Update audit rows + commit** (7 rows, classification `lifecycle`/`lib-internal`, disposition `migrated`)

```bash
git add libs/atlas-socket libs/atlas-rest libs/atlas-seeder docs/tasks/task-115-safe-goroutine-helper/migration-audit.md
git commit -m "refactor(task-115): atlas-socket/rest/seeder onto routine.Go"
```

---

### Task 8: Migrate `services/atlas-channel` (largest service, ~60 sites)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/go.mod`
- Modify: every file the guard flags under `services/atlas-channel/` — the heavy ones: `kafka/consumer/map/consumer.go` (19), `movement/processor.go` (10), `kafka/consumer/party/consumer.go` (8), `kafka/consumer/session/consumer.go` (7), `kafka/consumer/messenger/consumer.go` (4), `kafka/consumer/asset/consumer.go` (3), plus 2-site files (`socket/init.go`, `kafka/consumer/{pet,party/member,monster,drop}/consumer.go`) and the `main.go` sites (:318, :327)

**Interfaces:**
- Consumes: `routine.Go` (Task 1); guard binary (Task 3) as the site enumerator and completion oracle.
- Produces: atlas-channel with zero guard findings.

- [ ] **Step 1: Add the dependency**

```bash
cd services/atlas-channel/atlas.com/channel
go mod edit -require=github.com/Chronicle20/atlas/libs/atlas-routine@v0.0.0-00010101000000-000000000000 \
  -replace=github.com/Chronicle20/atlas/libs/atlas-routine=../../../../libs/atlas-routine
go mod tidy
```

- [ ] **Step 2: Enumerate this service's sites**

```bash
( cd tools/goroutineguard && GOWORK=off go build -o /tmp/goroutineguard ./cmd/goroutineguard )
( cd services/atlas-channel/atlas.com/channel && /tmp/goroutineguard ./... 2>&1 | grep 'bare go statement' )
```

This exact enumerate-migrate-recheck loop is the recipe for Tasks 9–10 as well.

- [ ] **Step 3: Migrate every flagged site using the Mechanical Transform Rules**

Worked examples for the two named-call shapes in `main.go` (`l` and `tdm` in scope):

`main.go:318` (method call on a struct literal):

```go
	routine.Go(l, tdm.Context(), func(_ context.Context) {
		(&projection.ApplyLoop{
			// ... struct fields byte-identical ...
		}).Run(tdm.Context(), l)
	})
```

`main.go:327` (curried ticker registration):

```go
	routine.Go(l, tdm.Context(), func(_ context.Context) {
		tasks.Register(l, tdm.Context())(channel3.NewHeartbeat(l, tdm.Context(), time.Second*10))
	})
```

Consumer/handler files: every Kafka handler receives `l logrus.FieldLogger, ctx context.Context` (or has them one closure level up) — rule 1 applies; anonymous `go func() { ... }()` bodies wrap unchanged with `func(_ context.Context)`. Processor files: use the processor's own fields (`p.l`, `p.ctx`). If any site genuinely has neither in scope, plumb from the nearest constructor (rule 2) and note it in the audit row — never substitute `context.Background()` inside a service.

- [ ] **Step 4: Verify**

```bash
( cd services/atlas-channel/atlas.com/channel && /tmp/goroutineguard ./... && go test -race ./... && go vet ./... && go build ./... )
```

Expected: guard prints nothing (exit 0 for this module), tests/vet/build clean.

- [ ] **Step 5: Update audit rows + commit**

Fill every atlas-channel row (classification `handler-spawned`/`ticker`/`lifecycle` as appropriate; disposition `migrated`).

```bash
git add services/atlas-channel docs/tasks/task-115-safe-goroutine-helper/migration-audit.md
git commit -m "refactor(task-115): atlas-channel onto routine.Go"
```

---

### Task 9: Migrate the multi-site services (atlas-maps, atlas-login, atlas-monsters, atlas-monster-death, atlas-buffs, atlas-data, atlas-world, atlas-pets)

**Files (per service, module root `services/<svc>/atlas.com/<inner>/`):**
- Modify: each service's `go.mod` (require+replace, path `../../../../libs/atlas-routine`)
- Modify: flagged files, e.g. `atlas-maps` `main.go` (3), `tasks/respawn.go` (2), `map/processor.go` (2); `atlas-login` `main.go` (2), `socket/init.go` (2); `atlas-monsters` `main.go`, `tasks/task.go`, `monster/processor.go`; `atlas-monster-death` `kafka/consumer/monster/consumer.go` (2); `atlas-buffs` `main.go` (2), `character/processor.go` (2); `atlas-data` `data/processor.go` (2); `atlas-world` `main.go` (2), `tasks/task.go`; `atlas-pets` `main.go`, `tasks/task.go`, `pet/task.go`

**Interfaces:**
- Consumes: `routine.Go` (Task 1); guard binary enumerate-migrate-recheck loop (Task 8 Step 2 recipe).
- Produces: these 8 services with zero guard findings.

- [ ] **Step 1: For each service, add the dependency** (same `go mod edit` + `go mod tidy` as Task 8 Step 1, run in each module root).

- [ ] **Step 2: Migrate every flagged site.** Worked example — the PRD's motivating site, `services/atlas-monsters/atlas.com/monsters/monster/processor.go:700` (`ProcessorImpl` has fields `p.l`, `p.ctx` — verified at processor.go:71-73):

```go
	if animDelay > 0 {
		routine.Go(p.l, p.ctx, func(_ context.Context) {
			time.Sleep(animDelay)
			p.applyAnimationDelayedEffect(uniqueId, executeEffect, postExecute)
		})
	} else {
```

Ticker registrations in `main.go`/`tasks/task.go` follow the Task 8 `tasks.Register` example.

- [ ] **Step 3: Verify each module** (guard exit 0 for the module + `go test -race ./...` + `go vet ./...` + `go build ./...`, per Task 8 Step 4).

- [ ] **Step 4: Update audit rows + commit**

```bash
git add services/atlas-maps services/atlas-login services/atlas-monsters services/atlas-monster-death services/atlas-buffs services/atlas-data services/atlas-world services/atlas-pets docs/tasks/task-115-safe-goroutine-helper/migration-audit.md
git commit -m "refactor(task-115): multi-site services onto routine.Go"
```

---

### Task 10: Migrate the long-tail services (remaining 24)

**Files:** `go.mod` + flagged files in each of: atlas-account, atlas-asset-expiration, atlas-ban, atlas-cashshop, atlas-character, atlas-character-factory, atlas-configurations, atlas-doors, atlas-drops, atlas-expressions, atlas-families, atlas-guilds, atlas-invites, atlas-marriages, atlas-merchant, atlas-mounts, atlas-npc-conversations, atlas-party-quests, atlas-reactors, atlas-renders, atlas-saga-orchestrator, atlas-skills, atlas-summons, atlas-transports (list = every service the guard still flags; the pre-migration grep found sites in exactly the 33 services named across Tasks 8–10 — trust the guard, not this list, for completeness).

**Interfaces:**
- Consumes: `routine.Go` (Task 1); guard enumerate-migrate-recheck loop (Task 8 Step 2 recipe).
- Produces: zero guard findings across all of `services/`.

- [ ] **Step 1: Enumerate what's left repo-wide**

```bash
./tools/goroutine-guard.sh 2>&1 | grep 'bare go statement' || echo CLEAN
```

- [ ] **Step 2: For each still-flagged service:** add the go.mod require+replace (Task 8 Step 1), migrate each site via the Mechanical Transform Rules (most are 1–2 sites: `main.go` ticker registrations and `tasks/task.go` loops — the Task 8 `tasks.Register` example applies verbatim; `service/teardown.go` and REST-handler spawns follow the anonymous-func rule with the in-scope `l`/`ctx`).

- [ ] **Step 3: Verify each changed module** (guard 0 + `go test -race` + vet + build).

- [ ] **Step 4: Update audit rows + commit**

```bash
git add services/ docs/tasks/task-115-safe-goroutine-helper/migration-audit.md
git commit -m "refactor(task-115): remaining services onto routine.Go"
```

---

### Task 11: Zero-findings gate + audit table completion

**Files:**
- Modify: `docs/tasks/task-115-safe-goroutine-helper/migration-audit.md` (final dispositions, summary line)

**Interfaces:**
- Consumes: everything above.
- Produces: the FR-2.6/FR-3 acceptance evidence — guard exit 0 and a fully-dispositioned audit table.

- [ ] **Step 1: Full guard run**

```bash
./tools/goroutine-guard.sh && echo GUARD-CLEAN
```

Expected: `GUARD-CLEAN` (exit 0 — self-test, then every module under `services/` and `libs/` with zero diagnostics). If any finding remains, migrate it (return to the owning task's recipe) — do not allowlist to make the number zero; the only sanctioned allow entry is `testutil/helpers.go`.

- [ ] **Step 2: Redis guard still clean**

```bash
./tools/redis-key-guard.sh && echo REDIS-CLEAN
```

- [ ] **Step 3: Audit table completeness check**

Every row has a non-blank classification, logger source, ctx source, and disposition; row count equals the Task 3 baseline count; exactly one row dispositioned `allowlisted`. Cross-check the in-code marker count:

```bash
grep -rn 'goroutine-guard:allow' --include='*.go' services/ libs/ | grep -v _test.go
```

Expected: exactly 1 hit (`libs/atlas-model/testutil/helpers.go`).

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-115-safe-goroutine-helper/migration-audit.md
git commit -m "docs(task-115): complete migration audit — guard clean at zero findings"
```

---

### Task 12: Guidelines enforcement (DOM-25) + docs

**Files:**
- Modify: `.claude/agents/backend-guidelines-reviewer.md` (append DOM-25 row after the DOM-24 row, ~line 101)
- Modify: `.claude/skills/backend-dev-guidelines/SKILL.md` (Quick Start Checklist bullet)
- Modify: `.claude/skills/backend-dev-guidelines/resources/anti-patterns.md` (table row)
- Modify: `CLAUDE.md` (Build & Verification item 6)
- Modify: `docs/architectural-improvements.md` (RR-6 resolution — conditional, see Step 4)

**Interfaces:**
- Consumes: guard + helper names/paths as landed above.
- Produces: FR-4 complete.

- [ ] **Step 1: DOM-25 row in the reviewer agent**

Append to the DOM checklist table in `.claude/agents/backend-guidelines-reviewer.md`, immediately after the DOM-24 row:

```markdown
| DOM-25 | Goroutines spawned via routine.Go | Grep the changed packages for bare go statements: `grep -rnE '^\s*go (func|[A-Za-z_])' --include='*.go' <pkg>`, excluding `_test.go` files. For any hit, check for a `//goroutine-guard:allow <justification>` marker on the same line or the line above. | Non-test code contains no bare `go` statements: every goroutine is spawned via `routine.Go(l, ctx, fn)` from `github.com/Chronicle20/atlas/libs/atlas-routine` (which recovers and logs panics so one bad goroutine cannot crash the pod), except sites carrying a justified `//goroutine-guard:allow` marker. Mechanical verification: `tools/goroutine-guard.sh` exits 0 from the repo root. FAIL any new bare `go` statement or marker without a justification. |
```

- [ ] **Step 2: Skill content**

In `.claude/skills/backend-dev-guidelines/SKILL.md`, add to the Quick Start Checklist:

```markdown
- [ ] Background goroutines spawned via `routine.Go(l, ctx, fn)` from `libs/atlas-routine` — never a bare `go` statement (DOM-25; enforced by `tools/goroutine-guard.sh`)
```

In `.claude/skills/backend-dev-guidelines/resources/anti-patterns.md`, add to the anti-pattern table:

```markdown
| Bare `go` statements | An unrecovered panic in the goroutine crashes the whole pod — spawn via `routine.Go(l, ctx, fn)` from `libs/atlas-routine`; enforced by `tools/goroutine-guard.sh` (DOM-25). Test-scaffolding exceptions need a justified `//goroutine-guard:allow` marker. |
```

- [ ] **Step 3: CLAUDE.md Build & Verification item 6** (after item 5, matching its voice):

```markdown
6. **`tools/goroutine-guard.sh` clean from the repo root.** Bans bare `go`
   statements outside `libs/atlas-routine` and justified
   `//goroutine-guard:allow` sites (RR-6, task-115) — every goroutine must be
   spawned via `routine.Go`. Runs alongside `go vet ./...`.
```

- [ ] **Step 4: Mark RR-6 resolved — conditional**

```bash
grep -n 'RR-6' docs/architectural-improvements.md || echo ABSENT
```

- If **present** (the reliability-review doc rewrite has landed on main and been rebased in): add under the RR-6 heading:

```markdown
**Resolved by task-115:** all bare `go` statements in non-test code migrated to `routine.Go` (`libs/atlas-routine`); regression-blocked by `tools/goroutine-guard.sh` (CI) and DOM-25.
```

- If **ABSENT**: the RR-6 section currently exists only in the main checkout's *uncommitted* rewrite of this doc (verified during planning — the committed copy on this branch has no `RR-*` items). Do NOT invent an RR-6 section. Record in the PR description and in `context.md`: "mark RR-6 resolved in docs/architectural-improvements.md at rebase time, once the reliability-review rewrite lands on main" — and re-run this step after the pre-PR rebase.

- [ ] **Step 5: Commit**

```bash
git add .claude/agents/backend-guidelines-reviewer.md .claude/skills/backend-dev-guidelines CLAUDE.md docs/architectural-improvements.md
git commit -m "docs(task-115): DOM-25 goroutine rule, CLAUDE.md guard item, RR-6 resolution"
```

(Drop `docs/architectural-improvements.md` from the `git add` if Step 4 took the ABSENT path.)

---

### Task 13: Branch-wide verification (CLAUDE.md gate)

**Files:** none (verification only; fix-and-recommit anything it surfaces).

**Interfaces:**
- Consumes: the whole branch.
- Produces: the evidence for every PRD acceptance criterion that is command-verifiable.

- [ ] **Step 1: Per-module test/vet/build over every changed module**

```bash
git diff --name-only $(git merge-base HEAD main) | grep 'go\.mod$'
```

For each listed module directory (expect: `libs/atlas-routine`, `libs/atlas-lock`, `libs/atlas-kafka`, `libs/atlas-model`, `libs/atlas-socket`, `libs/atlas-rest`, `libs/atlas-seeder`, ~33 service modules, `tools/goroutineguard`):

```bash
( cd <moddir> && go test -race ./... && go vet ./... && go build ./... )
```

(`tools/goroutineguard` needs `GOWORK=off` prefixes.) Expected: all clean. Record any failure, fix, re-run.

- [ ] **Step 2: Both guards from the repo root**

```bash
./tools/goroutine-guard.sh && ./tools/redis-key-guard.sh && echo GUARDS-CLEAN
```

- [ ] **Step 3: Docker bake — mandatory, every service go.mod was touched**

```bash
docker buildx bake all-go-services
```

Expected: every target builds. This is the only step that catches a missing `Dockerfile` COPY/loop entry (Task 1 Step 7) — `go build` against `go.work` cannot. Expect this to take a long time; do not skip or sample (CLAUDE.md rule 4).

- [ ] **Step 4: Acceptance sweep against the PRD**

Walk `prd.md` §10 checkboxes; each is now backed by a command output or a committed artifact. Confirm `migration-audit.md` row count == baseline count and the single allowlist entry. Then invoke `superpowers:requesting-code-review` before any PR (CLAUDE.md: Code Review Before PR).

- [ ] **Step 5: Commit any verification fixes**

```bash
git status --short   # commit fixes if any step above forced changes
```
