package idasrc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf16"
)

// DispatchUpdate is one entry's confirmed dispatch selector plus a provenance note
// (e.g. "inferred-high-confidence (0.87) @0x.."). The note is written only when the
// entry has no existing note field (never clobbers a hand-authored note).
type DispatchUpdate struct {
	Dispatch []Selector
	Note     string
}

// WriteDispatch inserts a "dispatch" selector array into each named function,
// operating SURGICALLY on the raw JSON: the target function's object bytes are
// augmented in place and every other byte of the file — formatting, key order,
// and hand-authored fields the export structs do not model (region, size,
// note/_note, mixed indentation, Python-style escaping) — is preserved verbatim.
// A typed marshal round-trip would silently drop those unmodeled fields, so it is
// NOT used here. Unknown FNames are an error. An empty update map is a no-op (the
// file is left byte-for-byte unchanged — WriteDispatch never reformats).
func WriteDispatch(path string, updates map[string]DispatchUpdate) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}

	entries, err := orderedFunctionRaws(b)
	if err != nil {
		return err
	}
	known := map[string]string{}
	for _, e := range entries {
		known[e.key] = e.raw
	}
	for fname := range updates {
		if _, ok := known[fname]; !ok {
			return fmt.Errorf("idasrc: WriteDispatch: unknown FName %q", fname)
		}
	}

	indentUnit := detectIndent(b)
	text := string(b)
	// Walk entries in file order, advancing a cursor past each entry's object
	// bytes — so a target is spliced at ITS occurrence even when an earlier (or
	// later) function has a byte-identical value object. A plain
	// strings.Replace(.,.,1) would rewrite the first occurrence and corrupt the
	// wrong sibling; advancing the cursor past every entry disambiguates by
	// position.
	cursor := 0
	for _, e := range entries {
		rel := strings.Index(text[cursor:], e.raw)
		if rel < 0 {
			return fmt.Errorf("idasrc: WriteDispatch: object for %q not found at/after cursor", e.key)
		}
		at := cursor + rel
		if up, ok := updates[e.key]; ok {
			augmented, err := insertDispatchField(e.raw, up, indentUnit)
			if err != nil {
				return fmt.Errorf("idasrc: WriteDispatch %q: %w", e.key, err)
			}
			text = text[:at] + augmented + text[at+len(e.raw):]
			cursor = at + len(augmented)
		} else {
			cursor = at + len(e.raw)
		}
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

// CallSpec is one op + comment to prepend to an entry's calls array.
type CallSpec struct {
	Op      string
	Comment string
}

// PrependCall inserts a leading call (op + comment) at the FRONT of each named
// entry's "calls" array, surgically: the target object bytes are augmented in
// place and every other byte — formatting, key order, unmodeled fields — is
// preserved. Mirrors WriteDispatch's positional cursor walk so identical-body
// siblings are disambiguated by position. Unknown FNames are an error; an empty
// update map is a no-op. Each target must have a non-empty calls array.
func PrependCall(path string, updates map[string]CallSpec) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	entries, err := orderedFunctionRaws(b)
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, e := range entries {
		known[e.key] = true
	}
	for fname := range updates {
		if !known[fname] {
			return fmt.Errorf("idasrc: PrependCall: unknown FName %q", fname)
		}
	}
	text := string(b)
	cursor := 0
	for _, e := range entries {
		rel := strings.Index(text[cursor:], e.raw)
		if rel < 0 {
			return fmt.Errorf("idasrc: PrependCall: object for %q not found at/after cursor", e.key)
		}
		at := cursor + rel
		if spec, ok := updates[e.key]; ok {
			aug, err := prependCallToCalls(e.raw, spec)
			if err != nil {
				return fmt.Errorf("idasrc: PrependCall %q: %w", e.key, err)
			}
			text = text[:at] + aug + text[at+len(e.raw):]
			cursor = at + len(aug)
		} else {
			cursor = at + len(e.raw)
		}
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

// prependCallToCalls inserts a new call object as the first element of the raw
// object's "calls" array, matching the indentation of the existing first element.
// Requires a non-empty calls array (every remediation target has >=1 read).
func prependCallToCalls(raw string, spec CallSpec) (string, error) {
	ci := strings.Index(raw, "\"calls\":")
	if ci < 0 {
		return "", fmt.Errorf("no \"calls\" field")
	}
	open := strings.IndexByte(raw[ci:], '[')
	if open < 0 {
		return "", fmt.Errorf("no calls array")
	}
	open += ci
	fb := strings.IndexByte(raw[open+1:], '{')
	cb := strings.IndexByte(raw[open+1:], ']')
	if fb < 0 || (cb >= 0 && cb < fb) {
		return "", fmt.Errorf("calls array is empty")
	}
	fb += open + 1

	lineStart := strings.LastIndexByte(raw[:fb], '\n') + 1
	elemIndent := raw[lineStart:fb]
	if strings.TrimLeft(elemIndent, " \t") != "" {
		// The first element's `{` is not at the start of its line (a compact
		// `"calls": [{` layout). Refuse rather than emit invalid JSON.
		return "", fmt.Errorf("compact calls layout unsupported (first element not at line start)")
	}
	innerNL := strings.IndexByte(raw[fb:], '\n')
	if innerNL < 0 {
		return "", fmt.Errorf("malformed first call element")
	}
	fieldLineStart := fb + innerNL + 1
	fieldNonWS := fieldLineStart
	for fieldNonWS < len(raw) && (raw[fieldNonWS] == ' ' || raw[fieldNonWS] == '\t') {
		fieldNonWS++
	}
	fieldIndent := raw[fieldLineStart:fieldNonWS]

	opJSON, _ := json.Marshal(spec.Op)
	comJSON, _ := json.Marshal(spec.Comment)
	var nb strings.Builder
	nb.WriteString(elemIndent + "{\n")
	nb.WriteString(fieldIndent + "\"op\": " + string(matchPythonEscaping(opJSON)) + ",\n")
	nb.WriteString(fieldIndent + "\"comment\": " + string(matchPythonEscaping(comJSON)) + "\n")
	nb.WriteString(elemIndent + "},\n")

	return raw[:lineStart] + nb.String() + raw[lineStart:], nil
}

type functionRaw struct {
	key string
	raw string // exact object bytes, "{...}", as they appear in the file
}

// orderedFunctionRaws returns each function's key and exact object bytes in file
// order, by streaming the "functions" object with a json.Decoder (json.RawMessage
// captures the value's bytes verbatim).
func orderedFunctionRaws(b []byte) ([]functionRaw, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, err
	}
	fnsRaw, ok := top["functions"]
	if !ok {
		return nil, fmt.Errorf("idasrc: WriteDispatch: no \"functions\" object")
	}
	dec := json.NewDecoder(bytes.NewReader(fnsRaw))
	if _, err := dec.Token(); err != nil { // opening '{'
		return nil, err
	}
	var out []functionRaw
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("idasrc: WriteDispatch: expected string key")
		}
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
		out = append(out, functionRaw{key: key, raw: string(val)})
	}
	return out, nil
}

// insertDispatchField returns the function object text with a "dispatch" field
// (and, when the entry has no existing note, a provenance "notes" field) inserted
// immediately before the "calls" field, matching the object's field indentation.
func insertDispatchField(raw string, up DispatchUpdate, indentUnit string) (string, error) {
	fi := firstFieldIndent(raw)
	selBytes, err := json.MarshalIndent(up.Dispatch, fi, indentUnit)
	if err != nil {
		return "", err
	}
	var ins strings.Builder
	ins.WriteString(fi + "\"dispatch\": " + string(matchPythonEscaping(selBytes)) + ",\n")
	if up.Note != "" && !hasNoteKey(raw) {
		nb, _ := json.Marshal(up.Note)
		ins.WriteString(fi + "\"notes\": " + string(matchPythonEscaping(nb)) + ",\n")
	}

	marker := "\n" + fi + "\"calls\""
	idx := strings.Index(raw, marker)
	if idx < 0 {
		return "", fmt.Errorf("no \"calls\" field to anchor dispatch insertion")
	}
	at := idx + 1 // start of the "calls" line
	return raw[:at] + ins.String() + raw[at:], nil
}

// firstFieldIndent returns the leading whitespace of the first field line inside
// an object's raw bytes (the indentation to use for an inserted field).
func firstFieldIndent(raw string) string {
	nl := strings.IndexByte(raw, '\n')
	if nl < 0 {
		return "  "
	}
	for _, line := range strings.Split(raw[nl+1:], "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		return line[:len(line)-len(trimmed)]
	}
	return "  "
}

// hasNoteKey reports whether the object already carries any note field variant.
func hasNoteKey(raw string) bool {
	return strings.Contains(raw, "\"notes\":") ||
		strings.Contains(raw, "\"note\":") ||
		strings.Contains(raw, "\"_note\":")
}

// detectIndent returns the single-level indentation unit used by the file (the
// leading whitespace of the first indented line). Defaults to two spaces.
func detectIndent(b []byte) string {
	for _, line := range strings.Split(string(b), "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || trimmed == "{" {
			continue
		}
		if indent := line[:len(line)-len(trimmed)]; indent != "" {
			return indent
		}
	}
	return "  "
}

// matchPythonEscaping rewrites Go's json escaping to match the baselines' original
// Python (json.dumps ensure_ascii=True) form: HTML metacharacters stay literal
// (&, <, >) and every non-ASCII rune is escaped to lowercase \uXXXX (surrogate
// pairs above the BMP). Applied to inserted field text so a selector/notes value
// matches the surrounding style.
func matchPythonEscaping(b []byte) []byte {
	for _, c := range []rune{'&', '<', '>'} {
		esc := []byte(fmt.Sprintf(`\u%04x`, c))
		b = bytes.ReplaceAll(b, esc, []byte(string(c)))
	}
	var out strings.Builder
	out.Grow(len(b))
	for _, r := range string(b) {
		if r < 128 {
			out.WriteRune(r)
			continue
		}
		if r > 0xFFFF {
			r1, r2 := utf16.EncodeRune(r)
			fmt.Fprintf(&out, `\u%04x\u%04x`, r1, r2)
		} else {
			fmt.Fprintf(&out, `\u%04x`, r)
		}
	}
	return []byte(out.String())
}
