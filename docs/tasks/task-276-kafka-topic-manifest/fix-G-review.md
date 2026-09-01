# Review: Fix G — post-merge flagless gate failures (commit a961930e7)

Range reviewed: `695210c29..a961930e7` (single commit `a961930e7`).
Brief: `.superpowers/sdd/plan/fix-G-brief.md`
Report: `.superpowers/sdd/plan/fix-G-report.md`

## Scope confirmed

`git show --stat a961930e7` shows exactly 7 files changed, matching the
brief's Files list plus one item the brief itself pre-authorized (see G2
below): `handler_test.go`. No file outside that set was touched, and
`go.work` / `go.work.sum` are unchanged (`git diff 695210c29..a961930e7 --
go.work go.work.sum` is empty), consistent with the report's claim that an
incidental `gofumpt`-triggered `go.work.sum` update was reverted before
commit. Scope matches the brief.

## G1 — atlas-channel `processor_test.go` cast

`services/atlas-channel/atlas.com/channel/movement/processor_test.go:33` now
reads `sharedCapture.Messages(string(movement2.EnvCommandCharacterMovement))`,
matching the existing cast at `teleport_test.go:156`. `go build ./...` and
`go vet ./movement/...` in `services/atlas-channel/atlas.com/channel` are
clean. PASS.

## G2 — player-npc topic typing (the brief-vs-repo discrepancy)

**Verified: the implementer's reading is correct, and the fix does silence
the analyzer diagnostic the brief's line numbers point to.**

- Read `tools/topicguard/analyzer.go` (diagnostic 1, `checkBareTokenLiteral`
  / `reportIfUntypedConstRef`, lines ~152-198): the analyzer's
  "bare topic literal" report fires on a call argument to a
  `topic.Token`-typed parameter that is either (a) a literal string in the
  call itself, or (b) an *identifier/selector reference to a constant whose
  own declared type is not `topic.Token`* — i.e. an untyped string constant
  reaching the parameter only via Go's implicit conversion. Both diagnostic
  message text ("bare topic literal %q reaching a topic.Token parameter")
  and case (b) match what the brief quoted from the gate log.
- Confirmed the two consumer.go files were already passing the *typed
  selector expression* (`msg.EnvEventTopicStatus` /
  `playernpcmsg.EnvEventTopicStatus`) at exactly the brief's cited lines —
  `grep -n` shows lines 39/47 (`services/atlas-messages/.../consumer.go`)
  and 26/34 (`services/atlas-saga-orchestrator/.../consumer.go`), which
  matches the brief's line numbers exactly. The brief's "bare string
  literal" wording was a paraphrase of the analyzer's own diagnostic text,
  not literally "a string literal typed at the call site" — the
  implementer's read is correct and their report documents this precisely.
- Before this commit, both `kafka/message/playernpc/kafka.go` files declared
  `EnvCommandTopic`/`EnvEventTopicStatus` as untyped `string` constants
  (`git show` diff: `EnvCommandTopic = "COMMAND_TOPIC_PLAYER_NPC"` →
  `EnvCommandTopic topic.Token = "COMMAND_TOPIC_PLAYER_NPC"`). Per
  `reportIfUntypedConstRef`, `isTopicTokenType(c.Type())` would have
  returned `false` pre-fix (untyped `string`) → diagnostic fires; post-fix
  it returns `true` → diagnostic is suppressed. The retype is the correct
  and sufficient fix; nothing further is needed in the two `consumer.go`
  files themselves.
- Both `atlas-messages` and `atlas-saga-orchestrator` modules build and vet
  clean (`go build ./...`, `go vet ./...` both modules — verified directly,
  see Testing below).
- The one downstream break the retype surfaced —
  `saga/handler_test.go:1782` (`capture.Messages(playernpcmsg.EnvCommandTopic)`
  compiled pre-retype only via implicit untyped-const conversion) — was fixed
  with the same `string(...)` cast convention used everywhere else on this
  branch's `producertest.Messages` call sites (confirmed by grep: every
  `*Capture.Messages(...)` call site in the diff and in `teleport_test.go`
  uses `string(...)`). This was explicitly pre-authorized by the brief's
  text quoted in the report ("If retyping a constant to `topic.Token` in G2
  breaks an unrelated call site in the same module, fix that call site with
  a `string(...)` cast"). Not scope creep.

Verdict on issue 1: not blocking. The brief's wording was imprecise but its
line numbers and root cause were accurate; the implementer's fix is correct
and verified to close the analyzer diagnostic.

## G3 — saga-orchestrator `testmain_test.go` env token (masking check)

**Verified: the test now asserts real production behaviour; it is not
masking a gap.**

- Production path: `saga/handler.go:1451` —
  `producer.ProviderImpl(h.l)(h.ctx)(playernpcmsg.EnvCommandTopic)(DeployPlayerNpcCommandProvider(...))`.
  This is the exact same `playernpcmsg.EnvCommandTopic` symbol the new
  `os.Setenv` line in `testmain_test.go` sets — not a different constant, not
  a coincidentally-matching literal.
- `producer.ProviderImpl` (`libs/atlas-kafka/producer/provider.go:16-25`)
  routes every token through `ManagerWriterProvider(l)(token)`, which
  ultimately resolves via `topic.EnvProvider` (`libs/atlas-kafka/topic/provider.go`),
  confirmed fail-closed: `os.LookupEnv(string(token))` returns an error
  (`"topic token [%s] has no value in the environment"`) when unset or
  empty, and that error is not translated into a fallback topic name
  anywhere in `EnvProvider`.
- Confirmed via `go build`/`go vet` on `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`
  (clean) that the production code path and the test's env line reference
  the identical symbol; the test therefore exercises the real fail-closed
  contract, not a synthetic pass.

Verdict on issue 2: not blocking. PASS.

## G4 — gofumpt on atlas-player-npcs

Both files gained exactly one blank line between adjacent `const` blocks
(`character/kafka.go:17`, `playernpc/kafka.go:21`), matching the diff.
`gofumpt -l` on both files (via `go run mvdan.cc/gofumpt@latest -l <files>`)
produced no output — both are fully formatted. No `go.work`/`go.work.sum`
side effects remain in the commit. PASS.

## `string(...)` cast consistency

All four `Messages(...)`/`capture.Messages(...)` call sites touched or
adjacent in this diff (`processor_test.go:33`, `teleport_test.go:156`,
`handler_test.go:1782`) use the identical `string(SomeToken)` cast shape.
Consistent with the rest of the branch. PASS.

## Testing performed by this review

Ran directly, not merely trusted from the report:

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./movement/...   # clean
cd services/atlas-messages/atlas.com/messages && go build ./... && go vet ./...          # clean
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go vet ./saga/...  # clean
cd services/atlas-player-npcs/atlas.com/player-npcs && go build ./... && go vet ./...    # clean
gofumpt -l on the two G4 files                                                            # no output
```

Per the brief's explicit exclusion, `tools/verify.sh` was not run and G5
(`verify_test.sh`/`go.work`, under a separate concurrent commit) was not
evaluated or flagged.

## Not evaluable

- Full `tools/go-analyzer-guards.sh` (topicguard analyzer binary) was not
  built and run against the two fixed modules in this review — verification
  of G2 relies on direct reading of `tools/topicguard/analyzer.go`'s logic
  plus confirming the const retype and clean `go vet`, which is sufficient
  to establish the diagnostic no longer fires (the analyzer's own
  `reportIfUntypedConstRef` early-returns on `isTopicTokenType(c.Type())`
  true) but was not exercised end-to-end as a guard run. Non-blocking: the
  brief's own instructions explicitly excluded running `tools/verify.sh` or
  the analyzer guards from this fix's verification scope, and the static
  read is unambiguous.
- G5 (`verify_test.sh` / `go.work`) is out of scope per the task brief and
  was not evaluated.

## Verdict

APPROVED. All four root causes (G1-G4) are correctly and minimally fixed,
scope matches the brief plus one pre-authorized fallout fix, both points
flagged for special scrutiny (G2's brief-vs-repo discrepancy, G3's
env-token/production-path correspondence) check out under direct
verification, and all four touched modules build and vet clean.
