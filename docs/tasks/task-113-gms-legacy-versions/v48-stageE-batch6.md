# v48 Stage E — Batch 6 (npc: NPC dialog / CScriptMan / npc-shop)

Anchor v61 fast-path, IDB port 13337 (GMS_v48_1_DEVM.exe). Commit `a16262ea9a`
on `task-113-gms-legacy-versions`.

## Result: 20 promoted / 5 n-a / 0 blocked

- **18 tier-1 sub-structs** promoted ❌/🟡 → ✅ (byte fixtures + pinned evidence).
- **2 op cells** (NPC_TALK, NPC_TALK_MORE serverbound) promoted → ✅.
- **5 sub-structs** dispositioned n-a (genuinely v48-absent; residual incomplete
  accepted, cap = anchor).

## The v48 conversation dispatcher (re-derived, body-verified)

`sub_5B0AE4@0x5b0ae4` reads Decode1 speakerTypeId, Decode4 speakerTemplateId,
Decode1 msgType, then a switch over **cases 0..9 only** (v61 had 0..0xE; default
is a no-op). Each arm's reply `COutPacket(47)+Encode1(msgType)` echoes the case
index, so the byte layout is a distinct, older table — NOT v61's rotated one:

| byte | arm | reader | body |
|---|---|---|---|
| 0 | Say | sub_5B0C11@0x5b0c11 | str, b prev, b next |
| 1 | (yes/no, a6=0) | sub_5B0D5C@0x5b0d5c | str |
| 2 | AskYesNo (a6=1) | sub_5B0D5C@0x5b0d5c | str |
| 3 | AskText (GetText) | sub_5B0E90@0x5b0e90 | str, str, s, s |
| 4 | AskNumber (GetNumber) | sub_5B1037@0x5b1037 | str, i, i, i |
| 5 | AskMenu (#L, dialog type 4) | sub_5B1195@0x5b1195 | str ONLY (no count) |
| 6 | AskAvatar | sub_5B12E8@0x5b12e8 | str, b count, i×count |
| 7 | MemberShopAvatar | sub_5B1494@0x5b1494 | str, b count, i×count |
| 8 | AskPet | sub_5B1640@0x5b1640 | str, b count, (buf8+b)×count |
| 9 | AskPetAll | sub_5B18B5@0x5b18b5 | str, b count, b exc, (buf8+b)×count |

## NPC_TALK_MORE messageType table — RE-DERIVED (was carried-UNVERIFIED)

Old template was the v61 layout (ASK_QUIZ=5 collided with the real AskMenu arm,
ASK_MENU=7 hit MemberShopAvatar, ASK_PET=10 was a no-op, ASK_TEXT=13 a no-op —
a live desync/crash for v48 NPC dialogs). Corrected `template_gms_48_1.json`
handler 71 (0x2F) to:

`SAY=0, ASK_YES_NO=2, ASK_TEXT=3, ASK_NUMBER=4, ASK_MENU=5, ASK_AVATAR=6,
ASK_MEMBER_SHOP_AVATAR=7, ASK_PET=8, ASK_PET_ALL=9`

(ASK_YES_NO kept at 2 — case 2 is a real yes/no arm; byte 1 is the a6=0
variant with no distinct Atlas struct. ASK_BOX_TEXT/ASK_QUIZ/ASK_SPEED_QUIZ
removed — no v48 arm.)

## NPC_TALK sb body shape: oid + x + y (NOT oid-only)

`sub_568A2A@0x568a2a` builds `COutPacket(46) + Encode4(npcObjId
@0x569297/0x569380) + Encode2(userX @0x5692b0/0x569399) + Encode2(userY
@0x5692ca/0x5693b3)`. Registry primary corrected `sub_56D1F0` (only the field
hit-test that RETURNS the clicked NPC — no COutPacket) → `sub_568A2A` (the real
sender). Gate `startConversationHasXY` now includes v48.

## Shop senders (all COutPacket(48) mode-prefix, body after mode byte)

- BUY `sub_5B7422@0x5b7422`: Encode1(0) + Encode2 slot + Encode4 itemId +
  Encode2 count. NO discountPrice (v48 < 72 gate).
- SELL `sub_5B7693@0x5b7693`: Encode1(1) + Encode2 slot + Encode4 itemId +
  Encode2 count.
- RECHARGE `sub_5B78C0@0x5b78c0`: Encode1(2) + Encode2 slot.

## Codec gates added (v48-only branches, legacy untouched)

1. `AskMenuConversationDetail`: count byte now `MajorAtLeast(61) && !MajorAtLeast(83)`
   (v48 menu is str-only; v61..v82 keep the merged-menu count byte).
2. `StartConversation.startConversationHasXY`: `|| MajorVersion()==48`.
3. `ShopList`: v48 (`< 61`) branch — no maxPerSlot short; ammo (rechargeable
   itemId/10000==207) reads unitPrice(8) then quantity(2) always
   (`CShopDlg::SetShopDlg sub_5B430A@0x5b430a`).

## Arms n-a'd (genuinely absent; dispatcher handles 0..9 only)

SayImage (no image-dialog arm), AskQuiz (v61 case is sub_84816E, absent),
AskSpeedQuiz (sub_8482CB, absent), AskBoxText (only single-line GetText at case
3), AskSlideMenu (v83+ feature). Added to `_unimplemented.json` with evidence;
stale static-diff reports removed (→ "no audit report"). Matches the anchor
(v61 leaves AskSlideMenu incomplete too).

## Export hygiene

Surgically spliced 5 missing readers (sub_5B0AE4/5B0D5C/5B0E90/5B78C0/5B430A)
and enriched sub_568A2A into `docs/packets/ida-exports/gms_v48.json` via
targeted text edits — no full re-export, no whole-file reformat.

## Verification

- `go test -race ./libs/atlas-packet/npc/...` — green; `go vet` clean;
  `go build ./libs/atlas-packet/...` clean.
- `packet-audit matrix --check` — **exit 0**; problem-grep (STATUS.md) — **0**.
- v48 conflicts — **0**; v48 verified 87.
- Regression: v83 367 / v84 345 / v87 379 / v95 399 / jms 362 / v72 216 /
  v79 228 / v61 208 — **none dropped**; no conflicts on any version.
- Branch after commit: `task-113-gms-legacy-versions`. Commit touches only
  gms_v48 npc cells (43 files); no out-of-scope report-regen drift.

## Notes / concerns

- The 5 n-a residuals remain ❌ in the matrix (the tool renders no sub-struct
  n-a state) — accepted per brief (cap = anchor).
- ASK_YES_NO byte 1 vs 2 is genuinely ambiguous (both are yes/no dialogs from
  the same handler sub_5B0D5C, differing only by the a6 button flag); kept the
  pre-existing value 2 (a real arm) to avoid an unjustified behavior change.
