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

// exempt reproduces task-276 Fix E's regression: KAFKA_TOPIC_MANIFEST_PATH
// lexically matches rawEnvTopicPattern but names a filesystem path, not a
// Kafka topic, so it is allowlisted (allowlist.txt) and must not fire.
func exempt() string {
	return os.Getenv("KAFKA_TOPIC_MANIFEST_PATH")
}

// stillFires proves the allowlist is narrow: a genuine raw topic-name env
// read for a name NOT in allowlist.txt must still fire, even though it
// shares the "TOPIC" substring with the exempted entry above.
func stillFires() string {
	return os.Getenv("COMMAND_TOPIC_ACCOUNT") // want `raw environment read of topic token "COMMAND_TOPIC_ACCOUNT"; reference a topic.Token constant instead`
}
