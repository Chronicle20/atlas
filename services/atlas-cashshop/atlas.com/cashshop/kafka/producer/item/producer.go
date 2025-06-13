package item

import (
	"atlas-cashshop/kafka/message/item"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreateStatusEventProvider(id uint32, cashId int64, templateId uint32, quantity uint32, purchasedBy uint32, flag uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(id))
	value := &item.StatusEvent[item.StatusEventCreatedBody]{
		CharacterId: id,
		Type:        item.StatusCreated,
		Body: item.StatusEventCreatedBody{
			CashId:      cashId,
			TemplateId:  templateId,
			Quantity:    quantity,
			PurchasedBy: purchasedBy,
			Flag:        flag,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
