package matrix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
)

// LoadedReport is the subset of report.Packet the matrix consumes, read back
// from the committed per-packet JSON files.
type LoadedReport struct {
	WriterName  string
	IDAName     string
	Address     string
	AtlasFile   string
	Verdict     diff.Verdict
	FlatInvalid bool
}

// LoadReports reads every per-packet JSON in an audit dir, skipping the
// non-report artifacts (SUMMARY/_pending/_unimplemented and any _-prefixed file).
// A missing dir is not an error: gms_v84 has no audit reports yet (design §3);
// its cells grade incomplete from absence.
func LoadReports(auditDir string) (map[string]LoadedReport, error) {
	out := map[string]LoadedReport{}
	entries, err := os.ReadDir(auditDir)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, "_") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(auditDir, name))
		if err != nil {
			return nil, err
		}
		var r LoadedReport
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, fmt.Errorf("%s/%s: %w", auditDir, name, err)
		}
		if r.WriterName == "" {
			continue // not a report-shaped JSON
		}
		out[r.WriterName] = r
	}
	return out, nil
}

// PacketID derives the canonical packet identity "pkgdir/Struct" from a
// report: AtlasFile (normalized to libs/atlas-packet-relative — older
// committed reports carry a ../../ prefix, newer ones are repo-relative
// per PR #729) plus WriterName.
func PacketID(r LoadedReport) string {
	f := r.AtlasFile
	if i := strings.Index(f, "libs/atlas-packet/"); i >= 0 {
		f = f[i+len("libs/atlas-packet/"):]
	}
	return filepath.ToSlash(filepath.Dir(f)) + "/" + r.WriterName
}
