# Task 165: Mist (AffectedArea) writer rollout procedure

This document is the reproducible procedure for enabling the `AffectedAreaCreated`
and `AffectedAreaRemoved` clientbound writers on **existing** live tenants, after
this branch wired both opcodes into the seed templates for all eleven supported
client versions.

It is written for the operator executing the rollout in a given environment
(Task 11). Follow Section 3 as a full sweep — every tenant on every wired
version, not a spot-check — and fill in Section 4 as you go.

## 1. Why a manual patch is required

Seed templates (`services/atlas-configurations/seed-data/templates/*.json`) are
only consulted when a **new** tenant is created — atlas-configurations copies the
template's `socket.writers` (and `socket.handlers`) array into the tenant's
stored configuration at creation time. Tenants that already existed before this
branch merged do **not** pick up the new writer entries automatically; their
stored configuration is frozen at whatever it contained when they were created
or last patched.

This is the same class of bug documented in `bug_new_opcodes_not_in_live_tenant_config.md`:
a new opcode present only in the seed template is silently absent from a live
tenant's configuration, and any packet for it is dropped with a "no opcode
mapping" / "writer not found" warning at send time — not a crash, so it can go
unnoticed until someone triggers the feature.

Additionally, atlas-channel's configuration projection does **not** hot-reload
writer maps: it resolves the tenant's writer table into an in-memory dispatch
map at process start (and on the configuration-status event that triggers a
reload), so even after the tenant's stored configuration is patched, a running
atlas-channel instance will keep dispatching against its stale in-memory map
until it is restarted.

Net effect: for every existing tenant on a wired version, this rollout requires
(a) directly PATCHing the tenant's stored configuration to add the two writer
entries, and (b) restarting atlas-channel afterward.

## 2. Per-version writer entries (wired set)

Discovery on this branch wired **all eleven** supported versions — no version
was left unwired. The `services`/`writer`/`opCode` values below are copied
**verbatim** from each seed template under
`services/atlas-configurations/seed-data/templates/`; hex-literal width
(2 vs 3 digits) matches the source file exactly — do not re-pad or re-normalize
when copy-pasting into a PATCH body.

| Template file | region | majorVersion | `AffectedAreaCreated` entry | `AffectedAreaRemoved` entry |
|---|---|---|---|---|
| `template_gms_12_1.json` | GMS | 12 | `{"opCode": "0x8F", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0x90", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |
| `template_gms_48_1.json` | GMS | 48 | `{"opCode": "0xCA", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0xCB", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |
| `template_gms_61_1.json` | GMS | 61 | `{"opCode": "0xD2", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0xD3", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |
| `template_gms_72_1.json` | GMS | 72 | `{"opCode": "0xF3", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0xF4", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |
| `template_gms_79_1.json` | GMS | 79 | `{"opCode": "0xFB", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0xFC", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |
| `template_gms_83_1.json` | GMS | 83 | `{"opCode": "0x111", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0x112", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |
| `template_gms_84_1.json` | GMS | 84 | `{"opCode": "0x118", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0x119", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |
| `template_gms_87_1.json` | GMS | 87 | `{"opCode": "0x122", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0x123", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |
| `template_gms_92_1.json` | GMS | 92 | `{"opCode": "0x140", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0x141", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |
| `template_gms_95_1.json` | GMS | 95 | `{"opCode": "0x148", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0x149", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |
| `template_jms_185_1.json` | JMS | 185 | `{"opCode": "0x126", "writer": "AffectedAreaCreated", "services": ["channel"]}` | `{"opCode": "0x127", "writer": "AffectedAreaRemoved", "services": ["channel"]}` |

No version was left unwired by this branch — every (region, majorVersion) pair
that atlas-configurations currently ships a seed template for now has both
entries. There is nothing to exclude from the sweep below.

## 3. Procedure

Run this once per environment. `<configurations-service-base>` is the
in-cluster or externally reachable base URL for atlas-configurations in that
environment; `<namespace>` is the Kubernetes namespace atlas-channel runs in
for that environment.

### 3.1 Enumerate tenant configurations

```
GET <configurations-service-base>/api/configurations/tenants
```

Paginate through the full result set (the endpoint is paginated — follow
`page[number]`/`page[size]` until the response is exhausted). For each tenant
record, read its `region` and `majorVersion` attributes.

### 3.2 Select tenants in scope

Select **every** tenant whose `(region, majorVersion)` pair matches a row in
Section 2's table. This must be a full sweep of all matching tenants in the
environment, not a sample.

### 3.3 Per tenant: read-modify-write with idempotency guard

For each selected tenant:

1. `GET <configurations-service-base>/api/configurations/tenants/<tenantId>`
   and inspect `attributes.socket.writers`.
2. **Idempotency guard**: if an entry with `"writer": "AffectedAreaCreated"`
   is already present in `socket.writers`, skip this tenant entirely (do not
   PATCH, do not duplicate the entry) — it has already been rolled out or was
   created after this branch's seed templates took effect. Record it as
   already-current in Section 4 and move on.
3. Otherwise, append **both** the `AffectedAreaCreated` and
   `AffectedAreaRemoved` entries for this tenant's `majorVersion` (copied
   verbatim from Section 2) to `attributes.socket.writers`, keeping every
   existing entry in the array unchanged.
4. `PATCH <configurations-service-base>/api/configurations/tenants/<tenantId>`
   with the full updated tenant body, wrapped in the mandatory JSON:API
   envelope:

   ```json
   {
     "data": {
       "type": "tenants",
       "id": "<tenantId>",
       "attributes": {
         "...": "...full tenant attributes, including the updated socket.writers array..."
       }
     }
   }
   ```

   This PATCH route is registered through `RegisterInputHandler`, which
   unmarshals the request body as JSON:API directly — there is no adapter that
   unwraps a bare-attributes body. A request body that omits the
   `{"data": {"type": "tenants", "id": ..., "attributes": {...}}}` envelope is
   silently rejected (fails to unmarshal); it is not optional.

### 3.4 Restart atlas-channel

After all tenants for the environment have been patched (or confirmed
already-current):

```
kubectl -n <namespace> rollout restart deployment atlas-channel
kubectl -n <namespace> rollout status deployment atlas-channel
```

This is required because atlas-channel's configuration projection resolves the
writer dispatch map once and does not hot-reload it; without a restart the
process keeps using its pre-patch writer map even though the stored tenant
configuration is now correct.

### 3.5 Verify per tenant

For each patched tenant, after the restart completes:

1. Confirm atlas-channel's startup logs no longer contain a "no opcode
   mapping" warning that names either `AffectedAreaCreated` or
   `AffectedAreaRemoved` for that tenant.
2. Trigger mist activity for that tenant (or wait for naturally occurring mist
   activity — mob mist-drop skills, mist-producing reactors, etc.) and confirm
   no `writer not found` or `Unable to broadcast AffectedArea` log lines
   appear for that tenant afterward.
3. Record the result in Section 5.

## 4. Mist client-side lifetime

The `AffectedAreaCreated` packet carries **no duration**, on any version. The
client holds a mob mist until the server sends `AffectedAreaRemoved`, which
atlas-maps emits when the mist expires server-side. See `discovery.md`
section "Semantics of the trailing `+0x30` slot" for the record layout this
rests on.

An earlier revision of this branch mistook the packet's `Decode2` field for a
lifetime and set it to `Duration / 100`. That field is a **delay before the
client draws the mist** (`tStart = get_update_time() + 100 * skillDelay`, gated
in `CAffectedAreaPool::Update`), so the mist stayed invisible for its whole
duration and was then removed at roughly the instant it first appeared —
observed in `atlas-pr-1226` as a sub-second flash. It is now sent as 0 (draw
immediately), and the trailing word — really `nElemAttr`, not a time — is 0 too.

When verifying this rollout (Section 3.5), expect the mist to appear
**immediately** on cast and to persist until its server-side duration elapses.
A mist that takes visible time to appear, or that flashes and vanishes, means
the draw-delay regression is back.


## 5. Environment record

Fill in one row per tenant as the sweep executes.

| Environment | Tenant id | Region/version | Patched (date) | Restarted | Verified |
|---|---|---|---|---|---|
| `<environment>` | `<tenantId>` | `<region>/<majorVersion>` | `<yyyy-mm-dd>` | `<yes/no>` | `<yes/no>` |
