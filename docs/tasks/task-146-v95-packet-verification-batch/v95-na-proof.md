# v95 positive-absence proof: `CreateSecurityHandle`, `WorldSelectHandle`, `ServerLoad`

Task 16 (W4). All three symbols route only in legacy/jms templates today
(`template_jms_185_1.json` opcode `0x1A` for `CreateSecurityHandle`,
`template_gms_12_1.json` opcode `0x03` for `WorldSelectHandle` and opcode
`0x02` for `ServerLoad`) and none is present in `template_gms_95_1.json`
(confirmed by direct grep: zero matches for any of the three symbol names in
that file). That template absence is *our own routing choice*, not evidence
of client behavior, so per `docs/packets/audits/VERIFYING_A_PACKET.md` ("Is
this cell `n-a`?") the claim is proven independently against the v95 client
binary itself (IDA session `ecc757f4`,
`E:\Programs\Nexon\IDBs_v9\GMS\v95_0\GMS_v95.0_U_DEVM.exe.i64`, confirmed via
`idb_list`).

No matrix row exists for any of these three symbols and none is created —
Open Question 5 is answered "no" (see brief). This file is the only record.

---

## 1. `CreateSecurityHandle` — jms-only, opcode `0x1A`, no `fname`

**What is being claimed absent:** whatever client-side "create a security
handle" feature the jms_185 seed template's opcode `0x1A` was wired against.
The Atlas handler (`services/atlas-login/atlas.com/login/socket/handler/create_security.go`)
never decodes the incoming reader at all — it only replies with a
`LoginAuth` "MapLogin"/"MapLogin1" screen packet — so there was never an
Atlas-side field layout to compare against a receive arm.

**Anchor 1 — the symbol doesn't even resolve in the jms_185 IDB itself.**
`func_query {database: a977912e, name_regex: "SecurityHandle|CreateSecurity"}`
against `MapleStory_dump_SCY.exe.i64` (jms v185, the IDB the template's
opcode was presumably modeled on) returns **zero** results. The op this
template entry represents was never traced to a named client function even
in its own origin version — `CreateSecurityHandle` is an Atlas-chosen label
with no known jms fname behind it, let alone a gms one.

**Anchor 2 — the v95 client has no security-handle-shaped class at all.**
`func_query {database: ecc757f4, name_regex: "Security"}` against
`GMS_v95.0_U_DEVM.exe.i64` returns 28 symbols, all under one class:
`CSecurityClient` (`GetInstance`/`CreateInstance`/`InitModule`/`StartModule`/
`OnCheckClientIntegrityRequest`@`0xa1ac10`/`OnPacket`@`0xa1ae80`/etc.) plus a
family of `CSecurityException`/`CSecurityThreatDetected`/`CSecurityInitFailed`
exception types. This is v95's **anti-cheat client-integrity-check**
subsystem (`OnCheckClientIntegrityRequest` takes a `CInPacket&` and is a
challenge/response scan, not an account-security-card "create handle"
feature). It is a structurally different feature family under a
superficially similar class-name prefix — not the thing `CreateSecurityHandle`
was modeling.

**Anchor 3 — the wire slot itself is occupied by something else in v95.**
`docs/packets/registry/gms_v95.yaml` records serverbound opcode `26`
(`0x1A` — the exact opcode `CreateSecurityHandle` occupies in the jms_185
template) as `CLIENT_START_ERROR`, `fname: CClientSocket::OnConnect`; opcode
`27` (`0x1B`) is `CLIENT_ERROR`, `fname: CSecurityClient::OnCheckClientIntegrityRequest`
(the anti-cheat function from Anchor 2, already accounted for under its own
op). Neither is a security-handle-creation flow. The opcode slot v95 assigns
at this position belongs to an already-attributed, unrelated feature.

**Conclusion:** `CreateSecurityHandle` is `n-a` on gms_v95. No matrix row
exists to hold the verdict (Open Question 5); nothing further to record in
tooling.

---

## 2. `WorldSelectHandle` — `template_gms_12_1` only, opcode `0x03`, no `fname`

**What is being claimed absent:** a serverbound op carrying a `worldId` byte
sent by the client after picking a world from the world-select screen
(Atlas model `libs/atlas-packet/login/serverbound/server_select.go`,
`ServerSelect{worldId}`, comment `// CLogin::SendWorldSelectPacket`). There
is no `gms_v12` registry file at all (confirmed: `docs/packets/registry/`
has no `gms_v12.yaml`), so there is no registry fname to check either.

**Anchor 1 — no `Send*WorldSelect*` function exists anywhere in the v95
binary.** `func_query {database: ecc757f4, name_regex: "WorldSelect"}`
returns 37 symbols, every one of them either `CUIWorldSelect` (the
world-select **UI** class: `OnCreate`, `OnKey`, `OnButtonClicked`,
`DrawWorldItems`, `MakeBalloon`, `SetFocusWorld`, …) or
`CUITransferWorldSelectDlg` (a cash-shop world-transfer dialog, unrelated).
`CLogin::GotoWorldSelect`@`0x5daf20` is present but is a **local UI step
transition** (`int bFromVAC` parameter, no `CInPacket`), not a packet sender.
No `SendWorldSelectPacket`, no `SendSelectWorldPacket`, nothing under
`CLogin` that sends a `worldId` byte on the wire when a world is picked.

**Anchor 2 — decompiling the one function this regex surfaces that *does*
send a packet shows it sends the wrong shape to be `WorldSelectHandle`.**
`CLogin::GotoWorldSelect`@`0x5daf20` (called when returning to the
world-select screen, `bFromVAC == 0`) does:
`COutPacket::COutPacket(&oPacket, 12)` @`0x5daf9b` then
`CClientSocket::SendPacket(...)` @`0x5dafb3` with **zero** `Encode*` calls in
between — an empty-body opcode-12 packet. This matches
`docs/packets/registry/gms_v95.yaml`'s `PLAYER_DC` entry (serverbound opcode
12, `fname: CLogin::GotoWorldSelect`) exactly: it is a bare "returning to
world select" notification, not a payload carrying the chosen `worldId`.

**Anchor 3 — the actual "I picked a world" wire message in v95 carries the
choice differently, and is already verified.**
`libs/atlas-packet/login/serverbound/world_character_list_request.go`
(`WorldCharacterListRequest`, Atlas const `WorldCharacterListHandle` =
`CharacterListWorldHandle`, comment `// CLogin::SendLoginPacket`) is routed
at v95 template opcode `0x05`. Its test file carries an IDA-backed marker
`packet-audit:verify packet=login/serverbound/ServerListRequest
version=gms_v95 ida=0x5d9730` for the sibling `ServerListRequest` op (empty
body, routed at both `0x04` and `0x0B`), and the `WorldCharacterListRequest`
codec itself already encodes `worldId` + `channelId` (+ `socketAddr` for GMS
≥72) in one shot with IDA-anchored per-version comments going back to v61.
In other words: v95's client sends world **and** channel together as soon as
the player commits, directly producing a `CharacterList` reply
(`CLogin::OnSelectWorldResult`, opcode `11` in `CLogin::OnPacket`'s switch —
see §3 below) with no separate "world selected, awaiting confirmation" round
trip in between. There is no protocol step left for a standalone
`WorldSelectHandle`(worldId-only) message to occupy.

**Conclusion:** `WorldSelectHandle` is `n-a` on gms_v95. No matrix row
exists; task-folder record only.

---

## 3. `ServerLoad` (writer, `libs/atlas-packet/login/clientbound/server_load.go`)

**What is being claimed absent:** a clientbound op reporting a world-load
state/percentage (`ServerLoad{code byte}`), historically sent after
`WorldSelectHandle` and before the character list, in the legacy
`template_gms_12_1.json` flow (opcode `0x02`) where it is also the response
Atlas's own `WorldSelectHandleFunc` announces
(`services/atlas-login/atlas.com/login/socket/handler/server_select.go`).
There is no `gms_v95` audit report or sub-struct row for it (confirmed: not
in `docs/packets/audits/gms_v95/`).

**Anchor 1 — the complete clientbound login dispatch switch has no arm for
it.** `CLogin::OnPacket`@`0x5df940` (the sole dispatcher for every
`CInPacket`-carrying login-domain server→client message; found by first
enumerating all `On*@CLogin@@` receive handlers via `func_query`, then
decompiling their single common caller) is a flat `switch (nType)` with
explicit cases **0–15, 21, 24–27** and a `default` that falls through to
`CStage::OnPacket` / `CMapLoadable::OnPacket` (shared menu/map-loading
infrastructure, not login-specific) only for `nType` in `{141,142,143}` or
`{144,145,146}`. Every case target was decompiled/enumerated by name in the
same query (`OnCheckPasswordResult`, `OnGuestIDLoginResult`,
`OnAccountInfoResult`, `OnCheckUserLimitResult`, `OnSetAccountResult`,
`OnConfirmEULAResult`, `OnCheckPinCodeResult`, `OnUpdatePinCodeResult`,
`OnViewAllCharResult`, `OnSelectCharacterByVACResult`,
`OnWorldInformation`, `OnSelectWorldResult`, `OnSelectCharacterResult`,
`OnCheckDuplicatedIDResult`, `OnCreateNewCharacterResult`,
`OnDeleteCharacterResult`, `OnEnableSPWResult`, `OnLatestConnectedWorld`,
`OnRecommendWorldMessage`, `OnExtraCharInfoResult`, `OnCheckSPWResult`).
None is named or shaped like a load-percentage/load-gauge handler, and this
switch is *exhaustive* for the type — the `default` arm only ever delegates
to non-login base-class dispatch, it never falls through to an
undocumented `CLogin` member.

**Anchor 2 — mandatory sibling cross-check (playbook rule 3) against
`SERVERSTATUS`, which is present and verified on v95.** Case `3` in the
switch above is `CLogin::OnCheckUserLimitResult`@`0x5d2250`, decompiled:

```
void CLogin::OnCheckUserLimitResult(CLogin *this, CInPacket *iPacket) {
  this->m_bRequestSent = 0;
  v2 = CInPacket::Decode1(iPacket);
  v3 = CInPacket::Decode1(iPacket);
  CUIWorldSelect::UserLimitResult(TSingleton<CUIWorldSelect>::ms_pInstance, v2, v3);
}
```

`xrefs_to(0x606870 /* CUIWorldSelect::UserLimitResult */)` confirms
`OnCheckUserLimitResult` is `UserLimitResult`'s **only** caller. This is the
receive side of the `ServerStatusRequest` → `ServerStatus` pair already
implemented in Atlas
(`libs/atlas-packet/login/serverbound/server_status_request.go`,
`libs/atlas-packet/login/clientbound/server_status.go`, handler
`services/atlas-login/atlas.com/login/socket/handler/server_status.go`,
routed at v95 template opcode `0x06`). Per the sibling-cross-check rule: does
this receive arm decode/populate the *same state* a `ServerLoad` would? No —
it is a coarse two-value capacity/crowding flag shown while a world is
*highlighted* in the world-select list (before commit), not a transient
"please wait, loading NN%" gauge shown *after* a world+channel is already
committed and awaiting entry. The two features are adjacent in the UI flow
but decode different state; `SERVERSTATUS` being present does not prove a
`ServerLoad` arm exists — it proves the opposite question (population
warning) is already answered elsewhere.

**Anchor 3 — the flow that would need a load-gauge step doesn't have room
for one.** Per §2 Anchor 3, `WorldCharacterListRequest`
(`CharacterListWorldHandle`, v95 opcode `0x05`) sends `worldId` + `channelId`
+ `socketAddr` in a single serverbound message, and `CLogin::OnPacket` case
`11` (`OnSelectWorldResult`) is the very next login-domain packet the client
expects to receive after that — it decodes directly into `CharacterList`
(per `docs/packets/spike-login-v95.md` Packet 2, already IDA-verified for
v95). There is no protocol slot between "client sends its world+channel
choice" and "server replies with the character list" for an intervening
load-percentage message to occupy, and the dispatch switch (Anchor 1) has no
arm that could receive one if the server sent it anyway.

**Conclusion:** `ServerLoad` (writer) is `n-a` on gms_v95, proven positively
(not by a failed name search — by exhausting `CLogin`'s clientbound dispatch
switch and cross-checking the one structurally adjacent sibling that *is*
present). Recorded in `docs/packets/audits/gms_v95/_unimplemented.json` as a
`{packet, reason}` entry with `packet: "login/clientbound/ServerLoad"`
(login candidates in `tools/packet-audit/cmd/run.go` always carry an empty
`pkg`, so `qualifiedWriterName("", "ServerLoad")` returns the bare struct
name `ServerLoad` rather than a `Login`-prefixed name; corroborated by
sibling gms_v95 reports for other bare-pkg login candidates —
`ServerStatus.json`, `AfterLogin.json`, `ServerIP.json` — which all carry
`WriterName` equal to the unprefixed struct name) and a `reason` summarizing
the three anchors above with their addresses.

---

## IDA session record

- `database ecc757f4` = `E:\Programs\Nexon\IDBs_v9\GMS\v95_0\GMS_v95.0_U_DEVM.exe.i64` (confirmed via `idb_list`).
- `database a977912e` = `E:\Programs\Nexon\IDBs_v9\JMS\v185\MapleStory_dump_SCY.exe.i64` (confirmed via `idb_list`; used only for Anchor 1 in §1).
- Queries run: `func_query name_regex="WorldSelect"` (ecc757f4), `func_query name_regex="Security"` (ecc757f4), `func_query name_regex="ServerLoad|WorldLoad|LoadGauge"` (ecc757f4, zero hits), `func_query name_regex="SecurityHandle|CreateSecurity"` (a977912e, zero hits), `func_query name_regex="On.*@CLogin@@"` (ecc757f4), `decompile 0x5df940` (`CLogin::OnPacket`), `decompile 0x5d2250` (`CLogin::OnCheckUserLimitResult`), `decompile 0x5daf20` (`CLogin::GotoWorldSelect`), `xrefs_to 0x606870` (`CUIWorldSelect::UserLimitResult`).
