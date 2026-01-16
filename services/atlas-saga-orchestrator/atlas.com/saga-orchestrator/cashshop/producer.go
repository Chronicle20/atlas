package cashshop

import (
	"atlas-saga-orchestrator/kafka/message/cashshop"
	cashshopCompartment "atlas-saga-orchestrator/kafka/message/cashshop/compartment"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func AdjustCurrencyProvider(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &cashshop.AdjustCurrencyCommand{
		TransactionId: transactionId,
		AccountId:     accountId,
		CurrencyType:  currencyType,
		Amount:        amount,
		Type:          cashshop.CommandTypeAdjustCurrency,
	}
	return producer.SingleMessageProvider(key, value)
}

// AcceptCommandProvider creates an ACCEPT command for the cash shop compartment
func AcceptCommandProvider(characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, transactionId uuid.UUID, cashId int64, templateId uint32, referenceId uint32, referenceType string, referenceData []byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshopCompartment.Command[cashshopCompartment.AcceptCommandBody]{
		AccountId:       accountId,
		CharacterId:     characterId,
		CompartmentType: compartmentType,
		Type:            cashshopCompartment.CommandAccept,
		Body: cashshopCompartment.AcceptCommandBody{
			TransactionId: transactionId,
			CompartmentId: compartmentId,
			CashId:        cashId,
			TemplateId:    templateId,
			ReferenceId:   referenceId,
			ReferenceType: referenceType,
			ReferenceData: referenceData,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ReleaseCommandProvider creates a RELEASE command for the cash shop compartment
func ReleaseCommandProvider(characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, transactionId uuid.UUID, assetId uint32, cashId int64, templateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshopCompartment.Command[cashshopCompartment.ReleaseCommandBody]{
		AccountId:       accountId,
		CharacterId:     characterId,
		CompartmentType: compartmentType,
		Type:            cashshopCompartment.CommandRelease,
		Body: cashshopCompartment.ReleaseCommandBody{
			TransactionId: transactionId,
			CompartmentId: compartmentId,
			AssetId:       assetId,
			CashId:        cashId,
			TemplateId:    templateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
