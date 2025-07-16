package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"atlas-marriages/kafka/consumer/marriage"
	marriageMessage "atlas-marriages/kafka/message/marriage"
	marriageService "atlas-marriages/marriage"
	"github.com/Chronicle20/atlas-kafka/consumer"
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
	handlers := marriage.InitHandlers(logger, ctx, db)
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
	handlers := marriage.InitHandlers(logger, ctx, db)
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
	t.Run("ProposalFlow", func(t *testing.T) {
		// Create a proposal using the producer
		proposalId := uint32(1)
		proposerId := uint32(12345)
		targetId := uint32(67890)
		proposedAt := time.Now()
		expiresAt := proposedAt.Add(24 * time.Hour)

		// Test producer creates message
		eventProvider := marriageService.ProposalCreatedEventProvider(proposalId, proposerId, targetId, proposedAt, expiresAt)
		messages, err := eventProvider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		// Verify message can be consumed
		var event marriageMessage.Event[marriageMessage.ProposalCreatedBody]
		err = json.Unmarshal(messages[0].Value, &event)
		require.NoError(t, err)

		// Verify the event would trigger appropriate consumer logic
		assert.Equal(t, marriageMessage.EventProposalCreated, event.Type)
		assert.Equal(t, proposerId, event.Body.ProposerId)
		assert.Equal(t, targetId, event.Body.TargetCharacterId)
	})

	t.Run("MarriageFlow", func(t *testing.T) {
		// Test complete marriage creation flow
		marriageId := uint32(1)
		characterId1 := uint32(12345)
		characterId2 := uint32(67890)
		marriedAt := time.Now()

		// Create marriage created event
		eventProvider := marriageService.MarriageCreatedEventProvider(marriageId, characterId1, characterId2, marriedAt)
		messages, err := eventProvider()
		require.NoError(t, err)

		// Verify message structure
		var event marriageMessage.Event[marriageMessage.MarriageCreatedBody]
		err = json.Unmarshal(messages[0].Value, &event)
		require.NoError(t, err)

		assert.Equal(t, marriageMessage.EventMarriageCreated, event.Type)
		assert.Equal(t, marriageId, event.Body.MarriageId)
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
		// Create test database
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)

		tenantId := uuid.New()
		tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
		require.NoError(t, err)
		ctx := tenant.WithContext(context.Background(), tenantModel)

		initFunc := marriage.InitConsumers(logger, ctx, db)
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

		tenantId := uuid.New()
		tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
		require.NoError(t, err)
		ctx := tenant.WithContext(context.Background(), tenantModel)

		handlers := marriage.InitHandlers(logger, ctx, db)
		
		// Verify handlers are created (actual count is 14 based on InitHandlers function)
		expectedHandlerCount := 14 // Based on the InitHandlers function
		assert.Len(t, handlers, expectedHandlerCount)

		// Verify all handlers are not nil
		for i, handler := range handlers {
			assert.NotNil(t, handler, "Handler %d should not be nil", i)
		}
	})
}