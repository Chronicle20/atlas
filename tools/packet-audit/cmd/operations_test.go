package cmd

import (
	"bytes"
	"os"
	"path/filepath"
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
