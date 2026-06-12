package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
)

type Packet struct {
	WriterName  string
	IDAName     string
	Address     string
	Variant     string
	BranchDepth int
	AtlasFile   string
	Rows        []diff.Row
	Verdict     diff.Verdict
	// FlatInvalid marks a packet whose Atlas writer branches on a condition the
	// analyzer could not reduce to a version predicate — a data-dependent field
	// or a version-derived local the flatten doesn't trace — so a flat positional
	// diff cannot faithfully compare it (the analyzer merges/picks one branch; the
	// client reads the runtime branch). Such a packet's verdict is capped to
	// Deferred (🔍): the row-level divergence is a modeling limitation, NOT a
	// verified wire bug.
	FlatInvalid bool
}

func WritePacket(outDir string, p Packet) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(
		filepath.Join(outDir, p.WriterName+".md"),
		[]byte(renderMarkdown(p)),
		0o644,
	); err != nil {
		return err
	}
	js, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, p.WriterName+".json"), js, 0o644)
}

func renderMarkdown(p Packet) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s (← `%s`)\n\n", p.WriterName, p.IDAName)
	fmt.Fprintf(&b, "- **IDA:** %s\n", p.Address)
	fmt.Fprintf(&b, "- **Atlas file:** `%s`\n", p.AtlasFile)
	fmt.Fprintf(&b, "- **Variant:** %s\n", p.Variant)
	fmt.Fprintf(&b, "- **Branch depth:** %d\n", p.BranchDepth)
	fmt.Fprintf(&b, "- **Verdict:** %s\n", p.Verdict.Symbol())
	if p.FlatInvalid {
		b.WriteString("- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.\n")
	}
	b.WriteString("\n")
	b.WriteString("## Wire-level diff\n\n")
	b.WriteString("| # | Atlas writes | v? reads | Verdict | Note |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, r := range p.Rows {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n",
			r.Index, atlasCol(r), idaCol(r), r.Verdict.Symbol(), escapeMD(r.Note))
	}
	b.WriteString("\n")
	return b.String()
}

func atlasCol(r diff.Row) string {
	return r.AtlasOp.String()
}

func idaCol(r diff.Row) string {
	return fmt.Sprintf("%s `%s`", r.IDAOp.String(), escapeMD(r.IDAComment))
}

func escapeMD(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
