package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
