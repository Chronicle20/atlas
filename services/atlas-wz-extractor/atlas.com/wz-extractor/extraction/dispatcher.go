package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	mext "atlas-wz-extractor/kafka/message/extraction"
	"atlas-wz-extractor/rest"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type producerProvider func(ctx context.Context) func(token string) producer.MessageProducer

const (
	jobTTLSeconds  = 24 * 60 * 60
	lockRefreshDiv = 3 // refresh every TTL/3
)

func handleExtract(p Processor, store job.Store, tl *lock.TenantLock, prod producerProvider, dirs Dirs) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			xmlOnly := r.URL.Query().Get("xmlOnly") == "true"
			imagesOnly := r.URL.Query().Get("imagesOnly") == "true"
			ll := d.Logger().WithFields(logrus.Fields{
				"tenantId": t.Id().String(),
				"region":   t.Region(),
				"version":  TenantPath(t),
			})

			tenantInput := filepath.Join(dirs.InputDir, TenantPath(t))
			wzFiles, err := filepath.Glob(filepath.Join(tenantInput, "*.wz"))
			if err != nil {
				ll.WithError(err).Error("glob failed")
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if len(wzFiles) == 0 {
				http.Error(w, "no .wz files staged for tenant; upload via PATCH /api/wz/input first", http.StatusBadRequest)
				return
			}

			jobId := uuid.NewString()
			lockKey := job.LockKey(t.Id().String(), t.Region(), t.MajorVersion(), t.MinorVersion())

			acquired, err := tl.Acquire(d.Context(), lockKey, jobId)
			if err != nil {
				ll.WithError(err).Error("redis lock Acquire failed")
				http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
				return
			}
			if !acquired {
				http.Error(w, "another extraction is already in flight for this tenant", http.StatusConflict)
				return
			}

			// wipeCharacterCache must run once before any unit message is published.
			if !xmlOnly {
				imgOutPath := filepath.Join(dirs.OutputImgDir, TenantPath(t))
				if err := wipeCharacterCache(imgOutPath); err != nil {
					ll.WithError(err).Warn("Unable to wipe character cache.")
				}
			}

			now := time.Now().UTC()
			wzNames := make([]string, 0, len(wzFiles))
			units := make([]job.Unit, 0, len(wzFiles))
			for _, full := range wzFiles {
				name := filepath.Base(full)
				wzNames = append(wzNames, name)
				units = append(units, job.NewUnitBuilder().SetWzFile(name).SetStatus(job.UnitPending).Build())
			}

			j := job.NewJobBuilder().
				SetId(jobId).
				SetTenantId(t.Id().String()).
				SetRegion(t.Region()).
				SetMajorVersion(t.MajorVersion()).SetMinorVersion(t.MinorVersion()).
				SetStatus(job.JobPending).
				SetUnitsTotal(len(wzNames)).
				SetXmlOnly(xmlOnly).SetImagesOnly(imagesOnly).
				SetCreatedAt(now).SetUpdatedAt(now).Build()

			if err := store.Create(d.Context(), j, units, jobTTLSeconds); err != nil {
				ll.WithError(err).Error("Create job failed")
				_ = tl.Release(d.Context(), lockKey, jobId)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if err := store.MarkJobRunning(d.Context(), jobId); err != nil {
				ll.WithError(err).Error("MarkJobRunning failed")
				_ = tl.Release(d.Context(), lockKey, jobId)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			// Publish one START_EXTRACTION_UNIT per WZ file.
			emit := prod(d.Context())(mext.EnvCommandTopic)
			publishErr := error(nil)
			published := 0
			for _, name := range wzNames {
				prov := mext.StartExtractionUnitProvider(jobId, name, xmlOnly, imagesOnly)
				if err := emit(prov); err != nil {
					publishErr = err
					break
				}
				published++
			}
			if publishErr != nil {
				ll.WithError(publishErr).WithField("publishedSoFar", published).Error("producer error after partial publish")
				_ = store.MarkUnitsSkippedByStatus(d.Context(), jobId, []job.UnitStatus{job.UnitPending})
				_, _ = store.MarkJobTerminal(d.Context(), jobId, job.JobFailed)
				_ = tl.Release(d.Context(), lockKey, jobId)
				http.Error(w, "kafka publish failed", http.StatusInternalServerError)
				return
			}

			// Spawn refresh goroutine; lifetime bounded by lock TTL or job-poll loop.
			startLockRefresh(d.Logger(), store, tl, lockKey, jobId, tl.TTL()/lockRefreshDiv)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jobId":      jobId,
				"unitsTotal": len(wzNames),
				"status":     "running",
			})

			_ = p
		}
	}
}

// startLockRefresh runs a goroutine that periodically refreshes the tenant
// lock until the job reaches a terminal status (or the lock disappears).
func startLockRefresh(l logrus.FieldLogger, store job.Store, tl *lock.TenantLock, lockKey, jobId string, period time.Duration) {
	if period <= 0 {
		return
	}
	go func() {
		ctx := context.Background()
		ticker := time.NewTicker(period)
		defer ticker.Stop()
		for {
			<-ticker.C
			j, _, err := store.Get(ctx, jobId)
			if err != nil {
				return
			}
			if j.Status() != job.JobRunning && j.Status() != job.JobPending {
				return
			}
			if err := tl.Refresh(ctx, lockKey, jobId); err != nil {
				l.WithError(err).Warn("tenant-lock refresh failed; will exit refresh loop")
				return
			}
		}
	}()
}

// silence unused-import warnings if the producer package's types are only
// referenced indirectly through the type alias.
var _ producer.MessageProducer
var _ kafka.Message
var _ = model.FixedProvider[[]kafka.Message]
