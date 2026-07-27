package outbox

import (
	"encoding/base64"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"gorm.io/datatypes"
)

// Header values are base64-encoded inside the stored jsonb. Tenant version
// headers are raw big-endian uint16 bytes (see atlas-kafka
// TenantHeaderDecorator): they always contain a NUL byte, which Postgres
// jsonb rejects, and may be invalid UTF-8 (e.g. version 185 = 0xB9), which
// encoding/json silently mangles to U+FFFD. Base64 keeps the round trip
// byte-exact. Keys are plain ASCII and stay unencoded.
func encodeHeaders(h map[string]string) (datatypes.JSON, error) {
	if len(h) == 0 {
		return datatypes.JSON([]byte("{}")), nil
	}
	enc := make(map[string]string, len(h))
	for k, v := range h {
		enc[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	b, err := json.Marshal(enc)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

func decodeHeaders(j datatypes.JSON) ([]kafka.Header, error) {
	if len(j) == 0 {
		return nil, nil
	}
	var enc map[string]string
	if err := json.Unmarshal(j, &enc); err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	hs := make([]kafka.Header, 0, len(enc))
	for k, v := range enc {
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}
		hs = append(hs, kafka.Header{Key: k, Value: b})
	}
	return hs, nil
}
