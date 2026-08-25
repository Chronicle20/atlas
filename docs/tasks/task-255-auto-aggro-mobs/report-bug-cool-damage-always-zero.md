# Report: bug-cool-damage-always-zero

Task: task-255-auto-aggro-mobs
Brief: docs/tasks/task-255-auto-aggro-mobs/bug-cool-damage-always-zero.md
Agent: atlas-implementer (sonnet)
Status: DONE
Commit: 06159cb0b — `fix(atlas-data): read coolDamage/coolDamageProb as scalar info leaves`

## Implemented

`services/atlas-data/atlas.com/data/monster/reader.go` — `getCoolDamage` now reads
both keys as scalar leaves of `info`, matching the sibling scalar reads
(`removeOnMiss`, `fixedDamage`) in the same block:

```go
func getCoolDamage(node *xml.Node) coolDamage {
	return coolDamage{
		Damage:      uint32(node.GetIntegerWithDefault("coolDamage", 0)),
		Probability: uint32(node.GetIntegerWithDefault("coolDamageProb", 0)),
	}
}
```

Call site at `reader.go:78` unchanged, per the brief.

## Tests

`services/atlas-data/atlas.com/data/monster/reader_test.go`
- The main `TestReader` fixture had no cool-damage keys, so
  `<int name="coolDamage" value="200"/>` and
  `<int name="coolDamageProb" value="10"/>` were added as siblings of
  `firstAttack`, mirroring the WZ example in the bug file (`9700000.img.xml`).
- The existing assertion moved from `coolDamage{0,0}` to `coolDamage{200, 10}`.
  Field order `{Damage, Probability}` was checked against `rest.go:97-100`
  before writing the literal.
- Added `TestReaderCoolDamageAbsentDefaultsZero`, mirroring
  `TestReaderFirstAttackAbsentDefaultsFalse`; uses monster id `9300317` (distinct
  from `9300316`) to avoid registry collisions.

## Blast radius sweep (report only)

Consumers of `cool_damage`:
- `services/atlas-monster-death/atlas.com/monster/monster/information/rest.go`
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go`

Both only declare the DTO struct and field. `grep -rn "\.CoolDamage"` across all
services found no dereference outside `atlas-data`'s own reader and its tests.
No service branches on `.CoolDamage` / `.Damage` / `.Probability`.

Same shape as `first_attack` in atlas-monster-death: serialized-but-unused DTO
data. The value goes from universally `{0,0}` to populated on the 24 templates
that define it, with no behavioral consumer today. No consumer files modified.

## Verification

`cd services/atlas-data/atlas.com/data && go build ./... && go test ./...`
— build clean, all packages pass.

Controller re-ran uncached to confirm, not trusting the agent's claim:
`go test ./monster/... -count=1 -run TestReader` → `ok atlas-data/monster 0.013s`.

Repo-wide gate: blocked by pre-existing toolchain drift, see the ledger and the
bug files' Resolution sections.

## Carried forward

Correctness of `getSelfDestruction`, `getLoseItems`, `getRevives`,
`getResistances`, `getSkills`, `getAttacks` against the full WZ corpus remains
unaudited.
