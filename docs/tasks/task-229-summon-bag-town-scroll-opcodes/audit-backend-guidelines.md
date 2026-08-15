# Backend Audit — task-229-summon-bag-town-scroll-opcodes

- **Scope:** Go files changed 314ff8ad0..481f27e8b (branch is otherwise YAML/JSON packet templates + evidence records)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-14
- **Build:** PASS
- **Tests:** PASS (atlas-packet: `go test -race ./...` all packages `ok`; tools/packet-audit: `go test ./...` all packages `ok`)
- **Overall:** PASS

## Scope note (read before the findings)

Neither `libs/atlas-packet` nor `tools/packet-audit` is a DDD domain service — there
is no `model.go`, `entity.go`, `processor.go`, `resource.go`, `rest.go`, or
`administrator.go` anywhere in either module. The DOM-*/SUB-*/File-Responsibilities
checklists are written against the service-layer DDD pattern (immutable Model +
Builder + Processor + GORM entity + JSON:API REST layer) and are structurally
inapplicable to a wire-protocol codec library and its CLI tooling. Confirmed by
inspection: `grep -rln "type Builder" libs/atlas-packet` returns zero hits anywhere
in the module (not just the diff) — the library-wide constructor idiom is
`NewXxx(...)` value constructors, not the Builder pattern, and that idiom predates
this branch (e.g. `libs/atlas-packet/inventory/serverbound/lottery_item_use.go:26`).
I graded the diff against the conventions that DO apply to this library — private
fields + getters (the immutability half of DDD-Model, which this library observes
throughout), the project's test-helper-file ban, DOM-21 (atlas-constants reuse),
and the explicit checks the task asked for — rather than inventing DOM-* pass/fail
rows for a checklist section that doesn't fit the code.

## Build & Test Results

```
cd libs/atlas-packet && go build ./...   # exit 0
cd libs/atlas-packet && go vet ./...     # exit 0
cd libs/atlas-packet && go test -race ./...   # all packages ok (cached where unmodified)
cd tools/packet-audit && go build ./...  # exit 0
cd tools/packet-audit && go vet ./...    # exit 0
cd tools/packet-audit && go test ./...   # all packages ok
```

## Findings

### 1. PASS — Immutability / private-fields-plus-getters convention observed

`SummonBagItemUse` and `ReturnScrollItemUse` embed `ItemUse` by value and declare
no new fields (`libs/atlas-packet/inventory/serverbound/summon_bag_item_use.go:15-17`,
`libs/atlas-packet/inventory/serverbound/return_scroll_item_use.go:16-18`). The
embedded `ItemUse`'s fields are unexported (`operation`, `updateTime`, `source`,
`itemId` — `libs/atlas-packet/inventory/serverbound/item_use.go:20-26`) and reached
only through getters (`item_use.go:32-38`); the wrappers add no exported field
access and no setters. Deliberately do NOT redeclare `Encode`/`Decode`/`Operation`
so the wire format stays byte-identical to the shared body — per the task brief,
this is correct and I am not recommending redeclaration.

### 2. PASS — Comments adequately guard against accidental deletion

Both new files carry a "Nothing calls this... Do not 'simplify' it away (task-229)"
sentence directly in the type doc comment (`summon_bag_item_use.go:11-12`,
`return_scroll_item_use.go:12-13`), plus the rationale for why the shared codec was
rejected. A future maintainer running a dead-code sweep would need to actively
ignore an explicit in-file instruction to delete these, which is about as strong a
guard as a comment can provide. No finding.

### 3. MINOR — `NewSummonBagItemUse()` / `NewReturnScrollItemUse()` are unused, even by their own tests

`summon_bag_item_use.go:19-21` and `return_scroll_item_use.go:20-22` each declare an
exported constructor. Neither is called anywhere in the repository:

```
grep -rn "NewSummonBagItemUse\|NewReturnScrollItemUse" --include="*.go" .
→ only the two declaration sites themselves
```

Contrast with the sibling audit-covered-but-still-live codec `LotteryItemUse`, whose
constructor IS exercised by production: `NewLotteryItemUse()` is called from
`services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go:56`.
`SummonBagItemUse`/`ReturnScrollItemUse` are audit-only by design (per the task
brief and design.md D1) so a production call site is correctly absent — but their
own `_test.go` files don't use the constructor either; both
`TestSummonBagItemUseRoundTrip` (`summon_bag_item_use_test.go:33,39`) and
`TestReturnScrollItemUseRoundTrip` (`return_scroll_item_use_test.go:28,34`)
construct the struct via a raw composite literal
(`SummonBagItemUse{ItemUse: ItemUse{...}}`) instead of calling `NewSummonBagItemUse()`.
The exported constructor is therefore dead code with zero exercised paths — go
vet doesn't flag it (unused *functions* aren't a vet check, only unused
imports/locals), so the build stays green, but it's a maintenance liability: a
future edit to the constructor's default wiring (e.g. changing which `operation`
string it seeds) would go untested. Not blocking — this doesn't violate a
checklist item, since no analogous DOM check exists for this library — but should
be fixed by either deleting the two constructors or having the round-trip tests
build `input` through them instead of a literal.

### 4. PASS — Tests are behavioral, not tautological (round-trip half)

`pt.RoundTrip` (`libs/atlas-packet/test/roundtrip.go:22-32`) actually encodes,
re-decodes through a real `request.Reader`, and asserts zero unconsumed bytes
(`roundtrip.go:30-32`) — a genuine round-trip assertion, not a mock. Both new
`TestXxxRoundTrip` functions iterate `pt.Variants` (11 tenant variants —
`libs/atlas-packet/test/context.go:19-35`) and then re-check each field getter
against the input (`summon_bag_item_use_test.go:41-52`,
`return_scroll_item_use_test.go:36-47`). This is meaningful coverage.

### 5. INFORMATIONAL — `TestXxxMatchesSharedBody` is guaranteed to pass by Go's embedding semantics as currently written, but has legitimate regression value

`TestSummonBagItemUseMatchesSharedBody` (`summon_bag_item_use_test.go:59-73`) and
`TestReturnScrollItemUseMatchesSharedBody` (`return_scroll_item_use_test.go:52-66`)
compare `wrapper.Encode(...)` byte-for-byte against `shared.Encode(...)`. Because
the wrapper structs declare no `Encode` override, `wrapper.Encode` IS
`ItemUse.Encode` called through the promoted method — there is no code path by
which these two byte slices could currently differ. This isn't a tautology in the
"asserts nothing real" sense the guideline worries about, though: it's a
regression guard against a *future* PR that adds a wrapper-local `Encode` method
(the exact "simplify it away" or "customize it" move the header comment warns
against) — it would immediately start failing then. Not a finding, just flagging
that its present-day pass is structural rather than behavioral, so don't read it
as proof the wire format was independently re-derived.

### 6. PASS — No test-helper-file violation

No `*_testhelpers.go` file was added. All fixture construction lives inline in the
`_test.go` files themselves (`summon_bag_item_use_test.go`,
`return_scroll_item_use_test.go`) or uses the existing shared helper package
`libs/atlas-packet/test` (`pt.Variants`, `pt.CreateContext`, `pt.RoundTrip` — all
three imported and used, e.g. `summon_bag_item_use_test.go:6,30,32,40`). Complies
with CLAUDE.md "Test Helper Pattern."

### 7. PASS — DOM-21 (no reinvented atlas-constants types)

No new `type`, `const` block, or numeric-literal item/inventory/world classification
was introduced. The only "domain-shaped" additions are `CharacterItemUseSummonBagHandle`
/ `CharacterItemUseTownScrollHandle` string constants, and those already existed
pre-branch (`git diff 314ff8ad0..481f27e8b -- libs/atlas-packet/inventory/serverbound/item_use.go`
produces no output — that file is untouched, exactly as the task brief states). The
test-fixture item IDs (`2100000` in `summon_bag_item_use_test.go:37`, `2030000` in
`return_scroll_item_use_test.go:32`) are opaque wire values fed through a byte
encoder, not classification inputs — no `itemId / 10000` style branching exists
anywhere in the diff to check against `libs/atlas-constants`. No finding.

### 8. PASS — Error handling unchanged from established convention

`ItemUse.Decode` has no error return (`item_use.go:53-58`) and the wrappers embed
it without alteration — this matches the pre-existing convention for this whole
codec family (every other `serverbound` struct in the package follows the same
error-free `Decode` shape). Not a new deviation introduced by this branch.

### 9. MINOR — `export.go:44-47` regex extension is under-tested against embedded-substring false positives

```go
// export.go:42-47
// fnameToken matches an FName-looking Class::Method token, OR a bare IDA
// sub_XXXXXX symbol (used to scrape candidate roster entries out of
// _pending.md prose)...
var fnameToken = regexp.MustCompile(`[A-Z][A-Za-z0-9_]+::[A-Za-z0-9_]+|sub_[0-9A-Fa-f]+`)
```

The `sub_[0-9A-Fa-f]+` alternative has no leading word-boundary anchor (`\b` or
equivalent), so `FindAllString` will match `sub_<hex>` as a **substring** of any
longer identifier that happens to contain that literal sequence — e.g.
`consub_1a2b3c`, `some_sub_deadbeef` (an underscore-joined identifier is enough;
`_pending.md` prose in this repo does use underscore_separated tokens elsewhere,
e.g. registry keys like `gms_v72`). `tools/packet-audit/cmd/fname_token_test.go`
adds three tests
(`TestFnameTokenMatchesBareSubSymbol:10`, `TestFnameTokenStillMatchesClassMethod:19`,
`TestFnameTokenRejectsBareEnglishWords:28`) — the third only proves plain English
words *without* a trailing `_<hex>` don't match (`subject`, `submit`, `subroutine
call`, `sub-total`), which is a different (and easier) property than "an
embedded `sub_<hex>` inside a larger token isn't scraped." No test exercises the
actual over-broad-matching risk the regex change introduces. Practical blast
radius is low — a false-positive token only ever reaches `candidatesFromFName`
(`run.go:2185`), which returns no candidate for any string that isn't an exact
literal `case` match, so a spurious scrape is silently absorbed rather than
mis-routed to the wrong packet. Still, this is a shared tool other packet tasks
depend on (per the task brief), and the gap is real: recommend adding
`\bsub_[0-9A-Fa-f]+\b` (or an equivalent explicit boundary check) plus a test
case such as `"reference: xsub_1a2b3c and yconsub_deadbeef"` asserting the
substring form is rejected.

### 10. PASS — New `candidatesFromFName` cases are individually tested

`run.go:2202-2237` adds six new `case` arms (`CWvsContext::SendMobSummonItemUseRequest`,
`sub_955499`, `sub_904154`, `CWvsContext::SendPortalScrollUseRequest`,
`CWvsContext::SendReturnScrollUseRequest`, `sub_841AA5`), each carrying an
inline comment citing the IDA address and why the fname form differs per version.
`disambiguation_test.go:180-215` (`TestCandidatesItemUseFamilyWrappers`) exercises
all six plus a regression guard that the potion sender
(`CWvsContext::SendStatChangeItemUseRequest`) still keys to the shared `ItemUse`
codec, not one of the new wrappers. Good 1:1 coverage.

### 11. PASS — v48 fname correction is evidence, not assertion

The `sub_719DD9` case removed from `run.go` (previously mis-attributed to
`SendStatChangeItemUseRequest`) is replaced with a doc comment
(`run.go:2185-2193`) explaining the re-decompile that proved the old mapping
wrong, and the corresponding `item_use_test.go` marker/fixture was re-pinned to
the correct address `0x70db3c` (`item_use_test.go` diff, `TestItemUseBytesV48`)
with the byte fixture comments updated to the new offsets. This is a textbook
"verify against source, correct the record" fix per the project's grounding
rules — not a finding, called out as compliant because the correction is fully
cited (old address, why it was wrong, new address, new decompile line numbers).

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- `libs/atlas-packet/inventory/serverbound/summon_bag_item_use.go:19-21` and
  `return_scroll_item_use.go:20-22`: exported constructors `NewSummonBagItemUse`/
  `NewReturnScrollItemUse` are never called, including by their own tests — either
  delete them or route the round-trip tests' `input` construction through them.
- `tools/packet-audit/cmd/export.go:44-47`: the `sub_[0-9A-Fa-f]+` regex
  alternative has no word-boundary anchor and can match as a substring of a
  larger identifier; add an explicit boundary (`\b`) and a test case proving
  embedded-substring forms are rejected, since this scraper is shared
  infrastructure other packet tasks rely on.

## Resolution (post-review)

Both non-blocking findings are fixed on this branch (commit
`fix(task-229): address the non-blocking code-review findings`):

- `summon_bag_item_use_test.go` / `return_scroll_item_use_test.go` now build both
  `input` and `output` through `NewSummonBagItemUse` / `NewReturnScrollItemUse`,
  so the exported constructors are exercised and the operation defaulting they
  encode is what the round-trip asserts against. The constructors are kept
  (rather than deleted) to match `NewLotteryItemUse`'s shape in the same package.
- `tools/packet-audit/cmd/export.go:47` anchors the `sub_` alternative as
  `\bsub_[0-9A-Fa-f]+\b`. `TestFnameTokenRejectsEmbeddedSubSymbol` pins the
  rejection of `prefix_sub_841AA5`, `sub_841AA5_thunk`, `xsub_841AA5` and
  `sub_841AA5z`; the two pre-existing accept-cases still pass unchanged.
