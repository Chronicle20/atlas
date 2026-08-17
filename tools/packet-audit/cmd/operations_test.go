package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
)

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// writeAllTemplates writes body to every version's template filename under
// dir (via matrix.VersionKeys / matrix.TemplatePath) so operationsRun finds a
// file for each version in its loop.
func writeAllTemplates(t *testing.T, dir, body string) {
	t.Helper()
	for _, vk := range matrix.VersionKeys {
		mustWrite(t, filepath.Join(dir, filepath.Base(matrix.TemplatePath(vk))), body)
	}
}

func TestNodeRoundTripPreservesOrder(t *testing.T) {
	src := []byte(`{
  "region": "GMS",
  "socket": {
    "writers": [
      {
        "opCode": "0x180",
        "writer": "CashShopOperation"
      }
    ]
  }
}
`)
	n, err := parseNode(src)
	if err != nil {
		t.Fatal(err)
	}
	out, err := encodeNode(n)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, src) {
		t.Errorf("round-trip changed bytes:\n got: %s\nwant: %s", out, src)
	}
}

func TestSetOperationsInjectsInYAMLOrder(t *testing.T) {
	src := []byte(`{
  "socket": {
    "writers": [
      {
        "opCode": "0x180",
        "writer": "CashShopOperation"
      }
    ]
  }
}
`)
	n, _ := parseNode(src)
	doc := dispatcherDoc{Writer: "CashShopOperation", Operations: []opEntry{
		{Key: "PURCHASE_SUCCESS", Modes: map[string]int{"gms_v95": 100}},
		{Key: "LOAD_INVENTORY_SUCCESS", Modes: map[string]int{"gms_v95": 88}},
	}}
	w := findEntryNode(entriesOf(n, "writers"), "writer", "CashShopOperation")
	if w == nil {
		t.Fatal("writer not found")
	}
	if !setTable(w, "operations", doc.Operations, expectedFor(doc.Operations, "gms_v95")) {
		t.Fatal("setTable reported no change")
	}
	got := tableOf(w, "operations")
	if got["PURCHASE_SUCCESS"] != 100 || got["LOAD_INVENTORY_SUCCESS"] != 88 {
		t.Errorf("operations wrong: %v", got)
	}
	// Re-running with the same expected table must be a no-op (idempotent).
	if setTable(w, "operations", doc.Operations, expectedFor(doc.Operations, "gms_v95")) {
		t.Error("second setTable should be idempotent")
	}
	out, _ := encodeNode(n)
	// Insertion order from the YAML must be preserved in the emitted JSON.
	pi := bytes.Index(out, []byte("PURCHASE_SUCCESS"))
	li := bytes.Index(out, []byte("LOAD_INVENTORY_SUCCESS"))
	if pi < 0 || li < 0 || pi > li {
		t.Errorf("operations not in YAML order:\n%s", out)
	}
}

func TestAddWriterAppendsEntry(t *testing.T) {
	src := []byte(`{
  "socket": {
    "writers": [
      {
        "opCode": "0x01",
        "writer": "Existing"
      }
    ]
  }
}
`)
	n, _ := parseNode(src)
	doc := dispatcherDoc{Writer: "NewOp", Operations: []opEntry{
		{Key: "A", Modes: map[string]int{"gms_v87": 2}},
	}}
	if !addEntry(n, doc, "gms_v87", "0x14B") {
		t.Fatal("addEntry failed")
	}
	w := findEntryNode(entriesOf(n, "writers"), "writer", "NewOp")
	if w == nil {
		t.Fatal("new writer not found after add")
	}
	if got := tableOf(w, "operations"); got["A"] != 2 {
		t.Errorf("new writer operations wrong: %v", got)
	}
	out, _ := encodeNode(n)
	if !bytes.Contains(out, []byte(`"0x14B"`)) || !bytes.Contains(out, []byte(`"NewOp"`)) {
		t.Errorf("encoded output missing new writer:\n%s", out)
	}
	// Existing writer preserved verbatim.
	if !bytes.Contains(out, []byte(`"writer": "Existing"`)) {
		t.Errorf("existing writer lost:\n%s", out)
	}
}

func TestExpectedTableOmitsAbsentVersion(t *testing.T) {
	entries := []opEntry{
		{Key: "A", Modes: map[string]int{"gms_v83": 1}},
		{Key: "B", Modes: map[string]int{"gms_v83": 2, "gms_v95": 9}},
	}
	got := expectedFor(entries, "gms_v95")
	if len(got) != 1 || got["B"] != 9 {
		t.Errorf("expected only B=9 for gms_v95, got %v", got)
	}
}

// TestHandlerDocGeneratesHandlerOperations proves a dispatcher doc with a
// `handler:` field drives socket.handlers (serverbound) exactly like a
// `writer:` doc drives socket.writers.
func TestHandlerDocGeneratesHandlerOperations(t *testing.T) {
	src := []byte(`{
  "socket": {
    "handlers": [
      {
        "opCode": "0x6F",
        "handler": "CharacterInteractionHandle",
        "options": { "operations": {} }
      }
    ]
  }
}
`)
	n, _ := parseNode(src)
	doc := dispatcherDoc{Handler: "CharacterInteractionHandle", Operations: []opEntry{
		{Key: "MERCHANT_ORGANIZE", Modes: map[string]int{"gms_v61": 38}},
		{Key: "MERCHANT_WITHDRAW_MESO", Modes: map[string]int{"gms_v61": 41}},
	}}
	if doc.arrayKey() != "handlers" || doc.entryKey() != "handler" || doc.targetName() != "CharacterInteractionHandle" {
		t.Fatalf("doc target resolution wrong: %s/%s/%s", doc.arrayKey(), doc.entryKey(), doc.targetName())
	}
	w := findEntryNode(entriesOf(n, doc.arrayKey()), doc.entryKey(), doc.targetName())
	if w == nil {
		t.Fatal("handler entry not found")
	}
	if !setTable(w, "operations", doc.Operations, expectedFor(doc.Operations, "gms_v61")) {
		t.Fatal("setTable reported no change")
	}
	got := tableOf(w, "operations")
	if got["MERCHANT_ORGANIZE"] != 38 || got["MERCHANT_WITHDRAW_MESO"] != 41 {
		t.Errorf("handler operations wrong: %v", got)
	}
}

// TestAddEntryAppendsHandlerWhenAbsent proves the generator ADDS a handler
// entry (opCode + handler + operations) when the template lacks it and the
// YAML supplies the version opcode.
func TestAddEntryAppendsHandlerWhenAbsent(t *testing.T) {
	src := []byte(`{
  "socket": {
    "handlers": [
      { "opCode": "0x01", "handler": "LoginHandle" }
    ]
  }
}
`)
	n, _ := parseNode(src)
	doc := dispatcherDoc{Handler: "CharacterInteractionHandle", Operations: []opEntry{
		{Key: "CREATE", Modes: map[string]int{"gms_v79": 0}},
	}}
	if !addEntry(n, doc, "gms_v79", "0x78") {
		t.Fatal("addEntry failed")
	}
	w := findEntryNode(entriesOf(n, "handlers"), "handler", "CharacterInteractionHandle")
	if w == nil {
		t.Fatal("new handler not found after add")
	}
	out, _ := encodeNode(n)
	if !bytes.Contains(out, []byte(`"handler": "CharacterInteractionHandle"`)) || !bytes.Contains(out, []byte(`"0x78"`)) {
		t.Errorf("encoded output missing new handler:\n%s", out)
	}
}

func TestOperations_GeneratesErrorsTable(t *testing.T) {
	dir := t.TempDir()
	dispatchers := filepath.Join(dir, "dispatchers")
	templates := filepath.Join(dir, "templates")
	mustMkdirAll(t, dispatchers)
	mustMkdirAll(t, templates)

	mustWrite(t, filepath.Join(dispatchers, "d.yaml"), `
writer: TestWriter
op: TEST_OP
operations:
  - key: A_SUCCESS
    modes:
      gms_v83: 1
errors:
  - key: INVALID_COUPON_CODE
    modes:
      gms_v83: 7
  - key: COUPON_EXPIRED
    modes:
      gms_v83: 8
`)
	writeAllTemplates(t, templates, `{"socket":{"writers":[{"opCode":"0x1","writer":"TestWriter","options":{"operations":{"A_SUCCESS":1}}}],"handlers":[]}}`)

	var out, errOut bytes.Buffer
	if rc := operationsRun(operationsOpts{DispatchersDir: dispatchers, TemplatesDir: templates}, &out, &errOut); rc != 0 {
		t.Fatalf("generate rc=%d stderr=%s", rc, errOut.String())
	}

	raw := mustRead(t, filepath.Join(templates, "template_gms_83_1.json"))
	if !strings.Contains(raw, `"errors"`) {
		t.Fatalf("errors table not written: %s", raw)
	}
	if !strings.Contains(raw, `"INVALID_COUPON_CODE": 7`) || !strings.Contains(raw, `"COUPON_EXPIRED": 8`) {
		t.Fatalf("errors values wrong: %s", raw)
	}

	out.Reset()
	errOut.Reset()
	if rc := operationsRun(operationsOpts{DispatchersDir: dispatchers, TemplatesDir: templates, Check: true}, &out, &errOut); rc != 0 {
		t.Fatalf("check after generate rc=%d stderr=%s", rc, errOut.String())
	}
}

func TestOperations_CheckDetectsErrorsDriftMissingExtra(t *testing.T) {
	cases := []struct {
		name     string
		errsNode string
		want     string
	}{
		{"drift", `"errors":{"INVALID_COUPON_CODE":9}`, "operations DRIFT"},
		{"missing", `"errors":{}`, "operations MISSING"},
		{"extra", `"errors":{"INVALID_COUPON_CODE":7,"BOGUS":3}`, "operations EXTRA"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			dispatchers := filepath.Join(dir, "dispatchers")
			templates := filepath.Join(dir, "templates")
			mustMkdirAll(t, dispatchers)
			mustMkdirAll(t, templates)
			mustWrite(t, filepath.Join(dispatchers, "d.yaml"), `
writer: TestWriter
op: TEST_OP
errors:
  - key: INVALID_COUPON_CODE
    modes:
      gms_v83: 7
`)
			writeAllTemplates(t, templates,
				`{"socket":{"writers":[{"opCode":"0x1","writer":"TestWriter","options":{`+c.errsNode+`}}],"handlers":[]}}`)

			var out, errOut bytes.Buffer
			rc := operationsRun(operationsOpts{DispatchersDir: dispatchers, TemplatesDir: templates, Check: true}, &out, &errOut)
			if rc != 1 {
				t.Fatalf("rc=%d want 1; stderr=%s", rc, errOut.String())
			}
			if !strings.Contains(errOut.String(), c.want) {
				t.Fatalf("stderr %q does not contain %q", errOut.String(), c.want)
			}
		})
	}
}

// TestExpectedTableResolvesAliasOnlyWhereAnchorExists pins the `alias_of`
// contract: an alias takes its anchor's byte on every version the anchor
// covers, and is OMITTED — never invented — on a version whose switch has no
// case for the anchor.
func TestExpectedTableResolvesAliasOnlyWhereAnchorExists(t *testing.T) {
	entries := []opEntry{
		{Key: "CANNOT_TRANSFER_NO_EMPTY_SLOTS", Modes: map[string]int{"gms_v83": 223}},
		{Key: "no_character_slot", AliasOf: "CANNOT_TRANSFER_NO_EMPTY_SLOTS"},
	}
	got := expectedFor(entries, "gms_v83")
	if got["no_character_slot"] != 223 {
		t.Errorf("alias should take the anchor's byte, got %v", got)
	}
	// gms_v48 has no case for the anchor, so neither key may appear.
	if got := expectedFor(entries, "gms_v48"); len(got) != 0 {
		t.Errorf("alias must be omitted where the anchor is absent, got %v", got)
	}
}

// TestOperations_AliasKeyRoundTrips proves an aliased key survives both
// directions: generate emits it next to its anchor, and --check accepts it
// instead of reporting it EXTRA.
func TestOperations_AliasKeyRoundTrips(t *testing.T) {
	dir := t.TempDir()
	dispatchers := filepath.Join(dir, "dispatchers")
	templates := filepath.Join(dir, "templates")
	mustMkdirAll(t, dispatchers)
	mustMkdirAll(t, templates)

	mustWrite(t, filepath.Join(dispatchers, "d.yaml"), `
writer: TestWriter
op: TEST_OP
errors:
  - key: CANNOT_TRANSFER_OUT
    modes:
      gms_v83: 222
  - key: name_taken
    alias_of: CANNOT_TRANSFER_OUT
`)
	writeAllTemplates(t, templates, `{"socket":{"writers":[{"opCode":"0x1","writer":"TestWriter","options":{"errors":{}}}],"handlers":[]}}`)

	var out, errOut bytes.Buffer
	if rc := operationsRun(operationsOpts{DispatchersDir: dispatchers, TemplatesDir: templates}, &out, &errOut); rc != 0 {
		t.Fatalf("generate rc=%d stderr=%s", rc, errOut.String())
	}
	raw := mustRead(t, filepath.Join(templates, "template_gms_83_1.json"))
	if !strings.Contains(raw, `"name_taken": 222`) {
		t.Fatalf("alias not emitted with the anchor's byte: %s", raw)
	}

	out.Reset()
	errOut.Reset()
	if rc := operationsRun(operationsOpts{DispatchersDir: dispatchers, TemplatesDir: templates, Check: true}, &out, &errOut); rc != 0 {
		t.Fatalf("check after generate rc=%d stderr=%s", rc, errOut.String())
	}
}

// TestOperations_RejectsMalformedAlias covers the two malformed-doc shapes
// validateAliases exists to stop: a dangling anchor and an alias that also
// declares modes. Both exit 3 (doc error), not 1 (finding).
func TestOperations_RejectsMalformedAlias(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want string
	}{
		{"dangling", `
writer: TestWriter
op: TEST_OP
errors:
  - key: ANCHOR
    modes:
      gms_v83: 1
  - key: alias
    alias_of: NOT_A_KEY
`, "is not a non-alias key"},
		{"alias with modes", `
writer: TestWriter
op: TEST_OP
errors:
  - key: ANCHOR
    modes:
      gms_v83: 1
  - key: alias
    alias_of: ANCHOR
    modes:
      gms_v83: 2
`, "declares both alias_of and modes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			dispatchers := filepath.Join(dir, "dispatchers")
			templates := filepath.Join(dir, "templates")
			mustMkdirAll(t, dispatchers)
			mustMkdirAll(t, templates)
			mustWrite(t, filepath.Join(dispatchers, "d.yaml"), c.doc)
			writeAllTemplates(t, templates, `{"socket":{"writers":[],"handlers":[]}}`)

			var out, errOut bytes.Buffer
			rc := operationsRun(operationsOpts{DispatchersDir: dispatchers, TemplatesDir: templates, Check: true}, &out, &errOut)
			if rc != 3 {
				t.Fatalf("rc=%d want 3; stderr=%s", rc, errOut.String())
			}
			if !strings.Contains(errOut.String(), c.want) {
				t.Fatalf("stderr %q does not contain %q", errOut.String(), c.want)
			}
		})
	}
}

// writeMinimalTemplates writes a minimal socket doc — with a handful of
// pre-existing handlers spanning a range of opCodes — to every version's
// template file under dir/templates, and returns that templates dir. Used by
// tests that exercise addEntry's insert-when-absent path.
func writeMinimalTemplates(t *testing.T, dir string) string {
	t.Helper()
	tplDir := filepath.Join(dir, "templates")
	mustMkdirAll(t, tplDir)
	writeAllTemplates(t, tplDir, `{"socket":{"handlers":[
  {"opCode":"0x28","handler":"ExistingLow"},
  {"opCode":"0xE5","handler":"ExistingMid"},
  {"opCode":"0xFF","handler":"ExistingHigh"}
],"writers":[]}}`)
	return tplDir
}

// writeDispatcherDoc writes a single dispatcher enumeration YAML directly
// into dir, matching the DispatchersDir: dir passed to operationsRun below.
func writeDispatcherDoc(t *testing.T, dir, contents string) {
	t.Helper()
	mustMkdirAll(t, dir)
	mustWrite(t, filepath.Join(dir, "d.yaml"), contents)
}

func handlerEntry(t *testing.T, path, name string) map[string]interface{} {
	t.Helper()
	var d struct {
		Socket struct {
			Handlers []map[string]interface{} `json:"handlers"`
		} `json:"socket"`
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	for _, e := range d.Socket.Handlers {
		if e["handler"] == name {
			return e
		}
	}
	t.Fatalf("handler %q not found in %s", name, path)
	return nil
}

func handlerOpCodes(t *testing.T, path string) []int {
	t.Helper()
	var d struct {
		Socket struct {
			Handlers []struct {
				OpCode string `json:"opCode"`
			} `json:"handlers"`
		} `json:"socket"`
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	var out []int
	for _, e := range d.Socket.Handlers {
		n, err := strconv.ParseInt(e.OpCode, 0, 32)
		if err != nil {
			t.Fatalf("bad opCode %q", e.OpCode)
		}
		out = append(out, int(n))
	}
	return out
}

// A generated handler entry MUST carry a validator: BuildHandlerMap drops a
// handler whose validator name is not in the validator map, with only a
// Warnf, so a validator-less entry silently never routes.
func TestAddEntryEmitsValidatorAndServices(t *testing.T) {
	dir := t.TempDir()
	tplDir := writeMinimalTemplates(t, dir)
	writeDispatcherDoc(t, dir, `
handler: CashShopCouponCodeHandle
validator: LoggedInValidator
services: [channel]
fname: CCashShop::OnStatusCoupon
op: COUPON_CODE
opcodes:
  gms_v83: "0xE6"
`)
	if code := operationsRun(operationsOpts{DispatchersDir: dir, TemplatesDir: tplDir}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("generate exit = %d, want 0", code)
	}
	e := handlerEntry(t, filepath.Join(tplDir, "template_gms_83_1.json"), "CashShopCouponCodeHandle")
	if e["validator"] != "LoggedInValidator" {
		t.Errorf("validator = %v, want LoggedInValidator", e["validator"])
	}
	if got := e["services"]; !reflect.DeepEqual(got, []interface{}{"channel"}) {
		t.Errorf("services = %v, want [channel]", got)
	}
	if e["opCode"] != "0xE6" {
		t.Errorf("opCode = %v, want 0xE6", e["opCode"])
	}
	if _, ok := e["options"]; ok {
		t.Errorf("options should be omitted when the doc declares no tables, got %v", e["options"])
	}
}

// A generated entry must land at its ascending-opCode position so
// tools/template-opcode-order-guard.sh stays green.
func TestAddEntryInsertsInSortedPosition(t *testing.T) {
	dir := t.TempDir()
	tplDir := writeMinimalTemplates(t, dir) // contains handlers 0x28, 0xE5, 0xFF
	writeDispatcherDoc(t, dir, `
handler: CashShopCouponCodeHandle
validator: LoggedInValidator
services: [channel]
op: COUPON_CODE
opcodes:
  gms_v83: "0xE6"
`)
	if code := operationsRun(operationsOpts{DispatchersDir: dir, TemplatesDir: tplDir}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("generate exit = %d, want 0", code)
	}
	codes := handlerOpCodes(t, filepath.Join(tplDir, "template_gms_83_1.json"))
	for i := 1; i < len(codes); i++ {
		if codes[i] < codes[i-1] {
			t.Fatalf("handlers not ascending at %d: 0x%X after 0x%X (%v)", i, codes[i], codes[i-1], codes)
		}
	}
	if !slices.Contains(codes, 0xE6) {
		t.Fatalf("0xE6 not inserted: %v", codes)
	}
}
