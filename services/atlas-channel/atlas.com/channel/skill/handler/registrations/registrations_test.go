package registrations_test

import (
	"testing"

	channelhandler "atlas-channel/skill/handler"
	_ "atlas-channel/skill/handler/registrations"

	"github.com/stretchr/testify/require"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// TestPoisonMistRegistered asserts the Poison Mist handler is reachable
// through the identity registry once `registrations` is blank-imported
// (task-187 dispatch contract; task-200).
//
// This lives in the registrations package's own external test package
// (registrations_test) rather than inside poisonmist's test suite: a test
// under skill/handler/poisonmist compiles poisonmist's own production
// code -- including its self-registering init() -- into the test binary
// regardless of what registrations.go imports, so it can never observe a
// missing or broken blank import there. This package does NOT import
// poisonmist directly; Lookup only succeeds because registrations.go's
// blank import runs poisonmist's init(), so removing that import genuinely
// fails this test.
func TestPoisonMistRegistered(t *testing.T) {
	_, ok := channelhandler.Lookup(skill2.FirePoisonMagicianPoisonMist)
	require.True(t, ok, "poisonmist handler not registered via registrations blank import")
}
