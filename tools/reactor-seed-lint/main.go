package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
)

var reactorFilePattern = regexp.MustCompile(`^reactor-(.+)\.json$`)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: reactor-seed-lint <seed-root> <schema-path>")
		os.Exit(2)
	}
	root := os.Args[1]
	schemaPath := os.Args[2]
	if err := lint(root, schemaPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type reactorScript struct {
	ReactorId string `json:"reactorId"`
}

func lint(root, schemaPath string) error {
	schema, err := compileSchema(schemaPath)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	var errs []string

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
		if !reactorFilePattern.MatchString(base) {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 5 { // <region>/<version>/reactor-actions/reactors/<file>
			return nil
		}
		subdomainPath := strings.Join(parts[2:len(parts)-1], "/")
		if subdomainPath != "reactor-actions/reactors" {
			return nil
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

		if env.Data.Type != "reactor-action" {
			errs = append(errs, fmt.Sprintf("%s: data.type %q, want %q", path, env.Data.Type, "reactor-action"))
		}

		fileID, err := seeder.ExtractEntityID(base, reactorFilePattern)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			return nil
		}
		if fileID != env.Data.ID {
			errs = append(errs, fmt.Sprintf("%s: data.id %q, filename id %q", path, env.Data.ID, fileID))
		}

		var script reactorScript
		if err := json.Unmarshal(env.Data.Attributes, &script); err != nil {
			errs = append(errs, fmt.Sprintf("%s: parse attributes: %v", path, err))
			return nil
		}
		if script.ReactorId != env.Data.ID {
			errs = append(errs, fmt.Sprintf("%s: attributes.reactorId %q, data.id %q", path, script.ReactorId, env.Data.ID))
		}

		if err := validateAttributes(schema, env.Data.Attributes); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
		}

		return nil
	})

	if len(errs) > 0 {
		return fmt.Errorf("linter found %d issue(s):\n%s", len(errs), strings.Join(errs, "\n"))
	}
	return nil
}
