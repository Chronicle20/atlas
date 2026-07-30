// Command gen is the atlas-constants identity generator (task-187 Tasks 2 + 4).
//
// Run from libs/atlas-constants/gen:
//
//	go run .                       # regenerate skill/job identities_gen.go AND all 22 per-version version_*_gen.go
//	go run . -check                # drift check: exit 1 if any generated file is stale
//	go run . -bootstrap-identities # one-off: re-derive identities.yaml from the current *Id = Id(N) const blocks
//	go run . -author-semantics     # one-off: re-derive gen/semantics/<r>_<maj>_<min>.yaml from divergences.csv
//
// identities.yaml is the checked-in source of truth for the Identity
// namespace (task 2). gen/semantics/<r>_<maj>_<min>.yaml (task 4) is the
// checked-in per-version source of truth for wireId<->Identity binding;
// -author-semantics re-derives it from
// docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv.
//
// -bootstrap-identities is a one-off extraction tool (task-2 brief Step 1)
// -- it overwrites identities.yaml purely from the mechanical const-block
// scan and does NOT preserve hand-added entries (e.g. the Big-Bang-
// introduced identities with no v83 constant); re-run it only when
// deliberately re-deriving the const-sourced baseline, then re-apply the
// hand-added entries. Likewise -author-semantics overwrites every
// gen/semantics/*.yaml file from divergences.csv; re-run it only when
// divergences.csv itself changed.
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
)

const (
	identitiesYAMLPath = "identities.yaml"
	skillConstantsPath = "../skill/constants.go"
	jobConstantsPath   = "../job/constants.go"
	skillGenPath       = "../skill/identities_gen.go"
	jobGenPath         = "../job/identities_gen.go"
)

func main() {
	bootstrap := flag.Bool("bootstrap-identities", false, "re-derive identities.yaml from the skill/job const blocks and overwrite it")
	authorSemantics := flag.Bool("author-semantics", false, "re-derive gen/semantics/<r>_<maj>_<min>.yaml from divergences.csv and overwrite them")
	check := flag.Bool("check", false, "drift check: exit 1 if any generated file is stale relative to its source of truth")
	flag.Parse()

	if *bootstrap {
		if err := runBootstrap(); err != nil {
			fmt.Fprintln(os.Stderr, "bootstrap failed:", err)
			os.Exit(1)
		}
		return
	}

	if *authorSemantics {
		if err := runAuthorSemantics(); err != nil {
			fmt.Fprintln(os.Stderr, "author-semantics failed:", err)
			os.Exit(1)
		}
		return
	}

	ids, err := LoadIdentities(identitiesYAMLPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load failed:", err)
		os.Exit(1)
	}
	if err := ValidateIdentityTokens(ids); err != nil {
		fmt.Fprintln(os.Stderr, "validation failed:", err)
		os.Exit(1)
	}

	skillGo, jobGo := EmitIdentities(ids)
	skillFmt, err := gofmtOrRaw(skillGo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "formatting skill output failed:", err)
		os.Exit(1)
	}
	jobFmt, err := gofmtOrRaw(jobGo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "formatting job output failed:", err)
		os.Exit(1)
	}

	if *check {
		if err := checkDrift(skillGenPath, skillFmt); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := checkDrift(jobGenPath, jobFmt); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		nVersionFiles, err := emitAllVersionFiles(true, gofmtOrRaw, checkDrift, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("OK: skill/identities_gen.go, job/identities_gen.go, and %d per-version version_*_gen.go files are up to date\n", nVersionFiles)
		return
	}

	if err := os.WriteFile(skillGenPath, skillFmt, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "writing", skillGenPath, "failed:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(jobGenPath, jobFmt, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "writing", jobGenPath, "failed:", err)
		os.Exit(1)
	}

	nSkill, nJob := countByDomain(ids)
	fmt.Printf("wrote %s (%d identities) and %s (%d identities)\n", skillGenPath, nSkill, jobGenPath, nJob)

	nVersionFiles, err := emitAllVersionFiles(false, gofmtOrRaw, nil, writeGeneratedFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "emitting per-version semantics files failed:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d per-version version_*_gen.go files (skill+job x %d versions)\n", nVersionFiles, len(provisionedVersions))
}

func writeGeneratedFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func gofmtOrRaw(src string) ([]byte, error) {
	out, err := format.Source([]byte(src))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func checkDrift(path string, want []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w (run `go run .` to generate it)", path, err)
	}
	if string(got) != string(want) {
		return fmt.Errorf("%s is stale relative to identities.yaml -- run `go run .` to regenerate", path)
	}
	return nil
}

func countByDomain(ids []IdentityEntry) (nSkill, nJob int) {
	for _, id := range ids {
		switch id.Domain {
		case "skill":
			nSkill++
		case "job":
			nJob++
		}
	}
	return
}

func runBootstrap() error {
	ids, err := bootstrapFromConstants(skillConstantsPath, jobConstantsPath)
	if err != nil {
		return err
	}
	if err := ValidateIdentityTokens(ids); err != nil {
		return err
	}
	out, err := marshalIdentities(ids)
	if err != nil {
		return fmt.Errorf("marshalling identities.yaml: %w", err)
	}
	if err := os.WriteFile(identitiesYAMLPath, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", identitiesYAMLPath, err)
	}
	nSkill, nJob := countByDomain(ids)
	fmt.Printf("bootstrapped %s: %d skill identities, %d job identities\n", identitiesYAMLPath, nSkill, nJob)
	return nil
}
