# Character Spawn Point Plumbing — Design

Version: v1
Status: Approved
Created: 2026-08-27
PRD: [prd.md](prd.md)
Issue: https://github.com/Chronicle20/atlas/issues/1528

---

## 1. Scope of this document

The PRD fixes a decode-seam defect replicated across eight services. The edits themselves are
mechanical — un-stub an accessor, add a field assignment, add a cast. This design does not
re-litigate them. It resolves the four things that are *not* mechanical:

1. Where the `uint32` → `byte` narrowing lives (§3).
2. Why the existing round-trip tests cannot detect this defect class, and what shape of test can
   (§4). This is the load-bearing decision: the wrong test shape produces a green suite that proves
   nothing.
3. Three places where the PRD's acceptance criteria are not simultaneously satisfiable as written,
   and how each is resolved (§5).
4. What "byte-identical wire output" is provable with, given that no byte-encoding harness exists
   in either affected writer package (§6).

§7 records the verified inventory the plan phase consumes. §8 lists the deviations from the PRD
that these decisions imply, so the plan and the reviewer see them declared rather than discovered.

## 2. Verified starting state

Every claim below was read out of the worktree, not recalled.

- Eight services carry `func (m Model) SpawnPoint() byte { return 0 }` over a `spawnPoint uint32`
  backing field. Verified at `atlas-channel` `character/model.go:240`, `atlas-login`
  `character/model.go:222`, `atlas-query-aggregator` `character/model.go:224`, `atlas-cashshop`
  `character/model.go:211`, `atlas-npc-shops` `character/model.go:208`, `atlas-pets`
  `character/model.go:207`, `atlas-consumables` `character/model.go:213`, `atlas-messages`
  `character/model.go:205`.
- `atlas-character` is the only correct implementation: `character/model.go:137` returns `uint32`.
- A repo-wide grep for `SpawnPoint()` yields exactly three non-test consumers of the eight stubs:
  `atlas-channel/.../socket/writer/character_data.go:47`,
  `atlas-login/.../socket/writer/character_list.go:56`,
  `atlas-query-aggregator/.../character/rest.go:128`. The `byte` → `uint32` return-type change is
  therefore a three-call-site compile break, fully enumerated.
- The wire field is one unconditional byte: `libs/atlas-packet/character/data.go:41`
  (`SpawnPoint byte`), written at `data.go:333` via `w.WriteByte(m.Stats.SpawnPoint)` with no
  version gate around that statement.
- **All eight services already have a `Builder` with `SetSpawnPoint(v uint32)`** — `atlas-channel`
  `builder.go:136`, `atlas-login` `:82`, `atlas-cashshop` `:114`, `atlas-npc-shops` `:114`,
  `atlas-pets` `:111`, `atlas-consumables` `:117`, `atlas-messages` `:109`,
  `atlas-query-aggregator` `:123`. No setter needs to be added for `spawnPoint` anywhere.
- **All eight `Transform`s write `SpawnPoint` except `atlas-pets`**, whose `Transform`
  (`character/rest.go:106`) writes only `Id`, `X`, `Y`, `Stance`. This is new information relative
  to the PRD, which discussed only pets' `Extract`. It is what forces decision §5.1.

## 3. Narrowing placement

Confirmed as the PRD specifies (FR-1/FR-2), and worth stating why rather than treating it as
settled by fiat.

`Model.SpawnPoint()` returns `uint32` in all eight services. Narrowing to `byte` happens at the two
wire call sites with an explicit `byte(...)` cast.

The alternative — keeping the `byte` return and narrowing inside the accessor — was rejected
because it makes truncation a property of the *model* rather than of the *wire*. Under that
alternative `atlas-query-aggregator`'s REST re-serve (`rest.go:128`), which is not a wire path,
would silently truncate any spawn point above 255 on its way back out as JSON. The model type must
match its backing field and its upstream source of truth; the one layer that genuinely requires a
byte is the one that should say so.

The narrowing is lossy above 255 and nothing prevents it. That is a pre-existing property of the
wire format, not something this task introduces, and it is not this task's job to add clamping or
an error path — the PRD explicitly holds the wire format constant. §4.3 pins the truncation
behavior in a test so it is documented rather than latent.

## 4. Test shape — the central decision

### 4.1 Why the existing tests are blind to this defect

`atlas-npc-shops/.../character/rest_test.go` is the representative case. It builds a `RestModel`
with `X: 10, Y: 12, Stance: 14, SpawnPoint: 11`, then asserts:

```go
m,   _ := Extract(rm)
rm2, _ := Transform(m)
m2,  _ := Extract(rm2)
reflect.DeepEqual(m, m2)
```

That test passes today, with `Extract` dropping `x`, `y`, and `stance` on the floor and
`SpawnPoint()` hardcoded to zero. It passes because it asserts **idempotence of `Extract∘Transform`**,
and a field that `Extract` drops is zero on *both* sides of the comparison. A dropped field is
precisely the fixed point such a test cannot see.

This is the defect class the task exists to close, so a new test written in the same shape would
close nothing. The test doc comment even claims it proves "every field set by `Extract` survives" —
a true statement that is vacuous for fields `Extract` never sets.

### 4.2 The shape that works: RestModel-anchored value assertions

Every new assertion is anchored to a `RestModel` literal with a **non-zero, type-distinct** value
and compares against that literal, never against another derived model:

```go
rm := RestModel{ /* ...populated fields..., */ SpawnPoint: 11 }

m, err := Extract(rm)
// ...
if got := m.SpawnPoint(); got != 11 {
    t.Errorf("SpawnPoint() = %d, want 11", got)
}
```

`RestModel` is an exported plain struct and the declared input type of `Extract`. Constructing one
directly is using the function's real API, not a test-only constructor — the CLAUDE.md prohibition
is on `*_testhelpers.go` shortcuts into unexported state, which this is not. The `Model` side of
each test still uses the package `Builder` (§5.3).

Non-zero values are mandatory, not stylistic: `SpawnPoint: 0` would pass against the stub.

### 4.3 Narrowing tests at the two wire call sites

`atlas-channel`'s `BuildCharacterData` is directly unit-testable and returns the struct — the
existing `socket/writer/character_data_test.go` already calls it with a `character.NewBuilder()...`
fixture. Two assertions go there:

- `SetSpawnPoint(7)` ⇒ `cd.Stats.SpawnPoint == 7` — proves the value reaches the wire struct.
- `SetSpawnPoint(256)` ⇒ `cd.Stats.SpawnPoint == 0` — pins the documented truncation from §3, so a
  future reader finds the behavior asserted rather than inferred.

`atlas-login`'s `character_list.go:56` feeds `packetmodel.NewCharacterStatistics(...)`, whose
`SpawnPoint()` accessor (`libs/atlas-packet/model/character_statistics.go:86`) is exported. The
equivalent pair of assertions goes against the built statistics value. `atlas-login`'s writer
package has no existing test file for this path; adding one is in scope.

## 5. Resolved tensions with the PRD

Three of the PRD's acceptance criteria are not jointly satisfiable with its own functional
requirements. Each was put to the user and decided.

### 5.1 `atlas-pets` — `Transform` also gains `SpawnPoint`

**Problem.** The PRD's acceptance criterion asks for `Extract(Transform(m)).SpawnPoint() ==
m.SpawnPoint()` per service. In `atlas-pets` that is unsatisfiable: `Transform` (`rest.go:106`)
emits only `Id`/`X`/`Y`/`Stance`, so the value is destroyed on the outbound leg regardless of what
`Extract` does. §2 of the PRD lists pets' broader `Extract`/`Transform` gap as a non-goal.

**Decision (user).** Add `SpawnPoint: m.spawnPoint` to pets' `Transform` as well as
`spawnPoint: m.SpawnPoint` to its `Extract`.

**Rationale and blast radius.** Both legs are the same field's plumbing, and a one-sided fix leaves
pets the only service where the seam is still broken in one direction. `character.Transform` in
`atlas-pets` has **zero callers outside its own `rest_test.go`** (verified by grep), so this changes
no live payload. The remaining ~26 fields pets drops in both directions stay out of scope per §2 and
§9(2); this is a `spawnPoint`-only widening, and the plan must not let it drift into a general pets
repair.

**Consequence.** `atlas-pets`' existing `TestTransformRoundTrip` is `Model`-anchored
(`Model{id,x,y,stance}` → `Transform` → `Extract` → `DeepEqual`). It continues to pass unchanged. It
gains a `spawnPoint` value in its fixture so it actually exercises the new assignment.

### 5.2 `atlas-npc-shops` — builder gains `SetX` / `SetY` / `SetStance`

**Problem.** FR-7 requires `x`/`y`/`stance` in npc-shops' `Extract`, and the matching acceptance
criterion requires a test with non-zero values. FR-8 forbids adding `SetX`/`SetY`/`SetStance` to its
builder. But the builder is the only sanctioned way to originate a `Model` with non-zero unexported
fields: the `Builder` struct declares `x`/`y`/`stance` (`builder.go:75-77`) and `Clone`/`Build`
propagate them (`:39-41`, `:148-150`), yet no setter exists to introduce them.

**Decision (user).** Add `SetX(int16)`, `SetY(int16)`, and `SetStance(byte)` to
`atlas-npc-shops/.../character/builder.go`. **This overrides FR-8.**

**Rationale.** The gap is an oversight in the builder, not a deliberate restriction — every sibling
service's builder that carries these fields also sets them, and `Clone`/`Build` already round-trip
them, so the type is already committed to carrying the values. Three one-line setters completing an
existing builder is a smaller anomaly than a test that reaches around the Builder pattern the
guidelines mandate.

**Consequence.** The acceptance-criteria line "no `SetX`/`SetY`/`SetStance` was added to its
builder" is superseded; the reviewer will see setters with no production caller and must not flag
them. Recorded in §8.

### 5.3 Builder vs. `RestModel` literal in test setup

Both are used, for different halves of each test, and the plan should be explicit so a reviewer does
not read the mix as inconsistency:

- **Input to `Extract`** — a `RestModel` struct literal. It is the function's declared parameter
  type and an exported plain struct; there is no builder for it and none should be added.
- **Input to `Transform` / to the writers** — the package `Builder`, per CLAUDE.md. Every service
  already has `SetSpawnPoint`, and after §5.2 npc-shops has the positional setters too, so no test
  needs to reach into unexported fields.

## 6. FR-9 — evidence for unchanged wire output

**Problem.** FR-9 demands `CHARACTER_DATA` and `CHARACTER_LIST` encode byte-identically. Neither
`atlas-channel`'s nor `atlas-login`'s writer test package contains any byte-level encode test — a
grep for a tenant model, an options map, or an `Encode(` call across both `socket/writer/*_test.go`
sets returns nothing. `CharacterData.Encode` (`libs/atlas-packet/character/data.go:123`) needs a
tenant-carrying `context.Context` and an options map, and the acceptance criteria forbid touching
`libs/atlas-packet` — where the byte-level fixtures for this codec already live.

**Decision (user): struct-level pin plus narrowing tests.** The §4.3 assertions on
`cd.Stats.SpawnPoint` and on `CharacterStatistics.SpawnPoint()` are the evidence.

**Why this is sufficient, stated as an argument the reviewer can check rather than a convenience.**
Byte identity for a value `v` follows from two facts, both verifiable in this diff:

1. The only input to the encoder that this task can change is `Stats.SpawnPoint`. Nothing else in
   the diff touches a writer, a codec, a version gate, or an opcode — `libs/atlas-packet` is
   untouched, which the acceptance criteria already require and which `git diff --stat` confirms
   mechanically.
2. With the source column at `0`, `Stats.SpawnPoint` is `0` before the change (hardcoded stub) and
   `0` after (`byte(uint32(0))`). The §4.3 test asserts the post-change value directly.

Given (1) and (2), identical encoder input over an unchanged encoder yields identical bytes. The
byte layout itself remains pinned by the existing tests in `libs/atlas-packet/model/` — which this
task neither modifies nor needs to.

**Rejected alternative.** Standing up tenant-context encode harnesses in two service test packages
and comparing full packet slices against golden blobs captured from the base commit. It is literal
proof, but it builds durable new test infrastructure in two services to observe a constant-zero
byte, and duplicates coverage that already exists one layer down. If the field ever acquires a real
producer, that is the moment to invest in it.

## 7. Edit inventory

Ordered so the compile break in the middle column is introduced and repaired within one unit of
work per service. `atlas-channel` and `atlas-login` are the only two with a wire consumer and should
be done first, so any surprise surfaces on the player-facing paths before the latent copies.

| # | Service | Accessor (`→ uint32`, `return m.spawnPoint`) | `Extract` | Other edits |
|---|---|---|---|---|
| 1 | `atlas-channel` | `character/model.go:240` | add `spawnPoint: m.SpawnPoint` (`rest.go:125`) | `byte(...)` at `socket/writer/character_data.go:47` |
| 2 | `atlas-login` | `character/model.go:222` | add `.SetSpawnPoint(m.SpawnPoint)` (`rest.go:129`) | `byte(...)` at `socket/writer/character_list.go:56` |
| 3 | `atlas-query-aggregator` | `character/model.go:224` | unchanged | drop `uint32(...)` at `character/rest.go:128` |
| 4 | `atlas-cashshop` | `character/model.go:211` | add `spawnPoint: m.SpawnPoint` (`rest.go:128`) | — |
| 5 | `atlas-pets` | `character/model.go:207` | add `spawnPoint: m.SpawnPoint` (`rest.go:97`) | add `SpawnPoint: m.spawnPoint` to `Transform` (`rest.go:106`) — §5.1 |
| 6 | `atlas-npc-shops` | `character/model.go:208` | add `x`, `y`, `stance` (`rest.go:134`) | add `SetX`/`SetY`/`SetStance` to `builder.go` — §5.2 |
| 7 | `atlas-consumables` | `character/model.go:213` | unchanged | — |
| 8 | `atlas-messages` | `character/model.go:205` | unchanged | — |
| — | `atlas-character` | **no change** | — | system of record, already `uint32` |
| — | `libs/atlas-packet` | **no change** | — | wire format is correct |

Test additions, by file:

| File | Assertion |
|---|---|
| `character/rest_test.go` × 7 (channel, login, cashshop, npc-shops, pets, consumables, messages) | `Extract(rm).SpawnPoint() == rm.SpawnPoint` with a non-zero literal |
| `atlas-query-aggregator/.../character/rest_test.go` (**new file**) | same, plus `Transform(m).SpawnPoint == 11` proving the dropped cast preserves `uint32` |
| `atlas-npc-shops/.../character/rest_test.go` | additionally `X`/`Y`/`Stance` survive `Extract`, non-zero and mutually distinct |
| `atlas-channel/.../socket/writer/character_data_test.go` | `SetSpawnPoint(7)` ⇒ `cd.Stats.SpawnPoint == 7`; `SetSpawnPoint(256)` ⇒ `== 0` |
| `atlas-login/.../socket/writer/character_list_test.go` (**new file**) | same pair against `CharacterStatistics.SpawnPoint()` |

`atlas-query-aggregator` has no `character/rest_test.go` today; it is the one service whose fix is
observable in a live response value (§5 of the PRD), so it gets one.

## 8. Deviations from the PRD

Declared here so the plan carries them forward and the reviewer does not report them as defects.

1. **FR-8 is overridden.** `SetX`/`SetY`/`SetStance` **are** added to `atlas-npc-shops`' builder
   (§5.2, user decision). The corresponding acceptance-criteria bullet is superseded. The setters
   will have no production caller.
2. **`atlas-pets`' `Transform` is modified**, which §2 of the PRD lists under a non-goal (§5.1, user
   decision). Strictly `SpawnPoint` only; pets' other ~26 dropped fields remain out of scope.
3. **FR-9's byte-identity is proved structurally, not by byte comparison** (§6, user decision). No
   golden packet fixture is captured and no encode harness is built.

Unchanged from the PRD: no producer for `spawnPoint` is added and none is filed; the eight model
copies are not deduplicated; the `Rank`/`RankMove`/`JobRank`/`JobRankMove` stubs and
`atlas-messages`' `Stance()` stub are untouched; `atlas-character` and `libs/atlas-packet` are
untouched.

## 9. Risks

- **The compile break is the safety net, and it is complete.** Three call sites, all enumerated in
  §2 by grep. `go build ./...` across the monorepo catches any fourth. The risk is not that a caller
  is missed — it is that a caller is "fixed" by adding a cast without thinking; §4.3's assertions
  exist so the two wire casts are exercised rather than assumed.
- **Uniform edits across eight near-identical files invite copy-paste drift** — wrong package, wrong
  field name, a `byte(...)` cast left in place. Per-service `go build`/`go test` before moving on,
  and a final flagless `tools/verify.sh`.
- **A green suite that proves nothing** is the real failure mode here (§4.1). Every new assertion
  must use a non-zero value; a reviewer should treat any zero-valued `spawnPoint` fixture in a new
  test as a finding.
