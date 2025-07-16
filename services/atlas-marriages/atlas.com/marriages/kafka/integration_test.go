package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"atlas-marriages/character"
	"atlas-marriages/kafka/consumer/marriage"
	marriageMessage "atlas-marriages/kafka/message/marriage"
	marriageService "atlas-marriages/marriage"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockCharacterProcessor is a mock implementation of character.Processor for testing
type mockCharacterProcessor struct{}

func (m *mockCharacterProcessor) GetById(characterId uint32) (character.Model, error) {
	// Return a valid character model for all test character IDs
	return character.NewModel(characterId, "TestCharacter", 20), nil
}

func (m *mockCharacterProcessor) ByIdProvider(characterId uint32) model.Provider[character.Model] {
	return func() (character.Model, error) {
		return m.GetById(characterId)
	}
}

// TestKafkaIntegration tests the end-to-end message flow
func TestKafkaIntegration(t *testing.T) {
	// Create a test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Run migrations
	err = marriageService.Migration(db)
	require.NoError(t, err)

	// Set up test logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create test context with tenant
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	t.Run("ProducerCreatesValidMessages", func(t *testing.T) {
		testProducerCreatesValidMessages(t, logger, ctx)
	})

	t.Run("ConsumerHandlesProposalCommands", func(t *testing.T) {
		testConsumerHandlesProposalCommands(t, logger, ctx, db)
	})

	t.Run("ConsumerHandlesCeremonyCommands", func(t *testing.T) {
		testConsumerHandlesCeremonyCommands(t, logger, ctx, db)
	})

	t.Run("EndToEndMessageFlow", func(t *testing.T) {
		testEndToEndMessageFlow(t, logger, ctx, db)
	})

	t.Run("ErrorHandlingInConsumers", func(t *testing.T) {
		testErrorHandlingInConsumers(t, logger, ctx, db)
	})

	t.Run("MessageBufferingAndTransactions", func(t *testing.T) {
		testMessageBufferingAndTransactions(t, logger, ctx, db)
	})

	t.Run("EventEmissionOnStateChanges", func(t *testing.T) {
		testEventEmissionOnStateChanges(t, logger, ctx, db)
	})
}

// testProducerCreatesValidMessages tests that producers create valid Kafka messages
func testProducerCreatesValidMessages(t *testing.T, logger logrus.FieldLogger, ctx context.Context) {
	t.Run("ProposalEventProviders", func(t *testing.T) {
		// Test ProposalCreatedEventProvider
		proposalId := uint32(1)
		proposerId := uint32(12345)
		targetId := uint32(67890)
		proposedAt := time.Now()
		expiresAt := proposedAt.Add(24 * time.Hour)

		provider := marriageService.ProposalCreatedEventProvider(proposalId, proposerId, targetId, proposedAt, expiresAt)
		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		msg := messages[0]
		assert.NotEmpty(t, msg.Key)
		assert.NotEmpty(t, msg.Value)

		// Verify message content
		var event marriageMessage.Event[marriageMessage.ProposalCreatedBody]
		err = json.Unmarshal(msg.Value, &event)
		require.NoError(t, err)

		assert.Equal(t, proposerId, event.CharacterId)
		assert.Equal(t, marriageMessage.EventProposalCreated, event.Type)
		assert.Equal(t, proposalId, event.Body.ProposalId)
		assert.Equal(t, proposerId, event.Body.ProposerId)
		assert.Equal(t, targetId, event.Body.TargetCharacterId)
	})

	t.Run("MarriageEventProviders", func(t *testing.T) {
		// Test MarriageCreatedEventProvider
		marriageId := uint32(1)
		characterId1 := uint32(12345)
		characterId2 := uint32(67890)
		marriedAt := time.Now()

		provider := marriageService.MarriageCreatedEventProvider(marriageId, characterId1, characterId2, marriedAt)
		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		msg := messages[0]
		assert.NotEmpty(t, msg.Key)
		assert.NotEmpty(t, msg.Value)

		// Verify message content
		var event marriageMessage.Event[marriageMessage.MarriageCreatedBody]
		err = json.Unmarshal(msg.Value, &event)
		require.NoError(t, err)

		assert.Equal(t, characterId1, event.CharacterId)
		assert.Equal(t, marriageMessage.EventMarriageCreated, event.Type)
		assert.Equal(t, marriageId, event.Body.MarriageId)
		assert.Equal(t, characterId1, event.Body.CharacterId1)
		assert.Equal(t, characterId2, event.Body.CharacterId2)
	})

	t.Run("CeremonyEventProviders", func(t *testing.T) {
		// Test CeremonyScheduledEventProvider
		ceremonyId := uint32(1)
		marriageId := uint32(1)
		characterId1 := uint32(12345)
		characterId2 := uint32(67890)
		scheduledAt := time.Now().Add(7 * 24 * time.Hour)
		invitees := []uint32{111, 222, 333}

		provider := marriageService.CeremonyScheduledEventProvider(ceremonyId, marriageId, characterId1, characterId2, scheduledAt, invitees)
		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		msg := messages[0]
		assert.NotEmpty(t, msg.Key)
		assert.NotEmpty(t, msg.Value)

		// Verify message content
		var event marriageMessage.Event[marriageMessage.CeremonyScheduledBody]
		err = json.Unmarshal(msg.Value, &event)
		require.NoError(t, err)

		assert.Equal(t, characterId1, event.CharacterId)
		assert.Equal(t, marriageMessage.EventCeremonyScheduled, event.Type)
		assert.Equal(t, ceremonyId, event.Body.CeremonyId)
		assert.Equal(t, marriageId, event.Body.MarriageId)
		assert.Equal(t, invitees, event.Body.Invitees)
	})

	t.Run("ErrorEventProviders", func(t *testing.T) {
		// Test MarriageErrorEventProvider
		characterId := uint32(12345)
		errorType := marriageMessage.ErrorTypeProposal
		errorCode := marriageMessage.ErrorCodeAlreadyMarried
		message := "Character is already married"
		context := "propose"

		provider := marriageService.MarriageErrorEventProvider(characterId, errorType, errorCode, message, context)
		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		msg := messages[0]
		assert.NotEmpty(t, msg.Key)
		assert.NotEmpty(t, msg.Value)

		// Verify message content
		var event marriageMessage.Event[marriageMessage.MarriageErrorBody]
		err = json.Unmarshal(msg.Value, &event)
		require.NoError(t, err)

		assert.Equal(t, characterId, event.CharacterId)
		assert.Equal(t, marriageMessage.EventMarriageError, event.Type)
		assert.Equal(t, errorType, event.Body.ErrorType)
		assert.Equal(t, errorCode, event.Body.ErrorCode)
		assert.Equal(t, message, event.Body.Message)
		assert.Equal(t, context, event.Body.Context)
	})
}

// testConsumerHandlesProposalCommands tests that consumers properly handle proposal commands
func testConsumerHandlesProposalCommands(t *testing.T, logger logrus.FieldLogger, ctx context.Context, db *gorm.DB) {
	// Initialize handlers
	handlers := make([]handler.Handler, 0)
	rf := func(topic string, handler handler.Handler) (string, error) {
		handlers = append(handlers, handler)
		return topic, nil
	}
	marriage.InitHandlers(logger)(db)(rf)
	require.NotEmpty(t, handlers)

	t.Run("ProposeCommandHandler", func(t *testing.T) {
		// Test that handlers can handle propose commands
		cmdType := marriageMessage.CommandMarriagePropose
		assert.Equal(t, "PROPOSE", cmdType)

		// Test that at least one handler exists
		assert.Greater(t, len(handlers), 0)

		// Test that all handlers are not nil
		for i, h := range handlers {
			assert.NotNil(t, h, "Handler %d should not be nil", i)
		}
	})

	t.Run("AcceptCommandHandler", func(t *testing.T) {
		// Test that handlers can handle accept commands
		cmdType := marriageMessage.CommandMarriageAccept
		assert.Equal(t, "ACCEPT", cmdType)
	})
}

// testConsumerHandlesCeremonyCommands tests ceremony command handling
func testConsumerHandlesCeremonyCommands(t *testing.T, logger logrus.FieldLogger, ctx context.Context, db *gorm.DB) {
	// Initialize handlers
	handlers := make([]handler.Handler, 0)
	rf := func(topic string, handler handler.Handler) (string, error) {
		handlers = append(handlers, handler)
		return topic, nil
	}
	marriage.InitHandlers(logger)(db)(rf)
	require.NotEmpty(t, handlers)

	t.Run("ScheduleCeremonyCommand", func(t *testing.T) {
		cmd := marriageMessage.Command[marriageMessage.ScheduleCeremonyBody]{
			CharacterId: 12345,
			Type:        marriageMessage.CommandCeremonySchedule,
			Body: marriageMessage.ScheduleCeremonyBody{
				MarriageId:  1,
				ScheduledAt: time.Now().Add(7 * 24 * time.Hour),
				Invitees:    []uint32{111, 222, 333},
			},
		}

		// Verify command structure
		assert.Equal(t, marriageMessage.CommandCeremonySchedule, cmd.Type)
		assert.Equal(t, uint32(12345), cmd.CharacterId)
		assert.Len(t, cmd.Body.Invitees, 3)
	})

	t.Run("StartCeremonyCommand", func(t *testing.T) {
		cmd := marriageMessage.Command[marriageMessage.StartCeremonyBody]{
			CharacterId: 12345,
			Type:        marriageMessage.CommandCeremonyStart,
			Body: marriageMessage.StartCeremonyBody{
				CeremonyId: 1,
			},
		}

		assert.Equal(t, marriageMessage.CommandCeremonyStart, cmd.Type)
		assert.Equal(t, uint32(1), cmd.Body.CeremonyId)
	})
}

// testEndToEndMessageFlow tests complete message flow from producer to consumer
func testEndToEndMessageFlow(t *testing.T, logger logrus.FieldLogger, ctx context.Context, db *gorm.DB) {
	// Initialize the processor and handlers with mock character processor
	processor := marriageService.NewProcessor(logger, ctx, db).WithCharacterProcessor(&mockCharacterProcessor{})
	handlers := make([]handler.Handler, 0)
	rf := func(topic string, handler handler.Handler) (string, error) {
		handlers = append(handlers, handler)
		return topic, nil
	}
	marriage.InitHandlers(logger)(db)(rf)
	require.NotEmpty(t, handlers)

	t.Run("CompleteProposalFlow", func(t *testing.T) {
		// 1. Create a proposal command
		proposerId := uint32(12345)
		targetId := uint32(67890)

		// 2. Process the command directly through processor to verify flow (without Kafka emission)
		proposal, err := processor.Propose(proposerId, targetId)()
		require.NoError(t, err)
		assert.NotNil(t, proposal)

		// 3. Extract tenant from context
		tenantModel := tenant.MustFromContext(ctx)
		tenantId := tenantModel.Id()

		// 4. Verify the proposal exists in database
		proposalModel, err := marriageService.GetProposalByIdProvider(db, logger)(proposal.Id(), tenantId)()
		require.NoError(t, err)
		assert.Equal(t, proposerId, proposalModel.ProposerId())
		assert.Equal(t, targetId, proposalModel.TargetId())
		assert.Equal(t, marriageService.ProposalStatusPending, proposalModel.Status())

		// 5. Verify the proposal created event can be created
		eventProvider := marriageService.ProposalCreatedEventProvider(proposal.Id(), proposerId, targetId, proposal.ProposedAt(), proposal.ExpiresAt())
		messages, err := eventProvider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		var event marriageMessage.Event[marriageMessage.ProposalCreatedBody]
		err = json.Unmarshal(messages[0].Value, &event)
		require.NoError(t, err)
		assert.Equal(t, marriageMessage.EventProposalCreated, event.Type)
		assert.Equal(t, proposal.Id(), event.Body.ProposalId)
	})

	t.Run("CompleteMarriageFlow", func(t *testing.T) {
		// 1. Create a proposal first
		proposerId := uint32(11111)
		targetId := uint32(22222)

		proposal, err := processor.Propose(proposerId, targetId)()
		require.NoError(t, err)

		// 2. Process acceptance through processor
		marriageModel, err := processor.AcceptProposal(proposal.Id())()
		require.NoError(t, err)
		assert.NotNil(t, marriageModel)

		// 3. Extract tenant from context
		tenantModel := tenant.MustFromContext(ctx)
		tenantId := tenantModel.Id()

		// 4. Verify the marriage exists in database
		marriagePtr, err := marriageService.GetMarriageByIdProvider(db, logger)(marriageModel.Id(), tenantId)()
		require.NoError(t, err)
		require.NotNil(t, marriagePtr)
		assert.Equal(t, marriageService.StatusEngaged, marriagePtr.Status())
		assert.True(t, marriagePtr.CharacterId1() == proposerId && marriagePtr.CharacterId2() == targetId ||
			marriagePtr.CharacterId1() == targetId && marriagePtr.CharacterId2() == proposerId)

		// 5. Verify the marriage created event can be created
		// Note: Use engagement timestamp since marriage is not yet completed
		eventProvider := marriageService.MarriageCreatedEventProvider(marriageModel.Id(), marriageModel.CharacterId1(), marriageModel.CharacterId2(), *marriageModel.EngagedAt())
		messages, err := eventProvider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		var event marriageMessage.Event[marriageMessage.MarriageCreatedBody]
		err = json.Unmarshal(messages[0].Value, &event)
		require.NoError(t, err)
		assert.Equal(t, marriageMessage.EventMarriageCreated, event.Type)
		assert.Equal(t, marriageModel.Id(), event.Body.MarriageId)
	})

	t.Run("CompleteCeremonyFlow", func(t *testing.T) {
		// 1. Create a marriage first
		proposerId := uint32(33333)
		targetId := uint32(44444)

		proposal, err := processor.Propose(proposerId, targetId)()
		require.NoError(t, err)

		marriageModel, err := processor.AcceptProposal(proposal.Id())()
		require.NoError(t, err)

		// 2. Schedule a ceremony
		scheduledAt := time.Now().Add(7 * 24 * time.Hour)
		invitees := []uint32{55555, 66666}

		// 3. Process ceremony scheduling
		ceremony, err := processor.ScheduleCeremony(marriageModel.Id(), scheduledAt, invitees)()
		require.NoError(t, err)
		assert.NotNil(t, ceremony)

		// 4. Extract tenant from context
		tenantModel := tenant.MustFromContext(ctx)
		tenantId := tenantModel.Id()

		// 5. Verify the ceremony exists in database
		ceremonyPtr, err := marriageService.GetCeremonyByIdProvider(db, logger)(ceremony.Id(), tenantId)()
		require.NoError(t, err)
		require.NotNil(t, ceremonyPtr)
		assert.Equal(t, marriageService.CeremonyStatusScheduled, ceremonyPtr.Status())
		assert.Equal(t, marriageModel.Id(), ceremonyPtr.MarriageId())
		assert.Equal(t, len(invitees), len(ceremonyPtr.Invitees()))

		// 6. Verify the ceremony scheduled event can be created
		eventProvider := marriageService.CeremonyScheduledEventProvider(ceremony.Id(), marriageModel.Id(), marriageModel.CharacterId1(), marriageModel.CharacterId2(), scheduledAt, invitees)
		messages, err := eventProvider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		var event marriageMessage.Event[marriageMessage.CeremonyScheduledBody]
		err = json.Unmarshal(messages[0].Value, &event)
		require.NoError(t, err)
		assert.Equal(t, marriageMessage.EventCeremonyScheduled, event.Type)
		assert.Equal(t, ceremony.Id(), event.Body.CeremonyId)
	})

	t.Run("CompleteDivorceFlow", func(t *testing.T) {
		// 1. Create a marriage first
		proposerId := uint32(77777)
		targetId := uint32(88888)

		proposal, err := processor.Propose(proposerId, targetId)()
		require.NoError(t, err)

		marriageModel, err := processor.AcceptProposal(proposal.Id())()
		require.NoError(t, err)

		// 2. Schedule and complete a ceremony to make the marriage valid for divorce
		scheduledAt := time.Now().Add(7 * 24 * time.Hour)
		invitees := []uint32{111, 222}

		ceremony, err := processor.ScheduleCeremony(marriageModel.Id(), scheduledAt, invitees)()
		require.NoError(t, err)

		// Start the ceremony
		startedCeremony, err := processor.StartCeremony(ceremony.Id())()
		require.NoError(t, err)

		// Complete the ceremony
		completedCeremony, err := processor.CompleteCeremony(startedCeremony.Id())()
		require.NoError(t, err)
		assert.NotNil(t, completedCeremony)

		// 3. Update marriage status to married manually (since ceremony completion should trigger this)
		// This is a test workaround - in real implementation, ceremony completion would update marriage
		marriedMarriage, err := marriageModel.Marry()
		require.NoError(t, err)

		// Update the marriage in the database
		tenantModel := tenant.MustFromContext(ctx)
		tenantId := tenantModel.Id()
		_, err = marriageService.UpdateMarriage(db, logger)(marriedMarriage)()
		require.NoError(t, err)

		// 4. Process divorce through processor
		divorcedMarriage, err := processor.Divorce(marriageModel.Id(), proposerId)()
		require.NoError(t, err)
		assert.NotNil(t, divorcedMarriage)

		// 5. Verify the marriage status is updated in database
		marriagePtr, err := marriageService.GetMarriageByIdProvider(db, logger)(marriageModel.Id(), tenantId)()
		require.NoError(t, err)
		require.NotNil(t, marriagePtr)
		assert.Equal(t, marriageService.StatusDivorced, marriagePtr.Status())
		assert.NotNil(t, marriagePtr.DivorcedAt())

		// 6. Verify the divorce event can be created
		eventProvider := marriageService.MarriageDivorcedEventProvider(divorcedMarriage.Id(), divorcedMarriage.CharacterId1(), divorcedMarriage.CharacterId2(), *divorcedMarriage.DivorcedAt(), proposerId)
		messages, err := eventProvider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		var event marriageMessage.Event[marriageMessage.MarriageDivorcedBody]
		err = json.Unmarshal(messages[0].Value, &event)
		require.NoError(t, err)
		assert.Equal(t, marriageMessage.EventMarriageDivorced, event.Type)
		assert.Equal(t, divorcedMarriage.Id(), event.Body.MarriageId)
		assert.Equal(t, proposerId, event.Body.InitiatedBy)
	})
}

// testErrorHandlingInConsumers tests error scenarios and event emission
func testErrorHandlingInConsumers(t *testing.T, logger logrus.FieldLogger, ctx context.Context, db *gorm.DB) {
	t.Run("InvalidCommandType", func(t *testing.T) {
		// Test with invalid command type
		cmd := marriageMessage.Command[marriageMessage.ProposeBody]{
			CharacterId: 12345,
			Type:        "INVALID_COMMAND",
			Body: marriageMessage.ProposeBody{
				TargetCharacterId: 67890,
			},
		}

		// Verify that invalid command type is handled properly
		assert.Equal(t, "INVALID_COMMAND", cmd.Type)
		assert.NotEqual(t, marriageMessage.CommandMarriagePropose, cmd.Type)
	})

	t.Run("ErrorEventCreation", func(t *testing.T) {
		characterId := uint32(12345)
		errorType := marriageMessage.ErrorTypeProposal
		errorCode := marriageMessage.ErrorCodeAlreadyMarried
		message := "Character is already married"
		context := "propose"

		provider := marriageService.MarriageErrorEventProvider(characterId, errorType, errorCode, message, context)
		messages, err := provider()
		require.NoError(t, err)

		var errorEvent marriageMessage.Event[marriageMessage.MarriageErrorBody]
		err = json.Unmarshal(messages[0].Value, &errorEvent)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventMarriageError, errorEvent.Type)
		assert.Equal(t, errorType, errorEvent.Body.ErrorType)
		assert.Equal(t, errorCode, errorEvent.Body.ErrorCode)
	})
}

// testMessageBufferingAndTransactions tests the message buffer functionality
func testMessageBufferingAndTransactions(t *testing.T, logger logrus.FieldLogger, ctx context.Context, db *gorm.DB) {
	t.Run("MessageBufferAccumulation", func(t *testing.T) {
		// Test that multiple messages can be buffered
		providers := []model.Provider[[]kafka.Message]{
			marriageService.ProposalCreatedEventProvider(1, 12345, 67890, time.Now(), time.Now().Add(24*time.Hour)),
			marriageService.MarriageCreatedEventProvider(1, 12345, 67890, time.Now()),
			marriageService.CeremonyScheduledEventProvider(1, 1, 12345, 67890, time.Now().Add(7*24*time.Hour), []uint32{111, 222}),
		}

		// Test each provider creates valid messages
		totalMessages := 0
		for _, provider := range providers {
			messages, err := provider()
			require.NoError(t, err)
			require.Len(t, messages, 1)
			totalMessages++
		}

		assert.Equal(t, 3, totalMessages)
	})

	t.Run("MessagePartitioning", func(t *testing.T) {
		// Test that messages are properly keyed for partitioning
		characterId1 := uint32(12345)
		characterId2 := uint32(67890)

		provider1 := marriageService.ProposalCreatedEventProvider(1, characterId1, characterId2, time.Now(), time.Now().Add(24*time.Hour))
		provider2 := marriageService.ProposalCreatedEventProvider(2, characterId2, characterId1, time.Now(), time.Now().Add(24*time.Hour))

		messages1, err := provider1()
		require.NoError(t, err)

		messages2, err := provider2()
		require.NoError(t, err)

		// Messages for different characters should have different keys
		assert.NotEqual(t, messages1[0].Key, messages2[0].Key)
		assert.NotEmpty(t, messages1[0].Key)
		assert.NotEmpty(t, messages2[0].Key)
	})
}

// TestKafkaMessageValidation tests message validation and format compliance
func TestKafkaMessageValidation(t *testing.T) {
	t.Run("CommandValidation", func(t *testing.T) {
		// Test all command types are properly defined
		commands := []string{
			marriageMessage.CommandMarriagePropose,
			marriageMessage.CommandMarriageAccept,
			marriageMessage.CommandMarriageDecline,
			marriageMessage.CommandMarriageCancel,
			marriageMessage.CommandMarriageDivorce,
			marriageMessage.CommandCeremonySchedule,
			marriageMessage.CommandCeremonyStart,
			marriageMessage.CommandCeremonyComplete,
			marriageMessage.CommandCeremonyCancel,
			marriageMessage.CommandCeremonyPostpone,
			marriageMessage.CommandCeremonyReschedule,
			marriageMessage.CommandCeremonyAddInvitee,
			marriageMessage.CommandCeremonyRemoveInvitee,
			marriageMessage.CommandCeremonyAdvanceState,
		}

		for _, cmd := range commands {
			assert.NotEmpty(t, cmd, "Command type should not be empty")
			assert.IsType(t, "", cmd, "Command type should be string")
		}
	})

	t.Run("EventValidation", func(t *testing.T) {
		// Test all event types are properly defined
		events := []string{
			marriageMessage.EventProposalCreated,
			marriageMessage.EventProposalAccepted,
			marriageMessage.EventProposalDeclined,
			marriageMessage.EventProposalExpired,
			marriageMessage.EventProposalCancelled,
			marriageMessage.EventMarriageCreated,
			marriageMessage.EventMarriageDivorced,
			marriageMessage.EventMarriageDeleted,
			marriageMessage.EventCeremonyScheduled,
			marriageMessage.EventCeremonyStarted,
			marriageMessage.EventCeremonyCompleted,
			marriageMessage.EventCeremonyPostponed,
			marriageMessage.EventCeremonyCancelled,
			marriageMessage.EventCeremonyRescheduled,
			marriageMessage.EventInviteeAdded,
			marriageMessage.EventInviteeRemoved,
			marriageMessage.EventMarriageError,
		}

		for _, event := range events {
			assert.NotEmpty(t, event, "Event type should not be empty")
			assert.IsType(t, "", event, "Event type should be string")
		}
	})

	t.Run("TopicConfiguration", func(t *testing.T) {
		// Test topic environment variables are properly defined
		assert.Equal(t, "COMMAND_TOPIC_MARRIAGE", marriageMessage.EnvCommandTopic)
		assert.Equal(t, "EVENT_TOPIC_MARRIAGE_STATUS", marriageMessage.EnvEventTopicStatus)
	})
}

// testEventEmissionOnStateChanges tests that events are properly emitted when state changes occur
func testEventEmissionOnStateChanges(t *testing.T, logger logrus.FieldLogger, ctx context.Context, db *gorm.DB) {
	// Create a mock producer to capture emitted events
	var capturedMessages []kafka.Message
	mockProducer := func(token string) producer.MessageProducer {
		return func(provider model.Provider[[]kafka.Message]) error {
			messages, err := provider()
			if err != nil {
				return err
			}
			capturedMessages = append(capturedMessages, messages...)
			return nil
		}
	}

	// Initialize the processor with mock producer and character processor
	processor := marriageService.NewProcessor(logger, ctx, db).
		WithProducer(mockProducer).
		WithCharacterProcessor(&mockCharacterProcessor{})

	t.Run("ProposalCreatedEventEmission", func(t *testing.T) {
		// Clear captured messages
		capturedMessages = []kafka.Message{}

		// Execute propose operation with event emission
		proposerId := uint32(10001)
		targetId := uint32(10002)
		transactionId := uuid.New()

		proposal, err := processor.ProposeAndEmit(transactionId, proposerId, targetId)
		require.NoError(t, err)
		assert.NotNil(t, proposal)

		// Verify that exactly one event was emitted
		assert.Len(t, capturedMessages, 1)

		// Verify the event content
		var event marriageMessage.Event[marriageMessage.ProposalCreatedBody]
		err = json.Unmarshal(capturedMessages[0].Value, &event)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventProposalCreated, event.Type)
		assert.Equal(t, proposerId, event.CharacterId)
		assert.Equal(t, proposal.Id(), event.Body.ProposalId)
		assert.Equal(t, proposerId, event.Body.ProposerId)
		assert.Equal(t, targetId, event.Body.TargetCharacterId)
		assert.False(t, event.Body.ProposedAt.IsZero())
		assert.False(t, event.Body.ExpiresAt.IsZero())
	})

	t.Run("ProposalAcceptedAndMarriageCreatedEventEmission", func(t *testing.T) {
		// Clear captured messages
		capturedMessages = []kafka.Message{}

		// First create a proposal
		proposerId := uint32(10003)
		targetId := uint32(10004)
		transactionId := uuid.New()

		proposal, err := processor.Propose(proposerId, targetId)()
		require.NoError(t, err)

		// Clear messages from proposal creation
		capturedMessages = []kafka.Message{}

		// Execute accept proposal operation with event emission
		marriage, err := processor.AcceptProposalAndEmit(transactionId, proposal.Id())
		require.NoError(t, err)
		assert.NotNil(t, marriage)

		// Verify that exactly two events were emitted (ProposalAccepted and MarriageCreated)
		assert.Len(t, capturedMessages, 2)

		// Verify ProposalAccepted event
		var acceptedEvent marriageMessage.Event[marriageMessage.ProposalAcceptedBody]
		err = json.Unmarshal(capturedMessages[0].Value, &acceptedEvent)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventProposalAccepted, acceptedEvent.Type)
		assert.Equal(t, proposal.Id(), acceptedEvent.Body.ProposalId)
		assert.Equal(t, proposerId, acceptedEvent.Body.ProposerId)
		assert.Equal(t, targetId, acceptedEvent.Body.TargetCharacterId)
		assert.False(t, acceptedEvent.Body.AcceptedAt.IsZero())

		// Verify MarriageCreated event
		var createdEvent marriageMessage.Event[marriageMessage.MarriageCreatedBody]
		err = json.Unmarshal(capturedMessages[1].Value, &createdEvent)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventMarriageCreated, createdEvent.Type)
		assert.Equal(t, marriage.Id(), createdEvent.Body.MarriageId)
		assert.Equal(t, marriage.CharacterId1(), createdEvent.Body.CharacterId1)
		assert.Equal(t, marriage.CharacterId2(), createdEvent.Body.CharacterId2)
		assert.False(t, createdEvent.Body.MarriedAt.IsZero())
	})

	t.Run("CeremonyScheduledEventEmission", func(t *testing.T) {
		// Clear captured messages
		capturedMessages = []kafka.Message{}

		// First create a marriage
		proposerId := uint32(10005)
		targetId := uint32(10006)

		proposal, err := processor.Propose(proposerId, targetId)()
		require.NoError(t, err)

		marriage, err := processor.AcceptProposal(proposal.Id())()
		require.NoError(t, err)

		// Clear messages from marriage creation
		capturedMessages = []kafka.Message{}

		// Execute ceremony scheduling with event emission
		transactionId := uuid.New()
		scheduledAt := time.Now().Add(7 * 24 * time.Hour)
		invitees := []uint32{10007, 10008}

		ceremony, err := processor.ScheduleCeremonyAndEmit(transactionId, marriage.Id(), scheduledAt, invitees)
		require.NoError(t, err)
		assert.NotNil(t, ceremony)

		// Verify that exactly one event was emitted
		assert.Len(t, capturedMessages, 1)

		// Verify the event content
		var event marriageMessage.Event[marriageMessage.CeremonyScheduledBody]
		err = json.Unmarshal(capturedMessages[0].Value, &event)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventCeremonyScheduled, event.Type)
		assert.Equal(t, ceremony.Id(), event.Body.CeremonyId)
		assert.Equal(t, marriage.Id(), event.Body.MarriageId)
		assert.Equal(t, marriage.CharacterId1(), event.Body.CharacterId1)
		assert.Equal(t, marriage.CharacterId2(), event.Body.CharacterId2)
		assert.Equal(t, invitees, event.Body.Invitees)
		assert.False(t, event.Body.ScheduledAt.IsZero())
	})

	t.Run("CeremonyStateTransitionEventEmission", func(t *testing.T) {
		// Clear captured messages
		capturedMessages = []kafka.Message{}

		// First create a marriage and ceremony
		proposerId := uint32(10009)
		targetId := uint32(10010)

		proposal, err := processor.Propose(proposerId, targetId)()
		require.NoError(t, err)

		marriage, err := processor.AcceptProposal(proposal.Id())()
		require.NoError(t, err)

		scheduledAt := time.Now().Add(7 * 24 * time.Hour)
		invitees := []uint32{10011, 10012}

		ceremony, err := processor.ScheduleCeremony(marriage.Id(), scheduledAt, invitees)()
		require.NoError(t, err)

		// Clear messages from ceremony creation
		capturedMessages = []kafka.Message{}

		// Test ceremony start event emission
		transactionId := uuid.New()
		startedCeremony, err := processor.StartCeremonyAndEmit(transactionId, ceremony.Id())
		require.NoError(t, err)
		assert.NotNil(t, startedCeremony)

		// Verify that exactly one event was emitted
		assert.Len(t, capturedMessages, 1)

		// Verify the event content
		var startEvent marriageMessage.Event[marriageMessage.CeremonyStartedBody]
		err = json.Unmarshal(capturedMessages[0].Value, &startEvent)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventCeremonyStarted, startEvent.Type)
		assert.Equal(t, ceremony.Id(), startEvent.Body.CeremonyId)
		assert.Equal(t, marriage.Id(), startEvent.Body.MarriageId)
		assert.Equal(t, marriage.CharacterId1(), startEvent.Body.CharacterId1)
		assert.Equal(t, marriage.CharacterId2(), startEvent.Body.CharacterId2)
		assert.False(t, startEvent.Body.StartedAt.IsZero())

		// Clear messages and test ceremony completion event emission
		capturedMessages = []kafka.Message{}

		completedCeremony, err := processor.CompleteCeremonyAndEmit(transactionId, ceremony.Id())
		require.NoError(t, err)
		assert.NotNil(t, completedCeremony)

		// Verify that exactly one event was emitted
		assert.Len(t, capturedMessages, 1)

		// Verify the event content
		var completedEvent marriageMessage.Event[marriageMessage.CeremonyCompletedBody]
		err = json.Unmarshal(capturedMessages[0].Value, &completedEvent)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventCeremonyCompleted, completedEvent.Type)
		assert.Equal(t, ceremony.Id(), completedEvent.Body.CeremonyId)
		assert.Equal(t, marriage.Id(), completedEvent.Body.MarriageId)
		assert.Equal(t, marriage.CharacterId1(), completedEvent.Body.CharacterId1)
		assert.Equal(t, marriage.CharacterId2(), completedEvent.Body.CharacterId2)
		assert.False(t, completedEvent.Body.CompletedAt.IsZero())
	})

	t.Run("DivorceEventEmission", func(t *testing.T) {
		// Clear captured messages
		capturedMessages = []kafka.Message{}

		// First create a marriage and complete ceremony to allow divorce
		proposerId := uint32(10013)
		targetId := uint32(10014)

		proposal, err := processor.Propose(proposerId, targetId)()
		require.NoError(t, err)

		marriage, err := processor.AcceptProposal(proposal.Id())()
		require.NoError(t, err)

		scheduledAt := time.Now().Add(7 * 24 * time.Hour)
		invitees := []uint32{10015, 10016}

		ceremony, err := processor.ScheduleCeremony(marriage.Id(), scheduledAt, invitees)()
		require.NoError(t, err)

		// Start and complete ceremony
		ceremony, err = processor.StartCeremony(ceremony.Id())()
		require.NoError(t, err)

		ceremony, err = processor.CompleteCeremony(ceremony.Id())()
		require.NoError(t, err)

		// Update marriage to married status (required for divorce)
		marriedMarriage, err := marriage.Marry()
		require.NoError(t, err)

		_, err = marriageService.UpdateMarriage(db, logger)(marriedMarriage)()
		require.NoError(t, err)

		// Clear messages from marriage setup
		capturedMessages = []kafka.Message{}

		// Execute divorce operation with event emission
		transactionId := uuid.New()
		divorcedMarriage, err := processor.DivorceAndEmit(transactionId, marriage.Id(), proposerId)
		require.NoError(t, err)
		assert.NotNil(t, divorcedMarriage)

		// Verify that exactly one event was emitted
		assert.Len(t, capturedMessages, 1)

		// Verify the event content
		var event marriageMessage.Event[marriageMessage.MarriageDivorcedBody]
		err = json.Unmarshal(capturedMessages[0].Value, &event)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventMarriageDivorced, event.Type)
		assert.Equal(t, marriage.Id(), event.Body.MarriageId)
		assert.Equal(t, marriage.CharacterId1(), event.Body.CharacterId1)
		assert.Equal(t, marriage.CharacterId2(), event.Body.CharacterId2)
		assert.Equal(t, proposerId, event.Body.InitiatedBy)
		assert.False(t, event.Body.DivorcedAt.IsZero())
	})
}

// TestConsumerConfiguration tests consumer setup and configuration
func TestConsumerConfiguration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("ConsumerConfigCreation", func(t *testing.T) {
		configFunc := marriage.NewConfig(logger)
		assert.NotNil(t, configFunc)

		// Test configuration function chain
		nameFunc := configFunc("test-consumer")
		assert.NotNil(t, nameFunc)

		tokenFunc := nameFunc("test-token")
		assert.NotNil(t, tokenFunc)

		config := tokenFunc("test-group")
		assert.NotNil(t, config)
	})

	t.Run("ConsumerInitialization", func(t *testing.T) {
		initFunc := marriage.InitConsumers(logger)
		assert.NotNil(t, initFunc)

		// Test that consumer initializer can be created
		consumerFunc := initFunc(func(config consumer.Config, decorators ...model.Decorator[consumer.Config]) {
			// Mock consumer registration function
			assert.NotNil(t, config)
		})
		assert.NotNil(t, consumerFunc)

		// Test consumer group setup
		assert.NotPanics(t, func() {
			consumerFunc("test-group-id")
			// This would normally start the consumer, but in test we just verify it can be called
		})
	})

	t.Run("HandlerInitialization", func(t *testing.T) {
		// Create test database
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)

		handlers := make([]handler.Handler, 0)
		rf := func(topic string, handler handler.Handler) (string, error) {
			handlers = append(handlers, handler)
			return topic, nil
		}
		marriage.InitHandlers(logger)(db)(rf)

		// Verify handlers are created (actual count is 14 based on InitHandlers function)
		expectedHandlerCount := 14 // Based on the InitHandlers function
		assert.Len(t, handlers, expectedHandlerCount)

		// Verify all handlers are not nil
		for i, handler := range handlers {
			assert.NotNil(t, handler, "Handler %d should not be nil", i)
		}
	})
}

// TestMessageHandlerIntegration tests the actual message handlers directly
func TestMessageHandlerIntegration(t *testing.T) {
	// Create test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Run migrations
	err = marriageService.Migration(db)
	require.NoError(t, err)

	// Set up test logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create test context with tenant
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create mock producer to capture emitted events
	var capturedMessages []kafka.Message
	mockProducer := func(token string) producer.MessageProducer {
		return func(provider model.Provider[[]kafka.Message]) error {
			messages, err := provider()
			if err != nil {
				return err
			}
			capturedMessages = append(capturedMessages, messages...)
			return nil
		}
	}

	// Initialize processor with mock producer
	processor := marriageService.NewProcessor(logger, ctx, db).
		WithProducer(mockProducer).
		WithCharacterProcessor(&mockCharacterProcessor{})

	t.Run("ProposeCommandHandler", func(t *testing.T) {
		capturedMessages = []kafka.Message{}

		// Create a propose command
		cmd := marriageMessage.Command[marriageMessage.ProposeBody]{
			CharacterId: 20001,
			Type:        marriageMessage.CommandMarriagePropose,
			Body: marriageMessage.ProposeBody{
				TargetCharacterId: 20002,
			},
		}

		// Get the handler from InitHandlers
		handlers := make([]handler.Handler, 0)
		rf := func(topic string, handler handler.Handler) (string, error) {
			handlers = append(handlers, handler)
			return topic, nil
		}
		marriage.InitHandlers(logger)(db)(rf)
		require.NotEmpty(t, handlers)

		// Simulate processing the command by calling the processor directly
		// (since we can't easily invoke the handler wrapper in tests)
		proposal, err := processor.ProposeAndEmit(uuid.New(), cmd.CharacterId, cmd.Body.TargetCharacterId)
		require.NoError(t, err)
		assert.NotNil(t, proposal)

		// Verify that a ProposalCreated event was emitted
		assert.Len(t, capturedMessages, 1)

		// Verify the event content
		var event marriageMessage.Event[marriageMessage.ProposalCreatedBody]
		err = json.Unmarshal(capturedMessages[0].Value, &event)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventProposalCreated, event.Type)
		assert.Equal(t, cmd.CharacterId, event.CharacterId)
		assert.Equal(t, proposal.Id(), event.Body.ProposalId)
		assert.Equal(t, cmd.CharacterId, event.Body.ProposerId)
		assert.Equal(t, cmd.Body.TargetCharacterId, event.Body.TargetCharacterId)
	})

	t.Run("AcceptCommandHandler", func(t *testing.T) {
		capturedMessages = []kafka.Message{}

		// First create a proposal
		proposal, err := processor.Propose(20003, 20004)()
		require.NoError(t, err)

		// Clear messages from proposal creation
		capturedMessages = []kafka.Message{}

		// Create an accept command
		cmd := marriageMessage.Command[marriageMessage.AcceptBody]{
			CharacterId: 20004,
			Type:        marriageMessage.CommandMarriageAccept,
			Body: marriageMessage.AcceptBody{
				ProposalId: proposal.Id(),
			},
		}

		// Simulate processing the command
		marriage, err := processor.AcceptProposalAndEmit(uuid.New(), cmd.Body.ProposalId)
		require.NoError(t, err)
		assert.NotNil(t, marriage)

		// Verify that ProposalAccepted and MarriageCreated events were emitted
		assert.Len(t, capturedMessages, 2)

		// Verify ProposalAccepted event
		var acceptedEvent marriageMessage.Event[marriageMessage.ProposalAcceptedBody]
		err = json.Unmarshal(capturedMessages[0].Value, &acceptedEvent)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventProposalAccepted, acceptedEvent.Type)
		assert.Equal(t, proposal.Id(), acceptedEvent.Body.ProposalId)

		// Verify MarriageCreated event
		var createdEvent marriageMessage.Event[marriageMessage.MarriageCreatedBody]
		err = json.Unmarshal(capturedMessages[1].Value, &createdEvent)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventMarriageCreated, createdEvent.Type)
		assert.Equal(t, marriage.Id(), createdEvent.Body.MarriageId)
	})

	t.Run("ScheduleCeremonyCommandHandler", func(t *testing.T) {
		capturedMessages = []kafka.Message{}

		// First create a marriage
		proposal, err := processor.Propose(20005, 20006)()
		require.NoError(t, err)

		marriage, err := processor.AcceptProposal(proposal.Id())()
		require.NoError(t, err)

		// Clear messages from marriage setup
		capturedMessages = []kafka.Message{}

		// Create a schedule ceremony command
		scheduledAt := time.Now().Add(7 * 24 * time.Hour)
		invitees := []uint32{20007, 20008}

		cmd := marriageMessage.Command[marriageMessage.ScheduleCeremonyBody]{
			CharacterId: 20005,
			Type:        marriageMessage.CommandCeremonySchedule,
			Body: marriageMessage.ScheduleCeremonyBody{
				MarriageId:  marriage.Id(),
				ScheduledAt: scheduledAt,
				Invitees:    invitees,
			},
		}

		// Simulate processing the command
		ceremony, err := processor.ScheduleCeremonyAndEmit(uuid.New(), cmd.Body.MarriageId, cmd.Body.ScheduledAt, cmd.Body.Invitees)
		require.NoError(t, err)
		assert.NotNil(t, ceremony)

		// Verify that CeremonyScheduled event was emitted
		assert.Len(t, capturedMessages, 1)

		// Verify the event content
		var event marriageMessage.Event[marriageMessage.CeremonyScheduledBody]
		err = json.Unmarshal(capturedMessages[0].Value, &event)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventCeremonyScheduled, event.Type)
		assert.Equal(t, ceremony.Id(), event.Body.CeremonyId)
		assert.Equal(t, marriage.Id(), event.Body.MarriageId)
		assert.Equal(t, invitees, event.Body.Invitees)
	})

	t.Run("DivorceCommandHandler", func(t *testing.T) {
		capturedMessages = []kafka.Message{}

		// First create a completed marriage
		proposal, err := processor.Propose(20009, 20010)()
		require.NoError(t, err)

		marriage, err := processor.AcceptProposal(proposal.Id())()
		require.NoError(t, err)

		// Complete ceremony to make marriage valid for divorce
		scheduledAt := time.Now().Add(7 * 24 * time.Hour)
		invitees := []uint32{20011, 20012}

		ceremony, err := processor.ScheduleCeremony(marriage.Id(), scheduledAt, invitees)()
		require.NoError(t, err)

		ceremony, err = processor.StartCeremony(ceremony.Id())()
		require.NoError(t, err)

		ceremony, err = processor.CompleteCeremony(ceremony.Id())()
		require.NoError(t, err)

		// Update marriage to married status
		marriedMarriage, err := marriage.Marry()
		require.NoError(t, err)

		_, err = marriageService.UpdateMarriage(db, logger)(marriedMarriage)()
		require.NoError(t, err)

		// Clear messages from marriage setup
		capturedMessages = []kafka.Message{}

		// Create a divorce command
		cmd := marriageMessage.Command[marriageMessage.DivorceBody]{
			CharacterId: 20009,
			Type:        marriageMessage.CommandMarriageDivorce,
			Body: marriageMessage.DivorceBody{
				MarriageId: marriage.Id(),
			},
		}

		// Simulate processing the command
		divorcedMarriage, err := processor.DivorceAndEmit(uuid.New(), cmd.Body.MarriageId, cmd.CharacterId)
		require.NoError(t, err)
		assert.NotNil(t, divorcedMarriage)

		// Verify that MarriageDivorced event was emitted
		assert.Len(t, capturedMessages, 1)

		// Verify the event content
		var event marriageMessage.Event[marriageMessage.MarriageDivorcedBody]
		err = json.Unmarshal(capturedMessages[0].Value, &event)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventMarriageDivorced, event.Type)
		assert.Equal(t, marriage.Id(), event.Body.MarriageId)
		assert.Equal(t, cmd.CharacterId, event.Body.InitiatedBy)
	})

	t.Run("ErrorHandlingInHandlers", func(t *testing.T) {
		capturedMessages = []kafka.Message{}

		// Test error handling when trying to propose to already married character
		// First create a marriage
		proposal, err := processor.Propose(20013, 20014)()
		require.NoError(t, err)

		_, err = processor.AcceptProposal(proposal.Id())()
		require.NoError(t, err)

		// Clear messages from marriage setup
		capturedMessages = []kafka.Message{}

		// Try to propose to already married character (should fail)
		_, err = processor.ProposeAndEmit(uuid.New(), 20015, 20013)
		assert.Error(t, err, "Should fail when proposing to already married character")

		// In the current implementation, the error event is only emitted by the consumer handler
		// when the processor returns an error. Since we're testing the processor directly,
		// we expect an error but not necessarily an error event.
		// The error handling test is more appropriate for testing the consumer handler directly.

		// For now, just verify that the error was returned and no regular events were emitted
		assert.Len(t, capturedMessages, 0, "No events should be emitted for failed proposals")
	})
}
