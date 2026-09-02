package bareliteral

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

func put(t topic.Token) {}

const untyped = "EVENT_TOPIC_UNTYPED"

func f() {
	put("EVENT_TOPIC_LITERAL") // want `bare topic literal "EVENT_TOPIC_LITERAL" reaching a topic.Token parameter; declare it as a topic.Token constant`
	put(untyped)               // want `bare topic literal "EVENT_TOPIC_UNTYPED" reaching a topic.Token parameter; declare it as a topic.Token constant`
}
