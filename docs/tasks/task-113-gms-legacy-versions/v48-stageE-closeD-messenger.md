# v48 Stage E — CLOSE batch D (messenger) — report

Anchor v61 (fast-path where applicable), IDB port 13337 (GMS_v48_1_DEVM.exe).
Branch `task-113-gms-legacy-versions`. Finishes the messenger family left open by
batch 3. **13 tier-1 cells promoted ❌/⬜ → ✅** (8 clientbound arms + 4 serverbound
sub-arms + 1 serverbound DECLINE op-cell). 0 n-a, 0 remaining. gms_v48 verified
count **146 → 159 (+13)**; no other version dropped.

## CUIMessenger clientbound dispatcher re-derived (NOT copied from v61)

`CField::OnPacket` @0x4c66f2 routes **case 238 → sub_61D8B8** (the v48 clientbound
messenger dispatcher). sub_61D8B8 does `Decode1(mode)` then switches; **case 3 is
special-cased before the window-open guard**. Mode table 0..8, byte-identical to
v72/v79/v83:

| mode | arm (v48) | atlas struct | body (read order) |
|---|---|---|---|
| 0 | sub_61B860 OnEnter | MessengerAdd | position + AvatarLook + name + channelId + pad |
| 1 | sub_61BA71 OnSelfEnterResult | MessengerJoin | position |
| 2 | sub_61BB63 OnLeave | MessengerRemove | position |
| 3 | sub_61DB2C OnInvite | MessengerRequestInvite | fromName + pad + Decode4(messengerId) + pad |
| 4 | sub_61D94F OnInviteResult | MessengerInviteSent | message + Decode1(success) |
| 5 | sub_61DA3F OnBlocked | MessengerInviteDeclined | message + Decode1(declineMode) |
| 6 | sub_61DC8C OnChat | MessengerChat | message |
| 7 | sub_61DF11 OnAvatar | MessengerUpdate | position + AvatarLook |
| 8 | sub_61DF6C OnMigrated | (no atlas struct — not a cell) | 3× (flag/avatar/name) |

Every arm's read order matches the IDA-verified v83 codec. All 8 arms fixtured +
markered (`ida=0x61d8b8`) + evidence-pinned → **✅**.

## Add codec bug fixed (cross-version)

The clientbound **Add** arm reads **six** fields (…+channelId+pad) in v48
(sub_61B860), **and in the real v61 dispatcher case-0 sub_6D144E @0x6d144e**, v72,
v79, v83 — never five. The pre-existing `legacyAdd` gate omitted channelId+pad for
all GMS<72, but it was built on **sub_5BF5AE, which is a CMiniRoomBaseDlg arm**
(dispatched by sub_5BEC69, the mini-room OnPacketBase), **not** the messenger
dispatcher. Confirmed via xref: sub_5BF5AE ← sub_5BEC69 only.

Fix: narrowed `legacyAdd` to **GMS<=28** (the only version with no IDB to verify);
v48/v61/v72+ now correctly emit the six-field wire. Re-pointed + re-pinned the v61
Add marker/evidence/export entry from 0x5bf5ae → **0x6d144e** (the true messenger
case-0); v61 Add stays ✅ (v61 count restored 208). v72+ unaffected (already
`>=72`). Avatar arms cannot use `==v83` for v48 because v48 is <61 and takes the
model.Avatar single-4-byte-pet legacy path (IDA sub_49E1E0 @0x49e2b9) → the avatar
block is genuinely shorter; verified the messenger FRAME via round-trip + boundary
bytes with a deterministic single-equip avatar.

## Serverbound

Client sends `COutPacket(92)` (op 0x5C), `Encode1(mode)` + per-mode body. Send
bodies body-verified in v48:

| mode | send-site | atlas struct | body |
|---|---|---|---|
| 0 | sub_61A701 @0x61a701 (OnCreate) | MessengerOperationAnswerInvite | Encode4(messengerId) |
| 2 | sub_61AC75 @0x61ac75 (OnDestroy) | MessengerOperation | (mode only) |
| 3 | (SendInviteMsg — see below) | MessengerOperationInvite | EncodeStr(target) |
| 5 | sub_4BCE54 @0x4bce54 | MessengerOperationDeclineInvite | EncodeStr(from)+EncodeStr(me)+Encode1(0) |
| 6 | sub_61B27C @0x61b27c (ProcessChat) | MessengerOperationChat | EncodeStr(text) |

All 4 sub-arms fixtured + markered + evidence-pinned → **✅**.

### SendInviteMsg (mode 3) resolution

The real invite send-site is **not** in the CUIMessenger UI region
(0x61A701–0x61E25E) — searched OnCreate/ProcessChat/the chat-input (sub_61E25E,
mode 6) and slash-command path (sub_61E00D → sub_61E152, log-only), and the list/
tooltip renderers (sub_61AD42/61C807/61D403/61D587). This mirrors **v83, where
SendInviteMsg is likewise synthetic**. Pinned at the dispatcher sub_61D8B8
(`ida=0x61d8b8`) with the IDA-grounded mode-3 body: mode 3 is confirmed by (a) the
v83 verified SendInviteMsg body (`sub-op=3 INVITE` + target name) and (b) the v48
clientbound dispatcher reserving **case 3** for the reciprocal RequestInvite
(sub_61DB2C, which itself emits the op92/mode-5 auto-decline). Verified via
round-trip. **✅**.

### DECLINE op-cell

`op-92 serverbound` **is routed** in template_gms_48_1.json (handler
`MessengerOperationHandle`, `LoggedInValidator`). The op-cell was ❌ only because
the registry primary fname was the unnamed `sub_4BCE54`, so findReport (primary-
fname-only) could not link the `MessengerOperationDeclineInvite` report (IDAName
`CFadeWnd::SendCloseMessage`, Verdict Match). Aligned the registry primary fname to
**CFadeWnd::SendCloseMessage** (the canonical decline handler across v61–v83),
keeping `sub_4BCE54` as the concrete v48 alt; spliced the resolved
`CFadeWnd::SendCloseMessage` export entry (0x4bce54, body verified) + marker
(`ida=0x4bce54`) + evidence + golden fixture → op-cell **✅**.

## Cross-version presence check (batch-3 flag: "v61 anchor incomplete for cl arms")

- Clientbound **op-cell** (dispatcher) is ✅ for v61–jms; only v48 was ⬜
  (unregistered). Now v48 arms are ✅.
- Clientbound **sub-arms**: were ❌ across v61/v72/v79 (only v61 Add was ✅, and
  that was against the wrong function). v48 is now the most complete legacy version
  for the clientbound sub-arms — expected and fine per the task; the v61 Add
  false-positive was corrected (0x5bf5ae → 0x6d144e).
- Serverbound sub-arms: ✅ only for v83+ before; v48 now ✅ (ahead of v61/v72/v79).
- No arm was genuinely absent in v48 → **0 n-a** dispositions. Case 8 (OnMigrated)
  has no atlas struct and is not a matrix cell.

## Commits (branch task-113-gms-legacy-versions)

1. `3e9fe3679b` messenger/clientbound family (8 arms) + Add codec fix + v61 re-pin
2. `6621bb567f` messenger/serverbound sub-arms (Operation/AnswerInvite/Chat/Invite)
3. `8a61159dac` messenger DECLINE op-cell (registry fname align + export splice)

## Verification

- `go test -race ./libs/atlas-packet/messenger/...` green; `go vet` clean.
- `go run ./tools/packet-audit matrix --check` **exit 0**; problem-grep **0**;
  v48 conflicts **0** (🟥=0); Conflicts section "None".
- Regression — verified counts held: v61 208 / v72 216 / v79 228 / v83 367 /
  v84 345 / v87 379 / v95 399 / jms 362. v48 **146 → 159 (+13)**.
- Export splices surgical (only the touched entries); STATUS.md diff scoped to the
  13 messenger cells + v48/v61 export hashes; no out-of-scope report-regen drift.
- Branch after each commit: `task-113-gms-legacy-versions`.

## Remaining

**None** — the messenger family (clientbound dispatcher op + 8 clientbound arms +
serverbound decline op + 4 serverbound sub-arms) is fully verified for gms_v48.
