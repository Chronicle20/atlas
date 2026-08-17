# Task 27 Review — `PENDING_CHANGE_RESOLVED` consumer (CANCEL_* + pink text)

- **Diff:** `e08f71d5f..9d708a983` (`980406da1` implementation, `9d708a983` lint fix)
- **Scope:** `services/atlas-channel` only
- **Build:** PASS — `go build ./...` in `services/atlas-channel/atlas.com/channel` clean.
- **Tests:** PASS — `go test ./kafka/consumer/pendingchange/... -v` — 6/6 (`go vet` and `gofmt -l` both clean).

## Priority checks

### 1. Offline path writes nothing and acks nothing — PASS

`services/atlas-channel/atlas.com/channel/kafka/consumer/pendingchange/consumer.go:120-131` gates the entire delivery (both the `CANCEL_*` write and the pink text) behind `session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {...})`.

`IfPresentByCharacterId` (`services/atlas-channel/atlas.com/channel/session/processor.go:183-191`):
```go
func (p *ProcessorImpl) IfPresentByCharacterId(ch channel.Model) func(characterId uint32, f model.Operator[Model]) error {
	return func(characterId uint32, f model.Operator[Model]) error {
		s, err := p.ByCharacterIdModelProvider(ch)(characterId)()
		if err != nil {
			return nil
		}
		return f(s)
	}
}
```
When no session exists, the provider errors and `f` (the write closure) is never invoked — the callback body, including `session.Announce`, never runs. `TestOfflineCharacterWritesNothing` (`consumer_test.go:261-273`) exercises this against a real, empty session registry (`characterOffline` is a documented no-op, `consumer_test.go:127-130`) and asserts `env.wroteAnything()` is false. There is no separate "ack" mechanism in this consumer at all — no explicit offset-commit/notified_at call — so "writes nothing" is the complete story here; confirmed correct.

### 2. `APPLIED` produces neither packet nor pink text — PASS

`consumer.go:82-92` uses an explicit allow-list switch: only `StatusCancelled, StatusRejected, StatusExpired` fall through; every other value (`StatusApplied`, `StatusPending`, or any future/unrecognized status) hits the `default: return` before the packet-selection switch or the pink-text call are ever reached. `TestAppliedResolutionSendsNoCancelPacket` (`consumer_test.go:245-257`) asserts both `!wroteAnyCancelPacket()` and `!wroteAnything()` (zero writes total, not just zero cancel packets) — correctly proves no pink text either.

### 3. Packet selection per arm — PASS

`consumer.go:96-116`:
- `NAME_CHANGE` + `Status==REJECTED` + `Reason==name_taken` → `charcb.CancelNameChangeByOtherWriter` / `charcb.NewCancelNameChangeByOther().Encode` (`consumer.go:104-105`), imported from `libs/atlas-packet/character/clientbound` — confirmed the constant and type live there (`libs/atlas-packet/character/clientbound/cancel_name_change_by_other.go:12,58,60`), not in `cash/clientbound`.
- `NAME_CHANGE` otherwise-terminal → `cashcb.CashShopCancelNameChangeResultWriter` / `cashcb.CancelNameChangeResultCancelledBody()` (`consumer.go:107-108`), confirmed in `libs/atlas-packet/cash/clientbound/cancel_name_change_result.go:15,178-183`.
- `WORLD_TRANSFER` terminal → `cashcb.CashShopCancelTransferWorldResultWriter` / `cashcb.CancelTransferWorldResultCancelledBody()` (`consumer.go:110-112`), confirmed in `libs/atlas-packet/cash/clientbound/cancel_transfer_world_result.go:15,49` (`CancelTransferWorldResultCancelledBody` resolves the `"CANCELLED"` operations key, the `nResult==0x00` arm, matching the constant name used in `consumer.go:112`).

`CancelNameChangeByOther.Encode` (`libs/atlas-packet/character/clientbound/cancel_name_change_by_other.go:66-71`) writes nothing to the `response.Writer` — confirmed empty body as the brief states.

Pink text on the by-other arm: `resolutionPinkText` (`consumer.go:141-150`) is called unconditionally after packet selection (`consumer.go:118`) regardless of which arm was taken, and for `ChangeTypeNameChange` it always formats `b.RequestedName` into the message (`consumer.go:144`) — so the by-other arm gets the same requested-value-naming pink text as every other name-change arm. `TestNameTakenRejectionUsesCancelByOther` (`consumer_test.go:210-228`) explicitly asserts `wrotePinkTextContaining("Xray")` with a comment noting the empty-body rationale. Confirmed correct — the pink text is not skipped or degraded on this arm.

### 4. Event body struct mirror — PASS, verified field-for-field

`services/atlas-channel/.../pendingchange/kafka.go:47-66` (`StatusEvent[E]`, `ResolvedEventBody`) diffed directly against `services/atlas-character/atlas.com/character/kafka/message/pending_change/kafka.go:18-41`:

| Field | atlas-character (producer) | atlas-channel (mirror) | Match |
|---|---|---|---|
| `StatusEvent.TransactionId` | `uuid.UUID \`json:"transactionId"\`` | same | yes |
| `StatusEvent.CharacterId` | `uint32 \`json:"characterId"\`` | same | yes |
| `StatusEvent.WorldId` | `world.Id \`json:"worldId"\`` | same | yes |
| `StatusEvent.Type` | `string \`json:"type"\`` | same | yes |
| `StatusEvent.Body` | `E \`json:"body"\`` | same | yes |
| `ResolvedEventBody.PendingChangeId` | `uuid.UUID \`json:"pendingChangeId"\`` | same | yes |
| `ResolvedEventBody.ChangeType` | `string \`json:"changeType"\`` | same | yes |
| `ResolvedEventBody.Status` | `string \`json:"status"\`` | same | yes |
| `ResolvedEventBody.Reason` | `string \`json:"reason"\`` | same | yes |
| `ResolvedEventBody.RequestedName` | `string \`json:"requestedName"\`` | same | yes |
| `ResolvedEventBody.DestinationWorldId` | `world.Id \`json:"destinationWorldId"\`` | same | yes |

Field order also matches exactly. The status/type/reason string constants were separately cross-checked against `services/atlas-character/atlas.com/character/pending_change/entity.go:14-25` (`TypeNameChange="NAME_CHANGE"`, `TypeWorldTransfer="WORLD_TRANSFER"`, `StatusPending/Applied/Cancelled/Rejected/Expired`) and `eligibility.go:157,161` / `processor.go:197` (`"name_taken"`) — all identical to `pendingchange/kafka.go:14-41`. No mismatch found.

### 5. `produceWriters()` and consumer registration — PASS

`services/atlas-channel/atlas.com/channel/main.go:649-651`:
```go
cashcb.CashShopCancelNameChangeResultWriter,
cashcb.CashShopCancelTransferWorldResultWriter,
charcb.CancelNameChangeByOtherWriter,
```
All three present, immediately after the two `Check*` writers Task 26 registered (line 647-648), consistent with the "registration follows whoever emits" rule. `CashShopCheckNameChangeWriter` is not present in this block — correctly left untouched per Addendum 3/the brief's explicit instruction not to flag it.

Consumer registration: `main.go:230` (`pendingchange.InitConsumers(l)(cmf)(consumerGroupId)`) and `main.go:487` (`register(pendingchange.InitHandlers(fl)(sc)(wp)(rh))`) — both present, in the same position/shape as sibling consumers (e.g. `party_quest` immediately above each).

## Other findings

### Finding A (Important, pre-existing, not introduced by this diff) — `EVENT_TOPIC_CHARACTER_PENDING_CHANGE` is absent from `deploy/k8s/base/env-configmap.yaml`

Per DOM-23, every topic a service consumes should appear in the shared configmap as `KEY: "KEY"`. Grepping `deploy/k8s/base/env-configmap.yaml` for `PENDING_CHANGE` returns nothing — the key does not exist anywhere in that file (confirmed alongside the neighboring `EVENT_TOPIC_CHARACTER_*` entries at lines 100-104, none of which is `EVENT_TOPIC_CHARACTER_PENDING_CHANGE`).

This gap predates this diff — `EnvEventTopic = "EVENT_TOPIC_CHARACTER_PENDING_CHANGE"` was introduced by commit `b716c7425` (Task 4, atlas-character's producer side), not by `980406da1`/`9d708a983`, and `deploy/k8s/base/env-configmap.yaml` is untouched in this diff range. It is not this task's file to modify per its stated scope (`services/atlas-channel` only), but this task's new consumer (`pendingchange/kafka.go:12`) depends on the same env key and inherits the same gap.

Practical impact is currently masked, not absent: `topic.EnvProvider` (`libs/atlas-kafka/topic/*.go:14-25`) falls back to the literal token string when the env var is unset, and because this repo's convention sets `KEY: "KEY"` (env var name equals topic name), the fallback happens to produce the same topic string the configmap entry would have produced. That coincidence is exactly the kind of drift DOM-23 exists to prevent — if the naming convention or the fallback behavior ever changes, or if a real deployment sets a divergent value, this silently breaks with no build-time or test-time signal. Recommend adding `EVENT_TOPIC_CHARACTER_PENDING_CHANGE: "EVENT_TOPIC_CHARACTER_PENDING_CHANGE"` to `deploy/k8s/base/env-configmap.yaml`, either as part of closing out this task or as a fast-follow — it is producible right now and isn't blocked on anything.

### Everything else checked, no findings

- Pink-text convention (Addendum 5): `writer.WorldMessagePinkTextBody("", "", msg)` used on both `Announce` calls in `consumer.go:127-130`, matching the `medal=""`/`characterName=""` convention established in `socket/handler/cash_shop_check_transfer_world_possible.go:148`.
- World/tenant filtering: `consumer.go:75-78` uses `sc.IsWorld(t, e.WorldId)` (`server/model.go:60`) to ignore events for other worlds before any write; `TestResolvedIgnoresOtherWorld` (`consumer_test.go:276-293`) exercises this.
- CANCEL_* write failure (e.g. tenant template with no binding, expected on v48/jms) does not suppress the pink text — `consumer.go:127-130` logs the write error at Debug and unconditionally proceeds to send the pink text as a second, independent `Announce` call. This matches the brief's "the pink text is not version-gated the way the packet may be" requirement.
- `go build`, `go vet`, `go test`, `gofmt -l` all clean, module-local, in `services/atlas-channel/atlas.com/channel` (per Addendum 4's reduced verification scope — `-race`/`tools/verify.sh`/`tools/lint.sh` intentionally not run here).
- Fix-round commit `9d708a983` removes only the unused `discardConn` test stub; zero production-code diff, confirmed by `git diff --stat` (only `consumer_test.go` touched, -9 lines).

## Summary

### Blocking (must fix)
- None found in this diff's own scope.

### Non-blocking (should fix)
- Finding A: add `EVENT_TOPIC_CHARACTER_PENDING_CHANGE: "EVENT_TOPIC_CHARACTER_PENDING_CHANGE"` to `deploy/k8s/base/env-configmap.yaml`. Pre-existing gap from Task 4, currently masked by the `EnvProvider` fallback coincidentally matching the naming convention, but should be closed before this feature is considered fully wired for deployment.
