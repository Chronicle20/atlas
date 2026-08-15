# Scissors of Karma — Implementation Context

Companion to [`plan.md`](plan.md). Reference material an implementer needs that does not belong inside a task's steps: where the existing patterns live, which decisions are already settled and why, and the traps that have teeth.

---

## 1. Where things are

### Existing code the plan mirrors

| What you're writing | Copy the shape from |
|---|---|
| The `ItemUseKarmaScissors` codec | `libs/atlas-packet/cash/serverbound/item_use_seal.go` (+ its test) — field-identical |
| The handler arm | the Sealing Lock arm, `services/atlas-channel/.../socket/handler/character_cash_item_use.go:265-332` |
| The two-step saga | the Item Tag arm, same file `:226-259` |
| The version-scoped type resolver | `viciousHammerCashSlotItemType`, same file `:760-765` |
| The client-unlock announce | the incubator NPC-absent arm, same file `:367` |
| `asset.ApplyKarma` / `ClearKarma` | `services/atlas-inventory/.../asset/processor.go:329-357` (`ApplyLock`/`ClearLock`) |
| `compartment.ApplyAssetKarma` | `services/atlas-inventory/.../compartment/processor.go:1045-1077` (`ApplyAssetLock`) |
| The `APPLY_KARMA` command + consumer | `services/atlas-inventory/.../kafka/{message,consumer}/compartment/` (`APPLY_LOCK`, `:35`/`:186`/`:96`/`:384`) |
| The saga action / payload / unmarshal | `libs/atlas-saga` `ApplyAssetLock` at `model.go:220`, `payloads.go:1088`, `unmarshal.go:582` |
| Orchestrator dispatch + producer | `.../saga/handler.go:949,1139`; `.../compartment/producer.go:126`; `.../compartment/processor.go:44,130` |
| The per-compartment data client | `services/atlas-trades/.../data/item/` — five wire models dispatched on the compartment; the canonical solution to "atlas-data has no unified item resource" |
| The compartment-byte decode | `services/atlas-trades/.../trade/restriction.go:50-71` (`stageableInventoryType`) |
| A compensation reverse-walk test | `.../saga/meso_sack_compensation_test.go`, `point_reset_compensation_test.go` |

### Verified line references (as of `cb6734a4d`)

Line numbers drift. Every one below was read during planning; if a reference misses, grep for the named symbol rather than trusting the number.

- `libs/atlas-constants/asset/flag.go:7-11` — `FlagSpikes`/`FlagKarmaUse` both `0x02`; `FlagKarmaEquip` `0x10`
- `libs/atlas-packet/cash/serverbound/item_use.go:21-23` — `UpdateTimeFirst`, gate `GMS >= 87 || JMS`
- `character_cash_item_use.go:55-59` — the `cashItemInSlotFunc` ownership pre-check (gate 0a, already in place)
- `character_cash_item_use.go:261-265` — the seal arm's GMS ≥ 95 recompute to `65`
- `character_cash_item_use.go:696-697` — `CashSlotItemTypeSealTimed` (64) / `SealTimedV95` (65)
- `character_cash_item_use.go:1103-1109` — the bare `category == 552` branch
- `character_cash_item_use.go:679` — the terminal `not implemented` warn a karma use currently falls through to
- `services/atlas-saga-orchestrator/.../saga/processor.go:1597-1598` — trade **settlement** expansion (masks karma)
- `services/atlas-saga-orchestrator/.../saga/processor.go:1682-1683` — trade **unwind** expansion (does NOT mask)
- `services/atlas-saga-orchestrator/.../saga/processor.go:1723` — `assetDataFromSnapshot`
- `services/atlas-saga-orchestrator/.../saga/compensator.go:277` — the cash-item-use saga-type arm
- `services/atlas-merchant/.../shop/processor.go:1038` — the **buy** path (masks karma)
- `services/atlas-merchant/.../shop/processor.go:628,783,1354` — shop-closure / listing-removal / Frederick returns (do NOT mask)
- `libs/atlas-saga/payloads.go:1128` — `AssetSnapshot`, carrying `Flag uint16`

### The seven services carrying the inverted karma pair

| Service | Getter | Setter |
|---|---|---|
| `atlas-inventory` | `asset/model.go:84` | `asset/builder.go:147` |
| `atlas-channel` | `asset/model.go:92` | `asset/builder.go:180` |
| `atlas-login` | `inventory/compartment/asset/model.go:82` | `.../builder.go:144` |
| `atlas-cashshop` | `asset/reference_data.go:54, 373` | `reference_data.go:259, 586` (already correct) |
| `atlas-consumables` | `asset/model.go:82` | `asset/builder.go:144` |
| `atlas-storage` | `asset/model.go:82` | `asset/builder.go:142` |
| `atlas-query-aggregator` | `asset/model.go:82` | `asset/builder.go:144` |

`atlas-pets` and `atlas-npc-shops` also expose `karmaUsed`, but as plain bools on their own models with **no flag arithmetic**. They are untouched — consistent with pets being out of scope.

---

## 2. Decisions already made — do not re-litigate

These were settled in the design phase from decompiles and WZ data. Re-deriving them costs hours; contradicting them breaks the feature.

**The WZ spellings are `info/tradeAvailable` (target) and `info/karma` (scissors), both integers.** Resolved by decoding `StringPool::ms_aString` out of the gms_v95 IDB and validating the decoder against two independently-known entries first. `tradeAvailable` is pinned structurally, not by name similarity: `BUNDLEITEM.nAppliableKarmaType` sits at offset `0x14`, and its parse-run neighbours (`only`@`0xC`, `tradeBlock`@`0x10`, `notSale`@`0x18`) confirm the alignment. Both are confirmed present in the shipped v83 corpus — `tradeAvailable` occurs 159 times, including `Character.wz/Cap/01002357.img` (Zakum Helmet).

**Eligibility is one predicate with no version gate.** The gms_v83 client asks "is `tradeAvailable` non-zero?"; v87 and v95 ask "does it *equal* the scissors' `karma`?". `KarmaEligible` covers both because the data selects the branch: v83-era scissors carry no `karma` node, so the equality clause is vacuous. This is why **gms_v84 needed no decompile** and why **OQ-4 dissolved** — no code path compares a karma type against a literal, so `5520001` works the moment a tenant's WZ carries it.

**The predicate is deliberately stricter than the client in two places.** Gate 2's `targetKarma != 0` closes a real client hole (a v95-era tenant shipping scissors without `karma` makes the client accept *every ordinary item*), and gate 4 (already-tradeable) has no client counterpart at all. Both are recorded so a future reader does not "fix" them back into parity.

**Pets are refused, not supported.** Three independent reasons: no pet carries `tradeAvailable` in the shipped data; the client's pet karma bit is `0x01`, which is `FlagLock` in Atlas's shared column (the client gets away with it only because `GW_ItemSlotPet::IsProtectedItem` hard-returns 0); and Atlas does not model pets as flag-carrying inventory assets. Supporting them later needs a separate flag column — a data-model change the PRD excludes.

**A merchant sale consumes the mark.** Not answerable from client behaviour — the mark is a server-owned bit the client only renders. It is a server policy call, settled by the grant's own wording (`…SO_1_TIME_OF_TRADING_HAS_BEEN_ENABLED`): the unit consumed is a transfer of ownership. Listing-only semantics would let a player launder one mark into unlimited transfers by re-listing.

**Consumption happens in the snapshot, not after the transfer.** Both paths already move an asset by destroying it on one side and re-materialising it from a snapshot carrying `Flag`. Masking the bit there makes the clear and the transfer the same write — atomicity is structural, and the cancelled-trade case falls out for free because unwind replays the same snapshot unmasked.

**Constant values do not change, and the names are not fixed.** `FlagKarmaUse`/`FlagKarmaEquip` read backwards from the client's usage but are load-bearing in seven services. They get a comment; a rename is a separate, larger change with no behavioural benefit.

---

## 3. Traps

**`0x02` is `FlagSpikes` on an equip.** This is the sharpest failure mode in the task: a careless "karma is 0x02" renders spikes on every karma'd equip, visibly, on live characters. `KarmaFlagFor` is the *only* thing permitted to select a karma bit, and every mutation is a targeted `SetFlag`/`ClearFlag`, never an assignment. Task 1, 4, 14 and 15 each carry a spikes-specific regression test.

**The seal and karma cash-slot types collide pre-95.** `CashSlotItemTypeSealTimed` is `64`, and so is the GMS ≥ 95 karma type. Today they are disjoint only because the seal arm recomputes itself to `65` at GMS ≥ 95 — a coincidence, not a design. Version-scoped resolvers on both sides plus `TestKarmaAndSealResolversAreDisjoint` are what make it structural. **Placement matters too:** put the karma arm *after* the seal arm in the dispatch chain, exactly where the plan says.

**Two copies of the `APPLY_KARMA` command body, in two Go modules.** atlas-inventory's and atlas-saga-orchestrator's `ApplyKarmaCommandBody` must be byte-identical — same field names, same json tags. A field renamed in one and not the other decodes into a zero-valued body at runtime and **fails no build**. (The trade contract has a CI guard for exactly this class of bug; the compartment contract does not.)

**Every handler on `COMMAND_TOPIC_COMPARTMENT` sees every command on it.** The `if c.Type != CommandApplyKarma { return }` guard is not boilerplate — without it an `APPLY_LOCK` body unmarshals into an `ApplyKarmaCommandBody` and mutates the wrong thing.

**`inventory.Type` is a signed `int8`.** A raw wire value above 127 arrives negative and silently addresses a nonexistent compartment if merely converted. Both `knownInventoryType` (channel) and the existing `stageableInventoryType` (trades) exist for this reason.

**Item-data lookup failure is a refusal, never a permissive default.** This is a contract atlas-trades already holds itself to (`errItemDataUnknown` sits *above* the `tradeBlock` check so a karma mark can never rescue an unreadable lookup). The karma gates inherit it at both layers.

**The client is input-locked from the moment it sends.** gms_v83 `@0x830FB5` gates on `CanSendExclRequest(500, 0)` and sets the lock. A refusal that returns silently wedges the client until the next unlocking packet. The success path unlocks itself via the non-silent `INVENTORY_OPERATION` driven by the `UPDATED` event; only refusals need the explicit empty-`StatChanged` announce.

**Saga step order is load-bearing.** Scissors destroyed *first*, mark applied *second*. A failure to apply then compensates by restoring the scissors rather than leaving a free trade behind.

**Gate 3 at the inventory layer is the idempotency guarantee.** Setting a bit is idempotent at the bit level, so the refusal is about the *audit*: a redelivered command must never let a second scissors be silently consumed against an already-marked item. Kafka is at-least-once here.

**Don't mask karma in `assetDataFromSnapshot`.** It is shared by settlement *and* unwind. The mask goes at the settlement call site only, or a cancelled trade eats the mark.

**`tools/lint.sh --check` false-fails without nvm on PATH,** and contends on a cross-worktree golangci-lint lock. Run `tools/lint.sh` (fix mode) before `--check`.

---

## 4. Dependency order

Tasks 1–5 are independent of each other and can be done in any order. After that:

```
1 (KarmaFlagFor) ──┬─> 4 (7-service fix)
                   ├─> 7 (inventory processors) ──> 8 (kafka) ──┐
                   ├─> 13 (trades)                              │
                   ├─> 14 (settlement clear)                    │
                   └─> 15 (merchant)                            │
2 (codec + 552) ───────────────────────────> 12 (handler arm)   │
3 (atlas-data) ──> 6 (inv client) ──> 7      ▲                  │
             └───> 10 (channel clients) ─────┤                  │
5 (saga contract) ──> 9 (orchestrator) <─────┴──────────────────┘
11 (type resolver) ─> 12
                       └─> 16 (manifest + gate)
```

Task 9 depends on Task 8 (it mirrors that Kafka contract) and Task 5 (the action). Task 16 is last.

---

## 5. What "done" requires beyond green tests

Per CLAUDE.md, and enumerated in plan.md Task 16:

- `go test -race ./...`, `go vet ./...`, `go build ./...` in **every** changed module — thirteen of them.
- `docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched. None should be; **verify** rather than assume, because `go build` against the workspace will not catch a missing `COPY libs/...` in the shared Dockerfile.
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` clean from the repo root.
- No template guard applies unless a template changes — `USE_CASH_ITEM` is expected to be bound everywhere already. Confirm it; if any template lacks the binding, `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh` both become mandatory.
- `packet-completeness-critic` before the PR, alongside `plan-adherence-reviewer` and `backend-guidelines-reviewer` via `superpowers:requesting-code-review`. Code review runs **before** the PR, not after.

## 6. Not in scope

Named here so they are not mistaken for gaps:

- Cash-shop *purchase* of scissors (the generic commodity path already covers it).
- A new packet opcode — `USE_CASH_ITEM` already carries this.
- The general absence of untradeable gating on item **drop** and **NPC sale**: neither consults `FlagUntradeable` nor `tradeBlock` today. A pre-existing gap, not karma's to close.
- Storage, Duey and mailing transfer paths.
- Any atlas-ui surface.
- Renaming `FlagKarmaUse`/`FlagKarmaEquip`.
- Pet karma targets (§2).
