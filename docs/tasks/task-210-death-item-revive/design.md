# Death Items — Wheel of Destiny revive & tomb effect — Design

Task: task-210-death-item-revive
Status: Draft
Created: 2026-08-10
Input: [`prd.md`](prd.md)

---

## 0. Summary of what the client research changed

The PRD was written with five open questions and an explicit instruction that no
packet layout be asserted until derived. That derivation is now done, and it
overturns three of the PRD's structural assumptions. Read this section before
the rest of the document; the architecture below follows from it.

| PRD assumption | Client evidence | Consequence |
|---|---|---|
| `USE_DEATHITEM` is an alternative *revive trigger* that consumes a charge (FR-2.2), racing the implicit `MAP_CHANGE` path (FR-2.3) | `CUIRevive::OnCreate` calls `CUserLocal::RequestUpgradeTombEffect()` at **dialog-construction time**, before the player has clicked anything. The actual revive is `CUIRevive::Revive` → `CField::SendTransferFieldRequest` = `MAP_CHANGE`. | `USE_DEATHITEM` is a **cosmetic broadcast request**, not a consume request. FR-2.2 and the whole single-consume problem of FR-2.3 dissolve. |
| `USE_DEATHITEM` is `n-a` on v72/v79 (FR-2.5) | v72 `CUserLocal::RequestUpgradeTombEffect` @`0x867654` emits opcode `52` (`0x34`); v79 @`0x8b2ff0` emits opcode `51` (`0x33`). Both were `sub_*` in the IDBs and are now named. | Two extra matrix cells and two extra template handler bindings. Registry rows must be added for v72/v79. |
| The wheel is consumed whenever the player holds one | `MAP_CHANGE` carries `bPremium`, set to `1` only from the dialog's OK button and `0` from Cancel (`CUIRevive::OnButtonClicked` → `Revive(1)` / `Revive(0)`). Atlas decodes this byte (`Change.Premium()`) and **ignores it**. | The live bug is bigger than "no protocol": a player who presses *Cancel* still has their wheel destroyed today. Honouring `premium` is the core behaviour fix. |

### Evidence index

All addresses below were read this session via `ida-pro-mcp`. Functions that were
unnamed have been renamed in the IDBs (v83 `0x95af8e`, v72 `0x867654`, v84
`0x999277`/`0x97297c`/`0x8cbdba`/`0x4aeb51`/`0xa73d4f`, v92 `0x8ee9f0`/`0x8cea40`/`0x81d770`).

| Version | IDB | `RequestUpgradeTombEffect` | opcode | `OnShowUpgradeTombEffect` |
|---|---|---|---|---|
| gms_v48 | `GMS_v48_1_DEVM.exe.i64` | absent — no reference to `5510000` anywhere in the image (scanned 1,198,699 insns, 0 hits) | n-a | absent |
| gms_v61 | `GMS_v61.1_U_DEVM.exe.i64` | absent — same scan, 1,615,793 insns, 0 hits | n-a | absent |
| gms_v72 | `GMS_v72.1_U_DEVM.exe.i64` | `0x867654` | `52` = `0x034` | `0x88d0e4`, dispatched from `CUserPool::OnUserRemotePacket` `0x87c046` case `177` = `0x0B1` |
| gms_v79 | `GMS_v79_1_DEVM.exe.i64` | `0x8b2ff0` | `51` = `0x033` | `0x8d9fe6`, dispatcher `0x8c8d4a` case `181` = `0x0B5` |
| gms_v83 | `MapleStory_dump.exe.i64` | `0x95af8e` | `53` = `0x035` | `0x983e40` |
| gms_v84 | `GMS_v84.1_U_DEVM.i64` | `0x999277` | `53` = `0x035` | `0x9c4206` |
| gms_v87 | `GMSv87_4GB.exe.i64` | `0x9dd673` | `0x38` | `0xa098f2` |
| gms_v92 | `GMS_v92_1_DEVM.exe.i64` | `0x8ee9f0` | `0x3B` | `0x9307e0` |
| gms_v95 | `GMS_v95.0_U_DEVM.exe.i64` | `0x908320` | `58` = `0x03A` | `0x954090`, dispatcher `0x94b390` case `221` = `0x0DD` |
| jms_v185 | `MapleStory_dump_SCY.exe.i64` | `0xa25fc9` | `0x2D` | `0xa57a4e` |

v48 and v61 are `n-a` for **both** ops, confirmed by absence of the item id
rather than by absence of a symbol name — the stronger evidence.

---

## 1. Derived wire layouts

### 1.1 `USE_DEATHITEM` (serverbound)

v95 `CUserLocal::RequestUpgradeTombEffect` @`0x908320`:

```
x = this->m_ptRevive.x;  y = this->m_ptRevive.y;
COutPacket(&oPacket, 58);
Encode4(&oPacket, 0x541370);   // 5510000 — Wheel of Destiny, hard-coded
Encode4(&oPacket, x);
Encode4(&oPacket, y);
SendPacket(...);
CUser::ShowUpgradeTombEffect(this, 5510000, x, y);   // local effect, client-side
```

Every version (v72, v79, v83, v84, v87, v92, v95, jms185) is byte-identical
modulo the opcode: three 4-byte little-endian fields, `itemId`, `x`, `y`. No
version divergence, so **no `MajorAtLeast` gate is needed** (FR-1.3 is satisfied
vacuously and must not be satisfied by inventing a gate).

`itemId` is a constant `5510000` in the client, not a player-chosen item. It is
still decoded and validated server-side rather than assumed.

### 1.2 `SHOW_UPGRADE_TOMB_EFFECT` (clientbound)

v95 `CUserRemote::OnShowUpgradeTombEffect` @`0x954090` reads three `Decode4`s
(`itemId`, `nPosX`, `nPosY`) and calls `CUser::ShowUpgradeTombEffect`. The
`characterId` is **not** read by the handler — `CUserPool::OnUserRemotePacket`
@`0x94b390` consumes a leading `Decode4` to resolve the remote user before
switching on the opcode. Every `CUserRemote::On*` op in this pool shares that
prefix.

Wire, therefore: `characterId:int, itemId:int, x:int, y:int`. v83 `0x983e40`
and v72 `0x88d0e4` decompile identically. No version divergence.

### 1.3 The two packets are a matched pair

The server's entire job for this feature's *visual* half is: on receiving
`USE_DEATHITEM(itemId, x, y)` from a session, emit
`SHOW_UPGRADE_TOMB_EFFECT(characterId, itemId, x, y)` to every **other** session
in the map. The reviving player needs nothing — they already ran
`CUser::ShowUpgradeTombEffect` locally inside `RequestUpgradeTombEffect`. That
settles **OQ-4: no local counterpart is sent.** Sending one would double the
effect on the owner's screen.

This collapses the coordinate-sourcing problem the PRD's FR-3.5 would have
created (the implicit `MAP_CHANGE` path has no x/y to broadcast). It also means
the tomb effect never needs to fire from the revive saga — there is no version
where the client omits `USE_DEATHITEM` but expects the effect, because v72 and
v79 both send it.

---

## 2. Architecture

Three separable units, deliberately kept independent. Each can be built,
tested, and reviewed without the others.

```
┌─ Unit A ── libs/atlas-packet ─────────────────────────────────┐
│ character/serverbound/use_death_item.go  UseDeathItem         │
│ character/clientbound/upgrade_tomb.go    ShowUpgradeTombEffect│
│   both: immutable struct + Encode + Decode + byte fixtures    │
└───────────────────────────────────────────────────────────────┘
             │ decoded                       ▲ encoded
             ▼                               │
┌─ Unit B ── atlas-channel socket/handler ──────────────────────┐
│ use_death_item.go — validate, then relay to other sessions.   │
│ NO state change, NO saga, NO item consumption.                │
└───────────────────────────────────────────────────────────────┘

┌─ Unit C ── atlas-channel respawn/processor.go ────────────────┐
│ premium gate · charge accounting · protect-on-die effect      │
│ Driven by MAP_CHANGE only. Never touched by Unit B.           │
└───────────────────────────────────────────────────────────────┘
```

The PRD's FR-2.3 asked for a "revive already in flight" guard to stop the two
paths double-consuming. **That guard is not built**, because Unit B does not
consume. Building it would be defending against a race that the client cannot
produce. The acceptance criterion "a test proves a single death consumes exactly
one charge even when both paths fire" is still met, and more strongly: a test
fires `USE_DEATHITEM` followed by `MAP_CHANGE` and asserts exactly one
`DestroyAsset` step, because only `MAP_CHANGE` emits one.

### 2.1 Unit B — the handler

```
UseDeathItemHandleFunc(l, ctx, wp):
  decode UseDeathItem{itemId, x, y}
  c ← character.GetById(s.CharacterId())
  reject (log at Warn, return) unless:
     c.Hp() == 0                       — must actually be dead
     item.IsWheelOfFortune(itemId)     — client hard-codes 5510000
     wheel asset present with Quantity() >= 1
  channelmap.ForOtherSessionsInMap(s.Field(), s.CharacterId(),
     session.Announce(l)(ctx)(wp)(ShowUpgradeTombEffectWriter)(
        NewShowUpgradeTombEffect(c.Id(), itemId, x, y).Encode))
```

The rejections matter: the packet is unauthenticated player input carrying
free-form coordinates, and it broadcasts to every other client in the map. An
unvalidated relay lets a living player spam tombstones at arbitrary map
positions. The HP and possession checks reduce that to "a dead player who owns a
wheel can show a tomb where the client says they died," which is the client's own
contract.

Coordinates are relayed as received rather than substituted from a server-side
position, because the server does not track `m_ptRevive` and the client's own
local effect uses exactly these values — substituting would desync owner and
bystanders. This is a deliberate, bounded trust of client input for a purely
cosmetic field.

### 2.2 Unit C — respawn processor

Current `Respawn` (`services/atlas-channel/atlas.com/channel/respawn/processor.go`)
does: find wheel → redirect target map → build saga that destroys 1 wheel, sets
HP 50, deducts exp, cancels buffs, warps. Four changes:

**C1 — honour `premium`.** `MapChangeHandleFunc`
(`socket/handler/map_change.go:54`) currently calls
`respawn.Respawn(channel, characterId, mapId)` and throws `p.Premium()` away.
The signature grows one parameter, `useDeathItem bool` (`p.Premium() != 0`). The
in-map redirect and the wheel consumption both become conditional on
`useDeathItem && hasUsableWheel` instead of `hasWheelOfFortune` alone.

Note the client sends `premium = 1` from button id 6 in *all three* dialog
branches, including the no-wheel notices — so `premium` is a permission, not a
proof. The possession check stays.

**C2 — charge accounting.** `hasWheelOfFortune` becomes a lookup that returns the
asset, and usability is `Quantity() >= 1`. `saga.DestroyAsset` with
`Quantity: 1, RemoveAll: false` already expresses a partial decrement, so the
existing step is correct as written; what is new is that the *pre*-decrement
quantity gates usability and the *post*-decrement quantity (`Quantity() - 1`) is
reported to the client. Same treatment for the protective item, which today is
destroyed unconditionally.

**C3 — protect-on-die effect.** When `findProtectiveItem` selects an item, emit
`CharacterProtectOnDieItemUseEffectBody` to the owner and
`...ForeignBody` to other sessions in the map. Arguments:

- `safetyCharm` = `item.IsSafetyCharm(id)` — true only for `5130000`.
- `usesRemaining` = post-decrement quantity, clamped to `byte`.
- `days` = whole days between now and `asset.Expiration()`, `0` when the
  expiration is the zero time. See §4 OQ-3 for why this is the defensible source.
- `itemId` = the template id; the codec drops it from the wire when
  `safetyCharm` is true (`effect.go:404`), so it is only observable for the ETC
  items.

**C4 — failure semantics.** A failed charge decrement must abort the revive
rather than grant a free in-map respawn (PRD §8). The decrement is a saga step,
so this is already the saga's behaviour: if `consume_wheel_of_fortune` fails the
saga does not proceed to `warp_to_spawn`. No new code; a test pins it.

### 2.3 What is NOT in Unit C

`is_fieldtype_upgradetomb_usable` (v95 `0x4b7a30`) is the client's own gate on
whether to even offer the wheel:

```
field types {1,3,4,5,7,10,11,15} → never;
otherwise: mapId/100000000 != 9 && mapId/1000 != 200090 && mapId/1000000 != 390
```

Mirroring this server-side is **out of scope** and left as a documented finding
(OQ-5, §4). The field-type axis is a client `CField` subclass with no server
equivalent, so a partial mirror (map-id ranges only) would be a half-gate
presented as a gate. The client already refuses to send `premium = 1` in those
maps, and C1 makes the server obey `premium`, which closes the practical hole
without inventing a server-side field-type model.

---

## 3. Alternatives considered

**A1 — Make `USE_DEATHITEM` the revive trigger (the PRD's FR-2.2), with a
per-death guard against `MAP_CHANGE`.** Rejected on evidence:
`RequestUpgradeTombEffect` fires from `OnCreate`, before the player chooses, and
the client then *also* sends `MAP_CHANGE` on either button. Treating the first
as a revive would revive players who pressed Cancel and would make the guard
load-bearing for correctness rather than a safety net. This is the option the
PRD assumed; it is wrong, and the cost of being wrong is a wheel destroyed
against the player's wishes on every single death near a wheel.

**A2 — Broadcast the tomb effect from the revive saga instead of relaying the
request.** Would need `x`/`y` for the death position, which the channel does not
hold, and would fire the effect at warp time rather than dialog time — visibly
late. Its only advantage would be covering versions where the client does not
send `USE_DEATHITEM`, and there are none. Rejected.

**A3 — Also emit `CHARACTER_EFFECT` mode `UPGRADE_TOMB_ITEM_USE` (21 on
v61–v87, 23 on v95, 20 on jms185; already in every template's `operations`
table).** This is a real, adjacent effect arm with no emitter. It is *not*
`SHOW_UPGRADE_TOMB_EFFECT` and the PRD does not ask for it. Left alone and
recorded here so a future reader does not mistake it for a gap this task
created.

**Recommendation: the three-unit split above.** It is smaller than the PRD's
plan, it removes a concurrency mechanism rather than adding one, and it fixes a
live player-facing bug (Cancel destroys your wheel) that the PRD did not know
existed.

---

## 4. Open questions — resolved and remaining

- **OQ-1 — When does the client send `USE_DEATHITEM`? RESOLVED.** At
  `CUIRevive::OnCreate`, unconditionally, when
  `is_fieldtype_upgradetomb_usable(fieldType, mapId)` and
  `CWvsContext::GetItemCount(5510000) > 0`. Not a button, not a consume. v95
  `0x83cea0`; identical structure in v84 `0x8cbdba` and v92 `0x81d770`. No live
  capture needed.

- **OQ-2 — Where do charges live? RESOLVED as inventory quantity.** The client
  gates the dialog on `CWvsContext::GetItemCount(5510000) > 0` — an inventory
  count, not a WZ field. The client's WZ model for the protection items,
  `CItemInfo::PROTECTONDIEITEM` (8 bytes: `nItemID:int`, `nRecoveryRate:int`),
  carries **no** use-count field. There is therefore no WZ-declared charge count
  for the client to read, and FR-4.1 stands as written: charges = quantity. No
  live `atlas-data` query is required.
  *Side finding, not in scope:* `nRecoveryRate` is the client's per-item revive
  recovery rate, while `createRespawnSaga` hard-codes `SetHP{Amount: 50}`.
  Recorded, not changed.

- **OQ-3 — What feeds `EffectProtectOnDie.days`? PARTIALLY RESOLVED.** The
  v83 `CUser::OnEffect` arm for mode 6 (`0x937e81`, jumptable case 6) reads
  `Decode1` ×3 — `safetyCharm`, then two bytes — and when `safetyCharm != 0`
  formats StringPool string `0x0B96` with the **third** byte first and the
  **second** byte second; when `safetyCharm == 0` it reads a further `Decode4`.
  So the existing codec's field order and conditional are structurally
  confirmed, and both bytes reach a user-visible message. Which of the two the
  message calls "days" is in `String.wz`, not the binary, and is **not proven
  here**. The implementation sources `days` from `asset.Expiration()` (whole
  days remaining, `0` when unset) because that is the only value on the asset
  that could mean "days"; if the live message renders it wrongly, the fix is a
  one-line swap and a note here — it is not a value invented to fill a slot.

- **OQ-4 — Local tomb effect? RESOLVED: no.** `RequestUpgradeTombEffect` calls
  `CUser::ShowUpgradeTombEffect` on itself before sending. The server must not
  echo to the owner.

- **OQ-5 — Field-limit gate on the wheel? ANSWERED, deliberately not
  implemented.** `is_fieldtype_upgradetomb_usable` @`0x4b7a30` (v95) is
  reproduced verbatim in §2.3. It is client-side; the server-side equivalent
  needs a field-type model Atlas does not have. Honouring `premium` (C1) makes
  the client's gate effective in practice. Recorded as a known, bounded gap.

### Post-implementation

Recorded after execution, against the state of the branch at PR time.

- **OQ-3 CLOSED — the field mapping is correct as implemented.** Re-read of the
  whole v83 `CUser::OnEffect` (@0x9377d9) rather than the mode-6 entry alone
  settles it without a live client. The arm reads `safetyCharm` (`v215`), then
  byte2 into `v54`, then byte3 into `v214`, and on `safetyCharm != 0` calls
  `ZXString<char>::Format(&iPacket, SP_2966_THE_EXP_DID_NOT_DROP_AFTER_USING_THE_SAFETY_CHARM_ONCE_%D_DAYS_%D_TIMES_LEFT, v214, v54)`.
  The named string pins the argument order: byte3 → "days", byte2 → "times
  left". That is exactly `EffectProtectOnDie{usesRemaining, days}`, so
  `expirationDays` feeding byte3 from `asset.Expiration()` is right and no swap
  is needed. Live confirmation from testing is consistent: a single
  non-expiring Safety Charm rendered "(0 days / 0 times left)" — quantity 1
  minus the consumed one is 0 uses, and a zero `Expiration()` is 0 days.
- **The same read also closed a gap the PRD never listed: mode 21.**
  `CUser::OnEffect` case 21 (@0x9387d0) reads one byte and CHATLOG_ADDs
  SP_5241 "You have used 1 Wheel of Destiny in order to revive at the current
  map. (%d left)". `CharacterUpgradeTombItemUseEffectBody` already existed in
  `libs/atlas-packet/character/effect_body.go:306` with no emitter — the exact
  situation `EffectProtectOnDie` was in before this task — and every one of the
  eight templates already carries `UPGRADE_TOMB_ITEM_USE` in the
  `CharacterEffect` operations table, so no template change was needed. The
  channel now announces it to the reviving player from the shared revive
  outcome. Owner only: `CUserPool::OnUserRemotePacket` (@0x9724f9) routes the
  foreign effect into the *same* `CUser::OnEffect` arm on the remote user
  object, so broadcasting mode 21 would print that first-person sentence in
  every bystander's chat log.
- **FR-5.1's foreign broadcast is deliberately NOT implemented.** The PRD asked
  for `EffectProtectOnDieForeign` to the rest of the map on the assumption that
  it rendered something over the dying character. It does not: because
  `CUserPool::OnUserRemotePacket` routes the foreign effect into the same
  `CUser::OnEffect` arm on the remote user object, mode 6 on a bystander's
  client only CHATLOG_ADDs the first-person SP_2966 sentence into that
  bystander's own chat log — someone else's "The EXP did not drop after using
  the Safety Charm once…" with no visual and no attribution. That is noise, not
  an effect, so only the owner announcement is sent. Same call for mode 21.
  `CharacterProtectOnDieItemUseEffectForeignBody` and
  `CharacterUpgradeTombItemUseEffectForeignBody` remain in `libs/atlas-packet`
  (codecs are protocol surface, not policy) with no emitter.
- **The only on-screen foreign feedback is the tomb effect.**
  `SHOW_UPGRADE_TOMB_EFFECT` (`CUserRemote::OnShowUpgradeTombEffect` @0x983e40 →
  `CUser::ShowUpgradeTombEffect`) is the one packet in this task that draws
  anything over another player, and it is broadcast from
  `CharacterUseDeathItemHandleFunc` as designed.
- **OQ-5 remains a deliberate, documented gap.** Unchanged from above; no
  field-type model was added.
- **Two template defects were fixed that the PRD did not contain**, both found
  during planning rather than by the client:
  - `template_gms_95_1.json` and `template_jms_185_1.json` each bound the
    writer name `CharacterEffect` **twice**, at two distinct opcodes (v95 `0xE0`
    and `0xE9`; jms185 `0xCC` and `0xD5`). `RegisterTenantWriterOptions` keys
    its table by writer *name*, so one silently won and the other's options were
    lost. The lower opcode — identified as the foreign arm by the registry — is
    now named `CharacterEffectForeign`, matching every other v61+ template.
    Without this, Unit C's foreign broadcast could not resolve a writer on two
    of the eight versions. The duplicate-binding guard does not catch this: it
    only bans the same name at the *same* opcode.
  - `template_gms_92_1.json` was missing **both** `CUser::OnEffect` writers.
    They were added with an `operations` mode table derived from the v92 binary
    itself (`CUser::OnEffect` @`0x8e5510`), not copied from a neighbour —
    necessarily, since v87 and v95 disagree (`PROTECT_ON_DIE_ITEM_USE` = 6 vs
    8). v92's value is **6**. Only that index is load-bearing for this feature;
    the rest of the 26-arm table was derived in the same pass but only a sample
    was independently re-verified.
- **A third defect surfaced during execution, in this task's own earlier work:**
  the two codec test files initially shipped `packet-audit:verify` markers for
  all eight versions *before* any cell had been verified, which
  `packet-audit matrix --check` was already failing as orphan markers at the
  base commit. Markers are now added per version as that version's cell is
  actually verified.
- **§6's broadcast-level assertions did not land as written.** The Unit B bullet
  called for asserting "exactly one foreign broadcast, correct
  `characterId`/`itemId`/`x`/`y`, and no announce to the owner's own session,"
  and the Unit C bullet called for asserting the owner's `EffectProtectOnDie`
  and other sessions' `EffectProtectOnDieForeign` announces. What shipped
  instead is table-driven tests over the pure predicates
  (`canShowTombEffect` in `socket/handler/use_death_item_test.go`,
  `planRespawn`/`usesRemaining`/`expirationDays` in `respawn/plan_test.go`) —
  the session/broadcast plumbing itself is untested. `context.md` §3
  pre-adjudicated the mocking difficulty for `planRespawn` specifically; it
  does not cover these broadcast-call assertions, so this is a real delta
  between design and branch, not a pre-agreed reduction in scope. The
  owner-exclusion invariant for Unit B (§1.3 OQ-4: no echo to the sender) is
  therefore currently structural — one `ForOtherSessionsInMap` call site in
  `socket/handler/use_death_item.go` and no owner-directed `session.Announce`
  call anywhere in the file — rather than test-enforced.
- **Tooling note for future packet work, not a defect of this branch:** the
  packet-audit harvest tool's `-ida-database` flag does not work against the
  live ida-pro-mcp server, and its fallback silently harvests whichever image
  the server considers active — which returned v72 provenance regardless of the
  requested target in every batch. Each version's export entries were therefore
  hand-constructed from direct `decompile(addr, database=<session>)` calls after
  proving the image with `survey_binary`. Left unfixed here as out of scope; a
  harvest trusted blindly would produce a silently false verification.

---

## 5. Registry, template, and matrix work

### 5.1 Registry additions (new rows, both `provenance: ida-discovered`)

`docs/packets/registry/gms_v72.yaml` and `gms_v79.yaml` currently have holes
where `USE_DEATHITEM` belongs (v72 serverbound has `FACE_EXPRESSION` at 50 and
nothing at 51/52; v79 has `FACE_EXPRESSION` at 49 and nothing at 50/51):

| Registry | op | direction | opcode | fname | ida.address |
|---|---|---|---|---|---|
| `gms_v72.yaml` | `USE_DEATHITEM` | serverbound | 52 | `CUserLocal::RequestUpgradeTombEffect` | 8812116 (`0x867654`) |
| `gms_v79.yaml` | `USE_DEATHITEM` | serverbound | 51 | `CUserLocal::RequestUpgradeTombEffect` | 9121776 (`0x8b2ff0`) |

The other twelve rows already exist. `feature-na-evidence.yaml` gains v48/v61
entries for both ops citing the image-wide absence of `5510000`.

### 5.2 Template bindings — all sixteen slots verified free

Checked against every template's existing `opCode` set; no collisions, so
`tools/template-duplicate-binding-guard.sh` and
`tools/template-opcode-order-guard.sh` are satisfied by inserting at the sorted
position.

| Template | handler `opCode` | writer `opCode` |
|---|---|---|
| `template_gms_72_1.json` | `0x34` | `0xB1` |
| `template_gms_79_1.json` | `0x33` | `0xB5` |
| `template_gms_83_1.json` | `0x35` | `0xC3` |
| `template_gms_84_1.json` | `0x35` | `0xC7` |
| `template_gms_87_1.json` | `0x38` | `0xD0` |
| `template_gms_92_1.json` | `0x3B` | `0xDF` |
| `template_gms_95_1.json` | `0x3A` | `0xDD` |
| `template_jms_185_1.json` | `0x2D` | `0xC9` |

Handler entries carry `"validator": "LoggedInValidator"` — a handler with no
validator is silently dropped at load. Writer entries carry the `fname`
`CUserRemote::OnShowUpgradeTombEffect`. `services: ["channel"]` on both.

### 5.3 Matrix

Sixteen cells promote, not fourteen: 8 × `USE_DEATHITEM` (v72, v79, v83, v84,
v87, v92, v95, jms185) and 8 × `SHOW_UPGRADE_TOMB_EFFECT` (same set).
`STATUS.md:579` currently shows `⬜` with no opcode for v72/v79 on
`USE_DEATHITEM`; those become `0x034`/`0x033` and `✅`. v48/v61 stay `n-a` on
both rows. `STATUS.md` and `status.json` are regenerated, never hand-edited.

### 5.4 The v92 protect-on-die blocker

`template_gms_92_1.json` has **no `CharacterEffect` writer entry at all** — so
`CharacterProtectOnDieItemUseEffectBody`, which resolves its mode through
`WithResolvedCode("operations", "PROTECT_ON_DIE_ITEM_USE")` on that writer's
options, cannot be emitted on v92. Every other v61+ template has it, with
`PROTECT_ON_DIE_ITEM_USE` = 6 (v61–v87, jms185 = 6; v95 = 8) and the full
26-entry operations table.

This is a pre-existing template gap, not one this task creates, and it is
producible: the v92 `CharacterEffect` writer entry (opcode + full `operations`
table) is derived from the v92 client's `CUser::OnEffect` jumptable the same way
§1 derived the two new codecs, and added in this task. Emitting the effect on
seven versions and silently skipping the eighth would be exactly the "documented
gap" the project bans.

---

## 6. Testing

**Unit A — codecs.** Per-version byte fixtures with the `packet-audit:verify`
marker, encode and decode, driven by `packet-verifier` per cell. Sixteen cells.

**Unit B — handler.** Table-driven, using the project Builder pattern for
session/character/inventory fixtures:
- dead + owns wheel + `itemId` 5510000 → exactly one foreign broadcast, correct
  `characterId`/`itemId`/`x`/`y`, and **no** announce to the owner's own session.
- alive → no broadcast.
- dead, no wheel → no broadcast.
- dead, wheel at quantity 0 → no broadcast.
- `itemId` not 5510000 → no broadcast.

**Unit C — respawn.**
- `premium = 1` + usable wheel → target map is the current map; saga contains
  `consume_wheel_of_fortune` with `Quantity: 1, RemoveAll: false`.
- `premium = 0` + usable wheel → target map is `mapData.ReturnMapId()`; **no**
  consume step. (Regression test for the bug C1 fixes.)
- `premium = 1`, no wheel / quantity 0 → return map, no consume.
- wheel at quantity 3 → decrement, asset survives; at quantity 1 → asset
  destroyed. Both through the same `DestroyAsset` step.
- protective item consumed → owner receives `EffectProtectOnDie` and other
  sessions receive `EffectProtectOnDieForeign`, `usesRemaining` = pre − 1,
  `safetyCharm` true only for 5130000.
- `USE_DEATHITEM` then `MAP_CHANGE` for one death → exactly one `DestroyAsset`
  step across both (the FR-2.3 acceptance criterion, met structurally).
- failed consume step → no `warp_to_spawn`.

**Guards.** `tools/template-opcode-order-guard.sh`,
`tools/template-duplicate-binding-guard.sh`,
`tools/template-movement-types-guard.sh`, `tools/lint.sh --check`,
`tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/skill-job-id-guard.sh`, plus `go test -race`, `go vet`, `go build`, and
`docker buildx bake atlas-channel` / `atlas-configurations`.

---

## 7. Deltas from the PRD

Recorded so the plan phase does not silently re-import the superseded text.

| PRD | Change | Reason |
|---|---|---|
| FR-2.2 | `USE_DEATHITEM` no longer drives the revive outcome | §0, `CUIRevive::OnCreate` |
| FR-2.3 | No single-consume guard is built; the criterion is met by construction | only one path consumes |
| FR-2.5 | v72/v79 gain the opcode; `n-a` claim withdrawn | v72 `0x034`, v79 `0x033` |
| FR-3.4 | Broadcast is a relay of the client's own coordinates, owner excluded | §1.3, OQ-4 |
| FR-3.5 | Broadcast fires from the handler, not the revive outcome | §1.3 |
| FR-1.3 | No version gate — every version is byte-identical | §1.1 |
| FR-6.1 | 16 cells, not 14 | §5.3 |
| — | New: honour `Change.Premium()`; new: v92 `CharacterEffect` writer entry | §2.2 C1, §5.4 |
