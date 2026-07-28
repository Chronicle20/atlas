# v48 Stage E — Batch 3 (guild, messenger) — report

Anchor v61, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch `task-113-gms-legacy-versions`.

## Summary

- **Guild family: COMPLETE.** All 15 in-scope guild serverbound sub-structs verified
  (9 GUILD_OPERATION arms + 6 GUILD_BBS arms), and the **GUILD_OPERATION serverbound
  op-row promoted ❌/🟡 → ✅**.
- **Messenger family (12 cells): NOT completed** — investigated, dispatcher partly
  mapped; documented below as remaining work.
- gms_v48 verified count 25 → **40** (+15). No other version dropped
  (v61 208 / v72 216 / v79 228 / v83 367 / v84 345 / v87 379 / v95 399 / jms 362).
- `matrix --check` exit 0; problem-grep 0; v48 conflicts stay 0. 4 commits.

## Per-cell outcome — guild (all ✅)

Discriminator subop verified from the v48 send body; body order matches the verified
v83 codec (fast-path "= v83"), codecs are version-agnostic (no gate change needed).

### GUILD_OPERATION serverbound (opcode 96 / 0x60), fast-path partial arms
| cell | v48 send-site | subop | body |
|---|---|---|---|
| GuildRequestCreate | CField::InputGuildName @0x4c5965 | 2 | EncodeStr(name) |
| GuildAgreementResponse | CField::SendCreateGuildAgreeMsg @0x4c5a18 | 0x1E | Encode4(unk)+Encode1(agreed) |
| GuildInviteRequest | CField::SendInviteGuildMsg @0x4c5a89 | 5 | EncodeStr(target) |
| GuildKick | CField::SendKickGuildMsg @0x4c5e06 | 8 | Encode4(cid)+EncodeStr(name) |
| GuildSetEmblem | CField::SendSetGuildMarkMsg @0x4c635c | 0xF | Encode2+Encode1+Encode2+Encode1 |

GuildAgreementResponse: report ✅ but the **op-row absorbs its report** for v48 because
the v48 registry GUILD_OPERATION-sb *primary* fname is `CField::SendCreateGuildAgreeMsg`
(v61/v83 use the `CUIFadeYesNo::OnButtonClicked` dispatcher fname, which does **not
exist in v48** — v48 uses `CUtilDlg::YesNo` blocking modals). So the AgreementResponse
verification promotes the **op-row to ✅** and the sub-struct row renders folded
(`incomplete`). This is the same matrix-fold class the brief flags for the BBS op-row —
documented, not a tool fix.

### GUILD_OPERATION serverbound — unnamed send helpers (found by body-search, spliced into export)
| cell | v48 send-site (sub) | subop | body |
|---|---|---|---|
| GuildSetNotice | sub_4C63D8 @0x4c63d8 | 0x10 | EncodeStr(notice) |
| GuildWithdraw | sub_4C5CC4 @0x4c5cc4 (YesNo==6) | 7 | Encode4(cid)+EncodeStr(name) |
| GuildSetTitleNames | sub_4C624A @0x4c624a | 0xD | 5×EncodeStr(title) |
| GuildSetMemberTitle | sub_4C61E4 @0x4c61e4 | 0xE | Encode4(targetId)+Encode1(newTitle) |

These four CField guild-send methods are **unnamed in the v48 IDB** (the harvester's
demangled-name lookup left the export entries `unresolved`). Located by scanning the
COutPacket(96) send cluster around the named guild sends; each body-verified against
the v83 codec and its export entry surgically spliced (single-entry, no full re-export).

### GUILD_BBS serverbound — **BBS_OPERATION opcode discovered = 109 (0x6D)**
All six are unnamed subs in the CUIGuildBBS region (0x605000–0x60B000), body-verified.
The v48 registry had **no BBS_OPERATION serverbound entry**; the opcode was recovered
directly from the send bodies.

| cell | v48 send-site (sub) | subop | body |
|---|---|---|---|
| GuildBBSCreateOrEditThread | CUIGuildBBS::OnRegister sub_608D55 @0x608d55 | 0 | Encode1(modify)+[Encode4(threadId)]+Encode1(notice)+EncodeStr(title)+EncodeStr(message)+Encode4(emoticonId) |
| GuildBBSDeleteThread | CUIGuildBBS::OnDelete sub_608F31 @0x608f31 | 1 | Encode4(threadId) |
| GuildBBSListThreads | CUIGuildBBS::SendLoadListRequest sub_6091B1 @0x6091b1 | 2 | Encode4(startIndex) |
| GuildBBSDisplayThread | CUIGuildBBS::SendViewEntryRequest sub_609211 @0x609211 | 3 | Encode4(threadId) |
| GuildBBSReplyThread | CUIGuildBBS::OnComment sub_608FE6 @0x608fe6 | 4 | Encode4(threadId)+EncodeStr(message) |
| GuildBBSDeleteReply | CUIGuildBBS::OnCommentDelete sub_6090F4 @0x6090f4 | 5 | Encode4(threadId)+Encode4(replyId) |

`GuildBBSCreateOrEditThread` report is Verdict 🔍 (3) — the static differ mis-aligns
on the conditional `threadId` field. This is a **pre-existing differ limitation, not a
v48 regression: the verified v83 report is identically Verdict 3.** The cell verifies via
marker + fresh evidence (the byte-fixture test is authoritative over the static diff),
exactly as v83 does.

## Guild mode tables re-derived (from the v48 switch, not copied from v61)
- **GUILD_OPERATION cb** mode table was carried-UNVERIFIED. Not in this batch's cell
  set (no guild clientbound sub-struct is ❌/🟡 for v48). No change made.
- **GUILD_OPERATION serverbound** subops derived from each send body: 2=REQUEST_CREATE,
  5=INVITE, 7=WITHDRAW, 8=KICK, 0xD=SET_TITLE_NAMES, 0xE=SET_MEMBER_TITLE,
  0xF=SET_MARK, 0x10=SET_NOTICE, 0x1E=AGREE. Matches v83.
- **BBS_OPERATION serverbound** (opcode 109): 0=REGISTER, 1=DELETE, 2=LIST, 3=DISPLAY,
  4=REPLY(comment), 5=DELETE_REPLY(comment-delete). Matches v83.

## BBS op-row jms-fold residual
The `BBS_OPERATION` op-row remains `n-a` for v48 (no registry opcode entry added). Per
the brief this op-row is stuck on the jms `lookupAnyVersion` matrix-fold (tooling
limitation / +98-row restructure = separate task). The six **sub-struct** cells (the
actual in-scope cells) are all verified independently via report + marker + evidence,
which does not require the registry op entry. Left as the known tooling residual.

## Messenger family — NOT completed (remaining work, honest status)
The messenger clientbound opcode is **unregistered in v48**, and every CUIMessenger
function is an **unnamed sub**. Serverbound send region located at 0x61A701–0x61E25E,
opcode **92** (registry, already body-verified for the decline arm). Modes mapped so far
from the v48 send bodies:

| atlas codec (fname) | v48 send (sub) | mode | body |
|---|---|---|---|
| OperationAnswerInvite (CUIMessenger::OnCreate) | sub_61A701 | 0 | Encode4(messengerId) |
| Operation (CUIMessenger::OnDestroy) | sub_61AC75 | 2 | (mode only) |
| OperationDeclineInvite (CFadeWnd::SendCloseMessage) | sub_6D3765 | 5 | (v61-verified anchor) |
| OperationChat (CUIMessenger::ProcessChat) | sub_61B27C | 6 | EncodeStr(charName+msg) |
| OperationInvite (CUIMessenger::SendInviteMsg) | **unmapped** | — | EncodeStr(target) |

Not done for messenger, and why it is a larger sub-campaign rather than a quick finish:
- The 8 **clientbound** cells need the full `CUIMessenger::OnPacket` dispatch mode table
  re-derived from the v48 switch (brief: v61 fixed to 8 modes in the 2–10 range). Most
  of these cells have **no v61 anchor** (v61 is itself `incomplete` for Chat/InviteDeclined/
  InviteSent/Join/Remove/RequestInvite/Update) — so they are primary derivations, not
  fast-path mirrors.
- The 4 **serverbound** cells need the `OperationInvite` send located plus the base
  `Operation` (mode-only) special-case validated against the differ, then 4 export
  splices + reports + markers + evidence.

This was left unstarted-to-completion rather than committed half-done; no fabricated
bytes/modes were introduced. Next session: map `CUIMessenger::SendInviteMsg` + the
`CUIMessenger::OnPacket` clientbound dispatcher, then splice/verify per the guild pipeline
proven here.

## Pipeline established (for the remaining messenger + future legacy batches)
1. Body-verify the v48 send/read at port 13337.
2. Splice a single resolved entry into `docs/packets/ida-exports/gms_v48.json`
   (never full re-export).
3. Regenerate reports with `--output docs/packets/audits` (parent dir; the version
   subdir is derived from the template — passing `.../gms_v48` writes a junk nested
   `gms_v48/gms_v48`), then **revert the tool's out-of-scope report drift**
   (AuthSuccess/ChatMulti/ReactorHitRequest/SUMMARY/MonsterCarnival are stale in the
   committed tree relative to current codecs — unrelated to this batch).
4. Add the `packet-audit:verify … version=gms_v48 ida=0x<fn-addr>` marker — the ida
   address **must equal the evidence/function address** or `matrix --check` flags an
   orphan marker.
5. `evidence pin` (derives hash+addr from the spliced export entry).
6. `matrix` + `matrix --check`.

## Commits (branch task-113-gms-legacy-versions)
1. `35a7d5a4e8` guild/Operation partial arms (Kick, Invite, RequestCreate, SetEmblem, AgreementResponse→op-row)
2. `10f4573637` guild/SetNotice (sub_4C63D8)
3. `60fbfe131a` guild Withdraw/SetTitleNames/SetMemberTitle
4. `63d0df8690` guild BBS serverbound family (opcode 109)

## Gates / verification
- No codec version-gates were needed (all guild serverbound codecs are version-agnostic;
  v48 bodies == v83).
- `go test ./libs/atlas-packet/guild/...` green; `go vet` clean.
- `matrix --check` exit 0; problem-grep 0; v48 conflicts 0.
- Regression: v61 208 / v72 216 / v79 228 / v83 367 / v84 345 / v87 379 / v95 399 /
  jms 362 — all held.
- Branch after each commit: `task-113-gms-legacy-versions`.
