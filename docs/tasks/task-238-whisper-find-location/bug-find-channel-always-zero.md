# bug: /find on a remote channel names the wrong channel (reads "Scania - 0")

**Reproduced:** tenant `625de849-e34f-45c8-95e6-b8e794774422`, GMS 83.1, ephemeral
env `atlas-pr-1407` (images `pr-1407-62439e6` = branch head). Characters `atlas`
(id 1, channel 0) and `Chronicle` (id 2, channel 1). `/find <name>` from a
channel the target is not on.

**Observed:** the client prints the different-channel message but names channel 0
regardless of which channel the target is actually on.

**Expected:** PRD FR-6 — the message names the target's real channel.

**Root cause: atlas-login emits the world's channel list in unordered order, and
the client's channel-name array is indexed by list position.**

The v83 client never formats a channel *number*. `CField::OnWhisper` @0x53228e,
findMode arm `case 3` (both the 0x09 and 0x48 arms) passes the Decode4 value to
`CWvsContext::GetChannelName` @0x532f73, which is a bounds-checked index into the
`m_asChannelName` array at `CWvsContext+0x36D4`. That array is filled by
`CWvsContext::SetWorldInfo` @0xa02dde from `CLogin::SendLoginPacket` @0x5f6d6a,
which copies the WORLD_INFORMATION channel entries **in packet order**, index
0,1,2… — it does not read the per-channel id byte at entry+12 (that byte is
never consulted; `SendLoginPacket` copies entry+16, the adult flag, into the
second array). Entry layout confirmed against `CLogin::OnWorldInformation`
@0x5f95b7: `{DecodeStr name, Decode4 capacity, Decode1 worldId, Decode1
channelNo, Decode1 adult}`.

**So the protocol requires packet position == channel id.** It does for the
login channel-select too: the client encodes the selected list *position* as the
channel to connect to.

atlas-login does not honour that. `announceServerList`
(`socket/handler/server_list.go:92`, and the duplicate in
`kafka/consumer/account/session/consumer.go:281`) iterates `x.Channels()` in
whatever order atlas-world returned, and atlas-world's `?include=channels`
currently returns **[1, 0]** for this world — stable across five consecutive
requests:

```
$ curl .../api/worlds/?include=channels | jq '[.included[].attributes.channelId]'
[1, 0]   [1, 0]   [1, 0]   [1, 0]   [1, 0]
```

`server_list_entry.go:94` names each entry `fmt.Sprintf("%s - %d", worldName,
x.ChannelId())`, so the client's array ends up:

| index | name |
|---|---|
| 0 | `Scania - 1` |
| 1 | `Scania - 0` |

`/find` sends the target's real channel id — verified correct end to end:
atlas-maps holds `id 1 → channelId 0` and `id 2 → channelId 1` (live REST), the
logs show `branch=channel-remote` on every find, and
`whisper.go:220-229` writes `WriteInt(channelId)` unaltered. The client then uses
that id as an **array index**, so a target on channel 1 renders
`m_asChannelName[1]` = `"Scania - 0"` — the reported symptom.

Ruled out, with evidence: wrong decision branch (`character_chat_whisper.go:145-156`
sets `channelId: uint32(loc.ChannelId())`; logs show the channel-remote branch);
stale location row (live REST is correct); codec dropping the value
(`whisper.go:220-229`, `WriteInt` is 4-byte LE at
`libs/atlas-socket/response/writer.go:36`); `Announce` rewriting the body
(`session/processor.go:262-272` passes the encoder through). **Nothing on the
task-238 /find path is wrong.** This is a pre-existing atlas-login defect that
/find is the first feature to expose.

## Fix

- `services/atlas-login/atlas.com/login/socket/writer/server_list.go:16-24` —
  `ServerListEntryBody` is the single funnel both call sites go through. Sort the
  `channelLoad` slice by `ChannelId()` ascending before building `cls`, so packet
  position equals channel id. Do not mutate the caller's slice — copy first.
  Sorting here rather than at the two call sites keeps the positional invariant
  with the encoder that depends on it.
- `services/atlas-login/atlas.com/login/socket/writer/server_list_test.go` — new.
  Feed `[]model.Load` in descending channel order (ids 1 then 0, mirroring what
  atlas-world returns today), decode the produced body with
  `loginpkt.ServerListEntry.Decode`, and assert the channel entries come back
  ascending by id and that entry *i* names channel *i*. Must fail before the fix.

Leave `server_list_entry.go` alone: its encoder has pinned
`packet-audit:verify` fixtures, and the ordering contract belongs to the caller
that assembles the list.

## Not yet answered

- **Channel label base — user decision, do not guess.** `server_list_entry.go:94`
  formats the name with the 0-based `x.ChannelId()`, so after this fix channel 0
  still reads `"Scania - 0"` and world-select lists `Scania - 0 … Scania - N-1`.
  Every retail MapleStory labels channels 1-based. Changing it to
  `x.ChannelId() + 1` is a one-line, user-visible change to world-select and to
  every channel name the client renders. Not in scope for this fix; raised with
  the reporter separately.
- `server_list_entry.go:96` writes `byte(x.ChannelId() - 1)`, which underflows to
  `255` for channel 0, and the decoder at `:138` compensates with `+ 1`. The v83
  client stores this byte at entry+12 and never reads it, so it is inert today —
  but encoder, decoder and the 0-based registry disagree. Left alone here because
  the struct carries pinned verify fixtures; worth its own task.

## Outcome

- Fixed by `ceb83cc09` — sort the channel list by id in `ServerListEntryBody`
  (`services/atlas-login/atlas.com/login/socket/writer/server_list.go`), the
  single funnel both call sites use, so packet position equals channel id.
- Follow-up `6827395704f0` — 1-based channel labels (see the follow-up section
  below). This also required updating the pinned `gms_v48` fixture's expected
  name byte in `server_list_entry_v48_test.go`: contrary to what the follow-up
  brief claimed, `TestServerListEntryBytesV48` DOES assert the channel-name
  string. The length prefix is unchanged (`0x0A`; "Scania - 2" is still ten
  chars), so the fixture's pinned read order and field widths are untouched —
  only the content of a display string the client renders.
- Gate, range `62439e69b..HEAD`: `tools/verify.sh --quick` PASSES every check
  except the lint & format guard; `packet-audit matrix --check` and
  `fname-doc --check` both PASS, so the fixture edit staled no evidence record.
- **Lint guard: blocked, not failed.** Four runs, all ending in
  `Error: parallel golangci-lint is running`, each naming different modules this
  branch never touched, with "0 issues." for every module that completed. Cause
  identified: a concurrent golangci-lint (pid 3693706) in the
  `fix-whisper-cross-channel-delivery` worktree — the cross-worktree lock
  contention documented at `docs/verification.md:285`. Re-run once that
  finishes. No lint finding is attributable to this branch.
- **Still outstanding: flagless `tools/verify.sh`** (the `--quick` runs skip the
  Docker bake and `-race`), and the **live re-test**. Neither the gate nor any
  unit test can observe the reported symptom — it is a client-render defect.
  Re-test: `/find` the channel-1 character from channel 0 and confirm the named
  channel is correct. Expect world-select to read `Scania - 1` / `Scania - 2`
  where it previously read `Scania - 0` / `Scania - 1`.

---

## Follow-up: retail 1-based channel labels (approved by reporter)

Decision taken after the ordering fix landed (`ceb83cc09`): adopt retail
numbering. `server_list_entry.go:94` names channels with the 0-based
`x.ChannelId()`, so world-select currently lists `Scania - 0 … Scania - N-1` and
a /find against channel 0 reads `"Scania - 0"`. Every retail MapleStory labels
channels 1-based.

This is label-only. It does not change which channel a position maps to: the
client still selects and reports channels by array position, and the ordering
fix already guarantees position == channel id. Only the rendered string moves.

### Fix

- `libs/atlas-packet/login/clientbound/server_list_entry.go:94` — format the
  channel name with `x.ChannelId() + 1` so channel id 0 renders `"Scania - 1"`.
  Change the `:135` decoder comment's example to match. Do NOT touch `:96`
  (`byte(x.ChannelId() - 1)`) or the `:138` decoder — see the note below.
- `libs/atlas-packet/login/clientbound/server_list_entry_test.go` — add a case
  asserting the encoded name for `channel.Id(0)` is `"<world> - 1"`. Must fail
  before the change.

Safe against the pinned `packet-audit:verify` cells: no existing fixture asserts
the channel-name string. `TestServerListEntryV61Body` encodes zero channels;
`TestServerListEntryWorldIdInChannels` and `TestServerListEntryRoundTrip` skip
the name with `_ = r.ReadAsciiString()`. The v48 fixture pins the id byte
(`server_list_entry_v48_test.go:56`), not the name.

### Still out of scope

`server_list_entry.go:96` writes `byte(x.ChannelId() - 1)`, underflowing to `255`
for channel 0, with the decoder at `:138` compensating with `+ 1`. The v83 client
stores this byte at channel-entry+12 and never reads it (`SendLoginPacket`
@0x5f6d6a copies entry+16, the adult flag), so it is inert — but encoder, decoder
and the 0-based registry disagree. Changing it moves bytes under a pinned v48
fixture and deserves its own task.
