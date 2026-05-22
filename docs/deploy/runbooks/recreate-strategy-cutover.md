# Runbook: `RollingUpdate` → `Recreate` Deployment strategy cutover

## When to use

A Deployment that was previously deployed with `strategy.type=RollingUpdate`
and `strategy.rollingUpdate.{maxSurge,maxUnavailable}` needs to switch to
`strategy.type=Recreate`. Server-Side Apply (SSA) leaves the old
`rollingUpdate` keys attached to the resource as orphan-managed fields
even after the manifest drops them, which prevents the strategy change
from taking effect.

Confirmed bite on 2026-05-22 during atlas-data migration.

## Symptoms

- `kubectl describe deploy/atlas-<svc> | grep -A3 Strategy` shows
  `RollingUpdate` even after `kustomize build … | kubectl apply -f-`.
- Pods continue to roll one-at-a-time instead of being torn down before
  the next set comes up.

## Workaround

Strip the orphan fields with a JSON patch BEFORE re-applying:

```bash
kubectl -n atlas-main patch deploy/atlas-data --type=json -p='[
  {"op":"remove","path":"/spec/strategy/rollingUpdate"}
]'
```

Then re-apply the manifest:

```bash
kustomize build deploy/k8s/overlays/main | kubectl apply -f -
```

Verify:

```bash
kubectl -n atlas-main describe deploy/atlas-data | grep -A3 Strategy
```

Expected: `Strategy: Recreate`.

## Optional: prevent recurrence on first deploy elsewhere

For a future Deployment slated for `Recreate` from the start, add a
kustomize patch that explicitly nulls `/spec/strategy/rollingUpdate`:

```yaml
patches:
  - target:
      kind: Deployment
      name: atlas-<svc>
    patch: |-
      - op: remove
        path: /spec/strategy/rollingUpdate
```

Remove the patch after the first apply succeeds — keeping it around
forever costs nothing but adds noise.

## Notes

- atlas-main was unblocked on 2026-05-22 with the `kubectl patch` recipe.
- This issue only resurfaces on similar strategy migrations; routine
  Deployments are unaffected.
- Origin: SSA orphan-field semantics, not a kustomize bug.
