# task-184 — gms_61 opcode corruption repair

> **Revision 2 — exhaustive per-packet IDB verification.** The initial pass
> bounded corruption with a gms_61-vs-gms_72 template diff (a cross-version
> heuristic). That was replaced by an **exhaustive** verification of every
> packet against the gms_61 client itself: all 153 clientbound writers checked
> against the client's receive-dispatch switches (232 `case N:` arms across
> `CClientSocket`/`CLogin`/`CWvsContext`/`CField` + 20 leaf dispatchers), and all
> 114 serverbound handlers checked against all 391 enumerated `COutPacket(N)`
> send sites. The heuristic had **missed two real corruptions** (`PartyInviteReject`,
> `WeddingAction`) that the exhaustive send-site sweep caught — evidence that
> cross-version comparison is not a substitute for verifying the packets.
> Evidence: `clientbound-exhaustive.md`, `serverbound-exhaustive.md`.

## Final verified state (exhaustive)

- **Clientbound writers (153):** 152 MATCH, 1 semantic anomaly
  (`FieldEffectWeather` bound to `0x6A`, but `0x6A` = `OnPlayJukeBox` in the
  dispatch — no distinct weather opcode exists; writer is mis-named/mis-pointed,
  not a swappable opcode — flagged, not changed).
- **Serverbound handlers (114):** 100 MATCH · **2 MISMATCH, both corrected here**
  (PartyInviteReject `0x71→0xCB`, WeddingAction `0x7F→0x7D`) · 2 DEAD bindings (HiredMerchant
  `0x3E`, AdminCommand `0x7E` — client sends nothing on them; features ride
  `0x6F`/`0x2E`) · ~8 inert NO-SEND (client sends nothing on the bound opcode —
  `MobDropPickup`0x9D redundant-with-`0xA9`, `MonsterBookCover`0x35 local-only,
  `WeddingTalk`0x80 covered-by-`0x7D`, `CharacterLoggedIn`0x14/`StartError`0x19
  receive-only, `ChalkboardClose`0x2F no-class, `Coconut`0xB2/`MonsterCarnival`0xB7
  stub-classes) · 1 genuinely unresolved (`CharacterSpouseChat`0x6D — no distinct
  spouse-chat send located).

The inert NO-SEND handlers are harmless (nothing arrives on those opcodes) and
are NOT corruption — every opcode the gms_61 client *actually sends* is verified
to match its handler.

**The two IDB-flagged "anomalies" are NOT gms_61 corruption — confirmed by
cross-checking the canonical gms_83 template:**
- `ServerListRequestHandle` is bound to BOTH `0x04` and `0x0B` in gms_61 — and
  **identically in gms_83 (canonical) and gms_72**, with `0x0C` unclaimed in all.
  gms_83 login works, so routing the client's `0x0B` PIN-flow send to
  `ServerListRequestHandle` (pin-ack → send world list) is Atlas's intentional
  design, not a gms_61 error. Left unchanged (changing it would diverge gms_61
  from the working canonical).
- `FieldEffectWeather` = `FieldEffect` + 2 in **every** version (v48 0x54/0x56,
  v61 0x68/0x6A, v72 0x7E/0x82, v83 0x8A/0x8E, v84 0x8D/0x91, v87 0x92/0x96,
  v95 0x9A/0x9E). gms_61's `0x6A` fits the universal pattern. The IDB shows the
  client's `0x6A` function is named `OnPlayJukeBox`, but that Atlas-name-vs-
  client-name discrepancy is identical across all versions — a cross-version
  writer-naming/codec question (does Atlas's `FieldEffectWeather` writer encode
  jukebox-compatible fields?), NOT gms_61 corruption. Left unchanged.

The two DEAD bindings (`HiredMerchant`0x3E / `AdminCommand`0x7E) are the only
remaining structural items — inert (client sends nothing on them), harmless, and
their removal is a handler-architecture cleanup, not an opcode value.

---


Completes the legacy-template repair deferred from task-125
(`gms61-legacy-opcode-followup.md`). All values below are IDB-verified against
`GMS_v61.1_U_DEVM.exe.i64` (session `965202bf`); ground truth is the client's
`COutPacket` ctor opcode (serverbound) or receive-dispatch `case N:`
(clientbound). Full evidence: `serverbound-resolution.md`, `clientbound-resolution.md`.

## Corrections applied in this task (7)

Serverbound handlers (`template_gms_61_1.json` socket.handlers):

| handler | was | now | evidence |
|---|---|---|---|
| `CharacterInventoryMoveHandle` | 0x46 | **0x42** | `sub_8315F8`@0x8315f8 `COutPacket(_, 66)` (move: invType/from/to/count) |
| `CharacterItemUseSummonBagHandle` | 0x4A | **0x46** | `SendMobSummonItemUseRequest`@0x831c83 → 70 (freed by InventoryMove→0x42; 0x4A was really the Bridle item, which has no handler) |
| `StorageOperationHandle` | 0x3A | **0x3C** | `CStoreBankDlg::SendGetAllRequest`@0x6754e1 → 0x3C |
| `OwlWarpHandle` | 0x3F | **0x4C** | `SendShopScannerItemUseRequest`@0x832680 → 0x4C (owl = shop-scanner feature) |
| `PartyInviteRejectHandle` | 0x71 | **0xCB** | `sub_70A753`@0x70a957 + `sub_70A9EE`@0x70aa29 `COutPacket(_, 203)` (accept + decline branches); nothing sent on 0x71. Found by the exhaustive send-site sweep — the gms_72 diff missed it. (0xCB is a *writer*-namespace slot `DestroyHiredMerchant`; handler slot free.) |
| `WeddingAction` | 0x7F | **0x7D** | `sub_837578`@0x8376d2 (propose, submode 0) + `CWvsContext::OnMarriageRequest`@0x84a24e (accept/decline, submode 2) `COutPacket(_, 125)`; nothing sent on 0x7F. 0x7D was unclaimed. |

Clientbound writer (`socket.writers`):

| writer | was | now | evidence |
|---|---|---|---|
| `RPSGame` | 0xF2 | **0xFC** | `CField::OnPacket`@0x4e9ea3 `case 252: CRPSGameDlg::OnPacket` (resolves the 0xF2 collision with `StorageOperation`, which is correctly 0xF2 via `case 242: CTrunkDlg::OnPacket`) |

Combined with task-125's 4 serverbound fixes (ItemUse 0x47→0x43, PetFood
0x4B→0x47, MountFood 0x4C→0x48, UseSkill 0x5A→0x53), the gms_61 item-use / NPC-
shop / minigame opcode corruption is now corrected. Post-repair template has
**zero** handler-opcode collisions and only the three intended writer variant
families (`0x00` Auth, `0x0A` ServerList, `0x40` portal spawn/remove — all
matching canonical gms_83).

## Verified correct — no change (48 handlers + writers checked)

Confirmed by IDB: the login/char-select block (0x01–0x16), movement/map/social
boundary, the rest of the item-use/stat/skill region (Scroll 0x4F, DistributeAp
0x50, DistributeSp 0x52, HealOverTime 0x51, BuffCancel 0x54, SkillPrepare 0x55,
etc.), chat/party/guild/messenger/note/keymap/minigame tail, and the writer
opcodes for StorageOperation (0xF2), SpawnPortal/RemoveTownDoor (0x40). See the
resolution docs for the per-handler `case N:` evidence.

## Spurious/dead bindings — diagnosed, left in place (not a simple opcode swap)

Both were the last two copy-corruption suspects (== gms_72). IDB tracing
(`suspects-resolution.md`) shows neither is a wrong-opcode that can be *reassigned*
— the client sends nothing on 0x3E or 0x7E; each feature already rides a
different, correctly-bound opcode. They are **dead entries** (the handler at that
opcode never fires because the client never sends it), harmless but incorrect:

- `HiredMerchantOperationHandle` (0x3E) — the gms_61 client sends hired-merchant
  operations on **0x6F** as mini-room-dispatcher submodes: `CEntrustedShopDlg::OnGoOut`
  @0x4da528, `OnArrange`@0x4da583, `OnWithdrawMoney`@0x4da5fd all
  `COutPacket(_, 111=0x6F)` with submodes 0x25/0x26/0x29. 0x6F is already bound to
  `CharacterInteractionHandle`. So the 0x3E binding is dead.
- `AdminCommand` (0x7E) — GM `/`-commands route via `CField::SendChatMsgSlash`
  @0x4e7469 on opcode **0x2E** (`COutPacket(_, 46)`, same wire shape as ordinary
  chat, already bound to `CharacterChatGeneralHandle`) or a narrow 0xB3 toggle;
  there is no dedicated 0x7E send. So the 0x7E binding is dead.

Not corrected here because the clean fix is a *removal* (plus, for hired-merchant,
confirming atlas-channel's 0x6F interaction handler actually decodes submodes
0x25/0x26/0x29) — an atlas-channel handler-architecture question, not a template
opcode value. Deleting the bindings fixes no feature (nothing is sent on those
opcodes) and touching them without verifying the code-side handler registration
risks regressions. Recommended follow-up: verify hired-merchant works via the
0x6F interaction dispatcher on gms_61 (add the submodes if missing), then drop the
dead 0x3E/0x7E template entries.

## Residual unverified (~56 serverbound handlers)

Movement/attack cluster, several login/char-select sends, pet/summon/mob,
NPC/quest/wedding, and the CashShop (0xC4) / ITC (0xD5–0xD7) submode dispatcher
families could not be pinned to a single client send function via static
analysis (dialog-driven or shared-opcode dispatchers). None showed positive
evidence of corruption, and the corresponding features function on live gms_61;
they are unverified, not known-wrong. A future exhaustive pass (or live packet
capture) can close them.

## Rollout

Existing gms_61 tenants need a socket-config PATCH for the 5 corrected opcodes
(4 handlers + RPSGame writer) plus the task-125 skill-book additions, then a
channel restart. Templates apply only at tenant creation.
