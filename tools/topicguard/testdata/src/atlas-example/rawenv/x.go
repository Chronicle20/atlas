package rawenv

import "os"

func f() string {
	return os.Getenv("EVENT_TOPIC_RAW") // want `raw environment read of topic token "EVENT_TOPIC_RAW"; reference a topic.Token constant instead`
}

func g() (string, bool) {
	return os.LookupEnv("COMMAND_TOPIC_RAW") // want `raw environment read of topic token "COMMAND_TOPIC_RAW"; reference a topic.Token constant instead`
}

func ok() string {
	return os.Getenv("REST_PORT")
}
