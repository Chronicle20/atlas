# Fix report: world.RestModel never hydrates included channel attributes

## What I implemented

Applied the `## Fix` section of `bug-find-channel-name-still-wrong.md` verbatim,
touching exactly the two files it names.

### `services/atlas-login/atlas.com/login/world/rest.go`

Added `SetReferencedStructs`, implementing `jsonapi.UnmarshalIncludedRelations`,
following `services/atlas-channel/atlas.com/channel/party/rest.go:91-109`
verbatim in structure:

```go
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["channels"]; ok {
		for i := range r.Channels {
			if data, ok := refMap[r.Channels[i].GetID()]; ok {
				id := r.Channels[i].GetID()
				scm := channel.RestModel{}
				err := jsonapi.ProcessIncludeData(&scm, data, references)
				if err != nil {
					return err
				}
				err = scm.SetID(id)
				if err != nil {
					return err
				}
				r.Channels[i] = scm
			}
		}
	}
	return nil
}
```

Differs from the `party.RestModel` copy in one deliberate way: it iterates
`r.Channels` **by index** and assigns `r.Channels[i] = scm` in place, instead
of building a new slice (`nm`) and reassigning `r.Members = nm` at the end.
This was required to satisfy the brief's explicit instruction — "Preserve the
relationship order — do not sort here" — because `r.Channels` was already
populated by `SetToManyReferenceIDs` in relationship order (the ids
`SetToManyReferenceIDs` stored, e.g. `[chan0-uuid, chan1-uuid]`), and the
`included` array's iteration order in `references["channels"]` is a Go map
(unordered). Rebuilding via a fresh slice appended in map-iteration order
would have silently reintroduced the exact bug this fix closes (`included`
order is `[1, 0]` per the live evidence in the bug file). Indexing into the
existing `r.Channels[i]` and looking up each element's own id in the map
keeps the relationship order intact regardless of the `included` array's
order in the wire document.

### `services/atlas-login/atlas.com/login/world/rest_test.go` (new)

New test `TestRestModel_HydratesIncludedChannels`, modeled directly on
`services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest_test.go`
(`jsonapi.Unmarshal(doc, &target)` against a literal JSON:API document, no
HTTP server). Builds one `worlds` resource with a `channels` to-many
relationship listing two channel ids, and an `included` array carrying their
attributes in **`[1, 0]` order** (channel-id-1 first, channel-id-0 second) —
matching what atlas-world actually returns, per the bug file. Asserts each
extracted channel's `GetID()`, `ChannelId`, `Port`, and `CurrentCapacity`
land at the correct relationship-ordered index (`Channels[0]` = channel 0,
`Channels[1]` = channel 1) despite the reversed `included` order.

## Files NOT touched (per the brief's explicit exclusion)

- `libs/atlas-packet/login/clientbound/server_list_entry.go`
- `services/atlas-login/atlas.com/login/socket/writer/server_list.go`

## TDD evidence

**RED** — before the fix, ran the new test alone:

```
cd services/atlas-login/atlas.com/login && go test ./world/... -run TestRestModel_HydratesIncludedChannels -v
```

```
=== RUN   TestRestModel_HydratesIncludedChannels
    rest_test.go:77: Channels[0].Port = 0, want 7575
    rest_test.go:80: Channels[0].CurrentCapacity = 0, want 3
    rest_test.go:88: Channels[1].ChannelId = 0, want 1
    rest_test.go:91: Channels[1].Port = 0, want 7576
    rest_test.go:94: Channels[1].CurrentCapacity = 0, want 12
--- FAIL: TestRestModel_HydratesIncludedChannels (0.00s)
FAIL
FAIL	atlas-login/world	0.005s
```

Failure matches the documented root cause exactly: attributes came back zero
(Channels[0].ChannelId happened to already be 0 by coincidence — the
`GetID()` UUID assertions passed since `SetToManyReferenceIDs` already sets
those correctly; only the attribute fields were wrong).

**GREEN** — after adding `SetReferencedStructs`:

```
cd services/atlas-login/atlas.com/login && go test ./world/... -run TestRestModel_HydratesIncludedChannels -v
```

```
Go test: 1 passed in 2 packages
```

## Module-local verification

```
cd services/atlas-login/atlas.com/login && go build ./... && go test ./...
```

All packages built and all tests passed (`ok atlas-login/world 0.030s`, plus
every other package in the module — full output reviewed, no failures, no
skips beyond the pre-existing "no test files" packages). `go vet ./world/...`
was also run and produced no output.

## Files changed

- `services/atlas-login/atlas.com/login/world/rest.go` — added
  `SetReferencedStructs`.
- `services/atlas-login/atlas.com/login/world/rest_test.go` — new test file.

## Self-review

- Completeness: implements exactly the `## Fix` bullet for `rest.go` and the
  test bullet for `rest_test.go`; nothing in the brief's fix section was
  skipped.
- Discipline: did not touch the two files the brief explicitly excludes; did
  not widen scope to the "Not yet answered" items (the byte-underflow issue,
  or sweeping other `RestModel`s for the same missing method) — both are
  explicitly out of scope for this fix per the brief.
- Quality: variable/method naming (`scm`, `refMap`) matches the copied
  `party.RestModel` pattern's own naming (`srm`, `refMap`) for consistency
  within the codebase, adapted only where in-place assignment required a
  different loop shape.
- Testing: test asserts real behavior (attribute hydration under a specific
  `[1, 0]` included-order) via the actual `jsonapi.Unmarshal` entry point, not
  a mock of api2go's internals — matches the sibling regression test's
  pattern exactly. TDD followed: RED shown before GREEN, RED failure matches
  the documented root cause.

## Issues or concerns

None. The fix is narrowly scoped, verified against a live-shaped
`included`-order edge case, and both excluded files were left untouched
(confirmed via `git status --short` before commit, which showed only the two
intended files).

## Commit

`53537cf58` — `fix(atlas-login): hydrate included channel attributes on
world.RestModel`
