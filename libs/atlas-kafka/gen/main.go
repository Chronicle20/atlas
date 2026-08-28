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

const topicsYAMLPath = "topics.yaml"

func main() {
	check := flag.Bool("check", false, "drift check: exit 1 if topics.yaml is stale relative to the workspace scan")
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
	out := m.EmitTopicsYAML()

	if *check {
		if err := checkDrift(topicsYAMLPath, out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("OK: %s is up to date (%d topics)\n", topicsYAMLPath, len(m.Topics))
		return
	}

	if err := os.WriteFile(topicsYAMLPath, out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "writing", topicsYAMLPath, "failed:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d topics)\n", topicsYAMLPath, len(m.Topics))
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
