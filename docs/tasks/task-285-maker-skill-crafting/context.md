# Task 285 — Implementation Context

Companion to `plan.md`. Records the key files, the decisions the plan is built on, and the facts verified against the code rather than assumed.

---

## Shape of the work

27 tasks across nine Go modules and five surfaces. This is roughly 2.5x a typical Atlas task, and honestly so: the PRD specifies recipe ingestion, a **new service**, two codecs (one a five-arm dispatcher family), a channel handler and writer, and a new compensable saga action with its cross-service inventory seam. Nothing here was padded, and nothing was dropped.

**The branch is naturally three mergeable slices.** Tasks 1–5 (registry + `atlas-data`) and Tasks 6–9 (packets) are each independently useful and independently verifiable; the design says so explicitly of stage 1. If the branch gets unwieldy, splitting after Task 5 and again after Task 9 costs nothing, because no later task edits those files again.

**Sequencing rule:** Tasks 6–9 depend on Task 1. Tasks 11–14 depend on Task 10. Tasks 17–24 depend on Task 15, and Task 24 depends on all of 17–23. Tasks 25–26 depend on Tasks 7, 8, 9 and 24. Everything else is order-free.

---

## Key files, by module

### `libs/atlas-packet` (module root `libs/atlas-packet`)

| File | Role |
|---|---|
| `resolve.go:41-48` | `WithResolvedCode(codeProperty, key string, factory func(byte) packet.Encoder)` — the config-driven mode contract |
| `resolve.go:55-` | `ResolveCode` — returns **99 on any lookup failure**, which is why Task 26 soft-resolves |
| `field/clientbound/mts_operation.go:39` | `MtsOperationWriter` — the shared-writer-name const pattern |
| `field/clientbound/mts_operation.go:55-105` | `MtsResultRegisterSaleEntryFailed` — the conditional-tail arm to copy for `bNoItemAwarded`/`bCatalystUsed` |
| `field/mts_operation_body.go:68-72` | a body function in full — the exact `WithResolvedCode` call shape |
| `cash/clientbound/shop_operation_result.go` | the canonical consolidated per-arm family file |
| `cash/clientbound/shop_operation_body.go:13-149` | the fixed operation-key const block |
| `test/context.go:19-40` | `pt.Variants` — the version index table (see below) |
| `report/clientbound/sue_character_result_test.go:1-50` | a complete `packet-audit:verify` fixture test |
| `character/clientbound/maker_result.go` | **new** — the five arm structs |
| `character/maker_result_body.go` | **new** — the five body functions |
| `character/serverbound/maker_skill.go` | **new** — the request codec |

`pt.Variants` indices, needed by every fixture test:

| version | index | `v.Name` |
|---|---|---|
| `gms_v83` | 1 | `GMS v83` |
| `gms_v87` | 2 | `GMS v87` |
| `gms_v95` | 3 | `GMS v95` |
| `jms_v185` | 4 | `JMS v185` |
| `gms_v84` | 5 | `GMS v84` |
| `gms_v48` | 7 | `GMS v48` |
| `gms_v61` | 8 | `GMS v61` |
| `gms_v72` | 9 | `GMS v72` |
| `gms_v79` | 10 | `GMS v79` |
| `gms_v92` | 11 | `GMS v92` |

### `libs/atlas-saga` (module root `libs/atlas-saga`)

| File | Role |
|---|---|
| `model.go:241` | `AcceptToParcel` — the action const block to append to |
| `payloads.go:20-32` | `AwardItemActionPayload` / `ItemPayload` — what `AwardAsset` can express |
| `payloads.go:69-77` | `AwardMesosPayload` — `Amount int32`, so a negative charge is representable |
| `payloads.go:135-143` | `DestroyAssetFromSlotPayload` — note `TemplateId` at `:141`, which exists to enable compensation |
| `payloads.go:880-915` | `AcceptToMtsListingPayload` — the explicit stat block plus `Slots uint16` at `:907`, cloned by `AwardCraftedAssetPayload` |
| `unmarshal.go:480-499` | the `AcceptToParcel` unmarshal case to model |
| `builder.go:19-24`, `:52-64`, `:67-79` | `NewBuilder`, the **only** `AddStep`, `Build` |
| `payloads_test.go:9-37` | the payload round-trip + legacy-JSON test template |

### `services/atlas-data/atlas.com/data` (module `atlas-data`)

| File | Role |
|---|---|
| `document/entity.go:15-26` | the shared `documents` table — why C-4 means no migration |
| `document/storage.go:20` | `NewStorage[I string, M Identifier[I]](l, db, r, docType)` |
| `document/db_storage.go:120-157` | the `clause.OnConflict` upsert that makes FR-1.6 structural |
| `commodity/` | the whole domain template (`processor.go` 61, `reader.go` 35, `registry.go` 18, `resource.go` 113, `rest.go` 33, `mock/processor.go` 29) |
| `commodity/reader_test.go:2627-2666` | the fixture mechanism — a raw XML **string**, parsed via `xml.FromByteArrayProvider` |
| `quest/reader.go:132-229` | the ordered-child-list idiom |
| `xml/model.go:21`, `:124` | `ChildByName`, `GetIntegerWithDefault` |
| `data/processor.go:42-59`, `:62`, `:193-194`, `:226`, `:302-307` | worker const block, `Workers` slice, the branch to mirror, `RegisterFunc`, `RegisterFileData` |
| `main.go:193` | where `commodity.InitResource` joins the chain |

### `services/atlas-inventory/atlas.com/inventory` (module `atlas-inventory`)

| File | Role |
|---|---|
| `asset/processor.go:510-517` | `CreateOptions` — the gap; no explicit stats, no slot count |
| `compartment/processor.go:1246-1358` | `CreateAssetAndEmit` → `CreateAssetAndLock` → `CreateAsset`; options built at `:1327-1334`, `:1346-1353` |
| `kafka/message/compartment/kafka.go:109` | `CreateAssetCommandBody` — the **inventory** copy |
| `kafka/consumer/compartment/consumer.go:232-243` | `handleCreateAssetCommand`, guarded by `database.ApplyOnce` (task-208) |
| `kafka/message/compartment/kafka_test.go:10-`, `:28-` | the round-trip / omitempty test template |
| `compartment/resource.go:67-72` | `handleGetCompartmentByType` — **`type` is required**, 400 without it |

### `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` (module `atlas-saga-orchestrator`)

| File | Role |
|---|---|
| `compartment/processor.go:33-48`, `:65-75` | the `Processor` interface and the `RequestCreateItem` → `...WithStats` delegation precedent |
| `compartment/producer.go:16-34` | `RequestCreateAssetCommandProvider` |
| `kafka/message/compartment/kafka.go:127-135` | `CreateAssetCommandBody` — the **orchestrator** copy |
| `saga/model.go:212`, `:356`, `:1575-1580` | alias, payload alias, **and the orchestrator's own** `Step[T].UnmarshalJSON` case |
| `saga/handler.go:152`, `:944-945`, `:2537-2560` | interface method, `GetHandler` case, handler body |
| `saga/event_acceptance.go:259` | the completion/error event pair |
| `saga/compensator.go:331`, `:1267-1276`, `:2977-3003`, `:3303-3311` | `CompensateFailedStep`, the `AwardAsset` reverse-walk arm, `lateCompensableActions`, `dispatchLateInverse` |

### `services/atlas-channel/atlas.com/channel` (module `atlas-channel`)

| File | Role |
|---|---|
| `main.go:1017-1023` | the `handlerMap[...]` block to append to |
| `main.go:702` | `produceWriters()` — the writer-name slice |
| `compartment/requests.go:12-14`, `:21-39` | `ByType` and the **accommodation** request |
| `compartment/rest.go:131-170` | the accommodation input/output models |
| `character/requests.go:11-28` | the `CHARACTERS` client |
| `quest/requests.go:21-28` | the paginated-URL + `DrainProvider` idiom |
| `data/equipment/requests.go` | the `DATA` client (`data/equipment/%d`) |
| `kafka/consumer/mts/consumer.go` | `failNoticeOr` — the soft-resolve fallback |

### `services/atlas-reward-pools/atlas.com/reward-pools` — the `atlas-maker` template

| Path | Role |
|---|---|
| `main.go:41-74` | the whole bootstrap; `func main()` starts at 41 |
| `gachapon/` | the full 12-file seeded domain package — the `reagent`/`crystalband` template |
| `reward/` | the compute-only domain (no entity, no table) — the `recipe`/`craft` template |
| `reward/processor.go:225-254`, `:262`, `:269-276` | `totalWeight`, `selectWeightedIndex`, the `crypto/rand` roll, the zero-weight fallback |
| `seed/groups.go` | seed-catalog group registration (32 lines) |

---

## Decisions the plan is built on

### Binding corrections to the PRD (design §2)

| # | Correction | Consequence in the plan |
|---|---|---|
| C-1 | `MAKER_RESULT` is **result-code**-prefixed; the mode is the second field, and `nResult > 1` ends the body | Task 8's arm layout; the `FAILED` arm is bodyless; still a legitimate family, since `DISPATCHER_FAMILY.md` requires a config-resolved mode, not a leading one |
| C-2 | the wire layout is version-invariant | no `MajorAtLeast` gate is written unless Task 6 finds otherwise; FR-4.5 is satisfied vacuously |
| C-3 | the request encodes the mode **once**, inside the switch | Task 7's layout; the double encode in the evidence doc and FR-4.3 is a transcription artefact |
| C-4 | `atlas-data` has no per-domain tables | Tasks 2–5 add **no migration and no table**; PRD §6's three relational tables do not apply |
| C-5 | the archive carries `reqQuest`, which FR-1.2 omits | ingested *and* enforced; adds the `missing_prerequisite_quest` code |
| C-6 | the top-level group digit must be persisted | `Group` on the model; mode 3's leftover lookup is scoped to group `0` |

### Open questions, as resolved

| ID | Resolution | Where |
|---|---|---|
| OQ-0 | settled in the evidence doc | Task 1 |
| OQ-1 | **unresolvable on this machine** — only a GMS 83.1 dump exists locally. Carried as risk R-2, mitigated structurally by FR-1.5's default-don't-fail reading | Task 27 Step 7 |
| OQ-2 | derive from equip `reqLevel`; band table derived from `Load_MonsterCrystalLevel` | Task 18 |
| OQ-3 | `ItemMake.img` group `0` alone suffices; **no** drop-table join | Task 20 |
| OQ-4 | no audit table; the saga record plus the correlated info log is the history | Task 23 Step 5 |
| OQ-5 | tenant-owned seeded table, seeded from the node read by `Load_GemEffect` | Task 17 |
| OQ-6 | out of scope per the PRD's non-goals | — |
| OQ-7 | **consume 100** of the leftover on mode 3; the client hard-codes it, and a 1-vs-100 mismatch is visible and exploitable. Task 23 Step 1 re-checks against the reference server before implementing | Task 23 |

### Things the survey found that the design got wrong or under-counted

These are the corrections that would otherwise have surfaced at dispatch time.

1. **`saga/model.go` needs two changes, not one.** The design's §4.5.1 table lists a single "local re-export". The orchestrator also maintains its **own** `Step[T].UnmarshalJSON` (`:1575-1580`) alongside the shared library's. Omitting the second yields an untyped payload at runtime with no compile error. Task 13 Step 3 and its `TestStepUnmarshalAwardCraftedAssetLocal` exist for this.

2. **`saga/rest.go` is NOT a required wiring point.** The design lists it. Evidence: `payloadUnmarshalers` (`rest.go:75-85`) covers only nine actions, and `AcceptToParcel` — a full multi-step compensable action — was never added to it, falling through to the untyped default at `:96-104`. The plan explicitly instructs **not** to add an entry.

3. **The most recent saga action is the wrong template.** `MapleLifeUse` (`model.go:85`, commit `7f157fb03`) is single-step and non-compensable; it touched none of `unmarshal.go`, `event_acceptance.go`, or `compensator.go`. The plan points at `AcceptToParcel` (commit `9486b6088`) instead. An implementer following `git log` would have silently omitted four of five wiring points.

4. **`CreateAssetCommandBody` is declared twice**, independently, in two modules with no shared type. A field added to one and not the other decodes as zero with no error. Tasks 11 and 12 extend both, and Task 12 Step 5 plus Task 27 Step 5 assert they agree.

5. **The accommodation endpoint already exists.** `POST characters/{id}/inventory/accommodation` takes a **list** of `{itemId, quantity}` and returns an overall verdict plus per-item results. The design's §4.2.2 step 6 proposed computing free-slot capacity in `atlas-maker`; the plan calls this instead. The repo already owns the computation and duplicating it would drift.

6. **`saga.NewBuilder()` has no typed helpers** — only the generic `AddStep`. Every step of all three craft sequences is a bare `AddStep` call. Noted so no implementer goes looking for `AddAwardMesosStep`.

7. **No precedent exists for the `MAKER_SKILL` codec shape.** No serverbound codec in `libs/atlas-packet` self-decodes its dispatch mode and then reads a different field set per mode. `rps/serverbound/operation.go` doesn't branch; `character/clientbound/attack.go` branches on a **caller-injected** discriminant; `cash/serverbound/item_use.go` branches on a version gate. Task 7 is budgeted as new-pattern work.

8. **The registry's `MAKER_RESULT` addresses are bulk-import placeholders.** `gms_v72.yaml` carries `ida.address: 8772770` and the *same* decimal appears on the adjacent unrelated `PLAY_MINI_GAME_SOUND` and `KOREAN_EVENT` entries. It disagrees with the design's own C-1 citation of `0x86a152`. Task 6 Step 6 corrects both v72 and v79.

9. **`commodity/reader_test.go` does not build `xml.Node` literals.** It parses a raw XML string through `xml.FromByteArrayProvider`. The design's "synthetic `xml.Node` fixtures built inline in Go" reads as struct construction; Task 3 Step 1 states the actual mechanism.

10. **Minor line drift in the design's citations.** `data/processor.go`'s const block is 42–**60** (design said ~42–59); the `Workers` slice is line **62** (design said 61); `main.go`'s `commodity.InitResource` is line **193** (design said 192). `RegisterFileData` at 302–307 and the `WorkerCharacterCreation` branch at 194 are exactly right. The plan carries verified numbers and tells implementers to re-confirm at edit time.

---

## Deliberately oversized tasks

`tools/plan-lint.sh` F4 is advisory and permits a large task when this file says why.

- **Task 16 (`atlas-maker` registration)** — ~14 files across CI config, k8s base, both overlays, ingress and db-bootstrap. It is the *same* mechanical registration repeated across independent hand-maintained lists, four of which fail **silently** when missed (`docs/adding-a-new-service.md` exists because atlas-mts missed all four main-overlay enumerations). Splitting it produces halves that are individually green and jointly broken, which is precisely the failure mode the doc was written to prevent.
- **Task 17 / Task 18 (`reagent`, `crystalband`)** — 12–13 files each, but each is one scaffolded domain package built in the standard dependency order (`model` → `entity` → `builder` → processor/provider → `rest` → `resource` → tests). The package is the reviewable unit; half a domain package does not compile.
- **Task 19 (upstream REST clients)** — 6 packages × ~4 files. One two-file pattern repeated six times with no shared state. Each pair is trivially reviewable and they are faster to write together than to hand off individually.

### Deliberately multi-service (F4 warnings)

- **Task 25 and Task 26** each touch `services/atlas-channel` **and** `services/atlas-configurations/seed-data/templates/`. The second is not a second service's code — it is tenant configuration data, and it is the routing table without which the handler and writer this same task registers are unreachable. Landing the code in one task and its `options.operations` table in another produces a task that passes review while the feature does nothing. They stay together.

Every other task is within the ~6-file guideline and single-service.

---

## Model pinning

Two tasks are IDA-derivation work and must be dispatched with `model: opus`:

- **Task 6** — per-version wire derivation for both ops across all eight versions. This is the task that discharges risk R-1 and produces the evidence every codec is written against.
- **Tasks 17 and 18 Step 1** — deriving the reagent stat table from `Load_GemEffect` and the crystal level bands from `Load_MonsterCrystalLevel`.

Every other task is Sonnet work.

---

## Rollout — two out-of-repo manual steps

Neither can be done from this branch. Both fail only at runtime and are invisible to CI. Repeat both in the PR description.

1. **Create `atlas-maker-main` on postgres.home before merging.** Main has no wave-0 create job; pods crash-loop with `SQLSTATE 3D000` until the database exists. Owner = the app role; `uuid-ossp` is inherited from `template1`.
2. **Flip the GHCR package to public after the first image push.** The first `docker buildx bake` push creates `ghcr.io/chronicle20/atlas-maker/atlas-maker` **private**, and the cluster pulls anonymously — no `imagePullSecrets` on any Deployment. CI reports a clean build while the pod sits in `ImagePullBackOff` against a 401. Verify:

```bash
curl -s -o /dev/null -w '%{http_code}\n' \
  'https://ghcr.io/token?scope=repository%3Achronicle20%2Fatlas-maker%2Fatlas-maker%3Apull&service=ghcr.io'
```

200 = public, 401 = still private.

Also confirm before merge that the live tenant configuration is patched with the new `MakerSkillHandle` and `MakerResult` entries — **seed templates never retroactively apply** to an existing tenant (`bug_new_opcodes_not_in_live_tenant_config`).

---

## Carried risks

| ID | Risk | Disposition |
|---|---|---|
| R-1 | four versions unsampled at design time | **Discharged by Task 6**, which re-derives all eight before any codec is written |
| R-2 | OQ-1 unanswerable locally — only a GMS 83.1 dump on disk | **Genuine external blocker.** Mitigated structurally: a missing field ingests as zero, a missing archive ingests an empty recipe set rather than failing startup. Surfaced in Task 27 Step 7, not guessed at |
| R-3 | new-service registration is silent-failure-prone | Task 16 walks every row of `docs/adding-a-new-service.md`; Task 27 Step 2 re-runs the guard and the render checks it cannot perform |
| R-4 | OQ-7 leftover quantity | Task 23 Step 1 re-checks against the reference server before implementing; the constant carries the rationale |

---

## Gate outcomes

### Verification gates

Only the `FLAGLESS VERIFY EXIT=` line inside a log file counts as a verdict — the background-task
notification reports the trailing `echo`'s exit status, not the gate's, and read `0` on
`gate-final2.log`, the run that actually FAILED.

| Log | Range / commit | Invocation | Verdict |
|---|---|---|---|
| `.superpowers/sdd/plan/gate-26.log` | `331b0c0b5..3d4391dad` (covers `61ff8cbd8`, `3d4391dad`) | `tools/verify.sh --quick --base 331b0c0b5` | **SUPERSEDED — does not certify anything.** `--quick` skips the docker bake and does not count as the branch gate; ends with repeated lint `0 issues.` lines, no flagless verdict. |
| `.superpowers/sdd/plan/gate-final.log` | launched at `79f6bd566` | `tools/verify.sh` (flagless) | **SUPERSEDED — stopped, not failed.** Stopped deliberately mid-bake when later commits landed underneath the run; superseded by `gate-final2.log`. |
| `.superpowers/sdd/plan/gate-final2.log` | `493bf669f` (full range through docs commit) | `tools/verify.sh` (flagless) | **FAIL.** `FLAGLESS VERIFY EXIT=1`, `1 check(s) FAILED — the branch is not ready.` One failure: `go build/vet/test -race services/atlas-configurations/atlas.com/configurations` → `--- FAIL: TestValidate_AcceptsEverySeedTemplate` at `corpus_test.go:64` (`corpus size = 3403 entries, want 3387`), diagnosed as a stale expectation — the branch adds exactly 16 seed-template bindings (8 templates × `MakerSkillHandle`+`MakerResult`) and the corpus guard was never bumped. All other 91 modules, every guard, routes drift, version coverage, overlay env drift, tenant tables, pr-sparse mirror, sparse baseline scoping, lint and format passed. Fixed by fix round 3, commit `e846ac3eb`. |
| `.superpowers/sdd/plan/gate-final3.log` | `493bf669f..e846ac3eb` | `tools/verify.sh` (flagless) | **PASS, but superseded by a later HEAD.** `FLAGLESS VERIFY EXIT=0`, `All checks passed.` (the unqualified message — not the `--quick` "bake was skipped" variant, confirming the bake ran). `grep -c "FAILED\|--- FAIL"` over the full log returns 0. This was the first and only creditable verification of the branch **at that time**, but HEAD later moved to `0261eb2c4` (the `/api/crystal-bands` ingress fix), so per CLAUDE.md this PASS no longer certifies current HEAD. |
| `.superpowers/sdd/plan/gate-final4.log` | launched at `0261eb2c4` | `tools/verify.sh` (flagless) | **SUPERSEDED — stopped, not failed.** Stopped deliberately partway through (last line `── go build/vet/test -race services/atlas-rates/atlas.com/rates`, no `FLAGLESS VERIFY EXIT=` line) once it became clear this write-up commit would land on top of `0261eb2c4` and invalidate it anyway. Certifies nothing. |
| `.superpowers/sdd/plan/gate-final5.log` | the write-up commit (branch HEAD) | `tools/verify.sh` (flagless) | The outstanding gate. Verdict is whatever its `FLAGLESS VERIFY EXIT=` line says; absent that line the run did not finish and must be re-launched, never inferred. |

The branch currently has **no gate log that certifies current HEAD**. `gate-final5.log`
must finish with `FLAGLESS VERIFY EXIT=0` before the branch can be called verified.

### Review verdicts

| Unit | Agent / model | Verdict | Caused a fix? |
|---|---|---|---|
| Task 27 code slice (`3d4391dad..79f6bd566`) | `task-reviewer` / sonnet | CHANGES_REQUIRED, 1 blocking | Yes — fix round 1, commit `3877d5047` (widened a mis-cited `plan.md` line range in `maker_skill.go:23` from `1455-1458` to `1455-1461`; the byte value `2` was already correct, this was a citation defect only). No re-review dispatched for this fix (controller independently re-read the cited range). |
| `backend-guidelines-reviewer` full `SCAFFOLD-*` audit of `atlas-maker` | `backend-guidelines-reviewer` / sonnet | CHANGES_REQUIRED, 7 blocking | Yes — two parallel fix rounds on disjoint file sets: **round A** (`8ada1d8ba`) closed the five structural findings (FILE-01/06 topic-named file, DOM-02/03 missing `ToEntity`/`Make`, DOM-05 inline transform, DOM-01 missing `builder.go` files); **round B** (`e25600ea5`) added the missing `server.ParseEnvironment` stage to `atlas-maker`'s REST handler wrappers, a `docker-compose.core.yml` entry, and a `.bruno` collection. The `ParseEnvironment` gap was found repo-wide (33 of 33 services with a local `RegisterHandler` omit it) but the ruling scoped the fix to atlas-maker only, since it is the new service this branch introduces; the other 32 are carried to the PR body as a separate follow-up. |
| Backend-audit fix rounds A + B (`3877d5047..8ada1d8ba`) | `task-reviewer` / sonnet | APPROVED, 0 blocking | No | 
| `plan-adherence-reviewer`, Tasks 1-14 shard | `plan-adherence-reviewer` / sonnet | APPROVED_WITH_FINDINGS, 0 blocking | No — one non-blocking finding, closed by the controller directly (not a fix round): review artifacts for Tasks 3, 12, 13, 14 had never been mirrored into `docs/tasks/task-285-maker-skill-crafting/reviews/`. All missing artifacts except Task 3's were copied in from `.superpowers/sdd/plan/task-N-review.md` (commit `493bf669f`). **Task 3 has no review artifact anywhere** — it was never written; its APPROVED verdict exists only in `progress.md`'s ledger entry ("Task 3: complete (commits `cda7f3f1e..f6d91735e`, review clean)"). |
| `plan-adherence-reviewer`, Tasks 15-27 shard (incl. 26a, 26b) | `plan-adherence-reviewer` / sonnet | APPROVED, 0 blocking | No |
| `packet-completeness-critic` | `packet-completeness-critic` / sonnet | CLEAN, 0 findings (CHANGED-BUT-UNCLAIMED: 0, CLAIMED-BUT-UNVERIFIED: 0) | No |

Fix rounds on this branch, end to end: Task 27 code slice fix round 1 (`3877d5047`), backend-audit
fix round A (`8ada1d8ba`), backend-audit fix round B (`e25600ea5`), and fix round 3 for the
gate-final2 corpus-guard failure (`e846ac3eb`) — at least three fix rounds plus the corpus-guard
gate fix, all landed and re-verified. The final audit-fix review (`task-27-audit-fixes.md`, over
`3877d5047..8ada1d8ba`) closed **APPROVED, 0 blocking**, re-running build/test/lint/guard checks on
the merged tree rather than trusting either concurrent round's self-report.

One further producible fix landed after all reviews closed: the `/api/crystal-bands` ingress gap
(found as a non-blocking note in the audit-fix review — `deploy/shared/routes.conf` had no
`location` block for that prefix, so it fell through to the `atlas-ui` catch-all) was fixed directly
rather than carried to the PR body, via commit `0261eb2c4`. This is why `gate-final3.log`'s PASS no
longer covers current HEAD and `gate-final5.log` is the outstanding gate.
