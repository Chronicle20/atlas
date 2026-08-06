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

var knownEntryKeys = map[string]bool{
	"opCode": true, "validator": true, "handler": true, "writer": true,
	"fname": true, "options": true, "services": true,
}

// seedEntryMeta is the ONLY thing the generator models about a socket entry:
// enough to resolve an fname (opcode, implementation name, direction) plus
// its EXACT original raw bytes. Nothing else about the entry - key order,
// the "services" array's compact-vs-expanded style, whitespace - is ever
// re-derived from a struct: writeSeedTemplate patches Raw directly, so any
// field the generator does not touch is physically incapable of changing.
//
// This design exists because the naive approach (decode into a struct,
// json.MarshalIndent the whole document back out) does NOT round-trip these
// files byte-for-byte: encoding/json's Indent pass uniformly reformats the
// ENTIRE byte stream it is given, including the parts held as json.RawMessage
// for "verbatim" carry-through. Measured against the real seed templates
// (services/atlas-configurations/seed-data/templates/), that breaks in three
// independent ways: (1) every "services" array is hand-formatted on one
// compact line ("services": ["login"] or ["login","channel"]) but Indent
// always expands one element per line; (2) entries that carry both "services"
// and "options" always order "services" first in the source, but Indent
// applies whatever fixed struct-field order the Go type declares; (3) two of
// the eleven files (gms_12_1, gms_92_1) format the FIRST element of several
// arrays - including socket.handlers/writers themselves - with the opening
// brace inlined onto the array's own line ("handlers": [{ ... rather than
// "handlers": [\n  { ...), which Indent also cannot reproduce, and which
// recurs inside sections (worlds, characters, options.types) the generator
// never intends to touch at all. Patching each of those with a targeted
// regex is unbounded whack-a-mole; splicing bytes in place sidesteps all
// three (and any future one) by construction.
type seedEntryMeta struct {
	OpCode        string
	Name          string // the handler or writer implementation name, whichever is present
	IsHandler     bool   // true => anchor key is "handler"; false => "writer"
	ExistingFName string // "" if this entry has no fname key yet
	Raw           []byte // exact original bytes of this entry's JSON object
	NewFName      string // set by resolution; "" means nothing resolved this run
}

// effective is the fname this entry will carry after this run: the freshly
// resolved value if one was found, else whatever was already on disk.
func (e *seedEntryMeta) effective() string {
	if e.NewFName != "" {
		return e.NewFName
	}
	return e.ExistingFName
}

// seedGroup is one of socket.handlers / socket.writers: its own exact
// original raw bytes (the array literal, "[" through "]"), plus the ordered
// per-entry metadata used to relocate and patch each entry within those bytes.
type seedGroup struct {
	Raw     []byte
	Entries []*seedEntryMeta
}

type seedTemplate struct {
	Stem      string
	Path      string
	Raw       []byte // exact original full-file bytes, including trailing newline (or lack of one)
	SocketRaw []byte // exact original bytes of the "socket": {...} value
	Handlers  seedGroup
	Writers   seedGroup
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
		st, err := loadSeedTemplate(v.Template, path)
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
		entries := len(st.Handlers.Entries) + len(st.Writers.Entries)
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
// the tool does not model. Guard 1 of the two fidelity guards. It also captures
// the exact original raw bytes at every level (file, socket, each group, each
// entry) that writeSeedTemplate later splices into - Guard 2.
func loadSeedTemplate(stem, path string) (*seedTemplate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, err
	}
	for k := range top {
		if !knownTopLevelKeys[k] {
			return nil, fmt.Errorf("unmodelled top-level key %q - refusing to rewrite, "+
				"because marshalling through the known-key model would silently drop it", k)
		}
	}

	socketRaw, ok := top["socket"]
	if !ok {
		return nil, fmt.Errorf("%s: missing required top-level key \"socket\"", path)
	}
	var sock map[string]json.RawMessage
	if err := json.Unmarshal(socketRaw, &sock); err != nil {
		return nil, fmt.Errorf("parse socket: %w", err)
	}

	handlers, err := loadSeedGroup(sock, "handlers")
	if err != nil {
		return nil, err
	}
	writers, err := loadSeedGroup(sock, "writers")
	if err != nil {
		return nil, err
	}

	return &seedTemplate{
		Stem:      stem,
		Path:      path,
		Raw:       raw,
		SocketRaw: []byte(socketRaw),
		Handlers:  handlers,
		Writers:   writers,
	}, nil
}

// loadSeedGroup decodes one of socket.handlers / socket.writers: validates
// every entry's key set (Guard 1) and captures each entry's exact original
// raw bytes for later splicing (Guard 2). A group absent from socket entirely
// (as in a minimal test fixture) is returned zero-valued, not an error.
func loadSeedGroup(sock map[string]json.RawMessage, name string) (seedGroup, error) {
	raw, ok := sock[name]
	if !ok {
		return seedGroup{}, nil
	}
	var entryRaws []json.RawMessage
	if err := json.Unmarshal(raw, &entryRaws); err != nil {
		return seedGroup{}, fmt.Errorf("parse socket.%s: %w", name, err)
	}

	g := seedGroup{Raw: []byte(raw)}
	for i, er := range entryRaws {
		var keys map[string]json.RawMessage
		if err := json.Unmarshal(er, &keys); err != nil {
			return seedGroup{}, fmt.Errorf("parse socket.%s[%d]: %w", name, i, err)
		}
		for k := range keys {
			if !knownEntryKeys[k] {
				return seedGroup{}, fmt.Errorf("unmodelled socket-entry key %q at socket.%s[%d] - refusing to rewrite", k, name, i)
			}
		}

		var typed struct {
			OpCode  string `json:"opCode"`
			Handler string `json:"handler"`
			Writer  string `json:"writer"`
			FName   string `json:"fname"`
		}
		if err := json.Unmarshal(er, &typed); err != nil {
			return seedGroup{}, fmt.Errorf("parse socket.%s[%d]: %w", name, i, err)
		}
		isHandler := typed.Handler != ""
		implName := typed.Writer
		if isHandler {
			implName = typed.Handler
		}
		g.Entries = append(g.Entries, &seedEntryMeta{
			OpCode:        typed.OpCode,
			Name:          implName,
			IsHandler:     isHandler,
			ExistingFName: typed.FName,
			Raw:           []byte(er),
		})
	}
	return g, nil
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
	apply := func(entries []*seedEntryMeta, dir opregistry.Direction, kind string) {
		for _, e := range entries {
			code, ok := parseSeedOpCode(e.OpCode)
			if !ok {
				continue
			}
			fn := byOp[string(dir)+"|"+strconv.Itoa(code)]
			if fn == "" {
				continue
			}
			e.NewFName = fn
			if e.Name != "" {
				got[kind+"|"+e.Name] = fn
			}
		}
	}
	apply(st.Handlers.Entries, opregistry.DirServerbound, "handler")
	apply(st.Writers.Entries, opregistry.DirClientbound, "writer")
	return got
}

func applyFallback(st *seedTemplate, fallback []string, resolved map[string]map[string]string) {
	apply := func(entries []*seedEntryMeta, kind string) {
		for _, e := range entries {
			if e.Name == "" {
				continue
			}
			for _, src := range fallback {
				if fn := resolved[src][kind+"|"+e.Name]; fn != "" {
					e.NewFName = fn
					break
				}
			}
		}
	}
	apply(st.Handlers.Entries, "handler")
	apply(st.Writers.Entries, "writer")
}

func countResolved(st *seedTemplate) int {
	n := 0
	count := func(entries []*seedEntryMeta) {
		for _, e := range entries {
			if e.effective() != "" {
				n++
			}
		}
	}
	count(st.Handlers.Entries)
	count(st.Writers.Entries)
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

// writeSeedTemplate patches the resolved fname values into the ORIGINAL file
// bytes by byte-level splicing, bottom-up: entry -> group array -> socket ->
// file. Every byte the generator did not need to change - including a whole
// file where nothing resolved - is copied verbatim from st.Raw, never passed
// through json.Marshal. This is what makes Guard 2 (verbatim carry-through)
// actually hold against these files' real, non-uniform formatting instead of
// only holding for inputs that happen to already match encoding/json's output
// style.
func writeSeedTemplate(st *seedTemplate) error {
	handlersRaw, handlersChanged, err := spliceGroup(st.Handlers)
	if err != nil {
		return fmt.Errorf("%s: splice handlers: %w", st.Stem, err)
	}
	writersRaw, writersChanged, err := spliceGroup(st.Writers)
	if err != nil {
		return fmt.Errorf("%s: splice writers: %w", st.Stem, err)
	}
	if !handlersChanged && !writersChanged {
		// Every entry already carries its resolved fname (or nothing resolved
		// at all) - the file on disk already matches the desired state.
		return nil
	}

	socketRaw := st.SocketRaw
	if handlersChanged {
		socketRaw, err = spliceOnce(socketRaw, st.Handlers.Raw, handlersRaw)
		if err != nil {
			return fmt.Errorf("%s: relocate handlers within socket: %w", st.Stem, err)
		}
	}
	if writersChanged {
		socketRaw, err = spliceOnce(socketRaw, st.Writers.Raw, writersRaw)
		if err != nil {
			return fmt.Errorf("%s: relocate writers within socket: %w", st.Stem, err)
		}
	}

	fileRaw, err := spliceOnce(st.Raw, st.SocketRaw, socketRaw)
	if err != nil {
		return fmt.Errorf("%s: relocate socket within file: %w", st.Stem, err)
	}
	return os.WriteFile(st.Path, fileRaw, 0o644)
}

// spliceGroup rebuilds one group's array-literal bytes by walking its entries
// in order, copying the untouched bytes between them (whitespace, commas, the
// enclosing brackets) verbatim, and substituting each entry's own bytes with
// spliceEntry's result. Because every copied span comes straight from g.Raw,
// a group where no entry changed reproduces g.Raw byte-for-byte by
// construction - there is no formatting step left that could disturb it.
func spliceGroup(g seedGroup) ([]byte, bool, error) {
	out := make([]byte, 0, len(g.Raw))
	pos := 0
	changed := false
	for _, e := range g.Entries {
		idx := bytes.Index(g.Raw[pos:], e.Raw)
		if idx < 0 {
			return nil, false, fmt.Errorf("could not relocate entry %q in its array - internal offset bug", e.Name)
		}
		idx += pos
		out = append(out, g.Raw[pos:idx]...)
		entryRaw, entryChanged, err := spliceEntry(e)
		if err != nil {
			return nil, false, err
		}
		out = append(out, entryRaw...)
		changed = changed || entryChanged
		pos = idx + len(e.Raw)
	}
	out = append(out, g.Raw[pos:]...)
	return out, changed, nil
}

// spliceEntry returns this entry's raw bytes with its resolved fname applied:
// unchanged if nothing resolved or the resolved value already matches what is
// on disk; a targeted in-place value replacement if the entry already carries
// a DIFFERENT fname (a re-run after the registry moved); otherwise a new
// "fname" field spliced in immediately after the entry's "handler"/"writer"
// key, matching that key's own indentation, so a fresh insertion reads as a
// natural continuation of the existing entry rather than a reformat of it.
func spliceEntry(e *seedEntryMeta) ([]byte, bool, error) {
	if e.NewFName == "" || e.NewFName == e.ExistingFName {
		return e.Raw, false, nil
	}
	if e.ExistingFName != "" {
		oldField := []byte(`"fname": "` + e.ExistingFName + `"`)
		newField := []byte(`"fname": "` + e.NewFName + `"`)
		if bytes.Count(e.Raw, oldField) != 1 {
			return nil, false, fmt.Errorf("entry %q: could not uniquely locate existing fname %q to update", e.Name, e.ExistingFName)
		}
		return bytes.Replace(e.Raw, oldField, newField, 1), true, nil
	}

	anchorKey := "writer"
	if e.IsHandler {
		anchorKey = "handler"
	}
	anchor := []byte(`"` + anchorKey + `": "` + e.Name + `"`)
	if bytes.Count(e.Raw, anchor) != 1 {
		return nil, false, fmt.Errorf("entry %q: could not uniquely locate %q key to anchor fname insertion", e.Name, anchorKey)
	}
	loc := bytes.Index(e.Raw, anchor)
	anchorEnd := loc + len(anchor)

	lineStart := bytes.LastIndexByte(e.Raw[:loc], '\n') + 1
	indent := e.Raw[lineStart:loc]

	var insertion bytes.Buffer
	insertion.WriteString(",\n")
	insertion.Write(indent)
	insertion.WriteString(`"fname": "` + e.NewFName + `"`)

	out := make([]byte, 0, len(e.Raw)+insertion.Len())
	out = append(out, e.Raw[:anchorEnd]...)
	out = append(out, insertion.Bytes()...)
	out = append(out, e.Raw[anchorEnd:]...)
	return out, true, nil
}

// spliceOnce replaces the single occurrence of old within parent with new,
// failing loudly (rather than guessing) if old is not found in parent exactly
// once - the same "surprise is a hard stop" discipline as the unknown-key
// guards, applied to the byte-splicing machinery itself.
func spliceOnce(parent, old, new []byte) ([]byte, error) {
	if bytes.Count(parent, old) != 1 {
		return nil, fmt.Errorf("expected byte span is not uniquely present in its parent")
	}
	idx := bytes.Index(parent, old)
	out := make([]byte, 0, len(parent)-len(old)+len(new))
	out = append(out, parent[:idx]...)
	out = append(out, new...)
	out = append(out, parent[idx+len(old):]...)
	return out, nil
}
