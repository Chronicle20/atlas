# Task-081 — baseline faithfulness pass (live-IDA verified)

After the serverbound family-operation wrapper fix (commit `84a843672`) exposed
that several serverbound baselines were *data-incomplete* (omitted the real
sub-op byte, carried wrong addresses/directions, or were never IDA-anchored),
this pass verified each against the live IDBs (multi-instance ida-pro-mcp:
v83=13337, v87=13338, v95=13339, jms185=13340) and corrected what could be
confidently confirmed. **No values were hand-normalized — every correction
below cites a live decompile.**

## Corrected (verified, then edited baseline)

| Version | Entry | Was | Now (verified) | IDA evidence |
|---|---|---|---|---|
| v95 | `CUIMessenger::OnCreate` | `[Encode4]` (sub-op omitted) | `[Encode1(0,ENTER), Encode4]` | `COutPacket(143)+Encode1(0)+Encode4` @0x7f59d0 |
| v95 | `CUIMessenger::SendInviteMsg` | `[EncodeStr]` (sub-op omitted) | `[Encode1(3,INVITE), EncodeStr]` | `COutPacket(143)+Encode1(3)+EncodeStr` @0x7f5820 |
| v95 | `CField::SendSetGuildMarkMsg` | `addr 0x0`, sub-op omitted | `addr 0x52d8c0`, `[Encode1(0xF), Encode2, Encode1, Encode2, Encode1]` | `COutPacket(149)+Encode1(0xF)+Encode2+Encode1+Encode2+Encode1` @0x52d8c0 |
| v95 | `CField::SendSetGuildNoticeMsg` | `addr 0x0`, sub-op omitted | `addr 0x535180`, `[Encode1(0x10), EncodeStr]` | `COutPacket(149)+Encode1(0x10)+EncodeStr` @0x535180 |
| v87 | `CUIMessenger::OnCreate` | `clientbound [Decode4]` (direction bug) | `serverbound [Encode1(0,ENTER), Encode4]` | `COutPacket(0x80)+Encode1(0)+Encode4` @0x8b62ed |

These were all **false ✅** before — the audit passed only because *both* sides
dropped the same real wire byte (or, for v87 `OnCreate`, the wrong direction
masked it). They are now **faithful ✅**: the export carries the sub-op as its
leading field and the audit composes Atlas's `Operation`/`BBS` wrapper onto the
body (adaptive composition, commit `84a843672`). Verdicts unchanged; content now
correct.

**Per-version sub-op values differ** — verified, not assumed: guild SET_MARK is
`0xF`/15 in v95 but `9` in jms; this is exactly why each was checked against its
own IDB rather than copied across versions.

## Deliberately NOT corrected (insufficient evidence — flagged, not guessed)

- **v87 `CUIMessenger::SendInviteMsg`** — the baseline address `0x8b978f`
  decompiles to `CUIMessenger::OnPacket(CInPacket*)` (a clientbound handler — a
  mis-address). The real `?SendInviteMsg@CUIMessenger@@...` @0x8ba1a0 decompiles
  to a StringPool/display path with **no `COutPacket` build**, so the actual
  v87 invite-send wire could not be confidently identified. Left as-is pending
  deeper investigation (the send may be inlined elsewhere or routed through a
  helper). Do not fabricate a correction.
- **v87 `CUIMessenger::OnDestroy`** — shares the same bogus address `0x8b978f`
  as the mis-addressed SendInviteMsg. Out of scope here; flagged as a v87
  baseline-address quality issue.
- **jms `CClientSocket::OnMigrateCommand`** — already ✅ in the committed audit;
  the baseline correctly traced the success branch `[byte, ip-bytes, int16]`
  matching Atlas's `ChannelChange`. It was a candidate-real **false positive**
  (the `if(!Decode1){ip;port}` branch the triage heuristic flagged); no fix
  needed.

## Resolved after the initial pass — jms `OnNpcChangeController` (descent truncation)

Re-examined and FIXED (not a dispatch-selector case after all). Atlas's
`SpawnRequestController` hardcodes `WriteByte(1)` for localFlag, so it ALWAYS
sends the spawn-with-control case and the client ALWAYS takes the `if(localFlag)`
branch — the 8 trailing reads are unconditional for Atlas's packet. They were
truncated in the baseline because they live in a **sub-function descent**, traced
live: `OnNpcChangeController@0x720782 → SetLocalNpc@0x720242 (Decode4 templateId)
→ CNpc::Init@0x716da2 (Decode2 x, Decode2 cy, Decode1 stance, Decode2 fh, Decode2
rx0, Decode2 rx1, Decode1 miniMap)`. Full 10-field read order
`[byte,int32,int32,int16,int16,byte,int16,int16,int16,byte]` matches Atlas
exactly → **❌→✅**. (GMS v95's baseline already had the full order.)

## "atlas-short" bucket triage (client reads more than Atlas writes — bug-suspicious)

Surveyed all 10 flat `atlas-short` ❌ (the profile most likely to be a real
missing-field bug). Result: **1 cluster was a real baseline defect (fixed), the
rest are modeling artifacts** — no Atlas wire bug found.

- **`PetCommand` (v83/v87/v95/jms)** — FIXED. Baseline traced a phantom 4th
  `Encode1` "success flag"; the live client sends exactly 3 fields
  (`EncodeBuffer(8)+Encode1(commandWithName)+Encode1(command)`, verified each
  IDB). Removed → ✅ all four (commit `a1c779253`).
- **`CashShopOperationSetWishlist` (v95, 1 vs 10)** and **`GuildSetTitleNames`
  (v95, 1 vs 5)** — NOT bugs. Atlas Decode does `make([]T,N); for i<N` and Encode
  does `for range` (fixed-count arrays, no length prefix). The **static analyzer
  flattens the Encode loop to ONE iteration** (`FlattenWithRegistry` inlines a
  `KindRepeat` body 1×), so it shows 1 field vs the client's N. Atlas is correct.
  Proper fix = repeat-expansion in the diff (match a `KindRepeat` body against the
  client's N-run) — a diff-algorithm change with broad blast radius across every
  loop packet; registered as a tool follow-up, not landed here.
- **`BuffCancel` (v83/v87/v95)** — `nChangedStatPoint` is a CONDITIONAL trailing
  read (`only if IsMovementAffectingStat`); the leading field is a 16-byte UINT128
  stat mask. Conditional-field modeling nuance, not a flat missing field.
- **`MessengerOperationInvite` (v83)** — same unreliable v83 messenger baseline as
  v87 `SendInviteMsg` (traces a path that sends fromName+myName+pad); needs the
  real v83 invite-send re-traced before any correction.

## Remaining ❌ bucket map (post-pass, 314 total across 4 versions)

Structural categorization of what remains (none are confirmed Atlas bugs; all are
modeling limitations or untraced baselines):
- **110 `#` per-mode entries** — clientbound switch-dispatch handlers
  (`OnPacket#Mode`, `OnGuildResult#*`). The audit `run` path flattens; resolving
  needs the dispatch-selector wired into `run` (the validate/infer path already
  models these). Largest single lever.
- **103 BranchDepth>0** — Atlas-side mode-keyed Encode switch (the analyzer
  flattens all arms). Static diff is positionally invalid here (per
  `reference_packet_audit_tool_mechanics`); real check = byte-level wire tests.
- **52 width-mismatch** — opaque register-boundary structs (AvatarLook), loops
  (NpcShopList), and version-width differences.
- **39 client-short** — more dispatcher-prefix / descent-truncation / empty-baseline
  cases (e.g. v83 BBS entries with empty `calls`); each a tractable per-packet IDA
  trace like the ones fixed this pass.
- **10 atlas-short** — triaged above (PetCommand fixed; rest are loop/conditional).

## Net

5 serverbound entries upgraded from coincidental/incorrect ✅ to live-IDA-faithful
✅ across v87/v95; zero regressions. Two jms candidates reclassified as genuine
per-mode-branch (dispatch-selector follow-up). One v87 send left honestly
unresolved rather than guessed.
