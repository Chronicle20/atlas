# Review — Task 18 (DOM-04 close-out gate)

Commit under review: `775e6aacf` (range `c81210fb8..775e6aacf`).
Brief: `.superpowers/sdd/plan/task-18-brief.md`. Report: `.superpowers/sdd/plan/task-18-report.md`.

## Scope

`git diff --stat c81210fb8..775e6aacf`: 25 files — 6 evidence/inventory files regenerated,
`progress.md` appended, `handwork-notes.md` one-line edit, and 17 `rest.go` files across 9
service modules rewritten from getter-based to direct-field `Transform` bodies (D1 close-out).
This matches the report's description exactly. No surprise files, no scope creep.

## 1. Evidence/ledger work (Steps 1–3)

- `inventory-dom04-has-model.txt` regenerated to 30 lines (progress.md:2427), triaged into four
  groups. Verified by direct inspection:
  - **8 already-documented residues**: not independently re-verified line-by-line (low risk,
    matches Task 14/17 batches already reviewed in prior gates) — accepted on the strength of
    prior review, not re-litigated here (correctly out of this task's new-risk surface).
  - **3 "false residue" claims** (`dragons/dragon`, `summons/summon`, `storage/asset`) —
    **independently confirmed**: `dragons/dragon` has `func Transform(m Model) (RestModel, error)`
    at `services/atlas-dragons/atlas.com/dragons/dragon/resource.go:41`; `summons/summon` at
    `services/atlas-summons/atlas.com/summons/summon/resource.go:44`; `storage/asset` at
    `services/atlas-storage/atlas.com/storage/asset/processor.go:100`. Claim holds.
  - **2 accepted NO-RESTMODEL gap-fill** (`channel/data/tradeability`,
    `inventory/data/tradeability`) — plausible per design §8.2's `TransformCash`/`...Setup`
    pattern; not independently re-verified in depth (secondary to this task's real risk) but no
    red flag found.
  - **17 genuine forgotten packages** — **independently confirmed all 17** at the exact commit
    `775e6aacf` (via `git show 775e6aacf:<pkg>/rest.go`, not the live working tree, which has
    unrelated uncommitted Task 18b-A work-in-progress for 6 of these paths). Every one of the 17
    has a `RestModel` but no `Transform` in its own `rest.go`, and none hides a `Transform` in a
    sibling `resource.go`/`processor.go` (checked `cashshop/character` and `npc-shops/character`
    specifically, since both have subpackages that do have their own unrelated `Transform`s that
    could have caused a false read). List is accurate — no false gap found.
- Escalation to BLOCKED and the controller's Task 18b insertion are accepted per the reviewer's
  brief; not treated as a finding against this task.

## 2. D1 getter → direct-field rewrite (the real-risk item)

**Methodology check**: the report's in-scope/out-of-scope split is drawn from
`git diff --diff-filter=AM eaa5ce6f7..HEAD -- '*rest.go'`, i.e., the merge base, not guessed.
Independently reproduced: `git diff --diff-filter=AM eaa5ce6f7..HEAD -- '*rest.go' | grep -c
"^+func Transform"` → **185**, matching the report's "185 total" claim exactly. A second,
independent method (diffing `func Transform*` signatures present in `git show eaa5ce6f7:<file>`
against the current file for all 170 files touched anywhere in the branch) also produced exactly
**185** new functions — cross-confirms the count via an unrelated technique.

**Completeness check**: extracted the body of all 185 branch-added `Transform*` functions at HEAD
(`775e6aacf`) and grepped for `m.<ExportedGetter>()` calls (excluding the accepted
`m.Field()`/`m.field.WorldId()` nested-access pattern). Result: **zero** remaining violations
except the six lines in `atlas-character/character/session/history/rest.go`, which is exactly the
one function the report identifies and correctly excludes (a bare rename of pre-existing
`TransformToRest`, commit `968001916`, predating the branch). This independently confirms the
rewrite is both complete (no branch-added Transform left un-fixed) and correctly scoped (no
pre-existing Transform was touched that shouldn't have been).

**Behavior-equivalence check, per call site** — read the corresponding getter definition in each
touched file's `model.go` and confirmed every rewritten `m.field` read is a byte-for-byte plain
passthrough (`return m.field`, no copy/nil-guard/computed value/pointer deref) for all 17 files:

- `door/rest.go` (20 fields) — all plain fields on `door/model.go:34-54`; `WorldId`/`ChannelId`/
  `MapId`/`Instance` correctly kept as `m.field.WorldId()` etc. (nested `field.Model` value type),
  not flattened.
- `maps/location/rest.go` (6 fields) — plain fields, `maps/location/model.go:24-29`.
- `monster/information/rest.go` — `Attacks()`/`MonsterId()` plain fields, `model.go:14-20`.
- `monster/rest.go` (18 fields incl. `StatusEffects`) — plain fields, `model.go:44-130`;
  `WorldId`/`ChannelId`/`MapId`/`Instance` correctly kept nested via `m.field.WorldId()`.
- `mts/configuration/rest.go` — `FixedSaleHours: m.fixedSaleHours` verified against
  `FixedSaleDurationHours()` (`model.go:44-45`, plain passthrough despite the getter/field name
  mismatch — same storage).
- `parcel/rest.go` (19 fields) — plain fields.
- `pet/rest.go` (15 fields incl. `Excludes()` slice) — `Excludes()` confirmed plain
  (`model.go:101-103`).
- `quest/rest.go` (9 fields + `Progress()` slice) — plain fields.
- `reactor/rest.go` (13 fields) — plain, nested `field.Model` access preserved correctly.
- `trade/rest.go` (1 field) — plain.
- `character/configuration/rest.go` — `m.pendingExpiry.Hours()` plain.
- `dragons/character/rest.go` (5 fields) — plain.
- `guilds/party/rest.go` (`Transform` + `TransformMember`, 11 fields total) — plain, including
  `m.field.WorldId()`/`ChannelId()`/`MapId()`/`Instance()` for `MemberModel`.
- `inventory/data/setup/rest.go` (11 fields) — plain.
- `maps/reactor/rest.go` (13 fields, field named `f` not `field`) — plain, `m.f.WorldId()` etc.
  correctly preserved.
- `npc/conversation/rest.go` (`TransformChoice`, `TransformOperation`, `TransformOutcome`,
  `TransformOption`) — verified these 4 are the only branch-added functions in this large file;
  the many other `Transform*` functions in the same file (`TransformState`, `TransformDialogue`,
  `TransformGenericAction`, `TransformCraftAction`, `TransformTransportAction`,
  `TransformOptionSet`, etc.) that still use getters were **confirmed pre-existing at merge base
  `eaa5ce6f7`** (`git show eaa5ce6f7:<file> | grep '^func Transform'` lists all of them) — correctly
  left untouched.
- `trades/data/inventory/rest.go` — `m.assets`/`m.id`/`m.inventoryType`/`m.capacity` all plain
  (`model.go:69-73`), including the `Type()`→`inventoryType` name mismatch, verified equivalent.

No behavior change found in any of the 21 rewritten functions across 17 files.

**Non-blocking documentation nit**: `progress.md:2497-2501` says "Found 22 matches across 18
functions... remaining 21 [...] in 17 files were [...] rewritten" — 18 minus the 1 excluded
function is 17, not 21. The final function/file enumeration in `progress.md:2506-2525` is correct
(21 functions, 17 files, independently re-counted and matching); the "18 functions" figure in the
summary sentence is simply inconsistent with it. Cosmetic only — does not affect the actual
rewrite, which was independently verified complete and correct by the method above.

Module builds independently re-run and clean: `atlas-channel` (door, maps/location, monster,
mts/configuration, parcel, pet, quest, reactor, trade), `atlas-character` (configuration,
session/history), `atlas-dragons` (character), `atlas-guilds` (party), `atlas-inventory`
(data/setup), `atlas-maps` (reactor), `atlas-npc-conversations` (conversation), `atlas-trades`
(data/inventory).

## 3. Doc edits

`handwork-notes.md` — single-line edit updating the `maps/location` Task 17 prose from
`m.CharacterId()` to `m.characterId`, same underlying fact, consistent with the code change.
Verified.

## Not evaluable

- The 8 "already documented" and 2 "NO-RESTMODEL gap-fill" residue classifications were not
  independently re-verified line-by-line (lower-risk secondary claims, and the 8 were already
  covered by prior task reviews); flagged here for completeness rather than absorbed silently.

## Verdict rationale

All three deliverables hold up under independent, code-level verification: the false-residue
claims are real, the 17-forgotten-package list is accurate with no false gap, and — the
highest-risk item — the 21 getter→field rewrites are complete, correctly scoped against the merge
base, and behavior-neutral at every call site checked. One cosmetic arithmetic inconsistency in
the progress-log narrative is the only defect found.

```
verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-263-backend-guideline-conformance/review-task-18.md
scope_confirmed: commit 775e6aacf only — DOM-04 evidence/ledger regeneration, the 21-function
  D1 getter→direct-field rewrite across 17 rest.go files, and the one-line handwork-notes.md edit.
  17-package BLOCKED escalation and Task 18b insertion accepted as instructed, not re-litigated.
blocking: 0
non_blocking: 1
  - docs/tasks/task-263-backend-guideline-conformance/progress.md:2497-2501 — "22 matches across
    18 functions" is inconsistent with the correct final count of 21 functions / 17 files listed
    two paragraphs later; cosmetic narrative arithmetic error only, the actual rewrite is complete
    and correct.
not_evaluable: 1
  - the 8 "already documented" + 2 "NO-RESTMODEL gap-fill" residue classifications
    (progress.md:2434-2448) were not re-verified line-by-line; lower risk, covered by prior gates.
```
