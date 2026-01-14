package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
)

// MockRoute defines a single route handler for the mock server
type MockRoute struct {
	Method   string
	Path     string
	Response interface{}
	Status   int
}

// MockServerConfig holds configuration for the mock HTTP server
type MockServerConfig struct {
	Routes []MockRoute
}

// NewMockServer creates a new HTTP test server with the specified routes
// Routes are matched by method and path prefix
func NewMockServer(config MockServerConfig) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, route := range config.Routes {
			if r.Method == route.Method && matchPath(r.URL.Path, route.Path) {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				if route.Status != 0 {
					w.WriteHeader(route.Status)
				} else {
					w.WriteHeader(http.StatusOK)
				}

				if route.Response != nil {
					if str, ok := route.Response.(string); ok {
						_, _ = w.Write([]byte(str))
					} else {
						_ = json.NewEncoder(w).Encode(route.Response)
					}
				}
				return
			}
		}

		// No matching route found
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"status":"404","title":"Not Found"}]}`))
	}))
}

// matchPath checks if the request path matches the route path
// Supports both exact matches and prefix matches (when route ends with *)
func matchPath(requestPath, routePath string) bool {
	if strings.HasSuffix(routePath, "*") {
		prefix := strings.TrimSuffix(routePath, "*")
		return strings.HasPrefix(requestPath, prefix)
	}
	return requestPath == routePath
}

// JSONAPIDocument represents a JSON:API document structure
type JSONAPIDocument struct {
	Data     interface{}       `json:"data"`
	Included []interface{}     `json:"included,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
}

// JSONAPIResource represents a single JSON:API resource
type JSONAPIResource struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attributes"`
}

// NewJSONAPIDocument creates a JSON:API document with the given data
func NewJSONAPIDocument(data interface{}) JSONAPIDocument {
	return JSONAPIDocument{
		Data: data,
	}
}

// NewJSONAPIResource creates a JSON:API resource
func NewJSONAPIResource(resourceType, id string, attributes map[string]interface{}) JSONAPIResource {
	return JSONAPIResource{
		Type:       resourceType,
		ID:         id,
		Attributes: attributes,
	}
}

// NewJSONAPIListDocument creates a JSON:API document with an array of resources
func NewJSONAPIListDocument(resources []JSONAPIResource) JSONAPIDocument {
	data := make([]interface{}, len(resources))
	for i, r := range resources {
		data[i] = r
	}
	return JSONAPIDocument{
		Data: data,
	}
}

// NewJSONAPIErrorResponse creates a JSON:API error response
func NewJSONAPIErrorResponse(status, title, detail string) string {
	return `{"errors":[{"status":"` + status + `","title":"` + title + `","detail":"` + detail + `"}]}`
}

// NewJSONAPINotFoundResponse creates a standard 404 error response
func NewJSONAPINotFoundResponse() string {
	return NewJSONAPIErrorResponse("404", "Not Found", "The requested resource was not found")
}

// CreateCharacterResponse creates a mock character JSON:API response
func CreateCharacterResponse(id uint32, name string, level byte, jobId uint16, mapId uint32) string {
	return `{
		"data": {
			"type": "characters",
			"id": "` + uintToString(id) + `",
			"attributes": {
				"accountId": 1,
				"worldId": 0,
				"name": "` + name + `",
				"level": ` + byteToString(level) + `,
				"experience": 0,
				"gachaponExperience": 0,
				"strength": 4,
				"dexterity": 4,
				"intelligence": 4,
				"luck": 4,
				"hp": 50,
				"maxHp": 50,
				"mp": 5,
				"maxMp": 5,
				"meso": 0,
				"hpMpUsed": 0,
				"jobId": ` + uint16ToString(jobId) + `,
				"skinColor": 0,
				"gender": 0,
				"fame": 0,
				"hair": 30000,
				"face": 20000,
				"ap": 0,
				"sp": "0,0,0,0,0,0,0,0,0,0",
				"mapId": ` + uintToString(mapId) + `,
				"spawnPoint": 0,
				"gm": 0,
				"x": 0,
				"y": 0,
				"stance": 0
			}
		}
	}`
}

// helper functions for string conversion
func uintToString(v uint32) string {
	return itoa(int(v))
}

func uint16ToString(v uint16) string {
	return itoa(int(v))
}

func byteToString(v byte) string {
	return itoa(int(v))
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
