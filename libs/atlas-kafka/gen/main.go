// Command gen is the atlas-kafka topic manifest generator (task-276 Task 5).
//
// Run from libs/atlas-kafka/gen:
//
//	go run .          # scan the workspace and (re)write topics.yaml
//	go run . -check   # drift check: exit 1 if topics.yaml is stale
//
// It loads the whole go.work workspace via golang.org/x/tools/go/packages,
// collects every topic.Token constant declared under services/ and libs/,
// applies policies.yaml's cleanup policy, and emits topics.yaml -- the
// checked-in source of truth Tasks 6, 7, and 8 render ConfigMaps, overlays,
// and the drift gate from.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	topicsYAMLPath      = "topics.yaml"
	envConfigMapPath    = "../../../deploy/k8s/base/env-configmap.yaml"
	topicsConfigMapPath = "../../../deploy/k8s/base/kafka-topics-configmap.yaml"
)

const (
	envConfigMapBeginMarker = "# BEGIN generated:topics (libs/atlas-kafka/gen -- run tools/gen-topics.sh)"
	envConfigMapEndMarker   = "# END generated:topics"
)

func main() {
	check := flag.Bool("check", false, "drift check: exit 1 if any generated file is stale relative to the workspace scan")
	flag.Parse()

	repoRoot, err := repoRootFromGit()
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolving repo root failed:", err)
		os.Exit(1)
	}

	m, err := Scan(repoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		os.Exit(1)
	}
	topicsYAML := m.EmitTopicsYAML()
	topicsConfigMap := m.EmitTopicsConfigMap()

	envConfigMap, err := renderEnvConfigMap(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rendering", envConfigMapPath, "failed:", err)
		os.Exit(1)
	}

	if *check {
		var failed bool
		for _, f := range []struct {
			path string
			want []byte
		}{
			{topicsYAMLPath, topicsYAML},
			{envConfigMapPath, envConfigMap},
			{topicsConfigMapPath, topicsConfigMap},
		} {
			if err := checkDrift(f.path, f.want); err != nil {
				fmt.Fprintln(os.Stderr, err)
				failed = true
			}
		}
		if failed {
			os.Exit(1)
		}
		fmt.Printf("OK: generated files are up to date (%d topics)\n", len(m.Topics))
		return
	}

	for _, f := range []struct {
		path string
		out  []byte
	}{
		{topicsYAMLPath, topicsYAML},
		{envConfigMapPath, envConfigMap},
		{topicsConfigMapPath, topicsConfigMap},
	} {
		if err := os.WriteFile(f.path, f.out, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "writing", f.path, "failed:", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%d topics)\n", f.path, len(m.Topics))
	}
}

// renderEnvConfigMap splices m's generated topic block into the existing
// deploy/k8s/base/env-configmap.yaml, preserving every hand-written key and
// comment outside the marker region.
func renderEnvConfigMap(m Manifest) ([]byte, error) {
	existing, err := os.ReadFile(envConfigMapPath)
	if err != nil {
		return nil, err
	}
	return Splice(existing, envConfigMapBeginMarker, envConfigMapEndMarker, m.EmitEnvConfigMapBlock())
}

// repoRootFromGit resolves the repository root so Scan can be run from
// libs/atlas-kafka/gen -- a module deliberately not listed in go.work --
// while still loading the full workspace.
func repoRootFromGit() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
