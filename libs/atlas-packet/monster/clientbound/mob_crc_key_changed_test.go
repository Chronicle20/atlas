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

// TestMobCrcKeyChangedBytesV79 pins the exact wire bytes against the v79 client
// read order. MOB_CRC_KEY_CHANGED (op 227) is NOT a per-mob OnMobPacket case; it
// is dispatched at the CMobPool top level to CMobPool::OnMobCrcKeyChanged
// @0x647197 (GMS_v79_1_DEVM.exe, port 13340), which reads:
//
//	Decode4 @0x6471af — m_dwMobCrcKey (the new mob CRC key)
//
// No uniqueId prefix (pool-level, no GetMob). Byte-identical to v83; no codec change.
//
// packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=gms_v79 ida=0x647197
func TestMobCrcKeyChangedBytesV79(t *testing.T) {
	input := NewMobCrcKeyChanged(0x12345678)
	ctx := pt.CreateContext("GMS", 79, 1)
	want := []byte{
		0x78, 0x56, 0x34, 0x12, // crcKey uint32 LE = 0x12345678 (Decode4 @0x6471af)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 mobCrcKeyChanged bytes:\n got % x\nwant % x", got, want)
	}
}
