# Merging main (through task-190) into task-167

Merge of `origin/main` @ `a9ab9f62e` into `task-167-homing-beacon-bullseye`.
Four files conflicted. This records what was decided and why, because two of the
conflicts were semantic — task-167 and task-190 had made contradictory changes to
the same wire contract, not merely adjacent edits to the same lines.

## 1. Two-state mask and base blocks — task-190's model wins

`CharacterTemporaryStat.EncodeMask` / `getBaseTemporaryStats` /
`decodeBaseTemporaryStats`.

The two branches disagreed about whether the client reads a base-stat block per
SET mask bit, or reads the whole group unconditionally:

- **task-190 (main)**: bits and blocks are presence-gated on every version. A CTS
  claims only the stats it holds; absent members produce neither a bit nor a
  block. Its fixture pins a v83 mount give at 34 bytes — mask + defense + one
  13-byte block + trailer.
- **task-167 (branch)**: pre-95 clients read all 7 blocks unconditionally, so
  absent members were emitted as empty placeholder blocks with their mask bits
  always set; only v95's PartyBooster/GuidedBullet were `conditional`.

**Resolved to task-190's model.** It is trunk, it shipped, and its regression
tests assert the behaviour directly. Consequences inside this branch:

- `twoStateStat.conditional` removed — under presence-gating every member is
  conditional, so the flag distinguished nothing.
- The v61 Undead mask-bit special case in `EncodeMask` removed. It existed to
  preserve a bit that fixtures pinned as always-set; with presence-gating an
  absent Undead simply has no bit.
- `CancelMask` / `EncodeCancelMask` removed. Their whole purpose was to give the
  RESET path a present-stats-only mask while `EncodeMask` kept asserting the
  two-state group for SET. `EncodeMask` now does exactly that on both paths, so
  the pair was a second spelling of it. An unexported `activeMask()` was split
  out of `EncodeMask` so tests can assert the invariant without reading it back
  off the wire.

Kept from task-167, as genuinely additive and untouched by task-190:

- `narrowTimeField` — GMS v61's 12-byte base block and its bare 4-byte third
  field.
- v61 and v95 two-state group membership (both 6 members, different shapes).
- `NewGuidedBulletTemporaryStatWithOptions` — the populated HOMING_BEACON block.
- The v95 PartyBooster block.

Fixtures rewritten, not deleted: the per-version trailer sizes that
`TestCTSHomingBeaconPre95AbsentStaysEmpty` used to assert (88 bytes on v61's
6-member group, 110 on the classic 7-member group) still run, but now on
`TestCTSTwoStateGroupShape`, which reaches them by populating every member of the
group instead of by padding it. The falsifier for group membership survives the
change in gating.

## 2. BuffCancel trailing byte — unconditional, pending re-verification

task-190 writes the trailing `nSecondaryStatChangedPoint` byte unconditionally.
task-167 wrote it only when the cancel mask intersects a per-version movement
filter it derived (`movementAffectingStatNames` / `MovementAffectingMask`).

**Resolved to the unconditional write**, which is the fail-safe direction: an
unread trailing byte is slack the client ignores, while omitting one the client
does read runs it off the end of the packet.

This is not a judgement that task-167's filter is wrong — it is that the filter
is not yet safe to gate on. Per `evidence/movement-filter.md`, every version's
filter function was located and its constants counted, but the individual bit
NAMES resolve from symbols only on v61, v83 and v95. On v72 and v79 six of the
nine main-sequence names are unverified (and an index-for-index lookup against
the v83-tuned registry was explicitly falsified on v72 at bit 35). v92's 13
constants are none of them name-resolved, with Flying/Frozen inferred from v87's
positions. JMS raw shift 126 has no registry entry at all. One wrong bit in that
set omits a byte the client reads.

`MovementAffectingMask` and its membership test are **retained** in the model,
exported and test-covered, deliberately unwired. Closing the naming gap on
v72/v79/v92/JMS should flip the gate back on in `BuffCancel.Encode` /
`BuffCancelForeign.Encode`, not redo the derivation.

Fixture consequences: cancels that carry no movement-affecting stat are one byte
longer than task-167 pinned them. `TestBuffCancelBeaconOnlyV83` and
`TestBuffCancelBeaconOnlyV95` go 16 -> 17 bytes, `TestBuffCancelV48ByteFixture`
8 -> 9. The mask contents — the part task-167 derived — are unchanged.

## 3. atlas-buffs processor — mechanical

task-190 extracted `expireInto` and added `ExpireForCharacter`; task-167 threaded
a `noExpiry` flag through `Apply` and the status-event providers. No conflict of
meaning: took `expireInto`, added `eb.NoExpiry()` to the provider call inside it,
and added the `noExpiry` argument to task-190's four new `Apply` call sites in
`processor_test.go`.

## Superseded sections of this task's own docs

`design.md` §3 F1 (cancel-mask shape) and §5.5.3 (conditional trailing byte)
describe the pre-merge design. Both are superseded by the decisions above; the
evidence they cite remains valid and is what a re-verification pass should start
from.

## Verification

`libs/atlas-packet`, `services/atlas-buffs/atlas.com/buffs` and
`services/atlas-channel/atlas.com/channel` each build clean and pass their full
test suites; `gofmt -l` is empty across the changed trees.

---

# Follow-up: mount and GM-hide migrated off the MaxInt32 sentinel

PRD FR-2.1 rejects `duration = MaxInt32` as the representation of a
non-expiring buff and requires an explicit one. task-167 added that
(`buff.NewNoExpiryBuff`, `noExpiry` through the model/Kafka/REST contract) but
wired only the beacon to it, leaving the two existing sentinel users in place.
Both are now migrated, and the sentinel constants are gone:

- `skill/handler/mount.go` — `MountBuffDuration` deleted; the `applyBuff` seam
  drops its `duration` parameter and the production wiring calls
  `bp.ApplyNoExpiry`. Three call sites (tamed mount, skill-only mount,
  battleship).
- `skill/handler/hide/hide.go` — `HideBuffDuration` deleted; `applyHide` calls
  `bp.ApplyNoExpiry`.

The stale comments both files carried ("there is no 'never expires' path
through atlas-buffs") are gone with them.

`mount_test.go` loses its `applyDur` spy field and the three assertions that
pinned the duration to `MaxInt32`. The property they guarded is now structural:
the seam has no duration to get wrong.

## The wire-boundary bug this exposed

A no-expiry buff carries `expiresAt = time.Time{}`. `CharacterTemporaryStat`
encoded the per-stat expiry as `int32(v.ExpiresAt().Sub(time.Now()).Milliseconds())`,
and on the zero time that does not merely go negative — year 1 to now exceeds
what an int64 nanosecond `Duration` can represent, so the subtraction saturates
and the int32 truncation lands on an arbitrary negative (**-2077252342**). The
client reads that as long expired.

The beacon never hit it: `HOMING_BEACON` is in `baseStatNames`, so it encodes as
a base-stat block and skips the per-stat expiry field entirely. `MONSTER_RIDING`
is a base stat too, so mounts are also unaffected. But `DARK_SIGHT` is an
ordinary value stat (registry shift 90), so **GM hide would have gone out with a
garbage negative duration** — visible as hide dropping the instant it was cast.

Fixed at the wire boundary, in `remainingMillis` and `legacyDurationUnits`: the
zero time encodes as `MaxInt32` (`MaxInt16` units on the pre-v61 short). No
client has a no-expiry concept — each one reads a concrete duration — so the
sentinel has to reappear somewhere. Putting it in the encoder is what keeps it
out of the domain model and off the Kafka and REST contracts, which is what
FR-2.1 actually asked for. `legacyDurationUnits` also gained the clamp it was
missing, so a long finite buff saturates its short instead of wrapping.

Covered by `TestNoExpiryStatEncodesSaturatedDuration` (DARK_SIGHT, the stat that
actually reaches the field) and `TestLegacyDurationUnitsNoExpirySaturates`.
