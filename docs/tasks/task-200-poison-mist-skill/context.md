# Task 200 — Poison Mist: Implementation Context

Companion to [`plan.md`](plan.md). Everything here was verified against source at plan time; each claim carries a `file:line` citation so it can be re-checked rather than trusted.

---

## 1. What already exists (and must not be rebuilt)

Atlas owns most of the mist machinery already. Only the player-cast side is missing.

| Piece | Where | State |
|---|---|---|
| Mist registry (tenant-scoped, in-memory) | `services/atlas-maps/atlas.com/maps/mist/registry.go` | Complete. `Add`/`Remove`/`AllByTenant`/`UpdateLastTick`/`GetTenants`. |
| Mist model (immutable + Builder) | `services/atlas-maps/atlas.com/maps/mist/model.go` | Complete. Gains 4 fields in Tasks 3–4. |
| Mist lifecycle processor | `services/atlas-maps/atlas.com/maps/mist/processor.go` | `Create` / `Destroy`, both emitting. Gains kind normalization in Task 3. |
| Mist tick task (1 Hz) | `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go` | Character-disease branch only. Gains the MONSTER branch in Tasks 7–8. |
| `COMMAND_TOPIC_MIST` / `EVENT_TOPIC_MIST` | `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go` | Complete. Gains 4 additive fields. |
| Only producer of `CREATE` today | `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1067` `buildMistCreateBody` | Monster AREA_POISON. Behavior must not change. |
| `MIST_CREATED` → `AffectedAreaCreated` | `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go:104` | Complete. Reads two values off the event instead of consts in Task 4. |
| `AffectedAreaCreated` / `Removed` writers in all 11 seed templates | committed in `ae3341511` (#1226, task-165) | **Verify, do not re-add.** |
| Monster `APPLY_STATUS` with `PLAYER_SKILL` DoT | `services/atlas-channel/atlas.com/channel/monster/producer.go:13`; consumed at `services/atlas-monsters/.../kafka/consumer/monster/consumer.go:108` | Complete. atlas-maps becomes a second producer. |
| Monster rect endpoint client | `services/atlas-channel/atlas.com/channel/monster/requests.go:33` `inMapRectUrl` | Exists in atlas-channel; copied into atlas-maps in Task 6. |
| Skill identity registry | `services/atlas-channel/atlas.com/channel/skill/handler/registry.go` `Register` / `RegisterAttackCast` | Complete. TWO registries, both identity-keyed. `Register`/`Lookup` is the USE_SKILL path (`common.go:201`). `RegisterAttackCast`/`LookupAttackCast` is the ATTACK-packet path (`character_attack_common.go` `attackCastTryApply`) — Poison Mist uses this one; see design.md §4.2 correction note for why the two must stay separate. |

## 2. Verified facts the design rests on

Each of these was re-checked while writing the plan. They are the load-bearing claims; if one turns out false during implementation, stop and re-derive rather than working around it.

- **`FirePoisonMagicianPoisonMist ↔ 2111003` exists in every provisioned version.** All 11 `libs/atlas-constants/skill/version_*_gen.go` files (gms 12/48/61/72/79/83/84/87/92/95 + jms_185) contain exactly 4 references each — forward map, reverse map, set entry, name. Identity dispatch is safe on every tenant.
- **`POISON` magnitude is never read.** `services/atlas-monsters/.../monster/status_task.go:106-113` — `calculatePoisonDamage` returns `m.MaxHp() / uint32(70 - se.SourceSkillLevel())`, clamping the divisor at 1. `calculateVenomDamage` (line 115) is the one that reads its magnitude. Sending a non-zero `dot` would be dead payload.
- **Re-applying `POISON` refreshes, it does not stack.** `services/atlas-monsters/.../monster/builder.go:141-163` — `AddStatusEffect` calls `RemoveStatusEffectByType` for every non-`VENOM` status before appending. VENOM is the only stacking exception (cap 3, earliest-expiry eviction).
- **atlas-monsters already defaults a POISON tick to 1000 ms.** `services/atlas-monsters/.../kafka/consumer/monster/consumer.go:113-122` — when `tickInterval == 0` and the status set contains `POISON` or `VENOM`, it substitutes 1 s. `PlayerMistTickIntervalMs = 1000` makes this explicit rather than relying on the fallback.
- **The monster path hard-codes `TickIntervalMs: 1000` too.** `services/atlas-monsters/.../monster/processor.go:1096`. 1 Hz is the established DoT cadence on both ends of this contract.
- **`ApplyStatusCommandBody`'s key set is fixed at seven keys.** Producer `services/atlas-channel/.../kafka/message/monster/kafka.go:44-52`; consumer `services/atlas-monsters/.../kafka/consumer/monster/kafka.go:53-61`. Byte-identical. The new atlas-maps mirror must match exactly.
- **`Mist.Contains` is inclusive on all four edges.** `services/atlas-maps/.../mist/model.go` — `x >= minX && x <= maxX && y >= minY && y <= maxY`. `Rect()` must produce the same bounds; Task 3's test pins them together.
- **The matrix gaps are real.** `docs/packets/audits/STATUS.md:337` — `SPAWN_MIST × gms_v92` is `❌` at opcode `0x140`; line 340 — `REMOVE_MIST × gms_v92` is `🟡ᶠ` at `0x141`. Every other version is `✅` for both.
- **`AffectedAreaCreated` exposes getters for every wire field** (`libs/atlas-packet/field/clientbound/affected_area_created.go:113-126`), including `SkillDelay()`, `ElemAttr()`, `NType()`, `Phase()` — so Task 4's consumer test can assert on the broadcast packet without decoding bytes.

## 3. Decisions carried from the design (do not re-litigate)

| # | Decision | Why |
|---|---|---|
| D1a | Per-target poison duration = the mist's lifetime | No WZ `dotTime`. Refresh-on-reapply (§2 above) means the expiry is simply pushed forward each tick while the monster stays inside. |
| D1b | Tick interval = `PlayerMistTickIntervalMs` const (1000), not WZ | No WZ `dotInterval`. 1 Hz is already the de-facto cadence on both ends. |
| D1c | `POISON` magnitude = `0` | atlas-monsters never reads it. Resolves OQ-4 — the damage is HP-proportional, neither raw `dot` nor MA-scaled. |
| D2 | `nType = 0` | Its only in-packet semantics is the `== 3` area-buff-**item** test (v95 `@0x437ff8`, v83 `@0x431b66`). Rendering dispatches on `nSkillID`. `nType == 2` is reserved for Smokescreen's `IsSmokeAreaByPoint` — do not use it. |
| D3 | `dwOwnerId` = the casting character id | Stored at `AFFECTEDAREA+0x8` and never read on the create path. Its only readers (`IsSmokeAreaByPoint`, `GetAffectAreaByPoint`, `GetAr01Area*`) are gated on `nType == 2` or a caller-supplied skill id, so a monster-unique-id/character-id numeric collision can never route into an ally check. |
| D4 | Keep the `Disease*` JSON keys and Go field names | They are already the generic status quadruple. Renaming touches a live contract for zero behavioral gain. |
| D5 | `MaxPlayerMistDurationMs = 300_000` (reject, not clamp) | The largest legitimate `time` for `2111003` is 40 s at level 30. 7.5× headroom; can only fire on corrupt or unit-inverted data. |
| — | `prop` (41–70% apply chance) not implemented | Not in the PRD; near-no-op under refresh semantics. Recorded as a decision, not an oversight. The lever is one roll in the MONSTER tick branch. |

## 4. Traps

- **`COMMAND_TOPIC_MONSTER` is shared.** Every registered handler in atlas-monsters unmarshals every message on it. A same-named-but-narrower key in a sibling body produces decode-error spam on unrelated handlers ([[bug_monster_command_topic_shared_handler_unmarshal_collision]] — `UseSkillCommandBody.SkillId` is a `byte`, `ApplyStatusCommandBody.SourceSkillId` a `uint32`). Task 8's `applyStatusBody` mirrors the exact seven keys; a test pins the key set.
- **`skillDelay` is a DRAW DELAY, not a lifetime** ([[bug_affected_area_phase_is_draw_delay_not_lifetime]]). `tStart = get_update_time() + 100*skillDelay`; `CAffectedAreaPool::Update` gates the first draw on it. Any non-zero value hides the mist for that long. Must stay `0`.
- **Do not clamp the player-cast mist's lifetime.** The client computes its own `tEnd = tStart + 1000 * SKILLLEVELDATA::tTime` from its own WZ (v83 `@0x43200f`, v95 `@0x437c95`). A server clamp desynchronises rendering from server authority. The monster path's 60 s `MistDurationCapMs` is safe *there* only because arms 130/131 set no `tEnd` at all.
- **`buff-duration-guard.sh`** fails CI on a seconds-valued `duration` in a `COMMAND_TOPIC_CHARACTER_BUFF` body. The character branch's ms value has been flipped twice historically — leave it alone. The comment block at `mist_tick.go` explaining the reversal of `11e07dfa7` must survive the Task 7 extraction.
- **Regenerate the packet matrix AFTER merging main** — `toolSha` reads git HEAD ([[bug_packet_matrix_toolsha_reads_git_head]]).
- **A `❌` serverbound cell often means "unverified shared codec", not "missing codec"** ([[bug_matrix_redx_unverified_shared_codec]]). For the v92 clientbound cells here, the encoder already models v92 explicitly (`affected_area_created.go:141`), so expect a verification pass — but if the read order genuinely diverges, the codec change is in scope, not a follow-up.
- **atlas-maps' `monster.RestModel.Id` is `json:"-"`** and populated via `SetID` from the JSON:API document — it holds the monster **unique** id as a decimal string. `Extract` in that package is the identity transform, so the tick must `strconv.Atoi` it (atlas-channel does the same at `monster/rest.go:63`).
- **`tools/lint.sh --check` false-fails without nvm** and under cross-worktree golangci-lint lock contention ([[bug_lint_check_false_fails_without_nvm]]). If it fails oddly, check that before chasing a real lint error.

## 5. Data reality (the honest state)

`dot`, `dotInterval`, and `dotTime` **do not exist in any provisioned version's `Skill.wz`** — zero matches across the entire v83-era corpus, not just for `2111003`. They first appear in v1.17-era data, and even there as `common`-block *formula strings* that atlas-data's reader does not walk (it only reads `level`; see [[bug_v95_skill_common_formula_nodes_unparsed]]).

Task 1 implements the parse anyway: additive, zero-defaulted, forward-compatible. After this task all three fields are `0` on every skill on every tenant. **That is the reported data defect FR-1.5 asks for, not a bug introduced here.** A future re-ingest that carries the nodes needs no plumbing change — only a switch in the handler from the D1 constants to the parsed values, gated on non-zero.

What `2111003` actually carries per level on the provisioned corpus:

| level | mpCon | mad | time (s) | prop (%) | lt | rb |
|---|---|---|---|---|---|---|
| 1 | 21 | 32 | 4 | 41 | (-110,-82) | (110,83) |
| 4 | 24 | 38 | 8 | 44 | (-120,-90) | (120,90) |
| 7 | 27 | 44 | 12 | 47 | (-130,-97) | (130,98) |
| 19–21 | 39–41 | 68–72 | 28 | 59–61 | (-170,-127) | (170,128) |
| 30 (max) | 50 | 90 | 40 | 70 | (-200,-150) | (200,150) |

## 6. Task dependency graph

```
Task 1 (atlas-data reader) ──► Task 2 (channel effect hydration) ──┐
                                                                    │
Task 3 (mist kinds contract) ──┬──► Task 4 (render values on event) │
                               ├──► Task 5 (atlas-monsters explicit)│
                               └──► Task 7 (extract tickCharacters) ─┴──► Task 8 (MONSTER branch)
                                                    ▲
Task 6 (GetInMapRect) ──────────────────────────────┘

Task 9 (channel mist producer) ──► Task 10 (poisonmist handler)  [also needs Task 2]

Task 11 (v92 packet verify) — INDEPENDENT, may run in parallel with everything

Tasks 1-11 ──► Task 12 (verification gates + review)
```

Task 11 shares no files with Tasks 1–10 and can be dispatched concurrently. Tasks 3 and 6 are also independent of each other and of the Task 1→2 chain.

## 7. Verification commands (repo root)

```bash
# per-module
for m in atlas-data/atlas.com/data atlas-channel/atlas.com/channel \
         atlas-maps/atlas.com/maps atlas-monsters/atlas.com/monsters; do
  ( cd "services/$m" && go build ./... && go vet ./... && go test -race ./... )
done
( cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./... )

# guards
tools/lint.sh --check
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/buff-duration-guard.sh
tools/skill-job-id-guard.sh

# packet gates (regenerate the matrix AFTER merging main)
packet-audit matrix --check
packet-audit fname-doc --check
packet-audit operations --check
```

No `go.mod` change is expected, so no `docker buildx bake`. If one appears, the bake becomes mandatory for every affected service.

## 8. Key file index

**atlas-data**
- `services/atlas-data/atlas.com/data/skill/reader.go` — `getEffect` at :169; `time` s→ms at :195-198; `lt`/`rb` at :232-236 (insertion point for the DoT reads)
- `services/atlas-data/atlas.com/data/skill/effect/model.go` — `ModelBuilder` :13; `SetLT`/`SetRB` :283-290; `Build() RestModel` :376
- `services/atlas-data/atlas.com/data/skill/effect/rest.go` — `RestModel` :7
- `services/atlas-data/atlas.com/data/skill/reader_test.go` — `TestReader_LT_RB_Present` :2909 (the pattern to copy)

**atlas-channel**
- `skill/handler/registry.go` — `Register` :39, `Lookup` :44
- `skill/handler/common.go` — `UseSkill` :101; identity resolve :110; per-skill dispatch :201
- `skill/handler/mysticdoor/mysticdoor.go` — the closest precedent (seams + position-based external emit)
- `skill/handler/registrations/registrations.go` — blank-import list
- `data/skill/effect/model.go` :129 `LT()`, :134 `RB()`; `rest.go` :10 `RestModel`, :68 `Extract`
- `kafka/message/mist/kafka.go` — event-only today; gains the command side
- `kafka/consumer/mist/consumer.go` — `mistSkillDelay` :92, `mistElemAttr` :98, `mistPhase` :103, `handleMistCreated` :104
- `kafka/message/monster/kafka.go` :44 `ApplyStatusCommandBody`
- `monster/producer.go` :13 `ApplyStatusCommandProvider`; `monster/requests.go` :33 `inMapRectUrl`
- `monster/rest.go` :62 `Extract` (uniqueId via `strconv.Atoi`)

**atlas-maps**
- `mist/model.go` — `Contains` :163, `ShouldTick` :178, `Builder` :192
- `mist/processor.go` :62 `Create`
- `mist/producer.go` :15 `createdEventProvider`
- `mist/registry.go` :75 `Add`, :125 `AllByTenant`, :141 `UpdateLastTick`
- `mist/processor_test.go` :26 `recordingProducer`, :31 `newRecordingProducer`, :56 `newTestMistProcessor`
- `mist/model_test.go` :13 `mkField`
- `tasks/mist_tick.go` — `EnvCommandTopicCharacterBuff` :35; `applyDiseaseCommandProvider` :68; `MistTick` struct :102; `NewMistTick` :116; `runOnce` :148; `processTenant` :162
- `tasks/mist_tick_test.go` :27 `recordingProducer`, :57 `mkTickTenant`, :62 `newTestMistTick`
- `monster/processor.go` — `Processor` iface :14, `CountInMap` :33; `monster/requests.go` :11 `mapMonstersResource`; `monster/rest.go` :11 `RestModel`
- `monster/processor_drain_test.go` — httptest + `MONSTERS_SERVICE_URL` pattern
- `kafka/message/message.go` — `Buffer`, `Emit`

**atlas-monsters**
- `monster/processor.go` — `MistDurationCapMs` :1053, `executeMist` :1057, `buildMistCreateBody` :1067
- `monster/status_task.go` :106 `calculatePoisonDamage`
- `monster/builder.go` :141 `AddStatusEffect`
- `kafka/consumer/monster/consumer.go` :108 `handleApplyStatusCommand`; `kafka/consumer/monster/kafka.go` :53 `applyStatusCommandBody`

**packets**
- `libs/atlas-packet/field/clientbound/affected_area_created.go` — mob arms :57-58, `skillDelay` doc :43-49, getters :113-126, v92 `hasPhase` :141
- `docs/packets/audits/STATUS.md` :337 `SPAWN_MIST`, :340 `REMOVE_MIST`
- `docs/packets/audits/VERIFYING_A_PACKET.md` — the single-cell playbook

## 9. IDB addresses cited by the design

Kept here so they need not be re-derived. Sources: `GMS_v95.0_U_DEVM.exe` (PDB-backed) and `MapleStory_dump.exe` (v83).

| What | v95 | v83 |
|---|---|---|
| `CAffectedAreaPool::OnAffectedAreaCreated` | `0x437ec0` | `0x431e30` |
| `nType` decode → `AFFECTEDAREA+0x4` | `0x437f12` | — |
| `nType == 3` (area-buff **item** arm) | `0x437ff8` | `0x431b66` |
| `AffectedAreaAnimationCreated` | `0x4372c0` | inlined |
| `nSkillID == 130` (mob) | `0x4374a6` | `0x4321cb` |
| `nSkillID == 131` (mob) | `0x43736d` | `0x43206d` |
| `nSkillID == 2111003` arm | `0x437515` → `0x437b40` | `0x431d50` → `0x431f09` |
| `tEnd = tStart + 1000 * tTime` | `0x437c6b`–`0x437c9f` | `0x43200f` |
| `MakeLayer_Fog` | `0x437cd3` | `0x43203e` |
| `dwOwnerId` store `+0x8` | `0x437fb0` | `0x431b16` |
| `tStart = now + 100*skillDelay` | `0x437fa3` | `0x431b50` |
| `nElemAttr` store `+0x30` | `0x437fd9` | `0x431b3b` |
| `nPhase` store `+0x48` (GMS v92+) | `0x437fde` | n/a |
| `IsSmokeAreaByPoint` (`nType == 2`) | `0x434f40` | — |
| `GetAffectAreaByPoint` | `0x4350f0` | — |
| v92 `OnAffectedAreaCreated` | `0x4392a0` | — |
