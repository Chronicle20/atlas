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

## Filtering by environment

Every per-environment pod carries the label `atlas.env=<token>`, and PR-environment pods additionally carry `atlas.pr-number=<N>`. Use these labels to scope queries:

- `main` env: `atlas.env=main`
- PR env: `atlas.env=<4-char-hex>` (deterministic per PR — see `docs/runbooks/ephemeral-pr-deployments.md`)

### Loki

```logql
{atlas_env="a3f7"} |= "ERROR"
```

(Note: Promtail / Loki normalises Kubernetes label keys with dots to underscores at ingestion. `atlas.env` → `atlas_env` in LogQL selectors.)

### Prometheus

```promql
sum by (pod) (rate(http_request_duration_seconds_count{atlas_env="a3f7"}[5m]))
```

### Grafana

The `atlas-pr-environments` dashboard (when present in the cluster's Grafana) summarises open envs, time-to-ready, cleanup status, and bootstrap step durations. If absent, install via the standard Grafana dashboards-as-code mechanism on the cluster.
