package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
