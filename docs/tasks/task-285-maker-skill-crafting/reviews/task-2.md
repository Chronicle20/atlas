# Review: Task 2 — `atlas-data` `itemmake` RestModel and registry

Range: `6361eb1db..cda7f3f1e` (1 commit: `cda7f3f1e feat(data): add itemmake RestModel and document registry`)

## Scope

Diff touches exactly three new files under `services/atlas-data/atlas.com/data/itemmake/`:
`rest.go` (60 lines), `registry.go` (18 lines), `rest_test.go` (135 lines). No other files
changed. This matches the brief's declared scope exactly (model + registry only; no reader,
no processor, no storage, no REST resource, no migration).

## Requirement-by-requirement

1. **`RestModel` field set** (`rest.go:7-20`) — all 13 fields present with the exact names and
   types the brief mandates: `Id`, `Group`, `ReqLevel`, `ReqSkillLevel`, `ItemNum`, `Tuc`,
   `Meso`, `Catalyst`, `ReqItem`, `ReqEquip` (all `uint32`), `Recipe []MaterialRestModel`,
   `RandomReward []RewardRestModel`, `ReqQuest []QuestReqRestModel`. PASS.
2. **`MaterialRestModel{ItemId uint32; Count uint32}`** — `rest.go:23-26`. PASS.
3. **`RewardRestModel{ItemId uint32; ItemNum uint32; Prob uint32}`** — `rest.go:30-34`. PASS.
4. **`QuestReqRestModel{QuestId uint32; State uint32}`** — `rest.go:38-41`. PASS. Correction
   C-5 (`ReqQuest` omitted by the PRD but present in the archive) is honored — the field is
   modeled, not TODO'd or dropped.
5. **`Group` field present** (correction C-6) — `rest.go:8`, `json:"group"`. PASS. Nothing in
   this task exercises the group-0 leftover→crystal lookup (that's Task 3/21's job), but the
   field exists for it to be persisted, which is this task's whole job.
6. **`GetName()` returns `"itemMakes"`** — `rest.go:43-45`. PASS, matches
   `TestRestModelGetName`.
7. **`GetID`/`SetID` idiom copied from `commodity/rest.go`** — verified byte-for-byte against
   `services/atlas-data/atlas.com/data/commodity/rest.go:18-30`: identical `strconv.Itoa`/
   `strconv.Atoi` idiom, identical receiver types (value receiver on `GetName`/`GetID`,
   pointer receiver on `SetID`). PASS.
8. **`GetModelRegistry() *document.Registry[string, RestModel]`, `sync.Once`-guarded** —
   `registry.go`. Verified byte-for-byte against `commodity/registry.go:1-18`: same import
   path `atlas-data/document`, same var names `mmReg`/`mmOnce`, same constructor call
   `document.NewRegistry[string, RestModel]()`. PASS.
9. **No new table / no migration (C-4)** — confirmed no changes to `document/entity.go`, no
   new `Migration`/`AutoMigrate` call, no GORM struct tags anywhere in the diff. The three new
   files are the only files touched. PASS.
10. **`libs/atlas-constants` check** — `commodity/rest.go:9` (the pattern this task explicitly
    copies) also uses plain `uint32` for `ItemId`, not the `item.Id` alias from
    `libs/atlas-constants/item/constants.go:5`. `itemmake`'s use of plain `uint32` for
    `ItemId`/`QuestId`/etc. is therefore consistent with the established `RestModel` convention
    in this same domain family, and matches the brief's literal type list. Not a defect.
11. **No mocks needed** — this task introduces no new interface; `document.Registry` is
    pre-existing and unchanged. N/A.
12. **Line endings** — all three files are newly created (no existing file was edited), so
    CRLF-preservation risk does not apply. The diff hunks show plain LF throughout.

## Test quality

- `TestRestModelGetName` — trivial but correct, matches spec.
- `TestRestModelIdRoundTrip` — table-driven over 3 cases including the group-0 crystal id and
  the zero-id edge case; asserts both `GetID()` and `SetID()` round-trip. Genuine assertion,
  would fail if `GetID`/`SetID` used a different encoding (e.g., hex) or off-by-one.
- `TestRestModelJSONPreservesListOrder` — asserts each of 3 `Recipe` and 3 `RandomReward`
  entries individually by index with distinct values per index (not just length checks), so a
  sort-order regression or an accidental slice-reversal would be caught. Also checks the
  single `ReqQuest` entry's fields. This is a real regression test, not a restatement.
- `TestRestModelAbsentListsAreEmptyNotNilOnRoundTrip` — unmarshal JSON with all keys but the
  three list keys absent; asserts scalars are 0 and slices have `len() == 0`. Pins the "0/empty
  when absent" convention (FR-1.5). Note: it does not explicitly assert `out.Recipe == nil`
  vs `[]MaterialRestModel{}` — but the brief's own spec only asks for `len() == 0`, so this
  matches the brief precisely; not a gap against this task's contract.
- `TestGetModelRegistryIsSingleton` — pointer-identity check across two calls, correctly
  exercises the `sync.Once` semantics.

All 5 named tests are present, each asserting on the actual runtime behavior of the code, not
tautologies. This satisfies the "does a test fail without the change" concern — every one of
these would fail against an empty package (undefined symbols) and would also fail against a
plausible incorrect implementation (e.g., a struct-tag typo, unsorted encode, or a shared
non-once registry).

## Consistency with implementer report

Implementer's report claims 8/8 tests passing (5 top-level + 3 subtests), `gofmt`/`go vet`
clean, and lists exactly the 3 files committed. This matches the diff exactly — no drift
between report and diff.

## Findings

No blocking findings. No non-blocking findings of substance — the field-type note above (#10)
was investigated and confirmed to be consistent with, not a deviation from, existing
convention.

## Not evaluable

- The `document.Registry`/`document.NewRegistry` generic implementation itself (constraint
  satisfaction, tenant-scoping behavior) is unchanged in this diff and was only read for
  contract-compatibility purposes, not re-audited — out of this task's scope.
- Whether Task 3's reader will actually produce a `Group` value scoped correctly to `0` for
  the leftover→crystal lookup is Task 3/21's concern, not verifiable from this task alone.
