# task-250-inner-portal-registration — execution ledger

Plan: 12 tasks. Base (last gated commit): 5f299e4bb. Branch HEAD at start: 6f0799565.

## Ledger

- Task 1: dispatched (IDA derivation, 6 structure docs)
- Briefs: all 12 generated at `.superpowers/sdd/plan/task-N-brief.md`, fact block prepended, all carry `### Files`.
- IDB session map (adopted, do not reopen): v83=754107bf v84=46c2a2eb v87=c0829805 v92=019cd393 v95=ecc757f4 jms185=a977912e v48=12a398ce v61=921fdbb5 v72=99e435d8 v79=5a1cd4f3

- **Task 1: complete** (commit `b0cfe59e4`, docs-only). Six structure docs written; all six versions byte-identical 6-field layout (`fieldKey`, `portalName`, `x`, `y`, `targetX`, `targetY`); **no `MajorAtLeast` gate required** — only the registry opcode differs per version. Task 3 must carry that ruling.
  - Finding A: jms_v185 `COutPacket` ctor is at `0xa22313`, not design.md's `0xa2230e`. Opcode (96/0x060) matches. Design fact-block address is wrong; the derivation is not.
  - Finding B: gms_v92 has no `CheckPortal_Collision`/`FindPortalByName` symbol; the send site was located via `SendSkillUseRequest` xrefs + size filter, then confirmed by encode signature + opcode 112. Weaker path than v83/v84 — under review.
  - Gate for `5f299e4bb..b0cfe59e4` launched (`--quick --base 5f299e4bb`).
  - Gate `5f299e4bb..b0cfe59e4` (`--quick`): **PASS** (exit 0). Last gated commit = `b0cfe59e4`.
- Task 2: dispatched (⬜ columns v48/v61/v72/v79 + entry-distance threshold). Task 1 review dispatched in parallel.
  - **Task 1 review: APPROVED_WITH_FINDINGS** (0 blocking, 2 non-blocking). Artifact `docs/tasks/task-250-inner-portal-registration/review-task-1.md`.
  - **Finding B CLOSED by controller** (commit `8580d1042`). `sub_8FDBE0` in the v92 IDB *is* `CheckPortal_Collision` — unnamed, not absent. It calls `TryRegisterTeleport(0, 0, portal->pn, portal->tn, 1)`, giving v92 the same argument-role confirmation as v83/v84. Caller set matches 4 of v95's 5 at near-identical sizes incl. a byte-identical named `OnTeleport`. Renamed + IDB saved. **gms_v92 is derived, not inferred — same confidence tier as the other five. Task 3/Task 12 may treat all six equally.**
  - Hazard recorded: v92 `CheckPortal_Collision` case 9 sends adjacent opcode `0x6F`/111 with a FOUR-field body. Not a USE_INNER_PORTAL variant. Do not confuse.
  - Finding A (jms_v185 ctor `0xa22313` vs design.md `0xa2230e`) remains open, non-blocking; reviewer attributes it to ambiguous push-vs-ctor phrasing in design.md:88. Fold a design.md correction into a later docs commit.

- **Task 2: complete** (commit `3b6d4c08a`, docs-only) — but with a **BLOCKING SCOPE FINDING**.
  - Threshold derived and unblocks Task 8: `CheckPortal_Collision` → `CPortalList::FindPortal_Collision` (`0x6ab310`), per-portal `nHRange`/`nVRange` defaulting to 100 in `CPortalList::RestorePortal` (`0x6ad3c0`). Half-extents 50×50 → diagonal 71, +10 margin (the client's own `ptPos.y - 10` foothold-probe literal). **`maxPortalEntryDistance = 81`** — Task 8 copies this verbatim.
  - gms_v12 confirmed absent (CSV `0x000` sentinel). `template_gms_12_1.json` gets no route.
  - **v48/v61/v72/v79 are PRESENT, not absent.** The plan assumed absence from "no CSV column + no registry entry"; direct derivation found a live send site in all four:
    | version | export | opcode | fields |
    |---|---|---|---|
    | gms_v48 | `0x6a5462` | 80 | **5 — no `fieldKey`** |
    | gms_v61 | `0x7aa1e3` | 93 | 6 |
    | gms_v72 | `0x864562` | 100 | 6 |
    | gms_v79 | `0x8afc42` | 99 | 6 |
  - Controller re-derived gms_v48 independently and **confirms the 5-field shape**: `COutPacket(v47, 80)` then `EncodeStr` (portalName), `Encode2` ×4 (x, y, targetX, targetY) — no `Encode1` before the string.
  - Consequence: **Task 1's "no `MajorAtLeast` gate required" ruling is superseded if v48 is in scope** — `fieldKey` would need a gate at the v48/v61 boundary. Tasks 3, 10, 11, 12 all change scope.
  - All four versions already have `docs/packets/registry/gms_vNN.yaml` and `services/atlas-configurations/seed-data/templates/template_gms_NN_1.json` — they are first-class supported versions; only the op is undeclared.
  - **Escalated to the user for a scope ruling.** Tasks 4/5/6/7 are version-agnostic and proceed regardless.
  - **RULING (user): all ten versions in scope.** Written to `docs/tasks/task-250-inner-portal-registration/scope-amendment.md`, commit `7402ec7f8`. That file supersedes the version lists in plan.md/design.md/prd.md and MUST be read by every remaining plan task. gms_v12 is the only confirmed absence.
  - **Task 2b dispatched** (controller-created follow-on, no plan section): structure docs for v48/v61/v72/v79, IDB renames, the `MajorAtLeast` gate boundary for `fieldKey`, and the manifest/version-coverage updates. Registry op rows deliberately left to Task 11.
- Task 7: agent ended its turn waiting on a background test instead of finishing; resumed with instructions to complete in the foreground.

- **Task 7: complete** (commit `b1ddb4db8`). `atlas-character` `Move` now routes on `fh == 0` to `UpdatePosition` (preserves stored foothold) vs `Update` (overwrites). Three tests, RED→GREEN evidence in the report. Module-local `go build ./... && go test ./...` exit 0. **Review: APPROVED** (0 blocking, 2 non-blocking style notes) — `review-task-7.md`. Reviewer traced the seam into its one real producer (atlas-channel `movement/processor.go` `ForCharacter` + `foldMovementSummary`) and confirmed no existing publisher relies on `fh == 0` meaning "foothold zero"; independently verified the "UpdatePosition had no callers" premise against the pre-commit tree.
- **Task 2b: complete** (commit `01b9bbf45`). All four new versions re-derived live:
  | version | export | opcode | fields | fieldKey |
  |---|---|---|---|---|
  | gms_v48 | `0x6a5462` | 80 | 5 | absent |
  | gms_v61 | `0x7aa1e3` | 93 | 6 | present |
  | gms_v72 | `0x864562` | 100 | 6 | present |
  | gms_v79 | `0x8afc42` | 99 | 6 | present |
  - **GATE BOUNDARY SETTLED — `MajorAtLeast(61)` for `fieldKey`.** Both sides decompiled in the same pass. Source of truth: `structures/gms_v61.md` §Gate decision, repeated in `gms_v48.md`. The other five fields are ungated across all ten versions. **Task 3 takes the gate from there, not from Task 1's superseded no-gate ruling.**
  - All four functions were already named `?TryRegisterTeleport@CUserLocal@@...` in their IDBs; the rename step was a no-op and no `idb_save` was needed.
  - The four registries confirmed NOT to declare the op. Registries deliberately untouched — **Task 11 must add the op row (opcode + `fname`) to `gms_v48/61/72/79.yaml`**, not just the `packet:` declaration.
  - `coverage-manifest.yaml` now lists ten versions with the gate recorded.
- Gate `b0cfe59e4..b1ddb4db8` (`--quick --base b0cfe59e4`): **PASS** (exit 0). Covers `8580d1042`, `3b6d4c08a`, `7402ec7f8`, `b1ddb4db8`. **Last gated commit = `b1ddb4db8`.**

## Resume point

Plan tasks complete: **1, 2, 2b, 7**. Remaining: **3, 4, 5, 6, 8, 9, 10, 11, 12**.

Next task to dispatch: **Task 3** (`InnerPortal` codec in `libs/atlas-packet`) — now unblocked. Its brief at `.superpowers/sdd/plan/task-3-brief.md` predates the scope ruling, so the dispatch MUST tell the implementer to read `scope-amendment.md` and take the gate from `structures/gms_v61.md` (`MajorAtLeast(61)`), not the brief's inherited "no gate" assumption.

Tasks 4, 5, 6 are independent of Task 3 and of each other; 8 needs 4+5+6; 9 needs 8; 10/11 independent; 12 last.

Two open items to reconcile in the next session:
1. ~~Gate `b0cfe59e4..b1ddb4db8`~~ — reconciled: **PASS**. Next gate uses `--base b1ddb4db8`.
2. ~~Task 7 review~~ — reconciled: **APPROVED**, no action needed.

**No open items. Next session starts clean at Task 3.**

Open non-blocking finding carried forward: design.md:88's jms_v185 `COutPacket` ctor address (`0xa2230e`) disagrees with the derived `0xa22313`; fold a correction into a later docs commit.

---

## Session 2 (resumed at Task 3)

- Task 3 brief amended by the controller with a `## SCOPE AMENDMENT` section (ten versions, `fieldKey` gated at `(GMS && MajorAtLeast(61)) || JMS`, gms_v48 five-field golden fixture). Briefs 4/5/6 verified to carry `### Files`.
- Task 3: dispatched (`atlas-implementer`, sonnet).
- **Task 3: complete** (commit `122fe88c1`). `InnerPortal` codec + tests in `libs/atlas-packet/portal/serverbound/`. Ten-row golden-byte table (gms_v48 12-byte five-field, other nine 13-byte six-field) + `pt.Variants` round-trip. Module-local build/test exit 0. **Review: APPROVED** (0 blocking, 0 non-blocking) — `review-task-3.md`; reviewer independently confirmed the gate predicate uses the `MajorAtLeast` idiom and matches the v48/v61 structure docs.
  - Gate `b1ddb4db8..122fe88c1` (`--quick --base b1ddb4db8`) launched.
- Task 4: dispatched (`atlas-implementer`, sonnet) — portal whole-list request + cache + `GetInMapByName` in atlas-channel.
- **Task 4: implemented** (commits `8da96de49` + `1182985b8`). Portal whole-list request, cache in `data/portal/processor.go`, `data/portal/model.go` accessors, `GetInMapByName` rewired onto the cache, `processor_test.go` added. Module-local build/test exit 0.
  - **Controller ruling:** the implementer's flagged concern (`requestInMapByName`/`portalsByName` left dead after the rewire) was ruled a removal, not a keep — this repo does not land unreferenced request helpers. Removed in `1182985b8` as a separate commit.
  - Review dispatched over `122fe88c1..1182985b8`.
- Task 5: dispatched (`atlas-implementer`, sonnet) — `position` registry + session destroy-hook clearing.
  - **Task 4 review: APPROVED** (0 blocking, 0 non-blocking, 0 not-evaluable) — `review-task-4.md`. Reviewer confirmed the cache mirrors `data/map/processor.go:34-73` (tenant-scoped `cacheKey{tenantId, mapId}`, double-checked load), `GetInMapByName` returns a descriptive error with no second fetch on a filter miss, tests pin cache hit/miss/tenant-isolation via call counters, and a repo-wide grep shows zero remaining references to the removed helpers. **Task 4: complete.**
- **Task 5: implemented** (commit `d5433181a`). `position/registry.go` + `registry_test.go` (new), session destroy clears the entry, `session/position_hook_test.go` (new). Module-local build/test exit 0. Implementer notes `position.GetRegistry().Put` has no production caller yet **by design — Task 6 wires the writer**; it is not a dead helper.
  - Review dispatched over `1182985b8..d5433181a`.
- Task 6: dispatched (`atlas-implementer`, sonnet) — `TeleportCharacter` on `movement/processor.go` + registry write in `ForCharacter`.
  - **Task 5 review: APPROVED** (0 blocking, 0 non-blocking) — `review-task-5.md`. Registry is tenant-scoped via `Key{Tenant, CharacterId}` across `Put`/`Lookup`/`Clear` with a cross-tenant isolation test; `sync.RWMutex` + `sync.Once` mirrors `character/chakra/registry.go`; `go test -race` clean. Reviewer traced `clearLastPositionOnDestroy` to the single `ProcessorImpl.Destroy` funnel (`session/processor.go:422,474-478`) — every handler and kafka-consumer destroy path routes through it, so no leak path. **Task 5: complete.**
- **Task 6: implemented** (commit `a1fa2fcfb`). `movement.TeleportCharacter` (interface + impl + mock) and the last-position registry write in `ForCharacter`. Module-local build/test/gofmt clean. `TeleportCharacter` has no production caller yet — the inner-portal handler task consumes it.
  - Review dispatched over `d5433181a..a1fa2fcfb`, with a directed check on the cross-service seam into Task 7's `fh == 0` → `UpdatePosition` branch in atlas-character.

- **GATE `b1ddb4db8..122fe88c1` (gate-3): ERROR, not FAIL — discard its verdict.** It reported `lint.sh: LINT FAIL — services/atlas-channel/atlas.com/channel`, `undefined: position` at `movement/processor.go:84,95`. That is the Task 6 working tree mid-edit: the gate was launched for the Task 3 range but `tools/lint.sh` lints the **working tree**, not the commit range, and the Task 6 implementer was concurrently editing that file before adding the `"atlas-channel/position"` import. Import confirmed present at `movement/processor.go:12` after `a1fa2fcfb`.
  - **Operational lesson for the rest of this plan: never run the gate concurrently with an implementer that touches any module in the gate's fan-out.** The lint and format guards read the tree, so a concurrent edit produces a spurious FAIL. Read-only reviewers are safe to overlap; implementers are not.
  - Relaunched as gate-3-6 covering `b1ddb4db8..a1fa2fcfb` (`--quick --base b1ddb4db8`, log at `<scratchpad>/gate-3-6.log`) with no implementer running. **Unreconciled — the next session must read that log and record the verdict.** Last gated commit remains `b1ddb4db8` until it does.
  - **Task 6 review: APPROVED_WITH_FINDINGS** (0 blocking, 1 non-blocking) — `review-task-6.md`. Reviewer traced the seam by hand into atlas-character's `consumer/processor.go` and confirmed the emitted command drives Task 7's `fh == 0` → `UpdatePosition` branch correctly. **Task 6: complete.**
  - **REQUIRED FOLLOW-UP (next session, before the branch closes):** none of the three tests in `movement/teleport_test.go` assert the emitted Kafka command's `Fh=0` — they assert only the registry side-effect and the absence of an announce. This matches the brief's test table and the `movement/` package's existing convention (no test there asserts `CommandProducer` output), which is why it is non-blocking, but `Fh=0` is the safety-critical value for the Task 7 seam and CLAUDE.md requires a test asserting the NEW contract when a change crosses a service boundary. Add the wire-level assertion as a scoped commit. Do **not** run it concurrently with a gate.

## Resume point (session 2 end)

Plan tasks complete: **1, 2, 2b, 3, 4, 5, 6, 7**. Remaining: **8, 9, 10, 11, 12**.

Commits this session: `122fe88c1` (T3), `8da96de49` + `1182985b8` (T4), `d5433181a` (T5), `a1fa2fcfb` (T6). Branch HEAD: `a1fa2fcfb`.

Open items for the next session, in order:
1. **Reconcile gate-3-6** — read `<scratchpad>/gate-3-6.log` (range `b1ddb4db8..a1fa2fcfb`). If the log is truncated or the process died at `/clear`, relaunch `tools/verify.sh --quick --base b1ddb4db8` with no implementer running. Last gated commit is `b1ddb4db8` until this is reconciled.
2. **The `Fh=0` wire assertion** in `movement/teleport_test.go` (see the Task 6 follow-up above).
3. **Task 8** — needs 4+5+6, all now complete. `maxPortalEntryDistance = 81` comes verbatim from `structures/gms_v95.md` §Threshold derivation.
4. Then 9 (needs 8), 10, 11 (must add the op row — opcode + `fname` — to `gms_v48/61/72/79.yaml`, not just the `packet:` declaration), 12 last.

Every remaining task must read `scope-amendment.md`: **ten versions**, gms_v12 out of scope.

Still-open non-blocking finding carried forward: design.md:88's jms_v185 `COutPacket` ctor address (`0xa2230e`) disagrees with the derived `0xa22313`; fold a correction into a later docs commit.

---

## Session 3

- Gate-3-6's log from session 2 was confirmed truncated (killed at `/clear`, ended mid `go build/vet` fan-out). Verdict discarded, not inferred.
- **Gate `b1ddb4db8..a1fa2fcfb` relaunched** (`--quick --base b1ddb4db8`) with no implementer running. Log: `<scratchpad>/gate-3-6b.log`. Unreconciled at time of writing.
- Task 8 brief amended by the controller (`.superpowers/sdd/plan/task-8-brief.md`):
  - `## CONTROLLER AMENDMENTS` prepended — `maxPortalEntryDistance = 81` settled (source: `structures/gms_v95.md` §Threshold derivation), scope-amendment pointer, and a note that 4/5/6 are landed so signatures come from the tree.
  - `Step 4` appended — the carried-forward Task 6 follow-up (`Fh=0` wire assertion in `movement/teleport_test.go`), to be committed separately after the Task 8 commit.
- Task 8 dispatch deliberately held until the gate returns: `tools/lint.sh` reads the working tree, and Task 8 edits the same module the gate is linting. This is the session-2 operational lesson applied.
- **Gate `b1ddb4db8..a1fa2fcfb` (gate-3-6b): PASS** (exit 0, task id `bdzvlnq4w`). Covers `122fe88c1` (T3), `8da96de49`+`1182985b8` (T4), `d5433181a` (T5), `a1fa2fcfb` (T6). go-analyzer-guards PASS, skill/job-id clean, scope/producer-seam/env guards clean, lint & format 0 issues across 89 modules. **Last gated commit = `a1fa2fcfb`.**
  - Note: two task notifications arriving in this session (`bvwn6sgpf`, `b7zb3e9bp`) were stale re-reports of session-2 background gates, not this run. Verdict above is from `bdzvlnq4w`, the one this session launched.
- Task 8 dispatched (`atlas-implementer`, sonnet) — `portal.EnterInner` + the carried-forward `Fh=0` wire assertion as a separate commit.
- **Task 8: complete** (commits `b92ef4b37` + `bb38fa567`). `portal.EnterInner` (interface + impl + mock), `maxPortalEntryDistance = 81`, `processor_test.go` with `TestEnterInner` 10/10 subtests. Second commit closes the carried-forward Task 6 follow-up: `TestTeleportCharacter_EmitsFhZeroOnWire` in `movement/teleport_test.go` asserts `Fh=0` on the emitted command. Module-local build/test exit 0, no concerns raised.
  - **The Task 6 REQUIRED FOLLOW-UP is now CLOSED** (`bb38fa567`).
  - Review dispatched over `a1fa2fcfb..bb38fa567`.
- **Task 8 review: APPROVED_WITH_FINDINGS — 1 BLOCKING** — `review-task-8.md`. Everything the directed checks asked for verified with file:line evidence (81 + its derivation citation, refusals warn+nil, adopted coords always `dp.X()/dp.Y()`, int32 widening, tenant-scoped lookup, and `Fh==0` traced by hand into atlas-character's consumer).
  - **Blocking:** `portal/processor_test.go:155-158` — the "last-position registry miss" row never produces an actual miss. The harness fallback at `processor_test.go:192-196` `Put`s (100,200) for every row that omits `seedRegistry`, and that row omits it. The code's miss handling (`portal/processor.go:131-139`) is correct, but design §5.5 ("a miss must never refuse") is pinned by no test — the row would still pass if the branch were deleted or inverted.
  - Fix held until Task 9's implementer finishes; two implementers must not edit atlas-channel concurrently.
- Task 9: implementer ended its turn on a backgrounded build (the same failure mode Task 7 hit). Resumed with instructions to run build/test in the foreground and finish in-turn.
- **Task 9: complete** (commit `daa49c479`). `socket/handler/portal_inner.go` (new) + `handlerMap` registration in `main.go`. `InnerPortalHandle` already existed from Task 3, so the "add the constant if missing" fallback was not triggered. Module-local build/test: 2162 passed across 310 packages, exit 0.
  - Review dispatched over `bb38fa567..daa49c479`.
- Task 8 fix round 1 dispatched (resumed the Task 8 implementer) for the blocking test finding, with a required RED-evidence step: the miss row must fail if the miss branch is inverted.
- Task 10 brief amended: **ten templates, not six.** Added `template_gms_48/61/72/79_1.json` with opcodes 0x50/0x5D/0x64/0x63, each to be confirmed against its `structures/gms_vNN.md` before writing. `template_gms_12_1.json` stays unrouted (confirmed absent).
  - **Task 9 review: APPROVED** (0 blocking, 0 non-blocking) — `review-task-9.md`. Reviewer independently verified the argument mapping against the real `InnerPortal` accessors and the real `EnterInner` signature (no x/y ↔ targetX/targetY transposition), confirmed `main.go:902` uses the exact constant `Operation()` returns with no collision across the 134 handlerMap entries, and confirmed identity comes from session/ctx only.
- **Task 8 blocking finding CLOSED** (commit `db5744980`, test-only). The registry-miss subtest now produces a genuine miss. RED evidence recorded: with the `ok`/`!ok` branch inverted in `processor.go`, both `last-position_registry_miss` and `last_known_position_beyond_threshold` failed; branch restored and `processor.go` confirmed byte-identical to the prior commit. Build/test 2162 passed / 310 packages. **Task 8: complete.**
- Task 10 dispatched (`atlas-implementer`, sonnet) — ten seed templates, opcode confirmed per version against `structures/`.
- Task 11 dispatched (`atlas-implementer`, sonnet) — `candidatesFromFName` case + registry links; the four new registries get a FULL op row (opcode + fname), not just `packet:`.
  - The two run concurrently on disjoint file sets (templates JSON vs registry YAML + `tools/packet-audit`); each was told to stay out of the other's files.
  - Task 11 brief amended with the ten-version table and the "full row, not just packet:" distinction.
- **Task 11: implemented** (commit `bb7ec8dbd`), DONE_WITH_CONCERNS. `candidatesFromFName` case + registry links; all 10 registry YAMLs parse; `go build/test ./tools/packet-audit/...` 13/13 OK; `matrix --check` shows no dangling/orphan/conflict line mentioning `PortalInnerPortal`.
  - **REPO-WIDE HAZARD, worth carrying beyond this task:** every task worktree shares one `refs/stash` stack with the main `.git`. A concurrent agent's `git stash pop` clobbered this agent's stashed edit pass mid-task. It redid the edits and line-count-verified the diff before committing without stashing. **Agents must not use `git stash` in a worktree while another agent is active.** Final committed state is intact.
  - Concern 2 (carried to Task 12): `docs/packets/audits/STATUS.md` / `status.json` are now stale — any registry edit triggers this. The Task 11 brief's `### Files` did not include regeneration, so it was correctly left alone. **Task 12 must regenerate the matrix.**
  - Review dispatched over `db5744980..bb7ec8dbd`.
- **Task 10: implemented** (commit `836b773b2`). Ten seed templates route the op; `template_gms_12_1.json` untouched. `template-opcode-order-guard.sh` and `template-duplicate-binding-guard.sh` pass. `template-symbol-check.sh` fails on 5 of the 10 with a `ChatGeneralChat` dangling-symbol error the implementer verified pre-existing against `HEAD~1` — reviewer asked to re-confirm that independently.
  - **The stash hazard hit BOTH concurrent implementers.** Task 10's first pass of all ten edits was silently discarded by Task 11's `git stash pop`, and vice versa; each detected it, redid its edits, and confirmed `git diff --stat` before committing. Both reviewers were given an explicit directed check to prove the committed diffs are complete rather than accepting the redo on faith.
  - **Standing rule for this repo, not just this task: agents must never use `git stash` in a task worktree.** All worktrees share one `refs/stash` stack with the main `.git`, so concurrent agents clobber each other silently. To probe whether a failure is pre-existing, check out the parent commit into a scratch path or use `git show HEAD~1:<path>` — never stash.
  - Review dispatched over `bb7ec8dbd..836b773b2`.
- **Gate `a1fa2fcfb..836b773b2` launched** (`--quick --base a1fa2fcfb`, log `<scratchpad>/gate-8-11.log`), covering Tasks 8, 8-fix, 9, 11, 10. No implementer running. **Unreconciled.**
  - **Task 11 review: APPROVED** (0 blocking, 0 non-blocking, 1 not-evaluable) — `review-task-11.md`. Every one of the ten opcodes re-derived from the structure docs' `COutPacket` ctor citations rather than taken from the brief table; all four new rows complete; no `gms_v12.yaml` exists so the absence is structural. Stash-recovery claim independently checked: `git show --stat` shows exactly 11 files with per-file insertion counts consistent throughout — no half-applied block, no stray file. `qualifiedWriterName("portal","InnerPortal")` = `PortalInnerPortal` matches all ten `packet:` values. **Task 11: complete.**
  - Not-evaluable (carried, low risk): no true pre-edit `matrix --check` baseline was captured by either the implementer or the reviewer (both hit a 2-minute timeout). Post-edit output is confirmed clean of InnerPortal-related dangling/orphan/conflict lines, but no before/after diff of the full command exists. Task 12 regenerates the matrix anyway, which subsumes this.
  - **Task 10 review: APPROVED** (0 blocking, 0 non-blocking, 0 not-evaluable) — `review-task-10.md`. All ten opcodes re-derived from the structure docs (not the brief table) and all match; `template_gms_12_1.json` genuinely untouched; handle-constant chain confirmed end to end (`main.go:902` ↔ `inner_portal.go:14,56` ↔ the literal in all ten templates); opcode-order and duplicate-binding guards exit 0 across all 22 templates. The 5-template `ChatGeneralChat` symbol-check failure was independently reproduced AND proven pre-existing by diffing against parent `db5744980`. Stash narrative not taken on faith: final commit is exactly ten files, +90/-0, one `InnerPortalHandle` each, no strays. **Task 10: complete.**
- **Gate `a1fa2fcfb..836b773b2` (gate-8-11): PASS** (exit 0, task id `bdphfoksk`). Covers `b92ef4b37`, `bb38fa567` (T8), `daa49c479` (T9), `db5744980` (T8 fix), `bb7ec8dbd` (T11), `836b773b2` (T10). **Last gated commit = `836b773b2`.**

## Resume point (session 3 end)

Plan tasks complete: **1, 2, 2b, 3, 4, 5, 6, 7, 8, 9, 10, 11** — all reviewed, all APPROVED (Task 8's one blocking finding fixed and closed in `db5744980`). Remaining: **Task 12 only.**

Commits this session: `b92ef4b37`, `bb38fa567` (T8), `daa49c479` (T9), `db5744980` (T8 fix), `bb7ec8dbd` (T11), `836b773b2` (T10). Branch HEAD: `836b773b2`. Last gated commit: `836b773b2`. **No open items, no unreconciled gates, no outstanding reviews.**

Task 12 must:
1. Read `scope-amendment.md` — ten versions, `gms_v12` the only confirmed absence.
2. **Regenerate `docs/packets/audits/STATUS.md` / `status.json`** — they are stale because Task 11 edited the registries. This is a known, expected consequence, deliberately left to this task.
3. Fold in the still-open non-blocking docs correction carried since Task 1: `design.md:88` gives the jms_v185 `COutPacket` ctor as `0xa2230e`; the derived address is `0xa22313`. The opcode (96/0x060) is unaffected.

After Task 12: run `superpowers:finishing-a-development-branch` (the flagless `tools/verify.sh` must exit 0 — only that counts as verified), then `superpowers:requesting-code-review`.

**Standing repo hazard discovered this session:** all task worktrees share one `refs/stash` stack with the main `.git`. Two concurrent implementers clobbered each other's stashes this session. Agents must not use `git stash` in a worktree; probe pre-existing failures with `git show HEAD~1:<path>` instead.

---

## Session 4

- Resumed at Task 12 (the only remaining plan task). Actual branch HEAD is `bb7ec8dbd` (T11), which sits *above* `836b773b2` (T10) — session 3's "Branch HEAD: 836b773b2" line had the two commits in the wrong order. Gate `bdphfoksk` covered both, so **last gated commit = `bb7ec8dbd`**; the next gate uses `--base bb7ec8dbd`.
- Working-tree note: `services/atlas-ui/src/pages/*.tsx` (11 files, a `useGridRefresh`/`lastUpdatedAt` change) carry uncommitted edits **unrelated to task-250**. Left untouched; implementers told not to commit, revert or stash them. Flagged to the user.
- Task 12 brief amended (`.superpowers/sdd/plan/task-12-brief.md`, `## CONTROLLER AMENDMENTS`): ten versions not six; the derived export-address + IDB-session + opcode table for all ten (so the implementer does not re-derive or guess); gms_v48 five-field note; the carried `design.md:88` jms_v185 ctor correction (`0xa2230e` → `0xa22313`); matrix-regeneration expectations; the no-`git stash` hazard; the atlas-ui working-tree carve-out.
  - gms_v84's opcode cell deliberately says "take from `structures/gms_v84.md` / registry" rather than a number — the controller did not have it derived and does not invent one.
- Task 12 dispatched (`atlas-implementer`, sonnet).
- **Task 12: implemented** (commit `25a72cdbc`), DONE_WITH_CONCERNS. Ten export splices, ten evidence pins, ten `packet-audit:verify` markers, matrix regenerated (implementer reports all 10 USE_INNER_PORTAL cells ✅), plus the carried `design.md:88` ctor correction. `libs/atlas-packet` build/test clean.
  - **Concern 1 (open, for the reviewer):** `matrix --check` exits 1. Implementer attributes it entirely to pre-existing `decompile hash drift` on ~30 unrelated packets (character/messenger/pet/summon/…), claiming zero `portal` hits and no `status.json` change to those entries. **Not accepted on faith** — directed check 5 of the Task 12 review verifies it independently. If real, it is a pre-existing repo condition to scope separately, not a task-250 defect.
  - **Concern 2:** the brief's `--ida-database <session hash>` flag form was rejected by the live IDA-MCP server (no `idb_list`, no tool taking a `database` param). The implementer substituted `--ida-url http://…/mcp` per version and claims it confirmed targeting via `survey_binary` module names plus address agreement with the amendment table. Directed check 6 re-verifies. **The `--ida-database` instruction in the brief was the controller's, carried from the plan; if the flag genuinely does not exist the plan text is wrong and should be corrected before the PR.**
  - Gate `bb7ec8dbd..25a72cdbc` launched (`--quick --base bb7ec8dbd`, log `<scratchpad>/gate-12.log`). No implementer running. **Unreconciled.**
  - Review dispatched over `bb7ec8dbd..25a72cdbc` (`atlas-reviewer`, sonnet) with 8 directed checks.
- **Task 12 review: CHANGES_REQUIRED — 1 BLOCKING** — `review-task-12.md`. Both of the implementer's concerns were investigated; one of them inverted.
  - **BLOCKING: the splice stripped `note`/`notes`/`region`/`_note`/`discriminator` fields from hundreds of unrelated pre-existing entries** across all ten export JSONs (19–256 entries per file; e.g. gms_v92 `CUserLocal::OnIncComboResponse` lost its hand-traced task-217 decompile-failure note). Real data loss of curated IDA provenance.
  - **Concern 1 is REFUTED, not confirmed.** The `matrix --check` drift is *not* pre-existing: the reviewer ran `matrix --check` against `bb7ec8dbd` in a throwaway detached worktree — parent emits **5 lines, zero drift**; `25a72cdbc` emits **1124 lines** of `decompile hash drift` across ~30 unrelated packets. The drift is a direct downstream consequence of this commit's own export edits. The report's "pre-existing, not introduced here" premise was never checked against the parent and is false. *Lesson: the implementer's diagnosis asserted the one thing the brief explicitly told it to confirm; the directed check is what caught it.*
  - The one part of the report's claim that held: no drifted line mentions `portal`.
  - Everything else in the commit verified clean — ten markers with addresses matching the derived table, gms_v12 correctly absent, evidence records tool-written with real `--verifies` targets, all ten cells ✅, no `services/` or codec/registry/template files touched, `design.md:88` correction applied.
  - Not-evaluable ×2 (carried): only 3 of 10 `--ida-url` splice addresses were independently spot-checked beyond the amendment table; and whether the field-stripping is a `packet-audit export --splice` tool defect or specific to these invocations was not root-caused.
- Fix-round-1 brief written: `.superpowers/sdd/plan/task-12-fix1-brief.md`. Restore each export to **parent content + exactly the one new entry** (mechanically, NOT by re-running `--splice`), prove it by key-level diff rather than line count, confirm `matrix --check` returns to the parent's ~5-line shape, and bounded Step 5: diagnose the splice's lossy JSON round-trip and fix it only if the cause is contained (likely a struct missing those fields), as a separate commit.
- Fix dispatch **held** until gate-12 returns — `tools/lint.sh` reads the working tree and the fix edits files in the gate's fan-out (session-2 operational lesson).
- **Gate `bb7ec8dbd..25a72cdbc` (gate-12): PASS** (exit 0, task id `b3ddoined`). 89 modules build/vet, go analyzer guards, skill/job-id, scope guard, lint & format (89 modules + atlas-ui) all clean. **Last gated commit = `25a72cdbc`.**
  - **Note the gate's blind spot:** `--quick` does not run `packet-audit matrix --check`, so gate-12 passed a commit that the review independently proved introduces ~1120 matrix-drift failures. A green `--quick` gate is not evidence about matrix consistency; the final flagless `tools/verify.sh` and the explicit `matrix --check` in Task 12's final-verification checklist are the checks that cover it.
- Task 12 fix round 1 dispatched (fresh `atlas-implementer`, sonnet) after gate-12 returned. Dispatched **fresh rather than resuming** the Task 12 implementer: its report asserts the false "pre-existing drift" premise, and the fix's first job is to disprove that premise — a fresh context avoids anchoring on it. It writes into the same `task-12-report.md` (persistent task memory) and was told explicitly not to inherit that premise.
- **atlas-ui working-tree edits DISCARDED** (user ruling: "discard the ui edits if they're not yours"). Confirmed not task-250's before discarding: `git log 5f299e4bb..HEAD -- services/atlas-ui/` is empty — no commit on this branch touched atlas-ui.
  - They were also incomplete on their own terms: `lastUpdatedAt` appears **only** in the 11 modified page files; neither `useGridRefresh` nor the grid component it is passed to declares it anywhere in `services/atlas-ui/src`. An abandoned half-change that would not typecheck.
  - Backed up before discarding to `<scratchpad>/discarded-atlas-ui-lastUpdatedAt.patch` (247 lines) rather than destroyed outright. Note the scratchpad is session-scoped and will not survive indefinitely.
  - `git checkout -- services/atlas-ui/`; tree now clean of them. The final flagless `tools/verify.sh` no longer sees them.
- **Task 12 fix round 1: implementer killed mid-work by an API session limit, before any commit.** All its work was sitting uncommitted. Controller assessed the tree directly rather than re-dispatching blind:
  - **Ten exports correctly restored — verified by key-level diff, not line count.** For all ten files vs parent `bb7ec8dbd`: exactly one key added (`CUserLocal::TryRegisterTeleport`), 0 removed, 0 existing entries changed, 0 top-level changes. The stripped provenance is back.
  - All ten spliced addresses re-checked against the derived table: **10/10 match.**
  - **`matrix --check` now exits 0 with 2 lines** (two `n-a evidence consumed` notes) — against 1124 drift lines at `25a72cdbc`, and cleaner than parent's 5 (parent's extra 3 were stale-STATUS warnings, now regenerated away). **The blocking regression is fixed.**
  - `STATUS.md:615` shows all ten `USE_INNER_PORTAL` cells ✅.
  - Stray untracked `tools/packet-audit/docs/{packets/audits/STATUS.md,status.json}` — an artifact of a tool run from the wrong cwd — deleted by the controller. Must never be committed.
  - **Root cause found and it is contained:** `exportFn` in `tools/packet-audit/internal/idasrc/export.go` had no fields for `note`/`region`/`_note`, and `Selector.Discriminator` in `extract.go` carried `omitempty`, so the splice's unmarshal→marshal round trip silently dropped all four on every entry it rewrote. This is a **general `packet-audit export --splice` defect**, not a task-250 one — it would have silently eaten provenance on the next packet task too.
- Continuation dispatched (fresh `atlas-implementer`, sonnet) with `.superpowers/sdd/plan/task-12-fix1-cont-brief.md`: write the round-trip test, rule on Step 2, verify, and land two commits (A = exports + regenerated matrix, B = tool fix + test).
  - **Step 2 is a real open judgement, flagged rather than waved through:** dropping `omitempty` from `Selector.Discriminator` makes the tool emit `"discriminator": ""` on *every* selector it ever writes, including entries that legitimately omit it — and the ten restored exports do not exercise it, because they were restored mechanically rather than through the tool. The implementer must rule on the merits (narrow fix vs. a `RawMessage`/custom-marshal passthrough) and record its reasoning, with `matrix --check` still exit 0 afterwards.
- **Task 12 fix round 1: complete** (commits `3340ece91` exports+matrix, `3df557498` tool fix+test). `tools/packet-audit` 13/13 packages pass; `matrix --check`, `fname-doc --check`, `operations --check` all exit 0. Working tree clean apart from untracked task docs.
  - **The Step 2 judgement went the other way, and the concern was real.** The implementer rejected the prior agent's bare `omitempty` removal: a grep sweep found **~65 existing dispatch selectors that legitimately omit `discriminator`**, plus a live `WriteDispatch`/`infer.go` path that marshals `Selector` directly for `Default`/`Guard`-only arms — the bare removal would have rewritten all of them. Replaced with a custom `MarshalJSON`/`UnmarshalJSON` on `Selector` that records whether the source JSON carried an explicit `discriminator` key and reproduces that exact presence on remarshal. Contained to `extract.go` + the new test.
  - Gate `25a72cdbc..HEAD` launched (`--quick --base 25a72cdbc`, log `<scratchpad>/gate-12fix.log`). **Unreconciled.**
  - Review dispatched over `25a72cdbc..HEAD` (`atlas-reviewer`, sonnet), 7 directed checks — including an independent key-level re-diff of all ten exports against `bb7ec8dbd` (the controller's own check is not treated as sufficient for closing a blocking finding) and scrutiny of the custom marshaller, which is the highest-risk part of the fix.
- **Task 12 fix round 1 review: CHANGES_REQUIRED — 1 blocking, 1 non-blocking** — `review-task-12-fix1.md`.
  - **The original blocking data-loss finding is CONFIRMED CLOSED** by the reviewer's own key-level diff of all ten exports against `bb7ec8dbd`, independent of the controller's. The `Selector` custom marshaller correctly reproduces both discriminator presence states (verified at runtime).
  - **New blocking, an ordering artifact:** commit A (`3340ece91`) regenerated `STATUS.md`/`status.json` against the tool as it then stood; commit B (`3df557498`) then changed `export.go`/`extract.go` and nothing regenerated them again. The committed `Tool:`/`toolSha` (`a645319789…`) is stale against the current tool hash (`83326298d5…`), so **`matrix --check` exits 1 at HEAD** with "STATUS.md is stale". Data content — including all ten `USE_INNER_PORTAL` cells — is unaffected. Fix is a regenerate + commit.
    - *Lesson: `matrix --check` was genuinely exit 0 when both the controller and the implementer ran it, and false by the time the commits landed in that order. Any artifact carrying a fingerprint of the tool must be regenerated **after** the last tool change, and the check re-run at final HEAD — not at the moment the work was done.* Fix-round-2 brief Step 4 makes that explicit.
  - Non-blocking: `TestSpliceExportPreservesCuratedProvenanceKeys` covers only the explicit `"discriminator": ""` state, not the omitted-key state that ~65 real selectors use. Shipped code handles it (reviewer confirmed at runtime); it is just unpinned.
- Fix-round-2 brief written: `.superpowers/sdd/plan/task-12-fix2-brief.md` (regenerate + commit; add the omitted-discriminator test case asserting key *presence*, not value equality — value-only comparison is exactly how the original data loss escaped).
- Fix-round-2 dispatch **held** until gate-12fix returns (`tools/lint.sh` reads the working tree; the fix edits Go in the gate's fan-out).
- **Gate `25a72cdbc..3df557498` (gate-12fix): PASS** (exit 0, 56-line log, no FAIL lines). **Last gated commit = `3df557498`.** Note again that `--quick` does not run `matrix --check` — it did not and could not catch the stale-fingerprint finding above.
- Task 12 fix round 2 dispatched (`atlas-implementer`, sonnet) after the gate returned.
- **Task 12 fix round 2: complete** (commits `9306fa4cc`, `a175e81d7`). `TestSpliceExportPreservesOmittedDiscriminator` added; matrix regenerated. **Controller independently re-ran `matrix --check` at HEAD `a175e81d7`: exit 0, 2 lines, tree clean** — this claim had been wrong twice, so it was not accepted on report.
  - **Implementer's concern, worth carrying beyond task-250:** `toolTreeSHA()` hashes `HEAD`, which does not include the commit being created — so **regenerating the matrix in the same commit as a `.go` change reproduces the staleness bug**. That is what happened with `9306fa4cc` and forced the follow-up `a175e81d7`. Rule for future packet tasks: **commit `.go` changes first, then regenerate and commit the matrix as a final, separate commit.** The fix-round-2 review judges whether this diagnosis is correct.
- Gate `3df557498..a175e81d7` (gate-12fix2) launched and reconciled below.
- Review dispatched over `3df557498..HEAD` (`atlas-reviewer`, sonnet), scoped small — five directed checks, no re-audit of earlier task-12 work.
- **Gate `3df557498..a175e81d7` (gate-12fix2): PASS** (exit 0, no FAIL lines). **Last gated commit = `a175e81d7`.**

## Resume point (session 4 end)

**All 12 plan tasks complete and reviewed.** Branch HEAD: `a175e81d7`. Last gated commit: `a175e81d7`.

Commits this session: `25a72cdbc` (T12), `3340ece91` + `3df557498` (T12 fix 1), `9306fa4cc` + `a175e81d7` (T12 fix 2).

**Task 12 fix-round-2 review: APPROVED** (0 blocking, 0 non-blocking, 0 not-evaluable) — `review-task-12-fix2.md`. Reviewer ran all three `--check` commands itself at final HEAD (exit 0), diffed `STATUS.md`/`status.json` across the range and confirmed **only** the `Tool:`/`toolSha` line changed with all ten `USE_INNER_PORTAL` cells still ✅, and proved the new test genuinely goes RED (forced `extract.go:60` to always marshal an explicit discriminator, observed failure, restored, re-confirmed GREEN and a clean tree). It also verified the `toolTreeSHA()` diagnosis against `tools/packet-audit/cmd/matrix.go:487-496` (`git ls-tree -r HEAD`) — **the diagnosis is correct**, and it is a process trap, not a defect in this fix.

**NO OPEN ITEMS. All 12 plan tasks complete, all reviewed, all APPROVED. Every gate reconciled.** The branch is ready for close-out.

Then, in order:
1. `superpowers:finishing-a-development-branch` — **the flagless `tools/verify.sh` must exit 0.** Only the flagless run counts (it bakes and runs `-race`); every gate this branch has seen was `--quick`, which skips both AND does not run `matrix --check`.
2. `superpowers:requesting-code-review` — `plan-adherence-reviewer` (12 tasks + 1 controller-created Task 2b, no sharding needed at this size), `backend-guidelines-reviewer` over the changed Go in `services/atlas-channel`, `services/atlas-character`, `libs/atlas-packet` and now `tools/packet-audit`, and `packet-completeness-critic` against `coverage-manifest.yaml`. `frontend_review` is **no longer applicable** — the atlas-ui edits were discarded and no task-250 commit touches atlas-ui.
3. Then open the PR.

Notes the close-out must carry:
- **A general `packet-audit` tool defect was found and fixed on this branch** (`3df557498`, `9306fa4cc`): `SpliceExport`'s unmarshal→marshal round trip silently dropped `note`/`notes`/`region`/`_note` and the explicit `discriminator` key from every entry it rewrote. It destroyed curated task-217/task-096 IDA provenance across ~1000 entries in ten export files before being caught. This is **not** task-250 domain work and reviewers should expect it in the diff; it belongs here because task-250 is what surfaced it and the branch could not be green without it.
- **`toolTreeSHA()` hashes `HEAD`**, so a matrix regenerate committed together with a `.go` change is stale on arrival. Commit `.go` first, regenerate and commit the matrix last. This cost two extra commits this session and is a trap for the next packet task.
- Untracked task docs under `docs/tasks/task-250-inner-portal-registration/` (progress.md, agent-ledger.tsv, review-task-*.md) are **still uncommitted** — the close-out should commit them with the branch.
- The 11 unrelated `services/atlas-ui/src/pages/*.tsx` edits were discarded on the user's ruling; backup patch at `<scratchpad>/discarded-atlas-ui-lastUpdatedAt.patch`, which is session-scoped and will not survive.
