# Plan Audit — task-105-party-buddy-dispatcher-family

**Plan:** `docs/tasks/task-105-party-buddy-dispatcher-family/plan.md`
**Audit Date:** 2026-06-23
**Branch:** `task-105-party-buddy-dispatcher-family` (`main..HEAD`, 12 implementation commits `9c78c30c7`..`bff1ea3bc`)
**Verdict:** Plan faithfully implemented. No silently skipped or deferred work. All gates and builds green.

## Executive Summary

Every plan Task 1–9 is implemented with file:line evidence. Both party and buddy `Error`
catch-alls are split into discrete per-mode structs (15 party / 9 buddy), per-mode body funcs
replace the caller-selector bodies, run.go carries one `#`-entry per arm with both `#Error`
catch-alls removed, the v87/v95/jms operations tables are populated, all four atlas-channel call
sites are migrated, and both families are removed from the dispatcher-lint baseline
(`exempt_families: []`). The grounding-critical claims (party +1 shift, D8 "IDA wins" mode-only,
version-absent arms, buddy extra-byte version gating) all check out against the yaml/templates/code.
All four packet-audit gates exit 0; atlas-packet party/buddy build+test pass; atlas-channel builds.

## Per-Task Findings

### Task 1 — Author `party.yaml` + `buddy.yaml` (IDA-enumerated, 5 versions) — IMPLEMENTED
- `docs/packets/dispatchers/party.yaml` (145 lines): `writer: PartyOperation`, all 5 versions, per-arm
  IDA addresses in header (v83 0xa3e31c .. jms 0xb297e7), 25 operation keys with decimal mode bytes.
- `docs/packets/dispatchers/buddy.yaml` (75 lines): `writer: BuddyOperation`, 16 keys, 5 versions,
  byte-identical note documented.
- Both headers cite per-version function addresses and the StringPool-decrypt method (commits
  9c78c30c7, 1409ec980; address corrections in 5e117be67, 2eb67a8e5).

### Task 2 — Party discrete error structs; shared `Error` deleted — IMPLEMENTED
- 15 discrete structs in `libs/atlas-packet/party/clientbound/error.go`: AlreadyJoined1 (:38),
  BeginnerCannotCreate (:54), NotInParty (:70), AlreadyJoined2 (:86), PartyFull (:103),
  UnableToFindInChannel (:121), CannotKick (:139), OnlyWithinVicinity (:156), UnableToHandOver (:173),
  OnlySameChannel (:190), GmCannotCreate (:207), UnableToFindCharacter (:225), BlockingInvitations
  (:250), TakingCareOfInvitation (:280), RequestDenied (:310).
- Shared `Error`/`NewError` fully deleted (grep: only a historical comment reference in
  `error_test.go:188-189`). 62 `packet-audit:verify` markers in `error_test.go`.

### Task 3 — Party per-mode body funcs; `PartyErrorBody` removed — IMPLEMENTED
- 15 error body funcs in `party/clientbound/operation_body.go` (:128 PartyAlreadyJoined1Body ..
  :212 PartyRequestDeniedBody). Name-bearing arms (Blocking/TakingCare/RequestDenied) take `name`;
  mode-only arms take no param.
- `PartyErrorBody` removed — zero hits in `services/`+`libs/`.

### Task 4 — Buddy discrete error structs; shared `Error`/`BuddyErrorWriter` deleted — IMPLEMENTED
- 9 discrete structs in `libs/atlas-packet/buddy/clientbound/error.go`: ListFull (:45),
  OtherListFull (:60), AlreadyBuddy (:75), CannotBuddyGm (:90), CharacterNotFound (:105),
  UnknownError (:132), UnknownError2 (:163), UnknownError3 (:194), UnknownError4 (:225).
- `Error`/`NewBuddyError`/`BuddyErrorWriter` all deleted (zero grep hits). 45 verify markers.

### Task 5 — Buddy per-mode body funcs; `BuddyErrorBody` removed — IMPLEMENTED
- 9 error body funcs in `buddy/operation_body.go` (:62 BuddyListFullBody .. :110
  BuddyUnknownError4Body). `BuddyErrorBody` removed — zero hits.

### Task 6 — run.go per-mode `#`-entries; both `#Error` catch-alls removed — IMPLEMENTED
- `tools/packet-audit/cmd/run.go`: 15 party `#`-entries (:1382–:1521) and 9 buddy error `#`-entries
  (:1130–:1157) plus the non-error buddy/party entries.
- Both `OnPartyResult#Error` and `OnFriendResult#Error` catch-alls removed; remaining `#Error` hits
  belong to unrelated families (OnWhisper, OnGivePopularityResult, OnEntrustedShopCheckResult,
  CTrunkDlg). No `#Error` residue in `docs/packets/audits/`.

### Task 7 — Populate v87/v95/jms operations tables; tier-1 evidence; matrix — IMPLEMENTED
- Templates: v87/v95 PartyOperation=20 ops, jms=19 (UnableToFindCharacter version-absent in jms);
  BuddyOperation=16 in all three.
- +1 shift confirmed in all three templates: ALREADY_HAVE_JOINED_A_PARTY_2=17,
  FULL_CAPACITY=18; CANNOT_KICK=29; UNABLE_TO_FIND_THE_CHARACTER=37 (v87/v95) / absent (jms) —
  agrees with party.yaml.
- 116 party/buddy evidence files added under `docs/packets/evidence/`; 107 `packet-audit:verify`
  markers total (62 party + 45 buddy) — matches the plan's "107 tier-1 records."
- STATUS.md: `PARTY_OPERATION` and `BUDDYLIST` clientbound op-rows ✅ on all 5 versions.

### Task 8 — Migrate 4 atlas-channel call sites — IMPLEMENTED
- `socket/handler/party_operation.go:97` → `PartyUnableToFindCharacterBody()` (no name — D8),
  `:106` → `PartyUnableToFindInChannelBody()` (no name — D8).
- `kafka/consumer/invite/consumer.go:171` → `PartyRequestDeniedBody(targetName)`.
- `kafka/consumer/party/consumer.go:455` `partyErrorBody` switch over `PartyOperation*` consts,
  logged default (:498 "unmapped party error type; dropping"); ERROR_UNEXPECTED falls through.
- `kafka/consumer/buddylist/consumer.go:239` map literal over 6 `StatusEventError*` consts +
  logged default (:254). (Map-literal shape vs the plan's switch — functionally equivalent; all 6
  arms mapped, INV satisfied.)
- `PartyErrorBody`/`BuddyErrorBody` — zero hits in `services/`+`libs/`.

### Task 9 — De-baseline + gate sweep + runbook — IMPLEMENTED
- `docs/packets/dispatcher-lint-baseline.yaml:20` `exempt_families: []`; comment records campaign
  complete (commit bff1ea3bc).
- `docs/tasks/task-105-party-buddy-dispatcher-family/live-config-runbook.md` present (6.9 KB) with
  the EXECUTION-GATED / RECORDED-NOT-EXECUTED banner.

## Grounding-Critical Checks

- **Party +1 mode shift (v87/v95/jms):** ALREADY_HAVE_JOINED_A_PARTY_2 0x10→0x11, FULL_CAPACITY
  0x11→0x12, plus upper arms (CANNOT_KICK 25→29, CHANGE_LEADER 27→31, etc.) — yaml lines 107–142,
  templates, and run.go comments agree. PASS.
- **D8 "IDA wins":** UnableToFindCharacter (error.go:225) and UnableToFindInChannel (error.go:121)
  are MODE-ONLY structs (no `name` field); the channel call sites drop the name arg
  (party_operation.go:97/:106). PASS.
- **Version-absent arms:** UnableToFindInChannel / BlockingInvitations / TakingCareOfInvitation /
  RequestDenied omitted v87/v95/jms (yaml:114–119); UnableToFindCharacter omitted jms only
  (yaml:142, template jms None). PASS.
- **Buddy extra-byte version gating:** UnknownError/2/3/4 write the trailing byte under
  `t.IsRegion("GMS")`, mode-only on jms (error.go:137–158, 168–189, 199–220, 230+). UNKNOWN_1 /
  UNKNOWN_2 correctly have NO struct (only 9 buddy structs). PASS.

## Build & Test Results

| Gate / Module | Result |
|---|---|
| `packet-audit dispatcher-lint` | PASS (clean) |
| `packet-audit matrix --check` | PASS |
| `packet-audit operations --check` | PASS (1 benign absent-writer note: NoteOperation/jms, unrelated) |
| `packet-audit fname-doc --check` | PASS |
| `libs/atlas-packet` build + `go test ./party/... ./buddy/...` | PASS |
| `services/atlas-channel` build | PASS |

## Bottom Line

The plan is faithfully implemented. All 9 implementation tasks have file:line evidence; nothing was
silently skipped or deferred. The only deviation is cosmetic — the buddy call-site dispatch uses a
map literal rather than the plan's illustrative `switch`, which is functionally equivalent and
satisfies the same invariants. The runbook is correctly recorded-not-executed per design §9 / PRD §6.
Recommendation: READY for code review / PR (Task 10).
