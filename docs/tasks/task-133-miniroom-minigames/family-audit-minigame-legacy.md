# family-audit: character_interaction (MEMORY_GAME_* arms + balloon) — legacy version bring-up (gms_v48/61/72/79)

Read-only coverage audit. Scope: the `character_interaction` clientbound writer
(`docs/packets/dispatchers/character_interaction.yaml`, fname
`CMiniRoomBaseDlg::OnPacketBase`), its serverbound twin
(`character_interaction_handle.yaml`), and the balloon codec
(`interaction/clientbound/InteractionMiniRoomBalloon`, fname
`CUser::OnMiniRoomBalloon`) — for the four legacy versions gms_v48, gms_v61,
gms_v72, gms_v79. No codec/registry/template/yaml/evidence file was modified.

**IDA instances used** (confirmed via `list_instances` before reading):
v48=13337 (`GMS_v48_1_DEVM.exe`), v61=13338 (`GMS_v61.1_U_DEVM.exe`),
v72=13339 (`GMS_v72.1_U_DEVM.exe`), v79=13340 (`GMS_v79_1_DEVM.exe`).
v61/v72 clientbound OnPacket vtable-dispatch was **not** decompiled in this
pass (see §6 "Not done in this pass") — budget went to a full v48 derivation
(the hardest case, oldest client, no symbols) plus the yaml/registry/template
cross-checks that apply uniformly to all four.

## 0. Headline finding — the parent (v83–jms) clientbound arms aren't in the matrix either

Before the legacy-specific work-list: **none of the 11 clientbound
`MEMORY_GAME_*` arms have a `packet-audit:verify` marker for *any* version**,
including gms_v83/v95 which the task description says were "built."

- `run.go` has all 11 `#`-suffixed candidates (`CMiniRoomBaseDlg::OnPacketBase#MemoryGameReady`
  … `#MemoryGameResult`, `#EnterResultSuccessMiniGame`, `#EnterMiniGame`) —
  confirmed at `tools/packet-audit/cmd/run.go:2092-2143`.
- The structs exist (`libs/atlas-packet/interaction/clientbound/interaction_minigame.go`,
  `interaction_minigame_enter.go`, `interaction_minigame_room.go`).
- Audit **report files** exist on disk for gms_v83 and gms_v95 only
  (`docs/packets/audits/gms_v83/InteractionInteractionMiniGame*.{md,json}`,
  same for gms_v95) — none for v84/v87/jms_v185, none for the legacy four.
- But `grep -rn "packet-audit:verify" libs/atlas-packet/interaction/clientbound/*_test.go`
  returns **zero** hits for any `InteractionMiniGame*` packet name, on any
  version. Compare: the balloon (`InteractionMiniRoomBalloon`) and the base
  arms (`InteractionInteraction{Invite,Enter,Leave,Chat,UpdateMerchant}`) DO
  have markers.
- Consequence: `docs/packets/audits/status.json` has **no rows at all** for
  any `InteractionMiniGame*` packet (verified by grepping the full file — zero
  matches). The op-row aggregate the matrix shows for `PLAYER_INTERACTION`
  clientbound (`interaction/clientbound/InteractionInteractionChat`, all 9
  versions "verified") does **not** cover the mini-game arms — it is a
  different packet representing the base dispatcher only.

**Implication for the legacy bring-up ask:** there is no "5-version baseline"
to extend into 4 more columns in the matrix sense — the mini-game clientbound
family has never been wired into the verify pipeline (no markers → no rows →
no matrix cells) for *any* version. The legacy bring-up and the "finish v83
family verification" work are the same missing step (per-mode `#`-entry +
byte-fixture + marker), just needing 9 columns instead of 4. Recommendation
#1 below is to do the parent-version markers first, since the legacy
byte-fixtures will want the same test-file scaffolding.

## 1. Per-arm × per-version clientbound coverage

`character_interaction.yaml` currently defines `modes:` only for
`{gms_v83, gms_v84, gms_v87, gms_v95, jms_v185}` — **zero columns for
gms_v48/61/72/79** (confirmed: the file has no `gms_v48:`/`gms_v61:`/`gms_v72:`/`gms_v79:`
key anywhere in its `operations:` block). Per DISPATCHER_FAMILY.md's
"n-a" rule (registry-absent ⇒ n-a), that is **not** automatically true here —
the registry and the IDA evidence gathered below show the feature is
**present**, just unregistered, for all four legacy versions except the
balloon on v48 (unresolved, not confirmed n-a — see §3).

| Arm (yaml key) | gms_v48 | gms_v61 | gms_v72 | gms_v79 |
|---|---|---|---|---|
| MEMORY_GAME_ASK_TIE | 47 (Omok, IDA 0x5731a9 case, tentative label — see §2.3) | unresolved | unresolved | unresolved |
| MEMORY_GAME_TIE_ANSWER | 44 (bodyless, tentative) | unresolved | unresolved | unresolved |
| MEMORY_GAME_ASK_RETREAT | 43 (Omok, tentative) | unresolved | unresolved | unresolved |
| MEMORY_GAME_RETREAT_ANSWER | 48 (accept+stoneCount+turnSlot body match) | unresolved | unresolved | unresolved |
| MEMORY_GAME_READY | 51 (bodyless, both dialogs) | unresolved | unresolved | unresolved |
| MEMORY_GAME_UNREADY | 52 (bodyless, both dialogs) | unresolved | unresolved | unresolved |
| MEMORY_GAME_START | 54 (Omok firstMover byte / MemoryGame firstMover+deck) | unresolved | unresolved | unresolved |
| MEMORY_GAME_RESULT | 55 **or** 56 (ambiguous — see §2.3) | unresolved | unresolved | unresolved |
| MEMORY_GAME_SKIP | 56 **or** 55 (ambiguous — see §2.3) | unresolved | unresolved | unresolved |
| MEMORY_GAME_MOVE_STONE | 57 (Omok only, DecodeBuffer(8)+Decode1 = x,y,stoneType — high confidence) | unresolved | unresolved | unresolved |
| MEMORY_GAME_FIP_CARD | 61 (MemoryGame only, dual-branch card-select body — high confidence) | unresolved | unresolved | unresolved |
| *(extra, untracked)* | **58** — Omok-only, `Decode1`+message-picker (`==60` compare, resources 458/459); does not match any of the 11 current keys. Possibly EXIT_AFTER_GAME/FORFEIT-notification. Not modeled by Atlas at all. | — | — | — |

Every v48 number above is address-cited in §2. "unresolved" for v61/v72/v79
means: **not attempted in this pass** (budget), not "confirmed absent" — see
§4 for why absence is very unlikely (the serverbound codecs already verify
against real IDA addresses in COmokDlg/CMemoryGameDlg in all three).

### 1.1 Serverbound (`character_interaction_handle.yaml`) — for comparison

This file (the *handler*, opposite direction) already has full gms_v79
columns for all 14 MEMORY_GAME_* keys (lines 60-73), IDA-verified. It has
**zero** gms_v48/v61/v72 columns for any MEMORY_GAME_* key — the enum simply
stops at `MERCHANT_REMOVE_ITEM` (mode 36/38) for those three versions'
columns. But per §4, the four serverbound MEMORY_GAME sub-structs
(FlipCard/MoveStone/RetreatAnswer/TieAnswer) are **already verified** with
real IDA addresses for v48, v61, AND v72 — so this yaml is stale/incomplete
relative to evidence that already exists in the test files. Concrete modes
recovered from the verified fixtures:

| Key | gms_v48 | gms_v61 = gms_v72 | gms_v79 (yaml, already present) |
|---|---|---|---|
| MEMORY_GAME_RETREAT_ANSWER | 44 (0x2C) — `libs/atlas-packet/interaction/serverbound/operation_memory_game_retreat_answer_test.go` v48 marker `ida=0x573a54` | 45 (0x2D) — marker `ida=0x5f7ba3` (v61) / `ida=0x5febf2` (v72) | 54 |
| MEMORY_GAME_TIE_ANSWER | 48 (0x30) — marker `ida=0x573b11` | 49 (0x31) — marker `ida=0x5f7c60` (v61) / `ida=0x64e893` (v72) | 51 |
| MEMORY_GAME_MOVE_STONE | 57 (0x39) — marker `ida=0x578388` | 58 (0x3A) — marker `ida=0x5fc4d7` (v61) / `ida=0x65320c` (v72) | 63 |
| MEMORY_GAME_FIP_CARD | 61 (0x3D) — marker `ida=0x53875d` | 62 (0x3E) — marker `ida=0x5b10fa` (v61) / `ida=0x5ff6ba` (v72) | 67 |

These four rows are a **direct, mechanical yaml-population fix** (the
addresses and byte values are already sitting in the committed test files —
no new IDA work needed to add these four keys × three versions to
`character_interaction_handle.yaml`). This is Recommendation #2.

## 2. v48 clientbound derivation (full detail)

v48's `CMiniRoomBaseDlg::OnPacketBase`-equivalent is a size/type factory,
**not** a single named class hierarchy (the IDB has no `COmokDlg`/
`CMemoryGameDlg` symbols — `func_query name_regex ".*Omok.*"` and
`".*MemoryGame.*"` both returned zero hits). The dialog-type factory
`MiniRoom_CreateDlgByType` (0x5458cc) resolves the mini-room `roomType` byte
to five allocator+ctor pairs:

```
type 1 -> Alloc(2752) -> ctor sub_572B03   (vtable off_79F998, dlg resource 1418)
type 2 -> Alloc(2004) -> ctor sub_536BE5   (vtable off_79F598, dlg resource 1419)
type 3 -> Alloc(1680) -> CTradingRoomDlg_ctor_type3
type 4 -> Alloc(1804) -> CPersonalShopDlg_ctor_type4
type 5 -> Alloc(1836) -> CEntrustedShopDlg_ctor_type5
```

Types 3/4/5 are already-named (trade/personal-shop/entrusted-shop). Types
1/2 are the two mini-games by elimination. Base dispatcher
`CMiniRoomBaseDlg_OnClientbound_op239` (0x5459c4, **clientbound opcode 239,
unregistered — see §5**) default-cases into `*vtable[15]` (offset 60 bytes)
of whichever dialog is active — the per-mode game-dispatch slot. Reading that
vtable slot for each type:

- type-1 vtable slot @ `off_79F998+60` = **0x5731A9** ("Omok" — confirmed by
  mode-57 body matching `COmokDlg::PutStoneChecker`'s serverbound MoveStone
  shape, see below)
- type-2 vtable slot @ `off_79F598+60` = **0x5374C8** ("MemoryGame" — mode-61
  body matches the card-select-first/second dual-branch shape)

### 2.1 Type 1 (Omok) switch @ 0x5731a9 — 11 cases

| case | addr | body (Decode calls) | label |
|---|---|---|---|
| 43 (0x2B) | 0x573a54 | none; YesNo() prompt then **sends** serverbound `Encode1(0x2C)` | ASK_RETREAT (bodyless "ask") — confirmed via serverbound cross-ref: `InteractionOperationMemoryGameRetreatAnswer` v48 marker is `ida=0x573a54`, wire mode 0x2C=44 |
| 44 (0x2C) | 0x573ae1 | none; resource-string popup (id 444) only | TIE_ANSWER (bodyless notif) — by elimination + struct match (`InteractionMiniGameAnswerTie{mode byte}`, bodyless in `interaction_minigame.go:76`) |
| 47 (0x2F) | 0x573b11 | none; YesNo() prompt then **sends** serverbound `Encode1(0x30)` | ASK_TIE (bodyless "ask") — cross-ref: `InteractionOperationMemoryGameTieAnswer` v48 marker is `ida=0x573b11`, wire mode 0x30=48 |
| 48 (0x30) | 0x573b9e | `Decode1`(accept flag) + loop `Decode1`(stoneCount) + `Decode1`(turnSlot) | RETREAT_ANSWER (accept+stoneCount+turnSlot body) — matches `InteractionMiniGameRetreatAnswer{mode,accept,stoneCount,turnSlot}` field list exactly |
| 51 (0x33) | 0x573fb9 | none; toggles `this[674]=1`, calls shared `sub_57720A` | READY (bodyless) |
| 52 (0x34) | 0x57401f | none; toggles `this[674]=0`, calls same `sub_57720A` | UNREADY (bodyless) |
| 54 (0x36) | 0x57404d | `Decode1` (compared to `this[48]`=local turn slot) | START (firstMover byte) — matches `InteractionMiniGameStartOmok{mode,firstMover}` |
| 55 (0x37) | 0x573e1d | `Decode1`(outcome, special-cased ==1) + conditional `Decode1`(who) | RESULT **or** SKIP — see §2.3, ambiguous |
| 56 (0x38) | 0x5740df | `Decode1` (compared to `this[48]`) | SKIP **or** RESULT — see §2.3, ambiguous |
| 57 (0x39) | 0x57390a | `DecodeBuffer(8)` (x,y int32×2) + `Decode1`(stoneType) | MOVE_STONE — matches `InteractionMiniGameMoveStone{mode,x,y,stoneType}` exactly, high confidence |
| 58 (0x3A) | 0x573a10 | `Decode1`(byte, compared `==60`) then resource-picker (458/459) | **untracked extra arm** — not in the current 11-key set |

### 2.2 Type 2 (MemoryGame/Match Cards) switch @ 0x5374c8 — 8 cases

| case | addr | body | label |
|---|---|---|---|
| 43 (0x2B) | 0x537d1c | none; YesNo() then sends `Encode1(0x2C)` | mirrors type-1's case 43 exactly (byte-identical shape) |
| 44 (0x2C) | 0x537da9 | none; resource popup (444) | mirrors type-1's case 44 |
| 51 (0x33) | 0x537f4a | none; toggles `this[444]=1` | READY |
| 52 (0x34) | 0x537fb0 | none; toggles `this[444]=0` | UNREADY |
| 54 (0x36) | 0x537fde | `Decode1`(firstMover) + `Decode1`(deckSize) + `DecodeBuffer(4×deckSize)` | START — matches `InteractionMiniGameStartMatchCards{mode,firstMover,deck []uint32}` exactly |
| 55 (0x37) | 0x537dd9 | same outcome/who shape as type-1 case 55 | RESULT/SKIP — same ambiguity as §2.3 |
| 56 (0x38) | 0x5380c8 | `Decode1` + clears card-selection slots (`this[455]`/`this[456]`), resets 10s timer | RESULT/SKIP — same ambiguity |
| 61 (0x3D) | 0x537b7e | `Decode1`(flag) + `Decode1`(slot) + conditional 3rd `Decode1`(compare) | FIP_CARD — dual-branch (first-select vs second-select-with-match-check) matches the collapsed `MEMORY_GAME_FIP_CARD` key (covers both `CardSelectFirst`/`CardSelectSecond` per `run.go`) |

Type 2 has **no** 47/48/57/58 cases — no ASK_TIE/RETREAT_ANSWER/MOVE_STONE
counterpart, i.e. type-2's 43/44 pair (not type-1's 47/48 pair) is its ONLY
ask/notify pair. This means MemoryGame's mode-43/44 pair, despite sending the
same wire bytes as Omok's mode-43/44 pair, is structurally its **only**
special-request flow — consistent with Match Cards having no "retreat/undo
move" mechanic (cards, unlike stones, aren't physically un-placed).

### 2.3 Unresolved: RESULT vs SKIP at {55,56}, and case 58

Both dialog types have a `Decode1`+outcome-branch case (55/type1,
55/type2) and a `Decode1`+`this[48]`-compare case (56/type1, 56/type2) that
match the *shape* of RESULT (win/lose/tie determination) and SKIP (who's
turn was skipped) respectively, but I could not disambiguate which literal
number is which key with certainty in this pass — the modern (v83+) RESULT
struct (`InteractionMiniGameResult{mode,resultType,visitorWon,ownerRecord,
visitorRecord}`) carries two 20-byte `GW_MiniGameRecord` blobs that neither
v48 case reads, meaning v48's RESULT wire shape is simpler than v83's (a
plausible feature-evolution, but unverified) — so body-shape alone can't
rule this in. **Recommendation:** a `packet-verifier` pass on these two
addresses (0x573e1d / 0x5740df for Omok; 0x537dd9 / 0x5380c8 for MemoryGame)
should pull the referenced UI string resource IDs (438/439/1441/1442/1443 for
one, none for the other) via `GetBSTR`/`StringPool` lookups to read the
actual English text and settle RESULT vs SKIP definitively before writing
byte fixtures.

Case 58 (Omok-only, addr 0x573a10) doesn't match any of the current 11 keys.
Given the *serverbound* handler yaml (`character_interaction_handle.yaml`)
tracks `MEMORY_GAME_EXIT_AFTER_GAME`/`MEMORY_GAME_CANCEL_EXIT_AFTER_GAME`/
`MEMORY_GAME_FORFEIT`/`MEMORY_GAME_EXPEL` as additional serverbound-only keys
with no clientbound counterpart in `character_interaction.yaml` today, case
58 is a plausible clientbound notification for one of those (its
`Decode1(byte)==60` comparison and dual message resources look like a
binary "did they accept the exit" outcome). Flagged as **unresolved — needs
a dedicated arm**, not silently dropped.

## 3. Balloon (`UPDATE_CHAR_BOX` / `CUser::OnMiniRoomBalloon`)

Registry-confirmed opcodes (all `direction: clientbound`, `fname:
CUser::OnMiniRoomBalloon`):

| Version | Registry opcode | status.json state | Notes |
|---|---|---|---|
| gms_v48 | **absent** — no `UPDATE_CHAR_BOX` entry anywhere in `docs/packets/registry/gms_v48.yaml` | `n-a` | See below — this is an *inherited* n-a from a registry gap, not a fresh IDA-confirmed absence. |
| gms_v61 | 124 (`docs/packets/registry/gms_v61.yaml:848`) | `incomplete`, note "no audit report" | function not IDA-searched by name in this pass |
| gms_v72 | 150 (`gms_v72.yaml:1001`) | `incomplete`, note "no audit report" | same |
| gms_v79 | 154 (`gms_v79.yaml:1017`) | `incomplete`, note "no audit report" | `CUser::OnMiniRoomBalloon` **is** a named function in the v79 IDB at **0x8922ce** (confirmed via `func_query name_regex ".*MiniRoom.*"`), matching the task's stated grounding |

**v48 n-a is unconfirmed, not verified-absent.** `func_query` for
`.*MiniRoom.*` in the v48 IDB (port 13337) returned 9 hits (the base
dispatcher family + `MiniRoom_CreateDlgByType`) and **none** named or
resembling a balloon handler — but this is a name-search negative, not a
structural one (v48's dialog classes aren't named either, and I only found
`MiniRoom_CreateDlgByType`/`COmokDlg`/`CMemoryGameDlg`-equivalents by
walking the type-factory + vtable, which a plain name search would also have
missed). A rough opcode-delta heuristic (`PLAYER_INTERACTION_clientbound −
UPDATE_CHAR_BOX` = 120 on v61, 130 on v72, 138 on v79 — not monotonic enough
to extrapolate a v48 number with confidence) was **not** used to assert a
value; it's noted here only to justify why "confirm via `CUser::OnPacket`
opcode-switch walk" (not "assume n-a") is the right next step.

No `packet-audit:verify` marker or audit report exists for the balloon on
any of the four legacy versions (confirmed: `grep packet-audit:verify
mini_room_balloon_test.go` shows only gms_v83/gms_v95 markers).

## 4. Serverbound MEMORY_GAME_* codecs — already verified, all four versions

Contrary to the framing that legacy bring-up is greenfield: the four
serverbound sub-structs (`InteractionOperationMemoryGame{FlipCard,MoveStone,
RetreatAnswer,TieAnswer}`) are **already verified with real IDA addresses**
for gms_v48, gms_v61, gms_v72, AND gms_v79 (`status.json` cells:
`"state":"verified"` for all four, matching live `packet-audit:verify`
markers in `libs/atlas-packet/interaction/serverbound/*_test.go` — addresses
quoted in §1.1). This directly proves Omok/MemoryGame **are present and
functional in the client at all four legacy versions** — the earlier "is
this feature even version-absent" question is settled: it is not.

The stale `docs/packets/audits/gms_v48/InteractionOperationMemoryGame*.md`
report files (verdict 🚫/⚠️, "function not found in IDB") are **outdated
artifacts** predating the address being pinned in the test file's
`packet-audit:verify ida=` marker — they contradict the current, correct
test-file evidence and should be regenerated, not trusted. (Not fixed here —
read-only.)

## 5. Registry gaps

- **gms_v48 clientbound `PLAYER_INTERACTION` opcode 239 is unregistered.**
  `docs/packets/registry/gms_v48.yaml` has only the *serverbound* row
  (opcode 93, line 1013). The clientbound dispatcher function
  `CMiniRoomBaseDlg_OnClientbound_op239` is a **named** function in the v48
  IDB at **0x5459c4** (decompiled in this pass, §2) — the registry is simply
  missing the row, not the feature. This blocks the whole legacy clientbound
  family from being registered at all (yaml `opcodes:` map keys off the
  registry).

## 6. Operations-table cross-check (seed templates)

For every legacy version, `writer.options.operations` (CharacterInteraction,
clientbound) has **zero** `MEMORY_GAME_*` keys — confirmed by dumping
`socket.writers[].options.operations` for `template_gms_{48,61,72,79}_1.json`:
only `INVITE/INVITE_RESULT/ENTER/ENTER_RESULT/CHAT/CHAT_THING/LEAVE/
UPDATE_MERCHANT/PERSONAL_STORE_ITEM_SOLD` are present in all four.

For `handler.options.operations` (CharacterInteractionHandle, serverbound):

| Version | MEMORY_GAME_* keys present? |
|---|---|
| gms_v48 | **0 of 14** — table stops at `TRADE_CONFIRM:16` |
| gms_v61 | **0 of 14** — table stops at `MERCHANT_REMOVE_ITEM:36` |
| gms_v72 | **0 of 14** — identical to v61 |
| gms_v79 | **14 of 14** — full set present (`ASK_TIE:50` … `FIP_CARD:67`), matches `character_interaction_handle.yaml` exactly |

This means: **v48/v61/v72's already-verified serverbound MEMORY_GAME codecs
(§4) cannot correctly resolve a mode byte in production today** — the
tenant `operations` table those bodies must resolve against
(`atlas_packet.WithResolvedCode("operations", KEY, …)` per
DISPATCHER_FAMILY.md) has no entry for any MEMORY_GAME_* key on those three
versions, so `ResolveCode` would fall through to its 99-on-miss default. This
is a **live wiring gap**, not just a documentation gap — byte-correct codecs
that can never emit the right byte in a real tenant.

## 7. Divergence notes (yaml-header-flagged)

`character_interaction.yaml`'s header (lines 44-53) documents the
jms_v185 uniform −3 shift for MEMORY_GAME_* modes and the v83=v84=v87=v95
stability claim — both are IDA-cited for the built versions and out of scope
for this legacy audit (no jms/gms_83+ IDA work was done here). Nothing in
this pass contradicts those notes.

## 8. Not done in this pass

- **v61/v72 clientbound OnPacket vtable walk** was not performed (only v48's
  was, in full). Given v61=v72 for every other arm in this family (per the
  yaml header and the serverbound evidence: identical serverbound mode bytes
  62/58/45/49 for both), it is likely v61=v72 for the clientbound arms too,
  but this is an assumption, not IDA-confirmed — flag before trusting it.
- **v79 clientbound OnPacket vtable walk** was not performed. v79's
  `CMiniRoomBaseDlg::OnPacketBase` base dispatcher was re-confirmed
  (0x62cd21, default-dispatches via vtable+64 — note the **offset differs
  from v48's vtable+60**, a real per-version vtable-layout divergence any
  implementer must re-derive, not assume). The two game-dialog vtables and
  their mode switches were not located (the `dword_B07840` "current dialog"
  global is reused by ~40 unrelated dispatchers in v79, making the v48-style
  xref approach useless there; v79's dialog-factory equivalent to v48's
  `MiniRoom_CreateDlgByType` needs to be found via `OnInviteStatic`
  (0x62cdc8) or `OnEnterResultStatic` (0x62d78f) call graphs instead).
- RESULT vs SKIP disambiguation for v48 (§2.3) and case 58's identity (§2.3)
  are open.
- v48 balloon presence/absence (§3) is open.

## 9. Recommendations (ordered; do NOT execute here)

1. **Wire the parent versions first.** Before extending 4 more columns,
   give the 11 clientbound `MEMORY_GAME_*` `#`-entries `packet-audit:verify`
   markers + byte fixtures for gms_v83/v84/v87/v95/jms_v185 (the audit
   reports already exist for v83/v95 — turn those into fixtures via
   `packet-verifier`). This is what actually makes the family appear in
   `status.json` at all; the legacy work reuses the same fixture scaffolding.
2. **Mechanical yaml fix (no new IDA work):** add the four already-verified
   serverbound rows (RETREAT_ANSWER/TIE_ANSWER/MOVE_STONE/FIP_CARD) to
   `character_interaction_handle.yaml` for gms_v48/v61/v72, using the exact
   mode bytes and IDA addresses already committed in
   `libs/atlas-packet/interaction/serverbound/*_test.go` (table in §1.1).
   Follow [`RE_AUDITING_A_COLUMN.md`](../../docs/packets/RE_AUDITING_A_COLUMN.md)
   trigger 1 to confirm each byte against the live IDB before committing
   (the addresses are already pinned, so this is a confirm-and-transcribe
   pass, not fresh derivation).
3. **Populate the operations tables** (§6) for `handler` (v48/61/72,
   14 keys) and `writer` (all four, 11 keys, blocked on step 1's clientbound
   markers landing first) in all four `template_gms_*_1.json` files — this is
   what actually unblocks production emission of the already-verified
   serverbound codecs (§4's "live wiring gap").
4. **Register gms_v48 clientbound opcode 239** (§5) as `PLAYER_INTERACTION`
   in `docs/packets/registry/gms_v48.yaml`, citing
   `CMiniRoomBaseDlg_OnClientbound_op239` @0x5459c4.
5. Dispatch a `dispatcher-family-implementer` pass for the v48 clientbound
   11-arm set using the case/address table in §2.1/§2.2 as the starting
   derivation — but first resolve §2.3 (RESULT/SKIP swap risk) via a
   `packet-verifier` sub-pass that pulls the UI string resources at 438/439/
   444/1441/1442/1443/458/459 to disambiguate before locking in byte
   fixtures.
6. Repeat the v48-style vtable walk (§2) for v61/v72 (verify the "v61=v72,
   and v48-derived case *numbers* likely also apply since serverbound modes
   only differ by a flat +1" hypothesis) and for v79 (needs a fresh
   dialog-factory search per §8 — the `dword_B07840` xref approach doesn't
   work there).
7. Confirm v48 balloon presence/absence directly (§3) via a `CUser::OnPacket`
   opcode-switch walk rather than a name search, before committing to `n-a`
   in a future yaml/registry edit.
