package compartment

import (
	"atlas-compartment-transfer/kafka/message/cashshop/compartment"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func AcceptCommandProvider(accountId uint32, compartmentId uuid.UUID, compartmentType byte, transactionId uuid.UUID, referenceId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &compartment.Command[compartment.AcceptCommandBody]{
		AccountId:       accountId,
		CompartmentType: compartmentType,
		Type:            compartment.CommandAccept,
		Body: compartment.AcceptCommandBody{
			TransactionId: transactionId,
			CompartmentId: compartmentId,
			ReferenceId:   referenceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ReleaseCommandProvider(accountId uint32, compartmentId uuid.UUID, compartmentType byte, transactionId uuid.UUID, referenceId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &compartment.Command[compartment.ReleaseCommandBody]{
		AccountId:       accountId,
		CompartmentType: compartmentType,
		Type:            compartment.CommandRelease,
		Body: compartment.ReleaseCommandBody{
			TransactionId: transactionId,
			CompartmentId: compartmentId,
			AssetId:       referenceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
