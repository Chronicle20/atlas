package wallet

import (
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"net"
	"reflect"
	"testing"

	walletmsg "atlas-channel/kafka/message/wallet"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	testAccountId   = uint32(4242)
	testCharacterId = uint32(7)
)

// discardConn is a net.Conn whose Write swallows the encrypted frame.
type discardConn struct{ net.Conn }

func (discardConn) Write(b []byte) (int, error) { return len(b), nil }
func (discardConn) Close() error                { return nil }

type consumerEnv struct {
	t         *testing.T
	logger    *logrus.Logger
	ctx       context.Context
	tenant    tenant.Model
	sc        server.Model
	wp        writer.Producer
	sessionId uuid.UUID
	announced []string
}

func newConsumerEnv(t *testing.T, cashScene byte) *consumerEnv {
	t.Helper()

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l, _ := testlog.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	env := &consumerEnv{
		t:      t,
		logger: l,
		ctx:    tenant.WithContext(context.Background(), tm),
		tenant: tm,
	}

	env.sc = server.NewProcessor(l, context.Background()).Register(tm, channelconst.NewModel(0, 0), "127.0.0.1", 8484)

	sessionId := uuid.New()
	env.sessionId = sessionId
	s := session.NewSession(sessionId, tm, 0, discardConn{})
	session.AddSessionToRegistry(tm.Id(), s)
	t.Cleanup(func() { session.ClearRegistryForTenant(tm.Id()) })
	sp := session.NewProcessor(l, env.ctx)
	_ = sp.SetAccountId(sessionId, testAccountId)
	if r := sp.SetCharacterId(sessionId, testCharacterId); r.CharacterId() != testCharacterId {
		t.Fatalf("SetCharacterId: got %d, want %d", r.CharacterId(), testCharacterId)
	}
	if cashScene != session.CashSceneNone {
		sp.SetCashScene(sessionId, cashScene)
	}

	env.wp = env.capturingProducer()
	return env
}

// capturingProducer resolves both writers this consumer can invoke and
// records which one (if any) fired.
func (e *consumerEnv) capturingProducer() writer.Producer {
	return func(name string) (writer.BodyFunc, error) {
		return func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				e.announced = append(e.announced, name)
				return encoder(l, ctx)(map[string]interface{}{})
			}
		}, nil
	}
}

func TestHandleWalletUpdatedCashShopSceneRefreshOwnedSkipsAnnounce(t *testing.T) {
	env := newConsumerEnv(t, session.CashSceneCashShop)

	handleWalletUpdated(env.sc, env.wp)(env.logger, env.ctx, walletmsg.StatusEvent[walletmsg.StatusEventUpdatedBody]{
		AccountId: testAccountId,
		Type:      walletmsg.StatusEventTypeUpdated,
		Body: walletmsg.StatusEventUpdatedBody{
			Credit:            1000,
			Points:            2000,
			Prepaid:           3000,
			TransactionId:     uuid.New(),
			SceneRefreshOwned: true,
		},
	})

	if len(env.announced) != 0 {
		t.Fatalf("announced = %v, want nothing when SceneRefreshOwned is set", env.announced)
	}
}

func TestHandleWalletUpdatedCashShopSceneRefreshUnownedAnnounces(t *testing.T) {
	env := newConsumerEnv(t, session.CashSceneCashShop)

	handleWalletUpdated(env.sc, env.wp)(env.logger, env.ctx, walletmsg.StatusEvent[walletmsg.StatusEventUpdatedBody]{
		AccountId: testAccountId,
		Type:      walletmsg.StatusEventTypeUpdated,
		Body: walletmsg.StatusEventUpdatedBody{
			Credit:            1000,
			Points:            2000,
			Prepaid:           3000,
			SceneRefreshOwned: false,
		},
	})

	if got := env.announced; !reflect.DeepEqual(got, []string{cashcb.CashQueryResultWriter}) {
		t.Fatalf("announced = %v, want exactly [%s]", got, cashcb.CashQueryResultWriter)
	}
}

// TestHandleWalletUpdatedNonNilTransactionIdWithoutFlagStillAnnounces pins the
// ruling against the rejected TransactionId heuristic (bug-round-2-gift-notice
// -step2-ruling.md): a non-Nil TransactionId alone must NOT suppress the
// refresh -- only the explicit SceneRefreshOwned flag does.
func TestHandleWalletUpdatedNonNilTransactionIdWithoutFlagStillAnnounces(t *testing.T) {
	env := newConsumerEnv(t, session.CashSceneCashShop)

	handleWalletUpdated(env.sc, env.wp)(env.logger, env.ctx, walletmsg.StatusEvent[walletmsg.StatusEventUpdatedBody]{
		AccountId: testAccountId,
		Type:      walletmsg.StatusEventTypeUpdated,
		Body: walletmsg.StatusEventUpdatedBody{
			Credit:            1000,
			Points:            2000,
			Prepaid:           3000,
			TransactionId:     uuid.New(),
			SceneRefreshOwned: false,
		},
	})

	if got := env.announced; !reflect.DeepEqual(got, []string{cashcb.CashQueryResultWriter}) {
		t.Fatalf("announced = %v, want exactly [%s] -- a non-Nil TransactionId alone must not suppress the refresh", got, cashcb.CashQueryResultWriter)
	}
}

func TestHandleWalletUpdatedMtsSceneUnaffectedBySceneRefreshOwned(t *testing.T) {
	for _, owned := range []bool{true, false} {
		env := newConsumerEnv(t, session.CashSceneMts)

		handleWalletUpdated(env.sc, env.wp)(env.logger, env.ctx, walletmsg.StatusEvent[walletmsg.StatusEventUpdatedBody]{
			AccountId: testAccountId,
			Type:      walletmsg.StatusEventTypeUpdated,
			Body: walletmsg.StatusEventUpdatedBody{
				Credit:            1000,
				Points:            2000,
				Prepaid:           3000,
				TransactionId:     uuid.New(),
				SceneRefreshOwned: owned,
			},
		})

		if got := env.announced; !reflect.DeepEqual(got, []string{fieldcb.MtsOperation2Writer}) {
			t.Fatalf("SceneRefreshOwned=%v: announced = %v, want exactly [%s] -- the MTS arm must not be guarded by this flag", owned, got, fieldcb.MtsOperation2Writer)
		}
	}
}
