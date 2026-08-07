# Task 197 — Code Review Audit

Three reviewer agents audited the completed branch in parallel per the project's
[Code Review Pattern](../../../CLAUDE.md). Each agent's full findings are in its own file;
this page is the index and the record of what was done about each finding.

| Reviewer | Findings file | Verdict |
|---|---|---|
| `plan-adherence-reviewer` | [audit-plan-adherence.md](./audit-plan-adherence.md) | Full plan adherence, no issues |
| `backend-guidelines-reviewer` | [audit-backend.md](./audit-backend.md) · [audit-backend.json](./audit-backend.json) | One Important finding — fixed |
| `frontend-guidelines-reviewer` | [audit-frontend.md](./audit-frontend.md) | Three FAILs — two fixed, one pre-existing |

## Findings and dispositions

### FILE-01 — `buyWithTokens` lived in a bare-topic filename (Important) — FIXED

`shops/token.go` held a `*ProcessorImpl` method. The guideline names a bare topic
filename as a FAIL; the repo convention for splitting Processor methods out is
`processor_<group>.go` (precedents: `atlas-mts` listing/holding/wish,
`atlas-inventory` compartment).

This collided with `plan.md`, which mandates the filename `shops/token.go` in its
File Structure and in Tasks 3 and 4. The conflict was escalated; the ruling was that
the guideline governs, since nothing imports a Go file by name and the rename is
mechanical. Fixed in `e52dcb878` — pure rename of `token.go` → `processor_token.go`
and `token_test.go` → `processor_token_test.go`, zero content change.

### FE-10 — item-search query key omitted the tenant id — FIXED

`useItemSearch`'s query key was not tenant-scoped. The defect was masked in practice
by `queryClient.clear()` on tenant switch, and the same shape already exists in
`SkillSearchCombobox`. The code was extracted verbatim from `ItemSearchCombobox` in
Task 6, so this was pre-existing debt surfaced by a second call site rather than new
breakage. Escalated because fixing it deviates from Task 6's binding
"behaviour-preserving extraction" constraint; the ruling was to fix it now. Fixed in
`be55ccadc`.

### FE-14 — no query-key factory, no `as const` — FIXED

Same line, same escalation and ruling. `be55ccadc` adds an exported `itemSearchKeys`
factory following the in-repo `itemStringKeys` precedent
(`lib/hooks/api/useItemStrings.ts`), and the hook now builds its key from it.

### FE-15 — dialog uses raw `useState` rather than react-hook-form + zodResolver — NOT FIXED

Confirmed pre-existing by diff against the merge base: `NpcShopCommodityDialog` already
used raw `useState` before this branch. Converting it is out of scope — the branch's
binding constraint was payload invariance, and a form-library migration would change
the very submission path the task had to hold still.

## Deviations from `plan.md` (all reviewer-confirmed)

The plan's literal code was wrong in three places. Each deviation was escalated and ruled on
rather than applied silently.

1. **The `4310000` literal in a doc comment.** Plan Task 4's mandated code block contained the
   constant inside a comment, which broke the plan's own Global Constraint and its Task 9
   Step 9 grep gate. Reworded in `1249edbb1`; the `0x41C3F0` grounding citation is retained.
2. **`<Label htmlFor>` on the `ItemPicker` trigger.** Plan Task 8 mandated it, but per HTML-AAM
   a `<label for>` targeting a labelable `<button>` overrides the button's own text as its
   accessible name — which breaks the plan's own mandated
   `getByRole("button", { name: "Select an item…" })` assertions. The picker rows convey their
   field name through a `role="group"` + `aria-labelledby` wrapper instead (`27d2c30ed`). The
   frontend reviewer verified the group's computed accessible name is the field label and the
   trigger keeps its own text.
3. **`FIELDS` typed `key: keyof CommodityAttributes`.** That fails `tsc -b`, because
   `CommodityAttributes` carries optional `unitPrice?`/`slotMax?` which widen `form[key]` to
   `number | undefined`. Narrowed to a union of the seven used keys. Field order is unchanged.

## Gap the plan did not anticipate

`services/atlas-npc-shops/docs/domain.md` still described token-priced commodities as
unimplemented, and an invariant still claimed every buy requires sufficient mesos. Both
statements were made false by this branch. Corrected against the actual code in `52a878b13`.

## Verification at review time

`go test -race ./...`, `go vet ./...`, `go build ./...` clean in `atlas-npc`;
`tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` all clean
(6 pre-existing ESLint warnings in files this branch never touched, 0 errors);
atlas-ui `npm run build` clean and `npm run test` at 194 files / 1400 tests passing,
including the untouched `ItemSearchCombobox.test.tsx` regression harness.

`docker buildx bake` was not required — no `go.mod` or `go.sum` changed on the branch.
