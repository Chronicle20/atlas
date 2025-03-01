package account

const (
	EnvCommandTopicCreateAccount = "COMMAND_TOPIC_CREATE_ACCOUNT"
)

type createCommand struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}
