# May 19, 2026 — Manual recovery of PR 491 + PR 522 wedged teardowns

## Trigger

User noticed Argo CD wasn't cleaning up PR #491 after merge (merged 2026-05-18T22:11:56Z). Investigation also revealed PR #522 (label-removed, still OPEN on GitHub) was in the same wedged state.

## Diagnosis

Both `Application` CRDs in `argocd` namespace had:
- `deletionTimestamp` set (issued by ApplicationSet controller within ~30s of PR-close / label-removal)
- `resources-finalizer.argocd.argoproj.io` already drained (namespace + workload pruned)
- `post-delete-finalizer.argocd.argoproj.io` and `.../cleanup` still present
- `status.conditions[].message`: `DeletionError: namespaces "atlas-pr-<N>" not found`

Root cause: the ApplicationSet syncs with `CreateNamespace=true`, so the destination namespace is an Argo-managed resource and gets pruned by `resources-finalizer` before the PostDelete cleanup Job can be created in it. See `prd.md` §4.1 for the structural fix.

## Per-PR env-hash table

`ATLAS_ENV = first_4_chars( hex( sha256( "pr-${PR_NUMBER}" ) ) )`

| PR | computed env (real) | annotation on Application (drifted, ignored) |
|---|---|---|
| 491 | `ed86` | `f78b` |
| 522 | `a476` | `d496` |

## Pre-recovery state (counts)

| Item | 491 (`ed86`) | 522 (`a476`) |
|---|---|---|
| Postgres DBs | 30 | 29 |
| Kafka topics | 132 | 132 |
| Kafka consumer groups | 38 | 4 |
| Redis keys | 32 | 21 |
| ghcr image tags `pr-N-*` | 0 (already cleaned by `pr-cleanup.yml` on merge) | 56 (one tag per docker_image service) |
| `bot/pr-N-resolved` branch | present | present |
| Argo `Application` | wedged Terminating | wedged Terminating |

## Recovery commands run

### 1. Drop Postgres databases (in `postgres/postgres-deployment-54849df685-mds7v`)

```sh
for ENV in ed86 a476; do
  psql -U postgres -d postgres -tAc \
    "SELECT datname FROM pg_database WHERE datname ~ '-${ENV}\$';" \
  | while read db; do
      [ -z "$db" ] && continue
      psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS \"$db\" WITH (FORCE);"
    done
done
```
Result: 30 + 29 DBs dropped, verified count 0 each.

### 2. Delete Kafka topics + consumer groups (in `kafka/kafka-broker-0`, `BS=localhost:9092`)

```sh
KT=/opt/kafka/bin/kafka-topics.sh
KG=/opt/kafka/bin/kafka-consumer-groups.sh

for ENV in ed86 a476; do
  # Topics
  $KT --bootstrap-server $BS --list 2>/dev/null \
    | grep -E -- "-${ENV}\$" \
    | xargs -r -n1 $KT --bootstrap-server $BS --delete --topic

  # Consumer groups (use read loop, not xargs — group names contain spaces and brackets)
  $KG --bootstrap-server $BS --list 2>/dev/null \
    | grep -E "\\[${ENV}\\]\$" \
    | while IFS= read -r g; do
        $KG --bootstrap-server $BS --delete --group "$g"
      done
done
```
Result: 132 + 132 topics deleted, 38 + 4 groups deleted, verified count 0 each.

Note: the xargs `-d '\n'` form from `cleanup.sh` (line ~58) failed silently in my first attempt — the issue is that `cleanup.sh` uses `xargs -r -d '\n' -n 1` which works on most environments but didn't work for me here. The `while IFS= read -r g` form is more portable. Worth verifying `cleanup.sh`'s real behavior under the actual PostDelete Job image.

### 3. Delete Redis keys (in `redis/redis-f8db9547-q7thc`)

```sh
for ENV in ed86 a476; do
  redis-cli --scan --pattern "${ENV}:*" | xargs -r -n 500 redis-cli DEL
done
```
Result: 32 + 21 keys deleted, verified count 0 each.

### 4. ghcr image tags

PR 491: already cleaned by `pr-cleanup.yml` on merge (workflow run `26063473151` succeeded).

PR 522: triggered `gh workflow run pr-cleanup.yml -f pr-number=522` (workflow_dispatch); run `26091852915` handled the cleanup with the repo `GHCR_TOKEN` secret (the local PAT lacks `write:packages`).

### 5. Patch out Application finalizers (last step)

```sh
for n in 491 522; do
  kubectl -n argocd patch application.argoproj.io atlas-pr-${n} \
    --type=merge -p '{"metadata":{"finalizers":[]}}'
done
```
Result: both Applications removed within seconds. `kubectl get applications -n argocd` shows neither.

## Remaining manual tasks (NOT done in this session)

- **`bot/pr-491-resolved` and `bot/pr-522-resolved` branches still exist on GitHub.** Local PAT (`~/.config/atlas/gh.env`) and the cluster's `argocd-repo-creds-chronicle20-atlas` Secret both return `403 "Resource not accessible by personal access token"` on `DELETE /repos/Chronicle20/atlas/git/refs/heads/bot/pr-N-resolved`. Operator needs to delete these manually with a PAT that has `Contents: write` on the repo, OR via the GitHub web UI.

## Latent bugs discovered during recovery (see `prd.md` for full treatment)

1. **Finalizer-ordering wedge** — affects every teardown, not just 491/522.
2. **24h `cleanup-grace` annotation is a no-op** — ApplicationSet deletes Application before the cron's grace logic runs.
3. **Cleanup token can't delete branches** — `argocd-repo-creds-chronicle20-atlas` lacks `Contents: write`. Silent failure on every cron run.
4. **`atlas.env` annotation drift** — annotation reads `f78b`/`d496`; pod labels and actual resources use `ed86`/`a476`. If cleanup had run, it would have targeted the wrong env. Root cause unknown.

## Pre-existing orphan inventory at end of session

`kubectl get applications -n argocd | grep atlas-pr-` returns nothing. ApplicationSet has zero generated Applications (no open PRs carry `deploy-env` label).

Other env hashes may still have residue from earlier wedges that nobody noticed. Sweep across the full DB / topic / Redis state would be needed to enumerate. Not done here. See `prd.md` §4.6 for the sweep tooling that will land with this task.
