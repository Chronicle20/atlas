// Command gen is the atlas-constants identity generator (task-187 Task 2).
//
// Run from libs/atlas-constants/gen:
//
//	go run .                       # regenerate skill/identities_gen.go + job/identities_gen.go from identities.yaml
//	go run . -check                # drift check: exit 1 if the generated files are stale
//	go run . -bootstrap-identities # one-off: re-derive identities.yaml from the current *Id = Id(N) const blocks
//
// identities.yaml is the checked-in source of truth. -bootstrap-identities
// is a one-off extraction tool (task-2 brief Step 1) -- it overwrites
// identities.yaml purely from the mechanical const-block scan and does NOT
// preserve hand-added entries (e.g. the Big-Bang-introduced identities with
// no v83 constant); re-run it only when deliberately re-deriving the
// const-sourced baseline, then re-apply the hand-added entries.
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
	check := flag.Bool("check", false, "drift check: exit 1 if skill/identities_gen.go or job/identities_gen.go are stale relative to identities.yaml")
	flag.Parse()

	if *bootstrap {
		if err := runBootstrap(); err != nil {
			fmt.Fprintln(os.Stderr, "bootstrap failed:", err)
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
		fmt.Println("OK: skill/identities_gen.go and job/identities_gen.go are up to date")
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
