package writer

import (
	"testing"

	"atlas-channel/merchant"

	"github.com/stretchr/testify/require"
)

func TestShopScannerRecords_Conversion(t *testing.T) {
	listings := []merchant.SearchListing{
		merchant.NewSearchListing(merchant.SearchListingSeed{
			Title: "cheap stuff", WorldId: 0, ChannelId: 3, MapId: 910000004,
			OwnerId: 30001, ShopType: 1, State: merchant.StateOpen,
			ItemId: 2060000, ItemType: 2, BundleSize: 100, BundlesRemaining: 3,
			PricePerBundle: 5000,
		}),
	}
	records := ShopScannerRecords(listings, map[uint32]string{30001: "OwnerA"})
	require.Len(t, records, 1)
	r := records[0]
	require.Equal(t, "OwnerA", r.OwnerName())
	require.Equal(t, uint32(910000004), r.MapId())
	require.Equal(t, "cheap stuff", r.Title())
	require.Equal(t, uint32(3), r.Bundles())      // nNumber = bundles available
	require.Equal(t, uint32(100), r.BundleSize()) // nSet = quantity per bundle
	require.Equal(t, uint32(5000), r.Price())
	require.Equal(t, uint32(30001), r.OwnerId()) // dwMiniRoomSN echo
	// nChannelID must match the base SetField sends (CStage::OnSetField stores
	// m_nChannelID = Decode4 verbatim). CUIShopScanResult::LoadCurPageItemList
	// enables a result row's enter button only when record.nChannelID ==
	// m_nChannelID, so a same-channel store on channel 3 must encode 3 — a -1
	// offset (copied from the channel-select screen) rendered every store as
	// "closed" (task-127).
	require.Equal(t, byte(3), r.ChannelId())
	require.Equal(t, byte(2), r.InventoryType())
	require.Nil(t, r.Asset())
}

func TestShopScannerRecords_EquipRowGetsAsset(t *testing.T) {
	listings := []merchant.SearchListing{
		merchant.NewSearchListing(merchant.SearchListingSeed{
			Title: "swords", WorldId: 0, ChannelId: 1, MapId: 910000004,
			OwnerId: 30002, ShopType: 2, State: merchant.StateOpen,
			ItemId: 1302000, ItemType: 1, BundleSize: 1, BundlesRemaining: 1,
			PricePerBundle: 150000,
			Snapshot: merchant.SnapshotRestModel{
				Strength: 5, Dexterity: 3, WeaponAttack: 17, Slots: 7,
			},
		}),
	}
	records := ShopScannerRecords(listings, map[uint32]string{})
	require.Len(t, records, 1)
	require.Equal(t, byte(1), records[0].InventoryType())
	require.NotNil(t, records[0].Asset())
	require.Equal(t, "", records[0].OwnerName()) // missing name -> empty string
}
