# Spike: Messenger serverbound enum (B3.1) + declineMode sub-enum (B3.2)

Two verification spikes against JMS v185 (loaded IDB). Both are VERDICT tasks:
decompile, compare to Atlas, record verdict, fix-in-task ONLY on a genuine divergence.

**Both verdicts: VERIFIED — NO FIX.** No code changed; this note is the deliverable.

---

## Task B3.1 — messenger serverbound `Operation` full enum

**File under audit:** `libs/atlas-packet/messenger/serverbound/operation.go`
**Plan premise:** "confirm no modes beyond 0/2/3/5/6; verify atlas-messengers routing matches."

### Atlas coverage

`Operation.Decode` reads a single leading `mode byte`; the handler then re-dispatches on
that byte. Routing is config-driven, not hardcoded — `atlas-channel`
`socket/handler/messenger_operation.go::MessengerOperationHandleFunc` reads the mode and
matches it against the `operations` code map seeded in
`services/atlas-configurations/seed-data/templates/template_gms_83_1.json` (opcode `0x7A`):

```
ANSWER_INVITE  : 0   -> messengersb.OperationAnswerInvite  { Decode4 messengerId }
CLOSE          : 2   -> (no body) Leave
INVITE         : 3   -> messengersb.OperationInvite        { DecodeStr target }
DECLINE_INVITE : 5   -> messengersb.OperationDeclineInvite { DecodeStr from, DecodeStr my, Decode1 }
CHAT           : 6   -> messengersb.OperationChat          { DecodeStr msg }
```

So Atlas handles exactly the set **{0, 2, 3, 5, 6}**.

### IDA findings (JMS185) — every client send site

The serverbound messenger opcode in JMS185 is **`0x7B` (123)** (Atlas's GMS83 seed uses
`0x7A` — version-specific opcode mapping, irrelevant to the mode enum, which the server
echoes verbatim). The client builds the op packet with `COutPacket(0x7B); Encode1(mode); …`.
All five emit sites:

| FName @ addr | mode | body after Encode1(mode) |
|---|---|---|
| `CUIMessenger::OnCreate` @ `0x8e16c2` | **0** (enter/create) | `Encode4` messengerId |
| `CUIMessenger::OnDestroy` @ `0x8e173b` | **2** (close) | (none) |
| `CUIMessenger::SendInviteMsg` @ `0x8e4f02` | **3** (invite) | `EncodeStr` target name |
| `CUIMessenger::OnInvite` (decline branch) @ `0x8e4777` | **5** (decline) | `EncodeStr` fromName, `EncodeStr` myName, `Encode1(1)` |
| `CUIMessenger::ProcessChat` @ `0x8e50ac` | **6** (chat) | `EncodeStr` message |

The complete serverbound mode set the client emits is **{0, 2, 3, 5, 6}**. Modes **1 and 4
are server→client only** (`OnSelfEnterResult`=1, `OnInviteResult`=4 — seen in the clientbound
dispatcher `CUIMessenger::OnPacket` @ `0x8e447e`, which switches 0/1/2/3/4/5/6/7/8) and are
never sent by the client.

### Verdict — B3.1: VERIFIED, NO FIX

Atlas's serverbound enum and routing cover exactly the client's mode set {0,2,3,5,6}. No
mode is missing; no client-emitted mode is unhandled. The plan premise holds.

**Minor cosmetic note (not a divergence, no fix):** the mode-5 decline body's trailing byte
is named `alwaysZero` in `OperationDeclineInvite`, but the client (`OnInvite` decline branch
@ `0x8e47c7`) emits `Encode1(1)` — i.e. it sends **1**, not 0. Atlas reads-and-logs this
field and never gates on it, so there is no functional impact. The field name is mildly
inaccurate; left as-is to keep the spike scoped to enum coverage.

---

## Task B3.2 — messenger `declineMode` sub-enum

**File under audit:** `libs/atlas-packet/messenger/clientbound/invite_declined.go`
**Plan premise:** "In `OnBlocked` (mode=5) confirm `if v3` -> StringPool 0x31A vs 0x31B;
confirm only 0/1 vs more."

### Atlas coverage

`InviteDeclined` wire shape: `WriteByte(mode)`, `WriteAsciiString(message)`,
`WriteByte(declineMode)`. `declineMode` is a `byte` that round-trips faithfully
(`invite_declined_test.go::TestMessengerInviteDeclinedRoundTrip`). mode=5 is the clientbound
`OnBlocked` discriminator (per `OnPacket` @ `0x8e447e` case 5).

### IDA findings (JMS185)

`CUIMessenger::OnBlocked` @ **`0x8e4601`**, decompiled read order:

```
DecodeStr  v12            ; blocker/target name string
v2 = Decode1(iPacket)     ; the declineMode byte
if ( v2 )                 ; pure boolean test
    StringPool::GetString(..., 0x341)   ; "... has declined / is offline" variant A
else
    StringPool::GetString(..., 0x342)   ; variant B
ZXString::Format(result, <chosen template>, name)
```

The declineMode byte selects between exactly **two** StringPool templates via a boolean
`if (v2)`. There is no `switch`, no `>= n` comparison, no third branch — the value space is
**{0, non-zero}** (effectively 0/1). The sibling `OnInviteResult` @ `0x8e4515` uses the
identical boolean idiom (`if (v2)` -> 0x33F vs 0x340), corroborating the pattern.

**StringPool ID note:** the plan cited `0x31A` / `0x31B`; in JMS185 the actual IDs are
**`0x341` / `0x342`** — version drift in the StringPool table. The *structure* (single byte,
two-way boolean branch) is identical, which is what governs Atlas's model.

### Verdict — B3.2: VERIFIED, NO FIX

The declineMode value space is boolean (0 vs non-zero selecting two message templates).
Atlas models declineMode as a `byte` that round-trips any value faithfully — it neither
truncates nor mis-orders, so it correctly covers the client's value space. No sub-enum
divergence. No fix needed.

---

## Summary

| Task | Client value space (IDA) | Atlas model | Verdict |
|---|---|---|---|
| B3.1 serverbound enum | {0,2,3,5,6} (5 emit sites, all FName@addr above) | config-driven {0,2,3,5,6} + per-mode bodies | VERIFIED, NO FIX |
| B3.2 declineMode | boolean 0/non-zero (`OnBlocked` @ 0x8e4601, 0x341 vs 0x342) | `byte` round-trip | VERIFIED, NO FIX |

Baseline tests green at audit time:
`go test ./messenger/...` -> `clientbound ok`, `serverbound ok`. No code changed.
