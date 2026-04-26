# atlas-effective-stats: equipment bonuses silently dropped — Product Requirements Document

Version: v1
Status: Draft (filed as a follow-up from task-027)
Created: 2026-04-26
---

## 1. Overview

`atlas-effective-stats` exposes `GET /api/worlds/{worldId}/channels/{channelId}/characters/{characterId}/stats` which is supposed to return a character's post-equip computed stats — primary stats, HP/MP caps, attack/defense, etc. — plus a `bonuses[]` array describing where each delta came from (passive skill, equipment piece, active buff).

In practice **equipment bonuses never reach the response**. The service fetches the character's equip compartment from atlas-inventory but only captures the asset *IDs*, not the per-asset stats (HP/MP/STR/etc.). The ID-only stub asset has `Slot == 0` (Go zero value), so the `IsEquipped()` filter (`Slot < 0`) skips every entry and the bonus list comes back empty.

Discovered while wiring `useCharacterEffectiveStats` into the redesigned Character Detail page in task-027: the UI displays HP/MP correctly but is unable to show any equipment bonus because the backend never reports one.

## 2. Goals

Primary goals:
- Fix `atlas-effective-stats` so equipment HP / MP / STR / DEX / INT / LUK / W.ATK / M.ATK / W.DEF / M.DEF / Accuracy / Avoidability / Speed / Jump bonuses derived from the character's equipped items appear in the `bonuses[]` array of `GET /api/worlds/.../characters/{id}/stats` and are reflected in the corresponding `Computed.maxHp`, `Computed.strength`, etc.
- Add an integration test that asserts equipped items contribute to the response, using a fixture character with an HP / MP-bearing equipment piece.
- Verify the fix end-to-end against a real character with equipment in a deployed environment (the symptom reproduces today on character `12` in the dev cluster — Medal slot -49, templateId 1142107, hp +47, mp +50).

Non-goals:
- No new endpoints. No schema changes. No changes to atlas-inventory's REST surface.
- No multipliers or stat-curve changes — only flat per-equip bonuses (the existing `fetchEquipmentBonuses` shape).
- No changes to atlas-effective-stats's buff or passive-skill bonus paths; those continue to work correctly today.
- No changes to the atlas-ui presentation layer in task-027. The frontend already displays a `+bonus` annotation when `effective.maxHP > character.maxHp`; once the backend emits the correct delta, the UI will surface it without code changes.

## 3. User Stories

- As a GM looking at the Character Detail page, I see the character's HP cap reflect their currently equipped gear (e.g. `1500 / 2000 +500` when a Medal grants +500 HP), so I can verify gear is being recognized without spelunking the database.
- As a backend developer, when I call `GET /api/worlds/0/channels/0/characters/{id}/stats`, the `bonuses[]` array contains one `{source: "equipment:<assetId>", statType: "TypeMaxHp", amount: <hp>}` entry per equipment piece that grants HP, so I can debug stat issues by reading the response.
- As a future agent extending this service (e.g. cash equipment, account-bonus stats), I can rely on `compartment.Assets[]` carrying full per-asset attributes after `RequestEquipCompartment` returns, instead of having to re-fetch each asset by ID.

## 4. Functional Requirements

### 4.1 Diagnosis (already done — recorded for the fix author)

Reproduction steps (all run from a kubectl-attached pod in the `atlas` namespace):

1. Pick a character with HP-granting equipment. On the dev cluster, character `12` has Medal at slot -49 (templateId 1142107) granting hp=47, mp=50; verified via:

    ```
    curl -s 'http://atlas-inventory:8080/api/characters/12/inventory/compartments?type=1&include=assets' \
      -H 'TENANT_ID: <tenant>' -H 'REGION: GMS' -H 'MAJOR_VERSION: 83' -H 'MINOR_VERSION: 1' \
      | jq '.included[] | select(.type=="assets" and .attributes.slot < 0) | {slot: .attributes.slot, hp: .attributes.hp, mp: .attributes.mp}'
    ```

2. Call atlas-effective-stats for the same character:

    ```
    curl -s 'http://atlas-ingress/api/worlds/0/channels/0/characters/12/stats' \
      -H 'TENANT_ID: ...' -H 'REGION: GMS' -H 'MAJOR_VERSION: 83' -H 'MINOR_VERSION: 1' | jq .
    ```

    Observed: `data.attributes.maxHP === character.attributes.maxHp` and `data.attributes.bonuses` contains only the passive-skill speed entry. Expected: a `{source: "equipment:<assetId>", statType: "TypeMaxHp", amount: 47}` and `{statType: "TypeMaxMp", amount: 50}` entry, with `Computed.maxHp` increased by 47 over the character's base.

### 4.2 Root cause

`services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest.go`:

```go
type CompartmentRestModel struct {
    Id            string           `json:"-"`
    InventoryType int8             `json:"type"`
    Capacity      uint32           `json:"capacity"`
    Assets        []AssetRestModel `json:"-"`
}

func (r *CompartmentRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
    if name == "assets" {
        for _, idStr := range IDs {
            id, err := strconv.Atoi(idStr)
            if err != nil { return err }
            r.Assets = append(r.Assets, AssetRestModel{Id: uint32(id)})
        }
    }
    return nil
}
```

`SetToManyReferenceIDs` is invoked by `jsonapi.Unmarshal` (jtumidanski/api2go) for the *relationship-IDs* phase. It creates stub `AssetRestModel{Id: uint32(id)}` entries — every other field (Slot, Hp, Mp, Strength, …) defaults to zero. `jsonapi.Unmarshal` does not currently call any method on `CompartmentRestModel` that would hydrate those stubs from the response's `included` array, so the per-asset attributes are silently lost.

Downstream, `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go::fetchEquipmentBonuses`:

```go
for _, asset := range compartment.Assets {
    if !asset.IsEquipped() {            // Slot < 0 — but stub Slot is 0
        continue
    }
    equipData, ok := asset.GetEquipableData()  // also gates on Slot < 0
    if !ok { continue }
    ...
}
```

Because every stub has `Slot == 0`, both `IsEquipped()` and `GetEquipableData()` return false / `(_, false)`, so the loop body never runs. `bonuses` returns `[]stat.Bonus{}` and `m.WithBonuses(equipmentBonuses)` adds no equipment bonuses.

### 4.3 Required behavior after fix

For each equipped asset (slot < 0) in compartment type 1 (equip), with at least one non-zero stat in {strength, dexterity, intelligence, luck, hp, mp, weaponAttack, magicAttack, weaponDefense, magicDefense, accuracy, avoidability, speed, jump}, atlas-effective-stats MUST emit one `stat.Bonus` per non-zero stat with `source = fmt.Sprintf("equipment:%d", asset.Id)`, the matching `stat.Type`, and `amount = int32(equipData.<Stat>)`. (This is the existing logic in `fetchEquipmentBonuses` — it just needs accurate input.)

`Computed.maxHp` MUST equal `Base.maxHp + sum(equipment HP bonuses) + sum(buff HP bonuses) + sum(passive HP bonuses)` after the fix.

### 4.4 Three viable remediations

The task author should pick one. Recommendation: **option A**, as it preserves the JSON:API consumer pattern used by every other atlas service.

**Option A — full hydration via api2go's `included` resolver.** Make `CompartmentRestModel` participate in the included-resource hydration phase in addition to ID resolution. With jtumidanski/api2go, this typically means implementing one of the additional unmarshal hooks (e.g. `SetToManyReferences(name string, references []jsonapi.MarshalIdentifier)` or whichever the fork exposes) to receive the *fully-decoded* assets from `included`, then writing them into `r.Assets` instead of stubs. Verify by adding a unit test on `CompartmentRestModel` that round-trips a JSON:API document with one compartment + one asset in `included` and asserts `compartment.Assets[0].Slot`, `.Hp`, `.Mp` are non-zero.

**Option B — separate per-asset fetches.** After `RequestEquipCompartment` returns the stub asset list, fan out one `GET /api/characters/{id}/inventory/assets/{assetId}` call per stub to hydrate each asset. Lower risk in isolation (no library archeology) but adds N+1 latency proportional to inventory size and breaks if atlas-inventory removes the per-asset endpoint.

**Option C — switch to a flat endpoint.** Replace `RequestEquipCompartment` with a call against `/api/characters/{id}/inventory` which already returns assets denormalized in `included` (verified during diagnosis — atlas-ui's `useInventory` consumes this shape today). Trade-off: pulls *all* compartments, not just equip, so the service does some work it doesn't strictly need.

### 4.5 Test coverage

- New unit test: `CompartmentRestModel` JSON:API round-trip with a compartment + assets in `included`, asserting full hydration of every numeric attribute. (Option A only.)
- New integration test in atlas-effective-stats's character package: with a stub atlas-inventory returning one compartment whose `included` contains an asset at slot -1 with hp=100, calling `GetEffectiveStats` should produce a `Computed.maxHp` of `Base.maxHp + 100` and a `Bonus` with `source == "equipment:<id>"`, `statType == TypeMaxHp`, `amount == 100`.
- Smoke test post-deploy: re-run the curl from §4.1 against a real character with an HP-bonus piece (e.g. character 12 on the dev cluster) and verify the response now includes the equipment bonus.

## 5. API Surface

No request/response shape changes. The existing `GET /api/worlds/.../stats` response remains:

```json
{
  "data": {
    "type": "effective-stats",
    "id": "12",
    "attributes": {
      "strength": 4, "dexterity": 25, "luck": 57, "intelligence": 4,
      "maxHP": 447, "maxMP": 240,
      "weaponAttack": 0, "weaponDefense": 0, "magicAttack": 0, "magicDefense": 0,
      "accuracy": 0, "avoidability": 0, "speed": 20, "jump": 0,
      "bonuses": [...]
    }
  }
}
```

After the fix, for character 12 the `bonuses` array additionally contains entries derived from each equipped asset's flat stats; `maxHP` and `maxMP` are correspondingly larger. No consumer code changes — the atlas-ui `useCharacterEffectiveStats` hook + `AttributesPanel` already render `+bonus` when present.

## 6. Data Model

No schema changes. No persistence layer involvement.

## 7. Service Impact

### services/atlas-effective-stats (primary, only)

- `external/inventory/rest.go` — change `CompartmentRestModel`'s JSON:API hooks so `Assets` is hydrated with full per-asset attributes (Option A) OR change `requests.go::RequestEquipCompartment` to a flat-fetch shape (Option C) OR add a hydration step in `initializer.go::fetchEquipmentBonuses` (Option B).
- `character/initializer.go` — no logical change; the existing `fetchEquipmentBonuses` body is correct, it just needs non-stub input.
- Tests: see §4.5.

### Other services

Not affected:
- atlas-inventory — no changes; today's response shape is fine, the bug is in how the consumer parses it.
- atlas-character — not touched.
- atlas-ui (task-027) — no changes; the frontend already renders bonuses correctly when the backend emits them. The `AttributesPanel.tsx` `+bonus` rendering and the HP/MP bar `bonus` prop will start showing values automatically after the backend fix deploys.

## 8. Non-Functional Requirements

- **Performance:** Option A and C should be a wash vs. today (one HTTP call). Option B adds N requests per character page load — to be avoided unless it's the only viable path.
- **Backward compat:** Existing consumers parse `data.attributes` directly; they don't care whether `bonuses` is empty or populated. No migration needed.
- **Multi-tenancy:** Tenant headers already propagate through `requests.GetRequest` via `TenantHeaderDecorator` — verified today (calling atlas-inventory without tenant headers returns 400; atlas-effective-stats's request must therefore include them).
- **Logging:** `fetchEquipmentBonuses` already logs `"Fetched %d equipment bonuses for character [%d]."` at debug level. After the fix this number should be > 0 for characters with stat-bearing equipment — useful for triage.

## 9. Open Questions

- (For the design phase) Which of the three remediation options does the project want? Recommendation in §4.4 is Option A. If jtumidanski/api2go's hydration hooks aren't sufficient for full-struct included resolution, fall back to Option C (single denormalized inventory fetch) over Option B (N+1 round-trips).
- Are other atlas services affected by the same `SetToManyReferenceIDs`-only pattern? A grep for `SetToManyReferenceIDs` across `services/` will surface any consumer that may also be silently dropping included-resource attributes; out of scope here but worth a follow-up audit.

## 10. Acceptance Criteria

- [ ] `GET /api/worlds/0/channels/0/characters/12/stats` (or any character with HP-bonus equipment) returns a `bonuses` array containing at least one entry with `source` matching `^equipment:\d+$` and a non-zero `amount`.
- [ ] `Computed.maxHp` in the same response is strictly greater than `character.attributes.maxHp` for that character (where the character has HP-bonus equipment).
- [ ] Symmetric assertion for MaxMp / Strength / Dexterity / Intelligence / Luck / WeaponAttack / etc. when equipment grants those stats.
- [ ] New unit test exists for the JSON:API round-trip of `CompartmentRestModel` (Option A) — fails on `main` before the fix, passes after.
- [ ] New integration test exists in `services/atlas-effective-stats/atlas.com/effective-stats/character/` covering the `fetchBaseStats` + `fetchEquipmentBonuses` chain with a stub inventory response.
- [ ] `go build ./...` and `go test ./...` pass in `services/atlas-effective-stats/atlas.com/effective-stats`.
- [ ] Smoke verified against the dev cluster: character 12's effective-stats response includes `equipment:<id>` bonuses for the Medal granting `+47 HP / +50 MP`.
- [ ] No frontend changes required (atlas-ui task-027 work already renders `+bonus` automatically once the backend reports it).

## 11. Diagnosis Artifacts (for the fix author)

For posterity, the chain that pinpointed this:

1. atlas-ingress access logs showed exactly one browser request: `GET /api/worlds/0/channels/0/characters/12/stats → 200`. Hook is firing.
2. Direct curl of the endpoint returned `bonuses: [{source: "passive:1002", statType: "speed", amount: 20}]` — only the passive skill, no equipment.
3. Direct curl of atlas-inventory `/api/characters/12/inventory/compartments?type=1&include=assets` (with tenant headers) returned 6 equipped assets with one of them (Medal at slot -49) carrying `hp: 47, mp: 50`. Inventory data is correct.
4. Reading `external/inventory/rest.go` → only `SetToManyReferenceIDs` is implemented on `CompartmentRestModel`, capturing IDs only.
5. Reading `character/initializer.go::fetchEquipmentBonuses` → loop guard `IsEquipped()` checks `Slot < 0`. With stub assets having `Slot == 0`, the loop is dead.
