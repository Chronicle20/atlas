package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A tiny status.json with one packet whose cells we control per test. gms_v87
// and gms_v95 straddle the boundary under test.
func writeGateStatus(t *testing.T, dir, v87State, v95State string) string {
	t.Helper()
	js := `{
  "toolSha": "x",
  "exportHashes": {},
  "rows": [
    {
      "kind": "op",
      "op": "CHARACTER_SPAWN",
      "packet": "character/clientbound/CharacterSpawn",
      "direction": "clientbound",
      "cells": {
        "gms_v84": {"state": "verified"},
        "gms_v87": {"state": "` + v87State + `"},
        "gms_v95": {"state": "` + v95State + `"}
      }
    }
  ]
}`
	p := filepath.Join(dir, "status.json")
	if err := os.WriteFile(p, []byte(js), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func writeGates(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "gates.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// PASSES clean: a gate whose both straddling versions are verified. FAILS: the
// same gate when the upper side has no verified fixture. (task-169 T4.2 /
// FR-3.1b — both directions.)
func TestGateCheckBothDirections(t *testing.T) {
	gates := `gates:
  - packet: character/clientbound/CharacterSpawn
    direction: clientbound
    field: v88+ spawn field
    boundary: ">87"
    lower_version_key: gms_v87
    upper_version_key: gms_v95
`
	dir := t.TempDir()
	gp := writeGates(t, dir, gates)

	// clean: both verified → --check exits 0.
	sp := writeGateStatus(t, dir, "verified", "verified")
	var out, errb bytes.Buffer
	if rc := gateCheckRun(gateCheckConfig{GatesPath: gp, StatusPath: sp, Check: true}, &out, &errb); rc != 0 {
		t.Fatalf("both-verified: want exit 0, got %d; stderr=%s", rc, errb.String())
	}

	// violation: upper (gms_v95) unverified → --check exits non-zero and names it.
	sp2dir := t.TempDir()
	sp2 := writeGateStatus(t, sp2dir, "verified", "incomplete")
	out.Reset()
	errb.Reset()
	if rc := gateCheckRun(gateCheckConfig{GatesPath: gp, StatusPath: sp2, Check: true}, &out, &errb); rc == 0 {
		t.Fatalf("upper-unverified: want non-zero exit, got 0; stderr=%s", errb.String())
	}
	if !strings.Contains(errb.String(), "gms_v95") {
		t.Errorf("stderr should name the missing side gms_v95, got: %s", errb.String())
	}
}

// A partial-by-design gate (one side legitimately unpinned) passes with a
// reason; the same shape with no reason is a config error and fails.
func TestGateCheckPartial(t *testing.T) {
	dir := t.TempDir()
	sp := writeGateStatus(t, dir, "verified", "incomplete")

	partialOK := `gates:
  - packet: character/clientbound/CharacterSpawn
    direction: clientbound
    boundary: ">87"
    lower_version_key: gms_v87
    upper_version_key: gms_v95
    expect: partial
    reason: v95 fixture is a documented coverage gap
`
	var out, errb bytes.Buffer
	if rc := gateCheckRun(gateCheckConfig{GatesPath: writeGates(t, dir, partialOK), StatusPath: sp, Check: true}, &out, &errb); rc != 0 {
		t.Fatalf("partial-with-reason: want exit 0, got %d; stderr=%s", rc, errb.String())
	}

	partialNoReason := `gates:
  - packet: character/clientbound/CharacterSpawn
    direction: clientbound
    boundary: ">87"
    lower_version_key: gms_v87
    upper_version_key: gms_v95
    expect: partial
`
	d2 := t.TempDir()
	out.Reset()
	errb.Reset()
	if rc := gateCheckRun(gateCheckConfig{GatesPath: writeGates(t, d2, partialNoReason), StatusPath: sp, Check: true}, &out, &errb); rc == 0 {
		t.Fatalf("partial-without-reason: want non-zero exit, got 0")
	}
	if !strings.Contains(errb.String(), "reason") {
		t.Errorf("stderr should flag the missing reason, got: %s", errb.String())
	}
}

// A gate naming a packet absent from the matrix is a config error (fails under
// --check) — guards against typos in gates.yaml silently passing.
func TestGateCheckUnknownPacket(t *testing.T) {
	dir := t.TempDir()
	sp := writeGateStatus(t, dir, "verified", "verified")
	gates := `gates:
  - packet: bogus/clientbound/DoesNotExist
    direction: clientbound
    boundary: ">87"
    lower_version_key: gms_v87
    upper_version_key: gms_v95
`
	var out, errb bytes.Buffer
	if rc := gateCheckRun(gateCheckConfig{GatesPath: writeGates(t, dir, gates), StatusPath: sp, Check: true}, &out, &errb); rc == 0 {
		t.Fatalf("unknown packet: want non-zero exit, got 0")
	}
	if !strings.Contains(errb.String(), "no matrix row") {
		t.Errorf("stderr should flag the unknown packet, got: %s", errb.String())
	}
}

// The committed docs/packets/gates.yaml must be green against the real
// status.json — every seeded gate is both-sides-verified (guards the CI gate we
// wired blocking). Run from the repo root.
func TestGateCheckRealTreePasses(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	gp := filepath.Join(repoRoot, "docs", "packets", "gates.yaml")
	sp := filepath.Join(repoRoot, "docs", "packets", "audits", "status.json")
	if _, err := os.Stat(gp); err != nil {
		t.Skipf("gates.yaml not found: %v", err)
	}
	var out, errb bytes.Buffer
	if rc := gateCheckRun(gateCheckConfig{GatesPath: gp, StatusPath: sp, Check: true}, &out, &errb); rc != 0 {
		t.Fatalf("real-tree gate-check must be green, got exit %d; stderr:\n%s", rc, errb.String())
	}
}
