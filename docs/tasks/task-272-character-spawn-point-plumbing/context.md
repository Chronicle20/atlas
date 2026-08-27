# Context — task-272 Character Spawn Point Plumbing

Companion to `plan.md`. Records the verified facts the plan was written from, the
decisions it inherits, and the two places a reviewer will otherwise flag correct work
as a defect.

## The defect in one paragraph

Eight services mirror `atlas-character`'s character model. Each declares a
`spawnPoint uint32` backing field, a `RestModel.SpawnPoint uint32` JSON:API field, and
a builder `SetSpawnPoint(uint32)` — and then defines
`func (m Model) SpawnPoint() byte { return 0 }`. Four of the eight additionally never
assign `spawnPoint` in `Extract`. Two of the stubs feed player-facing packets
(`CHARACTER_DATA`, `CHARACTER_LIST`); a third launders the constant zero back into a
JSON:API response other services read as data. `atlas-character` itself is correct and
is not touched.

The persisted column is always `0` — `UpdateSpawnPoint`
(`services/atlas-character/atlas.com/character/character/administrator.go:284`) has zero
callers repo-wide — so the observable wire output does not change. That byte-identical
outcome is the safety property, not a coincidence.

## Key files

| Path | Role |
|---|---|
| `services/atlas-{channel,login,query-aggregator,cashshop,pets,npc-shops,consumables,messages}/atlas.com/*/character/model.go` | the eight stubbed accessors |
| `.../character/rest.go` | `Extract` (inbound seam) and `Transform` (outbound) |
| `.../character/builder.go` | `SetSpawnPoint(uint32)` already present in all eight |
| `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:47` | wire consumer #1 — `CHARACTER_DATA` |
| `services/atlas-login/atlas.com/login/socket/writer/character_list.go:56` | wire consumer #2 — `CHARACTER_LIST` |
| `services/atlas-query-aggregator/.../character/rest.go:128` | REST re-serve consumer — the laundered zero |
| `libs/atlas-packet/character/data.go:41`, `:333` | `SpawnPoint byte`, `w.WriteByte`, no version gate — read-only |
| `libs/atlas-packet/model/character_statistics.go:86` | `CharacterStatistics.SpawnPoint() byte` — read-only |

Module roots (each `go build`/`go test` cwd): `services/<svc>/atlas.com/<module>`, where
the module names are `channel`, `login`, `query-aggregator`, `cashshop`, `pets`, `npc`
(note: `atlas-npc-shops`' module directory is `npc`, not `npc-shops`), `consumables`,
`messages`.

## Verified facts the plan depends on

- Exactly three non-test consumers of the eight stubs exist repo-wide, all enumerated
  above. The `byte` → `uint32` return-type change is therefore a fully-bounded compile
  break.
- Every service is a separate Go module, so each task's compile break is contained. Task
  ordering is a convenience (wire-facing services first, per design §7), not a
  correctness constraint.
- All eight builders already have `SetSpawnPoint(uint32)`. No `spawnPoint` setter needs
  adding anywhere.
- `atlas-pets` is the only service whose `Transform` also drops `spawnPoint` (it emits
  `Id`/`X`/`Y`/`Stance` only). `character.Transform` there has zero callers outside its
  own `rest_test.go`.
- `atlas-npc-shops`' `Builder` declares `x`/`y`/`stance` and `Build` copies them, but no
  setters exist — the reason a non-zero positional fixture cannot be built the sanctioned
  way today.
- `libs/atlas-packet/model/character_list_entry.go:37` exports `Statistics()`, so the
  new `atlas-login` writer test reads the packet value through an existing getter.
- `location.GetField` in the new `atlas-login` writer test fails fast (no MAPS base URL
  configured → invalid request), logs a warning, and renders `mapId = 0`. It does not
  hang and does not panic. `requests` uses `tenant.FromContext`, not `MustFromContext`.
- A bare `character.Model{}` panics in `RemainingSp()` in `atlas-channel` and
  `atlas-login`; every fixture must set `Sp` to a parseable string (`"0"`).

## Why the existing tests never caught this

Every affected package's round-trip test asserts idempotence of `Extract∘Transform`
(build a fixture, extract, transform, extract again, `reflect.DeepEqual`). A field that
`Extract` drops is zero on **both** sides of that comparison, so it is exactly the fixed
point such a test cannot see. `atlas-npc-shops`' test passes today with `X: 10, Y: 12,
Stance: 14, SpawnPoint: 11` in the fixture and all four dropped or masked.

Consequently every assertion this plan adds is **anchored to the `RestModel` literal or
the builder input**, never to a second derived model, and every fixture value is
non-zero. A zero-valued `spawnPoint` in a new test is a review finding.

`atlas-cashshop`'s existing fixture carried `SpawnPoint: 0` and is changed to `11` for
this reason (Task 4).

## Decisions inherited from the design — a reviewer must not flag these

1. **`SetX`/`SetY`/`SetStance` are added to `atlas-npc-shops`' builder with no production
   caller.** Design §5.2, user decision, explicitly overriding PRD FR-8 and the matching
   acceptance-criteria bullet. The setters complete a builder that already carries and
   copies the fields.
2. **`atlas-pets`' `Transform` is modified**, which PRD §2 lists under a non-goal. Design
   §5.1, user decision: a one-sided fix would leave pets the only service still broken in
   one direction. Strictly `spawnPoint`; pets' other ~26 dropped fields stay dropped.
3. **FR-9's byte-identity is proved structurally, not by byte comparison.** Design §6,
   user decision. No golden packet fixture is captured and no tenant-context encode
   harness is built in either writer package. The argument is: `libs/atlas-packet` and
   `atlas-character` are untouched (Task 9 Step 2 checks this mechanically), and the only
   encoder input this diff can move is `Stats.SpawnPoint`, which the new tests pin at the
   post-change value. Identical input over an unchanged encoder yields identical bytes.

## Explicitly out of scope

No producer for `spawnPoint` is added and no follow-up task is filed for one (user
decision) — the value stays `0`. The eight model copies are not deduplicated. The
`Rank`/`RankMove`/`JobRank`/`JobRankMove` `return 0` stubs in seven services are
untouched. `atlas-messages`' `Stance() byte { return 0 }` stub
(`character/model.go:225`) is untouched. `atlas-pets`' broader ~26-field
`Extract`/`Transform` gap is untouched. `atlas-character` and `libs/atlas-packet` do not
appear in the diff.

## Task sizing

Nine tasks: one per service (Tasks 1-8), plus an evidence-only acceptance sweep (Task 9)
that owns the flagless `tools/verify.sh` run. No task exceeds five files or crosses a
service boundary. `atlas-consumables` and `atlas-messages` are two-file accessor-only
tasks and were deliberately **not** merged, because merging them would cross a service
boundary for no saving. Nothing is deliberately oversized.
