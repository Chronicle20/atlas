# Review: Task 10 — Route USE_INNER_PORTAL in seed templates

Commit reviewed: `836b773b2` (`git show 836b773b2`), parent `db5744980`.
Brief: `.superpowers/sdd/plan/task-10-brief.md` (CONTROLLER AMENDMENTS section —
ten templates, not six).
Implementer report: `.superpowers/sdd/plan/reports/task-10-report.md`.

## Scope confirmed

`git show --stat 836b773b2` touches exactly ten files, all
`services/atlas-configurations/seed-data/templates/template_{gms_48,gms_61,
gms_72,gms_79,gms_83,gms_84,gms_87,gms_92,gms_95,jms_185}_1.json`, +9/-0 lines
each. No Go file, no other template, no unrelated file rode along. Matches
the brief's ten-file scope exactly.

## Directed checks

### 1. Ten templates, not six

Confirmed via `git show --stat`: all ten in-scope templates carry the new
entry (`grep -c "InnerPortalHandle" template_*.json`, 1 hit each for the ten,
0 for `template_gms_12_1.json`). `template_gms_12_1.json` does not appear in
the commit's file list at all (the one text match is inside the commit
message body, not a diff hunk) — confirmed genuinely untouched. **PASS.**

### 2. Opcode vs. derivation docs

Cross-checked each template's inserted `opCode` against
`docs/tasks/task-250-inner-portal-registration/structures/{gms_vNN,jms_v185}.md`
directly (not the brief's summary table):

| Template | Diff opCode | Structure-doc opcode | Match |
|---|---|---|---|
| gms_48 | `0x50` | `COutPacket(...,80)` = `0x050` | yes |
| gms_61 | `0x5D` | `COutPacket(...,93)` = `0x05D` | yes |
| gms_72 | `0x64` | `COutPacket(...,100)` = `0x064` | yes |
| gms_79 | `0x63` | `COutPacket(...,99)` = `0x063` | yes |
| gms_83 | `0x65` | `COutPacket(v58, 0x65)` = 101 dec, registry `opcode:101` matches | yes |
| gms_84 | `0x65` | `COutPacket(&v65, 101)` = `0x65` | yes |
| gms_87 | `0x68` | `COutPacket(&a3, 0x68)` = 104 dec, registry `opcode:104` matches | yes |
| gms_92 | `0x70` | doc quotes `COutPacket(&v74, 0x70u)`; registry `opcode:112` (=`0x70`) matches | yes |
| gms_95 | `0x71` | `COutPacket(&rc, 113)` = `0x071`, registry `opcode:113` matches | yes |
| jms_185 | `0x60` | `COutPacket(v67, 0x60)`; registry `opcode:96` (=`0x60`) matches | yes |

All ten match. v92 hazard note (adjacent `0x6F`/111 is a different op) is
correctly avoided — the inserted opcode is `0x70`/112, matching both the
doc's decompiled constructor call and the registry cross-check quoted in the
structure doc. **PASS.**

### 3. Handler binding constant

`services/atlas-channel/atlas.com/channel/main.go:902`:
```go
handlerMap[portal2.InnerPortalHandle] = handler.InnerPortalHandleFunc
```
`libs/atlas-packet/portal/serverbound/inner_portal.go:14`:
`const InnerPortalHandle = "InnerPortalHandle"`, and `Operation()` (line 56)
returns that constant. All ten templates bind the literal string
`"InnerPortalHandle"`, matching both the registered handler key and the
codec's `Operation()` return value exactly. **PASS.**

### 4. Sorted-opcode rule

`tools/template-opcode-order-guard.sh` → `OK: 22 template arrays are in
ascending opcode order.` (exit 0). Manually confirmed in each diff hunk that
the new entry is inserted immediately before the next-higher existing
`opCode` (e.g. gms_48: `0x50` inserted before `0x51`; gms_92: `0x70` inserted
before `0x71`), per `docs/packets/TEMPLATE_CONVENTIONS.md`'s ascending-order
rule (§"Rule: ascending opcode order (enforced)"). **PASS.**

### 5. Independent confirmation of the ten edits (not on the implementer's word)

- `tools/template-duplicate-binding-guard.sh` → `OK: 22 template arrays carry
  no duplicate (name, opCode) binding.` (exit 0) — no duplicated binding.
- `git show --stat 836b773b2` lists exactly the ten intended files, `+90/-0`
  total (9 lines × 10) — no partial application, no stray file from another
  task's stash.
- Each of the ten files shows exactly one `InnerPortalHandle` occurrence
  (`grep -rc "InnerPortalHandle" template_*.json`) — no doubled entries.
- `python3 -m json.tool` was not re-run by this review, but
  `tools/template-opcode-order-guard.sh` and
  `tools/template-duplicate-binding-guard.sh` both parse all 22 template
  files successfully (exit 0), which is not possible on invalid JSON — this
  transitively confirms all ten edited files are still valid JSON.
- The implementer's stash-pop incident narrative is corroborated by the final
  landed diff being clean and complete; independent verification above does
  not rely on that narrative. **PASS.**

### 6. `ChatGeneralChat` dangling-symbol claim

Ran `tools/template-symbol-check.sh` on all ten edited templates individually
(with per-invocation timeouts; the tool is slow — one run timed out at 120s
mid-batch and was resumed file-by-file):

- OK: `template_gms_48_1.json`, `template_gms_61_1.json`,
  `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`
- `DANGLING: ChatGeneralChat (no registered string literal found)` /
  `FAIL: template has dangling symbol references`: `template_gms_72_1.json`,
  `template_gms_79_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`,
  `template_jms_185_1.json`

Exactly 5 templates fail, matching the implementer's claim. Verified
pre-existing independently (not on the implementer's word): extracted
`template_gms_72_1.json` at the parent commit (`git show db5744980:...`) and
confirmed `ChatGeneralChat` already appears once in that pre-task-10 version
(`grep -c ChatGeneralChat` → `1`), and `diff` between the parent and current
version of that file shows zero lines touching `ChatGeneralChat` — the only
diff hunk is the ten-line `InnerPortalHandle` insertion, entirely unrelated.
The dangling symbol is therefore genuinely pre-existing and not introduced by
this commit. **PASS** (claim verified, not accepted on faith).

## Other observations

- Working tree has unrelated uncommitted `atlas-ui` page modifications
  (`AccountsPage.tsx`, `BansPage.tsx`, etc.) — pre-existing dirty state from a
  concurrent task in this shared-`.git` worktree setup, not part of
  `836b773b2` and out of this review's scope.

## Verdict

All six directed checks pass with direct evidence. No blocking findings.

```text
verdict: APPROVED
artifact: docs/tasks/task-250-inner-portal-registration/review-task-10.md
scope_confirmed: git show 836b773b2 — ten template_*.json seed-data files, +90/-0, InnerPortalHandle route insertion only; no Go module touched
blocking: 0
non_blocking: 0
not_evaluable: 0
```
