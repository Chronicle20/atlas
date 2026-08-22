# Extra-Expression (Emote) Cash Items — Design

Version: v1
Status: Draft
Created: 2026-08-21
Input: `docs/tasks/task-247-extra-expression-items/prd.md` (approved)

---

## 0. Summary of the shape

Three independent slices of work that share one file:

1. **Gate** — `CharacterExpressionHandleFunc` grows two guards (range, then
   ownership) in front of the existing `expression.Change` call. New constants
   and one package-level test seam. Confined to `atlas-channel` plus one small
   new file in `libs/atlas-constants/item`.
2. **Thread** — `duration` and `byItemOption` become real parameters on four
   Kafka structs, two processors, two producers, and one packet constructor.
   Purely additive, mechanical, and compiler-checked end to end.
3. **Correct the backlog** — one table row in
   `docs/research/missing-features/items-and-consumables.md`.

Slices 1 and 2 touch the same handler function but no other shared line, so they
can be implemented and reviewed independently. Slice 2 is the wider blast radius
(two services); slice 1 is the one with a security argument behind it.

---

## 1. Client evidence resolved during design

The PRD left four open questions. Three are answered here from the GMS v95 IDB
(session `ecc757f4`); the fourth is a scope call.

### 1.1 Open Question 1 — RESOLVED, and it changes an assumption

`CUserLocal::UseFuncKeyMapped` @`0x932e20` was decompiled. It has **two**
`SendEmotionChange` call sites, and neither behaves the way the PRD tentatively
assumed:

| Site | FUNCKEY type | Code | Meaning |
|---|---|---|---|
| `0x933898` | `case 3u` | `v44 = v11->nID % 100 + 8;` then `if ((v3 & 0x100) != 0 \|\| !CWvsContext::IsExist(lParam, v11->nID)) return 1;` then `SendEmotionChange(pInst, v44, 0, -1)` | **Extra-expression** key. `nID` is the `516xxxx` item id. Client gates on `CWvsContext::IsExist(itemId)` @`0x9f3150` — its own inventory-ownership check. |
| `0x933859` | `case 6u` | `SendEmotionChange(pInst, v11->nID - 99, 0, -1)` | **Base emote** key. `nID` is `100..106`, so the emote is `1..7`. |

Two consequences, both material:

- **The client already performs exactly the ownership check this task adds
  server-side.** `case 3u`'s `CWvsContext::IsExist(nID)` is the client-side twin
  of FR-2.1. Our gate is not inventing a rule; it is enforcing a rule the stock
  client already obeys and a modified client can simply delete. This is the
  strongest available justification for FR-2.4's drop-on-not-owned.
- **The base-emote path also sends `duration = -1`.** The PRD hoped
  `UseFuncKeyMapped`'s base-emote case might send `0`. It does not. Combined
  with `CDraggableItem::OnDoubleClicked` → `SendEtcCashItemUseRequest` `case 6:`
  @`0xa02c86` (also `-1`), **every keyboard- and item-driven emote a GMS v95
  client sends carries `duration = -1` and `byItemOption = 0`.**

So FR-3.1's verbatim forwarding is not a marginal change on v95 — it changes the
broadcast `duration` for *every* emote from the current hardcoded `0` to
`0xFFFFFFFF`. See §5.1 for how this design handles that.

### 1.2 What the receiving client does with `duration`

`CUser::OnEmotion` @`0x8e0150`:

```
v3 = Decode4();                              // emotion
v4 = Decode4();                              // duration
this->m_bEmotionByItemOption = Decode1();
CAvatar::SetEmotion(&this->CAvatar, v3, v4);
```

`CAvatar::SetEmotion` @`0x466b00` re-checks `nEmotion <= 0x17` before applying —
an out-of-range emote is silently dropped by the receiver — then calls
`CAvatar::PrepareFaceLayer(this, tDuration)` @`0x4647d0`, which sets
`m_tEmotionEnd = tDuration + get_update_time()` and animates the face layer with
`GA_REPEAT`.

**Unverified:** the exact expiry predicate that reads `m_tEmotionEnd` was not
located (`xrefs_to_field` on `CAvatar::m_tEmotionEnd` returns nothing in this
IDB, and the assignment's decompiled operand is register-aliased). With
`tDuration = -1` the end stamp lands one tick in the past, so a naive
`now > m_tEmotionEnd` check would expire the emote immediately. Whether the
client guards on `tDuration > 0` first is **not established**. §5.1 records this
as the one live risk in slice 2 and names the fallback.

Counter-evidence in favour of forwarding verbatim: an official GMS server relays
what the client encoded, and the client encodes `-1` for every emote it
originates. Reproducing that is by construction correct; synthesising `0`
instead is the deviation. The PRD's FR-3.1/FR-3.5 choose verbatim, and this
design implements that.

### 1.3 Open Question 2 — no clamp

`CAvatar::SetEmotion` bounds the *emote*, not the duration, and `m_tEmotionEnd`
is a millisecond stamp with no observed ceiling check. There is no evidence a
large value misbehaves beyond "the face layer stays up." Clamping would also
break the `-1` round trip that §1.1 shows is the normal case. **No clamp.**

### 1.4 Open Question 4 — `TransactionId` is in scope

`atlas-expressions`' `Command` declares `TransactionId uuid.UUID`;
`atlas-channel`'s does not, so `handleChangeCommand` threads a zero UUID into
`ChangeAndEmit` and out onto every `StatusEvent`. Fixing it is one field plus
one `uuid.New()` in a function this task already edits. **Include it** (§4.4).

---

## 2. Slice 1 — the gate

### 2.1 Where the emote→item mapping lives

`libs/atlas-constants/item/` already owns small per-family files with named ids
and predicates (`death_protection.go`, `vegas_spell.go`). Add
`libs/atlas-constants/item/expression.go` in that mould.

The PRD's `itemId = 5159992 + emote` is correct but opaque. Derive it from the
classification that already exists rather than landing the magic number:

```go
package item

// Emote ids the client can legitimately originate.
//
// CAvatar::SetEmotion@0x466b00 (GMS v95) applies the emotion only when
// nEmotion <= 0x17, and CWvsContext::SendEmotionChange@0x9f9386 refuses to
// send anything above the same bound. 23 is therefore the widest value a
// stock client can put on the wire in either direction.
const MaxEmoteId = uint32(23)

// MaxBaseEmoteId is the highest emote every character has without owning a
// cash item. Emotes above it are the ClassificationExpression (516) unlocks.
const MaxBaseEmoteId = uint32(7)

// IsExtraExpressionEmote reports whether an emote requires an owned
// ClassificationExpression cash item.
func IsExtraExpressionEmote(emote uint32) bool {
	return emote > MaxBaseEmoteId && emote <= MaxEmoteId
}

// ExtraExpressionItemId maps an extra-expression emote to the cash item that
// unlocks it, reporting false for emotes outside the gated range.
//
// CWvsContext::SendEtcCashItemUseRequest@0xa02c86 and
// CUserLocal::UseFuncKeyMapped@0x933874 (GMS v95) both compute the emote as
// nItemID % 100 + 8, so the item's index within classification 516 is
// emote - 8: emote 8 -> 5160000, emote 22 -> 5160014.
func ExtraExpressionItemId(emote uint32) (Id, bool) {
	if !IsExtraExpressionEmote(emote) {
		return Id(0), false
	}
	return Id(uint32(ClassificationExpression)*10000 + emote - MaxBaseEmoteId - 1), true
}
```

`uint32(ClassificationExpression)*10000` is `5160000`, so the expression reduces
to `5159992 + emote` exactly as FR-2.1 specifies — but the reader can see *why*.
FR-2.3 is satisfied without a branch: emote 23 yields `5160015`, an item id no
v83.1 character can hold, so the ownership check fails on its own.

`MaxEmoteId` deliberately lives here rather than in the handler so the
constants-reuse convention has one home for both halves of the rule.

### 2.2 Handler structure

`CharacterExpressionHandleFunc` becomes:

```go
func CharacterExpressionHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(...) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.ExpressionRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		emote := p.Emote()

		// FR-1.1 — CWvsContext::SendEmotionChange@0x9f9386 refuses to send
		// above 0x17, so a larger value cannot come from a stock client.
		if emote > item.MaxEmoteId {
			l.Warnf("Character [%d] requested out-of-range expression [%d]. Dropping.", s.CharacterId(), emote)
			return
		}

		// FR-2.1 — extra expressions require the 516xxxx cash item. The stock
		// client applies the same rule itself: CUserLocal::UseFuncKeyMapped
		// case 3 @0x933884 gates on CWvsContext::IsExist(nItemID).
		if itemId, ok := item.ExtraExpressionItemId(emote); ok {
			owns, err := expressionItemOwnedFunc(l, ctx, s.CharacterId(), itemId)
			if err != nil {
				// FR-2.5 — fail closed.
				l.WithError(err).Warnf("Unable to verify character [%d] owns item [%d] for expression [%d]. Dropping.", s.CharacterId(), itemId, emote)
				return
			}
			if !owns {
				l.Warnf("Character [%d] requested expression [%d] without owning item [%d]. Dropping.", s.CharacterId(), emote, itemId)
				return
			}
		}

		_ = expression.NewProcessor(l, ctx).Change(s.CharacterId(), s.Field(), emote, p.Duration(), p.ByItemOption())
	}
}
```

Emotes `0`–`7` never enter the `if` body, so FR-1.3 and the NFR "zero-lookup
path for base emotes" hold structurally rather than by a separate branch.

### 2.3 The ownership seam

Per FR-2.6/FR-2.7, follow the package's established package-var injection
precedent (`cashItemInSlotFunc`, `karmaCharacterProcessorFunc` in
`character_cash_item_use.go:1009-1036`). Declare it beside the handler, in
`character_expression.go`:

```go
// expressionItemOwnedFunc is a test seam for the extra-expression ownership
// check (package-var injection precedent: cashItemInSlotFunc in
// character_cash_item_use.go). Handler tests must not require a live character
// service to assert which branch a request reached.
var expressionItemOwnedFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, itemId item.Id) (bool, error) {
	cp := character.NewProcessor(l, ctx)
	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return false, err
	}
	_, ok := c.Inventory().Cash().FindFirstByItemId(uint32(itemId))
	return ok, nil
}
```

This is the decorated-model path FR-2.6 names, and it is the exact idiom already
used for a cash-permit ownership check at
`socket/handler/character_interaction.go:122-131`. **No new method is added to
`character.Processor`** — the existing surface covers it, and adding
`GetItemByTemplateId` would be a new abstraction for one caller.

Returning `(bool, error)` rather than `(*asset.Model, error)` keeps the seam
minimal: nothing downstream needs the asset, and a narrower seam is a narrower
thing for a test to fake.

### 2.4 Why no `CashSlotItemType(6)` arm

Restating the PRD's non-goal so a later reader does not "fix" it: §1.1's
`case 3u`/`case 6:` evidence shows type-6 items are converted to an emote
request by the client before any cash-item-use packet is built. An arm in
`character_cash_item_use.go` would be unreachable from a stock client and
would duplicate the gate this task puts on the reachable path. A test asserting
`character_cash_item_use.go` is untouched is not practical; the acceptance
criterion is a review check against the branch diff.

### 2.5 Tests (slice 1)

New `socket/handler/character_expression_test.go`, table-driven over the seam:

| Case | Emote | Seam behaviour | Assert |
|---|---|---|---|
| base emote | 5 | seam must not be called | forwarded; seam call count == 0 |
| base emote upper bound | 7 | seam must not be called | forwarded; seam call count == 0 |
| owned extra | 8 | returns `(true, nil)` for `5160000` | forwarded; seam called with `5160000` |
| unowned extra | 8 | returns `(false, nil)` | dropped |
| lookup error | 8 | returns `(false, err)` | dropped (fail closed) |
| gated upper bound | 23 | returns `(false, nil)` for `5160015` | dropped |
| out of range | 24 | seam must not be called | dropped; seam call count == 0 |

"Forwarded" is asserted at the seam boundary only — the emit itself goes through
`expression.NewProcessor(...).Change`, which produces to Kafka. Two options:

- **Chosen:** assert the *drop* cases by seam-call-count and by the absence of
  any further effect, and assert the *forward* cases reach the seam with the
  right item id. The Kafka emit is not exercised; `Change` failing without a
  broker is tolerated (the handler already discards its error with `_ =`).
- Rejected: a second seam over `expression.Processor` purely to observe the
  emit. It would double the seam surface to assert something the range/ownership
  logic does not decide.

`libs/atlas-constants/item/expression_test.go` covers the mapping boundaries
FR-2.3 and the acceptance criteria name: `8 → 5160000`, `22 → 5160014`,
`23 → 5160015`, plus `7 → (0, false)` and `24 → (0, false)`.

Test setup uses the repo Builder pattern for `session.Model`; no
`*_testhelpers.go`.

---

## 3. Slice 2 — threading `duration` and `byItemOption`

### 3.1 Types on the wire between services

| Field | Type | Rationale |
|---|---|---|
| `Duration` | `int32` | `ExpressionRequest.Duration()` is `int32` and the real value is `-1`. Carrying it as `int32` keeps `-1` legible in the JSON payload (`"duration": -1`) instead of `4294967295`, which matters for operator debugging and for the PRD's documented contract in §5. |
| `ByItemOption` | `bool` | Matches both ends. |

The `int32`→`uint32` narrowing happens exactly once, at the writer boundary
(§3.4), which is where FR-3.5's bit-pattern requirement lives.

### 3.2 `atlas-channel` chain

```
character_expression.go
  → expression.Processor.Change(characterId, field, expression uint32, duration int32, byItemOption bool)
  → SetCommandProvider(characterId, field, expression, duration, byItemOption)
  → kafka/message/expression.Command{ ..., Duration int32, ByItemOption bool }
```

and inbound:

```
kafka/message/expression.Event{ ..., Duration int32, ByItemOption bool }
  → kafka/consumer/expression.handleEvent
  → charpkt.NewCharacterExpression(e.CharacterId, e.Expression, uint32(e.Duration), e.ByItemOption)
```

The TODO block at `consumer.go:57-61` is deleted, not amended (FR-3.3).

### 3.3 `atlas-expressions` chain

```
kafka/message/expression.Command{ ..., Duration int32, ByItemOption bool }
  → handleChangeCommand
  → Processor.ChangeAndEmit(transactionId, characterId, field, expression, duration, byItemOption)
  → Processor.Change(mb, ...) — widened identically
  → changeInput gains duration/byItemOption
  → expressionEventProvider(transactionId, characterId, field, expression, duration, byItemOption)
  → kafka/message/expression.StatusEvent{ ..., Duration int32, ByItemOption bool }
```

`expression.Model` and `registry.add` are **not** touched (FR-3.8). `Change`
still writes only `(characterId, field, expression)` into the TTL registry,
because the only reader — `revertExpression` — needs fixed zero values anyway.

`revertExpression` (`expression/task.go:50`) becomes
`expressionEventProvider(transactionId, exp.CharacterId(), exp.Field(), 0, 0, false)`.
Passing the literals explicitly at the one call site that wants them is
preferable to a `revertExpressionEventProvider` wrapper — it keeps FR-3.7's
guarantee visible at the line that provides it.

`Processor` is an interface with a generated mock at `expression/mock/processor.go`;
widening `Change`/`ChangeAndEmit` requires updating the mock's function-field
signatures and any test that sets them.

### 3.4 The packet constructor (FR-3.4)

**Chosen: widen the constructor.**

```go
func NewCharacterExpression(characterId uint32, expression uint32, duration uint32, byItemOption bool) CharacterExpression {
	return CharacterExpression{characterId: characterId, expression: expression, duration: duration, byItemOption: byItemOption}
}
```

Rationale over a `WithByItemOption` variant:

- The struct is immutable with no other `With…` methods; introducing one for a
  single field sets a precedent the package does not otherwise follow.
- The compiler enumerates every call site. There are four
  (`kafka/consumer/expression/consumer.go:62`, `v61_test.go:724`,
  `v72_test.go:506`, `v79_test.go:520`) and the three test call sites are all
  GMS ≤ 87 fixtures where `byItemOption` is not encoded, so `false` is the
  correct addition and **their asserted byte strings do not change** — which is
  precisely the FR-3.4 "Go-surface change only" guarantee, demonstrated rather
  than asserted.
- `clientbound/expression_test.go` constructs the struct literally, not through
  the constructor, so the round-trip evidence test is untouched.

`uint32(e.Duration)` at `consumer.go:62` is the FR-3.5 conversion: Go's
`int32`→`uint32` conversion is defined to preserve the bit pattern, so `-1`
becomes `0xFFFFFFFF`, and `Writer.WriteInt` emits it little-endian as
`FF FF FF FF`. Add a comment at that line stating the intent, because a bare
`uint32(...)` of a signed value reads like a bug.

### 3.5 Version behaviour (FR-3.6)

Nothing in this slice touches an `Encode`/`Decode` gate. On GMS ≤ 87 and JMS,
`ExpressionRequest.Decode` never reads the two fields, so they arrive at the
handler as `0`/`false`, travel through Kafka as `0`/`false`, and hit
`CharacterExpression.Encode`, whose own gate declines to write them. The v83,
v84, v87 and JMS v185 byte output is unchanged by construction.

### 3.6 Tests (slice 2)

- **`libs/atlas-packet/character/clientbound`** — extend
  `TestCharacterExpressionRoundTrip`? No: it already covers the fields per
  variant. Add a dedicated `-1` byte-level assertion instead, since the existing
  test's `duration: 3000` cannot exercise the sign path:
  `NewCharacterExpression(12345, 8, uint32(int32(-1)), false).Encode` under a
  GMS v95 context must end in `FF FF FF FF 00`. This is the acceptance
  criterion's "byte-level test on the writer".
- **`atlas-channel`** — `character/expression/producer_test.go` (new, if absent):
  `SetCommandProvider` populates `Duration`/`ByItemOption`.
- **`atlas-expressions`** — extend `expression/processor_test.go` for the
  widened `Change` and `expression/task_test.go` to assert the revert path still
  emits `duration = 0, byItemOption = false`.
- The existing `v61`/`v72`/`v79` evidence tests must pass with their **assertion
  strings unmodified** — only the constructor call gains `, false`. Any diff to
  an expected byte string in those files is a design violation.

---

## 4. Slice 3 and adjacent items

### 4.1 Backlog correction (FR-4.1)

The PRD cites lines 31 and 80. The working copy of
`docs/research/missing-features/items-and-consumables.md` has since been edited;
the claim now appears **once**, at line 49:

```
| Extra-expression (emote) items | `ClassificationExpression` → type 6 | — | S |
```

inside the "Remaining one-off cash types" table, whose preamble reads *"All
mapped in `GetCashSlotItemType` but unimplemented (fall to warn)"* — which is
the false premise. There is no longer a second itemisation to correct;
implementers must re-grep rather than trust the PRD's line numbers.

Replace the row's premise with the evidence, and note the entry is closed by
this task:

> Extra-expression (emote) items are **not** routed through the cash-item-use
> handler. `CDraggableItem::OnDoubleClicked` @`0x50814b` checks
> `get_etc_cash_item_type` first and calls
> `CWvsContext::SendEtcCashItemUseRequest` @`0x508165`, whose `case 6:`
> @`0xa02c86` issues `CWvsContext::SendEmotionChange(nItemID % 100 + 8, 0, -1)`.
> The keyboard path is the same: `CUserLocal::UseFuncKeyMapped` `case 3u`
> @`0x933874`. A `CashSlotItemType(6)` dispatch arm would be dead code. The real
> gap was the unvalidated emote path, closed by task-247.

If the row's table position makes the prose awkward, move the entry out of the
table into a short numbered subsection rather than truncating the evidence.

### 4.2 `TransactionId` (§1.4)

Add `TransactionId uuid.UUID \`json:"transactionId"\`` to `atlas-channel`'s
`Command` and set it to `uuid.New()` in `SetCommandProvider`. This makes
`atlas-expressions`' existing `c.TransactionId` non-zero for the first time,
which flows into `StatusEvent.TransactionId` — additive and observable only in
logs/traces. Keep it a separate commit from slices 1 and 2 so it can be dropped
in review without unpicking anything else.

---

## 5. Risks and alternatives considered

### 5.1 `duration = -1` on the wire (the one real risk)

§1.1 establishes that after this change **every** GMS v95 emote broadcast
carries `0xFFFFFFFF` instead of today's `0`, and §1.2 leaves unverified whether
the receiving client's expiry check tolerates a past `m_tEmotionEnd`.

- **Why proceed:** the value is what the client itself encoded; an official
  server relays it; and synthesising `0` is the deviation, not the fidelity.
  FR-3.1 and FR-3.5 make this call explicitly.
- **How it would present:** on a v95 tenant, remote observers see the emote
  flash and vanish rather than hold. The emoting player's own client is
  unaffected (it applied the emote locally before sending).
- **Fallback if live testing shows that:** clamp negatives to `0` at the single
  writer boundary in `kafka/consumer/expression/consumer.go` —
  `d := e.Duration; if d < 0 { d = 0 }` — restoring today's observable
  behaviour for the `-1` case while keeping positive durations honest. This is a
  three-line change at one site precisely because §3.1 confines the conversion
  there. **Do not** pre-emptively add the clamp; it would make the acceptance
  criterion "`-1` arrives as `0xFFFFFFFF`" unsatisfiable.
- **Not deferred:** this is a documented risk with a named, cheap remediation,
  not a blocker. Phase 5 (`/fix-pr-bug`) is the right place if live testing
  contradicts §1.2.

### 5.2 One character-service round trip per extra emote

The gate calls `GetById(InventoryDecorator)`, which fetches the character's full
inventory. Considered and rejected: caching ownership per session. The client's
own 2000 ms cooldown (`SendEmotionChange` @`0x9f9386`) bounds the rate, base
emotes never take this path, and a session-scoped cache would go stale when the
item is traded or expires — turning a cheap correct check into a cheap wrong
one. Revisit only if profiling shows it matters.

### 5.3 Rejected: validating against `atlas-data` rather than a constant

An alternative to the arithmetic mapping is asking `atlas-data` which `516xxxx`
items exist for the tenant's version and gating on that set. Rejected: it adds a
second service dependency to a hot cosmetic path, and it is strictly weaker than
the client's own rule, which is pure arithmetic with no data lookup
(`nItemID % 100 + 8`). Mirroring the client's arithmetic is the higher-fidelity
choice; a nonexistent item id (emote 23) is handled by the ownership check
failing, exactly as FR-2.3 requires.

### 5.4 Rejected: server-side cooldown and morph block

Out of scope per the PRD. Recording why so it is not re-litigated: both are
purely cosmetic UX gates, and implementing the cooldown would require
per-character timing state in `atlas-expressions` (the registry is a 5-second
TTL store keyed for a different purpose). No exploit surface is left open — the
gate that matters, ownership, is enforced.

---

## 6. Implementation order

Three commits, in this order, each independently green:

1. **`libs/atlas-constants/item/expression.go` + its test.** No dependents yet;
   proves the mapping in isolation.
2. **Slice 2 (thread duration/byItemOption).** Touches both services and the
   packet lib. Land before slice 1 so the handler's final `Change(...)` call in
   slice 1 is written once against its final signature.
3. **Slice 1 (the gate) + handler tests.** Depends on 1 and 2.

Then, separately: **4. `TransactionId`** and **5. the backlog correction.**

Commit 2 is the one that must not be split across services — `atlas-channel`'s
producer and `atlas-expressions`' consumer are compiled independently but share
the JSON contract, and a half-landed slice 2 is silently lossy rather than
broken. Land both services' halves together.

## 7. Verification

- Module-local `go build ./... && go test ./...` in `libs/atlas-constants`,
  `libs/atlas-packet`, `services/atlas-channel`, `services/atlas-expressions`.
- Flagless `tools/verify.sh` exits 0 before the branch is called done.
- `backend-guidelines-reviewer` over the changed Go packages in both services,
  and `plan-adherence-reviewer` over the plan.
- Manual confirmation, from the branch diff, that
  `socket/handler/character_cash_item_use.go` is untouched.
- Manual confirmation that no expected-byte string in `v61_test.go`,
  `v72_test.go`, `v79_test.go` changed.
