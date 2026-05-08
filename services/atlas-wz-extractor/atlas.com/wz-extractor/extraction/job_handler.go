package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/rest"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type jobUnitJSON struct {
	WzFile      string  `json:"wzFile"`
	Status      string  `json:"status"`
	StartedAt   *string `json:"startedAt"`
	CompletedAt *string `json:"completedAt"`
	Error       *string `json:"error"`
}

type jobAttributesJSON struct {
	TenantId       string        `json:"tenantId"`
	Region         string        `json:"region"`
	MajorVersion   uint16        `json:"majorVersion"`
	MinorVersion   uint16        `json:"minorVersion"`
	Status         string        `json:"status"`
	XmlOnly        bool          `json:"xmlOnly"`
	ImagesOnly     bool          `json:"imagesOnly"`
	UnitsTotal     int           `json:"unitsTotal"`
	UnitsCompleted int           `json:"unitsCompleted"`
	UnitsFailed    int           `json:"unitsFailed"`
	CreatedAt      string        `json:"createdAt"`
	UpdatedAt      string        `json:"updatedAt"`
	CompletedAt    *string       `json:"completedAt"`
	Units          []jobUnitJSON `json:"units"`
}

type jobResource struct {
	Type       string            `json:"type"`
	Id         string            `json:"id"`
	Attributes jobAttributesJSON `json:"attributes"`
}

type jobEnvelope struct {
	Data jobResource `json:"data"`
}

func handleJobStatus(store job.Store) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			jobId := vars["jobId"]
			j, units, err := store.Get(d.Context(), jobId)
			if errors.Is(err, job.ErrNotFound) {
				http.Error(w, "job not found", http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Error("Get job failed")
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			fmtTime := func(t time.Time) string {
				if t.IsZero() {
					return ""
				}
				return t.UTC().Format(time.RFC3339)
			}
			optTime := func(t time.Time) *string {
				if t.IsZero() {
					return nil
				}
				s := t.UTC().Format(time.RFC3339)
				return &s
			}

			ujs := make([]jobUnitJSON, 0, len(units))
			for _, u := range units {
				var errPtr *string
				if u.ErrorMessage() != "" {
					e := u.ErrorMessage()
					errPtr = &e
				}
				ujs = append(ujs, jobUnitJSON{
					WzFile:      u.WzFile(),
					Status:      string(u.Status()),
					StartedAt:   optTime(u.StartedAt()),
					CompletedAt: optTime(u.CompletedAt()),
					Error:       errPtr,
				})
			}

			env := jobEnvelope{
				Data: jobResource{
					Type: "wzExtractionJob",
					Id:   j.Id(),
					Attributes: jobAttributesJSON{
						TenantId:       j.TenantId(),
						Region:         j.Region(),
						MajorVersion:   j.MajorVersion(),
						MinorVersion:   j.MinorVersion(),
						Status:         string(j.Status()),
						XmlOnly:        j.XmlOnly(),
						ImagesOnly:     j.ImagesOnly(),
						UnitsTotal:     j.UnitsTotal(),
						UnitsCompleted: j.UnitsCompleted(),
						UnitsFailed:    j.UnitsFailed(),
						CreatedAt:      fmtTime(j.CreatedAt()),
						UpdatedAt:      fmtTime(j.UpdatedAt()),
						CompletedAt:    optTime(j.CompletedAt()),
						Units:          ujs,
					},
				},
			}
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_ = json.NewEncoder(w).Encode(env)
		}
	}
}
