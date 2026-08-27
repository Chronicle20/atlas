package seed

import (
	"atlas-channel/maplelife"
	"atlas-channel/saga"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	seedmsg "atlas-channel/kafka/message/seed"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	mlcb "github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	testAccountId     = uint32(7)
	testSubmitCharId  = uint32(42) // Entry.CharacterId -- the submitter, holds the item
	testCreatedCharId = uint32(99) // event Body.CharacterId -- the newly created character
	testWorldId       = world.Id(1)
	testItemId        = item.Id(5431000)
	testSlot          = slot.Position(-3)
	testTransactionId = "tx-1"
)

const (
	testByteSuccess = 0x70
	testByteUnknown = 0x71
)

// discardConn is a minimal net.Conn stub -- session.Announce's success path
// runs the real announceEncrypted -> con.Write chain, so a live, harmless
// sink is needed (precedent: discardConn in
// socket/handler/character_damage_test.go).
type discardConn struct{}

func (discardConn) Read(_ []byte) (int, error)         { return 0, io.EOF }
func (discardConn) Write(b []byte) (int, error)        { return len(b), nil }
func (discardConn) Close() error                       { return nil }
func (discardConn) LocalAddr() net.Addr                { return nil }
func (discardConn) RemoteAddr() net.Addr               { return nil }
func (discardConn) SetDeadline(_ time.Time) error      { return nil }
func (discardConn) SetReadDeadline(_ time.Time) error  { return nil }
func (discardConn) SetWriteDeadline(_ time.Time) error { return nil }

var _ net.Conn = discardConn{}

func mustTenant(t *testing.T, region string, major uint16, minor uint16) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return m
}

type testEnv struct {
	t    *testing.T
	ten  tenant.Model
	ctx  context.Context
	sc   server.Model
	l    logrus.FieldLogger
	hook *testlog.Hook

	wp        writer.Producer
	announced []struct {
		writer string
		body   []byte
	}

	destroyCalls []saga.Saga
	destroyErr   error
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	ch := channel.NewModel(testWorldId, channel.Id(0))
	sc := server.NewProcessor(logrus.New(), ctx).Register(ten, ch, "127.0.0.1", 8080)

	l, hook := testlog.NewNullLogger()

	env := &testEnv{t: t, ten: ten, ctx: ctx, sc: sc, l: l, hook: hook}

	env.wp = func(name string) (swriter.BodyFunc, error) {
		return func(bl logrus.FieldLogger, bctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(bl, bctx)(map[string]interface{}{
					"operations": map[string]interface{}{
						mlcb.MapleLifeErrorSuccess:      float64(testByteSuccess),
						mlcb.MapleLifeErrorUnknownError: float64(testByteUnknown),
					},
				})
				env.announced = append(env.announced, struct {
					writer string
					body   []byte
				}{writer: name, body: b})
				return b
			}
		}, nil
	}

	origDestroy := destroyCashItemFunc
	destroyCashItemFunc = func(_ logrus.FieldLogger, _ context.Context, s saga.Saga) error {
		env.destroyCalls = append(env.destroyCalls, s)
		return env.destroyErr
	}
	t.Cleanup(func() { destroyCashItemFunc = origDestroy })

	t.Cleanup(func() {
		maplelife.GetRegistry().ClearAccount(ten, testAccountId)
		session.ClearRegistryForTenant(ten.Id())
	})

	return env
}

// putSubmittedEntry seeds a PhaseSubmitted registry entry under testAccountId
// with the given transaction id -- CharacterId is deliberately
// testSubmitCharId (42), NOT testCreatedCharId (99), so a test that swapped
// the submitting/created character ids would fail.
func (e *testEnv) putSubmittedEntry(transactionId string) {
	e.t.Helper()
	maplelife.GetRegistry().Put(e.ten, testAccountId, maplelife.Entry{
		CharacterId:   testSubmitCharId,
		WorldId:       testWorldId,
		ItemId:        testItemId,
		Slot:          testSlot,
		Phase:         maplelife.PhaseSubmitted,
		TransactionId: transactionId,
		CandidateName: "Zulu",
		At:            time.Now(),
	})
}

func (e *testEnv) entryExists() bool {
	_, ok := maplelife.GetRegistry().Get(e.ten, testAccountId)
	return ok
}

// connectSession registers a live session for testAccountId so
// IfPresentByAccountId resolves it.
func (e *testEnv) connectSession() {
	e.t.Helper()
	sp := session.NewProcessor(e.l, e.ctx)
	sessionId := uuid.New()
	sp.Create(e.sc.Channel(), 0)(sessionId, discardConn{})
	sp.SetAccountId(sessionId, testAccountId)
	sp.SetCharacterId(sessionId, testSubmitCharId)
}

func (e *testEnv) lastArm() (byte, bool) {
	if len(e.announced) == 0 {
		return 0, false
	}
	a := e.announced[len(e.announced)-1]
	if a.writer != mlcb.MapleLifeErrorWriter {
		e.t.Fatalf("wrote [%s], want [%s]", a.writer, mlcb.MapleLifeErrorWriter)
	}
	if len(a.body) < 1 {
		e.t.Fatalf("body too short: %x", a.body)
	}
	return a.body[0], true
}

func (e *testEnv) dispatchCreated(evt seedmsg.StatusEvent[seedmsg.CreatedStatusEventBody]) {
	handleCreatedStatusEvent(e.sc, e.wp)(e.l, e.ctx, evt)
}

func (e *testEnv) dispatchFailed(evt seedmsg.StatusEvent[seedmsg.FailedStatusEventBody]) {
	handleFailedStatusEvent(e.sc, e.wp)(e.l, e.ctx, evt)
}

func createdEvent(accountId uint32, transactionId string, createdCharId uint32) seedmsg.StatusEvent[seedmsg.CreatedStatusEventBody] {
	return seedmsg.StatusEvent[seedmsg.CreatedStatusEventBody]{
		AccountId:     accountId,
		TransactionId: transactionId,
		Type:          seedmsg.StatusEventTypeCreated,
		Body:          seedmsg.CreatedStatusEventBody{CharacterId: createdCharId},
	}
}

func assertDestroySagaCorrect(t *testing.T, s saga.Saga) {
	t.Helper()
	if s.SagaType != saga.MapleLifeUse {
		t.Errorf("SagaType = %q, want %q", s.SagaType, saga.MapleLifeUse)
	}
	if len(s.Steps) != 1 {
		t.Fatalf("Steps = %d, want 1", len(s.Steps))
	}
	step := s.Steps[0]
	if step.Action != saga.DestroyAssetFromSlot {
		t.Errorf("Action = %q, want %q", step.Action, saga.DestroyAssetFromSlot)
	}
	p, ok := step.Payload.(saga.DestroyAssetFromSlotPayload)
	if !ok {
		t.Fatalf("Payload type = %T, want DestroyAssetFromSlotPayload", step.Payload)
	}
	if p.CharacterId != testSubmitCharId {
		t.Errorf("CharacterId = %d, want %d (the SUBMITTING character, not the created one [%d])", p.CharacterId, testSubmitCharId, testCreatedCharId)
	}
	if p.InventoryType != byte(inventory.TypeValueCash) {
		t.Errorf("InventoryType = %d, want %d", p.InventoryType, byte(inventory.TypeValueCash))
	}
	if p.Slot != int16(testSlot) {
		t.Errorf("Slot = %d, want %d", p.Slot, int16(testSlot))
	}
	if p.TemplateId != uint32(testItemId) {
		t.Errorf("TemplateId = %d, want %d", p.TemplateId, uint32(testItemId))
	}
	if p.Quantity != 1 {
		t.Errorf("Quantity = %d, want 1", p.Quantity)
	}
}

func TestSeedCreatedConsumesItemAndAnnounces(t *testing.T) {
	t.Run("matched by transaction id", func(t *testing.T) {
		env := newTestEnv(t)
		env.putSubmittedEntry(testTransactionId)
		env.connectSession()

		env.dispatchCreated(createdEvent(testAccountId, testTransactionId, testCreatedCharId))

		if len(env.destroyCalls) != 1 {
			t.Fatalf("destroy calls = %d, want 1", len(env.destroyCalls))
		}
		assertDestroySagaCorrect(t, env.destroyCalls[0])

		arm, ok := env.lastArm()
		if !ok {
			t.Fatal("expected a MAPLELIFE_ERROR announce")
		}
		if arm != testByteSuccess {
			t.Errorf("arm = %#x, want SUCCESS %#x", arm, testByteSuccess)
		}
		if env.entryExists() {
			t.Error("registry entry should have been removed")
		}
	})

	t.Run("fallback by account id", func(t *testing.T) {
		env := newTestEnv(t)
		env.putSubmittedEntry(testTransactionId)
		env.connectSession()

		env.dispatchCreated(createdEvent(testAccountId, "", testCreatedCharId))

		if len(env.destroyCalls) != 1 {
			t.Fatalf("destroy calls = %d, want 1", len(env.destroyCalls))
		}
		assertDestroySagaCorrect(t, env.destroyCalls[0])

		arm, ok := env.lastArm()
		if !ok {
			t.Fatal("expected a MAPLELIFE_ERROR announce")
		}
		if arm != testByteSuccess {
			t.Errorf("arm = %#x, want SUCCESS %#x", arm, testByteSuccess)
		}
		if env.entryExists() {
			t.Error("registry entry should have been removed")
		}
	})

	t.Run("wrong transaction id", func(t *testing.T) {
		env := newTestEnv(t)
		env.putSubmittedEntry(testTransactionId)
		env.connectSession()

		env.dispatchCreated(createdEvent(testAccountId, "tx-other", testCreatedCharId))

		if len(env.destroyCalls) != 0 {
			t.Fatalf("destroy calls = %d, want 0", len(env.destroyCalls))
		}
		if len(env.announced) != 0 {
			t.Fatalf("announced = %d, want 0", len(env.announced))
		}
		if !env.entryExists() {
			t.Error("pending entry should have been left intact")
		}
		foundWarn := false
		for _, e := range env.hook.AllEntries() {
			if e.Level == logrus.WarnLevel {
				foundWarn = true
			}
		}
		if !foundWarn {
			t.Errorf("expected a warning-level log entry, got: %+v", env.hook.AllEntries())
		}
	})

	t.Run("wrong tenant", func(t *testing.T) {
		env := newTestEnv(t)
		env.putSubmittedEntry(testTransactionId)
		env.connectSession()

		otherTenant := mustTenant(t, "GMS", 83, 1)
		otherCtx := tenant.WithContext(context.Background(), otherTenant)
		t.Cleanup(func() { maplelife.GetRegistry().ClearAccount(otherTenant, testAccountId) })

		handleCreatedStatusEvent(env.sc, env.wp)(env.l, otherCtx, createdEvent(testAccountId, testTransactionId, testCreatedCharId))

		if len(env.destroyCalls) != 0 {
			t.Fatalf("destroy calls = %d, want 0", len(env.destroyCalls))
		}
		if len(env.announced) != 0 {
			t.Fatalf("announced = %d, want 0", len(env.announced))
		}
		if !env.entryExists() {
			t.Error("tenant A's entry should be intact")
		}
	})

	t.Run("duplicate delivery", func(t *testing.T) {
		env := newTestEnv(t)
		env.putSubmittedEntry(testTransactionId)
		env.connectSession()

		evt := createdEvent(testAccountId, testTransactionId, testCreatedCharId)
		env.dispatchCreated(evt)
		env.dispatchCreated(evt)

		if len(env.destroyCalls) != 1 {
			t.Fatalf("destroy calls = %d, want exactly 1 across two deliveries", len(env.destroyCalls))
		}
	})
}

func TestSeedFailedLeavesItemAndAnnounces(t *testing.T) {
	t.Run("matched", func(t *testing.T) {
		env := newTestEnv(t)
		env.putSubmittedEntry(testTransactionId)
		env.connectSession()

		env.dispatchFailed(seedmsg.StatusEvent[seedmsg.FailedStatusEventBody]{
			AccountId:     testAccountId,
			TransactionId: testTransactionId,
			Type:          seedmsg.StatusEventTypeFailed,
			Body:          seedmsg.FailedStatusEventBody{Reason: "name_taken"},
		})

		if len(env.destroyCalls) != 0 {
			t.Fatalf("destroy calls = %d, want 0", len(env.destroyCalls))
		}
		arm, ok := env.lastArm()
		if !ok {
			t.Fatal("expected a MAPLELIFE_ERROR announce")
		}
		if arm != testByteUnknown {
			t.Errorf("arm = %#x, want UNKNOWN_ERROR %#x", arm, testByteUnknown)
		}
		if env.entryExists() {
			t.Error("registry entry should have been removed")
		}
		foundInfo := false
		for _, e := range env.hook.AllEntries() {
			if e.Data["reason"] == "name_taken" {
				foundInfo = true
			}
		}
		if !foundInfo {
			t.Errorf("expected the failure reason to be logged, got: %+v", env.hook.AllEntries())
		}
	})

	t.Run("fallback by account id", func(t *testing.T) {
		env := newTestEnv(t)
		env.putSubmittedEntry(testTransactionId)
		env.connectSession()

		env.dispatchFailed(seedmsg.StatusEvent[seedmsg.FailedStatusEventBody]{
			AccountId: testAccountId,
			Type:      seedmsg.StatusEventTypeFailed,
			Body:      seedmsg.FailedStatusEventBody{Reason: "name_taken"},
		})

		if len(env.destroyCalls) != 0 {
			t.Fatalf("destroy calls = %d, want 0", len(env.destroyCalls))
		}
		arm, ok := env.lastArm()
		if !ok {
			t.Fatal("expected a MAPLELIFE_ERROR announce")
		}
		if arm != testByteUnknown {
			t.Errorf("arm = %#x, want UNKNOWN_ERROR %#x", arm, testByteUnknown)
		}
		if env.entryExists() {
			t.Error("registry entry should have been removed")
		}
	})

	t.Run("unrelated account", func(t *testing.T) {
		env := newTestEnv(t)
		env.putSubmittedEntry(testTransactionId)
		env.connectSession()

		env.dispatchFailed(seedmsg.StatusEvent[seedmsg.FailedStatusEventBody]{
			AccountId: 8,
			Type:      seedmsg.StatusEventTypeFailed,
			Body:      seedmsg.FailedStatusEventBody{Reason: "name_taken"},
		})

		if len(env.destroyCalls) != 0 {
			t.Fatalf("destroy calls = %d, want 0", len(env.destroyCalls))
		}
		if len(env.announced) != 0 {
			t.Fatalf("announced = %d, want 0", len(env.announced))
		}
		if !env.entryExists() {
			t.Error("account 7's entry should be intact")
		}
	})
}

func TestSeedCreatedWithDisconnectedSessionStillConsumes(t *testing.T) {
	env := newTestEnv(t)
	env.putSubmittedEntry(testTransactionId)
	// Deliberately no connectSession() call -- account 7 has no live session.

	env.dispatchCreated(createdEvent(testAccountId, testTransactionId, testCreatedCharId))

	if len(env.destroyCalls) != 1 {
		t.Fatalf("destroy calls = %d, want 1 -- the entitlement was spent and the character exists regardless of session state", len(env.destroyCalls))
	}
	assertDestroySagaCorrect(t, env.destroyCalls[0])

	if len(env.announced) != 0 {
		t.Fatalf("announced = %d, want 0 -- no session to write to", len(env.announced))
	}

	foundInfo := false
	for _, e := range env.hook.AllEntries() {
		if e.Level == logrus.InfoLevel {
			foundInfo = true
		}
	}
	if !foundInfo {
		t.Errorf("expected an info-level log entry, got: %+v", env.hook.AllEntries())
	}
}

func TestSeedCreatedDestroyFailureIsLoggedNotRolledBack(t *testing.T) {
	env := newTestEnv(t)
	env.putSubmittedEntry(testTransactionId)
	env.connectSession()
	env.destroyErr = errors.New("saga producer unavailable")

	env.dispatchCreated(createdEvent(testAccountId, testTransactionId, testCreatedCharId))

	var errEntry *logrus.Entry
	for _, e := range env.hook.AllEntries() {
		if e.Level == logrus.ErrorLevel {
			errEntry = e
		}
	}
	if errEntry == nil {
		t.Fatalf("expected an error-level log entry, got: %+v", env.hook.AllEntries())
	}
	if errEntry.Data["account_id"] != testAccountId {
		t.Errorf("account_id = %v, want %d", errEntry.Data["account_id"], testAccountId)
	}
	if errEntry.Data["submitting_char_id"] != testSubmitCharId {
		t.Errorf("submitting_char_id = %v, want %d", errEntry.Data["submitting_char_id"], testSubmitCharId)
	}
	if errEntry.Data["created_char_id"] != testCreatedCharId {
		t.Errorf("created_char_id = %v, want %d", errEntry.Data["created_char_id"], testCreatedCharId)
	}
	if errEntry.Data["item_id"] != testItemId {
		t.Errorf("item_id = %v, want %d", errEntry.Data["item_id"], testItemId)
	}
	if errEntry.Data["submit_transaction_id"] != testTransactionId {
		t.Errorf("submit_transaction_id = %v, want %s", errEntry.Data["submit_transaction_id"], testTransactionId)
	}

	// No compensating character deletion is attempted -- there is nothing in
	// this package that could even construct one; the absence is structural,
	// not merely untested. The character-creation SUCCESS is still announced
	// because the character genuinely exists.
	arm, ok := env.lastArm()
	if !ok {
		t.Fatal("expected a MAPLELIFE_ERROR announce despite the destroy failure")
	}
	if arm != testByteSuccess {
		t.Errorf("arm = %#x, want SUCCESS %#x", arm, testByteSuccess)
	}
}
