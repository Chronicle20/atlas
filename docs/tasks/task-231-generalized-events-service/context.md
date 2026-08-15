# task-231 — Implementation Context

Companion to [`plan.md`](plan.md). Read this first if you are picking the task
up cold: it is the map of what exists, what was decided and why, and where the
plan deliberately departs from the design or the PRD.

Sources: [`prd.md`](prd.md) (v1) → [`design.md`](design.md) (§1 records seven
grounding corrections G1–G7 that supersede the PRD where they disagree).

---

## 1. Shape of the work

Forty tasks in eight phases. Phases A, B and C are self-contained and land
first; nothing downstream depends on them being deployed, only merged.

| Phase | Tasks | Service | Lands independently? |
|---|---|---|---|
| A — spawn provenance | 1–4 | `atlas-monsters` | Yes — no consumer yet, all wire fields optional |
| B — voyage identity | 5–7 | `atlas-transports` | Yes — new events have no consumer; G2 means nothing breaks |
| C — rate mappings | 8 | `atlas-rates` | Yes — two map entries |
| D — generic core | 9–20 | `atlas-events` (new) | Yes — verifiable with no event registered |
| E — Crimson Balrog | 21–30 | `atlas-events`, `atlas-channel` | Needs A, B, D |
| F — Anniversary | 31–34 | `atlas-buffs`, `atlas-events` | Needs C, D |
| G — Web UI | 35–38 | `atlas-ui` | Needs D's REST surface |
| H — boundary + gate | 39–40 | repo | Last |

---

## 2. Key existing files, verified while planning

Every path below was read; line numbers are from the branch point.

**`atlas-transports`**
- `transport/processor.go:136-191` — `UpdateRoute`. `AwaitingReturn` (`:149-162`)
  warps en-route characters and emits **nothing**; `InTransit` (`:173-186`) warps
  staging→en-route per channel then emits one `DEPARTED`.
- `transport/model.go:100` — `UpdateState`; `:115-123` `Transition`;
  `:126-143` `timeOfDay` / `materializeBoundary`; `:148-227` `Evaluate` with the
  midnight-crossing branch at `:165-171` and `:207-217`; `:290-297`
  `TripScheduleModel` with its `tripId`.
- `transport/producer.go` — the two existing status providers, keyed on route id.
- `kafka/message/transport/kafka.go` — `StatusEvent[E]`, `ArrivedStatusEventBody{MapId}`.

**`atlas-monsters`**
- `kafka/consumer/monster/kafka.go` — every command body. `killCommandBody` and
  `catchCommandBody` carry the standing warning that this topic fans every
  message to every handler.
- `kafka/consumer/monster/consumer.go:339` — `handleSpawnFieldCommand`;
  `:308` `handleDestroyFieldCommand`; registrations at `:75-80`.
- `monster/model.go:34-56` `Model`; `monster/builder.go:11-63` `Clone` +
  `ModelBuilder`; `monster/registry.go:25-52` `storedMonster`, `:361`
  `CreateMonster`, `:376` `GetMonstersInMap`.
- `monster/kafka.go:59-82` — `statusEvent[E]` and `statusEventFromField`, the
  single constructor every status event goes through.
- `monster/processor.go:216` `Create`, `:1339` `Destroy`, `:1351` `DestroyInField`.

**`atlas-channel`**
- `kafka/consumer/route/consumer.go:62` and `:82` — the type guards that already
  satisfy FR-V6 (design G2).
- `kafka/consumer/map/consumer.go:167` `SpawnForSelf`, with the `routine.Go` run
  from `:188` to `:350`. The `IsBoatInMap` block at `:314` is the precedent for
  a per-map REST lookup on this path; the weather block at `:337` is the
  precedent for announcing `FieldEffectBackgroundMusicBody`.
- `socket/writer/conti_move.go` — `ContiMoveBody(state, subState)`.
- `main.go:778` `ContiMoveWriter`, `:803` `FieldEffectWriter` — both registered.
- `weather/{processor,requests,rest}.go` — the external-client shape to copy.

**`atlas-buffs`**
- `kafka/message/character/kafka.go` — `ApplyCommandBody` (its `Duration`
  comment is the authoritative statement that the unit is milliseconds).
- `character/registry.go:70` `Apply`, `:147` `GetCharacters`, `:162` `Cancel`,
  `:238` `CancelByStatTypes` — the shape `CancelByCorrelation` mirrors.
- `character/processor.go:125` `CancelAll` — the emit-per-removed-buff pattern.
- `kafka/consumer/characterstatus/consumer.go:52` — the LOGIN reaction
  `atlas-events` copies.
- `buff/model.go:92,114` — `MarshalJSON` / `UnmarshalJSON`, where the
  correlation field goes.

**`atlas-rates`**
- `kafka/message/buff/kafka.go:47-76` — `ConversionAdditive/Direct/Fixed` and
  `buffToRateMappings` (three entries today).

**Reference implementations to copy**
- `services/atlas-party-quests/atlas.com/party-quests/definition/` — the whole
  package shape, including `subdomain.go`'s `seeder.Subdomain`.
- `services/atlas-party-quests/atlas.com/party-quests/main.go` — service bootstrap.
- `libs/atlas-outbox/drainer.go:223` and
  `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/store.go:242`
  — the two existing `SELECT … FOR UPDATE SKIP LOCKED` precedents.
- `libs/atlas-database/tenant_scope.go:22` — `WithoutTenantFilter`.
- `services/atlas-ui/src/pages/RewardPoolsPage.tsx` +
  `reward-pools-columns.tsx` + `services/api/reward-pools.service.ts` — the UI
  page/columns/service triple.

---

## 3. Decisions this plan locks in

Design decisions are argued in `design.md`; this lists the ones an implementer
would otherwise be tempted to re-open.

1. **Voyage identity is derived, never stored** (design §7.1, §14 A3). A UUIDv5
   over `(tenant, route, trip, departure date)`. Survives restart, a Redis flush
   of the route registry, and two replicas deriving independently.
2. **The scheduler uses `SKIP LOCKED`, not leader election** (design §5.1, §14 A2).
   Leader election would make the scheduler single-replica, contradicting FR-N6.
3. **Monster tracking is a set, not a counter** (design §9.5, §14 A4). A counter
   is not idempotent under redelivery and cannot tolerate `KILLED` before
   `CREATED`.
4. **Completion is a guarded UPDATE, not a lock.** `RowsAffected == 0` is how a
   racing path learns it lost, which is FR-B20 as a database predicate.
5. **Anniversary uses `EXP_BUFF_RATE` + `ITEM_UP_BY_ITEM`, both
   `ConversionDirect`.** `EVENT_RATE` is rejected because it is in the JMS
   movement-affecting set (design §10.3, evidence table).
6. **The map-entry visual is a query, not a push** (design §14 A1). The push
   variant makes the visual depend on consumer lag; a character walking in
   during lag would see nothing with no second chance.
7. **`DESTROY_BY_SOURCE` is field-scoped** (design §15.6). Global would need a
   new Redis secondary index maintained on every spawn and death.
8. **The BGM is not restored on the monsters-eliminated path** (design §15.4).
   `atlas-data` does not expose Map.wz `info/bgm` — verified: `grep -n "bgm|Bgm"`
   over `services/atlas-data/atlas.com/data/map/*.go` returns nothing outside
   tests. Restoring it would mean hard-coding a guessed string.

---

## 4. Findings from the planning pass

Three things found while reading the tree that are not in `design.md`:

**F-A — There are eleven socket templates, not nine, and five do not route
`ContiMove`.** PRD F3 and design §11 both say "all nine seed templates route it."
`ls services/atlas-configurations/seed-data/templates/` shows **eleven** files.
`grep -l '"ContiMove"'` matches six: `gms_79`, `gms_83`, `gms_84`, `gms_87`,
`gms_95`, `jms_185` — exactly the six versions
`libs/atlas-packet/field/clientbound/conti_move.go` carries verify markers for.
The five that do not (`gms_12`, `gms_48`, `gms_61`, `gms_72`, `gms_92`) are
partial bring-ups generally: `gms_92` routes 135 writers against `gms_83`'s 220,
and `gms_12` is 25 KB against 206 KB. So the absence is a version-bring-up gap,
not something this task introduces or can close (routing an opcode there means
deriving it per version from the IDB — `/bringup-version` work). **Task 30
records this with the verbatim command output** rather than restating F3's claim.

**F-B — `atlas-buffs` can already enumerate every character in a tenant.**
`Registry.GetCharacters(ctx)` (`character/registry.go:147`) is what `ExpireBuffs`
uses for its fleet sweep, so FR-A15's "one command, not one per character" needs
no new index or scan machinery — `CancelByCorrelation` is the same sweep with a
different predicate.

**F-C — `Evaluate` already selects the trip; it just discards it.**
`processStateChange` (`transport/model.go:230`) calls `Evaluate` and keeps only
`.State`. Task 5 widens `Transition` rather than adding a second evaluation
pass, so the state and the trip identity can never disagree about which side of
a boundary they are on.

---

## 5. Tasks deliberately left large

Per the plan-task sizing rule (split above ~6 files or more than one service),
three tasks exceed it on purpose:

- **Task 10 (k8s/ingress/DB registration)** touches ~12 deployment files across
  two overlays. Splitting it would leave the repo in a state where
  `kustomize build` fails, and four of the files are generator-owned — they must
  be regenerated together or the next generator run reverts half the change.
- **Task 15 (occurrence persistence)** defines three entities and their
  administrator in one task because the paired occurrence+transition transaction
  and the guarded completion are the same invariant; a reviewer cannot
  meaningfully accept one without the other.
- **Task 18 (the poller)** carries claim, reclaim, retry and dispatch together
  because the concurrent two-instance test — which PRD §20.3 demands explicitly —
  exercises all four at once.

Every other task is under six files, single-service, and ends in a commit.

---

## 6. Out-of-repo steps a human must do

Neither is producible from this branch; both are called out in Task 10 Step 6.

1. **Create `atlas-events-main` on postgres.home manually.** Main has no wave-0
   create job (`docs/adding-a-new-service.md` §6.1). PR envs are covered by
   `ATLAS_DB_NAMES`.
2. **Flip the GHCR package to Public after the first build.** The first
   `docker buildx bake` push creates
   `ghcr.io/chronicle20/atlas-events/atlas-events` **private**, and the cluster
   pulls anonymously with no `imagePullSecrets` — so the pod sits in
   `ImagePullBackOff` forever while CI is green. Verify with:
   ```bash
   curl -s -o /dev/null -w '%{http_code}\n' \
     'https://ghcr.io/token?scope=repository%3Achronicle20%2Fatlas-events%2Fatlas-events%3Apull&service=ghcr.io'
   ```
   200 = public, 401 = still private.

---

## 7. Things to verify before trusting them

Named here so nobody treats them as settled:

- **The `item_drop` rate-type string.** Task 8 Step 1 requires grepping
  `services/atlas-rates/atlas.com/rates/rate/` for the literal the calculator
  consumes. Do not type `"item_drop"` from this document — a wrong value
  composes into a rate type nothing reads, silently.
- **The route `state` strings.** Task 22's `VoyageUnderway` compares against
  them; read `transport/state.go` rather than guessing `"in_transit"`.
- **The Anniversary buff `sourceId`.** Must not collide with a skill id. Task 33
  Step 4 requires grepping existing producers before choosing.
- **Whether the v83 client renders an icon for `EXP_BUFF_RATE` /
  `ITEM_UP_BY_ITEM`.** Explicitly **unverified** and out of scope (design §10.3).
  Both rates are computed server-side in `atlas-rates`, so a missing icon is
  cosmetic; what must be right is the `GIVE_BUFF` mask position, and that is
  what the design's evidence table establishes.
- **The Postgres test harness.** Task 18's concurrency test needs real Postgres
  — `SKIP LOCKED` is the thing under test and sqlite lacks it. Find the repo's
  existing container harness before writing a new one.
