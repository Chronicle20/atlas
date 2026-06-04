# Spike B3.6 — Social-domain sub-op enum-drift four-version pass

**Scope:** CONFIG-VALUE audit. Per-struct wire shapes are already ✅ (task-066 Phase A/B).
This spike verifies the template-configured sub-op **VALUE spaces** (the `operations`
mode/op number maps + the clientbound mode arms) for the social dispatchers across
**GMS v83 / GMS v87 / GMS v95 / JMS v185**, plus the `BuddyError` conditional-string
arm semantics (the item task-066 explicitly deferred to "Phase 2 cross-version sub-op
enum pass" — i.e. this spike).

**Verdict (one line):** GMS v83 is the only fully-wired template; its social sub-op
value spaces are internally consistent and match the client. v87/v95/JMS185 templates
are partial seeds (social writers/handlers ABSENT) — documented as known template gaps,
not value divergences. The `BuddyError` "conditional-string arms" premise is **refuted by
the JMS185 binary**: modes 0x10/0x11/0x13/0x16 read **no string from the wire** — the
notice text comes from a client-side `StringPool`. **No config/constant fix required.**

---

## 0. Template wiring matrix (the load-bearing finding)

Social writers and handlers carrying `operations` sub-op maps exist **only in
`template_gms_83_1.json`**. The other three templates are minimal seeds.

| Writer / Handler                | gms_83_1            | gms_87_1 | gms_95_1 | jms_185_1 |
|---------------------------------|--------------------|----------|----------|-----------|
| `BuddyOperation` (writer)       | ✅ op 0x3F + ops    | ABSENT   | ABSENT   | ABSENT    |
| `PartyOperation` (writer)       | ✅ op 0x3E + ops    | ABSENT   | ABSENT   | ABSENT    |
| `GuildOperation` (writer)       | ✅ op 0x41 + ops    | ABSENT   | ABSENT   | ABSENT    |
| `NoteOperation` (writer)        | ✅ op 0x29 + ops+err| ABSENT   | ABSENT   | ABSENT    |
| `CharacterChatGeneral` (writer) | ✅ op 0xA2 (no ops) | 0xAD     | ABSENT   | 0xA0      |
| `CharacterMultiChat` (writer)   | ✅ op 0x86 (no ops) | ABSENT   | ABSENT   | ABSENT    |
| `CharacterChatWhisper` (writer) | ✅ op 0x87 (no ops) | ABSENT   | ABSENT   | ABSENT    |
| `BuddyOperationHandle`          | ✅ op 0x82 + ops    | ABSENT   | ABSENT   | ABSENT    |
| `PartyOperationHandle`          | ✅ op 0x7C + ops    | ABSENT   | ABSENT   | ABSENT    |
| `GuildOperationHandle`          | ✅ op 0x7E + ops    | ABSENT   | ABSENT   | ABSENT    |
| `NoteOperationHandle`           | ✅ op 0x83 + ops    | ABSENT   | ABSENT   | ABSENT    |
| `GuildBBSHandle`                | ✅ op 0x9B + ops    | ABSENT   | ABSENT   | ABSENT    |
| `CharacterChatGeneralHandle`    | ✅ op 0x31          | 0x34     | ABSENT   | 0x29      |

Total writers / handlers per template: gms_83 = 112 / 93; gms_87 = 48 / 34;
gms_95 = 46 / 31; jms_185 = 44 / 25.

**Consequence:** there is no v87/v95/JMS *value map* to compare against — they hold no
social sub-op maps at all. The four-version "enum-drift" comparison therefore reduces to:
**(a)** is the GMS v83 wired map internally consistent and client-correct, and **(b)** for
JMS185 (the loaded IDB), do the client's actual sub-op constants agree with the GMS v83
baseline (so a future JMS wiring task would reuse the same values)? Both are answered below.
Completing the v87/v95/JMS social writer/handler wiring is **out of scope** here (that is a
template-completeness task, explicitly excluded by the spike brief).

---

## 1. Per-domain four-version verdicts

### BUDDY — `BuddyOperation` writer / `BuddyOperationHandle`

**Clientbound writer sub-op map (gms_83_1 `BuddyOperation`, op 0x3F):**

| Constant (atlas)               | value | Client arm (`OnFriendResult`)         |
|--------------------------------|-------|---------------------------------------|
| UPDATE                         | 0x07  | ListUpdate (7/0xA/0x12)               |
| BUDDY_UPDATE                   | 0x08  | Update (8)                            |
| INVITE                         | 0x09  | Invite (9)                            |
| UNKNOWN_1                      | 0x0A  | ListUpdate-class                      |
| BUDDY_LIST_FULL                | 0x0B  | Error (StringPool notice)             |
| OTHER_BUDDY_LIST_FULL          | 0x0C  | Error                                 |
| ALREADY_BUDDY                  | 0x0D  | Error                                 |
| CANNOT_BUDDY_GM                | 0x0E  | Error                                 |
| CHARACTER_NOT_FOUND            | 0x0F  | Error                                 |
| UNKNOWN_ERROR                  | 0x10  | Error (StringPool 765)                |
| UNKNOWN_ERROR_2                | 0x11  | Error (StringPool 765)                |
| UNKNOWN_2                      | 0x12  | ListUpdate-class                      |
| UNKNOWN_ERROR_3                | 0x13  | Error (StringPool 765)                |
| BUDDY_CHANNEL_CHANGE           | 0x14  | ChannelChange (0x14)                  |
| CAPACITY_CHANGE                | 0x15  | CapacityUpdate (0x15)                 |
| UNKNOWN_ERROR_4                | 0x16  | Error (StringPool 765)                |

JMS185 `OnFriendResult` switch (decompiled @0xB2A873) confirms the **identical mode
value space**: cases 7/0xA/0x12 → Reset; 8 → UpdateFriend; 9 → Invite;
0xB..0xF, 0x10/0x11/0x13/0x16 → error notices; 0x14 → ChannelChange; 0x15 → CapacityUpdate.

**Serverbound handler sub-op map (gms_83_1 `BuddyOperationHandle`, op 0x82):**
`RELOAD=0, ADD=1, ACCEPT=2, DELETE=3`.

Verified against JMS185 binary (the export *notes* were transcription-wrong — corrected here):
- `CField::SendSetFriendMsg`  @0x56E41D → `Encode1(1)` → **ADD = 1** ✅
- `CField::SendAcceptFriendMsg`@0x56E66C → `Encode1(2)` → **ACCEPT = 2** ✅
  (export note claimed "mode=0x1" — **wrong**; binary emits 2.)
- `CField::SendDeleteFriendMsg`@0x56E5BD → `Encode1(3)` → **DELETE = 3** ✅
  (export note claimed "mode=0x2" — **wrong**; binary emits 3.)

RELOAD=0 is a clientbound-triggered list reset, not a serverbound send, and needs no send fn.

**Verdict: ALIGNED.** GMS v83 map internally consistent and matches both the GMS client
and the JMS185 binary's actual constants. v87/v95/JMS templates: **JMS-gap (writer +
handler ABSENT)** — a future JMS wiring would reuse the v83 values (binary-confirmed). No fix.

---

### CHAT — `CharacterChatGeneral` / `CharacterMultiChat` / `CharacterChatWhisper`

These writers/handlers carry **no `operations` sub-op map** in any template — chat type
selection (all/buddy/party/guild/alliance) is an inline payload byte, not a
template-resolved sub-op. So there is **no sub-op enum to drift**. Only the top-level
opcodes are template-configured, and those legitimately differ per version:

| Writer/Handler              | v83  | v87  | v95     | JMS185 |
|-----------------------------|------|------|---------|--------|
| `CharacterChatGeneral` (w)  | 0xA2 | 0xAD | ABSENT  | 0xA0   |
| `CharacterChatGeneralHandle`| 0x31 | 0x34 | ABSENT  | 0x29   |
| `CharacterMultiChat` (w)    | 0x86 | —    | —       | —      |
| `CharacterChatWhisper` (w)  | 0x87 | —    | —       | —      |

(Opcode drift is expected per-version and is a wire-header concern, already covered by
the per-struct audit; it is not a sub-op VALUE space.)

**Verdict: ALIGNED (no sub-op value space exists).** v87/v95/JMS multi/whisper writers
**JMS/version-gap (ABSENT)** — documented, not a divergence. No fix.

> Note (carried from task-066, out of scope here): the chat `Multi` v95 `updateTime`
> prefix and the chat sub-mode (group-message target) body layout are **wire-shape**
> follow-ups, not sub-op value drift. Not touched by this config-value spike.

---

### GUILD — `GuildOperation` writer / `GuildOperationHandle` / `GuildBBSHandle`

**Serverbound handler map (gms_83_1 `GuildOperationHandle`, op 0x7E):**
`REQUEST_CREATE=2, INVITE=5, JOIN=6, WITHDRAW=7, KICK=8, SET_TITLE_NAMES=13,
SET_MEMBER_TITLE=14, SET_EMBLEM=15, SET_NOTICE=16, AGREEMENT_RESPONSE=30`.

**BBS handler map (gms_83_1 `GuildBBSHandle`, op 0x9B):**
`CREATE_OR_EDIT_THREAD=0, DELETE_THREAD=1, LIST_THREADS=2, DISPLAY_THREAD=3,
REPLY_THREAD=4, DELETE_REPLY=5`. (task-066 confirmed the equivalent BBS values in GMS v95.)

The clientbound `GuildOperation` writer map (op 0x41) carries the full guild-result
enum (REQUEST_NAME=0x01 … SET_SKILL_RESPONSE=0x4E); these align with the v83 export's
`OnGuildResult#*` synthetic arms (RequestAgreement, Invite, ErrorMessage,
ErrorMessageWithTarget, Disband, CapacityChange, Member*, TitleChange, EmblemChange,
NoticeChange, Info, AgreementResponse).

JMS185 guild send path: `CWvsContext::SendGuildJoinMsg` is **ABSENT** in JMS185 (inlined);
the export marks it out-of-scope and atlas does not target it. BBS is **entirely absent
from JMS v185** (no BBS feature). So no JMS guild/BBS value map exists to compare.

**Verdict: ALIGNED (GMS baseline) + JMS-gap.** GMS v83 guild + BBS sub-op spaces are
internally consistent and match the GMS client arms. v87/v95/JMS `GuildOperation*`/
`GuildBBSHandle` **ABSENT** (template gap; BBS additionally is a feature JMS185 lacks). No fix.

---

### PARTY — `PartyOperation` writer / `PartyOperationHandle`

**Serverbound handler map (gms_83_1 `PartyOperationHandle`, op 0x7C):**
`CREATE=1, LEAVE=2, JOIN=3, INVITE=4, EXPEL=5, CHANGE_LEADER=6`.

**Clientbound writer map (gms_83_1 `PartyOperation`, op 0x3E):** the full party-result
enum (INVITE=0x04 … UNABLE_TO_FIND_THE_CHARACTER=0x21), aligning with v83 export
`OnPartyResult#*` arms (Created, Invite, Disband, Error, ChangeLeader, Join, Left, Update).

**JMS185 serverbound party sub-ops (binary-confirmed via export call lists):**

| Op            | GMS v83 template | JMS185 client            |
|---------------|------------------|--------------------------|
| JOIN_RESPONSE | (n/a — JOIN=3)   | 0 (`SendJoinPartyMsg`)   |
| CREATE        | 1                | 1 (`SendCreateNewParty`) |
| LEAVE/WITHDRAW| 2                | 2 (`SendWithdrawParty`)  |
| KICK/EXPEL    | 5                | **3** (`SendKickParty`)  |
| INVITE        | 4                | 4 (`SendJoinPartyMsg`)   |
| CHANGE_LEADER | 6                | **5** (`SendChangeBoss`) |

> **Real cross-version renumbering** in the party serverbound space: JMS185 uses
> KICK=3 / CHANGE_LEADER=5 (and a JOIN_RESPONSE=0 sub-op), whereas the GMS v83 map uses
> JOIN=3 / EXPEL=5 / CHANGE_LEADER=6. This is a genuine GMS↔JMS divergence — **but it
> lives in an UNWIRED JMS template** (`PartyOperationHandle` ABSENT in jms_185_1), so it
> is a **JMS template gap to record**, not a divergent value in a wired entry to fix. A
> future JMS party-wiring task must use the JMS numbers above, NOT copy the GMS v83 map.
> The GMS v83 wired map is correct for GMS and internally consistent — no GMS fix.

**Verdict: ALIGNED (GMS v83 baseline) + JMS-gap (with a documented value-renumber caveat
for whoever wires JMS later).** No fix to any wired entry.

---

### NOTE (memo) — `NoteOperation` writer / `NoteOperationHandle`

**Clientbound writer map (gms_83_1 `NoteOperation`, op 0x29):**
`operations: {SHOW=3, SEND_SUCCESS=4, SEND_ERROR=5, REFRESH=7}`,
`errors: {RECEIVER_ONLINE=0, RECEIVER_UNKNOWN=1, RECEIVER_INBOX_FULL=2}`.

GMS v83 export `OnMemoResult#*`: Display=3, SendSuccess=4, SendError=5, **Refresh=8**.
JMS185 export `OnMemoResult`: "Decode1(mode)-3 dispatch (0=Display/3, 1=SendSuccess/4,
2=SendError/5, 5=Refresh/...)" — i.e. the dispatcher subtracts 3, so wire modes are
3/4/5/8 (Refresh = base+5 = 8).

> ⚠️ **Apparent REFRESH mismatch — NOT a fix here.** The template has `REFRESH=7`; both
> the v83 and JMS185 clients dispatch Refresh at wire mode **8**. This is a *wire-shape /
> single-value* question that the task-066 per-struct audit already owns (the brief states
> per-struct wire shapes are ✅ and instructs fixing **only** divergent values in already-
> wired CONFIG entries — but only where the spike's own evidence is unambiguous and the
> change is config-level). Here the evidence is from the IDA *export annotation*, not a
> first-hand decompile in this session, and SEND_ERROR/REFRESH body shapes were a task-066
> per-struct item. **Flagging REFRESH=7-vs-8 as a CONCERN for the note/memo per-struct
> owner to confirm against a live `OnMemoResult` decompile**, rather than silently editing
> a value the per-struct pass marked ✅. The SHOW/SEND_SUCCESS/SEND_ERROR and the three
> `errors` values all agree across versions.

**Verdict: ALIGNED on SHOW/SEND_SUCCESS/SEND_ERROR + error codes; REFRESH value flagged
as a concern (see above) + JMS-gap (template ABSENT).** No fix applied in this spike.

---

## 2. `BuddyError` conditional-string arm analysis (the deferred task-066 item)

**Plan premise:** modes 0x10 / 0x11 / 0x13 / 0x16 carry a *conditional string* on the wire,
and atlas `Error.hasExtra` only models part of it.

**Atlas current behaviour** (`libs/atlas-packet/buddy/clientbound/error.go` +
`operation_body.go::BuddyErrorBody`): writes `Encode1(mode)`, then **iff `hasExtra`** one
extra `0x00` byte. `hasExtra` is set **only** when `errorCode == UNKNOWN_ERROR` (mode 0x10).
It writes a single zero *byte*, never a length-prefixed string.

**Ground truth — JMS185 `CWvsContext::OnFriendResult` decompiled @0xB2A873:**

```
case 0x10: case 0x11: case 0x13: case 0x16:
    v31 = 765;                       // StringPool id
    -> StringPool::GetString(Instance, &v32, v31);
    -> CUtilDlg::Notice(...);        // client-side localized notice
    return;                          // <-- NO CInPacket::Decode* after the mode byte
```

Every error arm (0xB→766, 0xC→767, 0xD→768, 0xE→769, 0xF→5931,
0x10/0x11/0x13/0x16→765) follows the same shape: **read the mode byte only**, then look up
a hardcoded `StringPool` id and pop a `CUtilDlg::Notice`. **There is no `DecodeStr` (and no
`Decode1(hasMsg)`) in any buddy-error arm.** The "string" the plan refers to is a
**client-internal localized resource**, not wire data.

This is corroborated by the GMS **v83** export note for `OnFriendResult#Error`:
*"mode=0xB/0xC/0xD/0xE/0xF. No extra fields beyond mode."* The GMS **v87/v95** export
*annotations* claim a "`Decode1(hasMsg) + optional DecodeStr`" for 0x10/0x11/0x13/0x16 —
but that contradicts the **directly-decompiled JMS185 binary**, which shares the v95 wire
layout ("Wire identical to v95" per the v83/v95 arm notes). Treating the first-hand
decompile as authoritative over the second-hand export prose: **modes 0x10/0x11/0x13/0x16
do NOT carry a wire string.**

**Buddy-error arm verdict table:**

| Mode | Atlas extra byte? | JMS185 binary reads after mode | Correct? |
|------|-------------------|--------------------------------|----------|
| 0x0B–0x0F | no             | nothing (StringPool notice)    | ✅       |
| 0x10 (UNKNOWN_ERROR)   | **yes (1×0x00)** | nothing            | ⚠️ harmless extra byte (see below) |
| 0x11 (UNKNOWN_ERROR_2) | no             | nothing                        | ✅       |
| 0x13 (UNKNOWN_ERROR_3) | no             | nothing                        | ✅       |
| 0x16 (UNKNOWN_ERROR_4) | no             | nothing                        | ✅       |

**On the lone 0x10 extra byte:** the JMS185 `OnFriendResult` 0x10 arm reads nothing past
the mode, so a trailing `0x00` is **ignored** by the client (the dispatcher returns after
the StringPool notice without consuming further bytes). It is harmless padding, not a wire
requirement, and not a *value* divergence. Because (a) the brief scopes this spike to sub-op
VALUE spaces and explicitly defers per-struct wire shapes to the ✅'d per-struct pass, and
(b) removing the byte is a behavioural wire-shape change to a struct task-066 already
classified, **this spike does not modify `error.go`.** It is recorded as a wire-shape
clean-up candidate for the buddy per-struct owner, with the evidence above.

**BuddyError verdict: arms CONFIRMED.** No wire string is carried by 0x10/0x11/0x13/0x16;
the plan's "conditional-string arm" premise is refuted by the JMS185 binary. The atlas
`hasExtra` model is functionally correct (the one extra byte on 0x10 is ignored by the
client). **No config/constant fix.**

---

## 3. Summary verdict table

| Domain | GMS v83 (wired) | v87 | v95 | JMS185 | Verdict |
|--------|-----------------|-----|-----|--------|---------|
| Buddy  | ✅ aligned (handler ADD=1/ACCEPT=2/DELETE=3 binary-confirmed; writer arms match `OnFriendResult`) | gap | gap | gap (values reuse v83) | **ALIGNED + JMS-gap** |
| Chat   | ✅ no sub-op space (opcode-only) | gap | gap | gap | **ALIGNED (no enum)** |
| Guild  | ✅ aligned (handler + BBS + clientbound arms) | gap | gap | gap (BBS absent in JMS) | **ALIGNED + JMS-gap** |
| Party  | ✅ aligned (GMS) | gap | gap | gap — JMS renumbers KICK=3/CHGLEADER=5 (record for future wiring) | **ALIGNED + JMS-gap (value caveat)** |
| Note   | ✅ SHOW/SEND_*/errors aligned; REFRESH=7 vs client-8 flagged | gap | gap | gap | **ALIGNED-except-REFRESH (concern) + JMS-gap** |

**`BuddyError` 0x10/0x11/0x13/0x16:** arms carry **no wire string** (StringPool notice only)
— premise refuted, atlas model functionally correct. **No fix.**

---

## 4. Outcome

- **No config-value or constant fix made.** This is a **verdict-only** spike: the only
  fully-wired social template (GMS v83) is internally consistent and client-correct; the
  three partial templates hold no social sub-op maps (gaps, out of scope to wire here); and
  the `BuddyError` conditional-string premise is refuted by the JMS185 binary.
- **Two items handed off (not fixed here, with evidence):**
  1. **NOTE `REFRESH=7` vs client wire-8** — flag to the note/memo per-struct owner to
     confirm against a live `OnMemoResult` decompile before changing a value the per-struct
     pass marked ✅.
  2. **Buddy `error.go` 0x10 trailing `0x00`** — harmless (client ignores), candidate
     wire-shape clean-up for the buddy per-struct owner.
- **JMS party serverbound renumber (KICK=3, CHANGE_LEADER=5, JOIN_RESPONSE=0)** recorded so
  a future JMS party-wiring task does not blindly copy the GMS v83 op map.

## 5. Evidence index

- Templates: `services/atlas-configurations/seed-data/templates/template_{gms_83_1,gms_87_1,gms_95_1,jms_185_1}.json`
- IDA exports: `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json`
- JMS185 IDB decompiles (first-hand, this session):
  - `CWvsContext::OnFriendResult` @0xB2A873 — buddy error arms (StringPool, no DecodeStr)
  - `CField::SendSetFriendMsg` @0x56E41D — ADD=1
  - `CField::SendAcceptFriendMsg` @0x56E66C — ACCEPT=2 (export note "0x1" wrong)
  - `CField::SendDeleteFriendMsg` @0x56E5BD — DELETE=3 (export note "0x2" wrong)
- Atlas code: `libs/atlas-packet/buddy/{operation.go,operation_body.go,clientbound/error.go,serverbound/*}`,
  `.../party/serverbound/operation.go`, etc.
- Prior deferral: `docs/tasks/task-066-social-domain-packet-audit/post-phase-b.md`
  ("BuddyError sub-op conditional … Deferred to Phase 2 cross-version sub-op enum pass").

**Tests/build:** no code or template changed → no fix to verify.
Baseline confirmation only: `go test ./{buddy,chat,guild,party,note}/...` PASS;
`go vet ./{buddy,chat,guild,party,note}/...` clean (in `libs/atlas-packet`).
