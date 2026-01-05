package monster

import (
	consumer2 "atlas-quest/kafka/consumer"
	"atlas-quest/kafka/message/monster"
	"atlas-quest/quest"
	"context"
	"strconv"

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
			rf(consumer2.NewConfig(l)("monster_status")(monster.EnvEventTopicMonsterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(monster.EnvEventTopicMonsterStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMonsterKilledEvent(db))))
		}
	}
}

func handleMonsterKilledEvent(db *gorm.DB) message.Handler[monster.StatusEvent[monster.StatusEventKilledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster.StatusEvent[monster.StatusEventKilledBody]) {
		if e.Type != monster.EventMonsterStatusKilled {
			return
		}

		// Process monster kill for each character that dealt damage
		for _, entry := range e.Body.DamageEntries {
			if entry.CharacterId == 0 {
				continue
			}

			// Get all started quests for this character
			quests, err := quest.NewProcessor(l, ctx, db).GetByCharacterIdAndState(entry.CharacterId, quest.StateStarted)
			if err != nil {
				l.WithError(err).Debugf("Unable to get started quests for character [%d].", entry.CharacterId)
				continue
			}

			// For each quest, check if it tracks this monster and update progress
			// The infoNumber for monster kills is typically the monsterId
			for _, q := range quests {
				// Check if this quest tracks this monster (by having progress with infoNumber = monsterId)
				if p, found := q.GetProgress(e.MonsterId); found {
					// Increment the kill count
					currentCount := parseProgress(p.Progress())
					newCount := currentCount + 1
					err = quest.NewProcessor(l, ctx, db).SetProgress(entry.CharacterId, q.QuestId(), e.MonsterId, strconv.Itoa(int(newCount)))
					if err != nil {
						l.WithError(err).Errorf("Unable to update monster kill progress for quest [%d] character [%d].", q.QuestId(), entry.CharacterId)
					} else {
						l.Debugf("Updated monster [%d] kill progress for quest [%d] character [%d]: %d -> %d.", e.MonsterId, q.QuestId(), entry.CharacterId, currentCount, newCount)
					}
				}
			}
		}
	}
}

func parseProgress(progress string) uint32 {
	if progress == "" {
		return 0
	}
	val, err := strconv.Atoi(progress)
	if err != nil {
		return 0
	}
	return uint32(val)
}
