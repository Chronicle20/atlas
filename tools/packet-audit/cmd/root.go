package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
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
	fmt.Fprintln(stderr, "packet-audit: pipeline not yet implemented")
	return 3
}
