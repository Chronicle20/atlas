# Gate merge fix report — atlas-monster-death/party TestTransformRoundTrip

## What I implemented

Per the brief's settled diagnosis, `Transform` in
`services/atlas-monster-death/atlas.com/monster/party/rest.go` silently
dropped `leaderId` and `members`, so any populated `Model` failed the
round-trip test after the merge paired this branch's new `Transform` with
main's pre-existing `Extract`.

1. Completed `Transform(m Model) (RestModel, error)` to populate
   `LeaderId: m.leaderId` and `Members` by mapping each `m.members` entry
   through a new `TransformMember`.
2. Added `TransformMember(m MemberModel) (MemberRestModel, error)` as the
   true inverse of `ExtractMember`, reading `WorldId()`, `ChannelId()`,
   `MapId()`, and `Instance()` off `field.Model` (confirmed via
   `go doc github.com/Chronicle20/atlas/libs/atlas-constants/field Model`
   rather than assumed).
3. Strengthened `TestTransformRoundTrip` in `rest_test.go` to build a
   `Model` with a non-empty `leaderId` and two members, each built via
   `ExtractMember` (same convention noted in the brief for
   `services/atlas-login/atlas.com/login/guild/rest_test.go`), so the field
   value is constructed identically on both sides of the round trip. This
   makes the member-less nil-vs-empty-slice asymmetry moot, as the brief
   flagged as the preferred resolution.

Did not touch `Extract`, `ExtractMember`, or the three pre-existing
`Extract`-drops-field defects (`atlas-channel/character`,
`atlas-cashshop/character`, `atlas-npc-shops/character`) — out of scope per
the brief.

## What I tested

Module-local build and test, from the module root:

```
cd <repo-root>/services/atlas-monster-death/atlas.com/monster
go build ./... && go test ./party/...
```

Output:
```
Go test: 6 passed in 2 packages
```

(all 6 tests in `party` pass, including the strengthened
`TestTransformRoundTrip`; the other package in the module was already
passing and untouched)

`gofmt -l` on both touched files produced no output (clean):
```
gofmt -l services/atlas-monster-death/atlas.com/monster/party/rest.go services/atlas-monster-death/atlas.com/monster/party/rest_test.go
```

## Files changed

- `services/atlas-monster-death/atlas.com/monster/party/rest.go` —
  completed `Transform`; added `TransformMember`.
- `services/atlas-monster-death/atlas.com/monster/party/rest_test.go` —
  strengthened `TestTransformRoundTrip` with populated `leaderId` and two
  members built through `ExtractMember`.

## Self-review

- `Transform`/`TransformMember` are exact structural inverses of
  `Extract`/`ExtractMember`, including the `make([]MemberRestModel, 0)`
  non-nil-empty-slice convention mirroring `Extract`'s
  `make([]MemberModel, 0)`, so both directions normalize consistently.
- Test builds member field data (`WorldId`, `ChannelId`, `MapId`,
  `Instance`) through the real `field.NewBuilder(...).SetInstance(...).Build()`
  path (via `ExtractMember`), not hand-constructed, per the brief's
  guidance.
- No new dependencies, no changes outside the two named files, no
  `TODO`/stub code.
- Did not weaken the test or change `Extract`'s semantics — matches the
  brief's explicit ambiguity resolution (complete `Transform`, not paper
  over with a weaker assertion).

## Issues or concerns

None. Fix is scoped exactly to the one failing check named in the brief;
the module-local gate is green.

## Commit

`21bdd9fec fix(atlas-monster-death): complete party Transform as true inverse of Extract`

Verified after commit: branch is `task-263-backend-guideline-conformance`,
worktree root is the `task-263-backend-guideline-conformance` worktree
under `.worktrees/`.
