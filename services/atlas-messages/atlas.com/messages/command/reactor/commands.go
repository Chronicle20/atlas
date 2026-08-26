package reactor

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/kafka/message/reactor"
	"atlas-messages/message"
	"context"
	"regexp"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func ReactorDestroyAllCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			ch := f.Channel()
			re := regexp.MustCompile(`^@reactor destroy all$`)
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
					f := field.NewBuilder(ch.WorldId(), ch.Id(), f.MapId()).Build()

					err := producer.ProviderImpl(l)(ctx)(reactor.EnvCommandTopic)(reactor.DestroyInFieldCommandProvider(ch.WorldId(), ch.Id(), f.MapId(), f.Instance()))
					if err != nil {
						return msgProc.IssuePinkText(f, 0, "Failed to destroy all reactors.", []uint32{c.Id()})
					}

					return msgProc.IssuePinkText(f, 0, "Destroyed all reactors in map and cleared cooldowns.", []uint32{c.Id()})
				}
			}, true
		}
	}
}
