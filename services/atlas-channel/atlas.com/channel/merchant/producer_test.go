package merchant

import (
	"encoding/json"
	"testing"
	"time"

	"atlas-channel/asset"
	merchant2 "atlas-channel/kafka/message/merchant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestAddListingCommandProvider_PopulatesItemFields proves the fix for the
// owl-search prerequisite gap: an in-game AddListing command must carry the
// resolved asset's ItemId, ItemType, AssetId and a full ItemSnapshot rather
// than zero values, since atlas-merchant persists these fields verbatim
// (kafka/consumer/merchant/consumer.go handleAddListingCommand) and the owl
// search path is keyed on ItemId.
func TestAddListingCommandProvider_PopulatesItemFields(t *testing.T) {
	shopId := uuid.New()
	compartmentId := uuid.New()
	expiration := time.Now().Add(24 * time.Hour).Truncate(time.Second).UTC()

	// An equip-type asset (helmet, item id 1002140) resolved from the
	// character's equip compartment slot.
	a := asset.NewModelBuilder(777, compartmentId, 1002140).
		SetSlot(3).
		SetExpiration(expiration).
		SetStrength(5).
		SetDexterity(3).
		SetIntelligence(1).
		SetLuck(2).
		SetHp(10).
		SetMp(10).
		SetWeaponAttack(4).
		SetSlots(7).
		SetLevel(0).
		MustBuild()

	provider := AddListingCommandProvider(uint32(42), shopId, byte(1), int16(3), uint16(1), uint16(1), uint32(5000), a)

	msgs, err := provider()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var cmd merchant2.Command[merchant2.CommandAddListingBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))

	require.Equal(t, merchant2.CommandAddListing, cmd.Type)
	require.Equal(t, uint32(42), cmd.CharacterId)
	require.Equal(t, shopId.String(), cmd.Body.ShopId)

	// The three fields that were previously left zeroed, breaking owl search.
	require.Equal(t, uint32(1002140), cmd.Body.ItemId, "ItemId must be the asset's template id")
	require.Equal(t, byte(1), cmd.Body.ItemType, "ItemType must be 1 for an equip-compartment listing")
	require.Equal(t, uint32(777), cmd.Body.AssetId, "AssetId must be the resolved asset's row id, consumed by the merchant's ReleaseAsset hop")

	require.Equal(t, expiration, cmd.Body.ItemSnapshot.Expiration)
	require.Equal(t, uint16(5), cmd.Body.ItemSnapshot.Strength)
	require.Equal(t, uint16(3), cmd.Body.ItemSnapshot.Dexterity)
	require.Equal(t, uint16(1), cmd.Body.ItemSnapshot.Intelligence)
	require.Equal(t, uint16(2), cmd.Body.ItemSnapshot.Luck)
	require.Equal(t, uint16(10), cmd.Body.ItemSnapshot.Hp)
	require.Equal(t, uint16(10), cmd.Body.ItemSnapshot.Mp)
	require.Equal(t, uint16(4), cmd.Body.ItemSnapshot.WeaponAttack)
	require.Equal(t, uint16(7), cmd.Body.ItemSnapshot.Slots)
}

// TestAddListingCommandProvider_StackableItemType proves a non-equip
// (stackable) listing maps to ItemType 2, matching the ShopScannerRecords
// convention (socket/writer/shop_scanner.go treats ItemType==1 as equip and
// everything else as a plain stackable render).
func TestAddListingCommandProvider_StackableItemType(t *testing.T) {
	shopId := uuid.New()
	compartmentId := uuid.New()

	a := asset.NewModelBuilder(900, compartmentId, 2000000).
		SetSlot(1).
		SetQuantity(50).
		MustBuild()

	provider := AddListingCommandProvider(uint32(1), shopId, byte(2), int16(1), uint16(50), uint16(1), uint32(100), a)

	msgs, err := provider()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var cmd merchant2.Command[merchant2.CommandAddListingBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))

	require.Equal(t, byte(2), cmd.Body.ItemType, "ItemType must be 2 for a non-equip listing")
	require.Equal(t, uint32(2000000), cmd.Body.ItemId)
	require.Equal(t, uint32(900), cmd.Body.AssetId)
	require.Equal(t, uint32(50), cmd.Body.ItemSnapshot.Quantity)
}
