# task-172: Legacy GMS + JMS WZ ingest support — Design

## Problem

atlas-data cannot ingest WZ data from pre-v83 GMS clients or from JMS. Feeding
GMS v48 through the web UI's "process data" flow silently produces zero
documents; GMS v12 (monolithic `Data.wz`) and JMS v185 (mixed per-image
encryption + `List.wz`) were expected to fail for structural reasons.

Goal: all three sets become first-class ingest inputs — upload the zip, hit
process, get correct documents — so live tenants can eventually run on them.
Scope is **ingest only**; packet/socket bring-up for these client versions is
separate follow-up work.

## Verified root causes

All findings below were empirically verified by running the real
`libs/atlas-wz` parser against sample data (GMS v12, GMS v48, JMS v185
archives; ~28k images total). None are speculation.

### RC-1: Encryption auto-detection picks the wrong key (breaks ALL three sets)

`wz.Open` → `detectVersion` (`libs/atlas-wz/wz/file.go`) brute-forces
(encryption, version) pairs and validates a candidate by trial-parsing the
first directory entry and checking the decoded *offset* lands in-file. But
offset decryption depends only on the version hash — **not on the AES key**.
The loop tries `EncryptionGMS` first, offsets validate, and the wrong key is
locked in. The sample archives are actually **unencrypted**
(`EncryptionNone`), so every directory/image name decodes to garbage.

Downstream, `wztoxml` serializes garbled filenames, the XML walk
(`walk.go`) matches nothing, and ingest "succeeds" with zero documents —
exactly the observed silent no-op.

Version detection itself is correct for all three sets: hash 1753 = v48,
1651 = v12, 53078 = v185.

With the correct key: **v48 parses 100 % clean** (8,062 images, 0 failures).
v48 needs *only* this fix.

### RC-2: Monolithic `Data.wz` not routable (GMS v12)

v12 ships one PKG1 archive whose root holds `Character/`, `Effect/`, `Etc/`,
`Item/`, `Map/`, `Mob/`, `Npc/`, `Reactor/`, `Skill/`, `Sound/`, `String/`,
`UI/` as subdirectories plus root-level `smap.img`/`zmap.img` (the Base.wz
role). All 3,613 images parse cleanly. The gap is purely routing: the worker
fan-out (`data/workers/registry.go`) fetches fixed per-archive object names
(`Item.wz`, `Mob.wz`, …) from MinIO via `fetchArchive`
(`workers/runtime.go:121`), so every fetch 404s and ingest aborts.

v12 also *lacks* whole categories (no Quest, Morph, TamingMob; no
`Item/Cash`), so workers must tolerate missing archives/directories.

### RC-3: JMS mixed per-image encryption driven by `List.wz`

JMS v185 archives are mostly unencrypted, but a subset of images is
AES-encrypted with the `B9 7D 63 E9` IV (the lib's `EncryptionKMS` variant).
`List.wz` is not PKG1 — it is a flat encrypted path list (1,218 entries,
decodable with the same KMS key: `[int32 len][len UTF-16 units XOR
keystream][u16 null]`). Its entry set matches the encrypted images **exactly**
(verified for `String.wz`: the 9 listed images are precisely the 9 that only
parse with the KMS key).

Mix ratios span the full range: `Mob.wz` is 100 % KMS-encrypted (1,738
images), `Effect.wz`/`UI.wz` 0 %, everything else in between. Per-image key
fallback parses **every JMS image with zero failures**.

### RC-4: Old String layout (v12 + v48, not JMS)

The String worker (`workers/stringw.go`) expects modern flat images
`Consume.img`, `Cash.img`, `Etc.img`, `Ins.img`, `Pet.img` + nested
`Eqp.img`. Old clients instead have a single `String/Item.img` whose
top-level children are `Cash`, `Con`, `Eqp`, `Etc`, `Ins`, `Pet` — same
`name`/`desc` leaves, and `Eqp` is nested by sub-category exactly like the
modern `Eqp.img`. Without an adapter, item-name registries and the item
search index come up empty. JMS v185 already uses the modern flat layout.

`Etc.wz/Commodity.img` exists in all three sets, so the Commodity worker is
unaffected.

## Design (approach A — approved)

Four components: two in `libs/atlas-wz`, two in atlas-data. No new config
surface, no new services, no changes to upload/process UX.

### C-1 (lib): two-phase version + key detection

Restructure `detectVersion` into:

1. **Version phase** (mechanics unchanged): brute-force version 1–1000
   validating first-entry offsets — key-independent, so run it with any
   single key instead of once per encryption type.
2. **Key phase**: with the version fixed, decode the first directory entry
   name(s) under each `EncryptionType` candidate and score for sanity —
   printable ASCII/BMP text, plausible entry name (`*.img` or a directory
   name). Select the unique sane candidate; if none or ambiguous, return a
   descriptive error (never silently pick one).

This preserves behavior for genuinely GMS-/KMS-encrypted archives (their
names only decode sanely under the right key) and fixes unencrypted archives
misdetected as GMS.

Sanity check operates on directory-entry names only (cheap, already parsed
during validation) — no full-tree scan at open time.

### C-2 (lib): per-image key fallback (JMS mixed encryption)

`Image.parse` currently decodes with the file-level key. Change: when the
image tag string fails validation (the existing "unexpected image tag …
(expected Property)" path), retry the parse with each other key variant; on
success, cache the winning key on the image and use it for all strings and
canvas-block decryption within that image. Emit a debug log on fallback.

- The retry triggers only on tag-validation failure, so verified-working
  archives see zero behavior change and no extra cost.
- `List.wz` is **not required**: fallback is empirically complete across all
  ~19k JMS images. `List.wz` remains unparsed and ignored by ingest (it is
  not in the worker registry).
- Concurrency: key cache assignment happens under the existing per-file
  `parseMu`, same as the rest of lazy parse state.

### C-3 (service): monolithic `Data.wz` virtual archives

`fetchArchive` (`workers/runtime.go`) is the single chokepoint every worker
uses. Extend it:

1. Try the per-archive object (`…/<Name>.wz`) as today.
2. On MinIO "not found", check the scope for `Data.wz` (fetched once and
   memoized via the existing `archiveCache`). If present, locate the root
   subdirectory matching the archive stem (`Item.wz` → `Item/`) and return a
   **sub-archive view**: a `*wz.File` veneer over that subdirectory that
   shares the parent's reader, key, and parse mutex. New lib constructor
   (e.g. `wz.NewSubFile(parent *File, root *Directory, name string)`) —
   `NewFileWithRoot` is close but lacks the backing reader needed for canvas
   reads.
3. `Base.wz` resolves to the `Data.wz` **root itself** (root-level
   `smap.img`/`zmap.img`), so the character worker's smap/zmap sidecars
   (`character.go:159`) work unchanged.
4. If neither the per-archive object nor a `Data.wz` subdirectory exists,
   the worker **logs and skips** instead of aborting the whole ingest run
   (v12 has no Quest/Morph/TamingMob). A missing *expected* archive in a
   split-layout upload still fails loudly as today — skip-tolerance applies
   only to registered-but-absent categories, reported per-worker in the job
   log.

Upload path needs no changes: `Data.wz` is a valid `.wz` zip entry and lands
in MinIO under the same scope key scheme.

### C-4 (service): String worker old-layout adapter

In `workers/stringw.go`: after serialization, if the flat item images
(`Consume.img.xml` etc.) are absent but `Item.img.xml` exists, feed the
existing registry initializers from `Item.img`'s subtrees instead:

| Old `Item.img` child | Modern equivalent | Reader |
|---|---|---|
| `Con` | `Consume.img` | flat (`InitStringFlat`) |
| `Cash` | `Cash.img` | flat |
| `Etc` | `Etc.img` | flat |
| `Ins` | `Ins.img` | flat |
| `Pet` | `Pet.img` | flat |
| `Eqp` | `Eqp.img` | nested by sub-category |

The initializers take a parsed subtree rather than a file path where needed
(small refactor); mapping is mechanical because leaf shape (`name`/`desc`)
is identical. Modern layout remains the primary path; the adapter engages
only when flat images are absent, so JMS and v83+ are untouched.

### C-5 (nice-to-have, small): declared-version cross-check

`wz.File` already computes the game version during detection; expose it
(`GameVersion()`) and have the ingest worker **warn** (not fail) when it
disagrees with the tenant-declared major version (`Params.MajorVersion`).
Catches uploads landing under the wrong tenant version — today this is
completely silent.

## Data flow (after change)

Unchanged end-to-end: upload zip → MinIO → `POST /data/process` → k8s Job
(`MODE=ingest`) → `RunWorkers` → String prerequisite → parallel workers →
`wztoxml` → domain readers → `documents` + search index + MinIO assets.
Every change is inside `wz.Open`/`Image.parse` (lib) and
`fetchArchive`/`stringw.go` (service).

## Error handling

- Detection ambiguity (key phase): hard error naming the candidates tried —
  never a silent guess.
- Per-image fallback exhausted: existing per-image error path (warn + error
  from `Properties()`, image skipped by serializer) — unchanged semantics.
- Missing category in monolithic layout: per-worker log + skip; job
  succeeds; skips enumerated in job output.
- Version mismatch (C-5): warn only.

## Testing

- **Lib unit tests (binary fixtures)**: add a minimal test-only WZ *writer*
  (test package helper) that emits tiny PKG1 archives — enough for a
  directory + a few string/int properties. Fixtures: (a) unencrypted, (b)
  GMS-encrypted, (c) KMS-encrypted, (d) mixed per-image encryption, (e)
  monolithic root with category subdirs. Tests assert: correct
  (version, key) detection for a–c; per-image fallback for d; `NewSubFile`
  parse + canvas read for e. No real game archives in the repo.
- **Service tests**: `fetchArchive` monolithic resolution (per-archive
  object absent, `Data.wz` present) against a fixture archive via the
  existing MinIO test seam; String adapter mapping old `Item.img` subtrees
  into registries; missing-category skip path.
- **E2E verification (manual, documented in the task folder)**: run ingest
  against the three real sample sets (out-of-repo, `/tmp/wz`) and check
  document counts per type are nonzero and sane for each set. This also
  flushes the residual risk below.
- Existing regression: full `go test -race` on atlas-wz + atlas-data;
  current GMS v83/v95 ingest fixtures must stay green (proves detection
  restructure didn't disturb the supported path).

## Residual risk / explicitly out of scope

- **Domain-reader schema drift inside old `.img` data** (e.g. v12-era skill
  or map fields that domain readers expect but old data lacks): binary
  parsing is proven clean, but semantic completeness of extracted documents
  for v12/v48 is only provable by the E2E ingest run. Structural fallout
  from that run is handled iteratively inside this task; anything that
  turns out to be a *feature-sized* semantic gap (e.g. a document type old
  clients simply don't have) is documented in the task folder rather than
  silently stubbed.
- Packet/socket bring-up for v12/v48/JMS clients — separate work.
- `List.wz`-driven exact decryption — deliberately not needed (see C-2).
- Sound.wz ingestion — no worker exists today for any version; unchanged.

## Alternatives considered

- **B: config-declared encryption per region/version + List.wz-driven
  decryption**: rejected — region↔key mapping is factually wrong for these
  samples ("GMS" archives are unencrypted), adds config surface and an
  upload dependency, and the heuristic approach is empirically complete.
- **C: offline pre-conversion tool**: rejected — manual out-of-band step,
  splits provenance from the upload→process UX, and still needs the same
  format knowledge.
