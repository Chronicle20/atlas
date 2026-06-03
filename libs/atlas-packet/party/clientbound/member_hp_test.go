package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestPartyMemberHPByteOutput verifies the byte output of MemberHP across all
// tenant variants. IDA CUserRemote::OnReceiveHP@0x953f50 reads Decode4(hp)+Decode4(maxHp);
// characterId is consumed upstream by CUserPool::OnUserRemotePacket (dispatcher-prefix).
// Expected wire: WriteInt(characterId=4) + WriteInt(hp=4) + WriteInt(maxHp=4) = 12 bytes,
// version-independent (no gate in encoder).
func TestPartyMemberHPByteOutput(t *testing.T) {
	const wantBytes = 12 // characterId(4) + hp(4) + maxHp(4)
	input := NewPartyMemberHP(1234, 5000, 10000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			if len(got) != wantBytes {
				t.Errorf("byte count: got %d, want %d", len(got), wantBytes)
			}
		})
	}
}

func TestPartyMemberHP(t *testing.T) {
	input := NewPartyMemberHP(1234, 5000, 10000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
