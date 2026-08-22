# Duey: receive-only dialog, and the missing send confirmation

Two defects found while live-testing the NPC entry point, plus one gate
failure they surfaced. All three are fixed on this branch
(`5b99d25a4`, `94cb58a6f`, `1610d359c`).

## 1. The NPC dialog had no Send tab (5b99d25a4)

Symptom: talking to Duey opened a window that could only collect parcels.

`PARCEL[OPEN]`'s leading bool is `CParcelDlg`'s `m_nMode`, **not** a "quick
delivery is available" flag. design.md §5.2 recorded the construction
(`CParcelDlg(bool ? 2 : 0)`, v83 @0x6f5b32) but not what the mode does to the
tab set, so the codec field was named `quickEnabled` and atlas-channel derived
it per tenant from a classification-533 version gate. That gate is true for
every tenant in span, so every NPC open built `CParcelDlg(2)`.

`CParcelDlg::OnCreate`'s tab loop (v83 @0x6f4a50, v95 @0x691d87) inserts:
mode 0 → tabs 0,1,2 (Receive + Send + QuickSend); mode 1 → tab 2 only (what
`OPEN_QUICK` builds); mode 2 → tab 0 only (Receive, no way to send). Tab 0 is
Receive — the v95 ctor names the button arrays `m_pBtReceive[2]` /
`m_pBtSend[4]` / `m_pBtQuickSend[4]`, and `SetCtrl`'s page-0 branch drives
exactly that 2-button group plus the parcel list.

Fix: field renamed `receiveOnly` through the codec, the channel passes `false`
unconditionally, and the per-tenant helper is deleted. No wire bytes changed —
the per-version fixtures still encode the same byte, so no cell re-verification
was needed.

## 2. A successful send produced no response at all (94cb58a6f)

`PARCEL[SUCCESSFULLY_SENT]` (0x12) had a codec and per-version fixtures but no
caller. The client disables its send controls on submit and only 0x12
re-enables them (`OnPacket` default arm → `NoticeResult` + `SetCtrlEnabled(1)`
+ `ResetSendInfo`/`CloseParcelDlg`, v83 @0x6f579d), so the sender was left with
a dead dialog and no confirmation.

`accept_to_parcel` is the last step of `parcel_send`, so atlas-parcel's
`handleAcceptToParcel` now emits `PARCEL_SENT` on the existing status topic
addressed to the sender (`AcceptToParcelCommandBody.CharacterId`), and
atlas-channel answers with `PARCEL[SUCCESSFULLY_SENT]` through
`IfPresentByCharacterId`. This reuses the `PARCEL_ARRIVED` → `ALARM_NAMED`
shape, so the channel needs no view of saga completion. Accepted trade: a
replayed accept re-emits the notice (the row create is idempotent, the notice
is not).

## 3. `sparse baseline scoping` was failing (1610d359c)

Pre-existing on the branch, unrelated to the above but blocking the gate: the
four parcel topic vars and `DB_NAME=atlas-parcel` were unsuffixed in
`overlays/pr-sparse`, which addresses a namespace nobody publishes to rather
than the baseline's. Regenerated from `gen-topic-config.sh` and
`gen-db-name-suffix.sh`. Guard now reports 174/174 topics and 37/37 DB_NAMEs.

## Next step

Re-run the flagless `tools/verify.sh` (the `--quick` run before these commits
failed only on §3). Then code review, then PR. Nothing here depends on
conversation history — the addresses above and the commit messages carry the
reasoning.
