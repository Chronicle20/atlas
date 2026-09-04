# Experience Gain Distribution Mapping — Test Coverage — Implementation Plan

Task: `task-296-experience-gain-distribution-tests`
Design: [design.md](design.md) · PRD: [prd.md](prd.md)
Module root for every `go build` / `go test` below: `services/atlas-channel/atlas.com/channel`

---

## Ordering contract (design §4.7)

The order below is load-bearing, not cosmetic. Tasks 2 and 3 make the tests
green against the **pre-refactor `if/else` chain**; Task 4 converts that chain
to a `switch` and the *unchanged* tests must still be green. Two green runs on
either side of the conversion are what makes "behavior-preserving" evidence
rather than assertion. Do not merge Task 2 and Task 4.

## Fixed value scheme (design §4.3)

Every single-type case gets a distinct non-zero `Amount`. `int32`-target cases
use thousands; `byte`-target cases use small values ≤ 255 so no case asserts a
truncated value. `Attr1` values are distinct from every `Amount` and from each
other.

| Type | `Amount` | `Attr1` | Expected config fields |
|---|---|---|---|
| `WHITE` | 1000 | — | `White: true, Amount: 1000` |
| `YELLOW` | 2000 | — | `Amount: 2000` (`White` stays false) |
| `CHAT` | 3000 | — | `InChat: true, Amount: 3000` |
| `MONSTER_BOOK` | 4000 | — | `MonsterBookBonus: 4000` |
| `MONSTER_EVENT` | 11 | — | `MobEventBonusPercentage: 11` |
| `PLAY_TIME` | 22 | 33 | `MobEventBonusPercentage: 22, PlayTimeHour: 33` |
| `WEDDING` | 5000 | — | `WeddingBonusEXP: 5000` |
| `SPIRIT_WEEK` | 55 | — | `QuestBonusRate: 55` |
| `PARTY` | 6000 | 44 | `PartyBonusExp: 6000, PartyBonusEventRate: 44` |
| `ITEM` | 7000 | — | `ItemBonusEXP: 7000` |
| `INTERNET_CAFE` | 8000 | — | `PremiumIPExp: 8000` |
| `RAINBOW_WEEK` | 9000 | — | `RainbowWeekEventEXP: 9000` |
| `PARTY_RING` | 10000 | — | `PartyEXPRingEXP: 10000` |
| `CAKE_PIE` | 11000 | — | `CakePieEventBonus: 11000` |

`PartyBonusPercentage` and `QuestBonusRemainCount` are written by no type and
are zero in every `want` (FR-12) — satisfied implicitly by the whole-struct
equality assertion, since a struct literal leaves them zero.

---

## Task 1: Document the distribution constants and add the exhaustiveness registry

Adds the FR-24/FR-25 doc comment above the `ExperienceDistributionType*`
constants and the `AllExperienceDistributionTypes` registry slice that Task 3's
exhaustiveness test iterates. No behavior change; nothing reads the slice yet.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go` — doc comment above the `ExperienceDistributionType*` constants (currently `kafka.go:110-123`, inside the `const` block that opens at line 101); new `AllExperienceDistributionTypes` package-level `var` immediately after that `const` block closes (currently line 126)
- `services/atlas-channel/atlas.com/channel/socket/model/experience_status.go` — read-only; the authoritative field comments the doc comment must agree with
- `docs/tasks/task-296-experience-gain-distribution-tests/prd.md` — read-only; §4.4 is the source table

A `var` cannot live inside a `const` block, so the slice goes after the block's
closing paren, before `type StatusEvent[E any] struct` (currently line 128).

- [ ] **Step 1: Add the doc comment above the constants**

Insert immediately above `ExperienceDistributionTypeWhite = "WHITE"`, inside
the existing `const` block. Content — the wording below is derived from
`experience_status.go`'s field comments, which win over the PRD table wherever
they differ:

```go
	// ExperienceDistributionType* are the distribution kinds carried in an
	// EXPERIENCE_CHANGED event's Distributions slice. Each maps to a distinct
	// field of socket/model.IncreaseExperienceConfig, and each field renders a
	// DIFFERENT line in the client's experience-gain message.
	//
	// Only WHITE, YELLOW and CHAT set the primary "You have gained experience"
	// amount. EVERY other type is a secondary modifier line rendered IN
	// ADDITION to that primary line -- picking one of them alone renders
	// "You have gained experience (+0)" with a bonus line beneath it.
	//
	//	WHITE          Amount, White=true          "You have gained experience", white text
	//	YELLOW         Amount, White=false         "You have gained experience", yellow text
	//	CHAT           Amount, InChat=true         chat-window experience line
	//	MONSTER_BOOK   MonsterBookBonus            right side, yellow: Bonus Event EXP
	//	MONSTER_EVENT  MobEventBonusPercentage     in chat, pink: bonus EXP per 3rd monster defeated
	//	PLAY_TIME      MobEventBonusPercentage,    right side, yellow: Bonus EXP for hunting over
	//	               PlayTimeHour (from Attr1)   (N) hrs
	//	WEDDING        WeddingBonusEXP             right side, yellow: Bonus Wedding EXP
	//	SPIRIT_WEEK    QuestBonusRate              Earned 'Spirit Week Event' bonus EXP
	//	PARTY          PartyBonusExp,              right side, yellow: Bonus Event Party EXP
	//	               PartyBonusEventRate (Attr1)
	//	ITEM           ItemBonusEXP                right side, yellow: Equip Item Bonus EXP
	//	INTERNET_CAFE  PremiumIPExp                right side, yellow: Internet Cafe EXP Bonus
	//	RAINBOW_WEEK   RainbowWeekEventEXP         right side, yellow: Rainbow Week Bonus Event EXP
	//	PARTY_RING     PartyEXPRingEXP             v95+ only
	//	CAKE_PIE       CakePieEventBonus           v95+ only
	//
	// ITEM is the trap: it is the "Equip Item Bonus EXP" MODIFIER line, not
	// "experience that came from an item". See task-277,
	// docs/tasks/task-277-stored-exp-items/bug-redeem-renders-as-item-bonus.md.
	//
	// A new constant added here MUST also be added to
	// AllExperienceDistributionTypes below, or
	// TestExperienceDistributionTypeExhaustiveness will not notice it is
	// unmapped.
```

- [ ] **Step 2: Add the registry slice**

After the `const` block closes, before `type StatusEvent[E any] struct`:

```go
// AllExperienceDistributionTypes enumerates every ExperienceDistributionType*
// constant, in declaration order. Its sole purpose is exhaustiveness
// enforcement: TestExperienceDistributionTypeExhaustiveness iterates it and
// fails when a type has no case in the consumer's mapping table. Adding a
// constant above without adding it here silently defeats that check.
var AllExperienceDistributionTypes = []string{
	ExperienceDistributionTypeWhite,
	ExperienceDistributionTypeYellow,
	ExperienceDistributionTypeChat,
	ExperienceDistributionTypeMonsterBook,
	ExperienceDistributionTypeMonsterEvent,
	ExperienceDistributionTypePlayTime,
	ExperienceDistributionTypeWedding,
	ExperienceDistributionTypeSpiritWeek,
	ExperienceDistributionTypeParty,
	ExperienceDistributionTypeItem,
	ExperienceDistributionTypeInternetCafe,
	ExperienceDistributionTypeRainbowWeek,
	ExperienceDistributionTypePartyRing,
	ExperienceDistributionTypeCakePie,
}
```

Exactly 14 entries, matching the 14 constants.

- [ ] **Step 3: Verify**

From `services/atlas-channel/atlas.com/channel`:

```
go build ./...
go vet ./kafka/message/character/...
gofmt -l kafka/message/character/kafka.go
```

`gofmt -l` must print nothing. Comment-only + one `var`; no test change expected.

---

## Task 2: Extract `buildIncreaseExperienceConfig` with the chain copied verbatim

Pure mechanical extraction (FR-1 – FR-4). The `if/else` chain moves **byte for
byte** out of `announceExperienceGain`'s innermost closure into a package-level
unexported function. **Do not convert it to a `switch` in this task** — that is
Task 4, and the intervening green test run is the evidence that the conversion
preserved behavior.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go` — remove the inline `c := ...` + loop at lines 367-405; add `buildIncreaseExperienceConfig` above `announceExperienceGain` (which begins at line 362)

Import aliases already present and sufficient: `character2 "atlas-channel/kafka/message/character"` (line 10) and `model2 "atlas-channel/socket/model"` (line 19). No import changes.

- [ ] **Step 1: Add the function**

Place it immediately above `func announceExperienceGain(...)`:

```go
// buildIncreaseExperienceConfig maps an EXPERIENCE_CHANGED event's
// distributions onto the client's IncreaseExperienceConfig. It is pure: no
// logger, no context, no session, no package-level state. A nil or empty
// slice yields the zero config. An unrecognized ExperienceType is silently
// ignored (see the type table in kafka/message/character/kafka.go).
func buildIncreaseExperienceConfig(ds []character2.ExperienceDistributions) model2.IncreaseExperienceConfig {
	c := model2.IncreaseExperienceConfig{}
	for _, d := range ds {
		// ... the existing chain from consumer.go:369-404, verbatim ...
	}
	return c
}
```

The chain body is transcribed unchanged from `consumer.go:369-404`, including
the `else if` form, the `character2.ExperienceDistributionType*` operands, the
`int32(...)` / `byte(...)` conversions, and the arm order. Nothing is renamed
and no arm is reordered.

- [ ] **Step 2: Reduce the call site**

`announceExperienceGain`'s innermost closure becomes:

```go
				return func(s session.Model) error {
					c := buildIncreaseExperienceConfig(distributions)

					err := session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(charpkt.CharacterStatusMessageOperationIncreaseExperienceBody(
						// ... 17-argument splat, unchanged ...
					))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to announce experience gain to character [%d].", s.CharacterId())
						return err
					}
					return nil
				}
```

The four-level currying, the parameter names, the `session.Announce` chain, the
positional 17-argument order, the error log string, and the returns are all
unchanged (FR-3).

- [ ] **Step 3: Verify**

From `services/atlas-channel/atlas.com/channel`:

```
go build ./...
go test ./kafka/consumer/character/...
gofmt -l kafka/consumer/character/consumer.go
```

Diff review: `git diff` on `consumer.go` must show only the extraction — no
changed literal, operator, conversion, or argument.

---

## Task 3: The mapping and exhaustiveness tests

Writes both test functions (FR-13 – FR-23) and runs them against the
**pre-refactor chain** from Task 2. This green run is the baseline.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer_test.go` — append the shared case type, the package-level `distributionMappingCases` var, and the two test functions
- `services/atlas-channel/atlas.com/channel/socket/model/experience_status.go` — read-only; the field names in every `want`
- `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go` — read-only; the constants and `AllExperienceDistributionTypes`

Patterns to copy: `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer_test.go:54-200` (`TestSnapshotHandlers` — the table-driven precedent, and the file's `package character` clause and import block at lines 1-19). No tenant, server, session, or snapshot helper is needed here; these tests are pure.

- [ ] **Step 1: Add the shared case table**

Package-level, so both test functions can read it (design §4.1):

```go
// distributionMappingCase drives both TestBuildIncreaseExperienceConfig and
// TestExperienceDistributionTypeExhaustiveness. `types` names the distribution
// types the case is the coverage owner of; multi-distribution and unknown-type
// cases declare nil so they contribute nothing to the coverage set.
type distributionMappingCase struct {
	name  string
	types []string
	given []character2.ExperienceDistributions
	want  model2.IncreaseExperienceConfig
}

var distributionMappingCases = []distributionMappingCase{ /* 19 cases, below */ }
```

The 14 single-type cases, each with a one-element `types`, exactly per the
plan's value-scheme table:

| `name` | `types` | `given` (type, Amount, Attr1) | `want` |
|---|---|---|---|
| `White_PrimaryWhiteText` | `WHITE` | `WHITE`, 1000, 0 | `{White: true, Amount: 1000}` |
| `Yellow_PrimaryYellowText` | `YELLOW` | `YELLOW`, 2000, 0 | `{Amount: 2000}` |
| `Chat_PrimaryInChat` | `CHAT` | `CHAT`, 3000, 0 | `{InChat: true, Amount: 3000}` |
| `MonsterBook_BonusEventExp` | `MONSTER_BOOK` | `MONSTER_BOOK`, 4000, 0 | `{MonsterBookBonus: 4000}` |
| `MonsterEvent_MobEventPercentage` | `MONSTER_EVENT` | `MONSTER_EVENT`, 11, 0 | `{MobEventBonusPercentage: 11}` |
| `PlayTime_MobEventPercentageAndHours` | `PLAY_TIME` | `PLAY_TIME`, 22, 33 | `{MobEventBonusPercentage: 22, PlayTimeHour: 33}` |
| `Wedding_BonusWeddingExp` | `WEDDING` | `WEDDING`, 5000, 0 | `{WeddingBonusEXP: 5000}` |
| `SpiritWeek_QuestBonusRate` | `SPIRIT_WEEK` | `SPIRIT_WEEK`, 55, 0 | `{QuestBonusRate: 55}` |
| `Party_BonusExpAndEventRate` | `PARTY` | `PARTY`, 6000, 44 | `{PartyBonusExp: 6000, PartyBonusEventRate: 44}` |
| `Item_EquipItemBonusExpNotPrimary` | `ITEM` | `ITEM`, 7000, 0 | `{ItemBonusEXP: 7000}` |
| `InternetCafe_PremiumIpExp` | `INTERNET_CAFE` | `INTERNET_CAFE`, 8000, 0 | `{PremiumIPExp: 8000}` |
| `RainbowWeek_BonusEventExp` | `RAINBOW_WEEK` | `RAINBOW_WEEK`, 9000, 0 | `{RainbowWeekEventEXP: 9000}` |
| `PartyRing_ExpRingExp_v95Plus` | `PARTY_RING` | `PARTY_RING`, 10000, 0 | `{PartyEXPRingEXP: 10000}` |
| `CakePie_EventBonus_v95Plus` | `CAKE_PIE` | `CAKE_PIE`, 11000, 0 | `{CakePieEventBonus: 11000}` |

`given` entries are written as
`[]character2.ExperienceDistributions{{ExperienceType: character2.ExperienceDistributionTypeWhite, Amount: 1000}}`
— the constant, never the raw string. `Attr1: 0` is omitted where it is zero.

`Item_EquipItemBonusExpNotPrimary` carries an inline comment restating the
task-277 trap: `Amount` stays zero, so an ITEM-only award renders
"You have gained experience (+0)".

The five multi-distribution cases, all `types: nil` (design §4.4):

| `name` | `given` | `want` |
|---|---|---|
| `WhiteAndChat_PrimaryAwardShape` | `WHITE` 1500; `CHAT` 1500 | `{White: true, InChat: true, Amount: 1500}` |
| `PrimaryPlusBonuses_Accumulate` | `WHITE` 2500; `PARTY` 600/Attr1 66; `ITEM` 770 | `{White: true, Amount: 2500, PartyBonusExp: 600, PartyBonusEventRate: 66, ItemBonusEXP: 770}` |
| `WhiteThenYellow_LastWins` | `WHITE` 1200; `YELLOW` 3400 | `{Amount: 3400}` (`White` false — FR-8/FR-9) |
| `EmptySlice_ZeroConfig` | `nil` | `{}` |
| `UnknownType_DeathIgnored` | `"DEATH"` 9999; `WHITE` 1300 | `{White: true, Amount: 1300}` |

`WhiteAndChat_PrimaryAwardShape` uses the same amount in both entries because
that is what the producer emits — `services/atlas-character/atlas.com/character/character/processor.go:791-792`
appends `WHITE` and `CHAT` with the same `amount`.

`UnknownType_DeathIgnored` uses the literal string `"DEATH"` (not a constant —
atlas-channel has none) because that is a **real** type the producer emits:
`services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:59`
declares `ExperienceDistributionTypeDeath = "DEATH"` and
`services/atlas-character/atlas.com/character/character/processor.go:840` sends
it. atlas-channel has no arm for it, so it is silently dropped today (FR-6).
The case pins that observed behavior. Its comment must say so.

- [ ] **Step 2: Write the mapping test**

```go
func TestBuildIncreaseExperienceConfig(t *testing.T) {
	for _, tc := range distributionMappingCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildIncreaseExperienceConfig(tc.given)
			if got != tc.want {
				t.Errorf("config mismatch\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}
```

Whole-struct equality (design §4.2): `IncreaseExperienceConfig` is comparable —
17 fields, all `bool`/`int32`/`byte`, no slice, map, func or pointer
(`socket/model/experience_status.go:3-21`). Do **not** use
`reflect.DeepEqual`, and do **not** compare field by field. Unnamed fields in
`want` are zero, which is what satisfies FR-12 and FR-15's "every other field
at zero" for free.

- [ ] **Step 3: Write the exhaustiveness test**

```go
func TestExperienceDistributionTypeExhaustiveness(t *testing.T) {
	covered := map[string]string{} // distribution type -> owning case name
	for _, tc := range distributionMappingCases {
		for _, dt := range tc.types {
			if prev, ok := covered[dt]; ok {
				t.Fatalf("distribution type %q claimed by two cases: %q and %q", dt, prev, tc.name)
			}
			covered[dt] = tc.name
		}
	}

	registered := map[string]bool{}
	for _, dt := range character2.AllExperienceDistributionTypes {
		registered[dt] = true
		if _, ok := covered[dt]; !ok {
			t.Errorf("distribution type %q is in AllExperienceDistributionTypes but has no case in distributionMappingCases", dt)
		}
	}

	for dt, name := range covered {
		if !registered[dt] {
			t.Errorf("case %q covers distribution type %q, which is not in AllExperienceDistributionTypes", name, dt)
		}
	}
}
```

Both directions name the offending type (FR-22, FR-23). Duplicate coverage is a
test-authoring error and `t.Fatalf`s (design §4.5).

- [ ] **Step 4: Verify — the baseline green run**

From `services/atlas-channel/atlas.com/channel`:

```
go test ./kafka/consumer/character/... -run 'TestBuildIncreaseExperienceConfig|TestExperienceDistributionTypeExhaustiveness' -v
go test ./kafka/consumer/character/...
gofmt -l kafka/consumer/character/consumer_test.go
```

All 19 subtests plus the exhaustiveness test must pass **against the `if/else`
chain**. Record the `-v` output in `progress.md`; this is the pre-conversion
baseline Task 4 is measured against.

---

## Task 4: Convert the chain to a `switch` with per-arm client-line comments

FR-5 – FR-7. The tests from Task 3 do **not** change; they must be green again
after this conversion, and that second green run is the behavior-preservation
evidence.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go` — the chain inside `buildIncreaseExperienceConfig`

- [ ] **Step 1: Convert**

```go
	for _, d := range ds {
		switch d.ExperienceType {
		case character2.ExperienceDistributionTypeWhite:
			// Primary "You have gained experience" line, white text.
			c.White = true
			c.Amount = int32(d.Amount)
		case character2.ExperienceDistributionTypeYellow:
			// Primary "You have gained experience" line, yellow text. The
			// White=false write only matters when a prior WHITE in the same
			// slice set it true.
			c.White = false
			c.Amount = int32(d.Amount)
		...
		}
	}
```

Rules for the conversion:

- Same 14 arms, same order as the chain, same right-hand sides character for
  character. `PLAY_TIME` and `PARTY` each keep both assignments in one case.
- **No `default` clause.** Silent fall-through on an unknown type is the
  behavior (FR-6), and a logging `default` would break the function's purity
  (FR-2) and change behavior.
- Each case gets a one-line comment naming the client line it renders, sourced
  from `socket/model/experience_status.go`'s field comments (which win over the
  PRD table where the wording differs).
- The `ITEM` case gets the full trap comment, verbatim:

```go
		case character2.ExperienceDistributionTypeItem:
			// ITEM renders the "Equip Item Bonus EXP" modifier line on the
			// right side -- NOT "experience that came from an item". It does
			// not touch the primary amount. Choosing this for an item-sourced
			// EXP award renders "You have gained experience (+0)". See
			// task-277,
			// docs/tasks/task-277-stored-exp-items/bug-redeem-renders-as-item-bonus.md.
			c.ItemBonusEXP = int32(d.Amount)
```

- [ ] **Step 2: Verify — the post-conversion green run**

From `services/atlas-channel/atlas.com/channel`:

```
go build ./...
go test ./kafka/consumer/character/... -run 'TestBuildIncreaseExperienceConfig|TestExperienceDistributionTypeExhaustiveness' -v
go test ./...
gofmt -l kafka/consumer/character/consumer.go
```

`git diff` on `consumer_test.go` for this task must be **empty** — if the tests
needed editing to pass, the conversion changed behavior. Record the `-v` output
in `progress.md` alongside Task 3's.

---

## Task 5: Mutation checks — prove the suite would have caught task-277

Design §4.6. These are temporary local edits: apply, run, capture the failure
output, revert. **The captured output is the deliverable** — a claim without it
is exactly the "verified from a partial run" this repo forbids. Nothing from
this task is committed except the `progress.md` record.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go` — temporarily edited, then reverted
- `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go` — temporarily edited, then reverted
- `docs/tasks/task-296-experience-gain-distribution-tests/progress.md` — new file (created by `/execute-task`); where the failure outputs land

- [ ] **Step 1: Registry mutation A — added type with no case**

Append `"FAKE_TYPE"` to `AllExperienceDistributionTypes`. Run
`go test ./kafka/consumer/character/... -run TestExperienceDistributionTypeExhaustiveness`.
Expect a failure naming `"FAKE_TYPE"` as registered-but-uncovered. Capture,
then `git checkout -- kafka/message/character/kafka.go`.

- [ ] **Step 2: Registry mutation B — removed type still covered**

Delete `ExperienceDistributionTypeCakePie` from the slice. Run the same test.
Expect a failure naming `"CAKE_PIE"` as covered-but-unregistered. Capture, then
revert.

- [ ] **Step 3: Mapping mutation — the task-277 bug, reintroduced**

Change the `ITEM` case to also write `c.Amount = int32(d.Amount)`. Run
`go test ./kafka/consumer/character/... -run TestBuildIncreaseExperienceConfig -v`.
Expect `Item_EquipItemBonusExpNotPrimary` to fail with `Amount: 7000` in `got`
and `Amount: 0` in `want`, and expect `PrimaryPlusBonuses_Accumulate` to fail
too (its `ITEM` entry would clobber the `WHITE` amount). Capture, then revert.

- [ ] **Step 4: Confirm the tree is clean and green**

```
git status --porcelain   # only progress.md, if anything
go test ./...
```

From `services/atlas-channel/atlas.com/channel`. Any leftover mutation is a
failure of this task.

---

## Task 6: Repo-wide verification gate

- [ ] **Step 1: Run the flagless gate**

From the worktree root:

```
tools/verify.sh
```

Must exit 0. `--quick` / `--no-docker` do not count (CLAUDE.md, "Done means
verified"). Dispatch `task-verifier` for this rather than running it in a large
context.

- [ ] **Step 2: Review gates**

Before the PR: `backend-guidelines-reviewer` over the three changed Go files,
and a `task-reviewer` pass over the branch's commit range. Both per
`docs/review-protocol.md`.
