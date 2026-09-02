# Fix I — verify-final6 gate FAIL: missing .gitignore for the new generator binary

## Verdict
`verify-final6.log` trailer: `FLAGLESS VERIFY EXIT=1`. Exactly one check failed,
`verify_test.sh`, with 5 assertions. Every other gate passed, including the
93-module build/vet, all drift guards, and the lint & format guard.

## Root cause (confirmed, not inferred)
This branch added a new generator module, `libs/atlas-kafka/gen`. Building or
running it (`cd libs/atlas-kafka/gen && go run .`) makes Go name the output
binary after the directory, producing `libs/atlas-kafka/gen/gen` — an 8.4 MB
ELF. That path is NOT in `.gitignore`, so it lands as an untracked file *inside
a `libs/` module*. `verify.sh` therefore correctly concluded that `libs/`
changed and fanned the closure out to its 2 consumers.

The branch already carries the precedent it needed: `.gitignore:75` is
`/libs/atlas-constants/gen/gen`, with the comment "`go build ./...` in
libs/atlas-constants/gen names its binary after the directory (gen) — trivially
recreated, never a deliverable." The new generator module was added without the
matching entry.

## Evidence
- `git status --porcelain` showed `?? libs/atlas-kafka/gen/gen`; `git check-ignore -v` → NOT IGNORED.
- The 5 failures all name the same string:
  - `no libs change, no fan-out` — want `none`, got `shared-lib-closure:libs/atlas-kafka/gen (2 consumers)`
  - `a dirty go.work.sum alone does not fan out` — want `none`, got the same closure
  - `module count agrees` — want `0`, got `2`
  - `a dirty go.work.sum alone selects zero modules` — want `0`, got `2`
  - `the fan-out reason names the closure (got '')` — knock-on from the polluted baseline
- After `rm -f libs/atlas-kafka/gen/gen`, `tools/verify.sh --facts --quick --base f46773f31`
  reports `changed_libs=none`, `fanout_reason=none`, `modules_selected=0` — precisely the
  values the failing assertions wanted.

## Fix
One line in `.gitignore`, following the `libs/atlas-constants/gen/gen` precedent
verbatim. No change to `verify.sh`, `verify_test.sh`, or `go-work.sh` — the gate
was reporting the truth about a dirty tree; the tree was wrong, not the gate.

## Note for the next gate run
The stray binary has been removed in the working tree. Any future `go run .` in
that module recreates it, which is exactly why the ignore entry is the durable
fix rather than a one-time cleanup.
