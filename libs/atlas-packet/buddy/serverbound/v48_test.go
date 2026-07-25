package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v48 BUDDYLIST_MODIFY (serverbound op 100 / 0x64) is a mode-prefix
// dispatcher. The leading Encode1(mode) byte is consumed by the channel-side
// BuddyOperationHandle dispatcher; each sub-struct below models only the body
// that follows. All three arms live in the same 0x4c6xxx cluster
// (GMS_v48_1_DEVM.exe, port 13337), each opening COutPacket(100):
//
//	ADD    sub_4C6452 @0x4c6538: Encode1(1) + EncodeStr(name)        (mode 1)
//	ACCEPT sub_4C6643 @0x4c66aa: Encode1(2) + Encode4(fromCharId)    (mode 2)
//	DELETE sub_4C659B @0x4c65fb: Encode1(3) + Encode4(buddyCharId)   (mode 3)
//
// The v48 ADD arm sends ONLY the buddy name (no group). The group name is a
// later addition: IDA-verified absent at v48 (@0x4c6538) and v61 (@0x4e9cf7),
// present at v72 (@0x51568e), v79 (@0x51c72a) and v87
// (CField::SendSetFriendMsg @0x558844). OperationAdd gates the group on
// MajorVersion() > 61, so the v48 wire is name-only.

// packet-audit:verify packet=buddy/serverbound/BuddyOperationAdd version=gms_v48 ida=0x4c6452
func TestOperationAddV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := OperationAdd{name: "TestBuddy"}
	want := []byte{
		0x09, 0x00, // name length = 9 (WriteAsciiString len prefix, EncodeStr @0x4c655f)
		'T', 'e', 's', 't', 'B', 'u', 'd', 'd', 'y', // "TestBuddy" (name)
		// NO group string on the v48 wire (group is >v61 only)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v48 BuddyOperationAdd golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// packet-audit:verify packet=buddy/serverbound/BuddyOperationAccept version=gms_v48 ida=0x4c6643
func TestOperationAcceptV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := OperationAccept{fromCharacterId: 12345}
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // fromCharacterId = 12345 (Encode4 @0x4c66c2)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v48 BuddyOperationAccept golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// packet-audit:verify packet=buddy/serverbound/BuddyOperationDelete version=gms_v48 ida=0x4c659b
func TestOperationDeleteV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := OperationDelete{buddyCharacterId: 67890}
	want := []byte{
		0x32, 0x09, 0x01, 0x00, // buddyCharacterId = 67890 (Encode4 @0x4c6613)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v48 BuddyOperationDelete golden mismatch\n got: % x\nwant: % x", got, want)
	}
}
