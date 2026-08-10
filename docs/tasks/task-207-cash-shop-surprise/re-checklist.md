# Reverse-engineering checklist — task-207

FR-5 requires every one of the ten matrix columns to end as
implemented-and-verified or `n-a`-with-proof. This file enumerates the IDB work
that must happen during `/design-task` before any codec is written. Nothing here
may be answered from general MapleStory knowledge.

Resolve each IDB session from `idb_list` by binary **name** and pass it as the
`database` parameter (port-based `select_instance` is dead since task-138). Use
`func_query` with `name_regex` for lookups.

## Per-version questions

For each version, answer all four:

1. Does `CUICashItemGachapon` (or its unnamed equivalent) exist in this build?
2. What opcode does its button-click handler send, and what is the body read
   order?
3. Which clientbound path carries the result — the standalone
   `CCashShop::OnCashItemGachaponResult` opcode, or a `CCashShop::OnCashItemResult`
   dispatcher arm?
4. What is the result body's exact read order?

| Version | Registry says | What must be determined |
|---|---|---|
| gms_v48 | result `0x101`; no trigger | Whether the UI class exists at all; if not, `n-a` proof both directions |
| gms_v61 | result `0x100`; no trigger | Same |
| gms_v72 | result `0x124`; no trigger. Registry note: `v72-EXTRA cash op (no v79 equiv)` (`gms_v72.yaml:1877`) | Same |
| gms_v79 | **neither** in registry | Confirm absence in the binary → `n-a` proof both directions |
| gms_v83 | result `0x14D`; trigger `0x0A1` **`provenance: csv-import`** (`gms_v83.yaml:2856-2860`) | Re-derive BOTH opcodes from the IDB and upgrade the registry provenance. This is the primary target version — do it first |
| gms_v84 | result `0x154`; trigger `0x0A5`; `GACHAPON_OPEN_*` modes 165/166 | Which path the client actually takes when the standalone opcode AND the dispatcher arm both exist. `gms_v84.yaml:2261` notes cases 340/341 both route to the same handler |
| gms_v87 | result `0x15E`; trigger `0x0A9`; modes 171/172 | Same |
| gms_v92 | result `0x180`; trigger `0x0B6`; **no `GACHAPON_OPEN_*` modes in the yaml** | Whether the arms exist in the v92 dispatcher. Absence from the yaml is not proof. Note `CASHSHOP_OPERATION` itself is ❌ on v92 (`STATUS.md:406`, `0x178`) |
| gms_v95 | result `0x188`; trigger `0x0B9`; modes 183/184 | Same as v84/v87. v95 is PDB-backed, so it is the best column for deriving the shared read order |
| jms_v185 | result `0x16D`; no trigger; `GACHAPON_OPEN_*` recorded `n-a` by task-183 | Whether the UI class exists; if not, `n-a` proof both directions |

## Cross-cutting items

- **`GachaponOpenDone` field semantics (PRD Q4).** Read
  `CUICashGachapon::OnCashGachaponOpenResult` to learn what `resultCode`
  (int32) and `resultParam2` (byte) mean, and what value of `isCashItem` makes
  the client append the 55-byte `GW_CashItemInfo` blob.
  Struct: `libs/atlas-packet/cash/clientbound/shop_operation_result_gachapon.go:33`.
- **Error codes (PRD Q5, FR-6.3).** Locate the cash-shop error-string table and
  identify the codes for locker-full, item-not-found, and a generic failure.
  `BuyFailed` carries a single `errorCode` byte
  (`shop_operation_result_failed.go:128`).
- **`GACHAPON_OPEN_FAILED` (FR-6.2).** Decide from the IDB whether that arm is
  worth implementing on v84/v87/v95. It has dispatcher modes but no codec.
- **WZ presence (PRD Q7).** Independently of the IDBs, confirm item `5222000`
  exists in the ingested WZ data for each target version, and inspect its
  `Item/Cash` spec node. A version whose data lacks the item is `n-a`
  regardless of packet support.

## Output

Each `n-a` conclusion must be recorded as a proof that satisfies the `n-a`
consistency gate — absence verified against the binary, not inferred from a
missing registry entry.
