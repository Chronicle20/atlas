# bug: after the ordering + 1-based-label fixes, /find still names "Scania - 1" for a channel-1 target

Follow-on to `bug-find-channel-always-zero.md`. Reported after `ceb83cc09`
(sort the channel list) and `682739570` (1-based channel labels) were deployed.

**Reproduced:** not by me — this is the reporter's live observation. Environment
confirmed: tenant `625de849-e34f-45c8-95e6-b8e794774422`, GMS 83.1, ephemeral env
`atlas-pr-1407`. All four relevant services run
`ghcr.io/.../pr-1407-9632c91` (branch head, both fixes included); the pods
started `2026-08-19T13:09:2xZ`, and the observed `/find` calls are at
`13:34:18Z` and `13:34:21Z` — **after** the rollout, so the fixed build is what
answered them.

**Observed (reporter):** two characters, one on the channel they call "1" and one
on the channel they call "2"; `/find` reports `<name> is at 'Scania - 1'` for
both.

**Expected:** the channel-1 (internal id 1) target renders `Scania - 2`.

## Root cause: NOT established. The server is provably correct on this build.

The whole server-side chain is confirmed from live logs, not inferred:

1. **atlas-world channel registry** (live REST, `/api/worlds/?include=channels`):
   world 0 has channels `{id 0, port 8301}` and `{id 1, port 7901}`. Two channels.
2. **atlas-maps location rows** (live REST): character 1 (`atlas`) → `channelId 0`,
   character 2 (`Chronicle`) → `channelId 1`.
3. **The client selected the right positions.** atlas-login logged
   `[CharacterListWorldHandle] read [gameStartMode [2], worldId [0], channelId [0]]`
   at `13:33:53Z` (session `1fea9191…`) and `channelId [1]` at `13:34:03Z`
   (session `f0323b77…`). `CLogin::SendLoginPacket` @0x5f6d6a encodes the selected
   **list position** as that byte, so the two clients picked positions 0 and 1 and
   landed on channels 0 and 1. Position→channel is now correct.
4. **/find read the correct channel.** atlas-channel logged the atlas-maps
   response inline at `13:34:21.262Z`:
   `path=…/api/characters/2/location response={"worldId":0,"channelId":1,"mapId":240000000,…,"state":"IN_FIELD"}`,
   followed at `13:34:21.263Z` by `/find resolved branch=channel-remote requester_id=1 target_name=chronicle`.
   So the value handed to the codec was **1**, not 0.
5. **The codec writes it verbatim.** `WhisperFindResultChannel.Encode`
   (`libs/atlas-packet/field/clientbound/whisper.go:220-229`) writes
   `mode, name, byte(3), WriteInt(channelId)` with no adjustment and no version gate.
6. **The client uses it as an array index.** `CField::OnWhisper` @0x53228e, both
   the `0x09` and `0x48` arms, `case 3`:
   `GetChannelName(v118, v2, v122)` where `v122 = Decode4`. `CWvsContext::GetChannelName`
   @0x532f73 is a bounds-checked index into `m_asChannelName`; out of range it
   substitutes the global `WindowName`, not another channel's name.
7. **The array is filled in packet order.** `CLogin::OnWorldInformation` @0x5f95b7
   appends each channel entry as it decodes it (`{DecodeStr name, Decode4 capacity,
   Decode1 worldId, Decode1 channelNo, Decode1 adult}`); the `channelNo` byte at
   entry+12 is stored and never used for placement. `SendLoginPacket` copies
   name[i] → `m_asChannelName[i]` and calls `CWvsContext::SetWorldInfo` @0xa02dde.
   `SetWorldInfo` has exactly two xrefs, both in CLogin — nothing on the channel
   server re-writes the array.
8. **Both announce call sites go through the sort.**
   `socket/handler/server_list.go:97` and
   `kafka/consumer/account/session/consumer.go:286` both call
   `writer.ServerListEntryBody`, which copies and sorts ascending by `ChannelId`
   before encoding. `server_list_entry.go:94` then names entry *i* with
   `ChannelId()+1`.

Steps 1-8 predict the array `[0]="Scania - 1"` (ch 0), `[1]="Scania - 2"` (ch 1),
and therefore `Chronicle is at 'Scania - 2'`. The reporter saw `Scania - 1`.

**The break is between the packet atlas-login sent and the array the client held.**
No server-side artifact I can read contradicts the fix.

### Ruled out, with evidence

- Wrong channel in atlas-maps — live REST and the inline request log both say 1.
- Wrong decision branch — `branch=channel-remote` on both finds.
- Codec dropping/rewriting the value — `whisper.go:220-229`, unconditional `WriteInt`.
- A second, unsorted announce path in atlas-login — grep shows only the two call
  sites, both funnelled through `ServerListEntryBody`.
- The inert `byte(ChannelId()-1)` at `server_list_entry.go:96` — `OnWorldInformation`
  stores it at entry+12 and `SendLoginPacket` never reads it (it reads entry+16,
  the adult flag).
- An out-of-bounds index — that path returns the `WindowName` global, which would
  render as the window title, not as another `Scania - N`.

### Leading hypothesis (unconfirmed): stale client-side world list

`CLogin::OnWorldInformation` @0x5f95b7 **appends** to `m_WorldItem`
(`sub_5FDE1F(&this->m_WorldItem, -1)`) and never clears it, and
`SendLoginPacket` @0x5f6d6a linearly scans for the **first** entry whose worldId
matches. A client process that has been through more than one world-list round —
including one that was already running before the `13:09` rollout — therefore
keeps using the **oldest** copy of world 0, i.e. the pre-fix channel array, no
matter how many times it returns to the login screen. Only killing and relaunching
`MapleStory.exe` discards it.

This is consistent with the reported symptom but does not fully explain it: the
pre-fix array was `[0]="Scania - 1"` (ch 1), `[1]="Scania - 0"` (ch 0), which
would render `Chronicle is at 'Scania - 0'`, not `'Scania - 1'`. So either the
client held some third array, or the "both said Scania - 1" report conflates the
two results (the `atlas` find, from requester 2, correctly reads `Scania - 1`
under *every* hypothesis).

## Fix

None yet — do not dispatch an implementer against a root cause that is not
established. The next step is an observation, not a code change.

## Not yet answered — need the reporter

Run this on a **freshly launched** client (kill `MapleStory.exe` first; do not
just back out to the login screen — see the hypothesis above), and report verbatim:

1. On the world-select screen, the exact text of **every** channel entry, in
   screen order. Expected after the fix: `Scania - 1`, `Scania - 2`.
2. Which entry you clicked for each of the two accounts.
3. The exact `/find` line **in both directions** — the message the channel-0
   character sees for the channel-1 target, and the message the channel-1
   character sees for the channel-0 target.

(3) is the discriminator. If the channel-0→channel-1 find reads `Scania - 2`
after a client restart, the server fixes are correct and the earlier observation
was a stale client. If it still reads `Scania - 1` on a fresh client, the world
list the client received is not what the code above produces, and the next step
is to capture the WORLD_INFORMATION bytes on the wire.

## Outcome

Open.
