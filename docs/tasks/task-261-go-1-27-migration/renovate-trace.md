# Renovate AC-14 reasoning trace: Go toolchain and golang image bumps never auto-merge

AC-14 cannot be demonstrated with a live Renovate PR on this branch, so it is
satisfied by reasoning over the resulting `packageRules` array in `renovate.json`
after Task 9's two appended overrides (indices 11 and 12, 0-based).

## 1. Renovate semantics relied on

`packageRules` is a JSON array. Renovate evaluates every rule that matches a
given dependency update in array order and applies their fields cumulatively;
where two matching rules set the same field (here, `automerge`), **the later
rule in the array wins**. This is documented Renovate behavior and is the
mechanism Task 9 uses: appending the override rules at the *end* of the array
guarantees they are evaluated last for any update they match, overriding any
earlier `automerge: true` set for the same dependency.

## 2. `gomod` + `go` package: every matching rule, in array order

Command run against the post-change `renovate.json`:

```bash
jq -r 'to_entries | .[] | select(.key=="packageRules") | .value | to_entries | .[] \
  | select(.value.matchManagers // [] | index("gomod")) \
  | select((.value.matchPackageNames // ["<unfiltered>"]) | index("go") or index("<unfiltered>")) \
  | "\(.key)\t\(.value.matchPackageNames // "unfiltered")\tautomerge=\(.value.automerge)"' renovate.json
```

Output (pasted verbatim):

```
0	["go"]	automerge=true
1	["go"]	automerge=false
4	unfiltered	automerge=true
5	unfiltered	automerge=false
6	unfiltered	automerge=true
7	unfiltered	automerge=false
11	["go"]	automerge=false
```

| index | matchManagers | matchPackageNames | automerge | notes |
|---|---|---|---|---|
| 0 | gomod | `["go"]` | true | `go-version` group, minor/patch (pre-existing) |
| 1 | gomod | `["go"]` | false | major only (pre-existing) |
| 4 | gomod | none (`matchPackagePatterns: ["^github.com/Chronicle20/"]`) | true | Chronicle20 libs, minor/patch. The query's fallback of `matchPackageNames // ["<unfiltered>"]` treats any rule lacking `matchPackageNames` as "unfiltered", which is a blind spot: this rule is actually scoped by `matchPackagePatterns` to `github.com/Chronicle20/*` and does not match the `go` package. It is listed here only because the Step-3 query (taken verbatim from the brief) cannot see `matchPackagePatterns`. |
| 5 | gomod | none (`matchPackagePatterns`) | false | Chronicle20 libs, major. Same blind-spot caveat as index 4 — does not actually match `go`. |
| 6 | gomod | none (genuinely unfiltered) | true | `matchUpdateTypes: ["patch", "minor"]`, no package filter at all — this is the rule that carries the pre-existing drift (see §5). |
| 7 | gomod | none (genuinely unfiltered) | false | major only, no package filter. |
| 11 | gomod | `["go"]` | false | **Task 9's new override.** Last matching rule in the array. |

The two rows at indices 4 and 5 are a false-positive artifact of the jq query
used (it treats the *absence* of `matchPackageNames` as "matches everything",
which is technically true for Renovate's own matching engine only when no
other `match*` field narrows the rule — here `matchPackagePatterns` does
narrow it to Chronicle20 packages, so index 4/5 never actually apply to the
`go` package in practice). They do not change the conclusion: whether or not
they are counted, index 11 is still the last row and it is `automerge=false`.

## 3. `dockerfile` + `golang` package: every matching rule, in array order

Command:

```bash
jq -r 'to_entries | .[] | select(.key=="packageRules") | .value | to_entries | .[] \
  | select(.value.matchManagers // [] | index("dockerfile")) \
  | select((.value.matchPackageNames // ["<unfiltered>"]) | index("golang") or index("<unfiltered>")) \
  | "\(.key)\t\(.value.matchPackageNames // "unfiltered")\tautomerge=\(.value.automerge)"' renovate.json
```

Output (pasted verbatim):

```
2	["golang"]	automerge=true
12	["golang"]	automerge=false
```

| index | matchManagers | matchPackageNames | automerge | notes |
|---|---|---|---|---|
| 2 | dockerfile | `["golang"]` | true | `dockerfile-golang` group (pre-existing) |
| 12 | dockerfile | `["golang"]` | false | **Task 9's new override.** Last matching rule in the array. |

No genuinely unfiltered `dockerfile` rule exists in `renovate.json` (the only
other `dockerfile` rule, index 3, is scoped to `matchPackageNames: ["node"]`
and does not match `golang`), so this table is clean with no blind-spot rows.

## 4. Conclusion

In both tables, the last matching rule sets `automerge: false`:

- `gomod` + `go`: last matching rule is index 11, `automerge: false`.
- `dockerfile` + `golang`: last matching rule is index 12, `automerge: false`.

Because Renovate applies matching `packageRules` in array order with later
rules overriding earlier ones on conflicting fields, neither a Go toolchain
(`go` gomod package) bump nor a `golang` Dockerfile base-image bump can
auto-merge after this change, regardless of update type (patch, minor, or
major).

## 5. Pre-change counterfactual

Before Task 9, the last rule matching `gomod` + `go` for a minor/patch update
was index 6 — the genuinely unfiltered `"Auto-merge patch/minor Go
dependencies"` rule (`matchManagers: ["gomod"]`, `matchUpdateTypes: ["patch",
"minor"]`, no package filter, `automerge: true`). This rule sits after index 0
(the `go-version` rule, also `automerge: true` for minor/patch) in the array,
so it did not need to *change* the outcome for `go` — both agreed. But it is
mechanically why editing index 0's `automerge` field in place to `false` would
not have achieved AC-14: index 6 has no package filter, still matches the
`go` package on every minor/patch update, sits after index 0 in the array, and
would win, re-enabling auto-merge. This is exactly how the 1.24 → 1.25 → 1.26
partial-landing drift this task's PRD documents came to exist: a Go toolchain
bump PR was auto-merged by the unfiltered rule even though only some of the
~110 pin sites are visible to Renovate's `gomod` manager. Task 9's fix must
therefore append new override rules at the *end* of `packageRules`, not edit
index 0, so they are evaluated after every other matching rule including
index 6.
