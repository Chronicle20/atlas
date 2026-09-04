package _map

import (
	monsterData "atlas-maps/data/map/monster"
	"atlas-maps/kafka/message"
	mapKafka "atlas-maps/kafka/message/map"
	monsterKafka "atlas-maps/kafka/message/monster"
	"atlas-maps/map/character"
	"atlas-maps/map/environment"
	monster2 "atlas-maps/map/monster"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()

	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	monster2.InitRegistry(rc)

	os.Exit(m.Run())
}

type mockCharacterProcessor struct {
	mu                        sync.Mutex
	enterCalls                []enterCall
	exitCalls                 []exitCall
	getCharactersInMapFunc    func(transactionId uuid.UUID, f field.Model) ([]uint32, error)
	getMapsWithCharactersFunc func() []character.MapKey
}

type enterCall struct {
	transactionId uuid.UUID
	f             field.Model
	characterId   uint32
}

type exitCall struct {
	transactionId uuid.UUID
	f             field.Model
	characterId   uint32
}

func (m *mockCharacterProcessor) GetCharactersInMap(transactionId uuid.UUID, f field.Model) ([]uint32, error) {
	if m.getCharactersInMapFunc != nil {
		return m.getCharactersInMapFunc(transactionId, f)
	}
	return nil, nil
}

func (m *mockCharacterProcessor) GetCharactersInMapAllInstances(_ uuid.UUID, _ world.Id, _ channel.Id, _ _map.Id) ([]uint32, error) {
	return nil, nil
}

func (m *mockCharacterProcessor) GetMapsWithCharacters() []character.MapKey {
	if m.getMapsWithCharactersFunc != nil {
		return m.getMapsWithCharactersFunc()
	}
	return nil
}

func (m *mockCharacterProcessor) GetFieldsWithCharacters(_ tenant.Model) []character.FieldOccupancy {
	return nil
}

func (m *mockCharacterProcessor) Enter(transactionId uuid.UUID, f field.Model, characterId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enterCalls = append(m.enterCalls, enterCall{
		transactionId: transactionId,
		f:             f,
		characterId:   characterId,
	})
}

func (m *mockCharacterProcessor) Exit(transactionId uuid.UUID, f field.Model, characterId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exitCalls = append(m.exitCalls, exitCall{
		transactionId: transactionId,
		f:             f,
		characterId:   characterId,
	})
}

func (m *mockCharacterProcessor) ExitAll(_ uint32) []character.MapKey {
	return nil
}

func (m *mockCharacterProcessor) GetEnterCalls() []enterCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]enterCall(nil), m.enterCalls...)
}

func (m *mockCharacterProcessor) GetExitCalls() []exitCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]exitCall(nil), m.exitCalls...)
}

type mockProducerProvider struct {
	mu       sync.Mutex
	messages map[topic.Token][]kafka.Message
}

func newMockProducerProvider() *mockProducerProvider {
	return &mockProducerProvider{
		messages: make(map[topic.Token][]kafka.Message),
	}
}

func (m *mockProducerProvider) Provider() producer.Provider {
	return func(token topic.Token) producer.MessageProducer {
		return func(provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				return err
			}
			m.mu.Lock()
			defer m.mu.Unlock()
			m.messages[token] = append(m.messages[token], msgs...)
			return nil
		}
	}
}

func (m *mockProducerProvider) GetMessages(t topic.Token) []kafka.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]kafka.Message(nil), m.messages[t]...)
}

func createTestContext() context.Context {
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(ctx, te)
}

func createTestProcessor(l logrus.FieldLogger, ctx context.Context, cp character.Processor, pp *mockProducerProvider) *ProcessorImpl {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		p:   pp.Provider(),
		cp:  cp,
		db:  nil,
	}
}

func TestProcessorImpl_Enter(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	mockCp := &mockCharacterProcessor{}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	instance := uuid.Nil
	characterId := uint32(12345)
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	buf := message.NewBuffer()
	err := p.Enter(buf)(transactionId, f, characterId)
	if err != nil {
		t.Fatalf("Enter returned error: %v", err)
	}

	// Verify character processor Enter was called
	enterCalls := mockCp.GetEnterCalls()
	if len(enterCalls) != 1 {
		t.Fatalf("Expected 1 Enter call, got %d", len(enterCalls))
	}

	call := enterCalls[0]
	if call.transactionId != transactionId {
		t.Errorf("Expected transactionId %v, got %v", transactionId, call.transactionId)
	}
	if call.f.WorldId() != worldId {
		t.Errorf("Expected worldId %v, got %v", worldId, call.f.WorldId())
	}
	if call.f.ChannelId() != channelId {
		t.Errorf("Expected channelId %v, got %v", channelId, call.f.ChannelId())
	}
	if call.f.MapId() != mapId {
		t.Errorf("Expected mapId %v, got %v", mapId, call.f.MapId())
	}
	if call.f.Instance() != instance {
		t.Errorf("Expected instance %v, got %v", instance, call.f.Instance())
	}
	if call.characterId != characterId {
		t.Errorf("Expected characterId %v, got %v", characterId, call.characterId)
	}

	// Verify message was buffered
	messages := buf.GetAll()
	if len(messages[mapKafka.EnvEventTopicMapStatus]) != 1 {
		t.Fatalf("Expected 1 message in buffer, got %d", len(messages[mapKafka.EnvEventTopicMapStatus]))
	}

	// Verify message content
	msg := messages[mapKafka.EnvEventTopicMapStatus][0]
	var event mapKafka.StatusEvent[mapKafka.CharacterEnter]
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.TransactionId != transactionId {
		t.Errorf("Expected event transactionId %v, got %v", transactionId, event.TransactionId)
	}
	if event.WorldId != worldId {
		t.Errorf("Expected event worldId %v, got %v", worldId, event.WorldId)
	}
	if event.ChannelId != channelId {
		t.Errorf("Expected event channelId %v, got %v", channelId, event.ChannelId)
	}
	if event.MapId != mapId {
		t.Errorf("Expected event mapId %v, got %v", mapId, event.MapId)
	}
	if event.Instance != instance {
		t.Errorf("Expected event instance %v, got %v", instance, event.Instance)
	}
	if event.Type != mapKafka.EventTopicMapStatusTypeCharacterEnter {
		t.Errorf("Expected event type %v, got %v", mapKafka.EventTopicMapStatusTypeCharacterEnter, event.Type)
	}
	if event.Body.CharacterId != characterId {
		t.Errorf("Expected event characterId %v, got %v", characterId, event.Body.CharacterId)
	}
}

func TestProcessorImpl_EnterAndEmit(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	mockCp := &mockCharacterProcessor{}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	instance := uuid.Nil
	characterId := uint32(12345)
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	err := p.EnterAndEmit(transactionId, f, characterId)
	if err != nil {
		t.Fatalf("EnterAndEmit returned error: %v", err)
	}

	// Verify character processor Enter was called
	enterCalls := mockCp.GetEnterCalls()
	if len(enterCalls) != 1 {
		t.Fatalf("Expected 1 Enter call, got %d", len(enterCalls))
	}

	// Verify message was emitted via producer
	messages := mockPp.GetMessages(mapKafka.EnvEventTopicMapStatus)
	if len(messages) != 1 {
		t.Fatalf("Expected 1 emitted message, got %d", len(messages))
	}

	// Verify message content
	var event mapKafka.StatusEvent[mapKafka.CharacterEnter]
	if err := json.Unmarshal(messages[0].Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Type != mapKafka.EventTopicMapStatusTypeCharacterEnter {
		t.Errorf("Expected event type %v, got %v", mapKafka.EventTopicMapStatusTypeCharacterEnter, event.Type)
	}
	if event.Body.CharacterId != characterId {
		t.Errorf("Expected event characterId %v, got %v", characterId, event.Body.CharacterId)
	}
}

func TestProcessorImpl_Exit(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	mockCp := &mockCharacterProcessor{}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	instance := uuid.Nil
	characterId := uint32(12345)
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	buf := message.NewBuffer()
	err := p.Exit(buf)(transactionId, f, characterId)
	if err != nil {
		t.Fatalf("Exit returned error: %v", err)
	}

	// Verify character processor Exit was called
	exitCalls := mockCp.GetExitCalls()
	if len(exitCalls) != 1 {
		t.Fatalf("Expected 1 Exit call, got %d", len(exitCalls))
	}

	call := exitCalls[0]
	if call.transactionId != transactionId {
		t.Errorf("Expected transactionId %v, got %v", transactionId, call.transactionId)
	}
	if call.f.WorldId() != worldId {
		t.Errorf("Expected worldId %v, got %v", worldId, call.f.WorldId())
	}
	if call.f.ChannelId() != channelId {
		t.Errorf("Expected channelId %v, got %v", channelId, call.f.ChannelId())
	}
	if call.f.MapId() != mapId {
		t.Errorf("Expected mapId %v, got %v", mapId, call.f.MapId())
	}
	if call.f.Instance() != instance {
		t.Errorf("Expected instance %v, got %v", instance, call.f.Instance())
	}
	if call.characterId != characterId {
		t.Errorf("Expected characterId %v, got %v", characterId, call.characterId)
	}

	// Verify message was buffered
	messages := buf.GetAll()
	if len(messages[mapKafka.EnvEventTopicMapStatus]) != 1 {
		t.Fatalf("Expected 1 message in buffer, got %d", len(messages[mapKafka.EnvEventTopicMapStatus]))
	}

	// Verify message content
	msg := messages[mapKafka.EnvEventTopicMapStatus][0]
	var event mapKafka.StatusEvent[mapKafka.CharacterExit]
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Type != mapKafka.EventTopicMapStatusTypeCharacterExit {
		t.Errorf("Expected event type %v, got %v", mapKafka.EventTopicMapStatusTypeCharacterExit, event.Type)
	}
	if event.Body.CharacterId != characterId {
		t.Errorf("Expected event characterId %v, got %v", characterId, event.Body.CharacterId)
	}
}

func TestProcessorImpl_ExitAndEmit(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	mockCp := &mockCharacterProcessor{}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	instance := uuid.Nil
	characterId := uint32(12345)
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	err := p.ExitAndEmit(transactionId, f, characterId)
	if err != nil {
		t.Fatalf("ExitAndEmit returned error: %v", err)
	}

	// Verify character processor Exit was called
	exitCalls := mockCp.GetExitCalls()
	if len(exitCalls) != 1 {
		t.Fatalf("Expected 1 Exit call, got %d", len(exitCalls))
	}

	// Verify message was emitted via producer
	messages := mockPp.GetMessages(mapKafka.EnvEventTopicMapStatus)
	if len(messages) != 1 {
		t.Fatalf("Expected 1 emitted message, got %d", len(messages))
	}

	// Verify message content
	var event mapKafka.StatusEvent[mapKafka.CharacterExit]
	if err := json.Unmarshal(messages[0].Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Type != mapKafka.EventTopicMapStatusTypeCharacterExit {
		t.Errorf("Expected event type %v, got %v", mapKafka.EventTopicMapStatusTypeCharacterExit, event.Type)
	}
}

func TestProcessorImpl_TransitionMap(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	mockCp := &mockCharacterProcessor{}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	oldMapId := _map.Id(100000000)
	newMapId := _map.Id(100000001)
	oldInstance := uuid.Nil
	newInstance := uuid.Nil
	characterId := uint32(12345)
	newField := field.NewBuilder(worldId, channelId, newMapId).SetInstance(newInstance).Build()
	oldField := field.NewBuilder(worldId, channelId, oldMapId).SetInstance(oldInstance).Build()

	buf := message.NewBuffer()
	p.TransitionMap(buf)(transactionId, newField, characterId, oldField)

	// Verify character processor Exit was called on old map
	exitCalls := mockCp.GetExitCalls()
	if len(exitCalls) != 1 {
		t.Fatalf("Expected 1 Exit call, got %d", len(exitCalls))
	}
	if exitCalls[0].f.MapId() != oldMapId {
		t.Errorf("Expected Exit on old map %v, got %v", oldMapId, exitCalls[0].f.MapId())
	}

	// Verify character processor Enter was called on new map
	enterCalls := mockCp.GetEnterCalls()
	if len(enterCalls) != 1 {
		t.Fatalf("Expected 1 Enter call, got %d", len(enterCalls))
	}
	if enterCalls[0].f.MapId() != newMapId {
		t.Errorf("Expected Enter on new map %v, got %v", newMapId, enterCalls[0].f.MapId())
	}

	// Verify two messages were buffered (exit + enter)
	messages := buf.GetAll()
	if len(messages[mapKafka.EnvEventTopicMapStatus]) != 2 {
		t.Fatalf("Expected 2 messages in buffer, got %d", len(messages[mapKafka.EnvEventTopicMapStatus]))
	}
}

func TestProcessorImpl_TransitionMapAndEmit(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	mockCp := &mockCharacterProcessor{}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	oldMapId := _map.Id(100000000)
	newMapId := _map.Id(100000001)
	oldInstance := uuid.Nil
	newInstance := uuid.Nil
	characterId := uint32(12345)
	newField := field.NewBuilder(worldId, channelId, newMapId).SetInstance(newInstance).Build()
	oldField := field.NewBuilder(worldId, channelId, oldMapId).SetInstance(oldInstance).Build()

	err := p.TransitionMapAndEmit(transactionId, newField, characterId, oldField)
	if err != nil {
		t.Fatalf("TransitionMapAndEmit returned error: %v", err)
	}

	// Verify messages were emitted
	messages := mockPp.GetMessages(mapKafka.EnvEventTopicMapStatus)
	if len(messages) != 2 {
		t.Fatalf("Expected 2 emitted messages, got %d", len(messages))
	}
}

func TestProcessorImpl_TransitionChannel(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	mockCp := &mockCharacterProcessor{}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	oldChannelId := channel.Id(1)
	newChannelId := channel.Id(2)
	mapId := _map.Id(100000000)
	instance := uuid.Nil
	characterId := uint32(12345)
	newField := field.NewBuilder(worldId, newChannelId, mapId).SetInstance(instance).Build()

	buf := message.NewBuffer()
	p.TransitionChannel(buf)(transactionId, newField, oldChannelId, characterId)

	// Verify character processor Exit was called on old channel
	exitCalls := mockCp.GetExitCalls()
	if len(exitCalls) != 1 {
		t.Fatalf("Expected 1 Exit call, got %d", len(exitCalls))
	}
	if exitCalls[0].f.ChannelId() != oldChannelId {
		t.Errorf("Expected Exit on old channel %v, got %v", oldChannelId, exitCalls[0].f.ChannelId())
	}

	// Verify character processor Enter was called on new channel
	enterCalls := mockCp.GetEnterCalls()
	if len(enterCalls) != 1 {
		t.Fatalf("Expected 1 Enter call, got %d", len(enterCalls))
	}
	if enterCalls[0].f.ChannelId() != newChannelId {
		t.Errorf("Expected Enter on new channel %v, got %v", newChannelId, enterCalls[0].f.ChannelId())
	}

	// Verify two messages were buffered (exit + enter)
	messages := buf.GetAll()
	if len(messages[mapKafka.EnvEventTopicMapStatus]) != 2 {
		t.Fatalf("Expected 2 messages in buffer, got %d", len(messages[mapKafka.EnvEventTopicMapStatus]))
	}
}

func TestProcessorImpl_TransitionChannelAndEmit(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	mockCp := &mockCharacterProcessor{}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	oldChannelId := channel.Id(1)
	newChannelId := channel.Id(2)
	mapId := _map.Id(100000000)
	instance := uuid.Nil
	characterId := uint32(12345)
	newField := field.NewBuilder(worldId, newChannelId, mapId).SetInstance(instance).Build()

	err := p.TransitionChannelAndEmit(transactionId, newField, oldChannelId, characterId)
	if err != nil {
		t.Fatalf("TransitionChannelAndEmit returned error: %v", err)
	}

	// Verify messages were emitted
	messages := mockPp.GetMessages(mapKafka.EnvEventTopicMapStatus)
	if len(messages) != 2 {
		t.Fatalf("Expected 2 emitted messages, got %d", len(messages))
	}
}

func TestProcessorImpl_GetCharactersInMap(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()

	expectedCharacters := []uint32{123, 456, 789}
	mockCp := &mockCharacterProcessor{
		getCharactersInMapFunc: func(transactionId uuid.UUID, f field.Model) ([]uint32, error) {
			return expectedCharacters, nil
		},
	}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	instance := uuid.Nil
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	characters, err := p.GetCharactersInMap(transactionId, f)
	if err != nil {
		t.Fatalf("GetCharactersInMap returned error: %v", err)
	}

	if len(characters) != len(expectedCharacters) {
		t.Fatalf("Expected %d characters, got %d", len(expectedCharacters), len(characters))
	}

	for i, expected := range expectedCharacters {
		if characters[i] != expected {
			t.Errorf("Expected character %d at index %d, got %d", expected, i, characters[i])
		}
	}
}

func TestProcessorImpl_GetCharactersInMap_Empty(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()

	mockCp := &mockCharacterProcessor{
		getCharactersInMapFunc: func(transactionId uuid.UUID, f field.Model) ([]uint32, error) {
			return []uint32{}, nil
		},
	}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	instance := uuid.Nil
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	characters, err := p.GetCharactersInMap(transactionId, f)
	if err != nil {
		t.Fatalf("GetCharactersInMap returned error: %v", err)
	}

	if len(characters) != 0 {
		t.Fatalf("Expected 0 characters, got %d", len(characters))
	}
}

func TestProcessorImpl_Enter_WithInstance(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	mockCp := &mockCharacterProcessor{}
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	transactionId := uuid.New()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	instance := uuid.New()
	characterId := uint32(12345)
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	buf := message.NewBuffer()
	err := p.Enter(buf)(transactionId, f, characterId)
	if err != nil {
		t.Fatalf("Enter returned error: %v", err)
	}

	// Verify character processor Enter was called with instance
	enterCalls := mockCp.GetEnterCalls()
	if len(enterCalls) != 1 {
		t.Fatalf("Expected 1 Enter call, got %d", len(enterCalls))
	}

	call := enterCalls[0]
	if call.f.Instance() != instance {
		t.Errorf("Expected instance %v, got %v", instance, call.f.Instance())
	}

	// Verify message content includes instance
	messages := buf.GetAll()
	msg := messages[mapKafka.EnvEventTopicMapStatus][0]
	var event mapKafka.StatusEvent[mapKafka.CharacterEnter]
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Instance != instance {
		t.Errorf("Expected event instance %v, got %v", instance, event.Instance)
	}
}

func TestExit_LastCharacterClearsEnvironment(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	cp := character.NewProcessor(logger, ctx)
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, cp, mockPp)

	transactionId := uuid.New()
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(910010000)).Build()
	characterId := uint32(1)

	cp.Enter(transactionId, f, characterId)
	if _, err := environment.NewProcessor(logger, ctx).Set(f, field.ObjectKindObstacle, "a", 1); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	buf := message.NewBuffer()
	if err := p.Exit(buf)(transactionId, f, characterId); err != nil {
		t.Fatalf("Exit returned error: %v", err)
	}

	entries := environment.NewProcessor(logger, ctx).GetAll(f)
	if len(entries) != 0 {
		t.Fatalf("Expected environment state to be cleared, got %d entries", len(entries))
	}
}

func TestExit_RemainingCharacterKeepsEnvironment(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	cp := character.NewProcessor(logger, ctx)
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, cp, mockPp)

	transactionId := uuid.New()
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(910010000)).Build()
	characterId1 := uint32(1)
	characterId2 := uint32(2)

	cp.Enter(transactionId, f, characterId1)
	cp.Enter(transactionId, f, characterId2)
	if _, err := environment.NewProcessor(logger, ctx).Set(f, field.ObjectKindObstacle, "a", 1); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	buf := message.NewBuffer()
	if err := p.Exit(buf)(transactionId, f, characterId1); err != nil {
		t.Fatalf("Exit returned error: %v", err)
	}

	entries := environment.NewProcessor(logger, ctx).GetAll(f)
	if len(entries) != 1 {
		t.Fatalf("Expected environment state to be kept, got %d entries", len(entries))
	}
	if entries[0].State != 1 {
		t.Errorf("Expected state 1, got %d", entries[0].State)
	}
}

func TestExit_OtherFieldUnaffected(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	cp := character.NewProcessor(logger, ctx)
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, cp, mockPp)

	transactionId := uuid.New()
	f1 := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(910010000)).Build()
	f2 := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(910010100)).Build()
	characterId := uint32(1)

	cp.Enter(transactionId, f1, characterId)
	cp.Enter(transactionId, f2, characterId)
	ep := environment.NewProcessor(logger, ctx)
	if _, err := ep.Set(f1, field.ObjectKindObstacle, "a", 1); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if _, err := ep.Set(f2, field.ObjectKindObstacle, "a", 1); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	buf := message.NewBuffer()
	if err := p.Exit(buf)(transactionId, f1, characterId); err != nil {
		t.Fatalf("Exit returned error: %v", err)
	}

	if entries := ep.GetAll(f1); len(entries) != 0 {
		t.Fatalf("Expected f1 environment state to be cleared, got %d entries", len(entries))
	}
	if entries := ep.GetAll(f2); len(entries) != 1 {
		t.Fatalf("Expected f2 environment state to remain, got %d entries", len(entries))
	}
}

func TestExit_NoEnvironmentTrackedIsNoop(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	cp := character.NewProcessor(logger, ctx)
	mockPp := newMockProducerProvider()

	p := createTestProcessor(logger, ctx, cp, mockPp)

	transactionId := uuid.New()
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(910010000)).Build()
	characterId := uint32(1)

	cp.Enter(transactionId, f, characterId)

	buf := message.NewBuffer()
	if err := p.Exit(buf)(transactionId, f, characterId); err != nil {
		t.Fatalf("Exit returned error: %v", err)
	}

	entries := environment.NewProcessor(logger, ctx).GetAll(f)
	if len(entries) != 0 {
		t.Fatalf("Expected no environment state, got %d entries", len(entries))
	}
}

// mockMonsterDataProcessor implements monsterData.Processor for seeding the
// spawn point registry via InitializeForMap. GetSpawnPoints returns the
// unfiltered slice; the registry's Classify partitions it by MobTime/Hide.
type mockMonsterDataProcessor struct {
	spawnPoints []monsterData.SpawnPoint
}

func (m *mockMonsterDataProcessor) SpawnPointProvider(_ _map.Id) model.Provider[[]monsterData.SpawnPoint] {
	return func() ([]monsterData.SpawnPoint, error) { return m.spawnPoints, nil }
}

func (m *mockMonsterDataProcessor) GetSpawnPoints(_ _map.Id) ([]monsterData.SpawnPoint, error) {
	return m.spawnPoints, nil
}

// seedSpawnPoints initializes the shared spawn point registry for f (keyed on
// the tenant in ctx) with points, and returns the resulting MapKey.
func seedSpawnPoints(t *testing.T, ctx context.Context, f field.Model, points []monsterData.SpawnPoint) character.MapKey {
	t.Helper()
	mapKey := character.MapKey{Tenant: tenant.MustFromContext(ctx), Field: f}
	dp := &mockMonsterDataProcessor{spawnPoints: points}
	l, _ := test.NewNullLogger()
	if err := monster2.GetRegistry().InitializeForMap(ctx, mapKey, dp, l); err != nil {
		t.Fatalf("InitializeForMap: %v", err)
	}
	return mapKey
}

func oneTimeSpawnPoints(n int) []monsterData.SpawnPoint {
	points := make([]monsterData.SpawnPoint, 0, n)
	for i := 1; i <= n; i++ {
		points = append(points, monsterData.SpawnPoint{Id: uint32(i), Template: 9300044, MobTime: -1})
	}
	return points
}

func recurringSpawnPoints(n int) []monsterData.SpawnPoint {
	points := make([]monsterData.SpawnPoint, 0, n)
	for i := 1; i <= n; i++ {
		points = append(points, monsterData.SpawnPoint{Id: uint32(i), Template: 9300044, MobTime: 0})
	}
	return points
}

func TestProcessorImpl_Exit_RearmsAndDestroysOnEmpty(t *testing.T) {
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	transactionId := uuid.New()
	characterId := uint32(12345)

	tests := []struct {
		name string
		// seed builds the field's spawn points and, for the fired cases,
		// claims the one-time batch before Exit runs. seededOneTime tracks
		// whether the field was one-time-seeded, for the post-Exit re-arm
		// check.
		seed             func(t *testing.T, ctx context.Context, f field.Model) character.MapKey
		seededOneTime    bool
		remainingChars   []uint32
		wantDestroyMsgs  int
		wantClaimedAfter int
	}{
		{
			name: "last character leaves a fired field",
			seed: func(t *testing.T, ctx context.Context, f field.Model) character.MapKey {
				mapKey := seedSpawnPoints(t, ctx, f, oneTimeSpawnPoints(10))
				if _, err := monster2.GetRegistry().ClaimOneTimeSpawnPoints(ctx, mapKey); err != nil {
					t.Fatalf("ClaimOneTimeSpawnPoints (seed fire): %v", err)
				}
				return mapKey
			},
			seededOneTime:    true,
			remainingChars:   nil,
			wantDestroyMsgs:  1,
			wantClaimedAfter: 10,
		},
		{
			name: "characters remain",
			seed: func(t *testing.T, ctx context.Context, f field.Model) character.MapKey {
				mapKey := seedSpawnPoints(t, ctx, f, oneTimeSpawnPoints(10))
				if _, err := monster2.GetRegistry().ClaimOneTimeSpawnPoints(ctx, mapKey); err != nil {
					t.Fatalf("ClaimOneTimeSpawnPoints (seed fire): %v", err)
				}
				return mapKey
			},
			seededOneTime:    true,
			remainingChars:   []uint32{2222},
			wantDestroyMsgs:  0,
			wantClaimedAfter: 0,
		},
		{
			name: "never-fired field empties",
			seed: func(t *testing.T, ctx context.Context, f field.Model) character.MapKey {
				return seedSpawnPoints(t, ctx, f, recurringSpawnPoints(4))
			},
			remainingChars:  nil,
			wantDestroyMsgs: 0,
		},
		{
			name: "unseeded field empties",
			seed: func(t *testing.T, ctx context.Context, f field.Model) character.MapKey {
				return character.MapKey{}
			},
			remainingChars:  nil,
			wantDestroyMsgs: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger, _ := test.NewNullLogger()
			ctx := createTestContext()
			f := field.NewBuilder(worldId, channelId, mapId).Build()
			mapKey := tc.seed(t, ctx, f)

			remainingChars := tc.remainingChars
			mockCp := &mockCharacterProcessor{
				getCharactersInMapFunc: func(uuid.UUID, field.Model) ([]uint32, error) { return remainingChars, nil },
			}
			mockPp := newMockProducerProvider()
			p := createTestProcessor(logger, ctx, mockCp, mockPp)

			buf := message.NewBuffer()
			if err := p.Exit(buf)(transactionId, f, characterId); err != nil {
				t.Fatalf("Exit returned error: %v", err)
			}

			messages := buf.GetAll()
			if len(messages[mapKafka.EnvEventTopicMapStatus]) != 1 {
				t.Fatalf("Expected 1 CHARACTER_EXIT message, got %d", len(messages[mapKafka.EnvEventTopicMapStatus]))
			}
			destroyMsgs := messages[monsterKafka.EnvCommandTopic]
			if len(destroyMsgs) != tc.wantDestroyMsgs {
				t.Fatalf("Expected %d DESTROY_FIELD message(s), got %d", tc.wantDestroyMsgs, len(destroyMsgs))
			}

			if tc.wantDestroyMsgs > 0 {
				var cmd monsterKafka.FieldCommand[monsterKafka.DestroyFieldBody]
				if err := json.Unmarshal(destroyMsgs[0].Value, &cmd); err != nil {
					t.Fatalf("Failed to unmarshal destroy message: %v", err)
				}
				if cmd.Type != monsterKafka.CommandTypeDestroyField {
					t.Errorf("Expected type %q, got %q", monsterKafka.CommandTypeDestroyField, cmd.Type)
				}
				if cmd.MapId != f.MapId() {
					t.Errorf("Expected mapId %v, got %v", f.MapId(), cmd.MapId)
				}
			}

			if tc.seededOneTime {
				claimed, err := monster2.GetRegistry().ClaimOneTimeSpawnPoints(ctx, mapKey)
				if err != nil {
					t.Fatalf("post-Exit ClaimOneTimeSpawnPoints: %v", err)
				}
				if len(claimed) != tc.wantClaimedAfter {
					t.Errorf("Expected post-Exit claim to yield %d claimed points, got %d", tc.wantClaimedAfter, len(claimed))
				}
			}
		})
	}
}

// TestProcessorImpl_Exit_RearmIsPerFieldKey is the FR-3.4 regression: re-arming
// one channel instance of a map must not affect a sibling channel instance's
// one-time state.
func TestProcessorImpl_Exit_RearmIsPerFieldKey(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := createTestContext()
	worldId := world.Id(1)
	mapId := _map.Id(100000000)
	transactionId := uuid.New()
	characterId := uint32(12345)

	fieldCh0 := field.NewBuilder(worldId, channel.Id(0), mapId).Build()
	fieldCh1 := field.NewBuilder(worldId, channel.Id(1), mapId).Build()

	mapKeyCh0 := seedSpawnPoints(t, ctx, fieldCh0, oneTimeSpawnPoints(10))
	mapKeyCh1 := seedSpawnPoints(t, ctx, fieldCh1, oneTimeSpawnPoints(10))
	if _, err := monster2.GetRegistry().ClaimOneTimeSpawnPoints(ctx, mapKeyCh0); err != nil {
		t.Fatalf("ClaimOneTimeSpawnPoints ch0 (seed fire): %v", err)
	}
	if _, err := monster2.GetRegistry().ClaimOneTimeSpawnPoints(ctx, mapKeyCh1); err != nil {
		t.Fatalf("ClaimOneTimeSpawnPoints ch1 (seed fire): %v", err)
	}

	mockCp := &mockCharacterProcessor{
		getCharactersInMapFunc: func(uuid.UUID, field.Model) ([]uint32, error) { return nil, nil },
	}
	mockPp := newMockProducerProvider()
	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	buf := message.NewBuffer()
	if err := p.Exit(buf)(transactionId, fieldCh0, characterId); err != nil {
		t.Fatalf("Exit returned error: %v", err)
	}

	claimedCh0, err := monster2.GetRegistry().ClaimOneTimeSpawnPoints(ctx, mapKeyCh0)
	if err != nil {
		t.Fatalf("ClaimOneTimeSpawnPoints ch0: %v", err)
	}
	if len(claimedCh0) != 10 {
		t.Errorf("Expected channel 0 to be re-armed with 10 points, got %d", len(claimedCh0))
	}

	claimedCh1, err := monster2.GetRegistry().ClaimOneTimeSpawnPoints(ctx, mapKeyCh1)
	if err != nil {
		t.Fatalf("ClaimOneTimeSpawnPoints ch1: %v", err)
	}
	if len(claimedCh1) != 0 {
		t.Errorf("Expected channel 1 to remain disarmed, got %d claimed points", len(claimedCh1))
	}
}

// TestProcessorImpl_Exit_LogsRearm is the FR-4.3 regression: a re-arm must be
// observable at debug level.
func TestProcessorImpl_Exit_LogsRearm(t *testing.T) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := createTestContext()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	transactionId := uuid.New()
	characterId := uint32(12345)

	f := field.NewBuilder(worldId, channelId, mapId).Build()
	mapKey := seedSpawnPoints(t, ctx, f, oneTimeSpawnPoints(10))
	if _, err := monster2.GetRegistry().ClaimOneTimeSpawnPoints(ctx, mapKey); err != nil {
		t.Fatalf("ClaimOneTimeSpawnPoints (seed fire): %v", err)
	}

	mockCp := &mockCharacterProcessor{
		getCharactersInMapFunc: func(uuid.UUID, field.Model) ([]uint32, error) { return nil, nil },
	}
	mockPp := newMockProducerProvider()
	p := createTestProcessor(logger, ctx, mockCp, mockPp)

	buf := message.NewBuffer()
	if err := p.Exit(buf)(transactionId, f, characterId); err != nil {
		t.Fatalf("Exit returned error: %v", err)
	}

	want := fmt.Sprintf("Re-armed one-time spawn points for field [%s].", f.Id())
	found := false
	for _, entry := range hook.AllEntries() {
		if entry.Message == want {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected a debug log entry %q, got none", want)
	}
}
