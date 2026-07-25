# v48 Stage E — BATCH 1 (small families) report

Scope: the SMALL families — login, chat, stat, drop, reactor, account, monster,
buddy, storage, note. In-scope in-scope tier-1 gms_v48 ❌/🟡 cells: **28**
(23 unique structs; op + sub-struct rows collapse onto shared structs).
Anchor = gms_v61 fast-path. IDB port 13337 (GMS_v48_1_DEVM.exe).

## Result summary

- **22 cells promoted ✅** (17 unique structs) — v48 verified **0 → 17**.
- **1 cell dispositioned n-a** (monster/carnival/MonsterCarnival — v48-absent).
- **5 cells BLOCKED** (genuine version divergence / unresolved v48 send site).
- matrix `--check` exit **0**; STATUS.md orphan/dangling/stale/drift/unresolv/malformed grep **0**.
- v48 conflicts **0** (unchanged). No sibling regression: v61 208 / v72 216 /
  v79 228 / v83 367 / v84 345 / v87 379 / v95 399 / jms 362 all unchanged.
- `go test ./libs/atlas-packet/...` ALL PASS; `go vet` clean.

## Per-cell outcome

### NOTE (commit 81a001ba28)
- note/clientbound/NoteDisplay (MEMO_RESULT) — ✅ =v61. CWvsContext::OnMemoResult
  @0x71d8e2, Display mode byte 2 (=v61), per-entry sub_49CCDB read order identical.
  Sibling arms (SendSuccess=3/SendError=4, no Refresh) verified to flip the op row.
- note/serverbound/NoteOperationDiscard (NOTE_ACTION) — ✅ =v61. CMemoListDlg::SetRet
  @0x534dc4 (opcode 101 vs v61 119, Δ-18, no layout change).
- note/serverbound/NoteOperation — BLOCKED. CWvsContext::OnMemoNotify_Receive does
  not exist as a real function in the v48 IDB; the only serverbound NOTE_ACTION
  producer is SetRet's discard path. No standalone request-send site (note-send
  exists in-game via a different unfound path, so n-a would be wrong — left blocked).
- note/serverbound/NoteOperationSend — BLOCKED. CCashShop::OnCashItemResLoadGiftDone
  absent in v48 IDB (v48 CCashShop lacks the gift-note path); ❌ on all 9 versions,
  never cracked. Not fabricated.

### CHAT + STAT + DROP (commit 54e36fe715)
- stat/clientbound/Changed (STAT_CHANGED) — ✅ =v61. OnStatChanged @0x71aa68,
  DecodeChangeStat @0x49ba4a bit-ascending, bytes identical to v61.
- chat/clientbound/ChatWorldMessageSimple (SERVERMESSAGE) — ✅ =v61. OnBroadcastMsg
  @0x71c356 simple modes Decode1(mode)+DecodeStr(msg); switch 0-9 (narrower than v61 0-10).
- chat/serverbound/ChatMulti (MULTI_CHAT) — ✅ =v61. sub_65EB4F @0x65eb4f.
- chat/serverbound/ChatWhisper (WHISPER) — ✅ =v61. sub_4C4F3B @0x4c4f3b. Report generated.
- drop/serverbound/DropPickUp (ITEM_PICKUP) — ✅ =v61. sub_70D987 @0x70d987, no crc
  (pre-83). Report generated.
- run.go: 3 candidatesFromFName cases added (sub_65EB4F/sub_4C4F3B/sub_70D987). Opcodes
  already routed in template. ChatMulti report regenerated to drop the bogus
  CUIStatusBar::SendGroupMessage stub shadow (committed export NOT modified).

### LOGIN + ACCOUNT (commit f08691333b)
- login/serverbound/AfterLogin — ✅ =v61 (sub_503956 @0x503956 dialog-result==2 arm).
- login/serverbound/AllCharacterListRequest — ✅ =v61 (sub_502293 @0x502293 bare
  COutPacket(12), empty body for GMS<87).
- account/serverbound/RegisterPin — ✅ v48-derived (sub_503956 result==1 arm:
  Encode1(1)+EncodeStr(pin)).
- login/clientbound/AuthSuccess — ✅ v48-derived + **wire fix**. sub_500931 @0x500931
  success path has NO trailing Decode4(nNumOfCharacter); gated to `MajorVersion()>=61`
  in auth_success.go (Encode+Decode). v48 wire = 41 bytes.
- login/clientbound/ServerIP — ✅ v48-derived (sub_502B70 @0x502b70 code==0 arm:
  ip/port/clientId/auth/premium; premium int present, v48>12).

### MONSTER + REACTOR (commit daae358a89)
- monster/serverbound/MonsterMovementRequest (MOVE_LIFE) — ✅ v48-gated. sub_550383
  @0x550383 has NO Encode4(hackedCode) that v61+ carries; gated hackedCode
  `GMS>=61 || JMS` in movement.go. candidatesFromFName sub_550383 added; stub-key
  placeholder stripped from v48 export so the report keys to the real send site.
- reactor/serverbound/ReactorHitRequest (DAMAGE_REACTOR) — ✅ v48-gated. Layout
  evolution IDA-confirmed: v48 @0x5a5d1a = oid+dwHitOption+delay (3 fields); isSkill
  added at v72, skillId added at v79. Two-boundary gate in hit.go: isSkill `GMS>=72`,
  skillId `GMS>=79`. All v79+ fixtures still pass.
- monster/carnival/serverbound/MonsterCarnival — n-a. IDA-confirmed absent (no Carnival
  fn or string in the v48 binary; CPQ is a later feature). Spurious Stage-D
  MonsterCarnival.{json,md} report removed → cell drops to n-a.

### BUDDY + STORAGE (commit 12aa5681c5)
- buddy/serverbound/BuddyOperationAccept (BUDDYLIST_MODIFY op100) — ✅ mirror-twin.
  sub_4C6643 @0x4c66aa: mode 2 + Encode4(fromCharId). Opcode 0x64 already routed.
- buddy/serverbound/BuddyOperationDelete — ✅ mirror-twin. sub_4C659B @0x4c65fb:
  mode 3 + Encode4(buddyCharId).
- buddy/serverbound/BuddyOperationAdd — ✅ v48-gated. sub_4C6452 @0x4c6538 sends
  EncodeStr(name) with NO group name (group name IDA-confirmed absent v48/v61, present
  v72+); gated `MajorVersion()>61` in operation_add.go. 3 CField::Send*FriendMsg entries
  surgically spliced into gms_v48.json (real addresses+calls, 38-line diff, no rewrite).
- storage/serverbound/StorageOperationMeso — BLOCKED.
- storage/serverbound/StorageOperationRetrieveAsset — BLOCKED.
- storage/serverbound/StorageOperationStoreAsset — BLOCKED.
  v48 op52 storage is an OLDER protocol generation. Exhaustive enumeration of every
  COutPacket(52) send site (byte-search + dispatcher sub_582BDC + CTrunkDlg::OnPacket
  @0x58332c) found only modes 2 (parcel/gift), 4/5 (withdraw item BY itemId — not
  invType+slot), 7 (close). No GetMoney(amount), no GetItem(invType+slot), no
  PutItem(slot+itemId+qty) arm — the exact shapes the Atlas StorageOperationMeso/
  RetrieveAsset/StoreAsset structs (v87 twins @0x81c15c/0x81bc1f/0x81bdfc) model.
  The codec fnames aren't in the v48 export and can't be honestly grounded. Not
  fabricated; storage codecs/twins untouched. Follow-up: v48 bank deposit/meso may
  live at a different opcode (inventory-move family) — outside these 6 cells' scope.

## Codec changes (all version-gated for the legacy range; v61+ UNCHANGED)
- login/clientbound/auth_success.go — nNumOfCharacter gated `MajorVersion()>=61`.
- monster/serverbound/movement.go — hackedCode gated `GMS>=61 || JMS`.
- reactor/serverbound/hit.go — isSkill gated `GMS>=72`, skillId gated `GMS>=79`.
- buddy/serverbound/operation_add.go — buddy-group name gated `MajorVersion()>61`.

## Gates / verification
- matrix --check exit 0; problem-word grep 0; v48 conflicts 0.
- go test ./libs/atlas-packet/... ALL PASS (incl. -race on changed pkgs); go vet clean.
- Regression: all 8 sibling verified counts unchanged.
- Branch verified `task-113-gms-legacy-versions` after every commit.

## Commits (5)
- 81a001ba28  note (NoteDisplay + NoteOperationDiscard)
- 54e36fe715  stat/Changed, chat/{WorldMessageSimple,Multi,Whisper}, drop/PickUp
- f08691333b  login/account (AuthSuccess, ServerIP, AfterLogin, AllCharacterListRequest, RegisterPin)
- daae358a89  monster/MonsterMovementRequest + reactor/ReactorHitRequest; MonsterCarnival n-a
- 12aa5681c5  buddy/BuddyOperation{Add,Accept,Delete}
