package party_quest

import (
	"atlas-messages/character"
	"atlas-messages/command"
	pq "atlas-messages/kafka/message/party_quest"
	"atlas-messages/kafka/producer"
	"atlas-messages/message"
	party_quest "atlas-messages/party_quest"
	"context"
	"fmt"
	"regexp"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

func PQRegisterCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
		return func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`^@pq\s+register\s+(\S+)$`)
			match := re.FindStringSubmatch(m)
			if len(match) < 2 {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			questId := match[1]

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					msgProc := message.NewProcessor(l, ctx)
					f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()

					err := producer.ProviderImpl(l)(ctx)(pq.EnvCommandTopic)(pq.RegisterCommandProvider(ch.WorldId(), c.Id(), questId, ch.Id(), uint32(c.MapId())))
					if err != nil {
						return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Failed to register for PQ [%s].", questId), []uint32{c.Id()})
					}

					return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Registering for PQ [%s].", questId), []uint32{c.Id()})
				}
			}, true
		}
	}
}

func PQStageCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
		return func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`^@pq\s+stage$`)
			match := re.FindStringSubmatch(m)
			if match == nil {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					msgProc := message.NewProcessor(l, ctx)
					f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()

					inst, err := party_quest.NewProcessor(l, ctx).GetByCharacter(c.Id())
					if err != nil {
						return msgProc.IssuePinkText(f, 0, "Not currently in a party quest.", []uint32{c.Id()})
					}

					err = producer.ProviderImpl(l)(ctx)(pq.EnvCommandTopic)(pq.StageAdvanceCommandProvider(ch.WorldId(), c.Id(), inst.Id()))
					if err != nil {
						return msgProc.IssuePinkText(f, 0, "Failed to advance PQ stage.", []uint32{c.Id()})
					}

					return msgProc.IssuePinkText(f, 0, "Advancing PQ stage.", []uint32{c.Id()})
				}
			}, true
		}
	}
}
