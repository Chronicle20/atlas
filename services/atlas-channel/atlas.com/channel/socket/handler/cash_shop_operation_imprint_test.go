package handler

import (
	messageCashShop "atlas-channel/kafka/message/cashshop"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// cashShopOperationsOptions is the minimal "operations" mode table
// CashShopOperationHandleFunc needs to resolve BUY_NAME_CHANGE (46) /
// BUY_WORLD_TRANSFER (49), mirroring the config isCashShopOperation reads at
// runtime (cash_shop_operation.go:200).
func cashShopOperationsOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			CashShopOperationBuyNameChange:    float64(46),
			CashShopOperationBuyWorldTransfer: float64(49),
		},
	}
}

func buyNameChangePacket(t *testing.T, oldName, newName string, serialNumber uint32) *request.Reader {
	t.Helper()
	w := response.NewWriter(logrus.New())
	w.WriteByte(46)
	w.WriteInt(serialNumber)
	w.WriteAsciiString(oldName)
	w.WriteAsciiString(newName)
	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

func buyWorldTransferPacket(t *testing.T, targetWorld byte, serialNumber uint32) *request.Reader {
	t.Helper()
	w := response.NewWriter(logrus.New())
	w.WriteByte(49)
	w.WriteInt(serialNumber)
	w.WriteInt(uint32(targetWorld))
	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

// jsonAPIAttrs writes a minimal {"data":{"type":...,"id":...,"attributes":...}}
// document, the shape every REST client this handler calls (character,
// commodity, pendingchange) expects — confirmed against
// pendingchange/processor_test.go's jsonAPIErrorBody sibling and
// character/rest.go's json tags.
func jsonAPIAttrs(resourceType, id string, attrs map[string]any) []byte {
	doc := map[string]any{
		"data": map[string]any{
			"type":       resourceType,
			"id":         id,
			"attributes": attrs,
		},
	}
	b, _ := json.Marshal(doc)
	return b
}

func jsonAPIErrorDetail(status, title, detail string) []byte {
	doc := map[string]any{
		"errors": []map[string]any{
			{"status": status, "title": title, "detail": detail},
		},
	}
	b, _ := json.Marshal(doc)
	return b
}

// newBuyHandlerTestServer stands in for both atlas-character (character
// lookup + pending-change creation) and atlas-data (commodity resolution),
// routed by path prefix. pendingChangeStatus/pendingChangeBody drive the
// POST .../pending-changes response; everything else in this test always
// succeeds. onPendingChangePost, if non-nil, is invoked synchronously right
// after the POST body is captured and before the response is written --
// tests use it to snapshot state (e.g. "has the purchase command been
// emitted yet?") at the moment the pending record is inserted, to pin the
// insert-first ordering.
func newBuyHandlerTestServer(t *testing.T, characterName string, templateId uint32, pendingChangeStatus int, pendingChangeBody []byte, onPendingChangePost func()) (*httptest.Server, *[]byte) {
	t.Helper()
	var capturedPost []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && bytes.Contains([]byte(r.URL.Path), []byte("pending-changes")):
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			capturedPost = buf
			if onPendingChangePost != nil {
				onPendingChangePost()
			}
			w.WriteHeader(pendingChangeStatus)
			if pendingChangeBody != nil {
				_, _ = w.Write(pendingChangeBody)
			}
		case bytes.Contains([]byte(r.URL.Path), []byte("/commodity/items/")):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(jsonAPIAttrs("commodities", "1", map[string]any{"itemId": templateId}))
		default:
			// character lookup
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(jsonAPIAttrs("characters", "1", map[string]any{"name": characterName}))
		}
	}))
	return srv, &capturedPost
}

// TestBuyNameChangeCreatesAPendingRequest pins task-227 task 38's whole
// contract for the name-change arm, in order:
//
//	(a) the PENDING record is inserted BEFORE the purchase command is emitted;
//	(b) the emitted REQUEST_PURCHASE command carries isPoints=false (i.e.
//	    currency=0), the commodity's serialNumber, and a non-zero TransactionId;
//	(c) that TransactionId is the pending record's own Id, so the consumer can
//	    resolve back to it;
//	(d) no done body is written from the handler -- the consumer answers the
//	    client now.
func TestBuyNameChangeCreatesAPendingRequest(t *testing.T) {
	const characterId = uint32(12345)
	const serialNumber = uint32(12345)
	pendingChangeId := uuid.New()

	captured, restore := installCapturingProducer()
	defer restore()

	var purchaseEmittedBeforePost bool
	onPost := func() {
		purchaseEmittedBeforePost = len((*captured)[messageCashShop.EnvCommandTopic]) > 0
	}

	srv, capturedPost := newBuyHandlerTestServer(t, "Romeo", 5990000, http.StatusCreated,
		jsonAPIAttrs("pending-changes", pendingChangeId.String(), map[string]any{"characterId": characterId, "type": "NAME_CHANGE", "status": "PENDING"}),
		onPost)
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, buyNameChangePacket(t, "Romeo", "Sierra", serialNumber), cashShopOperationsOptions())

	if !bytes.Contains(*capturedPost, []byte(`"requestedName":"Sierra"`)) {
		t.Errorf("POST body = %s, want requestedName Sierra", *capturedPost)
	}
	if !bytes.Contains(*capturedPost, []byte(`"type":"NAME_CHANGE"`)) {
		t.Errorf("POST body = %s, want type NAME_CHANGE", *capturedPost)
	}

	// (a) insert-first: the purchase command must not exist yet at the
	// moment the pending-change POST lands.
	if purchaseEmittedBeforePost {
		t.Fatal("purchase command was already emitted when the pending-change POST arrived -- insert-first violated")
	}

	msgs := (*captured)[messageCashShop.EnvCommandTopic]
	if len(msgs) != 1 {
		t.Fatalf("REQUEST_PURCHASE messages emitted = %d, want 1", len(msgs))
	}
	var cmd messageCashShop.Command[messageCashShop.RequestPurchaseCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal REQUEST_PURCHASE command: %v", err)
	}

	// (b)
	if cmd.Body.Currency != 0 {
		t.Errorf("Body.Currency = %d, want 0 (isPoints=false)", cmd.Body.Currency)
	}
	if cmd.Body.SerialNumber != serialNumber {
		t.Errorf("Body.SerialNumber = %d, want %d", cmd.Body.SerialNumber, serialNumber)
	}
	if cmd.Body.TransactionId == uuid.Nil {
		t.Fatal("Body.TransactionId is nil, want a non-zero id correlating to the pending record")
	}

	// (c)
	if cmd.Body.TransactionId != pendingChangeId {
		t.Errorf("Body.TransactionId = %s, want the pending record's own id %s", cmd.Body.TransactionId, pendingChangeId)
	}

	// (d)
	if rec.calls != 0 {
		t.Fatalf("handler announced %d packets directly, want 0 -- the consumer answers the client now (writer=%q)", rec.calls, rec.lastName)
	}
}

// TestBuyWorldTransferCreatesAPendingRequest mirrors
// TestBuyNameChangeCreatesAPendingRequest for the world-transfer arm.
func TestBuyWorldTransferCreatesAPendingRequest(t *testing.T) {
	const characterId = uint32(12346)
	const serialNumber = uint32(12346)
	pendingChangeId := uuid.New()

	captured, restore := installCapturingProducer()
	defer restore()

	var purchaseEmittedBeforePost bool
	onPost := func() {
		purchaseEmittedBeforePost = len((*captured)[messageCashShop.EnvCommandTopic]) > 0
	}

	srv, capturedPost := newBuyHandlerTestServer(t, "Romeo", 5990001, http.StatusCreated,
		jsonAPIAttrs("pending-changes", pendingChangeId.String(), map[string]any{"characterId": characterId, "type": "WORLD_TRANSFER", "status": "PENDING"}),
		onPost)
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, buyWorldTransferPacket(t, 2, serialNumber), cashShopOperationsOptions())

	if !bytes.Contains(*capturedPost, []byte(`"destinationWorldId":2`)) {
		t.Errorf("POST body = %s, want destinationWorldId 2", *capturedPost)
	}
	if !bytes.Contains(*capturedPost, []byte(`"type":"WORLD_TRANSFER"`)) {
		t.Errorf("POST body = %s, want type WORLD_TRANSFER", *capturedPost)
	}

	// (a) insert-first: the purchase command must not exist yet at the
	// moment the pending-change POST lands.
	if purchaseEmittedBeforePost {
		t.Fatal("purchase command was already emitted when the pending-change POST arrived -- insert-first violated")
	}

	msgs := (*captured)[messageCashShop.EnvCommandTopic]
	if len(msgs) != 1 {
		t.Fatalf("REQUEST_PURCHASE messages emitted = %d, want 1", len(msgs))
	}
	var cmd messageCashShop.Command[messageCashShop.RequestPurchaseCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal REQUEST_PURCHASE command: %v", err)
	}

	// (b)
	if cmd.Body.Currency != 0 {
		t.Errorf("Body.Currency = %d, want 0 (isPoints=false)", cmd.Body.Currency)
	}
	if cmd.Body.SerialNumber != serialNumber {
		t.Errorf("Body.SerialNumber = %d, want %d", cmd.Body.SerialNumber, serialNumber)
	}
	if cmd.Body.TransactionId == uuid.Nil {
		t.Fatal("Body.TransactionId is nil, want a non-zero id correlating to the pending record")
	}

	// (c)
	if cmd.Body.TransactionId != pendingChangeId {
		t.Errorf("Body.TransactionId = %s, want the pending record's own id %s", cmd.Body.TransactionId, pendingChangeId)
	}

	// (d)
	if rec.calls != 0 {
		t.Fatalf("handler announced %d packets directly, want 0 -- the consumer answers the client now (writer=%q)", rec.calls, rec.lastName)
	}
}

// newBuyHandlerTestServerWithInvalidTransactionId stands in for
// atlas-character with a pending-change POST that reports success but
// returns a non-UUID Id -- the (unreachable in practice, but not handled
// until this fix round) case Finding 2 covers. It also records whether the
// self-scoped cancel route (".../pending-changes/cancel") was called, so a
// test can assert the record was cancelled instead of silently orphaned.
func newBuyHandlerTestServerWithInvalidTransactionId(t *testing.T, characterName string, templateId uint32, changeType string) (srv *httptest.Server, cancelCalled *bool) {
	t.Helper()
	cancelCalled = new(bool)
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && bytes.Contains([]byte(r.URL.Path), []byte("pending-changes/cancel")):
			*cancelCalled = true
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && bytes.Contains([]byte(r.URL.Path), []byte("pending-changes")):
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(jsonAPIAttrs("pending-changes", "not-a-valid-uuid", map[string]any{"characterId": uint32(0), "type": changeType, "status": "PENDING"}))
		case bytes.Contains([]byte(r.URL.Path), []byte("/commodity/items/")):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(jsonAPIAttrs("commodities", "1", map[string]any{"itemId": templateId}))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(jsonAPIAttrs("characters", "1", map[string]any{"name": characterName}))
		}
	}))
	return srv, cancelCalled
}

// TestBuyNameChangeAbortsPurchaseWhenTransactionIdInvalid pins Finding 2 of
// task-227 task 38's fix round 1: a pending-change record whose Id fails to
// parse as a UUID must NOT result in a purchase being charged (uuid.Nil is
// "no correlation", not a safe fallback to charge against), and the
// just-created record must be cancelled rather than left orphaned.
func TestBuyNameChangeAbortsPurchaseWhenTransactionIdInvalid(t *testing.T) {
	const characterId = uint32(22345)
	captured, restore := installCapturingProducer()
	defer restore()

	srv, cancelCalled := newBuyHandlerTestServerWithInvalidTransactionId(t, "Romeo", 5990000, "NAME_CHANGE")
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, buyNameChangePacket(t, "Romeo", "Sierra", 22345), cashShopOperationsOptions())

	msgs := (*captured)[messageCashShop.EnvCommandTopic]
	if len(msgs) != 0 {
		t.Fatalf("REQUEST_PURCHASE messages emitted = %d, want 0 -- a transaction id parse failure must not charge the player", len(msgs))
	}
	if !*cancelCalled {
		t.Fatal("pending-change cancel route was not called -- the just-created record is orphaned")
	}
	if rec.calls != 1 || rec.lastName != chatpkt.WorldMessageWriter {
		t.Fatalf("announced calls=%d name=%q, want 1 pink-text rejection", rec.calls, rec.lastName)
	}
}

// TestBuyWorldTransferAbortsPurchaseWhenTransactionIdInvalid mirrors
// TestBuyNameChangeAbortsPurchaseWhenTransactionIdInvalid for the
// world-transfer arm, whose rejection route is TRANSFER_WORLD_FAILED rather
// than pink text.
func TestBuyWorldTransferAbortsPurchaseWhenTransactionIdInvalid(t *testing.T) {
	const characterId = uint32(22346)
	captured, restore := installCapturingProducer()
	defer restore()

	srv, cancelCalled := newBuyHandlerTestServerWithInvalidTransactionId(t, "Romeo", 5990001, "WORLD_TRANSFER")
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, buyWorldTransferPacket(t, 2, 22346), cashShopOperationsOptions())

	msgs := (*captured)[messageCashShop.EnvCommandTopic]
	if len(msgs) != 0 {
		t.Fatalf("REQUEST_PURCHASE messages emitted = %d, want 0 -- a transaction id parse failure must not charge the player", len(msgs))
	}
	if !*cancelCalled {
		t.Fatal("pending-change cancel route was not called -- the just-created record is orphaned")
	}
	if rec.calls != 1 || rec.lastName != cashcb.CashShopOperationWriter {
		t.Fatalf("announced calls=%d name=%q, want 1 TRANSFER_WORLD_FAILED", rec.calls, rec.lastName)
	}
}

// TestBuyNameChangeRejectionReachesTheClient pins FR-5.1: since no
// NAME_CHANGE_FAILED arm exists, a rejection must still reach the client —
// via pink text (chatpkt.WorldMessageWriter), not a silent drop.
func TestBuyNameChangeRejectionReachesTheClient(t *testing.T) {
	const characterId = uint32(12347)
	srv, _ := newBuyHandlerTestServer(t, "Romeo", 5990000, http.StatusConflict,
		jsonAPIErrorDetail("409", "Conflict", "name_reserved"), nil)
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, buyNameChangePacket(t, "Romeo", "Tango", 12347), cashShopOperationsOptions())

	if rec.calls != 1 {
		t.Fatalf("announced %d packets, want 1 (the pink-text rejection)", rec.calls)
	}
	if rec.lastName != chatpkt.WorldMessageWriter {
		t.Fatalf("announced writer = %q, want %q (pink text — no NAME_CHANGE_FAILED arm exists)", rec.lastName, chatpkt.WorldMessageWriter)
	}
	if !bytes.Contains(rec.lastBody, []byte("already in use")) {
		t.Errorf("body = %q, want a message reflecting name_reserved", rec.lastBody)
	}
}

// TestBuyNameChangeOldNameMismatchIsRefused pins the non-negotiable
// server-side authorization check: the client's OldName() is never trusted.
func TestBuyNameChangeOldNameMismatchIsRefused(t *testing.T) {
	const characterId = uint32(12348)
	var posted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posted = true
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jsonAPIAttrs("characters", "1", map[string]any{"name": "ActualName"}))
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, buyNameChangePacket(t, "SpoofedOldName", "NewName", 1), cashShopOperationsOptions())

	if posted {
		t.Fatal("pending-change POST fired despite an OldName mismatch")
	}
	if rec.calls != 1 || rec.lastName != chatpkt.WorldMessageWriter {
		t.Fatalf("announced calls=%d name=%q, want 1 pink-text rejection", rec.calls, rec.lastName)
	}
}
