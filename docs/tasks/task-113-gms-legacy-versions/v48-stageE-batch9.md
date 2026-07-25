# v48 Stage E — Batch 9 (character sub-family A) report

Anchor v61 fast-path, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch
`task-113-gms-legacy-versions`. Scope: char-mgmt + char-info + stat/AP/SP.

## Result summary

**8 gms_v48 character tier-1 cells promoted ❌/🟡 → ✅**, v48 verified count
**105 → 113 (+8)**. All other versions UNCHANGED (v61 208, v72 216, v79 228,
v83 367, v84 345, v87 379, v95 399, jms 362). `matrix --check` exit 0;
problem-grep 0; v48 conflicts 0.

Every read order was BODY-VERIFIED against the live v48 decompile — no
blind mirrors, no fabricated bytes.

## Per-cell outcomes

### Promoted this batch (✅ — 8)

| Cell | dir | v48 handler @addr | outcome |
|---|---|---|---|
| CHAR_NAME_RESPONSE | cb | sub_5016DB @0x5016db | ✅ =v83 (DecodeStr(name)+Decode1(result)) |
| DELETE_CHAR_RESPONSE | cb | sub_5017B6 @0x5017b6 | ✅ =v83 (Decode4(charId)+Decode1(result)) |
| CHECK_CHAR_NAME | sb | sub_500693 @0x500693 | ✅ =v83 (EncodeStr(name)) |
| CREATE_CHAR | sb | sub_500545 @0x500545 | ✅ (EncodeStr+8×Enc4+Enc1(gender)+4×Enc1(str/dex/int/luk); matches <=61 gate) |
| DELETE_CHAR | sb | sub_50043F @0x50043f | ✅ (Enc4(dob)+Enc4(charId); matches <=82 legacy branch) |
| DISTRIBUTE_SP | sb | sub_71CEB3 @0x71ceb3 | ✅ (Enc4(updateTime)+Enc4(skillId)) |
| CHAR_INFO_REQUEST | sb | sub_71D059 @0x71d059 | ✅ v48-gated (`<61` drops petInfo) |
| HEAL_OVER_TIME | sb | SendStatChangeRequest @0x71a482 | ✅ =v79 (`<83`: val+hp+mp+option, no updateTime) |

Each: v48 byte-golden `Test…V48ByteOutput` / `…ByteOutputV48` with a
`packet-audit:verify … version=gms_v48 ida=0x…` marker; pinned a TIER1-FIXTURE
evidence record (function → resolved sub_XXXX, `verifies:`); matrix regenerated;
all confirmed `state: verified` in status.json.

### char-management opcode BODY-verification (NAME=13 / ADD=14 / DELETE=15)

v48 handlers are unnamed `sub_XXXX` (no rotated symbol, unlike v79); opcode is
truth-by-body. Decompiled and confirmed the **same body→op mapping as v61/v83**:

- **CHAR_NAME_RESPONSE = op 13** (sub_5016DB): `DecodeStr(name)@0x5016eb +
  Decode1(result)@0x501705` — verified + fixtured.
- **ADD_NEW_CHAR_ENTRY = op 14** (sub_501973): `Decode1(result)@0x501987`; on
  success `GW_CharacterStat::Decode (sub_49B627) + AvatarLook::Decode` into the
  free slot — body-verified, NOT fixtured (nested; see remaining).
- **DELETE_CHAR_RESPONSE = op 15** (sub_5017B6): `Decode4(charId)@0x5017c6 +
  Decode1(result)@0x5017cc` — verified + fixtured.

No opcode rotation in v48 (distinct from the v79 rotated-symbol crash bug).

## Gate added

- **InfoRequest** (`character/serverbound/info_request.go`): added a
  `GMS && MajorVersion() < 61` gate that omits the trailing `petInfo` bool.
  v48 `CUser::SendCharacterInfoRequest = sub_71D059 @0x71d059` sends only
  `Encode4(updateTime)+Encode4(charId)` and SENDS (no third Encode1). The v61
  twin `sub_845B68 @0x845b68` appends `Encode1(petInfo)` (confirmed via the v61
  IDB + the v61 registry note). Boundary is exactly `<61`. v61+ UNCHANGED.
  Fixed `TestInfoRequestRoundTrip` to skip the petInfo assertion for legacy GMS
  (<61, i.e. v28/v48) since it cannot round-trip a field that is not on the wire.

## Registry alignment (op-cell linkage)

The 6 serverbound char ops had registry PRIMARY fname = `sub_XXXX` while their
audit reports carry the canonical `CLogin::…`/`CWvsContext::…` IDAName, so the op
row could not link its report (fell to a sub-struct row). Mirrored the proven
clientbound structure (primary = canonical fname, `fname_alts: [sub_XXXX]`,
`ida.address` = resolved sub) for CHECK_CHAR_NAME, CREATE_CHAR, DELETE_CHAR,
DISTRIBUTE_SP, CHAR_INFO_REQUEST → their **op cells** now verify. DISTRIBUTE_AP
alignment was **reverted** (see remaining — demux worst-of blocker). No new v48
conflicts; other versions untouched (per-version registry files).

## Verification bars

- `go test ./libs/atlas-packet/character/...` — green.
- `go vet` (character clientbound+serverbound) — clean.
- `go run ./tools/packet-audit matrix --check` — exit 0.
- problem-grep (`orphan|dangling|stale|drift|unresolv|malformed` STATUS.md) — 0.
- Regression: verified counts UNCHANGED for every existing version. v48 105 → **113**.
- Branch after commit: `task-113-gms-legacy-versions`.

## Commit (1)

1. `761d8aa69f` — verify char-mgmt/stat serverbound + name/delete cb (tier-1):
   8 cells, InfoRequest `<61` gate, 5-entry registry alignment.
   `git add` scoped to the 10 character test/codec files + gms_v48.yaml + 9
   `docs/packets/evidence/gms_v48/character.*` + STATUS.md/status.json. No
   out-of-scope report-regen drift (AuthSuccess/ChatMulti/ReactorHitRequest/
   SUMMARY/MonsterCarnival untouched; only `matrix` was run, never report-gen).

## EXACT remaining in-scope cells (6 op + 4 sub = 7 promotable units)

1. **DISTRIBUTE_AP** (sb, op) — **demux blocker.** The report IDAName is
   `CWvsContext::SendAbilityUpRequest#DistributeAp`; the shared base fname also
   carries a `#AutoDistributeAp` arm. In v48 both `#`-suffixed fnames are
   UNRESOLVED (empty address) in `gms_v48.json`, and op grading is worst-of
   across the demux siblings, so the unverified AutoDistributeAp arm caps the op
   at incomplete. The v48 send-site `sub_71CD00 @0x71cd00` IS body-verified
   (`Enc4(updateTime)+Enc4(dwFlag)`) and its byte-golden + evidence are committed
   (`TestDistributeApV48ByteOutput`). Unblock = disposition the v48
   AutoDistributeAp arm n-a (auto-stat-assign is a later feature; v48
   SendAbilityUpRequest is single-stat) via `_unimplemented.json` + remove the
   `#AutoDistributeAp` export stub, then the DISTRIBUTE_AP op grades clean. This
   is the AUTO_DISTRIBUTE_AP sibling cell (batch-adjacent), left un-touched to
   avoid an in-flight half-change.

2. **CHAR_INFO** (cb, op) — **MAJOR structural divergence; needs a v48 legacy
   codec branch.** `CWvsContext::OnCharacterInfo @0x71caed` (IDA-verified read
   order): `Decode4(charId) + Decode1(level) + Decode2(job) + Decode2(fame) +
   DecodeStr(guild) + Decode1(petFlag)` then a **single** flag-gated pet
   (`Decode4(templateId)+DecodeStr(name)+Decode1(level)+Decode2(closeness)+
   Decode1(fullness)+Decode2(skill)+Decode4(itemId)`), then
   `Decode1(mountFlag)+3×Decode4` (SetTamingMobInfo), then
   `Decode1(wishCount)+wishCount×Decode4` — **and returns.** v48 **OMITS** the
   marriage-ring bool, the alliance string, the medalInfo byte, the entire
   monster-book block (5 ints) AND the medal block, and has **no bool-terminated
   multi-pet loop** (single pet only). The shared `info.go` codec has no v48 path
   and would emit a wildly wrong wire. Fixture is blocked on writing a dedicated
   `GMS <? ` legacy encode/decode branch (boundary TBD vs v61 — v61 @0x8455ed
   per the info.go note reads the 5 monster-book ints, so v48 diverges BELOW
   v61's shape too; likely a `<61` or version-specific branch that drops
   marriage/alliance/medalInfo/monsterbook/medal and single-pets the pet block).

3. **CHARLIST** (cb, op) — needs a `<61` trailing-slots gate + nested entry
   fixture. `sub_5013ED @0x5013ed` (IDA-verified): `Decode1(status) +
   Decode1(count) + count×[GW_CharacterStat::Decode + AvatarLook::Decode +
   Decode1(rankFlag)?DecodeBuffer(16):memset]` and the loop **ENDS — v48 reads
   NO trailing Decode4** (the `characterSlots` field). The `list.go` codec writes
   `characterSlots` (WriteInt) for all GMS>28, so v48 needs a gate: after the
   entry loop, `if GMS && MajorVersion() < 61 { return }` (v61/v72/v79 read it
   per the list.go v79 note; v28 already returns early). Fixture additionally
   requires byte-verifying the v48 `GW_CharacterStat::Decode (sub_49B627)` +
   `AvatarLook::Decode` entry shapes against the CharacterListEntry codec —
   deferred to avoid a false pass on the nested legacy stat/avatar encoding.

4. **ADD_NEW_CHAR_ENTRY** (cb, op) — nested. `sub_501973 @0x501973` body-verified
   (`Decode1(result) + GW_CharacterStat::Decode + AvatarLook::Decode`), same
   nested entry shape as CHARLIST; blocked on the same legacy stat/avatar entry
   byte-verification.

5. **VIEW_ALL_CHAR** (cb, op) + **4 sub-structs**
   (`CharacterViewAllCharacters/Count/Error/SearchFailed`) — nested dispatcher.
   `sub_50232D @0x50232d` (OnViewAllCharResult) switches on `Decode1(sub-mode
   0–7)`: mode-0 per-char `GW_CharacterStat::Decode + AvatarLook::Decode +
   Decode1?DecodeBuffer(16)`, mode-1 `Decode4(count)+Decode4`. Same nested
   stat/avatar entry blocker as CHARLIST/ADD; the Error/SearchFailed arms are
   likely v48-present-or-absent to be dispositioned per-arm.

6. **VIEW_ALL_CHAR** (sb, op) — bare login-flow request. `sub_502293 @0x502293`
   sends bare `COutPacket(11)` then bare `COutPacket(12)` (no body fields). Needs
   an audit report generated for the login-flow serverbound struct
   (`CLogin::SendViewAllCharPacket`) + candidatesFromFName + routed-in-template
   before an (empty-body) fixture; the report is currently absent ("no audit
   report").

Out of sub-family A (not this batch): `character/clientbound/CharacterSetTamingMobInfo`
(SET_TAMING_MOB_INFO — spawn/pet family, batch 10/11).

## Notes / concerns

- The heavy remaining cells (CHAR_INFO legacy branch, CHARLIST/ADD/VIEW_ALL
  nested stat+avatar fixtures) each require verifying the v48 `GW_CharacterStat`
  / `AvatarLook` legacy encodings byte-for-byte before a golden can be trusted —
  a substantial, error-prone chunk. Stopped cleanly per the brief's budget rule
  rather than risk a false pass; each remaining item has its IDA-verified read
  order captured above so the follow-up is a fixturing/codec-branch task, not a
  re-discovery.
- DISTRIBUTE_AP is the only cell with a committed-but-unpromoted fixture; its
  byte-golden + evidence are valid, and it promotes the instant the
  AutoDistributeAp demux arm is dispositioned n-a for v48.
