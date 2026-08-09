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
	"testing"
	"time"

	cashshop2 "atlas-channel/kafka/message/cashshop"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
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

	cleanup := registerRealSession(t, tm, characterId, accountId)
	defer cleanup()

	wp, captured := recordingWriterProducer(t)

	h := handleStatusEventSurpriseOpened(sc, wp)
	h(nullLogger(), ctx, cashshop2.StatusEvent[cashshop2.SurpriseOpenedEventBody]{
		CharacterId: characterId,
		Type:        cashshop2.StatusEventTypeSurpriseFailed, // wrong type for this handler
		Body: cashshop2.SurpriseOpenedEventBody{
			BoxCashId:    423456789,
			BoxRemaining: 1,
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
