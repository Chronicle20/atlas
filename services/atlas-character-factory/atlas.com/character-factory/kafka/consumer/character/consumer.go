package character

import (
	consumer2 "atlas-character-factory/kafka/consumer"
	character3 "atlas-character-factory/kafka/message/character"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/async"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(character3.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func createdValidator(t tenant.Model) func(name string) func(l logrus.FieldLogger, ctx context.Context, event character3.StatusEvent[character3.StatusEventCreatedBody]) bool {
	return func(name string) func(l logrus.FieldLogger, ctx context.Context, event character3.StatusEvent[character3.StatusEventCreatedBody]) bool {
		return func(l logrus.FieldLogger, ctx context.Context, event character3.StatusEvent[character3.StatusEventCreatedBody]) bool {
			if !t.Is(tenant.MustFromContext(ctx)) {
				return false
			}
			if event.Type != character3.EventCharacterStatusTypeCreated {
				return false
			}
			if name != event.Body.Name {
				return false
			}
			return true
		}
	}
}

func createdHandler(rchan chan uint32, _ chan error) message.Handler[character3.StatusEvent[character3.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, m character3.StatusEvent[character3.StatusEventCreatedBody]) {
		rchan <- m.CharacterId
	}
}

func AwaitCreated(l logrus.FieldLogger) func(name string) async.Provider[uint32] {
	t, _ := topic.EnvProvider(l)(character3.EnvEventTopicCharacterStatus)()
	return func(name string) async.Provider[uint32] {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			hid, err := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(createdValidator(tenant.MustFromContext(ctx))(name), createdHandler(rchan, echan))))
			if err != nil {
				echan <- err
			}
			l.Debugf("Creating OneTime topic consumer to await [%s] character creation. Handler [%s].", name, hid)
			go func() {
				<-ctx.Done()
				l.Debugf("Removing handler [%s].", hid)
				_ = consumer.GetManager().RemoveHandler(t, hid)
			}()
		}
	}
}
