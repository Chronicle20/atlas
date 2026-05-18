# Kafka Message Error Handling — Surface Silent Drops — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every malformed Kafka message logged at Error level with structured topic / partition / offset / payload context, while preserving the current commit-and-skip return contract (`true, nil`) and the existing `Handler[M]` signature.

**Architecture:** Surgical edit to `libs/atlas-kafka/message/handler.go` — add one package-level constant, replace the single Debug log inside `AdaptHandler[M]` with an Error log carrying `logrus.Fields`, leave the rest of the function (validator path, persistent/one-time path, `adapt[M]`) untouched. A new `handler_test.go` covers four cases (happy / malformed / oversized / validator-rejects) by routing a fresh `logrus.Logger` through a `bytes.Buffer` with the JSON formatter, then parsing each emitted line as JSON to assert level, message text, and structured fields.

**Tech Stack:** Go 1.25, `github.com/sirupsen/logrus`, `github.com/segmentio/kafka-go`, `encoding/json`, standard `testing` (no testify — match the existing `t.Fatalf` style in `libs/atlas-kafka`).

---

## File Structure

- **Modify:** `libs/atlas-kafka/message/handler.go`
  - Add `const previewMax = 256` at package scope.
  - Add imports: `fmt`.
  - Inside `AdaptHandler[M]`, replace the single `Debugf` call with the new structured `Errorf` call. Everything else in the file stays byte-identical.
- **Create:** `libs/atlas-kafka/message/handler_test.go`
  - Package: `message_test` (black-box; we only need exported symbols).
  - Defines one local `fakeEvent` struct.
  - Defines one helper `decodeLogLines(t, buf)` that parses each newline-terminated JSON object out of the captured logrus buffer.
  - Four `Test*` functions, one per PRD §4.3 case.

No other files anywhere in the repo are modified. No `go.mod`, `go.sum`, Dockerfile, or k8s manifest changes — `libs/atlas-kafka` already has `logrus` and `kafka-go` as direct dependencies and consumers pick up the new behavior via `go.work` replace on next rebuild.

---

## Task 1: Pin the current behavior with a failing test for the malformed-JSON path

**Files:**
- Create: `libs/atlas-kafka/message/handler_test.go`

**Rationale:** Before changing any production code, write the test that captures the *new* expected behavior. The current Debug log will not satisfy the Error-level assertion, so this test will fail until Task 2 lands. This is the TDD red step.

- [ ] **Step 1.1: Create the new test file with the malformed-JSON test**

Create `libs/atlas-kafka/message/handler_test.go`:

```go
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
```

- [ ] **Step 1.2: Run the test and verify it fails**

Run: `cd libs/atlas-kafka && go test ./message/ -run TestAdaptHandler_MalformedJSON_LogsErrorAndCommits -v`

Expected: FAIL. The current production code logs at Debug level with no structured fields, so the assertion `expected exactly 1 error-level log entry, got 0` (or similar — depending on which assertion trips first) should fire. Confirm the *reason* for failure is the missing Error log, not a compile error.

- [ ] **Step 1.3: Commit the failing test**

```bash
git add libs/atlas-kafka/message/handler_test.go
git commit -m "test(atlas-kafka): pin Error-level unmarshal-failure logging"
```

---

## Task 2: Implement the Error-level structured log

**Files:**
- Modify: `libs/atlas-kafka/message/handler.go:1-11` (imports), `:42-49` (the unmarshal-error branch), plus one new top-level const.

- [ ] **Step 2.1: Update `handler.go` to log at Error level with the required structured fields**

Replace the entire contents of `libs/atlas-kafka/message/handler.go` with:

```go
package message

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// previewMax bounds the byte prefix of a malformed Kafka payload that we
// include in the error log. Sized to capture the leading envelope of a typical
// JSON event without risking oversized log lines or sensitive-data exposure.
const previewMax = 256

type Validator[M any] func(l logrus.FieldLogger, ctx context.Context, m M) bool

type Handler[M any] func(l logrus.FieldLogger, ctx context.Context, m M)

type Config[M any] struct {
	persistent bool
	validator  Validator[M]
	handler    Handler[M]
}

//goland:noinspection GoUnusedExportedFunction
func PersistentConfig[M any](handler Handler[M]) Config[M] {
	return Config[M]{
		persistent: true,
		validator:  func(l logrus.FieldLogger, ctx context.Context, m M) bool { return true },
		handler:    handler,
	}
}

//goland:noinspection GoUnusedExportedFunction
func OneTimeConfig[M any](validator Validator[M], handler Handler[M]) Config[M] {
	return Config[M]{
		persistent: false,
		validator:  validator,
		handler:    handler,
	}
}

//goland:noinspection GoUnusedExportedFunction
func AdaptHandler[M any](config Config[M]) handler.Handler {
	h := func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		tem := model.Map[kafka.Message, M](adapt[M])(model.FixedProvider(msg))
		m, err := tem()
		if err != nil {
			preview := msg.Value
			if len(preview) > previewMax {
				preview = preview[:previewMax]
			}
			l.WithFields(logrus.Fields{
				"topic":           msg.Topic,
				"partition":       msg.Partition,
				"offset":          msg.Offset,
				"payload_size":    len(msg.Value),
				"payload_preview": fmt.Sprintf("%q", preview),
				"message_type":    fmt.Sprintf("%T", *new(M)),
			}).WithError(err).Errorf("Failed to unmarshal Kafka message; offset will be committed and the message dropped.")
			return true, nil
		}

		process := config.validator(l, ctx, m)
		if !process {
			return true, nil
		}

		config.handler(l, ctx, m)
		return config.persistent, nil
	}
	return h
}

func adapt[M any](msg kafka.Message) (M, error) {
	var event M
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		return event, err
	}
	return event, nil
}
```

- [ ] **Step 2.2: Run the Task 1 test and verify it now passes**

Run: `cd libs/atlas-kafka && go test ./message/ -run TestAdaptHandler_MalformedJSON_LogsErrorAndCommits -v`

Expected: PASS.

- [ ] **Step 2.3: Run `go vet` on the lib**

Run: `cd libs/atlas-kafka && go vet ./...`

Expected: no output (clean exit).

- [ ] **Step 2.4: Commit the production change**

```bash
git add libs/atlas-kafka/message/handler.go
git commit -m "feat(atlas-kafka): log unmarshal failures at Error with structured fields"
```

---

## Task 3: Add the happy-path test

**Files:**
- Modify: `libs/atlas-kafka/message/handler_test.go` (append).

- [ ] **Step 3.1: Append the happy-path test**

Append to `libs/atlas-kafka/message/handler_test.go`:

```go
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
```

- [ ] **Step 3.2: Run the happy-path test**

Run: `cd libs/atlas-kafka && go test ./message/ -run TestAdaptHandler_ValidMessage_InvokesHandlerAndDoesNotErrorLog -v`

Expected: PASS.

- [ ] **Step 3.3: Commit**

```bash
git add libs/atlas-kafka/message/handler_test.go
git commit -m "test(atlas-kafka): cover happy-path AdaptHandler dispatch"
```

---

## Task 4: Add the oversized-payload truncation test

**Files:**
- Modify: `libs/atlas-kafka/message/handler_test.go` (append).

- [ ] **Step 4.1: Append the oversized-payload test**

Append to `libs/atlas-kafka/message/handler_test.go`:

```go
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
```

- [ ] **Step 4.2: Run the oversized-payload test**

Run: `cd libs/atlas-kafka && go test ./message/ -run TestAdaptHandler_OversizedPayload_TruncatesPreview -v`

Expected: PASS.

- [ ] **Step 4.3: Commit**

```bash
git add libs/atlas-kafka/message/handler_test.go
git commit -m "test(atlas-kafka): pin payload preview truncation at 256 bytes"
```

---

## Task 5: Add the validator-rejects test

**Files:**
- Modify: `libs/atlas-kafka/message/handler_test.go` (append).

- [ ] **Step 5.1: Append the validator-rejects test**

Append to `libs/atlas-kafka/message/handler_test.go`:

```go
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
```

- [ ] **Step 5.2: Run the validator-rejects test**

Run: `cd libs/atlas-kafka && go test ./message/ -run TestAdaptHandler_ValidatorRejects_NoErrorLog -v`

Expected: PASS.

- [ ] **Step 5.3: Commit**

```bash
git add libs/atlas-kafka/message/handler_test.go
git commit -m "test(atlas-kafka): pin validator-rejection path stays silent"
```

---

## Task 6: Full lib verification

**Files:** none.

- [ ] **Step 6.1: Run the full `atlas-kafka` test suite with the race detector**

Run: `cd libs/atlas-kafka && go test -race ./...`

Expected: all packages PASS, no race warnings. If anything in the lib started failing because of a side effect of the handler change, stop and investigate — do not proceed.

- [ ] **Step 6.2: Run `go vet` on the full lib**

Run: `cd libs/atlas-kafka && go vet ./...`

Expected: no output.

- [ ] **Step 6.3: Run `go build` on the full lib**

Run: `cd libs/atlas-kafka && go build ./...`

Expected: no output.

---

## Task 7: Workspace-level verification (sanity check across consumers)

**Files:** none.

The PRD acceptance criteria require `go build ./...` to pass in every service that consumes the lib. The Go workspace (`go.work` at repo root) builds all services with the in-tree `libs/atlas-kafka`, so a single workspace-level build is enough to catch any consumer regression caused by our change. We do not need a per-service `docker build` in this task because **no `go.mod` files and no `Dockerfile`s change** (the four-place lib list in each service's Dockerfile already includes `libs/atlas-kafka`).

- [ ] **Step 7.1: Run workspace-level vet**

Run (from worktree root): `go vet ./...`

Expected: no output. If any service surfaces a vet error tied to the new log call, stop and fix.

- [ ] **Step 7.2: Run workspace-level build**

Run (from worktree root): `go build ./...`

Expected: no output. Validates that every service still compiles with the updated lib bytes.

- [ ] **Step 7.3: Run workspace-level race tests for the lib itself**

Run (from worktree root): `go test -race ./libs/atlas-kafka/...`

Expected: PASS. (Running `go test -race ./...` across the entire repo is much heavier and not required by this task — only `libs/atlas-kafka` changed.)

- [ ] **Step 7.4: Commit nothing (verification only) and document the result in the PR description**

No commit. Confirm the three steps above all came back clean before declaring the task complete. The PR description must:
- Note that this is a single bundled PR for `libs/atlas-kafka` only.
- Explicitly list the deferred follow-ups (handler error return, Prometheus/OTel metrics, DLQ topic, producer hardening) so they remain visible.

---

## Acceptance Verification

Map each PRD §10 item to a task:

- [x] `handler.go` logs unmarshal failures at Error with the required structured fields → Task 2.
- [x] `handler.go` still returns `(true, nil)` on unmarshal failure → Asserted in Task 1.
- [x] No exported symbols change signatures → Visual diff after Task 2; the function bodies inside `PersistentConfig`, `OneTimeConfig`, `AdaptHandler`, `adapt` keep the same parameter and return shapes.
- [x] New unit tests cover the four cases under `go test -race ./...` → Tasks 1, 3, 4, 5; verified together in Task 6.
- [x] `go vet ./...` passes in `libs/atlas-kafka` → Task 6.
- [x] `go build ./...` passes in `libs/atlas-kafka` and every service consuming it → Task 6 + Task 7.
- [x] PR description notes deferred follow-ups → Task 7 Step 4.
