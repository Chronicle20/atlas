# Backend Guidelines Audit — task-139 merge (main → task-139-pet-auto-pot-validation) + follow-ups

- **Scope:** `libs/atlas-packet/resolve.go`+test, `libs/atlas-packet/model/asset.go` (`encodePetCashItemInfo`, `resolvePetSkillWireMask`), `services/atlas-channel/.../consumable/processor.go`+mock, `services/atlas-channel/.../socket/handler/pet_item_use.go`, `character_skill_use.go`, `enable_actions.go`. Diffed against `8a9f70301` (main tip merged in) and `c97ccf6f2` (pre-merge branch tip).
- **Date:** 2026-08-07
- **Mindset:** default FAIL until file:line evidence shows otherwise.

## Build & Test Results

```
cd libs/atlas-packet && go build ./...     -> PASS (no output)
cd libs/atlas-packet && go vet ./...       -> PASS (exit 0)
cd libs/atlas-packet && go test -race ./... -> PASS (all packages ok, including
   ./cash/serverbound, ./pet/serverbound, ./model)

cd services/atlas-channel/atlas.com/channel && go build ./... -> PASS
cd services/atlas-channel/atlas.com/channel && go vet ./...   -> PASS (exit 0)
cd services/atlas-channel/atlas.com/channel && go test -race ./... -> PASS
   (105 `ok` package results, zero FAIL lines; full log at /tmp/channel_test_out.txt)
```

`tools/goroutine-guard.sh` clean for both trees — no bare `go` statements were introduced by this diff (verified by grep of every changed file plus a full guard run; only pre-existing lib exemptions reported).

## DOM-25 — Client wire values are config-resolved, never hardcoded

**Verdict: PASS**, with one accuracy note on the resolver contract split.

- `libs/atlas-packet/resolve.go:98-140` — `ResolveCode16` is a new **soft** uint16 resolver: on any miss (missing property, missing key, non-map, unparseable/out-of-range value) it returns `(0, false)` and logs at `Debug`, never a guessed value. Confirmed by `resolve_test.go:160-202` (`TestResolveCode16`), which table-drives every miss path (absent key, absent property, unparseable string, out-of-range float64, non-map property) and asserts `ok == false` in each case.
- `libs/atlas-packet/resolve.go:142-181` — `ResolveValue` is the new **loud** uint32 resolver (mirrors `ResolveCode`'s "log Error, return zero-value+false" contract, not the byte sentinel `99` since there is no safe uint32 sentinel). Confirmed by `resolve_test.go:222-241` (`TestResolveValueMisses`), table-driving 5 miss cases, all asserting `ok == false`.
- The two original resolvers (`ResolveCode` at `resolve.go:29-62`, `ResolveName` at `resolve.go:70-96`) are untouched by this diff (`git diff 8a9f70301..HEAD -- libs/atlas-packet/resolve.go` shows only additive hunks at the end of the file) — their loud-99/soft-false contracts are intact and all 12 of their pre-existing tests still exercise the original behavior (`resolve_test.go:10-158`).
- The four resolvers are coherent as a family: `ResolveCode`/`ResolveName` are the mandatory-table byte pair (loud sentinel 99), `ResolveCode16`/`ResolveValue` are optional-table pairs at wider widths with no safe sentinel (soft `false`). No behavioral overlap or contract collision between them — they key off disjoint call sites (`ResolveCode16` only used by `resolvePetSkillWireMask`, `ResolveValue` unused within this diff's file set but exported for wider adoption per its doc comment).
- `libs/atlas-packet/model/asset.go:589-608` (`resolvePetSkillWireMask`) — translates the Atlas-canonical `petFlag uint16` into the tenant's wire `usPetSkill` bits purely through `atlas_packet.ResolveCode16(l, options, "petSkill", string(k))` (line 601); a bit with no table entry logs `Debug` and is silently dropped from the wire mask (line 603), never guessed. No client wire byte is a Go literal in this function.
- Fail-closed on the *mandatory* gate: `services/atlas-channel/.../socket/handler/pet_item_use.go:67-72` — `gate, _ := readerOptions["skillGate"].(string); if gate != skillGateEquipAbility && gate != skillGatePetSkillFlag { reject("skill_gate_unconfigured"); return }`. `skillGateEquipAbility`/`skillGatePetSkillFlag` (`pet_item_use.go:32-34`) are semantic config-key strings compared against the resolved tenant option, not client wire bytes — this matches the documented "domain services emit semantic keys" pattern, not a DOM-25 violation.
- Per-version seed template coverage confirmed for every template that binds `PetItemUseHandle` (all 10: `template_gms_{48,61,72,79,83,84,87,92,95}_1.json`, `template_jms_185_1.json`) — each carries a `skillGate` handler option (`equipAbility` for all 9 GMS templates, `petSkillFlag` for jms) and a `petSkill` writer table on both `CharacterInventoryChange` and `SetField` (GMS templates: `{autoSpeaking}`; jms: `{autoSpeaking, consumeHP, consumeMP}`). The narrower GMS table is consistent with the design (GMS gates on worn-equip ability, not the persisted flag, so `consumeHP`/`consumeMP` wire bits are never needed there) — not a completeness gap.
- Rollout-checklist requirement satisfied: `docs/tasks/task-139-pet-auto-pot-validation/` contains `design.md`, `plan.md`, `context.md` documenting the new table and its per-version seeding; the two most recent commits on the branch (`f7fc16631`, `1bc731215`) are explicit fixes wiring `equipAbility`/`petSkill` into `gms_48`/`gms_92` templates that had been missed on a first pass — evidence the rollout was tracked and iterated, not silently assumed.

**Encode-side version gate** — `libs/atlas-packet/model/asset.go:364-380` (`encodePetCashItemInfo`) gates `remainLife`/trailing `attribute` writes on `t.MajorAtLeast(N)`/`t.Region()`, which are tenant-identity checks (not client wire bytes), so they are outside DOM-25's scope; the gate itself is IDA-cited in the inline comment (v48/v61 @0x49c77e/@0x4b52f2, v72 @0x4d06dd, v79/v83 @0x4d84c4/@0x4e4219) and pinned by `libs/atlas-packet/model/asset_test.go:378-435` (`TestAssetPetCashItemTrailerVersionGate`), which asserts byte-length deltas per version.

## DOM-21 — Reuse atlas-constants, no redefinition

**Verdict: PASS.**

- `libs/atlas-constants/pet/skill/*.go` (added by prior commits `d9084c748`, `c65a74464`, not part of this diff) defines `skill.Key`, `skill.Flag`, `skill.All()`, `skill.Has()`, `skill.Apply()` as the canonical pet-skill bit semantics. This diff's `resolvePetSkillWireMask` (`asset.go:589`) and `pet_item_use.go` (`resolveSkillSources`, `matchPetAbilityEquips`) consume that package directly (`petskill "github.com/Chronicle20/atlas/libs/atlas-constants/pet/skill"`, `asset.go:16`; `pet_item_use.go:21`) rather than redefining bit constants locally.
- `services/atlas-channel/.../character_cash_item_use.go:846-848` — the diff *replaces* a bare numeric literal (`category == 519`) with `category == item.ClassificationPetSkill`, confirmed to equal `Classification(519)` at `libs/atlas-constants/item/constants.go:87`. This is a DOM-21 improvement, not a new violation.
- No new domain type, alias, or numeric constant was found in the reviewed files that duplicates an existing `libs/atlas-constants/` type (checked pet id, item classification, inventory type, skill flag namespaces).

## Consumable Processor asymmetry — `quantity` on `RequestItemConsume` vs `RequestItemConsumeWithPet`

**Verdict: justified design choice, not a defect.**

- `consumable/processor.go:18-19` — `RequestItemConsume(..., quantity int16, ...)` vs `RequestItemConsumeWithPet(..., petId uint64)` (no quantity param; hardcodes `1` at line 58: `RequestItemConsumeWithPetCommandProvider(f, characterId, source, itemId, 1, petId)`).
- The asymmetry tracks a real wire difference, not an oversight: `libs/atlas-packet/cash/serverbound/item_use_pet_skill.go:35-37` — the type-28 (`0519` pet skill pouch) sub-body decodes *only* `petId uint64`; there is no wire quantity field for this path (cash-shop items are used one at a time via `CashItemUse`, unlike stackable-slot consumables that carry an `itemConNo`/quantity field decoded elsewhere). `processor.go:52-55` documents this explicitly: "the auto-pot path deliberately does NOT use [`RequestItemConsumeWithPet`]: its pet validation happens at the socket handler and nothing downstream needs the pet" — and the auto-pot handler (`pet_item_use.go:160`) calls the *plain* `RequestItemConsume` with a literal `1`, while `RequestItemConsumeWithPet` is reserved for the case-28 cash pouch path (`character_cash_item_use.go:76`) where the wire genuinely carries no quantity to forward.
- Since no call site can ever supply a meaningful non-1 quantity for the pet-pouch path (the wire has no such field), a `quantity int16` parameter on `RequestItemConsumeWithPet` would be dead weight, not a safety net. **Not a finding.**

## Parallel-fetch fail-closed audit — `pet_item_use.go`

**Verdict: PASS — the barrier and per-branch variable ownership are provably race-free**, with one non-blocking design-intent note.

- `pet_item_use.go:86-116` — `model.NewGroup(ctx)` / `model.Submit` / `pg.Wait()` is backed by `errgroup.Group` (`libs/atlas-model/model/parallel_group.go:31-54`, unmodified by this diff): `Submit` closures always `return nil, nil` regardless of the captured provider's own error (`parallel_group.go:41-48`), so the captured `pmErr`/`spawnedErr`/`cErr`/`ciErr` variables — not `pg.Wait()`'s return value — carry the real failure signal, and `pg.Wait()` (`errgroup` → `sync.WaitGroup.Wait`) is a genuine happens-before barrier guaranteeing every closure's writes are visible before the handler reads them at `pet_item_use.go:119-145`.
- No data race: each captured variable is written by exactly one goroutine. `pm`/`pmErr` are written only inside the `hasPetId` branch's `Submit` (`pet_item_use.go:98-101`); `spawnedPets`/`spawnedErr` only inside the `!hasPetId` branch's `Submit` (`pet_item_use.go:103-106`) — the two branches are mutually exclusive (`if hasPetId {...} else {...}`, lines 97-107), so only one of the two ever executes, and `c`/`cErr` (line 108-111) and `ci`/`ciErr` (line 112-115) are each written by their own single dedicated goroutine. `go test -race` passed clean on the whole `atlas-channel` module (see Build & Test Results), consistent with this analysis, though no test in `pet_item_use_test.go` currently drives the full handler end-to-end under `-race` (the test file only unit-tests the pure helpers — see the DOM-24 note below).
- `hasPetId`/`petId==0` branch is fail-closed on both arms: `hasPetId` true → `pmErr != nil` rejects with `pet_not_found` (`pet_item_use.go:120-123`); `hasPetId` false → `spawnedErr != nil` rejects with `pet_not_found` (`pet_item_use.go:125-128`), and if the fetch itself succeeded but yields no eligible pet, `resolveSpawnedPet` (`pet_item_use.go:169-176`) returns `("", false)` unless it finds a slot `>= 0`, which is also rejected (line 132-135). `ciErr`/`cErr` are each checked afterward and reject (`not_consumable` / `fetch_failed`, lines 137-145). Every exit path from the fetch block either populates a valid `pm`/`c`/`ci` or returns via `reject(...)` before those values are used — there is no path where an error is silently ignored and stale/zero-value data flows into `evaluateAutoPot`.
- **Design-intent note (not a guideline violation):** `hasPetId := p.PetId() != 0` (`pet_item_use.go:80`) is applied uniformly regardless of whether the client version's wire format even carries a `petId` field. The inline comment at lines 74-79 states resolution must "never fall back from one to the other" between the wire-petId and spawned-pet paths, but the code's `!= 0` test cannot distinguish "version has no wire petId field" from "version has the field and the client sent literal 0." On a version where `hasLeadingPetId(t)` is true (`libs/atlas-packet/pet/serverbound/item_use.go:57`), a client sending `petId=0` — which per the same file's comment (`pet_item_use.go:61`, "a real Atlas pet id is never 0") should never happen legitimately — would still be routed into the spawned-pet fallback rather than being rejected outright as `pet_not_found`. This is not exploitable to target another player's pet: `resolveSpawnedPet` only ever returns the *calling character's own* spawned pet (`pet.NewProcessor(...).GetByOwner(s.CharacterId())`, line 104), and `evaluateAutoPot`'s ownership check (`petOwnerId != characterId`, line 203) still gates the result. Flagging only because it contradicts the comment's own stated invariant ("never fall back from one to the other") — worth a version-gated `hasPetId := hasLeadingPetId(t) && p.PetId() != 0` tightening if the team wants the code to match the comment precisely, but it is not a fail-closed break.

## File Responsibilities Checklist

| ID | Package | Check | Status | Evidence |
|----|---------|-------|--------|----------|
| FILE-01 | `consumable` | Processor interface+impl in `processor.go` | PASS | `consumable/processor.go:17-84` — interface, `NewProcessor`, and all `ProcessorImpl` methods (incl. new `RequestItemConsumeWithPet`) live in `processor.go`; none found in `producer.go`/`mock/processor.go` other than the mock's own trivial forwarders. |
| FILE-03 | `consumable` | Kafka producer funcs in `producer.go` | PASS | `consumable/producer.go:31-49` — new `RequestItemConsumeWithPetCommandProvider` lives in `producer.go` alongside its siblings, not in `processor.go`. |
| FILE-06 | `consumable` | No package-named catch-all file | PASS | `consumable/` has only `processor.go`, `producer.go`, `producer_test.go`, `mock/` — no `consumable.go` collapsing multiple responsibilities. |
| SUB-04 | `socket/handler` (pet_item_use, enable_actions) | No manual JSON parsing | PASS | `pet_item_use.go` and `enable_actions.go` contain no `json.Unmarshal`/`json.NewDecoder`/`io.ReadAll` (grep clean). |

## Test Coverage Gap (non-blocking)

- `consumable/producer_test.go` was **not** modified by this diff (`git diff 8a9f70301..HEAD -- consumable/producer_test.go` is empty) and only covers `RequestItemConsumeCommandProvider` (`TestRequestItemConsumeCommandProvider_CarriesQuantity`, lines 20-45). The new `RequestItemConsumeWithPetCommandProvider` (`producer.go:31-49`) and the new `PetId` field on `RequestItemConsumeBody` (`kafka/message/consumable/kafka.go:39`) have no direct unit test asserting the emitted Kafka message actually carries `petId` on the wire. This is a coverage gap against the testing-guide's expectation that new producer logic gets a table-driven/round-trip test, not a build-breaking defect — the behavior is exercised transitively through `pet_item_use_test.go`'s coverage of the handler's pure helper functions, but nothing pins the JSON `petId` field end-to-end the way `TestRequestItemConsumeCommandProvider_CarriesQuantity` pins `quantity`.

## Summary

### Blocking (must fix)
- None. Build, vet, and race tests are clean across both modules; no DOM-25/DOM-21 violations found; the parallel-fetch fail-closed contract is intact.

### Non-Blocking (should fix)
- Add a unit test for `RequestItemConsumeWithPetCommandProvider` mirroring `TestRequestItemConsumeCommandProvider_CarriesQuantity`, asserting the emitted command's `PetId` field round-trips through JSON.
- Consider tightening `hasPetId := p.PetId() != 0` (`pet_item_use.go:80`) to also check `hasLeadingPetId(t)` so the code enforces, rather than merely documents, "never fall back from one to the other" between the wire-petId and spawned-pet resolution paths — currently a benign but comment-contradicting edge case, not exploitable given the downstream ownership check.
