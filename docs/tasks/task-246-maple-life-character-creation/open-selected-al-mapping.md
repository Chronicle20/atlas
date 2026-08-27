# OPEN: what do `SendCreateNewCharacter`'s four `SelectedAL` values mean?

**Status:** open — blocks shipping. A placeholder mapping is currently landed in
`50c79fadf` and is **invented, not derived**.
**Raised:** controller session 3, on Task 13's `DONE_WITH_CONCERNS`.
**Owner of the answer:** an IDA derivation pass. This is producible work, not an
external blocker.

## The gap

`CUICharacterSaleDlg::SendCreateNewCharacter` (the `Cash/0543` sub-body) carries,
after the shared `ItemUse` header's `nPOS`/`nItemID`:

```
EncodeStr  sName
Encode4    al[0] .. al[3]     // CUICharacterSaleDlg::GetSelectedAL(this, i)
Encode4    nGender
Encode4    nCurrentClass
Encode4    nSP
Encode4    update_time
```

(`libs/atlas-packet/cash/serverbound/item_use_maple_life.go:26-34`;
derivation.md §2.5, and §2.1 line 283 / §2.4 line 393 for the `GetSelectedAL`
call sites.)

`factory.Processor.SeedCharacter`
(`services/atlas-channel/atlas.com/channel/character/factory/processor.go:36-40`)
needs eighteen values:

```go
SeedCharacter(accountId, worldId, name string, jobIndex uint32, subJobIndex uint16,
  face, hair, color, skinColor uint32, gender byte,
  top, bottom, shoes, weapon uint32,
  strength, dexterity, intelligence, luck byte) (string, error)
```

Four `al` values plus `nGender`, `nCurrentClass` and `nSP` do not obviously
supply that set, and **nothing in the task's materials says how they map**.

## What is currently landed, and why it must not ship

Task 13's implementer flagged this honestly and implemented a documented
best-effort placeholder:

- `al[0..3]` → `face`, `hair`, `hairColor`, `skinColor` (positional)
- `nCurrentClass` → `jobIndex`
- everything else (`top`, `bottom`, `shoes`, `weapon`, `subJobIndex`, all four
  stats) → `0`

No test exercises it, because `seedCharacterFunc` is swapped in every test — so
the whole suite passes regardless of whether the mapping is right. That is
exactly the shape of a defect that ships green.

Per CLAUDE.md, a plausible guess at a wire meaning is not a finding: unverified
is "unknown / unverified." Two independent reasons to distrust the placeholder:

1. **The name suggests a different kind of value.** `GetSelectedAL` returns a
   *selected avatar-look* — plausibly an index into the sale dialog's own option
   list, not a `face`/`hair` template id. If so the mapping is wrong in kind, not
   merely in order, and the factory would validate the ids and return 400 for
   every creation.
2. **The derivation explicitly forbids reasoning by analogy here.** §2.6 answers
   FR-1.1 with: `CUICharacterSaleDlg::SendCreateNewCharacter` and the ordinary
   login `charsb.CreateCharacter` "share no field for field, no opcode, and no
   encode-order structure… **nothing about `CreateCharacter` may be assumed for
   the Maple Life codec**." The placeholder mapping is precisely such an
   assumption, borrowed from `CreateCharacter`'s field order.

## What the derivation pass must answer

1. Decompile `CUICharacterSaleDlg::GetSelectedAL` on at least gms_v83 and
   gms_v95 (the two ends of the in-scope range) and establish what the four
   values *are* — template ids, option indices into a dialog-local table, or
   something else — and which appearance slot each index corresponds to.
2. If they are indices, find where the dialog builds the table they index into
   (`LoadNewCharInfo` and `ShiftNewCharEquip` are named neighbours worth
   checking — derivation.md line 884) and determine what the server needs in
   order to resolve an index to a template id.
3. Establish where `top`/`bottom`/`shoes`/`weapon` come from at all — whether
   the sale dialog sends them, derives them from `nCurrentClass`, or leaves the
   server to apply the tenant's creation template defaults.
4. Establish what `nSP` and `nCurrentClass` mean here, and whether `subJobIndex`
   and the four stats are meant to be server-chosen defaults (the way
   atlas-login's seed path chooses 1/50/5/0) rather than client-supplied.

IDA sessions recorded by this task (derivation.md §"IDA sessions used"):
`gms_v84` = `46c2a2eb`, `gms_v87` = `c0829805`, `gms_v92` = `019cd393`,
`gms_v95` = `ecc757f4`. `gms_v83` is discovered but not adopted
(`MapleStory_dump.exe.i64`) — adopt with `idb_open` before use.

## Until it is answered

The landed placeholder must stay **confined to one clearly-marked place** and be
named as unverified in its doc comment, so it cannot be mistaken for a derived
mapping by the next reader. It must not be duplicated, and no test may be
written that asserts it as correct behavior — a test would convert a known
placeholder into an apparently-verified contract.
