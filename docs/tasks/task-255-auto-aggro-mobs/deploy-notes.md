# task-255 — Deploy notes (auto-aggro mobs)

Seed templates apply only at tenant **creation**. The `AUTO_AGGRO` codec +
route landed in this task is dormant in every **existing** live tenant until
its config is PATCHed and `atlas-channel` is restarted (memory
`bug_new_opcodes_not_in_live_tenant_config`; see
[`docs/packets/IMPLEMENTING_A_PACKET.md`](../../packets/IMPLEMENTING_A_PACKET.md#after-it-merges--roll-out-to-live-tenants)).
The handler map is built once at startup — the config projection does not
hot-reload it, so the PATCH alone has no effect until the restart below.

This is a single serverbound opcode (`CMob::ApplyControl`) with no version
gates on the wire body, so every catalog version gets exactly one
`socket.handlers[]` row.

## PATCH table — `socket.handlers[]` entry per version

| version key | template | opCode | validator | handler | fname |
|---|---|---|---|---|---|
| gms_v48 | `template_gms_48_1.json` | `0x82` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v61 | `template_gms_61_1.json` | `0x9C` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v72 | `template_gms_72_1.json` | `0xB3` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v79 | `template_gms_79_1.json` | `0xB5` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v83 | `template_gms_83_1.json` | `0xBD` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v84 | `template_gms_84_1.json` | `0xC2` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v87 | `template_gms_87_1.json` | `0xC9` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v92 | `template_gms_92_1.json` | `0xDD` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v95 | `template_gms_95_1.json` | `0xE4` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| jms_v185 | `template_jms_185_1.json` | `0xC3` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |

PATCH shape — append to the live tenant config's `socket.handlers[]` array
(do not remove or reorder existing entries):

```jsonc
// gms_v83 example
{ "opCode": "0xBD", "validator": "LoggedInValidator", "handler": "AutoAggro" }
```

## Rollout checklist (per live tenant)

1. **Identify the tenant's version** and select the matching row above.
2. **PATCH the live tenant config** via the atlas-configurations REST PATCH
   path — append the row's `socket.handlers[]` entry.
3. **Restart `atlas-channel`** for that tenant — the handler map is built
   once at startup.
4. **Post-deploy checks** (via `mcp__kubernetes__pods_log` /
   `mcp__grafana__query_loki_logs`):
   - `grep "Unable to locate validator"` over the channel logs == **0**.
   - No new error/fatal logs after restart.
   - No `unhandled message op 0xXX` at info level for the tenant's `AUTO_AGGRO`
     opcode once a client sends it (mob comes into aggro range).

## Tuning dials (do not change without re-reading design.md §12)

- `monster.AutoAggroProximityThreshold` = **40** — the proximity gate on
  `atlas-channel`. Design §12: raising it is a valid response to a
  false-negative report, but **never past 100** — the client's own
  `nMoveAction` 18/19 bias adds `+100` to the distance score, and a threshold
  at or above 100 would defeat that bias and misclassify those move actions.
- `monster.AutoAggroRefreshInterval` = **5s** — throttles lease refresh
  requests for an already-aggro'd mob.
- `monster.AutoAggroLeaseTtlMs` = **15000** (15s) — how long a lease survives
  without a refresh before `atlas-monsters` releases it back to passive.
  Sized so two missed 5s refreshes are tolerated before decay.

## Manual live-channel verification (design §10)

Perform on at least one PATCHed tenant post-restart:

- A Jr. Necki (aggressive) turns toward and attacks an unprovoked nearby
  character.
- A Ribbon Pig (non-aggressive) is unaffected — no unsolicited chase/attack.
- Walking away from an aggro'd mob returns it to passive within ~15s (the
  lease TTL).
- A Dark Sight Rogue is not attacked by a nearby aggressive mob. Design §5:
  no server-side Dark Sight gate is added — client-side `CMob::Update` never
  arms an attack against an invisible player. If a mob does attack a
  dark-sighted character in this check, that disproves the client-side
  suppression and the recorded escalation is a channel-side buff-state gate,
  not a config change.
