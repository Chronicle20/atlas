# Task-081 — buddy family (`CWvsContext::OnFriendResult`) live mode-map

Grinding the `#`-per-mode bucket faithfully, family by family. This is the buddy
family, traced live against the v83 IDB (`OnFriendResult@0xa3f2e8`).

## v83 mode-switch (verified `switch(Decode1 mode)`)

| modes | reads (after mode byte) | `#` entry | Atlas writer |
|---|---|---|---|
| 7, 8, 0xA, 0x12 | `CFriend::UpdateFriend@0xa400ad`: `Decode4 charId, Decode1 flag` + `GW_Friend` buffer | `#Update` / `#ListUpdate` | update.go / list_update.go |
| 9 | `Decode4 originatorId, DecodeStr name, GW_Friend(39) buffer, Decode1 inShop` | `#Invite` | invite.go |
| 0xB–0x13, 0x16 | `Decode1 errCode, if(Decode1) DecodeStr name` | `#Error` | error.go |
| 0x14 | `Decode4 charId, Decode1 channelId, Decode4 mapId` | `#ChannelChange` ✅ | channel_change.go |
| 0x15 | `Decode1 capacity` | `#CapacityUpdate` ✅ | capacity_update.go |

## Findings

1. **Atlas writers are correct and IDA-verified — the ❌ are stale baselines.**
   - `invite.go` version-gates `jobId/level` (`hasJobLevel := Region!="GMS" ||
     Major>=87`) with an IDA-cited comment (v83 omits them; v87/v95/jms include
     them between name and the GW_Friend buffer). The audit's "width mismatch"
     was the **wrong baseline** (a phantom `count + buddy-loop` that does not
     exist in v83 mode 9), NOT an Atlas bug. **BuddyInvite is not a wire bug.**
   - `#ChannelChange` / `#CapacityUpdate` baselines already match → ✅.

2. **This bucket is dominated by sub-structs + loops + version-gating.**
   - `update.go` writes `[mode, charId, GW_Friend(bm.Encode), inShop]` — sub-struct.
   - `list_update.go` writes `[mode, count, loop<GW_Friend>, loop<inShop ints>]` —
     loop + sub-struct.
   - `invite.go` writes the `GW_Friend` buffer (`model.Buddy.Encode`) — sub-struct.
   - Only `error.go` (`[mode, if(hasExtra) byte]`) is sub-struct-free.

   The audit defers sub-struct field comparison (🔍 `sub-struct: b — see
   _substruct/`). So correcting these baselines moves them **❌ → 🔍** (an honest
   "structure verified present, field-level comparison deferred"), not always
   ❌ → ✅. That is still real progress against the ❌ count, but the residual 🔍
   reflects a genuine analyzer limitation (sub-struct/GW_* buffer comparison),
   not a bug.

## Net for the family
Atlas buddy encoders are correct (IDA-verified, version-gated). The ❌ are
baseline-tracing defects + sub-struct deferrals. No Atlas wire bug. Faithfully
clearing them requires either (a) rebuilding each baseline including a faithful
`GW_Friend` sub-struct representation, or (b) a tool change to field-compare the
GW_* buffers — tracked as the sub-struct follow-up. The flat ❌→✅ flips in this
family are limited to `#Error` (sub-struct-free); the rest resolve to 🔍.
