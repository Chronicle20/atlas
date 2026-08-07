# Plan Audit — task-145-player-reports (Plan Adherence)

**Plan Path:** `docs/tasks/task-145-player-reports/plan.md` (+ `scope-amendment.md` tasks 26–30)
**Audit Date:** 2026-08-05
**Branch:** `task-145-player-reports`
**Base Branch:** `main` (merge-base `9ecf9c9759a9db059fb40f497e7c6a04e43f06fd`)
**Range audited:** `9ecf9c9759a9db059fb40f497e7c6a04e43f06fd..6a9e91e561dbd7138274e636ec9fa545f304cd2f` (76 commits / 1668 files)

This report covers **plan-task adherence only**. Frontend-guidelines detail for
Task 20 lives separately in `docs/tasks/task-145-player-reports/audit.md` and is
not duplicated here.

## Executive Summary

All 30 tasks (base plan Tasks 1–25, plus scope-amendment Tasks 26–30) are
**DONE**. All 150 plan.md checkboxes are checked. Every module that changed
builds and tests clean under `-race`, every project guard (`redis-key-guard`,
`goroutine-guard`, `template-opcode-order-guard`, `service-registration-guard`,
`lint.sh --check`) and every `packet-audit` gate (`matrix --check`,
`operations --check`, `fname-doc --check`, `doc-freshness --check`,
`gate-check --check`) is clean, and `docker buildx bake atlas-ban
atlas-messages` (the two services whose `go.mod` changed) succeeded. I
independently re-derived the packet-verification campaign's cell count from
`docs/packets/audits/STATUS.md` rather than trusting prose: the correct
reconciled figure is **39 scoped cells, 36 promoted (✅), 3 blocked** on a
wedged jms IDA session — which matches the coordinator's framing exactly and
supersedes the scope-amendment's own stated "41," a genuine over-count the
branch caught and corrected in its final commit. The several places where
plan/amendment text turned out to be wrong (gms_92 exclusion, jms Task
29/30's resolution, the "8 vs 5" Class-B count, the 41-vs-39 cell count) were
all corrected with primary evidence and documented honestly, not used to
quietly narrow scope. Two minor, non-blocking findings are noted below (a
literal-NFR-wording tension in chat-capture's synchronous I/O, and a stale
"8" in scope-amendment.md's Class-B heading).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Packet codecs — SueCharacterResult, ClaimAvailableTime, ClaimSvrStatusChanged | DONE | `libs/atlas-packet/report/clientbound/{sue_character_result,claim_available_time,claim_status_changed}.go` + tests, verbatim per plan interfaces; golden+round-trip tests pass across all 11 version variants |
| 2 | Packet codec — ClaimResult (Success + Notice) | DONE | `libs/atlas-packet/report/clientbound/claim_result.go`; discrete structs, not a dispatcher family, per design §3.1 |
| 3 | Packet codec — ClaimRequest (serverbound) | DONE | `libs/atlas-packet/report/serverbound/claim_request.go` |
| 4 | atlas-ban — Kafka contract + report entity/model/builder | DONE | `kafka/message/report/kafka.go`, `report/{entity,model,builder}.go` match plan verbatim (topics, `MaxDescriptionLength=2000`, `MaxChatLogBytes=16384`, `open/reviewed/actioned` lifecycle) |
| 5 | atlas-ban — provider/administrator/migration wiring | DONE | `report/{provider,administrator}.go`; `main.go:57` registers `report.Migration` |
| 6 | atlas-ban — character & chat REST client packages | DONE | `character/{requests,rest,processor}.go`, `chat/{requests,rest,processor,rest_test}.go`, env-resolved via `requests.RootUrl` |
| 7 | atlas-ban — report processor + status-event producer | DONE | `report/{producer,processor}.go`; commit `165003cf6` fixed a real UTF-8 rune-boundary truncation bug the plan's own sample code would have introduced (byte-slicing could split multi-byte runes → invalid UTF-8 → Postgres insert failure), covered by dedicated boundary tests — a genuine improvement over the plan text, not a deviation-as-gap |
| 8 | atlas-ban — REPORT command consumer + main wiring | DONE | `kafka/consumer/report/consumer.go`; `main.go` registers `InitConsumers`/`InitHandlers` |
| 9 | atlas-ban — REST resource, routes, mock, README | DONE | `resource.go`/`rest.go`/`mock/processor.go` match plan; docs exceed plan (separate `domain.md`/`kafka.md`/`rest.md`/`storage.md`, not just README) |
| 10 | libs/atlas-redis — bounded append on TenantKeyedSortedSet | DONE | `keyed_sorted_set.go:57-72` `AddBounded`: pipelined ZADD + ZREMRANGEBYSCORE (age) + ZREMRANGEBYRANK (count) + EXPIRE; no new Redis client introduced |
| 11 | atlas-messages — chat capture buffer + capture wiring | DONE (see Finding 1) | `chat/{model,config,registry,processor}.go`; wired synchronously in `message/processor.go` after command-registry checks (slash commands excluded), before each Kafka emit; captured types match GENERAL/BUDDY/PARTY/GUILD/ALLIANCE/WHISPER/MESSENGER, excludes PET/PINK_TEXT; env defaults 900s/200 lines correct |
| 12 | atlas-messages — chat history REST endpoint | DONE | `chat/resource.go` registers `GET /api/chat/history`; ingress-exposure comment block cites Amendment 2 |
| 13 | atlas-channel — report Kafka contract + domain package | DONE | `services/atlas-channel/.../kafka/message/report/kafka.go` is byte-identical (`diff` = empty) to atlas-ban's contract file |
| 14 | atlas-channel — sue handler rewrite + claim handler | DONE | `socket/handler/sue_character.go` now forwards to `report.NewProcessor(...).Sue(...)`; `socket/handler/claim_request.go` new, registered `main.go:866` |
| 15 | atlas-channel — result/enable writers | DONE | All 4 writers registered in `produceWriters()` `main.go:817-820`; result/mode codes route through `atlas_packet.WithResolvedCode("operations", ...)` — no hardcoded literal found (DOM-25 clean) |
| 16 | atlas-channel — report status consumer | DONE | `kafka/consumer/report/consumer.go` implements the `(kind,status,errorCode)`→packet mapping; adds a `reportAnnouncer` seam pre-checking writer presence (beneficial addition beyond plan) |
| 17 | atlas-channel — claim-enable emission on session bootstrap | DONE | `kafka/consumer/session/consumer.go:173-186`, dispatched via `routine.Go` (not the plan's bare `go func(){}`, correctly following RR-6 goroutine-guard); config-gated — v61 template has no claim writers, so v61 sessions correctly receive no claim-enable send |
| 18 | Seed templates, dispatcher-doc ops tables, topic env vars, live-config patch doc | DONE | Per-version entries verified by JSON parse: gms_61 sue-only; gms_48/gms_12/jms zero report entries; gms_72/79/83/84/87/92/95 full 5-op set; `template-opcode-order-guard.sh` clean (22 arrays ascending); `operations --check` clean; `live-config-patch.md` (526 lines) documents the patch, prioritized by real user-visible impact |
| 19 | Ingress route for /api/reports (+ /api/chat/history per Amendment 2) | DONE | Both routes at `deploy/shared/routes.conf:337,342` and `deploy/compose/routes.conf:337,342`; commit `d5d8aad6b` correctly used the live `tools/gen-routes.sh` instead of the plan-named dead `sync-k8s-ingress-routes.sh` |
| 20 | atlas-ui — report types, service, hooks | DONE | Confirmed by separate `audit.md`; FE-17 test gap it flagged is **closed** by commit `f7c3cb97f` (`reports.service.test.ts`, 105 lines, asserts JSON:API PATCH envelope) |
| 21 | atlas-ui — ReportsPage (list) + registration | DONE | `pages/ReportsPage.tsx` + `pages/reports-columns.tsx` (6 columns + view action); route `App.tsx:266`; nav item in `app-sidebar-items.ts:50` (plan's exact line target had since been refactored — correctly adapted); breadcrumbs `lib/breadcrumbs/routes.ts:315-322`; test `ReportsPage.test.tsx` |
| 22 | atlas-ui — ReportDetailPage + status update dialog | DONE | 3-column layout (description/chat log/transcript) at `ReportDetailPage.tsx:155-224`; chat log rendered as plain text (no `dangerouslySetInnerHTML`); JSON:API PATCH envelope `reports.service.ts:40-44`; `UpdateReportStatusDialog.tsx` + test |
| 23 | IDA — name v83 CLAIM_REQUEST send-site, byte-verify (FR-6.2) | DONE | `CWvsContext::SendClaimRequest` named via 3-caller structural xref match, opcode 0x6A confirmed; `docs/packets/registry/gms_v83.yaml:2405-2410` |
| 23b | Registry correction — CLAIM_REQUEST on v72/v79 (FR-6.4) | DONE | v72 (0x69 @ `0x91f2b4`) / v79 (0x68 @ `0x9711ff`) registry rows added from live decompile, corrected from stale `csv-import` |
| 23c | Verified absences (FR-6.5) | DONE | v48 sue / v48+v61 claim absences reconfirmed in `packet-findings.md` §7.2a, self-flagged as a weaker evidentiary bar than the later jms sweep (§7.3/§7.4) — honest self-assessment |
| 24 | Packet verification campaign | DONE — see Finding 2 for reconciled numbers | 36/39 promoted cells with `packet-audit:verify` markers + per-cell IDA address + pinned evidence records; `matrix --check`/`gate-check --check` clean |
| 25 | Deferrals, docs, final verification gates | DONE | `docs/TODO.md` lines ~402-441 file exactly the real deferrals (quota/mesos, accused-notification, gms-12 blocked with corrected rationale, 3 jms cells blocked-not-unscoped); commit `6a9e91e56` self-corrects a false "3/3 jms promoted" claim in `packet-findings.md` §7.3 to the true "0/3" |
| 26 | New — gms_92 registry | DONE | `docs/packets/registry/gms_v92.yaml` (3902 lines); `-ida-database` flag added to `discover_ops.go:47` and `verify_serverbound.go:43`; `discover_gms_v92.md` worklist committed |
| 27 | New — gms_92 IDA export | DONE | `docs/packets/ida-exports/gms_v92.json` present |
| 28 | New — gms_92 matrix column (+ Amendment 3 template wiring) | DONE | `gms_v92` in `matrix.VersionKeys`/`shortLabels`; `PROCESS.md` version_count 10; `docs/packets/audits/gms_v92/` (1459 reports); pet/summon opcode fix (`0xC8/0xC9`→Pet*, real summon at `0xCB-0xD0`) and `DeleteCharacterHandle` `0x17`→`0x18` fix both confirmed live in `template_gms_92_1.json`; handler/writer count now 62/119 (up from the broken 47/65 stub), still below full neighbor parity by design (only the 67 hard-gate conflicts were required) |
| 29 | New — jms_185 CLAIM_REQUEST send-site (resolved as absence, not naming) | DONE, honestly documented | v83 send-site named (see Task 23); jms CLAIM_REQUEST resolved as a **verified absence** (commit `bc0637e56`) after 5 independent exhaustive searches (§7.3) — a different outcome than the amendment's literal "name the send-site" framing, but correctly and transparently recorded, including a self-correcting "Registry disposition — corrected by Task 24" note |
| 30 | New — jms sue absence | DONE | Commit `c26bfca35`/`01a89aaa4`; `packet-findings.md` §7.4 rules out the strongest candidate (`sub_575186`) on two independent grounds, checks all 7 callers + remaining 81 `SendChatMsgSlash` sites |

**Completion Rate:** 30/30 tasks (100%); 150/150 plan.md checkboxes (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every plan task and every amendment task has direct file:line evidence
of completion. The 3 jms clientbound-claim cells that did not promote are not
a skipped task — they are Task 24's own explicitly-scoped, explicitly-blocked
subset (infrastructure outage: `idb_list` reported the jms session
`is_active:true` but `lookup_funcs` timed out against it throughout the
campaign window while sibling sessions responded normally), recorded as such
in `docs/TODO.md`'s "Blocked (need re-verification once unblocked, not scope
decisions)" section rather than silently dropped.

## Findings

### Finding 1 (Low/Informational) — Task 11 chat capture is synchronous, arguably against the PRD's literal NFR wording

`prd.md` §8 states: "chat capture must not add per-message synchronous I/O in
the hot chat path beyond the existing Kafka emit; the buffer update rides the
existing event stream." The actual implementation calls `captureLine`
(a pipelined Redis round-trip) **inline and synchronously** inside
`message/processor.go`, before each chat-command's existing Kafka emit — not
off a separate consumer of `EnvEventTopicChat`. `design.md` §7.1 explicitly
chose this "single choke point" approach, but that decision was never
reconciled against the PRD's literal text, and no amendment records the
change. This is not a functional defect (builds/tests/guards are all clean,
and the added I/O is a single pipelined Redis call, not unbounded work), but
it is a real, undocumented divergence from an NFR as written. Recommend
either amending the PRD/design note to acknowledge the tradeoff explicitly,
or confirming the added latency is acceptable and closing the gap in docs.

### Finding 2 (Informational) — Reconciled verification-campaign arithmetic (39 scoped / 36 promoted / 3 blocked, not the amendment's "41")

I independently rebuilt the cell table from `docs/packets/audits/STATUS.md`
(not from prose) for all 5 report packets × 10 version columns:

| packet | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms |
|---|---|---|---|---|---|---|---|---|---|---|
| SUE_CHARACTER_RESULT | ❌ oos | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ absent |
| CLAIM_RESULT | ❌ oos | ❌ oos | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ **blocked** |
| CLAIM_AVAILABLE_TIME | ❌ oos | ❌ oos | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ **blocked** |
| CLAIM_STATUS_CHANGED | ❌ oos | ❌ oos | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ **blocked** |
| CLAIM_REQUEST | ⬜ absent | ⬜ absent | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ absent |

(Rows confirmed directly from `docs/packets/audits/STATUS.md` lines 75, 77,
78, 83, 647.) That's **36 ✅ cells**, **3 in-scope-but-blocked** (jms's
clientbound claim trio), for a reconciled scope of **39**. The
scope-amendment's own stated "31 → 41" over-counted jms by assuming both jms
CLAIM_REQUEST and jms SUE_CHARACTER_RESULT would land in-scope; investigation
found both to be genuine absences (excluded, per FR-6.5, not scope-cut). This
matches the coordinator's briefed figures (39/36/3) exactly, and the
branch's own final commit (`6a9e91e561dbd7138274e636ec9fa545f304cd2f`)
independently caught and corrected an earlier "3/3 jms promoted" overclaim in
`packet-findings.md` to the true "0/3" — evidence of the correction discipline
being real, not performative.

### Finding 3 (Trivial) — Stale count in scope-amendment.md's Class-B heading

`scope-amendment.md:227` headlines "Class B — 8 registry-vs-audit-report
conflicts," but the table beneath it (`:238-244`) lists exactly 5 rows, and
the surrounding prose (`:246`, `:361`) both say "5"/"five." The "8" was never
corrected. Cosmetic only — no functional impact, doesn't affect any gate or
matrix output.

### Finding 4 (Informational) — Live-tenant PATCH documented but not applied; exposure-risk doc landed in service docs rather than the task folder

Per `scope-amendment.md`, the live `GMS v92` tenant PATCH (fixing the
pet/summon misdecode and `DeleteCharacterHandle` opcode bugs, both
pre-existing and unrelated to sue/claim but discovered during this branch's
work) was intentionally **not applied** — "the patch itself was not attempted
here — it is a live-system change and the coordinator is surfacing it to the
user directly, per instruction." This is disclosed and intentional, not a
silent gap, but it is a live outstanding production action item worth
carrying forward (see Action Items). Separately, Amendment 2 says "Task 25's
documentation must state this exposure plainly" (about `/api/chat/history`
now being unauthenticated and reachable, including whispers); that statement
does exist, correctly, in `services/atlas-messages/README.md:9-13` and
`docs/rest.md:4-10` (landed via Task 12/25's commits), just not literally
inside `live-config-patch.md` or another task-folder doc as the amendment's
wording implies. Substance is present; location is a minor deviation from the
amendment's letter.

## Build & Test Results

| Module | Build | Vet | Test (`-race`) | Notes |
|---|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | PASS | `report/...` golden + round-trip across all version variants |
| `libs/atlas-redis` | PASS | PASS | PASS | |
| `services/atlas-ban/atlas.com/ban` | PASS | PASS | PASS | 27+ `report` subtests incl. HTTP handler + cross-tenant isolation tests |
| `services/atlas-messages/atlas.com/messages` | PASS | PASS | PASS | `chat` package included |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS | `report`, `kafka/consumer/report`, `socket/writer` all green |
| `tools/packet-audit` | PASS | PASS | PASS | all 13 packages `ok` |
| `services/atlas-ui` | PASS (`npm run build`) | — | PASS (`npm test`, 195 files / 1400 tests) | node v22.22.2 via nvm |

| Gate / Guard | Result |
|---|---|
| `tools/redis-key-guard.sh` | clean |
| `tools/goroutine-guard.sh` | clean |
| `tools/template-opcode-order-guard.sh` | clean (22 arrays ascending) |
| `tools/service-registration-guard.sh` | clean (ingress/k8s-configmap files changed, guard confirms) |
| `tools/lint.sh --check` (nvm use 22 first) | clean — "0 issues." on every module checked; only noise is pre-existing `generated_file_filter` cache warnings referencing unrelated sibling worktrees (task-153/task-189/task-147), not this branch |
| `packet-audit matrix --check` | clean |
| `packet-audit operations --check` | clean (0 absent-writer notes) |
| `packet-audit fname-doc --check` | clean |
| `packet-audit doc-freshness --check` | clean (10 versions, 7 CI gates) |
| `packet-audit gate-check --check` | clean |
| `docker buildx bake atlas-ban atlas-messages` | PASS (exit 0) — the two services whose `go.mod` changed (`atlas-ban` for the `atlas-constants` require fix, `atlas-messages` for the redis/chat work); `atlas-channel`'s `go.mod` was not touched, so no bake required there |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Non-blocking, recommend before merge or as immediate follow-up) Reconcile
   Task 11's synchronous chat-capture implementation against PRD §8's literal
   "rides the existing event stream" NFR wording — either amend the PRD/design
   doc to record the accepted tradeoff explicitly, or confirm no async
   refactor is actually needed.
2. (Non-blocking, cosmetic) Fix the stale "8" → "5" in `scope-amendment.md:227`'s
   Class-B heading.
3. (Outstanding, disclosed, not a branch gap) Apply the live `GMS v92` tenant
   config PATCH per `live-config-patch.md` — the pet/summon misdecode and
   `DeleteCharacterHandle` opcode fixes are real production bugs on the live
   v92 tenant today and were explicitly not patched as part of this branch;
   this needs a human-directed live-config change + `atlas-channel`/`atlas-login`
   restart, tracked separately from this branch's merge readiness.
4. (Outstanding, disclosed, not a branch gap) Re-run the 3 blocked jms
   clientbound-claim verifications (`CLAIM_RESULT`/`CLAIM_AVAILABLE_TIME`/
   `CLAIM_STATUS_CHANGED`) once the wedged jms IDA session (`b6864e54`) is
   healthy again — already tracked in `docs/TODO.md`.
