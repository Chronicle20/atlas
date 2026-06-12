package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

type Options struct {
	CSVClientbound string
	CSVServerbound string
	Template       string
	AtlasPacket    string
	IDASource      string // "mcp" or path to export JSON
	Output         string
	VerifyExport   bool
}

func Run(args []string, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "export" {
		return runExport(args[1:], stderr)
	}
	if len(args) > 0 && args[0] == "validate" {
		return runValidate(args[1:], stderr)
	}
	if len(args) > 0 && args[0] == "infer" {
		return runInfer(args[1:], stderr)
	}
	if len(args) > 0 && args[0] == "decompose" {
		return runDecompose(args[1:], stderr)
	}
	if len(args) > 0 && args[0] == "triage" {
		return runTriage(args[1:], stderr)
	}
	if len(args) > 0 && args[0] == "resolve-dispatch" {
		return runResolveDispatch(args[1:], stderr)
	}
	if len(args) > 0 && args[0] == "diff-shape" {
		return runDiffShape(args[1:], stderr)
	}
	fs := flag.NewFlagSet("packet-audit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	opts := Options{}
	fs.StringVar(&opts.CSVClientbound, "csv-clientbound", "", "ClientBound CSV path")
	fs.StringVar(&opts.CSVServerbound, "csv-serverbound", "", "ServerBound CSV path")
	fs.StringVar(&opts.Template, "template", "", "template_<region>_<major>_<minor>.json path")
	fs.StringVar(&opts.AtlasPacket, "atlas-packet", "libs/atlas-packet", "atlas-packet library root")
	fs.StringVar(&opts.IDASource, "ida-source", "mcp", "'mcp' or path to ida-exports JSON")
	fs.StringVar(&opts.Output, "output", "docs/packets/audits", "output dir")
	fs.BoolVar(&opts.VerifyExport, "verify-export", false, "cross-check MCP vs export and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "packet-audit: flag parse error:", err)
		return 3
	}
	if opts.VerifyExport {
		fmt.Fprintln(stderr, "packet-audit: verify-export not yet implemented")
		return 3
	}
	if opts.CSVClientbound == "" || opts.CSVServerbound == "" || opts.Template == "" {
		fmt.Fprintln(stderr, "packet-audit: missing required flags --csv-clientbound, --csv-serverbound, --template")
		return 3
	}
	return runPipeline(opts, stderr)
}

// runExport is the export-subcommand flag wrapper. It parses export flags,
// derives paths from --version, picks the (non-deterministic) provenance
// timestamp here so the core (exportRun) stays deterministic given its opts,
// builds the real IDA-MCP client, and delegates to exportRun.
func runExport(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var eo exportOpts
	fs.StringVar(&eo.IDAURL, "ida-url", "http://192.168.20.3:13337/mcp", "IDA-MCP HTTP endpoint")
	fs.DurationVar(&eo.IDATimeout, "ida-timeout", 60*time.Second, "per-call IDA-MCP timeout")
	fs.StringVar(&eo.Version, "version", "", "target version key, e.g. gms_v95 (required)")
	fs.IntVar(&eo.DescentDepth, "descent-depth", 6, "max helper-descent recursion depth")
	fs.StringVar(&eo.Output, "output", "", "output JSON path (required)")
	var generatedAt string
	fs.StringVar(&generatedAt, "generated-at", "", "fixed provenance timestamp (default: now / $PACKET_AUDIT_GENERATED_AT)")
	var idaPort int
	fs.IntVar(&idaPort, "ida-port", 0, "IDA-MCP instance port to select (0 = default active instance; e.g. 13338 for a second loaded IDB)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "packet-audit export: flag parse error:", err)
		return 3
	}
	if eo.Version == "" || eo.Output == "" {
		fmt.Fprintln(stderr, "packet-audit export: missing required flags --version, --output")
		return 3
	}

	eo.PriorExport = "docs/packets/ida-exports/" + eo.Version + ".json"
	eo.Pending = "docs/packets/ida-exports/_pending.md"

	// Provenance timestamp is the only non-deterministic input; resolve it here
	// (NOT in exportRun) so the core is a pure function of its opts.
	eo.GeneratedAt = generatedAt
	if eo.GeneratedAt == "" {
		eo.GeneratedAt = os.Getenv("PACKET_AUDIT_GENERATED_AT")
	}
	if eo.GeneratedAt == "" {
		eo.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}

	hc := &http.Client{Timeout: eo.IDATimeout}
	var client idasrc.MCPClient
	if idaPort != 0 {
		client = idasrc.NewMCPHTTPClientWithInstance(eo.IDAURL, hc, idaPort)
	} else {
		client = idasrc.NewMCPHTTPClient(eo.IDAURL, hc)
	}
	return exportRun(eo, client, os.Stdout, stderr)
}

// runValidate is the validate-subcommand flag wrapper. It parses validate flags,
// derives the baseline path from --version (overridable), builds the real
// IDA-MCP client, and delegates to validateRun.
func runValidate(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var vo validateOpts
	var version, idaURL string
	var idaTimeout time.Duration
	fs.StringVar(&version, "version", "", "target version key, e.g. gms_v95 (required)")
	fs.StringVar(&vo.Baseline, "baseline", "", "baseline export JSON path (default: docs/packets/ida-exports/<version>.json)")
	fs.StringVar(&vo.Report, "report", "", "output markdown report path (required)")
	fs.StringVar(&vo.Allowlist, "allowlist", "", "unimplemented-case allowlist (default: docs/packets/audits/<auditdir>/_unimplemented.json)")
	fs.StringVar(&idaURL, "ida-url", "http://192.168.20.3:13337/mcp", "IDA-MCP HTTP endpoint")
	fs.DurationVar(&idaTimeout, "ida-timeout", 60*time.Second, "per-call IDA-MCP timeout")
	fs.IntVar(&vo.DescentDepth, "descent-depth", 6, "max helper-descent recursion depth")
	var idaPort int
	fs.IntVar(&idaPort, "ida-port", 0, "IDA-MCP instance port to select (0 = default active instance; e.g. 13338 for a second loaded IDB)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "packet-audit validate: flag parse error:", err)
		return 3
	}
	if version == "" || vo.Report == "" {
		fmt.Fprintln(stderr, "packet-audit validate: missing required flags --version, --report")
		return 3
	}
	if vo.Baseline == "" {
		vo.Baseline = "docs/packets/ida-exports/" + version + ".json"
	}
	if vo.Allowlist == "" {
		// The jms audit dir is named jms_v185, not gms_jms_185 (region/major
		// convention). Map the version key to its audit dir for the default path.
		auditDir := version
		if version == "gms_jms_185" {
			auditDir = "jms_v185"
		}
		vo.Allowlist = "docs/packets/audits/" + auditDir + "/_unimplemented.json"
	}

	hc := &http.Client{Timeout: idaTimeout}
	var client idasrc.MCPClient
	if idaPort != 0 {
		client = idasrc.NewMCPHTTPClientWithInstance(idaURL, hc, idaPort)
	} else {
		client = idasrc.NewMCPHTTPClient(idaURL, hc)
	}
	return validateRun(vo, client, os.Stdout)
}

// runInfer is the infer-subcommand flag wrapper. It parses infer flags, derives
// the baseline path from --version (overridable), builds the real IDA-MCP client,
// and delegates to inferRun.
func runInfer(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit infer", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var io_ inferOpts
	var version, idaURL string
	var idaTimeout time.Duration
	fs.StringVar(&version, "version", "", "target version key, e.g. gms_v95 (required)")
	fs.StringVar(&io_.Baseline, "baseline", "", "baseline export JSON path (default: docs/packets/ida-exports/<version>.json)")
	fs.StringVar(&io_.Out, "out", "", "output proposal JSON path (required)")
	fs.Float64Var(&io_.MinConfidence, "min-confidence", 0.6, "high-confidence threshold for the roll-up")
	fs.StringVar(&idaURL, "ida-url", "http://192.168.20.3:13337/mcp", "IDA-MCP HTTP endpoint")
	fs.DurationVar(&idaTimeout, "ida-timeout", 60*time.Second, "per-call IDA-MCP timeout")
	fs.IntVar(&io_.DescentDepth, "descent-depth", 6, "max helper-descent recursion depth")
	var idaPort int
	fs.IntVar(&idaPort, "ida-port", 0, "IDA-MCP instance port to select (0 = default active instance; e.g. 13338 for a second loaded IDB)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "packet-audit infer: flag parse error:", err)
		return 3
	}
	if version == "" || io_.Out == "" {
		fmt.Fprintln(stderr, "packet-audit infer: missing required flags --version, --out")
		return 3
	}
	if io_.Baseline == "" {
		io_.Baseline = "docs/packets/ida-exports/" + version + ".json"
	}

	hc := &http.Client{Timeout: idaTimeout}
	var client idasrc.MCPClient
	if idaPort != 0 {
		client = idasrc.NewMCPHTTPClientWithInstance(idaURL, hc, idaPort)
	} else {
		client = idasrc.NewMCPHTTPClient(idaURL, hc)
	}
	return inferRun(io_, client, os.Stdout)
}

// runDiffShape is the diff-shape-subcommand flag wrapper. It parses flags,
// derives the baseline path from --version (overridable), builds the real IDA-MCP
// client, and delegates to diffShapeRun (a read-only diagnostic).
func runDiffShape(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit diff-shape", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var do diffShapeOpts
	var version, idaURL string
	var idaTimeout time.Duration
	fs.StringVar(&version, "version", "", "target version key, e.g. gms_v95 (required)")
	fs.StringVar(&do.Baseline, "baseline", "", "baseline export JSON path (default: docs/packets/ida-exports/<version>.json)")
	fs.StringVar(&do.Report, "report", "", "output markdown report path (required)")
	fs.StringVar(&idaURL, "ida-url", "http://192.168.20.3:13337/mcp", "IDA-MCP HTTP endpoint")
	fs.DurationVar(&idaTimeout, "ida-timeout", 60*time.Second, "per-call IDA-MCP timeout")
	fs.IntVar(&do.DescentDepth, "descent-depth", 6, "max helper-descent recursion depth")
	var idaPort int
	fs.IntVar(&idaPort, "ida-port", 0, "IDA-MCP instance port to select (0 = default active instance; e.g. 13338 for a second loaded IDB)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "packet-audit diff-shape: flag parse error:", err)
		return 3
	}
	if version == "" || do.Report == "" {
		fmt.Fprintln(stderr, "packet-audit diff-shape: missing required flags --version, --report")
		return 3
	}
	if do.Baseline == "" {
		do.Baseline = "docs/packets/ida-exports/" + version + ".json"
	}

	hc := &http.Client{Timeout: idaTimeout}
	var client idasrc.MCPClient
	if idaPort != 0 {
		client = idasrc.NewMCPHTTPClientWithInstance(idaURL, hc, idaPort)
	} else {
		client = idasrc.NewMCPHTTPClient(idaURL, hc)
	}
	return diffShapeRun(do, client, os.Stdout)
}

// runResolveDispatch is the resolve-dispatch-subcommand flag wrapper. It parses
// flags, derives the baseline path from --version (overridable), builds the real
// IDA-MCP client, and delegates to resolveDispatchRun. Unlike infer, this command
// MUTATES the baseline (writing high-confidence selectors) and emits a
// confirmation worklist of the low-confidence picks.
func runResolveDispatch(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit resolve-dispatch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var ro resolveDispatchOpts
	var version, idaURL string
	var idaTimeout time.Duration
	fs.StringVar(&version, "version", "", "target version key, e.g. gms_v95 (required)")
	fs.StringVar(&ro.Baseline, "baseline", "", "baseline export JSON path (default: docs/packets/ida-exports/<version>.json)")
	fs.StringVar(&ro.Worklist, "worklist", "", "output confirmation worklist markdown path (required)")
	fs.Float64Var(&ro.MinConfidence, "min-confidence", 0.6, "auto-accept threshold")
	fs.StringVar(&idaURL, "ida-url", "http://192.168.20.3:13337/mcp", "IDA-MCP HTTP endpoint")
	fs.DurationVar(&idaTimeout, "ida-timeout", 60*time.Second, "per-call IDA-MCP timeout")
	fs.IntVar(&ro.DescentDepth, "descent-depth", 6, "max helper-descent recursion depth")
	var idaPort int
	fs.IntVar(&idaPort, "ida-port", 0, "IDA-MCP instance port to select (0 = default active instance; e.g. 13338 for a second loaded IDB)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "packet-audit resolve-dispatch: flag parse error:", err)
		return 3
	}
	if version == "" || ro.Worklist == "" {
		fmt.Fprintln(stderr, "packet-audit resolve-dispatch: missing required flags --version, --worklist")
		return 3
	}
	if ro.Baseline == "" {
		ro.Baseline = "docs/packets/ida-exports/" + version + ".json"
	}

	hc := &http.Client{Timeout: idaTimeout}
	var client idasrc.MCPClient
	if idaPort != 0 {
		client = idasrc.NewMCPHTTPClientWithInstance(idaURL, hc, idaPort)
	} else {
		client = idasrc.NewMCPHTTPClient(idaURL, hc)
	}
	return resolveDispatchRun(ro, client, os.Stdout)
}

// runDecompose is the decompose-subcommand flag wrapper. It parses decompose
// flags, derives the baseline + audit-dir paths from --version (overridable),
// builds the real IDA-MCP client, and delegates to decomposeRun.
func runDecompose(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit decompose", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var do decomposeOpts
	var idaURL string
	var idaTimeout time.Duration
	fs.StringVar(&do.Version, "version", "", "target version key, e.g. gms_v83 (required)")
	fs.StringVar(&do.Baseline, "baseline", "", "baseline export JSON path (default: docs/packets/ida-exports/<version>.json)")
	fs.StringVar(&do.AuditDir, "audit-dir", "", "committed audit dir (default: docs/packets/audits/<version>)")
	fs.StringVar(&do.Out, "out", "", "output extended baseline JSON path (required)")
	fs.StringVar(&do.Report, "report", "", "output markdown report path (required)")
	fs.StringVar(&idaURL, "ida-url", "http://192.168.20.3:13337/mcp", "IDA-MCP HTTP endpoint")
	fs.DurationVar(&idaTimeout, "ida-timeout", 60*time.Second, "per-call IDA-MCP timeout")
	fs.IntVar(&do.DescentDepth, "descent-depth", 6, "max helper-descent recursion depth")
	var idaPort int
	fs.IntVar(&idaPort, "ida-port", 0, "IDA-MCP instance port to select (0 = default active instance; e.g. 13338 for a second loaded IDB)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "packet-audit decompose: flag parse error:", err)
		return 3
	}
	if do.Version == "" || do.Out == "" || do.Report == "" {
		fmt.Fprintln(stderr, "packet-audit decompose: missing required flags --version, --out, --report")
		return 3
	}
	if do.Baseline == "" {
		do.Baseline = "docs/packets/ida-exports/" + do.Version + ".json"
	}
	if do.AuditDir == "" {
		do.AuditDir = "docs/packets/audits/" + do.Version
	}

	hc := &http.Client{Timeout: idaTimeout}
	var client idasrc.MCPClient
	if idaPort != 0 {
		client = idasrc.NewMCPHTTPClientWithInstance(idaURL, hc, idaPort)
	} else {
		client = idasrc.NewMCPHTTPClient(idaURL, hc)
	}
	return decomposeRun(do, client, os.Stdout)
}

// runTriage is the triage-subcommand flag wrapper. It parses triage flags,
// derives the baseline + audit-dir paths from --version (overridable), builds
// the real IDA-MCP client, and delegates to triageRun.
func runTriage(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit triage", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var to triageOpts
	var idaURL string
	var idaTimeout time.Duration
	fs.StringVar(&to.Version, "version", "", "target version key, e.g. gms_v95 (required)")
	fs.StringVar(&to.Baseline, "baseline", "", "baseline export JSON path (default: docs/packets/ida-exports/<version>.json)")
	fs.StringVar(&to.AuditDir, "audit-dir", "", "committed audit dir (default: docs/packets/audits/<version>)")
	fs.StringVar(&to.Report, "report", "", "output markdown worklist path (required)")
	fs.StringVar(&idaURL, "ida-url", "http://192.168.20.3:13337/mcp", "IDA-MCP HTTP endpoint")
	fs.DurationVar(&idaTimeout, "ida-timeout", 60*time.Second, "per-call IDA-MCP timeout")
	fs.IntVar(&to.DescentDepth, "descent-depth", 6, "max helper-descent recursion depth")
	var idaPort int
	fs.IntVar(&idaPort, "ida-port", 0, "IDA-MCP instance port to select (0 = default active instance; e.g. 13338 for a second loaded IDB)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "packet-audit triage: flag parse error:", err)
		return 3
	}
	if to.Version == "" || to.Report == "" {
		fmt.Fprintln(stderr, "packet-audit triage: missing required flags --version, --report")
		return 3
	}
	if to.Baseline == "" {
		to.Baseline = "docs/packets/ida-exports/" + to.Version + ".json"
	}
	if to.AuditDir == "" {
		to.AuditDir = "docs/packets/audits/" + to.Version
	}

	hc := &http.Client{Timeout: idaTimeout}
	var client idasrc.MCPClient
	if idaPort != 0 {
		client = idasrc.NewMCPHTTPClientWithInstance(idaURL, hc, idaPort)
	} else {
		client = idasrc.NewMCPHTTPClient(idaURL, hc)
	}
	return triageRun(to, client, os.Stdout)
}
