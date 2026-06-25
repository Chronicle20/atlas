# Task-110 Interaction Packet-Fixture Verification — Context

Companion to `plan.md`. Key files, decisions, and corrections an executor needs
before touching a cell. All facts below are resolved from source in this worktree,
not from memory.

## Goal in one line

Drive the **12 remaining `incomplete` serverbound cells** in the `interaction`
family to `verified` (✅) across all applicable versions, landing each as the three
coupled artifacts (byte-fixture marker + pinned evidence + audit report), with the
matrix regenerated.

## The 12 cells (confirmed against `docs/packets/audits/status.json`)

| Packet (`interaction/serverbound/…`) | fname | Incomplete versions | Verified | Class |
|---|---|---|---|---|
| `InteractionOperationInvite` | `CField::SendInviteTradingRoomMsg` | v83, v87, jms | v84, v95 | **A** |
| `InteractionOperationMemoryGameTieAnswer` | `CMemoryGameDlg::OnTieRequest` | v84 | v83, v87, v95, jms | **E** |
| `InteractionOperationMerchantPutItem` | `CPersonalShopDlg::PutItem#Merchant` | v83, v84, v87, jms | v95 | **E-arm** |
| `InteractionOperationMerchantRemoveItem` | `CPersonalShopDlg::MoveItemToInventory#Merchant` | v83, v84, v87, jms | v95 | **E-arm** |

These are the ONLY incomplete interaction cells. Clearing all four packets clears
the family's serverbound coverage (the clientbound dispatcher arms were graduated to
✅ in task-096; do not touch them).

## Export-presence finding (CONFIRMED by grep, drives the work-classes)

`grep -c` over `docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v95,gms_jms_185}.json`:

| fname | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|
| `SendInviteTradingRoomMsg` (Invite) | 1 | 1 | 1 | 1 | 1 |
| `OnTieRequest` (TieAnswer) | 1 | **0** | 1 | 1 | 1 |
| `BuyItem#Merchant` (MerchantBuy — already ✅ all) | 1 | 1 | 1 | 1 | 1 |
| `PutItem#Merchant` (MerchantPutItem) | **0** | **0** | **0** | 1 | **0** |
| `MoveItemToInventory#Merchant` (MerchantRemoveItem) | **0** | **0** | **0** | 1 | **0** |

Three facts fall out:
1. **`BuyItem#Merchant` present on all five versions + `MerchantBuy` ✅ everywhere** ⇒
   the entrusted-merchant feature exists pre-v95 in every binary, and the exporter's
   `#case`-arm splitting *does* capture merchant send sites. The missing
   `PutItem#Merchant`/`MoveItemToInventory#Merchant` arms on v83/v84/v87/jms are a
   **harvest-depth gap**, not a missing feature → producible by re-harvest + surgical
   splice, never `n-a`.
2. The base functions `CPersonalShopDlg::PutItem` and `::MoveItemToInventory` are
   present (named) on every version → this is *arm extraction from an already-named
   function*, not byte-signature naming of an unnamed sub_XXXX.
3. `OnTieRequest` absent on v84 only, but ✅ on the byte-identical v83 twin → single
   true Class-E (name the present-but-unnamed v84 function against the v83 twin, splice).

## Work-classes (effort tiers)

- **Class A — report-gen only, no live IDA.** Client fn already in the committed
  export; cell is `incomplete` only because the per-version report was never copied
  in. Cells: **Invite v83/v87/jms (3)**.
- **Class E — fn absent from one version's export; name-and-splice.** Open the v84
  IDB, confirm `OnTieRequest` present-but-unnamed, name it (twin-match v83), surgically
  splice the absent entry into `gms_v84.json`. Then it becomes Class A. Cell:
  **TieAnswer v84 (1)**.
- **Class E-arm — named base fn present, `#Merchant` arm not in export; re-harvest the
  arm + splice.** For each of v83/v84/v87/jms: decompile the base fn, locate the
  entrusted-merchant send arm, confirm its write order matches the Atlas codec, splice
  the `#Merchant` case-keyed entry (absent-only), then Class A. Cells:
  **MerchantPutItem v83/v84/v87/jms (4) + MerchantRemoveItem v83/v84/v87/jms (4) = 8**.

## The codecs (all in `libs/atlas-packet/interaction/serverbound/`)

Single Go module `libs/atlas-packet`. `services/atlas-channel` only *consumes* them.
**Only `libs/atlas-packet` is touched (test files only**, absent a proven wire delta).
No `go.mod` changes → **no `docker buildx bake` required** (the bake gate triggers
only on `go.mod` changes).

All four structs have **uniform (un-gated) Encode/Decode — no `MajorVersion()` gate**:

- `operation_invite.go` — `OperationInvite`: `WriteInt(targetCharacterId)` (1 field).
- `operation_memory_game_tie_answer.go` — `OperationMemoryGameTieAnswer`:
  `WriteBool(response)` (1 field).
- `operation_merchant_put_item.go` — `OperationMerchantPutItem`:
  `WriteByte(inventoryType) · WriteInt16(slot) · WriteShort(quantity) · WriteShort(set) · WriteInt(price)`.
  Byte-identical to the all-versions-✅ `OperationPersonalStorePutItem`.
- `operation_merchant_remove_item.go` — `OperationMerchantRemoveItem`:
  `WriteShort(index)` (1 field). Identical to the ✅ `OperationPersonalStoreRemoveItem`.

Because the codecs are uniform, the per-version work is **adding marker rows to the
existing tests + producing export/report/evidence artifacts** — NOT editing the wire
codec (absent a proven delta; §6 of design covers the fix-first contingency).

## The byte-fixture pattern (house style for this family)

A "verified" interaction serverbound cell uses the existing **`pt.Variants` table
RoundTrip test** plus, where the design's "full-body byte test" rule needs a concrete
byte assertion, a `Test…Bytes` function. Reference precedents IN THIS PACKAGE:

- **`operation_merchant_buy_test.go`** is the exact E-arm precedent: a `RoundTrip`
  test carrying v84/v87/jms markers PLUS a `TestOperationMerchantBuyBytes` that pins
  the hex wire bytes, carrying the v83/v95 markers, with a comment explaining the
  `#Merchant` arm shares the base `CPersonalShopDlg::BuyItem` (op 0x22 vs 0x17) and
  carries the same body in v83 and v95. **Copy this shape for MerchantPutItem and
  MerchantRemoveItem.**
- **`operation_personal_store_put_item_test.go`** is the all-5-versions-✅ uniform-codec
  precedent: a single RoundTrip test with all five `packet-audit:verify` markers.
- `libs/atlas-packet/party/clientbound/invite_test.go` (`TestInviteByteOutput`) is the
  cross-package byte-count reference cited by the playbook — useful for version-divergent
  packets, but the interaction codecs are uniform so a hex-pin (MerchantBuy style) is the
  closer model.

`pt.Variants` index → version map (from `libs/atlas-packet/test/context.go`):
`[0]=GMS v28, [1]=GMS v83, [2]=GMS v87, [3]=GMS v95, [4]=JMS v185, [5]=GMS v84, [6]=GMS v86`
(v84/v86 appended, not inserted, so positional refs stay valid). The RoundTrip loop
covers all of them; markers map by `version=` key, not by index.

## Existing marker coverage (what's already in the test files)

- `operation_invite_test.go` — markers for **gms_v95, gms_v84** (RoundTrip). Need to ADD
  gms_v83, gms_v87, jms_v185.
- `operation_memory_game_tie_answer_test.go` — markers for **gms_v95, gms_v87, gms_v83,
  jms_v185** (RoundTrip). Need to ADD gms_v84.
- `operation_merchant_put_item_test.go` — marker for **gms_v95** only (RoundTrip). Need
  to ADD gms_v83, gms_v84, gms_v87, jms_v185 (+ a `…Bytes` hex pin, MerchantBuy style).
- `operation_merchant_remove_item_test.go` — marker for **gms_v95** only (RoundTrip).
  Need to ADD the same four (+ a `…Bytes` hex pin).

## Tooling — exact invocations (from `VERIFYING_A_PACKET.md` §7–10)

**Report-gen (ROOT command, deterministic, no live IDA):**
```
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_<v>.json \
  -ida-source docs/packets/ida-exports/<export>.json \
  -output /tmp/rpt
```
Writes `/tmp/rpt/<version>/<Writer>.{json,md}`. Copy the specific
`Interaction<Struct>.{json,md}` into `docs/packets/audits/<version>/`.
`qualifiedWriterName` = TitleCase(pkg)+struct → e.g. `Interaction`+`OperationInvite`
= `InteractionOperationInvite.json` (confirmed against existing v95 reports).

**Evidence pin (tier-1):**
```
go run ./tools/packet-audit evidence pin --packet interaction/serverbound/<Struct> \
  --version <key> --ida "<FName-with-#suffix>" --category TIER1-FIXTURE
```
Writes `docs/packets/evidence/<version>/interaction.serverbound.<Struct>.yaml`.
`--ida` is the function name **exactly as the export key spells it** (carry the
`#Merchant` suffix where applicable). After it succeeds, hand-add the `verifies:` line:
```
verifies:
    - libs/atlas-packet/interaction/serverbound/<test file>#<TestName>
```
(Precedent: `docs/packets/evidence/gms_v83/interaction.serverbound.InteractionOperationMerchantBuy.yaml`.)

**Matrix regen + check:**
```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```
`--check` may exit 1 from the pre-existing registry-seed conflict backlog. Bar = "no
NEW problems": zero orphan/dangling/stale/drift lines mentioning an interaction packet,
and the conflict count must not increase. Also run the fname-doc / operations `--check`
variants and apply the same bar.

## Export-splice discipline (Class E / E-arm)

The export is **NON-idempotent** — never overwrite a committed export (re-running
`packet-audit export` drifts ~150 unrelated keys). Surgical splice only:

1. Harvest to a **temp file** with `-prior-export "" -pending <roster.md> -descent-depth 12`,
   `-ida-url http://<host>:<port>/mcp -ida-port <port>` pointing at the **target** version.
2. Splice **only** the needed entry into the committed export (absent-only):
   - TieAnswer v84: add `CMemoryGameDlg::OnTieRequest`.
   - Merchant arms: add `CPersonalShopDlg::PutItem#Merchant` / `::MoveItemToInventory#Merchant`
     for the target version. Leave the base `PutItem` / `MoveItemToInventory` entries untouched.
3. Strip any `COutPacket`-delegate harvest artifact (`{op: Delegate, ref: COutPacket}`)
   from the spliced send entry (it's the packet ctor, not a wire read).
4. **Distrust IDB names — `COutPacket(&pkt, OPCODE)` is truth.** PLAYER_INTERACTION is a
   serverbound dispatcher; the leading wire byte is the sub-op mode — confirm the body
   that follows matches the Atlas codec, not the IDB symbol.

`n-a` is justified ONLY if, after *attempting* the splice, the IDB genuinely lacks the
arm — recorded with the IDB-confirmed reason. Given `BuyItem#Merchant` present on all
versions, `n-a` is NOT the expected outcome for any merchant cell.

## Linkage — already wired (no new `candidatesFromFName` case needed)

`tools/packet-audit/cmd/run.go` already maps all four fnames (confirmed):
- line ~1851 → `OperationInvite` (interaction)
- line ~1877 → `OperationMemoryGameTieAnswer` (interaction)
- line ~1888 → `OperationMerchantPutItem` (interaction)
- line ~1892 → `OperationMerchantRemoveItem` (interaction)

The executor *verifies* these still resolve after any splice, but ADDS none.

## CORRECTIONS to design.md (resolved from source — use these paths)

1. **Template filenames carry a `_1` suffix.** Design §1.2 wrote
   `template_gms_83.json` etc. The actual files are
   `services/atlas-configurations/seed-data/templates/template_{gms_83_1,gms_84_1,gms_87_1,gms_95_1,jms_185_1}.json`
   (also `template_gms_12_1.json`, `template_gms_92_1.json` exist but are out of scope).
   Use the `_1`-suffixed names in every report-gen `-template` arg.
2. **The interaction route token varies by version.** `grep -c CharacterInteractionHandle`
   hits the gms_83_1/gms_84_1/jms_185_1 templates but the gms_87_1/gms_95_1 templates spell
   it `"CharacterInteraction"` (without `Handle`). v95 cells are already ✅, so routing
   *exists* on every version — confirm the exact handler entry per version before claiming
   a cell rather than grepping one literal. No `routedElsewhere && !routed` conflict is
   expected (PLAYER_INTERACTION is routed everywhere); confirm, don't assume.

## Version → IDA port (enumerate `list_instances`, don't hardcode — ports vary by launch)

v83=13341, v84=13337, v87=13340, v95=13339, jms=13338. `select_instance(port)` is shared
global state — serialize Phase C by IDB, never two harvests in parallel. Match the loaded
IDB by **binary name**, not by assumed port.

## Verification gate (branch "done")

In `libs/atlas-packet` (the only changed module):
`go test -race ./...`, `go vet ./...`, `go build ./...` clean. `tools/redis-key-guard.sh`
clean from repo root. **No `docker buildx bake`** (no `go.mod` touched). `matrix --check`
+ fname-doc/operations `--check`: no new interaction lines, conflict count not increased.
All four packets ✅ on every applicable version in `status.json`; each promoted cell has a
`packet-audit:verify` byte-fixture + fresh pinned evidence + audit report, committed together.

## Dispatcher-family STATUS.md trap

PLAYER_INTERACTION's STATUS.md op row grades **worst-of-all-siblings**. Per-packet cells
flip in `status.json` as they land, but the STATUS.md row only flips once the worst sibling
is done. A mid-campaign STATUS.md still showing the row red is expected, not a regression.
Because these 12 are the only remaining incomplete siblings, completing all four packets
flips the row too.
