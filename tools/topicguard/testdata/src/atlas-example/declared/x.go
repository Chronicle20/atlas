package declared

import (
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

func put(t topic.Token) {}

const EnvSomeTopic topic.Token = "EVENT_TOPIC_SOME"

func f() {
	put(EnvSomeTopic)
}

func g() string {
	return os.Getenv(string(EnvSomeTopic))
}
