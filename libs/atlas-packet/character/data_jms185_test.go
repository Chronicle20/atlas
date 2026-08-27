package character

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestCharacterDataByteOutputJMS185 pins the CharacterData blob nested inside
// SET_FIELD (op 0x7B) for jms_v185 byte-for-byte. The enclosing frame is pinned
// by TestSetFieldByteOutputJMS185 in field/clientbound (which carries the
// coverage-matrix verification marker for this cell) and asserts this blob as an opaque
// span; this test removes that opacity — every byte below cites a
// CInPacket read in MapleStory_dump_SCY.exe (IDB session a977912e).
//
// Read order, CharacterData::Decode @0x5137af, called with bBackwardUpdate = 0
// from CStage::OnSetField @0x7eebf4:
//
//	DecodeBuffer(p, 8)          @0x5137cf → dbcharFlag (64-bit mask; -1 = every
//	                                        section present, so every gate below
//	                                        is taken).
//	Decode1                     @0x5137d6 → SN-list flag. Zero here, so the two
//	                                        Decode4-counted DecodeBuffer(8) loops
//	                                        @0x5137f3/@0x513825 are skipped.
//	if (flag & 1)  GW_CharacterStat::Decode @0x513852 — see the stat block below.
//	               Decode1      @0x513863 → buddy capacity.
//	               Decode1      @0x513869 → linked-name flag (0 → no DecodeStr
//	                                        @0x513878).
//	if (flag & 2)  GW_CharacterStat::DecodeMoney @0x5138a8 → one Decode4 (meso,
//	                                        @0x50eeb6).
//	               DecodeBuffer(..., 0xC) @0x5138b8 → 12 opaque JMS bytes.
//	if (flag & 0x80)  5 × Decode1 @0x513a86 (loop j=1..5 @0x513a47) → the five
//	                                        inventory slot limits.
//	if (flag & 0x100000) Decode4 @0x513b23 + Decode4 @0x513b39 → the equip-slot
//	                                        extension expiry FILETIME.
//	if (flag & 4)  four Decode2-terminated item lists:
//	                 @0x513b7c equipped, @0x513c3c cash-equipped,
//	                 @0x513cf5 equip inventory, @0x513db4 dragon/mechanic
//	                 (accepted only for slot 1000..1003 @0x513ddb).
//	loop j=2..5 (@0x513e07): if (flag & (1<<j)) a Decode1-terminated list
//	                 @0x513e60 → use / setup / etc / cash inventories.
//	if (flag & 0x100)   Decode2 skill count @0x513f46.
//	if (flag & 0x8000)  Decode2 cooldown count @0x51400e.
//	if (flag & 0x200)   Decode2 started-quest count @0x514065, then a SECOND
//	                    Decode2 count @0x5140b1 of DecodeStr/DecodeStr pairs.
//	if (flag & 0x4000)  Decode2 completed-quest count @0x51411d.
//	if (flag & 0x400)   Decode2 mini-game-record count @0x514172.
//	if (flag & 0x800)   Decode2 couple @0x514227, Decode2 friend @0x514255,
//	                    Decode2 marriage @0x514283 ring-record counts.
//	if (flag & 0x1000)  5 × Decode4 @0x5142dc then 10 × Decode4 @0x5142f5 →
//	                    regular and VIP teleport-rock maps.
//	if (flag & 0x7C)    Decode2 count @0x51431c of sub_510F3A @0x510f3a records.
//	if (flag & 0x20000) Decode4 @0x5145c9 → monster-book cover.
//	if (flag & 0x10000) sub_51121F @0x5145f9 → Decode1 mode @0x51123b; mode 0
//	                    reads Decode2 count @0x5113be then (Decode2, Decode1) per
//	                    card @0x5113e6/@0x5113f6.
//	if (flag & 0x40000) Decode2 QuestEx count @0x51467c.
//	if (flag & 0x80000) Decode2 count @0x5146e1 of (Decode4, Decode2) pairs.
//
// GW_CharacterStat::Decode @0x50ec17 with bBackwardUpdate = 0:
//
//	Decode4 @0x50ec3a characterId · DecodeBuffer(13) @0x50ec4b name ·
//	Decode1 @0x50ec62 gender · Decode1 @0x50ec77 skin · Decode4 @0x50ec8c face ·
//	Decode4 @0x50eca1 hair · DecodeBuffer(24) @0x50ecac three pet locker SNs ·
//	Decode1 @0x50ecbb level · Decode2 @0x50ecc7 job ·
//	nine Decode2 @0x50ecdd/@0x50ecf1/@0x50ed05/@0x50ed19/@0x50ed2d/@0x50ed41/
//	@0x50ed55/@0x50ed69/@0x50ed7d → STR, DEX, INT, LUK, HP, MaxHP, MP, MaxMP, AP ·
//	Decode2 @0x50edd2 SP (the else arm; the extended-SP arm @0x50edc9 is taken
//	only when sub_5163A2(job) — job/1000==3 || job/100==22 || job==2001 — and job
//	0 is not such a job) · Decode4 @0x50edf7 EXP · Decode2 @0x50ee11 fame ·
//	Decode4 @0x50ee2b gachaExp · Decode4 @0x50ee45 map id ·
//	Decode1 @0x50ee5f portal · Decode2 @0x50ee6a · DecodeBuffer(8) @0x50ee7c ·
//	Decode4 @0x50ee8a · Decode4 @0x50ee97 · Decode4 @0x50eea3.
//
// The model below mirrors the character that reproduced the crash in
// docs/tasks/task-273-jms185-channel-enter-crash: id 40, level 1, map 10000,
// empty inventory.
func TestCharacterDataByteOutputJMS185(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)

	cd := CharacterData{
		Stats: CharacterStats{
			Id: 40, Name: "Chronicle", Gender: 0, SkinColor: 0,
			Face: 20000, Hair: 30000,
			Level: 1, JobId: 0,
			Str: 12, Dex: 5, Int: 4, Luk: 4,
			Hp: 50, MaxHp: 50, Mp: 5, MaxMp: 5,
			Ap: 0, Sp: 0, Exp: 0, Fame: 0, GachaExp: 0,
			MapId: 10000, SpawnPoint: 0,
		},
		BuddyCapacity: 20,
		Meso:          0,
		Inventory: InventoryData{
			EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
			EtcCapacity: 24, CashCapacity: 24,
			EquipSlotExtExpire: 0,
		},
	}

	empty := []byte{0xFF, 0xC9, 0x9A, 0x3B} // _map.EmptyMapId = 999999999

	var expected []byte
	add := func(b ...byte) { expected = append(expected, b...) }

	// --- header -------------------------------------------------------------
	add(0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF) // dbcharFlag = -1 @0x5137cf
	add(0x00)                                           // SN-list flag @0x5137d6

	// --- GW_CharacterStat::Decode @0x513852 ---------------------------------
	add(0x28, 0x00, 0x00, 0x00)                                              // characterId 40 @0x50ec3a
	add('C', 'h', 'r', 'o', 'n', 'i', 'c', 'l', 'e', 0x00, 0x00, 0x00, 0x00) // name, 13 bytes @0x50ec4b
	add(0x00)                                                                // gender @0x50ec62
	add(0x00)                                                                // skin @0x50ec77
	add(0x20, 0x4E, 0x00, 0x00)                                              // face 20000 @0x50ec8c
	add(0x30, 0x75, 0x00, 0x00)                                              // hair 30000 @0x50eca1
	for i := 0; i < 24; i++ {                                                // three pet locker SNs @0x50ecac
		add(0x00)
	}
	add(0x01)                                           // level @0x50ecbb
	add(0x00, 0x00)                                     // job @0x50ecc7
	add(0x0C, 0x00)                                     // STR @0x50ecdd
	add(0x05, 0x00)                                     // DEX @0x50ecf1
	add(0x04, 0x00)                                     // INT @0x50ed05
	add(0x04, 0x00)                                     // LUK @0x50ed19
	add(0x32, 0x00)                                     // HP @0x50ed2d
	add(0x32, 0x00)                                     // MaxHP @0x50ed41
	add(0x05, 0x00)                                     // MP @0x50ed55
	add(0x05, 0x00)                                     // MaxMP @0x50ed69
	add(0x00, 0x00)                                     // AP @0x50ed7d
	add(0x00, 0x00)                                     // SP @0x50edd2
	add(0x00, 0x00, 0x00, 0x00)                         // EXP @0x50edf7
	add(0x00, 0x00)                                     // fame @0x50ee11
	add(0x00, 0x00, 0x00, 0x00)                         // gachaExp @0x50ee2b
	add(0x10, 0x27, 0x00, 0x00)                         // map 10000 @0x50ee45
	add(0x00)                                           // portal @0x50ee5f
	add(0x00, 0x00)                                     // @0x50ee6a
	add(0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // @0x50ee7c
	add(0x00, 0x00, 0x00, 0x00)                         // @0x50ee8a
	add(0x00, 0x00, 0x00, 0x00)                         // @0x50ee97
	add(0x00, 0x00, 0x00, 0x00)                         // @0x50eea3

	add(0x14) // buddy capacity 20 @0x513863
	add(0x00) // linked-name flag @0x513869

	// --- money + JMS 12-byte block ------------------------------------------
	add(0x00, 0x00, 0x00, 0x00) // meso @0x50eeb6
	add(0x28, 0x00, 0x00, 0x00) // DecodeBuffer(0xC) @0x5138b8: characterId,
	add(0x00, 0x00, 0x00, 0x00) //   dama,
	add(0x00, 0x00, 0x00, 0x00) //   reserved

	// --- inventory ----------------------------------------------------------
	add(0x18, 0x18, 0x18, 0x18, 0x18)                   // 5 slot limits @0x513a86
	add(0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // equip-ext expiry @0x513b23/@0x513b39
	add(0x00, 0x00)                                     // equipped terminator @0x513b7c
	add(0x00, 0x00)                                     // cash-equipped terminator @0x513c3c
	add(0x00, 0x00)                                     // equip-inventory terminator @0x513cf5
	add(0x00, 0x00)                                     // dragon/mechanic terminator @0x513db4
	add(0x00)                                           // use terminator @0x513e60 (j=2)
	add(0x00)                                           // setup terminator @0x513e60 (j=3)
	add(0x00)                                           // etc terminator @0x513e60 (j=4)
	add(0x00)                                           // cash terminator @0x513e60 (j=5)

	// --- skills / quests / mini-games / rings --------------------------------
	add(0x00, 0x00) // skill count @0x513f46
	add(0x00, 0x00) // cooldown count @0x51400e
	add(0x00, 0x00) // started-quest count @0x514065
	add(0x00, 0x00) // quest string-pair count @0x5140b1
	add(0x00, 0x00) // completed-quest count @0x51411d
	add(0x00, 0x00) // mini-game-record count @0x514172
	add(0x00, 0x00) // couple-ring count @0x514227
	add(0x00, 0x00) // friend-ring count @0x514255
	add(0x00, 0x00) // marriage-ring count @0x514283

	// --- teleport rocks @0x5142dc / @0x5142f5 -------------------------------
	for i := 0; i < 5; i++ {
		add(empty...)
	}
	for i := 0; i < 10; i++ {
		add(empty...)
	}

	// --- tail ---------------------------------------------------------------
	add(0x00, 0x00)             // flag&0x7C record count @0x51431c
	add(0x00, 0x00, 0x00, 0x00) // monster-book cover @0x5145c9
	add(0x00)                   // monster-book card mode @0x51123b
	add(0x00, 0x00)             // monster-book card count @0x5113be
	add(0x00, 0x00)             // QuestEx count @0x51467c
	add(0x00, 0x00)             // flag&0x80000 count @0x5146e1

	actual := pt.Encode(t, ctx, cd.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Fatalf("jms_v185 CharacterData golden mismatch (%d bytes vs %d):\n got %v\nwant %v",
			len(actual), len(expected), actual, expected)
	}
}
