// Command wzdiff compares this repository's parse of a WZ archive against
// a HaRepacker-style ".img.xml" dump of the same tree, and separately
// dumps the parse trace of one named image for offset-level diagnosis.
//
// Usage:
//
//	wzdiff --archive <path.wz> --reference <harepacker-dump-dir> [--allowlist <file>] [--trace <image>]
//
// Exit 0 only when there are zero unallowlisted deltas and both sides
// enumerate the same number of images; exit 1 otherwise, after printing a
// report in evidence-wz-parse-divergence-reactor.txt's format.
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
	reference := flag.String("reference", "", "path to the HaRepacker-style .img.xml dump directory (required unless --trace)")
	allowlistPath := flag.String("allowlist", "", "path to the reference-resolution allowlist (optional)")
	traceImage := flag.String("trace", "", "dump the parse trace for this one image (by name) to stdout, then exit")
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

	if *reference == "" {
		fmt.Fprintln(os.Stderr, "wzdiff: --reference is required unless --trace is given")
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

	if len(result.Divergent) == 0 && result.ImagesOurs == result.ImagesReference {
		os.Exit(0)
	}
	os.Exit(1)
}
