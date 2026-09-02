package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// versionDirPattern matches the version-directory naming convention the
// seeder itself emits: fmt.Sprintf("%d_%d", major, minor) in
// libs/atlas-seeder/catalog.go's filesystemSource.Roots. A directory that
// matches this shape is a genuine <region>/<major>_<minor> version root
// regardless of what subdomains it happens to carry; one that doesn't
// (e.g. deploy/seed/shared/all, whose "version" segment is the literal
// string "all") is the version-agnostic shared root documented in
// tools/catalog-lint/subdomains.go and libs/atlas-seeder's
// NewFilesystemCatalogSourceWithShared, not a map-action version root.
var versionDirPattern = regexp.MustCompile(`^\d+_\d+$`)

// mapActionDoc is one map-action seed document gathered during the tree
// walk, carrying enough context to run the replication, spawn-guard and
// schema checks in checkMapActions.
type mapActionDoc struct {
	path    string
	region  string
	version string
	hook    string
	id      string
	raw     []byte
}

// mapActionAttributes is the subset of a map-action document's
// data.attributes needed to check the spawnIfAbsent guard. Params stay
// map[string]string per the wire contract: every operation param is a
// string.
type mapActionAttributes struct {
	Rules []struct {
		ID         string `json:"id"`
		Operations []struct {
			Type   string            `json:"type"`
			Params map[string]string `json:"params"`
		} `json:"operations"`
	} `json:"rules"`
}

// mapActionSchemaPath resolves services/atlas-map-actions/docs/map_script_schema.json
// relative to the repository root. It returns ok=false when the repository
// root cannot be determined (not a git checkout) or the schema file does not
// exist there — callers must surface that explicitly rather than silently
// skipping schema validation.
func mapActionSchemaPath() (string, bool) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", false
	}
	toplevel := strings.TrimSpace(string(out))
	if toplevel == "" {
		return "", false
	}
	p := filepath.Join(toplevel, "services", "atlas-map-actions", "docs", "map_script_schema.json")
	if _, err := os.Stat(p); err != nil {
		return "", false
	}
	return p, true
}

// checkMapActions runs the three map-action seed invariants (design §6,
// PRD FR-1.6, FR-2.2): byte-identical replication across every version
// root, a "spawnIfAbsent": "true" guard on every spawn_monster operation,
// and schema validity of data.attributes. It returns one message per
// violation; an empty slice means the tree is clean.
func checkMapActions(root string, docs []mapActionDoc, schemaPath string) []string {
	var errs []string

	errs = append(errs, checkMapActionReplication(root, docs)...)
	errs = append(errs, checkMapActionSpawnGuards(docs)...)
	errs = append(errs, checkMapActionSchema(docs, schemaPath)...)

	return errs
}

func rootOf(d mapActionDoc) string {
	return d.region + "/" + d.version
}

func relKeyOf(d mapActionDoc) string {
	return "map-actions/" + d.hook + "/" + filepath.Base(d.path)
}

// discoverRoots re-reads the top two directory levels of root
// (<region>/<version>) to find every genuine version root, including ones
// that hold no map-action document at all — the case a doc-derived root
// set alone would miss (a root missing its entire map-actions/ directory
// must still surface as a replication violation, not escape the check
// silently). A directory counts as a version root when its name matches
// versionDirPattern; deploy/seed/shared/all does not (its version segment
// is "all", not "<major>_<minor>") and is excluded, whether or not it
// happens to carry a map-actions/ directory.
func discoverRoots(root string) []string {
	var roots []string
	regionEntries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	for _, re := range regionEntries {
		if !re.IsDir() || strings.HasPrefix(re.Name(), "_") {
			continue
		}
		versionEntries, err := os.ReadDir(filepath.Join(root, re.Name()))
		if err != nil {
			continue
		}
		for _, ve := range versionEntries {
			if !ve.IsDir() || !versionDirPattern.MatchString(ve.Name()) {
				continue
			}
			roots = append(roots, re.Name()+"/"+ve.Name())
		}
	}
	return roots
}

func checkMapActionReplication(root string, docs []mapActionDoc) []string {
	var errs []string

	allRoots := map[string]bool{}
	for _, r := range discoverRoots(root) {
		allRoots[r] = true
	}
	for _, d := range docs {
		allRoots[rootOf(d)] = true
	}
	var sortedRoots []string
	for r := range allRoots {
		sortedRoots = append(sortedRoots, r)
	}
	sort.Strings(sortedRoots)

	groups := map[string][]mapActionDoc{}
	var groupKeys []string
	for _, d := range docs {
		key := relKeyOf(d)
		if _, ok := groups[key]; !ok {
			groupKeys = append(groupKeys, key)
		}
		groups[key] = append(groups[key], d)
	}
	sort.Strings(groupKeys)

	for _, key := range groupKeys {
		members := groups[key]
		byRoot := map[string]mapActionDoc{}
		for _, m := range members {
			byRoot[rootOf(m)] = m
		}

		for _, r := range sortedRoots {
			if _, ok := byRoot[r]; !ok {
				// Report against the first root that does have it, so the
				// message names a concrete present/missing pair.
				var present string
				for _, r2 := range sortedRoots {
					if _, ok := byRoot[r2]; ok {
						present = r2
						break
					}
				}
				errs = append(errs, fmt.Sprintf("%s: present in %s, missing from %s", key, present, r))
			}
		}

		if len(sortedRoots) == 0 {
			continue
		}
		var first mapActionDoc
		var firstRoot string
		haveFirst := false
		for _, r := range sortedRoots {
			if m, ok := byRoot[r]; ok {
				first = m
				firstRoot = r
				haveFirst = true
				break
			}
		}
		if !haveFirst {
			continue
		}
		for _, r := range sortedRoots {
			if r == firstRoot {
				continue
			}
			m, ok := byRoot[r]
			if !ok {
				continue
			}
			if !bytes.Equal(first.raw, m.raw) {
				errs = append(errs, fmt.Sprintf("%s: differs between %s and %s", key, firstRoot, r))
			}
		}
	}

	return errs
}

func checkMapActionSpawnGuards(docs []mapActionDoc) []string {
	var errs []string

	for _, d := range docs {
		env, err := parseMapActionAttributes(d.raw)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", d.path, err))
			continue
		}
		for _, rule := range env.Rules {
			for i, op := range rule.Operations {
				if op.Type != "spawn_monster" {
					continue
				}
				if op.Params["spawnIfAbsent"] != "true" {
					errs = append(errs, fmt.Sprintf("%s: rule %q operation %d: spawn_monster requires \"spawnIfAbsent\": \"true\"", d.path, rule.ID, i+1))
				}
			}
		}
	}

	return errs
}

func checkMapActionSchema(docs []mapActionDoc, schemaPath string) []string {
	if schemaPath == "" {
		return nil
	}

	compiler := jsonschema.NewCompiler()
	sch, err := compiler.Compile(schemaPath)
	if err != nil {
		return []string{fmt.Sprintf("map-action schema: compile %s: %v", schemaPath, err)}
	}

	var errs []string
	for _, d := range docs {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(d.raw, &raw); err != nil {
			errs = append(errs, fmt.Sprintf("%s: schema: %v", d.path, err))
			continue
		}
		var data map[string]json.RawMessage
		if err := json.Unmarshal(raw["data"], &data); err != nil {
			errs = append(errs, fmt.Sprintf("%s: schema: %v", d.path, err))
			continue
		}
		var attrs any
		if err := json.Unmarshal(data["attributes"], &attrs); err != nil {
			errs = append(errs, fmt.Sprintf("%s: schema: %v", d.path, err))
			continue
		}
		if err := sch.Validate(attrs); err != nil {
			errs = append(errs, fmt.Sprintf("%s: schema: %v", d.path, err))
		}
	}

	return errs
}

func parseMapActionAttributes(raw []byte) (mapActionAttributes, error) {
	var env struct {
		Data struct {
			Attributes mapActionAttributes `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return mapActionAttributes{}, fmt.Errorf("parse map-action attributes: %w", err)
	}
	return env.Data.Attributes, nil
}
