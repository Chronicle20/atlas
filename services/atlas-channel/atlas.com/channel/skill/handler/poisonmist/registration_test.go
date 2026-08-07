package poisonmist_test

import (
	"testing"

	channelhandler "atlas-channel/skill/handler"
	_ "atlas-channel/skill/handler/registrations"

	"github.com/stretchr/testify/require"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// TestRegistration asserts the handler is reachable through the identity
// registry once `registrations` is imported (task-187 dispatch contract).
//
// This lives in an external (poisonmist_test) package rather than the
// internal poisonmist test file: registrations blank-imports poisonmist, so
// an internal test file (package poisonmist) importing registrations would
// create an import cycle. An external test package has no such cycle -- it
// depends on both poisonmist and registrations without being either.
func TestRegistration(t *testing.T) {
	_, ok := channelhandler.Lookup(skill2.FirePoisonMagicianPoisonMist)
	require.True(t, ok, "poisonmist handler not registered")
}
