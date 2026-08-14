package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
// succeeds.
func newBuyHandlerTestServer(t *testing.T, characterName string, templateId uint32, pendingChangeStatus int, pendingChangeBody []byte) (*httptest.Server, *[]byte) {
	t.Helper()
	var capturedPost []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && bytes.Contains([]byte(r.URL.Path), []byte("pending-changes")):
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			capturedPost = buf
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

func TestBuyNameChangeCreatesAPendingRequest(t *testing.T) {
	const characterId = uint32(12345)
	srv, captured := newBuyHandlerTestServer(t, "Romeo", 5990000, http.StatusCreated,
		jsonAPIAttrs("pending-changes", "1", map[string]any{"characterId": characterId, "type": "NAME_CHANGE", "status": "PENDING"}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, buyNameChangePacket(t, "Romeo", "Sierra", 12345), cashShopOperationsOptions())

	if rec.calls != 1 {
		t.Fatalf("announced %d packets, want 1 (the success arm)", rec.calls)
	}
	if rec.lastName != cashcb.CashShopOperationWriter {
		t.Errorf("announced writer = %q, want %q", rec.lastName, cashcb.CashShopOperationWriter)
	}
	if !bytes.Contains(*captured, []byte(`"requestedName":"Sierra"`)) {
		t.Errorf("POST body = %s, want requestedName Sierra", *captured)
	}
	if !bytes.Contains(*captured, []byte(`"type":"NAME_CHANGE"`)) {
		t.Errorf("POST body = %s, want type NAME_CHANGE", *captured)
	}
}

func TestBuyWorldTransferCreatesAPendingRequest(t *testing.T) {
	const characterId = uint32(12346)
	srv, captured := newBuyHandlerTestServer(t, "Romeo", 5990001, http.StatusCreated,
		jsonAPIAttrs("pending-changes", "1", map[string]any{"characterId": characterId, "type": "WORLD_TRANSFER", "status": "PENDING"}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, buyWorldTransferPacket(t, 2, 12346), cashShopOperationsOptions())

	if rec.calls != 1 {
		t.Fatalf("announced %d packets, want 1 (the success arm)", rec.calls)
	}
	if rec.lastName != cashcb.CashShopOperationWriter {
		t.Errorf("announced writer = %q, want %q", rec.lastName, cashcb.CashShopOperationWriter)
	}
	if !bytes.Contains(*captured, []byte(`"destinationWorldId":2`)) {
		t.Errorf("POST body = %s, want destinationWorldId 2", *captured)
	}
	if !bytes.Contains(*captured, []byte(`"type":"WORLD_TRANSFER"`)) {
		t.Errorf("POST body = %s, want type WORLD_TRANSFER", *captured)
	}
}

// TestBuyNameChangeRejectionReachesTheClient pins FR-5.1: since no
// NAME_CHANGE_FAILED arm exists, a rejection must still reach the client —
// via pink text (chatpkt.WorldMessageWriter), not a silent drop.
func TestBuyNameChangeRejectionReachesTheClient(t *testing.T) {
	const characterId = uint32(12347)
	srv, _ := newBuyHandlerTestServer(t, "Romeo", 5990000, http.StatusConflict,
		jsonAPIErrorDetail("409", "Conflict", "name_reserved"))
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
