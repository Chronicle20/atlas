package pending_change

import (
	pendingchange2 "atlas-character/kafka/message/pending_change"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

func createdEventProvider(m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.CharacterId()))
	value := &pendingchange2.StatusEvent[pendingchange2.CreatedEventBody]{
		TransactionId: m.TransactionId(),
		CharacterId:   m.CharacterId(),
		WorldId:       m.SourceWorldId(),
		Type:          pendingchange2.EventTypeCreated,
		Body: pendingchange2.CreatedEventBody{
			PendingChangeId:    m.Id(),
			ChangeType:         m.Type(),
			RequestedName:      m.RequestedName(),
			DestinationWorldId: m.DestinationWorldId(),
			ExpiresAt:          m.ExpiresAt(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// resolvedEventProvider routes the notification on the record's source world.
// That is correct at original emission time: a transfer only ever resolves at
// LOGOUT, when the character has not moved yet, and every other resolution
// leaves the character where it was.
func resolvedEventProvider(m Model) model.Provider[[]kafka.Message] {
	return resolvedEventProviderForWorld(m, m.SourceWorldId())
}

// resolvedEventProviderForWorld routes the notification on an explicitly chosen
// world. The LOGIN catch-up (RenotifyForCharacter) uses it with the character's
// CURRENT world: for an APPLIED world transfer the character has already moved
// to the destination by the time they log back in, so re-emitting on
// SourceWorldId would announce a successful transfer to the world the player
// just left — i.e. never announce it at all.
func resolvedEventProviderForWorld(m Model, worldId world.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.CharacterId()))
	value := &pendingchange2.StatusEvent[pendingchange2.ResolvedEventBody]{
		TransactionId: m.TransactionId(),
		CharacterId:   m.CharacterId(),
		WorldId:       worldId,
		Type:          pendingchange2.EventTypeResolved,
		Body: pendingchange2.ResolvedEventBody{
			PendingChangeId:    m.Id(),
			ChangeType:         m.Type(),
			Status:             m.Status(),
			Reason:             m.Reason(),
			RequestedName:      m.RequestedName(),
			DestinationWorldId: m.DestinationWorldId(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// sagaTransactionId derives a stable, purpose-scoped transaction id from the
// pending-change id. The consumption saga and the refund saga must NOT share a
// transaction id: atlas-saga-orchestrator stores sagas keyed by transaction id
// (saga.ProcessorImpl.Put), so reusing the record's own id would make the
// refund overwrite the completed consumption. Deriving rather than minting also
// keeps each id reproducible from the record alone, which is what makes an
// operator able to correlate the two commands after the fact.
func sagaTransactionId(m Model, purpose string) uuid.UUID {
	return uuid.NewSHA1(m.Id(), []byte(purpose))
}

const (
	sagaPurposeDestroyAsset = "pending_change:destroy_asset"
	sagaPurposeAwardAsset   = "pending_change:award_asset"
)

// destroyAssetCommandProvider consumes the coupon at request acceptance
// (FR-2.8). Only the item path has an asset; the purchase path's entitlement is
// consumed by atlas-cashshop off PENDING_CHANGE_CREATED.
func destroyAssetCommandProvider(m Model) model.Provider[[]kafka.Message] {
	s := sharedsaga.NewBuilder().
		SetTransactionId(sagaTransactionId(m, sagaPurposeDestroyAsset)).
		SetSagaType(sharedsaga.CashShopOperation).
		SetInitiatedBy(sagaInitiator).
		AddStep("consume_pending_change_coupon", sharedsaga.Pending, sharedsaga.DestroyAsset, sharedsaga.DestroyAssetPayload{
			CharacterId: m.CharacterId(),
			TemplateId:  m.AssetId(),
			Quantity:    1,
		}).
		Build()
	return sagaCommandProvider(s)
}

// awardAssetCommandProvider refunds the coupon on every non-APPLIED exit
// (FR-2.8). It is reachable from exactly one place — the Resolve branch guarded
// by transition's moved == true — which is what makes a redelivered cancel mint
// nothing (design §3.10).
func awardAssetCommandProvider(m Model) model.Provider[[]kafka.Message] {
	s := sharedsaga.NewBuilder().
		SetTransactionId(sagaTransactionId(m, sagaPurposeAwardAsset)).
		SetSagaType(sharedsaga.CashShopOperation).
		SetInitiatedBy(sagaInitiator).
		AddStep("refund_pending_change_coupon", sharedsaga.Pending, sharedsaga.AwardAsset, sharedsaga.AwardItemActionPayload{
			CharacterId: m.CharacterId(),
			Item: sharedsaga.ItemPayload{
				TemplateId: m.AssetId(),
				Quantity:   1,
			},
		}).
		Build()
	return sagaCommandProvider(s)
}

const sagaInitiator = "atlas-character/pending_change"

func sagaCommandProvider(s sharedsaga.Saga) model.Provider[[]kafka.Message] {
	return producer.SingleMessageProvider([]byte(s.TransactionId.String()), &s)
}
