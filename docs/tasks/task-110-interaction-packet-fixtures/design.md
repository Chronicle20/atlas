# Interaction Packet-Fixture Verification Campaign — Design

Task: task-110-interaction-packet-fixtures
Phase: 2 (Design)
Created: 2026-06-24
Status: Draft for review

---

## 1. Problem & Framing

The `interaction` family (CMiniRoomBaseDlg / PLAYER_INTERACTION — trade, personal store,
entrusted-merchant, memory mini-game) is ~90% verified in the coverage matrix
(`docs/packets/audits/STATUS.md`). The clientbound dispatcher arms (InteractionEnter /
UpdateMerchant) were graduated to ✅ in the task-096 dispatcher campaign. What remains is
**exactly 12 `incomplete` serverbound cells** (confirmed by `jq` over `status.json`, matching
the PRD count), concentrated in **four** serverbound packets:

| Packet (`interaction/serverbound/…`) | fname | Incomplete versions | Verified |
|---|---|---|---|
| `InteractionOperationInvite` | `CField::SendInviteTradingRoomMsg` | v83, v87, jms | v84, v95 |
| `InteractionOperationMemoryGameTieAnswer` | `CMemoryGameDlg::OnTieRequest` | v84 | v83, v87, v95, jms |
| `InteractionOperationMerchantPutItem` | `CPersonalShopDlg::PutItem#Merchant` | v83, v84, v87, jms | v95 |
| `InteractionOperationMerchantRemoveItem` | `CPersonalShopDlg::MoveItemToInventory#Merchant` | v83, v84, v87, jms | v95 |

These 12 are the *only* incomplete interaction cells; clearing all four packets clears the
family's serverbound coverage.

The PRD framed this as "mostly a port-from-a-verified-sibling shape." Investigation
**partly confirms and partly corrects** that framing. The decisive fact, recorded before any
cell is touched, is an **export-presence split** that sorts the 12 cells into three work-classes
of very different effort (§4). All assumptions below are resolved from source, not memory.

### 1.1 The export-presence finding (reshapes the work)

`grep` of `docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v95,gms_jms_185}.json` for the
in-scope client functions (`Y` = present, `0` = absent):

| fname (packet) | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|
| `CField::SendInviteTradingRoomMsg` (Invite) | Y | Y | Y | Y | Y |
| `CMemoryGameDlg::OnTieRequest` (TieAnswer) | Y | **0** | Y | Y | Y |
| `CPersonalShopDlg::PutItem` (PersonalStore put — base, already verified all versions) | Y | Y | Y | Y | Y |
| `CPersonalShopDlg::PutItem#Merchant` (MerchantPutItem) | **0** | **0** | **0** | Y | Y? **0** |
| `CPersonalShopDlg::MoveItemToInventory` (PersonalStore remove — base) | Y | Y | Y | Y | Y |
| `CPersonalShopDlg::MoveItemToInventory#Merchant` (MerchantRemoveItem) | **0** | **0** | **0** | Y | **0** |
| `CPersonalShopDlg::BuyItem#Merchant` (MerchantBuy — already ✅ all versions) | Y | Y | Y | Y | Y |

(`PutItem#Merchant` / `MoveItemToInventory#Merchant` are present **only** on v95; absent on
v83/v84/v87/jms.)

Three independent facts fall out of this table and ground every decision in §4:

1. **The `#Merchant` arms are producible, not `n-a`.** `CPersonalShopDlg::BuyItem#Merchant` is
   present on **all five** versions and `MerchantBuy` is already ✅ everywhere. That proves (a)
   the entrusted-merchant feature exists pre-v95 in every binary, and (b) the exporter's
   `#case`-arm splitting *does* capture entrusted-merchant send sites. So the missing
   `PutItem#Merchant` / `MoveItemToInventory#Merchant` arms on v83/v84/v87/jms are a
   **harvest-depth gap in the export**, not a missing feature — they are produced by re-harvest +
   surgical splice (§4 Class E-arm, §5), never recorded as `n-a`.
2. **The base functions are named on every version.** `CPersonalShopDlg::PutItem` and
   `::MoveItemToInventory` (the personal-store arms) are present everywhere — so this is *arm
   extraction from an already-named function*, not byte-signature naming of an unnamed sender.
   That is a materially cheaper Class-E than the login/character campaigns' "name an unnamed
   sub_XXXX" work.
3. **TieAnswer v84 is a single true-Class-E hole.** `OnTieRequest` is present on v83 (the
   byte-identical v84 sibling) and ✅ there, but absent from the v84 export. The v84 IDB holds it
   present-but-unnamed; name it against the v83 twin + splice (§5).

### 1.2 Corrections to PRD assumptions (resolved from source)

- **Codec ownership (PRD §7 said "atlas-channel").** All four wire codecs and their fixture tests
  live in **`libs/atlas-packet/interaction/serverbound/`** — a single Go module, `libs/atlas-packet`.
  `services/atlas-channel` only *consumes* them via the `CharacterInteractionHandle` handler. **The
  only changed Go module is `libs/atlas-packet`** (test files only, unless a fixture proves a wire
  delta — none expected, §6). No `go.mod` is touched, so **no `docker buildx bake` is required** by
  the CLAUDE.md build gate (it triggers only on `go.mod` changes); the gate reduces to
  `go test -race`/`go vet`/`go build` on `libs/atlas-packet` plus the `packet-audit … --check` trio.
- **All four ops are already wired** in `tools/packet-audit/cmd/run.go` `candidatesFromFName`
  (Invite → `OperationInvite` line ~1850; TieAnswer → `OperationMemoryGameTieAnswer` line ~1876;
  `PutItem#Merchant` → `OperationMerchantPutItem` line ~1887; `MoveItemToInventory#Merchant` →
  `OperationMerchantRemoveItem` line ~1891). **No new linkage case is needed** — the executor must
  verify these still resolve after any export splice, but adds none.
- **PLAYER_INTERACTION is routed serverbound in every template.** `CharacterInteractionHandle` is
  present at the per-version opcode (0x7B gms_83, 0x7D gms_84, 0x7C jms_185, and the v87/v95
  equivalents) in `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_95,jms_185_1}.json`
  (jms file is **`template_jms_185_1.json`** — never assume the bare name). So **no
  `routedElsewhere && !routed` template-wiring conflict is expected** — PRD Open Question 3 resolved:
  these are missing-fixture/missing-export holes, not routing gaps. The executor confirms the route
  per version before claiming a cell rather than trusting this table blind.

### 1.3 PRD Open Questions — resolved

- **Q1 (merchant: port vs per-version shift):** The Atlas codecs are **uniform** — `OperationMerchantPutItem`
  encodes `byte inventoryType · int16 slot · uint16 quantity · uint16 set · uint32 price` with **no
  `MajorVersion()` gate**, byte-for-byte identical to the **all-versions-verified**
  `OperationPersonalStorePutItem`; `OperationMerchantRemoveItem` is a single `uint16 index`, identical
  to the verified `OperationPersonalStoreRemoveItem`. The wire shape is therefore a port; the executor
  *confirms* it per version by decompiling the merchant arm (§5), not by assumption.
- **Q2 (Invite: shared codec):** `OperationInvite` is a single `WriteInt(targetCharacterId)`, uniform,
  fname present on all versions → pure report-gen (Class A).
- **Q3 (routedElsewhere conflict):** none expected (§1.2).

---

## 2. The 12 cells, enumerated by (packet, op, fname, versions, class)

Column order matches the matrix: **v83 · v84 · v87 · v95 · jms_v185**. Every incomplete cell's
verbatim `status.json` note is **"no audit report."** The verdict symbol in any future report is the
report's verdict — *not* the cell state; the executor re-adjudicates each cell, never trusts a verdict
blind.

| # | Packet | Incomplete versions | fname | Class | Existing test marker coverage |
|---|---|---|---|---|---|
| 1–3 | `InteractionOperationInvite` | v83, v87, jms | `CField::SendInviteTradingRoomMsg` | **A** (report-gen only) | `operation_invite_test.go` has markers for v95, v84 |
| 4 | `InteractionOperationMemoryGameTieAnswer` | v84 | `CMemoryGameDlg::OnTieRequest` | **E** (name+splice v84) | `operation_memory_game_tie_answer_test.go` has v83/v87/v95/jms |
| 5–8 | `InteractionOperationMerchantPutItem` | v83, v84, v87, jms | `CPersonalShopDlg::PutItem#Merchant` | **E-arm** (splice merchant arm + report-gen) | `operation_merchant_put_item_test.go` has v95 only |
| 9–12 | `InteractionOperationMerchantRemoveItem` | v83, v84, v87, jms | `CPersonalShopDlg::MoveItemToInventory#Merchant` | **E-arm** | `operation_merchant_remove_item_test.go` has v95 only |

All four structs are **standalone codecs with uniform (un-gated) Encode/Decode** — the per-version
work is adding the missing `packet-audit:verify` marker rows to the existing `pt.Variants` table tests
and producing the export/report/evidence artifacts, never editing the wire codec (absent a proven
delta).

---

## 3. Definition of "verified" for an interaction serverbound cell

Per `VERIFYING_A_PACKET.md` §9, a serverbound cell needs **three coupled artifacts that all agree**,
**and** the op must be **routed** in that version's template (§1.2 confirms routing for all five):

1. The `// packet-audit:verify packet=interaction/serverbound/<Struct> version=<key> ida=<0xaddr>`
   marker above a full-body byte-fixture (§6), using the `pt.Variants`-indexed table pattern (reference:
   `libs/atlas-packet/party/clientbound/invite_test.go` → `TestInviteByteOutput`). Every byte cites a
   decompile line in a comment.
2. A **pinned evidence record** `docs/packets/evidence/<version>/interaction.serverbound.<Struct>.yaml`
   with a `verifies:` line and an `ida.function` resolved from the export (carrying the `#Merchant`
   suffix where applicable — `evidence pin --ida "CPersonalShopDlg::PutItem#Merchant"`).
3. An **audit report** `docs/packets/audits/<version>/Interaction<Struct>.json` (and `.md`), generated
   deterministically by the ROOT command against the export (`-ida-source`). Evidence with no report →
   `matrix --check` "dangling evidence" failure.

The byte-fixture is the load-bearing artifact — a full-body byte test, **not** a mode-byte or
length enumeration (per the "dispatcher mode-byte is a false pass" rule). For the merchant ops the
body is small but must be exercised end-to-end (`inventoryType…price`).

---

## 4. Work-classes (drives plan ordering)

The class is a hypothesis from §1.1's export table; the executor confirms it per cell (export grep →
report-gen → decompile only when needed).

- **Class A — report-gen only (no live IDA).** Client function present in the committed export; the
  cell is `incomplete` only because the per-version report was never copied in. Action: run ROOT
  report-gen to a temp `-output` → copy `InteractionOperationInvite.{json,md}` into
  `docs/packets/audits/<v>/` → add the missing marker row + pin evidence → regen matrix.
  **Cells: Invite v83, v87, jms (3).** Cheapest slice; sequenced first to bank early promotions and
  shake out the report-gen command + `--check` baseline before the harder slices.

- **Class E (true) — function absent from one version's export; name-and-splice.** `OnTieRequest` is
  absent from the v84 export but ✅ on v83 (byte-identical sibling). Open the v84 IDB
  (port per §8), confirm `OnTieRequest` is present-but-unnamed (byte-signature anchored, twin-matched
  to v83's verified function), **name it**, and **surgically splice** the single absent entry into
  `gms_v84.json` (absent-only; never overwrite). Then it becomes Class A. **Cell: TieAnswer v84 (1).**

- **Class E-arm — named base function present, `#Merchant` case-arm not in export; re-harvest the arm +
  surgical splice.** `CPersonalShopDlg::PutItem` / `::MoveItemToInventory` are present on every version
  (personal-store arm), but the **entrusted-merchant arm** (`#Merchant` case key) was harvested only on
  v95. For each of v83/v84/v87/jms: select that IDB, decompile the base function, locate the
  entrusted-merchant send arm (the same arm that produced the all-versions-present `BuyItem#Merchant`
  proves the pattern), confirm its write order matches the Atlas codec (§1.3 Q1), produce the
  `#Merchant` case-keyed export entry via **surgical splice** (§5), then Class A (report-gen + marker +
  evidence). **Cells: MerchantPutItem v83/v84/v87/jms (4) + MerchantRemoveItem v83/v84/v87/jms (4) = 8.**
  This is the highest-effort slice (8 of 12 cells, all requiring a live IDB) and is sequenced last.

---

## 5. Export-splice discipline (Class E and E-arm)

The export is **non-idempotent** — re-running `packet-audit export` over a committed file drifts ~150
unrelated function keys and degrades other cells (`VERIFYING_A_PACKET.md` §10). Both Class-E paths
therefore use **surgical splice**, never a wholesale re-export:

1. Harvest to a **temp file** with `-prior-export "" -pending <roster.md> -descent-depth 12`, pointing
   `-ida-url`/`-ida-port` at the **target** version's IDA instance (one instance per harvest;
   `select_instance` is shared global state — never two in parallel).
2. Splice **only** the needed entry into the committed export:
   - **TieAnswer v84:** add `CMemoryGameDlg::OnTieRequest` (absent-only).
   - **Merchant arms:** add `CPersonalShopDlg::PutItem#Merchant` / `::MoveItemToInventory#Merchant`
     for the target version (absent-only). The base `PutItem` / `MoveItemToInventory` entries already
     exist and are **left untouched**.
3. Strip any `COutPacket`-delegate harvest artifact from the spliced send entry (§10 of the playbook).
4. **Distrust IDB names — the `COutPacket(&pkt, OPCODE)` integer is truth.** Cross-check the merchant
   arm's opcode against the registry; PLAYER_INTERACTION is a serverbound dispatcher, so the leading
   wire byte is the sub-op mode — confirm the body that follows matches the Atlas codec, not the IDB
   symbol.

If, after *attempting* the splice, a version's IDB genuinely lacks the merchant arm (no entrusted-merchant
send distinct from personal-store) or `OnTieRequest`, that is the only path to a **justified `n-a`** —
recorded with the IDB-confirmed reason, never inferred from the export's absence. Given
`BuyItem#Merchant` is present on all versions, `n-a` is not the expected outcome for any merchant cell.

---

## 6. Wire-delta contingency (fix-first, expected empty)

The Atlas codecs are uniform and the merchant bodies are byte-identical to the all-versions-verified
personal-store bodies, so the expected outcome of every per-version decompile is **"client write order
matches the Atlas codec → cell promotes."** If a decompile reveals a genuine wire delta (e.g. a
version writes the merchant slot encoding differently), the playbook §4 fix-first rule applies: **the
wire fix is its own commit + its own review FIRST**, then the fixture. This is flagged as a contingency,
not an expected step. Watch specifically for the v84 off-by-one class (`>83` should be `>=87`) — though
these codecs have **no** `MajorVersion()` gate, so that trap does not apply here; v84 is expected to be
byte-identical to v83.

---

## 7. Sequencing

Three phases, ordered cheapest-and-most-decoupling first:

- **Phase A — Invite (Class A, 3 cells).** No live IDA. Establishes the report-gen command invocation,
  the temp-`-output` copy step, the marker-row + evidence-pin pattern, and a clean `matrix --check`
  baseline for the family. Banks 3 promotions fast.
- **Phase B — TieAnswer v84 (Class E, 1 cell).** Single live-IDB name-and-splice against the v83 twin;
  smallest E to prove the splice workflow before fanning it across 8 cells.
- **Phase C — Merchant put/remove (Class E-arm, 8 cells).** Per-version (v83/v84/v87/jms) merchant-arm
  decompile + splice + report-gen, serialized by IDB (one `select_instance` at a time). Two packets ×
  four versions; the v95 verified report is the structural template for each.

Within each phase, land each cell as its three coupled artifacts in one commit (marker + evidence +
regenerated `STATUS.md`/`status.json`), so a bisect of the branch shows a monotonic climb in the matrix.

### 7.1 Dispatcher-family-trap note

PLAYER_INTERACTION fans out to many per-op writers, so its **STATUS.md op row grades worst-of-all
siblings** (`VERIFYING_A_PACKET.md` §2). Each per-packet cell promotes in `status.json` as it lands, but
the STATUS.md PLAYER_INTERACTION row only flips once the worst sibling is done. Because these 12 cells
are the *only* remaining incomplete interaction siblings (§1), completing all four packets flips the row
too — but a mid-campaign STATUS.md that still shows the row red is expected and not a regression.

---

## 8. Tooling & instances

- IDA via the documented MCP API; `select_instance(port)` to the IDB whose loaded version matches the
  cell, confirmed before any read. Ports per the PRD: **v83=13341, v84=13337, v87=13340, v95=13339,
  jms=13338** — but enumerate `list_instances` and match the loaded IDB rather than hardcoding, since
  ports vary by launch order.
- Report-gen is the ROOT command with `-csv-clientbound`/`-csv-serverbound`/`-template <per-version>`/
  `-ida-source <export>` to a temp `-output`; copy the specific `Interaction<Struct>.{json,md}` into
  `docs/packets/audits/<version>/`.
- `evidence pin --packet <id> --version <key> --ida "<FName-with-#suffix>" --category TIER1-FIXTURE`,
  then hand-add the `verifies:` line.

---

## 9. Verification gate (branch "done")

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in **`libs/atlas-packet`** (the only
  changed module).
- `tools/redis-key-guard.sh` clean from repo root (no redis surface here, but the gate runs).
- **No `docker buildx bake`** — no `go.mod` is touched (§1.2); the bake gate is conditioned on `go.mod`
  changes.
- `go run ./tools/packet-audit matrix --check` plus the fname-doc / operations `--check` runs: **no new**
  orphan/dangling/stale/drift lines mentioning any interaction packet, and the pre-existing conflict
  count must not increase (the §8 `--check` caveat: it may still exit 1 from the unrelated registry-seed
  conflict backlog — the bar is "introduce no new problems," not a clean exit until that backlog is zero).
- All four packets show ✅ on every applicable version in `status.json`; each promoted cell carries a
  `packet-audit:verify` byte-fixture + fresh pinned evidence + an audit report, committed together.

---

## 10. Risks & mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Merchant `#Merchant` arm not separable in an older IDB (shares the exact personal-store send) | Low — `BuyItem#Merchant` present on all versions proves the split exists | Decompile the base function; if the merchant arm is a genuine distinct send, splice it; only if IDA-confirmed identical-with-no-distinct-send → justified `n-a` with reason (§5) |
| Accidental full re-export drifts unrelated cells | Medium (operator error) | Surgical absent-only splice only; harvest to temp; never overwrite a committed export (§5) |
| Wire delta on some version (codec actually differs) | Low — uniform Atlas codec, personal-store twin ✅ all versions | Fix-first: wire fix its own commit + review before the fixture (§6) |
| `matrix --check` exits 1 from pre-existing backlog masks a real new problem | Medium | Diff the `--check` output for lines mentioning interaction packets specifically; conflict count must not rise (§9) |
| Two IDA harvests racing on shared `select_instance` state | Medium | Serialize Phase C by IDB, one `select_instance` at a time (§7) |

---

## 11. Out of scope (per PRD non-goals)

- No new trade/personal-store/merchant features — verification only.
- No changes to the already-verified clientbound dispatcher arms (task-096).
- No opcode reshifts unless a fixture proves the registry opcode wrong — then **surface** the conflict,
  don't silently patch.
