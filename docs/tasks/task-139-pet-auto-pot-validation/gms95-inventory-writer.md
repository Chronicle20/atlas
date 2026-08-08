# gms_95 `CharacterInventoryChange` writer — resolved (PicResult removed, not relocated)

## Summary

`template_gms_95_1.json` had no writer entry for `CharacterInventoryChange`
(`InventoryChangeWriter`, `libs/atlas-packet/inventory/clientbound/change.go:16`), leaving every
gms_95 inventory-operation announcement (including task-139's `FLAG_CHANGED` pet re-announce) an
unroutable, silently-dropped writer. The apparent "gap" between `PicResult` (0x1C) and
`StatChanged` (0x1E) was a false lead: `PicResult`'s presence in the gms_95 template at all was
the defect, and `0x1C` genuinely belongs to `CharacterInventoryChange`.

**Amendment:** an earlier pass in this review chain relocated `PicResult` to `0x1B` instead of
removing it, reasoning from a positional coincidence with `CLogin::OnCheckSPWResult` (case 27) in
the older v83/v87/v92 templates. Code review caught that this was an unverified inference — no
source ties the Atlas op name `PicResult` to the fname `CLogin::OnCheckSPWResult`, and
`docs/packets/ida-exports/_pending.md` §7 (line 183) already, frozen, classifies `PicResult` as
**version-absent for v95** in the same row as `ServerLoad`/`ServerSelect` (both of which are
correctly absent from `template_gms_95_1.json` already — confirmed by grep, neither string
appears in the file). That relocation has been reverted; `PicResult` is now removed from
`template_gms_95_1.json` entirely, matching the frozen registry. `CharacterInventoryChange`
remains wired at `0x1C` exactly as originally derived — that half of the work was independently
re-verified by the reviewer and stands unchanged.

- `PicResult` — **removed** from `template_gms_95_1.json` (version-absent for v95, per
  `_pending.md` §7 line 183)
- `CharacterInventoryChange` — added at `0x1C` with `services: ["channel"]` and
  `options.petSkill.autoSpeaking: "0x100"` (unchanged from the original fix)

## Step A — is the gms_95 writer table broadly trustworthy?

Sampled 15 more writer entries spread from `0x29` to `0xE5` (well beyond the initial two), cross-
checked against the real v95 receive-dispatch chain (session `e4abcb98`,
`GMS_v95.0_U_DEVM.exe.i64`):

| Template writer | opCode (dec) | Real dispatcher / case | Result |
|---|---|---|---|
| `MapTransferResult` | 0x29 (41) | `CWvsContext::OnPacket` case 41 → `OnMapTransferResult` @ `0x9f9f90` | MATCH (exact name) |
| `SetTamingMobInfo` | 0x2F (47) | case 47 → `OnSetTamingMobInfo` @ `0x9f7280` | MATCH (exact name) |
| `GuildBBS` | 0x3B (59) | case 59 → `OnGuildBBSPacket` @ `0x9ccf20` | MATCH |
| `CharacterInfo` | 0x3D (61) | case 61 → `OnCharacterInfo` @ `0xa05750` | MATCH (exact name) |
| `ShopScannerResult` | 0x49 (73) | case 73 → `OnShopScannerResult` @ `0xa076c0` | MATCH (exact name) |
| `MonsterBookSetCard` | 0x56 (86) | case 86 → `OnMonsterBookSetCard` @ `0x9ddcb0` | MATCH (exact name) |
| `MonsterBookSetCover` | 0x57 (87) | case 87 → `OnMonsterBookSetCover` @ `0x9cfa70` | MATCH (exact name) |
| `AvatarMegaphoneResult` | 0x72 (114) | case 114 → `OnAvatarMegaphoneRes` @ `0xa016c0` | MATCH |
| `SetField` | 0x8D (141) | `CStage::OnPacket` @ `0x71b0b0` case 141 → `OnSetField` @ `0x71a0a0` | MATCH (exact name) |
| `FieldEffect` | 0x9A (154) | `CField::OnPacket` @ `0x546d50` case 154 → `OnFieldEffect` @ `0x53b790` | MATCH (exact name) |
| `SetQuestClear` | 0xA6 (166) | case 166 → `OnSetQuestClear` @ `0x52c870` | MATCH (exact name) |
| `FootholdInfo` | 0xB0 (176) | case 176 → `OnFootHoldInfo` @ `0x53a810` | MATCH (exact name) |
| `CharacterSpawn` | 0xB3 (179) | `CUserPool::OnPacket` @ `0x94ddf0`, `nType==0xB3` → `OnUserEnterField` @ `0x94db40` | MATCH (semantic: enter field = spawn) |
| `CharacterMovement` | 0xD2 (210) | `CUserPool::OnUserRemotePacket` @ `0x94b390` case 210 → `OnMove` @ `0x948a80` | MATCH (exact name) |
| `GuildNameChanged` | 0xE4 (228) | case 228 (same function) → `OnGuildNameChanged` @ `0x9550b0` | MATCH (exact name) |

Bonus matches surfaced along the way while tracing `OnUserRemotePacket` (not double-counted in
the 15): `CharacterAttackMelee/Ranged/Magic/Energy` (0xD3–0xD6) → `OnAttack` cases 211–214;
`CharacterSkillPrepareForeign` (0xD7) → `OnSkillPrepare` case 215; `CharacterSkillCancelForeign`
(0xD9) → `OnSkillCancel` case 217; `CharacterDamage` (0xDA) → `OnHit` case 218;
`CharacterExpression` (0xDB) → `OnEmotion` case 219; `GuildEmblemChanged` (0xE5) →
`OnGuildMarkChanged` case 229.

**Tally: 15/15 sampled entries MATCH** (plus 8 more incidental matches), spanning `0x29` through
`0xE5` — essentially the entire low-to-mid opcode range. Combined with the original
`StatChanged` (0x1E, MATCH) and `PicResult` (0x1C, MISMATCH) pair, that's **16 matches out of 17
checked cells**. The one mismatch is isolated to a single entry, not a run — the table is
broadly trustworthy; the `PicResult` conflict was a standalone defect, not a version-wide shift.
Code review independently re-decompiled `CWvsContext::OnPacket` and re-checked this sample against
`CField::OnPacket`, `CStage::OnPacket`, `CUserPool::OnPacket`, and `CUserPool::OnUserRemotePacket`,
confirming every entry — this section stands as originally reported.

## Step B — what is `PicResult`, and does v95 have it?

`libs/atlas-packet/login/clientbound/pic_result.go:12` — `PicResultWriter = "PicResult"`, a
1-byte codec (`WriteByte(0)` / `ReadByte()`). Grepped every reference to it:
`services/atlas-login/atlas.com/login/main.go:310` (declared in the `availableWriters` list) and
the codec file itself — **no handler in `atlas-login` ever calls
`session.Announce(...)(loginCB.PicResultWriter)(...)`**. Read
`services/atlas-login/atlas.com/login/socket/handler/character_selected_pic.go`
(`CharacterSelectedPicHandleFunc`, the PIC-check handler): on failure it announces `ServerIPWriter`
with an error code, on success it just calls `UpdateState` — it never emits `PicResult` either.
So in the current atlas-login implementation, `PicResultWriter` is declared-but-dead code; there is
also a `// TODO` in `character_selected_register_pic.go` for a future PIC-registration emit path,
which is exactly why getting the wire fact right (rather than leaving an unverified inference in
the template) matters — a future wiring of that TODO must not silently land on the wrong opcode.

**Corrected finding (superseding the original relocation to `0x1B`):**

`docs/packets/ida-exports/_pending.md` §7, line 183 — a frozen, live-IDB-validated
VERSION-ABSENT registry — reads:

> `ServerLoad`, `ServerSelect`, `PicResult` (GMS v12-era / state-machine-routed) | v95 | Pre-v95 or
> non-`CLogin`-dispatched; trivial shapes manually cross-checked.

This classifies `PicResult` as **absent from v95** in the same row, and by the same reasoning, as
`ServerLoad` and `ServerSelect` — both of which are already correctly absent from
`template_gms_95_1.json` (confirmed: neither string appears in the file). The pattern is
meaningful, not coincidental: all three are pre-v95/state-machine-routed features that the
frozen registry says v95 does not have.

Code review additionally decompiled `CLogin::OnPacket` @ `0x5df940` in full: case 27 (`0x1B`)
dispatches to `CLogin::OnCheckSPWResult`, a real, distinct client function — but nothing in
`status.json`, the ops CSVs, or any support doc ties that fname to Atlas's `PicResult` writer. My
earlier "0x1B" conclusion rested only on the two op names sharing a slot number in the older
v83/v87/v92 templates — a positional coincidence, not an established wire fact. Asserting `0x1B`
would have written an unverified inference into a live tenant config; if the `character_selected_
register_pic.go` TODO is ever wired up, that guess would have silently routed the emit into
`OnCheckSPWResult`'s handler instead of failing loudly.

**Outcome: `PicResult` does not exist in v95** (frozen-registry classification, corroborated by
the absence of any `OnPicResult`-shaped function anywhere in the v95 IDB found during the original
IDA trace). Per the three offered outcomes, this is outcome (b): the entry is dead for v95 —
removed from `template_gms_95_1.json` rather than relocated to an unverified opcode.

## Step C — wired

`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`:

```
      {
        "opCode": "0x19",
        "writer": "ServerListRecommendations",
        "services": ["login"]
      },
      {
        "opCode": "0x1C",
        "writer": "CharacterInventoryChange",
        "services": ["channel"],
        "options": {
          "petSkill": {
            "autoSpeaking": "0x100"
          }
        }
      },
      {
        "opCode": "0x1E",
        "writer": "StatChanged",
        "services": ["channel"]
      },
```

Sorted order preserved: `0x19 → 0x1C → 0x1E` (no `0x1B` entry — `PicResult` is gone, not moved).
`CharacterInventoryChange` matches the key set used by `template_gms_87_1.json`'s
`CharacterInventoryChange` entry (`opCode`/`writer`/`services`/`options.petSkill.autoSpeaking`).
**Correction to the original report:** `template_gms_92_1.json`'s `CharacterInventoryChange` entry
has no `options` block at all — only gms_87 matches the full key set; gms_92 does not.

## Verification (foreground, worktree root)

```
$ python3 -c "import json; json.load(open('services/atlas-configurations/seed-data/templates/template_gms_95_1.json'))"
JSON_OK

$ tools/template-opcode-order-guard.sh
OK: 22 template arrays are in ascending opcode order.

$ go run ./tools/packet-audit operations --check
operations check OK (0 absent-writer note(s))

$ (cd services/atlas-configurations/atlas.com/configurations && go test -race ./... && go build ./...)
ok      atlas-configurations/outbox ...
ok      atlas-configurations/seeder ...
ok      atlas-configurations/services ...
ok      atlas-configurations/services/service ...
ok      atlas-configurations/services/task ...
ok      atlas-configurations/templates ...
ok      atlas-configurations/tenants ...
ok      atlas-configurations/tenants/characters ...
ok      atlas-configurations/tenants/characters/preset ...
(build: clean, no output)
```

**Absent-writer note count: unchanged (0 → 0)**, same as after the original fix.
`packet-audit operations --check`'s "absent-writer" tracking is scoped to dispatcher-family
`operations` mode tables, not top-level template writer completeness, so removing `PicResult` (a
plain writer, not part of an `operations` mode table) doesn't move that counter either.

**Matrix regeneration: not required, and not touched.** This change only edits
`template_gms_95_1.json`. `docs/packets/feature-na-evidence.yaml`, `docs/packets/audits/STATUS.md`,
and `status.json` were not touched, per instructions (another agent owns those concurrently).

## Files changed

- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — `PicResult` entry
  removed entirely (version-absent for v95); `CharacterInventoryChange` added at `0x1C`.
