# task-096 — Deploy notes (CField map/field packet family rollout)

## Summary

task-096 verified the **CField** map/field packet family — ~66 new/linked ops
across the 75 work-list rows. The work landed as:

- **Codecs** in `libs/atlas-packet/field/{clientbound,serverbound}/` (chat,
  whisper, spouse-chat, admin/slash, doors, snowball/coconut/guild-boss
  minigames, tournament, wedding, foothold/MTS, OX-quiz, jukebox, quest-time,
  pyramid/ariant/sheep-ranch scoreboards, etc.).
- **Wiring** into `atlas-channel` (writer registration + serverbound handler
  registration).
- **Routes** added to the 5 seed templates
  (`services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_95,jms_185}_1.json`).

Seed templates apply only at tenant **creation**. The codecs + routes landed in
this task are dormant in every **existing** live tenant until its config is
PATCHed and `atlas-channel` is restarted (memory
`bug_new_opcodes_not_in_live_tenant_config`). This doc is the per-version
opcode table + rollout checklist for that operation.

The tables below are the exact `socket.writers` / `socket.handlers` entries this
task added to each seed template (derived from `git diff main...HEAD` on the
five `template_*_1.json` files). Apply the matching rows to each live tenant of
the same version. Opcodes are hex, as the config stores them.

> **Opcodes differ per version** — never copy one version's row to another. v84
> in particular is **not** v83 + same opcodes: the v84 client shifted the
> clientbound/serverbound opcode tables above ~0x3D, so payloads are
> byte-identical to v83 but the opcodes are not (memory
> `bug_v84_opcode_table_shifted_vs_v83`).

> **Every handler entry carries `validator: LoggedInValidator`**. A handler
> entry with a missing/unknown validator is silently dropped by
> `BuildHandlerMap` and the op no-ops (memory
> `bug_socket_handler_missing_validator_silently_dropped`).

PATCH shape — each entry is appended to the live tenant config's arrays:

```jsonc
// socket.writers[]
{ "opCode": "0x88", "writer": "SpouseChat" }

// socket.handlers[]
{ "opCode": "0x85", "validator": "LoggedInValidator", "handler": "UseDoor" }
```

Per-version added-entry counts (from the diff):

| version   | new handlers | new writers |
|-----------|-------------:|------------:|
| gms_v83   | 12 | 46 |
| gms_v84   | 12 | 45 |
| gms_v87   | 13 | 49 |
| gms_v95   | 14 | 50 |
| jms_v185  | 11 | 44 |

---

## gms_v83

### Serverbound — `socket.handlers[]` (all `LoggedInValidator`)
| opCode | handler |
|---|---|
| 0x72 | SueCharacter |
| 0x76 | AdminChat |
| 0x80 | AdminCommand |
| 0x81 | AdminLog |
| 0x85 | UseDoor |
| 0x8B | WeddingAction |
| 0x8C | WeddingTalk |
| 0xD3 | Snowball |
| 0xD4 | LeftKnockback |
| 0xD5 | Coconut |
| 0xD6 | MatchTable |
| 0xD7 | GuildBoss |

### Clientbound — `socket.writers[]`
| opCode | writer |
|---|---|
| 0x83 | BlockedMap |
| 0x84 | BlockedServer |
| 0x85 | ForcedMapEquip |
| 0x88 | SpouseChat |
| 0x89 | SummonItemUnavailable |
| 0x8B | FieldObstacleOnOff |
| 0x8C | FieldObstacleOnOffList |
| 0x8D | FieldObstacleAllReset |
| 0x8F | PlayJukebox |
| 0x90 | AdminResult |
| 0x91 | OxQuiz |
| 0x92 | GmEventInstructions |
| 0x94 | ContiMove |
| 0x96 | SetQuestClear |
| 0x97 | SetQuestTime |
| 0x98 | AriantResult |
| 0x99 | SetObjectState |
| 0x9A | StopClock |
| 0x9B | AriantArenaShowResult |
| 0x9C | StalkResult |
| 0x9D | PyramidGauge |
| 0x9E | PyramidScore |
| 0x119 | SnowballState |
| 0x11A | SnowballHit |
| 0x11B | SnowballMessage |
| 0x11C | SnowballTouch |
| 0x11D | CoconutHit |
| 0x11E | CoconutScore |
| 0x11F | GuildBossHealerMove |
| 0x120 | GuildBossPulleyStateChange |
| 0x129 | AriantArenaUserScore |
| 0x12B | SheepRanchInfo |
| 0x12C | SheepRanchClothes |
| 0x12D | WitchTowerScoreUpdate |
| 0x12E | HorntailCave |
| 0x12F | ZakumShrine |
| 0x13B | Tournament |
| 0x13C | TournamentMatchTable |
| 0x13D | TournamentSetPrize |
| 0x13E | TournamentUew |
| 0x13F | TournamentCharacters |
| 0x140 | WeddingProgress |
| 0x141 | WeddingCeremonyEnd |
| 0x15B | MtsOperation2 |
| 0x15C | MtsOperation |
| 0x162 | ViciousHammer |

---

## gms_v84

### Serverbound — `socket.handlers[]` (all `LoggedInValidator`)
| opCode | handler |
|---|---|
| 0x72 | SueCharacter |
| 0x78 | AdminChat |
| 0x84 | AdminCommand |
| 0x85 | AdminLog |
| 0x89 | UseDoor |
| 0x8F | WeddingAction |
| 0x90 | WeddingTalk |
| 0xD9 | Snowball |
| 0xDA | LeftKnockback |
| 0xDB | Coconut |
| 0xDC | MatchTable |
| 0xDD | GuildBoss |

### Clientbound — `socket.writers[]`
| opCode | writer |
|---|---|
| 0x86 | BlockedMap |
| 0x87 | BlockedServer |
| 0x88 | ForcedMapEquip |
| 0x8B | SpouseChat |
| 0x8C | SummonItemUnavailable |
| 0x8E | FieldObstacleOnOff |
| 0x8F | FieldObstacleOnOffList |
| 0x90 | FieldObstacleAllReset |
| 0x92 | PlayJukebox |
| 0x93 | AdminResult |
| 0x94 | OxQuiz |
| 0x95 | GmEventInstructions |
| 0x97 | ContiMove |
| 0x99 | SetQuestClear |
| 0x9A | SetQuestTime |
| 0x9B | AriantResult |
| 0x9C | SetObjectState |
| 0x9D | StopClock |
| 0x9E | AriantArenaShowResult |
| 0xA0 | PyramidGauge |
| 0xA1 | PyramidScore |
| 0x120 | SnowballState |
| 0x121 | SnowballHit |
| 0x122 | SnowballMessage |
| 0x123 | SnowballTouch |
| 0x124 | CoconutHit |
| 0x125 | CoconutScore |
| 0x126 | GuildBossHealerMove |
| 0x127 | GuildBossPulleyStateChange |
| 0x130 | AriantArenaUserScore |
| 0x132 | SheepRanchInfo |
| 0x133 | SheepRanchClothes |
| 0x134 | WitchTowerScoreUpdate |
| 0x135 | HorntailCave |
| 0x136 | ZakumShrine |
| 0x142 | Tournament |
| 0x143 | TournamentMatchTable |
| 0x144 | TournamentSetPrize |
| 0x145 | TournamentUew |
| 0x146 | TournamentCharacters |
| 0x147 | WeddingProgress |
| 0x148 | WeddingCeremonyEnd |
| 0x15B | MtsOperation2 |
| 0x15C | MtsOperation |
| 0x169 | ViciousHammer |

> v84 lacks the `StalkResult` writer row that v83/v87/v95/jms carry.

---

## gms_v87

### Serverbound — `socket.handlers[]` (all `LoggedInValidator`)
| opCode | handler |
|---|---|
| 0x75 | SueCharacter |
| 0x7C | AdminChat |
| 0x86 | GuildOperationHandle |
| 0x88 | AdminCommand |
| 0x89 | AdminLog |
| 0x8D | UseDoor |
| 0x93 | WeddingAction |
| 0x94 | WeddingTalk |
| 0xE0 | Snowball |
| 0xE1 | LeftKnockback |
| 0xE2 | Coconut |
| 0xE3 | MatchTable |
| 0xE4 | GuildBoss |

### Clientbound — `socket.writers[]`
| opCode | writer |
|---|---|
| 0x8B | BlockedMap |
| 0x8C | BlockedServer |
| 0x8D | ForcedMapEquip |
| 0x8E | CharacterMultiChat |
| 0x8F | CharacterChatWhisper |
| 0x90 | SpouseChat |
| 0x91 | SummonItemUnavailable |
| 0x93 | FieldObstacleOnOff |
| 0x94 | FieldObstacleOnOffList |
| 0x95 | FieldObstacleAllReset |
| 0x97 | PlayJukebox |
| 0x98 | AdminResult |
| 0x99 | OxQuiz |
| 0x9A | GmEventInstructions |
| 0x9C | ContiMove |
| 0x9E | SetQuestClear |
| 0x9F | SetQuestTime |
| 0xA0 | AriantResult |
| 0xA1 | SetObjectState |
| 0xA2 | StopClock |
| 0xA3 | AriantArenaShowResult |
| 0xA4 | StalkResult |
| 0xA5 | PyramidGauge |
| 0xA6 | PyramidScore |
| 0xAA | FootholdInfo |
| 0x12A | SnowballState |
| 0x12B | SnowballHit |
| 0x12C | SnowballMessage |
| 0x12D | SnowballTouch |
| 0x12E | CoconutHit |
| 0x12F | CoconutScore |
| 0x130 | GuildBossHealerMove |
| 0x131 | GuildBossPulleyStateChange |
| 0x13A | AriantArenaUserScore |
| 0x13C | SheepRanchInfo |
| 0x13D | SheepRanchClothes |
| 0x13E | WitchTowerScoreUpdate |
| 0x13F | HorntailCave |
| 0x140 | ZakumShrine |
| 0x14C | Tournament |
| 0x14D | TournamentMatchTable |
| 0x14E | TournamentSetPrize |
| 0x14F | TournamentUew |
| 0x150 | TournamentCharacters |
| 0x151 | WeddingProgress |
| 0x152 | WeddingCeremonyEnd |
| 0x170 | MtsOperation2 |
| 0x171 | MtsOperation |
| 0x177 | ViciousHammer |

---

## gms_v95

### Serverbound — `socket.handlers[]` (all `LoggedInValidator`)
| opCode | handler |
|---|---|
| 0x7E | SueCharacter |
| 0x8B | AdminChat |
| 0x97 | AdminCommand |
| 0x98 | AdminLog |
| 0x9C | UseDoor |
| 0x9E | SlideRequest |
| 0xA3 | WeddingAction |
| 0xA4 | WeddingTalk |
| 0xFF | Snowball |
| 0x100 | LeftKnockback |
| 0x101 | Coconut |
| 0x102 | MatchTable |
| 0x103 | GuildBoss |
| 0x10E | RequestFootholdInfo |

### Clientbound — `socket.writers[]`
| opCode | writer |
|---|---|
| 0x93 | BlockedMap |
| 0x94 | BlockedServer |
| 0x95 | ForcedMapEquip |
| 0x96 | CharacterMultiChat |
| 0x97 | CharacterChatWhisper |
| 0x98 | SpouseChat |
| 0x99 | SummonItemUnavailable |
| 0x9B | FieldObstacleOnOff |
| 0x9C | FieldObstacleOnOffList |
| 0x9D | FieldObstacleAllReset |
| 0x9F | PlayJukebox |
| 0xA0 | AdminResult |
| 0xA1 | OxQuiz |
| 0xA2 | GmEventInstructions |
| 0xA4 | ContiMove |
| 0xA6 | SetQuestClear |
| 0xA7 | SetQuestTime |
| 0xA8 | AriantResult |
| 0xA9 | SetObjectState |
| 0xAA | StopClock |
| 0xAB | AriantArenaShowResult |
| 0xAC | StalkResult |
| 0xAD | PyramidGauge |
| 0xAE | PyramidScore |
| 0xB0 | FootholdInfo |
| 0x152 | SnowballState |
| 0x153 | SnowballHit |
| 0x154 | SnowballMessage |
| 0x155 | SnowballTouch |
| 0x156 | CoconutHit |
| 0x157 | CoconutScore |
| 0x158 | GuildBossHealerMove |
| 0x159 | GuildBossPulleyStateChange |
| 0x162 | AriantArenaUserScore |
| 0x164 | SheepRanchInfo |
| 0x165 | SheepRanchClothes |
| 0x166 | AriantScore |
| 0x168 | WitchTowerScoreUpdate |
| 0x169 | HorntailCave |
| 0x16A | ZakumShrine |
| 0x176 | Tournament |
| 0x177 | TournamentMatchTable |
| 0x178 | TournamentSetPrize |
| 0x179 | TournamentUew |
| 0x17A | TournamentCharacters |
| 0x17B | WeddingProgress |
| 0x17C | WeddingCeremonyEnd |
| 0x19B | MtsOperation2 |
| 0x19C | MtsOperation |
| 0x1A9 | ViciousHammer |

---

## jms_v185

### Serverbound — `socket.handlers[]` (all `LoggedInValidator`)
| opCode | handler |
|---|---|
| 0x78 | AdminChat |
| 0x83 | AdminCommand |
| 0x84 | AdminLog |
| 0x88 | UseDoor |
| 0x89 | SlideRequest |
| 0xDE | Snowball |
| 0xDF | LeftKnockback |
| 0xE0 | Coconut |
| 0xE1 | MatchTable |
| 0xE2 | GuildBoss |
| 0xED | RequestFootholdInfo |

### Clientbound — `socket.writers[]`
| opCode | writer |
|---|---|
| 0x81 | BlockedMap |
| 0x82 | BlockedServer |
| 0x83 | ForcedMapEquip |
| 0x84 | CharacterMultiChat |
| 0x85 | CharacterChatWhisper |
| 0x86 | SummonItemUnavailable |
| 0x88 | FieldObstacleOnOff |
| 0x89 | FieldObstacleOnOffList |
| 0x8A | FieldObstacleAllReset |
| 0x8C | PlayJukebox |
| 0x8D | AdminResult |
| 0x8E | OxQuiz |
| 0x8F | GmEventInstructions |
| 0x91 | ContiMove |
| 0x93 | SetQuestClear |
| 0x94 | SetQuestTime |
| 0x95 | SetObjectState |
| 0x96 | StopClock |
| 0x97 | AriantArenaShowResult |
| 0x98 | StalkResult |
| 0x99 | PyramidGauge |
| 0x9A | PyramidScore |
| 0x9C | FootholdInfo |
| 0x131 | SnowballState |
| 0x132 | SnowballHit |
| 0x133 | SnowballMessage |
| 0x134 | SnowballTouch |
| 0x135 | CoconutHit |
| 0x136 | CoconutScore |
| 0x137 | GuildBossHealerMove |
| 0x138 | GuildBossPulleyStateChange |
| 0x141 | AriantArenaUserScore |
| 0x143 | SheepRanchInfo |
| 0x144 | SheepRanchClothes |
| 0x145 | HorntailCave |
| 0x146 | WitchTowerScoreUpdate |
| 0x148 | ZakumShrine |
| 0x154 | Tournament |
| 0x155 | TournamentMatchTable |
| 0x156 | TournamentSetPrize |
| 0x157 | TournamentUew |
| 0x158 | TournamentCharacters |
| 0x159 | WeddingProgress |
| 0x15A | WeddingCeremonyEnd |

> jms lacks the `SpouseChat` and `AriantResult` writer rows that gms carries.

---

## Rollout checklist (per live tenant)

1. **Identify the tenant's version** (gms_83 / gms_84 / gms_87 / gms_95 /
   jms_185) and select the matching tables above.
2. **PATCH the live tenant config** via atlas-tenants — append the version's
   `socket.writers` rows to `socket.writers[]` and the `socket.handlers` rows
   (each with `"validator": "LoggedInValidator"`) to `socket.handlers[]`. Do not
   remove or reorder existing entries.
   - Use the atlas-configurations REST PATCH path (the same mechanism task-086
     used). The config projection does **not** hot-reload handler/writer maps,
     so the PATCH alone has no effect until the restart in step 3.
3. **Restart `atlas-channel`** for that tenant — the handler/writer maps are
   built once at startup, not hot-reloaded.
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

---

## Resolved (fixed in commit `fa2fd9bbf`)

- **`WEDDING_ACTION`/`WEDDING_TALK` v84 routing — RESOLVED.** The stale v84
  `ALLIANCE_OPERATION` (143→**147 / 0x93**) and `DENY_ALLIANCE_REQUEST`
  (144→**148 / 0x94**) opcodes were re-derived from the v84 IDB (cross-ref v83
  PDB + v95), freeing 0x8F/0x90. `WeddingAction` (0x8F) and `WeddingTalk` (0x90)
  are now routed in `template_gms_84_1.json` `socket.handlers` (each with
  `LoggedInValidator`) — both cells ✅ via a **real route**, not opcode-occupancy.
  The two handler rows are now reflected in the v84 PATCH table above.
- **v84 UNNAMED phantom rows — RESOLVED.** 13 stale csv `n/a` placeholder rows
  (incl. `UNNAMED_R364` / `UNNAMED_R366` / `UNNAMED_R369`) were removed from
  `gms_v84.yaml`.
- **matrix-grader regression test — RESOLVED.** A regression test was added
  (commit `de8859a59`) covering the op-identity-aware `routedElsewhere` change
  (`tools/packet-audit`, commit `baa937176`).

## Known caveats / follow-ups (document honestly; do not block rollout)

1. **2 out-of-scope serverbound ops remain ❌:**
   - `CField::SendLocationWhisper` / `CField::SendChatMsgWhisper` (serverbound
     WHISPER), and
   - `CUIStatusBar::SendCoupleMessage` (serverbound SPOUSE_CHAT).
   They share op-NAMES with the verified **clientbound** WHISPER / SPOUSE_CHAT
   ops but were not in the 75-op work-list. Follow-up candidates.

2. **CSV transposition** — `docs/packets/MapleStory Ops - ClientBound.csv` has
   `HORNTAIL_CAVE` ↔ `WITCH_TOWER_SCORE_UPDATE` swapped in its GMS v83 + v87
   columns. The **registries were corrected** (IDB-verified, commit 44488a7db);
   the CSV (a reference resource) was left as-is. The tables in this doc reflect
   the corrected registry opcodes.

3. **Broader repo-wide v84 opcode-table reshift debt (NON-CField).** Freeing
   0x8F/0x90 for wedding moved ALLIANCE to 147/148, but the v84 opcode-table
   reshift is still incomplete repo-wide: ~8 serverbound + ~35 clientbound
   duplicate `(opcode, direction)` pairs remain in `gms_v84.yaml` where a real
   op shares an opcode with a stale csv-import row of a **different (non-CField)**
   family (e.g. ALLIANCE now at 147/148 sits on stale
   `ADD_FAMILY` / `SEPARATE_FAMILY_BY_SENIOR`). `matrix --check` does **not**
   flag these — no task-096 cell is mis-graded, and **no task-096 CField op is
   in any remaining duplicate** — but they should be closed by a dedicated
   repo-wide "complete the gms_v84 reshift" follow-up task.

---

## Verification state

- All 75 work-list ops are ✅ / ⬜ (no work-list op left ❌; the 2 ❌ ops in
  caveat #1 are out-of-scope, not work-list rows).
- **1228 total verified cells** across the matrix.
- `tools/packet-audit matrix --check` exits **0**.
- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in the changed
  modules.
- **No `go.mod` change** — no new shared lib added, so no `docker buildx bake`
  step was required for this task.
