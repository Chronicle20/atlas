# Live tenant-config PATCH — RPS opcodes/handlers/writers/operations

**Task 20 (companion note).** Seed templates only apply at **tenant creation**
(`services/atlas-configurations/seed-data/templates/template_*.json` are read
by the seeder, not re-applied to existing tenants). An already-provisioned
tenant's stored socket config in `atlas-configurations` does **not**
automatically pick up the RPS `socket.handlers`/`socket.writers`/`operations`
entries added to the seed templates in this task — see memory
`bug_new_opcodes_not_in_live_tenant_config` (same symptom class as the
monster-book cover opcodes, #659). Any tenant created **before** this task
lands needs its stored config PATCHed by hand, followed by an
`atlas-channel` restart.

Symptom if skipped: the client sends `RPS_ACTION` (or the server tries to
send `RPS_GAME`), `libs/atlas-socket/server.go` dispatcher logs `Read a
unhandled message with op 0x%02X.` (INFO) and silently drops the packet — no
Kafka command emitted, no clientbound frame sent, no error surfaced to the
player.

> **Amendment (2026-07-17, round-loop completion).** The `RPSGame` writer's
> `operations` table gained a **`START_SELECT: 9`** mode (the frame that enables
> the client's R/P/S buttons). A tenant provisioned before this landed will
> render the fee-confirm dialog and board but the throw buttons stay disabled
> after "Start" — the same silent-config-drift symptom. The per-version writer
> entries in §2 already include `START_SELECT`; a live tenant needs the same
> one-key add to its stored `RPSGame` writer `operations`
> (`{ "OPEN": 8, "START_SELECT": 9, "RESULT": 11, "END": 13 }`), then an
> `atlas-channel` restart. (The serverbound `RPSActionHandle` `operations` table
> is unchanged — `RETRY: 5` was already present.) The new `rps-rewards` config
> resource (certificate ladder + `consolationMeso`) is a **separate** PATCH on
> atlas-tenants — see `reward-ladder.md`.

---

## 1. Resource to PATCH

- Service: `atlas-configurations`
- Base path: `/api` (see `services/atlas-configurations/atlas.com/configurations/main.go` `prefix: "/api/"`)
- Resource: `tenants` (JSON:API type, `tenants.RestModel.GetName()` returns `"tenants"`)
- Routes (`services/atlas-configurations/atlas.com/configurations/tenants/resource.go`):
  - `GET  /api/configurations/tenants/{tenantId}` — fetch the current full tenant document
  - `PATCH /api/configurations/tenants/{tenantId}` — **replaces the whole tenant document** (`handleUpdateConfigurationTenant` marshals the entire `RestModel` — region/majorVersion/minorVersion/usesPin/socket/characters/npcs/worlds/cashShop — into one `resource_data` JSONB row; there is no partial-field merge)

**Because PATCH is a full-document replace, you MUST:**
1. `GET` the tenant's current config first.
2. Splice the new `socket.handlers` entry and `socket.writers` entry into the
   existing arrays (don't drop any existing handler/writer/operations data).
3. `PATCH` the **entire** modified document back, wrapped in a JSON:API
   envelope (`RegisterInputHandler` requires `{"data": {"type": "tenants",
   "id": "<tenantId>", "attributes": {...}}}` — see memory
   `bug_ui_jsonapi_envelope_required_for_input_handlers`; a bare `{...}` body
   400s with "Source JSON is empty and has no attributes payload object").

Example (illustrative shape only — always GET the real document first,
never hand-author the full body from scratch):

```bash
# 1. Fetch current config
curl -s http://atlas-configurations/api/configurations/tenants/<tenantId> | jq '.data.attributes' > current.json

# 2. Edit current.json: append the handler entry to .socket.handlers[]
#    and the writer entry to .socket.writers[] for this tenant's version
#    (see §2 below for the exact per-version entries).

# 3. PATCH the full document back
curl -s -X PATCH http://atlas-configurations/api/configurations/tenants/<tenantId> \
  -H 'Content-Type: application/vnd.api+json' \
  -d "$(jq -n --slurpfile attrs current.json --arg id "<tenantId>" \
        '{data: {type: "tenants", id: $id, attributes: $attrs[0]}}')"
```

---

## 2. Exact entries per version

Identify the tenant's `majorVersion` (from the GET response) and splice the
matching pair below into `socket.handlers` / `socket.writers`. These are
copied verbatim from the Task 20 seed-template edits (IDA-verified, Tasks
14/16 — do not alter opcodes or operations values).

### v83 (gms_83, majorVersion 83)

```json
{
  "opCode": "0x088",
  "validator": "LoggedInValidator",
  "handler": "RPSActionHandle",
  "options": {
    "operations": { "START": 0, "SELECT": 1, "UPDATE": 2, "CONTINUE": 3, "EXIT": 4, "RETRY": 5 }
  }
}
```
```json
{
  "opCode": "0x138",
  "writer": "RPSGame",
  "options": { "operations": { "OPEN": 8, "START_SELECT": 9, "RESULT": 11, "END": 13 } }
}
```

### v84 (gms_84, majorVersion 84)

```json
{
  "opCode": "0x08C",
  "validator": "LoggedInValidator",
  "handler": "RPSActionHandle",
  "options": {
    "operations": { "START": 0, "SELECT": 1, "UPDATE": 2, "CONTINUE": 3, "EXIT": 4, "RETRY": 5 }
  }
}
```
```json
{
  "opCode": "0x13F",
  "writer": "RPSGame",
  "options": { "operations": { "OPEN": 8, "START_SELECT": 9, "RESULT": 11, "END": 13 } }
}
```

### v87 (gms_87, majorVersion 87)

```json
{
  "opCode": "0x090",
  "validator": "LoggedInValidator",
  "handler": "RPSActionHandle",
  "options": {
    "operations": { "START": 0, "SELECT": 1, "UPDATE": 2, "CONTINUE": 3, "EXIT": 4, "RETRY": 5 }
  }
}
```
```json
{
  "opCode": "0x149",
  "writer": "RPSGame",
  "options": { "operations": { "OPEN": 8, "START_SELECT": 9, "RESULT": 11, "END": 13 } }
}
```

### v95 (gms_95, majorVersion 95)

```json
{
  "opCode": "0x0A0",
  "validator": "LoggedInValidator",
  "handler": "RPSActionHandle",
  "options": {
    "operations": { "START": 0, "SELECT": 1, "UPDATE": 2, "CONTINUE": 3, "EXIT": 4, "RETRY": 5 }
  }
}
```
```json
{
  "opCode": "0x173",
  "writer": "RPSGame",
  "options": { "operations": { "OPEN": 8, "START_SELECT": 9, "RESULT": 11, "END": 13 } }
}
```

### jms185 (jms_185, majorVersion 185)

```json
{
  "opCode": "0x08B",
  "validator": "LoggedInValidator",
  "handler": "RPSActionHandle",
  "options": {
    "operations": { "START": 0, "SELECT": 1, "UPDATE": 2, "CONTINUE": 3, "EXIT": 4, "RETRY": 5 }
  }
}
```
```json
{
  "opCode": "0x151",
  "writer": "RPSGame",
  "options": { "operations": { "OPEN": 8, "START_SELECT": 9, "RESULT": 11, "END": 13 } }
}
```

### v92 — NOT included

`template_gms_92_1.json` is intentionally **not** touched (v92 is parked, no
IDB available to verify the RPS opcodes/mode bytes for that version — see
Task 20 scope). Do not PATCH v92 tenants with any of the values above; they
are unverified for that version.

---

## 3. Restart `atlas-channel` — required, not optional

Per `bug_new_opcodes_not_in_live_tenant_config`: the channel's config
projection diff (`configuration/projection/apply.go`, `ListenerConfig` diff)
only compares IP/Port/Region/Version fields — **not** `Socket.Handlers` /
`Socket.Writers`. The per-tenant handler/writer maps are built once, at
listener-creation time, from `tenantCfg.Socket.Handlers` /
`tenantCfg.Socket.Writers` (`main.go` `produceHandlers()`/`produceWriters()`
wiring plus the tenant's stored opcode table). A handlers/writers-only config
change does **not** hot-reload.

After PATCHing every affected tenant's config:

```bash
kubectl rollout restart deployment/atlas-channel -n <namespace>
```

Wait for the rollout to complete before considering the RPS feature live for
that tenant. Verify by checking `atlas-channel` startup logs for the tenant
in question and/or exercising the RPS NPC flow end-to-end (client sends
`RPS_ACTION` START on entering the NPC 9000019 minigame; server should reply
`RPS_GAME` OPEN, not silently drop the packet).

---

## 4. Scope note

This note covers **only** already-provisioned tenants. New tenants created
after this task lands get the RPS wiring automatically from the updated seed
templates (`template_gms_83_1.json`, `_84_1`, `_87_1`, `_95_1`,
`template_jms_185_1.json`) — no manual PATCH needed for them.
