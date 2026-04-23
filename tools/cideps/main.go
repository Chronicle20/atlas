package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type output struct {
	GoServices     []GoServiceRow     `json:"go-services"`
	GoLibraries    []GoLibraryRow     `json:"go-libraries"`
	DockerServices []DockerServiceRow `json:"docker-services"`
	Reason         string             `json:"reason"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cideps", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		root            = fs.String("root", ".", "repo root")
		configPath      = fs.String("config", ".github/config/services.json", "path to services.json")
		changedLibsArg  = fs.String("changed-libs", "", "comma-separated lib short names")
		changedSvcsArg  = fs.String("changed-services", "", "comma-separated service short names")
		forceAll        = fs.Bool("force-all", false, "treat everything as affected")
	)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	g, err := BuildGraph(*root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	sel := Select(g, SelectInput{
		ChangedLibs:     splitCSV(*changedLibsArg),
		ChangedServices: splitCSV(*changedSvcsArg),
		ForceAll:        *forceAll,
	})

	out := output{
		GoServices:     cfg.EnrichGoServices(sel.Services),
		GoLibraries:    cfg.EnrichGoLibraries(sel.Libs),
		DockerServices: cfg.EnrichDockerServices(sel.Services),
		Reason:         buildReason(sel, *forceAll, *changedLibsArg, *changedSvcsArg),
	}

	var warnings []string
	cfg.Warnings(&warnings)
	for _, w := range warnings {
		fmt.Fprintln(stderr, "warning:", w)
	}

	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func buildReason(sel Selection, forceAll bool, changedLibs, changedSvcs string) string {
	if forceAll {
		return "force-all: rebuilding all services and libraries"
	}
	return fmt.Sprintf(
		"changed-libs=[%s] changed-services=[%s] → %d services, %d libraries affected",
		changedLibs, changedSvcs, len(sel.Services), len(sel.Libs),
	)
}
