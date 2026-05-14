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
	fmt.Fprintf(&b, "- **Verdict:** %s\n\n", p.Verdict.Symbol())
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
