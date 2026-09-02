package bareliteral

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

func put(t topic.Token) {}

const untyped = "EVENT_TOPIC_UNTYPED"

// legacyTopicName is the shape that caused task-288: a legacy Kafka topic
// NAME, lowercase and dotted, held in an untyped constant. It reached a
// topic.Token parameter by implicit conversion and was then looked up as an
// environment variable of that literal name. The diagnostic originally
// filtered on the value matching `^[A-Z0-9_]*TOPIC[A-Z0-9_]*$`, so exactly
// the values still needing migration were the ones it could not see.
const legacyTopicName = "tenant.status"

func f() {
	put("EVENT_TOPIC_LITERAL") // want `bare topic literal "EVENT_TOPIC_LITERAL" reaching a topic.Token parameter; declare it as a topic.Token constant`
	put(untyped)               // want `bare topic literal "EVENT_TOPIC_UNTYPED" reaching a topic.Token parameter; declare it as a topic.Token constant`
	put(legacyTopicName)       // want `bare topic literal "tenant.status" reaching a topic.Token parameter; declare it as a topic.Token constant`
	put("tenant.status")       // want `bare topic literal "tenant.status" reaching a topic.Token parameter; declare it as a topic.Token constant`
}
