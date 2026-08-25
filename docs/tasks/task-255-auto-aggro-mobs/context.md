# task-255 — Implementation Context

Companion to [plan.md](plan.md). Everything here was read out of the repo or the
IDBs during planning; nothing is recalled from general MapleStory knowledge.

---

## 1. Key files by area

### Packet layer (`libs/atlas-packet`, module root `libs/atlas-packet`)

| File | Why it matters |
|---|---|
| `monster/serverbound/mob_drop_pickup_request.go` | The shape to copy: identical two-`Encode4` serverbound mob codec, immutable struct + `Operation`/`String`/`Encode`/`Decode`, `packet-audit:fname` marker. |
| `monster/serverbound/mob_drop_pickup_request_test.go` | The golden-byte + `pt.Variants` round-trip + per-version `packet-audit:verify` marker pattern. |
| `test/context.go:18-41` | `pt.Variants` — twelve tenant variants, including v48/v61/v72/v79/v92. Appended, never inserted, so positional indices stay valid. |
| `tools/packet-audit/cmd/run.go:1198` | `candidatesFromFName` — a NEW serverbound op does not link to a codec (and therefore gets no audit report and cannot pin evidence) until its primary `fname` gets a case here. |

### atlas-monsters (module root `services/atlas-monsters/atlas.com/monsters`)

| File | Why it matters |
|---|---|
| `monster/processor.go:400-440` | `startControl(uniqueId, controllerId, forceAggro)` — the atomic controller-transfer-with-aggro already used by Monster Magnet. `SetAggro`'s transfer branch reuses it verbatim. |
| `monster/processor.go:1822-1880` | `ForceControl` — the exact rejection/log/`return nil` discipline `SetAggro` mirrors, plus the `inFieldFn` and `hiddenFn` gate order. |
| `monster/processor.go:78-81, 979-984` | `testInformationLookup` — the package-level hook that lets `SetAggro`'s `firstAttack` gate be tested without HTTP. Do **not** add a second seam. |
| `monster/registry.go:695-830` | `DecaySummary` / `DecayDamageEntries` / `ClearSummary` / `ClearDamageEntries` — the `r.reg.Update`-over-`storedMonster` pattern (`Model` has no builder path for entry-list mutation) and the "captured values reflect the final retry" comment discipline. |
| `monster/registry.go:26-55` | `storedMonster` — the Redis JSON shape. `AggroRefreshedMs` goes next to `LastDamageTakenMs`, `omitempty`. |
| `monster/aggro_task.go:76-107` | `MonsterAggroDecayTask.Run` — `bossLookupFn`, `emit` and `nowFn` are struct fields, so the lease branch is directly testable. |
| `monster/information/{model,builder,rest}.go` | `RestModel.FirstAttack` (`first_attack`) already exists at `rest.go:35`; only the `Model` field, accessor, `Extract` mapping and builder setter are missing. |
| `kafka/consumer/monster/{kafka,consumer}.go` | `CLEAR_AGGRO` / `FORCE_CONTROL` are the symmetric precedent for `SET_AGGRO`, including the "edit both together" mirror comment. |

Test helpers (all pre-existing, all reused rather than re-invented):
`newTestTenant` (`cooldown_test.go:28`), `newAggroedMonster` (`clear_aggro_test.go:13`),
`recordingProcessor` (`control_assignment_test.go:17`), `forceControlProcessor`
(`force_control_test.go:13`), `newAggroTaskWithRecorder` (`aggro_task_test.go:28`).
CLAUDE.md forbids new `*_testhelpers.go` files; the plan's one new helper
(`setAggroProcessor`) lives inside the test file that uses it, exactly as
`forceControlProcessor` does.

### atlas-channel (module root `services/atlas-channel/atlas.com/channel`)

| File | Why it matters |
|---|---|
| `monster/live_mirror.go` | `LiveEntry` + `LiveMirror`; the `sync.Once` singleton + sweeper + `//goroutine-guard:allow` shape the rate gate copies, and the "events never create entries" invariant. |
| `monster/producer.go:191-224` | `ClearAggroCommandProvider` / `ForceControlCommandProvider` — the provider shape and the monster-id keying that orders commands for the same monster. |
| `monster/processor.go` | The `Processor` interface + `ProcessorImpl`; `ClearAggro`/`ForceControl` are one-line producer wrappers, and `SetAggro` is another. |
| `kafka/consumer/monster/consumer.go:320-406` | `handleStatusEventStartControl` and `handleStatusEventAggroChanged` — **already** re-issue `StartControlMonsterBody(m, aggro)` and already thread `ControllerHasAggro` through a handover. FR-6.1/FR-6.2 need tests, not code. |
| `kafka/consumer/monster/consumer.go:416-460` | `monsterGetByIdFn` / `controlGrantFn` / `announceFn` — the established `var xFn = func(...)` seam pattern the new handler and its tests use. |
| `socket/handler/monster_catch_item_use.go` | The acting-handler shape (decode, debug-log, forward) as opposed to the decode-and-log stubs. |
| `main.go:907-922` | `produceHandlers()` — the `monstersb.*Handle` registration block. |

### Packet coverage artifacts

`docs/packets/registry/*.yaml` (opcode + fname + provenance),
`docs/packets/ida-exports/*.json` (the `functions` map `evidence pin` resolves
against), `docs/packets/audits/<version>/<Writer>.{json,md}` (the third required
artifact), `docs/packets/evidence/<version>/*.yaml`, and the generated
`STATUS.md` / `status.json`.

---

## 2. Decisions carried from design.md that shape the plan

1. **The client sends from ANY client, not just the controller.** `CMob::ApplyControl`
   has no controller test and none at its `CMob::Update` call site. That is what
   makes design §2's hybrid arbitration necessary — the PRD's literal
   "requester must be the current controller" gate would no-op the common case,
   because Atlas elects controllers by least-loaded count, not proximity
   (`getControllerCandidate`, `processor.go:306`).
2. **`bPickUpDrop` alone fires the packet.** The `firstAttack` gate in
   `atlas-monsters` is therefore load-bearing, not belt-and-braces: without it,
   drop-picking mobs would turn aggressive.
3. **No version divergence.** All ten binaries encode the same two `Encode4`.
   The codec takes no `MajorAtLeast` gate; adding one would be noise.
4. **Ten routed versions, not six.** The `n-a` on v48/v61/v72/v79 was a CSV blind
   spot (the ops CSVs have no column before GMS v83), not a client absence — all
   four binaries have `CMob::ApplyControl`. Those four registries have **no
   `AUTO_AGGRO` entry at all** today; the plan adds them. Nothing ends up `n-a`,
   so no `feature-na-evidence.yaml` entry is written.
5. **v84 opcode 189 → 194.** Confirmed against the v84 binary (`0x684492`), and
   adjacent to `MOB_DROP_PICKUP_REQUEST` = 195, which task-092 corrected the same
   way. No other serverbound entry in `gms_v84.yaml` claims 189 or 194.
6. **Aggro lease, because nothing else would ever release it.** The decay sweep
   skips monsters with zero damage entries, and auto-aggro deliberately writes
   none (FR-4.5). A permanently-set flag is not inert: `startControl`'s re-pick
   gate and `UseSkill`'s `ControllerHasAggro` gate both read it, so the mob would
   keep making skill decisions against nobody. Stamping the lease inside
   `ControlWithAggro` also closes the same latent hole for Monster Magnet, whose
   `CLEAR_AGGRO`-then-`FORCE_CONTROL` sequence produces exactly this state.
7. **No server-side Dark Sight check.** Task-231 established that attack *arming*
   is gated client-side; a per-packet buff lookup on a once-per-second-per-mob
   path buys nothing observable. Design §5 records the escalation if live testing
   disagrees.

---

## 3. Dependencies and ordering

Two independent chains that only meet at the final gate:

```
Packet chain:   T1 (codec + candidatesFromFName)
                 -> T2 (registry)  -> T3 (templates)
                 -> T4 (export splice, needs live IDA)
                 -> T5 (reports + evidence + matrix)

Service chain:  T6 (information.FirstAttack) ─┐
                T7 (registry lease + SetAggro) ┴-> T9 (Processor.SetAggro) -> T10 (consumer arm)
                T8 (lease release)  [needs T7]
                T11 (producer + mirror controller) -> T12 (rate gate) -> T13 (handler + main.go)
                T14 (lifecycle tests)  [independent]
                T15 (manifest + deploy notes)  [needs T3's opcode table]
```

Hard ordering constraints worth calling out:

- **T1 before T4.** The export's `direction` backfill reads
  `candidatesFromFName`; without the arm the spliced entry lands with an empty
  direction.
- **T3 before T5.** The root pipeline only writes a report for an op the
  template routes; an unrouted op silently produces no `MonsterAutoAggro.json`.
- **T4 before T5.** `evidence pin --ida "CMob::ApplyControl"` resolves the
  address out of the export's `functions` map, and the fname is currently
  absent from all ten exports.
- **T7 before T8 and T9.** Both consume `AggroSummary` / `AggroRefreshedMs`.
- **T11 before T12/T13.** The gate reads `LiveEntry.ControlCharacterId`; the
  handler calls `Processor.SetAggro`.
- **T13 last on the channel side.** `tools/template-symbol-check.sh` fails
  between T3 and T13 (routes referencing an unregistered `AutoAggro` handler).
  That is expected and self-resolving; do not "fix" it by reverting T3.

## 4. External prerequisite (the one genuine dependency)

Task 4 needs a live ida-pro-mcp server with all ten IDBs loaded. During
planning all ten were open and `?ApplyControl@CMob@@IAEXJ@Z` was confirmed
present at the design's addresses on v83, v84 and v48 (spot-checked; design §1.2
recorded all ten and renamed the five that were unnamed). Session ids are **not
stable across restarts** — re-run `idb_list` and match by binary filename.

If IDA is unreachable at execution time, Tasks 1-3 and 6-15 still land; Tasks 4
and 5 block, and the branch is not PR-ready (the coverage-matrix acceptance
criteria are unmet). That is a genuine external blocker, not a producible one —
surface it rather than pinning fabricated evidence.

## 5. Task sizing notes

Every task is at or under the ~6-file guideline and touches one service, with
three deliberate exceptions, all of them the *same mechanical change repeated*
(which the sizing rule explicitly exempts):

- **Task 2** — ten registry YAMLs. One entry per file, from a table.
- **Task 3** — ten seed templates. One JSON object per file, from a table.
- **Task 4** — ten export splices. One tool invocation per file, from a table.
- **Task 5** — ten report copies + ten evidence pins. Same, and it must stay
  whole: the matrix regeneration and its `--check` grade all ten cells at once,
  so splitting it would produce a half-promoted row and a failing gate at the
  split point.

Task 1 spans two module roots (`libs/atlas-packet` and `tools/packet-audit`),
but the `run.go` edit is three lines and splitting it would leave the codec
unlinked — the exact state that makes Task 5 fail with no report.

## 6. Things that are deliberately NOT in scope

- Server-side mob AI, pathing, or a proximity sweep in `atlas-monsters`.
- Mob attack initiation (the existing mob-attack/mob-skill paths carry it once
  aggro is set).
- Re-ingesting `firstAttack` from WZ — `atlas-data` already parses and serves it.
- Changing the damage-driven aggro path or `monster/aggro.go`'s decay constants.
- `template_gms_12_1.json` — task-175 owns it and it has no registry column.
- Renaming `MobDropPickupRequest.mobCrc` to match the new codec's `mobId`
  (the name is a misnomer, but the rename is a separate change).
