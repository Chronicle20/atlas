package kite

import (
	kite2 "atlas-channel/kafka/message/kite"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// CreateCommandProvider produces the CREATE command keyed on characterId, so
// one character's placements are totally ordered within a partition
// (matching chalkboard/producer.go:14).
func CreateCommandProvider(f field.Model, characterId uint32, name string, templateId uint32, message string, x int16, y int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &kite2.Command[kite2.CreateCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        kite2.CommandKiteCreate,
		Body: kite2.CreateCommandBody{
			Name:       name,
			TemplateId: templateId,
			Message:    message,
			X:          x,
			Y:          y,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
