# GMS v95.1 `Skill.wz` `common` node grammar — evidence-backed inventory

## Provenance

- Source object: MinIO `atlas-wz` bucket, key `shared/regions/GMS/versions/95.1/Skill.wz`
  (endpoint `minio.minio.svc.cluster.local:9000` from Secret `atlas-minio-credentials`
  in namespace `atlas-main`; reached via `kubectl -n minio port-forward svc/minio 19000:9000`
  + `curl --aws-sigv4`).
- Local copy size 163.7 MB, `md5 = 2d77583108eb928b65f2904196a894ef`.
- Parsed with `libs/atlas-wz` `wz.Open`. **`File.GameVersion()` reports `95`** — the
  archive self-identifies as v95, not substituted from another version.
- Analysis programs: standalone scratch module `wzscan/` (+ `wzscan/pass2`, `wzscan/pass3`)
  with `replace` directives to the repo libs. Nothing in the repo was modified.
- Every number below is a full-archive census, not a sample.

## Archive shape (context)

Skill.wz root contains 118 images. Only the **numeric-named root images** (`000`, `100`,
… `9000` — 112 of them) have an `info` + `skill` structure. The non-numeric images
(`Attacktype`, `BFSkill`, `ItemSkill`, `MCGuardian`, `MCSkill`, `MobSkill`) and the
`Dragon/` sub-directory images have a completely different top-level shape (numeric
effect ids, or `info`/`stand`/`move`/animation names) and **contain no `skill` node at
all**. This matches `atlas-data`'s `skill.ParseJobId`, which rejects non-numeric image
names.

**Exhaustive proof of `common` placement**: a recursive walk of every property in every
image found **635 nodes named `common`, and every single one is at the path
`/<jobId>/skill/<skillId>/common`**. There are no `common` nodes anywhere else in the
archive (not under `level/<n>/`, not under `info`, not in the non-numeric images).

---

## 1. Counts

Scope = the 954 skill entries under `/<numeric jobId>/skill/`:

| bucket | count |
|---|---|
| **total skills** | **954** |
| `level` only | **319** |
| `common` only | **633** |
| **both** `common` and `level` | **2** |
| neither | **0** |

Non-numeric / `Dragon/` images: 0 skills (no `skill` node) — they contribute nothing to
any bucket.

Cross-check: 319 + 633 + 2 = 954 ✅, and 633 + 2 = 635 = total `common` node count ✅.

Level-entry-count distribution for the 321 skills that have a `level` subtree:

| level entries | skills |
|---|---|
| 1 | 279 |
| 2 | 10 |
| 3 | 20 |
| 4 | 10 |
| 30 | 2 (the two dual skills, §6) |

---

## 2. Complete set of `common` child keys

**65 distinct key names**, 2 946 leaf occurrences total (2 374 string + 634 int +
452 vector — the vector count is `lt`/`rb` = 226+226).

Node types seen across the whole archive: **`string`, `int`, `vector` only.** No float,
double, short, long, null, canvas, convex, sound, or UOL ever appears under a `common`
node.

| key | count | node type(s) | example 1 (skill) | example 2 (skill) |
|---|---|---|---|---|
| `maxLevel` | 635 | int×632, **string×3** | `20` (0000012) | `20` (1001003) |
| `mpCon` | 460 | string×460 | `6+2*u(x/5)` (1001003) | `3+2*u(x/5)` (1001004) |
| `x` | 315 | string×314, **int×1** | `x` (0000012) | `x` (10000012) |
| `time` | 300 | string×299, **int×1** | `100+10*x` (1001003) | `10*x` (1101004) |
| `damage` | 277 | string×277 | `160+5*x` (1001004) | `80+5*x` (1001005) |
| `lt` | 226 | **vector×226** | `(-300,-200)` (1101006) | `(-200,-113)` (1101008) |
| `rb` | 226 | **vector×226** | `(300,200)` (1101006) | `(-10,0)` (1101008) |
| `mobCount` | 200 | string×200 | `6` (1001005) | `3` (1101008) |
| `prop` | 118 | string×118 | `2*x` (1100002) | `30+3*x` (1111003) |
| `y` | 93 | string×93 | `x` (0000012) | `x` (10000012) |
| `cooltime` | 80 | string×80 | `60` (1111007) | `660-60*x` (1121011) |
| `attackCount` | 47 | string×47 | `2` (1111010) | `2` (11111004) |
| `range` | 46 | string×46 | `180` (1001005) | `200` (1101008) |
| `mastery` | 32 | string×32 | `10+2*x` (1100000) | `10+2*x` (11100000) |
| `z` | 29 | string×29 | `x` (0000012) | `x` (10000012) |
| `dot` | 20 | string×20 | `90+4*x` (12111003) | `40+2*x` (12111005) |
| `cr` | 20 | string×20 | `20+x` (1221009) | `10+2*x` (1310009) |
| `dotInterval` | 20 | string×20 | `1` (12111003) | `1` (12111005) |
| `dotTime` | 20 | string×20 | `5+d(x/4)` (12111003) | `2+u(x/5)` (12111005) |
| `bulletCount` | 20 | string×20 | `2` (13001003) | `4` (13111001) |
| `damR` | 17 | string×17 | `u(x/4)` (1111002) | `u(x/4)` (11111001) |
| `criticaldamageMin` | 15 | string×15 | `5+u(x/2)` (1221009) | `5+x` (1310009) |
| `speed` | 15 | string×15 | `3*x` (13100004) | `-20+x` (13101006) |
| `mhpR` | 13 | string×13 | `2*x` (1000006) | `2*x` (11000005) |
| `mad` | 12 | string×12 | `40` (11111007) | `40` (1211006) |
| `v` | 12 | string×12 | `5+d(x/4)` (11110005) | `5+d(x/6)` (1120003) |
| `ignoreMobpdpR` | 10 | string×10 | `4*x` (1120012) | `5+u(x/2)` (1221009) |
| `epad` | 10 | string×10 | `10+2*x` (1220013) | `4*x` (13111005) |
| `w` | 9 | string×9 | `135+2*x` (21110002) | `200+3*x` (21120002) |
| `u` | 9 | string×9 | `5` (2301002) | `14+2*u(x/5)` (35100008) |
| `jump` | 8 | string×8 | `4*x` (13111005) | `x` (14101003) |
| `epdd` | 8 | string×8 | `20*x` (13111005) | `15*x` (1320009) |
| `emdd` | 8 | string×8 | `20*x` (13111005) | `15*x` (1320009) |
| `itemConNo` | 8 | string×8 | `1` (14111000) | `1` (2311002) |
| `itemCon` | 8 | string×8 | `4006001` (14111000) | `4006000` (2311002) |
| `selfDestruction` | 7 | string×7 | `180+5*x` (33101008) | `250+5*x` (35111002) |
| `pdd` | 7 | string×7 | `10*x` (1001003) | `20*x` (11001001) |
| `eva` | 6 | string×6 | `10+7*x` (13001002) | `6*x` (1320009) |
| `pad` | 6 | string×6 | `x` (1101006) | `x` (11101003) |
| `asrR` | 6 | string×6 | `x` (1220006) | `10+x` (2321005) |
| `acc` | 6 | string×6 | `10+7*x` (13001002) | `6*x` (1320009) |
| `mmpR` | 5 | string×5 | `2*x` (12000005) | `2*x` (2000006) |
| `mdd` | 5 | string×5 | `8*x` (12001002) | `5*x` (1301006) |
| `morph` | 5 | string×5 | `1003` (13111005) | `1000` (15111002) |
| `t` | 5 | string×5 | `0.5*x` (1120004) | `0.5*x` (1220005) |
| `er` | 5 | string×5 | `2*x` (3110007) | `2*x` (3210007) |
| `pddR` | 5 | string×5 | `15*x` (32120009) | `30` (35111013) |
| `terR` | 5 | string×5 | `x` (1220006) | `x` (32110000) |
| `madX` | 4 | string×4 | `3*x` (2120009) | `3*x` (2220009) |
| `bulletConsume` | 4 | string×4 | `3` (14111002) | `3` (4111005) |
| `subProp` | 4 | string×4 | `30+5*x` (2111007) | `30+5*x` (2211007) |
| `hpCon` | 3 | string×3 | `80-2*x` (15111005) | `14+4*u(x/3)` (22161003) |
| `hp` | 3 | string×3 | `100+20*x` (1320008) | `10*x` (2301002) |
| `emhp` | 3 | string×3 | `50*x` (35001002) | `600+20*x` (35120000) |
| `criticaldamageMax` | 3 | string×3 | `x` (3121002) | `x` (3221002) |
| `expR` | 3 | string×3 | `30` (35111013) | `30` (5111007) |
| `emmp` | 3 | string×3 | `50*x` (35001002) | `600+20*x` (35120000) |
| `itemConsume` | 2 | string×2 | `2331000` (5211004) | `2331001` (5211005) |
| `mddR` | 2 | string×2 | `15*x` (32120009) | `2*x` (4341006) |
| `mp` | 2 | string×2 | `2*x` (1110000) | `2*x` (11110000) |
| `subTime` | 1 | string×1 | `3+u(x/4)` (1201006) | — |
| `padX` | 1 | string×1 | `x` (35100000) | — |
| `action` | 1 | string×1 | `slashStorm2` (4311003) | — |
| `moneyCon` | 1 | string×1 | `270+15*x` (4111004) | — |
| `mesoR` | 1 | string×1 | `2*x` (4220009) | — |

### Type outliers to design for

- `lt` / `rb` are **always `vector`** (WZ `Shape2D#Vector2D`), never expressions. They are
  the only vector-typed keys.
- `maxLevel` is int 632× and **string 3×** (§7).
- **Exactly two keys are ever int-typed besides `maxLevel`**:
  - `/311/skill/3111003/common/time` = int `0`
  - `/3311/skill/33111005/common/x` = int `1`
- Every other `common` child in the archive is a `string`.

---

## 3. Complete expression grammar

Tokenized **all 2 374 string values under all 635 `common` nodes** (not a sample).

### Value classification

| class | count |
|---|---|
| plain integer literal (`^-?[0-9]+$`) | 543 |
| matches the strict expression shape | 1 829 |
| matches neither | **2** |

The two exceptions:

1. `/211/skill/2111002/common/damage` = `" 375+5*x"` — **a leading ASCII space**. This is the
   only whitespace-bearing value in the entire archive; an evaluator must trim.
2. `/431/skill/4311003/common/action` = `"slashStorm2"` — **not an expression at all**;
   `action` names a client animation. It is the only non-numeric, non-expression value.

No empty string values (0). No value contains a comma (0).

### Operators (complete, whole-archive)

| operator | occurrences |
|---|---|
| `*` | 1 298 |
| `+` | 1 198 |
| `/` | 684 |
| `-` | 204 |

**That is the entire operator set.** No `%`, no `^`, no comparison, no boolean, no ternary,
no assignment. The tokenizer's unclassified-character bucket is **empty (0)** — every byte
in every `common` string value was classified.

### Function calls (complete)

| function | occurrences |
|---|---|
| `d` | 398 |
| `u` | 286 |

Only two. Semantics (from usage, unambiguous): `d(…)` = floor / round-down, `u(…)` =
ceil / round-up. **Every call site is arity-1.**

Distinct argument forms across all 684 call sites — **all 20 are of the form `x/N`**:

```
d(x/2)  121   u(x/2)   85   d(x/5)   78   u(x/5)   59
d(x/4)   52   u(x/10)  49   d(x/6)   49   u(x/3)   35
d(x/3)   35   d(x/10)  28   u(x/4)   24   u(x/7)   23
d(x/7)   11   d(x/11)  11   d(x/15)  11   u(x/6)    6
u(x/8)    3   d(x/8)    2   u(x/19)   1   u(x/15)   1
```

Distinct divisors: `2 3 4 5 6 7 8 10 11 15 19`. The `/` operator therefore only ever
appears **inside** a `u()`/`d()` argument in this archive — there is no bare division at
top level.

### Free variable identifiers (complete)

| identifier | occurrences |
|---|---|
| `x` | 1 829 |
| `slashStorm2` | 1 (the `action` value, not a variable) |

**`x` is the only variable in the entire grammar.** It is the skill level. There is **no**
`y`, `u`, `d`, `level`, or any other free identifier used as a variable. (`x`, `y`, `z`,
`u`, `v`, `w`, `t` do appear as *key names* — but their *values* are expressions over `x`.)

### Numeric literals

133 distinct literals. **Exactly one has a decimal point: `0.5`.**

### Grammar (derived, complete for this archive)

```
expr    := term (op term)*
op      := '+' | '-' | '*' | '/'
term    := ['-'] atom
atom    := number | 'x' | func '(' expr ')'
func    := 'u' | 'd'
number  := [0-9]+ ('.' [0-9]+)?          -- only decimal seen: 0.5
```

with the caveat that in observed data `/` only occurs inside a func argument, and func
arguments are always exactly `x/<int>`.

---

## 4. Nesting, parentheses, floats, unary minus

| property | answer | evidence |
|---|---|---|
| **Nested function calls** | **NEVER.** 0 of 2 374 | no `u(`/`d(` occurs inside another call's parens |
| **Parentheses for precedence** (non-call parens) | **NEVER.** 0 of 2 374 | every `(` in the archive is immediately preceded by `u` or `d` |
| **Floats / decimals** | **Yes, 5 occurrences, all the literal `0.5`** | see below |
| **Unary minus** | **Yes, 45 occurrences** | see below |

Decimal-literal expressions (complete list, 5):

```
/112/skill/1120004/common/t     = "0.5*x"
/122/skill/1220005/common/t     = "0.5*x"
/132/skill/1320005/common/t     = "0.5*x"
/2112/skill/21120004/common/t   = "0.5*x"
/2112/skill/21120007/common/t   = "5+0.5*x"
```

Unary-minus examples (10 of 45):

```
/110/skill/1101004/common/x      = "-2"
/1110/skill/11101001/common/x    = "-2"
/120/skill/1201004/common/x      = "-2"
/120/skill/1201006/common/x      = "-10-1*x"
/120/skill/1201006/common/y      = "-10-1*x"
/1210/skill/12101001/common/x    = "-4*x"
/1210/skill/12101004/common/x    = "-2"
/130/skill/1301004/common/x      = "-2"
/1310/skill/13101001/common/x    = "-2"
/1310/skill/13101006/common/speed = "-20+x"
```

Note: `-2` is classified above as a "plain integer" and also carries a unary minus — an
evaluator must accept a leading `-` on a bare literal.

### Most complex expression found

Maximum operator count anywhere in the archive is **4**, tied between two verbatim-identical
values:

```
skill 21001003  (image /2100)  key: x   value: "-1-1*u(x/10)"
skill 22141002  (image /2214)  key: x   value: "-1-1*u(x/10)"
```

This is the ceiling of the grammar's complexity: unary minus + subtraction + multiplication
+ one single-level `u()` call whose argument is a division.

Longest by character count (14 chars, 3 operators):

```
skill 11101004  (image /1110)  key: range  value: "150+50*u(x/10)"
skill 4341005   (image /434)   key: range  value: "350+50*d(x/15)"
```

---

## 5. `common` vs `level` key-name collisions

Distinct keys seen under any `level/<n>` entry (34):

```
acc attackCount ball cooltime criticaldamageMax damage damagepc dateExpire dot
dotInterval dotTime eva fixdamage hit hs itemCon itemConNo jump lt mad mastery
mdd mobCount mpCon pad pdd prop range rb speed time x y z
```

**28 names appear in both** `common` and `level`:

```
acc attackCount cooltime criticaldamageMax damage dot dotInterval dotTime eva
itemCon itemConNo jump lt mad mastery mdd mobCount mpCon pad pdd prop range rb
speed time x y z
```

Level-only names (6): `ball damagepc dateExpire fixdamage hit hs`.
Common-only names (37): the remainder of §2's 65, e.g. `maxLevel mhpR emhp emmp epad
epdd emdd terR asrR er expR mesoR moneyCon morph selfDestruction subProp subTime
ignoreMobpdpR bulletCount bulletConsume itemConsume damR madX padX pddR mddR mmpR cr
criticaldamageMin hp hpCon mp t u v w z action`.

**Does any collide-by-name key mean something different? Findings:**

- **No.** For all 28 collisions the *quantity* is identical; only the *representation*
  differs: under `common` the value is a formula over level `x`, under `level/<n>` it is
  the already-materialized value for level `n`. This is directly demonstrable on skill
  2211002 (§6): `common/mpCon = "20+6*d(x/4)"` vs `level/1/mpCon = 21`; `common/lt =
  (-250,-150)` vs `level/1/lt = (-110,-82)`. Same field, two encodings.
- **One genuine hazard, not a name collision but a namespace collision:** the key name
  `x` under `common` (315 occurrences) is a *skill parameter* (a generic "x" value the
  client applies to the skill), while the identifier `x` *inside every expression* is the
  *skill level*. They are unrelated. An evaluator must not feed `common/x`'s value in as
  the variable `x`. Concretely, `/110/skill/1101004/common/x = "-2"` and
  `/3311/skill/33111005/common/x = 1` (int) — those are parameter values, and the same
  skills' other keys evaluate over level.
- `lt`/`rb` are `vector` under **both** `common` and `level`, so they are the one pair
  that is never an expression on either side and must be passed through, not evaluated.
- `maxLevel` exists **only** under `common` — it has no `level`-subtree counterpart, so
  there is no collision to resolve for it.

---

## 6. The exact list of skills with BOTH `common` and `level`

**Exactly 2, confirming the memory note: `2211002` and `2211006`** (both in image `/2211`,
job 2211 — Evan). Full verbatim dumps:

### skill 2211002

`common` (7 children, all leaves):
```
maxLevel [int]    = 20
mpCon    [string] = "20+6*d(x/4)"
damage   [string] = "350+5*x"
mobCount [string] = "6"
time     [string] = "u(x/10)"
lt       [vector] = (-250,-150)
rb       [vector] = (250,150)
```

`level`: **30 entries, 1..30**

```
level[1]:
  hs       [string] = "h1"
  mpCon    [int]    = 21
  mad      [int]    = 32
  mastery  [int]    = 1
  mobCount [int]    = 6
  time     [int]    = 1
  lt       [vector] = (-110,-82)
  rb       [vector] = (110,83)

level[30]:
  hs       [string] = "h30"
  mpCon    [int]    = 50
  mad      [int]    = 90
  mastery  [int]    = 10
  mobCount [int]    = 6
  time     [int]    = 2
  lt       [vector] = (-200,-150)
  rb       [vector] = (200,150)
```

### skill 2211006

`common` (5 children, all leaves):
```
maxLevel [int]    = 20
mpCon    [string] = "16+6*d(x/7)"
damage   [string] = "460+5*x"
time     [string] = "u(x/10)"
x        [string] = "1"
```

`level`: **30 entries, 1..30**

```
level[1]:
  hs      [string] = "h1"
  mpCon   [int]    = 14
  mad     [int]    = 80
  mastery [int]    = 1
  time    [int]    = 1
  x       [int]    = 1

level[30]:
  hs      [string] = "h30"
  mpCon   [int]    = 22
  mad     [int]    = 140
  mastery [int]    = 10
  time    [int]    = 2
  x       [int]    = 1
```

### Facts the "which wins" decision must account for

1. **The two disagree on level count.** `common/maxLevel = 20`, but the `level` subtree has
   **30** entries (1..30). A "common wins" rule caps these skills at 20; a "level wins"
   rule yields 30.
2. **The two disagree on values.** For 2211002 level 1: `common/mpCon` evaluated at x=1 is
   `20+6*d(1/4)` = `20+6*0` = **20**, but `level/1/mpCon` = **21**. At level 30 the formula
   gives `20+6*d(30/4)` = `20+6*7` = **62**, vs `level/30/mpCon` = **50**. They are not
   reconcilable by rounding — the tables were hand-tuned away from the formula.
   For 2211006 level 1: `16+6*d(1/7)` = **16** vs `level/1/mpCon` = **14**.
3. **`level` carries keys `common` does not**: `mad` and `mastery` and `hs` appear only in
   the level tables for both skills. `common` carries `damage` (and for 2211002, `lt`/`rb`
   and `mobCount`) that the level tables partly duplicate and partly omit — 2211002's
   `damage` exists **only** under `common`, while `mad` exists **only** under `level`.
   Neither subtree is a superset of the other.
4. Therefore the two subtrees are **not** alternative encodings of the same data for these
   two skills; a correct reader must merge (level table wins per-key where present,
   `common` fills the keys the table lacks) or must explicitly document that it drops one.
   Choosing "common wins" would lose `mad`/`mastery`; choosing "level wins" would lose
   `damage`. **This is a design decision, not something the archive settles.**

---

## 7. `maxLevel` under `common`

- Present on **every one of the 635 `common` nodes** — 0 nodes lack it.
- Type: **`int` 632×, `string` 3×**. The three string occurrences, verbatim:
  ```
  /1310/skill/13100004/common/maxLevel [string] = "10"
  /132/skill/1320005/common/maxLevel   [string] = "30"
  /3212/skill/32120001/common/maxLevel [string] = "30"
  ```
- **It is NEVER an expression.** All 3 string values are plain integer literals; no `x`,
  no operator, no function call ever appears in a `maxLevel` value.
- Distinct values (7): `1 5 10 15 20 25 30`. Distribution:
  `20`×331, `30`×166 (164 int + 2 string), `10`×94 (93 int + 1 string), `5`×24, `15`×17,
  `25`×2, `1`×1.

**Answer: always an integer semantically; but a parser must accept it as either a WZ int
node or a WZ string node holding a decimal integer.**

---

## 8. Non-leaf `common` children

**None. Zero.** Across all 635 `common` nodes and all 2 946 children, there is not a single
`sub` (nested `imgdir`), `canvas`, or `convex` child. Every `common` child is a leaf of type
`string`, `int`, or `vector`. `common` is exactly one level deep.

---

## Summary for the PRD author

An evaluator for v95 `common` needs to:

1. Read `common` as a flat map of 65 possible keys → leaf value; types `string | int | vector`.
2. Treat `lt`/`rb` (vector) as pass-through — never evaluate.
3. Treat `maxLevel` as an integer, accepting int-or-string node types.
4. Treat `action` (1 occurrence) as an opaque string, not an expression.
5. `TrimSpace` the value before parsing (1 occurrence needs it: skill 2111002 `damage`).
6. Evaluate the remaining string values with a grammar of: integer and one decimal literal,
   the single variable `x` (= skill level), operators `+ - * /`, unary minus, and two
   arity-1 functions `u` (ceil) and `d` (floor). **No nesting, no precedence parentheses,
   no other identifiers, no other operators.** A correct precedence implementation
   (`* /` above `+ -`) is required — the data does rely on it (`-1-1*u(x/10)`).
7. Decide and document the `common`-vs-`level` merge policy for skills 2211002 and 2211006.
