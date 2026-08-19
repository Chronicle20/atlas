# Status — cross-channel chat delivery fixes

Two branches, both implemented and reviewed. Neither is live re-tested.

## Branch 1 — `fix-whisper-cross-channel-delivery` (off `main`, base 748510af2)

| Commit | What |
|---|---|
| `420ef7f98` | bug diagnosis (docs) |
| `30b459905` | whisper delivery: world-scoped recipient gate |
| `489bb3327` | same fix for multi/messenger/pink-text |
| `515f64afd` | DOM-20: collapse handler tests into table-driven subtests |

- `atlas-reviewer`: **APPROVED_WITH_FINDINGS**, 0 blocking —
  `review-cross-channel-delivery.md`. Exactly-once verified by hand: each pod has
  a `SERVICE_ID`-derived consumer group so every pod sees every `chat_event`, and
  `IfPresentByCharacterId` filters on `(characterId, worldId, channelId)`.
- `backend-guidelines-reviewer`: **CHANGES_REQUIRED**, 2 blocking — `audit.md`.

**Flagless `tools/verify.sh`: PASS (exit 0)**, run twice — before and after
`515f64afd`. Scope is 1 module (`services/atlas-channel/atlas.com/channel`);
see the go.work.sum note below.

### Outstanding on branch 1

1. ~~**DOM-20**~~ — **DONE** in `515f64afd`. 13 top-level tests became 11
   subtests across 3 tables plus 2 standalone whisper-only cases; coverage
   verified 1:1 by enumerating `go test -v` subtest names. (The audit said "17
   ad hoc test functions"; there were 13. The duplication it flagged was real,
   the count was not.)
2. **DOM-25 (rejected for this branch, needs a decision).** The audit flags
   `NewWhisperSendResult(0x0A, ...)` / `NewWhisperReceive(0x12, ...)` in
   `consumer.go`. Both are verbatim on `main` (748510af2 lines 172 and 178); the
   diff only re-indents them. Resolving them from tenant writer options is a
   packet-configuration task: `CharacterChatWhisper` has `"options": null` in all
   nine seed templates (gms 48/61/72/79/83/84/87/95, jms 185), so the per-version
   mode table must be authored from the client — it cannot be guessed. It also
   overlaps task-238, which emits the sibling find modes `0x09`/`0x48`.
   **Recommend a separate packet task.**
3. Nothing. Branch 1 is ready for PR pending the live re-test below.

## Branch 2 — `fix-whisper-send-result-presence` (stacked on `task-238-whisper-find-location`, base 62439e69b)

| Commit | What |
|---|---|
| `7ecf2681c` | bug diagnosis (docs) |
| `2633d67d3` | `whisperDecision` table; report `success:false` for unreachable targets |
| `d89df07c4` | dedupe the failure announce back to one call site (audit response) |

- `atlas-reviewer`: **APPROVED**, 0 blocking — `review-whisper-send-result.md`.
  Fail-open on infrastructure error verified in code; presence switch is
  exhaustive so no reachable target is misclassified.
- `backend-guidelines-reviewer`: **CHANGES_REQUIRED**, 1 blocking (DOM-25, the
  duplicated `0x0A`) — addressed by `d89df07c4`, which returns the literal to a
  single occurrence. The remaining single literal is the same pre-existing
  DOM-25 issue as branch 1 item 2.

### Outstanding on branch 2

1. Flagless `tools/verify.sh` not yet run. Held back deliberately so two docker
   bakes do not contend; run it once branch 1's gate returns.
2. Cannot merge before task-238 / PR #1407.

## The `go.work.sum` trap (repo-wide, worth knowing)

Running any `go` command in a worktree dirties `go.work.sum` with 3 lines of
checksum churn (old `/go.mod` hashes). `verify.sh` classifies any change under
`go.work`/`libs/` as a shared-lib change, so a dirty tree silently escalates the
gate from **1 module to 89** with `-race`:

    dirty:  fanout_reason=shared-lib:go.work.sum   modules_selected=89
    clean:  fanout_reason=none                     modules_selected=1

The first gate attempt on this branch ran 90+ minutes for that reason and was
abandoned mid-lint at 68/89 with zero issues — it was not failing, just doing
89x the necessary work. `git checkout -- go.work.sum` immediately before
launching the gate fixes it. The durable fix is refreshing `go.work.sum` on
`main` so the toolchain stops regenerating it; that is out of scope here and
affects everyone's builds.

## Live re-test (both branches)

Original repro: tenant `625de849-e34f-45c8-95e6-b8e794774422`, GMS 83.1,
namespace `atlas-pr-1407`. `Atlas` (id 1) on channel 0 whispers `Chronicle`
(id 2) on channel 1 — recipient received nothing, no error logged. Re-test needs
a deploy. Branch 2's presence path additionally needs task-238's atlas-maps
migration running.

## Known latent issue (neither branch's; surfaced by review)

`deploy/k8s/base/atlas-channel.yaml:44-45` — `SERVICE_ID` is a hard-coded
literal and `replicas: 1`. Scaling atlas-channel past one replica without making
`SERVICE_ID` per-pod-unique puts all pods in one consumer group, partitioning the
topic so a `chat_event` can reach a pod hosting none of the relevant channels.
This breaks the exactly-once assumption these fixes rely on, and every other
session-routed Kafka consumer in the service, because the session registry is
pod-local with no cross-pod affinity. atlas-channel is effectively not
horizontally scalable today.
