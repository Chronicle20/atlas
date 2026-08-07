# Mist Broadcast Writer Wiring — Design

Task: task-165-mist-writer-template-wiring
Status: Approved PRD → design
Created: 2026-07-10
Revised: 2026-08-07 — rescoped to all supported versions (PRD v2)

---

## 1. Problem recap (one paragraph)

The `AffectedAreaCreated` / `AffectedAreaRemoved` clientbound writers are fully
implemented in `libs/atlas-packet/field/clientbound/`, and the atlas-channel
mist consumer already emits them — but the name→opcode resolution chain is
broken in two independent places: `produceWriters()` in
`services/atlas-channel/atlas.com/channel/main.go` never declares either name
(so `BuildWriterProducer` filters them out of the writer map), and **no** seed
template — none of the eleven — registers them under `socket.writers`. Every
mist broadcast dies in `session.Announce` with "writer not found". This design
covers the two wiring fixes across every supported version, the coverage-matrix
completion for both mist ops, opcode discovery for the three templates that
lack a sourced opcode, live-tenant rollout, and end-to-end verification. No
codec, consumer, or producer changes on any already-verified version.

## 2. Architecture context (what already works)

Data flow, verified against source:

```
atlas-maps mist domain ──┐
atlas-monsters (mob AoE) ─┴─> EVENT_TOPIC_MIST (created/destroyed)
        │
        v
atlas-channel kafka/consumer/mist/consumer.go
        │  handleMistCreated  → fieldpkt.NewAffectedAreaCreated(...)
        │  handleMistDestroyed → fieldpkt.NewAffectedAreaRemoved(...)
        v
_map.ForSessionsInMap(f, session.Announce(...)(wp)(WriterName)(body.Encode))
        │
        v  wp lookup: WriterName → opcode   ← BROKEN LINK (both sides)
client socket
```

The only broken link is the `wp` lookup. `BuildWriterProducer`
(`libs/atlas-opcodes/producer.go`) intersects two inputs:

1. the service's declared writer list (`produceWriters()` — Go, name only), and
2. the tenant's `socket.writers` config (name → opCode, from seed template at
   tenant creation).

A name absent from either side is silently dropped (startup warning only).
Both sides are missing here, on every version. This is the known
`routedElsewhere && !routed` template-wiring gap family.

## 3. The version surface

Eleven seed templates exist; `packet-audit` tracks nine of them as matrix
columns (`tools/packet-audit/internal/matrix/model.go:14`). gms_92 and gms_12
are template-only — they route packets but have no registry and no column.

State as of 2026-08-07, all verified against source:

| Version | Template | Writers in template | SPAWN op | REMOVE op | Matrix SPAWN | Matrix REMOVE |
|---|---|---|---|---|---|---|
| gms_12 | ✔ | 42 | **unknown** | **unknown** | no column | no column |
| gms_48 | ✔ | 62 | **unknown** | **unknown** | ⬜ | ⬜ |
| gms_61 | ✔ | 153 | 0x0D2 | 0x0D3 | ✅ | 🟡ᶠ |
| gms_72 | ✔ | 160 | 0x0F3 | 0x0F4 | ✅ | 🟡ᶠ |
| gms_79 | ✔ | 196 | 0x0FB | 0x0FC | ✅ *(soft)* | 🟡ᶠ |
| gms_83 | ✔ | 209 | 0x111 | 0x112 | ❌ | ✅ |
| gms_84 | ✔ | 209 | 0x118 | 0x119 | ❌ | ✅ |
| gms_87 | ✔ | 207 | 0x122 | 0x123 | ❌ | ✅ |
| gms_92 | ✔ | 65 | **unknown** | **unknown** | no column | no column |
| gms_95 | ✔ | 215 | 0x148 | 0x149 | ❌ | ✅ |
| jms_185 | ✔ | 199 | 0x126 | 0x127 | ❌ | ✅ |

This splits the work into two tiers: **Tier A** (eight versions, opcode known —
pure wiring plus fixture work) and **Tier B** (three versions, opcode unknown —
discovery gated).

## 4. Design decisions

### 4.1 FR-1 — Go writer declaration (no alternatives worth taking)

Add two entries to the `produceWriters()` slice
(`services/atlas-channel/atlas.com/channel/main.go:608`), after
`fieldcb.KiteDestroyWriter` at line 718:

- `fieldcb.AffectedAreaCreatedWriter` ("AffectedAreaCreated")
- `fieldcb.AffectedAreaRemovedWriter` ("AffectedAreaRemoved")

No other Go change — the consumer already references the constants, so there is
no import or call-site work. This is version-independent: one declaration
serves every tenant, and it is the same two lines whether Tier B resolves or
not.

Accepted side effect: tenants on templates that remain unwired gain one more
line each in the existing startup warning enumerating declared-but-unmapped
writers. Same posture as many other writers; no mitigation needed.

### 4.2 FR-2 — Tier A template wiring (eight versions)

Add two `{"opCode": "0xNNN", "writer": "<name>"}` entries (no `options` — the
`Encode` closures ignore the options map) to `socket.writers` in each Tier A
template. Opcodes come from `docs/packets/registry/<ver>.yaml` and agree with
`docs/packets/audits/STATUS.md:346-347`.

**The insertion point is structurally uniform**, which makes this mechanical
rather than eight separate judgement calls. In all eight templates the field
pool band is laid out identically relative to SPAWN_MIST (call it `M`):
`DropDestroy` at `M−4`, nothing at `M`/`M+1` (the two gaps this task fills),
`SpawnDoor` at `M+2`, `RemoveDoor` at `M+3`. So in every file both new entries
slot immediately **before the `SpawnDoor` entry**:

| Template | Created | Removed | SpawnDoor anchor |
|---|---|---|---|
| `template_gms_61_1.json` | 0x0D2 | 0x0D3 | 0x0D4 |
| `template_gms_72_1.json` | 0x0F3 | 0x0F4 | 0x0F5 |
| `template_gms_79_1.json` | 0x0FB | 0x0FC | 0x0FD |
| `template_gms_83_1.json` | 0x111 | 0x112 | 0x113 |
| `template_gms_84_1.json` | 0x118 | 0x119 | 0x11A |
| `template_gms_87_1.json` | 0x122 | 0x123 | 0x124 |
| `template_gms_95_1.json` | 0x148 | 0x149 | 0x14A |
| `template_jms_185_1.json` | 0x126 | 0x127 | 0x128 |

Ascending opcode order is a hard rule
(`docs/packets/TEMPLATE_CONVENTIONS.md`) enforced by
`tools/template-opcode-order-guard.sh` in CI, so the guard — not eyeballing —
is the acceptance check for placement.

### 4.3 FR-2b — Tier B: discovery, and what happens when it fails

The v1 PRD declared gms_92/gms_12 out of scope because they had no IDB. That is
no longer true: both are packet-named now. And v48's ⬜ turns out to be an
artifact, not a ruling — `docs/packets/ida-exports/gms_v48.json` carries
`{"unresolved": true, "comment": "function not found in IDB"}` for both mist
functions, and the v48 registry describes itself as a "clean PARTIAL" with deep
pool leaves explicitly deferred. Nothing anywhere asserts the feature is absent.

**Approaches considered for Tier B:**

- **A (chosen): bounded per-version discovery with three declared outcomes.**
  Walk each version's IDB for the affected-area pool dispatcher, read the two
  opcodes, name the functions. Resolve each version to Outcome A (found → wire
  + verify), B (proven absent → n-a evidence, leave unwired), or C
  (inconclusive → leave unwired, record what was walked). The task completes
  with any mix of outcomes; Tier A does not wait on Tier B.
- **B (rejected): derive the opcode band by interpolation.** This is the
  task-088 precedent — v12/v92 summon writers were seeded `0xAF–0xB4` /
  `0xC2–0xC7` and flagged "derived-unverified — confirm against capture". The
  operator explicitly ruled this out for task-165, and the reasoning is sound:
  a wrong opcode produces a packet the client silently discards, which presents
  identically to the bug being fixed. Worse, task-088's own notes concede v92
  "may already be on the v95 restructured high-band" — the band is not merely
  uncertain, its *structure* is uncertain. We would be shipping a plausible
  guess into the exact failure mode this task exists to close.
- **C (rejected): stand up full packet-audit columns for gms_92/gms_12.** That
  is a version bring-up (`bringup-version` skill, registry + export + column +
  n-a sweep), not a mist fix. Out of proportion; the task needs two opcodes,
  not a column.

**Discovery anchors per version** (starting points, not conclusions — each
execution step re-derives):

- **gms_48:** v61's `CAffectedAreaPool::OnPacket` @0x423eb7 is a two-arm
  dispatcher (`a2==210` → Created, `a2==211` → Removed). Locate the v48
  analogue by positional correlation off the already-named v61 pool
  dispatchers, then read the two case constants. v48's CField base is Δ≈−20 vs
  v61 and CUserPool base is 100 — the shift is non-uniform, so the case
  constants must be *read*, never computed.
- **gms_92:** method of record is dispatcher positional correlation against the
  PDB-backed v95 IDB (`CField::OnPacket` v92=0x5406b0 / v95=0x546d50), aligning
  by already-named neighbour arms. Opcode shift v92↔v95 is irregular. Disasm is
  authoritative over Hex-Rays for SEH functions.
- **gms_12:** no IDB session currently open (the IDB exists on disk). Stage
  dispatcher `CField::OnPacket`@0x47502d. v12 drift vs v48 is non-uniform
  (SkillUse +23, TransferField +11) — match by class + payload fingerprint,
  never by opcode number.

**The n-a bar for Outcome B** is `docs/packets/feature-na-evidence.yaml`'s own
stated bar: the opcode slot is a different op, or a binary-wide search for the
op's construction/gate found nothing, or the receive handler lacks the feature
branch. "I searched for the name and it wasn't there" is explicitly not
evidence — which is precisely why the `lookup_funcs` miss that opened this
investigation is treated as *undiscovered*, not *absent*. Note that only gms_48
has a matrix column, so only gms_48 needs a `feature-na-evidence.yaml` entry;
for gms_92/gms_12 the absence record lives in the task folder.

**Codec implications.** If a Tier B version is discovered and its read order
diverges from the current codec, extending `affected_area_created.go` with a
new version gate is in scope — but only in the additive direction. The existing
`Region()=="GMS" && MajorVersion()>=95` `tStart` gate must keep producing
byte-identical output for v61–jms_185, which the existing fixtures enforce. Use
the `MajorAtLeast` idiom, never a raw `> N` comparison (the
`MajorVersion()>83` off-by-one is a shipped bug in this repo).

### 4.4 FR-3 — Matrix completion: two directions, not one

The v1 design planned a single `TestAffectedAreaCreatedByteOutput` covering
five versions. The landscape has since split three ways, so the work is now:

**(a) SPAWN_MIST ❌→✅ on v83/v84/v87/v95/jms_185.** Unchanged from v1: one
`TestAffectedAreaCreatedByteOutput` with per-version subtests, each comparing
the full encoded body against an explicit `want := []byte{...}`, and five
`packet-audit:verify` markers in its doc comment. Read orders from the exports:

| Version | Address | tStart? | Body |
|---|---|---|---|
| gms_v83 | `0x431a63` | no | 39 |
| gms_v84 | `0x4326ca` | no | 39 |
| gms_v87 | `0x432f3f` | no | 39 |
| gms_v95 | `0x437ec0` | **yes** | 43 |
| jms_v185 | `0x436572` | no | 39 |

The v1 PRD's parenthetical "v95/jms fixtures must include tStart" was wrong and
was already corrected in the v1 design; it stays corrected. The codec gate is
`GMS>=95`, which excludes JMS, and the jms export notes say "NO leading
tStart". Only gms_v95 carries it. A contradicting jms decompile at execution
time triggers the STOP clause (§6).

**(b) REMOVE_MIST 🟡ᶠ→✅ on v61/v72/v79 — new work, not in the v1 plan.** These
three columns did not exist when this branch was cut. The 🟡ᶠ glyph means
"tier-1 needs byte-fixture": the codec is right, the pin is missing.
`CAffectedAreaPool::OnAffectedAreaRemoved` addresses from the exports: v61
`0x4246b0`, v72 `0x42ec4e`, v79 `0x42f0de`. The v83+ `TestAffectedAreaRemovedByteOutput`
is the shape to extend — it already pins the full body across five versions
with five markers.

**(c) v79 SPAWN_MIST re-pin — a soundness fix, not a promotion.** v79's
evidence record points `verifies:` at `TestAffectedAreaCreatedWireShape`, whose
entire v79 assertion is:

```go
if len(b) != 39 { t.Errorf(...) }
```

A length check. The cell reads ✅ but nothing pins a single byte's *value* for
v79 — the same class of false-verify the project has been burned by before, and
exactly the "approach B" the v1 design rejected for v83+ on the grounds that
"length assertions are exactly the 'passes on 1 byte'-family shortcut". It was
rejected for the new cells but is currently load-bearing for an existing one.
Fold v79 into the byte-output test (it is byte-identical to the v61/v72 path)
and re-point the evidence record at it. This changes no wire bytes and no
matrix glyph; it changes what the glyph *means*.

Consolidation note: the file currently has `TestAffectedAreaCreatedByteOutputV72`
and `...V61` as separate per-version functions, plus the shape test. Rather than
adding a third convention, the plan folds v61/v72/v79 into the same
table-driven `TestAffectedAreaCreatedByteOutput` as v83+ and retires the
per-version duplicates, keeping one convention per codec pair. The existing
`WireShape` and `Fields` tests stay as-is (they still guard the RECT math and
the version gate) but stop being evidence targets.

**Evidence records** follow the existing Removed shape exactly (`packet`,
`direction`, `version`, `category: TIER1-FIXTURE`, `ida.function/address/decompile_sha256`,
`verifies:`). `decompile_sha256` is always tool-computed from the actual
decompile used for derivation — never copied, never guessed. An address that
does not resolve or hash cleanly is a stop-and-ask, not a substitution.

### 4.5 FR-4 — Live tenant rollout

Seed templates only apply at tenant creation, so existing tenants are patched
directly:

1. **Enumerate** tenants per environment via the atlas-configurations REST
   surface; select those on any wired version.
2. **Patch** each tenant's channel `socket.writers` configuration:
   read-modify-write the existing configuration JSON, appending the two
   version-correct entries (idempotency guard: skip if `"AffectedAreaCreated"`
   already present). PATCH goes through `RegisterInputHandler`, so the JSON:API
   envelope is mandatory.
3. **Restart** atlas-channel pods per environment — the configuration
   projection does not hot-reload writer maps.
4. **Document** the per-version patch payload and the enumerate → patch →
   restart → verify sequence in `rollout.md`, with a row per wired version.
   Repo-relative paths and placeholder tenant ids only.

The rollout table is now keyed off the *actual* wired set, which is not known
until FR-2b resolves — so `rollout.md` is written after the template work, and
its version table is generated from what was wired rather than hard-coded.

Alternative rejected: a one-off migration script in atlas-configurations.
Heavier than needed, adds code that runs once, and prior rollouts of this exact
family were done via config PATCH + restart.

Full sweep, not spot-check: every tenant on every wired version in each target
environment; the rollout doc records which tenants were patched.

### 4.6 FR-5 — End-to-end verification

In a deploy environment (`deploy-env` PR label flow):

- **Mob mist:** trigger a mob with an AREA_POISON mob skill; mist renders and
  disappears on expiry. Concrete mob/map chosen at execution time from WZ/repo
  data — not from memory.
- **Player mist:** a mist skill cast renders for caster and a second observer on
  the same map. Skill id resolved from repo/WZ data at execution time. **Skill
  ids are version-specific** — the id that works on v83 is not assumed valid on
  v61 or v48; re-resolve per tenant version.
- **Coverage:** at least one modern version (v83+) and one legacy version
  (v61/v72/v79). The legacy path exercises a different opcode band and an older
  template generation, so a modern-only pass would not evidence it. Additional
  versions are best-effort, recorded as such.
- **Logs:** no `writer not found` and no `Unable to broadcast AffectedArea...`
  lines on the patched tenant; the startup writer-map warning no longer lists
  either name for patched tenants.

## 5. Component/change inventory

| Area | File(s) | Change |
|---|---|---|
| atlas-channel | `services/atlas-channel/atlas.com/channel/main.go` | +2 lines in `produceWriters()` |
| atlas-configurations | 8× Tier A `seed-data/templates/template_*.json` | +2 `socket.writers` entries each, opcode-ordered |
| atlas-configurations | up to 3× Tier B templates | +2 entries each, **conditional on FR-2b Outcome A** |
| libs/atlas-packet | `field/clientbound/affected_area_test.go` | consolidated Created byte-output test (v61→jms_185) + extended Removed byte-output test (v61/v72/v79) |
| libs/atlas-packet | `field/clientbound/affected_area_created.go` | **only if** a Tier B version diverges — additive version gate, no byte change on existing versions |
| docs/packets | `evidence/<ver>/…FieldAffectedAreaCreated.yaml` ×5 new (v83/84/87/95/jms) + v79 re-point | new/updated evidence pins |
| docs/packets | `evidence/<ver>/…FieldAffectedAreaRemoved.yaml` ×3 new (v61/72/79) | new evidence pins |
| docs/packets | `registry/gms_v48.yaml` | +2 ops, **conditional on FR-2b** |
| docs/packets | `feature-na-evidence.yaml` | +entry, **conditional on FR-2b Outcome B for gms_48** |
| docs/packets | `audits/STATUS.md`, `audits/status.json` | regenerated |
| docs/packets | `ida-exports/gms_v48.json` | re-exported if v48 discovery names the functions |
| task docs | `rollout.md`, `discovery.md` | rollout procedure + per-env record; Tier B discovery record |
| live config | atlas-tenants channel config per tenant | +2 writer entries; channel restart |

Explicitly unchanged: the mist consumer, atlas-maps mist domain, atlas-monsters
producers, and the codec's wire output for every version already ✅.

## 6. Sequencing

1. **Tier A fixtures first** (FR-3 a/b/c: Created ×5 new, Removed ×3 new, v79
   re-pin) — verification before wiring, so the wiring lands on fixture-verified
   codecs and any read-order surprise fires before templates are touched.
2. **FR-1 + FR-2 together** (Go declaration + eight templates), then
   `packet-audit matrix --check` and both template guards at exit 0.
3. **Tier B discovery** (FR-2b), per version, each resolving to A/B/C. Wire and
   verify any Outcome A. This is deliberately *after* Tier A so a stalled
   discovery cannot hold up the eight versions that are ready.
4. **Build gate:** `go test -race ./...` + `go vet ./...` in libs/atlas-packet,
   atlas-channel, atlas-configurations; `go build ./...` in atlas-channel;
   `docker buildx bake atlas-channel`; `tools/redis-key-guard.sh`; both template
   guards.
5. Code review, PR with `deploy-env` label.
6. FR-4 rollout (patch + restart), then FR-5 e2e on one modern + one legacy
   version.

Tier A is independently shippable. If Tier B stalls entirely, the task still
delivers mists on eight of eleven versions with the remaining three documented —
that is a partial-scope outcome to report, not a silent narrowing.

## 7. Error handling & failure modes

- **Read order ≠ codec on an already-✅ version (FR-3):** STOP and escalate —
  codec changes on verified versions are out of scope. Expected not to fire.
- **jms tStart ambiguity:** ruled in §4.4(a); only a contradicting jms decompile
  reopens it, via the same STOP clause.
- **Tier B discovery finds a divergent layout:** additive version gate is
  allowed (§4.3), but a change that moves bytes on any ✅ version is a STOP.
- **Tier B discovery inconclusive:** Outcome C — leave unwired, record the walk.
  Never substitute a derived opcode. This is the single most important failure
  rule in the task: the fallback is *no wiring*, not *guessed wiring*.
- **Decompile/evidence hash trouble:** unresolved address or export mismatch is
  a stop-and-ask (unresolved-fname rule), never a substituted hash.
- **Template guard failure:** `template-opcode-order-guard.sh` prints the exact
  offending pair; move the entry to its sorted position. Re-sorting is safe.
- **Tenant patch validation rejection:** full-sweep enumeration and
  read-modify-write (not blind overwrite) prevent partial/clobbered configs;
  idempotency guard makes re-runs safe.
- **"Fix didn't work" in deploy env:** first check the running image contains
  the change and the tenant config contains the entries (verify binary/config
  before re-debugging), then pod logs.

## 8. Testing strategy

- **Unit/byte level:** consolidated `TestAffectedAreaCreatedByteOutput` (full-body
  pins, v61→jms_185, including v84 which is currently untested and v79 which is
  currently length-only); extended `TestAffectedAreaRemovedByteOutput` (v61/v72/v79
  added to the existing five). Existing WireShape / Fields tests unchanged and
  still passing, but no longer load-bearing as evidence.
- **Machine checks:** `packet-audit matrix --check` exit 0; marker ↔ evidence ↔
  STATUS.md consistency is enforced by the tool, not prose.
  `tools/template-opcode-order-guard.sh` and `tools/template-movement-types-guard.sh`
  exit 0.
- **Service level:** existing mist consumer tests (broadcaster seam) are
  untouched; the writer-map change is config+declaration only and is covered by
  the e2e pass.
- **E2E:** §4.6 in the deploy environment — visible rendering, absence of the
  two error log lines, both mob-cast and player-cast paths, on one modern and
  one legacy version.

## 9. Open questions

- FR-2b outcomes for gms_48 / gms_92 / gms_12 are unknown until the discovery
  passes run. Each is designed to terminate in a recorded outcome rather than
  blocking the task.
- Everything else is pinned. The v1 PRD's v95/jms tStart parenthetical is
  resolved in §4.4(a) (jms has no tStart).
