package cmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"gopkg.in/yaml.v3"
)

// operations validates (and generates) the tenant template `operations` mode
// tables against the IDA-verified per-version dispatcher enumerations in
// docs/packets/dispatchers/*.yaml — the SOURCE OF TRUTH for each writer's
// string→mode map. Keeping the templates in sync means a mode byte that drifts
// from the client's actual switch (→ ResolveCode returns 99 → client crash) is
// caught in CI, not at runtime.
//
//   default   — generate: write each writer's options.operations from the YAML.
//   --check   — exit 1 on drift (template != YAML), a missing table (YAML covers
//               the version but the writer lacks operations), or extra/stale
//               keys. A writer absent from a template is reported, not failed.

type operationsOpts struct {
	DispatchersDir string
	TemplatesDir   string
	Check          bool
}

type dispatcherDoc struct {
	Writer     string `yaml:"writer"`
	FName      string `yaml:"fname"`
	Op         string `yaml:"op"`
	Operations []struct {
		Key   string         `yaml:"key"`
		Modes map[string]int `yaml:"modes"`
	} `yaml:"operations"`
}

func runOperations(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit operations", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var o operationsOpts
	fs.StringVar(&o.DispatchersDir, "dispatchers-dir", "docs/packets/dispatchers", "per-version dispatcher enumeration YAML dir")
	fs.StringVar(&o.TemplatesDir, "templates-dir", "services/atlas-configurations/seed-data/templates", "tenant seed templates dir")
	fs.BoolVar(&o.Check, "check", false, "verify templates match the enumerations; do not write")
	if err := fs.Parse(args); err != nil {
		return 3
	}
	return operationsRun(o, os.Stdout, stderr)
}

func operationsRun(o operationsOpts, stdout, stderr io.Writer) int {
	docs, err := loadDispatcherDocs(o.DispatchersDir)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit operations: %v\n", err)
		return 3
	}

	var drift, missing, extra, absent []string
	wroteN := 0
	for _, vk := range matrix.VersionKeys {
		tplPath := filepath.Join(o.TemplatesDir, filepath.Base(matrix.TemplatePath(vk)))
		raw, err := os.ReadFile(tplPath)
		if err != nil {
			fmt.Fprintf(stderr, "packet-audit operations: %v\n", err)
			return 3
		}
		root, err := parseNode(raw)
		if err != nil {
			fmt.Fprintf(stderr, "packet-audit operations: parse %s: %v\n", tplPath, err)
			return 3
		}
		writers := writersOf(root)
		changed := false
		for _, doc := range docs {
			expected := expectedTable(doc, vk)
			w := findWriterNode(writers, doc.Writer)
			if w == nil {
				if len(expected) > 0 {
					absent = append(absent, fmt.Sprintf("%s: writer %q not in template (cannot populate %d ops)", vk, doc.Writer, len(expected)))
				}
				continue
			}
			if len(expected) == 0 {
				continue
			}
			got := operationsOf(w)
			for k, want := range expected {
				if gv, ok := got[k]; !ok {
					missing = append(missing, fmt.Sprintf("%s %s: key %q missing (want %d)", vk, doc.Writer, k, want))
				} else if gv != want {
					drift = append(drift, fmt.Sprintf("%s %s: key %q is %d, want %d", vk, doc.Writer, k, gv, want))
				}
			}
			for k := range got {
				if _, ok := expected[k]; !ok {
					extra = append(extra, fmt.Sprintf("%s %s: key %q in template but not in enumeration", vk, doc.Writer, k))
				}
			}
			if !o.Check && setOperations(w, doc, expected) {
				changed = true
			}
		}
		if !o.Check && changed {
			out, err := encodeNode(root)
			if err != nil {
				fmt.Fprintf(stderr, "packet-audit operations: encode %s: %v\n", tplPath, err)
				return 3
			}
			if err := os.WriteFile(tplPath, out, 0o644); err != nil {
				fmt.Fprintf(stderr, "packet-audit operations: write %s: %v\n", tplPath, err)
				return 3
			}
			wroteN++
		}
	}

	sort.Strings(drift)
	sort.Strings(missing)
	sort.Strings(extra)
	sort.Strings(absent)
	if o.Check {
		for _, s := range drift {
			fmt.Fprintf(stderr, "operations DRIFT: %s\n", s)
		}
		for _, s := range missing {
			fmt.Fprintf(stderr, "operations MISSING: %s\n", s)
		}
		for _, s := range extra {
			fmt.Fprintf(stderr, "operations EXTRA: %s\n", s)
		}
		for _, s := range absent {
			fmt.Fprintf(stderr, "operations note (writer absent): %s\n", s)
		}
		if n := len(drift) + len(missing) + len(extra); n > 0 {
			fmt.Fprintf(stderr, "operations: %d drift, %d missing, %d extra — run `packet-audit operations` to regenerate\n", len(drift), len(missing), len(extra))
			return 1
		}
		fmt.Fprintf(stdout, "operations check OK (%d absent-writer note(s))\n", len(absent))
		return 0
	}
	for _, s := range absent {
		fmt.Fprintf(stderr, "operations note (writer absent): %s\n", s)
	}
	fmt.Fprintf(stdout, "operations: wrote %d template(s)\n", wroteN)
	return 0
}

func loadDispatcherDocs(dir string) ([]dispatcherDoc, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	var out []dispatcherDoc
	for _, p := range matches {
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var d dispatcherDoc
		if err := yaml.Unmarshal(raw, &d); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		if d.Writer != "" {
			out = append(out, d)
		}
	}
	return out, nil
}

func expectedTable(d dispatcherDoc, version string) map[string]int {
	m := map[string]int{}
	for _, op := range d.Operations {
		if v, ok := op.Modes[version]; ok {
			m[op.Key] = v
		}
	}
	return m
}

// ---- order-preserving recursive JSON node ----

type node struct {
	kind byte // 'o' object, 'a' array, 's' scalar
	keys []string
	obj  map[string]*node
	arr  []*node
	raw  json.RawMessage // scalars (string/number/bool/null)
}

func parseNode(b []byte) (*node, error) {
	return readNode(json.RawMessage(b))
}

// readNode parses one JSON value, preserving object key order AND scalar bytes
// verbatim (so \uXXXX escapes, number formatting, etc. survive a round-trip
// untouched — only the operations maps we set are re-emitted).
func readNode(raw json.RawMessage) (*node, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty JSON value")
	}
	switch trimmed[0] {
	case '{':
		n := &node{kind: 'o', obj: map[string]*node{}}
		dec := json.NewDecoder(bytes.NewReader(trimmed))
		if _, err := dec.Token(); err != nil { // opening {
			return nil, err
		}
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key := kt.(string)
			var cv json.RawMessage
			if err := dec.Decode(&cv); err != nil {
				return nil, err
			}
			child, err := readNode(cv)
			if err != nil {
				return nil, err
			}
			n.keys = append(n.keys, key)
			n.obj[key] = child
		}
		return n, nil
	case '[':
		n := &node{kind: 'a'}
		dec := json.NewDecoder(bytes.NewReader(trimmed))
		if _, err := dec.Token(); err != nil { // opening [
			return nil, err
		}
		for dec.More() {
			var cv json.RawMessage
			if err := dec.Decode(&cv); err != nil {
				return nil, err
			}
			child, err := readNode(cv)
			if err != nil {
				return nil, err
			}
			n.arr = append(n.arr, child)
		}
		return n, nil
	default:
		return &node{kind: 's', raw: append(json.RawMessage{}, trimmed...)}, nil
	}
}

func (n *node) writeCompact(buf *bytes.Buffer) {
	switch n.kind {
	case 'o':
		buf.WriteByte('{')
		for i, k := range n.keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			n.obj[k].writeCompact(buf)
		}
		buf.WriteByte('}')
	case 'a':
		buf.WriteByte('[')
		for i, c := range n.arr {
			if i > 0 {
				buf.WriteByte(',')
			}
			c.writeCompact(buf)
		}
		buf.WriteByte(']')
	default:
		buf.Write(n.raw)
	}
}

func encodeNode(n *node) ([]byte, error) {
	var compact bytes.Buffer
	n.writeCompact(&compact)
	var out bytes.Buffer
	if err := json.Indent(&out, compact.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func writersOf(root *node) []*node {
	if root.kind != 'o' {
		return nil
	}
	socket := root.obj["socket"]
	if socket == nil || socket.kind != 'o' {
		return nil
	}
	w := socket.obj["writers"]
	if w == nil || w.kind != 'a' {
		return nil
	}
	return w.arr
}

func findWriterNode(writers []*node, name string) *node {
	for _, w := range writers {
		if w.kind == 'o' {
			if wn := w.obj["writer"]; wn != nil && wn.kind == 's' {
				var s string
				if json.Unmarshal(wn.raw, &s) == nil && s == name {
					return w
				}
			}
		}
	}
	return nil
}

func operationsOf(w *node) map[string]int {
	out := map[string]int{}
	opts := w.obj["options"]
	if opts == nil || opts.kind != 'o' {
		return out
	}
	ops := opts.obj["operations"]
	if ops == nil || ops.kind != 'o' {
		return out
	}
	for _, k := range ops.keys {
		c := ops.obj[k]
		if c.kind != 's' {
			continue
		}
		var num json.Number
		if json.Unmarshal(c.raw, &num) == nil {
			if iv, err := num.Int64(); err == nil {
				out[k] = int(iv)
				continue
			}
		}
		var s string
		if json.Unmarshal(c.raw, &s) == nil {
			if iv, err := strconv.ParseUint(s, 0, 16); err == nil {
				out[k] = int(iv)
			}
		}
	}
	return out
}

// setOperations writes the writer's options.operations in YAML declaration
// order. Returns true if anything changed.
func setOperations(w *node, doc dispatcherDoc, expected map[string]int) bool {
	before := nodeBytes(w)
	opts := w.obj["options"]
	if opts == nil || opts.kind != 'o' {
		opts = &node{kind: 'o', obj: map[string]*node{}}
		w.keys = append(w.keys, "options")
		w.obj["options"] = opts
	}
	ops := &node{kind: 'o', obj: map[string]*node{}}
	for _, op := range doc.Operations {
		v, ok := expected[op.Key]
		if !ok {
			continue
		}
		ops.keys = append(ops.keys, op.Key)
		ops.obj[op.Key] = &node{kind: 's', raw: json.RawMessage(strconv.Itoa(v))}
	}
	if _, ok := opts.obj["operations"]; !ok {
		opts.keys = append(opts.keys, "operations")
	}
	opts.obj["operations"] = ops
	return !bytes.Equal(before, nodeBytes(w))
}

func nodeBytes(n *node) []byte {
	var b bytes.Buffer
	n.writeCompact(&b)
	return b.Bytes()
}
