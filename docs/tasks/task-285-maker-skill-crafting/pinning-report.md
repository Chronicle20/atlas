# Task 7 pinning report — `character/serverbound/MakerSkill`

Result: **all eight cells promoted `❌ incomplete` → `✅ verified`**.
Commit `640b8caa1d4dc05751024b02d2dc029a3e2d2352`.

## What was blocking

`CUIItemMaker::RequestItemMake` was absent from the `functions` map of all eight
committed exports under `docs/packets/ida-exports/`, so `evidence pin` failed
"function not in export" and a bare marker made `matrix --check` exit 1 with an
orphan-marker error. Task 7's implementer correctly removed the markers rather
than substituting an fname.

## What was done

1. **Surgical export splice, not a bulk re-export.** Each of the eight
   `RequestItemMake` bodies was decompiled fresh against the IDB resolved by
   `idb_list` (binary filename matched, session id passed as `database`), and
   one hand-authored entry per export was inserted as text. `git diff --numstat`
   showed `62 / 0` on every file — additions only, no existing entry touched.

   | version | export | address | `COutPacket` opcode | registry opcode | session / IDB |
   |---|---|---|---|---|---|
   | gms_v72 | `gms_v72.json` | `0x760cc3` | `0x70` = 112 | 112 | `99e435d8` GMS_v72.1_U_DEVM.exe.i64 |
   | gms_v79 | `gms_v79.json` | `0x795dc3` | 111 | 111 | `5a1cd4f3` GMS_v79_1_DEVM.exe.i64 |
   | gms_v83 | `gms_v83.json` | `0x827096` | `0x71` = 113 | 113 | `754107bf` MapleStory_dump.exe.i64 |
   | gms_v84 | `gms_v84.json` | `0x8524b7` | 113 | 113 | `46c2a2eb` GMS_v84.1_U_DEVM.i64 |
   | gms_v87 | `gms_v87.json` | `0x88afd1` | `0x74` = 116 | 116 | `c0829805` GMSv87_4GB.exe.i64 |
   | gms_v92 | `gms_v92.json` | `0x7afdc0` | `0x7C` = 124 | 124 | `019cd393` GMS_v92_1_DEVM.exe.i64 |
   | gms_v95 | `gms_v95.json` | `0x7d58d0` | 125 | 125 | `ecc757f4` GMS_v95.0_U_DEVM.exe.i64 |
   | jms_v185 | `gms_jms_185.json` | `0x8b1040` | `0x6C` = 108 | 108 | `a977912e` MapleStory_dump_SCY.exe.i64 |

   Every opcode cross-checks against the registry (§10 "distrust IDB names").
   Each entry records eleven guarded `calls` read straight off the decompile —
   the mode-1|2 arm (`Encode4` class, `Encode4` target, `Encode1` catalyst,
   `Encode4` gem count, looped `Encode4` gem id), the mode-3 arm, and the mode-4
   arm — confirming Task 7's finding that the recipe class is encoded exactly
   ONCE, inside the selected arm, and that a class outside 1..4 sends an empty
   body.

2. **Named the two unnamed senders.** The v84 and v92 IDBs carried no
   `CUIItemMaker` symbols; `sub_8524B7` / `sub_7AFDC0` were renamed to
   `?RequestItemMake@CUIItemMaker@@IAEHXZ` and the IDBs saved, so the export key
   is honest for a future harvest (playbook "unnamed sender → name it"). Both
   were confirmed by decompilation before renaming — same guard chain, same
   `COutPacket` opcode as the registry, same per-arm encode order as v83/v95.

3. **Report-less promotion path.** `MAKER_SKILL` is routed in **no** seed
   template — that is plan Task 25's scope — so the ROOT report generator
   produces no `CharacterMakerSkill.json` (verified: a full v95 report run wrote
   1509 files, none of them MakerSkill). Rather than pre-empt Task 25 by wiring
   eight templates, the cells promote through the documented report-less
   byte-fixture path (`internal/matrix/grade.go`, `!hasReport` branch, and the
   `registryDeclaresPacket` carve-out in `cmd/matrix.go` that keeps such records
   out of the dangling-evidence rule): each of the eight registry entries gained

       packet: character/serverbound/MakerSkill

   `character/` is a `packet_prefixes` entry in `docs/packets/evidence/tiers.yaml`,
   so the packet is tier-1 and only a linked fixture + fresh evidence promotes it.
   Because the op is routed nowhere, `routedElsewhere` is false on every version
   and no template-wiring conflict is raised.

4. **Evidence pinned** for all eight (`TIER1-FIXTURE`), each with a `verifies:`
   pointing at its per-version byte test. Pinned addresses match the markers
   exactly.

5. **Markers restored.** The eight `EVIDENCE (pin pending …)` blocks in
   `libs/atlas-packet/character/serverbound/maker_skill_test.go` became real
   `// packet-audit:verify packet=character/serverbound/MakerSkill version=… ida=…`
   lines. The IDA-evidence comments above them are unchanged, and
   `maker_skill.go` was not touched.

## Gates (verbatim exit codes)

    matrix --check EXIT=0
    fname-doc --check EXIT=0
    fname-doc check OK (272 structs without an audit report carry no fname)
    operations --check EXIT=0
    operations check OK (0 absent-writer note(s))
    dispatcher-lint EXIT=0
    dispatcher-lint: clean

`go test ./character/serverbound/...` in `libs/atlas-packet` and
`go test ./...` in `tools/packet-audit` both pass. STATUS.md and status.json
were regenerated with the tool (never hand-edited) and are committed in the same
commit as the test and evidence; `git show --stat 640b8caa1` confirms both files
are in it.

## The row

    | MAKER_SKILL | CUIItemMaker::RequestItemMake |  |  | ⬜ |  | ⬜ | 0x070 | ✅ | 0x06F | ✅ | 0x071 | ✅ | 0x071 | ✅ | 0x074 | ✅ | 0x07C | ✅ | 0x07D | ✅ | 0x06C | ✅ |

`gms_v48` / `gms_v61` remain `⬜ n-a` as instructed. The only other STATUS.md
movement is the eight export hashes and the per-version summary counts, each
`+1 verified / -1 incomplete`; the conflict count stays at 0 on every version.
