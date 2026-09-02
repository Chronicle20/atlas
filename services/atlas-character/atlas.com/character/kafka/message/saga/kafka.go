// Package saga carries the COMMAND_TOPIC_SAGA topic token. atlas-character
// produces onto it to consume and to refund the cash coupon backing a pending
// change (design §3.10); the saga bodies themselves come from the shared
// libs/atlas-saga contract, so there is nothing to mirror here.
package saga

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_SAGA"
)
