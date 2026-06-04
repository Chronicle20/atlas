# task-080 Packet-Audit Closeout — Code Review (F2)

**Verdict: READY_TO_MERGE.** Both reviewer agents passed with only Minor findings, all addressed.

Detailed reports:
- [Plan adherence](audit-plan-adherence.md) — `plan-adherence-reviewer`
- [Backend guidelines](audit-backend-guidelines.md) — `backend-guidelines-reviewer`

## Plan adherence — FULL functional adherence
All 38 tasks (36 planned + emergent A5 + the two surfaced follow-ups) accounted for across 49 commits, each backed by landed code with byte-level tests or an IDA-grounded verdict/no-op spike. Four affected modules build + test green; region-dispatched bodies stay within the 2-guard nesting cap; closed-item regression guard holds (storage Show, MonsterControl, SETFIELD/WarpToMap untouched).

Disproved-premise outcomes (correct, evidence-backed): B1.3 nItemPos (no-op, would corrupt every quest packet), B1.2 gate (corrected `>83`→`GMS && >=95`), B1.5/B5.1 chat/cash gate boundaries. B3.1–B3.6 all verdict-only (wire shapes already correct). Two real bugs surfaced, not buried: JMS isPoints→currency (fixed on-branch) and BuddyInvite missing fields (registered as a separate follow-up, `docs/packets/ida-exports/_pending.md` §9).

## Backend guidelines — all objective gates pass
Build/vet/test clean across the four touched modules. Encode/Decode region-version **symmetry verified** (the load-bearing check) for every dispatching codec. Region-dispatch idiom ≤2 guards. mist domain immutable-model + Builder + pure/side-effecting split correct. Handler→processor layering clean (B2.2 TODO stub replaced). DOM-21 satisfied (atlas-constants reuse).

## Findings addressed
- **gofmt** (backend Minor): 7 task-080-touched files reformatted — commit `b647bfb3e`.
- **SUMMARY legend** (plan-adherence Minor, §4.7 presentational): self-documenting accepted-exclusions legend added to the `writeSummary` generator + four SUMMARYs regenerated (verdict counts unchanged) — commit `5fd7e28b0`.

## Findings NOT actioned (justified)
- **Pre-existing `// TODO` markers** in `npc_continue_conversation.go` + `cashshop/processor.go`: confirmed identical on `main` (not introduced by task-080); the return-text TODO is blocked on an out-of-scope processor-signature change. Left as-is.
- **Remaining SUMMARY ❌/🔍** (v83 80/4, v87 75/2, v95 77/8, jms 77/2): all dispositioned accepted-exclusions (export read-order truncation, opaque register-boundary types, version-absent, representation-equivalence) in the curated `_pending.md` registry — zero open actionable deferrals. PRD §4.8 satisfied; the analyzer's fixable false-positive classes (A1–A5) are all resolved.
- **`tools/redis-key-guard.sh`** FAIL: pre-existing `atlas-monster-book` go.sum hygiene issue (empty task-080 diff); the two task-080-touched modules pass it directly.

## Verify gates (F1) — all task-080 gates PASS
`go test -race` + `go vet` + `go build` clean (4 modules); nesting cap clean; **`docker buildx bake atlas-maps` + `atlas-channel` both built**; template JSON valid; regression guard clean.

## Open follow-ups (tracked as separate tasks, NOT _pending deferrals)
1. JMS cash currency: **resolved** this branch (commit `b70b07079`).
2. **BuddyInvite missing client fields** — real buddy-domain wire bug (missing 2×Decode4 + GW_Friend trailing + inShop byte, 🔍 all 4 versions). Out of task-080's scope; registered as a follow-up.
