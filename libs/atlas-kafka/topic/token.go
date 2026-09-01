package topic

// Token is the NAME OF THE ENVIRONMENT VARIABLE that carries a topic's
// per-environment name -- never the topic name itself. The distinction is
// load-bearing: overlays suffix the name, the manifest carries only the
// token, and every function that takes a resolved name keeps taking a
// plain string.
//
// Declare tokens as `X topic.Token = "COMMAND_TOPIC_Y"`. tools/topicguard
// rejects a bare literal reaching a Token parameter, and libs/atlas-kafka/gen
// collects every Token constant by type into libs/atlas-kafka/gen/topics.yaml.
type Token string
