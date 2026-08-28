# Review — Task 25-B (commit `9ce61b728`)

Brief: `.superpowers/sdd/plan/task-25b-brief.md`
Report: `.superpowers/sdd/plan/task-25b-report.md`
Scope: commit `9ce61b728` only. 25-A's commit `4ed1cef25` reviewed separately
(`docs/tasks/task-263-backend-guideline-conformance/review-task-25a.md`); its DOM-04 content is
re-reviewed here only where 25-B was required to touch it (Step 0 and Step 0b).

## Diff surface

`git diff 4ed1cef25..9ce61b728` touches exactly one file:
`docs/tasks/task-263-backend-guideline-conformance/exemptions.md` (201 insertions, 38 deletions).
No `.go` file, `sweep.sh`, `classify-*.tsv`, `inventory-*` file, or `handwork-notes.md` is in the
diff — confirmed with `git show --stat 9ce61b728 --name-only`. This matches the brief's rules and
report's self-review claim.

## Step 0 — "Excluded from this section" correction

Re-checked the `resource.go` present/absent column independently with `ls`, not the report's claim:

```
services/atlas-channel/atlas.com/channel/party_quest/resource.go        -> No such file
services/atlas-maps/atlas.com/maps/data/map/monster/resource.go         -> No such file
services/atlas-npc-conversations/atlas.com/npc/cosmetic/resource.go     -> No such file
services/atlas-transports/atlas.com/transports/instance/resource.go     -> present (4.8K)
services/atlas-marriages/atlas.com/marriages/marriage/resource.go       -> present (5.8K)
```

Matches the brief's table and `exemptions.md:167-232` exactly. The subsection now carries the
controller's N/A-by-trigger disposition and the five-row table, with per-row reasoning tying the
direction present to the package's inbound-client vs. JSON:API-server role, and reads as resolved —
not as five open findings (the old heading "controller finding" / "flagged for the controller, not
ruled on here" language is gone, replaced with "controller ruling" / "Disposition: N/A by trigger").
25-A's note that `marriage` was `atlas-marriages`'s only 176-row entry is preserved verbatim
(`exemptions.md:230-232`). PASS.

## Step 0b — six blocking citations from `review-task-25a.md`

All six headings resolved against HEAD:

| heading | citation | `sed -n` at HEAD | verdict |
|---|---|---|---|
| `### services/atlas-maps` (:87) | `monster/rest.go:11` | `type RestModel struct` | exact |
| `.../merchant/data/portal` (:335) | `rest.go:7`/`:10`/`:15`/`:31` | `type RestModel struct` / `.Target` / `.ScriptName` / `type Model struct` | exact |
| `.../channel/mts/listing` (:355) | `rest.go:14` | `type RestModel struct` | exact |
| `.../channel/mts/wish` (:359) | `rest.go:10` | `type RestModel struct` | exact |
| `.../npc-shops/npc/character` (:367) | `rest.go:12`, `model.go:13` | `type RestModel struct` / `type Model struct` | exact |
| `.../messages/data/map` (:375) | `model.go:3`, `messages/map/processor.go:40` | `type Model struct{}` / `func (p *ProcessorImpl) Exists(mapId _map2.Id) bool {` | exact |

All six now carry a real `<path>.go:<line>`. Confirmed via `git diff 4ed1cef25..9ce61b728` that only
these six bullets' evidence sentences changed inside Section 3 — no other bullet in Section 3 was
touched, restructured, or reworded (the diff hunk for `mts/listing`/`mts/wish`/`npc-shops/character`/
`messages/data/map`/`merchant/data/portal` shows single-sentence citation insertions only, disposition
sentences byte-identical). PASS.

## Citation accuracy at HEAD — the review's core test

Sampled 30+ citations across all five of 25-B's new sections, deliberately targeting the packages the
task brief called out as relocated (`atlas-cashshop/asset`, `atlas-npc-conversations/conversation`,
`atlas-channel/transport/route`, `atlas-pets/data/pet`), plus the `NewBuilder`-name-taken and
FILE-05 entries.

- `atlas-cashshop/asset` — all 8 constructors (`model.go:116`; `builder.go:36/275/497/536/569/604/660`)
  resolved exactly to the named declaration.
- `atlas-npc-conversations/conversation` — all 20 sibling builders (`builder.go:30` through `:1468`)
  resolved exactly; `wc -l` confirms `builder.go` is 1567 lines, well past the highest cited offset.
- `atlas-channel/transport/route` — `builder.go:31/166/173/225/244`, all five resolve; `grep -n "^func New"`
  on HEAD shows exactly these five constructors and no `NewModelBuilder` — the entry's claim that the
  TSV's sixth constructor no longer exists is correct.
- `atlas-pets/data/pet` — `builder.go:14` is `func NewBuilder() *Builder {`, `builder.go:85` is
  `func NewSkillModelBuilder() *SkillModelBuilder {` — the entry's claim of a rename from
  `NewModelBuilder` to `NewBuilder` is correct (only `NewBuilder`/`NewSkillModelBuilder` exist at
  HEAD; the TSV's `NewModelBuilder (model.go:65)` is gone).
- `atlas-channel/asset` (NewBuilder-name-taken) — `builder.go:119` and `:127` (brief cited `:126`;
  entry used the re-derived `:127`) both resolve exactly.
- `atlas-character/character` (NewBuilder-name-taken) — `builder.go:32/93/133/166/170` all resolve
  exactly (`type Builder struct`, `func NewBuilder(...)`, `type modelBuilder struct`,
  `func NewEmptyBuilder() *modelBuilder`, `func CloneModel(m Model) *modelBuilder`).
- FILE-05 entity builders — all 4 `entity_builder.go` citations resolve to `type entityBuilder struct`
  at the exact lines cited.
- FILE-05 excluded tree — `libs/atlas-packet/model/skill_usage_info.go:143` resolves to
  `type SkillUsageInfoBuilder struct`.
- Additional spot-checks: `npc-conversations/conversation/quest` (`builder.go:24/111`),
  `party-quests/reward` (`builder.go:11/47`), `quest/quest` (`builder.go:22`,
  `entity_builder.go:24`), `transports/transport` (`builder.go:30/175/240`) — all resolve exactly.
- DOM-01 trigger-not-fired (5 `NO-MODEL-GO` packages) — `ls <pkg>/model.go` fails for all five
  (`libs/atlas-database`, `libs/atlas-packet/model`, `channel/mist`, `channel/skill/handler`,
  `marriages/kafka/message`), matching the entry's claim of absence.

Zero citation failures found across every sample taken (30+ distinct `file:line` pairs, all five new
sections plus both Step 0/0b corrections). This is a materially different result from a naive
transcription of the stale TSV, and the two content divergences the report flags
(`transport/route` losing a duplicate constructor, `pets/data/pet`'s rename) are real, confirmed at
HEAD, and would only surface from genuine re-derivation — strong evidence the implementer actually
re-derived rather than copied. PASS.

## Acceptance criterion 3 — every `### ` heading followed by a `.go:<line>` citation

Independently re-ran the same check the report describes (not trusting its printed output):

```
total headings: 62
bad: []
```

All 62 `### ` headings in the finished file (25-A's plus 25-B's) are followed, before the next
`### `, by at least one backtick-quoted `<path>.go:<line>`. PASS.

## Acceptance criterion 4 — `out of scope` grep

```
grep -n "out of scope" exemptions.md
```
returns 4 hits (lines 280, 292, 298, 310), all inside 25-A's DOM-04 content
(`atlas-drops/data/foothold`, `atlas-saga-orchestrator/reactor/drop`, `atlas-pets/data/position`,
and the `atlas-transports/transport` batch), and each hit line itself carries a `file:line` citation
(e.g. line 280: `rest.go:76`; line 310: `rest.go:50` etc.). PASS.

## Counts (Step 4)

- DOM-01 sibling builders: 61 `SIBLING-EXEMPT` rows dedupe to 25 packages — independently counted
  with `awk '/^## DOM-01 — sibling builders/,/^## DOM-01 — trigger not fired/'` piped to
  `grep -c '^- `services'`, which returns 25. Matches the report.
- DOM-01 trigger-not-fired: 5 packages, matches.
- DOM-01 NewBuilder-name-taken: 2 entries, matches.
- FILE-05 entity builders: 4 entries, matches.
- FILE-05 excluded tree: 1 entry, matches.
- Two content divergences from the TSV (not count divergences) are documented and confirmed above.

## Repo conventions

- No literal home or absolute path in the committed file: `grep -nE "/home/|/root/|/Users/"` returns
  no hits. PASS.
- Only `exemptions.md` staged/committed; commit message matches the brief's required subject line.
  PASS.

## Not evaluable

- None. Every check the brief specified was independently reproducible within this unit's scope.

## Verdict rationale

All five sections and both corrections (Step 0, Step 0b) hold up under independent re-derivation.
Every sampled citation — including the two flagged content divergences — resolves exactly at HEAD.
Both hard acceptance criteria (heading-citation coverage, `out of scope` grep) pass on independent
re-run, not just on the implementer's own script output. No forbidden file was touched. No blocking
findings.

---

verdict: APPROVED
artifact: docs/tasks/task-263-backend-guideline-conformance/review-task-25b.md
scope_confirmed: commit 9ce61b728 only — exemptions.md's 5 new DOM-01/FILE-05 sections, the Step 0
"Excluded from this section" rewrite, and the Step 0b six-citation fix in Section 3. 25-A's
unmodified DOM-04 content was not re-reviewed.
blocking: 0
non_blocking: 0
not_evaluable: 0
