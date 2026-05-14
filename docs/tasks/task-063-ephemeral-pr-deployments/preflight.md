# Pre-flight findings (task-063)

## CREATEDB on db-credentials user
- Result: PASS 2026-05-08
- Action taken (if FAIL): n/a

### Evidence
Query against `postgres.home` as the user from `atlas/db-credentials`:

```
 rolname | rolcreatedb
---------+-------------
 atlas   | t
(1 row)
```

### Note on secret hygiene
`kubectl get secret -n atlas db-credentials -o jsonpath='{.data.DB_USER}' | base64 -d` returns `atlas \r\n` (a literal trailing space plus CR+LF). The same applies to `DB_PASSWORD`. Authentication only succeeds after stripping the trailing whitespace (`tr -d ' \r\n'`); the literal-space form fails with `password authentication failed for user "atlas "`. Atlas services in-cluster appear to tolerate this today, but the per-PR overlay tooling introduced in later phases should either strip whitespace defensively or the secret should be re-issued without the trailing whitespace. Tracking this here so a downstream phase can address it if needed.

## Kafka auto.create.topics.enable
- Result: true 2026-05-08
- Action taken (if false): n/a — broker is permissive; per-PR overlay does NOT need a PreSync topic-creation hook for ephemeral envs

### Evidence
The plan's stock label selector returns no pods (`kubectl get pod -A -l app.kubernetes.io/name=kafka` is empty) and `--entity-name 0` is wrong for this cluster (node.id is 1, not 0). Reproducible commands:

Pod selector that worked:
```
kubectl get pod -n kafka -l app=kafka -o name
# pod/kafka-broker-0
```

Broker image is `apache/kafka:4.1.1` (vanilla Apache, not Strimzi/Bitnami/Confluent). `kafka-configs.sh` lives at `/opt/kafka/bin/kafka-configs.sh`. PLAINTEXT listener is on `localhost:9092`.

Two independent confirmations:

1) Container env:
```
$ kubectl exec -n kafka kafka-broker-0 -- env | grep -i auto
KAFKA_AUTO_CREATE_TOPICS_ENABLE=true
```

2) Rendered broker config (`/opt/kafka/config/server.properties`):
```
$ kubectl exec -n kafka kafka-broker-0 -- grep -i auto.create /opt/kafka/config/server.properties
auto.create.topics.enable=true
```

`kafka-configs.sh --describe --entity-name 0` reported "broker '0' doesn't exist and doesn't have dynamic config" — confirmed via `node.id=1` in the same `server.properties`. The plan example's `--entity-name 0` should be parameterised on the actual broker/node id when used elsewhere; here the static property file is authoritative.

## Longhorn capacity for PR envs
- Per-env PVC footprint (sum of three PVC requests): 30 Gi (PVCs: atlas-data-pvc=10Gi, atlas-wz-input-pvc=10Gi, atlas-assets-pvc=10Gi)
- Longhorn free space: 167.48 Gi usable (502.44 Gi raw across 4 nodes ÷ default-replica-count=3)
- Soft cap on concurrent PR envs: floor(167.48 / 30) = 5
- StorageClass reclaimPolicy: Delete

### Evidence
PVC sizes (`kubectl get pvc -n atlas atlas-data-pvc atlas-wz-input-pvc atlas-assets-pvc -o custom-columns=NAME:.metadata.name,REQUEST:.spec.resources.requests.storage,USED:.status.capacity.storage`):

```
NAME                 REQUEST   USED
atlas-data-pvc       10Gi      10Gi
atlas-wz-input-pvc   10Gi      10Gi
atlas-assets-pvc     10Gi      10Gi
```

Per-node Longhorn free space (`kubectl get nodes.longhorn.io -n longhorn-system -o 'custom-columns=NAME:.metadata.name,FREE:.status.diskStatus.*.storageAvailable'`), bytes:

```
NAME     FREE
eos      128240844800   # 119.43 GiB
gaia     229638144000   # 213.87 GiB
helios    88709529600   #  82.62 GiB
theia     92903833600   #  86.52 GiB
```

Total raw free = 539,492,352,000 B ≈ 502.44 GiB. Longhorn `default-replica-count` setting:

```
$ kubectl get setting -n longhorn-system default-replica-count -o jsonpath='{.value}'
3
```

Usable free = 502.44 / 3 ≈ 167.48 GiB. Per-env footprint = 3 × 10 GiB = 30 GiB. Soft cap = floor(167.48 / 30) = 5 concurrent PR envs.

StorageClass reclaimPolicy (`kubectl get storageclass longhorn -o jsonpath='{.reclaimPolicy}'`):

```
Delete
```

`Delete` is the expected value — when a PR namespace is torn down, the PVCs are deleted and Longhorn will reclaim their PVs automatically. The cleanup CronJob does NOT need to explicitly delete orphaned PVs.

## MetalLB pool capacity
- Pool ranges: `192.168.23.230-192.168.23.250` (single IPAddressPool `cluster-pool` in `metallb-system`, `autoAssign: false`); pool size = 250 − 230 + 1 = 21 IPs
- Allocated IPs: 5 (reserved: .230 traefik, .231 atlas-login, .232 atlas-channel, .235 nginx-proxy-manager, .237 tempo-collector)
- Free IPs: 21 − 5 = 16
- Soft cap on concurrent PR envs (game-socket): 16 (each PR env claims one LB IP backing both its atlas-login and atlas-channel Services)
- If main becomes symmetric (atlas-main namespace), atlas-login-lb and atlas-channel-lb keep their existing .231/.232 reservations during cutover; PR envs draw from the remaining pool.

### Evidence
IPAddressPool definition (`kubectl get ipaddresspools.metallb.io -A -o yaml`):

```
spec:
  addresses:
  - 192.168.23.230-192.168.23.250
  autoAssign: false
  avoidBuggyIPs: false
```

Allocated LoadBalancer external IPs (`kubectl get svc -A -o wide | awk '/LoadBalancer/ {print $5}' | sort -u`):

```
192.168.23.230
192.168.23.231
192.168.23.232
192.168.23.235
192.168.23.237
```

Mapping (namespace / name / IP):

```
atlas               atlas-channel-lb         192.168.23.232
atlas               atlas-login-lb           192.168.23.231
kube-system         traefik                  192.168.23.230
nginx-proxy-manager nginx-proxy-manager-lb   192.168.23.235
observability       tempo-collector-lb       192.168.23.237
```

No services were in `<pending>` state at survey time.

### Binding constraint
Task 0.4 set the Longhorn-derived soft cap at 5 concurrent PR envs. MetalLB allows 16. The smaller of the two (Longhorn = 5) is the binding constraint on concurrent PR envs; MetalLB has comfortable headroom.

## Longhorn RecurringJobs and PR PVC exclusion
- BackupTarget: `nfs://nas.home:/volume1/LonghornBackup` (BackupTarget CR named `default`, pollInterval 5m)
- RecurringJobs: `backup-daily` (cron `0 2 * * *`, retain 7, concurrency 2) and `backup-weekly` (cron `0 3 * * 0`, retain 4, concurrency 2); both target group `default` only, task `backup`
- PR PVC exclusion mechanism: dedicated StorageClass `longhorn-pr` with `parameters.recurringJobSelector: '[]'` (empty JSON array) — referenced from PR-overlay PVCs via `spec.storageClassName`. Rationale below; see "Mechanism rationale" for why label-on-PVC was rejected.
- Longhorn version: `longhornio/longhorn-manager:v1.9.1` (Helm chart `longhorn-105.3.0_up1.9.1`); `longhorn-manager` is a DaemonSet, not a Deployment — `kubectl get deployment` returns NotFound, must use `kubectl get daemonset`.

### Evidence

RecurringJob CRD shape (`kubectl get recurringjob -n longhorn-system -o yaml`, two items, abridged):
```
apiVersion: longhorn.io/v1beta2
kind: RecurringJob
metadata: {name: backup-daily, namespace: longhorn-system}
spec:
  concurrency: 2
  cron: 0 2 * * *
  groups: [default]
  retain: 7
  task: backup
---
apiVersion: longhorn.io/v1beta2
kind: RecurringJob
metadata: {name: backup-weekly, namespace: longhorn-system}
spec:
  concurrency: 2
  cron: 0 3 * * 0
  groups: [default]
  retain: 4
  task: backup
```

Sample PV labels (`kubectl get pv pvc-b6afea0d-1d86-41ff-90d8-b9327a36bb39 -o yaml`, the PV bound to `atlas/atlas-data-pvc`): the K8s `PersistentVolume` carries **no labels** (`labels: {}` in the applied configuration; no `labels:` block in `metadata`). The `recurringjob.longhorn.io/source: enabled` label the plan expected on the PV is **not present on this cluster** — group membership is tracked on the Longhorn `Volume` CR in `longhorn-system`, not on the K8s PV.

Sample Longhorn Volume CR labels (`kubectl get volume -n longhorn-system pvc-b6afea0d-1d86-41ff-90d8-b9327a36bb39 -o jsonpath='{.metadata.labels}'`):
```json
{
  "backup-target": "default",
  "longhornvolume": "pvc-b6afea0d-1d86-41ff-90d8-b9327a36bb39",
  "recurring-job-group.longhorn.io/default": "enabled",
  "setting.longhorn.io/remove-snapshots-during-filesystem-trim": "ignored",
  "setting.longhorn.io/replica-auto-balance": "ignored",
  "setting.longhorn.io/snapshot-data-integrity": "ignored"
}
```

Spot-checked all 19 Longhorn Volumes in the cluster: every one has `recurring-job-group.longhorn.io/default: enabled`. None deviate. The label is auto-added by Longhorn during dynamic provisioning (volumes have no `kubectl.kubernetes.io/last-applied-configuration` annotation, so this isn't from `kubectl apply`).

Longhorn version (`kubectl get daemonset -n longhorn-system longhorn-manager -o jsonpath='{.spec.template.spec.containers[0].image}'`):
```
longhornio/longhorn-manager:v1.9.1
```

### Mechanism rationale

Longhorn 1.9.1 supports three exclusion mechanisms (per docs):
1. Set `recurring-job-group.longhorn.io/<group>: ""` (empty value) on the Longhorn Volume CR.
2. Set the per-job label `recurring-job.longhorn.io/<job-name>: ""` on the Volume CR.
3. Use a separate StorageClass whose `recurringJobSelector` parameter excludes the default group.

Options 1 and 2 require **post-creation patching** of each Longhorn Volume CR (a separate API object in `longhorn-system`, not the K8s PVC). The PR overlay would need a sync hook or operator that watches PVCs created in the PR namespace, looks up their corresponding Longhorn Volume by name (`pvc-<uid>`), and patches its labels. This is fragile (race vs. recurring-job controller's first reconcile) and adds a controller dependency.

Option 3 — dedicated StorageClass — is fully declarative: the PR overlay defines a `StorageClass longhorn-pr` (one-time, cluster-scoped) with `parameters.recurringJobSelector: '[]'`. PR-overlay PVCs reference it via `spec.storageClassName: longhorn-pr`. Longhorn applies the empty selector at provision time, so the Volume CR is never tagged into the `default` group. No race, no controller, no patching.

Trade-off: a cluster-scoped StorageClass must be created once outside the per-PR overlay (e.g., installed as part of the env-agent's bootstrap or hand-applied). It is not destroyed when a PR is torn down. This is acceptable: the StorageClass costs nothing when unused, and reusing it across all PR envs eliminates per-PR drift.

### Open caveat for downstream tasks

The plan's Step 2 example (`grep -A20 'labels:'` on the PV) assumes the K8s PV carries the recurring-job labels. **It does not on this cluster.** Any task that walks PVs by recurring-job label to identify which volumes will be backed up must instead query Longhorn `Volume` CRs in `longhorn-system`. The PR cleanup and capacity-counting tasks should treat this as the source of truth.

## Canonical Tenant + Service configs for bootstrap
- Captured from live main env on 2026-05-08
- Tenant payload: docs/tasks/task-063-ephemeral-pr-deployments/canonical-tenant.json (`.data.id` stripped; bootstrap mints a fresh UUID per PR)
- Service configs: docs/tasks/task-063-ephemeral-pr-deployments/canonical-services/{login,channel,drops}-service.json (ids preserved; SERVICE_ID env vars are pinned)
- Service UUIDs (deployed): login=`e7fb1d7e-47b8-46bd-97dc-867d93530856`, channel=`e7fb1d7e-47b8-46bd-97dc-867d93530000`, drops=`00000000-0000-0000-0000-000000000000`
- Note: atlas-character-factory and atlas-world also read `SERVICE_ID=00000000-0000-0000-0000-000000000000`; the drops-service config record satisfies all three.

### Endpoints used (in-pod, port-forward equivalents)
- Tenant list:    `GET http://localhost:8080/api/tenants` (in `deploy/atlas-tenants`)
- Tenant detail:  `GET http://localhost:8080/api/tenants/{tenantId}` (in `deploy/atlas-tenants`)
- Service config: `GET http://localhost:8080/api/configurations/services/{serviceId}` (in `deploy/atlas-configurations`)

All four GETs returned HTTP 200 with `application/vnd.api+json`. No auth header was required against either Deployment's in-pod localhost endpoint.

### Top-level shape per artefact
- `canonical-tenant.json`: `data.type=tenants`, `data.attributes={name, region, majorVersion, minorVersion}`. World/channel topology is **not** stored on the Tenant — it lives inside the channel-service config under `tenants[].worlds[].channels[]` (matches `docs/onboarding.md` §"Step 2").
- `login-service.json`: `data.type=services`, `data.attributes={type:"login-service", tasks:[{type:"timeout",...}], tenants:[{id, port}]}`.
- `channel-service.json`: `data.type=services`, `data.attributes={type:"channel-service", tasks:[{type:"timeout",...}], tenants:[{id, ipAddress, worlds:[{id, channels:[{id, port}]}]}]}`. Note `ipAddress=192.168.23.232` (atlas-channel-lb MetalLB VIP from §"MetalLB pool capacity") — the bootstrap will need to substitute the per-PR LB IP rather than reuse this literal.
- `drops-service.json`: `data.type=services`, `data.attributes={type:"drops-service", tasks:[{type:"drop_expiration_task",...}]}`. **No `tenants` array** — drops-service is tenant-agnostic at the config layer.

### Tooling note
The plan's `curl -fsS` command fails inside both `atlas-tenants` and `atlas-configurations` containers (Alpine images without curl). The captures used busybox `wget -qO- --header='Accept: application/vnd.api+json'` instead. The bootstrap should not assume curl is present in either container; if the env-agent reads/writes via in-pod exec, prefer wget or run curl from a sidecar/jobpod.
