# OTel Span Metrics + Client-Write Latency Instrumentation — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-30
---

## 1. Overview

Atlas ships traces from each Go service directly to Tempo (`TRACE_ENDPOINT=tempo.home:4317` via OTLP gRPC), but it does not currently emit span-derived histograms to Prometheus. As a result, there is no continuous p50/p95/p99 signal for any operation. Every latency question is answered today by a one-off Loki-and-arithmetic dive (see task-038's verification of the use-item flow on 2026-04-30 — three traces, manual timestamp subtraction, ~5–15 ms of guessing on uninstrumented hops).

A second observability gap compounds this: the actual moment Atlas writes a packet to a connected game client is **completely unmeasured**. `session.Announce` (`services/atlas-channel/atlas.com/channel/session/processor.go:168`) is the single chokepoint through which every outbound client packet flows, and it produces no log line, no metric, and no span on success. The user-perceived latency of "I clicked the use-potion key, and my HP bar updated" is therefore knowable only by chaining together log timestamps from atlas-inventory and atlas-character and adding a fudge factor for the un-traced Kafka hop back into atlas-channel.

This task closes both gaps. It (1) turns on a span-derived metric pipeline so `traces_spanmetrics_*` series land in Prometheus and Grafana can show p50/p95/p99 over time for any instrumented span, and (2) adds a single manual OTel span around `session.Announce` with `writerName` as an attribute, making per-packet-type client-write latency a measurable quantity. A single Grafana dashboard panels the result so the use-item flow — and any future flow we care about — is observable without a Loki-and-arithmetic ritual.

## 2. Goals

Primary goals:

- **Continuous span-derived metrics in Prometheus.** `traces_spanmetrics_calls_total`, `traces_spanmetrics_latency_bucket` (or equivalents emitted by whichever pathway the design phase selects) are scraped and queryable for every span produced by Atlas services. `histogram_quantile(0.5, sum by (le) (rate(traces_spanmetrics_latency_bucket{service_name="atlas-channel"}[5m])))` returns a number.
- **`session.Announce` is a measurable span.** Every successful call produces an OTel span with at minimum the attributes `writer.name`, `tenant.id`, `world.id`. Spanmetrics for this span are queryable in Prometheus broken down by `writer.name` (bounded — ~30 packet types).
- **Use-item latency dashboard.** A Grafana dashboard exists, lives in source control under `deploy/`, and panels at minimum: (a) p50/p95/p99 of `session.Announce` latency by `writer.name`, (b) p50/p95/p99 of the `RequestItemConsume` HTTP/Kafka entry handler in atlas-channel, (c) p50/p95/p99 of the inventory reserve and consume command handlers in atlas-inventory.
- **The dashboard is generic enough to extend.** Adding "investigate ranged-attack latency" or "investigate map-warp latency" tomorrow is a panel addition, not a dashboard rewrite.
- **Zero regression in existing trace volume or sampling.** The signal we already have in Tempo is unaffected.

Non-goals:

- Tracing-driven alerting / SLO definitions (separate, follow-up task once we *have* the metrics to define an SLO against).
- Client-side timing (we instrument server-only; the client is external).
- Span instrumentation of every business-logic function in atlas-channel — only `session.Announce`. Auto-instrumentation (otelhttp / Kafka middlewares) covers the rest of the use-item flow well enough.
- Manual span instrumentation of atlas-consumables / atlas-inventory handlers. Auto-instrumentation already produces spans for their REST entry points and Kafka consumers; manual spans there would mostly add custom attributes for searchability, not new measurements. Deferred to a follow-up task if the dashboard reveals a missing slice.
- Re-architecting the OTel pipeline beyond what's needed to emit spanmetrics.
- Cardinality-explosion attributes: `character.id`, `transaction.id`, `session.id` are NOT span attributes promoted to metric dimensions. (They may live on the *span itself* for trace search, but `metrics_generator` / spanmetrics dimensions stays curated.)

## 3. User Stories

- As an Atlas developer asking "is the use-item flow getting slower over time?", I want to open a Grafana dashboard and see a 7-day p50/p95/p99 line chart of `session.Announce` for `InventoryChangeWriter` and `StatChangedWriter`, so that I don't have to repeat the manual Loki dive each time.
- As an Atlas developer who just shipped a latency-touching change (e.g., task-038), I want to compare the dashboard before-and-after deploy so that I can verify the change actually moved the needle in production, not just in local benchmarks.
- As an operator triaging a "the game feels laggy" report, I want to see at a glance which packet writers are slow and which aren't, so I can localize the problem without `kubectl exec` and grep.
- As an Atlas developer adding a new feature with measurable latency expectations, I want to define a one-line `tracer.Start(ctx, "FeatureName.entryPoint")` and have spanmetrics auto-publish it, so adding observability doesn't require new dashboard plumbing per feature.
- As an operator concerned about Tempo storage / Prometheus scrape budget, I want a `TRACE_SAMPLING_RATIO` env var (default 1.0) so I can throttle trace volume without code changes if the dev cluster gets noisy.

## 4. Functional Requirements

### 4.1 Span-derived metrics pipeline

The system SHALL emit per-span call counts and latency histograms to Prometheus for every span produced by services in the `atlas` namespace. Two viable architectures, to be picked in `/design-task`:

- **(a) Tempo `metrics_generator` with `span-metrics` processor.** Tempo already has `metrics_generator.registry` and `metrics_generator.storage.remote_write` configured (see `<infra-repo>/observability-tempo.yml`). Enabling span-metrics requires an `overrides` block with `defaults.metrics_generator.processors: [span-metrics]` (and optionally `service-graphs`) plus a `metrics_generator.processor.span_metrics.dimensions` list to promote `writer.name`, `tenant.id`, `world.id` into metric labels. **Pros:** ~10 lines of YAML, no new component, no service-side changes. **Cons:** Tempo-coupled; if we ever leave Tempo, the metrics path goes with it.
- **(b) OTel collector / Grafana Alloy with `spanmetrics` connector.** Either deploy a new OpenTelemetry Collector or extend the existing Alloy DaemonSet (`<infra-repo>/observability-alloy.yml`, currently log-collection-only) with an OTLP receiver, the `spanmetrics` connector, an OTLP exporter to Tempo, and a Prometheus remote-write exporter. Atlas service `TRACE_ENDPOINT` flips from `tempo.home:4317` to the collector's OTLP endpoint. **Pros:** vendor-neutral, future-proof, pipeline-flexible (filtering, redaction, etc.). **Cons:** new deployable component, every Atlas service env-var changes, more moving pieces to operate.

Either pathway SHALL produce metric series with at minimum these labels (using OTel-collector spanmetrics naming as the canonical reference; Tempo's labels differ slightly but cover the same dimensions):
- `service_name` — Atlas service emitting the span.
- `span_name` — name passed to `tracer.Start`.
- `span_kind` — server / client / internal / consumer / producer.
- `status_code` — OK / ERROR / UNSET.
- For `session.Announce` spans specifically: `writer_name`, `tenant_id`, `world_id` (see §4.2).

### 4.2 `session.Announce` span

In `services/atlas-channel/atlas.com/channel/session/processor.go`, the function `Announce` (currently at line 168) wraps a call to `s.announceEncrypted(w(l, ctx)(encoder))`. The implementation SHALL be modified so that the actual write is wrapped in:

```go
ctx, span := otel.Tracer("atlas-channel/session").Start(ctx, "session.Announce")
defer span.End()
span.SetAttributes(
    attribute.String("writer.name", writerName),
    attribute.String("tenant.id", tenantID),
    attribute.Int("world.id", int(worldID)),
)
// ... existing announceEncrypted call ...
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

The exact attribute extraction (where `tenantID` and `worldID` come from in the call context) is a design detail. `writerName` is already an explicit parameter at this call site. The span SHALL be created at the start of the actual write attempt — i.e., inside the innermost lambda at line 173–179 — not at the outer curried-function invocation.

The span SHALL NOT include packet payload bytes, character ID, account ID, or session UUID as attributes. (They may be added later via Tempo trace search if needed; they MUST NOT become spanmetrics labels — see §4.1 and §8.2.)

The span name `session.Announce` SHALL be stable. `writer.name` is the variable axis.

### 4.3 Tracer initialization unchanged

`services/atlas-channel/atlas.com/channel/tracing/tracing.go` already initializes a global TracerProvider via `tracing.InitTracer(serviceName)` and registers it via `otel.SetTracerProvider`. No change is required to that file. Subsequent `otel.Tracer("...")` calls in business code use the global provider transparently.

### 4.4 Configurable sampling ratio

The trace SDK SHALL support a `TRACE_SAMPLING_RATIO` env var, parsed as a float in `[0.0, 1.0]`, default `1.0`. When set, the SDK uses `trace.ParentBased(trace.TraceIDRatioBased(ratio))` so that decisions propagate through child spans. This allows operators to dial down trace volume without redeploys.

The env var SHALL be read in `tracing.InitTracer` (or a sibling helper). When unset or unparseable, the default is `1.0` (always sample) and a warning log is emitted.

The env var SHALL be added to `deploy/k8s/env-configmap.yaml` and `deploy/compose/.env.example` with default value `"1.0"`.

### 4.5 Grafana dashboard

A Grafana dashboard JSON file SHALL be checked into the repo at `deploy/grafana/dashboards/atlas-latency.json` (path subject to design-phase decision). The dashboard SHALL be **generic** — designed to be the home for *any* Atlas latency investigation, not just use-item — but its initial panels SHALL cover the use-item flow specifically:

1. **Outbound packet latency by writer** — multi-line chart, p50/p95/p99 of `session.Announce` latency, broken down by `writer_name`. Time range selector at top.
2. **`session.Announce` call rate by writer** — stacked area, `rate(traces_spanmetrics_calls_total{span_name="session.Announce"}[1m])` by `writer_name`. Lets us spot when a writer goes silent.
3. **Use-item entry-handler latency** — p50/p95/p99 of the atlas-channel handler that consumes the client's `CharacterItemUseHandle` packet (auto-instrumented span name TBD; design phase to identify).
4. **Inventory reserve latency** — p50/p95/p99 of atlas-inventory's `RequestReserve` handler span.
5. **Inventory consume latency** — p50/p95/p99 of atlas-inventory's `ConsumeAsset` handler span.
6. **Saga-orchestrator skip rate by reason** — `rate(... )` of saga skip events grouped by `reason`. Verifies tasks-038's nil-UUID guard continues to fire correctly and lets us spot if `saga_not_found` regressed.

Each panel SHALL include the PromQL query in its definition for easy copy-paste into ad-hoc Explore queries.

The dashboard SHALL be provisioned to Grafana via the same mechanism the rest of the clusterobservability stack uses (likely a Grafana sidecar configmap; design phase to confirm). If the cluster deployment does not yet have a dashboards-as-code mechanism, this task SHALL add one — minimally, a configmap mounted into the Grafana pod with the file-provider provisioning datasource pointing at it.

### 4.6 Documentation

A new doc at `docs/observability.md` SHALL describe:
- How the trace pipeline works end-to-end (services → OTLP → Tempo / collector → metrics_generator / spanmetrics → Prometheus → Grafana).
- How to add a new manual span (one paragraph + one code snippet).
- How to add a new dimension to spanmetrics (one paragraph; will be a Tempo override or collector connector config change depending on §4.1 outcome).
- How to extend the dashboard (a panel JSON template).
- The cardinality budget (§8.2) and which attributes are forbidden as spanmetrics dimensions.

## 5. API Surface

No HTTP / JSON:API surface changes. No Kafka topic or message-shape changes. The only "API" is the OTLP gRPC stream Atlas services already produce.

## 6. Data Model

No data-model changes. No DB migrations. No new persistent state in Atlas services. The metrics-generator-side state lives in Tempo's `metrics_generator.storage.path` (already configured at `/var/tempo/generator/wal`) and in Prometheus's TSDB.

## 7. Service Impact

| Service / repo | Change |
|---|---|
| `services/atlas-channel/atlas.com/channel/session/processor.go` | Wrap the `announceEncrypted` call in `Announce` with an OTel span (§4.2). |
| `services/atlas-channel/atlas.com/channel/tracing/tracing.go` | Read `TRACE_SAMPLING_RATIO` env var; configure SDK sampler accordingly (§4.4). |
| `services/atlas-channel/atlas.com/channel/go.mod` | Pull in `go.opentelemetry.io/otel/attribute` and `codes` if not already present. |
| `deploy/k8s/env-configmap.yaml` | Add `TRACE_SAMPLING_RATIO: "1.0"`. |
| `deploy/compose/.env.example` | Add `TRACE_SAMPLING_RATIO=1.0`. |
| `deploy/grafana/dashboards/atlas-latency.json` (new) | Dashboard JSON (§4.5). |
| `deploy/grafana/dashboards-provisioning.yaml` (new, if needed) | Grafana file-provider provisioning config. |
| `docs/observability.md` (new) | Documentation (§4.6). |
| **External: `<infra-repo>/`** | One of: (a) `observability-tempo.yml` overrides block + `metrics_generator.processor.span_metrics.dimensions`, OR (b) new OTel collector deployment / Alloy extension. Out-of-tree change tracked here for completeness. |

The other Atlas services (atlas-consumables, atlas-inventory, atlas-saga-orchestrator, etc.) require **zero code changes** in this task. They emit spans today via existing instrumentation; spanmetrics will pick them up automatically once the pipeline is enabled.

## 8. Non-Functional Requirements

### 8.1 Performance

- The added span around `session.Announce` SHALL have negligible per-call overhead. OTel SDK span creation+end is sub-microsecond when the active sampler is no-op; with a 100% sampler the cost is dominated by attribute encoding and is bounded by Atlas's outbound packet rate (a single connected character produces at most a few packets per second; even a 50-character session is well under 1k spans/sec, comfortably within Tempo + Prometheus capacity).
- The pipeline addition (Tempo overrides or new collector) SHALL NOT increase the typical request critical-path latency. Spanmetrics processing happens *after* spans are persisted; it is asynchronous to the request.

### 8.2 Cardinality budget (security-adjacent — observability quality)

Spanmetrics labels are the dimension that determines Prometheus series count. Each unique combination of label values is a distinct series. To prevent Prometheus from being overwhelmed, the following labels are **allowed** as spanmetrics dimensions:

- `service_name` — bounded (~50 services).
- `span_name` — bounded (curated set of manual span names + auto-instrumented handler names).
- `span_kind`, `status_code` — small enums.
- `writer.name` — bounded (~30 packet writers).
- `tenant.id` — bounded (a handful of tenants in dev; bounded to project policy in prod).
- `world.id` — bounded (single-digit worlds per tenant).

The following are **forbidden** as spanmetrics dimensions, even if they appear on the underlying span:

- `character.id`, `account.id` — unbounded (any user).
- `session.id`, `transaction.id`, `request.id` — unbounded UUID-per-call.
- `item.id` (templateId) — bounded, but ~10k values; not a useful dimension for the use-item dashboard.
- Any free-form string (player name, error message text, etc.).

The §4.6 documentation MUST restate this list. The metrics-generator / spanmetrics processor configuration MUST explicitly list the allowed dimensions; the default of "all attributes become labels" is unacceptable.

### 8.3 Multi-tenancy

`tenant.id` is a span attribute on `session.Announce` (§4.2) and a spanmetrics dimension (§8.2). Dashboard panels SHALL include a `$tenant` Grafana variable (default: all) so operators can scope to a single tenant when investigating a region-specific issue. Existing Atlas tenant-extraction patterns (`tenant.MustFromContext(ctx)`) apply.

### 8.4 Observability of the observability pipeline

When span-metrics generation is enabled in Tempo, Tempo exposes `tempo_metrics_generator_*` metrics about its own pipeline health. The dashboard SHALL include one panel showing `tempo_metrics_generator_processed_spans_total` rate so we can detect a stuck pipeline.

If pathway (b) is chosen (collector / Alloy), equivalent collector self-metrics SHALL be panelled.

### 8.5 Backward compatibility

Existing Tempo trace ingestion SHALL be unaffected. A sudden absence of `traces_spanmetrics_*` series after deploy is the only failure mode that would indicate the pipeline is broken; existing trace search continues to work either way.

## 9. Open Questions

1. **Pipeline pathway choice** — (a) Tempo `metrics_generator` overrides vs (b) collector / Alloy spanmetrics connector. Deferred to `/design-task`.
2. **Dashboard provisioning mechanism** — does the cluster Grafana already have a file-provider for dashboards-as-code, or do we need to add one in this task? Design phase to confirm by reading `<infra-repo>/observability-grafana.yml`.
3. **Spanmetrics naming convention** — `traces_spanmetrics_*` (Tempo and collector spanmetrics processor) vs `traces_span_metrics_*` (older OTel collector versions). Implementation MUST verify which is emitted by the chosen pathway and use that prefix in PromQL.
4. **Does the cluster Tempo deployment have `metrics_generator` actually enabled at runtime?** Config has the storage block but no `target` line in the Tempo command-line flags / env was inspected during this PRD write-up. Design phase to verify and patch if missing.
5. **The session.Announce span name** — is `session.Announce` the right name, or should it be `client.outbound.write` (more vendor-neutral) or `atlas.session.announce`? Defer to design.

## 10. Acceptance Criteria

Concrete, testable. Each item MUST hold true on a cluster after the task ships.

1. ✅ `histogram_quantile(0.5, sum by (le) (rate(traces_spanmetrics_latency_bucket{service_name="atlas-channel", span_name="session.Announce"}[5m])))` (or the Tempo-flavoured equivalent) returns a numeric value in Grafana Explore.
2. ✅ The same query, scoped `writer_name="InventoryChangeWriter"`, returns a numeric value separately from `writer_name="StatChangedWriter"`.
3. ✅ `traces_spanmetrics_calls_total{service_name="atlas-channel", span_name="session.Announce"}` increments while the user uses items in-game.
4. ✅ The Grafana dashboard at `deploy/grafana/dashboards/atlas-latency.json` (or wherever design lands) loads without errors and displays at minimum the six panels listed in §4.5.
5. ✅ A `$tenant` variable on the dashboard switches all panels to a single-tenant view.
6. ✅ Setting `TRACE_SAMPLING_RATIO=0.5` in the configmap and rolling atlas-channel halves the rate of new traces in Tempo (verifiable via Tempo's own admin metrics or by Loki-counting trace IDs over a fixed window) WITHOUT halting the pipeline.
7. ✅ Setting `TRACE_SAMPLING_RATIO=1.0` (or unsetting it) restores 100% sampling.
8. ✅ Forbidden cardinality dimensions (§8.2) do NOT appear in `count by (__name__) ({__name__=~"traces_spanmetrics_.*"})` output.
9. ✅ A panel showing `rate(tempo_metrics_generator_processed_spans_total[5m])` (or collector equivalent) is non-zero and stable.
10. ✅ The use-item flow latency from the task-038 trace (request → outbound packet) is visible on the dashboard within 30 seconds of an in-game item use, broken down per service/handler/writer.
11. ✅ A new manual span added in any Atlas service (one-line `tracer.Start`) appears in spanmetrics within 60 seconds with no further configuration, *unless* it requires a new dimension — in which case the spanmetrics-config update is documented in `docs/observability.md`.
12. ✅ `docs/observability.md` exists and a new developer can follow it to add a span + verify it appears in Grafana, end-to-end, in under 10 minutes.
13. ✅ Existing Tempo trace search continues to work; trace links from Loki ("view trace in Tempo") continue to resolve; no regression in trace volume or trace-search latency.
