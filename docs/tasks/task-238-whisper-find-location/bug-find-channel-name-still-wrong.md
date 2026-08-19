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

**Confirmed with a fresh client** (reporter, second round): `Atlas` on internal
channel 0, `Chronicle` on internal channel 1; `/find chronicle` → `Scania - 1`
AND `/find atlas` → `Scania - 1`. Both array positions render the same name.

## Root cause: atlas-login never reads the `included` channel attributes, so every channel's `ChannelId` is 0

`world.RestModel` (`services/atlas-login/atlas.com/login/world/rest.go:12-27`)
carries `Channels []channel.RestModel` and implements the marshal-side
`GetReferences` / `GetReferencedIDs` / `GetReferencedStructs`, plus the
unmarshal-side `SetToManyReferenceIDs` (`:78-90`). `SetToManyReferenceIDs`
builds one `channel.RestModel{}` per relationship id and sets **only the UUID**:

```go
r.Channels[i] = channel.RestModel{}
err := r.Channels[i].SetID(id)
```

The channel attributes live in the document's `included` array. api2go copies
those into the target **only** if the target implements
`jsonapi.UnmarshalIncludedRelations`, i.e. `SetReferencedStructs`
(`api2go@v1.0.4/jsonapi/unmarshal.go:173-195` — `setIncludedIntoTarget` type-asserts
and silently returns when the assertion fails). `world.RestModel` does not
implement it. Nothing errors; `Channels` simply comes back as N structs with a
UUID and **zero values everywhere else** — `ChannelId 0`, `CurrentCapacity 0`,
`Port 0`.

So `announceServerList` (`socket/handler/server_list.go:91-94` and the
duplicate at `kafka/consumer/account/session/consumer.go:281-284`) builds
`model2.NewChannelLoad(c.ChannelId(), c.CurrentCapacity())` = `(0, 0)` for
**every** channel, and `server_list_entry.go:94` names each entry
`fmt.Sprintf("%s - %d", worldName, 0+1)` → `"Scania - 1"` for both. The client's
`m_asChannelName` is `["Scania - 1", "Scania - 1"]`, and `GetChannelName`
returns `Scania - 1` for any index — exactly the reported symptom.

This is also the real cause of the ORIGINAL bug. Before `682739570` the same
zeroed id rendered `"Scania - 0"` at every position, which is why `/find` "named
channel 0 regardless of which channel the target is actually on". The
`[1, 0]` ordering observed in atlas-world's response was real but was never the
mechanism: `ceb83cc09`'s sort is a **no-op today** because every sort key is 0.
Keep it — it becomes load-bearing the moment the ids are populated.

### Why connecting to a channel still worked

The client encodes the selected **list position** and atlas-login uses that
value as a channel *id* to re-fetch the endpoint directly
(`GET /api/worlds/0/channels/{n}`, visible in the login log). That path never
reads `world.Model.Channels()`, so routing was correct while the names were not.
The reporter's `Atlas`→channel 0 / `Chronicle`→channel 1 placements are genuine
(confirmed against atlas-maps).

### Blast radius beyond the channel name

Same zeroed structs feed the per-channel **capacity** in WORLD_INFORMATION, so
the world-select load gauge has always read empty. `?include=channels` has
exactly one consumer in the repo (`world/requests.go:13`); atlas-channel's
mirror deliberately omits the include, so no other service is affected.

### Ruled out, with evidence

- Wrong channel in atlas-maps — live REST and the inline request log at
  `13:34:21.262Z` both return `channelId 1` for character 2.
- Wrong decision branch — `branch=channel-remote` on both finds.
- Codec dropping/rewriting the value — `whisper.go:220-229` writes `WriteInt`
  unconditionally, no version gate.
- Client-side indexing — `CField::OnWhisper` @0x53228e case 3 →
  `CWvsContext::GetChannelName` @0x532f73, a bounds-checked index into an array
  filled in packet order by `CLogin::OnWorldInformation` @0x5f95b7 /
  `SendLoginPacket` @0x5f6d6a. `SetWorldInfo` @0xa02dde has two xrefs, both in
  CLogin. The client behaves exactly as documented; it was handed two identical
  names.
- A stale client-side world list (the earlier hypothesis) — the reporter
  re-tested on a freshly launched client and the symptom is unchanged.

## Fix

- `services/atlas-login/atlas.com/login/world/rest.go` — add
  `func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error`
  so `world.RestModel` satisfies `jsonapi.UnmarshalIncludedRelations`. Populate
  `r.Channels` from `references["channels"]`, keyed by the ids
  `SetToManyReferenceIDs` already stored, using `jsonapi.ProcessIncludeData` and
  re-applying `SetID` afterwards. The pattern to copy verbatim is
  `services/atlas-channel/atlas.com/channel/party/rest.go:91-109`. Preserve the
  relationship order — do not sort here; `ServerListEntryBody` owns ordering.
- `services/atlas-login/atlas.com/login/world/rest_test.go` — new. Unmarshal a
  JSON:API document for one world whose `channels` relationship lists two ids and
  whose `included` carries their attributes **in `[1, 0]` order** (what
  atlas-world actually returns), then assert each extracted channel's
  `ChannelId`, `CurrentCapacity` and `Port` survive. Must fail before the change
  (today every field is zero).
- Do NOT touch `libs/atlas-packet/login/clientbound/server_list_entry.go` or
  `services/atlas-login/atlas.com/login/socket/writer/server_list.go` — the
  `+1` label and the sort are both correct and become load-bearing once the ids
  are real. The existing `server_list_test.go` ordering test still passes.

## Not yet answered

- `server_list_entry.go:96` still writes `byte(x.ChannelId() - 1)`, underflowing
  to `255` for channel 0, with the decoder at `:138` compensating with `+1`. The
  v83 client stores this at channel-entry+12 and never reads it, so it stays
  inert — but encoder, decoder and the 0-based registry disagree. Unchanged from
  the previous bug file; still deserves its own task.
- Whether any other atlas-login `RestModel` with a to-many relationship has the
  same missing `SetReferencedStructs`. Out of scope for this fix, worth a sweep.

## Outcome

Open — fix not yet applied.
