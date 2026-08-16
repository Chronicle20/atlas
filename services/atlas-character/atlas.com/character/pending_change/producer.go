package pending_change

import (
	pendingchange2 "atlas-character/kafka/message/pending_change"
	"strconv"

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
	sagaPurposeDestroyAsset  = "pending_change:destroy_asset"
	sagaPurposeAwardAsset    = "pending_change:award_asset"
	sagaPurposeConsumeCoupon = "pending_change:consume_coupon"
)

// NameChangeCouponTemplateId is the cash-shop name-change coupon.
//
// Grounded, not assumed: derivation.md §3 reads CCashShop::ProcessBuy on every
// GMS version v48-v95 and finds 5400000 compared as an EXACT id (5401000 is the
// world-transfer sibling). The client's item-USE dispatcher buckets by prefix
// instead (nItemID / 1000 == 5400, mirrored at
// atlas-channel character_cash_item_use.go), but no second 5400xxx id exists in
// any GMS binary examined, so the exact id is the whole band in practice. If a
// tenant ever ships another 5400xxx coupon, this is the list to extend —
// applyNameChange emits one consumption step per entry.
var nameChangeCouponTemplateIds = []uint32{5400000}

// destroyAssetCommandProvider consumes the coupon at request acceptance
// (FR-2.8). Only the item path has an asset. The purchase path has none: its
// entitlement is the NX charge itself, taken by atlas-cashshop's normal
// Purchase flow off the REQUEST_PURCHASE command atlas-channel emits with this
// record's id as the transaction id. atlas-cashshop does not consume
// PENDING_CHANGE_CREATED — it has no reference to that event at all — so
// nothing here is keyed off it. (An earlier version of this comment claimed the
// opposite and contradicted Create's own doc a few lines up in processor.go.)
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

// consumeCouponsCommandProvider consumes EVERY name-change coupon the character
// holds once the rename actually lands.
//
// Consumption is at APPLY, not at request acceptance, because on the purchase
// path there is no coupon in the inventory when the request is made — the
// cash-shop purchase materialises it afterwards. Destroying on apply is the only
// point at which the item reliably exists.
//
// DestroyAllAssets rather than DestroyAsset: cash items do not stack (each
// instance carries its own cashId and occupies its own slot), and DestroyAsset
// resolves a template to the FIRST matching slot only — so a player holding two
// coupons would keep one.
func consumeCouponsCommandProvider(m Model, templateId uint32) model.Provider[[]kafka.Message] {
	s := sharedsaga.NewBuilder().
		SetTransactionId(sagaTransactionId(m, sagaPurposeConsumeCoupon+":"+strconv.FormatUint(uint64(templateId), 10))).
		SetSagaType(sharedsaga.CashShopOperation).
		SetInitiatedBy(sagaInitiator).
		AddStep("consume_name_change_coupons", sharedsaga.Pending, sharedsaga.DestroyAllAssets, sharedsaga.DestroyAllAssetsPayload{
			CharacterId: m.CharacterId(),
			TemplateId:  templateId,
		}).
		Build()
	return sagaCommandProvider(s)
}

const sagaInitiator = "atlas-character/pending_change"

func sagaCommandProvider(s sharedsaga.Saga) model.Provider[[]kafka.Message] {
	return producer.SingleMessageProvider([]byte(s.TransactionId.String()), &s)
}

// --- The world-transfer saga (design §3.11) --------------------------------

const sagaPurposeWorldTransfer = "pending_change:world_transfer"

// The five step ids. Fixed order, and the compensations are keyed off the step
// payload types, so the order here IS the reverse-walk order.
const (
	stepValidateWorldTransfer   = "validate_world_transfer"
	stepLeaveGuildForTransfer   = "leave_guild_for_transfer"
	stepLeavePartyForTransfer   = "leave_party_for_transfer"
	stepSeverBuddiesForTransfer = "sever_buddies_for_transfer"
	stepChangeCharacterWorld    = "change_character_world"
)

// worldTransferCommandProvider builds the five-step WorldTransfer saga in the
// fixed order validate -> leave_guild -> leave_party -> sever_buddies ->
// change_character_world.
//
// change_character_world is LAST on purpose: it is a single-row update, so a
// failure anywhere leaves the character in the source world with only
// recoverable severances applied — which is the whole of FR-4.8.
//
// guildTitle and buddyIds are snapshot values captured by the caller BEFORE
// any severance runs. They exist solely so the compensations can be exact: a
// guild re-join is not a client-driveable recovery, and the severed buddy ids
// cannot be re-read once deleted. Passing them through the payload is not
// redundancy — it is the only copy that survives the severance.
//
// SourceWorldId comes from the record rather than being re-read, exactly as
// design §4 intends ("character_pending_changes.source_world_id exists
// precisely so compensation does not have to reconstruct where the character
// came from").
func worldTransferCommandProvider(m Model, guildId uint32, guildTitle byte, partyId uint32, buddyIds []uint32) model.Provider[[]kafka.Message] {
	s := sharedsaga.NewBuilder().
		SetTransactionId(sagaTransactionId(m, sagaPurposeWorldTransfer)).
		SetSagaType(sharedsaga.WorldTransfer).
		SetInitiatedBy(sagaInitiator).
		AddStep(stepValidateWorldTransfer, sharedsaga.Pending, sharedsaga.ValidateWorldTransfer, sharedsaga.ValidateWorldTransferPayload{
			CharacterId:        m.CharacterId(),
			SourceWorldId:      m.SourceWorldId(),
			DestinationWorldId: m.DestinationWorldId(),
			PendingChangeId:    m.Id(),
		}).
		AddStep(stepLeaveGuildForTransfer, sharedsaga.Pending, sharedsaga.LeaveGuildForTransfer, sharedsaga.LeaveGuildForTransferPayload{
			CharacterId: m.CharacterId(),
			WorldId:     m.SourceWorldId(),
			GuildId:     guildId,
			Title:       guildTitle,
		}).
		AddStep(stepLeavePartyForTransfer, sharedsaga.Pending, sharedsaga.LeavePartyForTransfer, sharedsaga.LeavePartyForTransferPayload{
			CharacterId: m.CharacterId(),
			WorldId:     m.SourceWorldId(),
			PartyId:     partyId,
		}).
		AddStep(stepSeverBuddiesForTransfer, sharedsaga.Pending, sharedsaga.SeverBuddiesForTransfer, sharedsaga.SeverBuddiesForTransferPayload{
			CharacterId: m.CharacterId(),
			WorldId:     m.SourceWorldId(),
			BuddyIds:    buddyIds,
		}).
		AddStep(stepChangeCharacterWorld, sharedsaga.Pending, sharedsaga.ChangeCharacterWorld, sharedsaga.ChangeCharacterWorldPayload{
			CharacterId:        m.CharacterId(),
			SourceWorldId:      m.SourceWorldId(),
			DestinationWorldId: m.DestinationWorldId(),
			PendingChangeId:    m.Id(),
		}).
		Build()
	return sagaCommandProvider(s)
}
