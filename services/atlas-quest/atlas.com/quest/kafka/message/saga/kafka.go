package saga

import (
	sharedsaga "github.com/Chronicle20/atlas-saga"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_SAGA"
)

// Re-export types from atlas-saga shared library
type (
	Type   = sharedsaga.Type
	Status = sharedsaga.Status
	Action = sharedsaga.Action
)

// Re-export constants from atlas-saga shared library
const (
	Pending   = sharedsaga.Pending
	Completed = sharedsaga.Completed
	Failed    = sharedsaga.Failed

	AwardAsset      = sharedsaga.AwardAsset
	AwardExperience = sharedsaga.AwardExperience
	AwardMesos      = sharedsaga.AwardMesos
	AwardFame       = sharedsaga.AwardFame
	CreateSkill     = sharedsaga.CreateSkill
	UpdateSkill     = sharedsaga.UpdateSkill

	// ConsumeItem maps to DestroyAsset action on the wire
	ConsumeItem = sharedsaga.DestroyAsset

	QuestStart       = sharedsaga.QuestStart
	QuestComplete    = sharedsaga.QuestComplete
	QuestRestoreItem = sharedsaga.QuestRestoreItem
)
