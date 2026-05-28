package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestRequestAgreementRoundTrip(t *testing.T) {
	input := NewRequestAgreement(0x01, 100, "LeaderName", "GuildName")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := RequestAgreement{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestErrorMessageRoundTrip(t *testing.T) {
	input := NewErrorMessage(0x2A)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ErrorMessage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestErrorMessageWithTargetRoundTrip(t *testing.T) {
	input := NewErrorMessageWithTarget(0x2B, "TargetPlayer")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ErrorMessageWithTarget{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestEmblemChangeRoundTrip(t *testing.T) {
	input := NewEmblemChange(0x11, 500, 3, 2, 5, 4)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := EmblemChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestMemberStatusUpdateRoundTrip(t *testing.T) {
	input := NewMemberStatusUpdate(0x0F, 500, 1001, true)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MemberStatusUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestMemberTitleUpdateRoundTrip(t *testing.T) {
	input := NewMemberTitleUpdate(0x10, 500, 1001, 3)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MemberTitleUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestNoticeChangeRoundTrip(t *testing.T) {
	input := NewNoticeChange(0x0E, 500, "Guild notice text")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := NoticeChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestMemberLeftRoundTrip(t *testing.T) {
	input := NewMemberLeft(0x0C, 500, 1001, "LeaverName")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MemberLeft{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestMemberExpelRoundTrip(t *testing.T) {
	input := NewMemberExpel(0x0D, 500, 1001, "ExpelledName")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MemberExpel{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestMemberJoinedRoundTrip(t *testing.T) {
	input := NewMemberJoined(0x0B, 500, 1001, "JoinerName", 100, 50, 2, true, 1)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MemberJoined{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestInviteByteOutput verifies the byte output of guild Invite across all tenant variants.
// IDA evidence:
//   v83 OnGuildResult@0xa37490 invite path: Decode4(guildId)+DecodeStr(inviterName)
//        — no unknown/skillId fields.
//   v87 OnGuildResult@0xacf7d3@0xacf9c7: Decode4(guildId)+DecodeStr(inviterName)+Decode4(unknown)+Decode4(skillId)
//        — v87 already reads unknown+skillId; gate widened from v95plus to v84plus (GMS > 83).
//   v95 OnGuildResult: same as v87.
// Wire layout: mode(1)+guildId(4)+name(2+len)+[unknown(4)+skillId(4)].
// originatorName="InviterName" → 2+11=13 bytes.
//   v83:  1+4+13 = 18 bytes
//   v84+: 1+4+13+4+4 = 26 bytes
func TestInviteByteOutput(t *testing.T) {
	cases := []struct {
		variant   pt.TenantVariant
		wantBytes int
	}{
		{pt.Variants[0], 18}, // GMS v28  — no unknown/skillId
		{pt.Variants[1], 18}, // GMS v83  — no unknown/skillId
		{pt.Variants[2], 26}, // GMS v87  — with unknown+skillId (IDA confirmed v87@0xacf9c7)
		{pt.Variants[3], 26}, // GMS v95  — with unknown+skillId
		{pt.Variants[4], 26}, // JMS v185 — with unknown+skillId
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			input := NewInvite(0x05, 500, "InviterName", 0, 0)
			got := input.Encode(nil, ctx)(nil)
			if len(got) != tc.wantBytes {
				t.Errorf("byte count: got %d, want %d", len(got), tc.wantBytes)
			}
		})
	}
}

func TestInviteRoundTrip(t *testing.T) {
	input := NewInvite(0x05, 500, "InviterName", 0, 0)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Invite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestTitleChangeRoundTrip(t *testing.T) {
	titles := [5]string{"Master", "Jr. Master", "Member", "Newbie", "Recruit"}
	input := NewTitleChange(0x12, 500, titles)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := TitleChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestDisbandRoundTrip(t *testing.T) {
	input := NewDisband(0x1A, 500)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Disband{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestCapacityChangeRoundTrip(t *testing.T) {
	input := NewCapacityChange(0x13, 500, 100)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := CapacityChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
