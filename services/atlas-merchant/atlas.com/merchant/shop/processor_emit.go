package shop

import (
	asset2 "atlas-merchant/kafka/message/asset"
	character "atlas-merchant/kafka/message/character"
	"atlas-merchant/kafka/message/compartment"
	message "atlas-merchant/kafka/message"
	merchant "atlas-merchant/kafka/message/merchant"
	"atlas-merchant/listing"
	"encoding/json"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

func (p *ProcessorImpl) OpenShopAndEmit(shopId uuid.UUID, characterId uint32) error {
	if err := p.OpenShop(shopId); err != nil {
		return err
	}

	m, err := p.GetById(shopId)
	if err != nil {
		return err
	}

	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return buf.Put(merchant.EnvStatusEventTopic, StatusEventShopOpenedProvider(characterId, shopId, m))
	})
}

func (p *ProcessorImpl) CloseShopAndEmit(shopId uuid.UUID, characterId uint32, reason CloseReason) error {
	m, err := p.GetById(shopId)
	if err != nil {
		return err
	}

	var listings []ListingSnapshot
	if m.ShopType() == CharacterShop && reason != CloseReasonDisconnect {
		ls, _ := p.GetListings(shopId)
		for _, l := range ls {
			listings = append(listings, ListingSnapshot{
				ItemId:       l.ItemId(),
				ItemType:     l.ItemType(),
				Quantity:     l.Quantity(),
				ItemSnapshot: l.ItemSnapshot(),
			})
		}
	}

	if err := p.CloseShop(shopId, reason); err != nil {
		return err
	}

	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		for _, ls := range listings {
			acceptItemToBuffer(buf, characterId, ls)
		}
		return buf.Put(merchant.EnvStatusEventTopic, StatusEventShopClosedProvider(characterId, shopId, reason))
	})
}

func (p *ProcessorImpl) EnterMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error {
	if err := p.EnterMaintenance(shopId); err != nil {
		return err
	}

	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return buf.Put(merchant.EnvStatusEventTopic, StatusEventMaintenanceEnteredProvider(characterId, shopId))
	})
}

func (p *ProcessorImpl) ExitMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error {
	closed, err := p.ExitMaintenance(shopId)
	if err != nil {
		return err
	}

	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		if closed {
			return buf.Put(merchant.EnvStatusEventTopic, StatusEventShopClosedProvider(characterId, shopId, CloseReasonEmpty))
		}
		return buf.Put(merchant.EnvStatusEventTopic, StatusEventMaintenanceExitedProvider(characterId, shopId))
	})
}

func (p *ProcessorImpl) EnterShopAndEmit(characterId uint32, shopId uuid.UUID) error {
	err := p.EnterShop(characterId, shopId)
	if err != nil {
		if IsShopFull(err) {
			return message.Emit(p.producer)(func(buf *message.Buffer) error {
				return buf.Put(merchant.EnvStatusEventTopic, StatusEventCapacityFullProvider(characterId, shopId))
			})
		}
		return err
	}

	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return buf.Put(merchant.EnvStatusEventTopic, StatusEventVisitorEnteredProvider(characterId, shopId))
	})
}

func (p *ProcessorImpl) ExitShopAndEmit(characterId uint32, shopId uuid.UUID) error {
	if err := p.ExitShop(characterId, shopId); err != nil {
		return err
	}

	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return buf.Put(merchant.EnvStatusEventTopic, StatusEventVisitorExitedProvider(characterId, shopId))
	})
}

func (p *ProcessorImpl) AddListingAndEmit(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot json.RawMessage, flag uint16, inventoryType byte, assetId uint32) (listing.Model, error) {
	created, err := p.AddListing(shopId, itemId, itemType, bundleSize, bundleCount, pricePerBundle, itemSnapshot, flag)
	if err != nil {
		return listing.Model{}, err
	}

	quantity := uint32(bundleSize) * uint32(bundleCount)
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		transactionId := uuid.New()
		return buf.Put(compartment.EnvCommandTopic, ReleaseAssetCommandProvider(transactionId, characterId, inventoryType, assetId, quantity))
	})
	if err != nil {
		return created, err
	}

	return created, nil
}

func (p *ProcessorImpl) PurchaseBundleAndEmit(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (PurchaseResult, error) {
	result, err := p.PurchaseBundle(buyerCharacterId, shopId, listingIndex, bundleCount)
	if err != nil {
		return PurchaseResult{}, err
	}

	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		// Deduct mesos from buyer.
		transactionId := uuid.New()
		if err := buf.Put(character.EnvCommandTopic, ChangeMesoCommandProvider(transactionId, worldId, buyerCharacterId, result.ShopOwnerId, "MERCHANT", -int32(result.TotalCost))); err != nil {
			return err
		}

		// Grant items to buyer.
		if result.ItemSnapshot != nil {
			var ad asset2.AssetData
			if err := json.Unmarshal(result.ItemSnapshot, &ad); err == nil {
				ad.Quantity = uint32(result.BundleSize) * uint32(result.BundlesPurchased)
				invType, ok := inventory.TypeFromItemId(item.Id(result.ItemId))
				if ok {
					itemTransactionId := uuid.New()
					if err := buf.Put(compartment.EnvCommandTopic, AcceptAssetCommandProvider(itemTransactionId, buyerCharacterId, byte(invType), result.ItemId, ad)); err != nil {
						return err
					}
				}
			}
		}

		// Credit mesos to owner (character shops only; hired merchants accumulate in DB).
		if result.ShopType == CharacterShop && result.NetAmount > 0 {
			creditTransactionId := uuid.New()
			if err := buf.Put(character.EnvCommandTopic, ChangeMesoCommandProvider(creditTransactionId, worldId, result.ShopOwnerId, buyerCharacterId, "MERCHANT", int32(result.NetAmount))); err != nil {
				return err
			}
		}

		if err := buf.Put(merchant.EnvListingEventTopic, ListingEventPurchasedProvider(shopId, listingIndex, buyerCharacterId, bundleCount, result.BundlesRemaining)); err != nil {
			return err
		}

		if result.ShopClosed {
			if err := buf.Put(merchant.EnvStatusEventTopic, StatusEventShopClosedProvider(result.ShopOwnerId, shopId, CloseReasonSoldOut)); err != nil {
				return err
			}
		}

		return nil
	})

	return result, err
}

// ListingSnapshot captures listing data before shop closure.
type ListingSnapshot struct {
	ItemId       uint32
	ItemType     byte
	Quantity     uint16
	ItemSnapshot json.RawMessage
}

func acceptItemToBuffer(buf *message.Buffer, characterId uint32, ls ListingSnapshot) {
	if ls.ItemSnapshot == nil {
		return
	}

	var ad asset2.AssetData
	if err := json.Unmarshal(ls.ItemSnapshot, &ad); err != nil {
		return
	}

	ad.Quantity = uint32(ls.Quantity)

	invType, ok := inventory.TypeFromItemId(item.Id(ls.ItemId))
	if !ok {
		return
	}

	transactionId := uuid.New()
	_ = buf.Put(compartment.EnvCommandTopic, AcceptAssetCommandProvider(transactionId, characterId, byte(invType), ls.ItemId, ad))
}

// IsShopFull checks if the error is a shop capacity error.
func IsShopFull(err error) bool {
	return errors.Is(err, ErrShopFull)
}
