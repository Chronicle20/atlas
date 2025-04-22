package equipable

import (
	"atlas-consumables/kafka/message/equipable"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

type Change func(b *ModelBuilder)

func changeEquipableProvider(characterId uint32, e Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &equipable.Command[equipable.ChangeBody]{
		CharacterId: characterId,
		Id:          e.Id(),
		Type:        equipable.CommandChange,
		Body: equipable.ChangeBody{
			Strength:       e.Strength(),
			Dexterity:      e.Dexterity(),
			Intelligence:   e.Intelligence(),
			Luck:           e.Luck(),
			HP:             e.HP(),
			MP:             e.MP(),
			WeaponAttack:   e.WeaponAttack(),
			MagicAttack:    e.MagicAttack(),
			WeaponDefense:  e.WeaponDefense(),
			MagicDefense:   e.MagicDefense(),
			Accuracy:       e.Accuracy(),
			Avoidability:   e.Avoidability(),
			Hands:          e.Hands(),
			Speed:          e.Speed(),
			Jump:           e.Jump(),
			Slots:          e.Slots(),
			OwnerName:      e.OwnerName(),
			Locked:         e.Locked(),
			Spikes:         e.Spikes(),
			KarmaUsed:      e.KarmaUsed(),
			Cold:           e.Cold(),
			CanBeTraded:    e.CanBeTraded(),
			LevelType:      e.LevelType(),
			Level:          e.Level(),
			Experience:     e.Experience(),
			HammersApplied: e.HammersApplied(),
			Expiration:     e.Expiration(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
