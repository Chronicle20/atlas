# Review: Task 7 — Restore `atlas-channel` compilation at the four call sites

Range: `3c4d49bcc..60f87d269` (matches `git log -1 --format=%H 60f87d269` == HEAD)
Module: `services/atlas-channel/atlas.com/channel`

## Scope confirmed

`git diff --stat 3c4d49bcc..HEAD` shows exactly the three files the brief named:

```
kafka/consumer/asset/consumer.go   | 3 ++-
socket/writer/character_info.go    | 1 +
socket/writer/character_spawn.go   | 1 +
3 files changed, 4 insertions(+), 1 deletion(-)
```

The fourth named file, `character_data.go`, was correctly left untouched — see
below. No file outside these four was touched. No service-side model field
was added; no ring data was plumbed. This matches the task's scope
constraint exactly.

## Findings

### PASS — all four call sites addressed, zero values only

- `socket/writer/character_spawn.go:60` (now printed at the trailing arg
  line) passes `packetmodel.RingSet{}` to `charpkt.NewCharacterSpawn(...)`.
  Confirmed against the constructor signature in
  `libs/atlas-packet/character/clientbound/spawn.go:52`
  (`rings model.RingSet` as the trailing param) — type matches exactly.
- `socket/writer/character_info.go:54` (now `false,` appended before
  `.Encode(...)`) passes `false` to `charpkt.NewCharacterInfo(...)`.
  Confirmed against `libs/atlas-packet/character/clientbound/info.go:66`
  (`hasMarriageRing bool`) — matches.
- `kafka/consumer/asset/consumer.go:420` passes `packetmodel.RingSet{}` to
  `charcb.NewCharacterAppearanceUpdate(c.Id(), ava, packetmodel.RingSet{})`.
  Confirmed against
  `libs/atlas-packet/character/clientbound/appearance_update.go:23`
  (`rings model.RingSet`) — matches. New import
  `packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"` was
  added since no existing alias for that package existed in this file.
- `socket/writer/character_data.go` — verified no edit was made and none was
  needed: `CharacterData.Rings` (type `model.RingRecords`,
  `libs/atlas-packet/character/data.go:121`) is never set in the struct
  literal at `character_data.go:23-49`, so it is already at its Go zero
  value. Correct call, matches the brief's conditional instruction ("no
  edit needed unless the build complains").

### PASS — no missed call site

`grep -rn "NewCharacterSpawn(\|NewCharacterInfo(\|NewCharacterAppearanceUpdate("`
across the whole module returns exactly the three call sites above — no
other production or test call site of these three constructors exists in
`services/atlas-channel`.

### PASS — build, vet, and tests are green

```
$ go build ./...          # exit 0
$ go vet ./...             # exit 0
$ go test ./...            # all `ok` or `[no test files]`, no failures
```

Matches the implementer report's claimed test output.

### FINDING (blocking) — `consumer.go` import block is not gofmt-clean

`gofmt -l services/atlas-channel/atlas.com/channel` flags
`kafka/consumer/asset/consumer.go`. The new import was inserted in the wrong
alphabetical position within the `atlas-packet/*` group:

```
kafka/consumer/asset/consumer.go:32-37 (as committed)
	invcb "github.com/Chronicle20/atlas/libs/atlas-packet/inventory/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	messengerpkt "github.com/Chronicle20/atlas/libs/atlas-packet/messenger"
	messengercb "github.com/Chronicle20/atlas/libs/atlas-packet/messenger/clientbound"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
```

`libs/atlas-packet/messenger` sorts before `libs/atlas-packet/model`
alphabetically, but the diff placed `packetmodel` (the `.../model` import)
above the two `messenger*` imports. `gofmt -d` confirms:

```
-	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
 	messengerpkt "github.com/Chronicle20/atlas/libs/atlas-packet/messenger"
 	messengercb "github.com/Chronicle20/atlas/libs/atlas-packet/messenger/clientbound"
+	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
 	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
```

This does not break the build (Go does not require import-block ordering to
compile), and `go vet` passes. But `tools/lint.sh` (the shared lint & format
guard `tools/verify.sh` invokes, per `tools/lint.sh:1-14`) enforces
formatting TREE-WIDE via golangci-lint v2's `gofumpt`+`goimports`
formatters, which is a strict superset of `gofmt`; since plain `gofmt -l`
already flags this file, the format guard will also flag it and the
flagless `tools/verify.sh` will not exit 0 until it is fixed. The task's own
fact block lists "lint & format guard (91 module(s))" among
`applicable_guards`, and the implementer's report explicitly claims
"followed the existing import-alias convention ... placed the new import in
the existing alphabetized `atlas-packet/*` block" — which is inaccurate; the
placement is out of order. Trivial one-line fix (move the `packetmodel`
import two lines down, after `messengercb`), but it is a real, cited defect
the implementer's self-review missed, and it blocks "done means verified"
until corrected.

## Not evaluable

- Whether `tools/verify.sh`'s lint & format guard actually runs `gofmt -l`
  (or an equivalent) over this specific module was not executed as part of
  this review (out of scope for a per-unit diff review; `gofmt -l` was run
  directly instead, which is the authoritative primitive the guard would be
  built on).

## Verdict rationale

Functionally complete and correctly scoped: all four call sites handled,
none missed, none over-scoped, build/vet/test all green, parameter types
verified against the actual constructor signatures Tasks 5/6 introduced.
The single defect — an out-of-order import in `consumer.go` that `gofmt -l`
already flags, and that the repo's tree-wide format guard
(`tools/lint.sh`, invoked by `tools/verify.sh`) would also flag — is
trivial to fix (move one import line) but is a genuine gate failure, not
merely cosmetic noise: it means the flagless `tools/verify.sh` cannot
currently exit 0 for this branch, which the repo's "done means verified"
rule requires before calling this ready for PR. CHANGES_REQUIRED, with a
one-line fix.

## Fix Round 1 Re-review — Formatting (scoped)

### Open finding (from Task 7 Review)

`services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go:32-37` — the `packetmodel` import was out of alphabetical order in its group, appearing before `messengerpkt` and `messengercb` instead of after `messengercb`. `gofmt -l` flagged the file, and the tree-wide format guard would block verification.

### Fix applied

Commit `cc49a9fb8` moved the `packetmodel` import line from line 35 (before `messengerpkt`) to line 37 (after `messengercb`), placing it in correct alphabetical order within the `atlas-packet/*` import group.

### Verification

**Diff scope (verified against commit `cc49a9fb8`)**

- Only one file modified: `services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go`
- Diff stat: 1 insertion(+), 1 deletion(-)
- Change: line 35 removed, line 37 added (the `packetmodel` import line, moved only)
- No logic changed, no other files touched, no import aliases modified

**Formatting check**

```bash
$ gofmt -l services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go
(no output)
```

File is now properly formatted per gofmt.

**Module-wide formatting check**

```bash
$ gofmt -l services/atlas-channel/atlas.com/channel
(no output)
```

All files in the module pass formatting.

**Alphabetical order (verified in place)**

Current import block (lines 32-40):
```
cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
invpkt "github.com/Chronicle20/atlas/libs/atlas-packet/inventory"
invcb "github.com/Chronicle20/atlas/libs/atlas-packet/inventory/clientbound"
messengerpkt "github.com/Chronicle20/atlas/libs/atlas-packet/messenger"
messengercb "github.com/Chronicle20/atlas/libs/atlas-packet/messenger/clientbound"
packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
routine "github.com/Chronicle20/atlas/libs/atlas-routine"
tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
```

Alphabetically correct: `packetmodel` (starting with 'p') sorts after `messengercb` (starting with 'm').

### Verdict

**ADDRESSED**

The import was correctly reordered to alphabetical order, the file is now gofmt-clean, and no extraneous changes were introduced. The formatting gate will no longer block this branch.
