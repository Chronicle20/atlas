package discover

import (
	"os"
	"path/filepath"
	"testing"
)

// readSendFixture loads the shared send_funcs.c.txt fixture.
func readSendFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "send_funcs.c.txt"))
	if err != nil {
		t.Fatalf("send fixture not found: %v", err)
	}
	return string(b)
}

// TestParseSendOpcodes_FullFixture exercises all four cases in the fixture
// file: single-packet decimal, two-packet function, hex-literal with 'u', and
// variable-opcode (which must be skipped).
func TestParseSendOpcodes_FullFixture(t *testing.T) {
	text := readSendFixture(t)
	got := ParseSendOpcodes(text)

	// Expected: 54 (decimal, single), 20 and 26 (two-packet), 0x3A=58 (hex+u).
	// nType (variable) must NOT appear.
	want := []int{20, 26, 54, 58}
	if len(got) != len(want) {
		t.Fatalf("ParseSendOpcodes: got %v (len %d), want %v (len %d)", got, len(got), want, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ParseSendOpcodes[%d]: got %d (0x%X), want %d (0x%X)", i, got[i], got[i], want[i], want[i])
		}
	}
}

// TestParseSendOpcodes_SingleDecimal confirms a single decimal literal is extracted.
func TestParseSendOpcodes_SingleDecimal(t *testing.T) {
	text := `
void CLogin::SendCheckPasswordPacket(CLogin *this)
{
  COutPacket oPacket;
  COutPacket::COutPacket(&oPacket, 54);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &oPacket);
  COutPacket::~COutPacket(&oPacket);
}
`
	got := ParseSendOpcodes(text)
	if len(got) != 1 || got[0] != 54 {
		t.Errorf("single decimal: got %v, want [54]", got)
	}
}

// TestParseSendOpcodes_TwoPackets confirms both opcodes from a two-packet
// function are extracted in sorted order.
func TestParseSendOpcodes_TwoPackets(t *testing.T) {
	text := `
void CLogin::OnConnect(CLogin *this)
{
  COutPacket oPacket;
  COutPacket oPacket2;
  COutPacket::COutPacket(&oPacket, 26);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &oPacket);
  COutPacket::~COutPacket(&oPacket);
  COutPacket::COutPacket(&oPacket2, 20);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &oPacket2);
  COutPacket::~COutPacket(&oPacket2);
}
`
	got := ParseSendOpcodes(text)
	want := []int{20, 26}
	if len(got) != len(want) {
		t.Fatalf("two-packet: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("two-packet[%d]: got %d, want %d", i, got[i], want[i])
		}
	}
}

// TestParseSendOpcodes_HexLiteral confirms a hex literal with 'u' suffix is extracted.
func TestParseSendOpcodes_HexLiteral(t *testing.T) {
	text := `
void CLogin::SendSetGenderPacket(CLogin *this, int nGender)
{
  COutPacket oPacket;
  COutPacket::COutPacket(&oPacket, 0x3Au);
  COutPacket::Encode1(&oPacket, nGender);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &oPacket);
  COutPacket::~COutPacket(&oPacket);
}
`
	got := ParseSendOpcodes(text)
	if len(got) != 1 || got[0] != 0x3A {
		t.Errorf("hex literal: got %v, want [%d (0x3A)]", got, 0x3A)
	}
}

// TestParseSendOpcodes_VariableOpcodeSkipped confirms that a variable 2nd
// argument does not produce any opcode in the output.
func TestParseSendOpcodes_VariableOpcodeSkipped(t *testing.T) {
	text := `
void CField::SendVariableOpPacket(CField *this, int nType)
{
  COutPacket oPacket;
  COutPacket::COutPacket(&oPacket, nType);
  COutPacket::Encode4(&oPacket, 42u);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &oPacket);
  COutPacket::~COutPacket(&oPacket);
}
`
	got := ParseSendOpcodes(text)
	if len(got) != 0 {
		t.Errorf("variable opcode: expected empty slice, got %v", got)
	}
}

// TestParseSendOpcodes_Dedup confirms duplicate opcodes in the same function
// are deduplicated.
func TestParseSendOpcodes_Dedup(t *testing.T) {
	text := `
void SomeFunc()
{
  COutPacket p1;
  COutPacket p2;
  COutPacket::COutPacket(&p1, 42);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &p1);
  COutPacket::~COutPacket(&p1);
  COutPacket::COutPacket(&p2, 42);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::GetInstance(), &p2);
  COutPacket::~COutPacket(&p2);
}
`
	got := ParseSendOpcodes(text)
	if len(got) != 1 || got[0] != 42 {
		t.Errorf("dedup: got %v, want [42]", got)
	}
}

// TestParseSendOpcodes_Empty confirms an empty string produces an empty slice.
func TestParseSendOpcodes_Empty(t *testing.T) {
	got := ParseSendOpcodes("")
	if len(got) != 0 {
		t.Errorf("empty text: expected nil/empty slice, got %v", got)
	}
}
