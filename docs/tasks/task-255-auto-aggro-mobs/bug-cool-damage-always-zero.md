# bug: `cool_damage` is always `{0, 0}` — same `ChildByName` defect as `firstAttack`

Task: task-255-auto-aggro-mobs
PR: atlas-pr-1460
Reported: 2026-08-25
Status: diagnosed, root cause established

Found by code review of commit `38d4b98b4` while answering the second open
question in `bug-first-attack-always-false.md`. Folded into this PR by explicit
user decision — it is the same defect, in the same function block, in the same
file the auto-aggro fix already touches.

## Observed

`services/atlas-data/atlas.com/data/monster/reader.go:239-247`:

```go
func getCoolDamage(node *xml.Node) coolDamage {
	c, err := node.ChildByName("coolDamage")
	if err != nil {
		return coolDamage{}
	}
	damage := uint32(c.GetIntegerWithDefault("coolDamage", 0))
	probability := uint32(c.GetIntegerWithDefault("coolDamageProb", 0))
	return coolDamage{Damage: damage, Probability: probability}
}
```

This is the identical shape as the `getFirstAttack` defect fixed in `38d4b98b4`:
`ChildByName` is used to look up what is actually a scalar leaf attribute, so it
always errors and the function always returns the zero value.

## Root cause

`coolDamage` and `coolDamageProb` are scalar `<int>` leaves of the `info`
imgdir, not a child imgdir. Verified across the whole WZ mob corpus
(`/mnt/d/Source/AtlasMS/wz/Mob.wz/*.img.xml`):

```
grep -l 'imgdir name="coolDamage"' ... | wc -l   ->  0
grep -l '<int name="coolDamage"'    ... | wc -l   -> 24
```

Zero occurrences of an `<imgdir name="coolDamage">` anywhere; 24 templates carry
the leaf form. Example, `9700000.img.xml`, inside `<imgdir name="info">`:

```xml
<int name="removeOnMiss" value="1"/>
<int name="coolDamage" value="200"/>
<int name="coolDamageProb" value="10"/>
<int name="category" value="8"/>
```

Both keys sit as direct siblings of `removeOnMiss`, which the same reader
already reads correctly via `node.GetIntegerWithDefault("removeOnMiss", 0)`.

Consequence: `m.CoolDamage` is `{Damage: 0, Probability: 0}` for every monster
in the game, and has been for as long as the helper has existed. The 24
templates that define cool damage lose it silently.

## Expected

`cool_damage.damage` and `cool_damage.probability` carry the WZ values for the
24 templates that define them, and `{0, 0}` for every template that does not.

## Fix

- `services/atlas-data/atlas.com/data/monster/reader.go:239-247` — read both
  keys directly off the `info` node, exactly as the sibling scalars in the same
  function do:

  ```go
  func getCoolDamage(node *xml.Node) coolDamage {
      return coolDamage{
          Damage:      uint32(node.GetIntegerWithDefault("coolDamage", 0)),
          Probability: uint32(node.GetIntegerWithDefault("coolDamageProb", 0)),
      }
  }
  ```

  Keep the helper here rather than inlining — unlike `FirstAttack` it builds a
  struct from two keys, so a named function still earns its place. The call site
  at `reader.go:78` (`m.CoolDamage = getCoolDamage(node)`) does not change.

- `services/atlas-data/atlas.com/data/monster/reader_test.go` — check the main
  `TestReader` fixture for `coolDamage`/`coolDamageProb` keys and whether an
  existing assertion asserts the zero value. If the fixture has the keys, the
  assertion is wrong the same way the `FirstAttack` one was and must be
  corrected to the real values; if it does not, ADD the keys to the fixture and
  assert them, so the parse is actually covered.
- Add a test for the absent-key default asserting `{0, 0}`, following the
  `TestReaderFirstAttackAbsentDefaultsFalse` case added in `38d4b98b4`.

## Blast radius

`cool_damage` goes from universally `{0,0}` to populated on 24 templates.
Sweep for consumers before finishing — the `first_attack` sweep found that
serialized-but-unused DTO fields are common in this codebase, so establish
whether anything actually branches on `cool_damage` and report what you find.
Do not change consumers; report only.

## Not yet answered

- Whether any service consumes `cool_damage` behaviorally, or whether it is
  display/dead data like `first_attack` was in atlas-monster-death.
- Whether `getSelfDestruction`, `getLoseItems`, `getRevives`, `getResistances`,
  `getSkills` and `getAttacks` are genuinely correct. The prior review asserted
  the first two target real child imgdirs, but the full set was never checked
  against the corpus the way `firstAttack` and `coolDamage` now have been.

## Resolution

Fixed by commit `06159cb0b` — `fix(atlas-data): read coolDamage/coolDamageProb as
scalar info leaves`. Report: `report-bug-cool-damage-always-zero.md`.

- Folded into PR 1460 by explicit user decision rather than deferred.
- Blast-radius sweep answered the first open question: no service branches on
  `cool_damage`; both consumer DTOs declare the field and never dereference it.
  Serialized-but-unused, same as `first_attack` in atlas-monster-death.
- Module verification passes; controller re-ran `go test ./monster/... -count=1`
  uncached to confirm.
- No separate code review was dispatched for this commit: it is a mechanical
  repeat of `38d4b98b4`, already reviewed, and the reviewer is the party that
  found and characterised this defect in the first place.
- Repo-wide gate blocked by the same pre-existing golangci-lint / go1.27
  toolchain drift described in `bug-first-attack-always-false.md`.
- Second open question (`getSelfDestruction`, `getLoseItems`, `getRevives`,
  `getResistances`, `getSkills`, `getAttacks` vs the WZ corpus) remains OPEN.
