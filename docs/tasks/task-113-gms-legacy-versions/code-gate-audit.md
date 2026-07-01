# Cross-Version Code-Gate Audit (FR-7)

**Single source of truth for task-113 version-gate enumeration.**
Later passes (Stage F) fill the `v79 / v72 / v61 / v48` columns and add a `Correct?/Action`
verdict per row. **No fix to a gate may alter the existing v83/v84/v87/v95/JMS185 evaluation.**

---

## Pre-computed evaluation facts (for Stage F — do NOT fill columns here)

All four target legacy versions satisfy `48 ≤ major ≤ 79 < 83`:

| Predicate form | v79 | v72 | v61 | v48 |
|---|---|---|---|---|
| `MajorVersion() >= 83` / `>= 84` / `>= 87` / `>= 90` / `>= 95` | false | false | false | false |
| `MajorVersion() > 82` / `> 87` | false | false | false | false |
| `MajorVersion() > 28` / `> 12` | **true** | **true** | **true** | **true** |
| `MajorVersion() <= 28` / `< 28` | false | false | false | false |
| `MajorVersion() <= 12` | false | false | false | false |
| `MajorVersion() == 0` | false | false | false | false |
| `MajorVersion() >= 73` | **true** | false | false | false |
| `MajorVersion() != 83` / `!= 90` | **true** | **true** | **true** | **true** |
| `MajorVersion() <= 87` / `< 84` / `< 87` / `< 95` / `<= 95` | **true** | **true** | **true** | **true** |

> **WARNING — `>= 73` is the ONLY intra-legacy discriminator found.** It evaluates **true** for
> v79 and **false** for v72, v61, and v48. This gate most likely requires per-version attention
> during Stage F.

> **Off-by-one risk:** see `bug_majorversion_gt83_is_off_by_one_v87` in project memory.  
> Fixes must encode the *intended* version range, not a coincidentally correct boundary.
> Confirm semantics from IDA before changing any predicate.

> OQ-6 note: The boolean evaluations above are analytically correct for `48 ≤ major ≤ 79`, but
> Stage F must still CONFIRM each gate's correctness against IDA/behavior — do not assume all
> true/false evaluations automatically mean the gate is correctly coded.

> **Note on original grep coverage:** The staged grep covered `MajorVersion()` numeric comparisons
> and `Region()=="GMS"` patterns. An extended grep was run for `MajorAtLeast()`, `MajorAtMost()`,
> and `IsRegion("GMS")` variants — those additional gates are included in this table.
> Every file:line was verified against actual grep output (not invented).

---

## Table columns

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|

_See sectioned tables below._

---

## libs/atlas-packet/buddy

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| buddy/invite: hasJobLevel gate (job field present if non-GMS or GMS>=87) | `libs/atlas-packet/buddy/clientbound/invite.go:51` (×2 lines 51,77) | `Region()!="GMS" \|\| MajorVersion()>=87` | FALSE → job field absent (GMS<87) | | | | yes | — (v79<87; field is a v87+ addition, = v83 body) |
| buddy/error: GMS-only codec branch (8 instances) | `libs/atlas-packet/buddy/clientbound/error.go:139` (×8 lines 139–244) | `IsRegion("GMS")` | TRUE → GMS codec | | | | yes | — (v79 is GMS; BUDDYLIST family fixture-verified `e4e04f902`) |

---

## libs/atlas-packet/cash

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| cash/query_result: GMS+>12 fields | `libs/atlas-packet/cash/clientbound/query_result.go:42` (×2 lines 42,54) | `Region()=="GMS" && MajorVersion()>12` | TRUE → fields present | | | | yes | — (v79>12) |
| cash/shop_inventory: GMS>=95 or JMS inventory format | `libs/atlas-packet/cash/clientbound/shop_inventory.go:133` | `(Region()=="GMS" && MajorVersion()>=95) \|\| Region()=="JMS"` | FALSE → legacy format | | | | yes | — (v79<95; v95+ format is a later addition) |
| cash/shop_open: very-old version (<=12) branch | `libs/atlas-packet/cash/clientbound/shop_open.go:45` (×2 lines 45,120) | `MajorVersion()<=12` | FALSE → skip pre-13 field | | | | yes | — (v79>12; branch is for <=v12 only) |
| cash/shop_open: GMS region-only fields (no version guard) | `libs/atlas-packet/cash/clientbound/shop_open.go:36` (×6 lines 36,44,97,111,119,180) | `Region()=="GMS"` | TRUE → GMS fields | | | | yes | — (v79 is GMS) |
| cash/shop_open: GMS+>12 or JMS fields (~10 instances) | `libs/atlas-packet/cash/clientbound/shop_open.go:52` (×10 lines 52,60,66,85,94,127,135,141,162,177) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | TRUE → fields present | | | | yes | — (v79>12) |
| cash/shop_open: GMS+>12 only (2 instances) | `libs/atlas-packet/cash/clientbound/shop_open.go:90` (×2 lines 90,170) | `Region()=="GMS" && MajorVersion()>12` | TRUE → fields present | | | | yes | — (v79>12) |
| cash/serverbound/item_use: GMS>=95 decode paths | `libs/atlas-packet/cash/serverbound/item_use.go:38` (×2 lines 38,50) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → legacy decode | | | | yes | — (v79<95) |
| cash/shop_operation_buy: GMS>=87 field | `libs/atlas-packet/cash/serverbound/shop_operation_buy.go:58` (×2 lines 58,88) | `Region()=="GMS" && MajorVersion()>=87` | FALSE → field absent | | | | yes | — (v79<87) |
| cash/shop_operation_buy_couple: GMS>=95 field | `libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:57` (×2 lines 57,90) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |
| cash/shop_operation_buy_friendship: GMS>=95 field | `libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:57` (×2 lines 57,90) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |
| cash/shop_operation_gift: GMS>=87 field | `libs/atlas-packet/cash/serverbound/shop_operation_gift.go:66` (×2 lines 66,99) | `Region()=="GMS" && MajorVersion()>=87` | FALSE → field absent | | | | yes | — (v79<87) |
| cash/shop_operation_gift: GMS>=95 field | `libs/atlas-packet/cash/serverbound/shop_operation_gift.go:60` (×2 lines 60,93) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |
| cash/shop_operation_rebate_locker_item: GMS>=95 field | `libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go:53` (×2 lines 53,82) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |

---

## libs/atlas-packet/character

### clientbound

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| character/attack: GMS>=95 attack fields | `libs/atlas-packet/character/clientbound/attack.go:107` (×2 lines 107,165) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → legacy | | | | yes | — (v79<95; attacks fixture-verified `8b8174034`) |
| character/damage: GMS>=95 damage fields | `libs/atlas-packet/character/clientbound/damage.go:55` (×2 lines 55,78) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → legacy | | | | yes | — (v79<95; damage fixture-verified `8a83c379a`) |
| character/expression (cb): GMS>87 expression fields | `libs/atlas-packet/character/clientbound/expression.go:62` (×2 lines 62,80) | `Region()=="GMS" && MajorVersion()>87` | FALSE → fields absent | | | | yes | — (v79<=87; expression fixture-verified `8a83c379a`) |
| character/info: GMS<=87 or JMS info field | `libs/atlas-packet/character/clientbound/info.go:129` (×2 lines 129,203) | `(Region()=="GMS" && MajorVersion()<=87) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79<=87; char-info fixture-verified `45116cdcb`) |
| character/info: GMS>=87 or JMS info field (MajorAtLeast) | `libs/atlas-packet/character/clientbound/info.go:139` (×2 lines 139,213) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | FALSE → field absent | | | | yes | — (v79<87) |
| character/item_upgrade: GMS>87 upgrade fields | `libs/atlas-packet/character/clientbound/item_upgrade.go:91` (×2 lines 91,114) | `Region()=="GMS" && MajorVersion()>87` | FALSE → fields absent | | | | yes | — (v79<=87; items fixture-verified `b829593ee`) |
| character/item_upgrade: GMS>87 or JMS upgrade fields | `libs/atlas-packet/character/clientbound/item_upgrade.go:98` (×2 lines 98,119) | `(Region()=="GMS" && MajorVersion()>87) \|\| Region()=="JMS"` | FALSE → fields absent | | | | yes | — (v79<=87) |
| character/list: GMS<=28 legacy field | `libs/atlas-packet/character/clientbound/list.go:56` (×2 lines 56,91) | `Region()=="GMS" && MajorVersion()<=28` | FALSE → skip pre-29 field | | | | yes | — (v79>28; charlist fixture-verified `9364e3c45`) |
| character/list: GMS region-only inner gate | `libs/atlas-packet/character/clientbound/list.go:61` (×2 lines 61,96) | `Region()=="GMS"` | TRUE → GMS slots field | | | | yes | — (v79 is GMS) |
| character/list: any-region >87 field | `libs/atlas-packet/character/clientbound/list.go:63` (×2 lines 63,98) | `MajorVersion()>87` | FALSE → field absent | | | | yes | — (v79<=87) |
| character/spawn: GMS>87 or JMS spawn field | `libs/atlas-packet/character/clientbound/spawn.go:79` (×2 lines 79,193) | `(Region()=="GMS" && MajorVersion()>87) \|\| Region()=="JMS"` | FALSE → field absent | | | | yes | — (v79<=87; spawn fixture-verified `8a83c379a`) |
| character/spawn: GMS<95 inner block | `libs/atlas-packet/character/clientbound/spawn.go:135` (×2 lines 135,233) | `Region()=="GMS" && MajorVersion()<95` | TRUE → inner block present | | | | yes | — (v79<95) |
| character/spawn: GMS region-only inner gate | `libs/atlas-packet/character/clientbound/spawn.go:141` (×2 lines 141,239) | `Region()=="GMS"` | TRUE → GMS block | | | | yes | — (v79 is GMS) |
| character/spawn: 2nd-effect byte, now `>=83 && <=87` (was `<=87`) | `libs/atlas-packet/character/clientbound/spawn.go:153` (Encode; Decode :256) | `Region()=="GMS" && MajorVersion()>=83 && MajorVersion()<=87` | FALSE → no 2nd-effect byte | | | | yes (FIXED) | Old `<=87` wrote the 2nd (dragon) effect byte for v79 → misaligned. Campaign added `>=83` lower bound (`0225cd68e`); v79 CUserRemote::Init @0x8d5f67 has ONE effect byte. v84..v87 unchanged. |
| character/spawn: any-region >87 field | `libs/atlas-packet/character/clientbound/spawn.go:145` (×2 lines 145,243) | `MajorVersion()>87` | FALSE → field absent | | | | yes | — (v79<=87) |
| character/spawn: GMS>=87 (MajorAtLeast) field | `libs/atlas-packet/character/clientbound/spawn.go:85` (×2 lines 85,199) | `IsRegion("GMS") && MajorAtLeast(87)` | FALSE → field absent | | | | yes | — (v79<87) |
| character/status_message: GMS>=95 status field | `libs/atlas-packet/character/clientbound/status_message.go:528` (×2 lines 528,561) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → v95 fields absent | | | | yes | — (v79<95; status fixture-verified `f1e3a5b56`) |
| character/view_all: GMS>87 view-all field | `libs/atlas-packet/character/clientbound/view_all.go:83` (×2 lines 83,103) | `Region()=="GMS" && MajorVersion()>87` | FALSE → field absent | | | | yes | — (v79<=87; view-all fixture-verified `394b01d2c`) |

### data

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| character/data: GMS>28 or JMS (dominant codec gate, ~23 instances) | `libs/atlas-packet/character/data.go:114` (×23 instances) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | TRUE → codec present | | | | yes | — (v79>28; char-data fixture-verified `bd9a1134e`/`45116cdcb`) |
| character/data: GMS>28 && <=87 or JMS narrow range | `libs/atlas-packet/character/data.go:148` (×2 lines 148,207) | `(Region()=="GMS" && MajorVersion()>28 && MajorVersion()<=87) \|\| Region()=="JMS"` | TRUE → narrow-range field | | | | yes | — (v79 in 29..87) |
| character/data: any-region >12 inner field | `libs/atlas-packet/character/data.go:286` (×2 lines 286,362; nested in Region=="GMS" block) | `MajorVersion()>12` | TRUE → field present | | | | yes | — (v79>12) |
| character/data: any-region >=87 inner field | `libs/atlas-packet/character/data.go:293` (×2 lines 293,369; nested in Region=="GMS" block) | `MajorVersion()>=87` | FALSE → field absent | | | | yes | — (v79<87) |
| character/data: GMS>12 or JMS | `libs/atlas-packet/character/data.go:386` (×4 lines 386,471,643,664) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79>12) |
| character/data: GMS<28 legacy field (~10 instances) | `libs/atlas-packet/character/data.go:419` (×10 lines 419,432,441,450,459,490,496,502,508,514) | `Region()=="GMS" && MajorVersion()<28` | FALSE → skip pre-28 field | | | | yes | — (v79>28) |
| character/data: GMS region-only inner gate | `libs/atlas-packet/character/data.go:153` (×4 lines 153,212,285,361) | `Region()=="GMS"` | TRUE → GMS block | | | | yes | — (v79 is GMS) |
| character/data: GMS>=84 Evan job guard (MajorAtLeast) | `libs/atlas-packet/character/data.go:269` (×2 lines 269,342) | `IsRegion("GMS") && MajorAtLeast(84) && isEvanJob(...)` | FALSE → no Evan block | | | | yes | — (v79<84; Evan/dual-job blocks are v84+) |

### serverbound

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| character/create: GMS>=73 or JMS fields (intra-legacy discriminator!) | `libs/atlas-packet/character/serverbound/create.go:113` (×2 lines 113,148) | `(Region()=="GMS" && MajorVersion()>=73) \|\| Region()=="JMS"` | **TRUE** → jobIndex present | | | | yes | — (v79>=73 → reads jobIndex; matches v83 body, char-create fixture-verified `220509626`. v72/v61/v48 take the else default jobIndex=1) |
| character/create: GMS>=87 or JMS field (MajorAtLeast) | `libs/atlas-packet/character/serverbound/create.go:116` | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | FALSE → no subJobIndex | | | | yes | — (v79<87) |
| character/create: GMS>28 not JMS field | `libs/atlas-packet/character/serverbound/create.go:130` | `(Region()=="GMS" && MajorVersion()>28) && Region()!="JMS"` | TRUE → gender byte | | | | yes | — (v79>28) |
| character/create: GMS<=28 field | `libs/atlas-packet/character/serverbound/create.go:133` (×2 lines 133,183) | `Region()=="GMS" && MajorVersion()<=28` | FALSE → skip stat bytes | | | | yes | — (v79>28) |
| character/create: GMS<=28 or JMS field | `libs/atlas-packet/character/serverbound/create.go:176` | `(Region()=="GMS" && MajorVersion()<=28) \|\| Region()=="JMS"` | FALSE → gender read on wire | | | | yes | — (v79>28) |
| character/create: GMS not >=87 field (MajorAtLeast) | `libs/atlas-packet/character/serverbound/create.go:157` | `IsRegion("GMS") && !MajorAtLeast(87)` | TRUE → subJobIndex defaults 0 | | | | yes | — (v79<87) |
| character/delete: GMS>82 BPIC-present branch | `libs/atlas-packet/character/serverbound/delete.go:51` (×2 lines 51,64) | `Region()=="GMS" && MajorVersion()>82` | FALSE → no-BPIC branch | | | | yes | — (v79 usesPin=false per delta §e; BPIC/second-password is v83+) |
| character/delete: GMS else (no BPIC) branch | `libs/atlas-packet/character/serverbound/delete.go:53` (×2 lines 53,67; else-if of above) | `Region()=="GMS"` (else-if) | TRUE → GMS no-BPIC branch | | | | yes | — (v79 is GMS, >82 false → this else-if) |
| character/expression (sb): GMS>87 expression fields | `libs/atlas-packet/character/serverbound/expression.go:58` (×2 lines 58,73) | `Region()=="GMS" && MajorVersion()>87` | FALSE → fields absent | | | | yes | — (v79<=87) |
| character/heal_over_time: GMS<=95 or JMS field | `libs/atlas-packet/character/serverbound/heal_over_time.go:81` (×2 lines 81,98) | `(Region()=="GMS" && MajorVersion()<=95) \|\| Region()=="JMS"` | TRUE → option byte present | | | | yes | — (v79<=95; NB the leading updateTime dword is now legacy-gated — see campaign row heal_over_time `<83`) |
| character/move: GMS>28 field | `libs/atlas-packet/character/serverbound/move.go:73` (×2 lines 73,101) | `Region()=="GMS" && MajorVersion()>28` | TRUE → field present | | | | yes | — (v79>28; move fixture-verified `8a83c379a`) |
| character/move: GMS>=84 fields (MajorAtLeast, 6 instances) | `libs/atlas-packet/character/serverbound/move.go:64` (×6 lines 64,69,76,92,97,104) | `IsRegion("GMS") && MajorAtLeast(84)` | FALSE → v83-path fields | | | | yes | — (v79<84; v84 movement fields absent, = v83) |

---

## libs/atlas-packet/chat

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| chat/multi: GMS>=95 updateTime field | `libs/atlas-packet/chat/serverbound/multi.go:54` (×2 lines 54,71) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → no updateTime | | | | yes | — (v79<95; MULTI_CHAT fixture-verified `362aac324`) |
| chat/whisper: GMS>=87 or JMS gate | `libs/atlas-packet/chat/serverbound/whisper.go:28` | `(Region()=="GMS" && MajorVersion()>=87) \|\| Region()=="JMS"` | FALSE → legacy whisper | | | | yes | — (v79<87; WHISPER fixture-verified `362aac324`) |

---

## libs/atlas-packet/field

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| field/admin_result: GMS>=95 branch | `libs/atlas-packet/field/clientbound/admin_result.go:110` (×2 lines 110,201) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → skip | | | | yes | — (v79<95) |
| field/admin_result: GMS>=87 && <95 branch | `libs/atlas-packet/field/clientbound/admin_result.go:129` (×2 lines 129,219) | `Region()=="GMS" && MajorVersion()>=87 && MajorVersion()<95` | FALSE → skip | | | | yes | — (v79<87) |
| field/admin_result: GMS>=84 && <87 branch | `libs/atlas-packet/field/clientbound/admin_result.go:146` (×2 lines 146,235) | `Region()=="GMS" && MajorVersion()>=84 && MajorVersion()<87` | FALSE → skip | | | | yes | — (v79<84) |
| field/admin_result: v83 branch, now `>=83 && <84` (was `<84`) | `libs/atlas-packet/field/clientbound/admin_result.go:163` (Encode; Decode :266) | `Region()=="GMS" && MajorVersion()>=83 && MajorVersion()<84` | FALSE → v79 uses new `<83` branch | | | | yes (FIXED) | Old `<84` fired the v83-shaped branch for v79 (leading string), but v79 @0x52075c has NO leading `sAt(0)` string → wrong layout. Campaign added `>=83` lower bound + a distinct `<83` branch (`cb19e519d`); see campaign row admin_result `<83`. |
| field/affected_area_created: GMS>=95 area type field | `libs/atlas-packet/field/clientbound/affected_area_created.go:92` | `Region()=="GMS" && MajorVersion()>=95` | FALSE → no area-type field | | | | yes | — (v79<95; AffectedAreaCreated fixture-verified `4975246fb`) |
| field/foothold_info: any-region MajorAtLeast(95) gate | `libs/atlas-packet/field/clientbound/foothold_info.go:88` | `MajorAtLeast(95)` _(no Region guard)_ | FALSE → field absent | | | | yes | — (v79<95) |
| field/set_field: GMS>=95 field | `libs/atlas-packet/field/clientbound/set_field.go:52` (×2 lines 52,100) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |
| field/set_field: GMS>28 or JMS field | `libs/atlas-packet/field/clientbound/set_field.go:62` (×2 lines 62,110) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79>28) |
| field/set_field: GMS>=87 or JMS (MajorAtLeast, 4 instances) | `libs/atlas-packet/field/clientbound/set_field.go:47` (×4 lines 47,77,95,125) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | FALSE → field absent | | | | yes | — (v79<87) |
| field/warp_to_map: GMS>=95 field (4 instances) | `libs/atlas-packet/field/clientbound/warp_to_map.go:98` (×4 lines 98,118,144,161) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |
| field/warp_to_map: GMS>28 or JMS field | `libs/atlas-packet/field/clientbound/warp_to_map.go:107` (×2 lines 107,153) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | TRUE → nNotifierCheck present | | | | yes | — (v79>28; the nested revive byte is now legacy-gated — see campaign row warp_to_map revive) |
| field/warp_to_map: GMS>28 field | `libs/atlas-packet/field/clientbound/warp_to_map.go:123` (×2 lines 123,166) | `Region()=="GMS" && MajorVersion()>28` | TRUE → field present | | | | yes | — (v79>28) |
| field/warp_to_map: GMS>=87 or JMS (MajorAtLeast) | `libs/atlas-packet/field/clientbound/warp_to_map.go:93` (×2 lines 93,139) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | FALSE → field absent | | | | yes | — (v79<87) |
| field/witch_tower_score_update: GMS MajorAtLeast(95) | `libs/atlas-packet/field/clientbound/witch_tower_score_update.go:38` | `Region()=="GMS" && MajorAtLeast(95)` | FALSE → field absent | | | | yes | — (v79<95) |
| field/change (sb): chase flag, now `>=79` (was `>=83`) | `libs/atlas-packet/field/serverbound/change.go:74` (Encode; Decode :106) | `Region()=="GMS" && MajorVersion()>=79` | **TRUE** → chase flag present | | | | yes (FIXED) | Old `>=83` dropped the chase byte for v79 → desync. v79 CField::SendTransferFieldRequest @0x51b950 Encode1(dword_B0D450)@0x51ba3e emits it; campaign lowered to `>=79` (`526ef9b08`). v83/84/87/95 unchanged. |
| field/general (sb): GMS>=87 or JMS (MajorAtLeast) | `libs/atlas-packet/field/serverbound/general.go:46` (×2 lines 46,60) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | FALSE → legacy | | | | yes | — (v79<87) |
| field/sue_character: any-region >=95 field | `libs/atlas-packet/field/serverbound/sue_character.go:61` (×2 lines 61,75) | `MajorVersion()>=95` _(no Region guard)_ | FALSE → field absent | | | | yes | — (v79<95) |

---

## libs/atlas-packet/guild

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| guild/operation: GMS>=84 or JMS trailing ints (MajorAtLeast) | `libs/atlas-packet/guild/clientbound/operation.go:769` (×2 lines 769,786) | `(IsRegion("GMS") && MajorAtLeast(84)) \|\| Region()=="JMS"` | FALSE → no trailing ints | | | | yes | — (v79<84; GUILD_OPERATION 38-arm family fixture-verified `64fe23844`) |

---

## libs/atlas-packet/interaction

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| interaction/operation_chat: GMS>=87 or JMS gate | `libs/atlas-packet/interaction/serverbound/operation_chat.go:33` | `(Region()=="GMS" && MajorVersion()>=87) \|\| Region()=="JMS"` | FALSE → legacy | | | | yes | — (v79<87; PLAYER_INTERACTION family fixture-verified `4e2c1fa29`) |

---

## libs/atlas-packet/login

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| login/auth_login_failed: GMS branch | `libs/atlas-packet/login/clientbound/auth_login_failed.go:34` (×2 lines 34,47) | `Region()=="GMS"` | TRUE → GMS branch | | | | yes | — (v79 is GMS) |
| login/auth_permanent_ban: GMS branch | `libs/atlas-packet/login/clientbound/auth_permanent_ban.go:34` (×2 lines 34,56) | `Region()=="GMS"` | TRUE → GMS branch | | | | yes | — (v79 is GMS) |
| login/auth_permanent_ban: non-GMS branch | `libs/atlas-packet/login/clientbound/auth_permanent_ban.go:42` (×2 lines 42,60) | `Region()!="GMS"` | FALSE → not taken | | | | yes | — (v79 is GMS) |
| login/auth_success: GMS>=95 field | `libs/atlas-packet/login/clientbound/auth_success.go:51` (×2 lines 51,113) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |
| login/auth_success: GMS region-only gate | `libs/atlas-packet/login/clientbound/auth_success.go:44` (×4 lines 44,57,106,119) | `Region()=="GMS"` | TRUE → GMS block | | | | yes | — (v79 is GMS) |
| login/auth_success: >12 inner field (nested in GMS block) | `libs/atlas-packet/login/clientbound/auth_success.go:58` (×4 lines 58,63,120,125) | `MajorVersion()>12` | TRUE → field present | | | | yes | — (v79>12) |
| login/auth_success: MajorAtLeast(84) inner field (nested in GMS block) | `libs/atlas-packet/login/clientbound/auth_success.go:81` (×2 lines 81,143) | `MajorAtLeast(84)` | FALSE → field absent | | | | yes | — (v79<84) |
| login/auth_temporary_ban: GMS branch | `libs/atlas-packet/login/clientbound/auth_temporary_ban.go:48` (×2 lines 48,64) | `Region()=="GMS"` | TRUE → GMS branch | | | | yes | — (v79 is GMS) |
| login/server_ip: GMS>12 or JMS field | `libs/atlas-packet/login/clientbound/server_ip.go:74` (×2 lines 74,92) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79>12; SERVER_IP is login-region Δ0, opcode unchanged per delta) |
| login/server_list_entry: GMS>12 or JMS field | `libs/atlas-packet/login/clientbound/server_list_entry.go:80` (×2 lines 80,123) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79>12) |
| login/server_list_entry: GMS region-only gate | `libs/atlas-packet/login/clientbound/server_list_entry.go:56` (×2 lines 56,97) | `Region()=="GMS"` | TRUE → GMS block | | | | yes | — (v79 is GMS) |
| login/server_list_entry: >12 inner field (nested) | `libs/atlas-packet/login/clientbound/server_list_entry.go:57` (×2 lines 57,98) | `MajorVersion()>12` | TRUE → field present | | | | yes | — (v79>12) |
| login/all_character_list_request: GMS>=87 (MajorAtLeast) | `libs/atlas-packet/login/serverbound/all_character_list_request.go:57` (×2 lines 57,72) | `IsRegion("GMS") && MajorAtLeast(87)` | FALSE → legacy | | | | yes | — (v79<87; VIEW_ALL_CHAR sb sender verified `279504ee3`) |
| login/character_select: GMS>12 field | `libs/atlas-packet/login/serverbound/character_select.go:47` (×2 lines 47,59) | `Region()=="GMS" && MajorVersion()>12` | TRUE → field present | | | | yes | — (v79>12) |
| login/character_select_register_pic: GMS region-only | `libs/atlas-packet/login/serverbound/character_select_register_pic.go:58` (×2 lines 58,72) | `Region()=="GMS"` | TRUE → GMS shape (path unused) | | | | yes | — (v79 is GMS; but usesPin=false per delta §e so v79 uses the non-PIC select path — this register-PIC handler is not exercised for v79) |
| login/character_select_with_pic: GMS region-only | `libs/atlas-packet/login/serverbound/character_select_with_pic.go:53` (×2 lines 53,67) | `Region()=="GMS"` | TRUE → GMS shape (path unused) | | | | yes | — (v79 is GMS; usesPin=false → the with-PIC path is not exercised for v79) |
| login/request: GMS region-only | `libs/atlas-packet/login/serverbound/request.go:78` (×2 lines 78,95) | `Region()=="GMS"` | TRUE → GMS shape | | | | yes | — (v79 is GMS) |
| login/server_status_request: GMS region-only | `libs/atlas-packet/login/serverbound/server_status_request.go:36` (×2 lines 36,48) | `Region()=="GMS"` | TRUE → GMS shape | | | | yes | — (v79 is GMS) |
| login/world_character_list_request: GMS>28 field | `libs/atlas-packet/login/serverbound/world_character_list_request.go:53` (×2 lines 53,70) | `Region()=="GMS" && MajorVersion()>28` | TRUE → field present | | | | yes | — (v79>28) |
| login/world_character_list_request: GMS>12 field | `libs/atlas-packet/login/serverbound/world_character_list_request.go:58` (×2 lines 58,76) | `Region()=="GMS" && MajorVersion()>12` | TRUE → field present | | | | yes | — (v79>12) |

---

## libs/atlas-packet/model

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| model/asset: GMS>12 or JMS (6 instances) | `libs/atlas-packet/model/asset.go:195` (×6 lines 195,208,246,260,377,415) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79>12) |
| model/asset: GMS>28 or JMS (4 instances) | `libs/atlas-packet/model/asset.go:213` (×4 lines 213,264,347,419) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79>28) |
| model/asset: GMS>=84 (MajorAtLeast) | `libs/atlas-packet/model/asset.go:217` (×2 lines 217,428) | `IsRegion("GMS") && MajorAtLeast(84)` | FALSE → field absent | | | | yes | — (v79<84) |
| model/attack_info: GMS>=84 DR-block fields (6 instances) | `libs/atlas-packet/model/attack_info.go:83` (×6 lines 83,88,96,192,200,210) | `Region()=="GMS" && MajorVersion()>=84` | FALSE → no DR block | | | | yes | — (v79<84; attacks fixture-verified `8b8174034`) |
| model/attack_info: GMS>=95 fields (~14 instances) | `libs/atlas-packet/model/attack_info.go:93` (×14 instances) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → fields absent | | | | yes | — (v79<95) |
| model/avatar: GMS<=28 fields (5 instances) | `libs/atlas-packet/model/avatar.go:50` (×5 lines 50,62,70,104,116) | `Region()=="GMS" && MajorVersion()<=28` | FALSE → skip pre-29 fields | | | | yes | — (v79>28) |
| model/avatar: GMS>28 or JMS field | `libs/atlas-packet/model/avatar.go:78` (×2 lines 78,141) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79>28) |
| model/character_list_entry: GMS<=28 fields | `libs/atlas-packet/model/character_list_entry.go:59` (×2 lines 59,86) | `Region()=="GMS" && MajorVersion()<=28` | FALSE → skip pre-29 fields | | | | yes | — (v79>28) |
| model/character_statistics: GMS>=95 field | `libs/atlas-packet/model/character_statistics.go:113` (×2 lines 113,189) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |
| model/character_statistics: GMS>28 or JMS (4 instances) | `libs/atlas-packet/model/character_statistics.go:98` (×4 lines 98,135,175,211) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79>28) |
| model/character_statistics: GMS region-only inner gate | `libs/atlas-packet/model/character_statistics.go:142` (×2 lines 142,218) | `Region()=="GMS"` | TRUE → GMS block | | | | yes | — (v79 is GMS) |
| model/character_statistics: >12 inner field (nested) | `libs/atlas-packet/model/character_statistics.go:143` (×2 lines 143,219) | `MajorVersion()>12` | TRUE → field present | | | | yes | — (v79>12) |
| model/character_statistics: >=87 inner field (nested) | `libs/atlas-packet/model/character_statistics.go:150` (×2 lines 150,226) | `MajorVersion()>=87` | FALSE → field absent | | | | yes | — (v79<87) |
| model/character_temporary_stat: GMS>=95 stat mask | `libs/atlas-packet/model/character_temporary_stat.go:174` (×2 lines 174,723) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → legacy mask | | | | yes | — (v79<95) |
| model/character_temporary_stat: post87 GMS or JMS (MajorAtLeast) | `libs/atlas-packet/model/character_temporary_stat.go:176` | `(Region()=="GMS" && MajorAtLeast(87)) \|\| jms` | FALSE → legacy | | | | yes | — (v79<87) |
| model/character_temporary_stat: GMS>=87 stat enable (MajorAtLeast) | `libs/atlas-packet/model/character_temporary_stat.go:105` | `IsRegion("GMS") && MajorAtLeast(87)` | FALSE → legacy | | | | yes | — (v79<87) |
| model/damage_info: per-mob CRC, now `>=79` (was `>=83`) | `libs/atlas-packet/model/damage_info.go:48` (Decode; Encode :79) | `Region()=="GMS" && MajorVersion()>=79` | **TRUE** → CRC present | | | | yes (FIXED) | Old `>=83` dropped the per-mob CRC for v79 → 4-byte/target desync. v79 IDA: TryDoingMeleeAttack@0x8c2c57 / BodyAttack@0x8b77d3 / MagicAttack@0x8af1c4 all Encode4 the mob CRC. Campaign lowered to `>=79` (`8b8174034`); no in-range version changes (all were already >=83). |
| model/damage_taken_info: GMS>=95 taken-damage field | `libs/atlas-packet/model/damage_taken_info.go:103` (×2 lines 103,136) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |
| model/monster: GMS>12 or JMS (4 instances) | `libs/atlas-packet/model/monster.go:497` (×4 lines 497,509,527,539) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79>12) |
| model/monster: GMS>=87 or JMS (MajorAtLeast) | `libs/atlas-packet/model/monster.go:512` (×2 lines 512,542) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | FALSE → legacy | | | | yes | — (v79<87) |
| model/movement: not-GMS or >=88 boundary (MajorAtLeast) | `libs/atlas-packet/model/movement.go:131` (×2 lines 131,222) | `!IsRegion("GMS") \|\| MajorAtLeast(88)` | FALSE → GMS-<88 path | | | | yes | — (v79 is GMS & <88 → same movement branch as v83; delta §c: OnMove decode = v83) |
| model/skill_prepare_info: GMS>=95 or JMS | `libs/atlas-packet/model/skill_prepare_info.go:22` | `(Region()=="GMS" && MajorVersion()>=95) \|\| Region()=="JMS"` | FALSE → legacy | | | | yes | — (v79<95) |

---

## libs/atlas-packet/monster

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| monster/catch_monster: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/monster/clientbound/catch_monster.go` (`v95CatchLayout`) | `IsRegion("GMS") && MajorAtLeast(95)` | FALSE → no success byte | | | | yes | — (v79<95; per-mob uniqueId prefix now legacy-gated — see campaign row catch_monster) |
| monster/monster_special_effect_by_skill: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/monster/clientbound/monster_special_effect_by_skill.go` (`v95` layout) | `IsRegion("GMS") && MajorAtLeast(95)` | FALSE → legacy | | | | yes | — (v79<95; uniqueId prefix now legacy-gated — see campaign row) |
| monster/clientbound/movement: GMS>=87 or JMS (MajorAtLeast, 4 instances) | `libs/atlas-packet/monster/clientbound/movement.go:56` (×4 lines 56,63,77,84) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | FALSE → legacy | | | | yes | — (v79<87; MonsterMovement fixture-verified `87f12e20f`) |
| monster/clientbound/spawn: GMS>12 or JMS | `libs/atlas-packet/monster/clientbound/spawn.go:47` (×2 lines 47,64) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | TRUE → field present | | | | yes | — (v79>12) |
| monster/serverbound/movement: GMS>=84 or JMS (MajorAtLeast) | `libs/atlas-packet/monster/serverbound/movement.go:71` (×2 lines 71,106) | `(IsRegion("GMS") && MajorAtLeast(84)) \|\| Region()=="JMS"` | FALSE → legacy | | | | yes | — (v79<84; monster sb movement verified `747c669c5`) |
| monster/serverbound/movement: GMS>=87 or JMS (MajorAtLeast, 4 instances) | `libs/atlas-packet/monster/serverbound/movement.go:80` (×4 lines 80,86,115,121) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | FALSE → legacy | | | | yes | — (v79<87) |

---

## libs/atlas-packet/npc

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| npc/conversation: GMS>=87 or JMS (MajorAtLeast) | `libs/atlas-packet/npc/clientbound/conversation.go:354` | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | FALSE → legacy | | | | yes | — (v79<87; NpcAskMemberShop fixture-verified `8d0e78757`; ASK_MENU/AskMemberShopAvatar legacy count byte now added — see campaign rows) |
| npc/shop_list: GMS>=87 shop field | `libs/atlas-packet/npc/clientbound/shop_list.go:54` (×2 lines 54,83) | `Region()=="GMS" && MajorVersion()>=87` | FALSE → field absent | | | | yes | — (v79<87; NpcShop verified `905834f7d`) |
| npc/shop_list: GMS>=95 shop field | `libs/atlas-packet/npc/clientbound/shop_list.go:57` (×2 lines 57,86) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |
| npc/shop_buy: GMS region-only | `libs/atlas-packet/npc/serverbound/shop_buy.go:41` (×2 lines 41,54) | `Region()=="GMS"` | TRUE → GMS shape | | | | yes | — (v79 is GMS) |

---

## libs/atlas-packet/party

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| party/invite: GMS>=87 or JMS (MajorAtLeast) | `libs/atlas-packet/party/clientbound/invite.go:45` (×2 lines 45,63) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | FALSE → legacy | | | | yes | — (v79<87; PARTY_OPERATION 22-arm family fixture-verified `9a9ceaf61`) |
| party/town_portal: GMS>=95 gate | `libs/atlas-packet/party/clientbound/town_portal.go:68` | `Region()=="GMS" && MajorVersion()>=95` | FALSE → legacy | | | | yes | — (v79<95) |
| party/member_data: GMS>=95 field | `libs/atlas-packet/party/member_data.go:87` (×2 lines 87,125) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95; PartyMemberHP fixture-verified `9215bfbf9`) |

---

## libs/atlas-packet/pet

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| pet/chat: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/pet/serverbound/chat.go:60` (×2 lines 60,77) | `IsRegion("GMS") && MajorAtLeast(95)` | FALSE → legacy | | | | yes | — (v79<95; pet family fixture-verified `4d52195a2`) |
| pet/drop_pick_up: GMS>=87 (MajorAtLeast) | `libs/atlas-packet/pet/serverbound/drop_pick_up.go:71` (×2 lines 71,97) | `IsRegion("GMS") && MajorAtLeast(87)` | FALSE → legacy | | | | yes | — (v79<87; the trailing crc is now legacy-gated separately — see campaign row pet/drop_pick_up crc) |

---

## libs/atlas-packet/stat

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| stat/changed: GMS>=95 stat-change field | `libs/atlas-packet/stat/clientbound/changed.go:51` (×2 lines 51,106) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95; stat/Changed fixture-verified `9215bfbf9`) |

---

## libs/atlas-packet/summon

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| summon/clientbound/attack: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/summon/clientbound/attack.go:83` (trailing flag byte) | `IsRegion("GMS") && MajorAtLeast(95)` | FALSE → no flag byte | | | | yes | — (v79<95; summon fixture-verified `de77edad4`; leading charLevel byte now gated `>=83` — see campaign row) |
| summon/clientbound/spawn: JMS v185 gate (MajorAtLeast 185, checked first) | `libs/atlas-packet/summon/clientbound/spawn.go:135` | `MajorAtLeast(185)` _(JMS = v185; no Region guard — true only for JMS)_ | FALSE → not taken | | | | yes | — (v79<185) |
| summon/clientbound/spawn: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/summon/clientbound/spawn.go:137` | `IsRegion("GMS") && MajorAtLeast(95)` | FALSE → no avatar look | | | | yes | — (v79<95; SLV byte now gated `>=83` — see campaign row spawnHasSkillLevel) |
| summon/serverbound/attack: JMS v185 gate (MajorAtLeast 185, checked first) | `libs/atlas-packet/summon/serverbound/attack.go:119` | `MajorAtLeast(185)` _(JMS = v185; no Region guard — true only for JMS)_ | FALSE → not taken | | | | yes | — (v79<185) |
| summon/serverbound/attack: GMS>=84 (MajorAtLeast) | `libs/atlas-packet/summon/serverbound/attack.go:121` | `IsRegion("GMS") && MajorAtLeast(84)` | FALSE → legacy | | | | yes | — (v79<84) |
| summon/serverbound/attack: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/summon/serverbound/attack.go:136` | `IsRegion("GMS") && MajorAtLeast(95)` | FALSE → legacy | | | | yes | — (v79<95) |

---

## libs/atlas-packet/ui

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| ui/lock: GMS>=90 screen-lock field | `libs/atlas-packet/ui/clientbound/lock.go:33` (×2 lines 33,44) | `Region()=="GMS" && MajorVersion()>=90` | FALSE → field absent | | | | yes | — (v79<90) |

---

## libs/atlas-seeder

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| seeder/catalog: zero-version validation guard | `libs/atlas-seeder/catalog.go:54` | `MajorVersion()==0 \|\| MinorVersion()==0` _(not a legacy discriminator; seeder-internal validity check)_ | FALSE → validation passes | | | | yes | — (v79.1 has non-zero major=79 & minor=1; guard only rejects unset versions) |

---

## services/atlas-account

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| account/processor: GMS>=87 behaviour gate (MajorAtLeast) | `services/atlas-account/atlas.com/account/account/processor.go:126` | `IsRegion("GMS") && MajorAtLeast(87)` | FALSE → legacy behavior | | | | yes | — (v79<87) |

---

## services/atlas-channel

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| channel/main: GMS<=28 legacy mode init | `services/atlas-channel/atlas.com/channel/main.go:391` | `Region()=="GMS" && MajorVersion()<=28` | FALSE → ShortReadWriter (2-byte opcode framing) | | | | yes (OQ-6 IDA-confirmed) | — v79 uses 2-byte opcodes: `CClientSocket::ProcessPacket @0x48e209` scrutinee is `CInPacket::Decode2 @0x421b87` (returns uint16). `ByteReadWriter` is for `<=v28` only; v79>28 → Short. |
| channel/session: GMS<=12 session mode | `services/atlas-channel/atlas.com/channel/session/model.go:40` | `Region()=="GMS" && MajorVersion()<=12` | FALSE → standard AES-OFB (shuffled IV) | | | | yes (OQ-6 confirmed) | — v79>12 → standard maple AES-OFB else-branch. The zero-fill-IV path (`FillIvZeroGenerator`) is a pre-v13 behavior; v79's encrypted socket + standard `COutPacket(1)` login handshake (delta §f `SendCheckPasswordPacket @0x5cbf50`) confirm modern crypto. (Cipher primitives are not separately symbol-named in the v79 IDB; basis = framing + handshake.) |
| channel/character_cash_item_use: GMS>=95 item-use fields (~30 instances) | `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:32` (×30 instances, lines 32–494) | `Region()=="GMS" && MajorVersion()>=95` | FALSE → legacy | | | | yes | — (v79<95) |
| channel/socket/model/damage_taken_info: GMS>=95 field | `services/atlas-channel/atlas.com/channel/socket/model/damage_taken_info.go:66` | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |
| channel/socket/writer/character_attack_common: GMS>=95 field | `services/atlas-channel/atlas.com/channel/socket/writer/character_attack_common.go:180` | `Region()=="GMS" && MajorVersion()>=95` | FALSE → field absent | | | | yes | — (v79<95) |

---

## services/atlas-character

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| character/processor: GMS MajorAtMost(94) gate (equivalent to <=94, i.e., <95) | `services/atlas-character/atlas.com/character/character/processor.go:56` | `IsRegion("GMS") && MajorAtMost(94)` | TRUE → pre-95 branch | | | | yes | — (v79<=94) |

---

## services/atlas-login

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| login/main: GMS<=28 legacy mode init | `services/atlas-login/atlas.com/login/main.go:277` | `Region()=="GMS" && MajorVersion()<=28` | FALSE → ShortReadWriter (2-byte opcode framing) | | | | yes (OQ-6 IDA-confirmed) | — same basis as channel/main: v79 `ProcessPacket @0x48e209` reads opcode via `Decode2 @0x421b87` (uint16); `ByteReadWriter` is `<=v28`-only, v79>28 → Short. |
| login/session: GMS<=12 session mode | `services/atlas-login/atlas.com/login/session/model.go:35` | `Region()=="GMS" && MajorVersion()<=12` | FALSE → standard AES-OFB (shuffled IV) | | | | yes (OQ-6 confirmed) | — same basis as channel/session: v79>12 → standard AES-OFB; zero-fill-IV path is pre-v13 only; confirmed by v79 encrypted socket + standard `COutPacket(1)` handshake (`SendCheckPasswordPacket @0x5cbf50`). |

---

## services/atlas-tenants (test assertions — not production gates)

> These rows are TEST FILE assertions that happen to use `MajorVersion()` comparisons. They are
> NOT version-gated behavior branches; they are test correctness checks. Included for completeness
> per the `!=83` / `!=90` predicate forms in the brief's pre-computed facts list.

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| tenants builder/processor test !=83 assertion | `services/atlas-tenants/atlas.com/tenants/tenant/builder_test.go:41` (×3 lines 41,126; processor_test.go:122) | `MajorVersion()!=83` _(test assertion only; t.Fatalf if false)_ | TRUE → assertion holds | | | | yes (test-only) | — (v79≠83; not a runtime gate — fixed-value 83 test setup) |
| tenants builder/processor test !=90 assertion | `services/atlas-tenants/atlas.com/tenants/tenant/builder_test.go:148` (×2 lines 148; processor_test.go:230) | `MajorVersion()!=90` _(test assertion only; t.Fatalf if false)_ | TRUE → assertion holds | | | | yes (test-only) | — (v79≠90; not a runtime gate) |

---

## Campaign-added legacy (`< 83`) gates — Stage E (v79)

> These gates did **not** exist on `main`. The task-113 Stage E v79 campaign added
> each one (pattern `t.Region()=="GMS" && t.MajorVersion() < 83`, or the
> `MajorAtLeast(83)`/`!(…<83)` equivalents) to carry a v79 wire divergence that the
> pre-existing `>=NN` gates could not express. All are IDA-verified against
> `GMS_v79_1_DEVM.exe` (port 13340) and byte-fixtured in the cited commit.
> **No v83/v84/v87/v95/JMS evaluation changes** (each new predicate is FALSE for
> those versions, or gated at `>=83`). v72/v61/v48 columns intentionally blank.

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| character/add_entry: legacy [code][stat][avatar] with no list-entry trailer | `libs/atlas-packet/character/clientbound/add_entry.go:16` (`legacyAddEntry`) | `Region()=="GMS" && MajorVersion()>28 && MajorVersion()<83` | **TRUE** → legacy codec (stat+avatar, no trailer) | | | | yes | v79 add handler @0x5ceb55 (Decode1 code → GW_CharacterStat::Decode → AvatarLook::Decode, family/rank zeroed locally). Commit `bd9a1134e`/`1fb29a2bb`. |
| character/effect_skill_use: caster-level byte (both self + foreign) | `libs/atlas-packet/character/clientbound/effect_skill_use.go:20` (`effectSkillUseIncludesCharacterLevel`) | `!(Region()=="GMS" && MajorVersion()<83)` | FALSE → charLevel byte OMITTED | | | | yes | v79 CUser::OnEffect @0x89112c case 1 reads skillId + one (skillLevel) byte only; v83 @0x9377d9 reads an extra leading charLevel byte. Commit `e047a34d4`. |
| character/keymap: FUNCKEY entry count (89 vs 90) | `libs/atlas-packet/character/clientbound/keymap.go:16` (`keyMapEntryCount`) | `Region()=="GMS" && MajorVersion()<83 → 89 else 90` | TRUE → 89 entries | | | | yes | v79 CFuncKeyMappedMan::OnInit @0x569e69 `v5=89` (memcpy 0x1BD=445=89×5). Commit `9364e3c45`. |
| character/skill_change: per-skill 8-byte expiration field | `libs/atlas-packet/character/clientbound/skill_change.go:16` (`skillChangeHasExpiration`) | `Region()!="GMS" \|\| MajorVersion()>=83` | FALSE → no expiration field | | | | yes | v79 @0x968f0e reads 3 Decode4 per skill (no DecodeBuffer(8)); v83 @0xa1e48c adds the 8-byte buffer. Commit `820eae5a0`. |
| character/list: hasPic / m_bLoginOpt byte | `libs/atlas-packet/character/clientbound/list.go:60` (Encode; Decode :101) | `!(Region()=="GMS" && MajorVersion()<83)` | FALSE → hasPic byte OMITTED | | | | yes | v79 char-list decoder sub_5CE522 @0x5CE522 reads slot count (Decode4) with no login-option byte. Commit `bd9a1134e`. |
| character/status_message: trailing rainbowWeekEventEXP int (IncEXP arm) | `libs/atlas-packet/character/clientbound/status_message.go:530` (Encode; Decode :567) | `!(Region()=="GMS" && MajorVersion()<83)` | FALSE → 7th exp int OMITTED | | | | yes | v79 OnMessage IncEXP arm sub_96BD0D @0x96bd0d reads 6 exp ints; v83 @0xa21ac5 reads a 7th. Commit `f1e3a5b56`. |
| character/spawn: leading level byte | `libs/atlas-packet/character/clientbound/spawn.go:70` (`legacy`, Encode; Decode :189) | `Region()=="GMS" && MajorVersion()<83` (write when NOT legacy) | TRUE(legacy) → level byte OMITTED | | | | yes | v79 CUserRemote::Init sub_8D589E reads name first (@0x8d58c9), no leading Decode1 level; v83 @0x97f589 reads level. Commit `0225cd68e`. |
| character/spawn: trailing team (carnival) byte | `libs/atlas-packet/character/clientbound/spawn.go:165` (Encode; Decode :266) | `Region()!="JMS" && !(Region()=="GMS" && MajorVersion()<83)` | FALSE → team byte OMITTED | | | | yes | v79 base CField::DecodeFieldSpecificData @0x513a15 forwards only the CUser (never the packet) → no team byte. Commit `0225cd68e`. |
| character/heal_over_time (sb): leading updateTime dword | `libs/atlas-packet/character/serverbound/heal_over_time.go:84` (Encode; Decode :105) | `!(Region()=="GMS" && MajorVersion()<83)` | FALSE → no leading updateTime | | | | yes | v79 CWvsContext::SendStatChangeRequest @0x96944a has no get_update_time — only Encode4(val)+Encode2(hp)+Encode2(mp)+Encode1(option). Commit `45116cdcb`. |
| drop/pick_up (sb): trailing client-CRC uint32 | `libs/atlas-packet/drop/serverbound/pick_up.go:17` (`pickUpHasCRC`) | `MajorAtLeast(83)` | FALSE → no crc | | | | yes | v79 CWvsContext::SendDropPickUpRequest @0x954e9d sends fieldKey+updateTime+x+y+dropId only; v83 @0xa09118 / v95 @0x9d5d50 Encode4 crc. Commit `e34f14f27`. |
| field/warp_to_map: revive byte (nested in `>28` block) | `libs/atlas-packet/field/clientbound/warp_to_map.go:112` (Encode; Decode :160) | `(Region()=="GMS" && MajorVersion()>=83) \|\| Region()=="JMS"` | FALSE → no revive byte | | | | yes | v79 CStage::OnSetField else-branch @0x6f07d9 reads mapId (Decode4 @0x6f0997) right after nNotifierCheck — no revive. Commit `526ef9b08`. |
| login/after_login (sb): accountId int between opt2 and pin | `libs/atlas-packet/login/serverbound/after_login.go:56` (Encode; Decode :74) | `Region()=="GMS" && MajorVersion()<83` | **TRUE** → accountId int present | | | | yes | v79 CLogin::OnSetAccountResult @0x5d0800 & OnCheckPinCodeResult @0x5d0aaf build COutPacket(9)=Enc1(pinMode)+Enc1(opt2)+Enc4(accountId)+EncStr(pin); v83 @0x5fc731 omits the int. Commit `491d8182b`. |
| monster/catch_monster + inc_mob_charge_count + monster_special_effect_by_skill: leading per-mob uniqueId prefix | `libs/atlas-packet/monster/clientbound/catch_monster.go:65` (`legacyMobPoolPrefix`; shared by all three) | `IsRegion("GMS") && !MajorAtLeast(83)` | TRUE → uniqueId prefix present | | | | yes | v79 CMobPool::OnMobPacket @0x646d46 consumes Decode4 uniqueId (@0x646d50 → GetMob) before dispatching these per-mob packets. Commits `87f12e20f`,`ebf1355c7`. **Follow-up flagged in-code:** sibling per-mob packets (MonsterHealth/Movement) carry this prefix unconditionally, so v83+ likely need it too — left frozen per campaign scope. |
| npc/conversation AskMenu: trailing avatar-style count byte | `libs/atlas-packet/npc/clientbound/conversation.go:193` | `IsRegion("GMS") && !MajorAtLeast(83)` | TRUE → count byte (=0) written | | | | yes | v79 CScriptMan::OnAskMenu @0x6c8863 reads DecodeStr + Decode1(count) + Decode4×count (avatar ids); v83 @0x746fad reads a plain string. Atlas uses plain #L# menus → count=0. Commit `8d0e78757`. |
| npc/conversation AskMemberShopAvatar: legacy SN-count byte vs v83+ int32 style list | `libs/atlas-packet/npc/clientbound/conversation.go:288,294` | `Region()=="GMS" && MajorVersion()<83` (count=0) / `(Region()=="GMS" && MajorVersion()>=83)\|\|JMS` (candidate list) | TRUE(legacy) → count byte (=0), no candidate list | | | | yes | v79 CScriptMan::OnAskMembershopAvatar @0x6c8bc8 reads Decode1(count) + count×(DecodeBuffer(8) SN + Decode1) — incompatible with the v83+ int32 list; Atlas has no SN data → count=0. Commit `8d0e78757`. |
| pet/drop_pick_up (sb): trailing crc uint32 | `libs/atlas-packet/pet/serverbound/drop_pick_up.go:66` (Encode; Decode :99) | `(IsRegion("GMS") && MajorAtLeast(83)) \|\| IsRegion("JMS")` | FALSE → no crc | | | | yes | v79 CPet::SendDropPickUpRequest sub_6923af Encode4(dropId)@0x692451 then 3 bools, no crc; v83 @0x705c7c adds Encode4(crc)@0x705d29. Commit `7a65ea90c`. |
| summon/clientbound/attack: leading char-level byte | `libs/atlas-packet/summon/clientbound/attack.go:74` (Encode; Decode :102) | `MajorAtLeast(83)` | FALSE → no charLevel byte | | | | yes | v79 CSummonedPool::OnAttack sub_71CFE9 reads action byte first (@0x71d06f) then count (@0x71d08b) — no leading charLevel; v83+ read charLevel first. Commit `8a660cbc9`. |
| summon/clientbound/spawn: SLV byte after charLevel | `libs/atlas-packet/summon/clientbound/spawn.go:149` (`spawnHasSkillLevel`) | `MajorAtLeast(83)` | FALSE → no SLV byte | | | | yes | v79 CSummonedPool::OnCreated sub_89268A @0x89268a reads one (charLevel) byte before the x/y Init blob; v83+ read charLevel + SLV (two bytes). Commit `8a660cbc9`. |

> **services/atlas-channel constructor call-sites** (`socket/writer/catch_monster.go`,
> `inc_mob_charge_count.go`, `monster_special_effect_by_skill.go`) were updated to
> thread the new `uniqueId` argument into the codec constructors. These are not
> version gates (the gating lives in `legacyMobPoolPrefix` above); listed for
> completeness.

---

## Completeness check

All distinct predicate forms from the brief's self-review list are represented:

| Predicate form | Representative row(s) |
|---|---|
| `>=95` | character/attack, cash/shop_inventory, stat/changed, etc. |
| `>28` | character/data (dominant gate), character/move sb, login/world_char_list, etc. |
| `>12` | cash/query_result, login/character_select, character/data, etc. |
| `<=28` | character/list cb, character/create sb, avatar, character_list_entry, etc. |
| `<28` | character/data (~10 instances) |
| `<=12` | cash/shop_open, channel/session, login/session |
| `==0` | seeder/catalog (zero-version validity guard) |
| `>=73` | character/create serverbound (intra-legacy discriminator) |
| `>=87` | buddy/invite, chat/whisper, cash/shop_operation_buy, npc/shop_list, etc. |
| `>87` | character/expression cb, character/item_upgrade, character/spawn, character/view_all, etc. |
| `>82` | character/delete serverbound |
| `>=84` | model/attack_info (DR-block), guild/operation, summon/sb/attack, etc. |
| `>=83` | field/change sb, model/damage_info |
| `>=90` | ui/lock |
| `!=83` | tenants test assertions |
| `!=90` | tenants test assertions |
| `<95` | character/spawn cb |
| `<87` | field/admin_result (as upper bound in `>=87 && <95`, `>=84 && <87` ranges) |
| `<84` | field/admin_result |
| `<=87` | character/info cb, character/spawn cb, character/heal_over_time sb |
| `<=95` | character/heal_over_time sb |
| `<=94` (MajorAtMost) | character/processor (services) |
| `>=88` (MajorAtLeast) | model/movement |
| `185` (MajorAtLeast, JMS) | summon/clientbound/spawn, summon/serverbound/attack |
| Region-only (`=="GMS"`) | buddy/error, cash/shop_open, character/delete sb, login auth packets, npc/shop_buy, etc. |
| Region-only (`!="GMS"`) | buddy/invite (outer form), login/auth_permanent_ban |
| No Region guard (numeric only) | character/list `>87`, character/spawn `>87`/`<=87`, field/sue_character `>=95`, field/foothold_info `MajorAtLeast(95)`, seeder `==0`, summon `MajorAtLeast(185)` |

**Total semantic rows: ~152** (grouped from ~600+ raw grep hits across ~80 files)

> **Note on completeness:** The original staged grep (`MajorVersion()` numeric comparisons +
> `Region()=="GMS"`) was incomplete. An additional grep for `MajorAtLeast()`, `MajorAtMost()`,
> `IsRegion("GMS")`, and `Region()!="GMS"` was required to capture the full set. All rows in
> this table were verified against actual grep output.
