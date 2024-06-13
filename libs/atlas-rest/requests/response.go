package requests

import (
	"github.com/manyminds/api2go/jsonapi"
	"io"
	"net/http"
)

func processResponse[A any](r *http.Response, rb *A) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	err = jsonapi.Unmarshal(body, rb)
	if err != nil {
		return err
	}

	return nil
}
