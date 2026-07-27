# Backend Audit — task-128 merge-resolution review

- **Branch:** task-128-item-tag-seal-incubator (post-merge with main, 79+ commits)
- **Scope:** hand-resolved merge-conflict files + gms_61/72/79 version-support addition (NOT the original task-128 feature)
- **Date:** 2026-07-15
- **Build/Vet:** clean on inventory, saga-orchestrator modules (re-verified affected packages)
- **Overall:** PASS — no confirmed defects in the merge resolution

## Focus checks (transaction correctness, outbox atomicity, interface/mock completeness, lock ordering)

### services/atlas-inventory/.../asset/processor.go — PASS
- UpdateOwner/ApplyLock/ClearLock added to interface (lines 50-52) and implemented with `*ProcessorImpl` receivers (lines 311, 327, 345). `var _ Processor` assertion holds (build passes).
- All three write via `updateOwner`/`updateFlagAndExpiration` on `p.db.WithContext(p.ctx)` (tx-aware when invoked through WithTransaction(tx)) and emit the UPDATED status event via `mb.Put` into the caller-supplied buffer (lines 318, 337, 352). Events therefore land in whatever emitter the buffer is bound to.
- ChangeTemplateAndEmit (357) uses the correct `ExecuteTransaction + outbox.EmitProvider(...,tx)` idiom.

### services/atlas-inventory/.../compartment/processor.go — PASS
- SetAssetOwnerAndEmit (1001) and ApplyAssetLockAndEmit (1035) use `database.ExecuteTransaction(p.db) { message.Emit(outbox.EmitProvider(p.l,p.ctx,tx))(...) }`, structurally identical to the reference sibling IncreaseCapacityAndEmit (617). Events go to the transactional outbox, NOT the direct producer.
- No double-nested transaction: the inner SetAssetOwner (1009) / ApplyAssetLock (1043) do NOT open a second ExecuteTransaction — they operate directly on `p.db` (which is `tx` when reached via the AndEmit wrapper) and rebind the asset sub-processor with `p.assetProcessor.WithTransaction(p.db)` (lines 1021, 1026, 1055, 1060). Contrast IncreaseCapacity which does open a nested savepoint; both are valid, and the simpler no-nest form here is correct because these methods perform a single read + single asset write.
- Lock ordering matches main's AndEmit idiom: outer tx opened first, then `invLock` acquired inside (lines 1012-1014, 1046-1048) — same order as IncreaseCapacity/ConsumeAsset/DestroyAsset/ExpireAsset. No new lock/tx inversion introduced.
- Note (not a finding): RequestReserveAndEmit (750) and CancelReservationAndEmit (797) still use the direct `message.Emit(p.producer)` path. This is main's pre-existing behavior — those methods mutate the in-memory ReservationRegistry rather than persist DB rows, so no transactional-outbox atomicity is required. Outside the hand-merged surface; consistent with main.

### asset/mock/processor.go + compartment/mock/processor.go — PASS
- All four new interface entries have mock fields and methods (asset mock 34-36/201-232; compartment mock 51-54/340-366). Nil-Func paths return safe non-nil curried funcs (asset mock 205-209, 216-220, 227-231) — no panic risk. Build/vet clean.

### services/atlas-tenants/.../configuration/processor.go — PASS
- CreateIncubatorRewardAndEmit (749), UpdateIncubatorRewardAndEmit (838), DeleteIncubatorRewardAndEmit (871) are byte-for-byte structurally identical to the sibling CreateMtsConfigAndEmit (995) / UpdateMtsConfigAndEmit / DeleteMtsConfigAndEmit: `ExecuteTransaction + message.EmitWithResult/Emit[...](outbox.EmitProvider(p.l,p.ctx,tx)) + NewProcessor(p.l,p.ctx,tx)`. The removed `p.p` field is not referenced. Correct.

### services/atlas-saga-orchestrator/.../saga/event_acceptance.go — PASS
- `EventKindAssetUpdated: OutcomeSuccess` present in outcomeTable (line 272), consistent with the other Asset* event kinds (271-275). Referenced by the SetAssetOwner/ApplyAssetLock step outcome mapping (115-116). SetAssetOwner/ApplyAssetLock are routed in saga/handler.go (902-903) and decoded in saga/model.go (1517-1524).

### services/atlas-saga-orchestrator/.../saga/producer.go — PASS
- No `kproducer` alias remnants anywhere in the module (grep clean); all references use `producer.`. Build clean.

### atlas-channel character_cash_item_use.go (+_test.go) — PASS
- `updateTimeFirst := t.MajorVersion() >= 87` (line 47) matches the packet-lib gate. mustTenant test helper (test 11-18) uses `tenant.Create(uuid, region, major, minor)` and drives table-driven version cases. Dispatch arms (item tag, seal, incubator, cube, vega, point-reset, hammer, store-search) form a coherent union with no duplicate/overlapping arm.

### libs/atlas-packet/cash/serverbound/item_use.go — PASS
- `t.MajorVersion() >= 87` gate applied symmetrically in Encode (49) and Decode (61). Correctly excludes v84 (byte-identical to v83, per the off-by-one memory note). Comment (26) documents the >=87-not-95 rationale.

## Version-support addition

### libs/atlas-packet/incubator/clientbound/result_test.go — PASS
- Real byte fixtures for gms_v61/72/79 (test 40, cases 49-51) asserting the `short` layout, with per-version `packet-audit:verify` markers (27-34) and distinct IDA addresses. Encode gate `Region=="GMS" && MajorVersion()>=95` (result.go 46) correctly leaves the three legacy versions on the short layout.

### seed-data/templates/template_gms_{61,72,79}_1.json — PASS
- IncubatorResult writer added with opCode `0x42` in each template's writers block (gms_61 726-727, gms_72 766-767, gms_79 796). gms_72's second 0x42 occurrence (754) is a serverbound handler (carries a `validator`, separate opcode namespace), NOT a writer collision. Within the writers block, 0x42→IncubatorResult is unique per file.

## Summary
No blocking or non-blocking findings. The merge resolution faithfully reconciles main's ProcessorImpl/outbox refactor with task-128's methods; transaction nesting, outbox atomicity, lock ordering, and interface/mock completeness are all correct.
