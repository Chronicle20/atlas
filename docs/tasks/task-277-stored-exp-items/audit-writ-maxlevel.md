# Audit — can a Writ bank EXP that can never be redeemed?

Date: 2026-09-02. Trigger: live PR test on `atlas-pr-1567`, where using `2370000`
credited 100,000 stored ("gachapon") EXP rather than character EXP. That is the
designed behaviour (`design.md` §0), and the EXP-bar charge affordance redeemed
it successfully. The report did surface a latent risk worth settling.

## The risk

Two different level gates guard the two halves of the feature:

- **Credit** (`services/atlas-consumables/atlas.com/consumables/consumable/solomon.go:82`)
  rejects when `character.Level() > item.MaxLevel()` — the item's `info/maxLevel`.
- **Redeem** (`services/atlas-character/atlas.com/character/character/processor.go:1543`)
  refuses above `storedExperienceMaxLevel = 50`, mirroring the client's
  `CUIStatusBar::TryUseTempExp` gate (`design.md` §1.3).

If any Writ carried `info/maxLevel > 50`, a character in the window
`51 .. maxLevel` could bank EXP the redeem path would then permanently refuse, stranding it.

## Result: the window does not exist

Queried the live `atlas-data` consumable endpoint in `atlas-pr-1567` for the whole
family, tenant `dbf2f7ba-2533-4885-8cf2-2934697efdd3` (GMS 83.1):

| item | `info/maxLevel` | `spec/exp` |
|---|---|---|
| 2370000 | 50 | 100000 |
| 2370001 | 50 | 50000 |
| 2370002 | 50 | 30000 |
| 2370003 | 50 | 20000 |
| 2370004 | 50 | 10000 |
| 2370005 | 50 | 5000 |
| 2370006 | 50 | 3000 |
| 2370007 | 50 | 2000 |
| 2370008 | 50 | 1000 |
| 2370009 | 50 | 500 |
| 2370010 | 50 | 300 |
| 2370011 | 50 | 200 |
| 2370012 | 50 | 100 |

Every member is `maxLevel == 50`, exactly equal to the redeem bound, so the credit
gate and the redeem gate coincide and the stranding window is empty. No code change
is warranted. `2370000`'s `spec/exp` of 100000 also confirms the observed 100k
credit was the correct amount.

## Scope of this check

GMS 83.1 only — the sole tenant in that namespace. Other version columns serve
their own `Item.wz`. The bound is a property of the WZ data, not of the code, so a
tenant whose data set `maxLevel > 50` would reopen the window; the coincidence
above is not enforced anywhere. Clamping the credit gate to
`min(maxLevel, 50)` would make it structural rather than incidental — not done,
since no shipped data needs it.
