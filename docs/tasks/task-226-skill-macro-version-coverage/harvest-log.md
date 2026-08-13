# Task 2 harvest log — skill-macro send/receive functions

Scope: splice `CMacroSysMan::FlushToSvr` (SKILL_MACRO, serverbound) and
`CWvsContext::OnMacroSysDataInit` (MACRO_SYS_DATA_INIT, clientbound) into the
ten committed IDA exports under `docs/packets/ida-exports/`, with real
(non-null) `calls`.

## IDA sessions (`idb_list`, all ten in-scope IDBs loaded)

| version | binary (filename) | session id used for `-ida-database` |
|---|---|---|
| gms_v48 | GMS_v48_1_DEVM.exe | 93cc947e |
| gms_v61 | GMS_v61.1_U_DEVM.exe | 415bf585 |
| gms_v72 | GMS_v72.1_U_DEVM.exe | c8acae95 |
| gms_v79 | GMS_v79_1_DEVM.exe | 1438cecd |
| gms_v83 | MapleStory_dump.exe | 41f13e0d |
| gms_v84 | GMS_v84.1_U_DEVM | 5881cf84 |
| gms_v87 | GMSv87_4GB.exe | d51ecbd3 |
| gms_v92 | GMS_v92_1_DEVM.exe | acdfccff |
| gms_v95 | GMS_v95.0_U_DEVM.exe | 79906a1e |
| jms_v185 | MapleStory_dump_SCY.exe | b6864e54 |

The live IDA-MCP HTTP endpoint used by `packet-audit export` was
`http://192.168.20.3:8745/mcp` (the session-based server; the CLI's baked-in
default `http://192.168.20.3:13337/mcp` is the old dead per-port server —
never used). `-ida-database <session id>` was passed on every `export`
invocation; sessions were re-verified via a fresh `idb_list` immediately
before harvesting (ids were stable across the whole task).

## Per-version results

| version | FlushToSvr addr | opcode read | registry opcode | match | OnMacroSysDataInit addr | named/renamed |
|---|---|---|---|---|---|---|
| gms_v48 | NOT FOUND | n/a | not registered | n/a | NOT FOUND | n/a |
| gms_v61 | NOT FOUND | n/a | not registered | n/a | 0x849bce | already named |
| gms_v72 | 0x5e39e0 | 109 (0x6D) | 109 | yes | 0x92126b | already named |
| gms_v79 | 0x6022db | 108 (0x6C) | 108 | yes | 0x97311a | already named |
| gms_v83 | 0x631919 | 110 (0x6E) | 110 | yes | 0xa290f8 | already named |
| gms_v84 | 0x646e7f | 110 (0x6E) | 110 | yes | 0xa748bb | already named |
| gms_v87 | 0x66a505 | 113 (0x71) | 113 | yes | 0xac0d6e | already named |
| gms_v92 | 0x602ed0 | 121 (0x79) | 121 | yes | 0x9c5390 | already named |
| gms_v95 | 0x60ea20 | 122 (0x7A) | 122 | yes | 0x9f0c70 | already named |
| jms_v185 | 0x6a3466 | 105 (0x69) | 105 | yes | 0xb10384 | already named |

"opcode read" is the literal integer argument in
`COutPacket::COutPacket(&pkt, OPCODE)` inside `CMacroSysMan::FlushToSvr`,
read from a live decompile of the send function itself (not trusted from any
symbol name) — see "Opcode confirmation" below. Every non-NOT-FOUND version
agrees with the registry's SKILL_MACRO opcode from Global Constraints.

### FlushToSvr: named vs renamed

`CMacroSysMan::FlushToSvr` was already correctly named in the IDB for
gms_v83, gms_v87, gms_v95, jms_v185 (matches design's prior "8 versions
already name it" note — those 4 plus v84/v92 below, once corrected). It was
**unnamed** (`sub_XXXXXX`) in gms_v72, gms_v79, gms_v84, gms_v92 — located via
the byte signature `6A <op> 8D` (a shorter match than
`6A <op> 8D 8D ?? ?? ?? ??` from VERIFYING_A_PACKET.md §10 — the actual
compiled `lea ecx, [ebp+var]` in this codebase's release builds encodes as
`8D 4D <disp8>` or `8D 8D <disp32>` depending on frame offset size, so the
literal `??` count in §10's signature does not universally apply; matching
only through the `8D` opcode byte and then decompiling to confirm was more
robust), confirmed by decompile (COutPacket ctor opcode integer matches
registry), then renamed via `mcp__ida-pro__rename` to
`?FlushToSvr@CMacroSysMan@@QAEXXZ` and saved via `idb_save`:

- gms_v72: `sub_5E39E0` → renamed (address unchanged, 0x5e39e0)
- gms_v79: `sub_6022DB` → renamed (address unchanged, 0x6022db)
- gms_v84: `sub_646E7F` → renamed (address unchanged, 0x646e7f)
- gms_v92: `sub_602ED0` → renamed (address unchanged, 0x602ed0)

### OnMacroSysDataInit: named vs renamed

Already named in all nine versions where it exists (gms_v48 excluded — not
found at all). No renames were needed for the two top-level target functions
on the clientbound side; all rename work on that side went into the helper
chain (below).

## Registry fname bug found (confirmed, NOT fixed here — Task 5's job)

`docs/packets/registry/gms_v72.yaml` and `docs/packets/registry/gms_v79.yaml`
both currently carry `fname: sub_6022DB` for `SKILL_MACRO`. Live decompile
confirms:

- `sub_6022DB` (address `0x6022db`) is v79's own, CORRECT
  `CMacroSysMan::FlushToSvr` — `push 6Ch` (108) matches v79's registered
  opcode exactly.
- v72's real `CMacroSysMan::FlushToSvr` is at a DIFFERENT address, `0x5e39e0`
  — `push 6Dh` (109) matches v72's registered opcode. The registry's v72
  entry's `ida.address: 6175200` (= `0x5e39e0` in decimal) is actually
  CORRECT; only the `fname` string is wrong (it was copied from v79's
  `sub_6022DB` symbol instead of pointing at v72's own `sub_5E39E0`).

Both addresses are now real, resolved, real-`calls` entries in the spliced
exports under their own correct keys (v72 export key uses v72's own address
0x5e39e0; v79 export key uses v79's own address 0x6022db) — the registry
`fname` correction itself is left to Task 5 per the plan's mutual-exclusivity
of these two tasks.

## v48 / v61: absence, searched not assumed

**gms_v48.** Binary-wide search extent:

- `func_query name_regex "MacroSysMan|OnMacroSysDataInit|FlushToSvr"` → 0
  hits (whole binary, no `count` cap).
- `find_regex "macro"` (whole binary, case-insensitive string search) → 0
  hits.
- No `MACROSYSDATA`, `SINGLEMACRO`, or `SetMacro` symbols anywhere in the
  binary.
- Registry: neither `SKILL_MACRO` nor `MACRO_SYS_DATA_INIT` is registered for
  gms_v48.

Both functions are recorded `NOT FOUND — see Task 4` (export already carries
an `unresolved: true` stub for both from a prior harvest attempt; this task
made no change to `gms_v48.json`, confirmed by `git diff --stat` below).

**gms_v61.** `CWvsContext::OnMacroSysDataInit` exists and is named
(`0x849bce`); its full clientbound chain (→ `CMacroSysMan::SetMacro`
`0x59744b` → `MACROSYSDATA::Decode` `0x4b796d` → `SINGLEMACRO::Decode`
`0x4b78d5`) was locatable, unnamed, and is now fully real (see below) — this
task corrects the earlier framing that v61's receive side was a dead end; it
was just unnamed.

`CMacroSysMan::FlushToSvr` (the SEND side) was NOT located, matching the
pre-existing finding in `docs/packets/registry/discover_gms_v61.md` ("v61
SKILL_MACRO send-site not located... likely client-only or inlined"). This
task's own search: whole-binary `func_query name_regex` for
`MacroSysMan|SetMacro|MACROSYSDATA|SINGLEMACRO` returns exactly the four
symbols in the confirmed clientbound chain and nothing else — no unnamed
`CMacroSysMan`-class candidate remains reachable from `OnMacroSysDataInit`'s
2-level call graph, and there is no known opcode to byte-signature-search
for (v61 has no `SKILL_MACRO` registry entry to confirm against). Recorded
`NOT FOUND — see Task 4`, consistent with the prior audit.

## Tool limitation found: constructed-then-delegated Send/Receive functions parse as `calls: null`

`packet-audit export`'s regex parser (`ParseDecompile` /
`idasrc/parse.go`) only seeds its packet-alias tracking set from a **literal**
`COutPacket::EncodeN(...)` / `CInPacket::DecodeN(...)` call **inside the same
function body**. Both target functions across every version construct the
packet (or receive the `CInPacket&` parameter) and then hand the ENTIRE
encode/decode off to a single helper call — `CMacroSysMan::FlushToSvr` calls
`MACROSYSDATA::Encode(this+8, pkt)` with no direct `Encode*` call of its own;
`CWvsContext::OnMacroSysDataInit` calls `CMacroSysMan::SetMacro(instance,
pkt)` (or, in gms_v92 only, `MACROSYSDATA::Decode(pkt)` directly) with no
direct `Decode*` call of its own. Because the alias-seed regex never fires,
the harvester emits `calls: null` for these two functions in EVERY version,
even after the correct address is resolved.

This is a real, verified-by-decompile call relationship, not a fabrication —
each Delegate below was confirmed by reading the actual decompiled body. Per
VERIFYING_A_PACKET.md §10 ("surgically splice ONLY the needed entries...
hand-edit the target export entries") this task hand-authored a single
`Delegate` call for `FlushToSvr` / `OnMacroSysDataInit` (and, where present,
the intermediate `CMacroSysMan::SetMacro`) pointing at the real helper, and
then harvested that helper (`MACROSYSDATA::Encode` / `MACROSYSDATA::Decode`)
through the normal automated path — which DOES parse correctly, because
those helper functions DO call `COutPacket::Encode1` / `CInPacket::Decode1`
directly, and their own further delegation to `SINGLEMACRO::Encode` /
`SINGLEMACRO::Decode` is picked up automatically by the harvester's
Delegate-descent (`descent-depth 12`).

Several `MACROSYSDATA`/`SINGLEMACRO`/`CMacroSysMan::SetMacro` helper
functions were unnamed in the target IDB and were renamed (via `rename` +
`idb_save`) to their canonical mangled forms so the harvester could resolve
them by name and so the export key demangles cleanly:

| version | MACROSYSDATA::Encode | MACROSYSDATA::Decode | CMacroSysMan::SetMacro |
|---|---|---|---|
| gms_v61 | n/a (FlushToSvr not found) | renamed 0x4b796d | renamed 0x59744b |
| gms_v72 | renamed 0x4d366f | renamed 0x4d36b4 | renamed 0x5e39bf |
| gms_v79 | renamed 0x4db93f | renamed 0x4db984 | renamed 0x6022ba |
| gms_v83 | already named 0x4e776b | already named 0x4e77b0 | already named 0x6318f8 |
| gms_v84 | renamed 0x4efc2b | renamed 0x4efc70 | renamed 0x646e5e |
| gms_v87 | renamed 0x50854a | already named 0x50858f | already named 0x66a4e4 |
| gms_v92 | already named 0x4f4c00 | already named 0x4f4c50 | n/a (v92 has no separate SetMacro — OnMacroSysDataInit calls MACROSYSDATA::Decode directly) |
| gms_v95 | already named 0x4f9860 | already named 0x4f98b0 | already named 0x60e580 |
| jms_v185 | renamed 0x515451 | already named 0x515496 | already named 0x6a3445 |

The `SINGLEMACRO::Encode`/`SINGLEMACRO::Decode` per-slot helpers one level
further down were also renamed where unnamed. In gms_v72, gms_v79, gms_v84,
gms_v87 (encode side only), and jms_v185 (encode side only), the renamed
function's decompile still displays as `sub_XXXXXX` instead of the demangled
form — a Hex-Rays quirk where the inferred return type (often `int`, since no
type info was ever set) does not match the mangled signature's declared
`void` return, so the demangler falls back to the raw symbol for display
purposes. This is COSMETIC ONLY: `idb_list`/`func_query` confirm the rename
took (`?Encode@SINGLEMACRO@@...` / `?Decode@SINGLEMACRO@@...` is the real,
persisted name), and the harvester's `GetFunctionByName` resolves it
correctly regardless — the export simply keys these entries under whatever
name the decompile text displayed for the delegate call site (e.g.
`sub_4D35CB` for gms_v72's encode-side `SINGLEMACRO::Encode`), which is the
same convention the existing exports already use for similar cases.

## gms_v92: two additional hand-corrections (auto-harvest gaps, verified by decompile)

gms_v92's Hex-Rays output casts the packet-pointer argument at both call
sites that matter here (`COutPacket::Encode1((unsigned int *)a2, v4)` in
`MACROSYSDATA::Encode`; `v2 = (volatile LONG *)a2;` then
`SINGLEMACRO::Decode(ptr, v2)` in `MACROSYSDATA::Decode`) — the alias
tracker's regexes require a bare identifier immediately after `(` / `= &?`,
so a cast defeats them:

1. **`MACROSYSDATA::Encode`** (0x4f4c00): the harvester correctly recorded
   the `Decode1`-equivalent primitive but MISSED the delegate to
   `SINGLEMACRO::Encode` (0x4f4ab0) entirely (`calls` had 1 entry, not 2).
   Hand-added the missing `Delegate` after confirming the call by decompile.
2. **`MACROSYSDATA::Decode`** (0x4f4c50): the harvester recorded a **WRONG**
   `Delegate` — it followed `&a2` (address-of, taken for the buffer-resize
   call) to `sub_4ef7f0`, which decompiles as `ZArray<SINGLEMACRO>::_Realloc`
   (a pure buffer-management helper, never touches the wire) — and MISSED the
   real wire-reading callee `SINGLEMACRO::Decode` (0x4f4b90), because that
   call uses the aliased-through-a-cast `v2`, not the seeded `a2`. Corrected:
   the bogus `_Realloc` delegate was dropped (same precedent as VERIFYING
   §10's "COutPacket-delegate harvest artifact" — a harvested call that isn't
   actually a wire read gets stripped) and replaced with the real
   `SINGLEMACRO::Decode` delegate, confirmed by decompile.
3. **`SINGLEMACRO::Encode`** (0x4f4ab0) itself also came back `calls: null`
   from the automated harvester (harvested separately via a follow-up `export`
   call) — its 3× `uint32` skill-id writes are emitted by Hex-Rays as an
   inlined manual pointer-store loop (buffer growth + `*(_DWORD*)(...) = v`),
   never as a textual `COutPacket::Encode4(...)` call, so the parser has
   nothing to match. Hand-authored as `[DecodeStr, Decode1, Decode4]` — the
   same canonical 3-op shape every sibling version's `SINGLEMACRO::Encode` /
   `SINGLEMACRO::Decode` already has (op names are direction-agnostic
   `DecodeN` canonical strings per `idasrc/parse.go`'s `opName()`; this
   matches the 3-field `[nSkillId0, nSkillId1, nSkillId2]`-shaped struct seen
   identically in every other version's real, auto-harvested entry) —
   verified against `SINGLEMACRO::Decode`'s real, auto-harvested sibling
   entry (0x4f4b90: `DecodeStr, Decode1, Decode4`) for the same version.

## Harvest commands (representative — one IDB at a time, `-ida-database <session>`)

```
go run ./tools/packet-audit export \
  -version gms_v83 \
  -prior-export "<dir-stub with FlushToSvr=serverbound, OnMacroSysDataInit=clientbound, MACROSYSDATA::Encode=serverbound, MACROSYSDATA::Decode=clientbound, CMacroSysMan::SetMacro=clientbound>" \
  -pending "<roster: CMacroSysMan::FlushToSvr, CWvsContext::OnMacroSysDataInit, MACROSYSDATA::Encode, CMacroSysMan::SetMacro>" \
  -descent-depth 12 \
  -ida-url http://192.168.20.3:8745/mcp \
  -ida-database 41f13e0d \
  -output /tmp/harvest-gms_v83.json
# → export: 5 resolved, 2 descended-helper, 0 unresolved
```

The `-prior-export` direction stub was required: with `-prior-export ""` (as
literally written in the plan's Step 3), the harvester has no direction
signal for a brand-new fname (no registry `candidatesFromFName` case exists
for these two functions), and the exporter's direction-fallback defaults to
`clientbound` — which for `CMacroSysMan::FlushToSvr` (a SEND function) makes
the parser scan for `CInPacket::DecodeN` instead of `COutPacket::EncodeN` and
silently find nothing. Passing a tiny `{"functions": {"<name>": {"direction":
"..."}}}` stub as `-prior-export` fixes the direction without re-harvesting
anything real (`priorDirections()` only reads the `direction` field). This
stub also unions into the roster (`buildRoster` treats `-prior-export`'s keys
as roster members too), which is why some per-version unresolved-noise
entries (`CMacroSysMan::SetMacro` reported unresolved for gms_v92, where it
does not exist) appear in the raw harvest output — harmless, and not spliced
into the committed export.

Each version was harvested one IDB at a time (`-ida-database` targeted the
single live session for that version), never in parallel.

## Splice method

Followed VERIFYING_A_PACKET.md §10 exactly: no `packet-audit export` was ever
run against a committed `--output` path directly, and `-splice` was
deliberately NOT used (`bug_packet_audit_export_splice_drops_fields`: it
round-trips the whole file through a Go struct that drops unrecognized JSON
fields like `region`/`note` and re-sorts/re-indents everything, corrupting
~20 unrelated entries per invocation). Instead: harvested to
`/tmp/claude-.../harvest-<version>.json`, then spliced by hand — literally,
with a small Python helper that (a) parses the committed file only to know
which keys already exist, (b) for an EXISTING key (only
`CWvsContext::OnMacroSysDataInit` in gms_v61/v72/v79, which held a
`calls: null` stub) locates that exact key's `{...}` span by brace-depth
scanning and replaces ONLY that span's text, and (c) for NEW keys, inserts
new entries as plain text immediately before the file's closing
`\n  }\n}\n`, comma-separated. The rendered JSON text is byte-for-byte
hand-matched to the file's existing formatting convention (2-space nesting,
`ensure_ascii`-style `\uXXXX` escaping for any non-ASCII, `address` /
`direction` / `calls` field order, `op` / `comment` / `guard` / `ref` call
order) — verified by round-tripping a real existing entry through the same
renderer and diffing byte-for-byte against the file (0 differences) before
touching any committed file.

An earlier attempt that re-serialized the WHOLE committed file through
`json.dump(..., ensure_ascii=True)` was caught and reverted before committing
anything: several pre-existing comments in `gms_v72.json` (and presumably
others) store literal UTF-8 em-dashes rather than `—` escapes, so a
uniform re-dump re-escaped ~5 unrelated lines per file — exactly the drift
this playbook step exists to catch. `git diff --stat` after the fix confirms
only the intended keys changed (see below).

## Verification

```
$ git diff --stat docs/packets/ida-exports/
 docs/packets/ida-exports/gms_jms_185.json | 101 ++++++++++++++++++++++++++++++
 docs/packets/ida-exports/gms_v61.json     |  55 +++++++++++++++-
 docs/packets/ida-exports/gms_v72.json     | 100 ++++++++++++++++++++++++++++-
 docs/packets/ida-exports/gms_v79.json     | 100 ++++++++++++++++++++++++++++-
 docs/packets/ida-exports/gms_v83.json     | 101 ++++++++++++++++++++++++++++++
 docs/packets/ida-exports/gms_v84.json     | 101 ++++++++++++++++++++++++++++++
 docs/packets/ida-exports/gms_v87.json     | 101 ++++++++++++++++++++++++++++++
 docs/packets/ida-exports/gms_v92.json     |  91 +++++++++++++++++++++++++++
 docs/packets/ida-exports/gms_v95.json     |  97 ++++++++++++++++++++++++++++
 9 files changed, 841 insertions(+), 6 deletions(-)
```

`gms_v48.json` shows NO diff (correctly untouched — both functions remain
absent/unresolved there). Per-file changed-key audit (`git diff <file> | grep
'^[+-]    "'`, deduped) confirms every touched key is one of: the two target
fnames, the intermediate `CMacroSysMan::SetMacro`, the two `MACROSYSDATA::*`
helpers, and the `SINGLEMACRO::*` (or `sub_XXXXXX` fallback-named)
grandchild helpers — no other function key appears in any diff.

```
$ go run ./tools/packet-audit matrix --check
matrix --check: docs/packets/audits/STATUS.md is stale — regenerate and commit
matrix --check: docs/packets/audits/status.json is stale
exit status 1
```

Expected — the export content hashes embedded in `status.json`/`STATUS.md`
changed along with the nine export files. Regenerated (`go run
./tools/packet-audit matrix`, no `--check`), diff confirmed minimal (only the
9 changed `exportHashes` values; `gms_v48`'s hash is unchanged; no cell
verdicts/rows changed — no fixtures or registry fnames were touched by this
task, so no cell could promote yet), then re-ran:

```
$ go run ./tools/packet-audit matrix --check
exit status 0
```

Clean. `docs/packets/audits/STATUS.md` and `docs/packets/audits/status.json`
are committed alongside the nine export files and this log.

## Judgment calls

1. **`-prior-export ""` (literal empty, per the plan's Step 3 command) does
   not work as written** — it defaults direction to clientbound for a
   brand-new fname, silently producing `calls: null` for the serverbound
   `FlushToSvr` even once its address resolves. Used a tiny direction-stub
   JSON as `-prior-export` instead (documented above), consistent with the
   plan's *intent* (a targeted, non-destructive harvest) while fixing the
   direction defect the plan's literal command triggers.
2. **Both target functions parse as `calls: null` from the automated
   harvester in every single version**, not because they're unresolvable but
   because of a structural blind spot in `ParseDecompile`'s alias-seeding
   (documented above in detail). Rather than accept `calls: null` (which
   fails the task's own "real (non-null) calls" requirement) or hack the Go
   tool (out of scope for a harvest task), hand-spliced a single verified
   `Delegate` call per function, and separately, automatically harvested the
   real helper it delegates to. This is the same "surgical hand-splice"
   mechanism VERIFYING_A_PACKET.md §10 already prescribes for the
   `COutPacket`-delegate-artifact case, applied to a symmetrical new case.
3. **v72/v79 registry `fname: sub_6022DB` bug**: confirmed live (v79's own
   address, wrongly reused for v72's registry entry) — recorded above with
   exact addresses/opcodes for Task 5 to consume; NOT edited in the registry
   YAML (out of this task's scope per the plan).
4. **v92 has no `CMacroSysMan::SetMacro`** at all (inlined away relative to
   every other version) — `OnMacroSysDataInit`'s hand-spliced Delegate points
   directly at `MACROSYSDATA::Decode` for this one version only.
5. **v92's two auto-harvest defects** (missed delegate in `Encode`; wrong
   delegate in `Decode`) were corrected by hand after decompile verification,
   not left as `calls: null` or accepted as spuriously wrong — see the
   dedicated section above.
6. **v61's clientbound (receive) chain is fully real**, correcting the
   framing that v61 has zero recoverable content here; only the serverbound
   (send) side remains genuinely not-found, matching the pre-existing
   `discover_gms_v61.md` finding.
7. **v48: no change made.** Both functions remain `unresolved: true` exactly
   as before this task — confirmed by binary-wide search (0 name-regex hits,
   0 string hits, no registry entries), consistent with the design's
   framing that this is Task 4's determination to make, not this task's.
8. **`sub_XXXXXX`-keyed helper entries** (six across gms_v72/v79/v84, one
   each in gms_v87/jms_v185/gms_v61) were left under their decompile-displayed
   raw name rather than forced to redisplay as `SINGLEMACRO::Encode`/`Decode`
   — the underlying IDB rename is real and persisted (confirmed via
   `func_query`), only Hex-Rays' OWN pseudocode rendering falls back to
   `sub_` for these specific functions (return-type/signature mismatch); the
   export entry key must match whatever the decompiled Delegate call site
   literally displays, or the reference chain breaks. This mirrors the
   existing convention already used elsewhere in these same export files for
   similar unresolved-typeinfo cases.
