// Package marker scans libs/atlas-packet test files for
// `packet-audit:verify` linkage comments (task-085 design §7).
package marker

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Marker holds the parsed fields of one packet-audit:verify comment.
type Marker struct {
	Packet  string
	Version string
	Address string
	File    string // path relative to scan root
	Line    int
}

const prefix = "// packet-audit:verify "

// Scan walks root for *_test.go files and collects all markers.
// Returned errs cover malformed markers and duplicate (packet,version) —
// both fail matrix --check.
func Scan(root string) ([]Marker, []string, error) {
	var all []Marker
	var errs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path // fall back to the absolute path in marker locations
		}
		ms, es := scanReader(f, filepath.ToSlash(rel))
		all = append(all, ms...)
		errs = append(errs, es...)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]Marker{}
	for _, m := range all {
		k := m.Packet + "|" + m.Version
		if prev, dup := seen[k]; dup {
			errs = append(errs, fmt.Sprintf("duplicate marker for %s × %s (%s:%d and %s:%d)",
				m.Packet, m.Version, prev.File, prev.Line, m.File, m.Line))
			continue
		}
		seen[k] = m
	}
	return all, errs, nil
}

func scanReader(r io.Reader, file string) ([]Marker, []string) {
	var ms []Marker
	var errs []string
	sc := bufio.NewScanner(r)
	line := 0
	seen := map[string]int{}
	for sc.Scan() {
		line++
		txt := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(txt, strings.TrimSpace(prefix)) {
			continue
		}
		m := Marker{File: file, Line: line}
		for _, kv := range strings.Fields(strings.TrimPrefix(txt, strings.TrimSpace(prefix))) {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) != 2 {
				continue
			}
			switch parts[0] {
			case "packet":
				m.Packet = parts[1]
			case "version":
				m.Version = parts[1]
			case "ida":
				m.Address = parts[1]
			}
		}
		if m.Packet == "" || m.Version == "" || m.Address == "" {
			errs = append(errs, fmt.Sprintf("%s:%d: malformed packet-audit:verify marker (need packet=, version=, ida=)", file, line))
			continue
		}
		k := m.Packet + "|" + m.Version
		if prev, dup := seen[k]; dup {
			errs = append(errs, fmt.Sprintf("%s: duplicate marker for %s × %s (lines %d and %d)", file, m.Packet, m.Version, prev, line))
			continue
		}
		seen[k] = line
		ms = append(ms, m)
	}
	if err := sc.Err(); err != nil {
		errs = append(errs, fmt.Sprintf("%s: scan error: %v", file, err))
	}
	return ms, errs
}
