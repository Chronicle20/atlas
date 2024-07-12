package requests

import (
	"github.com/sirupsen/logrus"
	"net/http"
)

//goland:noinspection GoUnusedExportedFunction
func MakePatchRequest[A any](url string, i interface{}, configurators ...Configurator) Request[A] {
	return func(l logrus.FieldLogger) (A, error) {
		return createOrUpdate[A](l)(http.MethodPatch)(url, i, configurators...)
	}
}
