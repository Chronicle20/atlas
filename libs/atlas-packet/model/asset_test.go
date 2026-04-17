package model

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
