# MOB/MONSTER Packet Family — Byte-Plumbing Batch 1 — Design

Status: Approved (design phase)
Created: 2026-06-13
PRD: `docs/tasks/task-092-mob-packet-family/prd.md`

---

## 1. Scope & Locked Decisions

This task implements byte-exact codecs for the **42 currently-unimplemented MOB/MONSTER
operations** (PRD §4.1) across the five supported versions (gms_v83, gms_v84, gms_v87,
gms_v95, jms_v185), wires each into `atlas-channel` + the five seed templates, and promotes
every applicable coverage-matrix cell to `verified`. **Gameplay behavior is out of scope.**

Three design decisions were settled with the user before writing this document:

| # | Decision | Resolution |
|---|----------|-----------|
| D1 | Carnival cluster (E, 9 ops) — split out or keep? | **Keep all 42 ops in task-092.** One batch closes the whole MOB/MONSTER family. |
| D2 | Per-op deliverable scope | **Codec + route + byte-test only.** No producer-trigger or action stubs in `atlas-monsters`/`atlas-monster-book`. Serverbound handlers decode-and-log; clientbound writers are registered seams with no emitter. |
| D3 | Where the reusable recipe lives | **New `docs/packets/IMPLEMENTING_A_PACKET.md`**, cross-linking the existing `VERIFYING_A_PACKET.md`. |

D2 is the load-bearing simplification: it removes PRD Open Question #4 (stub ownership)
entirely. Nothing in `atlas-monsters` or `atlas-monster-book` changes. The "byte-plumbing"
deliverable is exactly: **a codec the matrix can verify + the wiring that lets a future
behavior task call it.**

### What "verified" actually requires

The matrix promotes a cell to `verified` (tier-1, since `monster/` and `character/` are
both tier-1 prefixes per `docs/packets/evidence/tiers.yaml`) from:

1. An immutable codec in `libs/atlas-packet/...` with **both** `Encode` and `Decode`
   (the `Packet` interface requires both; both are needed for the round-trip test).
2. A per-version byte-fixture test carrying a `// packet-audit:verify packet=… version=… ida=0x…` marker.
3. A pinned evidence record (`docs/packets/evidence/<version>/<packet_dots>.yaml`) whose
   `decompile_sha256` matches the current IDA export — **mandatory for tier-1**.

The seed-template route + `atlas-channel` registration are **not** required by the matrix
grader for `verified`, but they are required by the PRD acceptance criteria and they are
what keeps the registry⇄template⇄code applicability consistent (a missing/extra route is a
`conflict` cell, which is a `--check` blocker). So every op gets all of: codec, test+marker,
evidence pin, template routes (×5 with validators), and `atlas-channel` registration.

---

## 2. Architecture

Four layers, each with a single responsibility and a well-defined seam to the next. The work
is the same shape for all 42 ops; only the byte layout differs.

```
┌─ libs/atlas-packet/<family>/<dir>/<op>.go ────────────────────────────┐
│  Immutable model (private fields + getters + constructor).            │
│  Operation() string  → writer/handler NAME (the wiring key)           │
│  Encode(l, ctx)(opts) []byte   — clientbound payload (no opcode)      │
│  Decode(l, ctx)(r, opts)       — serverbound parse                    │
│  Version-branches on tenant.MustFromContext(ctx) (Region/MajorAtLeast)│
└───────────────┬───────────────────────────────────────────────────────┘
                │  Operation() name
┌───────────────▼─ atlas-channel ───────────────────────────────────────┐
│  clientbound:  produceWriters() += NAME ; socket/writer/<op>.go Body   │
│  serverbound:  produceHandlers()[NAME] = <op>Func (decode + log) ;     │
│                validator = LoggedInValidator (NoOp for conn-level)     │
└───────────────┬───────────────────────────────────────────────────────┘
                │  NAME ↔ opcode binding (per version)
┌───────────────▼─ atlas-configurations seed templates (×5) ────────────┐
│  socket.handlers[] += {opCode, validator, handler:NAME}               │
│  socket.writers[]  += {opCode, writer:NAME}                           │
│  opCode value comes from docs/packets/registry/<version>.yaml         │
└───────────────┬───────────────────────────────────────────────────────┘
                │  applicability (registry ⇄ template ⇄ code)
┌───────────────▼─ packet-audit matrix ─────────────────────────────────┐
│  scans verify markers + evidence + templates → grades each cell       │
│  `matrix` regenerates STATUS.md/status.json ; `matrix --check` gates  │
└────────────────────────────────────────────────────────────────────────┘
```

**Runtime data flow is asymmetric and intentional:**

- **Serverbound** ops are live end-to-end after this task: client sends → `atlas-channel`
  handler decodes + logs (no "unhandled op" warning, no action taken). This is genuinely
  useful and not dead code.
- **Clientbound** ops have a registered writer with **no emitter** in this task. The writer
  `Body` helper and `produceWriters()` entry exist so a future behavior task calls
  `session.Announce(...)(NAME)(Body(...))` without re-wiring. This is the one place we
  knowingly land an uncalled seam (documented in IMPLEMENTING_A_PACKET.md so reviewers
  don't flag it as dead code).

---

## 3. The Unit of Work — Per-Op Recipe

Every op is one independent unit, executed against four existing skeletons. This recipe **is**
the second deliverable (§9); it gets written up verbatim in `IMPLEMENTING_A_PACKET.md`.

### Step 1 — Derive structure from the IDB

For each applicable version, `select_instance(port)` then `decompile` the registry entry's
`fname`, descending into helper reads/writes (same descent rule as the exporter). Record the
ordered field list with widths (`Decode1/2/4/Str/Buffer`) and every per-version delta.

Multi-instance IDA ports (PRD §4.2): **v83=13337, v87=13338, v95=13339, jms=13340, v84=13341.**

Guards before coding:
- **Cluster-F fname mislabels** (PRD Open Q#3): several non-universal ops have stale registry
  fnames (e.g. `MOB_SPEAKING→OnIncMobChargeCount`). Confirm each `fname` against the IDB
  *before* deriving; if wrong, fix the registry entry (provenance `manual`, with an IDA
  citation) in the same commit. Same staleness class as task-085 v84.
- **MONSTER_BOOK_COVER (serverbound)** has no registry `fname` (PRD Open Q#2). Derive its
  send-site from the IDB during this step; populate the registry `fname` (provenance
  `ida-discovered`).
- **v84 ≡ v83 caveat**: per project memory, v84 packet structure is byte-identical to v83
  below the shifted-opcode-table region; do not invent v84 deltas the IDB doesn't show. Use
  `MajorAtLeast(87)`-style gates, never `>83`, so v84 takes the v83 path.

### Step 2 — Model + codec

Add `libs/atlas-packet/<family>/<dir>/<op>.go` following the existing
`monster/clientbound/spawn.go` / `monster/serverbound/movement.go` pattern exactly:

```go
const <Op>Writer = "<Name>"        // or <Op>Handle for serverbound
type <Op> struct { /* private fields */ }
func New<Op>(...) <Op> { ... }     // constructor; getters only, no setters
func (m <Op>) Operation() string { return <Op>Writer }

func (m <Op>) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
    w := response.NewWriter(l)
    t := tenant.MustFromContext(ctx)
    return func(options map[string]interface{}) []byte {
        // fields in client read-order; version-branch on t.Region()/t.MajorAtLeast(n)
        return w.Bytes()
    }
}
func (m *<Op>) Decode(l logrus.FieldLogger, ctx context.Context) func(*request.Reader, map[string]interface{}) {
    t := tenant.MustFromContext(ctx)
    return func(r *request.Reader, options map[string]interface{}) { /* mirror of Encode */ }
}
```

Both directions are implemented for every op (the round-trip test drives both). Per-version
structural variants live **inside** `Encode`/`Decode` via `ctx`, not as separate types,
unless a version diverges enough to warrant its own model (decide per op; default = single
model with branches). Reuse shared sub-structs from `libs/atlas-packet/model/`
(`Movement`, `MonsterTemporaryStat`, etc.) rather than re-deriving them.

### Step 3 — Wire

- **clientbound**: append `<Op>Writer` to `produceWriters()` (main.go) and add a thin
  `socket/writer/<op>.go` `Body` helper that constructs the model from domain inputs and
  returns `New<Op>(...).Encode(l, ctx)(options)`.
- **serverbound**: add `produceHandlers()[<Op>Handle] = <Op>HandleFunc` and a
  `socket/handler/<op>.go` func that does `p := serverbound.<Op>{}; p.Decode(l, ctx)(r, ro);
  l.Debugf("[%s] read [%s]", p.Operation(), p.String())` — decode + log, no action.
- **all five seed templates**: add the `socket.handlers`/`socket.writers` entry with the
  per-version opcode from `docs/packets/registry/<version>.yaml`. **Every handler entry
  carries a `validator`** — `LoggedInValidator` by default, `NoOpValidator` only for
  connection-level ops. A handler entry with no/unknown validator is silently dropped by
  `BuildHandlerMap` (`continue`), so this is mandatory, not cosmetic.

Only the two existing validators exist (`LoggedInValidator`, `NoOpValidator`); no new
validator types are introduced.

### Step 4 — Verify

- Add `libs/atlas-packet/<family>/<dir>/<op>_test.go`: round-trip across `test.Variants`
  using `test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)` (reference:
  `reactor/clientbound/destroy_test.go`). For mask/mode-driven packets, add an explicit
  golden-byte assertion for the v83 baseline citing the decompile line per field — round-trip
  alone proves encode/decode symmetry, not byte-exactness vs the client.
- Add one `// packet-audit:verify packet=<pkg/dir/Struct> version=<key> ida=0x<addr>` marker
  per applicable version, above the test func.
- Pin evidence (tier-1 — required for every MOB op):
  `go run ./tools/packet-audit evidence pin --packet <id> --version <key> --ida "<FName>" --category TIER1-FIXTURE`,
  then add the `verifies:` list to the generated YAML.
- `go run ./tools/packet-audit matrix && go run ./tools/packet-audit matrix --check` →
  the cell must show ✅ and `--check` must exit 0.

---

## 4. Package & Direction Placement (the 42 ops)

Package is chosen by the **owning client class family** (matches the existing convention:
`set_taming_mob_info.go`, a `CWvsContext` op, lives under `character/clientbound/`):

| Owner class | Package | Tier-1? |
|---|---|---|
| `CMob::` / `CMobPool::` | `monster/{clientbound,serverbound}` | yes (`monster/`) |
| `CField_MonsterCarnival::` / `CUIMonsterCarnival::` | `monster/carnival/{clientbound,serverbound}` | yes (keeps `monster/` prefix) |
| `CWvsContext::` | `character/clientbound` | yes (`character/`) |
| `CUserLocal::` | `character/serverbound` | yes (`character/`) |

> Carnival packets are placed under `monster/carnival/...` specifically so the `monster/`
> tier-1 prefix in `tiers.yaml` still matches. A top-level `carnival/` package would **not**
> be tier-1 and could be wrongly promoted by a flat-diff verdict — avoid it.

**Cluster A — Mob combat/damage (10 ops):** `FIELD_DAMAGE_MOB`, `MOB_DAMAGE_MOB`,
`MOB_DAMAGE_MOB_FRIENDLY`, `MONSTER_BOMB`, `MOB_TIME_BOMB_END`, `MOB_SKILL_DELAY_END`
→ `monster/serverbound`; `TOUCH_MONSTER_ATTACK` → `character/serverbound` (CUserLocal);
`MOB_AFFECTED`, `MONSTER_SPECIAL_EFFECT_BY_SKILL`, `RESET_MONSTER_ANIMATION` → `monster/clientbound`.

**Cluster B — Catch/taming (4 ops):** `CATCH_MONSTER`, `CATCH_MONSTER_WITH_ITEM` →
`monster/clientbound` (CMob); `BRIDLE_MOB_CATCH_FAIL`, `SET_TAMING_MOB_INFO` →
`character/clientbound` (CWvsContext). **`SET_TAMING_MOB_INFO` is already implemented by
task-086** (`character/clientbound/set_taming_mob_info.go` + test) — see §6.

**Cluster C — Monster book (3 ops):** `MONSTER_BOOK_SET_CARD`, `MONSTER_BOOK_SET_COVER` →
`character/clientbound` (CWvsContext); `MONSTER_BOOK_COVER` → `character/serverbound`
(fname derived in Step 1).

**Cluster D — CRC/misc plumbing (4 ops):** `MOB_CRC_KEY_CHANGED` → `monster/clientbound`;
`MOB_CRC_KEY_CHANGED_REPLY`, `MOB_DROP_PICKUP_REQUEST` → `monster/serverbound`;
`MOB_BANISH_PLAYER` → `character/serverbound` (CUserLocal).

**Cluster E — Monster Carnival (9 ops):** `MONSTER_CARNIVAL` → `monster/carnival/serverbound`;
`MONSTER_CARNIVAL_START/OBTAINED_CP/PARTY_CP/SUMMON/MESSAGE/DIED/LEAVE/RESULT` →
`monster/carnival/clientbound`.

**Cluster F — Version-tail (non-universal, ~12 ops):** `INC_MOB_CHARGE_COUNT` (4v, cb),
`MOB_SKILL_DELAY` (4v), `MOB_SPEAKING` (4v), `MOB_ESCORT_COLLISION` (3v, sb),
`MOB_ESCORT_FULL_PATH` (2v), `MOB_ESCORT_STOP_END_REQUEST` (2v, sb),
`MOB_REQUEST_ESCORT_INFO` (2v, sb), `MOB_ATTACKED_BY_MOB` (1v), `MOB_ESCORT_RETURN_BEFORE/STOP/STOP_SAY` (1v),
`MOB_NEXT_ATTACK` (1v) → `monster/{clientbound,serverbound}` by direction. **Implement only for
the versions where each is applicable**; any version where the op is genuinely absent stays
`n/a` with an IDB-evidenced justification (never a silent skip).

> The exact op count per cluster is the PRD's; the registry (`gms_v83.yaml`) confirms all
> names. The IDB-derived applicability per version is settled in Step 1, not assumed here.

---

## 5. Cluster Sequencing

Sequence so the recipe is proven on the simplest ops first, then the shared-struct and
minigame clusters:

1. **D (CRC/misc)** — smallest, mostly fixed-width scalars; shakes out the end-to-end recipe
   and the IMPLEMENTING_A_PACKET.md draft on low-risk packets.
2. **A (combat/damage)** — exercises shared sub-structs (`Movement`, damage entries) and the
   `>83`→`>=87` gate discipline.
3. **B + C (catch/taming + monster book)** — `CWvsContext`/`CUserLocal` packages; includes
   the SET_TAMING_MOB_INFO dedup and the MONSTER_BOOK_COVER fname derivation.
4. **F (version-tail)** — applicability-per-version + the fname-mislabel guard; most `n/a`
   justifications live here.
5. **E (carnival)** — the coherent 9-op minigame sub-feature, landed last as one block.

Each cluster is independently committable (codec + wiring + tests + evidence + regenerated
STATUS.md), keeping reviews bounded even though all 42 live in one task/PR.

---

## 6. Dedup: SET_TAMING_MOB_INFO (Open Q#1 — RESOLVED)

task-086 already shipped the full encoder **and** test:
`libs/atlas-packet/character/clientbound/set_taming_mob_info.go`
(`characterId, level, exp, tiredness, levelUp`) + `set_taming_mob_info_test.go`, plus the
seed-template writer routes (opcodes: v83/v84/v87 `0x30`, v95 `0x2F`, jms `0x2D`).

**task-092 does not re-implement it.** It only:
- adds the `// packet-audit:verify` markers (one per version) if the existing test lacks them;
- pins the tier-1 evidence records;
- regenerates the matrix so the SET_TAMING_MOB_INFO cells flip to `verified`.

If a verify marker + evidence already exist and the cell is already `verified`, this op is a
no-op for task-092 (confirm during Cluster B).

---

## 7. Operational Rollout

Seed templates apply only at tenant **creation**; existing tenants do not pick up new opcodes
automatically (project memory: `bug_new_opcodes_not_in_live_tenant_config`). After the code
lands:

1. PATCH each live v83/v84/v87/v95/jms tenant config — add the new `socket.handlers`
   (with validator) and `socket.writers` entries with the per-version opcodes.
2. **Restart `atlas-channel`** — the handler/writer map is built once at startup; the config
   projection does not hot-reload handlers/writers.
3. Post-deploy checks: `kubectl logs <atlas-channel> | grep "Unable to locate validator"` == 0;
   no new error/fatal logs; serverbound ops no longer emit "unhandled message op 0xXX".

This mirrors the task-086 procedure (`deploy-notes.md §2/§5`). A short `deploy-notes.md` with
the full per-version opcode table is produced as part of this task.

---

## 8. Verification Gates

- `go test -race ./...` clean in every changed module (`libs/atlas-packet`, `atlas-channel`,
  `atlas-configurations`, plus `tools/packet-audit` if touched).
- `go vet ./...` clean; `tools/redis-key-guard.sh` clean.
- `go build ./...` clean for `atlas-channel` and `atlas-configurations`.
- **No `go.mod` touched** (`libs/atlas-packet` is already a workspace member; no new lib) →
  no new `Dockerfile` COPY lines and no `go.work` edit. `docker buildx bake atlas-channel`
  and `atlas-configurations` only if their `go.mod` changes (not expected). Confirm with
  `git diff --name-only -- '**/go.mod'` before claiming done.
- `go run ./tools/packet-audit matrix --check` exits 0; every targeted MOB/MONSTER cell is
  `verified` (or `n/a` with IDB evidence); zero `conflict` cells.
- Code review: `plan-adherence-reviewer` + `backend-guidelines-reviewer`.

---

## 9. Deliverable 2 — `docs/packets/IMPLEMENTING_A_PACKET.md`

A new companion to `VERIFYING_A_PACKET.md` (which covers verifying an *existing* codec). It
documents the §3 recipe as the reusable template for the remaining ~420 unimplemented ops in
other domains, including:

- the four-step recipe (derive → model+codec → wire → verify) with the exact code skeletons;
- the package-by-owner-class placement rule (§4) and the tier-1 prefix caveat;
- the "register a clientbound writer with no emitter" seam convention (so reviewers don't
  flag it as dead code) — the explicit output of decision D2;
- the validator-mandatory rule and the `BuildHandlerMap` silent-drop failure mode;
- the registry-fname-mislabel guard and the `>83`→`>=87` gate rule;
- the live-tenant patch + channel-restart rollout checklist.

It cross-links `VERIFYING_A_PACKET.md`, `tiers.yaml`, and the registry README.

---

## 10. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Registry fname mislabels (Cluster F) produce wrong byte layouts | Step-1 guard: confirm every `fname` against the IDB before coding; fix registry in-commit with IDA citation. |
| Carnival placed outside `monster/` loses tier-1 → flat-diff false-promotes | Place under `monster/carnival/...`; verified by `tiers.yaml` prefix match. |
| Clientbound writers with no emitter read as dead code in review | Documented seam convention in IMPLEMENTING_A_PACKET.md; backend-guidelines reviewer briefed via the doc. |
| Live tenants miss the new opcodes → silent drops | §7 rollout: PATCH live config + restart + `Unable to locate validator`==0 check. |
| v84 deltas invented where v84≡v83 | `MajorAtLeast(87)` gates only; never `>83`; derive deltas strictly from the v84 IDB. |
| MONSTER_BOOK_COVER serverbound has no fname | Derive send-site in Step 1; populate registry (`ida-discovered`) before coding. |
| 42 ops × ≤5 versions is large for one PR | Cluster-gated commits (§5); each cluster independently reviewable; matrix `--check` green per cluster. |

---

## 11. Out of Scope

- All gameplay behavior (mob-catch removal, monster-book card tracking, carnival match
  orchestration, CRC enforcement, banish action). Deferred to later behavior tasks.
- Producer-trigger stubs and serverbound action stubs in `atlas-monsters`/`atlas-monster-book`
  (decision D2 — not landed at all).
- Non-MOB/MONSTER operation families (separate batches, templated by IMPLEMENTING_A_PACKET.md).
- Reclassifying any live cell to `n/a` to "close" it (only genuine, IDB-evidenced
  version-absence may be `n/a`).

---

## 12. Open-Question Resolution Summary

| PRD Open Q | Resolution |
|---|---|
| #1 SET_TAMING_MOB_INFO vs task-086 | task-086 ships the encoder + test; task-092 only adds marker + evidence to flip the cell (§6). |
| #2 MONSTER_BOOK_COVER serverbound fname | Derive send-site from IDB in Step 1; populate registry. |
| #3 Cluster-F fname mislabels | Step-1 guard verifies every fname against the IDB before coding. |
| #4 Producer/handler stub ownership | Dissolved by D2 — no stubs land; nothing in atlas-monsters/atlas-monster-book changes. |
| #5 Carnival split | D1 — keep all 9 carnival ops in task-092, under `monster/carnival/...`. |
| #6 Behavior-stub convention | Dissolved by D2 — serverbound handlers decode+log; clientbound writers are uncalled seams, documented in IMPLEMENTING_A_PACKET.md. |
