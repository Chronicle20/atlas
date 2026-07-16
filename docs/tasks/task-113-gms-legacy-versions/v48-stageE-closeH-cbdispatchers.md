# v48 Stage E — CLOSE batch H (party-cb + guild-cb dispatchers) — report

Anchor v61. IDB port 13337 (`GMS_v48_1_DEVM.exe`). Branch
`task-113-gms-legacy-versions`.

## TL;DR / Status

**Neither cell promoted this batch. No code committed (tree left green).** The
batch premise — "party/guild OnPartyResult/OnGuildResult arms are MODE-ONLY,
need no String.wz/StringPool, just mirror v61's mode bytes" — is **materially
false and was disproven with IDA + a full StringPool decrypt**:

1. **StringPool IS required** to map each atlas arm → v48 mode byte, because the
   v48 dispatcher mode tables are **re-packed** (different case bytes than v61)
   **and** the StringPool notice ids are renumbered vs v61. The prior close-G
   report called the notice arms "SP-blocked"; **I UNBLOCKED them** by extracting
   and decrypting the v48 client StringPool directly from the PE resource
   (RT_RCDATA id 27892) — every entry verified against its embedded checksum
   (`seed==v26`, all `True`). Full decrypted text below.
2. **The data arms DO diverge** from v61/v83 and require shared-codec version
   gates. Party has THREE IDA-proven divergences; guild has several still
   **unresolved** (need v83 cross-decompiles).
3. The party gates ripple into a **shared test fixture (`GMS v28`)** whose wire
   semantics would change by **inference** (no v28 IDB) — a genuine
   design/boundary decision, and one that clashes with the campaign's
   "verify, don't invent" rule.

Because correct completion requires (a) a v28-boundary decision, (b) shared
generic-test surgery, and (c) a substantial guild port with **unresolved**
divergences, I stopped rather than land a large, partially-verifiable,
inference-based change under a false premise (CLAUDE.md: surface genuine
stop-and-ask cases; never present a spot-check as a full sweep). The verified
discovery below fully de-risks the follow-up.

## Method note (unblocks the "SP-blocked" claim)

The v48 StringPool is loaded at runtime from PE resource **27892 / RT_RCDATA**
(`sub_5D7739` → FindResource/LoadResource). It is NOT in the static IDB
(`.rsrc` not mapped), so the prior agent could not read it. I parsed the PE
resource directory from the on-disk exe (via `py_eval` on the IDA host),
extracted the blob, and reimplemented the block cipher from
`sub_5D7774`/`sub_5D75AF`:

- Block layout at `base + base[id]`: `[len(4)][data(len)][key(len)][chk(4)]`.
- `seed = 0xBAADF00D`; per-dword: `out = data ^ ROL(key,5)`; `seed = key +
  ROR(seed^data, 5)`; byte tail: `out = data^key`; `seed = key + ROR(seed^data,5)`.
- `ROL = sub_743A3A(x,5)`, `ROR = sub_7437F4(x,5)`. Validity: final `seed == chk`.

**All 40+ entries decrypted with valid checksums** — this is the same in-process
StringPool method task-105/task-103 used for v87/v95/jms, applied to v48's PE
resource. This method is reusable for any remaining v48 notice-arm work.

---

## PARTY_OPERATION cb — `CWvsContext::OnPartyResult` @0x729935 (FULLY MAPPED)

Switch on `Decode1(mode)`. Arm→mode resolved by decrypted StringPool text
(mode-only/name) and by read-order body (data). **Definitive:**

### Mode-only notice arms (encode = `[mode]`), SP text verified
| atlas arm | v48 mode | v48 SP id | decrypted text |
|---|---|---|---|
| AlreadyJoined1 | 8 | 310 | "Already have joined a party." |
| BeginnerCannotCreate | 9 | 311 | "A beginner can't create a party." |
| NotInParty | 12 | 312 | "You have yet to join a party." |
| AlreadyJoined2 | 15 | 310 | "Already have joined a party." |
| PartyFull | 16 | 313 | "The party you're trying to join is already in full capacity." |
| UnableToFindInChannel | 17 | 319 | "Unable to find the requested character in this channel." |
| GmCannotCreate | 24 | 2573 | "As a GM, you're forbidden from creating a party." |
| UnableToFindCharacter | 25 | 355 | "Unable to find the character." |

### Name arms (encode = `[mode][DecodeStr name]`), SP text verified
| atlas arm | v48 mode | v48 SP id | decrypted text |
|---|---|---|---|
| BlockingInvitations | 19 | 299 | "%s is currently blocking any party invitations." |
| TakingCareOfInvitation | 20 | 2435 | "'%s' is taking care of another invitation." |
| RequestDenied | 21 | 300 | "%s have denied request to the party." |

### Data arms (body-verified against the v48 switch)
| atlas arm | v48 mode | v48 read order | vs v83 |
|---|---|---|---|
| Invite | 4 | Decode4(partyId)+DecodeStr(name) | **DIVERGES: NO autoJoin byte** (v83 appends Decode1 autoJoin) |
| Update | 6 (also 26) | Decode4(partyId)+PARTYDATA::Decode | **DIVERGES: PARTYDATA has no leaderId** |
| Created | 7 | Decode4(partyId)+Decode4+Decode4+Decode2+Decode2 | = v83 |
| Left | 11 (true branch) | Decode4(partyId)+Decode4(targetId)+Decode1(=1)+Decode1(forced)+DecodeStr(name)+PARTYDATA | **DIVERGES: PARTYDATA leaderId** |
| Disband | 11 (else branch) | Decode4(partyId)+Decode4(targetId)+Decode1(=0), then stops | **DIVERGES: NO trailing repeated partyId** (v83 appends it) |
| Join | 14 | Decode4(partyId)+DecodeStr(name)+PARTYDATA | **DIVERGES: PARTYDATA leaderId** |
| TownPortal | 29 | Decode1(slot)+Decode4(town)+Decode4(field)+Decode2(x)+Decode2(y) | = v83 (no v95 skillId) |

### v48-ABSENT arms (would be n-a'd)
- **ChangeLeader** — no `Decode4+Decode1` change-leader body; v48 case 26 folds
  into the Update PARTYDATA body. (Matches the party/serverbound ChangeLeader
  n-a close-G already landed.)
- **CannotKick**, **OnlyWithinVicinity**, **UnableToHandOver**, **OnlySameChannel**
  — leadership-transfer/kick messages (v61 modes 25/27/28/29, high SP
  3971-3973); no v48 case decrypts to those texts. Post-v48 features.
- **MemberHP** (v48 case 27, Decode4×3) is a SEPARATE writer in atlas
  (`PartyMemberHP`, fname `CUserRemote::OnReceiveHP`), NOT part of the
  PARTY_OPERATION cell — out of scope for this op-cell.

### The PARTYDATA leaderId divergence (proven, high-confidence)
- `PARTYDATA::Decode` @0x49c925 = `DecodeBuffer(0x126 = 294)`; v83 = 298
  (`memset 0x12A`). Delta = 4 bytes.
- Case-7 (Created) sets `maps[0]` at CWvsContext `+10522` = PARTYDATA base
  (`+10348`) **+174** = ids(24)+names(6×13=78)+jobs(24)+levels(24)+channels(24).
  Maps immediately follow channels — **no leaderId gap**. v61/v83 (298) insert
  the 4-byte leaderId there. Boundary is (48, 61].

### What promotion WOULD require (party)
- Version-gate `WritePartyData`/`ReadPartyData` leaderId, `Invite` autoJoin
  byte, and `Disband` trailing partyId on `GMS MajorVersion < 61`.
- **Blocker:** those gates also change the shared `GMS v28` test variant
  (`libs/atlas-packet/test/context.go`, no v28 IDB). Applying an IDA-v48 finding
  to v28 is an **inference**, and it breaks the generic byte-count + round-trip
  tests (`update_test.go` etc. assume v28 == v83, leaderId present). The
  round-trip tests also assume leaderId survives — impossible when it's not on
  the wire. Resolving this needs an owner call on the legacy boundary + how the
  `GMS v28` fixture should behave. (I implemented the three gates, confirmed they
  break exactly the v28 generic party tests, then reverted to keep the tree
  green.)

---

## GUILD_OPERATION cb — `CWvsContext::OnGuildResult` @0x725559 (mapped; some arms UNRESOLVED)

Switch on `Decode1(mode)` (nested range-split; mode bytes shown as char literals
in the decompiler → decimal here). **v48 modes are re-packed vs v61** (v61
Invite=0x05, RequestName=0x01; v48 notices are in the 31-77 range with totally
different bytes). StringPool text decrypted & checksum-valid:

### Mode-only notice arms (encode = `[mode]`), SP text verified
| atlas arm | v48 mode | v48 SP id | decrypted text |
|---|---|---|---|
| CreateErrorDisagreed | 36 (0x24) | 2941 | "Somebody has disagreed to form a guild..." |
| CreateError | 38 (0x26) | 2947 | "The problem has happened during the process of forming the guild..." |
| JoinErrorAlreadyJoined | 40 (0x28) | 338 | "Already joined the guild." |
| JoinErrorMaxMembers | 41 (0x29) | 342 | "The guild ... has already reached the max number of users." |
| JoinErrorNotInChannel | 42 (0x2A) | 351 | "The character cannot be found in the current channel." |
| MemberQuitErrorNotInGuild | 45 (0x2D) | 340 | "You are not in the guild." |
| MemberExpelledErrorNotInGuild | 48 (0x30) | 340 | "You are not in the guild." |
| DisbandError | 52 (0x34) | 2948 | "The problem has happened during the process of disbanding the guild..." |
| CreateErrorCannotAsAdmin | 56 (0x38) | 348 | "Admin cannot make a guild." |
| IncreaseCapacityError | 59 (0x3B) | 2949 | "The problem has happened during the process of increasing the guild..." |
| QuestErrorLessThanSixMembers | 74 (0x4A) | 3197 | "There are less than 6 members remaining..." |
| QuestErrorDisconnected | 75 (0x4B) | 3198 | "The user that registered has disconnected..." |

Note: `CreateErrorNameInUse` = v48 mode **28 (0x1C)** (SP2942 "The name is
already in use...", + InputGuildName); `RequestName` = mode **1** and **28**
(InputGuildName); `RequestEmblem` = mode **17 (0x11)** (SendSetGuildMarkMsg).
Additional v48-only notice modes with no atlas arm: 31 (SP2946 gather-agreement
error), 33 (SP338), 34 (SP350), 35 (SP339 level-requirement) — candidate n-a or
new arms.

### Name arms (encode = `[mode][DecodeStr target]`), SP text verified
| atlas arm | v48 mode | v48 SP id | decrypted text |
|---|---|---|---|
| InviteErrorNotAcceptingInvites | 53 (0x35) | 323 | "%s is currently not accepting guild invite message." |
| InviteErrorAnotherInvite | 54 (0x36) | 2435 | "'%s' is taking care of another invitation." |
| InviteDenied | 55 (0x37) | 324 | "%s has denied your guild invitation." |

### Data arms — bodies vs atlas encoders
| atlas arm | v48 mode | v48 read order | status |
|---|---|---|---|
| RequestAgreement | 3 | Decode4(id)+DecodeStr+DecodeStr | = v83 ✓ |
| Invite | 5 | Decode4(guildId)+DecodeStr(name) | = v83 (atlas already gates trailing ints on ≥84; v48<84 → none) ✓ |
| MemberLeft | 44 (0x2C) | Decode4(guildId)+Decode4(cid)+DecodeStr(name) | = v83 ✓ |
| MemberExpel | 47 (0x2F) | Decode4(guildId)+Decode4(cid)+DecodeStr(name) | = v83 ✓ |
| Disband | 50 (0x32) | Decode4(guildId) | = v83 ✓ |
| MemberUpdate | 60 (0x3C) | Decode4(guildId)+Decode4(cid)+Decode4+Decode4 | = v83 (confirm level/job order) |
| MemberTitleUpdate | 61 (0x3D) | Decode4(guildId)+Decode4(cid)+Decode1(grade) | = v83 ✓ |
| TitleChange | 62 (0x3E) | Decode4(guildId)+5×DecodeStr | = v83 ✓ |
| EmblemChange | 66 (0x42) | Decode4(guildId)+Decode2+Decode1+Decode2+Decode1 | = v83 ✓ |
| NoticeChange | 68 (0x44) | Decode4(guildId)+DecodeStr | = v83 ✓ |
| ShowTitles | 73 (0x49) | Decode4(guildId)+Decode4(count)+loop[DecodeStr+5×Decode4] | = v83 ✓ (NOT mode 62) |
| QuestWaitingNotice | 76 (0x4C) | Decode1(chan)+Decode4(state) | = v83 ✓ |
| SetSkillResponse | 77 (0x4D) | Decode1(bool)+[DecodeStr] | = v83 ✓ |
| MemberJoined | 39 (0x27) | Decode4(guildId)+Decode4(cid)+GUILDMEMBER::Decode(33B) | **UNRESOLVED**: GUILDMEMBER=0x21=33B raw buffer; must byte-verify vs atlas `guild.Member` encode |
| Info | 32 (0x20) & 26 (0x1A) | GUILDDATA::Decode (variable) | **UNRESOLVED**: GUILDDATA layout — 5 rank titles, Decode1 count, 4·count ids + 33·count members, then Decode4+Decode2+Decode1+Decode2+Decode1+DecodeStr+Decode4; two decode arms (26/32) map to one atlas Info |
| MemberStatusUpdate | 58 (0x3A)? | Decode4(guildId)+Decode1(state) | **UNRESOLVED / likely DIVERGES**: atlas writes guildId+**charId**+online; v48 case 58 reads NO charId |
| MemberGrade (2nd) | 64 (0x40) | Decode4(guildId)+Decode4(cid)+Decode1(grade)+SP2960 | **UNRESOLVED**: two grade arms (61 & 64); atlas has one MemberTitleUpdate — which maps where? |
| CapacityChange / BoardAuthKey | 72 (0x48) | Decode4(guildId)+Decode4(value→+10686) | **UNRESOLVED / DIVERGES**: atlas CapacityChange = guildId+**Byte**(capacity); atlas BoardAuthKey = DecodeStr — neither matches Decode4+Decode4 |

The `= v83 ✓` arms are safe; the **UNRESOLVED** arms need v83 (@0xa3e31c-family)
cross-decompiles to decide gate-vs-remap (I did not run these). Because a cell
grades worst-of-siblings, guild cannot promote until every UNRESOLVED arm is
settled + GUILDDATA/GUILDMEMBER byte-layouts are confirmed against the atlas
encoders.

### Decrypted guild data-notice SP (for the data arms' chatlog text)
325 "%s You have joined a guild." · 327 "You have been expelled from the guild."
· 328 "You quitted the guild." · 329 "'%s' has been expelled..." · 331 "'%s'
quitted..." · 335 "The guild has been disbanded." · 336 "You have entered the
guild." · 337 "'%s' has joined the guild." · 2943 "...%s guild has been
registered..." · 2945 "...number of guild members has now increased to %d..." ·
2960 "[%s]'s position has been changed to [%s]." · 3015 "* Guild Notice : %s" ·
3199/3200/3201 (guild-quest waitlist).

---

## Gates / regression / matrix

- **No code committed. No test/marker/evidence/export/matrix changes landed.**
- Party codec gates (leaderId/autoJoin/disband, `GMS<61`) were implemented,
  shown to break exactly the `GMS v28` generic party tests, then **reverted**.
- Tree is green: `go test ./libs/atlas-packet/party/...` → ok (all 3 packages).
- `matrix --check`: unchanged from batch start (no artifacts touched); v48
  conflicts remain 0; existing verified counts NOT dropped (nothing changed).
- Branch verified: `git branch --show-current` = `task-113-gms-legacy-versions`.
- **Cells promoted: PARTY_OPERATION cb = NO; GUILD_OPERATION cb = NO.**

## Recommended follow-up (fully de-risked by this discovery)

1. **Owner decision** on the legacy party boundary: apply `GMS<61` legacy
   PARTYDATA (leaderId-absent) + Invite(no-autoJoin) + Disband(no-trailing) and
   update the `GMS v28` generic fixture expectations (byte counts + drop the
   round-trip leaderId assumption for `<61`) — OR scope the gate to only the
   versions with IDBs. Once decided, party promotes with the modes in this doc.
2. **Guild UNRESOLVED arms**: run the v83 OnGuildResult decompiles for modes
   58/61/64/72 and GUILDDATA/GUILDMEMBER, decide gate-vs-remap for
   MemberStatusUpdate (charId), the two grade arms, CapacityChange/BoardAuthKey,
   Info, MemberJoined; then fixture. All notice/name arms are ready (modes above).
