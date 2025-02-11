package monster

import (
	"atlas-monsters/kafka/producer"
	_map "atlas-monsters/map"
	"atlas-monsters/monster/information"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

func byIdProvider(ctx context.Context) func(monsterId uint32) model.Provider[Model] {
	return func(monsterId uint32) model.Provider[Model] {
		return func() (Model, error) {
			t := tenant.MustFromContext(ctx)
			return GetMonsterRegistry().GetMonster(t, monsterId)
		}
	}
}

func ByMapProvider(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
		return func() ([]Model, error) {
			t := tenant.MustFromContext(ctx)
			return GetMonsterRegistry().GetMonstersInMap(t, worldId, channelId, mapId), nil
		}
	}
}

func ControlledInMapProvider(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
		return model.FilteredProvider(ByMapProvider(ctx)(worldId, channelId, mapId), model.Filters(Controlled))
	}
}

func NotControlledInMapProvider(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
		return model.FilteredProvider(ByMapProvider(ctx)(worldId, channelId, mapId), model.Filters(NotControlled))
	}
}

func ControlledByCharacterInMapProvider(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) model.Provider[[]Model] {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) model.Provider[[]Model] {
		return model.FilteredProvider(ByMapProvider(ctx)(worldId, channelId, mapId), model.Filters(IsControlledBy(characterId)))
	}
}

func allByTenantProvider() model.Provider[map[tenant.Model][]Model] {
	return func() (map[tenant.Model][]Model, error) {
		return GetMonsterRegistry().GetMonsters(), nil
	}
}

func GetById(ctx context.Context) func(monsterId uint32) (Model, error) {
	return func(monsterId uint32) (Model, error) {
		return byIdProvider(ctx)(monsterId)()
	}
}

func GetInMap(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
		return ByMapProvider(ctx)(worldId, channelId, mapId)()
	}
}

func CreateMonster(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, input RestModel) (Model, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, input RestModel) (Model, error) {
		return func(worldId byte, channelId byte, mapId uint32, input RestModel) (Model, error) {
			l.Debugf("Attempting to create monster [%d] in world [%d] channel [%d] map [%d].", input.MonsterId, worldId, channelId, mapId)
			ma, err := information.GetById(l)(ctx)(input.MonsterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve information necessary to create monster [%d].", input.MonsterId)
				return Model{}, err
			}

			t := tenant.MustFromContext(ctx)
			m := GetMonsterRegistry().CreateMonster(t, worldId, channelId, mapId, input.MonsterId, input.X, input.Y, input.Fh, 5, input.Team, ma.HP(), ma.MP())

			cid, err := GetControllerCandidate(l)(ctx)(worldId, channelId, mapId)
			if err == nil {
				l.Debugf("Created monster [%d] with id [%d] will be controlled by [%d].", m.MonsterId(), m.UniqueId(), cid)
				m, err = StartControl(l)(ctx)(m.UniqueId(), cid)
				if err != nil {
					l.WithError(err).Errorf("Unable to start [%d] controlling [%d] in world [%d] channel [%d] map [%d].", cid, m.UniqueId(), m.WorldId(), m.ChannelId(), m.MapId())
				}
			}

			l.Debugf("Created monster [%d] in world [%d] channel [%d] map [%d]. Emitting Monster Status.", input.MonsterId, worldId, channelId, mapId)
			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicMonsterStatus)(emitCreated(m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId()))
			return m, nil
		}
	}
}

func FindNextController(l logrus.FieldLogger) func(ctx context.Context) model.Operator[Model] {
	return func(ctx context.Context) model.Operator[Model] {
		return func(m Model) error {
			cid, err := GetControllerCandidate(l)(ctx)(m.WorldId(), m.ChannelId(), m.MapId())
			if err != nil {
				return err
			}

			_, err = StartControl(l)(ctx)(m.UniqueId(), cid)
			if err != nil {
				l.WithError(err).Errorf("Unable to start [%d] controlling [%d] in world [%d] channel [%d] map [%d].", cid, m.UniqueId(), m.WorldId(), m.ChannelId(), m.MapId())
			}
			return err
		}
	}
}

func GetControllerCandidate(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) (uint32, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) (uint32, error) {
		return func(worldId byte, channelId byte, mapId uint32) (uint32, error) {
			l.Debugf("Identifying controller candidate for monsters in world [%d] channel [%d] map [%d].", worldId, channelId, mapId)

			controlCounts, err := model.CollectToMap(_map.CharacterIdsInMapProvider(l)(ctx)(worldId, channelId, mapId), characterIdKey, zeroValue)()
			if err != nil {
				l.WithError(err).Errorf("Unable to initialize controller candidate map.")
				return 0, err
			}
			err = model.ForEachSlice(ControlledInMapProvider(ctx)(worldId, channelId, mapId), func(m Model) error {
				controlCounts[m.ControlCharacterId()] += 1
				return nil
			})

			var index = uint32(0)
			for key, val := range controlCounts {
				if index == 0 {
					index = key
				} else if val < controlCounts[index] {
					index = key
				}
			}

			if index == 0 {
				return 0, errors.New("should not get here")
			} else {
				l.Debugf("Controller candidate has been determined. Character [%d].", index)
				return index, nil
			}
		}
	}
}

func zeroValue(id uint32) int {
	return 0
}

func characterIdKey(id uint32) uint32 {
	return id
}

func StartControl(l logrus.FieldLogger) func(ctx context.Context) func(uniqueId uint32, controllerId uint32) (Model, error) {
	return func(ctx context.Context) func(uniqueId uint32, controllerId uint32) (Model, error) {
		return func(uniqueId uint32, controllerId uint32) (Model, error) {
			m, err := GetById(ctx)(uniqueId)
			if err != nil {
				return Model{}, err
			}

			if m.ControlCharacterId() != 0 {
				err = StopControl(l)(ctx)(m)
				if err != nil {
					return Model{}, err
				}
			}

			m, err = GetById(ctx)(uniqueId)
			if err != nil {
				return Model{}, err
			}

			t := tenant.MustFromContext(ctx)
			m, err = GetMonsterRegistry().ControlMonster(t, m.UniqueId(), controllerId)
			if err == nil {
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicMonsterStatus)(emitStartControl(m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId(), m.ControlCharacterId()))
			}
			return m, err
		}
	}
}

func StopControl(l logrus.FieldLogger) func(ctx context.Context) model.Operator[Model] {
	return func(ctx context.Context) model.Operator[Model] {
		return func(m Model) error {
			t := tenant.MustFromContext(ctx)
			m, err := GetMonsterRegistry().ClearControl(t, m.UniqueId())
			if err == nil {
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicMonsterStatus)(emitStopControl(m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId(), m.ControlCharacterId()))
			}
			return err
		}
	}
}

func DestroyInTenant(l logrus.FieldLogger) func(ctx context.Context) func(t tenant.Model) model.Operator[[]Model] {
	return func(ctx context.Context) func(t tenant.Model) model.Operator[[]Model] {
		return func(t tenant.Model) model.Operator[[]Model] {
			return func(models []Model) error {
				tctx := tenant.WithContext(ctx, t)
				idp := model.SliceMap(IdTransformer)(model.FixedProvider(models))(model.ParallelMap())
				return model.ForEachSlice(idp, Destroy(l)(tctx), model.ParallelExecute())
			}
		}
	}
}

func DestroyAll(l logrus.FieldLogger, ctx context.Context) error {
	return model.ForEachMap(allByTenantProvider(), DestroyInTenant(l)(ctx), model.ParallelExecute())
}

func Destroy(l logrus.FieldLogger) func(ctx context.Context) func(uniqueId uint32) error {
	return func(ctx context.Context) func(uniqueId uint32) error {
		return func(uniqueId uint32) error {
			t := tenant.MustFromContext(ctx)
			m, err := GetMonsterRegistry().RemoveMonster(t, uniqueId)
			if err != nil {
				return err
			}

			return producer.ProviderImpl(l)(ctx)(EnvEventTopicMonsterStatus)(emitDestroyed(m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId()))
		}
	}
}

type MovementSummary struct {
	X      int16
	Y      int16
	Stance byte
}

func MovementSummaryProvider() (MovementSummary, error) {
	return MovementSummary{}, nil
}

func FoldMovement(summary MovementSummary, e Element) (MovementSummary, error) {
	res := MovementSummary{
		X:      summary.X,
		Y:      summary.Y,
		Stance: summary.Stance,
	}
	if e.TypeStr == MovementTypeNormal {
		res.X = e.X
		res.Y = e.Y
		res.Stance = e.MoveAction
	}
	return res, nil
}

func Move(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, id uint32, observerId uint32, skillPossible bool, skill int8, skillId int16, skillLevel int16, multiTarget []Position, randomTimes []int32, movement Movement) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, id uint32, observerId uint32, skillPossible bool, skill int8, skillId int16, skillLevel int16, multiTarget []Position, randomTimes []int32, movement Movement) error {
		t := tenant.MustFromContext(ctx)
		return func(worldId byte, channelId byte, id uint32, observerId uint32, skillPossible bool, skill int8, skillId int16, skillLevel int16, multiTarget []Position, randomTimes []int32, movement Movement) error {
			ms, err := model.Fold(model.FixedProvider(movement.Elements), MovementSummaryProvider, FoldMovement)()
			if err != nil {
				return err
			}
			GetMonsterRegistry().MoveMonster(t, id, ms.X, ms.Y, ms.Stance)

			err = producer.ProviderImpl(l)(ctx)(EnvEventTopicMovement)(emitMove(worldId, channelId, id, observerId, skillPossible, skill, skillId, skillLevel, multiTarget, randomTimes, movement))
			if err != nil {
				l.WithError(err).Errorf("Unable to relay monster [%d] movement to other characters in map.", id)
				return err
			}
			return nil
		}
	}
}

func Damage(l logrus.FieldLogger) func(ctx context.Context) func(id uint32, characterId uint32, damage int64) {
	return func(ctx context.Context) func(id uint32, characterId uint32, damage int64) {
		return func(id uint32, characterId uint32, damage int64) {
			t := tenant.MustFromContext(ctx)
			m, err := GetMonsterRegistry().GetMonster(t, id)
			if err != nil {
				l.WithError(err).Errorf("Unable to get monster [%d].", id)
				return
			}
			if !m.Alive() {
				l.Errorf("Character [%d] trying to apply damage to an already dead monster [%d].", characterId, id)
				return
			}

			s, err := GetMonsterRegistry().ApplyDamage(t, characterId, damage, m.UniqueId())
			if err != nil {
				l.WithError(err).Errorf("Error applying damage to monster %d from character %d.", m.UniqueId(), characterId)
				return
			}

			if s.Killed {
				err = producer.ProviderImpl(l)(ctx)(EnvEventTopicMonsterStatus)(emitKilled(s.Monster.WorldId(), s.Monster.ChannelId(), s.Monster.MapId(), s.Monster.UniqueId(), s.Monster.MonsterId(), s.Monster.X(), s.Monster.Y(), s.CharacterId, s.Monster.DamageSummary()))
				if err != nil {
					l.WithError(err).Errorf("Monster [%d] killed, but unable to display that for the characters in the map.", s.Monster.UniqueId())
				}
				_, err = GetMonsterRegistry().RemoveMonster(t, s.Monster.UniqueId())
				if err != nil {
					l.WithError(err).Errorf("Monster [%d] killed, but not removed from registry.", s.Monster.UniqueId())
				}
				return
			}

			if characterId != s.Monster.ControlCharacterId() {
				dl := s.Monster.DamageLeader() == characterId
				l.Debugf("Character [%d] has become damage leader. They should now control the monster.", characterId)
				if dl {
					// TODO this stop seems superfluous
					m, err := GetById(ctx)(s.Monster.UniqueId())
					if err != nil {
						return
					}

					err = StopControl(l)(ctx)(m)
					if err != nil {
						l.WithError(err).Errorf("Unable to stop [%d] from controlling monster [%d].", s.Monster.ControlCharacterId(), s.Monster.UniqueId())
					}
					m, err = StartControl(l)(ctx)(m.UniqueId(), characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to start [%d] controlling monster [%d].", characterId, m.UniqueId())
					}
				}
			}

			// TODO broadcast HP bar update
		}
	}
}

func DestroyInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32) error {
			return model.ForEachSlice(model.SliceMap[Model, uint32](IdTransformer)(ByMapProvider(ctx)(worldId, channelId, mapId))(model.ParallelMap()), Destroy(l)(ctx), model.ParallelExecute())
		}
	}
}

func IdTransformer(m Model) (uint32, error) {
	return m.UniqueId(), nil
}

func Teardown(l logrus.FieldLogger) func() {
	return func() {
		ctx, span := otel.GetTracerProvider().Tracer("atlas-monsters").Start(context.Background(), "teardown")
		defer span.End()

		err := DestroyAll(l, ctx)
		if err != nil {
			l.WithError(err).Errorf("Error destroying all monsters on teardown.")
		}
	}
}

func Controlled(m Model) bool {
	return m.ControlCharacterId() != 0
}

func NotControlled(m Model) bool {
	return m.ControlCharacterId() == 0
}

func IsControlledBy(id uint32) model.Filter[Model] {
	return func(m Model) bool {
		return m.ControlCharacterId() == id
	}
}
