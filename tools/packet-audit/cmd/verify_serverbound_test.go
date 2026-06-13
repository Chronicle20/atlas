package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
	"gopkg.in/yaml.v3"
)

// verifyFakeMCP is a fake MCPClient for verify-serverbound tests.
// It returns pre-wired decompile text keyed by hex address (lower-case).
type verifyFakeMCP struct {
	byAddr map[string]verifyFixture
}

type verifyFixture struct {
	decompText string
	decompErr  error
}

func (f *verifyFakeMCP) GetFunctionByName(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}

func (f *verifyFakeMCP) DecompileFunction(_ context.Context, addr string) (string, error) {
	if fix, ok := f.byAddr[strings.ToLower(addr)]; ok {
		if fix.decompErr != nil {
			return "", fix.decompErr
		}
		return fix.decompText, nil
	}
	return "", nil
}

func (f *verifyFakeMCP) GetCallees(_ context.Context, _ string) ([]idasrc.Callee, error) {
	return nil, nil
}

func (f *verifyFakeMCP) StructInfo(_ context.Context, _ string) (idasrc.StructLayout, error) {
	return idasrc.StructLayout{}, nil
}

// writeVerifyRegistry writes a minimal serverbound registry YAML to dir/<version>.yaml.
func writeVerifyRegistry(t *testing.T, dir, version string, entries []opregistry.Entry) string {
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

// writeAuditReport writes a LoadedReport-shaped JSON to dir/<name>.json.
func writeAuditReport(t *testing.T, dir string, r matrix.LoadedReport) {
	t.Helper()
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	p := filepath.Join(dir, r.WriterName+".json")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatalf("write audit report: %v", err)
	}
}

// TestVerifyServerbound exercises the main happy-path:
//   - entry 1 (CHECK_PASSWORD / CLogin::SendCheckPasswordPacket):
//     decompile returns opcode 54 = 0x36 → CONFIRMED.
//   - entry 2 (SET_GENDER / CLogin::SendSetGenderPacket):
//     decompile returns opcode 0x20 (not 0x08) → MISMATCH.
//   - entry 3 (VIEW_ALL_CHARS / CLogin::SendViewAllCharPacket):
//     decompile returns variable opcode only → UNRESOLVED.
func TestVerifyServerbound(t *testing.T) {
	// --- registry ---
	regDir := t.TempDir()
	entries := []opregistry.Entry{
		{Op: "CHECK_PASSWORD", Direction: opregistry.DirServerbound, Opcode: 54, FName: "CLogin::SendCheckPasswordPacket", Provenance: "csv-import"},
		{Op: "SET_GENDER", Direction: opregistry.DirServerbound, Opcode: 8, FName: "CLogin::SendSetGenderPacket", Provenance: "csv-import"},
		{Op: "VIEW_ALL_CHARS", Direction: opregistry.DirServerbound, Opcode: 22, FName: "CLogin::SendViewAllCharPacket", Provenance: "csv-import"},
	}
	writeVerifyRegistry(t, regDir, "gms_v83", entries)

	// --- audit reports ---
	auditsDir := t.TempDir()
	versionAuditDir := filepath.Join(auditsDir, "gms_v83")
	if err := os.MkdirAll(versionAuditDir, 0o755); err != nil {
		t.Fatalf("mkdir audit dir: %v", err)
	}

	// Report for entry 1: IDAName -> address 0x5e1100
	writeAuditReport(t, versionAuditDir, matrix.LoadedReport{
		WriterName: "CheckPasswordWriter",
		IDAName:    "CLogin::SendCheckPasswordPacket",
		Address:    "0x5e1100",
		AtlasFile:  "libs/atlas-packet/login/serverbound/check_password.go",
	})
	// Report for entry 2: IDAName -> address 0x5e2200
	writeAuditReport(t, versionAuditDir, matrix.LoadedReport{
		WriterName: "SetGenderWriter",
		IDAName:    "CLogin::SendSetGenderPacket",
		Address:    "0x5e2200",
		AtlasFile:  "libs/atlas-packet/login/serverbound/set_gender.go",
	})
	// Report for entry 3: IDAName -> address 0x5e3300
	writeAuditReport(t, versionAuditDir, matrix.LoadedReport{
		WriterName: "ViewAllCharsWriter",
		IDAName:    "CLogin::SendViewAllCharPacket",
		Address:    "0x5e3300",
		AtlasFile:  "libs/atlas-packet/login/serverbound/view_all_chars.go",
	})

	// --- fake MCP ---
	// entry 1: decompile returns opcode 54 → CONFIRMED
	decompConfirmed := `
void CLogin::SendCheckPasswordPacket(CLogin *this)
{
  COutPacket oPacket;
  COutPacket::COutPacket(&oPacket, 54);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &oPacket);
  COutPacket::~COutPacket(&oPacket);
}
`
	// entry 2: decompile returns opcode 0x20 (32), not 0x08 (8) → MISMATCH
	decompMismatch := `
void CLogin::SendSetGenderPacket(CLogin *this, int nGender)
{
  COutPacket oPacket;
  COutPacket::COutPacket(&oPacket, 0x20u);
  COutPacket::Encode1(&oPacket, nGender);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &oPacket);
  COutPacket::~COutPacket(&oPacket);
}
`
	// entry 3: decompile returns only variable opcode → UNRESOLVED
	decompVariable := `
void CLogin::SendViewAllCharPacket(CLogin *this)
{
  COutPacket oPacket;
  COutPacket::COutPacket(&oPacket, nDynamicOp);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &oPacket);
  COutPacket::~COutPacket(&oPacket);
}
`

	fc := &verifyFakeMCP{
		byAddr: map[string]verifyFixture{
			"0x5e1100": {decompText: decompConfirmed},
			"0x5e2200": {decompText: decompMismatch},
			"0x5e3300": {decompText: decompVariable},
		},
	}

	// --- run ---
	outMD := filepath.Join(regDir, "out.md")
	opts := verifyServerboundOpts{
		Version:     "gms_v83",
		RegistryDir: regDir,
		AuditsDir:   auditsDir,
		Out:         outMD,
	}
	var stderr strings.Builder
	code := verifyServerboundRun(opts, fc, &stderr)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}

	b, err := os.ReadFile(outMD)
	if err != nil {
		t.Fatalf("read worklist: %v", err)
	}
	s := string(b)

	// CHECK_PASSWORD must be in Confirmed.
	if !strings.Contains(s, "## Confirmed") {
		t.Error("missing ## Confirmed section")
	}
	if !strings.Contains(s, "CHECK_PASSWORD") {
		t.Errorf("CHECK_PASSWORD not in Confirmed:\n%s", s)
	}

	// SET_GENDER must be in Mismatch — REVIEW.
	if !strings.Contains(s, "## Mismatch") {
		t.Error("missing ## Mismatch section")
	}
	if !strings.Contains(s, "SET_GENDER") {
		t.Errorf("SET_GENDER not in Mismatch:\n%s", s)
	}
	// The found set (0x020 = 32) must appear in the mismatch row.
	if !strings.Contains(s, "0x020") {
		t.Errorf("found opcode 0x020 not shown in mismatch row:\n%s", s)
	}

	// VIEW_ALL_CHARS must be in Unresolved.
	if !strings.Contains(s, "## Unresolved") {
		t.Error("missing ## Unresolved section")
	}
	if !strings.Contains(s, "VIEW_ALL_CHARS") {
		t.Errorf("VIEW_ALL_CHARS not in Unresolved:\n%s", s)
	}
}

// TestVerifyServerboundMissingVersion tests that a missing --version flag
// returns exit code 3.
func TestVerifyServerboundMissingVersion(t *testing.T) {
	var stderr strings.Builder
	code := runVerifyServerbound([]string{}, &stderr)
	if code != 3 {
		t.Errorf("expected exit 3 for missing --version, got %d", code)
	}
}

// TestVerifyServerboundNoServerboundEntries tests that an empty registry
// (no serverbound entries) returns exit code 3 with an informative message.
func TestVerifyServerboundNoServerboundEntries(t *testing.T) {
	regDir := t.TempDir()
	// Registry with only clientbound entries.
	entries := []opregistry.Entry{
		{Op: "SOME_CB", Direction: opregistry.DirClientbound, Opcode: 1, FName: "CFoo::OnSomething", Provenance: "csv-import"},
	}
	writeVerifyRegistry(t, regDir, "gms_v83", entries)

	auditsDir := t.TempDir()
	outMD := filepath.Join(regDir, "out.md")
	opts := verifyServerboundOpts{
		Version:     "gms_v83",
		RegistryDir: regDir,
		AuditsDir:   auditsDir,
		Out:         outMD,
	}
	fc := &verifyFakeMCP{byAddr: map[string]verifyFixture{}}
	var stderr strings.Builder
	code := verifyServerboundRun(opts, fc, &stderr)
	if code != 3 {
		t.Errorf("expected exit 3 for no serverbound entries, got %d", code)
	}
	if !strings.Contains(stderr.String(), "no serverbound") {
		t.Errorf("stderr should mention 'no serverbound'; got: %s", stderr.String())
	}
}

// TestVerifyServerboundIDANameWithHashStripped tests that an IDAName with a
// '#variant' suffix is stripped correctly when building the fname->address map,
// so that the registry FName still resolves to the right address.
func TestVerifyServerboundIDANameWithHashStripped(t *testing.T) {
	regDir := t.TempDir()
	entries := []opregistry.Entry{
		{Op: "BUDDY_INVITE", Direction: opregistry.DirServerbound, Opcode: 30, FName: "CWvsContext::OnFriendResult", Provenance: "csv-import"},
	}
	writeVerifyRegistry(t, regDir, "gms_v83", entries)

	auditsDir := t.TempDir()
	versionAuditDir := filepath.Join(auditsDir, "gms_v83")
	if err := os.MkdirAll(versionAuditDir, 0o755); err != nil {
		t.Fatalf("mkdir audit dir: %v", err)
	}
	// Report whose IDAName has a '#Invite' variant suffix.
	writeAuditReport(t, versionAuditDir, matrix.LoadedReport{
		WriterName: "BuddyInvite",
		IDAName:    "CWvsContext::OnFriendResult#Invite",
		Address:    "0xa3f2e8",
		AtlasFile:  "libs/atlas-packet/buddy/serverbound/invite.go",
	})

	// Decompile returns opcode 30 → CONFIRMED.
	decompText := `
void CWvsContext::OnFriendResult(CWvsContext *this)
{
  COutPacket oPacket;
  COutPacket::COutPacket(&oPacket, 30);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &oPacket);
  COutPacket::~COutPacket(&oPacket);
}
`
	fc := &verifyFakeMCP{
		byAddr: map[string]verifyFixture{
			"0xa3f2e8": {decompText: decompText},
		},
	}

	outMD := filepath.Join(regDir, "out.md")
	opts := verifyServerboundOpts{
		Version:     "gms_v83",
		RegistryDir: regDir,
		AuditsDir:   auditsDir,
		Out:         outMD,
	}
	var stderr strings.Builder
	code := verifyServerboundRun(opts, fc, &stderr)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	b, _ := os.ReadFile(outMD)
	s := string(b)
	if !strings.Contains(s, "BUDDY_INVITE") || !strings.Contains(s, "## Confirmed") {
		t.Errorf("BUDDY_INVITE not in Confirmed after hash-strip:\n%s", s)
	}
	if !strings.Contains(s, "BUDDY_INVITE") {
		t.Errorf("BUDDY_INVITE missing from worklist:\n%s", s)
	}
	// Must be confirmed, not unresolved.
	confirmedIdx := strings.Index(s, "## Confirmed")
	unresolvedIdx := strings.Index(s, "## Unresolved")
	buddyIdx := strings.Index(s, "BUDDY_INVITE")
	if confirmedIdx < 0 || unresolvedIdx < 0 || buddyIdx < 0 {
		t.Fatalf("section structure unexpected:\n%s", s)
	}
	if buddyIdx > unresolvedIdx {
		t.Errorf("BUDDY_INVITE appears in Unresolved section, expected Confirmed:\n%s", s)
	}
}

// TestVerifyServerboundDecompileError tests that a per-function decompile
// error is non-fatal: the entry is recorded as UNRESOLVED and exit is 0.
func TestVerifyServerboundDecompileError(t *testing.T) {
	regDir := t.TempDir()
	entries := []opregistry.Entry{
		{Op: "CHECK_PASSWORD", Direction: opregistry.DirServerbound, Opcode: 54, FName: "CLogin::SendCheckPasswordPacket", Provenance: "csv-import"},
	}
	writeVerifyRegistry(t, regDir, "gms_v83", entries)

	auditsDir := t.TempDir()
	versionAuditDir := filepath.Join(auditsDir, "gms_v83")
	if err := os.MkdirAll(versionAuditDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeAuditReport(t, versionAuditDir, matrix.LoadedReport{
		WriterName: "CheckPasswordWriter",
		IDAName:    "CLogin::SendCheckPasswordPacket",
		Address:    "0x5e1100",
		AtlasFile:  "libs/atlas-packet/login/serverbound/check_password.go",
	})

	fc := &verifyFakeMCP{
		byAddr: map[string]verifyFixture{
			"0x5e1100": {decompErr: fmt.Errorf("Hex-Rays failed: stack too complex")},
		},
	}

	outMD := filepath.Join(regDir, "out.md")
	opts := verifyServerboundOpts{
		Version:     "gms_v83",
		RegistryDir: regDir,
		AuditsDir:   auditsDir,
		Out:         outMD,
	}
	var stderr strings.Builder
	code := verifyServerboundRun(opts, fc, &stderr)
	if code != 0 {
		t.Fatalf("decompile error must be non-fatal (exit 0), got %d: %s", code, stderr.String())
	}
	b, _ := os.ReadFile(outMD)
	s := string(b)
	if !strings.Contains(s, "CHECK_PASSWORD") {
		t.Errorf("CHECK_PASSWORD missing from worklist:\n%s", s)
	}
	// Must be in Unresolved, not Confirmed.
	confirmedIdx := strings.Index(s, "## Confirmed (0)")
	if confirmedIdx < 0 {
		t.Errorf("expected Confirmed (0), worklist:\n%s", s)
	}
	if !strings.Contains(s, "decompile error") {
		t.Errorf("unresolved reason should mention 'decompile error':\n%s", s)
	}
}
