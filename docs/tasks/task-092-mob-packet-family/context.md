# task-092 — Implementation Context

Dense fact sheet for the executor. Every value here was read from the worktree on
2026-06-13; re-verify against source before relying on a value that smells stale.

---

## 1. Key files & sites

### libs/atlas-packet (no go.mod change — workspace member)
- Codec packages: `monster/{clientbound,serverbound}/`, `monster/carnival/{clientbound,serverbound}/` (new),
  `character/{clientbound,serverbound}/`.
- Test helpers: package `github.com/Chronicle20/atlas/libs/atlas-packet/test`
  - `test.Variants` — `libs/atlas-packet/test/context.go:18` — **7 entries**:
    `{GMS v28, GMS v83, GMS v87, GMS v95, JMS v185, GMS v84, GMS v86}` (v84/v86 appended; v84≡v86≡v83 byte-wise).
  - `test.CreateContext(region string, major uint16, minor uint16) context.Context` — `context.go:31`.
  - `test.RoundTrip(t, ctx, encode, decode, options)` — `roundtrip.go:21`; asserts `reader.Available()==0` after decode.
- Wire I/O method names (verbatim, `libs/atlas-socket/response/writer.go` & `request/reader.go`):
  - Writer: `WriteInt8 WriteInt16 WriteInt32 WriteInt64 WriteInt(uint32) WriteShort(uint16) WriteLong(uint64) WriteByte WriteByteArray WriteBool WriteAsciiString WriteKeyValue Bytes Skip`
  - Reader: `ReadByte ReadInt8 ReadBool ReadBytes(int) ReadInt16 ReadInt32 ReadInt64 ReadUint16 ReadUint32 ReadUint64 ReadString(int16) ReadAsciiString Skip Position Seek Available GetRestAsBytes`
- Model convention (`monster/clientbound/spawn.go`, `character/clientbound/set_taming_mob_info.go`):
  private fields + getters, `New<Op>(...)` constructor, `Operation() string`, `String() string`,
  `Encode(l, ctx) func(map[string]interface{}) []byte`, `Decode(l, ctx) func(*request.Reader, map[string]interface{})`.
  Version-branch via `t := tenant.MustFromContext(ctx)` then `t.Region()` / `t.MajorAtLeast(n)` / `t.IsRegion("GMS")`.
  **Gate rule:** use `MajorAtLeast(87)`, never `>83` (v84/v86 must take the v83 path).

### atlas-channel — `services/atlas-channel/atlas.com/channel/`
- `main.go:592-693` `produceWriters() []string` — append each new `…cb.<Op>Writer` const.
- `main.go:695-770` `produceHandlers() map[string]handler.MessageHandler` — add `hm[…sb.<Op>Handle] = handler.<Op>HandleFunc`.
- `main.go:772-777` `produceValidators()` — only `NoOpValidator` + `LoggedInValidator` exist; do **not** add new validators.
- Import aliases already present: `monstercb "…/monster/clientbound"`, `monstersb "…/monster/serverbound"`
  (main.go:100-101). Add `carnivalcb`/`carnivalsb` and reuse `charcb`/`charsb` aliases as needed.
- Clientbound Body helper pattern — `socket/writer/monster_spawn.go`:
  `func <Op>Body(<domain args>) packet.Encode { return func(l, ctx) func(options) []byte { … return monsterpkt.New<Op>(…).Encode(l, ctx)(options) } }`.
- Serverbound handler pattern — `socket/handler/monster_movement.go`:
  `func <Op>HandleFunc(l, ctx, wp writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) { return func(s, r, ro) { p := serverbound.<Op>{}; p.Decode(l, ctx)(r, ro); l.Debugf("[%s] read [%s]", p.Operation(), p.String()) } }`.
- **go.mod touched** when channel files change → `docker buildx bake atlas-channel` required.

### atlas-configurations — seed templates
- Five files: `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_95}_1.json`, `template_jms_185_1.json`.
- Top-level nesting: `socket.handlers[]` (each `{opCode, validator, handler[, options]}`) and `socket.writers[]` (each `{opCode, writer[, options]}`).
- Entries are **ordered by ascending opCode** — insert in sorted position.
- `opCode` strings are hex (e.g. `"0xBC"`). Every handler entry MUST carry a `validator` or `BuildHandlerMap` silently `continue`s it (`libs/atlas-opcodes/producer.go:47-50`).
- **go.mod**: atlas-configurations go.mod does NOT import atlas-packet; editing only JSON templates does not change go.mod. But CLAUDE.md still wants `docker buildx bake atlas-configurations` only if its go.mod changes — JSON-only edits do not require a bake. Confirm with `git diff --name-only -- '**/go.mod'`.

### packet-audit tool — `tools/packet-audit/`
- `go run ./tools/packet-audit matrix` regenerates `docs/packets/audits/STATUS.md` + `status.json`.
- `go run ./tools/packet-audit matrix --check` (CI gate): exit 0 clean, 1 blocker (conflict/drift/orphan/stale).
- `go run ./tools/packet-audit evidence pin --packet <id> --version <key> --ida "<FName>" --category TIER1-FIXTURE`
  — reads the **static export** `docs/packets/ida-exports/<version>…json` (NO live IDA), computes `decompile_sha256`,
  writes `docs/packets/evidence/<version>/<packet_dots>.yaml`. **Fails if `<FName>` is not in that export.**
- Verify marker (in `*_test.go`): `// packet-audit:verify packet=<pkg/dir/Struct> version=<key> ida=0x<addr>` — all three keys required.
  The `ida=` address and the evidence `ida.address`/hash both come from the **export**; live-IDA addresses must match the export or `--check` flags an orphan marker.
- Evidence categories: `OPAQUE TRUNCATION REPRESENTATION OP-MODE-PREFIX LOOP-EXCLUSIVE-BRANCH VERSION-ABSENT TIER1-FIXTURE`.
  Use `TIER1-FIXTURE` for implemented ops; `VERSION-ABSENT` for genuine version-absence n/a justifications.
- Tier-1 prefixes (`docs/packets/evidence/tiers.yaml`): `monster/`, `character/`, `field/`, `party/`, … — **all MOB ops are tier-1** → require linked byte-test + pinned evidence; flat-diff cannot promote them.
- Grading (`tools/packet-audit/internal/matrix/grade.go`): `n/a` = registry Absent + no conflicting report;
  `conflict` = registry Absent but Atlas report/route exists, OR registry Present + implemented but this version's template omits the route while another routes it.

### registry — `docs/packets/registry/<version>.yaml`
- Schema (README): `op, direction, opcode, fname, [fname_alts], provenance, [ida.address], [note]`; unique `(op, direction)` per file. `opcode` is decimal in the yaml.
- provenance: `csv-import` (frozen historical), `ida-discovered`, `manual` (needs IDA citation in `note`).

---

## 2. Per-version opcodes (decimal/hex) — clientbound ops with registry rows

| Op | fname | v83 | v84 | v87 | v95 | jms185 |
|---|---|---|---|---|---|---|
| RESET_MONSTER_ANIMATION | CMob::OnSuspendReset | 244/0xF4 | 250/0xFA | 260/0x104 | 292/0x124 | 261/0x105 |
| MOB_AFFECTED | CMob::OnAffected | 245/0xF5 | 245/0xF5 | 261/0x105 | 293/0x125 | 262/0x106 |
| MONSTER_SPECIAL_EFFECT_BY_SKILL | CMob::OnSpecialEffectBySkill | 247/0xF7 | 247/0xF7 | 263/0x107 | 295/0x127 | 264/0x108 |
| MOB_CRC_KEY_CHANGED | CMobPool::OnMobCrcKeyChanged | 249/0xF9 | 249/0xF9 | 265/0x109 | 297/0x129 | 266/0x10A |
| CATCH_MONSTER | CMob::OnCatchEffect | 251/0xFB | 251/0xFB | 267/0x10B | 299/0x12B | 268/0x10C |
| CATCH_MONSTER_WITH_ITEM | CMob::OnEffectByItem | 252/0xFC | 252/0xFC | 268/0x10C | 300/0x12C | 269/0x10D |
| MOB_SPEAKING ⚠ | CMob::OnMobSpeaking (registry mislabels v83/84/87 as OnIncMobChargeCount) | 254/0xFE | 254/0xFE | 270/0x10E | 301/0x12D | ABSENT |
| INC_MOB_CHARGE_COUNT ⚠ | CMob::OnIncMobChargeCount (registry mislabels v83/87) | 255/0xFF | 255/0xFF | 271/0x10F | 302/0x12E | ABSENT |
| MOB_SKILL_DELAY ⚠ | CMob::OnMobSkillDelay (registry mislabels v87 as OnMobAttackedByMob) | 256/0x100 | 256/0x100 | 272/0x110 | 303/0x12F | ABSENT |
| SET_TAMING_MOB_INFO | CWvsContext::OnSetTamingMobInfo | 48/0x30 | 48/0x30 | 48/0x30 | 47/0x2F | 45/0x2D |
| BRIDLE_MOB_CATCH_FAIL | CWvsContext::OnBridleMobCatchFail | 79/0x4F | 79/0x4F | 81/0x51 | 82/0x52 | 73/0x49 |
| MONSTER_BOOK_SET_CARD | CWvsContext::OnMonsterBookSetCard | 83/0x53 | 83/0x53 | 85/0x55 | 86/0x56 | 87/0x57 |
| MONSTER_BOOK_SET_COVER | CWvsContext::OnMonsterBookSetCover | 84/0x54 | 84/0x54 | 86/0x56 | 87/0x57 | 88/0x58 |
| MONSTER_CARNIVAL_START | CField_MonsterCarnival::OnEnter | 289/0x121 | 289/0x121 | 306/0x132 | 346/0x15A | 313/0x139 |
| MONSTER_CARNIVAL_OBTAINED_CP | CField_MonsterCarnival::OnPersonalCP | 290/0x122 | 290/0x122 | 307/0x133 | 347/0x15B | 314/0x13A |
| MONSTER_CARNIVAL_PARTY_CP | CField_MonsterCarnival::OnTeamCP | 291/0x123 | 291/0x123 | 308/0x134 | 348/0x15C | 315/0x13B |
| MONSTER_CARNIVAL_SUMMON | CField_MonsterCarnival::OnRequestResult | 292/0x124 | 292/0x124 | 309/0x135 | 349/0x15D | 316/0x13C |
| MONSTER_CARNIVAL_MESSAGE | CField_MonsterCarnival::OnRequestResult | 293/0x125 | 293/0x125 | 310/0x136 | 350/0x15E | 317/0x13D |
| MONSTER_CARNIVAL_DIED | CField_MonsterCarnival::OnProcessForDeath | 294/0x126 | 294/0x126 | 311/0x137 | 351/0x15F | 318/0x13E |
| MONSTER_CARNIVAL_LEAVE | CField_MonsterCarnival::OnShowMemberOutMsg | 295/0x127 | 295/0x127 | 312/0x138 | 352/0x160 | 319/0x13F |
| MONSTER_CARNIVAL_RESULT | CField_MonsterCarnival::OnShowGameResult | 296/0x128 | 296/0x128 | 313/0x139 | 353/0x161 | 320/0x140 |

Cluster-F version-tail clientbound (present only where shown; ABSENT elsewhere = n/a unless IDB proves otherwise):
- MOB_ESCORT_FULL_PATH: v87 273/0x111, v95 304/0x130 (v83/v84/jms ABSENT)
- MOB_ATTACKED_BY_MOB: v95 309/0x135 only
- MOB_ESCORT_RETURN_BEFORE: v95 307/0x133 only
- MOB_NEXT_ATTACK: v95 308/0x134 only

**Serverbound ops** (FIELD_DAMAGE_MOB, MOB_DAMAGE_MOB, MOB_DAMAGE_MOB_FRIENDLY, TOUCH_MONSTER_ATTACK, MONSTER_BOMB, MOB_TIME_BOMB_END, MOB_SKILL_DELAY_END, MONSTER_BOOK_COVER, MOB_CRC_KEY_CHANGED_REPLY, MOB_BANISH_PLAYER, MOB_DROP_PICKUP_REQUEST, MONSTER_CARNIVAL, MOB_ESCORT_COLLISION, MOB_ESCORT_STOP_END_REQUEST, MOB_REQUEST_ESCORT_INFO):
these have serverbound registry rows — **opcodes to be read per file during Phase 1** (the recon agent's serverbound grep was unreliable; read each `<version>.yaml` directly). Record them into `structures/<version>.md` alongside the byte layout.

---

## 3. Registry / fname gaps to resolve in Phase 1 (before coding)

1. **MOB_SPEAKING / INC_MOB_CHARGE_COUNT / MOB_SKILL_DELAY** — registry fnames cross-mislabeled in v83/v84/v87 (opcode-cluster off-by-one). Confirm correct fname per version against the IDB; fix the yaml row (`provenance: manual`, IDA citation in `note`).
2. **MONSTER_BOOK_COVER (serverbound)** — registry row exists but `fname` missing/empty. Derive send-site from the IDB; set `fname` (`provenance: ida-discovered`, address in `ida.address`).
3. **MOB_ESCORT_RETURN_STOP, MOB_ESCORT_RETURN_STOP_SAY** — not found in ANY registry. If the IDB shows them (likely v95-only), add `(op, direction, opcode, fname, provenance: ida-discovered)` rows; else document as non-existent and drop from scope with evidence.
4. **Export resolvability** — before any `evidence pin`, confirm each op's `fname` resolves in `docs/packets/ida-exports/<version>…json`. If a fname is absent from the export, `evidence pin` fails; that op needs a re-export (task-081 playbook) — surface it as a blocker, do not fake the hash.

---

## 4. SET_TAMING_MOB_INFO dedup (confirmed state)

- `libs/atlas-packet/character/clientbound/set_taming_mob_info.go` EXISTS (fields `characterId, level, exp, tiredness, levelUp`; all `WriteInt` + `WriteBool`).
- `set_taming_mob_info_test.go` EXISTS but has **NO `// packet-audit:verify` markers** and there is **NO evidence record** in any `docs/packets/evidence/*/`.
- task-092 work for this op: add 5 verify markers + pin 5 evidence records + regenerate matrix. Do **not** re-implement the encoder. (The op is `clientbound` but is owned by `CWvsContext`; encoder has no `ctx` version branch — confirm v95/jms layout is identical in Phase 1; if a version differs, that is a wire-fix in its own commit.)

---

## 5. Current matrix state

All targeted MOB/MONSTER cells are `❌ incomplete` across all 5 versions (STATUS.md, verified 2026-06-13). None verified/partial yet. Target end-state: every applicable cell `✅ verified`, genuine absences `⬜ n/a` with `VERSION-ABSENT` evidence, zero `🟥 conflict`.

---

## 6. IDA multi-instance ports (Phase 1 only)

v83=13337, v87=13338, v95=13339, jms=13340, v84=13341 (PRD §4.2). One IDB loaded at a time — the user switches the active IDB; batch all derivations for a version before moving on. Subagents can reach IDA-MCP (memory: `reference_ida_harvest_subagents`).

---

## 7. Verification gates (CLAUDE.md)

`go test -race ./...` + `go vet ./...` clean in every changed module; `go build ./...` for atlas-channel;
`tools/redis-key-guard.sh` clean; `docker buildx bake atlas-channel` (channel go.mod is reached transitively — bake when channel code changes); `go run ./tools/packet-audit matrix --check` exit 0.
Run `go test`/`vet` with `GOWORK` as configured by the module; redis-key-guard wants `GOWORK=off` per memory.
