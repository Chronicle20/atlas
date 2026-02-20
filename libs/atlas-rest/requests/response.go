package requests

import (
	"github.com/jtumidanski/api2go/jsonapi"
)

func unmarshalResponse[A any](body []byte) (A, error) {
	var result A
	err := jsonapi.Unmarshal(body, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}
