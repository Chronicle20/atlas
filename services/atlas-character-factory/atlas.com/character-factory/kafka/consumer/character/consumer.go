package character

import (
	consumer2 "atlas-character-factory/kafka/consumer"
	character3 "atlas-character-factory/kafka/message/character"
	"atlas-character-factory/factory"
	"atlas-character-factory/saga"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/async"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
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

// characterCreatedHandler handles character created status events to create follow-up sagas
func characterCreatedHandler() message.Handler[character3.StatusEvent[character3.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event character3.StatusEvent[character3.StatusEventCreatedBody]) {
		if event.Type != character3.EventCharacterStatusTypeCreated {
			return
		}

		t := tenant.MustFromContext(ctx)
		
		// Retrieve the stored follow-up saga template
		template, exists := factory.GetFollowUpSagaTemplate(t.Id(), event.Body.Name)
		if !exists {
			l.Debugf("No follow-up saga template found for character [%s], skipping follow-up saga creation", event.Body.Name)
			return
		}

		// Remove the template from storage to avoid reprocessing
		factory.RemoveFollowUpSagaTemplate(t.Id(), event.Body.Name)

		// Generate a new transaction ID for the follow-up saga
		followUpTransactionId := uuid.New()
		
		l.Debugf("Creating follow-up saga for character [%s] with ID [%d] and transaction [%s]", 
			event.Body.Name, event.CharacterId, followUpTransactionId.String())

		// Build the follow-up saga with the actual character ID
		followUpSaga := factory.BuildCharacterCreationFollowUpSaga(
			followUpTransactionId,
			event.CharacterId, // Use the actual character ID from the event
			template.Input,
			template.Template,
		)

		// Emit the follow-up saga
		sagaProcessor := saga.NewProcessor(l, ctx)
		err := sagaProcessor.Create(followUpSaga)
		if err != nil {
			l.WithError(err).Errorf("Failed to emit follow-up saga for character [%s] with ID [%d]", 
				event.Body.Name, event.CharacterId)
			return
		}

		l.Debugf("Follow-up saga [%s] emitted successfully for character [%s] with ID [%d]", 
			followUpTransactionId.String(), event.Body.Name, event.CharacterId)
	}
}

// RegisterPersistentHandlers registers persistent message handlers
func RegisterPersistentHandlers(l logrus.FieldLogger, ctx context.Context) {
	t, _ := topic.EnvProvider(l)(character3.EnvEventTopicCharacterStatus)()
	
	// Create a persistent handler that re-registers itself
	var registerHandler func()
	registerHandler = func() {
		hid, err := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(
			func(l logrus.FieldLogger, ctx context.Context, event character3.StatusEvent[character3.StatusEventCreatedBody]) bool {
				// Accept all character created events
				return event.Type == character3.EventCharacterStatusTypeCreated
			},
			func(l logrus.FieldLogger, ctx context.Context, event character3.StatusEvent[character3.StatusEventCreatedBody]) {
				// Process the event
				characterCreatedHandler()(l, ctx, event)
				
				// Re-register the handler for the next event
				go registerHandler()
			})))
		if err != nil {
			l.WithError(err).Errorf("Failed to register character created handler")
		}
		
		l.Debugf("Registered persistent character created handler [%s]", hid)
	}
	
	// Start the registration
	registerHandler()
}
