package saga

const (
	EnvCommandTopic = "COMMAND_TOPIC_SAGA"
)

// Action types supported by saga-orchestrator
type Action string

const (
	AwardAsset      Action = "award_asset"
	AwardExperience Action = "award_experience"
	AwardMesos      Action = "award_mesos"
	AwardFame       Action = "award_fame"
	CreateSkill     Action = "create_skill"
	UpdateSkill     Action = "update_skill"
	ConsumeItem     Action = "destroy_asset"
)

// Status represents the status of a saga step
type Status string

const (
	Pending   Status = "pending"
	Completed Status = "completed"
	Failed    Status = "failed"
)

// Type represents the type of saga
type Type string

const (
	QuestStart       Type = "quest_start"
	QuestComplete    Type = "quest_complete"
	QuestRestoreItem Type = "quest_restore_item"
)
