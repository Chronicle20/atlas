package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ---------------------------------------------------------------------------
// Per-version byte fixtures for the discrete GuildOperation arms.
//
// Mode bytes are taken from docs/packets/dispatchers/guild.yaml (IDA-enumerated,
// task-103). v83/v84/v87/jms are byte-identical; v95 mode bytes are shifted
// (non-uniform). Read orders are cited per struct in operation.go (v83
// OnGuildResult@0xa37490; v84@0xa82e2b; v87@0xacf7d3; v95@0xa0d3b0; jms@0xb22518).
//
// Structural-arm verify markers (carried forward from the prior test file).
// packet-audit:verify packet=guild/clientbound/GuildCapacityChange version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildCapacityChange version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildMemberLeft version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildTitleChange version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildRequestAgreement version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildNoticeChange version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildMemberTitleUpdate version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildMemberStatusUpdate version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildDisband version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildEmblemChange version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildMemberJoined version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildMemberExpel version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildInvite version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildMemberLeft version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildTitleChange version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildRequestAgreement version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildNoticeChange version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildMemberTitleUpdate version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildMemberStatusUpdate version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildDisband version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildEmblemChange version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildMemberJoined version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildMemberExpel version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/GuildCapacityChange version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildDisband version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildEmblemChange version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildMemberExpel version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildMemberJoined version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildMemberLeft version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildMemberStatusUpdate version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildMemberTitleUpdate version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildNoticeChange version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildRequestAgreement version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildTitleChange version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildCapacityChange version=gms_v95 ida=0xa0dfe2
// packet-audit:verify packet=guild/clientbound/GuildDisband version=gms_v95 ida=0xa0dfcb
// packet-audit:verify packet=guild/clientbound/GuildEmblemChange version=gms_v95 ida=0xa0e394
// packet-audit:verify packet=guild/clientbound/GuildInvite version=gms_v95 ida=0xa0d664
// packet-audit:verify packet=guild/clientbound/GuildMemberExpel version=gms_v95 ida=0xa0dd06
// packet-audit:verify packet=guild/clientbound/GuildMemberJoined version=gms_v95 ida=0xa0dbc0
// packet-audit:verify packet=guild/clientbound/GuildMemberLeft version=gms_v95 ida=0xa0dd06
// packet-audit:verify packet=guild/clientbound/GuildMemberStatusUpdate version=gms_v95 ida=0xa0e563
// packet-audit:verify packet=guild/clientbound/GuildMemberTitleUpdate version=gms_v95 ida=0xa0e0b5
// packet-audit:verify packet=guild/clientbound/GuildNoticeChange version=gms_v95 ida=0xa0e44b
// packet-audit:verify packet=guild/clientbound/GuildTitleChange version=gms_v95 ida=0xa0e239
// packet-audit:verify packet=guild/clientbound/GuildMemberJoined version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildNoticeChange version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildTitleChange version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildCapacityChange version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildDisband version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildEmblemChange version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildInvite version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildMemberStatusUpdate version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildMemberTitleUpdate version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildMemberExpel version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildMemberLeft version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/GuildRequestAgreement version=gms_v84 ida=0xa82e2b

// modeOnlyArmModes maps a fixture version label → the mode byte from guild.yaml
// for a given arm. Used by the mode-only fixtures below; the wire is exactly the
// one mode byte.

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

// TestInviteByteOutput verifies the byte output of guild Invite across all tenant
// variants. F3 (task-103): v84 reads the 2 trailing ints like v87+, NOT v83.
//
//	v83/v86 (boundary <84): mode(1)+guildId(4)+name(2+len) only
//	v84/v87/v95/jms (boundary >=84 or JMS): + unknown(4)+skillId(4)
//
// originatorName="InviterName" → 2+11=13 bytes.
//
//	<84:  1+4+13 = 18 bytes ; >=84: 1+4+13+4+4 = 26 bytes
//
// packet-audit:verify packet=guild/clientbound/GuildInvite version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildInvite version=gms_v87 ida=0xacf7d3
func TestInviteByteOutput(t *testing.T) {
	cases := []struct {
		variant   pt.TenantVariant
		wantBytes int
	}{
		{pt.Variants[0], 18}, // GMS v28  — pre-84, no trailing ints
		{pt.Variants[1], 18}, // GMS v83  — no trailing ints (IDA v83@0xa37490 L1319-1320)
		{pt.Variants[2], 26}, // GMS v87  — trailing ints
		{pt.Variants[3], 26}, // GMS v95  — trailing ints
		{pt.Variants[4], 26}, // JMS v185 — trailing ints
		{pt.Variants[5], 26}, // GMS v84  — trailing ints (IDA v84@0xa82e2b L1212-1216, F3)
		{pt.Variants[6], 18}, // GMS v86  — pre-... wait, boundary is 84 so v86>=84 → 26
	}
	// v86 (>=84) also reads the trailing ints; correct the expectation.
	cases[6].wantBytes = 26
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
	input := NewInvite(0x05, 500, "InviterName", 7, 9)
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

// --- Mode-only arm byte fixtures (one mode byte; per guild.yaml) --------------

// modeOnlyFixture asserts a discrete mode-only struct encodes to exactly its mode byte.
func modeOnlyFixture(t *testing.T, mode byte, enc func(byte) []byte) {
	t.Helper()
	got := enc(mode)
	want := []byte{mode}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// packet-audit:verify packet=guild/clientbound/RequestName version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/RequestName version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/RequestName version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/RequestName version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/RequestName version=jms_v185 ida=0xb22518
func TestGuildRequestName(t *testing.T) {
	// gms_v83/84/87/jms mode 0x01; v95 mode 0x01 (no shift at <=0x11).
	for _, mode := range []byte{0x01} {
		m := NewRequestName(mode)
		modeOnlyFixture(t, mode, func(b byte) []byte { return m.Encode(nil, nil)(nil) })
	}
}

// packet-audit:verify packet=guild/clientbound/RequestEmblem version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/RequestEmblem version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/RequestEmblem version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/RequestEmblem version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/RequestEmblem version=jms_v185 ida=0xb22518
func TestGuildRequestEmblem(t *testing.T) {
	for _, mode := range []byte{0x11} {
		m := NewRequestEmblem(mode)
		modeOnlyFixture(t, mode, func(b byte) []byte { return m.Encode(nil, nil)(nil) })
	}
}

// modeOnlyArmCase couples a struct's per-version mode bytes to its encoder.
type modeOnlyArmCase struct {
	v83, v95 byte
	encode   func(byte) []byte
}

// packet-audit:verify packet=guild/clientbound/CreateErrorNameInUse version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/CreateErrorNameInUse version=gms_v84 ida=0xa82e2b
// packet-audit:verify packet=guild/clientbound/CreateErrorNameInUse version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/CreateErrorNameInUse version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/CreateErrorNameInUse version=jms_v185 ida=0xb22518
// packet-audit:verify packet=guild/clientbound/CreateErrorDisagreed version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/CreateErrorDisagreed version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/CreateError version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/CreateError version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/JoinErrorAlreadyJoined version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/JoinErrorAlreadyJoined version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/JoinErrorMaxMembers version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/JoinErrorMaxMembers version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/JoinErrorNotInChannel version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/JoinErrorNotInChannel version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/MemberQuitErrorNotInGuild version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/MemberQuitErrorNotInGuild version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/MemberExpelledErrorNotInGuild version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/MemberExpelledErrorNotInGuild version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/DisbandError version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/DisbandError version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/CreateErrorCannotAsAdmin version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/CreateErrorCannotAsAdmin version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/IncreaseCapacityError version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/IncreaseCapacityError version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/QuestErrorLessThanSixMembers version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/QuestErrorLessThanSixMembers version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/QuestErrorDisconnected version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/QuestErrorDisconnected version=gms_v95 ida=0xa0d3b0
func TestModeOnlyErrorArms(t *testing.T) {
	cases := map[string]modeOnlyArmCase{
		"CreateErrorNameInUse":          {0x1C, 0x1E, func(b byte) []byte { m := NewCreateErrorNameInUse(b); return m.Encode(nil, nil)(nil) }},
		"CreateErrorDisagreed":          {0x24, 0x26, func(b byte) []byte { m := NewCreateErrorDisagreed(b); return m.Encode(nil, nil)(nil) }},
		"CreateError":                   {0x26, 0x28, func(b byte) []byte { m := NewCreateError(b); return m.Encode(nil, nil)(nil) }},
		"JoinErrorAlreadyJoined":        {0x28, 0x2A, func(b byte) []byte { m := NewJoinErrorAlreadyJoined(b); return m.Encode(nil, nil)(nil) }},
		"JoinErrorMaxMembers":           {0x29, 0x2B, func(b byte) []byte { m := NewJoinErrorMaxMembers(b); return m.Encode(nil, nil)(nil) }},
		"JoinErrorNotInChannel":         {0x2A, 0x2C, func(b byte) []byte { m := NewJoinErrorNotInChannel(b); return m.Encode(nil, nil)(nil) }},
		"MemberQuitErrorNotInGuild":     {0x2D, 0x2F, func(b byte) []byte { m := NewMemberQuitErrorNotInGuild(b); return m.Encode(nil, nil)(nil) }},
		"MemberExpelledErrorNotInGuild": {0x30, 0x32, func(b byte) []byte { m := NewMemberExpelledErrorNotInGuild(b); return m.Encode(nil, nil)(nil) }},
		"DisbandError":                  {0x34, 0x36, func(b byte) []byte { m := NewDisbandError(b); return m.Encode(nil, nil)(nil) }},
		"CreateErrorCannotAsAdmin":      {0x38, 0x3A, func(b byte) []byte { m := NewCreateErrorCannotAsAdmin(b); return m.Encode(nil, nil)(nil) }},
		"IncreaseCapacityError":         {0x3B, 0x3D, func(b byte) []byte { m := NewIncreaseCapacityError(b); return m.Encode(nil, nil)(nil) }},
		"QuestErrorLessThanSixMembers":  {0x4A, 0x4D, func(b byte) []byte { m := NewQuestErrorLessThanSixMembers(b); return m.Encode(nil, nil)(nil) }},
		"QuestErrorDisconnected":        {0x4B, 0x4E, func(b byte) []byte { m := NewQuestErrorDisconnected(b); return m.Encode(nil, nil)(nil) }},
	}
	for name, c := range cases {
		t.Run(name+"/gms_v83", func(t *testing.T) { modeOnlyFixture(t, c.v83, c.encode) })
		t.Run(name+"/gms_v95", func(t *testing.T) { modeOnlyFixture(t, c.v95, c.encode) })
	}
}

// --- Target-bearing invite-error arm fixtures ({mode,target}) -----------------

// packet-audit:verify packet=guild/clientbound/InviteErrorNotAcceptingInvites version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/InviteErrorNotAcceptingInvites version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/InviteErrorAnotherInvite version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/InviteErrorAnotherInvite version=gms_v95 ida=0xa0d3b0
// packet-audit:verify packet=guild/clientbound/InviteDenied version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/InviteDenied version=gms_v95 ida=0xa0d3b0
func TestTargetBearingInviteErrors(t *testing.T) {
	// mode byte + 2-byte ascii length prefix + "Bob" (3 bytes) = 6 bytes total.
	want := func(mode byte) []byte { return []byte{mode, 0x03, 0x00, 'B', 'o', 'b'} }
	cases := []struct {
		name     string
		v83, v95 byte
		encode   func(byte) []byte
	}{
		{"InviteErrorNotAcceptingInvites", 0x35, 0x37, func(b byte) []byte {
			m := NewInviteErrorNotAcceptingInvites(b, "Bob")
			return m.Encode(nil, nil)(nil)
		}},
		{"InviteErrorAnotherInvite", 0x36, 0x38, func(b byte) []byte {
			m := NewInviteErrorAnotherInvite(b, "Bob")
			return m.Encode(nil, nil)(nil)
		}},
		{"InviteDenied", 0x37, 0x39, func(b byte) []byte {
			m := NewInviteDenied(b, "Bob")
			return m.Encode(nil, nil)(nil)
		}},
	}
	for _, c := range cases {
		t.Run(c.name+"/gms_v83", func(t *testing.T) {
			if got := c.encode(c.v83); !bytes.Equal(got, want(c.v83)) {
				t.Fatalf("got %v want %v", got, want(c.v83))
			}
		})
		t.Run(c.name+"/gms_v95", func(t *testing.T) {
			if got := c.encode(c.v95); !bytes.Equal(got, want(c.v95)) {
				t.Fatalf("got %v want %v", got, want(c.v95))
			}
		})
	}
}

// --- Structured arms previously without a discrete struct ----------------------

// packet-audit:verify packet=guild/clientbound/MemberUpdate version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/MemberUpdate version=gms_v95 ida=0xa0d3b0
func TestGuildMemberUpdateRoundTrip(t *testing.T) {
	input := NewMemberUpdate(0x3C, 500, 1001, 50, 100)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MemberUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// packet-audit:verify packet=guild/clientbound/ShowTitles version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/ShowTitles version=gms_v95 ida=0xa0d3b0
func TestGuildShowTitlesRoundTrip(t *testing.T) {
	entries := []GuildTitleEntry{
		{Name: "Alice", Values: [5]uint32{1, 2, 3, 4, 5}},
		{Name: "Bob", Values: [5]uint32{6, 7, 8, 9, 10}},
	}
	input := NewShowTitles(0x49, 500, entries)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ShowTitles{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// packet-audit:verify packet=guild/clientbound/QuestWaitingNotice version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/QuestWaitingNotice version=gms_v95 ida=0xa0d3b0
func TestGuildQuestWaitingNoticeRoundTrip(t *testing.T) {
	input := NewQuestWaitingNotice(0x4C, 3, 1)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := QuestWaitingNotice{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// packet-audit:verify packet=guild/clientbound/BoardAuthKeyUpdate version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/BoardAuthKeyUpdate version=gms_v95 ida=0xa0d3b0
func TestGuildBoardAuthKeyUpdateRoundTrip(t *testing.T) {
	input := NewBoardAuthKeyUpdate(0x4D, "auth-key-123")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := BoardAuthKeyUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// packet-audit:verify packet=guild/clientbound/SetSkillResponse version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/SetSkillResponse version=gms_v95 ida=0xa0d3b0
func TestGuildSetSkillResponseRoundTrip(t *testing.T) {
	for _, success := range []bool{true, false} {
		input := NewSetSkillResponse(0x4E, success, "ok")
		for _, v := range pt.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				output := SetSkillResponse{}
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			})
		}
	}
}
