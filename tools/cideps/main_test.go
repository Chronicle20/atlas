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
