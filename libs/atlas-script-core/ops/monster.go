package ops

import (
	"github.com/google/uuid"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const opSpawnMonster = "spawn_monster"

// SpawnMonster builds a SpawnMonster step.
//
// Parameters:
//   - monsterId (required) the monster template id.
//   - mapId     (optional) defaults to the target's map.
//   - x, y      (optional) default to the target's position when it carries one
//     (reactor-actions passes the reactor's coordinates), otherwise 0.
//   - count     (optional) defaults to 1.
//   - team      (optional) defaults to 0.
//
// Every parse failure is a hard error (FR-15). Instance is taken from the
// target only when the effective map id equals the target's map id; a spawn
// aimed at a different map carries uuid.Nil, because the target's instance
// belongs to the current map's field and would address a field that does not
// exist (OQ-3).
func SpawnMonster(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	monsterIdInt, err := requiredInt(p, r, characterId, opSpawnMonster, "monsterId")
	if err != nil {
		return Step{}, err
	}
	monsterId, err := rangedUint32(opSpawnMonster, "monsterId", monsterIdInt)
	if err != nil {
		return Step{}, err
	}

	mapIdInt, err := optionalInt(p, r, characterId, opSpawnMonster, "mapId", int(t.Field().MapId()))
	if err != nil {
		return Step{}, err
	}
	mapIdU, err := rangedUint32(opSpawnMonster, "mapId", mapIdInt)
	if err != nil {
		return Step{}, err
	}
	mapId := _map.Id(mapIdU)

	defX, defY, _ := t.Position()
	xInt, err := optionalInt(p, r, characterId, opSpawnMonster, "x", int(defX))
	if err != nil {
		return Step{}, err
	}
	x, err := rangedInt16(opSpawnMonster, "x", xInt)
	if err != nil {
		return Step{}, err
	}
	yInt, err := optionalInt(p, r, characterId, opSpawnMonster, "y", int(defY))
	if err != nil {
		return Step{}, err
	}
	y, err := rangedInt16(opSpawnMonster, "y", yInt)
	if err != nil {
		return Step{}, err
	}

	count, err := optionalInt(p, r, characterId, opSpawnMonster, "count", 1)
	if err != nil {
		return Step{}, err
	}

	teamInt, err := optionalInt(p, r, characterId, opSpawnMonster, "team", 0)
	if err != nil {
		return Step{}, err
	}
	team, err := rangedInt8(opSpawnMonster, "team", teamInt)
	if err != nil {
		return Step{}, err
	}

	instance := uuid.Nil
	if mapId == t.Field().MapId() {
		instance = t.Field().Instance()
	}

	return newStep(saga.SpawnMonster, saga.SpawnMonsterPayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		ChannelId:   t.Field().ChannelId(),
		MapId:       mapId,
		Instance:    instance,
		MonsterId:   monsterId,
		X:           x,
		Y:           y,
		Team:        team,
		Count:       count,
	}), nil
}
