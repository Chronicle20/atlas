# task-082 — DATA_SERVICE_URL revert (live debugging finding)

## Symptom
After deploying the fix to `atlas-main`, opening Character Info on a character with a
monster-book cover set no longer crashed, but rendered **no monster** — the fix appeared
to do nothing.

## Root cause
Task 10 added an explicit env override:

```yaml
- name: DATA_SERVICE_URL
  value: "http://atlas-ingress.atlas.svc.cluster.local:80/api/"
```

That value was copied from the kustomize **base** `env-configmap.yaml`. The live
`atlas-main` overlay rewrites the ingress host to `atlas-ingress.atlas-main.svc.cluster.local`;
the bare-`atlas` host does **not** resolve in that namespace. Verified from the
`atlas-monster-book` pod:

- `wget http://atlas-ingress.atlas.svc.cluster.local/...` → `bad address` (DNS NXDOMAIN)
- `wget http://atlas-ingress.atlas-main.svc.cluster.local/api/data/consumables/2380000` →
  `200 OK`, `{"monsterBook":true,"monsterId":100100,...}`

So `resolveCoverMobId` could never reach atlas-data, hit its fail-safe path, and persisted
`cover_mob_id = 0`. Combined with lazy backfill (OQ-2 — resolution runs only at cover-set
time), every cover resolved to 0, so the Character-Info cover field was always sent as 0.

Live confirmation: `GET /characters/19/monster-book` →
`{"coverCardId":2380000,"coverMonsterId":0}`.

## Fix
Remove the `DATA_SERVICE_URL` override entirely. `requests.RootUrl("DATA")` then falls back
to `BASE_SERVICE_URL`, which the overlay sets correctly per environment — exactly what
`atlas-npc-shops` relies on for its identical `RootUrl("DATA")` consumable call. No service
in `deploy/k8s/base/` uses a direct `*_SERVICE_URL` override; this restores that convention.

## Verify after deploy (Argo/GitOps reconcile)
1. Confirm the pod no longer has `DATA_SERVICE_URL` (falls back to `BASE_SERVICE_URL`).
2. **Re-set** the character's monster-book cover (lazy backfill — existing rows keep
   `cover_mob_id = 0` until next set).
3. `GET /characters/{id}/monster-book` → `coverMonsterId` non-zero (e.g. card `2380000` →
   `100100`); opening Character Info renders the monster with no crash.
