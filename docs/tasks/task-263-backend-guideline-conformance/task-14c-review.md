# Task 14 batch C review — `services/atlas-npc-conversations/atlas.com/npc/conversation`

Range reviewed: `13dabad91..f1ef34e`
Brief: `.superpowers/sdd/plan/task-14-brief-c.md`
Report: `.superpowers/sdd/plan/task-14-report-c.md`

## Scope confirmed

The diff touches exactly the three files the brief scoped: `rest.go` (+72/-37),
new `rest_transform_test.go` (+504), and `handwork-notes.md` (+4). Two commits
(`d7149238`, `f1ef34e`), matching the brief's Step 7 commit plan. No files outside
this package were touched. Scope matches the brief.

## 1. De-duplication — PASS

Confirmed via `git diff 13dabad91..f1ef34e -- .../rest.go`: all four inline mapping
blocks are gone and replaced with calls to the new named functions.

- `TransformDialogue` (`rest.go`): the inline `RestChoiceModel{...}` literal inside the
  choices loop is replaced by `TransformChoice(choice)`.
- `TransformGenericAction` (`rest.go`): the inline `RestOperationModel{...}` literal is
  replaced by `TransformOperation(operation)`; the inline condition/outcome-building
  block (which built `RestConditionModel` and `RestOutcomeModel` by hand) is replaced by
  `TransformOutcome(outcome)`.
- `TransformOptionSet` (`rest.go`): the inline `RestOptionModel{...}` literal is replaced
  by `TransformOption(option)`.

No inline copy survives next to any of the four new named functions — verified by reading
the full unified diff hunk by hunk, not just grepping for the new function names.

## 2. Each `Transform<X>` is a true inverse of its `Extract<X>` — PASS

Read `Extract`/`Transform` bodies side by side for all four pairs:

- `ExtractChoice`/`TransformChoice`: `Text`, `NextState`, `Context` — symmetric.
  `ExtractChoice` guards `SetContext` on `r.Context != nil`; `TransformChoice` always
  emits `m.Context()`, which is `nil` when unset (builder-backed accessor), so the nil
  case round-trips correctly (confirmed via the `Option`/`Choice` builders reviewed
  below and by the passing round-trip test, which builds `Choice` with a non-nil
  `Context`).
- `ExtractOperation`/`TransformOperation`: `OperationType`↔`Type()`, `Params`↔`Params()`
  — symmetric, no field dropped either direction.
- `ExtractOutcome`/`TransformOutcome`: both map `Conditions` (five sub-fields: `Type`,
  `Operator`, `Value`, `ReferenceId`, `Step`, `IncludeEquipped`) and `NextState`.
  `ExtractOutcome` only calls `SetNextState` when `r.NextState != ""`; `TransformOutcome`
  always emits `m.NextState()`. This is not a defaulting asymmetry (unlike
  `EndChat`/`Speaker`) because an empty string is itself a legitimate identity for a
  zero-value `NextState` — mutation-tested below and it fails correctly when the field is
  dropped, confirming it is genuinely load-bearing rather than a papered-over default.
- `ExtractOption`/`TransformOption`: `Id`, `Name`, `Materials`, `Quantities`, `Meso` —
  symmetric. `ExtractOption` guards `SetMaterials`/`SetQuantities` on `len(...) > 0`;
  `TransformOption` always emits `m.Materials()`/`m.Quantities()`, which are `nil` when
  unset, so the guarded builder calls are no-ops on the reverse leg and the round trip
  holds.

`RestConditionModel` is not a target: it has no standalone `Extract`/`Transform`; both
directions build/read `RestConditionModel` fields inline inside
`ExtractOutcome`/`TransformOutcome`, matching the brief's explicit scope note.

## 3. Test non-tautology — PASS (independently verified by live mutation)

Ran `go test ./conversation/... -run TestTransformRoundTrip -v` from the module root:
all 18 subtests (`State`, `Dialogue`, `Choice`, `GenericAction`, `Operation`, `Outcome`,
`CraftAction`, `TransportAction`, `GachaponAction`, `RPSAction`, `PartyQuestAction`,
`PartyQuestBonusAction`, `ListSelection`, `AskNumber`, `AskStyle`, `AskSlideMenu`,
`OptionSet`, `Option`) pass green, covering all 18 `Extract`/`Transform` pairs, not just
the four new ones.

I did not rely on the report's transcript. I performed two independent live mutations
myself, on two of the four new pairs:

**Mutation A — `TransformOption`, `Meso` field.** Changed `Meso: m.Meso()` to
`Meso: 0` in `rest.go`, re-ran `-run TestTransformRoundTrip`:

```
got  = conversation.OptionModel{id:0x2, name:"Shield", ..., meso:0x0}
want = conversation.OptionModel{id:0x2, name:"Shield", ..., meso:0xc8}
--- FAIL: TestTransformRoundTrip (0.00s)
    --- FAIL: TestTransformRoundTrip/OptionSet (0.00s)
    --- FAIL: TestTransformRoundTrip/Option (0.00s)
```

Field-level diff, both `Option` and `OptionSet` (which composes `Option`) fail. Reverted
via `cp` from a pre-mutation backup; `diff` against the backup post-revert was empty.

**Mutation B — `TransformOutcome`, `NextState` field.** Changed `NextState: m.NextState()`
to `NextState: ""`, re-ran the full suite:

```
--- FAIL: TestTransformRoundTrip (0.00s)
    --- FAIL: TestTransformRoundTrip/GenericAction (0.00s)
    --- FAIL: TestTransformRoundTrip/Outcome (0.00s)
```

`Outcome` and its composing parent `GenericAction` both fail. Reverted; `diff` against
the pre-mutation backup was empty after revert, and `git status --porcelain
services/atlas-npc-conversations` showed no residue.

Both mutations produced field-level failures, not suite-wide breakage or false passes —
the tests are not tautological.

## 4. Known asymmetries handled — PASS

- `Dialogue`/`State` subtests build fixtures with `SetSpeaker("CHARACTER")` (non-default;
  `normalizeSpeaker` default is `"NPC"`) and `SetEndChat(false)` (non-default; `EndChat`
  nil-defaults to `true`). This is a genuine identity test, not a default-value coincidence.
- Pointer-returning `Extract*` (`ExtractDialogue`, `ExtractGenericAction`,
  `ExtractCraftAction`, `ExtractTransportAction`, `ExtractGachaponAction`,
  `ExtractRPSAction`, `ExtractPartyQuestAction`, `ExtractPartyQuestBonusAction`,
  `ExtractListSelection`, `ExtractAskNumber`, `ExtractAskStyle`, `ExtractAskSlideMenu`)
  are all asserted non-nil (`if got == nil { t.Fatalf(...) }`) before dereference in
  their respective subtests — checked each occurrence in `rest_transform_test.go`.

## 5. Scope discipline (design §8.2) — PASS

`git diff 13dabad91..f1ef34e -- .../model.go` is empty — `model.go` (2430 lines,
builder-backed types) received no edits. No `TransformCondition`/`ExtractCondition`
function exists anywhere in the diff or the current file (`grep -n
"TransformCondition" rest.go model.go` returns nothing). `Extract*`/`Transform*` counts
in the current file are 18/18, matching the brief's target exactly — no overreach.

## 6. Exemption note — PASS

`handwork-notes.md` gained a `## Batch C (Task 14)` heading with one entry in the same
form as the batch A/B entries (wire types listed, `Extract*` locations with line numbers,
provided `Transform` names split new/pre-existing, out-of-scope items named). All paths
in the entry are repo-relative (`services/...`, `rest.go:<line>`), no absolute or home
path. Cross-checked all 18 cited `rest.go:<line>` references against the actual file with
`awk 'NR==...'` — every line number matches its named function exactly.

## Build/lint gate

Ran independently (not just trusted the report):

```
tools/lint.sh --check --fmt --go services/atlas-npc-conversations/atlas.com/npc
→ lint.sh: OK

go build ./... && go vet ./...  (from module root)
→ BUILD/VET OK
```

## Not evaluable

None. All checklist items in the review brief were directly evaluable within the diff's
scope.

## Verdict rationale

No blocking findings. De-duplication is real, all four new `Transform*` are exact
inverses of their `Extract*` pairs, the round-trip test covers all 18 pairs and is
demonstrably non-tautological (verified via two independent live mutations, not just the
report's transcript), known asymmetries are handled with non-default fixtures, scope
discipline held (no `model.go` edits, no `RestConditionModel` Transform), and the
exemption note matches the established form with accurate repo-relative line references.
