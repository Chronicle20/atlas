package consumer

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"
)

// DebugHandler returns an http.Handler that responds with a JSON:API document
// listing every Consumer registered on this Manager. The response is
// tenant-agnostic and read-only. Intended for mounting on a service's
// existing REST server under GET /api/debug/consumers; safe to expose on
// any internal (non-ingress) network.
func (m *Manager) DebugHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		consumers := m.Consumers()
		snapshots := make([]Snapshot, 0, len(consumers))
		for _, c := range consumers {
			snapshots = append(snapshots, c.Snapshot())
		}
		sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].Topic < snapshots[j].Topic })

		resources := make([]jsonAPIResource, 0, len(snapshots))
		for _, s := range snapshots {
			resources = append(resources, jsonAPIResource{
				Type:       "consumers",
				ID:         s.Topic,
				Attributes: snapshotToAttributes(s),
			})
		}

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(jsonAPIDocument{Data: resources})
	})
}

type jsonAPIDocument struct {
	Data []jsonAPIResource `json:"data"`
}

type jsonAPIResource struct {
	Type       string              `json:"type"`
	ID         string              `json:"id"`
	Attributes debugAttributes     `json:"attributes"`
}

type debugAttributes struct {
	Name                string    `json:"name"`
	Topic               string    `json:"topic"`
	GroupID             string    `json:"groupId"`
	Brokers             []string  `json:"brokers"`
	AliveSince          time.Time `json:"aliveSince"`
	LastFetchAt         time.Time `json:"lastFetchAt"`
	LastErrorAt         time.Time `json:"lastErrorAt"`
	LastError           string    `json:"lastError"`
	RecreateCount       int       `json:"recreateCount"`
	HandlerCount        int       `json:"handlerCount"`
	LastTimeoutAt       time.Time `json:"lastTimeoutAt"`
	ConsecutiveTimeouts int       `json:"consecutiveTimeouts"`
}

func snapshotToAttributes(s Snapshot) debugAttributes {
	return debugAttributes{
		Name:                s.Name,
		Topic:               s.Topic,
		GroupID:             s.GroupID,
		Brokers:             s.Brokers,
		AliveSince:          s.AliveSince,
		LastFetchAt:         s.LastFetchAt,
		LastErrorAt:         s.LastErrorAt,
		LastError:           s.LastError,
		RecreateCount:       s.RecreateCount,
		HandlerCount:        s.HandlerCount,
		LastTimeoutAt:       s.LastTimeoutAt,
		ConsecutiveTimeouts: s.ConsecutiveTimeouts,
	}
}
