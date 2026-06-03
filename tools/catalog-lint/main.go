package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: catalog-lint <root>")
		os.Exit(2)
	}
	root := os.Args[1]
	if err := lint(root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func lint(root string) error {
	var errs []string

	// 1. Each <region>/<major>_<minor>/ dir must contain a non-empty CATALOG_REVISION.
	regionEntries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read root: %w", err)
	}
	for _, regionEntry := range regionEntries {
		if !regionEntry.IsDir() || strings.HasPrefix(regionEntry.Name(), "_") {
			continue
		}
		regionDir := filepath.Join(root, regionEntry.Name())
		versionEntries, _ := os.ReadDir(regionDir)
		for _, versionEntry := range versionEntries {
			if !versionEntry.IsDir() {
				continue
			}
			versionDir := filepath.Join(regionDir, versionEntry.Name())
			rev, _ := os.ReadFile(filepath.Join(versionDir, "CATALOG_REVISION"))
			if len(strings.TrimSpace(string(rev))) == 0 {
				errs = append(errs, fmt.Sprintf("%s: missing or empty CATALOG_REVISION", versionDir))
			}
		}
	}

	// 2. Walk every *.json file. Skip names starting with _ or . and dirs starting with _ or .
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, _ error) error {
		base := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(base, "_") || strings.HasPrefix(base, ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(base, "_") || strings.HasPrefix(base, ".") {
			return nil
		}
		if !strings.HasSuffix(base, ".json") {
			return nil
		}
		// Determine the subdomain rule by walking from versionDir.
		rel, _ := filepath.Rel(root, path)
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 4 { // <region>/<version>/<subdomain-path>/<file>
			return nil
		}
		subdomainPath := strings.Join(parts[2:len(parts)-1], "/")
		rule, ok := ruleFor(subdomainPath)
		if !ok {
			return nil // unrecognized subdomain — not an error per se
		}
		b, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: read: %v", path, err))
			return nil
		}
		env, err := seeder.ParseEnvelope(b)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			return nil
		}
		if env.Data.Type != rule.typ {
			errs = append(errs, fmt.Sprintf("%s: type %q, want %q", path, env.Data.Type, rule.typ))
		}
		if rule.pattern != nil {
			id, err := seeder.ExtractEntityID(base, rule.pattern)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", path, err))
				return nil
			}
			if id != env.Data.ID {
				errs = append(errs, fmt.Sprintf("%s: data.id %q, filename id %q", path, env.Data.ID, id))
			}
		}
		return nil
	})

	if len(errs) > 0 {
		return fmt.Errorf("linter found %d issue(s):\n%s", len(errs), strings.Join(errs, "\n"))
	}
	return nil
}
