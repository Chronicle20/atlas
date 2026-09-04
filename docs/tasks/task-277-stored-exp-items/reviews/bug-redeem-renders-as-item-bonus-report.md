# Report — bug: stored-EXP redeem renders as "Equip Item Bonus EXP"

Task: task-277-stored-exp-items (bug fix, not a new plan task).

## Rulings applied (given by controller, not re-derived)

- Take the chat-log line: `showEffect = true`.
- Use `ExperienceDistributionTypeWhite`, not `YELLOW`.

## What I implemented

### `services/atlas-character/atlas.com/character/character/processor.go`

`RedeemStoredExperience` previously emitted the redeemed amount as a single
`ExperienceDistributionTypeItem` distribution with `showEffect = false`. That
type maps to the client's `itemBonusEXP` modifier field in atlas-channel's
`IncreaseExperience` fan-out, not the primary EXP amount, which is why the
client rendered `You have gained experience (+0)` / `Equip Item Bonus EXP
(+100000)`.

Changed to:

```go
experience := []ExperienceModel{NewExperienceModel(character2.ExperienceDistributionTypeWhite, redeemed, 0)}
return ip.AwardExperience(mb)(transactionId, characterId, channel, experience, true)
```

This keeps exactly one input distribution carrying `redeemed` (per the
brief's warning against a second amount-carrying entry, since
`AwardExperience` sums the input slice before persisting). `showEffect =
true` makes `AwardExperience` append its own White + Chat display pair
*after* summing `amount`, so the persisted EXP gain is still exactly
`redeemed`, not doubled. The appended White distribution and the input White
distribution carry the same amount and both set (not add to) the channel
consumer's `c.Amount`/`c.White` fields, so there is no double-render — one
"You have gained experience (+N)" line, plus the CHAT-line entry the
`showEffect` convention adds (matching the quest saga builder,
`services/atlas-quest/atlas.com/quest/kafka/producer/saga/builder.go:80-86`).

### `services/atlas-character/atlas.com/character/character/processor_stored_experience_test.go`

Renamed `TestRedeemStoredExperienceDistributionIsItem` to
`TestRedeemStoredExperienceDistributionIsWhite` and retargeted it:

- Asserts no distribution in the emitted `EXPERIENCE_CHANGED` is
  `ExperienceDistributionTypeItem`.
- Asserts at least one `WHITE` distribution with `Amount == 5000` (both the
  input distribution and the `showEffect`-appended one satisfy this, and both
  correctly carry the same, non-doubled amount).
- Asserts exactly one `CHAT` distribution with `Amount == 5000`, pinning that
  `showEffect` appended its pair exactly once.
- Asserts the character's persisted `Experience()` increased by exactly
  `5000` (`startingExperience + 5000`), not `10000` — pins the
  double-award regression the brief called out.

### `docs/tasks/task-277-stored-exp-items/design.md`

Line 372 area: corrected the recorded command shape from "a single
`ExperienceDistributionTypeItem` distribution" to "a single
`ExperienceDistributionTypeWhite` distribution and `showEffect = true`".

### `docs/tasks/task-277-stored-exp-items/plan.md`

- Line ~980: retargeted the recorded test description from
  `TestRedeemStoredExperienceDistributionIsItem` (asserting `ITEM`) to
  `TestRedeemStoredExperienceDistributionIsWhite` (asserting `WHITE` +
  showEffect pair, no `ITEM`, and non-doubled persisted experience).
- Line ~1062 area (code excerpt inside the plan): updated the excerpted
  `RedeemStoredExperience` snippet to match the new
  `ExperienceDistributionTypeWhite` / `showEffect = true` call, so the plan
  artifact matches the code.

## Not-yet-answered items — resolved per controller ruling

Both open questions in the bug file's "Not yet answered" section are settled
by the rulings given to me (chat line kept via `showEffect = true`; `WHITE`
not `YELLOW`), so I implemented accordingly and did not re-open them.

## Testing

Module-local, from `services/atlas-character/atlas.com/character`:

```
go build ./...
```
No output (success).

```
go test ./character/... -run "RedeemStoredExperience|StoredExperience" -v
```

```
=== RUN   TestCreditStoredExperience
...
--- PASS: TestCreditStoredExperience (0.01s)
=== RUN   TestCreditStoredExperienceUnknownCharacter
--- PASS: TestCreditStoredExperienceUnknownCharacter (0.00s)
=== RUN   TestRedeemStoredExperience
    --- PASS: TestRedeemStoredExperience/redeems_the_whole_balance (0.00s)
    --- PASS: TestRedeemStoredExperience/at_the_level_bound (0.00s)
    --- PASS: TestRedeemStoredExperience/above_the_level_bound_is_a_no-op (0.00s)
    --- PASS: TestRedeemStoredExperience/zero_balance_is_a_no-op (0.00s)
=== RUN   TestRedeemStoredExperienceIsExactlyOnce
--- PASS: TestRedeemStoredExperienceIsExactlyOnce (0.00s)
=== RUN   TestRedeemStoredExperienceDistributionIsWhite
--- PASS: TestRedeemStoredExperienceDistributionIsWhite (0.00s)
PASS
ok  	atlas-character/character	0.051s
```

Full module suite (`go test ./...`), also from
`services/atlas-character/atlas.com/character`:

```
ok  	atlas-character	0.022s
ok  	atlas-character/character	10.515s
ok  	atlas-character/configuration	0.010s
ok  	atlas-character/data/portal	0.010s
ok  	atlas-character/data/skill	0.010s
ok  	atlas-character/data/skill/effect	0.004s
ok  	atlas-character/data/skill/effect/statup	0.005s
?   	atlas-character/drop	[no test files]
ok  	atlas-character/equipslot	0.024s
?   	atlas-character/external/effective_stats	[no test files]
...
ok  	atlas-character/kafka/consumer/character	13.259s
ok  	atlas-character/kafka/consumer/drop	0.023s
...
ok  	atlas-character/kafka/message/character	0.006s
...
ok  	atlas-character/location	0.030s
ok  	atlas-character/pending_change	306.267s
ok  	atlas-character/session	0.060s
ok  	atlas-character/session/history	0.041s
ok  	atlas-character/skill	0.039s
ok  	atlas-character/teleport_rock	0.074s
[exited with code 0]
```

All packages pass, exit code 0. No prior test elsewhere in the module
referenced the old `ExperienceDistributionTypeItem` shape or the old test
name, confirmed by grep before commit.

## Files changed

- `services/atlas-character/atlas.com/character/character/processor.go`
- `services/atlas-character/atlas.com/character/character/processor_stored_experience_test.go`
- `docs/tasks/task-277-stored-exp-items/design.md`
- `docs/tasks/task-277-stored-exp-items/plan.md`

## Self-review

- Confirmed `AwardExperience` sums the *input* `experience` slice for the
  persisted amount before the `showEffect` append (processor.go:750-753 and
  787-794), so a single White input distribution plus the appended pair does
  not double the persisted EXP — pinned in the test by asserting
  `startingExperience + 5000` exactly.
- Confirmed the channel consumer's fan-out (`consumer.go:369-378`) *sets*
  rather than accumulates `c.Amount`/`c.White` per distribution type, so
  having two WHITE distributions in the event (input + showEffect-appended)
  with the same amount is idempotent, not a double-render.
- No other test or non-test code in the module referenced
  `ExperienceDistributionTypeItem` in connection with the redeem path.
- Left the untracked `docs/tasks/task-277-stored-exp-items/audit-writ-maxlevel.md`
  and `bug-redeem-renders-as-item-bonus.md` alone — they predate this task
  and are not part of this fix's diff.

## Issues or concerns

None. Build and full module test suite are green.

## Commit

`e82d77c42` — fix(character): redeem stored EXP as primary EXP, not item bonus
Branch: `task-277-stored-exp-items` (confirmed via `git branch --show-current`
after commit). Worktree confirmed via `git rev-parse --show-toplevel`.
