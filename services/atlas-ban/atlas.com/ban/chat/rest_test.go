package chat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestExtract(t *testing.T) {
	rm := RestModel{
		Id:         "0",
		Timestamp:  1720540800123,
		SenderId:   7,
		SenderName: "Alice",
		ChatType:   "GENERAL",
		Text:       "hello",
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Timestamp() != 1720540800123 || m.SenderId() != 7 || m.SenderName() != "Alice" ||
		m.ChatType() != "GENERAL" || m.Text() != "hello" {
		t.Errorf("mismatch: %+v", m)
	}
}

// chatHistoryResponse is a real JSON:API "chat-messages" list response
// carrying a relationships block, mirroring what atlas-messages's chat
// history endpoint could return. Served verbatim so the test drives the real
// api2go unmarshal path (via requests.SliceProvider -> GetRequest) rather
// than exercising Extract in isolation, which bypasses the unmarshal path
// entirely and would not have caught the missing EXT-01 relationship stubs.
const chatHistoryResponse = `{
  "data": [
    { "type": "chat-messages", "id": "0", "attributes": {
        "timestamp": 1720540800123, "senderId": 7, "senderName": "Alice",
        "chatType": "GENERAL", "text": "hello" },
      "relationships": { "sender": { "data": { "type": "characters", "id": "7" } } } }
  ]
}`

func TestRecentInvolving_UnmarshalsRelationships(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(chatHistoryResponse))
	}))
	defer srv.Close()

	t.Setenv("MESSAGES_SERVICE_URL", srv.URL+"/")

	ms, err := NewProcessor(logrus.New(), context.Background()).RecentInvolving([]uint32{7})
	if err != nil {
		t.Fatalf("RecentInvolving returned error: %v", err)
	}
	if gotPath != "/chat/history" {
		t.Errorf("request path: want /chat/history, got %q", gotPath)
	}
	if len(ms) != 1 {
		t.Fatalf("want 1 message, got %d", len(ms))
	}
	m := ms[0]
	if m.SenderId() != 7 || m.SenderName() != "Alice" || m.ChatType() != "GENERAL" || m.Text() != "hello" {
		t.Errorf("unexpected model: %+v", m)
	}
}

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		timestamp:  11,
		senderId:   22,
		senderName: "field3",
		chatType:   "field4",
		text:       "field5",
	}
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip lost data:\n got %+v\nwant %+v", got, m)
	}
}
