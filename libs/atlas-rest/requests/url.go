package requests

import (
	"os"
	"strings"
)

const (
	ServiceSuffix = "_SERVICE_URL"
	BaseService   = "BASE" + ServiceSuffix
)

//goland:noinspection GoUnusedExportedFunction
func RootUrl(domain string) string {
	if val, ok := os.LookupEnv(strings.ToUpper(domain) + ServiceSuffix); ok {
		return val
	}
	return os.Getenv(BaseService)
}
