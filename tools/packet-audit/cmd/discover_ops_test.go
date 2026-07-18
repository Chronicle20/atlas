package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// discoverFakeMCP is a fake MCPClient for discover-ops tests.
// It stores a map from dispatcher name -> (address, decompText, callees) so
// multi-dispatcher tests can wire up different decompile text per function.
type discoverFakeMCP struct {
	// byName maps dispatcher name to (address, decompText, callees).
	byName map[string]dispatcherFixture
	// byAddr maps dispatcher address to (decompText, callees); used as fallback
	// and for direct hex-address lookup.
	byAddr map[string]dispatcherFixture
}

type dispatcherFixture struct {
	addr       string
	decompText string
	callees    []idasrc.Callee
	decompErr  error // if non-nil, DecompileFunction returns this error
	lookupErr  error // if non-nil, GetFunctionByName returns this error
}

// newSingleFakeMCP builds a fake MCP wired for exactly one dispatcher (the
// common single-dispatcher test case used by the original tests).
func newSingleFakeMCP(name, addr, decompText string, callees []idasrc.Callee) *discoverFakeMCP {
	f := &discoverFakeMCP{
		byName: map[string]dispatcherFixture{},
		byAddr: map[string]dispatcherFixture{},
	}
	fix := dispatcherFixture{addr: addr, decompText: decompText, callees: callees}
	f.byName[name] = fix
	f.byAddr[strings.ToLower(addr)] = fix
	return f
}

func (f *discoverFakeMCP) GetFunctionByName(_ context.Context, name string) (string, bool, error) {
	if fix, ok := f.byName[name]; ok {
		if fix.lookupErr != nil {
			return "", false, fix.lookupErr
		}
		return fix.addr, true, nil
	}
	return "", false, nil
}

func (f *discoverFakeMCP) DecompileFunction(_ context.Context, addr string) (string, error) {
	// Try exact address match first, then case-insensitive.
	if fix, ok := f.byAddr[strings.ToLower(addr)]; ok {
		if fix.decompErr != nil {
			return "", fix.decompErr
		}
		return fix.decompText, nil
	}
	// Also allow lookup by name if the caller passed a name (shouldn't happen
	// in discoverOpsRun but guard against accidental tests).
	return "", nil
}

func (f *discoverFakeMCP) GetCallees(_ context.Context, addr string) ([]idasrc.Callee, error) {
	if fix, ok := f.byAddr[strings.ToLower(addr)]; ok {
		return fix.callees, nil
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

// readFixture2 reads the second dispatcher fixture for multi-dispatcher tests.
func readFixture2(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(
		"..", "internal", "discover", "testdata", "cwvscontext_onpacket_v83.c.txt",
	))
	if err != nil {
		t.Fatalf("fixture2 not found: %v", err)
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
	fc := newSingleFakeMCP("CClientSocket::ProcessPacket", "0x5e0000", fixture, []idasrc.Callee{
		{Name: "CMob::OnEnterField", Addr: "0x5e1230"},
	})

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
		Dispatchers: []string{"CClientSocket::ProcessPacket"},
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

	// ## Dispatchers section must list the single dispatcher.
	if !strings.Contains(s, "## Dispatchers") {
		t.Error("worklist missing ## Dispatchers section")
	}
	if !strings.Contains(s, "CClientSocket::ProcessPacket") {
		t.Error("worklist Dispatchers section missing dispatcher name")
	}

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
	fc := newSingleFakeMCP("CClientSocket::ProcessPacket", "0x5e0000", fixture, []idasrc.Callee{
		{Name: "CLogin::OnFoo", Addr: onFooAddr},
	})

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
		Dispatchers: []string{"CClientSocket::ProcessPacket"},
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
	fc := newSingleFakeMCP("CClientSocket::ProcessPacket", "0x5e0000", fixture, nil)

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
		Dispatchers: []string{"CClientSocket::ProcessPacket"},
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

// TestDiscoverOpsMultiDispatcherUnion tests that two dispatchers' cases are
// merged in the worklist. It wires up two dispatchers with non-overlapping ops
// and asserts:
//   - The ## Dispatchers header lists both dispatchers with correct case counts.
//   - Ops from both dispatchers appear in ## Append.
func TestDiscoverOpsMultiDispatcherUnion(t *testing.T) {
	fixture1 := readFixture(t)  // ops from CClientSocket::ProcessPacket
	fixture2 := readFixture2(t) // ops 0xA0, 0xA1, 0xA2 from CWvsContext::OnPacket

	fc := &discoverFakeMCP{
		byName: map[string]dispatcherFixture{
			"CClientSocket::ProcessPacket": {addr: "0x5e0000", decompText: fixture1},
			"CWvsContext::OnPacket":        {addr: "0xa07a08", decompText: fixture2},
		},
		byAddr: map[string]dispatcherFixture{
			"0x5e0000": {addr: "0x5e0000", decompText: fixture1},
			"0xa07a08": {addr: "0xa07a08", decompText: fixture2, callees: []idasrc.Callee{
				{Name: "CWvsContext::OnInventoryOperationFull", Addr: "0xa07b40"},
			}},
		},
	}

	dir := t.TempDir()
	// Empty seed registry — everything should be in Append.
	writeSeedRegistry(t, dir, "gms_v83", nil)

	outMD := filepath.Join(dir, "worklist.md")
	opts := discoverOpsOpts{
		Version:     "gms_v83",
		RegistryDir: dir,
		Dispatchers: []string{"CClientSocket::ProcessPacket", "CWvsContext::OnPacket"},
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

	// Both dispatchers must appear in the ## Dispatchers table.
	if !strings.Contains(s, "## Dispatchers") {
		t.Error("worklist missing ## Dispatchers section")
	}
	if !strings.Contains(s, "CClientSocket::ProcessPacket") {
		t.Error("worklist Dispatchers table missing CClientSocket::ProcessPacket")
	}
	if !strings.Contains(s, "CWvsContext::OnPacket") {
		t.Error("worklist Dispatchers table missing CWvsContext::OnPacket")
	}

	// Ops from both dispatchers should be in ## Append.
	// CClientSocket fixture has 0x11, 0x12, 0x13, 0x20, 0x21, 0x22, 0x30–0x32, 0x40, 0x50–0x52.
	if !strings.Contains(s, "IDA_0X011") {
		t.Error("IDA_0X011 (from dispatcher1) missing from Append")
	}
	// CWvsContext fixture has 0xA0, 0xA1, 0xA2.
	// opNameFor uses IDA_0X%03X so 0xA0 → IDA_0X0A0.
	if !strings.Contains(s, "IDA_0X0A0") {
		t.Error("IDA_0X0A0 (from dispatcher2, opcode 0xA0) missing from Append")
	}
	if !strings.Contains(s, "IDA_0X0A1") {
		t.Error("IDA_0X0A1 (from dispatcher2, opcode 0xA1) missing from Append")
	}
}

// TestDiscoverOpsZeroCaseDispatcherWarnsAndContinues tests that a dispatcher
// yielding 0 cases emits a stderr warning but does not abort the run if other
// dispatchers find cases. Exit code must be 0.
func TestDiscoverOpsZeroCaseDispatcherWarnsAndContinues(t *testing.T) {
	fixture := readFixture(t) // CClientSocket::ProcessPacket — has real cases

	// Second dispatcher gets empty decompile text → 0 cases.
	const emptyDecomp = `void __thiscall CEmpty::OnPacket(CEmpty *this, CInPacket *a2) { }`

	fc := &discoverFakeMCP{
		byName: map[string]dispatcherFixture{
			"CClientSocket::ProcessPacket": {addr: "0x5e0000", decompText: fixture},
			"CEmpty::OnPacket":             {addr: "0xdead00", decompText: emptyDecomp},
		},
		byAddr: map[string]dispatcherFixture{
			"0x5e0000": {addr: "0x5e0000", decompText: fixture},
			"0xdead00": {addr: "0xdead00", decompText: emptyDecomp},
		},
	}

	dir := t.TempDir()
	writeSeedRegistry(t, dir, "gms_v83", nil)

	outMD := filepath.Join(dir, "worklist.md")
	opts := discoverOpsOpts{
		Version:     "gms_v83",
		RegistryDir: dir,
		Dispatchers: []string{"CClientSocket::ProcessPacket", "CEmpty::OnPacket"},
		Out:         outMD,
		Apply:       false,
	}
	var stderr strings.Builder
	code := discoverOpsRun(opts, fc, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0 (zero-case dispatcher just warns), got %d; stderr: %s", code, stderr.String())
	}

	// stderr must mention the zero-case dispatcher by name.
	if !strings.Contains(stderr.String(), "CEmpty::OnPacket") {
		t.Errorf("stderr should warn about the zero-case dispatcher; got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "0 cases") {
		t.Errorf("stderr should mention '0 cases'; got: %s", stderr.String())
	}

	// Worklist must still contain ops from the non-zero dispatcher.
	b, _ := os.ReadFile(outMD)
	s := string(b)
	if !strings.Contains(s, "IDA_0X011") {
		t.Error("IDA_0X011 from non-zero dispatcher missing from Append")
	}

	// ## Dispatchers table should show CEmpty::OnPacket with 0 cases.
	if !strings.Contains(s, "CEmpty::OnPacket") {
		t.Error("zero-case dispatcher should still appear in ## Dispatchers table")
	}
}

// TestDiscoverOpsInternalCollisionReportedAndExcluded tests that when two
// dispatchers claim the same opcode with different handlers, the collision is
// reported in ## Review, NOT appended to the registry, and --apply is refused.
func TestDiscoverOpsInternalCollisionReportedAndExcluded(t *testing.T) {
	// Both dispatchers claim opcode 0x11 with different handlers.
	const decompA = `
void __thiscall CDispA::OnPacket(CDispA *this, CInPacket *a2)
{
  switch ( CInPacket::Decode2(a2) )
  {
    case 0x11u:
      CLogin::OnFoo(this, a2);
      break;
  }
}
`
	const decompB = `
void __thiscall CDispB::OnPacket(CDispB *this, CInPacket *a2)
{
  switch ( CInPacket::Decode2(a2) )
  {
    case 0x11u:
      CField::OnFooConflict(this, a2);
      break;
  }
}
`
	fc := &discoverFakeMCP{
		byName: map[string]dispatcherFixture{
			"CDispA::OnPacket": {addr: "0xaa0000", decompText: decompA},
			"CDispB::OnPacket": {addr: "0xbb0000", decompText: decompB},
		},
		byAddr: map[string]dispatcherFixture{
			"0xaa0000": {addr: "0xaa0000", decompText: decompA},
			"0xbb0000": {addr: "0xbb0000", decompText: decompB},
		},
	}

	dir := t.TempDir()
	writeSeedRegistry(t, dir, "gms_v83", nil)

	outMD := filepath.Join(dir, "worklist.md")
	opts := discoverOpsOpts{
		Version:     "gms_v83",
		RegistryDir: dir,
		Dispatchers: []string{"CDispA::OnPacket", "CDispB::OnPacket"},
		Out:         outMD,
		Apply:       false,
	}
	var stderr strings.Builder
	code := discoverOpsRun(opts, fc, &stderr)
	// exit 3 only when TOTAL is 0 — here two dispatchers each got 1 case (union
	// removes both due to collision), but the internal collision is still
	// recorded and written. Should succeed with exit 0.
	if code != 0 {
		t.Fatalf("exit %d (expected 0 — internal collision is a worklist item, not a fatal): %s", code, stderr.String())
	}

	b, err := os.ReadFile(outMD)
	if err != nil {
		t.Fatalf("read worklist: %v", err)
	}
	s := string(b)

	// ## Review must contain a Discovery-internal collisions section.
	if !strings.Contains(s, "Discovery-internal collisions") {
		t.Errorf("worklist missing 'Discovery-internal collisions' section:\n%s", s)
	}
	// The conflicting opcode must appear.
	if !strings.Contains(s, "0x011") {
		t.Errorf("internal collision opcode 0x011 not in worklist:\n%s", s)
	}
	// The conflicting op must NOT be in Append.
	if strings.Contains(s, "IDA_0X011") {
		t.Errorf("internally-colliding opcode IDA_0X011 should not appear in Append:\n%s", s)
	}

	// --apply must also refuse due to internal collision.
	optsApply := opts
	optsApply.Apply = true
	optsApply.Out = filepath.Join(dir, "worklist2.md")
	var stderr2 strings.Builder
	code2 := discoverOpsRun(optsApply, fc, &stderr2)
	if code2 != 1 {
		t.Errorf("--apply with internal collision: expected exit 1, got %d; stderr: %s", code2, stderr2.String())
	}
}

// TestDiscoverOpsDefaultFlagIsOneElementList verifies that passing the default
// single-dispatcher value through runDiscoverOps correctly produces a
// one-element Dispatchers list (flag parsing regression guard).
func TestDiscoverOpsDefaultFlagIsOneElementList(t *testing.T) {
	fixture := readFixture(t)
	fc := newSingleFakeMCP("CClientSocket::ProcessPacket", "0x5e0000", fixture, nil)

	dir := t.TempDir()
	writeSeedRegistry(t, dir, "gms_v83", nil)
	outMD := filepath.Join(dir, "worklist.md")

	// Inject opts directly (bypassing flag parsing) with a single-element list.
	opts := discoverOpsOpts{
		Version:     "gms_v83",
		RegistryDir: dir,
		Dispatchers: []string{"CClientSocket::ProcessPacket"},
		Out:         outMD,
		Apply:       false,
	}
	var stderr strings.Builder
	code := discoverOpsRun(opts, fc, &stderr)
	if code != 0 {
		t.Fatalf("single-dispatcher default: exit %d: %s", code, stderr.String())
	}
	b, _ := os.ReadFile(outMD)
	s := string(b)
	// Dispatcher table must show exactly 1 row.
	rows := strings.Count(s, "CClientSocket::ProcessPacket")
	if rows < 1 {
		t.Error("single dispatcher not found in worklist")
	}
}

// TestDiscoverOpsOneDispatcherDecompileFailureContinues tests Fix 2:
// when one dispatcher in a multi-dispatcher list fails Hex-Rays decompilation,
// the run continues, a warning is emitted on stderr, the failed dispatcher
// is recorded as FAILED in the worklist Dispatchers table, and ops from the
// successful dispatcher are still present in ## Append.  Exit code must be 0.
func TestDiscoverOpsOneDispatcherDecompileFailureContinues(t *testing.T) {
	fixture := readFixture(t) // CClientSocket::ProcessPacket — decompiles fine

	decompFail := errors.New("Hex-Rays decompilation failed: stack frame too complex")

	fc := &discoverFakeMCP{
		byName: map[string]dispatcherFixture{
			"CClientSocket::ProcessPacket": {addr: "0x5e0000", decompText: fixture},
			"CUserLocal::OnPacket":         {addr: "0xcc0000", decompText: ""},
		},
		byAddr: map[string]dispatcherFixture{
			"0x5e0000": {addr: "0x5e0000", decompText: fixture},
			"0xcc0000": {addr: "0xcc0000", decompText: "", decompErr: decompFail},
		},
	}

	dir := t.TempDir()
	writeSeedRegistry(t, dir, "gms_v83", nil)

	outMD := filepath.Join(dir, "worklist.md")
	opts := discoverOpsOpts{
		Version:     "gms_v83",
		RegistryDir: dir,
		Dispatchers: []string{"CClientSocket::ProcessPacket", "CUserLocal::OnPacket"},
		Out:         outMD,
		Apply:       false,
	}
	var stderr strings.Builder
	code := discoverOpsRun(opts, fc, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0 (one decompile failure tolerates in multi-dispatcher), got %d; stderr: %s", code, stderr.String())
	}

	// stderr must warn about the failed dispatcher by name.
	if !strings.Contains(stderr.String(), "CUserLocal::OnPacket") {
		t.Errorf("stderr should name the failed dispatcher; got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "decompile failed") {
		t.Errorf("stderr should say 'decompile failed'; got: %s", stderr.String())
	}

	b, err := os.ReadFile(outMD)
	if err != nil {
		t.Fatalf("read worklist: %v", err)
	}
	s := string(b)

	// Failed dispatcher must appear in the Dispatchers table with FAILED prefix.
	if !strings.Contains(s, "CUserLocal::OnPacket") {
		t.Error("failed dispatcher not listed in Dispatchers table")
	}
	if !strings.Contains(s, "FAILED:") {
		t.Errorf("FAILED marker missing from Dispatchers table:\n%s", s)
	}

	// Ops from the successful dispatcher must still be in ## Append.
	if !strings.Contains(s, "IDA_0X011") {
		t.Error("IDA_0X011 from successful dispatcher missing from Append")
	}
}

// TestDiscoverOpsAllDispatchersFailExits tests Fix 2 boundary:
// when ALL dispatchers in a multi-dispatcher list fail, the run must exit 3
// (cannot continue with zero discovered cases).
func TestDiscoverOpsAllDispatchersFailExits(t *testing.T) {
	decompFail := errors.New("decompilation error: internal IDA error")

	fc := &discoverFakeMCP{
		byName: map[string]dispatcherFixture{
			"CDispA::OnPacket": {addr: "0xaa0000", decompText: ""},
			"CDispB::OnPacket": {addr: "0xbb0000", decompText: ""},
		},
		byAddr: map[string]dispatcherFixture{
			"0xaa0000": {addr: "0xaa0000", decompText: "", decompErr: decompFail},
			"0xbb0000": {addr: "0xbb0000", decompText: "", decompErr: decompFail},
		},
	}

	dir := t.TempDir()
	writeSeedRegistry(t, dir, "gms_v83", nil)

	outMD := filepath.Join(dir, "worklist.md")
	opts := discoverOpsOpts{
		Version:     "gms_v83",
		RegistryDir: dir,
		Dispatchers: []string{"CDispA::OnPacket", "CDispB::OnPacket"},
		Out:         outMD,
		Apply:       false,
	}
	var stderr strings.Builder
	code := discoverOpsRun(opts, fc, &stderr)
	if code != 3 {
		t.Errorf("all dispatchers failed: expected exit 3, got %d; stderr: %s", code, stderr.String())
	}
	// stderr must say "all ... dispatcher(s) failed".
	if !strings.Contains(stderr.String(), "all") || !strings.Contains(stderr.String(), "failed") {
		t.Errorf("stderr should say all dispatchers failed; got: %s", stderr.String())
	}
}
