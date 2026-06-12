package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/discover"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

type verifyServerboundOpts struct {
	Version     string
	RegistryDir string
	AuditsDir   string
	IDAURL      string
	IDAPort     int
	Out         string
}

// runVerifyServerbound is the verify-serverbound subcommand entry point. It
// accepts an injectable MCPClient so tests can supply a fake; when client is
// nil the real HTTP client is constructed from flag values.
func runVerifyServerbound(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit verify-serverbound", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts verifyServerboundOpts
	fs.StringVar(&opts.Version, "version", "", "target version key, e.g. gms_v83 (required)")
	fs.StringVar(&opts.RegistryDir, "registry-dir", "docs/packets/registry", "directory containing <version>.yaml registry files")
	fs.StringVar(&opts.AuditsDir, "audits-dir", "docs/packets/audits", "parent directory containing per-version audit report dirs")
	fs.StringVar(&opts.IDAURL, "ida-url", "http://192.168.20.3:13337/mcp", "IDA-MCP HTTP endpoint")
	fs.IntVar(&opts.IDAPort, "ida-port", 0, "IDA-MCP instance port to select (0 = default active instance)")
	fs.StringVar(&opts.Out, "out", "", "worklist markdown output path (default: docs/packets/registry/verify_serverbound_<version>.md)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "packet-audit verify-serverbound: flag parse error:", err)
		return 3
	}
	if opts.Version == "" {
		fmt.Fprintln(stderr, "packet-audit verify-serverbound: missing required flag --version")
		return 3
	}
	if opts.Out == "" {
		opts.Out = filepath.Join(opts.RegistryDir, "verify_serverbound_"+opts.Version+".md")
	}

	hc := &http.Client{Timeout: 60 * time.Second}
	var client idasrc.MCPClient
	if opts.IDAPort != 0 {
		client = idasrc.NewMCPHTTPClientWithInstance(opts.IDAURL, hc, opts.IDAPort)
	} else {
		client = idasrc.NewMCPHTTPClient(opts.IDAURL, hc)
	}
	return verifyServerboundRun(opts, client, stderr)
}

// verifyResult holds one classified serverbound entry.
type verifyResult struct {
	Op     string
	Opcode int
	FName  string

	// classification fields
	confirmed  bool
	mismatch   bool
	foundSet   []int // non-empty on mismatch
	unresolved bool
	reason     string // unresolved reason
}

// verifyServerboundRun is the pure (injectable) core of verify-serverbound.
func verifyServerboundRun(opts verifyServerboundOpts, client idasrc.MCPClient, stderr io.Writer) int {
	ctx := context.Background()

	// Step 1: load registry for this version.
	regPath := filepath.Join(opts.RegistryDir, opts.Version+".yaml")
	vf, err := loadRegistryOrEmpty(regPath)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit verify-serverbound: load registry %q: %v\n", regPath, err)
		return 3
	}

	// Collect serverbound entries.
	var serverbound []opregistry.Entry
	for _, e := range vf.Entries {
		if e.Direction == opregistry.DirServerbound {
			serverbound = append(serverbound, e)
		}
	}
	if len(serverbound) == 0 {
		fmt.Fprintf(stderr, "packet-audit verify-serverbound: no serverbound entries in registry %q\n", regPath)
		return 3
	}

	// Step 2: load audit reports for this version and build fname -> Address map.
	auditDir := filepath.Join(opts.AuditsDir, opts.Version)
	reports, err := matrix.LoadReports(auditDir)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit verify-serverbound: load audit reports %q: %v\n", auditDir, err)
		return 3
	}

	// Build baseFName -> Address from reports. baseFName = IDAName before '#'.
	fnameToAddr := map[string]string{}
	for _, r := range reports {
		if r.Address == "" || strings.EqualFold(r.Address, "ABSENT") || r.Address == "0x0" {
			continue
		}
		idaName := r.IDAName
		if idx := strings.Index(idaName, "#"); idx >= 0 {
			idaName = idaName[:idx]
		}
		if idaName != "" {
			// Last writer wins (multiple reports for same IDA function are fine —
			// they all share the same address).
			fnameToAddr[idaName] = r.Address
		}
	}

	// Step 3: for each serverbound entry with a resolved send-function address,
	// decompile and classify.
	results := make([]verifyResult, 0, len(serverbound))
	for _, e := range serverbound {
		res := verifyResult{
			Op:     e.Op,
			Opcode: e.Opcode,
			FName:  e.FName,
		}

		addr, hasAddr := fnameToAddr[e.FName]
		if !hasAddr || e.FName == "" {
			res.unresolved = true
			if e.FName == "" {
				res.reason = "no fname in registry"
			} else {
				res.reason = "no audit report address for fname"
			}
			results = append(results, res)
			continue
		}

		text, err := client.DecompileFunction(ctx, addr)
		if err != nil {
			res.unresolved = true
			res.reason = fmt.Sprintf("decompile error: %v", err)
			results = append(results, res)
			continue
		}

		found := discover.ParseSendOpcodes(text)
		if len(found) == 0 {
			res.unresolved = true
			res.reason = "decompile returned no COutPacket opcode literals (dynamic opcodes or empty decompile)"
			results = append(results, res)
			continue
		}

		// Check membership.
		inSet := false
		for _, v := range found {
			if v == e.Opcode {
				inSet = true
				break
			}
		}
		if inSet {
			res.confirmed = true
		} else {
			res.mismatch = true
			res.foundSet = found
		}
		results = append(results, res)
	}

	// Separate into buckets.
	var confirmed, mismatches, unresolved []verifyResult
	for _, r := range results {
		switch {
		case r.confirmed:
			confirmed = append(confirmed, r)
		case r.mismatch:
			mismatches = append(mismatches, r)
		default:
			unresolved = append(unresolved, r)
		}
	}

	// Step 4: write worklist markdown.
	md := buildVerifyServerboundWorklist(opts.Version, confirmed, mismatches, unresolved)
	if err := os.MkdirAll(filepath.Dir(opts.Out), 0o755); err != nil {
		fmt.Fprintf(stderr, "packet-audit verify-serverbound: mkdir %q: %v\n", filepath.Dir(opts.Out), err)
		return 3
	}
	if err := os.WriteFile(opts.Out, []byte(md), 0o644); err != nil {
		fmt.Fprintf(stderr, "packet-audit verify-serverbound: write worklist: %v\n", err)
		return 3
	}

	fmt.Printf("verify-serverbound %s: confirmed=%d mismatch=%d unresolved=%d → %s\n",
		opts.Version, len(confirmed), len(mismatches), len(unresolved), opts.Out)
	return 0
}

// buildVerifyServerboundWorklist renders the verification results as markdown.
func buildVerifyServerboundWorklist(version string, confirmed, mismatches, unresolved []verifyResult) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# verify-serverbound worklist — %s\n\n", version)
	fmt.Fprintf(&sb, "Generated by `packet-audit verify-serverbound --version %s`.\n", version)
	fmt.Fprintln(&sb, "D7 serverbound verification pass: checks that the registry opcode for each serverbound")
	fmt.Fprintln(&sb, "entry matches the opcode literal passed to `COutPacket::COutPacket` in the client send function.")
	fmt.Fprintln(&sb, "Send-function addresses are sourced from committed audit reports for this version.")
	fmt.Fprintln(&sb)

	fmt.Fprintf(&sb, "## Confirmed (%d)\n\n", len(confirmed))
	if len(confirmed) == 0 {
		fmt.Fprintln(&sb, "_No confirmed entries._")
	} else {
		fmt.Fprintln(&sb, "Registry opcode found in the decompiled send function's COutPacket constructor call set.")
		fmt.Fprintln(&sb)
		fmt.Fprintln(&sb, "| Op | Opcode | FName |")
		fmt.Fprintln(&sb, "|---|---|---|")
		for _, r := range confirmed {
			fmt.Fprintf(&sb, "| `%s` | `0x%03X` | `%s` |\n", r.Op, r.Opcode, r.FName)
		}
	}
	fmt.Fprintln(&sb)

	fmt.Fprintf(&sb, "## Mismatch — REVIEW (%d)\n\n", len(mismatches))
	if len(mismatches) == 0 {
		fmt.Fprintln(&sb, "_No mismatches._")
	} else {
		fmt.Fprintln(&sb, "Registry opcode is NOT in the set of opcodes found in the decompiled send function.")
		fmt.Fprintln(&sb, "This may indicate a wrong fname in the registry or a wrong opcode assignment.")
		fmt.Fprintln(&sb)
		fmt.Fprintln(&sb, "| Op | FName | Registry Opcode | Found Opcodes |")
		fmt.Fprintln(&sb, "|---|---|---|---|")
		for _, r := range mismatches {
			found := formatOpcodeSet(r.foundSet)
			fmt.Fprintf(&sb, "| `%s` | `%s` | `0x%03X` | %s |\n", r.Op, r.FName, r.Opcode, found)
		}
	}
	fmt.Fprintln(&sb)

	fmt.Fprintf(&sb, "## Unresolved (%d)\n\n", len(unresolved))
	if len(unresolved) == 0 {
		fmt.Fprintln(&sb, "_No unresolved entries._")
	} else {
		fmt.Fprintln(&sb, "Entries that could not be verified: no address in audit reports, decompile failed, or dynamic opcodes only.")
		fmt.Fprintln(&sb)
		fmt.Fprintln(&sb, "| Op | Opcode | FName | Reason |")
		fmt.Fprintln(&sb, "|---|---|---|---|")
		for _, r := range unresolved {
			fmt.Fprintf(&sb, "| `%s` | `0x%03X` | `%s` | %s |\n", r.Op, r.Opcode, r.FName, r.reason)
		}
	}
	fmt.Fprintln(&sb)

	return sb.String()
}

// formatOpcodeSet renders a slice of opcodes as a comma-separated hex list
// suitable for inline use in a markdown table cell.
func formatOpcodeSet(opcodes []int) string {
	if len(opcodes) == 0 {
		return "_(none)_"
	}
	parts := make([]string, len(opcodes))
	for i, v := range opcodes {
		parts[i] = fmt.Sprintf("`0x%03X`", v)
	}
	return strings.Join(parts, ", ")
}
