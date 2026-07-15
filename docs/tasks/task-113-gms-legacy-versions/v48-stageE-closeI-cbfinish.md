# v48 Stage E — CLOSE batch I (party-cb + guild-cb finish) — report

Anchor v61. IDB port 13337 (`GMS_v48_1_DEVM.exe`), cross-refs v61 (13338) /
v83 dump (13342). Branch `task-113-gms-legacy-versions`.

## TL;DR

**Both op-cells PROMOTED.** `PARTY_OPERATION cb` v48 → ✅, `GUILD_OPERATION cb`
v48 → ✅ (bonus: `DENY_PARTY_REQUEST` sb v48 → ✅). `matrix --check` exit 0,
v48 conflicts 0, existing verified counts unchanged (v83 367 / v84 345 / v87 379
/ v95 399 / jms 362 / v72 216 / v79 228 / v61 208). Close-H's derived data was
applied and every load-bearing divergence re-verified against the live v48
switch. v28 folded into the v48 legacy wire per the controller decision.

## Method note

The whole batch grades **worst-of-all-arms sharing the dispatcher FName**
(`worstCandidateCell`): a cell promotes only when EVERY atlas writer report whose
base FName is `CWvsContext::OnPartyResult` / `OnGuildResult` grades ✅ (marker +
fresh evidence) or is removed (version-absent, matching the CannotKick "absent =
no report" convention). Evidence is pinned to the base dispatcher fname (v48
outlines every sub-arm inline → no per-arm addresses), exactly as v61 did.

## Part 1 — PARTY_OPERATION cb (@0x729935, IDA-confirmed)

Full v48 switch read confirmed close-H's map and all three data divergences:

| arm | v48 mode | outcome |
|---|---|---|
| AlreadyJoined1/2, BeginnerCannotCreate, NotInParty, PartyFull, UnableToFindInChannel, GmCannotCreate, UnableToFindCharacter | 8/15/9/12/16/17/24/25 | mode-only fixtured ✅ |
| BlockingInvitations, TakingCareOfInvitation, RequestDenied | 19/20/21 | name arm fixtured ✅ |
| Created | 7 | = v83, fixtured ✅ |
| Invite | 4 | **DIVERGES**: no autoJoin byte — gated `<61` ✅ |
| Update/Join/Left | 6/14/11(true) | **DIVERGES**: PARTYDATA no leaderId (294 vs 298) — gated `<61` ✅ |
| Disband | 11(else) | **DIVERGES**: no trailing repeated partyId — gated `<61` ✅ |
| TownPortal | 29 | = v83 (asserted for coverage; no tracked op-cell arm, no marker) |
| InviteReject (sb) | case 4 auto-decline, op 95 | fixtured `TestInviteRejectV48` ✅ |

**Data gates (all `GMS && MajorVersion()<61`, v61+ untouched):**
`party/member_data.go` (leaderId), `party/clientbound/invite.go` (autoJoin),
`party/clientbound/disband.go` (trailing partyId). IDA evidence:
`OnPartyResult` qmemcpy/memset = 0x126=294; case-4 reads only partyId+name;
case-11 else stops after Decode1(=0); `PARTYDATA::Decode@0x49c925`=DecodeBuffer(294).

**v28 test updates:** byte counts Update 303→299, Join 312→308, Left 318→314,
Invite 19→18; leaderId round-trip assertion made legacy-aware (expects 0 when
`<61`) in `update_test.go`, `join_test.go`, `left_test.go`, `member_data_test.go`.
Encode/decode symmetric (v48 round-trip tests added).

**n-a'd (IDA-verified wire-absent — no v48 case; stray stage-D reports removed):**
ChangeLeader (folds into Update 6/26), OnlySameChannel, OnlyWithinVicinity,
UnableToHandOver. (CannotKick already had no report.)

## Part 2 — GUILD_OPERATION cb (@0x725559, IDA-confirmed)

Full v48 switch read + v61/v83 cross-decompiles resolved every close-H UNRESOLVED
arm. Mode→arm map = the v48 seed template guild operations table (writer[29]),
each cross-checked against the switch body:

| close-H UNRESOLVED arm | resolution |
|---|---|
| MemberStatusUpdate (mode 58?) | Actually **mode 61** (Decode4+Decode4+Decode1 = guildId+cid+online) = v83 ✅. Mode 58 is CapacityChange (guildId+Decode1 byte). |
| mode 72 int-vs-byte | mode 72 (Decode4+Decode4→+10686) is NOT an atlas writer; the atlas CapacityChange = guildId+byte maps to **mode 58** = v83 ✅ |
| GUILDMEMBER layout (mode 39) | `GUILDMEMBER::Decode@0x49c982` = DecodeBuffer(**33**); v61 `@0x4b54f6` = DecodeBuffer(**37**). The 4B delta = trailing **AllianceTitle** int (guild alliances added at v61). **Gated `<61`.** |
| GUILDDATA layout (Info, modes 26/32) | `@0x49ca86` reads ONE trailing int after notice (points); v61+/v83 read two (points+**allianceId**). Same alliance feature. **Gated `<61`.** |
| two grade arms (61 & 64) | mode 61 = MemberStatusUpdate (online), mode 64 = MemberTitleUpdate (title) — both guildId+cid+Decode1 = v83 ✅. Two distinct client behaviours, both byte-stable. |

**Data gates (`GMS && MajorVersion()<61`):** `model/guild_member.go` (per-member
AllianceTitle), `guild/clientbound/info.go` (per-member AllianceTitle +
trailing allianceId). v61 GUILDMEMBER=37B confirms the (48,61] boundary exactly.
Every other data arm (RequestAgreement, Invite, MemberLeft/Expel, Disband,
CapacityChange, MemberUpdate, MemberStatusUpdate, MemberTitleUpdate, TitleChange,
EmblemChange, NoticeChange, ShowTitles, QuestWaitingNotice, SetSkillResponse) is
byte-identical to v83 — fixtured via cross-version equality. 35 present arms
fixtured + evidence.

**n-a'd:** BoardAuthKeyUpdate (guild-BBS board auth key — no v48 OnGuildResult
case; `GUILD_BBS_PACKET` absent in v48). Stray stage-D report removed.

**Blocker / finding (documented, not blocking the cell):** the v48 seed template
guild operations table (writer[29]) has a mode SWAP for the last two arms —
`SET_SKILL_RESPONSE=78` (no case 78 in the switch → default/crash) and
`BOARD_AUTH_KEY_UPDATE=77` (case 77 is actually the SetSkillResponse body
Decode1+[str]). The correct v48 assignment is `SET_SKILL_RESPONSE=77`, BoardAuthKey
absent. Not fixed here (runtime operations-table value, independent of the
codec-verification this batch targets; a fix touches `operations --check`). Flagged
for a template-correctness follow-up.

## Commits (both on `task-113-gms-legacy-versions`)
1. `task-113(v48): promote PARTY_OPERATION cb — legacy data-arm gates + fixtures`
   — 3 party codec gates, v48 party test, InviteReject v48, 4 arm-report n-a's,
   18 evidence, v28 test updates.
2. `task-113(v48): promote GUILD_OPERATION cb — legacy alliance gates + fixtures`
   — GuildMember/Info alliance gates, v48 guild test, BoardAuthKey n-a, 35 evidence.

## Verification
- `go test -race ./libs/atlas-packet/...` clean (incl. all 8 existing-version
  variants of the gated party/guild codecs AND the v28 round-trip tests).
- `go vet` / `go build ./libs/atlas-packet/...` clean.
- `go run ./tools/packet-audit matrix --check` → exit 0; v48 conflicts 0;
  problem count 0.
- Verified counts held: v48 163→**165** (PARTY_OPERATION+DENY_PARTY_REQUEST,
  then GUILD_OPERATION); v61 208 / v72 216 / v79 228 / v83 367 / v84 345 /
  v87 379 / v95 399 / jms 362 — none dropped.
- Commit scope verified (`git show --stat`): no out-of-scope report-regen drift
  (AuthSuccess/ChatMulti/ReactorHitRequest/SUMMARY/MonsterCarnival untouched).
- `git branch --show-current` == `task-113-gms-legacy-versions` after each commit.

## Cells promoted
- **PARTY_OPERATION cb v48 → ✅ (yes)**
- **GUILD_OPERATION cb v48 → ✅ (yes)**
- No still-blocked guild arm.
