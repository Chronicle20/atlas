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
