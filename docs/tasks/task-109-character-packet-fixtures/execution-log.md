# Task 109 — Execution Log & Plan Deviations

Companion to `plan.md`. Records execution-time discoveries that re-shaped the plan's
phase structure. **The deviations stay within the plan's own §C escalation rule**
("if a regenerated report is NOT verdict-clean, escalate that cell into its version's
Stage-2 IDB task") — they are not goal changes. All 47 cells remain in scope.

## Baseline (Task 0)
- Worktree/branch correct; `libs/atlas-packet` build/test/vet clean.
- `go run ./tools/packet-audit matrix --check` → **exit 0, 0 conflicts, 0 character lines.**
  (The plan/design expected a pre-existing non-zero conflict backlog; the committed
  state is actually clean.) **Acceptance bar: stay at 0 conflicts, introduce no new
  `character/*` problem.**
- Incomplete character cells = **exactly 47**, matching §C.

## Discovery 1 — "IDA-free Stage 1" is largely illusory (Tasks 1–3 dissolved)
Verified by running the report-gen pipeline (not from memory):

- **jms clientbound holes (orig Task 1, cells #1–5):** base fns present, but the resolver
  descends into delegates **absent from the jms export** — `CUserRemote::Init`,
  `SecondaryStat::DecodeForRemote`, `SecondaryStat::DecodeForLocal` — and hits an
  `unknown primitive "DecodeSub"` (8 hand-authored stub occurrences, jms-export-only;
  GMS exports expand these). No report is produced. Control run vs `gms_v95` → all five
  reports clean ⇒ jms-export gap, not a pipeline bug.
- **jms serverbound holes (orig Task 2, cells #6–8):** senders present, but delegate to
  `CLogin::SendRequest`, **absent from the jms export** ⇒ no report.
- **GMS KeyMapChange (orig Task 3, cells #9–11):** report-gen succeeds but is
  `FlatInvalid:true, verdicts [0,0,2,2,2,2]` — the **same TRUNCATION family** the plan
  flagged only for jms #47 (§A.3). Needs the SaveFuncKeyMap loop-89-vs-90 adjudication,
  i.e. a GMS-IDB step, not a clean report-gen.

**Consequence:** work consolidates strictly **per-IDB** (already mandated for Stage 2).
The escalated Stage-1 cells fold into their version's IDB session.

## Discovery 2 — jms IDB decompiles cleanly (the 17 jms cells are producible)
The instance at port **13338 is `MapleStory_dump_SCY.exe`**, not the `*_U_DEVM` build the
plan §B names. Probed `CLogin::OnCreateNewCharacterResult` @ 0x66ffa8 → **fully readable
pseudocode with named refs** (`GW_CharacterStat::Decode` @0x50ec17, `AvatarLook::Decode`
@0x51517e, `CInPacket::Decode1` …). So "SCY" is a usable de-SMC'd build, NOT the
undecompilable retail dump. ⇒ All jms cells are unblockable via **surgical, absent-only
export splices** harvested from this IDB. No human escalation; this is exactly the
"missing export → generate it" producible-work case (CLAUDE.md "No Deferring").

## Revised execution order (strictly one IDB at a time — select_instance is global)
1. **Task 4 — v83 IDB (13341):** orig #13,18,23,28,33,35,38 + escalated KeyMapChange v83 (#9).
   Creates the 3 NEW Phase-A test files.
2. **Task 5 — v84 IDB (13337):** orig #14,19,24,29,34,36,39,41,42,44,45 + escalated #12.
3. **Task 6 — v87 IDB (13340):** orig #15,20,25,30 + escalated KeyMapChange v87 (#10).
4. **Task 7 — v95 IDB (13339):** orig #16,21,26,31 + escalated KeyMapChange v95 (#11).
5. **Task 8 — jms IDB (13338):** ALL 17 jms cells. First the jms export-quality splices
   (CLogin::SendRequest, CUserRemote::Init, SecondaryStat::DecodeForLocal/ForRemote,
   GW_CharacterStat::Decode, AvatarLook::Decode; expand the 8 DecodeSub stubs into proper
   delegates), then the cells. Phase-A jms rows (#17,22,27,32) require Task 4 done first.
6. **Task 9 — final gate.**

## Cell dispositions / wire deltas
(appended as cells land)
