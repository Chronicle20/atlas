# task-188 pre-PR review — disposition

Three reviewers ran against `origin/main...HEAD` before the PR was opened:
`backend-guidelines-reviewer` (`audit.md` / `audit.json`),
`packet-completeness-critic` (`completeness-critic.md`), and a targeted review
of the `origin/main` merge resolution. This file records what was accepted and
what was not, so the findings are not re-litigated from the raw reports alone.

## Accepted and fixed

| Finding | Source | Disposition |
|---|---|---|
| `matrix --check` failing on `toolSha` | merge review | Fixed in `8e28035fd`. `toolSha` derives from git HEAD, so committing the merge staled the file that was clean when generated. Regenerated; only the `toolSha` line differed. |
| Missing `coverage-manifest.yaml` | completeness critic | Written — `coverage-manifest.yaml`, derived from the actual git + matrix delta. |
| Cash cell demotions undocumented | completeness critic | Documented — `cash-fixture-demotions.md`. |
| `USE_CASH_ITEM × gms_v48` silently de-verified | found while documenting the above | **Real defect.** `ef950fdb1` rewrote `cash/serverbound/v48_test.go` and deleted the two `USE_CASH_ITEM` megaphone fixtures along with the `CASHSHOP_OPERATION` ones, though its message accounts only for the latter. Both restored verbatim; they pass unchanged; the cell is `verified` again. |
| Raw version comparisons | backend reviewer | Converted, plus every other raw comparison in the same three files, for consistency. |

## Downgraded, with reasoning

**"3 blocking findings" (backend reviewer) → style nits.**

- `set_field.go:74,126` (`> 28`) is **not this branch's code** — both occurrences
  already exist on `origin/main`. Converted anyway while cleaning the file.
- The helpers are pure sugar: `libs/atlas-tenant/tenant.go:93-99` defines
  `MajorAtLeast(v)` as `majorVersion >= v` and `MajorAtMost(v)` as `<= v`. No
  behavioural difference, consistent with the reviewer's own "byte-identical"
  note.
- The cited rule was mis-stated. The backend guidelines contain no ban on
  numeric comparisons. The actual prescription is
  `docs/packets/IMPLEMENTING_A_PACKET.md:103`, and it is narrow: use
  `MajorAtLeast(87)`-style gates *"never `> 83`, which is an off-by-one that
  wrongly routes v84 down the v87 branch."* No flagged site sits near an
  adjacent-version boundary — there is no version between 28 and 29, 12 and 13,
  or 60 and 61.

The conversions were still made: greppability is what makes the v83/v84 class of
bug findable, which is the rule's real purpose.

**"CHANGED-BUT-UNCLAIMED gates" (completeness critic) → declared, not defective.**

All three cited gates carry per-version IDA evidence in their codec comments,
and the critic's own full matrix diff confirms zero non-v48 cell changes:

- `reactor/clientbound/spawn.go` cites both sides of the boundary (v48
  `@0x5a54b4`, v72 `@0x69207c`, v79 `@0x6b77bb` read no name; v83 `@0x735127`,
  v84, v87, v95 do). It is a **bug fix** — Atlas had been writing three legacy
  versions a trailing string their clients never read.
- `pet/serverbound/spawn.go` labels its v61 placement `INFERRED, not verified`
  in the source, states that `gms_v61.yaml` registers no serverbound
  `SPAWN_PET` so no cell depends on it, and says to re-derive before trusting.

The valid residue was that none of this was *declared* anywhere — which is
finding 1, now fixed. All four touched versions are listed in the manifest's
`versions:` with their evidence status.

## Merge resolution

All six conflict resolutions in `fe9963017` were independently verified against
the parent blobs, the registry, and the tooling: export union exact (1047 + 1114
→ 1115, set difference empty both ways), fnames matching `gms_v48.yaml:612,622`,
template re-sort lossless, corpus delta entirely v48, nine-template reformat
`json.loads`-equal to main. No content loss, no semantic regression.
