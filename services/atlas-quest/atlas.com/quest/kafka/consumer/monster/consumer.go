package monster

import (
	consumer2 "atlas-quest/kafka/consumer"
	"atlas-quest/kafka/message/monster"
	"atlas-quest/quest"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
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

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(monster.EnvEventTopicMonsterStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMonsterKilledEvent(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleMonsterKilledEvent(db *gorm.DB) message.Handler[monster.StatusEvent[monster.StatusEventKilledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster.StatusEvent[monster.StatusEventKilledBody]) {
		if e.Type != monster.EventMonsterStatusKilled {
			return
		}

		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()

		// Process monster kill for each character that dealt damage
		for _, entry := range e.Body.DamageEntries {
			if entry.CharacterId == 0 {
				continue
			}

			processor := quest.NewProcessor(l, ctx, db)

			// Get all started quests for this character
			quests, err := processor.GetByCharacterIdAndState(entry.CharacterId, quest.StateStarted)
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
					// Use uuid.Nil since this is not saga-initiated
					err = processor.SetProgress(uuid.Nil, entry.CharacterId, q.QuestId(), e.MonsterId, formatProgress(newCount))
					if err != nil {
						l.WithError(err).Errorf("Unable to update monster kill progress for quest [%d] character [%d].", q.QuestId(), entry.CharacterId)
					} else {
						l.Debugf("Updated monster [%d] kill progress for quest [%d] character [%d]: %d -> %d.", e.MonsterId, q.QuestId(), entry.CharacterId, currentCount, newCount)

						// Check for auto-complete after progress update
						nextQuestId, completed, err := processor.CheckAutoComplete(entry.CharacterId, q.QuestId(), f)
						if err != nil {
							l.WithError(err).Warnf("Unable to check auto-complete for quest [%d] character [%d].", q.QuestId(), entry.CharacterId)
						} else if completed {
							l.Infof("Auto-completed quest [%d] for character [%d].", q.QuestId(), entry.CharacterId)
							// Handle quest chain - auto-start next quest if present
							if nextQuestId > 0 {
								// Use uuid.Nil since this is not saga-initiated
								_, err = processor.StartChained(uuid.Nil, entry.CharacterId, nextQuestId, f)
								if err != nil {
									l.WithError(err).Errorf("Error starting chained quest [%d] for character [%d].", nextQuestId, entry.CharacterId)
								}
							}
						}
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
	var val int
	_, _ = fmt.Sscanf(progress, "%d", &val)
	return uint32(val)
}

// formatProgress formats a progress value as a 3-digit zero-padded string
func formatProgress(count uint32) string {
	return fmt.Sprintf("%03d", count)
}
