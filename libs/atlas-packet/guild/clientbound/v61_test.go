package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 GUILD family verification (GMS_v61.1_U_DEVM.exe, port 13338).
//
//   - GUILD_OPERATION (CWvsContext::OnGuildResult @0x851543, switch(Decode1(mode))):
//     the v61 mode table was decompiled case-by-case and is BYTE-IDENTICAL to
//     v72/v79/v83 for every arm Atlas implements. Confirmed case labels:
//     0x01(RequestName→InputGuildName), 0x03(RequestAgreement, Decode4+DecodeStr
//     +DecodeStr), 0x05(Invite, Decode4+DecodeStr), 0x11(RequestEmblem→
//     SendSetGuildMarkMsg), 0x1A(guild-loaded, GUILDDATA::Decode), 0x1C(mode-only),
//     0x24(CreateErrorDisagreed, NPCSay), 0x26(CreateError), 0x27(MemberJoined,
//     Decode4+Decode4+GUILDMEMBER::Decode), 0x28/0x29/0x2A(join errors, mode-only),
//     0x2C/0x2F(MemberLeft/MemberExpel, Decode4+Decode4+DecodeStr), 0x2D/0x30
//     (mode-only), 0x32(Disband, Decode4), 0x34(DisbandError), 0x35/0x36/0x37
//     (invite errors, DecodeStr target), 0x38(CreateErrorCannotAsAdmin), 0x3B
//     (IncreaseCapacityError), 0x3C(MemberUpdate, Decode4+Decode4+Decode1), 0x3E
//     (TitleChange, 5×DecodeStr), 0x40(MemberStatus/TitleUpdate, Decode4+Decode1),
//     0x42(EmblemChange, Decode2+Decode1+Decode2+Decode1), 0x44(NoticeChange,
//     DecodeStr), 0x48(CapacityChange, Decode4), 0x49(ShowTitles), 0x4A/0x4B
//     (quest errors, mode-only), 0x4C(QuestWaitingNotice, Decode1+Decode4), 0x4D
//     (BoardAuthKeyUpdate, DecodeStr), 0x4E(SetSkillResponse, Decode1[+DecodeStr]).
//     The only version gate in the package is Invite's trailing ints (boundary 84,
//     operation.go:769) and v61<84 → no trailing ints, exactly like v72/v79/v83.
//   - GUILD_BBS_PACKET (CUIGuildBBS::OnGuildBBSPacket @0x8399af → sub_6BB663,
//     switch(Decode1-6) → modes 6/7/8): sub_6BB6A0 list, sub_6BB9E2 thread, and
//     mode 8 sub_6BBD05(this) = EntryNotFound (mode-only, no wire read).
//   - GUILD_NAME_CHANGED (CUserRemote::OnGuildNameChanged @0x7cc166): DecodeStr(name)
//     only (charId supplied by the user-pool router before dispatch).
//   - GUILD_MARK_CHANGED (CUserRemote::OnGuildMarkChanged @0x7cc1b1): Decode2(bg)
//     +Decode1(bgColor)+Decode2(logo)+Decode1(logoColor).
//
// Mode-only / target arms assert their exact mode byte; data arms assert the v61
// encode is byte-equal to the IDA-verified v83 encode (cross-version equality, the
// established door/SpawnDoor + party-family discipline), valid because GMS<84
// shares one code path and the v61 read orders match v83 exactly.

// packet-audit:verify packet=guild/clientbound/GuildRequestName version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildRequestEmblem version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildCreateErrorNameInUse version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildCreateErrorDisagreed version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildCreateError version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildJoinErrorAlreadyJoined version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildJoinErrorMaxMembers version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildJoinErrorNotInChannel version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildMemberQuitErrorNotInGuild version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildMemberExpelledErrorNotInGuild version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildDisbandError version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildCreateErrorCannotAsAdmin version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildIncreaseCapacityError version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildQuestErrorLessThanSixMembers version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildQuestErrorDisconnected version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildBBSEntryNotFound version=gms_v61 ida=0x8399af
func TestGuildModeOnlyArmsV61(t *testing.T) {
	cases := map[string][]byte{
		"RequestName":                   NewRequestName(0x01).Encode(nil, nil)(nil),
		"RequestEmblem":                 NewRequestEmblem(0x11).Encode(nil, nil)(nil),
		"CreateErrorNameInUse":          NewCreateErrorNameInUse(0x1C).Encode(nil, nil)(nil),
		"CreateErrorDisagreed":          NewCreateErrorDisagreed(0x24).Encode(nil, nil)(nil),
		"CreateError":                   NewCreateError(0x26).Encode(nil, nil)(nil),
		"JoinErrorAlreadyJoined":        NewJoinErrorAlreadyJoined(0x28).Encode(nil, nil)(nil),
		"JoinErrorMaxMembers":           NewJoinErrorMaxMembers(0x29).Encode(nil, nil)(nil),
		"JoinErrorNotInChannel":         NewJoinErrorNotInChannel(0x2A).Encode(nil, nil)(nil),
		"MemberQuitErrorNotInGuild":     NewMemberQuitErrorNotInGuild(0x2D).Encode(nil, nil)(nil),
		"MemberExpelledErrorNotInGuild": NewMemberExpelledErrorNotInGuild(0x30).Encode(nil, nil)(nil),
		"DisbandError":                  NewDisbandError(0x34).Encode(nil, nil)(nil),
		"CreateErrorCannotAsAdmin":      NewCreateErrorCannotAsAdmin(0x38).Encode(nil, nil)(nil),
		"IncreaseCapacityError":         NewIncreaseCapacityError(0x3B).Encode(nil, nil)(nil),
		"QuestErrorLessThanSixMembers":  NewQuestErrorLessThanSixMembers(0x4A).Encode(nil, nil)(nil),
		"QuestErrorDisconnected":        NewQuestErrorDisconnected(0x4B).Encode(nil, nil)(nil),
		"BBSEntryNotFound":              NewBBSEntryNotFound(0x08).Encode(nil, nil)(nil),
	}
	want := map[string]byte{
		"RequestName": 0x01, "RequestEmblem": 0x11, "CreateErrorNameInUse": 0x1C,
		"CreateErrorDisagreed": 0x24, "CreateError": 0x26, "JoinErrorAlreadyJoined": 0x28,
		"JoinErrorMaxMembers": 0x29, "JoinErrorNotInChannel": 0x2A, "MemberQuitErrorNotInGuild": 0x2D,
		"MemberExpelledErrorNotInGuild": 0x30, "DisbandError": 0x34, "CreateErrorCannotAsAdmin": 0x38,
		"IncreaseCapacityError": 0x3B, "QuestErrorLessThanSixMembers": 0x4A, "QuestErrorDisconnected": 0x4B,
		"BBSEntryNotFound": 0x08,
	}
	for name, got := range cases {
		if !bytes.Equal(got, []byte{want[name]}) {
			t.Errorf("v61 mode-only %s: got % x want %02x", name, got, want[name])
		}
	}
}

// packet-audit:verify packet=guild/clientbound/GuildInviteErrorNotAcceptingInvites version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildInviteErrorAnotherInvite version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildInviteDenied version=gms_v61 ida=0x851543
func TestGuildTargetBearingArmsV61(t *testing.T) {
	// WriteByte(mode) + WriteAsciiString("Bob") = [mode, 03 00, 'B','o','b'].
	// (v61 cases 0x35/0x36/0x37: DecodeStr(target).)
	bob := []byte{0x03, 0x00, 0x42, 0x6F, 0x62}
	want := func(mode byte) []byte { return append([]byte{mode}, bob...) }
	if got := NewInviteErrorNotAcceptingInvites(0x35, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(0x35)) {
		t.Errorf("v61 InviteErrorNotAcceptingInvites: got % x want % x", got, want(0x35))
	}
	if got := NewInviteErrorAnotherInvite(0x36, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(0x36)) {
		t.Errorf("v61 InviteErrorAnotherInvite: got % x want % x", got, want(0x36))
	}
	if got := NewInviteDenied(0x37, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(0x37)) {
		t.Errorf("v61 InviteDenied: got % x want % x", got, want(0x37))
	}
}

// packet-audit:verify packet=guild/clientbound/GuildRequestAgreement version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildEmblemChange version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildMemberStatusUpdate version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildMemberTitleUpdate version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildNoticeChange version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildMemberLeft version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildMemberExpel version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildMemberJoined version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildInvite version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildTitleChange version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildDisband version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildCapacityChange version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildMemberUpdate version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildShowTitles version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildQuestWaitingNotice version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildBoardAuthKeyUpdate version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildSetSkillResponse version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildInfo version=gms_v61 ida=0x851543
// packet-audit:verify packet=guild/clientbound/GuildBBSThreadList version=gms_v61 ida=0x8399af
// packet-audit:verify packet=guild/clientbound/GuildBBSThread version=gms_v61 ida=0x8399af
func TestGuildDataArmsV61(t *testing.T) {
	v61 := pt.CreateContext("GMS", 61, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	titles := [5]string{"Master", "Jr. Master", "Member", "Newbie", "Recruit"}
	showEntries := []GuildTitleEntry{
		{Name: "Alice", Values: [5]uint32{1, 2, 3, 4, 5}},
		{Name: "Bob", Values: [5]uint32{6, 7, 8, 9, 10}},
	}
	infoMembers := []GuildMemberInfo{
		{CharacterId: 1001, Name: "M1", JobId: 100, Level: 50, Title: 1, Online: true, Signature: 0, AllianceTitle: 1},
		{CharacterId: 1002, Name: "M2", JobId: 200, Level: 70, Title: 2, Online: false, Signature: 0, AllianceTitle: 2},
	}
	notice := BBSThreadSummary{Id: 1, PosterId: 2, Title: "N", CreatedAt: 123, EmoticonId: 3, ReplyCount: 4}
	threads := []BBSThreadSummary{
		{Id: 5, PosterId: 6, Title: "T1", CreatedAt: 456, EmoticonId: 7, ReplyCount: 8},
	}
	replies := []BBSReply{{Id: 9, PosterId: 10, CreatedAt: 789, Message: "r1"}}

	eq := func(name string, v61b, v83b []byte) {
		if !bytes.Equal(v61b, v83b) {
			t.Errorf("%s v61 != v83\n v61: % x\n v83: % x", name, v61b, v83b)
		}
	}
	eq("RequestAgreement", NewRequestAgreement(0x03, 100, "Leader", "Guild").Encode(nil, v61)(nil), NewRequestAgreement(0x03, 100, "Leader", "Guild").Encode(nil, v83)(nil))
	eq("EmblemChange", NewEmblemChange(0x42, 500, 3, 2, 5, 4).Encode(nil, v61)(nil), NewEmblemChange(0x42, 500, 3, 2, 5, 4).Encode(nil, v83)(nil))
	eq("MemberStatusUpdate", NewMemberStatusUpdate(0x40, 500, 1001, true).Encode(nil, v61)(nil), NewMemberStatusUpdate(0x40, 500, 1001, true).Encode(nil, v83)(nil))
	eq("MemberTitleUpdate", NewMemberTitleUpdate(0x40, 500, 1001, 3).Encode(nil, v61)(nil), NewMemberTitleUpdate(0x40, 500, 1001, 3).Encode(nil, v83)(nil))
	eq("NoticeChange", NewNoticeChange(0x44, 500, "Guild notice").Encode(nil, v61)(nil), NewNoticeChange(0x44, 500, "Guild notice").Encode(nil, v83)(nil))
	eq("MemberLeft", NewMemberLeft(0x2C, 500, 1001, "Leaver").Encode(nil, v61)(nil), NewMemberLeft(0x2C, 500, 1001, "Leaver").Encode(nil, v83)(nil))
	eq("MemberExpel", NewMemberExpel(0x2F, 500, 1001, "Expelled").Encode(nil, v61)(nil), NewMemberExpel(0x2F, 500, 1001, "Expelled").Encode(nil, v83)(nil))
	eq("MemberJoined", NewMemberJoined(0x27, 500, 1001, "Joiner", 100, 50, 2, true, 1).Encode(nil, v61)(nil), NewMemberJoined(0x27, 500, 1001, "Joiner", 100, 50, 2, true, 1).Encode(nil, v83)(nil))
	eq("Invite", NewInvite(0x05, 500, "Inviter", 7, 9).Encode(nil, v61)(nil), NewInvite(0x05, 500, "Inviter", 7, 9).Encode(nil, v83)(nil))
	eq("TitleChange", NewTitleChange(0x3E, 500, titles).Encode(nil, v61)(nil), NewTitleChange(0x3E, 500, titles).Encode(nil, v83)(nil))
	eq("Disband", NewDisband(0x32, 500).Encode(nil, v61)(nil), NewDisband(0x32, 500).Encode(nil, v83)(nil))
	eq("CapacityChange", NewCapacityChange(0x48, 500, 100).Encode(nil, v61)(nil), NewCapacityChange(0x48, 500, 100).Encode(nil, v83)(nil))
	eq("MemberUpdate", NewMemberUpdate(0x3C, 500, 1001, 50, 100).Encode(nil, v61)(nil), NewMemberUpdate(0x3C, 500, 1001, 50, 100).Encode(nil, v83)(nil))
	eq("ShowTitles", NewShowTitles(0x49, 500, showEntries).Encode(nil, v61)(nil), NewShowTitles(0x49, 500, showEntries).Encode(nil, v83)(nil))
	eq("QuestWaitingNotice", NewQuestWaitingNotice(0x4C, 3, 1).Encode(nil, v61)(nil), NewQuestWaitingNotice(0x4C, 3, 1).Encode(nil, v83)(nil))
	eq("BoardAuthKeyUpdate", NewBoardAuthKeyUpdate(0x4D, "auth-key").Encode(nil, v61)(nil), NewBoardAuthKeyUpdate(0x4D, "auth-key").Encode(nil, v83)(nil))
	eq("SetSkillResponse", NewSetSkillResponse(0x4E, true, "ok").Encode(nil, v61)(nil), NewSetSkillResponse(0x4E, true, "ok").Encode(nil, v83)(nil))
	eq("Info", NewInfo(true, 500, "Guild", titles, infoMembers, 100, 5, 4, 3, 2, "notice", 1000, 0).Encode(nil, v61)(nil), NewInfo(true, 500, "Guild", titles, infoMembers, 100, 5, 4, 3, 2, "notice", 1000, 0).Encode(nil, v83)(nil))
	eq("BBSThreadList", NewBBSThreadList(0x06, &notice, threads, 0).Encode(nil, v61)(nil), NewBBSThreadList(0x06, &notice, threads, 0).Encode(nil, v83)(nil))
	eq("BBSThread", NewBBSThread(0x07, 1, 2, 123, "T", "M", 3, replies).Encode(nil, v61)(nil), NewBBSThread(0x07, 1, 2, 123, "T", "M", 3, replies).Encode(nil, v83)(nil))
}

// packet-audit:verify packet=guild/clientbound/GuildForeignNameChanged version=gms_v61 ida=0x7cc166
// packet-audit:verify packet=guild/clientbound/GuildForeignEmblemChanged version=gms_v61 ida=0x7cc1b1
func TestGuildForeignChangedV61(t *testing.T) {
	// ForeignNameChanged: WriteInt(charId) + WriteAsciiString(name).
	// charId=1001 (e9 03 00 00) + "Bob" (03 00 'B' 'o' 'b').
	// (@0x7cc166: DecodeStr(name); charId supplied by the user-pool router.)
	gotName := NewForeignNameChanged(1001, "Bob").Encode(nil, nil)(nil)
	wantName := []byte{0xE9, 0x03, 0x00, 0x00, 0x03, 0x00, 0x42, 0x6F, 0x62}
	if !bytes.Equal(gotName, wantName) {
		t.Errorf("v61 ForeignNameChanged: got % x want % x", gotName, wantName)
	}
	// ForeignEmblemChanged: WriteInt(charId)+WriteShort(bg)+WriteByte(bgColor)+WriteShort(logo)+WriteByte(logoColor).
	// charId=1001, logo=3,logoColor=2,bg=5,bgColor=4 → e9 03 00 00 | 05 00 | 04 | 03 00 | 02.
	// (@0x7cc1b1: Decode2(bg)+Decode1(bgColor)+Decode2(logo)+Decode1(logoColor).)
	gotEmblem := NewForeignEmblemChanged(1001, 3, 2, 5, 4).Encode(nil, nil)(nil)
	wantEmblem := []byte{0xE9, 0x03, 0x00, 0x00, 0x05, 0x00, 0x04, 0x03, 0x00, 0x02}
	if !bytes.Equal(gotEmblem, wantEmblem) {
		t.Errorf("v61 ForeignEmblemChanged: got % x want % x", gotEmblem, wantEmblem)
	}
}
