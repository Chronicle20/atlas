# The `Cash/0543` sub-body is the **submit**, not the dialog-open

**Status:** open — needs a ruling before Task 13 can be dispatched.
**Found:** controller session 3, while preparing Task 13's brief.
**Affects:** Task 11 (landed, gated, reviewed), Task 12 (in flight), Task 13 (not started), design §3 / §4.1 / §5.2.

## The defect

`design.md` §3 assumes the `Cash/0543` `ItemUse` sub-body is the **dialog-open**
signal, and normalises it onto:

```go
beginMapleLife(l, ctx, wp)(s session.Model, itemId item.Id, source slot.Position, updateTime uint32)
```

which records a `PhaseOpen` registry entry carrying only item / slot /
updateTime. Task 11 implemented exactly that, at
`socket/handler/character_cash_item_use.go:800-812` and
`socket/handler/maple_life_open.go:38-52`.

But `derivation.md` §2 — landed *after* the design, in Task 1 — proved that the
543 sub-body **is** `CUICharacterSaleDlg::SendCreateNewCharacter`. The committed
codec records it (`libs/atlas-packet/cash/serverbound/item_use_maple_life.go:13-14`,
`packet-audit:fname CUICharacterSaleDlg::SendCreateNewCharacter`), and its field
set is the full character-creation payload:

```
EncodeStr  sName
Encode4    al[0] .. al[3]      // avatar look
Encode4    nGender
Encode4    nCurrentClass
Encode4    nSP
Encode4    update_time
```

That is a submit, not an open. The concrete evidence that this is a live defect
and not a naming quibble: Task 11's arm decodes `sp` and then reads **only**
`sp.UpdateTime()`. `Name()`, `AL0()`–`AL3()`, `Gender()`, `CurrentClass()` and
`SP()` are decoded and discarded — the entire character the player just composed
is dropped on the floor.

## Why the design got it wrong

The design was written before the derivation existed, and §3 explicitly says so:
it wanted OQ-1 answerable "either way without restructuring anything
downstream." The two candidate entry points it planned for were the 543 sub-body
and `USE_MAPLELIFE` (opcode 303). Task 1 then found that **303 is an orphan CSV
placeholder no client path constructs**, so `USE_MAPLELIFE` was struck and
Task 6 wrote no `use.go`.

Removing 303 removed the only entry point the design had left for "open". What
remains is: there is **no open-time packet at all**. The client opens the sale
dialog locally. The only two serverbound messages in the whole feature are

1. the duplicate-name probe, on its own dedicated opcode (derivation §6.2,
   routing outcome (A)), sent while the player composes the name; and
2. the 543 submit, sent when the player confirms.

## Downstream consequences

**Task 11 (landed).** The 543 arm calls `beginMapleLife` and returns. The submit
payload is discarded and no character is ever created. The registry entry it
writes is `PhaseOpen` at a moment when the player has already finished.

**Task 12 (in flight).** Design §5.1 and my own Task 12 addendum told the
implementer to require a live `PhaseOpen` record before answering the probe.
Since the probe arrives *before* the 543 packet, no record can exist yet, so
that gate would reject every legitimate probe. §4.1's justification for the
state — that the probe might share `CHECK_CHAR_NAME`'s opcode and need
disambiguating — is void under routing outcome (A). **Already corrected in
flight**: the implementer has been told to drop the registry dependency
entirely.

**Task 13 (not started).** Its `handleMapleLifeCreate(s, sub)` has no caller —
the 543 arm, which is the only thing that could invoke it, was wired to
`beginMapleLife` instead. And its pre-check gate 1 ("live pending record, phase
`Open`") is unsatisfiable for the same reason as Task 12's: the record would be
created by the very packet being submitted.

**Registry.** `PhaseOpen` becomes vestigial. Entries would be created directly
in `PhaseSubmitted` at submit time. `OpenTTL` and the open-phase sweep lose
their subject. (`Take`, `TakeByTransactionId`, `Submit` and `SubmittedTTL` are
all still needed — Task 14's correlation is unaffected.)

## The shape of the fix

The 543 arm should run the submit flow, not the open flow: decode
`ItemUseMapleLife`, keep the name and look, run design §5.2's pre-checks —
minus gate 1, which no longer has a subject — then `POST characters/seed` and
record a `PhaseSubmitted` entry carrying the returned `transactionId`.

Design §5.2's remaining gates are unaffected in substance, but two of them lose
the `pending` record they were written against and must read from the packet and
session instead:

- **ownership** (gate 2) compared `cashItemInSlotFunc(...)` against
  `pending.ItemId` / `pending.Slot`; those now come from the 543 packet's own
  header (`itemId`, `source`), which is the same data the open record would have
  carried. Note this check already runs upstream at
  `character_cash_item_use.go:61-66` for the common prefix.
- **slot limit** (gate 3), **name re-check** (gate 4) and **session account /
  world** (gate 5) are unchanged.

## What is *not* affected

Tasks 1–10 stand. The packet layer, the codecs, the evidence records, the
template routing, the registry's `Submit`/`Take`/`TakeByTransactionId`
correlation surface, the factory REST client, and the `transactionId` on the
seed status envelope are all correct as landed. Task 14's consumer contract is
unchanged. This is a wiring defect in one arm plus a stale premise in three
design sections — not a re-derivation.
