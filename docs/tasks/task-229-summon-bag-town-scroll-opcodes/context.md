# task-229 — Implementation Context

Companion to `plan.md`. Everything here was read out of the repo or the IDA
session server on 2026-08-14; nothing is inferred. Line numbers are as of the
branch tip (`task-229-summon-bag-town-scroll-opcodes`).

---

## 1. What the task actually is

The player-facing bug is small: summoning sacks and town/return scrolls do
nothing on several tenant versions. The cause is that their **serverbound
opcodes are not bound in the seed templates**, so atlas-channel never dispatches
them. The server-side feature is complete and version-agnostic.

The task is large because the PRD also demands that each affected coverage-matrix
cell be promoted with pinned evidence — and the two ops turn out to have **no
codec linkage and no IDA evidence on any of the ten columns**, including the
"reference" ones. So the template fix is ~15 JSON lines; the verification is 20
IDA harvests.

## 2. Key files

### Production code — read, do not change

| Path | Why it matters |
|---|---|
| `libs/atlas-packet/inventory/serverbound/item_use.go` | The one definition of the shared 3-field body: `Encode4(updateTime) + Encode2(slot) + Encode4(itemId)` (`:45-59`). Handler-name constants live at `:14-17`. **Not modified by this task.** |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go` | `CharacterItemUseTownScrollHandleFunc` (:27) and `CharacterItemUseSummonBagHandleFunc` (:45); both decode the shared `ItemUse`. Untouched. |
| `services/atlas-channel/atlas.com/channel/main.go:940,952` | Both handler names are already registered. Untouched. |
| `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:490,658` | `ConsumeTownScroll` / `ConsumeSummoningSack`, dispatched **by item id**, not by handler name. Untouched. |

### The precedent to copy

`libs/atlas-packet/inventory/serverbound/lottery_item_use.go` +
`lottery_item_use_test.go` (task-131) are the in-repo model for a per-op codec in
this family: discrete struct, one `packet-audit:fname` marker, one
`packet-audit:verify` line per version, one evidence record per version, and
matching audit reports at `docs/packets/audits/<v>/InventoryLotteryItemUse.json`.

### Tooling the plan depends on

| Path | What it decides |
|---|---|
| `tools/packet-audit/cmd/run.go` ~`:2180-2210` | `candidatesFromFName` — the fname → `{struct, pkg, dir}` switch. A new serverbound op that is not in this switch cannot get a report. `sub_70D8DE`/`sub_719DD9` at `:2182-2193` are the "primary fname is an unnamed sub" precedent. |
| `tools/packet-audit/cmd/run.go:3649-3675` | `locateAtlasFile` — finds `type <name> struct` under `<pkg>/serverbound/`. |
| `tools/packet-audit/internal/matrix/grade.go:126-133` | Path B: the registry's `packet:` field resolves the packet id when no report exists. |
| `tools/packet-audit/internal/matrix/grade.go:206-215` | **No-report promotion:** `marker.Found && hasEvidence && evidence.Fresh` ⇒ `verified`. |
| `tools/packet-audit/internal/matrix/grade.go:219` | The literal `"no audit report"` note both target ops currently carry on nine of ten columns. |
| `tools/packet-audit/cmd/matrix.go:176-183` | "dangling evidence" `--check` failure, **exempted** when `registryDeclaresPacket` (`:392-403`) is true. |
| `tools/packet-audit/cmd/na_consistency.go:166-200` | The n-a gate iterates **family members only**. |
| `tools/packet-audit/internal/matrix/model.go:18-23` | `ExportPath` — jms_v185 maps to `gms_jms_185.json`. |
| `tools/packet-audit/internal/opregistry/opregistry.go:33-50` | `Entry` schema; `Packet` is the optional `packet:` key. |
| `docs/packets/evidence/tiers.yaml:54` | `packet_prefixes: inventory/` ⇒ both new packet ids are tier-1. |
| `libs/atlas-packet/test/…` | `Variants` (12 tenant variants incl. GMS v48/61/72/79/83/84/86/87/92/95 and JMS v185), `CreateContext`, `RoundTrip`. |

## 3. Measured baseline (2026-08-14)

- `go run ./tools/packet-audit matrix --check` → **exit 0**. The committed
  `STATUS.md` / `status.json` are current. Any later non-zero exit is caused by
  this task's changes.
- `USE_SUMMON_BAG` and `USE_RETURN_SCROLL` rows in `status.json`: **no `packet`
  key**, `tier1: false`, `gms_v48: n-a (opcode -1)`, all nine other columns
  `incomplete — "no audit report"`.
- `USE_ITEM`, `USE_UPGRADE_SCROLL`, `PET_FOOD` on `gms_v92`: `partial —
  "tier-1: needs byte-fixture test to verify"`. Reports and export entries
  already exist; only a marker + pinned evidence is missing.
- Loading each `docs/packets/ida-exports/*.json` and testing key membership:
  `CWvsContext::SendMobSummonItemUseRequest`, `…SendPortalScrollUseRequest`,
  `…SendReturnScrollUseRequest`, `sub_955499` and `sub_841AA5` are **absent from
  every export**. `SendStatChangeItemUseRequest`, `SendPetFoodItemUseRequest` and
  `SendUpgradeItemUseRequest` are present in all ten.
- All ten target IDBs are loaded and adoptable on the session server.

## 4. Template state before the change

| template | ItemUse | SummonBag | TownScroll | Scroll | PetFood |
|---|---|---|---|---|---|
| gms_12 | — | — | — | — | — |
| gms_48 | `0x41` | `0x3B` *(no `fname`)* | — | `0x42` | `0x3C` |
| gms_61 | `0x43` | `0x46` | `0x4E` | `0x4F` | `0x47` |
| gms_72 | `0x47` | `0x4A` | `0x54` | `0x55` | `0x4B` |
| gms_79 | `0x46` | `0x49` | `0x53` | `0x54` | `0x4A` |
| gms_83 | `0x48` | `0x4B` | `0x55` | `0x56` | `0x4C` |
| gms_84 | `0x48` | `0x4B` | `0x55` | `0x56` | `0x4C` |
| gms_87 | `0x4B` | **—** | **—** | `0x59` | `0x4F` |
| gms_92 | **—** | **—** | **—** | **—** | **—** |
| gms_95 | `0x4E` | **—** | **—** | `0x5D` | `0x52` |
| jms_185 | `0x40` | **—** | **—** | `0x4E` | `0x44` |

gms_92's entire item-use block below `0x54` is unrouted — not just the two target
ops. Its bound opcodes in that range are `0x4A OwlWarpHandle`,
`0x4D CompartmentSortHandle`, `0x4E CharacterInventoryMoveHandle`,
`0x54 MountFoodHandle`, `0x56 CharacterCashItemUseHandle`,
`0x58 MonsterCatchItemUseHandle`, `0x59 CharacterSkillBookUseHandle`,
`0x5A ShopScannerItemUseHandle`, `0x5B TeleportRockUseHandle` — so `0x4F`,
`0x52`, `0x53`, `0x5C` and `0x5D` are all free.

## 5. Decisions carried in from design.md

| id | decision |
|---|---|
| D1 | Per-op wrapper structs embedding `ItemUse`, plus generated audit reports. Aliasing the shared packet id is rejected: it would pin summon-bag and return-scroll cells to the *potion* sender's decompile. |
| D2 | The IDA harvest is the cost centre and is unavoidable — locate by `COutPacket::COutPacket(&pkt, <opcode>)`, name the sub, splice-only into the export, then report + fixture + pin. |
| D3 | Also bind `PetFoodHandle` on gms_92 (`0x53`). `ShopScannerItemUseHandle`, `CharacterItemUseLotteryHandle` and gms_12 stay out of scope. |
| D4 | Verify **all ten** columns, not just the five whose bindings change, so both op-rows finish complete rather than mixed. |
| D5 | Resolve gms_48 in-task: confirm `0x3B` against the v61 named twin, backfill `USE_SUMMON_BAG`, and either register or evidence-as-absent the return scroll. |

## 6. Traps that have already cost time on this codebase

- **`packet-audit export -splice` is banned.** It round-trips the whole export
  through a struct that drops unrecognized JSON fields (`region`, `note`) and
  reindents, silently corrupting ~20 entries you never touched. Hand-edit the
  export instead and check `git diff` scope.
- **`-ida-url` defaults to a dead port (13337)** in eight places. It *answers*,
  with a different binary's data, so a harvest looks well-formed and is wrong.
  Always pass `-ida-url http://192.168.20.3:8745/mcp -ida-database <session>`.
- **Session ids shift.** Resolve by binary filename from `idb_list` every time;
  `select_instance` / `-ida-port` are dead.
- **The obvious symbol is a mislabel.** `CWvsContext::SendMapTransferItemUseRequest`
  is the *teleport-rock* sender, not the return-scroll sender, on v61/v72/v79 —
  corrected on task-124 and recorded in `gms_v72.yaml:2414`, `gms_v79.yaml:2798`,
  `gms_v61.yaml:2473`. Expect the same trap at v48.
- **v79 mislabels the summon-bag sender** as `SendEngagementRequest`
  (`gms_v79.yaml:2918`). Read the opcode from the body.
- **v48's serverbound table is not a shifted copy of v61's.** v48 puts `USE_ITEM`
  at `0x41`, *after* `USE_SKILL_BOOK` `0x40`; v61+ put it first. Positional
  inference is invalid on that column.
- **A missing validator silently drops a handler** at load time. Every new
  template entry must name `LoggedInValidator`, which already exists in all five
  files.
- **`tools/lint.sh --check` false-fails without nvm on PATH**, and can contend on
  the golangci-lint lock across worktrees.
- **`evidence pin` fails "not in export"** whenever the committed export predates
  an IDB rename; every marker is then discarded as an orphan and no cell moves.

## 7. Verification commands (full set)

```bash
# packet tooling
cd tools/packet-audit && go test ./... && go vet ./... && cd ../..
cd libs/atlas-packet && go build ./... && go test -race ./... && go vet ./... && cd ../..

# packet-audit CI gates (.github/workflows/packet-matrix.yml)
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit doc-freshness --check
go run ./tools/packet-audit gate-check --check
go run ./tools/packet-audit matrix --check

# repo guards
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

No service `go.mod` is touched, so no `docker buildx bake` target is implicated —
confirm with `git diff --name-only main... | grep 'go.mod'` before skipping it.

## 8. Known residual, deliberately not closed here

- `ShopScannerItemUseHandle` is unbound on gms_87 and jms_185;
  `CharacterItemUseLotteryHandle` is unbound on gms_92. Same class of gap,
  explicitly out of scope per PRD §2.
- `template_gms_12_1.json` carries no item-use handlers at all and is not a
  matrix column. It needs a version pass, not a patch.
- Seed-template changes do not reach already-provisioned tenants. Whether that is
  handled by a reseed or a live config PATCH is the deployer's call (PRD §10) and
  must be stated in the PR body.
