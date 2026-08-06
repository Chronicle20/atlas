package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// seedFNameVersions maps a seed template's file stem to the registry file stem
// it joins against, and to the ordered fallback templates it borrows from when
// it has no registry of its own. Fallback resolves by IMPLEMENTATION NAME,
// which is valid because the implementation name is the definition identity
// within a direction - the opcode is version-specific and cannot be joined on.
var seedFNameVersions = []struct {
	Template string   // template file stem, e.g. "gms_83_1"
	Registry string   // registry file stem, e.g. "gms_v83"; empty means none
	Fallback []string // ordered template stems to borrow from; first hit wins
}{
	{Template: "gms_12_1", Registry: "", Fallback: []string{"gms_48_1", "gms_61_1"}},
	{Template: "gms_48_1", Registry: "gms_v48"},
	{Template: "gms_61_1", Registry: "gms_v61"},
	{Template: "gms_72_1", Registry: "gms_v72"},
	{Template: "gms_79_1", Registry: "gms_v79"},
	{Template: "gms_83_1", Registry: "gms_v83"},
	{Template: "gms_84_1", Registry: "gms_v84"},
	{Template: "gms_87_1", Registry: "gms_v87"},
	{Template: "gms_92_1", Registry: "", Fallback: []string{"gms_87_1", "gms_95_1"}},
	{Template: "gms_95_1", Registry: "gms_v95"},
	{Template: "jms_185_1", Registry: "jms_v185"},
}

var knownTopLevelKeys = map[string]bool{
	"region": true, "majorVersion": true, "minorVersion": true, "usesPin": true,
	"socket": true, "characters": true, "npcs": true, "worlds": true, "cashShop": true,
}

var knownSocketKeys = map[string]bool{
	"handlers": true, "writers": true,
}

var knownEntryKeys = map[string]bool{
	"opCode": true, "validator": true, "handler": true, "writer": true,
	"fname": true, "options": true, "services": true,
}

// seedDoc is the write model. Everything the generator does not touch is held
// as a verbatim json.RawMessage, so re-marshalling cannot alter its semantic
// content. Field order here is the output key order.
//
// NOTE (task-6 fix-up): this writes via json.MarshalIndent, which normalizes
// formatting across the WHOLE document - including RawMessage content - to a
// uniform 2-space-indent, one-array-element-per-line style. It does not
// reproduce the real seed templates' existing formatting quirks (every
// "services" array hand-compacted onto one line; entries with both "services"
// and "options" ordering services first; two files inlining an array's first
// "{" onto the opening "[" line). That was measured and is a deliberate,
// accepted tradeoff: the simpler, more robust generator over a minimal diff.
// The first --write run against the real templates (Task 7) will therefore
// produce a large, mostly-cosmetic diff; every run after that is a no-op
// (MarshalIndent's output is deterministic and idempotent - see
// TestSeedFName_ReRunIsIdempotent).
type seedDoc struct {
	Region       json.RawMessage `json:"region"`
	MajorVersion json.RawMessage `json:"majorVersion"`
	MinorVersion json.RawMessage `json:"minorVersion"`
	UsesPin      json.RawMessage `json:"usesPin"`
	Socket       seedSocket      `json:"socket"`
	Characters   json.RawMessage `json:"characters,omitempty"`
	NPCs         json.RawMessage `json:"npcs,omitempty"`
	Worlds       json.RawMessage `json:"worlds,omitempty"`
	CashShop     json.RawMessage `json:"cashShop,omitempty"`
}

type seedSocket struct {
	Handlers []seedEntry `json:"handlers"`
	Writers  []seedEntry `json:"writers"`
}

// seedEntry is the ONLY fully-modelled structure. loadSeedTemplate's
// unknown-key check is what makes that safe: a socket entry carrying a key
// outside knownEntryKeys stops the run rather than silently losing the key.
type seedEntry struct {
	OpCode    string          `json:"opCode"`
	Validator string          `json:"validator,omitempty"`
	Handler   string          `json:"handler,omitempty"`
	Writer    string          `json:"writer,omitempty"`
	FName     string          `json:"fname,omitempty"`
	Options   json.RawMessage `json:"options,omitempty"`
	Services  json.RawMessage `json:"services,omitempty"`
}

// Name returns the implementation name, whichever collection this entry is in.
func (e seedEntry) Name() string {
	if e.Handler != "" {
		return e.Handler
	}
	return e.Writer
}

type seedTemplate struct {
	Path string
	Doc  seedDoc
}

func runSeedFName(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("seed-fname", flag.ContinueOnError)
	fs.SetOutput(stderr)
	write := fs.Bool("write", false, "write the resolved fname values back to the template files")
	registryDir := fs.String("registry-dir", filepath.Join("docs", "packets", "registry"),
		"directory holding <version>.yaml op registries")
	templateDir := fs.String("template-dir", filepath.Join("services", "atlas-configurations", "seed-data", "templates"),
		"directory holding template_<version>.json seed files")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	// Load every template that is present. A version listed above but absent
	// from disk is skipped, which is what lets the tests drive a subset.
	templates := make(map[string]*seedTemplate)
	var order []string
	for _, v := range seedFNameVersions {
		path := filepath.Join(*templateDir, "template_"+v.Template+".json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		st, err := loadSeedTemplate(path)
		if err != nil {
			fmt.Fprintf(stderr, "FAIL %s: %v\n", filepath.Base(path), err)
			return 1
		}
		templates[v.Template] = st
		order = append(order, v.Template)
	}

	// Pass 1 - direct (direction, opcode) join for every version with a registry.
	// resolved[templateStem]["handler|LoginHandle"] = fname, which pass 2 borrows.
	resolved := make(map[string]map[string]string)
	for _, v := range seedFNameVersions {
		st, ok := templates[v.Template]
		if !ok || v.Registry == "" {
			continue
		}
		regPath := filepath.Join(*registryDir, v.Registry+".yaml")
		vf, err := opregistry.LoadVersion(regPath)
		if err != nil {
			fmt.Fprintf(stderr, "FAIL %s: %v\n", v.Registry, err)
			return 1
		}
		resolved[v.Template] = applyDirect(st, indexRegistryByOpcode(vf, v.Registry, stderr))
	}

	// Pass 2 - versions with no registry borrow by implementation name.
	for _, v := range seedFNameVersions {
		st, ok := templates[v.Template]
		if !ok || v.Registry != "" {
			continue
		}
		applyFallback(st, v.Fallback, resolved)
	}

	totalEntries, totalResolved := 0, 0
	for _, stem := range order {
		st := templates[stem]
		entries := len(st.Doc.Socket.Handlers) + len(st.Doc.Socket.Writers)
		got := countResolved(st)
		totalEntries += entries
		totalResolved += got
		fmt.Printf("%-12s %4d / %4d resolved\n", stem, got, entries)
	}
	fmt.Printf("%-12s %4d / %4d resolved\n", "TOTAL", totalResolved, totalEntries)

	if !*write {
		fmt.Println("(dry run - pass --write to update the template files)")
		return 0
	}
	for _, stem := range order {
		if err := writeSeedTemplate(templates[stem]); err != nil {
			fmt.Fprintf(stderr, "FAIL writing %s: %v\n", stem, err)
			return 1
		}
	}
	return 0
}

// loadSeedTemplate decodes a template and refuses it if it carries any JSON key
// the write model does not represent. Guard 1 of the two fidelity guards - the
// primary protection now that writeSeedTemplate normalizes formatting across
// the whole file: under a full reformat you cannot eyeball what got dropped,
// so a surprise key must still be a loud, non-zero-exit stop.
func loadSeedTemplate(path string) (*seedTemplate, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, err
	}
	for k := range top {
		if !knownTopLevelKeys[k] {
			return nil, fmt.Errorf("unmodelled top-level key %q - refusing to rewrite, "+
				"because marshalling through the known-key model would silently drop it", k)
		}
	}

	if raw, ok := top["socket"]; ok {
		var sock map[string]json.RawMessage
		if err := json.Unmarshal(raw, &sock); err != nil {
			return nil, fmt.Errorf("parse socket: %w", err)
		}
		for k := range sock {
			if !knownSocketKeys[k] {
				return nil, fmt.Errorf("unmodelled socket key %q - refusing to rewrite, "+
					"because marshalling through the known-key model would silently drop it", k)
			}
		}
		for _, group := range []string{"handlers", "writers"} {
			gr, ok := sock[group]
			if !ok {
				continue
			}
			var entries []map[string]json.RawMessage
			if err := json.Unmarshal(gr, &entries); err != nil {
				return nil, fmt.Errorf("parse socket.%s: %w", group, err)
			}
			for i, e := range entries {
				for k := range e {
					if !knownEntryKeys[k] {
						return nil, fmt.Errorf("unmodelled socket-entry key %q at socket.%s[%d] - refusing to rewrite", k, group, i)
					}
				}
			}
		}
	}

	var doc seedDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	return &seedTemplate{Path: path, Doc: doc}, nil
}

// indexRegistryByOpcode builds "direction|opcode" -> fname, applying the
// lexicographically-first-op tie-break where one opcode carries several
// distinct fnames, and logging every such choice.
func indexRegistryByOpcode(vf *opregistry.VersionFile, regStem string, stderr io.Writer) map[string]string {
	type cand struct{ op, fname string }
	groups := make(map[string][]cand)
	for _, e := range vf.Entries {
		fn := strings.TrimSpace(e.FName)
		if fn == "" {
			continue
		}
		k := string(e.Direction) + "|" + strconv.Itoa(e.Opcode)
		groups[k] = append(groups[k], cand{op: e.Op, fname: fn})
	}

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(map[string]string, len(groups))
	for _, k := range keys {
		cs := groups[k]
		sort.Slice(cs, func(i, j int) bool { return cs[i].op < cs[j].op })
		distinct := map[string]bool{}
		for _, c := range cs {
			distinct[c.fname] = true
		}
		if len(distinct) > 1 {
			names := make([]string, 0, len(cs))
			for _, c := range cs {
				names = append(names, c.op+"="+c.fname)
			}
			fmt.Fprintf(stderr, "ambiguous %s %s: %s - picking %s (lexicographically first op)\n",
				regStem, k, strings.Join(names, ", "), cs[0].op)
		}
		out[k] = cs[0].fname
	}
	return out
}

func applyDirect(st *seedTemplate, byOp map[string]string) map[string]string {
	got := make(map[string]string)
	apply := func(entries []seedEntry, dir opregistry.Direction, kind string) {
		for i := range entries {
			code, ok := parseSeedOpCode(entries[i].OpCode)
			if !ok {
				continue
			}
			fn := byOp[string(dir)+"|"+strconv.Itoa(code)]
			if fn == "" {
				continue
			}
			entries[i].FName = fn
			if n := entries[i].Name(); n != "" {
				got[kind+"|"+n] = fn
			}
		}
	}
	apply(st.Doc.Socket.Handlers, opregistry.DirServerbound, "handler")
	apply(st.Doc.Socket.Writers, opregistry.DirClientbound, "writer")
	return got
}

func applyFallback(st *seedTemplate, fallback []string, resolved map[string]map[string]string) {
	apply := func(entries []seedEntry, kind string) {
		for i := range entries {
			n := entries[i].Name()
			if n == "" {
				continue
			}
			for _, src := range fallback {
				if fn := resolved[src][kind+"|"+n]; fn != "" {
					entries[i].FName = fn
					break
				}
			}
		}
	}
	apply(st.Doc.Socket.Handlers, "handler")
	apply(st.Doc.Socket.Writers, "writer")
}

func countResolved(st *seedTemplate) int {
	n := 0
	for _, e := range st.Doc.Socket.Handlers {
		if e.FName != "" {
			n++
		}
	}
	for _, e := range st.Doc.Socket.Writers {
		if e.FName != "" {
			n++
		}
	}
	return n
}

func parseSeedOpCode(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 3 || s[0] != '0' || (s[1] != 'x' && s[1] != 'X') {
		return 0, false
	}
	n, err := strconv.ParseInt(s[2:], 16, 32)
	if err != nil {
		return 0, false
	}
	return int(n), true
}

// writeSeedTemplate re-encodes the document with two-space indentation and a
// trailing newline. Guard 2: every value the generator does not touch is a
// json.RawMessage, so its semantic content round-trips unchanged even though
// MarshalIndent normalizes the surrounding formatting (see the seedDoc
// doc-comment).
func writeSeedTemplate(st *seedTemplate) error {
	b, err := json.MarshalIndent(st.Doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(st.Path, append(b, '\n'), 0o644)
}
