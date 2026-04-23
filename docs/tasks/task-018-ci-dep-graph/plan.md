# Dep-Graph-Driven CI Matrix Selection — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the "any library change → rebuild all 56 services" behavior in CI with "rebuild only services whose Go module graph transitively depends on the changed library," via a new in-repo Go tool that parses `go.mod` files and emits narrowed GitHub Actions matrices.

**Architecture:** A small Go program at `tools/cideps/` walks `libs/*/go.mod` and `services/*/atlas.com/*/go.mod`, builds a module dependency graph keyed on directory-derived short names, and emits three matrix arrays (go-services, go-libraries, docker-services) on stdout. The composite action `.github/actions/detect-changes` invokes it after file diffing; on any non-zero exit the action falls back to the existing "build all" matrix construction.

**Tech Stack:** Go 1.25.5, `golang.org/x/mod/modfile` for go.mod parsing, `encoding/json` for output, GitHub Actions composite action shell glue, `jq` for matrix pass-through in the fallback branch.

---

## File Structure

**Created:**
- `tools/cideps/go.mod` — new Go module `github.com/Chronicle20/atlas/tools/cideps`
- `tools/cideps/go.sum`
- `tools/cideps/main.go` — flag parsing, orchestration, JSON stdout
- `tools/cideps/graph.go` — go.mod walking, parsing, graph construction, closure
- `tools/cideps/graph_test.go`
- `tools/cideps/config.go` — `services.json` loading and enrichment
- `tools/cideps/config_test.go`
- `tools/cideps/select.go` — selection algorithm (affected set computation)
- `tools/cideps/select_test.go`
- `tools/cideps/main_test.go` — end-to-end (input flags → stdout JSON)
- `tools/cideps/realrepo_test.go` — sanity test against the actual repo tree
- `tools/cideps/testdata/simple/libs/lib-a/go.mod`
- `tools/cideps/testdata/simple/libs/lib-b/go.mod`
- `tools/cideps/testdata/simple/services/svc-a/atlas.com/svc-a/go.mod`
- `tools/cideps/testdata/simple/services.json`
- `tools/cideps/testdata/transitive/libs/lib-a/go.mod`
- `tools/cideps/testdata/transitive/libs/lib-b/go.mod`
- `tools/cideps/testdata/transitive/libs/lib-c/go.mod`
- `tools/cideps/testdata/transitive/services/svc-a/atlas.com/svc-a/go.mod`
- `tools/cideps/testdata/transitive/services/svc-b/atlas.com/svc-b/go.mod`
- `tools/cideps/testdata/transitive/services.json`

**Modified:**
- `go.work` — add `./tools/cideps` to the `use (...)` block
- `.github/actions/detect-changes/action.yml` — invoke cideps, emit matrix outputs, fallback branch
- `.github/workflows/pr-validation.yml` — consume new action outputs, drop inline matrix construction
- `.github/workflows/main-publish.yml` — same

---

## Task 1: Scaffold the `tools/cideps` module

**Files:**
- Create: `tools/cideps/go.mod`
- Create: `tools/cideps/main.go`
- Modify: `go.work`

- [ ] **Step 1: Create the Go module**

Create `tools/cideps/go.mod`:

```
module github.com/Chronicle20/atlas/tools/cideps

go 1.25.5

require golang.org/x/mod v0.21.0
```

- [ ] **Step 2: Create a stub `main.go`**

Create `tools/cideps/main.go`:

```go
package main

func main() {}
```

- [ ] **Step 3: Add module to `go.work`**

Edit `go.work`, add `./tools/cideps` inside the `use (...)` block. Place it alphabetically under the `./tools/...` grouping if one exists, otherwise immediately after the last lib entry. Example diff target:

```
use (
    ./libs/atlas-constants
    ...
    ./libs/atlas-tenant
    ./tools/cideps
    ./services/atlas-account/atlas.com/account
    ...
)
```

- [ ] **Step 4: Download deps and verify build**

Run: `cd tools/cideps && go mod tidy && go build ./...`
Expected: creates `go.sum`, exits 0, no binary output (empty main).

- [ ] **Step 5: Commit**

```bash
git add tools/cideps/go.mod tools/cideps/go.sum tools/cideps/main.go go.work
git commit -m "chore(cideps): scaffold tools/cideps Go module"
```

---

## Task 2: Parse a single `go.mod` and extract atlas lib requires

**Files:**
- Create: `tools/cideps/graph.go`
- Create: `tools/cideps/graph_test.go`

- [ ] **Step 1: Write the failing test**

Create `tools/cideps/graph_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAtlasRequires_DirectAndIndirect(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	contents := `module atlas-svc

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-kafka v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/google/uuid v1.6.0
)

require (
	github.com/Chronicle20/atlas/libs/atlas-retry v0.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
)
`
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := parseAtlasRequires(path)
	if err != nil {
		t.Fatalf("parseAtlasRequires: %v", err)
	}
	want := map[string]struct{}{
		"atlas-kafka":  {},
		"atlas-tenant": {},
		"atlas-retry":  {},
	}
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d want=%d: %v", len(got), len(want), got)
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
}

func TestParseAtlasRequires_MalformedFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte("this is not a go.mod"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseAtlasRequires(path); err == nil {
		t.Fatal("expected error for malformed go.mod, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tools/cideps && go test -run TestParseAtlasRequires -v`
Expected: FAIL with "parseAtlasRequires undefined" or similar build error.

- [ ] **Step 3: Implement `parseAtlasRequires`**

Create `tools/cideps/graph.go`:

```go
package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/mod/modfile"
)

const atlasLibPrefix = "github.com/Chronicle20/atlas/libs/"

// parseAtlasRequires returns the set of atlas-lib short names (e.g. "atlas-kafka")
// that the go.mod at path requires — both direct and indirect.
func parseAtlasRequires(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	f, err := modfile.Parse(path, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	out := make(map[string]struct{})
	for _, r := range f.Require {
		if strings.HasPrefix(r.Mod.Path, atlasLibPrefix) {
			short := strings.TrimPrefix(r.Mod.Path, atlasLibPrefix)
			out[short] = struct{}{}
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd tools/cideps && go test -run TestParseAtlasRequires -v`
Expected: PASS for both subtests.

- [ ] **Step 5: Commit**

```bash
git add tools/cideps/graph.go tools/cideps/graph_test.go tools/cideps/go.sum
git commit -m "feat(cideps): parse atlas lib requires from go.mod"
```

---

## Task 3: Walk a repo tree and build the full module graph

**Files:**
- Modify: `tools/cideps/graph.go`
- Modify: `tools/cideps/graph_test.go`
- Create: `tools/cideps/testdata/simple/libs/lib-a/go.mod`
- Create: `tools/cideps/testdata/simple/libs/lib-b/go.mod`
- Create: `tools/cideps/testdata/simple/services/svc-a/atlas.com/svc-a/go.mod`

- [ ] **Step 1: Create the fixture tree**

Create `tools/cideps/testdata/simple/libs/lib-a/go.mod`:

```
module github.com/Chronicle20/atlas/libs/lib-a

go 1.25.5
```

Create `tools/cideps/testdata/simple/libs/lib-b/go.mod`:

```
module github.com/Chronicle20/atlas/libs/lib-b

go 1.25.5

require github.com/Chronicle20/atlas/libs/lib-a v0.0.0

replace github.com/Chronicle20/atlas/libs/lib-a => ../lib-a
```

Create `tools/cideps/testdata/simple/services/svc-a/atlas.com/svc-a/go.mod`:

```
module svc-a

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/lib-b v0.0.0
)
```

The lib short names in this fixture intentionally do **not** start with `atlas-` — the tool must not depend on that prefix; it keys purely on directory name.

- [ ] **Step 2: Write the failing test**

Add to `tools/cideps/graph_test.go`:

```go
func TestBuildGraph_Simple(t *testing.T) {
	g, err := BuildGraph("testdata/simple")
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	if got := g.Libs(); !equalSet(got, []string{"lib-a", "lib-b"}) {
		t.Errorf("libs=%v want [lib-a lib-b]", got)
	}
	if got := g.Services(); !equalSet(got, []string{"svc-a"}) {
		t.Errorf("services=%v want [svc-a]", got)
	}
	if got := g.DirectDeps("svc-a"); !equalSet(got, []string{"lib-b"}) {
		t.Errorf("deps(svc-a)=%v want [lib-b]", got)
	}
	if got := g.DirectDeps("lib-b"); !equalSet(got, []string{"lib-a"}) {
		t.Errorf("deps(lib-b)=%v want [lib-a]", got)
	}
	if got := g.DirectDeps("lib-a"); len(got) != 0 {
		t.Errorf("deps(lib-a)=%v want empty", got)
	}
}

// equalSet returns true if a and b contain the same elements (order-insensitive).
func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int)
	for _, s := range a {
		m[s]++
	}
	for _, s := range b {
		m[s]--
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd tools/cideps && go test -run TestBuildGraph_Simple -v`
Expected: FAIL with "BuildGraph undefined".

- [ ] **Step 4: Implement `BuildGraph` and the `Graph` type**

Append to `tools/cideps/graph.go`:

```go
import (
	// (keep existing imports; add these)
	"path/filepath"
	"sort"
)

// Graph is a dependency graph keyed on short names.
// Services and libs are tracked separately; edges go from any module to the
// libs it directly requires.
type Graph struct {
	services map[string]struct{}
	libs     map[string]struct{}
	deps     map[string]map[string]struct{} // module → direct lib deps
}

func (g *Graph) Libs() []string     { return sortedKeys(g.libs) }
func (g *Graph) Services() []string { return sortedKeys(g.services) }

func (g *Graph) DirectDeps(mod string) []string {
	if g.deps[mod] == nil {
		return nil
	}
	return sortedKeys(g.deps[mod])
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// BuildGraph walks root, parsing every libs/<lib>/go.mod and
// services/<svc>/atlas.com/<leaf>/go.mod, and returns the dependency graph.
func BuildGraph(root string) (*Graph, error) {
	g := &Graph{
		services: make(map[string]struct{}),
		libs:     make(map[string]struct{}),
		deps:     make(map[string]map[string]struct{}),
	}

	libsDir := filepath.Join(root, "libs")
	libEntries, err := os.ReadDir(libsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read libs dir: %w", err)
	}
	for _, e := range libEntries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		modPath := filepath.Join(libsDir, name, "go.mod")
		if _, err := os.Stat(modPath); err != nil {
			continue // not a Go module
		}
		reqs, err := parseAtlasRequires(modPath)
		if err != nil {
			return nil, err
		}
		g.libs[name] = struct{}{}
		g.deps[name] = reqs
	}

	svcsDir := filepath.Join(root, "services")
	svcEntries, err := os.ReadDir(svcsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read services dir: %w", err)
	}
	for _, e := range svcEntries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// services/<svc>/atlas.com/<leaf>/go.mod — leaf is the only subdir under atlas.com
		atlasCom := filepath.Join(svcsDir, name, "atlas.com")
		leafEntries, err := os.ReadDir(atlasCom)
		if err != nil {
			continue // no atlas.com subdir → not a Go service (e.g. atlas-ui, atlas-assets)
		}
		for _, leaf := range leafEntries {
			if !leaf.IsDir() {
				continue
			}
			modPath := filepath.Join(atlasCom, leaf.Name(), "go.mod")
			if _, err := os.Stat(modPath); err != nil {
				continue
			}
			reqs, err := parseAtlasRequires(modPath)
			if err != nil {
				return nil, err
			}
			g.services[name] = struct{}{}
			g.deps[name] = reqs
			break // one Go module per service is expected
		}
	}

	return g, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd tools/cideps && go test -run TestBuildGraph_Simple -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/cideps/graph.go tools/cideps/graph_test.go tools/cideps/testdata/simple
git commit -m "feat(cideps): build module graph from repo tree"
```

---

## Task 4: Compute transitive closure

**Files:**
- Modify: `tools/cideps/graph.go`
- Modify: `tools/cideps/graph_test.go`
- Create: `tools/cideps/testdata/transitive/libs/lib-a/go.mod`
- Create: `tools/cideps/testdata/transitive/libs/lib-b/go.mod`
- Create: `tools/cideps/testdata/transitive/libs/lib-c/go.mod`
- Create: `tools/cideps/testdata/transitive/services/svc-a/atlas.com/svc-a/go.mod`
- Create: `tools/cideps/testdata/transitive/services/svc-b/atlas.com/svc-b/go.mod`

- [ ] **Step 1: Create the transitive fixture**

Create `tools/cideps/testdata/transitive/libs/lib-a/go.mod`:

```
module github.com/Chronicle20/atlas/libs/lib-a

go 1.25.5
```

Create `tools/cideps/testdata/transitive/libs/lib-b/go.mod`:

```
module github.com/Chronicle20/atlas/libs/lib-b

go 1.25.5

require github.com/Chronicle20/atlas/libs/lib-a v0.0.0
```

Create `tools/cideps/testdata/transitive/libs/lib-c/go.mod`:

```
module github.com/Chronicle20/atlas/libs/lib-c

go 1.25.5
```

Create `tools/cideps/testdata/transitive/services/svc-a/atlas.com/svc-a/go.mod`:

```
module svc-a

go 1.25.5

require github.com/Chronicle20/atlas/libs/lib-b v0.0.0
```

(svc-a transitively depends on lib-a via lib-b.)

Create `tools/cideps/testdata/transitive/services/svc-b/atlas.com/svc-b/go.mod`:

```
module svc-b

go 1.25.5

require github.com/Chronicle20/atlas/libs/lib-c v0.0.0
```

(svc-b depends only on lib-c.)

- [ ] **Step 2: Write the failing test**

Add to `tools/cideps/graph_test.go`:

```go
func TestClosure_Transitive(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	cases := []struct {
		mod  string
		want []string
	}{
		{"svc-a", []string{"lib-a", "lib-b"}},
		{"svc-b", []string{"lib-c"}},
		{"lib-b", []string{"lib-a"}},
		{"lib-a", nil},
		{"lib-c", nil},
	}
	for _, tc := range cases {
		t.Run(tc.mod, func(t *testing.T) {
			got := g.Closure(tc.mod)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !equalSet(got, tc.want) {
				t.Errorf("closure(%s)=%v want=%v", tc.mod, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd tools/cideps && go test -run TestClosure_Transitive -v`
Expected: FAIL with "Closure undefined".

- [ ] **Step 4: Implement `Closure`**

Append to `tools/cideps/graph.go`:

```go
// Closure returns the set of atlas libs that mod transitively requires,
// as a sorted slice. Does not include mod itself.
func (g *Graph) Closure(mod string) []string {
	visited := make(map[string]struct{})
	var walk func(string)
	walk = func(m string) {
		for dep := range g.deps[m] {
			if _, ok := visited[dep]; ok {
				continue
			}
			visited[dep] = struct{}{}
			walk(dep)
		}
	}
	walk(mod)
	return sortedKeys(visited)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd tools/cideps && go test -run TestClosure_Transitive -v`
Expected: PASS for all subtests.

- [ ] **Step 6: Commit**

```bash
git add tools/cideps/graph.go tools/cideps/graph_test.go tools/cideps/testdata/transitive
git commit -m "feat(cideps): compute transitive lib closure for each module"
```

---

## Task 5: Implement the selection algorithm

**Files:**
- Create: `tools/cideps/select.go`
- Create: `tools/cideps/select_test.go`

- [ ] **Step 1: Write the failing test**

Create `tools/cideps/select_test.go`:

```go
package main

import "testing"

func TestSelect_DirectLibChange(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{
		ChangedLibs: []string{"lib-c"},
	})
	if !equalSet(sel.Services, []string{"svc-b"}) {
		t.Errorf("services=%v want [svc-b]", sel.Services)
	}
	if !equalSet(sel.Libs, []string{"lib-c"}) {
		t.Errorf("libs=%v want [lib-c]", sel.Libs)
	}
}

func TestSelect_TransitiveLibChange(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{
		ChangedLibs: []string{"lib-a"},
	})
	// svc-a → lib-b → lib-a, so svc-a is affected.
	// lib-b → lib-a, so lib-b is affected.
	if !equalSet(sel.Services, []string{"svc-a"}) {
		t.Errorf("services=%v want [svc-a]", sel.Services)
	}
	if !equalSet(sel.Libs, []string{"lib-a", "lib-b"}) {
		t.Errorf("libs=%v want [lib-a lib-b]", sel.Libs)
	}
}

func TestSelect_ChangedServiceUnion(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{
		ChangedLibs:     []string{"lib-c"},
		ChangedServices: []string{"svc-a"},
	})
	if !equalSet(sel.Services, []string{"svc-a", "svc-b"}) {
		t.Errorf("services=%v want [svc-a svc-b]", sel.Services)
	}
}

func TestSelect_NoChanges(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{})
	if len(sel.Services) != 0 || len(sel.Libs) != 0 {
		t.Errorf("expected empty selection, got services=%v libs=%v", sel.Services, sel.Libs)
	}
}

func TestSelect_ForceAll(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{ForceAll: true})
	if !equalSet(sel.Services, []string{"svc-a", "svc-b"}) {
		t.Errorf("services=%v want [svc-a svc-b]", sel.Services)
	}
	if !equalSet(sel.Libs, []string{"lib-a", "lib-b", "lib-c"}) {
		t.Errorf("libs=%v want [lib-a lib-b lib-c]", sel.Libs)
	}
}

func TestSelect_UnknownNameIgnored(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{
		ChangedLibs:     []string{"no-such-lib"},
		ChangedServices: []string{"no-such-svc"},
	})
	if len(sel.Services) != 0 || len(sel.Libs) != 0 {
		t.Errorf("unknown names should select nothing, got services=%v libs=%v", sel.Services, sel.Libs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tools/cideps && go test -run TestSelect -v`
Expected: FAIL with "Select undefined" / "SelectInput undefined".

- [ ] **Step 3: Implement selection**

Create `tools/cideps/select.go`:

```go
package main

type SelectInput struct {
	ChangedLibs     []string
	ChangedServices []string
	ForceAll        bool
}

type Selection struct {
	Services []string
	Libs     []string
}

// Select computes the affected-module set.
//
// Rules:
//   - ForceAll → every service and library in the graph.
//   - Otherwise a service is affected when it is in ChangedServices or when
//     its lib-closure intersects ChangedLibs.
//   - Otherwise a library is affected when it is in ChangedLibs or when its
//     lib-closure intersects ChangedLibs.
//   - Unknown names in ChangedLibs/ChangedServices are ignored silently.
func Select(g *Graph, in SelectInput) Selection {
	if in.ForceAll {
		return Selection{Services: g.Services(), Libs: g.Libs()}
	}

	changedLibs := make(map[string]struct{})
	for _, n := range in.ChangedLibs {
		if _, ok := g.deps[n]; ok {
			changedLibs[n] = struct{}{}
		}
	}

	affectedSvcs := make(map[string]struct{})
	for _, n := range in.ChangedServices {
		if _, ok := g.services[n]; ok {
			affectedSvcs[n] = struct{}{}
		}
	}
	for _, svc := range g.Services() {
		if _, done := affectedSvcs[svc]; done {
			continue
		}
		for _, lib := range g.Closure(svc) {
			if _, hit := changedLibs[lib]; hit {
				affectedSvcs[svc] = struct{}{}
				break
			}
		}
	}

	affectedLibs := make(map[string]struct{})
	for lib := range changedLibs {
		affectedLibs[lib] = struct{}{}
	}
	for _, lib := range g.Libs() {
		if _, done := affectedLibs[lib]; done {
			continue
		}
		for _, dep := range g.Closure(lib) {
			if _, hit := changedLibs[dep]; hit {
				affectedLibs[lib] = struct{}{}
				break
			}
		}
	}

	return Selection{
		Services: sortedKeys(affectedSvcs),
		Libs:     sortedKeys(affectedLibs),
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd tools/cideps && go test -run TestSelect -v`
Expected: PASS for all six subtests.

- [ ] **Step 5: Commit**

```bash
git add tools/cideps/select.go tools/cideps/select_test.go
git commit -m "feat(cideps): implement affected-set selection algorithm"
```

---

## Task 6: Load and enrich from `services.json`

**Files:**
- Create: `tools/cideps/config.go`
- Create: `tools/cideps/config_test.go`
- Create: `tools/cideps/testdata/simple/services.json`
- Create: `tools/cideps/testdata/transitive/services.json`

- [ ] **Step 1: Create fixture config files**

Create `tools/cideps/testdata/simple/services.json`:

```json
{
  "services": [
    {
      "name": "svc-a",
      "type": "go-service",
      "path": "services/svc-a",
      "module_path": "services/svc-a/atlas.com/svc-a",
      "docker_image": "example/svc-a",
      "docker_context": "."
    }
  ],
  "libraries": [
    { "name": "lib-a", "path": "libs/lib-a", "module_path": "libs/lib-a" },
    { "name": "lib-b", "path": "libs/lib-b", "module_path": "libs/lib-b" }
  ]
}
```

Create `tools/cideps/testdata/transitive/services.json`:

```json
{
  "services": [
    {
      "name": "svc-a",
      "type": "go-service",
      "path": "services/svc-a",
      "module_path": "services/svc-a/atlas.com/svc-a",
      "docker_image": "example/svc-a",
      "docker_context": "."
    },
    {
      "name": "svc-b",
      "type": "go-service",
      "path": "services/svc-b",
      "module_path": "services/svc-b/atlas.com/svc-b",
      "docker_image": "example/svc-b"
    }
  ],
  "libraries": [
    { "name": "lib-a", "path": "libs/lib-a", "module_path": "libs/lib-a" },
    { "name": "lib-b", "path": "libs/lib-b", "module_path": "libs/lib-b" },
    { "name": "lib-c", "path": "libs/lib-c", "module_path": "libs/lib-c", "coverage_threshold": 80 }
  ]
}
```

Note: `svc-b` omits `docker_context` to test the `docker_context // .path` fallback.

- [ ] **Step 2: Write the failing test**

Create `tools/cideps/config_test.go`:

```go
package main

import "testing"

func TestLoadConfig_Simple(t *testing.T) {
	cfg, err := LoadConfig("testdata/simple/services.json")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Services) != 1 || cfg.Services[0].Name != "svc-a" {
		t.Errorf("services=%+v", cfg.Services)
	}
	if len(cfg.Libraries) != 2 {
		t.Errorf("libraries=%+v", cfg.Libraries)
	}
}

func TestEnrich_DockerContextFallback(t *testing.T) {
	cfg, err := LoadConfig("testdata/transitive/services.json")
	if err != nil {
		t.Fatal(err)
	}
	svcRows := cfg.EnrichDockerServices([]string{"svc-a", "svc-b"})
	got := make(map[string]string)
	for _, r := range svcRows {
		got[r.Name] = r.DockerContext
	}
	if got["svc-a"] != "." {
		t.Errorf("svc-a docker_context=%q want .", got["svc-a"])
	}
	if got["svc-b"] != "services/svc-b" {
		t.Errorf("svc-b docker_context=%q want services/svc-b (fallback to path)", got["svc-b"])
	}
}

func TestEnrich_GoServicesFiltersType(t *testing.T) {
	cfg := &Config{
		Services: []ServiceEntry{
			{Name: "a", Type: "go-service", Path: "p", ModulePath: "mp", DockerImage: "di"},
			{Name: "b", Type: "static-service", Path: "p", DockerImage: "di"},
		},
	}
	rows := cfg.EnrichGoServices([]string{"a", "b"})
	if len(rows) != 1 || rows[0].Name != "a" {
		t.Errorf("rows=%+v want only go-service a", rows)
	}
}

func TestEnrich_GoLibraries_CoverageDefaultZero(t *testing.T) {
	cfg, err := LoadConfig("testdata/transitive/services.json")
	if err != nil {
		t.Fatal(err)
	}
	rows := cfg.EnrichGoLibraries([]string{"lib-a", "lib-c"})
	got := make(map[string]int)
	for _, r := range rows {
		got[r.Name] = r.CoverageThreshold
	}
	if got["lib-a"] != 0 {
		t.Errorf("lib-a coverage_threshold=%d want 0", got["lib-a"])
	}
	if got["lib-c"] != 80 {
		t.Errorf("lib-c coverage_threshold=%d want 80", got["lib-c"])
	}
}

func TestEnrich_UnknownNameProducesWarning(t *testing.T) {
	cfg, err := LoadConfig("testdata/transitive/services.json")
	if err != nil {
		t.Fatal(err)
	}
	rows := cfg.EnrichGoServices([]string{"svc-a", "ghost"})
	var warnings []string
	cfg.Warnings(&warnings)
	if len(rows) != 1 || rows[0].Name != "svc-a" {
		t.Errorf("rows=%+v; expected only svc-a", rows)
	}
	if len(warnings) != 1 {
		t.Errorf("warnings=%v; expected 1", warnings)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd tools/cideps && go test -run TestLoadConfig -v`
Expected: FAIL with "LoadConfig undefined".

- [ ] **Step 4: Implement config loading and enrichment**

Create `tools/cideps/config.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type ServiceEntry struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	Path              string `json:"path"`
	ModulePath        string `json:"module_path,omitempty"`
	DockerImage       string `json:"docker_image,omitempty"`
	DockerContext     string `json:"docker_context,omitempty"`
}

type LibraryEntry struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	ModulePath        string `json:"module_path"`
	CoverageThreshold int    `json:"coverage_threshold,omitempty"`
}

type Config struct {
	Services  []ServiceEntry `json:"services"`
	Libraries []LibraryEntry `json:"libraries"`

	warnings []string
}

func (c *Config) Warnings(dst *[]string) {
	*dst = append(*dst, c.warnings...)
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// GoServiceRow matches the matrix shape consumed by test-go-services.
type GoServiceRow struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ModulePath  string `json:"module_path"`
	DockerImage string `json:"docker_image,omitempty"`
}

// GoLibraryRow matches the matrix shape consumed by test-go-libraries.
type GoLibraryRow struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	ModulePath        string `json:"module_path"`
	CoverageThreshold int    `json:"coverage_threshold"`
}

// DockerServiceRow matches the matrix shape consumed by build-docker.
type DockerServiceRow struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	DockerContext string `json:"docker_context"`
	DockerImage   string `json:"docker_image"`
}

func (c *Config) EnrichGoServices(names []string) []GoServiceRow {
	byName := make(map[string]ServiceEntry, len(c.Services))
	for _, s := range c.Services {
		byName[s.Name] = s
	}
	sort.Strings(names)
	out := make([]GoServiceRow, 0, len(names))
	for _, n := range names {
		s, ok := byName[n]
		if !ok {
			c.warnings = append(c.warnings, fmt.Sprintf("services.json has no entry for %q", n))
			continue
		}
		if s.Type != "go-service" {
			continue
		}
		out = append(out, GoServiceRow{
			Name: s.Name, Path: s.Path, ModulePath: s.ModulePath, DockerImage: s.DockerImage,
		})
	}
	return out
}

func (c *Config) EnrichGoLibraries(names []string) []GoLibraryRow {
	byName := make(map[string]LibraryEntry, len(c.Libraries))
	for _, l := range c.Libraries {
		byName[l.Name] = l
	}
	sort.Strings(names)
	out := make([]GoLibraryRow, 0, len(names))
	for _, n := range names {
		l, ok := byName[n]
		if !ok {
			c.warnings = append(c.warnings, fmt.Sprintf("services.json has no entry for lib %q", n))
			continue
		}
		out = append(out, GoLibraryRow{
			Name: l.Name, Path: l.Path, ModulePath: l.ModulePath, CoverageThreshold: l.CoverageThreshold,
		})
	}
	return out
}

func (c *Config) EnrichDockerServices(names []string) []DockerServiceRow {
	byName := make(map[string]ServiceEntry, len(c.Services))
	for _, s := range c.Services {
		byName[s.Name] = s
	}
	sort.Strings(names)
	out := make([]DockerServiceRow, 0, len(names))
	for _, n := range names {
		s, ok := byName[n]
		if !ok {
			c.warnings = append(c.warnings, fmt.Sprintf("services.json has no entry for docker service %q", n))
			continue
		}
		if s.DockerImage == "" {
			continue
		}
		ctx := s.DockerContext
		if ctx == "" {
			ctx = s.Path
		}
		out = append(out, DockerServiceRow{
			Name: s.Name, Path: s.Path, DockerContext: ctx, DockerImage: s.DockerImage,
		})
	}
	return out
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd tools/cideps && go test -run "TestLoadConfig|TestEnrich" -v`
Expected: PASS for all five subtests.

- [ ] **Step 6: Commit**

```bash
git add tools/cideps/config.go tools/cideps/config_test.go tools/cideps/testdata/simple/services.json tools/cideps/testdata/transitive/services.json
git commit -m "feat(cideps): load services.json and enrich matrix rows"
```

---

## Task 7: Wire `main.go` — flags, orchestration, JSON stdout

**Files:**
- Modify: `tools/cideps/main.go`
- Create: `tools/cideps/main_test.go`

- [ ] **Step 1: Write the failing test**

Create `tools/cideps/main_test.go`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRun_TransitiveFixture_LibChange(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--root=testdata/transitive",
		"--config=testdata/transitive/services.json",
		"--changed-libs=lib-a",
		"--changed-services=",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}

	var out struct {
		GoServices      []GoServiceRow     `json:"go-services"`
		GoLibraries     []GoLibraryRow     `json:"go-libraries"`
		DockerServices  []DockerServiceRow `json:"docker-services"`
		Reason          string             `json:"reason"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v; stdout=%s", err, stdout.String())
	}

	// svc-a transitively depends on lib-a via lib-b
	if len(out.GoServices) != 1 || out.GoServices[0].Name != "svc-a" {
		t.Errorf("go-services=%+v want [svc-a]", out.GoServices)
	}
	if len(out.DockerServices) != 1 || out.DockerServices[0].Name != "svc-a" {
		t.Errorf("docker-services=%+v want [svc-a]", out.DockerServices)
	}
	// lib-b and lib-a both affected
	names := make([]string, 0, len(out.GoLibraries))
	for _, r := range out.GoLibraries {
		names = append(names, r.Name)
	}
	if !equalSet(names, []string{"lib-a", "lib-b"}) {
		t.Errorf("go-libraries=%v want [lib-a lib-b]", names)
	}
	if out.Reason == "" {
		t.Errorf("reason is empty")
	}
}

func TestRun_ForceAll(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--root=testdata/transitive",
		"--config=testdata/transitive/services.json",
		"--force-all",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	var out struct {
		GoServices     []GoServiceRow     `json:"go-services"`
		DockerServices []DockerServiceRow `json:"docker-services"`
		GoLibraries    []GoLibraryRow     `json:"go-libraries"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.GoServices) != 2 || len(out.DockerServices) != 2 || len(out.GoLibraries) != 3 {
		t.Errorf("force-all counts: services=%d docker=%d libs=%d",
			len(out.GoServices), len(out.DockerServices), len(out.GoLibraries))
	}
}

func TestRun_BadRoot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--root=testdata/does-not-exist",
		"--config=testdata/transitive/services.json",
		"--changed-libs=lib-a",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit, got 0; stdout=%s", stdout.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout on error, got %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Errorf("expected stderr message")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tools/cideps && go test -run TestRun -v`
Expected: FAIL with "run undefined".

- [ ] **Step 3: Implement `run` and wire `main`**

Replace `tools/cideps/main.go` entirely with:

```go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type output struct {
	GoServices     []GoServiceRow     `json:"go-services"`
	GoLibraries    []GoLibraryRow     `json:"go-libraries"`
	DockerServices []DockerServiceRow `json:"docker-services"`
	Reason         string             `json:"reason"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cideps", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		root            = fs.String("root", ".", "repo root")
		configPath      = fs.String("config", ".github/config/services.json", "path to services.json")
		changedLibsArg  = fs.String("changed-libs", "", "comma-separated lib short names")
		changedSvcsArg  = fs.String("changed-services", "", "comma-separated service short names")
		forceAll        = fs.Bool("force-all", false, "treat everything as affected")
	)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	g, err := BuildGraph(*root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	sel := Select(g, SelectInput{
		ChangedLibs:     splitCSV(*changedLibsArg),
		ChangedServices: splitCSV(*changedSvcsArg),
		ForceAll:        *forceAll,
	})

	out := output{
		GoServices:     cfg.EnrichGoServices(sel.Services),
		GoLibraries:    cfg.EnrichGoLibraries(sel.Libs),
		DockerServices: cfg.EnrichDockerServices(sel.Services),
		Reason:         buildReason(sel, *forceAll, *changedLibsArg, *changedSvcsArg),
	}

	var warnings []string
	cfg.Warnings(&warnings)
	for _, w := range warnings {
		fmt.Fprintln(stderr, "warning:", w)
	}

	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func buildReason(sel Selection, forceAll bool, changedLibs, changedSvcs string) string {
	if forceAll {
		return "force-all: rebuilding all services and libraries"
	}
	return fmt.Sprintf(
		"changed-libs=[%s] changed-services=[%s] → %d services, %d libraries affected",
		changedLibs, changedSvcs, len(sel.Services), len(sel.Libs),
	)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd tools/cideps && go test ./... -v`
Expected: ALL PASS.

- [ ] **Step 5: Smoke test against the real repo**

Run from the repo root:

```bash
go run ./tools/cideps \
  --changed-libs=atlas-saga \
  --config=.github/config/services.json
```

Expected: a JSON document on stdout with a non-empty `go-services` array (services depending on `atlas-saga`) and a `reason` field. Verify by eye that the service names look plausible.

- [ ] **Step 6: Commit**

```bash
git add tools/cideps/main.go tools/cideps/main_test.go
git commit -m "feat(cideps): wire CLI flags and stdout JSON output"
```

---

## Task 8: Real-repo sanity test

**Files:**
- Create: `tools/cideps/realrepo_test.go`

- [ ] **Step 1: Write the test**

Create `tools/cideps/realrepo_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot returns the path to the Atlas repo root, or skips the test if we
// can't locate it (e.g. the tool is vendored into another repo).
func repoRoot(t *testing.T) string {
	t.Helper()
	// tools/cideps is two levels deep from repo root.
	abs, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(abs, "go.work")); err != nil {
		t.Skipf("no go.work at %s, skipping real-repo test", abs)
	}
	return abs
}

func TestRealRepo_KnownEdges(t *testing.T) {
	root := repoRoot(t)
	g, err := BuildGraph(root)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	// Non-empty collections.
	if len(g.Services()) < 10 {
		t.Errorf("expected many services, got %d: %v", len(g.Services()), g.Services())
	}
	if len(g.Libs()) < 10 {
		t.Errorf("expected many libs, got %d: %v", len(g.Libs()), g.Libs())
	}

	// Known edges — update these if the corresponding go.mod files change.
	mustDep := func(mod, dep string) {
		t.Helper()
		for _, d := range g.DirectDeps(mod) {
			if d == dep {
				return
			}
		}
		t.Errorf("%s does not directly require %s; deps=%v", mod, dep, g.DirectDeps(mod))
	}
	mustDep("atlas-saga", "atlas-constants")
	mustDep("atlas-account", "atlas-kafka")
	mustDep("atlas-account", "atlas-tenant")
	mustDep("atlas-account", "atlas-rest")
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `cd tools/cideps && go test -run TestRealRepo -v`
Expected: PASS. If any `mustDep` fails, that's a real signal — either the graph builder is wrong or the referenced `go.mod` no longer has that require; inspect the relevant `go.mod` before assuming the test is wrong.

- [ ] **Step 3: Commit**

```bash
git add tools/cideps/realrepo_test.go
git commit -m "test(cideps): sanity check against real repo graph"
```

---

## Task 9: Integrate `cideps` into `detect-changes`

**Files:**
- Modify: `.github/actions/detect-changes/action.yml`

This task moves matrix construction *out of* `pr-validation.yml` and `main-publish.yml` and *into* the composite action, wrapping cideps with a fallback branch.

- [ ] **Step 1: Add matrix outputs to the action**

Open `.github/actions/detect-changes/action.yml`. In the `outputs:` block (currently ends with `has_workflow_changes` — note: the internal step outputs use underscores while the action outputs use dashes), add three new entries **after** the existing outputs, before `runs:`:

```yaml
  go-services-matrix:
    description: 'JSON array of Go services to test/build'
    value: ${{ steps.affected.outputs.go-services }}
  go-libraries-matrix:
    description: 'JSON array of Go libraries to test'
    value: ${{ steps.affected.outputs.go-libraries }}
  docker-services-matrix:
    description: 'JSON array of services to build Docker images for'
    value: ${{ steps.affected.outputs.docker-services }}
  selection-reason:
    description: 'Human-readable reason for the selection'
    value: ${{ steps.affected.outputs.reason }}
```

Also add one input the workflows can set:

```yaml
inputs:
  base-ref:
    ...existing...
  force-all:
    description: 'Force full rebuild (all services and libraries)'
    required: false
    default: 'false'
```

- [ ] **Step 2: Add the `Set up Go` + `Compute affected modules` steps**

Inside `runs.steps`, **after** the existing `Detect changed files and services` step (which has `id: detect`), append:

```yaml
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.25.5'
        cache: false

    - name: Compute affected modules
      id: affected
      shell: bash
      run: |
        set -eo pipefail

        CHANGED_LIBS=$(echo '${{ steps.detect.outputs.libraries }}' | jq -r '. // [] | join(",")')
        CHANGED_SERVICES=$(echo '${{ steps.detect.outputs.services }}' | jq -r '. // [] | join(",")')

        FORCE_FLAGS=""
        if [ "${{ steps.detect.outputs.has_go_workspace_changes }}" = "true" ] \
          || [ "${{ steps.detect.outputs.has_workflow_changes }}" = "true" ] \
          || [ "${{ inputs.force-all }}" = "true" ]; then
          FORCE_FLAGS="--force-all"
          echo "force-all triggered by: go.work=${{ steps.detect.outputs.has_go_workspace_changes }} workflow=${{ steps.detect.outputs.has_workflow_changes }} input=${{ inputs.force-all }}"
        fi

        CONFIG=".github/config/services.json"

        if OUT=$(go run ./tools/cideps \
            --changed-libs="$CHANGED_LIBS" \
            --changed-services="$CHANGED_SERVICES" \
            --config="$CONFIG" \
            $FORCE_FLAGS 2>cideps.err); then
          echo "$OUT" > affected.json
          echo "go-services=$(jq -c '."go-services"' affected.json)" >> $GITHUB_OUTPUT
          echo "go-libraries=$(jq -c '."go-libraries"' affected.json)" >> $GITHUB_OUTPUT
          echo "docker-services=$(jq -c '."docker-services"' affected.json)" >> $GITHUB_OUTPUT
          REASON=$(jq -r '.reason' affected.json)
          echo "reason=$REASON" >> $GITHUB_OUTPUT
          if [ -s cideps.err ]; then
            echo "::warning::cideps emitted warnings:"
            cat cideps.err
          fi
          {
            echo "### Affected modules";
            echo "$REASON";
            echo "";
            echo "- **Services**: $(jq -r '."go-services" | length' affected.json)";
            echo "- **Libraries**: $(jq -r '."go-libraries" | length' affected.json)";
            echo "- **Docker builds**: $(jq -r '."docker-services" | length' affected.json)";
          } >> "$GITHUB_STEP_SUMMARY"
        else
          echo "::warning::cideps failed with exit code $?, falling back to build-all"
          echo "cideps stderr:"
          cat cideps.err
          # Fallback: emit full matrices directly from services.json.
          echo "go-services=$(jq -c '[.services[] | select(.type == "go-service") | {name, path, module_path, docker_image}]' "$CONFIG")" >> $GITHUB_OUTPUT
          echo "go-libraries=$(jq -c '[.libraries[] | {name, path, module_path, coverage_threshold: (.coverage_threshold // 0)}]' "$CONFIG")" >> $GITHUB_OUTPUT
          echo "docker-services=$(jq -c '[.services[] | select(.docker_image != null) | {name, path, docker_context: (.docker_context // .path), docker_image}]' "$CONFIG")" >> $GITHUB_OUTPUT
          echo "reason=cideps failed; fell back to build-all" >> $GITHUB_OUTPUT
          {
            echo "### Affected modules";
            echo "cideps FAILED — fell back to full rebuild";
          } >> "$GITHUB_STEP_SUMMARY"
        fi
```

- [ ] **Step 3: Syntax-check the workflow**

Run: `yq '.' .github/actions/detect-changes/action.yml > /dev/null` (use `python -c "import yaml,sys; yaml.safe_load(open('.github/actions/detect-changes/action.yml'))"` if `yq` isn't installed).

Expected: exit 0, no parse errors.

- [ ] **Step 4: Commit**

```bash
git add .github/actions/detect-changes/action.yml
git commit -m "feat(ci): compute affected modules via cideps in detect-changes"
```

---

## Task 10: Switch `pr-validation.yml` to consume the new outputs

**Files:**
- Modify: `.github/workflows/pr-validation.yml`

- [ ] **Step 1: Plumb the new inputs/outputs through `detect-changes` job**

In `.github/workflows/pr-validation.yml`, edit the `detect-changes` job (starts at line ~26):

1. Pass `force-all` to the composite action. Change:

```yaml
      - name: Detect changes
        id: detect
        uses: ./.github/actions/detect-changes
```

to:

```yaml
      - name: Detect changes
        id: detect
        uses: ./.github/actions/detect-changes
        with:
          force-all: ${{ github.event.inputs.force-all }}
```

2. Replace the `go-services-matrix`, `go-libraries-matrix`, and `docker-services-matrix` outputs so they come from the action's new outputs instead of the inline step. In the `outputs:` block of the job (lines ~30-41), change the three matrix output lines to:

```yaml
      go-services-matrix: ${{ steps.detect.outputs.go-services-matrix }}
      go-libraries-matrix: ${{ steps.detect.outputs.go-libraries-matrix }}
      docker-services-matrix: ${{ steps.detect.outputs.docker-services-matrix }}
```

- [ ] **Step 2: Delete the inline `Build matrices` step**

Remove the entire `- name: Build matrices` step (starts at `id: matrix`, lines ~53–110 in the current file). It now lives in the composite action.

- [ ] **Step 3: Verify the workflow still parses**

Run: `python -c "import yaml; yaml.safe_load(open('.github/workflows/pr-validation.yml'))"`
Expected: no output, exit 0.

Also grep for any stale references: `grep -n 'steps.matrix' .github/workflows/pr-validation.yml` — expected: no matches.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/pr-validation.yml
git commit -m "refactor(ci): pr-validation consumes detect-changes matrix outputs"
```

---

## Task 11: Switch `main-publish.yml` to consume the new outputs

**Files:**
- Modify: `.github/workflows/main-publish.yml`

- [ ] **Step 1: Pass `force-all` into the composite action**

Change (lines ~45–48):

```yaml
      - name: Detect changes
        id: detect
        uses: ./.github/actions/detect-changes
```

to:

```yaml
      - name: Detect changes
        id: detect
        uses: ./.github/actions/detect-changes
        with:
          force-all: ${{ github.event.inputs.force-all }}
```

- [ ] **Step 2: Rewire the `docker-services-matrix` and `has-changes` job outputs**

In the job's `outputs:` block, change:

```yaml
      docker-services-matrix: ${{ steps.matrix.outputs.docker-services }}
      has-changes: ${{ steps.matrix.outputs.has-changes }}
```

to:

```yaml
      docker-services-matrix: ${{ steps.override.outputs.docker-services-matrix || steps.detect.outputs.docker-services-matrix }}
      has-changes: ${{ steps.has-changes.outputs.value }}
```

The `steps.override` and `steps.has-changes` step IDs are introduced in Step 3 below. GitHub Actions resolves `steps.<id>.outputs.*` at job-output evaluation time (after all steps run), so forward references are fine.

- [ ] **Step 3: Replace the inline `Build matrices` step with an override + has-changes pair**

Remove the entire `- name: Build matrices` step (lines ~49–98, starting `id: matrix`).

Replace it with **two steps in this order**: first the single-service override (which may set `docker-services-matrix`), then has-changes (which reads the effective matrix — override-wins-over-detect). Order matters: has-changes must run *after* override so it sees the overridden value.

```yaml
      - name: Apply single-service override
        id: override
        if: ${{ github.event.inputs.service != '' }}
        shell: bash
        run: |
          SPECIFIC="${{ github.event.inputs.service }}"
          echo "Overriding matrix to only build: $SPECIFIC"
          ROW=$(jq -c --arg name "$SPECIFIC" \
            '[.services[] | select(.name == $name) | select(.docker_image != null) | {name, path, docker_context: (.docker_context // .path), docker_image}]' \
            .github/config/services.json)
          echo "docker-services-matrix=$ROW" >> $GITHUB_OUTPUT

      - name: Compute has-changes flag and summary
        id: has-changes
        shell: bash
        env:
          OVERRIDE_MATRIX: ${{ steps.override.outputs.docker-services-matrix }}
          DETECT_MATRIX: ${{ steps.detect.outputs.docker-services-matrix }}
          REASON: ${{ steps.detect.outputs.selection-reason }}
        run: |
          if [ -n "$OVERRIDE_MATRIX" ]; then
            DOCKER="$OVERRIDE_MATRIX"
            EFFECTIVE_REASON="single-service override"
          else
            DOCKER="$DETECT_MATRIX"
            EFFECTIVE_REASON="$REASON"
          fi
          COUNT=$(echo "$DOCKER" | jq length)
          if [ "$COUNT" -gt 0 ]; then
            echo "value=true" >> $GITHUB_OUTPUT
          else
            echo "value=false" >> $GITHUB_OUTPUT
          fi

          {
            echo "### Publish Detection Summary";
            echo "- **Reason**: $EFFECTIVE_REASON";
            echo "- **Services to publish**: $COUNT";
            echo "";
          } >> "$GITHUB_STEP_SUMMARY"

          if [ "$COUNT" -gt 0 ]; then
            {
              echo "**Services:**";
              echo "$DOCKER" | jq -r '.[].name' | sed 's/^/- /';
            } >> "$GITHUB_STEP_SUMMARY"
          fi
```

Finally, update the job `outputs:` block's `docker-services-matrix` so downstream jobs consume the override-or-detect chain:

```yaml
      docker-services-matrix: ${{ steps.override.outputs.docker-services-matrix || steps.detect.outputs.docker-services-matrix }}
```

- [ ] **Step 4: Verify the workflow parses and references are consistent**

Run:

```bash
python -c "import yaml; yaml.safe_load(open('.github/workflows/main-publish.yml'))"
grep -n 'steps.matrix' .github/workflows/main-publish.yml
```

Expected: no YAML error; no `steps.matrix` references remain.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/main-publish.yml
git commit -m "refactor(ci): main-publish consumes detect-changes matrix outputs"
```

---

## Task 12: End-to-end smoke verification

**Files:**
- None. This task is a guided local sanity check before opening the PR.

- [ ] **Step 1: Run all cideps tests**

Run: `cd tools/cideps && go test ./... -v`
Expected: all tests pass.

- [ ] **Step 2: Run full Go build for the tool**

Run: `cd tools/cideps && go build ./...`
Expected: exit 0.

- [ ] **Step 3: Exercise cideps against three realistic scenarios**

From the repo root:

```bash
# 1. One lib change — should hit a subset of services
go run ./tools/cideps \
  --changed-libs=atlas-saga \
  --config=.github/config/services.json | jq '."go-services" | length, .reason'

# 2. A very-fanout lib — should hit most services
go run ./tools/cideps \
  --changed-libs=atlas-kafka \
  --config=.github/config/services.json | jq '."go-services" | length, .reason'

# 3. No changes — should emit empty arrays
go run ./tools/cideps --config=.github/config/services.json | jq '.'
```

Expected:
- Case 1: a modest number (likely <20), reason mentions the lib.
- Case 2: a larger number, possibly close to all services.
- Case 3: all three arrays are `[]`.

If case 2 returns *all* services, that's expected if `atlas-kafka` is universally required, and equivalent to today's behavior for that specific lib.

- [ ] **Step 4: Lint the action YAML files one more time**

```bash
for f in .github/actions/detect-changes/action.yml .github/workflows/pr-validation.yml .github/workflows/main-publish.yml; do
  python -c "import yaml; yaml.safe_load(open('$f'))" || { echo "YAML FAIL: $f"; exit 1; }
done
```

Expected: no output, exit 0.

- [ ] **Step 5: Push branch and open PR**

```bash
git push -u origin task-018-ci-dep-graph
gh pr create --fill
```

The PR itself will exercise the new CI path end-to-end. Watch the `Detect Changes` step summary to confirm the "Affected modules" section matches what cideps emitted locally.

---

## Self-Review Notes

- **Spec coverage:** every section of `design.md` has a matching task — tool (tasks 1–8), integration (task 9), workflow updates (tasks 10–11), validation (task 12). Error handling covered by the fallback branch in task 9. Edge cases (indirect deps, unknown names, missing services.json entries) are tested in tasks 2, 5, and 6.
- **Known gaps / deliberate omissions:** no Dockerfile lint (flagged as out-of-scope in the design); no phased rollout (dropped during brainstorming — fallback branch is the insurance).
- **Type consistency:** `Graph.DirectDeps`, `Graph.Closure`, `Graph.Libs`, `Graph.Services` are referenced consistently across tasks 3–8 with the same signatures. `SelectInput`, `Selection`, `GoServiceRow`, `GoLibraryRow`, `DockerServiceRow` are defined once and reused. The JSON field names (`go-services`, `go-libraries`, `docker-services`, `reason`) match between the Go output struct and the YAML consumer in task 9.
