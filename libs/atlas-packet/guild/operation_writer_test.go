package guild

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
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

func TestInviteWRoundTrip(t *testing.T) {
	input := NewInviteW(0x05, 500, "InviterName")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := InviteW{}
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
