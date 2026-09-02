# GMS v87 NPCs teleport instead of walking — `CMovePath` is directional

Reported live on k3s `atlas-main`, tenant GMS 87.1: an NPC with a walk path
teleports around the map. Root cause is in `libs/atlas-packet/model/movement.go`,
in the per-element `XOffset`/`YOffset` pair on absolute-position (NORMAL)
movement fragments. That pair sits at `CMovePath::ELEM +0x14/+0x16` (named
`xOffset`/`yOffset` outright in the GMS v95 symbols) and is written between the
foothold and the common `bMoveAction`/`tElapse` tail. GMS v87 is the only client
in the matrix whose `CMovePath` is asymmetric about it: `CMovePath::Encode`
@0x6c70fe writes both (`mov ax,[edi+14h]` / `[edi+16h]` at 6c720a / 6c7218),
while `CMovePath::Decode` @0x6c6e86 never reads them — that function is 154
instructions ending at 0x6c709a, and its absolute arm goes from `fh` (and the
attr-15 `fhFallStart`) straight to the tail. GMS v83 @0x68a33c and v84 @0x6a0fd0
have the field on neither side; GMS v92 @0x65ad60, GMS v95 @0x667920 and JMS
v185 @0x70b3ce all read back what they write. Atlas had a single
`gmsMovementElementOffsets` gate (`!GMS || MajorAtLeast(87)`) applied identically
to `NormalElement.Decode` and `NormalElement.Encode`, with a comment requiring
the two to stay textually identical. Decode at 87 is correct; Encode at 87 is
not, so Atlas echoes four bytes the v87 client never consumes. The client then
reads `xOffset`'s low byte as `bMoveAction` and `yOffset` as `tElapse`, and reads
the real `bMoveAction`/`tElapse` as the next fragment's attr and body — the whole
element loop desyncs, which is the teleporting. This is a regression from
`dd233f7a2` (task-218), which moved both sides from 88 to 87 at once; before it
v87 encoded correctly but decoded wrong, producing the "Code [253/254/255] not
configured for use in movement" flood that task was chasing. Fix is to split the
gate into `movementElementOffsetsInbound` (unchanged, `!GMS || >=87`) and
`movementElementOffsetsOutbound` (`!GMS || >=92`, the lowest boundary backed by a
direct read-side decompile; `deploy/k8s/base/versions.json` ships no GMS version
between 87 and 92, so 88..92 are indistinguishable for any tenant Atlas serves).

Two things found alongside and corrected in the same change. First, the pair
guarded four bytes later by `CClientOptMan::GetOpt(..., 2)` at +0x18/+0x1A is
*not* these offsets: the v95 symbols name them `usRandCnt`/`usActualRandCnt`,
move-rand counters present on every fragment type and symmetric in both
directions. The old comment mistook them for the offsets and recorded a
"runtime option" caveat that does not apply. Second, three clientbound
round-trip tests (character, monster, pet) covered GMS v87 and, once encode and
decode legitimately diverge there, silently assert nothing: `request.Reader`
returns 0 for a read past the end without advancing, so the decode over-reads,
corrupts its own output, and still leaves `Available() == 0`. They now go through
`test.MovementRoundTrip`, which skips the identity assertion on the directional
versions; blob width stays pinned by the move-path byte oracle in
`libs/atlas-packet/model`.
