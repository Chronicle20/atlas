package session

import "github.com/google/uuid"

const (
	EnvCommandTopic = "COMMAND_TOPIC_ACCOUNT_SESSION"

	CommandIssuerInternal = "INTERNAL"
	CommandIssuerLogin    = "LOGIN"
	CommandIssuerChannel  = "CHANNEL"

	CommandTypeCreate = "CREATE"
	CommandTypeLogout = "LOGOUT"
)

type command[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Issuer    string    `json:"author"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type createCommandBody struct {
	AccountName string `json:"accountName"`
	Password    string `json:"password"`
	IPAddress   string `json:"ipAddress"`
}

type logoutCommandBody struct {
}
