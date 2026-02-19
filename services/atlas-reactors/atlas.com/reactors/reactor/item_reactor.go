package reactor

import (
	"atlas-reactors/kafka/producer"
	dropMessage "atlas-reactors/kafka/message/drop"
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	kafkaProducer "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

const (
	itemReactorStateType               = int32(100)
	defaultItemReactorActivationDelay  = 5000
	envItemReactorActivationDelayMs    = "ITEM_REACTOR_ACTIVATION_DELAY_MS"
)

var (
	pendingActivations     = make(map[uint32]*time.Timer)
	pendingActivationsLock sync.Mutex
)

func getActivationDelay() time.Duration {
	if val := os.Getenv(envItemReactorActivationDelayMs); val != "" {
		if ms, err := strconv.Atoi(val); err == nil {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultItemReactorActivationDelay * time.Millisecond
}

func ActivateItemReactors(l logrus.FieldLogger) func(ctx context.Context) func(dropId uint32, itemId uint32, quantity uint32, dropX int16, dropY int16, characterId uint32, f field.Model) {
	return func(ctx context.Context) func(dropId uint32, itemId uint32, quantity uint32, dropX int16, dropY int16, characterId uint32, f field.Model) {
		return func(dropId uint32, itemId uint32, quantity uint32, dropX int16, dropY int16, characterId uint32, f field.Model) {
			t := tenant.MustFromContext(ctx)
			reactors := GetRegistry().GetInField(t, f)
			for _, r := range reactors {
				if matchesItemReactor(r, itemId, quantity, dropX, dropY) {
					scheduleItemReactorActivation(l, ctx)(r, dropId, characterId, f)
					return
				}
			}
		}
	}
}

func matchesItemReactor(r Model, itemId uint32, quantity uint32, dropX int16, dropY int16) bool {
	stateInfo := r.Data().StateInfo()
	events, ok := stateInfo[r.State()]
	if !ok {
		return false
	}

	for _, event := range events {
		if event.Type() != itemReactorStateType {
			continue
		}
		ri := event.ReactorItem()
		if ri == nil {
			continue
		}
		if ri.ItemId() != itemId {
			continue
		}
		if uint32(ri.Quantity()) > quantity {
			continue
		}
		if isPositionInReactorArea(dropX, dropY, r) {
			return true
		}
	}
	return false
}

func isPositionInReactorArea(dropX int16, dropY int16, r Model) bool {
	tl := r.Data().TL()
	br := r.Data().BR()

	left := r.X() + tl.X()
	top := r.Y() + tl.Y()
	right := r.X() + br.X()
	bottom := r.Y() + br.Y()

	return dropX >= left && dropX <= right && dropY >= top && dropY <= bottom
}

func scheduleItemReactorActivation(l logrus.FieldLogger, ctx context.Context) func(r Model, dropId uint32, characterId uint32, f field.Model) {
	return func(r Model, dropId uint32, characterId uint32, f field.Model) {
		pendingActivationsLock.Lock()
		defer pendingActivationsLock.Unlock()

		if _, exists := pendingActivations[r.Id()]; exists {
			l.Debugf("Reactor [%d] already has a pending item activation. Skipping.", r.Id())
			return
		}

		delay := getActivationDelay()
		reactorId := r.Id()

		l.Debugf("Scheduling item-reactor activation for reactor [%d] with drop [%d] in %v.", reactorId, dropId, delay)

		timer := time.AfterFunc(delay, func() {
			pendingActivationsLock.Lock()
			delete(pendingActivations, reactorId)
			pendingActivationsLock.Unlock()

			_, err := GetRegistry().Get(reactorId)
			if err != nil {
				l.Debugf("Reactor [%d] no longer exists. Skipping item-reactor activation.", reactorId)
				return
			}

			err = emitConsumeDropCommand(l)(ctx)(f, dropId)
			if err != nil {
				l.WithError(err).Errorf("Failed to emit CONSUME command for drop [%d].", dropId)
				return
			}

			err = Hit(l)(ctx)(reactorId, characterId, 0)
			if err != nil {
				l.WithError(err).Errorf("Failed to hit reactor [%d] after item-reactor activation.", reactorId)
			}
		})

		pendingActivations[reactorId] = timer
	}
}

func emitConsumeDropCommand(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, dropId uint32) error {
	return func(ctx context.Context) func(f field.Model, dropId uint32) error {
		return func(f field.Model, dropId uint32) error {
			return producer.ProviderImpl(l)(ctx)(dropMessage.EnvCommandTopicDrop)(consumeDropCommandProvider(f, dropId))
		}
	}
}

func consumeDropCommandProvider(f field.Model, dropId uint32) model.Provider[[]kafka.Message] {
	key := kafkaProducer.CreateKey(int(f.MapId()))
	value := &dropMessage.Command[dropMessage.CommandConsumeBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      dropMessage.CommandTypeConsume,
		Body: dropMessage.CommandConsumeBody{
			DropId: dropId,
		},
	}
	return kafkaProducer.SingleMessageProvider(key, value)
}

func CancelPendingActivation(reactorId uint32) {
	pendingActivationsLock.Lock()
	defer pendingActivationsLock.Unlock()

	if timer, exists := pendingActivations[reactorId]; exists {
		timer.Stop()
		delete(pendingActivations, reactorId)
	}
}

func CancelAllPendingActivations() {
	pendingActivationsLock.Lock()
	defer pendingActivationsLock.Unlock()

	for id, timer := range pendingActivations {
		timer.Stop()
		delete(pendingActivations, id)
	}
}
