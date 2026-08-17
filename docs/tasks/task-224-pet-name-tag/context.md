# task-224 Pet Name Tag — Implementation Context

Companion to [`plan.md`](plan.md). The plan says *what to do*; this says *what
you are walking into* — the files that already exist, the decisions already
made and why, and the traps that have bitten this codebase before.

---

## 1. The one-paragraph summary

Item `5170000` (Pet Name Tag, WZ classification 517) currently does nothing.
Two independent defects sit between the player and a rename: atlas-channel's
cash-slot type resolver has a `uint32` overflow that makes the 517 branch return
`0`, and atlas-pets has no rename path at all — no builder setter, no
administrator function, no Kafka command, no status event, no REST mutation.
On top of that, the clientbound `PET_NAMECHANGE` packet has never been
implemented (`❌` in all nine applicable matrix columns). This task closes all
three.

---

## 2. Key decisions already made (do not re-litigate)

| Decision | Why | Source |
|---|---|---|
| **Name bounds are 4–12, not 1–13** | The v83 client's own input dialog is `sub_9AC7CB(dlg, NULL, 4, 12, 0, 1)` @`0xa0bb2f`. The (min,max) reading is fixed by three sibling call sites: `CTabParty::OnInvite` (4,12) for a character name, `ask_SPW` (8,8) for the 8-digit password, `ask_guildname` (4,12). The `pets.name` column is `size:13` — one wider, and NOT the binding constraint. | design C-2 / OQ-5 |
| **The clientbound flag byte is GMS-only** | JMS v185 `CPet::OnNameChanged` @`0x76a5de` does exactly one `DecodeStr` and no `Decode1`; it branches on `sub_768D82(this)`, client state. A uniform body would desync every JMS rename. | design C-1 |
| **The server resolves the pet; the rule is slot 0** | The wire carries no pet id (one `EncodeStr`, and the pet-picker `SetUtilDlgEx_Pet` @`0x9acb27` is never called from this send path). The client's own arm calls `sub_46D2D5(this, 0)` @`0xa0ba47`, which resolves pet-locker index 0 = Atlas slot 0. So the server rule *matches the client by derivation*, not by convention. | design OQ-4 |
| **Rename first, consume second** | Inverse of task-220's meso sack. A rejected name must not cost a cash item. Costs one compensator branch. | PRD FR-7.2 |
| **Rejections render pink text, not a silent unlock** | `chatpkt.WorldMessageWriter` + `writer.WorldMessagePinkTextBody` is present in **all ten** templates and already load-bearing in two failure paths (`point_reset` @`saga/consumer.go:347-360`, `meso_sack_use` @`:365-381`). A silent unlock reads to the player as "the item did nothing" — the exact complaint that opened this task. | design OQ-2, A4 |
| **The `nameTag` byte is a shared constant, not a config key** | It is a `CLife::MakeNameTag` render selector that must equal `Activated.nameTag`, not a per-version wire code. `WithResolvedCode("operations", …)` exists for the latter. DOM-25 requires provenance, not a tenant key nobody will tune. | design A5 |
| **No atlas-data change, no WZ re-ingest** | `Cash/0517.img` carries only `z`, `slotMax`, `cash` and icons — no value node. The feature reads no WZ value at all, so the conclusion holds by construction, not by per-version survey. This is the one operational burden task-220 carried and this task does not. | design OQ-1 |
| **No profanity filter** | The repo has no profanity service to reuse; the client still runs `CCurseProcess::ProcessString` @`0xa0bb9a`, so the gap needs a modified client and the worst outcome is a rude pet name. Deliberate boundary. | PRD FR-4.5, design OQ-3 |

---

## 3. Landmark files

### The precedent to copy from — task-220 (Meso Sack)

Almost every layer of this task has a working analogue landed by task-220. When
in doubt, open the meso-sack file and match its shape.

| Layer | Meso-sack analogue |
|---|---|
| Handler | `services/atlas-channel/.../socket/handler/character_cash_item_use_meso_sack.go` |
| Saga builder | `buildMesoSackUseSaga` in the same file |
| Compensator | `compensateMesoSackUse` / `DispatchMesoSackRollbacks` @ `saga/compensator.go:1603+` |
| Timer classification | `saga/timer.go:180,207,248` |
| Failure rendering | `channel/kafka/consumer/saga/consumer.go:365-381` + `mesoSackFailureMessage` |

The **one deliberate divergence**: meso sack is consume-then-award; this is
rename-then-consume.

### Serverbound codec

`libs/atlas-packet/cash/serverbound/item_use_kite.go` is the direct model — it
is also a single `EncodeStr` sub-body with the same `updateTimeFirst` trailing
gate. Its doc comment is the format to imitate (IDA addresses, what is *not* on
the wire and why).

The common header gate lives in `libs/atlas-packet/cash/serverbound/item_use.go`
(`UpdateTimeFirst(t)`): GMS ≤ v84 trails `update_time` in the sub-body; GMS v87+
and JMS lead it in the header. The case-17 arm falls through to the shared tail
at `loc_A0E9EC`, so it obeys that gate like every sibling.

### Clientbound codec

`libs/atlas-packet/pet/clientbound/` holds the family. Every leaf body is
prefixed by `ownerId uint32` + `slot int8`, read upstream by
`CUser::OnPetPacket` before dispatch — byte-verified for the family in
`v61_test.go:11-14`, and encoded by `chat.go:37-38`, `command.go:51-52`,
`movement.go`, `exclude.go`. `PET_NAMECHANGE` inherits it unchanged; there is
nothing to guess about framing.

`activated.go` writes the same `nameTag` byte in the spawn body (currently always
zero). Task 3 points both at one constant so they cannot drift.

### atlas-pets

- `pet/administrator.go` — six existing `update*` functions. `updateName` follows
  them, with one deliberate difference: it does **not** treat
  `RowsAffected == 0` as an error (see §5 idempotency).
- `pet/processor.go:873-950` — `EvolveAndEmit` / `Evolve` is the exact
  transaction+message-buffer pair shape to copy for `RenameAndEmit` / `Rename`.
- `pet/producer.go:191` — `evolvedEventProvider` is the provider shape.
- `kafka/consumer/pet/consumer.go:59,155` — registration + handler shape.
- `pet/resource.go:18-30` — GET + POST-create only today; PATCH is new.

### atlas-channel handler test seams

`character_cash_item_use_meso_sack_test.go` already provides
`installCapturingProducer()`, `newCashItemUseTestSession(t, characterId)`,
`gaugeProducerRecorder`, and the `installCashItemDataSeam` package-var swap
pattern. Reuse them; the pet-name-tag tests only add one new seam installer for
`petsForOwnerFunc`. `rec.calls` counts announced packets — a rejection announces
2 (pink text + enable-actions), the happy path announces 0.

### atlas-saga-orchestrator

- `saga/handler.go:1352` — `handleEvolvePet` is the step-handler shape.
- `saga/event_acceptance.go:54-55,149,357` — the three blocks an event kind must
  be added to.
- `saga/model.go:1388` + `libs/atlas-saga/unmarshal.go:186` — the two payload
  unmarshal switches. `unmarshal_completeness_test.go` fails if either arm is
  missing; that is the intended gate.
- `saga/compensator.go:188-236` — the `With*Processor` injector block. There is
  **no** `WithPetProcessor` and the orchestrator's `pet` package has **no mock**
  (unlike `compartment`, `character`, `cashshop`…). Both are prerequisites for
  the compensation test and are part of Task 11, not a deferral.
- `saga/meso_sack_compensation_test.go` — the compensation-test template,
  including its `//go:build test` tag, `newMesoSackSaga` builder helper, and the
  `NewCompensator(...).WithXProcessor(mock).DispatchXRollbacks(s)` call shape.
- `saga/processor_test.go:610-660` — the `AcceptEvent` test arrangement
  (`NewBuilder()…Build()`, `GetCache().Put(ctx, s)`, then `AcceptEvent`).
- `saga/timer.go:170-207` — `reverseWalkSagaTypes`, `noReverseWalkSagaTypes`, and
  `allSagaTypes`. `TestEverySagaTypeIsClassified` fails when a type is added to
  `allSagaTypes` without landing in one of the first two lists. Read the comment
  above `noReverseWalkSagaTypes` — it explains exactly why the three-list scheme
  exists.

---

## 4. The defect being fixed (verified, not assumed)

`services/atlas-channel/.../socket/handler/character_cash_item_use.go:936-941`:

```go
if category == item.ClassificationPetImprints {
    if 10000*itemId/10000 != itemId {
        return CashSlotItemType(0)
    }
    return CashSlotItemType(17)
}
```

`item.Id` is `uint32` (`libs/atlas-constants/item/constants.go:5`). With
`itemId = 5170000`: `10000 * 5170000 = 51,700,000,000`, which mod 2³² is
`160,392,448`; `/10000 = 16039 ≠ 5170000`. The branch returns `0` and the item
falls through to the `l.Warnf("… of type [%d]")` at `:679`. The client's actual
predicate is `get_cashslot_item_type` @`0x48645b` `case 517: return a1 % 10000 != 0 ? 0 : 17;`.

Note that the constant `17` is already returned here — what is missing is the
*named* constant and the *arm*, plus this overflow. Type 17 is unambiguous: no
other classification maps to it (meso sacks return 19 on every version by
deliberate Atlas policy, documented at `:699-708`).

---

## 5. Traps this codebase has actually hit

- **Seed template writers require an `fname`.** One without it is silently
  dropped at seed time. (`bug_seed_template_writers_require_fname`)
- **New opcodes missing from a *live* tenant config are silently dropped.** The
  seed templates are not the live configs; reconciliation is a rollout step.
  (`bug_new_opcodes_not_in_live_tenant_config`)
- **Zero-padding a hex opcode creates a duplicate binding.** `0xB8` and `0x0B8`
  are the same numeric code; binding both makes the dispatch map's
  last-write-wins decide which options survive. Match each template's own
  spelling. (task-194 / `tools/template-duplicate-binding-guard.sh`)
- **Writers must be inserted at their sorted `opCode` position**, never appended
  next to a semantically-related entry.
  (`tools/template-opcode-order-guard.sh`)
- **`ForSessionsInMap` / `ForEachInMap` iterate in PARALLEL.** The broadcast
  callback must close over immutable values only.
  (`bug_channel_foreachinmap_parallel_shared_state`)
- **Kafka is at-least-once.** A redelivered `RENAME` must complete, not error —
  and must re-emit `NAME_CHANGED`, because that re-emission is what completes the
  orchestrator's step on the duplicate. A redelivered *consume* is the more
  dangerous half; that is the saga's problem, not the processor's.
  (`bug_kafka_redelivery_dupes_nonidempotent_handlers`)
- **The five-way pet Kafka mirror has no guard.** Only trade has one. A json-tag
  typo fails no build and decodes into a zero-valued body at runtime. Task 12's
  round-trip fixtures are the substitute — writing a sixth mirror guard was
  considered and rejected (the mirror set is wider than trade's two, and the test
  is cheaper for the same class).
  (`feedback_green_tests_miss_cross_service_seams`)
- **A round-trip fixture alone is NOT matrix evidence.** Each cell's read order
  must be derived from that version's own client, with an IDA address in the
  fixture comment. (`bug_matrix_roundtrip_fixture_false_verify`)
- **A registry edit stales the matrix.** Re-confirm opcodes at implementation
  time rather than trusting the tables in the PRD/design/plan.
  (`bug_registry_fname_change_stales_packet_matrix`)
- **`tools/lint.sh --check` false-fails without nvm on PATH**, and contends on a
  golangci-lint lock across worktrees. Confirm the environment before believing a
  failure. (`bug_lint_check_false_fails_without_nvm`)
- **IDA:** `select_instance(port)` is dead. Resolve the session from `idb_list`
  by binary **name** and pass it as the `database` parameter.
  (`reference_ida_mcp_new_api`)

---

## 6. Dependency graph between tasks

```
T1 (constants) ─────┬──────────────► T7 (pets processor) ──► T8 (PATCH)
                    └──────────────► T13 (channel handler)
T2 (sb codec) ─────────────────────► T13
T3 (cb codec) ──► T4 (matrix) 
             └─────────────────────► T14 (broadcast)
T5 (pets contract) ─┬─► T7
                    ├─► T10 (orchestrator mirror)
                    └─► T12 (channel mirror) ──► T13, T14
T6 (builder/admin) ──► T7
T9 (shared saga) ───┬─► T10 ──► T11 (compensation)
                    └─► T12
T15 (templates) — independent of all Go work
T16 (docs + gates) — last
```

Parallelizable clusters if executing with subagents: **{T1, T2, T3, T5, T6, T9,
T15}** have no cross-dependencies among themselves. T4 depends only on T3. The
rest are sequential as drawn.

---

## 7. Scope boundaries

**In scope:** everything in `plan.md`.

**Explicitly out:**
- Renaming anything other than a pet (character/guild/pet-*equipment* renames).
- Any atlas-ui surface.
- The other unimplemented one-off cash types in
  `docs/research/missing-features/items-and-consumables.md:33-49`.
- Changing how a pet's *initial* name is assigned at purchase
  (`services/atlas-cashshop/.../cashshop/processor.go:166-187`).
- Widening the 13-character DB column (it is not the binding constraint —
  the client caps at 12).
- A profanity filter.

---

## 8. Two things the design flagged as real work, not rounding errors

1. **v72, v79, v84, v87 `CPet::OnNameChanged` have NOT been read.** Four GMS
   reads (v61, v83, v92, v95) agree on `str + byte`, and JMS diverges. That is an
   expectation, not evidence — each of the four must be decompiled before its
   fixture is written. If one diverges, the gate in `name_changed.go` is already
   region-shaped, so adding a major boundary is a one-line change.
2. **The entire v92 column is `❌`** — every pet op and even `LOGIN_STATUS` sit
   unverified there, so v92 has had no verification pass at all. Its IDB is
   present (`GMS_v92_1_DEVM.exe.i64`) and `CPet::OnNameChanged` @`0x6967c0` was
   read during design, but promoting that cell is a fresh derivation rather than
   a copy of a verified neighbour.
