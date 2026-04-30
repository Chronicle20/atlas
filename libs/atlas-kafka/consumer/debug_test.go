package consumer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"go.opentelemetry.io/otel"
)

type debugDoc struct {
	Data []debugResource `json:"data"`
}

type debugResource struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Attributes debugAttributes `json:"attributes"`
}

type debugAttributes struct {
	Name                string    `json:"name"`
	Topic               string    `json:"topic"`
	GroupID             string    `json:"groupId"`
	Brokers             []string  `json:"brokers"`
	AliveSince          time.Time `json:"aliveSince"`
	LastFetchAt         time.Time `json:"lastFetchAt"`
	LastErrorAt         time.Time `json:"lastErrorAt"`
	LastError           string    `json:"lastError"`
	RecreateCount       int       `json:"recreateCount"`
	HandlerCount        int       `json:"handlerCount"`
	LastTimeoutAt       time.Time `json:"lastTimeoutAt"`
	ConsecutiveTimeouts int       `json:"consecutiveTimeouts"`
}

func decodeDebug(t *testing.T, body []byte) debugDoc {
	t.Helper()
	var d debugDoc
	if err := json.Unmarshal(body, &d); err != nil {
		t.Fatalf("decoding debug response: %v", err)
	}
	return d
}

func TestDebugHandler_Empty(t *testing.T) {
	consumer.ResetInstance()
	cm := consumer.GetManager()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/consumers", nil)
	cm.DebugHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/vnd.api+json" {
		t.Fatalf("expected application/vnd.api+json, got %q", ct)
	}
	doc := decodeDebug(t, rec.Body.Bytes())
	if len(doc.Data) != 0 {
		t.Fatalf("expected 0 consumers, got %d", len(doc.Data))
	}
}

func TestDebugHandler_RejectsNonGet(t *testing.T) {
	consumer.ResetInstance()
	cm := consumer.GetManager()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/debug/consumers", nil)
	cm.DebugHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET" {
		t.Fatalf("expected Allow: GET, got %q", allow)
	}
}

func TestDebugHandler_PopulatedConsumer(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	reader := &ChannelMockReader{msgCh: make(chan kafka.Message, 1)}
	reader.msgCh <- kafka.Message{Value: []byte("warmup")}

	rp := consumer.ConfigReaderProducer(func(config kafka.ReaderConfig) consumer.KafkaReader {
		return reader
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{"broker-1:9092", "broker-2:9092"}, "asset_status_event", "EVENT_TOPIC_ASSET_STATUS", "Channel Service - test")
	cm.AddConsumer(l, ctx, wg)(c)

	handlerDone := make(chan struct{})
	_, _ = cm.RegisterHandler("EVENT_TOPIC_ASSET_STATUS", func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
		close(handlerDone)
		return true, nil
	})

	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was never invoked")
	}
	// Give the fetch loop a moment to record lastFetchAt after the handler completes.
	time.Sleep(20 * time.Millisecond)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/consumers", nil)
	cm.DebugHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	doc := decodeDebug(t, rec.Body.Bytes())
	if len(doc.Data) != 1 {
		t.Fatalf("expected 1 consumer, got %d", len(doc.Data))
	}
	r := doc.Data[0]
	if r.Type != "consumers" {
		t.Fatalf("expected type=consumers, got %q", r.Type)
	}
	if r.ID != "EVENT_TOPIC_ASSET_STATUS" {
		t.Fatalf("expected id=EVENT_TOPIC_ASSET_STATUS, got %q", r.ID)
	}
	a := r.Attributes
	if a.Name != "asset_status_event" {
		t.Fatalf("unexpected name: %q", a.Name)
	}
	if a.Topic != "EVENT_TOPIC_ASSET_STATUS" {
		t.Fatalf("unexpected topic: %q", a.Topic)
	}
	if a.GroupID != "Channel Service - test" {
		t.Fatalf("unexpected groupId: %q", a.GroupID)
	}
	if len(a.Brokers) != 2 || a.Brokers[0] != "broker-1:9092" || a.Brokers[1] != "broker-2:9092" {
		t.Fatalf("unexpected brokers: %v", a.Brokers)
	}
	if a.HandlerCount != 1 {
		t.Fatalf("expected handlerCount=1, got %d", a.HandlerCount)
	}
	if a.AliveSince.IsZero() {
		t.Fatal("expected aliveSince to be set")
	}
	if a.RecreateCount != 0 {
		t.Fatalf("expected recreateCount=0 on steady-state first reader, got %d", a.RecreateCount)
	}
	if a.LastFetchAt.IsZero() {
		t.Fatal("expected lastFetchAt to be set after successful fetch")
	}
	if a.ConsecutiveTimeouts != 0 {
		t.Fatalf("expected consecutiveTimeouts=0 after a successful fetch, got %d", a.ConsecutiveTimeouts)
	}
	if !a.LastTimeoutAt.IsZero() {
		t.Fatalf("expected lastTimeoutAt zero on a consumer that has never timed out, got %v", a.LastTimeoutAt)
	}
}

