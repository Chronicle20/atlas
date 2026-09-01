# H4 diagnosis: bake-target assertions regressed by the go-work.sh drift check

## Symptom

Three `tools/verify_test.sh` assertions that passed in the previous gate run now fail with
empty `got` values:

```
FAIL - two changed go.mods select two bake targets   want: atlas-account,atlas-ban   got: (empty)
FAIL - two bake targets produce exactly one bake gate   want: 1   got: 0
FAIL - the gate names the target count   want: docker buildx bake (2 target(s))   got: (empty)
```

## Root cause (confirmed, not hypothesized)

Commit `b5552f669` ("fail loudly when a services/ or libs/ module is missing from go.work")
added `check_workspace_drift()` to `tools/lib/go-work.sh` and wired it into
`tools/verify.sh`'s `all_modules()` (verify.sh:243-256). That function is called
unconditionally at script top level via `changed_modules()` → `_changed_modules_out="$(changed_modules | sort -u)" || exit 1`
at verify.sh:425, **before** the `FACTS` branch at verify.sh:941 is ever reached. There is no
`if [ "$FACTS" -eq 1 ]` guard around module discovery — `--facts` still has to compute
`MODULES` to answer "what would be selected," so it inherits every failure mode of the real
module walk, including the new drift check.

The bake-target test block (`tools/verify_test.sh:355-374`) plants two probe "modules" to
generate two changed `go.mod` paths for `bake_targets()` to select on:

```sh
probe_ban_dir="$HERE/../services/atlas-ban/zz-verify-probe-${probe_tag}"
probe_account_dir="$HERE/../services/atlas-account/zz-verify-probe-${probe_tag}"
probe_bake_ban="$probe_ban_dir/go.mod"
probe_bake_account="$probe_account_dir/go.mod"
...
mkdir -p "$probe_ban_dir" "$probe_account_dir"
: > "$probe_bake_ban"
: > "$probe_bake_account"
...
assert_eq "two changed go.mods select two bake targets" "atlas-account,atlas-ban" \
  "$(facts_key bake_targets --base HEAD)"
```

These are empty placeholder `go.mod` files under `services/atlas-ban/` and
`services/atlas-account/`, created solely to give `bake_targets()` (which matches CHANGED
paths ending in `go.mod` against `.github/config/services.json`) two changed-go.mod paths to
key off of. They were never intended to be real Go modules and were never meant to be added
to `go.work`'s `use` list.

`all_modules()`'s `find "$ROOT/services" "$ROOT/libs" -name go.mod ...` picks up these probe
files as newly-discovered module directories. `check_workspace_drift()` then diffs that
discovered set against `workspace_module_dirs()` (go.work's `use` list, unmodified by the
test) and — correctly, by its own new contract — treats the two probe directories as modules
"found under services/ or libs/ but absent from go.work," which is now a **named, non-zero
failure** instead of a silently-dropped module. `verify.sh` exits 1 with the error on stderr
before printing anything to stdout, so every `facts_key`/`facts_selected` capture (which reads
only stdout) gets `(empty)`, exactly matching the observed `got: (empty)` and `got: 0`.

## Reproduction

With the tree otherwise clean, by hand:

```
mkdir -p services/atlas-ban/zz-verify-probe-repro services/atlas-account/zz-verify-probe-repro
: > services/atlas-ban/zz-verify-probe-repro/go.mod
: > services/atlas-account/zz-verify-probe-repro/go.mod
tools/verify.sh --facts --quick --base HEAD
```

Result: exit code 1, empty stdout, stderr:

```
verify.sh: ERROR — the following module(s) under services/ or libs/ are not in go.work's 'use' list:
verify.sh: ERROR —   .../services/atlas-account/zz-verify-probe-repro
verify.sh: ERROR —   .../services/atlas-ban/zz-verify-probe-repro
verify.sh: ERROR — add the missing module(s) to go.work's 'use' list to fix this
```

This reproduces the exact failure signature from the gate report. Probe directories removed
afterward; `git status --short` hash before and after the repro was identical
(`b6a613a67a64c1ec9f4cc1b5d8f38c76`), confirming the tree was left exactly as found.

## Is this the real regression, or interference from concurrent runs?

Real regression, not interference. The reproduction above was done on a clean tree (no
concurrent `verify_test.sh` runs, no leftover `zz-verify-probe-*` litter beyond what was
created and removed by this diagnosis) and deterministically reproduces the failure signature
from first principles by reading the source of `b5552f669` and `verify_test.sh`'s own
bake-target setup. The mechanism is structural — any `services/`/`libs/` `go.mod` created
outside `go.work`'s `use` list now aborts `verify.sh`, `--facts` included — and
`verify_test.sh`'s own bake-target test always creates exactly that condition on the very
next line after commit `b5552f669` started enforcing it. No coincidence with the two runaway
subagents is required to explain it; it would fail exactly this way on any run, concurrent or
not, once `b5552f669` landed. (The runaway-subagent litter at
`services/zz-verify-probe-broken-*` / `tools/zz-verify-probe-*_test.sh` is a separate, already
cleaned-up concern and was not present during this diagnosis's reproduction.)

## The design question: should a workspace-hygiene check abort `--facts`?

No — not in its current unconditional form. `--facts` is documented (verify.sh:67-71) as "print
WHAT this invocation would select... executes no gate." Its entire value proposition is that it
is cheap and side-effect-free to call repeatedly, including from tests that want to interrogate
selection logic without paying for `docker buildx bake` or `go build`/`go vet`/`go test`.

`check_workspace_drift()` is a legitimate check — the bug it fixes (`libs/atlas-kafka/gen`
silently unbuilt) was real. But it conflates two different failure classes under one gate:

1. **A genuine repo-state defect**: a committed module directory that's missing from
   `go.work`. This is exactly what `b5552f669` was written to catch, and it should fail loudly
   — on the real run, and arguably even under `--facts` *for a real invocation over real
   CHANGED paths*.
2. **Ephemeral test scaffolding**: `verify_test.sh`'s own probe `go.mod` files, which exist for
   the lifetime of a single assertion block, are guarded by an flock, and are deliberately never
   registered in `go.work` because they aren't real modules — they're bake-target bait.

The check has no way to distinguish these two cases; it walks the filesystem, not intent. That
means either the check needs an escape hatch for known-ephemeral probe paths, or (cleaner) the
check belongs to a phase that the bake-target test doesn't traverse.

**Recommendation**: `check_workspace_drift()`'s current placement — inside `all_modules()`,
invoked unconditionally from `changed_modules()` at module-discovery time — is too early and
too broad. It should not gate `--facts`'s answer to "what bake targets did you select," which
has nothing to do with Go workspace membership. Two viable fixes, in order of preference:

- **Preferred**: `bake_targets()` (verify.sh:551-570) is confirmed independent of
  `all_modules()`/`changed_modules()`/`MODULES` — it only reads `$CHANGED` and
  `.github/config/services.json`, nothing from the Go-modules layer. So the fix is to stop
  computing `_changed_modules_out="$(changed_modules | sort -u)" || exit 1` (verify.sh:425)
  unconditionally at script top level, and only run it (and therefore `all_modules()` /
  `check_workspace_drift()`) when the Go-modules layer will actually execute — i.e. on the real
  run, or on a `--facts` query that specifically needs the module list (not every `--facts`
  invocation, and specifically not `bake_targets`/`facts_key bake_targets`).
- **Fallback**: exempt the test's probe paths (`zz-verify-probe-*`) from the drift check by
  having `verify_test.sh` register them in a throwaway `go.work` copy for the duration of the
  probe — more invasive, and it papers over the real design smell (a `--facts`-cheap path
  shouldn't run a check whose entire purpose is to validate the real build graph).

Either way, the honest answer to the framing question is: **yes, this belongs on the real run,
not on `--facts`.** `--facts` should reflect what the real run would select, and workspace
membership doesn't change what's selected — the real run will independently fail on the same
drift the moment it tries to build/vet/lint that module. Aborting `--facts` early only breaks a
documented "no gate execution" contract for no additional safety.

## Recommended fix

In `tools/verify.sh`, stop computing `changed_modules()`/`all_modules()` (and therefore
`check_workspace_drift()`) unconditionally ahead of the `FACTS` branch when the `--facts`
invocation doesn't need Go module membership to answer the caller's query (e.g.
`facts_key bake_targets` only needs `CHANGED` + `.github/config/services.json`, not
`all_modules()`). Concretely: audit whether `bake_targets()` transitively depends on `MODULES`/
`all_modules()` at all — if not, defer `_changed_modules_out="$(changed_modules | sort -u)" ||
exit 1` (verify.sh:425) so it only runs when the Go-modules layer will actually execute (real
run) or when a `--facts` query specifically needs the module list, rather than unconditionally
at script top level.

Do **not** weaken `check_workspace_drift()` itself or make it skip probe-looking paths by
naming convention — that reintroduces exactly the silent-drop failure mode `b5552f669` was
written to close, just with a different exemption list. The fix belongs in *when* the check
runs, not in loosening *what* it checks.
