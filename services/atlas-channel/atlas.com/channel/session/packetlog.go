package session

import (
	"os"
	"strings"
	"sync"
)

// packetWriteLogEnv names the environment variable that selects which
// clientbound writers get their plaintext bytes logged as they are announced.
//
// Clientbound packets are otherwise invisible: the channel logs every
// serverbound read and every Kafka message it consumes, but nothing on the
// write side, so "the server decided to send X" and "the client received X"
// cannot be distinguished from logs alone. That gap is what forces a
// client-behaviour question (why is the client not clearing a flag the server
// believes it just cleared?) to be answered by inference instead of evidence.
//
// The value is a comma-separated list of writer names (the same names used in
// the tenant socket config, e.g. "StatChanged,CharacterBuffCancel"), or "*" for
// every writer. Unset — the default everywhere including production — disables
// the logging entirely and costs one atomic-ish map read per announce.
//
// "*" on a busy channel is loud: every movement broadcast, every mob spawn.
// Prefer naming the writers under investigation.
const packetWriteLogEnv = "CHANNEL_PACKET_WRITE_LOG"

var (
	packetWriteLogOnce sync.Once
	packetWriteLogAll  bool
	packetWriteLogSet  map[string]struct{}
)

// packetWriteLogEnabled reports whether writerName's encoded bytes should be
// logged. The environment is read once; changing the variable requires a
// restart, which is the same lifecycle as every other channel setting.
func packetWriteLogEnabled(writerName string) bool {
	packetWriteLogOnce.Do(func() {
		packetWriteLogAll, packetWriteLogSet = parsePacketWriteLog(os.Getenv(packetWriteLogEnv))
	})

	if packetWriteLogAll {
		return true
	}
	_, ok := packetWriteLogSet[writerName]
	return ok
}

// parsePacketWriteLog splits the raw environment value into its "everything"
// flag and its explicit writer set. Separated from the sync.Once so the parsing
// rules are testable without depending on process environment or on which test
// happened to run first.
func parsePacketWriteLog(raw string) (bool, map[string]struct{}) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}
	if raw == "*" {
		return true, nil
	}
	set := make(map[string]struct{})
	for _, name := range strings.Split(raw, ",") {
		if name = strings.TrimSpace(name); name != "" {
			set[name] = struct{}{}
		}
	}
	return false, set
}
