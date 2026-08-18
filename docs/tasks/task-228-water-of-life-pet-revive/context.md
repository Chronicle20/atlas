# Water of Life (Pet Revive) — Implementation Context

Companion to [`plan.md`](plan.md). Read this first if you are picking the task up cold: it collects the file locations, the load-bearing decisions and their evidence, and the traps that a reasonable reading of the codebase would walk into.

Task: `task-228-water-of-life-pet-revive` · Worktree: `.worktrees/task-228-water-of-life-pet-revive` · Branch: `task-228-water-of-life-pet-revive`

---

## 1. What the feature is

A pet in MapleStory is a time-limited cash item. When its lifespan runs out it does not vanish — it "dries up" into an inert doll that keeps its cash-inventory slot. The **Water of Life** (`5180000`, classification 518) reawakens it.

Atlas has none of this, and the precondition does not hold either: `atlas-asset-expiration` sweeps every online character once a minute and emits an expire command for *any* expired asset, and `atlas-inventory`'s `ExpireAsset` **deletes** it. A pet that expires in Atlas today is destroyed within sixty seconds. There is no doll to revive. This task builds both halves.

---

## 2. Key files by service

### `libs/atlas-constants`
| File | Why it matters |
|---|---|
| `item/constants.go:71` | `ClassificationPet = Classification(500)` — already exists; the pet exemption keys on it |
| `item/constants.go` (517/519) | `ClassificationWaterOfLife = 518` slots between `ClassificationPetImprints` and `ClassificationPetSkill` |
| `item/constants.go:133` | `GetClassification(itemId) = floor(id/10000)` |

### `libs/atlas-packet`
| File | Why it matters |
|---|---|
| `character/serverbound/chalkboard_close.go` | **The exact precedent.** The repo's only other empty-body serverbound codec — copy its shape and its test's tenant plumbing |
| `pet/serverbound/` | Where the new codec goes; `PET_AUTO_POT` (`pet/serverbound/PetItemUse`, fname `CWvsContext::SendStatChangeItemUseRequestByPetQ`) proves the matrix groups by feature, not client class |
| `docs/packets/gates.yaml` | Wire-*divergence* registry only. A fieldless codec gets no entry |

### `libs/atlas-saga`
| File | Why it matters |
|---|---|
| `model.go:40-50` | Saga `Type` const block (`SealingLockUse`, `ExpirationExtenderUse`, `RemoteMerchant`) |
| `model.go:98` | `EvolvePet Action = "evolve_pet"` — `RevivePet` goes beside it |
| `payloads.go:291-296` | `EvolvePetPayload{CharacterId, PetId}` — the model for `RevivePetPayload` |
| `unmarshal.go:186-191` | The `case EvolvePet:` arm; every new Action needs one or its payload silently stays `json.RawMessage` |

### `atlas-asset-expiration`
| File | Why it matters |
|---|---|
| `expiration/checker.go` | Home of `IsExpired` / `HasExpiration`; `IsReapable` joins them |
| `character/processor.go:57-125` | `CheckAndExpire` and the three sweeps (`checkInventory`, `checkStorage`, `checkCashshop`) — three call sites, one file |
| `character/processor.go` (emit*) | `CheckAndExpire(pp producer.Provider)` takes the producer as a **parameter**, which is what makes the three-sweep test possible without mocking Kafka |
| `inventory/processor_drain_test.go` | The httptest + `t.Setenv("INVENTORY_SERVICE_URL", …)` idiom to copy |

### `atlas-data`
| File | Why it matters |
|---|---|
| `cash/reader.go:79-80` | Reads `addTime` and `maxDays`; `life` goes on the next line |
| `cash/rest.go` | `RestModel` — note how `AddTime` and `MaxDays` document their units in comments; `Life` must too |
| `pet/reader.go:58` | Already parses the same `info/life` node for pet templates — the corroboration that the unit is days |

### `atlas-pets`
| File | Why it matters |
|---|---|
| `pet/processor.go:881-951` (`Evolve`) | **The shape to copy.** Mutates expiration and cascades an in-place inventory mutation from inside one transaction's outbox |
| `pet/processor.go:422` (`Spawn`) | Where the expiration gate goes, beside `ErrTooManySpawnedPets` / `ErrNeedMultiPetSkill` |
| `pet/processor.go:921` | `SetExpiration(time.Now().Add(2160 * time.Hour))` — 2160h = 90 days, the same lifespan WZ `life` encodes |
| `pet/administrator.go:93` (`updateOnEvolve`) | Writes template id **and** expiration. Deliberately not reused — revive gets its own narrow `updateOnRevive` |
| `pet/entity.go:13` | `Migration` is `AutoMigrate(&Entity{})`, so the new column needs no hand-written migration |
| `inventory/command.go:16` | `ChangeTemplate` — the cascade buffer function to mirror |
| `data/pet/` | The `requests.Provider` client shape the new `data/cash` client copies |
| `kafka/consumer/pet/consumer.go:155` | `handleEvolveCommand` — the consumer arm to mirror |
| `kafka/message/compartment/kafka.go` | A **mirror** of atlas-inventory's contract, trimmed to what this service produces |

### `atlas-inventory`
| File | Why it matters |
|---|---|
| `compartment/processor.go:1109-1122` | `ExtendAssetExpiration`'s trust-boundary comment — the reasoning `ResetPetExpiration` reuses verbatim |
| `compartment/processor.go:1131-1134` | The `maxDays == 0` hard-reject that makes `EXTEND_EXPIRATION` unusable here |
| `compartment/processor.go:2090-2115` (`ChangeTemplate`) | The lock + transaction + `IsPet() && PetId() == petId` walk that `ResetPetExpiration` mirrors |
| `asset/processor.go:356` (`ExtendExpiration`) | Reused as-is. Four guards: refuses a locked asset, refuses a permanent one, refuses a backwards move, **no-ops with an `UPDATED` emit on an equal value** |
| `asset/model.go:127` | `IsCash() && petId > 0` — the join between an inventory slot and a pet record |
| `data/cash/` | The extender's cash client; `Life` joins `AddTime` / `MaxDays` |
| `kafka/consumer/compartment/consumer.go:409` | `handleExtendExpirationCommand` — the arm to mirror |

### `atlas-saga-orchestrator`
| File | Why it matters |
|---|---|
| `saga/event_acceptance.go:54-55, 154` | `EventKind` consts and `acceptanceTable`. A missing entry fails `event_acceptance_test.go`'s coverage test |
| `saga/event_acceptance_test.go:14` | `allActions` is **hand-maintained**; a new Action must be added or the coverage test fails |
| `saga/handler.go:117, 828, 1401` | Handler interface, dispatch switch, `handleEvolvePet` |
| `saga/model.go:97, 265, 1404` | Action alias, payload alias, the local payload `switch` (its arm body differs from `libs/atlas-saga/unmarshal.go`'s — copy the local one) |
| `pet/processor.go`, `pet/producer.go` | `EvolveAndEmit` / `EvolveProvider` — the pattern for `Revive` |
| `kafka/consumer/pet/consumer.go:69` | `handleEvolvedEvent`; `kafka/consumer/note/consumer.go:66` is the `*_FAILED` arm shape (`StepCompleted(tx, false)`) |
| `saga/compensator_test.go` | Shows the reverse-walk refund the failure path relies on — `DestroyAsset` compensation already exists |

### `atlas-channel`
| File | Why it matters |
|---|---|
| `socket/handler/character_cash_item_use.go:334-432` | Task-222's extender arm — the saga-creation shape to copy (**but not its `enableActions` calls**; see §4) |
| `socket/handler/character_cash_item_use.go:1053` | The bare `if category == 518` branch FR-9.2 rewrites |
| `socket/handler/chalkboard_close.go` | The minimal handler shape |
| `socket/handler/pet_spawn.go` | `cp.GetById(cp.InventoryDecorator, …)` + `c.Inventory().Cash()` usage |
| `main.go:923` | Where `handlerMap[<Handle const>] = handler.<Func>` registrations live |
| `compartment/model.go:33-66` | `Assets()`, `FindBySlot`, `FindFirstByItemId` |
| `pet/processor.go:101` (`GetByOwner`) | Already drains every page of a character's pets — no new endpoint needed for FR-4.2 |
| `pet/rest.go` | `Expiration` already flows atlas-pets → channel end-to-end |
| `kafka/consumer/system_message/consumer.go:208` | The exact system-message announce call: `session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(charpkt.CharacterStatusMessageOperationSystemMessageBody(text))` |
| `kafka/consumer/asset/consumer.go:270-296` | The `UPDATED` arm that re-announces a slot with `invpkt.NewAddEntry` — FR-7 rides this, no new clientbound codec |
| `saga/model.go:67-68` | Where the `PetRevive` / `RevivePet` aliases go |
| `data/cash/rest.go` | The channel's cash client; `Life` joins `AddTime` / `MaxDays` |

### `atlas-configurations`
`seed-data/templates/template_gms_{83,84,87,92,95}_1.json` — the five templates that get the handler entry. Insertion position in every one is **immediately after `SueCharacter`**. Eleven templates exist; six are deliberately untouched.

---

## 3. Ground truth: opcodes, addresses, WZ

Opcodes, from `docs/packets/registry/gms_v{83,84,87,92,95}.yaml`:

| Version | Opcode | Registry note |
|---|---|---|
| gms_v83 | `0x075` (117) | |
| gms_v84 | `0x075` (117) | registry carries a `discover-ops` correction note; the v84 clientbound table shift does not apply to serverbound |
| gms_v87 | `0x078` (120) | |
| gms_v92 | `0x080` (128) | |
| gms_v95 | `0x081` (129) | |

Senders, decompiled per IDB (not inferred from the shared `fname`) — each is `COutPacket(op)` + `SendPacket` with **zero** `Encode*` calls:

| Version | IDB | Address |
|---|---|---|
| gms_v83 | `MapleStory_dump.exe` | `0xa1dce6` |
| gms_v84 | `GMS_v84.1_U_DEVM` | `0xa68f85` |
| gms_v87 | `GMSv87_4GB.exe` | `0xab501c` |
| gms_v92 | `GMS_v92_1_DEVM` | `0x9c6f90` |
| gms_v95 | `GMS_v95.0_U_DEVM` | `0x9f28e0` |

The v84 and v92 sites were unnamed at the start of this task; both were renamed to `?SendWaterOfLife@CWvsContext@@QAEXXZ`. Unnamed was not treated as absent.

Other addresses this task's reasoning rests on (all gms_v83):

- `CWvsContext::SendEtcCashItemUseRequest` `0xa1dc5b` — switches on `get_etc_cash_item_type` (`0x486845`); classification 518 → type 5 → `SendWaterOfLife()`
- `CDraggableItem::OnDoubleClicked` `0x4efdf7` — gates on `CanSendExclRequest(500)`
- `CWvsContext::SetExclRequestSent` `0xa0ebbc` — **one** caller in the whole binary: `SendConsumeCashItemUseRequest` `0xa0ea6f`
- `CWvsContext::CanSendExclRequest` `0x485bf7` — read-only; sets nothing
- `CWvsContext::OnMessage` `0xa209d4` — fourteen arms, none water-of-life specific; arm 9 is `OnSystemMessage` `0xa21a78`
- `CUIToolTip::GetPetDeadDate` `0x8ebfde` — formats string 678 ("The Water of Life has dried up") or 679 ("Water of Life dries up on …") **from the item slot alone**

WZ: `Item.wz/Cash/0518.img` (GMS 83.1 local extract) contains exactly one item, `05180000`, whose `info` block is `slotMax=1`, `cash=1`, `life=90` — **no `maxDays`, no `addTime`**.

Matrix state before this task (`docs/packets/audits/status.json`): `gms_v83/84/87/92/95` = `incomplete` ("no audit report"); `gms_v48/61/72/79/jms_v185` = `n-a` (`opcode: -1`). The row has no `packet` field yet.

---

## 4. Load-bearing decisions and why

**D1 — The pet exemption goes at the emitter, by classification.** The sweep is the single choke point; the three scans emit to three *different* services' topics. Exempting at the consumers would mean the same rule written three times, in three services, each able to drift. All three REST models already carry `TemplateId`, so no new fetch is introduced. Rejected: exempting inside `atlas-inventory.ExpireAsset` (covers only one of three); a pet-id allowlist (forbidden by FR-1.2, stale the day a tenant ingests a new pet); emitting and having consumers ignore (a Kafka message per doll per minute forever, for no effect).

**D2 — An expired pet is not summonable, and is never force-despawned.** The spawn gate is the *only* gate that matters, because after D1 nothing in Atlas observes the moment a pet expires — the per-minute sweep that used to notice it is exactly what D1 removes. Adding an expiry watcher purely to force-despawn a running pet would reintroduce that scan to serve a case the client has no concept of: there is no clientbound "your pet just dried up" packet, and `GetPetDeadDate` re-reads the item slot only when the tooltip is drawn. `pet.DespawnReasonExpired` (`kafka/message/pet/kafka.go:101`) is declared and referenced by nothing; it stays unused.

**D3 — One empty-body codec covers all five versions.** Verified per-IDB, not assumed from the shared `fname` (the "unverified shared codec" failure mode). No field diverges ⇒ no version gates ⇒ no `gates.yaml` entry. Version scope is enforced entirely by template routing and by the `n-a` matrix cells.

**D4 — No `EnableActions` on any path.** This is the one place where the natural reading of the codebase is wrong. Task-222's extender arm calls `enableActions` on every rejection because `SendConsumeCashItemUseRequest` arms the client's excl latch. That reasoning does **not** transfer: `SetExclRequestSent` has exactly one caller and it is not `SendWaterOfLife`; `CanSendExclRequest` is read-only; `SendWaterOfLife` constructs the packet and sends it, nothing else. The CASH-tab double-click gates on `CanSendExclRequest(500)` but never latches. Sending an unlock would be inert rather than harmful — and a lie in the code about what the client is doing.

**D5 — Two saga steps, not three; the inventory reset is a cascade.** The obvious three-step saga (consume → reset pet → reset asset) has two mutation steps that can half-apply, and a half-application is exactly the bug FR-5.4 names: a pet alive in `atlas-pets` and still a doll in the item slot, or the reverse. A saga can only *compensate* that after the fact. Instead `atlas-pets` writes its own row and buffers `RESET_PET_EXPIRATION` inside the same `database.ExecuteTransaction` + outbox — verbatim what `Evolve` does with `ChangeTemplate`. Either both commit or neither does. The saga is left responsible only for the cross-service consume/refund pair.

**D6 — The channel sends no expiration anywhere.** Channel → pets: `{characterId, petId, sourceTemplateId}`. Pets derives the absolute value from WZ and forwards it plus `sourceTemplateId`. Inventory re-derives the ceiling independently and rejects beyond it. Two independent derivations from the same tenant-scoped WZ node, and a forged channel message cannot name a lifespan at all — strictly stronger than task-222, where the channel computes and the service only bounds.

**D7 — `revive_transaction_id` on `pets` settles both hazards at once.** Two hazards: Kafka redelivery (at-least-once; this codebase has duplicated items through non-idempotent handlers before) and the 500 ms double-water race (D4 established the client throttles but does not latch, so two requests can be in flight before either revive lands — and both channel-side selections would pick the same doll). Naive guards fail one hazard each: "reject if already alive" refunds correctly for the race but wrongly refunds a redelivery; "no-op if already alive" is idempotent but silently eats the second water in the race. Only the transaction id distinguishes them.

| State | Action |
|---|---|
| `revive_transaction_id == command.TransactionId` | Redelivery. Re-buffer the cascade with the **stored** expiration, re-emit `REVIVED`, no write |
| expiration in the future, different transaction id | Live pet. `REVIVE_FAILED` → saga refunds the second water |
| expiration in the past | Revive |

Re-cascading the *stored* expiration (rather than skipping the cascade) is what makes the pair converge even if the first delivery's cascade was itself lost.

---

## 5. Traps

- **A handler entry with a missing or empty `validator` is silently dropped** at dispatch-map build time. Every new entry carries `LoggedInValidator`.
- **The template `handler` value must equal the codec's `Operation()` return** exactly, or the dispatch map never binds. `WaterOfLifeHandle = "WaterOfLifeHandle"` on both sides.
- **`tools/template-opcode-order-guard.sh` requires strictly ascending `opCode`.** New entries go at their sorted position — never appended next to a semantically-related entry.
- **`EXTEND_EXPIRATION` is not reusable.** It hard-rejects `maxDays == 0` and `0518.img` has no `maxDays` node. Relaxing that guard to accept a second cap source would make one command mean two things and weaken the extender's own ceiling.
- **Rejections inside `Revive` must buffer `REVIVE_FAILED` and return `nil`, not an error.** The transactional emit path discards the buffer when the closure errors, so a rejection returned as an error never reaches the saga — and the player's already-consumed Water of Life would wait out the flat saga timeout for its refund. `atlas-inventory`'s `CreateAssetAndEmit` (~L1160) documents the same trap and works around it by re-emitting on a direct producer.
- **`updateOnRevive` must use the transaction handle already in scope**, not a fresh one from the pool. A second pooled connection taken from inside a transaction deadlocks at pool size 1 (a known bug pattern in this repo).
- **Two Kafka contracts are mirrored across module boundaries with no guard script:** `ResetPetExpirationCommandBody` (atlas-pets ↔ atlas-inventory) and the pet status events (atlas-pets ↔ atlas-channel ↔ atlas-saga-orchestrator). A field name or json tag changed in one and not the other fails no build — it decodes into a zero-valued body at runtime: a pet revived to the zero time, i.e. still a doll. Plan Task 10 Step 7's `diff` is the manual stand-in; re-run it in Task 15.
- **`allActions` in `event_acceptance_test.go` is hand-maintained.** Adding an Action without adding it there passes the coverage test for the wrong reason.
- **Green unit tests on a stubbed seam are zero coverage.** The two seams to trace into their real consumers are the `RESET_PET_EXPIRATION` cascade and the asset `UPDATED` → channel add-entry announce.
- **The three-sweep regression test can pass for the wrong reason** if a stubbed URL falls through to the `default` arm and the sweep finds nothing. Prove the stubs work by checking the non-pet case *fails* without them.
- **`tools/lint.sh --check` false-fails without nvm on PATH.** If it errors before linting anything, load nvm and re-run.
- **Do not use expensive models for the verify/review subagents.** Pin `packet-verifier` and the reviewers to Sonnet/Haiku.

---

## 6. Dependencies between tasks

```
T1  constants ──────────────────────────────┐
T2  asset-expiration (independent)          │
T3  atlas-data life ───┬────────────────────┤
T4  codec ─────────────┼──► T5 templates    │
T6  saga lib ──────────┼─────────┐          │
T7  spawn gate (indep) │         │          │
T8  pets client+contracts ◄──────┘          │
     └──► T9 pets REVIVE                    │
     └──► T10 inventory RESET_PET_EXPIRATION│
     └──► T11 orchestrator ◄── T6           │
T12 channel handler ◄── T1, T4, T6, T3 ─────┘
T13 channel REVIVE_FAILED ◄── T8, T12
T14 matrix promotion ◄── T4
T15 full verification ◄── everything
```

T2 and T7 are fully independent and can run at any point. T3 must precede T8/T10/T12 (they read the `life` field over REST). T6 must precede T11 and T12.

---

## 7. Out of scope (stated, not fixed)

- **Pets already reaped** by the current behaviour are gone. No migration recovers them (NFR-7). Flag if a live tenant needs a data repair.
- **Dolls in storage or the cash-shop locker** persist (D1 stops the reap in all three locations) but are **not** eligible revive targets — the request carries no target and the client's pet UI reads the cash inventory. A doll in storage must be moved before it can be revived.
- Cash-shop *purchase* of `5180000` (the generic commodity path already covers it); pet evolution (538), pet name tags (517), pet skills (519), pet food (524/546); the other `SendEtcCashItemUseRequest` arms (store permit type 4, emotion type 6, gachapon remote type 57); expiration behaviour for any classification other than pets.
- A residual by design: `atlas-inventory` rejecting the cascade would leave the pet alive and its slot a doll. Both services derive from the same tenant-scoped WZ node, so a legitimate flow never hits it — the rejection path exists only for a forged `COMMAND_TOPIC_COMPARTMENT` message, where a divergence is preferable to an unbounded grant.
