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
| buddy/invite: hasJobLevel gate (job field present if non-GMS or GMS>=87) | `libs/atlas-packet/buddy/clientbound/invite.go:51` (×2 lines 51,77) | `Region()!="GMS" \|\| MajorVersion()>=87` | | | | | | |
| buddy/error: GMS-only codec branch (8 instances) | `libs/atlas-packet/buddy/clientbound/error.go:139` (×8 lines 139–244) | `IsRegion("GMS")` | | | | | | |

---

## libs/atlas-packet/cash

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| cash/query_result: GMS+>12 fields | `libs/atlas-packet/cash/clientbound/query_result.go:42` (×2 lines 42,54) | `Region()=="GMS" && MajorVersion()>12` | | | | | | |
| cash/shop_inventory: GMS>=95 or JMS inventory format | `libs/atlas-packet/cash/clientbound/shop_inventory.go:133` | `(Region()=="GMS" && MajorVersion()>=95) \|\| Region()=="JMS"` | | | | | | |
| cash/shop_open: very-old version (<=12) branch | `libs/atlas-packet/cash/clientbound/shop_open.go:45` (×2 lines 45,120) | `MajorVersion()<=12` | | | | | | |
| cash/shop_open: GMS region-only fields (no version guard) | `libs/atlas-packet/cash/clientbound/shop_open.go:36` (×6 lines 36,44,97,111,119,180) | `Region()=="GMS"` | | | | | | |
| cash/shop_open: GMS+>12 or JMS fields (~10 instances) | `libs/atlas-packet/cash/clientbound/shop_open.go:52` (×10 lines 52,60,66,85,94,127,135,141,162,177) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | | | | | | |
| cash/shop_open: GMS+>12 only (2 instances) | `libs/atlas-packet/cash/clientbound/shop_open.go:90` (×2 lines 90,170) | `Region()=="GMS" && MajorVersion()>12` | | | | | | |
| cash/serverbound/item_use: GMS>=95 decode paths | `libs/atlas-packet/cash/serverbound/item_use.go:38` (×2 lines 38,50) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| cash/shop_operation_buy: GMS>=87 field | `libs/atlas-packet/cash/serverbound/shop_operation_buy.go:58` (×2 lines 58,88) | `Region()=="GMS" && MajorVersion()>=87` | | | | | | |
| cash/shop_operation_buy_couple: GMS>=95 field | `libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:57` (×2 lines 57,90) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| cash/shop_operation_buy_friendship: GMS>=95 field | `libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:57` (×2 lines 57,90) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| cash/shop_operation_gift: GMS>=87 field | `libs/atlas-packet/cash/serverbound/shop_operation_gift.go:66` (×2 lines 66,99) | `Region()=="GMS" && MajorVersion()>=87` | | | | | | |
| cash/shop_operation_gift: GMS>=95 field | `libs/atlas-packet/cash/serverbound/shop_operation_gift.go:60` (×2 lines 60,93) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| cash/shop_operation_rebate_locker_item: GMS>=95 field | `libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go:53` (×2 lines 53,82) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |

---

## libs/atlas-packet/character

### clientbound

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| character/attack: GMS>=95 attack fields | `libs/atlas-packet/character/clientbound/attack.go:107` (×2 lines 107,165) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| character/damage: GMS>=95 damage fields | `libs/atlas-packet/character/clientbound/damage.go:55` (×2 lines 55,78) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| character/expression (cb): GMS>87 expression fields | `libs/atlas-packet/character/clientbound/expression.go:62` (×2 lines 62,80) | `Region()=="GMS" && MajorVersion()>87` | | | | | | |
| character/info: GMS<=87 or JMS info field | `libs/atlas-packet/character/clientbound/info.go:129` (×2 lines 129,203) | `(Region()=="GMS" && MajorVersion()<=87) \|\| Region()=="JMS"` | | | | | | |
| character/info: GMS>=87 or JMS info field (MajorAtLeast) | `libs/atlas-packet/character/clientbound/info.go:139` (×2 lines 139,213) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | | | | | | |
| character/item_upgrade: GMS>87 upgrade fields | `libs/atlas-packet/character/clientbound/item_upgrade.go:91` (×2 lines 91,114) | `Region()=="GMS" && MajorVersion()>87` | | | | | | |
| character/item_upgrade: GMS>87 or JMS upgrade fields | `libs/atlas-packet/character/clientbound/item_upgrade.go:98` (×2 lines 98,119) | `(Region()=="GMS" && MajorVersion()>87) \|\| Region()=="JMS"` | | | | | | |
| character/list: GMS<=28 legacy field | `libs/atlas-packet/character/clientbound/list.go:56` (×2 lines 56,91) | `Region()=="GMS" && MajorVersion()<=28` | | | | | | |
| character/list: GMS region-only inner gate | `libs/atlas-packet/character/clientbound/list.go:61` (×2 lines 61,96) | `Region()=="GMS"` | | | | | | |
| character/list: any-region >87 field | `libs/atlas-packet/character/clientbound/list.go:63` (×2 lines 63,98) | `MajorVersion()>87` | | | | | | |
| character/spawn: GMS>87 or JMS spawn field | `libs/atlas-packet/character/clientbound/spawn.go:79` (×2 lines 79,193) | `(Region()=="GMS" && MajorVersion()>87) \|\| Region()=="JMS"` | | | | | | |
| character/spawn: GMS<95 inner block | `libs/atlas-packet/character/clientbound/spawn.go:135` (×2 lines 135,233) | `Region()=="GMS" && MajorVersion()<95` | | | | | | |
| character/spawn: GMS region-only inner gate | `libs/atlas-packet/character/clientbound/spawn.go:141` (×2 lines 141,239) | `Region()=="GMS"` | | | | | | |
| character/spawn: any-region <=87 field | `libs/atlas-packet/character/clientbound/spawn.go:142` (×2 lines 142,240) | `MajorVersion()<=87` | | | | | | |
| character/spawn: any-region >87 field | `libs/atlas-packet/character/clientbound/spawn.go:145` (×2 lines 145,243) | `MajorVersion()>87` | | | | | | |
| character/spawn: GMS>=87 (MajorAtLeast) field | `libs/atlas-packet/character/clientbound/spawn.go:85` (×2 lines 85,199) | `IsRegion("GMS") && MajorAtLeast(87)` | | | | | | |
| character/status_message: GMS>=95 status field | `libs/atlas-packet/character/clientbound/status_message.go:528` (×2 lines 528,561) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| character/view_all: GMS>87 view-all field | `libs/atlas-packet/character/clientbound/view_all.go:83` (×2 lines 83,103) | `Region()=="GMS" && MajorVersion()>87` | | | | | | |

### data

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| character/data: GMS>28 or JMS (dominant codec gate, ~23 instances) | `libs/atlas-packet/character/data.go:114` (×23 instances) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | | | | | | |
| character/data: GMS>28 && <=87 or JMS narrow range | `libs/atlas-packet/character/data.go:148` (×2 lines 148,207) | `(Region()=="GMS" && MajorVersion()>28 && MajorVersion()<=87) \|\| Region()=="JMS"` | | | | | | |
| character/data: any-region >12 inner field | `libs/atlas-packet/character/data.go:286` (×2 lines 286,362; nested in Region=="GMS" block) | `MajorVersion()>12` | | | | | | |
| character/data: any-region >=87 inner field | `libs/atlas-packet/character/data.go:293` (×2 lines 293,369; nested in Region=="GMS" block) | `MajorVersion()>=87` | | | | | | |
| character/data: GMS>12 or JMS | `libs/atlas-packet/character/data.go:386` (×4 lines 386,471,643,664) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | | | | | | |
| character/data: GMS<28 legacy field (~10 instances) | `libs/atlas-packet/character/data.go:419` (×10 lines 419,432,441,450,459,490,496,502,508,514) | `Region()=="GMS" && MajorVersion()<28` | | | | | | |
| character/data: GMS region-only inner gate | `libs/atlas-packet/character/data.go:153` (×4 lines 153,212,285,361) | `Region()=="GMS"` | | | | | | |
| character/data: GMS>=84 Evan job guard (MajorAtLeast) | `libs/atlas-packet/character/data.go:269` (×2 lines 269,342) | `IsRegion("GMS") && MajorAtLeast(84) && isEvanJob(...)` | | | | | | |

### serverbound

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| character/create: GMS>=73 or JMS fields (intra-legacy discriminator!) | `libs/atlas-packet/character/serverbound/create.go:113` (×2 lines 113,148) | `(Region()=="GMS" && MajorVersion()>=73) \|\| Region()=="JMS"` | | | | | | |
| character/create: GMS>=87 or JMS field (MajorAtLeast) | `libs/atlas-packet/character/serverbound/create.go:116` | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | | | | | | |
| character/create: GMS>28 not JMS field | `libs/atlas-packet/character/serverbound/create.go:130` | `(Region()=="GMS" && MajorVersion()>28) && Region()!="JMS"` | | | | | | |
| character/create: GMS<=28 field | `libs/atlas-packet/character/serverbound/create.go:133` (×2 lines 133,183) | `Region()=="GMS" && MajorVersion()<=28` | | | | | | |
| character/create: GMS<=28 or JMS field | `libs/atlas-packet/character/serverbound/create.go:176` | `(Region()=="GMS" && MajorVersion()<=28) \|\| Region()=="JMS"` | | | | | | |
| character/create: GMS not >=87 field (MajorAtLeast) | `libs/atlas-packet/character/serverbound/create.go:157` | `IsRegion("GMS") && !MajorAtLeast(87)` | | | | | | |
| character/delete: GMS>82 BPIC-present branch | `libs/atlas-packet/character/serverbound/delete.go:51` (×2 lines 51,64) | `Region()=="GMS" && MajorVersion()>82` | | | | | | |
| character/delete: GMS else (no BPIC) branch | `libs/atlas-packet/character/serverbound/delete.go:53` (×2 lines 53,67; else-if of above) | `Region()=="GMS"` (else-if) | | | | | | |
| character/expression (sb): GMS>87 expression fields | `libs/atlas-packet/character/serverbound/expression.go:58` (×2 lines 58,73) | `Region()=="GMS" && MajorVersion()>87` | | | | | | |
| character/heal_over_time: GMS<=95 or JMS field | `libs/atlas-packet/character/serverbound/heal_over_time.go:81` (×2 lines 81,98) | `(Region()=="GMS" && MajorVersion()<=95) \|\| Region()=="JMS"` | | | | | | |
| character/move: GMS>28 field | `libs/atlas-packet/character/serverbound/move.go:73` (×2 lines 73,101) | `Region()=="GMS" && MajorVersion()>28` | | | | | | |
| character/move: GMS>=84 fields (MajorAtLeast, 6 instances) | `libs/atlas-packet/character/serverbound/move.go:64` (×6 lines 64,69,76,92,97,104) | `IsRegion("GMS") && MajorAtLeast(84)` | | | | | | |

---

## libs/atlas-packet/chat

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| chat/multi: GMS>=95 updateTime field | `libs/atlas-packet/chat/serverbound/multi.go:54` (×2 lines 54,71) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| chat/whisper: GMS>=87 or JMS gate | `libs/atlas-packet/chat/serverbound/whisper.go:28` | `(Region()=="GMS" && MajorVersion()>=87) \|\| Region()=="JMS"` | | | | | | |

---

## libs/atlas-packet/field

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| field/admin_result: GMS>=95 branch | `libs/atlas-packet/field/clientbound/admin_result.go:110` (×2 lines 110,201) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| field/admin_result: GMS>=87 && <95 branch | `libs/atlas-packet/field/clientbound/admin_result.go:129` (×2 lines 129,219) | `Region()=="GMS" && MajorVersion()>=87 && MajorVersion()<95` | | | | | | |
| field/admin_result: GMS>=84 && <87 branch | `libs/atlas-packet/field/clientbound/admin_result.go:146` (×2 lines 146,235) | `Region()=="GMS" && MajorVersion()>=84 && MajorVersion()<87` | | | | | | |
| field/admin_result: GMS<84 branch | `libs/atlas-packet/field/clientbound/admin_result.go:163` (×2 lines 163,251) | `Region()=="GMS" && MajorVersion()<84` | | | | | | |
| field/affected_area_created: GMS>=95 area type field | `libs/atlas-packet/field/clientbound/affected_area_created.go:92` | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| field/foothold_info: any-region MajorAtLeast(95) gate | `libs/atlas-packet/field/clientbound/foothold_info.go:88` | `MajorAtLeast(95)` _(no Region guard)_ | | | | | | |
| field/set_field: GMS>=95 field | `libs/atlas-packet/field/clientbound/set_field.go:52` (×2 lines 52,100) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| field/set_field: GMS>28 or JMS field | `libs/atlas-packet/field/clientbound/set_field.go:62` (×2 lines 62,110) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | | | | | | |
| field/set_field: GMS>=87 or JMS (MajorAtLeast, 4 instances) | `libs/atlas-packet/field/clientbound/set_field.go:47` (×4 lines 47,77,95,125) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | | | | | | |
| field/warp_to_map: GMS>=95 field (4 instances) | `libs/atlas-packet/field/clientbound/warp_to_map.go:98` (×4 lines 98,118,144,161) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| field/warp_to_map: GMS>28 or JMS field | `libs/atlas-packet/field/clientbound/warp_to_map.go:107` (×2 lines 107,153) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | | | | | | |
| field/warp_to_map: GMS>28 field | `libs/atlas-packet/field/clientbound/warp_to_map.go:123` (×2 lines 123,166) | `Region()=="GMS" && MajorVersion()>28` | | | | | | |
| field/warp_to_map: GMS>=87 or JMS (MajorAtLeast) | `libs/atlas-packet/field/clientbound/warp_to_map.go:93` (×2 lines 93,139) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | | | | | | |
| field/witch_tower_score_update: GMS MajorAtLeast(95) | `libs/atlas-packet/field/clientbound/witch_tower_score_update.go:38` | `Region()=="GMS" && MajorAtLeast(95)` | | | | | | |
| field/change (sb): GMS>=83 field | `libs/atlas-packet/field/serverbound/change.go:72` (×2 lines 72,101) | `Region()=="GMS" && MajorVersion()>=83` | | | | | | |
| field/general (sb): GMS>=87 or JMS (MajorAtLeast) | `libs/atlas-packet/field/serverbound/general.go:46` (×2 lines 46,60) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | | | | | | |
| field/sue_character: any-region >=95 field | `libs/atlas-packet/field/serverbound/sue_character.go:61` (×2 lines 61,75) | `MajorVersion()>=95` _(no Region guard)_ | | | | | | |

---

## libs/atlas-packet/guild

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| guild/operation: GMS>=84 or JMS trailing ints (MajorAtLeast) | `libs/atlas-packet/guild/clientbound/operation.go:769` (×2 lines 769,786) | `(IsRegion("GMS") && MajorAtLeast(84)) \|\| Region()=="JMS"` | | | | | | |

---

## libs/atlas-packet/interaction

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| interaction/operation_chat: GMS>=87 or JMS gate | `libs/atlas-packet/interaction/serverbound/operation_chat.go:33` | `(Region()=="GMS" && MajorVersion()>=87) \|\| Region()=="JMS"` | | | | | | |

---

## libs/atlas-packet/login

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| login/auth_login_failed: GMS branch | `libs/atlas-packet/login/clientbound/auth_login_failed.go:34` (×2 lines 34,47) | `Region()=="GMS"` | | | | | | |
| login/auth_permanent_ban: GMS branch | `libs/atlas-packet/login/clientbound/auth_permanent_ban.go:34` (×2 lines 34,56) | `Region()=="GMS"` | | | | | | |
| login/auth_permanent_ban: non-GMS branch | `libs/atlas-packet/login/clientbound/auth_permanent_ban.go:42` (×2 lines 42,60) | `Region()!="GMS"` | | | | | | |
| login/auth_success: GMS>=95 field | `libs/atlas-packet/login/clientbound/auth_success.go:51` (×2 lines 51,113) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| login/auth_success: GMS region-only gate | `libs/atlas-packet/login/clientbound/auth_success.go:44` (×4 lines 44,57,106,119) | `Region()=="GMS"` | | | | | | |
| login/auth_success: >12 inner field (nested in GMS block) | `libs/atlas-packet/login/clientbound/auth_success.go:58` (×4 lines 58,63,120,125) | `MajorVersion()>12` | | | | | | |
| login/auth_success: MajorAtLeast(84) inner field (nested in GMS block) | `libs/atlas-packet/login/clientbound/auth_success.go:81` (×2 lines 81,143) | `MajorAtLeast(84)` | | | | | | |
| login/auth_temporary_ban: GMS branch | `libs/atlas-packet/login/clientbound/auth_temporary_ban.go:48` (×2 lines 48,64) | `Region()=="GMS"` | | | | | | |
| login/server_ip: GMS>12 or JMS field | `libs/atlas-packet/login/clientbound/server_ip.go:74` (×2 lines 74,92) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | | | | | | |
| login/server_list_entry: GMS>12 or JMS field | `libs/atlas-packet/login/clientbound/server_list_entry.go:80` (×2 lines 80,123) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | | | | | | |
| login/server_list_entry: GMS region-only gate | `libs/atlas-packet/login/clientbound/server_list_entry.go:56` (×2 lines 56,97) | `Region()=="GMS"` | | | | | | |
| login/server_list_entry: >12 inner field (nested) | `libs/atlas-packet/login/clientbound/server_list_entry.go:57` (×2 lines 57,98) | `MajorVersion()>12` | | | | | | |
| login/all_character_list_request: GMS>=87 (MajorAtLeast) | `libs/atlas-packet/login/serverbound/all_character_list_request.go:57` (×2 lines 57,72) | `IsRegion("GMS") && MajorAtLeast(87)` | | | | | | |
| login/character_select: GMS>12 field | `libs/atlas-packet/login/serverbound/character_select.go:47` (×2 lines 47,59) | `Region()=="GMS" && MajorVersion()>12` | | | | | | |
| login/character_select_register_pic: GMS region-only | `libs/atlas-packet/login/serverbound/character_select_register_pic.go:58` (×2 lines 58,72) | `Region()=="GMS"` | | | | | | |
| login/character_select_with_pic: GMS region-only | `libs/atlas-packet/login/serverbound/character_select_with_pic.go:53` (×2 lines 53,67) | `Region()=="GMS"` | | | | | | |
| login/request: GMS region-only | `libs/atlas-packet/login/serverbound/request.go:78` (×2 lines 78,95) | `Region()=="GMS"` | | | | | | |
| login/server_status_request: GMS region-only | `libs/atlas-packet/login/serverbound/server_status_request.go:36` (×2 lines 36,48) | `Region()=="GMS"` | | | | | | |
| login/world_character_list_request: GMS>28 field | `libs/atlas-packet/login/serverbound/world_character_list_request.go:53` (×2 lines 53,70) | `Region()=="GMS" && MajorVersion()>28` | | | | | | |
| login/world_character_list_request: GMS>12 field | `libs/atlas-packet/login/serverbound/world_character_list_request.go:58` (×2 lines 58,76) | `Region()=="GMS" && MajorVersion()>12` | | | | | | |

---

## libs/atlas-packet/model

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| model/asset: GMS>12 or JMS (6 instances) | `libs/atlas-packet/model/asset.go:195` (×6 lines 195,208,246,260,377,415) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | | | | | | |
| model/asset: GMS>28 or JMS (4 instances) | `libs/atlas-packet/model/asset.go:213` (×4 lines 213,264,347,419) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | | | | | | |
| model/asset: GMS>=84 (MajorAtLeast) | `libs/atlas-packet/model/asset.go:217` (×2 lines 217,428) | `IsRegion("GMS") && MajorAtLeast(84)` | | | | | | |
| model/attack_info: GMS>=84 DR-block fields (6 instances) | `libs/atlas-packet/model/attack_info.go:83` (×6 lines 83,88,96,192,200,210) | `Region()=="GMS" && MajorVersion()>=84` | | | | | | |
| model/attack_info: GMS>=95 fields (~14 instances) | `libs/atlas-packet/model/attack_info.go:93` (×14 instances) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| model/avatar: GMS<=28 fields (5 instances) | `libs/atlas-packet/model/avatar.go:50` (×5 lines 50,62,70,104,116) | `Region()=="GMS" && MajorVersion()<=28` | | | | | | |
| model/avatar: GMS>28 or JMS field | `libs/atlas-packet/model/avatar.go:78` (×2 lines 78,141) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | | | | | | |
| model/character_list_entry: GMS<=28 fields | `libs/atlas-packet/model/character_list_entry.go:59` (×2 lines 59,86) | `Region()=="GMS" && MajorVersion()<=28` | | | | | | |
| model/character_statistics: GMS>=95 field | `libs/atlas-packet/model/character_statistics.go:113` (×2 lines 113,189) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| model/character_statistics: GMS>28 or JMS (4 instances) | `libs/atlas-packet/model/character_statistics.go:98` (×4 lines 98,135,175,211) | `(Region()=="GMS" && MajorVersion()>28) \|\| Region()=="JMS"` | | | | | | |
| model/character_statistics: GMS region-only inner gate | `libs/atlas-packet/model/character_statistics.go:142` (×2 lines 142,218) | `Region()=="GMS"` | | | | | | |
| model/character_statistics: >12 inner field (nested) | `libs/atlas-packet/model/character_statistics.go:143` (×2 lines 143,219) | `MajorVersion()>12` | | | | | | |
| model/character_statistics: >=87 inner field (nested) | `libs/atlas-packet/model/character_statistics.go:150` (×2 lines 150,226) | `MajorVersion()>=87` | | | | | | |
| model/character_temporary_stat: GMS>=95 stat mask | `libs/atlas-packet/model/character_temporary_stat.go:174` (×2 lines 174,723) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| model/character_temporary_stat: post87 GMS or JMS (MajorAtLeast) | `libs/atlas-packet/model/character_temporary_stat.go:176` | `(Region()=="GMS" && MajorAtLeast(87)) \|\| jms` | | | | | | |
| model/character_temporary_stat: GMS>=87 stat enable (MajorAtLeast) | `libs/atlas-packet/model/character_temporary_stat.go:105` | `IsRegion("GMS") && MajorAtLeast(87)` | | | | | | |
| model/damage_info: GMS>=83 damage field | `libs/atlas-packet/model/damage_info.go:48` (×2 lines 48,73) | `Region()=="GMS" && MajorVersion()>=83` | | | | | | |
| model/damage_taken_info: GMS>=95 taken-damage field | `libs/atlas-packet/model/damage_taken_info.go:103` (×2 lines 103,136) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| model/monster: GMS>12 or JMS (4 instances) | `libs/atlas-packet/model/monster.go:497` (×4 lines 497,509,527,539) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | | | | | | |
| model/monster: GMS>=87 or JMS (MajorAtLeast) | `libs/atlas-packet/model/monster.go:512` (×2 lines 512,542) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | | | | | | |
| model/movement: not-GMS or >=88 boundary (MajorAtLeast) | `libs/atlas-packet/model/movement.go:131` (×2 lines 131,222) | `!IsRegion("GMS") \|\| MajorAtLeast(88)` | | | | | | |
| model/skill_prepare_info: GMS>=95 or JMS | `libs/atlas-packet/model/skill_prepare_info.go:22` | `(Region()=="GMS" && MajorVersion()>=95) \|\| Region()=="JMS"` | | | | | | |

---

## libs/atlas-packet/monster

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| monster/catch_monster: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/monster/clientbound/catch_monster.go:55` | `IsRegion("GMS") && MajorAtLeast(95)` | | | | | | |
| monster/monster_special_effect_by_skill: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/monster/clientbound/monster_special_effect_by_skill.go:58` | `IsRegion("GMS") && MajorAtLeast(95)` | | | | | | |
| monster/clientbound/movement: GMS>=87 or JMS (MajorAtLeast, 4 instances) | `libs/atlas-packet/monster/clientbound/movement.go:56` (×4 lines 56,63,77,84) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | | | | | | |
| monster/clientbound/spawn: GMS>12 or JMS | `libs/atlas-packet/monster/clientbound/spawn.go:47` (×2 lines 47,64) | `(Region()=="GMS" && MajorVersion()>12) \|\| Region()=="JMS"` | | | | | | |
| monster/serverbound/movement: GMS>=84 or JMS (MajorAtLeast) | `libs/atlas-packet/monster/serverbound/movement.go:71` (×2 lines 71,106) | `(IsRegion("GMS") && MajorAtLeast(84)) \|\| Region()=="JMS"` | | | | | | |
| monster/serverbound/movement: GMS>=87 or JMS (MajorAtLeast, 4 instances) | `libs/atlas-packet/monster/serverbound/movement.go:80` (×4 lines 80,86,115,121) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | | | | | | |

---

## libs/atlas-packet/npc

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| npc/conversation: GMS>=87 or JMS (MajorAtLeast) | `libs/atlas-packet/npc/clientbound/conversation.go:354` | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | | | | | | |
| npc/shop_list: GMS>=87 shop field | `libs/atlas-packet/npc/clientbound/shop_list.go:54` (×2 lines 54,83) | `Region()=="GMS" && MajorVersion()>=87` | | | | | | |
| npc/shop_list: GMS>=95 shop field | `libs/atlas-packet/npc/clientbound/shop_list.go:57` (×2 lines 57,86) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| npc/shop_buy: GMS region-only | `libs/atlas-packet/npc/serverbound/shop_buy.go:41` (×2 lines 41,54) | `Region()=="GMS"` | | | | | | |

---

## libs/atlas-packet/party

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| party/invite: GMS>=87 or JMS (MajorAtLeast) | `libs/atlas-packet/party/clientbound/invite.go:45` (×2 lines 45,63) | `(IsRegion("GMS") && MajorAtLeast(87)) \|\| Region()=="JMS"` | | | | | | |
| party/town_portal: GMS>=95 gate | `libs/atlas-packet/party/clientbound/town_portal.go:68` | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| party/member_data: GMS>=95 field | `libs/atlas-packet/party/member_data.go:87` (×2 lines 87,125) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |

---

## libs/atlas-packet/pet

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| pet/chat: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/pet/serverbound/chat.go:60` (×2 lines 60,77) | `IsRegion("GMS") && MajorAtLeast(95)` | | | | | | |
| pet/drop_pick_up: GMS>=87 (MajorAtLeast) | `libs/atlas-packet/pet/serverbound/drop_pick_up.go:71` (×2 lines 71,97) | `IsRegion("GMS") && MajorAtLeast(87)` | | | | | | |

---

## libs/atlas-packet/stat

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| stat/changed: GMS>=95 stat-change field | `libs/atlas-packet/stat/clientbound/changed.go:51` (×2 lines 51,106) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |

---

## libs/atlas-packet/summon

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| summon/clientbound/attack: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/summon/clientbound/attack.go:83` (×2 lines 83,106) | `IsRegion("GMS") && MajorAtLeast(95)` | | | | | | |
| summon/clientbound/spawn: JMS v185 gate (MajorAtLeast 185, checked first) | `libs/atlas-packet/summon/clientbound/spawn.go:135` | `MajorAtLeast(185)` _(JMS = v185; no Region guard — true only for JMS)_ | | | | | | |
| summon/clientbound/spawn: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/summon/clientbound/spawn.go:137` | `IsRegion("GMS") && MajorAtLeast(95)` | | | | | | |
| summon/serverbound/attack: JMS v185 gate (MajorAtLeast 185, checked first) | `libs/atlas-packet/summon/serverbound/attack.go:119` | `MajorAtLeast(185)` _(JMS = v185; no Region guard — true only for JMS)_ | | | | | | |
| summon/serverbound/attack: GMS>=84 (MajorAtLeast) | `libs/atlas-packet/summon/serverbound/attack.go:121` | `IsRegion("GMS") && MajorAtLeast(84)` | | | | | | |
| summon/serverbound/attack: GMS>=95 (MajorAtLeast) | `libs/atlas-packet/summon/serverbound/attack.go:136` | `IsRegion("GMS") && MajorAtLeast(95)` | | | | | | |

---

## libs/atlas-packet/ui

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| ui/lock: GMS>=90 screen-lock field | `libs/atlas-packet/ui/clientbound/lock.go:33` (×2 lines 33,44) | `Region()=="GMS" && MajorVersion()>=90` | | | | | | |

---

## libs/atlas-seeder

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| seeder/catalog: zero-version validation guard | `libs/atlas-seeder/catalog.go:54` | `MajorVersion()==0 \|\| MinorVersion()==0` _(not a legacy discriminator; seeder-internal validity check)_ | | | | | | |

---

## services/atlas-account

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| account/processor: GMS>=87 behaviour gate (MajorAtLeast) | `services/atlas-account/atlas.com/account/account/processor.go:126` | `IsRegion("GMS") && MajorAtLeast(87)` | | | | | | |

---

## services/atlas-channel

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| channel/main: GMS<=28 legacy mode init | `services/atlas-channel/atlas.com/channel/main.go:391` | `Region()=="GMS" && MajorVersion()<=28` | | | | | | |
| channel/session: GMS<=12 session mode | `services/atlas-channel/atlas.com/channel/session/model.go:40` | `Region()=="GMS" && MajorVersion()<=12` | | | | | | |
| channel/character_cash_item_use: GMS>=95 item-use fields (~30 instances) | `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:32` (×30 instances, lines 32–494) | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| channel/socket/model/damage_taken_info: GMS>=95 field | `services/atlas-channel/atlas.com/channel/socket/model/damage_taken_info.go:66` | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |
| channel/socket/writer/character_attack_common: GMS>=95 field | `services/atlas-channel/atlas.com/channel/socket/writer/character_attack_common.go:180` | `Region()=="GMS" && MajorVersion()>=95` | | | | | | |

---

## services/atlas-character

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| character/processor: GMS MajorAtMost(94) gate (equivalent to <=94, i.e., <95) | `services/atlas-character/atlas.com/character/character/processor.go:56` | `IsRegion("GMS") && MajorAtMost(94)` | | | | | | |

---

## services/atlas-login

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| login/main: GMS<=28 legacy mode init | `services/atlas-login/atlas.com/login/main.go:277` | `Region()=="GMS" && MajorVersion()<=28` | | | | | | |
| login/session: GMS<=12 session mode | `services/atlas-login/atlas.com/login/session/model.go:35` | `Region()=="GMS" && MajorVersion()<=12` | | | | | | |

---

## services/atlas-tenants (test assertions — not production gates)

> These rows are TEST FILE assertions that happen to use `MajorVersion()` comparisons. They are
> NOT version-gated behavior branches; they are test correctness checks. Included for completeness
> per the `!=83` / `!=90` predicate forms in the brief's pre-computed facts list.

| Branch (semantic) | file:line | Predicate | v79 | v72 | v61 | v48 | Correct? | Action |
|---|---|---|---|---|---|---|---|---|
| tenants builder/processor test !=83 assertion | `services/atlas-tenants/atlas.com/tenants/tenant/builder_test.go:41` (×3 lines 41,126; processor_test.go:122) | `MajorVersion()!=83` _(test assertion only; t.Fatalf if false)_ | | | | | | |
| tenants builder/processor test !=90 assertion | `services/atlas-tenants/atlas.com/tenants/tenant/builder_test.go:148` (×2 lines 148; processor_test.go:230) | `MajorVersion()!=90` _(test assertion only; t.Fatalf if false)_ | | | | | | |

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
