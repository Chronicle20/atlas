package saga

import (
	"atlas-saga-orchestrator/kafka/message/gachapon"
	"atlas-saga-orchestrator/kafka/message/saga"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
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

func FailedStatusEventProvider(transactionId uuid.UUID, characterId uint32, sagaType string, errorCode string, reason string, failedStep string) model.Provider[[]kafka.Message] {
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
		},
	}
	return producer.SingleMessageProvider(key, value)
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
