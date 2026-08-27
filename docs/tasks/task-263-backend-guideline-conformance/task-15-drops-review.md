# Task 15 review — atlas-drops (`party`), commit a614c62c9

## Scope

`git show --stat a614c62c9` confirms exactly two files touched:
`services/atlas-drops/atlas.com/drops/party/rest.go` (+26) and
`services/atlas-drops/atlas.com/drops/party/rest_test.go` (+49). No other file in the commit.
Matches the brief (`task-15-brief-atlas-drops.md`): implement `Transform`/`TransformMember` as
inverses of the two existing `Extract*` functions in that package.

## 1. Field inventory — `Extract*`/`Transform*` correspondence

`services/atlas-drops/atlas.com/drops/party/model.go` (unexported fields, read directly):

- `Model`: `id uint32`, `members []MemberModel`
- `MemberModel`: `id uint32`, `field field.Model`, `online bool`

`services/atlas-drops/atlas.com/drops/party/rest.go`:

- `RestModel`: `Id uint32`, `Members []MemberRestModel`
- `MemberRestModel`: `Id uint32`, `WorldId world.Id`, `ChannelId channel.Id`, `MapId _map.Id`,
  `Instance uuid.UUID`, `Online bool`

`libs/atlas-constants/field/model.go` — `field.Model` has exactly four unexported fields:
`worldId`, `channelId`, `mapId`, `instance`, each with a public accessor (`WorldId()`,
`ChannelId()`, `MapId()`, `Instance()`). No fifth field, no hardcoded-value accessor on this type.

Verified line-by-line, not by equivalence-with-sibling assumption (per the task-263 caution — the
sibling `party` packages are near-identical in shape but were not diffed against, and were not
needed to be, since this review reads `atlas-drops`'s own `model.go`/`rest.go` directly):

| Extract field read | Transform field written | Match |
|---|---|---|
| `rm.Id` → `Model.id` | `m.id` → `RestModel.Id` | yes, `rest.go:116` / `:139` |
| `rm.Members` (via `ExtractMember`) → `Model.members` | `m.members` (via `TransformMember`) → `RestModel.Members` | yes, `rest.go:107-114` / `:130-137` |
| `rm.Id` → `MemberModel.id` | `m.id` → `MemberRestModel.Id` | yes, `rest.go:123` / `:146` |
| `field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build()` → `MemberModel.field` | `m.field.WorldId()`, `.ChannelId()`, `.MapId()`, `.Instance()` → `MemberRestModel.{WorldId,ChannelId,MapId,Instance}` | yes, `rest.go:124` / `:147-150` |
| `rm.Online` → `MemberModel.online` | `m.online` → `MemberRestModel.Online` | yes, `rest.go:125` / `:151` |

Every field of every one of the four types is accounted for on both sides. No field is dropped,
and no field appears on only one side of the round trip. **PASS.**

## 2. Hardcoded-value accessors

Grepped `model.go` and `builder.go` for the package: no accessor returns a hardcoded/constant
value (all four `Model`/`MemberModel` accessors and all four `field.Model` accessors return a
struct field verbatim). Nothing to exclude from `Transform`/`Extract`. **PASS.**

## 3. Live mutation (performed by this review, independent of the implementer's own mutation proof)

Mutated `party/rest.go:147` from `WorldId: m.field.WorldId(),` to `WorldId: world.Id(0),` via a
uniquely-anchored `python3` string replace (single occurrence in the file, confirmed by
`assert s.count(old) == 1` before writing).

```
$ go test ./party/... -run TestTransformRoundTrip -v
--- FAIL: TestTransformRoundTrip (0.00s)
    --- FAIL: TestTransformRoundTrip/Transform (0.00s)
        rest_test.go:163: Error: Should be true   # reflect.DeepEqual(got, m) fails
    --- FAIL: TestTransformRoundTrip/TransformMember (0.00s)
        rest_test.go:183: Error: Should be true   # reflect.DeepEqual(got, mm) fails
FAIL
```

Both subtests fail on the mutated field, confirming the round-trip assertion is load-bearing
(not tautological). Reverted with the inverse string replace; confirmed:

```
$ git diff --exit-code -- services/atlas-drops/atlas.com/drops/party/rest.go
$ echo $?
0
```

File is byte-identical post-revert. Re-ran tests green after revert. **PASS.**

## 4. Fixture non-defaultness

`rest_test.go:140-185`, `TestTransformRoundTrip`:

- `Transform` subtest: `Id=9`, member `Id=101`, `field` world=5/channel=2/map=300000000/instance
  `11111111-...`, `Online=true`.
- `TransformMember` subtest: `Id=202`, `field` world=6/channel=4/map=400000000/instance
  `22222222-...`, `Online=true`.

All numeric fields are non-zero and mutually distinct across the two subtests; both UUIDs are
non-nil and distinct from each other and from the pre-existing `TestExtract` fixtures
(`00000000-...0001`, `uuid.Nil`). `Online=true` is non-default (zero value of `bool` is `false`).
No field in either fixture holds a zero/default value that could mask a dropped mapping.
**PASS.**

## 5. Blast radius — docs, exemptions, sibling packages

- `git show --name-only a614c62c9 | grep -i '^docs/'` → no output. No `docs/` file touched.
- Task 15 explicitly does not require (and this commit does not add) any `handwork-notes.md`
  exemption entry — `atlas-drops/party` has a real `RestModel`, so no B1/NO-RESTMODEL exemption
  applies here.
- `git show --name-only a614c62c9` lists only the two `services/atlas-drops/...` files; no
  `atlas-guilds`, `atlas-party-quests`, or `atlas-channel` sibling `party` package was touched.
  **PASS.**

## 6. No orphan `Extract*`

```
$ grep -rn '^func Extract' services/atlas-drops/atlas.com/drops/party/
rest.go:106:func Extract(rm RestModel) (Model, error) {
rest.go:121:func ExtractMember(rm MemberRestModel) (MemberModel, error) {
```

Both now have inverses: `Transform` (`rest.go:129`) and `TransformMember` (`rest.go:144`). No
`Extract*` anywhere else in the package tree (only `rest.go` defines any `Extract`/`Transform`
function per the earlier full-directory grep). **PASS.**

## Build/format/test gate (module-local, informational — not a substitute for `tools/verify.sh`)

```
$ go build ./... && go vet ./... && go test ./...
ok  	atlas-drops/party ...  (and all other atlas-drops packages, all pass)
$ tools/lint.sh --check --fmt --go services/atlas-drops/atlas.com/drops
lint.sh: OK
```

## Verdict

No defects found. Every `Extract*`/`Transform*` pair is a verified exact field-for-field inverse,
the round-trip test is demonstrably non-tautological (live mutation reproduced by this review,
independent of the implementer's own proof), the fixtures are non-default, and the commit's
blast radius is exactly the two files it claims — no docs, no exemption note, no sibling package.
