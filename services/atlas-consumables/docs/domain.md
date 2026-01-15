# Consumable Domain

## Responsibility

Manages consumable item usage including potions, scrolls, pet food, summoning sacks, and equipment enhancement.

## Core Models

### Consumable Data Model

Represents consumable item template data.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Item template ID |
| success | uint32 | Scroll success rate (%) |
| cursed | uint32 | Scroll curse rate (%) |
| incSTR | uint32 | Strength increase |
| incDEX | uint32 | Dexterity increase |
| incINT | uint32 | Intelligence increase |
| incLUK | uint32 | Luck increase |
| incMHP | uint32 | Max HP increase |
| incMMP | uint32 | Max MP increase |
| incPAD | uint32 | Physical attack increase |
| incMAD | uint32 | Magic attack increase |
| incPDD | uint32 | Physical defense increase |
| incMDD | uint32 | Magic defense increase |
| incACC | uint32 | Accuracy increase |
| incEVA | uint32 | Evasion increase |
| incSpeed | uint32 | Speed increase |
| incJump | uint32 | Jump increase |
| spec | map[SpecType]int32 | Effect specifications |
| monsterSummons | []SummonModel | Monster summoning data |

### Spec Types

| Type | Key | Description |
|------|-----|-------------|
| SpecTypeHP | hp | Direct HP recovery |
| SpecTypeMP | mp | Direct MP recovery |
| SpecTypeHPRecovery | hpR | Percentage HP recovery |
| SpecTypeMPRecovery | mpR | Percentage MP recovery |
| SpecTypeMoveTo | moveTo | Town scroll destination map |
| SpecTypeWeaponAttack | pad | Weapon attack buff |
| SpecTypeMagicAttack | mad | Magic attack buff |
| SpecTypeWeaponDefense | pdd | Weapon defense buff |
| SpecTypeMagicDefense | mdd | Magic defense buff |
| SpecTypeSpeed | speed | Speed buff |
| SpecTypeEvasion | eva | Evasion buff |
| SpecTypeAccuracy | acc | Accuracy buff |
| SpecTypeJump | jump | Jump buff |
| SpecTypeTime | time | Buff duration (milliseconds) |
| SpecTypeInc | inc | Pet food fullness increment |

### Summon Model

| Field | Type | Description |
|-------|------|-------------|
| templateId | uint32 | Monster template ID |
| probability | uint32 | Spawn probability (0-100) |

### Buff Stat Model

| Field | Type | Description |
|-------|------|-------------|
| Type | TemporaryStatType | Stat type |
| Amount | int32 | Buff amount |

### Map Character Registry

In-memory registry tracking character locations.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant context |
| WorldId | byte | World identifier |
| ChannelId | byte | Channel identifier |
| MapId | uint32 | Map identifier |

## Invariants

- Item consumption requires valid item in specified inventory slot
- Scroll usage requires available equipment upgrade slots (except clean slate, spike, cold scrolls)
- Clean slate scrolls cannot add slots beyond original equipment slot count
- Pet food consumption requires a spawned pet with hunger (fullness < 100)
- Cash pet food applies only to matching pet templates
- Equipment cursed by failed scroll is destroyed

## Processors

### Consumable Processor

Handles item consumption requests.

- `RequestItemConsume`: Routes to appropriate consumer based on item classification
  - Classifications 200-202: Standard consumables
  - ClassificationConsumableTownWarp: Town scrolls
  - ClassificationConsumablePetFood: Pet food
  - ClassificationPetConsumable: Cash pet food
  - ClassificationConsumableSummoningSack: Summoning sacks
- `RequestScroll`: Processes equipment enhancement scroll usage
- `ApplyItemEffects`: Applies stat buffs and HP/MP recovery
- `ApplyConsumableEffect`: Applies effects without consuming inventory item

### Scroll Processing

- Success roll: random(0-100) vs success rate
- Clean slate: Adds 1 slot (up to original max)
- Spike/Cold scrolls: Set special properties
- Chaos scrolls: Randomize stats using weighted distribution
- Regular scrolls: Add stat increases, consume 1 slot
- Failed scrolls: Consume 1 slot unless white scroll used
- Cursed items: Destroyed on curse roll

### Chaos Scroll Stat Adjustment Distribution

| Adjustment | Probability |
|------------|-------------|
| -5 | 4.94% |
| -4 | 2.97% |
| -3 | 3.65% |
| -2 | 8.00% |
| -1 | 13.70% |
| 0 | 18.38% |
| +1 | 19.31% |
| +2 | 15.87% |
| +3 | 10.21% |
| +4 | 1.98% |
| +5 | 0.99% |

HP/MP adjustments are multiplied by 10.

### Equipable Processor

Handles equipment stat modifications.

- `ChangeStat`: Applies stat changes via Kafka command

### Pet Processor

Handles pet feeding.

- `HungriestByOwnerProvider`: Finds spawned pet with lowest fullness
- `AwardFullness`: Issues fullness award command

### Map Processor

Handles character teleportation.

- `WarpRandom`: Teleports character to random spawn point on target map

### Character Map Processor

Manages character location registry.

- `Enter`: Registers character location on login
- `Exit`: Removes character from registry on logout
- `TransitionMap`: Updates registry on map change
- `TransitionChannel`: Updates registry on channel change
- `GetMap`: Retrieves character's current map context
