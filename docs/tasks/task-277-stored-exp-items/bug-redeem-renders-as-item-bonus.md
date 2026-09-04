# Bug — stored-EXP redeem renders as "Equip Item Bonus EXP", primary amount is 0

Task: task-277-stored-exp-items. PR: atlas-pr-1567.
Tenant: `dbf2f7ba-2533-4885-8cf2-2934697efdd3`, GMS 83.1 (the only tenant in the
`atlas-pr-1567` namespace). World "Atlas2".

## Reproduced

Live, on `atlas-pr-1567`. Used a Writ of Solomon `2370000` (banks `spec/exp`
= 100000 — confirmed against the live `atlas-data` consumable endpoint), then
clicked the EXP bar to charge the banked balance (`USE_GACHA_EXP`).

## Observed

The redeem works — the EXP is really awarded and the counter really zeroes. Only
the on-screen feedback is wrong. The client renders two yellow lines:

```
You have gained experience (+0)
Equip Item Bonus EXP (+100000)
```

## Expected

One line attributing the whole redeemed amount to the primary EXP field, in the
same shape every other visible EXP award in the repo uses — e.g.
`You have gained experience (+100000)`.

## Root cause

`RedeemStoredExperience` hands `AwardExperience` a single distribution typed
`ExperienceDistributionTypeItem`:

`services/atlas-character/atlas.com/character/character/processor.go:1558-1559`

```go
experience := []ExperienceModel{NewExperienceModel(character2.ExperienceDistributionTypeItem, redeemed, 0)}
return ip.AwardExperience(mb)(transactionId, characterId, channel, experience, false)
```

`"ITEM"` does not mean "EXP that came from an item". In atlas-channel's
`IncreaseExperience` fan-out it maps to the client's `itemBonusEXP` field —
the *Equip Item Bonus EXP* modifier line, not the primary amount:

`services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:393-394`

```go
} else if d.ExperienceType == character2.ExperienceDistributionTypeItem {
    c.ItemBonusEXP = int32(d.Amount)
}
```

Only `WHITE`, `YELLOW` and `CHAT` set `c.Amount` (consumer.go:370-378). With none
of those present, `c.Amount` stays at its zero value, which is the `(+0)` on the
first line, and `c.White` stays `false`, which is why both lines are yellow.

The EXP itself is correct because `AwardExperience` sums `e.amount` over every
distribution regardless of type (processor.go:750-753) before persisting. This is
purely a presentation defect.

The design specified the wrong type (`design.md:372`), the plan carried it
(`plan.md:1062`), and `plan.md:980` pinned it with
`TestRedeemStoredExperienceDistributionIsItem`, which asserts the ITEM shape and
therefore must change with the fix.

The established convention for a visible EXP award is `WHITE` plus
`ShowEffect: true` — see the quest saga builder,
`services/atlas-quest/atlas.com/quest/kafka/producer/saga/builder.go:80-86`.
`AwardExperience` appends the White + Chat display pair itself when
`showEffect` is set (processor.go:787-794), and it does so *after* computing
`amount`, so the appended pair cannot double-award.

## Fix

- `services/atlas-character/atlas.com/character/character/processor.go:1558-1559`
  — emit the redeemed amount as `ExperienceDistributionTypeWhite` and pass
  `showEffect = true`. Keep exactly ONE input distribution carrying `redeemed`:
  `AwardExperience` sums the input slice, so adding a second one that also
  carries the amount would double the EXP actually granted.
- `services/atlas-character/atlas.com/character/character/processor_test.go` (or
  wherever `TestRedeemStoredExperienceDistributionIsItem` lives — find it by
  name) — retarget the assertion to the new shape: the emitted
  `EXPERIENCE_CHANGED` carries the White distribution with `Amount == 5000`, and
  the `showEffect` pair, and no `ITEM` distribution. Assert the character's
  persisted experience still increased by exactly `5000`, not `10000`, so the
  double-award regression is pinned.
- `docs/tasks/task-277-stored-exp-items/design.md:372` and `plan.md:980,1062` —
  correct the recorded distribution type so the artifacts match the code.

## Not yet answered

- Whether the chat-log line that `showEffect` adds is wanted for this path. The
  quest path has it; the Writ redeem is a comparable player-initiated award, so
  the fix takes it. If it reads as noise in live testing, dropping it means
  `showEffect = false` with the same White distribution — one line, no chat entry.
- Whether `white = true` (white text) or `YELLOW` better matches the client's
  own presentation for this specific redeem. No client evidence was gathered for
  this; `WHITE` is chosen to match repo convention, not from a decompile.
