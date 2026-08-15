package pendingchange

import (
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"net"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	chatcb "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const testCharacterId = uint32(1)

// testOperations supplies a resolved code for the "CANCELLED" operations-
// table key both CashShopCancelNameChangeResult and
// CashShopCancelTransferWorldResult resolve through (each writer gets its own
// options map at the real socket-writer registry; this test double shares
// one map since both writers' CANCELLED arms use the same key name and
// neither test asserts on the resolved byte value).
var testOperations = map[string]interface{}{
	cashcb.CancelNameChangeResultCancelled: float64(0x00),
}

// discardConn is a net.Conn whose Write swallows the encrypted frame.
// session.Model.announceEncrypted writes straight to the socket; the tests
// capture the packet at the writer seam instead, so the socket only has to
// not panic.
type discardConn struct{ net.Conn }

func (discardConn) Write(b []byte) (int, error) { return len(b), nil }
func (discardConn) Close() error                { return nil }

// announcement records one session.Announce call: which writer it went to
// and the encoded body bytes.
type announcement struct {
	writerName string
	body       []byte
}

type pendingChangeConsumerEnv struct {
	t         *testing.T
	logger    *logrus.Logger
	ctx       context.Context
	tenant    tenant.Model
	sc        server.Model
	wp        writer.Producer
	announced []announcement
}

// newPendingChangeConsumerEnv registers a server whose world/channel (0, 0)
// match session.NewSession's un-set default field, so
// IfPresentByCharacterId's world/channel filters actually match a
// directly-registered test session (mirrors atlas-channel's cashshop
// consumer_test.go newZeroFieldTestServer).
func newPendingChangeConsumerEnv(t *testing.T) *pendingChangeConsumerEnv {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l, _ := testlog.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	env := &pendingChangeConsumerEnv{
		t:      t,
		logger: l,
		ctx:    tenant.WithContext(context.Background(), tm),
		tenant: tm,
	}
	env.sc = server.NewProcessor(l, context.Background()).Register(tm, channelconst.NewModel(0, 0), "127.0.0.1", 8484)
	env.wp = env.capturingProducer()
	return env
}

func (e *pendingChangeConsumerEnv) capturingProducer() writer.Producer {
	options := map[string]interface{}{
		"operations": map[string]interface{}(testOperations),
	}
	return func(name string) (socketwriter.BodyFunc, error) {
		return func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(l, ctx)(options)
				e.announced = append(e.announced, announcement{writerName: name, body: b})
				return b
			}
		}, nil
	}
}

// characterOnline registers a live session for the character, backed by a
// real net.Conn (net.Pipe) whose client side is drained in the background so
// session.Announce's encrypted write never blocks.
func (e *pendingChangeConsumerEnv) characterOnline(characterId uint32) {
	e.t.Helper()
	serverConn, clientConn := net.Pipe()
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		buf := make([]byte, 4096)
		for {
			if _, err := clientConn.Read(buf); err != nil {
				return
			}
		}
	}()

	sessionId := uuid.New()
	s := session.NewSession(sessionId, e.tenant, 0, serverConn)
	session.AddSessionToRegistry(e.tenant.Id(), s)
	sp := session.NewProcessor(e.logger, e.ctx)
	_ = sp.SetAccountId(sessionId, 1)
	_ = sp.SetCharacterId(sessionId, characterId)

	e.t.Cleanup(func() {
		session.ClearRegistryForTenant(e.tenant.Id())
		_ = serverConn.Close()
		_ = clientConn.Close()
		<-drainDone
	})
}

// characterOffline is a no-op: the registry starts empty for every test, so
// simply not calling characterOnline leaves the character offline. Named for
// symmetry/readability at call sites.
func (e *pendingChangeConsumerEnv) characterOffline(_ uint32) {}

type resolvedEvent struct {
	CharacterId        uint32
	ChangeType         string
	Status             string
	Reason             string
	RequestedName      string
	DestinationWorldId world.Id
}

func (e *pendingChangeConsumerEnv) deliver(re resolvedEvent) {
	e.t.Helper()
	h := handleResolved(e.sc, e.wp)
	h(e.logger, e.ctx, StatusEvent[ResolvedEventBody]{
		CharacterId: re.CharacterId,
		WorldId:     e.sc.WorldId(),
		Type:        EventTypeResolved,
		Body: ResolvedEventBody{
			ChangeType:         re.ChangeType,
			Status:             re.Status,
			Reason:             re.Reason,
			RequestedName:      re.RequestedName,
			DestinationWorldId: re.DestinationWorldId,
		},
	})
}

func (e *pendingChangeConsumerEnv) wrote(writerName string) bool {
	for _, a := range e.announced {
		if a.writerName == writerName {
			return true
		}
	}
	return false
}

func (e *pendingChangeConsumerEnv) wrotePinkTextContaining(substr string) bool {
	for _, a := range e.announced {
		if a.writerName != chatcb.WorldMessageWriter {
			continue
		}
		if strings.Contains(string(a.body), substr) {
			return true
		}
	}
	return false
}

func (e *pendingChangeConsumerEnv) wroteAnyCancelPacket() bool {
	return e.wrote(cashcb.CashShopCancelNameChangeResultWriter) ||
		e.wrote(cashcb.CashShopCancelTransferWorldResultWriter) ||
		e.wrote(charcb.CancelNameChangeByOtherWriter)
}

func (e *pendingChangeConsumerEnv) wroteAnything() bool {
	return len(e.announced) != 0
}

// FR-2.9: an operator cancellation reaches an online player as the CANCEL_*
// packet AND as pink text (design §3.9 belt-and-braces for OQ-9).
func TestResolvedCancellationWritesBothThePacketAndPinkText(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOnline(testCharacterId)

	env.deliver(resolvedEvent{
		CharacterId: testCharacterId, ChangeType: ChangeTypeNameChange,
		Status: StatusCancelled, Reason: "operator_cancelled", RequestedName: "Whiskey",
	})

	if !env.wrote(cashcb.CashShopCancelNameChangeResultWriter) {
		t.Fatal("expected CashShopCancelNameChangeResult")
	}
	if !env.wrotePinkTextContaining("Whiskey") {
		t.Fatal("expected pink text naming the requested value")
	}
}

// FR-2.7: a name change invalidated because someone else took the name uses
// the BY_OTHER packet specifically, not the generic cancel result.
func TestNameTakenRejectionUsesCancelByOther(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOnline(testCharacterId)

	env.deliver(resolvedEvent{
		CharacterId: testCharacterId, ChangeType: ChangeTypeNameChange,
		Status: StatusRejected, Reason: ReasonNameTaken, RequestedName: "Xray",
	})

	if !env.wrote(charcb.CancelNameChangeByOtherWriter) {
		t.Fatal("expected CancelNameChangeByOther")
	}
	if env.wrote(cashcb.CashShopCancelNameChangeResultWriter) {
		t.Fatal("must not also send the generic cancel result")
	}
	if !env.wrotePinkTextContaining("Xray") {
		t.Fatal("expected pink text naming the requested value — CANCEL_NAME_CHANGE_BY_OTHER's body is empty")
	}
}

func TestWorldTransferResolutionUsesTheTransferCancelPacket(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOnline(testCharacterId)

	env.deliver(resolvedEvent{
		CharacterId: testCharacterId, ChangeType: ChangeTypeWorldTransfer,
		Status: StatusExpired, Reason: "expired", DestinationWorldId: world.Id(2),
	})

	if !env.wrote(cashcb.CashShopCancelTransferWorldResultWriter) {
		t.Fatal("expected CashShopCancelTransferWorldResult")
	}
}

// An APPLIED resolution is not a cancellation and must send neither.
func TestAppliedResolutionSendsNoCancelPacket(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOnline(testCharacterId)

	env.deliver(resolvedEvent{CharacterId: testCharacterId, ChangeType: ChangeTypeNameChange, Status: StatusApplied})

	if env.wroteAnyCancelPacket() {
		t.Fatal("APPLIED must not produce a cancel notification")
	}
	if env.wroteAnything() {
		t.Fatal("APPLIED must not produce any notification at all, including pink text")
	}
}

// No live session: nothing is written, so atlas-character leaves notified_at
// null and re-emits at the player's next login (RenotifyForCharacter, Task 9).
func TestOfflineCharacterWritesNothing(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOffline(testCharacterId)

	env.deliver(resolvedEvent{
		CharacterId: testCharacterId, ChangeType: ChangeTypeNameChange,
		Status: StatusCancelled, Reason: "operator_cancelled",
	})

	if env.wroteAnything() {
		t.Fatal("expected no writes for an offline character")
	}
}

// A resolution for another tenant/world must be ignored.
func TestResolvedIgnoresOtherWorld(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOnline(testCharacterId)

	h := handleResolved(env.sc, env.wp)
	h(env.logger, env.ctx, StatusEvent[ResolvedEventBody]{
		CharacterId: testCharacterId,
		WorldId:     world.Id(99),
		Type:        EventTypeResolved,
		Body: ResolvedEventBody{
			ChangeType: ChangeTypeNameChange, Status: StatusCancelled, Reason: "operator_cancelled", RequestedName: "Zulu",
		},
	})

	if env.wroteAnything() {
		t.Fatal("a resolution routed to a different world must be ignored")
	}
}
