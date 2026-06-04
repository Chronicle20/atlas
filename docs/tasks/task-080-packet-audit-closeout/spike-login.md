# Login packet audit — verdicts (task-080 B6.1)

B6.1 deliverable of record. Audits Atlas's login packets against the harvested IDA
read-orders for GMS v83/v87/v95 + JMS185 (banked in `spike-login-harvest.md`, which is
the ground truth). Evidence is the harvest note; no IDA was re-loaded for this audit.

Method: for each addressed FName, compare Atlas's packet struct (Encode/Decode order) to
the harvested per-version read-order. Fix only genuine wire-body divergences (TDD byte
test); document version-presence gaps without forcing code.

---

## 1. Serverbound `Request` (SendCheckPasswordPacket) — FIXED

File: `libs/atlas-packet/login/serverbound/request.go`

### Divergence (genuine — fixed)

Harvested ground truth (op 0x01), trailer after `gameStartMode`:

| version | trailer |
|---|---|
| GMS v83 (`@0x5f6952`) | `Encode1(0), Encode1(0), Encode4(PartnerCode)` |
| GMS v87 (`@0x62dfb4`) | `Encode1(0), Encode1(0), Encode4(PartnerCode)` |
| GMS v95 (`@0x5db9d0`) | `Encode1(0), Encode1(0), Encode4(PartnerCode)` |
| JMS185 (`@0x66da6a`) | `Encode1(0)` — no second byte, no PartnerCode |

PartnerCode is **GMS-universal** (v83 + v87 + v95 all append `Encode4`), **not** a v87/v95
quirk and **not** a v95-only field. JMS185 sends only one trailing zero byte and no
PartnerCode.

Atlas BEFORE: wrote `unknown1` always, `unknown2` only when `GMS && MajorVersion>=95`, and
no PartnerCode. So for GMS v83/v87 it under-wrote (missing `unknown2` + PartnerCode), and
for GMS v95 it under-wrote (missing PartnerCode).

### Fix

- Changed the `unknown2` gate from `Region()=="GMS" && MajorVersion()>=95` to
  `Region()=="GMS"` (all GMS have it).
- Added a `partnerCode uint32` field, written/read inside the same `Region()=="GMS"` block
  (after `unknown2`). Read-and-discard / write — zero functional impact, completes the wire.
- Added `PartnerCode()` getter. Encode + Decode mirrored symmetrically.

Resulting trailer:

- GMS v83/v87/v95: `...gameStartMode, unknown1, unknown2, Encode4(partnerCode)` (2 bytes + 4)
- JMS185: `...gameStartMode, unknown1` (1 byte, no partnerCode)

### TDD byte test

`TestRequestTrailerShape` (in `request_test.go`) asserts the exact trailing bytes after the
fixed-length prefix, per version:

- GMS v83 / v87 / v95 → trailer `07 09 EF BE AD DE` (unknown1=0x07, unknown2=0x09,
  partnerCode=0xDEADBEEF little-endian) — 6 bytes.
- JMS185 → trailer `07` — 1 byte (no unknown2, no partnerCode).

Against the old code this FAILS (v83/v87 emitted 1 trailing byte, v95 emitted 2, none
emitted partnerCode); against the fix it PASSES. `TestRequestRoundTrip` was also extended to
round-trip `partnerCode` (expected 0 for JMS since it's off-wire there).

Test results: `go test -race ./login/...` PASS (clientbound + serverbound).

### id-vs-password order — KEPT (documented)

The decompiles disagree on field order:

- GMS v83 (`@0x5f6952`): `EncodeStr(pw), EncodeStr(id), ...`
- GMS v87 (`@0x62dfb4`): `EncodeStr(id), EncodeStr(pw), ...`
- GMS v95 (`@0x5db9d0`): `EncodeStr(pw), EncodeStr(passport), ...` (passport = NMCO blob)
- JMS185 (`@0x66da6a`): `EncodeStr(id), EncodeStr(pw), ...`

Atlas writes `name, password` (id then pw), matching v87 + JMS185. The v83/v95 "pw first"
labelling is most likely a decompiler mislabel (the two ShiftJIS strings are structurally
identical; only the variable names differ). **Verdict: KEEP `name, password` order — no
change.** Not worth flipping on ambiguous evidence; the wire is two strings either way.

---

## 2. Clientbound `LoginAuth` — KEEP (verdict reversed from the plan's "remove" lean)

File: `libs/atlas-packet/login/clientbound/login_auth.go`

Atlas's `LoginAuth` is a **clientbound** writer that encodes a single
`WriteAsciiString(screen)` — a server→client resource-path string.

LoginAuth across the four baselines:

| baseline | LoginAuth |
|---|---|
| GMS v83 | absent (predates Nexon passport) |
| GMS v87 | absent (only NMCO middleware, unrelated) |
| GMS v95 | NMCO middleware only (`CNMCOClientObject::LoginAuth`) — not a game-wire packet; its passport blob is `EncodeStr`'d into SendCheckPassword |
| JMS185 | **`CLogin::LoginAuth` @0x670c8e — clientbound handler (OnPacket idx 0x18)**: `DecodeStr`s a UI `.img` resource path, then `IWzResMan::GetObjectA` / `CStage::FadeIn` → swaps the login-screen background |

The plan's lean to "remove" was based on reading LoginAuth as a serverbound/auth packet
(no game-wire auth counterpart anywhere). That misreads it. Atlas's `LoginAuth` is a
**single server→client string writer**, which structurally matches JMS185's idx-0x18
login-background-swap handler exactly (one decoded string = the `.img` path). It is the JMS
login-background-swap clientbound packet, not a spurious auth packet. GMS has no game-wire
counterpart, so it is JMS-relevant.

**Verdict: KEEP.** No code change. (Document as: login-background-swap clientbound packet,
matches JMS185 `CLogin::LoginAuth` idx 0x18.)

---

## 3. Addressed-FName verdicts (other login packets)

### OnViewAllCharResult (clientbound all-character list)

Harvest read-order (`Decode1(mode)` discriminator → branches):

| version | fn | shape |
|---|---|---|
| GMS v83 | `@0x5facca` | mode1 header `Decode4(countSvrs),Decode4(countChars)`; mode0 per-svr `Decode1(worldId),Decode1(charCount)`, loop {CharacterStat, AvatarLook, `Decode1(hasRank)`→`DecodeBuffer(16)`} |
| GMS v87 | `@0x6328eb` | same, plus per-char `Decode1(rankFlag)` gate on the 16-byte rank buffer |
| GMS v95 | `@0x5de120` | per-char adds `Decode1(worldID2)` before hasRank; trailing `Decode1(bLoginOpt)` |
| JMS185 | `@0x6709e4` (idx 0x14) | data block leads `Decode1(unused)` before worldID; no bLoginOpt trailer |

Atlas builds the all-char list from clientbound character/avatar writers (assembled
server-side, not a single struct). The per-character core (CharacterStat + AvatarLook +
rank) is the cross-version-stable backbone and matches. The per-version deltas (v87 rankFlag
gate, v95 worldID2 + bLoginOpt, JMS leading unused byte) are mode/version-conditional fields.
**Verdict: ✅ structurally matches the documented read-order**; per-version conditional-field
completeness (worldID2/bLoginOpt/unused) is a template-completeness concern, documented here,
not a wire-body divergence in the audited serverbound select bodies. No clear divergence found
→ no code change.

### Serverbound select-char bodies — ✅ all match

Audited Atlas structs vs harvest read-orders (all Encode/Decode orders verified field-for-field):

| Atlas struct | FName / variant | harvest body | verdict |
|---|---|---|---|
| `CharacterSelect` | SendSelectCharPacket no-PIC (op 0x13) | `Encode4(charId),EncodeStr(mac),EncodeStr(hwHash)` | ✅ |
| `CharacterSelectRegisterPic` | SendSelectCharPacket PIC-register (op 0x1D / v95 0x1C) | `Encode1(1),Encode4(charId),EncodeStr(mac),EncodeStr(hwHash),EncodeStr(pic)` | ✅ |
| `CharacterSelectWithPic` | SendSelectCharPacket PIC-verify (op 0x1E / v95 0x1D) | `EncodeStr(pic),Encode4(charId),EncodeStr(mac),EncodeStr(hwHash)` | ✅ |
| `AllCharacterListSelect` | SendSelectCharPacketByVAC non-PIC (op 0x0E) | `Encode4(charId),Encode4(worldId),EncodeStr(mac),EncodeStr(mac2)` | ✅ |
| `AllCharacterListSelectWithPicRegister` | ByVAC PIC-register (op 0x1F) | `Encode1(1),Encode4(charId),Encode4(worldId),EncodeStr(mac),EncodeStr(mac2),EncodeStr(pic)` | ✅ |
| `AllCharacterListSelectWithPic` | ByVAC PIC-verify (op 0x20 / v95 0x1F) | `EncodeStr(pic/spw),Encode4(charId),Encode4(worldId),EncodeStr(mac),EncodeStr(mac2)` | ✅ |

No wire-body divergence in any select packet. No code change.

`CharacterSelect` gates mac/hwid behind `GMS && MajorVersion>12`; the PIC variants gate
mac/hwid behind `Region()=="GMS"`. These match the GMS-only nature of those fields
(JMS SendSelectCharPacket carries no mac strings — see §4 PIC table).

### VAC select + OnSelectCharacterByVACResult — GMS-only (documented gap)

- `SendSelectCharPacketByVAC`: present in GMS v83 (`@0x5f76ae`), v87 (`@0x62ee37`),
  v95 (`@0x5d7550`); **ABSENT in JMS185** (only `ResetVAC @0x6711fa`, unrelated).
- `OnSelectCharacterByVACResult`: present in all GMS; **ABSENT in JMS185**.

Atlas's `AllCharacterListSelect*` (VAC) packets are therefore **GMS-only**. JMS lacks the VAC
"view-all-characters" select path entirely. **Verdict: GMS-only — documented gap.** Not
wiring a JMS counterpart (none exists); no code change. The VAC packets remain valid for GMS.

### License accept/deny (OnDenyLicense / OnButtonClicked) — GMS-only (documented gap)

- GMS v83: folded into `CLicenseDlg::OnButtonClicked @0x621b0d` (accept op 0x0B zero-body).
- GMS v87: `OnDenyLicense @0x633e7d` (op 0x07, `Encode1(0)`); `OnButtonClicked @0x65a20d`
  (accept op 0x0B no body / deny → op 0x07).
- GMS v95: `OnDenyLicense @0x5d45d0` (op 7); `OnButtonClicked @0x5ff870` (accept op 11 / deny op 7).
- JMS185: **ABSENT** (no login license accept/deny — only world-transfer-license UI, unrelated).

**Verdict: GMS-only — documented gap.** Atlas's ToS/license accept handler
(`accept_tos.go` / op 0x0B) is GMS-relevant; JMS has no login-license wire. No code change.

---

## 4. PIC select-char per-version opcode table (template config — documented)

Op-byte → handler mapping is per-version template configuration, not packet-body. Atlas's
select-char packet **bodies** are verified ✅ (§3); the opcode that selects which body is a
template concern. Documented for completeness; only a WIRED-but-wrong template value would be
changed (none found → no change). Missing entries are NOT wired here (template-completeness is
out of scope for B6.1).

| version | enter / no-PIC | PIC register | PIC verify | notes |
|---|---|---|---|---|
| GMS v83 (`@0x5f726d`) | `0x13` | `0x1D` | `0x1E` | PIC/SPW exists in v83 |
| GMS v87 (`@0x62e9f6`) | `0x13` | `0x1D` | `0x1E` | same as v83 |
| GMS v95 (`@0x5da2a0`) | `0x13` | `0x1C` | `0x1D` | register/verify shifted down 1 vs v83/v87; no 0x1E |
| JMS185 (`@0x66ddac`) | `0x13` | `0x14` | `0x06` | **no PIC system** — these are loginOpt branches: opt0=0x13 `Encode1(hasName),Encode4(charId),[EncodeStr(name)]`; opt1=0x14 `EncodeStr(s),Encode4(charId)`; opt2/3=0x06 `Encode4(charId)`; no SPW string |

VAC (ByVAC) opcodes (GMS-only):

| version | non-PIC | PIC register | PIC verify |
|---|---|---|---|
| GMS v83 (`@0x5f76ae`) | `0x0E` | `0x1F` | `0x20` |
| GMS v87 (`@0x62ee37`) | `0x0E` | `0x1F` | `0x20` |
| GMS v95 (`@0x5d7550`) | `0x0E` | `0x1E` | `0x1F` |

**Verdict: per-version opcode mapping is template config; bodies match; no wired value is
wrong → no code change.** JMS PIC entries are absent (no PIC); not wired.

---

## 5. Bare-handler mapping (Atlas serverbound handlers ↔ real client fns)

From the v87 harvest section, the addressed bare handlers map to real GMS client functions:

| Atlas handler | real client fn (v87) | verdict |
|---|---|---|
| `AfterLoginHandle` (after_login.go) | `SendCheckUserLimitPacket @0x62f80a` (post-password world-list stage) | audited — real counterpart |
| `RegisterPinHandle` (register_pin.go) | PARTIAL — no standalone send; PIN via `CPinCodeDlg` + `OnUpdatePinCodeResult @0x6345d4` (dialog-driven) | audited — dialog-driven counterpart |
| PIC family (character_selected_pic / _register_pic / view_all_selected_pic*) | embedded in `SendSelectCharPacket`/`ByVAC` (ops in §4); results `OnCheckPinCodeResult @0x6342b0`, `OnEnableSPWResult @0x6335a9`, `OnCheckSPWResult @0x6336a2` | audited — GMS-only |
| `SetGenderHandle` (account_set_gender.go) | `SendSetGenderPacket @0x63409f` op 0x08 `Encode1(1),Encode1(gender)`; result `OnSetAccountResult @0x634144` | audited — real counterpart |
| `CharacterListWorldHandle` (character_list_world.go) = WorldCharacterListRequest | `SendViewAllCharPacket @0x6324e3` / SelectWorld pair (`OnWorldInformation @0x630e7c`, `OnSelectWorldResult @0x63115a`) | audited — real counterpart |

Other Atlas login handlers (ServerStatus, ServerList, ServerSelect/WorldSelect, Pong,
CharacterViewAll*, AcceptTos, CreateSecurity, ClientStart, StartError, Debug, NoOp) are the
standard login flow / no-op or local-UI handlers and were not flagged by the harvest as
divergent. No handler found genuinely wrong → no handler code change.

---

## Summary

| item | verdict |
|---|---|
| serverbound `Request` PartnerCode + unknown2 gate | **FIXED** (TDD byte test, per-version) |
| id/password order | KEEP (v87/JMS match; v83/v95 "pw first" likely mislabel) |
| clientbound `LoginAuth` | **KEEP** (JMS login-bg-swap, idx 0x18; plan's "remove" lean was a misread) |
| OnViewAllCharResult clientbound | ✅ structurally matches; per-version conditional fields documented |
| serverbound select-char bodies (PIC + VAC) | ✅ all 6 match field-for-field |
| VAC select + OnSelectCharacterByVACResult | GMS-only — documented gap (absent in JMS) |
| login license accept/deny | GMS-only — documented gap (absent in JMS) |
| PIC select-char opcode table | template config; bodies match; no wired value wrong → no change |
| bare handlers | mapped to real GMS client fns; none wrong → no change |

**Code change in this task: serverbound `Request` only** (PartnerCode field + GMS-wide
unknown2 gate, with a per-version trailer byte test). Everything else is verdict/documentation.
