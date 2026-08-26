# Incident: PR #1459 teardown deadlock (2026-08-26)

Source evidence for task-264. Every value below was read from the live cluster
on 2026-08-26 between 12:33Z and 12:45Z; nothing here is reconstructed.

## Timeline

| time (UTC) | event | evidence |
|---|---|---|
| 11:48:57 | auto-sync operation starts | `status.operationState.startedAt` |
| 11:59:27 | Application health → `Degraded` | `status.health.lastTransitionTime` |
| ~12:01 | ImagePullBackOff pods created | pod age 32m read at 12:33 |
| **12:12:50** | **Application `deletionTimestamp` set** | `metadata.deletionTimestamp` |
| 12:15:05 | `atlas-minio-init` Job created (PreSync hook) | `metadata.creationTimestamp` |
| 12:15:35 | `atlas-kafka-precreate`, `atlas-pr-create-dbs` created | `metadata.creationTimestamp` |
| 12:16:24 | `atlas-pr-bootstrap` Job created (Sync hook) | `metadata.creationTimestamp` |
| 12:21:23 | bootstrap attempt fails: `retry exhausted after 60 attempts: http_ok_tenant .../api/data/status` | pod log |
| 12:26:26 | `atlas-pr-predelete-purge` runs | `status.startTime` |
| 12:26:32 | purge fails: 3× HTTP 502 deleting tenant `f24c9d1e-1026-48db-b2b5-7957d69e5202` | pod log |
| 12:26:27 | `atlas-pr-bootstrap` marked for deletion, retains `argocd.argoproj.io/hook-finalizer` | `metadata.deletionTimestamp` + `finalizers` |
| 12:31:38 onward | `89 objects remaining for deletion` repeating, N never decreasing | controller log |

Note the ordering: the sync had been in flight for **24 minutes** when the delete
request landed, and it continued creating new hook resources for **4 minutes
after** it. This is an in-flight-sync overlap, not a sync triggered post-delete.

## Root cause

`Job/atlas-pr-bootstrap` held the runtime finalizer
`argocd.argoproj.io/hook-finalizer`. Only the Argo controller removes that
finalizer, and it does so as part of completing or terminating the owning
operation. The operation could not complete (its hook was failing) and was never
terminated by the delete path, so:

1. the hook Job never deleted;
2. `resources-finalizer.argocd.argoproj.io` blocked on it;
3. 93 remaining namespaced objects — 63 Services, 9 ConfigMaps, 5 Secrets, 4
   ServiceAccounts, 3 Pods, 3 Roles, 3 RoleBindings, 2 Jobs, 1 Ingress — were
   never issued for deletion.

The bootstrap hook could not succeed because the images it waits on had been
purged from GHCR by `pr-cleanup.yml`'s `delete-images` job on PR close:

```
Failed to pull image "ghcr.io/chronicle20/atlas-data/atlas-data:pr-1459-99281705":
  ... not found
```

Same failure on `atlas-channel`, `atlas-login`, `atlas-messages`,
`atlas-configurations`.

## Operator error worth recording

The first remediation attempt was:

```sh
kubectl -n argocd patch application atlas-pr-1459 --type=merge -p '{"operation":null}'
```

This unblocked the pre-delete phase but **made the deadlock permanent**. Clearing
`.operation` removes the operation spec without transitioning
`status.operationState.phase`, which stayed `Running`. The controller only
processes an operation when `app.Operation != nil`, so nothing could ever
transition that phase or reap the hook finalizer.

The correct call while the operation still exists is `argocd app terminate-op`,
which sets `phase: Terminating` *and* reaps hook resources. Once `.operation` has
been cleared, that path is no longer available and the hook Job's finalizer must
be patched off directly.

This is the basis for FR-4.4.

## Not a defect

The PreDelete purge failure is by design.
`deploy/k8s/overlays/pr/predelete-purge.yaml:1-7` documents the hook as
best-effort with the sweep CronJob as backstop. Both backstops were confirmed
active in the `argocd` namespace: `atlas-pr-cleanup` (`5 * * * *`) and
`atlas-pr-sweep-orphans` (`0 */6 * * *`). Tenant `f24c9d1e…` is reclaimed on the
next sweep; no manual purge is required.
