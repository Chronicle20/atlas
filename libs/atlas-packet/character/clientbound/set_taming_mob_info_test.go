package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterSetTamingMobInfo version=gms_v83 ida=0xa29115
// packet-audit:verify packet=character/clientbound/CharacterSetTamingMobInfo version=gms_v84 ida=0xa748d8
// packet-audit:verify packet=character/clientbound/CharacterSetTamingMobInfo version=gms_v87 ida=0xac0d8b
// packet-audit:verify packet=character/clientbound/CharacterSetTamingMobInfo version=gms_v95 ida=0x9f7280
// packet-audit:verify packet=character/clientbound/CharacterSetTamingMobInfo version=jms_v185 ida=0xb103a1
//
// Layout (Decode4 charId, Decode4 level, Decode4 exp, Decode4 fatigue, Decode1
// levelUp flag) is byte-identical across all 5 versions (IDA-verified
// CWvsContext::OnSetTamingMobInfo at each address above) — no version gate.
func TestSetTamingMobInfoFieldOrder(t *testing.T) {
	m := NewSetTamingMobInfo(100200, 5, 1234, 42, true)
	got := m.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x68, 0x87, 0x01, 0x00, // characterId 100200
		0x05, 0x00, 0x00, 0x00, // level 5
		0xd2, 0x04, 0x00, 0x00, // exp 1234
		0x2a, 0x00, 0x00, 0x00, // tiredness 42
		0x01, // levelUp true
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("SET_TAMING_MOB_INFO layout mismatch\n got % x\nwant % x", got, want)
	}
}

// SetTamingMobInfo v48 byte-fixture — SET_TAMING_MOB_INFO, op 40 (0x28).
//
// Client read — CWvsContext::OnSetTamingMobInfo (sub_72032B @0x72032b):
//   Decode4  charId    (GetUser)                    /*0x720343*/
//   Decode4  level     (v6[1244])                   /*0x72038a*/
//   Decode4  exp       (v7[1245])                   /*0x72039a*/
//   Decode4  tiredness (v8[1246])                   /*0x7203aa*/
//   Decode1  levelUp   (v10 -> level-up effect)     /*0x7203df*/
// Byte-identical to v61/v83 (no version gate). v48 op 40.
//
// packet-audit:verify packet=character/clientbound/CharacterSetTamingMobInfo version=gms_v48 ida=0x72032b
func TestSetTamingMobInfoV48ByteOutput(t *testing.T) {
	m := NewSetTamingMobInfo(100200, 5, 1234, 42, true)
	got := m.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x68, 0x87, 0x01, 0x00, // characterId 100200 /*0x720343*/
		0x05, 0x00, 0x00, 0x00, // level 5            /*0x72038a*/
		0xd2, 0x04, 0x00, 0x00, // exp 1234           /*0x72039a*/
		0x2a, 0x00, 0x00, 0x00, // tiredness 42       /*0x7203aa*/
		0x01, // levelUp true                          /*0x7203df*/
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 SET_TAMING_MOB_INFO layout mismatch\n got % x\nwant % x", got, want)
	}
}
