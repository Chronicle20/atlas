# v48 Stage E — CLOSE batch G (guild-cb + party producibles) — report

Anchor v61, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch `task-113-gms-legacy-versions`.

## Summary (N promoted / M n-a / K shared-gap / R remaining)

- **1 op-cell promoted**: PARTY_OPERATION serverbound v48 🟡 → ✅ (via
  PartyOperationInvite + PartyOperationExpel arm verification). v48 verified
  count 160 → **162** (+2), partial 4 → 2.
- **1 n-a**: party/serverbound PartyOperationChangeLeader (v48-absent).
- **3 shared-gap** cells documented (v83 anchor also ❌, out of scope).
- **2 cb dispatcher families remaining** (GUILD_OPERATION cb, PARTY_OPERATION cb)
  — data arms body-verifiable, **notice/error arms blocked on the v48 String.wz
  StringPool table** (genuine blocker, see below). Both cb mode tables were
  re-derived from the v48 switches (deliverable produced).
- `matrix --check` exit 0; problem-grep 0; v48 conflicts 0. No other version
  dropped. 2 commits.

## Regression check (all held exactly)
v48 162 (was 160, +2) · v61 208 · v72 216 · v79 228 · v83 367 · v84 345 ·
v87 379 · v95 399 · jms 362 · v48 conflicts 0.

---

## 1. PARTY_OPERATION serverbound — Invite + Expel VERIFIED, ChangeLeader n-a

Body-verified from the v48 send sites (opcode 94, mode-prefixed dispatcher):

| arm | v48 send-site | wire | atlas body | outcome |
|---|---|---|---|---|
| PartyOperationInvite | CField::SendJoinPartyMsg @0x4c54dd | `COutPacket(94)+Encode1(4)+EncodeStr(name)` | `WriteAsciiString(name)` | ✅ (mode 4) |
| PartyOperationExpel | CField::SendKickPartyMsg @0x4c55f3 | `COutPacket(94)+Encode1(5)+Encode4(charId)` | `WriteInt(charId)` | ✅ (mode 5) |
| PartyOperationChangeLeader | CField::SendChangePartyBossMsg | *absent in v48* | — | n-a |

- Reports for Invite/Expel already graded **Verdict 0** (match) and the export
  already carried the resolved send addresses; the missing artifacts were the
  v48 marker + evidence pin. Added both; matrix promoted the PARTY_OPERATION sb
  op-row (keyed by the registry primary fname `CField::SendJoinPartyMsg`).
- **ChangeLeader n-a**: `func_query name_regex 'PartyBoss|ChangeParty|PartyLeader|
  ChangeBoss'` over the v48 IDB returns **0 matches**; the complete `*Party*`
  set is `PARTYDATA::Decode@0x49c925`, `SendJoinPartyMsg@0x4c54dd`,
  `SendKickPartyMsg@0x4c55f3`, `OnTownPortalChanged@0x4ca8d7`,
  `OnPartyResult@0x729935`, `GetPartyTownPortal@0x72ac8d` — no change-boss send.
  Party leader-handover post-dates this 2009-era client (v83 has the discrete
  `CField::SendChangePartyBossMsg`). Added `_unimplemented.json` entry, stripped
  the unresolved export stub, removed the stale Verdict-4 report — mirrors the
  v61/v72/v79 anchor residual for this arm.

## 2. guild/serverbound GuildAgreementResponse — already ✅ (batch3)
Matrix line 628: v48 0x060 **✅**. Verified in batch3 via
`CField::SendCreateGuildAgreeMsg @0x4c5a18` (op-row absorbs its report; v48 uses
`CUtilDlg::YesNo` blocking modals instead of the `CUIFadeYesNo::OnButtonClicked`
dispatcher fname the later versions use). No change needed this batch. The
v48 OnGuildResult decompile (below) independently corroborates: mode `3`
(`v4==3`) calls `CField::SendCreateGuildAgreeMsg(agreed)` after building the
create-guild agree dialog.

---

## 3. Both cb mode tables RE-DERIVED (from the v48 switches)

### PARTY_OPERATION clientbound — `CWvsContext::OnPartyResult` @0x729935
`switch(Decode1(mode))`. v48 is **re-packed and structurally divergent** from
v61 (chatlog/StringPool notices, not v61's discrete `sub_678022` notice arms):

| mode | body (read order @0x729935) | atlas arm |
|---|---|---|
| 4 | Decode4(partyId)+DecodeStr(inviterName)+blacklist → CreateInvite/deny(op95) | Invite (data) |
| 6, 26 | Decode4(partyId)+PARTYDATA::Decode | Update (data) |
| 7 | Decode4(partyId)+Decode4+Decode4+Decode2+Decode2 (town-portal fields) | Created (data) |
| 8, 9, 12, 15, 16, 17, 24, 25 | `sub_5D75AF(SP<id>)`+ChatLogAdd — **notice arms** | notice (SP-blocked) |
| 11 | Decode4(partyId)+Decode4(memberId)+Decode1(bool)+PARTYDATA::Decode | Left/Disband (data) |
| 14 | Decode4(partyId)+DecodeStr(name)+PARTYDATA::Decode | Join (data) |
| 19, 20, 21 | DecodeStr(name)+`sub_5D75AF(SP299/2435/300)`+ChatLogAdd — **name-notice** | notice (SP-blocked) |
| 27 | Decode4(charId)+Decode4(hp)+Decode4(maxHp) → silent member-HP update | MemberHP (data) |
| 28 | Decode1(bool)+[DecodeStr]/SP318 — **notice** | notice (SP-blocked) |
| 29 | Decode1(idx<6)+Decode4+Decode4+Decode2+Decode2 → town-portal slot | TownPortal (data) |
| default | SP318 notice | — |

Confirms the brief's Stage-C note: UPDATE=6/CREATED=7/LEAVE=11/JOIN=14/TOWN_PORTAL=29.
Also INVITE=4, MEMBER_HP=27. **ChangeLeader has no discrete cb arm** (case 26
falls into the Update body — no `Decode1(leaderFlag)`), so PartyChangeLeader cb
is likely v48-absent/folded (to be confirmed with the notice-arm pass).

### GUILD_OPERATION clientbound — `CWvsContext::OnGuildResult` @0x725559
`switch(Decode1(mode))` (function size 0x1967; nested range-split switch). Mode
bytes decode as ASCII in the pseudocode; decimal values given here:

| mode | body @0x725559 | kind |
|---|---|---|
| 1 | (no wire read) → CField::InputGuildName | notice/action |
| 3 | Decode4(guess)==ctx guildId → DecodeStr+DecodeStr, build agree dlg, `SendCreateGuildAgreeMsg` | agreement (sb) |
| 5 | Decode4(guildId)+DecodeStr(name), blacklist → CreateGuildInvite / deny(op97 +53) | Invite |
| 21 (0x15) | Decode4(guildId)+DecodeStr(inviterName) …guild-mark set path | SetEmblem-ish |
| 28 | SP2942 NPCSay + InputGuildName | RequestName |
| 31 (0x1F) | SP2946 NPCSay | notice |
| 32 (0x20) | GUILDDATA::Clear+GUILDDATA::Decode; SP2943/SP325 chatlog | Info (data) |
| 33 (0x21)/34/35 | SP338/339/350 notices | notice |
| 36 (0x24) | SP2941 NPCSay | notice |
| 38 (0x26) | SP2947 NPCSay | notice |
| 39 (0x27) | Decode4(guildId)+Decode4(cid); if not-self GUILDMEMBER::Decode + insert + SP337 chatlog; else SP336 + re-send op96 mode0 | MemberJoined (data) |
| 40 (0x28)/41/42 | SP338/342/351 notices | notice |
| 44 (0x2C)/47 (0x2F) | Decode4(guildId)+DecodeStr(name); member-left/expel; SP327/328/329/330/331 chatlog + GUILDDATA::Clear | MemberLeft/Expel (data) |
| 45 (0x2D)/48 (0x30) | SP340 notices | notice |
| 50 | Decode4==guildId → disband; SP2944 NPCSay / SP335 chatlog; GUILDDATA::Clear | Disband (data) |
| 58 (0x3A) | Decode4+Decode1(loginState) → member status update; SP2945 NPCSay | MemberStatusUpdate (data) |
| 59 (0x3B) | SP2949 NPCSay | notice |
| 60 (0x3C) | Decode4(cid)+Decode4(level)+Decode4(job) → member level/job update | MemberUpdate (data) |
| 61 (0x3D) | Decode4(cid)+Decode1(newGrade) → member grade; if grade→build alarm | MemberTitleUpdate (data) |
| 62 (0x3E) | Decode4==guildId → 5×DecodeStr(gradeName) | ShowTitles (data) |
| 64 (0x40) | Decode4(cid)+Decode1(grade) → member grade change + SP2960 chatlog | MemberGrade (data) |
| 66 (0x42) | Decode2(capacity)+Decode1+Decode2+Decode1 → guild capacity/mark; SP3007 notice | CapacityChange/Emblem (data) |
| 68 (0x44) | DecodeStr(notice) → guild notice change; SP3015 chatlog | NoticeChange (data) |
| 72 (0x48) | Decode4 → set guild board auth key (10686) | BoardAuthKeyUpdate (data) |
| 73 (0x49) | Decode4+Decode4(count){DecodeStr+5×Decode4} → guild ranking list | ShowRanking (data) |
| 74 (0x4A)/75/76/77/82/83/84/85/86 | SP3197/3198/348/351/… notices | notice |
| 76 (0x4C) | Decode1(chan)+Decode4(state) → guild-quest waiting; SP3199/3200/3201 chatlog | QuestWaitingNotice (data) |
| 77 (0x4D) | Decode1(bool)+[DecodeStr]/SP350 | notice |

(Non-notice **data** arms are body-verifiable; **notice** arms depend on the
StringPool blocker below.)

---

## 4. GENUINE BLOCKER — cb notice arms need the v48 String.wz StringPool table

Both cb families route every error/notice arm through
`sub_5D75AF(&out, <int id>)` → `sub_5D7774(id, &out)` (StringPool lookup). What I
tried: decompiled `sub_5D75AF @0x5d75af` and its resolver `sub_5D7774`; the string
text is **resolved at runtime from the loaded String.wz StringPool**, not stored
statically in the IDB. The v48 ids (party: 299,300,308–313,318,319,355,2435,2573;
guild: 323,324,325,327–331,335–342,348,350,351,2941–2949,2960,2961,3007,3015,
3197–3201) are a **different renumbering** from the v61/v72 notice ids (v61
mapped SP333=AlreadyJoined1 etc. from a known table with a +13 offset vs v72),
and the v48 dispatcher is structurally chatlog-driven rather than one-notice-per-mode.

Mapping each v48 notice mode → the correct atlas notice struct (AlreadyJoined1,
BeginnerCannotCreate, PartyFull, GuildInviteDenied, GuildJoinError*, …) therefore
requires the **v48 String.wz StringPool id→string reference**, which is not
present in the IDB. Guessing the mapping would fabricate the mode→struct binding
and is refused per the no-invent rule. This is a stop-and-ask: the notice-arm
mapping unblocks once the v48 StringPool string table (or a verified id→string
cross-reference) is available.

Because the cb op-cells grade **worst-of-siblings**, neither GUILD_OPERATION cb
nor PARTY_OPERATION cb can promote to ✅ until the notice arms are mapped — so no
cb op-cell was moved this batch, and no partial cb fixtures were committed
(avoids a mixed-state dispatcher).

## 5. Shared-gap cells (v83 anchor also ❌ — out of task-113 scope, not fixtured)
- **DENY_PARTY_REQUEST cb** — v83 anchor incomplete.
- **party sb PartyOperation-base** (`CField::SendCreateNewPartyMsg`) — also
  unresolved in the v48 export; v83 incomplete.
- **party sb PartyOperationJoin** — v83 incomplete.
Recorded as consistent-scoping residual; no fabrication.

---

## EXACT remaining
1. **PARTY_OPERATION cb** (op 50 @0x729935): data arms Invite/Update/Created/
   Left/Join/MemberHP/TownPortal are body-verifiable now (mode table §3); notice
   arms (modes 8,9,12,15,16,17,19,20,21,24,25,28,default) blocked on v48
   StringPool. ChangeLeader cb likely v48-absent (confirm).
2. **GUILD_OPERATION cb** (@0x725559): ~20 data arms body-verifiable (mode table
   §3); ~20 notice/NPCSay arms blocked on v48 StringPool.
3. Both blocked strictly on the **v48 String.wz StringPool id→string table**.

## Commits (branch task-113-gms-legacy-versions)
1. `9b9346746e` — verify party/serverbound Invite+Expel (tier-1); 160→162.
2. `ec6dfef8a0` — disposition party/serverbound ChangeLeader n-a (v48-absent).

## Verification
- `go test ./libs/atlas-packet/party/... ./libs/atlas-packet/guild/...` green.
- `go vet ./libs/atlas-packet/party/serverbound/...` clean.
- `matrix --check` exit 0; problem-grep 0; v48 conflicts 0.
- Branch after each commit: `task-113-gms-legacy-versions`.
