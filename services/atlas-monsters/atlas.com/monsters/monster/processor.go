package monster

import (
	"atlas-monsters/kafka/producer"
	_map "atlas-monsters/map"
	"atlas-monsters/monster/information"
	"atlas-monsters/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func byIdProvider(_ logrus.FieldLogger, _ opentracing.Span, tenant tenant.Model) func(monsterId uint32) model.Provider[Model] {
	return func(monsterId uint32) model.Provider[Model] {
		return func() (Model, error) {
			return GetMonsterRegistry().GetMonster(tenant, monsterId)
		}
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(monsterId uint32) (Model, error) {
	return func(monsterId uint32) (Model, error) {
		return byIdProvider(l, span, tenant)(monsterId)()
	}
}

func CreateMonster(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, input RestModel) (Model, error) {
	return func(worldId byte, channelId byte, mapId uint32, input RestModel) (Model, error) {
		l.Debugf("Attempting to create monster [%d] in world [%d] channel [%d] map [%d].", input.MonsterId, worldId, channelId, mapId)
		ma, err := information.GetById(l, span, tenant)(input.MonsterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve information necessary to create monster [%d].", input.MonsterId)
			return Model{}, err
		}
		m := GetMonsterRegistry().CreateMonster(tenant, worldId, channelId, mapId, input.MonsterId, input.X, input.Y, input.Fh, 5, input.Team, ma.HP(), ma.MP())

		cid, err := GetControllerCandidate(l, span, tenant)(worldId, channelId, mapId)
		if err == nil {
			l.Debugf("Created monster [%d] with id [%d] will be controlled by [%d].", m.MonsterId(), m.UniqueId(), cid)
			m, err = StartControl(l, span, tenant)(m.UniqueId(), cid)
			if err != nil {
				l.WithError(err).Errorf("Unable to start [%d] controlling [%d] in world [%d] channel [%d] map [%d].", cid, m.UniqueId(), m.WorldId(), m.ChannelId(), m.MapId())
			}
		}

		l.Debugf("Created monster [%d] in world [%d] channel [%d] map [%d]. Emitting Monster Status.", input.MonsterId, worldId, channelId, mapId)
		_ = producer.ProviderImpl(l)(span)(EnvEventTopicMonsterStatus)(emitCreated(tenant, m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId()))
		return m, nil
	}
}

func FindNextController(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(uniqueId uint32) {
	return func(uniqueId uint32) {
		m, err := GetById(l, span, tenant)(uniqueId)
		if err != nil {
			return
		}

		cid, err := GetControllerCandidate(l, span, tenant)(m.WorldId(), m.ChannelId(), m.MapId())
		if err == nil {
			_, err = StartControl(l, span, tenant)(m.UniqueId(), cid)
			if err != nil {
				l.WithError(err).Errorf("Unable to start [%d] controlling [%d] in world [%d] channel [%d] map [%d].", cid, m.UniqueId(), m.WorldId(), m.ChannelId(), m.MapId())
			}
		}
	}
}

func GetControllerCandidate(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) (uint32, error) {
	return func(worldId byte, channelId byte, mapId uint32) (uint32, error) {
		l.Debugf("Identifying controller candidate for monsters in world [%d] channel [%d] map [%d].", worldId, channelId, mapId)
		ids, err := _map.GetCharacterIdsInMap(l, span, tenant)(worldId, channelId, mapId)
		if err != nil {
			l.Debugf("No characters are found in this map. No controller to assign.")
			return 0, err
		}

		var controlCounts map[uint32]int
		controlCounts = make(map[uint32]int)

		for _, id := range ids {
			controlCounts[id] = 0
		}

		ms := GetMonsterRegistry().GetMonstersInMap(tenant, worldId, channelId, mapId)
		for _, m := range ms {
			if m.ControlCharacterId() != 0 {
				controlCounts[m.ControlCharacterId()] += 1
			}
		}

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

func StartControl(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(uniqueId uint32, controllerId uint32) (Model, error) {
	return func(uniqueId uint32, controllerId uint32) (Model, error) {
		m, err := GetById(l, span, tenant)(uniqueId)
		if err != nil {
			return Model{}, err
		}

		if m.ControlCharacterId() != 0 {
			m, err = StopControl(l, span, tenant)(m.UniqueId())
			if err != nil {
				return Model{}, err
			}
		}
		m, err = GetMonsterRegistry().ControlMonster(tenant, m.UniqueId(), controllerId)
		if err == nil {
			_ = producer.ProviderImpl(l)(span)(EnvEventTopicMonsterStatus)(emitStartControl(tenant, m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId(), m.ControlCharacterId()))
		}
		return m, err
	}
}

func StopControl(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(uniqueId uint32) (Model, error) {
	return func(uniqueId uint32) (Model, error) {
		m, err := GetById(l, span, tenant)(uniqueId)
		if err != nil {
			return Model{}, err
		}

		m, err = GetMonsterRegistry().ClearControl(tenant, m.UniqueId())
		if err == nil {
			_ = producer.ProviderImpl(l)(span)(EnvEventTopicMonsterStatus)(emitStopControl(tenant, m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId(), m.ControlCharacterId()))
		}
		return m, err
	}
}

func DestroyAll(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) {
	ms := GetMonsterRegistry().GetMonsters()
	for _, x := range ms {
		Destroy(l, span, tenant)(x.UniqueId())
	}
}

func Destroy(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(uniqueId uint32) {
	return func(uniqueId uint32) {
		m, err := GetMonsterRegistry().RemoveMonster(tenant, uniqueId)
		if err == nil {
			_ = producer.ProviderImpl(l)(span)(EnvEventTopicMonsterStatus)(emitDestroyed(tenant, m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId()))
		}
	}
}

func Move(_ logrus.FieldLogger, _ opentracing.Span, tenant tenant.Model) func(id uint32, x int, y int, stance int) {
	return func(id uint32, x int, y int, stance int) {
		GetMonsterRegistry().MoveMonster(tenant, id, x, y, stance)
	}
}

func Damage(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32, characterId uint32, damage int64) {
	return func(id uint32, characterId uint32, damage int64) {
		m, err := GetMonsterRegistry().GetMonster(tenant, id)
		if err != nil {
			l.WithError(err).Errorf("Unable to get monster [%d].", id)
			return
		}
		if !m.Alive() {
			l.Errorf("Character [%d] trying to apply damage to an already dead monster [%d].", characterId, id)
			return
		}

		s, err := GetMonsterRegistry().ApplyDamage(tenant, characterId, damage, m.UniqueId())
		if err != nil {
			l.WithError(err).Errorf("Error applying damage to monster %d from character %d.", m.UniqueId(), characterId)
			return
		}

		if s.Killed {
			err = producer.ProviderImpl(l)(span)(EnvEventTopicMonsterStatus)(emitKilled(tenant, s.Monster.WorldId(), s.Monster.ChannelId(), s.Monster.MapId(), s.Monster.UniqueId(), s.Monster.MonsterId(), s.Monster.X(), s.Monster.Y(), s.CharacterId, s.Monster.DamageSummary()))
			if err != nil {
				l.WithError(err).Errorf("Monster [%d] killed, but unable to display that for the characters in the map.", s.Monster.UniqueId())
			}
			_, err = GetMonsterRegistry().RemoveMonster(tenant, s.Monster.UniqueId())
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
				m, err = StopControl(l, span, tenant)(s.Monster.UniqueId())
				if err != nil {
					l.WithError(err).Errorf("Unable to stop [%d] from controlling monster [%d].", s.Monster.ControlCharacterId(), s.Monster.UniqueId())
				}
				m, err = StartControl(l, span, tenant)(s.Monster.UniqueId(), characterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to start [%d] controlling monster [%d].", characterId, s.Monster.UniqueId())
				}
			}
		}

		// TODO broadcast HP bar update
	}
}
