# Pre-change byte baseline (captured on branch task-226 before any codec move)

Fixture model (identical in every later fixture in this task):
  entry[0] = name "Buff",   shout true,  skillIds 1001003 / 1001004 / 0
  entry[1] = name "Attack", shout false, skillIds 1001005 / 0       / 0

## Shipped clientbound encoder — `model.Macros.Encode` (`libs/atlas-packet/model/macros.go:20`)
| variant | len | hex |
|---|---|---|
| GMS v83 | 41 | 02040042756666012b460f002c460f0000000000060041747461636b002d460f000000000000000000 |
| GMS v84 | 41 | 02040042756666012b460f002c460f0000000000060041747461636b002d460f000000000000000000 |

Shout is written UPRIGHT here (`macros.go:53` `w.WriteBool(m.shout)`).

## Dead clientbound encoder — `character.SkillMacro.Encode` (`libs/atlas-packet/character/skill_macro.go:41`)
Writes shout INVERTED (`skill_macro.go:47` `w.WriteBool(!e.Shout)`), so its byte 7
differs from the shipped encoder's for the same model. Not referenced by any
production announce site (see design.md §1.1); superseded by Task 6.

## Shipped serverbound decoder — `character.SkillMacro.Decode` (`libs/atlas-packet/character/skill_macro.go:56`)
Reads shout INVERTED (`skill_macro.go:62` `shout := !r.ReadBool()`), i.e. the
opposite polarity from the shipped encoder above. Exactly one of the two is
correct; Task 3 decides which from the IDB.
