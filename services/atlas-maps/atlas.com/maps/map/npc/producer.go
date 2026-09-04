package npc

import (
	npcKafka "atlas-maps/kafka/message/npc"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// emit is the Kafka-publish seam for the npc package, overridable in tests
// without a real Kafka connection -- mirrors atlas-channel's
// doorAnnounce/playerNpcAnnounce package-level var-swap pattern
// (kafka/consumer/map/player_npc.go).
var emit = func(l logrus.FieldLogger, ctx context.Context, t topic.Token, prov model.Provider[[]kafka.Message]) error {
	return producer.ProviderImpl(l)(ctx)(t)(prov)
}

// CreatedEventProvider builds the CREATED status event for a newly placed
// scripted NPC m on field f, mirroring map/weather's WeatherStartEventProvider
// shape (single-message, keyed by producer.CreateKey(mapId)).
func CreatedEventProvider(f field.Model, m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &npcKafka.StatusEvent[npcKafka.CreatedStatusEventBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		UniqueId:  m.UniqueId(),
		Type:      npcKafka.EventNpcStatusTypeCreated,
		Body: npcKafka.CreatedStatusEventBody{
			NpcId: m.NpcId(),
			X:     m.X(),
			Y:     m.Y(),
			Fh:    m.Fh(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
