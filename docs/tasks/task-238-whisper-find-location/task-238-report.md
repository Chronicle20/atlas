# Task report: fix find/channel-list ordering bug

## What I implemented

Fixed `services/atlas-login/atlas.com/login/socket/writer/server_list.go`
(`ServerListEntryBody`) to sort a **copy** of the `channelLoad` slice ascending
by `ChannelId()` before building the `packetmodel.ChannelLoad` slice passed to
`loginpkt.NewServerListEntry`. This is the single funnel both call sites
(`socket/handler/server_list.go:92` and
`kafka/consumer/account/session/consumer.go:281`) go through, so both are
fixed by the one change. The caller's slice is not mutated (`copy` into a new
`ordered` slice first).

Left `server_list_entry.go` untouched, per the brief and the ruling — both
items in "Not yet answered" (0-based vs 1-based channel label, and the
underflowing `byte(x.ChannelId()-1)` field) are out of scope.

## TDD evidence

New test: `services/atlas-login/atlas.com/login/socket/writer/server_list_test.go`
— `TestServerListEntryBody_OrdersChannelsByIdRegardlessOfInputOrder`. Feeds
`[]model.Load{channel 1, channel 0}` (descending, mirroring atlas-world's
`[1, 0]` today), encodes with `ServerListEntryBody`, decodes the produced body
with `loginpkt.ServerListEntry.Decode`, and asserts entry *i* names channel
*i*.

**RED** — stashed only `server_list.go` (kept the new test), ran:
```
$ go test ./socket/writer/...
--- FAIL: TestServerListEntryBody_OrdersChannelsByIdRegardlessOfInputOrder (0.00s)
    server_list_test.go:47: entry 0: got channel id 1, want 0
    server_list_test.go:47: entry 1: got channel id 0, want 1
FAIL
FAIL	atlas-login/socket/writer	0.006s
FAIL
```
Failure is exactly the reported bug: unsorted input round-trips unsorted.

**GREEN** — restored the fix (`git stash pop`), ran the full module suite:
```
$ cd services/atlas-login/atlas.com/login && go build ./... && go test ./...
```
All packages `ok` (or `no test files`), including `ok atlas-login/socket/writer 0.008s`.
Output pristine, no warnings.

## Files changed

- `services/atlas-login/atlas.com/login/socket/writer/server_list.go` — sort
  channel loads ascending by id before encoding; doc comment explains why
  (client indexes by packet position, not by any on-wire channel id).
- `services/atlas-login/atlas.com/login/socket/writer/server_list_test.go` —
  new; RED-before-fix, GREEN-after-fix as shown above.

## Self-review

- Scope: exactly the two files named in the brief's `## Fix` section; did not
  touch `server_list_entry.go` or its pinned `packet-audit:verify` fixtures.
- No mutation of the caller's `channelLoad` slice (`copy` before `sort.Slice`).
- Test decodes through the real `loginpkt.ServerListEntry.Decode`, not a mock
  — it exercises the actual wire contract, matching the pattern in
  `libs/atlas-packet/login/clientbound/server_list_entry_test.go`.
- `go build ./...` and `go test ./...` both clean from the module root.

## Issues / concerns

None. The two "Not yet answered" items in the bug doc (1-based channel label,
underflowing per-channel byte) remain genuinely out of scope per the
controller's ruling — not something I could silently finish without a product
decision on user-visible channel labeling.

Note: `docs/tasks/task-238-whisper-find-location/bug-find-channel-always-zero.md`
was already present in the worktree as an untracked file when I started (the
diagnosis artifact this task fixes). I did not commit it — it's not one of the
two files named in `## Fix`, and committing someone else's diagnosis artifact
under my fix commit felt out of scope. Flagging in case the controller wants
it tracked.

## Outcome

- Fixed by: this task (task-238 bug fix)
- Live re-test: pending (not run by implementer per Contract 2 — module-local
  only)
