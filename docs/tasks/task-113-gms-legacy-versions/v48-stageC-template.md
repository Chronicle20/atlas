# GMS v48 — Stage C seed template

> Task 4.C (Stage C) of task-113. Anchor = `gms_v61`. Output:
> `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`.
> Method: anchor-diff from `template_gms_61_1.json` — every handler/writer opCode
> re-derived from `docs/packets/registry/gms_v48.yaml` (58 sb + 93 cb = 151
> body/named-verified entries, the Stage B clean-partial); dispatcher mode tables
> re-derived from the v48 IDB (`GMS_v48_1_DEVM.exe`, port 13337) where Stage A
> flagged divergence. Content sections (characters/npcs/worlds/cashShop) carried
> verbatim from the v61 anchor (version-agnostic seeds; identical across all
> sibling templates).

## Version fields
`region=GMS`, `majorVersion=48`, `minorVersion=1`, `usesPin=false` (Stage A (e):
PIN slots at login op 6/7 as v61, no SPW cases — `usesPin` carries false).

## Counts
- **Handlers: 57.** = 58 serverbound registry ops − 1 drop (`ANTI_MACRO_RESULT`,
  no Atlas handler in ANY version template — atlas-channel does not implement an
  anti-macro handler; dropping mirrors the anchor).
- **Writers: 54 entries** covering **49 distinct clientbound ops.** 44 of the 93
  clientbound registry ops have **no Atlas writer in any version** (atlas-channel
  only seeds writers for packets it actually emits; the v61 anchor likewise has
  139 writers for 220 cb registry ops). The 5 extra entries are legitimate
  multi-writer opcodes carried from the anchor: `0x01` LOGIN_STATUS ×4
  (AuthSuccess/AuthTemporaryBan/AuthPermanentBan/AuthLoginFailed), `0x0A`
  WORLD_INFORMATION ×2 (ServerListEntry/ServerListEnd), `0x36` SPAWN_PORTAL ×2
  (SpawnPortal/RemoveTownDoor).

## Validation
- JSON valid (`python3 -m json.tool` clean).
- `go build ./...` clean in atlas-configurations; `go test ./templates/...` ok.
- 0 handlers missing a validator. Validators: 1 `NoOpValidator` (LoginHandle, per
  anchor), 56 `LoggedInValidator`. (Pong/StartError/CharacterLoggedIn are NOT in
  the v48 serverbound registry — see gap below — so LoginHandle is the only
  NoOp-validated handler.)
- 0 duplicate opCodes within handlers. Writer opCode duplicates are the 3
  intentional multi-writer groups above (present in the v61 anchor by design).

## Mode tables — RE-DERIVED from the v48 IDB (4)
Distrusted the carried table; verified the switch body.

| writer/op | v48 op | IDB switch (addr) | result |
|---|---|---|---|
| **WorldMessage** (SERVERMESSAGE) | 0x37 (55) | `CWvsContext::OnBroadcastMsg` @ `0x71c356` | arms 0–7 body-verified = NOTICE/POP_UP/MEGAPHONE/SUPER_MEGAPHONE/TOP_SCROLL/PINK_TEXT/BLUE_TEXT/NPC. **Dropped ITEM_MEGAPHONE(v61=8)/YELLOW_MEGAPHONE(9)/MULTI_MEGAPHONE(10)** — v48 bytes 8/9 are a shared generic-notice variant (NO `GW_ItemSlotBase::Decode`; confirms Stage A "no item-megaphone"), byte 10 absent. |
| **PartyOperation** (PARTY_OPERATION cb) | 0x32 (50) | `CWvsContext::OnPartyResult` @ `0x729935` | structural arms re-packed vs v61: INVITE=4, **UPDATE=6** (v61 7), **CREATED=7** (v61 8), **LEAVE/DISBAND/EXPEL=11** (v61 12), **JOIN=14** (v61 15), CHANGE_LEADER=26 (shares refresh path 6/26), **TOWN_PORTAL=29** (v61 36 — per-slot town-portal coord update, confirms Stage A). |
| **FieldEffect** (FIELD_EFFECT) | 0x54 (84) | `sub_4C7B59` @ `0x4c7b59` | arms 0–6 body-verified = SUMMON/TREMBLE/OBJECT/SCREEN/SOUND/BOSS_HP/BACKGROUND_MUSIC. **Dropped REWARD_RULLET(v61=7)** — no type-7 arm in the v48 switch. |
| **CharacterStatusMessage** (SHOW_STATUS_INFO) | 0x21 (33) | `CWvsContext::OnMessage` @ `0x71b1b8` | arms **0–9 = v61** (Stage A verified equal; contiguous, no spurious INCREASE_SKILL_POINT). Carried v61 table (it IS the v48 table). |

## Mode tables — CARRIED-UNVERIFIED → FLAG for Stage E (4)
Carried from the v61 anchor because Stage A did not extract them; the v48 switch
was NOT decompiled for these. Per the repeated "carried-from-anchor table is
wrong" bug pattern, Stage E MUST re-derive each from the v48 switch:

1. **GuildOperation** (GUILD_OPERATION cb, 0x35/53) — `CWvsContext::OnGuildResult`
   @ `0x725559`. Carried v61 (30+ arms). Stage A explicitly deferred.
2. **CharacterInteractionHandle** (PLAYER_INTERACTION sb, 0x5D/93) — interaction
   miniroom/trade dispatcher `operations`+`enterError`. Carried v61.
3. **PartyOperationHandle** (PARTY_OPERATION sb request, 0x5E/94) — serverbound
   party-request `operations`. Carried v61. (Distinct from the cb PartyOperation
   above, which IS re-derived.)
4. **NPCContinueConversationHandle** (NPC_TALK_MORE sb, 0x2F/47) — `messageType`.
   Carried v61. Registry Stage-B body-verified the endpoints only (SAY=0,
   ASK_AVATAR/membership=8, consistent with v61); the middle arms
   (ASK_YES_NO/BOX_TEXT/NUMBER/QUIZ/MENU) are UNVERIFIED for v48 — the v83 shift
   bug (ASK_MENU=4) shows these can move. Playthrough-critical; verify in Stage E.

Note: the `types` movement-fragment tables (CharacterMoveHandle etc.) are an
Atlas-internal classification, not a wire mode byte read from a version switch —
carried verbatim, not flagged.

## v48-absent drops
- **Serverbound (1):** `ANTI_MACRO_RESULT` (op 83) — registered in v48 but no
  Atlas handler exists in any version; dropped.
- **Clientbound (44):** ops with no Atlas writer in any version — login/char-select
  results (CONFIRM_EULA_RESULT, SELECT_CHARACTER_BY_VAC, RELOG_RESPONSE), shim
  (KOREAN_INTERNET_CAFE_SHIT, AUTHEN_MESSAGE, IDA_0X014/015), the 16 unresolved
  `IDA_0X*` CWvsContext/CField placeholder subs, and packets atlas-channel never
  emits (INVENTORY_GROW, SKILL_USE_RESULT, MAP_TRANSFER_RESULT, WEDDING_PHOTO,
  CLAIM_RESULT/AVAILABLE_TIME/STATUS_CHANGED, QUEST_CLEAR, INCUBATOR_RESULT,
  SUE_CHARACTER_RESULT, ALLIANCE_OPERATION, SHOP_SCANNER_RESULT, MAPLE_TV_USE_RES,
  AVATAR_MEGAPHONE_RESULT, SET_AVATAR_MEGAPHONE, DESTROY_SHOP_RESULT, and the 5
  CASHSHOP_* result ops). All consistent with the v61 anchor (which also omits
  them). Stage A's TOUCH_MONSTER_ATTACK / MONSTER_BOOK_COVER / MESO_DROP were
  already evidence-dropped upstream (absent from the v48 registry entirely).
  - `SUE_CHARACTER_RESULT` (op 44) could plausibly map to the v61 `StalkResult`
    writer (0x77), but the v61 registry itself left 0x77 unresolved (`IDA_0X09C`),
    so mapping would be a guess — dropped + noted rather than fabricated.

## In-scope gap (upstream registry / Stage B deferral — NOT producible in Stage C)
The v48 **serverbound registry has no login-side handlers except LoginHandle
(LOGIN_PASSWORD) and SetGenderHandle**: no Pong, StartError, ServerListRequest,
ServerStatus, AfterLogin, RegisterPin, ViewAll, CharacterSelect, CheckName,
Create/DeleteCharacter, ChannelChange-request, CharacterLoggedIn. Stage B's
serverbound pass (`discover_gms_v48.md` worklist) enumerated the in-scope
playthrough send-sites and left the login/char-select serverbound flow
un-enumerated (same gap v79 Stage C hit — progress line 7, "Controller decision
pending"). The template faithfully reflects the registry; closing this needs a
Stage B serverbound login-flow enumeration (IDB send-site work), not a template
edit. Surfaced for the controller.

## Commit
`f05eb43155362a15648650489c827db56a37d03c`
