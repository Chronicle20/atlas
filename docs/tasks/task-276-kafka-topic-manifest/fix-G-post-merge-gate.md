# Fix G — post-merge flagless gate failures (verify-final4.log, VERIFY EXIT=1)

The pre-merge green (`614155c81`) is stale. After merging `origin/main` (`bfeca5bf6`) and
migrating the newly-landed `atlas-player-npcs` service (`695210c29`), the flagless
`tools/verify.sh` reports **8 failed checks over 5 root causes**. Four are mechanical
merge fallout; the fifth is a semantic conflict between this branch and a test main
brought with it.

## G1 — `atlas-channel` movement test passes a `topic.Token` where a `string` is wanted

```
movement/processor_test.go:33:46: cannot use movement2.EnvCommandCharacterMovement
(constant "COMMAND_TOPIC_CHARACTER_MOVEMENT" of string type topic.Token) as string
value in argument to sharedCapture.Messages
```

Same defect the merge resolution already fixed in `teleport_test.go`; main's
table-driven restructure introduced a second call site in `processor_test.go` that the
conflict resolution did not cover. `Messages` takes a `string`, so the call site needs
the explicit `string(...)` cast the rest of the branch uses.

**This one compile error is the sole cause of 5 of the 8 failing checks** — the module
build, and the four guards that type-check the same package (analyzer guards, scope
guard, producer seam guard, env domain guard) plus its lint target. It is not four
independent findings.

## G2 — main's new player-npc consumers use bare topic literals

`libs/atlas-kafka`'s analyzer guard flags four sites:

- `services/atlas-messages/atlas.com/messages/kafka/consumer/playernpc/consumer.go:39,47`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/playernpc/consumer.go:26,34`

each passing the raw string `"EVENT_TOPIC_PLAYER_NPC_STATUS"` into a `topic.Token`
parameter. Both services already own an `Env*` constant for it in their
`kafka/message/playernpc/kafka.go`, but those constants are still **untyped strings**
(main wrote them against the pre-branch API). Type them `topic.Token` and route the
consumers through them.

## G3 — `atlas-saga-orchestrator` saga tests emit nothing

```
topic token [COMMAND_TOPIC_PLAYER_NPC] has no value in the environment
"[]" should have 1 item(s), but has 0 — handler must emit exactly one DEPLOY command
```

`EnvProvider` is fail-closed on this branch: an unset token emits no message rather than
producing to a bogus topic. `saga/testmain_test.go` sets three tokens but not the
player-npc command topic that main's `TestDeployPlayerNpcAction` needs. Same
`os.Setenv(string(Token), string(Token))` pattern as the existing three lines.

## G4 — gofumpt on `atlas-player-npcs`

Missing blank line between adjacent `const` blocks, introduced by the `topicmod` codemod
in `695210c29`:

- `services/atlas-player-npcs/atlas.com/player-npcs/kafka/message/character/kafka.go:17`
- `services/atlas-player-npcs/atlas.com/player-npcs/kafka/message/playernpc/kafka.go:21`

## G5 — `verify_test.sh` × 2: this branch narrowed `all_modules()`; main's test asserts the old invariant

Both failures share one cause. This branch's Fix commits `1ef79e732` and `c2a97ac85`
changed `all_modules()` from "every `go.mod` under `services/` and `libs/`" to that set
**intersected with `go.work`'s `use` list**. Main then merged in a `verify_test.sh` that
asserts the pre-filter invariant:

1. `the escape hatch restores the old behaviour (module count)` — want 93, got 92.
   The missing module is `libs/atlas-kafka/gen`, added by this branch's Task 5 and never
   added to `go.work`. (Its sibling `libs/atlas-constants/gen` *is* in `go.work`.)
2. `the broken module is reported FAILED, unstripped (got '')` — the test plants a
   deliberately un-compilable module at `services/zz-verify-probe-broken-<tag>/` and
   asserts `verify.sh` reports it FAILED. Not being in `go.work`, it is now filtered out
   and **silently never built**.

Failure 2 is the substantive one: it is not a stale expectation, it is a real hole this
branch opened. A new service directory that is not yet listed in `go.work` is now
silently invisible to the whole gate — which is exactly how `libs/atlas-kafka/gen` slipped
through in failure 1. Adding `libs/atlas-kafka/gen` to `go.work` fixes failure 1 but
leaves failure 2 and the hole behind it untouched.

**Needs a ruling before implementation — see the controller's question to the user.**
