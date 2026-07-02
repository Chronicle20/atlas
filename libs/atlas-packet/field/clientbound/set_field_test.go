package clientbound

import (
	"bytes"
	"testing"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSetField version=gms_v79 ida=0x6f07d9
// packet-audit:verify packet=field/clientbound/FieldSetField version=gms_v83 ida=0x776020
// packet-audit:verify packet=field/clientbound/FieldSetField version=gms_v87 ida=0x7c429c
// packet-audit:verify packet=field/clientbound/FieldSetField version=gms_v95 ida=0x71a0a0
// packet-audit:verify packet=field/clientbound/FieldSetField version=jms_v185 ida=0x7eea69
// packet-audit:verify packet=field/clientbound/FieldSetField version=gms_v84 ida=0x798987
// packet-audit:verify packet=field/clientbound/FieldSetField version=gms_v72 ida=0x6c0c9b
// TestSetFieldByteOutputV79 pins the gms_v79 SET_FIELD (op 0x76) clientbound
// wire HEADER and TRAILER. IDA: CStage::OnSetField @0x6f07d9 (GMS_v79_1_DEVM.exe)
// reads, in order —
//
//	Decode4(channelId)          @0x6f080c → channel id (int32 LE).
//	Decode1(sNotifierMessage)   @0x6f082b → notifier byte.
//	Decode1(bCharacterData)     @0x6f0838 → full-character-data flag.
//	Decode2(nNotifierCheck)     @0x6f084f → notifier count (0 → no notifier strings).
//	if (bCharacterData) 3×Decode4(seeds) @0x6f08f3/@0x6f08fd/@0x6f0912 → 3 RNG seeds.
//	  CharacterData::Decode      @0x6f093b → the opaque CharacterData blob.
//	DecodeBuffer(v93, 8)        @0x6f0a76 → 8-byte timestamp.
//
// v79 is GMS<87 so there is NO decode-opt header and NO logout-gift block (both
// MajorAtLeast(87)-gated); GMS<95 omits m_dwOldDriverID. Per the §5 opaque
// caveat, the CharacterData middle cannot cite per-field decompile lines — it is
// derived from the Atlas CharacterData encoder and asserted as an opaque span;
// the header (channelId..3 seeds) and the trailing 8-byte timestamp are pinned
// byte-exact to the decompile lines above.
func TestSetFieldByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	cd := charpkt.CharacterData{
		Stats: charpkt.CharacterStats{
			Id: 1000, Name: "TestChar", Gender: 0, SkinColor: 1,
			Face: 20000, Hair: 30000,
			Level: 50, JobId: 312, Str: 100, Dex: 50, Int: 30, Luk: 20,
			Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
			Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
			MapId: 100000000, SpawnPoint: 0,
		},
		BuddyCapacity: 20,
		Meso:          100000,
		Inventory: charpkt.InventoryData{
			EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
			EtcCapacity: 24, CashCapacity: 24,
			Timestamp: 94354848000000000,
		},
	}
	input := SetField{
		channelId:     channel.Id(1),
		characterData: cd,
		damageSeeds:   []uint32{0x11111111, 0x22222222, 0x33333333, 0x44444444},
		timestamp:     0x0011223344556677,
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)

	header := []byte{
		0x01, 0x00, 0x00, 0x00, // channelId=1 @0x6f080c
		0x01,       // sNotifierMessage @0x6f082b
		0x01,       // bCharacterData @0x6f0838
		0x00, 0x00, // nNotifierCheck=0 @0x6f084f
		0x11, 0x11, 0x11, 0x11, // seed[0] @0x6f08f3
		0x22, 0x22, 0x22, 0x22, // seed[1] @0x6f08fd
		0x33, 0x33, 0x33, 0x33, // seed[2] @0x6f0912
	}
	trailer := []byte{0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00} // timestamp int64-LE @0x6f0a76
	cdBytes := pt.Encode(t, ctx, cd.Encode, nil)                      // opaque CharacterData span (§5)
	expected := append(append(append([]byte{}, header...), cdBytes...), trailer...)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 set_field golden mismatch:\n got %v\nwant %v", actual, expected)
	}
}

// TestSetFieldByteOutputV72 pins the gms_v72 SET_FIELD (op 114) clientbound
// wire HEADER and TRAILER. IDA: CStage::OnSetField @0x6c0c9b (GMS_v72.1_U_DEVM.exe)
// reads, in order —
//
//	Decode4(channelId)          @0x6c0cce → channel id (int32 LE).
//	Decode1(sNotifierMessage)   @0x6c0ced → notifier byte.
//	Decode1(bCharacterData)     @0x6c0cfa → full-character-data flag.
//	Decode2(nNotifierCheck)     @0x6c0d11 → notifier count (0 → no notifier strings).
//	if (bCharacterData) 3×Decode4(seeds) @0x6c0db5/@0x6c0dbf/@0x6c0dd4 → 3 RNG seeds.
//	  CharacterData::Decode      @0x6c0dfd → the opaque CharacterData blob.
//	DecodeBuffer(v102, 8)       @0x6c0f38 → 8-byte timestamp.
//
// v72 is GMS<87 so there is NO decode-opt header and NO logout-gift block (both
// MajorAtLeast(87)-gated); GMS<95 omits m_dwOldDriverID. This matches the v79
// legacy branch byte-for-byte (framing read order identical), so the codec needs
// no version gate. Per the §5 opaque caveat, the CharacterData middle cannot cite
// per-field decompile lines — it is derived from the Atlas CharacterData encoder
// and asserted as an opaque span; the header (channelId..3 seeds) and the trailing
// 8-byte timestamp are pinned byte-exact to the decompile lines above.
func TestSetFieldByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	cd := charpkt.CharacterData{
		Stats: charpkt.CharacterStats{
			Id: 1000, Name: "TestChar", Gender: 0, SkinColor: 1,
			Face: 20000, Hair: 30000,
			Level: 50, JobId: 312, Str: 100, Dex: 50, Int: 30, Luk: 20,
			Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
			Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
			MapId: 100000000, SpawnPoint: 0,
		},
		BuddyCapacity: 20,
		Meso:          100000,
		Inventory: charpkt.InventoryData{
			EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
			EtcCapacity: 24, CashCapacity: 24,
			Timestamp: 94354848000000000,
		},
	}
	input := SetField{
		channelId:     channel.Id(1),
		characterData: cd,
		damageSeeds:   []uint32{0x11111111, 0x22222222, 0x33333333, 0x44444444},
		timestamp:     0x0011223344556677,
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)

	header := []byte{
		0x01, 0x00, 0x00, 0x00, // channelId=1 @0x6c0cce
		0x01,       // sNotifierMessage @0x6c0ced
		0x01,       // bCharacterData @0x6c0cfa
		0x00, 0x00, // nNotifierCheck=0 @0x6c0d11
		0x11, 0x11, 0x11, 0x11, // seed[0] @0x6c0db5
		0x22, 0x22, 0x22, 0x22, // seed[1] @0x6c0dbf
		0x33, 0x33, 0x33, 0x33, // seed[2] @0x6c0dd4
	}
	trailer := []byte{0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00} // timestamp int64-LE @0x6c0f38
	cdBytes := pt.Encode(t, ctx, cd.Encode, nil)                      // opaque CharacterData span (§5)
	expected := append(append(append([]byte{}, header...), cdBytes...), trailer...)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 set_field golden mismatch:\n got %v\nwant %v", actual, expected)
	}
}

// TestSetFieldByteOutputV61 pins the gms_v61 SET_FIELD (op 0x5C = 92) clientbound
// framing. IDA: CStage::OnSetField @0x659fd3 (GMS_v61.1_U_DEVM.exe). v61 is GMS<87
// and <95, so — like v72 — there is NO decode-opt header, NO m_dwOldDriverID, and
// NO logout-gift block; the >28 damage-seed path (3 seeds) applies. Framing read
// order is byte-identical to v72; per the §5 opaque caveat the CharacterData middle
// is asserted as an opaque span while the header and trailing timestamp are pinned.
// packet-audit:verify packet=field/clientbound/FieldSetField version=gms_v61 ida=0x659fd3
func TestSetFieldByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	cd := charpkt.CharacterData{
		Stats: charpkt.CharacterStats{
			Id: 1000, Name: "TestChar", Gender: 0, SkinColor: 1,
			Face: 20000, Hair: 30000,
			Level: 50, JobId: 312, Str: 100, Dex: 50, Int: 30, Luk: 20,
			Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
			Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
			MapId: 100000000, SpawnPoint: 0,
		},
		BuddyCapacity: 20,
		Meso:          100000,
		Inventory: charpkt.InventoryData{
			EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
			EtcCapacity: 24, CashCapacity: 24,
			Timestamp: 94354848000000000,
		},
	}
	input := SetField{
		channelId:     channel.Id(1),
		characterData: cd,
		damageSeeds:   []uint32{0x11111111, 0x22222222, 0x33333333, 0x44444444},
		timestamp:     0x0011223344556677,
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)

	header := []byte{
		0x01, 0x00, 0x00, 0x00, // channelId=1
		0x01,       // sNotifierMessage
		0x01,       // bCharacterData
		0x00, 0x00, // nNotifierCheck=0
		0x11, 0x11, 0x11, 0x11, // seed[0]
		0x22, 0x22, 0x22, 0x22, // seed[1]
		0x33, 0x33, 0x33, 0x33, // seed[2]
	}
	trailer := []byte{0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00} // timestamp int64-LE
	cdBytes := pt.Encode(t, ctx, cd.Encode, nil)                      // opaque CharacterData span (§5)
	expected := append(append(append([]byte{}, header...), cdBytes...), trailer...)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v61 set_field golden mismatch:\n got %v\nwant %v", actual, expected)
	}
}

func TestSetFieldRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cd := charpkt.CharacterData{
				Stats: charpkt.CharacterStats{
					Id: 1000, Name: "TestChar", Gender: 0, SkinColor: 1,
					Face: 20000, Hair: 30000,
					Level: 50, JobId: 312, Str: 100, Dex: 50, Int: 30, Luk: 20,
					Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
					Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
					MapId: 100000000, SpawnPoint: 0,
				},
				BuddyCapacity: 20,
				Meso:          100000,
				Inventory: charpkt.InventoryData{
					EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
					EtcCapacity: 24, CashCapacity: 24,
					Timestamp: 94354848000000000,
				},
			}
			input := NewSetField(channel.Id(1), cd)
			output := SetField{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterData().Stats.Id != cd.Stats.Id {
				t.Errorf("stats id: got %v, want %v", output.CharacterData().Stats.Id, cd.Stats.Id)
			}
			if output.CharacterData().Stats.Name != cd.Stats.Name {
				t.Errorf("stats name: got %q, want %q", output.CharacterData().Stats.Name, cd.Stats.Name)
			}
			if output.CharacterData().Meso != cd.Meso {
				t.Errorf("meso: got %v, want %v", output.CharacterData().Meso, cd.Meso)
			}
		})
	}
}
