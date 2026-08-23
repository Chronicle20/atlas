# Task 12 — Scoped re-review (fix round 2)

Range: `3c658eb49..1a7cc20d6` (1 commit,
`1a7cc20d6 fix(orchestrator): check Sscanf error in parcel asset lookup, extend expansion-gate test`)

Scope: only this fix commit. The rest of Task 12 was previously approved and
is not re-reviewed here.

## Finding 1 — unchecked `fmt.Sscanf` in `expandTransferToParcel`'s asset-lookup loop

**Verdict: ADDRESSED**

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:2203-2216`
now reads:

```go
assetId, perr := strconv.ParseUint(comp.Assets[i].Id, 10, 32)
if perr != nil {
	p.l.WithError(perr).Warnf("Asset id [%s] in character [%d]'s compartment is not numeric. Skipping it.", comp.Assets[i].Id, payload.CharacterId)
	continue
}
if uint32(assetId) == payload.AssetId {
	foundAsset = &comp.Assets[i]
	break
}
```

- The parse error is checked and no longer dropped; `strconv` is imported at
  `processor.go:20`.
- Compared byte-for-byte against `expandTransferToTrade`
  (`processor.go:1458-1473`), the new code is a verbatim mirror of that
  function's asset-lookup idiom (same comment, same `ParseUint`/skip/`continue`
  pattern, same log call shape) — the implementer's claim that it copied that
  idiom is confirmed by reading the source, not by report text.
- Semantics: a malformed entry is `continue`d over (not selected, not
  defaulted to `assetId == 0`), so the loop keeps scanning the rest of
  `comp.Assets`. If the true target entry is malformed, the loop finishes
  without matching, `foundAsset` stays `nil`, and execution falls into the
  existing `if foundAsset == nil { return nil, fmt.Errorf("no asset found with
  id [...]...") }` path at `processor.go:2223-2226` — i.e. the correct
  "asset missing" error, not a silent wrong-asset match on 0. This satisfies
  the check requested: a skipped malformed entry still terminates at the
  `asset missing` error rather than mis-selecting a later asset.

## Finding 2 — `TestIsExpandableActionCoversExpansionSwitch` composites list

**Verdict: ADDRESSED**

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/mts_expansion_test.go:273`
adds `TransferToParcel, WithdrawFromParcel` to the `composites` slice.

- Confirmed the guard is real, not vacuous: `isExpandableAction`
  (`processor.go:1179-1190`) is a `switch` over `Action` with an explicit
  `default: return false`. `TransferToParcel`/`WithdrawFromParcel` are listed
  in the `case` at `processor.go:1181-1185` (added by an earlier, already-approved
  commit in this task, not by this fix). If that case arm were absent,
  `isExpandableAction(TransferToParcel)` would fall to the `default` branch and
  return `false`, and `require.Truef` in the test loop would fail — so the
  guard genuinely fails when an expandable action is missing from the switch,
  confirming it is not a test that passes regardless.
- Ran the test directly:
  `go test ./saga/... -run TestIsExpandableActionCoversExpansionSwitch -v` →
  `--- PASS: TestIsExpandableActionCoversExpansionSwitch (0.00s)`.

## Sweep claim (fix-round-1 code, `parcel/rest.go` and `parcel/requests.go`)

Spot-checked both files in
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/`:

- `rest.go` (94 lines) — pure data/JSON:API-model file, no parsing, no
  ignored error returns.
- `requests.go` (36 lines) — `getBaseRequest` error is checked and
  propagated; `RequestParcel` returns `requests.GetRequest[RestModel](url)(l,
  ctx)`'s result directly, no discarded error. No `Sscanf` or other unchecked
  conversions present.

No dropped errors found in either file — consistent with the implementer's
sweep claim for this pair of files. (Scope note: only these two files named
in the task brief were spot-checked, per instructions; no broader sweep of
other packages was performed.)

## Build/test verification (module-local, this diff only)

- `go build ./...` in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` — clean, no output.
- `go vet ./...` — clean, no output.
- `go test ./saga/...` — `ok` for both `saga` and `saga/mock`.

## New breakage introduced by this fix diff

None found. The diff is additive/corrective only: one loop body swap in
`expandTransferToParcel` and one slice-literal addition in a test. No other
call sites, exported signatures, or control flow changed.

## Not evaluable

None — both findings and the sweep claim were fully checkable within the
2-file, ~4KB diff plus the one comparison file (`expandTransferToTrade`) the
diff's own comment cites as the mirrored idiom.

## Notes (non-blocking, out of scope for this fix)

- None.
