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

	// Non-Go services (atlas-ui, atlas-assets, atlas-pr-bootstrap, …) live in
	// services.json but never enter BuildGraph (no go.mod under atlas.com/).
	// Under --force-all, Select returns only graph services, which silently
	// drops non-Go services from the docker-services matrix even when their
	// Dockerfile sources changed. Merge them back in here so the rebuild
	// surface matches what services.json actually declares.
	dockerSvcNames := append([]string(nil), sel.Services...)
	if *forceAll {
		for _, s := range cfg.Services {
			if s.DockerImage == "" {
				continue
			}
			if !containsString(dockerSvcNames, s.Name) {
				dockerSvcNames = append(dockerSvcNames, s.Name)
			}
		}
	} else {
		// Not force-all: respect what the caller passed via --changed-services,
		// even for non-Go services (Select already preserves these, but be
		// defensive in case the graph filter drops one).
		for _, n := range splitCSV(*changedSvcsArg) {
			if !containsString(dockerSvcNames, n) {
				dockerSvcNames = append(dockerSvcNames, n)
			}
		}
	}

	out := output{
		GoServices:     cfg.EnrichGoServices(sel.Services),
		GoLibraries:    cfg.EnrichGoLibraries(sel.Libs),
		DockerServices: cfg.EnrichDockerServices(dockerSvcNames),
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

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
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
