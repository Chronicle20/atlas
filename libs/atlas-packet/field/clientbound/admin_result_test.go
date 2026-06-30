package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldAdminResult version=gms_v79 ida=0x52075c
// packet-audit:verify packet=field/clientbound/FieldAdminResult version=gms_v83 ida=0x5352e9
// packet-audit:verify packet=field/clientbound/FieldAdminResult version=gms_v84 ida=0x54156f
// packet-audit:verify packet=field/clientbound/FieldAdminResult version=gms_v87 ida=0x55cac3
// packet-audit:verify packet=field/clientbound/FieldAdminResult version=gms_v95 ida=0x53bc20
// packet-audit:verify packet=field/clientbound/FieldAdminResult version=jms_v185 ida=0x57255f
//
// ADMIN_RESULT is a mode-demux flattened like SPOUSE_CHAT; the flat read order
// differs per version, so the golden below is asserted for the v83 baseline and the
// round-trip exercises every variant against its own version-branched schema.
func TestAdminResultGolden(t *testing.T) {
	// v83 flat post-mode schema: S,B,B,B,I,B,B,S,S,S,B,B,B
	//   strs = 4, bytes = 8, mapId = 1.
	bs := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	ss := []string{"a", "bb", "ccc", "dddd"}
	input := NewAdminResult(0x0B, bs, ss, 0x01020304)
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	// v83 post-mode order: s(0),b(0),b(1),b(2),i,b(3),b(4),s(1),s(2),s(3),b(5),b(6),b(7)
	want := []byte{
		0x0B,                           // mode             (Decode1 @0x5352e9)
		0x01, 0x00, 'a',                // s[0]="a"         (DecodeStr)
		0x11,                           // b[0]             (Decode1)
		0x22,                           // b[1]             (Decode1)
		0x33,                           // b[2]             (Decode1)
		0x04, 0x03, 0x02, 0x01,         // mapId=0x01020304 (Decode4)
		0x44,                           // b[3]             (Decode1)
		0x55,                           // b[4]             (Decode1)
		0x02, 0x00, 'b', 'b',           // s[1]="bb"        (DecodeStr)
		0x03, 0x00, 'c', 'c', 'c',      // s[2]="ccc"       (DecodeStr)
		0x04, 0x00, 'd', 'd', 'd', 'd', // s[3]="dddd"      (DecodeStr)
		0x66,                           // b[5]             (Decode1)
		0x77,                           // b[6]             (Decode1)
		0x88,                           // b[7]             (Decode1)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("golden mismatch:\n got %v\nwant %v", actual, want)
	}
}

// TestAdminResultByteOutputV79 pins the gms_v79 ADMIN_RESULT (op 0x88) clientbound
// flat union. IDA: CField::OnAdminResult @0x52075c (GMS_v79_1_DEVM.exe). The export
// harvest's flat post-mode order is b,b,b,i,b,b,s,s,s,b,b,b (3 strings, vs v83's 4)
// — addresses cited in the codec's v79 branch. mode (Decode1 @0x520776) leads.
func TestAdminResultByteOutputV79(t *testing.T) {
	bs := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	ss := []string{"a", "bb", "ccc"}
	input := NewAdminResult(0x0B, bs, ss, 0x01020304)
	ctx := test.CreateContext("GMS", 79, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x0B,                   // mode             (Decode1 @0x520776)
		0x11,                   // b[0]             (Decode1 @0x520d9b)
		0x22,                   // b[1]             (Decode1 @0x520b5f)
		0x33,                   // b[2]             (Decode1 @0x520b71)
		0x04, 0x03, 0x02, 0x01, // mapId=0x01020304 (Decode4 @0x520cd9)
		0x44,                      // b[3]          (Decode1 @0x520bb5)
		0x55,                      // b[4]          (Decode1 @0x5207c4)
		0x01, 0x00, 'a',           // s[0]="a"      (DecodeStr @0x5207d5)
		0x02, 0x00, 'b', 'b',      // s[1]="bb"     (DecodeStr @0x520827)
		0x03, 0x00, 'c', 'c', 'c', // s[2]="ccc"    (DecodeStr @0x520836)
		0x66, // b[5]                              (Decode1 @0x52096d)
		0x77, // b[6]                              (Decode1 @0x520a39)
		0x88, // b[7]                              (Decode1 @0x520ad0)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("v79 golden mismatch:\n got %v\nwant %v", actual, want)
	}
}

func TestAdminResultRoundTrip(t *testing.T) {
	bs := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	ss := []string{"a", "bb", "ccc", "dddd"}
	input := NewAdminResult(0x0B, bs, ss, 0x01020304)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
