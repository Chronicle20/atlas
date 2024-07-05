package channel

import (
	"atlas-world/tenant"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	topicTokenStatus = "TOPIC_CHANNEL_SERVICE"

	EventStatusStarted  = "STARTED"
	EventStatusShutdown = "SHUTDOWN"
)

type channelServerEvent struct {
	Tenant    tenant.Model `json:"tenant"`
	Status    string       `json:"status"`
	WorldId   byte         `json:"worldId"`
	ChannelId byte         `json:"channelId"`
	IpAddress string       `json:"ipAddress"`
	Port      int          `json:"port"`
}

func lookupTopic(l logrus.FieldLogger) func(token string) string {
	return func(token string) string {
		t, ok := os.LookupEnv(token)
		if !ok {
			l.Warnf("%s environment variable not set. Defaulting to env variable.", token)
			return token

		}
		return t
	}
}
