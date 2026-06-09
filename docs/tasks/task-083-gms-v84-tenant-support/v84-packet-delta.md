# v83 → v84 Packet / Opcode / Version-Branch Delta

Source of truth for task-083 (FR-1.4). Every code/template change cites a row here.

## 0. IDB inventory & dispatch-table anchors
| IDB | port | dispatch table (inbound) addr | dispatch table (outbound) addr | naming density |
|---|---|---|---|---|

## 1. Inbound (handler) opcode map  (FR-1.1, FR-1.3)
| logical name | v83 opcode | v84 opcode | classification | evidence (IDB fn/addr or ref version) |
|---|---|---|---|---|

## 2. Outbound (writer) opcode map  (FR-1.1, FR-1.3)
| logical name | v83 opcode | v84 opcode | classification | evidence |
|---|---|---|---|---|

## 3. Packet-structure delta (FR-1.2)
### 3.1 In-scope flows (exhaustive): login handshake, auth, world/channel list, character list, character select / PIC-PIN, enter-channel, map load (spawn/field), movement, chat
### 3.2 Spot-checked elsewhere (what was checked, what was assumed)

## 4. usesPin determination (OQ-1)

## 5. Version-branch audit table (FR-3.1, FR-3.3)

Grep command: `grep -rn 'Region()\|MajorVersion()\|MinorVersion()' services/ libs/ --include='*.go' | grep -v '_test.go' | grep -E '==|!=|>=|<=|>|<'`

Total hits: **412**

Evaluation key (for GMS tenant, region="GMS"):
- `> 83` → v83: **false**, v84: **true** (boundary predicate — changes at v84)
- `>= 87` → v83: false, v84: false (unchanged)
- `>= 95` → v83: false, v84: false (unchanged)
- `>= 90` → v83: false, v84: false (unchanged)
- `>= 83` → v83: true, v84: true (unchanged)
- `>= 73` → v83: true, v84: true (unchanged)
- `> 87` → v83: false, v84: false (unchanged)
- `> 82` → v83: true, v84: true (unchanged)
- `> 28` → v83: true, v84: true (unchanged)
- `> 12` → v83: true, v84: true (unchanged)
- `<= 12` → v83: true, v84: true (unchanged)
- `<= 28` → v83: true, v84: true (unchanged)
- `<= 83` → v83: **true**, v84: **false** (boundary predicate — changes at v84)
- `<= 87` → v83: true, v84: true (unchanged)
- `<= 95` → v83: true, v84: true (unchanged)
- `== 83` → v83: **true**, v84: **false** (changes at v84)
- `!= "GMS"` or `!= "JMS"` → region comparisons, not version; evaluated as-is for GMS

In-scope flows: login handshake, auth, world/channel list, character list, character select/PIC-PIN, enter-channel, map load (spawn/field), movement, chat.

| branch site (file:line) | predicate | v83 result | v84 result | correct for v84? | action | delta evidence |
|---|---|---|---|---|---|---|
| libs/atlas-packet/buddy/clientbound/invite.go:50 | `Region() != "GMS" \|\| MajorVersion() >= 87` | false (GMS && 83<87) | false (GMS && 84<87) | yes — no change | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/buddy/clientbound/invite.go:76 | `Region() != "GMS" \|\| MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/query_result.go:41 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/query_result.go:53 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_inventory.go:132 | `(Region() == "GMS" && MajorVersion() >= 95) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:36 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:39 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:44 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:45 | `MajorVersion() <= 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:52 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:56 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:60 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:66 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:85 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:90 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:94 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:97 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:111 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:114 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:119 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:120 | `MajorVersion() <= 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:127 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:131 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:135 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:141 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:162 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:170 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:177 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:180 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/item_use.go:38 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/item_use.go:50 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy.go:44 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy.go:57 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy.go:75 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy.go:87 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:46 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:56 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:80 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:89 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:46 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:56 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:80 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:89 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:49 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:59 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:65 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:83 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:92 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:98 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go:42 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go:52 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go:72 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go:81 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/attack.go:107 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/attack.go:165 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/damage.go:55 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/damage.go:78 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/expression.go:62 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/expression.go:65 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/expression.go:80 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/expression.go:83 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/info.go:106 | `(Region() == "GMS" && MajorVersion() <= 87) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/info.go:116 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (char-info/spawn flow) |
| libs/atlas-packet/character/clientbound/info.go:173 | `(Region() == "GMS" && MajorVersion() <= 87) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/info.go:183 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (char-info/spawn flow) |
| libs/atlas-packet/character/clientbound/item_upgrade.go:91 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/item_upgrade.go:98 | `(Region() == "GMS" && MajorVersion() > 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/item_upgrade.go:114 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/item_upgrade.go:119 | `(Region() == "GMS" && MajorVersion() > 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:56 | `Region() == "GMS" && MajorVersion() <= 28` | true | true | yes — v84 still > 28, predicate is false; note: the `<= 28` check is the early-return path | unchanged (correct) | pending Phase A (character list) |
| libs/atlas-packet/character/clientbound/list.go:61 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:63 | `MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:91 | `Region() == "GMS" && MajorVersion() <= 28` | true | true | yes — same as :56 (early-return path) | unchanged (correct) | pending Phase A (character list) |
| libs/atlas-packet/character/clientbound/list.go:96 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:98 | `MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:47 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:81 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:66 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:101 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:79 | `(Region() == "GMS" && MajorVersion() > 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:85 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (character spawn/map load) |
| libs/atlas-packet/character/clientbound/spawn.go:128 | `Region() == "GMS" && MajorVersion() < 95` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:134 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:135 | `MajorVersion() <= 87` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:138 | `MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:182 | `(Region() == "GMS" && MajorVersion() > 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:188 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (character spawn/map load) |
| libs/atlas-packet/character/clientbound/spawn.go:219 | `Region() == "GMS" && MajorVersion() < 95` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:225 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:226 | `MajorVersion() <= 87` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:229 | `MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/status_message.go:528 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/status_message.go:561 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/view_all.go:83 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/view_all.go:103 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:114 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:125 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:148 | `(Region() == "GMS" && MajorVersion() > 28 && MajorVersion() <= 87) \|\| Region() == "JMS"` | true | true | yes — v84 is in (28,87] | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:152 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:170 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:181 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:207 | `(Region() == "GMS" && MajorVersion() > 28 && MajorVersion() <= 87) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:211 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:243 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:266 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:272 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:273 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:280 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:310 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:333 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:339 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:340 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:347 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:364 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:372 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:380 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:390 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:397 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:403 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:410 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:419 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:428 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:437 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:449 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:457 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:468 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:474 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:480 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:486 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:492 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:502 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:526 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:575 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:598 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:621 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:642 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:664 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:672 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:682 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:693 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/serverbound/create.go:113 | `(Region() == "GMS" && MajorVersion() >= 73) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/create.go:116 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (character create — subJobIndex field) |
| libs/atlas-packet/character/serverbound/create.go:129 | `(Region() == "GMS" && MajorVersion() > 28) && Region() != "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/create.go:132 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/create.go:147 | `(Region() == "GMS" && MajorVersion() >= 73) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/create.go:153 | `Region() == "GMS" && MajorVersion() <= 83` | true | **false** | **NO** | migrate+correct | pending Phase A (character create — subJobIndex field) |
| libs/atlas-packet/character/serverbound/create.go:162 | `Region() != "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/create.go:172 | `(Region() == "GMS" && MajorVersion() <= 28) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/create.go:179 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/delete.go:51 | `Region() == "GMS" && MajorVersion() > 82` | true | true | yes | unchanged (correct) | pending Phase A (character delete/PIC) |
| libs/atlas-packet/character/serverbound/delete.go:53 | `Region() == "GMS"` (else-if of `> 82`) | true | true | yes (never reached since `> 82` is true) | unchanged (correct) | pending Phase A (character delete/PIC) |
| libs/atlas-packet/character/serverbound/delete.go:64 | `Region() == "GMS" && MajorVersion() > 82` | true | true | yes | unchanged (correct) | pending Phase A (character delete/PIC) |
| libs/atlas-packet/character/serverbound/delete.go:67 | `Region() == "GMS"` (else-if of `> 82`) | true | true | yes (never reached) | unchanged (correct) | pending Phase A (character delete/PIC) |
| libs/atlas-packet/character/serverbound/expression.go:58 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/expression.go:73 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/heal_over_time.go:60 | `Region() == "GMS" && MajorVersion() <= 95` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/heal_over_time.go:74 | `Region() == "GMS" && MajorVersion() <= 95` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/move.go:56 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (movement — dr0/dr1 header fields) |
| libs/atlas-packet/character/serverbound/move.go:61 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (movement — dr2/dr3 fields) |
| libs/atlas-packet/character/serverbound/move.go:65 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (movement) |
| libs/atlas-packet/character/serverbound/move.go:68 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (movement — dwKey/crc32 fields) |
| libs/atlas-packet/character/serverbound/move.go:82 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (movement — dr0/dr1 decode) |
| libs/atlas-packet/character/serverbound/move.go:87 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (movement — dr2/dr3 decode) |
| libs/atlas-packet/character/serverbound/move.go:91 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (movement) |
| libs/atlas-packet/character/serverbound/move.go:94 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (movement — dwKey/crc32 decode) |
| libs/atlas-packet/chat/serverbound/general.go:45 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (chat) |
| libs/atlas-packet/chat/serverbound/general.go:57 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (chat) |
| libs/atlas-packet/chat/serverbound/multi.go:54 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/chat/serverbound/multi.go:71 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/chat/serverbound/whisper.go:60 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/chat/serverbound/whisper.go:75 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/affected_area_created.go:91 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/effect_weather.go:40 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/effect_weather.go:70 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/set_field.go:46 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (enter-channel/SetField — decode opt header) |
| libs/atlas-packet/field/clientbound/set_field.go:50 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/field/clientbound/set_field.go:60 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/field/clientbound/set_field.go:75 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (enter-channel/SetField — logout gifts block) |
| libs/atlas-packet/field/clientbound/set_field.go:92 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (enter-channel/SetField — decode opt decode) |
| libs/atlas-packet/field/clientbound/set_field.go:96 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/field/clientbound/set_field.go:106 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/field/clientbound/set_field.go:121 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (enter-channel/SetField — logout gifts decode) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:56 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (map load/WarpToMap — decode opt header) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:60 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:69 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:80 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:85 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:96 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (map load/WarpToMap — decode opt decode) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:100 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:109 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:117 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:122 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/serverbound/change.go:71 | `Region() == "GMS" && MajorVersion() >= 83` | true | true | yes | unchanged (correct) | pending Phase A (map load/portal change) |
| libs/atlas-packet/field/serverbound/change.go:100 | `Region() == "GMS" && MajorVersion() >= 83` | true | true | yes | unchanged (correct) | pending Phase A (map load/portal change) |
| libs/atlas-packet/guild/clientbound/operation.go:430 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (guild operation) |
| libs/atlas-packet/guild/clientbound/operation.go:447 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (guild operation) |
| libs/atlas-packet/interaction/serverbound/operation_chat.go:32 | `(Region() == "GMS" && MajorVersion() >= 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/login/clientbound/auth_login_failed.go:34 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_permanent_ban.go:42 | `Region() != "GMS"` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_permanent_ban.go:56 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_permanent_ban.go:60 | `Region() != "GMS"` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:44 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:51 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:57 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:58 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:63 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:81 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:106 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:113 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:119 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:120 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:125 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:143 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_temporary_ban.go:48 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_temporary_ban.go:64 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/server_ip.go:74 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/server IP) |
| libs/atlas-packet/login/clientbound/server_ip.go:92 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/server IP) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:56 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:57 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:64 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:80 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:97 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:98 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:105 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:123 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/serverbound/all_character_list_request.go:56 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (character list — all-char request fields) |
| libs/atlas-packet/login/serverbound/all_character_list_request.go:70 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (character list — all-char request decode) |
| libs/atlas-packet/login/serverbound/character_select.go:47 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (character select) |
| libs/atlas-packet/login/serverbound/character_select_register_pic.go:58 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (PIC-PIN) |
| libs/atlas-packet/login/serverbound/character_select_register_pic.go:72 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (PIC-PIN) |
| libs/atlas-packet/login/serverbound/character_select_with_pic.go:53 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (character select with PIC) |
| libs/atlas-packet/login/serverbound/request.go:78 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (login handshake/auth request) |
| libs/atlas-packet/login/serverbound/request.go:95 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (login handshake/auth request) |
| libs/atlas-packet/login/serverbound/server_status_request.go:36 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:53 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:58 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:70 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:76 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/model/asset.go:195 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:208 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:213 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:243 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:257 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:261 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:344 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:374 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:412 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:416 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/attack_info.go:76 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:84 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:90 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:94 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:124 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:133 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:141 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:146 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:163 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/avatar.go:50 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:62 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:70 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:78 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:104 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:116 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:141 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/character_list_entry.go:59 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character list) |
| libs/atlas-packet/model/character_list_entry.go:86 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character list) |
| libs/atlas-packet/model/character_statistics.go:98 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/model/character_statistics.go:113 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:135 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/model/character_statistics.go:142 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:143 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:150 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:175 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/model/character_statistics.go:189 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:211 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/model/character_statistics.go:218 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:219 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:226 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_temporary_stat.go:105 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (ShadowPartner buff encoding for enter-channel/spawn) |
| libs/atlas-packet/model/character_temporary_stat.go:169 | `(Region() == "GMS" && MajorVersion() >= 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_temporary_stat.go:178 | `(Region() == "GMS" && MajorVersion() >= 95) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/damage_info.go:47 | `Region() == "GMS" && MajorVersion() >= 83` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/damage_taken_info.go:103 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/damage_taken_info.go:136 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/monster.go:497 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/monster.go:509 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/monster.go:512 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster model spawn fields) |
| libs/atlas-packet/model/monster.go:526 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/monster.go:538 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/monster.go:541 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster model spawn fields decode) |
| libs/atlas-packet/model/movement.go:128 | `Region() != "GMS" \|\| MajorVersion() > 83` | false (GMS and <=83) | **true** (GMS and v84 > 83) | **NO** | migrate+correct | pending Phase A (movement element XOffset/YOffset decode) |
| libs/atlas-packet/model/movement.go:217 | `Region() != "GMS" \|\| MajorVersion() > 87` | false | false | yes | unchanged (correct) | pending Phase A (movement element XOffset/YOffset encode) |
| libs/atlas-packet/monster/clientbound/movement.go:55 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster movement) |
| libs/atlas-packet/monster/clientbound/movement.go:62 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster movement) |
| libs/atlas-packet/monster/clientbound/movement.go:76 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster movement) |
| libs/atlas-packet/monster/clientbound/movement.go:83 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster movement) |
| libs/atlas-packet/monster/clientbound/spawn.go:46 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/monster/clientbound/spawn.go:63 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/monster/serverbound/movement.go:70 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster movement SB) |
| libs/atlas-packet/monster/serverbound/movement.go:79 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster movement SB) |
| libs/atlas-packet/monster/serverbound/movement.go:85 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster movement SB) |
| libs/atlas-packet/monster/serverbound/movement.go:105 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster movement SB) |
| libs/atlas-packet/monster/serverbound/movement.go:114 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster movement SB) |
| libs/atlas-packet/monster/serverbound/movement.go:120 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (monster movement SB) |
| libs/atlas-packet/npc/clientbound/conversation.go:352 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (NPC conversation) |
| libs/atlas-packet/npc/clientbound/shop_list.go:53 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/npc/clientbound/shop_list.go:56 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/npc/clientbound/shop_list.go:82 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/npc/clientbound/shop_list.go:85 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/npc/serverbound/shop_buy.go:40 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/npc/serverbound/shop_buy.go:53 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/party/clientbound/invite.go:44 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (party invite) |
| libs/atlas-packet/party/clientbound/invite.go:62 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | migrate+correct | pending Phase A (party invite) |
| libs/atlas-packet/party/member_data.go:73 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/party/member_data.go:101 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/pet/serverbound/chat.go:56 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (pet chat) |
| libs/atlas-packet/pet/serverbound/chat.go:70 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (pet chat) |
| libs/atlas-packet/pet/serverbound/drop_pick_up.go:69 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (pet drop pick-up) |
| libs/atlas-packet/pet/serverbound/drop_pick_up.go:94 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | pending Phase A (pet drop pick-up) |
| libs/atlas-packet/socket/serverbound/channel_connect.go:61 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/socket/serverbound/channel_connect.go:78 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/stat/clientbound/changed.go:51 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/stat/clientbound/changed.go:106 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/ui/clientbound/lock.go:33 | `Region() == "GMS" && MajorVersion() >= 90` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/ui/clientbound/lock.go:44 | `Region() == "GMS" && MajorVersion() >= 90` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-seeder/catalog.go:54 | `MajorVersion() == 0 \|\| MinorVersion() == 0` | n/a (not a boolean gate on version branches; it is a validation guard) | n/a | yes | unchanged (correct) | not a behavioral branch on v83 vs v84 |
| services/atlas-account/atlas.com/account/account/processor.go:165 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | known bug: default-gender incorrectly set to 10 (UI-choose) for v84; should be 0 for v84 until UI-choose is verified for GMS v84. Cited in MEMORY.md `processor.go > 83`. |
| services/atlas-account/atlas.com/account/account/processor.go:394 | `!a.TOS() && Region() != "JMS"` | n/a (TOS check, not a version comparison) — wait, grep matched `!=` in `!= "JMS"` | true (non-JMS GMS tenant) | true | yes | unchanged (correct) | not a MajorVersion branch; TOS is account state |
| services/atlas-channel/atlas.com/channel/main.go:378 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (channel socket init — ByteReadWriter for very old versions) |
| services/atlas-channel/atlas.com/channel/session/model.go:40 | `Region() == "GMS" && MajorVersion() <= 12` | false | false | yes | unchanged (correct) | pending Phase A (login handshake — crypto IV generator) |
| services/atlas-channel/atlas.com/channel/session/model.go:49 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (login handshake) |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:32 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:150 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:171 | `Region() == "GMS" && MajorVersion() >= 95 && itemId%10 == 3` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:185 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:191 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:237 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:304 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:332 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:345 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:352 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:360 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:367 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:374 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:384 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:394 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:400 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:408 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:414 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:421 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:428 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:435 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:442 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:449 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:456 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:463 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:470 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:477 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:484 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:489 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:494 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/init.go:27 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (channel socket init) |
| services/atlas-channel/atlas.com/channel/socket/init.go:33 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (channel socket init) |
| services/atlas-channel/atlas.com/channel/socket/model/damage_taken_info.go:66 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/writer/character_attack_common.go:180 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-character/atlas.com/character/character/processor.go:1336 | `Region() == "GMS" && MajorVersion() == 83` | true | **false** | **NO** | migrate+correct | known bug: auto-AP distribution (beginners lv1-10) uses `== 83` predicate; v84 falls into the `else` branch (normal AP grant). Cited in MEMORY.md `processor.go == 83`. |
| services/atlas-login/atlas.com/login/kafka/consumer/account/session/consumer.go:105 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (auth — JMS license agreement) |
| services/atlas-login/atlas.com/login/main.go:277 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (login socket init — ByteReadWriter for very old versions) |
| services/atlas-login/atlas.com/login/session/model.go:35 | `Region() == "GMS" && MajorVersion() <= 12` | false | false | yes | unchanged (correct) | pending Phase A (login handshake — crypto IV generator) |
| services/atlas-login/atlas.com/login/session/model.go:44 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (login handshake) |
| services/atlas-login/atlas.com/login/socket/init.go:26 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (login socket init) |
| services/atlas-login/atlas.com/login/socket/init.go:32 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (login socket init) |
| services/atlas-renders/atlas.com/renders/character/handler.go:65 | `urlTenant != t.Id().String() \|\| urlRegion != t.Region() \|\| ...` | n/a (string equality comparison used for request validation, not a behavioral version gate) | n/a | yes | unchanged (correct) | not a MajorVersion behavioral branch |
| services/atlas-renders/atlas.com/renders/character/handler.go:66 | `urlVersion != fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())` | n/a (MajorVersion() used in string format for validation, not comparison operator) | n/a | yes | unchanged (correct) | not a boolean-comparison behavioral branch |
| libs/atlas-tenant/tenant.go:70 | `tenant.Region() != m.Region()` | n/a (identity comparison in Is()) | n/a | yes | unchanged (correct) | not a behavioral version gate; equality check on tenant identity |
| libs/atlas-tenant/tenant.go:73 | `tenant.MajorVersion() != m.MajorVersion()` | n/a (identity comparison in Is()) | n/a | yes | unchanged (correct) | not a behavioral version gate; equality check on tenant identity |
| libs/atlas-tenant/tenant.go:76 | `tenant.MinorVersion() != m.MinorVersion()` | n/a (identity comparison in Is()) | n/a | yes | unchanged (correct) | not a behavioral version gate; equality check on tenant identity |

**Additional rows (line numbers verified from grep output):**

| libs/atlas-packet/character/clientbound/spawn.go:142 | `Region() == "JMS"` (else-if of `t.Region() == "GMS"` block at :134) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:233 | `Region() == "JMS"` (else-if of `t.Region() == "GMS"` block at :225) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:131 | `Region() == "JMS"` (JMS inventory extra block) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:143 | `Region() == "JMS"` (JMS inventory extra decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:153 | `Region() == "GMS"` (in `> 28` block: GMS vs JMS branch) | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:156 | `Region() == "JMS"` (else-if in `> 28` block) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:190 | `Region() == "JMS"` (JMS extra field decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:202 | `Region() == "JMS"` (JMS extra field decode 2) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:212 | `Region() == "GMS"` (in `> 28` decode block: GMS vs JMS) | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:215 | `Region() == "JMS"` (else-if in decode `> 28` block) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:283 | `Region() == "JMS"` (JMS extra stats encode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:350 | `Region() == "JMS"` (JMS extra stats decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:617 | `Region() == "JMS"` (JMS quest started extra short) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:638 | `Region() == "JMS"` (JMS quest started extra short decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/create.go:121 | `Region() != "JMS"` (hairColor/skinColor gate) | true | true | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/field/clientbound/set_field.go:53 | `Region() == "JMS"` (JMS extra fields in encode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/set_field.go:99 | `Region() == "JMS"` (JMS extra fields in decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/warp_to_map.go:63 | `Region() == "JMS"` (JMS extra bytes encode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/warp_to_map.go:103 | `Region() == "JMS"` (JMS extra bytes decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/login/clientbound/auth_login_failed.go:47 | `Region() == "GMS"` (Decode branch — same predicate as :34 in grep; line offset differs between Encode/Decode) | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_permanent_ban.go:34 | `Region() == "GMS"` (Encode block — note: table had :35 which was correct per source read; actual grep line is :34) | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:84 | `Region() == "JMS"` (JMS Encode else-if block) | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:146 | `Region() == "JMS"` (JMS Decode else-if block) | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/serverbound/character_select.go:59 | `Region() == "GMS" && MajorVersion() > 12` (Decode branch — same predicate as :47; :59 is in Decode) | true | true | yes | unchanged (correct) | pending Phase A (character select) |
| libs/atlas-packet/login/serverbound/character_select_with_pic.go:67 | `Region() == "GMS"` (Decode branch of :53) | true | true | yes | unchanged (correct) | pending Phase A (character select with PIC) |
| libs/atlas-packet/login/serverbound/server_status_request.go:48 | `Region() == "GMS"` (Decode else-branch — same predicate as :36; :48 is in Decode) | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:60 | `Region() == "JMS"` (Encode JMS branch in socketAddr block) | false | false | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:78 | `Region() == "JMS"` (Decode JMS branch in socketAddr block) | false | false | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/model/asset.go:203 | `Region() == "JMS"` (JMS extra byte in equip encode) | false | false | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:219 | `Region() == "JMS"` (JMS extra fields in cash equip encode block) | false | false | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:252 | `Region() == "JMS"` (JMS extra byte in cash equip encode) | false | false | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:407 | `Region() == "JMS"` (JMS extra byte in equip decode) | false | false | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:427 | `Region() == "JMS"` (JMS extra fields in equip decode block) | false | false | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/character_statistics.go:153 | `Region() == "JMS"` (JMS extra stats in Encode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:229 | `Region() == "JMS"` (JMS extra stats in Decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |

## 6. Provisioning runbook (FR-5.1) + restart sequence (OQ-6)
