package saga

import (
	"atlas-npc-conversations/conversation"
	npcmsg "atlas-npc-conversations/kafka/message/npc"
	sagamsg "atlas-npc-conversations/kafka/message/saga"
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	segkafka "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// messageRecorder counts kafka.Writer.WriteMessages calls per topic, so
// tests can assert a message was actually emitted on a given topic (e.g. the
// character-status topic Dispose publishes to) without a real broker.
type messageRecorder struct {
	mu     sync.Mutex
	counts map[string]int
}

func newMessageRecorder() *messageRecorder {
	return &messageRecorder{counts: make(map[string]int)}
}

func (r *messageRecorder) record(topic string, n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counts[topic] += n
}

func (r *messageRecorder) count(topic string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.counts[topic]
}

// stubWriter is a producer.Writer test double that swallows every message
// without touching the network, so handler paths that emit kafka messages
// (npcSender.Dispose, dialogue sends) don't block trying to reach a real
// broker during the test. Writes are tallied in the recorder by topic.
type stubWriter struct {
	topic    string
	recorder *messageRecorder
}

func (w stubWriter) Topic() string { return w.topic }
func (w stubWriter) WriteMessages(_ context.Context, msgs ...segkafka.Message) error {
	w.recorder.record(w.topic, len(msgs))
	return nil
}
func (w stubWriter) Close() error { return nil }

// stubKafkaProducer points the shared kafka producer Manager singleton at a
// stub Writer factory for the duration of the test, and resets it afterward.
// Returns the recorder so tests can assert which topics were written to.
func stubKafkaProducer(t *testing.T) *messageRecorder {
	t.Helper()
	rec := newMessageRecorder()
	kafkaproducer.ResetInstance()
	kafkaproducer.GetManager(kafkaproducer.ConfigWriterFactory(func(topicName string) kafkaproducer.Writer {
		return stubWriter{topic: topicName, recorder: rec}
	}))
	t.Cleanup(kafkaproducer.ResetInstance)
	return rec
}

// testStateContainer is a minimal conversation.StateContainer implementation
// for tests that need FindState and States() (the latter is required for the
// ConversationContext.MarshalJSON round-trip through the redis-backed
// registry - see conversation/model_json.go).
type testStateContainer struct {
	start  string
	states []conversation.StateModel
}

func (c testStateContainer) StartState() string { return c.start }

func (c testStateContainer) States() []conversation.StateModel { return c.states }

func (c testStateContainer) FindState(id string) (conversation.StateModel, error) {
	for _, s := range c.states {
		if s.Id() == id {
			return s, nil
		}
	}
	return conversation.StateModel{}, errors.New("state not found")
}

// newSagaConsumerTestContext wires a miniredis-backed conversation registry
// and a tenant-bearing context, mirroring the setup used in
// conversation/processor_rps_test.go.
func newSagaConsumerTestContext(t *testing.T) (context.Context, logrus.FieldLogger) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	conversation.InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	return tctx, l
}

func buildTestField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
}

// TestHandleStatusEventCompleted_RPSAction_ClearsPendingAndDisposes verifies
// that a COMPLETED status event for an rpsAction conversation (identified by
// the rpsAction_failureState context flag Task 23 stores) clears the pending
// saga and ends/disposes the conversation - the client dialog takes over via
// the atlas-rps GameOpened event, so no further NPC dialogue is needed.
func TestHandleStatusEventCompleted_RPSAction_ClearsPendingAndDisposes(t *testing.T) {
	rec := stubKafkaProducer(t)
	ctx, l := newSagaConsumerTestContext(t)

	sagaId := uuid.New()
	characterId := uint32(42)

	container := testStateContainer{start: "playRPS", states: []conversation.StateModel{}}
	cc := conversation.NewConversationContextBuilder().
		SetField(buildTestField()).
		SetCharacterId(characterId).
		SetNpcId(9000019).
		SetCurrentState("playRPS").
		SetConversation(container).
		SetPendingSagaId(sagaId).
		AddContextValue("rpsAction_failureState", "noMeso").
		Build()
	conversation.GetRegistry().SetContext(ctx, characterId, cc)

	event := sagamsg.StatusEvent[sagamsg.StatusEventCompletedBody]{
		TransactionId: sagaId,
		Type:          sagamsg.StatusEventTypeCompleted,
		Body:          sagamsg.StatusEventCompletedBody{},
	}

	handler := handleStatusEventCompleted(l, nil)
	handler(l, ctx, event)

	// The conversation should be ended (registry context cleared) - the RPS
	// game dialog now owns the client UI.
	if _, err := conversation.GetRegistry().GetPreviousContext(ctx, characterId); err == nil {
		t.Fatalf("expected conversation context to be cleared after RPS entry saga success (disposed), but it is still present")
	}

	// The saga index should also have been cleared (ClearContext removes
	// both). A lookup by sagaId should now fail.
	if _, err := conversation.GetRegistry().GetContextBySagaId(ctx, sagaId); err == nil {
		t.Fatalf("expected saga index entry to be cleared after RPS entry saga success")
	}

	// npcSender.Dispose must have actually been invoked (not just End()) -
	// this is what distinguishes the rpsAction branch from the generic
	// "no success state stored" fallback, which only ends the conversation.
	// Dispose publishes to the character-status topic.
	if got := rec.count(npcmsg.EnvEventTopicCharacterStatus); got == 0 {
		t.Errorf("expected npcSender.Dispose to publish to topic %q, but no message was recorded", npcmsg.EnvEventTopicCharacterStatus)
	}
}

// TestHandleStatusEventFailed_RPSAction_RoutesToStoredFailureState verifies
// that a FAILED status event (e.g. NOT_ENOUGH_MESO on the AwardMesos step)
// routes the conversation to the state stored under rpsAction_failureState in
// Task 23's processRPSActionState.
func TestHandleStatusEventFailed_RPSAction_RoutesToStoredFailureState(t *testing.T) {
	stubKafkaProducer(t)
	ctx, l := newSagaConsumerTestContext(t)

	sagaId := uuid.New()
	characterId := uint32(42)

	choiceA, err := conversation.NewChoiceBuilder().SetText("Ok").SetNextState("").Build()
	if err != nil {
		t.Fatalf("build choice A: %v", err)
	}
	choiceB, err := conversation.NewChoiceBuilder().SetText("Cancel").SetNextState("").Build()
	if err != nil {
		t.Fatalf("build choice B: %v", err)
	}
	dialogue, err := conversation.NewDialogueBuilder().
		SetDialogueType(conversation.SendOk).
		SetText("You don't have enough mesos.").
		AddChoice(choiceA).
		AddChoice(choiceB).
		Build()
	if err != nil {
		t.Fatalf("build dialogue: %v", err)
	}
	failureState, err := conversation.NewStateBuilder().SetId("noMeso").SetDialogue(dialogue).Build()
	if err != nil {
		t.Fatalf("build failure state: %v", err)
	}

	container := testStateContainer{
		start:  "playRPS",
		states: []conversation.StateModel{failureState},
	}
	cc := conversation.NewConversationContextBuilder().
		SetField(buildTestField()).
		SetCharacterId(characterId).
		SetNpcId(9000019).
		SetCurrentState("playRPS").
		SetConversation(container).
		SetPendingSagaId(sagaId).
		AddContextValue("rpsAction_failureState", "noMeso").
		Build()
	conversation.GetRegistry().SetContext(ctx, characterId, cc)

	event := sagamsg.StatusEvent[sagamsg.StatusEventFailedBody]{
		TransactionId: sagaId,
		Type:          sagamsg.StatusEventTypeFailed,
		Body: sagamsg.StatusEventFailedBody{
			ErrorCode:  "NOT_ENOUGH_MESO",
			Reason:     "insufficient funds",
			FailedStep: "deduct_entry_cost",
		},
	}

	handler := handleStatusEventFailed(l, nil)
	handler(l, ctx, event)

	stored, err := conversation.GetRegistry().GetPreviousContext(ctx, characterId)
	if err != nil {
		t.Fatalf("GetPreviousContext: %v", err)
	}
	if stored.CurrentState() != "noMeso" {
		t.Errorf("CurrentState() = %q, want %q (rpsAction_failureState)", stored.CurrentState(), "noMeso")
	}
	if stored.PendingSagaId() != nil {
		t.Errorf("PendingSagaId() = %v, want nil (cleared)", stored.PendingSagaId())
	}
	if _, exists := stored.Context()["rpsAction_failureState"]; exists {
		t.Errorf("rpsAction_failureState context key should have been cleaned up after routing")
	}
}
