package clientbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
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
			MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42}, false)
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
	input := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{50200004}, 1142007, MonsterBookInfo{}, MountInfo{}, false)
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
			input := NewCharacterInfo(100, 70, 312, 50, "TestGuild", pets, wishList, 1142000, MonsterBookInfo{}, MountInfo{}, false)
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
			// Legacy GMS v29..v60 (v48) writes a SINGLE flag-gated pet (codec:
			// character/clientbound/info.go:88-108, IDA @0x71cb6f/@0x71cbe1); only
			// pets[0] survives the round-trip there. v61+/JMS/v83+ write the
			// bool-terminated multi-pet loop (@0x8455ed) and round-trip the full list.
			wantPets := pets
			if v.Region == "GMS" && v.MajorVersion > 28 && v.MajorVersion < 61 {
				wantPets = pets[:1]
			}
			if len(output.Pets()) != len(wantPets) {
				t.Errorf("pets count: got %v, want %v", len(output.Pets()), len(wantPets))
			} else {
				for i, p := range output.Pets() {
					if p.TemplateId != wantPets[i].TemplateId {
						t.Errorf("pet[%d] templateId: got %v, want %v", i, p.TemplateId, wantPets[i].TemplateId)
					}
					if p.Name != wantPets[i].Name {
						t.Errorf("pet[%d] name: got %v, want %v", i, p.Name, wantPets[i].Name)
					}
				}
			}
			if len(output.WishList()) != len(input.WishList()) {
				t.Errorf("wishList count: got %v, want %v", len(output.WishList()), len(input.WishList()))
			}
			// The medal block only rides the wire for GMS v72+ and JMS; the legacy
			// GMS <=61 clients (verified v61 @0x8455ed) omit it, so medalId is not
			// round-tripped there.
			if (v.Region == "GMS" && v.MajorVersion > 61) || v.Region == "JMS" {
				if output.MedalId() != input.MedalId() {
					t.Errorf("medalId: got %v, want %v", output.MedalId(), input.MedalId())
				}
			}
		})
	}
}

func TestCharacterInfo_MonsterBookCover(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	want := MonsterBookInfo{Level: 5, NormalCards: 10, SpecialCards: 3, TotalCards: 13, Cover: 2380001}
	in := NewCharacterInfo(1, 10, 100, 0, "", nil, nil, 0, want, MountInfo{}, false)
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
	in := NewCharacterInfo(1, 10, 100, 0, "", nil, nil, 0, want, MountInfo{}, false)
	out := CharacterInfo{}
	pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	if out.MonsterBookCover() != 100100 {
		t.Errorf("cover = %d, want 100100", out.MonsterBookCover())
	}
}

// TestCharacterInfoJMSGolden pins the full jms_v185 wire for a CharacterInfo with
// a pet, an active mount, a wishlist, and a monster-book block. jms read order is
// CWvsContext::OnCharacterInfo @0xb0aa6e:
//
//	Decode4(charId), Decode1(level), Decode2(job), Decode2(fame), Decode1(married),
//	DecodeStr(guild), DecodeStr(alliance), Decode4(v32)+Decode4(p) consumed by
//	SetUserInfo, Decode1(medalInfo byte), Decode1(pet flag)→SetMultiPetInfo (per-pet
//	Decode4/Str/1/2/1/2/4, bool-terminated @0x9bb959), Decode1(mount flag)+3×Decode4,
//	Decode1(wish count)+count×int, SomethingMonsterBook @0x70522a (5×Decode4),
//	MedalAchievementInfo::Decode @0x9bcacf (Decode4 medalId + Decode2 quest count),
//	then a trailing Decode4 count (jms-only; codec emits 0). The trailing int is the
//	4-byte jms delta over v83 (99 vs 95 bytes).
func TestCharacterInfoJMSGolden(t *testing.T) {
	v := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	pets := []InfoPet{{Slot: 0, TemplateId: 5000000, Name: "Kitty", Level: 15, Closeness: 200, Fullness: 80}}
	mb := MonsterBookInfo{Level: 5, NormalCards: 10, SpecialCards: 3, TotalCards: 13, Cover: 2380001}
	mount := MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42}
	in := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{1002000, 1002001}, 1142007, mb, mount, false)

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
	in := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{1002000, 1002001}, 1142007, mb, mount, false)

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

// TestCharacterInfoV92Golden pins the full gms_v92 CharacterInfo wire against
// CWvsContext::OnCharacterInfo @0x9daa40. v92's gates (MajorVersion<=87 for
// monster-book, MajorVersion>61 for medal, MajorAtLeast(87) for chair) all
// resolve identically to v95 (info.go:178,190,194), so the wire is
// byte-identical to v95 -- cross-checked below against a live v95 encode of
// the same fixture, not merely asserted.
// packet-audit:verify packet=character/clientbound/CharacterInfo version=gms_v92 ida=0x9daa40
func TestCharacterInfoV92Golden(t *testing.T) {
	ctx := pt.CreateContext("GMS", 92, 1)
	pets := []InfoPet{{Slot: 0, TemplateId: 5000000, Name: "Kitty", Level: 15, Closeness: 200, Fullness: 80}}
	mount := MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42}
	in := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{1002000, 1002001}, 1142007, MonsterBookInfo{}, mount, false)

	got := in.Encode(nil, ctx)(nil)
	want, _ := hex.DecodeString("393000003264000a00000900546573744775696c6400000001404b4c0005004b697474790fc80050000000000000000107000000d20400002a00000002104a0f00114a0f00f76c1100000000000000")
	if !bytes.Equal(got, want) {
		t.Errorf("v92 CharacterInfo wire (len got=%d want=%d):\n got %x\nwant %x", len(got), len(want), got, want)
	}
	v95 := in.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	if !bytes.Equal(got, v95) {
		t.Errorf("v92 CharacterInfo must equal v95:\n v92 %x\n v95 %x", got, v95)
	}

	// Marriage-ring bool (site D) round-trips on the v92 arm too: flipping
	// the flag changes exactly the one marriage-flag byte at offset 9.
	inFalse := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", nil, nil, 0, MonsterBookInfo{}, MountInfo{}, false)
	inTrue := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", nil, nil, 0, MonsterBookInfo{}, MountInfo{}, true)
	gotFalse := inFalse.Encode(nil, ctx)(nil)
	gotTrue := inTrue.Encode(nil, ctx)(nil)
	const marriageOffset = 9
	if gotFalse[marriageOffset] != 0x00 || gotTrue[marriageOffset] != 0x01 {
		t.Fatalf("v92 marriage-flag bytes: false=%#x true=%#x", gotFalse[marriageOffset], gotTrue[marriageOffset])
	}
	if len(gotFalse) != len(gotTrue) {
		t.Fatalf("v92 marriage flag must not change length: false=%d true=%d", len(gotFalse), len(gotTrue))
	}
}

// TestCharacterInfoV48Golden pins the full gms_v48 CharacterInfo wire.
//
// Client read order — CWvsContext::OnCharacterInfo (GMS_v48_1_DEVM.exe, port 13337
// @0x71caed):
//
//	Decode4(charId) /*0x71cb22*/, Decode1(level) /*0x71cb49*/, Decode2(job) /*0x71cb53*/,
//	Decode2(fame) /*0x71cb5d*/, DecodeStr(guild) /*0x71cb64*/, Decode1(pet flag) /*0x71cb6f*/;
//	if set a SINGLE pet: Decode4(templateId) /*0x71cbe1*/, DecodeStr(name) /*0x71cbe9*/,
//	Decode1(level) /*0x71cbfb*/, Decode2(closeness) /*0x71cc05*/, Decode1(fullness) /*0x71cc0e*/,
//	Decode2(skill) /*0x71cc18*/, Decode4(itemId) /*0x71cc20*/ (NO "more pets" terminator);
//	Decode1(mount flag) /*0x71cc62*/ + 3×Decode4 /*0x71cc77*/ → SetTamingMobInfo;
//	Decode1(wish count) /*0x71cc97*/ + count×Decode4 (DecodeBuffer 4*n) /*0x71ccc3*/. Returns.
//
// v48 is MUCH shorter than v61+ (@0x8455ed): it OMITS the marriage-ring bool, the
// alliance string, the medalInfo byte, and the monster-book block, and single-pets the
// pet section (v61+ is a bool-terminated multi-pet loop). Same fixture input as the v79
// golden; the v48 wire is the v79 wire with those four sections removed.
//
// packet-audit:verify packet=character/clientbound/CharacterInfo version=gms_v48 ida=0x71caed
func TestCharacterInfoV48Golden(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	pets := []InfoPet{{Slot: 0, TemplateId: 5000000, Name: "Kitty", Level: 15, Closeness: 200, Fullness: 80}}
	mount := MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42}
	in := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{1002000, 1002001}, 1142007, MonsterBookInfo{}, mount, false)

	got := in.Encode(nil, ctx)(nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // charId 12345          @0x71cb22
		0x32,       // level 50               @0x71cb49
		0x64, 0x00, // job 100                @0x71cb53
		0x0a, 0x00, // fame 10                @0x71cb5d
		0x09, 0x00, 'T', 'e', 's', 't', 'G', 'u', 'i', 'l', 'd', // guild "TestGuild" @0x71cb64
		0x01,                   // pet flag = 1           @0x71cb6f
		0x40, 0x4b, 0x4c, 0x00, // pet templateId 5000000 @0x71cbe1
		0x05, 0x00, 'K', 'i', 't', 't', 'y', // pet name "Kitty" @0x71cbe9
		0x0f,       // pet level 15           @0x71cbfb
		0xc8, 0x00, // pet closeness 200      @0x71cc05
		0x50,       // pet fullness 80        @0x71cc0e
		0x00, 0x00, // pet skill 0            @0x71cc18
		0x00, 0x00, 0x00, 0x00, // pet itemId 0           @0x71cc20
		0x01,                   // mount flag = 1         @0x71cc62
		0x07, 0x00, 0x00, 0x00, // mount level 7          @0x71cc77
		0xd2, 0x04, 0x00, 0x00, // mount exp 1234
		0x2a, 0x00, 0x00, 0x00, // mount tiredness 42
		0x02,                   // wish count 2           @0x71cc97
		0x10, 0x4a, 0x0f, 0x00, // wish 1002000           @0x71ccc3
		0x11, 0x4a, 0x0f, 0x00, // wish 1002001
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 CharacterInfo wire (len got=%d want=%d):\n got %x\nwant %x", len(got), len(want), got, want)
	}
}

// TestCharacterInfoMarriageFlag drives the marriage-ring bool (site D) through
// NewCharacterInfo's hasMarriageRing parameter. FR-8 guard: the legacy GMS v28
// arm (no marriage bool on the wire at all) must stay byte-identical regardless
// of the flag's value. OQ-3 guard: on the modern arm, flipping the flag must
// change exactly the one marriage-flag byte — v83 @0xa2370b, v87 @0xabb181, v95
// @0xa05750, jms @0xb0aa6e all read sCommunity (guildName) unconditionally next
// with no partner block, so a populated marriage arm here would desynchronise
// the stream.
func TestCharacterInfoMarriageFlag(t *testing.T) {
	t.Run("false, modern", func(t *testing.T) {
		ctx := pt.CreateContext("GMS", 83, 1)
		in := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", nil, nil, 0, MonsterBookInfo{}, MountInfo{}, false)
		got := in.Encode(nil, ctx)(nil)
		want, _ := hex.DecodeString("393000003264000a00000900546573744775696c640000000000000000000000000000000000000000000000000000000000000000")
		if !bytes.Equal(got, want) {
			t.Errorf("false marriage flag wire:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("true, modern", func(t *testing.T) {
		ctx := pt.CreateContext("GMS", 83, 1)
		inFalse := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", nil, nil, 0, MonsterBookInfo{}, MountInfo{}, false)
		inTrue := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", nil, nil, 0, MonsterBookInfo{}, MountInfo{}, true)
		gotFalse := inFalse.Encode(nil, ctx)(nil)
		gotTrue := inTrue.Encode(nil, ctx)(nil)

		if len(gotTrue) != len(gotFalse) {
			t.Fatalf("true/false length mismatch: got %d want %d", len(gotTrue), len(gotFalse))
		}
		// marriage-flag offset: charId(4) + level(1) + jobId(2) + fame(2) = 9.
		const marriageOffset = 9
		if gotFalse[marriageOffset] != 0x00 {
			t.Fatalf("false marriage-flag byte: got %#x want 0x00", gotFalse[marriageOffset])
		}
		if gotTrue[marriageOffset] != 0x01 {
			t.Errorf("true marriage-flag byte: got %#x want 0x01", gotTrue[marriageOffset])
		}
		diffs := 0
		for i := range gotTrue {
			if gotTrue[i] != gotFalse[i] {
				diffs++
			}
		}
		if diffs != 1 {
			t.Errorf("true vs false must differ in exactly one byte, got %d differing bytes", diffs)
		}
		// The guild-name length-prefixed string follows immediately after the
		// marriage byte in both cases — no partner block was inserted.
		wantGuildPrefix, _ := hex.DecodeString("0900546573744775696c64") // len(9) + "TestGuild"
		got := gotTrue[marriageOffset+1 : marriageOffset+1+len(wantGuildPrefix)]
		if !bytes.Equal(got, wantGuildPrefix) {
			t.Errorf("guildName must follow the marriage byte directly:\n got %x\nwant %x", got, wantGuildPrefix)
		}
	})

	t.Run("legacy arm untouched", func(t *testing.T) {
		// The legacy arm (GMS v29..v60, gate `MajorVersion() > 28 && < 61`,
		// info.go:90) has no marriage bool at all — v61 is the first client
		// to add it (info.go comment, IDA @0x8455ed). GMS v28 itself is
		// EXCLUDED from this gate (28 is not > 28) and falls through to the
		// modern arm, so it is not a legacy-arm example; GMS v48 (already
		// pinned byte-for-byte by TestCharacterInfoV48Golden) is the correct
		// FR-8 fixture — same input, hasMarriageRing=true this time, must
		// produce the byte-identical legacy wire.
		ctx := pt.CreateContext("GMS", 48, 1)
		pets := []InfoPet{{Slot: 0, TemplateId: 5000000, Name: "Kitty", Level: 15, Closeness: 200, Fullness: 80}}
		mount := MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42}
		in := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{1002000, 1002001}, 1142007, MonsterBookInfo{}, mount, true)
		got := in.Encode(nil, ctx)(nil)
		wantV48 := []byte{
			0x39, 0x30, 0x00, 0x00,
			0x32,
			0x64, 0x00,
			0x0a, 0x00,
			0x09, 0x00, 'T', 'e', 's', 't', 'G', 'u', 'i', 'l', 'd',
			0x01,
			0x40, 0x4b, 0x4c, 0x00,
			0x05, 0x00, 'K', 'i', 't', 't', 'y',
			0x0f,
			0xc8, 0x00,
			0x50,
			0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			0x01,
			0x07, 0x00, 0x00, 0x00,
			0xd2, 0x04, 0x00, 0x00,
			0x2a, 0x00, 0x00, 0x00,
			0x02,
			0x10, 0x4a, 0x0f, 0x00,
			0x11, 0x4a, 0x0f, 0x00,
		}
		if !bytes.Equal(got, wantV48) {
			t.Errorf("gms v48 (legacy arm) wire must be untouched by hasMarriageRing:\n got %x\nwant %x", got, wantV48)
		}
	})
}

func TestCharacterInfoEmptyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterInfo(200, 30, 100, 0, "", nil, nil, 0, MonsterBookInfo{}, MountInfo{}, false)
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
