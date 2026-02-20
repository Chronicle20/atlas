package _map

import (
	"atlas-maps/kafka/message"
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/kafka/producer"
	"atlas-maps/map/character"
	monster2 "atlas-maps/map/monster"
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	kafkaProducer "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
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

func (m *mockCharacterProcessor) GetMapsWithCharacters() []character.MapKey {
	if m.getMapsWithCharactersFunc != nil {
		return m.getMapsWithCharactersFunc()
	}
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
	messages map[string][]kafka.Message
}

func newMockProducerProvider() *mockProducerProvider {
	return &mockProducerProvider{
		messages: make(map[string][]kafka.Message),
	}
}

func (m *mockProducerProvider) Provider() producer.Provider {
	return func(token string) kafkaProducer.MessageProducer {
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

func (m *mockProducerProvider) GetMessages(topic string) []kafka.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]kafka.Message(nil), m.messages[topic]...)
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
