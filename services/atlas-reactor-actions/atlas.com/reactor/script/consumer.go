package script

import (
	"context"

	consumer2 "atlas-reactor-actions/kafka/consumer"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// commandEvent represents a Kafka command message
type commandEvent[E any] struct {
	WorldId       byte   `json:"worldId"`
	ChannelId     byte   `json:"channelId"`
	MapId         uint32 `json:"mapId"`
	ReactorId     uint32 `json:"reactorId"`
	Classification string `json:"classification"`
	ReactorName   string `json:"reactorName"`
	ReactorState  int8   `json:"reactorState"`
	X             int16  `json:"x"`
	Y             int16  `json:"y"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

// hitBody represents the body of a hit reactor command
type hitBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
	IsSkill     bool   `json:"isSkill"`
}

// triggerBody represents the body of a trigger reactor command
type triggerBody struct {
	CharacterId uint32 `json:"characterId"`
}

// InitConsumers initializes Kafka consumers for reactor actions
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
		return func(groupId string) {
			rf(
				consumer2.NewConfig(l)("reactor_actions_command")(EnvCommandTopic)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
			)
		}
	}
}

// InitHandlers initializes Kafka message handlers
func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandFunc(l, db))))
	}
}

// handleCommandFunc returns a handler function for reactor commands
func handleCommandFunc(l logrus.FieldLogger, db *gorm.DB) func(logrus.FieldLogger, context.Context, commandEvent[interface{}]) {
	return func(fl logrus.FieldLogger, ctx context.Context, command commandEvent[interface{}]) {
		switch command.Type {
		case CommandTypeHit:
			handleHitCommand(l, ctx, db, command)
		case CommandTypeTrigger:
			handleTriggerCommand(l, ctx, db, command)
		default:
			l.Warnf("Unknown command type: %s", command.Type)
		}
	}
}

// handleHitCommand handles a reactor hit command
func handleHitCommand(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, command commandEvent[interface{}]) {
	l.Debugf("Received reactor hit command for reactor [%s] in map [%d]", command.Classification, command.MapId)

	// Extract body data
	body, ok := command.Body.(map[string]interface{})
	if !ok {
		l.Warnf("Invalid hit command body format")
		return
	}

	characterId := uint32(0)
	if v, ok := body["characterId"].(float64); ok {
		characterId = uint32(v)
	}

	// Create processor with tenant context from Kafka message
	processor := NewProcessor(l, ctx, db)

	// Process the reactor hit script
	result := processor.ProcessHit(command.Classification, command.ReactorState, characterId)

	if result.Error != nil {
		l.WithError(result.Error).Errorf("Failed to process reactor hit script [%s] for character [%d]",
			command.Classification, characterId)
		return
	}

	l.Debugf("Reactor hit script [%s] result: matchedRule=%s, operations=%d",
		command.Classification, result.MatchedRule, len(result.Operations))

	// Execute operations if any
	if len(result.Operations) > 0 {
		executeOperations(l, ctx, command, characterId, result)
	}
}

// handleTriggerCommand handles a reactor trigger command
func handleTriggerCommand(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, command commandEvent[interface{}]) {
	l.Debugf("Received reactor trigger command for reactor [%s] in map [%d]", command.Classification, command.MapId)

	// Extract body data
	body, ok := command.Body.(map[string]interface{})
	if !ok {
		l.Warnf("Invalid trigger command body format")
		return
	}

	characterId := uint32(0)
	if v, ok := body["characterId"].(float64); ok {
		characterId = uint32(v)
	}

	// Create processor with tenant context from Kafka message
	processor := NewProcessor(l, ctx, db)

	// Process the reactor trigger script
	result := processor.ProcessTrigger(command.Classification, command.ReactorState, characterId)

	if result.Error != nil {
		l.WithError(result.Error).Errorf("Failed to process reactor trigger script [%s] for character [%d]",
			command.Classification, characterId)
		return
	}

	l.Debugf("Reactor trigger script [%s] result: matchedRule=%s, operations=%d",
		command.Classification, result.MatchedRule, len(result.Operations))

	// Execute operations if any
	if len(result.Operations) > 0 {
		executeOperations(l, ctx, command, characterId, result)
	}
}

// executeOperations executes the operations from a matched rule
func executeOperations(l logrus.FieldLogger, ctx context.Context, command commandEvent[interface{}], characterId uint32, result ProcessResult) {
	// Build reactor context
	rc := ReactorContext{
		WorldId:        world.Id(command.WorldId),
		ChannelId:      channel.Id(command.ChannelId),
		MapId:          command.MapId,
		ReactorId:      command.ReactorId,
		Classification: command.Classification,
		ReactorName:    command.ReactorName,
		X:              command.X,
		Y:              command.Y,
	}

	// Create executor
	executor := NewOperationExecutor(l, ctx)

	// Execute all operations
	if err := executor.ExecuteOperations(rc, characterId, result.Operations); err != nil {
		l.WithError(err).Errorf("Failed to execute operations for reactor [%s]", command.Classification)
	}
}
