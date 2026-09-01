package main

import (
	"fmt"
	"go/constant"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v3"
)

// tokenTypePkgPath and tokenTypeName identify the topic.Token named type
// (libs/atlas-kafka/topic/token.go) that every topic constant must be
// declared with. A *types.Const only counts toward the manifest if its
// type resolves to exactly this named type.
const (
	tokenTypePkgPath = "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	tokenTypeName    = "Token"
)

// policiesYAMLRelPath is policies.yaml's path relative to the repo root.
const policiesYAMLRelPath = "libs/atlas-kafka/gen/policies.yaml"

// Scan loads the go.work workspace rooted at repoRoot, collects every
// topic.Token constant declared under services/ and libs/, and applies
// policies.yaml's cleanup policy to produce a Manifest.
//
// A partial go/packages load (any package reporting an error) is a hard
// error -- it must never silently produce an incomplete manifest.
func Scan(repoRoot string) (Manifest, error) {
	tokens, err := scanTokens(repoRoot)
	if err != nil {
		return Manifest{}, err
	}

	pol, err := loadPolicy(filepath.Join(repoRoot, policiesYAMLRelPath))
	if err != nil {
		return Manifest{}, err
	}

	return applyPolicy(tokens, pol)
}

// scanTokens loads every package under services/ and libs/ (via the
// workspace's go.work, which this module deliberately is not a member of --
// GOWORK is set explicitly so the load sees all 95 `use` directives) and
// returns every topic.Token constant found, keyed by its string value, with
// the sorted, de-duplicated set of packages (by PkgPath) that declare it.
func scanTokens(repoRoot string) (map[string][]string, error) {
	patterns, err := workspaceLoadPatterns(repoRoot)
	if err != nil {
		return nil, err
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo,
		Dir:   repoRoot,
		Tests: false, // FR-1.7: _test.go tokens drop out structurally
		Env:   append(os.Environ(), "GOWORK="+filepath.Join(repoRoot, "go.work")),
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("loading workspace: %w", err)
	}

	// FR-2.3: a partial load must never produce a manifest.
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return nil, fmt.Errorf("package %s failed to load: %s", pkg.PkgPath, pkg.Errors[0])
		}
	}

	tokens := make(map[string][]string)
	for _, pkg := range pkgs {
		if pkg.Types == nil {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			c, ok := scope.Lookup(name).(*types.Const)
			if !ok {
				continue
			}
			if !isTokenType(c.Type()) {
				continue
			}
			val := constant.StringVal(c.Val())
			tokens[val] = appendUnique(tokens[val], pkg.PkgPath)
		}
	}
	for tok := range tokens {
		sort.Strings(tokens[tok])
	}
	return tokens, nil
}

// workspaceLoadPatterns reads go.work's `use` directives and returns a
// `<dir>/...` load pattern for every one rooted under services/ or libs/
// (tools/ is excluded -- brief FR scope is topic constants declared by
// services and shared libraries).
//
// This can't be the two static globs "./services/..." and "./libs/..."
// (the brief's illustrative snippet): in workspace mode the go command
// only accepts a directory pattern that exactly matches -- or is nested
// inside -- one of go.work's `use` directories, and every services/*
// module lives one level deeper than its service directory (e.g.
// services/atlas-account/atlas.com/account), so "./services/..." errors
// with "directory prefix services does not contain modules listed in
// go.work". Building the pattern list from the `use` directives themselves
// is the only form that is not one edit away from silently discovering
// zero packages, which -- unlike a load error -- FR-2.3's partial-load
// guard cannot catch.
func workspaceLoadPatterns(repoRoot string) ([]string, error) {
	goWorkPath := filepath.Join(repoRoot, "go.work")
	data, err := os.ReadFile(goWorkPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", goWorkPath, err)
	}
	wf, err := modfile.ParseWork(goWorkPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", goWorkPath, err)
	}

	var patterns []string
	for _, use := range wf.Use {
		dir := filepath.ToSlash(use.Path)
		dir = strings.TrimPrefix(dir, "./")
		if strings.HasPrefix(dir, "services/") || strings.HasPrefix(dir, "libs/") {
			patterns = append(patterns, "./"+dir+"/...")
		}
	}
	if len(patterns) == 0 {
		return nil, fmt.Errorf("%s: no `use` directives under services/ or libs/", goWorkPath)
	}
	sort.Strings(patterns)
	return patterns, nil
}

// isTokenType reports whether t is exactly the topic.Token named type.
func isTokenType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == tokenTypePkgPath && obj.Name() == tokenTypeName
}

func appendUnique(pkgs []string, pkgPath string) []string {
	for _, p := range pkgs {
		if p == pkgPath {
			return pkgs
		}
	}
	return append(pkgs, pkgPath)
}

// policy is policies.yaml's shape: the hand-authored, single home for
// cleanup policy (design §6). Every token listed here must exist in the
// scan; there is deliberately no code path for a token appearing in more
// than one policy list, because compact is currently the only non-default
// policy.
type policy struct {
	Compact []string `yaml:"compact"`
}

// loadPolicy reads policies.yaml from path.
func loadPolicy(path string) (policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return policy{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var pol policy
	if err := yaml.Unmarshal(data, &pol); err != nil {
		return policy{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return pol, nil
}

// applyPolicy merges the scanned tokens with pol's cleanup policy into a
// sorted Manifest. Every token pol.Compact names must appear in tokens --
// a stale policy entry (naming a token that no longer exists) is an error,
// never a silently-dropped policy line.
func applyPolicy(tokens map[string][]string, pol policy) (Manifest, error) {
	compact := make(map[string]bool, len(pol.Compact))
	for _, tok := range pol.Compact {
		if _, ok := tokens[tok]; !ok {
			return Manifest{}, fmt.Errorf("policies.yaml: compact token %q not found by the scan", tok)
		}
		compact[tok] = true
	}

	entries := make([]Entry, 0, len(tokens))
	for tok, pkgs := range tokens {
		cleanup := "delete"
		if compact[tok] {
			cleanup = "compact"
		}
		sorted := append([]string(nil), pkgs...)
		sort.Strings(sorted)
		entries = append(entries, Entry{Token: tok, Cleanup: cleanup, Packages: sorted})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Token < entries[j].Token })

	return Manifest{Topics: entries}, nil
}
