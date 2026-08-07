## Frontend guidelines

- **Scope:** `services/atlas-ui/src/components/features/monsters/MonsterSkillChip.tsx` (only atlas-ui file touched by this branch; diff is 6 lines in `summarizeEffect`).
- **Guidelines Source:** `.claude/skills/frontend-dev-guidelines/SKILL.md` + resources.
- **Build:** Not re-run per instructions (pre-verified: `npm run build` exits 0, `npx vitest run` 1376/1376 passed).

### Diff under audit

```diff
-  if (a.duration > 0) rows.push({ label: "Duration", value: `${a.duration}s` });
+  // `duration` is MILLISECONDS from atlas-data (task-190 FR-1.1 — mobskill
+  // reader.go is the single seconds→ms conversion point). Render as seconds.
+  if (a.duration > 0)
+    rows.push({
+      label: "Duration",
+      value: `${(a.duration / 1000).toLocaleString()}s`,
+    });
```
(`services/atlas-ui/src/components/features/monsters/MonsterSkillChip.tsx:116-120`)

### FE-* checklist (applicable items only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | No `: any` / `as any` in `MonsterSkillChip.tsx` (grep clean). |
| FE-02 | No manual class concatenation | PASS | All `className` values are static string literals (e.g. `MonsterSkillChip.tsx:99` `className="space-y-0.5 text-xs"`); no `+`/template concatenation. |
| FE-03 | No direct API client calls in components | PASS | No `@/lib/api/client` import; uses `mobSkillsService.getMobSkillDetail` (`MonsterSkillChip.tsx:13-15,37`). |
| FE-04 | No inline Zod schemas | PASS | No `z.object`/`z.string` in file. |
| FE-05 | No spinners for content loading | PASS | Loading state renders text `"Loading effect…"` (`MonsterSkillChip.tsx:79`), not `animate-spin`. |
| FE-06 | No hardcoded colors | PASS | Only semantic classes used in/near the diff (`text-muted-foreground`, `MonsterSkillChip.tsx:117` area); no `bg-white`/`text-gray-*` etc. |
| FE-07 | No state mutation | PASS | `rows` built via `.push` on a locally-created array then returned — not a mutation of external/component state (no `setState` involved). No `.sort`/`.splice` on props/state. |
| FE-08 | No default exports | PASS | `export function MonsterSkillChip(...)` (`MonsterSkillChip.tsx:22`); `summarizeEffect`/`SkillTooltipBody` are internal named helpers, not default-exported. |
| FE-12 | JSON:API model shape | PASS (unchanged) | `MobSkillDetailAttributes` is the `attributes` payload of `MobSkillDetailData { id, type, attributes }` in `services/atlas-ui/src/services/api/mob-skills.service.ts:12-34` — correct JSON:API shape. Not touched by this diff. |
| FE-16 | Schema paired with inferred type | N/A | No Zod schema involved in this change. |
| FE-17 | Tests exist for changed component | **FAIL** | No test file for the component exists anywhere in the tree: `find services/atlas-ui/src -iname "*MonsterSkillChip*"` returns only the source file, no `__tests__/MonsterSkillChip.test.tsx`. The diff changes a user-visible arithmetic/formatting rule (÷1000, `s` suffix) with no test asserting the rendered output for typical/zero/large `duration` values — exactly the class of bug this task exists to fix (see `bug_mob_disease_duration_seconds_as_ms.md`: a prior seconds/ms mixup already reached production undetected). |

### Correctness review (arithmetic / formatting / adjacent field)

- **Typical value** (e.g. `duration = 30000` ms, i.e. 30s WZ duration): `(30000/1000).toLocaleString()` → `"30"` → renders `"30s"`. Correct.
- **Zero**: guarded by `if (a.duration > 0)` (`MonsterSkillChip.tsx:116`), same guard style as `mp_con`/`hp`/`prop` rows — the row is simply omitted, consistent with sibling behavior. No `"0s"` or `NaN` risk.
- **Large value** (e.g. `duration = 7200000` ms = 2h): `(7200000/1000).toLocaleString()` → `"7,200"` → `"7,200s"`. Locale grouping applied correctly; consistent with `mp_con.toLocaleString()` (`MonsterSkillChip.tsx:115`), `count.toLocaleString()` and `limit.toLocaleString()` (`MonsterSkillChip.tsx:126,128`) — all numeric rows in this component use `toLocaleString()`, so the new Duration row's formatting is consistent with its siblings, not an outlier.
- **Non-multiple-of-1000 duration** (data anomaly, e.g. `1500`): `1.5.toLocaleString()` → `"1.5"` → `"1.5s"`. Not a crash, but worth flagging as a minor display edge case if the backend ever emits a non-thousand-aligned ms value — not expected per the reader.go conversion, and out of scope for a frontend-only fix.
- **`a.interval` row left untouched**: confirmed at `MonsterSkillChip.tsx:124` — `if (a.interval > 0) rows.push({ label: "Interval", value: `${a.interval}s` });` is unchanged by the diff (verified via `git diff 9fac48430..HEAD -- services/atlas-ui`, which shows only the `a.duration` hunk). Per task scope, `interval` is a distinct WZ field never touched by the ms conversion in `mobskill/reader.go`, so leaving it unscaled is correct — not a regression.

### Other consumers of `duration` (sweep for missed scaling)

Swept `mobSkillsService` (`services/atlas-ui/src/services/api/mob-skills.service.ts`), `useMobSkillData.ts` (`services/atlas-ui/src/lib/hooks/useMobSkillData.ts`), and grepped the whole `services/atlas-ui/src` tree for `MobSkillDetailAttributes` and `duration`:

- `mob-skills.service.ts:51-55` (`getMobSkillDetail`) returns `data.attributes` verbatim — no transformation, so it does not need scaling itself (correctly leaves that to the one render-time consumer).
- `useMobSkillData.ts` only calls `mobSkillsService.getMobSkillName` (`useMobSkillData.ts:39`) — it never touches `duration` at all.
- `grep -rn "MobSkillDetailAttributes"` across `services/atlas-ui/src` returns matches only in `mob-skills.service.ts` (type definition) and `MonsterSkillChip.tsx` (sole consumer). No second component/page renders `MobSkillDetailAttributes.duration`.
- **Conclusion: `MonsterSkillChip.tsx` is the only atlas-ui consumer of this field.** No other 1000×-inflated-render risk exists in the current tree.

### Summary

**Overall: NEEDS-WORK** (build/tests pass per pre-verified state, but one FE FAIL exists).

#### Blocking (must fix)
- **FE-17**: No test exists for `MonsterSkillChip.tsx`, and none was added to cover the new ÷1000 duration-formatting logic despite this being an arithmetic/unit-correctness change directly tied to the bug this task fixes. Add a render test (e.g. `components/features/monsters/__tests__/MonsterSkillChip.test.tsx`) asserting the "Duration" row shows `"30s"` for a 30000ms detail payload (and that it's omitted for `duration: 0`).

#### Non-Blocking (should fix)
- None beyond the above. Arithmetic, `toLocaleString()` consistency with sibling rows, the zero-guard, and the untouched `interval` field all check out correctly against the diff.

---

## Plan adherence

**Plan:** `docs/tasks/task-190-disease-duration-cancel-debuff/plan.md` (15 tasks, 111 checkbox items — none ticked in the plan file itself; the task was executed and verified against the tree/commits directly, not against checkbox state).
**Branch:** `task-190-disease-duration-cancel-debuff` (24 commits over `main`, 99 files changed, +6327/-143).

### Task-by-task verdict

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | atlas-data single ms conversion | MET | `services/atlas-data/atlas.com/data/mobskill/reader.go:72` — `m.Duration = uint32(node.GetIntegerWithDefault("time", 0)) * 1000`. Commit `c61a37cb7`. |
| 2 | atlas-monsters remove two double-conversions | MET | `monster/processor.go:1070` `durMs := int64(sd.Duration())` (no `*1000`); `:1109` `duration := time.Duration(sd.Duration()) * time.Millisecond`. `executeDebuff` (`:1246`) untouched — see FR-1.3 below. `TestBuildMistCreateBody_UnderCapIsNotClamped` present at `processor_test.go:1255`. Commit `8d1590dca`. |
| 3 | atlas-maps mist tick ms + comment | MET | `tasks/mist_tick.go:91` `Duration: int32(m.DiseaseDuration().Milliseconds())`, stale-comment block replaced and names `11e07dfa7` (lines 80-90). `mist_tick_test.go` updated (12 lines changed), not deleted. Commit `d29612e50`. |
| 4 | atlas-ui render ms as seconds | MET | `MonsterSkillChip.tsx:116-120` — `(a.duration / 1000).toLocaleString()`. Commit `b0faeda10`. (Frontend-guidelines section above flags a missing unit test for this diff — a real gap, but the task itself was implemented.) |
| 5 | Contract comment + 7-producer audit | MET | `services/atlas-buffs/.../kafka.go:47` authoritative statement present. Pointer comment `// milliseconds — contract owner: atlas-buffs kafka/message/character/kafka.go (task-190)` found in all 6 expected producer copies (atlas-channel, atlas-monsters/disease.go, atlas-maps/mist_tick.go, atlas-consumables, atlas-summons, atlas-messages) — confirmed by grep. `producer-audit.md` exists (8.8KB) with the full table, Range check (max observed 6000s, ≈358× below the int32 overflow bound), and an atlas-messages GM-override path investigated and correctly classified as a legitimate seconds-input surface, not a defect. Commit `6d4a89bbd`. |
| 6 | buffdurationguard CI analyzer | MET | `tools/buffdurationguard/{analyzer.go,analyzer_test.go,cmd/...,testdata/src/{bad,good}}` all present. CI job wired at `.github/workflows/pr-validation.yml:163` (`buff-duration-guard:`), added to aggregate `needs:` (`:738`) and its result is actually consumed (`:757`, checked in the failure expression at `:788`) — not a silent no-op. `CLAUDE.md:61` item 11 documents it. `producer-audit.md` "FR-3.2 guard demonstration" section contains the reintroduce/exit=1/restore/exit=0 transcript as required. Commits `68c011b5a`, `c0e0907c7` (legitimate review fix narrowing the allow-marker's honesty about cross-package scope — no behavior change, fixtures untouched). |
| 7 | CANCEL_DEBUFF codec | MET | `libs/atlas-packet/character/serverbound/cancel_debuff.go` — zero-field struct, empty `Encode`/`Decode`, `CancelDebuffHandle` const, matches plan verbatim. `packet-audit:verify` markers present in `cancel_debuff_test.go` for all 9 matrix versions. Commit `60a228e31`. |
| 8 | atlas-channel throttle registry | MET | `character/statreset/registry.go` — `Window = 1000*time.Millisecond` (`:32`), `GetRegistry`/`Allow`/`ClearCharacter` all present with matching signatures. `TestAllow_ThrottlesInsideWindow` present (`registry_test.go:38`), satisfying FR-2.3.1. Commit `a6b9a010b`. |
| 9 | atlas-channel EXPIRE command | MET | `kafka/message/buff/kafka.go` — `CommandTypeExpire = "EXPIRE"`, `ExpireCommandBody struct{}`; `character/buff/producer.go` — `ExpireCommandProvider`; `character/buff/processor.go:83-85` — `Processor.Expire` emits via Kafka only (satisfies NFR-1, no sync REST). Commits `3fca0dff2`, plus review-fix `1fc5adcd6` adding `TestExpireCommandProvider` (`producer_test.go:51`) pinning the wire value/envelope — a strengthening, not a deviation. |
| 10 | atlas-channel CANCEL_DEBUFF handler | MET | `socket/handler/character_cancel_debuff.go` — throttle-first-then-emit ordering exactly as specified, `tenant.MustFromContext(ctx)` (satisfies NFR-3), dropped-nudge log at `l.Debugf` (satisfies NFR-4), calls `buff.NewProcessor(l,ctx).Expire(...)`. `main.go:881` registers `handlerMap[charsb.CancelDebuffHandle] = handler.CancelDebuffHandleFunc`. `socket/init.go:56` calls `statreset.GetRegistry().ClearCharacter(t, s.CharacterId())` in the destroyer. No opcode literal in the handler (grep clean; the sole `0x63` hit in the whole Go diff is inside an explanatory doc comment, not code). Commit `4495385c5`. |
| 11 | atlas-buffs per-character sweep | MET | `character/processor.go:27` interface method, `:194` fleet sweep now calls shared `expireInto`, `:207` `ExpireForCharacter`, `:218` `expireInto`. `kafka/consumer/character/consumer.go:110-116` `handleExpire` registered and guards `c.Type != CommandTypeExpire`. `CommandTypeExpire`/`ExpireCommandBody` match atlas-channel's copy byte-for-byte (both `"EXPIRE"` / empty struct). Commit `f46a4be20`. |
| 12 | Route handler in all 10 templates | MET | `grep -c CancelDebuffHandle` returns exactly `1` for all 10 version templates and `0` for `template_gms_12_1.json` (untouched, satisfying design D8). All 10 entries preceded by `LoggedInValidator` (grep count = 10). `tools/template-opcode-order-guard.sh` exits 0 on the current tree. Commit `b3d8f5aae`. |
| 13 | Registry entries + matrix | MET (with in-scope expansion) | `docs/packets/registry/gms_v{48,61,72,79}.yaml` all carry the `CANCEL_DEBUFF` entry with the plan's exact opcode/address/note. `docs/packets/registry/gms_v92.yaml` does NOT exist (`ls` → No such file, satisfying design D7). `go run ./tools/packet-audit matrix --check` and `fname-doc --check` both exit 0 on the current tree. `status.json` shows all 9 matrix cells (`gms_v48`…`jms_v185`) at `"state": "verified"` with real opcodes (78/91/98/97/99/99/102/111/94) — the four false n-a corrections landed. Commit `805f059cc`, with `c05caabc5` regenerating a stale matrix `toolSha` after the `run.go` change (correct per the documented toolSha-reads-HEAD gotcha). The `candidatesFromFName` case for `CWvsContext::CheckTemporaryStatDuration` in `tools/packet-audit/cmd/run.go:780-784` was a necessary prerequisite for the nine-cell promotion via the standard `/verify-packet` tooling, not scope creep — consistent with the task's stated intent (Task 13's own text: "Each cell goes through the standard single-cell procedure"). |
| 14 | Backfill doc + 0x6C follow-up | MET (with reviewed renumbering) | `backfill.md` created, models `task-153`'s precedent, covers all 10 tenant versions with the opcode table, verification section, and the Skill.wz re-ingest rollout ordering. Follow-up filed in `docs/TODO.md:428-459` as **task-192** (not the plan's suggested `tools/task-numbers.sh next` output of 184, which the task correctly identified as a false negative for a reverted-and-deleted task-184 — documented via commits `06b9dc61d`/`05851f02e`). This is a deliberate, reviewed correction per the scope notes for this audit, not an unplanned deviation. |
| 15 | Verification sweep + review | MOSTLY MET — Step 10 UNRUN (by design) | Steps 1-7 (build/vet/test, go.mod/bake check, guards, packet gates, UI build, FR-3.2 re-demonstration, untouchable-site re-confirmation) all pass on the current tree per the task's own verification runs and my spot-checks above. Step 8 (this code review) is in progress — this document. Step 9 (address findings) is pending on the reviewers' output, including the frontend FE-17 blocking finding noted above. **Step 10 (live-tenant GMS 83.1 acceptance) is UNRUN** — it requires a deployed tenant and cannot be executed from a worktree; all six checklist items remain unchecked (`plan.md:2683-2688`), reported here as unrun, not as passing or failing. |

**Completion rate:** 15/15 tasks show file:line evidence of implementation (100%). 13/15 fully clean; Task 15 is gated on the in-progress review cycle (this document) and an unrun live-tenant acceptance step that is out of scope for a worktree-based audit.

### Must-not-happen constraints (checked explicitly, per the audit brief)

| Constraint | Result | Evidence |
|---|---|---|
| FR-1.3: `executeDebuff` unedited | **HOLDS** | `git diff main...HEAD -- services/atlas-monsters/.../processor.go \| grep executeDebuff` → no match. |
| Design D7: no `gms_v92.yaml` | **HOLDS** | `ls docs/packets/registry/gms_v92.yaml` → No such file or directory. |
| Design D8: `template_gms_12_1.json` unmodified | **HOLDS** | Absent from `git diff --name-only main...HEAD`; `CancelDebuffHandle` count in that file = 0. |
| FR-2.3.2/DOM-25: no hard-coded opcode literal | **HOLDS** | No opcode-literal hit in the handler, codec, or statreset package; the one `0x63` string anywhere in the Go diff is inside a doc comment explaining *why* opcodes aren't hard-coded, not a used value. |
| FR-1.6: seconds-pinning tests updated, not deleted | **HOLDS** | `git diff --stat -- '**/*_test.go'` shows only additions/expansions (447 insertions / 14 deletions, all deletions are replaced comment/assertion lines within the same test functions, which still exist under their original names: `TestBuildMistCreateBody`, `TestBuildMistCreateBody_DurationCap`, `TestMistTick_*`). No test function was removed. |

### Build & test status

Per the audit brief, the full sweep (go build/vet/test -race across every changed module, `libs/atlas-packet`, `tools/buffdurationguard`, all seven guard scripts, `matrix --check`, `fname-doc --check`, atlas-ui build + vitest) was already verified passing on the final tree by the task's own Task 15 run and is not re-run here. Spot-checks performed independently in this review: `go run ./tools/packet-audit matrix --check` (exit 0), `go run ./tools/packet-audit fname-doc --check` (exit 0), `./tools/template-opcode-order-guard.sh` (exit 0) — all confirmed clean from the repo root.

### Overall assessment

- **Plan adherence:** FULL for Tasks 1-14. Task 15 is a verification/review gate, not an implementation task — it is legitimately mid-flight (this review is part of it), with one item (live-tenant acceptance) structurally unrunnable from a worktree.
- **Recommendation:** NEEDS_FIXES before merge — driven by the frontend reviewer's FE-17 finding (no test for the `MonsterSkillChip.tsx` ÷1000 duration formatting), which plan Task 15 Step 9 requires be addressed on this branch before the audit is closed out. No plan-adherence gap blocks merge; the backend implementation across Tasks 1-14 is complete and evidenced end-to-end.

### Action items

1. Add a render test for `MonsterSkillChip.tsx`'s new duration formatting (FE-17, flagged above) per Task 15 Step 9's "fix what the audit surfaces, on this branch" requirement.
2. Task 15 Step 10 (live-tenant GMS 83.1 acceptance checklist, `plan.md:2683-2688`) remains to be run post-deploy; carry it forward as an explicit follow-up, not as a merge blocker for this worktree-based review.

---

## Backend guidelines

- **Service Path(s):** `libs/atlas-packet/character/serverbound`, `services/atlas-channel/atlas.com/channel/{character/statreset,character/buff,socket/handler}`, `services/atlas-buffs/atlas.com/buffs/{character,kafka/consumer/character}`, `services/atlas-data/atlas.com/data/mobskill`, `services/atlas-maps/atlas.com/maps/tasks`, `services/atlas-monsters/atlas.com/monsters/monster`, `tools/buffdurationguard`.
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`.
- **Build/Test:** Not re-run per audit brief (pre-verified clean for every changed module + all guard scripts). No finding below required a re-run.
- **Mindset applied:** default FAIL; every PASS below has a file:line citation.

### New/changed Go symbols — File Responsibilities Checklist

| ID | Symbol | File | Status | Evidence |
|----|--------|------|--------|----------|
| FILE-01 | `Processor.Expire` / `ProcessorImpl.Expire` | `services/atlas-channel/atlas.com/channel/character/buff/processor.go:25,83-86` | PASS | Interface method line 25, impl lines 83-86, same file as sibling `Apply`/`Cancel`/`CancelByTypes`. |
| FILE-01 | `ExpireForCharacter` / `expireInto` | `services/atlas-buffs/atlas.com/buffs/character/processor.go:207,218` | PASS | Both in `processor.go`, alongside `ExpireBuffs`/`ProcessPoisonTicks`. |
| FILE-01 | `handleExpire` | `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go:110-117` | PASS | Same file/shape as sibling `handleApply`/`handleCancel`/`handleCancelByTypes`. |
| FILE-05 | `ExpireCommandProvider` | `services/atlas-channel/atlas.com/channel/character/buff/producer.go:78-90` | PASS | Producer-file placement matches `ApplyCommandProvider`/`CancelCommandProvider` in the same file. |
| FILE-06 | `statreset` package (support pkg, no `model.go`) | `services/atlas-channel/atlas.com/channel/character/statreset/registry.go` | PASS | Single file, single responsibility (singleton rate-limiter: `GetRegistry`/`Allow`/`ClearCharacter`, lines 49-76). No catch-all bundling — not Processor+RestModel+requests collapsed into one file. Matches the established sibling singleton-registry shape (`shopscanner.GetRegistry`, referenced at `services/atlas-channel/atlas.com/channel/socket/init.go:53`). |
| FILE-06 | `CancelDebuffHandleFunc` | `services/atlas-channel/atlas.com/channel/socket/handler/character_cancel_debuff.go:1-48` | PASS | One handler per file, matching every other `character_*.go` file in the same package (e.g. `character_buff_cancel.go`-style siblings). |
| n/a | `CancelDebuff` codec | `libs/atlas-packet/character/serverbound/cancel_debuff.go:25-44` | PASS | `Operation()`/`String()`/`Encode()`/`Decode()` all in one file, matching every sibling codec in the package (e.g. `buff_cancel.go`). |

### DOM-25 — client wire values are config-resolved

- **PASS.** `grep -n "0x63\|0x4E\|0x5B\|0x6F"` across every new/changed Go file (codec, handler, registry, both processors, both producers, the consumer) returns zero matches — confirmed by direct grep, not by memory. The handler binds by name only: `services/atlas-channel/atlas.com/channel/main.go:881` — `handlerMap[charsb.CancelDebuffHandle] = handler.CancelDebuffHandleFunc`. Per-version opcode resolution lives entirely in tenant config: `services/atlas-configurations/seed-data/templates/template_gms_48_1.json` → `{"opCode": "0x4E", ...}`, `template_gms_61_1.json` → `0x5B`, `template_gms_83_1.json`/`template_gms_84_1.json` → `0x63`, `template_gms_95_1.json` → `0x6F` — five different opcodes for the same handler name, none hard-coded in Go.
- **PASS.** `template_gms_12_1.json` deliberately does not route `CancelDebuffHandle` — confirmed absent (`grep -L` above) — and this is a documented exception, not an omission: `docs/tasks/task-190-disease-duration-cancel-debuff/design.md:315-322` (design decision D8, "not routed, and that is not a gap").
- **PASS.** All 10 routed templates carry `"validator": "LoggedInValidator"` on the `CancelDebuffHandle` entry (spot-checked `template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_95_1.json` above) — an entry without a validator is silently dropped per `bug_socket_handler_missing_validator_silently_dropped.md`.

### DOM-21 — reuse `libs/atlas-constants`

- **PASS.** `statreset.Key{Tenant tenant.Model, CharacterId uint32}` (`registry.go:34-37`) introduces no new domain type — `uint32` for a character id and `tenant.Model` from the shared lib are both existing conventions used throughout the codebase, not a reinvented classification/enum. No new `const` block, item/inventory/weapon classification, or world/channel/map/job/skill/monster id type is declared anywhere in the diff's Go code.

### Multi-tenancy

- **PASS.** `services/atlas-channel/atlas.com/channel/socket/handler/character_cancel_debuff.go:38` — `t := tenant.MustFromContext(ctx)`, then used as the registry key (`registry.go:60-69`). Tenant is never read from the wire; the empty-body codec (`cancel_debuff.go`) carries no tenant data to begin with.

### Concurrency — `statreset.Registry`

- **PASS (singleton shape).** `sync.Once` + package-level `registry *Registry` (`registry.go:44-55`), matching the `cache.go` singleton pattern (`file-responsibilities.md:121-133`) even though the file is named `registry.go`, consistent with the existing sibling `shopscanner.GetRegistry()` singleton already in this codebase.
- **PASS (map-key safety).** `tenant.Model` (`libs/atlas-tenant/tenant.go:10-15`: `uuid.UUID` + `string` + two `uint16`s, all comparable) is safe as a map-key component; no slice/map field makes `Key` non-comparable.
- **PASS (session-destroy cleanup wired).** `services/atlas-channel/atlas.com/channel/socket/init.go:56` — `statreset.GetRegistry().ClearCharacter(t, s.CharacterId())` inside `SetDestroyer`, alongside the existing `shopscanner.GetRegistry().ClearCharacter` call (line 53) — same cleanup path, not a new one invented for this feature. Without it the map would leak one entry per character ever seen by the pod; the comment at `init.go:54-55` says so directly.
- **Minor, already adjudicated (not re-litigated as new):** `Registry.mutex` is declared `sync.RWMutex` but `Allow`/`ClearCharacter` both take the write lock (`registry.go:62-63,73-74`) — a plain `sync.Mutex` would express the same guarantee; no correctness impact since every call is already a write.

### DOM-24 — Kafka producer stubbed in tests that emit

- **PASS — `services/atlas-buffs/atlas.com/buffs/character/processor_test.go`** (exercises `ExpireForCharacter`, which calls `message.Emit` → `producer.ProviderImpl`): package has `services/atlas-buffs/atlas.com/buffs/character/testmain_test.go:10-13` — `TestMain` calling `producertest.InstallNoop()` (from the shared `atlas-kafka/producer/producertest` package, not a service-local `noopWriter`). No `t.Cleanup(producer.ResetInstance)` anywhere in the package (grep clean).
- **N/A — `services/atlas-channel/atlas.com/channel/character/buff/producer_test.go:48-70`** (`TestExpireCommandProvider`): `ExpireCommandProvider`/`producer.SingleMessageProvider` only builds a `[]kafka.Message` value in memory; it never calls `producer.ProviderImpl` or emits over the wire, so no stub is required for this test to be fast/correct.
- **Already adjudicated, not re-litigated:** the handler-level throttle-then-emit wiring (`CancelDebuffHandleFunc`) has no test exercising the real emit path; the throttle decision logic is fully unit-tested in isolation (`registry_test.go`). Matches the `mount_food_test.go` precedent per the audit brief.

### DOM-26 — goroutines via `routine.Go`

- **PASS.** `grep -rnE '^\s*go (func|[A-Za-z_])'` over every new/changed non-test file in scope (`statreset/`, `character_cancel_debuff.go`, `character/buff/`, `atlas-buffs/character/`, `kafka/consumer/character/`, `cancel_debuff.go`) returns zero matches — no bare `go` statements introduced.

### Disease-duration fix (atlas-data / atlas-maps / atlas-monsters) — correctness spot-check

Not owned by a DOM-* ID directly, but load-bearing for the whole task, so verified:

- **PASS.** Single conversion point: `services/atlas-data/atlas.com/data/mobskill/reader.go:72` — `m.Duration = uint32(node.GetIntegerWithDefault("time", 0)) * 1000`, with a comment declaring it the *only* seconds→ms conversion for mob-skill data.
- **PASS.** Both downstream double-conversions removed and not replaced with a second one: `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1070` (`durMs := int64(sd.Duration())`, no `* time.Second`) and `:1109` (`duration := time.Duration(sd.Duration()) * time.Millisecond`).
- **PASS.** `executeDebuff` (`services/atlas-monsters/atlas.com/monsters/monster/processor.go:1246`, `duration := int32(sd.Duration())`) is confirmed untouched — `git diff main...HEAD -- services/atlas-monsters/.../processor.go | grep executeDebuff` returns no hunk — matching the design's explicit "must not be edited" constraint.
- **PASS.** `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go:91` — `Duration: int32(m.DiseaseDuration().Milliseconds())` — and its test (`mist_tick_test.go`) updated in place to assert `int32(30_000)`, not deleted.
- **PASS.** The one legitimate remaining seconds-to-ms scaling site (`services/atlas-messages/atlas.com/messages/buff/processor.go`, `durationOverride * 1000` for the GM `@buff` chat command's human-authored seconds input) carries both an explanatory comment and a `//buffdurationguard:allow` marker. Per the audit brief this marker is already adjudicated as inert/documentation-only (the analyzer is single-package, one-hop) and is honestly labelled as such in both the code comment and the analyzer's package doc (`tools/buffdurationguard/analyzer.go:13-29`) — not re-litigated as a new finding.

### Summary

**Overall: PASS** — zero FAIL checks found across every DOM-*/FILE-*/SUB-* item applicable to this branch's Go changes. Build/test status pre-verified clean (not re-run here; no finding depended on re-running it).

#### Blocking (must fix)
- None.

#### Non-Blocking (should fix)
- `statreset.Registry.mutex` is `sync.RWMutex` but only ever write-locked (`registry.go:62-63,73-74`) — cosmetic, no correctness impact. Already flagged in prior review passes; carried forward here, not a new finding.

## Resolution — FE-17 (2026-08-04)

Addressed: added `services/atlas-ui/src/components/features/monsters/__tests__/MonsterSkillChip.test.tsx`, a render test covering the Duration row (30000ms detail payload renders `"30s"`, with an explicit regression guard that the raw `"30,000s"`/`"30000s"` value is absent), the `duration: 0` omission case, and a guard that the adjacent `interval` field stays unscaled. `MonsterSkillChip.tsx` itself was not modified. Non-vacuity of the new test was demonstrated by temporarily reverting the component's `/1000` scaling, confirming the test fails, then restoring the component and confirming it passes again.
