package _map

import (
	"context"
	"encoding/json"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"atlas-data/document"
)

const backfillPageSize = 500
const backfillProgressEvery = 5000

type BackfillResult struct {
	Processed  int   `json:"processed"`
	Inserted   int   `json:"inserted"`
	Updated    int   `json:"updated"`
	DurationMs int64 `json:"duration_ms"`
}

func Backfill(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) (BackfillResult, error) {
	return func(ctx context.Context) func(db *gorm.DB) (BackfillResult, error) {
		return func(db *gorm.DB) (BackfillResult, error) {
			start := time.Now()
			res := BackfillResult{}
			// Run without tenant filtering; we scan all tenants.
			scan := db.WithContext(database.WithoutTenantFilter(ctx))

			offset := 0
			for {
				var batch []document.Entity
				if err := scan.
					Where("type = ?", "MAP").
					Order("tenant_id, document_id").
					Limit(backfillPageSize).
					Offset(offset).
					Find(&batch).Error; err != nil {
					return res, err
				}
				if len(batch) == 0 {
					break
				}

				rows := make([]SearchIndexEntity, 0, len(batch))
				for _, doc := range batch {
					name, street, err := extractNameStreet(doc.Content)
					if err != nil {
						l.WithError(err).Warnf("Skipping map document %d for tenant %s: unable to extract name/streetName.", doc.DocumentId, doc.TenantId)
						continue
					}
					rows = append(rows, SearchIndexEntity{
						TenantId:   doc.TenantId,
						MapId:      doc.DocumentId,
						Name:       name,
						StreetName: street,
						UpdatedAt:  time.Now(),
					})
				}

				if len(rows) > 0 {
					txErr := db.WithContext(database.WithoutTenantFilter(ctx)).Transaction(func(tx *gorm.DB) error {
						return tx.Clauses(clause.OnConflict{
							Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "map_id"}},
							DoUpdates: clause.AssignmentColumns([]string{"name", "street_name", "updated_at"}),
						}).Create(&rows).Error
					})
					if txErr != nil {
						return res, txErr
					}
				}

				res.Processed += len(batch)
				if res.Processed%backfillProgressEvery < len(batch) {
					l.Infof("Backfill progress: processed %d map rows.", res.Processed)
				}

				if len(batch) < backfillPageSize {
					break
				}
				offset += backfillPageSize
			}

			// We count everything as upserted; distinguishing inserted vs updated
			// would require a per-row check which is not worth the cost.
			res.Inserted = res.Processed
			res.DurationMs = time.Since(start).Milliseconds()
			l.Infof("Backfill complete: processed %d rows in %dms.", res.Processed, res.DurationMs)
			return res, nil
		}
	}
}

func extractNameStreet(raw json.RawMessage) (string, string, error) {
	var rm RestModel
	if err := jsonapi.Unmarshal(raw, &rm); err != nil {
		return "", "", err
	}
	return rm.Name, rm.StreetName, nil
}
