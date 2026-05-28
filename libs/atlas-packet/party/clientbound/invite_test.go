package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestInviteByteOutput verifies the byte output of Invite across all tenant variants.
// IDA evidence:
//   v83 OnPartyResult@0xa3e31c case 4: Decode4(partyId)+DecodeStr(name)+Decode1(autoJoin)
//        — no originatorJobId/Level fields.
//   v95 OnPartyResult: Decode4(partyId)+DecodeStr(name)+Decode4(jobId)+Decode4(level)+Decode1(autoJoin).
// Wire layout: mode(1)+partyId(4)+name(2+len)+[jobId(4)+level(4)]+autoJoin(1).
// originatorName="PartyLeader" → 2+11=13 bytes.
//   v83: 1+4+13+1 = 19 bytes
//   v95: 1+4+13+4+4+1 = 27 bytes
func TestInviteByteOutput(t *testing.T) {
	cases := []struct {
		variant   pt.TenantVariant
		wantBytes int
	}{
		{pt.Variants[0], 19}, // GMS v28  — no jobId/level
		{pt.Variants[1], 19}, // GMS v83  — no jobId/level
		{pt.Variants[2], 19}, // GMS v87  — no jobId/level
		{pt.Variants[3], 27}, // GMS v95  — with jobId+level
		{pt.Variants[4], 27}, // JMS v185 — with jobId+level
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			input := NewInvite(16, 5000, "PartyLeader", 100, 50)
			got := input.Encode(nil, ctx)(nil)
			if len(got) != tc.wantBytes {
				t.Errorf("byte count: got %d, want %d", len(got), tc.wantBytes)
			}
		})
	}
}

func TestInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			v95plus := (v.Region == "GMS" && v.MajorVersion >= 95) || v.Region == "JMS"
			input := NewInvite(16, 5000, "PartyLeader", 100, 50)
			output := Invite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
			if output.OriginatorName() != input.OriginatorName() {
				t.Errorf("originatorName: got %v, want %v", output.OriginatorName(), input.OriginatorName())
			}
			if v95plus {
				if output.OriginatorJobId() != input.OriginatorJobId() {
					t.Errorf("originatorJobId: got %v, want %v", output.OriginatorJobId(), input.OriginatorJobId())
				}
				if output.OriginatorLevel() != input.OriginatorLevel() {
					t.Errorf("originatorLevel: got %v, want %v", output.OriginatorLevel(), input.OriginatorLevel())
				}
			}
		})
	}
}
