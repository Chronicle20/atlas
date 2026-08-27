# Review — task-240 tail artifacts (unreviewed at merge)

Reviewer pass over two self-concealing artifacts that landed without review:

1. `tools/packet-audit/cmd/run.go` as changed by commit `71fc76306`.
2. The hand-spliced `gms_v95.json` export entry added by commit `02ab37435`
   (plus the audit report / STATUS.md / status.json / evidence yaml it
   produced).

Scope was exactly these two units; no other file in either commit's diff was
treated as part of the reviewed surface except where correctness of the
reviewed hunk genuinely depended on it (e.g. `candidatesFromFName`,
`selectCandidates`, `idasrc.ParsePrim`, the Atlas struct the new candidate
points at, and the tests the new evidence file cites).

---

## Artifact 1 — `tools/packet-audit/cmd/run.go` @ `71fc76306`

### The diff

```go
 case "CCashShop::OnBuyPackage":
     return []candidate{{name: "ShopOperationBuyPackage", dir: csvpkg.DirServerbound, pkg: "cash"}}
+case "CCashShop::OnGiftPackage":
+    return []candidate{{name: "ShopOperationBuyOtherPackage", dir: csvpkg.DirServerbound, pkg: "cash"}}
 case "CCashShop::OnBuyCouple":
```

That is the entirety of the change to this file in this commit (confirmed via
`git show 71fc76306 -- tools/packet-audit/cmd/run.go`; the commit's other 36
files are docs/task artifacts and one test-comment tweak). It adds a single
new `case` arm to `candidatesFromFName`, the IDA-FName → Atlas-candidate
lookup table.

### Checks performed

- **Duplicate/collision risk** (`grep -n "ShopOperationBuyOtherPackage\|OnGiftPackage" tools/packet-audit/cmd/run.go`):
  `CCashShop::OnGiftPackage` and `ShopOperationBuyOtherPackage` each appear
  exactly once in the switch. `selectCandidates` keys candidates on
  `pkg::name::dir` (`run.go:271`); no existing candidate shares
  `cash::ShopOperationBuyOtherPackage::Serverbound`, so this cannot silently
  shadow or be shadowed by another FName under the first-claim dedup rule
  documented at `run.go:238-253`.
- **`--check` exit-code semantics unaffected**: the switch statement is
  reached only from `process()`/`selectCandidates()` (`run.go:161-168`); the
  exit-code mapping (`worstVerdict` → 0/1/2/3, `run.go:172-179`) is untouched
  by this diff. A new mapping can only add a packet to the audited set — it
  cannot change how an existing packet's verdict severity maps to an exit
  code.
- **False-PASS potential**: none identified. The new mapping does not weaken
  any existing comparison; it registers a new candidate for a function that
  had no prior mapping, so before this change `CCashShop::OnGiftPackage` was
  simply never audited (an "unlisted/orphan" cell, exactly as commit
  `02ab37435`'s message states). Confirmed the Atlas struct the new
  candidate resolves to actually exists:
  `libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package.go`
  — so `locateAtlasFile` cannot silently no-op the audit for lack of a target
  file.
- **Build**: `go build ./tools/packet-audit/...` — clean, no errors.

### Verdict: PASS, no findings

This is a minimal, additive, well-scoped change that follows the established
pattern of every sibling `case` arm in the same switch. It does not touch
`selectCandidates`, `orderedExportFNames`, verdict-severity thresholds, or
the `--check` exit path. No mechanism by which this diff could produce a
false PASS elsewhere in the matrix was found.

---

## Artifact 2 — hand-spliced `gms_v95.json` entry @ `02ab37435`

### What changed

`git show --stat 02ab37435` — six files: `docs/packets/audits/STATUS.md`,
`docs/packets/audits/status.json`, two new files under
`docs/packets/audits/gms_v95/CashShopOperationBuyOtherPackage.{json,md}`, one
new evidence yaml
(`docs/packets/evidence/gms_v95/cash.serverbound.CashShopOperationBuyOtherPackage.yaml`),
and `docs/packets/ida-exports/gms_v95.json` (+26/-0). **Only one key** was
added to `gms_v95.json` — `git show 02ab37435 -- docs/packets/ida-exports/gms_v95.json`
shows a pure insertion, no lines removed, and no other file under
`docs/packets/ida-exports/` touched.

### Internal shape consistency

Compared the new `"CCashShop::OnGiftPackage"` entry against sibling
`CCashShop::On*` entries in the same file (`OnBuyPackage` @ line 2849,
`OnBuyCouple` @ line 4269, plus ~20 more `CCashShop::On*` entries found via
grep). Field names (`address`, `direction`, `calls[].op`/`.comment`/`.guard`)
and shapes match exactly; the `guard` field's free-text-expression format
matches other conditional entries (e.g. line 200, 230).

One thing that initially looked like a defect and turned out not to be:
sibling `OnBuyPackage` (also `direction: "serverbound"`) uses `"op":
"Encode1"/"Encode4"`, while the new `OnGiftPackage` entry uses `"op":
"DecodeStr"/"Decode4"` for the same directional role. Traced this through
`tools/packet-audit/internal/idasrc/idasrc.go:13-45` (`Primitive.RawOp()`)
and `.../parse.go:250-266` (`opName`): the **current** parser always emits
the canonical `Decode*` op-string regardless of direction (a naming
convention, not a literal source-call echo), and
`tools/packet-audit/internal/idasrc/export.go:328-341` (`parsePrim`)
explicitly aliases `"Decode4"`/`"Encode4"` etc. to the same `Primitive` value
on read. So the two naming styles coexist in the file (553 `Encode*` vs 2146
`Decode*` occurrences — a legacy-vs-current-parser split across the whole
export, not something this commit introduced) and are functionally
identical to every downstream consumer. Not a defect.

### Cross-check against the claimed derivation

The commit message and the JSON `comment` fields assert two things: (1) an
inlined `nCommSN` write was missed by the regex parser and had to be added
by hand, and (2) a "mode byte" write was also missed. Only the `nCommSN`
write appears as a row in the new export entry; no mode-byte row exists.
That is *not* an inconsistency once cross-referenced against
`docs/tasks/task-240-cash-shop-stub-operations/derivation.md:208-262` (§D3a):
the "mode byte" is `a[v46++] = 33` at `0x490b74` — the outer CashShop
opcode-dispatch tag that routes into `OnGiftPackage` in the first place, the
same role played by `OnBuyPackage`'s `Encode1(0x20)` and `OnBuyCouple`'s
`Encode1(0x1E)`. `OnBuyPackage`'s sibling export entry (line 2849) likewise
omits this tag byte entirely and is independently marked ✅-verified in
STATUS.md, so omitting it here is consistent with the established
convention for this whole family of cash-shop sub-op functions, not a
missing field. (`OnBuyCouple`'s export *does* include its tag byte as row 0,
but the generated audit report for that packet — verified by reading
`docs/packets/audits/gms_v95/CashShopOperationBuyCouple.json` — drops it
from the compared `Rows` entirely via the tool's branch/dispatch-detection
logic; either convention produces the same effective N-field comparison.)

The Atlas struct `ShopOperationBuyOtherPackage`
(`libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package.go`)
has exactly the 4 fields the new export lists in the same order: `spw
string`, `serialNumber uint32`, `name string`, `message string`.

### Reproducibility (the load-bearing check)

Rather than trust the generated-file diff by inspection, regenerated both
artifacts from the committed export and confirmed byte-identical output:

- `go run ./tools/packet-audit matrix --check` → **exit 0** (matches the
  commit message's claim).
- `go run ./tools/packet-audit matrix` (full regeneration) → `git status
  --short` on `docs/packets/audits/STATUS.md` / `status.json` is empty; the
  committed matrix state is not hand-tweaked drift, it's exactly what the
  tool produces from the corrected export.
- Full report regeneration: `go run ./tools/packet-audit --csv-clientbound
  "docs/packets/MapleStory Ops - ClientBound.csv" --csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv"
  --template services/atlas-configurations/seed-data/templates/template_gms_95_1.json
  --atlas-packet libs/atlas-packet --ida-source docs/packets/ida-exports/gms_v95.json
  --output /tmp/pa-out` (the pipeline exits 1 overall, but only on two
  *unrelated* pre-existing packets — `SummonBagItemUse` and
  `ReturnScrollItemUse`, both "method Encode not found," nothing to do with
  cash shop). `diff /tmp/pa-out/gms_v95/CashShopOperationBuyOtherPackage.{json,md}
  docs/packets/audits/gms_v95/CashShopOperationBuyOtherPackage.{json,md}` —
  **byte-identical**. The committed audit report was genuinely produced by
  the deterministic pipeline off the corrected export, not fabricated by
  hand to look plausible.

### Test honesty

`docs/packets/evidence/.../cash.serverbound.CashShopOperationBuyOtherPackage.yaml`
cites two tests in `shop_operation_buy_other_package_test.go`:
`TestShopOperationBuyOtherPackageRoundTrip` and
`TestShopOperationBuyOtherPackageV95Bytes`. Read both — the byte-fixture
test pins the literal wire encoding
(`"040041424344" + "08070605" + "0300426f62" + "02004869"`) against the
exact field sequence/types the derivation and export claim (`spw` string,
`serialNumber` uint32 LE, `name` string, `message` string, no mode byte, no
`pointType`/`option`). `go test ./libs/atlas-packet/cash/serverbound/...
-run ShopOperationBuyOtherPackage -v` — both tests **PASS**. The byte test
is not a same-code-both-ways round-trip; it would fail if any field were
reordered or mistyped, which is exactly what it's there to pin.

### What I could not evaluate

I do not have IDA-Pro / MCP access in this environment, so I cannot
independently re-derive the address `0x4907b0`, confirm the mangled-name
lookup (`?OnGiftPackage@CCashShop@@QAEXJ@Z`) against the live `gms_v95` IDB,
or verify the claim that `COutPacket::m_aSendBuff` sits at offset `0x4` (the
basis for treating the inlined array writes as real packet-buffer writes and
not a scratch local). This is the one load-bearing fact the whole artifact
rests on that repo evidence alone cannot confirm or refute — it is asserted
in the commit message and in `derivation.md` §D3a with a decompile excerpt,
but the excerpt itself is prose transcribed by the implementing agent, not a
stored/hashed artifact I can check independently (the yaml's
`decompile_sha256` has nothing in the repo to hash against, since the
decompile was fed through a throwaway, uncommitted test per the commit
message).

### Verdict: APPROVED_WITH_FINDINGS (non-blocking)

Every check the repo can support was performed and passed: internal
consistency with sibling entries, consistency with the family's established
"mode byte is not part of the per-op body" convention, exact reproducibility
of the derived report/matrix/STATUS files from the hand-edited export, and a
byte-fixture test that would fail on a field-order or field-count defect and
currently passes. Nothing in the repo evidence contradicts the artifact.

The one gap is structural, not a defect I can point at: the address/offset
claim that makes the whole entry legitimate (rather than resting on a
plausible-looking but wrong reverse-engineering guess) is not independently
checkable from this environment. That is disclosed here rather than folded
into an approval, per the review brief's instruction to say so plainly
rather than guess.

---

## Compact verdict

```text
verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-240-cash-shop-stub-operations/review-tail-artifacts.md
scope_confirmed: reviewed exactly tools/packet-audit/cmd/run.go @ 71fc76306 and the
  gms_v95.json export entry (+ its generated audit report / STATUS.md / status.json /
  evidence yaml) @ 02ab37435; no other file in either commit was treated as in-scope
blocking: 0
non_blocking: 1
  - docs/packets/ida-exports/gms_v95.json:2867-2892 (evidence yaml decompile_sha256) —
    the IDA address (0x4907b0) and the m_aSendBuff-offset-0x4 claim that justifies
    treating the two inlined writes as real packet-buffer writes cannot be
    independently re-derived without IDA-Pro/MCP access; every repo-checkable
    consequence of the claim (report regen, matrix regen, byte-fixture test) is
    consistent and passing, but the root fact itself is asserted, not verifiable here
not_evaluable: 1
  - live gms_v95 IDB confirmation of 0x4907b0 / mangled-name lookup / m_aSendBuff
    offset — requires IDA-Pro tooling not available in this review environment
```
