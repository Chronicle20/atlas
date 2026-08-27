package clientbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	input := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 100, avatar, nil, true, 100, 200, 6, 0, model.RingSet{})
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
// codec change; here the body is 128 bytes: 238 minus the ~110 bytes of placeholder
// two-state base-stat blocks an empty CTS no longer emits (task-190).
//
// The cts base-stat blocks carry a tLastUpdated time interval, so the middle of the
// body is time-dependent; this golden pins the fully-deterministic header (through
// the SecondaryStat flag word) and the entire tail (avatar end through the corrected
// final-effect byte), which is where the wire delta lives.
func TestCharacterSpawnJMSGolden(t *testing.T) {
	v := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	guild := GuildEmblem{Name: "TestGuild", LogoBackground: 1, LogoBackgroundColor: 2, Logo: 3, LogoColor: 4}
	in := NewCharacterSpawn(12345, 50, "TestChar", guild, model.NewCharacterTemporaryStat(), 100, model.Avatar{}, nil, false, 100, 200, 3, 0, model.RingSet{})

	got := in.Encode(nil, ctx)(nil)

	if len(got) != 128 {
		t.Fatalf("jms CharacterSpawn length: got %d want 128 (admin+team bytes absent; no placeholder base-stat blocks)", len(got))
	}
	// Header through the 16-byte SecondaryStat flag word: charId, level,
	// name("TestChar"), guildName("TestGuild"), logo (2/1/2/1), empty-cts mask
	// (bits 110-116 = 0x001FC000 in the jms two-state group).
	wantPrefix, _ := hex.DecodeString(
		"3930000032080054657374436861720900546573744775696c6401000203000400000000000000000000000000000000")
	if !bytes.Equal(got[:48], wantPrefix) {
		t.Errorf("jms CharacterSpawn header+mask: got %x want %x", got[:48], wantPrefix)
	}
	// Tail from the avatar-end marker (ffff) through the corrected final-effect byte:
	// driver(0)+passenger(0)+choco(0)+itemEffect(0)+chair(0)+x(100)+y(200)+stance(3)+
	// foothold(0)+pets-terminator(0)+mount(1,0,0)+5 ring flags+newyear(jms skips)+
	// berserk/dragon(0)+jms final-effect(0). NO admin byte, NO team byte.
	wantTail, _ := hex.DecodeString(
		"0000000100000000ffff0000000000000000000000000000000000000000000000000000000000000000000000006400c8000300000001000000000000000000000000000000000000")
	if !bytes.Equal(got[55:], wantTail) {
		t.Errorf("jms CharacterSpawn tail:\n got %x\nwant %x", got[55:], wantTail)
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
	in := NewCharacterSpawn(12345, 50, "TestChar", guild, model.NewCharacterTemporaryStat(), 100, model.Avatar{}, nil, false, 100, 200, 3, 0, model.RingSet{})
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

// TestCharacterSpawnRingBlocks is the FR-9 guard for site B (CharacterSpawn):
// an empty model.RingSet must produce byte-identical output to the pre-task
// three-WriteByte(0) encoder, and a populated couple ring must replace the
// 3-byte ring span with the 21-byte populated couple block (flag + OwnSN +
// PartnerSN + ItemId), growing the total length by exactly 18.
func TestCharacterSpawnRingBlocks(t *testing.T) {
	guild := GuildEmblem{Name: "TestGuild", LogoBackground: 1, LogoBackgroundColor: 2, Logo: 3, LogoColor: 4}
	newInput := func(v pt.TenantVariant, rings model.RingSet) []byte {
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		in := NewCharacterSpawn(12345, 50, "TestChar", guild, model.NewCharacterTemporaryStat(), 100, model.Avatar{}, nil, false, 100, 200, 3, 0, rings)
		return in.Encode(nil, ctx)(nil)
	}

	t.Run("empty is unchanged", func(t *testing.T) {
		cases := []struct {
			variant pt.TenantVariant
			want    string
		}{
			{pt.Variants[1], "3930000032080054657374436861720900546573744775696c6401000203000400000000000000000000000000000000000064000000000000000100000000ffff000000000000000000000000000000000000000000000000000000006400c8000300000000010000000000000000000000000000000000000000"},
			{pt.Variants[3], "3930000032080054657374436861720900546573744775696c6401000203000400000000000000000000000000000000000064000000000000000100000000ffff000000000000000000000000000000000000000000000000000000000000000000000000000000006400c8000300000000010000000000000000000000000000000000000000000000"},
			{pt.Variants[4], "3930000032080054657374436861720900546573744775696c6401000203000400000000000000000000000000000000000064000000000000000100000000ffff0000000000000000000000000000000000000000000000000000000000000000000000006400c8000300000001000000000000000000000000000000000000"},
		}
		for _, c := range cases {
			t.Run(c.variant.Name, func(t *testing.T) {
				want, _ := hex.DecodeString(c.want)
				got := newInput(c.variant, model.RingSet{})
				if !bytes.Equal(got, want) {
					t.Errorf("%s empty RingSet output:\n got %x\nwant %x", c.variant.Name, got, want)
				}
			})
		}
	})

	t.Run("couple populated", func(t *testing.T) {
		v := pt.Variants[1] // GMS v83
		emptyHex := "3930000032080054657374436861720900546573744775696c6401000203000400000000000000000000000000000000000064000000000000000100000000ffff000000000000000000000000000000000000000000000000000000006400c8000300000000010000000000000000000000000000000000000000"
		empty, _ := hex.DecodeString(emptyHex)
		// The 3-byte ring span (couple + friendship + marriage flags) sits
		// after mount(12 bytes)+miniroom(1)+adboard(1) and before the v83
		// newyear/berserk/dragon/team tail (4 bytes): offset = len-4-3.
		offset := len(empty) - 4 - 3
		// fixture values shared with Task 2 (libs/atlas-packet/model/ring_test.go).
		// PartnerSN goes through a variable (not a bare constant conversion) so
		// the two's-complement reinterpretation is a runtime, not a constant,
		// conversion — the literal exceeds int64's range as an untyped constant.
		fixturePartnerSNU := uint64(0x99AABBCCDDEEFF00)
		fixturePartnerSN := int64(fixturePartnerSNU)
		couple := &model.PairRing{OwnSN: 0x1122334455667788, PartnerSN: fixturePartnerSN, ItemId: 0x00001234}
		got := newInput(v, model.RingSet{Couple: couple})

		// Only the couple arm's 1-byte flag grows to a 21-byte block (flag +
		// OwnSN + PartnerSN + ItemId); the friendship and marriage flags stay
		// at 1 byte each, unmoved. Growth is therefore 21-1 = 20 bytes, not
		// the whole 3-byte span for 21 bytes — the friendship/marriage flags
		// still ride the wire immediately after the couple block.
		if len(got) != len(empty)+20 {
			t.Fatalf("couple-populated length: got %d want %d (empty %d + 20)", len(got), len(empty)+20, len(empty))
		}
		if !bytes.Equal(got[:offset], empty[:offset]) {
			t.Errorf("prefix before ring span changed:\n got %x\nwant %x", got[:offset], empty[:offset])
		}
		// Friendship + marriage flags (2 bytes) and everything after are
		// unchanged, just shifted by the couple block's +20 bytes.
		if !bytes.Equal(got[offset+21:], empty[offset+1:]) {
			t.Errorf("suffix after couple block changed:\n got %x\nwant %x", got[offset+21:], empty[offset+1:])
		}
		// The replaced span must equal exactly what model.RingSet.EncodeField
		// (Task 2/3, already covered by its own tests) writes for this couple
		// ring in isolation: this proves wiring, not codec correctness.
		w := response.NewWriter(nil)
		(model.RingSet{Couple: couple}).EncodeField(w, tenant.MustFromContext(pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)))
		wantSpan := w.Bytes() // couple(21) + friendship flag(1) + marriage flag(1) = 23 bytes
		gotSpan := got[offset : offset+len(wantSpan)]
		if !bytes.Equal(gotSpan, wantSpan) {
			t.Errorf("ring span:\n got %x\nwant %x", gotSpan, wantSpan)
		}
	})
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
			input := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 312, avatar, nil, false, 100, 200, 3, 37, model.RingSet{})
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
			input := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 312, avatar, nil, true, 100, 200, 6, 37, model.RingSet{})
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
			input := NewCharacterSpawn(999, 80, "PetOwner", guild, cts, 100, avatar, pets, false, 50, 60, 4, 0, model.RingSet{})
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
