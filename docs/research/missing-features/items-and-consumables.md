# Missing features audit: items and consumables

Theme: item functionality — use effects, scrolls, cash items, pet items, equip mechanics (GMS <= v95, full v95 union).

**How this was audited.** Atlas sources: `services/atlas-consumables` (full read of `consumable/processor.go`, `data/consumable` spec model), `services/atlas-data/atlas.com/data/consumable/reader.go` (WZ field parse), `services/atlas-channel/atlas.com/channel/socket/handler/` (item/cash/pet handlers), `services/atlas-pets`, `services/atlas-mounts`, `services/atlas-rates`, `services/atlas-asset-expiration`, `services/atlas-merchant`, `libs/atlas-packet` (character data, asset model, cash/inventory/pet serverbound codecs), `libs/atlas-constants/item/constants.go`, seed templates `services/atlas-configurations/seed-data/templates/template_gms_{83,84,87,92,95}_1.json`, and the packet coverage matrix `docs/packets/audits/STATUS.md`. Reference behavior: Cosmic handlers (`UseCashItemHandler`, `ScrollHandler`, `UseCatchItemHandler`, `SkillBookHandler`, `ItemRewardHandler`, `ScriptedItemHandler`, `UseDeathItemHandler`, `TrockAddMapHandler`, `PetAutoPotHandler`, etc.) and extracted v83 WZ (`/wz/Item.wz/{Consume,Cash,...}` — spec children enumerated programmatically across all `Consume/*.img.xml`). Grep sweeps over `services/` **and** `libs/` are stated per finding. Item IDs are from Cosmic v83 WZ/`ItemId.java` unless marked otherwise.

**Big picture (as of 2026-08-07).** Atlas's `atlas-consumables` covers the potion/buff/cure/scroll/pet-food/summon-sack core well. `CharacterCashItemUseHandleFunc` (`services/atlas-channel/.../socket/handler/character_cash_item_use.go`) implements ~14 of the ~70 slot-item types its `GetCashSlotItemType` mapping enumerates — pet consumable, chalkboard, field-effect weather, note, teleport rock, item tag, sealing lock ×2, incubator, Vega's Spell ×2, point reset, Vicious Hammer, store search/Owl, plus the megaphone family in `character_cash_item_use_megaphone.go` — and is registered in 9 of 10 GMS templates (all but gms_12). The structural gaps that remain are the unimplemented slot types (the cube family is an explicit `not implemented` warn), the `SummonBag`/`TownScroll` registration hole on v87/v92/v95, and the largest of them: many `Item.wz` `spec` fields present in v83 data are neither parsed by `atlas-data` nor applied by `ApplyItemEffects`.

Legend for scope: S = one handler/branch, M = handler + service logic + events, L = new subsystem/multi-service.

---


## Wholly missing

### 3. Scripted items
- **Player experience:** items that open an NPC conversation when used (24 items in `Consume/0243.img.xml`, e.g. 2430000-2430005 — spec `script`+`npc`).
- **Expected:** Cosmic `ScriptedItemHandler.java`.
- **Atlas absence:** `atlas-data` parses `spec/script` and `npc` (`reader.go:74,161`) but grep `\.Script()` over `services/`: zero consumers; STATUS.md `SCRIPTED_ITEM` ❌ (566). atlas-npc-conversations has no item-triggered entry point (searched `scripted` in that service). Scope: **M**.

### 4. Monster catch items
- **Player experience:** use a catch item (13 items in `Consume/0227.img.xml`, 2270000+) on a low-HP mob to convert it into an item (Fish Net, ghost jars, etc.), with `bridleProp`/`mobHP` rules.
- **Expected:** Cosmic `UseCatchItemHandler.java`.
- **Atlas absence:** both clientbound writers are verified (`CATCH_MONSTER`/`CATCH_MONSTER_WITH_ITEM` ✅, STATUS.md 338–339; `BRIDLE_MOB_CATCH_FAIL` ✅ 104) and `atlas-data` parses `bridleMsgType/bridleProp/bridlePropChg/mobHP` (`reader.go:68-70,81`) — but grep `CatchMonster` outside `socket/writer`: only `main.go` writer registration. Serverbound `USE_CATCH_ITEM` ❌ all versions (569). Scope: **M**.

### 5. Solomon's Blessing / gach EXP tickets (stored-EXP items)
- **Expected:** Cosmic `UseSolomonHandler.java`, `UseGachaExpHandler.java`.
- **Atlas absence:** grep `solomon|GachaExp|gacha_exp` over `services/`+`libs/`: only the `gachExp` stat field encode in `libs/atlas-packet/character/data.go`. STATUS.md `USE_SOLOMON_ITEM` ❌ (664), `USE_GACHA_EXP` ❌ (666). Scope: **S/M**.

### 6. Water of Life (pet revive)
- **Expected:** Cosmic `UseWaterOfLifeHandler.java` (revives an expired pet).
- **Atlas absence:** grep `water.?of.?life|WaterOfLife` over `services/`+`libs/`: zero. STATUS.md `WATER_OF_LIFE` ❌ (613). Scope: **S/M** (atlas-pets has expiration already).

### 7. Remaining one-off cash types
All mapped in `GetCashSlotItemType` but unimplemented (fall to warn); Cosmic `UseCashItemHandler` branch given in parentheses. (Transformation/morph coupons, formerly the first row here, are implemented — see Present-but-partial #6. Pet name tag, formerly a row here, is implemented — see Present-but-partial #7.)
| Feature | Item(s) (v83 WZ) | Cosmic branch | Scope |
|---|---|---|---|
| Meso sacks | `Cash/0520.img.xml` (3) | 520 (gainMeso) | S |
| Congratulatory song / jukebox trigger | `Cash/0510` (1) | 510 (musicChange broadcast) — Atlas `PLAY_JUKEBOX` writer ✅ (STATUS 192) but grep `PlayJukebox`: no emitter | S |
| Pet skill items (add/remove pet loot skills) | category 519 | Cosmic petSkill flags | M |
| Duey quick-delivery item | `Cash/0533` (1) | 533 | S (duey itself is economy theme) |
| Name change / world transfer *items* | `Cash/0540` | 540 | M (note: cash-shop *purchase* codecs exist: `libs/atlas-packet/cash/serverbound/shop_operation_buy_name_change.go`, `..._buy_world_transfer.go`) |
| Maple Life character-creation items | `Cash/0543` (3) | 543 | M |
| MiuMiu travel store | `Cash/0545` (2) | 545 | M |
| Item-expiration extenders | `Cash/0550` (5) | 550 | S/M |
| Karma scissors | 5520000 (`Cash/0552`) | 552 (share-tag one trade) | M |
| Store permit (open hired merchant via item) | category 514→type 4 | 504-adjacent; atlas-merchant service exists (`shop/entity.go` stores `PermitItemId`) but the item-use entry is unimplemented — note `CashSlotItemTypeStoreSearch` (29, Owl) *is* now implemented; the permit type is not | S given merchant exists |
| Cosmetic coupons (hair/face/skin, types 1/2/3/35) | `Cash/0515x` | 506-family | M |

#### Extra-expression (emote) items — closed by task-247, and not a cash-item-use gap

Extra-expression items (`ClassificationExpression` = 516, `Item.wz/Cash/0516.img`,
`05160000`–`05160014`) are **not** routed through the cash-item-use handler, so a
`CashSlotItemType(6)` dispatch arm would be dead code.
`CDraggableItem::OnDoubleClicked` @`0x50814b` (GMS v95) checks
`get_etc_cash_item_type` first and calls
`CWvsContext::SendEtcCashItemUseRequest` @`0x508165`, whose `case 6:` @`0xa02c86`
issues `CWvsContext::SendEmotionChange(nItemID % 100 + 8, 0, -1)`. The keyboard
path is the same: `CUserLocal::UseFuncKeyMapped` `case 3u` @`0x933874`. GMS v87
matches (`SendEtcCashItemUseRequest` @`0xab4f91`).

The real gap was on the emote path — `CharacterExpressionHandleFunc` accepted any
emote id with no range check and no ownership check — and task-247 closes it.

### 8. Unapplied Item.wz `spec` effect families (use-tab consumables)
Enumerated every `spec` child across all v83 `Item.wz/Consume/*.img.xml` (counts = occurrences) and diffed against what `atlas-data/consumable/reader.go:118-162` parses and what `ApplyItemEffects` (`atlas-consumables/consumable/processor.go:112-182`) applies. **Not parsed at all** (so no downstream service can act):

| spec field | count | sample items | what breaks |
|---|---|---|---|
| `mesoupbyitem` | 45 | 2022124, 2022459-2022461, 2022529 | meso-drop-up potions do nothing |
| `itemupbyitem` | 61 | 2022462, 2022463, 2022530, 2022531 | item-drop-up potions do nothing |
| `expinc` | 4 | 2022442, 2022450-2022452 | instant-EXP items give 0 EXP |
| `exp` | 13 | 2370000-2370012 (EXP potions) | no effect |
| `mhpR`/`mmpR` (+`mhpRRate`/`mmpRRate`) | 10/9 | 2022198, 2022337, 2022366+ | max-HP/MP % buff items do nothing |
| `padRate`/`madRate`/`speedRate`/`accRate`/`evaRate`/`pddRate`/`mddRate` | 7 ea | 2022359, 2022365, 2022368+ | %-based buff items do nothing |
| `berserk`, `booster`, `repeatEffect` | 6 ea | 2022585-2022588, 2022616-2022617 | special combat buff items dead |
| `BFSkill` | 6 | 2022539, 2022542-2022549 | battlefield-skill items dead |
| `nuffSkill` | 3 | 2022164-2022166 | debuff-party items dead |
| `cp` | 3 | 2022157-2022159 | Monster Carnival CP potions give no CP |
| `ghost` | 3 | 2360000-2360002 (`0236.img.xml`) | ghost candies do nothing |
| `barrier` | 1 | 2022269 | — |
| `dojangshield` | 2 | 2022429, 2022541 | — |
| `respectPimmune`/`respectMimmune`/`respectFS` | 3/3/2 | 2386000, 2387000-2387003, 2385013 | immunity-pierce items dead |
| `party`/`inParty` | 8/2 | 2022160-2022163, 2022430-2022433 | party-wide buff items apply to self only/nothing |
| `defenseAtt`/`defenseState` | 41/15 | 2382003+, 2382021+ | — |
| `con` (map-conditional effect, `sMap`/`eMap`/`type`) | 277 | 2382000+ | map-restricted effects unenforced |
| `eventRate`/`eventPoint` | 1/1 | 2022500 | — |

**Parsed but never applied** (in the `SpecType` enum + reader, zero readers of the value outside model/rest — grep per constant over `services/`): `thaw` (`SpecTypeThaw`), `returnMapQR`, `ignoreContinent`, `expBuff` (2450000), `onlyPickup`, `randomMoveInFieldSet`, and the `morphRandom` table (`Morphs()` has zero callers). Scope: **M–L** in aggregate; each family individually S/M.

### 9. Potential / cube / enhancement system (v92+, part of the v95 union)
- **Player experience (v92/v95 tenants):** magnifying glass reveals potential (`ITEM_RELEASE_REQUEST`), potential scrolls (`ITEM_OPTION_UPGRADE_USE`), equip enhancement scrolls (`HYPER_UPGRADE_ITEM_USE`), plus the matching show-effect writers.
- **Evidence of expected:** STATUS.md rows exist **only** in the v92/v95/jms columns (⬜ for v83/84/87), which is version evidence these ops were introduced between v87 and v92: `HYPER_UPGRADE_ITEM_USE` ❌ (583), `ITEM_OPTION_UPGRADE_USE` ❌ (586), `ITEM_RELEASE_REQUEST` ❌ (590), `SHOW_ITEM_HYPER_UPGRADE_EFFECT` (238), `SHOW_ITEM_OPTION_UPGRADE_EFFECT` (247), `SHOW_ITEM_RELEASE_EFFECT` (249), `SHOW_RECOVERY_UPGRADE_COUNT_EFFECT` (263).
- **Atlas absence:** grep `potential` over `services/`+`libs/`: one unrelated comment in atlas-marriages. No potential lines in the asset model or inventory domain. Cosmic (v83) has no reference implementation — needs IDA/WZ work. Scope: **L**.

### 10. Equip item level/EXP (equip leveling)
- Packet layer supports it: `SetEquipmentMeta(slots, levelType, level, experience, hammersApplied, flag)` in `libs/atlas-packet/model/asset.go:138` — but grep `itemExp|levelInfo|itemLevel` over `services/`+`libs/`: no game logic ever grants equip EXP or levels. (Whether any GMS <= v95 equips actually use `levelInfo` — e.g. Timeless/Reverse at v87+ — is **unverified** from local WZ, which is v83-only.) Scope: **M**, contingent on version verification.

### 11. Effect items (`USE_ITEMEFFECT`)
- **Player experience:** toggling a cosmetic item effect (Cosmic `UseItemEffectHandler`/`CancelItemEffectHandler` — `setItemEffect` broadcast).
- **Atlas absence:** STATUS.md `USE_ITEMEFFECT` ❌ (539), `SHOW_ITEM_EFFECT` ❌ (258). Note Atlas's `CharacterItemCancelHandle` is the *stat-change buff cancel* (`CANCEL_ITEM_EFFECT` ✅, 561), a different op. Scope: **S/M**.

*(Cash-shop-surprise items and coupon codes are owned by the cash-shop/economy theme.)*

---

## Present but partial

### 1. Cash item use handler: ~14 of ~70 types implemented; cube family explicitly deferred
`character_cash_item_use.go` branches on PetConsumable(30), Chalkboard(32), FieldEffect/weather(16), Note(21), TeleportRock, ItemTag(25), Seal(26)+SealTimed, Incubator, Vega's Spell (pre-95 + 95), PointReset (tier1 + shared), ViciousHammer, StoreSearch(29), and the megaphone family; `CharacterCashItemUseHandle` is registered in every GMS template except gms_12 (counts across 12/48/61/72/79/83/84/87/92/95 = 0/1/1/1/1/1/1/1/1/1).

Everything not in that list hits the warn fallthrough, and the cube/potential family is an *explicit* rejection rather than a fallthrough (`character_cash_item_use.go:401-404`, `"attempted to use cube-family item [%d]; not implemented"` — see Wholly-missing #9). The remaining unimplemented types are itemized in Wholly-missing #7. There is also still a `// TODO for v83 there is a trailing updateTime`. Fix scope: each missing type as itemized above.

### 2. Summoning sacks and town scrolls: implemented, still not registered on v87/v92/v95
`ConsumeSummoningSack` and `ConsumeTownScroll` are fully implemented (`atlas-consumables/consumable/processor.go`; 283 sack items in `Consume/0210.img.xml`), but `CharacterItemUseSummonBagHandle`/`CharacterItemUseTownScrollHandle` registration counts across gms 12/48/61/72/79/83/84/87/92/95 are **0/0/1/1/1/1/1/0/0/0** — unwired on v87, v92, v95, and on gms_12/48. Both appear in the `gms_83 − gms_87` and `gms_83 − gms_95` handler diffs. Scope: **S**.

### 3. Morph potions: the 2212000 morph-*other* arm
Classification 221 routes through `ConsumeStandard`, and both fixed-`morph` potions and the `morphRandom` weighted table work. The morph-*other* arm does not:

> **2212000 "Maplemas Party Potion" morph-other packet flow.**
> Not covered by task-140. IDA-verified (v83_Me, `MapleStory_dump.exe`, evidence in
> `docs/tasks/task-140-morph-potion-routing/prd.md` §1): `CDraggableItem::OnDoubleClicked` gates
> `id/10000 == 221 && (id%10000)/1000 == 2` (2212xxx) into a dedicated target-picker dialog
> (`CUIRandomMorphDlg` via `CWvsContext::SendRandomMorphOtherRequest`) with its own serverbound
> request and clientbound response (`CWvsContext::OnRandomMorphRes`, "failed to find user" /
> "only in town" failure arms). The client never sends the normal use-item packet for 2212xxx, so
> the standard consume path is unreachable (task-140 still routes 221 uniformly, but 2212000 is never
> invoked through it). Needs its own packet-audit + implementation task: serverbound request handler,
> `OnRandomMorphRes` writer, town-only validation, target resolution, apply-morph-to-target (reuse
> task-140's `rollMorph` seam). Scope: feature-sized, per-version opcodes.

### 4. Scroll system: core solid, edge scrolls incomplete
Works: success/curse rolls, stat lines, slot/level bookkeeping, white scroll protection (hardcoded 2340000, matching v83 WZ `0234.img.xml`), clean slates (2049000–2049003 verified in `0204.img.xml`), chaos scrolls (stat-shuffle table `rollStatAdjustment`, `processor.go:778-804`), spikes/cold-protection, Legendary Spirit flag pass-through, `SHOW_SCROLL_EFFECT` writer ✅ (STATUS 220). Vega's Spell runs through this pipeline. "Guardian scroll" as a distinct <= v95 item could not be confirmed in Cosmic v83 sources/WZ (no `ItemId` constant, no `ScrollHandler` special case) — see Unverified.

### 5. Pet auto-pot: routed, but unvalidated
`PET_AUTO_POT` serverbound ✅ all versions (STATUS 681) via `PetItemUseHandle` (`socket/handler/pet_item_use.go`) — but it just forwards to the generic `RequestItemConsume`; unlike Cosmic `PetAutoPotHandler` it never checks a pet is spawned or that the pet has the auto-HP/MP skill items. Functionally the potion applies; abuse/validation gap only. The auto-HP/MP keymap slots are encoded (writers `CharacterKeyMapAutoHp/AutoMp` registered). Scope: **S**.

### 6. Transformation/morph coupons: implemented in task-219, with two inertness caveats
`CharacterCashItemUseHandle` now routes classification-530 items (`Cash/0530.img.xml`) into `ConsumeMorphCoupon` (task-219): `atlas-data`'s cash reader parses `spec/morph`+`spec/hp`, and `atlas-consumables` applies the morph plus an HP heal through the existing effect pipeline wherever the tenant's cash WZ carries a `Cash/0530` item with `spec/morph`. Two caveats keep it from being unconditionally live:
- **Inert on gms_12** — `template_gms_12_1.json` does not register `CharacterCashItemUseHandle` at all, the same pre-existing gap that already affects the whole cash-item-use family (Present-but-partial #1); not a regression from this task.
- **Inert for any tenant whose cash WZ was ingested before this change** — the reader change materialises `morph`/`hp` only for newly ingested data, so a pre-existing tenant's stored `5300000`-family items still lack those spec fields until re-ingested. Tracked as an operational follow-up in `docs/TODO.md` ("task-219 follow-up: cash WZ re-ingest for morph-coupon `spec/morph`/`spec/hp`"); until that runs, a coupon use on an un-reingested tenant consumes the item and applies nothing.

### 7. Pet name tag: implemented in task-224, across all ten GMS/JMS templates
`CharacterCashItemUseHandle` now routes classification-517 items (`Cash/0517.img.xml`, item `5170000`) into a rename saga (`PetNameTagUse`): the client-supplied name is validated (4–12 characters, trimmed), the lead pet's name is updated via atlas-pets' `RENAME` command, and `pet/clientbound/PetNameChanged` (`libs/atlas-packet/pet/clientbound/name_changed.go`) broadcasts the new name to the map. Unlike the morph coupon (#6), this feature reads no WZ `spec` value at all — `Cash/0517.img.xml` carries only `z`/`slotMax`/`cash`/icon canvases — so there is no ingest-order caveat. `PetNameChanged` is registered in all ten seed templates that carry a `PET_NAMECHANGE` matrix column (`gms_48`, `gms_61`, `gms_72`, `gms_79`, `gms_83`, `gms_84`, `gms_87`, `gms_92`, `gms_95`, `jms_185` — `gms_12` is outside this feature's version set entirely, unrelated to the separate gms_12 `CharacterCashItemUseHandle` gap in Present-but-partial #1). `PET_NAMECHANGE` is ✅ on all ten matrix columns (STATUS.md row 205) — including `gms_48`, where the opcode is present in the client (`CUser::OnPetPacket` case `'q'` / opcode 0x071 → `CPet::OnNameChanged`), contrary to an earlier plan-time assumption that v48 lacked it. See `docs/tasks/task-224-pet-name-tag/rollout.md` for the required live-tenant socket-config reconciliation step before enabling this on any existing tenant.

---

## Verified present (don't re-audit)

- **Standard consumables:** hp/mp/hpR/mpR recovery, cure of poison/darkness/weakness/seal/curse (cure-before-heal ordering, task-051), acc/eva/jump/speed/pad/mad/pdd/mdd buffs with `time` duration — `ApplyItemEffects` (`atlas-consumables/consumable/processor.go:112-182`), classifications 200/201/202/205.
- **Town/return scrolls** (`moveTo` + `returnMapId` fallback) — `ConsumeTownScroll` (v87/92/95 registration caveat: partial #2).
- **Summoning sacks** (weighted mob spawns) — `ConsumeSummoningSack` (same caveat).
- **Scroll core** incl. chaos, clean slate, white scroll, spikes/cold protection, legendary spirit flag, cursed destruction — `RequestScroll`/`ConsumeScroll`.
- **Pet food** (regular 212 hungriest-pet + cash 524 template-filtered; `PetCashFoodResult` ✅ STATUS 98), **pet spawn/multi-pet** (Follow-the-Lead skill gate `skill2.BeginnerMultiPetId/NoblesseMultiPetId`, 3-pet cap), **egg hatching**, **pet evolution** (weighted roll, task-089), **closeness/level awards**, **pet loot** (`PET_LOOT` ✅ 679) and **exclude list** (`PET_EXCLUDE_ITEMS` ✅ 682, `PET_EXCEPTION_LIST` ✅ 231), pet chat/commands — `services/atlas-pets`, atlas-channel pet handlers.
- **Mount food/revitalizer** — feed math, tiredness ticker, and `USE_MOUNT_FOOD` byte-verified on all 9 versions.
- **Chalkboards** (atlas-chalkboards + `ChalkboardUse` writer + close handler) and **weather/field-effect items** (saga `FieldEffectUse` → `BLOW_WEATHER` ✅ 191) — `CharacterCashItemUseHandle` now registered on every GMS template ≥48.
- **Chairs incl. portable** (atlas-chairs), including the server-authoritative recovery tick.
- **Asset expiration** across inventory/storage/cash-shop with `replace` item data from atlas-data (`services/atlas-asset-expiration`, README + `reader.go:176-179`).
- **Rate coupons** (cash EXP/drop coupons with duration + time windows, bonusExp equipment tiers) — `services/atlas-rates/.../character/item_tracker.go`.
- **Monster-book cards** consume-on-pickup (`spec/consumeOnPickup`, `reader.go:151`) and cover handler (`MonsterBookCover` handler, `MonsterBookSetCard/SetCover` writers).
- **Rechargeables** flag (bullets/stars) — `reader.go:173`.
- **Item buff cancel** (`CANCEL_ITEM_EFFECT` ✅ 561 → `CancelConsumableEffect`).

## Unverified / needs deeper data

- **"Guardian scroll"** (scroll preventing destruction on failure) — no Cosmic v83 evidence (no ItemId constant / ScrollHandler branch / obvious 204-range candidate). If it exists <= v95 it is post-v83 (unverified — general knowledge suggests protection scrolls are Big-Bang-era).
- **Equip durability ("tama")** — packet field exists v84+ and is stubbed to -1 (`asset.go:218`, IDA-verified comment); whether any GMS <= v95 item actually has durability is unverified (v83 WZ lacks it; no v87/v92/v95 WZ locally).
- **Equip item-level (`levelInfo`) usage <= v95** — see Wholly-missing #10; needs v87+/v95 WZ or IDA confirmation.
- **New Year Cards** (Cosmic has `NewYearCardHandler.java`; no STATUS.md row found for a NEW_YEAR op in my grep; version of introduction unverified — likely v95-era JMS/GMS, general knowledge).
- **Use-tab megaphones (classification 208, `ClassificationConsumableMegaphone`)** — which client op v83 sends for these (USE_ITEM vs USE_CASH_ITEM) was not determined; only the cash-tab (507/539) family is implemented.
- **`0238.img.xml` spec-bearing monster cards** (2382xxx with `con`/`defenseAtt`/`itemCode`) — the in-client meaning of these v83 card specs was not pinned down; flagged for the data pass.
- **Megaphone/Maple TV per-version wire layouts** — the implemented tiers are byte-fixtured and the legacy carve-outs are documented inline in `character_cash_item_use.go`; arms marked unsupported on a given legacy major are still un-reversed.
