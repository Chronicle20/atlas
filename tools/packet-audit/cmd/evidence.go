package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/evidence"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"gopkg.in/yaml.v3"
)

func runEvidence(args []string, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "pin" {
		fmt.Fprintln(stderr, "packet-audit evidence: unknown subcommand (expected: pin)")
		return 3
	}
	fs := flag.NewFlagSet("packet-audit evidence pin", flag.ContinueOnError)
	fs.SetOutput(stderr)
	packet := fs.String("packet", "", "packet id, e.g. buddy/clientbound/Invite (required)")
	version := fs.String("version", "", "version key, e.g. gms_v83 (required)")
	ida := fs.String("ida", "", "IDA function name as it appears in the export (required)")
	category := fs.String("category", "", "evidence category (required)")
	export := fs.String("export", "", "export JSON path (default: derived from version)")
	dir := fs.String("evidence-dir", "docs/packets/evidence", "evidence ledger dir")
	if err := fs.Parse(args[1:]); err != nil {
		return 3
	}
	if *packet == "" || *version == "" || *ida == "" || *category == "" {
		fmt.Fprintln(stderr, "packet-audit evidence pin: --packet, --version, --ida, --category are required")
		return 3
	}
	if *export == "" {
		*export = matrix.ExportPath(*version)
	}
	hash, err := evidence.FunctionHash(*export, *ida)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit evidence pin: %v\n", err)
		return 3
	}
	addr, err := functionAddress(*export, *ida)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit evidence pin: %v\n", err)
		return 3
	}
	dirOf := "clientbound"
	if strings.Contains(*packet, "/serverbound/") {
		dirOf = "serverbound"
	}
	rec := evidence.Record{
		Packet: *packet, Direction: dirOf, Version: *version, Category: *category,
		IDA: evidence.IDACitation{Function: *ida, Address: addr, DecompileSHA256: hash},
	}
	raw, err := yaml.Marshal(rec)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit evidence pin: %v\n", err)
		return 3
	}
	p := evidence.RecordPath(*dir, *version, *packet)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		fmt.Fprintf(stderr, "packet-audit evidence pin: %v\n", err)
		return 3
	}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		fmt.Fprintf(stderr, "packet-audit evidence pin: %v\n", err)
		return 3
	}
	fmt.Fprintf(os.Stdout, "pinned %s\n", p)
	return 0
}

func functionAddress(exportPath, fname string) (string, error) {
	raw, err := os.ReadFile(exportPath)
	if err != nil {
		return "", err
	}
	var file struct {
		Functions map[string]struct {
			Address string `json:"address"`
		} `json:"functions"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return "", err
	}
	fn, ok := file.Functions[fname]
	if !ok {
		return "", fmt.Errorf("%s: function %q not in export", exportPath, fname)
	}
	return fn.Address, nil
}
