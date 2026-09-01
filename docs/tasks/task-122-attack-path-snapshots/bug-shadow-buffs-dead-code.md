# bug: shadow verifier's buff comparison is dead code; no STAT_UPDATED arm on the buff snapshot feed

Source: adversarial review round (adversarial-attack-path.md blocking #1;
adversarial-event-coverage.md non-blocking #1). Both findings are in the buff
component of the task-122 snapshot and land together.

## Symptom

1. `compareProjection`'s buff-divergence branch (SOUL_ARROW / SHADOW_PARTNER
   projectile gates) can never execute: both call sites of `maybeShadow` pass
   `servedBuffs=nil`, so the `component="buffs"` divergence metric can never
   fire. design.md §8 claims shadow verification compares active
   projectile-gate buffs; the landed code does not. Buffs is precisely the
   component with a documented residual staleness risk (atlas-buffs restart
   drops buffs silently — event-coverage.md §5), so the safety net is blind
   where it matters most.
2. atlas-buffs emits `STAT_UPDATED` when a buff's Amount is mutated in place
   (`services/atlas-buffs/atlas.com/buffs/character/processor.go:229-256`,
   `UpdateStatValue` — live producers: Aran Combo orb count, Energy Charge).
   The channel buff consumer's snapshot feed has no handler for it, so any
   buff later read via `sp.GetBuffs()` after an in-place mutation would be
   frozen at its APPLIED-time value. Currently harmless only because both
   live producers are read via separate mirrors/live REST — an unstated,
   unenforced invariant.

## Root cause

1. The `servedBuffs` parameter was threaded through `maybeShadow` /
   `compareProjection` but never populated at either call site.
2. event-coverage.md §5 never enumerated `STAT_UPDATED`, so no consumer arm
   was planned.

## Fix

Two parts, same branch (task-122-attack-path-snapshots worktree):

1. **Thread served buffs into `maybeShadow`.** At both call sites
   (`services/atlas-channel/atlas.com/channel/character/snapshot/processor.go:61,151`),
   pass the buff set the snapshot actually served for the swing instead of
   nil, so `shadow.go:139`'s buff comparison runs. If a call site serves a
   composed model without having served buffs (buffs component invalid /
   not consulted for that read), passing nil remains correct for that site —
   the goal is that a swing whose projectile gate consulted snapshot buffs
   compares those same buffs. Add/extend a shadow test proving the buffs
   divergence metric fires on an injected divergence (currently impossible).
2. **Add a `STAT_UPDATED` arm to the buff snapshot feed.** In
   `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go`
   (existing handlers around lines 47-103), handle the STAT_UPDATED status by
   invalidating the buff component
   (`Registry.InvalidateBuffs`, `character/snapshot/registry.go:636`) —
   invalidate, don't apply: the thin-event pattern per design.md §2. Follow
   the existing snapshot-handler registration pattern in that consumer, and
   add a consumer test asserting STAT_UPDATED → buffs invalidated.

### Files

- services/atlas-channel/atlas.com/channel/character/snapshot/processor.go (call sites :61, :151)
- services/atlas-channel/atlas.com/channel/character/snapshot/shadow.go (compareProjection buff branch :139)
- services/atlas-channel/atlas.com/channel/character/snapshot/shadow_test.go
- services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go
- services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer_test.go
- Reference (read-only): services/atlas-buffs/atlas.com/buffs/character/processor.go:229-256 for the STAT_UPDATED payload shape

### Decisions already made

- STAT_UPDATED invalidates; it does not apply in place. If the event payload
  turns out to carry absolute values sufficient for an idempotent in-place
  apply, still invalidate — the arm exists as insurance, not a hot path, and
  the two live producers are read via mirrors anyway.
- No producer-side change in atlas-buffs.
- Module-local `go build ./... && go test ./...` only; the repo gate runs
  separately in a clean context.
