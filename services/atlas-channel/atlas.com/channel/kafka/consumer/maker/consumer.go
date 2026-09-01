// Package maker consumes the craft saga's terminal events off the shared
// EVENT_TOPIC_SAGA_STATUS topic and writes the corresponding MAKER_RESULT to
// the crafting character's session -- the only writer of that packet
// (FR-5.2, design §3.2 step 5). Task 25's socket handler writes the FAILED
// arm for a synchronous rejection; this package covers the three terminal
// paths that only the saga knows about: completion, compensation and
// timeout.
package maker

import (
	consumer2 "atlas-channel/kafka/consumer"
	sagamsg "atlas-channel/kafka/message/saga"
	"atlas-channel/listener"
	sagamodel "atlas-channel/saga"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// makerResultSuccessValue is the MAKER_RESULT nResult written on a completed
// craft saga. Design §4.3.2 (C-1): the client's guard treats nResult ∈ {0,1}
// as a legitimate result carrying a mode-shaped body; every completed craft
// arm this consumer writes reports plain success.
const makerResultSuccessValue = uint32(0)

// makerResultFailedValue mirrors socket/handler.makerResultFailedValue
// (grounded in libs/atlas-packet/character/clientbound/maker_result_test.go:432
// and plan.md:1455-1458): any nResult > 1 makes the arm bodyless on the wire,
// so the same sentinel covers every terminal-failure path this consumer
// handles -- compensation, timeout, and an unrecognized/undecodable manifest.
// Duplicated locally rather than exported across packages, matching how the
// saga/mts consumers each keep their own local failure-sentinel constants.
const makerResultFailedValue = uint32(2)

// InitConsumers registers the saga status event consumer. The topic is
// already registered by kafka/consumer/saga's InitConsumers (both packages
// listen on the same EVENT_TOPIC_SAGA_STATUS topic); Manager.AddConsumer is
// idempotent per topic, so this call is a documented no-op when that
// registration has already run and a normal registration otherwise -- this
// package must not assume ordering with kafka/consumer/saga.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("saga_status_event")(sagamsg.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

// InitHandlers wires this package's two terminal-event handlers onto the
// saga status topic, alongside kafka/consumer/saga's own handlers -- the
// underlying Consumer fans a single message out to every registered handler,
// so both packages see every event and each decides independently whether it
// applies.
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(sagamsg.EnvStatusEventTopic)()

				register := func(h handler.Handler) error {
					id, err := rf(t, h)
					if err != nil {
						return err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
					return nil
				}

				if err := register(message.AdaptHandler(message.PersistentConfig(handleCraftCompleted(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleCraftFailed(sc, wp)))); err != nil {
					return nil, err
				}
				return handles, nil
			}
		}
	}
}

// handleCraftCompleted writes the CREATE / CREATE_WITH_UPGRADE /
// MONSTER_CRYSTAL / DISASSEMBLE arm for a completed craft saga. The craft
// saga's type is the generic InventoryTransaction, shared with non-craft
// operations, so the discriminator is Results["kind"] ==
// saga.MakerCraftResultKind, not SagaType. Every branch that recognizes the
// event as a craft completion writes SOME MAKER_RESULT arm before returning
// -- an undecodable manifest or an unrecognized mode still gets the FAILED
// arm (FR-5.2 admits no silent path).
func handleCraftCompleted(sc server.Model, wp writer.Producer) message.Handler[sagamsg.StatusEvent[sagamsg.StatusEventCompletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e sagamsg.StatusEvent[sagamsg.StatusEventCompletedBody]) {
		if e.Type != sagamsg.StatusEventTypeCompleted {
			return
		}
		if resultKind(e.Body.Results) != sagamsg.MakerCraftResultKind {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		characterId := resultUint32(e.Body.Results, "characterId")
		if characterId == 0 {
			l.WithField("transaction_id", e.TransactionId.String()).Warn("Completed craft saga missing characterId in Results; cannot write MAKER_RESULT.")
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(characterId)
		if err != nil {
			l.WithField("character_id", characterId).Debug("Character not connected, skipping MAKER_RESULT notification.")
			return
		}
		if s.ChannelId() != sc.ChannelId() {
			return
		}

		manifest, ok := decodeCraftManifest(e.Body.Results)
		if !ok {
			l.WithField("transaction_id", e.TransactionId.String()).Warn("Completed craft saga's manifest did not decode; writing MAKER_RESULT FAILED so the client is not left locked.")
			announceMakerResult(l, ctx, wp, s, charpkt.MakerResultFailedBody(makerResultFailedValue))
			return
		}

		announceMakerResult(l, ctx, wp, s, craftResultBody(manifest))
	}
}

// craftResultBody selects the MAKER_RESULT arm for a decoded craft manifest,
// keyed on manifest.Mode -- not on the saga type or step list, both of which
// cannot distinguish mode 1 from mode 2 (manifest-carrier-derivation.md §2
// F2). An unrecognized mode falls back to FAILED rather than writing nothing.
func craftResultBody(manifest sagamodel.CraftManifestPayload) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	switch manifest.Mode {
	case 1:
		return charpkt.MakerResultCreateBody(makerResultSuccessValue, manifest.NoItemAwarded, manifest.TargetItemId, manifest.ItemNum, toMakerMaterials(manifest.Materials), manifest.GemItemIds, manifest.CatalystUsed, manifest.CatalystItemId, manifest.MesoCost)
	case 2:
		return charpkt.MakerResultCreateWithUpgradeBody(makerResultSuccessValue, manifest.NoItemAwarded, manifest.TargetItemId, manifest.ItemNum, toMakerMaterials(manifest.Materials), manifest.GemItemIds, manifest.CatalystUsed, manifest.CatalystItemId, manifest.MesoCost)
	case 3:
		return charpkt.MakerResultMonsterCrystalBody(makerResultSuccessValue, manifest.CrystalItemId, manifest.LeftoverItemId)
	case 4:
		return charpkt.MakerResultDisassembleBody(makerResultSuccessValue, manifest.DisassembledItemId, toMakerMaterials(manifest.Crystals), manifest.MesoCost)
	default:
		return charpkt.MakerResultFailedBody(makerResultFailedValue)
	}
}

// toMakerMaterials converts the saga-level manifest items to the clientbound
// (itemId, count) pairs the create/disassemble arms write.
func toMakerMaterials(items []sagamodel.CraftManifestItem) []charcb.MakerMaterial {
	out := make([]charcb.MakerMaterial, 0, len(items))
	for _, it := range items {
		out = append(out, charcb.NewMakerMaterial(it.ItemId, it.Count))
	}
	return out
}

// decodeCraftManifest re-marshals Results["craftManifest"] -- a
// map[string]any after the Kafka JSON round-trip -- into the typed
// CraftManifestPayload both ends bind to (manifest-carrier-derivation.md §5
// hop 3). The scalar resultUint32 helper cannot read a nested list, and no
// flattening of the map can express []CraftManifestItem.
func decodeCraftManifest(results map[string]any) (sagamodel.CraftManifestPayload, bool) {
	var out sagamodel.CraftManifestPayload
	if results == nil {
		return out, false
	}
	raw, ok := results["craftManifest"]
	if !ok {
		return out, false
	}
	bs, err := json.Marshal(raw)
	if err != nil {
		return out, false
	}
	if err := json.Unmarshal(bs, &out); err != nil {
		return out, false
	}
	return out, true
}

// handleCraftFailed writes the FAILED arm for a compensated or timed-out
// craft saga. A craft saga's type is the generic InventoryTransaction,
// shared with Duey/note-gift-forward/pet-destroy sagas -- but
// atlas-saga-orchestrator's EmitSagaFailed (task-285 Task 26a) routes a real,
// non-zero CharacterId ONLY for a craft saga (guarded on the presence of its
// RecordCraftManifest step); every other InventoryTransaction saga still
// resolves to CharacterId 0 there, so CharacterId != 0 is today an accurate
// craft discriminator without needing a Results marker (FAILED carries no
// Results at all).
func handleCraftFailed(sc server.Model, wp writer.Producer) message.Handler[sagamsg.StatusEvent[sagamsg.StatusEventFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e sagamsg.StatusEvent[sagamsg.StatusEventFailedBody]) {
		if e.Type != sagamsg.StatusEventTypeFailed {
			return
		}
		if e.Body.SagaType != sagamsg.SagaTypeInventoryTransaction {
			return
		}
		if e.Body.CharacterId == 0 {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.Body.CharacterId)
		if err != nil {
			l.WithField("character_id", e.Body.CharacterId).Debug("Character not connected, skipping MAKER_RESULT notification.")
			return
		}
		if s.ChannelId() != sc.ChannelId() {
			return
		}

		announceMakerResult(l, ctx, wp, s, charpkt.MakerResultFailedBody(makerResultFailedValue))
	}
}

// announceMakerResult writes the given MAKER_RESULT body to s.
func announceMakerResult(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) {
	if err := session.Announce(l)(ctx)(wp)(charcb.MakerResultWriter)(body)(s); err != nil {
		l.WithError(err).WithField("character_id", s.CharacterId()).Error("Unable to announce MAKER_RESULT to character.")
	}
}

// resultKind reads the "kind" marker off a saga COMPLETED Results map.
// Duplicated from kafka/consumer/saga's unexported helper of the same name
// (package-local, not shared across packages).
func resultKind(results map[string]any) string {
	if results == nil {
		return ""
	}
	if v, ok := results["kind"].(string); ok {
		return v
	}
	return ""
}

// resultUint32 reads a uint32 off a saga COMPLETED Results map, tolerating
// the float64 the value becomes after a JSON round-trip. Duplicated from
// kafka/consumer/saga's unexported helper of the same name.
func resultUint32(results map[string]any, key string) uint32 {
	if results == nil {
		return 0
	}
	switch v := results[key].(type) {
	case float64:
		return uint32(v)
	case uint32:
		return v
	case int:
		return uint32(v)
	default:
		return 0
	}
}
