# Spike Verdict — Mist / AffectedAreaCreated nType source + four-version read-order (task-080 B1.1b)

## Four-version read-order (from `docs/packets/ida-exports/*.json` + JMS185 live decompile)

`CAffectedAreaPool::OnAffectedAreaCreated` — CLIENT decode of the affected-area-created (mist) packet:

| field | width | v83 | v87 | v95 | JMS185 |
|---|---|---|---|---|---|
| dwId | Decode4 | ✓ | ✓ | ✓ | ✓ |
| nType | Decode4 | ✓ | ✓ | ✓ | ✓ |
| dwOwnerId | Decode4 | ✓ | ✓ | ✓ | ✓ |
| nSkillID | Decode4 | ✓ | ✓ | ✓ | ✓ |
| nSLV | Decode1 | ✓ | ✓ | ✓ | ✓ |
| phase | Decode2 | ✓ | ✓ | ✓ | ✓ |
| rcArea RECT | DecodeBuf 16 (4×int32 abs LT.x,LT.y,RB.x,RB.y) | ✓ | ✓ | ✓ | ✓ |
| **tStart** | Decode4 | — | — | **✓ (v95-only)** | — |
| tEnd | Decode4 | ✓ | ✓ | ✓ | ✓ |

- Non-v95 (v83/v87/JMS185): **8 fields = 39 bytes** (`4+4+4+4+1+2+16+4`).
- v95: **9 fields = 43 bytes** — adds `tStart(int4)` **between rcArea and tEnd**, gated `Region()=="GMS" && MajorVersion()>=95`.
- Addresses: v83 `0x431a63`, v87 `0x432f3f`, v95 `0x437ec0`, JMS185 `0x436572`.
- rcArea is an **absolute** RECT (origin + offset), read as one 16-byte buffer = 4×int32.

## nType source verdict (the B1.1b question)

**`nType` is the affected-area KIND discriminator, NOT the mist-vs-other rendering selector.**
- Client branches: `nType == 3` → the **item-area-buff** branch (a consumable/item-driven area). The mist-vs-other VISUAL is driven by **`nSkillID`** (a large `switch(nSkillID)` on 130/131/2111003/4221006/… picks the animation), not by nType.
- atlas-maps is the ONLY producer of these events, and it ONLY creates **skill/disease-driven mist** — `mist/processor.go::Create` builds from a `CreateCommandBody` that carries `Disease`/`DiseaseValue` + `SourceSkillId`/`SourceSkillLevel` and has **no type field**. atlas-maps never creates an item-area-buff (nType==3).
- Therefore the correct wire `nType` for every atlas-maps mist is a non-3 value; **`Type=0`** (the default added in B1.1a, never overridden in the create path) is correct.

**Decision: no create-path change. `Type` stays 0 (B1.1a default) for skill/disease mist. The `CreatedBody.Type` plumbing exists for completeness/future item-area-buff support but is correctly 0 today.** Recorded here rather than wiring a constant, since 0 is the right value and inventing a non-zero constant would be unjustified.

## Downstream
- B1.1c rewrites `AffectedAreaCreated` to this exact layout (abs RECT + v95 tStart gate).
- B1.1d wires the channel consumer to pass skill id/level/type through.
