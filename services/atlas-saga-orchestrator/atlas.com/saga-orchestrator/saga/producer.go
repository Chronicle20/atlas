package saga

import (
	"atlas-saga-orchestrator/kafka/message/gachapon"
	"atlas-saga-orchestrator/kafka/message/saga"
	"atlas-saga-orchestrator/kafka/producer"
	"context"

	kproducer "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func CompletedStatusEventProvider(s Saga) model.Provider[[]kafka.Message] {
	key := kproducer.CreateKey(int(s.TransactionId().ID()))

	body := saga.StatusEventCompletedBody{
		SagaType: string(s.SagaType()),
	}

	// For CharacterCreation sagas, include accountId and characterId in the results
	if s.SagaType() == CharacterCreation {
		body.Results = extractCharacterCreationResults(s)
	}

	value := &saga.StatusEvent[saga.StatusEventCompletedBody]{
		TransactionId: s.TransactionId(),
		Type:          saga.StatusEventTypeCompleted,
		Body:          body,
	}
	return kproducer.SingleMessageProvider(key, value)
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
	key := kproducer.CreateKey(int(transactionId.ID()))
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
	return kproducer.SingleMessageProvider(key, value)
}

// EmitSagaFailed emits exactly one StatusEventTypeFailed for the given saga,
// extracting accountId/characterId from the CharacterCreatePayload where present.
// Callers are expected to have already taken the terminal-state guard (see PRD §4.7).
// Returns the producer error, if any.
func EmitSagaFailed(l logrus.FieldLogger, ctx context.Context, s Saga, errorCode, reason, failedStep string) error {
	accountId, characterId := ExtractCharacterCreationIds(s)
	return EmitSagaFailedByIds(l, ctx, s.TransactionId(), string(s.SagaType()), accountId, characterId, errorCode, reason, failedStep)
}

// EmitSagaFailedByIds is the thin variant for paths where a full Saga struct is
// not in hand (e.g., the saga consumer's Put() error path, where validation
// rejected the incoming command before it was inserted).
func EmitSagaFailedByIds(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, sagaType string, accountId, characterId uint32, errorCode, reason, failedStep string) error {
	return producer.ProviderImpl(l)(ctx)(saga.EnvStatusEventTopic)(
		FailedStatusEventProvider(transactionId, accountId, characterId, sagaType, errorCode, reason, failedStep),
	)
}

func GachaponRewardWonEventProvider(payload EmitGachaponWinPayload, assetId uint32) model.Provider[[]kafka.Message] {
	key := kproducer.CreateKey(int(payload.CharacterId))
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
	return kproducer.SingleMessageProvider(key, value)
}
