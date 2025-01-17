package account

const (
	EnvCommandTopicCreateAccount = "COMMAND_TOPIC_CREATE_ACCOUNT"

	EnvEventTopicAccountStatus  = "EVENT_TOPIC_ACCOUNT_STATUS"
	EventAccountStatusCreated   = "CREATED"
	EventAccountStatusLoggedIn  = "LOGGED_IN"
	EventAccountStatusLoggedOut = "LOGGED_OUT"
)

type createCommand struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type statusEvent struct {
	AccountId uint32 `json:"account_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}
