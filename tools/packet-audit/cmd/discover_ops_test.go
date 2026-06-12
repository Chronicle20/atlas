package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
	"gopkg.in/yaml.v3"
)

// discoverFakeMCP is a fake MCPClient for discover-ops tests.
// GetFunctionByName returns a fixed address for the dispatcher name.
// DecompileFunction returns the Task 5.1 fixture text.
// GetCallees returns a callee list that maps sub_5E1230 -> "CMob::OnEnterField".
type discoverFakeMCP struct {
	dispatcherName string
	dispatcherAddr string
	decompText     string
	callees        []idasrc.Callee
}

func (f *discoverFakeMCP) GetFunctionByName(_ context.Context, name string) (string, bool, error) {
	if name == f.dispatcherName {
		return f.dispatcherAddr, true, nil
	}
	return "", false, nil
}

func (f *discoverFakeMCP) DecompileFunction(_ context.Context, addr string) (string, error) {
	if addr == f.dispatcherAddr {
		return f.decompText, nil
	}
	return "", nil
}

func (f *discoverFakeMCP) GetCallees(_ context.Context, addr string) ([]idasrc.Callee, error) {
	if addr == f.dispatcherAddr {
		return f.callees, nil
	}
	return nil, nil
}

func (f *discoverFakeMCP) StructInfo(_ context.Context, _ string) (idasrc.StructLayout, error) {
	return idasrc.StructLayout{}, nil
}

// readFixture reads the Task 5.1 fixture text for injection into the fake MCP.
func readFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(
		"..", "internal", "discover", "testdata", "process_packet_v83.c.txt",
	))
	if err != nil {
		t.Fatalf("fixture not found: %v", err)
	}
	return string(b)
}

// writeSeedRegistry writes a minimal seeded registry YAML to dir/<version>.yaml.
// entries is a list of pre-built entries. Returns the file path.
func writeSeedRegistry(t *testing.T, dir, version string, entries []opregistry.Entry) string {
	t.Helper()
	raw, err := yaml.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	p := filepath.Join(dir, version+".yaml")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	return p
}

// TestDiscoverOpsWorklist tests --apply=false: verifies the emitted markdown
// contains ## Append with a new op, ## Review with collision and missing items.
func TestDiscoverOpsWorklist(t *testing.T) {
	fixture := readFixture(t)
	fc := &discoverFakeMCP{
		dispatcherName: "CClientSocket::ProcessPacket",
		dispatcherAddr: "0x5e0000",
		decompText:     fixture,
		callees: []idasrc.Callee{
			{Name: "CMob::OnEnterField", Addr: "0x5e1230"},
		},
	}

	dir := t.TempDir()
	// Registry: LOGIN_STATUS (match at 0xC8), GHOST_OP (missing at 0xFF),
	// WORLD_WRONG (collision at 0x22 with different fname).
	seedEntries := []opregistry.Entry{
		{Op: "LOGIN_STATUS", Direction: opregistry.DirClientbound, Opcode: 0xC8, FName: "CLogin::OnCheckPasswordResult", Provenance: "csv-import"},
		{Op: "GHOST_OP", Direction: opregistry.DirClientbound, Opcode: 0xFF, FName: "CFoo::OnGhost", Provenance: "csv-import"},
		{Op: "WORLD_WRONG", Direction: opregistry.DirClientbound, Opcode: 0x22, FName: "CNotRight::OnWrong", Provenance: "csv-import"},
	}
	writeSeedRegistry(t, dir, "gms_v83", seedEntries)

	outMD := filepath.Join(dir, "worklist.md")
	opts := discoverOpsOpts{
		Version:     "gms_v83",
		RegistryDir: dir,
		Dispatcher:  "CClientSocket::ProcessPacket",
		Out:         outMD,
		Apply:       false,
	}
	var stderr strings.Builder
	code := discoverOpsRun(opts, fc, &stderr)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}

	b, err := os.ReadFile(outMD)
	if err != nil {
		t.Fatalf("read worklist: %v", err)
	}
	s := string(b)

	// ## Append section must contain new ops (0x11, 0x12, 0x13, 0x20, 0x21).
	if !strings.Contains(s, "## Append") {
		t.Error("worklist missing ## Append section")
	}
	// IDA_0X011 should be in the Append table (0x11 not in seed).
	if !strings.Contains(s, "IDA_0X011") {
		t.Errorf("IDA_0X011 not in Append:\n%s", s)
	}
	// sub_5E1230 should be resolved to CMob::OnEnterField via GetCallees.
	if !strings.Contains(s, "CMob::OnEnterField") {
		t.Errorf("sub_5E1230 not resolved to CMob::OnEnterField in worklist:\n%s", s)
	}

	// ## Review section must contain the collision and the missing op.
	if !strings.Contains(s, "## Review") {
		t.Error("worklist missing ## Review section")
	}
	if !strings.Contains(s, "WORLD_WRONG") {
		t.Errorf("collision WORLD_WRONG not in ## Review:\n%s", s)
	}
	if !strings.Contains(s, "GHOST_OP") {
		t.Errorf("missing GHOST_OP not in ## Review:\n%s", s)
	}
}

// TestDiscoverOpsApply tests --apply=true without collisions: registry gains
// appended entries and LoadVersion still validates.
func TestDiscoverOpsApply(t *testing.T) {
	fixture := readFixture(t)
	// CLogin::OnFoo (0x11) appears in the fixture; the fake callee map gives it
	// an explicit address so we can assert the IDA.Address is set correctly.
	const onFooAddr = "0x5e1100"
	fc := &discoverFakeMCP{
		dispatcherName: "CClientSocket::ProcessPacket",
		dispatcherAddr: "0x5e0000",
		decompText:     fixture,
		callees: []idasrc.Callee{
			{Name: "CLogin::OnFoo", Addr: onFooAddr},
		},
	}

	dir := t.TempDir()
	// Registry: only LOGIN_STATUS (match). No collision, no missing.
	seedEntries := []opregistry.Entry{
		{Op: "LOGIN_STATUS", Direction: opregistry.DirClientbound, Opcode: 0xC8, FName: "CLogin::OnCheckPasswordResult", Provenance: "csv-import"},
	}
	regPath := writeSeedRegistry(t, dir, "gms_v83", seedEntries)

	outMD := filepath.Join(dir, "worklist.md")
	opts := discoverOpsOpts{
		Version:     "gms_v83",
		RegistryDir: dir,
		Dispatcher:  "CClientSocket::ProcessPacket",
		Out:         outMD,
		Apply:       true,
	}
	var stderr strings.Builder
	code := discoverOpsRun(opts, fc, &stderr)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}

	// Registry must now have more entries and still validate.
	vf, err := opregistry.LoadVersion(regPath)
	if err != nil {
		t.Fatalf("LoadVersion after apply: %v", err)
	}
	if len(vf.Entries) <= len(seedEntries) {
		t.Errorf("no entries appended: got %d, want > %d", len(vf.Entries), len(seedEntries))
	}
	// The new entry for 0x11 must be present with correct provenance and IDA address.
	found := false
	for _, e := range vf.Entries {
		if e.Opcode == 0x11 && e.Direction == opregistry.DirClientbound {
			found = true
			if e.Provenance != "ida-discovered" {
				t.Errorf("new entry provenance = %q, want ida-discovered", e.Provenance)
			}
			if e.IDA == nil {
				t.Error("new entry missing IDA ref")
			} else {
				// IDA.Address must be the callee's address from the fake client,
				// not the dispatcher fallback address.
				wantAddr, _ := strconv.ParseUint(strings.TrimPrefix(onFooAddr, "0x"), 16, 64)
				if e.IDA.Address != wantAddr {
					t.Errorf("new entry IDA.Address = %d (0x%x), want %d (0x%x) from callee map",
						e.IDA.Address, e.IDA.Address, wantAddr, wantAddr)
				}
			}
		}
	}
	if !found {
		t.Error("opcode 0x11 not found in registry after --apply")
	}
}

// TestDiscoverOpsApplyRefusesCollision tests that --apply=true refuses when
// there are collisions: exit code 1 and registry file is unchanged.
func TestDiscoverOpsApplyRefusesCollision(t *testing.T) {
	fixture := readFixture(t)
	fc := &discoverFakeMCP{
		dispatcherName: "CClientSocket::ProcessPacket",
		dispatcherAddr: "0x5e0000",
		decompText:     fixture,
	}

	dir := t.TempDir()
	// Plant a collision: 0x22 exists in fixture as CLogin::OnWorldInformation
	// but registry says it's a different function.
	seedEntries := []opregistry.Entry{
		{Op: "WORLD_WRONG", Direction: opregistry.DirClientbound, Opcode: 0x22, FName: "CNotRight::OnWrong", Provenance: "csv-import"},
	}
	regPath := writeSeedRegistry(t, dir, "gms_v83", seedEntries)
	beforeBytes, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatalf("read registry before run: %v", err)
	}

	outMD := filepath.Join(dir, "worklist.md")
	opts := discoverOpsOpts{
		Version:     "gms_v83",
		RegistryDir: dir,
		Dispatcher:  "CClientSocket::ProcessPacket",
		Out:         outMD,
		Apply:       true,
	}
	var stderr strings.Builder
	code := discoverOpsRun(opts, fc, &stderr)
	if code != 1 {
		t.Errorf("expected exit 1 (collision blocker), got %d; stderr: %s", code, stderr.String())
	}
	afterBytes, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatalf("read registry after run: %v", err)
	}
	// File contents must be identical — bytes.Equal is immune to sub-second
	// mtime resolution issues that plagued the previous ModTime comparison.
	if !bytes.Equal(beforeBytes, afterBytes) {
		t.Error("registry file was modified despite collision blocker")
	}
}
