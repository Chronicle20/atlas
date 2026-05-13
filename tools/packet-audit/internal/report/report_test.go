package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/atlaspacket"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

func TestWritePacketMarkdownAndJSON(t *testing.T) {
	out := t.TempDir()
	pkt := Packet{
		WriterName:  "AuthSuccess",
		IDAName:     "CLogin::OnCheckPasswordResult",
		Address:     "0x5dc600",
		Variant:     "GMS/v95/modified",
		BranchDepth: 2,
		AtlasFile:   "libs/atlas-packet/login/clientbound/auth_success.go",
		Rows: []diff.Row{
			{Index: 0, AtlasOp: atlaspacket.Encode1, IDAOp: idasrc.Decode1, Verdict: diff.VerdictMatch},
			{Index: 1, AtlasOp: atlaspacket.Encode1, IDAOp: idasrc.Decode2, Verdict: diff.VerdictBlocker, Note: "width mismatch"},
		},
		Verdict: diff.VerdictBlocker,
	}
	if err := WritePacket(out, pkt); err != nil {
		t.Fatal(err)
	}
	md, err := os.ReadFile(filepath.Join(out, "AuthSuccess.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "❌") {
		t.Errorf("md missing blocker symbol: %s", md)
	}
	if _, err := os.Stat(filepath.Join(out, "AuthSuccess.json")); err != nil {
		t.Errorf("sidecar JSON missing: %v", err)
	}
}
