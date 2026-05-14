# Login Domain v95 Audit — Spike Report

**Scope:** 6 representative login packets compared against v95 IDA decompiles, to surface real version drift before designing the full audit pipeline and version-conditional encoder strategy.

**Sources:**
- `docs/packets/MapleStory Ops - ClientBound.csv` / `ServerBound.csv` — op→FName→version-opcode mapping
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — v95 op→writer/handler mapping
- IDA MCP against `E:\Programs\Nexon\IDBs_v9\GMS\v95_0\GMS_v95.0_U_DEVM.exe` (md5 `3c71fd8872d5efbe16183ae8c51f887d`)
- `libs/atlas-packet/login/{clientbound,serverbound}/*.go` — current Atlas implementation

**Versioning model under evaluation:** version-conditional fields inside the encoder (driven off `tenant.MustFromContext(ctx).MajorVersion()` / `Region()`, the pattern `AuthSuccess.Encode` already establishes).

---

## Packet 1 — `AuthSuccess` ← `CLogin::OnCheckPasswordResult` (success branch)

- **IDA:** `?OnCheckPasswordResult@CLogin@@IAEXAAVCInPacket@@@Z` @ `0x5dc600`, size `0xb2e`
- **Template (v95):** writer `AuthSuccess`, opcode `0x00`
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_success.go`
- **Status of existing version conditioning:** ALREADY USES `t.MajorVersion()` / `t.Region()` branches — confirms the pattern is viable.

### v95 wire layout (decoded from IDA, success branch with `byte0 == 0` and `byte1 ∈ {0,1}`)

| # | Type     | Field                          | Notes |
|---|----------|--------------------------------|-------|
| 1 | `byte`   | resultCode                     | must be 0 to reach success |
| 2 | `byte`   | post-auth flag                 | must be 0 or 1 to reach main body |
| 3 | `int32`  | (unused / reserved)            | always decoded in v95 |
| 4 | `int32`  | accountId                      | |
| 5 | `byte`   | gender                         | `0xA` = unverified path |
| 6 | `byte`   | GM / admin flag                | bool-ish |
| 7 | **`int16`** | subGradeCode + testerAccount | **packed: `lo=subGradeCode`, `hi.bit0=testerAccount`** |
| 8 | `byte`   | countryCode                    | |
| 9 | `string` | nexonClubID                    | **account-side string, not character name** |
| 10| `byte`   | purchaseExp / quiet-ban reason | |
| 11| `byte`   | quiet-ban code                 | |
| 12| `int64`  | chatUnblockDate (FILETIME)     | |
| 13| `int64`  | registerDate (FILETIME)        | |
| 14| `int32`  | numOfCharacter                 | |
| 15| `byte`   | pinFlag (`v43`)                | branches to `SendPacket(11)` if non-zero |
| 16| `byte`   | picFlag (`v44`)                | stashed at `pCtx[432]` |
| 17| `int64`  | clientKey / MAC                | decoded **unconditionally** at `LABEL_94` |

### Diff vs `auth_success.go`

| # | Atlas writes                | v95 reads      | Verdict |
|---|-----------------------------|----------------|---------|
| 1 | `WriteByte(0)`              | byte           | ✅ match |
| 2 | `WriteByte(0)`              | byte           | ✅ match |
| 3 | `WriteInt(0)` when GMS      | int32 (always) | ⚠️ region-gated in Atlas; v95 is GMS so OK in practice, but the gating premise is wrong — v95 client decodes this byte sequence whether or not we're in GMS |
| 4 | `WriteInt(accountId)`       | int32          | ✅ |
| 5 | `WriteByte(gender)`         | byte           | ✅ |
| 6 | `WriteBool(false)`          | byte (GM)      | ✅ |
| 7 | **`WriteByte(0)`** "admin"  | **`int16`**    | ❌ **1 byte short** — every field after this is misaligned on v95. This is the load-bearing bug for v95 support. |
| 8 | `WriteByte(0)` (cc, v>12)   | byte           | ✅ once admin is widened |
| 9 | `WriteAsciiString(m.name)`  | string (nexonClubID) | ⚠️ **semantic mismatch**: writes character name into the field v95 treats as account/club ID. Wire-compatible; behaviorally wrong if client surfaces this string. |
| 10| `WriteByte(0)` "qbr"        | byte           | ✅ |
| 11| `WriteByte(0)` "qb"         | byte           | ✅ |
| 12| `WriteLong(0)` "qb ts"      | int64          | ✅ wire (mislabeled — it's chat unblock, not quiet ban) |
| 13| `WriteLong(0)` "creation"   | int64          | ✅ |
| 14| `WriteInt(1)` "nNumOfChar"  | int32          | ✅ |
| 15| `WriteBool(!usesPin)`       | byte (pinFlag) | ⚠️ inverted semantic — see note |
| 16| `WriteByte(needsPic)`       | byte (picFlag) | ✅ wire |
| 17| `WriteLong(0)` only if v≥87 | int64 (always) | ✅ for v95; gating is fine since older versions presumably omit |

**Semantic note on (15):** v95 treats `v43 != 0` as "needs follow-up" (client immediately sends a `0x0B` packet back with no UI). Atlas writes `!usesPin`, meaning when `usesPin == false` Atlas sends `1` here, triggering the no-UI fast path. That may be intentional (matches existing v83 behavior) — flagged for design review, not declared a bug.

### Drift summary

| Severity | Issue | Recommended encoder shape |
|----------|-------|---------------------------|
| **Blocker** | Field 7 width: `byte` vs `int16` | `if mv >= 95 { WriteShort(subGrade) } else { WriteByte(0) }` (subGrade source likely 0 today — packed flags accepted at TODO) |
| Minor | Field 3 region gate too narrow | Drop `if Region == "GMS"` around the `WriteInt(0)` |
| Minor | Field 9 label/source | Add `nexonClubId` to the model alongside `name`; default = `name` for back-compat |
| Cosmetic | Field 12 label says "quiet ban timestamp"; actually chat-unblock | Rename for accuracy |

### Pattern observation

`AuthSuccess.Encode` already nests `t.Region()` and `t.MajorVersion() >= 87` / `> 12` checks. The version-conditional style works — the function is ~50 lines and still readable. Extending it to add a `>= 95` branch for the int16 widening is a 2-line change. Confirms the strategy scales for this packet.

---

## Packet 2 — `CharacterList` ← `CLogin::OnSelectWorldResult`

- **IDA:** `?OnSelectWorldResult@CLogin@@IAEXAAVCInPacket@@@Z` @ `0x5dda00`, size `0x583`
- **Template (v95):** writer `CharacterList`, opcode `0x0B`
- **Atlas file:** `libs/atlas-packet/character/clientbound/list.go` (note — *not* under `login/`; writer name is a cross-domain handle)
- **Status of existing version conditioning:** uses `t.Region()` + `t.MajorVersion() > 87` + `<= 28` branches.

### v95 wire layout (success path)

| # | Type     | Field                          |
|---|----------|--------------------------------|
| 1 | `byte`   | resultCode (must be 0 or 12)   |
| 2 | `byte`   | nCount (character entries)     |
| 3..2+N | (CharacterListEntry × N) | per-character payload |
| 3+N | `byte`   | m_bLoginOpt (`hasPic`)        |
| 4+N | `int32`  | m_nSlotCount                   |
| 5+N | `int32`  | m_nBuyCharCount                |

### Per-entry layout (recursive, drives a sub-audit)

Each entry is `GW_CharacterStat::Decode` + `AvatarLook::Decode` + `byte onFamily` + `byte hasRank` + (if `hasRank != 0`) `16 bytes` rank block (4 × int32: world rank, world rank move, job rank, job rank move).

### Diff vs `list.go`

| Region | Path | Verdict |
|--------|------|---------|
| Wrapper (status, count, hasPic, slotCount, buyCharCount) | GMS `> 87` | ✅ matches v95 |
| `buyCharCount` gating `MajorVersion > 87` | | ✅ for v95 (which is > 87); ⚠️ if older GMS template ever needs it, must extend |
| Per-entry payload | recursive into `model.CharacterListEntry.Encode` | 🔍 **deferred — true drift surface lives in `CharacterStat` + `AvatarLook` sub-structs**, which are shared with character-spawn writers (highest leverage to audit) |

### Drift summary

| Severity | Issue | Notes |
|----------|-------|-------|
| **Deferred — audit the sub-structs** | `CharacterListEntry.Encode` recurses into character stat & avatar look | These structs are shared across login + in-channel spawn packets. They are likely *the* dominant drift surface for v95 enablement. Recommend a dedicated sub-struct audit pass before encoder rewrites. |
| Info | Writer name `CharacterList` lives under `character/` not `login/` | The audit pipeline's writer→file resolver must walk the full `libs/atlas-packet/**/*.go` tree, not assume domain prefix. |

The wrapper itself is healthy — the existing version branches already gate the v95-specific fields correctly. The risk concentration is one level down.

---

## Packet 3 — `ServerListEntry` ← `CLogin::OnWorldInformation`

- **IDA:** `?OnWorldInformation@CLogin@@IAEXAAVCInPacket@@@Z` @ `0x5da7f0`, size `0x2e0`
- **Template (v95):** writer `ServerListEntry`, opcode `0x0A`
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_list_entry.go`
- **Status of existing version conditioning:** uses `t.Region()` + `t.MajorVersion() > 12`.

### v95 wire layout

| # | Type    | Field                       | Notes |
|---|---------|------------------------------|-------|
| 1 | `byte`  | nWorldID                     | dispatch — `0xFF` = end-of-list (`ServerListEnd`) |
| 2 | `string`| sName                        | world display name |
| 3 | `byte`  | nWorldState                  | |
| 4 | `string`| sWorldEventDesc              | |
| 5 | `int16` | nWorldEventEXP_WSE           | |
| 6 | `int16` | nWorldEventDrop_WSE          | |
| 7 | `byte`  | nBlockCharCreation           | |
| 8 | `byte`  | nChannelCount                | |
| 9 | N × channel record | per below | |
| 10| **`int16`** | nBalloonCount (!)        | NOT just a balloon-size sentinel |
| 11| M × balloon (int16 x, int16 y, string msg) | only if count>0 | |

### Per-channel record (v95)

| Field | Type | Atlas writes |
|-------|------|--------------|
| sName       | `string` | `fmt.Sprintf("%s - %d", worldName, channelId)` ✅ |
| nUserNo     | `int32`  | `WriteInt(capacity)` ✅ |
| nWorldID    | `byte`   | **`WriteByte(1)` — hardcoded** ❌ |
| nChannelID  | `byte`   | `byte(channelId - 1)` ✅ |
| bAdultChannel | `byte` | `WriteBool(false)` ✅ |

### Drift summary

| Severity | Issue | Fix |
|----------|-------|-----|
| **Bug** | Per-channel `nWorldID` hardcoded as `1` | Change to `byte(m.worldId)` — breaks multi-world tenants on any client version |
| Cosmetic | "balloon size" is actually `nBalloonCount` (int16) | Rename + leave value at 0 (no balloons sent) |
| Info | Top-level "balloon size" gate is `> 12` (GMS) — matches v95 expectation | ✅ no action |

**Cross-version note:** This bug isn't v95-specific — it's wrong for any version, but only visible when serving worlds other than ID 1. Spike turned up a latent bug unrelated to versioning.

---

## Packet 4 — `ServerIP` ← `CLogin::OnSelectCharacterResult`

- **IDA:** `?OnSelectCharacterResult@CLogin@@IAEXAAVCInPacket@@@Z` @ `0x5dea80`, size `0x358`
- **Template (v95):** writer `ServerIP`, opcode `0x0C`
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_ip.go`
- **Status of existing version conditioning:** `(GMS && > 12) || JMS` for `ulPremiumArgument`.

### v95 wire layout (success path: `v3==0`, or `v3==12 && subMode∈{0xB,0xD}`, or `v3==23`)

| # | Type    | Field                  |
|---|---------|------------------------|
| 1 | `byte`  | resultCode (v3)        |
| 2 | `byte`  | subMode (dwCharacterID switch) |
| 3 | `int32` | ipv4 (4 octets)        |
| 4 | `int16` | port                   |
| 5 | `int32` | dwCharacterID          |
| 6 | `byte`  | bAuthenCode            |
| 7 | `int32` | ulPremiumArgument      |

### Diff vs `server_ip.go`

Wire-perfect for v95. Only finding:

| Severity | Issue |
|----------|-------|
| Cosmetic | Atlas field `clientId` is actually `dwCharacterID` (the character to spawn in channel) — rename for accuracy |

This packet is the easiest to migrate — no real version drift, the existing `> 12` gate already covers v95.

---

## Packet 5 — `LoginHandle` decode (`Request`) ← `CLogin::SendCheckPasswordPacket`

- **IDA:** `?SendCheckPasswordPacket@CLogin@@QAEHPBD0@Z` @ `0x5db9d0`, size `0x46c`
- **Template (v95):** handler `LoginHandle`, opcode `0x01`
- **Atlas file:** `libs/atlas-packet/login/serverbound/request.go` (struct `Request`, NOT `login_auth.go` — the `login_auth.go` under `clientbound/` is a different writer entirely)
- **Status of existing version conditioning:** uses `Region() == "GMS" && MajorVersion() >= 95` already — confirms versioning hook already in this file.

### v95 wire layout (from IDA `SendCheckPasswordPacket`)

| # | Type    | Field              | Source in IDA |
|---|---------|--------------------|---------------|
| 1 | `string`| **password**       | `sPasswd` |
| 2 | `string`| **passport**       | `szPassport` filled by `CNMCOClientObject::GetNexonPassport` |
| 3 | `16 bytes`| machineId        | `CSystemInfo::GetMachineId` |
| 4 | `int32` | gameRoomClient     | `CSystemInfo::GetGameRoomClient` |
| 5 | `byte`  | gameStartMode      | `m_nGameStartMode` |
| 6 | `byte`  | (const 0)          | `Encode1(0)` literal |
| 7 | `byte`  | (const 0)          | `Encode1(0)` literal |
| 8 | `int32` | partnerCode        | `CConfig::GetPartnerCode` |

**The username (`sID` parameter) is never put on the wire.** It is stashed in `CWvsContext` for UI display. v95 authenticates against the Nexon backend client-side via `CNMCOClientObject::LoginAuth`, obtains a passport token, and sends only the token. The server is expected to validate the passport with Nexon to recover the user identity.

### v83 comparison (inferred from existing Atlas decoder + Cosmic-family server conventions)

`name, password, hwid, gameRoomClient, gameStartMode, unknown1` — username sent in-band, no passport, no partner code.

### Diff vs `request.go` Decode

| # | Atlas reads (v95 path)      | v95 sends            | Verdict |
|---|------------------------------|----------------------|---------|
| 1 | `ReadAsciiString` → name     | password             | ❌ **field semantic swap** |
| 2 | `ReadAsciiString` → password | passport             | ❌ **field semantic swap; passport is a Nexon token, not a password** |
| 3 | `ReadBytes(16)` → hwid       | machineId (16 bytes) | ✅ wire (label is "hwid" but it's machineId — same wire) |
| 4 | `ReadUint32` → gameRoomClient| int32                | ✅ |
| 5 | `ReadByte` → gameStartMode   | byte                 | ✅ |
| 6 | `ReadByte` → unknown1        | byte (const 0)       | ✅ |
| 7 | `ReadByte` → unknown2 (>=95) | byte (const 0)       | ✅ |
| 8 | — (not read)                 | `int32` partnerCode  | ❌ **missing — 4 bytes of trailing data left unread** |

### Drift summary

| Severity | Issue | Strategy implication |
|----------|-------|----------------------|
| **Blocker — architectural** | v95 introduces Nexon passport auth; username is not sent | Strategy of "version-conditional fields inside encoder" is **insufficient on its own** here: this isn't a field add, it's a different auth flow. Either (a) restrict v95 support to clients that bypass Nexon auth (modified clients only), or (b) implement passport validation against Nexon. Worth pinning down before designing the encoder pattern. |
| **Bug for v95** | Missing trailing `int32 partnerCode` | Add v95 branch reading the int32; otherwise the next packet's framing on a keep-alive connection would shift (though TCP packet framing is per-packet so this only matters if the socket reuses the buffer). |
| **Bug for v95** | Atlas reads `name` first, v95 sends `password` first | If only modified v95 clients are targeted, this swap is just a relabel; if stock v95 is targeted, the field-1 and field-2 semantics must swap conditionally. |

**This is the single most consequential finding from the spike.** It shows the version-conditional pattern works for additive drift (cf. AuthSuccess, CharacterList) but is *strained* by structural protocol changes like v95's passport auth. The PRD needs to decide whether stock v95 clients are in scope before committing to the encoder-conditional approach for serverbound packets.

---

## Packet 6 — `CharacterSelectedHandle` decode (`CharacterSelect`, no-PIC) ← `CLogin::SendSelectCharPacket`

- **IDA:** `?SendSelectCharPacket@CLogin@@QAEXXZ` @ `0x5da2a0`, size `0x534`
- **Template (v95):** handler `CharacterSelectedHandle`, opcode `0x13`
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select.go`
- **Status of existing version conditioning:** `GMS && > 12` for MAC/HWID strings.

### v95 has THREE wire formats from one function

`SendSelectCharPacket` branches on `m_bLoginOpt`:

| `m_bLoginOpt` | Sent opcode | Wire format | Atlas handler |
|---------------|-------------|-------------|---------------|
| 0 (PIC register) | 0x1C/0x1F+0x20 | `byte(1)`, `int32 charId`, `string mac`, `string macHdd`, `string sPic` | `RegisterPicHandle` / `CharacterViewAllSelectedPicRegisterHandle` |
| 1 (PIC verify)   | 0x1D/0x1E      | `string sPic`, `int32 charId`, `string mac`, `string macHdd` | `CharacterSelectedPicHandle` / `CharacterViewAllSelectedPicHandle` |
| 2 or 3 (no PIC)  | **0x13**       | `int32 charId`, `string mac`, `string macHdd` | **`CharacterSelectedHandle`** ← this audit |

### Diff vs `character_select.go` Decode (no-PIC variant, opcode `0x13`)

| # | Atlas reads (v95)           | v95 sends   | Verdict |
|---|------------------------------|-------------|---------|
| 1 | `ReadUint32` → characterId   | int32       | ✅ |
| 2 | `ReadAsciiString` → mac      | sMacAddress | ✅ |
| 3 | `ReadAsciiString` → hwid     | sMacAddressWithHDDSerial | ✅ (label says "hwid"; v95 source calls it "MAC + HDD serial" — same wire, semantic nit) |

### Drift summary

| Severity | Issue |
|----------|-------|
| Cosmetic | Field `hwid` is `sMacAddressWithHDDSerial` (a combined string) — rename to clarify it's not the same hwid sent by `LoginHandle` |
| Info — out of scope | PIC-handling siblings (`CharacterSelectedPicHandle` etc.) handle the other two wire shapes and need their own audit. |

The no-PIC path is wire-correct for v95. The PIC paths are where complexity (and likely drift) lives — flagged for follow-up.

---

## Spike Summary

### Findings by severity

**Blockers (must address before v95 production support):**
1. **`AuthSuccess` field 7 width (byte vs int16):** misaligns every subsequent field. Highest-impact wire bug for v95.
2. **`LoginHandle` request format change:** v95 uses Nexon passport auth and reorders fields. Not a pure additive drift — challenges the "version-conditional fields inside the encoder" strategy.

**Bugs (cross-version, surfaced by spike):**
3. **`ServerListEntry` per-channel `nWorldID` hardcoded to `1`:** breaks any tenant serving worlds other than world 1.

**Deferred / scope-expanding:**
4. **`CharacterListEntry` sub-struct** (`CharacterStat` + `AvatarLook`): likely *the* dominant drift surface for v95 enablement. Shared with character-spawn writers — recommend a dedicated sub-struct audit pass.
5. **PIC-handling serverbound siblings** (`CharacterSelectedPicHandle` family): need their own audit; PIC presence/order changed between versions.

**Cosmetic / semantic:**
6. Multiple field labels are off (e.g. `clientId` = `dwCharacterID`, "quiet ban timestamp" = `chatUnblockDate`, `hwid` in `Request` = `machineId`, `hwid` in `CharacterSelect` = `MacWithHDDSerial`).

### Strategy validation: "version-conditional fields inside the encoder"

| Class of drift | Strategy fits? | Examples |
|----------------|----------------|----------|
| **Field appended for newer version** | ✅ Yes — already in use | `AuthSuccess` field 17 (`>= 87`), `CharacterList` `nBuyCharCount` (`> 87`), `ServerIP` `ulPremiumArgument` (`> 12`), `CharacterSelect` MAC strings (`> 12`) |
| **Field width changed** | ✅ Yes, with discipline | `AuthSuccess` admin byte→int16 (`>= 95`) |
| **Field semantic / source changed (same wire)** | ⚠️ Works, but obscures intent in code | `AuthSuccess` name vs nexonClubID; `CharacterSelect` "hwid" vs MacWithHDDSerial |
| **Field order swap / replacement** | ❌ **Strained** — pollutes the decoder with two-arm `if` covering most of the function | `LoginHandle` v83 `(name, pw)` vs v95 `(pw, passport)` |
| **Entire flow / auth model changed** | ❌ **Not appropriate** — needs out-of-band integration (Nexon passport validation) | `LoginHandle` v95 |

**Verdict:** the strategy is right for **the long tail of clientbound packets** (most drift is additive). For the short list of serverbound packets where the auth/select flow restructures — especially `LoginHandle` — the PRD should either:
- (a) restrict v95 to "modified clients" that retain the v83 wire shape (much smaller engineering surface), or
- (b) split per-version sub-files for the few packets with structural drift while keeping additive packets in one file.

### Recommended audit pipeline shape (informed by spike)

1. **CSV→IDA resolver:** parse CSV, filter rows with non-zero v95 opcode, resolve each `FName` via `mcp__ida-pro__get_function_by_name` → address. ~270 rows total (clientbound + serverbound).
2. **Field extractor:** for each address, `decompile_function` and grep the `Decode1/2/4/8/Buffer/Str` (or `Encode*`) call sequence. Output a normalized field list (`[byte, byte, int32, int32, byte, ...]`).
3. **Atlas locator:** for each writer/handler name, locate the `.go` file across **all** `libs/atlas-packet/**/*.go` (writers can live outside the `login/` domain — cf. `CharacterList` → `character/clientbound/list.go`).
4. **Diff renderer:** static read of `Encode`/`Decode` (regex on `w.Write*` / `r.Read*` calls) to extract Atlas's sequence; compare against IDA's. Emit a per-packet markdown row.
5. **Human review gate:** never auto-edit `.go` files. Output is a report per-packet; humans apply fixes.

Recursive sub-struct handling (`CharacterStat`, `AvatarLook`, `ChannelLoad`, etc.) wants a second resolver layer keyed on the C++ struct name (`get_defined_structures` / `get_struct_info_simple` from IDA MCP).

### Open questions for `/spec-task`

- Are stock-Nexon v95 clients in scope, or only modified ones that retain v83 wire shape?
- Where should version-source live for encoders that don't currently take tenant context (any?)? `AuthSuccess` already uses `tenant.MustFromContext` — confirm this is universal in atlas-packet.
- Are we comfortable producing a parallel writer per version when drift is structural (e.g. `LoginHandle.v95.go`), or is one-file-many-branches the strict rule?

### Time accounting

Spike covered 6 packets in roughly half the budget; the IDA→atlas-packet resolution is fast enough that a full login-domain audit (~40 packets) is one focused session. The sub-struct audit (`CharacterStat`, `AvatarLook`) is the gating cost for total project scope — those structs participate in 20+ writers across login + channel domains.

