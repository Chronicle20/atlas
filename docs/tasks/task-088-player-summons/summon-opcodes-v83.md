# v83 Summon Packet Opcodes (harvested record)

Running record for task-088 Player Summons. Phase 6 folds this into
`summon-packet-delta.md`. Values harvested from Cosmic (v83 GMS server) opcode
enums and cross-checked against the v83 client IDB (`MapleStory_dump.exe`,
ida-pro instance port 13337).

## Sources

- **Cosmic** `~/source/Cosmic/src/main/java/net/opcodes/SendOpcode.java` and
  `RecvOpcode.java`. Spawn/remove use `SPAWN_SPECIAL_MAPOBJECT` /
  `REMOVE_SPECIAL_MAPOBJECT` (see `PacketCreator.spawnSummon`/`removeSummon`,
  lines 1149 / 1172).
- **IDA** `CSummonedPool::OnPacket` @ `0x938dd7` — the client's inbound dispatch
  for all summon packets. Decompile shows the exact opcode→handler switch
  (definitive for the writer / server→client opcodes).

## Writers (server -> client) — the opcode the CLIENT receives & dispatches

These are confirmed directly in `CSummonedPool::OnPacket` @ `0x938dd7`:

| Packet (writer name) | Cosmic SendOpcode const | hex | dec | IDA confirmation |
|----------------------|-------------------------|-----|-----|------------------|
| SummonSpawn  | SPAWN_SPECIAL_MAPOBJECT  | 0xAF | 175 | confirmed @ 0x938df0 (`if (a2 == 0xAF)` → enter-field decode, vtable+44) |
| SummonRemove | REMOVE_SPECIAL_MAPOBJECT | 0xB0 | 176 | confirmed @ 0x938e40 (`if (a2 == 0xB0)` → leave-field, sub_7A64EB + pool remove) |
| SummonMove   | MOVE_SUMMON   | 0xB1 | 177 | confirmed @ 0x938e77 (`case 0xB1:` → CSummonedPool::OnMove @ 0x7a6861) |
| SummonAttack | SUMMON_ATTACK | 0xB2 | 178 | confirmed @ 0x938e77 (`case 0xB2:` → CSummonedPool::OnAttack @ 0x7a6882) |
| SummonDamage | DAMAGE_SUMMON | 0xB3 | 179 | confirmed @ 0x938e77 (`case 0xB3:` → CSummonedPool::OnHit @ 0x7a6e5a) |
| SummonSkill  | SUMMON_SKILL  | 0xB4 | 180 | confirmed @ 0x938e77 (`case 0xB4:` → CSummonedPool::OnSkill @ 0x7a6ebe) |

All six writer opcodes match Cosmic's `SendOpcode` enum exactly (0xAF–0xB4) and
are IDA-confirmed in the client dispatch switch. No Cosmic-vs-IDA disagreement.

## Handlers (client -> server) — the SEND opcode the CLIENT uses

These are the client-outbound recv-side opcodes (what atlas handlers must be
registered against). Cosmic `RecvOpcode.java` lines 171–173:

| Packet (handler name) | Cosmic RecvOpcode const | hex | dec | IDA confirmation |
|-----------------------|-------------------------|-----|-----|------------------|
| SummonMoveHandle   | MOVE_SUMMON   | 0xAF | 175 | Cosmic-derived, IDA-unconfirmed |
| SummonAttackHandle | SUMMON_ATTACK | 0xB0 | 176 | Cosmic-derived, IDA-unconfirmed |
| SummonDamageHandle | DAMAGE_SUMMON | 0xB1 | 177 | Cosmic-derived, IDA-unconfirmed |

The send-side opcodes are emitted from inlined CUserLocal/CSummoned/CVecCtrl
send paths (no dedicated named function isolatable via quick search); per task
guidance these are recorded as Cosmic-derived and not blocked on. Note the v83
SEND and RECV opcode tables are independent: the recv-side MOVE_SUMMON (0xAF)
shares a numeric value with the send-side SummonSpawn (0xAF) but they are
distinct directions/tables.

## Phase-1 template change

Inserted into `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
under `socket.writers[]` (kept opcode-sorted, after PetCommandResponse 0xAE):

```json
{ "opCode": "0xAF", "writer": "SummonSpawn" },
{ "opCode": "0xB0", "writer": "SummonRemove" }
```
