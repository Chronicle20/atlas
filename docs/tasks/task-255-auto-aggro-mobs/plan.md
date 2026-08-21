# Auto-Aggro / First-Attack Mobs — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decode `AUTO_AGGRO` (`CMob::ApplyControl`) on all ten client versions and turn it into a validated, server-authoritative aggro grant so mobs whose WZ template carries `firstAttack` attack players unprovoked.

**Architecture:** Packet-in / Kafka-across / packet-out. `atlas-channel` decodes `AUTO_AGGRO{mobId, distance}`, applies cheap admission checks plus a per-(character, mob) rate gate, and emits a new `SET_AGGRO` command on `COMMAND_TOPIC_MONSTER`. `atlas-monsters` owns every authoritative gate (exists / alive / aggressive template / in field) and arbitrates: the current controller flips the flag in place; a non-controller takes control **with** aggro via the existing `startControl(..., forceAggro=true)` unless someone already holds aggro. An aggro **lease** (`aggroRefreshedMs` on the registry record + a new branch in `MonsterAggroDecayTask`) releases an auto-aggro'd, damage-entry-free mob after 15s. The control-packet re-issue on `AGGRO_CHANGED` and the aggro-through-handover behaviour already ship — those tasks add tests only.

**Tech Stack:** Go (multi-module monorepo), `libs/atlas-packet` codecs, Kafka (`segmentio/kafka-go`), Redis-backed tenant registry, `tools/packet-audit`, ida-pro-mcp for the export splice.

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md))

## Global Constraints

- **Never invent a value, name, opcode, output, or behavior.** Every byte and every opcode in this plan traces to an IDA address recorded in design §1.2.
- **No stubs.** No `// TODO`, no decode-and-log handler, no unimplemented status response. FR-3.5 is explicit: the handler must act.
- **Version gates use `MajorAtLeast(N)`**, never raw `> N`. This codec has **no** version divergence (design §1.2), so it takes no gate at all.
- **No wire change to an already-verified packet on any version** (FR-2.4).
- Before defining a new domain type, alias, or numeric constant, check `libs/atlas-constants/` for an existing equivalent. The three tuning constants introduced here (`AutoAggroProximityThreshold`, `AutoAggroRefreshInterval`, `AutoAggroLeaseTtl`) are behavioural dials with no `atlas-constants` equivalent and live beside the code that reads them.
- Seed-template `socket.handlers` entries **must** carry `"validator": "LoggedInValidator"` — a missing/unknown validator is silently dropped by `BuildHandlerMap` and the handler never registers.
- Template `handlers`/`writers` entries go at their **ascending `opCode` position**; `tools/template-opcode-order-guard.sh` enforces it.
- `packet-audit` regeneration is machine-graded: a cell that does not promote is a failure, not a prose claim.
- Every plan `go build ./... && go test ./...` runs from the module root named in the task's Files block. Implementers run **module-local** builds/tests only — repo-wide `tools/verify.sh` belongs to `atlas-verifier`.

### Per-version facts used throughout (design §1.2 — do not re-derive)

| Version | IDB session (`idb_list`) | `CMob::ApplyControl` | Opcode dec | Opcode hex | Registry today |
|---|---|---|---|---|---|
| gms_v48 | `12a398ce` | `0x551c79` | 130 | `0x82` | **entry absent** |
| gms_v61 | `921fdbb5` | `0x5ccf1c` | 156 | `0x9C` | **entry absent** |
| gms_v72 | `99e435d8` | `0x61d358` | 179 | `0xB3` | **entry absent** |
| gms_v79 | `5a1cd4f3` | `0x63d0e6` | 181 | `0xB5` | **entry absent** |
| gms_v83 | `754107bf` | `0x66e146` | 189 | `0xBD` | 189, `csv-import` |
| gms_v84 | `46c2a2eb` | `0x684492` | 194 | `0xC2` | **189 — wrong** |
| gms_v87 | `c0829805` | `0x6a9061` | 201 | `0xC9` | 201, `csv-import` |
| gms_v92 | `019cd393` | `0x636320` | 221 | `0xDD` | 221, `csv-import` |
| gms_v95 | `ecc757f4` | `0x640d20` | 228 | `0xE4` | 228, `csv-import` |
| jms_v185 | `a977912e` | `0x6eba3c` | 195 | `0xC3` | 195, `csv-import` |

Session ids were confirmed live during planning; re-run `mcp__ida-pro__idb_list` and match by **binary filename** before use — ids are not stable across server restarts.

Derived identity of the new packet (from `qualifiedWriterName(pkg, name)` = TitleCase(pkg)+name):

- struct `AutoAggro` in package `monster`, dir `serverbound`
- packet id / marker path: `monster/serverbound/MonsterAutoAggro`
- audit report filename: `MonsterAutoAggro.{json,md}`
- evidence record: `docs/packets/evidence/<version>/monster.serverbound.MonsterAutoAggro.yaml`
- `monster/` is a tier-1 `packet_prefixes` entry (`docs/packets/evidence/tiers.yaml:55`), so **every** cell needs an evidence pin.

---

## Task 1: `AutoAggro` codec + report linkage

### Files

- `libs/atlas-packet/monster/serverbound/auto_aggro.go` — **new file**; the codec
- `libs/atlas-packet/monster/serverbound/auto_aggro_test.go` — **new file**; golden bytes + round-trip + the ten `packet-audit:verify` markers
- `libs/atlas-packet/monster/serverbound/mob_drop_pickup_request.go` — read-only; the shape to copy (immutable struct, `Operation`, `String`, `Encode`, `Decode`)
- `libs/atlas-packet/monster/serverbound/mob_drop_pickup_request_test.go` — read-only; the marker + golden-byte test shape to copy
- `tools/packet-audit/cmd/run.go:1198` — add the `case "CMob::ApplyControl":` arm to `candidatesFromFName`, immediately after the `CMob::SendDropPickUpRequest` arm

Module roots: `libs/atlas-packet` (build/test), `tools/packet-audit` (build only).

Patterns to copy: `libs/atlas-packet/monster/serverbound/mob_drop_pickup_request.go:1` (identical two-`Encode4` shape).

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/monster/serverbound/auto_aggro_test.go`. Package `serverbound`; imports `bytes`, `testing`, and `pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"`.

One test function, `TestAutoAggro`, carrying **all ten** markers above it (exact text, one per line):

```
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v48 ida=0x551c79
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v61 ida=0x5ccf1c
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v72 ida=0x61d358
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v79 ida=0x63d0e6
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v83 ida=0x66e146
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v84 ida=0x684492
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v87 ida=0x6a9061
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v92 ida=0x636320
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v95 ida=0x640d20
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=jms_v185 ida=0x6eba3c
```

Test body:

```go
input := NewAutoAggro(0xAABBCCDD, 0x00000027)

// Golden bytes (v83 baseline). CMob::ApplyControl @0x66e146:
//   Encode4(_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS))  -> mobId  uint32 LE
//   Encode4(n = |dx|/10 + |dy|/3 [+100])              -> distance uint32 LE
got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
want := []byte{
    0xDD, 0xCC, 0xBB, 0xAA, // mobId    uint32 LE = 0xAABBCCDD (Encode4 @0x66e146)
    0x27, 0x00, 0x00, 0x00, // distance uint32 LE = 39         (Encode4 @0x66e146)
}
```

`t.Fatalf` on mismatch with `"AutoAggro layout mismatch\n got % x\nwant % x"`.

Then a `for _, v := range pt.Variants` loop with `t.Run(v.Name, ...)` calling
`pt.RoundTrip(t, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion), input.Encode, input.Decode, nil)` —
copy that loop verbatim from `mob_drop_pickup_request_test.go:37`.

Add three accessor assertions after the golden check (no subtests needed):

| assertion | expected |
|---|---|
| `input.MobId()` | `0xAABBCCDD` |
| `input.Distance()` | `0x27` (39) |
| `input.Operation()` | `AutoAggroHandle` (`"AutoAggro"`) |

- [ ] **Step 2: Run the test to verify it fails**

```
cd libs/atlas-packet && go test ./monster/serverbound/ -run TestAutoAggro
```

Expected: FAIL — `undefined: NewAutoAggro`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/monster/serverbound/auto_aggro.go`, package `serverbound`. Mirror `mob_drop_pickup_request.go` field-for-field in shape:

```go
const AutoAggroHandle = "AutoAggro"

// AutoAggro is the serverbound AUTO_AGGRO packet (CMob::ApplyControl): the
// client asks the server to make it the mob's aggro'd controller.
//
// CMob::Update calls ApplyControl unconditionally; it sends at most once per
// second, for any mob whose template carries bFirstAttack OR bPickUpDrop, from
// ANY client that can see the mob — controller or not. bPickUpDrop alone is
// enough to fire it, so the aggressive-template check is server-side
// (atlas-monsters SetAggro gate 3), never inferred from the packet's presence.
//
// Byte layout (IDA-verified, IDENTICAL across all ten versions — two Encode4;
// no version gate):
//   - mobId    : uint32 — CMob::m_dwMobID. The send site encodes
//     _ZtlSecureFuse(m_dwMobID, m_dwMobID_CS); fuse RECOVERS the plaintext
//     object id, so the wire carries the mob object id verbatim. The sibling
//     MobDropPickupRequest names the same value `mobCrc`; that name is a
//     misnomer inherited from the send site and is not propagated here.
//   - distance : uint32 — the client's own proximity score,
//     |dx|/10 + |dy|/3, +100 when nMoveAction & 0xFFFFFFFE == 0x12.
//     CMob::TryFirstAttack chases at score <= 40 (v95 @0x6482f0); the channel
//     admission gate adopts that bar.
//
// IDA basis: CMob::ApplyControl — v48 @0x551c79, v61 @0x5ccf1c, v72 @0x61d358,
// v79 @0x63d0e6, v83 @0x66e146, v84 @0x684492, v87 @0x6a9061, v92 @0x636320,
// v95 @0x640d20, jms_v185 @0x6eba3c.
//
// packet-audit:fname CMob::ApplyControl
type AutoAggro struct {
    mobId    uint32
    distance uint32
}

func NewAutoAggro(mobId uint32, distance uint32) AutoAggro {
    return AutoAggro{mobId: mobId, distance: distance}
}

func (m AutoAggro) MobId() uint32     { return m.mobId }
func (m AutoAggro) Distance() uint32  { return m.distance }
func (m AutoAggro) Operation() string { return AutoAggroHandle }
func (m AutoAggro) String() string {
    return fmt.Sprintf("mobId [%d], distance [%d]", m.mobId, m.distance)
}
```

`Encode` and `Decode` copy `MobDropPickupRequest`'s signatures exactly (`func (m AutoAggro) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte` and `func (m *AutoAggro) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{})`), writing/reading `mobId` then `distance` with `w.WriteInt` / `r.ReadUint32`.

- [ ] **Step 4: Run the test to verify it passes**

```
cd libs/atlas-packet && go test ./monster/serverbound/ -run TestAutoAggro -v
```

Expected: PASS, including every `pt.Variants` subtest.

- [ ] **Step 5: Link the fname in packet-audit**

In `tools/packet-audit/cmd/run.go`, inside `candidatesFromFName`, add immediately after the `case "CMob::SendDropPickUpRequest":` arm (ends at line 1200):

```go
case "CMob::ApplyControl":
    // task-255: AUTO_AGGRO — atlas AutoAggro (handle = "AutoAggro").
    // Two Encode4 (mobId, distance), identical on all ten versions.
    return []candidate{{name: "AutoAggro", pkg: "monster", dir: csvpkg.DirServerbound}}
```

- [ ] **Step 6: Build both modules**

```
cd libs/atlas-packet && go build ./... && go test ./monster/serverbound/
cd tools/packet-audit && go build ./... && go test ./...
```

Expected: both PASS.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/monster/serverbound/auto_aggro.go libs/atlas-packet/monster/serverbound/auto_aggro_test.go tools/packet-audit/cmd/run.go
git commit -m "feat(atlas-packet): add AUTO_AGGRO (CMob::ApplyControl) serverbound codec"
```

---

## Task 2: Registry corrections for `AUTO_AGGRO` (ten versions)

Mechanical, same edit repeated per file — batched deliberately (see context.md).

### Files

- `docs/packets/registry/gms_v48.yaml` — **add** an `AUTO_AGGRO` serverbound entry
- `docs/packets/registry/gms_v61.yaml` — **add**
- `docs/packets/registry/gms_v72.yaml` — **add**
- `docs/packets/registry/gms_v79.yaml` — **add**
- `docs/packets/registry/gms_v83.yaml:3085` — bump provenance + add `ida.address`
- `docs/packets/registry/gms_v84.yaml:3856` — **correct opcode 189 → 194**, bump provenance, replace note
- `docs/packets/registry/gms_v87.yaml:3249` — bump provenance + `ida.address`
- `docs/packets/registry/gms_v92.yaml:3530` — bump provenance + `ida.address`
- `docs/packets/registry/gms_v95.yaml:3602` — bump provenance + `ida.address`
- `docs/packets/registry/jms_v185.yaml:3189` — bump provenance + `ida.address`
- `docs/packets/registry/README.md` — read-only; provenance vocabulary

`ida.address` is a **decimal** integer in this schema (see `gms_v84.yaml:3868`, `address: 6835423` for `0x684CDF`).

- [ ] **Step 1: Add the four missing entries**

For v48 / v61 / v72 / v79, insert the entry **immediately before** that file's `- op: MOB_DROP_PICKUP_REQUEST` block (v48 `:1834`, v61 `:2667`, v72 `:2873`, v79 `:3223`), so the file stays in opcode order:

| file | opcode | ida.address (dec) | ida.address (hex) |
|---|---|---|---|
| `gms_v48.yaml` | 130 | 5577849 | `0x551c79` |
| `gms_v61.yaml` | 156 | 6082332 | `0x5ccf1c` |
| `gms_v72.yaml` | 179 | 6411096 | `0x61d358` |
| `gms_v79.yaml` | 181 | 6541542 | `0x63d0e6` |

Entry shape (values from the table; note text per file cites that file's hex address):

```yaml
- op: AUTO_AGGRO
  direction: serverbound
  opcode: 130
  fname: CMob::ApplyControl
  provenance: ida-discovered
  ida:
    address: 5577849
  note: 'task-255: the registry had NO AUTO_AGGRO entry for this version — the ops CSVs have no column before GMS v83, which is a CSV blind spot, not a client absence. CMob::ApplyControl @0x551c79 builds COutPacket(130) and encodes Encode4(fused mobId)+Encode4(proximity score). Adjacent to MOB_DROP_PICKUP_REQUEST=131.'
```

- [ ] **Step 2: Correct gms_v84**

Replace lines 3856-3861 of `docs/packets/registry/gms_v84.yaml` with:

```yaml
- op: AUTO_AGGRO
  direction: serverbound
  opcode: 194
  fname: CMob::ApplyControl
  provenance: ida-discovered
  ida:
    address: 6833298
  note: 'task-255: opcode corrected 189->194 (0xBD->0xC2). The prior 189 was seeded from the v83 CSV column (the CSVs have no v84 column) and is stale — the v84 serverbound table shifts in this cluster. IDA v84 CMob::ApplyControl @0x684492 builds COutPacket(194); Encode4(fused mobId)+Encode4(proximity score). Adjacent pair with MOB_DROP_PICKUP_REQUEST=195 (@0x684CDF), which was corrected the same way in task-092. No other serverbound entry in this file claims 189 or 194.'
```

- [ ] **Step 3: Bump provenance on the five csv-import rows**

For v83, v87, v92, v95, jms_v185: leave `op`, `direction`, `opcode`, `fname` unchanged; change `provenance: csv-import` → `provenance: ida-discovered`, and add an `ida:` block plus a `note`:

| file | opcode (unchanged) | ida.address (dec) | hex |
|---|---|---|---|
| `gms_v83.yaml` | 189 | 6742342 | `0x66e146` |
| `gms_v87.yaml` | 201 | 6983777 | `0x6a9061` |
| `gms_v92.yaml` | 221 | 6513440 | `0x636320` |
| `gms_v95.yaml` | 228 | 6556960 | `0x640d20` |
| `jms_v185.yaml` | 195 | 7256636 | `0x6eba3c` |

Note text (substitute the file's own opcode and hex address):

```yaml
  note: 'task-255: opcode confirmed against the binary. CMob::ApplyControl @0x66e146 builds COutPacket(189); Encode4(fused mobId)+Encode4(proximity score). Promoted off csv-import.'
```

- [ ] **Step 4: Verify no duplicate opcode was introduced**

```
python3 -c "
import yaml,glob
for f in sorted(glob.glob('docs/packets/registry/*.yaml')):
    d=yaml.safe_load(open(f))
    rows=[e for e in d if isinstance(e,dict) and e.get('op')=='AUTO_AGGRO']
    sb=[e for e in d if isinstance(e,dict) and e.get('direction')=='serverbound']
    for r in rows:
        dupes=[e['op'] for e in sb if e.get('opcode')==r['opcode']]
        print(f, r['opcode'], r['provenance'], 'dupes:', dupes)
"
```

Expected: exactly one `AUTO_AGGRO` row per file for all ten version files, every `provenance` reading `ida-discovered`, every opcode matching the Global Constraints table (v84 must read `194`), and every `dupes` list containing only `AUTO_AGGRO`. If the registry files are not a top-level YAML list, adapt the walk — do not skip the duplicate-opcode check.

- [ ] **Step 5: Commit**

```bash
git add docs/packets/registry/
git commit -m "fix(packets): ground AUTO_AGGRO registry entries in the binaries on all ten versions"
```

---

## Task 3: Route `AUTO_AGGRO` in the ten seed templates

Mechanical, same edit repeated per file — batched deliberately (see context.md).

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_48_1.json` — insert route `0x82`
- `services/atlas-configurations/seed-data/templates/template_gms_61_1.json` — insert route `0x9C`
- `services/atlas-configurations/seed-data/templates/template_gms_72_1.json` — insert route `0xB3`
- `services/atlas-configurations/seed-data/templates/template_gms_79_1.json` — insert route `0xB5`
- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` — insert route `0xBD`
- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json` — insert route `0xC2`
- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` — insert route `0xC9`
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — insert route `0xDD`
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — insert route `0xE4`
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — insert route `0xC3`
- `services/atlas-configurations/seed-data/templates/template_gms_12_1.json` — **do not touch** (task-175 owns it; no registry column)
- `docs/packets/TEMPLATE_CONVENTIONS.md` — read-only

- [ ] **Step 1: Insert the ten routes**

Each goes into the file's `socket.handlers` array at its ascending-`opCode` position. In eight of the ten files that is **immediately before** the existing `"handler": "MobDropPickupRequest"` entry (AUTO_AGGRO is always drop-pickup minus one). In `template_gms_48_1.json` and `template_gms_92_1.json` `MobDropPickupRequest` is not routed — there, insert **immediately after** the `"handler": "MonsterMovementHandle"` entry (`0x81` for v48, `0xDC` for v92).

Entry, with that file's opcode substituted, matching the surrounding 8-space indentation and the hand-compacted `"services"` array style already in the file:

```json
      {
        "opCode": "0xBD",
        "validator": "LoggedInValidator",
        "handler": "AutoAggro",
        "fname": "CMob::ApplyControl",
        "services": [
          "channel"
        ]
      },
```

Do **not** run `packet-audit seed-fname --write` — it reformats every array in the file (README "Semantic content is unaffected — only inter-token whitespace"), producing a large cosmetic diff. Hand-write the `fname`.

- [ ] **Step 2: Verify every file is still valid JSON and ordered**

```
for f in services/atlas-configurations/seed-data/templates/template_*_1.json; do python3 -c "import json,sys; json.load(open('$f'))" || echo "BAD $f"; done
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
```

Expected: no `BAD` lines; both guards exit 0.

- [ ] **Step 3: Confirm ten routes exist and gms_12 is untouched**

```
grep -l '"handler": "AutoAggro"' services/atlas-configurations/seed-data/templates/*.json | wc -l
git diff --name-only -- services/atlas-configurations/seed-data/templates/
```

Expected: `10`; the diff lists exactly the ten files above and **not** `template_gms_12_1.json`.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(config): route AUTO_AGGRO in all ten applicable seed templates"
```

---

## Task 4: Splice `CMob::ApplyControl` into the ten IDA exports

Requires a live ida-pro-mcp server with all ten IDBs open. `evidence pin` resolves the address from the export's `functions` map; `CMob::ApplyControl` is currently absent from **every** export, so this task is the precondition for Task 5.

### Files

- `docs/packets/ida-exports/gms_v48.json` — splice one entry
- `docs/packets/ida-exports/gms_v61.json` — splice one entry
- `docs/packets/ida-exports/gms_v72.json` — splice one entry
- `docs/packets/ida-exports/gms_v79.json` — splice one entry
- `docs/packets/ida-exports/gms_v83.json` — splice one entry
- `docs/packets/ida-exports/gms_v84.json` — splice one entry
- `docs/packets/ida-exports/gms_v87.json` — splice one entry
- `docs/packets/ida-exports/gms_v92.json` — splice one entry
- `docs/packets/ida-exports/gms_v95.json` — splice one entry
- `docs/packets/ida-exports/gms_jms_185.json` — splice one entry (jms uses this filename)
- `tools/packet-audit/README.md` — read-only; `export --splice` contract
- `docs/packets/audits/VERIFYING_A_PACKET.md` — read-only; §10 export hygiene

Module root for the tool: `tools/packet-audit`.

- [ ] **Step 1: Confirm the fname is genuinely absent from every export**

```
python3 -c "
import json,glob,os
for f in sorted(glob.glob('docs/packets/ida-exports/*.json')):
    d=json.load(open(f))
    print(os.path.basename(f), [k for k in d['functions'] if 'ApplyControl' in k])
"
```

Expected: an empty list on every line. (If a line is non-empty, that version needs no splice — skip it in Step 3.)

- [ ] **Step 2: Resolve the live session ids**

Call `mcp__ida-pro__idb_list` and match each session to its version by the `filename` field:
`GMS_v48_1_DEVM.exe.i64`, `GMS_v61.1_U_DEVM.exe.i64`, `GMS_v72.1_U_DEVM.exe.i64`, `GMS_v79_1_DEVM.exe.i64`, `MapleStory_dump.exe.i64` (v83), `GMS_v84.1_U_DEVM.i64`, `GMSv87_4GB.exe.i64`, `GMS_v92_1_DEVM.exe.i64`, `GMS_v95.0_U_DEVM.exe.i64`, `MapleStory_dump_SCY.exe.i64` (jms_v185).

For each, confirm the function is named at the address from the Global Constraints table with `mcp__ida-pro__lookup_funcs` — the expected `name` is the mangled `?ApplyControl@CMob@@IAEXJ@Z`, which `packet-audit`'s `demangleQualified` maps to `CMob::ApplyControl`. If any address does **not** resolve to that name, STOP and report — do not rename or guess.

- [ ] **Step 3: Targeted harvest + splice, one version at a time**

Write a one-line roster file first:

```bash
mkdir -p /tmp/task255 && printf 'CMob::ApplyControl\n' > /tmp/task255/roster.md
```

Then per version (`--prior-export ""` keeps the harvest to the roster; `--splice` merges exactly that entry into the committed file and preserves every other entry byte-for-byte):

```bash
cd tools/packet-audit && go run . export \
  --version gms_v83 \
  --output ../../docs/packets/ida-exports/gms_v83.json \
  --prior-export "" \
  --pending /tmp/task255/roster.md \
  --splice "CMob::ApplyControl" \
  --ida-database <session id> \
  --descent-depth 12
```

Never pass `--force`. Never re-harvest a whole export.

- [ ] **Step 4: Verify each splice landed and nothing else moved**

```
python3 -c "
import json,glob,os
for f in sorted(glob.glob('docs/packets/ida-exports/*.json')):
    d=json.load(open(f))
    e=d['functions'].get('CMob::ApplyControl')
    print(os.path.basename(f), e and (e.get('address'), e.get('direction')))
"
git diff --stat -- docs/packets/ida-exports/
```

Expected: every line prints the address from the Global Constraints table with `direction` `serverbound` (backfilled from the Task-1 `candidatesFromFName` arm — if it is empty, Task 1's arm is missing and must land first). The `git diff --stat` must show only small per-file insertions; a diff of hundreds of changed lines means a wholesale re-export happened — `git checkout` the file and redo with `--splice`.

If a splice fails with a `COutPacket` delegate error, strip that one `{op: Delegate, ref: COutPacket}` call from the spliced entry (VERIFYING_A_PACKET §10) and re-check.

- [ ] **Step 5: Commit**

```bash
git add docs/packets/ida-exports/
git commit -m "chore(packets): splice CMob::ApplyControl into the ten IDA exports"
```

---

## Task 5: Audit reports, evidence pins, matrix promotion

### Files

- `docs/packets/audits/gms_v48/MonsterAutoAggro.json` + `.md` — **new files** (generated then copied)
- `docs/packets/audits/gms_v61/MonsterAutoAggro.json` + `.md` — **new files**
- `docs/packets/audits/gms_v72/MonsterAutoAggro.json` + `.md` — **new files**
- `docs/packets/audits/gms_v79/MonsterAutoAggro.json` + `.md` — **new files**
- `docs/packets/audits/gms_v83/MonsterAutoAggro.json` + `.md` — **new files**
- `docs/packets/audits/gms_v84/MonsterAutoAggro.json` + `.md` — **new files**
- `docs/packets/audits/gms_v87/MonsterAutoAggro.json` + `.md` — **new files**
- `docs/packets/audits/gms_v92/MonsterAutoAggro.json` + `.md` — **new files**
- `docs/packets/audits/gms_v95/MonsterAutoAggro.json` + `.md` — **new files**
- `docs/packets/audits/jms_v185/MonsterAutoAggro.json` + `.md` — **new files**
- `docs/packets/evidence/<version>/monster.serverbound.MonsterAutoAggro.yaml` × 10 — **new files** (written by `evidence pin`, then hand-edited to add `verifies:`)
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated
- `docs/packets/evidence/gms_v83/monster.serverbound.MonsterMobDropPickupRequest.yaml` — read-only; the record shape to match

Module root: `tools/packet-audit`.

- [ ] **Step 1: Generate the ten audit reports**

The root pipeline is deterministic against the committed export — no live IDA needed. Per version (jms uses `--template template_jms_185_1.json` and `--ida-source docs/packets/ida-exports/gms_jms_185.json`):

```bash
cd tools/packet-audit && go run . \
  -csv-clientbound "../../docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "../../docs/packets/MapleStory Ops - ServerBound.csv" \
  -template ../../services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  -ida-source ../../docs/packets/ida-exports/gms_v83.json \
  -atlas-packet ../../libs/atlas-packet \
  -output /tmp/task255/rpt
```

Then copy **only** `MonsterAutoAggro.{json,md}` out of `/tmp/task255/rpt/<version>/` into `docs/packets/audits/<version>/`. Do not copy any other regenerated report — churn on unrelated reports is out of scope and degrades other cells.

If `MonsterAutoAggro.json` is not produced for a version, the linkage is broken: re-check Task 1's `candidatesFromFName` arm, the Task 3 route (the report is only written for a routed op), and the Task 4 splice. Do not hand-author a report.

- [ ] **Step 2: Pin the ten evidence records**

```bash
cd tools/packet-audit && go run . evidence pin \
  --packet monster/serverbound/MonsterAutoAggro \
  --version gms_v83 \
  --ida "CMob::ApplyControl" \
  --category TIER1-FIXTURE
```

…repeated for all ten version keys (`gms_v48 gms_v61 gms_v72 gms_v79 gms_v83 gms_v84 gms_v87 gms_v92 gms_v95 jms_v185`).

Then hand-edit each generated `docs/packets/evidence/<version>/monster.serverbound.MonsterAutoAggro.yaml` to append:

```yaml
verifies:
    - libs/atlas-packet/monster/serverbound/auto_aggro_test.go#TestAutoAggro
```

- [ ] **Step 3: Regenerate and check the matrix**

`tools/packet-audit` is its own Go module, and these subcommands resolve their
default paths relative to the **repo root** — running them from the module
directory emits "no template for gms_v87"-style warnings and can write a stray
`tools/packet-audit/docs/packets/...` tree. Build the binary once, then run it
from the repo root:

```bash
go build -o /tmp/task255/packet-audit ./tools/packet-audit   # from the module dir or via -C
/tmp/task255/packet-audit matrix && /tmp/task255/packet-audit matrix --check
/tmp/task255/packet-audit fname-doc --check
/tmp/task255/packet-audit operations --check
```

Expected: `matrix --check` exits 0; `fname-doc --check` and `operations --check` exit 0. (`fname-doc` may need a non-`--check` regeneration run first if it reports the new struct's comment — the codec already carries `packet-audit:fname CMob::ApplyControl`, so it should be clean.)

- [ ] **Step 4: Confirm the row promoted on all ten columns**

```
grep -n "AUTO_AGGRO" docs/packets/audits/STATUS.md
python3 -c "
import json
d=json.load(open('docs/packets/audits/status.json'))
r=[x for x in d['rows'] if x.get('op')=='AUTO_AGGRO'][0]
print(r.get('packet'), r.get('tier1'))
for k,v in r['cells'].items(): print(k, v['state'], v['opcode'])
"
```

Expected: `packet` is `monster/serverbound/MonsterAutoAggro`, `tier1` is `true`, and **every one of the ten cells** reads `verified` with the opcode from the Global Constraints table (v84 must read `194`). No `n-a`, no `incomplete`, no `partial`. A cell that did not promote is a failure — diagnose it against `VERIFYING_A_PACKET.md` "Failure modes", do not narrate around it.

- [ ] **Step 5: Confirm no `feature-na-evidence.yaml` entry is needed**

```
python3 -c "print('AUTO_AGGRO' in open('docs/packets/feature-na-evidence.yaml').read())"
```

Expected: `False`. Nothing is `n-a` (design §9), so no entry is written and none is removed.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/audits/ docs/packets/evidence/
git commit -m "test(packets): verify AUTO_AGGRO across all ten versions"
```

---

## Task 6: `information.Model.FirstAttack()`

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/information/model.go` — add `firstAttack bool` field + `FirstAttack()` accessor
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go:98` — map `rm.FirstAttack` in `Extract`
- `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go` — add `firstAttack` field, `SetFirstAttack`, and carry it in `Build()`
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest_test.go` — add the test
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go:35` — read-only; `FirstAttack bool \`json:"first_attack"\`` already exists on `RestModel`

Module root: `services/atlas-monsters/atlas.com/monsters`.

**Interfaces produced:** `information.Model.FirstAttack() bool`; `information.NewModelBuilder().SetFirstAttack(bool) *ModelBuilder`. Task 9 consumes both.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-monsters/atlas.com/monsters/monster/information/rest_test.go` (package `information`; copy the existing file's imports and table style):

`TestExtractFirstAttack` — table-driven, two cases:

| case | `RestModel.FirstAttack` | expect `Model.FirstAttack()` |
|---|---|---|
| aggressive template | `true` | `true` |
| passive template | `false` | `false` |

Build the input as `RestModel{FirstAttack: <case value>}` and call `Extract(rm)`; fail with `t.Fatalf("FirstAttack() = %v, want %v", got.FirstAttack(), want)`.

Second test, `TestModelBuilderSetFirstAttack`: `NewModelBuilder().SetFirstAttack(true).Build().FirstAttack()` must be `true`, and `NewModelBuilder().Build().FirstAttack()` must be `false` (zero value denies aggro — FR-5.2).

- [ ] **Step 2: Run the test to verify it fails**

```
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/information/ -run 'FirstAttack'
```

Expected: FAIL — `got.FirstAttack undefined`.

- [ ] **Step 3: Implement**

`model.go`: add `firstAttack bool` to the `Model` struct (after `friendly`), and

```go
// FirstAttack reports whether the template is aggressive — Mob/<id>.img/info/firstAttack.
// This is the gate that separates a genuinely aggressive mob from one that
// merely picks up drops: CMob::ApplyControl fires for bPickUpDrop templates too.
func (m Model) FirstAttack() bool {
    return m.firstAttack
}
```

`rest.go`: add `firstAttack: rm.FirstAttack,` to the `Model{...}` literal in `Extract`.

`builder.go`: add `firstAttack bool` to `ModelBuilder`, add `firstAttack: b.firstAttack,` to `Build()`'s returned literal, and add the setter following the `SetBoss` shape:

```go
// SetFirstAttack sets the aggressive-template flag on the builder. Used by
// tests that drive the firstAttack gate in ProcessorImpl.SetAggro.
func (b *ModelBuilder) SetFirstAttack(v bool) *ModelBuilder {
    b.firstAttack = v
    return b
}
```

- [ ] **Step 4: Run the tests**

```
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/information/
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/
git commit -m "feat(atlas-monsters): expose firstAttack on the monster information model"
```

---

## Task 7: Aggro lease state on the monster registry

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/model.go` — add `aggroRefreshedMs int64` + `AggroRefreshedMs()`; change `ControlWithAggro` to take `nowMs`
- `services/atlas-monsters/atlas.com/monsters/monster/builder.go` — add `aggroRefreshedMs` to `Clone`, the builder struct, `SetAggroRefreshedMs`, and `Build()`
- `services/atlas-monsters/atlas.com/monsters/monster/registry.go` — add `AggroRefreshedMs` to `storedMonster` / `toStored` / `fromStored`; change `ControlMonsterWithAggro` to take `nowMs`; add `SetAggro`
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go:414` — pass `nowMs` at the one `ControlMonsterWithAggro` call site
- `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go` — add the tests
- `services/atlas-monsters/atlas.com/monsters/monster/registry.go:695` — read-only; `DecaySummary` / `DecayDamageEntries` is the shape `AggroSummary` / `SetAggro` mirrors

Module root: `services/atlas-monsters/atlas.com/monsters`.

**Interfaces produced (Tasks 9 and 10 consume these):**

```go
type AggroSummary struct {
    Monster    Model
    Changed    bool // controllerHasAggro flipped false -> true on this call
    LeaseOnly  bool // already aggro'd by this controller; only the lease was stamped
}
func (r *Registry) SetAggro(t tenant.Model, uniqueId uint32, characterId uint32, nowMs int64) (AggroSummary, error)
func (r *Registry) ControlMonsterWithAggro(tenant tenant.Model, uniqueId uint32, characterId uint32, nowMs int64) (Model, error)
func (m Model) AggroRefreshedMs() int64
func (m Model) ControlWithAggro(characterId uint32, nowMs int64) Model
```

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go` (package `monster`; reuse the file's existing tenant/field/clear setup idiom — `newTestTenant(t)`, `tenant.WithContext`, `GetMonsterRegistry().Clear(ctx)`, `CreateMonster(...)`).

`TestRegistrySetAggro` — table-driven, one subtest per case. Each case seeds one monster (`CreateMonster(ctx, tm, testField(), 9000000, 0,0,0,0,0, 100000, 50, "", "")`), applies the "given" via `ControlMonster` / `ControlMonsterWithAggro`, then calls `SetAggro(tm, uid, characterId, nowMs)`:

| subtest | given | call | expect `Changed` | expect `LeaseOnly` | expect `Monster.ControllerHasAggro()` | expect `Monster.AggroRefreshedMs()` |
|---|---|---|---|---|---|---|
| `controller without aggro flips` | `ControlMonster(tm, uid, 7)` | `SetAggro(tm, uid, 7, 5000)` | `true` | `false` | `true` | `5000` |
| `controller with aggro stamps lease only` | `ControlMonsterWithAggro(tm, uid, 7, 1000)` | `SetAggro(tm, uid, 7, 9000)` | `false` | `true` | `true` | `9000` |
| `non-controller is not applied` | `ControlMonster(tm, uid, 7)` | `SetAggro(tm, uid, 9, 5000)` | `false` | `false` | `false` | `0` |
| `damage entries untouched` | `ControlMonster(tm, uid, 7)` then `ApplyDamage(tm, 8, 100, uid, 1000)` | `SetAggro(tm, uid, 7, 5000)` | `false` | `true` | `true` | `5000` |

> Corrected during execution (Task 7 review, adjudicated against design.md §6.5/§10):
> the last row originally read `Changed=true, LeaseOnly=false`. `ApplyDamage`
> flips `ControllerHasAggro` on the first hit against *any* controlled monster,
> regardless of who dealt the damage, so the given-sequence already leaves
> `ControllerHasAggro=true` before `SetAggro` runs — making this a
> controller-with-aggro call, which stamps the lease and emits nothing.

The last case additionally asserts `len(got.Monster.DamageEntries()) == 1` and `got.Monster.DamageEntries()[0].CharacterId == 8` — `SetAggro` must never write a damage entry (FR-4.5).

Plus `TestRegistrySetAggroMissingMonster`: `SetAggro(tm, 4242, 7, 5000)` returns `errMonsterNotFound` (assert with `errors.Is`).

Plus `TestControlMonsterWithAggroStampsLease`: `ControlMonsterWithAggro(tm, uid, 9, 7777)` returns a model with `ControllerHasAggro() == true`, `ControlCharacterId() == 9`, and `AggroRefreshedMs() == 7777`.

Plus `TestAggroRefreshedMsRoundTripsThroughRedis`: after `ControlMonsterWithAggro(tm, uid, 9, 7777)`, `GetMonster(tm, uid)` still reports `AggroRefreshedMs() == 7777` (proves the `storedMonster` field is serialized).

- [ ] **Step 2: Run to verify failure**

```
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'SetAggro|AggroRefreshedMs|ControlMonsterWithAggroStampsLease'
```

Expected: FAIL to compile — `r.SetAggro undefined`, `too many arguments in call to ControlMonsterWithAggro`.

- [ ] **Step 3: Implement the model/builder changes**

`model.go`:

```go
// aggroRefreshedMs is the aggro lease stamp: the last time a SET_AGGRO claim
// was accepted (or renewed) for this monster. Auto-aggro writes no damage
// entry, so the damage-entry decay sweep cannot release it; the lease is what
// MonsterAggroDecayTask reads to expire an auto-aggro'd mob (design §4).
aggroRefreshedMs int64
```

plus `func (m Model) AggroRefreshedMs() int64 { return m.aggroRefreshedMs }`, and change:

```go
func (m Model) ControlWithAggro(characterId uint32, nowMs int64) Model {
    return Clone(m).
        SetControlCharacterId(characterId).
        SetControllerHasAggro(true).
        SetAggroRefreshedMs(nowMs).
        Build()
}
```

`builder.go`: carry `aggroRefreshedMs: m.aggroRefreshedMs` in `Clone`, add the field to `ModelBuilder`, add `SetAggroRefreshedMs(v int64) *ModelBuilder` following `SetLastDamageTakenMs`'s shape, and add `aggroRefreshedMs: b.aggroRefreshedMs` to `Build()`.

- [ ] **Step 4: Implement the registry changes**

`registry.go`:

- add `AggroRefreshedMs int64 \`json:"aggroRefreshedMs,omitempty"\`` to `storedMonster`, next to `LastDamageTakenMs`;
- map it both ways in `toStored` and `fromStored`;
- change `ControlMonsterWithAggro(tenant tenant.Model, uniqueId uint32, characterId uint32, nowMs int64)` to call `m.ControlWithAggro(characterId, nowMs)`;
- add `AggroSummary` and `SetAggro`, written against `storedMonster` via `r.reg.Update` (the same reason `DecayDamageEntries` is: `Model` exposes no builder path that reads the pre-state for the emit decision). The closure derives `changed`/`leaseOnly` purely from `cur`, so the captured values reflect the final successful invocation under optimistic-lock retry:

```go
// SetAggro stamps the aggro lease for the monster's CURRENT controller and
// flips controllerHasAggro true if it was not already. It is a no-op when
// characterId is not the current controller — arbitration for a non-controller
// claimant is the processor's job (design §2), not the registry's. No damage
// entry is created on any path (FR-4.5).
func (r *Registry) SetAggro(t tenant.Model, uniqueId uint32, characterId uint32, nowMs int64) (AggroSummary, error)
```

Body: inside the `Update` closure, `if cur.ControlCharacterId != characterId { return cur }` (leaving both flags false); otherwise set `cur.AggroRefreshedMs = nowMs`, and if `cur.ControllerHasAggro` was false set it true and `changed = true`, else `leaseOnly = true`. Map `atlasredis.ErrNotFound` to `errMonsterNotFound` exactly as `DecayDamageEntries` does.

`processor.go:414`: pass the timestamp — see Task 9 Step 3 for `p.nowFn`; until that lands, use `time.Now().UnixMilli()` inline and Task 9 replaces it.

- [ ] **Step 5: Run the tests**

```
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/
```

Expected: PASS, including the pre-existing `force_control_test.go` and `clear_aggro_test.go` suites.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/model.go services/atlas-monsters/atlas.com/monsters/monster/builder.go services/atlas-monsters/atlas.com/monsters/monster/registry.go services/atlas-monsters/atlas.com/monsters/monster/registry_test.go services/atlas-monsters/atlas.com/monsters/monster/processor.go
git commit -m "feat(atlas-monsters): add the aggro lease and registry SetAggro"
```

---

## Task 8: Aggro lease release in the decay sweep

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/aggro.go` — add `AutoAggroLeaseTtlMs`
- `services/atlas-monsters/atlas.com/monsters/monster/aggro_task.go:76` — add the no-damage-entries lease branch in `Run`
- `services/atlas-monsters/atlas.com/monsters/monster/registry.go` — add `ReleaseAggroLease`
- `services/atlas-monsters/atlas.com/monsters/monster/aggro_task_test.go` — add the tests
- `services/atlas-monsters/atlas.com/monsters/monster/aggro_task_test.go:28` — read-only; `newAggroTaskWithRecorder` is the harness (injects `bossLookupFn`, `emit`, and the task struct's `nowFn`)

Module root: `services/atlas-monsters/atlas.com/monsters`.

**Interfaces produced:** `monster.AutoAggroLeaseTtlMs` (int64, 15000); `(*Registry).ReleaseAggroLease(t tenant.Model, uniqueId uint32) (DecaySummary, error)`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-monsters/atlas.com/monsters/monster/aggro_task_test.go`. Use `newAggroTaskWithRecorder(t, nil)` for the task and set `tk.nowFn = func() int64 { return <fixed> }` per case; seed monsters with `GetMonsterRegistry()` directly as the existing tests do.

`TestAggroDecayTaskReleasesExpiredAutoAggroLease` — table-driven, four cases. Each seeds one non-boss monster in `field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()` via `CreateMonster(ctx, ten, f, 9300018, 0,0,0,5,0, 1000, 50, "", "")`, then applies the given, sets `tk.nowFn`, and calls `tk.Run()`:

| subtest | given | `nowFn` | expect emitted `AGGRO_CHANGED` | expect `ControllerHasAggro()` after |
|---|---|---|---|---|
| `expired lease releases` | `ControlMonsterWithAggro(ten, uid, 7, 1000)`, no damage entries | `20000` | 1 event, `Type == EventStatusAggroChanged`, `ControllerCharacterId == 7`, `ControllerHasAggro == false` | `false` |
| `lease inside ttl is kept` | `ControlMonsterWithAggro(ten, uid, 7, 1000)`, no damage entries | `10000` | 0 events | `true` |
| `damage entries take the existing path` | `ControlMonsterWithAggro(ten, uid, 7, 1000)` then `ApplyDamage(ten, 7, 100, uid, 19000)` | `20000` | 0 events (entry is not idle: `20000-19000 < AggroIdleThresholdMs`) | `true` |
| `no aggro, no work` | `ControlMonster(ten, uid, 7)`, no damage entries | `20000` | 0 events | `false` |

Plus `TestAggroDecayTaskLeaseSkipsBosses`: seed as in case 1 but pass `map[uint32]bool{9300018: true}` to `newAggroTaskWithRecorder`; expect 0 events and the flag still `true`.

Plus `TestAggroDecayTaskLeaseLeavesControllerIntact` (case-1 setup): after `Run()`, `ControlCharacterId()` is still `7` — losing aggro is not losing control, matching `DecayDamageEntries`.

- [ ] **Step 2: Run to verify failure**

```
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'AggroDecayTaskLease|AggroDecayTaskReleasesExpired'
```

Expected: FAIL — the release never happens (0 events where 1 is expected).

- [ ] **Step 3: Implement**

`aggro.go`, appended to the existing const block's file:

```go
// AutoAggroLeaseTtlMs is how long an auto-aggro claim survives without a
// refresh. Auto-aggro creates no damage entry (FR-4.5), so the damage-entry
// decay path can never release it; this lease is what does. The channel
// forwards a refresh at most once per AutoAggroRefreshInterval (5s), so 15s
// tolerates two missed refreshes before the mob goes passive.
const AutoAggroLeaseTtlMs = int64(15_000)
```

`registry.go`:

```go
// ReleaseAggroLease clears controllerHasAggro on a monster whose auto-aggro
// lease has expired, leaving the controller and the (empty) damage-entry list
// alone. Returns a DecaySummary so the sweep's emit decision is identical to
// the damage-decay path's.
func (r *Registry) ReleaseAggroLease(t tenant.Model, uniqueId uint32) (DecaySummary, error)
```

Body mirrors `ClearDamageEntries` (`r.reg.Update` over `storedMonster`): capture `controllerCharacterId = cur.ControlCharacterId`; if `cur.ControllerHasAggro` set it false and `aggroFlippedOff = true`; do **not** touch `cur.DamageEntries`.

`aggro_task.go`, inside `Run`'s per-monster loop, replacing the bare `if len(entries) == 0 { continue }`:

```go
entries := m.DamageEntries()
if len(entries) == 0 {
    // Auto-aggro lease (design §4): a mob aggro'd with no damage entries is
    // invisible to the damage-decay path below, so it would stay aggro'd
    // forever and keep making skill decisions against nobody.
    if m.ControllerHasAggro() && nowMs-m.AggroRefreshedMs() > AutoAggroLeaseTtlMs {
        summary, err := GetMonsterRegistry().ReleaseAggroLease(ten, m.UniqueId())
        if err != nil {
            tk.l.WithError(err).Errorf("Aggro lease release failed for monster [%d].", m.UniqueId())
            continue
        }
        if summary.AggroFlippedOff {
            _ = tk.emit(ten, EnvEventTopicMonsterStatus, aggroChangedStatusEventProvider(summary.Monster, summary.ControllerCharacterId, false))
            tk.l.Debugf("Auto-aggro lease expired; monster [%d] released from controller [%d].", summary.Monster.UniqueId(), summary.ControllerCharacterId)
        }
    }
    continue
}
```

The boss skip above it is unchanged, so bosses never reach this branch.

- [ ] **Step 4: Run the tests**

```
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/
```

Expected: PASS, including the pre-existing `aggro_task_test.go` cases.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/aggro.go services/atlas-monsters/atlas.com/monsters/monster/aggro_task.go services/atlas-monsters/atlas.com/monsters/monster/registry.go services/atlas-monsters/atlas.com/monsters/monster/aggro_task_test.go
git commit -m "feat(atlas-monsters): expire auto-aggro leases in the decay sweep"
```

---

## Task 9: `ProcessorImpl.SetAggro` — the authoritative gates and arbitration

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — add `nowFn` to `ProcessorImpl` + `NewProcessor`, add `SetAggro` to the `Processor` interface and the impl
- `services/atlas-monsters/atlas.com/monsters/monster/set_aggro_test.go` — **new file**; the gate + arbitration tests
- `services/atlas-monsters/atlas.com/monsters/monster/force_control_test.go` — read-only; `forceControlProcessor` is the seam-wiring pattern to copy
- `services/atlas-monsters/atlas.com/monsters/monster/clear_aggro_test.go:13` — read-only; `newAggroedMonster` helper
- `services/atlas-monsters/atlas.com/monsters/monster/control_assignment_test.go:17` — read-only; `recordingProcessor` helper

Module root: `services/atlas-monsters/atlas.com/monsters`.

**Interfaces consumed:** `information.Model.FirstAttack()` (Task 6), `Registry.SetAggro` / `AggroSummary` / `ControlMonsterWithAggro(..., nowMs)` (Task 7).
**Interfaces produced:** `Processor.SetAggro(uniqueId uint32, characterId uint32) error` — Task 10's consumer arm calls it.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-monsters/atlas.com/monsters/monster/set_aggro_test.go`, package `monster`.

Local helper mirroring `forceControlProcessor`, plus a fixed clock:

```go
func setAggroProcessor(ctx context.Context, tm tenant.Model, emitted *int, inField []uint32, hidden map[uint32]struct{}, nowMs int64) *ProcessorImpl
```

— it calls `recordingProcessor(ctx, tm, emitted)`, sets `p.inFieldFn` / `p.hiddenFn` like `forceControlProcessor` does, and sets `p.nowFn = func() int64 { return nowMs }`.

Template lookups go through the existing package hook, saved and restored per test:

```go
prevHook := testInformationLookup
testInformationLookup = func(_ uint32) (information.Model, error) {
    return information.NewModelBuilder().SetFirstAttack(true).Build(), nil
}
defer func() { testInformationLookup = prevHook }()
```

(copy that save/restore idiom from `drain_mp_test.go:28`).

`TestSetAggro_Gates` — table-driven; each case seeds via `newAggroedMonster(t, ctx, tm, <controller>, nil)` unless noted, then calls `p.SetAggro(uid, claimant)`:

| subtest | given | information hook | claimant | expect err | expect emitted | expect `ControllerHasAggro()` | expect `ControlCharacterId()` |
|---|---|---|---|---|---|---|---|
| `unknown monster` | nothing seeded; call `SetAggro(4242, 9)` | `FirstAttack=true` | 9 | `nil` | 0 | — | — |
| `dead monster` | controller 7; then `ApplyDamage(tm, 7, 100000, uid, 1000)` to zero HP | `FirstAttack=true` | 7 | `nil` | 0 (beyond the damage path's own) | `false` | 7 |
| `passive template` | controller 7 | `FirstAttack=false` | 7 | `nil` | 0 | `false` | 7 |
| `information lookup error` | controller 7 | returns `errors.New("boom")` | 7 | `nil` | 0 | `false` | 7 |
| `character not in field` | controller 7, `inField = []uint32{7}` | `FirstAttack=true` | 9 | `nil` | 0 | `false` | 7 |

For the `dead monster` case, count emissions with a fresh `emitted` counter reset to 0 immediately before the `SetAggro` call.

`TestSetAggro_Arbitration` — table-driven; `inField` is `[]uint32{7, 9}` and the information hook returns `FirstAttack=true` throughout:

| subtest | given | claimant | expect emitted | expect `ControllerHasAggro()` | expect `ControlCharacterId()` | expect `AggroRefreshedMs()` |
|---|---|---|---|---|---|---|
| `controller without aggro flips and emits once` | `ControlMonster(tm, uid, 7)` | 7 | 1 | `true` | 7 | `nowMs` (5000) |
| `controller with aggro stamps lease, emits nothing` | `ControlMonsterWithAggro(tm, uid, 7, 1000)` | 7 | 0 | `true` | 7 | `nowMs` (5000) |
| `non-controller takes control with aggro` | `ControlMonster(tm, uid, 7)` | 9 | 2 (STOP_CONTROL then START_CONTROL) | `true` | 9 | `nowMs` (5000) |
| `non-controller loses to an existing aggro holder` | `ControlMonsterWithAggro(tm, uid, 7, 1000)` | 9 | 0 | `true` | 7 | `1000` (unchanged) |
| `uncontrolled monster is claimed` | no controller (`newAggroedMonster(..., 0, nil)`) | 9 | 1 (START_CONTROL only) | `true` | 9 | `nowMs` (5000) |
| `GM-hidden claimant is dropped` | `ControlMonster(tm, uid, 7)`, `hidden = {9:{}}` | 9 | 0 | `false` | 7 | `0` |

`TestSetAggro_LeavesDamageEntriesUntouched` (FR-4.5): seed `newAggroedMonster(t, ctx, tm, 7, []uint32{8})`, call `p.SetAggro(uid, 7)`, then assert `len(got.DamageEntries()) == 1` and `got.DamageEntries()[0].CharacterId == 8`.

- [ ] **Step 2: Run to verify failure**

```
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestSetAggro'
```

Expected: FAIL to compile — `p.SetAggro undefined`, `p.nowFn undefined`.

- [ ] **Step 3: Add the clock seam**

In `processor.go`, add `nowFn func() int64` to `ProcessorImpl` and, in `NewProcessor`, `p.nowFn = func() int64 { return time.Now().UnixMilli() }`. Replace the inline `time.Now().UnixMilli()` Task 7 left at the `ControlMonsterWithAggro` call in `startControl` with `p.nowFn()`.

- [ ] **Step 4: Implement `SetAggro`**

Add `SetAggro(uniqueId uint32, characterId uint32) error` to the `Processor` interface (in the `// Commands` block, after `ClearAggro`), and the implementation next to `ForceControl`. Every rejection is a logged drop returning `nil`, for the same reason `ForceControl` does it — a stale client-driven target must not wedge the consumer.

```go
// SetAggro grants auto-aggro on a client AUTO_AGGRO claim (design §2 hybrid).
// The channel is not the authority: every gate below runs here, in the service
// that owns the monster registry.
//
// Gates, in order — each a Debugf drop naming the failing gate:
//  1. the monster exists;
//  2. the monster is alive;
//  3. the template is aggressive (firstAttack). CMob::ApplyControl also fires
//     for bPickUpDrop-only templates, so this gate is what keeps drop-picking
//     mobs passive. A lookup error or cache miss DENIES (FR-5.2);
//  4. the claimant is in the monster's field;
//  5. arbitration:
//       claimant is the controller  -> stamp the lease; flip + emit
//                                      AGGRO_CHANGED only if it was not set;
//       someone else holds aggro    -> drop (anti-thrash, design §2);
//       otherwise                   -> startControl(..., forceAggro=true),
//                                      which emits START_CONTROL with
//                                      ControllerHasAggro true and excludes
//                                      GM-hidden claimants.
//
// No damage entry is written on any path: auto-aggro confers no drop ownership
// and no kill credit (FR-4.5).
func (p *ProcessorImpl) SetAggro(uniqueId uint32, characterId uint32) error
```

Implementation notes the implementer must follow exactly:

- gate 1 uses `p.GetById(uniqueId)`; on error log `"SET_AGGRO for monster [%d]: monster no longer exists; dropping."` and return `nil`.
- gate 2 is `if !m.Alive()`.
- gate 3 uses the existing package hook so tests need no HTTP:
  `if testInformationLookup != nil { info, err = testInformationLookup(m.MonsterId()) } else { info, err = information.NewProcessor(p.l, p.ctx).GetById(m.MonsterId()) }` — copy that exact branch from `processor.go:979`. `err != nil` **or** `!info.FirstAttack()` is a drop.
- gate 4 uses `p.inFieldFn(m.Field())` and the same membership loop `ForceControl` uses; an `inFieldFn` error is a drop.
- gate 5's controller branch calls `GetMonsterRegistry().SetAggro(p.t, uniqueId, characterId, p.nowFn())` and emits `aggroChangedStatusEventProvider(summary.Monster, characterId, true)` on `EnvEventTopicMonsterStatus` **only** when `summary.Changed`.
- gate 5's transfer branch checks `p.hiddenFn()` for the claimant exactly as `ForceControl` does (dropping a hidden claimant), then calls `p.startControl(uniqueId, characterId, true)` and logs a warning on error.

- [ ] **Step 5: Run the tests**

```
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/
```

Expected: PASS, including `force_control_test.go` and `clear_aggro_test.go`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/set_aggro_test.go
git commit -m "feat(atlas-monsters): implement server-authoritative SetAggro"
```

---

## Task 10: `SET_AGGRO` on `COMMAND_TOPIC_MONSTER` — consumer side

### Files

- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go` — add `CommandTypeSetAggro` and `setAggroCommandBody`
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go` — add `handleSetAggroCommand` and its `PersistentConfig` registration
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go` — add the unmarshal test
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go:218` — read-only; `handleForceControlCommand` is the exact shape to mirror

Module root: `services/atlas-monsters/atlas.com/monsters`.

**Interfaces consumed:** `monster.Processor.SetAggro` (Task 9).

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go` (package `monster`; copy the file's existing JSON-round-trip style).

`TestSetAggroCommandUnmarshal`: unmarshal the literal

```json
{"worldId":1,"channelId":2,"mapId":100000000,"instance":"00000000-0000-0000-0000-000000000000","monsterId":4242,"type":"SET_AGGRO","body":{"characterId":777}}
```

into `command[setAggroCommandBody]` and assert `Type == CommandTypeSetAggro`, `MonsterId == 4242`, `Body.CharacterId == 777`.

`TestSetAggroCommandTypeConstant`: `CommandTypeSetAggro == "SET_AGGRO"`.

- [ ] **Step 2: Run to verify failure**

```
cd services/atlas-monsters/atlas.com/monsters && go test ./kafka/consumer/monster/ -run SetAggro
```

Expected: FAIL to compile — `undefined: setAggroCommandBody`.

- [ ] **Step 3: Implement**

`kafka.go`: add `CommandTypeSetAggro = "SET_AGGRO"` to the const block, immediately after `CommandTypeForceControl`, and:

```go
// setAggroCommandBody asks the processor to grant auto-aggro on a client
// AUTO_AGGRO claim. characterId is the claimant; every gate is applied by the
// processor, never by the channel. Mirrors atlas-channel's
// monster2.SetAggroCommandBody — edit both together. `characterId uint32`
// already appears with that name and type in sibling bodies, so it introduces
// no unmarshal collision on this shared, fan-to-every-handler topic.
type setAggroCommandBody struct {
    CharacterId uint32 `json:"characterId"`
}
```

`consumer.go`: register in `InitHandlers` immediately after the `handleForceControlCommand` registration, and add the handler after `handleForceControlCommand`:

```go
func handleSetAggroCommand(l logrus.FieldLogger, ctx context.Context, c command[setAggroCommandBody]) {
    if c.Type != CommandTypeSetAggro {
        return
    }

    p := monster.NewProcessor(l, ctx)
    if err := p.SetAggro(c.MonsterId, c.Body.CharacterId); err != nil {
        l.WithError(err).Errorf("SET_AGGRO failed for monster [%d] character [%d].", c.MonsterId, c.Body.CharacterId)
    }
}
```

- [ ] **Step 4: Run the tests**

```
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/
git commit -m "feat(atlas-monsters): consume SET_AGGRO on COMMAND_TOPIC_MONSTER"
```

---

## Task 11: `SET_AGGRO` producer + `ControlCharacterId` on the live mirror (channel)

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go` — add `CommandTypeSetAggro` + `SetAggroCommandBody`
- `services/atlas-channel/atlas.com/channel/monster/producer.go` — add `SetAggroCommandProvider`
- `services/atlas-channel/atlas.com/channel/monster/processor.go` — add `SetAggro` to the `Processor` interface and impl
- `services/atlas-channel/atlas.com/channel/monster/live_mirror.go` — add `ControlCharacterId` to `LiveEntry`, populate it in `LiveEntryFromModel`, add `UpdateControl`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:320` — maintain `ControlCharacterId` on START_CONTROL / STOP_CONTROL
- `services/atlas-channel/atlas.com/channel/monster/producer_magnet_test.go` — add the provider-shape test
- `services/atlas-channel/atlas.com/channel/monster/live_mirror_test.go` — add the mirror test

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces produced (Task 12 consumes):**
`monster.Processor.SetAggro(f field.Model, monsterId uint32, characterId uint32) error`;
`monster.LiveEntry.ControlCharacterId uint32`;
`(*monster.LiveMirror).UpdateControl(t tenant.Model, uniqueId uint32, controllerId uint32)`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/monster/producer_magnet_test.go`, copying `TestForceControlCommandProviderShape` verbatim in structure:

`TestSetAggroCommandProviderShape` — `SetAggroCommandProvider(magnetTestField(), 4242, 777)()` produces exactly 1 message which unmarshals into `monster2.Command[monster2.SetAggroCommandBody]` with `Type == monster2.CommandTypeSetAggro`, `MonsterId == 4242`, `Body.CharacterId == 777`, and (raw-map check) `body` equal to `{"characterId":777}`.

Append to `services/atlas-channel/atlas.com/channel/monster/live_mirror_test.go` (copy the file's existing tenant/entry setup):

`TestLiveMirrorUpdateControl` — table-driven, three cases against a mirror seeded with `Put(t, 42, LiveEntry{ControlCharacterId: 7})`:

| subtest | call | expect `Lookup(t, 42).ControlCharacterId` | expect `ok` |
|---|---|---|---|
| `updates an existing entry` | `UpdateControl(t, 42, 9)` | `9` | `true` |
| `clears on handover to nobody` | `UpdateControl(t, 42, 0)` | `0` | `true` |
| `absent entry is not created` | `UpdateControl(t, 99, 9)` then `Lookup(t, 99)` | — | `false` |

The third case pins the same invariant `UpdateMp`/`UpdateAggro` carry: events must never create entries, because the envelope cannot supply the full projection.

- [ ] **Step 2: Run to verify failure**

```
cd services/atlas-channel/atlas.com/channel && go test ./monster/ -run 'SetAggroCommandProviderShape|LiveMirrorUpdateControl'
```

Expected: FAIL to compile — `undefined: SetAggroCommandProvider`, `undefined: UpdateControl`.

- [ ] **Step 3: Implement the command + producer + processor method**

`kafka/message/monster/kafka.go`: add `CommandTypeSetAggro = "SET_AGGRO"` after `CommandTypeForceControl`, and

```go
// SetAggroCommandBody asks atlas-monsters to grant auto-aggro on a client
// AUTO_AGGRO claim. characterId is the claimant. The proximity score the
// packet carries is deliberately NOT on this command: it is a channel-side
// admission criterion, and atlas-monsters makes no decision from it.
//
// Mirrors atlas-monsters' setAggroCommandBody — edit both together.
type SetAggroCommandBody struct {
    CharacterId uint32 `json:"characterId"`
}
```

`monster/producer.go`: add `SetAggroCommandProvider(f field.Model, monsterId uint32, characterId uint32) model.Provider[[]kafka.Message]` copying `ForceControlCommandProvider` exactly, substituting the type and body.

`monster/processor.go`: add `SetAggro(f field.Model, monsterId uint32, characterId uint32) error` to the `Processor` interface after `ForceControl`, and the impl:

```go
// SetAggro asks atlas-monsters to grant auto-aggro on a client AUTO_AGGRO
// claim. The channel applies only cheap local admission checks; every
// authoritative gate runs in atlas-monsters.
func (p *ProcessorImpl) SetAggro(f field.Model, monsterId uint32, characterId uint32) error {
    p.l.Debugf("Requesting auto-aggro of monster [%d] for character [%d].", monsterId, characterId)
    return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(SetAggroCommandProvider(f, monsterId, characterId))
}
```

- [ ] **Step 4: Implement the mirror change**

`monster/live_mirror.go`: add `ControlCharacterId uint32` to `LiveEntry` (after `ControllerHasAggro`), set `ControlCharacterId: mo.ControlCharacterId()` in `LiveEntryFromModel`, and add `UpdateControl` copying `UpdateAggro` field-for-field with the doc comment:

```go
// UpdateControl sets the entry's current controller. Update only — see
// UpdateMp for why events never create entries. This is an optimisation for
// the auto-aggro rate gate (it lets the channel prefer "am I the controller"
// without a REST call); it is never an authority.
```

`kafka/consumer/monster/consumer.go`: in `handleStatusEventStartControl`, next to the existing `UpdateAggro` call, add `monster.GetLiveMirror().UpdateControl(tenant.MustFromContext(ctx), e.UniqueId, e.Body.ActorId)`. In `handleStatusEventStopControl`, next to its `UpdateAggro(..., false)` call, add `monster.GetLiveMirror().UpdateControl(tenant.MustFromContext(ctx), e.UniqueId, 0)`.

- [ ] **Step 5: Run the tests**

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./monster/ ./kafka/consumer/monster/
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go services/atlas-channel/atlas.com/channel/monster/ services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go
git commit -m "feat(atlas-channel): add the SET_AGGRO command and mirror the current controller"
```

---

## Task 12: Auto-aggro rate gate (channel)

### Files

- `services/atlas-channel/atlas.com/channel/monster/auto_aggro_gate.go` — **new file**; the constants and the gate
- `services/atlas-channel/atlas.com/channel/monster/auto_aggro_gate_test.go` — **new file**; the gate tests
- `services/atlas-channel/atlas.com/channel/monster/live_mirror.go` — read-only; the sweeper shape (`sweepLoop` / `SweepStale`) this gate copies
- `libs/atlas-constants/` — read-only; checked for an existing proximity/interval constant (there is none; these are behavioural dials)

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces produced (Task 13 consumes):**

```go
const AutoAggroProximityThreshold = uint32(40)
const AutoAggroRefreshInterval = 5 * time.Second
const autoAggroMinInterval = 1 * time.Second
func GetAutoAggroGate() *AutoAggroGate
func (g *AutoAggroGate) Admit(t tenant.Model, characterId uint32, mobId uint32, aggroed bool, now time.Time) bool
func (g *AutoAggroGate) SweepStale(now time.Time, maxAge time.Duration) int
func (g *AutoAggroGate) EvictTenant(tid uuid.UUID)
```

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/monster/auto_aggro_gate_test.go`, package `monster`. Build a bare gate directly (`&AutoAggroGate{perTenant: map[uuid.UUID]map[autoAggroKey]time.Time{}}`) so the singleton's sweeper goroutine is not involved; use a fixed `base := time.Unix(1_700_000_000, 0)`.

`TestAutoAggroGateAdmit` — table-driven; each case runs a sequence of `Admit` calls on one gate and asserts the returned bool of each:

| subtest | sequence (`Admit(t, char, mob, aggroed, at)`) | expected results |
|---|---|---|
| `first claim admits` | `(7, 42, false, base)` | `true` |
| `unaggroed repeat inside 1s is blocked` | `(7,42,false,base)`, `(7,42,false,base+900ms)` | `true`, `false` |
| `unaggroed repeat after 1s admits` | `(7,42,false,base)`, `(7,42,false,base+1100ms)` | `true`, `true` |
| `aggroed refresh inside 5s is blocked` | `(7,42,false,base)`, `(7,42,true,base+2s)` | `true`, `false` |
| `aggroed refresh after 5s admits` | `(7,42,false,base)`, `(7,42,true,base+6s)` | `true`, `true` |
| `different mob is independent` | `(7,42,false,base)`, `(7,43,false,base)` | `true`, `true` |
| `different character is independent` | `(7,42,false,base)`, `(8,42,false,base)` | `true`, `true` |

`TestAutoAggroGateTenantIsolation`: two distinct tenants (`tenant.Create(uuid.New(), "GMS", 83, 1)` twice) admitting the same `(character, mob)` at `base` both return `true`.

`TestAutoAggroGateSweepStale`: after `Admit(t, 7, 42, false, base)`, `SweepStale(base.Add(31*time.Minute), 30*time.Minute)` returns `1`, and a subsequent `Admit(t, 7, 42, false, base.Add(31*time.Minute))` returns `true`.

`TestAutoAggroGateEvictTenant`: after `Admit`, `EvictTenant(tm.Id())` then `Admit(..., base.Add(100*time.Millisecond))` returns `true`.

`TestAutoAggroConstants`: `AutoAggroProximityThreshold == 40`, `AutoAggroRefreshInterval == 5*time.Second`.

- [ ] **Step 2: Run to verify failure**

```
cd services/atlas-channel/atlas.com/channel && go test ./monster/ -run AutoAggroGate
```

Expected: FAIL to compile — `undefined: AutoAggroGate`.

- [ ] **Step 3: Implement**

Create `services/atlas-channel/atlas.com/channel/monster/auto_aggro_gate.go`, package `monster`. Model it on `live_mirror.go`: a `sync.Once` singleton with a `sync.RWMutex`-guarded `map[uuid.UUID]map[autoAggroKey]time.Time`, and a `sweepLoop` started once in `GetAutoAggroGate` carrying the same `//goroutine-guard:allow` comment shape `GetLiveMirror` uses (`tools/goroutine-guard.sh` requires the annotation).

```go
// AUTO_AGGRO arrives at most once per second per mob per client, from EVERY
// client that can see the mob (CMob::ApplyControl has no controller test), so
// a dense aggressive map fans in at N-clients × M-mobs per second. This gate is
// the only thing between that and a Kafka storm (design §6.3, §12).
const (
    // AutoAggroProximityThreshold is the client's own chase bar: CMob::TryFirstAttack
    // chases at |dx|/10 + |dy|/3 <= 40 (v95 @0x6482f0). A claim scoring above it is
    // dropped, which is also what lets the atlas-monsters lease expire when a player
    // walks away — the client keeps sending, but with a score out of range.
    AutoAggroProximityThreshold = uint32(40)

    // AutoAggroRefreshInterval throttles lease refreshes for an already-aggro'd mob.
    // atlas-monsters' AutoAggroLeaseTtlMs (15s) tolerates two missed refreshes.
    AutoAggroRefreshInterval = 5 * time.Second

    // autoAggroMinInterval floors the not-yet-aggro'd path. The stock client already
    // self-throttles to 1s; this guards a modified one.
    autoAggroMinInterval = 1 * time.Second

    autoAggroSweepInterval = 5 * time.Minute
    autoAggroMaxEntryAge   = 30 * time.Minute
)

type autoAggroKey struct {
    characterId uint32
    mobId       uint32
}
```

`Admit` takes the write lock, picks `interval := autoAggroMinInterval` or `AutoAggroRefreshInterval` when `aggroed`, returns `false` (without touching the stamp) if a stamp exists and `now.Sub(stamp) < interval`, otherwise stores `now` and returns `true`. `SweepStale` and `EvictTenant` copy `LiveMirror`'s implementations.

Register the gate with the same tenant-drain evictor the mirror uses, if `listener.RegisterEvictor` is wired for the mirror in `main.go`; otherwise the staleness sweep alone is sufficient and no wiring is added.

- [ ] **Step 4: Run the tests**

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./monster/ && tools/goroutine-guard.sh
```

Expected: PASS; the goroutine guard exits 0.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monster/auto_aggro_gate.go services/atlas-channel/atlas.com/channel/monster/auto_aggro_gate_test.go
git commit -m "feat(atlas-channel): add the auto-aggro rate gate"
```

---

## Task 13: `AutoAggroHandleFunc` + registration

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/auto_aggro.go` — **new file**; the handler
- `services/atlas-channel/atlas.com/channel/socket/handler/auto_aggro_test.go` — **new file**; decode + admission tests
- `services/atlas-channel/atlas.com/channel/main.go:913` — register `handlerMap[monstersb.AutoAggroHandle]`
- `services/atlas-channel/atlas.com/channel/socket/handler/monster_catch_item_use.go` — read-only; the acting-handler shape
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:416` — read-only; the `var xFn = func(...)` seam pattern the test injects through

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces consumed:** `serverbound.AutoAggro` (Task 1), `monster.GetAutoAggroGate` / `AutoAggroProximityThreshold` (Task 12), `monster.Processor.SetAggro` / `LiveEntry.ControlCharacterId` (Task 11).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/auto_aggro_test.go`, package `handler`.

`TestAutoAggroDecode` — copy `monster_catch_item_use_test.go` verbatim in structure: encode `monstersb.NewAutoAggro(0x07654321, 39)` under `pt.CreateContext("GMS", 83, 1)`, wrap in `request.Request` / `request.NewRequestReader`, decode into a fresh `monstersb.AutoAggro`, and assert `MobId() == 0x07654321`, `Distance() == 39`, `Operation() == monstersb.AutoAggroHandle`.

`TestAutoAggroAdmission` — table-driven over the handler's admission decision, driven through the package-level seams the handler defines (`autoAggroMirrorLookupFn`, `autoAggroEmitFn`), each saved and restored with `defer`. Each case builds the packet, runs the handler against a `session.Model` for character `7` in field `world 1 / channel 2 / map 100000000`, and asserts whether `autoAggroEmitFn` was called and with what:

| subtest | packet `distance` | mirror entry | expect emit | expect emitted `(monsterId, characterId)` |
|---|---|---|---|---|
| `valid claim forwards` | `39` | present, `Field` = session field, `ControllerHasAggro` false | yes | `(0x07654321, 7)` |
| `score above threshold is dropped` | `41` | present, same field | no | — |
| `score at threshold forwards` | `40` | present, same field | yes | `(0x07654321, 7)` |
| `mob absent from mirror is dropped` | `39` | absent | no | — |
| `mob in another field is dropped` | `39` | present, `Field` = map `104000000` | no | — |
| `rate gate closed is dropped` | `39` | present, same field | no (second call in the same instant) | — |

The `rate gate closed` case runs the handler twice back-to-back with the same session and packet, asserting exactly **one** emit across both calls.

Reset the gate between cases with `monster.GetAutoAggroGate().EvictTenant(<tenant id>)`.

- [ ] **Step 2: Run to verify failure**

```
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run AutoAggro
```

Expected: FAIL to compile — `undefined: AutoAggroHandleFunc`.

- [ ] **Step 3: Implement the handler**

Create `services/atlas-channel/atlas.com/channel/socket/handler/auto_aggro.go`, package `handler`, with two package-level seams above the handler (the pattern `consumer.go:416` establishes):

```go
// autoAggroMirrorLookupFn resolves the live-mirror entry for a mob. Injected so
// the handler's admission logic is testable without a populated singleton.
var autoAggroMirrorLookupFn = func(t tenant.Model, uniqueId uint32) (monster.LiveEntry, bool) {
    return monster.GetLiveMirror().Lookup(t, uniqueId)
}

// autoAggroEmitFn forwards an admitted claim as SET_AGGRO.
var autoAggroEmitFn = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32, characterId uint32) error {
    return monster.NewProcessor(l, ctx).SetAggro(f, monsterId, characterId)
}
```

```go
// AutoAggroHandleFunc decodes AUTO_AGGRO (CMob::ApplyControl) and forwards it as
// SET_AGGRO. Every check here is cheap and local — the session's character and
// field, the client's own proximity score, the live mirror, and the rate gate.
// None of them is the authority: atlas-monsters owns the monster registry and
// re-applies every gate (FR-3.3). A rejected claim is a debug-logged drop with
// no response packet; AUTO_AGGRO has no client-visible failure path (FR-3.4).
func AutoAggroHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
```

Body order, each failing branch a `Debugf` naming the failing check plus the mob id and character id, then `return`:

1. decode into `var p monstersb.AutoAggro`; `l.Debugf("[%s] read [%s]", p.Operation(), p.String())`;
2. `if s.CharacterId() == 0` → drop;
3. `if p.Distance() > monster.AutoAggroProximityThreshold` → drop;
4. `e, ok := autoAggroMirrorLookupFn(tenant.MustFromContext(ctx), p.MobId())`; `if !ok` → drop;
5. `if e.Field.Id() != s.Field().Id()` → drop;
6. `if !monster.GetAutoAggroGate().Admit(tenant.MustFromContext(ctx), s.CharacterId(), p.MobId(), e.ControllerHasAggro, time.Now())` → drop;
7. `_ = autoAggroEmitFn(l, ctx, s.Field(), p.MobId(), s.CharacterId())`.

`main.go`: add, immediately after line 913's `MobDropPickupRequestHandle` registration:

```go
handlerMap[monstersb.AutoAggroHandle] = handler.AutoAggroHandleFunc
```

- [ ] **Step 4: Run the tests**

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/ ./monster/
tools/template-symbol-check.sh
```

Expected: tests PASS; `template-symbol-check.sh` exits 0 (the ten `"handler": "AutoAggro"` routes now resolve to a registered symbol).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/auto_aggro.go services/atlas-channel/atlas.com/channel/socket/handler/auto_aggro_test.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(atlas-channel): handle AUTO_AGGRO and emit SET_AGGRO"
```

---

## Task 14: Lifecycle regression tests (FR-6.1, FR-6.2)

`handleStatusEventAggroChanged` and `StatusEventStartControlBody.ControllerHasAggro` already ship the re-issue and the through-handover behaviour (design §0, §9). This task pins them so the auto-aggro path cannot regress them silently — tests only, no behaviour change.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/aggro_lifecycle_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:384` — read-only; `handleStatusEventAggroChanged`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:320` — read-only; `handleStatusEventStartControl`
- `services/atlas-channel/atlas.com/channel/socket/writer/` — read-only; `StartControlMonsterBody(m, aggro)`

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the tests**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/aggro_lifecycle_test.go`, package `monster`. Drive both handlers through the existing package seams (`monsterGetByIdFn`, `controlGrantFn`, `announceFn` at `consumer.go:416`), each saved and restored with `defer`.

`TestAggroChangedReissuesControlWithAggroSet` — table-driven, two cases. Each stubs `monsterGetByIdFn` to return a built monster (`monster.NewModelBuilder(uniqueId, f, templateId).SetControlCharacterId(7).MustBuild()`), captures the announce through `announceFn`, and invokes `handleStatusEventAggroChanged(sc, wp)` with a `StatusEvent[StatusEventAggroChangedBody]` whose `Type` is `monster2.EventStatusAggroChanged`:

| subtest | `Body.ControllerHasAggro` | expect announced writer | expect the `aggro` argument threaded into `StartControlMonsterBody` |
|---|---|---|---|
| `aggro on` | `true` | `monsterpkt.MonsterControlWriter` | `true` |
| `aggro off` | `false` | `monsterpkt.MonsterControlWriter` | `false` |

Assert the writer name through the `announceFn` seam's `writerName` argument, and the aggro value by capturing it from the `StatusEventAggroChangedBody` the handler passes on — mirror however the existing consumer tests in this package assert an announce; if none does, assert the mirror side effect instead: after the handler runs, `monster.GetLiveMirror().Lookup(t, uniqueId).ControllerHasAggro` equals the body's value (the handler's first action).

`TestStartControlCarriesAggroThroughHandover` (FR-6.2) — stub `controlGrantFn` to record its `aggro` and `characterId` arguments, then invoke `handleStatusEventStartControl(sc, wp)` with `Body.ActorId = 9`, `Body.ControllerHasAggro = true`. Assert the recorded `aggro` is `true` and `characterId` is `9` — a mob that is aggro'd stays aggro'd through a controller change; the flag is populated truthfully, not reset.

`TestClearAggroOrderingUnchanged` is **not** added here — the Monster Magnet ordering invariant already has a test; Step 2 only re-runs it.

- [ ] **Step 2: Run the tests, including the pre-existing magnet ordering test**

```
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/monster/ ./monster/
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/
```

Expected: PASS, including `producer_magnet_test.go` and `clear_aggro_test.go`.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/monster/aggro_lifecycle_test.go
git commit -m "test(atlas-channel): pin control re-issue and aggro-through-handover"
```

---

## Task 15: Coverage manifest and deploy notes

### Files

- `docs/tasks/task-255-auto-aggro-mobs/coverage-manifest.yaml` — **new file**; input to `packet-completeness-critic`
- `docs/tasks/task-255-auto-aggro-mobs/deploy-notes.md` — **new file**; the live-tenant PATCH table
- `docs/packets/IMPLEMENTING_A_PACKET.md` — read-only; "After it merges — roll out to live tenants"

- [ ] **Step 1: Write the coverage manifest**

Create `docs/tasks/task-255-auto-aggro-mobs/coverage-manifest.yaml` declaring the ten `AUTO_AGGRO` op×version cells and **nothing else** — the critic flags CLAIMED-BUT-UNVERIFIED and CHANGED-BUT-UNCLAIMED against exactly this list:

```yaml
task: task-255-auto-aggro-mobs
packets:
  - packet: monster/serverbound/MonsterAutoAggro
    op: AUTO_AGGRO
    direction: serverbound
    fname: CMob::ApplyControl
    versions:
      - gms_v48
      - gms_v61
      - gms_v72
      - gms_v79
      - gms_v83
      - gms_v84
      - gms_v87
      - gms_v92
      - gms_v95
      - jms_v185
```

If `packet-completeness-critic` reports the schema differs from what it reads, match the schema an existing task's `coverage-manifest.yaml` uses (`git ls-files 'docs/tasks/*/coverage-manifest.yaml'`) rather than inventing one.

- [ ] **Step 2: Write the deploy notes**

Create `docs/tasks/task-255-auto-aggro-mobs/deploy-notes.md` stating that seed templates apply only at tenant **creation**, so each live tenant's config must be PATCHed with the new `socket.handlers` entry and `atlas-channel` restarted (the handler map is built once at startup). Include the full per-version PATCH table:

| version key | template | opCode | validator | handler | fname |
|---|---|---|---|---|---|
| gms_v48 | `template_gms_48_1.json` | `0x82` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v61 | `template_gms_61_1.json` | `0x9C` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v72 | `template_gms_72_1.json` | `0xB3` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v79 | `template_gms_79_1.json` | `0xB5` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v83 | `template_gms_83_1.json` | `0xBD` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v84 | `template_gms_84_1.json` | `0xC2` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v87 | `template_gms_87_1.json` | `0xC9` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v92 | `template_gms_92_1.json` | `0xDD` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| gms_v95 | `template_gms_95_1.json` | `0xE4` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |
| jms_v185 | `template_jms_185_1.json` | `0xC3` | `LoggedInValidator` | `AutoAggro` | `CMob::ApplyControl` |

Add the post-deploy checks from the playbook: `grep "Unable to locate validator"` == 0; no new error/fatal logs; no `unhandled message op 0xXX` for the routed opcodes. Note the tuning dials (`AutoAggroProximityThreshold` = 40, `AutoAggroRefreshInterval` = 5s, `AutoAggroLeaseTtlMs` = 15s) and design §12's guidance that the threshold must not be raised past 100.

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-255-auto-aggro-mobs/coverage-manifest.yaml docs/tasks/task-255-auto-aggro-mobs/deploy-notes.md
git commit -m "docs(task-255): coverage manifest and live-tenant deploy notes"
```

---

## Final gate (controller, not an implementer task)

- [ ] Flagless `tools/verify.sh` exits 0 (dispatch `atlas-verifier`; only the flagless invocation counts).
- [ ] `backend-guidelines-reviewer` over the changed Go packages in `atlas-channel`, `atlas-monsters`, and `libs/atlas-packet`.
- [ ] `plan-adherence-reviewer` over this plan.
- [ ] `packet-completeness-critic` against `coverage-manifest.yaml` — no CHANGED-BUT-UNCLAIMED, no CLAIMED-BUT-UNVERIFIED.
- [ ] Manual live-channel verification per design §10: Jr. Necki turns and attacks unprovoked; Ribbon Pig unchanged; walking away goes passive within ~15s; a Dark Sight Rogue is not attacked (if it is, design §5's recorded escalation is a channel-side buff gate).
