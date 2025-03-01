package account

import "github.com/google/uuid"

const (
	EnvCommandTopicCreateAccount = "COMMAND_TOPIC_CREATE_ACCOUNT"

	EnvCommandSessionTopic = "COMMAND_TOPIC_ACCOUNT_SESSION"

	SessionCommandIssuerInternal = "INTERNAL"
	SessionCommandIssuerLogin    = "LOGIN"
	SessionCommandIssuerChannel  = "CHANNEL"

	SessionCommandTypeCreate        = "CREATE"
	SessionCommandTypeProgressState = "PROGRESS_STATE"
	SessionCommandTypeLogout        = "LOGOUT"
)

type createCommand struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type sessionCommand[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Issuer    string    `json:"author"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type createSessionCommandBody struct {
	AccountName string `json:"accountName"`
	Password    string `json:"password"`
	IPAddress   string `json:"ipAddress"`
}

type progressStateSessionCommandBody struct {
	State  uint8       `json:"state"`
	Params interface{} `json:"params"`
}

type logoutSessionCommandBody struct {
}
