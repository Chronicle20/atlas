# Runbook: live-config patch for the OnMessage (CharacterStatusMessage) mode-table fix

> **EXECUTION-GATED.** Do not run any step here until task-104 has merged AND the
> updated images are deployed to the target environment AND an operator has
> authorized the change for that environment. This document is authored as part
> of task-104; it is **not** executed by the implementation.

## Why this is needed

Seed templates
(`services/atlas-configurations/seed-data/templates/template_*.json`) apply **only
at tenant creation**. Existing tenants do not receive corrected/added writer
config when a template changes (see project memory
`bug_new_opcodes_not_in_live_tenant_config`). task-104 changed the
`CharacterStatusMessage` writer config in two ways that must be projected onto
**already-provisioned** tenants:

1. **gms_v83 tenants — CORRECTION.** The live v83 `operations` map carries the old
   (buggy) table: it includes `INCREASE_SKILL_POINT: 4` and has
   `INCREASE_FAME..SKILL_EXPIRE` one too high. On v83 the client switch has 14
   cases (0–0xD) with **no** skill-point arm, so today fame renders as meso, every
   arm from fame up is off by one, and `SKILL_EXPIRE` (sent at 14) exceeds the v83
   `default` boundary and is dropped. (Same class as
   `bug_v83_status_message_operations_off_by_one`.)

2. **jms_v185 tenants — ADDITION.** Live jms tenants have **no**
   `CharacterStatusMessage` writer entry at all (the seed template never had one),
   so jms status messages never resolved a mode → `ResolveCode` 99. task-104 adds
   the writer at opCode `0x25` with the full 16-mode table.

3. **gms_v84 / gms_v87 / gms_v95 tenants — NO CHANGE EXPECTED.** Their live tables
   already match the corrected yaml (the regenerate step only rewrote gms_83 and
   jms_185). Confirm, but do not patch unless a tenant has drifted.

## Target data (what the writer's `operations` map must become)

`socket.writers[]` entry with `writer: "CharacterStatusMessage"`:

- **gms_v83** (opCode `0x27`) — corrected map (no `INCREASE_SKILL_POINT`):
  ```json
  { "DROP_PICK_UP":0, "QUEST_RECORD":1, "CASH_ITEM_EXPIRE":2, "INCREASE_EXPERIENCE":3,
    "INCREASE_FAME":4, "INCREASE_MESO":5, "INCREASE_GUILD_POINT":6, "GIVE_BUFF":7,
    "GENERAL_ITEM_EXPIRE":8, "SYSTEM_MESSAGE":9, "QUEST_RECORD_EX":10,
    "ITEM_PROTECT_EXPIRE":11, "ITEM_EXPIRE_REPLACE":12, "SKILL_EXPIRE":13 }
  ```
- **jms_v185** (opCode `0x25`, ADD the whole writer) — 16-mode map:
  ```json
  { "DROP_PICK_UP":0, "QUEST_RECORD":1, "CASH_ITEM_EXPIRE":2, "INCREASE_EXPERIENCE":3,
    "INCREASE_SKILL_POINT":4, "INCREASE_FAME":5, "INCREASE_MESO":6, "INCREASE_GUILD_POINT":7,
    "GIVE_BUFF":8, "GENERAL_ITEM_EXPIRE":9, "SYSTEM_MESSAGE":10, "QUEST_RECORD_EX":11,
    "ITEM_PROTECT_EXPIRE":12, "ITEM_EXPIRE_REPLACE":13, "SKILL_EXPIRE":14,
    "JMS_COUNTER_NOTICE":15 }
  ```
- **gms_v84 / gms_v87 / gms_v95** (opCode `0x27` for v84/v87, `0x26` for v95) —
  already correct (`INCREASE_SKILL_POINT:4 … SKILL_EXPIRE:14`); verify only.

These mirror the committed seed templates exactly — diff against
`template_{gms_83_1,jms_185_1}.json` if in doubt; the templates are the source of
truth.

## Procedure (per environment)

atlas-configurations serves the tenant config REST API. The base URL is the
configurations service ingress for the environment (the same host the channel/
world services read config from).

### 1. Enumerate live tenants and their versions

```bash
# All tenants + their region/major/minor.
curl -s "$CONFIG_URL/configurations/tenants" | jq -r \
  '.data[] | "\(.id)\t\(.attributes.region) v\(.attributes.majorVersion).\(.attributes.minorVersion)"'
```

Bucket them: `v83` (correction), `v185`/jms (addition), `v84/v87/v95` (verify only).

### 2. For each gms_v83 tenant — PATCH the CharacterStatusMessage operations

1. `GET /configurations/tenants/{tenantId}` and locate the
   `socket.writers[]` element with `writer == "CharacterStatusMessage"`.
2. Replace its `options.operations` with the **gms_v83 corrected map** above
   (remove `INCREASE_SKILL_POINT`; renumber `INCREASE_FAME..SKILL_EXPIRE` to
   4..13). Leave the `opCode` (`0x27`) and every other writer/handler untouched.
3. `PATCH /configurations/tenants/{tenantId}` with the **full** updated config in
   a **JSON:API envelope** — `RegisterInputHandler` rejects a bare body
   (`{ "data": { "type": "<GetName()>", "attributes": { …full tenant config… } } }`;
   see `bug_ui_jsonapi_envelope_required_for_input_handlers`). The endpoint is a
   whole-document update, so send the config you GET'd with only the operations
   map changed.

### 3. For each jms_v185 tenant — ADD the CharacterStatusMessage writer

Same GET/PATCH, but the writer does not exist yet: append a new element to
`socket.writers[]`:
```json
{ "opCode": "0x25", "writer": "CharacterStatusMessage",
  "options": { "operations": { …the jms_v185 16-mode map above… } } }
```
Confirm `0x25` is not already used by another **writer** in that tenant
(it is used by a serverbound *handler*, `CharacterMagicAttackHandle` — that is a
different opcode space and does not conflict).

### 4. For gms_v84 / gms_v87 / gms_v95 tenants — verify only

GET the config and confirm the `CharacterStatusMessage` operations map already
matches (`INCREASE_SKILL_POINT:4 … SKILL_EXPIRE:14`). Patch only if drifted.

### 5. Restart the channel pods

The config projection does **not** hot-reload socket writers/handlers; the
channel builds its writer map at startup. After patching:

```bash
kubectl -n <namespace> rollout restart deployment/atlas-channel
kubectl -n <namespace> rollout status  deployment/atlas-channel
```
(Restart channel pods for every world/channel that serves the patched tenants.)

## Post-restart verification

1. **No unhandled-message logs.** Channel logs show no
   `unhandled message op` / `cannot find` for the status-message opcode
   (`0x27` v83 / `0x25` jms) after a status event.
2. **v83 semantics correct.** On a v83 tenant, a **fame** gain renders as fame
   (not meso), a **meso** gain renders as meso, and a **skill expire** renders
   (previously dropped). A spot check across fame/meso/guild-point/skill-expire
   confirms the off-by-one is gone.
3. **jms status messages work.** On a jms tenant, a drop pick-up / exp / fame
   status message renders (previously the writer was absent so nothing was sent).

## Rollback

Re-PATCH the affected tenants with the pre-change config (captured by the GET in
step 2/3) and `rollout restart` again. The change is config-only; no schema or
data migration is involved.
