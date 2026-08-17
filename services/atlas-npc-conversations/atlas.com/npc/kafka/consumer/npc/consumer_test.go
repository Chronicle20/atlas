package npc

import (
	"atlas-npc-conversations/conversation"
	"atlas-npc-conversations/conversation/item"
	npc2 "atlas-npc-conversations/kafka/message/npc"
	"atlas-npc-conversations/test"
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// testTenantId is the fixed tenant every helper in this file uses. Each test
// gets its own SQLite in-memory database (test.SetupTestDB / CleanupTestDB is
// a full open/close cycle) and its own miniredis-backed conversation
// registry, so reusing one tenant id across tests does not leak state between
// them — only within a single test's own db/registry pair.
var testTenantId = uuid.New()

// testLogger returns a logger that discards output, matching the pattern
// used throughout the service's existing tests (logtest.NewNullLogger).
func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := logtest.NewNullLogger()
	return l
}

// testCtx returns a tenant-bearing context for testTenantId, mirroring
// countTestTenant in conversation/item/processor_test.go.
func testCtx(t *testing.T) context.Context {
	t.Helper()
	te, err := tenant.Create(testTenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), te)
}

// initTestRegistry wires a fresh miniredis-backed conversation registry, the
// same setup newItemTestProcessor uses in
// conversation/processor_item_test.go. StartItem/Start read and write this
// package-level registry, so it must be live before the handler under test
// runs.
func initTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	conversation.InitRegistry(rc)
}

// emptyDB stands up an item-conversation table with no rows and a fresh
// conversation registry — the "no conversation authored" and "ordinary
// NPC-talk path" fixture.
func emptyDB(t *testing.T) *gorm.DB {
	t.Helper()
	initTestRegistry(t)
	db := test.SetupTestDB(t, item.MigrateTable)
	t.Cleanup(func() { test.CleanupTestDB(t, db) })
	return db
}

// seedItemConversation stands up an item-conversation table with one
// authored scripted-item dialogue and a fresh conversation registry.
func seedItemConversation(t *testing.T, itemId uint32, npcId uint32, scriptName string) *gorm.DB {
	t.Helper()
	initTestRegistry(t)
	db := test.SetupTestDB(t, item.MigrateTable)
	t.Cleanup(func() { test.CleanupTestDB(t, db) })

	// The outcome loops back to its own state ("intro" -> "intro") rather than
	// ending the conversation, mirroring parkingItemStateMachine in
	// conversation/processor_item_test.go. Without a self-loop, ProcessState
	// completes and clears the registry after the first StartItem call, so a
	// second call would see no live context and be treated as a fresh start
	// (STARTED) instead of the redelivery/conflict cases these tests exercise.
	outcome, err := conversation.NewOutcomeBuilder().SetNextState("intro").Build()
	if err != nil {
		t.Fatalf("building fixture outcome: %v", err)
	}
	ga, err := conversation.NewGenericActionBuilder().AddOutcome(outcome).Build()
	if err != nil {
		t.Fatalf("building fixture genericAction: %v", err)
	}
	state, err := conversation.NewStateBuilder().SetId("intro").SetGenericAction(ga).Build()
	if err != nil {
		t.Fatalf("building fixture state: %v", err)
	}

	m, err := item.NewBuilder().
		SetItemId(itemId).
		SetNpcId(npcId).
		SetScriptName(scriptName).
		SetStartState("intro").
		AddState(state).
		Build()
	if err != nil {
		t.Fatalf("building item conversation model: %v", err)
	}

	if _, err := item.NewProcessor(testLogger(t), testCtx(t), db).Create(m); err != nil {
		t.Fatalf("seeding item conversation: %v", err)
	}
	return db
}

// captured is what the emit seam recorded, so the tests can assert on
// emissions without a broker.
type captured struct {
	count  int
	evType string
	reason string
}

// install replaces the emit seam for one test and restores it after. It
// decodes whichever StatusEvent body arrived so a single helper serves both
// types.
func install(t *testing.T, c *captured) {
	t.Helper()
	orig := emitConversationStatus
	emitConversationStatus = func(_ logrus.FieldLogger, _ context.Context, p model.Provider[[]kafka.Message]) error {
		msgs, err := p()
		if err != nil {
			return err
		}
		for _, m := range msgs {
			var probe struct {
				Type string `json:"type"`
				Body struct {
					Reason string `json:"reason"`
				} `json:"body"`
			}
			if err := json.Unmarshal(m.Value, &probe); err != nil {
				return err
			}
			c.count++
			c.evType = probe.Type
			c.reason = probe.Body.Reason
		}
		return nil
	}
	t.Cleanup(func() { emitConversationStatus = orig })
}

// A saga-driven start (non-nil transactionId) emits exactly one STARTED.
func TestStartItemConversation_EmitsStartedOnSuccess(t *testing.T) {
	var c captured
	install(t, &c)
	db := seedItemConversation(t, 2430013, 9010000, "item_2430013")

	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), npc2.Command[npc2.CommandItemConversationStartBody]{
		TransactionId: uuid.New(),
		NpcId:         9010000,
		CharacterId:   1234,
		Type:          npc2.CommandTypeStartItemConversation,
		Body:          npc2.CommandItemConversationStartBody{ItemId: 2430013, Slot: 5, AccountId: 77},
	})

	if c.count != 1 || c.evType != npc2.StatusEventTypeStarted {
		t.Fatalf("emitted %d event(s) of type %q, want 1 STARTED", c.count, c.evType)
	}
}

// A content gap is not a fault. START_ERROR fails the awaiting step, the
// following destroy never runs, and the player keeps the item.
func TestStartItemConversation_EmitsNoConversationAuthoredWhenUnauthored(t *testing.T) {
	var c captured
	install(t, &c)
	db := emptyDB(t) // no conversation authored for 2430013

	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), npc2.Command[npc2.CommandItemConversationStartBody]{
		TransactionId: uuid.New(),
		NpcId:         9010000,
		CharacterId:   1234,
		Type:          npc2.CommandTypeStartItemConversation,
		Body:          npc2.CommandItemConversationStartBody{ItemId: 2430013, Slot: 5},
	})

	if c.count != 1 || c.evType != npc2.StatusEventTypeStartError {
		t.Fatalf("emitted %d event(s) of type %q, want 1 START_ERROR", c.count, c.evType)
	}
	if c.reason != npc2.StartErrorNoConversationAuthored {
		t.Errorf("reason: got %q, want %q", c.reason, npc2.StartErrorNoConversationAuthored)
	}
}

// A DIFFERENT transaction against a live context is a genuine conflict.
func TestStartItemConversation_EmitsConversationInProgressOnConflict(t *testing.T) {
	var c captured
	db := seedItemConversation(t, 2430013, 9010000, "item_2430013")
	cmd := npc2.Command[npc2.CommandItemConversationStartBody]{
		TransactionId: uuid.New(),
		NpcId:         9010000,
		CharacterId:   1234,
		Type:          npc2.CommandTypeStartItemConversation,
		Body:          npc2.CommandItemConversationStartBody{ItemId: 2430013, Slot: 5},
	}
	install(t, &captured{})                                                // throwaway sink: the seed call must not reach the real producer
	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), cmd) // occupies the character

	install(t, &c) // only observe the SECOND command
	second := cmd
	second.TransactionId = uuid.New()
	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), second)

	if c.count != 1 || c.evType != npc2.StatusEventTypeStartError {
		t.Fatalf("emitted %d event(s) of type %q, want 1 START_ERROR", c.count, c.evType)
	}
	if c.reason != npc2.StartErrorConversationInProgress {
		t.Errorf("reason: got %q, want %q", c.reason, npc2.StartErrorConversationInProgress)
	}
}

// Redelivery: the SAME transaction id against its own live context re-emits
// STARTED, never START_ERROR. Emitting an error here would fail a saga that
// had already succeeded — Kafka is at-least-once, so this is the realistic
// case.
func TestStartItemConversation_RedeliveryReemitsStarted(t *testing.T) {
	var c captured
	db := seedItemConversation(t, 2430013, 9010000, "item_2430013")
	cmd := npc2.Command[npc2.CommandItemConversationStartBody]{
		TransactionId: uuid.New(),
		NpcId:         9010000,
		CharacterId:   1234,
		Type:          npc2.CommandTypeStartItemConversation,
		Body:          npc2.CommandItemConversationStartBody{ItemId: 2430013, Slot: 5},
	}
	install(t, &captured{}) // throwaway sink: the seed call must not reach the real producer
	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), cmd)

	install(t, &c) // only observe the redelivery
	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), cmd)

	if c.count != 1 || c.evType != npc2.StatusEventTypeStarted {
		t.Fatalf("redelivery emitted %d event(s) of type %q, want 1 STARTED", c.count, c.evType)
	}
}

// The ordinary NPC-talk path is unchanged: no transaction, no status event.
func TestStartConversation_NilTransactionEmitsNoStatus(t *testing.T) {
	var c captured
	install(t, &c)
	db := emptyDB(t)

	handleStartConversationCommand(db)(testLogger(t), testCtx(t), npc2.Command[npc2.CommandConversationStartBody]{
		TransactionId: uuid.Nil,
		NpcId:         1002000,
		CharacterId:   1234,
		Type:          npc2.CommandTypeStartConversation,
		Body:          npc2.CommandConversationStartBody{},
	})

	if c.count != 0 {
		t.Fatalf("non-saga start emitted %d event(s), want 0", c.count)
	}
}
