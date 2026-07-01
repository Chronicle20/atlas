package clientbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=character/clientbound/CharacterInfo version=gms_v83 ida=0xa2370b
// packet-audit:verify packet=character/clientbound/CharacterInfo version=gms_v87 ida=0xabb181
// packet-audit:verify packet=character/clientbound/CharacterInfo version=gms_v95 ida=0xa05750
// packet-audit:verify packet=character/clientbound/CharacterInfo version=gms_v84 ida=0xa6eda8
// packet-audit:verify packet=character/clientbound/CharacterInfo version=jms_v185 ida=0xb0aa6e
// packet-audit:verify packet=character/clientbound/CharacterInfo version=gms_v79 ida=0x96d8d5

// TestCharacterInfo_MountRoundTrip locks the tamed-mob block: when a mount is
// active the writer emits flag=1 + level/exp/tiredness (3×int32), and the decoder
// reads them back. Layout is version-uniform (v83/v87/v95/JMS).
func TestCharacterInfo_MountRoundTrip(t *testing.T) {
	for _, v := range []struct {
		region   string
		maj, min uint16
	}{{"GMS", 83, 1}, {"GMS", 87, 1}, {"GMS", 95, 1}, {"JMS", 185, 1}} {
		ctx := pt.CreateContext(v.region, v.maj, v.min)
		in := NewCharacterInfo(1, 10, 100, 0, "", nil, nil, 0, MonsterBookInfo{},
			MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42})
		out := CharacterInfo{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
		if got := out.Mount(); got != (MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42}) {
			t.Errorf("%s v%d mount round-trip: got %+v", v.region, v.maj, got)
		}
	}
}

func TestCharacterInfoEncode(t *testing.T) {
	pets := []InfoPet{
		{Slot: 0, TemplateId: 5000001, Name: "Kitty", Level: 10, Closeness: 100, Fullness: 50},
	}
	input := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{50200004}, 1142007, MonsterBookInfo{}, MountInfo{})
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			encoded := input.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Error("expected non-empty encoded bytes")
			}
		})
	}
}

func TestCharacterInfoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pets := []InfoPet{
				{Slot: 0, TemplateId: 5000000, Name: "MiniDog", Level: 15, Closeness: 200, Fullness: 80},
				{Slot: 1, TemplateId: 5000001, Name: "MiniCat", Level: 10, Closeness: 100, Fullness: 50},
			}
			wishList := []uint32{1002000, 1002001, 1002002}
			input := NewCharacterInfo(100, 70, 312, 50, "TestGuild", pets, wishList, 1142000, MonsterBookInfo{}, MountInfo{})
			output := CharacterInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Level() != input.Level() {
				t.Errorf("level: got %v, want %v", output.Level(), input.Level())
			}
			if output.JobId() != input.JobId() {
				t.Errorf("jobId: got %v, want %v", output.JobId(), input.JobId())
			}
			if output.Fame() != input.Fame() {
				t.Errorf("fame: got %v, want %v", output.Fame(), input.Fame())
			}
			if output.GuildName() != input.GuildName() {
				t.Errorf("guildName: got %v, want %v", output.GuildName(), input.GuildName())
			}
			if len(output.Pets()) != len(input.Pets()) {
				t.Errorf("pets count: got %v, want %v", len(output.Pets()), len(input.Pets()))
			} else {
				for i, p := range output.Pets() {
					if p.TemplateId != pets[i].TemplateId {
						t.Errorf("pet[%d] templateId: got %v, want %v", i, p.TemplateId, pets[i].TemplateId)
					}
					if p.Name != pets[i].Name {
						t.Errorf("pet[%d] name: got %v, want %v", i, p.Name, pets[i].Name)
					}
				}
			}
			if len(output.WishList()) != len(input.WishList()) {
				t.Errorf("wishList count: got %v, want %v", len(output.WishList()), len(input.WishList()))
			}
			if output.MedalId() != input.MedalId() {
				t.Errorf("medalId: got %v, want %v", output.MedalId(), input.MedalId())
			}
		})
	}
}

func TestCharacterInfo_MonsterBookCover(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	want := MonsterBookInfo{Level: 5, NormalCards: 10, SpecialCards: 3, TotalCards: 13, Cover: 2380001}
	in := NewCharacterInfo(1, 10, 100, 0, "", nil, nil, 0, want, MountInfo{})
	out := CharacterInfo{}
	pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	if out.MonsterBook() != want {
		t.Errorf("monster book = %+v, want %+v", out.MonsterBook(), want)
	}
}

// TestCharacterInfo_CoverCarriesArbitraryValue locks the contract the channel
// writer depends on (task-082): the cover field carries whatever uint32 the
// writer supplies — now a mob id, e.g. 100100 — not a card-id-specific value.
func TestCharacterInfo_CoverCarriesArbitraryValue(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	want := MonsterBookInfo{Level: 1, NormalCards: 0, SpecialCards: 0, TotalCards: 0, Cover: 100100}
	in := NewCharacterInfo(1, 10, 100, 0, "", nil, nil, 0, want, MountInfo{})
	out := CharacterInfo{}
	pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	if out.MonsterBookCover() != 100100 {
		t.Errorf("cover = %d, want 100100", out.MonsterBookCover())
	}
}

// TestCharacterInfoJMSGolden pins the full jms_v185 wire for a CharacterInfo with
// a pet, an active mount, a wishlist, and a monster-book block. jms read order is
// CWvsContext::OnCharacterInfo @0xb0aa6e:
//   Decode4(charId), Decode1(level), Decode2(job), Decode2(fame), Decode1(married),
//   DecodeStr(guild), DecodeStr(alliance), Decode4(v32)+Decode4(p) consumed by
//   SetUserInfo, Decode1(medalInfo byte), Decode1(pet flag)→SetMultiPetInfo (per-pet
//   Decode4/Str/1/2/1/2/4, bool-terminated @0x9bb959), Decode1(mount flag)+3×Decode4,
//   Decode1(wish count)+count×int, SomethingMonsterBook @0x70522a (5×Decode4),
//   MedalAchievementInfo::Decode @0x9bcacf (Decode4 medalId + Decode2 quest count),
//   then a trailing Decode4 count (jms-only; codec emits 0). The trailing int is the
//   4-byte jms delta over v83 (99 vs 95 bytes).
func TestCharacterInfoJMSGolden(t *testing.T) {
	v := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	pets := []InfoPet{{Slot: 0, TemplateId: 5000000, Name: "Kitty", Level: 15, Closeness: 200, Fullness: 80}}
	mb := MonsterBookInfo{Level: 5, NormalCards: 10, SpecialCards: 3, TotalCards: 13, Cover: 2380001}
	mount := MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42}
	in := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{1002000, 1002001}, 1142007, mb, mount)

	got := in.Encode(nil, ctx)(nil)
	want, _ := hex.DecodeString(
		"393000003264000a00000900546573744775696c6400000001404b4c0005004b697474790fc80050000000000000000107000000d20400002a00000002104a0f00114a0f00050000000a000000030000000d000000e1502400f76c1100000000000000")
	if !bytes.Equal(got, want) {
		t.Errorf("jms CharacterInfo wire (len got=%d want=%d):\n got %x\nwant %x", len(got), len(want), got, want)
	}
}

// TestCharacterInfoV79Golden pins the full gms_v79 CharacterInfo wire.
//
// Client read order — CWvsContext::OnCharacterInfo (GMS_v79_1_DEVM.exe @0x96d8d5):
//
//	Decode4(charId) /*0x96d90a*/, Decode1(level) /*0x96d931*/, Decode2(job) /*0x96d934*/,
//	Decode2(fame) /*0x96d93e*/, Decode1(married) /*0x96d955*/, DecodeStr(guild) /*0x96d95c*/,
//	DecodeStr(alliance) /*0x96d96b*/, Decode1(medalInfo byte) /*0x96d980*/,
//	Decode1(first pet flag) /*0x96d983*/ → sub_86040E pet loop @0x86040e (per pet:
//	  Decode4(templateId), DecodeStr(name), Decode1(level), Decode2(closeness),
//	  Decode1(fullness), Decode2(skill), Decode4(itemId), Decode1(next flag) — bool-term),
//	Decode1(mount flag)+3×Decode4 /*0x96da02..0x96da26*/ → SetTamingMobInfo,
//	Decode1(wish count)+count×Decode4 (DecodeBuffer 4*n) /*0x96da4d..*/,
//	sub_651B3B monster-book @0x651b3b: 5×Decode4 (level,normal,special,total,cover-mobid),
//	sub_8613D0 medal @0x8613d0: Decode4(medalId) + Decode2(quest count) + count×Decode2.
//	NO trailing chair int (the >=87 branch is absent in v79; sub_8613D0 is the last read).
//
// v79 gates == v83: monster-book present (GMS<=87), chair absent (GMS<87). The wire is
// therefore byte-identical to v83 and equals the jms golden MINUS the jms-only trailing
// int (dword §3.1). Cross-checked against a v83-context encode of the same fixture.
func TestCharacterInfoV79Golden(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	pets := []InfoPet{{Slot: 0, TemplateId: 5000000, Name: "Kitty", Level: 15, Closeness: 200, Fullness: 80}}
	mb := MonsterBookInfo{Level: 5, NormalCards: 10, SpecialCards: 3, TotalCards: 13, Cover: 2380001}
	mount := MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42}
	in := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{1002000, 1002001}, 1142007, mb, mount)

	got := in.Encode(nil, ctx)(nil)
	// == jms golden without the jms-only trailing 4-byte int.
	want, _ := hex.DecodeString(
		"393000003264000a00000900546573744775696c6400000001404b4c0005004b697474790fc80050000000000000000107000000d20400002a00000002104a0f00114a0f00050000000a000000030000000d000000e1502400f76c11000000")
	if !bytes.Equal(got, want) {
		t.Errorf("v79 CharacterInfo wire (len got=%d want=%d):\n got %x\nwant %x", len(got), len(want), got, want)
	}
	// Cross-version equality: v79 shape is byte-identical to v83.
	v83 := in.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	if !bytes.Equal(got, v83) {
		t.Errorf("v79 CharacterInfo must equal v83:\n v79 %x\n v83 %x", got, v83)
	}
}

func TestCharacterInfoEmptyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterInfo(200, 30, 100, 0, "", nil, nil, 0, MonsterBookInfo{}, MountInfo{})
			output := CharacterInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Pets()) != 0 {
				t.Errorf("pets count: got %v, want 0", len(output.Pets()))
			}
			if len(output.WishList()) != 0 {
				t.Errorf("wishList count: got %v, want 0", len(output.WishList()))
			}
		})
	}
}
