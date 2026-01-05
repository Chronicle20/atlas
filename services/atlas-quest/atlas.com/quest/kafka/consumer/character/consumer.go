package character

import (
	consumer2 "atlas-quest/kafka/consumer"
	"atlas-quest/kafka/message/character"
	"atlas-quest/quest"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status")(character.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(character.EnvEventTopicCharacterStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMapChangedEvent(db))))
		}
	}
}

func handleMapChangedEvent(db *gorm.DB) message.Handler[character.StatusEvent[character.StatusEventMapChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[character.StatusEventMapChangedBody]) {
		if e.Type != character.EventCharacterStatusTypeMapChanged {
			return
		}

		if e.CharacterId == 0 {
			return
		}

		targetMapId := uint32(e.Body.TargetMapId)

		// Get all started quests for this character
		quests, err := quest.NewProcessor(l, ctx, db).GetByCharacterIdAndState(e.CharacterId, quest.StateStarted)
		if err != nil {
			l.WithError(err).Debugf("Unable to get started quests for character [%d].", e.CharacterId)
			return
		}

		// For each quest, check if it tracks this map and update progress
		// The infoNumber for map visits is typically the mapId
		for _, q := range quests {
			// Check if this quest tracks this map (by having progress with infoNumber = mapId)
			if _, found := q.GetProgress(targetMapId); found {
				// Set progress to "1" to indicate the map has been visited
				err = quest.NewProcessor(l, ctx, db).SetProgress(e.CharacterId, q.QuestId(), targetMapId, "1")
				if err != nil {
					l.WithError(err).Errorf("Unable to update map visit progress for quest [%d] character [%d].", q.QuestId(), e.CharacterId)
				} else {
					l.Debugf("Updated map [%d] visit progress for quest [%d] character [%d].", targetMapId, q.QuestId(), e.CharacterId)
				}
			}
		}
	}
}
