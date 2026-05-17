package message_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type fakeEvent struct {
	ID int `json:"id"`
}

// newCapturingLogger returns a logger that writes JSON-formatted entries into
// the returned buffer at Debug level (so nothing is filtered out before the
// test inspects it).
func newCapturingLogger() (*logrus.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	l := logrus.New()
	l.SetOutput(buf)
	l.SetFormatter(&logrus.JSONFormatter{})
	l.SetLevel(logrus.DebugLevel)
	return l, buf
}

// decodeLogLines splits the buffer on newlines and parses each non-empty line
// as a JSON object.
func decodeLogLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("decoding log line %q: %v", line, err)
		}
		out = append(out, entry)
	}
	return out
}

func TestAdaptHandler_MalformedJSON_LogsErrorAndCommits(t *testing.T) {
	l, buf := newCapturingLogger()

	called := 0
	cfg := message.PersistentConfig[fakeEvent](func(_ logrus.FieldLogger, _ context.Context, _ fakeEvent) {
		called++
	})
	h := message.AdaptHandler[fakeEvent](cfg)

	msg := kafka.Message{
		Topic:     "EVENT_TOPIC_FAKE",
		Partition: 7,
		Offset:    123,
		Value:     []byte("{not json"),
	}

	persistent, err := h(l, context.Background(), msg)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !persistent {
		t.Fatalf("expected persistent=true on malformed message, got false")
	}
	if called != 0 {
		t.Fatalf("expected handler to NOT be invoked, was called %d times", called)
	}

	entries := decodeLogLines(t, buf)
	var errorEntries []map[string]any
	for _, e := range entries {
		if e["level"] == "error" {
			errorEntries = append(errorEntries, e)
		}
	}
	if len(errorEntries) != 1 {
		t.Fatalf("expected exactly 1 error-level log entry, got %d (all entries: %v)", len(errorEntries), entries)
	}

	e := errorEntries[0]
	if topic, _ := e["topic"].(string); topic != "EVENT_TOPIC_FAKE" {
		t.Errorf("expected topic=EVENT_TOPIC_FAKE, got %v", e["topic"])
	}
	// JSON numbers decode to float64 in map[string]any.
	if partition, _ := e["partition"].(float64); partition != 7 {
		t.Errorf("expected partition=7, got %v", e["partition"])
	}
	if offset, _ := e["offset"].(float64); offset != 123 {
		t.Errorf("expected offset=123, got %v", e["offset"])
	}
	if size, _ := e["payload_size"].(float64); size != float64(len(msg.Value)) {
		t.Errorf("expected payload_size=%d, got %v", len(msg.Value), e["payload_size"])
	}
	preview, _ := e["payload_preview"].(string)
	if !strings.Contains(preview, "{not json") {
		t.Errorf("expected payload_preview to contain raw bytes, got %q", preview)
	}
	wantType := fmt.Sprintf("%T", *new(fakeEvent))
	if mt, _ := e["message_type"].(string); mt != wantType {
		t.Errorf("expected message_type=%q, got %v", wantType, e["message_type"])
	}
	if msgText, _ := e["msg"].(string); !strings.Contains(msgText, "offset will be committed and the message dropped") {
		t.Errorf("expected msg to mention commit-and-drop, got %q", msgText)
	}
	// logrus's WithError convention surfaces the underlying error under the "error" field.
	if _, ok := e["error"]; !ok {
		t.Errorf("expected the underlying unmarshal error to be present under the \"error\" field, entry: %v", e)
	}
}

func TestAdaptHandler_ValidMessage_InvokesHandlerAndDoesNotErrorLog(t *testing.T) {
	l, buf := newCapturingLogger()

	called := 0
	var received fakeEvent
	cfg := message.PersistentConfig[fakeEvent](func(_ logrus.FieldLogger, _ context.Context, m fakeEvent) {
		called++
		received = m
	})
	h := message.AdaptHandler[fakeEvent](cfg)

	payload, err := json.Marshal(fakeEvent{ID: 42})
	if err != nil {
		t.Fatalf("marshalling fixture: %v", err)
	}
	msg := kafka.Message{
		Topic:     "EVENT_TOPIC_FAKE",
		Partition: 0,
		Offset:    1,
		Value:     payload,
	}

	persistent, err := h(l, context.Background(), msg)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !persistent {
		t.Fatalf("expected persistent=true from PersistentConfig, got false")
	}
	if called != 1 {
		t.Fatalf("expected handler called exactly once, got %d", called)
	}
	if received.ID != 42 {
		t.Fatalf("expected ID=42, got %d", received.ID)
	}

	for _, e := range decodeLogLines(t, buf) {
		if e["level"] == "error" {
			t.Fatalf("expected no error-level log entries on happy path, got %v", e)
		}
	}
}

func TestAdaptHandler_OversizedPayload_TruncatesPreview(t *testing.T) {
	l, buf := newCapturingLogger()

	cfg := message.PersistentConfig[fakeEvent](func(_ logrus.FieldLogger, _ context.Context, _ fakeEvent) {
		t.Fatal("handler must not be invoked on malformed message")
	})
	h := message.AdaptHandler[fakeEvent](cfg)

	// 10 KB of 'A' bytes — not valid JSON, so unmarshal will fail and trigger
	// the logging path.
	body := bytes.Repeat([]byte("A"), 10000)
	msg := kafka.Message{
		Topic:     "EVENT_TOPIC_FAKE",
		Partition: 3,
		Offset:    9999,
		Value:     body,
	}

	persistent, err := h(l, context.Background(), msg)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !persistent {
		t.Fatalf("expected persistent=true, got false")
	}

	entries := decodeLogLines(t, buf)
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 log entry, got %d", len(entries))
	}
	e := entries[0]
	if e["level"] != "error" {
		t.Fatalf("expected error-level entry, got level=%v", e["level"])
	}
	if size, _ := e["payload_size"].(float64); size != float64(len(body)) {
		t.Errorf("expected payload_size=%d (full length), got %v", len(body), e["payload_size"])
	}

	preview, _ := e["payload_preview"].(string)
	// payload_preview is %q-quoted, so the field value is a Go-quoted string
	// literal. Strip the surrounding quotes to inspect the prefix bytes.
	if len(preview) < 2 || preview[0] != '"' || preview[len(preview)-1] != '"' {
		t.Fatalf("expected payload_preview to be a Go-quoted string, got %q", preview)
	}
	unquoted := preview[1 : len(preview)-1]
	// Truncated to previewMax=256 raw bytes. With 'A' bytes there's no
	// escaping, so the unquoted length must equal exactly 256.
	if len(unquoted) != 256 {
		t.Errorf("expected payload_preview unquoted length=256, got %d", len(unquoted))
	}
	if !strings.HasPrefix(unquoted, "AAAA") {
		t.Errorf("expected preview to start with AAAA, got %q", unquoted[:min(8, len(unquoted))])
	}
}

func TestAdaptHandler_ValidatorRejects_NoErrorLog(t *testing.T) {
	l, buf := newCapturingLogger()

	called := 0
	validator := func(_ logrus.FieldLogger, _ context.Context, _ fakeEvent) bool { return false }
	cfg := message.OneTimeConfig[fakeEvent](validator, func(_ logrus.FieldLogger, _ context.Context, _ fakeEvent) {
		called++
	})
	h := message.AdaptHandler[fakeEvent](cfg)

	payload, err := json.Marshal(fakeEvent{ID: 1})
	if err != nil {
		t.Fatalf("marshalling fixture: %v", err)
	}
	msg := kafka.Message{
		Topic:     "EVENT_TOPIC_FAKE",
		Partition: 0,
		Offset:    1,
		Value:     payload,
	}

	persistent, err := h(l, context.Background(), msg)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !persistent {
		t.Fatalf("expected (true, nil) on validator rejection (commit-and-skip), got persistent=false")
	}
	if called != 0 {
		t.Fatalf("expected handler NOT to be invoked when validator rejects, was called %d times", called)
	}

	for _, e := range decodeLogLines(t, buf) {
		if e["level"] == "error" {
			t.Fatalf("expected no error-level log entry on validator rejection, got %v", e)
		}
	}
}
