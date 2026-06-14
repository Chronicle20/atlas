package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldPlayJukebox version=gms_v83 ida=0x535224
// packet-audit:verify packet=field/clientbound/FieldPlayJukebox version=gms_v84 ida=0x5414aa
// packet-audit:verify packet=field/clientbound/FieldPlayJukebox version=gms_v87 ida=0x55c9fe
// packet-audit:verify packet=field/clientbound/FieldPlayJukebox version=gms_v95 ida=0x537940
// packet-audit:verify packet=field/clientbound/FieldPlayJukebox version=jms_v185 ida=0x572488
func TestPlayJukeboxGolden(t *testing.T) {
	input := NewPlayJukebox(0x0049DA9C, "Hero")
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{
		0x9C, 0xDA, 0x49, 0x00, // itemId 4839068
		0x04, 0x00, 'H', 'e', 'r', 'o', // playerName
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestPlayJukeboxRoundTrip(t *testing.T) {
	input := NewPlayJukebox(0x0049DA9C, "Hero")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
