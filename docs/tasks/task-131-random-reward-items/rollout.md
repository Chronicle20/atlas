# Runbook: reward-items data re-publish + live-tenant config rollout (task-131)

> **EXECUTION-GATED — RECORDED, NOT EXECUTED.** Do not run any step here until
> task-131 has merged AND the updated images are deployed to the target
> environment AND an operator has authorized the change for that environment.
> This document is authored as part of task-131 (design §5.7/§5.8); the
> implementation does not execute it.

## Why this is needed

task-131 adds two things that do not retroactively apply to already-provisioned
tenants:

1. **New atlas-data reward-node fields.** `consumable/reader.go` now parses
   per-entry `Effect`, `worldMsg`, and `period` off each `RewardRestModel` node
   (`reader.go:164-176`). These fields live inside the stored consumable JSON
   documents; existing rows were ingested before this change and simply don't
   have them.
2. **New serverbound handler opcode.** `CharacterItemUseLotteryHandle` is a new
   entry in `socket.handlers`. Seed templates
   (`services/atlas-configurations/seed-data/templates/template_*.json`) only
   apply at **tenant creation** — existing tenants never re-read the seed, so
   the handler is absent from their live socket config until it is patched in
   (project memory `bug_new_opcodes_not_in_live_tenant_config`).

Both gaps are safe to leave unpatched for a while (see "Blast radius" below) but
must be closed before the feature is usable end-to-end on existing tenants.

## Blast radius / what happens if you skip this

- **Skipping the data re-publish:** `Effect`/`worldMsg`/`period` default
  cleanly to `""`/`""`/`-1` when absent (`consumable/reader.go:169-170`,
  covered by `reader_test.go:1065-1071`). A reward grant on stale data still
  works — it just grants the item with no buff effect, no world announce, and
  no expiration. **Never a crash.**
- **Skipping the handler-opcode patch:** the client still sends the lottery-use
  opcode (it routes reward-node Consume items through
  `SendLotteryItemUseRequest` unconditionally — see design §2.1). With no
  matching entry in `socket.handlers`, atlas-channel logs "unhandled message
  op" and drops the packet; the player's reward box silently does nothing when
  double-clicked. This is the reproduction of
  `bug_new_opcodes_not_in_live_tenant_config` for this feature specifically.

Given the above, the data re-publish is lower urgency (graceful degradation)
than the handler-opcode patch (dead feature until patched). Both are documented
here; sequence them together per environment for a single clean cutover.

---

## Step 1: Data re-publish (Effect/worldMsg/period on existing reward rows)

atlas-data consumables are stored JSON documents keyed by region/version (and,
for non-canonical tenants, per-tenant overrides). Existing rows lack the new
fields until the item data is re-processed from WZ.

Perform this only if a canonical baseline is in play for the environment
(canonical ingest + baseline publish/restore is the standard atlas-data
provisioning path — see `services/atlas-data/atlas.com/data/baseline/`).

### 1.1 Re-ingest the canonical tenant

```bash
curl -s -X POST "$DATA_SERVICE_URL/api/data/process?scope=shared" \
  -H "X-Atlas-Operator: 1" \
  -H "TENANT_ID: $CANONICAL_TENANT_ID"
```

`scope=shared` re-ingests the canonical region/version data set (requires the
`X-Atlas-Operator: 1` header — `runtime/rest/resource.go:37-45`). This
re-parses Item.wz Consume/Install/Etc reward nodes through the updated reader
and picks up `Effect`/`worldMsg`/`period` on every reward entry.

Poll status if needed:

```bash
curl -s "$DATA_SERVICE_URL/api/data/process?scope=shared" \
  -H "TENANT_ID: $CANONICAL_TENANT_ID"
```

### 1.2 Publish the new canonical baseline

```bash
curl -s -X POST "$DATA_SERVICE_URL/api/data/baseline/publish" \
  -H "Content-Type: application/vnd.api+json" \
  -d '{
    "data": {
      "type": "baselinePublishes",
      "attributes": {
        "region": "GMS",
        "majorVersion": 83,
        "minorVersion": 1
      }
    }
  }'
```

Repeat per region/majorVersion/minorVersion baseline that needs the refreshed
canonical snapshot (`baseline/rest.go` `PublishInputModel`). This writes a new
canonical snapshot (with sha256) to the baseline store (MinIO).

### 1.3 Restore per live tenant

For tenants that read the canonical baseline **by copy** (not by fallback),
restore the freshly-published baseline onto each live tenant database:

```bash
curl -s -X POST "$DATA_SERVICE_URL/api/data/baseline/restore" \
  -H "Content-Type: application/vnd.api+json" \
  -d '{
    "data": {
      "type": "baselineRestores",
      "attributes": {
        "region": "GMS",
        "majorVersion": 83,
        "minorVersion": 1,
        "tenantId": "<tenant-uuid>"
      }
    }
  }'
```

Tenants on the **canonical-fallback read path** (`storage.go:44-60`) pick up
the new fields from the publish in step 1.2 alone — restore is not required
for those tenants, only for tenants with their own materialized per-tenant
copy. When unsure which category a tenant is in, restoring is always safe
(idempotent overwrite from the published baseline).

---

## Step 2: Handler opcode PATCH + atlas-channel restart

Seed templates only apply at tenant creation. For every **existing** v83, v84,
v87, and v95 tenant, PATCH the tenant's socket config to add the
`CharacterItemUseLotteryHandle` entry, then restart atlas-channel so it rebuilds
its in-memory handler map (config projection does not hot-reload
handlers/writers — project memory `bug_new_opcodes_not_in_live_tenant_config`).

### 2.1 Per-version opcode / entry to add

Add this element to `socket.handlers[]` — **`validator` is mandatory**;
`BuildHandlerMap` silently `continue`s (drops the entry with only a warning)
when the validator key doesn't resolve (`libs/atlas-opcodes/producer.go:47-50`,
channel `main.go:904-909`):

| version | opCode | handler | validator |
|---|---|---|---|
| v83 | `0x070` | `CharacterItemUseLotteryHandle` | `LoggedInValidator` |
| v84 | `0x070` | `CharacterItemUseLotteryHandle` | `LoggedInValidator` |
| v87 | `0x073` | `CharacterItemUseLotteryHandle` | `LoggedInValidator` |
| v95 | `0x07C` | `CharacterItemUseLotteryHandle` | `LoggedInValidator` |

```json
{ "opCode": "0x070", "validator": "LoggedInValidator", "handler": "CharacterItemUseLotteryHandle" }
```

(substitute the per-version `opCode` from the table above)

### 2.2 Enumerate live tenants and their versions

```bash
curl -s "$CONFIG_URL/configurations/tenants" | jq -r \
  '.data[] | "\(.id)\t\(.attributes.region) v\(.attributes.majorVersion).\(.attributes.minorVersion)"'
```

Bucket into v83 / v84 / v87 / v95 (patch) and v92 / jms (skip — see Step 3).

### 2.3 PATCH each tenant's socket config

1. `GET /configurations/tenants/{tenantId}`.
2. In `socket.handlers[]`, append the entry from §2.1 for that tenant's
   version. Leave every other handler/writer and existing `opCode`s untouched.
   Confirm the opCode is not already used by another **handler** in that
   tenant's config before appending.
3. `PATCH /configurations/tenants/{tenantId}` with the **full** updated config
   in a JSON:API envelope — `RegisterInputHandler` rejects a bare body
   (`{ "data": { "type": "<GetName()>", "attributes": { …full tenant config… } } }`;
   project memory `bug_ui_jsonapi_envelope_required_for_input_handlers`). The
   endpoint is a whole-document update; send what you GET'd with only
   `socket.handlers` appended to.

### 2.4 Restart the channel pods

```bash
kubectl -n <namespace> rollout restart deployment/atlas-channel
kubectl -n <namespace> rollout status  deployment/atlas-channel
```

Restart channel pods for every world/channel that serves the patched tenants.

### 2.5 Post-restart verification

1. **No "unhandled message op" logs** for the per-version opcode in §2.1 after
   a player double-clicks a reward-node Consume item on a patched tenant.
2. **Reward grant works end-to-end**: double-clicking a reward box (e.g. one
   of the v83 56 Consume reward boxes, design §2.5 scope) consumes the box and
   grants an item; if the item's data carries `Effect`/`worldMsg` (post
   Step 1), the buff/announce fires too.
3. **No regression on unpatched item-use opcodes** (`CharacterItemUseHandle`
   family) — ordinary consumables still work.

---

## Step 3: v92 — explicitly out of scope, do not patch

**Do not** add the `CharacterItemUseLotteryHandle` handler entry, and do not
run the Step 1 data re-publish path expecting it to matter, for:

- **v92** — dropped from this task's implementation (context.md: "v92 is
  DROPPED from this task"). v92 has no loaded IDA instance, so the opcode
  (`0x07B`, registry/CSV lineage only) and the full `LOTTERY_USE`
  operations-table shape are unverified, and v92 has no operations tables in
  its seed template at all. Adding an unverified opcode to a live v92 tenant
  risks colliding with an existing handler/writer or routing to the wrong body.
  The handler entry was therefore **not** registered in
  `template_gms_92_1.json` (an earlier scope-expansion commit added it; it was
  removed before this task's PR — see the code-review remediation). If v92
  support is wanted, it is a separate, larger follow-up task (build out the v92
  template's operations tables and writers once a v92 IDB exists to verify
  against) — not a live-tenant PATCH under this runbook.

> **jms is IN scope** (design §2.6, verified post-merge). Its handler
> (`CharacterItemUseLotteryHandle`, opcode `0x06B`) IS registered in
> `template_jms_185_1.json`, IDA-verified against the jms IDB
> (`CWvsContext::SendLotteryItemUseRequest`), and the matrix cell is promoted to
> ✅ (STATUS.md). jms tenants get the feature exactly like v83/v84/v87/v95, so
> the Step 1 data re-publish applies to jms too.

If a v92 tenant receives reward-node Consume items in its item data (from
Step 1's canonical re-ingest, which is not version-gated), those items will
simply have no server-side handler for the lottery-use opcode until a future
task closes that gap — the client-side routing behavior described in design
§2.1 is unaffected by data changes, only by the missing handler registration.

## Rollback

- **Data**: baseline publish/restore is non-destructive to game state (item
  definitions only); to roll back, publish/restore the prior canonical
  snapshot (identified by its sha256 from the previous publish response).
- **Config**: re-PATCH the affected tenants with the pre-change config
  (captured by the GET in §2.3) and `rollout restart` again. The handler change
  is config-only; no schema or data migration is involved.
