# Recipe: fix one divergent clientbound writer (task-181)

Proven on SnowballState (`05d47c7`). Follow exactly. Work ONLY in the task
worktree (`<repo-root>/.worktrees/task-181-v79-template-audit`); first `cd`
there and confirm `git branch --show-current` == `task-181-v79-template-audit`.
Never edit the main repo. One writer per agent; agents are serialized (shared
files: template, exports, registry, status.json) — do not assume any other agent
is running.

## IDB sessions (mcp__ida-pro, use the `database` param; NEVER trust the export)
- v79 `88dfa464` · v83 `e7e0e4fe` · v87 `f21502cf` · v95 `10e5b966` · jms `cbcc4711`.
- v84 is byte-identical to v83 (project fact) — cite v83's finding for v84.
- Find a handler: `mcp__ida-pro__func_query` name_regex `On<Name>@CField_<Feature>`; decompile with `mcp__ida-pro__decompile`.

## Steps
1. **RE the true read-order** in v79 + v83 + v95 (and v87 + jms if quick).
   Decompile `CField_<Feature>::On<Name>`. Write down the exact Decode sequence
   (Decode1=byte, Decode2=uint16, Decode4=uint32, DecodeStr=ascii string,
   loops, conditionals). v95 is PDB-backed — best field names.
2. **Compare** to the atlas struct/Encode in `libs/atlas-packet/.../clientbound/<file>.go`.
   State the divergence precisely.
3. **Re-model** the struct + `Encode` + `Decode` to the true wire. Off-wire flags
   (client gates on its own state) → recover in Decode from `r.Available()`;
   count-prefixed lists → `[]entry` with `count=len`, loop in Encode/Decode.
   Update the channel wrapper (grep `New<Struct>` for its ONE caller in
   services/atlas-channel/.../writer/) to the new signature.
4. **Goldens**: rewrite the `_test.go` golden(s) with correct expected bytes;
   add `Test<Name>ByteOutputV79` (ctx GMS 79) and keep/refresh the version
   markers `// packet-audit:verify packet=<pkt> version=gms_vNN ida=0x...`
   (addresses from step 1). Run `go test -race ./<pkg>/` — must pass.
5. **Exports**: for EACH version, check `docs/packets/ida-exports/gms_vNN.json`
   `functions["CField_<Feature>::On<Name>"].calls` against the true order. If
   wrong (it usually is — that's the false-pass source), splice the correct
   op list (keep the version's `address`). For a VARIABLE count-loop, the export
   holds count + ONE iteration shape — do NOT expand it; a codec whose Encode
   loop flattens to that shape grades fine. jms often has no export file → its
   report is a documented residual (live-mcp regen only).
6. **Evidence** (only versions whose export you changed): re-pin —
   `go run ./tools/packet-audit evidence pin --packet <pkt> --version gms_vNN --ida CField_<Feature>::On<Name> --category TIER1-FIXTURE`.
   GOTCHA: invoke `evidence pin` as the FIRST arg. Do NOT pass root flags
   (-csv-*/-template/-ida-source) — they make main run the full pipeline instead.
7. **Route** (if not already in the v79 template): add `{"opCode":"0x<hex>","writer":"<Name>"}`
   to `template_gms_79_1.json` writers (opcode = the case value in the subclass
   `OnPacket` switch), then re-sort writers ascending by numeric opcode. Add a
   registry entry in `docs/packets/registry/gms_v79.yaml` mirroring the
   `SNOWBALL_STATE` block (op, direction clientbound, opcode decimal, fname,
   packet, provenance ida-discovered, ida.address decimal, note).
8. **Selective report regen** per changed version (the committed v79 report set
   is ~217-file stale — regen churns it; keep ONLY your writer's report):
   ```
   go run ./tools/packet-audit -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
     -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
     -template services/atlas-configurations/seed-data/templates/template_gms_<NN>_1.json \
     -ida-source docs/packets/ida-exports/gms_vNN.json
   git checkout -- $(git diff --name-only -- docs/packets/audits/gms_vNN/ | grep -viE '<WriterReportName>')
   git clean -fdq docs/packets/audits/gms_vNN/
   ```
   (Strays `FieldAriantScore`, `CashItemUse*`, `NpcShopOperation*` are recreated
   by regen every time — the `git clean` removes them; never commit them.)
9. **Status + verify**: `go run ./tools/packet-audit matrix` then
   `go run ./tools/packet-audit matrix --check` — MUST exit 0. Confirm your
   writer's cells show ✅ in `docs/packets/audits/STATUS.md`. Re-run the pkg
   tests + `go vet` + `cd services/atlas-channel/atlas.com/channel && go build ./...`.
10. **Commit** on the branch. Report: the divergence found, per-version cell
    status (✅), any jms residual, and the exact `matrix --check` result (paste it).

Final change set for one writer looks like: the codec .go + its _test.go, the
channel wrapper, the template, the registry, the changed exports, the re-pinned
evidence, that writer's per-version reports, and status.json/STATUS.md — nothing
else. If `git status` shows other reports changed, you didn't do the selective
revert in step 8; redo it.
