# Review — Task 7: Correct the missing-features backlog entry

Commit range: `9a31743..78b692e` (single docs-only commit `78b692e`)
Brief: `.superpowers/sdd/plan/task-7-brief.md`
Report: `.superpowers/sdd/plan/task-7-report.md`

## Scope

`git diff --stat 9a31743..78b692e` shows exactly one file touched:
`docs/research/missing-features/items-and-consumables.md` (+15/-1). Matches
the brief's declared file. No code files touched, consistent with a
docs-only task; no build/test expected or run.

## What changed

- Deleted the row `| Extra-expression (emote) items | \`ClassificationExpression\` → type 6 | — | S |`
  from the "Remaining one-off cash types" table (§7), whose preamble ("All
  mapped in `GetCashSlotItemType` but unimplemented (fall to warn)") was the
  false premise for this row.
- Added a subsection `#### Extra-expression (emote) items — closed by
  task-247, and not a cash-item-use gap` immediately after the table,
  verbatim to the brief's Step 2 text.

## Fact-checking the new text against the branch's actual code

1. **"not routed through the cash-item-use handler, so a `CashSlotItemType(6)`
   dispatch arm would be dead code."**
   Verified: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
   still maps `category == item.ClassificationExpression` to
   `CashSlotItemType(6)` at line 1352 (existing pre-branch code, part of
   `GetCashSlotItemType`), but there is no `case 6:` arm in the dispatch
   switch that acts on it (`grep -n "case 6:"` → no hit). The file is absent
   from `git diff --stat 4f4b32b9c..9a31743` for this path — confirms tasks
   1–6 did not touch it, matching the brief's "MUST NOT be modified" guard
   and the doc's claim.

2. **"The real gap was on the emote path — `CharacterExpressionHandleFunc`
   accepted any emote id with no range check and no ownership check — and
   task-247 closes it."**
   Verified against `services/atlas-channel/atlas.com/channel/socket/handler/character_expression.go`
   (current state, delivered by commits `a9f733017` "range-check and
   ownership-gate extra expression emotes" and `9a31743a0` within
   `4f4b32b9c..9a31743`):
   - Range check: `if emote > item.MaxEmoteId { ... Dropping ... return }`
     (line ~42).
   - Ownership check: `if itemId, ok := item.ExtraExpressionItemId(emote); ok`
     → calls `expressionItemOwnedFunc`, fails closed on error (`return` on
     `err != nil` before checking `owns`), and drops on `!owns`.
   Both claims are true of the code the branch actually delivers.

3. **Item-id range `05160000`–`05160014`.**
   Verified against `libs/atlas-constants/item/expression.go`:
   `MaxBaseEmoteId = 7`, `MaxEmoteId = 23`, and
   `ExtraExpressionItemId` computes `ClassificationExpression*10000 + emote -
   MaxBaseEmoteId - 1`. For emote 8 (lowest gated emote) this is `5160000`;
   for emote 23 (highest) it is `5160015`, which the code's own comment
   states "has no entry in v83.1 data" — i.e. the last *valid* item is
   `5160014`. The doc's stated range matches this exactly.

4. **Ownership fail-closed claim (implicit, consistent with FR-2.5 in the
   brief's global constraints)** — confirmed above: an `expressionItemOwnedFunc`
   error returns before the `!owns` branch is reached, so a broken lookup
   drops rather than passes.

No claim in the new text overstates what the branch delivers, and no stale
duplicate of the removed premise (`ClassificationExpression.*type 6`,
`Extra-expression`) survives anywhere else under
`docs/research/missing-features/`.

## Findings

None blocking. None non-blocking.

## Not evaluable

None — the entire surface (one markdown file, cross-checked against the two
Go files it makes factual claims about) was reviewable within scope.

## Verdict

APPROVED
