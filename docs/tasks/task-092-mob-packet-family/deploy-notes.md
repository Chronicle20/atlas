# task-092 — Deploy notes (MOB/MONSTER packet family rollout)

Seed templates apply only at tenant **creation**. The codecs + routes landed in
this task are dormant in every **existing** live tenant until its config is
PATCHed and `atlas-channel` is restarted (memory
`bug_new_opcodes_not_in_live_tenant_config`). This doc is the per-version
opcode table + rollout checklist for that operation.

The tables below are the exact `socket.writers` / `socket.handlers` entries this
task added to each seed template (derived from `git diff main` on the five
`template_*_1.json` files). Apply the matching rows to each live tenant of the
same version. Opcodes are hex, as the config stores them.

> **Opcodes differ per version** — never copy one version's row to another. v84
> in particular is **not** v83 + same opcodes: the v84 client shifted the
> clientbound/serverbound opcode tables (+7 cb / +6 sb above ~0x3D), so the
> payloads are byte-identical to v83 but the opcodes are not (memory
> `bug_v84_opcode_table_shifted_vs_v83`).

> **Every handler entry carries `validator: LoggedInValidator`** (all 52 added
> handlers across the five templates). A handler entry with a missing/unknown
> validator is silently dropped by `BuildHandlerMap` and the op no-ops (memory
> `bug_socket_handler_missing_validator_silently_dropped`).

PATCH shape — each entry is appended to the live tenant config's arrays:

```jsonc
// socket.writers[]
{ "opCode": "0xF9", "writer": "MobCrcKeyChanged" }

// socket.handlers[]
{ "opCode": "0xA4", "validator": "LoggedInValidator", "handler": "MobCrcKeyChangedReply" }
```

---

## gms_v83

### Clientbound — `socket.writers[]`
| opCode | writer |
|---|---|
| 0x4F | BridleMobCatchFail |
| 0xF4 | ResetMonsterAnimation |
| 0xF5 | MobAffected |
| 0xF7 | MonsterSpecialEffectBySkill |
| 0xF9 | MobCrcKeyChanged |
| 0xFB | CatchMonster |
| 0xFC | CatchMonsterWithItem |
| 0xFD | MobSpeaking |
| 0xFE | IncMobChargeCount |
| 0xFF | MobAttackedByMob |
| 0x121 | MonsterCarnivalStart |
| 0x122 | MonsterCarnivalObtainedCP |
| 0x123 | MonsterCarnivalPartyCP |
| 0x124 | MonsterCarnivalSummon |
| 0x125 | MonsterCarnivalMessage |
| 0x126 | MonsterCarnivalDied |
| 0x127 | MonsterCarnivalLeave |
| 0x128 | MonsterCarnivalResult |

### Serverbound — `socket.handlers[]` (all `LoggedInValidator`)
| opCode | handler |
|---|---|
| 0x38 | MobBanishPlayer |
| 0xA4 | MobCrcKeyChangedReply |
| 0xBE | MobDropPickupRequest |
| 0xBF | FieldDamageMob |
| 0xC1 | MonsterBomb |
| 0xC2 | MobDamageMob |
| 0xDA | MonsterCarnival |

---

## gms_v84

### Clientbound — `socket.writers[]`
| opCode | writer |
|---|---|
| 0x51 | BridleMobCatchFail |
| 0xFA | ResetMonsterAnimation |
| 0xFB | MobAffected |
| 0xFD | MonsterSpecialEffectBySkill |
| 0xFF | MobCrcKeyChanged |
| 0x101 | CatchMonster |
| 0x102 | CatchMonsterWithItem |
| 0x103 | MobSpeaking |
| 0x104 | IncMobChargeCount |
| 0x105 | MobSkillDelay |
| 0x106 | MobAttackedByMob |
| 0x128 | MonsterCarnivalStart |
| 0x129 | MonsterCarnivalObtainedCP |
| 0x12A | MonsterCarnivalPartyCP |
| 0x12B | MonsterCarnivalSummon |
| 0x12C | MonsterCarnivalMessage |
| 0x12D | MonsterCarnivalDied |
| 0x12E | MonsterCarnivalLeave |
| 0x12F | MonsterCarnivalResult |

### Serverbound — `socket.handlers[]` (all `LoggedInValidator`)
| opCode | handler |
|---|---|
| 0x38 | MobBanishPlayer |
| 0xAA | MobCrcKeyChangedReply |
| 0xBE | MobDropPickupRequest |
| 0xC4 | FieldDamageMob |
| 0xC7 | MobDamageMob |
| 0xC8 | MobSkillDelayEnd |
| 0xE0 | MonsterCarnival |

> v84-specific: `MobSkillDelay` (cb 0x105) is present in v84 but **absent in
> v83** — a genuine v84≠v83 delta (codec gates it `MajorAtLeast(84)` for that
> op). v84 lacks the monster-book writers/handler routes that v87+ carry.

---

## gms_v87

### Clientbound — `socket.writers[]`
| opCode | writer |
|---|---|
| 0x51 | BridleMobCatchFail |
| 0x55 | MonsterBookSetCard |
| 0x56 | MonsterBookSetCover |
| 0x104 | ResetMonsterAnimation |
| 0x105 | MobAffected |
| 0x107 | MonsterSpecialEffectBySkill |
| 0x109 | MobCrcKeyChanged |
| 0x10B | CatchMonster |
| 0x10C | CatchMonsterWithItem |
| 0x10D | MobSpeaking |
| 0x10E | IncMobChargeCount |
| 0x10F | MobSkillDelay |
| 0x110 | MobAttackedByMob |
| 0x132 | MonsterCarnivalStart |
| 0x133 | MonsterCarnivalObtainedCP |
| 0x134 | MonsterCarnivalPartyCP |
| 0x135 | MonsterCarnivalSummon |
| 0x136 | MonsterCarnivalMessage |
| 0x137 | MonsterCarnivalDied |
| 0x138 | MonsterCarnivalLeave |
| 0x139 | MonsterCarnivalResult |

### Serverbound — `socket.handlers[]` (all `LoggedInValidator`)
| opCode | handler |
|---|---|
| 0x3B | MobBanishPlayer |
| 0x3C | MonsterBookCover |
| 0xAE | MobCrcKeyChangedReply |
| 0xCA | MobDropPickupRequest |
| 0xCB | FieldDamageMob |
| 0xCC | MonsterDamageFriendlyHandle |
| 0xCD | MonsterBomb |
| 0xCE | MobDamageMob |
| 0xCF | MobSkillDelayEnd |
| 0xE7 | MonsterCarnival |

---

## gms_v95

### Clientbound — `socket.writers[]`
| opCode | writer |
|---|---|
| 0x52 | BridleMobCatchFail |
| 0x56 | MonsterBookSetCard |
| 0x57 | MonsterBookSetCover |
| 0x124 | ResetMonsterAnimation |
| 0x125 | MobAffected |
| 0x127 | MonsterSpecialEffectBySkill |
| 0x129 | MobCrcKeyChanged |
| 0x12B | CatchMonster |
| 0x12C | CatchMonsterWithItem |
| 0x12D | MobSpeaking |
| 0x12E | IncMobChargeCount |
| 0x12F | MobSkillDelay |
| 0x130 | MobEscortFullPath |
| 0x131 | MobEscortStop |
| 0x132 | MobEscortStopSay |
| 0x133 | MobEscortReturnBefore |
| 0x134 | MobNextAttack |
| 0x135 | MobAttackedByMob |
| 0x15A | MonsterCarnivalStart |
| 0x15B | MonsterCarnivalObtainedCP |
| 0x15C | MonsterCarnivalPartyCP |
| 0x15D | MonsterCarnivalSummon |
| 0x15E | MonsterCarnivalMessage |
| 0x15F | MonsterCarnivalDied |
| 0x160 | MonsterCarnivalLeave |
| 0x161 | MonsterCarnivalResult |

### Serverbound — `socket.handlers[]` (all `LoggedInValidator`)
| opCode | handler |
|---|---|
| 0x3D | MobBanishPlayer |
| 0x3E | MonsterBookCover |
| 0xBE | MobCrcKeyChangedReply |
| 0xE5 | MobDropPickupRequest |
| 0xE6 | FieldDamageMob |
| 0xE7 | MonsterDamageFriendlyHandle |
| 0xE8 | MonsterBomb |
| 0xE9 | MobDamageMob |
| 0xEA | MobSkillDelayEnd |
| 0xEB | MobTimeBombEnd |
| 0xEC | MobEscortCollision |
| 0xED | MobRequestEscortInfo |
| 0xEE | MobEscortStopEndRequest |
| 0x106 | MonsterCarnival |

> v95 is the only version carrying the full Cluster-F escort/next-attack tail
> (`MobEscort*`, `MobNextAttack`, `MobAttackedByMob`, `MobTimeBombEnd`).

---

## jms_v185

### Clientbound — `socket.writers[]`
| opCode | writer |
|---|---|
| 0x49 | BridleMobCatchFail |
| 0x57 | MonsterBookSetCard |
| 0x58 | MonsterBookSetCover |
| 0x105 | ResetMonsterAnimation |
| 0x106 | MobAffected |
| 0x108 | MonsterSpecialEffectBySkill |
| 0x10A | MobCrcKeyChanged |
| 0x10C | CatchMonster |
| 0x10D | CatchMonsterWithItem |
| 0x10E | MobSpeaking |
| 0x10F | MobSkillDelay |
| 0x110 | MobEscortFullPath |
| 0x112 | MobEscortStopSay |
| 0x113 | MobEscortReturnBefore |
| 0x114 | MobAttackedByMob |
| 0x139 | MonsterCarnivalStart |
| 0x13A | MonsterCarnivalObtainedCP |
| 0x13B | MonsterCarnivalPartyCP |
| 0x13C | MonsterCarnivalSummon |
| 0x13D | MonsterCarnivalMessage |
| 0x13E | MonsterCarnivalDied |
| 0x13F | MonsterCarnivalLeave |
| 0x140 | MonsterCarnivalResult |

### Serverbound — `socket.handlers[]` (all `LoggedInValidator`)
| opCode | handler |
|---|---|
| 0x30 | MobBanishPlayer |
| 0x31 | MonsterBookCover |
| 0x9E | MobCrcKeyChangedReply |
| 0xC4 | MobDropPickupRequest |
| 0xC5 | FieldDamageMob |
| 0xC6 | MonsterDamageFriendlyHandle |
| 0xC7 | MonsterBomb |
| 0xC8 | MobDamageMob |
| 0xC9 | MobSkillDelayEnd |
| 0xCA | MobTimeBombEnd |
| 0xCB | MobEscortCollision |
| 0xCC | MobRequestEscortInfo |
| 0xCD | MobEscortStopEndRequest |
| 0xE5 | MonsterCarnival |

> jms lacks `MOB_SPEAKING`/`INC_MOB_CHARGE_COUNT` as universal ops; the
> escort-tail writer rows (0x110/0x112/0x113) correspond to jms-specific
> registry rows (see RESUME-STATE residual #2 — registry-row dedupe is a
> Stage-4 cleanup item, not a rollout blocker).

---

## Rollout checklist (per live tenant)

1. **Identify the tenant's version** (gms_83 / gms_84 / gms_87 / gms_95 /
   jms_185) and select the matching table above.
2. **PATCH the live tenant config** — append the version's `socket.writers`
   rows to `socket.writers[]` and the `socket.handlers` rows (each with
   `"validator": "LoggedInValidator"`) to `socket.handlers[]`. Do not remove or
   reorder existing entries.
   - Use the atlas-configurations REST PATCH path (the same mechanism task-086
     used). The config projection does **not** hot-reload handler/writer maps,
     so the PATCH alone has no effect until the restart in step 3.
3. **Restart `atlas-channel`** for that tenant — the handler/writer maps are
   built once at startup.
4. **Post-deploy checks** (per memory `reference_observability`, via
   `mcp__kubernetes__pods_log` / `mcp__grafana__query_loki_logs`):
   - `grep "Unable to locate validator"` over the channel logs == **0**
     (a hit means a handler entry shipped without a registered validator).
   - No new error/fatal logs after restart.
   - The new serverbound ops no longer emit `unhandled message op 0x<NN>` at
     info level when the client triggers them.
   - Smoke a clientbound writer only if a feature emits it — most are
     intentional dormant seams (no emitter yet), so absence of traffic is
     expected, not a failure.

## Notes / known residuals (do not block rollout)

- A handful of cells stay ❌ in the matrix because their **evidence pin** is
  blocked (inlined/unnamed client senders): `MOB_BANISH_PLAYER` (v83/v84),
  `MOB_TIME_BOMB_END` (v83/v84/v87), `MONSTER_BOMB` (v84),
  `MOB_DROP_PICKUP_REQUEST` (v84), `MONSTER_BOOK_COVER` (v84). The codec **and
  route still ship** for these — only the verification pin is missing, so the
  rollout entries above are still correct and worth applying.
- `TOUCH_MONSTER_ATTACK` is **not** in any table — it was deferred to its own
  task (version-divergent attack packet, not byte-plumbing); its opcode stays
  unhandled by design.
