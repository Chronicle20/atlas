# Task 6 IDA Re-derivation Audit — Parcel / DueyAction mode-byte authority

Read-only independent re-derivation against the live IDBs, performed without
reading `task-6-report.md` beforehand. All addresses below were read directly
from the IDBs in this session (sessions per `idb_list`: v72=f2a2e7c1,
v79=f36df4cd, v83=41f09cce, v87=0d9cf8b6, v92=e3328b84, jms_v185=05eb9c27).

## 1. v79 DUEY_ACTION opcode (highest risk)

Decompiled all four send-sites directly on v79 (session `f36df4cd`,
`GMS_v79_1_DEVM.exe.i64`):

- `CTabSend::SendParcel` @0x68170f — `COutPacket::COutPacket(v16, 63)` /*0x681826*/, `Encode1(v16, 2u)` (SEND)
- `CTabReceive::ReceiveParcel` @0x67ed0c — `COutPacket::COutPacket(v11, 63)` /*0x67edaf*/, `Encode1(v11, 4u)` (RECEIVE)
- `CTabReceive::DiscardParcel` (sub_67EE2C, unnamed in this IDB) @0x67ee2c — `COutPacket::COutPacket(v11, 63)` /*0x67ee96*/, `Encode1(v11, 5u)` (DISCARD)
- `CParcelDlg::CloseParcelDlg` (sub_6836B0, unnamed) @0x6836b0 — `COutPacket::COutPacket(v3, 63)` /*0x6836c5*/, `Encode1(v3, 7u)` (CLOSE)

All four independently construct `COutPacket(this, 63)` = 0x3F. **Confirmed: v79
DUEY_ACTION opcode = 63 / 0x3F**, matching `duey_action.yaml`'s registry entry
and the implementer's claimed addresses exactly (same four addresses, same
four mode bytes).

## 2. duey_action.yaml mode bytes — verified on 4 columns (v72, v79, v83, jms_v185)

| Version | opcode | SEND addr | RECEIVE addr | DISCARD addr | CLOSE addr | Modes read |
|---|---|---|---|---|---|---|
| gms_v72 (f2a2e7c1) | 64 (0x40) | 0x65d940 | 0x65af41 | 0x65b061 | 0x65f8e1 | 2/4/5/7 — matches |
| gms_v79 (f36df4cd) | 63 (0x3F) | 0x68170f | 0x67ed0c | 0x67ee2c | 0x6836b0 | 2/4/5/7 — matches |
| gms_v83 (41f09cce) | 65 (0x41) | 0x6f36a8 (+ quick-send 0x6f1df5) | 0x6f0ca3 (own IDB, not the v72 addr the yaml cites) | 0x6f0dc3 | 0x6f5691 | 2/4/5/7 — matches |
| jms_v185 (05eb9c27) | 57 (0x39) | 0x753aa5 | 0x75110b | 0x75122b | 0x7559c8 | 3/5/6/8 — matches, confirms uniform **+1** shift |

Note on v83 RECEIVE: the committed yaml cites `CTabReceive::ReceiveParcel
@0x65AF41` for the RECEIVE derivation, which is actually the **v72** address
(same symbol name, different build) — not a v83 address. I independently
located and decompiled v83's own `CTabReceive::ReceiveParcel` at **0x6f0ca3**
(`func_query` on session 41f09cce) and it sends `COutPacket(&v16, 0x41)` /
`Encode1(&v16, 4u)` — mode 4, opcode 0x41 (65), matching the committed value.
This is a **citation slip in the yaml's provenance comment** (pointing at the
v72 address under a "v83 SOURCE OF TRUTH" heading), not a value error — the
mode byte itself (RECEIVE=4) is correct on both v72 and v83. Flagging as a
non-blocking documentation nit, not a BLOCKING byte disagreement.

All four columns' mode bytes (SEND=2, RECEIVE=4, DISCARD=5, CLOSE=7, jms
+1-shifted to 3/5/6/8) match the committed yaml exactly, at the exact
addresses the implementer's provenance notes cite (except the v83/v72
citation slip above).

## 3. parcel.yaml per-version columns — verified on v83 (anchor) + v87 + v92

`CParcelDlg::OnPacket` decompiled on all three:

- v83 (41f09cce) @0x6f56ea — explicit cases 8, 23, 24, 25, 26, 27; default arm
  calls `NoticeResult` then `SetCtrlEnabled`, with `a1==18` gating
  `CloseParcelDlg`.
- v87 (0d9cf8b6) @0x7346db — same explicit case set {8,23,24,25,26,27}, same
  body shapes (Decode4+Decode1 for 23, DecodeStr+Decode1+CUIFadeYesNo for 25,
  etc.), default arm gated on `iPacket==18`.
- v92 (e3328b84) @0x689190 — same explicit case set {8,23,24,25,26,27}, same
  shapes, default arm gated on `v1==18`.

All three are structurally and value-identical. **No divergence found.**

Notice-range (`NoticeResult`) cross-check, covering exactly the bodyless keys
9–16, 18–22, 26(n/a), 28 the task called out:

- v83 `NoticeResult` @0x6f5be2: dispatches case values
  {10,11,12,13,14,15,16} then {17,18,19(alias 28's string),21,22,28} — 13
  distinct case values, none at 9 or 20 (those two keys are the
  silent-"enable actions" arms — no notice text, just `SetCtrlEnabled`,
  consistent with the yaml's OPEN/RECV_ENABLE_ACTIONS naming).
- v87 `sub_734BF3` @0x734bf3: identical case-value set
  {10,11,12,13,14,15,16,17,18,19(alias 28),21,22,28} — different StringPool
  numeric IDs (expected, per-build string table), same dispatch shape.
- v92 `sub_6807D0` @0x6807d0: identical case-value set (10–19 inclusive, then
  21,22, with 19 and 28 sharing the same string ID 3983) — same dispatch
  shape.

All string-emitting case values byte-identical to v83 across both additional
versions checked, including the notice range. **No divergence found; the
byte-identical claim holds for the two versions I re-checked beyond the
anchor.**

Not independently checked in this audit: v72, v79, v84, v95 columns of
`parcel.yaml` (the task asked for "at least two" beyond what's already
established as anchor; v87 and v92 satisfy that). These four remain
unverified by me — I did not re-derive their `parcel.yaml` values, only their
`duey_action.yaml` values (v72, v79) as noted above.

## 4. jms_v185 partial column — 7 keys confirmed, shortfall confirmed genuine

`CParcelDlg::OnPacket` @0x755a21 on jms_v185 (05eb9c27):

| yaml key | claimed mode | confirmed at | body shape match |
|---|---|---|---|
| OPEN | 10 | case 10 @0x755e42+ | CUniqueModeless check, `CParcelDlg::CParcelDlg(...)`, `SetParcelDlg` — matches v83 case 8 shape |
| SUCCESSFULLY_SENT | 19 | default arm, `iPacket==19` @0x755ad9 | gates `CloseParcelDlg` exactly like v83's `a1==18` — same side-effect check, +1 shifted |
| PARCEL_REMOVED | 24 | case 24 @0x755d43+ | Decode4+Decode1, branch on `==3`, matches v83 case 23 shape |
| PARCEL_ARRIVED | 25 | case 25 @0x755cca+ | decode + AddNewParcel-equivalent, matches v83 case 24 shape |
| ALARM_NAMED | 26 | case 26 @0x755bf5+ | DecodeStr+Decode1+CUIFadeYesNo::CreateParcelAlarm-equivalent, matches v83 case 25 shape |
| OPEN_QUICK | 27 | case 27 @0x755bcf+ | CUniqueModeless check, `CParcelDlg::CParcelDlg(v9, 1)`, matches v83 case 26 shape |
| ALARM_GENERIC | 28 | case 28 @0x755b52+ | Decode1+CUIFadeYesNo+default string (`sStrDefault`), matches v83 case 27 shape |

All 7 populated jms_v185 values in `parcel.yaml` are confirmed correct by
independent decompile of the same function.

Additionally observed: **case 9** in jms_v185's `OnPacket` is a genuinely new
arm with no v83 counterpart — decodes a signed int, and on a negative value
sends an *outbound* ack packet (`COutPacket(v35, 57)`, mode 1) back to the
server. This corroborates the yaml's claim that JMS added a new arm at 9,
pushing OPEN to 10.

**Shortfall verification** — `sub_755FD3` (jms_v185's `NoticeResult`
equivalent) @0x755fd3 dispatches case values {12,13,14,15,16,17,18} (first
switch) and {19,20(alias 29's string via goto),22,23,29} (second switch) = 12
distinct case values total (12,13,14,15,16,17,18,19,20,22,23,29). v83's
`NoticeResult` dispatches 13 distinct case values
(10,11,12,13,14,15,16,17,18,19,21,22,28). **12 vs 13 — one fewer distinct
slot in jms_v185, exactly as the implementer's report claims.** This is not
an artifact of an incomplete search; both switches were read in full and the
counts are exact.

I attempted to see whether the numeric offset alone could resolve a 1:1
mapping: the low end of JMS's range (12–18) is a clean +2 shift from v83's
10–16 arm-for-arm, but from JMS 19 onward the mapping breaks — JMS has 19, 20,
22, 23, 29 where a uniform +2 (from v83's 17,18,19,21,22,28) would predict
19, 20, 21, 23, 24, 30. JMS is missing 21 and 24/30 doesn't appear; 29 doesn't
fit a uniform offset from 28 either. A purely numeric extrapolation is
therefore unsound, matching the implementer's reasoning.

Unlike v83's IDB — which carries symbolic `SP_NNNN_ENGLISH_TEXT` labels on
its StringPool ids (evidently from a prior localization/labeling pass) — the
jms_v185 IDB's StringPool ids in this switch are bare integers (0x235,
4041, 4036, 4037, 4038, 4039, 4040, 4034, 4042, 4043, 4044, etc.) with no
attached text in this session. Confirming the semantic identity of any one
JMS slot against a specific v83 English string would require pulling the
actual (Japanese) string data from the JMS string-pool resource and having
it translated/correlated — a step outside what this IDB alone can produce,
and outside this audit's scope/effort budget. **I could not derive the
remaining 14-key mapping either, and did not find grounds to overturn the
"underdetermined" ruling.** The shortfall is real; the "leave 14 keys unset"
decision stands as correct, not an excuse.

## Summary of disagreements

None on any mode byte. One non-blocking documentation nit: the
`duey_action.yaml` provenance comment for the v83 RECEIVE derivation cites a
v72 address (0x65AF41) under a "v83 SOURCE OF TRUTH" heading; v83's own
`CTabReceive::ReceiveParcel` lives at 0x6f0ca3 and independently confirms the
same mode byte (4), so no value is wrong — only the cited address is from the
wrong build.
