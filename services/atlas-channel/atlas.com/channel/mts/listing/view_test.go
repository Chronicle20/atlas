package listing_test

import (
	mtslisting "atlas-channel/mts/listing"
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestToMtsItemEquipCarriesStatsAndOwner pins the fix for the in-game MTS browse
// view dropping an equip's real stats + item-tag owner. atlas-mts persists the
// scrolled stats and the owner on the listing (verified live: a listed +2 weapon-
// attack cape stores weaponAttack=2, owner="Atlas"); the browse ITCITEM must
// render them, not a bare base-template blob. The equip blob leads with the
// GW_ItemSlotBase equip type byte 0x01 (not the stackable 0x02 that the old bare
// blob emitted), round-trips the stat block, and carries the owner as sOwner.
func TestToMtsItemEquipCarriesStatsAndOwner(t *testing.T) {
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	// Pink Adventurer Cape (equip 1102041): +2 weapon attack, 5 upgrade slots, tag.
	m, err := mtslisting.Extract(mtslisting.RestModel{
		ItcSn: 1, TemplateId: 1102041, Quantity: 1,
		WeaponAttack: 2, Slots: 5, Owner: "Atlas", ListValue: 1000,
	})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	asset := mtslisting.ToMtsItem(m).Item()
	encoded := asset.Encode(logrus.New(), ctx)(map[string]interface{}{})
	if len(encoded) == 0 {
		t.Fatal("empty ITCITEM blob")
	}

	// Leads with the equip type byte — not the old bare stackable 0x02 blob that
	// dropped every equip stat.
	if encoded[0] != 0x01 {
		t.Fatalf("equip ITCITEM leads with 0x%02x, want equip type 0x01", encoded[0])
	}

	// The scrolled stat block round-trips (weapon attack + upgrade slots).
	var out model.Asset
	req := request.Request(encoded)
	reader := request.NewRequestReader(&req, 0)
	out.Decode(logrus.New(), ctx)(&reader, nil)
	if out.WeaponAttack() != 2 {
		t.Errorf("decoded weaponAttack = %d, want 2", out.WeaponAttack())
	}
	if out.Slots() != 5 {
		t.Errorf("decoded slots = %d, want 5", out.Slots())
	}

	// The item-tag owner (sOwner) is on the wire. The equip decoder intentionally
	// discards the name, so assert on the encoded bytes.
	if !bytes.Contains(encoded, []byte("Atlas")) {
		t.Errorf("owner tag %q not encoded in equip ITCITEM blob", "Atlas")
	}
}

// TestToMtsItemNonEquipStaysBareStackable guards the regression boundary: a
// non-equip listing must still emit the bare stackable blob (leading type byte
// 0x02, no leading inventory-slot byte) that the v83 client's per-item decoder
// requires. The owner still threads through for a tagged stackable.
func TestToMtsItemNonEquipStaysBareStackable(t *testing.T) {
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	// Nautilus return scroll (USE / stackable).
	m, err := mtslisting.Extract(mtslisting.RestModel{
		ItcSn: 2, TemplateId: 2030019, Quantity: 3, ListValue: 200,
	})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	asset := mtslisting.ToMtsItem(m).Item()
	encoded := asset.Encode(logrus.New(), ctx)(map[string]interface{}{})
	if len(encoded) == 0 {
		t.Fatal("empty ITCITEM blob")
	}
	if encoded[0] != 0x02 {
		t.Fatalf("stackable ITCITEM leads with 0x%02x, want bare stackable type 0x02", encoded[0])
	}
}
