package csv

import "testing"

func TestLoadClientbound(t *testing.T) {
	m, err := Load("testdata/clientbound_sample.csv", DirClientbound)
	if err != nil {
		t.Fatal(err)
	}
	row, ok := m.ByFName("CLogin::OnCheckPasswordResult")
	if !ok {
		t.Fatal("FName not found")
	}
	if got := row.Opcode("GMS", 95); got != 0x00 {
		t.Errorf("v95 opcode: got 0x%02x, want 0x00", got)
	}
	if row.Direction != DirClientbound {
		t.Errorf("direction: got %v, want clientbound", row.Direction)
	}
}

func TestLoadServerbound(t *testing.T) {
	m, err := Load("testdata/serverbound_sample.csv", DirServerbound)
	if err != nil {
		t.Fatal(err)
	}
	row, ok := m.ByFName("CLogin::SendSelectCharPacket")
	if !ok {
		t.Fatal("FName not found")
	}
	if got := row.Opcode("GMS", 95); got != 0x13 {
		t.Errorf("v95 opcode: got 0x%02x, want 0x13", got)
	}
}
