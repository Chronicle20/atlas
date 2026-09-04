# Experience Gain Distribution Mapping — Test Coverage — Design

Version: v1
Status: Draft
Created: 2026-09-04
PRD: [prd.md](prd.md)

---

## 1. Scope and shape of the change

Three files in `services/atlas-channel`, no cross-service seam, no wire change:

| File | Change |
|---|---|
| `atlas.com/channel/kafka/message/character/kafka.go` | Doc comment above the `ExperienceDistributionType*` block; new `AllExperienceDistributionTypes` slice. |
| `atlas.com/channel/kafka/consumer/character/consumer.go` | New pure `buildIncreaseExperienceConfig`; `announceExperienceGain` reduced to a call plus the existing announce. |
| `atlas.com/channel/kafka/consumer/character/consumer_test.go` | `TestBuildIncreaseExperienceConfig` (table) + `TestExperienceDistributionTypeExhaustiveness`. |

Current state confirmed in the worktree:

- `consumer.go:367-405` — the 14-arm `if/else` chain, inline inside the innermost closure of `announceExperienceGain`, building `c` immediately before the 17-argument `charpkt.CharacterStatusMessageOperationIncreaseExperienceBody(...)` call at `consumer.go:407-412`.
- `kafka.go:110-123` — the 14 untyped string constants, inside the same `const` block as the `StatusEventType*` and `StatusEventActorTypeCharacter` constants.
- `kafka.go:173-177` — `ExperienceDistributions{ExperienceType string; Amount uint32; Attr1 uint32}`.
- `socket/model/experience_status.go:3-21` — `IncreaseExperienceConfig`, 19 fields, all `bool` / `int32` / `byte`. **The struct is comparable** — no slices, maps, funcs, or pointers — which the test design depends on (§4.2).
- `consumer_test.go:54-200` — `TestSnapshotHandlers`, the table-driven precedent this task matches.

The refactor is behavior-preserving by construction: the chain arms move into `switch` cases with the same right-hand sides, and the fall-through-on-unknown semantics of a chain with no trailing `else` are exactly the semantics of a `switch` with no `default`.

## 2. Where the pure function lives

**Decision: unexported in the existing consumer package (`consumer.go`), per FR-1.**

Alternatives weighed:

- *`socket/model`, next to `IncreaseExperienceConfig`.* Superficially attractive — the config is that package's type. Rejected: it would make `socket/model` import `kafka/message/character`, pointing a socket-layer package at a Kafka message package. That is a new dependency edge in the wrong direction for one function, and CLAUDE.md's "never call another layer's internals across a service boundary" rule cuts the same way inside the service.
- *A new file `experience.go` in the consumer package.* Neutral on coupling. Rejected as scope creep: the function is ~40 lines and its only caller is 15 lines below it in `consumer.go`. Splitting it out separates the mapping from its sole call site for no test benefit — the test can reach an unexported symbol in the same package either way.
- *Exported.* Rejected: no cross-package caller exists, and the test is in `package character` (the existing test file's package clause), so unexported is sufficient.

Signature, verbatim from FR-1:

```go
func buildIncreaseExperienceConfig(ds []character2.ExperienceDistributions) model2.IncreaseExperienceConfig
```

The existing import aliases in `consumer.go` (`character2 "atlas-channel/kafka/message/character"` at line 10, `model2 "atlas-channel/socket/model"` at line 19) already provide both names; no import changes.

`announceExperienceGain` keeps its four-level currying, its `session.Announce` call, its positional 17-argument splat, and its error log verbatim. Its innermost closure becomes:

```go
return func(s session.Model) error {
    c := buildIncreaseExperienceConfig(distributions)
    err := session.Announce(...)(s)
    ...
}
```

FR-4 (nil/empty → zero config) needs no code: `for range nil` is zero iterations and `c` starts at the zero value. It is asserted (FR-19), not implemented.

## 3. How the mapping is expressed

**Decision: a `switch` on `d.ExperienceType`, one case per constant, each with a client-line comment (FR-5, FR-7).**

The alternative worth naming is a **package-level `map[string]func(*model2.IncreaseExperienceConfig, character2.ExperienceDistributions)`** — an applier table. It has one real advantage: exhaustiveness could be checked against the map's keys at runtime without a hand-maintained slice, which is strictly less duplication than the FR-21 registry. It loses on three counts:

1. It turns 14 straight-line assignments into 14 closures over a mutable pointer, which is harder to read at the point where the trap lives — and readability *is* the deliverable here (FR-7, FR-24).
2. It defeats the PRD's "byte-identical field assignments" guarantee: a map's arms are no longer visibly the same statements as the chain's, so behavior preservation becomes a review argument rather than a diff.
3. It removes nothing. The registry slice is still wanted at `kafka.go` for FR-24's documentation purpose, next to the constants where a producer-side developer reads them — not in the consumer package where a map would live.

So: `switch`, matching FR-5 exactly. `PLAY_TIME` and `PARTY` each keep both of their assignments in one case. No `default` clause — FR-6's silent fall-through is the behavior, and adding a `default:` that logs would be a behavior change (and would require a logger, breaking FR-2's purity).

The `ITEM` case carries the task-277 trap comment explicitly, e.g.:

```go
// ITEM renders the "Equip Item Bonus EXP" modifier line on the right side --
// NOT "experience that came from an item". It does not touch the primary
// amount. Choosing this for an item-sourced EXP award renders
// "You have gained experience (+0)". See task-277,
// docs/tasks/task-277-stored-exp-items/bug-redeem-renders-as-item-bonus.md.
```

The remaining 13 cases carry a one-line client-line comment sourced from the §4.4 PRD table, which itself agrees with the field comments already on `IncreaseExperienceConfig` (`experience_status.go`). Where the two differ in wording, the struct's existing comment wins — it is the older, live-verified text.

## 4. Test architecture

### 4.1 Two test functions, not one

`TestBuildIncreaseExperienceConfig` owns the mapping table (FR-13 through FR-20). `TestExperienceDistributionTypeExhaustiveness` owns the registry cross-check (FR-22, FR-23). Splitting them keeps a failure diagnosable at a glance: a wrong field is a mapping bug, a missing type is a coverage bug, and they have different fixes.

The exhaustiveness check needs the set of types the mapping table covers, so the table must be reachable from both. It becomes a package-level `var` in the test file rather than a local inside the test func:

```go
type distributionMappingCase struct {
    name  string
    types []string // distribution types this case covers; drives exhaustiveness
    given []character2.ExperienceDistributions
    want  model2.IncreaseExperienceConfig
}

var distributionMappingCases = []distributionMappingCase{ ... }
```

`types` is declared per case rather than derived from `given`, so that multi-distribution and unknown-type cases (which contain types they are not the coverage owner of, or types that are not real) can declare `nil` and contribute nothing to the coverage set. Exactly the 14 single-type cases carry a one-element `types`.

### 4.2 Assertion strategy

`IncreaseExperienceConfig` is comparable (§1), so each case asserts with a single struct equality:

```go
got := buildIncreaseExperienceConfig(tc.given)
if got != tc.want {
    t.Errorf("config mismatch\n got: %+v\nwant: %+v", got, tc.want)
}
```

This satisfies FR-15's "every other field at zero" requirement for free and with no maintenance cost: `want` is written as a struct literal naming only the populated fields, and every unnamed field is zero by definition. A field-by-field comparison, or a `reflect.DeepEqual`, would be strictly worse — the former needs 19 lines per case and must be extended when a 20th field is added, the latter is slower and gives a worse diff for a comparable struct.

Consequence worth stating: **adding a field to `IncreaseExperienceConfig` does not break these tests** (the new field is zero on both sides). That is correct — this suite asserts the distribution mapping, not the struct's shape. A new field only matters once a distribution writes to it, at which point FR-22's registry check fires.

### 4.3 Value discipline (FR-14, FR-15)

Each of the 14 single-type cases uses a distinct non-zero `Amount`. Constraint the PRD does not spell out but the design must: five fields are `byte` (`MobEventBonusPercentage`, `PlayTimeHour`, `QuestBonusRate`, `PartyBonusEventRate`, plus `PartyBonusPercentage` which no type writes). Their cases' amounts must be distinct **and** ≤ 255, or the narrowing conversion of FR-11 truncates and the case asserts a truncated value — technically still correct, but it obscures the mapping under an unrelated concern.

Scheme: `int32`-target cases use large distinct values (`1000`, `2000`, `3000`, …); `byte`-target cases use small distinct values (`11`, `22`, `33`, …). `Attr1` for `PLAY_TIME` and `PARTY` takes a value distinct from that case's `Amount` and from every other case's, satisfying FR-14 — an `Amount`/`Attr1` swap in either arm produces a mismatch on both fields.

`MONSTER_EVENT` and `PLAY_TIME` both write `MobEventBonusPercentage` (FR-10); their cases use different byte values so a case that ran the wrong arm fails.

FR-11's overflow behavior is deliberately not exercised. The PRD puts it out of scope, and a truncation case would document Go's conversion rules rather than this mapping.

### 4.4 Multi-distribution cases (FR-16 – FR-20)

Five cases, `types: nil`:

| Case | Given | Asserts |
|---|---|---|
| `WhiteAndChat_PrimaryAwardShape` | `WHITE`, `CHAT` same amount | `Amount` set, `White=true`, `InChat=true` — the shape `AwardExperience` emits for a `showEffect` award |
| `PrimaryPlusBonuses_Accumulate` | `WHITE`, `PARTY`, `ITEM` | primary amount, `PartyBonusExp`/`PartyBonusEventRate`, `ItemBonusEXP` all independently set |
| `WhiteThenYellow_LastWins` | `WHITE`, then `YELLOW` w/ different amounts | `Amount` = the `YELLOW` amount, `White=false` (FR-8, FR-9) |
| `EmptySlice_ZeroConfig` | `nil` | zero config (FR-4) |
| `UnknownType_Ignored` | a bogus string + `WHITE` | only `WHITE`'s effect present (FR-6) |

`WhiteAndChat_PrimaryAwardShape` uses one amount for both entries because that is what the producer emits (`atlas-character/.../processor.go:787-794`); the case pins the observed shape, not a hypothetical one.

`UnknownType_Ignored` uses a string that is deliberately near-miss (e.g. `"EQUIP_ITEM"`) rather than `"GARBAGE"`, so it reads as the realistic failure — a producer emitting a type this consumer does not know.

### 4.5 Exhaustiveness mechanism

**Decision: hand-maintained `AllExperienceDistributionTypes` slice + a test that cross-checks it against the mapping table in both directions (FR-21 – FR-23).**

Alternatives weighed:

- *`golangci-lint`'s `exhaustive` linter.* The natural Go answer, and it would need no registry. Rejected because it only works on a **defined type** (`type ExperienceDistributionType string`) with typed constants. These constants are untyped strings assigned to a `string` struct field on a published Kafka message shape; introducing a defined type would touch `ExperienceDistributions`, its JSON round-trip, and every producer/consumer reference — including `skill/handler/heal/heal.go:224`. That is a wire-adjacent refactor the PRD's §5 explicitly forecloses.
- *Reflection over the `const` block.* Not possible — Go constants are erased at compile time and are not reachable via `reflect` or any runtime facility. A `go:generate` AST walk over `kafka.go` could produce the slice mechanically, but it adds a generator, a generated file, and a CI freshness check to guard 14 lines. Rejected as disproportionate; the slice sits three lines below the constants and a reviewer adding a constant sees it.
- *Chosen: the slice.* Its whole value is that it is adjacent to the constants and is what the test iterates. The failure mode it does not catch — a developer adding a constant and updating neither the slice nor the table — is unguarded, and that is an accepted limit, stated here so nobody assumes otherwise. The mitigation is proximity plus the FR-24 doc comment, which a constant-adding diff cannot avoid touching.

The test:

```go
func TestExperienceDistributionTypeExhaustiveness(t *testing.T) {
    covered := map[string]string{} // type -> case name that covers it
    for _, tc := range distributionMappingCases {
        for _, dt := range tc.types { ... first-wins, or t.Fatalf on duplicate ... }
    }
    // FR-22: every registered type has a case
    // FR-23: every covered type is registered
}
```

Both directions report the offending type name in the failure message, per FR-22/FR-23.

Duplicate coverage (two cases both claiming `types: []string{"ITEM"}`) is treated as a test-authoring error and fails loudly. It is cheap to detect and prevents a silently-shadowed case.

### 4.6 The two acceptance-criteria mutation checks

The PRD requires observing failure, not asserting it in code:

1. Add a fake constant to `AllExperienceDistributionTypes` → `TestExperienceDistributionTypeExhaustiveness` fails naming it; remove a real one → the reverse check fails naming it.
2. Change the `ITEM` arm to write `c.Amount` → the `Item` case fails on both `Amount` and `ItemBonusEXP`.

These are temporary local edits, run, observe, revert. The plan phase must schedule them as explicit verification steps with the failure output recorded in `progress.md` — they are the only evidence that the suite would have caught task-277, and a claim without the captured output is exactly the "verified from a partial run" this repo forbids.

### 4.7 Pre-refactor equivalence (FR-3, FR-5, acceptance "tests pass against the pre-refactor implementation")

Order of work, which the plan should preserve: write the tests against the **existing inline chain** first — extracting only `buildIncreaseExperienceConfig` with the chain copied verbatim into it — and confirm green. Then convert the chain to a `switch` and confirm green again, unchanged tests. Two commits, two green runs. That sequencing is what makes "behavior-preserving" evidence rather than assertion; doing the extraction and the conversion in one step leaves no run that distinguishes them.

## 5. Documentation placement

FR-24/FR-25's comment goes **above the `ExperienceDistributionType*` constants inside the existing `const` block** in `kafka.go`, not on the `ExperienceDistributions` struct and not in a separate markdown doc. The trap is picked at the constant, so the warning belongs at the constant, in a place `go doc` and every editor hover surfaces.

Content: the §4.4 per-type field-and-client-line table in comment form, plus the FR-25 sentence — only `WHITE`, `YELLOW`, and `CHAT` set the primary "You have gained experience" amount; every other type is a secondary modifier line rendered in addition to it.

`AllExperienceDistributionTypes` is declared as a package-level `var` immediately after the `const` block closes (a `var` cannot live inside a `const` block), in declaration order, with a short comment stating its purpose is exhaustiveness enforcement and that a new constant must be added to it.

Duplication is accepted between three places: the constant comment (`kafka.go`), the switch-arm comments (`consumer.go`), and the struct field comments (`experience_status.go`). Each serves a different reader at a different moment — the producer choosing a type, the maintainer reading the mapping, the packet author reading the config. Consolidating to one location would remove the comment from two of the three moments where it prevents the bug.

## 6. Risks and non-risks

- **Transcription risk during extraction.** The one real risk: 14 arms hand-moved into cases, with an easy off-by-one between `Amount` and `Attr1` or between two same-typed fields. Mitigated by §4.7's ordering — the tests exist and are green before the `switch` conversion — and by §4.2's whole-struct assertion, which fails on a stray write to any field.
- **Not a risk: behavior change at runtime.** No caller, signature, log, or announce argument changes; the only new symbol reachable outside the consumer package is a `var` nothing else reads yet.
- **Not a risk: multi-tenancy or packet versioning.** The function is tenant-agnostic and version-agnostic. `PARTY_RING` / `CAKE_PIE` are v95+ *at the wire*, but the config field is set unconditionally today and stays that way; the tests assert the config, never the encoded bytes.
- **Test flake surface: none.** Pure in-memory, no Docker, Kafka, DB, session, goroutine, or clock. The existing `TestSnapshotHandlers` in the same file touches a package-level snapshot registry; the new tests share no state with it and impose no ordering requirement.

## 7. Verification

- `go build ./...` and `go test ./...` in `services/atlas-channel`.
- The two §4.6 mutation checks, run and their failure output recorded.
- Flagless `tools/verify.sh` exits 0 before the branch is called done.
- `backend-guidelines-reviewer` over the changed Go files, plus a `task-reviewer` pass, before the PR opens.
