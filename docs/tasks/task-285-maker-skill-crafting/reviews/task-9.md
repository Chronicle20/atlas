# Review — Task 9: `MAKER_RESULT` dispatcher YAML, operations tables, byte fixtures, matrix promotion

**Commit under review:** `ab31624bd` (35 files, +1422/-37)
**Brief:** `.superpowers/sdd/plan/task-9-brief.md` (+ CONTROLLER OVERRIDES)
**Report:** `.superpowers/sdd/plan/task-9-report.md`
**Binding ruling:** `docs/tasks/task-285-maker-skill-crafting/ruling-failed-arm.md`

## Verdict

**CHANGES_REQUIRED.**

The row's promotion to `✅` on all eight versions does **not** mean all five arms
are evidence-backed. It means only `MakerResultCreate` (1 of 4 non-degenerate
mode arms) carries a `packet-audit:verify` marker + pinned evidence. The
implementer's own cited precedent (`CWvsContext::OnClaimResult`) does not
actually match the shape here, and by the matrix's own documented standard
(`docs/packets/evidence/families.yaml`), this op should currently be capped at
`🧩` (family), not `✅`. This is exactly the kind of gap this branch has already
been bitten by once (a green matrix that cannot see a partial rollout).

## Central question — is the ✅ real or manufactured?

### 1. What `registryDeclaresPacket` actually exempts

`tools/packet-audit/cmd/matrix.go:388-401`:

```go
// registryDeclaresPacket reports whether the given version's registry has any op
// whose `packet:` field equals pkt. Such an op is the no-report byte-fixture
// promotion path (commit 6c202cb7): its evidence record is intentionally
// report-less and must not be flagged as dangling by --check.
func registryDeclaresPacket(reg opregistry.Registry, version, pkt string) bool {
	vf, ok := reg.Versions[version]
	...
	for _, e := range vf.Entries {
		if e.Packet == pkt {
			return true
		}
```

A registry op entry carries **exactly one** `packet:` field (confirmed:
`docs/packets/registry/gms_v72.yaml:1411` etc. carry a single
`packet: character/clientbound/MakerResultCreate` line). Any evidence record
whose `pkt` is one of the other four arm names (`MakerResultCreateWithUpgrade`,
`MakerResultMonsterCrystal`, `MakerResultDisassemble`, `MakerResultFailed`)
would find no matching registry entry → `registryDeclaresPacket` returns
`false` → `--check` reports it as dangling evidence. This confirms the
implementer's claim mechanically: pinning all 5 arms × 8 versions would
produce 32 dangling-evidence failures given the current one-`packet:`-per-op
registry schema. **Confirmed.**

### 2. Does the `claim_result_test.go` / `ClaimResultNotice` precedent genuinely have the claimed shape?

Read `libs/atlas-packet/report/clientbound/claim_result_test.go:32-99`.
`ClaimResultSuccess` (mode 2) is the **only** mode that reads anything beyond
the mode byte. Every other reachable mode value (`0x03, 0x41-0x45, 0x47, 0x48`)
is a **bare mode byte with no further packet reads** — the test's own comment
at line 68 states this explicitly ("none of these branches perform any further
CInPacket read"), and `TestClaimResultNoticeByteOutputV72` asserts all eight of
those values against the *same* one-byte expectation.

So `ClaimResult`'s dispatcher has, in substance, **one** distinct body shape
(`Success`) plus a uniform degenerate shape (`Notice`, identical across every
one of its mode values). That is a fundamentally different situation from
`MakerResult`, whose un-pinned siblings — `CreateWithUpgrade` (materials/gem/
catalyst/meso, byte-identical to `Create` except the mode field, per
`maker_result_test.go:301-317`), `MonsterCrystal` (16-byte crystal/leftover
pair, no meso field), and `Disassemble` (item + reward-stack loop + meso) — are
**three additional, structurally distinct, non-degenerate bodies**, none of
which carries a marker or pinned evidence record. Only `Failed` (bodyless,
4 bytes, `nResult` only) is genuinely analogous to `ClaimResultNotice`.

**The precedent as cited is a misread, not a match.** `ClaimResult` legitimately
has only one non-trivial arm; `MakerResult` has four.

### 3. Does ✅ mean all five arms are evidence-backed?

**No — it means only `MakerResultCreate` is.** And this is a gap the matrix
cannot see given the current state of `docs/packets/evidence/families.yaml`,
not the house standard. Evidence:

- `docs/packets/evidence/families.yaml:1-20` defines the matrix's own policy
  for exactly this shape: "a single opcode carries N logically distinct
  sub-packets selected by a leading mode/discriminator byte, and each arm
  reads a different body... The matrix therefore CAPS any op whose registry
  fname is one of these at the 🧩 `family` state; it can never reach ✅ on
  one sub-handler."
- `docs/tasks/task-285-maker-skill-crafting/wire-derivation.md:138,212-213`
  confirms `CUserLocal::OnMakerResult` compiles to exactly this shape: a
  `switch (nMode) { case 1: case 2: ...; case 3: ...; case 4: ...; default }`
  dispatching to per-mode bodies.
- Every entry currently documented in `families.yaml` (CashShop, CITC,
  CShopDlg, CUIMessenger, CTrunkDlg, CRPSGameDlg) required **all** applicable
  mode arms to carry a `packet-audit:verify` byte-fixture + fresh pinned
  evidence before the entry was removed and the row allowed to reach ✅ (the
  "GRADUATED" comments spell this out arm-count by arm-count, e.g. "all 57
  mode arms verified... task-183 added 48 new mode-arm codecs").
- `tools/packet-audit/internal/matrix/grade.go:60-67,169-195` documents the
  same rule in code comments: "a mode-prefix dispatcher is NEVER lifted to ✅
  by per-version mode-byte enumeration alone... A dispatcher stays capped at
  🧩 (StateFamily) until every mode arm Atlas supports has an IMPLEMENTED +
  byte-fixture-VERIFIED body."
- `CUserLocal::OnMakerResult` was **never added** to `families.yaml` by this
  commit or any prior one (confirmed: no hits for `MakerResult`/`OnMakerResult`
  in the file, and `git diff --stat ab31624bd^ ab31624bd -- docs/packets/evidence/families.yaml`
  shows no change to that file at all).

Because `in.Families[baseFName(ref.FName)]` (`tools/packet-audit/internal/matrix/grade.go:145,192-196,225-227,255-258`)
is empty for this fname, `gradeCore` never applies the family cap and instead
takes the ordinary "marker + fresh evidence promotes" path on the single
registry-declared arm — reaching `StateVerified` (✅) on evidence that, by the
matrix's own stated design intent for this exact packet shape, should not be
sufficient. This is a real gap: either `CUserLocal::OnMakerResult` should have
been entered in `families.yaml` (capping the row at 🧩 pending full-arm
coverage, matching every other documented multi-arm dispatcher), or the three
non-degenerate un-pinned arms need their own promotion path. Neither happened;
the report's "same single-declared-arm shape as `ClaimResult`" framing papers
over the difference. **Blocking.**

## IDA export splices — verified

- **Additions only:** `git diff --numstat ab31624bd^ ab31624bd -- docs/packets/ida-exports/`
  → `106  0` for each of the six touched files (`gms_v83/84/87/92/95`,
  `gms_jms_185`). No deletions, no pre-existing entry displaced.
- **JSON well-formed:** all six parse cleanly (`python3 -c "import json; json.load(...)"`,
  confirmed individually).
- **Addresses/sizes match `wire-derivation.md` §4** exactly:
  `gms_v83` 0x95dad3/0x6df, `gms_v84` 0x99bdbc/0x6df, `gms_v87` 0x9e01b2/0x634,
  `gms_v92` 0x8f5d70/0x8a0, `gms_v95` 0x9102f0/0x8a0, `gms_jms_185` 0xa29527/0x633
  — all confirmed against both `wire-derivation.md:51-56` and the spliced
  entries' `address`/`notes` fields.
- No fabricated-looking hash or substituted fname spotted; the `decompile_sha256`
  values in the new evidence records are well-formed 64-hex-char strings.
  **Not evaluable** (no `ida-pro` access — cannot independently verify a
  decompiled read-order or hash against the client binary).

## Override compliance

- **Override 1 (no `FAILED` key):** `docs/packets/dispatchers/maker_result.yaml`
  carries only `CREATE`, `CREATE_WITH_UPGRADE`, `MONSTER_CRYSTAL`, `DISASSEMBLE`,
  with the required comment citing `guild.Info`/`GuildInfoBody` in place of the
  key. `MakerResultFailedBody` was not touched by this commit (confirmed no
  `character/maker_result_body.go` in the diff's file list). **Honoured.**
- **Override 2 (all five arms fixtured + round-trip):** Confirmed in
  `libs/atlas-packet/character/clientbound/maker_result_test.go` — one test per
  arm (`TestMakerResultCreateByteOutput`, `...CreateWithUpgradeByteOutput`,
  `...MonsterCrystalByteOutput`, `...DisassembleByteOutput`,
  `...FailedByteOutput`), each with an `Encode` fixture assertion and a
  `Decode`+re-`Encode` round-trip, ranged over all eight versions.
  `TestMakerResultFailedByteOutput` asserts `len(actual) == 4`
  (`maker_result_test.go:403-405`); `TestMakerResultMonsterCrystalByteOutput`
  asserts `len(actual) == 16` (`:338-340`). **Honoured**, independent of the
  marker-placement finding above (fixtures exist even where markers don't).
- **Override 3 (opcodes re-derived, not trusted from brief):** all eight
  registry opcodes (199/203/217/221/230/250/248/226 decimal) match the YAML's
  hex (`0xC7/0xCB/0xD9/0xDD/0xE6/0xFA/0xF8/0xE2`) exactly — confirmed by
  reading `docs/packets/registry/<version>.yaml` directly. **Honoured.**
- **Override 4 (toolSha trap):** `status.json`'s ida-export source hashes for
  the six touched exports changed in this commit (`git diff` confirmed) and
  are committed alongside `STATUS.md`; gates re-run against the committed tree
  (below) exit 0 without a rebuild-cache trap. **Honoured.**

## Fixture arithmetic — recomputed independently

Computed each little-endian 32-bit encoding from the stated decimal with
`struct.pack('<I', n)`:

| decimal | computed LE | fixture uses |
|---|---|---|
| 1082002 | `92 82 10 00` | `92 82 10 00` ✓ |
| 4011001 | `F9 33 3D 00` | `F9 33 3D 00` ✓ |
| 4011002 | `FA 33 3D 00` | `FA 33 3D 00` ✓ |
| 4000000 | `00 09 3D 00` | `00 09 3D 00` ✓ |
| 4000001 | `01 09 3D 00` | `01 09 3D 00` ✓ |
| 4021313 | `41 5C 3D 00` | `41 5C 3D 00` ✓ |
| 4130944 (brief's catalyst decimal) | `80 08 3F 00` (brief said `80 03 3F 00` — also wrong) | not used |
| 4130000 (implementer's corrected catalyst decimal) | `D0 04 3F 00` | `D0 04 3F 00` ✓ |

All three explicitly-claimed corrections (1082002, 4011001, 4000000) check out,
and the self-inconsistent catalyst pair is confirmed self-inconsistent (the
brief's own hex doesn't even match its own decimal) — the implementer's
substituted `4130000`/`D0 04 3F 00` is internally consistent.

**Non-blocking note:** the report's "brief had four wrong values" accounting
does not mention `4021313` (gem item id), but the brief's own text
(task-9-brief.md:133, `41 5F 3D 00 nItemID = 4021313`) is *also* wrong —
`4021313` = `0x3D5C41` = `41 5C 3D 00`, not `41 5F 3D 00`. The **code** is
correct (`maker_result_test.go:77,121` use `0x41, 0x5C, 0x3D, 0x00`), so this
is a documentation-accounting gap only, not a code defect.

Meso fields also verified: 1200 → `B0 04 00 00` ✓, 500 → `F4 01 00 00` ✓.

## Registry / seed template consistency

- All eight registry `packet:` lines read
  `packet: character/clientbound/MakerResultCreate` (confirmed for
  `gms_v72`/`gms_v92`, spot-checked; pattern consistent across the diff).
- All eight seed templates (`template_gms_{72,79,83,84,87,92,95}_1.json`,
  `template_jms_185_1.json`) gained exactly the same 4-key `operations` block
  (`CREATE:1, CREATE_WITH_UPGRADE:2, MONSTER_CRYSTAL:3, DISASSEMBLE:4`),
  confirmed by diff on `template_gms_83_1.json`. `template_gms_{12,48,61}_1.json`
  are untouched (`git diff --stat` shows exactly 8 template files changed).

## Gates re-run against the committed tree (this review's own run)

```
$ go run ./tools/packet-audit matrix --check
note  n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 (...)
note  n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (...)
EXIT=0

$ go run ./tools/packet-audit operations --check
operations check OK (0 absent-writer note(s))
EXIT=0

$ go run ./tools/packet-audit fname-doc --check
fname-doc check OK (277 structs without an audit report carry no fname)
EXIT=0

$ go run ./tools/packet-audit dispatcher-lint
dispatcher-lint: clean
EXIT=0
```

All four gates genuinely exit 0 against the committed tree (`git status` clean
at HEAD `6b589e743`, one docs-only commit atop `ab31624bd`). This does not
contradict the central finding above — none of these four gates couple YAML
operations keys to Go consumption or enforce the `families.yaml` membership
decision; a green gate here cannot detect the missing family-cap entry, which
is exactly the seam this review is for.

`docs/packets/audits/STATUS.md:334` confirms the `✅` on all eight applicable
versions and `⬜` on v48/v61, matching the report.

## Scope confirmation

Reviewed `ab31624bd` in full (dispatcher YAML, registry `packet:` links, six
ida-export splices, eight seed-template `operations` blocks, byte-fixture test
file, body-resolution test file, STATUS.md/status.json regeneration). No scope
mismatch — the commit matches the brief's file list and the report's
description of it.

## Findings summary

### Blocking
1. **`docs/packets/audits/STATUS.md:334` / `docs/packets/evidence/families.yaml`
   (no entry) / `tools/packet-audit/cmd/matrix.go:388-401`** — `MAKER_RESULT`
   is promoted to `✅` on all eight versions on the strength of one
   marker+evidence-backed arm (`MakerResultCreate`) out of four non-degenerate
   mode arms. `CUserLocal::OnMakerResult` matches `families.yaml`'s own stated
   criteria for a capped mode-byte dispatcher (confirmed switch shape in
   `wire-derivation.md:138,212-213`), and every other multi-arm dispatcher
   currently in that file required full-arm byte-fixture+evidence coverage
   before reaching ✅. The cited `CWvsContext::OnClaimResult` precedent does
   not hold up: its unmarked sibling (`ClaimResultNotice`) is a degenerate,
   structurally uniform shape across all its mode values
   (`claim_result_test.go:65-99`), unlike `MakerResult`'s three distinct
   unmarked bodies (`CreateWithUpgrade`, `MonsterCrystal`, `Disassemble`). The
   row should either be capped at 🧩 (add the fname to `families.yaml`) until
   all four mode-carrying arms are evidence-pinned, or a promotion mechanism
   that covers all four arms needs to be found — not asserted as done via a
   mismatched analogy.

### Non-blocking
1. **`.superpowers/sdd/plan/task-9-report.md` (accounting gap)** — the report
   claims "four wrong values" in the brief's fixture hex but omits a fifth:
   `4021313`'s brief hex (`41 5F 3D 00`) is also wrong (correct is
   `41 5C 3D 00`). The shipped code fixture is correct
   (`maker_result_test.go:77,121`); only the report's self-accounting is
   incomplete.

### Not evaluable
1. Whether the six spliced IDA export entries' Decode-site addresses and
   comments genuinely reflect each version's own decompilation — no
   `ida-pro` access to verify against the client binaries. Internal
   consistency (additions-only diff, JSON well-formedness, address/size
   agreement with `wire-derivation.md` §4, cross-file shape consistency) was
   checked and is clean.
2. Whether the `decompile_sha256` values in the six new evidence records are
   the genuine hash of each version's decompiled function body — same
   `ida-pro` limitation.
