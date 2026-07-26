# gms_95 `CharacterInventoryChange` writer — resolved (PicResult moved 0x1C → 0x1B)

## Summary

`template_gms_95_1.json` had no writer entry for `CharacterInventoryChange`
(`InventoryChangeWriter`, `libs/atlas-packet/inventory/clientbound/change.go:16`), leaving every
gms_95 inventory-operation announcement (including task-139's `FLAG_CHANGED` pet re-announce) an
unroutable, silently-dropped writer. The apparent "gap" between `PicResult` (0x1C) and
`StatChanged` (0x1E) was a false lead: `PicResult`'s opcode was itself wrong (off by one), and
`0x1C` genuinely belongs to `CharacterInventoryChange`. An earlier pass in this session stopped
at that mismatch and reported it rather than guessing; this pass resolves it. Both are now fixed:

- `PicResult` moved `0x1C` → `0x1B`
- `CharacterInventoryChange` added at `0x1C` with `services: ["channel"]` and
  `options.petSkill.autoSpeaking: "0x100"`

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
broadly trustworthy; `PicResult` was a standalone defect, not a version-wide shift. Proceeding to
a targeted fix was safe.

## Step B — what is `PicResult`, and does v95 have it?

`libs/atlas-packet/login/clientbound/pic_result.go:12` — `PicResultWriter = "PicResult"`, a
1-byte codec (`WriteByte(0)` / `ReadByte()`). Grepped every reference to it:
`services/atlas-login/atlas.com/login/main.go:310` (declared in the `availableWriters` list) and
the codec file itself — **no handler in `atlas-login` ever calls
`session.Announce(...)(loginCB.PicResultWriter)(...)`**. Read
`services/atlas-login/atlas.com/login/socket/handler/character_selected_pic.go`
(`CharacterSelectedPicHandleFunc`, the PIC-check handler): on failure it announces `ServerIPWriter`
with an error code, on success it just calls `UpdateState` — it never emits `PicResult` either.
So in the CURRENT atlas-login implementation, `PicResultWriter` is declared-but-dead: this
diagnosis is about the wire opcode assignment being wrong, not about atlas actually sending a
broken packet today.

For the client-side identity, the decisive evidence turned out to be **already sitting in the
coverage matrix** (`docs/packets/audits/status.json`, read-only — not modified per the
coordinator's instruction):

| op | fname | gms_v83 | gms_v87 | gms_v95 |
|---|---|---|---|---|
| `CHECK_SPW_RESULT` | `CLogin::OnCheckSPWResult` | opcode 28 | opcode 28 | **opcode 27** |
| `inventory/clientbound/InventoryAdd` | `CWvsContext::OnInventoryOperation` | opcode 29 (verified) | opcode 29 | **opcode 28 (verified)** |

This is an independent, pre-existing, separately-derived record (not created by me) that
confirms exactly what my own IDA trace found in this session:
- For v83/v87, `OnCheckSPWResult` sits at 28 (0x1C) — the *same* slot the templates label
  `PicResult` — and `OnInventoryOperation` sits one slot higher, at 29 (0x1D), matching those
  templates' `CharacterInventoryChange` entries exactly.
- For v95, both shifted down by exactly one slot: `OnCheckSPWResult` → 27 (0x1B),
  `OnInventoryOperation` → 28 (0x1C) — matching my IDA trace of `CWvsContext::OnPacket` case 28 in
  session `e4abcb98` precisely, and explaining the mismatch found in the original stop: `PicResult`
  kept its v83-era opcode (0x1C) instead of shifting down with `OnCheckSPWResult` to 0x1B when
  v95's opcode table moved the inventory packet into 0x1C.

**Outcome: `PicResult` exists in v95, at a different opcode — 0x1B (27), not 0x1C (28).** This is
outcome (a) from the three offered: fixed to the correct value, backed by both my own IDA trace
of the v95 dispatch (`CLogin::OnPacket` @ `0x5df940`, and the CHECK_SPW_RESULT/InventoryAdd pair
in `status.json`) and the pre-existing matrix record for the same fname across three versions.

## Step C — wired

`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`:

```
      {
        "opCode": "0x1B",
        "writer": "PicResult",
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

Sorted order preserved: `0x19 → 0x1B → 0x1C → 0x1E`. `PicResult`'s shape (`services: ["login"]`,
no `options`) is unchanged from before — only its `opCode` moved. `CharacterInventoryChange`
matches the key set used by `template_gms_87_1.json`/`template_gms_92_1.json`'s
`CharacterInventoryChange` entries, with the `options.petSkill.autoSpeaking: "0x100"` bit
task-139 IDA-verified for every GMS version.

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

**Absent-writer note count: unchanged (0 → 0).** Confirmed by stashing the template change and
re-running the same check: baseline was already `0 absent-writer note(s)`, and it's still `0`
after the fix. `packet-audit operations --check`'s "absent-writer" tracking is scoped to
dispatcher-family `operations` mode tables (per-arm sub-dispatch), not top-level template writer
completeness — `CharacterInventoryChange` is a plain writer entry, not part of an `operations`
mode table, so its prior absence was never counted there, and the fix doesn't change that
metric's denominator either.

**Matrix regeneration: not required, and not touched.** `docs/packets/audits/status.json` already
records `inventory/clientbound/InventoryAdd × gms_v95 = verified, opcode 28` and
`CHECK_SPW_RESULT × gms_v95 = incomplete, opcode 27` — both predate this change and both already
match the corrected template. This template fix brings the tenant socket config into agreement
with an already-pinned matrix record; it doesn't add, remove, or alter any matrix cell. Per
instructions, `docs/packets/feature-na-evidence.yaml`, `docs/packets/audits/STATUS.md`, and
`status.json` were not touched.

## Files changed

- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — `PicResult`
  0x1C→0x1B; `CharacterInventoryChange` added at 0x1C.
