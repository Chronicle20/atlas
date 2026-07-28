# Skill Cast Consumption Fidelity — Design

Task: task-160-skill-cast-consumption
Status: Approved PRD → Design
Date: 2026-07-10

## 1. Summary

Three changes, all in atlas-channel (plus one tiny addition each to `libs/atlas-packet` and `libs/atlas-constants`):

1. **`itemConNo` plumbing (FR-1)** — `consumable.Processor.RequestItemConsume` gains a `quantity int16` parameter; `UseSkill` passes the effect's `ItemConsumeAmount()` (floored to 1) and picks the lowest-index slot holding ≥ that amount.
2. **Cast-time `bulletConsume` (FR-2)** — a new gate at the top of `UseSkill`: when `e.BulletConsume() > 0`, find a single USE-compartment slot holding ≥ that many matching projectiles, request their consumption, and rewrite the SHADOW_CLAW statup amount to the client's expected encoding before the buff applies. No qualifying slot → the cast is rejected with zero side effects.
3. **Attack-path SHADOW_CLAW skip (FR-3)** — claw attacks while SHADOW_CLAW is active skip projectile consumption, alongside the existing Soul Arrow skip.

Both PRD open questions (§9.1, §9.2) are resolved below with IDA-verified answers. The §9.1 answer upgrades "maybe a small buff change" to a **hard requirement**: the SHADOW_CLAW buff amount must encode the consumed star's item id, or star-free throwing does not work at all.

## 2. Resolved Open Questions (IDA-verified)

### 2.1 §9.1 — SHADOW_CLAW buff amount encoding: REQUIRED, formula verified

Verified against four client binaries via ida-pro-mcp. The client's bullet resolver `CUserLocal::GetProperBulletPosition` (named in the v87/v95 IDBs; same function unnamed in v83 at `0x949B5D` and jms185 at `0xA0A205`) has an explicit SpiritJavelin branch:

```c
// v83 0x949C4C region (identical logic in v87 0x9C4A50, v95 0x907461, jms185 0xA0A2F4 — all `add …, 1F95EFh`)
if (SecondaryStat.nSpiritJavelin > 0) {
    bulletItemId = nSpiritJavelin + 2069999;      // buff value → star item id
    // scan USE slots for exactly that item id with quantity > 0; NO consume count required
} else {
    // normal path: slot needs quantity >= per-attack consume (×2 under Shadow Partner)
}
```

- The buff **value** is decoded as an **int16** in `SecondaryStat::DecodeForLocal` (v83 `0x781D0E`, SpiritJavelin block at `0x784383–0x784427`, `Decode2` = short). `SecondaryStat::DecodeForRemote` also reads CTS_SpiritJavelin, so remote clients render the correct star for foreign throws — the foreign buff writer passes the same `Amount()` through, so fixing the amount once at cast time covers both encodes.
- **Therefore: SHADOW_CLAW statup amount = `consumedStarItemId − 2069999`** (e.g. Ilbi 2070006 → 7). Amount 0 makes the client resolve bullet item 2069999 (nonexistent), find no slot, and refuse to attack: the buff would be actively harmful. This is now FR-2's fourth requirement, implemented in atlas-channel at cast time (the amount depends on *which* star was consumed — runtime information that WZ data cannot supply, so atlas-data's amount-0 emission at `services/atlas-data/atlas.com/data/skill/reader.go:298` stays as-is and atlas-channel overrides it).
- The `+2069999` constant is byte-identical across v83 / v87 / v95 / jms185 (`0x1F95EF` immediate in all four IDBs — every version we have an IDB for). It is a temporary-stat *value packing*, not a writer-table wire byte, and the codebase already packs client-interpreted stat values in domain code (Sharp Eyes `x<<8|y` at `reader.go:283`). It will be a named constant with the four IDA citations, not config.

### 2.2 §9.2 — The Shadow Stars use packet carries an item-id hint (not a slot)

Already in the decoder: `libs/atlas-packet/model/skill_usage_info.go:32` reads a `uint32` after the skill level when `skillId == skill.NightLordShadowStarsId` into `spiritJavelinItemId`. IDA confirms its provenance: the client's cast gate `CUserLocal::GetSpiritJavelinItemID` (v83 `0x949F5C`) picks a star **item id** and sends it — there is no slot position in this packet. The field has no getter yet; we add one and use it to pin slot selection to the star the client chose.

**New IDA finding — client gate aggregates across slots.** `GetSpiritJavelinItemID` requires claw equipped (weapon type 47), `bulletConsume > 0` (SKILLLEVELDATA+280), an item with `id/10000 == 207` that `IsAbleToConsume`, and **total quantity ≥ bulletConsume summed across every slot holding that item id**. See §5.3 for the resulting divergence from the PRD's single-slot rule.

## 3. Architecture Overview

```
UseSkill (skill/handler/common.go)
│
├─ [NEW, first] bulletConsume gate (bullet_consume.go)
│    e.BulletConsume() > 0 ?
│      load caster (InventoryDecorator) → equipped weapon → projectile classification
│      slot := hint-preferred lowest slot with qty ≥ bulletConsume   ── fail → warn, ABORT cast (no side effects)
│      consumable.RequestItemConsume(…, quantity=bulletConsume)      ── FR-2.4/2.5
│      statups := rewriteShadowClawAmount(statups, itemId−2069999)   ── §2.1
│
├─ HP/MP consume                                   (unchanged)
├─ itemCon consume                                 [CHANGED: quantity = max(1, ItemConsumeAmount());
│                                                   slot = lowest with qty ≥ N via new compartment helper;
│                                                   shortfall → warn + skip, cast proceeds]
├─ cooldown / mount / buff apply / mob buffs / per-skill dispatcher   (unchanged, buff apply
│                                                   consumes the possibly-rewritten statups)
│
consumable.Processor.RequestItemConsume            [CHANGED: +quantity int16 param → provider]
│
Kafka REQUEST_ITEM_CONSUME {source, itemId, quantity}   (schema unchanged)
│
atlas-consumables reserve→consume with quantity          (unchanged, already honors it)

Projectile attack path (socket/handler/character_attack_projectile.go)
└─ Plan(): [NEW] claw + SHADOW_CLAW buff → skip consumption (before computeCount, so
           Shadow Partner ×2 cannot resurrect it), right after the Soul Arrow skip
```

## 4. Detailed Design

### 4.1 FR-1 — `RequestItemConsume` quantity (signature change, not a sibling method)

`services/atlas-channel/atlas.com/channel/consumable/processor.go`:

```go
func (p *Processor) RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id,
    source slot.Position, quantity int16, updateTime uint32) error {
    …(RequestItemConsumeCommandProvider(f, characterId, source, itemId, quantity))
}
```

**Decision: change the signature** rather than add a quantity-bearing sibling. Six existing call sites (`character_item_use.go` ×3, `character_cash_item_use.go`, `pet_food.go`, `pet_item_use.go`) each add a literal `1` — the compiler enforces the sweep (PRD acceptance criterion 3), and a sibling method would invite silent divergence later. `RequestItemConsumeCommandProvider` already takes `quantity int16`; the hardcoded `1` at `processor.go:30` disappears.

Guard: `quantity < 1` is floored to 1 inside the processor (defense in depth for the WZ "0"-string default; FR-1.1).

### 4.2 FR-1 — `UseSkill` itemCon path

In `common.go`, the itemCon block becomes:

```go
amount := int16(e.ItemConsumeAmount())
if amount < 1 { amount = 1 }
if a, found := c.Inventory().CompartmentByType(invType).FindFirstByItemIdWithQuantity(itemId, amount); found {
    _ = requestItemConsumeFunc(consumable.NewProcessor(l, ctx), f, charcon.Id(characterId),
        itemconst.Id(itemId), slot.Position(a.Slot()), amount, 0)
} else {
    l.Warnf("… required [%d]× item [%d] … no single slot qualifies; cast permitted …", …)
}
```

New helper on `compartment.Model` (`services/atlas-channel/atlas.com/channel/compartment/model.go`):

```go
// FindFirstByItemIdWithQuantity returns the matching asset in the lowest-index
// slot whose quantity is at least `quantity`.
func (m Model) FindFirstByItemIdWithQuantity(templateId uint32, quantity int16) (*asset.Model, bool)
```

It must **sort candidates by slot ascending** before scanning — `FindFirstByItemId` relies on incidental asset order, and the projectile path's `resolvePlan` already establishes the sort-by-slot convention (`character_attack_projectile.go:256`). Shortfall behavior (warn + skip + cast proceeds) is unchanged from today's stance (FR-1.3).

### 4.3 FR-2 — cast-time bullet consume (`skill/handler/bullet_consume.go`, new file)

New unexported entry point called first in `UseSkill`:

```go
// consumeCastBullets settles a bulletConsume cast cost. Returns
// (statups, true) on success — statups with SHADOW_CLAW re-amounted — or
// (nil, false) when the cast must be rejected (FR-2.3).
func consumeCastBullets(l logrus.FieldLogger, ctx context.Context, f field.Model,
    characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) ([]statup.Model, bool)
```

Steps:

1. **Trigger**: only runs when `e.BulletConsume() > 0` — data-driven, no skill-id key (FR-2.1). Attack skills with `bulletConsume` (4111005, 14111002, 5201001) never reach `UseSkill`, so no double-consume is possible.
2. **Load caster** with `cp.GetById(cp.InventoryDecorator)` — provides both `Inventory()` and `Equipment()` (the decorator rebuilds the equipment look from the equipable compartment). A load failure rejects the cast (unlike itemCon's permissive stance — here the cost *is* the skill, FR-2.3 rationale).
3. **Weapon → classification**: read the main weapon (slot "weapon" via `c.Equipment().Get("weapon")`, same as `equippedWeapon` in the projectile handler) and map its `item.WeaponType` to the required projectile `item.Classification`. The mapping currently lives as `requiredClassification` in `socket/handler/character_attack_projectile.go:211` — a package `skill/handler` cannot import (`socket/handler` imports `skill/handler`; importing back is a cycle). **Decision: move the mapping to `libs/atlas-constants/item`** as `ProjectileClassificationForWeapon(WeaponType) (Classification, bool)` and refactor the projectile handler to call it. It is a pure WeaponType→Classification fact, exactly what atlas-constants is for (DOM-21); duplicating it in two handlers is the alternative and was rejected. Non-ranged weapon → reject the cast (warn); the client never sends this (its gate requires a claw), so hitting it implies desync or forgery.
4. **Slot selection** (FR-2.2 + §2.2 hint): over `c.Inventory().Consumable().Assets()`, candidates = assets whose `item.GetClassification` matches, sorted by slot ascending.
   - If `info.SpiritJavelinItemId() != 0` **and** that item id's classification matches: restrict candidates to that item id (client-chosen star). A hint that fails classification validation is ignored (forgery guard), falling through to the generic scan.
   - Pick the lowest-index candidate slot with `Quantity() ≥ int(e.BulletConsume())`. Single-slot draw only (PRD rule; §5.3 records the divergence).
5. **No qualifying slot** → log warn with `characterId, skillId, classification/itemId, required` (NFR observability) and return `(nil, false)`. `UseSkill` returns immediately: **no HP/MP consume, no itemCon, no cooldown, no buff, no party/mob application** (FR-2.3; gating before HP/MP also avoids desyncing MP on a rejected cast).
6. **Success** → `RequestItemConsume(f, characterId, itemId, slot, int16(e.BulletConsume()), 0)` — the same reserve→consume flow as every other consumable (FR-2.4), request emitted before the buff applies (FR-2.5, and §9.3's accepted async window: a failed downstream reservation consumes nothing but the buff stands — same failure envelope as today's quantity-1 consumes).
7. **Statup rewrite** (§2.1): return a copy of `e.StatUps()` where the `SHADOW_CLAW` entry's amount is replaced with `int32(itemId) − shadowClawStarEncodingBase`:

```go
// shadowClawStarEncodingBase converts a throwing-star item id to the int16
// value the client expects in the SHADOW_CLAW temporary stat: the client
// computes bulletItemId = value + 2069999 in CUserLocal::GetProperBulletPosition
// (IDA: v83 0x949C4C, v87 0x9C4A50, v95 0x907461, jms185 0xA0A2F4 — all +0x1F95EF).
const shadowClawStarEncodingBase = 2069999
```

Precedent for a runtime-synthesized statup amount: the mount path builds `MONSTER_RIDING` via `statup.NewModel` with the vehicle id read from equipment (`mount.go` / `statup/model.go:16`). The rewritten slice feeds the existing `buff.NewProcessor(…).Apply(…)` call unchanged; the generic buff pipeline (`character_buff_give.go` `AddStat(c.Type(), …, c.Amount(), …)`) carries it to local *and* foreign encodes with no writer changes.

`UseSkill` wiring:

```go
statups := e.StatUps()
if e.BulletConsume() > 0 {
    var ok bool
    if statups, ok = consumeCastBullets(l, ctx, f, characterId, info, e); !ok {
        return nil
    }
}
// … existing flow, with the buff-apply block using `statups` instead of e.StatUps()
```

### 4.4 FR-3 — projectile-attack SHADOW_CLAW skip

In `ProjectileProcessorImpl.Plan` (`character_attack_projectile.go`), immediately after the Soul Arrow skip at line 107:

```go
if weaponType == item.WeaponTypeClaw && hasBuff(buffs, ts.TemporaryStatTypeShadowClaw) {
    p.l.…Debugf("Skipping projectile consumption: Shadow Stars active.")
    return nil, false
}
```

- Placed before `computeCount`, so Shadow Partner's ×2 cannot resurrect a consume (FR-3.1).
- The buff-lookup-failure fallback at lines 97–105 (treat as no buffs → over-consume rather than break the attack) automatically covers this skip too (FR-3.2) — no new code.
- Client parity confirmed by §2.1: while SpiritJavelin is active the client requires only `quantity > 0` of the encoded star and consumes nothing per throw.
- `ts.TemporaryStatTypeShadowClaw` already exists (`libs/atlas-constants/character/temporary_stat.go:46`, `"SHADOW_CLAW"`); the buff service already stores/serves it (it applies today with amount 0).

### 4.5 `libs/atlas-packet` — `SpiritJavelinItemId()` getter

`skill_usage_info.go` decodes the field but exposes no getter. Add `func (m *SkillUsageInfo) SpiritJavelinItemId() uint32` (builder setter already exists). No decode change — the read at line 32 is already correct per §2.2.

## 5. Alternatives Considered

### 5.1 Quantity plumbing shape

| Option | Verdict |
|---|---|
| **A. Add `quantity` to `RequestItemConsume` (chosen)** | 6 call sites, compile-enforced sweep, single emission path. |
| B. New sibling `RequestItemConsumeQuantity` | No call-site churn, but two paths to the same command drift apart; acceptance criterion 3 explicitly wants the compile-verified sweep. |
| C. Options struct / functional options | Overkill for one added scalar on an internal processor. |

### 5.2 Where the SHADOW_CLAW amount gets encoded

| Option | Verdict |
|---|---|
| **A. Cast-time rewrite in atlas-channel (chosen)** | The amount depends on which star was consumed — only the cast path knows. Mirrors the mount MONSTER_RIDING synthesis precedent. Zero changes to atlas-data, atlas-buffs, writers. |
| B. atlas-data emits the amount | Impossible — WZ data doesn't know the player's inventory. |
| C. Encode at the writer layer from a semantic item id (DOM-25-style config resolution) | The statup amount would carry the full item id through atlas-buffs, and the writer would subtract a config-resolved base per stat type. Rejected: the buff writer is generic over stat types (no per-stat transform hooks); the codebase already packs client-interpreted stat values in domain code (Sharp Eyes `x<<8|y`, atlas-data `reader.go:283`); and the base is IDA-verified byte-identical across all four available IDBs (v83/v87/v95/jms185), so there is no per-version table for config to express. A named constant with IDA citations is the honest representation. |

### 5.3 Single-slot vs client-parity aggregate draw (deliberate, documented divergence)

The PRD (FR-2.2) fixes single-slot-with-enough, user-selected, Cosmic parity. IDA (§2.2) shows the v83 client gates on the **aggregate** count per item id across slots. Divergent case: a Night Lord with (say) 150 + 150 Ilbis in two slots — the client sends the cast; the server finds no single slot ≥ 200, rejects, warns. Player sees the cast animation but no buff and no star loss; retrying after stacking into one slot works.

| Option | Verdict |
|---|---|
| **A. Single-slot (chosen — PRD rule)** | One consume command, matches Cosmic, matches the reserve→consume single-slot contract (FR-2.4). Failure mode is benign (nothing consumed, nothing granted, warn logged). |
| B. Multi-slot draw mirroring the client | Full client parity, but needs N consume commands or the compartment `RequestReserve` multi-draw flow (as the attack path's `Emit` does), with partial-failure semantics across slots for a single cast. Materially more machinery for an edge the WZ-verified v83 catalog makes rare. |

The FR-2.3 warn log is the detector: if `bullet_consume_blocked` warnings show up in practice with the hinted item present across multiple slots, option B is the known upgrade path (the attack path's `resolvePlan`/`Emit` already model it). Not built now (YAGNI).

### 5.4 Where the weapon→projectile mapping lives

Chosen: `libs/atlas-constants/item.ProjectileClassificationForWeapon` (DOM-21; two consumers in different packages that cannot import each other). Alternative — duplicating the switch in `skill/handler` — rejected as a divergence seed for a pure constants fact.

## 6. Testing

Builder-pattern setup throughout (no `*_testhelpers.go`), package-level var-func seams per the established `common.go` convention (`loadCasterFunc`, `propRollFunc`, …).

New seams in `skill/handler`: `requestItemConsumeFunc` (wraps `consumable.Processor.RequestItemConsume`) and `applyBuffStatupsFunc` (wraps the buff apply), both `t.Cleanup`-restored.

| Test | Pins |
|---|---|
| `common.go` itemCon: effect with `itemConNo=N>1` | consume request carries quantity N from the lowest slot with ≥ N (AC-1, AC-2) |
| `common.go` itemCon: `itemConNo=0` | quantity 1 (AC-1) |
| `common.go` itemCon: no slot with ≥ N | no emission, warn, cast proceeds (buff still applies) (AC-2) |
| `compartment.FindFirstByItemIdWithQuantity` | lowest-slot-wins incl. unsorted asset order; exact-quantity boundary; not-found |
| `consumable` producer test | `RequestItemConsumeBody.Quantity == N` on the wire (FR-1.4) |
| Shadow Stars cast, qualifying stack (hint honored) | consume of exactly 200 from the hinted star's slot; SHADOW_CLAW statup amount == `itemId−2069999`; buff applied (AC-4) |
| Shadow Stars cast, hint absent/invalid | falls back to lowest qualifying star slot by classification |
| Shadow Stars cast, no single slot ≥ 200 | zero emissions, no buff apply, warn (AC-5) |
| Shadow Stars cast, non-claw weapon / caster load failure | rejected, zero side effects |
| `Plan()` claw + SHADOW_CLAW | `(nil, false)` — no consume, including with Shadow Partner also active (AC-6) |
| `Plan()` claw without SHADOW_CLAW / bow with SHADOW_CLAW-irrelevant buffs | consumption unchanged; Soul Arrow skip regression (AC-6, AC-7) |
| Existing call-site regressions (item use, pet food, cash item) | still emit quantity 1 (AC-3, AC-7) |
| atlas-consumables (optional, PRD §7) | `quantity > 1` command reserves that quantity through the `ConsumeBare` fallback (`consumables/consumable/processor.go:206-221`) |

## 7. Verification

Per CLAUDE.md, in every changed module (`atlas-channel`, `libs/atlas-packet`, `libs/atlas-constants`, optionally `atlas-consumables`): `go test -race ./...`, `go vet ./...`, `go build ./...`; then `docker buildx bake atlas-channel` (and `atlas-consumables` if its module is touched) from the worktree root; `tools/redis-key-guard.sh` from repo root. No Dockerfile/`go.work` changes (no new lib).

## 8. Risks & Non-Goals

- **Async consume vs buff (accepted, §9.3)**: the reservation is the authoritative gate; a downstream reservation failure consumes nothing but leaves the buff applied. Window exists today for quantity-1 consumes; unchanged failure envelope.
- **Aggregate-slot divergence**: see §5.3 — benign failure, logged, upgrade path documented.
- **Exact-200 edge (client-faithful)**: consuming a whole 200-stack leaves 0 of that star; the client then refuses star-free throws until the player holds ≥ 1 of the encoded star. This matches retail client logic (§2.1's `quantity > 0` requirement) — not a server bug, no mitigation.
- Non-goals unchanged from PRD §2: passive no-consume mechanics (task-007 TODO), itemCon shortfall stance, multi-slot itemCon draws, `updateTime` plumbing.
