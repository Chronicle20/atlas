# Task 262 — Reference-fidelity adjudication (design §2.2 gate)

## Inputs

Neither input is committed. They are referred to here by placeholder:

| Placeholder | Path (uncommitted, local) | What it is |
|---|---|---|
| `$WZ_ARCHIVE` | `tmp/83.1_wz/Reactor.wz` | PKG1 binary, 54,133,811 bytes (51.6 MiB), mtime 2010-02-17. One file of a complete 18-file stock GMS v83 WZ set. |
| `$WZ_REFERENCE` | `tmp/<tenant-uuid>/GMS/83.1/Reactor.wz/` | A **directory** of 421 `.img.xml` files. There is no `.wz` binary anywhere under that tenant directory. |

All offsets below are absolute byte offsets into `$WZ_ARCHIVE`. All traces were
produced with `wzdiff --archive "$WZ_ARCHIVE" --trace <image>`; all hexdumps with
`od -Ad -tx1z -j <offset> -N <len> "$WZ_ARCHIVE"`. The decode is against
`libs/atlas-wz/wz/image.go` (`parsePropertyList` / `parsePropertyValue` /
`parseExtendedProperty` / `parseCanvasProperty`).

## Step 1 — the baseline reproduces

`wzdiff` against the two inputs above exits 1, reports `local image count: 421
ours: 419`, `images only in HaRepacker dump: [9400300 9400301]`, and 19 divergent
images. Its body is **byte-identical** to the committed
`evidence-wz-parse-divergence-reactor.txt`; the only difference is the
header/log-line ordering that logrus writes to stderr. The tool is trustworthy.

## Verdict — a third label is required: `INPUT-MISMATCH`

The gate was framed as a two-way choice: each delta is either a `PARSER-DEFECT`
in `libs/atlas-wz/wz` or a `REFERENCE-RESOLUTION` artifact of how HaRepacker
exported the dump. **Neither label applies to any of the 21 items.** The byte
adjudication below shows that `$WZ_ARCHIVE` and the archive that produced
`$WZ_REFERENCE` are *different files with different content*. A structural diff
between them cannot adjudicate a parser at all.

Two independent classes of evidence establish this.

### A. Our parse is byte-faithful to `$WZ_ARCHIVE` everywhere it was checked

Every WZ type-9 sub-object declares its own byte length immediately before its
body. The Task 2 trace hook records `actualEnd` — where the parse really
finished — *before* the recovery reseek to `endPos` heals it. Across the traces
of all 19 divergent images there are **1136 type-9 sub-objects and 0 instances
of `actualEnd != endPos`**:

```
$ cat /tmp/tr-*.txt | grep -o 'endPos=[0-9]* actualEnd=[0-9]*' \
    | awk -F'[= ]' '$2!=$4' | wc -l
0
$ cat /tmp/tr-*.txt | grep -c 'kind=sub'
1136
```

A parser that dropped or gained a child, or mis-sized a name, would have to
mis-size it by exactly zero bytes 1136 times in a row to produce that. Every
specific disputed node was then hand-decoded from the raw bytes (§ per-image
below) and in every case the archive literally says what our parser reported.

### B. The reference dataset is a different, heavily customised WZ set

`wzdiff` was run on three further WZ files from the *same* stock set against the
*same* tenant dump:

```
Etc.wz    local image count: 24    ours: 22     only in dump: [DeveloperNpc MapNeighbors]
Quest.wz  local image count: 6     ours: 6
Npc.wz    local image count: 6962  ours: 1620
```

`Npc.wz` in the reference dump contains **5342 NPC images that do not exist in
the stock archive**, in the `9901xxx` custom range. `Etc.wz` contains
`DeveloperNpc.img` and `MapNeighbors.img`, neither of which is stock GMS v83
content. No property reader, and no HaRepacker export setting, can conjure 5342
images. The reference dump is an export of a *customised private-server WZ
dataset*, not an export of `$WZ_ARCHIVE`.

The PRD (line 20) describes `$WZ_ARCHIVE` as "the tenant's `GMS/83.1/Reactor.wz`
(PKG1, 51.6 MiB)". That identification is wrong: the 51.6 MiB PKG1 file lives in
`tmp/83.1_wz/` alongside a full stock 2010-02-17 WZ set, and the tenant directory
contains no `.wz` binary at all. The two inputs were conflated when the evidence
file was first produced, and every downstream conclusion in the PRD inherits that.

### Step 2's clause 2 — the HaRepacker export setting

Not recorded, and it does not matter here: clause 1 (byte adjudication) is
conclusive on its own, and clause 2 is explicitly "recorded, not trusted over
(1)". Two properties of the exporter *are* directly observable from the dump and
are worth stating, because they rule out the two resolution behaviours the gate
suspected:

- **It does not resolve 1×1 stub canvases.** Reference `9400300.img.xml` and
  reference `9202005.img.xml` `/5/canvas:0` both contain literal
  `width="1" height="1"` canvases. An exporter that expanded `_inlink`/`_outlink`
  stubs would not have left those.
- **It does not resolve `uol`.** Reference `2618000.img.xml` contains literal
  `/imgdir:6/imgdir:hit/uol:0 = ../0` and `uol:1 = ../1`.

So "HaRepacker resolved it" was never available as an explanation for the
`2519000`/`2519001` canvas deltas or the `2006001` `uol` delta.

## Per-image adjudication

### `2006000` — INPUT-MISMATCH

Reference has an extra `/0/event` subtree. Our `/0` property list header:

```
/0 kind=list start=1587506 count=10
1587506:  0a 00 ff 0c 09 b1
```

`0a` is the literal `ReadWzInt` child count (10), immediately followed by
`00 ff 0c` — an inline 1-char ASCII name (`0`) — and property type `09`. The
list is coherent and admits exactly 10 children. `event` is not one of them.

### `2006001` — INPUT-MISMATCH (answers both halves of the brief's question)

- (a) **Yes**, `/1/uol:0 = ../0/0` is literally in the bytes:
  `/1/0 kind=uol start=758863 end=758872 ../0/0`, and `/1/hit kind=uol` at
  `758887`.
- (b) **Yes**, `imgdir:2` … `imgdir:7` exist literally in the bytes, each with
  its own `kind=list` header and its own UOL children (`/2` count=2 @758913,
  `/3` @758969, `/4` @759025, `/5` @759081, `/6` with `event`+`hit` @759151ff,
  `/7`).

So neither half is a parser defect, and the image is not `MIXED`. The reference
image is simply a different image: its root children, in stored order, are
`[info, action, 1, 0]` — `action` before the numbered directories, and only two
of them.

### `2406000` — INPUT-MISMATCH

Reference has `/info/int:activateByTouch = 1` and `/0/event`; we have neither.
`/info` decoded from `32568701`:

```
32568701  1b 01 00 00 00 | 00 00 | 01 | 00 fc 52 64 fe 6c | 08 | 00 08 <16 bytes>
32568733  0b 00 ff 0c 09 1c ff 00
```

`1b 01 00 00 00` — offset-referenced string block resolving to `Property`;
`00 00` — the 2-byte skip; **`01` — the literal child count**; `00 fc …` — inline
ASCII name of length `0xfc = -4` → `name`; `08` — string; `00 08` + 16 bytes — an
8-char UTF-16 value. That ends at `32568734`, exactly the `endPos` implied by the
sub-object's own `declaredSize=33`. There is no room for a second child.
`activateByTouch` is not in this archive.

`/0`'s count byte at `32568749` is `02` (children `0` and `hit`); the reference
has three.

### `2408001` — INPUT-MISMATCH (not a name defect)

The archive contains **both spellings, at different paths**:

```
/1/rpeat  @52240360:  00 fb 4e 75 f6 68 92 | 03
/3/repeat @52685787:  00 fa 4e 60 e3 6c 87 21 | 03
```

`0xfb` as a signed byte is `-5` → a 5-character ASCII name; `0xfa` is `-6` → a
6-character ASCII name. The length is in the byte stream, ahead of the ciphertext.
A 5-byte name cannot decrypt to the 6-character string `repeat`. `$WZ_ARCHIVE`
genuinely spells `/1`'s property `rpeat` and `/3`'s `repeat`; the reference
archive spells both `repeat`. This is the one delta that most looked like a
parser defect and it is not one.

### `2502002` — INPUT-MISMATCH

Two `state` values differ (we read 1, reference reads 0) and the reference has an
extra `/0/event/1/int:2`.

```
/0/event/0/state @5775489:  01 ...
/0/event/1/state @5775575:  01 ...
/0/event/1 kind=list start=5775560 count=6
```

Both `state` values are the literal single-byte `ReadWzInt` `01`. `/0/event/1`'s
literal child count is 6 (`type, state, 0, 1, lt, rb`). In the reference XML the
extra `<int name="2" value="1"/>` appears **after** `lt`/`rb` in `imgdir:1` but in
its natural position in `imgdir:0` — an appended property, i.e. the signature of
a hand edit in an editor, not of a different decode of the same bytes.

### `2519000`, `2519001` — INPUT-MISMATCH (answers the brief's question: yes, a genuine 1×1)

The brief asked whether `1/canvas:0` is genuinely a 1×1 stub in the bytes. It is,
and it carries **no** `_inlink`/`_outlink`:

```
2519000  /1/0 kind=list  count=1                 (sole child: origin)
         /1/0 kind=canvas w=1 h=1 format=1 dataOffset=47557355 dataSize=11
         /1/0 kind=sub    declaredSize=50 endPos=47557366 actualEnd=47557366
2519001  /1/0 kind=list  count=1
         /1/0 kind=canvas w=1 h=1 format=1 dataOffset=52756815 dataSize=11
         /1/0 kind=sub    declaredSize=50 endPos=52756826 actualEnd=52756826
```

The canvas has exactly one child property, `origin`, which decodes to `(0,0)`,
and 11 bytes of pixel data — and the sub-object consumes its declared 50 bytes
exactly. The reference has `100×121 origin (49,121)` / `92×124 origin (47,124)`.
Since there is no link property to resolve, and since the exporter demonstrably
preserves 1×1 stubs elsewhere (§ Step 2 clause 2), the reference's canvas is
different stored content, not a resolved version of ours. `origin` differing as
well settles it: `origin` is a sibling property of the canvas, and no link
resolution rewrites it.

### `2519002`, `2519003` — INPUT-MISMATCH (answers the brief's question: yes, a literal link and no body)

```
2519002 root kind=list start=13965571 count=2      (info, action)
        /info kind=list start=13965590 count=2     (info, link)
        /info kind=sub declaredSize=43 endPos=13965626 actualEnd=13965626
2519003 root kind=list start=527703   count=2      (info, action)
        /info kind=list start=527722   count=2     (info, link)
        /info kind=sub declaredSize=43 endPos=527758 actualEnd=527758
```

The whole image is 2 root children — there is no `imgdir:0` and no `imgdir:1`
body anywhere in it — and `info` holds exactly `info` and `link`. Our two-node
parse is a faithful read of `$WZ_ARCHIVE`.

This is the one pair where the brief's `REFERENCE-RESOLUTION` hypothesis is
*consistent* with the delta taken alone: the reference's `2519002` content does
match the reference's `2519000` content, which is what a link-resolving exporter
would produce, and the reference's `info` has no `link` child. But the same
exporter demonstrably does **not** resolve `uol` or 1×1 stubs, and the
`2519000`/`2519001` deltas above prove the reference's `2519000` itself differs
from ours in stored bytes. The economical explanation covering all 21 items is
the input mismatch, so that is the label recorded here. If the controller wants
this pair adjudicated independently of the input-mismatch finding, it needs the
*matching* archive; from `$WZ_ARCHIVE` alone the delta is not attributable to our
reader either way.

### `2618000` — INPUT-MISMATCH (not a name misread; not a `#N` suffix artifact)

Root list at `34198176`, literal `count=9`: `info` plus directories named `0`
through `7`. Each name is its own inline 3-byte string block at a distinct
offset (`0` @34198227, `1` @34203037, `2` @34215669, `3` @34228860, `4`
@34241881, `5` @34255286, `6` @34268752, `7` @34325801) — tag byte, length byte
`0xff` (= −1, one ASCII char), one ciphertext byte. **The names `6` and `7` are
read from the bytes, not synthesised, and there are no duplicate siblings here,
so `wzdiff`'s decode-order-scoped `#N` suffix is not in play.** The structure is
not shifted by us.

It is shifted in the reference: the reference's root children, **in stored
order**, are `[info, 0, 1, 2, 3, 4, 5, 8, 6, 7]`. A directory named `8` sits
between `5` and `6`. That is one more directory than `$WZ_ARCHIVE` has, inserted
mid-list — which is exactly why the reference's `/7` holds what our `/6` holds.

The `delay` sub-question resolves the same way:

```
/7/0/delay @34325863:  80 50 c3 00 00
```

`0x80` is the WZ compressed-int escape; the following little-endian int32 is
`0x0000c350` = **50000**. Our read is exact. The reference's `150` is a
different stored value in a different file.

### `2618003`, `2618004`, `2618005`, `2618007`, `9208003` — INPUT-MISMATCH

All five are the same shape: the reference has an extra `/0/event` subtree we do
not emit. In all five the literal `/0` child count is `02`:

```
2618003 @8853542:   02 00 ff 0c 09 bd
2618004 @14208525:  02 00 ff 0c 09 f4
2618005 @6826492:   02 00 ff 0c 09 5d
2618007 @639076:    02 00 ff 0c 09 bd
9208003 @30536166:  02 00 ff 0c 09 50
```

Each count byte is immediately followed by a well-formed inline 1-char name and
property type `09`, and each parent sub-object consumed its declared size
exactly. Two children, not three.

### `2618006` — INPUT-MISMATCH

Root list at `27268074`, literal `count=5`: `info, 0, 1, 2, 3`. The reference's
root children are `[info, 0, 1]` — two fewer directories, with content
correspondingly re-indexed. Nothing here is a name or offset error on our side;
the reference image has a different shape.

### `3002000` — INPUT-MISMATCH (our parse of `$WZ_ARCHIVE` is right; see caveat)

The brief asked whether the bytes contain the `/1/imgdir:event` subtree with
`timeOut=2000` that only our parse produces. **They do.**

```
/1 kind=list start=41964503 count=2               (0, event)
/1 kind=stringblock name=event start=41983657 end=41983662
/1/event kind=list start=41983674 count=2         (0, timeOut)
/1/event/0 kind=list count=2                      (type, state)
/1/event/0 kind=sub declaredSize=23  endPos=41983706 actualEnd=41983706
/1/event/timeOut kind=prop type=3 start=41983716 end=41983721
/1/event kind=sub declaredSize=54  endPos=41983721 actualEnd=41983721
/1 kind=sub declaredSize=19225 endPos=41983721 actualEnd=41983721
```

`/1` declares two children; the second is named `event`; `event` declares two
children, `0` and `timeOut`; and every enclosing sub-object consumes its declared
size to the byte. We do not fabricate this subtree — it is in `$WZ_ARCHIVE`.

**Caveat on PRD Open Question 4.** The PRD asks whether HaRepacker is *skipping a
record it does not model*. The answer is no — it is not skipping anything,
because it never saw this subtree: the reference archive's `3002000` `/1` does
not contain it. The correct statement is "our parse of `$WZ_ARCHIVE` is right and
the reference is a different file", **not** "the reference is lossy". PRD
acceptance criterion "0 divergent images in either direction" cannot be met for
`3002000` by fixing the reader, because there is nothing to fix.

### `9202005` — INPUT-MISMATCH

Reference has `/5/event` and an (empty) `/5/hit`. Literal `/5` count byte at
`13411658` is `01`; its sole child is a `1×1` canvas named `0`
(`dataSize=11`, `declaredSize=50` consumed exactly). Note that the reference
agrees with us that this canvas is `1×1` — it is not among the flagged deltas —
which is the same-file cross-check that shows the exporter preserves 1×1 stubs.

### `9208004` — INPUT-MISMATCH

Two things differ. The extra `event`/`hit` subtrees under `/0`…`/3`: literal
child counts are `0d` (13) at `14679079`, `15128666`, `15578229` and `0e` (14) at
`16027792` — the reference needs 15/15/15/15. And `/4/canvas:0/vector:origin`,
where we read `(0,0)` and the reference reads `(115,222)`:

```
/4/0 kind=list  count=1
/4/0 kind=canvas w=1 h=1 format=1 dataOffset=16667064 dataSize=11
/4/0 kind=sub   declaredSize=50 endPos=16667075 actualEnd=16667075
```

Both sides agree the canvas is `1×1` (the dimensions are not among the flagged
deltas) and disagree only on `origin`. A same-dimension canvas with a different
`origin` is stored-content difference by definition — there is no resolution or
decode step that rewrites `origin` while leaving `width`/`height` alone.

### `9400300`, `9400301` — INPUT-MISMATCH

Present only in the reference (421 vs 419). Both are Romeo-and-Juliet
party-quest reactors, and `9400300`'s `/0/canvas:0` is a `1×1` stub with
`origin (33,80)` — the same origin as `2618000`'s frames, i.e. content derived
from another reactor. `$WZ_ARCHIVE`'s Reactor directory has no `94*` entry at
all. A directory reader that silently dropped two entries would desync the
remaining ones; instead 402 of the 419 images we do enumerate are structurally
identical to the reference. The two images are absent from the archive, not lost
by the reader.

## Auditable label table (21 items)

| # | image | label | key byte evidence |
|---|---|---|---|
| 1 | `2006000` | INPUT-MISMATCH | `/0` count `0a` @1587506 |
| 2 | `2006001` | INPUT-MISMATCH | `uol` @758863/758887; `/2`…`/7` lists @758913–759151 |
| 3 | `2406000` | INPUT-MISMATCH | `/info` count `01` @32568708; `/0` count `02` @32568749 |
| 4 | `2408001` | INPUT-MISMATCH | name lengths `0xfb`(5) @52240360, `0xfa`(6) @52685787 |
| 5 | `2502002` | INPUT-MISMATCH | `state` `01` @5775489, @5775575; `/0/event/1` count 6 |
| 6 | `2519000` | INPUT-MISMATCH | `1×1` canvas, `dataSize=11`, `declaredSize=50` @47557312 |
| 7 | `2519001` | INPUT-MISMATCH | `1×1` canvas, `dataSize=11`, `declaredSize=50` @52756772 |
| 8 | `2519002` | INPUT-MISMATCH | root count 2 @13965571; `/info` count 2 @13965590 |
| 9 | `2519003` | INPUT-MISMATCH | root count 2 @527703; `/info` count 2 @527722 |
| 10 | `2618000` | INPUT-MISMATCH | root count 9 @34198176, names `0`–`7` literal; `delay` `80 50 c3 00 00` @34325863 |
| 11 | `2618003` | INPUT-MISMATCH | `/0` count `02` @8853542 |
| 12 | `2618004` | INPUT-MISMATCH | `/0` count `02` @14208525 |
| 13 | `2618005` | INPUT-MISMATCH | `/0` count `02` @6826492 |
| 14 | `2618006` | INPUT-MISMATCH | root count 5 @27268074 |
| 15 | `2618007` | INPUT-MISMATCH | `/0` count `02` @639076 |
| 16 | `3002000` | INPUT-MISMATCH | `/1` count 2 @41964503; `/1/event` `declaredSize=54` consumed |
| 17 | `9202005` | INPUT-MISMATCH | `/5` count `01` @13411658 |
| 18 | `9208003` | INPUT-MISMATCH | `/0` count `02` @30536166 |
| 19 | `9208004` | INPUT-MISMATCH | counts `0d`/`0d`/`0d`/`0e`; `/4/0` `1×1` `declaredSize=50` @16667021 |
| 20 | `9400300` | INPUT-MISMATCH | absent from the archive's directory (419 entries) |
| 21 | `9400301` | INPUT-MISMATCH | absent from the archive's directory (419 entries) |

Counts: `PARSER-DEFECT` **0**, `REFERENCE-RESOLUTION` **0**, `MIXED` **0**,
`UNDETERMINED` **0**, `INPUT-MISMATCH` **21**.

## Consequence for PRD FR-5

**Every** row of the FR-5 table is withdrawn as a defect claim. Not one of the 19
divergences, and neither of the 2 missing images, is attributable to
`libs/atlas-wz/wz`. Specifically:

- The "whole subtrees lost" rows (`2006000`, `2406000`, `2618003`–`2618007`,
  `9202005`, `9208003`, `9208004`) are content the archive does not contain.
- The "subtree gained" row (`3002000`) is content the archive *does* contain and
  the reference file does not.
- The "property name mangled" row (`2408001` `rpeat`) is the archive's own
  spelling, proved by the name-length byte.
- The "419 vs 421" row is the archive's own image count.

The acceptance criterion "0 divergent images in either direction against the
HaRepacker reference" is **unreachable with these two inputs** and would remain
unreachable after any correct change to the reader. It can only be restated
against a reference dump exported from the *same* archive.

## What would make this gate answerable

One of:

1. The `.wz` binary that the reference dump was exported from. Then `wzdiff`
   compares like with like and any residual delta is a genuine parser question.
2. A HaRepacker (or other independent literal reader) export of `$WZ_ARCHIVE`
   itself. Then the same diff runs in the other direction.

Until one of those exists there is no `REFERENCE-RESOLUTION` delta to allowlist,
and `allowlist.tsv` is therefore **not** produced by this task: writing one would
assert 19 reference-resolution artifacts that the bytes do not support. That call
belongs to the controller, not to this gate.

## Reproduction

```sh
cd libs/atlas-wz
go build -o /tmp/wzdiff ./cmd/wzdiff

# Step 1 baseline
/tmp/wzdiff --archive "$WZ_ARCHIVE" --reference "$WZ_REFERENCE"

# Per-image trace used for every offset quoted above
/tmp/wzdiff --archive "$WZ_ARCHIVE" --trace 2408001

# The declared-size sweep
/tmp/wzdiff --archive "$WZ_ARCHIVE" --trace <image> \
  | grep -o 'endPos=[0-9]* actualEnd=[0-9]*' | awk -F'[= ]' '$2!=$4'

# Cross-check on other WZ files of the same stock set
/tmp/wzdiff --archive "<stock>/Npc.wz" --reference "<tenant>/GMS/83.1/Npc.wz"
```
