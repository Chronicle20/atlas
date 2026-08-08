package model

import (
	"bytes"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestEncodeSlotVersionGate guards the equip inventory slot width: byte for
// legacy GMS (<83), short for v83+. The only wire difference between a v79 and a
// v83 regular-equip encode is that widened slot, so v83 must be exactly 1 byte
// longer, and the 0x01 type discriminator lands right after a 1-byte (v79) or
// 2-byte (v83) slot. pt.Variants has no version in [29,82], so this pins it.
func TestEncodeSlotVersionGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	a := NewAsset(false, -5, 1302000, time.Time{}). // equipped slot -5, equip template
							SetEquipmentStats(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0).
							SetEquipmentMeta(0, 0, 0, 0, 0, 0)

	v79 := a.Encode(l, test.CreateContext("GMS", 79, 1))(nil)
	v83 := a.Encode(l, test.CreateContext("GMS", 83, 1))(nil)

	if len(v83) != len(v79)+1 {
		t.Errorf("equip encode lengths: v79=%d v83=%d, want v83 == v79+1 (byte vs short slot)", len(v79), len(v83))
	}
	if v79[0] != 5 || v79[1] != 1 {
		t.Errorf("v79 equip prefix = %v, want [5 1] (byte slot then 0x01 type)", v79[:2])
	}
	if v83[0] != 5 || v83[1] != 0 || v83[2] != 1 {
		t.Errorf("v83 equip prefix = %v, want [5 0 1] (short slot then 0x01 type)", v83[:3])
	}
}

func TestAssetEquipable(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	a := NewAsset(false, -5, 1302000, exp). // equip slot -5, templateId in equip range (1xxxxxx)
						SetEquipmentStats(10, 11, 12, 13, 100, 50, 80, 70, 30, 25, 15, 20, 10, 5, 3).
						SetEquipmentMeta(7, 1, 2, 500, 3, 0x0001)

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()
			encoded := a.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Fatal("encoded bytes should not be empty")
			}
			// Verify we can re-encode and get the same bytes (deterministic).
			encoded2 := a.Encode(l, ctx)(nil)
			if len(encoded) != len(encoded2) {
				t.Fatalf("re-encode produced different length: %d vs %d", len(encoded), len(encoded2))
			}
		})
	}
}

func TestAssetCashEquipable(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	a := NewAsset(false, -11, 1302000, exp).
		SetEquipmentStats(10, 11, 12, 13, 100, 50, 80, 70, 30, 25, 15, 20, 10, 5, 3).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0).
		SetCashId(90000001)

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()
			encoded := a.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Fatal("encoded bytes should not be empty")
			}
		})
	}
}

func TestAssetStackable(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	a := NewAsset(false, 3, 2000000, exp). // consumable (2xxxxxx)
						SetStackableInfo(100, 0x0001, 0)

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()
			encoded := a.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Fatal("encoded bytes should not be empty")
			}
		})
	}
}

func TestAssetPetCashItem(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	a := NewAsset(false, 1, 5000000, exp). // cash item (5xxxxxx)
						SetPetInfo(1001, "Snowy", 10, 100, 200)

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()
			encoded := a.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Fatal("encoded bytes should not be empty")
			}
		})
	}
}

func TestAssetCashItem(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	a := NewAsset(false, 1, 5000000, exp). // cash item (5xxxxxx)
						SetCashId(90000002).
						SetStackableInfo(1, 0, 0)

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()
			encoded := a.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Fatal("encoded bytes should not be empty")
			}
		})
	}
}

func TestAssetZeroPosition(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	a := NewAsset(true, 3, 2000000, exp).
		SetStackableInfo(50, 0, 0)

	ctx := test.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()
	encoded := a.Encode(l, ctx)(nil)

	// When zeroPosition=true, the slot byte should NOT be written.
	// Compare with zeroPosition=false to verify slot is omitted.
	b := NewAsset(false, 3, 2000000, exp).
		SetStackableInfo(50, 0, 0)
	encodedWithSlot := b.Encode(l, ctx)(nil)

	// zeroPosition=true should produce 1 byte less (int8 slot)
	if len(encodedWithSlot)-len(encoded) != 1 {
		t.Errorf("expected 1 byte difference for slot, got %d", len(encodedWithSlot)-len(encoded))
	}
}

func TestAssetGetters(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	a := NewAsset(false, -5, 1302000, exp).
		SetEquipmentStats(10, 11, 12, 13, 100, 50, 80, 70, 30, 25, 15, 20, 10, 5, 3).
		SetEquipmentMeta(7, 1, 2, 500, 3, 0x0001).
		SetCashId(12345).
		SetPetInfo(42, "Buddy", 5, 80, 150)

	assertEqual(t, "ZeroPosition", false, a.ZeroPosition())
	assertEqual(t, "Slot", int16(-5), a.Slot())
	assertEqual(t, "TemplateId", uint32(1302000), a.TemplateId())
	assertEqual(t, "Expiration", exp, a.Expiration())
	assertEqual(t, "Strength", uint16(10), a.Strength())
	assertEqual(t, "Dexterity", uint16(11), a.Dexterity())
	assertEqual(t, "Intelligence", uint16(12), a.Intelligence())
	assertEqual(t, "Luck", uint16(13), a.Luck())
	assertEqual(t, "Hp", uint16(100), a.Hp())
	assertEqual(t, "Mp", uint16(50), a.Mp())
	assertEqual(t, "WeaponAttack", uint16(80), a.WeaponAttack())
	assertEqual(t, "MagicAttack", uint16(70), a.MagicAttack())
	assertEqual(t, "WeaponDefense", uint16(30), a.WeaponDefense())
	assertEqual(t, "MagicDefense", uint16(25), a.MagicDefense())
	assertEqual(t, "Accuracy", uint16(15), a.Accuracy())
	assertEqual(t, "Avoidability", uint16(20), a.Avoidability())
	assertEqual(t, "Hands", uint16(10), a.Hands())
	assertEqual(t, "Speed", uint16(5), a.Speed())
	assertEqual(t, "Jump", uint16(3), a.Jump())
	assertEqual(t, "Slots", uint16(7), a.Slots())
	assertEqual(t, "LevelType", byte(1), a.LevelType())
	assertEqual(t, "Level", byte(2), a.Level())
	assertEqual(t, "Experience", uint32(500), a.Experience())
	assertEqual(t, "HammersApplied", uint32(3), a.HammersApplied())
	assertEqual(t, "Flag", uint16(0x0001), a.Flag())
	assertEqual(t, "CashId", int64(12345), a.CashId())
	assertEqual(t, "PetId", uint32(42), a.PetId())
	assertEqual(t, "PetName", "Buddy", a.PetName())
	assertEqual(t, "PetLevel", byte(5), a.PetLevel())
	assertEqual(t, "Closeness", uint16(150), a.Closeness())
	assertEqual(t, "Fullness", byte(80), a.Fullness())
}

func TestAssetTypeDetection(t *testing.T) {
	equip := NewAsset(false, -1, 1302000, time.Time{}) // 1xxxxxx = equip
	if !equip.IsEquipment() {
		t.Error("expected IsEquipment")
	}

	cashEquip := NewAsset(false, -1, 1302000, time.Time{}).SetCashId(1)
	if !cashEquip.IsCashEquipment() {
		t.Error("expected IsCashEquipment")
	}

	consumable := NewAsset(false, 1, 2000000, time.Time{}) // 2xxxxxx = use
	if !consumable.IsConsumable() {
		t.Error("expected IsConsumable")
	}

	setup := NewAsset(false, 1, 3000000, time.Time{}) // 3xxxxxx = setup
	if !setup.IsSetup() {
		t.Error("expected IsSetup")
	}

	etc := NewAsset(false, 1, 4000000, time.Time{}) // 4xxxxxx = etc
	if !etc.IsEtc() {
		t.Error("expected IsEtc")
	}

	cash := NewAsset(false, 1, 5000000, time.Time{}) // 5xxxxxx = cash
	if !cash.IsCash() {
		t.Error("expected IsCash")
	}

	pet := NewAsset(false, 1, 5000000, time.Time{}).SetPetInfo(1, "Pet", 1, 100, 100)
	if !pet.IsPet() {
		t.Error("expected IsPet")
	}
}

func TestMsTime(t *testing.T) {
	if MsTime(time.Time{}) != -1 {
		t.Error("zero time should return -1")
	}
	ts := time.Unix(1000, 0)
	expected := int64(1000)*10000000 + 116444736000000000
	if MsTime(ts) != expected {
		t.Errorf("expected %d, got %d", expected, MsTime(ts))
	}
}

func TestAssetDeterministicEncode(t *testing.T) {
	// Verify that encoding the same asset with the same context produces identical bytes.
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	a := NewAsset(false, -5, 1302000, exp).
		SetEquipmentStats(10, 11, 12, 13, 100, 50, 80, 70, 30, 25, 15, 20, 10, 5, 3).
		SetEquipmentMeta(7, 1, 2, 500, 3, 0x0001)

	ctx := test.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()

	// We need fresh writers each time since writer state is consumed.
	bytes1 := a.Encode(l, ctx)(nil)
	bytes2 := a.Encode(l, ctx)(nil)

	if len(bytes1) != len(bytes2) {
		t.Fatalf("lengths differ: %d vs %d", len(bytes1), len(bytes2))
	}
	for i := range bytes1 {
		if bytes1[i] != bytes2[i] {
			t.Fatalf("byte %d differs: %02x vs %02x", i, bytes1[i], bytes2[i])
		}
	}
}

func assertEqual[T comparable](t *testing.T, name string, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", name, expected, actual)
	}
}

func TestAssetOwnerEncoded(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	base := NewAsset(false, -5, 1302000, exp).
		SetEquipmentStats(10, 11, 12, 13, 100, 50, 80, 70, 30, 25, 15, 20, 10, 5, 3).
		SetEquipmentMeta(7, 1, 2, 500, 3, 0x0001)
	named := base.SetOwner("Tumi")
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			plain := base.Encode(l, ctx)(nil)
			withOwner := named.Encode(l, ctx)(nil)
			if len(withOwner) != len(plain)+len("Tumi") {
				t.Fatalf("owner bytes not encoded: len(withOwner)=%d len(plain)=%d", len(withOwner), len(plain))
			}
			// empty owner must be byte-identical to the pre-change encoding
			baseEmptyOwner := base.SetOwner("")
			empty := baseEmptyOwner.Encode(l, ctx)(nil)
			if !bytes.Equal(empty, plain) {
				t.Fatal("empty owner changed the wire bytes")
			}
		})
	}
}

func TestAssetOwnerEncodedStackable(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	base := NewAsset(false, 3, 2000000, exp).SetStackableInfo(50, 0, 0)
	named := base.SetOwner("Tumi")
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 83, 1)
	plain := base.Encode(l, ctx)(nil)
	withOwner := named.Encode(l, ctx)(nil)
	if len(withOwner) != len(plain)+len("Tumi") {
		t.Fatalf("owner bytes not encoded on stackable: %d vs %d", len(withOwner), len(plain))
	}
}

// TestAssetPetSerialNumber pins GW_ItemSlotBase::liCashItemSN for a pet. The
// client matches the cash-shop locker entry on this value
// (CCashShop::OnCashItemResMoveLtoSDone GMS v83 @0x47aee2 compares it against
// GW_CashItemInfo::liSN) and binds the spawned pet to its inventory slot on it
// (CPet::GetItemSlot @0x703af3). Emitting the Atlas pet id for a cash-purchased
// pet is what left the withdrawn pet stuck in the locker UI.
func TestAssetPetSerialNumber(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 83, 1)

	// type(1) templateId(4) bool(1) => liCashItemSN occupies bytes [6,14).
	const snOffset = 6
	read := func(b []byte) uint64 {
		var v uint64
		for i := 7; i >= 0; i-- {
			v = v<<8 | uint64(b[snOffset+i])
		}
		return v
	}

	const serial = uint64(8688106441904477350) // a real serial from the reported case
	// The serial exceeds MaxUint32, so a pet id can never stand in for it: this
	// also pins that it is not truncated on the way out.
	purchased := NewAsset(true, 0, 5000012, time.Time{}).
		SetCashId(int64(serial)).
		SetPetInfo(1, "Mr. Roboto", 1, 100, 0).
		SetPetSerialNumber(serial)
	if got := purchased.PetSerialNumber(); got != serial {
		t.Fatalf("PetSerialNumber() = %d, want %d", got, serial)
	}
	if got := read(purchased.Encode(l, ctx)(map[string]interface{}{})); got != serial {
		t.Fatalf("encoded liCashItemSN = %d, want %d (not the pet id)", got, serial)
	}

	// No serial recorded (pet never came from the cash shop): fall back to the
	// pet id so it stays addressable. There is no locker entry to match. The
	// asset's own cashId must NOT be substituted — the pet record is the single
	// source, and for such a pet it has no serial to give.
	granted := NewAsset(true, 0, 5000012, time.Time{}).
		SetCashId(999).
		SetPetInfo(7, "Mr. Roboto", 1, 100, 0)
	if got := granted.PetSerialNumber(); got != 7 {
		t.Fatalf("PetSerialNumber() = %d, want the pet id 7", got)
	}
	if got := read(granted.Encode(l, ctx)(map[string]interface{}{})); got != 7 {
		t.Fatalf("encoded liCashItemSN = %d, want 7", got)
	}
}

// TestAssetPetCashItemDeadDate pins GW_ItemSlotPet::dateDead (the 8-byte
// FILETIME at struct offset +89, GW_ItemSlotPet::RawDecode GMS v83 @0x4e4219).
//
// The client reads a pet as dried up iff
// CompareFileTime(dateDead, 150842304000000000 /* 2079-01-01 */) >= 0
// (GMS v83 sub_4E4044 @0x4E4044; CUIToolTip::GetPetDeadDate @0x8ebfde then
// renders "The water of life has dried up"). CompareFileTime is unsigned, so
// the MsTime zero sentinel (-1 = max FILETIME) is ABOVE the threshold and
// reads as dead. Two invariants follow, and both are pinned here:
//
//  1. dateDead comes from the pet's own life clock (SetPetDeadDate), never
//     from the cash item's Expiration.
//  2. an unset dead date encodes as 0, not as the -1 sentinel.
func TestAssetPetCashItemDeadDate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 83, 1)

	// type(1) templateId(4) bool(1) petId(8) dateExpire(8) name(13) level(1)
	// closeness(2) fullness(1) => dateDead occupies bytes [39,47).
	const deadDateOffset = 39
	const permanentThreshold = int64(150842304000000000) // dword_AF30B0, 2079-01-01

	read := func(b []byte) int64 {
		var v int64
		for i := 7; i >= 0; i-- {
			v = v<<8 | int64(b[deadDateOffset+i])
		}
		return v
	}

	itemExpiration := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	petDeath := time.Date(2026, 11, 6, 0, 7, 41, 0, time.UTC)

	base := NewAsset(true, 0, 5000012, itemExpiration).
		SetCashId(123).
		SetPetInfo(42, "Mr. Roboto", 3, 100, 50)

	// The sentinel this field must never emit. Pinned so the contrast below is
	// self-evidencing: encoding MsTime(zero) here is exactly the regression.
	if sentinel := MsTime(time.Time{}); uint64(sentinel) < uint64(permanentThreshold) {
		t.Fatalf("MsTime zero sentinel = %d; expected it to exceed the dried-up threshold", sentinel)
	}

	// 1. No dead date recorded: must be 0 (alive), NOT the MsTime -1 sentinel.
	unset := read(base.Encode(l, ctx)(map[string]interface{}{}))
	if unset != 0 {
		t.Fatalf("unset dateDead = %d, want 0", unset)
	}
	if unset >= permanentThreshold {
		t.Fatalf("unset dateDead = %d reads as dried up (>= %d)", unset, permanentThreshold)
	}

	// 2. Recorded dead date wins, and it is the PET's date, not the item's.
	dated := base.SetPetDeadDate(petDeath)
	got := read(dated.Encode(l, ctx)(map[string]interface{}{}))
	if got != MsTime(petDeath) {
		t.Fatalf("dateDead = %d, want %d (pet life clock)", got, MsTime(petDeath))
	}
	if got == MsTime(itemExpiration) {
		t.Fatal("dateDead must not be sourced from the cash item Expiration")
	}
	if got >= permanentThreshold {
		t.Fatalf("dateDead = %d reads as dried up (>= %d)", got, permanentThreshold)
	}
}

// TestAssetPetCashItemSkillMask pins the DOM-25 translation: the Atlas-canonical
// petFlag mask must never reach the wire directly. It only encodes a wire bit
// when the tenant's petSkill options table configures one for that semantic key.
func TestAssetPetCashItemSkillMask(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 83, 1)
	expiration := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	base := NewAsset(true, 0, 5000017, expiration).
		SetCashId(123).
		SetPetInfo(42, "Mr. Roboto", 3, 100, 50)

	// Layout of encodePetCashItemInfo with zeroPosition=true:
	// type(1) templateId(4) bool(1) petId(8) time(8) name(13) level(1)
	// closeness(2) fullness(1) expiration(8) attribute(2) => skill short at offset 49.
	const skillOffset = 49

	zeroFlag := base.Encode(l, ctx)(map[string]interface{}{})

	// 1. petFlag set but no petSkill table configured -> byte-identical to zero-flag encode.
	flagged := base.SetPetFlag(2) // FlagConsumeHP (1<<1), Atlas-canonical
	noTable := flagged.Encode(l, ctx)(map[string]interface{}{})
	if !bytes.Equal(zeroFlag, noTable) {
		t.Fatal("petFlag with no petSkill table must encode byte-identical to zero flag")
	}

	// 2. petFlag set with a configured table -> wire bit at the skill short.
	withTable := flagged.Encode(l, ctx)(map[string]interface{}{
		"petSkill": map[string]interface{}{"consumeHP": "0x20"},
	})
	if len(withTable) != len(zeroFlag) {
		t.Fatalf("length changed: got %d, want %d", len(withTable), len(zeroFlag))
	}
	if withTable[skillOffset] != 0x20 || withTable[skillOffset+1] != 0x00 {
		t.Errorf("skill short = %#x %#x, want 0x20 0x00", withTable[skillOffset], withTable[skillOffset+1])
	}
	// everything else unchanged
	for i := range zeroFlag {
		if i == skillOffset || i == skillOffset+1 {
			continue
		}
		if withTable[i] != zeroFlag[i] {
			t.Fatalf("byte %d changed: got %#x, want %#x", i, withTable[i], zeroFlag[i])
		}
	}

	// 3. multiple flags OR together (autoSpeaking 1<<8 canonical -> 0x100 wire).
	multiFlags := base.SetPetFlag(2 | 256) // FlagConsumeHP | FlagAutoSpeaking
	multi := multiFlags.Encode(l, ctx)(map[string]interface{}{
		"petSkill": map[string]interface{}{"consumeHP": "0x20", "autoSpeaking": "0x100"},
	})
	if multi[skillOffset] != 0x20 || multi[skillOffset+1] != 0x01 {
		t.Errorf("multi skill short = %#x %#x, want 0x20 0x01", multi[skillOffset], multi[skillOffset+1])
	}

	if got := flagged.PetFlag(); got != 2 {
		t.Errorf("PetFlag() = %d, want 2", got)
	}
}

// TestAssetPetCashItemTrailerVersionGate pins the version-gated pet trailer
// (GW_ItemSlotPet::RawDecode, IDA-verified): v48 (@0x49c77e) and v61
// (@0x4b52f2) read neither remainLife nor the trailing attribute short, v72
// adds remainLife only (@0x4d06dd), v79 (@0x4d84c4) and v83 (@0x4e4219) read
// both. JMS is not a legacy client and keeps the full trailer. This is the
// regression guard: every version except v48/v61/v72 must stay
// byte-identical to today's always-full-trailer encode (57 bytes for this
// fixture).
func TestAssetPetCashItemTrailerVersionGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	expiration := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	base := NewAsset(true, 0, 5000017, expiration).
		SetCashId(123).
		SetPetInfo(42, "Mr. Roboto", 3, 100, 50)

	const wantV83Len = 57 // today's always-full-trailer length; pinned via a pre-change baseline run.

	lengths := map[string]int{}
	for _, v := range []struct {
		name   string
		region string
		major  uint16
	}{
		{"v48", "GMS", 48},
		{"v61", "GMS", 61},
		{"v72", "GMS", 72},
		{"v79", "GMS", 79},
		{"v83", "GMS", 83},
		{"v84", "GMS", 84},
		{"v87", "GMS", 87},
		{"v95", "GMS", 95},
		{"jms", "JMS", 185},
	} {
		ctx := test.CreateContext(v.region, v.major, 1)
		encoded := base.Encode(l, ctx)(map[string]interface{}{})
		lengths[v.name] = len(encoded)
	}

	if lengths["v83"] != wantV83Len {
		t.Fatalf("v83 length = %d, want %d (regression baseline)", lengths["v83"], wantV83Len)
	}
	if lengths["v48"] != wantV83Len-6 {
		t.Errorf("v48 length = %d, want %d (6 bytes shorter than v83: no remainLife, no trailing attribute)", lengths["v48"], wantV83Len-6)
	}
	if lengths["v61"] != wantV83Len-6 {
		t.Errorf("v61 length = %d, want %d (6 bytes shorter than v83: no remainLife, no trailing attribute)", lengths["v61"], wantV83Len-6)
	}
	if lengths["v72"] != wantV83Len-2 {
		t.Errorf("v72 length = %d, want %d (2 bytes shorter than v83: remainLife present, no trailing attribute)", lengths["v72"], wantV83Len-2)
	}
	for _, name := range []string{"v79", "v84", "v87", "v95", "jms"} {
		if lengths[name] != wantV83Len {
			t.Errorf("%s length = %d, want %d (unchanged from today)", name, lengths[name], wantV83Len)
		}
	}
}

// Ensure Asset satisfies the Encode signature pattern used by writers.
func TestAssetEncodeSignature(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	a := NewAsset(false, 1, 2000000, time.Time{}).SetStackableInfo(1, 0, 0)
	ctx := test.CreateContext("GMS", 83, 1)

	// Verify the Encode method returns a function matching the writer pattern.
	var encodeFn func(map[string]interface{}) []byte = a.Encode(l, ctx)
	w := response.NewWriter(l)
	w.WriteByteArray(encodeFn(nil))
	if len(w.Bytes()) == 0 {
		t.Error("expected non-empty bytes from writer")
	}
}
