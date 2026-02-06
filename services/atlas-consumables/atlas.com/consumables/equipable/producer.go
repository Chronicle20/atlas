package equipable

import (
	"atlas-consumables/asset"
	"atlas-consumables/kafka/message/equipable"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

type Change func(b *asset.EquipableReferenceDataBuilder)

func changeEquipableProvider(a asset.Model[asset.EquipableReferenceData]) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &equipable.Command[equipable.ChangeBody]{
		Id:   a.ReferenceId(),
		Type: equipable.CommandChange,
		Body: equipable.ChangeBody{
			Strength:       a.ReferenceData().Strength(),
			Dexterity:      a.ReferenceData().Dexterity(),
			Intelligence:   a.ReferenceData().Intelligence(),
			Luck:           a.ReferenceData().Luck(),
			HP:             a.ReferenceData().HP(),
			MP:             a.ReferenceData().MP(),
			WeaponAttack:   a.ReferenceData().WeaponAttack(),
			MagicAttack:    a.ReferenceData().MagicAttack(),
			WeaponDefense:  a.ReferenceData().WeaponDefense(),
			MagicDefense:   a.ReferenceData().MagicDefense(),
			Accuracy:       a.ReferenceData().Accuracy(),
			Avoidability:   a.ReferenceData().Avoidability(),
			Hands:          a.ReferenceData().Hands(),
			Speed:          a.ReferenceData().Speed(),
			Jump:           a.ReferenceData().Jump(),
			Slots:          a.ReferenceData().Slots(),
			OwnerName:      "", //TODO
			Locked:         a.ReferenceData().IsLocked(),
			Spikes:         a.ReferenceData().HasSpikes(),
			KarmaUsed:      a.ReferenceData().IsKarmaUsed(),
			Cold:           a.ReferenceData().IsCold(),
			CanBeTraded:    a.ReferenceData().CanBeTraded(),
			LevelType:      a.ReferenceData().LevelType(),
			Level:          a.ReferenceData().Level(),
			Experience:     a.ReferenceData().Experience(),
			HammersApplied: a.ReferenceData().HammersApplied(),
			Expiration:     a.Expiration(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
