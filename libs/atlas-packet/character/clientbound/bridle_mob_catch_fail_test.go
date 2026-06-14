package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterBridleMobCatchFail version=gms_v83 ida=0xa0800e
// packet-audit:verify packet=character/clientbound/CharacterBridleMobCatchFail version=gms_v84 ida=0xa522fc
// packet-audit:verify packet=character/clientbound/CharacterBridleMobCatchFail version=gms_v87 ida=0xa9d692
// packet-audit:verify packet=character/clientbound/CharacterBridleMobCatchFail version=gms_v95 ida=0x9d9a80
// packet-audit:verify packet=character/clientbound/CharacterBridleMobCatchFail version=jms_v185 ida=0xaec5ed
func TestBridleMobCatchFail(t *testing.T) {
	input := NewBridleMobCatchFail(0x01, 0x00226CA0, 0x00)

	// Golden bytes (v83 baseline). CWvsContext::OnBridleMobCatchFail @0xa0800e:
	//   v15 = Decode1(a1)  -> reason byte
	//   v1  = Decode4(a1)  -> itemId int32 LE (GetBridleItem lookup)
	//         Decode4(a1)  -> trailing int32, read but discarded
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x01,                   // reason byte (Decode1 @0xa0800e)
		0xA0, 0x6C, 0x22, 0x00, // itemId int32 LE = 0x00226CA0 (Decode4 @0xa0800e)
		0x00, 0x00, 0x00, 0x00, // unused int32 LE (trailing Decode4 @0xa0800e)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("BridleMobCatchFail layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
