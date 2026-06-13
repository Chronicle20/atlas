# Stage 0.5 — Tooling unblock (export refresh mechanism)

Discovered during execution: the 42 MOB/MONSTER functions are present in every IDB
but only as **mangled MSVC symbols** (`?OnAffected@CMob@@QAEXAAVCInPacket@@@Z`). Two
consequences and their fixes (committed `ca54ef850`):

## 1. Finding functions in IDA (the recurring trap)
- `lookup_funcs("CMob::OnAffected")` and even bare `lookup_funcs("OnAffected")` → **"Not found"** (exact match on the stored mangled name).
- **Correct lookup:** `func_query {queries:[{name_regex:"OnAffected@CMob", sort_by:"addr"}]}` → returns `addr`; then `decompile`/operate **by address**. Confirmed identical in v83 + v95.
- Memory `reference_ida_mcp_new_api` updated with a loud "read first" banner.

## 2. Export refresh (was the hard blocker)
`evidence pin` needs each op's fname present in `docs/packets/ida-exports/<version>.json`; none of the 42 were exported by task-085, and the `export` harvester resolved roster fnames through the same failing demangled lookup.

Fixes landed in `tools/packet-audit`:
- `export`: new optional `--prior-export` / `--pending` flags. Pass `--prior-export ""` + `--pending <roster>` to harvest a **targeted** roster (only the listed fnames) WITHOUT re-harvesting the existing ~306 records. Default behaviour unchanged when flags absent.
- `internal/idasrc` `GetFunctionByName`: falls back to `func_query name_regex:"^\?Method@Class@@"` when the exact `lookup_funcs` match misses, so demangled `Class::Method` roster names resolve. Live-verified: `export --version gms_v83 --prior-export "" --pending <3 fnames>` → `3 resolved, 0 unresolved`.

## Export-refresh procedure (per version, run during Stage 1)
1. Build a roster file with the version's applicable MOB fnames (demangled `Class::Method`, one per line), using the IDB-confirmed correct fnames (fix registry mislabels first).
2. Harvest to a temp file:
   ```
   go run ./tools/packet-audit export --version <vk> --output /tmp/<vk>-mob.json \
     --prior-export "" --pending <roster> \
     --ida-url http://192.168.20.3:13337/mcp --ida-port <port> --ida-timeout 60s
   ```
   Ports: v83=13337 v87=13338 v95=13339 jms=13340 v84=13341 (confirm via `list_instances`; ports are not stable across days).
3. **Merge absent-keys-only** into the real export (existing records stay byte-identical → no hash drift for other families):
   ```
   jq -s '.[0] as $real | .[1] as $new
     | $real | .functions = ($real.functions + ($new.functions | with_entries(select($real.functions[.key]==null))))' \
     docs/packets/ida-exports/<vk>.json /tmp/<vk>-mob.json > /tmp/<vk>-merged.json
   mv /tmp/<vk>-merged.json docs/packets/ida-exports/<vk>.json
   ```
4. After all 5 versions: commit the refreshed exports as a discrete step; run `go run ./tools/packet-audit matrix --check` and confirm exit 0 (no drift/stale/conflict) BEFORE any Stage-2 evidence pinning.

> The export `generated_at` differs per run; the merge preserves the real export's top-level `generated_at`/`binary`/`md5` (jq takes `$real` as the base), so only `functions` gains the new keys.

## Registry fname corrections (broader than plan assumed — from Stage 0.3)
- v83: MOB_SPEAKING→(real `OnMobSpeaking`), INC_MOB_CHARGE_COUNT→(real `OnIncMobChargeCount`) — two-way swap.
- v87: three-way rotation — MOB_SPEAKING, INC_MOB_CHARGE_COUNT, MOB_SKILL_DELAY all wrong; un-rotate all three.
- v84, v95: already correct — do NOT touch.
- MONSTER_BOOK_COVER: `fname:""` in ALL 5 — derive send-site per version.
- MOB_ESCORT_RETURN_STOP/_STOP_SAY: absent; v95 has `MOB_ESCORT_STOP`(305)/`MOB_ESCORT_STOP_SAY`(306) — reconcile names.
- Confirm every corrected fname against the IDB via `func_query` before editing the registry row (`provenance: manual`, IDA address in `note`).
