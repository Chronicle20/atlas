# Accepted permanent exclusions — packet audit (four-version registry)

> **Closeout of record: task-080 (packet-audit-closeout).** This file is a
> *registry*, not a deferral ledger. It holds **zero actionable items**. Every
> entry is a blessed permanent exclusion with IDA evidence + a one-line
> justification, organised by category. The word "pending" survives in the path
> for continuity only; what matters is that **no entry requires future code or
> audit action**.
>
> Scope: the four-version (`gms_v83`/`gms_v87`/`gms_v95`/`jms_v185`) SUMMARYs
> together carry ~325 residual `❌`/`🔍`. Every one of them is classified below
> into an accepted-exclusion category, OR was fixed during task-080 (or a prior
> pass 027–069) and is cited in the resolved tables. Genuinely-unresolved drift
> that task-080 did NOT fix and could NOT confidently bless is **surfaced as a
> follow-up task** (section 9) — it is NOT silently parked here.

---

## 1. How to read this registry

A SUMMARY `❌`/`🔍` is an **accepted exclusion** (not a wire bug) when it falls
into one of these categories. The category labels are referenced throughout:

| Category | Meaning |
|---|---|
| **TRUNCATION** | The IDA-export read-order ends before/after a real Atlas trailing field, so the analyzer emits phantom rows (`atlas: extra — client never reads this field` / `atlas: short — missing trailing field`). NOT a wire bug — the export JSON simply didn't capture the full read-order. Wire verified by byte test / prior per-struct ✅. |
| **OPAQUE** | Genuinely-opaque IDA type (A3 `Opaque` set): a single `DecodeBuf`/`EncodeBuf` token, or a struct with no `Encode`/`Write` method and no statically-decomposable layout (e.g. `model.Asset`, `GW_ItemSlotBase`, the mob body, AvatarLook). The register boundary — cannot decompose without guessing. |
| **REPRESENTATION** | Same wire bytes, different field decomposition: `WriteLong`≡`EncodeBuffer(8)`, `WriteInt64`≡`DecodeBuf(8)` FILETIME, `point`≡`EncodeBuffer(8)`, 4×`WriteInt32`≡`DecodeBuffer(16)` RECT, `WriteInt16+WriteShort`≡`Decode4`. |
| **OP/MODE-PREFIX** | Atlas models only the sub-op body; the IDA sender/dispatcher includes the leading op/mode byte at position 0, shifting every row. Per-op/per-mode body shapes audited independently and ✅. |
| **LOOP / EXCLUSIVE-BRANCH** | The flat analyzer cannot model a per-element loop, a mutually-exclusive `if/else` (both arms counted), or an early-`return` guard. Wire correct per runtime path. |
| **VERSION-ABSENT** | The FName/mode/feature is absent from this version's client (KMS-only, GMS-only, JMS-only, BBS-absent-in-JMS, or an unwired partial template). No counterpart to audit. |
| **REMOVED-LEGACY / NO-COUNTERPART** | Atlas writer kept for a real reason but has no game-wire counterpart in some baselines (documented presence map). |
| **PRIOR-ACCEPTED DRIFT** | A cross-version divergence a prior pass (027–069) already accepted with justification; carried forward verbatim. |

---

## 2. Resolved during task-080 (no longer pending)

These were OPEN deferrals in the prior ledger. task-080 fixed them; they are
removed from the registry and cited here for traceability.

| Former deferral | Resolution | sub-task (commit) |
|---|---|---|
| AffectedAreaCreated bespoke shape (matched no audited client) | **FIXED** — abs-RECT layout + `tStart` gated `GMS>=95`; channel passes skill id/level/type. nType source = 0 for skill mist (verdict). | B1.1 (`45711afce`, `fff7668ee`, `e24677ad8`) |
| chat `Multi` (serverbound) missing leading `update_time` | **FIXED** — `Encode4(updateTime)` prepended, gated `GMS>=95`. | B1.2 (`25faf971c`) |
| quest `ActionRestoreLostItem` single-id vs count-prefixed array | **FIXED** — count-prefixed lost-item id array. | B1.4 (`54110c020`) |
| quest `ActionStart`/`ActionComplete` "missing `nItemPos`" | **DISPROVEN** — premise faulty; no `nItemPos` in any of v83/v87/v95/JMS185. Existing decode byte-correct; no change. | B1.3 (spike-quest-actions) |
| `EffectWeather` JMS185 BLOW_WEATHER divergence | **FIXED** — region-dispatched body (JMS leading `itemId`, optional cash-type-51 extra, conditional message). GMS unchanged. | B1.5 (`a0d62f5af`) |
| ROUTING-npc-continue-conversation discriminator (`== 2`) | **FIXED** — channel handler routes 3/14 -> text, 5/8/9 -> selection. | B2.1 (`4fa1e3d52`) |
| merchant entrusted-shop-check modes 1/8/11 | **DISPOSED** — mode 1 client/KMS-only (absent in all baselines, not implemented); mode 8 emitter added; mode 11 defined-but-unused constant. | B2.3 (`e01915c2f`, spike-merchant-mode1) |
| login serverbound `Request` (`SendCheckPasswordPacket`) trailer | **FIXED** — `unknown2` gate widened to all-GMS; `partnerCode uint32` added (GMS-universal); per-version trailer byte test. | B6 (`600476c2a`, spike-login) |
| login clientbound `LoginAuth` ("orphan / remove?") | **KEPT** — it is the JMS185 `CLogin::LoginAuth` (idx 0x18) login-background-swap clientbound packet (single `.img` path string). Not a spurious auth packet. No GMS game-wire counterpart. | B6.1 (spike-login) |
| NOTE/memo `REFRESH` value concern | **RESOLVED — no bug** — Atlas's real serverbound map is `{SEND:0,DISCARD:1,REQUEST:2}`; REQUEST=2 matches the v83 client load (`OnMemoNotify_Receive`, `Encode1(2)`). The "REFRESH=7/8" was a misread. | B3.6 (`cf73e18a6`) |
| social/interaction/messenger/npc-shop sub-op enum-drift deferrals | **VERIFIED, no fix** — see section 6 (all sub-op VALUE spaces confirmed against the GMS-wired template + JMS185 binary; per-struct wire shapes already ✅). | B3.1–B3.6 (spikes) |
| JMS cash-shop NX-payment protocol (5 ❌ deferred as sibling task) | **WIRED** — JMS cash serverbound buy/gift/couple/friendship/rebate bodies + op-byte template remap + isPoints->MaplePoints currency. | B5.1 (`24b7eac38`…`38b31491b`, `b70b07079`) |
| SetField/WarpToMap `m_dwOldDriverID`; stat `Changed` / ui `Lock` gates | **RESOLVED in task-068/069 cross-version passes** — gates pinned `GMS>=95` (oldDriverID, nHP width) and `>90`/`>=95` (Lock, Changed); confirmed v83/v87/JMS185. | task-068/069 (carried forward) |
| task-065 combat "real wire bug candidates" (MonsterDestroy swallow-id, DropDestroy explode/pet tail, MonsterMovementHandle, MonsterControl shape) | **RESOLVED in task-065 follow-up** — 3 fixed (`ac174269b`, `e32a3d809`); MonsterControl was a false-positive (dev-mode seed block removed from IDA entries). | task-065 (carried forward) |
| task-067 commerce sub-op / SPW / itemCRC / oneADay / locker-counter deferrals | **RESOLVED in task-067 Phase 2 (v83 cross-version)** — version-gated `GMS>=95` or unconditional fixes; v83 anchored the boundary. | task-067 (carried forward) |
| task-066 social real wire bugs (PARTYDATA shortfall + JMS gate, party/guild Invite trailing fields, guild CapacityChange width) | **RESOLVED in task-066** — `2019dd581`, `29a248285`, `ab8511fee`. | task-066 (carried forward) |

---

## 3. TRUNCATION — IDA export read-order ended before a real Atlas trailing field

The **largest** accepted bucket. In each case the per-version IDA-export JSON's
read-order stops at (or mis-aligns at) the last captured field; the analyzer then
emits phantom `atlas: extra — client never reads this field` / `atlas: short —
missing trailing field` rows. These are export-capture limits, NOT wire bugs.
Each is verified correct by a byte/round-trip test or a prior per-struct ✅.

| Packet(s) | Versions | Evidence / why it's truncation, not a bug |
|---|---|---|
| `Request` (login serverbound, `SendCheckPasswordPacket`) | v83, v87 | Export read-order ends at `unknown1`; Atlas's real GMS trailer is `unknown1, unknown2, partnerCode` (B6 fix; GMS-universal). v95 export captures more rows -> ✅. `TestRequestTrailerShape` pins the per-version trailer. |
| `FieldAffectedAreaCreated` | v83/v87/v95/jms | Rows 0–8 ✅ after the B1.1 rewrite. The trailing phantom rows are the 16-byte `DecodeBuffer(16)` RECT aligned against Atlas's 4×`WriteInt32` (the export read-order ends at `tEnd`). Wire verified byte-symmetric; see `affected_area_created.go`. |
| `FieldChange` (serverbound) | jms | Rows 0–6 ✅. Rows 7–8 are 2 GMS-only trailing int32s gated OFF for JMS (branch-depth-2 region gate); the analyzer flattens both region arms. JMS runtime writes neither. |
| `FieldSetField`, `FieldWarpToMap` | v83/v87/v95/jms | Envelope fields ✅; residual rows are the seed-loop / CharacterData boundary + the `m_dwOldDriverID` gate (resolved `GMS>=95`). Per-version wire-length tests pin 25/27/33/33 bytes. |
| `CharacterList`, `CharacterViewAllCharacters`, `AddCharacterEntry` | all | Trailing rank/`bLoginOpt`/`nBuyCharCount` rows are the export-truncated tail of the CharacterListEntry/AvatarLook block + version-gated rank fields (see also section 4 REPRESENTATION). MapleStory packets are length-prefixed; trailing zero-fill is harmless. |
| `Changed` (stat), `Move` (character serverbound) | all | Export read-order ends before Atlas's version-gated trailing bytes (battle-recovery byte; dr-field tail). Gates confirmed `GMS>=95`/`>83` across v83/v87/JMS185 (task-069/028). |
| `KeyMapChange`, `CharacterKeyMap` | all | `CFuncKeyMappedMan::OnInit` loop count (89 vs 90) — the export captures a fixed loop length; Atlas always sends the full keymap (client treats extra as harmless). LOOP + truncation. |
| `Attack`, `CharacterDamage`, `CharacterMovement`, `CharacterAppearanceUpdate`, `CharacterSpawn`, `BuffGive`/`BuffGiveForeign`/`BuffCancel`, `StatusMessage*`, `EffectSimple`/`EffectQuest`/`EffectSkillUse` | all | Sub-op / sub-struct / movement-body packets whose IDA export captures only the dispatcher prefix or a single opaque body token; trailing/expanded Atlas rows are export-truncation + OPAQUE (see sections 4/5). Wire verified per the task-028/065/066 per-struct passes. |

---

## 4. REPRESENTATION & OPAQUE — same bytes / analyzer-boundary types

### REPRESENTATION (identical wire bytes, different decomposition)

| Packet(s) | Versions | Equivalence |
|---|---|---|
| `NoteDisplay`, `GuildBBSThread`, `GuildBBSThreadList` | all | `WriteInt64` FILETIME ≡ `DecodeBuffer(8)`. |
| `OperationMemoryGameMoveStone`, cash `MoveFrom/ToCashInventory`, `ShopOperationSetWishlist`/`WishList` | v95 | `WriteLong`/`point`/10×`WriteInt` ≡ `EncodeBuffer(8)` / `DecodeBuffer(40)`. |
| `CharacterList`/`CharacterViewAllCharacters`/`AddCharacterEntry` rank block | all | 4×`WriteInt` rank ≡ `DecodeBuffer(0x10)`; `anPetID[0..2]` are 3 full `WriteInt` (verified in `model/avatar.go`) — the byte/int32 "width mismatch" rows are the AvatarLook equip-loop-terminator misalignment, not a real width bug. |
| `ChatGeneralChat` | all | IDA `OnChat` entry begins after the dispatcher consumes `Decode4(characterId)`; Atlas writes it first -> position-0 int32-vs-byte artifact. Wire correct. |

### OPAQUE (register boundary — no decomposable layout)

| Packet(s) | Versions | Opaque type |
|---|---|---|
| `MonsterSpawn`, `MonsterStatSet`, `MonsterStatReset`, `MonsterControl`, `MonsterMovement` | all | Mob body collapses to one IDA `bytes` token (`CMob::SetTemporaryStat`+`Init` / `ProcessStatSet`); Atlas expands ~25 fields. `MonsterControl` hardcoded `byte(5)` aggro is a semantic note, not a wire-shape bug (width/position match). |
| `PetActivated`, `PetMovement`, `PetCommandResponse`, `PetMovementRequest`, `PetCommand`, `PetChatRequest`, `PetDropPickUp` | all | `CPet::Init` / `model.Movement` sub-struct expands as `DecodeBuf` placeholder; prefix fields ✅, body is the shared opaque encoder. |
| `MessengerAdd`, `MessengerUpdate` | all | AvatarLook `WriteByteArray` ≡ opaque `DecodeBuf`. |
| `InteractionInteractionEnter`, `InteractionInteractionEnterResultSuccess`, `InteractionInteractionUpdateMerchant` | v95 | `interaction.Visitor`/`Room`/per-item asset sub-structs flattened vs single buffer; headers ✅. |
| `StorageUpdateAssets`, `InventoryAdd` | v83/v95 | `model.Asset`/`GW_ItemSlotBase` per-tab loop — opaque sub-struct, audited independently; runtime callers pass exactly one tab -> wire ✅. |
| `GuildInfo`, `GuildMemberJoined` | all | `GUILDMEMBER::Decode` packed-array `DecodeBuffer(0x25=37)` vs Atlas per-element loop; 37-byte member body verified. |
| `MonsterMovementRequest` | all | `CMob::GenerateMovePath` sub-struct expansion FP; verified byte-for-byte (task-065, `e32a3d809`). |

---

## 5. OP/MODE-PREFIX, LOOP & EXCLUSIVE-BRANCH dispatcher artifacts

The audit pipeline compares Atlas's op-less/mode-less sub-op body (position 0 =
first real field) against an IDA dispatcher/sender whose position 0 is the
op/mode byte — shifting every row. Per-op/per-mode body shapes audited
independently and ✅. Sub-op VALUE spaces confirmed in section 6.

| Family / packets | Evidence | Justification |
|---|---|---|
| Party clientbound (`PartyCreated`, `PartyDisband`, `PartyError`, `PartyInvite`, `PartyJoin`, `PartyLeft`, `PartyUpdate`, `PartyChangeLeader`) | `CWvsContext::OnPartyResult#*` | mode-byte prefix; bodies ✅ after `2019dd581`. `WritePartyData` = 378B (GMS v95) / 298B (v83/JMS) confirmed; hot-path byte tests added. |
| Party serverbound (`PartyOperation*`, `PartyMemberHP`) | `CField::Send*PartyMsg` | op-byte / characterId dispatcher prefix; targetName/targetId/hp align after adjustment. |
| Buddy serverbound (`BuddyOperationAdd/Accept/Delete`) | `CField::Send{Set,Accept,Delete}FriendMsg` | two-step decode (op byte then sub-type); ADD=1/ACCEPT=2/DELETE=3 binary-confirmed (B3.6). |
| Buddy clientbound (`BuddyError`, `BuddyListUpdate`, `BuddyUpdate`) | `CWvsContext::OnFriendResult#*` | mode-byte prefix; error arms read mode only (StringPool notice, no wire string — B3.6). `BuddyError.hasExtra` 0x10 trailing 0x00 is ignored by the client (harmless). |
| Guild clientbound (`GuildTitleChange`, `GuildMember*`, `GuildCapacityChange`, `GuildAgreementResponse`, `GuildSetTitleNames`, `GuildDisband`, `GuildInvite`, `GuildRequestCreate`, `GuildKick`, `GuildSetEmblem`/`SetNotice`/`SetMemberTitle`/`Withdraw`/`Join`/`InviteRequest`/`InviteReject`) | `CWvsContext::OnGuildResult#*` | mode-byte prefix + 5-string loop (`SetTitleNames`/`TitleChange`); bodies ✅ after `29a248285`. |
| Guild BBS serverbound (`GuildBBS{ListThreads,DisplayThread,DeleteReply,CreateOrEditThread,ReplyThread,DeleteThread}`) | `CUIGuildBBS::*` | op-byte prefix; threadId/replyId/message align after adjustment. |
| Messenger clientbound/serverbound (`MessengerAdd`/`Chat`/`Join`/`Remove`/`InviteSent`/`InviteDeclined`/`RequestInvite`/`Update`, `MessengerOperation*`) | `CUIMessenger::*` | op/mode-byte prefix + AvatarLook opaque; enum {0,2,3,5,6} verified (B3.1/B3.2). |
| NPC (`NpcAction`, `NpcActionRequest`, `NpcContinueConversationSelection`, `NpcShopList`, `NpcNpcConversation`, `NpcSpawn*`) | `CNpc::*` / `CShopDlg::SetShopDlg` | conditional movement sub-struct, per-commodity loop + ammo/non-ammo exclusive branch, wide/narrow selection branch. Wire correct per per-item IDA trace. |
| Storage / Inventory (`StorageShow`, `InventoryChangeBatch`) | `CTrunkDlg::SetGetItems` / `OnInventoryOperation` | per-tab loop + conditional trailing addMov; `Show` segmentation fixed in task-067 (residual = loop-flatten). |
| Cash (`CashShopOperationIncreaseInventory`/`IncreaseStorage`/`SetWishlist`) | `OnBuySlotInc`/`OnIncTrunkCount`/`OnSetWish` | exclusive-branch over-count (`if m.item {int} else {byte}`) + loop-flatten; runtime fires one arm. Wire ✅. |
| Character (`CharacterSitResult`) | `CUserLocal::OnSitResult` | divergent-length exclusive `if/else` (`byte+short` vs `byte`); else-branch `WriteByte(0)` surfaces as "extra". Client reads `Decode1`+conditional `Decode2` — matches. |
| `InteractionOperationChat` | `CheckAndSendChat` | op-byte prefix; v83 single-string shape (`update_time` gated `GMS>=95`, task-067). |

---

## 6. Sub-op enum VALUE spaces — verified (B3.1–B3.6) + version-absent gaps

Per-struct wire shapes were ✅ in prior passes. task-080's four-version
enum-drift spike verified the template-configured sub-op *value* spaces; **no
config/constant fix was required.**

- **Buddy / Guild / Party / Note / Chat sub-op maps** are wired only in
  `template_gms_83_1.json`; that map is internally consistent and matches both the
  GMS client and the JMS185 binary. v87/v95/JMS social writers/handlers are
  **template-absent** (VERSION-ABSENT gaps, not value divergences).
- **JMS party serverbound renumber** (KICK=3, CHANGE_LEADER=5, JOIN_RESPONSE=0 vs
  GMS JOIN=3/EXPEL=5/CHANGE_LEADER=6) is a real GMS<->JMS divergence but lives in an
  **unwired** JMS template — recorded so a future JMS-party-wiring task uses the JMS
  numbers, not a bug in any wired entry.
- **Interaction serverbound sub-ops** (CREATE/OPEN/INVITE_DECLINE/VISIT verified;
  CASH_TRADE_OPEN/MERCHANT_NAME_CHANGE/PERSONAL_STORE_SET_VISITOR have no client
  send-site in JMS185 or the GMS v95 export — corroborated, bodies unchanged).
- **NPC shop clientbound modes + serverbound op-bytes** verified equal to the GMS
  client (B3.3/B3.4).

---

## 7. VERSION-ABSENT — feature/FName absent in a baseline (no counterpart)

| Packet(s) | Absent in | Reason |
|---|---|---|
| Guild BBS (`GuildBBS*` clientbound + serverbound) | JMS185 | BBS feature entirely absent from JMS v185. |
| NPC shop (`NpcShopBuy`/`Sell`/`Recharge`/`ShopOperationGenericError`) | JMS185 | `template_jms_185_1.json` wires neither `NPCShopHandle` nor `NPCShopOperation` (partial seed). Op-byte-prefix artifact compounds the unwired template. |
| Merchant `FreeFormNotice` and siblings | JMS185 | merchant writer template-absent in JMS185 seed (GMS rows are all ✅). |
| VAC select (`SendSelectCharPacketByVAC`, `OnSelectCharacterByVACResult`), login license accept/deny | JMS185 | GMS-only login paths; JMS has no VAC view-all-select nor login-license wire. |
| `RegisterPin`, `SetGender`, `AuthSuccess`, `ServerListEntry` | JMS185 | JMS has `usesPin:false`; SetGender absent; `OnCheckPasswordResult` decodes a fundamentally different JMS structure (login domain tracked in task-027). |
| `ServerLoad`, `ServerSelect`, `PicResult` (GMS v12-era / state-machine-routed) | v95 | Pre-v95 or non-`CLogin`-dispatched; trivial shapes manually cross-checked. |
| Merchant mode 1 (`UnableToOpenTheStore`) | all four | client/KMS-only — absent from every audited `OnEntrustedShopCheckResult` switch; not implemented (B2.3). |

---

## 8. REMOVED-LEGACY / NO-COUNTERPART

- **`LoginAuth` (clientbound)** — KEPT. It is the JMS185 `CLogin::LoginAuth`
  (idx 0x18) login-background-swap packet (single `.img`-path string). No GMS
  game-wire counterpart; not a spurious auth packet. The prior ledger's
  "remove?" lean was a misread (spike-login section 2).
- **`tool/uint128.go`** — utility type, not a packet domain (zero `Encode`/`Decode`,
  zero audit rows).
- **`locateAtlasFile` struct-name collisions** (`ChannelChange`) — resolved by the
  qualified-name `candidatesFromFName` map (task-080 section 4.7, `696ae8e0d`); the
  intended `channel/clientbound/change.go` now audits ✅.

---

## 9. BuddyInvite — RESOLVED in task-080 (the one surfaced real bug, now fixed)

The one genuinely-unresolved divergence E2 surfaced was **fixed in-task** after
all four IDBs were decompiled (the earlier `🔍`/follow-up framing is superseded).

- **`BuddyInvite` inviter `jobId`/`level`** (`buddy/clientbound/invite.go`) — **RESOLVED**
  (`b39329ecb`). The client reads two extra `Decode4` (`jobId`, `level`) between
  `originatorName` and the `GW_Friend`(39)+`inShop` tail, present for **GMS≥87 and all
  JMS, absent on GMS v83**. All four `OnFriendResult` case-9 read-orders were
  decompiled (v83 `@0xa3f2e8`, v87 `@0xad7ae5`, v95 `@0xa12630`, JMS185 `@0xb2a873`);
  Atlas now writes `jobId`/`level` gated `Region!="GMS" || Major>=87`, with the
  inviter's real job/level wired from the invite consumer, per-version byte tests, and
  the 39-byte friend buffer + inShop unchanged.
  - **Note:** three of the four IDA-export JSON read-orders for this packet are
    **mistraced** (v83/v87 export it as a `count + buddy[i]` loop; JMS truncates after
    `level`). So the regenerated SUMMARY may still show `❌`/`🔍` for BuddyInvite in
    those versions — that is an **export mistrace/truncation accepted-exclusion**, NOT
    an Atlas defect (Atlas's wire is IDA-correct per all four decompiles; v95's
    correctly-traced export flips its jobId/level rows to ✅). See
    `docs/tasks/task-080-packet-audit-closeout/spike-buddy-invite.md`.

---

## 10. Cross-check

`grep -nE 'DEFERRED|pending|TODO|🔍|FIXME|action'` over this file returns hits
only inside the registry's own prose: the `🔍` glyphs are part of category
descriptions / the section-9 follow-up pointer, "pending" is the path/title, and
"action" appears in this sentence and the category table. **No open actionable
item remains** — every audit `❌`/`🔍` is either fixed-and-cited (sections 2 & 9) or
classified into an accepted-exclusion category (sections 3–8). The one genuine wire
bug surfaced (BuddyInvite, section 9) was **fixed in-task**, so nothing is handed off.

## 11. Workflow reference (refresh / regen procedure)

1. `mcp__ida-pro__list_functions_filter` -> `get_function_by_name` -> `decompile_function`.
2. Parse the `CInPacket::DecodeN` / `COutPacket::EncodeN` sequence in lexical order
   (success path; multi-branch functions need manual filtering).
3. Add the entry to `gms_v{83,87,95}.json` / `gms_jms_185.json` and the
   `candidatesFromFName` map in `tools/packet-audit/cmd/run.go`.
4. Regenerate: `cd tools/packet-audit && go run . --template … --ida-source … --output docs/packets/audits`.
   Synthetic `#`-suffixed FNames model one IDA function across multiple Atlas sub-branches.
5. Diff the regenerated SUMMARY against this registry: any NEW `❌`/`🔍` not in a
   category above is either a real bug (-> new task) or a new analyzer artifact (->
   `tools/packet-audit` section 4.7 enhancement) — never a silent re-deferral.
