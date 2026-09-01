# Review: task-286 Task 2, fix round 2 (`8410746d2`)

Scope: commit `8410746d2` (range `55bfc577d..8410746d2`) only.
Files touched: `tools/scratch-sweep.sh` (+25/-11), `tools/scratch-sweep_test.sh` (+30).

## Brief for this round

Round-1 review proved a destructive bypass: the dangerous-root guard compared the
literal `$root` string, so a `..`-segment root (e.g. `$tmp/decoy/../real`) walked past
every check — including the absolute-path and two-component checks — and swept a real
directory. Required fix: canonicalize before any guard, re-run every dangerous-root
check against the canonical path, add a `..`-segment fixture test, and correct the false
comment that the resolved root is "always exactly what was configured."

## What changed

`tools/scratch-sweep.sh`:
- The absolute-path check (`case "$root" in /*) ;; *) die ... esac`) now runs FIRST,
  against the raw `$root`, before canonicalization — with a comment explaining why:
  `realpath -m` would otherwise absolutize a relative root against `$PWD` and defeat
  this guard.
- `root="$(realpath -m -- "$root")"` runs immediately after, before any dangerous-root
  check.
- The `/`, `/tmp`, `/var/tmp` case-arm, the `$HOME` comparison, and the two-component
  count check all now run against the reassigned (canonical) `$root`.
- The `$HOME` comparison itself now canonicalizes `$HOME` too
  (`home_canonical="$(realpath -m -- "$HOME")"`), so a symlinked or `..`-bearing `$HOME`
  is compared correctly.
- The old `/tmp/`, `/var/tmp/` trailing-slash arms and the `${HOME}/` arm are dropped —
  correctly so, since `realpath -m` output is always slash-normalized; keeping them
  would have been dead code.
- The stale comment claiming "the resolved root is always exactly what was configured"
  is gone, replaced by an accurate one.

`tools/scratch-sweep_test.sh`: adds four new assertions — two `..`-segment roots that
resolve to `/tmp` and to `$HOME` (both must be refused, both checked under `--dry-run`
only, consistent with the file's existing convention of never running `--now` against a
non-fixture path), plus one `..`-segment root that resolves inside the `mktemp` fixture
(swept with `--now`, verified both that the target file is gone and that the printed
summary names the canonical path, not the literal `..`-bearing string).

## Verification performed

1. **Original exploit, reproduced against the fixed script** (fixture-scoped, `--dry-run`
   only, no destructive proof needed since the guard now fires):

   ```
   tmp=$(mktemp -d); mkdir -p "$tmp/real" "$tmp/decoy"; touch "$tmp/real/canary"
   ATLAS_SCRATCH_ROOT="$tmp/decoy/../real" ./tools/scratch-sweep.sh --dry-run --now
   ```
   Output: `scratch-sweep: would remove 1 entry from <tmp>/real` — the canonical,
   resolved path, correctly identifying `canary` as a sweep candidate rather than being
   rejected by a stale literal-string comparison. The prior round's exploit (silently
   walking past every guard including the two-component and absolute-path checks) no
   longer applies: canonicalization happens before those checks run, so they see the
   real destination and would legitimately allow it (fixture-internal, not a dangerous
   root) — this is correct behavior, not a residual gap.

2. **New bypass attempts** (all `--dry-run`, no real system path swept):
   - Symlink to `/tmp`: `ATLAS_SCRATCH_ROOT="$tmp/tmp_link"` where `tmp_link -> /tmp` →
     refused (`refusing to sweep dangerous root: /tmp`, rc=2). `realpath -m` resolves
     symlinks as well as `.`/`..`, so this class of bypass is also closed, beyond what
     the brief strictly required.
   - Trailing slash on a dangerous root: `ATLAS_SCRATCH_ROOT="/tmp/"` → refused, rc=2.
   - Double-slash: `ATLAS_SCRATCH_ROOT="//tmp"` → refused, rc=2.
   - `..` that resolves to `/tmp` via a different path shape (`/tmp/a/..`) → refused,
     rc=2.
   - Relative root that *would* resolve to `/tmp` if canonicalized (`cd /tmp &&
     ATLAS_SCRATCH_ROOT="../tmp"`) → refused as "not an absolute path" (rc=2), confirming
     the round-1 relative-root guard still fires on the raw string and is not defeated
     by the reordering. This was the specific regression the implementer reports
     self-catching; confirmed still fixed.
   None of these attempts found a live bypass.

3. **Full test suite**: `bash tools/scratch-sweep_test.sh` → all 37 assertions pass,
   including the four new ones and all pre-existing ones (no regressions).

4. **Shellcheck**: `tools/shell-guard.sh --require-shellcheck` → `76 script(s) OK
   (syntax + shellcheck -S error)`.

## Findings

None blocking. The reordering (absolute-path check on the raw string first, then
canonicalize, then every dangerous-root check against the canonical value) is correct
and closes the round-1 exploit without reopening the round-0/round-1 relative-root
guard. The `$HOME` canonicalization is a genuine improvement over a literal `$HOME`
comparison that the brief didn't explicitly call out but which the same reasoning
requires. The stale comment is corrected. Test additions honestly pin the new behavior
(the two refusal assertions and the fixture-internal sweep-and-verify-canonical-name
assertion all fail against the pre-fix script, since none of `/`, `/tmp`, `/var/tmp`,
`$HOME`, or the two-component check would have compared canonical values there).

Non-blocking observations (not required by the brief, not blocking approval):
- `realpath -m` is coreutils-specific; no fallback exists if it's absent from the
  environment (e.g. a minimal BusyBox shell). This is unchanged from round 1's
  introduction of `realpath -m` and out of scope for this fix-only commit.
- The dropped comment `"and the two-component count AND the absolute-path
  requirement"` ordering match with the code is now: absolute-path check → canonicalize
  → dangerous-literal case → home check → component count. This matches the described
  intent; no discrepancy found.

## Not evaluable

None — the full diff surface (36 changed lines across two files) was read and every
claim in the round-2 report was independently reproduced.

## Verdict

APPROVED. The fix closes the round-1 destructive bypass, the implementer's self-reported
mid-round regression (breaking the relative-root guard) is verifiably still fixed by the
reorder, and no new bypass was found despite deliberate adversarial attempts (symlink,
trailing slash, double slash, alternate `..` shapes, relative-root-that-would-resolve-
dangerously).
