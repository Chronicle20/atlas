# task-096 — Implementation Context

Dense fact sheet for the executor. Every value here was read from the worktree on
2026-06-14; re-verify against source before relying on a value that smells stale. This is
the **direct successor to task-092** (MOB/MONSTER family) — the recipe is identical and is
codified in `docs/packets/IMPLEMENTING_A_PACKET.md` / `docs/packets/VERIFYING_A_PACKET.md`.
task-096 **follows** those docs; it does not re-author them.

The plan (`plan.md`) references this file by section (e.g. `context.md §1`) instead of
repeating data. Read this whole file before starting Stage 0.

---

## 1. Key files & sites

### libs/atlas-packet (no go.mod change — workspace member)
- Codec packages in scope: `field/{clientbound,serverbound}/` (the owner-class home for this
  family, including the relocated chat ops), and the existing `chat/{clientbound,serverbound}/`
  (source of the 3 relocated codecs; non-CField chat codecs stay there).
- New minigame ops stay in `field/{clientbound,serverbound}/` — the `CField_*` subclasses are
  still `field/`-owned (keep the tier-1 prefix; do NOT create a top-level `snowball/` etc.).
- Test helpers: package `github.com/Chronicle20/atlas/libs/atlas-packet/test`
  - `test.Variants` — `libs/atlas-packet/test/context.go` — entries include
    `{GMS v28, GMS v83, GMS v87, GMS v95, JMS v185, GMS v84, GMS v86}` (v84/v86≡v83 byte-wise).
    Re-read the file for the exact current set before relying on a count.
  - `test.CreateContext(region string, major uint16, minor uint16) context.Context`.
  - `test.RoundTrip(t, ctx, encode, decode, options)` — asserts `reader.Available()==0` after decode.
- Wire I/O method names (verbatim, `libs/atlas-socket/response/writer.go` & `request/reader.go`):
  - Writer: `WriteInt8 WriteInt16 WriteInt32 WriteInt64 WriteInt(uint32) WriteShort(uint16) WriteLong(uint64) WriteByte WriteByteArray WriteBool WriteAsciiString WriteKeyValue Bytes Skip`
  - Reader: `ReadByte ReadInt8 ReadBool ReadBytes(int) ReadInt16 ReadInt32 ReadInt64 ReadUint16 ReadUint32 ReadUint64 ReadString(int16) ReadAsciiString Skip Position Seek Available GetRestAsBytes`
- Model convention (`field/clientbound/set_field.go`, `field/clientbound/clock.go`,
  `chat/serverbound/general.go`): private fields + getters, `New<Op>(...)` constructor,
  `Operation() string`, `String() string`,
  `Encode(l, ctx) func(map[string]interface{}) []byte`,
  `Decode(l, ctx) func(*request.Reader, map[string]interface{})`.
  Version-branch via `t := tenant.MustFromContext(ctx)` then `t.Region()` / `t.MajorAtLeast(n)` /
  `t.IsRegion("GMS")`. **Gate rule:** use `MajorAtLeast(87)`, never `>83` (v84/v86 take the v83 path).
- Shared sub-structs live in `libs/atlas-packet/model/` (`Position`, `Movement`, etc.) — reuse,
  never re-derive.

### atlas-channel — `services/atlas-channel/atlas.com/channel/`
- `main.go:594` `func produceWriters() []string` — append each new `fieldcb.<Op>Writer` const.
- `main.go:721` `func produceHandlers() map[string]handler.MessageHandler` — add
  `hm[fieldsb.<Op>Handle] = handler.<Op>HandleFunc`.
- `main.go:810` `func produceValidators()` — only `NoOpValidator` + `LoggedInValidator` exist;
  do **not** add new validators.
- Import aliases already present (main.go:80-87): `chatCB`/`chatSB` (`chat/{clientbound,serverbound}`),
  `fieldcb`/`fieldsb` (`field/{clientbound,serverbound}`). The chat relocation repoints any
  `chatCB.X`/`chatSB.X` references for relocated ops to `fieldcb.X`/`fieldsb.X`.
- Clientbound Body helper pattern — `socket/writer/*.go`:
  `func <Op>Body(<domain args>) packet.Encode { return func(l, ctx) func(options) []byte { … return fieldcb.New<Op>(…).Encode(l, ctx)(options) } }`.
- Serverbound handler pattern — `socket/handler/*.go`:
  `func <Op>HandleFunc(l, ctx, wp writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) { return func(s, r, ro) { p := fieldsb.<Op>{}; p.Decode(l, ctx)(r, ro); l.Debugf("[%s] read [%s]", p.Operation(), p.String()) } }`.
- **atlas-channel go.mod is reached transitively** when channel files change → `docker buildx bake
  atlas-channel` only if its go.mod actually changes (codec is a workspace member; expected: no
  go.mod change — confirm with `git diff --name-only -- '**/go.mod'`).

### atlas-configurations — seed templates
- Five files: `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_95}_1.json`, `template_jms_185_1.json`.
- Top-level nesting: `socket.handlers[]` (each `{opCode, validator, handler[, options]}`) and
  `socket.writers[]` (each `{opCode, writer[, options]}`).
- Entries are **ordered by ascending opCode** — insert in sorted position.
- `opCode` strings are hex (e.g. `"0xBC"`). Every handler entry MUST carry a `validator` or
  `BuildHandlerMap` silently `continue`s it (`libs/atlas-opcodes/producer.go`,
  memory `bug_socket_handler_missing_validator_silently_dropped`).
- atlas-configurations go.mod does NOT import atlas-packet; editing only JSON templates does not
  change go.mod → no bake required for JSON-only edits.

### packet-audit tool — `tools/packet-audit/`
- `go run ./tools/packet-audit matrix` regenerates `docs/packets/audits/STATUS.md` + `status.json`.
- `go run ./tools/packet-audit matrix --check` (CI gate): exit 0 clean, 1 blocker
  (conflict/drift/orphan/stale).
- `go run ./tools/packet-audit evidence pin --packet <id> --version <key> --ida "<FName>" --category TIER1-FIXTURE`
  — reads the **static export** `docs/packets/ida-exports/<version>…json` (NO live IDA), computes
  `decompile_sha256`, writes `docs/packets/evidence/<version>/<packet_dots>.yaml`. **Fails if
  `<FName>` is not in that export.**
- Verify marker (in `*_test.go`): `// packet-audit:verify packet=<pkg/dir/Struct> version=<key> ida=0x<addr>`
  — all three keys required. The `ida=` address and the evidence `ida.address`/hash both come
  from the **export**; a live-IDA address that mismatches the export → `--check` flags an orphan marker.
- Evidence categories: `OPAQUE TRUNCATION REPRESENTATION OP-MODE-PREFIX LOOP-EXCLUSIVE-BRANCH VERSION-ABSENT TIER1-FIXTURE`.
  Use `TIER1-FIXTURE` for implemented ops; `VERSION-ABSENT` for genuine version-absence n/a.
- Tier-1 prefixes (`docs/packets/evidence/tiers.yaml`): `monster/`, `character/`, **`field/`**,
  `party/`, … — **all CField ops live under `field/` and are tier-1** → require a linked
  byte-test + pinned evidence; flat-diff cannot promote them. `chat/` is **NOT** a tier-1 prefix
  (this is the load-bearing reason for the D3 relocation — see design §1/§5).
- Grading (`tools/packet-audit/internal/matrix/grade.go`): `n/a` = registry Absent + no
  conflicting report; `conflict` = registry Absent but an Atlas report/route exists, OR registry
  Present + implemented but this version's template omits the route while another routes it.

### registry — `docs/packets/registry/<version>.yaml`
- Schema (README): `op, direction, opcode, fname, [fname_alts], provenance, [ida.address], [note]`;
  unique `(op, direction)` per file. `opcode` is decimal in the yaml.
- provenance: `csv-import` (frozen historical), `ida-discovered`, `manual` (needs IDA citation in `note`).

---

## 2. The 75-op work-list & owners

The authoritative grouped list is `structures/cfield-ops.md` (committed in design phase). Summary:

- **Core `CField::` — 45 ops** (chat, transfer/blocked, field-obstacle, quest/clock, GM events,
  boss timers, admin, MTS, door/guild, foothold/stalk `IDA_0x…` rows).
- **`CField_*` minigames — 30 ops** across SnowBall (6), Tournament (5), Wedding (4), Coconut (3),
  GuildBoss (3), ContiMove (2), AriantArena (2), Battlefield/sheep-ranch (2), Massacre (1),
  MassacreResult (1), Witchtower (1).

`dir` column legend: `CB`=clientbound (`On*`), `SB`=serverbound (`Send*`), `?`=unresolved
direction. `PKT`=already has a codec file somewhere.

**Already-implemented (likely A-rows — verify, do not duplicate).** Per-op confirmation is
Stage 0 triage's job, but these are the known starting points:
- `field/clientbound/` already has: `effect`, `effect_weather`, `clock`, `affected_area_*`,
  `kite_*`, `set_field`, `transport`, `warp_to_map`. Several already carry `// packet-audit:verify`
  markers (e.g. `clock_test.go`, `effect_test.go`) — those cells may already be ✅ for some
  versions; the ❌ in the work-list is for the **CField ops not yet linked**, not these.
- `chat/clientbound/`: `general`, `multi`, `whisper`, `world_message`, `world_message_extra`.
- `chat/serverbound/`: `general`, `multi`, `whisper`.
- The 3 relocation candidates (design §5) are `chat/serverbound/general.go` (GENERAL_CHAT),
  `chat/clientbound/multi.go` (MULTICHAT), `chat/clientbound/whisper.go` (WHISPER). Confirm each
  serves **only** `CField::`-owned ops before moving (the move-rule in design §5).
- `world_message*.go` stays in `chat/` (owner `CWvsContext::OnBroadcastMsg`, not CField).

### Registry/fname C-rows to resolve in triage (design §3, PRD §9)
- **Foothold/stalk cluster** `IDA_0X098/09C/09D/0A4/0AA/0AC/0B0/0B1` (+ `IDA_0X169`): fnames
  `OnStalkResult` / `OnFootHoldInfo` / `OnRequestFootHoldInfo` / `OnHontailTimer`. Resolve the
  real op-name + decide collapse-into-named-op / new-codec / ⬜ before any code.
- **`MTS_OPERATION` / `MTS_OPERATION2`** — both `CField::OnCharacterSale`. Decide two-modes-of-one
  vs two-distinct-structs by decompiling `OnCharacterSale`.
- **`USE_DOOR`** (`CField::TryEnterTownPortal`) and **`GUILD_OPERATION`** (`CField::InputGuildName`,
  marked `PKT`) — resolve direction (`?` in the list).
- **Minigame `?` rows** (`SNOWBALL`/`LEFT_KNOCKBACK`/`COCONUT`/`GUILD_BOSS`/`CONTI_MOVE` via
  `Update`/`BasicActionAttack`/`Init`) — those fnames are state/update methods, not send/recv
  sites; derive the real send-site fname (or confirm serverbound recv) before classifying.
- **Duplicate `WHISPER` rows**: `cfield-ops.md` lists `WHISPER`/`OnWhisper` twice — confirm whether
  this is two registry rows (e.g. whisper vs whisper-reply mode) or a list dupe; resolve in triage.

---

## 3. Per-version opcodes & directions — read from the registry, NOT memory

There is **no pre-baked opcode table** for this family (unlike task-092 context §2). Per-version
opcodes come from `docs/packets/registry/<version>.yaml`, **read per file in Stage 1** and recorded
into `structures/<version>.md` alongside the byte layout. CLAUDE.md forbids citing opcodes/bytes
from memory; the COutPacket/recv-dispatch opcode in the IDB is ground truth (registry csv decimals
can be off-by-one).

For each op × applicable version, Stage 1 records into `structures/<version>.md#<OP>`: demangled
fname, export address, ordered field list (`name : width : note`) in client read-order (with
version guards and loop bounds), and the registry opcode (decimal→hex).

---

## 4. IDA multi-instance ports (Stage 1 only)

v83=13337, v87=13338, v95=13339, jms=13340, v84=13341 (design §4). **One IDB loaded at a time** —
the user switches the active IDB; `select_instance(port)` is shared global state, so batch ALL
derivations for a version before moving on (memory `reference_ida_harvest_subagents`). Subagents
can reach IDA-MCP. jms uses the clean `*_U_DEVM` build, **never** the SMC retail dump.

Guards (memory `bug_v84_opcode_table_shifted_vs_v83`, task-085/092):
- v84 ≡ v83 below the shifted opcode-table region → record only deltas; use `MajorAtLeast(87)`
  gates, never invent v84 structural deltas the IDB doesn't show.
- Confirm every `fname` against the IDB before coding (stale-registry-fname class).
- The `*_U_DEVM` jms build is the only valid jms source.

---

## 5. Export / report mechanics (reuse task-092 findings)

- The export (`docs/packets/ida-exports/<version>…json`) is **NOT idempotent** — never overwrite a
  committed export; surgically splice only the needed fname entries (`VERIFYING_A_PACKET.md` §10).
- Strip the misclassified `COutPacket`-ctor `Delegate` artifact when report-gen descent fails on it.
- A `routedElsewhere && !routed` conflict = a real template-wiring gap; route it (if the tenant
  should support it) or don't claim the cell.
- Before any `evidence pin`, confirm the op's `fname` resolves in the matching export JSON's
  `functions` map. **ESCALATE every unresolved fname to the user — stop and ask; do not auto-
  re-export, substitute a fname, or fake the hash** (memory `feedback_unresolved_fname_escalate`).

---

## 6. Current matrix state

All 75 targeted `CField*` ops show at least one `❌ incomplete` cell (318 ❌ total) in
`docs/packets/audits/STATUS.md`. Target end-state: every applicable cell `✅ verified`, genuine
absences `⬜ n/a` with `VERSION-ABSENT` evidence, zero `🟥 conflict` attributable to this task.
Confirm the live baseline in Stage 0 (`matrix --check` exit code + STATUS.md snapshot) so
pre-existing non-CField failures are not attributed to task-096.

---

## 7. Verification gates (CLAUDE.md)

- `go test -race ./...` + `go vet ./...` clean in every changed module (`libs/atlas-packet`,
  `services/atlas-channel/atlas.com/channel`, `services/atlas-configurations`, plus
  `tools/packet-audit` if touched).
- `go build ./...` clean for atlas-channel and atlas-configurations.
- `GOWORK=off tools/redis-key-guard.sh` clean from the repo root (memory `reference_rediskeyguard_invariant`).
- `git diff --name-only -- '**/go.mod'` — expected empty; if a go.mod changed,
  `docker buildx bake atlas-<svc>` from the worktree root for that service.
- `go run ./tools/packet-audit matrix --check` exit 0; zero conflict cells; no orphan/dangling/
  stale/drift line mentioning a CField packet.
- Seed-template JSON valid; `handlers`/`writers` arrays ascending by opCode; every handler entry
  carries a validator.
- Code review: `plan-adherence-reviewer` + `backend-guidelines-reviewer` green before PR.
