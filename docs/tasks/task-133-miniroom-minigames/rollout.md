# task-133 deployment notes — Miniroom Minigames (Omok / Match Cards)

## Live-tenant config PATCH (REQUIRED)

Seed templates apply only at tenant creation. Existing tenants do **not** pick
up the Task 20 template changes — the atlas-channel handler/writer projection
does not hot-reload. Without a PATCH the client actions are silently dropped
("unhandled message op 0xXX" at info) or the writer resolves an absent mode
byte and the client crashes. For every live tenant: PATCH its socket
configuration, then restart atlas-channel.

Three pieces are added/merged per tenant (pull the exact rows for the tenant's
version from the tables below):

1. **Serverbound handler** `CharacterInteractionHandle` — merge the
   `CREATE`/`VISIT`/`CHAT`/`EXIT` lifecycle ops and the 14 `MEMORY_GAME_*`
   serverbound ops into `options.operations`. Handler carries
   `"validator": "LoggedInValidator"`. (opCode is the tenant's existing
   `PLAYER_INTERACTION` serverbound opcode — do not change it.)
2. **Clientbound writer** `CharacterInteraction` — merge the `MEMORY_GAME_*`
   clientbound ops (incl. `MEMORY_GAME_RESULT`) into `options.operations`. The
   `INVITE`/`INVITE_RESULT`/`ENTER`/`ENTER_RESULT`/`LEAVE`/`UPDATE_MERCHANT`
   rows already exist on live tenants; leave them. (opCode is the tenant's
   existing `PLAYER_INTERACTION` clientbound opcode.)
3. **New clientbound writer** `MiniRoom` (`UPDATE_CHAR_BOX`, the field balloon)
   — add the whole entry; it does not exist on live tenants.

### PLAYER_INTERACTION / UPDATE_CHAR_BOX opcodes per version

Source of truth = `docs/packets/registry/*.yaml` (v92 = `MapleStory Ops` CSV;
see `seed-unverified-notes.md`).

| Version  | SB handler opCode | CB writer opCode | `MiniRoom` opCode |
|----------|-------------------|------------------|-------------------|
| gms_v83  | 0x7B              | 0x13A            | 0xA5              |
| gms_v84  | 0x7D              | 0x141            | 0xA8              |
| gms_v87  | 0x81              | 0x14B            | 0xB0              |
| gms_v92  | 0x8D              | 0x16D            | 0xB6              |
| gms_v95  | 0x90              | 0x175            | 0xB8              |
| jms_v185 | 0x7C              | 0x153            | 0xA3              |

### Handler `CharacterInteractionHandle` — MEMORY_GAME rows to merge (GMS: v83/84/87/92/95)

`"validator": "LoggedInValidator"`. Lifecycle rows `CREATE:0, VISIT:4, CHAT:6,
EXIT:10` (already present on live tenants) plus:

```
"MEMORY_GAME_ASK_TIE": 50,
"MEMORY_GAME_TIE_ANSWER": 51,
"MEMORY_GAME_FORFEIT": 52,
"MEMORY_GAME_ASK_RETREAT": 54,
"MEMORY_GAME_RETREAT_ANSWER": 55,
"MEMORY_GAME_EXIT_AFTER_GAME": 56,
"MEMORY_GAME_CANCEL_EXIT_AFTER_GAME": 57,
"MEMORY_GAME_READY": 58,
"MEMORY_GAME_UNREADY": 59,
"MEMORY_GAME_EXPEL": 60,
"MEMORY_GAME_START": 61,
"MEMORY_GAME_SKIP": 63,
"MEMORY_GAME_MOVE_STONE": 64,
"MEMORY_GAME_FIP_CARD": 68
```

**jms_v185 handler — the entire mode ≥ 14 block is shifted −3** (verified
byte-for-byte against the jms IDB; do NOT copy the GMS values):

```
"MEMORY_GAME_ASK_TIE": 47,
"MEMORY_GAME_TIE_ANSWER": 48,
"MEMORY_GAME_FORFEIT": 49,
"MEMORY_GAME_ASK_RETREAT": 51,
"MEMORY_GAME_RETREAT_ANSWER": 52,
"MEMORY_GAME_EXIT_AFTER_GAME": 53,
"MEMORY_GAME_CANCEL_EXIT_AFTER_GAME": 54,
"MEMORY_GAME_READY": 55,
"MEMORY_GAME_UNREADY": 56,
"MEMORY_GAME_EXPEL": 57,
"MEMORY_GAME_START": 58,
"MEMORY_GAME_SKIP": 60,
"MEMORY_GAME_MOVE_STONE": 61,
"MEMORY_GAME_FIP_CARD": 65
```

Base lifecycle modes (`CREATE:0, VISIT:4, CHAT:6, EXIT:10`) are **unshifted**
in jms — same as GMS.

### Writer `CharacterInteraction` — MEMORY_GAME rows to merge (GMS: v83/84/87/92/95)

Merge into the existing `options.operations` (keep the existing
`INVITE/INVITE_RESULT/ENTER/ENTER_RESULT/LEAVE/UPDATE_MERCHANT` rows):

```
"MEMORY_GAME_ASK_TIE": 50,
"MEMORY_GAME_TIE_ANSWER": 51,
"MEMORY_GAME_ASK_RETREAT": 54,
"MEMORY_GAME_RETREAT_ANSWER": 55,
"MEMORY_GAME_READY": 58,
"MEMORY_GAME_UNREADY": 59,
"MEMORY_GAME_START": 61,
"MEMORY_GAME_RESULT": 62,
"MEMORY_GAME_SKIP": 63,
"MEMORY_GAME_MOVE_STONE": 64,
"MEMORY_GAME_FIP_CARD": 68
```

**jms_v185 writer — −3 shift:**

```
"MEMORY_GAME_ASK_TIE": 47,
"MEMORY_GAME_TIE_ANSWER": 48,
"MEMORY_GAME_ASK_RETREAT": 51,
"MEMORY_GAME_RETREAT_ANSWER": 52,
"MEMORY_GAME_READY": 55,
"MEMORY_GAME_UNREADY": 56,
"MEMORY_GAME_START": 58,
"MEMORY_GAME_RESULT": 59,
"MEMORY_GAME_SKIP": 60,
"MEMORY_GAME_MOVE_STONE": 61,
"MEMORY_GAME_FIP_CARD": 65
```

### New writer `MiniRoom` — full entry to add (per version)

```json
{ "opCode": "<MiniRoom opCode from table>", "writer": "MiniRoom", "options": {} }
```

Concretely: gms_v83 `0xA5`, gms_v84 `0xA8`, gms_v87 `0xB0`, gms_v92 `0xB6`,
gms_v95 `0xB8`, jms_v185 `0xA3`.

### Example PATCH body (gms_v83)

PATCH the tenant's `socket` configuration; splice these three deltas into the
existing `handlers` / `writers` arrays (merge into the matching existing
entries; add `MiniRoom` new):

```json
{
  "handler CharacterInteractionHandle.options.operations += ": {
    "MEMORY_GAME_ASK_TIE": 50, "MEMORY_GAME_TIE_ANSWER": 51,
    "MEMORY_GAME_FORFEIT": 52, "MEMORY_GAME_ASK_RETREAT": 54,
    "MEMORY_GAME_RETREAT_ANSWER": 55, "MEMORY_GAME_EXIT_AFTER_GAME": 56,
    "MEMORY_GAME_CANCEL_EXIT_AFTER_GAME": 57, "MEMORY_GAME_READY": 58,
    "MEMORY_GAME_UNREADY": 59, "MEMORY_GAME_EXPEL": 60,
    "MEMORY_GAME_START": 61, "MEMORY_GAME_SKIP": 63,
    "MEMORY_GAME_MOVE_STONE": 64, "MEMORY_GAME_FIP_CARD": 68
  },
  "writer CharacterInteraction.options.operations += ": {
    "MEMORY_GAME_ASK_TIE": 50, "MEMORY_GAME_TIE_ANSWER": 51,
    "MEMORY_GAME_ASK_RETREAT": 54, "MEMORY_GAME_RETREAT_ANSWER": 55,
    "MEMORY_GAME_READY": 58, "MEMORY_GAME_UNREADY": 59,
    "MEMORY_GAME_START": 61, "MEMORY_GAME_RESULT": 62,
    "MEMORY_GAME_SKIP": 63, "MEMORY_GAME_MOVE_STONE": 64,
    "MEMORY_GAME_FIP_CARD": 68
  },
  "writers += ": { "opCode": "0xA5", "writer": "MiniRoom", "options": {} }
}
```

(The pseudo-keys above name the target array element + merge intent; apply the
merge with your config-editing tooling — the real PATCH is the full `socket`
document with these entries spliced in.)

## Rollout order

1. **libs** (`libs/atlas-packet` — new mini-room clientbound/serverbound
   codecs) land first; they are additive and consumed by atlas-channel.
2. **atlas-mini-games** deploy (new service; must be present in
   `.github/config/services.json` AND `docker-bake.hcl` `go_services`, k8s
   readiness probe `/api/readyz`). New service, no old consumers to break.
3. **atlas-channel** deploy (mini-room handlers/writers + Kafka wiring).
4. **Tenant config PATCH** (the three deltas above), per live tenant.
5. **atlas-channel restart** — mandatory; the handler/writer projection does
   not hot-reload, so the PATCH is inert until channel pods restart.

## In-game acceptance checklist (v83 tenant, PRD §10 verbatim)

- [ ] On a live v83 tenant: create an Omok room with item 4080000-family in inventory (each of title, private+password verified), balloon appears; creation without the item fails with error 6; creation in a `fieldLimit & 0x80` map fails with error 11; creation with an open chalkboard fails with error 13.
- [ ] Second character joins via balloon (password enforced), readies; owner starts; a full Omok game plays to five-in-a-row; winner/loser records increment by exactly 1; tie flow, forfeit, retreat (accept + decline), skip, expel, exit-after-game, and owner-leave-closes-room all behave per FR-5/7/8.
- [ ] Match Cards playable end-to-end at all three board sizes with correct match/turn/score semantics and record updates.
- [ ] Disconnect mid-game forfeits and tears down the room; opponent receives win + correct leave packet.
- [ ] Records survive relog and are correct in the room UI encoding (marker 1, W/T/L, room-scoped score).
- [ ] `GET /api/characters/{id}/game-records` returns tenant-scoped records per §5.
- [ ] All six seed templates contain the new handler (with validator), writer, and `operations` entries; matrix/config checks pass; live-tenant PATCH runbook committed in the task docs.
- [ ] New clientbound packets have byte-fixture tests (full per-mode bodies — mode-byte enumeration alone is not verification) for every version with an available IDB; remaining versions wired and flagged unverified.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-mini-games, atlas-channel, and libs/atlas-packet; `docker buildx bake atlas-mini-games atlas-channel` clean; `tools/redis-key-guard.sh` clean.
- [ ] atlas-mini-games present in `.github/config/services.json` AND `docker-bake.hcl` `go_services`; k8s manifest readiness probe is `/api/readyz`.
- [ ] Code review (plan-adherence + backend-guidelines) run before PR.

## Rollout caveats (grounding gaps)

- **gms_v92 modes are UNVERIFIED (derived).** No v92 IDB exists and v92 is
  outside `matrix.VersionKeys`, so packet-audit does not manage it. The v92
  MEMORY_GAME mode values were copied from the GMS v83 enum (bracketed by the
  IDA-verified GMS neighbours v87 and v95, both = v83) and the opcodes came
  from the `MapleStory Ops` CSV v92 column (cross-validated against the IDA
  registries for the other versions, but not IDA-confirmed for v92
  specifically). Treat a v92 rollout as best-effort until a v92 IDB can
  confirm the sub-modes. See `seed-unverified-notes.md`.
- **Tier-1 fixture coverage is v83 + v95 only.** The clientbound mini-room
  packets carry full per-mode byte-fixture tests for gms_v83 and gms_v95. The
  room-enter (and other clientbound) matrix cells for gms_v84 / gms_v87 /
  jms_v185 beyond v83/v95 are **wired and mode-verified against their IDBs but
  not tier-1 fixture-claimed** — they are not asserted at the byte-fixture
  tier in the coverage matrix. Plan any v84/v87/jms in-game acceptance with
  that in mind.

## Verification evidence (gates, task-22 Step 1)

All gates run from the worktree root, exit 0.

**mini-games — `go test -race ./... && go vet ./... && go build ./...`** → `EXIT=0`
```
ok  atlas-mini-games/game        (cached)
ok  atlas-mini-games/game/matchcards (cached)
ok  atlas-mini-games/game/omok   (cached)
ok  atlas-mini-games/kafka/consumer/character (cached)
ok  atlas-mini-games/record      (cached)
```

**atlas-channel — `go test -race ./... && go vet ./... && go build ./...`** → `EXIT=0`
(all packages `ok`/`[no test files]`; no failures.)

**libs/atlas-packet — `go test -race ./... && go vet ./...`** → `EXIT=0`
(all packages `ok`/`[no test files]`; incl. `interaction`, `interaction/clientbound`,
`interaction/serverbound`.)

**`docker buildx bake atlas-mini-games atlas-channel`** → `BAKE_EXIT=0`
```
#130 naming to docker.io/library/atlas-mini-games:local done
#130 unpacking to docker.io/library/atlas-mini-games:local done
#130 naming to docker.io/library/atlas-channel:local done
```

**`tools/redis-key-guard.sh`** → `EXIT=0` (scanned all services incl.
atlas-mini-games and atlas-channel; no keyed go-redis violations.)

**packet-audit checkers** (`go run ./tools/packet-audit <sub> --check`):
```
dispatcher-lint: clean                                             (DL_EXIT=0)
matrix --check                                                     (MX_EXIT=0)
operations check OK (0 absent-writer note(s))                      (OP_EXIT=0)
fname-doc check OK (220 structs without an audit report carry no fname)  (FN_EXIT=0)
```

`fname-doc --check` is clean (exit 0) — the pre-existing
`MiniRoomBalloonRemove` (`#Remove` suffix) drift flagged in the ledger was
already resolved by Task 18; no additional fix was needed in Task 22.
