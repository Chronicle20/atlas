package requests

import (
	"github.com/manyminds/api2go/jsonapi"
	"io"
	"net/http"
)

func processResponse[A any](r *http.Response) (A, error) {
	var result A
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return result, err
	}
	defer r.Body.Close()

	err = jsonapi.Unmarshal(body, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
