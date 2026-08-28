# Task 7 Fix Round 1 Review

**Commit:** `b19fe1a92` — "fix(packets): commit regenerated matrix status + correct MAKER_RESULT address"

**Scope:** Re-review of two specific findings from the original Task 7 review.

## Finding 1: Blocking — STATUS.md/status.json stale toolSha (ADDRESSED)

**Original issue:** `matrix --check` exited 1 because STATUS.md and status.json carried stale `toolSha` hashes from before commit `ed190de3a` edited `tools/packet-audit/cmd/run.go`.

**Verification performed:**

1. **Commit composition** (`git show --stat b19fe1a92`): Files are in the commit.
   ```
   docs/packets/audits/STATUS.md                               | 2 +-
   docs/packets/audits/status.json                             | 2 +-
   ```

2. **Diff scope** (git show b19fe1a92): Only the `toolSha` field changed; no op rows altered.
   - STATUS.md: `ff53f81435f596f5a55959c1d8a24fd2edad0c02ede677be6fc0f64d90dcf20b` → `00ce36017e9bb01df15f9aaba0db1f9e6f90a5e8d70d36e70e3ded56bffae768`
   - status.json: Identical hash change
   - No changes to any operation rows, states, or notes

3. **MAKER_SKILL integrity** (grep "MAKER_SKILL" docs/packets/audits/STATUS.md): Row correctly remains `❌ incomplete` on all eight versions.
   ```
   ❌ | 0x06F | ❌ | 0x071 | ❌ | 0x071 | ❌ | 0x074 | ❌ | 0x07C | ❌ | 0x07D | ❌ | 0x06C | ❌
   ```
   This is correct — CUIItemMaker::RequestItemMake still does not resolve in the ida-exports functions maps, so the evidence-pin cannot land yet.

4. **Matrix check exit code** (`go run ./tools/packet-audit matrix --check`): Returns 0 (success).
   ```
   note	n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 (docs/packets/feature-na-evidence.yaml)
   note	n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (docs/packets/feature-na-evidence.yaml)
   ```
   No errors; gate passes.

**Verdict:** PASS — Blocking finding addressed. Tool regeneration was correctly committed with only the toolSha hash changed, no op state corruption.

---

## Finding 2: Non-blocking — wire-derivation.md address citations (ADDRESSED)

**Original issue:** `wire-derivation.md:492,503` cited the char-narrowed `Decode4` site as `0x86a1ce`, which disassembles to `mov esi, eax` (an assignment, not the call site). The implementer confirmed via IDA that the actual call site is `0x86a1c0`.

**Verification performed:**

1. **Citation correction in table** (line 491, gms_v72 row):
   - Before: `| `gms_v72` | `0x86a1bd` | `0x86a1ce` | ...`
   - After: `| `gms_v72` | `0x86a1bd` | `0x86a1c0` | ...`

2. **Citation correction in note text** (line 501):
   - Before: "at `gms_v72` `0x86a1ce`, and similarly at..."
   - After: "at `gms_v72` `0x86a1c0`, and similarly at..."

3. **Claim integrity:** The surrounding explanation remains unchanged in substance:
   > "That is a Hex-Rays register-width inference on an unused high byte, not a wire width — the callee is `?Decode4@CInPacket@@QAEKXZ` in every case, so **4 bytes are consumed**. Do not narrow any of these to a byte in the codec."

**Verdict:** PASS — Non-blocking finding addressed. Both citations corrected from the store address (`0x86a1ce`, `mov esi, eax`) to the actual Decode4 call site (`0x86a1c0`). The wire-width claim (4 bytes, do not narrow) remains accurate and unchanged.

---

## Codec files

Verified that `maker_skill.go` and `maker_skill_test.go` are untouched in this commit. No changes to the previously-verified codec implementation.

---

## Summary

All findings are addressed. The commit correctly:
- Regenerates and commits the stale STATUS.md/status.json with the new toolSha hash
- Makes no changes to any operation state or MAKER_SKILL completion status
- Corrects both citations in wire-derivation.md from the store address to the actual Decode4 call site
- Passes the matrix gate

**Result:** ADDRESSED (0 blocking findings, 0 non-blocking findings)
