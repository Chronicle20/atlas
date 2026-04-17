# Atlas Onboarding

Runbook for bringing a fresh Atlas environment to a playable state. Applies to
both `deploy/compose/` (local) and `deploy/k8s/` (cluster) — the steps are the
same, only the URL of the Web UI changes.

## What to expect on first boot

Atlas services fetch their config at startup from `atlas-configurations`, via
the ingress. On a cold environment (empty DB), three things aren't there yet:

1. No **tenant** → every service that asks for `/api/configurations/tenants`
   exits with "tenant not configured".
2. No **per-service configs** → services with a hard-coded `SERVICE_ID` env
   var exit with "Could not retrieve configuration" (500 from configurations).
3. No **game content** (drops tables, NPC scripts, quest dialogue, etc.) →
   those services start but have nothing to serve.

So on a cold boot you should expect a handful of services to crashloop until
you walk through the steps below. This is normal — the compose stack is wired
so that configurations + tenants come up first (they're the only services
with healthchecks), everything else waits on them via `depends_on`, and a
crashlooping downstream service does **not** block anything else.

## Step 1 — Tenant

*Web UI → **Templates***. Clone a template into a tenant.

A tenant is a single logical game shard: a region (GMS/JMS/…) + version
(e.g. v83.1) + a bundle of scripts, socket handlers, worlds, cash shop, etc.
Cloning from a template populates all of that in one shot, both in
`atlas-tenants` (the record the rest of the system IDs by) and in
`atlas-configurations.tenants/{id}` (the runtime config payload).

After this step, every service that was waiting on tenant config will
transition to healthy on its next restart cycle — no manual bounce needed.

## Step 2 — Per-service configs

*Web UI → **Services** → Create Service*. Three entries are required. The
IDs must match the `SERVICE_ID` env vars baked into the compose/k8s manifests
**exactly**, otherwise the container keeps asking configurations for a UUID
that doesn't exist and 500s forever.

| Service | Required UUID | Type |
|---|---|---|
| atlas-drops | `00000000-0000-0000-0000-000000000000` | drops-service |
| atlas-login | `e7fb1d7e-47b8-46bd-97dc-867d93530856` | login-service |
| atlas-channel | `e7fb1d7e-47b8-46bd-97dc-867d93530000` | channel-service |

If the Create Service dialog generates a UUID and doesn't let you type one,
take the generated UUID and update `SERVICE_ID` in
`deploy/compose/docker-compose.{core,socket}.yml` (or the equivalent k8s
manifest) to match, then recreate just those three services.

Each service record needs the tenant it applies to:
- **login-service**: `tenants: [{id, port}]` (login listening port per tenant)
- **channel-service**: `tenants: [{id, ipAddress, worlds: [{id, channels: [{id, port}]}]}]`
- **drops-service**: minimal, just the tenant list

## Step 3 — Game content *(optional, but required for gameplay)*

*Web UI → **Setup***. Offers buttons for:
- Drops tables
- Gachapons
- NPC conversations
- Quest conversations
- NPC shops
- Portal scripts
- Reactor scripts
- "Upload Game Data" (raw `.zip` export fed to `atlas-wz-extractor`)

You can seed in any order. Seeding publishes data into the relevant
service's DB via its REST API, so the services themselves don't need to be
restarted afterwards.

## Verifying end-to-end

After Steps 1 and 2, every container should be `Up`. Quick check:

```
docker compose --env-file deploy/compose/.env --project-name atlas \
  -f deploy/compose/docker-compose.yml \
  -f deploy/compose/docker-compose.core.yml \
  -f deploy/compose/docker-compose.socket.yml \
  ps --format '{{.Status}}' | sort | uniq -c
```

Anything still in `Restarting` means either (a) a service-config UUID
mismatch (see Step 2) or (b) an external dependency — Postgres / Redis /
Kafka on the host — is unreachable. For Kafka specifically, the compose
stack expects a **PLAINTEXT** listener at `kafka.home:9092`; the default k3s
broker deployment advertises PLAINTEXT on 9092 and the in-cluster
`INTERNAL` listener on 9093. Containers outside the cluster must use 9092.

## Common pitfalls

- **Cloning a tenant hangs and the UI retries**, leaving duplicate tenant
  rows. Root cause is usually `atlas-tenants` failing to emit the
  `tenant.status` Kafka event (broker unreachable or wrong listener). The
  DB row is written **before** the Kafka emit, so each retry creates
  another tenant. Fix Kafka connectivity first, then delete the extras via
  `DELETE /api/tenants/{id}`.
- **Services in `/services` created with generated UUIDs**. The containers
  won't find them — the UUIDs are pinned by env var. Either enter the UUID
  explicitly in the dialog, or update `SERVICE_ID` in the manifest.
- **`/setup` seed buttons succeed but gameplay still shows missing data**.
  Seeding writes per-tenant; if you seed before creating the tenant, the
  data has nowhere to attach. Always do Step 1 first.
