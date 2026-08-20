# Review: Task 8 — `service-config.sh` takes the id as a parameter

Commit range: `f6e33c8b7..100aaae68` (single commit `100aaae68`)
Brief: `.superpowers/sdd/plan/task-8-brief.md`
Report: `.superpowers/sdd/plan/task-8-report.md`

## Scope

`git diff --stat` shows exactly the two files the brief's `### Files` section
names:

```
services/atlas-pr-bootstrap/scripts/service-config.sh   | 67 ++++---------
services/atlas-pr-bootstrap/test/service_config_test.bats | 105 +++++----------------
```

No other files touched. Scope matches the brief.

## Findings

### 1. Published signature — verified exact

Code (`services/atlas-pr-bootstrap/scripts/service-config.sh:78`):

```
build_service_config() {
    local shape="$1" tmpl="$2" id="${3-}" entry
```

Sparse-arm validation (`:88-92`):

```
if [[ ! "$id" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; then
    log error "build_service_config: sparse mode requires a service id (type=$shape, got '${3-}')"
    return 1
fi
```

Isolated arm (`:98-102`) never references `$3`/`id` — byte-for-byte
unchanged behavior confirmed by reading the full function body.

This matches the report's stated interface exactly:
`build_service_config <shape: login|channel|none> <template> [<service_id>]`,
required-and-validated only when `ATLAS_MODE=sparse`, error text
`build_service_config: sparse mode requires a service id (type=$shape, got '<value>')`,
return 1. **PASS** — Task 9's dispatch can carry this signature forward
verbatim.

### 2. Unconverted `bootstrap.sh` call sites — claim is correct but imprecisely worded

Verified against the actual code, not the report's prose:

- `bootstrap.sh:488` (`body=$(build_service_config "$shape" "$payload_path")`)
  is inside `upsert_service_config`, called only from the `else` (isolated)
  branch at `bootstrap.sh:570-577`. Isolated arm ignores `$3` — genuinely
  unaffected.
- `bootstrap.sh:507` (`body=$(build_service_config "$shape" "$tmpl") || return 1`)
  is inside `create_service_config`, called only from the `if
  ATLAS_MODE=sparse` branch at `bootstrap.sh:555-568`. This call site is
  **never reached in isolated mode at all** — the report's blanket framing
  ("Those call sites still work today because ATLAS_MODE defaults to
  isolated, where $3 is ignored") is imprecise when applied to this one,
  since isolated-mode execution never calls `create_service_config` in the
  first place.

However, the report's very next clause correctly names the real
consequence: "any sparse-mode run through bootstrap.sh before Task 9 lands
will now correctly fail loudly instead of silently minting an id — which is
the intended behavior change." That is accurate: in sparse mode,
`build_service_config "$shape" "$tmpl"` now runs with `id=""`, fails the
UUID regex, logs `... requires a service id ...`, returns 1;
`create_service_config` propagates the failure via
`[ -z "$svc_id" ] ... ` / `|| return 1`; the caller's `|| exit 1`
(`bootstrap.sh:566-568`) turns it into a hard PostSync-hook failure — not a
silent bad write. So this is a **deliberate, documented fail-loud
deferral**, not a latent break. No unbound-variable risk either: both `$3`
references in `build_service_config` use `${3-}` default expansion, safe
under `set -euo pipefail` (`bootstrap.sh:14`).

**Non-blocking**: the self-review sentence conflates the two call sites'
different reachability under isolated mode; a reader who only reads that
sentence (not the clause after it) could wrongly conclude both are inert
today. Worth a wording fix in the report if it is amended, but it does not
change the actual runtime behavior, which is correct and intended per D2.

### 3. Newline contract — not pinned at the call site, correctly, because there is no call site yet

Task 8 does not add any `derive-service-id.sh` invocation; `$3` is a
generic parameter validated by regex and passed straight to `jq --arg id
"$id"` (`:93`). The bats test that pins `.data.id`
(`"sparse mode never reads or writes the pinned main service row"`,
`service_config_test.bats:83-96`) passes a literal, no-newline string and
asserts `.data.id` equals it byte-for-byte — this correctly proves
`build_service_config` does not itself add or strip anything around `$3`.

Whether a trailing newline from `derive-service-id.sh`'s stdout would
survive into `$3` is a Task 9 concern (the call site that does
`$(tools/derive-service-id.sh ...)` or reads `SERVICE_ID_<TYPE>`), not
decidable from this diff. The report is honest about this: it states the
newline question depends on how the Task-9 caller forwards the value
(`$(...)` always strips) and does not claim Task 8 pins that contract
end-to-end — it only claims (correctly) that this function's own contract
is pass-through. **PASS**, with the caveat carried forward to Task 9's
review: assert the actual `derive-service-id.sh` output forwards cleanly
into `$3` once that call site exists.

## Tests run directly (not the implementer's numbers)

```
$ bats services/atlas-pr-bootstrap/test
```
133 `ok`, 0 `not ok` (counted via `grep -c "^ok"` / `grep -c "^not ok"` on
a fresh run).

```
$ tools/shell-guard.sh --require-shellcheck
shell-guard: 71 script(s) OK (syntax + shellcheck -S error).
```

Both match the report's claimed numbers.

## Test honesty

- `"build_service_config: sparse fails loudly when no id is supplied"` and
  `"build_service_config: sparse rejects a malformed id"`
  (`service_config_test.bats:100-115`) both fail against the pre-change
  code: the old `build_service_config` had no third parameter and always
  succeeded via `new_uuid` regardless of extra trailing arguments, so a
  2-arg or 3-arg-with-garbage call never produced a non-zero exit or
  "requires a service id" output before this commit. These are honest
  regression tests for the new contract, not tests that pass either way.
- The rewritten `"sparse mode never reads or writes the pinned main service
  row"` tightens `!= pinned && != "null"` to an exact-string match
  (`e7ae96a2-c484-5617-8e28-2178b60a8378`), closing the loophole the old
  `new_uuid`-removal test comment explicitly called out ("" and "null" both
  satisfied the old assertions).
- Deleted `new_uuid:` and UUID-source test cases correctly track the
  function's deletion — none reference dead code.

## Not evaluable

None. Everything the brief and report claim for this unit is verifiable
within `service-config.sh` / `service_config_test.bats` plus the two
`bootstrap.sh` call sites the report itself names.

## Verdict rationale

Signature matches exactly and is safe to hand to Task 9. The one deferred
integration point (`bootstrap.sh`'s two call sites) was verified in the
actual code rather than taken on the report's word, and the sparse-mode
call site (`:507`, inside `create_service_config`) is confirmed to
fail-loud by design rather than silently misbehave — matching D2's intent.
The only issue found is a wording imprecision in the report's self-review
(non-blocking, does not affect code correctness).
