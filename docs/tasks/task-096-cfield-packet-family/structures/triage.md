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
