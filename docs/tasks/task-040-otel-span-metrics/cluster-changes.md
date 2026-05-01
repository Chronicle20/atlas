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
