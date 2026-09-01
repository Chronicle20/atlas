package clientbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestCharacterKeyMapJMS185ByteLength pins the jms_v185 KEYMAP wire length.
// CFuncKeyMappedMan::OnInit @0x5e79aa (JMS 185, IDB a977912e) reads 94
// FUNCKEY_MAPPED entries (`v5 = 94`, memcpy 0x1D6 = 470 = 94*5) with no
// m_uDataLen guard — unlike GMS v95's OnInit @0x568c30, which reads 89
// entries and guards on `m_uDataLen < 0x1BDu` (0x1BD = 445 = 89*5). A
// 90-entry (GMS-shaped) encode against a JMS client under-reads by 20 bytes
// and throws ZException(38) mid-decode (task-273 round 7). The body is
// 1 (flag byte) + 94*5 (entries) = 471 bytes.
func TestCharacterKeyMapJMS185ByteLength(t *testing.T) {
	keys := map[int32]KeyBinding{
		2:  {KeyType: 4, KeyAction: 10},
		16: {KeyType: 4, KeyAction: 8},
		41: {KeyType: 4, KeyAction: 11},
	}
	input := NewCharacterKeyMap(keys)
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("JMS", 185, 1)

	got := input.Encode(l, ctx)(nil)
	if len(got) != 471 {
		t.Errorf("jms_v185 keymap encode: got %d bytes, want 471 (1 flag byte + 94*5 entries)", len(got))
	}

	test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
}

// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=gms_v83 ida=0x58ddb4
// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=gms_v87 ida=0x5bd279
// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=gms_v95 ida=0x568c30
// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=jms_v185 ida=0x5e79aa
// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=gms_v84 ida=0x59dda7
func TestCharacterKeyMap(t *testing.T) {
	keys := map[int32]KeyBinding{
		2:  {KeyType: 4, KeyAction: 10},
		16: {KeyType: 4, KeyAction: 8},
		41: {KeyType: 4, KeyAction: 11},
	}
	input := NewCharacterKeyMap(keys)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestCharacterKeyMapResetToDefault(t *testing.T) {
	input := NewCharacterKeyMapResetToDefault()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
