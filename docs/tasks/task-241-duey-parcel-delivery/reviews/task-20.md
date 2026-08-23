# Review: Task 20 — NPC conversation entry point for Duey

Commit range: `0124fa9d8..a6acc9f38` (single commit `a6acc9f38`)
Brief: `.superpowers/sdd/plan/task-20-brief.md`
Design: `docs/tasks/task-241-duey-parcel-delivery/design.md` §9.6

## Scope

`git diff --stat 0124fa9d8..a6acc9f38`:

- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` (+22)
- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go` (+85)
- `services/atlas-npc-conversations/atlas.com/npc/saga/model.go` (+2)
- eight `deploy/seed/*/npc-conversations/npc/npc-9010009.json` (+30 each)

Matches the brief's file list exactly. `scope_confirmed`: reviewed all of the above; no
scope mismatch.

## Priority 1 — `open_duey` operation and payload

`operation_executor.go:2130-2153` (new `case "open_duey"`), compared against the
`open_storage` case it copies (`operation_executor.go:2101-2131`):

```go
case "open_duey":
    ctx, err := GetRegistry().GetPreviousContext(e.ctx, characterId)
    if err != nil {
        return "", "", "", nil, fmt.Errorf("failed to get conversation context for NPC ID: %w", err)
    }
    npcId := ctx.NpcId()

    payload := saga.ShowParcelPayload{
        CharacterId: characterId,
        NpcId:       npcId,
        WorldId:     f.WorldId(),
        ChannelId:   f.ChannelId(),
        Quick:       false,
    }

    return stepId, saga.Pending, saga.ShowParcel, payload, nil
```

- `CharacterId` — from the `createStepForOperation` parameter, same source as `open_storage`. PASS.
- `NpcId` — from `ctx.NpcId()`, i.e. `GetRegistry().GetPreviousContext(e.ctx, characterId)`, the same
  live conversation-context lookup `open_storage` uses (`operation_executor.go:2116-2121`). This is
  how the NPC id in seed data (9010009) actually reaches the payload at runtime — not hardcoded.
  Confirmed `ConversationContext.NpcId()` exists at `model.go:2258`. PASS.
- `WorldId` / `ChannelId` — from `f.WorldId()` / `f.ChannelId()`, the `field.Model` parameter, same
  source as `open_storage`. PASS.
- `Quick` — explicitly set to `false` (not left at zero-value implicitly; the assignment is present
  in the literal), matching the brief's "Quick is false on this path" (Step 3). Since `bool`'s zero
  value is `false`, an omitted field would look identical at runtime, but the source shows it is a
  deliberate assignment, not an omission. PASS.
- `stepId` is computed once at function entry (`stepId := fmt.Sprintf("%s-%d", operation.Type(), characterId)`,
  line 1142) and reused across all cases including this one, consistent with every other case. PASS.
- No `accountId`-equivalent param handling was added, correctly — the brief says `open_duey` takes no
  params, and the seed's `"params": {}` confirms nothing is expected. PASS.

No deviation from the `open_storage` template found.

## Priority 2 — the eight seed files (independent verification)

Ran `md5sum` directly (not trusting the report's claim):

```
1d8681975d89bd48cffbae25a418b14a  deploy/seed/gms/72_1/.../npc-9010009.json
1d8681975d89bd48cffbae25a418b14a  deploy/seed/gms/79_1/.../npc-9010009.json
1d8681975d89bd48cffbae25a418b14a  deploy/seed/gms/83_1/.../npc-9010009.json
1d8681975d89bd48cffbae25a418b14a  deploy/seed/gms/84_1/.../npc-9010009.json
1d8681975d89bd48cffbae25a418b14a  deploy/seed/gms/87_1/.../npc-9010009.json
1d8681975d89bd48cffbae25a418b14a  deploy/seed/gms/92_1/.../npc-9010009.json
1d8681975d89bd48cffbae25a418b14a  deploy/seed/gms/95_1/.../npc-9010009.json
1d8681975d89bd48cffbae25a418b14a  deploy/seed/jms/185_1/.../npc-9010009.json
```

All eight identical. Also checked the reference file `npc-2020004.json` across **all** version
directories present in the repo (11 total, including the 8 target versions plus `12_1`, `48_1`,
`61_1` which are out of this task's scope but present as additional evidence):

```
27e4839dced906b6278c77d91499fc55  (all 11 npc-2020004.json, gms + jms)
```

Also identical across every version. Both halves of the implementer's claim confirmed
independently. PASS.

JSON validity: `python3 -m json.tool` on one representative file succeeds (`OK72`); since all
eight are byte-identical this generalizes to all eight. PASS.

Structural coherence, read directly (`deploy/seed/gms/83_1/.../npc-9010009.json`):

```json
{
  "data": {
    "attributes": {
      "npcId": 9010009,
      "startState": "openDuey",
      "states": [
        {
          "genericAction": {
            "operations": [ { "params": {}, "type": "open_duey" } ],
            "outcomes": [ { "conditions": [], "nextState": null } ]
          },
          "id": "openDuey",
          "type": "genericAction"
        }
      ]
    },
    "id": "9010009",
    "type": "npc-conversation"
  }
}
```

`startState` ("openDuey") matches the single state's `id` — reachable. The single outcome's
`nextState: null` is a terminal transition, no dangling reference to a nonexistent state. Diffed
directly against the reference `npc-2020004.json` (`83_1`): identical shape, only the `npcId`,
`startState`/state `id` (`openStorage`→`openDuey`), operation `type`
(`open_storage`→`open_duey`), and the removed `accountId` param differ. Matches the brief's Step 4
description exactly. PASS.

## Priority 3 — `saga/model.go` re-export

`services/atlas-npc-conversations/atlas.com/npc/saga/model.go:+2`:

```go
ShowParcelPayload = sharedsaga.ShowParcelPayload   // in the type block
...
ShowParcel = sharedsaga.ShowParcel                 // in the const block
```

Read the full file. This is a pure type/const alias, placed alongside the existing
`ShowStoragePayload = sharedsaga.ShowStoragePayload` / `ShowStorage = sharedsaga.ShowStorage`
pair, in the same two blocks, same style, no redefinition or field reshaping. Confirmed this
mirrors the existing convention exactly — every other payload/action in the file follows the
identical `X = sharedsaga.X` pattern. Justified out-of-brief touch. PASS.

## Priority 4 — design §9.6 compliance

§9.6 requires the NPC id to live in seed data, not as a Go constant, with no new
`libs/atlas-constants` package for it. Checked:

- `grep -rn "9010009" libs/atlas-constants/` — no output. Not added there.
- `grep -rln "9010009" services/atlas-npc-conversations/` — only `operation_executor.go` (a comment:
  `// Used for Duey parcel delivery NPCs (e.g., NPC 9010009)`, line 2138) and
  `operation_executor_test.go` (test literals: `SetNpcId(9010009)`, assertion `p.NpcId != 9010009`).
  No `const` declaration anywhere. The runtime value flows only from
  `GetRegistry().GetPreviousContext(...).NpcId()`, which is populated from the conversation
  context set when the client interacts with the NPC (seed-data-driven), not a hardcoded constant.

PASS — no numeric NPC constant added, no `atlas-constants` extension.

## Test provenance — `TestExecuteOpenDuey`

Compared the test's expected values against the brief's own table (task-20-brief.md Step 1):

> `emits show_parcel` | NPC 9010009, character 100, world 0, channel 1 | returns action
> `saga.ShowParcel`, status `saga.Pending`, and a `ShowParcelPayload` with `CharacterId` 100,
> `NpcId` 9010009, `WorldId` 0, `ChannelId` 1, `Quick` false

The test (`operation_executor_test.go:490-537`) uses exactly these values: `characterId := uint32(100)`,
`SetNpcId(9010009)`, `field.NewBuilder(world.Id(0), channel.Id(1), ...)`, and asserts
`action == saga.ShowParcel`, `status == saga.Pending`, `p.CharacterId == 100`, `p.NpcId == 9010009`,
`p.WorldId == 0`, `p.ChannelId == 1`, `!p.Quick`. There is no existing `open_storage` test in this
file to have copied numeric values from (`grep -rn "open_storage" *_test.go` returns nothing) — the
brief's table is the only plausible source, and the values match it exactly. No evidence of
read-back-from-implementation pinning. The second subtest (`missing context`) asserts a genuine
error path (no context registered → `GetPreviousContext` fails → error containing "conversation
context", nil payload), which is a real negative-path assertion, not a tautology.

Ran both subtests directly: `go test ./conversation/... -run TestExecuteOpenDuey -v` — both PASS.
Also ran `go build ./...` for the module — clean.

## Findings

None. No blocking or non-blocking issues found within scope.

## Not evaluable

- Whether NPC 9010009's client-side script/dialogue script actually invokes this conversation
  script correctly at runtime (live client interaction) is outside this diff's reach — this task
  only wires the server-side operation and seed data, consistent with the brief.
