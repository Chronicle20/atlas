package field

import (
	"fmt"
	"strings"
)

const (
	IdFormat = "%d:%d:%d:%s"
)

type Id string

// ObjectKind selects which clientbound opcode carries a named field-object
// state change. Both kinds resolve to the same client dictionary
// (CMapLoadable::m_mNamedObj) and the same client function
// (CMapLoadable::SetObjectState), so the distinction is a transport
// preference on the server, not a behavioural one on the client.
type ObjectKind string

const (
	ObjectKindEnvironment ObjectKind = "ENVIRONMENT"
	ObjectKindObstacle    ObjectKind = "OBSTACLE"
)

// ParseObjectKind resolves a wire or script string to an ObjectKind. A blank
// value defaults to ObjectKindEnvironment: no authored script in the repo
// specifies a kind today, and the upstream conversion source
// (the Cosmic moveEnvironment script call) is the SetObjectState path.
func ParseObjectKind(s string) (ObjectKind, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "":
		return ObjectKindEnvironment, nil
	case string(ObjectKindEnvironment):
		return ObjectKindEnvironment, nil
	case string(ObjectKindObstacle):
		return ObjectKindObstacle, nil
	default:
		return "", fmt.Errorf("unrecognized object kind [%s]", s)
	}
}
