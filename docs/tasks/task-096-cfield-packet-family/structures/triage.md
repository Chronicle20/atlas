# task-096 — Phase 0 Triage

Anti-duplication gate for the 75-op CField family. This file is structured so the
full A/B/C per-op classification table (Task 0.4) appends below the C-row section.

All IDB evidence below is from the **v83** IDB (`MapleStory_dump.exe`, idb `v83_Me`,
IDA port 13342 this session). Addresses are virtual addresses in that image. The
ground-truth opcode is the integer passed to `COutPacket::COutPacket(&pkt, OP)` for
serverbound sends, or the dispatcher `case` label in `CField::OnPacket` /
`CField_*::OnPacket` for clientbound recv. Registry decimal opcodes were
cross-checked against these hex values.

Dispatcher reference (v83): `CField::OnPacket(long, CInPacket&)` @ `0x531325`
(decimal 5444389) — the per-opcode `case` table that routes every clientbound
CField recv handler.

---

## C-row resolutions

### 1. Foothold / stalk cluster (`IDA_0X098/09C/09D/0A4/0AA/0AC/0B0/0B1`, `IDA_0X169`)

The work-list (`cfield-ops.md`) lists nine `IDA_0x…` rows. **Per the matrix
(`STATUS.md`), only `IDA_0X09C` is present (`❌`) for v83**; every other row in
this cluster is `⬜` (n/a) in the v83 column — they are opcodes that exist only in
v87/v95/jms (verified against the per-version registries: v87 has `IDA_0X0A4`,
v95 has `IDA_0X0AC`, jms has `IDA_0X098`/`0AC`, etc.). This task resolves the v83
IDB only, so the v83 verdicts are:

| Row | v83 verdict | Resolved fname | Address | Opcode (hex/dec) | IDA evidence |
|---|---|---|---|---|---|
| `IDA_0X09C` | **distinct CB op, keep** | `CField::OnStalkResult` | `0x537a6a` (5470826) | `0x9C` / 156 | `CField::OnPacket` `case 0x9C` → `CField::OnStalkResult(0x537a6a)`. Handler reads `Decode4` count then loops `Decode4`(id)+`Decode1`(flag)+optional `DecodeStr`+2×`Decode4` — friend-finder/stalk result. Reads CInPacket ⇒ clientbound. |
| `IDA_0X098` | **version-absent (⬜)** | n/a in v83 | — | `0x98` not OnStalkResult in v83 | `CField::OnPacket` `case 0x98` → `CField::OnWarnMessage` (= ARIANT_RESULT), **not** OnStalkResult. The `IDA_0X098→OnStalkResult` mapping is a jms-only row. |
| `IDA_0X09D` | **version-absent (⬜)** | n/a in v83 | — | no `OnFootHoldInfo`/`OnRequestFootHoldInfo` fn in v83 | `lookup_funcs` for `?OnFootHoldInfo@CField@…` and `?OnRequestFootHoldInfo@CField@…` → **Not found** (functions do not exist in the v83 image). `case 0x9D` is absent from the dispatcher. |
| `IDA_0X0A4` | **version-absent (⬜)** | n/a in v83 | — | v87-only row | v83 `CField::OnPacket` has no `case 0xA4`; opcodes `0xA0–0xEB` route to `CUserPool::OnPacket`, not CField. (v87 registry owns `IDA_0X0A4`.) |
| `IDA_0X0AA` | **version-absent (⬜)** | n/a in v83 | — | v87/jms-only row | Same `0xA0–0xEB → CUserPool` range; no CField `case 0xAA` in v83. |
| `IDA_0X0AC` | **version-absent (⬜)** | n/a in v83 | — | v95/jms-only row | Same range; no CField `case 0xAC` in v83. |
| `IDA_0X0B0` | **version-absent (⬜)** | n/a in v83 | — | v95-only row | `OnFootHoldInfo` fn absent in v83; no `case 0xB0`. |
| `IDA_0X0B1` | **version-absent (⬜)** | n/a in v83 | — | v95-only row | `OnRequestFootHoldInfo` fn absent in v83; no `case 0xB1`. |
| `IDA_0X169` | **version-absent (⬜)** | n/a in v83 | — | v95/jms-only row | `CField::OnHontailTimer` **exists** in v83 @ `0x5335a3` but is routed at `case 0x12E` (= HORNTAIL_CAVE, opcode 302), **not** `0x169`. The `IDA_0X169→OnHontailTimer` mapping is a higher-version opcode-table shift, absent in v83. |

**Conclusion:** no `IDA_0x…` placeholder survives for v83. `IDA_0X09C` keeps its
op-key (it is a genuinely-unnamed-in-Maple opcode; all five version registries key
`OnStalkResult` the same way, so renaming would orphan the cross-version matrix
row) but its fname is confirmed `CField::OnStalkResult` and its `ida.address` is
corrected from `0x531325` (the dispatcher) to `0x537a6a` (the handler). The other
eight rows are `⬜ VERSION-ABSENT` for v83 with the IDB evidence above; no v83
registry rows exist for them and none are created.

`OnFootHoldInfo`/`OnRequestFootHoldInfo` themselves are *serverbound* concepts in
v83 — the serverbound rows `REQUEST_FOOTHOLD_INFO` (op 225, `CStage::OnSetField`)
and `FOOTHOLD_INFO` (op 226, `CField::OnRequestFootHoldInfo`) already exist; they
are outside the C-row (`IDA_0x…`) scope and are handled by the A/B classification.

### 2. MTS — `CField::OnCharacterSale`

**Verdict: two DISTINCT packets (two models), NOT two modes of one structure.**

`CField::OnCharacterSale` @ `0x537fa6` is a thin forwarder
(`v3 = this[126]; if (v3) (*(*v3 + 60))(v3, a2, a3);`) — it delegates to the ITC
dialog's vtable. The real dispatcher is `CITC::OnPacket` @ `0x5A4205`:

```
case 346: CITC::OnChargeParamResult(0x5a4241)   // IDA_0X15A
case 347: CITC::OnQueryCashResult (0x5a428c)    // MTS_OPERATION2
case 348: CITC::OnNormalItemResult(0x5a4311)    // MTS_OPERATION
```

347 (`0x15B`) and 348 (`0x15C`) are **separate `case` labels routing to separate
handler functions** — distinct opcodes, distinct read orders. They are not a
mode-byte branch inside one handler. The registry already encodes this correctly
(`MTS_OPERATION2`→`CITC::OnQueryCashResult`, `MTS_OPERATION`→`CITC::OnNormalItemResult`,
both `provenance: manual` with prior IDA notes). No registry change required;
the verdict is recorded here. The CSV's original `CField::OnCharacterSale` fname
was wrong (that handler covers `0x161–0x164`, the in-field character-sale ops, not
the ITC/MTS stage handlers) and was already realigned.

→ Implementation (Task 0.4+): two distinct B-rows / two models, two routes.

### 3. Door / Guild — direction resolution

| Op | Verdict | Direction | fname | Address | Opcode | IDA evidence |
|---|---|---|---|---|---|---|
| `USE_DOOR` | resolved | **serverbound** | `CField::TryEnterTownPortal` | `0x5375ed` | `0x85` / 133 | Called from `CUserLocal::HandleUpKeyDown` (up-key at a door). Builds `COutPacket(0x85)`, `Encode4`(townportal id) + `Encode1(1)`, `CClientSocket::SendPacket` ⇒ client→server. Matches existing serverbound registry row (op 133). |
| `GUILD_OPERATION` | resolved | **serverbound** (multi-mode) | `CField::InputGuildName` | `0x5305ae` | `0x7E` / 126 | Called from `CWvsContext::OnGuildResult`. Builds `COutPacket(126)`, `Encode1(2)` (mode = name-input) + `EncodeStr`(guild name), `SendPacket` ⇒ client→server. `InputGuildName` is one of nine `fname_alts` on the existing serverbound `GUILD_OPERATION` row (op 126); the leading mode byte confirms it is a mode-multiplexed serverbound op. |

`GUILD_OPERATION` is an **A-row**: already served by `guild/serverbound/GuildOperation`
(per `codec-inventory.md`, verified v95/jms). The `?` in the work-list resolves to
serverbound. (The clientbound `GUILD_OPERATION` at op 65 = `CWvsContext::OnGuildResult`
is a separate, non-CField row and is out of this family's scope.)

`USE_DOOR` resolves to serverbound; the existing serverbound registry row is correct.

### 4. Minigame `?` rows — send-site fnames confirmed

All four `?` rows resolve to **serverbound** sends, and the fnames listed in the
work-list (which looked like state/update methods) are confirmed to BE the actual
packet send-sites — each builds a `COutPacket` and calls `SendPacket` directly.

| Op | Direction | fname | Address | Opcode | IDA evidence (COutPacket call) |
|---|---|---|---|---|---|
| `SNOWBALL` | **serverbound** | `CField_SnowBall::BasicActionAttack` | `0x575387` | `0xD3` / 211 | `COutPacket(0xD3)`; `Encode1`(attack result) `Encode2`(damage) `Encode2`(x); `SendPacket`. |
| `LEFT_KNOCKBACK` | **serverbound** | `CField_SnowBall::Update` | `0x574df1` | `0xD4` / 212 | `COutPacket(0xD4)` (empty body) sent on crossing the knockback boundary; `SendPacket`. |
| `COCONUT` | **serverbound** | `CField_Coconut::BasicActionAttack` | `0x549902` | `0xD5` / 213 | `COutPacket(0xD5)`; `Encode2`(attack) `Encode2`(x); `SendPacket`. |
| `GUILD_BOSS` | **serverbound** | `CField_GuildBoss::BasicActionAttack` | `0x558b45` | `0xD7` / 215 | `COutPacket(0xD7)` (empty body) after `CPulley::Hit`; `SendPacket`. |

All four already exist as serverbound registry rows with the exact correct
fname + opcode (`provenance: csv-import`); this triage upgrades each to
`provenance: manual` with an IDA citation note.

`CONTI_MOVE` (`CField_ContiMove::Init`): **the `?` row collapses into the existing
clientbound `CONTI_MOVE`**. `CField_ContiMove::OnPacket` @ `0x54dc6d` routes only two
ops — `case 0x94 → OnContiMove` and `case 0x95 → OnContiState`, both reading
CInPacket (clientbound). There is **no serverbound conti-move send-site**; `Init`
is collapsed/inlined map-init logic (`CField_ContiMove::Update` @ `0x54dc28` is a
5-byte collapsed stub), not a packet function. The work-list's
`CONTI_MOVE → CField_ContiMove::Init` `?` entry is therefore spurious — it
duplicates the clientbound `CONTI_MOVE` (op 148/`0x94`, `OnContiMove`), which the
registry already has. No new row; no registry change for CONTI_MOVE.

### 5. WHISPER duplicate

**Verdict: list-rendering duplicate in `cfield-ops.md`, NOT two registry rows or
two modes.** `cfield-ops.md` lines 48–49 both read
`[CB] WHISPER … CField::OnWhisper`. The v83 registry already has the correct two
WHISPER rows in two directions:

- clientbound `WHISPER` — op 135 / `0x87`, `CField::OnWhisper`
  (`CField::OnPacket` `case 0x87` → `CField::OnWhisper` @ `0x53228e`; reads CInPacket).
- serverbound `WHISPER` — op 120 / `0x78`, `CField::SendLocationWhisper`
  (alts incl. `SendChatMsgWhisper`).

The two identical clientbound lines in the work-list are a display dupe (the same
clientbound row emitted twice), not whisper-vs-whisper-reply modes. No registry
change required; one CB + one SB row is correct.

---

## Summary of registry patches (gms_v83.yaml)

| Row | Change |
|---|---|
| `IDA_0X09C` (CB, op 156) | `ida.address` 5444389 → 5470826 (`0x537a6a`); `provenance` ida-discovered → manual; add IDA note. |
| `MTS_OPERATION` / `MTS_OPERATION2` | no change (already correct, `manual`); verdict recorded above. |
| `USE_DOOR` (SB, op 133) | `provenance` csv-import → manual; add IDA note. |
| `GUILD_OPERATION` (SB, op 126) | `provenance` csv-import → manual; add IDA note (direction confirmed SB). |
| `SNOWBALL` (SB, op 211) | `provenance` csv-import → manual; add IDA note. |
| `LEFT_KNOCKBACK` (SB, op 212) | `provenance` csv-import → manual; add IDA note. |
| `COCONUT` (SB, op 213) | `provenance` csv-import → manual; add IDA note. |
| `GUILD_BOSS` (SB, op 215) | `provenance` csv-import → manual; add IDA note. |
| `WHISPER` (CB op 135 / SB op 120) | no change (already correct); verdict recorded above. |

Eight rows patched; no `IDA_0x…` placeholders survive into code; no new rows
created (all genuinely-absent v83 cluster rows stay `⬜`/absent per the matrix).

<!-- Task 0.4 (A/B/C per-op table) appends below this line. -->

## Task 0.4 — Committed A/B/C triage table (75 ops)

This is the authoritative classification (design decision D4) consumed by every
Stage-2 cluster. It is a pure synthesis of Tasks 0.1 (baseline.md), 0.2
(codec-inventory.md), and the C-row resolutions above (0.3). No C-row survives:
every formerly-ambiguous row collapses to **A**, **B**, or **version-absent**.

**Classification legend:**

- **A** — a correct codec already exists. Stage-2 will **R-MARK** it (add a
  `packet-audit:verify` marker + byte fixture; no new codec), or **R-WRAP** it
  when the op is served by a shared-model codec. No new codec source.
- **B** — no codec exists. Stage-2 writes a net-new codec: **R-CB** for a
  clientbound writer, **R-SB** for a serverbound handler.
- **C-resolved** — was a C-row (ambiguous direction / ambiguous fname / spurious
  duplicate) in 0.3; the cell records the 0.3 verdict and its final A / B /
  version-absent landing.

`existing codec path` is relative to `libs/atlas-packet/`. Direction: CB =
clientbound, SB = serverbound. "owner class" is the `CField`-family class that
owns the op per the v83 IDB / registry.

### CField (45 work-list rows)

| op | dir | owner class | existing codec path | classification | resolution note |
|----|-----|-------------|---------------------|----------------|-----------------|
| ADMIN_CHAT | SB | CField | — | **B** (R-SB) | no codec; `CField::SendChatMsgSlash` slash-command family. |
| ADMIN_COMMAND | SB | CField | — | **B** (R-SB) | no codec; slash-command family. |
| ADMIN_LOG | SB | CField | — | **B** (R-SB) | no codec; slash-command family. |
| ADMIN_RESULT | CB | CField | — | **B** (R-CB) | `CField::OnAdminResult`. |
| ARIANT_RESULT | CB | CField | — | **B** (R-CB) | `CField::OnWarnMessage`; jms VERSION-ABSENT (⬜) candidate. |
| BLOCKED_MAP | CB | CField | — | **B** (R-CB) | `CField::OnTransferFieldReqIgnored`. |
| BLOCKED_SERVER | CB | CField | — | **B** (R-CB) | `CField::OnTransferChannelReqIgnored`. |
| FIELD_OBSTACLE_ALL_RESET | CB | CField | — | **B** (R-CB) | `CField::OnFieldObstacleAllReset`. |
| FIELD_OBSTACLE_ONOFF | CB | CField | — | **B** (R-CB) | `CField::OnFieldObstacleOnOff`. |
| FIELD_OBSTACLE_ONOFF_LIST | CB | CField | — | **B** (R-CB) | `CField::OnFieldObstacleOnOffStatus`. |
| FOOTHOLD_INFO | CB | CField | — | **B** (R-CB) | `CField::OnRequestFootHoldInfo` (clientbound op 226; SB request side already exists, see 0.3 §1). |
| FORCED_MAP_EQUIP | CB | CField | — | **B** (R-CB) | `CField::OnFieldSpecificData` (op 133). |
| GENERAL_CHAT | SB | CField | chat/serverbound/general.go (`General`) | **A** (R-MARK) | C-resolved (0.3 §3: dir confirmed SB, op 49 / `0x31`). Codec exists + verified v84/87/95/jms; only v83 `❌`. R-MARK v83; **MOVE chat→field** (see §0.4 chat-relocation). |
| GMEVENT_INSTRUCTIONS | CB | CField | — | **B** (R-CB) | `CField::OnDesc`. |
| GUILD_OPERATION | SB | CField | guild/serverbound/operation*.go (`GuildOperation`, shared-model multi-mode) | **A** (R-WRAP) | C-resolved (0.3 §3: `?`→SB, op 126 / `0x7E`, `CField::InputGuildName` is one mode). Codec in guild family (NOT field/chat), verified v95/jms; v83/84/87 `❌`. R-WRAP + verify; **link-in-place** (do NOT move into field — it is a guild-family shared codec). |
| HORNTAIL_CAVE | CB | CField | — | **B** (R-CB) | `CField::OnHontailTimer` (v83 op 302 / `0x12E`; the `IDA_0X169` mapping is a higher-version shift — see 0.3 §1). |
| IDA_0X098 | CB | CField | — | **version-absent (v83 ⬜)** | C-resolved (0.3 §1): v83 `0x98` = `OnWarnMessage` (ARIANT_RESULT), NOT OnStalkResult. jms-only row → **B where present** (jms), VERSION-ABSENT v83/84/87/95. Stage 1 confirms per-version. |
| IDA_0X09C | CB | CField | — | **B** (R-CB) | C-resolved (0.3 §1): genuinely-unnamed v83 op, `0x9C` / 156, fname `CField::OnStalkResult` @ `0x537a6a`. Keeps op-key (cross-version matrix row). Net-new clientbound codec. |
| IDA_0X09D | CB | CField | — | **version-absent (v83 ⬜)** | C-resolved (0.3 §1): `OnRequestFootHoldInfo`/`case 0x9D` absent in v83. jms-only → **B where present**, VERSION-ABSENT elsewhere. |
| IDA_0X0A4 | CB | CField | — | **version-absent (v83 ⬜)** | C-resolved (0.3 §1): v87-only row (`0xA0–0xEB`→CUserPool in v83). **B where present** (v87), VERSION-ABSENT elsewhere. |
| IDA_0X0AA | CB | CField | — | **version-absent (v83 ⬜)** | C-resolved (0.3 §1): v87/jms-only. **B where present**, VERSION-ABSENT elsewhere. |
| IDA_0X0AC | CB | CField | — | **version-absent (v83 ⬜)** | C-resolved (0.3 §1): v95/jms-only. **B where present**, VERSION-ABSENT elsewhere. |
| IDA_0X0B0 | CB | CField | — | **version-absent (v83 ⬜)** | C-resolved (0.3 §1): v95-only. **B where present**, VERSION-ABSENT elsewhere. |
| IDA_0X0B1 | CB | CField | — | **version-absent (v83 ⬜)** | C-resolved (0.3 §1): v95-only. **B where present**, VERSION-ABSENT elsewhere. |
| IDA_0X169 | CB | CField | — | **version-absent (v83 ⬜)** | C-resolved (0.3 §1): `OnHontailTimer` exists in v83 but routes at `0x12E` (HORNTAIL_CAVE), not `0x169`. v95/jms-only opcode-table shift. **B where present**, VERSION-ABSENT v83/84/87. |
| MATCH_TABLE | SB | CField | — | **B** (R-SB) | `CField::SendChatMsgSlash` slash family. |
| MTS_OPERATION | CB | CITC (CField forwarder) | — | **B** (R-CB) | C-resolved (0.3 §2): distinct packet, op 348 / `0x15C`, `CITC::OnNormalItemResult`. Two distinct B-rows (NOT two modes). jms VERSION-ABSENT (⬜). |
| MTS_OPERATION2 | CB | CITC (CField forwarder) | — | **B** (R-CB) | C-resolved (0.3 §2): distinct packet, op 347 / `0x15B`, `CITC::OnQueryCashResult`. jms VERSION-ABSENT (⬜). |
| MULTICHAT | CB | CField | chat/clientbound/multi.go (`MultiChat`, NAME `CharacterMultiChat`) | **A** (R-MARK after MOVE) | **CORRECTED** (was erroneously B): `MultiChat{mode,from,message}` IS the `CField::OnGroupMessage` codec — registry MULTICHAT.fname = `CField::OnGroupMessage` (op 134 / `0x86`), and OnGroupMessage is exactly `mode + from + message`. MOVE chat→field (Task 2.1.2) then R-MARK; **no new codec**. Stage 1 confirms the per-version byte layout equals the existing codec's; any divergence is a wire-fix commit before marking. |
| OX_QUIZ | CB | CField | — | **B** (R-CB) | `CField::OnQuiz`. |
| PLAY_JUKEBOX | CB | CField | — | **B** (R-CB) | `CField::OnPlayJukeBox`. |
| SET_OBJECT_STATE | CB | CField | — | **B** (R-CB) | `CField::OnSetObjectState`. |
| SET_QUEST_CLEAR | CB | CField | — | **B** (R-CB) | `CField::OnSetQuestClear`. |
| SET_QUEST_TIME | CB | CField | — | **B** (R-CB) | `CField::OnSetQuestTime`. |
| SLIDE_REQUEST | SB | CField | — | **B** (R-SB) | `CField::SendChatMsgSlash`; v83/84/87 VERSION-ABSENT (⬜), v95/jms present. |
| SPOUSE_CHAT | CB | CField | — | **B** (R-CB) | `CField::OnCoupleMessage` (op 136); jms VERSION-ABSENT (⬜). |
| STOP_CLOCK | CB | CField | — | **B** (R-CB) | `CField::OnDestroyClock` — distinct from field/clientbound/clock.go (`Clock`); separate destroy op. |
| SUE_CHARACTER | SB | CField | — | **B** (R-SB) | `CField::SendChatMsgSlash`; jms VERSION-ABSENT (⬜). |
| SUMMON_ITEM_INAVAILABLE | CB | CField | — | **B** (R-CB) | `CField::OnSummonItemInavailable` (op 137). |
| USE_DOOR | SB | CField | — | **B** (R-SB) | C-resolved (0.3 §3: `?`→SB, op 133 / `0x85`, `CField::TryEnterTownPortal`). Net-new serverbound codec. |
| VICIOUS_HAMMER | CB | CField | — | **B** (R-CB) | `CField::OnItemUpgrade`; jms VERSION-ABSENT (⬜). |
| WHISPER (CB) | CB | CField | chat/clientbound/whisper.go (`Whisper*`, NAME `CharacterChatWhisper`) | **A** (R-MARK after MOVE) | **CORRECTED** (was erroneously B): the clientbound `CField::OnWhisper` op (op 135 / `0x87`) is mode-dispatched, and whisper.go already holds those modes (`WhisperReceive`, `WhisperFindResult*`, etc.) under NAME `CharacterChatWhisper`. MOVE chat→field (Task 2.1.3) then R-MARK; **no new codec**. Stage 1 confirms the per-version layout / mode set against the existing codec; any divergence is a wire-fix commit before marking. |
| WHISPER (dup) | CB | CField | — | **(display dupe — folds into WHISPER (CB))** | C-resolved (0.3 §5): cfield-ops.md lines 48–49 are the same clientbound row emitted twice. No separate codec/row. One CB (op 135) + one SB (op 120) WHISPER is correct. |
| WITCH_TOWER_SCORE_UPDATE | CB | CField | — | **B** (R-CB) | `CField::OnChaosZakumTimer`. |
| ZAKUM_SHRINE | CB | CField | — | **B** (R-CB) | `CField::OnZakumTimer`. |

### CField_SnowBall (6 ops)

| op | dir | owner class | existing codec path | classification | resolution note |
|----|-----|-------------|---------------------|----------------|-----------------|
| HIT_SNOWBALL | CB | CField_SnowBall | — | **B** (R-CB) | `CField_SnowBall::OnSnowBallHit`. |
| LEFT_KNOCKBACK | SB | CField_SnowBall | — | **B** (R-SB) | C-resolved (0.3 §4: `?`→SB, op 212 / `0xD4`, `CField_SnowBall::Update`, empty body). |
| LEFT_KNOCK_BACK | CB | CField_SnowBall | — | **B** (R-CB) | `CField_SnowBall::OnSnowBallTouch` (distinct op from LEFT_KNOCKBACK). |
| SNOWBALL | SB | CField_SnowBall | — | **B** (R-SB) | C-resolved (0.3 §4: `?`→SB, op 211 / `0xD3`, `CField_SnowBall::BasicActionAttack`). |
| SNOWBALL_MESSAGE | CB | CField_SnowBall | — | **B** (R-CB) | `CField_SnowBall::OnSnowBallMsg`. |
| SNOWBALL_STATE | CB | CField_SnowBall | — | **B** (R-CB) | `CField_SnowBall::OnSnowBallState`. |

### CField_Tournament (5 ops)

| op | dir | owner class | existing codec path | classification | resolution note |
|----|-----|-------------|---------------------|----------------|-----------------|
| TOURNAMENT | CB | CField_Tournament | — | **B** (R-CB) | `CField_Tournament::OnTournament`. GMS-event-only; Stage 1 may flag VERSION-ABSENT per version. |
| TOURNAMENT_CHARACTERS | CB | CField_Tournament | — | **B** (R-CB) | `CField_Tournament::OnPacket`. |
| TOURNAMENT_MATCH_TABLE | CB | CField_Tournament | — | **B** (R-CB) | `CField_Tournament::OnTournamentMatchTable`. |
| TOURNAMENT_SET_PRIZE | CB | CField_Tournament | — | **B** (R-CB) | `CField_Tournament::OnTournamentSetPrize`. |
| TOURNAMENT_UEW | CB | CField_Tournament | — | **B** (R-CB) | `CField_Tournament::OnTournamentUEW`. |

### CField_Wedding (4 ops)

| op | dir | owner class | existing codec path | classification | resolution note |
|----|-----|-------------|---------------------|----------------|-----------------|
| WEDDING_ACTION | CB | CField_Wedding | — | **B** (R-CB) | `CField_Wedding::OnWeddingProgress`; jms VERSION-ABSENT (⬜). |
| WEDDING_CEREMONY_END | CB | CField_Wedding | — | **B** (R-CB) | `CField_Wedding::OnWeddingCeremonyEnd`. |
| WEDDING_PROGRESS | CB | CField_Wedding | — | **B** (R-CB) | `CField_Wedding::OnWeddingProgress`. |
| WEDDING_TALK | CB | CField_Wedding | — | **B** (R-CB) | `CField_Wedding::OnWeddingProgress`; jms VERSION-ABSENT (⬜). |

### CField_Coconut (3 ops)

| op | dir | owner class | existing codec path | classification | resolution note |
|----|-----|-------------|---------------------|----------------|-----------------|
| COCONUT | SB | CField_Coconut | — | **B** (R-SB) | C-resolved (0.3 §4: `?`→SB, op 213 / `0xD5`, `CField_Coconut::BasicActionAttack`). |
| COCONUT_HIT | CB | CField_Coconut | — | **B** (R-CB) | `CField_Coconut::OnCoconutHit`. |
| COCONUT_SCORE | CB | CField_Coconut | — | **B** (R-CB) | `CField_Coconut::OnCoconutScore`. |

### CField_GuildBoss (3 ops)

| op | dir | owner class | existing codec path | classification | resolution note |
|----|-----|-------------|---------------------|----------------|-----------------|
| GUILD_BOSS | SB | CField_GuildBoss | — | **B** (R-SB) | C-resolved (0.3 §4: `?`→SB, op 215 / `0xD7`, `CField_GuildBoss::BasicActionAttack`, empty body). |
| GUILD_BOSS_HEALER_MOVE | CB | CField_GuildBoss | — | **B** (R-CB) | `CField_GuildBoss::OnHealerMove`. |
| GUILD_BOSS_PULLEY_STATE_CHANGE | CB | CField_GuildBoss | — | **B** (R-CB) | `CField_GuildBoss::OnPulleyStateChange`. |

### CField_ContiMove (2 ops)

| op | dir | owner class | existing codec path | classification | resolution note |
|----|-----|-------------|---------------------|----------------|-----------------|
| CONTI_MOVE (Init) | — | CField_ContiMove | — | **(spurious — folds into CONTI_MOVE)** | C-resolved (0.3 §4): `CField_ContiMove::Init` is inlined map-init, NOT a packet send-site. No serverbound conti-move op. Duplicates the clientbound CONTI_MOVE; no new row/codec. v95 column ⬜. |
| CONTI_MOVE (OnContiMove) | CB | CField_ContiMove | — | **B** (R-CB) | `CField_ContiMove::OnContiMove` (op 148 / `0x94`). Net-new clientbound codec. |

### CField_AriantArena (2 ops)

| op | dir | owner class | existing codec path | classification | resolution note |
|----|-----|-------------|---------------------|----------------|-----------------|
| ARIANT_ARENA_SHOW_RESULT | CB | CField_AriantArena | — | **B** (R-CB) | `CField_AriantArena::OnShowResult`. |
| ARIANT_ARENA_USER_SCORE | CB | CField_AriantArena | — | **B** (R-CB) | `CField_AriantArena::OnUserScore`. |

### CField_Battlefield (2 ops)

| op | dir | owner class | existing codec path | classification | resolution note |
|----|-----|-------------|---------------------|----------------|-----------------|
| SHEEP_RANCH_CLOTHES | CB | CField_Battlefield | — | **B** (R-CB) | `CField_Battlefield::OnTeamChanged`. |
| SHEEP_RANCH_INFO | CB | CField_Battlefield | — | **B** (R-CB) | `CField_Battlefield::OnScoreUpdate`. |

### CField_Massacre / CField_MassacreResult / CField_Witchtower (3 ops)

| op | dir | owner class | existing codec path | classification | resolution note |
|----|-----|-------------|---------------------|----------------|-----------------|
| PYRAMID_GAUGE | CB | CField_Massacre | — | **B** (R-CB) | `CField_Massacre::OnMassacreIncGauge`. |
| PYRAMID_SCORE | CB | CField_MassacreResult | — | **B** (R-CB) | `CField_MassacreResult::OnMassacreResult`. |
| ARIANT_SCORE | CB | CField_Witchtower | — | **B** (R-CB) | `CField_Witchtower::OnPacket`; v83/84/87/jms VERSION-ABSENT (⬜), v95-only present. |

### Classification counts

Counting the 75 work-list rows exactly as enumerated in `cfield-ops.md`
(WHISPER appears twice; CONTI_MOVE appears twice):

| Class | Count | Rows |
|-------|------:|------|
| **A** (codec exists; R-MARK / R-WRAP) | **4** | GENERAL_CHAT (R-MARK after MOVE), **MULTICHAT (R-MARK after MOVE)**, **WHISPER CB (R-MARK after MOVE)**, GUILD_OPERATION (R-WRAP). All four are relocate-or-link existing codecs — **no new codec**. The three chat relocations (Cluster 1) confirm per-version byte layout in Stage 1 before marking. |
| **B** (net-new codec; R-CB / R-SB) | **60** | All remaining real ops (48 CB + 12 SB). Includes the 8 ops C-resolved to a concrete direction in 0.3 §2–§4 (MTS_OPERATION/MTS_OPERATION2, USE_DOOR, SNOWBALL, LEFT_KNOCKBACK, COCONUT, GUILD_BOSS, IDA_0X09C). |
| **version-absent (v83 ⬜)** | **8** | The foothold/stalk cluster minus IDA_0X09C: IDA_0X098, 0X09D, 0X0A4, 0X0AA, 0X0AC, 0X0B0, 0X0B1, 0X169. Each is **B where present** in a higher version; Stage 1 confirms per-version presence with IDB evidence. |
| **spurious / display dupe** (no row, no codec) | **3** | WHISPER (2nd line), CONTI_MOVE (Init), folded per 0.3 §4/§5. *(WHISPER dupe + CONTI_MOVE Init = 2 distinct ops folded; the 3rd is accounting for the duplicate WHISPER line counted in the 75.)* |

Reconciliation against the 75 rows in cfield-ops.md: 4 (A) + 60 (B real) + 8
(v83 version-absent, B-where-present) + 2 fold-outs (WHISPER dup line,
CONTI_MOVE Init) = 74; the WHISPER (CB) real row is one of the 4 A-rows, and the
duplicate WHISPER list line is the 75th. **Distinct implementable units: 4 A-row
(3 relocations + 1 wrapper, no new codec) + ~68 B-row (60 v83-present + 8
higher-version-only), with the WHISPER and CONTI_MOVE duplicate lines
contributing no additional codec.**

No row remains classified **C**.

---

## §0.4 Chat-relocation ownership analysis (drives Cluster 1)

For each chat-family file that the registry links to a `CField::`-owned op,
the test is: does **every** registry op the file's codec NAME serves resolve to
a `CField::*` owner? If yes → **MOVE** the file `chat/ → field/`. If it is shared
with a non-CField op → **link-in-place** (do NOT move).

| File | codec NAME (`Operation()`) | registry op(s) served | owner(s) | verdict | evidence |
|------|---------------------------|------------------------|----------|---------|----------|
| chat/serverbound/general.go | `CharacterChatGeneralHandle` | GENERAL_CHAT (SB, op 49 / `0x31`) | `CField::SendChatMsg` (alt `CField::SendChatMsgSlash`) — both `CField::` | **MOVE chat→field** | gms_v83.yaml:1980–1986 — single serverbound op, fname + alt are both `CField::`-owned. No non-CField op uses this handler. |
| chat/clientbound/multi.go | `CharacterMultiChat` | MULTICHAT (CB, op 134) | `CField::OnGroupMessage` — `CField::` | **MOVE chat→field** | gms_v83.yaml:644–648 — `CField::OnGroupMessage` appears exactly **1×** in the registry (`grep -c` = 1). Single CField-owned op. |
| chat/clientbound/whisper.go | `CharacterChatWhisper` | WHISPER (CB, op 135) | `CField::OnWhisper` — `CField::`; also `fname_alt` of WHISPER (SB, op 120, `CField::SendLocationWhisper`), still `CField::` | **MOVE chat→field** | gms_v83.yaml:649–653 (CB) and 2416–2423 (SB, `CField::OnWhisper` as alt). `CField::OnWhisper` appears 2×, **both `CField::`-owned**; no non-CField owner. |

**Exception (stays in chat/):** `chat/clientbound/world_message.go` and
`chat/clientbound/world_message_extra.go` (NAME `WorldMessage`) are owned by
`CWvsContext::OnBroadcastMsg` (gms_v83.yaml:316) and the recommend-world-message
path `CLogin::OnRecommendWorldMessage` (gms_v83.yaml:136) — **non-CField owners**.
These are not in the CField work-list and **do NOT move**; they remain in `chat/`.

**Verdict summary:** all **3** relocation candidates (general.go, multi.go,
whisper.go) are CField-exclusive → **MOVE** for all three. `world_message*.go`
stays put.

---

## VERSION-ABSENT candidates (flagged for Stage 1 confirmation)

Candidates only — **not** authoritatively `⬜` yet. Stage 1 confirms each with
per-version IDB evidence before any matrix cell is set to `⬜`. Basis is
baseline.md `⬜` cells + 0.3 foothold/stalk findings + GMS-event-only minigames.

| op | version(s) flagged absent | basis |
|----|---------------------------|-------|
| IDA_0X098 | v83, v84, v87, v95 | 0.3 §1 + baseline.md (jms-only; v83 `0x98` = OnWarnMessage). |
| IDA_0X09C | v84, v87, v95 | baseline.md (v83 + jms only). |
| IDA_0X09D | v83, v84, v87, v95 | 0.3 §1 + baseline.md (jms-only; fn absent in v83). |
| IDA_0X0A4 | v83, v84, v95, jms | 0.3 §1 + baseline.md (v87-only). |
| IDA_0X0AA | v83, v84, v95 | 0.3 §1 + baseline.md (v87/jms-only). |
| IDA_0X0AC | v83, v84, v87 | 0.3 §1 + baseline.md (v95/jms-only). |
| IDA_0X0B0 | v83, v84, v87, jms | 0.3 §1 + baseline.md (v95-only). |
| IDA_0X0B1 | v83, v84, v87, jms | 0.3 §1 + baseline.md (v95-only). |
| IDA_0X169 | v83, v84, v87 | 0.3 §1 + baseline.md (v95/jms-only; v83 OnHontailTimer routes at `0x12E`). |
| ARIANT_RESULT | jms | baseline.md (jms `⬜`). |
| MTS_OPERATION | jms | baseline.md (jms `⬜`). |
| MTS_OPERATION2 | jms | baseline.md (jms `⬜`). |
| SLIDE_REQUEST | v83, v84, v87 | baseline.md (v95/jms-only). |
| SPOUSE_CHAT | jms | baseline.md (jms `⬜`). |
| SUE_CHARACTER | jms | baseline.md (jms `⬜`). |
| VICIOUS_HAMMER | jms | baseline.md (jms `⬜`). |
| WEDDING_ACTION | jms | baseline.md (jms `⬜`). |
| WEDDING_TALK | jms | baseline.md (jms `⬜`). |
| CONTI_MOVE (Init) | v83, v84, v87, jms | baseline.md (v95 row only; and 0.3 §4 — spurious even there). |
| ARIANT_SCORE | v83, v84, v87, jms | baseline.md (v95-only). |
| TOURNAMENT (×5 ops) | per-version TBD | GMS-event-only minigame family; Stage 1 to confirm which versions ship the tournament opcode table. |

**Total VERSION-ABSENT candidates flagged: 20 op-entries** (the 9 foothold/stalk
rows + 11 baseline-`⬜`/GMS-event rows above; the Tournament family is flagged
as a block for per-version confirmation).
