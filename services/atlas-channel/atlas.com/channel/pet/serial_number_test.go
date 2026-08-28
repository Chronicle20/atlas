package pet_test

import (
	"atlas-channel/pet"
	"errors"
	"testing"
)

func mustPet(t *testing.T, id uint32, cashId uint64) pet.Model {
	t.Helper()
	m, err := pet.NewBuilder(id, cashId, 5000012, "Mr. Roboto").SetOwnerID(1).Build()
	if err != nil {
		t.Fatalf("Failed to build pet [%d]: %v", id, err)
	}
	return m
}

// SerialNumber is the identifier the CLIENT holds for a pet
// (GW_ItemSlotBase::liCashItemSN). A cash-purchased pet uses the cash serial it
// shares with its cash-shop asset, so that
// CCashShop::OnCashItemResMoveLtoSDone (GMS v83 @0x47aee2) can match the locker
// entry on withdraw. A pet that never came from the cash shop has no serial and
// no locker entry, so it falls back to its Atlas pet id.
func TestModelSerialNumber(t *testing.T) {
	const serial = uint64(8688106441904477350)

	if got := mustPet(t, 1, serial).SerialNumber(); got != serial {
		t.Fatalf("SerialNumber() = %d, want the cash serial %d", got, serial)
	}
	if got := mustPet(t, 7, 0).SerialNumber(); got != 7 {
		t.Fatalf("SerialNumber() = %d, want the pet id 7", got)
	}
}

func TestSelectBySerialNumber(t *testing.T) {
	const serial = uint64(8688106441904477350)
	purchased := mustPet(t, 1, serial)
	granted := mustPet(t, 7, 0)
	pets := []pet.Model{purchased, granted}

	t.Run("resolves a cash serial", func(t *testing.T) {
		got, err := pet.SelectBySerialNumber(pets, serial)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Id() != purchased.Id() {
			t.Fatalf("resolved pet %d, want %d", got.Id(), purchased.Id())
		}
	})

	t.Run("resolves a serial-less pet by its id", func(t *testing.T) {
		got, err := pet.SelectBySerialNumber(pets, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Id() != granted.Id() {
			t.Fatalf("resolved pet %d, want %d", got.Id(), granted.Id())
		}
	})

	// A pet that HAS a cash serial must not also be reachable by its Atlas id:
	// the client never sends that value for such a pet, so accepting it would
	// only widen what a crafted packet can address.
	t.Run("does not resolve a purchased pet by its id", func(t *testing.T) {
		if _, err := pet.SelectBySerialNumber(pets, 1); !errors.Is(err, pet.ErrPetNotFound) {
			t.Fatalf("err = %v, want ErrPetNotFound", err)
		}
	})

	t.Run("rejects an unknown serial", func(t *testing.T) {
		if _, err := pet.SelectBySerialNumber(pets, 999); !errors.Is(err, pet.ErrPetNotFound) {
			t.Fatalf("err = %v, want ErrPetNotFound", err)
		}
	})

	// A zero serial is the gms_48 "no petId on the wire" case reaching a handler
	// that does expect one; it must never resolve to an arbitrary pet.
	t.Run("rejects a zero serial", func(t *testing.T) {
		if _, err := pet.SelectBySerialNumber(pets, 0); !errors.Is(err, pet.ErrPetNotFound) {
			t.Fatalf("err = %v, want ErrPetNotFound", err)
		}
	})
}
