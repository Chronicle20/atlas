package equipable

import (
	"atlas-consumables/kafka/message/equipable"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

type Change func(m *equipable.ChangeBody)

func changeEquipableProvider(characterId uint32, itemId uint32, slot int16, changes ...Change) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))

	cb := &equipable.ChangeBody{}
	for _, change := range changes {
		change(cb)
	}

	value := &equipable.Command[equipable.ChangeBody]{
		CharacterId: characterId,
		ItemId:      itemId,
		Slot:        slot,
		Type:        equipable.CommandChange,
		Body:        *cb,
	}
	return producer.SingleMessageProvider(key, value)
}
