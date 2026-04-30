# OTel Span Metrics + Client-Write Latency Instrumentation — Design

Version: v1
Status: Draft
Created: 2026-04-30
Companion PRD: `prd.md`
Companion risks: `risks.md`

---

## 1. Overview

This design realizes the PRD by:

1. Enabling Tempo's built-in `metrics_generator` `span-metrics` processor with a curated dimension allowlist, producing `traces_spanmetrics_*` series in Prometheus.
2. Adding one manual OTel span around `session.Announce` in atlas-channel — the single chokepoint for outbound client packets — with `writer.name`, `tenant.id`, `world.id` attributes.
3. Extracting the 54 byte-identical copies of `tracing/tracing.go` across services into a new shared library `libs/atlas-tracing`, augmented with `TRACE_SAMPLING_RATIO` env-driven sampling.
4. Shipping a Grafana dashboard JSON in the Atlas repo, file-provider-provisioned into the bee Grafana via a configmap-mounted directory.
5. Documenting the end-to-end pipeline, extension recipes, and cardinality budget in a new `docs/observability.md`.

The design picks PRD §4.1 pathway (a) — Tempo `metrics_generator`. The single largest deviation from PRD §7 is the libs/atlas-tracing extraction; the rest is refinement of details deferred to design.

## 2. Architecture

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

The pipeline is in-cluster, asynchronous to request handling. Spanmetrics processing happens after spans are persisted; it is off the request critical path. Atlas service `TRACE_ENDPOINT` is unchanged at `tempo.home:4317`.

## 3. Components

### 3.1 `libs/atlas-tracing` (new shared library)

Replaces 54 byte-identical copies of `services/atlas-*/atlas.com/*/tracing/tracing.go`.

- Module: `github.com/Chronicle20/atlas/libs/atlas-tracing`
- Path: `libs/atlas-tracing/`
- Public API (preserves the existing surface):
  - `tracing.InitTracer(serviceName string) (*sdktrace.TracerProvider, error)`
  - `tracing.Teardown(l logrus.FieldLogger) func(tp *sdktrace.TracerProvider) func()`
- Behavioral additions:
  - Reads `TRACE_SAMPLING_RATIO` env var as `float64` in `[0.0, 1.0]`.
  - Defaults to `1.0` when unset, unparseable, or out of range. Logs a warning at WARN level via the package-level logger when an invalid value is received.
  - Configures the SDK sampler as `sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))` so that sampling decisions propagate from upstream service context, preserving causality.
- `go.work` gains `./libs/atlas-tracing` in its `use` block.
- Each of the 54 services:
  - Adds `require github.com/Chronicle20/atlas/libs/atlas-tracing v0.0.0` (and matching `replace ... => ../../../libs/atlas-tracing` if the existing go.mod files use replace directives — to be confirmed in plan-phase).
  - Deletes its `tracing/` package.
  - Updates `main.go`'s import from `<service-module>/tracing` to `github.com/Chronicle20/atlas/libs/atlas-tracing`.

The lib has its own unit tests covering the env-var parsing edge cases (unset / invalid / `0.0` / `0.5` / `1.0` / `>1.0`).

### 3.2 `session.Announce` manual span

Edit confined to `services/atlas-channel/atlas.com/channel/session/processor.go`, the inner-most lambda of `Announce` (currently lines 173–179). The function signature is unchanged.

Span definition:

- Tracer: `otel.GetTracerProvider().Tracer("atlas-channel")` — same provider name as existing manual spans in this file (`teardown`, `session-destroy`).
- Name: `session.Announce`.
- Attributes:
  - `writer.name` (string) — from existing `writerName string` parameter.
  - `tenant.id` (string) — from `tenant.MustFromContext(ctx).Id().String()`.
  - `world.id` (int) — from `int(s.WorldId())`.
- Scope: encloses both the `writerProducer(writerName)` lookup and the `s.announceEncrypted(...)` call.
- Error handling: on either failure path, `span.RecordError(err)` + `span.SetStatus(codes.Error, err.Error())`.

Forbidden span attributes (per PRD §8.2 cardinality budget): `character.id`, `account.id`, `session.id`, `transaction.id`, packet payload bytes. The implementation MUST NOT add any of these to the span — even though they could be added later via the `dimensions:` allowlist, omitting them at the span level is defence-in-depth.

### 3.3 Tempo overrides (out-of-tree)

Append to `~/source/k3s/bee/observability-tempo.yml` ConfigMap `tempo-config` `tempo.yaml` data:

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

Built-in dimensions (`service_name`, `span_name`, `span_kind`, `status_code`) are auto-included and don't need listing. `histogram_buckets` keeps Tempo's default exponential bucketing — `[0.002, 0.004, 0.008, 0.016, 0.032, 0.064, 0.128, 0.256, 0.512, 1.024, 2.048, 4.096, 8.192, 16.384]` seconds — which is sub-millisecond-floor and well-suited to the use-item flow.

Tempo 2.7.1 hot-reloads overrides; no Tempo restart is required. Verification: `kubectl logs -n observability tempo-0 | grep "reloaded"`.

The Tempo deployment runs with default `target=all`, which already includes the metrics-generator component. No `args:` change is needed (R3 verification — confirmed by reading the existing manifest).

### 3.4 Grafana dashboard + provisioning

Repo-side artifacts under new directory `deploy/grafana/`:

- `deploy/grafana/dashboards/atlas-latency.json` — the dashboard. `editable: false`. Stable UID. Contains six panels + one self-health panel + two template variables + one annotation. PromQL/LogQL is inlined in each panel definition for copy-pasteability.
- `deploy/grafana/dashboards-provider.yaml` — the file-provider config Grafana reads at `/etc/grafana/provisioning/dashboards`:

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

- `deploy/grafana/apply.sh` — idempotent bash helper:

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

- `deploy/grafana/README.md` — short, points at `docs/observability.md` for the full picture and at `apply.sh` as the entrypoint.

Cluster-side change to `~/source/k3s/bee/observability-grafana.yml`:

- Add to `volumeMounts`:
  ```yaml
  - name: dashboards
    mountPath: /etc/grafana/provisioning/dashboards
  ```
- Add to `volumes`:
  ```yaml
  - name: dashboards
    configMap:
      name: grafana-dashboards-atlas
  ```

The configmap *content* is owned by the Atlas repo (apply.sh recreates it from the JSON + provider YAML). The configmap *mount* is owned by the cluster manifest. Ordering: the Atlas-repo PR ships first (dashboards JSON, provider YAML, apply.sh), then the cluster-side mount is added.

### 3.5 Sampling env var wiring

- `deploy/k8s/env-configmap.yaml` — append `TRACE_SAMPLING_RATIO: "1.0"` near the existing `TRACE_ENDPOINT` line.
- `deploy/compose/.env.example` — append `TRACE_SAMPLING_RATIO=1.0`.
- `deploy/compose/.env` — append `TRACE_SAMPLING_RATIO=1.0` (the local-dev companion, per existing convention).

The env var is fleet-wide once the libs/atlas-tracing rollout completes. All 54 services honor it uniformly.

### 3.6 Documentation

New file `docs/observability.md` covering:

- **Pipeline diagram** — the ASCII art from §2 of this design.
- **How to add a manual span** — one paragraph + a code snippet:
  ```go
  ctx, span := otel.GetTracerProvider().Tracer("<service>").Start(ctx, "Feature.entryPoint")
  defer span.End()
  ```
  Note that spanmetrics auto-publishes the new span within ~60 seconds with no further config, *unless* it requires a new dimension.
- **How to add a new spanmetrics dimension** — paragraph describing the Tempo overrides edit + a warning about the cardinality budget.
- **How to add a new dashboard panel** — a panel JSON template + the `apply.sh` invocation.
- **Cardinality budget** — restated allowlist + restated forbidden list (PRD §8.2).
- **The 12-step smoke test** from §6 of this design, reproduced verbatim as the "verify your changes" recipe.
- **Sampling caveat** — `TRACE_SAMPLING_RATIO < 1.0` skews the rate panels proportionally; the dashboard annotates this.

## 4. Dashboard panels — definitions

Tempo emits Prometheus-compatible labels by transforming attribute keys: `writer.name` → `writer_name`, `tenant.id` → `tenant_id`, `world.id` → `world_id`.

Template variables:

- `$tenant` (multi, includeAll, default=All) — `label_values(traces_spanmetrics_calls_total, tenant_id)`
- `$writer` (multi, includeAll, default=All) — `label_values(traces_spanmetrics_calls_total{span_name="session.Announce"}, writer_name)`

Time range default: last 30 minutes.

Panels:

1. **`session.Announce` latency by writer** — multi-series time-series. Three queries (p50, p95, p99):
   ```
   histogram_quantile(0.50, sum by (le, writer_name) (rate(traces_spanmetrics_latency_bucket{service_name="atlas-channel", span_name="session.Announce", tenant_id=~"$tenant", writer_name=~"$writer"}[5m])))
   ```
   (Same shape for 0.95 and 0.99.) Y-axis: seconds. Legend: `{{writer_name}} p50`.

2. **`session.Announce` rate by writer** — stacked area:
   ```
   sum by (writer_name) (rate(traces_spanmetrics_calls_total{service_name="atlas-channel", span_name="session.Announce", tenant_id=~"$tenant"}[1m]))
   ```

3. **Use-item entry-handler latency in atlas-channel** — auto-instrumented packet-handler span. Span name: `CharacterItemUseHandler` (resolved during plan-phase by reading `services/atlas-channel/atlas.com/channel/socket/handler/`; placeholder until verified):
   ```
   histogram_quantile(0.95, sum by (le) (rate(traces_spanmetrics_latency_bucket{service_name="atlas-channel", span_name="CharacterItemUseHandler", tenant_id=~"$tenant"}[5m])))
   ```

4. **Inventory `compartment_command` consumer latency** — auto-instrumented Kafka consumer span (atlas-kafka emits `c.name` as the span name):
   ```
   histogram_quantile(0.95, sum by (le) (rate(traces_spanmetrics_latency_bucket{service_name="atlas-inventory", span_name="compartment_command", tenant_id=~"$tenant"}[5m])))
   ```
   This panel covers reserve, consume, and every other compartment command at consumer-group grain — finer slicing requires manual sub-spans (PRD §2 non-goal; documented escape hatch in observability.md).

5. **Inventory `consumable_command` consumer latency** — same shape as panel 4. Plan-phase to verify the consumer's actual `c.name` (may be `consumable_command` or another value) by reading `services/atlas-inventory/atlas.com/inventory/kafka/consumer/`.

6. **Saga skip rate by reason** (Loki):
   ```
   sum by (reason) (rate({service="atlas-saga-orchestrator"} |= "reason=" | logfmt | reason!="" [5m]))
   ```
   Verifies task-038's nil-UUID guard continues to fire and surfaces any `saga_not_found` regression. Loki-derived because the saga-orchestrator emits skip reasons as structured logrus fields today; no new span needed.

Self-health panel (PRD §8.4):

7. **Tempo metrics-generator throughput**:
   ```
   rate(tempo_metrics_generator_processed_spans_total[5m])
   ```
   Non-zero means the pipeline is alive.

Annotation: a static text annotation on every panel reading "Rates and counts are subject to TRACE_SAMPLING_RATIO. Default 1.0; check cluster configmap." (R4 defence-in-depth.)

## 5. Cardinality budget — enforcement mechanism

Tempo's `metrics_generator.processor.span_metrics.dimensions` list is the single point of enforcement. An attribute that is present on a span but NOT in the dimensions list is silently dropped from spanmetrics. This means:

- `character.id`, `account.id`, `session.id` may appear on spans (they're useful for trace search) but cannot leak into Prometheus series.
- The day someone adds a new attribute on `session.Announce` (e.g., `packet.size_bytes`), it does NOT automatically become a Prometheus label. Adding it requires a deliberate Tempo overrides edit.

Forbidden as Prometheus dimensions, even if they appear on spans:

- `character.id`, `account.id` — unbounded.
- `session.id`, `transaction.id`, `request.id` — unbounded UUIDs.
- `item.id` (templateId) — bounded but ~10k values; useful for trace search, not for the use-item dashboard.
- Free-form strings (player name, error message text).

This list is restated in `docs/observability.md`.

## 6. Smoke test sequence

This sequence verifies acceptance criteria #1–#9 and #13. AC #10 is verified during step 4. AC #11/#12 are verified by self-walking `docs/observability.md`.

1. `kubectl apply -f ~/source/k3s/bee/observability-tempo.yml`. Tempo's overrides hot-reload; confirm `kubectl logs -n observability tempo-0 | grep "reloaded"`.
2. `cd ~/source/atlas-ms/atlas/deploy/grafana && ./apply.sh`. Grafana picks up the configmap on the rollout-restart inside the script.
3. `kubectl rollout restart deployment/atlas-channel -n atlas`.
4. Log in to a test character. Use a potion 5 times. Walk a few maps.
5. Grafana Explore (Prometheus): `traces_spanmetrics_calls_total{service_name="atlas-channel", span_name="session.Announce"}` returns non-zero rows broken down by `writer_name`. ⇒ AC #3.
6. `histogram_quantile(0.95, sum by (le) (rate(traces_spanmetrics_latency_bucket{service_name="atlas-channel", span_name="session.Announce"}[5m])))` returns a number. ⇒ AC #1.
7. Same query scoped `writer_name="InventoryChangeWriter"` and `writer_name="StatChangedWriter"` — different numeric values. ⇒ AC #2.
8. Grafana → "Atlas Latency" dashboard loads. All panels render. `$tenant` variable populates. ⇒ AC #4, #5.
9. `rate(tempo_metrics_generator_processed_spans_total[5m])` non-zero. ⇒ AC #9.
10. `count by (__name__) ({__name__=~"traces_spanmetrics_.*"})` does not include any series with `character_id`, `account_id`, `session_id`, `transaction_id`, `item_id` labels. ⇒ AC #8.
11. Set `TRACE_SAMPLING_RATIO=0.5` in env-configmap, roll atlas-channel, observe `rate(traces_spanmetrics_calls_total{service_name="atlas-channel"}[1m])` halve under steady traffic. Restore to `1.0`. ⇒ AC #6, #7.
12. Tempo trace search via Grafana Explore (Tempo datasource) returns recent traces. ⇒ AC #13.

## 7. Service / repo impact (delta vs PRD §7)

| Item | PRD §7 | Design |
|---|---|---|
| `services/atlas-channel/.../session/processor.go` | edit | edit (the inner Announce lambda) |
| `services/atlas-channel/.../tracing/tracing.go` | edit | **delete** (replaced by libs import) |
| All 53 other services' `tracing/tracing.go` | unchanged | **delete** (replaced by libs import) |
| `libs/atlas-tracing/` | n/a | **new** |
| `go.work` | unchanged | append `./libs/atlas-tracing` |
| 54 service `go.mod` files | unchanged | each adds the new lib dependency |
| 54 service `main.go` files | unchanged | each: import path swap |
| `deploy/k8s/env-configmap.yaml` | add `TRACE_SAMPLING_RATIO` | same |
| `deploy/compose/.env.example` | add `TRACE_SAMPLING_RATIO` | same; also `.env` |
| `deploy/grafana/dashboards/atlas-latency.json` | new | new |
| `deploy/grafana/dashboards-provider.yaml` | maybe | new |
| `deploy/grafana/apply.sh` | n/a | new |
| `deploy/grafana/README.md` | n/a | new |
| `docs/observability.md` | new | new |
| `~/source/k3s/bee/observability-tempo.yml` | one of two paths | overrides block (path a) |
| `~/source/k3s/bee/observability-grafana.yml` | maybe | yes — adds dashboards volumeMount + volume |
| `~/source/k3s/bee/observability-alloy.yml` | maybe | unchanged |

## 8. Open items deferred to plan-phase

These do not block design approval but require code-reading during plan-write to lock exact strings:

1. **Resolve the use-item entry-handler span name** — read `services/atlas-channel/atlas.com/channel/socket/handler/` for the constant passed as `name` at `handle.go:57` for `CharacterItemUseHandle`-equivalent. Lock exact value into panel 3 PromQL.
2. **Resolve atlas-inventory consumer span names** — read `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go` and the consumable analog for the exact `c.name` strings. Lock into panels 4–5.
3. **Confirm go.mod replace-directive convention** — inspect a few existing service `go.mod` files for how they reference other libs (`replace` vs go.work-only). Mirror the convention in the libs/atlas-tracing dependency wiring.
4. **Confirm Dockerfile build path** — verify a service's Dockerfile builds with `go.work` available or with vendored deps. If go.work isn't honored at build time, the libs/atlas-tracing dep needs `replace ... => ../../../libs/atlas-tracing` in each go.mod and the Dockerfile build context must include the libs directory. Plan-phase verifies one service's Dockerfile build before fanning out.

## 9. Risks (delta vs `risks.md`)

The PRD's `risks.md` covers R1–R7. Two additions for this design:

### R8 — Docker build breakage during libs/atlas-tracing rollout

**Severity:** Medium. **Trigger:** A service's Dockerfile builds with neither `go.work` honored at build time nor a vendored copy of the new lib, and the new dependency cannot be resolved. **Mitigation:** Plan-phase verifies one service's Docker build end-to-end before fanning out to all 54. If the Dockerfile pattern needs adjustment (e.g., copying the libs directory into the build context, or adding a `replace` directive), it is fixed once and propagated. CLAUDE.md's "always verify Docker builds when changing shared libraries" applies. **Recovery:** Roll back the libs/atlas-tracing extraction; restore the per-service `tracing/tracing.go` files; sampling-ratio support is then atlas-channel-only as a fallback.

### R9 — `tenant.id` cardinality if tenant count grows

**Severity:** Low today, medium if the deployment ever multi-tenants beyond a small fixed set. **Trigger:** Tenant count in the dev cluster grows past ~50, or the prod fleet ever exceeds the project's planned tenant ceiling. **Mitigation:** Documented in the cardinality budget. If tenant count exceeds ~50, drop `tenant.id` from the dimensions allowlist (Tempo overrides edit) and rely on per-tenant Tempo workspace isolation instead.

## 10. Testing strategy

Test coverage maps to the components added:

- **`libs/atlas-tracing` unit tests** — env-var parsing edge cases (unset, invalid, `0.0`, `0.5`, `1.0`, `>1.0`, negative). Verify the resulting sampler is `ParentBased(TraceIDRatioBased(...))` with the parsed ratio. Verify warning is logged on invalid input. (Standard Go `testing` package.)
- **`session.Announce` integration** — extend an existing `session_test.go`-style test (or add one) to assert that calling `Announce(...)` produces a span with the correct name and attributes via a `MockTracerProvider` (the same pattern atlas-kafka uses at `libs/atlas-kafka/consumer/manager_test.go:84`).
- **Per-service smoke** — for each of the 54 services, after the libs swap, `go test ./...` must pass. The plan batches these in groups for speed.
- **Per-service Docker build smoke** — for each of the 54 services, the Dockerfile build must succeed. Plan groups these into a CI matrix or local script.
- **Out-of-tree verification** — the 12-step smoke test from §6 is the cluster-level acceptance test.

The libs/atlas-tracing extraction is the largest test-surface multiplier in this task. The plan's risk-management lever is to land the lib + atlas-channel migration in PR 1 (the smallest blast radius that satisfies AC #6/#7), then fan out the remaining 53 services in subsequent PRs grouped by build-cluster proximity. The PR phasing is a plan-phase decision; the design's commitment is "single new lib, all 54 services migrated within task-040 scope before this task is closed." Phased PRs are allowed; partial migration is not.

## 11. Out of scope (explicit non-goals, restated)

These are reaffirmed from PRD §2 and not changed by design:

- Tracing-driven alerting / SLO definitions.
- Client-side timing.
- Manual span instrumentation of business-logic functions other than `session.Announce`.
- Manual span instrumentation of atlas-consumables / atlas-inventory handlers (consumer-group grain is sufficient for v1).
- Migrating off Tempo to a vendor-neutral OTel collector (deferred; reversible in a future task).
- Re-architecting the OTel pipeline beyond enabling spanmetrics.
- Promoting `character.id`, `account.id`, `session.id`, `transaction.id`, `item.id` to spanmetrics dimensions.

---

End of design.
