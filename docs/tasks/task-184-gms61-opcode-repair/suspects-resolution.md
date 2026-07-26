# task-184: gms_61 opcode-repair — IDA resolution of two serverbound suspects

IDB: `GMS_v61.1_U_DEVM.exe.i64`, session `965202bf` (confirmed via `idb_list` before
any lookup). All addresses/lines below are verbatim `decompile` output from that
session. Read-only pass — no template or code files were modified.

---

## Handler 1: `HiredMerchantOperationHandle` (template opcode `0x3E`)

**Verdict: CORRUPT / SPURIOUS — 0x3E is not sent by the gms_61 client for hired-merchant
operations at all. The feature rides the already-bound opcode 0x6F
(`CharacterInteractionHandle`).**

### Evidence

All three entrusted-shop (hired merchant) dialog action handlers construct their
outbound packet with `COutPacket::COutPacket(_, 111)` — **111 decimal = 0x6F**, not
0x3E — followed by an `Encode1` submode byte:

```
?OnGoOut@CEntrustedShopDlg@@IAEXXZ  fn@0x4da528
  COutPacket::COutPacket((COutPacket *)v3, 111); /*0x4da53f*/
  COutPacket::Encode1((COutPacket *)v3, 0x25u); /*0x4da54d*/   // submode 37
  CClientSocket::SendPacket(...)                /*0x4da55c*/

?OnArrange@CEntrustedShopDlg@@IAEXXZ  fn@0x4da583
  COutPacket::COutPacket((COutPacket *)v7, 111); /*0x4da5c3*/
  COutPacket::Encode1((COutPacket *)v7, 0x26u);  /*0x4da5d0*/   // submode 38
  CClientSocket::SendPacket(...)                 /*0x4da5df*/

?OnWithdrawMoney@CEntrustedShopDlg@@IAEXXZ  fn@0x4da5fd
  COutPacket::COutPacket((COutPacket *)v3, 111); /*0x4da61d*/
  COutPacket::Encode1((COutPacket *)v3, 0x29u);  /*0x4da62b*/   // submode 41
  CClientSocket::SendPacket(...)                 /*0x4da63a*/
```

For comparison, the *pre-check* (opening the entrusted-shop-purchase dialog) is a
separate, already-known opcode:

```
?SendEntrustedShopCheckRequest@CWvsContext@@QAEXJJT_LARGE_INTEGER@@@Z  fn@0x848a01
  COutPacket::COutPacket((COutPacket *)v25, 59);  /*0x848b9f*/   // 0x3B
  COutPacket::Encode1((COutPacket *)v25, 0);      /*0x848baf*/
```
(0x3B matches the prior pass's note; it is not bound in the current gms_61
template at all, and is out of this task's scope.)

The sibling shop-family dialogs confirm the same 0x6F mini-room dispatcher pattern
with different submode ranges per feature:

```
?PutItem@CPersonalShopDlg@@QAEHV?$ZRef@UGW_ItemSlotBase@@@@JJ@Z  fn@0x60d81e
  COutPacket::COutPacket((COutPacket *)v28, 111);              /*0x60db63*/
  COutPacket::Encode1((COutPacket *)v28, v19 != 0 ? 32 : 21);  /*0x60db8e*/

?SendPutItemRequest@CTrunkDlg@@IAEXXZ  fn@0x68c7e6
  COutPacket::COutPacket((COutPacket *)v21, 111); /*0x68c971*/
  COutPacket::Encode1((COutPacket *)v21, 0xEu);   /*0x68c97f*/  // submode 14
```

(`CShopDlg::SendBuyRequest` fn@0x646c41, by contrast, uses a wholly different
opcode 57/0x39 for NPC-shop trading — irrelevant here, cited only to show that
"is it 0x6F" is genuinely feature-dependent and was checked, not assumed.)

### Cross-check against the current gms_61 template

`services/atlas-configurations/seed-data/templates/template_gms_61_1.json` already
binds opcode `0x6F` to `CharacterInteractionHandle` with an `operations` table that
includes:

```
"MERCHANT_PUT_ITEM": 31,
"MERCHANT_BUY": 32,
"MERCHANT_REMOVE_ITEM": 36
```

The three submodes found above — 37 (0x25, GoOut/exit), 38 (0x26, Arrange), and 41
(0x29, WithdrawMoney) — slot directly into the gap immediately after the existing
`MERCHANT_*` entries (32→36→37/38→41), i.e. they are the **missing** hired-merchant
sub-operations of the same feature family already wired at 0x6F. This is strong,
consistent (not just single-sample) evidence that hired-merchant operations are a
submode family of `CharacterInteractionHandle`, not a distinct handler/opcode.

### Collision check

The claim "0x3E is spurious" means there is no free "true opcode" to move
`HiredMerchantOperationHandle` to — the correct home (0x6F) is **already occupied**
by `CharacterInteractionHandle`, correctly. A correction would be: remove the
spurious `0x3E` → `HiredMerchantOperationHandle` binding, and (separately, out of
this read-only task's scope) add operations `37`, `38`, `41` to the existing `0x6F`
`operations` table. There is no reassignment-collision risk because no reassignment
is needed — only removal of the bogus binding.

Opcode `0x3E` itself: not found bound to any other client Send in the areas
surveyed (CEntrustedShopDlg, CPersonalShopDlg, CTrunkDlg, CShopDlg, CMiniRoomBaseDlg
send paths). A byte-level sweep of all 391 `COutPacket` constructor call sites for
literal `62` was not exhaustively completed (impractical volume for this pass); the
verdict above rests on the affirmative, cross-checked evidence that the actual
hired-merchant feature rides 0x6F, not on a negative "nothing else uses 0x3E" proof.

---

## Handler 2: `AdminCommand` (template opcode `0x7E`)

**Verdict: CORRUPT / SPURIOUS — the gms_61 client has no dedicated admin/GM-command
send opcode. Unrecognized "/"-prefixed chat text (gated behind a GM-only runtime
flag) rides the already-bound general-chat opcode 0x2E
(`CharacterChatGeneralHandle`).**

### Evidence

`CUIStatusBar::OnKey` fn@0x73be06 is the single dispatch point for chat-box submit
(Enter key). Every user-typed line starting with `/` (ASCII 47) is routed to one
function, with no separate GM/admin branch:

```
v11 = *Src == 47;                                      /*0x73bfc1*/
if ( v11 )                                              /*0x73bfc8*/
  CField::SendChatMsgSlash(field, v27[0]);              /*0x73bfcc*/
```

`CField::SendChatMsgSlash` fn@0x4e7469 (0x10a4 bytes) is a long chain of
string-resource comparisons, each dispatching to its own dedicated, already-known
opcode (whisper 0x4e8635, trade-invite 0x4e87e1, party create/withdraw/join/kick
0x4e898b/0x4e8a90/0x4e8b29/0x4e8cfb, guild create/invite/withdraw/kick
0x4e9260/0x4e92d1/0x4e94cf/0x4e95f7, friend add/delete 0x4e9c03/0x4e9d4c, field
transfer via numeric map-id 0x4e7522, etc.) — none of these ~40 named commands is
"admin"/"GM".

Near the end of the function, one command (resource id 194) is checked against a
runtime capability gate before sending, and that send carries **no string payload
at all** — just a single boolean:

```
v87 = sub_48242E((int)v110, v111[0]);                        /*0x4e8223*/  // matches resource-string 194
if ( v87 )
{
  v88 = v124;
  if ( !v124 || !(*(int (__thiscall **)(int, int *))(*(_DWORD *)(v124 + 4) + 72))(v124 + 4, &dword_975DD0) ) /*0x4e8244*/
    v88 = 0;
  if ( !v88 )
    goto LABEL_198;                                          // silently no-op if the gate fails
  COutPacket::COutPacket((COutPacket *)v112, 179);            /*0x4e825f*/   // 0xB3, bool-only toggle
  v89 = *(_DWORD *)(v88 + 1824) == 0;                         /*0x4e826e*/
  COutPacket::Encode1((COutPacket *)v112, v89);               /*0x4e8276*/
  CClientSocket::SendPacket(...);                             /*0x4e8285*/
}
```
This shape (no text argument, single toggle bool, gated behind a capability check
reading `dword_975DD0`) reads as a GM-only UI toggle (e.g. a "/hide"-style command),
**not** a generic "send arbitrary admin command text" packet — there is no room in
this packet for a command string. Opcode `0xB3` (179) is unbound as a serverbound
handler in the current gms_61 template (only bound as a *clientbound writer*,
`MoveMonsterAck`, at line 2617 — a disjoint numbering space, so no collision). This
opcode is a plausible home for a narrow `AdminHide`-type feature, but is NOT itself
evidence of a generic `AdminCommand`.

The actual generic fallback — reached only when the typed `/xxx` text matches none
of the ~40 named commands nor the resource-194 toggle nor an item-name emote — sends
the **raw text on the general-chat opcode**, gated by a single-bit runtime flag:

```
if ( (TSecType<unsigned char>::GetData(v122 + 8240) & 1) == 0 ) /*0x4e848b*/
  goto LABEL_198;                                              // non-privileged: nothing is sent
COutPacket::COutPacket((COutPacket *)v117, 46);                 /*0x4e8495*/   // 0x2E
sub_414F76(v4);                                                 /*0x4e84a7*/   // copy the raw "/..." text
COutPacket::EncodeStr(v111[0]);                                 /*0x4e84af*/
COutPacket::Encode1((COutPacket *)v117, 0);                     /*0x4e84b8*/
CClientSocket::SendPacket(...);                                 /*0x4e84c7*/
```

Opcode 46 decimal = **0x2E**, which the current gms_61 template already binds to
`CharacterChatGeneralHandle` (line 239). The ordinary chat-send path confirms the
same wire shape:

```
?SendChatMsg@CField@@QAEXABV?$ZXString@D@@H@Z  fn@0x4e73af  (called for plain map chat)
  COutPacket::COutPacket((COutPacket *)v5, 46); /*0x4e7411*/
  COutPacket::EncodeStr(v4);                    /*0x4e742e*/
  COutPacket::Encode1((COutPacket *)v5, a2);    /*0x4e7439*/
```
Identical opcode, identical `EncodeStr` + trailing `Encode1` byte shape. The only
difference in the admin fallback is that the send is skipped entirely unless the
per-character flag byte at `g_pWvsContext`-relative offset `+8240` has bit 0 set —
i.e., this is the code path that lets a privileged (GM) account forward an
unrecognized `/command args` string to the server as an ordinary chat packet, for
the server to parse and act on. `CField::OnAdminResult` fn@0x4ed45f (the clientbound
response handler) supports this reading: it decodes a mode byte selecting among
~20 different result shapes (channel-name lookup, map-string/warp lookup, a
`[name] : msg`-bracketed broadcast format, plain notices, etc.) — i.e. one
clientbound handler multiplexes many different GM-command results, consistent with
the server parsing free-form command text rather than receiving one opcode per
command.

**Caveat on rigor**: I could not resolve the exact semantic name of the
`+8240` bit (no type/RTTI info available for `CWvsContext` in this IDB —
`search_structs` returned empty) — I am reading it from behavior (it gates the only
"send-unmatched-slash-text" branch in the entire function) rather than from a
symbol name. This is circumstantial, not certain confirmation that the bit is
literally "GM level flag," though the behavior it gates is a very strong fit.

### Collision check

As with Handler 1: the "true" opcode for admin-style commands is `0x2E`
(`CharacterChatGeneralHandle`), which is **already correctly bound** in the current
gms_61 template. A correction is a removal of the spurious `0x7E` entry, not a
reassignment — no collision risk. Opcode `0x7E` itself was not found bound to any
client Send in the functions surveyed (`CField`, `CUIStatusBar::OnKey`,
`SendChatMsgSlash`, `OnAdminResult`'s call graph).

---

## Summary

| Handler | Template opcode | Verdict | True home |
|---|---|---|---|
| `HiredMerchantOperationHandle` | `0x3E` | CORRUPT / SPURIOUS | `0x6F` (`CharacterInteractionHandle`, already bound) — needs `operations` 37/38/41 added, no reassignment |
| `AdminCommand` | `0x7E` | CORRUPT / SPURIOUS | `0x2E` (`CharacterChatGeneralHandle`, already bound) — no dedicated admin opcode exists client-side |

No file other than this report was modified.
