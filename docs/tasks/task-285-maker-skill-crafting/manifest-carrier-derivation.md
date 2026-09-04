# Manifest carrier derivation (prerequisite for Task 26)

Branch `task-285-maker-skill-crafting`, HEAD `331b0c0b5`. Every claim below is quoted from
repo source at that commit.

## 0. The question

Task 26 writes `MAKER_RESULT`. Its CREATE / CREATE_WITH_UPGRADE / MONSTER_CRYSTAL /
DISASSEMBLE arms enumerate what the craft **actually consumed and produced**. Nothing on this
branch carries that from `atlas-maker` to `atlas-channel`:

- `atlas-maker`'s craft REST resource returns the transaction id only
  (`services/atlas-maker/atlas.com/maker/craft/rest.go`; `Processor.Create` returns
  `(uuid.UUID, error)` — `craft/processor.go:104`).
- `saga/producer.go`'s `CompletedStatusEventProvider` (`:22`) has arms only for
  `CharacterCreation`, `NoteSend` and MTS take-home. No maker arm exists.
- `craft.Plan.Consumptions` is a flat `[]Consumption` with fields
  `{InventoryType, Slot, Quantity, TemplateId}` and **no role discriminator**
  (`craft/plan.go:35-50`).

The transport is settled by user ruling: the manifest rides the **saga record**, `atlas-maker`
attaches it, `atlas-saga-orchestrator` echoes it into `StatusEventCompletedBody.Results`,
`atlas-channel` decodes it. This document derives the *concrete carrier* inside that ruling.

## 1. What the wire actually needs

From `libs/atlas-packet/character/maker_result_body.go` and
`libs/atlas-packet/character/clientbound/maker_result.go:43-92`:

| Arm | Body function (`maker_result_body.go`) | Parameters |
|---|---|---|
| CREATE | `MakerResultCreateBody` (`:26`) | `result, noItemAwarded, targetItemId, itemNum, materials []MakerMaterial, gemItemIds []uint32, catalystUsed, catalystItemId, mesoCost` |
| CREATE_WITH_UPGRADE | `MakerResultCreateWithUpgradeBody` (`:33`) | identical to CREATE |
| MONSTER_CRYSTAL | `MakerResultMonsterCrystalBody` (`:40`) | `result, crystalItemId, leftoverItemId` |
| DISASSEMBLE | `MakerResultDisassembleBody` (`:47`) | `result, disassembledItemId, crystals []MakerMaterial, mesoCost` |
| FAILED | `MakerResultFailedBody` (`:67`) | `result` only — no manifest |

`MakerMaterial` is `(itemId uint32, count uint32)` (`clientbound/maker_result.go:47-54`), and
the create body writes `nNumUsedItem` followed by one `(itemId, count)` pair per entry
(`:78-82`). The manifest must therefore be **aggregated per item id**, not per inventory slot
— see F4 below.

## 2. Per-mode step inventory

Derived by reading `services/atlas-maker/atlas.com/maker/craft/processor.go` end to end. The
craft saga is built with `saga.NewBuilder().SetSagaType(saga.InventoryTransaction).SetInitiatedBy("MAKER_SKILL")`
in all four modes (`:165-167`, `:248-250`, `:337-339`).

| Mode | Builder | Step ids, in order | Action per step | Payload |
|---|---|---|---|---|
| 1 CREATE | `createOrUpgrade` (`:141-204`) | `deduct_meso` | `AwardMesos` | `AwardMesosPayload{Amount: -int32(r.Meso())}` (`:169-176`) |
| | | `destroy_0` … `destroy_N-1` | `DestroyAssetFromSlot` | one per `Plan.Consumptions` entry (`appendDestroySteps`, `:375-385`) |
| | | `award_item` | `AwardCraftedAsset` **if** `inventory.TypeFromItemId(r.Id())` is Equip, else `AwardAsset` | `AwardCraftedAssetPayload` (`:182-188`) or `AwardItemActionPayload` (`:190-194`) |
| 2 CREATE_WITH_UPGRADE | `createOrUpgrade` (same function) | **byte-identical to mode 1** | same | same |
| 3 MONSTER_CRYSTAL | `crystal` (`:209-276`) | `deduct_meso` | `AwardMesos` | `AwardMesosPayload{Amount: -int32(r.Meso())}` (`:252-259`) |
| | | `destroy_0` … `destroy_N-1` | `DestroyAssetFromSlot` | `BuildCrystalPlan` — the leftover at `LeftoverConsumeQuantity` = 100, spread across slots (`plan.go:28`, `:132-135`) |
| | | `award_crystal` | `AwardAsset` | `AwardItemActionPayload{reward.ItemId, reward.ItemNum}` from `Draw(r.RandomRewards())` (`:241-244`, `:263-267`) |
| 4 DISASSEMBLE | `disassemble` (`:283-371`) | `destroy_equip` | `DestroyAssetFromSlot` | the claimed equip at `req.SlotPos`, quantity 1 (`:341-347`) |
| | | `award_crystal` | `AwardAsset` | `AwardItemActionPayload{crystalId, count}` from `cbp.CrystalForLevel(eq.ReqLevel())` (`:317`, `:349-353`) |
| | | `charge_meso` | `AwardMesos` | `AwardMesosPayload{Amount: -int32(DisassembleMesoCharge)}` = `-0` (`:355-362`) |

Five facts fall out, and they decide the carrier:

- **F1 — `AwardCraftedAsset` is not universal.** Mode 3 and mode 4 never emit it, and mode 1/2
  emit it only when the produced item resolves to `inventory.TypeValueEquip` (`:180-195`).
- **F2 — mode 1 and mode 2 are indistinguishable from the saga.** `createOrUpgrade` handles
  both and never branches on `req.Mode`; the only appearance of `req.Mode` in that function is
  the log field at `:199`. So `nMode` — which selects the CREATE vs CREATE_WITH_UPGRADE arm —
  **cannot be derived from the step list at all**. Any carrier must state the mode explicitly.
- **F3 — mode 3's produced crystal is a random draw** made inside `atlas-maker` (`:241`). It is
  recoverable from the `award_crystal` payload, but see F2: a carrier is required regardless.
- **F4 — destroy steps are per-slot, the wire is per-item.** `resolveConsumption`
  (`plan.go:60-84`) emits one `Consumption` per slot touched, so a material held across two
  slots yields two `DestroyAssetFromSlot` steps. Feeding those straight into `nNumUsedItem`
  would render the same item twice in the client's consumption log. The manifest must
  aggregate by `TemplateId`.
- **F5 — the gem list on the wire is the *applied* set, not the requested set.** `BuildCreatePlan`
  drops an unheld gem silently rather than rejecting (`plan.go:113-124`, via
  `resolveConsumption`'s shortfall handling), and `AppliedGems` (`plan.go:141-153`) is the
  dedup-and-hold-filtered subset. FR-3.2's "dropped entirely, not rejected" is exactly this.
  The manifest's `gemItemIds` must equal `AppliedGems`, never `req.GemItemIds`.

## 3. The precedent the orchestrator already sets

All three existing extractors in `saga/producer.go` reconstruct `Results` from **step payloads
already on the saga** and add no new transport:

| Extractor | Line | Reads |
|---|---|---|
| `extractCharacterCreationResults` | `:59-73` | the `CreateCharacter` step's `CharacterCreatePayload` + that step's `Result()` |
| `extractNoteSendResults` | `:79-89` | the `CreateNote` step's `CreateNotePayload.SenderId` |
| `extractMtsTakeHomeResults` | `:106-129` | discriminates on the presence of a `ReleaseFromMtsHolding` step, then reads `AcceptToCharacterPayload` |

`extractMtsTakeHomeResults` is the closest template: it stamps a `Results["kind"]` marker
(`MtsTakeHomeResultKind`, `producer.go:96`) because its saga type (`mts_operation`) is shared
with three other operations. The craft saga has the same problem, worse: its saga type is the
generic `InventoryTransaction`. So a `kind` marker is mandatory.

The saga record itself has no metadata slot — `saga.Saga` is
`{TransactionId, SagaType, InitiatedBy, Timeout, Steps}` (`libs/atlas-saga/builder.go:66-79`).
The manifest must therefore live on a **step payload**.

## 4. Options weighed

### Option A — widen `DestroyAssetFromSlotPayload` with a role field

`libs/atlas-saga/payloads.go:135-143`. **Rejected.**

- `design.md:634` already rejected the analogous widening of `AwardAsset` for exactly this
  reason: "Widening its payload widens the blast radius of every one of those, for a field only
  maker sets."
- It does not solve the problem. Role alone gives materials/gems/catalyst, but F2 means the
  mode is still missing, and mode 3's `crystalItemId` and the create arm's `itemNum` are still
  missing. Option A would have to be *combined* with one of the others, so it buys nothing.
- Mode 4 needs no role at all (one destroy), so the field would be dead weight there.

### Option B — extend `AwardCraftedAssetPayload`

`libs/atlas-saga/payloads.go:1078-1099`; introduced by this branch for maker
(`design.md:497-510`), so it genuinely has no other consumers. **Rejected on F1.**

- Mode 3 (`crystal`) and mode 4 (`disassemble`) never emit an `AwardCraftedAsset` step.
- Even mode 1/2 skip it when the recipe's output is not an equip (`processor.go:180-195`) — a
  non-equip recipe would produce a CREATE arm with no manifest.
- The brief's own requirement is "works for all four modes including disassemble". Option B
  fails three of the four. This is the decisive loser.

### Option C — encode the role in the `stepId` string

`destroy_material_0` / `destroy_gem_0` / `destroy_catalyst` instead of `destroy_0..N`.
Attractive because it adds nothing to any payload. **Rejected.**

- F2 again: `stepId` cannot carry the mode, `itemNum`, or mode 3's drawn `crystalItemId`.
- `stepId` today is a free-form correlation key. Nothing in the orchestrator parses its
  contents; the compensator and event-matching switch on `Action()`
  (`compensator.go:512-533`, `event_acceptance.go`), never on the id's spelling. Making it
  semantic creates a cross-service string contract that no type and no existing guard protects.

### Option D — a maker-specific, self-completing step carrying the manifest whole

**Chosen.** A new `Action` `record_craft_manifest` with a `CraftManifestPayload`, added as the
saga's **first** step in every one of the four modes.

- Adds nothing to any payload shared outside this feature: `CraftManifestPayload` is new, and
  no other service will ever emit the action.
- Works for all four modes uniformly, including disassemble, because `atlas-maker` builds the
  step unconditionally rather than piggybacking on a mode-dependent step.
- The precedent is already in the tree: `IncubatorResult` (`libs/atlas-saga/model.go:312`,
  `payloads.go:1408-1409`) is a self-completing, data-only action —
  `acceptanceTable[sharedsaga.IncubatorResult] = {}` (`event_acceptance.go:161`) and
  `handleIncubatorResult` calls `StepCompleted` directly (`handler.go:1316-1336`), matching
  `ValidateCharacterState`'s empty acceptance entry (`event_acceptance.go:180`). Our handler is
  the same shape minus the produce: it only marks the step complete.
- It survives a pod restart, which is the ruling's whole point: the orchestrator persists the
  saga as `SagaData []byte` jsonb (`saga/entity.go:23`) and rehydrates it through
  `Step[T].UnmarshalJSON`.
- Compensation costs nothing. `CompensateFailedStep` switches on the **failed** step's action
  (`compensator.go:512`) and a self-completing step with no external event can never be the
  failed step; the per-saga-type reverse-walks switch on `step.Action()` with no `default`
  error arm (e.g. `DispatchMtsOperationRollbacks`, `compensator.go:2518-2545`), so an unmatched
  action is silently skipped. It must **not** be added to `lateCompensableActions`
  (`compensator.go:3089`).

### Internal corollary inside `atlas-maker`

To guarantee the manifest describes exactly what the destroy steps destroy, tag the role on the
service-local `craft.Consumption` (`plan.go:35-41`) — a maker-internal type, zero cross-service
blast radius — and build both the destroy steps and the manifest from the same `Plan`. Deriving
the manifest from `req` instead would re-introduce the client-trust bug the Plan exists to
prevent: `BuildCreatePlan` silently drops an unheld gem (F5) and a catalyst the character does
not hold (`plan.go:126-129` + `resolveConsumption`'s shortfall handling), so a
`req`-derived manifest would report consumption that never happened.

## 5. The shape at each hop

### Hop 1 — `atlas-maker` → saga step (`libs/atlas-saga`)

```go
// libs/atlas-saga/model.go — new Action constant
RecordCraftManifest Action = "record_craft_manifest"

// libs/atlas-saga/payloads.go
type CraftManifestItem struct {
    ItemId uint32 `json:"itemId"`
    Count  uint32 `json:"count"`
}

type CraftManifestPayload struct {
    CharacterId uint32 `json:"characterId"`
    Mode        uint32 `json:"mode"` // 1|2|3|4, MAKER_SKILL's nMode

    // Modes 1 and 2 (CREATE / CREATE_WITH_UPGRADE).
    NoItemAwarded  bool                `json:"noItemAwarded"`
    TargetItemId   uint32              `json:"targetItemId,omitempty"`
    ItemNum        uint32              `json:"itemNum,omitempty"`
    Materials      []CraftManifestItem `json:"materials,omitempty"`
    GemItemIds     []uint32            `json:"gemItemIds,omitempty"`
    CatalystUsed   bool                `json:"catalystUsed"`
    CatalystItemId uint32              `json:"catalystItemId,omitempty"`

    // Mode 3 (MONSTER_CRYSTAL).
    CrystalItemId  uint32 `json:"crystalItemId,omitempty"`
    LeftoverItemId uint32 `json:"leftoverItemId,omitempty"`

    // Mode 4 (DISASSEMBLE).
    DisassembledItemId uint32              `json:"disassembledItemId,omitempty"`
    Crystals           []CraftManifestItem `json:"crystals,omitempty"`

    // Modes 1, 2 and 4. A COST, always non-negative — the negation of the
    // AwardMesos step's Amount.
    MesoCost uint32 `json:"mesoCost"`
}
```

`MesoCost` and `CatalystUsed`/`NoItemAwarded` deliberately carry no `omitempty`: a zero cost
and a false flag are meaningful and must be distinguishable from an absent field, matching
`AwardCraftedAssetPayload.Slots`'s stated reason (`payloads.go:1076-1077`).

Per-mode population, each value taken from the same source the steps are built from:

| Field | Mode 1/2 | Mode 3 | Mode 4 |
|---|---|---|---|
| `CharacterId` | `characterId` | same | same |
| `Mode` | `uint32(req.Mode)` | `3` | `4` |
| `NoItemAwarded` | `false` | — | — |
| `TargetItemId` | `uint32(r.Id())` | — | — |
| `ItemNum` | `r.ItemNum()` | — | — |
| `Materials` | plan entries with role *material*, aggregated by `TemplateId` (F4) | — | — |
| `GemItemIds` | `AppliedGems(snap, req.GemItemIds)` (F5) | — | — |
| `CatalystUsed` / `CatalystItemId` | true + `r.Catalyst()` **only if** a catalyst consumption is actually in the plan | — | — |
| `CrystalItemId` | — | `reward.ItemId` from `Draw` (`processor.go:241`) | — |
| `LeftoverItemId` | — | `req.LeftoverItemId` | — |
| `DisassembledItemId` | — | — | `uint32(req.EquipItemId)` |
| `Crystals` | — | — | `[{crystalId, count}]` from `CrystalForLevel` (`processor.go:317`) |
| `MesoCost` | `r.Meso()` | `r.Meso()` | `DisassembleMesoCharge` (= `0`) |

Mode 3's arm takes no material list (§1), so `Materials` stays empty there even though the
saga destroys 100 leftovers — the client hard-codes the `100` in its own log line
(`plan.go:12-27`). This is recorded so nobody later "fixes" it by populating the field.

### Hop 2 — orchestrator → `StatusEventCompletedBody.Results`

A new extractor in `saga/producer.go`, mirroring `extractMtsTakeHomeResults` (`:106-129`):

```go
const MakerCraftResultKind = "maker_craft"

// discriminator: the presence of a RecordCraftManifest step
func extractMakerCraftResults(s Saga) map[string]any {
    for _, step := range s.Steps() {
        if step.Action() != RecordCraftManifest {
            continue
        }
        p, ok := step.Payload().(CraftManifestPayload)
        if !ok {
            return nil
        }
        return map[string]any{
            "kind":          MakerCraftResultKind,
            "characterId":   p.CharacterId,
            "craftManifest": p,
        }
    }
    return nil
}
```

Wired into `CompletedStatusEventProvider` (`producer.go:22-56`) as a fourth
`if r := …; r != nil { body.Results = r }` arm, the same shape the MTS arm uses at `:46-48`.

`characterId` is lifted to the top level as a scalar so the channel resolves the session with
the existing `resultUint32` helper (`consumer.go:200`) before it decodes anything nested — the
same layering the take-home branch uses (`consumer.go:104-107`).

A compensated or timed-out saga never reaches `CompletedStatusEventProvider`, so it emits no
manifest, by construction. That is asserted, not assumed — see the brief's test list.

### Hop 3 — Kafka JSON round-trip → `atlas-channel` decode

`StatusEventCompletedBody.Results` is `map[string]any`
(`services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka.go:57-60`). A nested struct
value marshals to a nested JSON object and comes back as `map[string]any` whose lists are
`[]any` of `map[string]any` and whose numbers are `float64` — the existing
`TestResultDecoders_TolerateJSONFloat64` (`consumer_test.go:13-27`) proves the float64 part.

The scalar helpers `resultKind`/`resultUint32` therefore **cannot** read the nested list shape,
and no flattening of `map[string]any` can express `[]MakerMaterial`. The shape does survive the
round trip, but only through a re-marshal:

```go
// atlas-channel, Task 26
func decodeCraftManifest(results map[string]any) (saga.CraftManifestPayload, bool) {
    var out saga.CraftManifestPayload
    if results == nil {
        return out, false
    }
    raw, ok := results["craftManifest"]
    if !ok {
        return out, false
    }
    bs, err := json.Marshal(raw)
    if err != nil {
        return out, false
    }
    if err := json.Unmarshal(bs, &out); err != nil {
        return out, false
    }
    return out, true
}
```

This is safe because both ends bind to the **same** `libs/atlas-saga` struct and its JSON tags;
`atlas-channel` already aliases shared saga payloads
(`services/atlas-channel/atlas.com/channel/saga/model.go:34,131`), so Task 26 adds
`CraftManifestPayload = sharedsaga.CraftManifestPayload` there and nothing else.

Channel-side branch in `handleCompletedEvent` (`consumer.go:87`), matching the take-home idiom
at `:104-121`: the craft saga's `SagaType` is the generic `inventory_transaction`, so the
branch keys on `resultKind(e.Body.Results) == saga.MakerCraftResultKind`, **not** on
`e.Body.SagaType`.

## 6. Wiring inventory (what Task 26a must touch)

| File | Change | Existing precedent |
|---|---|---|
| `libs/atlas-saga/model.go` | `RecordCraftManifest` action constant | `IncubatorResult` at `:312` |
| `libs/atlas-saga/payloads.go` | `CraftManifestPayload`, `CraftManifestItem` | `IncubatorResultPayload` at `:1408` |
| `libs/atlas-saga/unmarshal.go` | `Step[T].UnmarshalJSON` case | `case IncubatorResult` at `:713-718` |
| `…/saga-orchestrator/saga/model.go` | action alias + payload alias + its own `UnmarshalJSON` case | aliases at `:218`/`:287`; case at `:1813-1818` |
| `…/saga-orchestrator/saga/event_acceptance.go` | `acceptanceTable` entry, empty slice (self-completing) | `sharedsaga.IncubatorResult: {}` at `:161` |
| `…/saga-orchestrator/saga/event_acceptance_test.go` | append to `allActions` | list at `:14` |
| `…/saga-orchestrator/saga/handler.go` | interface method, `GetHandler` case, `handleRecordCraftManifest` | `handleIncubatorResult` at `:1316-1336`, minus the produce |
| `…/saga-orchestrator/saga/producer.go` | `MakerCraftResultKind` + `extractMakerCraftResults` + the arm in `CompletedStatusEventProvider` | `:96`, `:106-129`, `:46-48` |
| `services/atlas-maker/.../craft/plan.go` | role tag on `Consumption`; roles set in `BuildCreatePlan`/`BuildCrystalPlan` | — |
| `services/atlas-maker/.../craft/processor.go` | build + prepend the manifest step in all four modes | `appendDestroySteps` at `:375-385` |
| `services/atlas-maker/.../craft/processor_test.go` | the `CompensableActions` assertion at `:464` must exempt the manifest action | — |

`…/saga-orchestrator/saga/rest.go`'s `payloadUnmarshalers` (`:75-85`) needs **no** entry: an
unregistered action returns the raw payload rather than erroring (`unmarshalPayload`, `:97-104`),
and craft sagas arrive over Kafka (`craft/emitter.go:28-32`), not REST. Registering one anyway
would be harmless but is not required; leaving it out means a REST-submitted craft saga would
carry an untyped payload and the extractor's type assertion would fail closed (no manifest),
which is the correct failure mode.

Two guard tests will fail loudly if any of the above is skipped:
`TestStepUnmarshal_EveryActionRepresented` (`unmarshal_completeness_test.go:24`) and the
`acceptanceTable` coverage test (`event_acceptance_test.go:69-73`).

## 7. Unresolved / surfaced

- **U1 — a failed craft saga emits `characterId` 0, so Task 26's FAILED arm has no session to
  write to.** `EmitSagaFailed` (`producer.go:181-206`) has arms for `MtsOperation`,
  `MesoSackUse` and `PetNameTagUse`; everything else falls through to
  `ExtractCharacterCreationIds`, which returns `(0, 0)` for a saga with no `CreateCharacter`
  step (`:140-155`). The craft saga's type is `InventoryTransaction`. This is a real gap in the
  FAILED path, it is *not* fixed by the manifest, and the user's ruling explicitly scoped this
  task to the completed arms. **Needs a ruling before Task 26 can write a routable FAILED arm.**
- **U2 — Task 26's brief is stale on the in-flight guard.** `.superpowers/sdd/plan/task-26-brief.md`
  says the channel consumer "must also release Task 23's in-flight guard". It does not: the
  guard lives in `atlas-maker` and is released by `atlas-maker`'s own terminal-event consumer
  (`services/atlas-maker/atlas.com/maker/kafka/consumer/saga/consumer.go:59` and `:74`, calling
  `craft.ReleaseInFlightByTransaction`), landed in commits `974cf0257` / `6f5615c20`. Task 26
  should drop that requirement.
- **U3 — `noItemAwarded` has no producer.** The wire supports a create result that awards
  nothing (`clientbound/maker_result.go:73-77`), but no path in `processor.go` builds a craft
  that completes without an award step. The manifest carries `false` in every case reachable
  today. Whether a legitimate award-nothing completion exists is unestablished from source.
- **U4 — the fact block's delta is empty.** `tools/task-facts.sh task-285 --base 331b0c0b5` was
  run with `base` equal to `HEAD`, so every `changed_*` field reports `none`/`0`. That is the
  literal tool output, not a claim that the branch changed nothing; the branch-point base used
  by earlier briefs was `9cd1ec5af`.
- **U5 — mode 3 leftover count is not on the wire.** The MONSTER_CRYSTAL arm takes no material
  list, so the 100 consumed leftovers (`plan.go:28`) are never enumerated to the client; the
  client hard-codes `100` in its own chat line. Recorded so it is not mistaken for a bug.
