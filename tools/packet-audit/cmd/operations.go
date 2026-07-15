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
	// Exactly one of Writer (clientbound) or Handler (serverbound) names the
	// socket entry this doc drives. Both live under socket.{writers,handlers}
	// with an identically-shaped options.operations table; the only structural
	// difference is the array + entry key ("writer" vs "handler").
	Writer  string `yaml:"writer"`
	Handler string `yaml:"handler"`
	FName   string `yaml:"fname"`
	Op      string `yaml:"op"`
	// Opcodes optionally supplies the per-version opCode (e.g. "0x14B").
	// When a template LACKS this entry entirely but the YAML provides the
	// version's opcode, `operations` generate ADDS the entry (opCode +
	// writer/handler + operations); `--check` flags its absence as missing.
	Opcodes    map[string]string `yaml:"opcodes"`
	Operations []struct {
		Key   string         `yaml:"key"`
		Modes map[string]int `yaml:"modes"`
	} `yaml:"operations"`
}

// arrayKey / entryKey / targetName resolve which socket array and entry key
// this doc drives (a writer under socket.writers, or a handler under
// socket.handlers).
func (d dispatcherDoc) arrayKey() string {
	if d.Handler != "" {
		return "handlers"
	}
	return "writers"
}

func (d dispatcherDoc) entryKey() string {
	if d.Handler != "" {
		return "handler"
	}
	return "writer"
}

func (d dispatcherDoc) targetName() string {
	if d.Handler != "" {
		return d.Handler
	}
	return d.Writer
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
		changed := false
		for _, doc := range docs {
			name := doc.targetName()
			expected := expectedTable(doc, vk)
			entries := entriesOf(root, doc.arrayKey())
			w := findEntryNode(entries, doc.entryKey(), name)
			if w == nil {
				if len(expected) == 0 {
					continue
				}
				oc, hasOC := doc.Opcodes[vk]
				if !hasOC {
					absent = append(absent, fmt.Sprintf("%s: %s %q not in template (cannot populate %d ops; add an opcodes entry to the YAML to wire it)", vk, doc.entryKey(), name, len(expected)))
					continue
				}
				if o.Check {
					missing = append(missing, fmt.Sprintf("%s %s: entry absent (should be wired at %s)", vk, name, oc))
					continue
				}
				if addEntry(root, doc, oc, expected) {
					changed = true
				}
				continue
			}
			if len(expected) == 0 {
				continue
			}
			got := operationsOf(w)
			for k, want := range expected {
				if gv, ok := got[k]; !ok {
					missing = append(missing, fmt.Sprintf("%s %s: key %q missing (want %d)", vk, name, k, want))
				} else if gv != want {
					drift = append(drift, fmt.Sprintf("%s %s: key %q is %d, want %d", vk, name, k, gv, want))
				}
			}
			for k := range got {
				if _, ok := expected[k]; !ok {
					extra = append(extra, fmt.Sprintf("%s %s: key %q in template but not in enumeration", vk, name, k))
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
		if d.Writer != "" || d.Handler != "" {
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
	kind  byte // 'o' object, 'a' array, 's' scalar
	keys  []string
	obj   map[string]*node
	arr   []*node
	raw   json.RawMessage // scalars: the value bytes; composites: original bytes (verbatim emit when clean)
	dirty bool            // this composite was modified and must be re-indented
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
	orig := append(json.RawMessage{}, trimmed...)
	switch trimmed[0] {
	case '{':
		n := &node{kind: 'o', obj: map[string]*node{}, raw: orig}
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
		n := &node{kind: 'a', raw: orig}
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
		return &node{kind: 's', raw: orig}, nil
	}
}

// subtreeDirty reports whether n or any descendant was modified.
func subtreeDirty(n *node) bool {
	if n.dirty {
		return true
	}
	for _, c := range n.obj {
		if subtreeDirty(c) {
			return true
		}
	}
	for _, c := range n.arr {
		if subtreeDirty(c) {
			return true
		}
	}
	return false
}

// emit writes n at the given indentation. A clean subtree (no modified
// descendant) is written from its original bytes VERBATIM, preserving the
// template's hand formatting (compact inline arrays/objects, \uXXXX escapes).
// Only nodes on a path to a modification are re-indented.
func (n *node) emit(buf *bytes.Buffer, indent string) {
	if n.kind == 's' || !subtreeDirty(n) {
		buf.Write(n.raw) // scalar value, or verbatim clean composite
		return
	}
	child := indent + "  "
	switch n.kind {
	case 'o':
		buf.WriteString("{\n")
		for i, k := range n.keys {
			buf.WriteString(child)
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteString(": ")
			n.obj[k].emit(buf, child)
			if i < len(n.keys)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString(indent + "}")
	case 'a':
		buf.WriteString("[\n")
		for i, c := range n.arr {
			buf.WriteString(child)
			c.emit(buf, child)
			if i < len(n.arr)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString(indent + "]")
	}
}

func encodeNode(n *node) ([]byte, error) {
	var out bytes.Buffer
	n.emit(&out, "")
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func entriesOf(root *node, arrayKey string) []*node {
	a := arrayNode(root, arrayKey)
	if a == nil {
		return nil
	}
	return a.arr
}

func findEntryNode(entries []*node, entryKey, name string) *node {
	for _, w := range entries {
		if w.kind == 'o' {
			if wn := w.obj[entryKey]; wn != nil && wn.kind == 's' {
				var s string
				if json.Unmarshal(wn.raw, &s) == nil && s == name {
					return w
				}
			}
		}
	}
	return nil
}

// writersArrayNode returns the socket.writers array node (for appending).
func arrayNode(root *node, arrayKey string) *node {
	if root.kind != 'o' {
		return nil
	}
	socket := root.obj["socket"]
	if socket == nil || socket.kind != 'o' {
		return nil
	}
	w := socket.obj[arrayKey]
	if w == nil || w.kind != 'a' {
		return nil
	}
	return w
}

// buildOperationsNode builds a fresh, dirty operations object node in YAML order.
func buildOperationsNode(doc dispatcherDoc, expected map[string]int) *node {
	ops := &node{kind: 'o', obj: map[string]*node{}, dirty: true}
	for _, op := range doc.Operations {
		v, ok := expected[op.Key]
		if !ok {
			continue
		}
		ops.keys = append(ops.keys, op.Key)
		ops.obj[op.Key] = &node{kind: 's', raw: json.RawMessage(strconv.Itoa(v))}
	}
	return ops
}

// addWriter appends a new writer entry {opCode, writer, options:{operations}} to
// socket.writers. Returns true on success.
func addEntry(root *node, doc dispatcherDoc, opcode string, expected map[string]int) bool {
	arr := arrayNode(root, doc.arrayKey())
	if arr == nil {
		return false
	}
	entryKey := doc.entryKey()
	ocBytes, _ := json.Marshal(opcode)
	wnBytes, _ := json.Marshal(doc.targetName())
	opts := &node{kind: 'o', obj: map[string]*node{}, dirty: true}
	opts.keys = []string{"operations"}
	opts.obj["operations"] = buildOperationsNode(doc, expected)
	w := &node{kind: 'o', dirty: true, obj: map[string]*node{
		"opCode":  {kind: 's', raw: ocBytes},
		entryKey:  {kind: 's', raw: wnBytes},
		"options": opts,
	}, keys: []string{"opCode", entryKey, "options"}}
	arr.arr = append(arr.arr, w)
	arr.dirty = true
	return true
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
	if _, ok := opts.obj["operations"]; !ok {
		opts.keys = append(opts.keys, "operations")
	}
	opts.obj["operations"] = buildOperationsNode(doc, expected)
	return !bytes.Equal(before, nodeBytes(w))
}

// nodeBytes is a deterministic compact serialization used only for change
// detection (order-sensitive; values from raw or recursively).
func nodeBytes(n *node) []byte {
	var b bytes.Buffer
	writeCanon(&b, n)
	return b.Bytes()
}

func writeCanon(buf *bytes.Buffer, n *node) {
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
			writeCanon(buf, n.obj[k])
		}
		buf.WriteByte('}')
	case 'a':
		buf.WriteByte('[')
		for i, c := range n.arr {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeCanon(buf, c)
		}
		buf.WriteByte(']')
	default:
		buf.Write(bytes.TrimSpace(n.raw))
	}
}
