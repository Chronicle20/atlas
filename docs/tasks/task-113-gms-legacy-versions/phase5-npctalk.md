# task-113 Phase 5 — NPC_TALK serverbound oid-only vs oid+x+y (v72/v79)

## Question

During the v48/v61 pass, NPC_TALK serverbound (client → server "I talked to
NPC") was confirmed to carry **oid + x + y** in v48 (`sub_568A2A@0x568A2A`,
`COutPacket(46)`) and v61 (`COutPacket(54)`). The v72 pass fixtured it as
**oid-only**; v79 as oid+x+y. The ledger flagged v72 as a possible false-pass.
This phase verifies the truth for both from their IDBs.

## Verdict: FALSE-PASS FIXED (v72) · CORRECT-AS-IS (v79)

- **v72 sends oid + x + y** — the shipped oid-only fixture was a false pass.
- **v79 sends oid + x + y** — the existing fixture was already correct.

## v72 send-site decompile

Registry `docs/packets/registry/gms_v72.yaml` already lists NPC_TALK serverbound
as **opcode 57, `CUserLocal::TalkToNpc`, address 6552977 = `0x63FD91`**
(`ida-discovered`). Decompiling that function (GMS_v72.1_U_DEVM.exe, port 13339)
shows three `COutPacket(57)` send-sites, each with the identical body:

```
COutPacket::COutPacket(pkt, 57)                       // 0x63fe09 / 0x64066f / 0x640857
COutPacket::Encode4(pkt, npcOid = this->obj @+152)    // Encode4 oid
v7 = (*(dword_AA3AB8+4))->vtbl[16]()                  // position accessor -> {x@0, y@4}
COutPacket::Encode2(pkt, *v7)                          // Encode2 userX
v8 = (*(dword_AA3AB8+4))->vtbl[16]()
COutPacket::Encode2(pkt, *(v8+4))                      // Encode2 userY
CClientSocket::SendPacket(g_pClientSocketInstance, pkt)
```

So v72 NPC_TALK serverbound = `Encode4(oid) + Encode2(x) + Encode2(y)` — the
same shape as v48/v61/v79/v83+.

**Origin of the false pass.** The prior fixture and evidence cited
`ida=0x70dd49`. Decompiling `0x70dd49` in v72 yields
`CUICharacterSaleDlg::OnCreate` (a 77 KB character-sale dialog), NOT an NPC_TALK
sender. `0x70dd49` is the **v48** address of `sub_70DD49` (the unrelated
ItemCancel / `CANCEL_ITEM_EFFECT` sender, `COutPacket(57)+Encode4(sourceId)`
ONLY); its oid-only shape and address were mis-copied onto v72's TalkToNpc. The
v72 IDA export note claiming a "`sub_70DD49@0x70dd49` … `Encode4(oid)` ONLY …
called from `sub_69FE41`" TalkToNpc was likewise fabricated — `0x69FE41` in v72
is an unrelated small allocator (`sub_69FE19`).

## v79 send-site decompile

`CUserLocal::TalkToNpc @ 0x8B7E10` (GMS_v79_1_DEVM.exe, port 13340):

```
COutPacket::COutPacket(pkt, 56)                        // 0x8b7e51
COutPacket::Encode4(pkt, oid = *(a3+164))              // Encode4 oid
v7 = this[1]->vtbl[16]()                               // position accessor
COutPacket::Encode2(pkt, *v7)                          // Encode2 userX
COutPacket::Encode2(pkt, *(v8+4))                      // Encode2 userY
CClientSocket::SendPacket(...)
```

oid + x + y — matches the existing v79 fixture. No change.

## Codec change

`libs/atlas-packet/npc/serverbound/start_conversation.go` — the shared
`StartConversation` codec is version-gated by `startConversationHasXY(t)`.
v72 was excluded; the gate is now:

```go
return t.MajorAtLeast(72) || t.MajorVersion() == 61 || t.MajorVersion() == 48
```

(previously `MajorAtLeast(79) || ==61 || ==48`). Non-GMS regions already
returned true. v83/84/87/95/jms remain oid+x+y (unchanged). Pre-v48 GMS with no
IDB (the `v28` test variant) stays oid-only. The function comment was rewritten
to document the Phase-5 verification.

## Fixtures updated

- `conversation_v72_test.go` — `TestStartConversationByteV72` now expects
  `34 08 00 00 FB FF C8 00` (oid=2100, x=-5, y=200); marker re-pinned
  `ida=0x70dd49 → ida=0x63fd91`; comment corrected.
- `start_conversation_test.go` — `TestStartConversationRoundTrip` comment
  corrected (removed the false "v72 = sub_70DD49 oid-only" claim; v28 is the
  oid-only variant).

## Evidence / export / audit artifacts re-pinned

- `docs/packets/ida-exports/gms_v72.json` — `CUserLocal::TalkToNpc` entry:
  address `0x70dd49 → 0x63fd91`, calls now `Encode4 oid + Encode2 userX +
  Encode2 userY`, note corrected.
- `docs/packets/evidence/gms_v72/npc.serverbound.NpcStartConversation.yaml` —
  address `→ 0x63fd91`, `decompile_sha256 →
  4a5b29c178a6b2c97654f15b28ce5f37ce307d5596a8072f7f02283a9060a4a3`
  (recomputed `evidence.FunctionHash` over the corrected export entry).
- `docs/packets/audits/gms_v72/NpcStartConversation.{json,md}` — corrected to
  the verified oid+x+y shape (3 rows ✅, `FlatInvalid: false`, address
  `0x63fd91`).
- `docs/packets/audits/gms_v72/SUMMARY.md` — NpcStartConversation ❌ → ✅.
- `docs/packets/audits/STATUS.md`, `status.json` — regenerated (only the
  gms_v72 export hash changed; the grade grid is unchanged).

## Verification

- `go test ./libs/atlas-packet/npc/...` — ok.
- `go test ./libs/atlas-packet/...` (all versions) — ok, no failures.
- `go vet ./libs/atlas-packet/npc/...` — clean.
- `go run ./tools/packet-audit matrix --check` — exit 0.
- Verified counts unchanged (NpcStartConversation was already ✅ via the fixture
  marker; only its wire semantics were wrong). Totals hold: v72 216, v79 228,
  v83 367, v84 345, v87 379, v95 399, jms 362, v61 208, v48 165.
