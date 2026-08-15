package shop

import (
	"testing"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"

	asset2 "atlas-merchant/kafka/message/asset"
)

func TestIsListableItemAcceptsKarmaMarkedEquip(t *testing.T) {
	flags := uint16(af.FlagUntradeable) | uint16(af.FlagKarmaEquip)
	if err := IsListableItem(1002357, flags); err != nil {
		t.Fatalf("IsListableItem refused a karma-marked untradeable equip: %v", err)
	}
}

func TestIsListableItemAcceptsKarmaMarkedBundle(t *testing.T) {
	flags := uint16(af.FlagUntradeable) | uint16(af.FlagKarmaUse)
	if err := IsListableItem(2280000, flags); err != nil {
		t.Fatalf("IsListableItem refused a karma-marked untradeable bundle: %v", err)
	}
}

// A SPIKED untradeable equip carries 0x02, the BUNDLE karma bit, and must still
// be refused.
func TestIsListableItemStillRefusesSpikedUntradeableEquip(t *testing.T) {
	flags := uint16(af.FlagUntradeable) | uint16(af.FlagSpikes)
	if err := IsListableItem(1002357, flags); err != ErrUntradeableItem {
		t.Fatalf("IsListableItem = %v, want ErrUntradeableItem", err)
	}
}

func TestIsListableItemStillRefusesUnmarkedUntradeable(t *testing.T) {
	if err := IsListableItem(1002357, uint16(af.FlagUntradeable)); err != ErrUntradeableItem {
		t.Fatalf("IsListableItem = %v, want ErrUntradeableItem", err)
	}
}

// ErrPetItem and ErrCashItem are untouched by the karma override.
func TestIsListableItemStillRefusesPetsAndCash(t *testing.T) {
	if err := IsListableItem(5000000, uint16(af.FlagKarmaUse)); err != ErrPetItem {
		t.Fatalf("IsListableItem(pet) = %v, want ErrPetItem", err)
	}
	if err := IsListableItem(5520000, uint16(af.FlagKarmaUse)); err != ErrCashItem {
		t.Fatalf("IsListableItem(cash) = %v, want ErrCashItem", err)
	}
}

func TestClearKarmaFromAssetData(t *testing.T) {
	// equip: the bit is cleared, FlagUntradeable survives
	equipFlag := uint16(af.FlagUntradeable) | uint16(af.FlagKarmaEquip)
	out := clearKarmaFromAssetData(1002357, asset2.AssetData{Flag: equipFlag})
	if af.HasFlag(out.Flag, af.FlagKarmaEquip) {
		t.Fatal("the equip karma bit survived the sale")
	}
	if !af.HasFlag(out.Flag, af.FlagUntradeable) {
		t.Fatal("clearing karma disturbed FlagUntradeable; the item must arrive UNTRADEABLE")
	}

	// bundle: the bit is cleared
	bundleFlag := uint16(af.FlagUntradeable) | uint16(af.FlagKarmaUse)
	out = clearKarmaFromAssetData(2280000, asset2.AssetData{Flag: bundleFlag})
	if af.HasFlag(out.Flag, af.FlagKarmaUse) {
		t.Fatal("the bundle karma bit survived the sale")
	}

	// spiked equip: FlagSpikes survives (0x02 on an equip is FlagSpikes, not
	// the bundle karma bit)
	spikedFlag := uint16(af.FlagSpikes)
	out = clearKarmaFromAssetData(1002357, asset2.AssetData{Flag: spikedFlag})
	if !af.HasFlag(out.Flag, af.FlagSpikes) {
		t.Fatal("a spiked equip lost its spikes at sale time")
	}

	// pet: the flag is untouched (KarmaFlagFor reports no bit for a pet, and
	// the pet bit 0x01 is FlagLock everywhere else)
	petFlag := uint16(af.FlagLock)
	out = clearKarmaFromAssetData(5000000, asset2.AssetData{Flag: petFlag})
	if out.Flag != petFlag {
		t.Fatalf("a pet's flag changed at sale time: %#x -> %#x", petFlag, out.Flag)
	}
}
