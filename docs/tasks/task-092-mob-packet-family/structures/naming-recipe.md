# IDB function-naming recipe (validated 2026-06-13)

For ops whose function is present but UNNAMED (`sub_XXXX`) in an IDB, so the export
harvester can't resolve them. Proven on v84 `CMob::OnSuspendReset` (0x682802):
rename → re-harvest gave `1 resolved, 0 unresolved`.

## Recipe
1. **Locate the candidate sub.**
   - Clientbound handlers: read the cluster dispatcher and take the case's target sub.
     - v84 mob cluster: `CMobPool::OnMobPacket @0x68FEF7` (switch on a normalized index — case numbers are OFFSET from raw opcodes; do NOT assume case == registry opcode).
     - Carnival: `CField_MonsterCarnival::OnPacket` dispatcher.
     - CWvsContext ops (book card/cover, taming, bridle): the CWvsContext packet dispatcher.
   - Serverbound senders (TryDoingBodyAttack, SendBanMapByMobRequest, UpdateTimeBomb,
     SendCollisionEscort): not in a dispatcher — find via xref from a known caller, or by
     locating the COutPacket build-site that writes the op's opcode, or by structural match
     to the named sibling-version function (these are named in v95/v87).
2. **Confirm identity by LAYOUT MATCH, not by opcode.** Decompile the candidate; its
   `CInPacket::Decode*` (handlers) / `COutPacket::Encode*` (senders) sequence must match the
   op's v83 named twin in `structures/gms_v83.md`. v84 ≡ v83 byte-wise, so the Decode
   sequence is identical. If the layout does not match, it's the wrong sub — do NOT rename.
3. **Rename to the mangled MSVC symbol** via `mcp__ida-pro__rename {batch:{func:{addr,name}}}`
   (dry_run first). Handler signature is uniform:
   `?<Method>@<Class>@@QAEXAAVCInPacket@@@Z` (public __thiscall void(CInPacket&)), e.g.
   `?OnAffected@CMob@@QAEXAAVCInPacket@@@Z`. Senders have their own signatures — copy the
   exact mangled name from the named sibling-version IDB (func_query there) when possible.
4. **Re-harvest** the targeted export for the now-named fname(s) and **merge absent-keys-only**
   into the version export (commands in `tooling-unblock.md`). Re-generate the version's
   `structures/<vk>.md` rows.
5. `matrix --check` must stay exit 0. Commit per version.

## Rule
Never guess-rename. If a function's layout doesn't confidently match the expected op, leave
it unnamed and report it as unresolved — an honest gap beats a mislabelled IDB.
