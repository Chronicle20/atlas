package cashshop

import (
	"atlas-channel/pendingchange"
	"atlas-channel/ring"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	cashshop2 "atlas-channel/kafka/message/cashshop"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// gachaponOperations mirrors the per-writer operations table
// CashItemGachaponSuccessBody/FailedBody resolve their mode byte from
// (WithResolvedCode, libs/atlas-packet/resolve.go). Values are arbitrary but
// distinct, only used to prove the correct code was selected.
var gachaponOperations = map[string]interface{}{
	cashpkt.CashItemGachaponModeSuccess: float64(0xC1),
	cashpkt.CashItemGachaponModeFailed:  float64(0xC0),
}

func nullLogger() *logrus.Logger {
	l, _ := testlog.NewNullLogger()
	return l
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// newZeroFieldTestServer registers a server whose world/channel (0, 0) match
// session.NewSession's un-set default field, so IfPresentByCharacterId's
// world/channel filters actually match a directly-registered test session.
func newZeroFieldTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, 0)
	return server.NewProcessor(nullLogger(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// recordingWriterProducer stubs writer.Producer: it resolves ONLY
// cashpkt.CashItemGachaponResultWriter, invokes the real encoder against
// gachaponOperations (exercising the real DOM-25 WithResolvedCode path), and
// records the produced wire bytes so the test can decode them.
func recordingWriterProducer(t *testing.T) (writer.Producer, *[][]byte) {
	t.Helper()
	var captured [][]byte
	wp := writer.Producer(func(name string) (socketwriter.BodyFunc, error) {
		if name != cashpkt.CashItemGachaponResultWriter {
			return nil, fmt.Errorf("unexpected writer requested: %s", name)
		}
		return func(l logrus.FieldLogger, ctx context.Context) func(packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(l, ctx)(map[string]interface{}{"operations": gachaponOperations})
				captured = append(captured, b)
				return b
			}
		}, nil
	})
	return wp, &captured
}

// registerRealSession registers a session backed by a real net.Conn
// (net.Pipe), draining the client side in the background so
// session.Announce's announceEncrypted write never blocks. Returns a cleanup
// func.
func registerRealSession(t *testing.T, tm tenant.Model, characterId uint32, accountId uint32) func() {
	t.Helper()
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
	s := session.NewSession(sessionId, tm, 0, serverConn)
	session.AddSessionToRegistry(tm.Id(), s)
	ctx := tenant.WithContext(context.Background(), tm)
	_ = session.NewProcessor(nullLogger(), ctx).SetCharacterId(sessionId, characterId)
	_ = session.NewProcessor(nullLogger(), ctx).SetAccountId(sessionId, accountId)

	return func() {
		session.ClearRegistryForTenant(tm.Id())
		_ = serverConn.Close()
		_ = clientConn.Close()
		<-drainDone
	}
}

// assetFixtureServer stands up a JSON:API server serving one "assets"
// resource at accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}/assets/{assetId},
// matching the RestModel type/tags in cashshop/inventory/asset/rest.go.
func assetFixtureServer(t *testing.T, accountId uint32, compartmentId uuid.UUID, assetId uint32, cashId int64, templateId uint32, commodityId uint32, quantity uint32) *httptest.Server {
	t.Helper()
	wantPath := fmt.Sprintf("/api/accounts/%d/cash-shop/inventory/compartments/%s/assets/%d", accountId, compartmentId.String(), assetId)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected asset fetch path: got %s want %s", r.URL.Path, wantPath)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		body := fmt.Sprintf(
			`{"data":{"type":"assets","id":"%d","attributes":{"compartmentId":%q,"cashId":"%d","templateId":%d,"commodityId":%d,"quantity":%d,"flag":0,"purchasedBy":%d,"expiration":%q}}}`,
			assetId, compartmentId.String(), cashId, templateId, commodityId, quantity, accountId, time.Unix(0, 0).UTC().Format(time.RFC3339),
		)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/api/")
	return srv
}

func decodeGachaponSuccess(t *testing.T, b []byte) cashpkt.CashItemGachaponSuccess {
	t.Helper()
	l := nullLogger()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var m cashpkt.CashItemGachaponSuccess
	m.Decode(l, context.Background())(&reader, nil)
	return m
}

func decodeGachaponFailed(t *testing.T, b []byte) cashpkt.CashItemGachaponFailed {
	t.Helper()
	l := nullLogger()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var m cashpkt.CashItemGachaponFailed
	m.Decode(l, context.Background())(&reader, nil)
	return m
}

// The SUCCESS arm's `sn` is the BOX's cash serial, not the reward's: the
// client uses it to locate the locker row to decrement or remove.
func TestHandleSurpriseOpenedAnnouncesBoxSn(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newZeroFieldTestServer(t, tm)

	characterId := uint32(6001)
	accountId := uint32(9001)
	compartmentId := uuid.New()
	rewardAssetId := uint32(500)

	cleanup := registerRealSession(t, tm, characterId, accountId)
	defer cleanup()

	assetFixtureServer(t, accountId, compartmentId, rewardAssetId, 777777, 5010000, 12345, 3)

	wp, captured := recordingWriterProducer(t)

	h := handleStatusEventSurpriseOpened(sc, wp)
	h(nullLogger(), ctx, cashshop2.StatusEvent[cashshop2.SurpriseOpenedEventBody]{
		CharacterId: characterId,
		Type:        cashshop2.StatusEventTypeSurpriseOpened,
		Body: cashshop2.SurpriseOpenedEventBody{
			CompartmentId:    compartmentId,
			BoxCashId:        123456789,
			BoxRemaining:     4,
			RewardAssetId:    rewardAssetId,
			RewardTemplateId: 5010000,
			RewardCount:      1,
		},
	})

	if len(*captured) != 1 {
		t.Fatalf("want exactly 1 announce, got %d", len(*captured))
	}
	res := decodeGachaponSuccess(t, (*captured)[0])
	if res.SN() != 123456789 {
		t.Fatalf("sn: got %d, want the BOX cash id 123456789 (not the reward's cashId 777777)", res.SN())
	}
	if res.Mode() != 0xC1 {
		t.Fatalf("mode: got %#x, want resolved SUCCESS code 0xC1", res.Mode())
	}
}

// remain 0 is how the client is told to remove the row entirely.
func TestHandleSurpriseOpenedCarriesZeroRemainOnLastBox(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newZeroFieldTestServer(t, tm)

	characterId := uint32(6002)
	accountId := uint32(9002)
	compartmentId := uuid.New()
	rewardAssetId := uint32(501)

	cleanup := registerRealSession(t, tm, characterId, accountId)
	defer cleanup()

	assetFixtureServer(t, accountId, compartmentId, rewardAssetId, 888888, 5010001, 12346, 1)

	wp, captured := recordingWriterProducer(t)

	h := handleStatusEventSurpriseOpened(sc, wp)
	h(nullLogger(), ctx, cashshop2.StatusEvent[cashshop2.SurpriseOpenedEventBody]{
		CharacterId: characterId,
		Type:        cashshop2.StatusEventTypeSurpriseOpened,
		Body: cashshop2.SurpriseOpenedEventBody{
			CompartmentId:    compartmentId,
			BoxCashId:        223456789,
			BoxRemaining:     0,
			RewardAssetId:    rewardAssetId,
			RewardTemplateId: 5010001,
			RewardCount:      1,
		},
	})

	if len(*captured) != 1 {
		t.Fatalf("want exactly 1 announce, got %d", len(*captured))
	}
	res := decodeGachaponSuccess(t, (*captured)[0])
	if res.Remain() != 0 {
		t.Fatalf("remain: got %d, want 0 (BoxRemaining)", res.Remain())
	}
}

// The FAILED arm is mode-only; the event's Reason is server-side and must
// NOT appear on the wire.
func TestHandleSurpriseFailedAnnouncesModeOnly(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newZeroFieldTestServer(t, tm)

	characterId := uint32(6003)
	accountId := uint32(9003)

	cleanup := registerRealSession(t, tm, characterId, accountId)
	defer cleanup()

	wp, captured := recordingWriterProducer(t)

	h := handleStatusEventSurpriseFailed(sc, wp)
	h(nullLogger(), ctx, cashshop2.StatusEvent[cashshop2.SurpriseFailedEventBody]{
		CharacterId: characterId,
		Type:        cashshop2.StatusEventTypeSurpriseFailed,
		Body: cashshop2.SurpriseFailedEventBody{
			Reason: "POOL_EMPTY",
		},
	})

	if len(*captured) != 1 {
		t.Fatalf("want exactly 1 announce, got %d", len(*captured))
	}
	b := (*captured)[0]
	if len(b) != 1 {
		t.Fatalf("FAILED arm must be mode-only (1 byte on the wire), got %d bytes: %v", len(b), b)
	}
	res := decodeGachaponFailed(t, b)
	if res.Mode() != 0xC0 {
		t.Fatalf("mode: got %#x, want resolved FAILED code 0xC0", res.Mode())
	}
}

// A status event for another tenant must be ignored.
func TestHandleSurpriseOpenedIgnoresOtherTenants(t *testing.T) {
	tm := newTestTenant(t)
	otherTenant := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), otherTenant)
	sc := newZeroFieldTestServer(t, tm)

	characterId := uint32(6004)
	accountId := uint32(9004)
	compartmentId := uuid.New()
	rewardAssetId := uint32(502)

	cleanup := registerRealSession(t, tm, characterId, accountId)
	defer cleanup()

	// No asset fixture server is stood up: a wrongly-routed event that
	// reaches the asset fetch would fail loudly rather than silently pass.
	wp, captured := recordingWriterProducer(t)

	h := handleStatusEventSurpriseOpened(sc, wp)
	h(nullLogger(), ctx, cashshop2.StatusEvent[cashshop2.SurpriseOpenedEventBody]{
		CharacterId: characterId,
		Type:        cashshop2.StatusEventTypeSurpriseOpened,
		Body: cashshop2.SurpriseOpenedEventBody{
			CompartmentId:    compartmentId,
			BoxCashId:        323456789,
			BoxRemaining:     2,
			RewardAssetId:    rewardAssetId,
			RewardTemplateId: 5010002,
			RewardCount:      1,
		},
	})

	if len(*captured) != 0 {
		t.Fatalf("other-tenant event must be ignored, got %d announces", len(*captured))
	}
}

// A status event of another type on the shared EVENT_TOPIC_CASH_SHOP_STATUS
// topic must be ignored by each surprise handler — the type guard that
// prevents a mismatched body from being announced.
func TestHandleSurpriseOpenedIgnoresWrongType(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newZeroFieldTestServer(t, tm)

	characterId := uint32(6005)
	accountId := uint32(9005)
	compartmentId := uuid.New()
	rewardAssetId := uint32(503)

	cleanup := registerRealSession(t, tm, characterId, accountId)
	defer cleanup()

	// Stand up a valid asset fixture so that, if the type guard were removed,
	// the asset fetch at consumer.go:176 would SUCCEED and a packet would be
	// captured. Without this, an unset CASHSHOP_SERVICE_URL makes the fetch
	// fail for an unrelated reason, and "0 announces" no longer proves the
	// guard did the work — see task-15 fix-round-1 finding.
	assetFixtureServer(t, accountId, compartmentId, rewardAssetId, 999999, 5010003, 12347, 1)

	wp, captured := recordingWriterProducer(t)

	h := handleStatusEventSurpriseOpened(sc, wp)
	h(nullLogger(), ctx, cashshop2.StatusEvent[cashshop2.SurpriseOpenedEventBody]{
		CharacterId: characterId,
		Type:        cashshop2.StatusEventTypeSurpriseFailed, // wrong type for this handler
		Body: cashshop2.SurpriseOpenedEventBody{
			CompartmentId:    compartmentId,
			BoxCashId:        423456789,
			BoxRemaining:     1,
			RewardAssetId:    rewardAssetId,
			RewardTemplateId: 5010003,
			RewardCount:      1,
		},
	})

	if len(*captured) != 0 {
		t.Fatalf("wrong-type event must be ignored, got %d announces", len(*captured))
	}
}

func TestHandleSurpriseFailedIgnoresWrongType(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newZeroFieldTestServer(t, tm)

	characterId := uint32(6006)
	accountId := uint32(9006)

	cleanup := registerRealSession(t, tm, characterId, accountId)
	defer cleanup()

	wp, captured := recordingWriterProducer(t)

	h := handleStatusEventSurpriseFailed(sc, wp)
	h(nullLogger(), ctx, cashshop2.StatusEvent[cashshop2.SurpriseFailedEventBody]{
		CharacterId: characterId,
		Type:        cashshop2.StatusEventTypeSurpriseOpened, // wrong type for this handler
		Body:        cashshop2.SurpriseFailedEventBody{Reason: "INTERNAL"},
	})

	if len(*captured) != 0 {
		t.Fatalf("wrong-type event must be ignored, got %d announces", len(*captured))
	}
}

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
	cashpkt.CashShopOperationNameChangeBuyDone:                float64(70),
	cashpkt.CashShopOperationTransferWorldDone:                float64(71),
	cashpkt.CashShopOperationTransferWorldFailed:              float64(72),
	// BUY_NORMAL_SUCCESS 141 is template_gms_83_1.json's real value.
	cashpkt.CashShopOperationBuyNormalDone: float64(141),
	// POP_UP is the WorldMessageMode key handleStatusEventError's name-change
	// pink-text fallback resolves (socket/writer/world_message.go's
	// getWorldMessageMode), not a CashShopOperation* key.
	"POP_UP": float64(73),
	// The remaining task-24a seam-trace keys are arbitrary but distinct --
	// only used to prove the correct code was selected for each new event's
	// consumer arm, the same convention gachaponOperations above documents.
	cashpkt.CashShopOperationRebateDone:      float64(150),
	cashpkt.CashShopOperationGiftDone:        float64(151),
	cashpkt.CashShopOperationBuyPackageDone:  float64(152),
	cashpkt.CashShopOperationGiftPackageDone: float64(153),
	cashpkt.CashShopOperationCoupleDone:      float64(154),
	cashpkt.CashShopOperationFriendshipDone:  float64(155),
}

var testErrors = map[string]interface{}{
	"COUPON_EXPIRED":      float64(178),
	"INVALID_COUPON_CODE": float64(176),
	// deliberately no UNKNOWN_ERROR key -- see
	// TestCouponFailedUnknownErrorFallsThroughToTheDefaultNotice.
	"WORLD_TRANSFER_UNAVAILABLE": float64(181),
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
	logHook     *testlog.Hook
	ctx         context.Context
	tenant      tenant.Model
	sc          server.Model
	wp          writer.Producer
	announced   []announcement
	assets      map[string]string
	rings       map[uint32]string
	ringFetches map[uint32]int
	characters  map[string]uint32
	walletDoc   string
	compartment uuid.UUID
}

func newConsumerEnv(t *testing.T) *consumerEnv {
	t.Helper()

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l, hook := testlog.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	env := &consumerEnv{
		t:           t,
		logger:      l,
		logHook:     hook,
		ctx:         tenant.WithContext(context.Background(), tm),
		tenant:      tm,
		assets:      make(map[string]string),
		rings:       make(map[uint32]string),
		ringFetches: make(map[uint32]int),
		characters:  make(map[string]uint32),
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
		if strings.HasSuffix(r.URL.Path, "/rings") {
			cid, _ := strconv.Atoi(r.URL.Query().Get("filter[characterId]"))
			env.ringFetches[uint32(cid)]++
			if doc, ok := env.rings[uint32(cid)]; ok {
				_, _ = w.Write([]byte(doc))
			} else {
				_, _ = w.Write([]byte(emptyRingsListDoc()))
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/characters") {
			name := r.URL.Query().Get("name")
			if id, ok := env.characters[name]; ok {
				_, _ = w.Write([]byte(characterByNameDoc(id, name)))
			} else {
				_, _ = w.Write([]byte(`{"data":[]}`))
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"status":"404","title":"Not Found"}]}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/")
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

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

// addSession registers an additional live session for characterId on this
// env's tenant/channel -- used to give a ring-purchase partner "a live
// session on this channel" the way handleStatusEventRingPurchased's own
// session correlation would see them.
func (e *consumerEnv) addSession(characterId uint32, accountId uint32) {
	e.t.Helper()
	sessionId := uuid.New()
	s := session.NewSession(sessionId, e.tenant, 0, discardConn{})
	session.AddSessionToRegistry(e.tenant.Id(), s)
	sp := session.NewProcessor(e.logger, e.ctx)
	_ = sp.SetAccountId(sessionId, accountId)
	_ = sp.SetCharacterId(sessionId, characterId)
}

func (e *consumerEnv) seedAsset(compartmentId uuid.UUID, assetId uint32) {
	e.t.Helper()
	path := fmt.Sprintf("/accounts/%d/cash-shop/inventory/compartments/%s/assets/%d", testAccountId, compartmentId.String(), assetId)
	e.assets[path] = assetDoc(assetId, compartmentId)
}

// emptyRingsListDoc is the atlas-cashshop GET /rings response for a
// character with no ring halves (ring/rest.go's RestModel/Extract shape,
// mirrored from ring/processor_test.go's ringsDoc).
func emptyRingsListDoc() string {
	return `{"data":[],"meta":{"total":0,"page":{"number":1,"size":250,"last":1}}}`
}

// ringsListDoc is a one-half GET /rings response for characterId, mirroring
// ring/processor_test.go's ringsDoc.
func ringsListDoc(characterId uint32, cashId int64) string {
	return fmt.Sprintf(
		`{"data":[{"id":"%s","type":"rings","attributes":{"pairId":"%s","characterId":%d,"partnerCharacterId":0,"assetId":1,"itemTemplateId":1112001,"ringType":"COUPLE","state":"ACTIVE","cashId":%d,"partnerCashId":0,"partnerName":"Partner"}}],"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`,
		uuid.New(), uuid.New(), characterId, cashId,
	)
}

// characterByNameDoc is the atlas-character GET /characters?name= list
// response resolving name to id (character/rest.go's RestModel shape).
func characterByNameDoc(id uint32, name string) string {
	return fmt.Sprintf(`{"data":[{"type":"characters","id":"%d","attributes":{"name":%q,"accountId":1,"worldId":0}}]}`, id, name)
}

// seedRingCache populates characterId's ring cache through the real
// Processor.Populate entry point (not a direct cache write) -- the ring
// cache is private to the ring package, so this test drives it exactly the
// way production code would.
func (e *consumerEnv) seedRingCache(characterId uint32, cashId int64) {
	e.t.Helper()
	e.rings[characterId] = ringsListDoc(characterId, cashId)
	if err := ring.NewProcessor(e.logger, e.ctx).Populate(characterId); err != nil {
		e.t.Fatalf("seedRingCache: Populate(%d): %v", characterId, err)
	}
}

// ringCacheEmpty reports whether characterId's cached ring halves are gone --
// observed by re-Populate-ing (idempotent: a no-op if still cached, per
// ring.Processor.Populate's doc comment) and checking whether the upstream
// GET /rings fixture was hit again, tracked via e.ringFetches. Avoids
// reaching into the ring package's private cache map.
func (e *consumerEnv) ringCacheEmpty(characterId uint32) bool {
	e.t.Helper()
	before := e.ringFetches[characterId]
	if err := ring.NewProcessor(e.logger, e.ctx).Populate(characterId); err != nil {
		e.t.Fatalf("ringCacheEmpty: Populate(%d): %v", characterId, err)
	}
	return e.ringFetches[characterId] > before
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

// decodeNameChangeBuyDone decodes the last announced CashShopOperation body
// as a NAME_CHANGE_BUY_DONE arm.
func (e *consumerEnv) decodeNameChangeBuyDone(t *testing.T) cashpkt.NameChangeBuyDone {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.NameChangeBuyDone{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// decodeTransferWorldDone decodes the last announced CashShopOperation body
// as a TRANSFER_WORLD_SUCCESS arm.
func (e *consumerEnv) decodeTransferWorldDone(t *testing.T) cashpkt.TransferWorldDone {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.TransferWorldDone{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// decodeTransferWorldFailed decodes the last announced CashShopOperation body
// as a TRANSFER_WORLD_FAILED arm.
func (e *consumerEnv) decodeTransferWorldFailed(t *testing.T) cashpkt.TransferWorldFailed {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.TransferWorldFailed{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// decodeCashInventoryPurchaseSuccess decodes the last announced
// CashShopOperation body as the generic PURCHASE_SUCCESS arm.
func (e *consumerEnv) decodeCashInventoryPurchaseSuccess(t *testing.T) cashpkt.CashShopPurchaseSuccess {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.CashShopPurchaseSuccess{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// decodeBuyNormalDone decodes the last announced CashShopOperation body as
// the BUY_NORMAL_SUCCESS arm.
func (e *consumerEnv) decodeBuyNormalDone(t *testing.T) cashpkt.BuyNormalDone {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.BuyNormalDone{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// decodeEnableEquipSlotExtSuccess decodes the last announced
// CashShopOperation body as the ENABLE_EQUIP_SLOT_EXT_SUCCESS arm.
func (e *consumerEnv) decodeEnableEquipSlotExtSuccess(t *testing.T) cashpkt.EnableEquipSlotExtSuccess {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.EnableEquipSlotExtSuccess{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// decodeRebateDone decodes the last announced CashShopOperation body as the
// REBATE_SUCCESS arm.
func (e *consumerEnv) decodeRebateDone(t *testing.T) cashpkt.RebateDone {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.RebateDone{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// decodeGiftDone decodes the announced CashShopOperation body as the
// GIFT_SUCCESS arm. Searches by writer name rather than taking the last
// announcement: GIFT_SUCCESS is followed by a CashQueryResult announce, so
// it is not always the last packet on the wire.
func (e *consumerEnv) decodeGiftDone(t *testing.T) cashpkt.GiftDone {
	t.Helper()
	for _, a := range e.announced {
		if a.writerName != cashpkt.CashShopOperationWriter {
			continue
		}
		m := cashpkt.GiftDone{}
		req := request.Request(a.body)
		r := request.NewRequestReader(&req, 0)
		m.Decode(e.logger, e.ctx)(&r, nil)
		return m
	}
	t.Fatal("no CashShopOperation packet was announced")
	return cashpkt.GiftDone{}
}

// decodeBuyPackageDone decodes the last announced CashShopOperation body as
// the BUY_PACKAGE_SUCCESS arm.
func (e *consumerEnv) decodeBuyPackageDone(t *testing.T) cashpkt.BuyPackageDone {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.BuyPackageDone{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// decodeGiftPackageDone decodes the last announced CashShopOperation body as
// the GIFT_PACKAGE_SUCCESS arm.
func (e *consumerEnv) decodeGiftPackageDone(t *testing.T) cashpkt.GiftPackageDone {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.GiftPackageDone{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// decodeCoupleDone decodes the last announced CashShopOperation body as the
// COUPLE_SUCCESS arm.
func (e *consumerEnv) decodeCoupleDone(t *testing.T) cashpkt.CoupleDone {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.CoupleDone{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// decodeFriendshipDone decodes the last announced CashShopOperation body as
// the FRIENDSHIP_SUCCESS arm.
func (e *consumerEnv) decodeFriendshipDone(t *testing.T) cashpkt.FriendshipDone {
	t.Helper()
	a := e.lastAnnounced()
	if a.writerName != cashpkt.CashShopOperationWriter {
		t.Fatalf("last announced writer = %s, want %s", a.writerName, cashpkt.CashShopOperationWriter)
	}
	m := cashpkt.FriendshipDone{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(e.logger, e.ctx)(&r, nil)
	return m
}

// pendingChangeRecordFixture is one row a pendingChangeFixtureServer's GET
// .../pending-changes response reports.
type pendingChangeRecordFixture struct {
	id     string
	typ    string
	status string
}

// pendingChangeFixtureServer stands in for atlas-character's pending-changes
// resource: GET .../pending-changes lists records (how the purchase-outcome
// consumer resolves a TransactionId to a PENDING record, task-227 task 39),
// and POST .../pending-changes/cancel resolves the self-scoped release. It
// records every request path -- for the error-arm tests, the whole of what
// the CHANNEL side can prove about "no refund minted" is that its own call
// graph is exactly one cancel POST and nothing else; refund-minting itself is
// atlas-character's Resolve() and is out of this task's file inventory.
func pendingChangeFixtureServer(t *testing.T, characterId uint32, records []pendingChangeRecordFixture) *[]string {
	t.Helper()
	var calls []string
	listPath := fmt.Sprintf("/characters/%d/pending-changes", characterId)
	cancelPath := listPath + "/cancel"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == listPath:
			items := make([]map[string]any, 0, len(records))
			for _, rec := range records {
				items = append(items, map[string]any{
					"type": "pending-changes",
					"id":   rec.id,
					"attributes": map[string]any{
						"characterId": characterId,
						"type":        rec.typ,
						"status":      rec.status,
					},
				})
			}
			b, _ := json.Marshal(map[string]any{"data": items})
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_, _ = w.Write(b)
		case r.Method == http.MethodPost && r.URL.Path == cancelPath:
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"status":"404","title":"Not Found"}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")
	return &calls
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

// TestPurchaseSuccessWithPendingNameChangeCashIdIsNonZero is the regression
// gate for the whole defect task-227 exists to fix: a name-change purchase's
// success announcement must carry the resolved asset's REAL cash serial, not
// the CashId=0 the pre-task-38 handler fabricated.
func TestPurchaseSuccessWithPendingNameChangeCashIdIsNonZero(t *testing.T) {
	env := newConsumerEnv(t)
	assetId := uint32(601)
	env.seedAsset(env.compartment, assetId)
	txId := uuid.New()
	pendingChangeFixtureServer(t, testCharacterId, []pendingChangeRecordFixture{
		{id: txId.String(), typ: pendingchange.TypeNameChange, status: pendingchange.StatusPending},
	})

	handleStatusEventPurchase(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.PurchaseEventBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypePurchase,
		Body: cashshop2.PurchaseEventBody{
			CompartmentId: env.compartment,
			AssetId:       assetId,
			TransactionId: txId,
		},
	})

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	body := env.decodeNameChangeBuyDone(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationNameChangeBuyDone) {
		t.Errorf("mode = %d, want the NAME_CHANGE_BUY_DONE mode %d", body.Mode(), env.modeFor(cashpkt.CashShopOperationNameChangeBuyDone))
	}
	if body.Item().CashId == 0 {
		t.Fatal("CashId = 0 -- this is the exact regression this phase exists to fix: the client cannot bind the purchased item without the asset's real cash serial")
	}
	if body.Item().CashId != 9001 {
		t.Errorf("CashId = %d, want the seeded asset's real cash id 9001", body.Item().CashId)
	}
}

// TestPurchaseSuccessWithPendingWorldTransferCashIdIsNonZero mirrors the
// name-change regression gate for the world-transfer arm.
func TestPurchaseSuccessWithPendingWorldTransferCashIdIsNonZero(t *testing.T) {
	env := newConsumerEnv(t)
	assetId := uint32(602)
	env.seedAsset(env.compartment, assetId)
	txId := uuid.New()
	pendingChangeFixtureServer(t, testCharacterId, []pendingChangeRecordFixture{
		{id: txId.String(), typ: pendingchange.TypeWorldTransfer, status: pendingchange.StatusPending},
	})

	handleStatusEventPurchase(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.PurchaseEventBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypePurchase,
		Body: cashshop2.PurchaseEventBody{
			CompartmentId: env.compartment,
			AssetId:       assetId,
			TransactionId: txId,
		},
	})

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	body := env.decodeTransferWorldDone(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationTransferWorldDone) {
		t.Errorf("mode = %d, want the TRANSFER_WORLD_SUCCESS mode %d", body.Mode(), env.modeFor(cashpkt.CashShopOperationTransferWorldDone))
	}
	if body.Item().CashId == 0 {
		t.Fatal("CashId = 0 -- this is the exact regression this phase exists to fix")
	}
}

// TestPurchaseSuccessUnrelatedBuyTakesPreExistingPath pins the concurrency
// case the old code (keyed only on CharacterId) could not tell apart: a
// purchase whose TransactionId is the zero UUID (every ordinary buy) must
// take the generic PURCHASE_SUCCESS path, never a name-change/world-transfer
// DONE arm. A pendingChangeFixtureServer IS stood up here -- with no records,
// so any call it receives is itself the finding -- to pin the actual
// mechanism: resolvePendingChange must short-circuit on the zero id and never
// call atlas-character at all. (requests.RootUrl falls back to an empty
// BASE_SERVICE_URL rather than panicking when no fixture is stood up, so an
// unreachable-URL assertion alone would not catch a deleted guard -- see
// consumer.go:38.)
func TestPurchaseSuccessUnrelatedBuyTakesPreExistingPath(t *testing.T) {
	env := newConsumerEnv(t)
	assetId := uint32(603)
	env.seedAsset(env.compartment, assetId)
	calls := pendingChangeFixtureServer(t, testCharacterId, nil)

	handleStatusEventPurchase(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.PurchaseEventBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypePurchase,
		Body: cashshop2.PurchaseEventBody{
			CompartmentId: env.compartment,
			AssetId:       assetId,
			TransactionId: uuid.Nil,
		},
	})

	if len(*calls) != 0 {
		t.Fatalf("calls = %v, want none -- resolvePendingChange must short-circuit on the zero TransactionId without calling atlas-character", *calls)
	}
	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	body := env.decodeCashInventoryPurchaseSuccess(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationPurchaseSuccess) {
		t.Errorf("mode = %d, want the generic PURCHASE_SUCCESS mode %d -- an unrelated buy must not take a name-change/world-transfer DONE arm", body.Mode(), env.modeFor(cashpkt.CashShopOperationPurchaseSuccess))
	}
}

// TestPurchaseSuccessBuyNormalAnnouncesBuyNormalDone pins the BUY_NORMAL
// success-routing branch (consumer.go's handleStatusEventPurchase): a
// purchase event discriminated as BUY_NORMAL must be answered with the
// dedicated BUY_NORMAL_SUCCESS body (mode 141 for this test tenant's GMS
// 83.1 template), not the generic PURCHASE_SUCCESS body an undiscriminated
// event still receives.
func TestPurchaseSuccessBuyNormalAnnouncesBuyNormalDone(t *testing.T) {
	env := newConsumerEnv(t)
	assetId := uint32(604)
	env.seedAsset(env.compartment, assetId)

	handleStatusEventPurchase(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.PurchaseEventBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypePurchase,
		Body: cashshop2.PurchaseEventBody{
			CompartmentId: env.compartment,
			AssetId:       assetId,
			TransactionId: uuid.Nil,
			Operation:     cashshop2.ErrorOperationBuyNormal,
		},
	})

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	body := env.decodeBuyNormalDone(t)
	if got, want := body.Mode(), env.modeFor(cashpkt.CashShopOperationBuyNormalDone); got != want {
		t.Errorf("mode = %d, want the BUY_NORMAL_SUCCESS mode %d", got, want)
	}
	if unwanted := env.modeFor(cashpkt.CashShopOperationPurchaseSuccess); body.Mode() == unwanted {
		t.Errorf("mode = %d, must not be the generic PURCHASE_SUCCESS mode %d -- a BUY_NORMAL purchase must not take the generic fallback path", body.Mode(), unwanted)
	}
	refs := body.Refs()
	if len(refs) != 1 {
		t.Fatalf("refs = %v, want exactly one entry", refs)
	}
	if refs[0].ItemId != 5000000 {
		t.Errorf("ItemId = %d, want 5000000 (assetDoc's fixture templateId)", refs[0].ItemId)
	}
	if refs[0].Quantity != 1 {
		t.Errorf("Quantity = %d, want 1 (assetDoc's fixture quantity)", refs[0].Quantity)
	}
}

// TestErrorWithPendingNameChangeCancelsRecordAndAnswersPinkText pins Step
// 1(c) for the name-change arm: an error event whose TransactionId resolves
// to a PENDING record releases it via the existing self-scoped cancel path
// and answers the client -- name-change has no dedicated failure arm, so it
// falls back to pink text, mirroring the synchronous rejection route
// (socket/handler/cash_shop_operation.go's announceCashShopRejection).
//
// The call list asserts the "no refund minted" absence check as far as the
// CHANNEL side can prove it: exactly one GET (resolve) and one POST
// (cancel), nothing else -- the channel never mints anything itself, so a
// call graph bigger than that would be the first sign of a spurious grant.
// Refund-minting itself is atlas-character's Resolve() (HasAsset()-gated)
// and is outside this task's file inventory.
func TestErrorWithPendingNameChangeCancelsRecordAndAnswersPinkText(t *testing.T) {
	env := newConsumerEnv(t)
	txId := uuid.New()
	calls := pendingChangeFixtureServer(t, testCharacterId, []pendingChangeRecordFixture{
		{id: txId.String(), typ: pendingchange.TypeNameChange, status: pendingchange.StatusPending},
	})

	handleStatusEventError(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.ErrorEventBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeError,
		Body: cashshop2.ErrorEventBody{
			Error:         "NOT_ENOUGH_CASH",
			TransactionId: txId,
		},
	})

	wantCalls := []string{
		fmt.Sprintf("GET /characters/%d/pending-changes", testCharacterId),
		fmt.Sprintf("POST /characters/%d/pending-changes/cancel", testCharacterId),
	}
	if !reflect.DeepEqual(*calls, wantCalls) {
		t.Fatalf("calls = %v, want exactly %v -- the channel side must release the record and mint nothing else", *calls, wantCalls)
	}

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{chatpkt.WorldMessageWriter}) {
		t.Fatalf("announced %v, want exactly [%s] -- name-change has no dedicated failure arm", got, chatpkt.WorldMessageWriter)
	}
	a := env.lastAnnounced()
	m := chatpkt.WorldMessageSimple{}
	req := request.Request(a.body)
	r := request.NewRequestReader(&req, 0)
	m.Decode(env.logger, env.ctx)(&r, nil)
	if m.Mode() != env.modeFor("POP_UP") {
		t.Errorf("mode = %d, want resolved POP_UP mode %d", m.Mode(), env.modeFor("POP_UP"))
	}
	if !strings.Contains(m.Message(), "name change") {
		t.Errorf("message = %q, want it to mention the name change request", m.Message())
	}
}

// TestErrorWithPendingWorldTransferCancelsRecordAndAnswersFailedArm mirrors
// the name-change case for world-transfer, which DOES have a dedicated
// TRANSFER_WORLD_FAILED arm and must use it (not the generic
// INVENTORY_CAPACITY_INCREASE_FAILED fallback).
func TestErrorWithPendingWorldTransferCancelsRecordAndAnswersFailedArm(t *testing.T) {
	env := newConsumerEnv(t)
	txId := uuid.New()
	calls := pendingChangeFixtureServer(t, testCharacterId, []pendingChangeRecordFixture{
		{id: txId.String(), typ: pendingchange.TypeWorldTransfer, status: pendingchange.StatusPending},
	})

	handleStatusEventError(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.ErrorEventBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeError,
		Body: cashshop2.ErrorEventBody{
			Error:         "WORLD_TRANSFER_UNAVAILABLE",
			TransactionId: txId,
		},
	})

	wantCalls := []string{
		fmt.Sprintf("GET /characters/%d/pending-changes", testCharacterId),
		fmt.Sprintf("POST /characters/%d/pending-changes/cancel", testCharacterId),
	}
	if !reflect.DeepEqual(*calls, wantCalls) {
		t.Fatalf("calls = %v, want exactly %v -- the channel side must release the record and mint nothing else", *calls, wantCalls)
	}

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	body := env.decodeTransferWorldFailed(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationTransferWorldFailed) {
		t.Errorf("mode = %d, want the TRANSFER_WORLD_FAILED mode %d, not the generic capacity-increase-failed fallback", body.Mode(), env.modeFor(cashpkt.CashShopOperationTransferWorldFailed))
	}
	if body.Mode() == env.modeFor(cashpkt.CashShopOperationInventoryCapacityIncreaseFailed) {
		t.Errorf("mode = %d collides with the generic capacity-increase-failed fallback", body.Mode())
	}
	if body.ErrorCode() != env.errorByteFor("WORLD_TRANSFER_UNAVAILABLE") {
		t.Errorf("errorCode = %d, want %d", body.ErrorCode(), env.errorByteFor("WORLD_TRANSFER_UNAVAILABLE"))
	}
}

// TestErrorUnrelatedFailureTakesPreExistingPath pins that an ordinary
// purchase failure (zero TransactionId) still takes the generic
// INVENTORY_CAPACITY_INCREASE_FAILED fallback unchanged, and never touches
// atlas-character at all. A pendingChangeFixtureServer IS stood up here --
// with no records, so any call it receives is itself the finding -- to pin
// the actual mechanism: resolvePendingChange must short-circuit on the zero
// id and never call atlas-character. (requests.RootUrl falls back to an
// empty BASE_SERVICE_URL rather than panicking when no fixture is stood up,
// so an unreachable-URL assertion alone would not catch a deleted guard --
// see consumer.go:38.)
func TestErrorUnrelatedFailureTakesPreExistingPath(t *testing.T) {
	env := newConsumerEnv(t)
	calls := pendingChangeFixtureServer(t, testCharacterId, nil)

	handleStatusEventError(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.ErrorEventBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeError,
		Body: cashshop2.ErrorEventBody{
			Error:         "UNKNOWN_ERROR",
			TransactionId: uuid.Nil,
		},
	})

	if len(*calls) != 0 {
		t.Fatalf("calls = %v, want none -- resolvePendingChange must short-circuit on the zero TransactionId without calling atlas-character", *calls)
	}
	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	if got := env.lastAnnouncedMode(); got != env.modeFor(cashpkt.CashShopOperationInventoryCapacityIncreaseFailed) {
		t.Errorf("mode = %d, want the generic capacity-increase-failed mode %d", got, env.modeFor(cashpkt.CashShopOperationInventoryCapacityIncreaseFailed))
	}
}

// TestEquipSlotIncreasedAnnouncesWireSlotIndexZeroNotTheCanonicalPosition
// pins the hazard consumer.go:544's doc comment describes: the event's
// SlotIndex (-59, the Atlas CANONICAL equipped-inventory pendant2 position)
// must never reach the wire directly. handleStatusEventEquipSlotIncreased
// must announce the ENABLE_EQUIP_SLOT_EXT_SUCCESS body with wire slotIndex
// 0 -- a regression that forwards e.Body.SlotIndex straight through would
// instead encode 65477 (the unsigned view of -59) and this assertion would
// fail loudly. Days is asserted separately, with a distinct value (30), so
// the two fields cannot be silently swapped or confused.
func TestEquipSlotIncreasedAnnouncesWireSlotIndexZeroNotTheCanonicalPosition(t *testing.T) {
	env := newConsumerEnv(t)

	handleStatusEventEquipSlotIncreased(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.EquipSlotIncreasedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeEquipSlotIncreased,
		Body: cashshop2.EquipSlotIncreasedBody{
			TransactionId: uuid.Nil,
			SlotIndex:     -59,
			Days:          30,
		},
	})

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	body := env.decodeEnableEquipSlotExtSuccess(t)
	if body.SlotIndex() != 0 {
		t.Errorf("slotIndex = %d, want the WIRE value 0 -- the canonical -59 (65477 unsigned) must never reach the packet", body.SlotIndex())
	}
	if body.Days() != 30 {
		t.Errorf("days = %d, want 30 unchanged", body.Days())
	}
}

// TestLockerRebatedAnnouncesDoneWithCashIdAndAmount pins the LOCKER_REBATED
// producer/consumer seam: LockerRebatedBody.CashId/Amount must land on
// RebateDone's sn/amount fields unchanged (Currency is mirrored for wire
// compatibility only -- CashShopRebateDoneBody never reads it, see
// LockerRebatedBody's doc comment).
func TestLockerRebatedAnnouncesDoneWithCashIdAndAmount(t *testing.T) {
	env := newConsumerEnv(t)

	handleStatusEventLockerRebated(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.LockerRebatedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeLockerRebated,
		Body: cashshop2.LockerRebatedBody{
			TransactionId: uuid.New(),
			CashId:        998877,
			Amount:        4500,
			Currency:      1,
		},
	})

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	body := env.decodeRebateDone(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationRebateDone) {
		t.Errorf("mode = %d, want the REBATE_SUCCESS mode %d", body.Mode(), env.modeFor(cashpkt.CashShopOperationRebateDone))
	}
	if body.SN() != 998877 {
		t.Errorf("sn = %d, want the event's CashId 998877", body.SN())
	}
	if body.Amount() != 4500 {
		t.Errorf("amount = %d, want the event's Amount 4500", body.Amount())
	}
}

// TestGiftPurchasedAnnouncesGiftDoneWithRecipientNameAndItem pins the
// GIFT_PURCHASED producer/consumer seam: GiftPurchasedBody's
// RecipientName/TemplateId/Quantity must land on GiftDone's
// recipientName/itemId/quantity unchanged.
// TestGiftPurchasedAnnouncesGiftDoneWithRecipientNameAndItem pins the client's
// gift batch state machine ordering (CCashShop::SendGiftsPacket, v83 IDB
// 0x46f940): GIFT_SUCCESS records the batch confirmation and must land
// before CASH_QUERY_RESULT drives the batch to its final notice, or the
// sender sees "The gifts could not be sent." (SP_562) instead of "All the
// gifts have been sent..." (SP_561).
func TestGiftPurchasedAnnouncesGiftDoneWithRecipientNameAndItem(t *testing.T) {
	env := newConsumerEnv(t)

	handleStatusEventGiftPurchased(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.GiftPurchasedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeGiftPurchased,
		Body: cashshop2.GiftPurchasedBody{
			TransactionId:        uuid.New(),
			RecipientName:        "Recipient",
			TemplateId:           5010000,
			Quantity:             1,
			Price:                900,
			RecipientCharacterId: 99,
		},
	})

	// GIFT_SUCCESS first, then CASH_QUERY_RESULT: only this order resolves
	// the client's gift batch to SP_561 rather than SP_562.
	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter, cashpkt.CashQueryResultWriter}) {
		t.Fatalf("announced %v, want exactly [%s, %s]", got, cashpkt.CashShopOperationWriter, cashpkt.CashQueryResultWriter)
	}
	body := env.decodeGiftDone(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationGiftDone) {
		t.Errorf("mode = %d, want the GIFT_SUCCESS mode %d", body.Mode(), env.modeFor(cashpkt.CashShopOperationGiftDone))
	}
	if body.RecipientName() != "Recipient" {
		t.Errorf("recipientName = %q, want %q", body.RecipientName(), "Recipient")
	}
	if body.ItemId() != 5010000 {
		t.Errorf("itemId = %d, want the event's TemplateId 5010000", body.ItemId())
	}
	if body.Quantity() != 1 {
		t.Errorf("quantity = %d, want 1", body.Quantity())
	}
}

// TestPackagePurchasedBuyForSelfProjectsAssetsIntoBuyPackageDone pins the
// PACKAGE_PURCHASED producer/consumer seam for the buy-for-self path. Unlike
// the COMMAND body, the STATUS body's RecipientCharacterId echoes the
// buyer's own identity on a buy-for-self purchase and is never zero
// (kafka/message/cashshop/kafka.go:372-375), so the fixture here sets it
// equal to CharacterId -- exactly what atlas-cashshop actually emits (Defect
// E, bug-cash-shop-live-testing-round-2.md). Each of Body.AssetIds must be
// resolved and projected into BuyPackageDone's item list.
func TestPackagePurchasedBuyForSelfProjectsAssetsIntoBuyPackageDone(t *testing.T) {
	env := newConsumerEnv(t)
	assetId := uint32(4242)
	env.seedAsset(env.compartment, assetId)

	handleStatusEventPackagePurchased(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.PackagePurchasedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypePackagePurchased,
		Body: cashshop2.PackagePurchasedBody{
			TransactionId:        uuid.New(),
			CompartmentId:        env.compartment,
			AssetIds:             []uint32{assetId},
			PackageTemplateId:    9000000,
			Price:                5000,
			RecipientCharacterId: testCharacterId,
			RecipientName:        "Buyer",
		},
	})

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	body := env.decodeBuyPackageDone(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationBuyPackageDone) {
		t.Errorf("mode = %d, want the BUY_PACKAGE_SUCCESS mode %d", body.Mode(), env.modeFor(cashpkt.CashShopOperationBuyPackageDone))
	}
	if len(body.Items()) != 1 {
		t.Fatalf("items = %d, want 1 — one entry per Body.AssetIds", len(body.Items()))
	}
}

// TestPackagePurchasedGiftAnnouncesGiftPackageDone pins the PACKAGE_PURCHASED
// gift path (RecipientCharacterId != CharacterId): Body.RecipientName/PackageTemplateId
// must land on GiftPackageDone's recipientName/packageId unchanged, and the
// asset lookup used by the self path must not run.
func TestPackagePurchasedGiftAnnouncesGiftPackageDone(t *testing.T) {
	env := newConsumerEnv(t)

	handleStatusEventPackagePurchased(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.PackagePurchasedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypePackagePurchased,
		Body: cashshop2.PackagePurchasedBody{
			TransactionId:        uuid.New(),
			CompartmentId:        env.compartment,
			AssetIds:             nil,
			PackageTemplateId:    9000001,
			Price:                5000,
			RecipientCharacterId: 99,
			RecipientName:        "GiftRecipient",
		},
	})

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	body := env.decodeGiftPackageDone(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationGiftPackageDone) {
		t.Errorf("mode = %d, want the GIFT_PACKAGE_SUCCESS mode %d", body.Mode(), env.modeFor(cashpkt.CashShopOperationGiftPackageDone))
	}
	if body.RecipientName() != "GiftRecipient" {
		t.Errorf("recipientName = %q, want %q", body.RecipientName(), "GiftRecipient")
	}
	if body.PackageId() != 9000001 {
		t.Errorf("packageId = %d, want the event's PackageTemplateId 9000001", body.PackageId())
	}
}

// TestRingPurchasedCoupleAnnouncesCoupleDoneWithPartnerName pins the
// RING_PURCHASED producer/consumer seam for RingType COUPLE:
// PartnerName/TemplateId/Quantity must land on CoupleDone's
// recipientName/itemId/quantity, and the buyer's own AssetId is projected
// into the item blob.
func TestRingPurchasedCoupleAnnouncesCoupleDoneWithPartnerName(t *testing.T) {
	env := newConsumerEnv(t)
	assetId := uint32(6060)
	env.seedAsset(env.compartment, assetId)

	handleStatusEventRingPurchased(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.RingPurchasedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeRingPurchased,
		Body: cashshop2.RingPurchasedBody{
			TransactionId: uuid.New(),
			CompartmentId: env.compartment,
			AssetId:       assetId,
			PartnerName:   "Partner",
			TemplateId:    1112000,
			Quantity:      1,
			RingType:      cashshop2.RingTypeCouple,
			PairId:        uuid.New(),
		},
	})

	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter}) {
		t.Fatalf("announced %v, want exactly [%s]", got, cashpkt.CashShopOperationWriter)
	}
	body := env.decodeCoupleDone(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationCoupleDone) {
		t.Errorf("mode = %d, want the COUPLE_SUCCESS mode %d", body.Mode(), env.modeFor(cashpkt.CashShopOperationCoupleDone))
	}
	if body.RecipientName() != "Partner" {
		t.Errorf("recipientName = %q, want the event's PartnerName %q", body.RecipientName(), "Partner")
	}
	if body.ItemId() != 1112000 {
		t.Errorf("itemId = %d, want the event's TemplateId 1112000", body.ItemId())
	}
	if body.Quantity() != 1 {
		t.Errorf("quantity = %d, want 1", body.Quantity())
	}
}

// TestRingPurchasedFriendshipAnnouncesFriendshipDone pins the same seam for
// RingType FRIENDSHIP, the switch's other live arm.
func TestRingPurchasedFriendshipAnnouncesFriendshipDone(t *testing.T) {
	env := newConsumerEnv(t)
	assetId := uint32(6061)
	env.seedAsset(env.compartment, assetId)

	handleStatusEventRingPurchased(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.RingPurchasedBody]{
		CharacterId: testCharacterId,
		Type:        cashshop2.StatusEventTypeRingPurchased,
		Body: cashshop2.RingPurchasedBody{
			TransactionId: uuid.New(),
			CompartmentId: env.compartment,
			AssetId:       assetId,
			PartnerName:   "Buddy",
			TemplateId:    1122000,
			Quantity:      1,
			RingType:      cashshop2.RingTypeFriendship,
			PairId:        uuid.New(),
		},
	})

	body := env.decodeFriendshipDone(t)
	if body.Mode() != env.modeFor(cashpkt.CashShopOperationFriendshipDone) {
		t.Errorf("mode = %d, want the FRIENDSHIP_SUCCESS mode %d", body.Mode(), env.modeFor(cashpkt.CashShopOperationFriendshipDone))
	}
	if body.RecipientName() != "Buddy" {
		t.Errorf("recipientName = %q, want the event's PartnerName %q", body.RecipientName(), "Buddy")
	}
	if body.ItemId() != 1122000 {
		t.Errorf("itemId = %d, want the event's TemplateId 1122000", body.ItemId())
	}
}

// TestRingPurchasedInvalidatesCache pins task-269 task 12's invalidation
// path. RingPurchasedBody carries no cashId for either half and no partner
// character id, so the cache is dropped rather than patched -- the buyer's
// entry always, and the partner's entry when Body.PartnerName resolves to a
// known character.
func TestRingPurchasedInvalidatesCache(t *testing.T) {
	const buyerId = uint32(100)
	const partnerId = uint32(200)
	const buyerCashId = int64(1111)
	const partnerCashId = int64(2222)

	newEvent := func() cashshop2.StatusEvent[cashshop2.RingPurchasedBody] {
		return cashshop2.StatusEvent[cashshop2.RingPurchasedBody]{
			CharacterId: buyerId,
			Type:        cashshop2.StatusEventTypeRingPurchased,
			Body: cashshop2.RingPurchasedBody{
				TransactionId: uuid.New(),
				CompartmentId: uuid.New(),
				AssetId:       9999,
				PartnerName:   "Partner",
				TemplateId:    1112000,
				Quantity:      1,
				RingType:      cashshop2.RingTypeCouple,
				PairId:        uuid.New(),
			},
		}
	}

	t.Run("buyer invalidated", func(t *testing.T) {
		env := newConsumerEnv(t)
		env.seedRingCache(buyerId, buyerCashId)

		handleStatusEventRingPurchased(env.sc, env.wp)(env.logger, env.ctx, newEvent())

		if !env.ringCacheEmpty(buyerId) {
			t.Fatal("buyer's ring cache entry survived RING_PURCHASED, want it gone")
		}
	})

	t.Run("partner invalidated when present", func(t *testing.T) {
		env := newConsumerEnv(t)
		env.seedRingCache(buyerId, buyerCashId)
		env.seedRingCache(partnerId, partnerCashId)
		env.characters["Partner"] = partnerId
		env.addSession(partnerId, 4343)

		handleStatusEventRingPurchased(env.sc, env.wp)(env.logger, env.ctx, newEvent())

		if !env.ringCacheEmpty(buyerId) {
			t.Fatal("buyer's ring cache entry survived RING_PURCHASED, want it gone")
		}
		if !env.ringCacheEmpty(partnerId) {
			t.Fatal("partner's ring cache entry survived RING_PURCHASED, want it gone")
		}
	})

	t.Run("partner absent is not an error", func(t *testing.T) {
		env := newConsumerEnv(t)
		env.seedRingCache(buyerId, buyerCashId)
		// No env.characters["Partner"] entry -- GetByName resolves to
		// atlasmodel.ErrEmptySlice, the "unknown partner" case.

		handleStatusEventRingPurchased(env.sc, env.wp)(env.logger, env.ctx, newEvent())

		if !env.ringCacheEmpty(buyerId) {
			t.Fatal("buyer's ring cache entry survived RING_PURCHASED, want it gone")
		}
	})

	t.Run("wrong tenant untouched", func(t *testing.T) {
		env := newConsumerEnv(t)
		env.seedRingCache(buyerId, buyerCashId)

		otherTenant, err := tenant.Create(uuid.New(), "GMS", 83, 1)
		if err != nil {
			t.Fatalf("tenant: %v", err)
		}
		otherCtx := tenant.WithContext(context.Background(), otherTenant)

		handleStatusEventRingPurchased(env.sc, env.wp)(env.logger, otherCtx, newEvent())

		if env.ringCacheEmpty(buyerId) {
			t.Fatal("tenant A's ring cache entry was dropped by a tenant B event, want it untouched")
		}
	})
}
