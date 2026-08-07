package main

import "testing"

// TestGeneratedFilesMatchSource is the test-level twin of `go run . -check`
// (see main.go's -check flag): it re-runs the full generator emission
// in-process -- identities_gen.go for both domains, every per-version
// version_<r>_<maj>_<min>_gen.go, constants/registry_gen.go, and
// skill/job's baseline_gen.go -- and diffs each byte-for-byte (via the same
// checkDrift helper main.go's -check path uses) against what's checked in.
// It fails the moment any generated file has drifted from its source of
// truth (identities.yaml, gen/semantics/*.yaml, provisionedVersions),
// which happens whenever those sources change without a `go run .`
// regeneration -- the class of bug `go build`/`go test` on the generated
// package alone cannot catch, because stale-but-syntactically-valid
// generated Go still compiles and (usually) still passes its own tests.
//
// This intentionally reuses main.go's own check-mode machinery
// (LoadIdentities, EmitIdentities, emitAllVersionFiles, checkRegistryAndBaseline,
// checkDrift, gofmtOrRaw) rather than re-implementing drift detection, so
// there is exactly one definition of "what generated output should look
// like" for both the CLI (`go run . -check`) and CI to share.
func TestGeneratedFilesMatchSource(t *testing.T) {
	ids, err := LoadIdentities(identitiesYAMLPath)
	if err != nil {
		t.Fatalf("load %s failed: %v", identitiesYAMLPath, err)
	}
	if err := ValidateIdentityTokens(ids); err != nil {
		t.Fatalf("identity token validation failed: %v", err)
	}

	skillGo, jobGo := EmitIdentities(ids)
	skillFmt, err := gofmtOrRaw(skillGo)
	if err != nil {
		t.Fatalf("formatting %s output failed: %v", skillGenPath, err)
	}
	jobFmt, err := gofmtOrRaw(jobGo)
	if err != nil {
		t.Fatalf("formatting %s output failed: %v", jobGenPath, err)
	}

	if err := checkDrift(skillGenPath, skillFmt); err != nil {
		t.Error(err)
	}
	if err := checkDrift(jobGenPath, jobFmt); err != nil {
		t.Error(err)
	}

	nVersionFiles, err := emitAllVersionFiles(true, gofmtOrRaw, checkDrift, nil)
	if err != nil {
		t.Fatalf("per-version semantics emission drifted: %v", err)
	}
	if nVersionFiles == 0 {
		t.Fatal("emitAllVersionFiles emitted 0 version files -- provisionedVersions is unexpectedly empty")
	}

	if err := checkRegistryAndBaseline(gofmtOrRaw, checkDrift, nil); err != nil {
		t.Fatalf("registry/baseline emission drifted: %v", err)
	}
}
