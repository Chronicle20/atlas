# task-181 — v79 clientbound writer routing (live-IDB verified)

**Verification method (correction).** The opcode check in `audit.md` §1 was done
against the checked-in registry export (`gms_v79.yaml`), a *secondary* source.
The writer routing below is verified the primary way: each writer's body was
compared against the **live v79 IDB** (`GMS_v79_1_DEVM.exe`, session 88dfa464)
read-order of the corresponding `CField_*::On*` / `CITC::On*` handler. Opcodes
come from decompiling the subclass `OnPacket` switch cases directly, not the
export.

The atlas writers already exist (implemented + golden-tested for gms_v83/v84/
v87/v95/jms_v185); they were simply unrouted in the v79 template. The opcode is
resolved from the template by writer name, so routing = adding the template
entry **iff** the atlas `Encode` body byte-for-byte matches the v79 read order.

## Routed (18) — atlas Encode == v79 read order (live IDB)

| Writer | v79 op | v79 handler (fname @addr) | body (v79 read == atlas Encode) |
|---|---|---|---|
| FieldTransportState | 0x8D | CField_ContiMove::OnContiState @0x537526 | byte,byte |
| AriantArenaShowResult | 0x93 | CField_AriantArena::OnShowResult @0x52914d | (empty) |
| SnowballHit | 0x104 | CField_SnowBall::OnSnowBallHit @0x5526ad | byte,short,short |
| SnowballMessage | 0x105 | CField_SnowBall::OnSnowBallMsg @0x5526e8 | byte,byte |
| SnowballTouch | 0x106 | CField_SnowBall::OnSnowBallTouch @0x55288e | (empty) |
| CoconutHit | 0x107 | CField_Coconut::OnCoconutHit @0x5332fa | short,short,byte |
| CoconutScore | 0x108 | CField_Coconut::OnCoconutScore @0x5332c8 | short,short |
| GuildBossHealerMove | 0x109 | CField_GuildBoss::OnHealerMove @0x54195c | short |
| GuildBossPulleyStateChange | 0x10A | CField_GuildBoss::OnPulleyStateChange @0x54198b | byte |
| MonsterCarnivalObtainedCP | 0x10C | CField_MonsterCarnival::OnPersonalCP @0x54849b | short,short |
| MonsterCarnivalPartyCP | 0x10D | CField_MonsterCarnival::OnTeamCP @0x5484cb | byte,short,short |
| MonsterCarnivalResult | 0x112 | CField_MonsterCarnival::OnShowGameResult @0x548a6a | byte |
| WitchTowerScoreUpdate | 0x117 | CField_Witchtower::OnScoreUpdate @0x561a4f | byte (v79<95 → atlas MajorAtLeast(95) int is gated off) |
| TournamentUew | 0x128 | CField_Tournament::OnTournamentUEW @0x5589e4 | byte |
| TournamentCharacters | 0x129 | CField_Tournament::OnPacket case 297 | (empty; client `nullsub` — receives, ignores) |
| WeddingProgress | 0x12A | CField_Wedding::OnWeddingProgress @0x55dfbb | byte,int,int (GMS `hasStep` → byte written) |
| WeddingCeremonyEnd | 0x12B | CField_Wedding::OnWeddingCeremonyEnd @0x55e5c7 | (empty) |
| MtsOperation2 | 0x143 | CITC::OnQueryCashResult @0x57f422 | int,int |

`test.Variants` does **not** include v79, so these are not covered by the
existing round-trip guard at v79; correctness rests on the read-order parity
above. Formal per-cell matrix promotion (add `packet-audit:verify version=gms_v79`
markers + regenerate `status.json`) is the remaining leaf step.

## Deferred (12) — v79 body diverges from the atlas (v83) writer; needs real work

| Writer | v79 op | Divergence (live IDB) |
|---|---|---|
| SnowballState | 0x103 | v79 reads `byte,int,int,(short,byte)×2,[short×3 first-time]`; atlas writes fixed `byte,int,int,short,byte,short×3`. Needs model (2 snowmen + first-time flag) + v79 gate. `OnSnowBallState @0x5525bf` |
| ContiMove | 0x8C | v79 `OnContiMove @0x5374c1` reads a leading byte then sub-dispatches (states 7–12; some read more); atlas writes only `byte`. Confirm which states the server emits. |
| Tournament | 0x125 | atlas writes 3 bytes (`mode,arg0,arg1`); v79 `OnTournament @0x5585af` reads ≤2 bytes. Diverges. |
| TournamentMatchTable | 0x126 | atlas `Encode` is an **empty stub**; v79 `OnTournamentMatchTable @0x55871f` reads a match-table struct (`sub_750E40`). Writer incomplete. |
| TournamentSetPrize | 0x127 | atlas writes `byte,byte,int,int` unconditionally; v79 `@0x5587e3` reads the 2 ints only when the flag byte is set. Conditional body. |
| AriantArenaUserScore | 0x113 | v79 `OnUserScore @0x528799` = `byte(count)` + **loop** `count×(str,int)`; atlas models a single `count,name,score`. Needs list model. |
| MonsterCarnivalStart | 0x10B | v79 `OnEnter @0x548324` = `byte + 6×short + N×byte` guardian loop; variable body. |
| MonsterCarnivalSummon | 0x10E | v79 `OnRequestResult(1) @0x54850a` = `byte,byte,str`. |
| MonsterCarnivalMessage | 0x10F | v79 `OnRequestResult(0) @0x54850a` = `byte`. |
| MonsterCarnivalDied | 0x110 | v79 `OnProcessForDeath @0x548774` = `byte,str,byte`. |
| MonsterCarnivalLeave | 0x111 | v79 `OnShowMemberOutMsg @0x5488ef` = `byte,byte,str`. |
| MtsOperation | 0x144 | v79 `CITC::OnNormalItemResult @0x57f4a7` is a **35-arm mode-prefix dispatcher** (0x15–0x3E). This is a dispatcher family, not a single writer — needs `DISPATCHER_FAMILY.md` treatment. |

## Absent in v79 (correctly unroutable) — from audit.md §5b/5c
SelectWorld, ServerListRecommendations, PicResult, PyramidGauge, PyramidScore,
SheepRanchInfo, SheepRanchClothes, VegaScroll (no client handler); UiOpen
(unresolved).
