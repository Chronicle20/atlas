package wzdiff

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// AllowEntry is one adjudicated reference-resolution divergence: a case
// where the HaRepacker dump shows a `link`/`_inlink`/`_outlink`/UOL
// reference already resolved (Atlas resolves those in its own consumers,
// not in the parser — see reactor/reader.go, map/reader.go,
// icons/extract.go) and our parser stayed deliberately literal.
//
// An entry is never a broad waiver: it names one image and one path, and
// Allowed only drops a Delta that matches both, in the recorded direction.
// It must never be used to make a PARSER-DEFECT delta disappear.
type AllowEntry struct {
	// Image is the image name the entry applies to, matching the key Run
	// uses to compare a tree pair (no ".img"/".img.xml" suffix).
	Image string
	// Path is the delta path the entry covers, in wzdiff's "/<Kind>:<Name>"
	// notation. It also covers every descendant of that path, since a
	// resolved reference substitutes a whole subtree.
	Path string
	// OnlyIn is the direction the entry covers: "reference" or "ours",
	// matching Delta.OnlyIn.
	OnlyIn string
	// Reason is a short, human-readable justification recorded for audit,
	// e.g. "UOL resolved by HaRepacker; Atlas resolves in reactor/reader.go".
	Reason string
}

// LoadAllowlist reads a tab-separated allowlist file: one entry per
// non-blank, non-comment line, fields in the order
// "image<TAB>path<TAB>onlyIn<TAB>reason". Lines starting with "#" (after
// leading whitespace) and blank lines are skipped. Deliberately not YAML:
// this is the smallest format that lets a reviewer read a diff of the
// allowlist file directly, and it adds no new module dependency.
func LoadAllowlist(path string) ([]AllowEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("wzdiff: open allowlist %s: %w", path, err)
	}
	defer f.Close()

	var entries []AllowEntry
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			return nil, fmt.Errorf("wzdiff: allowlist %s:%d: expected 4 tab-separated fields, got %d", path, lineNo, len(fields))
		}
		entries = append(entries, AllowEntry{
			Image:  fields[0],
			Path:   fields[1],
			OnlyIn: fields[2],
			Reason: fields[3],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("wzdiff: read allowlist %s: %w", path, err)
	}
	return entries, nil
}

// Allowed reports whether d, found on image, is covered by any entry in
// entries: same image, same direction, and d.Path is the entry's path or a
// descendant of it (a resolved reference substitutes a whole subtree, so
// covering the substituted node's path also covers everything under it).
func Allowed(entries []AllowEntry, image string, d Delta) bool {
	for _, e := range entries {
		if e.Image != image || e.OnlyIn != d.OnlyIn {
			continue
		}
		if d.Path == e.Path || strings.HasPrefix(d.Path, e.Path+"/") {
			return true
		}
	}
	return false
}
