# Review — commit fdb08036a (fix-sparse-baseline-restore-collision)

Scope: single commit `fdb08036a` on branch `fix-sparse-baseline-restore-collision`.
Files touched: `services/atlas-pr-bootstrap/scripts/bootstrap.sh`,
`services/atlas-pr-bootstrap/test/data_ingest_test.bats`,
`docs/tasks/fix-sparse-baseline-restore-collision/bug-sparse-baseline-restore-collision.md`.

Diff confirmed via `git show --stat fdb08036a` and `git show fdb08036a -- <file>`
matches the file list above; no scope drift.

## Claim verification

### 1. `baseline/rewriter.go` rewrites only tenant_id, copies PKs verbatim — CONFIRMED

`services/atlas-data/atlas.com/data/baseline/rewriter.go:37-53` (`Rewriter.Stream`):
iterates every field of every row via the COPY-binary field loop; only when
`int(i) == rw.TenantColIndex` does it substitute `rw.Target` — every other
field, `copyN(in, out, int(size))`'d straight through with no rewrite. The
row's own `id` field is one of those verbatim-copied fields (the `documents`
table's PK). Restoring into a database that already holds the same rows will
therefore collide on `documents_pkey`. Matches the brief exactly.

### 2. `document/storage.go` falls back to `canonical.TenantId` on 0 rows — CONFIRMED, both paths

`services/atlas-data/atlas.com/data/document/storage.go`:
- `ByIdProvider` (per-id, lines 28-63): on a registry+DB miss for the caller's
  tenant, it constructs `nt` from `canonical.TenantId(...)` and retries both
  the registry and DB lookup under that tenant context.
- `AllPagedProvider` (paged, lines 73-99): if the tenant-scoped page has
  `Total == 0`, it retries under `canonical.TenantId(...)` and returns that
  page if it has rows.

This is the load-bearing premise for the fix and it holds for both read
shapes the brief called out — not narrower than claimed.

One thing worth flagging precisely: `status.go`'s `queryStatus` (the endpoint
the guard actually reads) does **not** go through `document/storage.go` at
all — it does a direct `Count` filtered to the resolved `tenantId`, with no
fallback (`data/status.go:63-94`). So the *guard's* `docs=0` reading is a
literal, non-fallback tenant count, while the *application's* reads (through
`Storage`) do fall back. That asymmetry is exactly why the old guard's
"tenant owns 0 rows" signal was misleading and the new two-count guard is
the right fix — the guard now explicitly reads both the raw tenant count and
the raw canonical count (`?scope=shared`) rather than relying on any
in-process fallback.

### 3. `scope=shared` is operator-gated exactly as assumed — CONFIRMED

`services/atlas-data/atlas.com/data/data/status.go:120-134`
(`resolveStatusTenantId`): `case "shared":` checks
`r.Header.Get("X-Atlas-Operator") != "1"` and 403s otherwise. The new
`bootstrap.sh:536` call sends exactly `-H "X-Atlas-Operator: 1"`. Matches.

### 4. Guard cannot regress isolated/full mode — TRACED, holds

- Fresh empty isolated DB: `queryStatus` counts 0 rows for the tenant
  (`docs=0`) and 0 for `canonical.TenantId` (`canon=0`, since nothing has
  ever written under that id in a private DB) → `[ "$docs" = "0" ] && [
  "$canon" = "0" ]` is true → restore fires (`bootstrap.sh:540-557`). Same
  outcome as before the fix.
- Post-restore re-run: `Restorer.Restore` targets `target=TENANT_ID` (the
  caller's tenant, from `restore_body`'s `tenantId: $TENANT_ID`,
  `bootstrap.sh:546`), so the tenant now owns rows → `docs>0` on the next
  run → guard's `&&` is false → skip, same as before. Confirmed by reading
  `bootstrap.sh:540` and the restore body construction; `baseline/restore.go`
  was not further audited beyond confirming the `target` id it receives
  (out of the surface this commit touches).
- Sparse mode (the reported failure): tenant owns 0 rows (`docs=0`,
  measured live in the bug doc at documentCount 0) but canonical owns 49049
  (`canon=49049`) → `&&` is false → skip, exactly the fix's intent, and
  matches the live numbers quoted in the bug doc.

No regression path found in the two code paths touched.

## Shell correctness

### `document_count`'s integer validation — correct

`case "$n" in '' | *[!0-9]*) return 1 ;; esac` (`bootstrap.sh:91-93`) is a
standard POSIX bracket-expression negation (`!` inside `[...]`), valid glob
syntax for `case`, not an extglob feature. It rejects the empty string and
any string containing a non-digit — including `"null"`, `"lots"`, and
negative numbers. Verified directly against the bats cases (`document_count
fails on a JSON:API error body`, `... fails on a non-numeric count`, `...
fails when the request fails`) — all pass. Correct.

### `get_attr`'s new `"$@"` pass-through — does not break the two other callers

`grep -n "get_attr" bootstrap.sh` shows exactly three call sites:
- `document_count` (line 90) — the new caller, passes extra args.
- `data_processing_done` (lines 135-136) — `get_attr "$ATLAS_UI_BASE/api/data/status" documentCount` and `... updatedAt`, each with exactly 2 args.

`get_attr` does `local url="$1" attr="$2"; shift 2` then forwards `"$@"`
(now empty for the two 2-arg callers) before `"$url"`. Empty `"$@"`
expansion is a no-op in the curl invocation, so both existing callers are
byte-for-byte unaffected. Confirmed no third caller exists.

## Test honesty

- `setup()` (`data_ingest_test.bats:9-39`) extracts `get_attr` and
  `document_count` via `sed` into a sourced file, then asserts
  `declare -F get_attr` / `declare -F document_count` before running any
  test — guarding exactly the failure mode the commit message calls out
  (a missing definition would exit 127, which trivially satisfies every
  "must fail" assertion).
- Ran the suite live against the post-fix script:
  `bats services/atlas-pr-bootstrap/test/data_ingest_test.bats` → `1..10`,
  all 10 `ok`.
- Ran the suite against the pre-fix parent commit's `bootstrap.sh`
  (`git show 54dadd9e2:services/atlas-pr-bootstrap/scripts/bootstrap.sh`,
  which has no `document_count` function): all 10 tests fail, and — this is
  the part that matters — they fail **through the `setup()` guard**
  (`document_count not extracted from .../bootstrap.sh`, `return 1` in
  `setup`), not via a 127 from a missing function inside the test body. This
  is the correct failure mode; the anti-vacuous-pass guard actually holds.
- Ran the full bats suite in the module:
  `bats services/atlas-pr-bootstrap/test/*.bats` → 106/106 passing,
  matching the commit message's claim.
- Ran `shellcheck services/atlas-pr-bootstrap/scripts/bootstrap.sh`: 5
  findings, all on lines untouched by this diff — `SC1091` x3 (lines 17,
  24, 26, pre-existing `.` sourcing) and `SC2034`/`SC1010` (lines 563, 583,
  pre-existing `ATLAS_STEP=seed` / `ATLAS_STEP=done log info ...`). The
  commit message's "four remaining findings" underclaims by one line (I
  count 5 individual warnings across the SC1091/SC2034/SC1010 set, likely a
  miscount of related-but-distinct warnings vs. distinct rule IDs) but the
  substance is correct: zero new findings on the changed lines.

## Not evaluable

- **Ingress query-string pass-through for `?scope=shared`.** Could not
  search the repo's ingress manifests for `rewrite-target` /
  `configuration-snippet` / query-string-stripping annotations — the shared
  `/tmp` on this box hit `ENOSPC` mid-review (unrelated to this commit; a
  side effect of an earlier exploratory command in this session) and the
  Bash tool became unusable for the rest of the session. This was flagged
  by the requester as a known offline-untestable risk going in, so it is
  reported here as genuinely not evaluated rather than silently passed. If
  the ingress does normalize or strip query strings on this route,
  `document_count "$ATLAS_UI_BASE/api/data/status?scope=shared" ...` would
  hard-fail every sparse bootstrap at the `could not read the canonical
  document count` guard (fail-closed, not silently wrong — but still a
  hard bootstrap abort). Worth a live smoke test on the next sparse PR
  environment before trusting this path unconditionally.
- `baseline/restore.go`'s `Restorer.Restore` internals (lock acquisition,
  table order, `cleanupAfterFailure`) were read only far enough to confirm
  `target` is the caller's `TENANT_ID`; the wider restore-safety question is
  explicitly out of scope per the bug doc's "Not addressed here" section and
  was not touched by this commit.

## Verdict rationale

All four claims the brief asked to verify against source held up exactly as
stated, including the highest-value one (claim 2, both per-id and paged
fallback). The `get_attr`/`document_count` shell changes are correct and
backward-compatible with existing callers. The bats tests fail for the
right reason pre-fix and pass post-fix; the anti-vacuous-pass guard in
`setup()` was exercised live and does hold. Full suite and shellcheck match
the commit message's claims. The one item I could not check (ingress
query-string handling) was already known by the requester to be
untestable offline and is reported as not-evaluable rather than an
approval gap.
