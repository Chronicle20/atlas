package requests

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
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

// RootUrlFor resolves the base URL for domain in the environment carried on
// ctx. Every inter-service call in Atlas already goes through an ingress
// whose address is BASE_SERVICE_URL, and every environment deploys its own
// ingress — so environment-aware routing is a namespace substitution in
// that one string, resolved from the in-memory registry with no I/O
// (design §5.1). It NEVER falls back to the baseline: an unresolvable
// environment is an error before the call is made (FR-3.5, G4).
func RootUrlFor(ctx context.Context, domain string) (string, error) {
	if val, ok := os.LookupEnv(strings.ToUpper(domain) + ServiceSuffix); ok {
		return val, nil // per-domain override wins and is never rewritten
	}
	base := os.Getenv(BaseService)
	e, _ := env.FromContext(ctx)()
	if e == "" {
		return base, nil // legacy: byte-identical to RootUrl
	}
	// EnvironmentNamespace, NOT ServiceNamespace(e, "atlas-ingress"): every
	// environment deploys its own ingress, and the record's `overrides` map
	// does not list it. ServiceNamespace would therefore fall back to the
	// baseline and send this call into main — the exact leak M3 exists to
	// close. See Task 18's TestEnvironmentNamespaceNeverFallsBackToTheBaseline.
	ns, err := env.CurrentRegistry().EnvironmentNamespace(e)
	if err != nil {
		return "", fmt.Errorf("resolving ingress for environment %q: %w", e, err)
	}
	if ns == "" {
		// A registered record with an empty Namespace is not reachable
		// through today's registry API, but the fallback this would
		// otherwise permit — silently returning the baseline URL — is
		// exactly what FR-3.5/G4 forbid. Fail closed rather than transition
		// the operation to main.
		return "", fmt.Errorf("resolving ingress for environment %q: empty namespace", e)
	}
	rewritten, err := replaceNamespace(base, ns)
	if err != nil {
		return "", fmt.Errorf("rewriting %q for namespace %q: %w", base, ns, err)
	}
	return rewritten, nil
}

// replaceNamespace rewrites the namespace label of a cluster-local URL:
//
//	http://atlas-ingress.atlas-main.svc.cluster.local:80/api/
//	                     ^^^^^^^^^^
//
// A URL that does not match that shape (a local-debug host, an external
// address) is an error rather than a silent pass-through — passing it
// through would send the operation to the wrong environment.
func replaceNamespace(raw string, ns string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	host, port, splitErr := net.SplitHostPort(u.Host)
	if splitErr != nil {
		host = u.Host
	}
	parts := strings.Split(host, ".")
	if len(parts) < 5 || parts[2] != "svc" || parts[3] != "cluster" {
		return "", fmt.Errorf("host %q is not a cluster-local FQDN", host)
	}
	parts[1] = ns
	host = strings.Join(parts, ".")
	if port != "" {
		u.Host = net.JoinHostPort(host, port)
	} else {
		u.Host = host
	}
	return u.String(), nil
}
