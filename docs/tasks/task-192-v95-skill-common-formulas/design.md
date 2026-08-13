# v95 Skill `common` Formula Nodes — Design

Version: v1
Status: Draft (design phase)
Created: 2026-08-07
Inputs: [`prd.md`](prd.md), [`wz-common-grammar.md`](wz-common-grammar.md)

---

## 0. Summary

Three components, one integration seam:

1. **`skill/formula`** — a new pure package in atlas-data holding a
   tokenizer + recursive-descent parser + evaluator for the `common`
   expression grammar. Parse once, evaluate per level.
2. **A declared `common` key table** — the single place that says, for each
   `common` child key, whether it is an expression, a pass-through vector, an
   opaque string, or skill-level metadata, and what integer range its target
   field admits.
3. **Per-level `xml.Node` synthesis** — the `common` subtree is expanded into
   `maxLevel` synthetic `xml.Node` values that are then fed through the
   **existing, unmodified** `getEffect`. This is the load-bearing decision: it
   makes FR-5.3 (identical post-processing), FR-5.4 (identical defaults), and
   FR-6.1 ("populate on both read paths") fall out of the structure instead of
   being maintained by hand.

Plus error/stat plumbing (FR-7) and — an addition to the PRD's service impact —
surfacing the new attributes in the atlas-ui skill level table, which is a
whitelist and would otherwise render nothing for all 35 new keys.

### What this design changes relative to the PRD

Two substantive corrections, both sourced from the GMS v95.0 client binary
rather than from the archive census:

- **FR-3.3 is wrong in detail.** `u()` is not `ceil` and `d()` is not `floor`.
  See §1. The difference is unobservable on this archive and would become a
  silent divergence on any future one.
- **FR-3.2's "standard precedence" is not what the client implements.** The
  client's precedence is `*` → `/` → `-` → `+`, one operator per rewrite pass.
  See §1. Again unobservable on this archive; §1.4 explains exactly when it
  would stop being unobservable.

Both are resolved by specifying the client's semantics as the contract and
pinning the "agrees on this archive" claim with a differential test (§10.3)
rather than an assumption.

Also proposed: three amendments to the PRD's acceptance criteria (§12).

---

## 1. The client's evaluator (evidence, and why it changes the spec)

The PRD's grammar was derived by tokenizing the archive. That establishes what
the *data* contains; it cannot establish what the *client* computes from it.
OQ-1 asked for the client's behaviour. It is available and was read.

**Source.** IDA session `79906a1e`, `GMS_v95.0_U_DEVM.exe.i64`
(`E:\Programs\Nexon\IDBs_v9\GMS\v95_0\`). PDB-backed symbols; the two relevant
functions are named in the binary, not inferred:

| Address | Symbol |
|---|---|
| `0x6fe560` | `SKILLLEVELDATA::GetParsedCommonData(ZXString<char>, long) → long` |
| `0x6f9300` | `SKILLLEVELDATA::GetArithmeticData(ZXString<char>, int bCeiling) → long` |

(Float and ULONG siblings exist at `0x6fd950` / `0x6fdf60` and
`0x6f8460` / `0x6f8ba0`. `CSkillInfo::LoadLevelDataCommon` at `0x6f47a0` is the
loader that stores the raw strings into `SKILLLEVELDATACommon`; it performs no
arithmetic — evaluation is deferred to these accessors at use time.)

### 1.1 `GetParsedCommonData(expr, level)` — the outer driver

Decompiled control flow, in order:

1. `_strlwr` the whole expression. **Evaluation is case-insensitive.**
2. `Format(sParam, "%d", nLevel)` then `Replace(expr, <var-token>, sParam)` —
   the level is substituted **textually, before any parsing**, replacing the
   free variable token (a StringPool-interned literal; the archive census
   independently establishes `x` as the only free identifier — FR-3.1).
3. Three sequential rewrite loops, each `Find`ing an opening marker and the
   next `)`, slicing the interior with `Mid`, calling `GetArithmeticData` on
   it, and `Replace`-ing the whole `marker…)` span with the `"%d"`-formatted
   result:

   | Loop | Marker slice offset | `bCeiling` | Marker |
   |---|---|---|---|
   | 1 | `+2` (2-char marker) | **1** | `u(` |
   | 2 | `+2` (2-char marker) | 0 | `d(` |
   | 3 | `+1` (1-char marker) | 0 | `(` |

4. Finally `GetArithmeticData(remaining, bCeiling=0)` on the whole reduced
   string and return it.

Two consequences worth stating plainly:

- **Bare parentheses are supported** (loop 3), evaluated with truncation. The
  archive contains none (FR-3.8), but the client accepts them.
- **Nesting is not handled correctly.** Each loop takes the *first* opening
  marker and the *first* `)` after it, so `u(d(x/2))` mis-slices. The archive
  contains zero nesting. FR-3.8 asks the evaluator to "handle nesting
  correctly rather than assume the shape" — this design does implement proper
  nesting (a recursive-descent parser gets it for free), which is a
  *deliberate* superset of the client, documented in §4.4.

### 1.2 `GetArithmeticData(expr, bCeiling)` — the arithmetic core

It is a **string-rewriting machine**, not an AST evaluator. Each pass:

1. Find the highest-priority operator present, scanning in the fixed order
   `*`, `/`, `-`, `+` (the decompiled `nOperator` codes are 1, 2, 4, 3
   respectively). If the found `-` sits at index 0, re-scan from index 1 and
   set a flag that suppresses `-` as an operand boundary — **this is the unary
   minus handling** (FR-3.5).
2. Walk backwards/forwards to the operand boundaries, `Mid` out P1 and P2, and
   `atof` both. **Division is `nP1 / nP2` on floats — real division (FR-3.4
   confirmed).** Subtraction is `nP1 - nP2`; multiplication and addition are
   commutative in the decompilation.
3. `Format(tmp, "%d", nResult)` and `Replace` the `P1 op P2` span with that
   text. **The intermediate result is truncated to an integer after every
   single binary operation**, then re-parsed by `atof` on the next pass.
4. When no operator remains, `atoi` the string and return.

`bCeiling` applies **only** on the operation that spans the entire remaining
string (P1 starts at index 0 and P2 runs to the end), and does:

```
nResult = (int)(nResult + 0.999999)
```

### 1.3 Correction to FR-3.3 — `u` and `d` are not `ceil` and `floor`

| | Client | FR-3.3 as written |
|---|---|---|
| `d(v)` | `trunc(v)` — truncation **toward zero** | `floor(v)` |
| `u(v)` | `trunc(v + 0.999999)` | `ceil(v)` |

They agree for `v ≥ 0` (with `u`'s `0.999999` fudge chosen so exact integers
stay put: `trunc(2 + 0.999999) = 2`). They disagree for negatives:
`floor(-0.25) = -1` but `trunc(-0.25) = 0`; `ceil(-1.5) = -1` but
`trunc(-1.5 + 0.999999) = 0`.

Every `u()`/`d()` argument in the archive has the form `x/N` with `x ≥ 1` and
`N > 0`, so no negative ever reaches either function today and the archive
cannot distinguish the two definitions. **Implement the client's form.** Using
`math.Ceil`/`math.Floor` would be a correct-looking line that silently
diverges the first time a future archive writes `d(-x/2)`.

### 1.4 Correction to FR-3.2 — precedence is `*` → `/` → `-` → `+`

Not "`* /` bind tighter than `+ -`" with left-to-right within a tier. The
client resolves *all* multiplications first, then divisions, then
subtractions, then additions, one operator per pass.

Where this is observable, and why it isn't here:

- `+` vs `-` reordering is harmless: for the shapes in this archive
  (`a+b-c`, `a-b`, `-a-b*u(...)`) reassociating subtraction before addition
  yields identical results, because each operator's operands remain the
  adjacent terms.
- `*` vs `/` reordering **is** observable: `x/2*3` at `x=20` gives
  `20/(2*3) = 3` under the client and `(20/2)*3 = 30` under left-to-right.
  FR-3.4 records that `/` never appears outside a `u(`/`d(` argument in this
  archive, and every such argument is exactly `x/N` — so no expression here
  mixes `/` and `*` at the same level. **This is the guard that makes the two
  agree**, and it is a property of the data, not of the grammar.
- Per-operation truncation is observable whenever a non-integer intermediate
  feeds another operator outside a `u()`/`d()`. The only decimal in the
  archive is `0.5` in key `t` (`"0.5*x"`, `"5+0.5*x"`), and for those two
  shapes `trunc(5 + f) = 5 + trunc(f)` holds for `f ≥ 0`, so again they agree.

**Design position:** implement the client's precedence and per-operation
truncation exactly. The three "they happen to agree" arguments above are
premises about the current archive, and premises rot. Implementing the client
means the evaluator stays right when they do.

### 1.5 OQ-1 (rounding for `t`) — resolved

The client evaluates `t` through the same string machine. `GetParsedCommonData`
(the `long` accessor) truncates via `"%d"`/`atoi`; a `GetParsedCommonDataFloat`
sibling exists for callers that want the fraction. Since every other `common`
key in this design lands in an integer field, and `t` has no consumer at all
(OQ-4), **`t` is modelled as a truncated `int32`**, consistent with
`GetParsedCommonData`. The evaluator internally carries `float64` throughout,
so if a consumer later needs sub-second `t`, adding a float-typed field is a
one-line change with no evaluator work.

---

## 2. Architecture

```
                     Skill.wz  <imgdir name="<skillId>">
                                       |
                     +-----------------+------------------+
                     |                                    |
              has "common"?  ---- yes ---->        no ----+---> has "level"?
                     |                                          |
                     v                                          v
        +------------------------------+                 (unchanged today)
        | 1. read commonKeys table     |                        |
        | 2. classify each child leaf  |                        |
        | 3. parse each expression ONCE|                        |
        | 4. for L in 1..maxLevel:     |                        |
        |      evaluate -> int64       |                        |
        |      range-check -> string   |                        |
        |      emit synthetic xml.Node |                        |
        +---------------+--------------+                        |
                        |                                       |
                        +--------------> getEffect() <----------+
                                    (ONE implementation)
                                             |
                                     effect.RestModel
```

### 2.1 The synthesis seam — why the `common` path produces an `xml.Node`

`getEffect` (`skill/reader.go:169-426`) is ~260 lines: seconds→ms conversion,
the `prop`/`hpR`/`mpR` ÷100 rules, the barrier/map-protection statups, the
over-time statup block, the 40-arm per-skill statup / monsterStatus chain, mob
information, abnormal statuses. FR-5.3 requires the `common` path to apply
*all* of it, identically, forever.

There are exactly three ways to get that:

| Option | Verdict |
|---|---|
| **A. Duplicate** `getEffect` for `common` | Rejected. Guaranteed drift: the next per-skill statup arm added to one copy and not the other is a silent, per-version behavioural fork. |
| **B. Refactor** `getEffect` to read from an abstract `valueSource` interface | Rejected. Touches every one of ~60 read sites, changes the `level` path's code (regression surface for a task whose acceptance criterion is "`level` behaves as today"), and buys nothing: `xml.Node` *already is* that interface. |
| **C. Synthesize** a per-level `xml.Node` and call the existing `getEffect` | **Chosen.** Zero lines of `getEffect` change for FR-5.3/5.4. New keys (FR-6) are added once, in `getEffect`, and are automatically populated on *both* paths. |

Two details make C exact rather than approximate:

- **Node name.** The synthetic node's `Name` is `strconv.Itoa(level)`, so
  `levelFromNode` (`reader.go:489-495`) — which `mountStatupsForSkill` depends
  on for the per-level SpaceShip vehicle id — works unmodified.
- **Sub-nodes.** `common` is exactly one level deep, zero non-leaf children
  archive-wide (FR-2.1). So the synthetic node has no `ChildNodes`, and
  `getMob` / `applyMobInformation` correctly find nothing and fall to their
  defaults — which is exactly what a `level` node without those children does.

### 2.2 FR-7.5 compliance under synthesis

FR-7.5 forbids routing `common` expression values through
`xml.GetIntegerWithDefault`, because its silent-default behaviour
(`xml/model.go:82-102`) is precisely how a formula string degrades to `0`
without a trace.

Under synthesis, `getEffect` *does* call `GetIntegerWithDefault` on the
synthetic node — but never on an expression. The synthesis step evaluates
first and writes only canonical `strconv.FormatInt` output, which
`ParseInt` cannot fail on. The class of bug FR-7.5 targets is eliminated at
its source: **an expression that cannot be evaluated never reaches node
synthesis at all** — it becomes an FR-7.1 error and the key is simply absent
from the synthetic node (falling to the same default the `level` path uses for
an absent key).

This satisfies FR-7.5's intent rather than its letter. It is pinned by an
explicit test (§10.2) asserting that every `IntegerNode.Value` on a synthesized
node round-trips through `strconv.ParseInt` without error.

---

## 3. Component 1 — `skill/formula`

Location: `services/atlas-data/atlas.com/data/skill/formula/`. New package
inside atlas-data (FR-8.1): no second consumer exists, and
`libs/atlas-script-core`'s `EvaluateArithmeticExpression` shares neither
grammar nor consumer.

### 3.1 API

```go
// Expr is a parsed, level-independent expression. Safe for repeated
// evaluation; parse once per common key, evaluate maxLevel times (FR-8.3).
type Expr struct { /* unexported AST root */ }

// Parse tokenizes and parses src. It is pure: no tenant, no context, no
// logger (FR-8.2). The returned error names the offending offset and token.
func Parse(src string) (Expr, error)

// Evaluate computes the expression with the free variable x bound to level.
// It returns the client-faithful integer result (see design §1).
func (e Expr) Evaluate(level int) (int64, error)

// EvaluateFloat is Evaluate without the final truncation, retained for a
// future float-typed field (design §1.5). Not used by the reader today.
func (e Expr) EvaluateFloat(level int) (float64, error)
```

### 3.2 Grammar as implemented

Recursive descent, with the client's precedence tiers from §1.4 (loosest
first):

```
expr    := addTier
addTier := subTier ('+' subTier)*
subTier := divTier ('-' divTier)*
divTier := mulTier ('/' mulTier)*
mulTier := unary   ('*' unary)*
unary   := ['-'] atom
atom    := number | 'x' | func '(' expr ')' | '(' expr ')'
func    := 'u' | 'd'
number  := [0-9]+ ('.' [0-9]+)?
```

Evaluation rules, each traceable to §1:

- Input is `strings.TrimSpace`d (FR-3.7) and lowercased (§1.1 step 1) before
  tokenizing. Interior whitespace is skipped by the tokenizer.
- All arithmetic is `float64` (FR-3.6). Division is float division (FR-3.4).
- **After each binary operation the result is truncated toward zero** (§1.2
  step 3). `truncate(v) = math.Trunc(v)`.
- `d(v)` → `math.Trunc(v)`. `u(v)` → `math.Trunc(v + 0.999999)`. Bare
  `(v)` → `math.Trunc(v)`. (§1.1, §1.3.)
- `x` evaluates to `float64(level)`.
- The top-level result is truncated toward zero and returned as `int64`.

Note the tier ordering is *not* a typo: `-` binds tighter than `+`, and `/`
binds tighter than `-` but looser than `*`. §1.4 is the justification; the
package doc comment carries the same explanation so the next reader does not
"fix" it.

### 3.3 Bounds (NFR-5)

The evaluator is not a general interpreter. It rejects, as parse errors:

- any identifier other than `x` — there is no symbol table;
- any function other than `u`/`d`, and any arity other than 1;
- input longer than **256 bytes** (archive maximum is 14);
- parse recursion deeper than **16** (archive maximum nesting is 0);
- more than **64 tokens**.

These are backstops against a malformed or hostile archive hanging or
stack-overflowing the ingest job, not tuning knobs. They sit ~20× above the
observed maxima.

---

## 4. Component 2 — reader integration

### 4.1 The declared key table

One table, in `skill/reader.go` (or a sibling `skill/common.go`), is the single
source of truth for how each `common` child is handled:

```go
type commonKind int

const (
    commonExpr   commonKind = iota // evaluate per level
    commonVector                   // pass through unevaluated (lt, rb)
    commonOpaque                   // never evaluate (action)
    commonMeta                     // skill-level, not per-effect (maxLevel)
)

type commonKey struct {
    name string
    kind commonKind
    min  int64 // inclusive target-field range, for commonExpr only
    max  int64
}
```

Rationale for `min`/`max`: every key ultimately lands in a narrow Go type
(`x` is `int16`, `MHPRRate` is `uint16`, `damage` is `uint32`). An unchecked
narrowing turns an evaluated `-2` into `uint16(65534)` — a silent-wrap bug of
exactly the kind FR-7.5 exists to prevent, just one layer down. Synthesis
range-checks against the declared bounds and treats a violation as an FR-7.1
evaluation failure.

This check applies **only to the `common` path**. The `level` path's
conversions are left exactly as they are today, because FR-6.5 and the
"`level` behaves as today" criterion forbid changing them.

An **unknown** `common` key — one not in the table — is treated as
`commonExpr` with the full `int32` range and is logged at **WARN** naming the
skill and key. It is not an error (it costs nothing to carry) but it is
visible, so a future archive introducing a key gets noticed instead of
silently dropped.

### 4.2 `produceSkill` changes (FR-1)

```
common, hasCommon := node.ChildByName("common")
if hasCommon {
    effects, maxLevel, failures = expandCommon(l, t, skillId, common)
} else if level, hasLevel := node.ChildByName("level"); hasLevel {
    effects  = getEffects(t, skillId, buff, level.ChildNodes)   // unchanged
    maxLevel = clamp255(len(effects))                            // unchanged
} else {
    effects, maxLevel = nil, 0                                   // FR-1.3
}
```

- **FR-1.2 (COMMON wins unconditionally):** the `level` subtree is not read
  when `common` is present. The two-skill regression (2211002, 2211006 lose
  `mad`/`mastery` and cap at 20 instead of 30) is accepted per the PRD and
  pinned by a test so it stays a decision rather than becoming a bug report.
- **FR-1.4:** detection is `ChildByName("common") == nil`, structural only.
  No region/major/minor is consulted anywhere in this change.
- **FR-5.2:** `maxLevel` comes from the declared `common/maxLevel`, not from
  `len(effects)`.

### 4.3 `expandCommon`

```
1. maxLevel := parse common/maxLevel   (accept int node OR string node, FR-2.3)
   - missing / non-integer / <= 0  -> FR-7.4 failure, return zero effects
2. for each leaf child of common:
     classify via the key table
     commonMeta   -> skip (maxLevel, and any future skill-level key)
     commonOpaque -> skip (action, FR-2.5)
     commonVector -> remember the raw PointNode for pass-through (FR-2.4)
     commonExpr   -> formula.Parse(value)  [ONCE — FR-8.3]
                     parse error -> FR-7.1 failure, drop this key only
   Leaf values may be int-typed rather than string-typed (FR-2.6); an int leaf
   is handed to Parse as its decimal text, which parses to a constant.
3. for level := 1 .. maxLevel:
     synth := xml.Node{Name: strconv.Itoa(level)}
     for each parsed expr, in the table's declared order (NFR-4 determinism):
         v, err := expr.Evaluate(level)
         err or out-of-range -> FR-7.1 failure, omit this key at this level
         synth.IntegerNodes = append(..., xml.IntegerNode{name, FormatInt(v)})
     synth.PointNodes = the remembered lt/rb vectors, verbatim
     effects = append(effects, getEffect(t, skillId, buff, synth))
```

Iteration order is the declared table order, never a Go map range — NFR-4
requires byte-identical documents for identical input.

Note step 2's parse-once placement: `maxLevel` is at most 30, so parsing once
and evaluating 30 times replaces ~30 parses per key with 1. Across the archive
that is ~1 829 parses instead of ~55 000 (NFR-1).

### 4.4 Deliberate superset of the client

The client's `GetParsedCommonData` mis-slices nested calls (§1.1). This design
parses properly and therefore evaluates `u(d(x/2))` correctly, per FR-3.8's
"SHOULD still handle them correctly". This is a **conscious divergence on
input the archive does not contain**: for every expression that exists today
the two are identical, and where they differ the client's behaviour is a bug,
not a contract. Recorded here so it is not later mistaken for an oversight.

---

## 5. Component 3 — effect model extension (FR-6)

### 5.1 Naming rule

**Go field name = PascalCase of the wz key; JSON tag = the wz key verbatim.**

No key is given a "descriptive" name. `epad` becomes `Epad`, not
`EnhancedWeaponAttack`; `asrR` becomes `AsrR`. The meanings of these keys are
not verified from any source — OQ-4 says so explicitly for the single letters,
and the same caution applies to the abbreviations. Encoding a guess into a
field name makes it permanent and load-bearing. The JSON tag matching the wz
key also keeps the existing convention (`hpR`, `mpR`, `mobCount`, `damage`).

Two exceptions, both argued below.

### 5.2 The 35 keys

33 new fields plus 2 reuses. All new numeric fields are **`int32`**: the
evaluator produces a signed integer, several keys are legitimately negative
(`x = "-2"`, `"-1-1*u(x/10)"`), and no downstream consumer exists yet whose
range would justify a narrower type.

| wz key | Go field | json tag | type |
|---|---|---|---|
| `range` | `Range` | `range` | int32 |
| `mastery` | `Mastery` | `mastery` | int32 |
| `z` | `Z` | `z` | int32 |
| `dot` | `Dot` | `dot` | int32 |
| `cr` | `Cr` | `cr` | int32 |
| `dotInterval` | `DotInterval` | `dotInterval` | int32 |
| `dotTime` | `DotTime` | `dotTime` | int32 |
| `damR` | `DamR` | `damR` | int32 |
| `criticaldamageMin` | `CriticaldamageMin` | `criticaldamageMin` | int32 |
| `mhpR` | *(reuse)* `MHPRRate` | `MHPRRate` | uint16 |
| `v` | `V` | `v` | int32 |
| `ignoreMobpdpR` | `IgnoreMobpdpR` | `ignoreMobpdpR` | int32 |
| `epad` | `Epad` | `epad` | int32 |
| `w` | `W` | `w` | int32 |
| `u` | `U` | `u` | int32 |
| `epdd` | `Epdd` | `epdd` | int32 |
| `emdd` | `Emdd` | `emdd` | int32 |
| `selfDestruction` | `SelfDestruction` | `selfDestruction` | int32 |
| `asrR` | `AsrR` | `asrR` | int32 |
| `mmpR` | *(reuse)* `MMPRRate` | `MMPRRate` | uint16 |
| `t` | `T` | `t` | int32 |
| `er` | `Er` | `er` | int32 |
| `pddR` | `PddR` | `pddR` | int32 |
| `terR` | `TerR` | `terR` | int32 |
| `madX` | `MadX` | `madX` | int32 |
| `subProp` | `SubProp` | `subProp` | int32 |
| `emhp` | `Emhp` | `emhp` | int32 |
| `criticaldamageMax` | `CriticaldamageMax` | `criticaldamageMax` | int32 |
| `expR` | `ExpR` | `expR` | int32 |
| `emmp` | `Emmp` | `emmp` | int32 |
| `itemConsume` | `ConsumeItemId` | `consumeItemId` | int32 |
| `mddR` | `MddR` | `mddR` | int32 |
| `subTime` | `SubTime` | `subTime` | int32 |
| `padX` | `PadX` | `padX` | int32 |
| `mesoR` | `MesoR` | `mesoR` | int32 |

Each gets: a private field on `effect.ModelBuilder`, a `Set…` builder setter, a
line in `Build()`, and a field on `effect.RestModel`. Each is read in
`getEffect` via `node.GetIntegerWithDefault("<wzkey>", 0)` — one line each,
which by §2.1's construction populates it on **both** the `common` and `level`
paths (FR-6.1).

`damage` (FR-6.2) needs no new field. It is already read at `reader.go:239`
with default 100; the `common` path populates it for free via synthesis. This
is the flagship fix — 277 occurrences.

### 5.3 OQ-3 — reuse `MHPRRate`/`MMPRRate`: **yes**

Evidence: `atlas-ui`'s skill level table already labels `MHPRRate` as
"Max HP Recovery %" and `mhpr` as "HP Recovery"
(`services/atlas-ui/src/lib/skills/level-table.ts:32-35`) — a reading
consistent with the wz key `mhpR` being a max-HP recovery percentage. The
fields are currently never populated by any reader, so every consumer reads
`0` today; the only consumer that binds them at all
(`atlas-channel/.../effect/rest.go:96-97`) copies them into a model field
nothing reads. Populating them is therefore additive in practice.

They stay `uint16` (FR-6.5 forbids retyping). The range check from §4.1
(`0 … 65535`) makes an out-of-range value a loud failure instead of a wrap.

### 5.4 OQ-2 — `itemConsume` vs `itemCon`: separate field, named `consumeItemId`

FR-6.4 forbids folding `common/itemConsume` into the existing `ItemConsume`
field. That field is fed by wz `itemCon` (`reader.go:245`) and — the actual
trap — already carries the JSON tag `"itemConsume"`. So the new key cannot use
its own name as a tag without colliding with a differently-sourced field.

Decision: new field `ConsumeItemId int32`, JSON tag `consumeItemId`, with a
doc comment stating it is wz `common/itemConsume`, distinct from the
`itemConsume` JSON attribute which is wz `itemCon`.

OQ-2's semantic question (are they the same concept renamed post-Big-Bang?)
remains **unresolved** — nothing in the archive or the client settles it. Both
values (`2331000`, `2331001`) are item ids, as are `itemCon`'s, which is
suggestive but not evidence. The two keys never co-occur on the same node, so
if a later reverse-engineering pass proves them identical, merging is a safe,
non-breaking follow-up. Carrying them separately now is the choice that cannot
be wrong.

### 5.5 Document size

35 new attributes × ~12 700 v95 effects. No `omitempty` — the existing model
uses none (except the `lt`/`rb` pointers) and conditional key presence makes
consumer code and golden tests worse. The `content json` column absorbs it; no
schema change (PRD §6).

---

## 6. Component 4 — failure handling and run stats (FR-7)

### 6.1 Threading

`produceSkill` has no logger and no way to report anything but a hard error
today. The chain is
`workers.Skill.Run → registerAllInDirectory → skill.RegisterSkill → skill.Read → produceSkill`.

Design: make derivation stats an explicit return value, mirroring the
`countingRegister` / `logJobDocCount` pattern that already sits immediately
adjacent in `data/workers/skill.go:113-137` for JOB documents.

```go
// skill package
type Stats struct {
    Processed      int
    FromCommon     int
    FromLevel      int
    Neither        int
    SkillsWithFailures int
    Failures           int
}

func (s *Stats) Add(o Stats)

type Derivation struct {
    Models []RestModel
    Stats  Stats
}

func Read(l) (ctx) (np model.Provider[xml.Node]) model.Provider[Derivation]
func (p *ProcessorImpl) RegisterSkill(path string) (Stats, error)
```

`Processor.Register` unwraps `Derivation`, writes `Models`, returns `Stats`.
The worker accumulates across job images and logs the summary.

Why not a package-level registry singleton (the `GetSkillStringRegistry`
pattern)? Because the stats are per-run, not per-tenant-lifetime, and hidden
global mutable state makes the unit tests order-dependent. An explicit return
value is testable and matches the adjacent precedent.

`RegisterSkill`'s signature change requires a `RegisterFunc` adapter in the
worker, exactly analogous to the existing `countingRegister`. `Read`'s only
non-test caller is `skill/processor.go:56`.

### 6.2 Per-failure ERROR (FR-7.1, FR-7.2)

At each failure point (`maxLevel` unparseable — FR-7.4; expression parse
failure; evaluation failure; range violation), emit **one ERROR** carrying, via
`logrus.WithFields`, structured keys: `tenant`, `jobId`, `skillId`, `key`,
`expression`, `level` (where applicable), and the underlying error. Structured
fields, not a formatted sentence, so NFR-3's "greppable by skill id" holds.

Scope of a failure (FR-7.2):

| Failure | Blast radius |
|---|---|
| `maxLevel` missing/unparseable | that **skill** — zero effects, `maxLevel` 0; job image continues |
| one key's expression fails to parse | that **key**, all levels — other keys still expand |
| one key fails to evaluate / is out of range at level L | that **key at level L** — the key falls to `getEffect`'s default |

None of these returns an error from `produceSkill`. Today `reader.go:73-76`
aborts the whole job image on any `produceSkill` error; that path stays for
genuine structural errors (a non-numeric skill node name) but is never reached
by a formula failure.

### 6.3 Run summary (FR-7.3)

At the end of `workers.Skill.Run`, after both `registerAllInDirectory` passes:

```
INFO  skills: processed=954 fromCommon=635 fromLevel=319 neither=0 failures=0
ERROR skills: 12 skill(s) had >=1 common evaluation failure   [only when > 0]
```

The ERROR line is what makes FR-7.2's permissiveness safe: a single ERROR can
scroll past in a 954-skill run, an aggregate at the end cannot.

---

## 7. Component 5 — atlas-ui surfacing (addition to PRD §7)

The PRD states "no other Go service changes", which is correct — but
`services/atlas-ui/src/lib/skills/level-table.ts` builds the skill level table
from an **explicit whitelist** (`FIELD_LABELS`, lines 19-53), and
`services/atlas-ui/src/services/api/skills.service.ts` declares `SkillEffect`
with a closed set of optional numeric keys. Landing this task without touching
either means all 35 new attributes are ingested, persisted, and served — and
render nowhere.

That is a producible gap, not a follow-up: it is ~70 lines of TypeScript.

In scope:

- add the 35 keys to the `SkillEffect` interface (`skills.service.ts`);
- add them to `FIELD_LABELS` with human labels, placed after the existing
  entries so current column order is unchanged.

For the keys whose meaning is unverified, the label is the key itself (`"z"`,
`"u"`, `"v"`, `"w"`, `"t"`) rather than an invented gloss — consistent with
§5.1. `buildLevelTable` already omits all-zero columns, so non-v95 tenants see
no new columns at all.

Verification for this component follows the atlas-ui rule: `npm run build`
(which type-checks tests), not just `vitest`.

Out of scope: `atlas-channel` / `atlas-messages` copies of the effect
RestModel. They decode only the fields they declare; unknown JSON attributes
are ignored. Nothing there reads a new key, so adding fields would be dead
code.

---

## 8. OQ-5 — census of other regions' archives

Unresolved and **in scope for the plan**, not deferred. FR-1.4 makes the
handling universal either way, so it does not gate the code — but "JMS v185 is
post-Big-Bang and plausibly uses `common`" is a claim, and an unverified claim
about whether this task fixes another tenant should not survive into the PR
description.

Method (same as the v95/v92 census that produced `wz-common-grammar.md`): for
each region/version archive in MinIO `atlas-wz`, open `Skill.wz` with
`libs/atlas-wz` `wz.Open`, assert `File.GameVersion()`, walk all root images,
and count skill nodes with `common` / with `level` / with both / with neither.
Report the table in the PR description and, if any non-GMS-v95 archive has
`common` nodes, re-run the tokenizer over its values to confirm the FR-3
grammar is still complete before claiming the fix applies there.

---

## 9. Alternatives considered and rejected

| Alternative | Why rejected |
|---|---|
| Reuse `libs/atlas-script-core`'s `EvaluateArithmeticExpression` | Two integer operands, one operator, no variables, no calls, no parens, no precedence, no decimals. Every one of those is required. (PRD §1; FR-8.1.) |
| Put the evaluator in a new shared lib | One consumer. Project rule: audit before adding a module. A shared lib for a single caller is cost without benefit; promoting later is mechanical. |
| Evaluate with `go/parser` + `go/constant` or a third-party expression library | Both are general interpreters over a much larger grammar — the opposite of NFR-5, and neither reproduces the client's `*`→`/`→`-`→`+` precedence or its per-operation truncation. The whole point is to match a specific quirky machine. |
| Standard `ceil`/`floor` and standard precedence (i.e. FR-3 as literally written) | Agrees with the client on this archive by coincidence of the data (§1.4), diverges silently on any archive with a negative `u()`/`d()` argument or a `/` outside a call argument. |
| Port the client's string-rewriting algorithm literally | Reproduces the client bit-for-bit including its nesting bug, but is unreadable, allocation-heavy (a `Replace` per operator per level), and gives terrible error messages — it cannot say "unexpected token at offset 7", only "the result was garbage". A recursive-descent parser with the client's *semantics* gets the same numbers with real diagnostics. |
| Duplicate `getEffect` for the `common` path | §2.1 option A — guaranteed drift. |
| Abstract `getEffect` behind a `valueSource` interface | §2.1 option B — large blast radius on the path this task must not regress. |
| Version-gate `common` handling to GMS ≥ 95 | Explicitly forbidden by FR-1.4, and the failure mode it invites (a future archive silently regressing to empty effects) is the exact bug this task exists to fix. |
| Backfill existing documents in place instead of re-ingest | Documents are opaque JSON blobs (PRD §6); the derivation is not invertible from the stored content. Re-ingest is the only correct path. |

---

## 10. Test strategy

### 10.1 `skill/formula` unit tests

Table-driven over `(expr, level) → expected`. Mandatory cases, each mapped to
its requirement:

- **Real division (FR-3.4):** `u(x/2)` at x=1 → 1; `d(x/4)` at x=1 → 0;
  `d(x/4)` at x=4 → 1.
- **`u`/`d` are trunc-based, not ceil/floor (§1.3):** `u(x/5)` at x=5 → 1 (not
  2 — the `0.999999` fudge must not push an exact integer up); and a direct
  negative-argument case asserting `d(-x/2)` at x=1 → 0 and
  `u(-x/2)` at x=3 → 0, pinning the client's semantics against a future
  "cleanup" to `math.Floor`/`math.Ceil`.
- **Precedence (§1.4):** `x/2*3` at x=20 → 3, asserting `*` binds tighter than
  `/`. This value is *different* from left-to-right evaluation, so the test
  fails loudly if someone reorders the tiers.
- **Unary minus (FR-3.5):** `-2` → -2; `-10-1*x` at x=3 → -13.
- **Decimal (FR-3.6):** `0.5*x` at x=1 → 0, at x=3 → 1; `5+0.5*x` at x=5 → 7.
- **Leading whitespace (FR-3.7):** `" 375+5*x"` at x=1 → 380.
- **Max complexity (FR-3.9):** `-1-1*u(x/10)` at x=1 → -2, at x=20 → -3;
  `150+50*u(x/10)` at x=1 → 200, at x=20 → 250.
- **Nesting superset (FR-3.8, §4.4):** `u(d(x/2))` parses and evaluates.
- **Bounds (NFR-5):** over-length input, unknown identifier `y`, unknown
  function `f(x)`, arity-2 `u(x,1)`, unbalanced paren — each a parse error,
  none a panic.

### 10.2 Reader tests

- **FR-4 `x`-namespace regression:** skill `1101004`, where `common/x = "-2"`
  while sibling keys evaluate over level. Assert the effect's `X` is -2 at
  every level *and* that the sibling keys vary with level — i.e. the value of
  `common/x` was never bound as the variable.
- **Expansion:** a `common` node with `maxLevel` 20 produces exactly 20
  effects, in ascending level order, evaluated at x = 1..20.
- **`maxLevel` node type (FR-2.3):** an `int` node and a `string` node both
  parse (the three string-typed skills are 13100004, 1320005, 32120001).
- **Int-typed non-`maxLevel` leaves (FR-2.6):** `3111003/common/time` = 0 and
  `33111005/common/x` = 1.
- **Pass-through (FR-2.4, FR-2.5):** `lt`/`rb` vectors survive unevaluated;
  `action = "slashStorm2"` is never evaluated and never appears as an effect
  field.
- **FR-1.2 dual-node pin:** skills 2211002 and 2211006 derive from `common`,
  yield `maxLevel` 20, and have `MagicAttack` 0 and `Mastery` 0 — the accepted
  regression, asserted so it stays a decision.
- **FR-1.3:** a skill with neither subtree yields 0 effects and does not panic.
- **FR-7.5 pin (§2.2):** every `IntegerNode.Value` on a synthesized node
  round-trips through `strconv.ParseInt` without error.
- **FR-7 behaviour:** a deliberately malformed expression produces exactly one
  ERROR with the required fields, does not abort the job image, and increments
  the failure counter; a missing `maxLevel` scopes the failure to that skill.
- **Range check (§4.1):** an expression evaluating outside the target field's
  range is a failure, not a silent wrap.
- **`level`-path non-regression:** a `level`-only fixture produces existing
  attributes with byte-identical values to today (see §12 amendment 1).

### 10.3 Differential test against the archive

The strongest available check on §1.4's "they agree on this archive" premise:
evaluate all 2 374 `common` string values at every level 1..`maxLevel` under
(a) this design's client-faithful evaluator and (b) a naive float-AST
evaluator with standard precedence and a single final truncation, and assert
the two agree everywhere.

If they agree, the premise is proven rather than argued. If they disagree
anywhere, the disagreement is a real finding that must be reconciled against
the client before the task lands.

This requires the archive, so it runs as an offline verification step against
the MinIO object (md5 `2d77583108eb928b65f2904196a894ef`), not as a committed
unit test — committing a 163 MB fixture is not an option. The committed
artifact is the extracted `(skillId, key, expression)` corpus and the expected
values, so the assertion is reproducible in CI without the archive.

### 10.4 Gates

Per `CLAUDE.md`: `go test -race ./...`, `go vet ./...`, `go build ./...` in
`services/atlas-data`; `tools/lint.sh --check`, `tools/goroutine-guard.sh`,
`tools/redis-key-guard.sh` from the repo root; `npm run build` in atlas-ui;
`docker buildx bake atlas-data` only if `go.mod` changes (it should not — no
new dependency). Code review before the PR.

---

## 11. Risks

| Risk | Mitigation |
|---|---|
| §1.4's agreement premise is wrong somewhere in the archive | §10.3's differential test converts the premise into a checked fact before landing. |
| A future maintainer "fixes" the non-standard precedence or replaces `trunc(v+0.999999)` with `math.Ceil` | The package doc comment carries §1's IDB citation; §10.1's `x/2*3` and negative-argument cases fail loudly on either change. |
| The 35 new attributes bloat every skill document on every tenant | Additive zero-valued fields; measured, not guessed, in the plan's verification step. If the growth is material, `omitempty` is a reversible follow-up decision — but it is not taken pre-emptively (§5.5). |
| The 2211002/2211006 regression surfaces later as a player-facing bug report | Pinned by test (§10.2) and recorded in the PRD; the PR description should name both skills explicitly. |
| Unknown-key WARN spam on a future archive | It is one WARN per (skill, key) per ingest, not per level. Acceptable for a signal that only fires when the archive changes shape. |
| `RegisterSkill` signature change ripples into tests | Only one non-test caller (`skill/processor.go:56`); `reader_test.go` is large but the change is mechanical. |

---

## 12. Proposed amendments to the PRD

1. **"Byte-identical" is not achievable and should not be the criterion.**
   PRD §10 asks that a `level`-only skill serialize byte-identically to today.
   FR-6.1 requires the 35 new keys to be populated on *both* paths, so a v92
   `level` node carrying e.g. `mastery` will now serve a non-zero `mastery` —
   an improvement, but not byte-identity. Every effect also gains 35
   zero-valued attributes.
   Proposed replacement: *"For a `level`-only skill, no **existing**
   `effect.RestModel` attribute changes name, type, or value. New attributes
   appear with the value derived from the `level` node (zero where the key is
   absent)."*

2. **FR-3.3 restated** as `u(v) = trunc(v + 0.999999)`, `d(v) = trunc(v)`,
   per §1.3.

3. **FR-3.2 restated** as the client's `*` → `/` → `-` → `+` precedence with
   per-operation truncation toward zero, per §1.4 — replacing "standard
   precedence".

Additionally, PRD §7's "No other Go service changes" stands, but the service
impact should record the atlas-ui change from §7 of this document.

---

## 13. Work breakdown (input to `/plan-task`)

1. `skill/formula` package: tokenizer, parser, evaluator, bounds. Unit tests
   (§10.1) first — the grammar is fully specified, so this is TDD-shaped.
2. `effect` model: 33 new fields + 2 reuses across `model.go` and `rest.go`
   (§5.2).
3. `getEffect`: one read line per new key (§5.2). This is the change that
   populates both paths.
4. Common key table + `expandCommon` + `produceSkill` precedence (§4).
5. Stats/error plumbing through `Read` → `Register` → `RegisterSkill` →
   `workers.Skill.Run` (§6).
6. Reader tests (§10.2).
7. atlas-ui `SkillEffect` + `FIELD_LABELS` (§7).
8. OQ-5 census across all region/version archives; report the table (§8).
9. Differential verification against the real v95.1 archive (§10.3), then the
   end-to-end checks: skill 1001003 values, and the 0-of-954 `maxLevel: 0`
   census.
10. Full gate sweep (§10.4) + code review.
