package cashshop

import (
	"atlas-saga-orchestrator/kafka/message/cashshop"
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
