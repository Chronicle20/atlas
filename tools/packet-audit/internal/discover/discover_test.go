package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDispatchSwitch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "process_packet_v83.c.txt"))
	if err != nil {
		t.Fatal(err)
	}
	cases, err := ParseDispatch(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	byOp := map[int]DispatchCase{}
	for _, c := range cases {
		byOp[c.Opcode] = c
	}
	if c := byOp[0x11]; c.Handler != "CLogin::OnFoo" {
		t.Errorf("0x11 -> %+v", c)
	}
	// fallthrough pair: both opcodes map to the same handler
	if byOp[0x12].Handler == "" || byOp[0x12].Handler != byOp[0x13].Handler {
		t.Errorf("fallthrough not shared: %+v / %+v", byOp[0x12], byOp[0x13])
	}
	// decimal label (200 == 0xC8) must be found
	if c := byOp[200]; c.Handler == "" {
		t.Errorf("decimal label 200 not found; got %+v", c)
	}
	// unnamed callee preserved as sub_ address-name, not dropped
	found := false
	for _, c := range cases {
		if c.Handler == "sub_5E1230" {
			found = true
		}
	}
	if !found {
		t.Error("sub_ handler dropped — discovery must keep unnamed handlers")
	}
}
