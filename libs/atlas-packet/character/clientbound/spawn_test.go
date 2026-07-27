package clientbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterSpawn version=gms_v83 ida=0x972100
// packet-audit:verify packet=character/clientbound/CharacterSpawn version=gms_v87 ida=0x9f7084
// packet-audit:verify packet=character/clientbound/CharacterSpawn version=gms_v95 ida=0x94db40
// packet-audit:verify packet=character/clientbound/CharacterSpawn version=gms_v84 ida=0x9b20a0
// packet-audit:verify packet=character/clientbound/CharacterSpawn version=jms_v185 ida=0xa43ddd
func TestCharacterSpawnEncode(t *testing.T) {
	avatar := model.Avatar{}
	cts := model.NewCharacterTemporaryStat()
	guild := GuildEmblem{Name: "TestGuild"}
	input := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 100, avatar, nil, true, 100, 200, 6, 0)
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

// TestCharacterSpawnJMSGolden pins the jms_v185 wire for CharacterSpawn against
// CUserPool::OnUserEnterField @0xa43ddd → CUserRemote::Init @0xa52876. The jms
// read order (IDA-verified, jms export CUserRemote::Init calls):
//
//	level, name, guildName, guild logo (2/1/2/1), SecondaryStat::DecodeForRemote,
//	jobId, AvatarLook::Decode, driver(int)+passenger(int) [jms], choco(int),
//	itemEffect(int), chair(int), x, y, stance, foothold(short) → pet while-loop
//	(NO bShowAdminEffect byte), mount(3 ints), miniRoom/adBoard/couple/friend/
//	marriage flags, dragon-effect flag (call 46), final-effect flag (call 47).
//
// The jms client has NO admin byte after the foothold and NO trailing team byte —
// both are GMS-only. Those two bytes were the jms wire delta fixed in this commit's
// codec change; here the body is 238 bytes (was 240 with the spurious bytes).
//
// The cts base-stat blocks carry a tLastUpdated time interval, so the middle of the
// body is time-dependent; this golden pins the fully-deterministic header (through
// the SecondaryStat flag word) and the entire tail (avatar end through the corrected
// final-effect byte), which is where the wire delta lives.
func TestCharacterSpawnJMSGolden(t *testing.T) {
	v := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	guild := GuildEmblem{Name: "TestGuild", LogoBackground: 1, LogoBackgroundColor: 2, Logo: 3, LogoColor: 4}
	in := NewCharacterSpawn(12345, 50, "TestChar", guild, model.NewCharacterTemporaryStat(), 100, model.Avatar{}, nil, false, 100, 200, 3, 0)

	got := in.Encode(nil, ctx)(nil)

	if len(got) != 238 {
		t.Fatalf("jms CharacterSpawn length: got %d want 238 (admin+team bytes must be absent)", len(got))
	}
	// Header through the 16-byte SecondaryStat flag word: charId, level,
	// name("TestChar"), guildName("TestGuild"), logo (2/1/2/1), empty-cts mask
	// (bits 110-116 = 0x001FC000 in the jms two-state group).
	wantPrefix, _ := hex.DecodeString(
		"3930000032080054657374436861720900546573744775696c6401000203000400c01f00000000000000000000000000")
	if !bytes.Equal(got[:48], wantPrefix) {
		t.Errorf("jms CharacterSpawn header+mask: got %x want %x", got[:48], wantPrefix)
	}
	// Tail from the avatar-end marker (ffff) through the corrected final-effect byte:
	// driver(0)+passenger(0)+choco(0)+itemEffect(0)+chair(0)+x(100)+y(200)+stance(3)+
	// foothold(0)+pets-terminator(0)+mount(1,0,0)+5 ring flags+newyear(jms skips)+
	// berserk/dragon(0)+jms final-effect(0). NO admin byte, NO team byte.
	wantTail, _ := hex.DecodeString(
		"0000000100000000ffff0000000000000000000000000000000000000000000000000000000000000000000000006400c8000300000001000000000000000000000000000000000000")
	if !bytes.Equal(got[165:], wantTail) {
		t.Errorf("jms CharacterSpawn tail:\n got %x\nwant %x", got[165:], wantTail)
	}
}

// TestCharacterSpawnV48Golden pins the very-legacy GMS v48 SPAWN_PLAYER wire against
// CUserRemote::Init sub_6BBC17 @0x6bbc17 (GMS_v48_1_DEVM.exe, port 13337). The v48 read
// order diverges from the v79 legacy path in four IDA-verified ways:
//  1. CTS-foreign (sub_5CBA1F @0x6bbcde) is an 8-byte mask; empty CTS = 8 zero bytes,
//     no base-stat blocks.
//  2. NO Decode2(jobId) — the CTS foreign goes straight to AvatarLook::Decode @0x6bbcea.
//  3. Single-pet flag (Decode1 → sub_58C7CC @0x6bbe5e), not the 3-slot bool loop.
//  4. Six tail flags (miniroom @0x6bbed5 / adboard @0x6bc045 / couple @0x6bc174 /
//     friend @0x6bc1bf / marriage @0x6bc20a / final-effect @0x6bc25c) — NO new-year-card
//     byte, NO trailing team byte.
//
// Empty CTS + empty avatar make the whole wire deterministic (no base-stat time block).
// packet-audit:verify packet=character/clientbound/CharacterSpawn version=gms_v48 ida=0x6bbc17
func TestCharacterSpawnV48Golden(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	guild := GuildEmblem{Name: "TestGuild", LogoBackground: 1, LogoBackgroundColor: 2, Logo: 3, LogoColor: 4}
	in := NewCharacterSpawn(12345, 50, "TestChar", guild, model.NewCharacterTemporaryStat(), 100, model.Avatar{}, nil, false, 100, 200, 3, 0)
	got := in.Encode(nil, ctx)(nil)

	if len(got) != 99 {
		t.Fatalf("v48 CharacterSpawn length: got %d want 99 (no level, no jobId, 8-byte mask, single-pet flag, 6 tail flags)", len(got))
	}
	// Header through the 8-byte CTS-foreign mask: charId, name("TestChar"), guildName
	// ("TestGuild"), logo(2/1/2/1), empty 8-byte mask. No level byte (legacy), and the
	// mask is immediately followed by the avatar (no jobId).
	wantHeader, _ := hex.DecodeString("393000000800546573744368617209005465737447756" +
		"9" + "6c64010002030004" + "0000000000000000")
	if !bytes.Equal(got[:39], wantHeader) {
		t.Errorf("v48 CharacterSpawn header+mask: got %x want %x", got[:39], wantHeader)
	}
	// Bytes 39..60 are the empty avatar (proves avatar directly follows the mask — no
	// jobId short was inserted). Compare against the standalone avatar encoding.
	avatarBytes := model.Avatar{}.Encode(nil, ctx)(nil)
	if !bytes.Equal(got[39:39+len(avatarBytes)], avatarBytes) {
		t.Errorf("v48 CharacterSpawn avatar: got %x want %x", got[39:39+len(avatarBytes)], avatarBytes)
	}
	// Tail: choco+itemEffect+chair (3 ints) + x(100)+y(200)+stance(3) + fh(0) + admin(0)
	// + pet-flag(0) + mount(1,0,0) + 6 ring/effect flags. No new-year-card, no team.
	wantTail, _ := hex.DecodeString("000000000000000000000000" + "6400c80003" + "0000" +
		"00" + "00" + "010000000000000000000000" + "000000000000")
	if !bytes.Equal(got[60:], wantTail) {
		t.Errorf("v48 CharacterSpawn tail:\n got %x\nwant %x", got[60:], wantTail)
	}
}

func testSpawnAvatar() model.Avatar {
	equip := map[slot.Position]uint32{5: 1040002, 6: 1060002, 7: 1072001}
	masked := map[slot.Position]uint32{}
	pets := map[int8]uint32{}
	return model.NewAvatar(0, 1, 20000, false, 30000, equip, masked, pets)
}

func TestCharacterSpawnRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			avatar := testSpawnAvatar()
			cts := model.NewCharacterTemporaryStat()
			guild := GuildEmblem{Name: "TestGuild", LogoBackground: 1, LogoBackgroundColor: 2, Logo: 3, LogoColor: 4}
			// enteringField=false for exact round-trip
			input := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 312, avatar, nil, false, 100, 200, 3, 37)
			output := CharacterSpawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			// Legacy GMS (< v83) SPAWN_PLAYER carries no level byte on the wire
			// (v79 CUserRemote::Init @0x8d589e reads name first), so level is not
			// round-trippable for those variants. v83+ and JMS transmit it.
			legacy := v.Region == "GMS" && v.MajorVersion < 83
			if !legacy && output.Level() != input.Level() {
				t.Errorf("level: got %v, want %v", output.Level(), input.Level())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Guild().Name != input.Guild().Name {
				t.Errorf("guildName: got %v, want %v", output.Guild().Name, input.Guild().Name)
			}
			// Pre-v61 GMS (v48) SPAWN_PLAYER carries no jobId short on the wire
			// (CUserRemote::Init sub_6BBC17 reads CTS-foreign then AvatarLook with
			// no Decode2 between), so jobId is not round-trippable for those variants.
			legacyV48 := v.Region == "GMS" && v.MajorVersion < 61
			if !legacyV48 && output.JobId() != input.JobId() {
				t.Errorf("jobId: got %v, want %v", output.JobId(), input.JobId())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
			if output.Stance() != input.Stance() {
				t.Errorf("stance: got %v, want %v", output.Stance(), input.Stance())
			}
			if output.Fh() != 37 {
				t.Errorf("fh: got %v, want %v", output.Fh(), 37)
			}
		})
	}
}

func TestCharacterSpawnEnteringFieldEncodesFhZero(t *testing.T) {
	// entering-field spawns are intentionally airborne (y-42, stance 6):
	// the wire fh must stay 0 even when the model carries a real foothold.
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			avatar := testSpawnAvatar()
			cts := model.NewCharacterTemporaryStat()
			guild := GuildEmblem{Name: "TestGuild"}
			input := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 312, avatar, nil, true, 100, 200, 6, 37)
			output := CharacterSpawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Fh() != 0 {
				t.Errorf("entering-field fh on the wire: got %v, want 0", output.Fh())
			}
		})
	}
}

func TestCharacterSpawnWithPetsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			avatar := testSpawnAvatar()
			cts := model.NewCharacterTemporaryStat()
			guild := GuildEmblem{Name: "Guild"}
			pets := []SpawnPet{
				{Slot: 0, Pet: model.Pet{TemplateId: 5000001, Name: "Dog", Id: 100, X: 10, Y: 20, Stance: 1, Foothold: 5}},
				{Slot: 1, Pet: model.Pet{TemplateId: 5000002, Name: "Cat", Id: 200, X: 30, Y: 40, Stance: 2, Foothold: 6}},
			}
			input := NewCharacterSpawn(999, 80, "PetOwner", guild, cts, 100, avatar, pets, false, 50, 60, 4, 0)
			output := CharacterSpawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			// Pre-v61 GMS (v48) SPAWN_PLAYER carries a single-pet flag (sub_58C7CC),
			// not the 3-slot bool loop — only the first pet survives the round-trip.
			legacyV48 := v.Region == "GMS" && v.MajorVersion < 61
			wantCount := len(input.Pets())
			if legacyV48 && wantCount > 1 {
				wantCount = 1
			}
			if len(output.Pets()) != wantCount {
				t.Errorf("pets count: got %v, want %v", len(output.Pets()), wantCount)
			} else {
				for i, p := range output.Pets() {
					if p.Pet.TemplateId != pets[i].Pet.TemplateId {
						t.Errorf("pet[%d] templateId: got %v, want %v", i, p.Pet.TemplateId, pets[i].Pet.TemplateId)
					}
				}
			}
		})
	}
}
