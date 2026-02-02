# Domain

## rate

### Responsibility

Defines rate types and factors that contribute to computed rate multipliers.

### Core Models

**Type** (`rate/model.go`)

Rate type identifier. Possible values:
- `exp`
- `meso`
- `item_drop`
- `quest_exp`

**Factor** (`rate/model.go`)

A single contribution to a rate.

| Field | Type | Description |
|-------|------|-------------|
| source | string | Origin identifier (e.g., `world`, `buff:2311003`, `item:1234567`) |
| rateType | Type | Which rate this factor affects |
| multiplier | float64 | The multiplier value (1.0 = no change) |

**Computed** (`rate/model.go`)

Aggregated rates for a character.

| Field | Type | Description |
|-------|------|-------------|
| expRate | float64 | Experience multiplier |
| mesoRate | float64 | Meso drop multiplier |
| itemDropRate | float64 | Item drop multiplier |
| questExpRate | float64 | Quest experience multiplier |

### Invariants

- All rate types default to 1.0 when no factors exist.
- Factors with the same source and rateType replace existing factors.
- Computed rates are the product of all factors for each type.

### Processors

**ComputeFromFactors** (`rate/model.go`)

Multiplies all factors for each type to produce Computed rates.

---

## character

### Responsibility

Manages character rate state including factors, item tracking, and lazy initialization.

### Core Models

**Model** (`character/model.go`)

Holds all rate factors for a character.

| Field | Type | Description |
|-------|------|-------------|
| tenant | tenant.Model | Tenant context |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| characterId | uint32 | Character identifier |
| factors | []rate.Factor | Rate factors for this character |

**TrackedItem** (`character/item_tracker.go`)

Represents an item being tracked for time-based rate calculations.

| Field | Type | Description |
|-------|------|-------------|
| TemplateId | uint32 | Item template identifier |
| ItemType | ItemType | `ItemTypeBonusExp` or `ItemTypeCoupon` |
| RateType | rate.Type | Which rate this item affects |
| BonusExpTiers | []equipment.BonusExpTier | Tiers for bonusExp items |
| EquippedSince | *time.Time | When the item was equipped (bonusExp items) |
| AcquiredAt | time.Time | When the coupon was acquired |
| BaseRate | float64 | Base rate multiplier (coupons) |
| DurationMins | int32 | Active duration in minutes (coupons) |

**ItemType** (`character/item_tracker.go`)

| Value | Description |
|-------|-------------|
| ItemTypeBonusExp | Equipment with bonusExp tiers |
| ItemTypeCoupon | Cash coupons with rate/time |

### Invariants

- Characters are created lazily on first rate query or map change.
- BonusExp items only provide a bonus when equipped (EquippedSince is not nil).
- Coupon items expire after DurationMins minutes from AcquiredAt.
- Factors with the same source and rateType are replaced, not duplicated.

### Processors

**Processor** (`character/processor.go`)

| Method | Description |
|--------|-------------|
| GetRates | Retrieves computed rates and factors for a character |
| AddFactor | Adds or updates a rate factor |
| RemoveFactor | Removes a specific rate factor |
| RemoveFactorsBySource | Removes all factors from a specific source |
| UpdateWorldRate | Updates world rate for all characters in a world |
| AddBuffFactor | Adds a rate factor from a buff |
| RemoveBuffFactor | Removes a specific buff rate factor |
| RemoveAllBuffFactors | Removes all rate factors from a specific buff |
| AddItemFactor | Adds a rate factor from an item |
| RemoveItemFactor | Removes a specific item rate factor |
| RemoveAllItemFactors | Removes all rate factors from a specific item |
| TrackBonusExpItem | Starts tracking equipment with time-based EXP bonus tiers |
| TrackCouponItem | Starts tracking a cash coupon with time-limited rate bonus |
| UntrackItem | Stops tracking a time-based rate item |
| UpdateBonusExpEquippedSince | Updates equippedSince timestamp for a bonusExp item |
| GetItemRateFactors | Returns current rate factors from all tracked items |

**Registry** (`character/registry.go`)

In-memory cache for character rate models.

| Method | Description |
|--------|-------------|
| Get | Retrieves a character's rate model |
| GetOrCreate | Retrieves or creates a character's rate model |
| Update | Replaces a character's rate model |
| AddFactor | Adds a factor to a character |
| RemoveFactor | Removes a factor from a character |
| RemoveFactorsBySource | Removes all factors from a source |
| GetAllForWorld | Returns all characters in a world |
| UpdateWorldRate | Updates world rate for all characters in a world |
| Delete | Removes a character from the registry |

**ItemTracker** (`character/item_tracker.go`)

In-memory tracker for time-based rate items.

| Method | Description |
|--------|-------------|
| TrackItem | Starts tracking a time-based rate item |
| UntrackItem | Stops tracking an item |
| UpdateEquippedSince | Updates equippedSince timestamp for a bonusExp item |
| GetTrackedItem | Returns a tracked item if it exists |
| GetAllTrackedItems | Returns all tracked items for a character |
| ComputeItemRateFactors | Calculates current rate factors from tracked items |
| CleanupExpiredItems | Removes expired coupon items |

**InitializeCharacterRates** (`character/initializer.go`)

Queries inventory, buffs, and session data to initialize rate tracking for a character. Called lazily on first rate query or map change event.

---

## bonusexp

### Responsibility

Calculates current bonus EXP tier for equipped items based on gameplay time.

### Processors

**ComputeCurrentTier** (`bonusexp/calculator.go`)

Calculates the current tier and multiplier for an equipped item.

Takes into account:
- Session history (only gameplay time counts)
- Midnight reset rule (hours reset at midnight, but tier is retained until logout or unequip)
- equippedSince timestamp
