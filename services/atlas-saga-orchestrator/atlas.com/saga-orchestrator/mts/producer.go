package mts

import (
	mtsCustody "atlas-saga-orchestrator/kafka/message/mts/custody"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// AcceptToMtsListingProvider creates an ACCEPT_TO_MTS_LISTING command for the
// atlas-mts custody consumer. Keyed by SellerId so all custody commands for a
// seller's listing are ordered.
func AcceptToMtsListingProvider(transactionId uuid.UUID, params AcceptToMtsListingParams) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(params.SellerId))
	value := &mtsCustody.Command[mtsCustody.AcceptToMtsListingCommandBody]{
		TransactionId: transactionId,
		Type:          mtsCustody.CommandAcceptToMtsListing,
		Body: mtsCustody.AcceptToMtsListingCommandBody{
			ListingId:       params.ListingId,
			WorldId:         params.WorldId,
			SellerId:        params.SellerId,
			SellerAccountId: params.SellerAccountId,
			SellerName:      params.SellerName,
			SaleType:        params.SaleType,
			TemplateId:      params.TemplateId,
			Quantity:        params.Quantity,
			Strength:        params.Strength,
			Dexterity:       params.Dexterity,
			Intelligence:    params.Intelligence,
			Luck:            params.Luck,
			HP:              params.HP,
			MP:              params.MP,
			WeaponAttack:    params.WeaponAttack,
			MagicAttack:     params.MagicAttack,
			WeaponDefense:   params.WeaponDefense,
			MagicDefense:    params.MagicDefense,
			Accuracy:        params.Accuracy,
			Avoidability:    params.Avoidability,
			Hands:           params.Hands,
			Speed:           params.Speed,
			Jump:            params.Jump,
			Slots:           params.Slots,
			Level:           params.Level,
			ItemLevel:       params.ItemLevel,
			ItemExp:         params.ItemExp,
			RingId:          params.RingId,
			ViciousCount:    params.ViciousCount,
			Flags:           params.Flags,
			ListValue:       params.ListValue,
			BuyNowPrice:     params.BuyNowPrice,
			CommissionRate:  params.CommissionRate,
			Category:        params.Category,
			SubCategory:     params.SubCategory,
			EndsAt:          params.EndsAt,
			MinIncrement:    params.MinIncrement,
			OfferWishSerial:  params.OfferWishSerial,
			OfferWishOwnerId: params.OfferWishOwnerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ReleaseFromMtsHoldingProvider creates a RELEASE_FROM_MTS_HOLDING command.
// Keyed by the holding id so replays of the same release are ordered.
func ReleaseFromMtsHoldingProvider(transactionId uuid.UUID, holdingId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(holdingId.ID()))
	value := &mtsCustody.Command[mtsCustody.ReleaseFromMtsHoldingCommandBody]{
		TransactionId: transactionId,
		Type:          mtsCustody.CommandReleaseFromMtsHolding,
		Body: mtsCustody.ReleaseFromMtsHoldingCommandBody{
			HoldingId: holdingId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RestoreMtsHoldingProvider creates a RESTORE_MTS_HOLDING command (the
// compensating inverse of RELEASE_FROM_MTS_HOLDING). Keyed by the holding id so
// replays of the same restore are ordered.
func RestoreMtsHoldingProvider(transactionId uuid.UUID, holdingId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(holdingId.ID()))
	value := &mtsCustody.Command[mtsCustody.RestoreMtsHoldingCommandBody]{
		TransactionId: transactionId,
		Type:          mtsCustody.CommandRestoreMtsHolding,
		Body: mtsCustody.RestoreMtsHoldingCommandBody{
			HoldingId: holdingId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// MoveListingToHoldingProvider creates an MTS_MOVE_LISTING_TO_HOLDING command.
// Keyed by the buyer id so the buyer's custody moves are ordered.
func MoveListingToHoldingProvider(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32, worldId byte, resultKind string, price uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(buyerId))
	value := &mtsCustody.Command[mtsCustody.MtsMoveListingToHoldingCommandBody]{
		TransactionId: transactionId,
		Type:          mtsCustody.CommandMtsMoveListingToHolding,
		Body: mtsCustody.MtsMoveListingToHoldingCommandBody{
			ListingId:  listingId,
			BuyerId:    buyerId,
			WorldId:    worldId,
			ResultKind: resultKind,
			Price:      price,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RemoveMtsListingProvider creates a REMOVE_MTS_LISTING command (the late-comp
// inverse of ACCEPT_TO_MTS_LISTING). Keyed by the listing id so replays of the
// same removal are ordered.
func RemoveMtsListingProvider(transactionId uuid.UUID, listingId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(listingId.ID()))
	value := &mtsCustody.Command[mtsCustody.RemoveMtsListingCommandBody]{
		TransactionId: transactionId,
		Type:          mtsCustody.CommandRemoveMtsListing,
		Body: mtsCustody.RemoveMtsListingCommandBody{
			ListingId: listingId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RestoreListingFromHoldingProvider creates a RESTORE_LISTING_FROM_HOLDING
// command (the late-comp inverse of MTS_MOVE_LISTING_TO_HOLDING). Keyed by the
// buyer id, matching the forward move's key so the reverse is ordered with it.
func RestoreListingFromHoldingProvider(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(buyerId))
	value := &mtsCustody.Command[mtsCustody.RestoreListingFromHoldingCommandBody]{
		TransactionId: transactionId,
		Type:          mtsCustody.CommandRestoreListingFromHolding,
		Body: mtsCustody.RestoreListingFromHoldingCommandBody{
			ListingId: listingId,
			BuyerId:   buyerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
