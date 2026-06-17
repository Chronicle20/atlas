# task-096 — Serverbound CField codecs (recipe R-SB)

7 serverbound CField codecs implemented + verified. Send-site FNames carry a
`#suffix` tag in the per-version ida-exports so each maps to a distinct op via
`candidatesFromFName` (several share a base send-site).

| Op | Struct | FName (send-site) | Layout | v83 | v84 | v87 | v95 | jms |
|----|--------|-------------------|--------|-----|-----|-----|-----|-----|
| SNOWBALL | Snowball | `CField_SnowBall::BasicActionAttack#Snowball` | attack byte, damage u16, x u16 | 0xD3 | 0xD9 | 0xE0 | 0xFF | 0xDE |
| LEFT_KNOCKBACK | LeftKnockback | `CField_SnowBall::Update#LeftKnockback` | empty | 0xD4 | 0xDA | 0xE1 | 0x100 | 0xDF |
| COCONUT | Coconut | `CField_Coconut::BasicActionAttack#Coconut` | attack u16, x u16 | 0xD5 | 0xDB | 0xE2 | 0x101 | 0xE0 |
| GUILD_BOSS | GuildBoss | `CField_GuildBoss::BasicActionAttack#GuildBoss` | empty | 0xD7 | 0xDD | 0xE4 | 0x103 | 0xE2 |
| USE_DOOR | UseDoor | `CField::TryEnterTownPortal#UseDoor` | portalFieldId u32, flag byte | 0x85 | 0x89 | 0x8D | 0x9C | 0x88 |
| WEDDING_ACTION | WeddingAction | `CField_Wedding::OnWeddingProgress#Action` | step byte | 0x8B | 0x8F | 0x93 | 0xA3 | ⬜ |
| WEDDING_TALK | WeddingTalk | `CField_Wedding::OnWeddingProgress#Talk` | empty | 0x8C | 0x90 | 0x94 | 0xA4 | ⬜ |

WeddingAction/WeddingTalk are jms-version-absent (no jms registry rows / markers /
routes); their jms cell grades `⬜`.

## Known issue: v84 WEDDING handler routes omitted (stale ALLIANCE registry collision)

WeddingAction/WeddingTalk handler routes were added to the `gms_83`, `gms_87`,
`gms_95` seed templates but **NOT** to `gms_84`.

Reason: in v84 the IDA-discovered WEDDING send-site opcodes shifted +4 vs v83
(139→143 / 140→144 = 0x8F/0x90; `provenance: ida-discovered`,
`address: 0x5911E6`). The v84 registry rows for `ALLIANCE_OPERATION` (serverbound,
opcode 143/0x8F) and `DENY_ALLIANCE_REQUEST` (serverbound, opcode 144/0x90) are
**stale `csv-import` values carried over from the v83 CSV** — their `fname`
(`CFadeWnd::SendCloseMessage`) is not even present in the v84 export, and the
v84 serverbound opcode table is known to have shifted (memory
`bug_v84_opcode_table_shifted_vs_v83`). They were never re-derived for v84.

Adding a `WeddingAction` handler at v84 0x8F (and `WeddingTalk` at 0x90) makes
the matrix's op-identity guard see those opcodes routed to WEDDING rather than
ALLIANCE, which flips the (correctly verified, messenger-shared-codec)
`ALLIANCE_OPERATION × {v83,v87,v95,jms}` cells from ✅ to 🟥 — 4 template-wiring
conflicts on an out-of-scope op.

The WEDDING_ACTION/WEDDING_TALK × v84 matrix cells still grade ✅ **without** the
explicit v84 handler route, because the opcodes 0x8F/0x90 are already occupied by
the v84 clientbound FIELD_OBSTACLE_ONOFF_LIST / FIELD_OBSTACLE_ALL_RESET entries
(opcode-occupancy satisfies the routing check) and the codec carries its
marker + pinned evidence + audit report.

**Functional gap (flagged for follow-up):** the v84 channel will not dispatch
inbound WEDDING_ACTION/WEDDING_TALK packets at 0x8F/0x90 until the stale v84
`ALLIANCE_OPERATION` / `DENY_ALLIANCE_REQUEST` serverbound opcodes are re-derived
against the v84 IDB (so the colliding WEDDING handler routes can be added without
fabricating ALLIANCE's true shifted opcode). This is a v84 registry-data
correction outside the R-SB scope and requires IDB adjudication — not invented
here. v83/v87/v95 wedding handlers are routed and functional.
