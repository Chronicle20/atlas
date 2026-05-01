# atlas-tracing

Shared OTel tracer setup for Atlas Go services. Exposes `InitTracer(serviceName)` and `Teardown(l)` that previously lived as 54 byte-identical copies under `services/atlas-*/.../tracing/tracing.go`.

Reads `TRACE_ENDPOINT` (OTLP gRPC target) and `TRACE_SAMPLING_RATIO` (float in `[0.0, 1.0]`, default `1.0`) from the environment. See `docs/observability.md` in the repo root for the pipeline overview.
