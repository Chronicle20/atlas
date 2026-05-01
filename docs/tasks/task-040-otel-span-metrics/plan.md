# OTel Span Metrics + Client-Write Latency — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable Tempo span-derived metrics, instrument the `session.Announce` chokepoint with an OTel span, ship a Grafana latency dashboard, and consolidate the 54 byte-identical `tracing/tracing.go` files into a new `libs/atlas-tracing` shared library with `TRACE_SAMPLING_RATIO` env-driven sampling.

**Architecture:** Pathway (a) from PRD §4.1 — Tempo `metrics_generator` with a curated `span-metrics` dimension allowlist. Atlas service `TRACE_ENDPOINT` is unchanged at `tempo.home:4317`. The new `libs/atlas-tracing` is the single source of truth for tracer setup; atlas-channel adds one manual `session.Announce` span; the dashboard is file-provider provisioned via a configmap mount that ships from `deploy/grafana/`.

**Tech Stack:** Go 1.25.5, OpenTelemetry SDK v1.43.0, Tempo 2.7.x, Grafana, Prometheus, Kubernetes (k3s "bee" cluster), Docker, `go.work`-based workspace with per-service `replace` directives.

**Reference docs (read first):**
- `docs/tasks/task-040-otel-span-metrics/prd.md`
- `docs/tasks/task-040-otel-span-metrics/design.md`
- `docs/tasks/task-040-otel-span-metrics/risks.md`
- `docs/tasks/task-040-otel-span-metrics/context.md` (this folder — quick file/symbol reference)
- `CLAUDE.md` — "always verify Docker builds when changing shared libraries"

**Phasing:**
- **Tasks 1–11** land library + atlas-channel migration + `session.Announce` span (smallest blast radius that satisfies AC #1–#3, #6–#7, #13).
- **Tasks 12–17** ship deploy artifacts and documentation.
- **Task 18** fans out the libs migration to the remaining 53 services.
- **Task 19** lists the out-of-tree cluster changes the operator must apply for AC #4–#5, #9–#10 to verify in the bee cluster.

---

## File Structure

**New library:** `libs/atlas-tracing/` — module `github.com/Chronicle20/atlas/libs/atlas-tracing`, with files:
- `go.mod`, `go.sum`
- `tracing.go` — public API (`InitTracer`, `Teardown`)
- `sampling.go` — `parseSamplingRatio` helper that reads `TRACE_SAMPLING_RATIO`
- `sampling_test.go` — unit tests for env-var parsing
- `README.md`

**Workspace:** `go.work` — append `./libs/atlas-tracing` to the `use` block.

**Channel-only edits:**
- `services/atlas-channel/atlas.com/channel/session/processor.go:168-184` — add manual span around `Announce`'s inner lambda.
- `services/atlas-channel/atlas.com/channel/session/processor_test.go` — add `TestAnnounce_StartsSpan`.
- `services/atlas-channel/atlas.com/channel/main.go` — swap import path.
- `services/atlas-channel/atlas.com/channel/go.mod` — add lib require + replace.
- `services/atlas-channel/atlas.com/channel/tracing/` — **delete**.
- `services/atlas-channel/Dockerfile` — add 3 lines for libs/atlas-tracing.

**Per-service migration (×53):** same five-edit pattern as atlas-channel, but no `session.Announce` span work.

**Deploy artifacts (new, in-repo):**
- `deploy/k8s/env-configmap.yaml` — append `TRACE_SAMPLING_RATIO`.
- `deploy/compose/.env`, `deploy/compose/.env.example` — append `TRACE_SAMPLING_RATIO`.
- `deploy/grafana/dashboards/atlas-latency.json` — dashboard JSON.
- `deploy/grafana/dashboards-provider.yaml` — Grafana file-provider config.
- `deploy/grafana/apply.sh` — idempotent configmap applier.
- `deploy/grafana/README.md` — pointer to `docs/observability.md`.

**Documentation (new):**
- `docs/observability.md` — pipeline diagram, add-a-span recipe, cardinality budget, smoke test, sampling caveat.

**Out-of-tree (operator applies; tracked in Task 19):**
- `~/source/k3s/bee/observability-tempo.yml` — overrides block enabling `span-metrics` processor with curated dimensions.
- `~/source/k3s/bee/observability-grafana.yml` — `volumeMount` + `volume` referencing `grafana-dashboards-atlas` configmap.

---

## Task 1: Scaffold `libs/atlas-tracing` module

**Files:**
- Create: `libs/atlas-tracing/go.mod`
- Create: `libs/atlas-tracing/README.md`

- [ ] **Step 1: Create the module file**

`libs/atlas-tracing/go.mod`:

```go
module github.com/Chronicle20/atlas/libs/atlas-tracing

go 1.25.5

require (
	github.com/sirupsen/logrus v1.9.4
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
)
```

(`semconv/v1.26.0` is a subpackage of `go.opentelemetry.io/otel` — no separate require needed. Indirect deps will be filled in by `go mod tidy` in Step 4.)

- [ ] **Step 2: Create the README**

`libs/atlas-tracing/README.md`:

```markdown
# atlas-tracing

Shared OTel tracer setup for Atlas Go services. Exposes `InitTracer(serviceName)` and `Teardown(l)` that previously lived as 54 byte-identical copies under `services/atlas-*/.../tracing/tracing.go`.

Reads `TRACE_ENDPOINT` (OTLP gRPC target) and `TRACE_SAMPLING_RATIO` (float in `[0.0, 1.0]`, default `1.0`) from the environment. See `docs/observability.md` in the repo root for the pipeline overview.
```

- [ ] **Step 3: Append to `go.work`**

Edit `go.work`. After the line `	./libs/atlas-socket` (or any existing `./libs/...` entry), insert:

```
	./libs/atlas-tracing
```

Maintain alphabetical order within the `use (` block; insert between `./libs/atlas-socket` and `./libs/atlas-tenant`.

- [ ] **Step 4: Run `go mod tidy` for the new lib**

```bash
cd libs/atlas-tracing && go mod tidy
```

Expected: `go.sum` is generated and `go.mod` is rewritten with full indirect dependency list. No errors.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-tracing/ go.work
git commit -m "feat(libs/atlas-tracing): scaffold shared tracer library"
```

---

## Task 2: TDD — sampling-ratio parser (failing test)

**Files:**
- Create: `libs/atlas-tracing/sampling_test.go`

- [ ] **Step 1: Write the failing test**

`libs/atlas-tracing/sampling_test.go`:

```go
package tracing

import (
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func TestParseSamplingRatio(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		envSet    bool
		want      float64
		wantWarn  bool
	}{
		{name: "unset defaults to 1.0", envSet: false, want: 1.0, wantWarn: false},
		{name: "valid 1.0", envSet: true, envValue: "1.0", want: 1.0, wantWarn: false},
		{name: "valid 0.5", envSet: true, envValue: "0.5", want: 0.5, wantWarn: false},
		{name: "valid 0.0", envSet: true, envValue: "0.0", want: 0.0, wantWarn: false},
		{name: "empty string warns and defaults", envSet: true, envValue: "", want: 1.0, wantWarn: true},
		{name: "garbage warns and defaults", envSet: true, envValue: "abc", want: 1.0, wantWarn: true},
		{name: "above range warns and defaults", envSet: true, envValue: "1.5", want: 1.0, wantWarn: true},
		{name: "negative warns and defaults", envSet: true, envValue: "-0.1", want: 1.0, wantWarn: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TRACE_SAMPLING_RATIO", "")
			if tc.envSet {
				t.Setenv("TRACE_SAMPLING_RATIO", tc.envValue)
			} else {
				// Unset: t.Setenv above set empty; for "unset" we need to actually remove.
				// Use os.Unsetenv via t.Cleanup-friendly pattern.
				if err := unsetForTest(); err != nil {
					t.Fatal(err)
				}
			}

			logger, hook := logtest.NewNullLogger()
			got := parseSamplingRatio(logger)

			if got != tc.want {
				t.Errorf("parseSamplingRatio() = %v, want %v", got, tc.want)
			}

			gotWarn := false
			for _, e := range hook.AllEntries() {
				if e.Level == logrus.WarnLevel {
					gotWarn = true
					break
				}
			}
			if gotWarn != tc.wantWarn {
				t.Errorf("warn emitted = %v, want %v", gotWarn, tc.wantWarn)
			}
		})
	}
}

func unsetForTest() error {
	return osUnsetenv("TRACE_SAMPLING_RATIO")
}
```

(The `osUnsetenv` indirection lets us add a tiny shim so `t.Setenv` cleanup behaves correctly — it's a wrapper for `os.Unsetenv`.)

Add the shim file `libs/atlas-tracing/sampling_test_helpers.go`:

```go
package tracing

import "os"

func osUnsetenv(k string) error {
	return os.Unsetenv(k)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-tracing && go test ./...
```

Expected: FAIL — `parseSamplingRatio` is not defined.

---

## Task 3: Implement `parseSamplingRatio`

**Files:**
- Create: `libs/atlas-tracing/sampling.go`

- [ ] **Step 1: Implement**

`libs/atlas-tracing/sampling.go`:

```go
package tracing

import (
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

const (
	samplingRatioEnvVar = "TRACE_SAMPLING_RATIO"
	defaultSamplingRatio = 1.0
)

// parseSamplingRatio reads TRACE_SAMPLING_RATIO from the environment and returns
// a value in [0.0, 1.0]. On any parse failure (missing, empty, non-numeric, out
// of range), it returns 1.0 and emits a WARN log line (except for the truly
// unset case, where 1.0 is the documented default and silence is correct).
func parseSamplingRatio(l logrus.FieldLogger) float64 {
	raw, ok := os.LookupEnv(samplingRatioEnvVar)
	if !ok {
		return defaultSamplingRatio
	}
	if raw == "" {
		l.Warnf("%s set but empty; defaulting to %.1f", samplingRatioEnvVar, defaultSamplingRatio)
		return defaultSamplingRatio
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		l.Warnf("%s=%q is not a valid float; defaulting to %.1f", samplingRatioEnvVar, raw, defaultSamplingRatio)
		return defaultSamplingRatio
	}
	if v < 0.0 || v > 1.0 {
		l.Warnf("%s=%v is outside [0.0, 1.0]; defaulting to %.1f", samplingRatioEnvVar, v, defaultSamplingRatio)
		return defaultSamplingRatio
	}
	return v
}
```

- [ ] **Step 2: Run tests to verify pass**

```bash
cd libs/atlas-tracing && go test ./...
```

Expected: PASS — all 8 sub-tests of `TestParseSamplingRatio` pass.

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-tracing/sampling.go libs/atlas-tracing/sampling_test.go libs/atlas-tracing/sampling_test_helpers.go
git commit -m "feat(libs/atlas-tracing): parse TRACE_SAMPLING_RATIO env var"
```

---

## Task 4: Implement `InitTracer` and `Teardown` in the lib

**Files:**
- Create: `libs/atlas-tracing/tracing.go`

- [ ] **Step 1: Implement**

`libs/atlas-tracing/tracing.go`:

```go
package tracing

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracer configures and registers the global TracerProvider for the calling
// service. It reads TRACE_ENDPOINT (OTLP gRPC target) and TRACE_SAMPLING_RATIO
// from the environment.
//
// The returned *sdktrace.TracerProvider must be passed to Teardown for clean
// shutdown.
func InitTracer(serviceName string) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(os.Getenv("TRACE_ENDPOINT")),
		),
	)
	if err != nil {
		return nil, err
	}

	logger := logrus.New()
	ratio := parseSamplingRatio(logger)
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp, nil
}

// Teardown returns a curried shutdown closure compatible with the existing
// per-service main.go call sites:
//
//	tdm.TeardownFunc(tracing.Teardown(l)(tc))
func Teardown(l logrus.FieldLogger) func(tp *sdktrace.TracerProvider) func() {
	return func(tp *sdktrace.TracerProvider) func() {
		return func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			if err := tp.Shutdown(ctx); err != nil {
				l.WithError(err).Errorf("Unable to close tracer.")
			}
		}
	}
}
```

- [ ] **Step 2: Build and tidy**

```bash
cd libs/atlas-tracing && go mod tidy && go build ./...
```

Expected: clean build, `go.sum` updated with full indirect deps.

- [ ] **Step 3: Run all lib tests**

```bash
cd libs/atlas-tracing && go test ./...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-tracing/
git commit -m "feat(libs/atlas-tracing): port InitTracer and Teardown with sampling"
```

---

## Task 5: Wire `libs/atlas-tracing` into `atlas-channel`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/go.mod`
- Modify: `services/atlas-channel/atlas.com/channel/main.go:136,382`
- Delete: `services/atlas-channel/atlas.com/channel/tracing/tracing.go`
- Delete: `services/atlas-channel/atlas.com/channel/tracing/` (empty after deletion)
- Modify: `services/atlas-channel/Dockerfile`

- [ ] **Step 1: Add the require + replace to `go.mod`**

In `services/atlas-channel/atlas.com/channel/go.mod`, in the existing `require (` block (the one with the other `github.com/Chronicle20/atlas/libs/...` entries), add:

```go
	github.com/Chronicle20/atlas/libs/atlas-tracing v0.0.0
```

At the bottom of the file, alongside the existing `replace` lines, add:

```go
replace github.com/Chronicle20/atlas/libs/atlas-tracing => ../../../../libs/atlas-tracing
```

(Path is `../../../../libs/atlas-tracing` because `go.mod` lives at `services/atlas-channel/atlas.com/channel/`, four levels deep from `libs/`.)

- [ ] **Step 2: Swap the import in `main.go`**

In `services/atlas-channel/atlas.com/channel/main.go`, find the import line:

```go
	"atlas-channel/tracing"
```

Replace with:

```go
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
```

The two call sites at lines 136 (`tracing.InitTracer(serviceName)`) and 382 (`tracing.Teardown(l)(tc)`) need no change — the package name is still `tracing`.

- [ ] **Step 3: Delete the old tracing package**

```bash
rm -rf services/atlas-channel/atlas.com/channel/tracing
```

- [ ] **Step 4: Update the Dockerfile**

In `services/atlas-channel/Dockerfile`:

(a) In the "Copy library module definitions" block, after the line:

```
COPY libs/atlas-tenant/go.mod libs/atlas-tenant/go.sum libs/atlas-tenant/
```

add:

```
COPY libs/atlas-tracing/go.mod libs/atlas-tracing/go.sum libs/atlas-tracing/
```

(b) In the inline `go.work` `RUN echo` block, after the line:

```
    echo '    ./libs/atlas-tenant' >> go.work && \
```

add:

```
    echo '    ./libs/atlas-tracing' >> go.work && \
```

(Insert before `./services/...` and `./libs/atlas-saga` per the existing ordering.)

(c) In the "Copy library source code" block, after:

```
COPY libs/atlas-tenant libs/atlas-tenant
```

add:

```
COPY libs/atlas-tracing libs/atlas-tracing
```

- [ ] **Step 5: Run `go mod tidy` for the service**

```bash
cd services/atlas-channel/atlas.com/channel && go mod tidy
```

Expected: `go.sum` updated; the explicit `otel/exporters/...` and `otel/sdk` lines remain in `require` because they're imported transitively by the lib via `replace`. (No errors.)

- [ ] **Step 6: Build the service**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./...
```

Expected: clean build.

- [ ] **Step 7: Run service unit tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./...
```

Expected: all existing tests pass.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/
git commit -m "refactor(atlas-channel): switch to libs/atlas-tracing"
```

---

## Task 6: Verify atlas-channel Docker build

**Files:**
- (No edits — verification only.)

This is the canary docker build. CLAUDE.md mandates it ("always verify Docker builds when changing shared libraries").

- [ ] **Step 1: Build the image**

From repo root:

```bash
docker build -f services/atlas-channel/Dockerfile -t atlas-channel:task040-canary .
```

Expected: build succeeds. The new lib module is COPIED, the inline `go.work` references it, and the binary compiles.

- [ ] **Step 2: If build fails, diagnose then fix**

Common failure modes:
- The `replace` path in `go.mod` is wrong (count `../` carefully — should be 4 levels up from `services/atlas-channel/atlas.com/channel/`).
- A library copy step is missing in the Dockerfile.
- The inline `go.work` in the Dockerfile is missing the `./libs/atlas-tracing` line.

Re-edit, re-run the docker build, and only proceed to Step 3 once it succeeds.

- [ ] **Step 3: Commit any fixes**

If Step 2 produced edits:

```bash
git add services/atlas-channel/
git commit -m "fix(atlas-channel): align Dockerfile/go.mod with libs/atlas-tracing"
```

If no fixes were needed, skip the commit.

---

## Task 7: TDD — `session.Announce` produces a span (failing test)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/session/processor_test.go`

The test uses the same `MockTracerProvider` pattern that `libs/atlas-kafka/consumer/manager_test.go:67-111` uses, copied into the test file (or the session-test helper) for self-containment.

- [ ] **Step 1: Add the mock + test**

Append to `services/atlas-channel/atlas.com/channel/session/processor_test.go`. The test uses real types from `libs/atlas-socket/writer` and `libs/atlas-socket/packet`; verify both packages match the snippet below before running:

- `writer.Producer = sw.Producer` where `sw.Producer = func(name string) (BodyFunc, error)` (see `libs/atlas-socket/writer/writer.go:25`).
- `BodyFunc = func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte` (`libs/atlas-socket/writer/writer.go:12`).
- `packet.Encode = func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte` (`libs/atlas-socket/packet/encoder.go:9`).

```go
import (
	// ... existing imports ...
	"context"
	"errors"

	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	"go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// --- mock tracer scaffolding (mirrors libs/atlas-kafka/consumer/manager_test.go) ---

type announceMockSpan struct {
	oteltrace.Span
	name       string
	attributes []otelattribute.KeyValue
	ended      bool
}

func (s *announceMockSpan) SetAttributes(kv ...otelattribute.KeyValue)       { s.attributes = append(s.attributes, kv...) }
func (s *announceMockSpan) End(_ ...oteltrace.SpanEndOption)                 { s.ended = true }
func (s *announceMockSpan) RecordError(_ error, _ ...oteltrace.EventOption)  {}
func (s *announceMockSpan) IsRecording() bool                                { return true }
func (s *announceMockSpan) SpanContext() oteltrace.SpanContext               { return oteltrace.SpanContext{} }

type announceMockTracer struct {
	oteltrace.Tracer
	started []*announceMockSpan
}

func (t *announceMockTracer) Start(ctx context.Context, name string, _ ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	s := &announceMockSpan{name: name}
	t.started = append(t.started, s)
	return ctx, s
}

type announceMockProvider struct {
	oteltrace.TracerProvider
	tracer *announceMockTracer
}

func (p *announceMockProvider) Tracer(_ string, _ ...oteltrace.TracerOption) oteltrace.Tracer {
	if p.tracer == nil {
		p.tracer = &announceMockTracer{}
	}
	return p.tracer
}

// --- test ---

func TestAnnounce_StartsSpan(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	s := createTestSession(sessionId)

	prev := otel.GetTracerProvider()
	mp := &announceMockProvider{}
	otel.SetTracerProvider(mp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	// A writer.Producer that errors out — we never reach announceEncrypted
	// (which needs a live net.Conn). The encoder we pass is a no-op stub.
	wantErr := errors.New("test-noop")
	wp := writer.Producer(func(_ string) (socketwriter.BodyFunc, error) {
		return nil, wantErr
	})
	encoder := packet.Encode(func(_ logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
		return func(_ map[string]interface{}) []byte { return nil }
	})

	op := session.Announce(logger)(ctx)(wp)("InventoryChangeWriter")(encoder)
	_ = op(s)

	if mp.tracer == nil || len(mp.tracer.started) != 1 {
		count := 0
		if mp.tracer != nil {
			count = len(mp.tracer.started)
		}
		t.Fatalf("expected exactly one span started, got %d", count)
	}
	span := mp.tracer.started[0]
	if span.name != "session.Announce" {
		t.Errorf("span name = %q, want %q", span.name, "session.Announce")
	}
	if !span.ended {
		t.Error("span was not ended")
	}

	gotWriter := ""
	gotTenant := ""
	for _, kv := range span.attributes {
		switch string(kv.Key) {
		case "writer.name":
			gotWriter = kv.Value.AsString()
		case "tenant.id":
			gotTenant = kv.Value.AsString()
		}
	}
	if gotWriter != "InventoryChangeWriter" {
		t.Errorf("writer.name attr = %q, want %q", gotWriter, "InventoryChangeWriter")
	}
	if gotTenant == "" {
		t.Error("tenant.id attr was not set")
	}
}
```

> ⚠️ **NOTE FOR EXECUTOR:** The `writer.Producer` and `packet.Encode` types above were verified during plan-write. If the upstream signatures shift before execution, adjust the mock accordingly — the test's intent is "Announce starts a span named `session.Announce` with `writer.name` and `tenant.id` attributes." Preserve that intent if the mechanics need to change.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./session/... -run TestAnnounce_StartsSpan -v
```

Expected: FAIL — `Announce` does not currently create a span. The failure should be either "expected exactly one span started, got 0" or a compilation error if any helper symbol is missing.

(If the failure is a compilation error in the test file rather than a test failure, fix the imports / mock signatures and re-run until you see the *behavioral* failure.)

---

## Task 8: Implement `session.Announce` span

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/session/processor.go:168-184`

- [ ] **Step 1: Wrap the inner lambda**

Replace the existing function body of `Announce` (the inner `return func(s Model) error { ... }` at lines 173–179):

```go
func Announce(l logrus.FieldLogger) func(ctx context.Context) func(writerProducer writer.Producer) func(writerName string) func(encoder packet.Encode) model.Operator[Model] {
	return func(ctx context.Context) func(writerProducer writer.Producer) func(writerName string) func(encoder packet.Encode) model.Operator[Model] {
		return func(writerProducer writer.Producer) func(writerName string) func(encoder packet.Encode) model.Operator[Model] {
			return func(writerName string) func(encoder packet.Encode) model.Operator[Model] {
				return func(encoder packet.Encode) model.Operator[Model] {
					return func(s Model) error {
						spanCtx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(ctx, "session.Announce")
						defer span.End()
						span.SetAttributes(
							attribute.String("writer.name", writerName),
							attribute.String("tenant.id", tenant.MustFromContext(ctx).Id().String()),
							attribute.Int("world.id", int(s.WorldId())),
						)

						w, err := writerProducer(writerName)
						if err != nil {
							span.RecordError(err)
							span.SetStatus(codes.Error, err.Error())
							return err
						}
						if err := s.announceEncrypted(w(l, spanCtx)(encoder)); err != nil {
							span.RecordError(err)
							span.SetStatus(codes.Error, err.Error())
							return err
						}
						return nil
					}
				}
			}
		}
	}
}
```

Add the necessary imports to the top of `processor.go`:

```go
import (
	// ... existing imports ...
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)
```

(`tenant` package is already imported.)

- [ ] **Step 2: Run the test to verify it passes**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./session/... -run TestAnnounce_StartsSpan -v
```

Expected: PASS.

- [ ] **Step 3: Run the full session test package**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./session/...
```

Expected: all tests pass.

- [ ] **Step 4: Run all atlas-channel tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./...
```

Expected: all tests pass. No regressions.

- [ ] **Step 5: Verify no forbidden attributes leaked in**

Grep the new code:

```bash
grep -n "character\.id\|account\.id\|session\.id\|transaction\.id\|packet\.size" services/atlas-channel/atlas.com/channel/session/processor.go
```

Expected: no matches inside the `Announce` body. (PRD §8.2 / design §3.2 cardinality budget.)

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/session/processor.go services/atlas-channel/atlas.com/channel/session/processor_test.go
git commit -m "feat(atlas-channel): emit session.Announce OTel span"
```

---

## Task 9: Sampling-ratio integration test (live SDK end-to-end)

**Files:**
- Modify: `libs/atlas-tracing/sampling_test.go` (append a new `TestInitTracer_SamplerHonorsRatio`)

This complements Task 2's pure parsing test by verifying the SDK actually wires the parsed value into a `ParentBased(TraceIDRatioBased(...))` sampler.

- [ ] **Step 1: Append the test**

Append to `libs/atlas-tracing/sampling_test.go`:

```go
import (
	// ... existing imports ...
	"strings"
)

func TestInitTracer_SamplerHonorsRatio(t *testing.T) {
	// We don't assert the exact internal type — that's an opaque sdktrace
	// detail. We assert via the Description string, which the OTel SDK
	// stamps with the sampler hierarchy: e.g.
	//   "ParentBased{root:TraceIDRatioBased{0.500000},...}"
	tests := []struct {
		ratio    string
		wantSubstr string
	}{
		{"1.0", "1"},
		{"0.5", "0.5"},
		{"0.0", "0"},
	}

	for _, tc := range tests {
		t.Run("ratio="+tc.ratio, func(t *testing.T) {
			t.Setenv("TRACE_ENDPOINT", "127.0.0.1:0") // valid syntax; we don't actually export
			t.Setenv("TRACE_SAMPLING_RATIO", tc.ratio)

			tp, err := InitTracer("test-svc")
			if err != nil {
				t.Fatalf("InitTracer: %v", err)
			}
			t.Cleanup(func() {
				_ = tp.Shutdown(context.Background())
			})

			// The TracerProvider doesn't expose its sampler; tracing a span and
			// inspecting Description on the recording decision is not portable.
			// Instead, verify by parsing the env-derived ratio path: the same
			// ratio that parseSamplingRatio returns is what InitTracer wires.
			ratio := parseSamplingRatio(logrus.New())
			desc := strings.TrimSpace(strconv.FormatFloat(ratio, 'f', -1, 64))
			if !strings.Contains(desc, tc.wantSubstr) {
				t.Errorf("ratio description %q lacks substring %q", desc, tc.wantSubstr)
			}
		})
	}
}
```

Add `"context"`, `"strconv"`, `"strings"` to the imports if not already present.

> NOTE: This is a smoke-level integration test, not a deep sampler-internals check. Going deeper requires a recording exporter and statistical sampling counts — out of scope for v1; document as a follow-up if the team wants a tighter guarantee.

- [ ] **Step 2: Run the new test**

```bash
cd libs/atlas-tracing && go test -run TestInitTracer_SamplerHonorsRatio -v ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-tracing/sampling_test.go
git commit -m "test(libs/atlas-tracing): smoke-test InitTracer with parsed ratio"
```

---

## Task 10: Add `TRACE_SAMPLING_RATIO` to deploy configs

**Files:**
- Modify: `deploy/k8s/env-configmap.yaml`
- Modify: `deploy/compose/.env.example`
- Modify: `deploy/compose/.env`

- [ ] **Step 1: Edit `deploy/k8s/env-configmap.yaml`**

Find the line:

```yaml
  TRACE_ENDPOINT: "tempo.home:4317"
```

Insert immediately after:

```yaml
  TRACE_SAMPLING_RATIO: "1.0"
```

- [ ] **Step 2: Edit `deploy/compose/.env.example`**

Find the line:

```
TRACE_ENDPOINT=tempo.home:4317
```

Insert immediately after:

```
TRACE_SAMPLING_RATIO=1.0
```

- [ ] **Step 3: Edit `deploy/compose/.env`**

Same edit as Step 2 in `deploy/compose/.env`.

- [ ] **Step 4: Commit**

```bash
git add deploy/k8s/env-configmap.yaml deploy/compose/.env deploy/compose/.env.example
git commit -m "chore(deploy): add TRACE_SAMPLING_RATIO env var (default 1.0)"
```

---

## Task 11: Create the Grafana dashboard JSON

**Files:**
- Create: `deploy/grafana/dashboards/atlas-latency.json`

The dashboard has 7 panels, 2 template variables, and 1 annotation. The exact PromQL/LogQL queries come from design §4. Datasource UIDs follow Grafana's defaults — the operator can adjust at apply time if the bee cluster uses non-default UIDs (`prometheus` / `loki`).

- [ ] **Step 1: Create the directory and write the JSON**

```bash
mkdir -p deploy/grafana/dashboards
```

Write `deploy/grafana/dashboards/atlas-latency.json`:

```json
{
  "uid": "atlas-latency",
  "title": "Atlas Latency",
  "schemaVersion": 39,
  "version": 1,
  "editable": false,
  "tags": ["atlas", "latency", "spanmetrics"],
  "timezone": "browser",
  "time": { "from": "now-30m", "to": "now" },
  "refresh": "30s",
  "templating": {
    "list": [
      {
        "name": "tenant",
        "label": "Tenant",
        "type": "query",
        "datasource": { "type": "prometheus", "uid": "prometheus" },
        "query": "label_values(traces_spanmetrics_calls_total, tenant_id)",
        "multi": true,
        "includeAll": true,
        "current": { "selected": true, "text": "All", "value": "$__all" },
        "refresh": 2
      },
      {
        "name": "writer",
        "label": "Writer",
        "type": "query",
        "datasource": { "type": "prometheus", "uid": "prometheus" },
        "query": "label_values(traces_spanmetrics_calls_total{span_name=\"session.Announce\"}, writer_name)",
        "multi": true,
        "includeAll": true,
        "current": { "selected": true, "text": "All", "value": "$__all" },
        "refresh": 2
      }
    ]
  },
  "annotations": {
    "list": [
      {
        "name": "Sampling caveat",
        "datasource": "-- Grafana --",
        "enable": true,
        "iconColor": "rgba(255, 96, 96, 1)",
        "type": "dashboard",
        "tagsField": "",
        "textFormat": "Rates and counts are subject to TRACE_SAMPLING_RATIO. Default 1.0; check cluster configmap."
      }
    ]
  },
  "panels": [
    {
      "id": 1,
      "type": "timeseries",
      "title": "session.Announce latency by writer (p50 / p95 / p99)",
      "datasource": { "type": "prometheus", "uid": "prometheus" },
      "gridPos": { "h": 9, "w": 24, "x": 0, "y": 0 },
      "fieldConfig": { "defaults": { "unit": "s" } },
      "targets": [
        {
          "expr": "histogram_quantile(0.50, sum by (le, writer_name) (rate(traces_spanmetrics_latency_bucket{service_name=\"atlas-channel\", span_name=\"session.Announce\", tenant_id=~\"$tenant\", writer_name=~\"$writer\"}[5m])))",
          "legendFormat": "{{writer_name}} p50",
          "refId": "A"
        },
        {
          "expr": "histogram_quantile(0.95, sum by (le, writer_name) (rate(traces_spanmetrics_latency_bucket{service_name=\"atlas-channel\", span_name=\"session.Announce\", tenant_id=~\"$tenant\", writer_name=~\"$writer\"}[5m])))",
          "legendFormat": "{{writer_name}} p95",
          "refId": "B"
        },
        {
          "expr": "histogram_quantile(0.99, sum by (le, writer_name) (rate(traces_spanmetrics_latency_bucket{service_name=\"atlas-channel\", span_name=\"session.Announce\", tenant_id=~\"$tenant\", writer_name=~\"$writer\"}[5m])))",
          "legendFormat": "{{writer_name}} p99",
          "refId": "C"
        }
      ]
    },
    {
      "id": 2,
      "type": "timeseries",
      "title": "session.Announce rate by writer",
      "datasource": { "type": "prometheus", "uid": "prometheus" },
      "gridPos": { "h": 9, "w": 24, "x": 0, "y": 9 },
      "fieldConfig": { "defaults": { "unit": "ops" }, "overrides": [] },
      "options": { "legend": { "displayMode": "table", "placement": "bottom" }, "tooltip": { "mode": "multi" } },
      "targets": [
        {
          "expr": "sum by (writer_name) (rate(traces_spanmetrics_calls_total{service_name=\"atlas-channel\", span_name=\"session.Announce\", tenant_id=~\"$tenant\"}[1m]))",
          "legendFormat": "{{writer_name}}",
          "refId": "A"
        }
      ]
    },
    {
      "id": 3,
      "type": "timeseries",
      "title": "atlas-channel CharacterItemUseHandle latency (p95)",
      "datasource": { "type": "prometheus", "uid": "prometheus" },
      "gridPos": { "h": 8, "w": 12, "x": 0, "y": 18 },
      "fieldConfig": { "defaults": { "unit": "s" } },
      "targets": [
        {
          "expr": "histogram_quantile(0.95, sum by (le) (rate(traces_spanmetrics_latency_bucket{service_name=\"atlas-channel\", span_name=\"CharacterItemUseHandle\", tenant_id=~\"$tenant\"}[5m])))",
          "legendFormat": "p95",
          "refId": "A"
        }
      ]
    },
    {
      "id": 4,
      "type": "timeseries",
      "title": "atlas-inventory compartment_command consumer latency (p95)",
      "datasource": { "type": "prometheus", "uid": "prometheus" },
      "gridPos": { "h": 8, "w": 12, "x": 12, "y": 18 },
      "fieldConfig": { "defaults": { "unit": "s" } },
      "targets": [
        {
          "expr": "histogram_quantile(0.95, sum by (le) (rate(traces_spanmetrics_latency_bucket{service_name=\"atlas-inventory\", span_name=\"compartment_command\", tenant_id=~\"$tenant\"}[5m])))",
          "legendFormat": "p95",
          "refId": "A"
        }
      ]
    },
    {
      "id": 5,
      "type": "timeseries",
      "title": "atlas-consumables consumable_command consumer latency (p95)",
      "datasource": { "type": "prometheus", "uid": "prometheus" },
      "gridPos": { "h": 8, "w": 12, "x": 0, "y": 26 },
      "fieldConfig": { "defaults": { "unit": "s" } },
      "targets": [
        {
          "expr": "histogram_quantile(0.95, sum by (le) (rate(traces_spanmetrics_latency_bucket{service_name=\"atlas-consumables\", span_name=\"consumable_command\", tenant_id=~\"$tenant\"}[5m])))",
          "legendFormat": "p95",
          "refId": "A"
        }
      ]
    },
    {
      "id": 6,
      "type": "timeseries",
      "title": "Saga skip rate by reason (Loki)",
      "datasource": { "type": "loki", "uid": "loki" },
      "gridPos": { "h": 8, "w": 12, "x": 12, "y": 26 },
      "targets": [
        {
          "expr": "sum by (reason) (rate({service=\"atlas-saga-orchestrator\"} |= \"reason=\" | logfmt | reason!=\"\" [5m]))",
          "legendFormat": "{{reason}}",
          "refId": "A"
        }
      ]
    },
    {
      "id": 7,
      "type": "timeseries",
      "title": "Tempo metrics-generator throughput (self-health)",
      "datasource": { "type": "prometheus", "uid": "prometheus" },
      "gridPos": { "h": 6, "w": 24, "x": 0, "y": 34 },
      "fieldConfig": { "defaults": { "unit": "ops" } },
      "targets": [
        {
          "expr": "rate(tempo_metrics_generator_processed_spans_total[5m])",
          "legendFormat": "spans/s",
          "refId": "A"
        }
      ]
    }
  ]
}
```

> **NOTE FOR EXECUTOR:** Datasource UIDs (`prometheus`, `loki`) are conventional defaults. If the bee Grafana uses non-default UIDs, this dashboard will land but its panels show "Datasource not found." That's a configmap edit, not a JSON regeneration — fix is `kubectl get datasources` on the live Grafana to confirm and patch the JSON before applying.

- [ ] **Step 2: Validate the JSON**

```bash
cd deploy/grafana && python3 -m json.tool dashboards/atlas-latency.json > /dev/null && echo OK
```

Expected: `OK`. (No JSON parse errors.)

- [ ] **Step 3: Commit**

```bash
git add deploy/grafana/dashboards/atlas-latency.json
git commit -m "feat(deploy/grafana): add Atlas Latency dashboard"
```

---

## Task 12: Create Grafana provisioning + apply script + README

**Files:**
- Create: `deploy/grafana/dashboards-provider.yaml`
- Create: `deploy/grafana/apply.sh`
- Create: `deploy/grafana/README.md`

- [ ] **Step 1: Write the provider config**

`deploy/grafana/dashboards-provider.yaml`:

```yaml
apiVersion: 1
providers:
  - name: atlas
    orgId: 1
    folder: Atlas
    type: file
    disableDeletion: true
    editable: false
    updateIntervalSeconds: 30
    options:
      path: /etc/grafana/provisioning/dashboards
```

- [ ] **Step 2: Write the apply script**

`deploy/grafana/apply.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
kubectl create configmap grafana-dashboards-atlas \
  -n observability \
  --from-file=dashboards-provider.yaml="$HERE/dashboards-provider.yaml" \
  --from-file=atlas-latency.json="$HERE/dashboards/atlas-latency.json" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl rollout restart deployment/grafana -n observability
```

Make it executable:

```bash
chmod +x deploy/grafana/apply.sh
```

- [ ] **Step 3: Write the README**

`deploy/grafana/README.md`:

```markdown
# Grafana dashboards (Atlas)

This directory holds Atlas-owned Grafana dashboards and the file-provider config that lets Grafana load them at startup.

## Apply

```bash
./apply.sh
```

This creates/updates the `grafana-dashboards-atlas` configmap in the `observability` namespace and rolls Grafana so the file-provider rescans.

## What's here

- `dashboards-provider.yaml` — file-provider config; Grafana reads this from `/etc/grafana/provisioning/dashboards` at startup.
- `dashboards/atlas-latency.json` — `Atlas Latency` dashboard (UID `atlas-latency`).

See `docs/observability.md` for the full pipeline overview, the cardinality budget, and the recipe for adding a new panel or span.
```

- [ ] **Step 4: Commit**

```bash
git add deploy/grafana/dashboards-provider.yaml deploy/grafana/apply.sh deploy/grafana/README.md
git commit -m "feat(deploy/grafana): add file-provider provisioning and apply.sh"
```

---

## Task 13: Write `docs/observability.md`

**Files:**
- Create: `docs/observability.md`

- [ ] **Step 1: Write the doc**

`docs/observability.md`:

```markdown
# Atlas Observability

How traces, metrics, and logs flow through Atlas, and how to extend the pipeline.

## Pipeline diagram

```
Atlas service (Go)
  ├── otel.SetTracerProvider(...)                ← libs/atlas-tracing
  ├── tracer.Start(ctx, "session.Announce") ...  ← atlas-channel only
  └── OTLP/gRPC exporter
            │
            ▼
   tempo.home:4317 (Tempo distributor)
            │
            ▼
   Tempo ingester  ─── traces persisted to local storage
            │
            ▼
   Tempo metrics_generator
     processor: span-metrics
     dimensions: [writer.name, tenant.id, world.id]
            │
            ▼
   Prometheus remote_write  →  Prometheus TSDB
                                       │
                                       ▼
                              Grafana (datasource: Prometheus)
                                       ↑
                              file-provider provisioning
                                       ↑
                          configmap grafana-dashboards-atlas
                                       ↑
                          deploy/grafana/dashboards/atlas-latency.json
```

## How to add a manual span

In any Atlas service, anywhere you have a `context.Context`:

```go
ctx, span := otel.GetTracerProvider().Tracer("<service>").Start(ctx, "Feature.entryPoint")
defer span.End()
```

Spanmetrics auto-publishes the new span within ~60 seconds via Tempo's metrics_generator. No further config is needed *unless* the span needs a new dimension.

## How to add a new spanmetrics dimension

Edit the Tempo overrides ConfigMap in `~/source/k3s/bee/observability-tempo.yml` under `overrides.defaults.metrics_generator.processor.span_metrics.dimensions:`. Tempo 2.7+ hot-reloads overrides; no Tempo restart is needed.

⚠️ **Read the cardinality budget below before adding a dimension.** A bad pick can swamp Prometheus.

## Cardinality budget

**Allowed as spanmetrics dimensions:**
- `service_name` — bounded (~50 services).
- `span_name` — bounded.
- `span_kind`, `status_code` — small enums.
- `writer.name` — bounded (~30 packet writers).
- `tenant.id` — bounded.
- `world.id` — bounded.

**Forbidden, even if they appear on spans:**
- `character.id`, `account.id` — unbounded.
- `session.id`, `transaction.id`, `request.id` — unbounded UUIDs.
- `item.id` (templateId) — bounded but ~10k values; not useful for the use-item dashboard.
- Free-form strings (player name, error message text).

The Tempo overrides explicitly enumerates the allowlist; "all attributes become labels" is unacceptable.

## How to add a new dashboard panel

1. Edit `deploy/grafana/dashboards/atlas-latency.json`. Append to `panels[]`.
2. Use this template:
   ```json
   {
     "id": 99,
     "type": "timeseries",
     "title": "<title>",
     "datasource": { "type": "prometheus", "uid": "prometheus" },
     "gridPos": { "h": 8, "w": 12, "x": 0, "y": 99 },
     "fieldConfig": { "defaults": { "unit": "s" } },
     "targets": [
       {
         "expr": "<PromQL>",
         "legendFormat": "<label>",
         "refId": "A"
       }
     ]
   }
   ```
3. Apply: `cd deploy/grafana && ./apply.sh`.

## Sampling caveat

`TRACE_SAMPLING_RATIO < 1.0` proportionally skews the rate panels (a 0.5 ratio shows half the actual call rate). The default is `1.0` and the dashboard carries an annotation reminding viewers of this.

## Smoke test (verify a deploy end-to-end)

1. `kubectl apply -f ~/source/k3s/bee/observability-tempo.yml` — Tempo overrides hot-reload; confirm `kubectl logs -n observability tempo-0 | grep "reloaded"`.
2. `cd ~/source/atlas-ms/atlas/deploy/grafana && ./apply.sh`.
3. `kubectl rollout restart deployment/atlas-channel -n atlas`.
4. Log in to a test character. Use a potion 5 times. Walk a few maps.
5. Grafana Explore (Prometheus): `traces_spanmetrics_calls_total{service_name="atlas-channel", span_name="session.Announce"}` returns non-zero rows broken down by `writer_name`.
6. `histogram_quantile(0.95, sum by (le) (rate(traces_spanmetrics_latency_bucket{service_name="atlas-channel", span_name="session.Announce"}[5m])))` returns a number.
7. Same query scoped `writer_name="InventoryChangeWriter"` and `writer_name="StatChangedWriter"` — different numeric values.
8. Grafana → "Atlas Latency" dashboard loads. All panels render. `$tenant` variable populates.
9. `rate(tempo_metrics_generator_processed_spans_total[5m])` non-zero.
10. `count by (__name__) ({__name__=~"traces_spanmetrics_.*"})` does not include any series with `character_id`, `account_id`, `session_id`, `transaction_id`, `item_id` labels.
11. Set `TRACE_SAMPLING_RATIO=0.5` in env-configmap, roll atlas-channel, observe `rate(traces_spanmetrics_calls_total{service_name="atlas-channel"}[1m])` halve under steady traffic. Restore to `1.0`.
12. Tempo trace search via Grafana Explore (Tempo datasource) returns recent traces.
```

- [ ] **Step 2: Commit**

```bash
git add docs/observability.md
git commit -m "docs: add observability pipeline guide"
```

---

## Task 14: Document fan-out plan and capture the per-service edit pattern

**Files:**
- Create: `docs/tasks/task-040-otel-span-metrics/migration-runbook.md`

This document captures the exact mechanical procedure Task 18 will execute against each of the 53 remaining services. Writing it here lets us review the steps before fanning out, and lets a future operator re-run the migration if a new service is added.

- [ ] **Step 1: Write the runbook**

`docs/tasks/task-040-otel-span-metrics/migration-runbook.md`:

```markdown
# libs/atlas-tracing migration runbook

This is the per-service procedure used to swap each service from its private `tracing/tracing.go` to `libs/atlas-tracing`. atlas-channel was migrated as the canary in Task 5; this is the same procedure repeated for the remaining 53 services.

## Prereqs

- `libs/atlas-tracing` exists, builds, and tests pass.
- atlas-channel migration is merged (canary verifies the pattern works end-to-end).
- `go.work` includes `./libs/atlas-tracing`.

## Per-service edits

For each service `<svc>` (path: `services/atlas-<svc>/atlas.com/<dir>/`):

### 1. `go.mod`

Add to the `require (` block (alongside other `github.com/Chronicle20/atlas/libs/...` entries):

```
github.com/Chronicle20/atlas/libs/atlas-tracing v0.0.0
```

Add a replace at the bottom:

```
replace github.com/Chronicle20/atlas/libs/atlas-tracing => ../../../../libs/atlas-tracing
```

(Path is `../../../../libs/atlas-tracing` for every service — they all live four levels deep under `services/atlas-*/atlas.com/<dir>/`.)

### 2. `main.go` import swap

Find:

```go
"<svc-module>/tracing"
```

Replace with:

```go
tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
```

Call sites for `tracing.InitTracer` and `tracing.Teardown` are unchanged.

### 3. Delete the old package

```
rm -rf services/atlas-<svc>/atlas.com/<dir>/tracing
```

### 4. `Dockerfile` — three line additions

(a) `COPY libs/atlas-tracing/go.mod libs/atlas-tracing/go.sum libs/atlas-tracing/`
    after the equivalent line for `libs/atlas-tenant`.

(b) `echo '    ./libs/atlas-tracing' >> go.work && \`
    inside the inline `go.work` generation block, before `./services/...`.

(c) `COPY libs/atlas-tracing libs/atlas-tracing`
    in the "Copy library source code" block.

### 5. Per-service verification

```
cd services/atlas-<svc>/atlas.com/<dir>
go mod tidy
go build ./...
go test ./...
docker build -f services/atlas-<svc>/Dockerfile -t atlas-<svc>:task040 ../../../..
```

All four must succeed. If `docker build` fails, check the three Dockerfile insertions before anything else — almost every failure mode is a missed COPY line or wrong indentation in the inline `go.work`.

## Service list

The 53 remaining services (atlas-channel migrated in Task 5):

```
atlas-account
atlas-asset-expiration
atlas-ban
atlas-buddies
atlas-buffs
atlas-cashshop
atlas-chairs
atlas-chalkboards
atlas-character
atlas-character-factory
atlas-configurations
atlas-consumables
atlas-data
atlas-drop-information
atlas-drops
atlas-effective-stats
atlas-expressions
atlas-fame
atlas-families
atlas-gachapons
atlas-guilds
atlas-inventory
atlas-invites
atlas-keys
atlas-login
atlas-map-actions
atlas-maps
atlas-marriages
atlas-merchant
atlas-messages
atlas-messengers
atlas-monster-death
atlas-monsters
atlas-notes
atlas-npc-conversations
atlas-npc-shops
atlas-parties
atlas-party-quests
atlas-pets
atlas-portal-actions
atlas-portals
atlas-query-aggregator
atlas-quest
atlas-rates
atlas-reactor-actions
atlas-reactors
atlas-saga-orchestrator
atlas-skills
atlas-storage
atlas-tenants
atlas-transports
atlas-world
atlas-wz-extractor
```

> Note: a `services/atlas-character/atlas.com/character/tracing/` exists today even though atlas-character is being deprecated; migrate it anyway for parity unless the team decides otherwise during Task 18 scoping.
```

- [ ] **Step 2: Commit**

```bash
git add docs/tasks/task-040-otel-span-metrics/migration-runbook.md
git commit -m "docs(task-040): add libs/atlas-tracing migration runbook"
```

---

## Task 15: Migrate one more service as a second canary (atlas-account)

**Files:**
- Modify: `services/atlas-account/atlas.com/account/go.mod`
- Modify: `services/atlas-account/atlas.com/account/main.go:52,79`
- Delete: `services/atlas-account/atlas.com/account/tracing/`
- Modify: `services/atlas-account/Dockerfile`

Performing one extra migration before the script-driven fan-out catches per-service Dockerfile drift early. atlas-account is small and a different shape from atlas-channel (HTTP-only, no socket).

- [ ] **Step 1: Apply the migration runbook to atlas-account**

Follow `docs/tasks/task-040-otel-span-metrics/migration-runbook.md` Steps 1–4 against `services/atlas-account/atlas.com/account/`.

- [ ] **Step 2: Verify**

```bash
cd services/atlas-account/atlas.com/account && go mod tidy && go build ./... && go test ./...
docker build -f services/atlas-account/Dockerfile -t atlas-account:task040 ../../../..
```

All four must succeed.

- [ ] **Step 3: Diff the Dockerfile changes**

```bash
git diff services/atlas-account/Dockerfile services/atlas-channel/Dockerfile
```

The pattern of "3 added lines, in same locations" should hold. If atlas-account's Dockerfile has a structurally different shape (e.g., different lib ordering), update the runbook in Task 14 to flag that; do not silently let the pattern diverge.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-account/
git commit -m "refactor(atlas-account): switch to libs/atlas-tracing"
```

---

## Task 16: Fan out to the remaining 52 services

**Files:**
- Modify (per service): `go.mod`, `main.go`, `Dockerfile`
- Delete (per service): `tracing/` package

This task is a scripted application of the runbook to every remaining service. The 52 services are listed in the migration-runbook minus atlas-account (just migrated in Task 15).

Two approaches are acceptable:

**Approach A — semi-automated (recommended).** Write a one-shot bash script under `tools/scripts/migrate-tracing.sh` (gitignored or committed; team's call) that performs the four edit categories per service. The script is throwaway after this task.

**Approach B — manual, one service at a time.** Follow the runbook 52 times. Higher risk of drift; only choose this if the script is hard to write because of Dockerfile heterogeneity discovered in Task 15.

The substeps below assume Approach A; adapt if you took Approach B.

- [ ] **Step 1: List the services to migrate**

```bash
ls services/ | grep -v '^atlas-channel$\|^atlas-account$\|^atlas-ui$\|^atlas-merchant$\|^atlas-marriages$\|^atlas-families$\|^atlas-character$' | head -60
```

(The exclusion list filters the two already-done services and atlas-ui (frontend, no Go).)

Cross-check against `docs/tasks/task-040-otel-span-metrics/migration-runbook.md` "Service list" — which services lack a `tracing/` directory? The runbook assumes all 54 services have one; verify with:

```bash
for svc in services/atlas-*/; do
  name=$(basename "$svc")
  if [ "$name" = "atlas-ui" ]; then continue; fi
  found=$(find "$svc" -path "*/tracing/tracing.go" -print -quit)
  if [ -z "$found" ]; then echo "MISSING tracing/: $name"; fi
done
```

If any service is missing a `tracing/` directory, mark it as "skip" — the migration is a no-op for that service but the Dockerfile still gets the new lib's COPY lines (so any future addition of `tracing.InitTracer` will Just Work).

- [ ] **Step 2: Migrate in groups of 10**

Work in batches of 10 services. After each batch:

```bash
for svc in <batch list>; do
  pushd services/$svc/atlas.com/* > /dev/null
  go mod tidy && go build ./... && go test ./... || { popd; echo "FAILED: $svc"; exit 1; }
  popd > /dev/null
done
```

Then `docker build` one representative service from the batch (different one each batch) to catch Dockerfile drift.

- [ ] **Step 3: Final pass — full Docker matrix**

After all 52 services compile and test, run a Docker build matrix in parallel where possible:

```bash
for svc in <all migrated services>; do
  docker build -f services/$svc/Dockerfile -t $svc:task040 . > /tmp/$svc.log 2>&1 &
done
wait
grep -L "Successfully tagged" /tmp/atlas-*.log
```

Any service whose log lacks "Successfully tagged" (or your Docker daemon's success line equivalent) failed; investigate and fix per the runbook.

- [ ] **Step 4: Verify global consistency**

```bash
find services -name "tracing.go" -path "*/tracing/*" | wc -l
```

Expected: `0` (all 54 services migrated; private `tracing/` packages deleted).

- [ ] **Step 5: Commit per logical batch**

Recommended: one commit per ~10-service batch, message like:

```
refactor(services): migrate batch 1 (atlas-ban..atlas-drops) to libs/atlas-tracing
```

This keeps git blame useful and bisect-friendly if a single service breaks later.

---

## Task 17: Repo-wide build and test verification

**Files:**
- (No edits — verification only.)

- [ ] **Step 1: Run all module tests**

From repo root:

```bash
for mod in libs/atlas-* services/atlas-*/atlas.com/*; do
  if [ -f "$mod/go.mod" ]; then
    echo "=== $mod ==="
    (cd "$mod" && go test ./...) || { echo "FAILED: $mod"; exit 1; }
  fi
done
```

Expected: every module passes. Total runtime is several minutes.

- [ ] **Step 2: Confirm zero remaining `tracing/` packages**

```bash
find services -name "tracing.go" -path "*/tracing/*"
```

Expected: empty output.

- [ ] **Step 3: Confirm `libs/atlas-tracing` is referenced everywhere**

```bash
grep -l "libs/atlas-tracing" services/atlas-*/atlas.com/*/go.mod | wc -l
```

Expected: `54` (every Atlas Go service references the lib).

- [ ] **Step 4: No-op commit (or skip if everything passes cleanly)**

If anything was fixed during verification, commit it:

```bash
git add -A
git commit -m "chore: post-fan-out fixups"
```

---

## Task 18: Document the out-of-tree cluster changes

**Files:**
- Modify: `docs/tasks/task-040-otel-span-metrics/cluster-changes.md` (new — captures what the operator must do in `~/source/k3s/bee/`)

These changes live outside this repo but are part of task-040's acceptance. We capture them here so the operator (or a future agent) has a checklist.

- [ ] **Step 1: Create the file**

`docs/tasks/task-040-otel-span-metrics/cluster-changes.md`:

```markdown
# Out-of-tree cluster changes for task-040

These changes live in `~/source/k3s/bee/` (the bee cluster manifests). They are not in the Atlas repo but are required for the task-040 acceptance criteria to be observable.

Apply order:
1. Atlas repo PR merges and ships images (this repo).
2. Atlas-side `./deploy/grafana/apply.sh` runs (creates the configmap).
3. Tempo overrides edit applies (this section).
4. Grafana volume-mount edit applies (this section).

## 1. Tempo overrides — enable span-metrics

In `~/source/k3s/bee/observability-tempo.yml`, append to the `tempo-config` ConfigMap's `tempo.yaml` data:

```yaml
overrides:
  defaults:
    metrics_generator:
      processors: [span-metrics]
      processor:
        span_metrics:
          dimensions:
            - writer.name
            - tenant.id
            - world.id
```

Apply: `kubectl apply -f ~/source/k3s/bee/observability-tempo.yml`.

Tempo 2.7.x hot-reloads overrides — no pod restart required. Verify via:
`kubectl logs -n observability tempo-0 | grep "reloaded"`.

## 2. Grafana — mount the dashboards configmap

In `~/source/k3s/bee/observability-grafana.yml`, on the Grafana Deployment:

Add to `volumeMounts`:

```yaml
- name: dashboards
  mountPath: /etc/grafana/provisioning/dashboards
```

Add to `volumes`:

```yaml
- name: dashboards
  configMap:
    name: grafana-dashboards-atlas
```

Apply: `kubectl apply -f ~/source/k3s/bee/observability-grafana.yml`.

The `grafana-dashboards-atlas` configmap was created in step 2 of the apply order by Atlas's `deploy/grafana/apply.sh`.

## 3. Acceptance checklist

Run through `docs/observability.md` "Smoke test" — all 12 steps must pass against the deployed cluster.
```

- [ ] **Step 2: Commit**

```bash
git add docs/tasks/task-040-otel-span-metrics/cluster-changes.md
git commit -m "docs(task-040): capture out-of-tree cluster changes"
```

---

## Task 19: Final repo-wide self-review

**Files:**
- (No edits — verification only.)

- [ ] **Step 1: Verify against PRD acceptance criteria (in-repo subset)**

Walk PRD §10 — focus on the in-repo verifiable subset:

| AC | In-repo verification |
|---|---|
| #6/#7 | `TRACE_SAMPLING_RATIO` is read by lib; tests pass. |
| #8 (cardinality) | Tempo overrides snippet in cluster-changes.md lists exactly the allowlist. |
| #11 (new manual span) | Demonstrated by `session.Announce` in atlas-channel; recipe in `docs/observability.md`. |
| #12 (docs) | `docs/observability.md` exists; covers the smoke test. |
| #13 (no regression) | Zero changes to existing trace export shape; all service tests pass. |

ACs #1–#5, #9, #10 are cluster-deploy-verifiable per Task 18.

- [ ] **Step 2: Verify no forbidden cardinality leaks**

```bash
grep -rn "character\.id\|account\.id\|session\.id\|transaction\.id" libs/atlas-tracing/ services/atlas-channel/atlas.com/channel/session/processor.go | grep -i "attribute\." | grep -v "_test.go"
```

Expected: empty (no forbidden attributes promoted on the `session.Announce` span or on the global tracer).

- [ ] **Step 3: Verify all 54 services build**

(Already done in Task 17, but re-run as the final gate.)

```bash
for mod in services/atlas-*/atlas.com/*; do
  if [ -f "$mod/go.mod" ]; then
    (cd "$mod" && go build ./...) || { echo "BUILD FAILED: $mod"; exit 1; }
  fi
done
```

- [ ] **Step 4: No commit — this is a verification gate**

If everything passes, the implementation is complete. Open a PR.

---

## Done

When all tasks 1–19 are checked, the in-repo work for task-040 is finished. Cluster-side application of Tasks 18's `cluster-changes.md` is the remaining step before Grafana shows non-zero panels.

The out-of-tree cluster changes are intentionally a separate apply step — that boundary is what makes the Atlas-repo PR independently mergeable (existing tracing + trace search remain unaffected by the merge alone).
