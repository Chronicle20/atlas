# USE_INNER_PORTAL / CUserLocal::TryRegisterTeleport — version coverage re-derivation

Task 2 of the task-250 plan. PRD FR-6.2 / Q9.4 forbid inheriting the ⬜
columns by assumption; this file records the derivation for every version
that had no CSV column and no registry entry.

## Step 1 — gms_v12 (confirmed absent)

`docs/packets/MapleStory Ops - ServerBound.csv:237`:

```
USE_INNER_PORTAL,CUserLocal::TryRegisterTeleport,,0x000,101,0x065,104,0x068,112,0x070,113,0x071,177,0x0B1,96,0x060
```

The `GMS v12` column (first Index/Op pair after the `FName` column) holds
`0x000` — the sheet's absent sentinel. **gms_v12 is confirmed absent.**
`services/atlas-configurations/seed-data/templates/template_gms_12_1.json`
gets no route for this op.

## Step 2 — v48 / v61 / v72 / v79 (re-derived from IDB — NOT absent)

The plan's `⬜` premise for these four versions was **"no CSV column and no
registry entry — absence is currently inferred, not derived."** Direct IDB
derivation below shows the inference was wrong: **all four versions are
present**, each with a live `TryRegisterTeleport` send site that constructs a
`COutPacket` and calls `CClientSocket::SendPacket`. This is a **blocking
finding** — see "Blocking finding" below.

### gms_v48 — session `12a398ce` (`GMS_v48_1_DEVM.exe.i64`)

- `func_query "*TryRegisterTeleport*"` → one hit:
  `?TryRegisterTeleport@CUserLocal@@IAEHPBUSKILLENTRY@@JPBD1H@Z` @ `0x6a5462`
  (size `0x4e4`).
- Decompile of `0x6a5462` shows a live send site gated by `if ( a4 )`
  (the `sPortalName` argument):
  ```c
  COutPacket::COutPacket((COutPacket *)v47, 80);
  v57 = 0;
  ZXString<char>::ReleaseBuffer(a4, -1);
  COutPacket::EncodeStr(0);
  v14 = (this->GetPos)(...);          // pos.x
  COutPacket::Encode2((COutPacket *)v47, *v14);
  v15 = (this->GetPos)(...);          // pos.y
  COutPacket::Encode2((COutPacket *)v47, *(_WORD *)(v15 + 4));
  COutPacket::Encode2((COutPacket *)v47, *(_WORD *)(v11 + 12));   // target x
  COutPacket::Encode2((COutPacket *)v47, *(_WORD *)(v11 + 16));   // target y
  CClientSocket::SendPacket((CClientSocket *)g_pClientSocketInstance, (const struct COutPacket *)v47);
  ```
  **Opcode `80` (0x050).**
- **Field-shape divergence found:** this send site has **no
  `COutPacket::Encode1` call at all** (confirmed absent from the function's
  `refs` list — only `EncodeStr`/`Encode2` appear). v48 emits **5 fields**:
  `portalName, x, y, targetX, targetY` — **no leading `fieldKey` byte**,
  unlike v61/v72/v79/v95 (6 fields). This is a genuine per-version field-shape
  gate, not a placeholder.
- Registry cross-check: `docs/packets/registry/gms_v48.yaml` has no
  serverbound opcode 80 entry — opcode 80 is free (clientbound opcode 80 is
  `MULTICHAT`, a disjoint numbering space). The gap is a real cataloging
  omission, not a collision.
- **Outcome: PRESENT.** Export `0x6a5462`, opcode `80` (0x050), 5-field
  layout (no `fieldKey`).

### gms_v61 — session `921fdbb5` (`GMS_v61.1_U_DEVM.exe.i64`)

- `func_query` → `?TryRegisterTeleport@CUserLocal@@IAEHPBUSKILLENTRY@@JPBD1H@Z`
  @ `0x7aa1e3` (size `0x520`).
- Decompile shows a live send site gated by `if ( Src )`:
  ```c
  COutPacket::COutPacket((COutPacket *)v49, 93);
  v59 = 0;
  field = get_field();
  COutPacket::Encode1((COutPacket *)v49, *((_BYTE *)field + 248));   // fieldKey
  ...
  COutPacket::EncodeStr((COutPacket *)v49, v47);                      // portalName
  COutPacket::Encode2((COutPacket *)v49, *v15);                       // x
  COutPacket::Encode2((COutPacket *)v49, *(_WORD *)(v16 + 4));        // y
  COutPacket::Encode2((COutPacket *)v49, *(_WORD *)(v11 + 12));       // targetX
  COutPacket::Encode2((COutPacket *)v49, *(_WORD *)(v11 + 16));       // targetY
  CClientSocket::SendPacket(...)
  ```
  **Opcode `93` (0x05D).** 6-field layout — same shape as gms_v95
  (`fieldKey, portalName, x, y, targetX, targetY`).
- **Outcome: PRESENT.** Export `0x7aa1e3`, opcode `93` (0x05D), 6-field
  layout (matches v95 shape).

### gms_v72 — session `99e435d8` (`GMS_v72.1_U_DEVM.exe.i64`)

- `func_query` → `?TryRegisterTeleport@CUserLocal@@IAEHPBUSKILLENTRY@@JPBD1H@Z`
  @ `0x864562` (size `0x5ca`).
- Decompile shows a live send site gated by `if ( Src )`:
  ```c
  COutPacket::COutPacket((COutPacket *)v59, 100);
  v70 = 0;
  field = get_field();
  COutPacket::Encode1((COutPacket *)v59, *((_BYTE *)field + 276));   // fieldKey
  ...
  COutPacket::EncodeStr((COutPacket *)v59, v55);                      // portalName
  COutPacket::Encode2((COutPacket *)v59, *v15);                       // x
  COutPacket::Encode2((COutPacket *)v59, *(_WORD *)(v16 + 4));        // y
  COutPacket::Encode2((COutPacket *)v59, *(_WORD *)(v11 + 12));       // targetX
  COutPacket::Encode2((COutPacket *)v59, *(_WORD *)(v11 + 16));       // targetY
  CClientSocket::SendPacket(...)
  ```
  **Opcode `100` (0x064).** 6-field layout — same shape as gms_v95.
- **Outcome: PRESENT.** Export `0x864562`, opcode `100` (0x064), 6-field
  layout (matches v95 shape).

### gms_v79 — session `5a1cd4f3` (`GMS_v79_1_DEVM.exe.i64`)

- `func_query` → `?TryRegisterTeleport@CUserLocal@@IAEHPBUSKILLENTRY@@JPBD1H@Z`
  @ `0x8afc42` (size `0x5cf`).
- Decompile shows a live send site gated by `if ( Src )`:
  ```c
  COutPacket::COutPacket((COutPacket *)v58, 99);
  v69 = 0;
  field = get_field();
  COutPacket::Encode1((COutPacket *)v58, *((_BYTE *)field + 276));   // fieldKey
  ...
  COutPacket::EncodeStr((COutPacket *)v58, v55[0]);                   // portalName
  COutPacket::Encode2((COutPacket *)v58, *v15);                       // x
  COutPacket::Encode2((COutPacket *)v58, *(_WORD *)(v16 + 4));        // y
  COutPacket::Encode2((COutPacket *)v58, *(_WORD *)(v11 + 12));       // targetX
  COutPacket::Encode2((COutPacket *)v58, *(_WORD *)(v11 + 16));       // targetY
  CClientSocket::SendPacket(...)
  ```
  **Opcode `99` (0x063).** 6-field layout — same shape as gms_v95.
- **Outcome: PRESENT.** Export `0x8afc42`, opcode `99` (0x063), 6-field
  layout (matches v95 shape).

## Ruling applied (Task 2b, 2026-08-21)

The blocking finding below was resolved by the user during `/execute-task`:
**all ten versions are in scope** (`docs/tasks/task-250-inner-portal-registration/scope-amendment.md`).
Only gms_v12 remains confirmed absent. Task 2b re-confirmed the v48/v61/v72/v79
send sites directly against their IDBs (not merely carrying forward this
file's findings) and produced full structure docs at `structures/gms_v48.md`,
`structures/gms_v61.md`, `structures/gms_v72.md`, `structures/gms_v79.md`.

The v48 field-shape divergence recorded below (5 fields, no `fieldKey`) holds
under re-derivation. The gate boundary is now settled: `fieldKey` is present
from gms_v61 upward and absent only on gms_v48 — `MajorAtLeast(61)`. See
`structures/gms_v61.md` "Gate decision" for the two-sided confirmation
(gms_v48 read as absent, gms_v61 read as present, both in the same pass).

`coverage-manifest.yaml` now lists all ten versions and its `fields:` line
records this gate. The "blocking finding" framing below is preserved as the
historical record of Task 2's original derivation; it is superseded by this
ruling, not deleted.

## Blocking finding (historical — resolved, see ruling above)

**All four of v48, v61, v72, v79 are present, not absent.** Per the task
brief: *"If any version is present, it joins the Task 1 / Task 10 / Task 12
in-scope list in full — registry entry, template route, codec coverage,
fixture — and this plan must be amended before those tasks run."*

This is reported here as a blocking finding and **not** actioned by this
task — no registry entries, template routes, codec version-gates, or
fixtures for v48/v61/v72/v79 are added by Task 2. Consequences for the rest
of the task-250 plan (as originally scoped by Task 2; superseded by the
ruling above, which brings all four into full scope for Task 2b onward):

- **Task 1** (`structures/gms_v95.md`, coverage-manifest scope) needs its
  version list widened from six to ten versions
  (`gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92,
  gms_v95, jms_v185`).
- **v48 additionally needs a real `MajorAtLeast`-style field-shape gate**:
  it is the only version in the whole family that omits the `fieldKey` byte
  (5 fields, not 6). The "no gate is required" ruling in
  `structures/gms_v95.md` (from Task 1) was correct for the six versions it
  covered, but is no longer true once v48 is in scope.
- **Task 10** (registry/template routing) needs four more registry entries
  (opcodes 80/93/100/99 for v48/v61/v72/v79 respectively) and four more
  template routes.
- **Task 12** (fixtures/verification) needs four more byte-fixture cells,
  including the v48 5-field shape as a distinct fixture from the 6-field
  shape used everywhere else.

This file records the finding per the brief's explicit instruction to stop
and report rather than amend the plan unilaterally. The six-version scope
already declared in `coverage-manifest.yaml` (Task 1) is left unchanged by
Task 2; only this document adds the v48/v61/v72/v79 findings, kept separate
from the manifest as instructed ("Record the gms_v12 / v48 / v61 / v72 / v79
absences in `version-coverage.md`, not in the manifest").

## Summary table

| version | session | outcome | export | opcode | field count | shape |
|---|---|---|---|---|---|---|
| gms_v12 | (CSV only) | confirmed absent | — | `0x000` sentinel | — | — |
| gms_v48 | `12a398ce` | **PRESENT** | `0x6a5462` | `80` (0x050) | 5 | no `fieldKey` — diverges |
| gms_v61 | `921fdbb5` | **PRESENT** | `0x7aa1e3` | `93` (0x05D) | 6 | matches v95 |
| gms_v72 | `99e435d8` | **PRESENT** | `0x864562` | `100` (0x064) | 6 | matches v95 |
| gms_v79 | `5a1cd4f3` | **PRESENT** | `0x8afc42` | `99` (0x063) | 6 | matches v95 |
| gms_v83 | (Task 1 scope) | present | — | `101` (0x065, CSV) | 6 | matches v95 |
| gms_v84 | (Task 1 scope) | present | — | (per registry) | 6 | matches v95 |
| gms_v87 | (Task 1 scope) | present | — | `104` (0x068, CSV) | 6 | matches v95 |
| gms_v92 | (Task 1 scope) | present | — | `112` (0x070, CSV) | 6 | matches v95 |
| gms_v95 | `ecc757f4` | present | `0x913690` | `113` (0x071) | 6 | reference shape |
| jms_v185 | (Task 1 scope) | present | — | `96` (0x060, CSV) | 6 | matches v95 |
