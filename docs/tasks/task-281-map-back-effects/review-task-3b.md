# Review — Task 3b (Name the unnamed BackEffect handler functions in the IDBs)

Range: `265d9668b..4f87fdfb4` (single commit `4f87fdfb4`)
Brief: `.superpowers/sdd/plan/task-3b-brief.md`
Report: `.superpowers/sdd/plan/task-3b-report.md`

## Scope containment — PASS

`git diff --name-only 265d9668b..4f87fdfb4`:

```
docs/packets/ida-exports/gms_v61.json
docs/packets/ida-exports/gms_v72.json
docs/packets/ida-exports/gms_v79.json
docs/packets/ida-exports/gms_v92.json
```

Exactly the four files the brief authorized, nothing else. No Go code, no
`libs/atlas-packet/` changes, no `docs/packets/registry/*.yaml`, no template
edits, no `packet-audit matrix`/`evidence pin` artifacts anywhere in the
commit. `git status --porcelain -uno` after checkout is clean — no residual
untracked artifacts left by this unit.

## Export diff — minimal and correct — PASS, with one report-accuracy note

`git diff --numstat 265d9668b..4f87fdfb4`:

```
7  8   gms_v61.json
5  0   gms_v72.json
5  0   gms_v79.json
10 0   gms_v92.json
```

Total: **27 insertions / 8 deletions**. The report and the task prompt both
say "28 insertions / 9 deletions" — off by one in each direction from the
actual `git diff --numstat` total. Cosmetic, does not affect the substance of
the change; noted as non-blocking.

Per-file key diff (`git diff` filtered to top-level fname keys added/removed):

- `gms_v61.json`: only `CMapLoadable::OnSetBackEffect` (existing stub, now
  resolved from `"address": ""`/`unresolved: true` to `0x5a8316`) and a new
  `CMapLoadable::OnClearBackEffect` entry (`0x5a871b`) — file.py:5298-5316.
- `gms_v72.json`: only a new `CMapLoadable::OnClearBackEffect` entry
  (`0x5f5f54`) — file.py:6123-6127.
- `gms_v79.json`: only a new `CMapLoadable::OnClearBackEffect` entry
  (`0x614977`) — file.py:6598-6602.
- `gms_v92.json`: two new entries, `CMapLoadable::OnSetBackEffect`
  (`0x606d80`) and `CMapLoadable::OnClearBackEffect` (`0x612ef0`) —
  file.py:14176-14186.

No entry outside BackEffect changed in any file (confirmed by grepping the
diff for changed top-level `"    "<key>": {` lines — only the four listed
keys appear).

Well-formedness and convention checks:

- All four files parse as valid JSON (`python3 -m json.load` succeeded on
  each).
- All four files end with a trailing newline; no tabs (2-space indent
  preserved, matches surrounding convention).
- `generated_at` / `md5` top-level metadata in `gms_v79.json` is byte-identical
  before/after (verified by diffing the parsed values) — confirms the
  implementer's claim that the file was hand-rewritten to preserve the
  original serialization rather than accepting the tool's sorted-key
  re-serialization. The other three files show the same surgical,
  single-block-insertion pattern in `git diff` (no reflow of unrelated
  regions), which is consistent with the same hand-preservation approach.
- `"calls": null` on all spliced entries — expected per the review brief,
  not a finding.

## Address/symbol agreement — PASS

Cross-checked every renamed/spliced address against
`docs/tasks/task-281-map-back-effects/structures/{gms_v61,gms_v72,gms_v79,gms_v92}.md`:

| Export | Address | Structures doc match |
|---|---|---|
| v61 `OnSetBackEffect` | `0x5a8316` | `structures/gms_v61.md:55` |
| v61 `OnClearBackEffect` | `0x5a871b` | `structures/gms_v61.md:56` |
| v72 `OnClearBackEffect` | `0x5f5f54` | `structures/gms_v72.md:29` (`sub_5F5F54`) |
| v79 `OnClearBackEffect` | `0x614977` | `structures/gms_v79.md:26` |
| v92 `OnSetBackEffect` | `0x606d80` | `structures/gms_v92.md:29` |
| v92 `OnClearBackEffect` | `0x612ef0` | `structures/gms_v92.md:30` |

All match exactly. (`ReloadBack` and `Field::BackEffect::Decode` addresses —
`0x5163ae`, `0x5a81e2`, `0x5f5a1b`, `0x612d80` — are IDB-only renames per the
brief; they correctly do not appear as top-level fname keys in the exports,
since the exports track packet-handler fnames, not their internal callees.
This is consistent with the brief's Files list, which only names the four
export JSONs and does not ask for internal-callee export entries.)

## Claims that need checking — not independently verifiable without IDA

1. **v79 "already symbolized, only the export entry was missing."** The
   export diff is consistent with this claim: `gms_v79.json` gained only
   `CMapLoadable::OnClearBackEffect`; `OnSetBackEffect` at `0x614572` was
   already present in the pre-commit export unchanged (verified: `git show
   265d9668b:.../gms_v79.json` already has `OnSetBackEffect` resolved). The
   underlying claim that the IDB itself already carried the symbol (so no
   `idb_save`/rename occurred on that session) cannot be confirmed from the
   repo — there is no git-tracked artifact of IDB rename state. **Not
   evaluable.**
2. **v92 `0x612d80` ReloadBack — inferred name promoted to confirmed via
   shape match + already-symbolized caller thunk at `0x612ef0`.** The
   structures doc (`structures/gms_v92.md:96-102`, written in an earlier
   unit) already states the name was "inferred" at that time and that
   `sub_612D80` "rebuilds the back layers ... which is the `ReloadBack`
   body," matching the report's summary of the pre-existing evidence. The
   *additional* confirmation this unit claims to have performed (point-for-
   point structural comparison against `CAnimationDisplayer::SetCenterOrigin`
   at v95 `0x442e10`, and reading `0x612ef0`'s decompile as a pure thunk to
   `sub_612D80`) requires a live IDA decompile session I do not have access
   to. **Not evaluable** — the report's narrative is internally consistent
   and consistent with the structures doc, but the decompile itself is
   unverifiable from repo state alone.
3. **Per-address decompile confirmations generally** (the `RemoveAll` ->
   `Getcenter`/vtbl+100/vtbl+64 -> `RestoreBack` -> `GetVecCtrl` ->
   `SetCenterOrigin` shape chain cited for each `ReloadBack` rename, and the
   `Decode1;Decode4;Decode1;Decode4` shape for the v61 decode callee) are
   IDA-only findings with no git-tracked counterpart to check against.
   **Not evaluable.**

## Non-blocking notes

- Report/prompt insertion-deletion count (28/9) is off by one from the
  actual `git diff --numstat` total (27/8). Trivial, does not affect
  correctness of the committed content.
- The report's "Out of scope, noted" section (undocumented decode callees on
  v72/v79) correctly declines to widen the IDB mutation beyond the brief's
  target table — appropriate restraint, not a defect.

## Verdict rationale

Everything checkable from repo state — scope containment, diff minimality,
JSON well-formedness, key-order/formatting preservation, and address/symbol
agreement against the committed structures docs — passes. The claims that
require a live IDA session (decompile shape confirmations, IDB-side rename
state) are correctly out of reach for this review and are reported as
not-evaluable rather than accepted on faith.
