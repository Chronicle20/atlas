# Review: bug-maple-life-v84-registration

Commit range: `903537fe8..HEAD` (`2a34e5e69` diagnosis, `7f1dc0a43` fix, `2e8ca1bc4` report).
Requirement: `docs/tasks/task-246-maple-life-character-creation/bug-maple-life-v84-registration.md`.

## Scope reviewed

`git diff --stat 903537fe8..HEAD`:

```
docs/packets/registry/gms_v84.yaml                                       |  24 +
.../bug-maple-life-v84-registration-report.md                            | 171 ++++++
.../bug-maple-life-v84-registration.md                                   | 206 +++++++
derivation.md                                                            |  75 +++
.../atlas.com/configurations/socket/corpus_test.go                       |   4 +-
.../seed-data/templates/template_gms_84_1.json                          | 655 +++++++++++++++++++++
6 files changed, 1133 insertions(+), 2 deletions(-)
```

Matches the report's file list plus the two bug-tracking docs and the report
itself. No files touched outside this set. `libs/atlas-packet/maplelife/**`
was checked (per the brief's fifth bullet) and correctly left untouched
because no version gate exists there — verified below.

## 1. Opcode fidelity — PASS

- `template_gms_84_1.json` handler: `opCode: "0x107"`, `handler:
  MapleLifeCheckNameHandle`, `fname: CUICharacterSaleDlg::SendCheckDuplicateIDPacket`
  (`services/atlas-configurations/seed-data/templates/template_gms_84_1.json:2190-2197`
  in the post-change file).
- Writers: `opCode: "0x167"` → `MapleLifeResult`, `opCode: "0x168"` →
  `MapleLifeError`, both `services: [channel]`.
- Registry `docs/packets/registry/gms_v84.yaml`: `MAPLELIFE_CHECK_NAME`
  serverbound 263 (`0x107` == 263, decimal-hex agree), `MAPLELIFE_RESULT`
  clientbound 359 (`0x167`), `MAPLELIFE_ERROR` clientbound 360 (`0x168`).
  `ida.address` decimal values (8378474 / 8378697 / 8378991) convert exactly
  to `0x7fd86a` / `0x7fd949` / `0x7fda6f` — checked by hand.
- `grep -n "349\|350" <diff>` finds the two numbers only inside prose that
  explicitly retracts them ("that prediction was wrong — the dispatcher says
  359/360") in the bug file and in the new `derivation.md` correction. They
  are never used as a registered opcode value anywhere in the diff.
- No new opcode collisions: cross-checked the full registry's
  `(direction, opcode)` pairs after the edit — the only duplicate present is
  a pre-existing `serverbound 236` collision (COUPON_CODE / OPEN_ITEMUI),
  unrelated to this change, matching the implementer's own claim.

## 2. Operations tables — PASS

`template_gms_84_1.json`'s new writer entries:
- `MapleLifeResult`: `AVAILABLE:0, TAKEN:1, UNKNOWN_ERROR:255`
- `MapleLifeError`: `SUCCESS:52, NAME_TAKEN_AT_SUBMIT:54, UNKNOWN_ERROR:255`

Diffed against `template_gms_83_1.json`'s `0x15D`/`0x15E` entries
(`grep -n -A10 '"0x15D"'` / `'"0x15E"'`) — tables are byte-for-byte identical
in key order, values, and `fname`s. This is the v83 table, not v87's
54/56 — confirmed by direct read of both blocks.

## 3. `corpus_test.go` 3384 → 3387 — PASS

`TestValidate_AcceptsEverySeedTemplate` computes `total` by walking every
`template_*.json` under `seed-data/templates` and summing
`len(handlers)+len(writers)` — a real count, not a mock or a
hand-maintained tally. The diff adds exactly 1 handler
(`MapleLifeCheckNameHandle`) + 2 writers (`MapleLifeResult`,
`MapleLifeError`) to `template_gms_84_1.json`, i.e. +3, matching
3384 → 3387 arithmetically.

Ran the test directly (`go test ./atlas.com/configurations/socket/... -run
TestValidate -v` from `services/atlas-configurations`): `PASS`. The
assertion is not weakened — it is still an exact-equality check on a
narrative-annotated total, same shape as before, with the count and the
new narrative clause both correctly reflecting the added bindings. This is
a genuinely stale assertion the seed-data edit forced, not a masked defect.

## 4. `mapleLife` block — PASS

Extracted the `"mapleLife": { ... }` object (brace-matched, raw bytes) from
`template_gms_83_1.json`, `_87_1`, `_92_1`, `_95_1`, and the new
`_84_1`. All five are byte-identical (12,981 bytes each, `==` true across
all pairs checked). This confirms the implementer's claim of a verbatim,
byte-identical copy — not paraphrased or independently authored. Since the
four already-shipped confirmed builds (83/87/92/95) all carry the exact
same block, gms_84 (bracketed between 83 and 87) carrying the same block is
internally coherent, not a blind duplication of unrelated data — consistent
with the brief's "model it on gms_83_1's block... if the two templates'
data diverge in a way that makes a straight copy wrong, stop and report."
No divergence exists to report.

## 5. Structural consistency — PASS

- Handler insertion point: between the entries at `0x104` (`ItcOperationHandle`)
  and `0x10B` (`ItemUpgradeUpdateHandle`) in `template_gms_84_1.json` — the
  same relative position `0x100` occupies between `ItcOperationHandle` and
  `ItemUpgradeUpdateHandle` in `template_gms_83_1.json`.
- Writer key order (`opCode`, `writer`, `fname`, `options.operations`,
  `services`) matches `template_gms_83_1.json`'s `0x15D`/`0x15E` entries
  exactly, field for field.
- No CR bytes introduced (`git diff ... | grep -c $'\r'` → 0); the added
  lines carry the same line endings as the rest of the file.

## Additional checks

- `go build ./...` and `go test ./...` from
  `services/atlas-configurations/atlas.com/configurations`: all green,
  including `socket`, `templates`, and `tenants/maplelife`.
- JSON validity of the modified template confirmed via `json.load`.
- `derivation.md`: the new `§2.0-CORRECTION` is a pure append after `§6.4`;
  `§2.0` itself is untouched, honoring the file's "do not renumber" header.
- Fifth brief bullet (`libs/atlas-packet/maplelife/**` version-gate check):
  confirmed by direct read of `clientbound/error.go` and by the
  implementer's documented `grep` — there is no `MajorAtLeast`/version gate
  in the codec Encode/Decode paths; the file's own comment states "There is
  deliberately NO version gate in Encode/Decode." Per the brief's own
  fallback ("if the codecs are version-agnostic, change nothing"), no edit
  here was correct.

## Non-blocking findings

1. **Stale "VERSION-ABSENT" comments left in `libs/atlas-packet/maplelife/`.**
   `clientbound/error.go:69,82`, `clientbound/result.go:61,73`,
   `serverbound/check_name.go:28`, and the three `*_test.go` files'
   "FOUR in-scope cells... gms_v84 is [out-of-scope]" comments still assert
   gms_v84 is VERSION-ABSENT / out of scope — a claim this same commit range
   retracts in `derivation.md`. The brief's fifth bullet only asked whether
   a version *gate* excludes v84 (it does not, correctly left unchanged),
   not whether the surrounding comments are accurate; and adding v84 test
   coverage there is explicitly packet-verifier work per the brief's last
   bullet. Still, these comments now contradict the retracted finding and
   will mislead the next reader until the packet-verifier pass touches
   them. Worth a follow-up note, not a blocker for this fix.
2. **`## Resolution` section of the bug file left unfilled**, as the
   implementer flagged. Reasonable given it depends on the repo-wide gate
   verdict, which is a separate step from this review.

## Not evaluable

- The underlying IDA-session claims (dispatcher decompile at `0x7fd845`,
  the swapped `CField` symbols at `0x5443af`/`0x544395`, task-129's "routes
  359/360" annotation) cannot be independently re-derived from within this
  repo; they are asserted evidence from a live IDA session not reproducible
  here. Per the task ruling, evidence records under
  `docs/packets/evidence/gms_v84/` are explicitly out of scope for this
  review (packet-verifier's job) — the derivation's internal consistency
  (opcode math, cross-references, retraction of 349/350) was checked, but
  the raw disassembly cannot be.
- jms_v185 is explicitly out of scope per the task ruling and was not
  reviewed.

## Verdict

APPROVED_WITH_FINDINGS — no blocking defects found; two non-blocking notes
above (stale doc comments; unfilled Resolution section, both expected to be
closed by later passes).
