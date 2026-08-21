# Scope amendment — task-250 covers ten versions, not six

Status: **ACTIVE — supersedes the version lists in `plan.md`, `design.md` and `prd.md`.**
Ruled: 2026-08-21, by the user, during `/execute-task` after Task 2.

Every remaining plan task must read this file. Where it disagrees with
`plan.md`, this file wins.

---

## What changed

`plan.md` scoped six versions (gms_v83, gms_v84, gms_v87, gms_v92, gms_v95,
jms_v185) and treated gms_v48 / v61 / v72 / v79 as **absent**, inferring that
from "no CSV column and no registry entry."

Task 2 derived those four directly against their IDBs. **All four are
present** — each has a live `CUserLocal::TryRegisterTeleport` send site:

| version | export | opcode | fields |
|---|---|---|---|
| gms_v48 | `0x6a5462` | 80 | **5 — no `fieldKey`** |
| gms_v61 | `0x7aa1e3` | 93 | 6 |
| gms_v72 | `0x864562` | 100 | 6 |
| gms_v79 | `0x8afc42` | 99 | 6 |

All four already carry a `docs/packets/registry/gms_vNN.yaml` and a
`services/atlas-configurations/seed-data/templates/template_gms_NN_1.json`.
They are first-class supported versions; only this op was undeclared.

The gms_v48 shape was re-derived independently by the controller and holds —
the send site encodes `EncodeStr` (portalName) then four `Encode2`s, with **no
`Encode1` ahead of the string**:

```c
COutPacket::COutPacket((COutPacket *)v47, 80);
ZXString<char>::ReleaseBuffer(a4, -1);
COutPacket::EncodeStr(0);                                     // portalName
COutPacket::Encode2((COutPacket *)v47, *v14);                 // x
COutPacket::Encode2((COutPacket *)v47, *(_WORD *)(v15 + 4));  // y
COutPacket::Encode2((COutPacket *)v47, *(_WORD *)(v11 + 12)); // targetX
COutPacket::Encode2((COutPacket *)v47, *(_WORD *)(v11 + 16)); // targetY
```

## The ruling

**In scope: all ten versions** — gms_v48, gms_v61, gms_v72, gms_v79, gms_v83,
gms_v84, gms_v87, gms_v92, gms_v95, jms_v185.

**Out of scope: gms_v12 only** — confirmed absent, CSV `0x000` sentinel.
`template_gms_12_1.json` gets no route.

## Consequences per plan task

- **Task 1** — its "no `MajorAtLeast` gate is required" ruling is
  **superseded**. Four more structure docs are needed. Handled by the
  follow-on derivation pass (Task 2b) rather than by re-running Task 1.
- **Task 3 (codec)** — `fieldKey` is **version-gated**, present from the
  v48/v61 boundary upward. Use the `MajorAtLeast` idiom; a raw `> N`
  comparison is not acceptable (PRD FR-2.3). The exact boundary constant is
  whatever Task 2b's confirmation of gms_v61 establishes — take it from
  `structures/gms_v61.md`, do not assume it.
  The other five fields (`portalName`, `x`, `y`, `targetX`, `targetY`) are
  ungated: identical across all ten versions.
- **Task 10 (seed templates)** — routes **ten** templates, not six. Adds
  `template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_72_1.json`,
  `template_gms_79_1.json`. `template_gms_12_1.json` still gets no route.
- **Task 11 (packet-audit linkage)** — the `packet:` declaration is needed in
  ten registry files. The four new ones also need the op itself declared
  (opcode + `fname: CUserLocal::TryRegisterTeleport`); it is currently absent
  from `gms_v48.yaml`, `gms_v61.yaml`, `gms_v72.yaml`, `gms_v79.yaml`.
- **Task 12 (evidence, markers, matrix)** — ten cells, not six. gms_v48 needs
  its **own** fixture asserting the five-field body; it cannot share the
  six-field fixture.
- **`coverage-manifest.yaml`** — must list all ten versions, and its `fields:`
  line must record the gate rather than "no gate required."

## Unchanged by this amendment

- `maxPortalEntryDistance = 81` (Task 2, `structures/gms_v95.md`
  §Threshold derivation). Task 8 copies it verbatim.
- Tasks 4, 5, 6, 7 are version-agnostic and are unaffected.
- The packet id remains `portal/serverbound/PortalInnerPortal`.
