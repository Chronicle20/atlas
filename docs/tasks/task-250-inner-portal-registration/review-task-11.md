# Review: Task 11 — packet-audit linkage for USE_INNER_PORTAL

Commit reviewed: `bb7ec8dbd` (range `db5744980..bb7ec8dbd`)
Brief: `.superpowers/sdd/plan/task-11-brief.md` (CONTROLLER AMENDMENTS section — ten
registries, not six)
Report: `.superpowers/sdd/plan/reports/task-11-report.md`

## Scope confirmed

`git show --stat bb7ec8dbd` shows exactly the 11 files the brief (as amended)
calls for: `tools/packet-audit/cmd/run.go` and the ten registry YAMLs
(`gms_v48/61/72/79/83/84/87/92/95`, `jms_v185`). No stray file from another
task's stash leaked in, and the concurrently-edited
`services/atlas-configurations/seed-data/templates/*.json` /
`services/atlas-ui/**` files are untouched by this commit (they show only as
working-tree churn, per the given instruction to ignore them).

## Findings

### 1. All ten registries — PASS

- Six pre-existing rows (`gms_v83/84/87/92/95`, `jms_v185`) each got exactly
  one `packet: portal/serverbound/PortalInnerPortal` line inserted between
  `fname:`/`note:` and `provenance:` (matching the brief's specified key
  order), no other field touched. Verified via diff hunks: each file shows a
  single `+1` insertion.
- Four new rows (`gms_v48/61/72/79`) each got a full `USE_INNER_PORTAL` block:
  `op`, `direction`, `opcode`, `fname`, `provenance: ida-discovered`,
  `ida.address`, `note`, `packet`. Key order and shape match the existing
  `ida-discovered` + `packet` precedent in the same files, e.g.
  `docs/packets/registry/gms_v48.yaml:974-981` (USE_ITEM entry: fname →
  provenance → ida → note → packet). Confirmed one `USE_INNER_PORTAL` block
  and one `packet:` line per file, no duplicates (`grep -c` = 1/1 across all
  ten files).
- `gms_v12.yaml` does not exist in the repo (`docs/packets/registry/`
  contains no such file) — confirms the "gms_v12 gets no row" requirement is
  satisfied because there is nothing to add a row to.

### 2. Opcode derivation — PASS, independently verified

Checked each registry's opcode directly against its
`docs/tasks/task-250-inner-portal-registration/structures/gms_vNN.md` /
`jms_v185.md` `COutPacket` constructor citation (not trusting the brief's
table):

| version | structure doc | registry opcode | match |
|---|---|---|---|
| v48 | `COutPacket::COutPacket(v47, 80)` → 80 (0x050) | 80 | yes |
| v61 | `COutPacket::COutPacket(v49, 93)` → 93 (0x05D) | 93 | yes |
| v72 | `COutPacket::COutPacket(v59, 100)` → 100 (0x064) | 100 | yes |
| v79 | `COutPacket::COutPacket(v58, 99)` → 99 (0x063) | 99 | yes |
| v83 | `COutPacket::COutPacket(v58, 0x65)` → 101 | 101 | yes |
| v84 | `COutPacket::COutPacket(&v65, 101)` → 101 | 101 | yes |
| v87 | `COutPacket::COutPacket(&a3, 0x68)` → 104 | 104 | yes |
| v92 | `COutPacket::COutPacket(&v74, 0x70u)` → 112 | 112 | yes |
| v95 | `COutPacket::COutPacket(&rc, 113)` → 113 | 113 | yes |
| jms185 | `COutPacket::COutPacket(v67, 0x60)` → 96 | 96 | yes |

All ten agree with the brief's derived-values table and with the actual
structure docs. No disagreement found.

The v48 row correctly omits `fieldKey` from its note (5-field body, "zero
Encode1 calls" per the structure doc), consistent with
`libs/atlas-packet/portal/serverbound/inner_portal.go`'s `encodesFieldKey`
gate (`GMS && MajorAtLeast(61) || JMS`), which is pre-existing code this task
did not touch but which corroborates the v48-is-the-exception claim.

### 3. `candidatesFromFName` case — PASS

`tools/packet-audit/cmd/run.go:2852-2858` (post-diff) adds the case
immediately after `CUserLocal::CheckPortal_Collision` inside the
`--- World: portal (serverbound) ---` block, same comment-then-case shape as
its neighbour, correctly returning
`{name: "InnerPortal", pkg: "portal", dir: csvpkg.DirServerbound}`.
`qualifiedWriterName("portal", "InnerPortal")` (`run.go:223-228`) yields
`PortalInnerPortal`, matching the `packet:` value written into all ten
registries. `libs/atlas-packet/portal/serverbound/inner_portal.go` exists,
carries the `// packet-audit:fname CUserLocal::TryRegisterTeleport` marker,
and its 6-field `Encode`/`Decode` (fieldKey + portalName + x + y + targetX +
targetY, fieldKey gated by `encodesFieldKey`) matches the comment's wire
description. `go build ./tools/packet-audit/...` succeeds; only one
non-comment occurrence of the new case (`grep -c TryRegisterTeleport` = 2:
one comment reference, one case label — no duplicate case).

### 4. Stash-incident claim — independently verified, not taken on faith

- `git show --stat bb7ec8dbd` shows exactly 11 files changed, matching the
  brief's file list plus the four controller-amendment registries — no extra
  or missing file.
- Line counts per file match the report's own claim (9 insertions for each
  of the four new-row registries, 1 insertion for each of the six
  `packet:`-only registries, 10 insertions for `run.go` — confirmed via
  `git show bb7ec8dbd -- <file>` per-file diff stat).
- No duplicate or half-applied blocks: every registry has exactly one
  `USE_INNER_PORTAL` op and one `packet:` line; `run.go` has exactly one new
  `case "CUserLocal::TryRegisterTeleport":`.
- `git status --short` in the current worktree shows no stray files from
  another task's stash — only `services/atlas-ui/**` working-tree churn from
  a concurrent, unrelated task (out of scope per the given instruction).
- `go build` and `go test ./tools/packet-audit/...` both pass cleanly.

The report's claim of a full, clean redo after the stash-clobber incident is
corroborated by the actual committed diff.

### 5. `matrix --check` — corroborated

Re-ran `go run ./tools/packet-audit matrix --check` independently; grepping
the output for `dangling|orphan|conflict|PortalInnerPortal|InnerPortal|
USE_INNER_PORTAL` returns zero matches. The only failures are the expected
`STATUS.md`/`status.json` staleness (out of scope per this review's
directed-check #5, deferred to Task 12).

## Not evaluable

- The report's `matrix --check` claim was not independently baselined
  against a true pre-edit state (the report itself notes it could not
  capture a real pre-edit baseline either, due to the 2-minute Bash default
  timing out on the first attempt). This review confirms the *post-edit*
  output contains no InnerPortal-related dangling/orphan/conflict lines,
  which is the substance of what the brief's Step 3 asks for, but a true
  before/after diff of the full `matrix --check` output was not produced by
  either the implementer or this review.

## Verdict rationale

Every directed check passes with `file:line`/command evidence gathered
independently of the implementer's report. No blocking or non-blocking
defect found.
