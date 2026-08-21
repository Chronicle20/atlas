# gms_v48 — `USE_INNER_PORTAL` / `CUserLocal::TryRegisterTeleport`

Session: `12a398ce` (`GMS_v48_1_DEVM.exe.i64`)

## USE_INNER_PORTAL

- Export address of `CUserLocal::TryRegisterTeleport`: **`0x6a5462`**
  (`?TryRegisterTeleport@CUserLocal@@IAEHPBUSKILLENTRY@@JPBD1H@Z`, already
  named in the IDB — `func_query "*TryRegisterTeleport*"` returns this
  address directly; no rename required for this version).
- `COutPacket` constructor: `COutPacket::COutPacket((COutPacket *)v47, 80);`
  at **`0x6a557a`** — opcode constant **`80`** (`0x050`).
- Registry cross-check: `docs/packets/registry/gms_v48.yaml` — **grepped for
  `TryRegisterTeleport` / `USE_INNER_PORTAL`, no hit.** The registry does not
  yet declare this op; Task 11 adds it with opcode `80`.

### Ordered field table

Re-confirmed directly against the IDB (decompile of `0x6a5462`, session
`12a398ce`):

| # | field | width | client expression |
|---|---|---|---|
| 1 | `portalName` | ASCII string (u16 len + bytes) | `COutPacket::EncodeStr(0)` — preceded by `ZXString<char>::ReleaseBuffer(a4, -1)`, `a4` = `sPortalName` |
| 2 | `x` | `int16` | `COutPacket::Encode2((COutPacket *)v47, *v14)` — `v14 = this->GetPos()` |
| 3 | `y` | `int16` | `COutPacket::Encode2((COutPacket *)v47, *(_WORD *)(v15 + 4))` — `v15 = this->GetPos()` |
| 4 | `targetX` | `int16` | `COutPacket::Encode2((COutPacket *)v47, *(_WORD *)(v11 + 12))` — `v11` is the target `PORTAL*` from `sub_596D67(a5)` (`a5` = `sTargetPortalName`), offset `+12` = `ptPos.x` |
| 5 | `targetY` | `int16` | `COutPacket::Encode2((COutPacket *)v47, *(_WORD *)(v11 + 16))` — same `PORTAL*`, offset `+16` = `ptPos.y` |

**No `fieldKey` byte.** The function's `refs` list contains `EncodeStr` and
three `Encode2` entries but **no `Encode1`** anywhere in the function — this
was independently re-verified against the live decompile in this pass, not
merely carried forward from `version-coverage.md`. gms_v48 is a genuine
5-field body, distinct from the 6-field shape used from gms_v61 upward.

Send site is gated by `if (a4)` (`sPortalName != NULL`) inside the branch
that follows a successful `CWvsPhysicalSpace2D::GetFootholdUnderneath` probe
under the target portal — same overall shape as the later versions, just
without the `fieldKey` encode.

### Caller

Not re-walked in this pass (the field layout was re-confirmed by decompiling
the send site directly, which is sufficient for the codec's field order —
Task 2's `version-coverage.md` already established the send site is live and
reachable). The same two-caller shape (`CheckPortal_Collision` +
`HandleUpKeyDown`-equivalent) is expected but not independently re-derived
here.

### Per-version delta

**Diverges from gms_v61 and every later version:** no `fieldKey` field. See
the gate decision below.

## Gate decision

Confirmed directly against the IDB in this pass (not carried forward
unverified): gms_v48's send site (`0x6a5462`) has **zero** `Encode1` calls —
the field sequence is `EncodeStr, Encode2, Encode2, Encode2, Encode2` (5
fields). gms_v61's send site (`0x7aa1e3`, see `gms_v61.md`) opens with
`COutPacket::Encode1((COutPacket *)v49, *((_BYTE *)field + 248))` — a
`fieldKey` byte — before its `EncodeStr` call. gms_v72 and gms_v79 both carry
the same `Encode1` ahead of `EncodeStr` (see their structure docs). gms_v83
through jms_v185 all carry it too (Task 1, `gms_v95.md`).

**The boundary is between gms_v48 and gms_v61 — no version in between is in
scope, so the gate is exact:** `fieldKey` is present for every in-scope
version at or above gms_v61, absent only for gms_v48.

**Gate constant: `MajorAtLeast(61)`.** In `Encode`, emit `fieldKey` only when
`tenant.MustFromContext(ctx).MajorAtLeast(61)` is true; gms_v48 (major 48)
takes the no-`fieldKey` path, every other in-scope version (61, 72, 79, 83,
84, 87, 92, 95, and jms_v185 by region) takes the `fieldKey` path. The other
five fields (`portalName`, `x`, `y`, `targetX`, `targetY`) are **ungated** —
identical field order, widths, and semantics across all ten versions.
