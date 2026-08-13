package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOVE_DRAGON is int ownerCharacterId + the raw CMovePath blob. The blob already
// begins with the start position, so it must NOT be written separately.
//
// packet-audit:verify packet=dragon/clientbound/DragonMove version=gms_v95 ida=0x50ad30
func TestDragonMoveIsOwnerIdPlusOpaqueBlob(t *testing.T) {
	blob := []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}
	got := test.Encode(t, test.CreateContext("GMS", 95, 1), NewDragonMove(4242, blob).Encode, nil)

	want := append([]byte{0x92, 0x10, 0x00, 0x00}, blob...)
	if !bytes.Equal(got, want) {
		t.Fatalf("move bytes = % X, want % X", got, want)
	}
}

func TestDragonMoveRoundTrip(t *testing.T) {
	var out DragonMove
	test.RoundTrip(t, test.CreateContext("GMS", 95, 1),
		NewDragonMove(4242, []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}).Encode, out.Decode, nil)
	if out.OwnerCharacterId() != 4242 || len(out.RawMovement()) != 6 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
