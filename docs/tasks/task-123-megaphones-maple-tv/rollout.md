# Runbook: live-tenant rollout for avatar-megaphone / Maple TV (task-123)

> **EXECUTION-GATED — RECORDED, NOT EXECUTED.** Do not run any step here until
> task-123 has merged AND the updated images (`atlas-channel`, `atlas-world`,
> `atlas-saga-orchestrator`, `atlas-configurations`) are deployed to the
> target environment AND an operator has authorized the change for that
> environment. This document is written as part of task-123 (plan.md Task
> 17, FR-6.3); the implementation does not execute it.

## Why this is needed

Seed templates (`services/atlas-configurations/seed-data/templates/template_*.json`)
apply **only at tenant creation**. task-123 (commit `f25492f1c`) added six new
`socket.writers[]` entries — `SetAvatarMegaphone`, `ClearAvatarMegaphone`,
`AvatarMegaphoneResult`, `TvSetMessage`, `TvClearMessage`,
`TvSendMessageResult` — to the five gms/jms seed templates. Existing
(already-provisioned) tenants never re-read the seed, so these writers are
absent from their live socket config until patched in (project memory
`bug_new_opcodes_not_in_live_tenant_config`). Three of the six writers also
carry an `options` block (`errorCodes` / `messageTypes`) that resolves a
client-interpreted byte at encode time (DOM-25); an unpatched tenant that
somehow has the writer opcode but not its `options` block will still degrade
those bytes to `99` (see Pitfall 3 below).

## Blast radius / what happens if you skip this

- **Missing writer opcode entirely** — atlas-channel has no writer configured
  for that name on that tenant. Any code path that emits
  `SetAvatarMegaphoneWriter` / `TvSetMessageWriter` / etc. for that tenant
  fails to produce a packet (writer lookup miss); the feature is silently
  dead for that tenant — no client crash, but nothing renders.
- **Writer present but `options` missing/incomplete** — `ResolveCode`
  (`libs/atlas-packet/resolve.go:27-44`) returns `99` and logs
  `"Defaulting to 99 which will likely cause a client crash"` for every call
  that needed `errorCodes`/`messageTypes`. This is worse than a dead
  feature: a resolved-but-wrong mode byte is sent to the client, which can
  crash it.

Given the above, patch **both** the six writer entries and the three
`options` blocks together — do not stage them separately.

---

## Step 1: Per-version writer / handler / options deltas

All values below are copied verbatim from the seed templates as landed in
commit `f25492f1c` (`git show f25492f1c -- services/atlas-configurations/seed-data/templates/`).
Spot-verify against the live templates in this worktree before patching, in
case a later commit changed them.

### 1.1 The six new writers (opcodes per version)

`gms_12` and `gms_92` are **untouched** by task-123 (design D9) — do not
patch those tenants.

| Writer | gms_83 | gms_84 | gms_87 | gms_95 | jms_185 |
|---|---|---|---|---|---|
| `SetAvatarMegaphone` | `0x6F` | `0x72` | `0x72` | `0x73` | `0x5A` |
| `ClearAvatarMegaphone` | `0x70` | `0x73` | `0x73` | `0x74` | `0x5B` |
| `AvatarMegaphoneResult` | `0x6E` | `0x71` | `0x71` | `0x72` | **absent — jms has no `AvatarMegaphoneResult` entry; do not add one** |
| `TvSetMessage` | `0x155` | `0x15F` | `0x16A` | `0x195` | `0x17A` |
| `TvClearMessage` | `0x156` | `0x160` | `0x16B` | `0x196` | `0x17B` |
| `TvSendMessageResult` | `0x157` | `0x161` | `0x16C` | `0x197` | `0x17C` |

Entry shape (no `options`, e.g. `SetAvatarMegaphone`):

```json
{ "opCode": "0x6F", "writer": "SetAvatarMegaphone" }
```

(substitute the per-version `opCode` from the table above; every other field
of the tenant config is untouched)

### 1.2 `CharacterCashItemUseHandle` handler opcode (already required — documented for completeness)

This handler already exists in every template pre-task-123 (task-123 did not
add it — it routes megaphone/TV item-use through the existing USE_CASH_ITEM
path). Listed here so an operator patching a tenant that is missing it for
an unrelated reason knows the correct value per version. Verified against
the live templates (`template_gms_{83,84,87,95}_1.json`,
`template_jms_185_1.json`, grep for `CharacterCashItemUseHandle`):

| version | opCode |
|---|---|
| gms_83 | `0x4F` |
| gms_84 | `0x4F` |
| gms_87 | `0x52` |
| gms_95 | `0x55` |
| jms_185 | `0x47` |

If a tenant is somehow missing this handler, its entry must include a
`validator` (see Pitfall 2) — check the live template for the exact
validator name (`LoggedInValidator` in every template as of this writing)
before adding it.

### 1.3 A1 delta — the three writer-options tables (DOM-25(d), REQUIRED)

These are **not** optional polish — seed templates never retroactively apply
to live tenants, so an unpatched live tenant resolves an unconfigured
`errorCodes`/`messageTypes` key via `ResolveCode`
(`libs/atlas-packet/resolve.go:27-44`), which returns `99` and logs
`"Defaulting to 99 which will likely cause a client crash."` Every version
in the writer table above must also carry the matching `options` block.

**`AvatarMegaphoneResult.options.errorCodes`** — gms only (jms has no
`AvatarMegaphoneResult` writer, see 1.1):

```json
{
  "opCode": "0x6E",
  "writer": "AvatarMegaphoneResult",
  "options": {
    "errorCodes": {
      "WAITING_LINE": 83,
      "LEVEL_GATE": 84
    }
  }
}
```

(substitute the per-version `opCode` from §1.1; identical `errorCodes` values
across gms_83/84/87/95 — confirmed by diffing all four templates in commit
`f25492f1c`)

**`TvSendMessageResult.options.errorCodes`** — all five versions (gms +
jms):

```json
{
  "opCode": "0x157",
  "writer": "TvSendMessageResult",
  "options": {
    "errorCodes": {
      "GM_MESSAGE": 1,
      "QUEUE_TOO_LONG": 2,
      "WRONG_USER": 3
    }
  }
}
```

**`TvSetMessage.options.messageTypes`** — all five versions (gms + jms):

```json
{
  "opCode": "0x155",
  "writer": "TvSetMessage",
  "options": {
    "messageTypes": {
      "NORMAL": 0,
      "STAR": 1,
      "HEART": 2
    }
  }
}
```

`SetAvatarMegaphone`, `ClearAvatarMegaphone`, and `TvClearMessage` carry no
`options` block — do not add one.

### 1.4 Forward dependency — Task 18 WorldMessage `operations` table

Task 18 (plan.md, not yet landed as of this runbook's authoring — see
plan.md "Task 18: WorldMessage dispatcher family enrollment") IDA-derives
and may **correct** the per-version `WorldMessage` `operations` mode table
in `template_gms_{84,87,95}_1.json` / `template_jms_185_1.json` (the
existing v83-derived copies are unverified for those four versions per
design risk 2 / project memory `bug_operations_mode_tables_missing_v87_v95_jms`).
If Task 18 lands corrections, those tenants need an **additional** PATCH to
their `WorldMessage` writer's `options.operations` map, following the exact
same GET→modify→PATCH→restart procedure in Step 2 below — this runbook does
not re-derive those values; consult Task 18's committed
`docs/packets/dispatchers/worldmessage.yaml` and the corrected templates
once that task lands.

---

## Step 2: PATCH procedure for a live tenant's socket configuration

atlas-configurations serves the tenant config REST API; `$CONFIG_URL` is the
configurations-service ingress for the environment (e.g. the
`dev.atlas.home`-style ingress used elsewhere in this repo's runbooks).

### 2.1 Endpoint and envelope

- Route registration: `services/atlas-configurations/atlas.com/configurations/tenants/resource.go:24-29`.
  - `GET /configurations/tenants/{tenantId}` — `handleGetConfigurationTenant`.
  - `PATCH /configurations/tenants/{tenantId}` — `handleUpdateConfigurationTenant`,
    registered via `rest.RegisterInputHandler[RestModel]`.
- The tenant `RestModel.GetName()` returns `"tenants"`
  (`services/atlas-configurations/atlas.com/configurations/tenants/rest.go:24-26`).
- Request bodies are deserialized with `jsonapi.Unmarshal(body, &model)`
  (`libs/atlas-rest/server/context.go:56`) — this is a strict JSON:API
  unmarshal; **a bare JSON body 400s** (project memory
  `bug_ui_jsonapi_envelope_required_for_input_handlers`). Every PATCH must be
  wrapped:

```json
{
  "data": {
    "type": "tenants",
    "attributes": {
      "region": "GMS",
      "majorVersion": 83,
      "minorVersion": 1,
      "usesPin": false,
      "socket": { "handlers": [ /* … */ ], "writers": [ /* … */ ] },
      "characters": { /* … */ },
      "npcs": [ /* … */ ],
      "worlds": [ /* … */ ],
      "cashShop": { /* … */ }
    }
  }
}
```

### 2.2 Full-replace, not partial patch

`handleUpdateConfigurationTenant` → `ProcessorImpl.UpdateById`
(`services/atlas-configurations/atlas.com/configurations/tenants/processor.go:114-139`)
marshals the **entire** `input RestModel` and writes it as the tenant's
stored config document (`update(...)(db)`, line 134) — this is a
whole-document overwrite, not a merge. **Every field you do not include in
the PATCH body is deleted from the tenant's config, not left alone.** The
procedure is get→modify→put:

1. `GET /configurations/tenants/{tenantId}` and take the full `data.attributes`
   object from the response.
2. In `attributes.socket.writers[]`, append the six entries from §1.1 (with
   the `options` blocks from §1.3 on `AvatarMegaphoneResult` /
   `TvSendMessageResult` / `TvSetMessage`) for that tenant's version. Leave
   every other writer, every handler, and every other top-level field
   (`characters`, `npcs`, `worlds`, `cashShop`, …) exactly as returned by the
   GET. Confirm none of the six opcodes from §1.1 collide with an existing
   **writer** opcode already present in that tenant's `socket.writers[]`
   before appending (writers and handlers are separate opcode spaces —
   `RestModel` in `services/atlas-configurations/atlas.com/configurations/tenants/socket/rest.go:8-11`).
3. `PATCH /configurations/tenants/{tenantId}` with the modified full
   `attributes` object wrapped in the `{"data":{"type":"tenants","attributes":{...}}}`
   envelope from §2.1.
4. The PATCH also enqueues a tenant-status Kafka event
   (`enqueueTenantStatus`, `services/atlas-configurations/atlas.com/configurations/tenants/processor.go:28-46`,
   topic named by `EnvTenantStatusTopic` =
   `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`, `processor.go:22`, when that
   env var is set) — this drives the config-status projection consumed
   elsewhere, but it does **not** hot-reload atlas-channel's in-memory
   socket handler/writer maps (see Step 3).

### 2.3 Enumerate live tenants and their versions

```bash
curl -s "$CONFIG_URL/configurations/tenants" | jq -r \
  '.data[] | "\(.id)\t\(.attributes.region) v\(.attributes.majorVersion).\(.attributes.minorVersion)"'
```

Bucket into gms_v83 / gms_v84 / gms_v87 / gms_v95 / jms_v185 (patch per §1.1)
and gms_v12 / gms_v92 (skip — untouched by task-123, design D9).

---

## Step 3: Restart atlas-channel

Config projection does not hot-reload socket handlers or writers —
atlas-channel builds its opcode→handler and opcode→writer maps once at
startup (`libs/atlas-opcodes/producer.go`). A PATCH alone has no runtime
effect until the pods are restarted:

```bash
kubectl -n <namespace> rollout restart deployment/atlas-channel
kubectl -n <namespace> rollout status  deployment/atlas-channel
```

New pods must reach **Ready** before testing — a pod mid-rollout may still
be serving the pre-patch handler/writer map. Restart channel pods for every
world/channel that serves the patched tenants.

---

## Step 4: Deploy-order note

`atlas-world` and `atlas-saga-orchestrator` must be deployed **before or
with** `atlas-channel`, not after:

- atlas-channel's `handleMapleTVUse` / avatar-megaphone handlers perform a
  synchronous REST check against atlas-world's broadcast queue before
  consuming the item — `GET /worlds/{worldId}/broadcast-queues/{family}`
  (`services/atlas-channel/atlas.com/channel/worldbroadcast/requests.go:13-14`,
  called from `worldbroadcast.Processor.GetWaitSeconds`,
  `services/atlas-channel/atlas.com/channel/worldbroadcast/processor.go:31`).
  If atlas-world doesn't yet expose this endpoint, the REST call fails and
  the handler **rejects conservatively without consuming the item**
  (`character_cash_item_use_megaphone.go:200-208`, `:300-306` — "Unable to
  check TV/avatar queue... rejecting without consuming") — not a crash, but
  the feature silently no-ops.
- atlas-channel creates sagas using the `EnqueueWorldBroadcast` /
  `EmitMegaphone` actions (`character_cash_item_use_megaphone.go:247-248`,
  `:319-320`); if atlas-saga-orchestrator does not yet recognize those
  action names, saga creation/execution fails for those item uses.
- The three topic env vars added in Task 15 —
  `COMMAND_TOPIC_WORLD_BROADCAST`, `EVENT_TOPIC_MEGAPHONE`,
  `EVENT_TOPIC_WORLD_BROADCAST_STATUS` (confirmed present in
  `deploy/k8s/base/env-configmap.yaml:78,123,156` and both overlay
  `kustomization.yaml`s) — must exist in the environment (topic created,
  env var populated) **before any of the three services' pods are
  restarted**, since each reads its topic env vars once at startup.

Recommended order per environment: deploy/restart atlas-world and
atlas-saga-orchestrator first (with the topic env vars already present),
confirm they're Ready, **then** perform the config PATCH (Step 2) and
restart atlas-channel (Step 3).

---

## Pitfall callouts

1. **New-opcode-silently-dropped.** A live tenant whose `socket.writers[]`
   lacks one of the six opcodes in §1.1 does not error — the writer lookup
   simply has nothing configured for that name, and any packet meant to use
   it is never produced. Symptom: the feature does nothing for that tenant,
   no log noise pointing at the cause unless you already know to look.
   (project memory `bug_new_opcodes_not_in_live_tenant_config`.) Fix: patch
   the live config (Step 2) + restart channel (Step 3).
2. **Missing-validator-silently-dropped-handler.** This applies to
   `socket.handlers[]` entries specifically (not the six writers added by
   this task — writers have no validator field). `BuildHandlerMap`
   (`libs/atlas-opcodes/producer.go:44-50`) looks up `hc.Validator` in the
   validator map and, on a miss, logs a `Warnf` and **`continue`s** — the
   handler is silently dropped from the opcode table with only a warning in
   the logs, no error surfaced anywhere else. Every `socket.handlers` entry
   added or patched (e.g. if `CharacterCashItemUseHandle` is ever
   missing from a tenant and needs adding per §1.2) must include a
   `validator` key with a name that exists in atlas-channel's validator
   map.
3. **New writer-options table missing on a live tenant → resolves to 99.**
   The A1 delta in §1.3: if `AvatarMegaphoneResult`, `TvSendMessageResult`,
   or `TvSetMessage` is patched in with its `opCode` but without its
   `options` block (or with an incomplete one), `ResolveCode`
   (`libs/atlas-packet/resolve.go:27-44`) returns `99` for the missing key
   and logs `"Defaulting to 99 which will likely cause a client crash."`
   Unlike Pitfall 1, this is not silent-no-op — a byte the client doesn't
   expect is actually sent, which can crash the client. Always patch the
   `opCode` and its `options` block together (§2.2 step 2).

---

## Step 5: Legacy GMS (v48/61/72/79) — task-123 legacy-phase-2 delta

Legacy-phase-1 (`.superpowers/sdd/legacy-megaphone-protocol.md`) IDA-verified
the serverbound basic/super megaphone codecs and the clientbound
WorldMessage/SetAvatarMegaphone shapes for gms_v48/61/72/79 but made no
template or handler-gate changes. Legacy-phase-2 (this phase) wires those
findings into the seed templates and opens the channel handler gate. Unlike
§1.1's five versions, `CharacterCashItemUseHandle` and the `WorldMessage`
writer (with its full `operations` mode table) were **already present** in
all four legacy templates from an earlier version-bring-up pass — verified
unchanged, no delta to record for those two. The delta is the three
avatar-megaphone writers, newly added:

| Writer | gms_48 | gms_61 | gms_72 | gms_79 |
|---|---|---|---|---|
| `SetAvatarMegaphone` | `0x42` | `0x54` | `0x67` | `0x69` |
| `ClearAvatarMegaphone` | `0x43` | `0x55` | `0x68` | `0x6A` |
| `AvatarMegaphoneResult` | `0x41` | `0x53` | `0x66` | `0x68` |

`AvatarMegaphoneResult.options.errorCodes` (IDA-verified per version —
`CWvsContext::OnAvatarMegaphoneRes`, `Decode1() - <base>`; `v3==0` →
WAITING_LINE, `v3==1` → LEVEL_GATE, matching the gms_83 audit's semantic
mapping):

| version | WAITING_LINE | LEVEL_GATE | IDA address |
|---|---|---|---|
| gms_48 | 48 | 49 | `0x7211cd` |
| gms_61 | 55 | 56 | `0x84aa30` |
| gms_72 | 63 | 64 | `0x9220de` |
| gms_79 | 75 | 76 | `0x974213` |

`SetAvatarMegaphone` opcodes confirmed against
`CWvsContext::OnSetAvatarMegaphone` (addresses in legacy-megaphone-protocol.md
§4). `ClearAvatarMegaphone` for v61/72/79 confirmed against
`CWvsContext::OnClearAvatarMegaphone` (`0x84accd` / `0x92237d` / `0x9744b2`);
v48's counterpart was unnamed in the IDB (`sub_721465`) — decompiled and
confirmed via its `CAvatarMegaphone::ByeAvatarMegaphone` call (the `Bye`
counterpart to `OnSetAvatarMegaphone`'s `HelloAvatarMegaphone`), then renamed
to `CWvsContext::OnClearAvatarMegaphone` in the v48 IDB (port 13337) so the
opcode (67 / `0x43`, immediately following `SET_AVATAR_MEGAPHONE`=66) is not
a guess.

These three writers exist for **clientbound render only** — a legacy client
in the same map/world as a v83+ sender can now render an avatar-megaphone
broadcast. Legacy clients still cannot *send* one (see below).

### Handler gate change

`character_cash_item_use.go`'s `MajorVersion() < 83` item-loss guard was
refined from an all-or-nothing block to a per-tier check (see the code
comment at the gate for the full citation):

- **Basic (tier 1) / Super (tier 2) megaphone** — now **ALLOWED** on
  gms_48/61/72/79: serverbound codec + clientbound WorldMessage arms were
  IDA-verified (protocol spec §2/§3) and the writer/handler opcodes already
  existed in the templates (verified this phase).
- **Avatar megaphone** (any tier) — still **BLOCKED** on all four legacy
  versions: no legacy build's serverbound send case could be reliably
  located (protocol spec §5a); consuming the item would destroy it with
  nothing verified to decode. The new writers above only enable *receiving*
  a v83+ sender's avatar-megaphone broadcast, not sending one.
- **Maple TV (tier 4/5) / item megaphone (tier 6) / triple megaphone (tier
  7)** — still **BLOCKED**: no legacy send case identified (protocol spec
  §5b for TV; item/triple confirmed absent from the legacy dispatcher).
- v83+/JMS: unchanged — the `MajorVersion() < 83` branch is never entered,
  so every tier keeps dispatching exactly as before this phase.

No live-tenant PATCH runbook changes beyond the table above — this section
applies the same get→modify→PATCH→restart procedure (§2) and deploy-order
note (§4) to gms_48/61/72/79 tenants once this phase merges and is deployed.

---

## Rollback

- Config: re-PATCH the affected tenants with the pre-change `attributes`
  object (captured by the GET in §2.2 step 1) and `rollout restart` again.
  The change is config-only; no schema or data migration is involved.
- Deploy order: if atlas-channel was restarted before atlas-world /
  atlas-saga-orchestrator were ready and megaphone/TV item uses are
  rejecting with "Unable to check TV/avatar queue" warnings, no rollback is
  needed beyond finishing the atlas-world / atlas-saga-orchestrator rollout
  — the channel handlers already fail closed (reject without consuming) in
  that state, so no player-visible corruption occurs, only a temporarily
  dead feature.
