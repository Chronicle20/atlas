# CharacterInteraction — serverbound `CharacterInteractionHandle` legacy coverage audit

Read-only family audit (family-auditor). Diagnostic only — no codec, template,
yaml, registry, or evidence record was mutated. Produced 2026-07-14 on branch
`task-127-owl-shop-search`.

## Family header

| field | value |
|---|---|
| Family | CharacterInteraction (mini-room / trade / personal shop / hired merchant / mini-game) |
| Clientbound writer | `CharacterInteraction` / fname `CMiniRoomBaseDlg::OnPacketBase` / op `PLAYER_INTERACTION` (dispatcher yaml: `docs/packets/dispatchers/character_interaction.yaml`) |
| **Serverbound handler (this audit)** | `CharacterInteractionHandle` — the client→server action dispatcher, keyed by `options.operations` in each seed template. **No dispatcher yaml governs it** (see §6). |
| Direction audited | serverbound (client sends `[recvOp][mode]…`) |
| Arm count (v83 reference) | 45 modes |
| IDBs read | gms_v61 (13338), gms_v72 (13339, via task-113 disposition), gms_v79 (13340), gms_v83 (13342, reference), jms_v185 (13344) |

**Headline finding (changes the task premise):** the rich hired-merchant
**management** dialog (`CEntrustedShopDlg`: organize / withdraw-meso /
merchant-off / merchant-exit / view-visit-list / view-black-list /
add-/remove-black-list-by-name) **is a v83+ feature.** In gms_v61 / v72 / v79 the
client binary has **only** `CWvsContext::OnEntrustedShopCheckResult` (the permit
check recv) and a shared `CPersonalShopDlg` that carries hired-merchant
put/buy/remove via a subclass flag — there is **no** `CEntrustedShopDlg`
management dialog, so those eight management modes **do not exist** on
v61/v72/v79 and cannot be "completed" in those templates. jms_v185 is the
opposite case: it has the full feature but its template is a 3-mode stub.

---

## §1. Current serverbound tables + recv opcode (all 9 versions)

Extracted from `services/atlas-configurations/seed-data/templates/template_<ver>_1.json`,
`handler == "CharacterInteractionHandle"`. `recvOp` = the `opCode` the handler is
bound to (= the client's PLAYER_INTERACTION send opcode).

| version | recvOp | #modes | state |
|---|---|---|---|
| gms_v48 | 0x5D | 7 | CREATE/INVITE/VISIT/CHAT + TRADE put/meso/confirm only (no store/merchant) |
| gms_v61 | 0x6F | 17 | personal-store + basic merchant (put/buy/remove); **no OPEN/EXIT**, no merchant-mgmt |
| gms_v72 | 0x79 | 17 | byte-identical map to v61 |
| gms_v79 | 0x78 | **0** | **EMPTY — handler bound but dispatches nothing (see §3c)** |
| gms_v83 | 0x7B | 45 | full reference (IDA-cross-checked, §2) |
| gms_v84 | 0x7D | 45 | identical to v83 |
| gms_v87 | 0x81 | 45 | identical to v83 |
| gms_v95 | 0x90 | 45 | identical to v83 |
| jms_v185 | 0x7C | **3** | **STUB — PERSONAL_STORE_BUY=20, MERCHANT_BUY=31, PERSONAL_STORE_SET_BLACK_LIST=27 (see §3d)** |

recvOp confirmed against the task brief (v61=0x6F, v72=0x79, v79=0x78) and, for
jms, **derived from IDA**: every jms send site constructs `COutPacket(124)` =
`0x7C` = the jms template `opCode`.

---

## §2. gms_v83 reference — every serverbound mode IDA-cross-checked

All `CEntrustedShopDlg` / `CPersonalShopDlg` senders in the v83 IDB
(`MapleStory_dump.exe`, port 13342) construct `COutPacket(123)` = `0x7B`
(matches template). Merchant-management block (the arms the task targets):

| sender (fname@addr) | Encode1 mode | = template key |
|---|---|---|
| `CEntrustedShopDlg::OnGoOut` @0x51923b | 0x27 (39) | MERCHANT_MERCHANT_OFF=39 ✓ |
| `CEntrustedShopDlg::OnArrange` @0x519294 | 0x28 (40) | MERCHANT_ORGANIZE=40 ✓ |
| `CEntrustedShopDlg::SetRet` @0x51809d (nRet==8) | 0x29 (41) | MERCHANT_EXIT=41 ✓ |
| `CEntrustedShopDlg::OnWithdrawMoney` @0x51930f | 0x2B (43) | MERCHANT_WITHDRAW_MESO=43 ✓ |
| `CEntrustedShopDlg::OnVisitList` @0x5194af | 0x2E (46) | MERCHANT_VIEW_VISIT_LIST=46 ✓ |
| `CEntrustedShopDlg::OnBlackList` @0x519382 | 0x2F (47) | MERCHANT_VIEW_BLACK_LIST=47 ✓ |
| `CEntrustedShopDlg::AddBlackList` @0x519611 | 0x30 (48) | MERCHANT_ADD_TO_BLACK_LIST=48 ✓ |
| `CEntrustedShopDlg::DeleteBlackList` @0x519695 | 0x31 (49) | MERCHANT_REMOVE_FROM_BLACK_LIST=49 ✓ |

The v83 template is authoritative; v84/v87/v95 inherit it byte-identically.

---

## §3. IDA-derived actual client mode bytes — legacy versions

### §3a. gms_v61 (opcode 0x6F=111) — `GMS_v61.1_U_DEVM.exe`, port 13338

Send sites located by clustering `push 6Fh` (`6A 6F`) into `COutPacket::COutPacket`
(@0x5ffc4f). The `CPersonalShopDlg` + mini-room base block is 0x60c250–0x60e31d;
the merchant/personal split is a virtual (`(*(this+92))(this)` → merchant flag).

**VERIFIED present, values match template:**

| sender@addr | Encode1 | mode key(s) | template? |
|---|---|---|---|
| sub_60D2E9 | 0x0B(11)+Encode1(1) | **OPEN**=11 | **MISSING from template** |
| sub_60C250 / sub_68B29B (SetRet nRet==2) | 0x0A(10) | **EXIT**=10 | **MISSING from template** |
| sub_68CBE3 | 0x10(16) | TRADE_CONFIRM=16 | ✓ present |
| sub_60DBE2 | `flag?31:20` | MERCHANT_PUT_ITEM=31 / PERSONAL_STORE_PUT_ITEM=20 | ✓ present |
| sub_60D81E | `flag?32:21` | MERCHANT_BUY=32 / PERSONAL_STORE_BUY=21 | ✓ present |
| sub_60DF59 | `flag?36:25` | MERCHANT_REMOVE_ITEM=36 / PERSONAL_STORE_REMOVE_ITEM=25 | ✓ present |
| sub_60E20A | 0x1A(26)+slot+str | PERSONAL_STORE_ADD_TO_BLACKLIST=26 | ✓ present |
| sub_60E154 | 0x1C(28)+count+strs | PERSONAL_STORE_SET_BLACK_LIST=28 | ✓ present |
| sub_60E31D | 0x1B(27)+slot+str | *(personal-store auto-reban of idle visitor; 1h timeout)* | **not in template** — a real mode present in the binary, unnamed in template; likely the `PERSONAL_STORE_SET_VISITOR` analog |

**VERSION-ABSENT (n-a) on v61 — no `CEntrustedShopDlg`:** MERCHANT_ORGANIZE,
MERCHANT_WITHDRAW_MESO, MERCHANT_MERCHANT_OFF, MERCHANT_EXIT,
MERCHANT_VIEW_VISIT_LIST, MERCHANT_VIEW_BLACK_LIST, MERCHANT_ADD_TO_BLACK_LIST,
MERCHANT_REMOVE_FROM_BLACK_LIST. `func_query *EntrustedShop*` yields only
`CWvsContext::OnEntrustedShopCheckResult` @0x848c1c. The bytes gms uses for these
(0x2C–0x31) are occupied in v61 by the **mini-game dialogs** (blocks 0x5b5953…,
0x5fc4d7… — confirmed by `play_minigame_sound` and 8-byte move buffers), so the
gms values are not even re-usable. Do **not** add these to the v61 template.

**Net v61 gap = OPEN(11) + EXIT(10) only.** The 17 present modes are all
correct; the table is *incomplete* (missing the two base lifecycle modes the
personal-shop go-live/close needs), not *wrong*.

### §3b. gms_v72 (opcode 0x79=121) — `GMS_v72.1_U_DEVM.exe`, port 13339

Template is a byte-identical map of v61 (same 17 modes/values). Task-113 already
IDA-dispositioned the merchant-management arms here: `docs/packets/audits/gms_v72/_unimplemented.json`
records `CEntrustedShopDlg::AddBlackList` / `DeleteBlackList` as **version-absent**
("`*EntrustedShop*` yields only `CWvsContext::OnEntrustedShopCheckResult`
@0x91ff18 … the CEntrustedShopDlg blacklist feature is post-v72"). Same shape as
v61: personal-store + basic merchant present; merchant-management n-a. Net gap =
OPEN(11) + EXIT(10) (values pending a v72-IDB confirm; expected identical to v61).

### §3c. gms_v79 (opcode 0x78=120) — `GMS_v79_1_DEVM.exe`, port 13340 — CRITICAL

`func_query *EntrustedShop*|*PersonalShopDlg*` → **only**
`CWvsContext::OnEntrustedShopCheckResult` @0x971dd8. Structurally identical to
v61/v72: no `CEntrustedShopDlg` (merchant-management n-a), but the personal-store
/ trade / mini-game senders exist (unnamed). **Yet the v79 template's
`operations` map is EMPTY (0 modes).** The handler is bound to 0x78 but can
resolve **no** action — the entire interaction feature (trade, personal shop,
basic merchant, mini-games) is **dead on v79**. This is the single most severe
finding. v79 needs its full personal-store/trade/base/mini-game table populated
from the v79 IDB (values expected ≈ v61/v72). Merchant-management stays n-a.

### §3d. jms_v185 (opcode 0x7C=124) — `MapleStory_dump_SCY.exe`, port 13344

Full feature present; `CEntrustedShopDlg` symbols intact. All senders construct
`COutPacket(124)`=0x7C. **Every store/merchant mode is exactly `gms_v83 − 3`**
(the yaml's documented jms shift, confirmed end-to-end):

| sender (fname@addr) | Encode1 | mode key | = v83 − 3 |
|---|---|---|---|
| `CPersonalShopDlg::PutItem` @0x762a9e | `flag?30:19` | MERCHANT_PUT_ITEM=30 / PERSONAL_STORE_PUT_ITEM=19 | 33/22 − 3 ✓ |
| `CPersonalShopDlg::MoveItemToInventory` @0x762e26 | `flag?35:24` | MERCHANT_REMOVE_ITEM=35 / PERSONAL_STORE_REMOVE_ITEM=24 | 38/27 − 3 ✓ |
| `CPersonalShopDlg::OnClickBanButton` @0x7630d5 | 0x19(25)+slot+str | PERSONAL_STORE_ADD_TO_BLACKLIST=25 | 28 − 3 ✓ |
| `CEntrustedShopDlg::OnGoOut` @0x54b798 | 0x24(36) | MERCHANT_MERCHANT_OFF=36 | 39 − 3 ✓ |
| `CEntrustedShopDlg::OnArrange` @0x54b7f1 | 0x25(37) | MERCHANT_ORGANIZE=37 | 40 − 3 ✓ |
| `CEntrustedShopDlg::SetRet` @0x54a623 (nRet==8) | 0x26(38) | MERCHANT_EXIT=38 | 41 − 3 ✓ |
| `CEntrustedShopDlg::OnWithdrawMoney` @0x54b86c | 0x28(40) | MERCHANT_WITHDRAW_MESO=40 | 43 − 3 ✓ |
| `CEntrustedShopDlg::OnVisitList` @0x54ba0c | 0x2B(43) | MERCHANT_VIEW_VISIT_LIST=43 | 46 − 3 ✓ (yaml-verified) |
| `CEntrustedShopDlg::OnBlackList` @0x54b8df | 0x2C(44) | MERCHANT_VIEW_BLACK_LIST=44 | 47 − 3 ✓ (yaml-verified) |
| `CEntrustedShopDlg::AddBlackList` @0x54bb75 | 0x2D(45) | MERCHANT_ADD_TO_BLACK_LIST=45 | 48 − 3 ✓ |
| `CEntrustedShopDlg::DeleteBlackList` @0x54bbf9 | 0x2E(46) | MERCHANT_REMOVE_FROM_BLACK_LIST=46 | 49 − 3 ✓ |

The existing 3-mode stub (PERSONAL_STORE_BUY=20, MERCHANT_BUY=31,
PERSONAL_STORE_SET_BLACK_LIST=27) is likewise `gms_v83 − 3` (23/34/30 − 3) and
therefore **correct** — jms just needs the remaining ~25 modes filled in.

---

## §4. Ready-to-apply serverbound tables (per version) + unresolved flags

### gms_v61 / gms_v72 — ADD these two rows to the existing 17 (nothing else)

```
"OPEN": 11,     // sub_60D2E9  (personal-shop go-live) — IDA-verified v61
"EXIT": 10      // sub_60C250 / SetRet nRet==2         — IDA-verified v61
```

- **v72**: confirm 11/10 on the v72 IDB (port 13339) before applying — expected
  identical to v61 (templates are otherwise byte-identical).
- Merchant-management modes: **n-a on v61/v72** — do not add.

### gms_v79 — POPULATE the empty table (full derivation required)

The whole `operations` map is empty. Derive every present mode from the v79 IDB
(port 13340). Expected value set = v61/v72 (same era, same recv-op family). At
minimum the personal-store + trade + base (OPEN/EXIT/CREATE/INVITE/VISIT/CHAT) +
mini-game modes. Merchant-management: n-a. **Every v79 byte is UNRESOLVED until
derived — do not copy v61 blind; verify.**

### jms_v185 — extend the 3-mode stub

**IDA-VERIFIED, apply directly** (all `gms_v83 − 3`):

```
"PERSONAL_STORE_PUT_ITEM": 19,   "PERSONAL_STORE_REMOVE_ITEM": 24,
"PERSONAL_STORE_ADD_TO_BLACKLIST": 25,
"MERCHANT_PUT_ITEM": 30,         "MERCHANT_REMOVE_ITEM": 35,
"MERCHANT_MERCHANT_OFF": 36,     "MERCHANT_ORGANIZE": 37,
"MERCHANT_EXIT": 38,             "MERCHANT_WITHDRAW_MESO": 40,
"MERCHANT_VIEW_VISIT_LIST": 43,  "MERCHANT_VIEW_BLACK_LIST": 44,
"MERCHANT_ADD_TO_BLACK_LIST": 45,"MERCHANT_REMOVE_FROM_BLACK_LIST": 46
// (stub, already correct): PERSONAL_STORE_BUY=20, MERCHANT_BUY=31, PERSONAL_STORE_SET_BLACK_LIST=27
```

**UNRESOLVED for jms — DO NOT guess; IDA-derive before templating.** The base
mini-room modes are documented version-stable (CREATE=0, INVITE=2, VISIT=4,
CHAT=6, EXIT=10, OPEN=11), but the uniform −3 store shift implies **3 modes were
dropped between CHAT(6) and PERSONAL_STORE_PUT(19)**, and I did not verify which.
Flag every one of these jms bytes as stop-and-ask:
`INVITE_DECLINE`, `CASH_TRADE_OPEN`, `TRADE_PUT_ITEM`, `TRADE_ADD_MESO`,
`TRADE_CONFIRM`, `TRANSACTION`, `PERSONAL_STORE_SET_VISITOR`,
`FIELD_ADD_TO_BLACK_LIST`, `FIELD_REMOVE_FROM_BLACK_LIST`, and the whole
`MEMORY_GAME_*` block. (Confirm by decompiling the jms trade / mini-game senders,
`COutPacket(124)` sites near the personal-shop block at 0x762–0x763.)

---

## §5. Per-arm × per-version coverage matrix (status.json)

`docs/packets/audits/status.json` grades the serverbound **codec structs** (byte
layouts) under `interaction/serverbound/*`, not the mode-byte *tables*. Quoted
states (V=verified, X=incomplete "no audit report", –=n-a):

| arm (interaction/serverbound/…) | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|---|---|---|---|
| Chat, Invite, Transaction, Trade{PutItem,AddMeso,Confirm} | V | V | V | V | V | V | V | V | V |
| PersonalStore{PutItem,Buy,RemoveItem,AddToBlackList,SetBlackList} | V | V | V | V | V | V | V | V | V |
| Field{AddToBlackList,RemoveFromBlackList} | V | V | V | V | V | V | V | V | V |
| Merchant{Buy,PutItem,RemoveItem} | V | V | V | V | V | V | V | V | V |
| MemoryGame{FlipCard,MoveStone,RetreatAnswer,TieAnswer} | V | V | V | V | V | V | V | V | V |
| **MerchantAddToBlackList** | – | **X** | **X** | **V** | V | V | V | V | V |
| **MerchantRemoveFromBlackList** | – | **X** | **X** | **V** | V | V | V | V | V |

Two coverage inconsistencies surfaced (both real findings, not passes):

1. **v72 disposition ↔ matrix mismatch.** `gms_v72/_unimplemented.json` marks
   `CEntrustedShopDlg::AddBlackList`/`DeleteBlackList` **version-absent**, yet the
   `MerchantAddToBlackList`/`RemoveFromBlackList` v72 cells read `incomplete`, not
   `n-a`. The disposition and the matrix cell disagree.
2. **v79 "verified" is suspect.** The v79 IDB has **no `CEntrustedShopDlg`**
   (§3c), so the by-name merchant-blacklist arms are version-absent there just
   like v61/v72 — but the matrix marks v79 `MerchantAddToBlackList`/`Remove…` =
   `verified` (a shared-codec byte-layout mark, no v79 evidence record). Combined
   with the **empty v79 template**, this "verified" is misleading: nothing routes
   these on v79. Reconcile to `n-a` (feature absent) after the §3c v79 derivation.

Note: no discrete struct exists for MERCHANT_ORGANIZE / WITHDRAW_MESO /
MERCHANT_OFF / MERCHANT_EXIT / VIEW_VISIT_LIST / VIEW_BLACK_LIST / OPEN / EXIT /
CASH_TRADE_OPEN — those modes carry **no body** (mode byte only) so they need no
codec, only a template entry. They are therefore invisible to the matrix; the
seed template is their *only* coverage surface. This is exactly the "sub-arm gaps
hidden behind the op-row aggregate" case the auditor exists to surface.

---

## §6. Operations-table cross-check & generator ownership (task item #4)

- **No serverbound dispatcher yaml exists.** `grep "direction: serverbound"
  docs/packets/dispatchers/*.yaml` → none. No yaml references
  `CharacterInteractionHandle`.
- The only interaction yaml — `docs/packets/dispatchers/character_interaction.yaml`
  — is **clientbound** (`writer: CharacterInteraction`, `direction: clientbound`)
  and covers the `PLAYER_INTERACTION` *writer* plus two clientbound-echo view
  keys (`MERCHANT_VIEW_VISIT_LIST`, `MERCHANT_VIEW_BLACK_LIST`).
- Therefore `packet-audit operations [--check]` does **not** generate or validate
  the serverbound `CharacterInteractionHandle` table. **It is hand-maintained**
  in the 9 seed templates.
- Cross-check against the clientbound yaml (which echoes request modes): yaml
  `jms_v185` `MERCHANT_VIEW_VISIT_LIST=43` / `MERCHANT_VIEW_BLACK_LIST=44` — my
  serverbound IDA read matches exactly (client echoes the request byte). No
  yaml/IDA conflict. The yaml carries **no** entries for the other legacy
  serverbound modes, so it provides no guard for this task's edits.

---

## §7. Divergence notes

- **Enum is version-shifted, not merely truncated.** v61/v72 personal-store modes
  sit two below v83 (v83 PUT=22 → v61 20) because v83 inserted `INVITE_DECLINE`,
  `EXIT`, `OPEN`, `CASH_TRADE_OPEN`, `TRANSACTION` above them. jms is a clean
  `gms_v83 − 3` across the whole store/merchant region.
- **Merchant-management is a v83+ dialog** (`CEntrustedShopDlg`). v48/v61/v72/v79
  have only the shared `CPersonalShopDlg` (+ the permit-check recv). The current
  implementation *honors* this for v48 (n-a) but **not** for v61/v72 (cells left
  `incomplete`) or v79 (cells wrongly `verified` + empty template).
- **v61 mode 0x1B(27)** (`sub_60E31D`, auto-reban idle visitor by name) is a real
  present mode with no template key — worth naming (`PERSONAL_STORE_SET_VISITOR`
  candidate) during the v61 pass.
- **jms −3 boundary is unverified** for the trade/transaction/mini-game region
  (§4). The shift is proven for the store/merchant block only.

---

## §8. Recommendations (do NOT act here — hand to implementer/verifier)

Ordered. Follow the fix playbook trigger 1 ("a family-audit bug") in
[`docs/packets/RE_AUDITING_A_COLUMN.md`](../../docs/packets/RE_AUDITING_A_COLUMN.md):
confirm each reported gap against the live IDB (`validate`/`infer`) before any
template change, then hand to a `dispatcher-family-implementer` (whole legacy
column) or `packet-verifier` (single unverified arm).

1. **jms_v185 — apply the §4 verified block now** (13 IDA-verified merchant +
   personal-store modes, all `gms_v83 − 3`). Highest value: the feature exists
   and only the stub blocks it. Then IDA-derive the §4 UNRESOLVED jms bytes
   (trade/transaction/mini-game) before adding those rows — stop-and-ask, no
   guessing.
2. **gms_v79 — populate the empty template** (`dispatcher-family-implementer`
   pass over the v79 IDB, port 13340). This is the most severe gap (feature
   entirely undispatched). Merchant-management stays n-a.
3. **gms_v61 / gms_v72 — add `OPEN`=11 and `EXIT`=10** (confirm on the v72 IDB).
   These are the only genuine gaps; the other 17 modes are IDA-correct.
4. **Reconcile the matrix** (§5): flip v61/v72/v79 `MerchantAddToBlackList` /
   `MerchantRemoveFromBlackList` cells to **n-a** (version-absent) and add the
   missing `gms_v79/_unimplemented.json` disposition for `CEntrustedShopDlg::*`,
   matching the existing v72/v48 records. Do **not** leave v79 reading `verified`
   over an empty template.
5. **Do NOT** attempt to add merchant-management modes to v48/v61/v72/v79
   templates — they are version-absent (`CEntrustedShopDlg` is v83+). Any such
   entry would be a fabricated byte.

### Stop-and-ask (unresolved, must not be guessed)

- Every gms_v79 serverbound byte (whole table empty; derive from IDB).
- jms_v185: `INVITE_DECLINE`, `CASH_TRADE_OPEN`, `TRADE_PUT_ITEM`,
  `TRADE_ADD_MESO`, `TRADE_CONFIRM`, `TRANSACTION`, `PERSONAL_STORE_SET_VISITOR`,
  `FIELD_ADD_TO_BLACK_LIST`, `FIELD_REMOVE_FROM_BLACK_LIST`, all `MEMORY_GAME_*`.
- gms_v72: OPEN/EXIT values (confirm on port 13339; expected 11/10).

---

## §3c-derived — gms_v79 verified table (`GMS_v79_1_DEVM.exe`, port 13340, send opcode 0x78=120)

Derived 2026-07-14 by decompiling every `COutPacket(120)` send site in the v79
IDB (clustered from `6A 78` = `push 78h`) and cross-referencing each mode name
against the **named** v83 senders in `MapleStory_dump.exe` (port 13342), whose
class/method symbols (`CMemoryGameDlg`, `COmokDlg`, `CTradingRoomDlg`,
`CField`, `CMiniRoomBaseDlg`, `CEntrustedShopDlg`) and `StringPool` string
constants make the mode→name mapping unambiguous.

**Headline: v79 uses a THIRD, distinct enum — not v61/v72's `-2` shift and not
v83's reference.** Base/lifecycle + trade + cash-trade + the two tie modes match
v83 **exactly (Δ0)**; the entire personal-store / merchant / field-blacklist
block and the mini-game block from `ASK_RETREAT` upward are uniformly **v83 − 1
(Δ−1)**. `FORFEIT` is an outlier at **49** (see note). Every byte below is read
directly from an `Encode1` at the cited sender, except `TRANSACTION` (unresolved).

### Verified mode → byte table

| mode key | v79 byte | sender fname@addr — `Encode1` read | v83 | Δ |
|---|---|---|---|---|
| CREATE | **0** | `CField::SendInviteTradingRoomMsg`@0x51b10b (else-branch) `Encode1(0)` | 0 | 0 |
| INVITE | **2** | `CField::SendInviteTradingRoomMsg`@0x51b10b `Init`+`Encode1(2)`; `CMiniRoomBaseDlg::OnEnterResultStatic`@0x62d8fa | 2 | 0 |
| INVITE_DECLINE | **3** | `CMiniRoomBaseDlg::SendInviteResult`@0x62d482 `Encode1(3)`; `SendCashInviteResult`@0x62d59a | 3 | 0 |
| VISIT | **4** | `CMiniRoomBaseDlg::SendInviteResult`@0x62d432 `Encode1(4)` | 4 | 0 |
| CHAT | **6** | `CMiniRoomBaseDlg::CheckAndSendChat` sub_62E124@0x62e15b `Encode1(6)` | 6 | 0 |
| EXIT | **10** | sub_688441@0x68846a `Encode1(0xA)`; trade `SetRet` sub_7356DD@0x735703; minigame `OnClickEndButton` sub_6773E6@0x677517 | 10 | 0 |
| OPEN | **11** | sub_6896A0@0x6896ca `Encode1(0xB)`; sub_671E4B@0x671ea0 | 11 | 0 |
| CASH_TRADE_OPEN | **14** | sub_6895B2@0x689648 `Encode1(0xE)`; `CField::SendInviteTradingRoomMsg`@0x51b10b (if-branch); `SendCashInviteResult`@0x62d541 | 14 | 0 |
| TRADE_PUT_ITEM | **15** | `CTradingRoomDlg::PutItem` sub_736C99@0x736e20 `Encode1(0xF)` (SP887 "how many will you trade") | 15 | 0 |
| TRADE_ADD_MESO | **16** | `CTradingRoomDlg::PutMoney` sub_736EC4@0x737028 `Encode1(0x10)` (SP414 "how much money…") | 16 | 0 |
| TRADE_CONFIRM | **17** | `CTradingRoomDlg::Trade` sub_73709A@0x737150 `Encode1(0x11)` (SP415 "are you sure you want to trade") | 17 | 0 |
| PERSONAL_STORE_PUT_ITEM | **21** | sub_68A3E3@0x68a673 `Encode1(flag?32:21)` | 22 | −1 |
| PERSONAL_STORE_BUY | **22** | sub_689CE7@0x68a38f `Encode1(flag?33:22)` | 23 | −1 |
| PERSONAL_STORE_REMOVE_ITEM | **26** | sub_68A756@0x68a83d `Encode1(flag?37:26)` | 27 | −1 |
| PERSONAL_STORE_ADD_TO_BLACKLIST | **27** | sub_68AA05@0x68aaae `Encode1(0x1B)` (click-ban a visitor, `EncodeStr` name) | 28 | −1 |
| PERSONAL_STORE_SET_VISITOR | **28** | sub_68AB52@0x68aba6 `Encode1(0x1C)` (idle-visitor auto-reban, 1h timeout) | 29 | −1 |
| PERSONAL_STORE_SET_BLACK_LIST | **29** | sub_68A951@0x68a980 `Encode1(0x1D)` (deliver full config blacklist) | 30 | −1 |
| FIELD_ADD_TO_BLACK_LIST | **30** | `CField::AddBlackList` sub_522C87@0x522ca2 `Encode1(0x1E)`+`EncodeStr` | 31 | −1 |
| FIELD_REMOVE_FROM_BLACK_LIST | **31** | `CField::DeleteBlackList` sub_522CFF@0x522d10 `Encode1(0x1F)`+`EncodeStr` | 32 | −1 |
| MERCHANT_PUT_ITEM | **32** | sub_68A3E3@0x68a673 `Encode1(flag?32:21)` | 33 | −1 |
| MERCHANT_BUY | **33** | sub_689CE7@0x68a38f `Encode1(flag?33:22)` | 34 | −1 |
| MERCHANT_REMOVE_ITEM | **37** | sub_68A756@0x68a83d `Encode1(flag?37:26)` | 38 | −1 |
| MEMORY_GAME_ASK_TIE | **50** | sub_672668@0x67268d `Encode1(0x32)`; sub_61D6A6@0x61d6cb | 50 | 0 |
| MEMORY_GAME_TIE_ANSWER | **51** | sub_677047@0x6770a9 `Encode1(0x33)`; sub_62298D@0x6229e6 | 51 | 0 |
| MEMORY_GAME_FORFEIT | **49** | sub_6770E1@0x677142 `Encode1(0x31)` (SP461 give-up confirm; ≙ v83 `CMemoryGameDlg::SendClaimGiveUp`@0x65364a `Encode1(0x34=52)`) | 52 | **−3** |
| MEMORY_GAME_ASK_RETREAT | **53** | sub_67717F@0x6771f8 `Encode1(0x35)` (≙ v83 `COmokDlg::SendRetreatRequest`@0x6e8c3b `Encode1(54)`) | 54 | −1 |
| MEMORY_GAME_RETREAT_ANSWER | **54** | sub_672728@0x67274d `Encode1(0x36)`+bool (≙ v83 `COmokDlg::OnRetreatRequest`@0x6e416b `Encode1(55)`) | 55 | −1 |
| MEMORY_GAME_EXIT_AFTER_GAME | **55** | sub_6773E6@0x677453 / sub_622C46@0x622cb3 `Encode1(0x37)` (≙ v83 `OnClickEndButton`@0x653970 `Encode1(56)`) | 56 | −1 |
| MEMORY_GAME_CANCEL_EXIT_AFTER_GAME | **56** | sub_6773E6@0x6774b1 / sub_622C46@0x622d11 `Encode1(0x38)` (≙ v83 `Encode1(57)`) | 57 | −1 |
| MEMORY_GAME_READY | **57** | sub_6772B1@0x6772da `Encode1(0x39)` (not-ready branch; ≙ v83 `CMemoryGameDlg::OnClickReadyButton`@0x6537ce `Encode1(58)`) | 58 | −1 |
| MEMORY_GAME_UNREADY | **58** | sub_6772B1@0x67730b `Encode1(0x3A)` (ready branch; ≙ v83 `Encode1(59)`) | 59 | −1 |
| MEMORY_GAME_EXPEL | **59** | sub_622BCB@0x622c1a `Encode1(0x3B)` (SP459 expel; ≙ v83 `CMemoryGameDlg::OnClickBanButton`@0x653888 `Encode1(0x3C=60)`) | 60 | −1 |
| MEMORY_GAME_START | **60** | sub_67725C@0x677285 `Encode1(0x3C)` (this[50]>=1; ≙ v83 `CMemoryGameDlg::OnClickStartButton`@0x653779 `Encode1(0x3D=61)`) | 61 | −1 |
| MEMORY_GAME_SKIP | **62** | sub_673525@0x67364b `Encode1(0x3E)` (turn-timer expiry auto-skip; v83 SKIP=63, gap at 62 in v83) | 63 | −1 |
| MEMORY_GAME_MOVE_STONE | **63** | sub_676FD6@0x676ff9 `Encode1(0x3F)`+`EncodeBuffer(8)`+byte (≙ v83 `COmokDlg::PutStoneChecker`@0x6e8a19 `Encode1(64)`) | 64 | −1 |
| MEMORY_GAME_FIP_CARD | **67** | sub_61E16E@0x61e18e `Encode1(0x43)`+card+card (≙ v83 `CMemoryGameDlg::SendTurnUpCard`@0x64ee2b `Encode1(68)`) | 68 | −1 |

### Version-absent (n-a) on gms_v79 — do NOT template

`func_query name_regex "EntrustedShop"` on the v79 IDB yields **only**
`CWvsContext::OnEntrustedShopCheckResult` @0x971dd8 — there is **no
`CEntrustedShopDlg`**. The eight hired-merchant **management** modes are a v83+
feature (v83 `CEntrustedShopDlg::*`) and have no v79 sender:
`MERCHANT_MERCHANT_OFF`, `MERCHANT_ORGANIZE`, `MERCHANT_EXIT`,
`MERCHANT_WITHDRAW_MESO`, `MERCHANT_VIEW_VISIT_LIST`, `MERCHANT_VIEW_BLACK_LIST`,
`MERCHANT_ADD_TO_BLACK_LIST`, `MERCHANT_REMOVE_FROM_BLACK_LIST`. (Note: the
`FIELD_ADD/REMOVE_FROM_BLACK_LIST` modes above are **present** — they are
`CField::` methods, NOT `CEntrustedShopDlg`, and exist on v79.)

### UNRESOLVED (do NOT guess)

- **`TRANSACTION`** — v83 byte 20. **No client `Encode1(20)` send site found in
  either the v79 or the v83 IDB** (both `CTradingRoomDlg::Trade` and
  `CCashTradingRoomDlg::Trade` send `TRADE_CONFIRM=17`, not 20). Mode 20 appears
  to be server-driven / not client-sent in this era. If Atlas never *decodes* a
  serverbound `TRANSACTION` on v79 it needs no template entry; if it must, the
  byte is unconfirmed (positionally either 19 if it follows the Δ−1 store block,
  or 20 if it follows the Δ0 trade block — **not verified, do not template a
  guessed value**). Escalate.

### Codec-correctness flag (not a byte issue — a BODY issue)

The tie pair's **bodies are swapped between v79 and v83**, even though the bytes
(50/51) match:
- **v83**: `ASK_TIE`(50) = no body (`SendTieRequest`); `TIE_ANSWER`(51) = 1 bool
  body (`OnTieRequest`).
- **v79**: `ASK_TIE`(50) sub_672668 sends `Encode1(50)`+**`Encode1(bool)`** (1-byte
  body); `TIE_ANSWER`(51) sub_677047 sends `Encode1(51)` with **no body**.

If the v79 `MemoryGameTieAnswer` / `AskTie` codecs are ported byte-identically
from v83 they will mis-decode on v79. The implementer must verify the per-mode
body when wiring v79 (the retreat pair does NOT have this inversion:
`RETREAT_ANSWER`=54 carries the bool on both versions).

### Ready-to-paste `operations` additions (populate the empty gms_v79 table)

```
"CREATE": 0,
"INVITE": 2,
"INVITE_DECLINE": 3,
"VISIT": 4,
"CHAT": 6,
"EXIT": 10,
"OPEN": 11,
"CASH_TRADE_OPEN": 14,
"TRADE_PUT_ITEM": 15,
"TRADE_ADD_MESO": 16,
"TRADE_CONFIRM": 17,
"PERSONAL_STORE_PUT_ITEM": 21,
"PERSONAL_STORE_BUY": 22,
"PERSONAL_STORE_REMOVE_ITEM": 26,
"PERSONAL_STORE_ADD_TO_BLACKLIST": 27,
"PERSONAL_STORE_SET_VISITOR": 28,
"PERSONAL_STORE_SET_BLACK_LIST": 29,
"FIELD_ADD_TO_BLACK_LIST": 30,
"FIELD_REMOVE_FROM_BLACK_LIST": 31,
"MERCHANT_PUT_ITEM": 32,
"MERCHANT_BUY": 33,
"MERCHANT_REMOVE_ITEM": 37,
"MEMORY_GAME_ASK_TIE": 50,
"MEMORY_GAME_TIE_ANSWER": 51,
"MEMORY_GAME_FORFEIT": 49,
"MEMORY_GAME_ASK_RETREAT": 53,
"MEMORY_GAME_RETREAT_ANSWER": 54,
"MEMORY_GAME_EXIT_AFTER_GAME": 55,
"MEMORY_GAME_CANCEL_EXIT_AFTER_GAME": 56,
"MEMORY_GAME_READY": 57,
"MEMORY_GAME_UNREADY": 58,
"MEMORY_GAME_EXPEL": 59,
"MEMORY_GAME_START": 60,
"MEMORY_GAME_SKIP": 62,
"MEMORY_GAME_MOVE_STONE": 63,
"MEMORY_GAME_FIP_CARD": 67
// TRANSACTION: UNRESOLVED — omit until a client sender is confirmed (see above)
// MERCHANT_MERCHANT_OFF / ORGANIZE / EXIT / WITHDRAW_MESO / VIEW_VISIT_LIST /
// VIEW_BLACK_LIST / ADD_TO_BLACK_LIST / REMOVE_FROM_BLACK_LIST: n-a (v83+ CEntrustedShopDlg)
```

**FORFEIT=49 note.** `MEMORY_GAME_FORFEIT` is the one mode that breaks the tidy
Δ pattern (Δ−3, not Δ0/−1). It is derived directly (sub_6770E1@0x677142 sends
`Encode1(0x31=49)`, guarded by a one-shot give-up confirm that matches v83
`CMemoryGameDlg::SendClaimGiveUp`'s behavior byte-for-byte except the mode
value). v79 simply places give-up at 49 (a slot free on v79 because 49 is a
`CEntrustedShopDlg` merchant-blacklist mode on v83, absent here). Not a
transcription error — read twice.

## §6-resolution — matrix/disposition fixes (task-127, do-mode)

- **v61/v79 dispositions added.** `CEntrustedShopDlg::AddBlackList` and
  `::DeleteBlackList` are now dispositioned version-absent in
  `docs/packets/audits/gms_v61/_unimplemented.json` and `.../gms_v79/...`,
  matching the pre-existing v72 disposition (all three binaries have only
  `CWvsContext::OnEntrustedShopCheckResult`, no `CEntrustedShopDlg`).
- **False v79 "verified" evidence removed.** The
  `InteractionOperationMerchantAddToBlackList`/`RemoveFromBlackList` × gms_v79
  cells were pinned to `CEntrustedShopDlg::AddBlackList @0x50588d` — but
  0x50588d is `sub_50588D`, a generic blacklist-name-entry dialog send
  (mode 0x2F, EncodeStr name; caller sub_505DEF, a CUtilDlgEx name prompt with
  a 20-entry cap), NOT the (non-existent) hired-merchant dialog. The byte
  layout coincidentally matched (both a bare name string), so the cell passed
  fixture while attributing a version-absent feature. Removed the two v79
  verify markers + audit reports + pinned evidence yamls; the disposition now
  makes the cells correctly n-a. matrix --check clean.
- **OPEN FINDING (separate, unrouted):** v79 mode `0x2F` (47) is a real
  client blacklist-by-name send (sub_50588D) that Atlas does NOT route — it is
  absent from the v79 CharacterInteractionHandle operations table and is not
  the field blacklist (FIELD_ADD=30 in v79). Likely a shared/personal-store
  blacklist dialog. Classifying + routing it is out of scope here (Atlas does
  not consume it today); noted so it is not silently lost.
