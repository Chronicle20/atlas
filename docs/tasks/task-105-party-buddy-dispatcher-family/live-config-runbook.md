# Runbook: live-config patch for the party + buddy operations tables (task-105)

> **EXECUTION-GATED — RECORDED, NOT EXECUTED.** Do not run any step here until
> task-105 has merged AND the updated images are deployed to the target
> environment AND an operator has authorized the change for that environment.
> This document is authored as part of task-105 (design §9 / PRD §6 scope the live
> patch OUT of this task — it is operational); the implementation does not execute it.

## Why this is needed

Seed templates (`services/atlas-configurations/seed-data/templates/template_*.json`)
apply **only at tenant creation**. Existing tenants do not receive added/corrected
writer config when a template changes (project memory
`bug_new_opcodes_not_in_live_tenant_config`). task-105 changed the
`PartyOperation` and `BuddyOperation` writer `operations` maps; the change must be
projected onto **already-provisioned** tenants for the newly-split party/buddy error
arms to resolve a mode byte instead of the `99` fallback that crashes the client.

The amplified gap (project memory `bug_operations_mode_tables_missing_v87_v95_jms`):

1. **gms_v87 / gms_v95 / jms_v185 tenants — ADDITION (critical).** Their live
   `PartyOperation` / `BuddyOperation` `operations` maps are **empty / near-empty**
   today (measured: v87 Party=1 op / Buddy=0; v95 Party=1 / Buddy=0; jms Party=`{TOWN_PORTAL}` /
   Buddy has no options block). So on those versions virtually every party/buddy arm
   — error AND non-error (invite, update, join, create…) — resolves to `99` →
   client crash. task-105 populates the **full** per-version table from
   `docs/packets/dispatchers/party.yaml` / `buddy.yaml`.
2. **gms_v83 / gms_v84 tenants — NO FUNCTIONAL CHANGE.** Their live tables already
   carry the full party/buddy operations with the correct **values**. task-105 only
   normalized the seed value FORMAT (hex-string `"0x09"` → int `9`); `ResolveCode`
   parses both, so live v83/v84 tenants need **no patch**. Verify only.

## Target data (what each writer's `operations` map must become)

The committed seed templates are the source of truth — diff against
`template_{gms_87_1,gms_95_1,jms_185_1}.json`. Key per-version facts to validate
after patching (the non-uniform drift — do NOT assume v87/v95/jms == v83):

### PartyOperation (`socket.writers[]` writer `"PartyOperation"`)
- LOW arms (cases ≤ JOIN/0x0F) are byte-identical to v83: INVITE 4, UPDATE 7,
  CREATED 8, ALREADY_HAVE_JOINED_A_PARTY_1 9, A_BEGINNER_CANT_CREATE_A_PARTY 10,
  LEAVE/DISBAND/EXPEL 12, YOU_HAVE_YET_TO_JOIN_A_PARTY 13, JOIN 15.
- **+1 SHIFT from ALREADY_HAVE_JOINED_A_PARTY_2 up** on v87/v95/jms:
  ALREADY_HAVE_JOINED_A_PARTY_2 **17**, THE_PARTY…FULL_CAPACITY **18**,
  CANNOT_KICK_ANOTHER_USER_IN_THIS_MAP **29**, CHANGE_LEADER **31**,
  THIS_CAN_ONLY_BE_GIVEN…VICINITY **32**, UNABLE_TO_HAND_OVER… **33**,
  YOU_MAY_ONLY_CHANGE…SAME_CHANNEL **34**, AS_A_GM…FORBIDDEN **36**,
  UNABLE_TO_FIND_THE_CHARACTER **37** (v87/v95 only).
- TOWN_PORTAL: v87 **41** (0x29), v95 **46** (0x2E), jms **40** (0x28).
- **VERSION-ABSENT** on v87/v95/jms (NOT in the table — do not invent a mode):
  UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL, IS_CURRENTLY_BLOCKING…,
  IS_TAKING_CARE…, HAVE_DENIED_REQUEST… . Additionally UNABLE_TO_FIND_THE_CHARACTER
  is absent on **jms** only.

### BuddyOperation (`socket.writers[]` writer `"BuddyOperation"`)
- Byte-identical across all 5 versions (buddy is NOT shifted): UPDATE 7,
  BUDDY_UPDATE 8, INVITE 9, UNKNOWN_1 10, BUDDY_LIST_FULL 11, OTHER_BUDDY_LIST_FULL 12,
  ALREADY_BUDDY 13, CANNOT_BUDDY_GM 14, CHARACTER_NOT_FOUND 15, UNKNOWN_ERROR 16,
  UNKNOWN_ERROR_2 17, UNKNOWN_2 18, UNKNOWN_ERROR_3 19, BUDDY_CHANNEL_CHANGE 20,
  CAPACITY_CHANGE 21, UNKNOWN_ERROR_4 22.
- jms tenants today have **no BuddyOperation options block at all** — add the whole
  16-entry map. (The UNKNOWN_ERROR family's trailing extra byte is GMS-only and is
  handled in the writer struct, not the mode table.)

## Procedure (per environment)

atlas-configurations serves the tenant config REST API; `$CONFIG_URL` is the
configurations-service ingress for the environment.

### 1. Enumerate live tenants and their versions
```bash
curl -s "$CONFIG_URL/configurations/tenants" | jq -r \
  '.data[] | "\(.id)\t\(.attributes.region) v\(.attributes.majorVersion).\(.attributes.minorVersion)"'
```
Bucket: `v87`/`v95`/`jms_v185` (POPULATE), `v83`/`v84` (verify only).

### 2. For each gms_v87 / gms_v95 / jms_v185 tenant — POPULATE the two writers
1. `GET /configurations/tenants/{tenantId}`.
2. In `socket.writers[]`, set the `options.operations` map of the `"PartyOperation"`
   writer and the `"BuddyOperation"` writer to the **full** per-version maps from the
   committed `template_{gms_87_1,gms_95_1,jms_185_1}.json` (party = the version-correct
   shifted/absent set above; buddy = the 16-entry map). On jms, if the `BuddyOperation`
   writer or its `options` block is missing, add it (opCode per the seed template).
   Leave every other writer/handler and the `opCode`s untouched.
3. `PATCH /configurations/tenants/{tenantId}` with the **full** updated config in a
   **JSON:API envelope** — `RegisterInputHandler` rejects a bare body
   (`{ "data": { "type": "<GetName()>", "attributes": { …full tenant config… } } }`;
   project memory `bug_ui_jsonapi_envelope_required_for_input_handlers`). The endpoint
   is a whole-document update; send what you GET'd with only the two operations maps changed.

### 3. For gms_v83 / gms_v84 tenants — verify only
GET the config; confirm the `PartyOperation`/`BuddyOperation` operations maps already
carry the full key set with the v83 values. Patch only if a tenant has drifted.

### 4. Restart the channel pods
The config projection does **not** hot-reload socket writers; the channel builds its
writer map at startup.
```bash
kubectl -n <namespace> rollout restart deployment/atlas-channel
kubectl -n <namespace> rollout status  deployment/atlas-channel
```
Restart channel pods for every world/channel that serves the patched tenants.

## Post-restart verification
1. **No 99-fallback / resolve-failure logs** for `PARTY_OPERATION` / `BUDDYLIST` after a
   party or buddy action on a patched v87/v95/jms tenant.
2. **Party errors render** on v87/v95/jms: trigger e.g. "already joined a party",
   "full capacity", "cannot kick in this map", a leader change, a town portal — each
   renders instead of crashing the client.
3. **Buddy errors render** on v87/v95/jms: buddy-list-full / already-buddy /
   character-not-found / cannot-buddy-GM each render.
4. **No regression on v83/v84** (unchanged) — party/buddy continue to work.

## Rollback
Re-PATCH the affected tenants with the pre-change config (captured by the GET in step 2)
and `rollout restart` again. The change is config-only; no schema or data migration.
