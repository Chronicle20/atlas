# Task-040 — Risks

## R1 — Cardinality blow-up in Prometheus

**Severity:** High.

**Trigger:** A spanmetrics dimension is configured that has unbounded values — most plausibly `character.id` accidentally promoted, or a free-form attribute (player name, error text) flowing through.

**Mitigation:** PRD §8.2 lists allowed and forbidden dimensions explicitly. Implementation MUST configure the dimensions list explicitly (not "all attributes"). Acceptance criterion #8 verifies post-deploy.

**Detection:** `count by (__name__) ({__name__=~"traces_spanmetrics_.*"})` over time. A series count that grows linearly with active players is the failure signature.

**Recovery:** Drop the offending dimension from spanmetrics config, restart Tempo / collector. Existing series go stale and are eventually compacted out by Prometheus.

## R2 — Pipeline pathway mis-selection

**Severity:** Medium.

**Trigger:** Design phase picks Tempo `metrics_generator` because it's simpler, then later we want to add filtering / redaction / cross-cluster routing that Tempo's pipeline doesn't support, forcing a re-architecture to a collector.

**Mitigation:** PRD §4.1 documents both pathways. Design phase weighs flexibility vs blast radius. The decision is reversible — switching from (a) to (b) later is a configmap edit + service env-var change, not a code rewrite, because the manual span instrumentation in atlas-channel is pathway-agnostic.

## R3 — Tempo `metrics_generator` not actually running

**Severity:** Medium.

**Trigger:** The bee Tempo deployment config has the `metrics_generator.registry` and `.storage` blocks (PRD §4.1) but the Tempo binary may not be configured with the `metrics-generator` target enabled in its `--target=` flag or `target:` config. If so, enabling overrides won't help — the component isn't running.

**Mitigation:** Design phase verifies (open question #4 in PRD §9). If not enabled, the Tempo deployment-spec change to add the target is a one-line patch but is required.

## R4 — Sampling skews aggregate metrics

**Severity:** Low.

**Trigger:** Operator sets `TRACE_SAMPLING_RATIO=0.1` to reduce noise; spanmetrics rate panels show 1/10th the actual call rate, leading to incorrect "the system is idle" conclusions.

**Mitigation:** PRD §4.4 documents this. The dashboard MUST include a small annotation noting that rates are subject to the sampling ratio. The default of `1.0` keeps this from being an everyday issue. A `TRACE_SAMPLING_RATIO` annotation panel showing the current value would be a defence-in-depth.

## R5 — `session.Announce` span overhead in hot paths

**Severity:** Low.

**Trigger:** Heavy in-combat sessions (50+ packets/sec per character) add measurable per-packet overhead from span creation + attribute encoding.

**Mitigation:** OTel SDK spans are sub-microsecond per call; even 1k spans/sec is well below the cost of the actual encrypt-and-write that follows. PRD §8.1 calls this out. If it ever becomes a problem, the `TRACE_SAMPLING_RATIO` lever is in place.

## R6 — Dashboard JSON drift between checked-in file and runtime Grafana

**Severity:** Low.

**Trigger:** Operator edits the dashboard in-place via the Grafana UI; the runtime version drifts from the file in source control; next provision overwrites the in-place edits and frustrates the operator.

**Mitigation:** PRD §4.5 specifies file-provider provisioning, which Grafana enforces as the source of truth. The dashboard MUST be marked `editable: false` (or the team agrees to a "edit via PR only" convention). Documentation in `docs/observability.md` SHOULD describe the edit workflow (clone → modify → PR → re-provision).

## R7 — Observability pipeline becomes a single point of failure for "is the game working?"

**Severity:** Low (development cluster).

**Trigger:** The dashboard becomes the primary tool for triaging player reports; if Tempo or Prometheus goes down, troubleshooting halts.

**Mitigation:** Out of scope for this task — Loki + structured logs continue to work as the fallback diagnostic. PRD §10 acceptance criterion #13 explicitly preserves the existing log-based path. This is a "nice problem to have" follow-up.
