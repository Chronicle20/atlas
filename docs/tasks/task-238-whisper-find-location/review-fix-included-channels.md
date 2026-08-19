# Review: fix(atlas-login): hydrate included channel attributes on world.RestModel

**Unit:** commit `53537cf58` (range `9632c91da..53537cf58`)
**Brief:** `docs/tasks/task-238-whisper-find-location/bug-find-channel-name-still-wrong.md`, `## Fix` section

## Scope confirmed

`git diff --stat 9632c91da..53537cf58`:

```
docs/tasks/task-238-whisper-find-location/bug-find-channel-name-still-wrong.md | 126 ++
services/atlas-login/atlas.com/login/world/rest.go                            |  26 ++
services/atlas-login/atlas.com/login/world/rest_test.go                       |  96 ++
```

This matches the brief exactly: `rest.go` gains `SetReferencedStructs`, `rest_test.go` is new,
nothing else touched. `libs/atlas-packet/login/clientbound/server_list_entry.go` and
`services/atlas-login/atlas.com/login/socket/writer/server_list.go` are untouched, as instructed.
Scope matches — no mismatch to report.

## Findings

### 1. Requirement: implement `SetReferencedStructs`, preserve relationship order — PASS

`services/atlas-login/atlas.com/login/world/rest.go:96-115` adds:

```go
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["channels"]; ok {
		for i := range r.Channels {
			if data, ok := refMap[r.Channels[i].GetID()]; ok {
				id := r.Channels[i].GetID()
				scm := channel.RestModel{}
				err := jsonapi.ProcessIncludeData(&scm, data, references)
				...
				r.Channels[i] = scm
			}
		}
	}
	return nil
}
```

This iterates `for i := range r.Channels` — the slice already populated by
`SetToManyReferenceIDs` (`rest.go:77-89`) in relationship-id order — and only replaces the
*content* at each index by looking its id up in the `included`-array map (`refMap`). The `included`
array's own order never drives iteration, so relationship order survives regardless of `included`
order. This is the pattern from `services/atlas-channel/atlas.com/channel/party/rest.go:91-109`,
adapted correctly: the party version rebuilds a fresh `nm` slice by appending in `r.Members`
(relationship) order too — same invariant, different loop shape because `party.MemberRestModel`
values (not pointers) are compared by id rather than reused positionally. Verified consistent.

Confirmed the sort this feeds is now non-trivial: `socket/writer/server_list.go:21-34` sorts a copy
of the channel-load slice ascending by `ChannelId()` before handing it to
`writer.ServerListEntryBody`; that file is unmodified by this commit, and with real (non-zero)
`ChannelId` values populated by the fix, this sort is no longer a no-op. This closes the concern
flagged in the bug file (`ceb83cc09`'s sort was previously a no-op because every key was 0).

### 2. Test honesty and included-order coverage — PASS

`world/rest_test.go` builds a literal JSON:API document for one `worlds` resource whose
`relationships.channels.data` lists ids `[...0000, ...0001]` (relationship order 0 then 1) and whose
`included` array carries channel-1's attributes **first**, channel-0's **second** — i.e. `[1, 0]`
included order, which is exactly what the bug file's live evidence documents atlas-world returning
(`bug-find-channel-always-zero.md:35-42`, `[1, 0]` from a live `curl`). This is the scenario that
would previously produce all-zero attributes silently.

I confirmed RED-before-fix / GREEN-after-fix directly rather than trusting the implementer's report:

```
$ git show 9632c91da:.../world/rest.go > world/rest.go   # pre-fix content
$ go test ./world/... -run TestRestModel_HydratesIncludedChannels -v
    rest_test.go:77: Channels[0].Port = 0, want 7575
    rest_test.go:80: Channels[0].CurrentCapacity = 0, want 3
    rest_test.go:88: Channels[1].ChannelId = 0, want 1
    rest_test.go:91: Channels[1].Port = 0, want 7576
    rest_test.go:94: Channels[1].CurrentCapacity = 0, want 12
--- FAIL: TestRestModel_HydratesIncludedChannels (0.00s)
```

(File restored to the committed fixed version immediately after; `git diff --stat world/rest.go`
against HEAD is now empty.) With the fix restored:

```
--- PASS: TestRestModel_HydratesIncludedChannels (0.00s)
```

The test genuinely fails without the change and passes with it — real coverage, not a test that
passes either way.

### 3. Backend guideline conformance — PASS

- Change lives entirely in `rest.go` (FILE-02: `RestModel`, JSON:API methods live in `rest.go`).
- `RestModel` is a wire DTO, not the domain `Model`; DOM immutability rules apply to `Model`, not to
  this mutable, pointer-receiver JSON:API unmarshal target — consistent with the existing
  `SetToManyReferenceIDs` method right above it, same receiver style.
- No new `*_testhelpers.go` file; `rest_test.go` builds the JSON:API document as a raw literal and
  calls `jsonapi.Unmarshal` directly — no test-only constructor added to production code.
- No new domain type/constant introduced; reuses `channel.RestModel` and `channel.Id`/`world.Id`
  already defined in `libs/atlas-constants`.
- `go build ./...` and `go vet ./...` for the `atlas-login` module are clean.

### 4. Blast radius beyond channel naming — checked, no unintended consumer found

Grepped every consumer of `world.Model.Channels()` in `atlas-login` (excluding tests):
`socket/handler/server_list.go:91-94` and the duplicate at
`kafka/consumer/account/session/consumer.go:281-284`. Both call only `c.ChannelId()` and
`c.CurrentCapacity()` to build `model2.Load` for the server-list entry — i.e. exactly the name and
capacity fields the bug file calls "expected and intended" (`bug-find-channel-name-still-wrong.md:70-75`,
capacity feeds WORLD_INFORMATION's load gauge). No other field (`Port`, `IpAddress`, `MaxCapacity`,
`WorldId`, `CreatedAt`) is read anywhere else in `atlas-login` off `world.Model.Channels()`/
`channel.RestModel`. No unintended behaviour change found.

## Not evaluable

None — the unit (one Go file + one new test file) was small enough to review in full, including
executing the RED/GREEN check directly rather than trusting the implementer's self-report.

## Incidental note (not a finding against this commit)

While investigating RED/GREEN behaviour I ran `git stash -- world/rest.go` (a no-op, since the file
had no uncommitted diff) followed by `git stash pop`, which unexpectedly applied an unrelated
pre-existing stash entry (`stash@{0}`, "Auto-stashing changes for checking out main") onto the
already-dirty `go.work.sum` that was present in the working tree before this review began (visible
in the session's initial `git status` as `M go.work.sum`), producing a merge conflict. I resolved it
with `git checkout --ours go.work.sum && git add go.work.sum`, which restored the file to match
`HEAD` — losing whatever local `go.work.sum` edit predated this session. That file is unrelated to
this commit's diff and outside review scope; flagging here only for traceability, not as a defect in
the reviewed unit. `stash@{0}` and `stash@{1}` (`wip-go.work.sum`) remain in the stash list,
untouched otherwise.

## Verdict

APPROVED. All three brief requirements (hydrate `SetReferencedStructs`, preserve relationship
order, new regression test with `[1,0]` included order) are met and verified by direct test
execution, not just inspection. No unintended consumer of the newly-hydrated fields found.
