# task-096 CField family batch — LEFT_KNOCK_BACK / OnStalkResult / ADMIN_RESULT / GUILD_OPERATION

IDA-derived client read orders for the four-op batch. Ports (confirmed via
`list_instances`, session 2026-06-15): v83=13342, v84=13337, v87=13341,
v95=13340, jms=13339. **v95 authoritative.**

Widths: 1 = Decode1 (byte), 4 = Decode4 (int32), str = DecodeStr
(int16 len-prefixed ASCII).

---

## Op 1 — LEFT_KNOCK_BACK (clientbound, EMPTY body) — struct `SnowballTouch`

- fname: `CField_SnowBall::OnSnowBallTouch`
- **EMPTY** — reads zero packet bytes. The function only calls
  `CUserLocal::SetImpact(0x12C, 1)` (knockback impulse); no `Decode*`.
  Confirmed by disasm in v84 (@0x584ceb) and v87 (@0x5a35f7): `push 1; push 12Ch;
  call SetImpact; retn`. v83/v95/jms export entries already carry `calls: []`.
- Per-version clientbound opcodes (CSV ClientBound authoritative; CSV has no v84
  column → v84 read from dispatcher):
  - v83 = 284 (0x11C)
  - v84 = 291 (0x123) — IDA: `sub_5849C6` SnowBall dispatcher case 291 → sub_584CEB
    (State 288 / Hit 289 / Msg 290 / **Touch 291**)
  - v87 = 301 (0x12D)
  - v95 = 341 (0x155)
  - jms = 308 (0x134)
- Export addresses: v83 0x575372, v84 0x584ceb, v87 0x5a35f7 (re-harvested this
  batch — was previously absent), v95 0x560510, jms 0x5c986c.
- Applicability: ALL 5 versions.

## Op 2 — IDA_0X09C / OnStalkResult (clientbound) — struct `StalkResult`

- fname: `CField::OnStalkResult`
- Real wire (count-prefixed loop; v83 @0x537a6a and v95 @0x539910 byte-identical):
  - Decode4 count
  - loop count times:
    - Decode4 charId
    - Decode1 flag (1 = RemoveStalkee, 0 = InsertStalkee)
    - if flag == 0: DecodeStr name + Decode4 x + Decode4 y (tagPOINT)
  The `InsertStalkee`/`RemoveStalkee`/`_Release` calls are UI application logic
  (non-wire) and are the `Delegate` entries in the export.
- Representative flat read order (matches export `calls`, one insert entry,
  version-invariant): Decode4(count) + Decode4(charId) + Decode1(flag) +
  DecodeStr(name) + Decode4(x) + Decode4(y).
- Per-version opcodes (CSV ClientBound; the op lands in a different `IDA_0X..`
  STATUS row per version because the placeholder op-name is opcode-derived):
  - v83 = 156 (0x9C)  [STATUS row IDA_0X09C]
  - v87 = 164 (0xA4)  [STATUS row IDA_0X0A4]
  - v95 = 172 (0xAC)  [STATUS row IDA_0X0AC]
  - jms = 152 (0x98)  [STATUS row IDA_0X098]
- Export addresses: v83 0x537a6a, v87 0x55f3e5, v95 0x539910, jms 0x574ca3.
- Applicability: v83/v87/v95/jms. **v84 ABSENT** (no `OnStalkResult` in the v84
  IDB/export; the foothold/stalk cluster is version-divergent) → ⬜ VERSION-ABSENT.

## Op 3 — ADMIN_RESULT (clientbound MODE-DEMUX) — struct `AdminResult`

- fname: `CField::OnAdminResult`
- Leading `Decode1(mode)` switch; each mode reads a different field set. v95
  (@0x53bc20) cases include: 4/5/6 (Decode1), 0xB (DecodeStr channel [+world+msg]),
  0x12 (Decode1), 0x15 (Decode1 flag [+Decode1 channel | +Decode4 mapId]),
  0x28/0x29 (none), 0x2A/0x2B (Decode1), 0x33-0x39/0x3A/0x47/0x48 (DecodeStr).
- Modeled like SPOUSE_CHAT: a leading `mode byte` + the flattened union of the
  representative mode fields, matching the export's flat `calls` POSITIONALLY so
  the round-trip closes and the analyzer (which can't parse switch guards) grades
  ✅. The flattened union DIFFERS PER VERSION (the export captured each binary's
  flat read), so Encode/Decode branch by version to emit each version's exact
  flat field sequence. Non-wire `Delegate` calls (ZXString operator+ string
  concat — v84 sub_476592, jms sub_4A586D) are stripped from those export entries
  (the §10 sanctioned report-gen fix, same surgical strip applied throughout
  task-096's "export strip" commits).
- Per-version flat read orders (Delegate-stripped):
  - v83 @0x5352e9: 1,str,1,1,1,4,1,1,str,str,str,1,1,1  (14)
  - v84 @0x54156f: 1,1,str,str,1,1,1,4,1,str,str,str,1,1,1  (15, after stripping 2 Delegate)
  - v87 @0x55cac3: 1,1,1,1,str,str,str,1,1,1,4,1,1,str,str  (15)
  - v95 @0x53bc20: 1,1,1,1,str,str,str,1,1,1,4,1,1,str,str,str,str  (17)
  - jms @0x57255f: 1,1,str,str,str,1,1,1,1,1,4  (11, after stripping 3 Delegate)
- Per-version opcodes:
  - v83 = 144 (0x90)
  - v84 = 147 (0x93) — IDA: `CField::OnPacket` @0x53D5A7 case 0x93 → sub_54156F
  - v87 = 152 (0x98)  (CSV/registry; NOT 159/160 — 159 is the v92 column)
  - v95 = 160 (0xA0)
  - jms = 141 (0x8D)
- Applicability: ALL 5 versions.

## Op 4 — GUILD_OPERATION (serverbound) — existing codec `guild/serverbound/Operation`

- audit-id: `guild/serverbound/GuildOperation` (= qualifiedWriterName guild+Operation).
- Registry primary fname (all versions): `CUIFadeYesNo::OnButtonClicked` (the
  GUILD_OPERATION sub-op-byte dispatcher). This is how v95/jms verified — pinned
  against a SYNTHETIC export entry (address 0x0, single `Encode1` op byte; the
  v95 entry was hand-authored in task-066 #609).
- Existing codec `Operation` reads only `op byte` (Decode1). Matches the synthetic
  1-byte dispatcher entry. `CField::InputGuildName` (the task's suggested fname)
  maps in `candidatesFromFName` to a DIFFERENT codec (`OperationRequestCreate` =
  op byte + guild name string), so pinning GuildOperation against InputGuildName
  would mis-grade (atlas-short). Correct path = mirror v95: synthetic
  `CUIFadeYesNo::OnButtonClicked` entry.
- `CUIFadeYesNo::OnButtonClicked` is ABSENT from the v83/v84/v87 exports (and the
  symbol isn't directly resolvable in those IDBs — it's a synthetic record, not a
  harvested function). To verify v83/v84/v87 the same way: APPEND a synthetic
  `CUIFadeYesNo::OnButtonClicked` entry (address 0x0, single `Encode1` op byte) to
  each of those exports — genuinely append-only (new key, 0 deletions).
- Per-version serverbound opcodes: v83 0x7E (126), v84 0x82 (130) [template;
  registry says 126 — pre-existing mismatch, template wins for routing], v87 0x86
  (134), v95 0x95 (149), jms 0x81. v83/v84 already routed with validators; v87
  needs the GuildOperationHandle route added.
