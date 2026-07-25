package saga

import (
	"atlas-saga-orchestrator/kafka/message/broadcast"
	"atlas-saga-orchestrator/kafka/message/conversation_reward_notice"
	"atlas-saga-orchestrator/kafka/message/gachapon"
	"atlas-saga-orchestrator/kafka/message/incubator"
	"atlas-saga-orchestrator/kafka/message/megaphone"
	"atlas-saga-orchestrator/kafka/message/saga"
	"context"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func CompletedStatusEventProvider(s Saga) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(s.TransactionId().ID()))

	body := saga.StatusEventCompletedBody{
		SagaType: string(s.SagaType()),
	}

	// For CharacterCreation sagas, include accountId and characterId in the results
	if s.SagaType() == CharacterCreation {
		body.Results = extractCharacterCreationResults(s)
	}

	// For a completed take-home (WithdrawFromMts) saga, include the take-home
	// marker + characterId + templateId so the channel's saga-status COMPLETED
	// handler can write MoveItcPurchaseItemLtoSDone to the originating session.
	// This is the ONLY place that fires after the full saga (release + grant)
	// completes; emitting earlier (e.g. from the release custody handler) would
	// signal success before the item is granted and wrongly on a compensated saga.
	if r := extractMtsTakeHomeResults(s); r != nil {
		body.Results = r
	}

	value := &saga.StatusEvent[saga.StatusEventCompletedBody]{
		TransactionId: s.TransactionId(),
		Type:          saga.StatusEventTypeCompleted,
		Body:          body,
	}
	return producer.SingleMessageProvider(key, value)
}

// extractCharacterCreationResults extracts accountId and characterId from a CharacterCreation saga's steps
func extractCharacterCreationResults(s Saga) map[string]any {
	results := make(map[string]any)
	for _, step := range s.Steps() {
		if step.Action() == CreateCharacter {
			if p, ok := step.Payload().(CharacterCreatePayload); ok {
				results["accountId"] = p.AccountId
			}
			if step.Result() != nil {
				if cid := extractUint32(step.Result(), "characterId"); cid != 0 {
					results["characterId"] = cid
				}
			}
			break
		}
	}
	return results
}

// MtsTakeHomeResultKind is the Results["kind"] marker the channel matches to
// recognize a completed WithdrawFromMts (take-home) saga and write
// MoveItcPurchaseItemLtoSDone. It distinguishes take-home from the other
// MtsOperation sagas (list / buy / settle) that share the same saga type but do
// NOT grant a holding back to a character's inventory.
const MtsTakeHomeResultKind = "mts_take_home"

// extractMtsTakeHomeResults returns the COMPLETED Results map for a take-home
// (WithdrawFromMts) saga, or nil if this is not one. WithdrawFromMts expands to
// release_from_mts_holding (ReleaseFromMtsHolding) + accept_to_character
// (AcceptToCharacter); the ReleaseFromMtsHolding action is unique to take-home
// among MtsOperation sagas, so its presence is the discriminator. The channel's
// saga-status COMPLETED handler reads characterId off the result to target the
// originating session. This fires from the single guarded terminal-completion
// emit, so the notice is sent only after the item was actually granted.
func extractMtsTakeHomeResults(s Saga) map[string]any {
	if s.SagaType() != MtsOperation {
		return nil
	}
	isTakeHome := false
	for _, step := range s.Steps() {
		if step.Action() == ReleaseFromMtsHolding {
			isTakeHome = true
			break
		}
	}
	if !isTakeHome {
		return nil
	}

	results := map[string]any{"kind": MtsTakeHomeResultKind}
	for _, step := range s.Steps() {
		if step.Action() != AcceptToCharacter {
			continue
		}
		if p, ok := step.Payload().(AcceptToCharacterPayload); ok {
			results["characterId"] = p.CharacterId
			results["templateId"] = p.TemplateId
		}
		break
	}
	return results
}

// ExtractCharacterCreationIds returns (accountId, characterId) from a
// CharacterCreation saga's CreateCharacter step. accountId is taken from
// the payload (known at saga acceptance); characterId is taken from the
// step result (known only after the step completes). Both are 0 if the
// saga has no CreateCharacter step (i.e., non-character-creation).
func ExtractCharacterCreationIds(s Saga) (accountId uint32, characterId uint32) {
	for _, step := range s.Steps() {
		if step.Action() != CreateCharacter {
			continue
		}
		if p, ok := step.Payload().(CharacterCreatePayload); ok {
			accountId = p.AccountId
		}
		if step.Result() != nil {
			characterId = extractUint32(step.Result(), "characterId")
		}
		return
	}
	return
}

// FailedStatusEventProvider builds a StatusEventTypeFailed provider.
// accountId is 0 for non-character-creation sagas (login resolves sessions by
// accountId, so only character-creation failures need a nonzero value — see
// PRD §4.5 / plan Phase 1.1).
func FailedStatusEventProvider(transactionId uuid.UUID, accountId uint32, characterId uint32, sagaType string, errorCode string, reason string, failedStep string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(transactionId.ID()))
	value := &saga.StatusEvent[saga.StatusEventFailedBody]{
		TransactionId: transactionId,
		Type:          saga.StatusEventTypeFailed,
		Body: saga.StatusEventFailedBody{
			Reason:      reason,
			FailedStep:  failedStep,
			CharacterId: characterId,
			AccountId:   accountId,
			SagaType:    sagaType,
			ErrorCode:   errorCode,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// EmitSagaFailed emits exactly one StatusEventTypeFailed for the given saga,
// extracting accountId/characterId from the CharacterCreatePayload where present.
// Callers are expected to have already taken the terminal-state guard (see PRD §4.7).
// Returns the producer error, if any.
func EmitSagaFailed(l logrus.FieldLogger, ctx context.Context, s Saga, errorCode, reason, failedStep string) error {
	// An mts_operation saga carries no CharacterCreation ids; extract the
	// originating character (buyer/seller/take-home recipient) and operation kind
	// so the channel can unhang the matching MTS dialog. Without this the FAILED
	// event has characterId 0 and no kind, and the client hangs forever (task-102).
	if s.SagaType() == MtsOperation {
		characterId, kind := extractMtsFailureTarget(s)
		return EmitMtsSagaFailed(l, ctx, s.TransactionId(), string(s.SagaType()), characterId, errorCode, reason, failedStep, kind)
	}
	accountId, characterId := ExtractCharacterCreationIds(s)
	return EmitSagaFailedByIds(l, ctx, s.TransactionId(), string(s.SagaType()), accountId, characterId, errorCode, reason, failedStep)
}

// extractMtsFailureTarget determines, for an mts_operation saga, which character
// to notify of the failure and which MTS operation was in flight, so the channel
// can write the matching clientbound *Failed arm. It mirrors the step
// discriminators used by extractMtsTakeHomeResults and the compensation
// reverse-walk (compensator.go):
//   - a ReleaseFromMtsHolding step => take-home (WithdrawFromMts); the character
//     is the AcceptToCharacter recipient.
//   - an MtsMoveListingToHolding step => buy/settle; the character is the buyer.
//   - an AcceptToMtsListing step => list (TransferToMts); the character is the seller.
//
// Returns (0, "") if none match (kind unknown; the channel then skips notifying,
// rather than guessing a dialog arm).
func extractMtsFailureTarget(s Saga) (characterId uint32, kind string) {
	steps := s.Steps()
	// Take-home first: ReleaseFromMtsHolding is unique to it among MTS sagas.
	for _, step := range steps {
		if step.Action() == ReleaseFromMtsHolding {
			for _, st := range steps {
				if st.Action() == AcceptToCharacter {
					if p, ok := st.Payload().(AcceptToCharacterPayload); ok {
						return p.CharacterId, saga.MtsFailureKindTakeHome
					}
				}
			}
			return 0, saga.MtsFailureKindTakeHome
		}
	}
	// Buy/settle: the move step carries the buyer id.
	for _, step := range steps {
		if step.Action() == MtsMoveListingToHolding {
			if p, ok := step.Payload().(MtsMoveListingToHoldingPayload); ok {
				return p.BuyerId, saga.MtsFailureKindBuy
			}
			return 0, saga.MtsFailureKindBuy
		}
	}
	// List: the accept-to-listing step carries the seller id.
	for _, step := range steps {
		if step.Action() == AcceptToMtsListing {
			if p, ok := step.Payload().(AcceptToMtsListingPayload); ok {
				return p.SellerId, saga.MtsFailureKindList
			}
			return 0, saga.MtsFailureKindList
		}
	}
	return 0, ""
}

// emitMtsSagaFailedFn is swappable in tests (SetEmitMtsSagaFailedForTest) so the
// MTS integration tests can capture the emitted characterId + kind without Kafka.
var emitMtsSagaFailedFn = emitMtsSagaFailedImpl

// EmitMtsSagaFailed emits a StatusEventTypeFailed for an mts_operation saga,
// carrying the originating characterId (accountId is unused — the channel resolves
// MTS sessions by characterId) and the MtsKind so the channel writes the correct
// *Failed dialog arm.
func EmitMtsSagaFailed(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, sagaType string, characterId uint32, errorCode, reason, failedStep, mtsKind string) error {
	return emitMtsSagaFailedFn(l, ctx, transactionId, sagaType, characterId, errorCode, reason, failedStep, mtsKind)
}

func emitMtsSagaFailedImpl(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, sagaType string, characterId uint32, errorCode, reason, failedStep, mtsKind string) error {
	return producer.ProviderImpl(l)(ctx)(saga.EnvStatusEventTopic)(
		MtsFailedStatusEventProvider(transactionId, characterId, sagaType, errorCode, reason, failedStep, mtsKind),
	)
}

// MtsFailedStatusEventProvider builds a StatusEventTypeFailed provider for an MTS
// saga failure, mirroring FailedStatusEventProvider but stamping MtsKind (and
// leaving accountId 0). Kept separate so the generic Failed path stays untouched.
func MtsFailedStatusEventProvider(transactionId uuid.UUID, characterId uint32, sagaType string, errorCode string, reason string, failedStep string, mtsKind string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(transactionId.ID()))
	value := &saga.StatusEvent[saga.StatusEventFailedBody]{
		TransactionId: transactionId,
		Type:          saga.StatusEventTypeFailed,
		Body: saga.StatusEventFailedBody{
			Reason:      reason,
			FailedStep:  failedStep,
			CharacterId: characterId,
			SagaType:    sagaType,
			ErrorCode:   errorCode,
			MtsKind:     mtsKind,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// emitSagaFailedByIdsFn is swappable in tests (SetEmitSagaFailedForTest) so
// integration tests can count Failed emissions without Kafka.
var emitSagaFailedByIdsFn = emitSagaFailedByIdsImpl

// EmitSagaFailedByIds is the thin variant for paths where a full Saga struct is
// not in hand (e.g., the saga consumer's Put() error path, where validation
// rejected the incoming command before it was inserted).
func EmitSagaFailedByIds(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, sagaType string, accountId, characterId uint32, errorCode, reason, failedStep string) error {
	return emitSagaFailedByIdsFn(l, ctx, transactionId, sagaType, accountId, characterId, errorCode, reason, failedStep)
}

func emitSagaFailedByIdsImpl(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, sagaType string, accountId, characterId uint32, errorCode, reason, failedStep string) error {
	return producer.ProviderImpl(l)(ctx)(saga.EnvStatusEventTopic)(
		FailedStatusEventProvider(transactionId, accountId, characterId, sagaType, errorCode, reason, failedStep),
	)
}

// ConversationRewardNoticeProvider builds a one-message provider for the
// conversation_reward_notice topic, used to render an item-gain effect or
// item-loss chat line on the client when a conversation-sourced AwardAsset /
// DestroyAsset / DestroyAssetFromSlot step completes with ShowEffect=true.
func ConversationRewardNoticeProvider(characterId uint32, kind string, itemId uint32, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &conversation_reward_notice.EventBody{
		CharacterId: characterId,
		Kind:        kind,
		ItemId:      itemId,
		Quantity:    quantity,
	}
	return producer.SingleMessageProvider(key, value)
}

// EmitConversationRewardNotice produces the message immediately on the
// conversation_reward_notice topic. Called from the orchestrator when a
// reward step completes with ShowEffect=true.
//
// Tests may override emitConversationRewardNoticeFn to capture or stub calls.
var emitConversationRewardNoticeFn = emitConversationRewardNoticeImpl

func EmitConversationRewardNotice(l logrus.FieldLogger, ctx context.Context, characterId uint32, kind string, itemId uint32, quantity uint32) error {
	return emitConversationRewardNoticeFn(l, ctx, characterId, kind, itemId, quantity)
}

func emitConversationRewardNoticeImpl(l logrus.FieldLogger, ctx context.Context, characterId uint32, kind string, itemId uint32, quantity uint32) error {
	return producer.ProviderImpl(l)(ctx)(conversation_reward_notice.EnvEventTopic)(
		ConversationRewardNoticeProvider(characterId, kind, itemId, quantity),
	)
}

func GachaponRewardWonEventProvider(payload EmitGachaponWinPayload, assetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(payload.CharacterId))
	value := &gachapon.RewardWonEvent{
		CharacterId:  payload.CharacterId,
		WorldId:      byte(payload.WorldId),
		ItemId:       payload.ItemId,
		Quantity:     payload.Quantity,
		Tier:         payload.Tier,
		GachaponId:   payload.GachaponId,
		GachaponName: payload.GachaponName,
		AssetId:      assetId,
	}
	return producer.SingleMessageProvider(key, value)
}

// IncubatorResultEventProvider builds the EVENT_TOPIC_INCUBATOR_RESULT message
// consumed by the channel service, which announces the incubator result via a
// packet. WorldId/ChannelId are narrowed from world.Id/channel.Id (both
// underlying byte) to the wire event's byte fields.
func IncubatorResultEventProvider(payload IncubatorResultPayload) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(payload.CharacterId))
	value := &incubator.ResultEvent{
		CharacterId: payload.CharacterId,
		WorldId:     byte(payload.WorldId),
		ChannelId:   byte(payload.ChannelId),
		ItemId:      payload.ItemId,
		Count:       payload.Count,
		EggId:       payload.EggId,
	}
	return producer.SingleMessageProvider(key, value)
}

// MegaphoneBroadcastEventProvider builds the megaphone.BroadcastEvent for the
// stateless megaphone tiers (MEGAPHONE/SUPER/ITEM/TRIPLE). Keyed by WorldId
// for single-partition ordering per world (D1).
func MegaphoneBroadcastEventProvider(payload EmitMegaphonePayload) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(payload.WorldId))
	value := &megaphone.BroadcastEvent{
		Tier:        payload.Tier,
		Scope:       payload.Scope,
		WorldId:     byte(payload.WorldId),
		ChannelId:   byte(payload.ChannelId),
		CharacterId: payload.CharacterId,
		SenderName:  payload.SenderName,
		SenderMedal: payload.SenderMedal,
		Messages:    payload.Messages,
		WhispersOn:  payload.WhispersOn,
		Item:        payload.Item,
	}
	return producer.SingleMessageProvider(key, value)
}

// WorldBroadcastEnqueueCommandProvider builds the broadcast.EnqueueCommand
// for the serialized world broadcast tiers (TV/AVATAR). Keyed by WorldId for
// single-partition ordering per world (D1) — atlas-world's per-world queue
// consumer depends on strictly ordered enqueue commands.
func WorldBroadcastEnqueueCommandProvider(payload EnqueueWorldBroadcastPayload) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(payload.WorldId))
	value := &broadcast.EnqueueCommand{
		Family:          payload.Family,
		WorldId:         byte(payload.WorldId),
		ChannelId:       byte(payload.ChannelId),
		CharacterId:     payload.CharacterId,
		SenderName:      payload.SenderName,
		SenderMedal:     payload.SenderMedal,
		Messages:        payload.Messages,
		WhispersOn:      payload.WhispersOn,
		ItemId:          payload.ItemId,
		TvMessageType:   payload.TvMessageType,
		DurationSeconds: payload.DurationSeconds,
		SenderLook:      payload.SenderLook,
		ReceiverName:    payload.ReceiverName,
		ReceiverLook:    payload.ReceiverLook,
	}
	return producer.SingleMessageProvider(key, value)
}
