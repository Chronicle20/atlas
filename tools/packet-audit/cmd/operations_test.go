package cmd

import (
	"bytes"
	"testing"
)

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
	doc := dispatcherDoc{Writer: "CashShopOperation", Operations: []struct {
		Key   string         `yaml:"key"`
		Modes map[string]int `yaml:"modes"`
	}{
		{Key: "PURCHASE_SUCCESS", Modes: map[string]int{"gms_v95": 100}},
		{Key: "LOAD_INVENTORY_SUCCESS", Modes: map[string]int{"gms_v95": 88}},
	}}
	w := findWriterNode(writersOf(n), "CashShopOperation")
	if w == nil {
		t.Fatal("writer not found")
	}
	if !setOperations(w, doc, expectedTable(doc, "gms_v95")) {
		t.Fatal("setOperations reported no change")
	}
	got := operationsOf(w)
	if got["PURCHASE_SUCCESS"] != 100 || got["LOAD_INVENTORY_SUCCESS"] != 88 {
		t.Errorf("operations wrong: %v", got)
	}
	// Re-running with the same expected table must be a no-op (idempotent).
	if setOperations(w, doc, expectedTable(doc, "gms_v95")) {
		t.Error("second setOperations should be idempotent")
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
	doc := dispatcherDoc{Writer: "NewOp", Operations: []struct {
		Key   string         `yaml:"key"`
		Modes map[string]int `yaml:"modes"`
	}{
		{Key: "A", Modes: map[string]int{"gms_v87": 2}},
	}}
	if !addWriter(n, doc, "0x14B", expectedTable(doc, "gms_v87")) {
		t.Fatal("addWriter failed")
	}
	w := findWriterNode(writersOf(n), "NewOp")
	if w == nil {
		t.Fatal("new writer not found after add")
	}
	if got := operationsOf(w); got["A"] != 2 {
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
	doc := dispatcherDoc{Operations: []struct {
		Key   string         `yaml:"key"`
		Modes map[string]int `yaml:"modes"`
	}{
		{Key: "A", Modes: map[string]int{"gms_v83": 1}},
		{Key: "B", Modes: map[string]int{"gms_v83": 2, "gms_v95": 9}},
	}}
	got := expectedTable(doc, "gms_v95")
	if len(got) != 1 || got["B"] != 9 {
		t.Errorf("expected only B=9 for gms_v95, got %v", got)
	}
}
