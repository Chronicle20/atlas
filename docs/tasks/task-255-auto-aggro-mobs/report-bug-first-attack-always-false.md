# Report: bug-first-attack-always-false

Task: task-255-auto-aggro-mobs
Brief: docs/tasks/task-255-auto-aggro-mobs/bug-first-attack-always-false.md
Agent: atlas-implementer (sonnet)
Status: DONE
Commit: 38d4b98b4 — `fix(atlas-data): read firstAttack as a scalar attribute, not a child imgdir`

## Implemented

`services/atlas-data/atlas.com/data/monster/reader.go`
- `m.FirstAttack = getFirstAttack(node)` replaced with the direct scalar read
  matching every sibling boolean in the block:
  `m.FirstAttack = node.GetIntegerWithDefault("firstAttack", 0) > 0`.
- `getFirstAttack` deleted. It called `node.ChildByName("firstAttack")`, which
  always errored because `firstAttack` is a scalar `<int>` leaf of `info`, not a
  child `imgdir` — so it always returned `false`.
- `math` import kept; still used by `math.MaxInt32` at line 52.

`services/atlas-data/atlas.com/data/monster/reader_test.go`
- `TestReader`'s `FirstAttack` assertion inverted to `expected true`, matching
  the fixture at line 42 (`<int name="firstAttack" value="1"/>`). The fixture was
  always correct; only the assertion was wrong.
- Added `TestReaderFirstAttackAbsentDefaultsFalse` (minimal fixture with no
  `firstAttack` key), following the existing `TestReaderFixedDamage` pattern, so
  the default path stays covered.

## Consumer sweep (report only, no changes)

`grep -rniE "first_attack|firstattack"` over `*.go`/`*.ts`/`*.tsx`/`*.yaml`/`*.json`,
excluding atlas-data and atlas-monsters:

1. `services/atlas-monster-death/atlas.com/monster/monster/information/rest.go:34`
   — deserializes `first_attack`, but `Extract` (same file, 86-91) copies only
   `hp` and `experience` into the domain `Model`. Parsed and dropped; no behavior
   reads it. Pre-existing dead field, not touched.
2. `services/atlas-ui/src/types/models/monster.ts:26` — TS DTO field.
3. `services/atlas-ui/src/pages/MonstersPage.tsx:246` — renders a badge when true.
4. `services/atlas-ui/src/pages/MonsterDetailPage.tsx:292` — pushes a
   `{ label: "First attack", tone: "warn" }` tag when true.
   (2-4 are pure display. Previously every monster showed `false`; now templates
   with `firstAttack=1` will show the badge. UI catching up to correct data, not
   a regression.)
5. `libs/atlas-packet/monster/serverbound/auto_aggro.go:19,33` and
   `libs/atlas-packet/monster/clientbound/reset_monster_animation.go:25` —
   comments citing client-side `bFirstAttack`/`TryFirstAttack` IDA symbols only.

The only genuine behavioral consumer is
`services/atlas-monsters/atlas.com/monsters/monster/processor.go:1926`
(`SetAggro` gate 3), which needed no change — `information/rest.go:35,104`
already mapped the field faithfully and was simply always receiving `false`.

## Module-local verification

`cd services/atlas-data/atlas.com/data && go build ./... && go test ./...`
— build clean, all packages pass, including `ok atlas-data/monster 0.109s`.

Repo-wide gate: see the ledger row for this unit.

## Concerns

- atlas-ui's admin pages will start rendering the "First attack" badge for
  monsters that previously always showed false. Intended effect.
- atlas-monster-death's unused `first_attack` field is pre-existing and out of
  scope.
- Live re-test in `atlas-pr-1460` not yet performed.
