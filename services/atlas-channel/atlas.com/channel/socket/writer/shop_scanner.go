package writer

import (
	"atlas-channel/merchant"

	merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	pktmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// ShopScannerResult CWvsContext::OnShopScannerResult
// ShopLinkResult CWvsContext::OnShopLinkResult
// Mode bytes and link codes are config-resolved from the tenant template's
// options.operations tables (never hard-coded), matching world_message.go.

func ShopScannerResultBody(itemId uint32, records []merchantcb.ShopScannerRecord) packet.Encode {
	return merchantpkt.ShopScannerResultBody(itemId, records)
}

func ShopScannerHotListBody(itemIds []uint32) packet.Encode {
	return merchantpkt.ShopScannerHotListBody(itemIds)
}

func ShopLinkResultBody(code merchantpkt.ShopLinkResultCode) packet.Encode {
	return merchantpkt.ShopLinkResultBody(code)
}

// ShopScannerRecords converts merchant search listings plus resolved owner
// names into wire records: the channel is encoded verbatim (same base as
// CStage::OnSetField stores in m_nChannelID), because
// CUIShopScanResult::LoadCurPageItemList enables a result row's enter button
// only when record.nChannelID == m_nChannelID — a -1 offset made every
// same-channel store render as "closed" (task-127). Equip rows (itemType 1)
// get a slotless GW_ItemSlotBase built from the listing's point-in-sale
// snapshot, and a missing owner name degrades to empty string rather than
// failing the search.
func ShopScannerRecords(listings []merchant.SearchListing, names map[uint32]string) []merchantcb.ShopScannerRecord {
	records := make([]merchantcb.ShopScannerRecord, 0, len(listings))
	for _, sl := range listings {
		var assetPtr *pktmodel.Asset
		if sl.ItemType() == 1 {
			snap := sl.ItemSnapshot()
			asset := pktmodel.NewAsset(true, 0, sl.ItemId(), snap.Expiration).
				SetEquipmentStats(snap.Strength, snap.Dexterity, snap.Intelligence, snap.Luck,
					snap.Hp, snap.Mp, snap.WeaponAttack, snap.MagicAttack, snap.WeaponDefense,
					snap.MagicDefense, snap.Accuracy, snap.Avoidability, snap.Hands, snap.Speed, snap.Jump).
				SetEquipmentMeta(snap.Slots, snap.LevelType, snap.Level, snap.Experience, snap.HammersApplied, snap.Flag)
			assetPtr = &asset
		}
		records = append(records, merchantcb.NewShopScannerRecord(
			names[sl.OwnerId()],
			sl.MapId(),
			sl.Title(),
			uint32(sl.BundlesRemaining()),
			uint32(sl.BundleSize()),
			sl.PricePerBundle(),
			sl.OwnerId(),
			byte(sl.ChannelId()),
			sl.ItemType(),
			assetPtr,
		))
	}
	return records
}
