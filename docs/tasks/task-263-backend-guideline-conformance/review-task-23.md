# Review: Task 23 — atlas-npc-conversations `conversation/model.go` 20-builder hand split

Commit under review: `26e1ec1e4` (`refactor(atlas-npc-conversations): move conversation builders into builder.go`)
Brief: `.superpowers/sdd/plan/task-23-brief.md`
Report: `.superpowers/sdd/plan/task-23-report.md`

## Scope confirmed

Commit touches exactly two files:

```
.../atlas.com/npc/conversation/builder.go   | 1567 ++++++++++++++++++++
.../atlas.com/npc/conversation/model.go     | 1558 -------------------
2 files changed, 1567 insertions(+), 1558 deletions(-)
```

This matches the brief's named files exactly (`model.go`, `builder.go`). `model_json.go` and the two test files named in the brief as read-only were not touched. No scope mismatch.

## 1. Verbatim-ness of all 20 declaration sets

Independently re-derived (not trusting the report's filtered diff). Extracted the full commit diff with `git diff -M 26e1ec1e4~1 26e1ec1e4 -- .../conversation`, then wrote a fresh Python filter (independent of the report's script) that:

- drops diff metadata lines (`diff `, `index `, `---`, `+++`, `@@`)
- drops blank lines, `package` lines, `import` lines, quoted-import lines, and bare `)` lines — symmetrically for both `-` and `+` sides (the known blank-line asymmetry trap from task-21b was checked for and does not apply here since blank lines are filtered from both sides equally)
- compares the **ordered** sequence of remaining `-` lines against the ordered sequence of remaining `+` lines (stronger than a multiset/Counter comparison, which cannot detect a same-content-different-order shuffle)

Result:

```
minus count (filtered): 1362
plus count (filtered): 1362
order-preserving equal: True
```

The entire ordered content of every hunk in the diff matches, line for line, between removed and added sides. No reformatting, no renaming, no reordering, no silent body edits, across all 20 sets — not sampled, the full diff.

## 2. Completeness in both directions

**model.go → builder.go, nothing left behind:**

```
grep -cP '^func New[A-Za-z0-9_]*Builder\(|^func \(b \*[A-Za-z0-9_]*Builder\)|^func Clone[A-Za-z0-9_]*\(' /tmp/base_model.go   -> 149
grep -cP  (same pattern) builder.go                                                                                          -> 149
grep -cP  (same pattern) model.go                                                                                            -> 0
```

All 149 constructor/method/Clone functions moved; none remain in `model.go`.

Brief's literal Step-4 check (`grep -c 'Builder' conversation/model.go`) returns `1`, not the expected `0`. Verified the hit:

```
model.go:444: // cannot be produced through the validated CraftActionBuilder.
```

This is a pre-existing doc-comment prose reference inside `NewCraftActionModelDirect` (a plain domain constructor, not a builder type), pointing at the sibling `CraftActionBuilder` now living in `builder.go`. Confirmed no actual declaration remains via the declaration-specific grep above (`0` hits). This is a literal deviation from the brief's exact expected grep output but not a substantive defect — noted as non-blocking.

**builder.go → no domain type/accessor dragged along (converse check):**

```
grep -nP '^func (?!\(b \*[A-Za-z0-9_]*Builder\))|^type (?!\w+Builder struct)' builder.go
```
returned only the 20 `New<Type>Builder(...)` constructor signatures (excluded by the negative lookahead only because it starts with `New`, but every match printed is in fact a `New*Builder` line, i.e., every top-level `func`/`type` in `builder.go` is either a `*Builder` receiver method, a `New*Builder` constructor, or a `type *Builder struct`). No domain type, plain accessor, or unrelated declaration is present in `builder.go`.

## 3. Count integrity

Base file (`git show b7f3e8d90:.../conversation/model.go`) `type <X>Builder struct` count: **20**. `builder.go` `type <X>Builder struct` count: **20**. Enumerated both lists — identical type names in identical order:

`StateBuilder, DialogueBuilder, ChoiceBuilder, GenericActionBuilder, OperationBuilder, ConditionBuilder, OutcomeBuilder, CraftActionBuilder, TransportActionBuilder, GachaponActionBuilder, RPSActionBuilder, PartyQuestActionBuilder, PartyQuestBonusActionBuilder, ListSelectionBuilder, AskNumberBuilder, AskStyleBuilder, AskSlideMenuBuilder, OptionSetBuilder, OptionBuilder, ConversationContextBuilder`

Not 19, not 21 — exactly 20, matching the brief's stated count.

## 4. No `Transform` functions added

```
grep -n 'func.*Transform' model.go builder.go
```
No matches in either file. PASS.

## 5. No constructor renames

All 20 constructors retain the `New<Type>Builder` name (verified in the completeness listing in §2 and the type-name enumeration in §3). No renames. Consistent with D4 / FR-15 not applying to these siblings-over-distinct-types.

## 6. Recomputed import blocks

```
model.go imports:   errors, strconv, github.com/google/uuid, .../atlas-constants/field
builder.go imports: errors,          github.com/google/uuid, .../atlas-constants/field   (strconv dropped)
```

Usage counts confirm both blocks are minimal and complete:

| symbol | model.go uses | builder.go uses |
|---|---|---|
| `errors.` | 4 | 59 |
| `strconv.` | 1 | 0 |
| `uuid.` | 5 | 4 |
| `field.` | 2 | 2 |

`strconv` is correctly retained in `model.go` (1 use) and correctly dropped from `builder.go` (0 uses). No unused or missing imports in either file.

`model_json.go` was checked for references to moved declarations: it references `CraftActionModel` (a domain type staying in `model.go`) via its `MarshalJSON`/`UnmarshalJSON` methods, not any `*Builder` type. No ripple effect on `model_json.go`'s own import block.

## 7. Build / test / vet

Re-ran independently from `services/atlas-npc-conversations/atlas.com/npc`:

```
go build ./...   -> success, no output
go vet ./...     -> success, no output
go test ./conversation/...  -> all packages ok (conversation, item, mock, npc, npc/mock, quest, quest/mock, recipe)
gofmt -l model.go builder.go -> no output (both clean)
```

## 8. Commit hygiene

`git show --stat 26e1ec1e4` confirms only `builder.go` (new) and `model.go` (modified) are in the commit — no other files rode along. Commit message is a plain relocation description. No lint/format fixes were folded into this commit (none were found to be necessary from the checks above; `golangci-lint` is not available in this environment either — confirmed via `which golangci-lint` / `golangci-lint version`, both returning "command not found" — so the `--new-from-rev` unmasking hazard genuinely could not be checked here and is correctly left to the repo-wide gate, per the task instructions not to substitute a bare `gofumpt` run).

## Not evaluable

- `tools/lint.sh` (golangci-lint-backed) could not be run in this environment — `golangci-lint` is not installed here either, matching the implementer's own finding. This is left to the repo-wide verification gate, per instructions given for this review; not treated as a blocking gap of this review.

## Verdict rationale

All 20 declaration sets verified verbatim (full ordered-diff match, not sampled), complete in both directions (149/149 functions), correct count (20/20, exact type-name match), no `Transform` additions, no renames, imports correctly recomputed, build/vet/test all green, commit touches only the two named files. The one deviation (brief's literal `grep -c Builder` expecting `0` vs actual `1`) is a pre-existing prose comment reference, not a functional defect, and is independently confirmed as harmless.
