package ops

import (
	"fmt"

	"github.com/google/uuid"

	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const (
	opStartQuest          = "start_quest"
	opStageClearAttemptPq = "stage_clear_attempt_pq"
)

// QuestDefaults carries the values a caller resolves from its own state when
// the script omits them. npc-conversations reads them from the Redis
// conversation context (questId and the conversation NPC); portal-actions
// passes the zero value.
type QuestDefaults struct {
	QuestId uint32
	NpcId   uint32
}

// StartQuest builds a StartQuest step.
//
// Parameters:
//   - questId (optional in params, but required overall) falls back to
//     d.QuestId; a zero on both sides is an error.
//   - npcId   (optional) falls back to d.NpcId, else 0.
func StartQuest(p map[string]string, r Resolver, t Target, characterId uint32, d QuestDefaults) (Step, error) {
	questId := d.QuestId
	if _, ok := p["questId"]; ok {
		questIdInt, err := requiredInt(p, r, characterId, opStartQuest, "questId")
		if err != nil {
			return Step{}, err
		}
		questId, err = rangedUint32(opStartQuest, "questId", questIdInt)
		if err != nil {
			return Step{}, err
		}
	}
	if questId == 0 {
		return Step{}, missingParam(opStartQuest, "questId")
	}

	npcId := d.NpcId
	if _, ok := p["npcId"]; ok {
		npcIdInt, err := requiredInt(p, r, characterId, opStartQuest, "npcId")
		if err != nil {
			return Step{}, err
		}
		npcId, err = rangedUint32(opStartQuest, "npcId", npcIdInt)
		if err != nil {
			return Step{}, err
		}
	}

	return newStep(saga.StartQuest, saga.StartQuestPayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		QuestId:     questId,
		NpcId:       npcId,
	}), nil
}

// StageClearAttemptPq builds a StageClearAttemptPq step. It backs both the
// `stage_clear_attempt` (reactor-actions) and `stage_clear_attempt_pq`
// (npc-conversations) script operations — FR-17 keeps both dispatch names
// valid.
//
// The orchestrator branches on which field is set
// (saga-orchestrator/saga/handler.go:3717-3734), so exactly one of them must
// be: reactor-actions resolves the PQ instance over REST and passes
// instanceId; npc-conversations passes uuid.Nil and lets the orchestrator
// look the instance up from characterId.
func StageClearAttemptPq(t Target, characterId uint32, instanceId uuid.UUID) (Step, error) {
	if instanceId != uuid.Nil {
		return newStep(saga.StageClearAttemptPq, saga.StageClearAttemptPqPayload{
			InstanceId: instanceId,
		}), nil
	}
	if characterId != 0 {
		return newStep(saga.StageClearAttemptPq, saga.StageClearAttemptPqPayload{
			CharacterId: characterId,
		}), nil
	}
	return Step{}, fmt.Errorf("%s: exactly one of instanceId or characterId must be set", opStageClearAttemptPq)
}
