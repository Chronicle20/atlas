package atlas_packet

import (
	"github.com/sirupsen/logrus"
)

// ResolveCode looks up a byte code from the runtime options map.
// Options are structured as nested maps: options[property][key] = float64(code).
// Returns 99 on any lookup failure (misconfigured opcode — will likely cause a client crash).
func ResolveCode(l logrus.FieldLogger, options map[string]interface{}, property string, key string) byte {
	genericCodes, ok := options[property]
	if !ok {
		l.Errorf("Code [%s] not configured in property [%s]. Defaulting to 99 which will likely cause a client crash.", key, property)
		return 99
	}

	codes, ok := genericCodes.(map[string]interface{})
	if !ok {
		l.Errorf("Code [%s] not configured in property [%s]. Defaulting to 99 which will likely cause a client crash.", key, property)
		return 99
	}

	res, ok := codes[key].(float64)
	if !ok {
		l.Errorf("Code [%s] not configured in property [%s]. Defaulting to 99 which will likely cause a client crash.", key, property)
		return 99
	}
	return byte(res)
}
