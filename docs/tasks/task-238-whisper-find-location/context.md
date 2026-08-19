# task-238 — Whisper `/find` Accurate Target Location — Context

Companion to [plan.md](plan.md). Everything an executor needs that is not a
task step: the decisions already made, the evidence behind them, and the traps.

---

## 1. Decisions taken during `/plan-task`

The design left three items open for the user (design §8). All three are now
closed.

| # | Item | Decision | Effect on the plan |
|---|---|---|---|
| 1 | The PRD reversal (design §0, §3) | **Accepted.** Build the `state` column on `atlas-maps`'s `character_locations`; do not build the PRD's `atlas-cashshop` presence store. | Tasks 2–5. `atlas-cashshop` has no diff. |
| 2 | The v92/v95 buddy-window x/y divergence (design §2.6) | **The divergence does not exist.** Re-derived against the live IDBs; finding withdrawn. | No wire change. The derivation instead retired a blocker, which became Task 9. |
| 3 | `Gm()` semantics (design §2.4) | **Widen repo-wide** to `m.gm > 0`, plus a `GmLevel() int` accessor. | Task 7. |

The user initially chose to fold the item-2 packet change into this branch. That
choice was made on the design's stated evidence; once the evidence was
re-derived and the divergence disappeared, there was nothing to fold. The
re-derivation is what the user then approved carrying, as Task 9.

---

## 2. The item-2 evidence, in full

This is the part most likely to be re-litigated, so the reasoning is recorded
here rather than left in conversation.

**What design v1 claimed.** `docs/packets/ida-exports/gms_v95.json`'s raw
`CField::OnWhisper` entry contains, twice:

```
{"op": "Decode4", "guard": "v3 == 72 && v28 == 1"}
```

Read literally that says: on mode 72 (`0x48`), at findMode 1, the client reads
two more int32s — i.e. the buddy-window arm expects x/y that Atlas does not
send. v1 recorded this as a pre-existing packet gap on `gms_v92` and `gms_v95`.

**Why that reading is wrong.** Three independent signals contradicted it before
any decompile:

1. The **curated per-arm record in the same file**,
   `CField::OnWhisper#FindResultMap`, says `x (mode 0x09 only)`,
   `guard: mode == 0x09`, note *"x/y are Decode4 (int32, IDA-confirmed v83 AND
   v95) when mode 0x09. Wire version-invariant."*
2. `gms_v92.json`'s find arms are all `"unresolved": true` — *"dispatcher-family
   arm not harvested for gms_v92 (task-27 scope)"*. There was **no** v92 arm
   evidence in the repo at all, so the v92 half of the claim rested on nothing.
3. The v92 raw entry still carries a parity test in one of its guards —
   `(v4 & 1) == 0 && (v4 & 0x40) != 0`, which is exactly `0x48` — so parity
   was plainly still being discriminated.

**The derivation.** `CField::OnWhisper` @`0x5448a0` (v95,
`GMS_v95.0_U_DEVM.exe`, IDA session `32c8836f`) and @`0x53e2a0` (v92,
`GMS_v92_1_DEVM.exe`, session `e3328b84`):

```c
case 9:
case 72:
  DecodeStr(name);
  v28 = Decode1();                   // findMode
  v29 = Decode4();                   // payload (mapId / channel / -1 / 0)
  if ( (v3 & 1) == 0 )               // EVEN mode -> 0x48
  {
    if ( (v3 & 0x40) != 0 ) { switch (v28) { case 2: case 3: case 1: } }
    goto LABEL_125;                  // <- no further decode. No x/y.
  }
  switch ( v28 )                     // ODD mode -> 0x09
  {
    case 1:
      v44 = Decode4();               // x
      nTargetPosition_Y = Decode4(); // y
    case 2: case 3: case 4: default: // no extra reads
  }
```

x/y is read **only** when the mode is odd **and** findMode is 1 — identical to
v83/v84/v87, and identical to what Atlas already encodes.

**The artifact.** `case 9:` and `case 72:` share one label. The exporter
attributed the shared label to its last value (`72`) and dropped the
`(v3 & 1)` branch that separates the two arms, producing the misleading
`v3 == 72 && v28 == 1`. The same collapse shows up differently on neighbours:
v87 renders the parity test as `<default>`, v84 as
`((unsigned __int8)v111 & 1) != 0`.

**Rule this reinforces:** a raw `calls` guard in an export is decompiler
output, not a curated finding. Where a raw guard and a curated per-arm record
disagree, the arm record wins — or the question goes back to the IDB.

`design.md` §2.6 and §8 were amended in this branch to record the withdrawal;
the original claim is preserved there struck through, not deleted.

---

## 3. Key files, by service

### `libs/atlas-constants`
- `character/presence.go` — new. `PresenceState` + `ParsePresenceState`.
  Checked first per repo convention: `libs/atlas-constants/character/` held
  only `constants.go`, `temporary_stat.go`, `energy_charge.go`, and there is no
  presence/liveness type anywhere else in the library.

### `atlas-maps` (`services/atlas-maps/atlas.com/maps`)
- `character/location/` — the whole package. `entity.go` (column),
  `model.go` (field/getter/builder/`ToEntity`), `administrator.go`
  (`setLocationState`, and the `upsertLocation` change that stops a position
  write from clobbering state), `processor.go` (`SetState` /
  `SetStateIfOnline`), `rest.go` (the `state` attribute).
  `resource.go` needs **no** change — it marshals whatever `Transform` returns.
- `kafka/consumer/character/consumer.go` — LOGIN (`:94`), LOGOUT (`:109`),
  CHANNEL_CHANGED (`:141`) each gain one call. `CREATED` and `CHANGE_MAP` are
  deliberately untouched.
- `kafka/consumer/cashshop/consumer.go` — has **no `*gorm.DB` today**; it passes
  `nil` to `_map.NewProcessor`. Threading the handle through changes
  `InitHandlers`'s signature and `main.go:93`. This is the least obvious part of
  the change and is why Task 5 is its own task.

### `atlas-channel` (`services/atlas-channel/atlas.com/channel`)
- `socket/handler/character_chat_whisper.go` — the rewrite. Both `// TODO`s go.
- `maps/location/requests.go` — `Get` + `Model` added alongside `GetField`
  (Task 6), plus `NewModelForTest` (Task 8). `resolve.go`'s `ResolveMapId` is
  **not** touched.
- `character/model.go:64`, `session/model.go:103` — the GM semantics.

### `libs/atlas-packet` and `docs/packets`
- Tests and evidence only. No `Encode`/`Decode` body changes anywhere.

---

## 4. Traps

**`location.Set` must not reset the state.** `upsertLocation` currently does
`db.Save(&e)`, a full-row overwrite. Left alone, every `Set` would silently
write the zero-value state, so `CHANGE_MAP` would knock an online character
offline. Task 2 Step 5 changes it to an explicit `OnConflict` /
`AssignmentColumns` update over the position columns only. The test that catches
a regression here is `TestSet_PreservesState`.

**`location.ResolveMapId` is a trap, not a helper.** It collapses *every*
failure — 404, 5xx, network, decode — to map id 0. That collapse is precisely
how a transport failure renders as a real, confidently-wrong location today. The
find path uses `location.Get` and answers the error shape on failure. Leave
`ResolveMapId` in the tree for its other callers and do not "improve" it here.

**`OFFLINE` must be terminal.** The cash-shop status topic and the character
status topic are separate Kafka topics with no mutual ordering guarantee, and
disconnecting from inside the cash shop emits a LOGOUT and **no**
`CHARACTER_EXIT`. So a `CHARACTER_EXIT` that arrives after a LOGOUT would
resurrect a logged-off character as `IN_FIELD` if the transition were
unconditional. That is what `SetStateIfOnline` exists for, and it is the only
staleness control — there is no TTL and no sweeper, so no invented timeout
constant.

**Channel 7, not 0 or 1.** Several tests deliberately use channel 7. The bug
being fixed is a hard-coded `0` in the channel arm, and the client adds one for
display, so a test written against channel 0 or 1 passes against the broken
code. Do not "simplify" those fixtures.

**GM level 2.** The `{2, true}` case in Task 7 is the whole point of that task.
A test that only covers levels 0 and 1 passes against the old `gm == 1`.

**Never re-run `packet-audit export`.** It is not idempotent — a re-run drifts
~150 unrelated function keys via Hex-Rays variance and degrades cells that have
nothing to do with this branch (`VERIFYING_A_PACKET.md` §10). Task 9
hand-authors eight entries and touches nothing else in `gms_v92.json`.

**Editing an export moves `status.json`'s `exportHashes`.** Task 9 changes
`gms_v92.json`, so `matrix --check` is checking more than the one row that was
intended to move. Read its full output; an unrelated v92 finding is a real
finding, not noise.

---

## 5. Task sizing notes

All nine tasks are within the ≤6-file / one-service guidance. Two are worth a
note:

- **Task 8** touches three files but is by far the largest in content — the
  full decision table plus its test matrix (`{FR-1…FR-7} × {0x09, 0x48}`). It is
  deliberately not split: the rules are one ordered table and splitting it would
  put half the ordering in one reviewer's context and half in another's, which
  is exactly the seam where an ordering bug hides. Its third file is a one-line
  addition to `maps/location/requests.go` — `NewModelForTest`, which the handler
  tests need to build a `location.Model` directly, and which sits next to the
  `SetBaseURLForTest` that exists for the same reason.
- **Task 9** spans `libs/atlas-packet` and `docs/packets/`, but they are one
  unit by the playbook's own rule — test, evidence and matrix must be committed
  together or `matrix --check` fails on dangling evidence. It is independent of
  Tasks 1–8 and can run in any order relative to them.

Task 1 is small enough to fold into Task 2, but is kept separate because it is a
different Go module and both services depend on it; a failure there should not
be reported as an `atlas-maps` failure.

---

## 6. Requirement coverage

| Requirement | Task |
|---|---|
| FR-1 unresolvable name | 8 |
| FR-2 cross-world gate | 8 |
| FR-3 GM concealment | 7, 8 |
| FR-4 cash shop (local and remote) | 2, 3, 5, 6, 8 |
| FR-5 same channel, on a map | 8 |
| FR-6 other channel, real channel id | 2, 3, 4, 6, 8 |
| FR-7 offline / never-logged-in / lookup failure | 1, 2, 4, 6, 8 |
| FR-9 logout from inside the cash shop | 4, 5 |
| FR-11 MTS reported as cash shop | 1 (enum comment), 8 (`CashSceneMts` case) |
| FR-12 both find arms | 8 |
| FR-13 observability | 8 |
| PRD §6.1 at-least-once idempotence | 5 |
| PRD coverage criterion — packet fixtures | 9 |
| OQ-1 presence store form / TTL | dissolved (design §3.5) — no separate store, so no sweeper |
| OQ-2 both arms accept the cash-shop body | answered (design §2.6) — confirmed on v83/v92/v95 |
| OQ-3 GM tier vs boolean | answered — boolean on `level > 0` (design §2.4) |
| OQ-4 instanced maps | answered — leave unchanged (design §2.7) |

PRD §4.2, §5.1, §5.2, §6.1's `atlas-cashshop` store and §7's service-impact table
are **superseded** by the design and are deliberately not implemented. See §1
above.
