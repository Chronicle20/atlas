package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// fname-doc maintains a `// packet-audit:fname <IDAName>` comment on every
// packet-definition struct in libs/atlas-packet, so each codec self-documents
// the IDA function it encodes/decodes. The fname is resolved from the committed
// audit reports (docs/packets/audits/<version>/<WriterName>.json), whose
// WriterName is Title(family-dir)+StructName — the same convention the audit
// tooling uses — joined to the report's IDAName.
//
// Two modes:
//   generate (default): insert/update the comment in place.
//   --check: fail (exit 1) on any DRIFT (comment present but != resolved fname)
//            or any resolvable struct MISSING the comment. Structs with no
//            matching audit report carry no fname (none exists to cite) and are
//            reported but never fail — we do not invent fnames.

type fnameDocOpts struct {
	PacketLib string
	AuditsDir string
	Families  string
	Check     bool
}

const fnameMarker = "// packet-audit:fname "

var (
	reOperation = regexp.MustCompile(`func \(\w+ \*?(\w+)\) Operation\(\) string`)
	reTypeDecl  = regexp.MustCompile(`(?m)^type (\w+) struct\b`)
)

func runFnameDoc(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit fname-doc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var o fnameDocOpts
	fs.StringVar(&o.PacketLib, "packet-lib", "libs/atlas-packet", "atlas-packet library root")
	fs.StringVar(&o.AuditsDir, "audits-dir", "docs/packets/audits", "audit reports parent dir")
	fs.StringVar(&o.Families, "families", "docs/packets/evidence/families.yaml", "dispatcher membership YAML (for the family tag)")
	fs.BoolVar(&o.Check, "check", false, "verify comments are present and current; do not write")
	if err := fs.Parse(args); err != nil {
		return 3
	}
	return fnameDocRun(o, os.Stdout, stderr)
}

func fnameDocRun(o fnameDocOpts, stdout, stderr io.Writer) int {
	writerToFName, err := loadReportFNames(o.AuditsDir)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit fname-doc: %v\n", err)
		return 3
	}
	dispatchers := loadDispatcherSet(o.Families)

	files, err := goFiles(o.PacketLib)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit fname-doc: %v\n", err)
		return 3
	}

	var drift, missing, updated, unresolved []string
	for _, f := range files {
		fam := familyOf(f, o.PacketLib)
		if fam == "" {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(stderr, "read %s: %v\n", f, err)
			return 3
		}
		structs := codecStructs(string(src))
		if len(structs) == 0 {
			continue
		}
		out := string(src)
		changed := false
		for _, st := range structs {
			fname, ok := writerToFName[title(fam)+st]
			if !ok {
				unresolved = append(unresolved, fmt.Sprintf("%s:%s", fam, st))
				continue
			}
			want := fnameMarker + fname
			if base := baseFNameLocal(fname); dispatchers[base] {
				want += "  (dispatcher family — see docs/packets/evidence/families.yaml)"
			}
			newOut, state := applyComment(out, st, want)
			switch state {
			case commentOK:
			case commentMissing:
				if o.Check {
					missing = append(missing, fmt.Sprintf("%s %s (want %q)", f, st, fname))
				} else {
					out, changed = newOut, true
					updated = append(updated, fmt.Sprintf("%s %s", f, st))
				}
			case commentDrift:
				if o.Check {
					drift = append(drift, fmt.Sprintf("%s %s (want %q)", f, st, fname))
				} else {
					out, changed = newOut, true
					updated = append(updated, fmt.Sprintf("%s %s", f, st))
				}
			}
		}
		if changed {
			if err := os.WriteFile(f, []byte(out), 0o644); err != nil {
				fmt.Fprintf(stderr, "write %s: %v\n", f, err)
				return 3
			}
		}
	}

	sort.Strings(unresolved)
	if o.Check {
		if len(drift) > 0 || len(missing) > 0 {
			for _, d := range drift {
				fmt.Fprintf(stderr, "fname-doc DRIFT: %s\n", d)
			}
			for _, m := range missing {
				fmt.Fprintf(stderr, "fname-doc MISSING: %s\n", m)
			}
			fmt.Fprintf(stderr, "fname-doc: %d drift, %d missing — run `packet-audit fname-doc` to fix\n", len(drift), len(missing))
			return 1
		}
		fmt.Fprintf(stdout, "fname-doc check OK (%d structs without an audit report carry no fname)\n", len(unresolved))
		return 0
	}
	fmt.Fprintf(stdout, "fname-doc: updated %d comment(s); %d struct(s) have no audit report (no fname to cite)\n", len(updated), len(unresolved))
	return 0
}

type commentState int

const (
	commentOK commentState = iota
	commentMissing
	commentDrift
)

// applyComment ensures the line immediately above `type <st> struct` is exactly
// `want`. Returns the (possibly rewritten) source and the prior state.
func applyComment(src, st, want string) (string, commentState) {
	lines := strings.Split(src, "\n")
	decl := "type " + st + " struct"
	for i, ln := range lines {
		if !strings.HasPrefix(ln, decl) {
			continue
		}
		if i > 0 && strings.HasPrefix(strings.TrimSpace(lines[i-1]), fnameMarker) {
			if strings.TrimSpace(lines[i-1]) == want {
				return src, commentOK
			}
			lines[i-1] = want
			return strings.Join(lines, "\n"), commentDrift
		}
		// Insert above the type decl. If the immediately-preceding doc block
		// contains a list/indented line, gofmt requires a blank `//` separator
		// between that block and a trailing non-list line; emit it so the
		// generated source stays gofmt-clean.
		ins := append([]string{}, lines[:i]...)
		if i > 0 && precedingBlockHasList(lines, i-1) {
			ins = append(ins, "//")
		}
		ins = append(ins, want)
		ins = append(ins, lines[i:]...)
		return strings.Join(ins, "\n"), commentMissing
	}
	return src, commentOK // struct decl not found on its own line (skip)
}

// precedingBlockHasList walks upward from line idx over a contiguous block of
// `//` comment lines and reports whether any is a list item or indented
// continuation (`//   - ...` / `//    foo`). gofmt separates such a block from a
// trailing plain comment line with a blank `//` line.
func precedingBlockHasList(lines []string, idx int) bool {
	for j := idx; j >= 0; j-- {
		t := strings.TrimRight(lines[j], " \t")
		if !strings.HasPrefix(strings.TrimSpace(t), "//") {
			break
		}
		body := strings.TrimPrefix(strings.TrimSpace(t), "//")
		if strings.HasPrefix(body, "  ") { // 2+ leading spaces after // => list/indented
			return true
		}
		ls := strings.TrimSpace(body)
		if strings.HasPrefix(ls, "- ") || strings.HasPrefix(ls, "* ") {
			return true
		}
	}
	return false
}

// codecStructs returns struct names in src that have an Operation() string method.
func codecStructs(src string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range reOperation.FindAllStringSubmatch(src, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

// fnamedocOrder is the priority order for resolving WriterName->IDAName: the
// first version that supplies a report wins. v95 is first (PDB-named,
// authoritative). The gms_jms_185 alias is kept for historical audit-dir compat.
var fnamedocOrder = []string{"gms_v95", "gms_v83", "gms_v84", "gms_v87", "gms_v79", "gms_v72", "gms_v61", "gms_v48", "jms_v185", "gms_jms_185"}

// loadReportFNames builds WriterName -> IDAName across all version audit dirs,
// preferring the v95 report (PDB-named, authoritative) when a writer appears in
// several versions.
func loadReportFNames(auditsDir string) (map[string]string, error) {
	order := fnamedocOrder
	out := map[string]string{}
	for _, v := range order {
		matches, _ := filepath.Glob(filepath.Join(auditsDir, v, "*.json"))
		for _, p := range matches {
			raw, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			var r struct{ WriterName, IDAName string }
			if json.Unmarshal(raw, &r) != nil {
				continue
			}
			if r.WriterName == "" || r.IDAName == "" {
				continue
			}
			if _, ok := out[r.WriterName]; !ok {
				out[r.WriterName] = r.IDAName
			}
		}
	}
	return out, nil
}

// loadDispatcherSet reads the dispatcher base fnames from families.yaml without
// pulling in the matrix package (avoids an import cycle / keeps this standalone).
func loadDispatcherSet(path string) map[string]bool {
	out := map[string]bool{}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	inList := false
	for _, ln := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "dispatchers:") {
			inList = true
			continue
		}
		if inList && strings.HasPrefix(t, "- ") {
			v := strings.TrimSpace(strings.TrimPrefix(t, "- "))
			if i := strings.Index(v, "#"); i >= 0 {
				v = strings.TrimSpace(v[:i])
			}
			if v != "" {
				out[v] = true
			}
		} else if inList && t != "" && !strings.HasPrefix(t, "#") {
			break
		}
	}
	return out
}

func goFiles(root string) ([]string, error) {
	var out []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		out = append(out, p)
		return nil
	})
	sort.Strings(out)
	return out, err
}

// familyOf returns the path component immediately under packetLib, e.g.
// libs/atlas-packet/field/clientbound/x.go -> "field".
func familyOf(file, packetLib string) string {
	rel, err := filepath.Rel(packetLib, file)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func baseFNameLocal(ida string) string {
	if i := strings.Index(ida, "#"); i >= 0 {
		return ida[:i]
	}
	return ida
}
