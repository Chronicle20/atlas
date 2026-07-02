package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v61: SERVERSTATUS_REQUEST send = sub_5655DF @0x5655df (GMS_v61.1_U_DEVM.exe,
// port 13338): COutPacket(6)@0x565612; Encode2(worldId)@0x56562e = 2-byte short.
// Matches the codec's GMS WriteShort. worldId=3 → 03 00.
//
// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v61 ida=0x5655df
func TestServerStatusRequestV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := ServerStatusRequest{worldId: world.Id(3)}
	want := []byte{0x03, 0x00} // worldId LE short
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 ServerStatusRequest body: got % x, want % x", got, want)
	}
}

// gms_v72: CLogin::SendCheckUserLimitPacket = sub_5B238A @0x5b238a
// (GMS_v72.1_U_DEVM.exe, port 13339): COutPacket(6) @0x5b23bb; Encode2(worldId)
// @0x5b23d7 = 2-byte short. Matches the codec's WriteShort. Marker-only (tier-0).
//
// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v72 ida=0x5b238a
// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v83 ida=0x5f8078
// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v87 ida=0x62f80a
// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v95 ida=0x5d43d0
// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v84 ida=0x60cfee
// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v79 ida=0x5cd1a2
func TestServerStatusRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerStatusRequest{worldId: world.Id(3)}
			output := ServerStatusRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
		})
	}
}
