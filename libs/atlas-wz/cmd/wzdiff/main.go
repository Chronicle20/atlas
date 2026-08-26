// Command wzdiff compares this repository's parse of a WZ archive against
// an ".img.xml" dump of the same tree in a HaRepacker-style convention,
// dumps the parse trace of one named image for offset-level diagnosis, and
// runs a whole-archive size-accounting self-check that needs no external
// dump at all — the archive's own declared sub-object sizes are the
// oracle (task-262).
//
// Usage:
//
//	wzdiff --archive <path.wz> --reference <dump-dir> [--allowlist <file>] [--trace <image>] [--selfcheck]
//
// With --reference, exit 0 only when there are zero unallowlisted deltas
// and both sides enumerate the same set of images (not merely the same
// count); exit 1 otherwise, after printing a report in
// evidence-wz-parse-divergence-reactor.txt's format.
//
// With --selfcheck, --reference is not required (mirroring --trace): exit
// 0 only when every type-9 sub-object's decode ended exactly where its own
// declared size said it would and every image parsed; exit 1 otherwise,
// after printing a size-accounting report.
//
// All comparison and formatting logic lives in the testable wzdiff
// package; main is flag parsing and dispatch only.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wzdiff"
)

func main() {
	archive := flag.String("archive", "", "path to the WZ archive (.wz) to compare (required)")
	reference := flag.String("reference", "", "path to the .img.xml dump directory (required unless --trace or --selfcheck)")
	allowlistPath := flag.String("allowlist", "", "path to the reference-resolution allowlist (optional)")
	traceImage := flag.String("trace", "", "dump the parse trace for this one image (by name) to stdout, then exit")
	selfcheck := flag.Bool("selfcheck", false, "run the whole-archive size-accounting self-check (no --reference needed), then exit")
	flag.Parse()

	l := logrus.New()

	if *archive == "" {
		fmt.Fprintln(os.Stderr, "wzdiff: --archive is required")
		flag.Usage()
		os.Exit(2)
	}

	if *traceImage != "" {
		if err := wzdiff.Trace(l, *archive, *traceImage, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "wzdiff:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *selfcheck {
		result, err := wzdiff.SelfCheck(l, *archive)
		if err != nil {
			fmt.Fprintln(os.Stderr, "wzdiff:", err)
			os.Exit(1)
		}
		wzdiff.WriteSelfCheckReport(os.Stdout, result)
		if len(result.Violations) == 0 && len(result.ParseErrors) == 0 {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if *reference == "" {
		fmt.Fprintln(os.Stderr, "wzdiff: --reference is required unless --trace or --selfcheck is given")
		flag.Usage()
		os.Exit(2)
	}

	var allow []wzdiff.AllowEntry
	if *allowlistPath != "" {
		var err error
		allow, err = wzdiff.LoadAllowlist(*allowlistPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "wzdiff:", err)
			os.Exit(1)
		}
	}

	result, err := wzdiff.Run(l, *archive, *reference, allow)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wzdiff:", err)
		os.Exit(1)
	}

	wzdiff.WriteReport(os.Stdout, result)

	if len(result.Divergent) == 0 && len(result.OnlyOurs) == 0 && len(result.OnlyReference) == 0 {
		os.Exit(0)
	}
	os.Exit(1)
}
