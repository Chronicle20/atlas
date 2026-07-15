package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 GUILD_OPERATION family verification — CWvsContext::OnGuildResult @0x725559
// (GMS_v48_1_DEVM.exe, port 13337). Every case byte and body below was read
// directly from the v48 switch; the mode→arm mapping is the v48 seed template's
// guild operations table (writer[29] of template_gms_48_1.json), cross-checked
// against the switch bodies (task-113 v48 close-I).
//
// The v48 guild wire is BYTE-IDENTICAL to v83 for every arm Atlas implements
// EXCEPT the guild-alliance additions, which are absent pre-v61 (IDA-verified):
//   - GUILDMEMBER is 33 bytes (GMS_v48 GUILDMEMBER::Decode@0x49c982
//     DecodeBuffer(0x21)) vs 37 bytes (GMS_v61 @0x4b54f6 DecodeBuffer(37)); the
//     4-byte delta is the trailing AllianceTitle int. Affects MemberJoined + the
//     per-member records inside Info.
//   - GUILDDATA (Info) reads ONE trailing int after the notice (points) at v48
//     (@0x49ca86) vs two (points + allianceId) at v61+/v83.
// Both are gated on GMS < 61 (model.GuildMember + guild/clientbound/info.go).
// v28 is unverified-by-inference (no v28 IDB) — folded into the v48 legacy shape.
//
// Every non-alliance data arm is asserted byte-equal to the IDA-verified v83
// encode (cross-version equality — the established door/party-family discipline;
// the shared codecs carry only the Invite<84 trailing-ints gate, off for v48/v83,
// and the <61 alliance gate, handled by the transform assertions below).
//
// v48-ABSENT: BoardAuthKeyUpdate (guild-BBS board auth key) has no v48
// OnGuildResult case — the v48 client has no guild web board (GUILD_BBS_PACKET is
// absent) — so it is n-a'd (its stray stage-D report is removed) and carries no
// fixture/marker/evidence.

// packet-audit:verify packet=guild/clientbound/GuildRequestName version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildRequestEmblem version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildCreateErrorNameInUse version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildCreateErrorDisagreed version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildCreateError version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildJoinErrorAlreadyJoined version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildJoinErrorMaxMembers version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildJoinErrorNotInChannel version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildMemberQuitErrorNotInGuild version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildMemberExpelledErrorNotInGuild version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildDisbandError version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildCreateErrorCannotAsAdmin version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildIncreaseCapacityError version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildQuestErrorLessThanSixMembers version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildQuestErrorDisconnected version=gms_v48 ida=0x725559
func TestGuildModeOnlyArmsV48(t *testing.T) {
	cases := map[string]struct {
		mode byte
		got  []byte
	}{
		"RequestName":                   {1, NewRequestName(1).Encode(nil, nil)(nil)},
		"RequestEmblem":                 {17, NewRequestEmblem(17).Encode(nil, nil)(nil)},
		"CreateErrorNameInUse":          {28, NewCreateErrorNameInUse(28).Encode(nil, nil)(nil)},
		"CreateErrorDisagreed":          {36, NewCreateErrorDisagreed(36).Encode(nil, nil)(nil)},
		"CreateError":                   {38, NewCreateError(38).Encode(nil, nil)(nil)},
		"JoinErrorAlreadyJoined":        {40, NewJoinErrorAlreadyJoined(40).Encode(nil, nil)(nil)},
		"JoinErrorMaxMembers":           {41, NewJoinErrorMaxMembers(41).Encode(nil, nil)(nil)},
		"JoinErrorNotInChannel":         {42, NewJoinErrorNotInChannel(42).Encode(nil, nil)(nil)},
		"MemberQuitErrorNotInGuild":     {45, NewMemberQuitErrorNotInGuild(45).Encode(nil, nil)(nil)},
		"MemberExpelledErrorNotInGuild": {48, NewMemberExpelledErrorNotInGuild(48).Encode(nil, nil)(nil)},
		"DisbandError":                  {52, NewDisbandError(52).Encode(nil, nil)(nil)},
		"CreateErrorCannotAsAdmin":      {56, NewCreateErrorCannotAsAdmin(56).Encode(nil, nil)(nil)},
		"IncreaseCapacityError":         {59, NewIncreaseCapacityError(59).Encode(nil, nil)(nil)},
		"QuestErrorLessThanSixMembers":  {74, NewQuestErrorLessThanSixMembers(74).Encode(nil, nil)(nil)},
		"QuestErrorDisconnected":        {75, NewQuestErrorDisconnected(75).Encode(nil, nil)(nil)},
	}
	for name, c := range cases {
		if !bytes.Equal(c.got, []byte{c.mode}) {
			t.Errorf("v48 mode-only %s: got % x want %02x", name, c.got, c.mode)
		}
	}
}

// packet-audit:verify packet=guild/clientbound/GuildInviteErrorNotAcceptingInvites version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildInviteErrorAnotherInvite version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildInviteDenied version=gms_v48 ida=0x725559
func TestGuildTargetBearingArmsV48(t *testing.T) {
	// WriteByte(mode) + WriteAsciiString("Bob") (v48 cases 53/54/55: DecodeStr).
	bob := []byte{0x03, 0x00, 0x42, 0x6F, 0x62}
	want := func(mode byte) []byte { return append([]byte{mode}, bob...) }
	if got := NewInviteErrorNotAcceptingInvites(53, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(53)) {
		t.Errorf("v48 InviteErrorNotAcceptingInvites: got % x want % x", got, want(53))
	}
	if got := NewInviteErrorAnotherInvite(54, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(54)) {
		t.Errorf("v48 InviteErrorAnotherInvite: got % x want % x", got, want(54))
	}
	if got := NewInviteDenied(55, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(55)) {
		t.Errorf("v48 InviteDenied: got % x want % x", got, want(55))
	}
}

// packet-audit:verify packet=guild/clientbound/GuildRequestAgreement version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildInvite version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildMemberLeft version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildMemberExpel version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildDisband version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildCapacityChange version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildMemberUpdate version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildMemberStatusUpdate version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildTitleChange version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildMemberTitleUpdate version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildEmblemChange version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildNoticeChange version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildShowTitles version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildQuestWaitingNotice version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildSetSkillResponse version=gms_v48 ida=0x725559
func TestGuildStableDataArmsV48(t *testing.T) {
	v48 := pt.CreateContext("GMS", 48, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	titles := [5]string{"Master", "Jr. Master", "Member", "Newbie", "Recruit"}
	showEntries := []GuildTitleEntry{
		{Name: "Alice", Values: [5]uint32{1, 2, 3, 4, 5}},
		{Name: "Bob", Values: [5]uint32{6, 7, 8, 9, 10}},
	}
	eq := func(name string, a, b []byte) {
		if !bytes.Equal(a, b) {
			t.Errorf("%s v48 != v83\n v48: % x\n v83: % x", name, a, b)
		}
	}
	eq("RequestAgreement", NewRequestAgreement(3, 100, "Leader", "Guild").Encode(nil, v48)(nil), NewRequestAgreement(3, 100, "Leader", "Guild").Encode(nil, v83)(nil))
	eq("Invite", NewInvite(5, 500, "Inviter", 7, 9).Encode(nil, v48)(nil), NewInvite(5, 500, "Inviter", 7, 9).Encode(nil, v83)(nil))
	eq("MemberLeft", NewMemberLeft(44, 500, 1001, "Leaver").Encode(nil, v48)(nil), NewMemberLeft(44, 500, 1001, "Leaver").Encode(nil, v83)(nil))
	eq("MemberExpel", NewMemberExpel(47, 500, 1001, "Expelled").Encode(nil, v48)(nil), NewMemberExpel(47, 500, 1001, "Expelled").Encode(nil, v83)(nil))
	eq("Disband", NewDisband(50, 500).Encode(nil, v48)(nil), NewDisband(50, 500).Encode(nil, v83)(nil))
	eq("CapacityChange", NewCapacityChange(58, 500, 100).Encode(nil, v48)(nil), NewCapacityChange(58, 500, 100).Encode(nil, v83)(nil))
	eq("MemberUpdate", NewMemberUpdate(60, 500, 1001, 50, 100).Encode(nil, v48)(nil), NewMemberUpdate(60, 500, 1001, 50, 100).Encode(nil, v83)(nil))
	eq("MemberStatusUpdate", NewMemberStatusUpdate(61, 500, 1001, true).Encode(nil, v48)(nil), NewMemberStatusUpdate(61, 500, 1001, true).Encode(nil, v83)(nil))
	eq("TitleChange", NewTitleChange(62, 500, titles).Encode(nil, v48)(nil), NewTitleChange(62, 500, titles).Encode(nil, v83)(nil))
	eq("MemberTitleUpdate", NewMemberTitleUpdate(64, 500, 1001, 3).Encode(nil, v48)(nil), NewMemberTitleUpdate(64, 500, 1001, 3).Encode(nil, v83)(nil))
	eq("EmblemChange", NewEmblemChange(66, 500, 3, 2, 5, 4).Encode(nil, v48)(nil), NewEmblemChange(66, 500, 3, 2, 5, 4).Encode(nil, v83)(nil))
	eq("NoticeChange", NewNoticeChange(68, 500, "Guild notice").Encode(nil, v48)(nil), NewNoticeChange(68, 500, "Guild notice").Encode(nil, v83)(nil))
	eq("ShowTitles", NewShowTitles(73, 500, showEntries).Encode(nil, v48)(nil), NewShowTitles(73, 500, showEntries).Encode(nil, v83)(nil))
	eq("QuestWaitingNotice", NewQuestWaitingNotice(76, 3, 1).Encode(nil, v48)(nil), NewQuestWaitingNotice(76, 3, 1).Encode(nil, v83)(nil))
	eq("SetSkillResponse", NewSetSkillResponse(77, true, "ok").Encode(nil, v48)(nil), NewSetSkillResponse(77, true, "ok").Encode(nil, v83)(nil))
}

// TestGuildAllianceGatedArmsV48 verifies the two alliance-divergent arms: v48
// MemberJoined drops the trailing AllianceTitle int (33B GUILDMEMBER vs 37B), and
// v48 Info drops the per-member AllianceTitle AND the trailing allianceId.
// packet-audit:verify packet=guild/clientbound/GuildMemberJoined version=gms_v48 ida=0x725559
// packet-audit:verify packet=guild/clientbound/GuildInfo version=gms_v48 ida=0x725559
func TestGuildAllianceGatedArmsV48(t *testing.T) {
	v48 := pt.CreateContext("GMS", 48, 1)
	v83 := pt.CreateContext("GMS", 83, 1)

	// MemberJoined: v48 == v83 with the trailing 4-byte AllianceTitle removed.
	{
		got := NewMemberJoined(39, 500, 1001, "Joiner", 100, 50, 2, true, 1).Encode(nil, v48)(nil)
		v83b := NewMemberJoined(39, 500, 1001, "Joiner", 100, 50, 2, true, 1).Encode(nil, v83)(nil)
		if want := v83b[:len(v83b)-4]; !bytes.Equal(got, want) {
			t.Errorf("MemberJoined v48 != v83-minus-allianceTitle\n v48: % x\n want: % x", got, want)
		}
		// mode(1)+guildId(4)+cid(4)+GUILDMEMBER(33) = 42
		if len(got) != 42 {
			t.Errorf("MemberJoined v48 byte count: got %d want 42", len(got))
		}
	}

	// Info: v48 drops 4 bytes per member (AllianceTitle) + 4 trailing (allianceId).
	{
		titles := [5]string{"Master", "Jr. Master", "Member", "Newbie", "Recruit"}
		members := []GuildMemberInfo{
			{CharacterId: 1001, Name: "M1", JobId: 100, Level: 50, Title: 1, Online: true, Signature: 0, AllianceTitle: 1},
			{CharacterId: 1002, Name: "M2", JobId: 200, Level: 70, Title: 2, Online: false, Signature: 0, AllianceTitle: 2},
		}
		got := NewInfo(true, 500, "Guild", titles, members, 100, 5, 4, 3, 2, "notice", 1000, 7).Encode(nil, v48)(nil)
		v83b := NewInfo(true, 500, "Guild", titles, members, 100, 5, 4, 3, 2, "notice", 1000, 7).Encode(nil, v83)(nil)
		if want := len(v83b) - 4*len(members) - 4; len(got) != want {
			t.Errorf("Info v48 byte count: got %d want %d (v83 %d - 4/member - 4 allianceId)", len(got), want, len(v83b))
		}
	}
}

// TestGuildDataArmsV48RoundTrip proves the alliance-gated codecs are symmetric at
// v48: pt.RoundTrip fails if the legacy decode leaves any of the legacy-encoded
// bytes unconsumed (or reads past them), i.e. a real encode/decode gate mismatch.
func TestGuildDataArmsV48RoundTrip(t *testing.T) {
	v48 := pt.CreateContext("GMS", 48, 1)

	in := NewMemberJoined(39, 500, 1001, "Joiner", 100, 50, 2, true, 0)
	out := MemberJoined{}
	pt.RoundTrip(t, v48, in.Encode, out.Decode, nil)

	titles := [5]string{"Master", "Jr. Master", "Member", "Newbie", "Recruit"}
	members := []GuildMemberInfo{
		{CharacterId: 1001, Name: "M1", JobId: 100, Level: 50, Title: 1, Online: true, Signature: 0, AllianceTitle: 0},
	}
	inInfo := NewInfo(true, 500, "Guild", titles, members, 100, 5, 4, 3, 2, "notice", 1000, 0)
	outInfo := Info{}
	pt.RoundTrip(t, v48, inInfo.Encode, outInfo.Decode, nil)
}
