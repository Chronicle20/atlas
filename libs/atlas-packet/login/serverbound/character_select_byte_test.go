package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// These byte-fixtures pin the serverbound body layout of the three
// CLogin::SendSelectCharPacket modes against the client COutPacket::EncodeN
// write order. All three modes live in one send function that branches on
// m_bLoginOpt; the audit reports give one shared address per version:
//
//	v83 @0x5f726d, v87 @0x62e9f6, v95 @0x5da2a0.
//
// Field encodings (atlas-socket/response):
//   WriteInt        -> LE uint32 (4 bytes)
//   WriteByte       -> 1 byte
//   WriteAsciiString-> LE uint16 length + ShiftJIS bytes (ASCII == identity)
//
// The opcode (0x13/0x1D/0x1E in v83/v87, 19/28/29 in v95) is the COutPacket
// header, written by the socket layer, not by Encode(); these fixtures assert
// the body only.

// helper: little-endian length-prefixed ascii string bytes.
func lp(s string) []byte {
	n := len(s)
	out := []byte{byte(n), byte(n >> 8)}
	return append(out, []byte(s)...)
}

func le4(v uint32) []byte {
	return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
}

// TestCharacterSelectByteOutput pins CHAR_SELECT (opcode 0x13/19) body:
// Encode4(charId) + EncodeStr(mac) + EncodeStr(hwid).
//
//	v83 case m_bLoginOpt<=3 @0x5f72fe: Encode4 v22 [0x5f7310],
//	    EncodeStr sub_5FCC81=mac [0x5f734b], EncodeStr sub_5FCDED=hwid [0x5f7381].
//	v87 same shape @0x62ea87: Encode4 [0x62ea99], EncodeStr mac [0x62ead4],
//	    EncodeStr hwid [0x62eb0a].
//	v95 case 2/3 @0x5da6ca: Encode4 [0x5da6dc],
//	    EncodeStr GetLocalMacAddress=mac [0x5da718],
//	    EncodeStr GetLocalMacAddressWithHDDSerialNo=hwid [0x5da753].
//
// packet-audit:verify packet=login/serverbound/CharacterSelect version=gms_v83 ida=0x5f726d
// packet-audit:verify packet=login/serverbound/CharacterSelect version=gms_v84 ida=0x60c1e3
// packet-audit:verify packet=login/serverbound/CharacterSelect version=gms_v87 ida=0x62e9f6
// packet-audit:verify packet=login/serverbound/CharacterSelect version=gms_v95 ida=0x5da2a0
// packet-audit:verify packet=login/serverbound/CharacterSelect version=jms_v185 ida=0x66ddac
// packet-audit:verify packet=login/serverbound/CharacterSelect version=gms_v79 ida=0x5ccae3
//
// gms_v72: CLogin::SendSelectCharPacket = sub_5B1D03 @0x5b1d03 (GMS_v72.1_U_DEVM.exe,
// port 13339): COutPacket(19) @0x5b1d6c; Encode4(charId) @0x5b1d8c; EncodeStr(mac=
// GetLocalMacAddress) @0x5b1dc7; EncodeStr(hwid=GetLocalMacAddressWithHDDSerialNo)
// @0x5b1dfd — charId + mac + hwid, same as the GMS >12 path. Fixtured below.
//
// packet-audit:verify packet=login/serverbound/CharacterSelect version=gms_v72 ida=0x5b1d03
//
// jms CLogin::SendSelectCharPacket @0x66ddac, m_bLoginOpt<=3 arm:
//
//	COutPacket(6) + Encode4(charId) — NO mac/hwid (differs from GMS).
func TestCharacterSelectByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := CharacterSelect{characterId: 0x01020304, mac: "MAC", hwid: "HW"}
	// GMS major>12: charId(4) + EncodeStr(mac) + EncodeStr(hwid).
	want := append(le4(0x01020304), lp("MAC")...)
	want = append(want, lp("HW")...)
	for _, v := range []struct {
		Name         string
		Region       string
		Major, Minor uint16
	} {
		{"GMS v72", "GMS", 72, 1},
		{"GMS v83", "GMS", 83, 1},
		{"GMS v84", "GMS", 84, 1},
		{"GMS v87", "GMS", 87, 1},
		{"GMS v95", "GMS", 95, 1},
	} {
		t.Run(v.Name, func(t *testing.T) {
			got := in.Encode(l, pt.CreateContext(v.Region, v.Major, v.Minor))(nil)
			require.Equal(t, want, got, "%s CHAR_SELECT body", v.Name)
		})
	}

	// jms: charId(4) ONLY (no mac/hwid).
	t.Run("JMS v185", func(t *testing.T) {
		wantJMS := le4(0x01020304)
		got := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
		require.Equal(t, wantJMS, got, "JMS v185 CHAR_SELECT body")
	})
}

// TestCharacterSelectWithPicByteOutput pins CHAR_SELECT_WITH_PIC (0x1E/29) body:
// EncodeStr(pic) + Encode4(charId) + EncodeStr(mac) + EncodeStr(hwid).
//
//	v83 case m_bLoginOpt==1 @0x5f745e: EncodeStr pic [0x5f747b],
//	    Encode4 [0x5f7486], EncodeStr mac [0x5f74c1], EncodeStr hwid [0x5f74f7].
//	v87 same @0x62ebe7: EncodeStr pic [0x62ec04], Encode4 [0x62ec0f],
//	    EncodeStr mac [0x62ec4a], EncodeStr hwid [0x62ec80].
//	v95 case 1 @0x5da598: EncodeStr sSPW=pic [0x5da5b7], Encode4 [0x5da5c1],
//	    EncodeStr mac [0x5da5fd], EncodeStr hwid [0x5da638].
//
// (Marker lives on TestCharacterSelectByteOutput — all three modes share the
// CLogin::SendSelectCharPacket send function and the same audit address, so the
// packet-id × version pair carries a single marker.)
func TestCharacterSelectWithPicByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := CharacterSelectWithPic{pic: "PIC", characterId: 0x01020304, mac: "MAC", hwid: "HW"}
	// GMS: EncodeStr(pic) + charId(4) + EncodeStr(mac) + EncodeStr(hwid).
	want := append(lp("PIC"), le4(0x01020304)...)
	want = append(want, lp("MAC")...)
	want = append(want, lp("HW")...)
	for _, v := range []struct {
		Name         string
		Region       string
		Major, Minor uint16
	}{
		{"GMS v83", "GMS", 83, 1},
		{"GMS v87", "GMS", 87, 1},
		{"GMS v95", "GMS", 95, 1},
	} {
		t.Run(v.Name, func(t *testing.T) {
			got := in.Encode(l, pt.CreateContext(v.Region, v.Major, v.Minor))(nil)
			require.Equal(t, want, got, "%s CHAR_SELECT_WITH_PIC body", v.Name)
		})
	}

	// jms case m_bLoginOpt==1 @0x66ddac: COutPacket(0x14) + EncodeStr(pic) +
	// Encode4(charId) — NO mac/hwid (differs from GMS).
	t.Run("JMS v185", func(t *testing.T) {
		wantJMS := append(lp("PIC"), le4(0x01020304)...)
		got := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
		require.Equal(t, wantJMS, got, "JMS v185 CHAR_SELECT_WITH_PIC body")
	})
}

// TestRegisterPicByteOutput pins REGISTER_PIC (0x1D/28) body:
// Encode1(mode) + Encode4(charId) + EncodeStr(mac) + EncodeStr(hwid) + EncodeStr(pic).
//
//	v83 case else @0x5f7592: Encode1(1) [0x5f75a0], Encode4 [0x5f75ab],
//	    EncodeStr mac [0x5f75e6], EncodeStr hwid [0x5f761c], EncodeStr pic [0x5f7635].
//	v87 same @0x62ed1b: Encode1(1) [0x62ed29], Encode4 [0x62ed34],
//	    EncodeStr mac [0x62ed6f], EncodeStr hwid [0x62eda5], EncodeStr pic [0x62edbe].
//	v95 case 0 @0x5da3bb: Encode1(1) [0x5da3cb], Encode4 [0x5da3d5],
//	    EncodeStr mac [0x5da411], EncodeStr hwid [0x5da44c], EncodeStr sSPW=pic [0x5da466].
//
// (Marker lives on TestCharacterSelectByteOutput — shared send function/address.)
func TestRegisterPicByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := CharacterSelectRegisterPic{mode: 1, characterId: 0x01020304, mac: "MAC", hwid: "HW", pic: "PIC"}
	// GMS: mode(1) + charId(4) + EncodeStr(mac) + EncodeStr(hwid) + EncodeStr(pic).
	want := []byte{0x01}
	want = append(want, le4(0x01020304)...)
	want = append(want, lp("MAC")...)
	want = append(want, lp("HW")...)
	want = append(want, lp("PIC")...)
	for _, v := range []struct {
		Name         string
		Region       string
		Major, Minor uint16
	}{
		{"GMS v83", "GMS", 83, 1},
		{"GMS v87", "GMS", 87, 1},
		{"GMS v95", "GMS", 95, 1},
	} {
		t.Run(v.Name, func(t *testing.T) {
			got := in.Encode(l, pt.CreateContext(v.Region, v.Major, v.Minor))(nil)
			require.Equal(t, want, got, "%s REGISTER_PIC body", v.Name)
		})
	}

	// jms case m_bLoginOpt==0 (else) @0x66ddac: COutPacket(0x13) + Encode1(flag) +
	// Encode4(charId) + if(flag) EncodeStr(pic) — NO mac/hwid (differs from GMS).
	// mode is non-zero so the client's if(flag) EncodeStr path matches atlas's
	// unconditional pic write.
	t.Run("JMS v185", func(t *testing.T) {
		wantJMS := []byte{0x01}
		wantJMS = append(wantJMS, le4(0x01020304)...)
		wantJMS = append(wantJMS, lp("PIC")...)
		got := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
		require.Equal(t, wantJMS, got, "JMS v185 REGISTER_PIC body")
	})
}
