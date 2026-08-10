// Package trade dispatches the atomic trade-escrow custody commands to
// atlas-trades on COMMAND_TOPIC_TRADE_CUSTODY (task-205 design §5A.2).
//
// Every command is keyed by the escrow row's uuid rather than by the owner, so
// all commands touching one escrow row — accept, release, and their late
// compensating inverses — land on the same partition and cannot be reordered
// relative to each other. A restore that overtook its release would resurrect a
// row the settlement had already consumed.
package trade

import (
	tradeCustody "atlas-saga-orchestrator/kafka/message/trade/custody"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// escrowKey derives the partition key from the escrow row id.
func escrowKey(escrowId uuid.UUID) []byte {
	return producer.CreateKey(int(escrowId.ID()))
}

// AcceptToTradeProvider creates an ACCEPT_TO_TRADE command for the atlas-trades
// custody consumer.
func AcceptToTradeProvider(transactionId uuid.UUID, params AcceptToTradeParams) model.Provider[[]kafka.Message] {
	value := &tradeCustody.Command[tradeCustody.AcceptToTradeCommandBody]{
		TransactionId: transactionId,
		Type:          tradeCustody.CommandAcceptToTrade,
		Body: tradeCustody.AcceptToTradeCommandBody{
			EscrowId:            params.EscrowId,
			RoomId:              params.RoomId,
			OwnerId:             params.OwnerId,
			TradeSlot:           params.TradeSlot,
			SourceInventoryType: params.SourceInventoryType,
			SourceSlot:          params.SourceSlot,
			AssetId:             params.AssetId,
			TemplateId:          params.TemplateId,
			Quantity:            params.Quantity,
			Strength:            params.Strength,
			Dexterity:           params.Dexterity,
			Intelligence:        params.Intelligence,
			Luck:                params.Luck,
			HP:                  params.HP,
			MP:                  params.MP,
			WeaponAttack:        params.WeaponAttack,
			MagicAttack:         params.MagicAttack,
			WeaponDefense:       params.WeaponDefense,
			MagicDefense:        params.MagicDefense,
			Accuracy:            params.Accuracy,
			Avoidability:        params.Avoidability,
			Hands:               params.Hands,
			Speed:               params.Speed,
			Jump:                params.Jump,
			Slots:               params.Slots,
			Level:               params.Level,
			ItemLevel:           params.ItemLevel,
			ItemExp:             params.ItemExp,
			RingId:              params.RingId,
			ViciousCount:        params.ViciousCount,
			Flags:               params.Flags,
			Owner:               params.Owner,
		},
	}
	return producer.SingleMessageProvider(escrowKey(params.EscrowId), value)
}

// ReleaseFromTradeProvider creates a RELEASE_FROM_TRADE command.
func ReleaseFromTradeProvider(transactionId uuid.UUID, escrowId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &tradeCustody.Command[tradeCustody.ReleaseFromTradeCommandBody]{
		TransactionId: transactionId,
		Type:          tradeCustody.CommandReleaseFromTrade,
		Body:          tradeCustody.ReleaseFromTradeCommandBody{EscrowId: escrowId},
	}
	return producer.SingleMessageProvider(escrowKey(escrowId), value)
}

// RestoreTradeEscrowProvider creates a RESTORE_TRADE_ESCROW command — the
// compensating inverse of a release.
func RestoreTradeEscrowProvider(transactionId uuid.UUID, escrowId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &tradeCustody.Command[tradeCustody.RestoreTradeEscrowCommandBody]{
		TransactionId: transactionId,
		Type:          tradeCustody.CommandRestoreTradeEscrow,
		Body:          tradeCustody.RestoreTradeEscrowCommandBody{EscrowId: escrowId},
	}
	return producer.SingleMessageProvider(escrowKey(escrowId), value)
}

// RemoveTradeEscrowProvider creates a REMOVE_TRADE_ESCROW command — the
// compensating inverse of an accept.
func RemoveTradeEscrowProvider(transactionId uuid.UUID, escrowId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &tradeCustody.Command[tradeCustody.RemoveTradeEscrowCommandBody]{
		TransactionId: transactionId,
		Type:          tradeCustody.CommandRemoveTradeEscrow,
		Body:          tradeCustody.RemoveTradeEscrowCommandBody{EscrowId: escrowId},
	}
	return producer.SingleMessageProvider(escrowKey(escrowId), value)
}
