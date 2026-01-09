package compartment

import (
	"atlas-compartment-transfer/kafka/message/storage/compartment"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func AcceptCommandProvider(worldId byte, accountId uint32, transactionId uuid.UUID, slot int16, templateId uint32, referenceId uint32, referenceType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &compartment.Command[compartment.AcceptCommandBody]{
		WorldId:   worldId,
		AccountId: accountId,
		Type:      compartment.CommandAccept,
		Body: compartment.AcceptCommandBody{
			TransactionId: transactionId,
			Slot:          slot,
			TemplateId:    templateId,
			ReferenceId:   referenceId,
			ReferenceType: referenceType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ReleaseCommandProvider(worldId byte, accountId uint32, transactionId uuid.UUID, assetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &compartment.Command[compartment.ReleaseCommandBody]{
		WorldId:   worldId,
		AccountId: accountId,
		Type:      compartment.CommandRelease,
		Body: compartment.ReleaseCommandBody{
			TransactionId: transactionId,
			AssetId:       assetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
