package cashshop

import (
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	cashshop2 "atlas-channel/kafka/message/cashshop"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	testAccountId   = uint32(4242)
	testCharacterId = uint32(7)
)

// The mode / reason bytes below are the gms_v83 template's real values
// (services/atlas-configurations/seed-data/templates/template_gms_83_1.json:
// operations USE_COUPON_SUCCESS 89, USE_COUPON_FAILED 92,
// INVENTORY_CAPACITY_INCREASE_FAILED 87; errors COUPON_EXPIRED 178). They are
// fed to the encoder through the writer options map exactly the way the live
// socket writer config feeds them, so the handlers must resolve them rather
// than hard-code anything (DOM-25).
var testOperations = map[string]interface{}{
	cashpkt.CashShopOperationUseCouponDone:                    float64(89),
	cashpkt.CashShopOperationUseCouponFailed:                  float64(92),
	cashpkt.CashShopOperationInventoryCapacityIncreaseFailed:  float64(87),
	cashpkt.CashShopOperationInventoryCapacityIncreaseSuccess: float64(86),
	cashpkt.CashShopOperationPurchaseSuccess:                  float64(66),
}

var testErrors = map[string]interface{}{
	"COUPON_EXPIRED":      float64(178),
	"INVALID_COUPON_CODE": float64(176),
}

// announcement records one session.Announce call: which writer it went to and
// the encoded body bytes (pre-encryption, post-Encode).
type announcement struct {
	writerName string
	body       []byte
}

// discardConn is a net.Conn whose Write swallows the encrypted frame.
// session.Model.announceEncrypted writes straight to the socket; the tests
// capture the packet at the writer seam instead, so the socket only has to
// not panic.
type discardConn struct{ net.Conn }

func (discardConn) Write(b []byte) (int, error) { return len(b), nil }
func (discardConn) Close() error                { return nil }

type consumerEnv struct {
	t           *testing.T
	logger      *logrus.Logger
	logHook     *logtest.Hook
	ctx         context.Context
	tenant      tenant.Model
	sc          server.Model
	wp          writer.Producer
	announced   []announcement
	assets      map[string]string
	walletDoc   string
	compartment uuid.UUID
}

func newConsumerEnv(t *testing.T) *consumerEnv {
	t.Helper()

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l, hook := logtest.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	env := &consumerEnv{
		t:           t,
		logger:      l,
		logHook:     hook,
		ctx:         tenant.WithContext(context.Background(), tm),
		tenant:      tm,
		assets:      make(map[string]string),
		compartment: uuid.New(),
	}
	env.walletDoc = walletDoc(1000, 2000, 3000)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if strings.HasSuffix(r.URL.Path, "/wallet") {
			_, _ = w.Write([]byte(env.walletDoc))
			return
		}
		if doc, ok := env.assets[r.URL.Path]; ok {
			_, _ = w.Write([]byte(doc))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"status":"404","title":"Not Found"}]}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/")

	env.sc = server.NewProcessor(l, context.Background()).Register(tm, channelconst.NewModel(0, 0), "127.0.0.1", 8484)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, tm, 0, discardConn{})
	session.AddSessionToRegistry(tm.Id(), s)
	t.Cleanup(func() { session.ClearRegistryForTenant(tm.Id()) })
	sp := session.NewProcessor(l, env.ctx)
	_ = sp.SetAccountId(sessionId, testAccountId)
	if r := sp.SetCharacterId(sessionId, testCharacterId); r.CharacterId() != testCharacterId {
		t.Fatalf("SetCharacterId: got %d, want %d", r.CharacterId(), testCharacterId)
	}

	env.wp = env.capturingProducer()
	return env
}

// capturingProducer stands in for the socket writer registry. It resolves the
// same options map the live writer config would supply and records every
// announced packet.
func (e *consumerEnv) capturingProducer() writer.Producer {
	options := map[string]interface{}{
		"operations": map[string]interface{}(testOperations),
		"errors":     map[string]interface{}(testErrors),
	}
	return func(name string) (writer.BodyFunc, error) {
		return func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(l, ctx)(options)
				e.announced = append(e.announced, announcement{writerName: name, body: b})
				return b
			}
		}, nil
	}
}

func (e *consumerEnv) seedAsset(compartmentId uuid.UUID, assetId uint32) {
	e.t.Helper()
	path := fmt.Sprintf("/accounts/%d/cash-shop/inventory/compartments/%s/assets/%d", testAccountId, compartmentId.String(), assetId)
	e.assets[path] = assetDoc(assetId, compartmentId)
}

func (e *consumerEnv) announcedWriters() []string {
	names := make([]string, 0, len(e.announced))
	for _, a := range e.announced {
		names = append(names, a.writerName)
	}
	return names
}

func (e *consumerEnv) lastAnnounced() announcement {
	e.t.Helper()
	if len(e.announced) == 0 {
		e.t.Fatal("nothing was announced")
	}
	return e.announced[len(e.announced)-1]
}

// lastAnnouncedMode reads the leading mode byte of the last announced
// CashShopOperation body.
func (e *consumerEnv) lastAnnouncedMode() byte {
	e.t.Helper()
	a := e.lastAnnounced()
	if len(a.body) < 1 {
		e.t.Fatalf("announced body too short: %v", a.body)
	}
	return a.body[0]
}

// lastAnnouncedReasonByte reads the reason byte that follows the mode byte on
// the failure arms.
func (e *consumerEnv) lastAnnouncedReasonByte() byte {
	e.t.Helper()
	a := e.lastAnnounced()
	if len(a.body) < 2 {
		e.t.Fatalf("announced body too short: %v", a.body)
	}
	return a.body[1]
}

func (e *consumerEnv) modeFor(key string) byte {
	e.t.Helper()
	v, ok := testOperations[key]
	if !ok {
		e.t.Fatalf("operations key [%s] not in the test options map", key)
	}
	return byte(v.(float64))
}

func (e *consumerEnv) errorByteFor(key string) byte {
	e.t.Helper()
	v, ok := testErrors[key]
	if !ok {
		e.t.Fatalf("errors key [%s] not in the test options map", key)
	}
	return byte(v.(float64))
}

// decodeUseCouponDone decodes the first announced CashShopOperation body back
// into a UseCouponDone so the test can assert against the real fields rather
// than raw offsets.
func (e *consumerEnv) decodeUseCouponDone(t *testing.T) cashpkt.UseCouponDone {
	t.Helper()
	for _, a := range e.announced {
		if a.writerName != cashpkt.CashShopOperationWriter {
			continue
		}
		m := cashpkt.UseCouponDone{}
		req := request.Request(a.body)
		r := request.NewRequestReader(&req, 0)
		m.Decode(e.logger, e.ctx)(&r, nil)
		return m
	}
	t.Fatal("no CashShopOperation packet was announced")
	return cashpkt.UseCouponDone{}
}

// reservedReasonBytes is the union of every version's reason bytes that change
// client state instead of showing a notice. Sourced from
// docs/tasks/task-206-cash-shop-coupon-codes/derivation.md:
// v48 112/114 (line 1408-1409), v61 127/129 (1548-1549), v72 140/142
// (1688-1689), v79 154/156 (1821-1822), v83 162/164/177 (261-263),
// v84 171/173/186 (424, 487-489), v87 177/179/192 (643-645),
// v92 0/2/15 (697-699), v95 identical to v92 (947), jms_v185 177/179
// (1270-1273; jms has no OnStatusExit byte).
type reservedByteSet map[byte][]string

func (s reservedByteSet) excludes(b byte) bool {
	_, present := s[b]
	return !present
}

func (e *consumerEnv) reservedReasonBytes() reservedByteSet {
	return reservedByteSet{
		112: {"gms_v48"}, 114: {"gms_v48"},
		127: {"gms_v61"}, 129: {"gms_v61"},
		140: {"gms_v72"}, 142: {"gms_v72"},
		154: {"gms_v79"}, 156: {"gms_v79"},
		162: {"gms_v83"}, 164: {"gms_v83"},
		171: {"gms_v84"}, 173: {"gms_v84"}, 186: {"gms_v84"},
		177: {"gms_v83", "gms_v87", "jms_v185"}, 179: {"gms_v87", "jms_v185"}, 192: {"gms_v87"},
		0: {"gms_v92", "gms_v95"}, 2: {"gms_v92", "gms_v95"}, 15: {"gms_v92", "gms_v95"},
	}
}

func assetDoc(assetId uint32, compartmentId uuid.UUID) string {
	return fmt.Sprintf(`{"data":{"type":"assets","id":"%d","attributes":{"compartmentId":"%s","cashId":"9001","templateId":5000000,"commodityId":10001,"quantity":1,"flag":0,"purchasedBy":%d,"expiration":"%s"}}}`,
		assetId, compartmentId.String(), testAccountId, time.Unix(0, 0).UTC().Format(time.RFC3339))
}

func walletDoc(credit uint32, points uint32, prepaid uint32) string {
	return fmt.Sprintf(`{"data":{"type":"wallets","id":"%s","attributes":{"accountId":%d,"credit":%d,"points":%d,"prepaid":%d}}}`,
		uuid.New().String(), testAccountId, credit, points, prepaid)
}

// TestCouponRedeemedAnnouncesSuccessAndRefreshesTheWallet pins the success
// path: the USE_COUPON_SUCCESS arm carries the awarded items and the
// maplePoint DELTA, and CashQueryResult follows so an open Cash Shop window
// picks up the new balance without a relog.
func TestCouponRedeemedAnnouncesSuccessAndRefreshesTheWallet(t *testing.T) {
	env := newConsumerEnv(t)
	assetId := uint32(555)
	env.seedAsset(env.compartment, assetId)

	handleStatusEventCouponRedeemed(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.CouponRedeemedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeCouponRedeemed,
		Body: cashshop2.CouponRedeemedBody{
			CompartmentId: env.compartment,
			AssetIds:      []uint32{assetId},
			MaplePoints:   1500,
		},
	})

	// USE_COUPON_SUCCESS first, then CASH_QUERY_RESULT so the open Cash Shop
	// window shows the new balance without a relog.
	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter, cashpkt.CashQueryResultWriter}) {
		t.Errorf("announced %v", got)
	}
	body := env.decodeUseCouponDone(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationUseCouponDone) {
		t.Errorf("mode = %d, want the USE_COUPON_SUCCESS mode %d", body.Mode(), env.modeFor(cashpkt.CashShopOperationUseCouponDone))
	}
	if body.MaplePoint() != 1500 {
		t.Errorf("maplePoint = %d, want the 1500 DELTA the event carried", body.MaplePoint())
	}
	if len(body.Items()) != 1 {
		t.Errorf("items = %d, want 1", len(body.Items()))
	}
	if body.Meso() != 0 {
		t.Errorf("meso = %d, want 0 — meso rewards are out of scope", body.Meso())
	}
}

// TestCouponRedeemedCurrencyOnlyStillAnnounces pins the contract that a
// currency-only coupon — no assets, and therefore the ZERO compartment uuid —
// is a normal success, not an error to skip. Consumers key off AssetIds, never
// off CompartmentId.
func TestCouponRedeemedCurrencyOnlyStillAnnounces(t *testing.T) {
	env := newConsumerEnv(t)

	handleStatusEventCouponRedeemed(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.CouponRedeemedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeCouponRedeemed,
		Body: cashshop2.CouponRedeemedBody{
			CompartmentId: uuid.Nil,
			AssetIds:      nil,
			MaplePoints:   250,
		},
	})

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter, cashpkt.CashQueryResultWriter}) {
		t.Errorf("announced %v", got)
	}
	body := env.decodeUseCouponDone(t)
	if len(body.Items()) != 0 {
		t.Errorf("items = %d, want 0", len(body.Items()))
	}
	if body.MaplePoint() != 250 {
		t.Errorf("maplePoint = %d, want 250", body.MaplePoint())
	}
}

// TestCouponFailedAnnouncesOnTheCouponArm pins that a coupon failure goes out
// on the USE_COUPON_FAILED mode, not the capacity-increase mode the generic
// ERROR handler uses.
func TestCouponFailedAnnouncesOnTheCouponArm(t *testing.T) {
	env := newConsumerEnv(t)
	handleStatusEventCouponFailed(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.CouponFailedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeCouponFailed,
		Body:        cashshop2.CouponFailedBody{Error: "COUPON_EXPIRED"},
	})
	if got := env.lastAnnounced().writerName; got != cashpkt.CashShopOperationWriter {
		t.Errorf("writer = %s, want %s", got, cashpkt.CashShopOperationWriter)
	}
	if got := env.lastAnnouncedMode(); got != env.modeFor(cashpkt.CashShopOperationUseCouponFailed) {
		t.Errorf("mode = %d, want the USE_COUPON_FAILED mode %d", got, env.modeFor(cashpkt.CashShopOperationUseCouponFailed))
	}
	if got := env.lastAnnouncedMode(); got == env.modeFor(cashpkt.CashShopOperationInventoryCapacityIncreaseFailed) {
		t.Errorf("mode = %d, which is the capacity-increase failure arm the generic ERROR handler uses", got)
	}
	if got := env.lastAnnouncedReasonByte(); got != env.errorByteFor("COUPON_EXPIRED") {
		t.Errorf("reason = %d, want %d — did the template errors table generate?", got, env.errorByteFor("COUPON_EXPIRED"))
	}
}

// TestCouponFailedUnknownErrorFallsThroughToTheDefaultNotice pins the
// deliberate absence of an UNKNOWN_ERROR key: it is the client jump table's
// DEFAULT case on every version, so CashShopUseCouponFailedBody short-circuits
// it to the documented default byte (99) before it ever reaches ResolveCode's
// generic "errors" lookup — see couponUnknownErrorDefaultByte in
// libs/atlas-packet/cash/clientbound/shop_operation_body.go. 99 is itself
// unlisted and therefore renders the client's default notice — the intended
// outcome. Nobody should "fix" this by adding a bogus UNKNOWN_ERROR key.
func TestCouponFailedUnknownErrorFallsThroughToTheDefaultNotice(t *testing.T) {
	env := newConsumerEnv(t)
	handleStatusEventCouponFailed(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.CouponFailedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeCouponFailed,
		Body:        cashshop2.CouponFailedBody{Error: "UNKNOWN_ERROR"},
	})
	if got := env.lastAnnouncedReasonByte(); got != 99 {
		t.Errorf("reason = %d, want 99 (unconfigured -> default notice)", got)
	}
	if !env.reservedReasonBytes().excludes(99) {
		t.Fatalf("99 collides with a reserved reason byte on %v — pick an explicit mapped key instead", env.reservedReasonBytes()[99])
	}
}

// TestCouponFailedUnknownErrorDoesNotLogAtErrorLevel guards the operational
// fix: UNKNOWN_ERROR is an ordinary path (missing locker row, transaction
// failure), not a misconfiguration, so it must never emit ResolveCode's
// generic ERROR-level "Defaulting to 99 which will likely cause a client
// crash" line — that wording is correct for a genuinely unconfigured opcode
// but sends operators chasing a client crash that will not happen here.
func TestCouponFailedUnknownErrorDoesNotLogAtErrorLevel(t *testing.T) {
	env := newConsumerEnv(t)
	handleStatusEventCouponFailed(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.CouponFailedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeCouponFailed,
		Body:        cashshop2.CouponFailedBody{Error: "UNKNOWN_ERROR"},
	})
	for _, e := range env.logHook.AllEntries() {
		if e.Level <= logrus.WarnLevel {
			t.Errorf("unexpected %s-level log for the UNKNOWN_ERROR default-notice path: %q", e.Level, e.Message)
		}
		if strings.Contains(e.Message, "will likely cause a client crash") {
			t.Errorf("UNKNOWN_ERROR must not fall through ResolveCode's generic crash-warning fallback, got: %q", e.Message)
		}
	}
}
