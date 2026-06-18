package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/clientbound/GuildBBSThread version=gms_v83 ida=0xa1233f
// packet-audit:verify packet=guild/clientbound/GuildBBSThreadList version=gms_v83 ida=0xa1233f
// packet-audit:verify packet=guild/clientbound/GuildBBSThread version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/clientbound/GuildBBSThreadList version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/clientbound/GuildBBSThread version=gms_v95 ida=0x7c6630
// packet-audit:verify packet=guild/clientbound/GuildBBSThreadList version=gms_v95 ida=0x7c46c0
// v84 BBS clientbound dispatch sub_841EC9 (Decode1-6): list sub_841F06 / view sub_84224E / notfound sub_842571.
// Read orders byte-identical to v83 (no mode drift; verified live v84 IDB @port 13337). Bodies hand-computed below.
// packet-audit:verify packet=guild/clientbound/GuildBBSThread version=gms_v84 ida=0x84224e
// packet-audit:verify packet=guild/clientbound/GuildBBSThreadList version=gms_v84 ida=0x841f06
// packet-audit:verify packet=guild/clientbound/GuildBBSThread version=jms_v185 ida=ABSENT
// packet-audit:verify packet=guild/clientbound/GuildBBSThreadList version=jms_v185 ida=ABSENT
func TestBBSThreadListEmpty(t *testing.T) {
	input := NewBBSThreadList(GuildBBSModeThreadList, nil, nil, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestBBSThreadListWithThreads(t *testing.T) {
	threads := []BBSThreadSummary{
		{Id: 1, PosterId: 100, Title: "Hello", CreatedAt: 116444736000000000, EmoticonId: 0, ReplyCount: 3},
		{Id: 2, PosterId: 200, Title: "Test", CreatedAt: 116444736100000000, EmoticonId: 1, ReplyCount: 0},
	}
	input := NewBBSThreadList(GuildBBSModeThreadList, nil, threads, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestBBSThread(t *testing.T) {
	replies := []BBSReply{
		{Id: 1, PosterId: 200, CreatedAt: 116444736000000000, Message: "Nice post!"},
	}
	input := NewBBSThread(GuildBBSModeThread, 1, 100, 116444736000000000, "Hello", "World", 0, replies)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=guild/clientbound/GuildBBSEntryNotFound version=gms_v83 ida=0x816c32
// packet-audit:verify packet=guild/clientbound/GuildBBSEntryNotFound version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/clientbound/GuildBBSEntryNotFound version=gms_v95 ida=0x7c8260
// v84 notfound arm sub_842571 via dispatch sub_841EC9 ((Decode1-6)==2, mode 8); mode-only, byte-identical to v83.
// packet-audit:verify packet=guild/clientbound/GuildBBSEntryNotFound version=gms_v84 ida=0x841ec9
// jms BBS clientbound is VERSION-ABSENT (no CUIGuildBBS symbol in the jms SCY IDB;
// no GUILD_BBS_PACKET in the jms registry; no GuildBBS writer in the jms template).
// Marked ABSENT consistently with BBSThread/BBSThreadList (the codebase's verified-absent model).
// packet-audit:verify packet=guild/clientbound/GuildBBSEntryNotFound version=jms_v185 ida=ABSENT
func TestBBSEntryNotFound(t *testing.T) {
	input := NewBBSEntryNotFound(GuildBBSModeEntryNotFound)
	got := input.Encode(nil, nil)(nil)
	if len(got) != 1 || got[0] != GuildBBSModeEntryNotFound {
		t.Fatalf("got %v, want [%d]", got, GuildBBSModeEntryNotFound)
	}
}
