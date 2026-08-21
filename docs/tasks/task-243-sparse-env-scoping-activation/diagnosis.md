# Bug: every message a sparse environment produces is dropped by the ownership gate

> **Read this first — two claims below were disproved by later evidence in the
> same investigation. They are left in place because the reasoning chain is
> what led to the real cause, but do not act on them:**
>
> 1. **"Cause 3: the service-status projection has no environment scoping" is
>    WRONG.** The projection *is* scoped, by service id:
>    `services/atlas-login/.../configuration/projection/subscriber.go:112`
>    does `if env.Id != s.ServiceId.String() { return true, nil }`. That is a
>    strictly stronger key than environment. The missing `ENVIRONMENT` header
>    on the service-status topic is real but not the cause of anything here.
> 2. **"Fault 5: control-plane deletes emit no tombstone" is WRONG.**
>    `services.ProcessorImpl.DeleteById`
>    (`services/atlas-configurations/.../services/processor.go:228`) enqueues
>    a nil-value tombstone explicitly for the compacted topic. The row that
>    replayed forever did so because *this investigation* deleted it with raw
>    SQL, bypassing the API. That was an operator error, not a product defect.
>
> **The actual root cause** — established after both corrections — is that an
> override Deployment cannot durably reference its own service-config row.
> `deploy/k8s/base/atlas-login.yaml:45` bakes the baseline's pinned
> `SERVICE_ID`; `bootstrap.sh` compensates with an out-of-band
> `kubectl set env SERVICE_ID=…`; and the Argo CD Application has
> `selfHeal: true`, which reverts it. Verified in the live cluster:
> `atlas-login` in `atlas-pr-1411` runs with
> `SERVICE_ID=e7fb1d7e-47b8-46bd-97dc-867d93530856`, which the database
> confirms is `main`'s `login-service` row — not either of `pr-1411`'s own
> two rows. That is why it binds main's ten tenants, including `ec876921` on
> port 8300, and why every message it emits pairs `ENVIRONMENT: pr-1411` with
> a tenant main owns.
>
> The five duplicate `login-service` rows are the same defect's debris:
> `service-config.sh:117-127` mints a fresh random `new_uuid()` per run, POSTs
> it, patches the Deployment, and Argo reverts the patch — once per bootstrap
> run.
>
> See `prd.md` in this folder for the scoped work that follows from this.

Found during the live test of `atlas-pr-1411` on 2026-08-20, after the
substrate-naming fix (PR #1432) got `atlas-login` healthy. Neither cause below
is a regression from that fix — these paths had simply never been reached,
because login crash-looped ahead of them.

Two independent faults, both required for the flow to work.

## Symptom

Client connects to the pr-1411 login LoadBalancer, submits credentials, and
the login screen hangs forever. No error surfaces to the client.

## Evidence

`atlas-login` (atlas-pr-1411) writes the command and then waits:

```
12:45:29.860  [LoginHandle] read [name [Atlas], password [...], ...]
12:45:29.860  Created kafka writer for topic [COMMAND_TOPIC_ACCOUNT_SESSION-main].
...
(EVENT_TOPIC_ACCOUNT_SESSION_STATUS-main stays idle indefinitely)
```

The message reached Kafka — offset 795 on `COMMAND_TOPIC_ACCOUNT_SESSION-main`:

```
CreateTime:1787229929919 {"sessionId":"bc030f70-ec7c-4968-a286-30272e973b2d",
  "accountId":0,"author":"LOGIN","type":"CREATE",
  "body":{"accountName":"Atlas", ... ,"ipAddress":"10.42.7.0", ...}}
```

The baseline's `atlas-account` (atlas-main, pod `...-nv2vk`) consumed it —
group `Account Service` at 796/796, lag 0 — and threw it away:

```
12:45:29.925  Message received {"sessionId":"bc030f70-...","type":"CREATE",...}
              originator=COMMAND_TOPIC_ACCOUNT_SESSION-main
12:45:29.926  Dropping message: environment is unresolvable. No deployment will process it.
              environment=pr-1411  topic=COMMAND_TOPIC_ACCOUNT_SESSION-main  log.level=error
```

Both drops below come from `decide()` in
`libs/atlas-kafka/consumer/gate.go`, which returns `gateDropUnresolvable`
from three separate arms. They are indistinguishable in the log — the message
text is identical for all three.

---

## Cause 1: a sparse environment is never promoted out of PROVISIONING

`gate.go`:

```go
if !r.IsActive(msgEnv) {
    return gateDropUnresolvable // FR-4.7 / D4
}
```

`IsActive` (`libs/atlas-env/registry.go:178`) requires `PhaseActive` exactly.
The pr-1411 record was `PROVISIONING`:

```json
{"name":"pr-1411","baseline":"main","namespace":"atlas-pr-1411",
 "tenant":"16f32d30-f2fb-408e-998c-571a4e85878c",
 "overrides":{"atlas-channel":"atlas-pr-1411","atlas-login":"atlas-pr-1411"},
 "phase":"PROVISIONING"}
```

**Nothing in the repository ever promotes it.** The only producer of an
`ACTIVE` phase outside tests and docs is
`deploy/k8s/overlays/main/environment-record.yaml:82` — the baseline creating
*itself*. Confirmed by sweep over `deploy/`, `services/`, `libs/`,
`.github/`, `tools/`.

The design names an owner that was never built:

- `deploy/k8s/overlays/pr-sparse/environment-record.yaml:1-6` — "It is created
  in PROVISIONING … Task 45's offset seeding and the override rollout happen
  next; **the phase is flipped to ACTIVE last**."
- `services/atlas-pr-bootstrap/scripts/bootstrap.sh:109-114` — bootstrap
  deliberately re-PATCHes the *current* phase and "must not promote it".
- PRD **FR-5.3**: transition to `ACTIVE` requires every override Deployment
  ready, every override consumer group initialized (FR-4.9), and the
  mandatory socket services (D6) bound.
- plan.md Task 45 Step 4 ("the activation gate (FR-5.3) needs an
  **observable** signal that the group is initialized") is the closest owner,
  and Task 45 is **unimplemented**: every step is `- [ ]`, and
  `deploy/k8s/base/atlas-kafka-precreate.yaml` contains zero occurrences of
  `seed_group` — though `deploy/k8s/base/atlas-kafka-precreate_test.sh` exists
  (Step 1 landed, Step 3 did not).

Why it stayed invisible: FR-5.2 says a PROVISIONING environment's ingress does
not route, so in the designed flow no pr-1411 traffic exists yet. The live
test bypassed that by connecting straight to the pr-1411 login LoadBalancer
(192.168.23.191).

Ruled out as a co-factor: the FR-7.7 tenant/environment mismatch arm.
`Reconcile` (`libs/atlas-env/tenants.go:58`) only errors when the header's
environment disagrees with the environment owning the tenant; main's record
carries `tenant: ""`, so the client's tenant
`ec876921-c363-4cc6-9c51-5bb8d57f9553` resolves to no environment and the
header is trusted.

---

## Cause 2 (dominant): the baseline's environment heartbeat is dead

PATCHing the record to `ACTIVE` by hand (HTTP 200; the baseline's account
projected the new phase at 12:52:26) did **not** unblock the flow. A retry at
12:59:32 produced the identical drop:

```
12:59:32.029  Message received {"sessionId":"771f139e-...","type":"CREATE",...}
12:59:32.030  Dropping message: environment is unresolvable. No deployment will process it.
              environment=pr-1411
```

With the record ACTIVE, the surviving arm is the staleness one
(`gate.go:59`):

```go
if r.Stale() && msgEnv != self {
    return gateDropUnresolvable
}
```

`Stale()` is `now - lastSeen > StaleAfter`, `StaleAfter = 120s`
(`libs/atlas-env/registry.go:10`), and `lastSeen` advances only when a message
arrives on `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main`. Every pod logs
that topic as idle; its high-water mark is **21**.

The liveness source is `environments.StartHeartbeat`
(`services/atlas-configurations/atlas.com/configurations/environments/heartbeat.go`),
wired at `main.go:77`, republishing `p.Republish(envlib.Self())` every 30s. It
fails on every tick:

```
12:58:34  environment heartbeat failed  error="record not found"
12:59:04  environment heartbeat failed  error="record not found"
12:59:34  environment heartbeat failed  error="record not found"
```

`Republish` → `GetByName(string(id))` → `byNameEntityProvider`, a plain
`WHERE name = ?` with no environment scoping. It reports "record not found"
because the id is **empty**: `envlib.Self()` reads `ATLAS_ENVIRONMENT`
(`libs/atlas-env/env.go:80`, `SelfVar`), and that variable is not set in the
running pod:

```
$ kubectl exec -n atlas-main deploy/atlas-configurations -- printenv ATLAS_ENVIRONMENT
(exit 1)
$ kubectl exec -n atlas-main deploy/atlas-account        -- printenv ATLAS_ENVIRONMENT
main
```

The `atlas-env` ConfigMap in `atlas-main` *does* carry `ATLAS_ENVIRONMENT:
main`. The key was added to `deploy/k8s/overlays/main/kustomization.yaml:48`
by **e786ac8ea (#1427, 2026-08-20)**. `envFrom` is resolved once at pod
creation, so only pods restarted after that commit have it:

| Deployment | pod start | `ATLAS_ENVIRONMENT` |
|---|---|---|
| `atlas-account` | 2026-08-20T12:08:40Z | `main` |
| `atlas-configurations` | 2026-08-19T19:42:19Z | unset |
| `atlas-ban` | 2026-08-19T19:42:40Z | unset |
| `atlas-buddies` | 2026-08-19T19:42:41Z | unset |

**47 of 130 pods in `atlas-main`, across 26 deployments, started before
2026-08-20 and are running with `env.Self() == ""`.**

### Why this is worse than one dead heartbeat

A pod with `Self() == ""` fails `IsOwner` for every non-legacy environment
(`libs/atlas-env/registry.go:218`):

```go
if _, overridden := rec.Overrides[service]; overridden {
    return e == r.self
}
return rec.Baseline == r.self     // "main" == ""  -> false
```

So even with the heartbeat restored, a pr-1411 message routed to one of those
26 services returns `gateSkipNotOwner` — which, unlike the unresolvable path,
is **silent**: no log line, only the
`atlas_kafka_gate_skipped_not_owner_total` counter. Nobody processes the
message and nothing says so. The login path happens to be clear only because
`atlas-account` restarted this morning.

---

## Remediation

Immediate (live state):

1. `kubectl rollout restart deploy/atlas-configurations -n atlas-main` —
   restores the heartbeat, un-staling every consumer's registry.
2. Roll the other 25 stale `atlas-main` deployments, or accept that sparse
   traffic reaching them is silently skipped.
3. pr-1411's record was PATCHed to `ACTIVE` by hand. This is live state only;
   nothing in the repo reproduces it for the next environment.

Code:

4. Implement the activation transition (Cause 1). Open questions before that
   can land: who performs it (a post-sync Job in `overlays/pr-sparse` is the
   obvious home, but FR-5.3 also demands observably-seeded consumer groups,
   which is unimplemented Task 45); and FR-5.4 atomicity, since registry
   caches update independently per pod (PRD open question 3).
5. Decide whether PROVISIONING should really drop. The overlay's own comment
   says that during PROVISIONING "baseline deployments continue to own the
   environment's services" — but `decide()` never reaches `IsOwner` for a
   non-ACTIVE environment, so the baseline does *not* own it and the message
   is discarded. Either the comment or the gate is wrong.
6. Fix the ConfigMap-propagation defect. Changing a key in `atlas-env` does
   not roll the Deployments consuming it via `envFrom` — no name-suffix hash,
   no checksum annotation on the pod templates. A semantically breaking
   control-plane change (#1427 adding `ATLAS_ENVIRONMENT`, which the entire
   ownership gate depends on) reached zero running pods and produced no
   failure signal anywhere except a `warning`-level heartbeat log. This is
   independent of task-232.

## Separate observation (not a cause of this hang)

pr-1411's environment record names tenant
`16f32d30-f2fb-408e-998c-571a4e85878c`, but the client connected and played as
`ec876921-c363-4cc6-9c51-5bb8d57f9553` — the shared control plane's existing
tenant. `cleanup.sh`'s sweep-tenant phase reads only the record's tenant, so
anything written under `ec876921` would not be reclaimed at teardown.
Possibly task-242 (sparse tenant provisioning) territory; noted so it is not
lost.

---

# Cause 3 (final, architectural): the service-status projection has no environment scoping

Clearing Causes 1 and 2 in live state did **not** fix the hang. A retry at
13:09:47 dropped identically. With the record ACTIVE and the registry fresh,
the only surviving arm of `decide()` is FR-7.7 `mismatched`.

## Why the message mismatches

`libs/atlas-env/tenants.go:58` `Reconcile`:

```go
tenantEnv, known := tr.EnvironmentOfTenant(tenantId)
if !known      { return headerEnv, nil }
if headerEnv == "" { return tenantEnv, nil }
if headerEnv != tenantEnv {
    return "", fmt.Errorf("%w: ...", ErrEnvironmentMismatch)   // -> drop
}
```

The client is served as tenant `ec876921-c363-4cc6-9c51-5bb8d57f9553`, which
is **main's** GMS 83.1 tenant:

```
       id                                 | environment | region | major | minor
 ec876921-c363-4cc6-9c51-5bb8d57f9553     | main        | GMS    |    83 |     1
 16f32d30-f2fb-408e-998c-571a4e85878c     | pr-1411     | GMS    |    83 |     1
```

So pr-1411's login emits `ENVIRONMENT: pr-1411` + tenant `ec876921` →
`Reconcile` mismatches → dropped.

Note the tenant-status events for pre-#1427 tenants carry **no** `environment`
attribute at all, and `MapRegistry.ApplyTenant` (`registry.go:67`) stores
unconditionally:

```go
func (r *MapRegistry) ApplyTenant(tenantId string, e Id) {
    r.tenants[tenantId] = e     // e == "" for a legacy tenant
```

so a legacy tenant is **known with environment `""`**, and every
environment-tagged message naming it is a hard mismatch rather than a
trusted-header legacy case. Everywhere else in this codebase `""` means
"legacy, don't filter" (FR-1.8); here it means "definitely not yours". That
asymmetry is worth a decision on its own.

## Why login is serving main's tenant

pr-1411's `services` rows had accumulated across syncs — 21 rows, of which 10
named main's tenant and the rest its own, both claiming port 8300:

```
      type       | names_main_tenant | count
 channel-service | f                 |     2
 channel-service | t                 |     5
 drops-service   | f                 |     7
 login-service   | f                 |     2
 login-service   | t                 |     5
```

Deleting the 10 wrong rows (`DELETE 10`, `environment='pr-1411'` only; main's
19 rows untouched) and restarting login **did not help** — port 8300 still
bound `ec876921`:

```
13:13:47.400  Creating login socket service for [GMS] [83.1] on port [8300].
              tenant=ec876921-c363-4cc6-9c51-5bb8d57f9553
```

Two reasons, both structural:

1. **The projection is topic-driven, not table-driven.** A SQL delete emits no
   tombstone, so every deleted row is still replayed from the compacted
   `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS-main` on the next restart. Any
   control-plane row removed without going through the API that writes a
   tombstone is immortal to consumers.

2. **The service-status topic carries no `ENVIRONMENT` header at all** —
   verified with `print.headers=true` across the whole topic, including rows
   written today:

   ```
   HEADERS=[NO_HEADERS] KEY=service:ea44a548-...   (main, tenant ec876921)
   HEADERS=[NO_HEADERS] KEY=service:efff85da-...   (pr-1411, tenant 16f32d30)
   ```

   With no header, `msgEnv == ""` and the gate returns `gateProcess` (FR-1.8)
   for **every** consumer. So pr-1411's login projects main's service rows as
   its own. It is currently listening for main's GMS 79.1 tenant on 7900,
   main's GMS 83.1 tenant on 8300, and so on — and since both environments'
   83.1 tenants claim port 8300, the winner is whichever event applied last.

Deleting rows cannot fix this: main's own legitimate row binds port 8300 to
`ec876921`, and pr-1411's login has no basis to reject it.

## Assessment

This is where symptom-chasing stops. Four live fixes, each revealing a new
defect in a different place — the signature of an architectural gap, not a
sequence of bugs:

| # | Fault | Fixed live? |
|---|---|---|
| 1 | Nothing promotes a sparse env to ACTIVE | hand-PATCHed |
| 2 | `ATLAS_ENVIRONMENT` absent in 26 baseline deployments → dead heartbeat → every registry permanently stale | 25 rollout-restarts |
| 3 | Legacy tenants project as environment `""`, which `Reconcile` treats as a hard mismatch rather than legacy | no |
| 4 | Sparse env's service rows duplicate per sync and named the baseline's tenant | 10 rows deleted — ineffective |
| 5 | Control-plane deletes emit no tombstone; compacted topic replays them forever | no |
| 6 | Service-status events carry no `ENVIRONMENT` header, so every environment projects every environment's service rows, including port bindings | no |

Faults 3, 5, and 6 are design-level and none has an owning task. Sparse
ephemeral environments cannot complete a login until at least 6 is resolved:
an override's socket-binding table must be scoped to the environments it owns.
