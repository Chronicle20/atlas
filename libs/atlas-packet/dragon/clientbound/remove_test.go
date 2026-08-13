package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// REMOVE_DRAGON has no body: the only field is the owner character id, and the
// client has no handler arm for the opcode at all.
func TestDragonRemoveIsOwnerIdOnly(t *testing.T) {
	got := test.Encode(t, test.CreateContext("GMS", 95, 1), NewDragonRemove(4242).Encode, nil)
	if !bytes.Equal(got, []byte{0x92, 0x10, 0x00, 0x00}) {
		t.Fatalf("remove bytes = % X, want 92 10 00 00", got)
	}
}

func TestDragonRemoveRoundTrip(t *testing.T) {
	var out DragonRemove
	test.RoundTrip(t, test.CreateContext("GMS", 95, 1), NewDragonRemove(4242).Encode, out.Decode, nil)
	if out.OwnerCharacterId() != 4242 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
