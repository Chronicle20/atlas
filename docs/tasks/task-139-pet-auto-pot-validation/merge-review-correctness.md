# Merge review: main -> task-139-pet-auto-pot-validation (55cf0a30f)

Reviewed read-only. Verified every claimed resolution independently via
`git diff <parent> 55cf0a30f -- <path>` against both `c97ccf6f2` (ours) and
`8a9f70301` (theirs), `git show <rev>:<path>` for point lookups, and a
throwaway key-level JSON differ (written outside the repo, under a scratch
`/tmp`-style job directory, not committed) for the 8 seed templates. Also ran
`go build ./...`, `go vet ./...`, `tools/template-duplicate-binding-guard.sh`,
`tools/template-opcode-order-guard.sh`, and a conflict-marker sweep.

## Strengths

- **resolve.go / resolve_test.go** — confirmed both `ResolveCode16` (ours,
  line 103 in the merged file) and `ResolveValue` (theirs, line 148) are
  present with their full bodies, the `math` import theirs needed is present,
  and both `TestResolveCode16` and `TestResolveValueValid`/`TestResolveValueMisses`
  test functions survive in `resolve_test.go`. No duplication, no loss.
- **consumable Processor/mock** — `RequestItemConsume` carries the merged
  `quantity int16` signature from main with the `quantity < 1 -> 1` guard;
  `RequestItemConsumeWithPet` (ours) and `RequestSkillBookUse` (theirs) are
  both present as interface methods, impl methods, and mock methods/fields.
  `var _ consumable.Processor = (*ProcessorMock)(nil)` still compiles, confirmed
  by `go build ./...` in atlas-channel.
- **character_skill_use.go** — `enableActions` was correctly left extracted
  into `enable_actions.go` (ours' move); `grep -rn '^func enableActions'`
  across the handler package returns exactly one hit, in `enable_actions.go`.
  `battleshipCastBlocked` (theirs) is present once, in `character_skill_use.go`,
  and is exercised in the same call sites theirs added. No duplicate
  definition, no dangling reference.
- **pet_item_use.go** — ours' full auto-pot validation block (`skillGate`
  check, `PetId` overflow guard, parallel fetch via `model.NewGroup`, `reject`
  closure, etc.) is intact; the only line-level change vs ours is the
  `RequestItemConsume` call gaining `quantity=1` literally, matching main's
  new signature.
- **gms_95 opCode `0x1C` special case is defensible, not a dropped
  main-side change.** Base had `writer: PicResult, services: [login]`. Ours
  had already reassigned it to `writer: CharacterInventoryChange,
  services: [channel]` (task-139's own earlier work) with
  `options.petSkill.autoSpeaking`. Main independently added
  `fname: CWvsContext::OnInventoryOperation` but left `writer`/`services`
  untouched (a documentation-only fname pass, not a semantic reassignment).
  Cross-checked against `docs/packets/audits/status.json`: the
  `INVENTORY_OPERATION` op (fnames: `["CWvsContext::OnInventoryOperation"]`,
  packet `inventory/clientbound/InventoryAdd`) is `verified` for `gms_v95` at
  `opcode: 28` (== `0x1C`). And `libs/atlas-packet/inventory/clientbound/change.go`
  defines `InventoryChangeWriter = "CharacterInventoryChange"` with an
  attached `// packet-audit:fname CWvsContext::OnInventoryOperation` comment on
  its `QuantityUpdate` type. So ours' writer (`CharacterInventoryChange`) and
  theirs' fname (`CWvsContext::OnInventoryOperation`) name **the same verified
  packet** — the resolver's choice (ours' writer + theirs' fname) is
  corroborated by the coverage matrix, not an invented combination. Main did
  not make a conflicting semantic change here that got dropped; it only
  labelled the opcode with the identity ours had already migrated it to.
- **fname / options merge across all 8 templates** — the JSON differ confirms,
  per template, that every `fname` main added on a *stable* (opCode, handler)
  key round-trips into the merged file unchanged, and every `options` block
  ours added round-trips unchanged. The only apparent "MISSING key" hits from
  the differ (e.g. gms_61 `0x47 CharacterItemUseHandle`, `0x4B PetFoodHandle`,
  `0x5A CharacterUseSkillHandle`) were false positives from my key = (opCode,
  name): main independently renumbered those opcodes in an unrelated fname
  pass (e.g. `CharacterItemUseHandle` moved `0x47 -> 0x43`, `PetFoodHandle`
  moved `0x4B -> 0x47`), and the merge correctly took theirs' renumbering in
  full — verified by hand for gms_61's `0x43/0x47/0x4B/0x53` cluster.
- **Duplicate leading-zero-padding MiniRoom entries were cleaned up, not
  dropped.** gms_83/87/95/jms_185 all had, on ours, two writer entries for the
  same opcode — one unpadded (`"0xA5"`, carrying `options: {}`) and one
  zero-padded (`"0x0A5"`, no options) — the exact task-194 duplicate-binding
  bug pattern called out in project memory. Main (post-fix) had only the
  single correct entry with `fname` added. The merge kept main's single
  clean entry with `fname`, which is the *correct* resolution — it removes a
  pre-existing bug from ours rather than "dropping" a legitimate entry.
  Confirmed by `tools/template-duplicate-binding-guard.sh` passing clean (22
  arrays, no duplicate bindings) on the current tree.
- **gms_48 / gms_92 (non-conflicting) undamaged.** gms_48's `PetItemUseHandle`
  (opCode `0x75`, `skillGate: equipAbility`) was task-139's own pre-merge work
  and is byte-identical across ours -> merge -> tip. gms_92's
  `PetItemUseHandle` (opCode `0xC8`) came only from main and had no
  `skillGate` at merge time; `f7fc16631` (post-merge, on-branch) added it
  immediately after, which matches the task's own account and is not a merge
  defect. `1bc731215`'s `options.petSkill.autoSpeaking` additions to gms_48/92
  are additive, well-scoped, and don't touch any conflicted key.
- **Build/vet/guards all clean.** `go build ./...` and `go vet ./...` pass in
  `services/atlas-channel/atlas.com/channel` and `libs/atlas-packet`.
  `tools/template-duplicate-binding-guard.sh` and
  `tools/template-opcode-order-guard.sh` both report OK.
  No `<<<<<<<`/`>>>>>>>` conflict markers in any `.go`/`.json` file.
- **STATUS.md/status.json regeneration (`ac832acaf`)** happened after the
  merge commit and is a real regeneration (125-line diff to status.json), not
  a stale carry-forward — consistent with the project's documented "toolSha
  reads git HEAD; regen AFTER merge" requirement.

## Issues

None found at Critical or Important severity.

### Minor

- **`pet_item_use.go:160` passes a hardcoded `1` for the new `quantity`
  parameter.** Checked: this is consistent with every other
  `RequestItemConsume` call site in the service — `character_cash_item_use.go:66`,
  `character_item_use.go:23/32/50`, `pet_food.go:23`, and
  `shopscanner/processor.go:79` all pass a literal `1` too. None of main's
  other call sites derive quantity dynamically from the packet or an
  `itemConNo` field, so the auto-pot handler is not an outlier — it's
  following the established (if arguably incomplete) pattern across the
  whole file. Not a merge-introduced regression; if this is wrong, it is a
  pre-existing gap in main, out of scope for this merge's correctness review.
- **NoteOperation / CashShopOperation `options` blocks differ from ours'
  pre-merge values in gms_61/72/79/83/84/87/95/jms_185**, expanding e.g.
  `errors.NO_NOTE_ITEM` and a much larger `CashShopOperation.operations` map.
  These are unrelated to the pet/skill conflict set (git resolved them
  non-conflictingly, since ours never touched those keys) and merged cleanly
  to main's newer/larger tables in every case checked — flagged here only so
  a human confirms this was an intentional main-side table expansion and not
  something task-139 needs to reconcile with its own (older, narrower)
  understanding of those wire tables. No evidence of loss either way; ours'
  older, narrower table was simply superseded by main's newer one, which is
  the expected non-conflicting-merge behavior.

## Assessment

**Ready to merge? Yes.**

Every one of the five claimed resolutions was independently verified against
both parents and, where non-obvious (the gms_95 `0x1C` writer/fname
combination in particular), cross-checked against the packet coverage matrix
and the codec's own `packet-audit:fname` annotation rather than taken on
trust. No dropped, duplicated, or mis-combined change was found in any of the
16 conflicted files. Build, vet, and the template guards are clean, and no
leftover conflict markers exist. The only points worth a human's attention
are noted above as Minor and are pre-existing/main-side, not merge defects.
