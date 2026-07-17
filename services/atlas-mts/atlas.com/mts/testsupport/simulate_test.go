package testsupport

import (
	"atlas-mts/listing"
	"atlas-mts/test"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	mtsmsg "atlas-mts/kafka/message/mts"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// recordedCommand captures one emitted MTS command for assertions.
type recordedCommand struct {
	topicToken string
	key        []byte
	value      []byte
}

// recordingProducer is a providerFn that captures instead of publishing —
// the same stand-in idiom kafka/consumer/mts/consumer_test.go uses.
type recordingProducer struct {
	mu       sync.Mutex
	commands []recordedCommand
}

func (r *recordingProducer) provider() providerFn {
	return func(ctx context.Context) func(token string) kprod.MessageProducer {
		return func(token string) kprod.MessageProducer {
			return func(p model.Provider[[]kafka.Message]) error {
				ms, err := p()
				if err != nil {
					return err
				}
				r.mu.Lock()
				defer r.mu.Unlock()
				for _, m := range ms {
					r.commands = append(r.commands, recordedCommand{topicToken: token, key: m.Key, value: m.Value})
				}
				return nil
			}
		}
	}
}

// newSimulateServer builds an httptest server with ONLY the purchase/bid
// routes wired to the recording producer.
func newSimulateServer(t *testing.T, db *gorm.DB, rec *recordingProducer) *httptest.Server {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	router := muxRouterWithSimulateRoutes(l, db, rec.provider())
	return httptest.NewServer(router)
}

// seedFixedListing persists an active fixed-sale listing with the given
// listValue, mirroring Task 1's test fixtures.
func seedFixedListing(t *testing.T, db *gorm.DB, listValue uint32) listing.Model {
	t.Helper()
	m, err := listing.NewBuilder(test.TestTenantId, 0, 999000001).
		SetSellerName("TestSeller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(listValue).
		SetCommissionRate(0.10).
		SetCategory("1").
		SetSubCategory("").
		Build()
	if err != nil {
		t.Fatalf("build fixed listing: %v", err)
	}
	stored, err := listing.CreateListing(db, m)
	if err != nil {
		t.Fatalf("seed fixed listing: %v", err)
	}
	return stored
}

// seedAuctionListing persists an active auction listing ending in d,
// mirroring Task 1's test fixtures.
func seedAuctionListing(t *testing.T, db *gorm.DB, listValue uint32, d time.Duration) listing.Model {
	t.Helper()
	end := time.Now().Add(d)
	m, err := listing.NewBuilder(test.TestTenantId, 0, 999000001).
		SetSellerName("TestSeller").
		SetSaleType(listing.SaleTypeAuction).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(listValue).
		SetCommissionRate(0.10).
		SetCategory("3").
		SetSubCategory("").
		SetEndsAt(&end).
		SetMinIncrement(1).
		Build()
	if err != nil {
		t.Fatalf("build auction listing: %v", err)
	}
	stored, err := listing.CreateListing(db, m)
	if err != nil {
		t.Fatalf("seed auction listing: %v", err)
	}
	return stored
}

// jsonApiBody marshals a JSON:API request envelope.
func jsonApiBody(t *testing.T, resourceType string, attributes map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"data": map[string]any{"type": resourceType, "attributes": attributes},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return b
}

// doPost posts body to path on ts with the tenant headers every testsupport
// route requires.
func doPost(t *testing.T, ts *httptest.Server, path string, body []byte) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("TENANT_ID", test.TestTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return res
}

func TestSimulatePurchaseEmitsBuyCommand(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	m := seedFixedListing(t, db, 1000)

	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-purchases", map[string]any{
		"listingId":      m.Id().String(),
		"buyerId":        2001,
		"buyerAccountId": 3001,
		"buyNow":         false,
	})
	res := doPost(t, ts, "/test/purchases", body)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", res.StatusCode)
	}
	if len(rec.commands) != 1 {
		t.Fatalf("expected 1 emitted command, got %d", len(rec.commands))
	}
	if rec.commands[0].topicToken != mtsmsg.EnvCommandTopic {
		t.Fatalf("topic token = %s, want %s", rec.commands[0].topicToken, mtsmsg.EnvCommandTopic)
	}
	var c mtsmsg.Command[mtsmsg.BuyCommandBody]
	if err := json.Unmarshal(rec.commands[0].value, &c); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if c.TransactionId == uuid.Nil {
		t.Fatalf("expected non-nil transaction id")
	}
	if c.Type != mtsmsg.CommandBuy || c.Body.WorldId != 0 || c.Body.Serial != m.Serial() || c.Body.BuyerId != 2001 || c.Body.BuyerAccountId != 3001 || c.Body.BuyNow {
		t.Fatalf("bad command: %+v", c)
	}
	wantKey := kprod.CreateKey(int(uint32(2001)))
	if !bytes.Equal(rec.commands[0].key, wantKey) {
		t.Fatalf("key = %v, want buyer-keyed %v", rec.commands[0].key, wantKey)
	}
}

func TestSimulatePurchaseUnknownListing404s(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)
	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-purchases", map[string]any{
		"listingId":      "00000000-0000-0000-0000-000000000001",
		"buyerId":        2001,
		"buyerAccountId": 3001,
	})
	res := doPost(t, ts, "/test/purchases", body)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
	if len(rec.commands) != 0 {
		t.Fatalf("expected no emission on 404, got %d", len(rec.commands))
	}
}

func TestSimulateBidEmitsPlaceBidCommand(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	m := seedAuctionListing(t, db, 1000, time.Hour)

	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-bids", map[string]any{
		"listingId":       m.Id().String(),
		"bidderId":        2001,
		"bidderAccountId": 3001,
		"amount":          1500,
	})
	res := doPost(t, ts, "/test/bids", body)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", res.StatusCode)
	}
	if len(rec.commands) != 1 {
		t.Fatalf("expected 1 emitted command, got %d", len(rec.commands))
	}
	var c mtsmsg.Command[mtsmsg.PlaceBidCommandBody]
	if err := json.Unmarshal(rec.commands[0].value, &c); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if c.TransactionId == uuid.Nil {
		t.Fatalf("expected non-nil transaction id")
	}
	if c.Type != mtsmsg.CommandPlaceBid || c.Body.WorldId != 0 || c.Body.Serial != m.Serial() || c.Body.BidderId != 2001 || c.Body.BidderAccountId != 3001 || c.Body.Amount != 1500 {
		t.Fatalf("bad command: %+v", c)
	}
	wantKey := kprod.CreateKey(int(uint32(2001)))
	if !bytes.Equal(rec.commands[0].key, wantKey) {
		t.Fatalf("key = %v, want bidder-keyed %v", rec.commands[0].key, wantKey)
	}
}

func TestSimulateBidOnFixedSale409s(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	m := seedFixedListing(t, db, 1000)

	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-bids", map[string]any{
		"listingId":       m.Id().String(),
		"bidderId":        2001,
		"bidderAccountId": 3001,
		"amount":          1500,
	})
	res := doPost(t, ts, "/test/bids", body)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
	if len(rec.commands) != 0 {
		t.Fatalf("expected no emission on 409, got %d", len(rec.commands))
	}
}

func TestSimulatePurchaseMissingBuyerId400s(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	m := seedFixedListing(t, db, 1000)

	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-purchases", map[string]any{
		"listingId":      m.Id().String(),
		"buyerId":        0,
		"buyerAccountId": 3001,
	})
	res := doPost(t, ts, "/test/purchases", body)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	if len(rec.commands) != 0 {
		t.Fatalf("expected no emission on 400, got %d", len(rec.commands))
	}
}

func TestSimulateBidZeroAmount400s(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	m := seedAuctionListing(t, db, 1000, time.Hour)

	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-bids", map[string]any{
		"listingId":       m.Id().String(),
		"bidderId":        2001,
		"bidderAccountId": 3001,
		"amount":          0,
	})
	res := doPost(t, ts, "/test/bids", body)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	if len(rec.commands) != 0 {
		t.Fatalf("expected no emission on 400, got %d", len(rec.commands))
	}
}

func TestSimulatePurchaseBuyNowEmitsBuyCommandWithBuyNowTrue(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	m := seedAuctionListing(t, db, 1000, time.Hour)

	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-purchases", map[string]any{
		"listingId":      m.Id().String(),
		"buyerId":        2001,
		"buyerAccountId": 3001,
		"buyNow":         true,
	})
	res := doPost(t, ts, "/test/purchases", body)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", res.StatusCode)
	}
	if len(rec.commands) != 1 {
		t.Fatalf("expected 1 emitted command, got %d", len(rec.commands))
	}
	var c mtsmsg.Command[mtsmsg.BuyCommandBody]
	if err := json.Unmarshal(rec.commands[0].value, &c); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !c.Body.BuyNow {
		t.Fatalf("expected BuyNow = true, got %+v", c.Body)
	}
}
