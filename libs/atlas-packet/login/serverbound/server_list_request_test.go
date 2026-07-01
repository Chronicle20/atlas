package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v72: the SERVERLIST_REQUEST send = sub_5B0067 @0x5b0067 (GMS_v72.1_U_DEVM.exe,
// port 13339): COutPacket(4) @0x5b0232 then SendPacket @0x5b0248 with NO Encode*
// calls between → empty body. Matches the codec's empty Encode. Marker-only (tier-0).
//
// packet-audit:verify packet=login/serverbound/ServerListRequest version=gms_v72 ida=0x5b0067
// packet-audit:verify packet=login/serverbound/ServerListRequest version=gms_v95 ida=0x5d9730
// packet-audit:verify packet=login/serverbound/ServerListRequest version=gms_v87 ida=0x62c951
// packet-audit:verify packet=login/serverbound/ServerListRequest version=jms_v185 ida=0x66c55a
// packet-audit:verify packet=login/serverbound/ServerListRequest version=gms_v83 ida=0x5f53c0
// packet-audit:verify packet=login/serverbound/ServerListRequest version=gms_v84 ida=0x609165
// packet-audit:verify packet=login/serverbound/ServerListRequest version=gms_v79 ida=0x5d0c27
func TestServerListRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerListRequest{}
			output := ServerListRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
