package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=gms_v83 ida=0x6797be
// packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=gms_v84 ida=0x690354
// packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=gms_v87 ida=0x6b5399
// packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=gms_v95 ida=0x657230
// packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=jms_v185 ida=0x6f8bcb
func TestMobCrcKeyChanged(t *testing.T) {
	input := NewMobCrcKeyChanged(0x12345678)

	// Golden bytes (v83 baseline). The packet is a single Decode4 of the new
	// mob CRC key — CMobPool::OnMobCrcKeyChanged @0x6797be:
	//   this->m_dwMobCrcKey = CInPacket::Decode4(iPacket)
	// uint32 little-endian.
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x78, 0x56, 0x34, 0x12, // crcKey uint32 LE = 0x12345678 (Decode4 @0x6797be)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobCrcKeyChanged layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
