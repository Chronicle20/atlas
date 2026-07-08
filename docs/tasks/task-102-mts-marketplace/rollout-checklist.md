# MTS (task-102) — live-config rollout checklist

The MTS feature adds new socket handler/writer opcodes and per-version
`operations` mode tables to the tenant seed templates. **Seed templates apply
only at tenant creation — existing tenants do NOT retroactively receive them.**
Symptom of a missed step: the client action no-ops and the channel logs
`unhandled message op 0xXX` at info ([[bug_new_opcodes_not_in_live_tenant_config]],
[[bug_socket_handler_missing_validator_silently_dropped]]).

For each existing tenant that should get MTS, patch the live tenant
configuration and restart the channel:

## 1. Socket handlers (serverbound) — `socket.handlers[]`
Add, per version, with a **validator on every entry** (a validator-less entry is
silently dropped by `BuildHandlerMap`):

| Handler | Validator | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|---|
| `EnterMtsHandle` | LoggedInValidator | 0x09C | 0x0A0 | 0x0A4 | 0x0B4 | 0x0A6 |
| `ItcStatusChargeHandle` | NoOpValidator | 0x0FB | 0x102 | 0x109 | 0x132 | 0x10A |
| `ItcQueryCashRequestHandle` | LoggedInValidator | 0x0FC | 0x103 | 0x10A | 0x133 | 0x10B |
| `ItcOperationHandle` | LoggedInValidator | 0x0FD | 0x104 | 0x10B | 0x134 | 0x10C |

> Note the **v84 opcodes are shifted +7** vs v83 (0x102/0x103/0x104, not the
> v83 0x0FB/0x0FC/0x0FD) — the registry had csv-carryover-stale values that this
> task corrected against IDA ground truth.

## 2. `ITC_OPERATION` `operations` mode table — handler `options.operations`
Per version (IDA-verified uniform across all five): `REGISTER_SALE:2`,
`SALE_CURRENT_ITEM:3`, `REGISTER_WISH_ENTRY:4`, `GET_ITC_LIST:5`,
`SEARCH_ITC_LIST:6`, `CANCEL_SALE:7`, `TAKE_HOME:8`, `SET_ZZIM:9`,
`DELETE_ZZIM:10`, `VIEW_WISH:11`, `BUY_WISH:12`, `CANCEL_WISH:13`, `BUY:16`,
`BUY_ZZIM:17`, `REGISTER_AUCTION:18`, `PLACE_BID:19`, `BUY_AUCTION_IMM:20`.
A missing/wrong table makes the inbound dispatcher's mode reverse-resolve fail →
the action no-ops.

## 3. Clientbound writers — `socket.writers[]`
The `MTS_OPERATION` / `MTS_OPERATION2` writers (task-096) must be present
(gms_v83/84/87/95: opcodes 0x15C/0x15B family; jms clientbound MTS results are
version-absent — see design.md §9.4). Confirm the writer entries exist.

## 3b. `noticeFailReasons` writer table — descriptive failure notices
The `MtsOperation` writer's options need the per-version `noticeFailReasons`
table (semantic key -> client `CITC::NoticeFailReason` byte: `NOT_ENOUGH_NX:66`,
`ITEM_SOLD:81`, ... — identical across gms v83/84/87/95, IDA-verified; see the
seed templates for the full ten-key table). atlas-mts emits semantic keys on
`BUY_FAILED`/`BID_FAILED`; the channel resolves them here into the mode-24
reason-notice arm. A tenant WITHOUT the table degrades gracefully to the bare
generic *Failed arm (no crash) — patch the live config to get the descriptive
messages.

## 4. `mts-configs` tenant config resource (economic knobs)
The `atlas-tenants` `mts-configs` resource provides per-tenant knobs (listing
fee, commission, caps, level gate, auction window, fixed-sale term, price
floor, page size, bid increment). On a miss, `atlas-mts` falls back to
defaults (5000/0.10/10/10/24/168/168/110/16/1), so MTS works without it — but
set it (via the
atlas-ui MTS config page) to customize. No seed file ships by default.

## 5. Service deploy
`atlas-mts` is a new service (`deploy/k8s/base/atlas-mts.yaml`, in
`kustomization.yaml`, `services.json`, `docker-bake.hcl go_services`). The nginx
ingress routes `/api/worlds/{worldId}/listings` and
`/api/characters/{characterId}/mts/*` to `atlas-mts:8080`
(`deploy/shared/routes.conf`). Ensure `atlas-mts` is deployed before enabling MTS
entry on the channel.

## 6. Restart the channel
After patching the live tenant config, restart `atlas-channel` so it re-reads the
handler/writer/operations tables (the projection does not hot-reload handlers).

## 7. E2E test routes (optional, dev only)
`atlas-mts` ships env-gated test routes (seed / force-expire / sweep /
simulated purchase / simulated bid) under `/api/test/*`. Enable with:

```
kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED=true
```

Disable with:

```
kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED-
```

These routes are **not** routed through the nginx ingress — reach them via a
port-forward to `atlas-mts:8080`. **Never enable in production.** Full usage
recipes (seeding listings/bids, forcing expirations, driving a purchase/bid
end-to-end) are in `e2e-test-playbook.md` (same folder).
